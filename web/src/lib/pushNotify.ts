import { ask } from "./rpc";

// Level 1（页面还开着、只是切了后台/最小化）：直接用 Notification API，不需要 Service Worker，
// 不需要订阅。服务端只有在这个玩家完全断线（player.Connected===false）时才会走 Level 2 Web
// Push——只要 WS 还连着，判断"要不要弹"这件事完全在前端本地做，不用服务端参与，也不会和
// Level 2 重复弹（见 shouldPush 的服务端注释）。

export type PushPreferences = { mentionEnabled: boolean; turnEnabled: boolean; seatEnabled: boolean };

let prefs: PushPreferences = { mentionEnabled: false, turnEnabled: false, seatEnabled: false };
let meId = "";

export function setPushPreferencesCache(next: PushPreferences) {
  prefs = next;
}

export function getPushPreferencesCache() {
  return prefs;
}

export function setPushMeId(id: string) {
  meId = id;
}

export function getPushMeId() {
  return meId;
}

function canShowNotification() {
  return typeof Notification !== "undefined" && Notification.permission === "granted" && document.hidden;
}

function showNotification(title: string, body: string, tag: string) {
  if (!canShowNotification()) return;
  try {
    // eslint-disable-next-line no-new
    new Notification(title, { body, tag });
  } catch {
    // 部分浏览器/权限状态下 new Notification() 会抛，静默忽略即可（不影响页面内其它功能）。
  }
}

export function notifyMentionIfHidden(author: string, text: string) {
  if (!prefs.mentionEnabled) return;
  showNotification("有人 @ 了你", `${author}：${text}`, "mention");
}

export function notifyTurnIfHidden() {
  if (!prefs.turnEnabled) return;
  showNotification("轮到你了", "对方已经行动，轮到你出招/落子了。", "turn");
}

export function notifySeatIfHidden() {
  if (!prefs.seatEnabled) return;
  showNotification("你的房间来人了", "有玩家坐上了对面的战斗席，快回来看看吧。", "seat");
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

// ---- Level 2：Web Push 订阅（可选，权限允许时才会用到） ----

function urlBase64ToUint8Array(base64String: string) {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, "+").replace(/_/g, "/");
  const rawData = atob(base64);
  const outputArray = new Uint8Array(rawData.length);
  for (let i = 0; i < rawData.length; i++) outputArray[i] = rawData.charCodeAt(i);
  return outputArray;
}

export async function ensurePushSubscription(): Promise<boolean> {
  if (typeof navigator === "undefined" || !("serviceWorker" in navigator) || !("PushManager" in window)) return false;
  if (Notification.permission !== "granted") return false;
  try {
    const registration = await navigator.serviceWorker.register("/sw.js");
    await navigator.serviceWorker.ready;
    let subscription = await registration.pushManager.getSubscription();
    if (!subscription) {
      const res = await fetch("/api/push/vapid-key");
      const data = await res.json();
      if (!data.publicKey) return false;
      subscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(data.publicKey)
      });
    }
    const json = subscription.toJSON();
    if (!json.endpoint || !json.keys?.p256dh || !json.keys?.auth) return false;
    await ask("push:subscribe", { endpoint: json.endpoint, keys: { p256dh: json.keys.p256dh, auth: json.keys.auth } });
    return true;
  } catch {
    return false;
  }
}

export async function disablePushSubscription(): Promise<void> {
  if (typeof navigator === "undefined" || !("serviceWorker" in navigator)) return;
  try {
    const registration = await navigator.serviceWorker.getRegistration();
    const subscription = await registration?.pushManager.getSubscription();
    if (subscription) {
      await ask("push:unsubscribe", { endpoint: subscription.endpoint }).catch(() => undefined);
      await subscription.unsubscribe();
    }
  } catch {
    // 取消订阅失败不影响本地偏好关闭，静默忽略。
  }
}
