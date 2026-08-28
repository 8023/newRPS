<script lang="ts">
  // 房间聊天 / 大厅聊天 tab 切换。源：ui/AppViews.tsx 2147-2181（Room 组件内联区块）。
  import type { PublicPlayer, RoomSnapshot } from "../../shared/types";
  import ChatBoardShell from "../chat/ChatBoardShell.svelte";
  import ChatPanel from "../chat/ChatPanel.svelte";

  let { room, chatPlayers }: { room: RoomSnapshot; chatPlayers: PublicPlayer[] } = $props();

  let chatTab = $state<"room" | "lobby">("room");
</script>

{#snippet tabs()}
  <div class="segmented chat-tabs">
    <button class={chatTab === "room" ? "active" : ""} onclick={() => (chatTab = "room")}>本房间</button>
    <button class={chatTab === "lobby" ? "active" : ""} onclick={() => (chatTab = "lobby")}>大厅</button>
  </div>
{/snippet}

<ChatBoardShell title={chatTab === "room" ? "房间聊天" : "大厅聊天"} collapseKey="roomChat" {tabs}>
  {#if chatTab === "room"}
    {#key `room-${room.id}`}
      <ChatPanel scope={room.id} players={chatPlayers} placeholder="点击头像可 @ 人" emptyText="还没有房间聊天" messagesClass="room-chat-messages" />
    {/key}
  {:else}
    {#key "lobby-in-room"}
      <ChatPanel scope="" players={chatPlayers} placeholder="点击头像可 @ 人" emptyText="大厅还没有留言" subscribeLobbyChannel messagesClass="room-chat-messages" />
    {/key}
  {/if}
</ChatBoardShell>
