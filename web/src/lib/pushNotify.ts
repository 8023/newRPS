import { ask } from "./rpc";

export type PushPreferences = { mentionEnabled: boolean; turnEnabled: boolean; seatEnabled: boolean; bondEnabled: boolean };

export type PushSubscriptionStatus =
  | { state: "checking"; message: string }
  | { state: "active"; message: string }
  | { state: "stopped"; message: string }
  | { state: "permission-required"; message: string }
  | { state: "denied"; message: string }
  | { state: "unsupported"; message: string }
  | { state: "error"; message: string };

export type PushTestResult = {
  subscriptionCount: number;
  acceptedCount: number;
  failedCount: number;
  expiredCount: number;
};

const requiredPushProtocolVersion = 3;
const pushStoppedKey = "rps-push-stopped";
let prefs: PushPreferences = { mentionEnabled: false, turnEnabled: false, seatEnabled: false, bondEnabled: false };
// 串行化 ensure：后到的调用（尤其 force:true）必须带上自己的 options 再跑，
// 不能复用进行中的非 force Promise，否则「已停止」标记清不掉。
let ensureTail: Promise<unknown> = Promise.resolve();

export function setPushPreferencesCache(next: PushPreferences) {
  prefs = next;
}

export async function requestNotificationPermission(): Promise<NotificationPermission> {
  if (typeof Notification === "undefined") return "denied";
  if (Notification.permission !== "default") return Notification.permission;
  return Notification.requestPermission();
}

export async function fetchPushPreferences(): Promise<PushPreferences> {
  const result = await ask<{ mentionEnabled: boolean; turnEnabled: boolean; seatEnabled: boolean; bondEnabled: boolean }>("push:getPreferences", {});
  const next = {
    mentionEnabled: result.mentionEnabled,
    turnEnabled: result.turnEnabled,
    seatEnabled: result.seatEnabled,
    bondEnabled: result.bondEnabled
  };
  setPushPreferencesCache(next);
  return next;
}

export async function updatePushPreferences(partial: Partial<PushPreferences>): Promise<void> {
  await ask("push:updatePreferences", partial);
  setPushPreferencesCache({ ...prefs, ...partial });
}

function pushSupported() {
  return typeof window !== "undefined"
    && typeof navigator !== "undefined"
    && "serviceWorker" in navigator
    && "PushManager" in window
    && typeof Notification !== "undefined";
}

function stoppedLocally() {
  return typeof localStorage !== "undefined" && localStorage.getItem(pushStoppedKey) === "1";
}

function urlBase64ToUint8Array(base64String: string) {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, "+").replace(/_/g, "/");
  const rawData = atob(base64);
  const outputArray = new Uint8Array(rawData.length);
  for (let i = 0; i < rawData.length; i++) outputArray[i] = rawData.charCodeAt(i);
  return outputArray;
}

function sameApplicationServerKey(subscription: PushSubscription, expected: Uint8Array) {
  const current = subscription.options.applicationServerKey;
  if (!current) return false;
  const bytes = new Uint8Array(current);
  return bytes.length === expected.length && bytes.every((value, index) => value === expected[index]);
}

function errorMessage(error: unknown) {
  if (error instanceof Error && error.message.trim()) return error.message.trim();
  return "浏览器推送订阅失败";
}

export function currentPushSubscriptionStatus(): PushSubscriptionStatus {
  if (!pushSupported()) {
    return { state: "unsupported", message: "当前浏览器不支持 Web Push" };
  }
  if (Notification.permission === "denied") {
    return { state: "denied", message: "浏览器已拒绝通知权限" };
  }
  if (Notification.permission !== "granted") {
    return { state: "permission-required", message: "尚未授予通知权限" };
  }
  if (stoppedLocally()) {
    return { state: "stopped", message: "这台设备已停止推送" };
  }
  return { state: "checking", message: "正在检查设备订阅…" };
}

