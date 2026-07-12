// Protobuf 二进制 WebSocket 客户端 + 状态通道（FULL/DELTA + 哈希校验）。

import { applyOps, sha256Hex, type PatchOp as DeltaOp } from "./delta";
import { decodeEnvelope, encodeEnvelope, jsonParse, type Envelope } from "./wire";

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
    // iOS Safari 切后台/回前台后 WS 常已死但 onclose 延迟；回前台主动探测并重连。
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
    // 已有连接或正在连：复用，避免同页/双标签秒级双开触发服务端 socket_duplicate
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
          // 若期间又建了更新的连接，忽略旧 socket 的 open
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
          else if (typeof event.data === "string") this.handleLegacyJSON(event.data);
        };
        ws.onerror = () => undefined;
        ws.onclose = (ev) => {
          // 关键：旧连接被服务端顶替关闭时，不能清空当前新 this.ws，否则会误重连打成死循环
          if (this.ws !== ws) return;

          this.stopHeartbeat();
          this.connected = false;
          this.ws = null;
          this.connectPromise = null;
          this.failPending("连接已断开");

          const reason = ev.reason || "";
          const replaced = ev.code === 1008 && /replaced/i.test(reason);

          if (!opened) {
            // 握手失败。1008+replaced 不是 token 坏；勿当 SESSION_INVALID 清 token。
            const error = new Error(reason || "WebSocket 连接失败") as Error & { data?: { code?: string } };
            if (!replaced && (ev.code === 1008 || /session|unauthorized|token|invalid/i.test(reason))) {
              error.data = { code: "SESSION_INVALID" };
            }
            this.emitLocal("connect_error", error);
            reject(error);
            return;
          }

          this.emitLocal("disconnect");
          // 同 SID 被其它标签/连接顶替：不要立刻重连（否则两标签互相踢）
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
    // iOS 弱网/切后台后多给一点退避，避免秒级狂连
    const delay = Math.min(12_000, 600 * Math.pow(1.6, this.reconnectAttempt));
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect().catch(() => undefined);
    }, delay);
  }

  private startHeartbeat() {
    this.stopHeartbeat();
    this.lastPongAt = Date.now();
    // 25s 一次应用层 ping：扛住多数反代 60s 空闲超时，并探测半开连接
    this.heartbeatTimer = setInterval(() => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
      if (Date.now() - this.lastPongAt > 70_000) {
        // 半开：强制关掉走重连
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

  /** 页面重新可见时：若已断则重连；若仍 OPEN 则立刻 ping 探活 */
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

  private handleLegacyJSON(raw: string) {
    try {
      const msg = JSON.parse(raw);
      if (typeof msg.id === "number" && this.pending.has(msg.id)) {
        const pending = this.pending.get(msg.id)!;
        clearTimeout(pending.timer);
        this.pending.delete(msg.id);
        if (msg.err) pending.reject(new Error(msg.err));
        else pending.resolve(msg.d ?? {});
        return;
      }
      if (msg.e) this.emitLocal(msg.e, msg.d);
    } catch {
      /* ignore */
    }
  }

  private handleBinary(buf: Uint8Array) {
    let env: Envelope;
    try {
      env = decodeEnvelope(buf);
    } catch {
      return;
    }
    // RPC response（id 可能为 0 以外的数字；不能用 if (env.id) 漏掉合法边界）
    if (typeof env.id === "number" && this.pending.has(env.id)) {
      const pending = this.pending.get(env.id)!;
      clearTimeout(pending.timer);
      this.pending.delete(env.id);
      if (env.err) pending.reject(new Error(env.err));
      else pending.resolve(jsonParse(env.raw) ?? {});
      return;
    }
    if (!env.event) return;

    // 状态通道 FULL / DELTA（kind 缺省为 0=RAW）
    if (env.kind === 1 || env.kind === 2) {
      void this.applyStateEnvelope(env);
      return;
    }
    // RAW 推送（含 player:batch 聚合）
    this.emitLocal(env.event, jsonParse(env.raw));
  }

  private async applyStateEnvelope(env: Envelope) {
    const channel = env.channel || env.event || "";
    const event = env.event || "";
    if (env.kind === 1 && env.full) {
      const doc = jsonParse(env.full);
      this.channels.set(channel, { doc, hash: env.hash || "", seq: env.seq || 0 });
      this.emitLocal(event, this.materializeEvent(event, doc));
      return;
    }
    if (env.kind === 2) {
      const cur = this.channels.get(channel);
      if (!cur || cur.doc == null) {
        this.requestFull(channel);
        return;
      }
      const ops: DeltaOp[] = (env.ops || []).map((op) => ({
        path: op.path,
        remove: op.remove,
        value: op.value ? jsonParse(op.value) : undefined
      }));
      let next: any;
      try {
        next = applyOps(cur.doc, ops);
      } catch {
        this.requestFull(channel);
        return;
      }
      const hash = await sha256Hex(next);
      if (env.hash && hash !== env.hash) {
        this.requestFull(channel);
        return;
      }
      this.channels.set(channel, { doc: next, hash: env.hash || hash, seq: env.seq || 0 });
      this.emitLocal(event, this.materializeEvent(event, next));
    }
  }

  /** 将 map 形态状态展开为业务层习惯的结构 */
  private materializeEvent(event: string, doc: any): any {
    if (event === "lobby:update" && doc && typeof doc === "object") {
      const players = doc.players;
      const rooms = doc.rooms;
      return {
        ...doc,
        players: Array.isArray(players) ? players : Object.values(players || {}),
        rooms: Array.isArray(rooms) ? rooms : Object.values(rooms || {})
      };
    }
    return doc;
  }

  private requestFull(channel: string) {
    if (!channel || this.resyncing.has(channel)) return;
    this.resyncing.add(channel);
    // 本地通道失效，避免继续在脏状态上叠 delta
    this.channels.delete(channel);
    this.emit("sync:full", { channel }, () => {
      this.resyncing.delete(channel);
    });
    setTimeout(() => this.resyncing.delete(channel), 8000);
  }

  emit(event: string, payload?: unknown, ack?: AckCallback) {
    const id = this.nextId++;
    const raw = new TextEncoder().encode(JSON.stringify(payload ?? {}));
    const body = encodeEnvelope({ event, id, kind: 0, raw });
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      if (ack) ack({ error: "未连接到服务器" });
      return;
    }
    if (ack) {
      const timer = setTimeout(() => {
        this.pending.delete(id);
        ack({ error: "请求超时" });
      }, 20_000);
      this.pending.set(id, {
        resolve: (value) => ack(value),
        reject: (error) => ack({ error: error.message }),
        timer
      });
    }
    this.ws.send(body);
  }

  disconnect() {
    this.intentionallyClosed = true;
    this.stopHeartbeat();
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.ws?.close();
    this.ws = null;
    this.connected = false;
  }
}

export const socket = new GameSocket();
