<script module lang="ts">
  import type { ContributionItem, ContributionKind } from "../../shared/types";
  import { voteRatio } from "../contributeSeries";

  /** 投稿列表排序：time=更新时间，like=点赞率，completion=系列完成率。 */
  export type ContributionSortKey = "time" | "like" | "completion";
  export type ContributionSort = { key: ContributionSortKey; dir: "desc" | "asc" };

  /** 时间按钮、点赞/完成按钮各自独立维护自己当前所在的档位，互不覆盖——点了点赞按钮
      不会把时间按钮打回默认档，反之亦然，两个按钮的状态都持续保留。primary 记录最近
      点的是哪一个：它是主排序键，另一个按钮的当前档位退居决胜属性（仅在主排序值相同时
      起作用，比如两条更新时间完全一样的投稿改按点赞率决出先后）。primary 为 null 表示
      两个按钮都还没点过，维持调用方给的原始顺序（玩家端是服务端返回顺序，后台是待审
      优先的审核队列顺序 sortReviewQueue），不强制打乱。 */
  export type ContributionSortState = {
    time: ContributionSort;
    ratio: ContributionSort;
    primary: "time" | "ratio" | null;
  };

  const arrow: Record<ContributionSort["dir"], string> = { desc: "⬇", asc: "⬆" };
  const keyLabel: Record<ContributionSortKey, string> = { time: "时间", like: "点赞", completion: "完成" };

  const timeCycle: ContributionSort[] = [{ key: "time", dir: "desc" }, { key: "time", dir: "asc" }];

  /** 「点赞/完成」按钮的循环序。随机任务没有完成率（那是"系列走完了几步"的统计，
      见 punishment_series_run_stats），所以只在点赞的两个方向之间切换。 */
  function ratioCycle(kind: ContributionKind): ContributionSort[] {
    const like: ContributionSort[] = [{ key: "like", dir: "desc" }, { key: "like", dir: "asc" }];
    if (kind !== "series") return like;
    return [...like, { key: "completion", dir: "desc" }, { key: "completion", dir: "asc" }];
  }

  export function defaultContributionSortState(kind: ContributionKind): ContributionSortState {
    return { time: timeCycle[0], ratio: ratioCycle(kind)[0], primary: null };
  }

  /** 点一次按钮走到循环里的下一档。 */
  function nextInCycle(current: ContributionSort, cycle: ContributionSort[]): ContributionSort {
    const i = cycle.findIndex((s) => s.key === current.key && s.dir === current.dir);
    return cycle[(i + 1) % cycle.length];
  }

  function sortLabel(s: ContributionSort): string {
    return `${keyLabel[s.key]}${arrow[s.dir]}`;
  }

  /** 排序值：没有任何评价时点赞率按 0 算（与列表里显示的「点赞 0%」一致），
      系列完成率同理。 */
  function sortValue(item: ContributionItem, key: ContributionSortKey): number {
    if (key === "time") return item.updatedAt || 0;
    if (key === "like") return voteRatio(item.likeCount, item.downCount);
    return item.completion?.rate ?? 0;
  }

  export function sortContributionItems(items: ContributionItem[], state: ContributionSortState): ContributionItem[] {
    if (!state.primary) return items;
    const primary = state[state.primary];
    const secondary = state.primary === "time" ? state.ratio : state.time;
    const signPrimary = primary.dir === "desc" ? -1 : 1;
    const signSecondary = secondary.dir === "desc" ? -1 : 1;
    return [...items].sort((a, b) => {
      const diffPrimary = sortValue(a, primary.key) - sortValue(b, primary.key);
      if (diffPrimary !== 0) return diffPrimary * signPrimary;
      const diffSecondary = sortValue(a, secondary.key) - sortValue(b, secondary.key);
      if (diffSecondary !== 0) return diffSecondary * signSecondary;
      return (b.updatedAt || 0) - (a.updatedAt || 0);
    });
  }
</script>

<script lang="ts">
  // 排序按钮组 + 模糊查询框：玩家端「参与共建」与后台「共建审核」共用同一套控件，
  // 放在各自左栏列表的顶部。两个按钮不着色（不代表"当前选中/激活"这层含义——两个
  // 按钮任何时候都各自处于某个具体档位，没有"未选中"的空状态），按钮文案本身
  // （如「时间⬇」「点赞⬆」）就是当前档位的完整说明。
  // 源：ui/ContributionListControls.tsx:58-97。
  import type { Snippet } from "svelte";

  let { kind, sort, onSort, search, onSearch, searchPlaceholder, children }: {
    kind: ContributionKind;
    sort: ContributionSortState;
    onSort: (next: ContributionSortState) => void;
    search: string;
    onSearch: (value: string) => void;
    searchPlaceholder: string;
    children?: Snippet;
  } = $props();

  const ratio = $derived(ratioCycle(kind));
</script>

<div class="contribute-list-controls">
  <div class="contribute-sort-row">
    <button
      type="button"
      class="small"
      title="按更新时间排序（点击切换新→旧 / 旧→新）"
      onclick={() => onSort({ ...sort, time: nextInCycle(sort.time, timeCycle), primary: "time" })}
    >{sortLabel(sort.time)}</button>
    <button
      type="button"
      class="small"
      title={kind === "series" ? "按点赞率 / 完成率排序（点击依次切换高→低、低→高）" : "按点赞率排序（点击切换高→低 / 低→高）"}
      onclick={() => onSort({ ...sort, ratio: nextInCycle(sort.ratio, ratio), primary: "ratio" })}
    >{sortLabel(sort.ratio)}</button>
  </div>
  <input class="contribute-list-search" value={search} oninput={(e) => onSearch(e.currentTarget.value)} placeholder={searchPlaceholder} />
  {@render children?.()}
</div>
