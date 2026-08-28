<script lang="ts">
  // TODO(svelte-migration Phase 7)：完整迁移 ui/AppViews.tsx:3740-3968 的 GlobalLeaderboardPanel。当前仅占位。
  import type { PublicPlayer } from "../../shared/types";

  let { players, onClose }: { players: PublicPlayer[]; onClose: () => void } = $props();
</script>

<div class="modal-backdrop" onclick={(event) => { if (event.target === event.currentTarget) onClose(); }}>
  <section class="panel">
    <p>排行榜（Svelte 迁移中），共 {players.length} 名玩家。</p>
    <button type="button" onclick={onClose}>关闭</button>
  </section>
</div>
