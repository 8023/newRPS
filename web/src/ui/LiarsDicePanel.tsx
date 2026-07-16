import { useEffect, useState } from "react";
import { Dices } from "lucide-react";
import type { PublicPlayer, RoomSnapshot } from "../shared/types";
import { ask } from "../lib/rpc";
import { socket } from "../ws";
import { displayPlayerName, PlayerBadge, roomPlayerById } from "./AppViews";

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

  if (!ld) return null;

  const isParticipant = ld.participantIds.includes(me.id);
  const isReady = ld.readyPlayerIds.includes(me.id);
  const isMyTurn = room.phase === "choosing" && ld.currentTurn === me.id;
  const minCount = ld.currentBid ? ld.currentBid.count : ld.participantIds.length + 1;
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
                {player ? <PlayerBadge player={player} compact /> : <span>{id}</span>}
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

      {isParticipant && myDice.length > 0 && room.phase !== "result" && (
        <div className="liarsdice-my-hand panel">
          <h4>我的骰子</h4>
          <div className="liarsdice-dice-row">
            {myDice.map((d, i) => <span key={i} className="liarsdice-die">{diceFace(d)}</span>)}
          </div>
        </div>
      )}

      {room.phase === "choosing" && (
        <div className="liarsdice-bid-panel panel">
          <h4>{ld.currentBid ? `当前叫点：${ld.currentBid.count} 个 ${diceFace(ld.currentBid.face)}（${playerName(room, ld.currentBid.playerId)}）` : "还没有人叫点"}</h4>
          {ld.onesWildDisabled && <p className="hint">本局已有人叫过 1，此后 1 不再视为万能点。</p>}
          {isMyTurn ? (
            <div className="liarsdice-bid-controls">
              <label>
                颗数
                <input type="number" min={minCount} value={bidCount} onChange={(event) => setBidCount(Math.max(minCount, Number(event.target.value) || minCount))} />
              </label>
              <label>
                点数
                <select value={bidFace} onChange={(event) => setBidFace(Number(event.target.value))}>
                  {[1, 2, 3, 4, 5, 6].map((face) => (
                    <option key={face} value={face} disabled={bidCount === (ld.currentBid?.count ?? 0) && face < minFace}>
                      {diceFace(face)} {face}
                    </option>
                  ))}
                </select>
              </label>
              <button className="primary" disabled={busy} onClick={submitBid}>叫点</button>
              {ld.currentBid && <button disabled={busy} onClick={() => act("liarsdice:challenge")}>开牌（质疑上家）</button>}
            </div>
          ) : (
            <p className="hint">{room.phase === "choosing" ? `等待 ${playerName(room, ld.currentTurn || "")} 叫点或开牌...` : ""}</p>
          )}
        </div>
      )}

      {room.phase === "result" && ld.ended && ld.revealedHands && (
        <div className="liarsdice-reveal panel">
          <h4>开牌结果</h4>
          <p>
            叫点 {ld.currentBid?.count} 个 {diceFace(ld.currentBid?.face || 0)}，实际 {ld.actualCount} 个
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
          {isParticipant && (
            <button className="primary" disabled={busy} onClick={() => act("liarsdice:nextRound")}>下一局</button>
          )}
        </div>
      )}
    </div>
  );
}
