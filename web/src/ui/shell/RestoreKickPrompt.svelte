<script lang="ts">
  // 自动登录（刷新页面/断线重连）撞见「已在别处登录」时的确认卡片。与 Login.svelte 内部
  // 自己的顶替登录确认是两条独立路径（那个是手动登录撞见），互不共用状态。
  // 源：App.tsx 587-596。
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { routerStore } from "../../lib/stores/routerStore.svelte";
</script>

<section class="login-card kick-confirm-card">
  <h2>该账号已在其他设备登录</h2>
  <p class="hint">继续会把另一台设备顶下线（那边会收到提示）。确定要继续吗？</p>
  <div class="kick-confirm-actions">
    <button disabled={sessionStore.restoreKickBusy} onclick={() => { sessionStore.restoreKickPending = null; routerStore.goto("login"); }}>取消</button>
    <button class="primary" disabled={sessionStore.restoreKickBusy} onclick={() => sessionStore.confirmRestoreKick()}>
      {sessionStore.restoreKickBusy ? "登录中…" : "确定顶替登录"}
    </button>
  </div>
</section>
