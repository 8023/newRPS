import type { MutableRefObject } from "react";

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
