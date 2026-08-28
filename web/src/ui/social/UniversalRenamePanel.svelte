<script lang="ts">
  // TODO(svelte-migration Phase 7)：完整迁移 ui/AppViews.tsx:4066-4151 的
  // UniversalRenamePanel（抢名大战改名面板）。
  import type { PublicPlayer } from "../../shared/types";

  let { targets }: { targets: PublicPlayer[] } = $props();
</script>

{#if targets.length > 0}
  <div class="panel">
    <p class="hint">抢名大战：{targets.length} 名玩家可被改名（Svelte 迁移中）</p>
  </div>
{/if}
