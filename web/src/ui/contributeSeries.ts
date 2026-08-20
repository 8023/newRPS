/** 系列步数纯防御性的技术硬顶：与后端 contributionstore.go 的 maxSeriesSteps 常量保持一致，
    只用来挡住畸形/超大 payload，不是玩法层面的实际约束——后台「最高步数」配置在这个范围内
    填多大都真实生效，不会被静默收紧。 */
export const MAX_SERIES_STEPS = 1000;
/** 后台「最高步数」留空/填非正数时的默认值，与上面的技术硬顶是两回事，见
    effectiveMaxSeriesSteps 的说明。 */
export const DEFAULT_MAX_SERIES_STEPS = 20;
export const DEFAULT_MIN_SERIES_STEPS = 10;
/** 每一步最多 8 份阵营文案，与后端 maxStepCandidates 一致。 */
export const MAX_STEP_VARIANTS = 8;

/** 多选 checkbox 通用的"选中态字符串数组"切换：已选则移除，未选则追加。 */
export function toggleId(list: readonly string[], id: string): string[] {
  return list.includes(id) ? list.filter((x) => x !== id) : [...list, id];
}

export type TaskVariantDraft = {
  text: string;
  factionIds: string[];
};

export type StepDraft = {
  variants: TaskVariantDraft[];
  inRandomPool: boolean;
  order: number;
  tagIds: string[];
  backgroundImage: string;
  backgroundOpacity: number;
};

export function emptyStepDraft(factionIds: string[], inRandomPool = true): StepDraft {
  return {
    variants: [{ text: "", factionIds: [...factionIds] }],
    inRandomPool,
    order: 50,
    tagIds: [],
    backgroundImage: "",
    backgroundOpacity: 0.22,
  };
}

export function emptySeriesStep(factionIds: string[] = []): StepDraft {
  return emptyStepDraft(factionIds, true);
}

export function effectiveMinSeriesSteps(value?: number): number {
  const n = !value || value <= 0 ? DEFAULT_MIN_SERIES_STEPS : value;
  return n > MAX_SERIES_STEPS ? MAX_SERIES_STEPS : n;
}

/** 后台「最高步数」配置的生效值：不填/非正数按默认值 20 兜底；配置了就在纯防御性技术硬顶
    （MAX_SERIES_STEPS）以内完全生效——与后端 effectiveMaxSeriesSteps（contribution_codec.go）
    保持一致的语义。 */
export function effectiveMaxSeriesSteps(value?: number): number {
  const n = !value || value <= 0 ? DEFAULT_MAX_SERIES_STEPS : value;
  return n > MAX_SERIES_STEPS ? MAX_SERIES_STEPS : n;
}

export function stepFactionUnion(step: StepDraft): string[] {
  const seen = new Set<string>();
  for (const variant of step.variants) {
    for (const id of variant.factionIds) {
      if (id) seen.add(id);
    }
  }
  return [...seen];
}

export function missingTargetFactions(step: StepDraft, targets: readonly string[]): string[] {
  const have = new Set(stepFactionUnion(step));
  return targets.filter((id) => !have.has(id));
}

export function seriesCoverageGaps(steps: readonly StepDraft[], targets: readonly string[]): Array<{ index: number; missing: string[] }> {
  return steps
    .map((step, index) => ({ index, missing: missingTargetFactions(step, targets) }))
    .filter((item) => item.missing.length > 0);
}

/** 被两份及以上文案同时勾选的阵营。单份文案时返回空集合。 */
export function overlappingFactionIds(variants: readonly TaskVariantDraft[]): string[] {
  const counts = new Map<string, number>();
  for (const variant of variants) {
    for (const id of variant.factionIds) {
      if (!id) continue;
      counts.set(id, (counts.get(id) || 0) + 1);
    }
  }
  return [...counts.entries()].filter(([, n]) => n > 1).map(([id]) => id);
}

export function stepHasFactionOverlap(step: StepDraft): boolean {
  return overlappingFactionIds(step.variants).length > 0;
}

export function buildSeriesContent(name: string, targetFactionIds: readonly string[], steps: readonly StepDraft[]) {
  return {
    name,
    targetFactionIds: [...targetFactionIds],
    steps: steps.map((step) => ({
      variants: step.variants.map((variant) => ({
        text: variant.text,
        factionIds: [...variant.factionIds],
      })),
      inRandomPool: step.inRandomPool,
      order: step.inRandomPool ? step.order : -1,
      tagIds: [...step.tagIds],
      backgroundImage: step.backgroundImage,
      backgroundOpacity: step.backgroundOpacity || 0.22,
    })),
  };
}

export function stepHasContent(step: StepDraft): boolean {
  if (step.backgroundImage.trim() || step.tagIds.length > 0) return true;
  return step.variants.some((variant) => variant.text.trim().length > 0);
}

export function seriesDraftHasContent(name: string, steps: readonly StepDraft[]): boolean {
  return name.trim().length > 0 || steps.some(stepHasContent);
}

