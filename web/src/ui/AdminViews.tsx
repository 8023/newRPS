import { type CSSProperties, useEffect, useRef, useState } from "react";
import { Download, RefreshCcw, Save, Settings, Shield, Upload } from "lucide-react";
import type { AppConfig, GenderFaction, LobbySnapshot, PublicPlayer, PunishmentTaskConfig, RoomInfoTagStyle, RoomNamePool } from "../shared/types";
import { DEFAULT_NAME_WAR_PENALTY_THRESHOLD, withAccessControlDefaults, withRankedScoreDefaults } from "../lib/normalize";
import { ask } from "../lib/rpc";
import { compressAdminImageForUpload } from "../lib/proofImage";
import { formatBytes, formatDuration } from "../lib/format";
import { encodeClaimCode } from "../lib/session";
import { PetBondGraphPanel } from "./PetBondGraphPanel";
import {
  FactionSelect, GenderSelectField, PlayerAvatar, PlayerBadge, RoomInfoTagList, RoomTagList, Select, Stat, Toggle,
  defaultRoomInfoTagStyle, factionStyle, formatGiveawayValue, genderChoiceError, lobbyRoomInfoTags, nextGenderIdForFaction, punishmentTasks,
  roomInfoTagOrder, roomInfoTagStyle, roomStatusText, safePlayerStats
} from "./AppViews";

export type AdminSection = "site" | "factions" | "titles" | "punishments" | "roomTags" | "roomInfoTags" | "nameWar" | "giveaway" | "petBond" | "extremeMode" | "rankedScore" | "accessControl" | "messages" | "users" | "rooms";
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

