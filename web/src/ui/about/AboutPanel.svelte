<script lang="ts">
  // 源：ui/AppViews.tsx:3581-3661
  import Info from "@lucide/svelte/icons/info";
  import Coffee from "@lucide/svelte/icons/coffee";
  import Send from "@lucide/svelte/icons/send";
  import ExternalLink from "@lucide/svelte/icons/external-link";
  import type { AppConfig } from "../../shared/types";
  import { doumiaoLinks, luv4uLinks } from "../../lib/constants";
  import { styleString } from "../../lib/style";

  let { config, onClose, onOpenHelp }: { config: AppConfig; onClose: () => void; onOpenHelp?: () => void } = $props();

  const board = $derived(config.announcementBoard);
  const showBoard = $derived(board.enabled && (board.title.trim() || board.content.trim()));
</script>

<div class="modal-backdrop sponsor-backdrop" onclick={(event) => { if (event.target === event.currentTarget) onClose(); }}>
  <section class="sponsor-modal" onclick={(event) => event.stopPropagation()}>
    <div class="modal-title sponsor-title">
      <div>
        <h2><Info size={20} /> 关于</h2>
        <p class="hint">喜欢这个小站的话，可以在这里关注、进群或请作者喝杯咖啡。</p>
        {#if onOpenHelp}
          <p class="hint" style="margin-top: 0.4em">
            第一次来？<a href="#" onclick={(e) => { e.preventDefault(); onOpenHelp?.(); }}>看看怎么玩</a>。
          </p>
        {/if}
      </div>
      <button type="button" class="icon-button" onclick={onClose}>×</button>
    </div>
    {#if showBoard}
      <div class="announcement-board">
        <span class="announcement-board-kicker">📢 {board.title}</span>
        <p class="announcement-board-content">{board.content}</p>
      </div>
    {/if}
    <div class="sponsor-hero">
      <div class="sponsor-hero-icon"><Coffee size={30} /></div>
      <div>
        <strong>谢谢你愿意支持抖喵酱</strong>
        <p>使用以下链接来联系、赞助本游戏屋原作作者。</p>
      </div>
    </div>
    <div class="sponsor-grid">
      <!-- key 用 href 而不是 id：luv4uLinks 里有两条 id 都是 "x" 的历史数据，href 才是真正唯一的。 -->
      {#each doumiaoLinks as item (item.href)}
        <a class="sponsor-card" href={item.href} target="_blank" rel="noreferrer" style={styleString({ "--sponsor-tone": item.tone })}>
          <span class="sponsor-icon" aria-hidden="true">
            {#if item.id === "telegram"}<Send size={22} />{:else}{item.icon}{/if}
          </span>
          <span class="sponsor-copy">
            <strong>{item.title}</strong>
            <small>{item.description}</small>
          </span>
          <ExternalLink size={16} />
        </a>
      {/each}
    </div>
    <div class="sponsor-hero">
      <div class="sponsor-hero-icon"><Coffee size={30} /></div>
      <div>
        <strong>谢谢你愿意支持 8023</strong>
        <p>使用以下链接来联系、赞助本游戏屋现任开发者与运维者。</p>
      </div>
    </div>
    <div class="sponsor-grid">
      {#each luv4uLinks as item (item.href)}
        <a class="sponsor-card" href={item.href} target="_blank" rel="noreferrer" style={styleString({ "--sponsor-tone": item.tone })}>
          <span class="sponsor-icon" aria-hidden="true">
            {#if item.id === "telegram"}<Send size={22} />{:else}{item.icon}{/if}
          </span>
          <span class="sponsor-copy">
            <strong>{item.title}</strong>
            <small>{item.description}</small>
          </span>
          <ExternalLink size={16} />
        </a>
      {/each}
    </div>
  </section>
</div>
