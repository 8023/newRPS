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

  public auth: { token?: string } = {};
  public connected = false;
  public io = {
    on: (event: string, handler: Handler) => {
      if (event === "reconnect_attempt") this.on("reconnect_attempt", handler);
    },
    off: (event: string, handler?: Handler) => {
      if (event === "reconnect_attempt") this.off("reconnect_attempt", handler);
    }
  };

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

  private wsURL(token: string) {
    const proto = location.protocol === "https:" ? "wss:" : "ws:";
    return `${proto}//${location.host}/ws?token=${encodeURIComponent(token)}`;
  }

  async connect() {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return this.connectPromise || Promise.resolve();
    }
    this.intentionallyClosed = false;
    this.authToken = String(this.auth.token || "");
    this.emitLocal("reconnect_attempt");
    this.connectPromise = new Promise<void>((resolve, reject) => {
      try {
        const ws = new WebSocket(this.wsURL(this.authToken));
        ws.binaryType = "arraybuffer";
        this.ws = ws;
        let opened = false;
        ws.onopen = () => {
          opened = true;
          this.connected = true;
          this.reconnectAttempt = 0;
          this.channels.clear();
          this.emitLocal("connect");
          resolve();
        };
        ws.onmessage = (event) => {
          if (event.data instanceof ArrayBuffer) this.handleBinary(new Uint8Array(event.data));
          else if (typeof event.data === "string") this.handleLegacyJSON(event.data);
        };
        ws.onerror = () => undefined;
        ws.onclose = (ev) => {
          this.connected = false;
          this.ws = null;
          this.connectPromise = null;
          this.failPending("连接已断开");
          if (!opened) {
            // 浏览器 WebSocket 拿不到 HTTP 状态码；1008=Policy Violation 常见于服务端拒绝会话。
            // 其余失败（网络、5xx 等）不要一律当成 SESSION_INVALID，否则会误清 token 导致登录态闪烁。
            const error = new Error(ev.reason || "WebSocket 连接失败") as Error & { data?: { code?: string } };
            if (ev.code === 1008 || /session|unauthorized|token|invalid/i.test(ev.reason || "")) {
              error.data = { code: "SESSION_INVALID" };
            }
            this.emitLocal("connect_error", error);
            reject(error);
            return;
          }
          this.emitLocal("disconnect");
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
    if (this.reconnectTimer) return;
    this.reconnectAttempt += 1;
    const delay = Math.min(5_000, 800 * this.reconnectAttempt);
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect().catch(() => undefined);
    }, delay);
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
