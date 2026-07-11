// 原生 WebSocket 客户端，兼容原 Socket.IO 的事件名与 ack 回调语义。
// 协议：
//   请求:  { e: string, id: number, d?: unknown }
//   响应:  { id: number, d?: unknown, err?: string }
//   推送:  { e: string, d?: unknown }

type Handler = (data: any) => void;
type AckCallback = (response: any) => void;

type Pending = {
  resolve: (value: any) => void;
  reject: (error: Error) => void;
  timer: ReturnType<typeof setTimeout>;
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
        this.ws = ws;
        let opened = false;
        ws.onopen = () => {
          opened = true;
          this.connected = true;
          this.reconnectAttempt = 0;
          this.emitLocal("connect");
          resolve();
        };
        ws.onmessage = (event) => this.handleMessage(String(event.data));
        ws.onerror = () => {
          // 握手失败时 onclose 会收尾；这里补一次 connect_error 方便清 token
        };
        ws.onclose = () => {
          this.connected = false;
          this.ws = null;
          this.connectPromise = null;
          this.failPending("连接已断开");
          if (!opened) {
            // 未成功 open（常见：token 无效/过期），通知上层重新签发 session
            const error = new Error("Session invalid") as Error & { data?: { code?: string } };
            error.data = { code: "SESSION_INVALID" };
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
      // 重连前若 token 已在 localStorage 更新，用最新 auth
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

  private handleMessage(raw: string) {
    let msg: { e?: string; id?: number; d?: any; err?: string };
    try {
      msg = JSON.parse(raw);
    } catch {
      return;
    }
    if (typeof msg.id === "number" && this.pending.has(msg.id)) {
      const pending = this.pending.get(msg.id)!;
      clearTimeout(pending.timer);
      this.pending.delete(msg.id);
      if (msg.err) pending.reject(new Error(msg.err));
      else pending.resolve(msg.d ?? {});
      return;
    }
    if (msg.e) {
      if (msg.e === "error" && msg.d?.code) {
        const error = new Error(msg.d.message || "连接错误") as Error & { data?: { code?: string } };
        error.data = { code: msg.d.code };
        this.emitLocal("connect_error", error);
        return;
      }
      this.emitLocal(msg.e, msg.d);
    }
  }

  emit(event: string, payload?: unknown, ack?: AckCallback) {
    const id = this.nextId++;
    const body = JSON.stringify({ e: event, id, d: payload ?? {} });
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      const err = { error: "未连接到服务器" };
      if (ack) ack(err);
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
