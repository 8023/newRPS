<script lang="ts">
  // 大厅「宠物乐园」：关系展示 / 认主 / 认宠。源：ui/AppViews.tsx:4376-4693。
  // config/me/lobby 原为 props，现直接读 sessionStore；onError 直接调 uiStore.notify。
  import type { PetBondState } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import { socket } from "../../ws";
  import { withPetBondDefaults } from "../../lib/normalize";
  import { emptyPetBondState, resolveLobbyPlayer, petBondBadgeFallback } from "../../lib/petBond";
  import { useMobileCollapse } from "../../lib/useMobileCollapse.svelte";
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import CollapseToggle from "../shell/CollapseToggle.svelte";
  import PetBondPlayerInfo from "./PetBondPlayerInfo.svelte";

  const config = $derived(sessionStore.config!);
  const me = $derived(sessionStore.me!.player);
  const lobby = $derived(sessionStore.lobby!);

  const petBondCfg = $derived(withPetBondDefaults(config.petBond));
  let bondState = $state<PetBondState>(emptyPetBondState());
  let busy = $state(false);
  let titlePetId = $state<string | null>(null);
  let titleDraft = $state("");
  const collapse = useMobileCollapse("petBond");

  function onError(message: string) {
    uiStore.notify(message);
  }

  async function reload() {
    try {
      // 已订阅时 getState 仍可用（手动刷新）；首次挂载走 subscribe。
      const next = await ask<PetBondState>("petbond:getState", {});
      bondState = { ...emptyPetBondState(), ...next, config: withPetBondDefaults(next?.config || petBondCfg) };
    } catch (error) {
      onError(error instanceof Error ? error.message : "加载宠物乐园失败");
    }
  }

  $effect(() => {
    void me.id;
    let alive = true;
    const onUpdate = (payload: PetBondState) => {
      if (!alive || !payload || typeof payload !== "object") return;
      bondState = { ...emptyPetBondState(), ...payload, config: withPetBondDefaults(payload.config || petBondCfg) };
    };
    const subscribe = () => {
      ask<PetBondState>("petbond:subscribe", {})
        .then((next) => {
          if (alive && next && typeof next === "object") {
            bondState = { ...emptyPetBondState(), ...next, config: withPetBondDefaults(next.config || petBondCfg) };
          }
        })
        .catch((error) => {
          if (alive) onError(error instanceof Error ? error.message : "加载宠物乐园失败");
        });
    };
    subscribe();
    socket.on("petbond:update", onUpdate);
    // WS 断线重连会换一个全新连接，服务端的频道订阅关系不会跨连接保留，必须重新 subscribe，
    // 否则重连后收不到任何 petbond:update，关系展示会停在断线前那一刻，需要整页刷新才恢复。
    socket.on("connect", subscribe);
    return () => {
      alive = false;
      socket.off("petbond:update", onUpdate);
      socket.off("connect", subscribe);
      ask("petbond:unsubscribe", {}).catch(() => undefined);
    };
  });

  // 根据大厅玩家开关与公开关系边签名兜底刷新，确保他人开启功能后无需整页刷新。
  const lobbyBondSig = $derived(
    lobby.players
      .map((p) => `${p.id}:${p.connected ? 1 : 0}${p.bondMasterEnabled ? 1 : 0}${p.bondPetEnabled ? 1 : 0}${p.bondPublicDisplay ? 1 : 0}`)
      .sort()
      .join("|")
  );
  const bondsSig = $derived(
    (lobby.petBonds || [])
      .map((e) => `${e.masterId}>${e.petId}:${e.petTitle || ""}`)
      .sort()
      .join("|")
  );
  $effect(() => {
    void lobbyBondSig;
    void bondsSig;
    void me.bondMasterEnabled;
    void me.bondPetEnabled;
    void me.bondPublicDisplay;
    void reload();
  });

  async function run(event: string, payload: Record<string, unknown>, okMsg?: string) {
    busy = true;
    try {
      const next = await ask<PetBondState>(event, payload);
      bondState = { ...emptyPetBondState(), ...next, config: withPetBondDefaults(next?.config || petBondCfg) };
      if (okMsg) onError(okMsg);
    } catch (error) {
      onError(error instanceof Error ? error.message : "操作失败");
    } finally {
      busy = false;
    }
  }

  const masterSlotsLeft = $derived(Math.max(0, petBondCfg.maxMastersPerPet - (bondState.masters?.length || 0)));
  const petSlotsLeft = $derived(Math.max(0, petBondCfg.maxPetsPerMaster - (bondState.pets?.length || 0)));

  // 候选列表：过滤掉"已是关系"，incoming 申请用户置顶并高亮。
  const masterCandidates = $derived(
    (bondState.masterCandidates || [])
      .filter((c) => c.status !== "already")
      .slice()
      .sort((a, b) => Number(Boolean(b.incoming)) - Number(Boolean(a.incoming)) || a.name.localeCompare(b.name, "zh"))
  );
  const petCandidates = $derived(
    (bondState.petCandidates || [])
      .filter((c) => c.status !== "already")
      .slice()
      .sort((a, b) => Number(Boolean(b.incoming)) - Number(Boolean(a.incoming)) || a.name.localeCompare(b.name, "zh"))
  );
  const chains = $derived(bondState.chains || []);
