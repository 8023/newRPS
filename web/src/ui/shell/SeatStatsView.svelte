<script lang="ts">
  // 源：ui/AppViews.tsx:3253-3261
  import type { RoomSnapshot } from "../../shared/types";
  let { stats }: { stats: RoomSnapshot["seatStats"]["A"] } = $props();
  const wins = $derived(Number.isFinite(Number(stats?.wins)) ? Number(stats.wins) : 0);
  const losses = $derived(Number.isFinite(Number(stats?.losses)) ? Number(stats.losses) : 0);
  const draws = $derived(Number.isFinite(Number(stats?.draws)) ? Number(stats.draws) : 0);
  const punishments = $derived(Number.isFinite(Number(stats?.punishments)) ? Number(stats.punishments) : 0);
  const rate = $derived.by(() => {
    const decisive = wins + losses;
    return decisive === 0 ? 0 : Math.round((wins / decisive) * 100);
  });
</script>

<small class="seat-stats">本席：{wins}胜 {losses}负 {draws}平 {punishments}惩罚 · 胜率 {rate}%</small>
