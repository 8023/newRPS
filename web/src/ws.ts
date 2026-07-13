// Protobuf 二进制 WebSocket 客户端（全链路无 JSON 文本载荷）。
// 状态通道：FULL / DELTA（路径补丁 + hash 校验，失败 sync:full）。

import { applyOps, crc32Hex, type PatchOp as DeltaOp } from "./delta";
import {
  decodeEnvelope,
  encodeEnvelope,
  encodeRawBodyDynamic,
  normalizeStateTree,
  protoValueToPlain,
  rawBodyToPlain,
  stateDocToPlain,
  type Envelope
} from "./wire";

type Handler = (data: any) => void;
type AckCallback = (response: any) => void;

type Pending = {
  resolve: (value: any) => void;
  reject: (error: Error) => void;
  timer: ReturnType<typeof setTimeout>;
};

type ChannelState = {
  doc: any;
  hash: string;
  seq: number;
};

class GameSocket {
  private ws: WebSocket | null = null;
  private handlers = new Map<string, Set<Handler>>();
  private pending = new Map<number, Pending>();
  private nextId = 1;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectAttempt = 0;
  private intentionallyClosed = false;
  private authToken = "";
  private connectPromise: Promise<void> | null = null;
  private channels = new Map<string, ChannelState>();
  private resyncing = new Set<string>();
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null;
  private lastPongAt = 0;
  private visibilityHandler: (() => void) | null = null;
  private pageShowHandler: ((ev: PageTransitionEvent) => void) | null = null;

  public auth: { token?: string; fingerprint?: string } = {};
  public connected = false;
  public io = {
    on: (event: string, handler: Handler) => {
      if (event === "reconnect_attempt") this.on("reconnect_attempt", handler);
    },
    off: (event: string, handler?: Handler) => {
      if (event === "reconnect_attempt") this.off("reconnect_attempt", handler);
    }
  };

  constructor() {
    if (typeof document !== "undefined") {
      this.visibilityHandler = () => {
        if (document.visibilityState === "visible") this.ensureAlive();
      };
      document.addEventListener("visibilitychange", this.visibilityHandler);
    }
    if (typeof window !== "undefined") {
      this.pageShowHandler = (ev: PageTransitionEvent) => {
        if (ev.persisted) this.ensureAlive();
      };
      window.addEventListener("pageshow", this.pageShowHandler);
    }
  }

  on(event: string, handler: Handler) {
    if (!this.handlers.has(event)) this.handlers.set(event, new Set());
    this.handlers.get(event)!.add(handler);
  }

  off(event: string, handler?: Handler) {
    if (!handler) {
      this.handlers.delete(event);
      return;
    }
    this.handlers.get(event)?.delete(handler);
  }

  private emitLocal(event: string, data?: unknown) {
    const set = this.handlers.get(event);
    if (!set) return;
    for (const handler of [...set]) handler(data);
  }

  private wsURL(token: string, fingerprint: string) {
    const proto = location.protocol === "https:" ? "wss:" : "ws:";
    const fp = fingerprint ? `&fp=${encodeURIComponent(fingerprint)}` : "";
    return `${proto}//${location.host}/ws?token=${encodeURIComponent(token)}${fp}`;
  }

  async connect() {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return this.connectPromise || Promise.resolve();
    }
    if (this.connectPromise) return this.connectPromise;

