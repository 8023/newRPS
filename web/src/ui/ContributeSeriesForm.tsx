import { useState } from "react";
import type { AppConfig } from "../shared/types";
import { StepEditor } from "./StepEditor";
import {
  buildSeriesContent,
  effectiveMaxSeriesSteps,
  effectiveMinSeriesSteps,
  emptySeriesStep,
  insertStepAfter,
  moveStep,
  removeStep,
  seriesCoverageGaps,
  stepHasFactionOverlap,
  toggleId,
  type StepDraft,
} from "./contributeSeries";

export function ContributeSeriesForm({
  config,
  name,
  anonymous,
  targetFactionIds,
  steps,
  busy,
  canSaveDraft,
  onName,
  onAnonymous,
  onTargets,
  onSteps,
  onSaveDraft,
  onSubmit,
  onError,
  showAnonymous = true,
  showSaveDraft = true,
  submitLabel = "提交审批",
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
  submitLabel?: string;
}) {
  const [attempted, setAttempted] = useState(false);
  const minSteps = effectiveMinSeriesSteps(config.punishmentRandomSettings?.minSeriesSteps);
  const maxSteps = effectiveMaxSeriesSteps(config.punishmentRandomSettings?.maxSeriesSteps);
  const gaps = seriesCoverageGaps(steps, targetFactionIds);
  const gapByIndex = new Map(gaps.map((item) => [item.index, item.missing]));
  const canSubmit = name.trim().length > 0
    && targetFactionIds.length > 0
    && steps.length >= minSteps
    && steps.length <= maxSteps
    && gaps.length === 0
    && steps.every((step) => step.variants.some((variant) => variant.text.trim()))
    && !steps.some(stepHasFactionOverlap);

  return (
    <form
      className="stack"
      noValidate
      onSubmit={(e) => {
        e.preventDefault();
        if (!canSubmit) { setAttempted(true); return; }
        onSubmit(buildSeriesContent(name, targetFactionIds, steps));
      }}
    >
      <label className="field-label">
        <span>系列名称</span>
        <input className={attempted && !name.trim() ? "field-invalid" : undefined} value={name} onChange={(e) => onName(e.target.value)} required />
      </label>
      <fieldset className={`checkbox-fieldset${attempted && targetFactionIds.length === 0 ? " field-invalid" : ""}`}>
        <legend>本系列面向哪些阵营</legend>
        <div className="checkbox-pill-row">
          {config.genderFactions.map((f) => (
            <button
              key={f.id}
              type="button"
              className={`checkbox-pill${targetFactionIds.includes(f.id) ? " active" : ""}`}
              onClick={() => onTargets(toggleId(targetFactionIds, f.id))}
            >
              {f.label}
            </button>
          ))}
        </div>
        {targetFactionIds.length === 0 ? <p className="notice">至少勾选一个阵营。</p> : null}
      </fieldset>
      {steps.map((step, i) => {
        const missing = gapByIndex.get(i) || [];
        return (
          <fieldset key={i} className={`contribute-step ${missing.length ? "step-gap" : ""}`}>
            <legend>第 {i + 1} 步</legend>
            {missing.length > 0 ? (
              <p className="notice">这一步还没覆盖：{missing.map((id) => config.genderFactions.find((f) => f.id === id)?.label || id).join("、")}</p>
            ) : null}
            <StepEditor
              config={config}
              value={step}
              onChange={(next) => onSteps((prevSteps) => prevSteps.map((item, idx) => (
                idx === i ? (typeof next === "function" ? next(item) : next) : item
              )))}
              onError={onError}
              showRandomPoolToggle
              showErrors={attempted}
              stepActions={(
                <>
                  <button type="button" disabled={i === 0} onClick={() => onSteps((prev) => moveStep(prev, i, -1))}>上移</button>
                  <button type="button" disabled={i === steps.length - 1} onClick={() => onSteps((prev) => moveStep(prev, i, 1))}>下移</button>
                  <button type="button" className="step-delete-button" disabled={steps.length <= 1} onClick={() => onSteps((prev) => removeStep(prev, i))}>删除</button>
                  <button type="button" disabled={steps.length >= maxSteps} onClick={() => onSteps((prev) => insertStepAfter(prev, i, emptySeriesStep(targetFactionIds), maxSteps))}>添加</button>
                </>
              )}
            />
          </fieldset>
        );
      })}
      {showAnonymous ? <label className="checkbox-inline"><input type="checkbox" checked={anonymous} onChange={(e) => onAnonymous(e.target.checked)} />匿名贡献</label> : null}
      <p className="hint">每步单独写文案；提交后整条系列一起审批。至少 {minSteps} 步，最多 {maxSteps} 步。当前 {steps.length} 步。</p>
      {steps.length < minSteps ? <p className="notice">系列任务至少需要 {minSteps} 步，还差 {minSteps - steps.length} 步。</p> : null}
      {steps.length > maxSteps ? <p className="notice">系列任务最多 {maxSteps} 步，超出 {steps.length - maxSteps} 步。</p> : null}
      <div className="contribute-submit-row">
        {showSaveDraft ? <button type="button" disabled={busy || !canSaveDraft} onClick={onSaveDraft}>保存草稿</button> : null}
        <button className="primary" disabled={busy} type="submit">{submitLabel}</button>
      </div>
    </form>
  );
}
