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

/** stickRef 是一个 `{ current: boolean }` 可变盒子（与原 React useRef 同构的约定），
    调用方（ChatPanel）用它在 effect 之外读写「是否贴底」而不触发额外渲染。 */
export function stickChatToBottom(element: HTMLElement | null, stickRef: { current: boolean }, setSticking: (value: boolean) => void) {
  if (!element) return;
  stickRef.current = true;
  setSticking(true);
  scrollToBottomSoon(element);
}

const COLLAPSE_STORAGE_PREFIX = "rps-collapse:";

export function readCollapsedFromStorage(key: string): boolean {
  try {
    return localStorage.getItem(COLLAPSE_STORAGE_PREFIX + key) === "1";
  } catch {
    return false;
  }
}

export function writeCollapsedToStorage(key: string, value: boolean) {
  try {
    localStorage.setItem(COLLAPSE_STORAGE_PREFIX + key, value ? "1" : "0");
  } catch {
    // 隐私模式等场景下 localStorage 可能不可用，退化为仅本次会话记忆
  }
}
