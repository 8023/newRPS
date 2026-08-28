<script module lang="ts">
  import type { CoinFace } from "../../shared/types";

  export function coinFaceLabel(face: CoinFace | undefined) {
    if (face === "char") return "字";
    if (face === "flower") return "花";
    return "？";
  }
</script>

<script lang="ts">
  // 源：ui/CoinFlipPanel.tsx
  import type { RoomSnapshot, PublicPlayer } from "../../shared/types";
  import { ask } from "../../lib/rpc";

  // 落定后展示结果横幅、重新允许下一次抛掷的等待时长；必须和 .coinflip-coin-inner 的
  // CSS transition 时长（见 styles.css）保持一致，否则要么横幅先于硬币停下出现，
  // 要么硬币停了还要再等一拍才出结果。
  const REVEAL_MS = 1000;

  let { room, me, onError }: { room: RoomSnapshot; me: PublicPlayer; onError: (message: string) => void } = $props();

  const mySeat = $derived(room.seats.A?.id === me.id ? "A" : null);
  const gameState = $derived(room.coinFlip);
  let busy = $state(false);
  let spinning = $state(false);
  let rotation = $state(0);
  let revealed = $state<{ result: CoinFace; correct: boolean } | null>(null);
  // 纯可变盒子，不参与渲染（对应原 React 版的 useRef）。
  let rotationValue = 0;
  let lastSettledAt = -1;

  $effect(() => {
    const settledAt = gameState?.settledAt || 0;
    if (!settledAt || settledAt === lastSettledAt) return;
    lastSettledAt = settledAt;
    const faceOffset = gameState?.result === "flower" ? 180 : 0;
    const base = rotationValue;
    // 每次都朝同一方向再多转 5 圈才停在目标面上，续着上一次的角度往前转，不会往回跳。
    const target = base - (base % 360) + 1800 + faceOffset;
    rotationValue = target;
    rotation = target;
    const elapsed = Date.now() - settledAt;
    if (elapsed >= REVEAL_MS) {
      // 页面是刷新/重新进入房间时才看到这次结果，硬币早就落定了，不重放动画。
      spinning = false;
      revealed = gameState ? { result: gameState.result, correct: gameState.correct } : null;
      return undefined;
    }
    spinning = true;
    revealed = null;
    const timer = window.setTimeout(() => {
      spinning = false;
      revealed = gameState ? { result: gameState.result, correct: gameState.correct } : null;
    }, REVEAL_MS - elapsed);
    return () => window.clearTimeout(timer);
  });

  async function guess(face: Exclude<CoinFace, "">) {
    if (busy || spinning) return;
    busy = true;
    try {
      await ask("coinflip:guess", { face });
    } catch (error) {
      onError(error instanceof Error ? error.message : "猜硬币失败");
    } finally {
      busy = false;
    }
  }

  const canGuess = $derived(Boolean(mySeat && room.phase === "choosing" && !spinning && !busy));

  const statusHint = $derived.by(() => {
    if (!mySeat) return room.seats.A ? "对方正在猜硬币，坐到参战席可以接棒继续玩。" : "坐到参战席即可开局，一个人就能玩。";
    if (spinning) return "硬币抛起来了……";
    if (room.phase === "punishment") return "猜错了，完成下面的惩罚任务后才能继续抛。";
    return "选「字」还是「花」，服务器立即抛硬币公开结算。";
  });

  // petalPath：单瓣花瓣路径——从花心附近的 r0 到瓣尖 r1，中点用两条对称二次贝塞尔鼓起到
  // 半宽 w，两端收尖成"眼形"，比正圆/椭圆更接近真实花瓣的轮廓。
  function petalPath(r0: number, r1: number, w: number) {
    const mid = (r0 + r1) / 2;
    return `M0,${-r0} Q${w},${-mid} 0,${-r1} Q${-w},${-mid} 0,${-r0} Z`;
  }

  const outerPath = petalPath(13, 44, 10);
  const outerAngles = Array.from({ length: 16 }, (_, i) => (360 / 16) * i);
  const innerPath = petalPath(7, 27, 6.5);
  const innerAngles = Array.from({ length: 12 }, (_, i) => (360 / 12) * i + 15);
</script>

<!-- ChrysanthemumMark：菊花面——外层大瓣 + 内层小瓣错位叠放出层次，花心压在最上层，
     瓣形用尖角"眼形"而不是正椭圆，配色沿用花面主题的粉色系深浅两档，
     与站点整体的扁平图形风格保持一致，不追求写实。 -->
{#snippet chrysanthemumMark()}
  <svg viewBox="0 0 100 100" class="coinflip-flower-svg" aria-hidden="true">
    <g transform="translate(50 50)">
      <g class="coinflip-flower-layer-outer">
        {#each outerAngles as deg (deg)}<path d={outerPath} transform={`rotate(${deg})`} />{/each}
      </g>
      <g class="coinflip-flower-layer-inner">
        {#each innerAngles as deg (deg)}<path d={innerPath} transform={`rotate(${deg})`} />{/each}
      </g>
      <circle class="coinflip-flower-core" cx="0" cy="0" r="9" />
    </g>
  </svg>
{/snippet}

<div class="coinflip-panel">
  <div class="coinflip-head">
    <div>
      <h3>🪙 猜硬币</h3>
      <p class="hint">{statusHint}</p>
    </div>
  </div>
  <div class="coinflip-stage">
    <div class="coinflip-coin">
      <div class="coinflip-coin-inner" style={`transform: rotateY(${rotation}deg)`}>
        <div class="coinflip-face coinflip-face-char"><span class="coinflip-face-digit">1</span></div>
        <div class="coinflip-face coinflip-face-flower">{@render chrysanthemumMark()}</div>
      </div>
    </div>
    {#if revealed && !spinning}
      <p class={`coinflip-result-banner ${revealed.correct ? "correct" : "wrong"}`}>
        开出「{coinFaceLabel(revealed.result)}」，{revealed.correct ? "猜中了！" : "猜错了。"}
      </p>
    {/if}
  </div>
  {#if mySeat}
    <div class="coinflip-choice-row">
      <button type="button" class="coinflip-choice-button" disabled={!canGuess} onclick={() => void guess("char")}>
        <span class="coinflip-choice-icon">壹</span><span>猜字</span>
      </button>
      <button type="button" class="coinflip-choice-button" disabled={!canGuess} onclick={() => void guess("flower")}>
        <span class="coinflip-choice-icon">✿</span><span>猜花</span>
      </button>
    </div>
  {:else}
    <p class="hint">猜硬币是单人玩法，坐到参战席即可自己开局，猜错就会被系统派发惩罚任务。</p>
  {/if}
</div>
