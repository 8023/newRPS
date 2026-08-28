<script lang="ts">
  // TODO(svelte-migration Phase 7)：完整迁移 ui/AppViews.tsx:4695-5241 的 ProfilePanel。当前仅占位。
  import type { AppConfig, PublicPlayer } from "../../shared/types";

  let { me, onClose }: {
    config: AppConfig;
    me: PublicPlayer;
    theme: "light" | "dark";
    onThemeChange: (theme: "light" | "dark") => void;
    onClose: () => void;
    onUpdated: (player: PublicPlayer) => void;
    onError: (message: string) => void;
    onLoggedOut: () => void;
  } = $props();
</script>

<div class="modal-backdrop" onclick={(event) => { if (event.target === event.currentTarget) onClose(); }}>
  <section class="panel">
    <p>个人设置：{me.name}（Svelte 迁移中）</p>
    <button type="button" onclick={onClose}>关闭</button>
  </section>
</div>
