<script lang="ts">
  // TODO(svelte-migration Phase 9)：完整迁移 ui/AdminViews.tsx（11 个后台分区）。当前仅占位。
  import type { AppConfig, LobbySnapshot } from "../../shared/types";

  let { onBack }: { config: AppConfig; lobby: LobbySnapshot; onBack: () => void; onError: (message: string) => void } = $props();
</script>

<section class="panel">
  <p>后台管理（Svelte 迁移中）</p>
  <button type="button" onclick={onBack}>返回</button>
</section>
