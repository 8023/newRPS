<script lang="ts">
  // 源：ui/AppViews.tsx:3662-3699
  import Shield from "@lucide/svelte/icons/shield";

  let { onConfirm }: { onConfirm: () => void } = $props();

  const MIN_WAIT_MS = 3000;

  let understand = $state<"yes" | "no" | null>(null);
  let canComply = $state<"yes" | "no" | null>(null);
  let waited = $state(false);

  $effect(() => {
    const timer = window.setTimeout(() => { waited = true; }, MIN_WAIT_MS);
    return () => window.clearTimeout(timer);
  });

  const canConfirm = $derived(waited && understand === "yes" && canComply === "yes");
</script>

<div class="security-disclaimer-backdrop" role="dialog" aria-modal="true" aria-labelledby="security-disclaimer-title">
  <section class="security-disclaimer-card">
    <h2 id="security-disclaimer-title"><Shield size={20} /> 平台用户安全与免责声明</h2>
    <p>
      欢迎来到抖喵游戏屋。本站仅供年满 18 岁的成年人休闲娱乐与文化交流，严禁任何形式的赌博、金钱交易、诈骗及泄露个人隐私的行为。游玩过程中请规范自身言行、尊重其他玩家，确保互动基于双方自愿，禁止利用游戏机制强迫或诱导他人。
    </p>
    <p>
      作为技术提供方，本站不对惩罚任务、线下接触等游戏玩法之外的任何个人行为或约定承担任何责任。因用户个人行为导致的财产损失、隐私泄露或人身安全问题，均由用户自行承担，本站概不负责。
    </p>
    <div class="security-disclaimer-question">
      <span>你是否清楚上述风险与规定？</span>
      <label><input type="radio" name="sd-understand" checked={understand === "yes"} onchange={() => (understand = "yes")} /> 清楚</label>
      <label><input type="radio" name="sd-understand" checked={understand === "no"} onchange={() => (understand = "no")} /> 不清楚</label>
    </div>
    <div class="security-disclaimer-question">
      <span>你能否做到遵守上述平台规则？</span>
      <label><input type="radio" name="sd-comply" checked={canComply === "yes"} onchange={() => (canComply = "yes")} /> 能</label>
      <label><input type="radio" name="sd-comply" checked={canComply === "no"} onchange={() => (canComply = "no")} /> 不能</label>
    </div>
    <div class="security-disclaimer-question"></div>
    <button class="primary" type="button" disabled={!canConfirm} onclick={onConfirm}>确定</button>
  </section>
</div>
