import { useEffect, useRef, useState } from "react";
import type { AppConfig, ContributionKind, ContributionStatus, ContributionSubmission, PublicPlayer } from "../shared/types";
import { ask } from "../lib/rpc";
import { ContributeSeriesForm } from "./ContributeSeriesForm";
import { StepEditor } from "./StepEditor";
import { ContributionStatusChip } from "./AppViews";
import { ContributionPreview, effectiveContributionContent, parseContributionDraft, type StepPreview } from "./ContributionPreview";
import {
  buildSeriesContent,
  contributionHasPublishedVersion,
  emptySeriesStep,
  emptyStepDraft,
  contributionDraftTitle,
  formatContributionListMeta,
  seriesDraftHasContent,
  stepHasContent,
  stepHasFactionOverlap,
  type StepDraft,
} from "./contributeSeries";

const tabs = [
  { id: "gender", label: "性别共建", noun: "性别类型" },
  { id: "task", label: "随机任务", noun: "随机任务" },
  { id: "series", label: "系列任务", noun: "系列任务" },
] as const;

// 与后端 contributionStore.saveDraft 的可编辑状态判定保持一致（contribution_save.go）：
// 待审批/复审中/下架申请中都是"已经在流转中"，不允许直接改内容，只能撤回/等审核。
const editableStatuses = new Set<ContributionStatus>(["draft", "rejected", "withdrawn", "approved", "revision_draft", "revision_rejected"]);

