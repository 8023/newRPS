<script lang="ts">
  // 源：ui/AppViews.tsx:2134-2146
  import type { RoomSnapshot } from "../../shared/types";
  import { roomPlayerList } from "../../lib/gameDisplay";
  import { useMobileCollapse } from "../../lib/useMobileCollapse.svelte";
  import CollapseToggle from "../shell/CollapseToggle.svelte";
  import RoomPlayerRow from "../shell/RoomPlayerRow.svelte";

  let { room }: { room: RoomSnapshot } = $props();
  const collapse = useMobileCollapse("roomPlayers");
  const roomPlayers = $derived(roomPlayerList(room));
</script>

<div class={`panel room-player-panel ${collapse.collapsed ? "collapsed" : ""}`}>
  <h3 class="sticky-panel-title">
    房间玩家名单
    <span class="panel-title-actions">
      <span>{roomPlayers.length} 人</span>
      <CollapseToggle collapsed={collapse.collapsed} onToggle={collapse.toggle} label="房间玩家名单" />
    </span>
  </h3>
  <div class={`room-player-list mobile-collapsible-body ${collapse.collapsed ? "collapsed" : ""}`}>
    {#each roomPlayers as item (`${item.role}-${item.player.id}`)}
      <RoomPlayerRow player={item.player} role={item.role} />
    {/each}
    {#if roomPlayers.length === 0}<p class="empty">暂无真人玩家</p>{/if}
  </div>
</div>
