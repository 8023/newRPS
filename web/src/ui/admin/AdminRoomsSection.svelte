<script lang="ts">
  // 「用户与房间」分区：三个子页签——用户管理（查询/编辑/踢出/找回密钥）、房间管理
  // （查看状态、强制结算/重开/关闭）、聊天管理（转给 AdminChatManager）。
  // 源：ui/AdminViews.tsx:1204-1341（renderUserManagement + activeSection === "rooms"）。
  import { untrack } from "svelte";
  import { lobbyRoomInfoTags, roomStatusText } from "../../lib/roomInfoTags";
  import { ADMIN_PLAYER_FILTER_BUTTONS } from "../../lib/adminHelpers";
  import { adminStore, type AdminRoomTab as AdminRoomTabType } from "../../lib/stores/adminStore.svelte";
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import AdminSectionHeader from "./AdminSectionHeader.svelte";
  import AdminPlayerEditor from "./AdminPlayerEditor.svelte";
  import AdminChatManager from "./AdminChatManager.svelte";
  import RoomTagList from "../shell/RoomTagList.svelte";
  import RoomInfoTagList from "../shell/RoomInfoTagList.svelte";

  // 与原版一致：这里用服务端权威 config（非管理员本地未保存草稿 draft）——房间列表标签
  // 渲染、玩家编辑器里的阵营/性别下拉都应该反映"当前实际生效"的配置，而不是管理员正在
  // 编辑但还没点保存的草稿。
  const config = $derived(sessionStore.config!);
  const lobby = $derived(sessionStore.lobby!);

  type ManagementTab = { id: AdminRoomTabType; label: string; detail: string; meta: string };
  const managementTabs = $derived<ManagementTab[]>([
    { id: "users", label: "用户管理", detail: "查询资料、踢出用户与找回密钥", meta: `${lobby.onlineCount} 人在线` },
    { id: "rooms", label: "房间管理", detail: "查看房间状态并处理当前对局", meta: `${lobby.rooms.length} 个房间` },
    { id: "announcement", label: "聊天管理", detail: "检索、删除或恢复历史消息", meta: "历史消息" }
  ]);
  const currentManagementTab = $derived(managementTabs.find((tab) => tab.id === adminStore.activeRoomTab) || managementTabs[0]);

  // 仅在切到「用户与房间」的用户管理页时拉取；过滤器切换由 togglePlayerFilter 自己负责。
  // ⚠ loadAdminPlayers() 的默认参数会同步读 playerFilters，不 untrack 的话过滤器就成了
  // 本 effect 的依赖，点一次过滤器会连发两次 admin:listPlayers（原 React 版 deps 只有
  // [logged, activeSection, activeRoomTab]，刻意不含 playerFilters）。
  $effect(() => {
    void adminStore.logged;
    void adminStore.activeSection;
    void adminStore.activeRoomTab;
    untrack(() => adminStore.loadAdminPlayers());
  });

  const nameKeyword = $derived(adminStore.playerNameSearch.trim().toLowerCase());
  const visiblePlayers = $derived(
    nameKeyword ? adminStore.adminPlayers.filter((player) => (player.name || "").toLowerCase().includes(nameKeyword)) : adminStore.adminPlayers
  );
  const listHint = $derived(`匹配过滤器 · 在线 ${adminStore.adminFilterOnlineCount} / 离线 ${adminStore.adminFilterOfflineCount}`);
</script>

