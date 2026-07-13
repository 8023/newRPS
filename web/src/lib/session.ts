import { getBrowserFingerprint } from "../fingerprint";
import { playerIdKey, playerSecretKey, tokenKey } from "./constants";
import { socket } from "../ws";

export function randomUuid() {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") return crypto.randomUUID();
  // 兜底：旧浏览器没有 crypto.randomUUID 时，用 getRandomValues 拼一个 v4 UUID。
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const hex = [...bytes].map((b) => b.toString(16).padStart(2, "0"));
  return `${hex.slice(0, 4).join("")}-${hex.slice(4, 6).join("")}-${hex.slice(6, 8).join("")}-${hex.slice(8, 10).join("")}-${hex.slice(10, 16).join("")}`;
}

// 长期身份：playerId + playerSecret 一旦生成就长期保存在本地，和短期的 session token 解耦。
// 积分、战绩、称号只跟 playerId 走，token 过期/重发都不会清空玩家档案。
export function ensurePlayerIdentity() {
  let playerId = localStorage.getItem(playerIdKey);
  let playerSecret = localStorage.getItem(playerSecretKey);
  if (!playerId || !playerSecret) {
    playerId = randomUuid();
    playerSecret = `${randomUuid()}-${randomUuid()}`;
    localStorage.setItem(playerIdKey, playerId);
    localStorage.setItem(playerSecretKey, playerSecret);
  }
  return { playerId, playerSecret };
}

/** 会话 token 形如 sid.exp.sig；本地预检是否仍在有效期内。 */
export function sessionTokenLooksValid(token: string | null | undefined): token is string {
  if (!token) return false;
  const parts = token.split(".");
  if (parts.length !== 3 || !parts[0] || !parts[1] || !parts[2]) return false;
  const exp = Number(parts[1]);
  if (!Number.isFinite(exp) || exp <= Date.now()) return false;
  return true;
}

export function hasCachedLogin(): boolean {
  // 只要本地有过进厅名字，刷新后就尝试自动 player:join（token 过期会由 ensureSessionToken 换发）。
  return Boolean(localStorage.getItem("rps-online-name"));
}

export async function ensureSessionToken(forceNew = false) {
  const existing = localStorage.getItem(tokenKey);
  // 注意：本地只能看格式/过期时间，无法验 HMAC。密钥轮换后的「假有效」token 必须 forceNew 才能换掉。
  if (!forceNew && sessionTokenLooksValid(existing)) return existing;
  if (existing) localStorage.removeItem(tokenKey);
  const fingerprint = await getBrowserFingerprint();
  const response = await fetch("/api/session", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Browser-Fingerprint": fingerprint
    },
    body: JSON.stringify({ fingerprint })
  });
  const data = await response.json();
  if (!response.ok || !data.token) throw new Error(data.message || "Session failed");
  localStorage.setItem(tokenKey, String(data.token));
  return String(data.token);
}

let connectSessionFlight: Promise<string> | null = null;
/** WS 握手失败后换发 token 的次数，成功 connect 后清零 */
let wsAuthRetryCount = 0;

export function getWsAuthRetryCount() {
  return wsAuthRetryCount;
}

export function resetWsAuthRetryCount() {
  wsAuthRetryCount = 0;
}

export function bumpWsAuthRetryCount() {
  wsAuthRetryCount += 1;
  return wsAuthRetryCount;
}

export async function connectSocketWithSession(options: { forceNewToken?: boolean } = {}) {
  // 并发调用合并为一次（指纹异步期间重复 mount/effect 容易双开 WS）
  if (connectSessionFlight && !options.forceNewToken) return connectSessionFlight;

  const flight = (async () => {
    const [token, fingerprint] = await Promise.all([
      ensureSessionToken(options.forceNewToken),
      getBrowserFingerprint()
    ]);
    socket.auth = { token, fingerprint };
    await socket.connect();
    return token;
  })();

  if (!options.forceNewToken) {
    connectSessionFlight = flight.finally(() => {
      if (connectSessionFlight === flight) connectSessionFlight = null;
    });
    return connectSessionFlight;
  }
  return flight;
}

export async function joinIdentityPayload() {
  const fingerprint = await getBrowserFingerprint();
  return { ...ensurePlayerIdentity(), fingerprint };
}
