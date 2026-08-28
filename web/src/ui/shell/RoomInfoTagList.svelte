<script lang="ts">
  // 源：ui/AppViews.tsx:3481-3487。大厅列表与房间头部共用。
  import type { RoomInfoTagView } from "../../lib/roomInfoTags";
  import { roomInfoTagStyle } from "../../lib/roomInfoTags";
  import { styleString } from "../../lib/style";

  let { tags }: { tags: RoomInfoTagView[] } = $props();
</script>

<div class="room-info-tags">
  {#each tags as tag (tag.key)}
    <span class="room-info-tag" style={styleString(roomInfoTagStyle(tag.style))}>{tag.text}</span>
  {/each}
</div>