<nav class="admin-management-nav" aria-label="用户与房间管理分类">
  {#each managementTabs as tab (tab.id)}
    <button
      type="button"
      class={adminStore.activeRoomTab === tab.id ? "active" : ""}
      onclick={() => (adminStore.activeRoomTab = tab.id)}
      aria-current={adminStore.activeRoomTab === tab.id ? "page" : undefined}
    >
      <span>
        <strong>{tab.label}</strong>
        <small>{tab.detail}</small>
      </span>
      <em>{tab.meta}</em>
    </button>
  {/each}
</nav>

<div class="config-section admin-section-card admin-management-content">
  <AdminSectionHeader title={currentManagementTab.label} subtitle={currentManagementTab.detail} />

  {#if adminStore.activeRoomTab === "announcement"}
    <AdminChatManager />
  {/if}

  {#if adminStore.activeRoomTab === "users"}
    <div class="admin-tab-content">
      <div class="admin-player-filters" role="group" aria-label="用户列表过滤器">
        <input
          class="admin-player-name-search"
          value={adminStore.playerNameSearch}
          oninput={(event) => (adminStore.playerNameSearch = event.currentTarget.value)}
          placeholder="搜索昵称，留空则不筛选"
          aria-label="按昵称搜索"
          autocomplete="off"
        />
        {#each ADMIN_PLAYER_FILTER_BUTTONS as item (item.key)}
          {@const active = adminStore.playerFilters[item.key]}
          <button type="button" class={`admin-filter-btn${active ? " active" : ""}`} aria-pressed={active} onclick={() => adminStore.togglePlayerFilter(item.key)}>
            {item.label}
          </button>
        {/each}
      </div>
      <div class="admin-list-section admin-player-list">
        <div class="admin-list-heading">
          <h3>玩家列表</h3>
          <span>
            {#if adminStore.adminPlayersLoading}
              加载中…
            {:else if nameKeyword}
              昵称匹配 {visiblePlayers.length} / 已加载 {adminStore.adminPlayers.length} 人 · {listHint}
            {:else if adminStore.adminPlayersTruncated}
              显示 {adminStore.adminPlayers.length} / 共 {adminStore.adminPlayersTotal} 人（已截断）· {listHint}
            {:else}
              {adminStore.adminPlayersTotal} 人 · {listHint}
            {/if}
          </span>
        </div>
        {#each visiblePlayers as player (player.id)}
          <AdminPlayerEditor
            {config}
            {player}
            onSave={async (payload) => {
              if (await adminStore.action("editPlayer", payload)) await adminStore.loadAdminPlayers();
            }}
            onKick={async () => {
              if (await adminStore.action("kick", { playerId: player.id })) await adminStore.loadAdminPlayers();
            }}
            onError={(m) => uiStore.notify(m)}
          />
        {/each}
        {#if !adminStore.adminPlayersLoading && visiblePlayers.length === 0}
          <p class="empty">{nameKeyword ? "没有昵称匹配的玩家" : "当前没有符合条件的玩家"}</p>
        {/if}
      </div>
    </div>
  {/if}

  {#if adminStore.activeRoomTab === "rooms"}
    <div class="admin-list-section">
      <div class="admin-list-heading">
        <h3>房间列表</h3>
        <span>{lobby.rooms.length} 间</span>
      </div>
      {#each lobby.rooms as room (room.id)}
        {@const canForceSeatOutcome = room.status === "playing" && room.gameId !== "liarsdice" && room.versus.A != null && room.versus.B != null}
        {@const seatALabel = room.versus.A?.player?.name || "A方"}
        {@const seatBLabel = room.versus.B?.player?.name || "B方"}
        <div class="admin-room">
          <div class="admin-card-title">
            <strong>{room.name}</strong>
            <small>{room.id} · {roomStatusText(room.status)} · {room.gameId === "liarsdice" ? `${room.players} 人参战` : room.gameId === "coinflip" ? `${room.players}/1 参战席` : `${room.players}/2 战斗席`} · {room.spectators} 观战</small>
          </div>
          {#if room.tags?.length}<RoomTagList tags={room.tags} />{/if}
          <RoomInfoTagList tags={lobbyRoomInfoTags(config, room)} />
          <div class="admin-action-row othello-admin-actions">
            <button class="danger-button" onclick={() => adminStore.action("closeRoom", { roomId: room.id })}>关闭房间</button>
            <button onclick={() => adminStore.action("forceNext", { roomId: room.id })}>重开</button>
            <button onclick={() => adminStore.action("forceSeatOutcome", { roomId: room.id, result: "A" })} disabled={!canForceSeatOutcome}>判 {seatALabel} 胜</button>
            <button onclick={() => adminStore.action("forceSeatOutcome", { roomId: room.id, result: "B" })} disabled={!canForceSeatOutcome}>判 {seatBLabel} 胜</button>
            <button onclick={() => adminStore.action("forceSeatOutcome", { roomId: room.id, result: "draw" })} disabled={!canForceSeatOutcome}>判平</button>
          </div>
        </div>
      {/each}
      {#if lobby.rooms.length === 0}<p class="empty">暂无房间</p>{/if}
    </div>
  {/if}
</div>
