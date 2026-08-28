<script lang="ts">
  // TODO(svelte-migration Phase 8)：完整迁移 ui/ContributeView.tsx 共建投稿系统。当前仅占位。
  import type { AppConfig, PublicPlayer } from "../../shared/types";

  let { onBack }: {
    config: AppConfig;
    me: PublicPlayer;
    onBack: () => void;
    onError: (message: string) => void;
    ensureSession: () => Promise<boolean>;
  } = $props();
</script>

<section class="panel">
  <p>参与共建（Svelte 迁移中）</p>
  <button type="button" onclick={onBack}>返回</button>
</section>
