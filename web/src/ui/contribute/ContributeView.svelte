<script lang="ts">
  // 参与共建（玩家端投稿入口）。源：ui/ContributeView.tsx。
  // config/me 原为 props，现直接读 sessionStore；onBack/onError/ensureSession 全部
  // 替换成直接调用 routerStore/uiStore/sessionStore 方法。
  import { untrack } from "svelte";
  import type { ContributionItem, ContributionKind, ContributionStatus } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import { routerStore } from "../../lib/stores/routerStore.svelte";
  import ContributeSeriesForm from "./ContributeSeriesForm.svelte";
  import StepEditor from "./StepEditor.svelte";
  import ContributionStatusChip from "../shell/ContributionStatusChip.svelte";
  import ContributionPreview, { asContributionDraft, type StepPreview } from "./ContributionPreview.svelte";
  import ContributionListControls, {
    defaultContributionSortState, sortContributionItems, type ContributionSortState
  } from "./ContributionListControls.svelte";
  import {
    buildSeriesContent, emptySeriesStep, emptyStepDraft, contributionDraftTitle, formatContributionListMeta,
    isValidOrder, seriesDraftHasContent, stepHasContent, stepHasFactionOverlap, voteRatio, type StepDraft
  } from "../contributeSeries";

  const config = $derived(sessionStore.config!);
  const me = $derived(sessionStore.me!.player);

  function onError(message: string) {
    uiStore.notify(message);
  }
  function onBack() {
    requestLeave(() => { routerStore.keepContribute = false; routerStore.goto("lobby"); });
  }

  const tabs = [
    { id: "task", label: "随机任务", noun: "随机任务" },
    { id: "series", label: "系列任务", noun: "系列任务" }
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
        factionIds: Array.isArray(v?.factionIds) ? v.factionIds.filter((id) => factionIds.includes(id)) : []
      }))
      : base.variants;
    return {
      variants,
      inRandomPool: raw.inRandomPool !== false,
      order: typeof raw.order === "number" && raw.order > 0 ? raw.order : base.order,
      tagIds: Array.isArray(raw.tagIds) ? raw.tagIds.filter((id) => tagIds.includes(id)) : [],
      backgroundImage: typeof raw.backgroundImage === "string" ? raw.backgroundImage : "",
      backgroundOpacity: typeof raw.backgroundOpacity === "number" ? raw.backgroundOpacity : base.backgroundOpacity
    };
  }

  const allFactionIds = $derived(config.genderFactions.map((f) => f.id));
  const allTagIds = $derived((config.punishmentTags || []).map((t) => t.id));

  let tab = $state<(typeof tabs)[number]["id"]>("task");
  let items = $state<ContributionItem[]>([]);
  let selectedId = $state<string | null>(null);
  let detailsById = $state<Record<string, ContributionItem>>({});
  // 表单初始值只取一次（与原 React useState(() => ...) 惰性初始化同构）：allFactionIds
  // 随 config 变化不应清空用户正在编辑的表单，untrack 显式声明"仅初始化时读取"。
  let taskStep = $state<StepDraft>(untrack(() => emptyStepDraft(allFactionIds, true)));
  let taskAnon = $state(false);
  // 点击过一次「提交」但校验没过（难度为空/越界等）：把还没填好的输入框标红，与
  // ContributeSeriesForm 的 attempted 是同一套视觉语言。
  let taskAttempted = $state(false);
  let seriesName = $state("");
  let seriesAnon = $state(false);
  let seriesTargets = $state<string[]>(untrack(() => [...allFactionIds]));
  let seriesSteps = $state<StepDraft[]>(untrack(() => [emptySeriesStep(allFactionIds)]));
  let busy = $state(false);
  // 列表顶部的模糊查询 / 排序：投稿多起来之后靠滚动找一条不现实，与后台共建审核共用
  // 同一套控件（ContributionListControls）。切换板块时一并重置，避免"随机任务板块里
  // 按完成率排"这种在另一板块没有意义的残留状态。
  let search = $state("");
  let sort = $state<ContributionSortState>(defaultContributionSortState("task"));
  let dirty = $state(false);
  let leaveConfirm = $state<{ action: () => void } | null>(null);
  // selectItem 的详情拉取是异步的：如果用户在它返回前就切换了板块/选中了别的项（甚至清空了
  // 选中回到"新建"），此时才回填表单会用过期内容覆盖掉用户已经看到的新状态。用这个盒子记录
  // "最新一次点击选中的是哪一条"，await 后比对，不一致就只更新缓存、不再回填表单。
  let selectedIdRef = { current: null as string | null };

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
    leaveConfirm = { action };
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
    items = res.items || [];
  }

  $effect(() => {
    reload().catch((e: unknown) => onError(e instanceof Error ? e.message : "加载失败"));
  });

  function resetFormFor(kind: ContributionKind) {
    dirty = false;
    if (kind === "task") {
      taskStep = emptyStepDraft(allFactionIds, true);
      taskAnon = false;
      taskAttempted = false;
      return;
    }
    seriesName = "";
    seriesAnon = false;
    seriesTargets = [...allFactionIds];
    seriesSteps = [emptySeriesStep(allFactionIds)];
  }

  function populateFormFromDetail(d: ContributionItem) {
    const raw = asContributionDraft(d.content);
    dirty = false;
    if (!raw) return;
    if (d.kind === "task") {
      taskStep = toStepDraft(raw, allFactionIds, allTagIds);
      taskAnon = d.anonymous;
      taskAttempted = false;
      return;
    }
    seriesName = raw.name || "";
    seriesAnon = d.anonymous;
    const retainedTargets = (raw.targetFactionIds || []).filter((id) => allFactionIds.includes(id));
    seriesTargets = retainedTargets.length ? retainedTargets : [...allFactionIds];
    const steps = raw.steps || [];
    seriesSteps = steps.length ? steps.map((s) => toStepDraft(s, allFactionIds, allTagIds)) : [emptySeriesStep(allFactionIds)];
  }

  function setSelected(id: string | null) {
    selectedIdRef.current = id;
    selectedId = id;
  }

  function selectKind(id: (typeof tabs)[number]["id"]) {
    requestLeave(() => {
      tab = id;
      setSelected(null);
      search = "";
      sort = defaultContributionSortState(id);
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
    dirty = false;
    let d = detailsById[item.id];
    if (!d || d.updatedAt !== item.updatedAt) {
      try {
        d = await contributionAsk<ContributionItem>("contribution:get", { id: item.id, kind: item.kind });
        detailsById = { ...detailsById, [item.id]: d! };
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
      if (!isPersistentAuthError(e)) throw e;
      const ok = await sessionStore.restoreSession();
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
      anonymous: currentAnonymous()
    });
    setSelected(draft.id);
    dirty = false;
    detailsById = { ...detailsById, [draft.id]: { ...draft, content } };
    return draft;
  }

  async function saveDraftOnly() {
    if (!formHasContent()) return false;
    busy = true;
    try {
      await persistDraft();
      await reload();
      return true;
    } catch (e) {
      onError(e instanceof Error ? e.message : "保存失败");
      return false;
    } finally {
      busy = false;
    }
  }

  async function saveAndSubmit() {
    busy = true;
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
      busy = false;
    }
  }

  async function confirmSaveAndLeave() {
    if (!leaveConfirm) return;
    const action = leaveConfirm.action;
    if (formHasContent()) {
      const ok = await saveDraftOnly();
      if (!ok) return;
    } else {
      dirty = false;
    }
    leaveConfirm = null;
    action();
  }

  function confirmDiscardAndLeave() {
    if (!leaveConfirm) return;
    const action = leaveConfirm.action;
    dirty = false;
    leaveConfirm = null;
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

  const currentTab = $derived(tabs.find((t) => t.id === tab)!);
  const tabItems = $derived(items.filter((it) => it.kind === tab));
  // 查询只匹配列表标题（系列=系列名，随机任务=服务端截出的首份文案前 24 字），
  // 与后台共建审核的查询口径一致——列表里没有的内容不参与匹配，免得搜得到却看不出来。
  const itemsForTab = $derived.by(() => {
    const q = search.trim().toLowerCase();
    const matched = q ? tabItems.filter((it) => (it.title || "").toLowerCase().includes(q)) : tabItems;
    return sortContributionItems(matched, sort);
  });
  const selectedDetail = $derived(selectedId ? detailsById[selectedId] : null);
  const selectedStatus = $derived(selectedDetail?.status);
  const selectedEditable = $derived(selectedStatus ? editableStatuses.has(selectedStatus) : false);
  const selectedDraft = $derived(selectedDetail ? asContributionDraft(selectedDetail.content) : null);
  const canSaveDraft = $derived(!busy && formHasContent() && (!selectedId || selectedEditable));
  const taskHasOverlap = $derived(tab === "task" && stepHasFactionOverlap(taskStep));
  const taskOrderInvalid = $derived(tab === "task" && !isValidOrder(taskStep.order));
  const showWithdraw = $derived(Boolean(selectedDetail && !nonWithdrawableStatuses.has(selectedDetail.status)));
</script>

{#snippet itemMetaLine(item: ContributionItem, short: boolean)}
  <ContributionStatusChip status={item.status} />
  {#if item.status !== "approved"}
    {formatContributionListMeta({ kind: item.kind, status: item.status, updatedAt: item.updatedAt, short })}
  {:else}
    {formatContributionListMeta({
      kind: item.kind, status: item.status, updatedAt: item.updatedAt,
      likeRatio: voteRatio(item.likeCount, item.downCount),
      completionRate: item.completion?.rate ?? 0, short
    })}
  {/if}
{/snippet}

{#snippet withdrawButton()}
  {#if showWithdraw}
    <button type="button" onclick={() => void withdrawSelected()}>撤回</button>
  {/if}
{/snippet}

<section class="contribute-page">
  <div class="panel contribute-panel">
    <div class="panel-title lobby-title">
      <h2>参与共建</h2>
      <div class="lobby-title-actions">
        <div class="contribute-tabs" role="tablist">
          {#each tabs as item (item.id)}
            <button type="button" class={`small${tab === item.id ? " active" : ""}`} onclick={() => selectKind(item.id)}>{item.label}</button>
          {/each}
        </div>
        <button type="button" class="small" onclick={onBack}>返回大厅</button>
      </div>
    </div>
    <p class="hint">你好，{me.name}。投稿通过后才会进入正式游戏。待审内容不会被抽到。</p>
    <div class="contribute-layout">
      <div class="contribute-sidebar">
        <ContributionListControls
          kind={tab} {sort} onSort={(next) => (sort = next)} {search} onSearch={(v) => (search = v)}
          searchPlaceholder={tab === "series" ? "搜索系列任务标题" : "搜索随机任务文案"}
        >
          <!-- 「投稿新 xxx」入口固定在列表上方而不是列表末尾：投稿攒到几百条之后，
               跟在滚动区最后的按钮要一路滑到底才够得着。放在 ContributionListControls
               内部（而不是与它平级）是为了跟搜索框、排序按钮共用同一份滚动条槽预留，
               三者与下面的列表卡片右边缘才会对齐。 -->
          <button type="button" class="contribute-item contribute-item-new" onclick={startNew}>
            ＋ 投稿{currentTab.noun}
          </button>
        </ContributionListControls>
        <div class="contribute-list">
          {#each itemsForTab as item (item.id)}
            <button type="button" class={`contribute-item${selectedId === item.id ? " active" : ""}`} onclick={() => void selectItem(item)}>
              <strong>{itemSummary(item)}</strong>
              <small class="hint">{@render itemMetaLine(item, true)}</small>
            </button>
          {/each}
          {#if tabItems.length > 0 && itemsForTab.length === 0}<p class="hint">没有匹配的投稿</p>{/if}
        </div>
      </div>
      <div class="contribute-main">
        {#if selectedId && !selectedDetail}
          <p class="hint">加载中…</p>
        {:else}
          <div class="stack">
            {#if selectedDetail}
              <div class="contribute-detail-head">
                <strong>{contributionDraftTitle(selectedDraft)}</strong>
                <small class="hint">{@render itemMetaLine(selectedDetail, false)}</small>
              </div>
              {#if selectedDetail.reviewComment}<p class="notice">批注：{selectedDetail.reviewComment}</p>{/if}
            {/if}
            {#if selectedDetail && !selectedEditable}
              {#if selectedDraft}
                <ContributionPreview draft={selectedDraft} kind={selectedDetail.kind} {factionLabel} {tagLabel} />
              {:else}
                <p class="notice">内容解析失败。</p>
              {/if}
              {#if selectedDetail.status === "pending"}
                <div class="row">
                  <button type="button" onclick={() => void withdrawSelected()}>撤回</button>
                </div>
              {/if}
            {:else if tab === "task"}
              <form class="stack" novalidate onsubmit={(e) => {
                e.preventDefault();
                if (taskHasOverlap || taskOrderInvalid) { taskAttempted = true; return; }
                void saveAndSubmit();
              }}>
                <StepEditor
                  {config}
                  value={taskStep}
                  onChange={(next) => { taskStep = typeof next === "function" ? next(taskStep) : next; dirty = true; }}
                  {onError}
                  showRandomPoolToggle={false}
                  showErrors={taskAttempted}
                >
                  {#snippet rightOfPreview()}
                    <label class="checkbox-inline"><input type="checkbox" checked={taskAnon} onchange={(e) => { taskAnon = e.currentTarget.checked; dirty = true; }} />匿名贡献</label>
                  {/snippet}
                </StepEditor>
                <div class="contribute-submit-row">
                  {@render withdrawButton()}
                  <button type="button" disabled={!canSaveDraft} onclick={() => { void saveDraftOnly(); }}>保存</button>
                  <button class="primary" disabled={busy || taskHasOverlap || taskOrderInvalid} type="submit">提交</button>
                </div>
              </form>
            {:else if tab === "series"}
              <ContributeSeriesForm
                {config}
                name={seriesName}
                anonymous={seriesAnon}
                targetFactionIds={seriesTargets}
                steps={seriesSteps}
                {busy}
                {canSaveDraft}
                onName={(value) => { seriesName = value; dirty = true; }}
                onAnonymous={(value) => { seriesAnon = value; dirty = true; }}
                onTargets={(value) => { seriesTargets = value; dirty = true; }}
                onSteps={(next) => { seriesSteps = typeof next === "function" ? next(seriesSteps) : next; dirty = true; }}
                onSaveDraft={() => { void saveDraftOnly(); }}
                onSubmit={() => { void saveAndSubmit(); }}
                {onError}
                saveDraftLabel="保存"
                submitLabel="提交"
                extraSubmitActions={withdrawButton}
              />
            {/if}
          </div>
        {/if}
      </div>
    </div>
  </div>
  {#if leaveConfirm}
    <div class="modal-backdrop" onclick={() => (leaveConfirm = null)}>
      <section class="panel contribute-unsaved-card" onclick={(event) => event.stopPropagation()} role="dialog" aria-modal="true" aria-labelledby="contribute-unsaved-title">
        <h2 id="contribute-unsaved-title">内容尚未保存</h2>
        <p class="hint">当前填写还没存成草稿。保存后再离开，或放弃这次写入。</p>
        <div class="contribute-unsaved-actions">
          <button type="button" disabled={busy} onclick={confirmDiscardAndLeave}>放弃</button>
          <button type="button" class="primary" disabled={busy} onclick={() => { void confirmSaveAndLeave(); }}>保存</button>
        </div>
      </section>
    </div>
  {/if}
</section>
