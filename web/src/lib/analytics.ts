import { socket } from "../main";

/** GA4 语义会话：30 分钟无活动过期。 */
const SESSION_KEY = "rps-analytics-session";
const SESSION_IDLE_MS = 30 * 60_000;
const FLUSH_MAX = 10;
const SERVER_BATCH_MAX = 40;
const MAX_BUFFER_EVENTS = 400;
const FLUSH_INTERVAL_MS = 5000;

type SessionStore = { id: string; lastAt: number };

type PendingEvent = {
  n: string;
  v?: string;
  d?: string;
  x?: number;
  t: number;
};

type AnalyticsBatch = {
  sid: string;
  ref?: string;
  land?: string;
  ev: PendingEvent[];
};

const buffer: PendingEvent[] = [];
let flushTimer: ReturnType<typeof setInterval> | null = null;
let firstBatch = true;
let landingView = "";
let sessionPingTimer: ReturnType<typeof setInterval> | null = null;
let started = false;

function readSession(): SessionStore {
  try {
    const raw = localStorage.getItem(SESSION_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as SessionStore;
      if (parsed?.id && typeof parsed.lastAt === "number") return parsed;
    }
  } catch {
    /* ignore */
  }
  return { id: crypto.randomUUID(), lastAt: 0 };
}

function writeSession(s: SessionStore) {
  try {
    localStorage.setItem(SESSION_KEY, JSON.stringify(s));
  } catch {
    /* ignore */
  }
}

function ensureSession(): string {
  const now = Date.now();
  let s = readSession();
  if (!s.id || now - s.lastAt > SESSION_IDLE_MS) {
    s = { id: crypto.randomUUID(), lastAt: now };
    firstBatch = true;
  } else {
    s.lastAt = now;
  }
  writeSession(s);
  return s.id;
}

function flush() {
  if (buffer.length === 0) return;
  if (!socket.connected) return; // 断线保留缓冲，重连后补发
  const sid = ensureSession();
  // 服务端硬上限是 40 条。离线期间缓冲可能超过这个值，必须只移除本批，
  // 否则服务端截断后客户端会把未发送事件一并 splice 掉。
  const batch: AnalyticsBatch = {
    sid,
    ev: buffer.splice(0, SERVER_BATCH_MAX)
  };
  if (firstBatch) {
    try {
      const ref = document.referrer;
      if (ref) batch.ref = ref;
    } catch {
      /* ignore */
    }
    if (landingView) batch.land = landingView;
    firstBatch = false;
  }
  // 无 ack：不传第三个参数，不登记 pending
  socket.emit("analytics:collect", batch);
}

/** 记录一条分析事件（批量上报）。 */
export function track(name: string, opts?: { view?: string; detail?: string; value?: number }) {
  if (!started) startAnalytics();
  ensureSession();
  const ev: PendingEvent = { n: name, t: Date.now() };
  if (opts?.view) ev.v = opts.view;
  if (opts?.detail) ev.d = opts.detail;
  if (opts?.value != null) ev.x = opts.value;
  buffer.push(ev);
  if (buffer.length > MAX_BUFFER_EVENTS) {
    buffer.splice(0, buffer.length - MAX_BUFFER_EVENTS);
  }
  if (buffer.length >= FLUSH_MAX) flush();
}

function onVisibility() {
  if (document.visibilityState === "hidden") flush();
}

function onBeforeUnload() {
  flush();
}

/** 启动批量 flush 定时器与可见性监听（幂等）。 */
export function startAnalytics() {
  if (started) return;
  started = true;
  ensureSession();
  if (flushTimer == null) {
    flushTimer = setInterval(flush, FLUSH_INTERVAL_MS);
  }
  document.addEventListener("visibilitychange", onVisibility);
  window.addEventListener("beforeunload", onBeforeUnload);
  // 会话心跳：仅页面可见时
  if (sessionPingTimer == null) {
    sessionPingTimer = setInterval(() => {
      if (document.visibilityState === "visible") track("session_ping");
    }, 60_000);
  }
}

/** 记录页面/弹窗浏览。 */
export function trackPageview(view: string) {
  if (!landingView) landingView = view;
  track("pageview", { view });
}

/** 登录成功。 */
export function trackLoginSuccess(detail: "restore" | "manual") {
  track("login_success", { detail });
}

/** 主题切换。 */
export function trackThemeToggle(detail: "light" | "dark") {
  track("theme_toggle", { detail });
}
