<script lang="ts">
  // 源：ui/AppViews.tsx:3134-3153
  import type { ChatMessage, PublicPlayer } from "../../shared/types";
  import ChatAvatar from "./ChatAvatar.svelte";
  import ChatName from "./ChatName.svelte";

  let { message, me, author, onMention }: {
    message: ChatMessage;
    me: PublicPlayer;
    author?: PublicPlayer;
    onMention?: (player?: PublicPlayer) => void;
  } = $props();

  const mine = $derived(message.playerId === me.id);
  // 精确匹配：消息 @ 的 playerId 列表命中我时高亮气泡。
  const mentionsMe = $derived(Array.isArray(message.mentions) && message.mentions.includes(me.id));
  // 只能 @ 别人：点头像插入 @昵称（自己的气泡不可点）。
  const canMention = $derived(Boolean(onMention) && !mine);
</script>

{#if message.system}
  <p class="chat-system">{message.text}</p>
{:else}
  <div class={`chat-bubble-row ${mine ? "mine" : ""} ${mentionsMe ? "mentioned" : ""}`}>
    {#if !mine}<ChatAvatar player={author} onMention={canMention && onMention ? () => onMention(author) : undefined} />{/if}
    <div class="chat-bubble">
      <div class="chat-meta">
        {#if author}<ChatName player={author} />{:else}<b>{message.author}</b>{/if}
        {#if message.authorRole}<em>{message.authorRole}</em>{/if}
      </div>
      <p>{message.text}</p>
    </div>
    {#if mine}<ChatAvatar player={author} />{/if}
  </div>
{/if}
