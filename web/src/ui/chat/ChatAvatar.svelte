<script lang="ts">
  // 源：ui/AppViews.tsx:3156-3165
  import type { PublicPlayer } from "../../shared/types";
  import { mentionLabel } from "../../lib/playerDisplay";

  let { player, onMention }: { player?: PublicPlayer; onMention?: () => void } = $props();
  const label = $derived(mentionLabel(player) || player?.name);
  const ch = $derived(label?.slice(0, 1) || "?");
</script>

{#if onMention}
  <button type="button" class="chat-avatar chat-avatar-button" title={`@ ${label}`} onclick={onMention}>
    {#if player?.avatarUrl}<img class="chat-avatar-image" src={player.avatarUrl} alt="" />{:else}{ch}{/if}
  </button>
{:else}
  <span class="chat-avatar">
    {#if player?.avatarUrl}<img class="chat-avatar-image" src={player.avatarUrl} alt="" />{:else}{ch}{/if}
  </span>
{/if}
