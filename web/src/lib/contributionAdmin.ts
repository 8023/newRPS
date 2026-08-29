// 后台「共建审核」的纯逻辑辅助：待审队列排序、总览计数聚合、投稿内容 → StepDraft 转换。
// 源：ui/AdminContributionReview.tsx 散落各处。
import type { AppConfig, ContributionItem, ContributionStatus } from "../shared/types";
import type { StepPreview } from "../ui/contribute/ContributionPreview.svelte";
import { emptyStepDraft, type StepDraft } from "../ui/contributeSeries";

export const kindLabel: Record<string, string> = {
  task: "随机任务",
  series: "系列任务"
};

// 需要管理员做决定的状态：排在任意列表的最前面。其余（已通过/已驳回/已撤回）按更新时间倒序跟在后面。
const pendingLikeStatuses = new Set<ContributionStatus>(["pending"]);

export function sortReviewQueue(items: ContributionItem[]): ContributionItem[] {
  const pendingGroup = items.filter((it) => pendingLikeStatuses.has(it.status)).sort((a, b) => a.updatedAt - b.updatedAt);
  const restGroup = items.filter((it) => !pendingLikeStatuses.has(it.status)).sort((a, b) => b.updatedAt - a.updatedAt);
  return [...pendingGroup, ...restGroup];
}

export function toAdminStepDraft(raw: StepPreview | null | undefined, config: AppConfig): StepDraft {
  const factionIds = config.genderFactions.map((f) => f.id);
  const tagIds = (config.punishmentTags || []).map((t) => t.id);
  const base = emptyStepDraft(factionIds, true);
  if (!raw || typeof raw !== "object") return base;
  return {
    variants: Array.isArray(raw.variants) && raw.variants.length
      ? raw.variants.map((v) => ({
        text: typeof v?.text === "string" ? v.text : "",
        factionIds: Array.isArray(v?.factionIds) ? v.factionIds.filter((id) => factionIds.includes(id)) : []
      }))
      : base.variants,
    inRandomPool: raw.inRandomPool !== false,
    order: typeof raw.order === "number" && raw.order > 0 ? raw.order : base.order,
    tagIds: Array.isArray(raw.tagIds) ? raw.tagIds.filter((id) => tagIds.includes(id)) : [],
    backgroundImage: typeof raw.backgroundImage === "string" ? raw.backgroundImage : "",
    backgroundOpacity: typeof raw.backgroundOpacity === "number" ? raw.backgroundOpacity : base.backgroundOpacity
  };
}

export type ContributionReviewCounts = { pending: number };
export type OverviewCounts = Record<string, number>;
export type PoolStats = { randomTasks?: number; series?: number };
export type PendingOverviewResponse = { counts: { task?: OverviewCounts; series?: OverviewCounts }; poolStats?: PoolStats };
export type JumpTarget = { status: ContributionStatus; nonce: number };

/** 随机任务 + 系列任务加总的待审条数，供顶部「待处理」选项卡摘要与侧边栏审核徽标共用同一份口径。 */
export function reviewQueueCounts(res: PendingOverviewResponse): ContributionReviewCounts {
  const t = res.counts?.task || {};
  const s = res.counts?.series || {};
  return { pending: (t.pending || 0) + (s.pending || 0) };
}
