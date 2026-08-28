<script lang="ts">
  // TODO(svelte-migration Phase 5-6)：完整迁移 ui/AppViews.tsx:1551-2219 的 Room 组件
  // 及其挂靠的 8 款游戏面板。当前仅占位，直接读 sessionStore.room，不接收 props。
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { routerStore } from "../../lib/stores/routerStore.svelte";

  const room = $derived(sessionStore.room!);
</script>

<section class="panel">
  <p>房间「{room.settings.name}」（Svelte 迁移中）</p>
  <button type="button" onclick={() => routerStore.goto("lobby")}>返回大厅</button>
</section>
