<script lang="ts">
  // 源：ui/AppViews.tsx:2251-2364
  import type { RoomSnapshot, PublicPlayer } from "../../shared/types";
  import { occupantDisplay, tictactoeDeltaText, tictactoeThemeStyle } from "../../lib/gameDisplay";
  import { formatGiveawayValue } from "../../lib/playerDisplay";
  import { useNow } from "../../lib/useNow.svelte";
  import { styleString } from "../../lib/style";

  let { room, me, onMove, onReady, onRestart, onGiveawayChoice }: {
    room: RoomSnapshot;
    me: PublicPlayer;
    onMove: (row: number, col: number) => void;
    onReady: () => void;
    onRestart: () => void;
    onGiveawayChoice: (mode: "normal" | "giveaway") => void;
  } = $props();

  const state = $derived(room.tictactoe);
  const mySeat = $derived(room.seats.A?.id === me.id ? "A" : room.seats.B?.id === me.id ? "B" : null);
  const giveawayPrompt = $derived(state?.giveawayPrompt);
  const giveawayBlockingTurn = $derived(Boolean(giveawayPrompt && giveawayPrompt.seat === state?.turn));
  const isMyGiveawayPrompt = $derived(Boolean(state && mySeat && giveawayPrompt?.seat === mySeat && room.phase === "choosing" && !state.ended));
  const isMyTurn = $derived(Boolean(state && mySeat && state.turn === mySeat && room.phase === "choosing" && !state.ended && !giveawayBlockingTurn));
  const waitingForReady = $derived(room.phase === "ready" && Boolean(room.seats.A && room.seats.B));
  const drawingFirst = $derived(waitingForReady && room.ready.A && room.ready.B);
  const myReady = $derived(mySeat ? room.ready[mySeat] : false);
  const turnName = $derived(state?.turn === "A" ? occupantDisplay(room.seats.A) : occupantDisplay(room.seats.B));
  const giveawayPromptName = $derived(giveawayPrompt?.seat === "A" ? occupantDisplay(room.seats.A) : occupantDisplay(room.seats.B));
  const now = useNow(() => 1000, () => Boolean(giveawayPrompt));
  const giveawaySecondsLeft = $derived(giveawayPrompt ? Math.max(0, Math.ceil((giveawayPrompt.expiresAt - now.value) / 1000)) : 0);
  const tictactoeGiveawayGain = formatGiveawayValue(0.3);
  const winningKeys = $derived(new Set((state?.winningLine || []).map((cell) => `${cell.row}-${cell.col}`)));
  const board = $derived(state?.board || Array.from({ length: 3 }, () => Array.from({ length: 3 }, () => null)));
  const boardTheme = $derived(styleString(tictactoeThemeStyle(room.settings.tictactoeBoardTheme)));
</script>

<div class="tictactoe-panel">
  <div class="tictactoe-head">
    <div>
      <h3>❌⭕ 井字棋</h3>
      <p class="hint">
        {#if !room.seats.A || !room.seats.B}等待两个战斗席坐满。
        {:else if drawingFirst}正在随机 X/O 先手...
        {:else if waitingForReady}双方准备后随机决定谁执 X 先手。
        {:else if state?.ended}{room.resultText || "对局结束"}
        {:else if giveawayPrompt?.forced}强制白给中，系统正在随机落子...
        {:else if giveawayPrompt}{isMyGiveawayPrompt ? "请选择不白给或白给落子。" : `等待 ${giveawayPromptName} 选择是否白给。`}
        {:else}{isMyTurn ? "轮到你落子。" : `轮到 ${turnName} 落子。`}{/if}
      </p>
    </div>
    {#if state}
      <div class="tictactoe-turn-card">
        <span>{state.xSeat === state.turn ? "❌ X" : "⭕ O"}</span>
        <strong>{state.moveCount} / 9</strong>
        {#if room.settings.enableRanked && state.rankedDelta}<small>本局排位：X {tictactoeDeltaText(state, "X")} / O {tictactoeDeltaText(state, "O")}</small>{/if}
      </div>
    {/if}
  </div>
  {#if waitingForReady}
    <div class={`tictactoe-ready-card ${drawingFirst ? "drawing" : ""}`}>
      <div class="tictactoe-draw-animation" aria-hidden="true"><span>❌</span><span>⭕</span><span>❌</span></div>
      <div>
        <strong>{drawingFirst ? "抽签中..." : myReady ? "你已准备" : "准备开始"}</strong>
        <p class="hint">A：{room.ready.A ? "已准备" : "未准备"} · B：{room.ready.B ? "已准备" : "未准备"}</p>
      </div>
      {#if mySeat && !myReady && !drawingFirst}<button class="primary" onclick={onReady}>准备</button>{/if}
      {#if mySeat && myReady && !drawingFirst}<button disabled>等待对方</button>{/if}
    </div>
  {/if}
  {#if giveawayPrompt && room.phase === "choosing" && !state?.ended}
    <div class={`othello-settlement-card tictactoe-giveaway-card ${giveawayPrompt.forced ? "forced" : ""}`}>
      <div>
        <strong>{giveawayPrompt.forced ? "强制白给" : isMyGiveawayPrompt ? "选择本手结算" : "等待本手结算"}</strong>
        <p class="hint">
          {#if giveawayPrompt.forced}{giveawayPromptName} 触发强制白给，将在 {giveawaySecondsLeft} 秒后自动随机乱下。
          {:else if isMyGiveawayPrompt}{giveawaySecondsLeft} 秒后自动选择不白给。
          {:else}等待 {giveawayPromptName} 选择，{giveawaySecondsLeft} 秒后默认不白给。{/if}
        </p>
        <p class="othello-settlement-help">不白给：正常手动落子；白给：系统随机选择一个空格落子，白给值 +{tictactoeGiveawayGain}%。</p>
      </div>
      {#if giveawayPrompt.forced}
        <div class="othello-settlement-forced"><span>🫴 白给乱下</span></div>
      {:else if isMyGiveawayPrompt}
        <div class="othello-settlement-actions tictactoe-giveaway-actions">
          <button class="primary" onclick={() => onGiveawayChoice("normal")}>不白给</button>
          <button class="soft-button" onclick={() => onGiveawayChoice("giveaway")}>白给 +{tictactoeGiveawayGain}%</button>
        </div>
      {:else}
        <div class="othello-settlement-forced"><span>⏳ 等待选择</span></div>
      {/if}
    </div>
  {/if}
  {#if state?.ended && mySeat && room.phase === "result"}
    <button class="primary tictactoe-restart-button" onclick={onRestart}>再来一局</button>
  {/if}
  <div class="tictactoe-board" role="grid" aria-label="井字棋棋盘" style={boardTheme}>
    {#each board as row, rowIndex (rowIndex)}
      {#each row as cell, colIndex (colIndex)}
        {@const winning = winningKeys.has(`${rowIndex}-${colIndex}`)}
        <button
          type="button"
          class={`tictactoe-cell ${cell ? cell.toLowerCase() : ""} ${winning ? "winning" : ""}`}
          disabled={!isMyTurn || Boolean(cell)}
          onclick={() => onMove(rowIndex, colIndex)}
          aria-label={`第 ${rowIndex + 1} 行第 ${colIndex + 1} 列`}
        >
          {cell ? (cell === "X" ? "×" : "○") : ""}
        </button>
      {/each}
    {/each}
  </div>
</div>
