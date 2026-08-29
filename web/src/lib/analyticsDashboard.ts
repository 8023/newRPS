// 后台「数据分析」面板的纯逻辑辅助：标签翻译、格式化、行/表格数据整形。
// 源：ui/AnalyticsPanel.tsx 顶部散落的纯函数（不含任何 recharts/组件相关代码）。
// 注意与 lib/analytics.ts 不是同一个东西——那个是客户端埋点上报（track/startAnalytics），
// 这里是后台统计仪表盘自己的展示层辅助，两者互不相关，只是命名相近，故意分开成两个文件
// 避免互相遮蔽 export。
import type { AnalyticsBucket, AnalyticsNamedSeries, AnalyticsRangeView } from "../shared/types";

export const CHART_COLORS = ["var(--chart-1)", "var(--chart-2)", "var(--chart-3)", "var(--chart-4)", "var(--chart-5)", "var(--chart-6)"];
export const SEQ_COLORS = ["var(--chart-seq-1)", "var(--chart-seq-2)", "var(--chart-seq-3)", "var(--chart-seq-4)", "var(--chart-seq-5)", "var(--chart-seq-6)"];

// data.gameRounds / data.roomCreates 的 series key 是后端存的 gameId（"unknown" 为
// game_id 为空的历史脏数据），图例/表头需要中文名，不能直接展示英文标识符。
export const GAME_LABELS: Record<string, string> = {
  rps: "猜拳", othello: "黑白棋", tictactoe: "井字棋", gomoku: "五子棋",
  jungle: "斗兽棋", chess: "国际象棋", liarsdice: "大话骰", unknown: "未知游戏"
};

// data.profileChanges / data.nameWarGiveaway 的 series key 是 player_activity_events.action
// （logPlayerActivity 的第一个参数，见 internal/server/handlers.go），同样是英文标识符。
export const ACTIVITY_LABELS: Record<string, string> = {
  rename: "改名", gender_change: "性别/阵营变更",
  self_title_change: "自定义称号", avatar_change: "更换头像",
  nameWar_enable: "开启名争", giveaway_enable: "开启白给",
  giveaway_board_submit: "白给留言板", nameWar_rename: "名争改名",
  extreme_enable: "开启极限"
};

// data.devices 的 key 是后端 parseUserAgent 粗分类出的 deviceType（mobile/tablet/desktop，
// 见 internal/server/useragent.go），面板展示需要中文。
export const DEVICE_LABELS: Record<string, string> = { mobile: "手机", tablet: "平板", desktop: "电脑" };

// data.viewPv 的 key 是前端路由/弹窗名，"明细·页面浏览" 表格需要中文页面名。
export const VIEW_LABELS: Record<string, string> = {
  login: "登录页", lobby: "大厅", room: "房间", admin: "后台管理",
  profile: "个人资料", leaderboard: "排行榜", about: "关于", help: "游戏说明"
};

export function labelFor(labels: Record<string, string> | undefined, key: string): string {
  return labels?.[key] || key;
}

export function relabelBuckets(rows: AnalyticsBucket[], labels: Record<string, string>): AnalyticsBucket[] {
  return rows.map((r) => ({ ...r, key: labelFor(labels, r.key) }));
}

// data.punishTagInclude / data.punishTagExclude 的 key 是标签 ID，「随机惩罚」图需要把
// 选中/排除两组按同一标签合并成一行，而不是分两张图各显示一半。
export type TagCompareRow = { key: string; include: number; exclude: number };
export function mergeTagCompareBuckets(include: AnalyticsBucket[], exclude: AnalyticsBucket[], labels: Record<string, string>): TagCompareRow[] {
  const map = new Map<string, TagCompareRow>();
  const bump = (rows: AnalyticsBucket[], field: "include" | "exclude") => {
    for (const r of rows) {
      const cur = map.get(r.key) || { key: labelFor(labels, r.key), include: 0, exclude: 0 };
      cur[field] += r.value;
      map.set(r.key, cur);
    }
  };
  bump(include, "include");
  bump(exclude, "exclude");
  return Array.from(map.values()).sort((a, b) => (b.include + b.exclude) - (a.include + a.exclude));
}

export function formatNum(n: number): string {
  if (!Number.isFinite(n)) return "—";
  if (Math.abs(n) >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (Math.abs(n) >= 10_000) return `${(n / 1000).toFixed(1)}k`;
  return String(Math.round(n));
}

export function formatDurationMs(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(0)}s`;
  if (ms < 3_600_000) return `${(ms / 60_000).toFixed(1)}m`;
  return `${(ms / 3_600_000).toFixed(1)}h`;
}

export function formatDelta(d: number): string {
  if (!Number.isFinite(d) || d === 0) return "持平";
  const pct = Math.round(d * 100);
  return pct > 0 ? `↑${pct}%` : `↓${Math.abs(pct)}%`;
}

// 图表横轴的日期刻度省略年份只显示 mm/dd；明细表格仍展示完整 yyyy-mm-dd。
export function formatDayTick(v: string): string {
  const m = /^\d{4}-(\d{2})-(\d{2})$/.exec(v);
  return m ? `${m[1]}/${m[2]}` : v;
}

export function seriesToChartRows(days: string[], series: AnalyticsNamedSeries[]) {
  return days.map((day, i) => {
    const row: Record<string, string | number> = { day };
    for (const s of series) row[s.key] = s.values[i] || 0;
    return row;
  });
}

export function trendRows(data: AnalyticsRangeView) {
  return data.series.days.map((day, i) => ({
    day,
    dau: data.series.dau[i] || 0,
    sessions: data.series.sessions[i] || 0,
    loggedDau: data.series.loggedDau[i] || 0,
    pageviews: data.series.pageviews[i] || 0,
    newVisitors: data.series.newVisitors[i] || 0,
    returning: data.series.returning[i] || 0,
    newUsers: data.newOldUsers?.new?.[i] || 0,
    oldLogin: data.newOldUsers?.oldLogin?.[i] || 0
  }));
}

export function funnelOrdered(rows: AnalyticsBucket[]): AnalyticsBucket[] {
  // 与 analytics_agg rebuildDay 的 funnel 键一致：五层均为设备 UV
  const order = ["visit", "lobby", "room", "round", "finish"];
  const labels: Record<string, string> = { visit: "访问", lobby: "进大厅", room: "进房", round: "开局", finish: "完成对局" };
  const map = new Map(rows.map((r) => [r.key, r.value]));
  return order.map((k) => ({ key: labels[k] || k, value: map.get(k) || 0 }));
}

// 后端 namedSeries 按当前窗口总量降序排列，顺序会在两次轮询之间悄悄变化；用 labels（游戏
// 白名单）里固定的 key 顺序重排，让同一张图表在任意两次轮询之间顺序恒定、配色不漂移。
export function orderSeriesStably(series: AnalyticsNamedSeries[], labels?: Record<string, string>): AnalyticsNamedSeries[] {
  if (!labels) return series;
  const order = Object.keys(labels);
  return [...series].sort((a, b) => {
    const ia = order.indexOf(a.key);
    const ib = order.indexOf(b.key);
    const ra = ia === -1 ? order.length : ia;
    const rb = ib === -1 ? order.length : ib;
    return ra !== rb ? ra - rb : a.key.localeCompare(b.key);
  });
}
