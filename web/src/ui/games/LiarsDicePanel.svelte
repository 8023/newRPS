<script lang="ts">
  // 源：ui/LiarsDicePanel.tsx
  import Dices from "@lucide/svelte/icons/dices";
  import type { RoomSnapshot, PublicPlayer } from "../../shared/types";
  import { seriesFactionWarning } from "../seriesFaction";
  import { ask } from "../../lib/rpc";
  import { socket } from "../../ws";
  import { roomPlayerById } from "../../lib/gameDisplay";
  import { displayPlayerName } from "../../lib/playerDisplay";
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import PlayerAvatar from "../shell/PlayerAvatar.svelte";
  import PlayerBadge from "../shell/PlayerBadge.svelte";

  const FACE_LABELS = ["", "⚀", "⚁", "⚂", "⚃", "⚄", "⚅"];
  function diceFace(value: number) {
    return FACE_LABELS[value] || String(value);
  }

  let { room, me, onError }: { room: RoomSnapshot; me: PublicPlayer; onError: (message: string) => void } = $props();

  function playerName(id: string, fallbackNames?: Record<string, string>) {
    const player = roomPlayerById(room, id);
    if (player) return displayPlayerName(player);
    return fallbackNames?.[id] || id;
  }

  const ld = $derived(room.liarsDice);
  let myDice = $state<number[]>([]);
  let bidCount = $state(1);
  let bidFace = $state(1);
  let busy = $state(false);

  $effect(() => {
    function onHand(data: { roomId?: string; dice?: number[] }) {
      if (data?.roomId === room.id && Array.isArray(data.dice)) myDice = data.dice;
    }
    socket.on("liarsdice:hand", onHand);
    return () => socket.off("liarsdice:hand", onHand);
  });

  $effect(() => {
    // 换局/我不在参战席时清掉手上骰子；PhaseResult 揭晓阶段改用 revealedHands 展示全员骰子。
    if (!ld || !ld.participantIds.includes(me.id) || room.phase === "result") return;
    if (ld.roundNumber === 0) myDice = [];
  });

  // 每次叫点变化（含轮到自己）时，默认选中"最小的可叫点数"：
  // 上家点数不为 6 则颗数不变、点数+1；点数为 6 则颗数+1、点数回到 1。
  $effect(() => {
    if (!ld) return;
    if (ld.currentBid) {
      if (ld.currentBid.face < 6) {
        bidCount = ld.currentBid.count;
        bidFace = ld.currentBid.face + 1;
      } else {
        bidCount = ld.currentBid.count + 1;
        bidFace = 1;
      }
    } else {
      bidCount = ld.participantIds.length + 1;
      bidFace = 1;
    }
  });

  const isParticipant = $derived(Boolean(ld?.participantIds.includes(me.id)));
  const isReady = $derived(Boolean(ld?.readyPlayerIds.includes(me.id)));
  const isMyTurn = $derived(room.phase === "choosing" && ld?.currentTurn === me.id);
  // 上家点数已是 6 时，同颗数已无有效点数可叫，颗数下限直接顶到 +1。
  const minCount = $derived(ld ? (ld.currentBid ? (ld.currentBid.face >= 6 ? ld.currentBid.count + 1 : ld.currentBid.count) : ld.participantIds.length + 1) : 1);
  const maxCount = $derived(ld ? ld.participantIds.length * 5 : 0);
  const minFace = $derived(ld?.currentBid && ld.currentBid.count === bidCount ? ld.currentBid.face + 1 : 1);

  async function act(event: string, payload: unknown = {}) {
    if (busy) return;
    busy = true;
    try {
      await ask(event, payload);
    } catch (error) {
      onError(error instanceof Error ? error.message : "操作失败");
    } finally {
      busy = false;
    }
  }

  function onBidCountChange(nextCount: number) {
    bidCount = nextCount;
    bidFace = ld?.currentBid && ld.currentBid.count === nextCount ? ld.currentBid.face + 1 : 1;
  }

  function submitBid() {
    if (!ld) return;
    let count = bidCount;
    let face = bidFace;
    if (ld.currentBid) {
      const valid = count > ld.currentBid.count || (count === ld.currentBid.count && face > ld.currentBid.face);
      if (!valid) {
        onError("叫点无效，必须比上家颗数更多，或颗数不变但点数更大");
        return;
      }
    } else if (count < ld.participantIds.length + 1) {
      count = ld.participantIds.length + 1;
    }
    act("liarsdice:bid", { count, face });
  }

  const faceOptions = [1, 2, 3, 4, 5, 6];
</script>

