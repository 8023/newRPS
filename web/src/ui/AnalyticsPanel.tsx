import { Fragment, useEffect, useMemo, useRef, useState, type CSSProperties, type ReactNode } from "react";
import {
  Bar, BarChart, CartesianGrid, Cell, ComposedChart, Legend, Line, LineChart,
  Pie, PieChart, Rectangle, ResponsiveContainer, Tooltip, XAxis, YAxis
} from "recharts";
import type {
  AnalyticsBucket, AnalyticsNamedSeries, AnalyticsRangeView, AnalyticsSessionBrief, AppConfig
} from "../shared/types";
import { ask } from "../lib/rpc";
import { AdminSectionHeader } from "./AdminViews";

const CHART_COLORS = [
  "var(--chart-1)", "var(--chart-2)", "var(--chart-3)",
  "var(--chart-4)", "var(--chart-5)", "var(--chart-6)"
];
const SEQ_COLORS = [
  "var(--chart-seq-1)", "var(--chart-seq-2)", "var(--chart-seq-3)",
  "var(--chart-seq-4)", "var(--chart-seq-5)", "var(--chart-seq-6)"
];

// Recharts 的数值 Y 轴默认会预留约 60px，短刻度图因此显得整体偏右。后台卡片已经
// 自带 14px 内边距，普通数值轴统一压到 40px，并只在右侧保留防止端点/圆角被裁切的
// 6px 安全区；类别文字轴、双 Y 轴与倾斜标签仍在各图中按实际内容单独设置。
const NUMERIC_Y_AXIS_WIDTH = 40;
const CARTESIAN_MARGIN = { top: 8, right: 6, left: 0, bottom: 0 };

// data.gameRounds / data.roomCreates 的 series key 是后端存的 gameId（"unknown" 为
// game_id 为空的历史脏数据），图例/表头需要中文名，不能直接展示英文标识符。
const GAME_LABELS: Record<string, string> = {
  rps: "猜拳", othello: "黑白棋", tictactoe: "井字棋", gomoku: "五子棋",
  jungle: "斗兽棋", chess: "国际象棋", liarsdice: "大话骰", unknown: "未知游戏"
};

// data.profileChanges / data.nameWarGiveaway 的 series key 是 player_activity_events.action
// （logPlayerActivity 的第一个参数，见 internal/server/handlers.go），同样是英文标识符。
const ACTIVITY_LABELS: Record<string, string> = {
  rename: "改名", gender_change: "性别/阵营变更",
  self_title_change: "自定义称号", avatar_change: "更换头像",
  nameWar_enable: "开启名争", giveaway_enable: "开启白给",
  giveaway_board_submit: "白给留言板", nameWar_rename: "名争改名",
  extreme_enable: "开启极限"
};

function labelFor(labels: Record<string, string> | undefined, key: string): string {
  return labels?.[key] || key;
}

// data.devices 的 key 是后端 parseUserAgent 粗分类出的 deviceType（mobile/tablet/desktop，
// 见 internal/server/useragent.go），面板展示需要中文。浏览器/操作系统/来源/省份/ISP 等
// 其它 HBarCard 用途的 key 本就该原样展示（浏览器名、系统名等），不套这张表。
const DEVICE_LABELS: Record<string, string> = { mobile: "手机", tablet: "平板", desktop: "电脑" };

function relabelBuckets(rows: AnalyticsBucket[], labels: Record<string, string>): AnalyticsBucket[] {
  return rows.map((r) => ({ ...r, key: labelFor(labels, r.key) }));
}

