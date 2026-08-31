<script lang="ts">
  // 设备类型/浏览器/操作系统/来源/省份/ISP 分布。源：ui/AnalyticsPanel.tsx 638-648。
  import type { AnalyticsRangeView } from "../../../shared/types";
  import { relabelBuckets, DEVICE_LABELS, PLOT_HEIGHT_DISTRIBUTION } from "../../../lib/analyticsDashboard";
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
  <ChartCard title="设备类型" minHeight={PLOT_HEIGHT_DISTRIBUTION}>
    <DonutChart rows={devices} />
  </ChartCard>
  <ChartCard title="浏览器" minHeight={PLOT_HEIGHT_DISTRIBUTION}>
    <HBarChart rows={browsers} showPercent />
  </ChartCard>
  <ChartCard title="操作系统" minHeight={PLOT_HEIGHT_DISTRIBUTION}>
    <HBarChart rows={os} showPercent />
  </ChartCard>
</div>

<div class="analytics-row-3">
  <ChartCard title="来源 Top10" minHeight={PLOT_HEIGHT_DISTRIBUTION}>
    <HBarChart rows={referrers} showPercent />
  </ChartCard>
  <ChartCard title="省份 Top10" minHeight={PLOT_HEIGHT_DISTRIBUTION}>
    <HBarChart rows={provinces} showPercent />
  </ChartCard>
  <ChartCard title="ISP Top10" minHeight={PLOT_HEIGHT_DISTRIBUTION}>
    <HBarChart rows={isps} showPercent />
  </ChartCard>
</div>