// 校验浏览器订阅、VAPID 公钥和服务器登记三层状态。登录恢复、打开设置及用户主动
// 开启时都会调用，因此清站点数据、换账号或 VAPID 轮换后可以自动修复。
async function performEnsurePushSubscription(options: { force?: boolean } = {}): Promise<PushSubscriptionStatus> {
  const initial = currentPushSubscriptionStatus();
  if (initial.state === "unsupported" || initial.state === "denied" || initial.state === "permission-required") {
    return initial;
  }
  if (initial.state === "stopped" && !options.force) return initial;
  if (options.force && typeof localStorage !== "undefined") localStorage.removeItem(pushStoppedKey);

  try {
    const registration = await navigator.serviceWorker.register("/push-sw-v3.js", { updateViaCache: "none" });
    await navigator.serviceWorker.ready;

    const res = await fetch("/api/push/vapid-key", { cache: "no-store" });
    if (!res.ok) throw new Error(`VAPID 公钥请求失败（HTTP ${res.status}）`);
    const data = await res.json() as { publicKey?: string; protocolVersion?: number };
    if (!data.publicKey) throw new Error("服务器未提供 VAPID 公钥");
    if (!data.protocolVersion || data.protocolVersion < requiredPushProtocolVersion) {
      throw new Error("当前运行的后端版本过旧，请重新构建并重启服务端");
    }
    const applicationServerKey = urlBase64ToUint8Array(data.publicKey);

    let subscription = await registration.pushManager.getSubscription();
    if (subscription && !sameApplicationServerKey(subscription, applicationServerKey)) {
      await ask("push:unsubscribe", { endpoint: subscription.endpoint }).catch(() => undefined);
      await subscription.unsubscribe();
      subscription = null;
    }
    if (!subscription) {
      subscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey
      });
    }

    const json = subscription.toJSON();
    if (!json.endpoint || !json.keys?.p256dh || !json.keys?.auth) {
      throw new Error("浏览器返回的订阅信息不完整");
    }
    await ask("push:subscribe", {
      endpoint: json.endpoint,
      keys: { p256dh: json.keys.p256dh, auth: json.keys.auth }
    });
    return { state: "active", message: "本设备推送已启用" };
  } catch (error) {
    const message = errorMessage(error);
    console.error("push subscription failed", error);
    return { state: "error", message };
  }
}

export function ensurePushSubscription(options: { force?: boolean } = {}): Promise<PushSubscriptionStatus> {
  const result = ensureTail.then(() => performEnsurePushSubscription(options));
  // 无论成败都接上下一环，避免一条失败把整条链永久 reject。
  ensureTail = result.then(() => undefined, () => undefined);
  return result;
}

export type DisablePushOptions = {
  /** 为 true（默认）时写入本机「已停止」标记；登出清订阅时应为 false，以免影响下次登录自动重订。 */
  markStopped?: boolean;
};

// 解除本机浏览器订阅与服务端登记。设置页「停止」与登出共用。
export async function disablePushSubscription(options: DisablePushOptions = {}): Promise<PushSubscriptionStatus> {
  const markStopped = options.markStopped !== false;
  if (markStopped && typeof localStorage !== "undefined") localStorage.setItem(pushStoppedKey, "1");
  if (!pushSupported()) {
    return markStopped
      ? { state: "stopped", message: "这台设备已停止推送" }
      : { state: "unsupported", message: "当前浏览器不支持 Web Push" };
  }
  try {
    const registration = await navigator.serviceWorker.getRegistration("/");
    const subscription = await registration?.pushManager.getSubscription();
    if (subscription) {
      let removeError: unknown;
      try {
        await ask("push:unsubscribe", { endpoint: subscription.endpoint });
      } catch (error) {
        removeError = error;
      }
      await subscription.unsubscribe();
      if (removeError) {
        console.warn("push server unsubscribe failed; local subscription removed", removeError);
      }
    }
    if (markStopped) return { state: "stopped", message: "这台设备已停止推送" };
    return { state: "permission-required", message: "本设备推送订阅已清除" };
  } catch (error) {
    const message = errorMessage(error);
    console.error("push unsubscribe failed", error);
    return { state: "error", message };
  }
}

export async function sendTestPush(): Promise<PushTestResult> {
  try {
    return await ask<PushTestResult>("push:test", {});
  } catch (error) {
    if (error instanceof Error && error.message === "unknown event") {
      throw new Error("当前运行的后端版本过旧，请重新构建并重启服务端");
    }
    throw error;
  }
}