{#if ld}
  <div class="liarsdice-panel">
    <div class="liarsdice-roster panel">
      <h3><Dices size={18} /> 参战席（{ld.participantIds.length}{room.settings.liarsDiceMaxPlayers ? `/${room.settings.liarsDiceMaxPlayers}` : ""}）</h3>
      <ul class="liarsdice-roster-list">
        {#each ld.participantIds as id (id)}
          {@const player = roomPlayerById(room, id)}
          {@const isTurn = room.phase === "choosing" && ld.currentTurn === id}
          {@const ready = ld.readyPlayerIds.includes(id)}
          <li class={isTurn ? "current-turn" : ""}>
            {#if player}
              <span class="seat-occupant-row">
                <PlayerAvatar {player} size={24} />
                <PlayerBadge {player} compact />
              </span>
            {:else}
              <span>{id}</span>
            {/if}
            {#if room.phase === "ready"}<em>{ready ? "已准备" : "未准备"}</em>{/if}
            {#if isTurn}<em class="liarsdice-turn-flag">叫点中</em>{/if}
            {#if typeof ld.diceCounts?.[id] === "number"}<small>🎲 {ld.diceCounts[id]}</small>{/if}
          </li>
        {/each}
      </ul>
      {#if room.phase === "ready" || room.phase === "waiting"}
        <div class="liarsdice-roster-actions">
          {#if !isParticipant}
            <button class="primary" disabled={busy || ld.participantIds.length >= (room.settings.liarsDiceMaxPlayers || 8)} onclick={() => {
              const warn = sessionStore.config ? seriesFactionWarning(room, sessionStore.config, me) : null;
              if (warn && !window.confirm(warn)) return;
              act("liarsdice:joinRoster");
            }}>加入参战席</button>
          {/if}
          {#if isParticipant}
            <button disabled={busy} onclick={() => act("liarsdice:leaveRoster")}>离开参战席</button>
            {#if !isReady}<button class="primary" disabled={busy} onclick={() => act("liarsdice:ready")}>已准备</button>{/if}
            {#if isReady}<span class="hint">已准备，等待其他人（{room.settings.liarsDiceMinPlayers || 3} 人以上且 5 秒无变动自动开局）</span>{/if}
          {/if}
        </div>
      {/if}
    </div>

    {#if isParticipant && myDice.length > 0 && room.phase !== "result" && room.phase !== "punishment"}
      <div class="liarsdice-my-hand panel">
        <h4>我的骰子</h4>
        <div class="liarsdice-dice-row">
          {#each myDice as d, i (i)}<span class="liarsdice-die">{diceFace(d)}</span>{/each}
        </div>
      </div>
    {/if}

    {#if room.phase === "choosing"}
      <div class="liarsdice-bid-panel panel">
        <h4>{ld.currentBid ? `当前叫点：${ld.currentBid.count} 个 ${ld.currentBid.face}（${playerName(ld.currentBid.playerId)}）` : "还没有人叫点"}</h4>
        {#if ld.onesWildDisabled}<p class="hint">本局已有人叫过 1，此后 1 不再视为万能点。</p>{/if}
        {#if isMyTurn}
          <div class="liarsdice-bid-controls">
            <label>
              颗数
              <select value={bidCount} onchange={(event) => onBidCountChange(Number(event.currentTarget.value))}>
                {#each Array.from({ length: Math.max(0, maxCount - minCount + 1) }, (_, i) => minCount + i) as count (count)}
                  <option value={count}>{count}</option>
                {/each}
              </select>
            </label>
            <label>
              点数
              <select value={bidFace} onchange={(event) => (bidFace = Number(event.currentTarget.value))}>
                {#each faceOptions as face (face)}
                  <option value={face} disabled={bidCount === (ld.currentBid?.count ?? 0) && face < (minFace ?? 1)}>{face}</option>
                {/each}
              </select>
            </label>
            <button class="primary" disabled={busy} onclick={submitBid}>叫点</button>
            {#if isParticipant && ld.currentBid && ld.currentBid.playerId !== me.id}
              <button class="liarsdice-challenge-btn" disabled={busy} onclick={() => act("liarsdice:challenge")}>开牌</button>
            {/if}
          </div>
        {:else}
          <p class="hint">等待 {playerName(ld.currentTurn || "")} 叫点...</p>
        {/if}
        <!-- 不能开自己叫的点：叫点者本人在此不再露出开牌按钮，避免赢家=输家=自己导致后续惩罚流程卡死 -->
        {#if !isMyTurn && isParticipant && ld.currentBid && ld.currentBid.playerId !== me.id}
          <button class="liarsdice-challenge-btn" disabled={busy} onclick={() => act("liarsdice:challenge")}>开牌</button>
        {/if}
      </div>
    {/if}

    {#if (room.phase === "result" || room.phase === "punishment") && ld.ended && ld.revealedHands}
      <div class="liarsdice-reveal panel">
        <h4>开牌结果</h4>
        <p>
          叫点 {ld.currentBid?.count} 个 {ld.currentBid?.face || 0}，实际 {ld.actualCount} 个
          {ld.onesWildDisabled ? "（1 不算万能点）" : "（1 可算万能点）"}
        </p>
        <ul class="liarsdice-reveal-list">
          {#each ld.participantIds as id (id)}
            <li class={id === ld.winnerId ? "winner" : id === ld.loserId ? "loser" : ""}>
              <span>{playerName(id)}</span>
              <span class="liarsdice-dice-row">
                {#each ld.revealedHands?.[id] || [] as d, i (i)}<span class="liarsdice-die small">{diceFace(d)}</span>{/each}
              </span>
              {#if id === ld.winnerId}<em>胜</em>{/if}
              {#if id === ld.loserId}<em>负</em>{/if}
            </li>
          {/each}
        </ul>
        <!-- 惩罚未完成前后端拒绝 liarsdice:nextRound（要求 phase===result），惩罚阶段先隐藏按钮 -->
        {#if isParticipant && room.phase === "result"}
          <button class="primary liarsdice-next-round-btn" disabled={busy} onclick={() => act("liarsdice:nextRound")}>再来一局</button>
        {/if}
      </div>
    {/if}
  </div>
{/if}
