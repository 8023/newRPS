<script lang="ts">
  // 全服公告弹窗：内容与自动消失定时器均来自 sessionStore.announcement。
  // 源：App.tsx 的 announcement 状态 + 对应 effect。
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";

  $effect(() => {
    const current = sessionStore.announcement;
    if (!current) return;
    const timer = window.setTimeout(() => {
      if (sessionStore.announcement === current) sessionStore.announcement = null;
    }, current.durationMs);
    return () => window.clearTimeout(timer);
  });
</script>

{#if sessionStore.announcement}
  <div class="announcement-popup" role="alert">
    <div>
      <b>全服公告</b>
      <p>{sessionStore.announcement.message}</p>
    </div>
    <button class="icon-button" type="button" aria-label="关闭公告" onclick={() => (sessionStore.announcement = null)}>×</button>
  </div>
{/if}
