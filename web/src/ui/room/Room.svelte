<script lang="ts">
  // 房间主视图：座位+比分、当前游戏面板、观战/强制白给、惩罚阶段、房间玩家名单、聊天、
  // 对局记录、图片预览。原 React 版把这些全部揉进一个 ~670 行的 Room 组件；这里按职责
  // 拆成 RpsPanel/OthelloPanel/.../PunishmentPhase/RoomPlayerListPanel/RoomChatSection/
  // RoundHistoryPanel 等独立组件，Room.svelte 只保留"这个房间真正只有它自己需要"的状态
  // （座位判定、petbond 订阅、图片预览、各游戏 RPC 转发），直接读 sessionStore，不接收 props。
  import type { PetBondState, PublicPlayer } from "../../shared/types";
  import Swords from "@lucide/svelte/icons/swords";
  import DoorOpen from "@lucide/svelte/icons/door-open";
  import Eye from "@lucide/svelte/icons/eye";
  import HeartHandshake from "@lucide/svelte/icons/heart-handshake";
  import { socket } from "../../ws";
  import { ask } from "../../lib/rpc";
  import { rankMultiplierForSettings } from "../../lib/playerDisplay";
  import { roomPlayerList } from "../../lib/gameDisplay";
  import { styleString } from "../../lib/style";
  import { roomInfoTags } from "../../lib/roomInfoTags";
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { routerStore } from "../../lib/stores/routerStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import { seriesFactionWarning } from "../seriesFaction";
  import ExtremeRankedBadge from "../shell/ExtremeRankedBadge.svelte";
  import RankMultiplierBadge from "../shell/RankMultiplierBadge.svelte";
  import RoomTagList from "../shell/RoomTagList.svelte";
  import RoomInfoTagList from "../shell/RoomInfoTagList.svelte";
  import SeatView from "../shell/SeatView.svelte";
  import Settlement from "../shell/Settlement.svelte";
  import RpsPanel from "../games/RpsPanel.svelte";
  import OthelloPanel from "../games/OthelloPanel.svelte";
  import OthelloScore from "../games/OthelloScore.svelte";
  import TicTacToePanel from "../games/TicTacToePanel.svelte";
  import TicTacToeScore from "../games/TicTacToeScore.svelte";
  import GomokuPanel from "../games/GomokuPanel.svelte";
  import GomokuScore from "../games/GomokuScore.svelte";
  import JunglePanel from "../games/JunglePanel.svelte";
  import JungleScore from "../games/JungleScore.svelte";
  import ChessPanel from "../games/ChessPanel.svelte";
  import ChessScore from "../games/ChessScore.svelte";
  import LiarsDicePanel from "../games/LiarsDicePanel.svelte";
  import CoinFlipPanel from "../games/CoinFlipPanel.svelte";
  import PunishmentPhase from "./PunishmentPhase.svelte";
  import RoomPlayerListPanel from "./RoomPlayerListPanel.svelte";
  import RoomChatSection from "./RoomChatSection.svelte";
  import RoundHistoryPanel from "./RoundHistoryPanel.svelte";

  const room = $derived(sessionStore.room!);
  const me = $derived(sessionStore.me!.player);
  const config = $derived(sessionStore.config!);

  function onError(message: string) {
    uiStore.notify(message);
  }

  const seats = $derived(room.seats || { A: null, B: null });
  const mySeat = $derived(seats.A?.id === me.id ? "A" : seats.B?.id === me.id ? "B" : null);
  const canGoSpectate = $derived(Boolean(
    mySeat && room.phase !== "punishment" && !room.choices[mySeat] &&
    !((room.settings.gameId === "tictactoe" || room.settings.gameId === "gomoku" || room.settings.gameId === "jungle" || room.settings.gameId === "chess") && room.phase === "choosing")
  ));
  const opponentSeat = $derived(mySeat === "A" ? "B" : mySeat === "B" ? "A" : null);
  const opponentPlayer = $derived(opponentSeat ? seats[opponentSeat] : null);
  const forceGiveawayEligibleGame = $derived(
    room.settings.gameId === "rps" || room.settings.gameId === "tictactoe" || room.settings.gameId === "gomoku" ||
    (room.settings.gameId === "othello" && room.settings.enableRanked)
  );

  let myPetIds = $state<Set<string>>(new Set());

  $effect(() => {
    if (!forceGiveawayEligibleGame) return;
    let alive = true;
    const applyPets = (payload: PetBondState) => {
      if (!alive || !payload || typeof payload !== "object") return;
      myPetIds = new Set((payload.pets || []).map((p) => p.playerId));
    };
    // 订阅 petbond 频道：只给打开强制白给相关 UI 的连接推送，避免全员广播。
    const subscribe = () => {
      ask<PetBondState>("petbond:subscribe", {}).then(applyPets).catch(() => {});
    };
    subscribe();
    socket.on("petbond:update", applyPets);
    // WS 断线重连会换一个全新连接，服务端的频道订阅关系不会跨连接保留，必须重新 subscribe，
    // 否则重连后收不到任何 petbond:update，本地数据永久停在断线前那一刻。
    socket.on("connect", subscribe);
    return () => {
      alive = false;
      socket.off("petbond:update", applyPets);
      socket.off("connect", subscribe);
      ask("petbond:unsubscribe", {}).catch(() => undefined);
    };
  });

  const canForceGiveaway = $derived(Boolean(
    forceGiveawayEligibleGame && mySeat && opponentPlayer && myPetIds.has(opponentPlayer.id) && opponentPlayer.giveawayEnabled
  ));

  async function forceGiveaway() {
    if (!opponentPlayer) return;
    if (!window.confirm(`确定强制 ${opponentPlayer.name} 白给吗？`)) return;
    await act("petbond:forceGiveaway", { targetId: opponentPlayer.id });
  }

  async function act(event: string, payload: unknown = {}) {
    try {
      await ask(event, payload);
    } catch (error) {
      onError(error instanceof Error ? error.message : "操作失败");
    }
  }

  // 聊天发言人查找源：房间名单优先（在房间内持续更新），叠加大厅名单兜底
  // （进房后大厅频道会取消订阅，可能不是最新的，但依旧好过完全查不到）。
  const chatPlayers = $derived.by(() => {
    const map = new Map<string, PublicPlayer>();
    for (const player of sessionStore.lobby?.players || []) map.set(player.id, player);
    for (const item of roomPlayerList(room)) map.set(item.player.id, item.player);
    return Array.from(map.values());
  });

  const leaveTitle = $derived.by(() => {
    if (room.phase === "punishment") {
      return punishedIds.includes(me.id) ? "惩罚完成前不能离开房间" : "离开后，服务器会自动处理你负责的审核或任务";
    }
    if (room.settings.gameId === "tictactoe" && room.phase === "choosing" && mySeat) return "井字棋对局进行中不能离开战斗席";
    if (room.settings.gameId === "gomoku" && room.phase === "choosing" && mySeat) return "五子棋对局进行中不能离开战斗席";
    if (room.settings.gameId === "jungle" && room.phase === "choosing" && mySeat) return "斗兽棋对局进行中不能离开战斗席";
    if (room.settings.gameId === "chess" && room.phase === "choosing" && mySeat) return "国际象棋对局进行中不能离开战斗席";
    return "离开房间";
  });
  const punishedIds = $derived(room.punishedPlayerIds || []);

  async function leaveCurrentRoom() {
    if (room.phase === "punishment" && punishedIds.includes(me.id)) {
      onError("惩罚完成前不能离开房间");
      return;
    }
    try {
      await ask("room:leave", {});
      // 与 room:closed / player:kicked 的处理保持一致：本地房态必须清掉，否则退房后
      // 若有一条乱序 room:update 命中同一房间 ID，或后台「返回」按钮凭 Boolean(room)
      // 判断，都会把人拽回一个已经离开的过期房间 UI。
      sessionStore.room = null;
      routerStore.goto("lobby");
    } catch (error) {
      onError(error instanceof Error ? error.message : "离开房间失败");
    }
  }

  function sit(seat: "A" | "B") {
    const warn = seriesFactionWarning(room, config, me);
    if (warn && !window.confirm(warn)) return;
    void act("room:sit", { seat });
  }

  let previewImage = $state<string | null>(null);

  // 座位制游戏各自的 RPC 转发（仅黑白棋/井字棋走这一层薄封装；其余 5 款游戏面板内部
  // 直接调用 ask()，是原 React 版就有的既存不对称写法，迁移时原样保留）。
  const playOthello = (row: number, col: number) => act("othello:move", { row, col });
  const playTicTacToe = (row: number, col: number) => act("tictactoe:move", { row, col });
  const chooseTicTacToeGiveaway = (mode: "normal" | "giveaway") => act("tictactoe:giveawayChoice", { mode });
  const settleOthelloMove = (mode: "normal" | "giveaway" | "tribute") => act("othello:settleMove", { mode });
  const restartOthello = () => act("othello:restart", {});
  const readyOthello = () => act("othello:ready", {});
  const readyTicTacToe = () => act("tictactoe:ready", {});
  const restartTicTacToe = () => act("tictactoe:restart", {});
  const requestOthelloSurrender = () => act("othello:requestSurrender", {});
  const requestOthelloUndo = () => act("othello:undoRequest", {});
  const respondOthelloUndo = (accept: boolean) => act("othello:undoRespond", { accept });
  const respondOthelloSurrender = (accept: boolean) => act("othello:respondSurrender", { accept });
  async function escapeOthello() {
    if (!window.confirm("确定要逃跑吗？本局会立即判负，并按剩余空格追加扣分。")) return;
    await act("othello:escape", {});
  }
