<script lang="ts">
  // TODO(svelte-migration Phase 9)：完整迁移 ui/AdminViews.tsx（11 个后台分区）。当前仅
  // 占位，直接读 sessionStore、通过 routerStore.leaveAdmin 返回，不接收 props。
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { routerStore } from "../../lib/stores/routerStore.svelte";

  function back() {
    routerStore.leaveAdmin(!!sessionStore.me, !!sessionStore.room);
  }
</script>

<section class="panel">
  <p>后台管理（Svelte 迁移中）</p>
  <button type="button" onclick={back}>返回</button>
</section>
