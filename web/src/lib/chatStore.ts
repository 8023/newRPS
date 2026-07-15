// 聊天 store：按 scope（"" = 大厅，roomId = 房间）维护消息、分页游标与加载态。
// 首屏 / 历史走 chat:load / chat:loadOlder RPC（读 SQLite）；增量走 chat:new 推送。
// 组件用 useChat(scope) 订阅。

import { useSyncExternalStore } from "react";
import { socket } from "../ws";
import { ask } from "./rpc";
import type { ChatMessage } from "../shared/types";

export type ChatScope = string; // "" 表示大厅

export type ChatState = {
  messages: ChatMessage[]; // 升序（旧→新）
  hasMore: boolean; // 是否还有更早历史
  loading: boolean;
  loadedOnce: boolean;
};

const EMPTY: ChatState = { messages: [], hasMore: false, loading: false, loadedOnce: false };
// 内存最多保留的条数：防止长会话无限增长（更早的历史仍可从 DB 翻回）。
const MAX_LIVE = 400;

const states = new Map<ChatScope, ChatState>();
const listeners = new Map<ChatScope, Set<() => void>>();

function getState(scope: ChatScope): ChatState {
  return states.get(scope) || EMPTY;
}

function setState(scope: ChatScope, next: ChatState) {
  states.set(scope, next);
  listeners.get(scope)?.forEach((fn) => fn());
}

function subscribe(scope: ChatScope, fn: () => void): () => void {
  let set = listeners.get(scope);
  if (!set) {
    set = new Set();
    listeners.set(scope, set);
  }
  set.add(fn);
  return () => {
    set!.delete(fn);
  };
}

/** 最早一条持久化消息（有 seq）的 seq，作为向前翻页游标。 */
function oldestSeq(messages: ChatMessage[]): number | undefined {
  let min: number | undefined;
  for (const m of messages) {
    if (typeof m.seq === "number" && m.seq > 0 && (min === undefined || m.seq < min)) min = m.seq;
  }
  return min;
}

function appendLive(scope: ChatScope, msg: ChatMessage) {
  const st = getState(scope);
  if (st.messages.some((m) => m.id === msg.id)) return;
  const messages = [...st.messages, msg].slice(-MAX_LIVE);
  setState(scope, { ...st, messages });
}

// 增量：新消息（含系统提示，roomId 空为大厅）。
socket.on("chat:new", (msg: ChatMessage) => {
  if (!msg || !msg.id) return;
  appendLive(msg.roomId || "", msg);
});

// 管理员清空：重置对应 scope。
socket.on("chat:cleared", (payload: { roomId?: string }) => {
  setState(payload?.roomId || "", { ...EMPTY, loadedOnce: true });
});

// 重连后刷新已激活过的 scope 的最近一页（补上断线期间遗漏的消息）。
// 由 App 的 connect 处理器调用——App 会 socket.off("connect") 清空所有 connect 监听，
// 所以这里不自己注册 connect，避免被误清。
export function refreshActiveChats() {
  for (const [scope, st] of states) {
    if (st.loadedOnce) void loadChat(scope);
  }
}

/** 拉取最近一页（100 条）。 */
export async function loadChat(scope: ChatScope): Promise<void> {
  setState(scope, { ...getState(scope), loading: true });
  try {
    const res = await ask<{ messages: ChatMessage[]; hasMore: boolean }>("chat:load", { roomId: scope });
    setState(scope, {
      messages: res.messages || [],
      hasMore: !!res.hasMore,
      loading: false,
      loadedOnce: true
    });
  } catch {
    setState(scope, { ...getState(scope), loading: false, loadedOnce: true });
  }
}

/** 向前翻页（更早 100 条），返回新增条数（供滚动位置保持用）。 */
export async function loadOlderChat(scope: ChatScope): Promise<number> {
  const st = getState(scope);
  if (st.loading || !st.hasMore) return 0;
  const beforeSeq = oldestSeq(st.messages);
  if (!beforeSeq) return 0;
  setState(scope, { ...st, loading: true });
  try {
    const res = await ask<{ messages: ChatMessage[]; hasMore: boolean }>("chat:loadOlder", {
      roomId: scope,
      beforeSeq
    });
    const older = res.messages || [];
    const cur = getState(scope);
    const existing = new Set(cur.messages.map((m) => m.id));
    const merged = [...older.filter((m) => !existing.has(m.id)), ...cur.messages];
    setState(scope, { messages: merged, hasMore: !!res.hasMore, loading: false, loadedOnce: true });
    return older.filter((m) => !existing.has(m.id)).length;
  } catch {
    setState(scope, { ...getState(scope), loading: false });
    return 0;
  }
}

export function useChat(scope: ChatScope): ChatState {
  return useSyncExternalStore(
    (cb) => subscribe(scope, cb),
    () => getState(scope)
  );
}

/** 输入框末尾插入一个 @提及（"@昵称 "），前置内容存在时补一个空格。 */
export function appendMentionText(existing: string, name: string): string {
  const base = existing.replace(/\s+$/, "");
  return (base ? base + " " : "") + "@" + name + " ";
}