</script>

<section class="room-layout">
  <div
    class={`panel room-header ${room.settings.roomBackgroundImage ? "has-room-header-background" : ""}`}
    style={room.settings.roomBackgroundImage ? styleString({ "--room-header-bg": `url(${room.settings.roomBackgroundImage})` }) : undefined}
  >
    <div>
      <h2><Swords size={20} /> {room.settings.name} <ExtremeRankedBadge enabled={room.settings.enableExtremeRanked} /> <RankMultiplierBadge multiplier={rankMultiplierForSettings(room.settings)} /></h2>
      {#if room.settings.enableTags && room.settings.tags?.length}<RoomTagList tags={room.settings.tags} />{/if}
      <RoomInfoTagList tags={roomInfoTags(config, room)} />
    </div>
    <button class="soft-button" title={leaveTitle} onclick={leaveCurrentRoom}><DoorOpen size={16} /> 离开</button>
  </div>

  {#if room.settings.gameId !== "liarsdice" && room.settings.gameId !== "coinflip"}
    <div class="battle-panel">
      <SeatView seat="A" {room} {me} onSit={() => sit("A")} />
      <div class="versus">
        <span class="versus-label">⚔️ 对战比分</span>
        <strong class="score-number">{room.score.A} : {room.score.B}</strong>
        {#if room.settings.gameId === "othello"}<OthelloScore {room} />
        {:else if room.settings.gameId === "tictactoe"}<TicTacToeScore {room} />
        {:else if room.settings.gameId === "gomoku"}<GomokuScore {room} />
        {:else if room.settings.gameId === "jungle"}<JungleScore {room} />
        {:else if room.settings.gameId === "chess"}<ChessScore {room} />
        {:else}<Settlement {room} />{/if}
      </div>
      <SeatView seat="B" {room} {me} onSit={() => sit("B")} />
    </div>
  {/if}

  <div class="room-content-grid">
    <div class="actions-panel panel">
      {#if room.settings.gameId === "othello"}
        <OthelloPanel {room} {me} onMove={playOthello} onSettle={settleOthelloMove} onRestart={restartOthello} onReady={readyOthello} onRequestUndo={requestOthelloUndo} onRespondUndo={respondOthelloUndo} onRequestSurrender={requestOthelloSurrender} onRespondSurrender={respondOthelloSurrender} onEscape={escapeOthello} />
      {:else if room.settings.gameId === "tictactoe"}
        <TicTacToePanel {room} {me} onMove={playTicTacToe} onReady={readyTicTacToe} onRestart={restartTicTacToe} onGiveawayChoice={chooseTicTacToeGiveaway} />
      {:else if room.settings.gameId === "liarsdice"}
        <LiarsDicePanel {room} {me} {onError} />
      {:else if room.settings.gameId === "gomoku"}
        <GomokuPanel {room} {me} {onError} />
      {:else if room.settings.gameId === "jungle"}
        <JunglePanel {room} {me} {onError} />
      {:else if room.settings.gameId === "chess"}
        <ChessPanel {room} {me} {onError} />
      {:else if room.settings.gameId === "coinflip"}
        <CoinFlipPanel {room} {me} {onError} />
      {:else}
        <RpsPanel {room} {me} {mySeat} {onError} />
      {/if}

      {#if canGoSpectate || canForceGiveaway}
        <div class="seat-action-row">
          {#if canGoSpectate}<button onclick={() => act("room:spectate")}><Eye size={16} /> 去观战席</button>{/if}
          {#if canForceGiveaway}<button class="force-giveaway-button" onclick={forceGiveaway}><HeartHandshake size={16} /> 强制白给</button>{/if}
        </div>
      {/if}

      {#if room.phase === "punishment"}
        <PunishmentPhase {room} {me} onOpenImage={(url) => (previewImage = url)} />
      {/if}
    </div>

    <div class="room-side-stack">
      <RoomPlayerListPanel {room} />
      <RoomChatSection {room} {chatPlayers} />
    </div>

    <RoundHistoryPanel {room} onOpenImage={(url) => (previewImage = url)} />
  </div>

  {#if previewImage}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="modal-backdrop image-preview-backdrop" onclick={() => (previewImage = null)}>
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div class="image-preview-modal" onclick={(event) => event.stopPropagation()}>
        <button class="icon-button image-preview-close" onclick={() => (previewImage = null)}>×</button>
        <img src={previewImage} alt="惩罚证明大图" loading="lazy" />
      </div>
    </div>
  {/if}
</section>
