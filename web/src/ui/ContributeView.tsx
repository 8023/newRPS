import { useEffect, useMemo, useRef, useState } from "react";
import type { AppConfig, ContributionItem, ContributionKind, ContributionStatus, PublicPlayer } from "../shared/types";
import { ask } from "../lib/rpc";
import { ContributeSeriesForm } from "./ContributeSeriesForm";
import { StepEditor } from "./StepEditor";
import { ContributionStatusChip } from "./AppViews";
import { ContributionPreview, asContributionDraft, type StepPreview } from "./ContributionPreview";
import { ContributionListControls, defaultContributionSortState, sortContributionItems, type ContributionSortState } from "./ContributionListControls";
import {
  buildSeriesContent,
  emptySeriesStep,
  emptyStepDraft,
  contributionDraftTitle,
  formatContributionListMeta,
  isValidOrder,
  seriesDraftHasContent,
  stepHasContent,
  stepHasFactionOverlap,
  toggleId,
  voteRatio,
  type StepDraft,
} from "./contributeSeries";

const tabs = [
  { id: "task", label: "随机任务", noun: "随机任务" },
  { id: "series", label: "系列任务", noun: "系列任务" },
] as const;

// 与后端 saveDraft/withdraw 的可编辑/可撤回状态判定保持一致（contribution_service.go）：
// draft/rejected/withdrawn/approved 都能继续编辑（编辑已通过投稿会另起一个 draft 版本，
// 并立即让旧正式内容退出任务池，待新版本重新审批通过后恢复）；pending 是唯一"正在审批中、不能改内容"的状态。
const editableStatuses = new Set<ContributionStatus>(["draft", "rejected", "withdrawn", "approved"]);
// 不能撤回的状态：draft（还没提交过，没什么可撤）、withdrawn（已经是这个状态了）。
const nonWithdrawableStatuses = new Set<ContributionStatus>(["draft", "withdrawn"]);

/** 把投稿内容还原成 StepEditor 需要的 StepDraft 形状，供"选中已有投稿→回填编辑器"使用；
    字段全部做防御性校验，早期投稿/手改内容缺字段时退回 emptyStepDraft 的默认值。 */
function toStepDraft(raw: StepPreview | null | undefined, factionIds: string[], tagIds: string[]): StepDraft {
  const base = emptyStepDraft(factionIds, true);
  if (!raw || typeof raw !== "object") return base;
  const variants = Array.isArray(raw.variants) && raw.variants.length
    ? raw.variants.map((v) => ({
      text: typeof v?.text === "string" ? v.text : "",
      factionIds: Array.isArray(v?.factionIds) ? v.factionIds.filter((id) => factionIds.includes(id)) : [],
    }))
    : base.variants;
  return {
    variants,
    inRandomPool: raw.inRandomPool !== false,
    order: typeof raw.order === "number" && raw.order > 0 ? raw.order : base.order,
    tagIds: Array.isArray(raw.tagIds) ? raw.tagIds.filter((id) => tagIds.includes(id)) : [],
    backgroundImage: typeof raw.backgroundImage === "string" ? raw.backgroundImage : "",
    backgroundOpacity: typeof raw.backgroundOpacity === "number" ? raw.backgroundOpacity : base.backgroundOpacity,
  };
}