export function moveStep<T>(steps: readonly T[], index: number, dir: -1 | 1): T[] {
  const nextIndex = index + dir;
  if (index < 0 || nextIndex < 0 || nextIndex >= steps.length) return [...steps];
  const next = [...steps];
  const current = next[index];
  const swap = next[nextIndex];
  if (current === undefined || swap === undefined) return next;
  next[index] = swap;
  next[nextIndex] = current;
  return next;
}

export function insertStepAfter<T>(steps: readonly T[], index: number, empty: T, max: number): T[] {
  if (steps.length >= max || index < 0 || index >= steps.length) return [...steps];
  const next = [...steps];
  next.splice(index + 1, 0, empty);
  return next;
}

export function removeStep<T>(steps: readonly T[], index: number): T[] {
  if (steps.length <= 1) return [...steps];
  return steps.filter((_, i) => i !== index);
}

/** 投稿列表未批准时的日期：精确到天，YY/MM/DD。 */
export function formatContributionDay(ms: number): string {
  const d = new Date(ms);
  if (!Number.isFinite(d.getTime())) return "";
  const yy = String(d.getFullYear()).slice(-2);
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${yy}/${mm}/${dd}`;
}

export const contributionStatusLabel: Record<string, string> = {
  draft: "草稿",
  pending: "待审批",
  approved: "已通过",
  rejected: "已驳回",
  withdrawn: "已撤回",
  revision_draft: "修订草稿",
  revision_pending: "修订审核中",
  revision_rejected: "修订被驳回",
  unpublish_pending: "下架申请",
};

export function contributionHasPublishedVersion(status: string): boolean {
  return status === "approved"
    || status === "revision_draft"
    || status === "revision_pending"
    || status === "revision_rejected"
    || status === "unpublish_pending";
}

/** 投稿标题：性别用 label，系列用 name（玩家/管理员都要靠它认出是哪个系列，会展示给
    玩家）；单个随机任务不设可读标题（玩家/管理员都不需要看到），退到第一份文案。 */
export function contributionDraftTitle(raw: { label?: string; name?: string; variants?: Array<{ text?: string }> } | null | undefined): string {
  if (!raw) return "(无标题)";
  const title = (raw.label || raw.name || raw.variants?.[0]?.text || "").trim();
  return title || "(无标题)";
}

/** 存在已上线版本的投稿点赞率/系列完成率尾巴，不含日期也不含状态词——「点赞 86%」或
    「点赞 86% · 完成 100%」。修订草稿/审核中/被驳回期间旧版仍在线，照常展示；从未
    发布过的状态（或性别投稿，没有点赞概念）返回空串。
    后台审核详情要在日期后面插一段提交者昵称，没法直接吃下面拼好整串的
    formatContributionListMeta，所以单独导出这一截给它拼。 */
export function contributionVoteTail(opts: {
  kind: string;
  status: string;
  likeRatio?: number | null;
  completionRate?: number | null;
}): string {
  if (!contributionHasPublishedVersion(opts.status) || opts.kind === "gender") return "";
  const like = `点赞 ${opts.likeRatio ?? 0}%`;
  if (opts.kind === "series") return `${like} · 完成 ${opts.completionRate ?? 0}%`;
  return like;
}

/** 投稿列表/详情第二行文案：日期（+ 当前线上版本的点赞率/系列完成率）。不含状态词——状态词
    改由 <ContributionStatusChip>（AppViews.tsx）单独渲染成药丸徽标放在这行开头，两者拼起来
    是「待审批 26/08/20」或「已通过 26/08/20 · 点赞 86% · 完成 100%」（徽标与日期之间不加
    「·」，其后的点赞率/完成率等内容才用「·」分隔），不在文字里重复一遍状态。玩家端「参与
    共建」与后台共建审核统一用这一套。 */
export function formatContributionListMeta(opts: {
  kind: string;
  status: string;
  updatedAt: number;
  likeRatio?: number | null;
  completionRate?: number | null;
}): string {
  const day = formatContributionDay(opts.updatedAt);
  const tail = contributionVoteTail(opts);
  return tail ? `${day} · ${tail}` : day;
}

export function formatContributionReviewSubtitle(counts: { pending?: number; revisionPending?: number; unpublishPending?: number }): string {
  return `待审 ${counts.pending ?? 0} · 复审 ${counts.revisionPending ?? 0} · 下架 ${counts.unpublishPending ?? 0}`;
}

export const DIFFICULTY_GUIDE =
  "在同一房间中，用户受到的惩罚次数越多，下一次抽中惩罚任务的难度就越高。数字小的任务更偏向于前戏，数字大的任务往往更接近高潮。难度填写示例请看「关于」->「玩法介绍」->「参与共建」一节";

export const TAG_GUIDE = "请选择该任务适用的标签，若一个任务符合多个标签，可以同时选择多个。";
