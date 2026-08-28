<script lang="ts">
  // 黑白棋/五子棋/斗兽棋/国际象棋共用：棋盘上方的每子/每局倒计时条，未启用对应计时器时不显示。
  // 源：ui/AppViews.tsx:2388-2426
  import type { RoomSnapshot, SeatKey } from "../../shared/types";
  import { formatClockMs } from "../../lib/gameDisplay";
  import { useServerNow } from "../../lib/useNow.svelte";

  type TimedGameState = {
    turn: SeatKey;
    blackSeat: SeatKey;
    ended?: boolean;
    moveDeadlineAt?: number;
    clockDeadlineAt?: number;
    clockRemaining?: Record<SeatKey, number>;
  };

  let { room, state, moveSeconds, gameMinutes, labels }: {
    room: RoomSnapshot;
    state?: TimedGameState;
    moveSeconds?: number;
    gameMinutes?: number;
    /** 双方总时长标签；默认 ⚫/⚪（黑白棋、五子棋） */
    labels?: { primary: string; secondary: string };
  } = $props();

  const ticking = $derived(Boolean(state && !state.ended && room.phase === "choosing" && (moveSeconds || gameMinutes)));
  const now = useServerNow(() => 250, () => ticking);

  const shouldRender = $derived(Boolean(state && !state.ended && room.phase === "choosing" && (moveSeconds || gameMinutes)));
  const whiteSeat = $derived<SeatKey | null>(state ? (state.blackSeat === "A" ? "B" : "A") : null);
  const primaryLabel = $derived(labels?.primary ?? "⚫");
  const secondaryLabel = $derived(labels?.secondary ?? "⚪");
  const moveSecondsLeft = $derived.by(() => {
    if (!state || !moveSeconds) return null;
    const deadline = Number(state.moveDeadlineAt) || 0;
    if (!deadline) return null;
    // 网络往返中点估算仍可能有几毫秒误差；显示值再按建房上限收口，绝不出现
    // "每子 30 秒"刚刷新却显示 31 秒的情况。
    return Math.min(moveSeconds, Math.max(0, Math.ceil((deadline - now.value) / 1000)));
  });

  function remainingMsFor(seat: SeatKey): number {
    if (!state) return 0;
    const clockAt = Number(state.clockDeadlineAt) || 0;
    if (state.turn === seat && clockAt) return Math.max(0, clockAt - now.value);
    return Number(state.clockRemaining?.[seat]) || 0;
  }
</script>

{#if shouldRender && state}
  <div class="game-clock-bar">
    {#if moveSeconds && moveSecondsLeft !== null}
      <span class={`game-clock-move ${moveSecondsLeft <= 10 ? "urgent" : ""}`}>本步剩余 {moveSecondsLeft} 秒</span>
    {/if}
    {#if gameMinutes}
      <span class="game-clock-total">
        {primaryLabel} {formatClockMs(remainingMsFor(state.blackSeat))} · {secondaryLabel} {formatClockMs(remainingMsFor(whiteSeat!))}
      </span>
    {/if}
  </div>
{/if}
