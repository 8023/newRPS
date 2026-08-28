<script lang="ts">
  // TODO(svelte-migration Phase 5-6)：完整迁移 ui/AppViews.tsx:1551-2219 的 Room 组件
  // 及其挂靠的 8 款游戏面板。当前仅占位。
  import type { AppConfig, LobbySnapshot, PublicPlayer, RoomSnapshot } from "../../shared/types";

  let {
    room, onBack
  }: {
    config: AppConfig;
    room: RoomSnapshot;
    me: PublicPlayer;
    lobby: LobbySnapshot | null;
    onBack: () => void;
    onError: (message: string) => void;
  } = $props();
</script>

<section class="panel">
  <p>房间「{room.settings.name}」（Svelte 迁移中）</p>
  <button type="button" onclick={onBack}>返回大厅</button>
</section>
