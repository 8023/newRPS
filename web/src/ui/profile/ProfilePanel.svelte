<script lang="ts">
  // TODO(svelte-migration Phase 7)：完整迁移 ui/AppViews.tsx:4695-5241 的 ProfilePanel。
  // 当前仅占位，直接读写 sessionStore.me/uiStore.theme，不接收 props。
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";

  const me = $derived(sessionStore.me!.player);

  function close() {
    uiStore.profileOpen = false;
  }
</script>

<div class="modal-backdrop" onclick={(event) => { if (event.target === event.currentTarget) close(); }}>
  <section class="panel">
    <p>个人设置：{me.name}（Svelte 迁移中）</p>
    <button type="button" onclick={close}>关闭</button>
  </section>
</div>