// 按当前配置里实际出现过的任务分组去重分组阵营，用于称号池/惩罚池编辑器——
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
  const [activePunishmentId, setActivePunishmentId] = useState(config.punishments[0]?.id || "");
  const [punishmentSearch, setPunishmentSearch] = useState("");
  const [announcementMessage, setAnnouncementMessage] = useState("");
  const [announcementSeconds, setAnnouncementSeconds] = useState("8");
  const [activeRoomTab, setActiveRoomTab] = useState<AdminRoomTab>("rooms");
  const [playerFilters, setPlayerFilters] = useState<AdminPlayerFilters>(DEFAULT_ADMIN_PLAYER_FILTERS);
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
    setActivePunishmentId((old) => nextConfig.punishments.some((item) => item.id === old) ? old : nextConfig.punishments[0]?.id || "");
    setDirty(false);
    setServerConfigChanged(false);
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
    const uploadFile = await compressAdminImageForUpload(file);
    if (uploadFile.size > 8 * 1024 * 1024) throw new Error("图片超过 8MB，请换一张或先压缩");
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
    { id: "site", label: "网站信息", detail: draft.site.name },
    { id: "factions", label: "性别与阵营", detail: `${draft.genders.length} 个性别 · ${draft.genderFactions.length} 个阵营` },
    { id: "titles", label: "称号池", detail: `${draft.titles.length} 个段位` },
    { id: "punishments", label: "惩罚池", detail: `${draft.punishments.length} 项` },
    { id: "roomTags", label: "房间标签", detail: `${draft.roomTags.length} 个标签` },
    { id: "roomInfoTags", label: "房间信息标签", detail: "房间头部彩色标签" },
    { id: "nameWar", label: "名字争夺战", detail: draft.nameWar.penaltyPrefix },
    { id: "giveaway", label: "白给模式", detail: draft.giveaway.panelTitle },
    { id: "petBond", label: "宠物乐园", detail: draft.petBond?.panelTitle || "宠物乐园" },
    { id: "extremeMode", label: "极限模式", detail: `${draft.extremeMode.emoji} ${draft.extremeMode.label}` },
    { id: "rankedScore", label: "排位分设置", detail: (() => { const rs = withRankedScoreDefaults(draft.rankedScore); return `积分显示上下限`; })() },
    { id: "accessControl", label: "防多开", detail: (() => { const ac = withAccessControlDefaults(draft.accessControl); return ac.registrationDisabled ? "已禁止新用户注册" : `同指纹 ${ac.maxOnlinePerIp} 在线 / ${ac.maxCreatesPer10Min} 新建`; })() },
    { id: "messages", label: "提示公告", detail: `${Object.keys(draft.messages).length} 条文案` },
    { id: "users", label: "用户管理", detail: playerFilters.online ? `在线筛选 · ${adminPlayersTotal} 人` : `${adminPlayersTotal} 人` },
    { id: "rooms", label: "房间管理", detail: `${lobby.rooms.length} 房间 · 在线 ${lobby.onlineCount}` }
  ];

  const currentNav = navItems.find((item) => item.id === activeSection) || navItems[0];

  function switchSection(section: AdminSection) {
    setActiveSection(section);
  }

  function renderSection() {
    if (activeSection === "site") {
      return (
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
      return (
        <div className="config-section admin-section-card">
          <AdminSectionHeader title="性别预设" subtitle="系统预设性别池，玩家只能从中选择；同一显示文字可勾选多个阵营，阵营由所选性别查表决定" />
          <div className="faction-gender-list">
            {genderGroups.map((group) => {
              const label = group.label;
              return (
                <div className="mini-card faction-gender-card" key={group.key}>
                  <div className="config-row faction-gender-row">
                    <button
                      type="button"
                      className="danger-button tiny-danger-button icon-button faction-gender-delete"
                      title="删除该性别预设"
                      onClick={() => {
                        const nextGenders = draft.genders.filter((item) => item.label !== label);
                        if (nextGenders.length === 0) {
                          onError("至少需要保留 1 个性别预设");
                          return;
                        }
                        patchGenders(nextGenders);
                      }}
                    >
                      🗑
                    </button>
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
                                nextGenders = [...draft.genders, { id: genderId, label, factionId: f.id }];
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
            const genderId = nextAdminId("gender", draft.genders.map((item) => item.id));
            patchGenders([...draft.genders, { id: genderId, label: `新性别${index}`, factionId: draft.genderFactions[0]?.id || "" }]);
          }}>添加性别预设</button>

          <AdminSectionHeader title="阵营" subtitle="玩家选择的阵营分组，标签颜色和任务分组在这里配置。" />
          <div className="punishment-manager faction-manager">
            <aside className="punishment-index-panel">
              <input value={factionSearch} onChange={(event) => setFactionSearch(event.target.value)} placeholder="搜索阵营 / ID" />
              <div className="punishment-index-list">
                {filteredFactions.map((item) => (
                  <button className={item.id === faction?.id ? "active" : ""} key={item.id} onClick={() => setActiveFactionId(item.id)}>
                    <span>{item.label}</span>
                    <small>{item.id} · {taskGroupLabel(item.taskGroup)}</small>
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
                  <label className="field-label"><span>阵营 ID（自动生成，一般不用改）</span><input value={faction.id} onChange={(event) => { setActiveFactionId(event.target.value); patchFactions(draft.genderFactions.map((item, itemIndex) => itemIndex === factionIndex ? { ...item, id: event.target.value } : item)); }} placeholder="阵营ID" /></label>
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
      return (
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
      );
    }

    if (activeSection === "punishments") {
      const playerRoomNameItemId = "__player_room_names__";
      const filteredPunishments = draft.punishments.filter((punishment) => {
        const keyword = punishmentSearch.trim().toLowerCase();
        if (!keyword) return true;
        return `${punishment.id} ${punishment.name} ${punishment.description}`.toLowerCase().includes(keyword);
      });
      const isPlayerRoomNameSelected = activePunishmentId === playerRoomNameItemId;
      const selectedIndex = Math.max(0, draft.punishments.findIndex((punishment) => punishment.id === activePunishmentId));
      const punishment = draft.punishments[selectedIndex];
      const taskGroups = taskGroupsFromFactions(draft.genderFactions);
      return (
        <div className="config-section admin-section-card">
          <AdminSectionHeader title="惩罚池" subtitle="编辑不同阵营系统惩罚任务、房间名生成。系统任务文案可用 {loser}/{winner} 插入败者/胜者昵称。" />
          <div className="punishment-manager">
            <aside className="punishment-index-panel">
              <input value={punishmentSearch} onChange={(event) => setPunishmentSearch(event.target.value)} placeholder="搜索惩罚名称 / ID / 简介" />
              <div className="punishment-index-list">
                {filteredPunishments.map((item) => (
                  <button className={!isPlayerRoomNameSelected && item.id === punishment?.id ? "active" : ""} key={item.id} onClick={() => setActivePunishmentId(item.id)}>
                    <span>{item.name}</span>
                    <small>{item.id} · {punishmentTasks(item, draft).length} 个任务</small>
                  </button>
                ))}
                {filteredPunishments.length === 0 && <p className="empty">没有匹配的惩罚</p>}
              </div>
              <button className={`special-index-item ${isPlayerRoomNameSelected ? "active" : ""}`} onClick={() => setActivePunishmentId(playerRoomNameItemId)}>
                <span>玩家发布任务房名</span>
                <small>玩家发布模式 · {draft.playerPunishmentRoomNamePool?.subjects.length || 0} 个关键词</small>
              </button>
              <button onClick={() => {
                const nextId = nextAdminId("punish", draft.punishments.map((item) => item.id));
                setActivePunishmentId(nextId);
                patch({ punishments: [...draft.punishments, { id: nextId, name: "新惩罚", description: "写下惩罚说明", cardImageUrl: "", cardImageOpacity: 0.26, roomBackgroundImages: [], variants: Object.fromEntries(taskGroups.map((group) => [group.id, "写下这个分组专属任务"])), tasks: [{ id: "task1", name: "默认任务", backgroundImages: [], backgroundOpacity: 0.22, variants: Object.fromEntries(taskGroups.map((group) => [group.id, "写下这个分组专属任务"])) }], roomNamePool: defaultAdminRoomNamePool() }] });
              }}>添加惩罚</button>
            </aside>
            {isPlayerRoomNameSelected ? (
              <div className="mini-card punishment-detail-panel player-punishment-room-name-card">
                <div className="admin-card-title">
                  <strong>玩家发布任务模式房名词库</strong>
                  <small>示例：{sampleRoomName(draft.playerPunishmentRoomNamePool)}</small>
                </div>
                <p className="hint">创建房间选择“玩家发布”时，会用这里生成随机房间名。它不属于某一个系统惩罚，所以作为惩罚池里的特殊项目管理。</p>
                <RoomNamePoolEditor title="玩家发布任务随机房名词库" pool={draft.playerPunishmentRoomNamePool || defaultAdminRoomNamePool()} onChange={(playerPunishmentRoomNamePool) => patch({ playerPunishmentRoomNamePool })} />
              </div>
            ) : punishment && (
              <div className="mini-card punishment-detail-panel">
                <div className="admin-card-title">
                  <strong>{punishment.name}</strong>
                  <small>{selectedIndex + 1} / {draft.punishments.length} · {punishmentTasks(punishment, draft).length} 个任务 · 示例：{sampleRoomName(punishment.roomNamePool)}</small>
                </div>
                <div className="admin-danger-row">
                  <button
                    type="button"
                    className="danger-button"
                    onClick={() => {
                      if (draft.punishments.length <= 1) {
                        onError("至少需要保留 1 个惩罚池");
                        return;
                      }
                      if (!window.confirm(`确定删除整个惩罚池「${punishment.name}」吗？里面的任务也会一起删除。`)) return;
                      const nextPunishments = draft.punishments.filter((_, itemIndex) => itemIndex !== selectedIndex);
                      setActivePunishmentId(nextPunishments[Math.max(0, selectedIndex - 1)]?.id || nextPunishments[0]?.id || "");
                      patch({ punishments: nextPunishments });
                    }}
                  >
                    删除这个惩罚池
                  </button>
                </div>
                <div className="punishment-admin-preview">
                  <button
                    className="punishment-choice-card active"
                    style={{
                      "--punishment-bg": punishment.cardImageUrl ? `url(${punishment.cardImageUrl})` : "none",
                      "--punishment-bg-opacity": String(punishment.cardImageOpacity ?? 0.26)
                    } as CSSProperties}
                  >
                    <span>{punishment.name}</span>
                    <small>{punishment.description}</small>
                  </button>
                </div>
                <div className="config-row">
                  <label className="field-label"><span>内部 ID（自动生成，一般不用改）</span><input value={punishment.id} onChange={(event) => { setActivePunishmentId(event.target.value); patch({ punishments: draft.punishments.map((item, itemIndex) => itemIndex === selectedIndex ? { ...item, id: event.target.value } : item) }); }} placeholder="内部ID" /></label>
                  <label className="field-label"><span>玩家可见名称</span><input value={punishment.name} onChange={(event) => patch({ punishments: draft.punishments.map((item, itemIndex) => itemIndex === selectedIndex ? { ...item, name: event.target.value } : item) })} placeholder="惩罚名称" /></label>
                </div>
                <label className="field-label"><span>通用说明</span><textarea value={punishment.description} onChange={(event) => patch({ punishments: draft.punishments.map((item, itemIndex) => itemIndex === selectedIndex ? { ...item, description: event.target.value } : item) })} placeholder="惩罚说明" /></label>
                <label className="field-label">
                  <span>卡片背景图 URL（推荐 1200 × 480）</span>
                  <input value={punishment.cardImageUrl || ""} onChange={(event) => patch({ punishments: draft.punishments.map((item, itemIndex) => itemIndex === selectedIndex ? { ...item, cardImageUrl: event.target.value } : item) })} placeholder="例如 /uploads/example.webp 或 https://..." />
                </label>
                <AdminImageUpload label="上传为卡片背景图" upload={uploadAdminImage} onError={onError} onUploaded={(cardImageUrl) => patch({ punishments: draft.punishments.map((item, itemIndex) => itemIndex === selectedIndex ? { ...item, cardImageUrl } : item) })} />
                <label className="field-label">
                  <span>卡片背景透明率（推荐 0.15 ~ 0.45）</span>
                  <input type="number" min={0} max={1} step={0.01} value={punishment.cardImageOpacity ?? 0.26} onChange={(event) => patch({ punishments: draft.punishments.map((item, itemIndex) => itemIndex === selectedIndex ? { ...item, cardImageOpacity: Number(event.target.value) } : item) })} />
                </label>
                <TagListEditor label="房间信息卡图库（推荐 1920 × 1080，jpg/webp；用于大厅房间卡和房间内信息卡，手机端会居中裁切）" placeholder="输入图片 URL 后回车" values={punishment.roomBackgroundImages || []} onChange={(roomBackgroundImages) => patch({ punishments: draft.punishments.map((item, itemIndex) => itemIndex === selectedIndex ? { ...item, roomBackgroundImages } : item) })} />
                <AdminImageUpload label="上传并加入房间信息卡图库" upload={uploadAdminImage} onError={onError} onUploaded={(imageUrl) => patch({ punishments: draft.punishments.map((item, itemIndex) => itemIndex === selectedIndex ? { ...item, roomBackgroundImages: [...(item.roomBackgroundImages || []), imageUrl] } : item) })} />
                <div className="punishment-task-list">
                  {punishmentTasks(punishment, draft).map((task, taskIndex) => (
                    <details className="mini-card punishment-task-editor" key={task.id} open={taskIndex === 0}>
                      <summary>
                        <strong>{task.name}</strong>
                        <small>{taskGroups.length} 个分组版本</small>
                        <button
                          type="button"
                          className="danger-button tiny-danger-button"
                          onClick={(event) => {
                            event.preventDefault();
                            event.stopPropagation();
                            if (!window.confirm(`确定删除任务「${task.name}」吗？`)) return;
                            const currentTasks = punishmentTasks(punishment, draft);
                            const nextTasks = currentTasks.filter((_, itemIndex) => itemIndex !== taskIndex);
                            patch({
                              punishments: draft.punishments.map((item, itemIndex) => itemIndex === selectedIndex
                                ? { ...item, tasks: nextTasks.length ? nextTasks : [newPunishmentTask(draft, item)] }
                                : item)
                            });
                          }}
                        >
                          删除任务
                        </button>
                      </summary>
                      <div className="config-row">
                        <label className="field-label"><span>任务 ID（自动生成，一般不用改）</span><input value={task.id} onChange={(event) => patchPunishmentTask(patch, draft, selectedIndex, taskIndex, { ...task, id: event.target.value })} placeholder="任务ID" /></label>
                        <label className="field-label"><span>任务名称</span><input value={task.name} onChange={(event) => patchPunishmentTask(patch, draft, selectedIndex, taskIndex, { ...task, name: event.target.value })} placeholder="任务名称" /></label>
                      </div>
                      <div className="config-row">
                        <label className="field-label">
                          <span>任务背景透明率（推荐 0.15 ~ 0.4）</span>
                          <input type="number" min={0} max={1} step={0.01} value={task.backgroundOpacity ?? 0.22} onChange={(event) => patchPunishmentTask(patch, draft, selectedIndex, taskIndex, { ...task, backgroundOpacity: Number(event.target.value) })} />
                        </label>
                      </div>
                      <TagListEditor label="任务背景图库（推荐 1200 × 520）" placeholder="输入图片 URL 后回车" values={task.backgroundImages || []} onChange={(backgroundImages) => patchPunishmentTask(patch, draft, selectedIndex, taskIndex, { ...task, backgroundImages })} />
                      <AdminImageUpload label="上传并加入任务背景图库" upload={uploadAdminImage} onError={onError} onUploaded={(imageUrl) => patchPunishmentTask(patch, draft, selectedIndex, taskIndex, { ...task, backgroundImages: [...(task.backgroundImages || []), imageUrl] })} />
                      <div className="variant-grid">
                        {taskGroups.map((group) => (
                          <label key={`${punishment.id}-${task.id}-${group.id}`}>
                            <span>{group.label}任务版本（{group.memberLabel}）</span>
                            <textarea value={task.variants?.[group.id] || ""} onChange={(event) => patchPunishmentTask(patch, draft, selectedIndex, taskIndex, { ...task, variants: { ...(task.variants || {}), [group.id]: event.target.value } })} placeholder={`系统任务·${group.label}，可用 {loser}/{winner}，如：{loser} 需要拥抱 {winner}`} />
                          </label>
                        ))}
                      </div>
                    </details>
                  ))}
                </div>
                <button onClick={() => patch({ punishments: draft.punishments.map((item, itemIndex) => itemIndex === selectedIndex ? { ...item, tasks: [...punishmentTasks(item, draft), newPunishmentTask(draft, item)] } : item) })}>给这个惩罚添加任务</button>
                <RoomNamePoolEditor title="随机房名词库" pool={punishment.roomNamePool || emptyRoomNamePool()} onChange={(roomNamePool) => patch({ punishments: draft.punishments.map((item, itemIndex) => itemIndex === selectedIndex ? { ...item, roomNamePool } : item) })} />
              </div>
            )}
          </div>
        </div>
      );
    }

    if (activeSection === "nameWar") {
      const preview = `${draft.nameWar.penaltyPrefix || "失名者"}-A7K2`;
      return (
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
          </div>
          <p className="hint">随机码固定为 4 位大写字母/数字；已有惩罚名不会因为你改前缀立刻变化，新触发的玩家会使用新前缀。失格线按数据库真实排位分判定，与展示封顶无关。</p>
        </div>
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
          <div className="config-row">
            <label className="field-label">
              <span>点赞每小时次数上限</span>
              <input type="number" min={1} step={1} value={draft.giveaway.likeVoteLimitPerHour} onChange={(event) => patch({ giveaway: { ...draft.giveaway, likeVoteLimitPerHour: Number(event.target.value) } })} />
            </label>
            <label className="field-label">
              <span>点赞降低值 (%)</span>
              <input type="number" min={0.1} max={100} step={0.1} value={draft.giveaway.likeVoteValue} onChange={(event) => patch({ giveaway: { ...draft.giveaway, likeVoteValue: Number(event.target.value) } })} />
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

    if (activeSection === "extremeMode") {
      const extreme = draft.extremeMode;
      const patchExtreme = (nextExtreme: AppConfig["extremeMode"]) => patch({ extremeMode: nextExtreme });
      return (
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
      );
    }

    if (activeSection === "rankedScore") {
      // 与防多开 accessControl 相同：缺对象时用默认合并，避免 draft.rankedScore 为 undefined 时白屏。
      const rankedScore = withRankedScoreDefaults(draft.rankedScore);
      const patchRankedScore = (next: Partial<AppConfig["rankedScore"]>) =>
        patch({ rankedScore: withRankedScoreDefaults({ ...rankedScore, ...next }) });
      return (
        <div className="config-section admin-section-card">
          <AdminSectionHeader title="排位分设置" subtitle="控制排行榜/个人资料等展示时的封顶值，及每日衰减比例。数据库中的存储值无限制；" />
          <div className="config-row">
            <label className="field-label"><span>展示上限</span><input type="number" min={1} value={rankedScore.max} onChange={(event) => patchRankedScore({ max: Number(event.target.value) })} /></label>
            <label className="field-label"><span>普通玩家展示下限</span><input type="number" max={-1} value={rankedScore.min} onChange={(event) => patchRankedScore({ min: Number(event.target.value) })} /></label>
            <label className="field-label"><span>名字争夺战展示下限</span><input type="number" value={rankedScore.nameWarMin} onChange={(event) => patchRankedScore({ nameWarMin: Number(event.target.value) })} /></label>
            <label className="field-label"><span>每日衰减比例</span><input type="number" min={0.01} max={1} step={0.01} value={rankedScore.dailyDecayRatio} onChange={(event) => patchRankedScore({ dailyDecayRatio: Number(event.target.value) })} /></label>
          </div>
        </div>
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
        <div className="config-section admin-section-card">
          <AdminSectionHeader title="房间标签" subtitle="玩家创建房间时，可以开启并选择这里配置好的标签。最多显示 5 个。" />
          <div className="admin-preview-card">
            <span>预览</span>
            <RoomTagList tags={draft.roomTags.slice(0, 5)} />
            <p>点下面标签可以删除；在输入框里输入文字后回车可以添加。</p>
          </div>
          <TagListEditor label="房间 Tag 池" placeholder="输入房间 Tag 后回车" values={draft.roomTags} onChange={(roomTags) => patch({ roomTags })} />
        </div>
      );
    }

    if (activeSection === "roomInfoTags") {
      return (
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
      );
    }

    if (activeSection === "messages") {
      return (
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
      );
    }

    if (activeSection === "users") {
      const listHint = `匹配过滤器 · 在线 ${adminFilterOnlineCount} / 离线 ${adminFilterOfflineCount}`;
      return (
        <div className="config-section admin-section-card">
          <AdminSectionHeader title="用户管理" subtitle="按过滤器查询玩家档案，协助改资料、踢出与找回认领密钥。" />
          <div className="admin-player-filters" role="group" aria-label="用户列表过滤器">
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
                  : adminPlayersTruncated
                    ? `显示 ${adminPlayers.length} / 共 ${adminPlayersTotal} 人（已截断）· ${listHint}`
                    : `${adminPlayersTotal} 人 · ${listHint}`}
              </span>
            </div>
            {adminPlayers.map((player) => (
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
            {!adminPlayersLoading && adminPlayers.length === 0 && (
              <p className="empty">当前没有符合条件的玩家</p>
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
          {activeRoomTab === "announcement" && (
            <div className="admin-action-row">
              <button className="danger-button" onClick={() => action("clearLobbyChat")}>清空大厅聊天</button>
            </div>
          )}
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
                      <button onClick={() => action("clearRoomChat", { roomId: room.id })}>清空房间聊天</button>
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
            {activeSection !== "users" && activeSection !== "rooms" && (
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

export function newPunishmentTask(draft: AppConfig, punishment: AppConfig["punishments"][number]): PunishmentTaskConfig {
  const tasks = punishmentTasks(punishment, draft);
  const nextIndex = tasks.length + 1;
  return {
    id: nextAdminId("task", tasks.map((task) => task.id)),
    name: `任务 ${nextIndex}`,
    backgroundImages: [],
    backgroundOpacity: 0.22,
    variants: Object.fromEntries(taskGroupsFromFactions(draft.genderFactions).map((group) => [group.id, "写下这个分组专属任务"]))
  };
}

export function patchPunishmentTask(patch: (next: Partial<AppConfig>) => void, draft: AppConfig, punishmentIndex: number, taskIndex: number, nextTask: PunishmentTaskConfig) {
  patch({
    punishments: draft.punishments.map((punishment, currentPunishmentIndex) => {
      if (currentPunishmentIndex !== punishmentIndex) return punishment;
      return {
        ...punishment,
        tasks: punishmentTasks(punishment, draft).map((task, currentTaskIndex) => currentTaskIndex === taskIndex ? nextTask : task)
      };
    })
  });
}

export function AdminSectionHeader({ title, subtitle }: { title: string; subtitle: string }) {
  return (
    <div className="admin-section-header">
      <h3>{title}</h3>
      <p className="hint">{subtitle}</p>
    </div>
  );
}

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
        <button className="danger-button" onClick={onKick}>踢出</button>
      </div>
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

export function AdminImageUpload({ label, upload, onUploaded, onError }: { label: string; upload: (file: File) => Promise<string>; onUploaded: (imageUrl: string) => void; onError: (message: string) => void }) {
  return (
    <label className="admin-image-upload">
      <Upload size={15} /> {label}
      <input
        type="file"
        accept="image/png,image/jpeg,image/webp"
        onChange={(event) => {
          const file = event.target.files?.[0];
          event.target.value = "";
          if (!file) return;
          upload(file).then(onUploaded).catch((error) => onError(error instanceof Error ? error.message : "上传失败"));
        }}
      />
    </label>
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
