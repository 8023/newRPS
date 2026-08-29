<script lang="ts">
  // 设备类型/浏览器/操作系统/来源/省份/ISP 分布。源：ui/AnalyticsPanel.tsx 638-648。
  import type { AnalyticsRangeView } from "../../../shared/types";
  import { relabelBuckets, DEVICE_LABELS } from "../../../lib/analyticsDashboard";
  import DonutChart from "../../../lib/charts/DonutChart.svelte";
  import HBarChart from "../../../lib/charts/HBarChart.svelte";
  import ChartCard from "./ChartCard.svelte";

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
    <DonutChart rows={devices} />
  </ChartCard>
  <ChartCard title="浏览器">
    <HBarChart rows={browsers} showPercent />
  </ChartCard>
  <ChartCard title="操作系统">
    <HBarChart rows={os} showPercent />
  </ChartCard>
</div>

<div class="analytics-row-3">
  <ChartCard title="来源 Top10">
    <HBarChart rows={referrers} showPercent />
  </ChartCard>
  <ChartCard title="省份 Top10">
    <HBarChart rows={provinces} showPercent />
  </ChartCard>
  <ChartCard title="ISP Top10">
    <HBarChart rows={isps} showPercent maxLabelChars={9} />
  </ChartCard>
</div>
