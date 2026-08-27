import { useEffect, useRef, useState } from "react";
import type { CoinFace, PublicPlayer, RoomSnapshot } from "../shared/types";
import { ask } from "../lib/rpc";

// 落定后展示结果横幅、重新允许下一次抛掷的等待时长；必须和 .coinflip-coin-inner 的
// CSS transition 时长（见 styles.css）保持一致，否则要么横幅先于硬币停下出现，
// 要么硬币停了还要再等一拍才出结果。
const REVEAL_MS = 1000;

export function coinFaceLabel(face: CoinFace | undefined) {
  if (face === "char") return "字";
  if (face === "flower") return "花";
  return "？";
}

export function CoinFlipPanel({ room, me, onError }: { room: RoomSnapshot; me: PublicPlayer; onError: (message: string) => void }) {
  const mySeat = room.seats.A?.id === me.id ? "A" : null;
  const state = room.coinFlip;
  const [busy, setBusy] = useState(false);
  const [spinning, setSpinning] = useState(false);
  const [rotation, setRotation] = useState(0);
  const [revealed, setRevealed] = useState<{ result: CoinFace; correct: boolean } | null>(null);
  const rotationRef = useRef(0);
  const lastSettledAtRef = useRef(-1);

  useEffect(() => {
    const settledAt = state?.settledAt || 0;
    if (!settledAt || settledAt === lastSettledAtRef.current) return;
    lastSettledAtRef.current = settledAt;
    const faceOffset = state?.result === "flower" ? 180 : 0;
    const base = rotationRef.current;
    // 每次都朝同一方向再多转 5 圈才停在目标面上，续着上一次的角度往前转，不会往回跳。
    const target = base - (base % 360) + 1800 + faceOffset;
    rotationRef.current = target;
    setRotation(target);
    const elapsed = Date.now() - settledAt;
    if (elapsed >= REVEAL_MS) {
      // 页面是刷新/重新进入房间时才看到这次结果，硬币早就落定了，不重放动画。
      setSpinning(false);
      setRevealed(state ? { result: state.result, correct: state.correct } : null);
      return undefined;
    }
    setSpinning(true);
    setRevealed(null);
    const timer = window.setTimeout(() => {
      setSpinning(false);
      setRevealed(state ? { result: state.result, correct: state.correct } : null);
    }, REVEAL_MS - elapsed);
    return () => window.clearTimeout(timer);
  }, [state?.settledAt, state?.result, state?.correct]);

  async function guess(face: Exclude<CoinFace, "">) {
    if (busy || spinning) return;
    setBusy(true);
    try {
      await ask("coinflip:guess", { face });
    } catch (error) {
      onError(error instanceof Error ? error.message : "猜硬币失败");
    } finally {
      setBusy(false);
    }
  }

  const canGuess = Boolean(mySeat && room.phase === "choosing" && !spinning && !busy);

  let statusHint: string;
  if (!mySeat) {
    statusHint = room.seats.A ? "对方正在猜硬币，坐到参战席可以接棒继续玩。" : "坐到参战席即可开局，一个人就能玩。";
  } else if (spinning) {
    statusHint = "硬币抛起来了……";
  } else if (room.phase === "punishment") {
    statusHint = "猜错了，完成下面的惩罚任务后才能继续抛。";
  } else {
    statusHint = "选「字」还是「花」，服务器立即抛硬币公开结算。";
  }

  return (
    <div className="coinflip-panel">
      <div className="coinflip-head">
        <div>
          <h3>🪙 猜硬币</h3>
          <p className="hint">{statusHint}</p>
        </div>
      </div>
      <div className="coinflip-stage">
        <div className="coinflip-coin">
          <div className="coinflip-coin-inner" style={{ transform: `rotateY(${rotation}deg)` }}>
            <div className="coinflip-face coinflip-face-char">
              <span className="coinflip-face-digit">1</span>
            </div>
            <div className="coinflip-face coinflip-face-flower">
              <ChrysanthemumMark />
            </div>
          </div>
        </div>
        {revealed && !spinning && (
          <p className={`coinflip-result-banner ${revealed.correct ? "correct" : "wrong"}`}>
            开出「{coinFaceLabel(revealed.result)}」，{revealed.correct ? "猜中了！" : "猜错了。"}
          </p>
        )}
      </div>
      {mySeat ? (
        <div className="coinflip-choice-row">
          <button type="button" className="coinflip-choice-button" disabled={!canGuess} onClick={() => void guess("char")}>
            <span className="coinflip-choice-icon">壹</span>
            <span>猜字</span>
          </button>
          <button type="button" className="coinflip-choice-button" disabled={!canGuess} onClick={() => void guess("flower")}>
            <span className="coinflip-choice-icon">✿</span>
            <span>猜花</span>
          </button>
        </div>
      ) : (
        <p className="hint">猜硬币是单人玩法，坐到参战席即可自己开局，猜错就会被系统派发惩罚任务。</p>
      )}
    </div>
  );
}

// petalPath：单瓣花瓣路径——从花心附近的 r0 到瓣尖 r1，中点用两条对称二次贝塞尔鼓起到
// 半宽 w，两端收尖成"眼形"，比正圆/椭圆更接近真实花瓣的轮廓。
function petalPath(r0: number, r1: number, w: number) {
  const mid = (r0 + r1) / 2;
  return `M0,${-r0} Q${w},${-mid} 0,${-r1} Q${-w},${-mid} 0,${-r0} Z`;
}

// ChrysanthemumMark：菊花面——外层大瓣 + 内层小瓣错位叠放出层次，花心压在最上层，
// 瓣形用尖角"眼形"而不是正椭圆，配色沿用花面主题的粉色系深浅两档，
// 与站点整体的扁平图形风格保持一致，不追求写实。
function ChrysanthemumMark() {
  const outerCount = 16;
  const outerPath = petalPath(13, 44, 10);
  const outerAngles = Array.from({ length: outerCount }, (_, i) => (360 / outerCount) * i);

  const innerCount = 12;
  const innerPath = petalPath(7, 27, 6.5);
  const innerAngles = Array.from({ length: innerCount }, (_, i) => (360 / innerCount) * i + 15);

  return (
    <svg viewBox="0 0 100 100" className="coinflip-flower-svg" aria-hidden="true">
      <g transform="translate(50 50)">
        <g className="coinflip-flower-layer-outer">
          {outerAngles.map((deg) => (
            <path key={deg} d={outerPath} transform={`rotate(${deg})`} />
          ))}
        </g>
        <g className="coinflip-flower-layer-inner">
          {innerAngles.map((deg) => (
            <path key={deg} d={innerPath} transform={`rotate(${deg})`} />
          ))}
        </g>
        <circle className="coinflip-flower-core" cx="0" cy="0" r="9" />
      </g>
    </svg>
  );
}
