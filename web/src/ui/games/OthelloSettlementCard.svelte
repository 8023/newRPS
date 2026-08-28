<script lang="ts">
  // 源：ui/AppViews.tsx:2603-2641
  import type { RoomSnapshot, PublicPlayer } from "../../shared/types";
  import { occupantDisplay } from "../../lib/gameDisplay";
  import { formatGiveawayValue } from "../../lib/playerDisplay";
  import { useNow } from "../../lib/useNow.svelte";

  let { room, me, pending, onSettle }: {
    room: RoomSnapshot;
    me: PublicPlayer;
    pending: NonNullable<NonNullable<RoomSnapshot["othello"]>["pendingSettlement"]>;
    onSettle: (mode: "normal" | "giveaway" | "tribute") => void;
  } = $props();

  const isMine = $derived(room.seats[pending.seat]?.id === me.id);
  const actorName = $derived(occupantDisplay(room.seats[pending.seat]));
  const opponentName = $derived(occupantDisplay(room.seats[pending.opponentSeat]));
  const now = useNow(() => 1000, () => true);
  const secondsLeft = $derived(Math.max(0, Math.ceil((pending.expiresAt - now.value) / 1000)));
  const forcedText = $derived(pending.forced === "tribute" ? "强制上贡" : pending.forced === "giveaway" ? "强制白给" : "");
  const giveawayGain = $derived(formatGiveawayValue(pending.flips * 0.1));
  const tributeGain = $derived(formatGiveawayValue(pending.flips * 0.2));
</script>

<div class={`othello-settlement-card ${pending.forced ? "forced" : ""}`}>
  <div>
    <strong>{pending.forced ? forcedText : isMine ? "选择本手结算" : "等待本手结算"}</strong>
    <p class="hint">
      {actorName} 本手翻 {pending.flips} 子，基础分 {pending.stake}。
      {#if pending.forced}{forcedText}将在 {secondsLeft} 秒后自动结算。
      {:else if isMine}{secondsLeft} 秒后自动选择不白给。
      {:else}等待 {actorName} 选择，{secondsLeft} 秒后默认不白给。{/if}
    </p>
    <p class="othello-settlement-help">
      不白给：本手按 {pending.stake} 分正常结算；白给：本手不结算排位分，白给值 +{giveawayGain}%；上贡：对方拿本手收益，你的白给值 +{tributeGain}%。
    </p>
  </div>
  {#if pending.forced}
    <div class="othello-settlement-forced"><span>{pending.forced === "tribute" ? "🎁 上贡给对方" : "🫴 白给本手"}</span></div>
  {:else if isMine}
    <div class="othello-settlement-actions">
      <button class="primary" onclick={() => onSettle("normal")}>不白给</button>
      <button class="soft-button" onclick={() => onSettle("giveaway")}>白给 +{giveawayGain}%</button>
      <button class="soft-button danger-soft" onclick={() => onSettle("tribute")}>上贡给 {opponentName} +{tributeGain}%</button>
    </div>
  {:else}
    <div class="othello-settlement-forced"><span>⏳ 等待选择</span></div>
  {/if}
</div>
