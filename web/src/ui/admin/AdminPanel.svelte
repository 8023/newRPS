<script lang="ts">
  /**
   * 后台管理面板composition root：登录门禁、侧边栏导航、9 个配置分区 + 数据分析 + 共建审核
   * 的分发、底部粘性保存/重置操作条。真正的编辑状态全部在 adminStore；本文件只负责
   * "当前该显示哪个分区"这一层路由，不持有任何配置字段——与既有 uiStore/routerStore/
   * sessionStore 分层一致，避免重蹈原 React App.tsx 上帝组件的覆辙。
   * 源：ui/AdminViews.tsx:97-1387（AdminPanel 组件）。
   */
  import RefreshCcw from "@lucide/svelte/icons/refresh-ccw";
  import Save from "@lucide/svelte/icons/save";
  import Settings from "@lucide/svelte/icons/settings";
  import Shield from "@lucide/svelte/icons/shield";
  import { formatContributionReviewSubtitle } from "../contributeSeries";
  import { adminStore, type AdminSection } from "../../lib/stores/adminStore.svelte";
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { routerStore } from "../../lib/stores/routerStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import AdminSiteSection from "./AdminSiteSection.svelte";
  import AdminFactionsSection from "./AdminFactionsSection.svelte";
  import AdminTitlesSection from "./AdminTitlesSection.svelte";
  import AdminPunishmentsSection from "./AdminPunishmentsSection.svelte";
  import AdminNameWarSection from "./AdminNameWarSection.svelte";
  import AdminGiveawaySection from "./AdminGiveawaySection.svelte";
  import AdminPetBondSection from "./AdminPetBondSection.svelte";
  import AdminRoomTagsSection from "./AdminRoomTagsSection.svelte";
  import AdminRoomsSection from "./AdminRoomsSection.svelte";
  import AdminContributionReview from "./AdminContributionReview.svelte";
  import type { Component } from "svelte";

  const SECTIONS_WITHOUT_SAVE = new Set<AdminSection>(["rooms", "analytics", "contributions"]);

  const draft = $derived(adminStore.draft);
  const lobby = $derived(sessionStore.lobby);

  // 草稿由服务端 config 单向同步进来；干净时直接采用新配置，脏（用户正在改）时只标记
  // "服务器配置已更新"，保存会覆盖服务器这份新配置。
  $effect(() => {
    adminStore.syncFromServerConfig(sessionStore.config);
  });
  $effect(() => {
    if (adminStore.logged) void adminStore.loadContributionCounts();
  });

  // 数据分析图表库单独成一个 chunk，只有管理员点开「数据分析」时才下载，连管理员改配置
  // 都不会加载——对应原版 React 的 lazy(() => import("./AnalyticsPanel"))。
  let AnalyticsPanelComponent = $state<Component<{ config: import("../../shared/types").AppConfig; onError: (message: string) => void }> | null>(null);
  $effect(() => {
    if (adminStore.activeSection === "analytics" && !AnalyticsPanelComponent) {
      import("./AnalyticsPanel.svelte").then((m) => (AnalyticsPanelComponent = m.default as typeof AnalyticsPanelComponent));
    }
  });

  const navItems = $derived.by((): Array<{ id: AdminSection; label: string; detail: string }> => {
    if (!draft || !lobby) return [];
    return [
      { id: "site", label: "网站管理", detail: `${draft.site.name} · ${Object.keys(draft.messages).length} 条文案` },
      { id: "analytics", label: "数据分析", detail: "访问 · 游戏 · 渠道" },
      { id: "factions", label: "性别与阵营", detail: `${draft.genders.length} 个性别 · ${draft.genderFactions.length} 个阵营` },
      { id: "titles", label: "排位与称号", detail: `${draft.titles.length} 个段位 · 积分展示上下限` },
      { id: "punishments", label: "惩罚任务", detail: `${(draft.punishmentTags || []).length} 个惩罚标签` },
      { id: "contributions", label: "共建审核", detail: formatContributionReviewSubtitle(adminStore.contributionCounts) },
      { id: "roomTags", label: "房间标签", detail: `${draft.roomTags.length} 个标签 · 房间头部彩色标签` },
      { id: "nameWar", label: "名争 / 极限", detail: `${draft.nameWar.penaltyPrefix} · ${draft.extremeMode.emoji} ${draft.extremeMode.label}` },
      { id: "giveaway", label: "白给模式", detail: draft.giveaway.panelTitle },
      { id: "petBond", label: "宠物乐园", detail: draft.petBond?.panelTitle || "宠物乐园" },
      { id: "rooms", label: "用户与房间", detail: `${lobby.onlineCount} 人在线 · ${lobby.rooms.length} 个房间` }
    ];
  });
  const currentNav = $derived(navItems.find((item) => item.id === adminStore.activeSection) || navItems[0]);
