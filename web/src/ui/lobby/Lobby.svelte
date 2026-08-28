<script lang="ts">
  // TODO(svelte-migration Phase 4)：完整迁移 ui/AppViews.tsx:508-612 的 Lobby 组件
  // （房间列表、创建房间、认宠面板、抢名/白给面板、大厅聊天）。当前仅占位，用于打通
  // Phase 3（外壳 + 登录）的可视化验收链路。
  import type { AppConfig, LobbySnapshot, PublicPlayer, RoomSnapshot } from "../../shared/types";

  let {
    config, lobby, me
  }: {
    config: AppConfig;
    lobby: LobbySnapshot;
    me: PublicPlayer;
    punishmentTagPrefs?: Record<string, string>;
    onError: (message: string) => void;
    onGoRoom: (room?: RoomSnapshot) => void;
    onContribute?: () => void;
    onPunishmentTagPrefsChange?: (prefs: Record<string, string>) => void;
  } = $props();
</script>

<section class="dashboard">
  <div class="panel lobby-main">
    <div class="panel-title lobby-title">
      <h2>大厅（Svelte 迁移中）</h2>
    </div>
    <p class="hint">已登录：{me.name}。房间列表 {lobby.rooms.length} 个，在线 {lobby.players.length} 人。完整大厅 UI 正在迁移中。</p>
  </div>
</section>
