<script lang="ts">
  // 源：ui/AppViews.tsx:3968-4028
  import type { PublicPlayer } from "../../shared/types";
  import { GAME_LEADERBOARD_TABS, type GlobalLeaderboardTab, gameWLDOf, isGameLeaderboardTab } from "../../lib/leaderboard";
  import { safePlayerStats, formatGiveawayValue, totalOnlineMsOf, contributionApprovedCountOf } from "../../lib/playerDisplay";
  import { formatDuration, formatOnlineDuration } from "../../lib/format";

  let { player, tab, now }: { player: PublicPlayer; tab: GlobalLeaderboardTab; now: number } = $props();
</script>

{#if tab === "totalWins"}
  {@const s = safePlayerStats(player)}
  <p class="global-rank-extra">全部游戏胜场优先 · 总局 {s.wins + s.losses + s.draws}</p>
{:else if tab === "onlineTime"}
  <p class="global-rank-extra">累计在线时长优先 · {formatOnlineDuration(totalOnlineMsOf(player))}</p>
{:else if tab === "contribution"}
  <p class="global-rank-extra">共建审批通过条数优先 · {contributionApprovedCountOf(player)} 条</p>
{:else if isGameLeaderboardTab(tab)}
  {@const g = gameWLDOf(player, tab)}
  {@const label = GAME_LEADERBOARD_TABS.find((item) => item.id === tab)?.label || "该游戏"}
  <p class="global-rank-extra">{label} 胜场优先 · 总局 {g.wins + g.losses + g.draws}</p>
{:else if tab === "nameWar"}
  {@const protectedMs = player.nameWarRenameProtectedUntil ? Math.max(0, player.nameWarRenameProtectedUntil - now) : 0}
  <p class="global-rank-extra">
    {player.nameWarPunished ? `失名中：${player.nameWarPenaltyName || "惩罚名生效"}` : "名字争夺战开启"}
    {player.nameWarAllowRename ? " · 允许他人改名" : ""}
    {protectedMs > 0 ? ` · 保护 ${formatDuration(protectedMs)}` : ""}
  </p>
{:else if tab === "giveaway"}
  {@const boardMs = player.giveawayBoardExpiresAt ? Math.max(0, player.giveawayBoardExpiresAt - now) : 0}
  <p class="global-rank-extra">
    白给 {formatGiveawayValue(player.giveawayValue || 0)}%
    {player.giveawayBoardText && boardMs > 0 ? ` · 已上板 ${formatDuration(boardMs)}` : ""}
    {player.giveawayBoardText ? ` · 👍 ${player.giveawayBoardLikes || 0} / 👎 ${player.giveawayBoardDislikes || 0}` : ""}
  </p>
{:else if tab === "extremePositive" || tab === "extremeNegative"}
  <p class="global-rank-extra">⚡ 极限模式 · 连胜 {player.extremeWinStreak || 0}</p>
{/if}
