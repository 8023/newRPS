import { type CSSProperties, Suspense, lazy, useEffect, useRef, useState } from "react";
import { Download, RefreshCcw, Save, Settings, Shield, Upload } from "lucide-react";
import type { AppConfig, GenderFaction, LobbySnapshot, PublicPlayer, PunishmentSeriesTaskConfig, PunishmentTaskConfig, RoomInfoTagStyle, RoomNamePool } from "../shared/types";
import { DEFAULT_NAME_WAR_PENALTY_THRESHOLD, DEFAULT_NAME_WAR_RENAME_MIN_POINTS, normalizePunishmentSeries, normalizePunishmentTasks, withAccessControlDefaults, withRankedScoreDefaults } from "../lib/normalize";
import { ask } from "../lib/rpc";
import { prepareProofImageForUpload } from "../lib/proofImage";
import { formatBytes, formatDuration } from "../lib/format";
import { encodeClaimCode } from "../lib/session";
import { PetBondGraphPanel } from "./PetBondGraphPanel";
import {
  FactionSelect, GenderSelectField, GiveawayChip, ModeChip, PlayerAvatar, PlayerBadge, RoomInfoTagList, RoomTagList, Select, Stat, Toggle,
  defaultRoomInfoTagStyle, displayPlayerName, factionStyle, formatGiveawayValue, genderChoiceError, genderStyle, lobbyRoomInfoTags, nextGenderIdForFaction,
  roomInfoTagOrder, roomInfoTagStyle, roomStatusText, safePlayerStats, titleStyle, titleTagStyleOrder
} from "./AppViews";

// recharts 只在管理员点开「数据分析」时才下载：AdminPanel 本身已是 lazy chunk，
// 这里再嵌一层让图表库单独成块，连管理员改配置时都不会加载。
const AnalyticsPanel = lazy(() => import("./AnalyticsPanel").then((m) => ({ default: m.AnalyticsPanel })));

export type AdminSection = "site" | "analytics" | "factions" | "titles" | "punishments" | "roomTags" | "nameWar" | "giveaway" | "petBond" | "accessControl" | "users" | "rooms";

const SECTIONS_WITHOUT_SAVE = new Set<AdminSection>(["users", "rooms", "analytics"]);
export type AdminRoomTab = "rooms" | "announcement";

/** 用户管理的筛选/排序开关（与后端 admin:listPlayers 字段对应）。 */
export type AdminPlayerFilters = {
  online: boolean;
  nameWar: boolean;
  sortGiveawayDesc: boolean;
  sortRankedDesc: boolean;
  recentLogin7d: boolean;
  rankedNonZero: boolean;
};

/**
 * 默认：在线 / 7天内登录 / 积分不为0 开启；
 * 开启名争、白给降序、积分降序关闭；未开排序时后端按积分升序。
 */
export const DEFAULT_ADMIN_PLAYER_FILTERS: AdminPlayerFilters = {
  online: true,
  nameWar: false,
  sortGiveawayDesc: false,
  sortRankedDesc: false,
  recentLogin7d: true,
  rankedNonZero: true
};

const ADMIN_PLAYER_FILTER_BUTTONS: { key: keyof AdminPlayerFilters; label: string }[] = [
  { key: "online", label: "在线" },
  { key: "nameWar", label: "开启名争" },
  { key: "sortGiveawayDesc", label: "白给降序" },
  { key: "sortRankedDesc", label: "积分降序" },
  { key: "recentLogin7d", label: "7天内登录" },
  { key: "rankedNonZero", label: "积分不为0" }
];

// 任务分组固定三选一：决定系统任务/称号按哪份文案分发，与阵营是多对一关系
// （比如“顺性别男”“男跨女”都属于 male，写系统任务时只需要写一份 male 文案）。
export const TASK_GROUP_OPTIONS: { id: string; label: string }[] = [
  { id: "male", label: "生理男" },
  { id: "female", label: "生理女" },
  { id: "default", label: "默认兜底" }
];

export function taskGroupLabel(group: string) {
  return TASK_GROUP_OPTIONS.find((item) => item.id === group)?.label || group || "默认兜底";
}

// 按当前配置里实际出现过的任务分组去重分组阵营，用于称号池编辑器——
// 与后端 internal/config/config.go 的 distinctTaskGroups 保持一致的语义。
export function taskGroupsFromFactions(factions: GenderFaction[]) {
  const byGroup = new Map<string, GenderFaction[]>();
  for (const faction of factions) {
    const group = faction.taskGroup || "default";
    if (!byGroup.has(group)) byGroup.set(group, []);
    byGroup.get(group)!.push(faction);
  }
  return [...byGroup.entries()].map(([id, members]) => ({
    id,
    label: taskGroupLabel(id),
    memberLabel: members.map((item) => item.label).join("、")
  }));
}

