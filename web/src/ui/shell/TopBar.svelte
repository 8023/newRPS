<script lang="ts">
  // 顶栏：站点名、连接状态、房间内的阶段摘要、右上角操作区。完全由 store 驱动，不接收
  // 任何 props——App.svelte 不需要替它转发 config/lobby/room/me。
  // 源：App.tsx 645-559 的 <header className="topbar">。
  import Crown from "@lucide/svelte/icons/crown";
  import Info from "@lucide/svelte/icons/info";
  import UserRound from "@lucide/svelte/icons/user-round";
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { routerStore } from "../../lib/stores/routerStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import { connectionStateText, phaseText } from "../../lib/playerDisplay";
  import PlayerBadge from "./PlayerBadge.svelte";

  const config = $derived(sessionStore.config!);
</script>

<header class="topbar">
  <div>
    <h1>{config.site.name}</h1>
    {#if routerStore.view === "room"}
      <span class={`connection-pill ${sessionStore.connectionState}`}>
        {sessionStore.connectionState === "connected" ? `在线 ${sessionStore.onlineCount()} 人` : connectionStateText(sessionStore.connectionState)}
      </span>
      {#if sessionStore.room}
        <span class="top-summary">⚔️ {phaseText(sessionStore.room.phase, sessionStore.room.settings.gameId)}</span>
      {/if}
    {:else}
      <span class={`connection-pill ${sessionStore.connectionState}`}>
        {sessionStore.connectionState === "connected" ? `在线 ${sessionStore.onlineCount()} 人` : connectionStateText(sessionStore.connectionState)}
      </span>
    {/if}
  </div>
  <div class="top-actions">
    {#if sessionStore.me}<PlayerBadge player={sessionStore.me.player} compact />{/if}
    <button class="soft-button top-sponsor-button" title="关于" onclick={() => (uiStore.aboutOpen = true)}>
      <Info size={18} /> <span>关于</span>
    </button>
    {#if sessionStore.me}
      <button class="soft-button top-profile-button" title="个人设置" onclick={() => (uiStore.profileOpen = true)}>
        <UserRound size={18} /> <span>个人设置</span>
      </button>
    {/if}
    {#if sessionStore.me && sessionStore.lobby}
      <button class="soft-button top-leaderboard-button" title="排行榜" onclick={() => (uiStore.leaderboardOpen = true)}>
        <Crown size={18} /> <span>排行榜</span>
      </button>
    {/if}
  </div>
</header>
