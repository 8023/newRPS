<script lang="ts">
  /**
   * 「聊天管理」面板：按用户名/聊天内容/房间名（或仅大厅）检索历史消息，多选后批量软删除
   * /恢复。取代旧版"一键清空大厅/房间聊天"——只能对检索出的具体违规留言操作，删除是软删除
   * （数据库保留、普通玩家读不到），删除后端会主动推 chat:deleted 通知在线客户端摘除本地视图。
   * 源：ui/AdminViews.tsx:1481-1724。onError/action 原为 props，现直接用 uiStore/adminStore。
   */
  import type { PublicPlayer } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import { displayPlayerName } from "../../lib/playerDisplay";
  import {
    type AdminChatMessage, chatRefKey, formatAdminChatTime, adminChatAuthorName, adminChatScopeLabel
  } from "../../lib/adminHelpers";
  import { adminStore } from "../../lib/stores/adminStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import AdminChatSpeaker from "./AdminChatSpeaker.svelte";

  let author = $state("");
  let text = $state("");
  let room = $state("");
  let lobbyOnly = $state(false);
  let includeDeleted = $state(false);
  let messages = $state<AdminChatMessage[]>([]);
  let authors = $state<Record<string, PublicPlayer>>({});
  let hasMore = $state(false);
  let loading = $state(false);
  let searched = $state(false);
  let searchError = $state<string | null>(null);
  let selected = $state<Set<string>>(new Set());
  let searchRequestGen = 0;

  async function runSearch(offset: number, append: boolean) {
    const gen = ++searchRequestGen;
    loading = true;
    searchError = null;
    try {
      const res = await ask<{ messages?: AdminChatMessage[]; hasMore?: boolean; authors?: Record<string, PublicPlayer> }>("admin:chatSearch", {
        author, text, room: lobbyOnly ? "" : room, lobbyOnly, includeDeleted, limit: 50, offset
      });
      if (gen !== searchRequestGen) return;
      const list = res.messages || [];
      const nextAuthors = res.authors || {};
      messages = append ? [...messages, ...list] : list;
      authors = append ? { ...authors, ...nextAuthors } : nextAuthors;
      hasMore = !!res.hasMore;
      searched = true;
      if (!append) selected = new Set();
    } catch (error) {
      if (gen !== searchRequestGen) return;
      const message = error instanceof Error ? error.message : "检索失败";
      searchError = message;
      searched = true;
      uiStore.notify(message);
    } finally {
      if (gen === searchRequestGen) loading = false;
    }
  }

  // 进入页签时拉最近消息；之后只在点「检索」、切换范围开关或加载更多时再请求。
  $effect(() => {
    void runSearch(0, false);
  });

  let skipToggleSearch = true;
  $effect(() => {
    void lobbyOnly;
    void includeDeleted;
    if (skipToggleSearch) {
      skipToggleSearch = false;
      return;
    }
    void runSearch(0, false);
  });

  function toggleSelect(key: string) {
    const next = new Set(selected);
    if (next.has(key)) next.delete(key); else next.add(key);
    selected = next;
  }

  function toggleSelectAll() {
    selected = selected.size === messages.length && messages.length > 0 ? new Set() : new Set(messages.map(chatRefKey));
  }

  async function applyToRefs(actionName: "chatSoftDelete" | "chatRestore", refs: { roomId: string; id: string }[]) {
    if (refs.length === 0) {
      uiStore.notify("请先勾选要操作的消息");
      return;
    }
    if (await adminStore.action(actionName, { refs })) {
      uiStore.notify(actionName === "chatSoftDelete" ? `已删除 ${refs.length} 条` : `已恢复 ${refs.length} 条`);
      await runSearch(0, false);
    }
  }

  const selectedRefs = $derived(messages.filter((m) => selected.has(chatRefKey(m))).map((m) => ({ roomId: m.roomId || "", id: m.id })));
  const allSelected = $derived(messages.length > 0 && selected.size === messages.length);
  const resultHint = $derived(
    loading && !searched
      ? "检索中…"
      : searchError
        ? "检索失败"
        : hasMore
          ? `已显示 ${messages.length} 条 · 还有更多`
          : `${messages.length} 条`
  );
</script>