export function AdminPanel({ config, lobby, onBack, onError }: { config: AppConfig; lobby: LobbySnapshot; onBack: () => void; onError: (message: string) => void }) {
  const [password, setPassword] = useState("");
  const [logged, setLogged] = useState(false);
  const [draft, setDraft] = useState<AppConfig>(config);
  const [activeSection, setActiveSection] = useState<AdminSection>("site");
  const [activeFactionId, setActiveFactionId] = useState(config.genderFactions[0]?.id || "");
  const [factionSearch, setFactionSearch] = useState("");
  const [activeTitleId, setActiveTitleId] = useState(config.titles[0]?.id || "");
  const [titleSearch, setTitleSearch] = useState("");
  const [punishmentTab, setPunishmentTab] = useState<"config" | "series" | "pool">("config");
  const [activeTagId, setActiveTagId] = useState(config.punishmentTags?.[0]?.id || "");
  const [punishmentSearch, setPunishmentSearch] = useState("");
  // 任务池 / 系列任务：独立草稿，与 config:save 无关
  const [taskPoolDraft, setTaskPoolDraft] = useState<PunishmentTaskConfig[]>([]);
  const [taskPoolLoaded, setTaskPoolLoaded] = useState(false);
  const [taskPoolDirty, setTaskPoolDirty] = useState(false);
  const [activeTaskId, setActiveTaskId] = useState("");
  const [seriesDraft, setSeriesDraft] = useState<PunishmentSeriesTaskConfig[]>([]);
  const [seriesLoaded, setSeriesLoaded] = useState(false);
  const [seriesDirty, setSeriesDirty] = useState(false);
  const [activeSeriesId, setActiveSeriesId] = useState("");
  const [taskPickerStep, setTaskPickerStep] = useState<{ seriesId: string; stepIndex: number } | null>(null);
  const [taskPickerQuery, setTaskPickerQuery] = useState("");
  const [announcementMessage, setAnnouncementMessage] = useState("");
  const [announcementSeconds, setAnnouncementSeconds] = useState("8");
  const [activeRoomTab, setActiveRoomTab] = useState<AdminRoomTab>("rooms");
  const [playerFilters, setPlayerFilters] = useState<AdminPlayerFilters>(DEFAULT_ADMIN_PLAYER_FILTERS);
  // 昵称搜索：纯前端对 admin:listPlayers 已按过滤器返回的列表做模糊匹配，不发额外请求；
  // 只匹配 player.name（原始昵称），不含称号/惩罚名/性别等展示层信息。
  const [playerNameSearch, setPlayerNameSearch] = useState("");
  const [adminPlayers, setAdminPlayers] = useState<PublicPlayer[]>([]);
  const [adminPlayersTotal, setAdminPlayersTotal] = useState(0);
  const [adminPlayersTruncated, setAdminPlayersTruncated] = useState(false);
  const [adminFilterOnlineCount, setAdminFilterOnlineCount] = useState(0);
  const [adminFilterOfflineCount, setAdminFilterOfflineCount] = useState(0);
  const [adminPlayersLoading, setAdminPlayersLoading] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [serverConfigChanged, setServerConfigChanged] = useState(false);
  const lastServerConfigText = useRef(JSON.stringify(config));
  const adminPlayersRequestGen = useRef(0);

  useEffect(() => {
    const nextText = JSON.stringify(config);
    if (nextText === lastServerConfigText.current) return;
    lastServerConfigText.current = nextText;
    if (dirty) {
      setServerConfigChanged(true);
      return;
    }
    applyServerConfig(config);
  }, [config, dirty]);

  function applyServerConfig(nextConfig: AppConfig) {
    lastServerConfigText.current = JSON.stringify(nextConfig);
    setDraft(nextConfig);
    setActiveFactionId((old) => nextConfig.genderFactions.some((item) => item.id === old) ? old : nextConfig.genderFactions[0]?.id || "");
    setActiveTitleId((old) => nextConfig.titles.some((item) => item.id === old) ? old : nextConfig.titles[0]?.id || "");
    setActiveTagId((old) => (nextConfig.punishmentTags || []).some((item) => item.id === old) ? old : nextConfig.punishmentTags?.[0]?.id || "");
    setDirty(false);
    setServerConfigChanged(false);
  }

  async function ensureTaskPoolLoaded() {
    if (taskPoolLoaded) return taskPoolDraft;
    try {
      const res = await ask<{ tasks: PunishmentTaskConfig[] }>("admin:action", { action: "punishmentTasksGet" });
      const tasks = normalizePunishmentTasks(res.tasks);
      setTaskPoolDraft(tasks);
      setTaskPoolLoaded(true);
      setTaskPoolDirty(false);
      setActiveTaskId((old) => tasks.some((t) => t.id === old) ? old : tasks[0]?.id || "");
      return tasks;
    } catch (error) {
      onError(error instanceof Error ? error.message : "加载任务池失败");
      return taskPoolDraft;
    }
  }

  async function ensureSeriesLoaded() {
    if (seriesLoaded) return seriesDraft;
    // 系列编辑需要任务池做选择器
    await ensureTaskPoolLoaded();
    try {
      const res = await ask<{ series: PunishmentSeriesTaskConfig[] }>("admin:action", { action: "punishmentSeriesGet" });
      const series = normalizePunishmentSeries(res.series);
      setSeriesDraft(series);
      setSeriesLoaded(true);
      setSeriesDirty(false);
      setActiveSeriesId((old) => series.some((s) => s.id === old) ? old : series[0]?.id || "");
      return series;
    } catch (error) {
      onError(error instanceof Error ? error.message : "加载系列任务失败");
      return seriesDraft;
    }
  }

  async function saveTaskPool() {
    try {
      const res = await ask<{ tasks: PunishmentTaskConfig[] }>("admin:action", {
        action: "punishmentTasksSave",
        tasks: taskPoolDraft
      });
      const tasks = normalizePunishmentTasks(res.tasks);
      setTaskPoolDraft(tasks);
      setTaskPoolDirty(false);
      setActiveTaskId((old) => tasks.some((t) => t.id === old) ? old : tasks[0]?.id || "");
      onError("任务池已保存");
    } catch (error) {
      onError(error instanceof Error ? error.message : "保存任务池失败");
    }
  }

  async function saveSeriesPool() {
    try {
      const res = await ask<{ series: PunishmentSeriesTaskConfig[] }>("admin:action", {
        action: "punishmentSeriesSave",
        series: seriesDraft
      });
      const series = normalizePunishmentSeries(res.series);
      setSeriesDraft(series);
      setSeriesDirty(false);
      setActiveSeriesId((old) => series.some((s) => s.id === old) ? old : series[0]?.id || "");
      onError("系列任务已保存");
    } catch (error) {
      onError(error instanceof Error ? error.message : "保存系列任务失败");
    }
  }

  function patchTaskPool(next: PunishmentTaskConfig[]) {
    setTaskPoolDraft(next);
    setTaskPoolDirty(true);
  }

  function patchSeriesDraft(next: PunishmentSeriesTaskConfig[]) {
    setSeriesDraft(next);
    setSeriesDirty(true);
  }

  async function login() {
    try {
      await ask("admin:login", { password });
      setLogged(true);
    } catch (error) {
      onError(error instanceof Error ? error.message : "登录失败");
    }
  }

  async function save() {
    try {
      const response = await ask<{ config: AppConfig }>("config:save", { password, nextConfig: draft });
      applyServerConfig(response.config);
      onError("配置保存成功");
    } catch (error) {
      onError(error instanceof Error ? error.message : "配置保存失败");
    }
  }

  async function resetDefault() {
    try {
      // 配置已按功能拆分并原地读写，无 default/active 双轨；此处从磁盘重新加载当前文件。
      const response = await ask<{ config: AppConfig }>("config:reset", { password });
      applyServerConfig(response.config);
      onError("已从磁盘重新加载配置");
    } catch (error) {
      onError(error instanceof Error ? error.message : "重新加载配置失败");
    }
  }

  async function exportConfig() {
    try {
      const response = await fetch("/api/config/export", {
        method: "GET",
        headers: { "Accept": "application/json", "X-Admin-Password": password }
      });
      if (!response.ok) {
        const data = await response.json().catch(() => null);
        throw new Error(data?.message || "配置导出失败");
      }
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = "rps-config.json";
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(url);
      onError("配置已导出");
    } catch (error) {
      onError(error instanceof Error ? error.message : "配置导出失败");
    }
  }

  function patch(next: Partial<AppConfig>) {
    setDirty(true);
    setDraft((old) => ({ ...old, ...next }));
  }

  function patchFactions(nextFactions: GenderFaction[]) {
    patch({ genderFactions: nextFactions });
  }

  function patchGenders(nextGenders: AppConfig["genders"]) {
    patch({ genders: nextGenders });
  }

  async function action(actionName: string, payload: Record<string, unknown> = {}) {
    try {
      await ask("admin:action", { action: actionName, ...payload });
      return true;
    } catch (error) {
      onError(error instanceof Error ? error.message : "管理操作失败");
      return false;
    }
  }

  async function loadAdminPlayers(filters: AdminPlayerFilters = playerFilters) {
    if (!logged || activeSection !== "users") return;
    const gen = ++adminPlayersRequestGen.current;
    setAdminPlayersLoading(true);
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
      if (gen !== adminPlayersRequestGen.current) return;
      setAdminPlayers(Array.isArray(response.players) ? response.players : []);
      setAdminPlayersTotal(Number(response.total) || 0);
      setAdminPlayersTruncated(!!response.truncated);
      setAdminFilterOnlineCount(Number(response.onlineCount) || 0);
      setAdminFilterOfflineCount(Number(response.offlineCount) || 0);
    } catch (error) {
      if (gen !== adminPlayersRequestGen.current) return;
      onError(error instanceof Error ? error.message : "加载玩家列表失败");
    } finally {
      if (gen === adminPlayersRequestGen.current) setAdminPlayersLoading(false);
    }
  }

  function togglePlayerFilter(key: keyof AdminPlayerFilters) {
    const next = { ...playerFilters, [key]: !playerFilters[key] };
    setPlayerFilters(next);
    void loadAdminPlayers(next);
  }

  useEffect(() => {
    if (!logged || activeSection !== "users") return;
    void loadAdminPlayers(playerFilters);
    // 仅在进入用户管理分区时拉取；过滤器切换由 togglePlayerFilter 负责。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [logged, activeSection]);

  async function sendAnnouncement() {
    try {
      await ask("admin:action", {
        action: "broadcastAnnouncement",
        message: announcementMessage,
        durationSeconds: Number(announcementSeconds)
      });
      setAnnouncementMessage("");
      onError("公告已发送");
    } catch (error) {
      onError(error instanceof Error ? error.message : "公告发送失败");
    }
  }

  async function uploadAdminImage(file: File) {
    const uploadFile = await prepareProofImageForUpload(file);
    const form = new FormData();
    form.append("password", password);
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

  const navItems: Array<{ id: AdminSection; label: string; detail: string }> = [
    { id: "site", label: "网站管理", detail: `${draft.site.name} · ${Object.keys(draft.messages).length} 条文案` },
    { id: "analytics", label: "数据分析", detail: "访问 · 游戏 · 渠道" },
    { id: "factions", label: "性别与阵营", detail: `${draft.genders.length} 个性别 · ${draft.genderFactions.length} 个阵营` },
    { id: "titles", label: "排位与称号", detail: `${draft.titles.length} 个段位 · 积分展示上下限` },
    { id: "punishments", label: "惩罚任务", detail: `${(draft.punishmentTags || []).length} 标签 · 任务池/系列独立保存` },
    { id: "roomTags", label: "房间标签", detail: `${draft.roomTags.length} 个标签 · 房间头部彩色标签` },
    { id: "nameWar", label: "名争 / 极限", detail: `${draft.nameWar.penaltyPrefix} · ${draft.extremeMode.emoji} ${draft.extremeMode.label}` },
    { id: "giveaway", label: "白给模式", detail: draft.giveaway.panelTitle },
    { id: "petBond", label: "宠物乐园", detail: draft.petBond?.panelTitle || "宠物乐园" },
    { id: "accessControl", label: "防多开", detail: (() => { const ac = withAccessControlDefaults(draft.accessControl); return ac.registrationDisabled ? "已禁止新用户注册" : `同指纹 ${ac.maxOnlinePerIp} 在线 / ${ac.maxCreatesPer10Min} 新建`; })() },
    { id: "users", label: "用户管理", detail: playerFilters.online ? `在线筛选 · ${adminPlayersTotal} 人` : `${adminPlayersTotal} 人` },
    { id: "rooms", label: "房间管理", detail: `${lobby.rooms.length} 房间 · 在线 ${lobby.onlineCount}` }
  ];

  const currentNav = navItems.find((item) => item.id === activeSection) || navItems[0];

  function switchSection(section: AdminSection) {
    setActiveSection(section);
  }

  function renderSection() {
    if (activeSection === "analytics") {
      return (
        <Suspense fallback={<div className="analytics-loading">正在加载图表…</div>}>
          <AnalyticsPanel onError={onError} config={config} />
        </Suspense>
      );
    }
    if (activeSection === "site") {
      return (
        <>
          <div className="config-section admin-section-card">
            <AdminSectionHeader title="提示公告" subtitle="修改公告板、密码错误、名字校验、保存提示等系统文案。" />
            <div className="admin-announcement-card">
              <div className="admin-card-title">
                <strong>发送全服公告</strong>
                <small>当前在线玩家和后台页面会立即弹出</small>
              </div>
              <textarea
                value={announcementMessage}
                maxLength={200}
                onChange={(event) => setAnnouncementMessage(event.target.value)}
                placeholder="输入公告内容，最多 200 字"
              />
              <div className="admin-announcement-actions">
                <label className="field-label">
                  <span>显示秒数</span>
                  <input type="number" min={3} max={60} value={announcementSeconds} onChange={(event) => setAnnouncementSeconds(event.target.value)} />
                </label>
                <button className="primary" onClick={sendAnnouncement}>发送公告</button>
              </div>
            </div>
            <div className="admin-announcement-card">
              <div className="admin-card-title">
                <strong>公告板</strong>
                <small>展示在顶栏「关于」面板里，不再是弹窗</small>
              </div>
              <Toggle
                label="开启公告板"
                value={draft.announcementBoard.enabled ?? false}
                onChange={(enabled) => patch({ announcementBoard: { ...draft.announcementBoard, enabled } })}
              />
              <label className="field-label">
                <span>公告标题</span>
                <input value={draft.announcementBoard.title} maxLength={32} onChange={(event) => patch({ announcementBoard: { ...draft.announcementBoard, title: event.target.value } })} placeholder="今日公告" />
              </label>
              <label className="field-label">
                <span>公告内容</span>
                <textarea value={draft.announcementBoard.content} maxLength={800} onChange={(event) => patch({ announcementBoard: { ...draft.announcementBoard, content: event.target.value } })} placeholder="写下想让玩家看到的内容" />
              </label>
            </div>
            <div className="admin-announcement-card">
              <div className="admin-card-title">
                <strong>安全与免责声明</strong>
                <small>每天每个浏览器显示一次，建角色前也会显示；文案固定，仅可整体开关</small>
              </div>
              <Toggle
                label="开启安全声明"
                value={draft.securityDisclaimer.enabled ?? false}
                onChange={(enabled) => patch({ securityDisclaimer: { enabled } })}
              />
            </div>
            <div className="config-row">
              {Object.entries(draft.messages).map(([key, value]) => (
                <label className="field-label" key={key}>
                  <span>{key}</span>
                  <input value={value} onChange={(event) => patch({ messages: { ...draft.messages, [key]: event.target.value } })} />
                </label>
              ))}
            </div>
          </div>
          <div className="config-section admin-section-card">
            <AdminSectionHeader title="网站信息" subtitle="修改网站名称、说明和管理员口令。" />
            <div className="admin-preview-card">
              <span>预览</span>
              <strong>{draft.site.name}</strong>
              <p>{draft.site.description || "暂无网站说明"}</p>
            </div>
            <label className="field-label"><span>网站名称</span><input value={draft.site.name} onChange={(event) => patch({ site: { ...draft.site, name: event.target.value } })} placeholder="网站名称" /></label>
            <label className="field-label"><span>网站说明</span><textarea value={draft.site.description} onChange={(event) => patch({ site: { ...draft.site, description: event.target.value } })} placeholder="网站说明" /></label>
            <label className="field-label"><span>管理员口令</span><input type="password" value={draft.site.adminPassword} onChange={(event) => patch({ site: { ...draft.site, adminPassword: event.target.value } })} placeholder="管理员口令" /></label>
          </div>
        </>
      );
    }

    if (activeSection === "factions") {
      const filteredFactions = draft.genderFactions.filter((faction) => {
        const keyword = factionSearch.trim().toLowerCase();
        if (!keyword) return true;
        return `${faction.id} ${faction.label}`.toLowerCase().includes(keyword);
      });
      const factionIndex = Math.max(0, draft.genderFactions.findIndex((faction) => faction.id === activeFactionId));
      const faction = draft.genderFactions[factionIndex];
      // 性别预设按显示文字分组：同一文案可以同时勾选多个阵营（各自是独立的 GenderOption，
      // 只是 id 不同），行内的勾选框直接对应"这个文案在这个阵营下是否存在一条预设"。
      // key 用分组内第一条 GenderOption 的 id（重命名时该条目位置/id 不变），不能用 label 本身
      // 当 key——否则每敲一个字符 label 就变一次，key 跟着变，React 会把整行（含 input）连同
      // 焦点一起卸载重建，表现为"输完一个字符就失焦"。
      const genderGroups: { key: string; label: string; factionIds: Set<string> }[] = [];
      const groupIndexByLabel = new Map<string, number>();
      draft.genders.forEach((gender) => {
        let idx = groupIndexByLabel.get(gender.label);
        if (idx === undefined) {
          idx = genderGroups.length;
          groupIndexByLabel.set(gender.label, idx);
          genderGroups.push({ key: gender.id, label: gender.label, factionIds: new Set() });
        }
        genderGroups[idx].factionIds.add(gender.factionId);
      });
      // 新增一条性别选项时插入到同一 label 现有条目之后（而非数组末尾），
      // 这样某条目被取消勾选删除后，该分组在 genderGroups 里的显示顺序不会因为
      // "仅存条目排到了数组更靠后位置"而发生跳动。
      const insertGenderOption = (list: AppConfig["genders"], label: string, item: AppConfig["genders"][number]) => {
        let lastIndex = -1;
        list.forEach((g, idx) => { if (g.label === label) lastIndex = idx; });
        if (lastIndex === -1) return [...list, item];
        return [...list.slice(0, lastIndex + 1), item, ...list.slice(lastIndex + 1)];
      };
      return (
        <>
          <div className="config-section admin-section-card">
            <AdminSectionHeader title="性别预设" subtitle="系统预设性别池，玩家只能从中选择；同一显示文字可勾选多个阵营，阵营由所选性别查表决定" />
            <div className="faction-gender-list">
              {genderGroups.map((group) => {
                const label = group.label;
                return (
                  <div className="mini-card faction-gender-card" key={group.key}>
                    <div className="config-row faction-gender-row">
                      <input
                        className="faction-gender-label-input"
                        value={label}
                        onChange={(event) => {
                          const nextLabel = event.target.value;
                          patchGenders(draft.genders.map((item) => item.label === label ? { ...item, label: nextLabel } : item));
                        }}
                        placeholder="显示文字"
                      />
                      <div className="faction-gender-checkboxes">
                        {draft.genderFactions.map((f) => (
                          <label className="faction-gender-checkbox" key={f.id}>
                            <input
                              type="checkbox"
                              checked={group.factionIds.has(f.id)}
                              onChange={(event) => {
                                let nextGenders: typeof draft.genders;
                                if (event.target.checked) {
                                  const genderId = nextAdminId("gender", draft.genders.map((item) => item.id));
                                  nextGenders = insertGenderOption(draft.genders, label, { id: genderId, label, factionId: f.id });
                                } else {
                                  nextGenders = draft.genders.filter((item) => !(item.label === label && item.factionId === f.id));
                                }
                                if (nextGenders.length === 0) {
                                  onError("至少需要保留 1 个性别预设");
                                  return;
                                }
                                patchGenders(nextGenders);
                              }}
                            />
                            <span>{f.label}</span>
                          </label>
                        ))}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
            <button onClick={() => {
              let index = 1;
              while (groupIndexByLabel.has(`新性别${index}`)) index += 1;
              const label = `新性别${index}`;
              const usedIds = draft.genders.map((item) => item.id);
              const newItems = draft.genderFactions.map((f) => {
                const id = nextAdminId("gender", usedIds);
                usedIds.push(id);
                return { id, label, factionId: f.id };
              });
              patchGenders([...draft.genders, ...newItems]);
            }}>添加性别预设</button>
          </div>
          <div className="config-section admin-section-card">
            <AdminSectionHeader title="阵营" subtitle="玩家选择的阵营分组，标签颜色和任务分组在这里配置。" />
            <div className="punishment-manager faction-manager">
              <aside className="punishment-index-panel">
                <input value={factionSearch} onChange={(event) => setFactionSearch(event.target.value)} placeholder="搜索阵营" />
                <div className="punishment-index-list">
                  {filteredFactions.map((item) => (
                    <button className={item.id === faction?.id ? "active" : ""} key={item.id} onClick={() => setActiveFactionId(item.id)}>
                      <span>{item.label}</span>
                      <small>{taskGroupLabel(item.taskGroup)}</small>
                    </button>
                  ))}
                  {filteredFactions.length === 0 && <p className="empty">没有匹配的阵营</p>}
                </div>
                <button onClick={() => {
                  const factionId = nextAdminId("faction", draft.genderFactions.map((item) => item.id));
                  setActiveFactionId(factionId);
                  patchFactions([...draft.genderFactions, { id: factionId, label: "新阵营", textColor: "#4d5c6f", backgroundColor: "#eef3f8", borderColor: "#c9d6e4", taskGroup: "default" }]);
                }}>添加阵营</button>
              </aside>
              {faction && (
                <div className="mini-card punishment-detail-panel faction-editor">
                  <div className="admin-card-title">
                    <strong>{faction.label}</strong>
                    <small>{factionIndex + 1} / {draft.genderFactions.length} · {taskGroupLabel(faction.taskGroup)}</small>
                  </div>
                  <div className="admin-preview-strip compact-preview-strip">
                    <span className="faction-preview" style={factionStyle(faction)}>预览：{faction.label}</span>
                  </div>
                  <div className="config-row">
                    <label className="field-label"><span>阵营名称</span><input value={faction.label} onChange={(event) => patchFactions(draft.genderFactions.map((item, itemIndex) => itemIndex === factionIndex ? { ...item, label: event.target.value } : item))} placeholder="阵营名称" /></label>
                    <label className="field-label">
                      <span>任务分组（决定系统任务/称号取哪份文案）</span>
                      <select value={faction.taskGroup} onChange={(event) => patchFactions(draft.genderFactions.map((item, itemIndex) => itemIndex === factionIndex ? { ...item, taskGroup: event.target.value } : item))}>
                        {TASK_GROUP_OPTIONS.map((group) => <option key={group.id} value={group.id}>{group.label}</option>)}
                      </select>
                    </label>
                  </div>
                  <div className="color-grid">
                    <ColorInput label="文字颜色" value={faction.textColor} onChange={(value) => patchFactions(draft.genderFactions.map((item, itemIndex) => itemIndex === factionIndex ? { ...item, textColor: value } : item))} />
                    <ColorInput label="背景颜色" value={faction.backgroundColor} onChange={(value) => patchFactions(draft.genderFactions.map((item, itemIndex) => itemIndex === factionIndex ? { ...item, backgroundColor: value } : item))} />
                    <ColorInput label="边框颜色" value={faction.borderColor} onChange={(value) => patchFactions(draft.genderFactions.map((item, itemIndex) => itemIndex === factionIndex ? { ...item, borderColor: value } : item))} />
                  </div>
                </div>
              )}
            </div>
          </div>
        </>
      );
    }

    if (activeSection === "titles") {
      const filteredTitles = draft.titles.filter((segment) => {
        const keyword = titleSearch.trim().toLowerCase();
        if (!keyword) return true;
        return `${segment.id} ${segment.minPercent} ${segment.maxPercent} ${segment.names.join(" ")}`.toLowerCase().includes(keyword);
      });
      const selectedIndex = Math.max(0, draft.titles.findIndex((segment) => segment.id === activeTitleId));
      const segment = draft.titles[selectedIndex];
      const taskGroups = taskGroupsFromFactions(draft.genderFactions);
      // 与防多开 accessControl 相同：缺对象时用默认合并，避免 draft.rankedScore 为 undefined 时白屏。
      const rankedScore = withRankedScoreDefaults(draft.rankedScore);
      const patchRankedScore = (next: Partial<AppConfig["rankedScore"]>) =>
        patch({ rankedScore: withRankedScoreDefaults({ ...rankedScore, ...next }) });
      return (
        <>
          <div className="config-section admin-section-card">
            <AdminSectionHeader title="排位分设置" subtitle="控制排行榜/个人资料等展示时的封顶值，及每日衰减比例。数据库中的存储值无限制；" />
            <div className="config-row">
              <label className="field-label"><span>展示上限</span><input type="number" min={1} value={rankedScore.max} onChange={(event) => patchRankedScore({ max: Number(event.target.value) })} /></label>
              <label className="field-label"><span>普通玩家展示下限</span><input type="number" max={-1} value={rankedScore.min} onChange={(event) => patchRankedScore({ min: Number(event.target.value) })} /></label>
              <label className="field-label"><span>名字争夺战展示下限</span><input type="number" value={rankedScore.nameWarMin} onChange={(event) => patchRankedScore({ nameWarMin: Number(event.target.value) })} /></label>
              <label className="field-label"><span>每日衰减比例</span><input type="number" min={0.01} max={1} step={0.01} value={rankedScore.dailyDecayRatio} onChange={(event) => patchRankedScore({ dailyDecayRatio: Number(event.target.value) })} /></label>
            </div>
          </div>
          <div className="config-section admin-section-card">
            <AdminSectionHeader title="称号池" subtitle="按排位分相对展示上下限的百分比分段后，按玩家阵营所属的生理性别从对应称号池随机装备默认称号。" />
            <div className="punishment-manager title-manager">
              <aside className="punishment-index-panel">
                <input value={titleSearch} onChange={(event) => setTitleSearch(event.target.value)} placeholder="搜索段位 ID / 百分比 / 称号" />
                <div className="punishment-index-list">
                  {filteredTitles.map((item) => (
                    <button className={item.id === segment?.id ? "active" : ""} key={item.id} onClick={() => setActiveTitleId(item.id)}>
                      <span>{item.id} · {item.minPercent}% ~ {item.maxPercent}%</span>
                      <small>通用 {item.names.length} 个 · {taskGroups.length} 个分组专属池</small>
                    </button>
                  ))}
                  {filteredTitles.length === 0 && <p className="empty">没有匹配的段位</p>}
                </div>
                <button onClick={() => {
                  const nextId = nextAdminId("title", draft.titles.map((item) => item.id));
                  setActiveTitleId(nextId);
                  patch({ titles: [...draft.titles, { id: nextId, minPercent: 0, maxPercent: 0, names: ["新称号"], factionNames: Object.fromEntries(taskGroups.map((group) => [group.id, ["新称号"]])) }] });
                }}>添加段位</button>
              </aside>
              {segment && (
                <div className="mini-card punishment-detail-panel">
                  <div className="admin-card-title">
                    <strong>{segment.id} · {segment.minPercent}% ~ {segment.maxPercent}%</strong>
                    <small>{selectedIndex + 1} / {draft.titles.length} · 通用 {segment.names.length} 个 · 相对展示上下限的百分比</small>
                  </div>
                  <div className="config-row compact">
                    <label className="field-label"><span>段位 ID（自动生成，一般不用改）</span><input value={segment.id} onChange={(event) => { setActiveTitleId(event.target.value); patch({ titles: draft.titles.map((item, itemIndex) => itemIndex === selectedIndex ? { ...item, id: event.target.value } : item) }); }} /></label>
                    <label className="field-label"><span>最低百分比（-100～100）</span><input type="number" min={-100} max={100} step={0.01} value={segment.minPercent} onChange={(event) => patch({ titles: draft.titles.map((item, itemIndex) => itemIndex === selectedIndex ? { ...item, minPercent: Number(event.target.value) } : item) })} /></label>
                    <label className="field-label"><span>最高百分比（-100～100）</span><input type="number" min={-100} max={100} step={0.01} value={segment.maxPercent} onChange={(event) => patch({ titles: draft.titles.map((item, itemIndex) => itemIndex === selectedIndex ? { ...item, maxPercent: Number(event.target.value) } : item) })} /></label>
                  </div>
                  <TagListEditor
                    label="通用称号（专属为空时兜底）"
                    placeholder="输入称号后回车"
                    values={segment.names}
                    onChange={(names) => patch({ titles: draft.titles.map((item, itemIndex) => itemIndex === selectedIndex ? { ...item, names } : item) })}
                  />
                  <div className="title-faction-grid">
                    {taskGroups.map((group) => (
                      <TagListEditor
                        key={`${segment.id}-${group.id}`}
                        label={`${group.label}专属称号（${group.memberLabel}）`}
                        placeholder={`输入${group.label}称号后回车`}
                        values={segment.factionNames?.[group.id] || []}
                        onChange={(names) => patch({ titles: draft.titles.map((item, itemIndex) => itemIndex === selectedIndex ? { ...item, factionNames: { ...(item.factionNames || {}), [group.id]: names } } : item) })}
                      />
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
          <div className="config-section admin-section-card">
            <AdminSectionHeader title="称号标签配色" subtitle="称号标签的底色按该称号的赋予来源区分：系统自动分配 / 玩家自定义 / 主人为宠物设置 / 管理员后台手动设置。" />
            <div className="admin-preview-card">
              <span>预览</span>
              <div className="title-tag-style-preview">
                {titleTagStyleOrder.map((item) => {
                  const style = draft.titleTagStyles?.[item.key] || defaultRoomInfoTagStyle(item.label);
                  return <span className="title-chip" key={item.key} style={titleStyle(style)}>{style.label}</span>;
                })}
              </div>
            </div>
            <div className="room-info-tag-admin-grid">
              {titleTagStyleOrder.map((item) => {
                const style = draft.titleTagStyles?.[item.key] || defaultRoomInfoTagStyle(item.label);
                const nextStyles = draft.titleTagStyles || {};
                const update = (nextStyle: RoomInfoTagStyle) => patch({ titleTagStyles: { ...nextStyles, [item.key]: nextStyle } });
                return (
                  <div className="mini-card room-info-tag-admin-card" key={item.key}>
                    <div className="admin-card-title">
                      <strong>{item.label}</strong>
                      <small>{item.key}</small>
                    </div>
                    <span className="title-chip preview" style={titleStyle(style)}>{style.label}</span>
                    <label className="field-label"><span>显示名字</span><input value={style.label} onChange={(event) => update({ ...style, label: event.target.value })} /></label>
                    <div className="color-grid">
                      <ColorInput label="文字颜色" value={style.textColor} onChange={(textColor) => update({ ...style, textColor })} />
                      <ColorInput label="背景颜色" value={style.backgroundColor} onChange={(backgroundColor) => update({ ...style, backgroundColor })} />
                      <ColorInput label="边框颜色" value={style.borderColor} onChange={(borderColor) => update({ ...style, borderColor })} />
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </>
      );
    }

    if (activeSection === "punishments") {
      const tags = draft.punishmentTags || [];
      const rs = draft.punishmentRandomSettings || { orderStep: 2, maxDifficultyOvershoot: 5 };
      const keyword = punishmentSearch.trim().toLowerCase();
      const filteredTags = tags.filter((t) => !keyword || `${t.id} ${t.name}`.toLowerCase().includes(keyword));
      const filteredTasks = taskPoolDraft.filter((t) => !keyword || `${t.id} ${t.name} ${t.text}`.toLowerCase().includes(keyword));
      const filteredSeries = seriesDraft.filter((s) => !keyword || `${s.id} ${s.name}`.toLowerCase().includes(keyword));
      const tagIndex = Math.max(0, tags.findIndex((t) => t.id === activeTagId));
      const taskIndex = Math.max(0, taskPoolDraft.findIndex((t) => t.id === activeTaskId));
      const seriesIndex = Math.max(0, seriesDraft.findIndex((s) => s.id === activeSeriesId));
      const activeTag = tags[tagIndex];
      const activeTask = taskPoolDraft[taskIndex];
      const activeSeries = seriesDraft[seriesIndex];
      const tagName = (id: string) => {
        if (id.startsWith("series:")) return id;
        return tags.find((t) => t.id === id)?.name || id;
      };
      const taskLabel = (id: string) => {
        const t = taskPoolDraft.find((x) => x.id === id);
        if (!t) return `${id}（已删除）`;
        return t.name || t.id;
      };
      return (
        <div className="config-section admin-section-card">
          <AdminSectionHeader title="惩罚任务" subtitle="系列任务和任务池需独立保存，任务文案可用 {loser}/{winner} 替代败者与胜者昵称。" />
          <div className="admin-subtab-row">
            <button type="button" className={punishmentTab === "config" ? "active" : ""} onClick={() => setPunishmentTab("config")}>惩罚配置</button>
            <button type="button" className={punishmentTab === "series" ? "active" : ""} onClick={() => { setPunishmentTab("series"); void ensureSeriesLoaded(); }}>系列任务</button>
            <button type="button" className={punishmentTab === "pool" ? "active" : ""} onClick={() => { setPunishmentTab("pool"); void ensureTaskPoolLoaded(); }}>任务池</button>
          </div>
          <input value={punishmentSearch} onChange={(event) => setPunishmentSearch(event.target.value)} placeholder="搜索名称 / ID / 文案" style={{ marginBottom: 12 }} />

          {punishmentTab === "config" && (
            <>
              <div className="punishment-manager">
                <aside className="punishment-index-panel">
                  <div className="punishment-index-list">
                    {filteredTags.map((item) => (
                      <button className={item.id === activeTag?.id ? "active" : ""} key={item.id} onClick={() => setActiveTagId(item.id)}>
                        <span>{item.name}</span>
                      </button>
                    ))}
                    {filteredTags.length === 0 && <p className="empty">没有匹配的标签</p>}
                  </div>
                  <button onClick={() => {
                    const nextId = nextAdminId("tag", tags.map((t) => t.id));
                    setActiveTagId(nextId);
                    patch({ punishmentTags: [...tags, { id: nextId, name: "新标签", roomBackgroundImages: [], roomNamePool: defaultAdminRoomNamePool() }] });
                  }}>添加标签</button>
                </aside>
                {activeTag && (
                  <div className="mini-card punishment-detail-panel">
                    <div className="admin-card-title">
                      <strong>{activeTag.name}</strong>
                      <small>{tagIndex + 1} / {tags.length} · 示例：{sampleRoomName(activeTag.roomNamePool)}</small>
                    </div>
                    <div className="config-row">
                      <label className="field-label"><span>任务标签</span><input value={activeTag.name} onChange={(event) => patch({ punishmentTags: tags.map((t, i) => i === tagIndex ? { ...t, name: event.target.value } : t) })} /></label>
                      <div className="field-label field-label-action">
                        <span>&nbsp;</span>
                        <button type="button" className="danger-button" onClick={() => {
                          if (tags.length <= 1) { onError("至少需要保留 1 个标签"); return; }
                          if (!window.confirm(`确定删除标签「${activeTag.name}」吗？保存配置后，任务池里带该标签的任务会自动摘除该标签。`)) return;
                          const nextTags = tags.filter((_, i) => i !== tagIndex);
                          setActiveTagId(nextTags[Math.max(0, tagIndex - 1)]?.id || nextTags[0]?.id || "");
                          patch({ punishmentTags: nextTags });
                        }}>删除这个标签</button>
                      </div>
                    </div>
                    <AdminBackgroundImageField label="房间信息卡图库" values={activeTag.roomBackgroundImages || []} upload={uploadAdminImage} onError={onError} onChange={(roomBackgroundImages) => patch({ punishmentTags: tags.map((t, i) => i === tagIndex ? { ...t, roomBackgroundImages } : t) })} />
                    <RoomNamePoolEditor title="随机房名词库" pool={activeTag.roomNamePool || emptyRoomNamePool()} onChange={(roomNamePool) => patch({ punishmentTags: tags.map((t, i) => i === tagIndex ? { ...t, roomNamePool } : t) })} />
                  </div>
                )}
              </div>
              <div className="mini-card player-punishment-room-name-card">
                <div className="admin-card-title">
                  <strong>玩家发布任务随机房名词库</strong>
                  <small>示例：{sampleRoomName(draft.playerPunishmentRoomNamePool)}</small>
                </div>
                <RoomNamePoolEditor title="使用玩家自定义任务时所生成的房间名" pool={draft.playerPunishmentRoomNamePool || defaultAdminRoomNamePool()} onChange={(playerPunishmentRoomNamePool) => patch({ playerPunishmentRoomNamePool })} />
              </div>
              <div className="mini-card">
                <div className="admin-card-title"><strong>随机任务全局难度</strong></div>
                <div className="config-row">
                  <label className="field-label">
                    <span>难度推进步长（默认 2）</span>
                    <input type="number" min={0} step={1} value={rs.orderStep ?? 2} onChange={(event) => patch({ punishmentRandomSettings: { ...rs, orderStep: Number(event.target.value) } })} />
                  </label>
                  <label className="field-label">
                    <span>难度上浮硬顶（默认 5）</span>
                    <input type="number" min={0} step={1} value={rs.maxDifficultyOvershoot ?? 5} onChange={(event) => patch({ punishmentRandomSettings: { ...rs, maxDifficultyOvershoot: Number(event.target.value) } })} />
                  </label>
                </div>
              </div>
            </>
          )}

          {punishmentTab === "pool" && (
            <>
              {!taskPoolLoaded && <p className="hint">正在加载任务池…</p>}
              {taskPoolLoaded && (
                <>
                  <div className="punishment-manager">
                    <aside className="punishment-index-panel">
                      <div className="punishment-index-list">
                        {filteredTasks.map((item) => {
                          const noTags = !(item.tagIds || []).length;
                          const excludedByOrder = item.order === -1;
                          return (
                            <button className={item.id === activeTask?.id ? "active" : ""} key={item.id} onClick={() => setActiveTaskId(item.id)}>
                              <span>{item.name || item.id}</span>
                              <small>
                                难度 {item.order}{excludedByOrder ? "（不参与随机）" : ""} · {(item.tagIds || []).map(tagName).join("、") || (noTags ? "无标签，不会被抽到" : "")}
                              </small>
                            </button>
                          );
                        })}
                        {filteredTasks.length === 0 && <p className="empty">没有匹配的任务</p>}
                      </div>
                      <button onClick={() => {
                        const nextId = nextAdminId("task", taskPoolDraft.map((t) => t.id));
                        setActiveTaskId(nextId);
                        patchTaskPool([...taskPoolDraft, newFlatPunishmentTask(nextId, tags[0]?.id)]);
                      }}>添加任务</button>
                    </aside>
                    {activeTask && (
                      <div className="mini-card punishment-detail-panel">
                        <div className="admin-card-title">
                          <strong>{activeTask.name || activeTask.id}</strong>
                          <small>{taskIndex + 1} / {taskPoolDraft.length} · {factionSummaryLabel(activeTask.factionIds || [], draft.genderFactions)}</small>
                        </div>
                        {!(activeTask.tagIds || []).length && <p className="hint">ℹ 这条任务未勾选标签，随机模式下不会被抽到（仍可被系列任务按 ID 引用）。</p>}
                        {activeTask.order === -1 && <p className="hint">ℹ 这条任务难度为 -1，随机模式下不会被抽到（仍可被系列任务按 ID 引用）。</p>}
                        <div className="config-row">
                          <label className="field-label"><span>任务名称</span><input value={activeTask.name} onChange={(event) => patchTaskPool(taskPoolDraft.map((t, i) => i === taskIndex ? { ...activeTask, name: event.target.value } : t))} /></label>
                          <div className="field-label field-label-action">
                            <span>&nbsp;</span>
                            <button type="button" className="danger-button" onClick={() => {
                              if (taskPoolDraft.length <= 1) { onError("至少需要保留 1 条任务"); return; }
                              if (!window.confirm(`确定删除任务「${activeTask.name || activeTask.id}」吗？已被系列引用的步骤运行时会跳过。`)) return;
                              const nextTasks = taskPoolDraft.filter((_, i) => i !== taskIndex);
                              setActiveTaskId(nextTasks[Math.max(0, taskIndex - 1)]?.id || nextTasks[0]?.id || "");
                              patchTaskPool(nextTasks);
                            }}>删除这条任务</button>
                          </div>
                        </div>
                        <label className="field-label">
                          <span>任务文案（可用 {"{loser}"}/{"{winner}"}）</span>
                          <textarea value={activeTask.text} onChange={(event) => patchTaskPool(taskPoolDraft.map((t, i) => i === taskIndex ? { ...activeTask, text: event.target.value } : t))} />
                        </label>
                        <div className="config-row">
                          <label className="field-label"><span>任务难度（-1~99）</span><input type="number" min={-1} max={99} value={activeTask.order} onChange={(event) => patchTaskPool(taskPoolDraft.map((t, i) => i === taskIndex ? { ...activeTask, order: Number(event.target.value) } : t))} /></label>
                          <label className="field-label"><span>背景透明率</span><input type="number" min={0} max={1} step={0.01} value={activeTask.backgroundOpacity ?? 0.22} onChange={(event) => patchTaskPool(taskPoolDraft.map((t, i) => i === taskIndex ? { ...activeTask, backgroundOpacity: Number(event.target.value) } : t))} /></label>
                        </div>
                        <div className="field-label">
                          <span>所属标签</span>
                          <div className="faction-checkbox-grid">
                            {tags.map((tag) => {
                              const checked = (activeTask.tagIds || []).includes(tag.id);
                              return (
                                <label key={tag.id} className="faction-checkbox">
                                  <input type="checkbox" checked={checked} onChange={(event) => {
                                    const next = event.target.checked
                                      ? [...(activeTask.tagIds || []), tag.id]
                                      : (activeTask.tagIds || []).filter((id) => id !== tag.id);
                                    patchTaskPool(taskPoolDraft.map((t, i) => i === taskIndex ? { ...activeTask, tagIds: next } : t));
                                  }} />
                                  {tag.name}
                                </label>
                              );
                            })}
                          </div>
                          {(activeTask.tagIds || []).some((id) => id.startsWith("series:")) && (
                            <p className="hint">含内部标签 {(activeTask.tagIds || []).filter((id) => id.startsWith("series:")).join("、")}（系列迁移产生，可保留或改勾真实标签）。</p>
                          )}
                        </div>
                        <div className="field-label">
                          <span>适用阵营</span>
                          <div className="faction-checkbox-grid">
                            {draft.genderFactions.map((faction) => {
                              const checked = (activeTask.factionIds || []).includes(faction.id);
                              return (
                                <label key={faction.id} className="faction-checkbox">
                                  <input type="checkbox" checked={checked} onChange={(event) => {
                                    const next = event.target.checked
                                      ? [...(activeTask.factionIds || []), faction.id]
                                      : (activeTask.factionIds || []).filter((id) => id !== faction.id);
                                    patchTaskPool(taskPoolDraft.map((t, i) => i === taskIndex ? { ...activeTask, factionIds: next } : t));
                                  }} />
                                  {faction.label}
                                </label>
                              );
                            })}
                          </div>
                          {!(activeTask.factionIds || []).length && <p className="hint">ℹ 这条任务未勾选阵营，不会被抽到（随机模式候选池和系列任务候选都不会用到它）。</p>}
                        </div>
                        <AdminBackgroundImageField label="任务背景图库" values={activeTask.backgroundImages || []} upload={uploadAdminImage} onError={onError} onChange={(backgroundImages) => patchTaskPool(taskPoolDraft.map((t, i) => i === taskIndex ? { ...activeTask, backgroundImages } : t))} />
                      </div>
                    )}
                  </div>
                  <div className="admin-subtab-save-row">
                    <button type="button" className="primary" disabled={!taskPoolDirty} onClick={() => void saveTaskPool()}>
                      <Save size={16} /> 保存任务池{taskPoolDirty ? " *" : ""}
                    </button>
                    <small className="hint">任务池独立保存，不影响下方配置保存。</small>
                  </div>
                </>
              )}
            </>
          )}

          {punishmentTab === "series" && (
            <>
              {!seriesLoaded && <p className="hint">正在加载系列任务…</p>}
              {seriesLoaded && (
                <>
                  {taskPoolDraft.length === 0 && <p className="hint" style={{ color: "var(--reject, #c45)" }}>请先在任务池里创建任务，再编排系列步骤。</p>}
                  <div className="punishment-manager">
                    <aside className="punishment-index-panel">
                      <div className="punishment-index-list">
                        {filteredSeries.map((item) => {
                          const usable = seriesIsUsable(item, taskPoolDraft, draft.genderFactions);
                          return (
                            <button className={item.id === activeSeries?.id ? "active" : ""} key={item.id} onClick={() => setActiveSeriesId(item.id)}>
                              <span className={usable ? undefined : "danger-hint"}>{item.name}{usable ? "" : " ⚠"}</span>
                              <small>{(item.steps || []).length} 步{usable ? "" : " · 阵营覆盖不全，不会生效"}</small>
                            </button>
                          );
                        })}
                        {filteredSeries.length === 0 && <p className="empty">没有系列任务（可为空）</p>}
                      </div>
                      <button onClick={() => {
                        const nextId = nextAdminId("series", seriesDraft.map((s) => s.id));
                        setActiveSeriesId(nextId);
                        patchSeriesDraft([...seriesDraft, newSeriesTask(nextId)]);
                      }}>添加系列任务</button>
                    </aside>
                    {activeSeries && (
                      <div className="mini-card punishment-detail-panel">
                        <div className="admin-card-title">
                          <strong>{activeSeries.name}</strong>
                          <small>{seriesIndex + 1} / {seriesDraft.length} · {(activeSeries.steps || []).length} 步 · 示例：{sampleRoomName(activeSeries.roomNamePool)}</small>
                        </div>
                        <div className="admin-danger-row">
                          <button type="button" className="danger-button" onClick={() => {
                            if (!window.confirm(`确定删除系列「${activeSeries.name}」吗？`)) return;
                            const next = seriesDraft.filter((_, i) => i !== seriesIndex);
                            setActiveSeriesId(next[Math.max(0, seriesIndex - 1)]?.id || next[0]?.id || "");
                            patchSeriesDraft(next);
                          }}>删除这个系列</button>
                          <button type="button" className="danger-soft" onClick={async () => {
                            if (!window.confirm(`重置所有人在系列「${activeSeries.name}」上的进度？`)) return;
                            try {
                              await ask("admin:action", { action: "resetPunishmentSeriesProgress", seriesId: activeSeries.id });
                              onError("已重置该系列的所有人进度。");
                            } catch (error) {
                              onError(error instanceof Error ? error.message : "重置失败");
                            }
                          }}>重置所有人进度</button>
                        </div>
                        <label className="field-label"><span>玩家可见名称</span><input value={activeSeries.name} onChange={(event) => patchSeriesDraft(seriesDraft.map((s, i) => i === seriesIndex ? { ...activeSeries, name: event.target.value } : s))} /></label>
                        <AdminBackgroundImageField label="房间信息卡图库" values={activeSeries.roomBackgroundImages || []} upload={uploadAdminImage} onError={onError} onChange={(roomBackgroundImages) => patchSeriesDraft(seriesDraft.map((s, i) => i === seriesIndex ? { ...activeSeries, roomBackgroundImages } : s))} />
                        <RoomNamePoolEditor title="随机房名词库" pool={activeSeries.roomNamePool || emptyRoomNamePool()} onChange={(roomNamePool) => patchSeriesDraft(seriesDraft.map((s, i) => i === seriesIndex ? { ...activeSeries, roomNamePool } : s))} />
                        <div className="punishment-task-list">
                          {(activeSeries.steps || []).map((step, subIndex) => {
                            const pickerOpen = taskPickerStep?.seriesId === activeSeries.id && taskPickerStep.stepIndex === subIndex;
                            const missing = seriesStepMissingFactionLabels(step.taskIds || [], taskPoolDraft, draft.genderFactions);
                            return (
                              <details className="mini-card punishment-task-editor" key={subIndex} open={subIndex === 0}>
                                <summary>
                                  <span className="punishment-step-summary-title">
                                    <strong>第 {subIndex + 1} 步</strong>
                                    <small className={missing.length > 0 ? "danger-hint" : ""} title={missing.length > 0 ? `没有覆盖到：${missing.join("、")}——命中这些阵营的玩家跑到这一步会被判定失效，整个系列都不会生效。` : undefined}>
                                      {(step.taskIds || []).length} 个候选{missing.length > 0 ? " ⚠" : ""}
                                    </small>
                                  </span>
                                  <span className="punishment-step-summary-actions">
                                    <button type="button" className="tiny-danger-button" disabled={subIndex === 0} onClick={(e) => {
                                      e.preventDefault(); e.stopPropagation();
                                      const steps = [...(activeSeries.steps || [])];
                                      const tmp = steps[subIndex]; steps[subIndex] = steps[subIndex - 1]; steps[subIndex - 1] = tmp;
                                      patchSeriesDraft(seriesDraft.map((s, i) => i === seriesIndex ? { ...activeSeries, steps } : s));
                                    }}>上移</button>
                                    <button type="button" className="tiny-danger-button" disabled={subIndex >= (activeSeries.steps || []).length - 1} onClick={(e) => {
                                      e.preventDefault(); e.stopPropagation();
                                      const steps = [...(activeSeries.steps || [])];
                                      const tmp = steps[subIndex]; steps[subIndex] = steps[subIndex + 1]; steps[subIndex + 1] = tmp;
                                      patchSeriesDraft(seriesDraft.map((s, i) => i === seriesIndex ? { ...activeSeries, steps } : s));
                                    }}>下移</button>
                                    <button type="button" className="tiny-danger-button" onClick={(e) => {
                                      e.preventDefault(); e.stopPropagation();
                                      if (pickerOpen) {
                                        setTaskPickerStep(null);
                                      } else {
                                        setTaskPickerStep({ seriesId: activeSeries.id, stepIndex: subIndex });
                                        setTaskPickerQuery("");
                                      }
                                    }}>{pickerOpen ? "收起" : "添加"}</button>
                                    <button type="button" className="danger-button tiny-danger-button" onClick={(e) => {
                                      e.preventDefault(); e.stopPropagation();
                                      if ((activeSeries.steps || []).length <= 1) { onError("至少保留一步"); return; }
                                      const steps = (activeSeries.steps || []).filter((_, i) => i !== subIndex);
                                      patchSeriesDraft(seriesDraft.map((s, i) => i === seriesIndex ? { ...activeSeries, steps } : s));
                                    }}>删除</button>
                                  </span>
                                </summary>
                                <div className="task-picker-selected">
                                  {(step.taskIds || []).map((tid, ti) => (
                                    <span className="task-picker-chip" key={`${tid}-${ti}`}>
                                      <span>{taskLabel(tid)}</span>
                                      <span className="task-picker-chip-actions">
                                        <button type="button" className="tiny-danger-button" disabled={ti === 0} onClick={() => {
                                          const ids = [...(step.taskIds || [])];
                                          const tmp = ids[ti]; ids[ti] = ids[ti - 1]; ids[ti - 1] = tmp;
                                          const steps = (activeSeries.steps || []).map((st, i) => i === subIndex ? { taskIds: ids } : st);
                                          patchSeriesDraft(seriesDraft.map((s, i) => i === seriesIndex ? { ...activeSeries, steps } : s));
                                        }}>上移</button>
                                        <button type="button" className="tiny-danger-button" disabled={ti >= (step.taskIds || []).length - 1} onClick={() => {
                                          const ids = [...(step.taskIds || [])];
                                          const tmp = ids[ti]; ids[ti] = ids[ti + 1]; ids[ti + 1] = tmp;
                                          const steps = (activeSeries.steps || []).map((st, i) => i === subIndex ? { taskIds: ids } : st);
                                          patchSeriesDraft(seriesDraft.map((s, i) => i === seriesIndex ? { ...activeSeries, steps } : s));
                                        }}>下移</button>
                                        <button type="button" className="danger-button tiny-danger-button" onClick={() => {
                                          if ((step.taskIds || []).length <= 1) { onError("每步至少 1 个任务"); return; }
                                          const ids = (step.taskIds || []).filter((_, i) => i !== ti);
                                          const steps = (activeSeries.steps || []).map((st, i) => i === subIndex ? { taskIds: ids } : st);
                                          patchSeriesDraft(seriesDraft.map((s, i) => i === seriesIndex ? { ...activeSeries, steps } : s));
                                        }}>删除</button>
                                      </span>
                                    </span>
                                  ))}
                                  {(step.taskIds || []).length === 0 && <p className="hint">还没选任务</p>}
                                </div>
                                {pickerOpen && (
                                  <div className="task-picker-panel">
                                    <input value={taskPickerQuery} onChange={(e) => setTaskPickerQuery(e.target.value)} placeholder="按标签 / 阵营 / 文案搜索" />
                                    <div className="task-picker-list">
                                      {taskPoolDraft
                                        .filter((t) => {
                                          const q = taskPickerQuery.trim().toLowerCase();
                                          if (!q) return true;
                                          const tagStr = (t.tagIds || []).map(tagName).join(" ");
                                          const facStr = (t.factionIds || []).map((id) => draft.genderFactions.find((f) => f.id === id)?.label || id).join(" ");
                                          return `${t.id} ${t.name} ${t.text} ${tagStr} ${facStr}`.toLowerCase().includes(q);
                                        })
                                        .map((t) => (
                                          <button type="button" key={t.id} className="task-picker-item" onClick={() => {
                                            const ids = [...(step.taskIds || []), t.id];
                                            const steps = (activeSeries.steps || []).map((st, i) => i === subIndex ? { taskIds: ids } : st);
                                            patchSeriesDraft(seriesDraft.map((s, i) => i === seriesIndex ? { ...activeSeries, steps } : s));
                                          }}>
                                            <strong>{t.name || t.id}</strong>
                                            <small>{(t.tagIds || []).map(tagName).join("、") || "无标签"} · {factionSummaryLabel(t.factionIds || [], draft.genderFactions)}</small>
                                            <span className="hint">{t.text}</span>
                                          </button>
                                        ))}
                                      {taskPoolDraft.length === 0 && <p className="empty">请先在任务池里创建任务</p>}
                                    </div>
                                  </div>
                                )}
                              </details>
                            );
                          })}
                        </div>
                        <button type="button" onClick={() => {
                          const firstTask = taskPoolDraft[0]?.id;
                          const steps = [...(activeSeries.steps || []), { taskIds: firstTask ? [firstTask] : [] }];
                          patchSeriesDraft(seriesDraft.map((s, i) => i === seriesIndex ? { ...activeSeries, steps } : s));
                        }}>添加子任务步骤</button>
                      </div>
                    )}
                  </div>
                  <div className="admin-subtab-save-row">
                    <button type="button" className="primary" disabled={!seriesDirty} onClick={() => void saveSeriesPool()}>
                      <Save size={16} /> 保存系列任务
                    </button>
                    <small className="hint">系列任务独立保存，不影响下方配置保存。</small>
                  </div>
                </>
              )}
            </>
          )}
        </div>
      );
    }

    if (activeSection === "nameWar") {
      const preview = `${draft.nameWar.penaltyPrefix || "失名者"}-A7K2`;
      const extreme = draft.extremeMode;
      const patchExtreme = (nextExtreme: AppConfig["extremeMode"]) => patch({ extremeMode: nextExtreme });
      return (
        <>
          <div className="config-section admin-section-card">
            <AdminSectionHeader title="名字争夺战" subtitle="设置惩罚名前缀、通用改名处标题和退出高难度后的称号。" />
            <div className="admin-preview-card">
              <span>预览</span>
              <strong>{preview}</strong>
              <p>{draft.nameWar.renamePanelTitle || draft.nameWar.loserPanelTitle || "通用改名处"} · {draft.nameWar.nameWarLoserLabel || "名争失格"} / {draft.nameWar.extremeForceClosedLabel || "极限强关"} · 退出高难度称号：{draft.nameWar.escapeTitle || "逃跑的人"}</p>
            </div>
            <div className="config-row">
              <label className="field-label">
                <span>惩罚名前缀 XXXX</span>
                <input value={draft.nameWar.penaltyPrefix} maxLength={16} onChange={(event) => patch({ nameWar: { ...draft.nameWar, penaltyPrefix: event.target.value } })} placeholder="例如：失名者" />
              </label>
              <label className="field-label">
                <span>旧失格者面板标题</span>
                <input value={draft.nameWar.loserPanelTitle} maxLength={24} onChange={(event) => patch({ nameWar: { ...draft.nameWar, loserPanelTitle: event.target.value } })} placeholder="名字争夺战失格者" />
              </label>
              <label className="field-label">
                <span>通用改名处标题</span>
                <input value={draft.nameWar.renamePanelTitle || ""} maxLength={24} onChange={(event) => patch({ nameWar: { ...draft.nameWar, renamePanelTitle: event.target.value } })} placeholder="通用改名处" />
              </label>
              <label className="field-label">
                <span>名争来源标签</span>
                <input value={draft.nameWar.nameWarLoserLabel || ""} maxLength={16} onChange={(event) => patch({ nameWar: { ...draft.nameWar, nameWarLoserLabel: event.target.value } })} placeholder="名争失格" />
              </label>
              <label className="field-label">
                <span>极限强关标签</span>
                <input value={draft.nameWar.extremeForceClosedLabel || ""} maxLength={16} onChange={(event) => patch({ nameWar: { ...draft.nameWar, extremeForceClosedLabel: event.target.value } })} placeholder="极限强关" />
              </label>
              <label className="field-label">
                <span>退出高难度称号</span>
                <input value={draft.nameWar.escapeTitle} maxLength={18} onChange={(event) => patch({ nameWar: { ...draft.nameWar, escapeTitle: event.target.value } })} placeholder="逃跑的人" />
              </label>
              <label className="field-label">
                <span>失格分阈值（真实分）</span>
                <input type="number" max={-1} value={draft.nameWar.penaltyThreshold ?? DEFAULT_NAME_WAR_PENALTY_THRESHOLD} onChange={(event) => patch({ nameWar: { ...draft.nameWar, penaltyThreshold: Number(event.target.value) } })} placeholder={String(DEFAULT_NAME_WAR_PENALTY_THRESHOLD)} />
              </label>
              <label className="field-label">
                <span>改名最低分（真实分）</span>
                <input type="number" min={1} value={draft.nameWar.renameMinPoints ?? DEFAULT_NAME_WAR_RENAME_MIN_POINTS} onChange={(event) => patch({ nameWar: { ...draft.nameWar, renameMinPoints: Number(event.target.value) } })} placeholder={String(DEFAULT_NAME_WAR_RENAME_MIN_POINTS)} />
              </label>
            </div>
            <p className="hint">随机码固定为 4 位大写字母/数字；已有惩罚名不会因为你改前缀立刻变化，新触发的玩家会使用新前缀。失格线、改名最低分均按数据库真实排位分判定，与展示封顶无关。</p>
          </div>
          <div className="config-section admin-section-card">
            <AdminSectionHeader title="极限模式" subtitle="修改极限模式名称、标志、折扣、整点扣分和连胜风险。" />
            <div className="admin-preview-card">
              <span>预览</span>
              <strong>{extreme.emoji} {extreme.label}</strong>
              <p>关闭后冷却 {extreme.cooldownHours} 小时；{extreme.winStreakThreshold} 连胜后 {Math.round((extreme.winStreakCrashChance ?? 0) * 100)}% 额外扣 {extreme.crashTargetPoints} 分。</p>
            </div>
            <div className="config-row">
              <label className="field-label"><span>显示名称</span><input value={extreme.label} maxLength={16} onChange={(event) => patchExtreme({ ...extreme, label: event.target.value })} /></label>
              <label className="field-label"><span>标志 Emoji</span><input value={extreme.emoji} maxLength={4} onChange={(event) => patchExtreme({ ...extreme, emoji: event.target.value })} /></label>
              <label className="field-label"><span>关闭后冷却小时</span><input type="number" min={1} max={168} value={extreme.cooldownHours} onChange={(event) => patchExtreme({ ...extreme, cooldownHours: Number(event.target.value) })} /></label>
              <label className="field-label"><span>连胜阈值</span><input type="number" min={1} max={100} value={extreme.winStreakThreshold} onChange={(event) => patchExtreme({ ...extreme, winStreakThreshold: Number(event.target.value) })} /></label>
              <label className="field-label"><span>连胜风险概率 0-1</span><input type="number" min={0} max={1} step={0.01} value={extreme.winStreakCrashChance ?? 0} onChange={(event) => patchExtreme({ ...extreme, winStreakCrashChance: Number(event.target.value) })} /></label>
              <label className="field-label"><span>连胜风险扣分</span><input type="number" min={1} max={1999} value={extreme.crashTargetPoints} onChange={(event) => patchExtreme({ ...extreme, crashTargetPoints: Number(event.target.value) })} /></label>
              <label className="field-label"><span>强关改名最低分</span><input type="number" min={1} max={999} value={extreme.forceRenameMinPoints || 1} onChange={(event) => patchExtreme({ ...extreme, forceRenameMinPoints: Number(event.target.value) })} /></label>
              <label className="field-label"><span>强关保护小时</span><input type="number" min={1} max={168} value={extreme.forceRenameProtectHours || 4} onChange={(event) => patchExtreme({ ...extreme, forceRenameProtectHours: Number(event.target.value) })} /></label>
            </div>
            <label className="field-label">
              <span>强行关闭提示</span>
              <textarea value={extreme.forceCloseWarning || ""} maxLength={180} onChange={(event) => patchExtreme({ ...extreme, forceCloseWarning: event.target.value })} placeholder="强行关闭极限模式后..." />
            </label>
            <div className="admin-card">
              <div className="admin-card-title">
                <strong>正分输分比例</strong>
                <small>0.9 表示只扣 90%</small>
              </div>
              <div className="config-row">
                {(["pos1", "pos2", "pos3", "pos4"] as const).map((key) => (
                  <label className="field-label" key={key}><span>{key}</span><input type="number" min={0} max={1} step={0.01} value={extreme.positiveLossRates[key]} onChange={(event) => patchExtreme({ ...extreme, positiveLossRates: { ...extreme.positiveLossRates, [key]: Number(event.target.value) } })} /></label>
                ))}
              </div>
            </div>
            <div className="admin-card">
              <div className="admin-card-title">
                <strong>负分赢分比例</strong>
                <small>最负分段按 neg4</small>
              </div>
              <div className="config-row">
                {(["neg1", "neg2", "neg3", "neg4"] as const).map((key) => (
                  <label className="field-label" key={key}><span>{key}</span><input type="number" min={0} max={1} step={0.01} value={extreme.negativeWinRates[key]} onChange={(event) => patchExtreme({ ...extreme, negativeWinRates: { ...extreme.negativeWinRates, [key]: Number(event.target.value) } })} /></label>
                ))}
              </div>
            </div>
            <div className="admin-card">
              <div className="admin-card-title">
                <strong>整点扣分</strong>
                <small>default 用于 0 分及负分</small>
              </div>
              <div className="config-row">
                {(["pos4", "pos3", "pos2", "pos1", "default"] as const).map((key) => (
                  <label className="field-label" key={key}><span>{key}</span><input type="number" min={0} max={999} value={extreme.hourlyDecay[key]} onChange={(event) => patchExtreme({ ...extreme, hourlyDecay: { ...extreme.hourlyDecay, [key]: Number(event.target.value) } })} /></label>
                ))}
              </div>
            </div>
          </div>
        </>
      );
    }

    if (activeSection === "giveaway") {
      return (
        <div className="config-section admin-section-card">
          <AdminSectionHeader title="白给模式" subtitle="修改大厅白给自救板的标题、说明和输入提示。" />
          <div className="admin-preview-card">
            <span>预览</span>
            <strong>{draft.giveaway.panelTitle}</strong>
            <p>{draft.giveaway.panelDescription}</p>
          </div>
          <div className="config-row">
            <label className="field-label">
              <span>大厅面板标题</span>
              <input value={draft.giveaway.panelTitle} maxLength={24} onChange={(event) => patch({ giveaway: { ...draft.giveaway, panelTitle: event.target.value } })} placeholder="白给自救板" />
            </label>
            <label className="field-label">
              <span>提交框提示</span>
              <input value={draft.giveaway.submitPlaceholder} maxLength={60} onChange={(event) => patch({ giveaway: { ...draft.giveaway, submitPlaceholder: event.target.value } })} placeholder="写下你的自我惩罚宣言..." />
            </label>
          </div>
          <label className="field-label">
            <span>面板说明</span>
            <textarea value={draft.giveaway.panelDescription} maxLength={160} onChange={(event) => patch({ giveaway: { ...draft.giveaway, panelDescription: event.target.value } })} placeholder="提交一点自我惩罚宣言..." />
          </label>
          <label className="field-label">
            <span>空状态文案</span>
            <input value={draft.giveaway.emptyText} maxLength={60} onChange={(event) => patch({ giveaway: { ...draft.giveaway, emptyText: event.target.value } })} placeholder="还没有人在白给自救板上。" />
          </label>
          <div className="config-row">
            <label className="field-label">
              <span>主动白给增量 (%)</span>
              <input type="number" min={0.1} max={100} step={0.1} value={draft.giveaway.activeBoostValue} onChange={(event) => patch({ giveaway: { ...draft.giveaway, activeBoostValue: Number(event.target.value) } })} />
            </label>
            <label className="field-label">
              <span>胜利扣减白给值 (%)</span>
              <input type="number" min={0.1} max={100} step={0.1} value={draft.giveaway.winPenaltyValue} onChange={(event) => patch({ giveaway: { ...draft.giveaway, winPenaltyValue: Number(event.target.value) } })} />
            </label>
          </div>
          <p className="hint">以下三档（普通用户 / 对自己宠物 / 对自己主人）的每小时次数上限、每次点赞降低值、每次倒赞增加值均可分别设置，按「投票者与被投票者的认主认宠关系」（只认直系，不含宠物的宠物等二级以上关系）挑选对应档位，且对每个上板玩家独立计时/计次（不是投票者的总量）。</p>
          <AdminSectionHeader title="普通用户" subtitle="投票者与被投票者之间没有认主认宠关系时使用。" />
          <div className="config-row">
            <label className="field-label">
              <span>点赞每小时次数上限</span>
              <input type="number" min={1} step={1} value={draft.giveaway.likeVoteLimitPerHour} onChange={(event) => patch({ giveaway: { ...draft.giveaway, likeVoteLimitPerHour: Number(event.target.value) } })} />
            </label>
            <label className="field-label">
              <span>点赞降低值 (%)</span>
              <input type="number" min={0.01} max={100} step={0.01} value={draft.giveaway.likeVoteValue} onChange={(event) => patch({ giveaway: { ...draft.giveaway, likeVoteValue: Number(event.target.value) } })} />
            </label>
          </div>
          <div className="config-row">
            <label className="field-label">
              <span>倒赞每小时次数上限</span>
              <input type="number" min={1} step={1} value={draft.giveaway.dislikeVoteLimitPerHour} onChange={(event) => patch({ giveaway: { ...draft.giveaway, dislikeVoteLimitPerHour: Number(event.target.value) } })} />
            </label>
            <label className="field-label">
              <span>倒赞增加值 (%)</span>
              <input type="number" min={0.01} max={100} step={0.01} value={draft.giveaway.dislikeVoteValue} onChange={(event) => patch({ giveaway: { ...draft.giveaway, dislikeVoteValue: Number(event.target.value) } })} />
            </label>
          </div>
          <AdminSectionHeader title="对自己宠物" subtitle="投票者是被投票者的直系主人时使用。" />
          <div className="config-row">
            <label className="field-label">
              <span>点赞每小时次数上限</span>
              <input type="number" min={1} step={1} value={draft.giveaway.petLikeVoteLimitPerHour} onChange={(event) => patch({ giveaway: { ...draft.giveaway, petLikeVoteLimitPerHour: Number(event.target.value) } })} />
            </label>
            <label className="field-label">
              <span>点赞降低值 (%)</span>
              <input type="number" min={0.01} max={100} step={0.01} value={draft.giveaway.petLikeVoteValue} onChange={(event) => patch({ giveaway: { ...draft.giveaway, petLikeVoteValue: Number(event.target.value) } })} />
            </label>
          </div>
          <div className="config-row">
            <label className="field-label">
              <span>倒赞每小时次数上限</span>
              <input type="number" min={1} step={1} value={draft.giveaway.petDislikeVoteLimitPerHour} onChange={(event) => patch({ giveaway: { ...draft.giveaway, petDislikeVoteLimitPerHour: Number(event.target.value) } })} />
            </label>
            <label className="field-label">
              <span>倒赞增加值 (%)</span>
              <input type="number" min={0.01} max={100} step={0.01} value={draft.giveaway.petDislikeVoteValue} onChange={(event) => patch({ giveaway: { ...draft.giveaway, petDislikeVoteValue: Number(event.target.value) } })} />
            </label>
          </div>
          <AdminSectionHeader title="对自己主人" subtitle="投票者是被投票者的直系宠物时使用。" />
          <div className="config-row">
            <label className="field-label">
              <span>点赞每小时次数上限</span>
              <input type="number" min={1} step={1} value={draft.giveaway.masterLikeVoteLimitPerHour} onChange={(event) => patch({ giveaway: { ...draft.giveaway, masterLikeVoteLimitPerHour: Number(event.target.value) } })} />
            </label>
            <label className="field-label">
              <span>点赞降低值 (%)</span>
              <input type="number" min={0.01} max={100} step={0.01} value={draft.giveaway.masterLikeVoteValue} onChange={(event) => patch({ giveaway: { ...draft.giveaway, masterLikeVoteValue: Number(event.target.value) } })} />
            </label>
          </div>
          <div className="config-row">
            <label className="field-label">
              <span>倒赞每小时次数上限</span>
              <input type="number" min={1} step={1} value={draft.giveaway.masterDislikeVoteLimitPerHour} onChange={(event) => patch({ giveaway: { ...draft.giveaway, masterDislikeVoteLimitPerHour: Number(event.target.value) } })} />
            </label>
            <label className="field-label">
              <span>倒赞增加值 (%)</span>
              <input type="number" min={0.01} max={100} step={0.01} value={draft.giveaway.masterDislikeVoteValue} onChange={(event) => patch({ giveaway: { ...draft.giveaway, masterDislikeVoteValue: Number(event.target.value) } })} />
            </label>
          </div>
          <p className="hint">胜利扣减仅对已开启白给模式的胜方生效（含断线判负）。</p>
        </div>
      );
    }

    if (activeSection === "petBond") {
      const pb = draft.petBond || { panelTitle: "宠物乐园", maxPetsPerMaster: 3, maxMastersPerPet: 3, maxTitleLength: 12 };
      return (
        <>
          <div className="config-section admin-section-card">
            <AdminSectionHeader title="宠物乐园（认主/认宠）" subtitle="配置大厅面板标题、主人/宠物数量上限与称号长度。" />
            <div className="admin-preview-card">
              <span>预览</span>
              <strong>🐾 {pb.panelTitle}</strong>
              <p>主人最多 {pb.maxPetsPerMaster} 宠 · 宠物最多 {pb.maxMastersPerPet} 主 · 称号 {pb.maxTitleLength} 字</p>
            </div>
            <div className="config-row">
              <label className="field-label">
                <span>大厅面板标题</span>
                <input value={pb.panelTitle} maxLength={24} onChange={(event) => patch({ petBond: { ...pb, panelTitle: event.target.value } })} placeholder="宠物乐园" />
              </label>
              <label className="field-label">
                <span>宠物称号最大字数</span>
                <input type="number" min={1} max={24} step={1} value={pb.maxTitleLength} onChange={(event) => patch({ petBond: { ...pb, maxTitleLength: Number(event.target.value) } })} />
              </label>
            </div>
            <div className="config-row">
              <label className="field-label">
                <span>每名主人最多宠物数</span>
                <input type="number" min={1} max={20} step={1} value={pb.maxPetsPerMaster} onChange={(event) => patch({ petBond: { ...pb, maxPetsPerMaster: Number(event.target.value) } })} />
              </label>
              <label className="field-label">
                <span>每名宠物最多主人数</span>
                <input type="number" min={1} max={20} step={1} value={pb.maxMastersPerPet} onChange={(event) => patch({ petBond: { ...pb, maxMastersPerPet: Number(event.target.value) } })} />
              </label>
            </div>
            <p className="hint">关闭玩家侧「开启认主/认宠」不会解除已有关系，只禁止新增；关闭「公开展示」则不出现在大厅关系图。</p>
          </div>
          <div className="config-section admin-section-card">
            <PetBondGraphPanel onError={onError} />
          </div>
        </>
      );
    }

    if (activeSection === "accessControl") {
      const patchAccessControl = (next: Partial<AppConfig["accessControl"]>) =>
        patch({ accessControl: withAccessControlDefaults({ ...draft.accessControl, ...next }) });
      return (
        <div className="config-section admin-section-card">
          <AdminSectionHeader title="新用户注册开关" subtitle="禁止新用户注册，防止批量注册攻击" />
          <div className={draft.accessControl?.registrationDisabled ? "admin-preview-card admin-preview-card-warning" : "admin-preview-card"}>
            <Toggle
              label="禁止新用户注册"
              value={!!draft.accessControl?.registrationDisabled}
              onChange={(value) => patchAccessControl({ registrationDisabled: value })}
            />
            {draft.accessControl?.registrationDisabled ? <p>当前已禁止新用户注册，新玩家会看到「暂停新用户注册」提示，无法进入游戏。</p> : null}
          </div>
          <AdminSectionHeader title="指纹 + IP 限制策略" subtitle="按「出口 IP + 浏览器指纹」组合限流，仅用于防止用户自己多开。" />
          <div className="config-row">
            <label className="field-label">
              <span>同指纹同时在线人数上限</span>
              <input
                type="number"
                min={1}
                max={100}
                value={draft.accessControl?.maxOnlinePerIp ?? ""}
                onChange={(event) => patchAccessControl({ maxOnlinePerIp: Number(event.target.value) || 1 })}
              />
            </label>
            <label className="field-label">
              <span>同指纹 10 分钟内新建玩家上限</span>
              <input
                type="number"
                min={1}
                max={200}
                value={draft.accessControl?.maxCreatesPer10Min ?? ""}
                onChange={(event) => patchAccessControl({ maxCreatesPer10Min: Number(event.target.value) || 1 })}
              />
            </label>
          </div>
          <p className="hint">指纹由 FingerprintJS 在浏览器生成，与 IP 一起哈希为设备键。</p>
          <AdminSectionHeader title="IP 限制策略" subtitle="按请求者 IP 限流，用于防止攻击者伪造浏览器指纹批量攻击。" />
          <div className="config-row">
            <label className="field-label">
              <span>同 IP 同时在线人数上限</span>
              <input
                type="number"
                min={1}
                max={500}
                value={draft.accessControl?.maxOnlinePerIpTotal ?? ""}
                onChange={(event) => patchAccessControl({ maxOnlinePerIpTotal: Number(event.target.value) || 1 })}
              />
            </label>
            <label className="field-label">
              <span>同 IP 10 分钟内新建玩家上限</span>
              <input
                type="number"
                min={1}
                max={500}
                value={draft.accessControl?.maxCreatesPerIp ?? ""}
                onChange={(event) => patchAccessControl({ maxCreatesPerIp: Number(event.target.value) || 1 })}
              />
            </label>
            <label className="field-label">
              <span>同 IP 10 分钟内签发会话上限</span>
              <input
                type="number"
                min={1}
                max={500}
                value={draft.accessControl?.maxSessionIssuePerIp ?? ""}
                onChange={(event) => patchAccessControl({ maxSessionIssuePerIp: Number(event.target.value) || 1 })}
              />
            </label>
          </div>
          <div className="config-row">
            <label className="field-label">
              <span>单个操作的 IP 兜底倍数</span>
              <input
                type="number"
                min={1}
                max={100}
                value={draft.accessControl?.ipBackstopMultiplier ?? ""}
                onChange={(event) => patchAccessControl({ ipBackstopMultiplier: Number(event.target.value) || 1 })}
              />
            </label>
            <label className="field-label">
              <span>IP 兜底最低下限（次/窗口）</span>
              <input
                type="number"
                min={1}
                max={1000}
                value={draft.accessControl?.ipBackstopMinLimit ?? ""}
                onChange={(event) => patchAccessControl({ ipBackstopMinLimit: Number(event.target.value) || 1 })}
              />
            </label>
          </div>
          <p className="hint">建房、出招、提交惩罚证明等每种操作各自的频率上限，会按这个倍数换算出一个「同一 IP 总量」上限（不管换了多少个会话/指纹），并保证不低于最低下限——这样即使脚本不断重新登录换身份，同一出口 IP 的总请求量仍会被卡住。</p>

          <AdminSectionHeader title="房间与证明图片" subtitle="限制单个玩家同时占用房间数量、上传证明图片速率，防止恶意消耗服务器资源。" />
          <div className="config-row">
            <label className="field-label">
              <span>单玩家同时开房数量上限</span>
              <input
                type="number"
                min={1}
                max={50}
                value={draft.accessControl?.maxActiveRoomsPerOwner ?? ""}
                onChange={(event) => patchAccessControl({ maxActiveRoomsPerOwner: Number(event.target.value) || 1 })}
              />
            </label>
            <label className="field-label">
              <span>单玩家 10 分钟内证明图上传上限</span>
              <input
                type="number"
                min={1}
                max={200}
                value={draft.accessControl?.maxProofUploadsPerPlayer ?? ""}
                onChange={(event) => patchAccessControl({ maxProofUploadsPerPlayer: Number(event.target.value) || 1 })}
              />
            </label>
          </div>
        </div>
      );
    }

    if (activeSection === "roomTags") {
      return (
        <>
          <div className="config-section admin-section-card">
            <AdminSectionHeader title="房间标签" subtitle="玩家创建房间时，可以开启并选择这里配置好的标签。最多显示 5 个。" />
            <div className="admin-preview-card">
              <span>预览</span>
              <RoomTagList tags={draft.roomTags.slice(0, 5)} />
              <p>点下面标签可以删除；在输入框里输入文字后回车可以添加。</p>
            </div>
            <TagListEditor label="房间 Tag 池" placeholder="输入房间 Tag 后回车" values={draft.roomTags} onChange={(roomTags) => patch({ roomTags })} />
          </div>
          <div className="config-section admin-section-card">
            <AdminSectionHeader title="房间信息标签" subtitle="修改房间顶部规则标签的名字和颜色。" />
            <div className="admin-preview-card">
              <span>预览</span>
              <RoomInfoTagList tags={roomInfoTagOrder.slice(0, 7).map((item) => {
                const style = draft.roomInfoTags?.[item.key] || defaultRoomInfoTagStyle(item.label);
                return { key: item.key, text: style.label, style };
              })} />
              <p>这些标签会显示在房间信息卡里，部分也会显示在大厅房间卡上。</p>
            </div>
            <div className="room-info-tag-admin-grid">
              {roomInfoTagOrder.map((item) => {
                const style = draft.roomInfoTags?.[item.key] || defaultRoomInfoTagStyle(item.label);
                const nextTags = draft.roomInfoTags || {};
                const update = (nextStyle: RoomInfoTagStyle) => patch({ roomInfoTags: { ...nextTags, [item.key]: nextStyle } });
                return (
                  <div className="mini-card room-info-tag-admin-card" key={item.key}>
                    <div className="admin-card-title">
                      <strong>{item.label}</strong>
                      <small>{item.key}</small>
                    </div>
                    <span className="room-info-tag preview" style={roomInfoTagStyle(style)}>{style.label}</span>
                    <label className="field-label"><span>显示名字</span><input value={style.label} onChange={(event) => update({ ...style, label: event.target.value })} /></label>
                    <div className="color-grid">
                      <ColorInput label="文字颜色" value={style.textColor} onChange={(textColor) => update({ ...style, textColor })} />
                      <ColorInput label="背景颜色" value={style.backgroundColor} onChange={(backgroundColor) => update({ ...style, backgroundColor })} />
                      <ColorInput label="边框颜色" value={style.borderColor} onChange={(borderColor) => update({ ...style, borderColor })} />
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </>
      );
    }

    if (activeSection === "users") {
      const listHint = `匹配过滤器 · 在线 ${adminFilterOnlineCount} / 离线 ${adminFilterOfflineCount}`;
      const nameKeyword = playerNameSearch.trim().toLowerCase();
      const visiblePlayers = nameKeyword
        ? adminPlayers.filter((player) => (player.name || "").toLowerCase().includes(nameKeyword))
        : adminPlayers;
      return (
        <div className="config-section admin-section-card">
          <AdminSectionHeader title="用户管理" subtitle="按过滤器查询玩家档案，协助改资料、踢出与找回认领密钥。" />
          <div className="admin-player-filters" role="group" aria-label="用户列表过滤器">
            <input
              className="admin-player-name-search"
              value={playerNameSearch}
              onChange={(event) => setPlayerNameSearch(event.target.value)}
              placeholder="搜索昵称，留空则不筛选"
              aria-label="按昵称搜索"
              autoComplete="off"
            />
            {ADMIN_PLAYER_FILTER_BUTTONS.map((item) => {
              const active = playerFilters[item.key];
              return (
                <button
                  type="button"
                  key={item.key}
                  className={`admin-filter-btn${active ? " active" : ""}`}
                  aria-pressed={active}
                  onClick={() => togglePlayerFilter(item.key)}
                >
                  {item.label}
                </button>
              );
            })}
          </div>
          <div className="admin-list-section admin-player-list">
            <div className="admin-list-heading">
              <h3>玩家列表</h3>
              <span>
                {adminPlayersLoading
                  ? "加载中…"
                  : nameKeyword
                    ? `昵称匹配 ${visiblePlayers.length} / 已加载 ${adminPlayers.length} 人 · ${listHint}`
                    : adminPlayersTruncated
                      ? `显示 ${adminPlayers.length} / 共 ${adminPlayersTotal} 人（已截断）· ${listHint}`
                      : `${adminPlayersTotal} 人 · ${listHint}`}
              </span>
            </div>
            {visiblePlayers.map((player) => (
              <AdminPlayerEditor
                key={player.id}
                config={config}
                player={player}
                onSave={async (payload) => {
                  if (await action("editPlayer", payload)) await loadAdminPlayers();
                }}
                onKick={async () => {
                  if (await action("kick", { playerId: player.id })) await loadAdminPlayers();
                }}
                onError={onError}
              />
            ))}
            {!adminPlayersLoading && visiblePlayers.length === 0 && (
              <p className="empty">{nameKeyword ? "没有昵称匹配的玩家" : "当前没有符合条件的玩家"}</p>
            )}
          </div>
        </div>
      );
    }

    if (activeSection === "rooms") {
      const stats = lobby.serverStats;
      return (
        <div className="config-section admin-section-card">
          <AdminSectionHeader title="房间管理" subtitle="查看运行状态，管理房间与聊天（用户管理已独立到左侧「用户管理」）。" />
          <div className="admin-preview-card">
            <span>运行状态</span>
            <p>在线 {lobby.onlineCount} 人 · 房间 {lobby.rooms.length} 个 · 运行 {formatDuration(Date.now() - stats.startedAt)}</p>
            <p>房间广播 {stats.roomBroadcasts} 次 · 大厅广播 {stats.lobbyBroadcasts} 次</p>
            <p>最近 1 分钟：房间 {stats.recentRoomBroadcasts} 次 · 大厅 {stats.recentLobbyBroadcasts} 次</p>
            <p>断线 {stats.disconnects} 次 · 重连 {stats.reconnects} 次</p>
            <p>最近房间快照 {formatBytes(stats.lastRoomSnapshotBytes)} · 最近大厅快照 {formatBytes(stats.lastLobbySnapshotBytes)}</p>
            <p>平均快照：房间 {formatBytes(stats.averageRoomSnapshotBytes)} · 大厅 {formatBytes(stats.averageLobbySnapshotBytes)}</p>
          </div>
          <div className="admin-action-tabs admin-room-tabs">
            {[
              { id: "rooms" as const, label: "房间", count: lobby.rooms.length },
              { id: "announcement" as const, label: "聊天管理", count: 0 }
            ].map((tab) => (
              <button
                type="button"
                className={activeRoomTab === tab.id ? "active" : ""}
                key={tab.id}
                onClick={() => setActiveRoomTab(tab.id)}
              >
                <span>{tab.label}</span>
                {tab.count > 0 && <em>{tab.count}</em>}
              </button>
            ))}
          </div>
          {activeRoomTab === "announcement" && <AdminChatManager onError={onError} action={action} />}
          {activeRoomTab === "rooms" && (
            <div className="admin-list-section">
              <div className="admin-list-heading">
                <h3>房间列表</h3>
                <span>{lobby.rooms.length} 间</span>
              </div>
              {lobby.rooms.map((room) => {
                const canForceSeatOutcome = room.status === "playing" && room.gameId !== "liarsdice"
                  && room.versus.A != null && room.versus.B != null;
                const seatALabel = room.versus.A?.player?.name || "A方";
                const seatBLabel = room.versus.B?.player?.name || "B方";
                return (
                  <div className="admin-room" key={room.id}>
                    <div className="admin-card-title">
                      <strong>{room.name}</strong>
                      <small>{room.id} · {roomStatusText(room.status)} · {room.players}/2 战斗席 · {room.spectators} 观战</small>
                    </div>
                    {room.tags?.length ? <RoomTagList tags={room.tags} /> : null}
                    <RoomInfoTagList tags={lobbyRoomInfoTags(config, room)} />
                    <div className="admin-action-row othello-admin-actions">
                      <button className="danger-button" onClick={() => action("closeRoom", { roomId: room.id })}>关闭房间</button>
                      <button onClick={() => action("forceNext", { roomId: room.id })}>重开</button>
                      <button onClick={() => action("forceSeatOutcome", { roomId: room.id, result: "A" })} disabled={!canForceSeatOutcome}>判 {seatALabel} 胜</button>
                      <button onClick={() => action("forceSeatOutcome", { roomId: room.id, result: "B" })} disabled={!canForceSeatOutcome}>判 {seatBLabel} 胜</button>
                      <button onClick={() => action("forceSeatOutcome", { roomId: room.id, result: "draw" })} disabled={!canForceSeatOutcome}>判平</button>
                    </div>
                  </div>
                );
              })}
              {lobby.rooms.length === 0 && <p className="empty">暂无房间</p>}
            </div>
          )}
        </div>
      );
    }

    return null;
  }

  return (
    <section className="admin-page">
      <div className="panel admin-login-card">
        <h2><Shield size={18} /> 管理员与文本工具</h2>
        <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} placeholder="管理员口令" />
        <button className="primary" onClick={login}>进入管理</button>
        <button onClick={onBack}>返回</button>
      </div>
      {logged && (
        <div className="admin-tool-shell">
          <nav className="admin-sidebar" aria-label="后台配置分类">
            {navItems.map((item) => (
              <button className={activeSection === item.id ? "active" : ""} key={item.id} onClick={() => switchSection(item.id)}>
                <span>{item.label}</span>
                <small>{item.detail}</small>
              </button>
            ))}
          </nav>
          <div className="panel visual-config admin-editor-panel">
            <div className="admin-editor-head">
              <div>
                <h2><Settings size={18} /> {currentNav.label}</h2>
                <p className="hint">{currentNav.detail}</p>
              </div>
              <div className="admin-edit-status">
                {dirty && <span>有未保存修改</span>}
                {serverConfigChanged && <small>服务器配置已更新，保存会覆盖当前服务器配置。</small>}
              </div>
            </div>
            {renderSection()}
            {!SECTIONS_WITHOUT_SAVE.has(activeSection) && (
              <div className="admin-sticky-actions">
                <button className="primary" onClick={save}><Save size={16} /> 保存配置</button>
                <button onClick={resetDefault} title="从磁盘重新读取 config/*.json"><RefreshCcw size={16} /> 重新加载</button>
                <button onClick={exportConfig}><Download size={16} /> 导出配置</button>
              </div>
            )}
          </div>
        </div>
      )}
    </section>
  );
}

export function nextAdminId(prefix: string, existingIds: string[]) {
  const safePrefix = prefix.replace(/[^a-zA-Z0-9_]/g, "_") || "item";
  const used = new Set(existingIds);
  let index = 1;
  while (used.has(`${safePrefix}${index}`)) index += 1;
  return `${safePrefix}${index}`;
}

// factionSummaryLabel：任务编辑器里展示"这条任务给哪些阵营"的一句话摘要。
// 未勾选任何阵营的任务永远不会被抽到（既不进入随机模式候选池，也不能作为系列任务的候选）。
export function factionSummaryLabel(factionIds: string[], genderFactions: GenderFaction[]) {
  if (!factionIds.length) return "不会被抽到";
  const labels = factionIds.map((id) => genderFactions.find((faction) => faction.id === id)?.label || id);
  return labels.join("、");
}

// seriesStepMissingFactionLabels：某一步的候选任务合起来未覆盖到的阵营展示名列表
// （与后端 seriesIsUsable 判定逻辑保持一致：未勾选阵营的任务不计入覆盖）。
export function seriesStepMissingFactionLabels(taskIds: string[], taskPool: PunishmentTaskConfig[], genderFactions: GenderFaction[]) {
  if (!genderFactions.length) return [];
  const taskById = new Map(taskPool.map((t) => [t.id, t]));
  const covered = new Set<string>();
  for (const id of taskIds) {
    const t = taskById.get(id);
    if (!t) continue;
    for (const fid of t.factionIds || []) covered.add(fid);
  }
  return genderFactions.filter((f) => !covered.has(f.id)).map((f) => f.label);
}

// seriesIsUsable：系列每一步候选任务合起来是否都覆盖了全部已定义阵营。覆盖不全时
// 后端会把该系列当"不存在"处理（建房面板选不到、运行时也不产出任务），这里同步
// 判定用于编辑器里标红提示，帮助管理员在保存前发现问题。
export function seriesIsUsable(series: PunishmentSeriesTaskConfig, taskPool: PunishmentTaskConfig[], genderFactions: GenderFaction[]) {
  const steps = series.steps || [];
  if (!steps.length) return false;
  return steps.every((step) => seriesStepMissingFactionLabels(step.taskIds || [], taskPool, genderFactions).length === 0);
}

export function newFlatPunishmentTask(id: string, defaultTagId?: string): PunishmentTaskConfig {
  return {
    id,
    name: "新任务",
    text: "写下这条任务的文案，可用 {loser}/{winner}",
    tagIds: defaultTagId ? [defaultTagId] : [],
    factionIds: [],
    order: 50,
    backgroundImages: [],
    backgroundOpacity: 0.22
  };
}

export function newSeriesTask(id: string): PunishmentSeriesTaskConfig {
  return {
    id,
    name: "新系列",
    roomBackgroundImages: [],
    roomNamePool: defaultAdminRoomNamePool(),
    steps: [{ taskIds: [] }]
  };
}

/** 后台聊天检索结果的一行；在玩家可见的聊天字段之上叠加房间名快照与软删除状态。 */
type AdminChatMessage = {
  id: string;
  roomId?: string;
  roomName?: string;
  playerId: string;
  author: string;
  text: string;
  at: number;
  deleted?: boolean;
  deletedAt?: number;
};

/** 定位一条聊天消息给后端 chatSoftDelete/chatRestore：roomId 空串表示大厅。 */
function chatRefKey(m: AdminChatMessage): string {
  return `${m.roomId || ""}\u0000${m.id}`;
}

function formatAdminChatTime(at: number) {
  const d = new Date(at);
  if (!Number.isFinite(d.getTime()) || d.getTime() <= 0) return "未知时间";
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function adminChatAuthorName(author: string) {
  const text = (author || "").trim();
  if (!text) return "匿名";
  const parts = text.split(" - ");
  return (parts[parts.length - 1] || text).trim() || "匿名";
}

function adminChatScopeLabel(m: AdminChatMessage) {
  if (!m.roomId) return "大厅";
  const name = (m.roomName || "").trim();
  return name || "已关闭房间";
}

function AdminChatSpeaker({
  player,
  fallbackName,
  isLobby,
  scopeLabel
}: {
  player?: PublicPlayer;
  fallbackName: string;
  isLobby: boolean;
  scopeLabel: string;
}) {
  const stats = player ? safePlayerStats(player) : undefined;
  const punished = Boolean(player?.nameWarPunished && player?.nameWarPenaltyName);
  return (
    <>
      <PlayerAvatar player={player} size={24} />
      {player ? (
        <>
          <span className="gender-chip" style={genderStyle(player)} title={player.factionLabel}>{player.genderLabel || "未知"}</span>
          <span className="title-chip" style={titleStyle(stats?.titleColors)}>{stats?.title || "无称号"}</span>
          <strong className={punished ? "name-war-pill" : ""}>{displayPlayerName(player)}</strong>
          <GiveawayChip player={player} />
          <ModeChip player={player} />
          {player.nameWarEnabled && player.nameWarPunished && !player.extremeModeEnabled && (
            <span className="mode-chip">⚔️ 名争</span>
          )}
          <span className={`admin-chat-presence ${player.connected ? "online" : "offline"}`}>
            {player.connected ? "在线" : "离线"}
          </span>
        </>
      ) : (
        <>
          <strong>{fallbackName}</strong>
          <span className="admin-chat-presence offline">离线</span>
        </>
      )}
      <span className={`admin-chat-scope ${isLobby ? "lobby" : "room"}`}>{scopeLabel}</span>
    </>
  );
}

/**
 * 「聊天管理」面板：按用户名/聊天内容/房间名（或仅大厅）检索历史消息，多选后批量软删除
 * /恢复。取代旧版"一键清空大厅/房间聊天"——只能对检索出的具体违规留言操作，删除是软删除
 * （数据库保留、普通玩家读不到），删除后端会主动推 chat:deleted 通知在线客户端摘除本地视图。
 */
function AdminChatManager({ onError, action }: {
  onError: (message: string) => void;
  action: (actionName: string, payload?: Record<string, unknown>) => Promise<boolean>;
}) {
  const [author, setAuthor] = useState("");
  const [text, setText] = useState("");
  const [room, setRoom] = useState("");
  const [lobbyOnly, setLobbyOnly] = useState(false);
  const [includeDeleted, setIncludeDeleted] = useState(false);
  const [messages, setMessages] = useState<AdminChatMessage[]>([]);
  const [authors, setAuthors] = useState<Record<string, PublicPlayer>>({});
  const [hasMore, setHasMore] = useState(false);
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);
  const [searchError, setSearchError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const searchRequestGen = useRef(0);

  async function runSearch(offset: number, append: boolean) {
    const gen = ++searchRequestGen.current;
    setLoading(true);
    setSearchError(null);
    try {
      const res = await ask<{ messages?: AdminChatMessage[]; hasMore?: boolean; authors?: Record<string, PublicPlayer> }>("admin:chatSearch", {
        author, text, room: lobbyOnly ? "" : room, lobbyOnly, includeDeleted, limit: 50, offset
      });
      if (gen !== searchRequestGen.current) return;
      const list = res.messages || [];
      const nextAuthors = res.authors || {};
      setMessages((prev) => (append ? [...prev, ...list] : list));
      setAuthors((prev) => (append ? { ...prev, ...nextAuthors } : nextAuthors));
      setHasMore(!!res.hasMore);
      setSearched(true);
      if (!append) setSelected(new Set());
    } catch (error) {
      if (gen !== searchRequestGen.current) return;
      const message = error instanceof Error ? error.message : "检索失败";
      setSearchError(message);
      setSearched(true);
      onError(message);
    } finally {
      if (gen === searchRequestGen.current) setLoading(false);
    }
  }

  useEffect(() => {
    void runSearch(0, false);
    // 进入页签时拉最近消息；之后只在点「检索」、切换范围开关或加载更多时再请求。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const skipToggleSearch = useRef(true);
  useEffect(() => {
    if (skipToggleSearch.current) {
      skipToggleSearch.current = false;
      return;
    }
    void runSearch(0, false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lobbyOnly, includeDeleted]);

  function toggleSelect(key: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key); else next.add(key);
      return next;
    });
  }

  function toggleSelectAll() {
    setSelected((prev) => (prev.size === messages.length && messages.length > 0 ? new Set() : new Set(messages.map(chatRefKey))));
  }

  async function applyToRefs(actionName: "chatSoftDelete" | "chatRestore", refs: { roomId: string; id: string }[]) {
    if (refs.length === 0) {
      onError("请先勾选要操作的消息");
      return;
    }
    if (await action(actionName, { refs })) {
      onError(actionName === "chatSoftDelete" ? `已删除 ${refs.length} 条` : `已恢复 ${refs.length} 条`);
      await runSearch(0, false);
    }
  }

  const selectedRefs = messages.filter((m) => selected.has(chatRefKey(m))).map((m) => ({ roomId: m.roomId || "", id: m.id }));
  const allSelected = messages.length > 0 && selected.size === messages.length;
  const resultHint = loading && !searched
    ? "检索中…"
    : searchError
      ? "检索失败"
    : hasMore
      ? `已显示 ${messages.length} 条 · 还有更多`
      : `${messages.length} 条`;

  return (
    <div className="admin-chat-manager">
      <form
        className="admin-chat-filter-card"
        onSubmit={(event) => {
          event.preventDefault();
          void runSearch(0, false);
        }}
      >
        <div className="admin-chat-filter-grid">
          <label className="field-label">
            <span>用户名</span>
            <input value={author} onChange={(event) => setAuthor(event.target.value)} placeholder="按发言人昵称" autoComplete="off" />
          </label>
          <label className="field-label">
            <span>聊天内容</span>
            <input value={text} onChange={(event) => setText(event.target.value)} placeholder="按消息关键字" autoComplete="off" />
          </label>
          <label className="field-label">
            <span>房间名</span>
            <input value={room} onChange={(event) => setRoom(event.target.value)} placeholder={lobbyOnly ? "只看大厅时不可用" : "留空则不限房间"} disabled={lobbyOnly} autoComplete="off" />
          </label>
        </div>
        <div className="admin-chat-filter-actions">
          <div className="admin-player-filters" role="group" aria-label="检索范围">
            <button
              type="button"
              className={`admin-filter-btn${lobbyOnly ? " active" : ""}`}
              aria-pressed={lobbyOnly}
              onClick={() => setLobbyOnly((old) => {
                const next = !old;
                if (next) setRoom("");
                return next;
              })}
            >
              只看大厅
            </button>
            <button
              type="button"
              className={`admin-filter-btn${includeDeleted ? " active" : ""}`}
              aria-pressed={includeDeleted}
              onClick={() => setIncludeDeleted((old) => !old)}
            >
              显示已删除
            </button>
          </div>
          <button className="primary" type="submit" disabled={loading}>{loading ? "检索中…" : "检索"}</button>
        </div>
      </form>

      <div className="admin-list-section admin-chat-list-section">
        <div className="admin-list-heading">
          <h3>检索结果</h3>
          <span>{resultHint}</span>
        </div>
        <div className="admin-chat-bulk-row">
          <label className="admin-chat-select-all">
            <input
              type="checkbox"
              checked={allSelected}
              onChange={toggleSelectAll}
              disabled={messages.length === 0}
            />
            全选
            {messages.length > 0 && <em>已选 {selected.size}/{messages.length}</em>}
          </label>
          <div className="admin-chat-bulk-actions">
            <button type="button" className="danger-button small" onClick={() => applyToRefs("chatSoftDelete", selectedRefs)} disabled={selected.size === 0}>删除所选</button>
            <button type="button" className="small" onClick={() => applyToRefs("chatRestore", selectedRefs)} disabled={selected.size === 0}>恢复所选</button>
          </div>
        </div>
        <div className="admin-chat-list" role="list">
          {messages.map((m) => {
            const key = chatRefKey(m);
            const isSelected = selected.has(key);
            const isLobby = !m.roomId;
            const player = m.playerId ? authors[m.playerId] : undefined;
            const authorLabel = player ? displayPlayerName(player) : adminChatAuthorName(m.author);
            return (
              <div
                className={`admin-chat-row${m.deleted ? " deleted" : ""}${isSelected ? " selected" : ""}`}
                key={key}
                role="listitem"
                onClick={() => toggleSelect(key)}
              >
                <input
                  type="checkbox"
                  checked={isSelected}
                  onChange={() => toggleSelect(key)}
                  onClick={(event) => event.stopPropagation()}
                  aria-label={`选择 ${authorLabel} 的消息`}
                />
                <div className="admin-chat-row-body">
                  <div className="admin-chat-row-meta">
                    <AdminChatSpeaker
                      player={player}
                      fallbackName={authorLabel}
                      isLobby={isLobby}
                      scopeLabel={adminChatScopeLabel(m)}
                    />
                    {m.deleted && <span className="admin-chat-deleted-chip">已删除</span>}
                    <time className="admin-chat-time" dateTime={m.at ? new Date(m.at).toISOString() : undefined}>{formatAdminChatTime(m.at)}</time>
                  </div>
                  <p>{m.text || "（空消息）"}</p>
                </div>
                <div className="admin-chat-row-actions">
                  {m.deleted
                    ? (
                      <button
                        type="button"
                        className="small"
                        onClick={(event) => {
                          event.stopPropagation();
                          void applyToRefs("chatRestore", [{ roomId: m.roomId || "", id: m.id }]);
                        }}
                      >
                        恢复
                      </button>
                    )
                    : (
                      <button
                        type="button"
                        className="danger-button small"
                        onClick={(event) => {
                          event.stopPropagation();
                          void applyToRefs("chatSoftDelete", [{ roomId: m.roomId || "", id: m.id }]);
                        }}
                      >
                        删除
                      </button>
                    )}
                </div>
              </div>
            );
          })}
          {!loading && messages.length === 0 && (
            <p className="empty">
              {searchError ? `检索失败：${searchError}` : searched ? "没有符合条件的聊天消息" : "正在加载最近消息…"}
            </p>
          )}
        </div>
        {hasMore && (
          <button type="button" className="admin-chat-more" onClick={() => runSearch(messages.length, true)} disabled={loading}>
            {loading ? "加载中…" : "加载更多"}
          </button>
        )}
      </div>
    </div>
  );
}

export function AdminSectionHeader({ title, subtitle }: { title: string; subtitle: string }) {
  return (
    <div className="admin-section-header">
      <h3>{title}</h3>
      <p className="hint">{subtitle}</p>
    </div>
  );
}

type AdminAltAccountsState = { loading: boolean; queried: boolean; error: string | null; players: PublicPlayer[]; truncated: boolean };
const ADMIN_ALT_ACCOUNTS_IDLE: AdminAltAccountsState = { loading: false, queried: false, error: null, players: [], truncated: false };

export function AdminPlayerEditor({ config, player, onSave, onKick, onError }: { config: AppConfig; player: PublicPlayer; onSave: (payload: Record<string, unknown>) => void; onKick: () => void; onError: (message: string) => void }) {
  const [name, setName] = useState(player.name);
  const [rankedPoints, setRankedPoints] = useState(String(safePlayerStats(player).sortRankedPoints));
  const [rankedPointsTouched, setRankedPointsTouched] = useState(false);
  const [title, setTitle] = useState(safePlayerStats(player).title);
  const [titleTouched, setTitleTouched] = useState(false);
  const [giveawayInput, setGiveawayInput] = useState(formatGiveawayValue(player.giveawayEnabled ? player.giveawayValue || 0 : 0));
  const [giveawayTouched, setGiveawayTouched] = useState(false);
  const [genderId, setGenderId] = useState(player.genderId);
  const [factionId, setFactionId] = useState(player.factionId);
  const [genderTouched, setGenderTouched] = useState(false);
  const focusedFieldRef = useRef<"name" | "rankedPoints" | "title" | "giveaway" | "gender" | null>(null);
  const titleCustom = !!safePlayerStats(player).titleCustom;
  const [altAccounts, setAltAccounts] = useState<AdminAltAccountsState>(ADMIN_ALT_ACCOUNTS_IDLE);

  useEffect(() => {
    const stats = safePlayerStats(player);
    if (focusedFieldRef.current !== "name") setName(player.name);
    if (focusedFieldRef.current !== "rankedPoints") {
      setRankedPoints(String(stats.sortRankedPoints));
      setRankedPointsTouched(false);
    }
    if (focusedFieldRef.current !== "title") {
      setTitle(stats.title);
      setTitleTouched(false);
    }
    if (focusedFieldRef.current !== "giveaway") {
      setGiveawayInput(formatGiveawayValue(player.giveawayEnabled ? player.giveawayValue || 0 : 0));
      setGiveawayTouched(false);
    }
    if (focusedFieldRef.current !== "gender") {
      setGenderId(player.genderId);
      setFactionId(player.factionId);
      setGenderTouched(false);
    }
  }, [player.id, player.name, player.stats.sortRankedPoints, player.stats.title, player.giveawayEnabled, player.giveawayValue, player.genderId, player.factionId]);

  useEffect(() => {
    setAltAccounts(ADMIN_ALT_ACCOUNTS_IDLE);
  }, [player.id]);

  async function queryAltAccounts() {
    setAltAccounts((prev) => ({ ...prev, loading: true, error: null }));
    try {
      const result = await ask<{ players?: PublicPlayer[]; truncated?: boolean }>("admin:action", {
        action: "altAccounts",
        playerId: player.id
      });
      setAltAccounts({ loading: false, queried: true, error: null, players: result.players || [], truncated: !!result.truncated });
    } catch (error) {
      setAltAccounts({
        loading: false,
        queried: true,
        error: error instanceof Error ? error.message : "查询失败",
        players: [],
        truncated: false
      });
    }
  }

  return (
    <div className="admin-player-editor">
      <div className="admin-player-head">
        <PlayerAvatar player={player} size={28} />
        <PlayerBadge player={player} compact />
        {player.nameWarEnabled && (
          <span className="mode-chip">
            {player.nameWarPunished ? `名字争夺战中：${player.nameWarPenaltyName || "惩罚名生效"}` : "名字争夺战已开启"}
          </span>
        )}
      </div>
      <div className="admin-player-row1">
        <label className="field-label">
          <span>名字</span>
          <input
            value={name}
            maxLength={12}
            onFocus={() => { focusedFieldRef.current = "name"; }}
            onBlur={() => { if (focusedFieldRef.current === "name") focusedFieldRef.current = null; }}
            onChange={(event) => setName(event.target.value)}
          />
        </label>
        <label className="field-label">
          <span>称号{titleCustom ? "（管理员自定义）" : ""}</span>
          <input
            value={title}
            maxLength={18}
            className={titleCustom ? "input-title-custom" : undefined}
            title={titleCustom ? "已由管理员手动设置，不随排位分变化自动改写；清空后保存可恢复自动称号" : undefined}
            onFocus={() => { focusedFieldRef.current = "title"; }}
            onBlur={() => { if (focusedFieldRef.current === "title") focusedFieldRef.current = null; }}
            onChange={(event) => { setTitle(event.target.value); setTitleTouched(true); }}
          />
        </label>
        <label className="field-label">
          <span>积分</span>
          <input
            type="number"
            value={rankedPoints}
            onFocus={() => { focusedFieldRef.current = "rankedPoints"; }}
            onBlur={() => { if (focusedFieldRef.current === "rankedPoints") focusedFieldRef.current = null; }}
            onChange={(event) => {
              setRankedPoints(event.target.value);
              setRankedPointsTouched(true);
            }}
          />
        </label>
        <label className="field-label">
          <span>白给值</span>
          <input
            type="number"
            min={0}
            max={100}
            step={0.1}
            value={giveawayInput}
            onFocus={() => { focusedFieldRef.current = "giveaway"; }}
            onBlur={() => { if (focusedFieldRef.current === "giveaway") focusedFieldRef.current = null; }}
            onChange={(event) => { setGiveawayInput(event.target.value); setGiveawayTouched(true); }}
            placeholder="0-100，精确到 0.1"
          />
        </label>
      </div>
      <div
        className="admin-player-row2"
        onFocus={() => { focusedFieldRef.current = "gender"; }}
        onBlur={() => { if (focusedFieldRef.current === "gender") focusedFieldRef.current = null; }}
      >
        <FactionSelect
          config={config}
          factionId={factionId}
          onFactionChange={(value) => {
            setFactionId(value);
            setGenderId((old) => nextGenderIdForFaction(config, value, old));
            setGenderTouched(true);
          }}
        />
        <GenderSelectField
          config={config}
          genderId={genderId}
          factionId={factionId}
          onGenderChange={(value) => { setGenderId(value); setGenderTouched(true); }}
        />
        <ClaimKeyRevealField playerId={player.id} onError={onError} />
      </div>
      <div className="admin-action-row">
        <button
          className="primary"
          onClick={() => {
            if (genderTouched) {
              const genderError = genderChoiceError(config, genderId);
              if (genderError) {
                onError(genderError);
                return;
              }
            }
            onSave({
              playerId: player.id,
              name,
              ...(rankedPointsTouched ? { rankedPoints: Number(rankedPoints) } : {}),
              ...(titleTouched ? { title } : {}),
              ...(giveawayTouched ? { giveawayValueInput: giveawayInput } : {}),
              ...(genderTouched ? { genderId } : {})
            });
          }}
        >
          保存玩家资料
        </button>
        <button onClick={queryAltAccounts} disabled={altAccounts.loading}>
          {altAccounts.loading ? "查询中…" : "小号查询"}
        </button>
        <button className="danger-button" onClick={onKick}>踢出</button>
      </div>
      {altAccounts.queried && (
        <div className="admin-alt-accounts">
          {altAccounts.error && <p className="empty">{altAccounts.error}</p>}
          {!altAccounts.error && altAccounts.players.length === 0 && (
            <p className="empty">未查询到该玩家关联设备登录过的其它账号</p>
          )}
          {!altAccounts.error && altAccounts.truncated && (
            <p className="hint">关联范围过大，已达到服务器查询上限，本次结果可能不完整。</p>
          )}
          {!altAccounts.error && altAccounts.players.length > 0 && altAccounts.players.map((alt) => (
            <div className="admin-alt-accounts-row" key={alt.id}>
              <PlayerAvatar player={alt} size={24} />
              <PlayerBadge player={alt} compact />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// 认领密钥显示框：默认显示占位文案；聚焦（点击）时签发一把新密钥（旧密钥立即作废，见后端
// showClaimKey 的无条件轮换）并自动复制到剪贴板；失焦 5 秒后（期间重新聚焦会取消这次复原、
// 不重新签发）恢复默认文案，再次聚焦视为全新一轮，会再签发一把新密钥。
function ClaimKeyRevealField({ playerId, onError }: { playerId: string; onError: (message: string) => void }) {
  const DEFAULT_TEXT = "点击显示认领密钥";
  const [value, setValue] = useState(DEFAULT_TEXT);
  const [revealed, setRevealed] = useState(false);
  const revertTimerRef = useRef<number | null>(null);

  useEffect(() => {
    return () => {
      if (revertTimerRef.current != null) window.clearTimeout(revertTimerRef.current);
    };
  }, []);

  async function reveal() {
    try {
      const result = await ask<{ claimKey: string; playerId: string }>("admin:action", {
        action: "showClaimKey",
        playerId
      });
      const code = encodeClaimCode(result.playerId, result.claimKey);
      setValue(code);
      setRevealed(true);
      try {
        await navigator.clipboard.writeText(code);
      } catch {
        onError("复制失败，请手动选中密钥");
      }
    } catch (error) {
      onError(error instanceof Error ? error.message : "获取认领密钥失败");
    }
  }

  function handleFocus() {
    if (revertTimerRef.current != null) {
      window.clearTimeout(revertTimerRef.current);
      revertTimerRef.current = null;
    }
    if (!revealed) void reveal();
  }

  function handleBlur() {
    revertTimerRef.current = window.setTimeout(() => {
      setValue(DEFAULT_TEXT);
      setRevealed(false);
      revertTimerRef.current = null;
    }, 5000);
  }

  return (
    <label className="field-label">
      <span>认领密钥</span>
      <input
        className="admin-claim-key-input"
        readOnly
        value={value}
        spellCheck={false}
        autoComplete="off"
        onFocus={handleFocus}
        onBlur={handleBlur}
      />
    </label>
  );
}

export function sampleRoomName(pool?: RoomNamePool) {
  const target = pool || defaultAdminRoomNamePool();
  const adjective = target.adjectives[0] || "";
  const subject = target.subjects[0] || "任务";
  const roomWord = target.roomWords[0] || "房间";
  return `${adjective}${subject}${roomWord}`;
}

export function RoomNamePoolEditor({ title, pool, onChange }: { title: string; pool: RoomNamePool; onChange: (pool: RoomNamePool) => void }) {
  return (
    <div className="room-name-pool-editor">
      <b>{title}</b>
      <TagListEditor label="形容词（可为空）" placeholder="输入形容词后回车" values={pool.adjectives} onChange={(adjectives) => onChange({ ...pool, adjectives })} />
      <TagListEditor label="名词/动词" placeholder="输入名词或动词后回车" values={pool.subjects} onChange={(subjects) => onChange({ ...pool, subjects })} />
      <TagListEditor label="房间词" placeholder="输入房间词后回车" values={pool.roomWords} onChange={(roomWords) => onChange({ ...pool, roomWords })} />
    </div>
  );
}

// AdminBackgroundImageField：单张背景图字段（房间信息卡图库 / 任务背景图库），
// 交互对齐个人资料页的头像上传：点击上传即设置/覆盖，点击清空则删除。
// 数据仍存成 string[]（历史上是随机图库），这里只取/写第一个元素，约束成最多 1 张。
export function AdminBackgroundImageField({ label, values, upload, onChange, onError }: { label: string; values: string[]; upload: (file: File) => Promise<string>; onChange: (values: string[]) => void; onError: (message: string) => void }) {
  const [busy, setBusy] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const current = values[0] || "";

  async function handleUpload(file: File) {
    setBusy(true);
    try {
      const imageUrl = await upload(file);
      onChange([imageUrl]);
    } catch (error) {
      onError(error instanceof Error ? error.message : "上传失败");
    } finally {
      setBusy(false);
      if (inputRef.current) inputRef.current.value = "";
    }
  }

  return (
    <div className="admin-bg-image-field">
      <span>{label}</span>
      <div className="admin-bg-image-row">
        {current ? <img className="admin-bg-image-preview" src={current} alt="" /> : <em>暂无背景图</em>}
        <input
          ref={inputRef}
          type="file"
          accept="image/png,image/jpeg,image/webp"
          className="admin-bg-image-input"
          disabled={busy}
          onChange={(event) => {
            const file = event.target.files?.[0];
            if (file) void handleUpload(file);
          }}
        />
        <button type="button" disabled={busy} onClick={() => inputRef.current?.click()}>
          <Upload size={15} /> {busy ? "上传中…" : current ? "更换背景图" : "上传背景图"}
        </button>
        <button type="button" disabled={busy || !current} onClick={() => onChange([])}>清空</button>
      </div>
    </div>
  );
}

export function TagListEditor({ label, placeholder, values, onChange }: { label: string; placeholder: string; values: string[]; onChange: (values: string[]) => void }) {
  const [draftTag, setDraftTag] = useState("");

  function addTag() {
    const next = draftTag.trim();
    if (!next) return;
    if (!values.includes(next)) onChange([...values, next]);
    setDraftTag("");
  }

  return (
    <div className="tag-list-editor">
      <span>{label}</span>
      <div className="tag-list">
        {values.map((value) => (
          <button type="button" className="tag-chip" key={value} onClick={() => onChange(values.filter((item) => item !== value))}>
            {value}<small>×</small>
          </button>
        ))}
        {values.length === 0 && <em>暂无词条</em>}
      </div>
      <div className="tag-input-row">
        <input value={draftTag} onChange={(event) => setDraftTag(event.target.value)} onKeyDown={(event) => { if (event.key === "Enter") { event.preventDefault(); addTag(); } }} placeholder={placeholder} />
        <button type="button" onClick={addTag}>添加</button>
      </div>
    </div>
  );
}

export function emptyRoomNamePool(): RoomNamePool {
  return { adjectives: [], subjects: [], roomWords: [] };
}

export function defaultAdminRoomNamePool(): RoomNamePool {
  return { adjectives: ["粉蓝", "闪亮", "神秘"], subjects: ["任务", "挑战", "惩罚"], roomWords: ["小屋", "房间", "擂台"] };
}

export function ColorInput({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <label className="color-input">
      <span>{label}</span>
      <input type="color" value={value} onChange={(event) => onChange(event.target.value)} />
      <input value={value} onChange={(event) => onChange(event.target.value)} placeholder="#RRGGBB" />
    </label>
  );
}
