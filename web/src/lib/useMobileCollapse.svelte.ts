// 手机端"折叠某个模块"的记忆偏好：只影响手机端展示（由 CSS 在对应断点内生效），
// 默认展开；偏好按 key 存 localStorage，跨大厅/房间重进也能延续。
import { readCollapsedFromStorage, writeCollapsedToStorage } from "./uiHelpers";

export function useMobileCollapse(key: string) {
  let collapsed = $state(readCollapsedFromStorage(key));

  function toggle() {
    collapsed = !collapsed;
    writeCollapsedToStorage(key, collapsed);
  }

  return {
    get collapsed() {
      return collapsed;
    },
    toggle
  };
}