    this.intentionallyClosed = false;
    this.authToken = String(this.auth.token || "");
    const fingerprint = String(this.auth.fingerprint || "");
    this.emitLocal("reconnect_attempt");
    this.connectPromise = new Promise<void>((resolve, reject) => {
      try {
        const ws = new WebSocket(this.wsURL(this.authToken, fingerprint));
        ws.binaryType = "arraybuffer";
        this.ws = ws;
        let opened = false;
        ws.onopen = () => {
          if (this.ws !== ws) return;
          opened = true;
          this.connected = true;
          this.reconnectAttempt = 0;
          this.lastPongAt = Date.now();
          this.channels.clear();
          this.startHeartbeat();
          this.emitLocal("connect");
          resolve();
        };
        ws.onmessage = (event) => {
          if (this.ws !== ws) return;
          if (event.data instanceof ArrayBuffer) this.handleBinary(new Uint8Array(event.data));
        };
        ws.onerror = () => undefined;
        ws.onclose = (ev) => {
          if (this.ws !== ws) return;
          this.stopHeartbeat();
          this.connected = false;
          this.ws = null;
          this.connectPromise = null;
          this.failPending("连接已断开");
          const reason = ev.reason || "";
          const replaced = ev.code === 1008 && /replaced/i.test(reason);
          if (!opened) {
            const error = new Error(reason || "WebSocket 握手失败") as Error & { data?: { code?: string } };
            if (!replaced) error.data = { code: "SESSION_HANDSHAKE_FAILED" };
            this.emitLocal("connect_error", error);
            reject(error);
            return;
          }
          this.emitLocal("disconnect");
          if (replaced) return;
          if (!this.intentionallyClosed) this.scheduleReconnect();
        };
      } catch (error) {
        this.connectPromise = null;
        reject(error instanceof Error ? error : new Error("WebSocket 连接失败"));
      }
    });
    return this.connectPromise;
  }

  private scheduleReconnect() {
    if (this.reconnectTimer || this.intentionallyClosed) return;
    this.reconnectAttempt += 1;
    const delay = Math.min(12_000, 600 * Math.pow(1.6, this.reconnectAttempt));
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect().catch(() => undefined);
    }, delay);
  }

  private startHeartbeat() {
    this.stopHeartbeat();
    this.lastPongAt = Date.now();
    this.heartbeatTimer = setInterval(() => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
      if (Date.now() - this.lastPongAt > 70_000) {
        try {
          this.ws.close();
        } catch {
          /* ignore */
        }
        return;
      }
      this.emit("ping", { t: Date.now() }, () => {
        this.lastPongAt = Date.now();
      });
    }, 25_000);
  }

  private stopHeartbeat() {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  private ensureAlive() {
    if (this.intentionallyClosed) return;
    if (!this.ws || this.ws.readyState === WebSocket.CLOSED || this.ws.readyState === WebSocket.CLOSING) {
      this.connected = false;
      this.connect().catch(() => undefined);
      return;
    }
    if (this.ws.readyState === WebSocket.OPEN) {
      this.emit("ping", { t: Date.now() }, () => {
        this.lastPongAt = Date.now();
      });
    }
  }

  private failPending(message: string) {
    for (const [id, pending] of this.pending) {
      clearTimeout(pending.timer);
      pending.reject(new Error(message));
      this.pending.delete(id);
    }
  }

  private handleBinary(buf: Uint8Array) {
    let env: Envelope;
    try {
      env = decodeEnvelope(buf);
    } catch {
      return;
    }
    // RPC 应答：id 可能是 number；兼容 Long/string
    const replyId = env.id != null ? Number(env.id) : NaN;
    if (Number.isFinite(replyId) && replyId !== 0 && this.pending.has(replyId)) {
      const pending = this.pending.get(replyId)!;
      clearTimeout(pending.timer);
      this.pending.delete(replyId);
      if (env.err) pending.reject(new Error(env.err));
      else pending.resolve(rawBodyToPlain(env.rawBody) ?? {});
      return;
    }
    if (!env.event) return;

    // FULL 状态
    if (env.kind === 1) {
      void this.applyFullState(env);
      return;
    }
    // DELTA：路径补丁 + 合并后哈希校验
    if (env.kind === 2) {
      void this.applyDeltaState(env);
      return;
    }
    // RAW 推送（player:batch / chat:append 等，不受状态 diff 影响）
    this.emitLocal(env.event, rawBodyToPlain(env.rawBody));
  }

  private async applyFullState(env: Envelope) {
    const channel = env.channel || env.event || "";
    let doc = stateDocToPlain(env.fullState);
    doc = normalizeStateTree(doc);
    this.channels.set(channel, { doc, hash: env.hash || "", seq: env.seq || 0 });
    if (env.event) this.emitLocal(env.event, doc);
  }

  private async applyDeltaState(env: Envelope) {
    const channel = env.channel || env.event || "";
    const cur = this.channels.get(channel);
    if (!cur || cur.doc == null) {
      this.requestFull(channel);
      return;
    }
    const rawOps = env.delta?.ops || [];
    // decodeEnvelope 已把 Value Message 转成 plain（含 0）
    const ops: DeltaOp[] = rawOps.map((op: any) => ({
      path: op.path || "",
      remove: Boolean(op.remove),
      value: op.remove ? undefined : op.value !== undefined ? op.value : protoValueToPlain(op.value)
    }));
    let next: any;
    try {
      next = applyOps(cur.doc, ops);
      next = normalizeStateTree(next);
    } catch {
      this.requestFull(channel);
      return;
    }
    const hash = crc32Hex(next);
    if (env.hash && hash !== env.hash) {
      this.requestFull(channel);
      return;
    }
    this.channels.set(channel, { doc: next, hash: env.hash || hash, seq: env.seq || 0 });
    if (env.event) this.emitLocal(env.event, next);
  }

  private requestFull(channel: string) {
    if (!channel || this.resyncing.has(channel)) return;
    this.resyncing.add(channel);
    this.channels.delete(channel);
    this.emit("sync:full", { channel }, () => {
      this.resyncing.delete(channel);
    });
    setTimeout(() => this.resyncing.delete(channel), 8000);
  }

  emit(event: string, payload?: unknown, ack?: AckCallback) {
    const id = this.nextId++;
    const rawBody = encodeRawBodyDynamic(payload ?? {});
    let body: Uint8Array;
    try {
      body = encodeEnvelope({ event, id, kind: 0, rawBody });
    } catch (error) {
      if (ack) ack({ error: error instanceof Error ? error.message : "编码失败" });
      return;
    }
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      if (ack) ack({ error: "未连接到服务器" });
      return;
    }
    // 必须先登记 pending 再 send，避免应答先到时丢 ack（表现为按钮“点了没反应”）
    if (ack) {
      const timer = setTimeout(() => {
        this.pending.delete(id);
        ack({ error: "请求超时" });
      }, 15_000);
      this.pending.set(id, {
        resolve: (value) => ack(value),
        reject: (error) => ack({ error: error.message }),
        timer
      });
    }
    try {
      this.ws.send(body);
    } catch {
      if (ack) {
        const pending = this.pending.get(id);
        if (pending) {
          clearTimeout(pending.timer);
          this.pending.delete(id);
        }
        ack({ error: "发送失败" });
      }
    }
  }

  close() {
    this.intentionallyClosed = true;
    this.stopHeartbeat();
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    try {
      this.ws?.close();
    } catch {
      /* ignore */
    }
    this.ws = null;
    this.connected = false;
  }
}

export const socket = new GameSocket();
