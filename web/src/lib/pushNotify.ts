import { ask } from "./rpc";

export type PushPreferences = { mentionEnabled: boolean; turnEnabled: boolean; seatEnabled: boolean };

export type PushSubscriptionStatus =
  | { state: "checking"; message: string }
  | { state: "active"; message: string }
  | { state: "stopped"; message: string }
  | { state: "permission-required"; message: string }
  | { state: "denied"; message: string }
  | { state: "unsupported"; message: string }
  | { state: "error"; message: string };

const pushStoppedKey = "rps-push-stopped";
let prefs: PushPreferences = { mentionEnabled: false, turnEnabled: false, seatEnabled: false };
let ensureInFlight: Promise<PushSubscriptionStatus> | null = null;

export function setPushPreferencesCache(next: PushPreferences) {
  prefs = next;
}

export function getPushPreferencesCache() {
  return prefs;
}

export async function requestNotificationPermission(): Promise<NotificationPermission> {
  if (typeof Notification === "undefined") return "denied";
  if (Notification.permission !== "default") return Notification.permission;
  return Notification.requestPermission();
}

export async function fetchPushPreferences(): Promise<PushPreferences> {
  const result = await ask<{ mentionEnabled: boolean; turnEnabled: boolean; seatEnabled: boolean }>("push:getPreferences", {});
  const next = { mentionEnabled: result.mentionEnabled, turnEnabled: result.turnEnabled, seatEnabled: result.seatEnabled };
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
    const registration = await navigator.serviceWorker.register("/sw.js", { updateViaCache: "none" });
    await navigator.serviceWorker.ready;

    const res = await fetch("/api/push/vapid-key", { cache: "no-store" });
    if (!res.ok) throw new Error(`VAPID 公钥请求失败（HTTP ${res.status}）`);
    const data = await res.json() as { publicKey?: string };
    if (!data.publicKey) throw new Error("服务器未提供 VAPID 公钥");
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
  if (ensureInFlight) return ensureInFlight;
  ensureInFlight = performEnsurePushSubscription(options).finally(() => {
    ensureInFlight = null;
  });
  return ensureInFlight;
}

export async function disablePushSubscription(): Promise<PushSubscriptionStatus> {
  if (typeof localStorage !== "undefined") localStorage.setItem(pushStoppedKey, "1");
  if (!pushSupported()) return { state: "unsupported", message: "当前浏览器不支持 Web Push" };
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
    return { state: "stopped", message: "这台设备已停止推送" };
  } catch (error) {
    const message = errorMessage(error);
    console.error("push unsubscribe failed", error);
    return { state: "error", message };
  }
}

export async function sendTestPush(): Promise<void> {
  await ask("push:test", {});
}
