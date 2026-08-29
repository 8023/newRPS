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
  import BucketTable from "./BucketTable.svelte";
  import SeriesTable from "./SeriesTable.svelte";

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
    {#snippet table()}
      <div class="analytics-table-scroll">
        <table class="analytics-data-table">
          <thead><tr><th>日期</th>{#each gameRoundAvg as s (s.key)}<th>{GAME_LABELS[s.key] || s.key}</th>{/each}<th>房间均值</th></tr></thead>
          <tbody>
            {#each data.series.days as d, i (d)}
              <tr><td>{d}</td>{#each gameRoundAvg as s (s.key)}<td>{(s.values[i] || 0).toFixed(1)}</td>{/each}<td>{((data.roomAvgMinutes || [])[i] || 0).toFixed(1)}</td></tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/snippet}
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
    {#snippet table()}
      <table class="analytics-data-table">
        <thead><tr><th>日期</th><th>发布</th><th>完成</th><th>驳回</th></tr></thead>
        <tbody>
          {#each data.series.days as d, i (d)}
            <tr><td>{d}</td><td>{data.punishment?.publish?.[i] || 0}</td><td>{data.punishment?.done?.[i] || 0}</td><td>{data.punishment?.reject?.[i] || 0}</td></tr>
          {/each}
        </tbody>
      </table>
    {/snippet}
    <p class="analytics-inline-kpi">完成率 <strong>{((data.punishment?.doneRate || 0) * 100).toFixed(1)}%</strong></p>
    <StackedBarChart data={punishmentRows} x="day" height={236} showPercent showTotal={false} series={[
      { key: "pending", label: "进行中", color: "var(--chart-1)" },
      { key: "done", label: "完成", color: "var(--chart-3)" },
      { key: "reject", label: "驳回", color: "var(--chart-critical)" }
    ]} />
  </ChartCard>
</div>

<div class="analytics-row-2">
  <ChartCard title="热门惩罚标签">
    {#snippet table()}
      <table class="analytics-data-table">
        <thead><tr><th>标签</th><th>选中</th><th>拒绝</th><th>合计</th></tr></thead>
        <tbody>
          {#each tagCompareRows.slice(0, 10) as r (r.key)}
            <tr><td>{r.key}</td><td>{r.include}</td><td>{r.exclude}</td><td>{r.include + r.exclude}</td></tr>
          {/each}
          {#if tagCompareRows.length === 0}<tr><td colspan="4" class="empty">暂无数据</td></tr>{/if}
        </tbody>
      </table>
    {/snippet}
    <StackedBarChart data={tagCompareRows.slice(0, 10)} x="key" height={260} angledLabels series={[
      { key: "include", label: "选中", color: "var(--chart-1)" },
      { key: "exclude", label: "拒绝", color: "var(--chart-critical)" }
    ]} />
  </ChartCard>
  <ChartCard title="热门系列任务">
    {#snippet table()}<BucketTable rows={seriesRows} />{/snippet}
    <HBarChart rows={seriesRows} height={260} />
  </ChartCard>
</div>

<div class="analytics-row-2">
  <ChartCard title="随机任务">
    {#snippet table()}
      <table class="analytics-data-table">
        <thead><tr><th>日期</th><th>总数</th><th>新增</th></tr></thead>
        <tbody>{#each data.series.days as d, i (d)}<tr><td>{d}</td><td>{(data.randomTaskPool?.total || [])[i] || 0}</td><td>{(data.randomTaskPool?.new || [])[i] || 0}</td></tr>{/each}</tbody>
      </table>
    {/snippet}
    <DualAxisLineChart
      data={data.series.days.map((day, i) => ({ day, total: (data.randomTaskPool?.total || [])[i] || 0, new: (data.randomTaskPool?.new || [])[i] || 0 }))}
      x="day"
      left={{ key: "new", label: "新增", color: "var(--chart-2)" }}
      right={{ key: "total", label: "总数", color: "var(--chart-1)" }}
    />
  </ChartCard>
  <ChartCard title="系列任务">
    {#snippet table()}
      <table class="analytics-data-table">
        <thead><tr><th>日期</th><th>总数</th><th>新增</th></tr></thead>
        <tbody>{#each data.series.days as d, i (d)}<tr><td>{d}</td><td>{(data.seriesTaskPool?.total || [])[i] || 0}</td><td>{(data.seriesTaskPool?.new || [])[i] || 0}</td></tr>{/each}</tbody>
      </table>
    {/snippet}
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
    {#snippet table()}<SeriesTable days={data.series.days} series={data.gameRounds || []} labels={GAME_LABELS} />{/snippet}
    {#if gameRoundsStable.length}
      <StackedBarChart data={seriesToChartRows(data.series.days, gameRoundsStable)} x="day" height={240} showPercent
        series={gameRoundsStable.map((s) => ({ key: s.key, label: GAME_LABELS[s.key] || s.key }))} />
    {:else}<p class="empty">暂无数据</p>{/if}
  </ChartCard>
  <ChartCard title="开房数">
    {#snippet table()}<SeriesTable days={data.series.days} series={data.roomCreates || []} labels={GAME_LABELS} />{/snippet}
    {#if roomCreatesStable.length}
      <StackedBarChart data={seriesToChartRows(data.series.days, roomCreatesStable)} x="day" height={240} showPercent
        series={roomCreatesStable.map((s) => ({ key: s.key, label: GAME_LABELS[s.key] || s.key }))} />
    {:else}<p class="empty">暂无数据</p>{/if}
  </ChartCard>
</div>
