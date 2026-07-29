import { useEffect, useState } from "react";
import { Dices } from "lucide-react";
import type { PublicPlayer, RoomSnapshot } from "../shared/types";
import { ask } from "../lib/rpc";
import { socket } from "../ws";
import { displayPlayerName, PlayerAvatar, PlayerBadge, roomPlayerById } from "./AppViews";

const FACE_LABELS = ["", "⚀", "⚁", "⚂", "⚃", "⚄", "⚅"];

function diceFace(value: number) {
  return FACE_LABELS[value] || String(value);
}

function playerName(room: RoomSnapshot, id: string, fallbackNames?: Record<string, string>) {
  const player = roomPlayerById(room, id);
  if (player) return displayPlayerName(player);
  return fallbackNames?.[id] || id;
}

export function LiarsDicePanel({ room, me, onError }: { room: RoomSnapshot; me: PublicPlayer; onError: (message: string) => void }) {
  const ld = room.liarsDice;
  const [myDice, setMyDice] = useState<number[]>([]);
  const [bidCount, setBidCount] = useState(1);
  const [bidFace, setBidFace] = useState(1);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    function onHand(data: { roomId?: string; dice?: number[] }) {
      if (data?.roomId === room.id && Array.isArray(data.dice)) setMyDice(data.dice);
    }
    socket.on("liarsdice:hand", onHand);
    return () => socket.off("liarsdice:hand", onHand);
  }, [room.id]);

  useEffect(() => {
    // 换局/我不在参战席时清掉手上骰子；PhaseResult 揭晓阶段改用 revealedHands 展示全员骰子。
    if (!ld || !ld.participantIds.includes(me.id) || room.phase === "result") return;
    if (ld.roundNumber === 0) setMyDice([]);
  }, [ld?.roundNumber, ld?.participantIds, me.id, room.phase]);

  // 每次叫点变化（含轮到自己）时，默认选中"最小的可叫点数"：
  // 上家点数不为 6 则颗数不变、点数+1；点数为 6 则颗数+1、点数回到 1。
  useEffect(() => {
    if (!ld) return;
    if (ld.currentBid) {
      if (ld.currentBid.face < 6) {
        setBidCount(ld.currentBid.count);
        setBidFace(ld.currentBid.face + 1);
      } else {
        setBidCount(ld.currentBid.count + 1);
        setBidFace(1);
      }
    } else {
      setBidCount(ld.participantIds.length + 1);
      setBidFace(1);
    }
  }, [ld?.currentBid?.count, ld?.currentBid?.face, ld?.participantIds.length]);

  if (!ld) return null;

  const isParticipant = ld.participantIds.includes(me.id);
  const isReady = ld.readyPlayerIds.includes(me.id);
  const isMyTurn = room.phase === "choosing" && ld.currentTurn === me.id;
  // 上家点数已是 6 时，同颗数已无有效点数可叫，颗数下限直接顶到 +1。
  const minCount = ld.currentBid
    ? ld.currentBid.face >= 6 ? ld.currentBid.count + 1 : ld.currentBid.count
    : ld.participantIds.length + 1;
  const maxCount = ld.participantIds.length * 5;
  const minFace = ld.currentBid && ld.currentBid.count === bidCount ? ld.currentBid.face + 1 : 1;

  async function act(event: string, payload: unknown = {}) {
    if (busy) return;
    setBusy(true);
    try {
      await ask(event, payload);
    } catch (error) {
      onError(error instanceof Error ? error.message : "操作失败");
    } finally {
      setBusy(false);
    }
  }

  function onBidCountChange(nextCount: number) {
    setBidCount(nextCount);
    const nextMinFace = ld?.currentBid && ld.currentBid.count === nextCount ? ld.currentBid.face + 1 : 1;
    setBidFace(nextMinFace);
  }

  function submitBid() {
    let count = bidCount;
    let face = bidFace;
    if (ld?.currentBid) {
      const valid = count > ld.currentBid.count || (count === ld.currentBid.count && face > ld.currentBid.face);
      if (!valid) {
        onError("叫点无效，必须比上家颗数更多，或颗数不变但点数更大");
        return;
      }
    } else if (count < ld!.participantIds.length + 1) {
      count = ld!.participantIds.length + 1;
    }
    act("liarsdice:bid", { count, face });
  }

  return (
    <div className="liarsdice-panel">
      <div className="liarsdice-roster panel">
        <h3><Dices size={18} /> 参战席（{ld.participantIds.length}{room.settings.liarsDiceMaxPlayers ? `/${room.settings.liarsDiceMaxPlayers}` : ""}）</h3>
        <ul className="liarsdice-roster-list">
          {ld.participantIds.map((id) => {
            const player = roomPlayerById(room, id);
            const isTurn = room.phase === "choosing" && ld.currentTurn === id;
            const ready = ld.readyPlayerIds.includes(id);
            return (
              <li key={id} className={isTurn ? "current-turn" : ""}>
                {player ? (
                  <span className="seat-occupant-row">
                    <PlayerAvatar player={player} size={24} />
                    <PlayerBadge player={player} compact />
                  </span>
                ) : <span>{id}</span>}
                {room.phase === "ready" && <em>{ready ? "已准备" : "未准备"}</em>}
                {isTurn && <em className="liarsdice-turn-flag">叫点中</em>}
                {typeof ld.diceCounts?.[id] === "number" && <small>🎲 {ld.diceCounts[id]}</small>}
              </li>
            );
          })}
        </ul>
        {(room.phase === "ready" || room.phase === "waiting") && (
          <div className="liarsdice-roster-actions">
            {!isParticipant && (
              <button className="primary" disabled={busy || ld.participantIds.length >= (room.settings.liarsDiceMaxPlayers || 8)} onClick={() => act("liarsdice:joinRoster")}>
                加入参战席
              </button>
            )}
            {isParticipant && (
              <>
                <button disabled={busy} onClick={() => act("liarsdice:leaveRoster")}>离开参战席</button>
                {!isReady && <button className="primary" disabled={busy} onClick={() => act("liarsdice:ready")}>已准备</button>}
                {isReady && <span className="hint">已准备，等待其他人（{room.settings.liarsDiceMinPlayers || 3} 人以上且 5 秒无变动自动开局）</span>}
              </>
            )}
          </div>
        )}
      </div>

      {isParticipant && myDice.length > 0 && room.phase !== "result" && room.phase !== "punishment" && (
        <div className="liarsdice-my-hand panel">
          <h4>我的骰子</h4>
          <div className="liarsdice-dice-row">
            {myDice.map((d, i) => <span key={i} className="liarsdice-die">{diceFace(d)}</span>)}
          </div>
        </div>
      )}

      {room.phase === "choosing" && (
        <div className="liarsdice-bid-panel panel">
          <h4>{ld.currentBid ? `当前叫点：${ld.currentBid.count} 个 ${ld.currentBid.face}（${playerName(room, ld.currentBid.playerId)}）` : "还没有人叫点"}</h4>
          {ld.onesWildDisabled && <p className="hint">本局已有人叫过 1，此后 1 不再视为万能点。</p>}
          {isMyTurn ? (
            <div className="liarsdice-bid-controls">
              <label>
                颗数
                <select value={bidCount} onChange={(event) => onBidCountChange(Number(event.target.value))}>
                  {Array.from({ length: Math.max(0, maxCount - minCount + 1) }, (_, i) => minCount + i).map((count) => (
                    <option key={count} value={count}>{count}</option>
                  ))}
                </select>
              </label>
              <label>
                点数
                <select value={bidFace} onChange={(event) => setBidFace(Number(event.target.value))}>
                  {[1, 2, 3, 4, 5, 6].map((face) => (
                    <option key={face} value={face} disabled={bidCount === (ld.currentBid?.count ?? 0) && face < minFace}>
                      {face}
                    </option>
                  ))}
                </select>
              </label>
              <button className="primary" disabled={busy} onClick={submitBid}>叫点</button>
              {isParticipant && ld.currentBid && (
                <button className="liarsdice-challenge-btn" disabled={busy} onClick={() => act("liarsdice:challenge")}>
                  开牌
                </button>
              )}
            </div>
          ) : (
            <p className="hint">等待 {playerName(room, ld.currentTurn || "")} 叫点...</p>
          )}
          {!isMyTurn && isParticipant && ld.currentBid && (
            <button className="liarsdice-challenge-btn" disabled={busy} onClick={() => act("liarsdice:challenge")}>
              开牌
            </button>
          )}
        </div>
      )}

      {(room.phase === "result" || room.phase === "punishment") && ld.ended && ld.revealedHands && (
        <div className="liarsdice-reveal panel">
          <h4>开牌结果</h4>
          <p>
            叫点 {ld.currentBid?.count} 个 {ld.currentBid?.face || 0}，实际 {ld.actualCount} 个
            {ld.onesWildDisabled ? "（1 不算万能点）" : "（1 可算万能点）"}
          </p>
          <ul className="liarsdice-reveal-list">
            {ld.participantIds.map((id) => (
              <li key={id} className={id === ld.winnerId ? "winner" : id === ld.loserId ? "loser" : ""}>
                <span>{playerName(room, id)}</span>
                <span className="liarsdice-dice-row">
                  {(ld.revealedHands?.[id] || []).map((d, i) => <span key={i} className="liarsdice-die small">{diceFace(d)}</span>)}
                </span>
                {id === ld.winnerId && <em>胜</em>}
                {id === ld.loserId && <em>负</em>}
              </li>
            ))}
          </ul>
          {/* 惩罚未完成前后端拒绝 liarsdice:nextRound（要求 phase===result），惩罚阶段先隐藏按钮 */}
          {isParticipant && room.phase === "result" && (
            <button className="primary liarsdice-next-round-btn" disabled={busy} onClick={() => act("liarsdice:nextRound")}>再来一局</button>
          )}
        </div>
      )}
    </div>
  );
}
