<script lang="ts">
  // 抢名大战改名面板。源：ui/AppViews.tsx:4066-4149。
  // config/me 原为 props，现直接读 sessionStore；onError 直接调 uiStore.notify。
  import type { PublicPlayer } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import { DEFAULT_NAME_WAR_RENAME_MIN_POINTS } from "../../lib/normalize";
  import { isExtremeRenameTarget, isNameWarLoser, nameWarRenameQuotaLeft } from "../../lib/playerDisplay";
  import { useMobileCollapse } from "../../lib/useMobileCollapse.svelte";
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import CollapseToggle from "../shell/CollapseToggle.svelte";

  let { targets }: { targets: PublicPlayer[] } = $props();

  const config = $derived(sessionStore.config!);
  const me = $derived(sessionStore.me!.player);

  let inputs = $state<Record<string, string>>({});
  let now = $state(Date.now());
  const nameWarQuota = $derived(nameWarRenameQuotaLeft(me, now));
  const nameWarMinPoints = $derived(Math.max(1, Math.round(config.nameWar?.renameMinPoints || DEFAULT_NAME_WAR_RENAME_MIN_POINTS)));
  const canNameWarRename = $derived(me.stats.rankedPoints >= nameWarMinPoints && nameWarQuota > 0);
  const extremeMinPoints = $derived(Math.max(1, Math.round(config.extremeMode.forceRenameMinPoints || 1)));
  const canExtremeRename = $derived(Boolean(me.extremeModeEnabled && me.stats.rankedPoints >= extremeMinPoints));
  const collapse = useMobileCollapse("nameWarLoser");

  $effect(() => {
    if (!targets.some((player) => (player.nameWarRenameProtectedUntil && player.nameWarRenameProtectedUntil > now) || (player.extremeRenameProtectedUntil && player.extremeRenameProtectedUntil > now))) return;
    const timer = window.setInterval(() => { now = Date.now(); }, 60_000);
    return () => window.clearInterval(timer);
  });

  async function renameTarget(targetId: string, kind: "nameWar" | "extreme") {
    const name = (inputs[targetId] || "").trim();
    if (!name) return;
    try {
      await ask("nameWar:renameTarget", { targetId, name, kind });
      inputs = { ...inputs, [targetId]: "" };
      uiStore.notify("名字修改成功");
    } catch (error) {
      uiStore.notify(error instanceof Error ? error.message : "修改失败");
    }
  }

  const panelTitle = $derived(config.nameWar.renamePanelTitle || config.nameWar.loserPanelTitle || "通用改名处");
</script>

<div class={`panel name-war-loser-panel ${collapse.collapsed ? "collapsed" : ""}`}>
  <div class="panel-header-row">
    <h2>🏷️ {panelTitle}</h2>
    <CollapseToggle collapsed={collapse.collapsed} onToggle={collapse.toggle} label={panelTitle} />
  </div>
  <div class={`mobile-collapsible-body ${collapse.collapsed ? "collapsed" : ""}`}>
    <p class="hint">名争改名需要 {nameWarMinPoints} 分以上，你剩余 {nameWarQuota} / 3 次；极限强关改名需要开启极限模式且至少 {extremeMinPoints} 分。</p>
    <div class="name-war-loser-list">
      {#each targets as player (player.id)}
        {@const nameWarTarget = isNameWarLoser(player)}
        {@const extremeTarget = isExtremeRenameTarget(player)}
        {@const nameWarProtectedMs = player.nameWarRenameProtectedUntil ? Math.max(0, player.nameWarRenameProtectedUntil - now) : 0}
        {@const extremeProtectedMs = player.extremeRenameProtectedUntil ? Math.max(0, player.extremeRenameProtectedUntil - now) : 0}
        {@const offlineKeepMs = !player.connected && player.disconnectedAt ? Math.max(0, 1_800_000 - (now - player.disconnectedAt)) : 0}
        {@const nameWarProtectedText = nameWarProtectedMs > 0 ? `名争保护 ${Math.ceil(nameWarProtectedMs / 3_600_000)} 小时` : "名争可改"}
        {@const extremeProtectedText = extremeProtectedMs > 0 ? `极限保护 ${Math.ceil(extremeProtectedMs / 3_600_000)} 小时` : "极限可改"}
        {@const inputValue = inputs[player.id] || ""}
        {@const selfTarget = player.id === me.id}
        {@const nameWarDisabled = !nameWarTarget || !canNameWarRename || selfTarget || nameWarProtectedMs > 0}
        {@const extremeDisabled = !extremeTarget || !canExtremeRename || selfTarget || extremeProtectedMs > 0}
        <div class="name-war-loser-card">
          <div class="admin-card-title">
            <strong>{player.nameWarPenaltyName || player.name}</strong>
            <small>
              <span class={`online-dot ${player.connected ? "online" : "offline"}`}>{player.connected ? "在线" : "离线"}</span>
              {player.stats.rankedPoints} 分
            </small>
          </div>
          <div class="room-info-tags">
            {#if nameWarTarget}<span class="room-info-tag">⚔️ {config.nameWar.nameWarLoserLabel || "名争失格"}</span>{/if}
            {#if extremeTarget}<span class="room-info-tag">⚡ {config.nameWar.extremeForceClosedLabel || "极限强关"}</span>{/if}
          </div>
          <p class="hint">
            {nameWarTarget ? nameWarProtectedText : ""}
            {nameWarTarget && extremeTarget ? " · " : ""}
            {extremeTarget ? extremeProtectedText : ""}
          </p>
          {#if offlineKeepMs > 0}<p class="hint">离线保留：约 {Math.ceil(offlineKeepMs / 60_000)} 分钟后从名单隐藏。</p>{/if}
          {#if player.nameWarRenamedByName}<p class="hint">名争最后改名者：{player.nameWarRenamedByName}</p>{/if}
          {#if player.extremeRenamedByName}<p class="hint">极限最后改名者：{player.extremeRenamedByName}</p>{/if}
          <div class="send-row">
            <input value={inputValue} maxlength="12" disabled={selfTarget || (!nameWarTarget && !extremeTarget)} oninput={(event) => (inputs = { ...inputs, [player.id]: event.currentTarget.value })} placeholder={selfTarget ? "不能改自己的名字" : "输入新名字"} />
            {#if nameWarTarget}<button disabled={nameWarDisabled || !inputValue.trim()} onclick={() => renameTarget(player.id, "nameWar")}>名争改名</button>{/if}
            {#if extremeTarget}<button disabled={extremeDisabled || !inputValue.trim()} onclick={() => renameTarget(player.id, "extreme")}>极限改名</button>{/if}
          </div>
        </div>
      {/each}
      {#if targets.length === 0}<p class="empty">暂无可改名目标</p>{/if}
    </div>
  </div>
</div>
