<script lang="ts">
  // 源：ui/HelpPanel.tsx
  import BookOpen from "@lucide/svelte/icons/book-open";
  import helpHtml from "../../../../help.md?html";

  let { onClose }: { onClose: () => void } = $props();
</script>

<div class="modal-backdrop sponsor-backdrop" onclick={(event) => { if (event.target === event.currentTarget) onClose(); }}>
  <section class="help-modal" onclick={(event) => event.stopPropagation()}>
    <div class="modal-title sponsor-title">
      <div>
        <h2><BookOpen size={20} /> 游戏说明</h2>
        <p class="hint">了解六款游戏规则、全局玩法机制与页面功能。</p>
      </div>
      <button type="button" class="icon-button" onclick={onClose}>×</button>
    </div>
    <div class="help-content">{@html helpHtml}</div>
  </section>
</div>
