<script lang="ts">
  /** 列表 + 详情的共建审核核心组件：随机任务/系列任务 tab 都复用它，与玩家端「参与共建」
      （ContributeView.svelte）同一套 .contribute-layout 骨架，只是右侧换成预览+审核操作而非
      编辑表单。源：ui/AdminContributionReview.tsx:255-534。 */
  import { untrack } from "svelte";
  import type { AppConfig, ContributionItem } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import { kindLabel, toAdminStepDraft, type JumpTarget } from "../../lib/contributionAdmin";
  import {
    contributionDraftTitle, contributionStatusLabel, contributionVoteTail, emptySeriesStep, emptyStepDraft,
    formatContributionDay, formatContributionListMeta, isValidOrder, stepHasFactionOverlap, voteRatio, type StepDraft
  } from "../contributeSeries";
  import ContributionListControls, { defaultContributionSortState, sortContributionItems, type ContributionSortState } from "../contribute/ContributionListControls.svelte";
  import ContributionPreview, { asContributionDraft } from "../contribute/ContributionPreview.svelte";
  import ContributionStatusChip from "../shell/ContributionStatusChip.svelte";
  import ContributeSeriesForm from "../contribute/ContributeSeriesForm.svelte";
  import StepEditor from "../contribute/StepEditor.svelte";
  import SubmitterHoverName from "./SubmitterHoverName.svelte";

  let { items, kind, config, password, onError, onChanged, jumpTarget, emptyText }: {
    items: ContributionItem[];
    kind: "task" | "series";
    config: AppConfig;
    password: string;
    onError: (message: string) => void;
    onChanged: () => void;
    jumpTarget?: JumpTarget | null;
    emptyText: string;
  } = $props();

  let currentId = $state<string | null>(null);
  let current = $state<ContributionItem | null>(null);
  let comment = $state("");
  let busy = $state(false);
  let search = $state("");
  // primary 为 null = 保持 sortReviewQueue 的待审优先队列顺序；管理员点过排序按钮之后才整体重排。
  // kind/config 只在组件挂载时取一次即可——父级 {#key section} 会在 kind 切换时整体重建本组件。
  let sort = $state<ContributionSortState>(untrack(() => defaultContributionSortState(kind)));
  let editing = $state(false);
  // 点击过一次「保存并立即发布/批准」但校验没过（难度为空/越界等）：标红对应输入框，
  // 与 ContributeSeriesForm 的 attempted 是同一套视觉语言。
  let editAttempted = $state(false);
  let editTask = $state<StepDraft>(untrack(() => emptyStepDraft(config.genderFactions.map((f) => f.id), true)));
  let editSeriesName = $state("");
  let editSeriesTargets = $state<string[]>([]);
  let editSeriesSteps = $state<StepDraft[]>([emptySeriesStep()]);
  const itemRefs: Record<string, HTMLButtonElement | null> = {};
  let consumedJumpNonce: number | null = null;

  const draft = $derived(asContributionDraft(current?.content));

  const filteredItems = $derived.by(() => {
    const q = search.trim().toLowerCase();
    const matched = q
      ? items.filter((it) => {
        const t = it.title || kindLabel[it.kind] || it.kind;
        return t.toLowerCase().includes(q) || (it.submitterName || "").toLowerCase().includes(q);
      })
      : items;
    return sortContributionItems(matched, sort);
  });

  function factionLabel(id: string) {
    return config.genderFactions.find((f) => f.id === id)?.label || id;
  }
  function tagLabel(id: string) {
    return (config.punishmentTags || []).find((t) => t.id === id)?.name || id;
  }

  async function openItem(id: string) {
    try {
      const detail = await ask<ContributionItem>("admin:action", { action: "contributionGet", id, kind, password });
      currentId = id;
      current = detail;
      comment = detail.reviewComment || "";
      editing = false;
      editAttempted = false;
    } catch (e) {
      onError(e instanceof Error ? e.message : "加载失败");
    }
  }

  $effect(() => {
    if (currentId && !items.some((it) => it.id === currentId)) {
      currentId = null;
      current = null;
    }
  });

  // 从「待处理·总览」跳转过来：items 加载完成前 jumpTarget 会先到，等 items 真正
  // 有内容后再定位对应条目，避免在列表还是空的那一瞬间误报"没有这个状态"。
  $effect(() => {
    if (!jumpTarget || consumedJumpNonce === jumpTarget.nonce || items.length === 0) return;
    consumedJumpNonce = jumpTarget.nonce;
    const match = items.find((it) => it.status === jumpTarget!.status);
    if (!match) {
      onError(`没有「${contributionStatusLabel[jumpTarget.status] || jumpTarget.status}」状态的${kindLabel[kind]}投稿`);
      return;
    }
    search = "";
    void openItem(match.id);
    requestAnimationFrame(() => itemRefs[match.id]?.scrollIntoView({ block: "center", behavior: "smooth" }));
  });

  async function act(action: string, reviewedContent?: string) {
    if (!currentId) return;
    busy = true;
    try {
      await ask("admin:action", { action, id: currentId, kind, comment, password, ...(reviewedContent == null ? {} : { reviewedContent }) });
      currentId = null;
      current = null;
      comment = "";
      editing = false;
      onChanged();
    } catch (e) {
      onError(e instanceof Error ? e.message : "操作失败");
    } finally {
      busy = false;
    }
  }

  function beginEditing() {
    if (!current || !draft) return;
    if (current.kind === "task") {
      editTask = toAdminStepDraft(draft, config);
    } else {
      const allFactionIds = config.genderFactions.map((f) => f.id);
      editSeriesName = draft.name || "";
      const targets = (draft.targetFactionIds || []).filter((id) => allFactionIds.includes(id));
      editSeriesTargets = targets.length ? targets : allFactionIds;
      editSeriesSteps = (draft.steps || []).length ? (draft.steps || []).map((step) => toAdminStepDraft(step, config)) : [emptySeriesStep(allFactionIds)];
    }
    editAttempted = false;
    editing = true;
  }

  function publishEdited(next: unknown) {
    void act("contributionPublish", JSON.stringify(next));
  }

  async function saveComment() {
    if (!currentId) return;
    busy = true;
    try {
      await ask("admin:action", { action: "contributionUpdateComment", id: currentId, kind, comment, password });
      onError("批注已保存");
      onChanged();
    } catch (e) {
      onError(e instanceof Error ? e.message : "保存失败");
    } finally {
      busy = false;
    }
  }

  const title = $derived(current ? (current.title || contributionDraftTitle(draft) || kindLabel[current.kind]) : "");
  const voteTail = $derived(current
    ? contributionVoteTail({ kind: current.kind, status: current.status, likeRatio: voteRatio(current.likeCount, current.downCount), completionRate: current.completion?.rate })
    : "");
