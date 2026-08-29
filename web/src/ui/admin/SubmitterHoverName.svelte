<script lang="ts">
  /** 投稿者昵称：悬停/聚焦时懒加载并展示玩家详情浮窗（头像+称号等），复用主宠关系力导向图
      悬停节点同一套 PlayerAvatar/PlayerBadge + 药丸浮窗语言（petbond-graph-popover）。
      即便投稿是匿名的，管理员审核时仍看得到真实身份——匿名只对其他玩家生效。
      源：ui/AdminContributionReview.tsx:539-573。 */
  import type { PublicPlayer } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import PlayerAvatar from "../shell/PlayerAvatar.svelte";
  import PlayerBadge from "../shell/PlayerBadge.svelte";

  let { playerId, label }: { playerId: string; label: string } = $props();

  let player = $state<PublicPlayer | null>(null);
  let open = $state(false);
  let loaded = false;

  async function ensureLoaded() {
    if (loaded || !playerId) return;
    loaded = true;
    try {
      const res = await ask<{ player: PublicPlayer }>("player:get", { playerId });
      player = res.player || null;
    } catch {
      player = null;
    }
  }
</script>

<span
  class="contribute-submitter"
  tabindex={playerId ? 0 : undefined}
  onmouseenter={() => { open = true; void ensureLoaded(); }}
  onmouseleave={() => (open = false)}
  onfocus={() => { open = true; void ensureLoaded(); }}
  onblur={() => (open = false)}
>
  {label}
  {#if open && player}
    <span class="contribute-submitter-popover">
      <PlayerAvatar {player} size={36} />
      <PlayerBadge {player} compact />
    </span>
  {/if}
</span>
