// 后台管理面板的会话/草稿状态：登录态、配置草稿（含脏检查与服务端并发更新提示）、
// 用户管理列表、公告发送、共建审核徽标计数。与 sessionStore 同构——这里是后台配置
// 编辑器专属的状态，不属于全站会话，所以单独开一个 store 而不是塞进 sessionStore。
// 依赖方向：adminStore → uiStore（notify），不依赖 sessionStore（config 由调用方
// 通过 syncFromServerConfig 喂入，保持单向数据流，避免循环 import）。
import type { AppConfig, PublicPlayer } from "../../shared/types";
import { ask } from "../rpc";
import { prepareProofImageForUpload } from "../proofImage";
import { DEFAULT_ADMIN_PLAYER_FILTERS, type AdminPlayerFilters } from "../adminHelpers";
import type { ContributionReviewCounts } from "../contributionAdmin";
import { uiStore } from "./uiStore.svelte";

export type AdminSection = "site" | "analytics" | "factions" | "titles" | "punishments" | "contributions" | "roomTags" | "nameWar" | "giveaway" | "petBond" | "rooms";
export type AdminRoomTab = "rooms" | "announcement" | "users";
export type { ContributionReviewCounts };

class AdminStore {
  password = $state("");
  logged = $state(false);
  draft = $state.raw<AppConfig | null>(null);
  dirty = $state(false);
  serverConfigChanged = $state(false);

  activeSection = $state<AdminSection>("site");
  activeFactionId = $state("");
  factionSearch = $state("");
  activeTitleId = $state("");
  titleSearch = $state("");
  activeTagId = $state("");
  punishmentSearch = $state("");
  announcementMessage = $state("");
  announcementSeconds = $state("8");
  activeRoomTab = $state<AdminRoomTab>("rooms");

  playerFilters = $state<AdminPlayerFilters>(DEFAULT_ADMIN_PLAYER_FILTERS);
  playerNameSearch = $state("");
  adminPlayers = $state.raw<PublicPlayer[]>([]);
  adminPlayersTotal = $state(0);
  adminPlayersTruncated = $state(false);
  adminFilterOnlineCount = $state(0);
  adminFilterOfflineCount = $state(0);
  adminPlayersLoading = $state(false);

  contributionCounts = $state<ContributionReviewCounts>({ pending: 0 });

  #lastServerConfigText = "";
  #adminPlayersRequestGen = 0;
  #sessionGeneration = 0;

  /** 恢复 React 版 AdminPanel 卸载语义：后台口令、登录态、草稿及各分区局部状态
      都不跨后台会话保留。递增请求代次，让已经离场的玩家列表请求不能回写新会话。 */
  resetSession() {
    this.#sessionGeneration += 1;
    this.password = "";
    this.logged = false;
    this.draft = null;
    this.dirty = false;
    this.serverConfigChanged = false;

    this.activeSection = "site";
    this.activeFactionId = "";
    this.factionSearch = "";
    this.activeTitleId = "";
    this.titleSearch = "";
    this.activeTagId = "";
    this.punishmentSearch = "";
    this.announcementMessage = "";
    this.announcementSeconds = "8";
    this.activeRoomTab = "rooms";

    this.playerFilters = { ...DEFAULT_ADMIN_PLAYER_FILTERS };
    this.playerNameSearch = "";
    this.adminPlayers = [];
    this.adminPlayersTotal = 0;
    this.adminPlayersTruncated = false;
    this.adminFilterOnlineCount = 0;
    this.adminFilterOfflineCount = 0;
    this.adminPlayersLoading = false;
    this.contributionCounts = { pending: 0 };

    this.#lastServerConfigText = "";
    this.#adminPlayersRequestGen += 1;
  }

  /** 服务端 config:update 推送到达时的合并策略：草稿干净就直接采用新配置；草稿脏（用户
      正在改）则只标记"服务器配置已更新"，保存时会覆盖服务器这份新配置——与原 React 版
      App.tsx 里 AdminPanel 组件内联的那个 effect 语义一致，抽成方法供 AdminPanel.svelte
      用 $effect 包一层调用。 */
  syncFromServerConfig(config: AppConfig | null) {
    if (!config) return;
    const nextText = JSON.stringify(config);
    if (nextText === this.#lastServerConfigText) return;
    this.#lastServerConfigText = nextText;
    if (this.dirty) {
      this.serverConfigChanged = true;
      return;
    }
    this.#applyServerConfig(config);
  }

  #applyServerConfig(nextConfig: AppConfig) {
    this.#lastServerConfigText = JSON.stringify(nextConfig);
    this.draft = nextConfig;
    if (!nextConfig.genderFactions.some((item) => item.id === this.activeFactionId)) {
      this.activeFactionId = nextConfig.genderFactions[0]?.id || "";
    }
    if (!nextConfig.titles.some((item) => item.id === this.activeTitleId)) {
      this.activeTitleId = nextConfig.titles[0]?.id || "";
    }
    if (!(nextConfig.punishmentTags || []).some((item) => item.id === this.activeTagId)) {
      this.activeTagId = nextConfig.punishmentTags?.[0]?.id || "";
    }
    this.dirty = false;
    this.serverConfigChanged = false;
  }

