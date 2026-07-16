// Web Push Service Worker：只做一件事——收到 push 就弹通知。
// 不缓存任何资源（不是 PWA 离线方案），所以不需要 install/activate 里的预缓存逻辑。

self.addEventListener("push", (event) => {
  let payload = { title: "抖喵游戏屋", body: "你有一条新消息" };
  try {
    if (event.data) payload = { ...payload, ...event.data.json() };
  } catch {
    // 非 JSON payload 时用默认文案，不阻断通知展示。
  }

  event.waitUntil(
    (async () => {
      // 如果用户已经有一个可见的标签页在看这个站点，就不重复弹系统通知——
      // 那边的页面早就通过 WebSocket 实时收到同一个事件了（Level 1 会自己决定要不要弹）。
      // 服务端理论上已经用 player.Connected 过滤过，这里只是双重保险。
      const clientsList = await self.clients.matchAll({ type: "window", includeUncontrolled: true });
      const hasVisibleClient = clientsList.some((c) => c.visibilityState === "visible" && c.focused);
      if (hasVisibleClient) return;

      await self.registration.showNotification(payload.title, {
        body: payload.body,
        tag: payload.tag
      });
    })()
  );
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  event.waitUntil(
    (async () => {
      const clientsList = await self.clients.matchAll({ type: "window", includeUncontrolled: true });
      if (clientsList.length > 0) {
        await clientsList[0].focus();
        return;
      }
      await self.clients.openWindow("/");
    })()
  );
});
