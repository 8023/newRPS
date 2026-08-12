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

// toBase64Url：浏览器指纹内容不受服务端控制，编码成 base64url 后再作为 Sec-WebSocket-Protocol
// 的一段值传输，保证只含 HTTP token 语法允许的字符（A-Z a-z 0-9 - _），避免握手失败。
// btoa 要求输入落在 Latin1 范围内，指纹理论上应该都是；万一不是就退回空指纹而不是让整次
// 连接尝试直接抛异常失败。
function toBase64Url(input: string): string {
  try {
    return btoa(input).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
  } catch {
    return "";
  }
}

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

  private wsURL() {
    const proto = location.protocol === "https:" ? "wss:" : "ws:";
    return `${proto}//${location.host}/ws`;
  }

  // authProtocols 把 token/指纹塞进 Sec-WebSocket-Protocol 握手头，而不是 URL 查询串——
  // 反向代理的访问日志默认会记录完整请求行/URL，token 一旦出现在其中，日志系统本身就
  // 变成了一个 24 小时有效的身份凭据泄露面。Sec-WebSocket-Protocol 是浏览器 WebSocket
  // 构造函数唯一能在握手阶段附带自定义值、又不出现在 URL 里的字段。
  // token 本身只含 base64url 字符和 "."，天然是合法的 HTTP token；指纹内容不受控，
  // 编码成 base64url 后再拼前缀，避免其中出现 HTTP token 语法不允许的字符导致握手失败。
  private authProtocols(token: string, fingerprint: string): string[] {
    const protocols = [`auth.${token}`];
    if (fingerprint) protocols.push(`fp.${toBase64Url(fingerprint)}`);
    return protocols;
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
        const ws = new WebSocket(this.wsURL(), this.authProtocols(this.authToken, fingerprint));
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
      this.emit("ping", { t: Date.now() }, (response) => {
        this.lastPongAt = Date.now();
        this.handlePingBuildId(response);
      });
    }, 25_000);
  }

  // 心跳应答里顺带的 buildId 转发成本地事件，与连接建立时的一次性 server:hello 推送
  // 走同一个下游处理逻辑，App 侧只需订阅一个事件即可覆盖两条通道。
  private handlePingBuildId(response: any) {
    const buildId = response?.buildId;
    if (typeof buildId === "string" && buildId) this.emitLocal("server:hello", { buildId });
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
      this.emit("ping", { t: Date.now() }, (response) => {
        this.lastPongAt = Date.now();
        this.handlePingBuildId(response);
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
    // 连续 seq：信任服务端 hash，跳过 O(树) 的 clone+normalize+CRC；
    // 乱序/缺口：原地打补丁后做全量 CRC，失败则 resync。
    // 每 32 个连续 delta 强制抽一次 CRC，防止归一化分叉长期潜伏（如 titleColors 缺键）。
    const envSeq = env.seq || 0;
    const sequential = envSeq > 0 && cur.seq > 0 && envSeq === cur.seq + 1;
    const sampleCrc = sequential && envSeq > 0 && envSeq % 32 === 0;
    let next: any;
    try {
      // 原地打补丁（避免 structuredClone 全树拷贝）；失败路径须丢弃文档并 resync。
      next = applyOps(cur.doc, ops, true);
      next = normalizeStateTree(next);
    } catch {
      // 与 delta.ts 不变量一致：mutate 失败后文档半残，显式丢弃再 requestFull。
      this.channels.delete(channel);
      this.requestFull(channel);
      return;
    }
    let localHash = "";
    if (env.hash && (!sequential || sampleCrc)) {
      localHash = crc32Hex(next);
      if (localHash !== env.hash) {
        this.channels.delete(channel);
        this.requestFull(channel);
        return;
      }
    }
    // 服务端未带 hash 时用本地算值，避免把旧 hash 记到新文档上。
    const nextHash = env.hash || localHash || (sequential ? cur.hash : crc32Hex(next));
    this.channels.set(channel, { doc: next, hash: nextHash, seq: envSeq });
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