  async login() {
    const generation = this.#sessionGeneration;
    try {
      await ask("admin:login", { password: this.password });
      if (generation !== this.#sessionGeneration) return;
      this.logged = true;
    } catch (error) {
      uiStore.notify(error instanceof Error ? error.message : "登录失败");
    }
  }

  async loadContributionCounts() {
    if (!this.logged) return;
    const generation = this.#sessionGeneration;
    try {
      // 侧边栏徽标只需要一个总数，不用打开整个共建审核面板，复用同一个总览接口
      // （AdminContributionReview.svelte 展开后台数据时也走它）。
      const res = await ask<{ counts?: { task?: Record<string, number>; series?: Record<string, number> } }>(
        "admin:action", { action: "contributionPendingOverview", password: this.password }
      );
      if (generation !== this.#sessionGeneration) return;
      const t = res.counts?.task || {};
      const s = res.counts?.series || {};
      this.contributionCounts = { pending: (Number(t.pending) || 0) + (Number(s.pending) || 0) };
    } catch {
      // 侧边栏摘要拉取失败不影响主流程，静默忽略。
    }
  }

  async save() {
    if (!this.draft) return;
    const generation = this.#sessionGeneration;
    try {
      const response = await ask<{ config: AppConfig }>("config:save", { password: this.password, nextConfig: this.draft });
      if (generation === this.#sessionGeneration) this.#applyServerConfig(response.config);
      uiStore.notify("配置保存成功");
    } catch (error) {
      uiStore.notify(error instanceof Error ? error.message : "配置保存失败");
    }
  }

  async resetDefault() {
    const generation = this.#sessionGeneration;
    try {
      // 配置已按功能拆分并原地读写，无 default/active 双轨；此处从磁盘重新加载当前文件。
      const response = await ask<{ config: AppConfig }>("config:reset", { password: this.password });
      if (generation === this.#sessionGeneration) this.#applyServerConfig(response.config);
      uiStore.notify("已从磁盘重新加载配置");
    } catch (error) {
      uiStore.notify(error instanceof Error ? error.message : "重新加载配置失败");
    }
  }

  patch(next: Partial<AppConfig>) {
    if (!this.draft) return;
    this.dirty = true;
    this.draft = { ...this.draft, ...next };
  }

  async action(actionName: string, payload: Record<string, unknown> = {}): Promise<boolean> {
    try {
      await ask("admin:action", { action: actionName, password: this.password, ...payload });
      return true;
    } catch (error) {
      uiStore.notify(error instanceof Error ? error.message : "管理操作失败");
      return false;
    }
  }

  async loadAdminPlayers(filters: AdminPlayerFilters = this.playerFilters) {
    if (!this.logged || this.activeSection !== "rooms" || this.activeRoomTab !== "users") return;
    const gen = ++this.#adminPlayersRequestGen;
    this.adminPlayersLoading = true;
    try {
      const response = await ask<{
        players?: PublicPlayer[];
        total?: number;
        onlineCount?: number;
        offlineCount?: number;
        truncated?: boolean;
      }>("admin:listPlayers", {
        online: filters.online,
        nameWar: filters.nameWar,
        sortGiveawayDesc: filters.sortGiveawayDesc,
        sortRankedDesc: filters.sortRankedDesc,
        recentLogin7d: filters.recentLogin7d,
        rankedNonZero: filters.rankedNonZero
      });
      if (gen !== this.#adminPlayersRequestGen) return;
      this.adminPlayers = Array.isArray(response.players) ? response.players : [];
      this.adminPlayersTotal = Number(response.total) || 0;
      this.adminPlayersTruncated = !!response.truncated;
      this.adminFilterOnlineCount = Number(response.onlineCount) || 0;
      this.adminFilterOfflineCount = Number(response.offlineCount) || 0;
    } catch (error) {
      if (gen !== this.#adminPlayersRequestGen) return;
      uiStore.notify(error instanceof Error ? error.message : "加载玩家列表失败");
    } finally {
      if (gen === this.#adminPlayersRequestGen) this.adminPlayersLoading = false;
    }
  }

  togglePlayerFilter(key: keyof AdminPlayerFilters) {
    const next = { ...this.playerFilters, [key]: !this.playerFilters[key] };
    this.playerFilters = next;
    void this.loadAdminPlayers(next);
  }

  async sendAnnouncement() {
    const generation = this.#sessionGeneration;
    try {
      await ask("admin:action", {
        action: "broadcastAnnouncement",
        password: this.password,
        message: this.announcementMessage,
        durationSeconds: Number(this.announcementSeconds)
      });
      if (generation === this.#sessionGeneration) this.announcementMessage = "";
      uiStore.notify("公告已发送");
    } catch (error) {
      uiStore.notify(error instanceof Error ? error.message : "公告发送失败");
    }
  }

  async uploadAdminImage(file: File): Promise<string> {
    const uploadFile = await prepareProofImageForUpload(file);
    const form = new FormData();
    form.append("password", this.password);
    form.append("image", uploadFile, uploadFile.name);
    const response = await fetch("/api/admin-image", { method: "POST", body: form });
    let data: { message?: string; imageUrl?: string } = {};
    try {
      data = await response.json();
    } catch {
      throw new Error(response.ok ? "服务器响应无效" : `上传失败（${response.status}）`);
    }
    if (!response.ok) throw new Error(data.message || "上传失败");
    return data.imageUrl as string;
  }
}

export const adminStore = new AdminStore();
