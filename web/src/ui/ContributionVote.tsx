import { useState } from "react";
import type { VoteCard } from "../shared/types";
import { ask } from "../lib/rpc";

export function formatVoteStats(card: Pick<VoteCard, "hasVotes" | "displayRatio" | "voteCount">) {
  if (!card.hasVotes) return `暂无评价 · ${card.voteCount} 人评价`;
  return `点赞率 ${card.displayRatio ?? 0}% · ${card.voteCount} 人评价`;
}

export function ContributionVote({ card, onError, onVoted }: {
  card: VoteCard;
  onError: (message: string) => void;
  onVoted?: (next: VoteCard) => void;
}) {
  const [busy, setBusy] = useState(false);
  const [current, setCurrent] = useState(card);

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
        <p className="contribution-vote-result">由 {current.contributorDisplayName} 贡献 · {formatVoteStats(current)}</p>
      ) : current.canVote ? (
        <div className="vote-actions">
          <span>这个任务怎么样？</span>
          <button type="button" disabled={busy} onClick={() => void vote(1)}>赞 👍</button>
          <button type="button" disabled={busy} onClick={() => void vote(-1)}>踩 👎</button>
        </div>
      ) : current.cannotVoteReason ? (
        <p className="hint">{current.cannotVoteReason}</p>
      ) : null}
    </div>
  );
}
