import { getBrowserFingerprint } from "../fingerprint";
import { playerIdKey, playerSecretKey, punishmentTagPrefsKey, tokenKey } from "./constants";
import { socket } from "../ws";
import { ask } from "./rpc";
import { disablePushSubscription } from "./pushNotify";

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
    playerSecret = randomUuid();
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

/** 读取自动 player:join 所需的性别/阵营本地缓存。 */
export function readCachedJoinGender(): { genderId: string; factionId: string } {
  const genderId = localStorage.getItem("rps-online-gender") || "male";
  const factionId = localStorage.getItem("rps-online-faction") || "";
  return { genderId, factionId };
}

/** 把服务端确认过的名字/性别/阵营写回本地，供下次自动 join 使用。 */
export function cacheJoinProfile(player: { name: string; genderId?: string | null; genderLabel?: string | null; factionId?: string | null }) {
  // genderId 可能因 proto 省略空串而变成 undefined，必须写成 "" 而不是 String(undefined)==="undefined"。
  const genderId = typeof player.genderId === "string" ? player.genderId : "";
  localStorage.setItem("rps-online-name", player.name || "");
  localStorage.setItem("rps-online-gender", genderId);
  localStorage.setItem("rps-online-faction", player.factionId || "");
}

/**
 * 随机任务开房「标签三态偏好」（tagId -> include|exclude）：纯本地浏览器偏好，仅用于
 * 开房面板预填，不再随玩家档案落库/跨设备同步——同一浏览器下次开房记得住就够了。
 */
export function readPunishmentTagPrefs(): Record<string, string> {
  try {
    const raw = localStorage.getItem(punishmentTagPrefsKey);
    if (!raw) return {};
    const parsed = JSON.parse(raw);
    return parsed && typeof parsed === "object" ? parsed as Record<string, string> : {};
  } catch {
    return {};
  }
}

export function writePunishmentTagPrefs(prefs: Record<string, string>) {
  try {
    localStorage.setItem(punishmentTagPrefsKey, JSON.stringify(prefs));
  } catch {
    // 本地存储写入失败（隐私模式/容量满）不影响本次开房，只是下次预填不到。
  }
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

/**
 * 清掉本地缓存的长期身份（playerId/playerSecret）。用于服务端明确回应"玩家身份校验
 * 失败"时——这条本地凭据服务端已经不认，留着只会让下一次 player:join 用同一对
 * playerId/playerSecret 再挂一次。清掉后 ensurePlayerIdentity() 会在下次调用时
 * 生成一对全新的身份，相当于自动执行"清 localStorage 重新注册"这个此前只能手动做的操作。
 */
export function clearPlayerIdentity() {
  localStorage.removeItem(playerIdKey);
  localStorage.removeItem(playerSecretKey);
}

/** 标准 UUID 字符串（库内 / localStorage / identity:claim 使用的 playerId 形态）。 */
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

function looksLikeUuid(value: string): boolean {
  return UUID_RE.test(value);
}

/**
 * 把 UUID 压成 16 字节的 base64url（22 字符，无 padding），仅用于认领码展示。
 * 库内与 API 仍使用 UUID；转换可逆。
 */
export function uuidToClaimIdPart(uuid: string): string {
  const hex = uuid.replace(/-/g, "");
  if (hex.length !== 32 || !/^[0-9a-f]+$/i.test(hex)) return uuid;
  const bytes = new Uint8Array(16);
  for (let i = 0; i < 16; i++) {
    bytes[i] = parseInt(hex.slice(i * 2, i * 2 + 2), 16);
  }
  let bin = "";
  for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i]!);
  return btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

/** 认领码左侧 base64url → UUID；失败返回 null。 */
export function claimIdPartToUuid(part: string): string | null {
  if (looksLikeUuid(part)) return part.toLowerCase();
  try {
    const padded = part.replace(/-/g, "+").replace(/_/g, "/");
    const padLen = (4 - (padded.length % 4)) % 4;
    const bin = atob(padded + "=".repeat(padLen));
    if (bin.length !== 16) return null;
    const hex = Array.from(bin, (c) => c.charCodeAt(0).toString(16).padStart(2, "0")).join("");
    return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
  } catch {
    return null;
  }
}

/**
 * 认领码（展示/分享用）：`base64url(playerId 的 16 字节).claimKey`。
 * 服务端下发的仍是 UUID + 明文 claimKey；编码只发生在前端展示层。
 * 旧码 `uuid.claimKey` 仍可由 decodeClaimCode 解析。
 */
export function encodeClaimCode(playerId: string, claimKey: string) {
  const idPart = looksLikeUuid(playerId) ? uuidToClaimIdPart(playerId) : playerId;
  return `${idPart}.${claimKey}`;
}

/**
 * 解析认领码 → 服务端 identity:claim 所需的 UUID playerId + claimKey。
 * 支持新格式（base64url.id）与旧格式（完整 UUID）。
 */
export function decodeClaimCode(code: string): { playerId: string; claimKey: string } | null {
  const trimmed = code.trim();
  const idx = trimmed.lastIndexOf(".");
  if (idx <= 0 || idx === trimmed.length - 1) return null;
  const rawId = trimmed.slice(0, idx);
  const claimKey = trimmed.slice(idx + 1);
  if (!claimKey) return null;
  const playerId = claimIdPartToUuid(rawId);
  if (!playerId) return null;
  return { playerId, claimKey };
}

export async function fetchClaimKey(): Promise<{ playerId: string; claimKey: string }> {
  return ask("identity:showClaimKey", {});
}

export async function refreshClaimKey(): Promise<{ playerId: string; claimKey: string }> {
  return ask("identity:refreshClaimKey", {});
}

/**
 * 用另一台设备的认领码认领身份：成功后覆写本地 playerId/playerSecret，调用方需要接着
 * 重新 player:join——务必用返回的 name/genderId（认领回来的账号真实值），不要用输入框里
 * 随手打的名字/性别，否则会把认领回来的账号资料覆盖掉。
 */
export async function claimIdentity(code: string): Promise<{ playerId: string; playerSecret: string; name: string; genderId: string }> {
  const parsed = decodeClaimCode(code);
  if (!parsed) throw new Error("认领码格式不对");
  const result = await ask<{ playerId: string; playerSecret: string; name: string; genderId: string }>("identity:claim", parsed);
  localStorage.setItem(playerIdKey, result.playerId);
  localStorage.setItem(playerSecretKey, result.playerSecret);
  return result;
}

/** 登出：撤销当前设备在服务端的凭据，再清空本地身份/会话缓存。调用方负责后续跳转登录页。 */
export async function logout() {
  // 先解绑本机 Web Push，避免登出后仍收到旧账号通知（不写入「用户主动停止」标记）。
  try {
    await disablePushSubscription({ markStopped: false });
  } catch {
    // 推送清理失败不阻断登出。
  }
  const playerSecret = localStorage.getItem(playerSecretKey);
  try {
    if (playerSecret) await ask("identity:logout", { playerSecret });
  } catch {
    // 撤销失败也要继续清本地——用户明确要登出，不能因为一次网络错误就卡住。
  }
  localStorage.removeItem(playerIdKey);
  localStorage.removeItem(playerSecretKey);
  localStorage.removeItem(tokenKey);
  localStorage.removeItem("rps-online-name");
  localStorage.removeItem("rps-online-gender");
  localStorage.removeItem("rps-online-faction");
}