</script>

<section class="admin-page">
  <div class="panel admin-login-card">
    <h2><Shield size={18} /> 管理员与文本工具</h2>
    <input type="password" value={adminStore.password} oninput={(event) => (adminStore.password = event.currentTarget.value)} placeholder="管理员口令" />
    <button class="primary" onclick={() => adminStore.login()}>进入管理</button>
    <button onclick={() => routerStore.goto("lobby")}>返回</button>
  </div>
  {#if adminStore.logged && draft && lobby && currentNav}
    <div class="admin-tool-shell">
      <nav class="admin-sidebar" aria-label="后台配置分类">
        {#each navItems as item (item.id)}
          <button class={adminStore.activeSection === item.id ? "active" : ""} onclick={() => (adminStore.activeSection = item.id)}>
            <span>{item.label}</span>
            <small>{item.detail}</small>
          </button>
        {/each}
      </nav>
      <div class="panel visual-config admin-editor-panel">
        <div class="admin-editor-head">
          <div>
            <h2><Settings size={18} /> {currentNav.label}</h2>
            <p class="hint">{currentNav.detail}</p>
          </div>
          <div class="admin-edit-status">
            {#if adminStore.dirty}<span>有未保存修改</span>{/if}
            {#if adminStore.serverConfigChanged}<small>服务器配置已更新，保存会覆盖当前服务器配置。</small>{/if}
          </div>
        </div>

        {#if adminStore.activeSection === "analytics"}
          {#if AnalyticsPanelComponent}
            <AnalyticsPanelComponent config={draft} onError={(m) => uiStore.notify(m)} />
          {:else}
            <div class="analytics-loading">正在加载图表…</div>
          {/if}
        {:else if adminStore.activeSection === "site"}
          <AdminSiteSection />
        {:else if adminStore.activeSection === "contributions"}
          <AdminContributionReview />
        {:else if adminStore.activeSection === "factions"}
          <AdminFactionsSection />
        {:else if adminStore.activeSection === "titles"}
          <AdminTitlesSection />
        {:else if adminStore.activeSection === "punishments"}
          <AdminPunishmentsSection />
        {:else if adminStore.activeSection === "nameWar"}
          <AdminNameWarSection />
        {:else if adminStore.activeSection === "giveaway"}
          <AdminGiveawaySection />
        {:else if adminStore.activeSection === "petBond"}
          <AdminPetBondSection />
        {:else if adminStore.activeSection === "roomTags"}
          <AdminRoomTagsSection />
        {:else if adminStore.activeSection === "rooms"}
          <AdminRoomsSection />
        {/if}

        {#if !SECTIONS_WITHOUT_SAVE.has(adminStore.activeSection)}
          <div class="admin-sticky-actions">
            <button class="primary" onclick={() => adminStore.save()}><Save size={16} /> 保存配置</button>
            <button onclick={() => adminStore.resetDefault()} title="从磁盘重新读取 config/*.json"><RefreshCcw size={16} /> 重新加载</button>
          </div>
        {/if}
      </div>
    </div>
  {/if}
</section>