// data.punishTagInclude / data.punishTagExclude 的 key 是标签 ID（config.punishmentTags[].id），
// 建房面板「随机惩罚」图需要把选中/排除两组按同一标签合并成一行，而不是分两张图各显示一半。
type TagCompareRow = { key: string; include: number; exclude: number };
function mergeTagCompareBuckets(
  include: AnalyticsBucket[], exclude: AnalyticsBucket[], labels: Record<string, string>
): TagCompareRow[] {
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

// data.viewPv 的 key 是前端路由/弹窗名（见 web/src/App.tsx 的 trackPageview 调用），
// 同样是英文标识符，"明细·页面浏览" 表格需要中文页面名。
const VIEW_LABELS: Record<string, string> = {
  login: "登录页", lobby: "大厅", room: "房间", admin: "后台管理",
  profile: "个人资料", leaderboard: "排行榜", about: "关于", help: "游戏说明"
};

function formatNum(n: number): string {
  if (!Number.isFinite(n)) return "—";
  if (Math.abs(n) >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (Math.abs(n) >= 10_000) return `${(n / 1000).toFixed(1)}k`;
  return String(Math.round(n));
}

function formatDurationMs(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(0)}s`;
  if (ms < 3_600_000) return `${(ms / 60_000).toFixed(1)}m`;
  return `${(ms / 3_600_000).toFixed(1)}h`;
}

function formatDelta(d: number): string {
  if (!Number.isFinite(d) || d === 0) return "持平";
  const pct = Math.round(d * 100);
  return pct > 0 ? `↑${pct}%` : `↓${Math.abs(pct)}%`;
}

// 图表横轴的日期刻度省略年份只显示 mm/dd；气泡提示（AnalyticsTooltip 的 label）与
// 明细表格仍展示完整 yyyy-mm-dd，不套用这个格式化。
function formatDayTick(v: string): string {
  const m = /^\d{4}-(\d{2})-(\d{2})$/.exec(v);
  return m ? `${m[1]}/${m[2]}` : v;
}

function AnalyticsTooltip({ active, payload, label, showTotal, showPercent, percentTotal, formatValue }: {
  active?: boolean;
  payload?: Array<{ name?: string; value?: number; color?: string }>;
  label?: string;
  /** 堆叠图（对局数/开房数/对局时长）悬停时附带当日各系列之和 */
  showTotal?: boolean;
  /** 每行附带占比：分母默认取当前悬停项各系列之和，也可用 percentTotal 指定外部分母
   * （如饼图/单系列柱状图，需要占「全部类目合计」而非「当前这一根柱子」的比例）。 */
  showPercent?: boolean;
  percentTotal?: number;
  /** 数值格式化，默认 formatNum（四舍五入取整）；均值类图表（如「对局时长」的分钟数，
   * 常见 <1 的小数）需要保留小数位，改传自定义格式化函数。 */
  formatValue?: (n: number) => string;
}) {
  if (!active || !payload?.length) return null;
  const fmt = formatValue || formatNum;
  const localTotal = payload.reduce((sum, p) => sum + (Number(p.value) || 0), 0);
  const percentDenom = percentTotal != null ? percentTotal : localTotal;
  return (
    <div className="analytics-tooltip">
      {label != null && label !== "" && <div className="analytics-tooltip-label">{label}</div>}
      {payload.map((p, i) => {
        const val = Number(p.value) || 0;
        const pct = showPercent && percentDenom > 0 ? `${((val / percentDenom) * 100).toFixed(1)}%` : null;
        return (
          <div key={i} className="analytics-tooltip-row">
            <span className="analytics-tooltip-swatch" style={{ background: p.color }} />
            <span>{p.name}</span>
            <strong>
              {fmt(val)}
              {pct != null && <span className="analytics-tooltip-pct"> ({pct})</span>}
            </strong>
          </div>
        );
      })}
      {showTotal && (
        <div className="analytics-tooltip-row analytics-tooltip-total">
          <span className="analytics-tooltip-swatch analytics-tooltip-swatch-total" />
          <span>总计</span>
          <strong>{fmt(localTotal)}</strong>
        </div>
      )}
    </div>
  );
}

/**
 * 堆叠柱状图分段：真正露在柱子顶端的那一段带圆角，其余分段间留 2px 间隙。
 * 每天各 series 的取值不同，"谁在顶端"必须按当前这一根柱子（payload）动态判断——
 * 不能固定为 seriesKeys 里声明顺序的最后一个，否则该 series 当天为 0 时，
 * 实际露出在顶端的前一个 series 会缺角（矩形直角顶在圆角柱堆里很显眼）。
 */
function StackedSegment(props: {
  x?: number; y?: number; width?: number; height?: number; fill?: string;
  payload?: Record<string, string | number>; dataKey?: string; seriesKeys?: string[];
}) {
  const { x = 0, y = 0, width = 0, height = 0, fill, payload, dataKey, seriesKeys = [] } = props;
  if (width <= 0 || height <= 0) return null;
  let topKey: string | undefined;
  for (const k of seriesKeys) {
    if (Number(payload?.[k] || 0) > 0) topKey = k;
  }
  if (dataKey === topKey) {
    return <Rectangle x={x} y={y} width={width} height={height} fill={fill} radius={[4, 4, 0, 0]} />;
  }
  const h = Math.max(0, height - 2);
  const yy = y + 2;
  if (h <= 0) return null;
  return <rect x={x} y={yy} width={width} height={h} fill={fill} rx={0} />;
}

function Sparkline({ data, color = "var(--chart-1)" }: { data: number[]; color?: string }) {
  if (!data.length) return <div className="analytics-spark" />;
  const rows = data.map((v, i) => ({ i, v }));
  return (
    <div className="analytics-spark">
      <ResponsiveContainer width="100%" height={36}>
        <LineChart data={rows} margin={{ top: 4, right: 0, left: 0, bottom: 0 }}>
          <Line type="monotone" dataKey="v" stroke={color} strokeWidth={2} dot={false}
            isAnimationActive={false} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}

function StatTile({
  label, value, delta, spark, format = formatNum
}: {
  label: string; value: number; delta: number; spark: number[]; format?: (n: number) => string;
}) {
  const good = delta > 0;
  const bad = delta < 0;
  return (
    <div className="analytics-stat">
      <span className="analytics-stat-label">{label}</span>
      <strong className="analytics-stat-value">{format(value)}</strong>
      <span className={`analytics-stat-delta${good ? " good" : ""}${bad ? " bad" : ""}`}>
        {formatDelta(delta)}
      </span>
      <Sparkline data={spark} color={good ? "var(--chart-good)" : bad ? "var(--chart-critical)" : "var(--chart-1)"} />
    </div>
  );
}

function ChartCard({
  title, children, table
}: {
  title: string; children: ReactNode; table?: ReactNode;
}) {
  const [asTable, setAsTable] = useState(false);
  return (
    <div className="analytics-card">
      <div className="analytics-card-head">
        <h3>{title}</h3>
        {table != null && (
          <button type="button" className="analytics-table-toggle" onClick={() => setAsTable((v) => !v)}>
            {asTable ? "图表" : "表格"}
          </button>
        )}
      </div>
      {asTable && table != null ? table : children}
    </div>
  );
}

function BucketTable({ rows, valueLabel = "数量" }: { rows: AnalyticsBucket[]; valueLabel?: string }) {
  if (!rows.length) return <p className="empty">暂无数据</p>;
  return (
    <table className="analytics-data-table">
      <thead><tr><th>项</th><th>{valueLabel}</th></tr></thead>
      <tbody>
        {rows.map((r) => (
          <tr key={r.key}><td>{r.key || "(空)"}</td><td>{formatNum(r.value)}</td></tr>
        ))}
      </tbody>
    </table>
  );
}

function SeriesTable({ days, series, labels }: {
  days: string[]; series: AnalyticsNamedSeries[]; labels?: Record<string, string>;
}) {
  if (!series.length) return <p className="empty">暂无数据</p>;
  return (
    <div className="analytics-table-scroll">
      <table className="analytics-data-table">
        <thead>
          <tr>
            <th>日期</th>
            {series.map((s) => <th key={s.key}>{labelFor(labels, s.key)}</th>)}
          </tr>
        </thead>
        <tbody>
          {days.map((d, i) => (
            <tr key={d}>
              <td>{d}</td>
              {series.map((s) => <td key={s.key}>{formatNum(s.values[i] || 0)}</td>)}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function RetentionHeatmap({ matrix, cohorts, offsets }: {
  matrix: number[][]; cohorts: string[]; offsets: number[];
}) {
  if (!matrix.length) return <p className="empty">暂无留存数据（需有成功注册的用户）</p>;
  // 只展示最近 14 个 cohort × offset 0-14，避免 30×30 过密
  const maxC = 14;
  const maxO = 15;
  const start = Math.max(0, cohorts.length - maxC);
  const cols = offsets.slice(0, maxO);
  const rows = cohorts.slice(start);
  const data = matrix.slice(start).map((r) => r.slice(0, maxO));
  return (
    <div className="analytics-heatmap-wrap">
      <div
        className="analytics-heatmap"
        style={{
          gridTemplateColumns: `72px repeat(${cols.length}, minmax(18px, 1fr))`
        } as CSSProperties}
      >
        <div className="analytics-heatmap-corner" />
        {cols.map((o) => (
          <div key={o} className="analytics-heatmap-colhead">D{o}</div>
        ))}
        {rows.map((c, ri) => (
          <Fragment key={c}>
            <div className="analytics-heatmap-rowhead">{c.slice(5)}</div>
            {cols.map((o, ci) => {
              const v = data[ri]?.[ci] ?? 0;
              const level = v <= 0 ? 0 : Math.min(6, Math.ceil(v / 16.7));
              return (
                <div
                  key={`${c}-${o}`}
                  className={`analytics-heatmap-cell seq-${level}`}
                  title={`${c} · D${o}: ${v.toFixed(1)}%`}
                >
                  {v > 0 ? Math.round(v) : ""}
                </div>
              );
            })}
          </Fragment>
        ))}
      </div>
    </div>
  );
}

function seriesToChartRows(days: string[], series: AnalyticsNamedSeries[]) {
  return days.map((day, i) => {
    const row: Record<string, string | number> = { day };
    for (const s of series) row[s.key] = s.values[i] || 0;
    return row;
  });
}

function trendRows(data: AnalyticsRangeView) {
  return data.series.days.map((day, i) => ({
    day,
    dau: data.series.dau[i] || 0,
    sessions: data.series.sessions[i] || 0,
    loggedDau: data.series.loggedDau[i] || 0,
    pageviews: data.series.pageviews[i] || 0,
    // 设备指纹维度（新老设备）
    newVisitors: data.series.newVisitors[i] || 0,
    returning: data.series.returning[i] || 0,
    // 用户 playerId 维度（新老用户）
    newUsers: data.newOldUsers?.new?.[i] || 0,
    oldLogin: data.newOldUsers?.oldLogin?.[i] || 0
  }));
}

export function AnalyticsPanel({ onError, config }: { onError: (message: string) => void; config: AppConfig }) {
  const [days, setDays] = useState(30);
  const [data, setData] = useState<AnalyticsRangeView | null>(null);
  const [sessions, setSessions] = useState<AnalyticsSessionBrief[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailTab, setDetailTab] = useState<"views" | "refs" | "prov" | "sess">("views");
  const genRef = useRef(0);
  // load() 被 60s 定时器长期持有；用 ref 而不是直接闭包 state，确保每次 tick 读到的都是
  // 当前的 tab / 是否已加载过明细，而不是 setInterval 注册那一刻（上次 days 变化时）的旧值——
  // 否则切到「最近会话」tab 后台自动刷新可能读到过期的 detailTab/sessions 判断条件。
  const detailTabRef = useRef(detailTab);
  const sessionsLoadedRef = useRef(false);
  useEffect(() => {
    detailTabRef.current = detailTab;
  }, [detailTab]);

  async function load(opts: { refresh?: boolean } = {}) {
    const gen = ++genRef.current;
    setLoading(true);
    try {
      const snap = await ask<AnalyticsRangeView>("admin:analytics", {
        days,
        refresh: !!opts.refresh
      });
      if (gen !== genRef.current) return;
      setData(snap);
      if (detailTabRef.current === "sess" || !sessionsLoadedRef.current) {
        try {
          const detail = await ask<{ recentSessions?: AnalyticsSessionBrief[] }>("admin:analyticsDetail", { days });
          if (gen === genRef.current) {
            sessionsLoadedRef.current = true;
            setSessions(detail.recentSessions || []);
          }
        } catch {
          /* detail 可选 */
        }
      }
    } catch (error) {
      if (gen !== genRef.current) return;
      onError(error instanceof Error ? error.message : "加载统计失败");
    } finally {
      if (gen === genRef.current) setLoading(false);
    }
  }

  useEffect(() => {
    void load();
    const t = setInterval(() => void load(), 60_000);
    return () => clearInterval(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [days]);

  // 首次切到“最近会话”时立即拉取明细；仅依赖主快照轮询会让该标签在
  // 最长 60 秒内一直显示旧的空状态。
  useEffect(() => {
    if (detailTab !== "sess") return;
    let cancelled = false;
    void ask<{ recentSessions?: AnalyticsSessionBrief[] }>("admin:analyticsDetail", { days })
      .then((detail) => {
        if (!cancelled) {
          sessionsLoadedRef.current = true;
          setSessions(detail.recentSessions || []);
        }
      })
      .catch(() => {
        /* 主面板数据仍可用，明细失败时保留旧值 */
      });
    return () => { cancelled = true; };
  }, [detailTab, days]);

  const hasData = !!data;
  const opacity = loading && hasData ? 0.6 : 1;
  const kpi = data?.kpi;
  const trends = data ? trendRows(data) : [];

  // punishTagInclude/Exclude 与 punishSeriesSelect 的 key 是内部 ID，需要按当前后台配置
  // 里的标签/系列名单翻译成中文名，查不到（已删除的标签/系列）时保底显示原始 ID。
  const tagLabels = useMemo(() => {
    const map: Record<string, string> = {};
    for (const t of config.punishmentTags || []) map[t.id] = t.name || t.id;
    return map;
  }, [config.punishmentTags]);
  const seriesLabels = useMemo(() => {
    const map: Record<string, string> = {};
    for (const s of config.punishmentSeriesSummaries || []) map[s.id] = s.name || s.id;
    return map;
  }, [config.punishmentSeriesSummaries]);
  const tagCompareRows = data ? mergeTagCompareBuckets(data.punishTagInclude || [], data.punishTagExclude || [], tagLabels) : [];
  // 会话时长分布 / 证明耗时分布悬停气泡的百分比分母：各自所有桶的合计，而不是当前这一根柱子。
  const sessionBucketsTotal = (data?.sessionBuckets || []).reduce((s, r) => s + r.value, 0);
  const proofMsTotal = (data?.punishment?.proofMs || []).reduce((s, r) => s + r.value, 0);

  return (
    <div className="analytics-panel">
      <AdminSectionHeader
        title="数据分析"
        subtitle="访问留存、设备渠道、游戏与站点玩法。数据来自日聚合快照，约每分钟刷新；不会拖慢游戏主链路。"
      />
      <div className="analytics-filters">
        {[7, 30, 90].map((d) => (
          <button key={d} type="button" className={days === d ? "active" : ""} onClick={() => setDays(d)}>
            近 {d} 天
          </button>
        ))}
        <button type="button" onClick={() => void load({ refresh: true })} disabled={loading}>
          刷新
        </button>
        {data?.builtAt ? (
          <span className="analytics-updated">
            最后更新 {new Date(data.builtAt).toLocaleTimeString()}
            {data.liveOnline != null && ` · 当前在线 ${data.liveOnline}`}
          </span>
        ) : null}
      </div>

      {!data && loading && <div className="analytics-loading">正在加载图表…</div>}
      {!data && !loading && <p className="empty">统计数据尚未就绪，请稍后再试</p>}

      {data && (
        <div className="analytics-grid" style={{ opacity }}>
          <div className="analytics-stats-row">
            <StatTile label="日活 (均)" value={kpi?.dauValue || 0} delta={kpi?.deltaDau || 0} spark={kpi?.sparkDau || []} />
            <StatTile label="会话数" value={kpi?.sessions || 0} delta={kpi?.deltaSessions || 0} spark={kpi?.sparkSessions || []} />
            <StatTile label="页面浏览" value={kpi?.pageviews || 0} delta={kpi?.deltaPageviews || 0} spark={kpi?.sparkPageviews || []} />
            <StatTile label="新访客" value={kpi?.newVisitors || 0} delta={kpi?.deltaNewVisitors || 0} spark={kpi?.sparkNewVisitors || []} />
            <StatTile label="平均会话" value={kpi?.avgSessionMs || 0} delta={kpi?.deltaAvgSessionMs || 0}
              spark={kpi?.sparkAvgSessionMs || []} format={formatDurationMs} />
            <StatTile label="峰值在线" value={kpi?.peakOnline || 0} delta={kpi?.deltaPeakOnline || 0} spark={kpi?.sparkPeakOnline || []} />
          </div>

          <div className="analytics-row-2">
            <ChartCard title="用户与会话" table={
              <table className="analytics-data-table">
                <thead><tr><th>日期</th><th>日活</th><th>会话</th><th>登录用户</th></tr></thead>
                <tbody>
                  {trends.map((r) => (
                    <tr key={r.day}><td>{r.day}</td><td>{r.dau}</td><td>{r.sessions}</td><td>{r.loggedDau}</td></tr>
                  ))}
                </tbody>
              </table>
            }>
              <ResponsiveContainer width="100%" height={240}>
                <LineChart data={trends} margin={CARTESIAN_MARGIN}>
                  <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
                  <XAxis dataKey="day" tickFormatter={formatDayTick} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <YAxis width={NUMERIC_Y_AXIS_WIDTH} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <Tooltip content={<AnalyticsTooltip />} />
                  <Legend />
                  <Line type="monotone" dataKey="dau" name="日活" stroke="var(--chart-1)" strokeWidth={2} dot={false}
                    activeDot={{ r: 4, strokeWidth: 2, stroke: "var(--chart-surface)" }} isAnimationActive={false} />
                  <Line type="monotone" dataKey="sessions" name="会话" stroke="var(--chart-2)" strokeWidth={2} dot={false}
                    activeDot={{ r: 4, strokeWidth: 2, stroke: "var(--chart-surface)" }} isAnimationActive={false} />
                  <Line type="monotone" dataKey="loggedDau" name="登录用户" stroke="var(--chart-3)" strokeWidth={2} dot={false}
                    activeDot={{ r: 4, strokeWidth: 2, stroke: "var(--chart-surface)" }} isAnimationActive={false} />
                </LineChart>
              </ResponsiveContainer>
            </ChartCard>

            <ChartCard title="页面浏览量" table={
              <table className="analytics-data-table">
                <thead><tr><th>日期</th><th>浏览量</th></tr></thead>
                <tbody>{trends.map((r) => <tr key={r.day}><td>{r.day}</td><td>{r.pageviews}</td></tr>)}</tbody>
              </table>
            }>
              <ResponsiveContainer width="100%" height={240}>
                <LineChart data={trends} margin={CARTESIAN_MARGIN}>
                  <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
                  <XAxis dataKey="day" tickFormatter={formatDayTick} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <YAxis width={NUMERIC_Y_AXIS_WIDTH} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <Tooltip content={<AnalyticsTooltip />} />
                  <Line type="monotone" dataKey="pageviews" name="页面浏览" stroke="var(--chart-1)" strokeWidth={2} dot={false}
                    activeDot={{ r: 4, strokeWidth: 2, stroke: "var(--chart-surface)" }} isAnimationActive={false} />
                </LineChart>
              </ResponsiveContainer>
            </ChartCard>
          </div>

          <div className="analytics-row-2">
            <ChartCard title="新老用户" table={
              <table className="analytics-data-table">
                <thead><tr><th>日期</th><th>新用户</th><th>老用户登录</th></tr></thead>
                <tbody>
                  {trends.map((r) => (
                    <tr key={r.day}><td>{r.day}</td><td>{r.newUsers}</td><td>{r.oldLogin}</td></tr>
                  ))}
                </tbody>
              </table>
            }>
              <ResponsiveContainer width="100%" height={220}>
                <LineChart data={trends} margin={CARTESIAN_MARGIN}>
                  <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
                  <XAxis dataKey="day" tickFormatter={formatDayTick} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <YAxis width={NUMERIC_Y_AXIS_WIDTH} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <Tooltip content={<AnalyticsTooltip showPercent />} />
                  <Legend />
                  <Line type="monotone" dataKey="newUsers" name="新用户" stroke="var(--chart-1)" strokeWidth={2} dot={false}
                    activeDot={{ r: 4, strokeWidth: 2, stroke: "var(--chart-surface)" }} isAnimationActive={false} />
                  <Line type="monotone" dataKey="oldLogin" name="老用户登录" stroke="var(--chart-3)" strokeWidth={2} dot={false}
                    activeDot={{ r: 4, strokeWidth: 2, stroke: "var(--chart-surface)" }} isAnimationActive={false} />
                </LineChart>
              </ResponsiveContainer>
            </ChartCard>

            <ChartCard title="新老设备" table={
              <table className="analytics-data-table">
                <thead><tr><th>日期</th><th>新设备</th><th>老设备</th></tr></thead>
                <tbody>
                  {trends.map((r) => (
                    <tr key={r.day}><td>{r.day}</td><td>{r.newVisitors}</td><td>{r.returning}</td></tr>
                  ))}
                </tbody>
              </table>
            }>
              <ResponsiveContainer width="100%" height={220}>
                <LineChart data={trends} margin={CARTESIAN_MARGIN}>
                  <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
                  <XAxis dataKey="day" tickFormatter={formatDayTick} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <YAxis width={NUMERIC_Y_AXIS_WIDTH} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <Tooltip content={<AnalyticsTooltip showPercent />} />
                  <Legend />
                  <Line type="monotone" dataKey="newVisitors" name="新设备" stroke="var(--chart-1)" strokeWidth={2} dot={false}
                    activeDot={{ r: 4, strokeWidth: 2, stroke: "var(--chart-surface)" }} isAnimationActive={false} />
                  <Line type="monotone" dataKey="returning" name="老设备" stroke="var(--chart-3)" strokeWidth={2} dot={false}
                    activeDot={{ r: 4, strokeWidth: 2, stroke: "var(--chart-surface)" }} isAnimationActive={false} />
                </LineChart>
              </ResponsiveContainer>
            </ChartCard>
          </div>

          <ChartCard title="用户留存热力图 (D0–D14)">
            <RetentionHeatmap
              matrix={data.retention?.matrix || []}
              cohorts={data.retention?.cohorts || []}
              offsets={data.retention?.offsets || []}
            />
          </ChartCard>

          <div className="analytics-row-3">
            <HBarCard title="设备类型" rows={relabelBuckets(data.devices || [], DEVICE_LABELS)} donut />
            <HBarCard title="浏览器" rows={data.browsers || []} limit={6} showPercent />
            <HBarCard title="操作系统" rows={data.os || []} limit={6} showPercent />
          </div>

          <div className="analytics-row-3">
            <HBarCard title="来源 Top10" rows={data.referrers || []} showPercent />
            <HBarCard title="省份 Top10" rows={data.provinces || []} showPercent />
            <HBarCard title="ISP Top10" rows={data.isps || []} showPercent />
          </div>

          <div className="analytics-row-2">
            <ChartCard
              title="对局时长"
              table={
                <RoundDurationTable
                  days={data.series.days}
                  gameSeries={data.gameRoundAvgMinutes || []}
                  roomAvg={data.roomAvgMinutes || []}
                  labels={GAME_LABELS}
                />
              }
            >
              <RoundDurationChart
                days={data.series.days}
                gameSeries={data.gameRoundAvgMinutes || []}
                roomAvg={data.roomAvgMinutes || []}
                labels={GAME_LABELS}
              />
            </ChartCard>
            <ChartCard title="惩罚任务" table={
              <table className="analytics-data-table">
                <thead><tr><th>日期</th><th>发布</th><th>完成</th><th>驳回</th></tr></thead>
                <tbody>
                  {data.series.days.map((d, i) => (
                    <tr key={d}>
                      <td>{d}</td>
                      <td>{data.punishment?.publish?.[i] || 0}</td>
                      <td>{data.punishment?.done?.[i] || 0}</td>
                      <td>{data.punishment?.reject?.[i] || 0}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            }>
              <p className="analytics-inline-kpi">
                完成率 <strong>{((data.punishment?.doneRate || 0) * 100).toFixed(1)}%</strong>
              </p>
              <ResponsiveContainer width="100%" height={236}>
                <BarChart data={data.series.days.map((day, i) => ({
                  day,
                  pending: Math.max(0,
                    (data.punishment?.publish?.[i] || 0)
                    - (data.punishment?.done?.[i] || 0)
                    - (data.punishment?.reject?.[i] || 0)),
                  done: data.punishment?.done?.[i] || 0,
                  reject: data.punishment?.reject?.[i] || 0
                }))} margin={CARTESIAN_MARGIN}>
                  <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
                  <XAxis dataKey="day" tickFormatter={formatDayTick} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <YAxis width={NUMERIC_Y_AXIS_WIDTH} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <Tooltip content={<AnalyticsTooltip showPercent />} />
                  <Legend />
                  {/* 发布量包含 assigned/pending/approved/rejected。用“进行中 + 完成 + 驳回”
                      互斥堆叠，柱子总高度才与发布量一致；不能把发布再作为第四段重复相加。 */}
                  <Bar dataKey="pending" name="进行中" stackId="a" fill="var(--chart-1)" maxBarSize={24}
                    shape={(p: any) => <StackedSegment {...p} dataKey="pending" seriesKeys={["pending", "done", "reject"]} />}
                    isAnimationActive={false} />
                  <Bar dataKey="done" name="完成" stackId="a" fill="var(--chart-3)" maxBarSize={24}
                    shape={(p: any) => <StackedSegment {...p} dataKey="done" seriesKeys={["pending", "done", "reject"]} />}
                    isAnimationActive={false} />
                  <Bar dataKey="reject" name="驳回" stackId="a" fill="var(--chart-critical)" maxBarSize={24}
                    shape={(p: any) => <StackedSegment {...p} dataKey="reject" seriesKeys={["pending", "done", "reject"]} />}
                    isAnimationActive={false} />
                </BarChart>
              </ResponsiveContainer>
            </ChartCard>
          </div>

          <div className="analytics-row-2">
            <TagCompareChart rows={tagCompareRows} />
            <HBarCard
              title="系列惩罚·任务选中"
              rows={relabelBuckets(data.punishSeriesSelect || [], seriesLabels)}
            />
          </div>

          <div className="analytics-row-2">
            <ChartCard
              title="对局数"
              table={<SeriesTable days={data.series.days} series={data.gameRounds || []} labels={GAME_LABELS} />}
            >
              <StackedGameChart days={data.series.days} series={data.gameRounds || []} labels={GAME_LABELS} showPercent />
            </ChartCard>
            <ChartCard title="开房数" table={<SeriesTable days={data.series.days} series={data.roomCreates || []} labels={GAME_LABELS} />}>
              <StackedGameChart days={data.series.days} series={data.roomCreates || []} labels={GAME_LABELS} showPercent />
            </ChartCard>
          </div>


          <div className="analytics-row-2">
            <ChartCard title="会话时长分布" table={<BucketTable rows={data.sessionBuckets || []} />}>
              <ResponsiveContainer width="100%" height={220}>
                <BarChart data={data.sessionBuckets || []} margin={CARTESIAN_MARGIN}>
                  <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
                  <XAxis dataKey="key" tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <YAxis width={NUMERIC_Y_AXIS_WIDTH} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <Tooltip content={<AnalyticsTooltip showPercent percentTotal={sessionBucketsTotal} />} />
                  <Bar dataKey="value" name="会话数" maxBarSize={24} radius={[4, 4, 0, 0]} isAnimationActive={false}>
                    {(data.sessionBuckets || []).map((_, i) => (
                      <Cell key={i} fill={SEQ_COLORS[i % SEQ_COLORS.length]} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </ChartCard>

            <ChartCard title="证明耗时分布" table={<BucketTable rows={data.punishment?.proofMs || []} />}>
              <ResponsiveContainer width="100%" height={220}>
                <BarChart data={data.punishment?.proofMs || []} margin={CARTESIAN_MARGIN}>
                  <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
                  <XAxis dataKey="key" tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <YAxis width={NUMERIC_Y_AXIS_WIDTH} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <Tooltip content={<AnalyticsTooltip showPercent percentTotal={proofMsTotal} />} />
                  <Bar dataKey="value" name="次数" maxBarSize={24} radius={[4, 4, 0, 0]} isAnimationActive={false}>
                    {(data.punishment?.proofMs || []).map((_, i) => (
                      <Cell key={i} fill={SEQ_COLORS[i % SEQ_COLORS.length]} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </ChartCard>
          </div>

          <div className="analytics-row-2">
            <ChartCard title="名争·白给" table={<SeriesTable days={data.series.days} series={data.nameWarGiveaway || []} labels={ACTIVITY_LABELS} />}>
              <StackedGameChart days={data.series.days} series={data.nameWarGiveaway || []} asLine labels={ACTIVITY_LABELS} />
            </ChartCard>
            <ChartCard title="主宠关系" table={
              <table className="analytics-data-table">
                <thead><tr><th>日期</th><th>总数</th><th>新增</th></tr></thead>
                <tbody>
                  {data.series.days.map((d, i) => (
                    <tr key={d}>
                      <td>{d}</td>
                      <td>{data.petBond?.total?.[i] || 0}</td>
                      <td>{data.petBond?.new?.[i] || 0}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            }>
              {/* 总数（存量）与新增（日增量）量级不同，双 Y 轴各自缩放。 */}
              <ResponsiveContainer width="100%" height={220}>
                <LineChart
                  data={data.series.days.map((day, i) => ({
                    day,
                    total: data.petBond?.total?.[i] || 0,
                    new: data.petBond?.new?.[i] || 0
                  }))}
                  margin={{ left: 4, right: 4 }}
                >
                  <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
                  <XAxis dataKey="day" tickFormatter={formatDayTick} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <YAxis
                    yAxisId="left"
                    tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }}
                    stroke="var(--chart-axis)"
                    width={40}
                  />
                  <YAxis
                    yAxisId="right"
                    orientation="right"
                    tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }}
                    stroke="var(--chart-axis)"
                    width={40}
                  />
                  <Tooltip content={<AnalyticsTooltip />} />
                  <Legend />
                  <Line yAxisId="left" type="monotone" dataKey="new" name="新增" stroke="var(--chart-2)" strokeWidth={2} dot={false} isAnimationActive={false} />
                  <Line yAxisId="right" type="monotone" dataKey="total" name="总数" stroke="var(--chart-1)" strokeWidth={2} dot={false} isAnimationActive={false} />
                </LineChart>
              </ResponsiveContainer>
            </ChartCard>
          </div>

          <div className="analytics-row-2">
            <ChartCard title="用户信息变更" table={<SeriesTable days={data.series.days} series={data.profileChanges || []} labels={ACTIVITY_LABELS} />}>
              <StackedGameChart days={data.series.days} series={data.profileChanges || []} asLine labels={ACTIVITY_LABELS} />
            </ChartCard>

            <ChartCard title="聊天活跃" table={
              <table className="analytics-data-table">
                <thead><tr><th>日期</th><th>大厅</th><th>房间</th></tr></thead>
                <tbody>
                  {data.series.days.map((d, i) => (
                    <tr key={d}>
                      <td>{d}</td>
                      <td>{data.chat?.lobby?.[i] || 0}</td>
                      <td>{data.chat?.room?.[i] || 0}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            }>
              {/* 大厅/房间消息量级常差 2～3 个数量级，双 Y 轴各自缩放。 */}
              <ResponsiveContainer width="100%" height={220}>
                <LineChart
                  data={data.series.days.map((day, i) => ({
                    day,
                    lobby: data.chat?.lobby?.[i] || 0,
                    room: data.chat?.room?.[i] || 0
                  }))}
                  margin={{ left: 4, right: 4 }}
                >
                  <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
                  <XAxis dataKey="day" tickFormatter={formatDayTick} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <YAxis
                    yAxisId="left"
                    tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }}
                    stroke="var(--chart-axis)"
                    width={40}
                  />
                  <YAxis
                    yAxisId="right"
                    orientation="right"
                    tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }}
                    stroke="var(--chart-axis)"
                    width={40}
                  />
                  <Tooltip content={<AnalyticsTooltip />} />
                  <Legend />
                  <Line yAxisId="left" type="monotone" dataKey="lobby" name="大厅消息" stroke="var(--chart-1)" strokeWidth={2} dot={false} isAnimationActive={false} />
                  <Line yAxisId="right" type="monotone" dataKey="room" name="房间消息" stroke="var(--chart-2)" strokeWidth={2} dot={false} isAnimationActive={false} />
                </LineChart>
              </ResponsiveContainer>
            </ChartCard>
          </div>

          <div className="analytics-row-2">
            <ChartCard title="单房对局" table={
              <table className="analytics-data-table">
                <thead><tr><th>日期</th><th>最多</th><th>平均</th></tr></thead>
                <tbody>
                  {data.series.days.map((d, i) => (
                    <tr key={d}>
                      <td>{d}</td>
                      <td>{data.roomRounds?.max?.[i] || 0}</td>
                      <td>{data.roomRounds?.avg?.[i] || 0}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            }>
              {/* max 与 avg 量级可能差较大，双 Y 轴各自缩放。 */}
              <ResponsiveContainer width="100%" height={220}>
                <LineChart
                  data={data.series.days.map((day, i) => ({
                    day,
                    max: data.roomRounds?.max?.[i] || 0,
                    avg: data.roomRounds?.avg?.[i] || 0
                  }))}
                  margin={{ left: 4, right: 4 }}
                >
                  <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
                  <XAxis dataKey="day" tickFormatter={formatDayTick} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <YAxis
                    yAxisId="left"
                    tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }}
                    stroke="var(--chart-axis)"
                    width={40}
                  />
                  <YAxis
                    yAxisId="right"
                    orientation="right"
                    tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }}
                    stroke="var(--chart-axis)"
                    width={40}
                  />
                  <Tooltip content={<AnalyticsTooltip />} />
                  <Legend />
                  <Line yAxisId="left" type="monotone" dataKey="max" name="单房最多" stroke="var(--chart-1)" strokeWidth={2} dot={false} isAnimationActive={false} />
                  <Line yAxisId="right" type="monotone" dataKey="avg" name="单房平均" stroke="var(--chart-2)" strokeWidth={2} dot={false} isAnimationActive={false} />
                </LineChart>
              </ResponsiveContainer>
            </ChartCard>

            <ChartCard title="聊天人数" table={
              <table className="analytics-data-table">
                <thead><tr><th>日期</th><th>大厅</th><th>房间</th></tr></thead>
                <tbody>
                  {data.series.days.map((d, i) => (
                    <tr key={d}>
                      <td>{d}</td>
                      <td>{data.chat?.speakers?.[i] || 0}</td>
                      <td>{data.chat?.speakersRoom?.[i] || 0}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            }>
              <ResponsiveContainer width="100%" height={220}>
                <LineChart data={data.series.days.map((day, i) => ({
                  day,
                  lobby: data.chat?.speakers?.[i] || 0,
                  room: data.chat?.speakersRoom?.[i] || 0
                }))} margin={CARTESIAN_MARGIN}>
                  <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
                  <XAxis dataKey="day" tickFormatter={formatDayTick} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <YAxis width={NUMERIC_Y_AXIS_WIDTH} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <Tooltip content={<AnalyticsTooltip />} />
                  <Legend />
                  <Line type="monotone" dataKey="lobby" name="大厅发言人" stroke="var(--chart-1)" strokeWidth={2} dot={false} isAnimationActive={false} />
                  <Line type="monotone" dataKey="room" name="房间发言人" stroke="var(--chart-2)" strokeWidth={2} dot={false} isAnimationActive={false} />
                </LineChart>
              </ResponsiveContainer>
            </ChartCard>
          </div>

          <ChartCard title="转化漏斗" table={<BucketTable rows={funnelOrdered(data.funnel || [])} />}>
            <ResponsiveContainer width="100%" height={200}>
              <BarChart layout="vertical" data={funnelOrdered(data.funnel || [])} margin={CARTESIAN_MARGIN}>
                <CartesianGrid stroke="var(--chart-grid)" horizontal={false} strokeDasharray="" />
                <XAxis type="number" tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                <YAxis type="category" dataKey="key" width={80} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                <Tooltip content={<FunnelTooltip steps={funnelOrdered(data.funnel || [])} />} />
                <Bar dataKey="value" name="设备 UV" maxBarSize={24} radius={[0, 4, 4, 0]} isAnimationActive={false}>
                  {funnelOrdered(data.funnel || []).map((_, i) => (
                    <Cell key={i} fill={SEQ_COLORS[i % SEQ_COLORS.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </ChartCard>

          <div className="analytics-card">
            <div className="analytics-card-head">
              <h3>明细</h3>
              <div className="analytics-tabs">
                {([
                  ["views", "页面浏览"],
                  ["refs", "来源"],
                  ["prov", "省份"],
                  ["sess", "最近会话"]
                ] as const).map(([id, label]) => (
                  <button key={id} type="button" className={detailTab === id ? "active" : ""} onClick={() => setDetailTab(id)}>
                    {label}
                  </button>
                ))}
              </div>
            </div>
            {detailTab === "views" && <BucketTable rows={relabelBuckets((data.viewPv || []).slice(0, 20), VIEW_LABELS)} />}
            {detailTab === "refs" && <BucketTable rows={(data.referrers || []).slice(0, 20)} />}
            {detailTab === "prov" && <BucketTable rows={(data.provinces || []).slice(0, 20)} />}
            {detailTab === "sess" && (
              <table className="analytics-data-table">
                <thead>
                  <tr>
                    <th>访客</th><th>玩家</th><th>时长</th><th>设备</th><th>省份</th><th>浏览</th>
                  </tr>
                </thead>
                <tbody>
                  {(sessions.length ? sessions : []).map((s, i) => (
                    <tr key={i}>
                      <td>{s.visitor}</td>
                      <td>{s.playerName || "—"}</td>
                      <td>{formatDurationMs(s.durationMs)}</td>
                      <td>{labelFor(DEVICE_LABELS, s.device)}</td>
                      <td>{s.province || "—"}</td>
                      <td>{s.pageviews}</td>
                    </tr>
                  ))}
                  {!sessions.length && (
                    <tr><td colSpan={6} className="empty">暂无会话（前端埋点上线后产生）</td></tr>
                  )}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function funnelOrdered(rows: AnalyticsBucket[]): AnalyticsBucket[] {
  // 与 analytics_agg rebuildDay 的 funnel 键一致：五层均为设备 UV
  const order = ["visit", "lobby", "room", "round", "finish"];
  const labels: Record<string, string> = {
    visit: "访问", lobby: "进大厅", room: "进房", round: "开局", finish: "完成对局"
  };
  const map = new Map(rows.map((r) => [r.key, r.value]));
  return order.map((k, i) => ({ key: labels[k] || k, value: map.get(k) || 0, stepIndex: i } as AnalyticsBucket & { stepIndex: number }));
}

/** 转化漏斗悬停：绝对设备 UV + 相对上一级 / 相对访问 的转化率 */
function FunnelTooltip({ active, payload, steps }: {
  active?: boolean;
  payload?: Array<{ name?: string; value?: number; color?: string; payload?: { key?: string; value?: number; stepIndex?: number } }>;
  steps: Array<{ key: string; value: number }>;
}) {
  if (!active || !payload?.length) return null;
  const p = payload[0];
  const value = Number(p.value) || 0;
  const label = p.payload?.key || "";
  const idx = typeof p.payload?.stepIndex === "number"
    ? p.payload.stepIndex
    : steps.findIndex((s) => s.key === label);
  const top = steps[0]?.value || 0;
  const prev = idx > 0 ? (steps[idx - 1]?.value || 0) : 0;
  const fmtPct = (num: number, den: number) => {
    if (den <= 0) return "—";
    return `${((num / den) * 100).toFixed(1)}%`;
  };
  return (
    <div className="analytics-tooltip">
      {label !== "" && <div className="analytics-tooltip-label">{label}</div>}
      <div className="analytics-tooltip-row">
        <span className="analytics-tooltip-swatch" style={{ background: p.color }} />
        <span>设备 UV</span>
        <strong>{formatNum(value)}</strong>
      </div>
      {idx > 0 && (
        <div className="analytics-tooltip-row analytics-tooltip-meta">
          <span>相对上一级</span>
          <strong>{fmtPct(value, prev)}</strong>
        </div>
      )}
      <div className="analytics-tooltip-row analytics-tooltip-meta">
        <span>相对访问</span>
        <strong>{idx === 0 ? "100%" : fmtPct(value, top)}</strong>
      </div>
    </div>
  );
}

// 随机惩罚标签「选中 / 拒绝」对照图：横轴为标签（按选中+拒绝总量取 top10），
// 纵轴为该标签在当前时间范围内的选中/拒绝次数堆叠柱——柱子总高度=该标签的热度，
// 色块占比=选中/拒绝的构成，与「惩罚任务」发布/完成/驳回堆叠图是同一视觉语言。
function TagCompareChart({ rows }: { rows: TagCompareRow[] }) {
  const top = rows.slice(0, 10);
  const table = (
    <table className="analytics-data-table">
      <thead><tr><th>标签</th><th>选中</th><th>拒绝</th><th>合计</th></tr></thead>
      <tbody>
        {top.map((r) => (
          <tr key={r.key}>
            <td>{r.key}</td><td>{formatNum(r.include)}</td><td>{formatNum(r.exclude)}</td>
            <td>{formatNum(r.include + r.exclude)}</td>
          </tr>
        ))}
        {!top.length && <tr><td colSpan={4} className="empty">暂无数据</td></tr>}
      </tbody>
    </table>
  );
  return (
    <ChartCard title="随机惩罚·标签选中/拒绝" table={table}>
      {top.length ? (
        <ResponsiveContainer width="100%" height={260}>
          <BarChart data={top} margin={{ top: 8, right: 6, left: 0, bottom: 32 }}>
            <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
            <XAxis dataKey="key" interval={0} angle={-25} textAnchor="end" height={56}
              tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
            <YAxis width={NUMERIC_Y_AXIS_WIDTH} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
            <Tooltip content={<AnalyticsTooltip showTotal />} />
            <Legend />
            <Bar dataKey="include" name="选中" stackId="a" fill="var(--chart-1)" maxBarSize={40}
              shape={(p: any) => <StackedSegment {...p} dataKey="include" seriesKeys={["include", "exclude"]} />}
              isAnimationActive={false} />
            <Bar dataKey="exclude" name="拒绝" stackId="a" fill="var(--chart-critical)" maxBarSize={40}
              shape={(p: any) => <StackedSegment {...p} dataKey="exclude" seriesKeys={["include", "exclude"]} />}
              isAnimationActive={false} />
          </BarChart>
        </ResponsiveContainer>
      ) : (
        <p className="empty">暂无数据</p>
      )}
    </ChartCard>
  );
}

function HBarCard({ title, rows, donut, showPercent, limit }: {
  title: string; rows: AnalyticsBucket[]; donut?: boolean;
  /** 非圆环（横向条形）分支是否附带百分比；圆环分支恒有百分比，不受此开关影响。 */
  showPercent?: boolean;
  /** 展示的柱数上限，默认圆环 4、条形 10；调用方须传入未截断的完整 rows，
   * 截断交给本组件做，否则下面的 fullTotal 会算不出「Top N 之外还有多少」。 */
  limit?: number;
}) {
  const n = limit ?? (donut ? 4 : 10);
  const top = rows.slice(0, n);
  // 圆环图：中心「合计」与切片占比只针对实际画出来的切片，保证饼图视觉上刚好凑满 100%。
  const topTotal = top.reduce((s, r) => s + r.value, 0);
  // 条形图：占比分母是 rows 的完整合计（所选时间段全部类目，而不只是画出来的 Top N）——
  // 否则 Top N 之外还有数据时，分母偏小会让百分比显得比实际更高。
  const fullTotal = rows.reduce((s, r) => s + r.value, 0);
  return (
    <ChartCard title={title} table={<BucketTable rows={top} />}>
      {donut && top.length > 0 ? (
        <div className="analytics-donut-wrap">
          <ResponsiveContainer width="100%" height={200}>
            <PieChart>
              <Pie data={top} dataKey="value" nameKey="key" innerRadius="60%" outerRadius="85%" paddingAngle={2}
                isAnimationActive={false}>
                {top.map((_, i) => <Cell key={i} fill={CHART_COLORS[i % CHART_COLORS.length]} />)}
              </Pie>
              {/* wrapperStyle 显式给个 z-index：中心「合计」叠层（analytics-donut-center）在
                  DOM 顺序上排在 Tooltip 之后，默认层叠顺序会盖住鼠标悬停在中心附近时弹出的
                  气泡，导致看不到内容；提高 Tooltip 自身层级即可让它盖住中心叠层。 */}
              <Tooltip content={<AnalyticsTooltip showPercent percentTotal={topTotal} />} wrapperStyle={{ zIndex: 20 }} />
            </PieChart>
          </ResponsiveContainer>
          <div className="analytics-donut-center">
            <strong>{formatNum(topTotal)}</strong>
            <span>合计</span>
          </div>
        </div>
      ) : (
        <ResponsiveContainer width="100%" height={200}>
          <BarChart layout="vertical" data={top} margin={{ top: 8, right: 6, left: 0, bottom: 0 }}>
            <CartesianGrid stroke="var(--chart-grid)" horizontal={false} strokeDasharray="" />
            <XAxis type="number" tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
            <YAxis type="category" dataKey="key" width={72} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
            <Tooltip content={<AnalyticsTooltip showPercent={showPercent} percentTotal={fullTotal} />} />
            <Bar dataKey="value" name="数量" maxBarSize={24} radius={[0, 4, 4, 0]} fill="var(--chart-1)" isAnimationActive={false} />
          </BarChart>
        </ResponsiveContainer>
      )}
    </ChartCard>
  );
}

// 后端 namedSeries 按当前窗口总量降序排列（且总量并列时取决于 Go map 遍历顺序），
// series 数组顺序会在两次轮询之间悄悄变化。堆叠柱状图的 Bar 挂载顺序决定了
// Recharts 内部的堆叠层序，而 <Bar> 用 key={s.key} 做 keyed diff——顺序变化时
// React 只会挪动已有实例，不一定触发层序重新登记，导致"当前渲染顺序"和"实际堆叠层序"
// 出现错位，StackedSegment 找出的"顶层"就可能不是视觉上真正露出的那一段（圆角错位到中间）。
// 用 labels（游戏白名单）里固定的 key 顺序重排 series，让同一张图表在任意两次轮询之间
// 顺序恒定，从根上消除这种错位，顺带也让每个游戏的配色不会跟着轮询到的顺序漂移。
function orderSeriesStably(series: AnalyticsNamedSeries[], labels?: Record<string, string>): AnalyticsNamedSeries[] {
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

function StackedGameChart({
  days, series: rawSeries, asLine, labels, showPercent
}: {
  days: string[]; series: AnalyticsNamedSeries[]; asLine?: boolean; labels?: Record<string, string>;
  /** 仅堆叠柱状图分支生效：气泡内每个游戏附带其占当天（悬停这一根柱子）各游戏合计的百分比。 */
  showPercent?: boolean;
}) {
  const series = orderSeriesStably(rawSeries, labels);
  if (!series.length) return <p className="empty">暂无数据</p>;
  const rows = seriesToChartRows(days, series);
  if (asLine) {
    return (
      <ResponsiveContainer width="100%" height={220}>
        <LineChart data={rows} margin={CARTESIAN_MARGIN}>
          <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
          <XAxis dataKey="day" tickFormatter={formatDayTick} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
          <YAxis width={NUMERIC_Y_AXIS_WIDTH} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
          <Tooltip content={<AnalyticsTooltip />} />
          <Legend />
          {series.map((s, i) => (
            <Line key={s.key} type="monotone" dataKey={s.key} name={labelFor(labels, s.key)}
              stroke={CHART_COLORS[i % CHART_COLORS.length]} strokeWidth={2} dot={false} isAnimationActive={false} />
          ))}
        </LineChart>
      </ResponsiveContainer>
    );
  }
  // 对局数 / 开房数：气泡里附带当日各游戏之和；showPercent 时每个游戏数值旁再附带其占
  // 当日合计（即 AnalyticsTooltip 默认的 localTotal）的百分比，不外传 percentTotal——
  // 分母跟着悬停的那一天走，不是整个选定区间的合计。
  return (
    <ResponsiveContainer width="100%" height={240}>
      <BarChart data={rows} margin={CARTESIAN_MARGIN}>
        <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
        <XAxis dataKey="day" tickFormatter={formatDayTick} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
        <YAxis width={NUMERIC_Y_AXIS_WIDTH} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
        <Tooltip content={<AnalyticsTooltip showTotal showPercent={showPercent} />} />
        <Legend />
        {series.map((s, i) => (
          <Bar key={s.key} dataKey={s.key} name={labelFor(labels, s.key)} stackId="a"
            fill={CHART_COLORS[i % CHART_COLORS.length]} maxBarSize={24}
            shape={(p: any) => <StackedSegment {...p} dataKey={s.key} seriesKeys={series.map((x) => x.key)} />}
            isAnimationActive={false} />
        ))}
      </BarChart>
    </ResponsiveContainer>
  );
}

// 「对局时长」：左轴堆叠柱是各游戏当天的单局均值分钟数（data.gameRoundAvgMinutes，
// 总耗时/该游戏当天总局数），右轴折线是全站不拆分游戏的单房均值分钟数（data.roomAvgMinutes，
// 总房间存活时长/当天开房总数）——两者量级、口径都不同，共享 X 轴（日期）但各自独立缩放，
// 与「单房对局」图的左右双轴写法一致（见下方 max/avg 折线）。均值常见 <1 分钟的小数，
// 悬停气泡与表格都保留 1 位小数，不能套用 formatNum 的整数取整。
function RoundDurationChart({ days, gameSeries: rawSeries, roomAvg, labels }: {
  days: string[]; gameSeries: AnalyticsNamedSeries[]; roomAvg: number[]; labels?: Record<string, string>;
}) {
  const series = orderSeriesStably(rawSeries, labels);
  if (!series.length) return <p className="empty">暂无数据</p>;
  const rows = days.map((day, i) => {
    const row: Record<string, string | number> = { day, roomAvg: roomAvg[i] || 0 };
    for (const s of series) row[s.key] = s.values[i] || 0;
    return row;
  });
  const fmtMinutes = (n: number) => `${n.toFixed(1)} 分钟`;
  return (
    <ResponsiveContainer width="100%" height={260}>
      <ComposedChart data={rows}>
        <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
        <XAxis dataKey="day" tickFormatter={formatDayTick} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
        <YAxis yAxisId="left" tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" width={40} />
        <YAxis yAxisId="right" orientation="right" tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" width={40} />
        {/* 柱（各游戏单局均值）与折线（全站单房均值）单位不同、不能相加，故不传 showTotal/
            showPercent，避免 AnalyticsTooltip 默认把两者混算出一个没有意义的“总计”。 */}
        <Tooltip content={<AnalyticsTooltip formatValue={fmtMinutes} />} />
        <Legend />
        {series.map((s, i) => (
          <Bar key={s.key} yAxisId="left" dataKey={s.key} name={labelFor(labels, s.key)} stackId="a"
            fill={CHART_COLORS[i % CHART_COLORS.length]} maxBarSize={24}
            shape={(p: any) => <StackedSegment {...p} dataKey={s.key} seriesKeys={series.map((x) => x.key)} />}
            isAnimationActive={false} />
        ))}
        <Line yAxisId="right" type="monotone" dataKey="roomAvg" name="房间均值"
          stroke="var(--chart-critical)" strokeWidth={2} dot={false} isAnimationActive={false} />
      </ComposedChart>
    </ResponsiveContainer>
  );
}

function RoundDurationTable({ days, gameSeries: rawSeries, roomAvg, labels }: {
  days: string[]; gameSeries: AnalyticsNamedSeries[]; roomAvg: number[]; labels?: Record<string, string>;
}) {
  const series = orderSeriesStably(rawSeries, labels);
  if (!series.length) return <p className="empty">暂无数据</p>;
  return (
    <div className="analytics-table-scroll">
      <table className="analytics-data-table">
        <thead>
          <tr>
            <th>日期</th>
            {series.map((s) => <th key={s.key}>{labelFor(labels, s.key)}</th>)}
            <th>房间均值</th>
          </tr>
        </thead>
        <tbody>
          {days.map((d, i) => (
            <tr key={d}>
              <td>{d}</td>
              {series.map((s) => <td key={s.key}>{(s.values[i] || 0).toFixed(1)}</td>)}
              <td>{(roomAvg[i] || 0).toFixed(1)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
