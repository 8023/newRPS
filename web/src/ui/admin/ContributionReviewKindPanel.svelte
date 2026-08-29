<script lang="ts">
  // 源：ui/AdminContributionReview.tsx:210-250
  import { untrack } from "svelte";
  import type { AppConfig, ContributionItem } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import { kindLabel, sortReviewQueue, type JumpTarget } from "../../lib/contributionAdmin";
  import ContributionSubmissionReviewPanel from "./ContributionSubmissionReviewPanel.svelte";

  let { kind, config, password, onError, onChanged, jumpTarget }: {
    kind: "task" | "series";
    config: AppConfig;
    password: string;
    onError: (message: string) => void;
    onChanged: () => void;
    jumpTarget: ({ kind: "task" | "series" } & JumpTarget) | null;
  } = $props();

  let items = $state<ContributionItem[]>([]);

  async function reload() {
    try {
      const res = await ask<{ items?: ContributionItem[] } | ContributionItem[]>("admin:action", { action: "contributionList", status: "", kind, password });
      const list = Array.isArray(res) ? res : (res.items || []);
      items = sortReviewQueue(list);
    } catch (e) {
      onError(e instanceof Error ? e.message : "加载失败");
    }
  }

  // 原 React 版 deps 是 [kind, reloadNonce]：只在 tab 切换与显式刷新时重拉。
  // untrack 同 AdminContributionReview——reload() 同步读 password，不隔离就会被口令框逐字符触发。
  $effect(() => {
    void kind;
    untrack(() => reload());
  });

  function handleChanged() {
    void reload();
    onChanged();
  }
</script>

<ContributionSubmissionReviewPanel
  {items} {kind} {config} {password} {onError}
  onChanged={handleChanged}
  {jumpTarget}
  emptyText={`暂无${kindLabel[kind]}投稿`}
/>
