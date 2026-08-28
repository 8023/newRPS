<script lang="ts">
  // 源：ui/AppViews.tsx:1445-1475。建房面板的游戏选择卡片图标。
  import ChessKnight from "@lucide/svelte/icons/chess-knight";
  import type { RoomSettings } from "../../shared/types";

  let { gameId }: { gameId: RoomSettings["gameId"] } = $props();
</script>

{#if gameId === "othello"}
  <!-- 黑白棋是两个并排的宽体 emoji，默认字号会撑破方形图标框、溢出圆角边框，缩小字号让它们老实待在框内。 -->
  <span class="game-choice-icon-pair">⚫⚪</span>
{:else if gameId === "tictactoe"}
  <!-- 井字棋/五子棋两个图标沿对角线排布，与黑白棋的水平并排区分开。 -->
  <span class="game-choice-icon-diagonal">
    <span>❌</span>
    <span>⭕</span>
  </span>
{:else if gameId === "gomoku"}
  <span class="game-choice-icon-diagonal game-choice-icon-diagonal-text">
    <span>●</span>
    <span>○</span>
  </span>
{:else if gameId === "jungle"}
  🦁
{:else if gameId === "chess"}
  <ChessKnight class="game-choice-icon-chess" size={30} strokeWidth={1.9} />
{:else if gameId === "liarsdice"}
  🎲
{:else if gameId === "coinflip"}
  🪙
{:else}
  <!-- 三个手势倒三角排布（上两下一），避免挤在一行导致溢出裁切。 -->
  <span class="game-choice-icon-rps">
    <span>✊</span>
    <span>✌️</span>
    <span>🖐️</span>
  </span>
{/if}
