<script lang="ts">
  // 投稿状态药丸徽标：待审批/已通过/已驳回等，颜色走 styles.css 里既有的 .status-chip +
  // .status-xxx 约定色（DESIGN.md「状态徽标」一节）。玩家端「参与共建」与后台共建审核
  // 现在统一复用它，放在投稿列表/详情第二行的开头（如「待审批 · 26/08/20」），不再各写一份。
  // 源：ui/AppViews.tsx:312-314
  import { contributionStatusLabel } from "../contributeSeries";

  let { status }: { status: string } = $props();
</script>

<span class={`status-chip status-${status}`}>{contributionStatusLabel[status] || status}</span>
