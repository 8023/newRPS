<script lang="ts">
  // 对局记录瀑布流：服务端语义是新→旧，展示上与聊天记录保持一致（旧的在上，新的在下）。
  // 源：ui/AppViews.tsx 里 Room 组件内联的 extraHistory/round-history 状态与 JSX 区块。
  import type { RoomSnapshot } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import { normalizeRoundHistoryItem, appendHistoryPage } from "../../lib/normalize";
  import { isNearScrollBottom, scrollToBottomSoon, stickChatToBottom } from "../../lib/uiHelpers";
  import { useMobileCollapse } from "../../lib/useMobileCollapse.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import CollapseToggle from "../shell/CollapseToggle.svelte";
  import RoundHistoryCard from "../shell/RoundHistoryCard.svelte";

  let { room, onOpenImage }: { room: RoomSnapshot; onOpenImage: (url: string) => void } = $props();

  const collapse = useMobileCollapse("roundHistory");

  let extraHistory = $state<RoomSnapshot["roundHistory"]>([]);
  let historyStick = $state(true);
  let historyLoading = $state(false);
  let historyListEl: HTMLDivElement | null = $state(null);
  const historyStickRef = { current: true };
  let historyLoadingFlag = false;

  $effect(() => {
    void room.id;
    extraHistory = [];
  });

  const visibleRoundHistory = $derived([...room.roundHistory, ...extraHistory.filter((item) => !room.roundHistory.some((fresh) => fresh.id === item.id))]);
  // visibleRoundHistory 是服务端语义的新→旧；展示上与聊天记录保持一致（旧的在上，新的在下），所以渲染时整体反转。
  const orderedRoundHistory = $derived([...visibleRoundHistory].reverse());
  const hasMoreHistory = $derived(visibleRoundHistory.length < room.roundHistoryTotal);

  // 瀑布流：每次向服务端补 5 条更早的对局记录（room:history 按 offset/limit 分页，offset 越大越旧）。
  async function loadMoreHistory(): Promise<number> {
    if (historyLoadingFlag || !hasMoreHistory) return 0;
    historyLoadingFlag = true;
    historyLoading = true;
    try {
      const result = await ask<{ items: RoomSnapshot["roundHistory"]; total: number }>("room:history", {
        roomId: room.id,
        offset: visibleRoundHistory.length,
        limit: 5
      });
      const items = (result.items || []).map(normalizeRoundHistoryItem);
      extraHistory = appendHistoryPage(extraHistory, items, room.roundHistory || []);
      return items.length;
    } catch (error) {
      uiStore.notify(error instanceof Error ? error.message : "加载对局记录失败");
      return 0;
    } finally {
      historyLoadingFlag = false;
      historyLoading = false;
    }
  }

  // 新记录到达时若已在底部则自动跟到最新；用户上滑查看历史时不会被打断。只依赖条数
  // （而不是整个数组）：见 plan.md §6.2，避免每次内容变化（如证明状态更新）都误触发滚动。
  const visibleCount = $derived(visibleRoundHistory.length);
  $effect(() => {
    const count = visibleCount;
    void count;
    if (historyListEl && historyStickRef.current) scrollToBottomSoon(historyListEl);
  });

  // 滚到顶部附近（展示上是「更旧的记录」）时瀑布流加载更早 5 条，并保持视口位置不跳动。
  async function handleHistoryScroll(event: Event) {
    const el = event.currentTarget as HTMLDivElement;
    const nextStick = isNearScrollBottom(el);
    if (historyStickRef.current !== nextStick) {
      historyStickRef.current = nextStick;
      historyStick = nextStick;
    }
    if (el.scrollTop < 200 && hasMoreHistory && !historyLoadingFlag) {
      const prevHeight = el.scrollHeight;
      const added = await loadMoreHistory();
      if (added > 0) {
        window.requestAnimationFrame(() => {
          el.scrollTop = el.scrollHeight - prevHeight;
        });
      }
    }
  }
</script>

<div class={`panel round-history ${collapse.collapsed ? "collapsed" : ""}`}>
  <h3 class="sticky-panel-title">
    📜 对局记录
    <span class="panel-title-actions">
      <span>{room.roundHistoryTotal || 0} 局</span>
      <CollapseToggle collapsed={collapse.collapsed} onToggle={collapse.toggle} label="对局记录" />
    </span>
  </h3>
  <div class={`mobile-collapsible-body ${collapse.collapsed ? "collapsed" : ""}`}>
    <div class="chat-scroll-shell">
      <div class="round-history-list" bind:this={historyListEl} onscroll={handleHistoryScroll}>
        {#if hasMoreHistory}<div class="chat-more-hint">{historyLoading ? "加载中…" : "↑ 上滑加载更早记录"}</div>{/if}
        {#each orderedRoundHistory as item (item.id)}
          <RoundHistoryCard {item} {onOpenImage} />
        {/each}
        {#if visibleRoundHistory.length === 0}<p class="empty">还没有对局记录</p>{/if}
      </div>
      {#if !historyStick && visibleRoundHistory.length > 0}
        <button type="button" class="chat-stick-button" onclick={() => stickChatToBottom(historyListEl, historyStickRef, (value) => (historyStick = value))}>
          ↓ 回到底部
        </button>
      {/if}
    </div>
  </div>
</div>
