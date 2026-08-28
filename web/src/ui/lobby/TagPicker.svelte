<script lang="ts">
  // 源：ui/AppViews.tsx:1534-1550
  let { options, value, onChange }: { options: string[]; value: string[]; onChange: (tags: string[]) => void } = $props();

  function toggle(tag: string) {
    if (value.includes(tag)) onChange(value.filter((item) => item !== tag));
    else onChange([...value, tag].slice(0, 5));
  }
</script>

<div class="room-tag-picker">
  {#each options as tag (tag)}
    <button type="button" class={value.includes(tag) ? "active" : ""} onclick={() => toggle(tag)}>#{tag}</button>
  {/each}
  {#if options.length === 0}
    <p class="empty">后台还没有配置房间 Tag</p>
  {/if}
</div>
