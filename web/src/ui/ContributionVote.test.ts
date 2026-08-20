import { describe, expect, it } from "vitest";
import { formatVoteStats } from "./ContributionVote";

describe("formatVoteStats", () => {
  it("shows empty copy when nobody voted", () => {
    expect(formatVoteStats({ hasVotes: false, voteCount: 0, displayRatio: null })).toBe("暂无评价 · 0 人评价");
  });

  it("shows ratio and real count when votes exist", () => {
    expect(formatVoteStats({ hasVotes: true, voteCount: 23, displayRatio: 87 })).toBe("点赞率 87% · 23 人评价");
  });
});
