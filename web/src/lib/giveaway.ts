// 白给投票额度展示换算。源：ui/AppViews.tsx:4265-4281。
import type { GiveawayVoteQuota } from "../shared/types";

/** 把 giveaway:vote / giveaway:voteQuotas 回包的原始额度换算成展示文案，窗口过期按本地时钟
 * 自动清零（与服务端懒重置逻辑一致，见 giveawayVoteQuotaFor）。quota 未拿到（还在首次加载）
 * 时调用方应跳过渲染，而不是当作满额度显示——避免闪一下错误数字。 */
export function describeGiveawayVoteQuota(quota: GiveawayVoteQuota | undefined, now: number) {
  if (!quota) return null;
  const windowMs = 3_600_000;
  const expired = !quota.windowStartedAt || now - quota.windowStartedAt >= windowMs;
  const likesUsed = expired ? 0 : quota.likesUsed;
  const dislikesUsed = expired ? 0 : quota.dislikesUsed;
  const likesLeft = Math.max(0, quota.likeLimit - likesUsed);
  const dislikesLeft = Math.max(0, quota.dislikeLimit - dislikesUsed);
  const base = `👍 还可 ${likesLeft}/${quota.likeLimit} · 👎 还可 ${dislikesLeft}/${quota.dislikeLimit}`;
  if (expired) return { likesLeft, dislikesLeft, text: base };
  // 刷新文案统一用「分钟」单位（向下取整），例如 2 小时 → 120 分钟；不足 1 分钟 → 0 分钟
  const remainMinutes = Math.floor(Math.max(0, quota.windowStartedAt + windowMs - now) / 60_000);
  return { likesLeft, dislikesLeft, text: `${base} · ${remainMinutes} 分钟后刷新` };
}
