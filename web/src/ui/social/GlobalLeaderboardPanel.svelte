<script lang="ts">
  // TODO(svelte-migration Phase 7)：完整迁移 ui/AppViews.tsx:3740-3968 的
  // GlobalLeaderboardPanel。当前仅占位，直接读 sessionStore.leaderboardSource，不接收 props。
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";

  function close() {
    uiStore.leaderboardOpen = false;
  }
</script>

<div class="modal-backdrop" onclick={(event) => { if (event.target === event.currentTarget) close(); }}>
  <section class="panel">
    <p>排行榜（Svelte 迁移中），共 {sessionStore.leaderboardSource.length} 名玩家。</p>
    <button type="button" onclick={close}>关闭</button>
  </section>
</div>
