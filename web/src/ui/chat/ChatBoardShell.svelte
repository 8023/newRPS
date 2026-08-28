<script lang="ts">
  // 源：ui/AppViews.tsx:2963-2981
  import { type Snippet, untrack } from "svelte";
  import { useMobileCollapse } from "../../lib/useMobileCollapse.svelte";
  import CollapseToggle from "../shell/CollapseToggle.svelte";

  let { title, collapseKey, tabs, children }: {
    title: string;
    collapseKey: string;
    tabs?: Snippet;
    children: Snippet;
  } = $props();

  const collapse = useMobileCollapse(untrack(() => collapseKey));
</script>

<div class={`panel chat-panel ${collapse.collapsed ? "collapsed" : ""}`}>
  <div class="chat-panel-head">
    <div class="chat-panel-head-main">
      <h3>{title}</h3>
      {#if tabs}{@render tabs()}{/if}
    </div>
    <CollapseToggle collapsed={collapse.collapsed} onToggle={collapse.toggle} label={title} />
  </div>
  {@render children()}
</div>
