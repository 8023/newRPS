<script lang="ts">
  // 设备类型/浏览器/操作系统/来源/省份/ISP 分布。源：ui/AnalyticsPanel.tsx 638-648。
  import type { AnalyticsRangeView } from "../../../shared/types";
  import { relabelBuckets, DEVICE_LABELS } from "../../../lib/analyticsDashboard";
  import DonutChart from "../../../lib/charts/DonutChart.svelte";
  import HBarChart from "../../../lib/charts/HBarChart.svelte";
  import ChartCard from "./ChartCard.svelte";
  import BucketTable from "./BucketTable.svelte";

  let { data }: { data: AnalyticsRangeView } = $props();
  const devices = $derived(relabelBuckets(data.devices || [], DEVICE_LABELS));
  const browsers = $derived((data.browsers || []).slice(0, 6));
  const os = $derived((data.os || []).slice(0, 6));
  const referrers = $derived((data.referrers || []).slice(0, 10));
  const provinces = $derived((data.provinces || []).slice(0, 10));
  const isps = $derived((data.isps || []).slice(0, 10));
</script>

<div class="analytics-row-3">
  <ChartCard title="设备类型">
    {#snippet table()}<BucketTable rows={devices} />{/snippet}
    <DonutChart rows={devices} />
  </ChartCard>
  <ChartCard title="浏览器">
    {#snippet table()}<BucketTable rows={browsers} />{/snippet}
    <HBarChart rows={browsers} />
  </ChartCard>
  <ChartCard title="操作系统">
    {#snippet table()}<BucketTable rows={os} />{/snippet}
    <HBarChart rows={os} />
  </ChartCard>
</div>

<div class="analytics-row-3">
  <ChartCard title="来源 Top10">
    {#snippet table()}<BucketTable rows={referrers} />{/snippet}
    <HBarChart rows={referrers} />
  </ChartCard>
  <ChartCard title="省份 Top10">
    {#snippet table()}<BucketTable rows={provinces} />{/snippet}
    <HBarChart rows={provinces} />
  </ChartCard>
  <ChartCard title="ISP Top10">
    {#snippet table()}<BucketTable rows={isps} />{/snippet}
    <HBarChart rows={isps} />
  </ChartCard>
</div>
