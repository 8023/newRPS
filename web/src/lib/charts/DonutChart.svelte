<script lang="ts">
  /**
   * 圆环图（设备类型分布）：用 LayerChart 的 Arc 原语手动摆放每一段扇形——角度自己算
   * （纯算术，不依赖 d3-shape 的 pie 生成器），换来不必深入 Chart 的 x/y 比例尺体系，
   * 一个不需要坐标轴的圆环图没必要为了配色比例尺硬套一整套笛卡尔上下文。
   * 悬停用原生 <title> 展示数值/占比，不接 LayerChart 的 TooltipContext（那套面向笛卡尔
   * 图表的量化提示框，对这种"只有 N 个扇形"的场景没有必要）。
   * 源：ui/AnalyticsPanel.tsx 的 HBarCard donut 分支（recharts <Pie>）。
   */
  import { Chart, Group, Arc } from "layerchart/svg";
  import { CHART_COLORS, formatNum } from "../analyticsDashboard";
  import ChartLegend from "./ChartLegend.svelte";

  let { rows, size = 176 }: { rows: { key: string; value: number }[]; size?: number } = $props();

  const total = $derived(rows.reduce((s, r) => s + r.value, 0));
  const arcs = $derived.by(() => {
    let angle = 0;
    return rows.map((r, i) => {
      const fraction = total > 0 ? r.value / total : 0;
      const startAngle = angle;
      const endAngle = angle + fraction * Math.PI * 2;
      angle = endAngle;
      return { ...r, startAngle, endAngle, fraction, color: CHART_COLORS[i % CHART_COLORS.length] };
    });
  });
</script>

<div class="analytics-donut-wrap">
  <!-- 数值覆盖层锚在圆环这一层而不是外层：外层要跟着卡片一起被拉伸（同一行的卡片等高），
       锚在外层的话圆环停在上方、"合计"却掉到拉伸后的正中间，两者会错位。 -->
  <div class="analytics-donut-ring" style={`width:${size}px;height:${size}px`}>
    <Chart data={rows} x="key" y="value" padding={4} axis={false} grid={false} tooltipContext={false}>
      {#snippet marks()}
        <Group center>
          {#each arcs as a (a.key)}
            <Arc
              startAngle={a.startAngle}
              endAngle={a.endAngle}
              innerRadius={size * 0.3}
              outerRadius={size * 0.42}
              cornerRadius={2}
              padAngle={0.015}
              fill={a.color}
            >
              {#snippet children()}
                <title>{a.key}：{formatNum(a.value)}（{(a.fraction * 100).toFixed(1)}%）</title>
              {/snippet}
            </Arc>
          {/each}
        </Group>
      {/snippet}
    </Chart>
    <div class="analytics-donut-center">
      <strong>{formatNum(total)}</strong>
      <span>合计</span>
    </div>
  </div>
</div>
<ChartLegend items={arcs.map((a) => ({ key: a.key, label: a.key, color: a.color, value: `${(a.fraction * 100).toFixed(1)}%` }))} />
