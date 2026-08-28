<script lang="ts">
  // 源：ui/AppViews.tsx:1488-1495。房主自定义的房间 Tag（区别于 RoomInfoTagList 的系统标签）。
  let { tags }: { tags: string[] } = $props();
</script>

<div class="room-tag-list">
  {#each tags as tag (tag)}
    <span class="room-tag">#{tag}</span>
  {/each}
</div>
