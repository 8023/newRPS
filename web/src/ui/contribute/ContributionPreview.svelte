<script module lang="ts">
  /** 共建投稿内容的只读预览：后台「共建审核」与玩家端「参与共建」（待审批/已通过投稿，
      此时表单不可编辑，见 ContributeView 的 editableStatuses）共用同一套渲染，
      保证两边看到的样式完全一致（步骤框线宽度、"第 x 步" 位置等都由这一份组件决定，
      不再各自维护一份容易走样的复制版）。源：ui/ContributionPreview.tsx。 */

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
</script>

<script lang="ts">
  let { draft, kind, factionLabel, tagLabel }: {
    draft: DraftPreview;
    kind: string;
    factionLabel: (id: string) => string;
    tagLabel: (id: string) => string;
  } = $props();
</script>

{#snippet variantCards(variants: VariantPreview[] | undefined)}
  {@const items = variants || []}
  {#if items.length === 0}
    <p class="hint">没有文案。</p>
  {:else}
    <div class="stack">
      {#each items as item, i (i)}
        <article class="step-variant">
          <p>{item.text}</p>
          <small class="hint">{(item.factionIds || []).length ? `适用：${(item.factionIds || []).map(factionLabel).join("、")}` : "未勾选阵营"}</small>
        </article>
      {/each}
    </div>
  {/if}
{/snippet}

{#snippet stepMeta(step: StepPreview)}
  {@const tags = (step.tagIds || []).map(tagLabel).filter(Boolean)}
  <small class="hint">
    {step.inRandomPool === false ? "不进入随机任务池" : `难度 ${step.order ?? "-"}`}
    {tags.length ? ` · 标签：${tags.join("、")}` : ""}
  </small>
  {#if step.backgroundImage}<img src={step.backgroundImage} alt="" class="contribute-preview-cover" />{/if}
{/snippet}

{#if kind === "series" || draft.steps}
  <div class="stack">
    {#if (draft.targetFactionIds || []).length > 0}
      <p class="hint">面向阵营：{(draft.targetFactionIds || []).map(factionLabel).join("、")}</p>
    {/if}
    {#each draft.steps || [] as step, i (i)}
      <fieldset class="contribute-step">
        <legend>第 {i + 1} 步</legend>
        {@render variantCards(step.variants)}
        {@render stepMeta(step)}
      </fieldset>
    {/each}
  </div>
{:else}
  <div class="stack">
    {@render variantCards(draft.variants)}
    {@render stepMeta(draft)}
  </div>
{/if}
