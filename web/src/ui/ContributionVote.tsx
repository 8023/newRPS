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
  // useState(card) 只在首次挂载时生效；父组件若因任务切换或重新取数传入新 card，仍需把
  // prop 同步进本地状态。vote() 自己的乐观更新不受影响，因为父组件不会同时回传旧 card。
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