export function ContributeView({ config, me, onBack, onError, ensureSession }: {
  config: AppConfig;
  me: PublicPlayer;
  onBack: () => void;
  onError: (message: string) => void;
  ensureSession?: () => Promise<boolean>;
}) {
  const allFactionIds = config.genderFactions.map((f) => f.id);
  const allTagIds = (config.punishmentTags || []).map((t) => t.id);
  const [tab, setTab] = useState<(typeof tabs)[number]["id"]>("task");
  const [items, setItems] = useState<ContributionItem[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detailsById, setDetailsById] = useState<Record<string, ContributionItem>>({});
  const [taskStep, setTaskStep] = useState<StepDraft>(() => emptyStepDraft(allFactionIds, true));
  const [taskAnon, setTaskAnon] = useState(false);
  // 点击过一次「提交」但校验没过（难度为空/越界等）：把还没填好的输入框标红，与
  // ContributeSeriesForm 的 attempted 是同一套视觉语言。
  const [taskAttempted, setTaskAttempted] = useState(false);
  const [seriesName, setSeriesName] = useState("");
  const [seriesAnon, setSeriesAnon] = useState(false);
  const [seriesTargets, setSeriesTargets] = useState<string[]>([...allFactionIds]);
  const [seriesSteps, setSeriesSteps] = useState<StepDraft[]>([emptySeriesStep(allFactionIds)]);
  const [busy, setBusy] = useState(false);
  // 列表顶部的模糊查询 / 排序：投稿多起来之后靠滚动找一条不现实，与后台共建审核共用
  // 同一套控件（ContributionListControls）。切换板块时一并重置，避免"随机任务板块里
  // 按完成率排"这种在另一板块没有意义的残留状态。
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState<ContributionSortState>(() => defaultContributionSortState(tab));
  const [dirty, setDirty] = useState(false);
  const [leaveConfirm, setLeaveConfirm] = useState<{ action: () => void } | null>(null);
  // selectItem 的详情拉取是异步的：如果用户在它返回前就切换了板块/选中了别的项（甚至清空了
  // 选中回到"新建"），此时才回填表单会用过期内容覆盖掉用户已经看到的新状态。用 ref 记录
  // "最新一次点击选中的是哪一条"，await 后比对，不一致就只更新缓存、不再回填表单。
  const selectedIdRef = useRef<string | null>(null);

  function factionLabel(id: string) {
    return config.genderFactions.find((f) => f.id === id)?.label || id;
  }
  function tagLabel(id: string) {
    return (config.punishmentTags || []).find((t) => t.id === id)?.name || id;
  }
  function formHasContent() {
    if (tab === "task") return stepHasContent(taskStep);
    return seriesDraftHasContent(seriesName, seriesSteps);
  }
  function requestLeave(action: () => void) {
    if (!dirty) {
      action();
      return;
    }
    setLeaveConfirm({ action });
  }
  function currentContent(): unknown {
    if (tab === "task") {
      // 兜底转数字，理由同 contributeSeries.ts 的 buildSeriesContent：真正拦截空值/
      // 越界的是提交前的 isValidOrder 校验，这里只是不让 "" 被序列化成字符串。
      return { ...taskStep, order: typeof taskStep.order === "number" ? taskStep.order : 0, inRandomPool: true };
    }
    return buildSeriesContent(seriesName, seriesTargets, seriesSteps);
  }
  function currentAnonymous() {
    return tab === "task" ? taskAnon : seriesAnon;
  }

  async function reload() {
    const res = await contributionAsk<{ items: ContributionItem[] }>("contribution:list", {});
    const list = res.items || [];
    setItems(list);
  }

  useEffect(() => {
    reload().catch((e: unknown) => onError(e instanceof Error ? e.message : "加载失败"));
  }, []);

  function resetFormFor(kind: ContributionKind) {
    setDirty(false);
    if (kind === "task") {
      setTaskStep(emptyStepDraft(allFactionIds, true));
      setTaskAnon(false);
      setTaskAttempted(false);
      return;
    }
    setSeriesName("");
    setSeriesAnon(false);
    setSeriesTargets([...allFactionIds]);
    setSeriesSteps([emptySeriesStep(allFactionIds)]);
  }

  function populateFormFromDetail(d: ContributionItem) {
    const raw = asContributionDraft(d.content);
    setDirty(false);
    if (!raw) return;
    if (d.kind === "task") {
      setTaskStep(toStepDraft(raw, allFactionIds, allTagIds));
      setTaskAnon(d.anonymous);
      setTaskAttempted(false);
      return;
    }
    setSeriesName(raw.name || "");
    setSeriesAnon(d.anonymous);
    const retainedTargets = (raw.targetFactionIds || []).filter((id) => allFactionIds.includes(id));
    setSeriesTargets(retainedTargets.length ? retainedTargets : [...allFactionIds]);
    const steps = raw.steps || [];
    setSeriesSteps(steps.length ? steps.map((s) => toStepDraft(s, allFactionIds, allTagIds)) : [emptySeriesStep(allFactionIds)]);
  }

  function setSelected(id: string | null) {
    selectedIdRef.current = id;
    setSelectedId(id);
  }

  function selectKind(id: (typeof tabs)[number]["id"]) {
    requestLeave(() => {
      setTab(id);
      setSelected(null);
      setSearch("");
      setSort(defaultContributionSortState(id));
      resetFormFor(id);
    });
  }

  function startNew() {
    requestLeave(() => {
      setSelected(null);
      resetFormFor(tab);
    });
  }

  async function selectItem(item: ContributionItem) {
    requestLeave(() => { void loadItem(item); });
  }

  async function loadItem(item: ContributionItem) {
    setSelected(item.id);
    setDirty(false);
    let d = detailsById[item.id];
    if (!d || d.updatedAt !== item.updatedAt) {
      try {
        d = await contributionAsk<ContributionItem>("contribution:get", { id: item.id, kind: item.kind });
        setDetailsById((prev) => ({ ...prev, [item.id]: d! }));
      } catch (e) {
        onError(e instanceof Error ? e.message : "加载失败");
        return;
      }
    }
    // 拉取期间用户可能已经切走了选中项，此时只更新缓存，不能再回填表单（见 selectedIdRef 注释）。
    if (selectedIdRef.current === item.id && editableStatuses.has(d.status)) {
      populateFormFromDetail(d);
    }
  }

  function isPersistentAuthError(e: unknown): boolean {
    const message = e instanceof Error ? e.message : String(e);
    return /持久身份/.test(message);
  }

  async function contributionAsk<T>(event: string, payload: unknown): Promise<T> {
    try {
      return await ask<T>(event, payload);
    } catch (e) {
      if (!ensureSession || !isPersistentAuthError(e)) throw e;
      const ok = await ensureSession();
      if (!ok) throw e;
      return await ask<T>(event, payload);
    }
  }

  async function persistDraft(): Promise<ContributionItem | null> {
    const content = currentContent();
    const draft = await contributionAsk<ContributionItem>("contribution:saveDraft", {
      id: selectedId || "",
      kind: tab,
      content,
      anonymous: currentAnonymous(),
    });
    setSelected(draft.id);
    setDirty(false);
    setDetailsById((prev) => ({ ...prev, [draft.id]: { ...draft, content } }));
    return draft;
  }

  async function saveDraftOnly() {
    if (!formHasContent()) return false;
    setBusy(true);
    try {
      await persistDraft();
      await reload();
      return true;
    } catch (e) {
      onError(e instanceof Error ? e.message : "保存失败");
      return false;
    } finally {
      setBusy(false);
    }
  }

  async function saveAndSubmit() {
    setBusy(true);
    try {
      const draft = await persistDraft();
      if (!draft) return;
      await contributionAsk("contribution:submit", { id: draft.id, kind: draft.kind });
      // 提交成功后清空表单，避免下一次新建时误复用上一条投稿的文案和封面。
      resetFormFor(tab);
      setSelected(null);
      await reload();
    } catch (e) {
      onError(e instanceof Error ? e.message : "提交失败");
      await reload();
    } finally {
      setBusy(false);
    }
  }

  async function confirmSaveAndLeave() {
    if (!leaveConfirm) return;
    const action = leaveConfirm.action;
    if (formHasContent()) {
      const ok = await saveDraftOnly();
      if (!ok) return;
    } else {
      setDirty(false);
    }
    setLeaveConfirm(null);
    action();
  }

  function confirmDiscardAndLeave() {
    if (!leaveConfirm) return;
    const action = leaveConfirm.action;
    setDirty(false);
    setLeaveConfirm(null);
    action();
  }

  async function withdrawSelected() {
    if (!selectedId) return;
    const kind = detailsById[selectedId]?.kind || items.find((it) => it.id === selectedId)?.kind || tab;
    try {
      await contributionAsk("contribution:withdraw", { id: selectedId, kind });
      await reload();
      setSelected(null);
    } catch (e) {
      onError(e instanceof Error ? e.message : "撤回失败");
    }
  }

  function itemSummary(item: ContributionItem): string {
    if (item.title) return item.title;
    const d = detailsById[item.id];
    if (!d) return "…";
    const raw = asContributionDraft(d.content);
    if (!raw) return "(内容解析失败)";
    return contributionDraftTitle(raw);
  }

  // 状态药丸徽标 + 日期（+ 已通过投稿的点赞率/完成率），与后台共建审核统一同一套样式
  // （见 ContributionStatusChip / formatContributionListMeta 的注释）。列表里日期不带
  // 年份（short），详情头里保留 YY/MM/DD——同一份数据、两处显示要求不同，靠 short 区分。
  function itemMetaLine(item: ContributionItem, short: boolean) {
    const chip = <ContributionStatusChip status={item.status} />;
    if (item.status !== "approved") {
      return <>{chip} {formatContributionListMeta({ kind: item.kind, status: item.status, updatedAt: item.updatedAt, short })}</>;
    }
    return (
      <>
        {chip}{" "}
        {formatContributionListMeta({
          kind: item.kind,
          status: item.status,
          updatedAt: item.updatedAt,
          likeRatio: voteRatio(item.likeCount, item.downCount),
          completionRate: item.completion?.rate ?? 0,
          short,
        })}
      </>
    );
  }

  const currentTab = tabs.find((t) => t.id === tab)!;
  const tabItems = items.filter((it) => it.kind === tab);
  // 查询只匹配列表标题（系列=系列名，随机任务=服务端截出的首份文案前 24 字），
  // 与后台共建审核的查询口径一致——列表里没有的内容不参与匹配，免得搜得到却看不出来。
  const itemsForTab = useMemo(() => {
    const q = search.trim().toLowerCase();
    const matched = q ? tabItems.filter((it) => (it.title || "").toLowerCase().includes(q)) : tabItems;
    return sortContributionItems(matched, sort);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [items, tab, search, sort]);
  const selectedDetail = selectedId ? detailsById[selectedId] : null;
  const selectedStatus = selectedDetail?.status;
  const selectedEditable = selectedStatus ? editableStatuses.has(selectedStatus) : false;
  const selectedDraft = selectedDetail ? asContributionDraft(selectedDetail.content) : null;
  const canSaveDraft = !busy && formHasContent() && (!selectedId || selectedEditable);
  const taskHasOverlap = tab === "task" && stepHasFactionOverlap(taskStep);
  const taskOrderInvalid = tab === "task" && !isValidOrder(taskStep.order);
  // 撤回按钮放在编辑表单最下面、保存/提交的左边（与它们同一行），而不是单独占一整行浮在
  // 表单上方——避免玩家一进编辑态就先看到一个孤立的危险操作。
  const withdrawButton = selectedDetail && !nonWithdrawableStatuses.has(selectedDetail.status)
    ? <button type="button" onClick={() => void withdrawSelected()}>撤回</button>
    : null;

  return (
    <section className="contribute-page">
      <div className="panel contribute-panel">
        <div className="panel-title lobby-title">
          <h2>参与共建</h2>
          <div className="lobby-title-actions">
            <div className="contribute-tabs" role="tablist">
              {tabs.map((item) => (
                <button key={item.id} type="button" className={`small${tab === item.id ? " active" : ""}`} onClick={() => selectKind(item.id)}>{item.label}</button>
              ))}
            </div>
            <button type="button" className="small" onClick={() => requestLeave(onBack)}>返回大厅</button>
          </div>
        </div>
        <p className="hint">你好，{me.name}。投稿通过后才会进入正式游戏。待审内容不会被抽到。</p>
        <div className="contribute-layout">
          <div className="contribute-sidebar">
            <ContributionListControls
              kind={tab}
              sort={sort}
              onSort={setSort}
              search={search}
              onSearch={setSearch}
              searchPlaceholder={tab === "series" ? "搜索系列任务标题" : "搜索随机任务文案"}
            >
              {/* 「投稿新 xxx」入口固定在列表上方而不是列表末尾：投稿攒到几百条之后，
                  跟在滚动区最后的按钮要一路滑到底才够得着。放在 ContributionListControls
                  内部（而不是与它平级）是为了跟搜索框、排序按钮共用同一份滚动条槽预留，
                  三者与下面的列表卡片右边缘才会对齐。 */}
              <button type="button" className="contribute-item contribute-item-new" onClick={startNew}>
                ＋ 投稿{currentTab.noun}
              </button>
            </ContributionListControls>
            <div className="contribute-list">
              {itemsForTab.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  className={`contribute-item${selectedId === item.id ? " active" : ""}`}
                  onClick={() => void selectItem(item)}
                >
                  <strong>{itemSummary(item)}</strong>
                  <small className="hint">{itemMetaLine(item, true)}</small>
                </button>
              ))}
              {tabItems.length > 0 && itemsForTab.length === 0 ? <p className="hint">没有匹配的投稿</p> : null}
            </div>
          </div>
          <div className="contribute-main">
            {selectedId && !selectedDetail ? <p className="hint">加载中…</p> : (
              <div className="stack">
                {selectedDetail ? (
                  <>
                    <div className="contribute-detail-head">
                      <strong>{contributionDraftTitle(selectedDraft)}</strong>
                      <small className="hint">{itemMetaLine(selectedDetail, false)}</small>
                    </div>
                    {selectedDetail.reviewComment ? <p className="notice">批注：{selectedDetail.reviewComment}</p> : null}
                  </>
                ) : null}
                {selectedDetail && !selectedEditable ? (
                  <>
                    {selectedDraft ? (
                      <ContributionPreview draft={selectedDraft} kind={selectedDetail.kind} factionLabel={factionLabel} tagLabel={tagLabel} />
                    ) : <p className="notice">内容解析失败。</p>}
                    {selectedDetail.status === "pending" ? (
                      <div className="row">
                        <button type="button" onClick={() => void withdrawSelected()}>撤回</button>
                      </div>
                    ) : null}
                  </>
                ) : (
                  <>
                    {tab === "task" && (
                      <form className="stack" noValidate onSubmit={(e) => {
                        e.preventDefault();
                        if (taskHasOverlap || taskOrderInvalid) { setTaskAttempted(true); return; }
                        void saveAndSubmit();
                      }}>
                        <StepEditor
                          config={config}
                          value={taskStep}
                          onChange={(next) => { setTaskStep(next); setDirty(true); }}
                          onError={onError}
                          showRandomPoolToggle={false}
                          showErrors={taskAttempted}
                          rightOfPreview={<label className="checkbox-inline"><input type="checkbox" checked={taskAnon} onChange={(e) => { setTaskAnon(e.target.checked); setDirty(true); }} />匿名贡献</label>}
                        />
                        <div className="contribute-submit-row">
                          {withdrawButton}
                          <button type="button" disabled={!canSaveDraft} onClick={() => { void saveDraftOnly(); }}>保存</button>
                          <button className="primary" disabled={busy || taskHasOverlap || taskOrderInvalid} type="submit">提交</button>
                        </div>
                      </form>
                    )}
                    {tab === "series" && (
                      <ContributeSeriesForm
                        config={config}
                        name={seriesName}
                        anonymous={seriesAnon}
                        targetFactionIds={seriesTargets}
                        steps={seriesSteps}
                        busy={busy}
                        canSaveDraft={canSaveDraft}
                        onName={(value) => { setSeriesName(value); setDirty(true); }}
                        onAnonymous={(value) => { setSeriesAnon(value); setDirty(true); }}
                        onTargets={(value) => { setSeriesTargets(value); setDirty(true); }}
                        onSteps={(next) => { setSeriesSteps(next); setDirty(true); }}
                        onSaveDraft={() => { void saveDraftOnly(); }}
                        onSubmit={() => { void saveAndSubmit(); }}
                        onError={onError}
                        saveDraftLabel="保存"
                        submitLabel="提交"
                        extraSubmitActions={withdrawButton}
                      />
                    )}
                  </>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
      {leaveConfirm ? (
        <div className="modal-backdrop" onClick={() => setLeaveConfirm(null)}>
          <section className="panel contribute-unsaved-card" onClick={(event) => event.stopPropagation()} role="dialog" aria-modal="true" aria-labelledby="contribute-unsaved-title">
            <h2 id="contribute-unsaved-title">内容尚未保存</h2>
            <p className="hint">当前填写还没存成草稿。保存后再离开，或放弃这次写入。</p>
            <div className="contribute-unsaved-actions">
              <button type="button" disabled={busy} onClick={confirmDiscardAndLeave}>放弃</button>
              <button type="button" className="primary" disabled={busy} onClick={() => { void confirmSaveAndLeave(); }}>保存</button>
            </div>
          </section>
        </div>
      ) : null}
    </section>
  );
}
