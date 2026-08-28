<script lang="ts">
  // TODO(svelte-migration Phase 4)：完整迁移 ui/AppViews.tsx:508-612 的 Lobby 组件
  // （房间列表、创建房间、认宠面板、抢名/白给面板、大厅聊天）。当前仅占位，用于打通
  // Phase 3（外壳 + 登录）的可视化验收链路。改为直接读 sessionStore/uiStore，不再需要
  // config/lobby/me/onGoRoom 等 props——真正实现时房间列表/建房会各自成组件，都直接
  // 读 store，不必再经手 Lobby 一层层转发。
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { routerStore } from "../../lib/stores/routerStore.svelte";

  const lobby = $derived(sessionStore.lobby!);
  const me = $derived(sessionStore.me!.player);
</script>

<section class="dashboard">
  <div class="panel lobby-main">
    <div class="panel-title lobby-title">
      <h2>大厅（Svelte 迁移中）</h2>
    </div>
    <p class="hint">已登录：{me.name}。房间列表 {lobby.rooms.length} 个，在线 {lobby.players.length} 人。完整大厅 UI 正在迁移中。</p>
    <button type="button" onclick={() => routerStore.goto("contribute")}>参与共建</button>
  </div>
</section>
