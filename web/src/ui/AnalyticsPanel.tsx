import { Fragment, useEffect, useRef, useState, type CSSProperties, type ReactNode } from "react";
import {
  Bar, BarChart, CartesianGrid, Cell, ComposedChart, Legend, Line, LineChart,
  Pie, PieChart, ResponsiveContainer, Tooltip, XAxis, YAxis
} from "recharts";
import type {
  AnalyticsBucket, AnalyticsNamedSeries, AnalyticsRangeView, AnalyticsSessionBrief
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

// data.gameRounds / data.roomCreates 的 series key 是后端存的 gameId（"unknown" 为
// game_id 为空的历史脏数据），图例/表头需要中文名，不能直接展示英文标识符。
const GAME_LABELS: Record<string, string> = {
  rps: "猜拳", othello: "黑白棋", tictactoe: "井字棋", gomoku: "五子棋",
  jungle: "斗兽棋", liarsdice: "大话骰", unknown: "未知游戏"
};

// data.activity 的 series key 是 player_activity_events.action（logPlayerActivity 的
// 第一个参数，见 internal/server/handlers.go / http.go），同样是英文标识符。
const ACTIVITY_LABELS: Record<string, string> = {
  create: "创建角色", rename: "改名", gender_change: "性别/阵营变更",
  self_title_change: "自定义称号", giveaway_board_submit: "白给留言板",
  avatar_clear: "清除头像", avatar_change: "更换头像",
  nameWar_enable: "开启抢名", nameWar_disable: "关闭抢名",
  giveaway_enable: "开启白给", giveaway_disable: "关闭白给",
  extreme_enable: "开启极限模式", extreme_disable: "关闭极限模式"
};

function labelFor(labels: Record<string, string> | undefined, key: string): string {
  return labels?.[key] || key;
}

// data.devices 的 key 是后端 parseUserAgent 粗分类出的 deviceType（mobile/tablet/desktop，
// 见 internal/server/useragent.go），面板展示需要中文。浏览器/操作系统/来源/省份/ISP 等
// 其它 HBarCard 用途的 key 本就该原样展示（浏览器名、系统名等），不套这张表。
const DEVICE_LABELS: Record<string, string> = { mobile: "手机", tablet: "平板", desktop: "电脑" };

// data.gameResults 的 key 是后端拼出的 "gameId:result"（见 internal/server/game_common.go
// logRoundOutcome 的 detail 拼接、game_liarsdice.go 的 ":win"），result 只会是
// win/draw/doubleLoss 三种之一，同样是英文标识符拼接，不能直接展示。
const RESULT_LABELS: Record<string, string> = { win: "分胜负", draw: "平局", doubleloss: "双输" };

function relabelBuckets(rows: AnalyticsBucket[], labels: Record<string, string>): AnalyticsBucket[] {
  return rows.map((r) => ({ ...r, key: labelFor(labels, r.key) }));
}

// data.gameResults 的 key 形如 "othello:draw"：先按 ":" 拆开游戏名/结果分别查表翻译，
// 查不到时保底显示原始片段，不整体丢弃。
function relabelGameResultBuckets(rows: AnalyticsBucket[]): AnalyticsBucket[] {
  return rows.map((r) => {
    const [gameID, result] = r.key.split(":");
    const gameLabel = labelFor(GAME_LABELS, gameID || r.key);
    const key = result ? `${gameLabel}·${labelFor(RESULT_LABELS, result)}` : gameLabel;
    return { ...r, key };
  });
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

function AnalyticsTooltip({ active, payload, label }: {
  active?: boolean;
  payload?: Array<{ name?: string; value?: number; color?: string }>;
  label?: string;
}) {
  if (!active || !payload?.length) return null;
  return (
    <div className="analytics-tooltip">
      {label != null && label !== "" && <div className="analytics-tooltip-label">{label}</div>}
      {payload.map((p, i) => (
        <div key={i} className="analytics-tooltip-row">
          <span className="analytics-tooltip-swatch" style={{ background: p.color }} />
          <span>{p.name}</span>
          <strong>{formatNum(Number(p.value) || 0)}</strong>
        </div>
      ))}
    </div>
  );
}

/** 堆叠柱段间隙：每段 height -= 2、y += 2。 */
function InsetSegment(props: {
  x?: number; y?: number; width?: number; height?: number; fill?: string; radius?: number | number[];
}) {
  const { x = 0, y = 0, width = 0, height = 0, fill } = props;
  const h = Math.max(0, height - 2);
  const yy = y + 2;
  if (h <= 0 || width <= 0) return null;
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
  if (!matrix.length) return <p className="empty">暂无留存数据（需前端埋点上线后的访客）</p>;
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
    newVisitors: data.series.newVisitors[i] || 0,
    returning: data.series.returning[i] || 0
  }));
}

