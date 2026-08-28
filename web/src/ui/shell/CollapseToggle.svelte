<script lang="ts">
  // 源：ui/AppViews.tsx:64-77
  import ChevronDown from "@lucide/svelte/icons/chevron-down";

  let { collapsed, onToggle, label }: { collapsed: boolean; onToggle: () => void; label: string } = $props();
</script>

<button
  type="button"
  class={`mobile-collapse-toggle ${collapsed ? "collapsed" : ""}`}
  onclick={onToggle}
  aria-expanded={!collapsed}
  aria-label={collapsed ? `展开${label}` : `收起${label}`}
  title={collapsed ? `展开${label}` : `收起${label}`}
>
  <ChevronDown size={16} />
</button>
