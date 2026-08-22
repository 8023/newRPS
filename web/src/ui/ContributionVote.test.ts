import { readFileSync } from "node:fs";
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

// TestVoteUI_neverUpdatesFromParent 钉住一个真实发生过的 stale-state 回归：ContributionVote
// 用 useState(card) 播种本地 current，只在组件首次挂载时生效。它在证明还没审核通过（这时
// canVote 恒定是 false）就已经随任务卡片一起挂载了（见 ContributionVoteLazy 的调用点），
// 之后父组件因为 proofStatus 变化重新请求 votePreview、拿到 canVote 变成 true 的新 card
// 传下来时，React 只是给同一个组件实例换 props，不会重新跑 useState 初始化——current 永远
// 停在挂载那一刻的旧值，赞/踩按钮实际上永远不会出现（端到端验证过：两个真实浏览器完整走完
// 创建房间→对局→提交证明→审核通过，评价按钮全程不出现）。修复是加一个
// useEffect(() => setCurrent(card), [card]) 把 prop 变化同步进本地 state。这里不跑真实
// React 渲染（项目未引入 @testing-library/react/jsdom），改为源码断言钉住这个 effect
// 还在——直接删掉它是造成回归的确切改法，之前发生过不止一次。
describe("ContributionVote markup", () => {
  it("keeps current state synced to the card prop via an effect", () => {
    const src = readFileSync(new URL("./ContributionVote.tsx", import.meta.url), "utf8");
    expect(src).toContain("useState(card)");
    expect(src).toMatch(/useEffect\(\(\) => \{\s*setCurrent\(card\);\s*\}, \[card\]\);/);
  });
});
