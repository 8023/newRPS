// Web Push Service Worker：不缓存业务资源，只负责接管更新、展示通知和点击导航。

self.addEventListener("install", () => {
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(self.clients.claim());
});

self.addEventListener("push", (event) => {
  let payload = { title: "抖喵游戏屋", body: "你有一条新消息", url: "/" };
  try {
    if (event.data) payload = { ...payload, ...event.data.json() };
  } catch {
    // 非 JSON payload 时用默认文案，不阻断通知展示。
  }

  event.waitUntil(
    (async () => {
      await self.registration.showNotification(payload.title, {
        body: payload.body,
        tag: payload.tag,
        icon: "/icon.svg",
        badge: "/icon.svg",
        data: { url: payload.url || "/" }
      });
    })()
  );
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const targetUrl = new URL(event.notification.data?.url || "/", self.location.origin).href;
  event.waitUntil(
    (async () => {
      const clientsList = await self.clients.matchAll({ type: "window", includeUncontrolled: true });
      for (const client of clientsList) {
        if (new URL(client.url).origin !== self.location.origin) continue;
        if ("navigate" in client && client.url !== targetUrl) await client.navigate(targetUrl);
        await client.focus();
        return;
      }
      await self.clients.openWindow(targetUrl);
    })()
  );
});
