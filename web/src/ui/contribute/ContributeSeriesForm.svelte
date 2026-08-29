<script lang="ts">
  // 源：ui/ContributeSeriesForm.tsx
  import type { Snippet } from "svelte";
  import type { AppConfig } from "../../shared/types";
  import StepEditor from "./StepEditor.svelte";
  import {
    buildSeriesContent, effectiveMaxSeriesSteps, effectiveMinSeriesSteps, emptySeriesStep, insertStepAfter,
    isValidOrder, moveStep, removeStep, seriesCoverageGaps, stepHasFactionOverlap, toggleId, type StepDraft
  } from "../contributeSeries";

  let {
    config, name, anonymous, targetFactionIds, steps, busy, canSaveDraft,
    onName, onAnonymous, onTargets, onSteps, onSaveDraft, onSubmit, onError,
    showAnonymous = true, showSaveDraft = true, saveDraftLabel = "保存草稿", submitLabel = "提交审批",
    extraSubmitActions
  }: {
    config: AppConfig;
    name: string;
    anonymous: boolean;
    targetFactionIds: string[];
    steps: StepDraft[];
    busy: boolean;
    canSaveDraft: boolean;
    onName: (value: string) => void;
    onAnonymous: (value: boolean) => void;
    onTargets: (value: string[]) => void;
    onSteps: (next: StepDraft[] | ((prev: StepDraft[]) => StepDraft[])) => void;
    onSaveDraft: () => void;
    onSubmit: (content: ReturnType<typeof buildSeriesContent>) => void;
    onError: (message: string) => void;
    showAnonymous?: boolean;
    showSaveDraft?: boolean;
    saveDraftLabel?: string;
    submitLabel?: string;
    /** 渲染在提交行最左侧（保存/提交按钮之前），供调用方插入"撤回"这类附加操作。 */
    extraSubmitActions?: Snippet;
  } = $props();

  let attempted = $state(false);
  const minSteps = $derived(effectiveMinSeriesSteps(config.punishmentRandomSettings?.minSeriesSteps));
  const maxSteps = $derived(effectiveMaxSeriesSteps(config.punishmentRandomSettings?.maxSeriesSteps));
  const gaps = $derived(seriesCoverageGaps(steps, targetFactionIds));
  const gapByIndex = $derived(new Map(gaps.map((item) => [item.index, item.missing])));
  const canSubmit = $derived(
    name.trim().length > 0
    && targetFactionIds.length > 0
    && steps.length >= minSteps
    && steps.length <= maxSteps
    && gaps.length === 0
    && steps.every((step) => step.variants.some((variant) => variant.text.trim()))
    && steps.every((step) => !step.inRandomPool || isValidOrder(step.order))
    && !steps.some(stepHasFactionOverlap)
  );

  function submitForm(e: SubmitEvent) {
    e.preventDefault();
    if (!canSubmit) { attempted = true; return; }
    onSubmit(buildSeriesContent(name, targetFactionIds, steps));
  }
</script>

<form class="stack" novalidate onsubmit={submitForm}>
  <label class="field-label">
    <span>系列名称</span>
    <input class={attempted && !name.trim() ? "field-invalid" : undefined} value={name} oninput={(e) => onName(e.currentTarget.value)} required />
  </label>
  <fieldset class={`checkbox-fieldset${attempted && targetFactionIds.length === 0 ? " field-invalid" : ""}`}>
    <legend>本系列面向哪些阵营</legend>
    <div class="checkbox-pill-row">
      {#each config.genderFactions as f (f.id)}
        <button type="button" class={`checkbox-pill${targetFactionIds.includes(f.id) ? " active" : ""}`} onclick={() => onTargets(toggleId(targetFactionIds, f.id))}>
          {f.label}
        </button>
      {/each}
    </div>
    {#if targetFactionIds.length === 0}<p class="notice">至少勾选一个阵营。</p>{/if}
  </fieldset>
  {#each steps as step, i (i)}
    {@const missing = gapByIndex.get(i) || []}
    <fieldset class={`contribute-step ${missing.length ? "step-gap" : ""}`}>
      <legend>第 {i + 1} 步</legend>
      {#if missing.length > 0}
        <p class="notice">这一步还没覆盖：{missing.map((id) => config.genderFactions.find((f) => f.id === id)?.label || id).join("、")}</p>
      {/if}
      <StepEditor
        {config}
        value={step}
        onChange={(next) => onSteps((prevSteps) => prevSteps.map((item, idx) => (
          idx === i ? (typeof next === "function" ? next(item) : next) : item
        )))}
        {onError}
        showRandomPoolToggle
        showErrors={attempted}
      >
        {#snippet stepActions()}
          <button type="button" disabled={i === 0} onclick={() => onSteps((prev) => moveStep(prev, i, -1))}>上移</button>
          <button type="button" disabled={i === steps.length - 1} onclick={() => onSteps((prev) => moveStep(prev, i, 1))}>下移</button>
          <button type="button" class="step-delete-button" disabled={steps.length <= 1} onclick={() => onSteps((prev) => removeStep(prev, i))}>删除</button>
          <button type="button" disabled={steps.length >= maxSteps} onclick={() => onSteps((prev) => insertStepAfter(prev, i, emptySeriesStep(targetFactionIds), maxSteps))}>添加</button>
        {/snippet}
      </StepEditor>
    </fieldset>
  {/each}
  {#if showAnonymous}
    <label class="checkbox-inline"><input type="checkbox" checked={anonymous} onchange={(e) => onAnonymous(e.currentTarget.checked)} />匿名贡献</label>
  {/if}
  <p class="hint">每步单独写文案；提交后整条系列一起审批。至少 {minSteps} 步，最多 {maxSteps} 步。当前 {steps.length} 步。</p>
  {#if steps.length < minSteps}<p class="notice">系列任务至少需要 {minSteps} 步，还差 {minSteps - steps.length} 步。</p>{/if}
  {#if steps.length > maxSteps}<p class="notice">系列任务最多 {maxSteps} 步，超出 {steps.length - maxSteps} 步。</p>{/if}
  <div class="contribute-submit-row">
    {#if extraSubmitActions}{@render extraSubmitActions()}{/if}
    {#if showSaveDraft}<button type="button" disabled={busy || !canSaveDraft} onclick={onSaveDraft}>{saveDraftLabel}</button>{/if}
    <button class="primary" disabled={busy} type="submit">{submitLabel}</button>
  </div>
</form>
