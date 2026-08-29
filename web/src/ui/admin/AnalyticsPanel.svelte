<script lang="ts">
  /**
   * 「数据分析」分区：访问留存、设备渠道、游戏与站点玩法。数据来自日聚合快照，每 60s 轮询
   * 刷新。图表全部通过 lib/charts/* 的 LayerChart 封装组件渲染；本文件只负责时间范围筛选、
   * 轮询、以及把同一份快照分发给下面 4 个按主题拆开的小节组件——避免把所有图表堆进一个
   * 数据分析组合根。源：ui/AnalyticsPanel.tsx（原 export function AnalyticsPanel）。
   * 由 AdminPanel.svelte 通过 `import("./AnalyticsPanel.svelte")` 动态加载，图表相关代码
   * （及体积不小的 layerchart/d3-*）只在管理员真正点开这个分区时才下载。
   */
  import type { AppConfig, AnalyticsRangeView } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import AnalyticsOverviewSection from "./analytics/AnalyticsOverviewSection.svelte";
  import AnalyticsAudienceSection from "./analytics/AnalyticsAudienceSection.svelte";
  import AnalyticsGameplaySection from "./analytics/AnalyticsGameplaySection.svelte";
  import AnalyticsSocialSection from "./analytics/AnalyticsSocialSection.svelte";
  import AnalyticsFunnelSection from "./analytics/AnalyticsFunnelSection.svelte";

  let { config, onError }: { config: AppConfig; onError: (message: string) => void } = $props();

  let days = $state(30);
  let data = $state<AnalyticsRangeView | null>(null);
  let loading = $state(false);
  let requestGen = 0;

  async function load(opts: { refresh?: boolean } = {}) {
    const gen = ++requestGen;
    loading = true;
    try {
      const snap = await ask<AnalyticsRangeView>("admin:analytics", { days, refresh: !!opts.refresh });
      if (gen !== requestGen) return;
      data = snap;
    } catch (error) {
      if (gen !== requestGen) return;
      onError(error instanceof Error ? error.message : "加载统计失败");
    } finally {
      if (gen === requestGen) loading = false;
    }
  }

  $effect(() => {
    void days;
    void load();
    const timer = setInterval(() => void load(), 60_000);
    return () => clearInterval(timer);
  });

  const opacity = $derived(loading && data ? 0.6 : 1);
</script>

<div class="analytics-panel">
  <p class="analytics-description">访问留存、设备渠道、游戏与站点玩法。数据来自日聚合快照，约每分钟刷新；不会拖慢游戏主链路。</p>
  <div class="analytics-filters">
    {#each [7, 30, 90] as d (d)}
      <button type="button" class={days === d ? "active" : ""} onclick={() => (days = d)}>近 {d} 天</button>
    {/each}
    <button type="button" onclick={() => load({ refresh: true })} disabled={loading}>刷新</button>
    {#if data?.builtAt}
      <span class="analytics-updated">
        最后更新 {new Date(data.builtAt).toLocaleTimeString()}
        {#if data.liveOnline != null}· 当前在线 {data.liveOnline}{/if}
      </span>
    {/if}
  </div>

  {#if !data && loading}<div class="analytics-loading">正在加载图表…</div>{/if}
  {#if !data && !loading}<p class="empty">统计数据尚未就绪，请稍后再试</p>{/if}

  {#if data}
    <div class="analytics-grid" style={`opacity:${opacity}`}>
      <div class="analytics-section-heading"><strong>核心趋势</strong><span>访问、会话与留存</span></div>
      <AnalyticsOverviewSection {data} />
      <div class="analytics-section-heading"><strong>设备与来源</strong><span>终端、渠道与地域分布</span></div>
      <AnalyticsAudienceSection {data} />
      <div class="analytics-section-heading"><strong>游戏与惩罚</strong><span>对局、任务与内容增长</span></div>
      <AnalyticsGameplaySection {data} {config} />
      <div class="analytics-section-heading"><strong>互动与行为</strong><span>社交玩法、聊天与用户操作</span></div>
      <AnalyticsSocialSection {data} />
      <div class="analytics-section-heading"><strong>转化与明细</strong><span>关键路径与原始记录</span></div>
      <AnalyticsFunnelSection {data} {days} />
    </div>
  {/if}
</div>
