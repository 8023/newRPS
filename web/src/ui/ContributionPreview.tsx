/** 共建投稿内容的只读预览：后台「共建审核」与玩家端「参与共建」（待审批/已通过投稿，
    此时表单不可编辑，见 ContributeView.tsx 的 editableStatuses）共用同一套渲染，
    保证两边看到的样式完全一致（步骤框线宽度、"第 x 步" 位置等都由这一份组件决定，
    不再各自维护一份容易走样的复制版）。 */

export type VariantPreview = { text?: string; factionIds?: string[] };
export type StepPreview = {
  variants?: VariantPreview[];
  inRandomPool?: boolean;
  order?: number;
  tagIds?: string[];
  backgroundImage?: string;
  backgroundOpacity?: number;
};
export type DraftPreview = {
  name?: string;
  targetFactionIds?: string[];
  variants?: VariantPreview[];
  steps?: StepPreview[];
  tagIds?: string[];
  order?: number;
  inRandomPool?: boolean;
  backgroundImage?: string;
};

/** ContributionItem.content 经 RPC 回来时已经是解析好的对象（StepDraft 或 SeriesDraft
    形状），不再是需要 JSON.parse 的字符串——这里只做一层防御性的形状校验。 */
export function asContributionDraft(content: unknown): DraftPreview | null {
  return content && typeof content === "object" ? (content as DraftPreview) : null;
}

export function ContributionPreview({ draft, kind, factionLabel, tagLabel }: {
  draft: DraftPreview;
  kind: string;
  factionLabel: (id: string) => string;
  tagLabel: (id: string) => string;
}) {
  if (kind === "series" || draft.steps) {
    return (
      <div className="stack">
        {(draft.targetFactionIds || []).length > 0 ? (
          <p className="hint">面向阵营：{(draft.targetFactionIds || []).map(factionLabel).join("、")}</p>
        ) : null}
        {(draft.steps || []).map((step, i) => (
          <fieldset key={i} className="contribute-step">
            <legend>第 {i + 1} 步</legend>
            <VariantCards variants={step.variants} factionLabel={factionLabel} />
            <StepMeta step={step} tagLabel={tagLabel} />
          </fieldset>
        ))}
      </div>
    );
  }
  return (
    <div className="stack">
      <VariantCards variants={draft.variants} factionLabel={factionLabel} />
      <StepMeta step={draft} tagLabel={tagLabel} />
    </div>
  );
}

function VariantCards({ variants, factionLabel }: {
  variants?: VariantPreview[];
  factionLabel: (id: string) => string;
}) {
  const items = variants || [];
  if (items.length === 0) return <p className="hint">没有文案。</p>;
  return (
    <div className="stack">
      {items.map((item, i) => (
        <article key={i} className="step-variant">
          <p>{item.text}</p>
          <small className="hint">{(item.factionIds || []).length ? `适用：${(item.factionIds || []).map(factionLabel).join("、")}` : "未勾选阵营"}</small>
        </article>
      ))}
    </div>
  );
}

function StepMeta({ step, tagLabel }: { step: StepPreview; tagLabel: (id: string) => string }) {
  const tags = (step.tagIds || []).map(tagLabel).filter(Boolean);
  return (
    <>
      <small className="hint">
        {step.inRandomPool === false ? "不进入随机任务池" : `难度 ${step.order ?? "-"}`}
        {tags.length ? ` · 标签：${tags.join("、")}` : ""}
      </small>
      {step.backgroundImage ? <img src={step.backgroundImage} alt="" className="contribute-preview-cover" /> : null}
    </>
  );
}
