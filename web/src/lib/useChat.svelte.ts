// useChat 的 Svelte runes 包装：chatStore.ts 本体是框架无关的 Map+订阅者 Set，
// 这里只负责把「scope 变化 → 重新订阅」这层桥接成 $state，语义与原 React 版
// useSyncExternalStore 一致（scope 变化时立即用最新快照初始化，不留一帧旧数据）。
import { getState, subscribe, type ChatScope, type ChatState } from "./chatStore";

export function useChat(getScope: () => ChatScope) {
  let state = $state<ChatState>(getState(getScope()));

  $effect(() => {
    const scope = getScope();
    state = getState(scope);
    return subscribe(scope, () => {
      state = getState(scope);
    });
  });

  return {
    get value() {
      return state;
    }
  };
}
