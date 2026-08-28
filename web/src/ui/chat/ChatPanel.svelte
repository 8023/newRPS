<script lang="ts">
  // ChatPanel：房间聊天与大厅聊天共用。scope="" 为大厅，否则为 roomId。
  // 首屏/历史走 chatStore（chat:load/loadOlder，读 SQLite），增量由 chatStore 监听 chat:new。
  // 滚到顶部瀑布流加载更早 100 条并保持滚动位置；点头像 @人；@到自己的气泡高亮。
  // 源：ui/AppViews.tsx:2989-3132。me/onError 原为 props，现直接读 sessionStore/uiStore。
  import type { PublicPlayer } from "../../shared/types";
  import { socket } from "../../ws";
  import { ask } from "../../lib/rpc";
  import { appendMentionText, loadChat, loadOlderChat } from "../../lib/chatStore";
  import { useChat } from "../../lib/useChat.svelte";
  import { useNow } from "../../lib/useNow.svelte";
  import { isNearScrollBottom, scrollToBottomSoon, stickChatToBottom } from "../../lib/uiHelpers";
  import { mentionLabel } from "../../lib/playerDisplay";
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import ChatBubble from "./ChatBubble.svelte";

  let {
    scope, players, placeholder, emptyText, subscribeLobbyChannel = false, messagesClass
  }: {
    scope: string;
    // 发言人当前资料：优先用调用方传入的在线/房间名单（更实时），再合并 chat:load 带回的
    // authors（服务端按 playerId 从内存玩家表取的当前资料，含离线玩家）。
    players: PublicPlayer[];
    placeholder?: string;
    emptyText?: string;
    subscribeLobbyChannel?: boolean;
    messagesClass?: string;
  } = $props();

  const me = $derived(sessionStore.me!.player);
  const chat = useChat(() => scope);

  const playersById = $derived.by(() => {
    const map = new Map<string, PublicPlayer>();
    // 先放历史加载的 authors，再用在线名单覆盖（同 id 以最新在线快照为准）。
    for (const [id, player] of Object.entries(chat.value.authors || {})) map.set(id, player);
    for (const player of players) map.set(player.id, player);
    return map;
  });

  let text = $state("");
  let pendingMentions: Array<{ playerId: string; name: string }> = [];
  let listEl: HTMLDivElement | null = $state(null);
  const stickRef = { current: true };
  let stick = $state(true);
  // 只在确实有带 expiresAt 的消息时才起 1Hz 定时器，避免大厅聊天面板（常驻挂载）无谓地
  // 每秒重渲染整个面板。
  const now = useNow(() => 1000, () => chat.value.messages.some((m) => m.expiresAt));

  $effect(() => {
    void scope;
    stickRef.current = true;
    stick = true;
    void loadChat(scope);
  });

  // 房间内的「大厅」tab：加入大厅聊天频道以收实时增量，卸载时退出。
  $effect(() => {
    if (!subscribeLobbyChannel) return;
    void scope;
    const subscribe = () => { ask("lobby:suggestions:subscribe", {}).catch(() => undefined); };
    subscribe();
    // WS 断线重连是全新连接，服务端不会记得这个订阅制频道，必须重新 subscribe，
    // 否则重连后收不到大厅聊天的实时增量，要等切一次 tab 或整页刷新才会恢复。
    socket.on("connect", subscribe);
    return () => {
      socket.off("connect", subscribe);
      ask("lobby:suggestions:unsubscribe", {}).catch(() => undefined);
    };
  });

  const visible = $derived(chat.value.messages.filter((m) => !m.expiresAt || m.expiresAt > now.value));
  // 只在「可见消息条数」真正变化时才滚底：直接依赖 visible（数组）会让 now 每秒重算过滤结果
  // 时也触发（哪怕条数没变），这里改成读一个数字派生值，Svelte 只在其输出值变化时才使
  // 下游失效，天然复现原 React 版 deps=[visible.length] 的窄依赖（见 plan.md §6.2）。
  const visibleCount = $derived(visible.length);

  $effect(() => {
    const count = visibleCount;
    void count;
    if (listEl && stickRef.current) scrollToBottomSoon(listEl);
  });

  async function handleScroll(event: Event) {
    const el = event.currentTarget as HTMLDivElement;
    const nextStick = isNearScrollBottom(el);
    if (stickRef.current !== nextStick) {
      stickRef.current = nextStick;
      stick = nextStick;
    }
    // 滚到顶部附近且还有更早历史：瀑布流加载并保持视口位置。
    if (el.scrollTop < 200 && chat.value.hasMore && !chat.value.loading) {
      const prevHeight = el.scrollHeight;
      const added = await loadOlderChat(scope);
      if (added > 0) {
        window.requestAnimationFrame(() => {
          el.scrollTop = el.scrollHeight - prevHeight;
        });
      }
    }
  }

  function insertMention(player?: PublicPlayer) {
    if (!player) return;
    const name = mentionLabel(player);
    if (!name) return;
    text = appendMentionText(text, name);
    pendingMentions.push({ playerId: player.id, name });
  }

  async function send() {
    const value = text.trim();
    if (!value) return;
    // 仅保留文本中仍存在 "@昵称" 的提及（用户可能删掉了某个 @）。
    const mentions = Array.from(
      new Set(pendingMentions.filter((m) => value.includes("@" + m.name)).map((m) => m.playerId))
    );
    text = "";
    pendingMentions = [];
    try {
      await ask("chat:send", { roomId: scope, text: value, mentions });
    } catch (error) {
      text = value;
      uiStore.notify(error instanceof Error ? error.message : "发送失败");
    }
  }
</script>

<div class="chat-scroll-shell">
  <div class={`messages ${messagesClass || ""}`} bind:this={listEl} onscroll={handleScroll}>
    {#if chat.value.hasMore}<div class="chat-more-hint">{chat.value.loading ? "加载中…" : "↑ 上滑加载更早消息"}</div>{/if}
    {#each visible as item (item.id)}
      <ChatBubble message={item} {me} author={item.playerId === me.id ? me : playersById.get(item.playerId)} onMention={insertMention} />
    {/each}
    {#if visible.length === 0}<p class="empty">{emptyText || "还没有消息"}</p>{/if}
  </div>
  {#if !stick && visible.length > 0}
    <button type="button" class="chat-stick-button" onclick={() => stickChatToBottom(listEl, stickRef, (value) => (stick = value))}>
      ↓ 回到底部
    </button>
  {/if}
</div>
<div class="send-row">
  <input
    value={text}
    oninput={(event) => (text = event.currentTarget.value)}
    onkeydown={(event) => { if (event.key === "Enter") send(); }}
    placeholder={placeholder || "发一句话..."}
  />
  <button onclick={send}>发送</button>
</div>