/** 把投稿内容 JSON 还原成 StepEditor 需要的 StepDraft 形状，供"选中已有投稿→回填编辑器"使用；
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
    backgroundOpacity: 0.22,
  };
}

type ContributionDetail = {
  submission: ContributionSubmission;
  version: { content: string; reviewedContent?: string };
  likeCount?: number;
  downCount?: number;
  voteCount?: number;
  realRatio?: number | null;
  completion?: { participants: number; rate: number | null };
};

export function ContributeView({ config, me, onBack, onError, ensureSession }: {
  config: AppConfig;
  me: PublicPlayer;
  onBack: () => void;
  onError: (message: string) => void;
  ensureSession?: () => Promise<boolean>;
}) {
  const allFactionIds = config.genderFactions.map((f) => f.id);
  const allTagIds = (config.punishmentTags || []).map((t) => t.id);
  const [tab, setTab] = useState<(typeof tabs)[number]["id"]>("gender");
  const [items, setItems] = useState<ContributionSubmission[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detailsById, setDetailsById] = useState<Record<string, ContributionDetail>>({});
  const [genderLabel, setGenderLabel] = useState("");
  const [factionId, setFactionId] = useState(config.genderFactions[0]?.id || "");
  const [taskStep, setTaskStep] = useState<StepDraft>(() => emptyStepDraft(allFactionIds, true));
  const [taskAnon, setTaskAnon] = useState(false);
  const [seriesName, setSeriesName] = useState("");
  const [seriesAnon, setSeriesAnon] = useState(false);
  const [seriesTargets, setSeriesTargets] = useState<string[]>([...allFactionIds]);
  const [seriesSteps, setSeriesSteps] = useState<StepDraft[]>([emptySeriesStep(allFactionIds)]);
  const [busy, setBusy] = useState(false);
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
    if (tab === "gender") return genderLabel.trim().length > 0;
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
    if (tab === "gender") return { label: genderLabel, factionId };
    if (tab === "task") return { ...taskStep, inRandomPool: true };
    return buildSeriesContent(seriesName, seriesTargets, seriesSteps);
  }
  function currentAnonymous() {
    if (tab === "gender") return false;
    if (tab === "task") return taskAnon;
    return seriesAnon;
  }

  // 详情缓存只用于「左栏摘要/点赞率展示」，绝不反向驱动表单回填——回填只在用户点击某一条
  // 投稿的那一刻（selectItem）触发一次；如果改成对 detailsById 的 useEffect 联动，别的投稿
  // 后台预取完成时会让 detailsById 引用变化，从而把用户正在编辑的表单内容用旧内容覆盖掉。
  async function prefetchDetails(list: ContributionSubmission[]) {
    const stale = list.filter((it) => {
      const cached = detailsById[it.id];
      return !cached || cached.submission.updatedAt !== it.updatedAt;
    });
    if (stale.length === 0) return;
    const results = await Promise.all(stale.map(async (it) => {
      try {
        return [it.id, await contributionAsk<ContributionDetail>("contribution:get", { id: it.id })] as const;
      } catch {
        return null;
      }
    }));
    setDetailsById((prev) => {
      const next = { ...prev };
      for (const r of results) if (r) next[r[0]] = r[1];
      return next;
    });
  }

  async function reload() {
    const res = await contributionAsk<{ items: ContributionSubmission[] }>("contribution:list", {});
    const list = res.items || [];
    setItems(list);
    void prefetchDetails(list);
  }

  useEffect(() => {
    reload().catch((e: unknown) => onError(e instanceof Error ? e.message : "加载失败"));
  }, []);

  function resetFormFor(kind: ContributionKind) {
    setDirty(false);
    if (kind === "gender") {
      setGenderLabel("");
      setFactionId(config.genderFactions[0]?.id || "");
      return;
    }
    if (kind === "task") {
      setTaskStep(emptyStepDraft(allFactionIds, true));
      setTaskAnon(false);
      return;
    }
    setSeriesName("");
    setSeriesAnon(false);
    setSeriesTargets([...allFactionIds]);
    setSeriesSteps([emptySeriesStep(allFactionIds)]);
  }

  function populateFormFromDetail(d: ContributionDetail) {
    const raw = parseContributionDraft(effectiveContributionContent(d.version));
    setDirty(false);
    if (!raw) return;
    if (d.submission.kind === "gender") {
      setGenderLabel(raw.label || "");
      setFactionId(raw.factionId && allFactionIds.includes(raw.factionId) ? raw.factionId : (config.genderFactions[0]?.id || ""));
      return;
    }
    if (d.submission.kind === "task") {
      setTaskStep(toStepDraft(raw, allFactionIds, allTagIds));
      setTaskAnon(d.submission.anonymous);
      return;
    }
    setSeriesName(raw.name || "");
    setSeriesAnon(d.submission.anonymous);
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
      resetFormFor(id);
    });
  }

  function startNew() {
    requestLeave(() => {
      setSelected(null);
      resetFormFor(tab);
    });
  }

  async function selectItem(item: ContributionSubmission) {
    requestLeave(() => { void loadItem(item); });
  }

  async function loadItem(item: ContributionSubmission) {
    setSelected(item.id);
    setDirty(false);
    let d = detailsById[item.id];
    if (!d || d.submission.updatedAt !== item.updatedAt) {
      try {
        d = await contributionAsk<ContributionDetail>("contribution:get", { id: item.id });
        setDetailsById((prev) => ({ ...prev, [item.id]: d! }));
      } catch (e) {
        onError(e instanceof Error ? e.message : "加载失败");
        return;
      }
    }
    // 拉取期间用户可能已经切走了选中项，此时只更新缓存，不能再回填表单（见 selectedIdRef 注释）。
    if (selectedIdRef.current === item.id && editableStatuses.has(d.submission.status)) {
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

  async function persistDraft(): Promise<ContributionSubmission | null> {
    const content = currentContent();
    const draft = await contributionAsk<ContributionSubmission>("contribution:saveDraft", {
      id: selectedId || "",
      kind: tab,
      content,
      anonymous: currentAnonymous(),
    });
    setSelected(draft.id);
    setDirty(false);
    setDetailsById((prev) => ({
      ...prev,
      [draft.id]: {
        ...prev[draft.id],
        submission: draft,
        version: { content: JSON.stringify(content) },
      },
    }));
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
      await contributionAsk("contribution:submit", { id: draft.id });
      // 提交成功后清空表单：上一次上传的封面图已绑定到这条投稿，若不清空，
      // 不刷新页面直接改文字连续提交下一条会因为图片归属校验失败而报错。
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
    try {
      await ask("contribution:withdraw", { id: selectedId });
      await reload();
      setSelected(null);
    } catch (e) {
      onError(e instanceof Error ? e.message : "撤回失败");
    }
  }

  async function requestUnpublishSelected() {
    if (!selectedId) return;
    try {
      await ask("contribution:requestUnpublish", { id: selectedId });
      await reload();
      setSelected(null);
    } catch (e) {
      onError(e instanceof Error ? e.message : "申请失败");
    }
  }

  function itemSummary(item: ContributionSubmission): string {
    if (item.title) return item.title;
    const d = detailsById[item.id];
    if (!d) return "…";
    const raw = parseContributionDraft(effectiveContributionContent(d.version));
    if (!raw) return "(内容解析失败)";
    return contributionDraftTitle(raw);
  }

  // 状态药丸徽标 + 日期（+ 已通过投稿的点赞率/完成率），与后台共建审核统一同一套样式
  // （见 ContributionStatusChip / formatContributionListMeta 的注释）。
  function itemMetaLine(item: ContributionSubmission) {
    const chip = <ContributionStatusChip status={item.status} />;
    if (item.publishedVersion <= 0) {
      return <>{chip} {formatContributionListMeta({ kind: item.kind, status: item.status, updatedAt: item.updatedAt })}</>;
    }
    const d = detailsById[item.id];
    if (!d) return <>{chip} …</>;
    return (
      <>
        {chip}{" "}
        {formatContributionListMeta({
          kind: item.kind,
          status: item.status,
          updatedAt: item.updatedAt,
          likeRatio: d.realRatio ?? 0,
          completionRate: d.completion?.rate ?? 0,
        })}
      </>
    );
  }

  const currentTab = tabs.find((t) => t.id === tab)!;
  const itemsForTab = items.filter((it) => it.kind === tab);
  const selectedDetail = selectedId ? detailsById[selectedId] : null;
  const selectedStatus = selectedDetail?.submission.status;
  const selectedEditable = selectedStatus ? editableStatuses.has(selectedStatus) : false;
  const selectedDraft = selectedDetail ? parseContributionDraft(effectiveContributionContent(selectedDetail.version)) : null;
  const canSaveDraft = !busy && formHasContent() && (!selectedId || selectedEditable);
  const taskHasOverlap = tab === "task" && stepHasFactionOverlap(taskStep);

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
            <div className="contribute-list">
              {itemsForTab.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  className={`contribute-item${selectedId === item.id ? " active" : ""}`}
                  onClick={() => void selectItem(item)}
                >
                  <strong>{itemSummary(item)}</strong>
                  <small className="hint">{itemMetaLine(item)}</small>
                </button>
              ))}
              <button type="button" className="contribute-item contribute-item-new" onClick={startNew}>
                ＋ 投稿{currentTab.noun}
              </button>
            </div>
          </div>
          <div className="contribute-main">
            {selectedId && !selectedDetail ? <p className="hint">加载中…</p> : (
              <div className="stack">
                {selectedDetail ? (
                  <>
                    <div className="contribute-detail-head">
                      <strong>{contributionDraftTitle(selectedDraft)}</strong>
                      <small className="hint">{itemMetaLine(selectedDetail.submission)}</small>
                    </div>
                    {selectedDetail.submission.reviewComment ? <p className="notice">批注：{selectedDetail.submission.reviewComment}</p> : null}
                  </>
                ) : null}
                {selectedDetail && !selectedEditable ? (
                  <>
                    {selectedDraft ? (
                      <ContributionPreview draft={selectedDraft} kind={selectedDetail.submission.kind} factionLabel={factionLabel} tagLabel={tagLabel} />
                    ) : <p className="notice">内容解析失败。</p>}
                    {(selectedDetail.submission.status === "pending" || selectedDetail.submission.status === "revision_pending") ? (
                      <div className="row">
                        <button type="button" onClick={() => void withdrawSelected()}>撤回</button>
                      </div>
                    ) : null}
                  </>
                ) : (
                  <>
                    {selectedDetail && contributionHasPublishedVersion(selectedDetail.submission.status)
                      && selectedDetail.submission.status !== "revision_pending"
                      && selectedDetail.submission.status !== "unpublish_pending" ? (
                      <div className="row">
                        <button type="button" onClick={() => void requestUnpublishSelected()}>申请下架</button>
                      </div>
                    ) : null}
                    {tab === "gender" && (
                      <form className="stack" onSubmit={(e) => { e.preventDefault(); void saveAndSubmit(); }}>
                        <div className="contribute-split-row">
                          <label className="field-label"><span>性别名称</span><input value={genderLabel} onChange={(e) => { setGenderLabel(e.target.value); setDirty(true); }} required /></label>
                          <label className="field-label">
                            <span>归属阵营</span>
                            <select value={factionId} onChange={(e) => { setFactionId(e.target.value); setDirty(true); }}>
                              {config.genderFactions.map((f) => <option key={f.id} value={f.id}>{f.label}</option>)}
                            </select>
                          </label>
                        </div>
                        <div className="contribute-submit-row">
                          <button type="button" disabled={!canSaveDraft} onClick={() => { void saveDraftOnly(); }}>保存草稿</button>
                          <button className="primary" disabled={busy} type="submit">提交审批</button>
                        </div>
                      </form>
                    )}
                    {tab === "task" && (
                      <form className="stack" onSubmit={(e) => { e.preventDefault(); void saveAndSubmit(); }}>
                        <StepEditor
                          config={config}
                          value={taskStep}
                          onChange={(next) => { setTaskStep(next); setDirty(true); }}
                          onError={onError}
                          showRandomPoolToggle={false}
                          rightOfPreview={<label className="checkbox-inline"><input type="checkbox" checked={taskAnon} onChange={(e) => { setTaskAnon(e.target.checked); setDirty(true); }} />匿名贡献</label>}
                        />
                        <div className="contribute-submit-row">
                          <button type="button" disabled={!canSaveDraft} onClick={() => { void saveDraftOnly(); }}>保存草稿</button>
                          <button className="primary" disabled={busy || taskHasOverlap} type="submit">提交审批</button>
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
