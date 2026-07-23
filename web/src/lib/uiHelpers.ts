import { type MutableRefObject, useState } from "react";

export function appendCappedUnique<T extends { id: string }>(items: T[], item: T, max: number) {
  if (items.some((old) => old.id === item.id)) return items;
  return [...items, item].slice(-max);
}

export function prependCappedUnique<T extends { id: string }>(items: T[], item: T, max: number) {
  if (items.some((old) => old.id === item.id)) return items;
  return [item, ...items].slice(0, max);
}

export function isNearScrollBottom(element: HTMLElement, threshold = 72) {
  return element.scrollHeight - element.scrollTop - element.clientHeight < threshold;
}

export function scrollToBottomSoon(element: HTMLElement) {
  window.requestAnimationFrame(() => {
    element.scrollTop = element.scrollHeight;
  });
}

export function stickChatToBottom(element: HTMLElement | null, stickRef: MutableRefObject<boolean>, setSticking: (value: boolean) => void) {
  if (!element) return;
  stickRef.current = true;
  setSticking(true);
  scrollToBottomSoon(element);
}

const COLLAPSE_STORAGE_PREFIX = "rps-collapse:";

function readCollapsedFromStorage(key: string): boolean {
  try {
    return localStorage.getItem(COLLAPSE_STORAGE_PREFIX + key) === "1";
  } catch {
    return false;
  }
}

// 手机端"折叠某个模块"的记忆偏好：只影响手机端展示（由 CSS 在对应断点内生效），
// 默认展开；偏好按 key 存 localStorage，跨大厅/房间重进也能延续。
export function useMobileCollapse(key: string) {
  const [collapsed, setCollapsed] = useState(() => readCollapsedFromStorage(key));

  function toggle() {
    setCollapsed((old) => {
      const next = !old;
      try {
        localStorage.setItem(COLLAPSE_STORAGE_PREFIX + key, next ? "1" : "0");
      } catch {
        // 隐私模式等场景下 localStorage 可能不可用，退化为仅本次会话记忆
      }
      return next;
    });
  }

  return { collapsed, toggle };
}
