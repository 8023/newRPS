<script lang="ts">
  // 源：ui/AppViews.tsx:3202-3251
  import type { RoomSnapshot, PublicPlayer, SeatKey } from "../../shared/types";
  import { choiceText, jungleSideLabel, chessSideLabel } from "../../lib/gameDisplay";
  import PlayerAvatar from "./PlayerAvatar.svelte";
  import PlayerBadge from "./PlayerBadge.svelte";
  import OfflineBadge from "./OfflineBadge.svelte";
  import SeatStatsView from "./SeatStatsView.svelte";

  let { seat, room, me, onSit }: { seat: SeatKey; room: RoomSnapshot; me: PublicPlayer; onSit: () => void } = $props();

  const occupant = $derived(room.seats[seat]);
  const choice = $derived(room.revealedChoices?.[seat] || (occupant?.id === me.id ? room.choices[seat] : room.choices[seat] ? "hidden" : undefined));
  const stats = $derived(room.seatStats[seat]);
  const battleSeatBlocked = $derived(Boolean(room.settings.enableRanked && Boolean(me.extremeModeEnabled) !== Boolean(room.settings.enableExtremeRanked)));
  const othelloTurn = $derived(room.settings.gameId === "othello" && room.othello?.turn === seat && room.phase === "choosing" && !room.othello.ended);
  const othelloColorLabel = $derived(
    room.settings.gameId === "othello" && room.othello
      ? (room.othello.blackSeat === seat ? "⚫ 黑棋" : "⚪ 白棋")
      : "随机后显示黑/白"
  );
  const tictactoeTurn = $derived(room.settings.gameId === "tictactoe" && room.tictactoe?.turn === seat && room.phase === "choosing" && !room.tictactoe.ended);
  const tictactoeMarkLabel = $derived(
    room.settings.gameId === "tictactoe" && room.tictactoe
      ? (room.tictactoe.xSeat === seat ? "❌ X" : "⭕ O")
      : "随机后显示 X/O"
  );
  const gomokuTurn = $derived(room.settings.gameId === "gomoku" && room.gomoku?.turn === seat && room.phase === "choosing" && !room.gomoku.ended);
  const gomokuMarkLabel = $derived(
    room.settings.gameId === "gomoku" && room.gomoku
      ? (room.gomoku.blackSeat === seat ? "⚫ 黑棋" : "⚪ 白棋")
      : "随机后显示黑/白"
  );
  const jungleTurn = $derived(room.settings.gameId === "jungle" && room.jungle?.turn === seat && room.phase === "choosing" && !room.jungle.ended);
  const jungleMarkLabel = $derived(room.settings.gameId === "jungle" ? jungleSideLabel(seat) : "");
  const chessTurn = $derived(room.settings.gameId === "chess" && room.chess?.turn === seat && room.phase === "choosing" && !room.chess.ended);
  const chessMarkLabel = $derived(room.settings.gameId === "chess" ? chessSideLabel(room.chess, seat) : "");
</script>

<div class={`seat-card seat-${seat.toLowerCase()}`}>
  <div class="seat-identity">
    <span class="seat-label">玩家 {seat}</span>
    {#if occupant}
      <strong class="seat-occupant-row">
        <PlayerAvatar player={occupant} size={24} />
        <PlayerBadge player={occupant} compact />
      </strong>
    {:else}
      <button disabled={battleSeatBlocked} title={battleSeatBlocked ? "当前排位类型不匹配，只能观战" : "坐到战斗席"} onclick={onSit}>
        {battleSeatBlocked ? "👀 只能观战" : "🪑 坐下"}
      </button>
    {/if}
  </div>
  {#if occupant}<OfflineBadge player={occupant} />{/if}
  <p class="choice-badge">
    {#if room.settings.gameId === "othello"}{othelloTurn ? `${othelloColorLabel}落子中` : othelloColorLabel}
    {:else if room.settings.gameId === "tictactoe"}{tictactoeTurn ? `${tictactoeMarkLabel}落子中` : tictactoeMarkLabel}
    {:else if room.settings.gameId === "gomoku"}{gomokuTurn ? `${gomokuMarkLabel}落子中` : gomokuMarkLabel}
    {:else if room.settings.gameId === "jungle"}{jungleTurn ? `${jungleMarkLabel} 走子中` : jungleMarkLabel}
    {:else if room.settings.gameId === "chess"}{chessTurn ? `${chessMarkLabel} 走子中` : chessMarkLabel}
    {:else}{choice ? choiceText(choice) : room.seats.A && room.seats.B ? "🤔 等待出拳" : "⏳ 等人"}{/if}
  </p>
  {#if occupant}<SeatStatsView {stats} />{/if}
</div>
