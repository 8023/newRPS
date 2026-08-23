import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { formatVoteStats } from "./ContributionVote";

describe("formatVoteStats", () => {
  it("shows empty copy when nobody voted", () => {
    expect(formatVoteStats({ hasVotes: false, voteCount: 0, displayRatio: null })).toBe("暂无评价 · 0 人评价");
  });

  it("shows ratio and real count when votes exist", () => {
    expect(formatVoteStats({ hasVotes: true, voteCount: 23, displayRatio: 87 })).toBe("23 人评价 · 点赞 87%");
  });
});

// 钉住 stale-state 回归：ContributionVote 用 useState(card) 播种本地 current，只在首次
// 挂载时生效；父组件若传入新 card，必须显式同步。这里不引入 DOM 测试依赖，直接钉住 effect。
describe("ContributionVote markup", () => {
  it("keeps current state synced to the card prop via an effect", () => {
    const src = readFileSync(new URL("./ContributionVote.tsx", import.meta.url), "utf8");
    expect(src).toContain("useState(card)");
    expect(src).toMatch(/useEffect\(\(\) => \{\s*setCurrent\(card\);\s*\}, \[card\]\);/);
  });
});
