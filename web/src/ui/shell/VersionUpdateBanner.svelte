<script lang="ts">
  // 源：App.tsx 的 remoteBuildId/dismissedRemoteBuildId 渲染分支。
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
</script>

{#if sessionStore.remoteBuildId && sessionStore.remoteBuildId !== sessionStore.dismissedRemoteBuildId}
  <div class="announcement-popup version-update-popup" role="alert">
    <div>
      <b>网站内容已更新</b>
      <p>检测到新版本，当前页面可能已经过期，建议刷新以获取最新功能与修复。</p>
    </div>
    <div class="version-update-actions">
      <button class="version-update-button version-update-refresh" type="button" onclick={() => window.location.reload()}>刷新</button>
      <button class="version-update-button version-update-ignore" type="button" onclick={() => (sessionStore.dismissedRemoteBuildId = sessionStore.remoteBuildId)}>忽略</button>
    </div>
  </div>
{/if}
