<script lang="ts">
  // PlayerAvatar：房间参战席、个人设置等处展示的圆形头像；未设置头像时退回姓名首字。
  // 源：ui/AppViews.tsx:266-285
  import type { PublicPlayer } from "../../shared/types";
  import { mentionLabel } from "../../lib/playerDisplay";
  import { styleString } from "../../lib/style";

  let { player, size = 28, className = "" }: { player?: PublicPlayer; size?: number; className?: string } = $props();

  const label = $derived(mentionLabel(player) || player?.name || "");
  const ch = $derived(label.slice(0, 1) || "?");
  const style = $derived(styleString({
    width: size,
    height: size,
    fontSize: Math.max(10, Math.round(size * 0.42)),
    lineHeight: 1
  }));
</script>

{#if player?.avatarUrl}
  <!-- draggable=false：避免用作力导向图节点等可拖拽场景时，浏览器把它当成图片触发原生的
       "拖拽另存为/打开"效果而不是我们自己的拖拽逻辑。 -->
  <img class={`player-avatar ${className}`} {style} src={player.avatarUrl} alt="" draggable="false" ondragstart={(event) => event.preventDefault()} />
{:else}
  <span class={`player-avatar player-avatar-fallback ${className}`} {style} aria-hidden="true">
    <span class="player-avatar-char">{ch}</span>
  </span>
{/if}
