<script lang="ts">
  // 锤子剪刀布是"默认游戏"，历史上没有独立的面板组件，直接内联在 Room 里；这里把它拆成
  // 独立组件，与其它 7 款游戏的面板保持同样的组织方式。
  // 源：ui/AppViews.tsx:1969-2000（Room 组件内联的 move-panel 分支）。
  import type { Move, RoomSnapshot, PublicPlayer, SeatKey } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import { choiceText, moveButtonIcon, moveButtonLabel } from "../../lib/gameDisplay";

  let { room, me, mySeat, onError }: {
    room: RoomSnapshot;
    me: PublicPlayer;
    mySeat: SeatKey | null;
    onError: (message: string) => void;
  } = $props();

  const choices = $derived(room.choices || {});
  let localChoice = $state<Move | null>(null);

  $effect(() => {
    if (!mySeat || (room.phase === "choosing" && !choices[mySeat])) localChoice = null;
  });

  const myChoice = $derived(mySeat ? (room.phase === "result" ? undefined : localChoice || choices[mySeat]) : undefined);
  const resultChoice = $derived(mySeat ? room.revealedChoices?.[mySeat] : undefined);
  const canChoose = $derived(Boolean(mySeat && room.phase !== "punishment" && (room.phase === "choosing" || room.phase === "result") && room.seats.A && room.seats.B));
  const canShowGiveawayButton = $derived(Boolean(mySeat && me.giveawayEnabled && room.seats.A && room.seats.B));

  async function choose(move: Move) {
    localChoice = move;
    try {
      await ask("room:move", { move });
    } catch (error) {
      localChoice = null;
      onError(error instanceof Error ? error.message : "出拳失败");
    }
  }

  // 撤回自己已出的拳（仅锤子剪刀布，对手还没出拳时）：乐观地先清掉本地出拳，让按钮立刻
  // 变回可选状态；如果服务端拒绝（比如已经被主人强制白给锁死、或对手已经出拳导致这一局
  // 刚好在这一瞬间结算了），choices[mySeat] 仍是服务端权威状态，myChoice 会自动回退显示
  // 原来的选择，不需要额外的失败回滚逻辑。
  async function withdrawMove() {
    localChoice = null;
    try {
      await ask("room:move:withdraw", {});
    } catch (error) {
      onError(error instanceof Error ? error.message : "撤回失败");
    }
  }
</script>

{#if mySeat}
  <div class="move-panel">
    <div>
      <h3>请选择出拳</h3>
      <p class="hint">
        {#if room.phase === "punishment"}惩罚完成前不能出拳。
        {:else if myChoice}你已出拳：{choiceText(myChoice)}，对方还没出拳，可以点按钮撤回。
        {:else if resultChoice}上一局：{choiceText(resultChoice)}，可直接开始下一局。
        {:else if canChoose}坐下不算出拳，点一个 emoji 才会锁定。
        {:else}等待另一位玩家坐下。{/if}
      </p>
      {#if room.forgiveAdvantageTargetId === me.id}
        <p class="hint forgive-advantage-hint">上局对方放过了你，本局你将受到"命运的安排"</p>
      {/if}
      {#if room.forgiveAdvantageBeneficiaryId === me.id}
        <p class="hint forgive-advantage-hint">上局你放过了对方，本局你将受到"命运的安排"</p>
      {/if}
    </div>
    <div class="move-row emoji-row">
      {#if myChoice}
        <button class={`move-withdraw-button${myChoice === "giveaway" ? " giveaway-move-button" : ""}`} disabled={!canChoose} onclick={() => void withdrawMove()}>
          {moveButtonIcon(myChoice)}<span>撤回{moveButtonLabel(myChoice)}</span>
        </button>
      {:else}
        <button disabled={!canChoose} onclick={() => choose("rock")}>✊<span>锤子</span></button>
        <button disabled={!canChoose} onclick={() => choose("scissors")}>✌️<span>剪刀</span></button>
        <button disabled={!canChoose} onclick={() => choose("paper")}>🖐️<span>布</span></button>
        {#if canShowGiveawayButton}<button class="giveaway-move-button" disabled={!canChoose} onclick={() => choose("giveaway")}>🫴<span>白给</span></button>{/if}
      {/if}
    </div>
  </div>
{/if}
