<script lang="ts">
  // toast 展示文案与离场态：uiStore.notice 更新时同步文案，定时先 leave 再清空，以便
  // 播放退出动效。进出场动画的定时器是这个组件自己的私有实现细节，不需要下放到 uiStore。
  // 源：App.tsx 的 toastText/toastLeaving 状态 + 对应 effect。
  import { uiStore } from "../../lib/stores/uiStore.svelte";

  let toastText = $state("");
  let toastLeaving = $state(false);

  $effect(() => {
    const notice = uiStore.notice;
    if (!notice) return;
    toastText = notice;
    toastLeaving = false;
    const leaveTimer = window.setTimeout(() => { toastLeaving = true; }, 3200);
    const clearTimer = window.setTimeout(() => {
      toastText = "";
      toastLeaving = false;
      uiStore.notice = "";
    }, 3520);
    return () => {
      window.clearTimeout(leaveTimer);
      window.clearTimeout(clearTimer);
    };
  });
</script>

{#if toastText}
  <div class={`notice toast-notice ${toastLeaving ? "toast-leave" : "toast-enter"}`} role="status">
    {toastText}
  </div>
{/if}
