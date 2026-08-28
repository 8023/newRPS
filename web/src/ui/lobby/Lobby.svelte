<script lang="ts">
  // 源：ui/AppViews.tsx:508-612。房间列表/建房/认宠/抢名/白给/大厅聊天的组合页——原
  // React 版这些全挤在一个函数组件里，现在拆成 RoomCard/CreateRoom/PetBondPanel/
  // UniversalRenamePanel/GiveawayPanel/Leaderboard/ChatBoardShell 各自独立的组件，
  // Lobby 本身只负责布局 + 拼装。config/lobby/me 全部直接读 sessionStore，不再需要
  // 从 App 一层层转发。
  import Users from "@lucide/svelte/icons/users";
  import { ask } from "../../lib/rpc";
  import { normalizeRoomSnapshot } from "../../lib/normalize";
  import { isRenameTarget, isRenameTargetVisible } from "../../lib/playerDisplay";
  import { seriesFactionWarningFor } from "../seriesFaction";
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import { routerStore } from "../../lib/stores/routerStore.svelte";
  import Leaderboard from "../shell/Leaderboard.svelte";
  import ChatBoardShell from "../chat/ChatBoardShell.svelte";
  import ChatPanel from "../chat/ChatPanel.svelte";
  import RoomCard from "./RoomCard.svelte";
  import CreateRoom from "./CreateRoom.svelte";
  import PetBondPanel from "../social/PetBondPanel.svelte";
  import UniversalRenamePanel from "../social/UniversalRenamePanel.svelte";
  import GiveawayPanel from "../social/GiveawayPanel.svelte";

  const config = $derived(sessionStore.config!);
  const lobby = $derived(sessionStore.lobby!);
  const me = $derived(sessionStore.me!.player);

  let showCreate = $state(false);
  let passwords = $state<Record<string, string>>({});
  let now = $state(Date.now());
  const renameTargets = $derived(lobby.players.filter((player) => isRenameTargetVisible(player, now)));

  $effect(() => {
    if (!lobby.players.some((player) => !player.connected && isRenameTarget(player) && player.disconnectedAt)) return;
    const timer = window.setInterval(() => { now = Date.now(); }, 60_000);
    return () => window.clearInterval(timer);
  });

  async function joinRoom(roomId: string) {
    try {
      const targetRoom = lobby.rooms.find((room) => room.id === roomId);
      if (targetRoom) {
        const warn = seriesFactionWarningFor(targetRoom.punishmentSource, targetRoom.punishmentSeriesId, config, me);
        if (warn && !window.confirm(warn)) return;
      }
      if (targetRoom?.enableRanked && Boolean(targetRoom.enableExtremeRanked) !== Boolean(me.extremeModeEnabled)) {
        const ok = window.confirm(me.extremeModeEnabled
          ? "你是极限模式玩家，进入普通排位房后只能在观战席，不能上桌。确认进入？"
          : "你不是极限模式玩家，进入极限排位房后只能在观战席，不能上桌。确认进入？");
        if (!ok) return;
      }
      if (targetRoom?.enableExtremeRanked && me.extremeModeEnabled) {
        const ok = window.confirm(`这是极限排位房，胜负会按极限模式规则结算，并存在连胜风险，确认进入？`);
        if (!ok) return;
      }
      if (targetRoom?.enableRanked && (targetRoom.rankMultiplier || 1) > 1) {
        const multiplier = targetRoom.rankMultiplier || 1;
        const effectiveStake = targetRoom.stake * multiplier;
        const ok = window.confirm(targetRoom.gameId === "othello"
          ? `这是 ${multiplier} 倍黑白棋排位房，每翻 1 子按 ${effectiveStake} 分实时结算，确认进入？`
          : `这是 ${multiplier} 倍排位房，本局胜负按 ${effectiveStake} 分结算，确认进入？`);
        if (!ok) return;
      }
      const result = await ask<{ room: import("../../shared/types").RoomSnapshot }>("room:join", { roomId, password: passwords[roomId] });
      sessionStore.room = normalizeRoomSnapshot(result.room);
      routerStore.goto("room");
    } catch (error) {
      uiStore.notify(error instanceof Error ? error.message : "加入失败");
    }
  }
</script>

<section class="dashboard">
  <div class="panel lobby-main">
    <div class="panel-title lobby-title">
      <h2><Users size={20} /> 大厅</h2>
      <div class="lobby-title-actions">
        <button type="button" class="small" onclick={() => routerStore.goto("contribute")}>参与共建</button>
        <button type="button" class="primary small" onclick={() => (showCreate = !showCreate)}>创建房间</button>
      </div>
    </div>
    <div class="room-list">
      {#each lobby.rooms as room (room.id)}
        <RoomCard
          {config} {room}
          password={passwords[room.id] || ""}
          onPasswordChange={(value) => (passwords = { ...passwords, [room.id]: value })}
          onJoin={() => joinRoom(room.id)}
        />
      {/each}
      {#if lobby.rooms.length === 0}<p class="empty">还没有房间，先创建一个吧。</p>{/if}
    </div>
    <div class="lobby-lower-grid">
      <div class="lobby-lower-col lobby-lower-col-left">
        <PetBondPanel />
      </div>
      <div class="lobby-lower-col lobby-lower-col-right">
        <UniversalRenamePanel targets={renameTargets} />
        <GiveawayPanel />
      </div>
    </div>
  </div>
  <aside class="side-column">
    <Leaderboard title="在线积分榜" players={lobby.rankedLeaderboard} />
    <ChatBoardShell title="大厅聊天" collapseKey="lobbyChat">
      <ChatPanel scope="" players={lobby.players} placeholder="点击头像可 @ 人" emptyText="还没有留言" messagesClass="lobby-suggestion-messages" />
    </ChatBoardShell>
  </aside>
  {#if showCreate}
    <CreateRoom onCreated={(nextRoom) => { if (nextRoom) sessionStore.room = nextRoom; routerStore.goto("room"); showCreate = false; }} onCancel={() => (showCreate = false)} />
  {/if}
</section>
