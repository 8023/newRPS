// 聊天 store：按 scope（"" = 大厅，roomId = 房间）维护消息、分页游标与加载态。
// 首屏 / 历史走 chat:load / chat:loadOlder RPC（读 SQLite）；增量走 chat:new 推送。
// 组件用 useChat(scope) 订阅。
//
// authors：按 playerId 索引的**当前**玩家公开资料（服务端从内存 s.players 组装，
// 含离线玩家）。渲染时优先用调用方传入的在线名单（更实时），找不到再退回本 map。

import { useSyncExternalStore } from "react";
import { socket } from "../ws";
import { ask } from "./rpc";
import type { ChatMessage, PublicPlayer } from "../shared/types";

export type ChatScope = string; // "" 表示大厅

export type ChatState = {
  messages: ChatMessage[]; // 升序（旧→新）
  hasMore: boolean; // 是否还有更早历史
  loading: boolean;
  loadedOnce: boolean;
  /** playerId → 当前资料（加载历史时服务端附带；与在线名单合并使用） */
  authors: Record<string, PublicPlayer>;
};

const EMPTY: ChatState = { messages: [], hasMore: false, loading: false, loadedOnce: false, authors: {} };
// 内存最多保留的条数：防止长会话无限增长（更早的历史仍可从 DB 翻回）。
const MAX_LIVE = 400;
// 上翻历史后的硬顶（含实时消息）；超出时丢弃最旧条目，仍可再 loadOlder 从 DB 取。
const MAX_LOADED = 800;

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

function mergeAuthors(base: Record<string, PublicPlayer>, extra?: Record<string, PublicPlayer> | null) {
  if (!extra) return base;
  return { ...base, ...extra };
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

// 管理员软删除：按 id 从对应 scope 的本地消息里摘除（不清空整个 scope）。
// 后端 chatStore.older/recent 已经过滤掉 deleted=1 的行，这里只是让当前在线的老/新
// 访客立即看不到，不必等下次刷新历史。
socket.on("chat:deleted", (payload: { roomId?: string; ids?: string[] }) => {
  const ids = new Set(payload?.ids || []);
  if (ids.size === 0) return;
  const scope = payload?.roomId || "";
  const st = getState(scope);
  if (!st.messages.some((m) => ids.has(m.id))) return;
  setState(scope, { ...st, messages: st.messages.filter((m) => !ids.has(m.id)) });
});

// 重连后刷新已激活过的 scope 的最近一页（补上断线期间遗漏的消息）。
// 由 App 的 connect 处理器调用——App 会 socket.off("connect") 清空所有 connect 监听，
// 所以这里不自己注册 connect，避免被误清。
export function refreshActiveChats() {
  for (const [scope, st] of states) {
    if (st.loadedOnce) void loadChat(scope);
  }
}

type ChatLoadResult = {
  messages: ChatMessage[];
  hasMore: boolean;
  authors?: Record<string, PublicPlayer>;
};

/** 拉取最近一页（100 条）。 */
export async function loadChat(scope: ChatScope): Promise<void> {
  setState(scope, { ...getState(scope), loading: true });
  try {
    const res = await ask<ChatLoadResult>("chat:load", { roomId: scope });
    setState(scope, {
      messages: res.messages || [],
      hasMore: !!res.hasMore,
      loading: false,
      loadedOnce: true,
      authors: res.authors || {}
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
    const res = await ask<ChatLoadResult>("chat:loadOlder", {
      roomId: scope,
      beforeSeq
    });
    const older = res.messages || [];
    const cur = getState(scope);
    const existing = new Set(cur.messages.map((m) => m.id));
    const freshOlder = older.filter((m) => !existing.has(m.id));
    let merged = [...freshOlder, ...cur.messages];
    // 硬顶：保留较新的尾部，避免无限上翻撑爆内存。
    if (merged.length > MAX_LOADED) {
      merged = merged.slice(merged.length - MAX_LOADED);
    }
    setState(scope, {
      messages: merged,
      hasMore: !!res.hasMore,
      loading: false,
      loadedOnce: true,
      authors: mergeAuthors(cur.authors, res.authors)
    });
    return freshOlder.length;
  } catch {
    setState(scope, { ...getState(scope), loading: false });
    return 0;
  }
}

export function useChat(scope: ChatScope): ChatState {
  return useSyncExternalStore(
    (fn) => subscribe(scope, fn),
    () => getState(scope),
    () => EMPTY
  );
}

/** 输入框末尾插入一个 @提及（"@昵称 "），前置内容存在时补一个空格。 */
export function appendMentionText(existing: string, name: string): string {
  const base = existing.replace(/\s+$/, "");
  return (base ? base + " " : "") + "@" + name + " ";
}