</script>

<div class="contribute-layout">
  <div class="contribute-sidebar">
    <ContributionListControls
      {kind} {sort} onSort={(next) => (sort = next)}
      {search} onSearch={(value) => (search = value)}
      searchPlaceholder={kind === "series" ? "搜索系列任务标题 / 投稿者" : "搜索随机任务文案 / 投稿者"}
    />
    <div class="contribute-list">
      {#each filteredItems as item (item.id)}
        <button
          bind:this={itemRefs[item.id]}
          type="button"
          class={`contribute-item${currentId === item.id ? " active" : ""}`}
          onclick={() => void openItem(item.id)}
        >
          <strong>{item.title || kindLabel[item.kind] || item.kind}</strong>
          <small class="hint">
            <ContributionStatusChip status={item.status} />
            {formatContributionListMeta({ kind: item.kind, status: item.status, updatedAt: item.updatedAt, likeRatio: voteRatio(item.likeCount, item.downCount), completionRate: item.completion?.rate, short: true })}
          </small>
        </button>
      {/each}
      {#if items.length === 0}<p class="hint">{emptyText}</p>{/if}
      {#if items.length > 0 && filteredItems.length === 0}<p class="hint">没有匹配的投稿</p>{/if}
    </div>
  </div>
  <div class="contribute-main">
    {#if current}
      <div class="stack">
        <div class="contribute-detail-head">
          <strong>{title}</strong>
          <small class="hint">
            <ContributionStatusChip status={current.status} />
            {formatContributionDay(current.updatedAt)} ·
            <SubmitterHoverName playerId={current.submitterId} label={current.anonymous ? `${current.submitterName}（匿名）` : current.submitterName} />
            {voteTail ? ` · ${voteTail}` : ""}
          </small>
        </div>
        {#if editing && current.kind === "task"}
          <form class="stack" novalidate onsubmit={(event) => {
            event.preventDefault();
            if (stepHasFactionOverlap(editTask) || !isValidOrder(editTask.order)) { editAttempted = true; return; }
            publishEdited({ ...editTask, inRandomPool: true });
          }}>
            <StepEditor {config} value={editTask} onChange={(next) => (editTask = typeof next === "function" ? next(editTask) : next)} {onError} showRandomPoolToggle={false} showErrors={editAttempted} />
            <div class="row">
              <button type="button" disabled={busy} onclick={() => (editing = false)}>取消编辑</button>
              <button type="submit" class="primary" disabled={busy || stepHasFactionOverlap(editTask) || !isValidOrder(editTask.order)}>保存并立即发布</button>
            </div>
          </form>
        {:else if editing && current.kind === "series"}
          <div class="stack">
            <ContributeSeriesForm
              {config}
              name={editSeriesName}
              anonymous={current.anonymous}
              targetFactionIds={editSeriesTargets}
              steps={editSeriesSteps}
              {busy}
              canSaveDraft={false}
              onName={(value) => (editSeriesName = value)}
              onAnonymous={() => undefined}
              onTargets={(value) => (editSeriesTargets = value)}
              onSteps={(next) => (editSeriesSteps = typeof next === "function" ? next(editSeriesSteps) : next)}
              onSaveDraft={() => undefined}
              onSubmit={(next) => publishEdited(next)}
              {onError}
              showAnonymous={false}
              showSaveDraft={false}
              submitLabel="保存并批准"
            />
            <div class="row">
              <button type="button" disabled={busy} onclick={() => (editing = false)}>取消编辑</button>
            </div>
          </div>
        {:else if draft}
          <ContributionPreview {draft} kind={current.kind} {factionLabel} {tagLabel} />
        {:else}
          <p class="notice">内容解析失败。</p>
        {/if}
        {#if current.likeCount || current.downCount}
          <p class="hint">获赞 {current.likeCount ?? 0} · 点踩 {current.downCount ?? 0} · 点赞率 {voteRatio(current.likeCount, current.downCount)}%</p>
        {/if}

        {#if !editing && current.status === "pending"}
          <label class="field-label"><span>批注</span><textarea value={comment} oninput={(e) => (comment = e.currentTarget.value)} placeholder="驳回时给投稿者看的说明（选填）"></textarea></label>
          <div class="row">
            <button type="button" disabled={busy} onclick={beginEditing}>编辑</button>
            <button type="button" class="primary" disabled={busy} onclick={() => void act("contributionPublish")}>批准</button>
            <button type="button" class="danger-button" disabled={busy} onclick={() => void act("contributionReject")}>驳回</button>
          </div>
        {/if}

        {#if !editing && current.status === "approved"}
          <div class="row">
            <button type="button" disabled={busy} onclick={beginEditing}>编辑</button>
            <button type="button" class="danger-button" disabled={busy} onclick={() => void act("contributionUnpublish")}>下架</button>
          </div>
        {/if}

        {#if !editing && current.status === "rejected"}
          <label class="field-label"><span>批注</span><textarea value={comment} oninput={(e) => (comment = e.currentTarget.value)} placeholder="驳回理由"></textarea></label>
          <div class="row">
            <button type="button" disabled={busy} onclick={() => void saveComment()}>保存批注</button>
            <button type="button" class="primary" disabled={busy} onclick={() => void act("contributionRevertReject")}>撤销驳回</button>
          </div>
        {/if}

        {#if !editing && current.status === "withdrawn"}
          <label class="field-label"><span>批注</span><textarea value={comment} oninput={(e) => (comment = e.currentTarget.value)} placeholder="取消撤回时可以给投稿者留言（选填）"></textarea></label>
          <div class="row">
            <button type="button" class="primary" disabled={busy} onclick={() => void act("contributionRevertWithdraw")}>取消撤回</button>
          </div>
        {/if}
      </div>
    {:else}
      <p class="hint">选择一条投稿</p>
    {/if}
  </div>
</div>
