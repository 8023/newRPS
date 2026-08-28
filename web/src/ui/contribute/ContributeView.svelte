<script lang="ts">
  // TODO(svelte-migration Phase 8)：完整迁移 ui/ContributeView.tsx 共建投稿系统。当前仅
  // 占位，直接读 sessionStore.config/me，不接收 props。
  import { routerStore } from "../../lib/stores/routerStore.svelte";

  function back() {
    routerStore.keepContribute = false;
    routerStore.goto("lobby");
  }
</script>

<section class="panel">
  <p>参与共建（Svelte 迁移中）</p>
  <button type="button" onclick={back}>返回</button>
</section>
