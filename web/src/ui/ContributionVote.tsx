import { useEffect, useState } from "react";
import type { VoteCard } from "../shared/types";
import { ask } from "../lib/rpc";

export function formatVoteStats(card: Pick<VoteCard, "hasVotes" | "displayRatio" | "voteCount">) {
  if (!card.hasVotes) return `暂无评价 · ${card.voteCount} 人评价`;
  return `${card.voteCount} 人评价 · 点赞 ${card.displayRatio ?? 0}%`;
}

export function ContributionVote({ card, onError, onVoted }: {
  card: VoteCard;
  onError: (message: string) => void;
  onVoted?: (next: VoteCard) => void;
}) {
  const [busy, setBusy] = useState(false);
  const [current, setCurrent] = useState(card);
  // useState(card) 只在组件首次挂载时生效——证明还没审核通过时任务卡片就已经挂载了这个
  // 组件（见 ContributionVoteLazy 的调用点），之后证明状态从 pending 变成 approved，父组件
  // 重新请求 votePreview 拿到新的 card（canVote 从 false 变 true）传下来，但 React 只是给
  // 同一个组件实例换 props，不会重新跑 useState 初始化，current 就永远停在挂载那一刻的旧值，
  // 评价按钮实际上永远不会出现。这里用 effect 把 prop 变化同步进本地 state，vote() 自己的
  // 乐观更新（setCurrent(next)）不受影响，因为投票后 eventId/proofStatus 不变、父组件不会
  // 再传入更旧的 card 把它覆盖回去。
  useEffect(() => {
    setCurrent(card);
  }, [card]);

  async function vote(value: 1 | -1) {
    setBusy(true);
    try {
      const next = await ask<VoteCard>("contribution:vote", { eventId: current.eventId, vote: value });
      setCurrent(next);
      onVoted?.(next);
    } catch (e) {
      onError(e instanceof Error ? e.message : "评价失败");
    } finally {
      setBusy(false);
    }
  }

  // 点赞率/贡献者在投票前是"剧透"：只有玩家自己点了赞或踩之后才展示，避免看到评价影响判断。
  const voted = current.myVote !== 0;

  return (
    <div className="contribution-vote">
      {voted ? (
        <p className="contribution-vote-result">
          <span className="vote-result-contributor">由 {current.contributorDisplayName} 贡献</span>
          <span className="vote-result-stats">{formatVoteStats(current)}</span>
        </p>
      ) : current.canVote ? (
        <div className="vote-actions">
          <span>评价任务：</span>
          <button type="button" disabled={busy} onClick={() => void vote(1)}>赞 👍</button>
          <button type="button" disabled={busy} onClick={() => void vote(-1)}>踩 👎</button>
        </div>
      ) : current.cannotVoteReason ? (
        <p className="hint">{current.cannotVoteReason}</p>
      ) : null}
    </div>
  );
}
