<script lang="ts">
  // 图表卡片外壳：标题 + 图表内容。详细数值统一由图表悬停气泡展示。
  // 源：ui/AnalyticsPanel.tsx 的 ChartCard。
  //
  // 卡片同时负责「滚到附近才挂载图表」：整个面板有 30 来张 LayerChart，一次性实例化要占
  // 主线程 2.6 秒（实测点开分区到出图 3.0s，其中 WS 往返只有 3ms、chunk 下载 20ms，其余
  // 全是组件挂载），这是"点开数据分析要等三秒"的全部成因。首屏之外的卡片先渲染等高占位块，
  // 进入视口前 600px 才真正挂载，首屏因此只需实例化 2-3 张图。占位块与图表等高，滚动条
  // 长度和滚动位置都不会跳。
  import type { Snippet } from "svelte";
  import { PLOT_HEIGHT_TREND } from "../../../lib/analyticsDashboard";

  let { title, minHeight = PLOT_HEIGHT_TREND, children }: { title: string; minHeight?: number; children: Snippet } = $props();

  let host = $state<HTMLElement>();
  let mounted = $state(false);

  $effect(() => {
    if (!host || mounted) return;
    // 老浏览器 / 测试环境没有 IntersectionObserver 时直接全量渲染，退回旧行为。
    if (typeof IntersectionObserver !== "function") {
      mounted = true;
      return;
    }
    const io = new IntersectionObserver(
      (entries) => {
        if (!entries.some((entry) => entry.isIntersecting)) return;
        mounted = true;
        io.disconnect();
      },
      { rootMargin: "600px 0px" }
    );
    io.observe(host);
    return () => io.disconnect();
  });
</script>

<div class="analytics-card" bind:this={host}>
  <div class="analytics-card-head">
    <h3>{title}</h3>
  </div>
  <div class="analytics-card-body" style={`min-height:${minHeight}px`}>
    {#if mounted}
      {@render children()}
    {:else}
      <div class="analytics-card-skeleton" aria-hidden="true"></div>
    {/if}
  </div>
</div>
