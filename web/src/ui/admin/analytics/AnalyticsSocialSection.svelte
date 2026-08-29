<script lang="ts">
  // 会话/证明耗时分布、名争·白给、主宠关系、用户信息变更、聊天活跃/人数、单房对局。
  // 源：ui/AnalyticsPanel.tsx 755-932。
  import type { AnalyticsRangeView } from "../../../shared/types";
  import { ACTIVITY_LABELS, CHART_COLORS, orderSeriesStably, seriesToChartRows } from "../../../lib/analyticsDashboard";
  import ColoredBarChart from "../../../lib/charts/ColoredBarChart.svelte";
  import LineTrendChart from "../../../lib/charts/LineTrendChart.svelte";
  import DualAxisLineChart from "../../../lib/charts/DualAxisLineChart.svelte";
  import ChartCard from "./ChartCard.svelte";

  let { data }: { data: AnalyticsRangeView } = $props();

  const nameWarGiveaway = $derived(orderSeriesStably(data.nameWarGiveaway || [], ACTIVITY_LABELS));
  const profileChanges = $derived(orderSeriesStably(data.profileChanges || [], ACTIVITY_LABELS));

  const chatRows = $derived(data.series.days.map((day, i) => ({ day, lobby: data.chat?.lobby?.[i] || 0, room: data.chat?.room?.[i] || 0 })));
  const roomRoundsRows = $derived(data.series.days.map((day, i) => ({ day, max: data.roomRounds?.max?.[i] || 0, avg: data.roomRounds?.avg?.[i] || 0 })));
  const speakersRows = $derived(data.series.days.map((day, i) => ({ day, lobby: data.chat?.speakers?.[i] || 0, room: data.chat?.speakersRoom?.[i] || 0 })));
</script>

<div class="analytics-row-2">
  <ChartCard title="会话时长分布">
    <ColoredBarChart rows={data.sessionBuckets || []} valueLabel="会话数" />
  </ChartCard>
  <ChartCard title="证明耗时分布">
    <ColoredBarChart rows={data.punishment?.proofMs || []} valueLabel="次数" />
  </ChartCard>
</div>

<div class="analytics-row-2">
  <ChartCard title="名争·白给">
    {#if nameWarGiveaway.length}
      <LineTrendChart data={seriesToChartRows(data.series.days, nameWarGiveaway)} x="day"
        series={nameWarGiveaway.map((s, i) => ({ key: s.key, label: ACTIVITY_LABELS[s.key] || s.key, color: CHART_COLORS[i % CHART_COLORS.length] }))} />
    {:else}<p class="empty">暂无数据</p>{/if}
  </ChartCard>
  <ChartCard title="主宠关系">
    <DualAxisLineChart
      data={data.series.days.map((day, i) => ({ day, total: (data.petBond?.total || [])[i] || 0, new: (data.petBond?.new || [])[i] || 0 }))}
      x="day"
      left={{ key: "new", label: "新增", color: "var(--chart-2)" }}
      right={{ key: "total", label: "总数", color: "var(--chart-1)" }}
    />
  </ChartCard>
</div>

<div class="analytics-row-2">
  <ChartCard title="用户信息变更">
    {#if profileChanges.length}
      <LineTrendChart data={seriesToChartRows(data.series.days, profileChanges)} x="day"
        series={profileChanges.map((s, i) => ({ key: s.key, label: ACTIVITY_LABELS[s.key] || s.key, color: CHART_COLORS[i % CHART_COLORS.length] }))} />
    {:else}<p class="empty">暂无数据</p>{/if}
  </ChartCard>

  <ChartCard title="聊天活跃">
    <DualAxisLineChart data={chatRows} x="day" left={{ key: "lobby", label: "大厅消息", color: "var(--chart-1)" }} right={{ key: "room", label: "房间消息", color: "var(--chart-2)" }} />
  </ChartCard>
</div>

<div class="analytics-row-2">
  <ChartCard title="单房对局">
    <DualAxisLineChart data={roomRoundsRows} x="day" left={{ key: "avg", label: "单房平均", color: "var(--chart-2)" }} right={{ key: "max", label: "单房最多", color: "var(--chart-1)" }} />
  </ChartCard>

  <ChartCard title="聊天人数">
    <LineTrendChart data={speakersRows} x="day" height={210} series={[
      { key: "lobby", label: "大厅发言人", color: "var(--chart-1)" },
      { key: "room", label: "房间发言人", color: "var(--chart-2)" }
    ]} />
  </ChartCard>
</div>
