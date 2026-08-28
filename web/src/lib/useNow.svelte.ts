// 局部时钟：仅在 enabled 时按 interval 刷新，避免挂载常驻的组件（大厅聊天面板等）
// 无谓地每秒重渲染。enabled/intervalMs 用 getter 传入以便随调用方状态变化。
import { socket } from "../ws";

export function useNow(getIntervalMs: () => number = () => 1000, getEnabled: () => boolean = () => true) {
  let now = $state(Date.now());

  $effect(() => {
    if (!getEnabled()) return;
    now = Date.now();
    const id = window.setInterval(() => { now = Date.now(); }, getIntervalMs());
    return () => window.clearInterval(id);
  });

  return {
    get value() {
      return now;
    }
  };
}

/** 棋钟使用服务端时间轴，避免玩家设备系统时间不同造成显示偏长或提前归零。 */
export function useServerNow(getIntervalMs: () => number = () => 1000, getEnabled: () => boolean = () => true) {
  let now = $state(socket.serverNow());

  $effect(() => {
    if (!getEnabled()) return;
    now = socket.serverNow();
    const id = window.setInterval(() => { now = socket.serverNow(); }, getIntervalMs());
    return () => window.clearInterval(id);
  });

  return {
    get value() {
      return now;
    }
  };
}
