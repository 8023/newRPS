import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { formatVoteStats } from "./ContributionVote.svelte";

describe("formatVoteStats", () => {
  it("shows empty copy when nobody voted", () => {
    expect(formatVoteStats({ hasVotes: false, voteCount: 0, displayRatio: null })).toBe("暂无评价 · 0 人评价");
  });

  it("shows ratio and real count when votes exist", () => {
    expect(formatVoteStats({ hasVotes: true, voteCount: 23, displayRatio: 87 })).toBe("23 人评价 · 点赞 87%");
  });
});

// 钉住 stale-state 回归：ContributionVote 用 $state(untrack(() => card)) 播种本地 current，
// 只在首次挂载时生效；父组件若传入新 card，必须显式用 $effect 同步（Svelte 5 版的等价物是
// React useState(card) 播种 + useEffect(() => setCurrent(card), [card]) 同步）。
describe("ContributionVote markup", () => {
  it("keeps current state synced to the card prop via an effect", () => {
    const src = readFileSync(new URL("./ContributionVote.svelte", import.meta.url), "utf8");
    expect(src).toContain("$state(untrack(() => card))");
    expect(src).toMatch(/\$effect\(\(\) => \{\s*current = card;\s*\}\);/);
  });
});
