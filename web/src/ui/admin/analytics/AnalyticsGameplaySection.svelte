<script lang="ts">
  // 对局时长/惩罚任务/热门标签/热门系列/随机任务与系列任务增长/对局数/开房数。
  // 源：ui/AnalyticsPanel.tsx 650-752。
  import type { AppConfig, AnalyticsRangeView } from "../../../shared/types";
  import { GAME_LABELS, mergeTagCompareBuckets, orderSeriesStably, relabelBuckets, seriesToChartRows } from "../../../lib/analyticsDashboard";
  import ComposedBarLineChart from "../../../lib/charts/ComposedBarLineChart.svelte";
  import StackedBarChart from "../../../lib/charts/StackedBarChart.svelte";
  import HBarChart from "../../../lib/charts/HBarChart.svelte";
  import DualAxisLineChart from "../../../lib/charts/DualAxisLineChart.svelte";
  import ChartCard from "./ChartCard.svelte";

  let { data, config }: { data: AnalyticsRangeView; config: AppConfig } = $props();

  const tagLabels = $derived.by(() => {
    const map: Record<string, string> = {};
    for (const t of config.punishmentTags || []) map[t.id] = t.name || t.id;
    return map;
  });
  const seriesLabels = $derived.by(() => {
    const map: Record<string, string> = {};
    for (const s of config.punishmentSeriesSummaries || []) map[s.id] = s.name || s.id;
    return map;
  });
  const tagCompareRows = $derived(mergeTagCompareBuckets(data.punishTagInclude || [], data.punishTagExclude || [], tagLabels));
  const seriesRows = $derived(relabelBuckets(data.punishSeriesSelect || [], seriesLabels));

  const gameRoundAvg = $derived(orderSeriesStably(data.gameRoundAvgMinutes || [], GAME_LABELS));
  const roundDurationRows = $derived(
    data.series.days.map((day, i) => {
      const row: Record<string, string | number> = { day, roomAvg: (data.roomAvgMinutes || [])[i] || 0 };
      for (const s of gameRoundAvg) row[s.key] = s.values[i] || 0;
      return row;
    })
  );

  const punishmentRows = $derived(
    data.series.days.map((day, i) => ({
      day,
      pending: Math.max(0, (data.punishment?.publish?.[i] || 0) - (data.punishment?.done?.[i] || 0) - (data.punishment?.reject?.[i] || 0)),
      done: data.punishment?.done?.[i] || 0,
      reject: data.punishment?.reject?.[i] || 0
    }))
  );

  const gameRoundsStable = $derived(orderSeriesStably(data.gameRounds || [], GAME_LABELS));
  const roomCreatesStable = $derived(orderSeriesStably(data.roomCreates || [], GAME_LABELS));
</script>

<div class="analytics-row-2">
  <ChartCard title="对局时长">
    {#if gameRoundAvg.length}
      <ComposedBarLineChart
        data={roundDurationRows} x="day"
        barSeries={gameRoundAvg.map((s) => ({ key: s.key, label: GAME_LABELS[s.key] || s.key }))}
        line={{ key: "roomAvg", label: "房间均值", color: "var(--chart-critical)" }}
      />
    {:else}
      <p class="empty">暂无数据</p>
    {/if}
  </ChartCard>

  <ChartCard title="惩罚任务">
    <p class="analytics-inline-kpi">完成率 <strong>{((data.punishment?.doneRate || 0) * 100).toFixed(1)}%</strong></p>
    <StackedBarChart data={punishmentRows} x="day" showPercent showTotal={false} series={[
      { key: "pending", label: "进行中", color: "var(--chart-1)" },
      { key: "done", label: "完成", color: "var(--chart-3)" },
      { key: "reject", label: "驳回", color: "var(--chart-critical)" }
    ]} />
  </ChartCard>
</div>

<div class="analytics-row-2">
  <ChartCard title="热门惩罚标签">
    <HBarChart rows={tagCompareRows.slice(0, 10)} showPercent series={[
      { key: "include", label: "选中", color: "var(--chart-1)" },
      { key: "exclude", label: "拒绝", color: "var(--chart-critical)" }
    ]} />
  </ChartCard>
  <ChartCard title="热门系列任务">
    <!-- showLegend：左边「热门惩罚标签」有图例，这张单系列图补一条同高的图例，
         两张图的横轴才落在同一条线上。 -->
    <HBarChart rows={seriesRows} showLegend valueLabel="选中次数" />
  </ChartCard>
</div>

<div class="analytics-row-2">
  <ChartCard title="随机任务">
    <DualAxisLineChart
      data={data.series.days.map((day, i) => ({ day, total: (data.randomTaskPool?.total || [])[i] || 0, new: (data.randomTaskPool?.new || [])[i] || 0 }))}
      x="day"
      left={{ key: "new", label: "新增", color: "var(--chart-2)" }}
      right={{ key: "total", label: "总数", color: "var(--chart-1)" }}
    />
  </ChartCard>
  <ChartCard title="系列任务">
    <DualAxisLineChart
      data={data.series.days.map((day, i) => ({ day, total: (data.seriesTaskPool?.total || [])[i] || 0, new: (data.seriesTaskPool?.new || [])[i] || 0 }))}
      x="day"
      left={{ key: "new", label: "新增", color: "var(--chart-2)" }}
      right={{ key: "total", label: "总数", color: "var(--chart-1)" }}
    />
  </ChartCard>
</div>

<div class="analytics-row-2">
  <ChartCard title="对局数">
    {#if gameRoundsStable.length}
      <StackedBarChart data={seriesToChartRows(data.series.days, gameRoundsStable)} x="day" showPercent
        series={gameRoundsStable.map((s) => ({ key: s.key, label: GAME_LABELS[s.key] || s.key }))} />
    {:else}<p class="empty">暂无数据</p>{/if}
  </ChartCard>
  <ChartCard title="开房数">
    {#if roomCreatesStable.length}
      <StackedBarChart data={seriesToChartRows(data.series.days, roomCreatesStable)} x="day" showPercent
        series={roomCreatesStable.map((s) => ({ key: s.key, label: GAME_LABELS[s.key] || s.key }))} />
    {:else}<p class="empty">暂无数据</p>{/if}
  </ChartCard>
</div>