<div class="admin-chat-manager">
  <form class="admin-chat-filter-card" onsubmit={(event) => { event.preventDefault(); void runSearch(0, false); }}>
    <div class="admin-chat-filter-grid">
      <label class="field-label">
        <span>用户名</span>
        <input value={author} oninput={(event) => (author = event.currentTarget.value)} placeholder="按发言人昵称" autocomplete="off" />
      </label>
      <label class="field-label">
        <span>聊天内容</span>
        <input value={text} oninput={(event) => (text = event.currentTarget.value)} placeholder="按消息关键字" autocomplete="off" />
      </label>
      <label class="field-label">
        <span>房间名</span>
        <input value={room} oninput={(event) => (room = event.currentTarget.value)} placeholder={lobbyOnly ? "只看大厅时不可用" : "留空则不限房间"} disabled={lobbyOnly} autocomplete="off" />
      </label>
    </div>
    <div class="admin-chat-filter-actions">
      <div class="admin-player-filters" role="group" aria-label="检索范围">
        <button
          type="button"
          class={`admin-filter-btn${lobbyOnly ? " active" : ""}`}
          aria-pressed={lobbyOnly}
          onclick={() => { const next = !lobbyOnly; lobbyOnly = next; if (next) room = ""; }}
        >只看大厅</button>
        <button type="button" class={`admin-filter-btn${includeDeleted ? " active" : ""}`} aria-pressed={includeDeleted} onclick={() => (includeDeleted = !includeDeleted)}>显示已删除</button>
      </div>
      <button class="primary" type="submit" disabled={loading}>{loading ? "检索中…" : "检索"}</button>
    </div>
  </form>

  <div class="admin-list-section admin-chat-list-section">
    <div class="admin-list-heading">
      <h3>检索结果</h3>
      <span>{resultHint}</span>
    </div>
    <div class="admin-chat-bulk-row">
      <label class="admin-chat-select-all">
        <input type="checkbox" checked={allSelected} onchange={toggleSelectAll} disabled={messages.length === 0} />
        全选
        {#if messages.length > 0}<em>已选 {selected.size}/{messages.length}</em>{/if}
      </label>
      <div class="admin-chat-bulk-actions">
        <button type="button" class="danger-button small" onclick={() => applyToRefs("chatSoftDelete", selectedRefs)} disabled={selected.size === 0}>删除所选</button>
        <button type="button" class="small" onclick={() => applyToRefs("chatRestore", selectedRefs)} disabled={selected.size === 0}>恢复所选</button>
      </div>
    </div>
    <div class="admin-chat-list" role="list">
      {#each messages as m (chatRefKey(m))}
        {@const key = chatRefKey(m)}
        {@const isSelected = selected.has(key)}
        {@const isLobby = !m.roomId}
        {@const player = m.playerId ? authors[m.playerId] : undefined}
        {@const authorLabel = player ? displayPlayerName(player) : adminChatAuthorName(m.author)}
        <div class={`admin-chat-row${m.deleted ? " deleted" : ""}${isSelected ? " selected" : ""}`} role="listitem" onclick={() => toggleSelect(key)}>
          <input type="checkbox" checked={isSelected} onchange={() => toggleSelect(key)} onclick={(event) => event.stopPropagation()} aria-label={`选择 ${authorLabel} 的消息`} />
          <div class="admin-chat-row-body">
            <div class="admin-chat-row-meta">
              <AdminChatSpeaker {player} fallbackName={authorLabel} {isLobby} scopeLabel={adminChatScopeLabel(m)} />
              {#if m.deleted}<span class="admin-chat-deleted-chip">已删除</span>{/if}
              <time class="admin-chat-time" datetime={m.at ? new Date(m.at).toISOString() : undefined}>{formatAdminChatTime(m.at)}</time>
            </div>
            <p>{m.text || "（空消息）"}</p>
          </div>
          <div class="admin-chat-row-actions">
            {#if m.deleted}
              <button type="button" class="small" onclick={(event) => { event.stopPropagation(); void applyToRefs("chatRestore", [{ roomId: m.roomId || "", id: m.id }]); }}>恢复</button>
            {:else}
              <button type="button" class="danger-button small" onclick={(event) => { event.stopPropagation(); void applyToRefs("chatSoftDelete", [{ roomId: m.roomId || "", id: m.id }]); }}>删除</button>
            {/if}
          </div>
        </div>
      {/each}
      {#if !loading && messages.length === 0}
        <p class="empty">{searchError ? `检索失败：${searchError}` : searched ? "没有符合条件的聊天消息" : "正在加载最近消息…"}</p>
      {/if}
    </div>
    {#if hasMore}
      <button type="button" class="admin-chat-more" onclick={() => runSearch(messages.length, true)} disabled={loading}>{loading ? "加载中…" : "加载更多"}</button>
    {/if}
  </div>
</div>
