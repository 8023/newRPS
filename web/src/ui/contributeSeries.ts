/** 系列步数纯防御性的技术硬顶：与后端 contributionstore.go 的 maxSeriesSteps 常量保持一致，
    只用来挡住畸形/超大 payload，不是玩法层面的实际约束——后台「最高步数」配置在这个范围内
    填多大都真实生效，不会被静默收紧。 */
export const MAX_SERIES_STEPS = 1000;
/** 后台「最高步数」留空/填非正数时的默认值，与上面的技术硬顶是两回事，见
    effectiveMaxSeriesSteps 的说明。 */
export const DEFAULT_MAX_SERIES_STEPS = 20;
export const DEFAULT_MIN_SERIES_STEPS = 5;
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
  /** 允许 ""（编辑中途清空输入框的过渡态）；提交/发布前必须先用 isValidOrder 校验掉。 */
  order: number | "";
  tagIds: string[];
  backgroundImage: string;
  backgroundOpacity: number;
};

/** 难度是否为可以提交/发布的合法值：1-99 之间的整数，"" 或越界都不算。 */
export function isValidOrder(order: number | ""): order is number {
  return typeof order === "number" && Number.isInteger(order) && order >= 1 && order <= 99;
}

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
      // 兜底转成数字：真正拦截空值/越界的是 canSubmit（isValidOrder），这里只是防止
      // "" 被 JSON.stringify 成字符串，导致后端 Order 字段类型不匹配、报出不友好的
      // "格式错误"而不是「难度必须在 1 到 99 之间」。
      order: step.inRandomPool ? (typeof step.order === "number" ? step.order : 0) : -1,
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

/** 投稿详情里的日期：精确到天，YY/MM/DD（如「26/08/21」）。 */
export function formatContributionDay(ms: number): string {
  const d = new Date(ms);
  if (!Number.isFinite(d.getTime())) return "";
  const yy = String(d.getFullYear()).slice(-2);
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${yy}/${mm}/${dd}`;
}

/** 投稿列表里的日期：不带年份，MM/DD（如「08/21」）——列表本就挤在窄侧栏里，年份挤占
    空间又意义不大；详情页仍用带年份的 formatContributionDay。 */
export function formatContributionDayShort(ms: number): string {
  const d = new Date(ms);
  if (!Number.isFinite(d.getTime())) return "";
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${mm}/${dd}`;
}

// 状态直接落在 sub_tasks/series 每一行上（见 shared/types.ts 的 ContributionItem 注释），
// 不再区分"初审"和"修订审核"——同一个状态值文案统一，不需要分组重复维护。
export const contributionStatusLabel: Record<string, string> = {
  draft: "草稿",
  pending: "待审批",
  approved: "已通过",
  rejected: "已驳回",
  withdrawn: "已撤回",
};

/** 当前是不是"正式生效"的一版——只有 approved 有点赞率/完成率可看，也只有它会真正出现在
    随机任务池/建房系列目录里。编辑一份已通过的投稿会另起草稿版本（见 saveDraft 的注释），
    这条投稿本身的状态就会变回 draft，不再计入"已发布"。 */
export function contributionHasPublishedVersion(status: string): boolean {
  return status === "approved";
}

/** 投稿标题：系列用 name（玩家/管理员都要靠它认出是哪个系列，会展示给玩家）；单个随机
    任务不设可读标题（玩家/管理员都不需要看到），退到第一份文案。 */
export function contributionDraftTitle(raw: { name?: string; variants?: Array<{ text?: string }> } | null | undefined): string {
  if (!raw) return "(无标题)";
  const title = (raw.name || raw.variants?.[0]?.text || "").trim();
  return title || "(无标题)";
}

/** 已通过投稿的点赞率/系列完成率尾巴，不含日期也不含状态词——「点赞 86%」或
    「点赞 86% · 完成 100%」。非 approved 状态返回空串。
    后台审核详情要在日期后面插一段提交者昵称，没法直接吃下面拼好整串的
    formatContributionListMeta，所以单独导出这一截给它拼。 */
export function contributionVoteTail(opts: {
  kind: string;
  status: string;
  likeRatio?: number | null;
  completionRate?: number | null;
}): string {
  if (!contributionHasPublishedVersion(opts.status)) return "";
  const like = `点赞 ${opts.likeRatio ?? 0}%`;
  if (opts.kind === "series") return `${like} · 完成 ${opts.completionRate ?? 0}%`;
  return like;
}

/** 投稿列表/详情第二行文案：日期（+ 当前线上版本的点赞率/系列完成率）。不含状态词——状态词
    改由 <ContributionStatusChip>（AppViews.tsx）单独渲染成药丸徽标放在这行开头，两者拼起来
    是「待审批 08/20」或「已通过 08/20 · 点赞 86% · 完成 100%」（徽标与日期之间不加
    「·」，其后的点赞率/完成率等内容才用「·」分隔），不在文字里重复一遍状态。玩家端「参与
    共建」与后台共建审核统一用这一套。short 为真时日期不带年份（列表用），为假/不传时
    保留 YY/MM/DD（详情用）。 */
export function formatContributionListMeta(opts: {
  kind: string;
  status: string;
  updatedAt: number;
  likeRatio?: number | null;
  completionRate?: number | null;
  short?: boolean;
}): string {
  const day = opts.short ? formatContributionDayShort(opts.updatedAt) : formatContributionDay(opts.updatedAt);
  const tail = contributionVoteTail(opts);
  return tail ? `${day} · ${tail}` : day;
}

export function formatContributionReviewSubtitle(counts: { pending?: number }): string {
  return `待审 ${counts.pending ?? 0} 条`;
}

/** 点赞率：后端只下发原始的 likeCount/downCount（这一个 id 名下所有历史版本行的赞踩计数
    之和），点赞率由前端就地算，不需要后端再算一遍传一个 ratio 字段。没有任何评价时返回 0
    （调用方结合 likeCount+downCount 是否为 0 决定要不要展示这一截）。 */
export function voteRatio(likeCount?: number, downCount?: number): number {
  const total = (likeCount ?? 0) + (downCount ?? 0);
  if (total <= 0) return 0;
  return Math.round(((likeCount ?? 0) / total) * 100);
}

export const DIFFICULTY_GUIDE =
  "在同一房间中，用户受到的惩罚次数越多，下一次抽中惩罚任务的难度就越高。数字小的任务更偏向于前戏，数字大的任务往往更接近高潮。难度填写示例请看「关于」->「玩法介绍」->「参与共建」一节";

export const TAG_GUIDE = "请选择该任务适用的标签，若一个任务符合多个标签，可以同时选择多个。";