</script>

<div class={`panel pet-bond-panel ${collapse.collapsed ? "collapsed" : ""}`}>
  <div class="panel-title compact-title">
    <h2>🐾 {petBondCfg.panelTitle}</h2>
    <span class="panel-title-actions">
      <CollapseToggle collapsed={collapse.collapsed} onToggle={collapse.toggle} label={petBondCfg.panelTitle} />
    </span>
  </div>
  <div class={`mobile-collapsible-body ${collapse.collapsed ? "collapsed" : ""}`}>
    <section class="pet-bond-section">
      <h3>关系展示</h3>
      <p class="hint">仅显示在线且开启「公开展示」的 2～3 级认主链；更长链会拆成多组。</p>
      <div class="pet-bond-chain-list">
        {#each chains as chain, idx (`${chain.playerIds.join(">")}-${idx}`)}
          <div class="pet-bond-chain">
            {#each chain.playerIds as id, i (`${id}-${i}`)}
              {@const player = resolveLobbyPlayer(lobby.players, id, { name: chain.playerNames[i], displayName: chain.playerNames[i] })}
              <span class="pet-bond-chain-node">
                {#if i > 0}<span class="pet-bond-chain-arrow">→</span>{/if}
                <PetBondPlayerInfo {player} size={26} />
              </span>
            {/each}
          </div>
        {/each}
        {#if chains.length === 0}<p class="empty">暂无公开关系链</p>{/if}
      </div>
    </section>

    <section class="pet-bond-section">
      <div class="pet-bond-section-header">
        <h3>认主</h3>
        <span class="pet-bond-quota">剩余 {masterSlotsLeft} 名</span>
      </div>
      <div class="pet-bond-member-list">
        {#each bondState.masters as m (m.playerId)}
          {@const player = resolveLobbyPlayer(lobby.players, m.playerId, petBondBadgeFallback(m))}
          <div class="pet-bond-member-row">
            <PetBondPlayerInfo {player} showStatus />
            <span class="pet-bond-row-actions">
              {#if m.releaseIncoming && m.releaseRequestId}
                <button type="button" class="small primary" disabled={busy} onclick={() => run("petbond:approve", { requestId: m.releaseRequestId }, "已解除关系")}>同意解除关系</button>
              {/if}
              {#if m.releasePending && m.releaseRequestId}
                <button type="button" class="small danger-soft" disabled={busy} onclick={() => run("petbond:cancel", { requestId: m.releaseRequestId }, "已撤销申请")}>撤销申请</button>
              {/if}
              {#if !m.releasePending && !m.releaseIncoming && m.connected}
                <button type="button" class="small soft-button" disabled={busy} onclick={() => run("petbond:requestRelease", { masterId: m.playerId, petId: me.id })}>申请解除关系</button>
              {/if}
            </span>
          </div>
        {/each}
        {#if bondState.masters.length === 0}<p class="hint">你还没有主人</p>{/if}
      </div>
      {#if me.bondMasterEnabled}
        <div class="pet-bond-candidate-list">
          <p class="hint">开启认宠的在线玩家：</p>
          {#each masterCandidates as c (c.playerId)}
            {@const player = resolveLobbyPlayer(lobby.players, c.playerId, petBondBadgeFallback(c))}
            <div class={`pet-bond-member-row ${c.incoming ? "pet-bond-pin" : ""}`}>
              <PetBondPlayerInfo {player} />
              <span class="pet-bond-row-actions">
                {#if c.incoming && c.incomingId}
                  <button type="button" class="small primary" disabled={busy} onclick={() => run("petbond:approve", { requestId: c.incomingId }, "已同意")}>{c.incomingLabel || "同意认宠请求"}</button>
                {:else if c.status === "pending" && c.requestId}
                  <button type="button" class="small danger-soft" disabled={busy} onclick={() => run("petbond:cancel", { requestId: c.requestId }, "已撤销申请")}>撤销申请</button>
                {:else}
                  <button type="button" class="small" disabled={busy || masterSlotsLeft <= 0} onclick={() => run("petbond:seekMaster", { targetId: c.playerId })}>申请认主</button>
                {/if}
              </span>
            </div>
          {/each}
          {#if masterCandidates.length === 0}<p class="empty">暂无可认主对象</p>{/if}
        </div>
      {:else}
        <p class="hint">在个人设置开启「开启认宠」后，可向在线玩家申请认主。</p>
      {/if}
    </section>

    <section class="pet-bond-section">
      <div class="pet-bond-section-header">
        <h3>认宠</h3>
        <span class="pet-bond-quota">剩余 {petSlotsLeft} 名</span>
      </div>
      <div class="pet-bond-member-list">
        {#each bondState.pets as p (p.playerId)}
          {@const player = resolveLobbyPlayer(lobby.players, p.playerId, petBondBadgeFallback(p))}
          <div class={`pet-bond-member-row ${p.newMasterPendingId || p.releaseIncoming ? "pet-bond-pin" : ""}`}>
            <PetBondPlayerInfo {player} showStatus />
            <span class="pet-bond-row-actions">
              {#if p.newMasterPendingId}
                <button type="button" class="small primary" disabled={busy} onclick={() => run("petbond:approve", { requestId: p.newMasterPendingId }, "已同意宠物认新主")}>
                  同意认新主{p.newMasterPendingName ? `（${p.newMasterPendingName}）` : ""}
                </button>
              {/if}
              <button type="button" class="small soft-button" disabled={busy} onclick={() => { titlePetId = p.playerId; titleDraft = p.petTitle || ""; }}>设置宠物称号</button>
              {#if p.releaseIncoming && p.releaseRequestId}
                <button type="button" class="small primary" disabled={busy} onclick={() => run("petbond:approve", { requestId: p.releaseRequestId }, "已解除关系")}>同意解除关系</button>
              {/if}
              {#if p.releasePending && p.releaseRequestId}
                <button type="button" class="small danger-soft" disabled={busy} onclick={() => run("petbond:cancel", { requestId: p.releaseRequestId }, "已撤销申请")}>撤销申请</button>
              {/if}
              {#if !p.releasePending && !p.releaseIncoming && p.connected}
                <button type="button" class="small soft-button" disabled={busy} onclick={() => run("petbond:requestRelease", { masterId: me.id, petId: p.playerId })}>申请解除关系</button>
              {/if}
            </span>
          </div>
        {/each}
        {#if bondState.pets.length === 0}<p class="hint">你还没有宠物</p>{/if}
      </div>
      {#if me.bondPetEnabled}
        <div class="pet-bond-candidate-list">
          <p class="hint">开启认主的在线玩家：</p>
          {#each petCandidates as c (c.playerId)}
            {@const player = resolveLobbyPlayer(lobby.players, c.playerId, petBondBadgeFallback(c))}
            <div class={`pet-bond-member-row ${c.incoming ? "pet-bond-pin" : ""}`}>
              <PetBondPlayerInfo {player} />
              <span class="pet-bond-row-actions">
                {#if c.incoming && c.incomingId}
                  <button type="button" class="small primary" disabled={busy} onclick={() => run("petbond:approve", { requestId: c.incomingId }, "已同意")}>{c.incomingLabel || "同意认主请求"}</button>
                {:else if c.status === "pending" && c.requestId}
                  <button type="button" class="small danger-soft" disabled={busy} onclick={() => run("petbond:cancel", { requestId: c.requestId }, "已撤销申请")}>撤销申请</button>
                {:else}
                  <button type="button" class="small" disabled={busy || petSlotsLeft <= 0} onclick={() => run("petbond:seekPet", { targetId: c.playerId })}>申请认宠</button>
                {/if}
              </span>
            </div>
          {/each}
          {#if petCandidates.length === 0}<p class="empty">暂无可认宠对象</p>{/if}
        </div>
      {:else}
        <p class="hint">在个人设置开启「开启认宠」后，可向在线玩家申请认宠。</p>
      {/if}
    </section>
  </div>

  {#if titlePetId}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="modal-backdrop" onclick={() => (titlePetId = null)}>
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <section class="logout-confirm-card" onclick={(e) => e.stopPropagation()}>
        <h3>设置宠物称号</h3>
        <p class="hint">设置对方称号标签，最多 {petBondCfg.maxTitleLength} 字。清空则恢复默认称号。</p>
        <input value={titleDraft} maxlength={petBondCfg.maxTitleLength} oninput={(e) => (titleDraft = e.currentTarget.value)} placeholder="例如：小奶猫" />
        <div class="kick-confirm-actions">
          <button type="button" disabled={busy} onclick={() => (titlePetId = null)}>取消</button>
          <button
            type="button"
            class="primary"
            disabled={busy}
            onclick={async () => {
              await run("petbond:setTitle", { petId: titlePetId, title: titleDraft }, "称号已更新");
              titlePetId = null;
            }}
          >保存</button>
        </div>
      </section>
    </div>
  {/if}
</div>
