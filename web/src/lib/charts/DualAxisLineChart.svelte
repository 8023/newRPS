<script lang="ts">
  /**
   * 双 Y 轴折线图：两个系列量级差异大（如"总数"存量 vs"新增"日增量、大厅消息 vs 房间消息）
   * 时各自独立缩放。LayerChart 单个 Chart 实例只有一条 y 刻度线，没有 recharts <YAxis yAxisId>
   * 那种原生双轴概念——这里用两个 Chart 严格重叠的常见写法达到同样效果：底层图渲染左轴 +
   * 网格 + 系列 A，上层图透明背景只渲染右轴 + 系列 B，两者共享同一个 x 轴（点状定位一致），
   * 顶层图自己不重复画 x 轴/网格，避免重叠。源：ui/AnalyticsPanel.tsx 内多处双轴 <LineChart>
   * （TotalNewChartCard、聊天活跃、单房对局）。
   */
  import { Chart, Spline } from "layerchart/svg";
  import { scalePoint } from "d3-scale";
  import { formatDayTick } from "../analyticsDashboard";
  import AnalyticsTooltip from "./AnalyticsTooltip.svelte";

  let { data, x, left, right, height = 220 }: {
    data: Record<string, string | number>[];
    x: string;
    left: { key: string; label: string; color: string };
    right: { key: string; label: string; color: string };
    height?: number;
  } = $props();

  const xScaleFactory = () => scalePoint().padding(0.5);
</script>

<div class="layerchart-card layerchart-dual-axis" style={`height:${height}px`}>
  {#if data.length === 0}
    <p class="empty">暂无数据</p>
  {:else}
    <Chart
      {data}
      x={x}
      xScale={xScaleFactory()}
      y={left.key}
      yNice
      series={[{ key: left.key, value: left.key, label: left.label, color: left.color }]}
      padding={{ left: 8, bottom: 24, top: 22, right: 32 }}
      axis
      grid={{ x: false }}
      tooltipContext={{ mode: "band" }}
      props={{
        // tickSpacing：band/point 轴默认全显，30 天会把标签叠成一片，按宽度抽稀。
        xAxis: { format: (v: unknown) => formatDayTick(String(v)), tickSpacing: 60, tickLabelProps: { class: "layerchart-tick" } },
        yAxis: { tickLabelProps: { class: "layerchart-tick" } }
      }}
    >
      {#snippet marks({ context })}
        <Spline seriesKey={left.key} stroke={left.color} strokeWidth={2} />
      {/snippet}
      <!-- 气泡由下层这张图统一渲染，但两条线都列出来：上层只是借来画右轴的透明叠层，
           自身 tooltipContext=false，若不在这里手动补上 right 系列，悬停就只看得到左轴那条
           （原 recharts 版是共享气泡，两条一起显示）。 -->
      {#snippet tooltip({ context })}
        <AnalyticsTooltip {context} xKey={x} series={[left, right]} />
      {/snippet}
    </Chart>
    <!-- 右轴叠层：透明背景 + 关闭指针事件，悬停/tooltip 仍由下层图表响应，只借这一层多画一条
         右侧刻度线和一条折线，模拟 recharts 的 yAxisId="right"。 -->
    <div class="layerchart-dual-axis-overlay">
      <Chart
        {data}
        x={x}
        xScale={xScaleFactory()}
        y={right.key}
        yNice
        series={[{ key: right.key, value: right.key, label: right.label, color: right.color }]}
        padding={{ left: 8, bottom: 24, top: 22, right: 32 }}
        axis="y"
        grid={false}
        tooltipContext={false}
        pointerEvents={false}
        props={{ yAxis: { placement: "right", tickLabelProps: { class: "layerchart-tick" } } }}
      >
        {#snippet marks({ context })}
          <Spline seriesKey={right.key} stroke={right.color} strokeWidth={2} />
        {/snippet}
      </Chart>
    </div>
    <div class="layerchart-dual-axis-legend">
      <span><i style={`background:${left.color}`}></i>{left.label}</span>
      <span><i style={`background:${right.color}`}></i>{right.label}</span>
    </div>
  {/if}
</div>