export function AnalyticsPanel({ onError }: { onError: (message: string) => void }) {
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
                <LineChart data={trends} margin={{ top: 8, right: 12, left: 0, bottom: 0 }}>
                  <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
                  <XAxis dataKey="day" tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <YAxis tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
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
                <LineChart data={trends} margin={{ top: 8, right: 12, left: 0, bottom: 0 }}>
                  <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
                  <XAxis dataKey="day" tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <YAxis tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <Tooltip content={<AnalyticsTooltip />} />
                  <Line type="monotone" dataKey="pageviews" name="页面浏览" stroke="var(--chart-1)" strokeWidth={2} dot={false}
                    activeDot={{ r: 4, strokeWidth: 2, stroke: "var(--chart-surface)" }} isAnimationActive={false} />
                </LineChart>
              </ResponsiveContainer>
            </ChartCard>
          </div>

          <ChartCard title="新访客 vs 回访" table={
            <table className="analytics-data-table">
              <thead><tr><th>日期</th><th>新访客</th><th>回访</th></tr></thead>
              <tbody>
                {trends.map((r) => (
                  <tr key={r.day}><td>{r.day}</td><td>{r.newVisitors}</td><td>{r.returning}</td></tr>
                ))}
              </tbody>
            </table>
          }>
            <ResponsiveContainer width="100%" height={220}>
              <BarChart data={trends} margin={{ top: 8, right: 12, left: 0, bottom: 0 }}>
                <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
                <XAxis dataKey="day" tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                <YAxis tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                <Tooltip content={<AnalyticsTooltip />} />
                <Legend />
                <Bar dataKey="newVisitors" name="新访客" stackId="a" fill="var(--chart-1)" maxBarSize={24}
                  shape={<InsetSegment />} isAnimationActive={false} />
                <Bar dataKey="returning" name="回访" stackId="a" fill="var(--chart-3)" maxBarSize={24}
                  radius={[4, 4, 0, 0]} isAnimationActive={false} />
              </BarChart>
            </ResponsiveContainer>
          </ChartCard>

          <ChartCard title="留存热力图 (D0–D14)">
            <RetentionHeatmap
              matrix={data.retention?.matrix || []}
              cohorts={data.retention?.cohorts || []}
              offsets={data.retention?.offsets || []}
            />
          </ChartCard>

          <div className="analytics-row-3">
            <HBarCard title="设备类型" rows={relabelBuckets(data.devices || [], DEVICE_LABELS).slice(0, 4)} donut />
            <HBarCard title="浏览器" rows={(data.browsers || []).slice(0, 6)} />
            <HBarCard title="操作系统" rows={(data.os || []).slice(0, 6)} />
          </div>

          <div className="analytics-row-3">
            <HBarCard title="来源 Top10" rows={(data.referrers || []).slice(0, 10)} />
            <HBarCard title="省份 Top10" rows={(data.provinces || []).slice(0, 10)} />
            <HBarCard title="ISP" rows={(data.isps || []).slice(0, 6)} donut />
          </div>

          <ChartCard title="会话时长分布" table={<BucketTable rows={data.sessionBuckets || []} />}>
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={data.sessionBuckets || []} margin={{ top: 8, right: 12, left: 0, bottom: 0 }}>
                <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
                <XAxis dataKey="key" tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                <YAxis tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                <Tooltip content={<AnalyticsTooltip />} />
                <Bar dataKey="value" name="会话数" maxBarSize={24} radius={[4, 4, 0, 0]} isAnimationActive={false}>
                  {(data.sessionBuckets || []).map((_, i) => (
                    <Cell key={i} fill={SEQ_COLORS[i % SEQ_COLORS.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </ChartCard>

          <ChartCard
            title="每日对局数（按游戏）"
            table={<SeriesTable days={data.series.days} series={data.gameRounds || []} labels={GAME_LABELS} />}
          >
            <StackedGameChart days={data.series.days} series={data.gameRounds || []} labels={GAME_LABELS} />
          </ChartCard>

          <div className="analytics-row-2">
            <HBarCard title="对局结果" rows={relabelGameResultBuckets(data.gameResults || [])} />
            <ChartCard title="开房数" table={<SeriesTable days={data.series.days} series={data.roomCreates || []} labels={GAME_LABELS} />}>
              <StackedGameChart days={data.series.days} series={data.roomCreates || []} labels={GAME_LABELS} />
            </ChartCard>
          </div>

          <div className="analytics-row-2">
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
              <ResponsiveContainer width="100%" height={200}>
                <LineChart data={data.series.days.map((day, i) => ({
                  day,
                  publish: data.punishment?.publish?.[i] || 0,
                  done: data.punishment?.done?.[i] || 0,
                  reject: data.punishment?.reject?.[i] || 0
                }))}>
                  <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
                  <XAxis dataKey="day" tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <YAxis tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <Tooltip content={<AnalyticsTooltip />} />
                  <Legend />
                  <Line type="monotone" dataKey="publish" name="发布" stroke="var(--chart-1)" strokeWidth={2} dot={false} isAnimationActive={false} />
                  <Line type="monotone" dataKey="done" name="完成" stroke="var(--chart-3)" strokeWidth={2} dot={false} isAnimationActive={false} />
                  <Line type="monotone" dataKey="reject" name="驳回" stroke="var(--chart-critical)" strokeWidth={2} dot={false} isAnimationActive={false} />
                </LineChart>
              </ResponsiveContainer>
            </ChartCard>

            <ChartCard title="证明耗时分布" table={<BucketTable rows={data.punishment?.proofMs || []} />}>
              <ResponsiveContainer width="100%" height={220}>
                <BarChart data={data.punishment?.proofMs || []}>
                  <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
                  <XAxis dataKey="key" tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <YAxis tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <Tooltip content={<AnalyticsTooltip />} />
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
            <ChartCard title="名争·白给等活动" table={<SeriesTable days={data.series.days} series={data.activity || []} labels={ACTIVITY_LABELS} />}>
              <StackedGameChart days={data.series.days} series={data.activity || []} asLine labels={ACTIVITY_LABELS} />
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
              <ResponsiveContainer width="100%" height={220}>
                <ComposedChart data={data.series.days.map((day, i) => ({
                  day,
                  total: data.petBond?.total?.[i] || 0,
                  new: data.petBond?.new?.[i] || 0
                }))}>
                  <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
                  <XAxis dataKey="day" tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <YAxis tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                  <Tooltip content={<AnalyticsTooltip />} />
                  <Legend />
                  <Bar dataKey="new" name="新增" fill="var(--chart-2)" maxBarSize={24} radius={[4, 4, 0, 0]} isAnimationActive={false} />
                  <Line type="monotone" dataKey="total" name="总数" stroke="var(--chart-1)" strokeWidth={2} dot={false} isAnimationActive={false} />
                </ComposedChart>
              </ResponsiveContainer>
            </ChartCard>
          </div>

          <ChartCard title="聊天活跃" table={
            <table className="analytics-data-table">
              <thead><tr><th>日期</th><th>大厅</th><th>房间</th><th>发言人</th></tr></thead>
              <tbody>
                {data.series.days.map((d, i) => (
                  <tr key={d}>
                    <td>{d}</td>
                    <td>{data.chat?.lobby?.[i] || 0}</td>
                    <td>{data.chat?.room?.[i] || 0}</td>
                    <td>{data.chat?.speakers?.[i] || 0}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          }>
            <ResponsiveContainer width="100%" height={220}>
              <LineChart data={data.series.days.map((day, i) => ({
                day,
                lobby: data.chat?.lobby?.[i] || 0,
                room: data.chat?.room?.[i] || 0,
                speakers: data.chat?.speakers?.[i] || 0
              }))}>
                <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
                <XAxis dataKey="day" tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                <YAxis tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                <Tooltip content={<AnalyticsTooltip />} />
                <Legend />
                <Line type="monotone" dataKey="lobby" name="大厅消息" stroke="var(--chart-1)" strokeWidth={2} dot={false} isAnimationActive={false} />
                <Line type="monotone" dataKey="room" name="房间消息" stroke="var(--chart-2)" strokeWidth={2} dot={false} isAnimationActive={false} />
                <Line type="monotone" dataKey="speakers" name="发言人数" stroke="var(--chart-3)" strokeWidth={2} dot={false} isAnimationActive={false} />
              </LineChart>
            </ResponsiveContainer>
          </ChartCard>

          <ChartCard title="转化漏斗" table={<BucketTable rows={funnelOrdered(data.funnel || [])} />}>
            <ResponsiveContainer width="100%" height={200}>
              <BarChart layout="vertical" data={funnelOrdered(data.funnel || [])} margin={{ left: 16, right: 16 }}>
                <CartesianGrid stroke="var(--chart-grid)" horizontal={false} strokeDasharray="" />
                <XAxis type="number" tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                <YAxis type="category" dataKey="key" width={80} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
                <Tooltip content={<AnalyticsTooltip />} />
                <Bar dataKey="value" name="人数" maxBarSize={24} radius={[0, 4, 4, 0]} isAnimationActive={false}>
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
  const order = ["visit", "login", "room", "round", "punish_done"];
  const labels: Record<string, string> = {
    visit: "访问", login: "登录", room: "进房", round: "开局", punish_done: "完成惩罚"
  };
  const map = new Map(rows.map((r) => [r.key, r.value]));
  return order.map((k) => ({ key: labels[k] || k, value: map.get(k) || 0 }));
}

function HBarCard({ title, rows, donut }: { title: string; rows: AnalyticsBucket[]; donut?: boolean }) {
  const top = rows.slice(0, donut ? 4 : 10);
  const total = top.reduce((s, r) => s + r.value, 0);
  return (
    <ChartCard title={title} table={<BucketTable rows={top} />}>
      {donut && top.length > 0 ? (
        <div className="analytics-donut-wrap">
          <ResponsiveContainer width="100%" height={180}>
            <PieChart>
              <Pie data={top} dataKey="value" nameKey="key" innerRadius="60%" outerRadius="85%" paddingAngle={2}
                isAnimationActive={false}>
                {top.map((_, i) => <Cell key={i} fill={CHART_COLORS[i % CHART_COLORS.length]} />)}
              </Pie>
              <Tooltip content={<AnalyticsTooltip />} />
            </PieChart>
          </ResponsiveContainer>
          <div className="analytics-donut-center">
            <strong>{formatNum(total)}</strong>
            <span>合计</span>
          </div>
        </div>
      ) : (
        <ResponsiveContainer width="100%" height={200}>
          <BarChart layout="vertical" data={top} margin={{ left: 8, right: 12 }}>
            <CartesianGrid stroke="var(--chart-grid)" horizontal={false} strokeDasharray="" />
            <XAxis type="number" tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
            <YAxis type="category" dataKey="key" width={72} tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
            <Tooltip content={<AnalyticsTooltip />} />
            <Bar dataKey="value" name="数量" maxBarSize={24} radius={[0, 4, 4, 0]} fill="var(--chart-1)" isAnimationActive={false} />
          </BarChart>
        </ResponsiveContainer>
      )}
    </ChartCard>
  );
}

function StackedGameChart({
  days, series, asLine, labels
}: {
  days: string[]; series: AnalyticsNamedSeries[]; asLine?: boolean; labels?: Record<string, string>;
}) {
  if (!series.length) return <p className="empty">暂无数据</p>;
  const rows = seriesToChartRows(days, series);
  if (asLine) {
    return (
      <ResponsiveContainer width="100%" height={220}>
        <LineChart data={rows}>
          <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
          <XAxis dataKey="day" tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
          <YAxis tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
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
  return (
    <ResponsiveContainer width="100%" height={240}>
      <BarChart data={rows}>
        <CartesianGrid stroke="var(--chart-grid)" vertical={false} strokeDasharray="" />
        <XAxis dataKey="day" tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
        <YAxis tick={{ fill: "var(--chart-ink-muted)", fontSize: 11 }} stroke="var(--chart-axis)" />
        <Tooltip content={<AnalyticsTooltip />} />
        <Legend />
        {series.map((s, i) => (
          <Bar key={s.key} dataKey={s.key} name={labelFor(labels, s.key)} stackId="a"
            fill={CHART_COLORS[i % CHART_COLORS.length]} maxBarSize={24}
            shape={i < series.length - 1 ? <InsetSegment /> : undefined}
            radius={i === series.length - 1 ? [4, 4, 0, 0] : undefined}
            isAnimationActive={false} />
        ))}
      </BarChart>
    </ResponsiveContainer>
  );
}
