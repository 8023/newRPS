<script module lang="ts">
  import type { ContributionStatus } from "../../shared/types";

  export const overviewStatusOrder: Array<{ status: ContributionStatus; label: string }> = [
    { status: "pending", label: "待审" },
    { status: "approved", label: "通过" },
    { status: "rejected", label: "驳回" },
    { status: "withdrawn", label: "撤回" }
  ];
</script>

<script lang="ts">
  // 源：ui/AdminContributionReview.tsx:180-208
  let { kind, label, counts, onJump }: {
    kind: "task" | "series";
    label: string;
    counts?: Record<string, number>;
    onJump: (kind: "task" | "series", status: ContributionStatus) => void;
  } = $props();
</script>

<div class="admin-overview-kind">
  <strong>{label}</strong>
  <div class="admin-overview-grid">
    {#each overviewStatusOrder as { status, label: statLabel } (status)}
      {@const n = counts?.[status] || 0}
      <button type="button" class="contribute-item admin-overview-stat" disabled={n === 0} onclick={() => onJump(kind, status)}>
        <strong>{n}</strong>
        <small>{statLabel}</small>
      </button>
    {/each}
  </div>
</div>
