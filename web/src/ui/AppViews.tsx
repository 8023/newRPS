import { type CSSProperties, type KeyboardEvent as ReactKeyboardEvent, type MutableRefObject, type ReactNode, type UIEvent as ReactUIEvent, useEffect, useMemo, useRef, useState } from "react";
import { ChevronDown, Coffee, Crown, DoorOpen, ExternalLink, Eye, HeartHandshake, Info, Moon, Pencil, Save, Send, Shield, Sun, Swords, Upload, UserRound, Users, BookOpen } from "lucide-react";
import type {
  AppConfig, ChatMessage, GenderColors, GenderFaction, LobbySnapshot, Move, PetBondState, PublicPlayer,
  PunishmentTaskConfig, RoomInfoTagStyle, RoomNamePool, RoomSettings, RoomSnapshot, RoundResult, SeatKey, SeatOccupant
} from "../shared/types";
import { DEFAULT_NAME_WAR_PENALTY_THRESHOLD, DEFAULT_NAME_WAR_RENAME_MIN_POINTS, withPetBondDefaults, withRankedScoreDefaults } from "../lib/normalize";
import { socket } from "../ws";
import {
  defaultGomokuRoomName, defaultJungleRoomName, defaultLiarsDiceRoomName, defaultOthelloRoomName, defaultRoomName, defaultTicTacToeRoomName,
  gameMinutesOptions, gomokuBoardThemes, jungleBoardThemes, moveSecondsOptions, othelloBoardThemes, playerSecretKey, doumiaoLinks, luv4uLinks, tictactoeBoardThemes, tokenKey
} from "../lib/constants";
import { ask } from "../lib/rpc";
import { cacheJoinProfile, claimIdentity, clearPlayerIdentity, encodeClaimCode, fetchClaimKey, joinIdentityPayload, logout, refreshClaimKey } from "../lib/session";
import { appendHistoryPage, normalizeRoomSnapshot, normalizeRoundHistoryItem } from "../lib/normalize";
import { prepareProofImageForUpload } from "../lib/proofImage";
import { prepareAvatarImageForUpload } from "../lib/avatarImage";
import { isNearScrollBottom, scrollToBottomSoon, stickChatToBottom, useMobileCollapse } from "../lib/uiHelpers";
import { appendMentionText, loadChat, loadOlderChat, useChat } from "../lib/chatStore";
import {
  disablePushSubscription, ensurePushSubscription, fetchPushPreferences,
  requestNotificationPermission, updatePushPreferences, type PushPreferences
} from "../lib/pushNotify";
import type { MeState } from "../lib/types";
import { LiarsDicePanel } from "./LiarsDicePanel";
import { GomokuPanel, GomokuScore } from "./GomokuPanel";
import { JunglePanel, JungleScore, jungleSideLabel } from "./JunglePanel";

import { formatDuration, formatOnlineDuration } from "../lib/format";

// 仅手机端生效的模块折叠开关（三角图标）；桌面端由 CSS 隐藏按钮并强制展开，不受折叠状态影响。
// collapsed/onToggle 由调用方通过 useMobileCollapse(sectionKey) 持有，与被折叠的内容共享同一份状态。
export function CollapseToggle({ collapsed, onToggle, label }: { collapsed: boolean; onToggle: () => void; label: string }) {
  return (
    <button
      type="button"
      className={`mobile-collapse-toggle ${collapsed ? "collapsed" : ""}`}
      onClick={onToggle}
      aria-expanded={!collapsed}
      aria-label={collapsed ? `展开${label}` : `收起${label}`}
      title={collapsed ? `展开${label}` : `收起${label}`}
    >
      <ChevronDown size={16} />
    </button>
  );
}

export function Login({ config, onDone, onError }: { config: AppConfig; onDone: (me: MeState) => void; onError: (message: string) => void }) {
  const [name, setName] = useState("");
  const [factionId, setFactionId] = useState(firstFactionId(config));
  const [genderId, setGenderId] = useState(() => firstGenderId(config, firstFactionId(config)));
  const [mode, setMode] = useState<"new" | "restore">("new");
  const [restoreCode, setRestoreCode] = useState("");
  const [restoreBusy, setRestoreBusy] = useState(false);
  const [pendingJoinPayload, setPendingJoinPayload] = useState<Record<string, unknown> | null>(null);

  async function doJoin(payload: Record<string, unknown>) {
    const result = await ask<MeState & { alreadyOnline?: true }>("player:join", payload);
    if (result.alreadyOnline) {
      setPendingJoinPayload(payload);
      return;
    }
    localStorage.setItem(tokenKey, result.token);
    // 以服务端确认后的资料为准缓存。
    if (result.player) cacheJoinProfile(result.player);
    if (result.reissuedSecret) localStorage.setItem(playerSecretKey, result.reissuedSecret);
    onDone(result);
  }

  async function submit() {
    const genderError = genderChoiceError(config, genderId);
    if (genderError) {
      onError(genderError);
      return;
    }
    try {
      await doJoin({ name, genderId, token: localStorage.getItem(tokenKey), ...(await joinIdentityPayload()) });
    } catch (error) {
      const message = error instanceof Error ? error.message : "进入失败";
      if (message === "玩家身份校验失败") {
        // 本地缓存的 playerId/playerSecret 服务端已经不认（常见于老账号未完成迁移）：
        // 清掉本地身份重试一次，等价于自动执行"清 localStorage 重新注册"。
        clearPlayerIdentity();
        try {
          await doJoin({ name, genderId, token: localStorage.getItem(tokenKey), ...(await joinIdentityPayload()) });
          return;
        } catch (retryError) {
          onError(retryError instanceof Error ? retryError.message : "进入失败");
          return;
        }
      }
      onError(message);
    }
  }

  async function submitRestore() {
    if (!restoreCode.trim()) {
      onError("请输入认领密钥");
      return;
    }
    setRestoreBusy(true);
    try {
      const claimed = await claimIdentity(restoreCode);
      await doJoin({
        name: claimed.name, genderId: claimed.genderId,
        token: localStorage.getItem(tokenKey), ...(await joinIdentityPayload())
      });
    } catch (error) {
      onError(error instanceof Error ? error.message : "认领失败");
    } finally {
      setRestoreBusy(false);
    }
  }

  async function confirmKick() {
    if (!pendingJoinPayload) return;
    try {
      await doJoin({ ...pendingJoinPayload, forceKick: true });
    } catch (error) {
      onError(error instanceof Error ? error.message : "进入失败");
    } finally {
      setPendingJoinPayload(null);
    }
  }

  if (pendingJoinPayload) {
    return (
      <section className="login-card kick-confirm-card">
        <h2>该账号已在其他设备登录</h2>
        <p className="hint">继续登录会把另一台设备顶下线（那边会收到提示）。确定要继续吗？</p>
        <div className="kick-confirm-actions">
          <button onClick={() => setPendingJoinPayload(null)}>取消</button>
          <button className="primary" onClick={confirmKick}>确定顶替登录</button>
        </div>
      </section>
    );
  }

  return (
    <section className="login-card">
      <h2>进入游戏</h2>
      {mode === "new" ? (
        <>
          <input value={name} onChange={(event) => setName(event.target.value)} maxLength={12} placeholder="你的名字，允许重复" />
          <FactionSelect
            config={config}
            factionId={factionId}
            onFactionChange={(nextFactionId) => {
              setFactionId(nextFactionId);
              setGenderId((old) => nextGenderIdForFaction(config, nextFactionId, old));
            }}
          />
          <GenderSelect
            config={config}
            genderId={genderId}
            factionId={factionId}
            onGenderChange={setGenderId}
          />
          <button className="primary" onClick={submit}>进入大厅</button>
          <button className="link-button" onClick={() => setMode("restore")}>已有账号？用认领密钥恢复</button>
        </>
      ) : (
        <>
          <p className="hint">在另一台设备的「个人设置」里获取认领密钥，粘贴到这里即可恢复该账号。</p>
          <input value={restoreCode} onChange={(event) => setRestoreCode(event.target.value)} placeholder="粘贴认领密钥" />
          <button className="primary" disabled={restoreBusy} onClick={submitRestore}>{restoreBusy ? "恢复中…" : "恢复账号"}</button>
          <button className="link-button" onClick={() => setMode("new")}>返回新建账号</button>
        </>
      )}
    </section>
  );
}

/** 认领密钥展示区：个人资料页 / 登出确认弹窗共用。 */
export function ClaimKeyPanel({ onError }: { onError: (message: string) => void }) {
  const [code, setCode] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [copied, setCopied] = useState(false);

  async function load() {
    setLoading(true);
    try {
      const result = await fetchClaimKey();
      setCode(encodeClaimCode(result.playerId, result.claimKey));
    } catch (error) {
      onError(error instanceof Error ? error.message : "获取认领密钥失败");
    } finally {
      setLoading(false);
    }
  }

  async function refresh() {
    setLoading(true);
    try {
      const result = await refreshClaimKey();
      setCode(encodeClaimCode(result.playerId, result.claimKey));
      setCopied(false);
    } catch (error) {
      onError(error instanceof Error ? error.message : "刷新认领密钥失败");
    } finally {
      setLoading(false);
    }
  }

  async function copy() {
    if (!code) return;
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      onError("复制失败，请手动选中密钥");
    }
  }

  return (
    <div className="claim-key-panel">
      <p className="hint">认领密钥用于在另一台设备登录同一个账号，粘贴一次就会失效，请勿分享给他人。</p>
      {code ? (
        <>
          <code className="claim-key-code">{code}</code>
          <div className="claim-key-actions">
            <button type="button" onClick={copy}>{copied ? "已复制" : "复制"}</button>
            <button type="button" disabled={loading} onClick={refresh}>刷新（旧密钥立即作废）</button>
          </div>
        </>
      ) : (
        <button type="button" disabled={loading} onClick={load}>{loading ? "获取中…" : "显示认领密钥"}</button>
      )}
    </div>
  );
}

// PlayerAvatar：房间参战席、个人设置等处展示的圆形头像；未设置头像时退回姓名首字。
export function PlayerAvatar({ player, size = 28, className = "" }: { player?: PublicPlayer; size?: number; className?: string }) {
  const label = mentionLabel(player) || player?.name || "";
  const ch = label.slice(0, 1) || "?";
  const style = {
    width: size,
    height: size,
    fontSize: Math.max(10, Math.round(size * 0.42)),
    lineHeight: 1
  };
  if (player?.avatarUrl) {
    // draggable=false：避免用作力导向图节点等可拖拽场景时，浏览器把它当成图片触发原生的
    // "拖拽另存为/打开"效果而不是我们自己的拖拽逻辑。
    return <img className={`player-avatar ${className}`} style={style} src={player.avatarUrl} alt="" draggable={false} onDragStart={(event) => event.preventDefault()} />;
  }
  return (
    <span className={`player-avatar player-avatar-fallback ${className}`} style={style} aria-hidden="true">
      <span className="player-avatar-char">{ch}</span>
    </span>
  );
}

export function PlayerBadge({ player, compact = false }: { player: PublicPlayer; compact?: boolean }) {
  const stats = safePlayerStats(player);
  const punished = Boolean(player.nameWarPunished && player.nameWarPenaltyName);
  return (
    <span className={`player-badge ${compact ? "compact" : ""}`}>
      <span className="gender-chip" style={genderStyle(player)} title={player.factionLabel}>{player.genderLabel}</span>
      <span className="title-chip" style={titleStyle(stats.titleColors)}>{stats.title}</span>
      <strong className={punished ? "name-war-pill" : ""}>{displayPlayerName(player)}</strong>
      <ModeChip player={player} />
      <GiveawayChip player={player} />
    </span>
  );
}

export function ModeChip({ player }: { player: PublicPlayer }) {
  const showNameWar = Boolean(player.nameWarEnabled && !player.nameWarPunished);
  if (player.extremeModeEnabled && showNameWar) return <span className="mode-chip">⚡⚔️ 极限名争</span>;
  if (player.extremeModeEnabled) return <span className="mode-chip">⚡ 极限</span>;
  if (showNameWar) return <span className="mode-chip">⚔️ 名争</span>;
  return null;
}

export function shouldShowGiveawayValue(player: PublicPlayer) {
  return Boolean(player.giveawayEnabled || (player.giveawayValue || 0) > 0);
}

export function GiveawayChip({ player }: { player: PublicPlayer }) {
  if (!shouldShowGiveawayValue(player)) return null;
  return <span className="giveaway-chip">白给 {formatGiveawayValue(player.giveawayValue || 0)}%</span>;
}

export function displayPlayerName(player: PublicPlayer) {
  if (player.nameWarPunished && player.nameWarPenaltyName) return `${player.extremeModeEnabled ? "极 " : ""}${player.nameWarPenaltyName}`;
  return player.name;
}

// 按阵营过滤可选性别：性别预设未设归属阵营（factionId 为空）时视为不限阵营都能选，
// 与后端 validGenderSubmission 的兼容口径保持一致。
export function gendersForFaction(config: AppConfig, factionId: string) {
  return config.genders.filter((gender) => !gender.factionId || gender.factionId === factionId);
}

export function firstFactionId(config: AppConfig) {
  return config.genderFactions[0]?.id || "";
}

export function firstGenderId(config: AppConfig, factionId: string) {
  return gendersForFaction(config, factionId)[0]?.id || "";
}

// 切换阵营时性别是否需要跟着重置：当前性别只有在还落在新阵营的可选池内才保留，
// 否则退回新阵营的第一个预设。
export function nextGenderIdForFaction(config: AppConfig, factionId: string, currentGenderId: string) {
  if (!currentGenderId) return currentGenderId;
  const pool = gendersForFaction(config, factionId);
  if (pool.some((gender) => gender.id === currentGenderId)) return currentGenderId;
  return pool[0]?.id || "";
}

export function genderStyle(player: PublicPlayer): CSSProperties {
  return {
    color: player.factionColors.textColor,
    backgroundColor: player.factionColors.backgroundColor,
    borderColor: player.factionColors.borderColor
  };
}

// 称号标签底色按赋予来源着色，服务端已解析好（见 PublicStats.titleColors），
// 这里兜底一份中性灰以防旧缓存数据缺该字段。
export const DEFAULT_TITLE_COLORS: GenderColors = { textColor: "#4d5c6f", backgroundColor: "#eef3f8", borderColor: "#c9d6e4" };

export function titleStyle(colors: GenderColors | null | undefined): CSSProperties {
  const c = colors || DEFAULT_TITLE_COLORS;
  return {
    color: c.textColor,
    backgroundColor: c.backgroundColor,
    borderColor: c.borderColor
  };
}

export function factionStyle(faction: GenderFaction): CSSProperties {
  return {
    color: faction.textColor,
    backgroundColor: faction.backgroundColor,
    borderColor: faction.borderColor
  };
}

// 阵营是独立下拉 5 选 1，不套用阵营颜色（颜色只用于玩家名字前的药丸标签）。
export function FactionSelect({ config, factionId, onFactionChange }: { config: AppConfig; factionId: string; onFactionChange: (factionId: string) => void }) {
  // 和 GenderSelectControl 一样：当前 factionId 若不在配置里（比如阵营被后台改名/删除后，
  // 页面还留着旧 id），补一个占位 option，避免原生 <select> 因 value 找不到匹配 <option>
  // 而静默显示成列表第一项，让人误以为已经选中了别的阵营、保存时却提交了这个失效的旧 id。
  const hasMatch = config.genderFactions.some((faction) => faction.id === factionId);
  return (
    <label className="field-label">
      <span>阵营</span>
      <select value={factionId} onChange={(event) => onFactionChange(event.target.value)}>
        {!hasMatch && factionId && <option value={factionId}>（未知阵营：{factionId}）</option>}
        {config.genderFactions.map((faction) => (
          <option key={faction.id} value={faction.id}>{faction.label}</option>
        ))}
      </select>
    </label>
  );
}

// 与后端 validGenderSubmission 保持一致：genderId 必须命中某个预设，阵营由该预设的
// factionId 查表决定，不再单独提交/校验。返回空字符串表示校验通过，否则返回给用户看的原因。
export function genderChoiceError(config: AppConfig, genderId: string) {
  if (!config.genders.some((gender) => gender.id === genderId)) return "所选性别不存在";
  return "";
}

export function isValidGenderChoice(config: AppConfig, genderId: string) {
  return genderChoiceError(config, genderId) === "";
}

// 性别下拉本体，供 GenderSelect（登录/个人资料/后台）共用。选项按 factionId 过滤；若当前
// genderId 不在过滤后的池子里（比如阵营调整前的历史数据），额外把它补回选项列表，避免原生
// <select> 因 value 找不到匹配 <option> 而静默显示错乱。
function GenderSelectControl({ config, genderId, factionId, onGenderChange }: {
  config: AppConfig;
  genderId: string;
  factionId: string;
  onGenderChange: (genderId: string) => void;
}) {
  const pool = gendersForFaction(config, factionId);
  const options = genderId && !pool.some((gender) => gender.id === genderId)
    ? [...pool, ...config.genders.filter((gender) => gender.id === genderId)]
    : pool;
  return (
    <select value={genderId} onChange={(event) => onGenderChange(event.target.value)}>
      {options.map((gender) => (
        <option key={gender.id} value={gender.id}>{gender.label}</option>
      ))}
    </select>
  );
}

// 性别：查表法预设下拉，候选池按已选阵营过滤。用原生 <select>——Safari 下 <input list>
// 组合框弹出列表的配色不跟随页面暗色主题，会出现白底白字看不清的问题，<select> 原生下拉正常。
export function GenderSelect({ config, genderId, factionId, onGenderChange }: {
  config: AppConfig;
  genderId: string;
  factionId: string;
  onGenderChange: (genderId: string) => void;
}) {
  return (
    <label className="field-label">
      <span>性别</span>
      <GenderSelectControl config={config} genderId={genderId} factionId={factionId} onGenderChange={onGenderChange} />
    </label>
  );
}

// 后台用：与 GenderSelect 完全一样，保留独立导出名字是为了不打乱调用点（三栏布局曾经把
// 性别单独占一栏，现在阵营/性别各占一栏，行为上两者已无差异）。
export const GenderSelectField = GenderSelect;

/** 展示用战绩：缺省/非数字按 0，称号空则「暂无称号」 */
/** 排行榜排序用真实分（有 sort* 用 sort*，否则退回展示分）。 */
export function sortRankedPointsOf(player: PublicPlayer) {
  const s = player?.stats;
  if (!s) return 0;
  return Number.isFinite(Number(s.sortRankedPoints)) ? Number(s.sortRankedPoints) : Number(s.rankedPoints) || 0;
}
export function sortHighestScoreOf(player: PublicPlayer) {
  const s = player?.stats;
  if (!s) return 0;
  return Number.isFinite(Number(s.sortHighestScore)) ? Number(s.sortHighestScore) : Number(s.highestScore) || 0;
}
export function sortLowestScoreOf(player: PublicPlayer) {
  const s = player?.stats;
  if (!s) return 0;
  return Number.isFinite(Number(s.sortLowestScore)) ? Number(s.sortLowestScore) : Number(s.lowestScore) || 0;
}

export function safePlayerStats(player: PublicPlayer | null | undefined) {
  const s = player?.stats || ({} as PublicPlayer["stats"]);
  const title = typeof s.title === "string" ? s.title.trim() : "";
  const rankedPoints = Number.isFinite(Number(s.rankedPoints)) ? Number(s.rankedPoints) : 0;
  const highestScore = Number.isFinite(Number(s.highestScore)) ? Number(s.highestScore) : 0;
  const lowestScore = Number.isFinite(Number(s.lowestScore)) ? Number(s.lowestScore) : 0;
  return {
    wins: Number.isFinite(Number(s.wins)) ? Number(s.wins) : 0,
    losses: Number.isFinite(Number(s.losses)) ? Number(s.losses) : 0,
    draws: Number.isFinite(Number(s.draws)) ? Number(s.draws) : 0,
    punishments: Number.isFinite(Number(s.punishments)) ? Number(s.punishments) : 0,
    rankedPoints,
    highestScore,
    lowestScore,
    sortRankedPoints: Number.isFinite(Number(s.sortRankedPoints)) ? Number(s.sortRankedPoints) : rankedPoints,
    sortHighestScore: Number.isFinite(Number(s.sortHighestScore)) ? Number(s.sortHighestScore) : highestScore,
    sortLowestScore: Number.isFinite(Number(s.sortLowestScore)) ? Number(s.sortLowestScore) : lowestScore,
    title: title || "暂无称号",
    titleCustom: !!s.titleCustom,
    titleColors: s.titleColors && typeof s.titleColors === "object" ? s.titleColors : DEFAULT_TITLE_COLORS,
    totalOnlineMs: Number.isFinite(Number(s.totalOnlineMs)) ? Number(s.totalOnlineMs) : 0
  };
}

/** 排行榜展示用累计在线时长（毫秒）。 */
export function totalOnlineMsOf(player: PublicPlayer) {
  return safePlayerStats(player).totalOnlineMs || 0;
}

export function Lobby({ config, lobby, me, onError, onGoRoom }: { config: AppConfig; lobby: LobbySnapshot; me: PublicPlayer; onError: (message: string) => void; onGoRoom: (room?: RoomSnapshot) => void }) {
  const [showCreate, setShowCreate] = useState(false);
  const [passwords, setPasswords] = useState<Record<string, string>>({});
  const [now, setNow] = useState(Date.now());
  const renameTargets = lobby.players.filter((player) => isRenameTargetVisible(player, now));

  useEffect(() => {
    if (!lobby.players.some((player) => !player.connected && isRenameTarget(player) && player.disconnectedAt)) return;
    const timer = window.setInterval(() => setNow(Date.now()), 60_000);
    return () => window.clearInterval(timer);
  }, [lobby.players]);

  async function joinRoom(roomId: string) {
    try {
      const targetRoom = lobby.rooms.find((room) => room.id === roomId);
      if (targetRoom?.enableRanked && Boolean(targetRoom.enableExtremeRanked) !== Boolean(me.extremeModeEnabled)) {
        const ok = window.confirm(me.extremeModeEnabled
          ? "你是极限模式玩家，进入普通排位房后只能在观战席，不能上桌。确认进入？"
          : "你不是极限模式玩家，进入极限排位房后只能在观战席，不能上桌。确认进入？");
        if (!ok) return;
      }
      if (targetRoom?.enableExtremeRanked && me.extremeModeEnabled) {
        const ok = window.confirm(`这是极限排位房，胜负会按极限模式规则结算，并存在连胜风险，确认进入？`);
        if (!ok) return;
      }
      if (targetRoom?.enableRanked && (targetRoom.rankMultiplier || 1) > 1) {
        const multiplier = targetRoom.rankMultiplier || 1;
        const effectiveStake = targetRoom.stake * multiplier;
        const ok = window.confirm(targetRoom.gameId === "othello"
          ? `这是 ${multiplier} 倍黑白棋排位房，每翻 1 子按 ${effectiveStake} 分实时结算，确认进入？`
          : `这是 ${multiplier} 倍排位房，本局胜负按 ${effectiveStake} 分结算，确认进入？`);
        if (!ok) return;
      }
      const result = await ask<{ room: RoomSnapshot }>("room:join", { roomId, password: passwords[roomId] });
      onGoRoom(normalizeRoomSnapshot(result.room));
    } catch (error) {
      onError(error instanceof Error ? error.message : "加入失败");
    }
  }


  return (
    <section className="dashboard">
      <div className="panel lobby-main">
        <div className="panel-title lobby-title">
          <h2><Users size={20} /> 大厅</h2>
          <button type="button" className="primary small" onClick={() => setShowCreate((value) => !value)}>创建房间</button>
        </div>
        <div className="room-list">
          {lobby.rooms.map((room) => (
            <div
              className={`room-card ${room.roomBackgroundImage ? "has-room-card-background" : ""}`}
              key={room.id}
              style={room.roomBackgroundImage ? { "--room-card-bg": `url(${room.roomBackgroundImage})` } as CSSProperties : undefined}
            >
              <div>
                <h3>{room.name} <ExtremeRankedBadge enabled={room.enableExtremeRanked} /> <RankMultiplierBadge multiplier={room.rankMultiplier} /></h3>
                {room.tags?.length ? <RoomTagList tags={room.tags} /> : null}
                <RoomVersusLine room={room} />
                <p>{roomStatusText(room.status)} · {room.gameId === "liarsdice" ? `${room.players} 人参战` : `${room.players}/2 战斗席`} · {room.spectators} 观战</p>
                <RoomInfoTagList tags={lobbyRoomInfoTags(config, room)} />
              </div>
              <div className="join-box">
                {room.hasPassword && <input placeholder="房间密码" value={passwords[room.id] || ""} onChange={(event) => setPasswords({ ...passwords, [room.id]: event.target.value })} />}
                <button onClick={() => joinRoom(room.id)}>加入</button>
              </div>
            </div>
          ))}
          {lobby.rooms.length === 0 && <p className="empty">还没有房间，先创建一个吧。</p>}
        </div>
        <div className="lobby-lower-grid">
          <div className="lobby-lower-col lobby-lower-col-left">
            <PetBondPanel config={config} me={me} lobby={lobby} onError={onError} />
          </div>
          <div className="lobby-lower-col lobby-lower-col-right">
            <UniversalRenamePanel config={config} targets={renameTargets} me={me} onError={onError} />
            <GiveawayPanel config={config} players={lobby.players} me={me} onError={onError} />
          </div>
        </div>
      </div>
      <aside className="side-column">
        <Leaderboard title="在线积分榜" players={lobby.rankedLeaderboard} />
        <ChatBoardShell title="大厅聊天" collapseKey="lobbyChat">
          <ChatPanel
            scope=""
            me={me}
            players={lobby.players}
            onError={onError}
            placeholder="点击头像可 @ 人"
            emptyText="还没有留言"
            messagesClass="lobby-suggestion-messages"
          />
        </ChatBoardShell>
      </aside>
      {showCreate && <CreateRoom config={config} me={me} onCreated={onGoRoom} onCancel={() => setShowCreate(false)} onError={onError} />}
    </section>
  );
}

export function CreateRoom({ config, me, onCreated, onCancel, onError }: { config: AppConfig; me: PublicPlayer; onCreated: (room?: RoomSnapshot) => void; onCancel: () => void; onError: (message: string) => void }) {
  const [settings, setSettings] = useState<RoomSettings>({
    name: defaultRoomName,
    gameId: "rps",
    enablePunishment: false,
    punishmentSource: "system",
    punishmentId: config.punishments[0]?.id,
    punishmentIds: config.punishments[0]?.id ? [config.punishments[0].id] : [],
    enableTags: false,
    tags: [],
    allowProofImage: true,
    tieDoublePunish: false,
    // 默认需要胜方审批证明（系统惩罚与自定义任务一致，避免提交后直接开新局）。
    requireOpponentConfirm: true,
    enableRanked: false,
    stake: 5,
    enableRankMultiplier: false,
    rankMultiplier: 1,
    enableExtremeRanked: false,
    othelloBoardTheme: "classic",
    tictactoeBoardTheme: "paper",
    gomokuBoardTheme: "wood",
    jungleBoardTheme: "forest",
    gomokuUndoLimit: 0,
    liarsDiceMinPlayers: 3,
    liarsDiceMaxPlayers: 3,
    othelloMoveSeconds: 0,
    othelloGameMinutes: 0,
    gomokuMoveSeconds: 0,
    gomokuGameMinutes: 0,
    jungleMoveSeconds: 0,
    jungleGameMinutes: 0
  });
  const [customRoomName, setCustomRoomName] = useState(false);

  useEffect(() => {
    setSettings((old) => {
      let changed = false;
      const next = { ...old };
      const validPunishmentIds = selectedPunishmentIdsForConfig(config, next);
      if ((next.punishmentSource || "system") === "system" && !sameStringArray(validPunishmentIds, next.punishmentIds || [])) {
        next.punishmentIds = validPunishmentIds;
        next.punishmentId = validPunishmentIds[validPunishmentIds.length - 1];
        changed = true;
      }
      if (next.tags?.some((tag) => !config.roomTags.includes(tag))) {
        next.tags = next.tags.filter((tag) => config.roomTags.includes(tag));
        next.enableTags = Boolean(next.tags.length);
        changed = true;
      }
      return changed ? next : old;
    });
  }, [config.punishments, config.roomTags]);

  function patch(next: Partial<RoomSettings>) {
    setSettings((old) => {
      const merged = { ...old, ...next };
      if (!customRoomName && next.gameId) {
        merged.name = next.gameId === "othello" ? defaultOthelloRoomName
          : next.gameId === "tictactoe" ? defaultTicTacToeRoomName
            : next.gameId === "liarsdice" ? defaultLiarsDiceRoomName
              : next.gameId === "gomoku" ? defaultGomokuRoomName
                : next.gameId === "jungle" ? defaultJungleRoomName
                  : defaultRoomName;
      }
      if (next.gameId === "othello" || merged.gameId === "othello") {
        merged.othelloBoardTheme = merged.othelloBoardTheme || "classic";
      }
      if (next.gameId === "tictactoe" || merged.gameId === "tictactoe") {
        merged.tictactoeBoardTheme = merged.tictactoeBoardTheme || "paper";
      }
      if (next.gameId === "gomoku" || merged.gameId === "gomoku") {
        merged.gomokuBoardTheme = merged.gomokuBoardTheme || "wood";
        merged.gomokuUndoLimit = merged.gomokuUndoLimit ?? 0;
      }
      if (next.gameId === "jungle" || merged.gameId === "jungle") {
        merged.jungleBoardTheme = merged.jungleBoardTheme || "forest";
      }
      if ("othelloMoveSeconds" in next && !moveSecondsOptions.includes(next.othelloMoveSeconds as typeof moveSecondsOptions[number])) {
        merged.othelloMoveSeconds = 0;
      }
      if ("othelloGameMinutes" in next && !gameMinutesOptions.includes(next.othelloGameMinutes as typeof gameMinutesOptions[number])) {
        merged.othelloGameMinutes = 0;
      }
      if ("gomokuMoveSeconds" in next && !moveSecondsOptions.includes(next.gomokuMoveSeconds as typeof moveSecondsOptions[number])) {
        merged.gomokuMoveSeconds = 0;
      }
      if ("gomokuGameMinutes" in next && !gameMinutesOptions.includes(next.gomokuGameMinutes as typeof gameMinutesOptions[number])) {
        merged.gomokuGameMinutes = 0;
      }
      if ("jungleMoveSeconds" in next && !moveSecondsOptions.includes(next.jungleMoveSeconds as typeof moveSecondsOptions[number])) {
        merged.jungleMoveSeconds = 0;
      }
      if ("jungleGameMinutes" in next && !gameMinutesOptions.includes(next.jungleGameMinutes as typeof gameMinutesOptions[number])) {
        merged.jungleGameMinutes = 0;
      }
      if (next.gameId === "liarsdice" || merged.gameId === "liarsdice") {
        merged.liarsDiceMinPlayers = merged.liarsDiceMinPlayers || 3;
        merged.liarsDiceMaxPlayers = merged.liarsDiceMaxPlayers || 3;
      }
      if ("liarsDiceMaxPlayers" in next) {
        const maxP = Math.min(8, Math.max(2, next.liarsDiceMaxPlayers || 3));
        merged.liarsDiceMaxPlayers = maxP;
        if ((merged.liarsDiceMinPlayers || 3) > maxP) merged.liarsDiceMinPlayers = maxP;
      }
      if ("liarsDiceMinPlayers" in next) {
        const maxP = merged.liarsDiceMaxPlayers || 3;
        merged.liarsDiceMinPlayers = Math.min(maxP, Math.max(2, next.liarsDiceMinPlayers || 3));
      }
      if (next.punishmentSource === "player") {
        merged.enablePunishment = true;
      }
      if (next.enableRanked === false) {
        merged.enableRankMultiplier = false;
        merged.rankMultiplier = 1;
        merged.enableExtremeRanked = false;
      }
      if (next.enableRankMultiplier) {
        merged.enableRanked = true;
        merged.enableExtremeRanked = false;
        if (!([2, 5, 10] as const).includes(merged.rankMultiplier as 2 | 5 | 10)) merged.rankMultiplier = 2;
      }
      if (next.enableExtremeRanked) {
        merged.enableRanked = true;
        merged.enableRankMultiplier = false;
        merged.rankMultiplier = 1;
      }
      if (!merged.enableRankMultiplier) {
        merged.rankMultiplier = 1;
      }
      if (!merged.enableRanked) {
        merged.enableExtremeRanked = false;
      }
      if (merged.gameId === "othello") {
        if (!([1, 2, 5, 10] as const).includes(merged.stake as 1 | 2 | 5 | 10)) merged.stake = 5;
        if (merged.enableExtremeRanked) {
          merged.enableRankMultiplier = false;
          merged.rankMultiplier = 1;
        }
      } else if (merged.gameId === "tictactoe") {
        if (!([5, 10, 20] as const).includes(merged.stake as 5 | 10 | 20)) merged.stake = 5;
      } else if (merged.gameId === "liarsdice") {
        if (!([5, 10, 20] as const).includes(merged.stake as 5 | 10 | 20)) merged.stake = 5;
      } else if (merged.gameId === "gomoku") {
        if (!([5, 10, 20] as const).includes(merged.stake as 5 | 10 | 20)) merged.stake = 5;
      } else if (merged.gameId === "jungle") {
        if (!([5, 10, 20] as const).includes(merged.stake as 5 | 10 | 20)) merged.stake = 5;
      } else if (!([5, 10, 20] as const).includes(merged.stake as 5 | 10 | 20)) {
        merged.stake = 5;
      }
      if (merged.punishmentSource === "system") {
        merged.punishmentIds = selectedPunishmentIdsForConfig(config, merged);
        merged.punishmentId = merged.punishmentIds[merged.punishmentIds.length - 1];
      } else {
        merged.punishmentIds = [];
        merged.punishmentId = undefined;
      }
      if (next.enableTags === false) {
        merged.tags = [];
      }
      if (next.tags) {
        merged.tags = next.tags.filter((tag) => config.roomTags.includes(tag)).slice(0, 5);
        merged.enableTags = merged.tags.length > 0;
      }
      if (!customRoomName && merged.enablePunishment && ("enablePunishment" in next || "punishmentId" in next || "punishmentIds" in next || "punishmentSource" in next)) {
        merged.name = generateRoomName(config, merged);
      }
      return merged;
    });
  }

  function togglePunishment(punishmentId: string) {
    const current = selectedPunishmentIdsForConfig(config, settings);
    const nextIds = current.includes(punishmentId)
      ? current.length > 1 ? current.filter((id) => id !== punishmentId) : current
      : [...current, punishmentId];
    patch({ punishmentIds: nextIds, punishmentId: nextIds[nextIds.length - 1] });
  }

  async function create() {
    try {
      const result = await ask<{ room: RoomSnapshot }>("room:create", { settings });
      onCreated(normalizeRoomSnapshot(result.room));
    } catch (error) {
      onError(error instanceof Error ? error.message : "创建失败");
    }
  }

  async function unlockMultiplierMode() {
    try {
      await ask("rankMultiplier:unlock", {});
      onError("倍率模式已解锁，已扣除 200 排位积分。");
    } catch (error) {
      onError(error instanceof Error ? error.message : "解锁失败");
    }
  }

  return (
    <div className="modal-backdrop" onClick={(event) => { if (event.target === event.currentTarget) event.stopPropagation(); }}>
      <section className="create-modal" onClick={(event) => event.stopPropagation()}>
        <div className="modal-title">
          <div>
            <h2>🏠 创建房间</h2>
            <p className="hint">选择玩法后就可以邀请朋友加入。</p>
          </div>
          <button type="button" className="icon-button" onClick={onCancel}>×</button>
        </div>
        <div className="create-scroll-area">
          <div className="create-box">
            <div className="create-section game-create-section">
              <h3>游戏</h3>
              <div className="game-choice-grid">
                {config.games.map((game) => (
                  <button
                    type="button"
                    className={`game-choice-card ${settings.gameId === game.id ? "active" : ""}`}
                    key={game.id}
                    onClick={() => patch({
                      gameId: game.id,
                      // 大话骰没有整局平局，隐藏并清掉平局双罚，避免歧义。
                      ...(game.id === "liarsdice" ? { tieDoublePunish: false } : {})
                    })}
                  >
                    <span className="game-choice-icon" aria-hidden="true">{gameIcon(game.id)}</span>
                    <span className="game-choice-copy">
                      <strong>{game.name}</strong>
                      <small>{game.description}</small>
                    </span>
                  </button>
                ))}
              </div>
              {settings.gameId === "othello" && <p className="hint">黑白棋支持真人 1v1、观战、聊天、排位和惩罚；排位房会支持白给/上贡结算。</p>}
              {settings.gameId === "tictactoe" && <p className="hint">井字棋支持真人 1v1、观战、聊天、排位和惩罚；双方准备后随机 X/O 先手。</p>}
              {settings.gameId === "liarsdice" && <p className="hint">大话骰支持 2-8 人参战，进房默认观战，可自由加入/离开参战席；全员准备且名单 5 秒无变动后自动开局。</p>}
              {settings.gameId === "gomoku" && <p className="hint">五子棋支持真人 1v1、观战、聊天、排位和惩罚；15x15 棋盘先连成五子者胜，可向对方请求悔棋或认输。</p>}
              {settings.gameId === "jungle" && <p className="hint">斗兽棋支持真人 1v1、观战、聊天、排位和惩罚；7×9 棋盘，先入对方兽穴或令对方无子可走者胜，可向对方申请认输。</p>}
              {settings.gameId === "jungle" && (
                <div className="game-timer-settings">
                  <label>
                    每步时长
                    <select
                      value={settings.jungleMoveSeconds ?? 0}
                      onChange={(event) => patch({ jungleMoveSeconds: Number(event.target.value) as RoomSettings["jungleMoveSeconds"] })}
                    >
                      {moveSecondsOptions.map((sec) => <option key={sec} value={sec}>{sec === 0 ? "不限" : `${sec} 秒`}</option>)}
                    </select>
                  </label>
                  <label>
                    每局时长
                    <select
                      value={settings.jungleGameMinutes ?? 0}
                      onChange={(event) => patch({ jungleGameMinutes: Number(event.target.value) as RoomSettings["jungleGameMinutes"] })}
                    >
                      {gameMinutesOptions.map((min) => <option key={min} value={min}>{min === 0 ? "不限" : `${min} 分钟`}</option>)}
                    </select>
                  </label>
                </div>
              )}
              {settings.gameId === "gomoku" && (
                <div className="game-timer-settings">
                  <label>
                    悔棋次数
                    <select
                      value={settings.gomokuUndoLimit ?? 0}
                      onChange={(event) => patch({ gomokuUndoLimit: Number(event.target.value) as RoomSettings["gomokuUndoLimit"] })}
                    >
                      {([0, 1, 3, 10] as const).map((limit) => (
                        <option key={limit} value={limit}>{limit === 0 ? "禁止" : `${limit}次`}</option>
                      ))}
                    </select>
                  </label>
                  <label>
                    每子时长
                    <select
                      value={settings.gomokuMoveSeconds ?? 0}
                      onChange={(event) => patch({ gomokuMoveSeconds: Number(event.target.value) as RoomSettings["gomokuMoveSeconds"] })}
                    >
                      {moveSecondsOptions.map((n) => (
                        <option key={n} value={n}>{n === 0 ? "不限" : `${n} 秒`}</option>
                      ))}
                    </select>
                  </label>
                  <label>
                    每局时长
                    <select
                      value={settings.gomokuGameMinutes ?? 0}
                      onChange={(event) => patch({ gomokuGameMinutes: Number(event.target.value) as RoomSettings["gomokuGameMinutes"] })}
                    >
                      {gameMinutesOptions.map((n) => (
                        <option key={n} value={n}>{n === 0 ? "不限" : `${n} 分钟`}</option>
                      ))}
                    </select>
                  </label>
                </div>
              )}
              {settings.gameId === "liarsdice" && (
                <div className="liarsdice-roster-settings">
                  <label>
                    最少参战人数
                    <select
                      value={settings.liarsDiceMinPlayers ?? 3}
                      onChange={(event) => patch({ liarsDiceMinPlayers: Number(event.target.value) })}
                    >
                      {Array.from({ length: (settings.liarsDiceMaxPlayers ?? 3) - 1 }, (_, i) => i + 2).map((n) => (
                        <option key={n} value={n}>{n} 人</option>
                      ))}
                    </select>
                  </label>
                  <label>
                    最多参战人数
                    <select
                      value={settings.liarsDiceMaxPlayers ?? 3}
                      onChange={(event) => patch({ liarsDiceMaxPlayers: Number(event.target.value) })}
                    >
                      {Array.from({ length: 7 }, (_, i) => i + 2).map((n) => (
                        <option key={n} value={n}>{n} 人</option>
                      ))}
                    </select>
                  </label>
                </div>
              )}
              {settings.gameId === "othello" && (
                <div className="game-timer-settings">
                  <label>
                    每子时长
                    <select
                      value={settings.othelloMoveSeconds ?? 0}
                      onChange={(event) => patch({ othelloMoveSeconds: Number(event.target.value) as RoomSettings["othelloMoveSeconds"] })}
                    >
                      {moveSecondsOptions.map((n) => (
                        <option key={n} value={n}>{n === 0 ? "不限" : `${n} 秒`}</option>
                      ))}
                    </select>
                  </label>
                  <label>
                    每局时长
                    <select
                      value={settings.othelloGameMinutes ?? 0}
                      onChange={(event) => patch({ othelloGameMinutes: Number(event.target.value) as RoomSettings["othelloGameMinutes"] })}
                    >
                      {gameMinutesOptions.map((n) => (
                        <option key={n} value={n}>{n === 0 ? "不限" : `${n} 分钟`}</option>
                      ))}
                    </select>
                  </label>
                </div>
              )}
              {settings.gameId === "othello" && (
                <div className="othello-theme-grid">
                  {othelloBoardThemes.map((theme) => (
                    <button
                      type="button"
                      className={`othello-theme-card ${settings.othelloBoardTheme === theme.id ? "active" : ""}`}
                      key={theme.id}
                      onClick={() => patch({ othelloBoardTheme: theme.id })}
                      style={{
                        "--theme-board": theme.board,
                        "--theme-cell": theme.cell,
                        "--theme-line": theme.line,
                        "--theme-border": theme.border,
                        "--theme-black-disc": theme.blackDisc,
                        "--theme-white-disc": theme.whiteDisc,
                        "--theme-black-ring": theme.blackRing,
                        "--theme-white-ring": theme.whiteRing
                      } as CSSProperties}
                    >
                      <span className="othello-theme-preview">
                        <i><b className="preview-disc black" /></i>
                        <i />
                        <i />
                        <i><b className="preview-disc white" /></i>
                      </span>
                      <strong>{theme.name}</strong>
                      <small>{theme.description}</small>
                    </button>
                  ))}
                </div>
              )}
              {settings.gameId === "tictactoe" && (
                <div className="tictactoe-theme-grid">
                  {tictactoeBoardThemes.map((theme) => (
                    <button
                      type="button"
                      className={`tictactoe-theme-card ${settings.tictactoeBoardTheme === theme.id ? "active" : ""}`}
                      key={theme.id}
                      onClick={() => patch({ tictactoeBoardTheme: theme.id })}
                      style={{
                        "--ttt-board": theme.board,
                        "--ttt-cell": theme.cell,
                        "--ttt-line": theme.line,
                        "--ttt-border": theme.border,
                        "--ttt-x": theme.x,
                        "--ttt-o": theme.o
                      } as CSSProperties}
                    >
                      <span className="tictactoe-theme-preview">
                        <i>×</i>
                        <i />
                        <i>○</i>
                        <i />
                        <i>×</i>
                        <i />
                        <i>○</i>
                        <i />
                        <i>×</i>
                      </span>
                      <strong>{theme.name}</strong>
                      <small>{theme.description}</small>
                    </button>
                  ))}
                </div>
              )}
              {settings.gameId === "gomoku" && (
                <div className="othello-theme-grid">
                  {gomokuBoardThemes.map((theme) => (
                    <button
                      type="button"
                      className={`othello-theme-card ${settings.gomokuBoardTheme === theme.id ? "active" : ""}`}
                      key={theme.id}
                      onClick={() => patch({ gomokuBoardTheme: theme.id })}
                      style={{
                        "--theme-board": theme.board,
                        "--theme-cell": theme.cell,
                        "--theme-line": theme.line,
                        "--theme-border": theme.border,
                        "--theme-black-disc": theme.blackDisc,
                        "--theme-white-disc": theme.whiteDisc,
                        "--theme-black-ring": theme.blackRing,
                        "--theme-white-ring": theme.whiteRing
                      } as CSSProperties}
                    >
                      <span className="othello-theme-preview">
                        <i><b className="preview-disc black" /></i>
                        <i />
                        <i />
                        <i><b className="preview-disc white" /></i>
                      </span>
                      <strong>{theme.name}</strong>
                      <small>{theme.description}</small>
                    </button>
                  ))}
                </div>
              )}
              {settings.gameId === "jungle" && (
                <div className="othello-theme-grid">
                  {jungleBoardThemes.map((theme) => (
                    <button
                      type="button"
                      className={`othello-theme-card ${settings.jungleBoardTheme === theme.id ? "active" : ""}`}
                      key={theme.id}
                      onClick={() => patch({ jungleBoardTheme: theme.id })}
                      style={{
                        "--theme-board": theme.board,
                        "--theme-cell": theme.land,
                        "--theme-line": theme.water,
                        "--theme-border": theme.border
                      } as CSSProperties}
                    >
                      <span className="othello-theme-preview">
                        <i />
                        <i />
                        <i />
                        <i />
                      </span>
                      <strong>{theme.name}</strong>
                      <small>{theme.description}</small>
                    </button>
                  ))}
                </div>
              )}
            </div>
            <div className="create-section">
              <h3>基础</h3>
              <input value={settings.name} onKeyDown={preventEnterSubmit} onChange={(event) => { setCustomRoomName(true); patch({ name: event.target.value }); }} placeholder="房间名" />
              <input value={settings.password || ""} onKeyDown={preventEnterSubmit} onChange={(event) => patch({ password: event.target.value || undefined })} placeholder="房间密码，可不填" />
              <Toggle label="显示房间 Tag" value={settings.enableTags ?? false} onChange={(value) => patch({ enableTags: value })} />
              {settings.enableTags && (
                <TagPicker
                  options={config.roomTags}
                  value={settings.tags || []}
                  onChange={(tags) => patch({ tags })}
                />
              )}
            </div>
            <div className="create-section">
              <h3>玩法</h3>
              <div className="ranked-choice-grid">
                <button type="button" className={`ranked-choice-card ${!settings.enableRanked ? "active" : ""}`} onClick={() => patch({ enableRanked: false, enableExtremeRanked: false })}>
                  <span>🎮 普通局</span>
                  <small>不增加/减少排位积分，适合随意对战。</small>
                </button>
                {(settings.gameId === "othello" ? ([1, 2, 5, 10] as const) : ([5, 10, 20] as const)).map((stake) => (
                  <button type="button" className={`ranked-choice-card ${settings.enableRanked && settings.stake === stake ? "active" : ""}`} key={stake} onClick={() => patch({ enableRanked: true, stake, enableExtremeRanked: Boolean(me.extremeModeEnabled) })}>
                    <span>{settings.gameId === "othello" ? "🏆 黑白棋排位" : settings.gameId === "tictactoe" ? "🏆 井字棋排位" : settings.gameId === "gomoku" ? "🏆 五子棋排位" : settings.gameId === "jungle" ? "🏆 斗兽棋排位" : me.extremeModeEnabled ? "⚡ 极限排位" : "🏆 排位"} {stake}{settings.gameId === "othello" ? " 分/子" : " 分"}</span>
                    <small>{
                      settings.gameId === "othello"
                        ? `每翻掉对方 1 子立即结算 ${stake} 分，终局不重复结算。`
                        : settings.gameId === "liarsdice"
                          ? `胜利 +${stake}，失败 -${stake}；其余参战玩家本局平，不计分。`
                          : me.extremeModeEnabled
                            ? "只能创建极限排位房；非极限玩家无法进入。"
                            : `胜利 +${stake}，失败 -${stake}；普通平局不扣分，平局双罚时双方 -${stake}。`
                    }</small>
                  </button>
                ))}
              </div>
              {settings.gameId === "othello" && <p className="hint">黑白棋排位按实时翻子结算，可选 1/2/5/10 分/子；支持倍率和极限模式，但两者不能同时开启。</p>}
              {settings.gameId === "tictactoe" && <p className="hint">井字棋排位按胜负固定分结算，可选 5/10/20 分；支持倍率和极限模式。</p>}
              {settings.gameId === "gomoku" && <p className="hint">五子棋排位按胜负固定分结算，可选 5/10/20 分；支持倍率和极限模式。</p>}
              {settings.gameId === "jungle" && <p className="hint">斗兽棋排位按胜负固定分结算，可选 5/10/20 分；支持倍率和极限模式。</p>}
              {settings.enableRanked && me.extremeModeEnabled && (
                <div className="multiplier-box extreme-mode-box">
                  <div className="multiplier-head">
                    <strong>⚡ 极限排位已开启</strong>
                    <span>禁用倍率</span>
                  </div>
                  <p className="hint">极限排位会按你的极限模式分段调整加减分；非极限玩家无法进入这个房间。</p>
                </div>
              )}
              {settings.enableRanked && !me.extremeModeEnabled && (
                <p className="hint">你没有开启极限模式，因此只能创建普通排位房。</p>
              )}
              {settings.enableRanked && (
                <div className="multiplier-box">
                  <div className="multiplier-head">
                    <strong>倍率模式</strong>
                    <span>{settings.enableRankMultiplier ? `x${settings.rankMultiplier || 1}` : "未开启"}</span>
                  </div>
                  {me.extremeModeEnabled ? (
                    <p className="hint danger-hint">极限模式不能开启倍率房间，也不能进入倍率房；黑白棋极限排位会按每次翻子实时套用极限折扣。</p>
                  ) : !me.rankMultiplierUnlocked ? (
                    <>
                      <p className="hint">提交 200 排位积分后，本次服务器运行期间可创建 2倍 / 5倍 / 10倍排位房。</p>
                      <button type="button" className="soft-button" disabled={me.stats.rankedPoints < 200} onClick={unlockMultiplierMode}>提交 200 积分解锁</button>
                      {me.stats.rankedPoints < 200 && <p className="hint danger-hint">你的排位积分不足 200，暂时不能解锁。</p>}
                    </>
                  ) : (
                    <>
                      <div className="multiplier-choice-grid">
                        {([1, 2, 5, 10] as const).map((multiplier) => (
                          <button
                            type="button"
                            className={`ranked-choice-card ${rankMultiplierForSettings(settings) === multiplier ? "active" : ""}`}
                            key={multiplier}
                            onClick={() => patch({ enableRankMultiplier: multiplier > 1, rankMultiplier: multiplier })}
                          >
                            <span>{multiplier === 1 ? "普通倍率" : `x${multiplier} 倍房`}</span>
                            <small>{multiplier === 1 ? "按基础赌分结算。" : settings.gameId === "othello" ? `每翻 1 子按 ${settings.stake * multiplier} 分结算。` : `胜负按 ${settings.stake * multiplier} 分结算。`}</small>
                          </button>
                        ))}
                      </div>
                      <p className="hint">当前：排位 {settings.stake}{settings.gameId === "othello" ? " 分/子" : " 分"} × {rankMultiplierForSettings(settings)} 倍 = {settings.gameId === "othello" ? `每翻 1 子 ${settings.stake * rankMultiplierForSettings(settings)} 分` : `胜负 ${settings.stake * rankMultiplierForSettings(settings)} 分`}。</p>
                    </>
                  )}
                </div>
              )}
            </div>
            <div className="create-section">
              <h3>惩罚</h3>
              <Toggle label="惩罚模式" value={settings.enablePunishment} onChange={(value) => patch({ enablePunishment: value })} />
              {settings.gameId === "othello" && <p className="hint">黑白棋惩罚会在终局、认输、逃跑或断线判负后触发；平局双罚开启时黑白棋平局双方都要惩罚。</p>}
              {settings.gameId === "tictactoe" && <p className="hint">井字棋惩罚会在终局或断线判负后触发；平局双罚开启时井字棋平局双方都要惩罚。</p>}
              {settings.gameId === "gomoku" && <p className="hint">五子棋惩罚会在终局、认输或断线判负后触发；平局双罚开启时五子棋平局双方都要惩罚。</p>}
              {settings.gameId === "jungle" && <p className="hint">斗兽棋惩罚会在终局、认输或断线判负后触发；平局双罚开启时斗兽棋平局双方都要惩罚。</p>}
              {settings.gameId === "liarsdice" && <p className="hint">大话骰惩罚仅对败者触发（叫点/开牌对决中的负方，或断线判负方）；其余参战玩家记平但不计分、不受罚。</p>}
              {settings.enablePunishment && (
                <>
                  <Select value={settings.punishmentSource || "system"} onChange={(value) => patch({ punishmentSource: value as RoomSettings["punishmentSource"] })} options={[
                    { value: "system", label: "系统任务" },
                    { value: "player", label: "玩家发布" }
                  ]} />
                  {(settings.punishmentSource || "system") === "system" ? (
                    <>
                      <p className="hint">已选择 {selectedPunishmentIdsForConfig(config, settings).length} 个惩罚池；每局会先随机一个惩罚池，再随机任务。</p>
                      <div className="punishment-choice-grid">
                        {config.punishments.map((punishment) => {
                          const active = selectedPunishmentIdsForConfig(config, settings).includes(punishment.id);
                          return (
                            <button
                              type="button"
                              className={`punishment-choice-card ${active ? "active" : ""}`}
                              key={punishment.id}
                              onClick={() => togglePunishment(punishment.id)}
                              style={{
                                "--punishment-bg": punishment.cardImageUrl ? `url(${punishment.cardImageUrl})` : "none",
                                "--punishment-bg-opacity": String(punishment.cardImageOpacity ?? 0.26)
                              } as CSSProperties}
                            >
                              <div className="punishment-choice-meta">
                                <em>{active ? "已选" : "可选"}</em>
                                <em>{punishmentTasks(punishment, config).length} 个任务</em>
                              </div>
                              <span>{punishment.name}</span>
                              <small>{punishment.description}</small>
                            </button>
                          );
                        })}
                      </div>
                    </>
                  ) : (
                    <p className="hint">本局结算后，由对手临时写惩罚任务；任务不会保存到后台配置。</p>
                  )}
                  {settings.gameId !== "liarsdice" && (
                    <>
                      <Toggle label="平局双罚" value={settings.tieDoublePunish} onChange={(value) => patch({ tieDoublePunish: value })} />
                      {settings.enableRanked && settings.tieDoublePunish && (
                        <p className="hint">排位平局双罚开启时，平局双方都会扣 {settings.stake} 分。</p>
                      )}
                    </>
                  )}
                  <Toggle label="惩罚需对手确认" value={settings.requireOpponentConfirm} onChange={(value) => patch({ requireOpponentConfirm: value })} />
                  <Toggle label="允许图片证明" value={settings.allowProofImage ?? true} onChange={(value) => patch({ allowProofImage: value })} />
                </>
              )}
            </div>
          </div>
        </div>
        <div className="modal-actions">
          <button type="button" onClick={onCancel}>取消</button>
          <button type="button" className="primary" onClick={create}>创建房间</button>
        </div>
      </section>
    </div>
  );
}

export function generateRoomName(config: AppConfig, settings: RoomSettings) {
  if (settings.gameId === "othello" && !settings.enablePunishment) return defaultOthelloRoomName;
  if (settings.gameId === "tictactoe" && !settings.enablePunishment) return defaultTicTacToeRoomName;
  if (settings.gameId === "gomoku" && !settings.enablePunishment) return defaultGomokuRoomName;
  if (settings.gameId === "jungle" && !settings.enablePunishment) return defaultJungleRoomName;
  if (settings.gameId === "liarsdice" && !settings.enablePunishment) return defaultLiarsDiceRoomName;
  const pool = settings.punishmentSource === "player"
    ? config.playerPunishmentRoomNamePool
    : primaryPunishmentForSettings(config, settings)?.roomNamePool;
  if (!pool) return defaultRoomName;
  const subject = randomItem(pool.subjects);
  const roomWord = randomItem(pool.roomWords);
  const adjective = pool.adjectives.length && Math.random() < 0.75 ? randomItem(pool.adjectives) : "";
  return `${adjective}${subject}${roomWord}`;
}

export function randomItem(items: string[]) {
  return items[Math.floor(Math.random() * items.length)] || "";
}

export function selectedPunishmentIdsForConfig(config: AppConfig, settings: RoomSettings) {
  const rawIds = settings.punishmentIds?.length ? settings.punishmentIds : settings.punishmentId ? [settings.punishmentId] : [];
  const validIds = rawIds.filter((id, index) =>
    rawIds.indexOf(id) === index &&
    config.punishments.some((punishment) => punishment.id === id)
  );
  if (validIds.length) return validIds;
  return config.punishments[0]?.id ? [config.punishments[0].id] : [];
}

export function primaryPunishmentForSettings(config: AppConfig, settings: RoomSettings) {
  const ids = selectedPunishmentIdsForConfig(config, settings);
  const lastId = ids[ids.length - 1];
  return config.punishments.find((punishment) => punishment.id === lastId);
}

export function sameStringArray(left: string[], right: string[]) {
  return left.length === right.length && left.every((item, index) => item === right[index]);
}

export function rankMultiplierForSettings(settings: Pick<RoomSettings, "enableRanked" | "enableRankMultiplier" | "rankMultiplier">) {
  if (!settings.enableRanked || !settings.enableRankMultiplier) return 1;
  return ([2, 5, 10] as const).includes(settings.rankMultiplier as 2 | 5 | 10) ? settings.rankMultiplier || 1 : 1;
}

export function RankMultiplierBadge({ multiplier }: { multiplier?: number }) {
  if (!multiplier || multiplier <= 1) return null;
  return <span className="rank-multiplier-badge">x{multiplier}</span>;
}

export function gameIcon(gameId: RoomSettings["gameId"]) {
  if (gameId === "othello") return "⚫⚪";
  if (gameId === "tictactoe") return "❌⭕";
  if (gameId === "gomoku") return "●○";
  if (gameId === "jungle") return "🦁";
  if (gameId === "liarsdice") return "🎲";
  return "✊✌️🖐️";
}

export function ExtremeRankedBadge({ enabled }: { enabled?: boolean }) {
  if (!enabled) return null;
  return <span className="rank-multiplier-badge extreme-ranked-badge">⚡ 极限</span>;
}

export function preventEnterSubmit(event: ReactKeyboardEvent<HTMLInputElement>) {
  if (event.key === "Enter") event.preventDefault();
}

export function RoomTagList({ tags }: { tags: string[] }) {
  return (
    <div className="room-tag-list">
      {tags.map((tag) => <span className="room-tag" key={tag}>#{tag}</span>)}
    </div>
  );
}

export function RoomVersusLine({ room }: { room: LobbySnapshot["rooms"][number] }) {
  // 大话骰是 N 人动态名单，没有 A/B 对阵；用骰子行取代 VS 行。
  if (room.gameId === "liarsdice") {
    return (
      <div className="room-versus-line liarsdice-line" title={`大话骰 · ${room.players} 人参战`}>
        <span aria-hidden="true">🎲</span>
        <b>{room.players > 0 ? `${room.players} 人参战` : "等待玩家加入"}</b>
      </div>
    );
  }
  const left = room.versus.A;
  const right = room.versus.B;
  const leftName = left?.player?.name || "等待玩家";
  const rightName = right?.player?.name || "等待玩家";
  return (
    <div className="room-versus-line" title={`${leftName} VS ${rightName}`}>
      <RoomVersusSeat occupant={left} />
      <b>VS</b>
      <RoomVersusSeat occupant={right} />
    </div>
  );
}

export function RoomVersusSeat({ occupant }: { occupant: LobbySnapshot["rooms"][number]["versus"]["A"] }) {
  if (!occupant) return <span className="empty">等待玩家</span>;
  return <PlayerBadge player={occupant.player} compact />;
}

export function TagPicker({ options, value, onChange }: { options: string[]; value: string[]; onChange: (tags: string[]) => void }) {
  function toggle(tag: string) {
    if (value.includes(tag)) onChange(value.filter((item) => item !== tag));
    else onChange([...value, tag].slice(0, 5));
  }
  return (
    <div className="room-tag-picker">
      {options.map((tag) => (
        <button type="button" className={value.includes(tag) ? "active" : ""} key={tag} onClick={() => toggle(tag)}>
          #{tag}
        </button>
      ))}
      {options.length === 0 && <p className="empty">后台还没有配置房间 Tag</p>}
    </div>
  );
}

export function Room({ config, room, me, lobby, onBack, onError }: { config: AppConfig; room: RoomSnapshot; me: PublicPlayer; lobby: LobbySnapshot | null; onBack: () => void; onError: (message: string) => void }) {
  const [chatTab, setChatTab] = useState<"room" | "lobby">("room");
  const [proofText, setProofText] = useState("");
  const [proofImage, setProofImage] = useState("");
  const [proofSubmitting, setProofSubmitting] = useState(false);
  const [localChoice, setLocalChoice] = useState<Move | null>(null);
  const [redoInputs, setRedoInputs] = useState<Record<string, string>>({});
  const [taskInputs, setTaskInputs] = useState<Record<string, string>>({});
  const [now, setNow] = useState(Date.now());
  const [previewImage, setPreviewImage] = useState<string | null>(null);
  const [extraHistory, setExtraHistory] = useState<RoomSnapshot["roundHistory"]>([]);
  const [historyStick, setHistoryStick] = useState(true);
  const [historyLoading, setHistoryLoading] = useState(false);
  const historyListRef = useRef<HTMLDivElement | null>(null);
  const historyStickRef = useRef(true);
  const historyLoadingRef = useRef(false);
  const { collapsed: roomPlayersCollapsed, toggle: toggleRoomPlayersCollapsed } = useMobileCollapse("roomPlayers");
  const { collapsed: roundHistoryCollapsed, toggle: toggleRoundHistoryCollapsed } = useMobileCollapse("roundHistory");
  const seats = room.seats || { A: null, B: null };
  const choices = room.choices || {};
  const mySeat = seats.A?.id === me.id ? "A" : seats.B?.id === me.id ? "B" : null;
  const myChoice = mySeat ? room.phase === "result" ? undefined : localChoice || choices[mySeat] : undefined;
  const resultChoice = mySeat ? room.revealedChoices?.[mySeat] : undefined;
  const canChoose = Boolean(mySeat && room.phase !== "punishment" && (room.phase === "choosing" || room.phase === "result") && seats.A && seats.B);
  const canShowGiveawayButton = Boolean(mySeat && me.giveawayEnabled && seats.A && seats.B);
  const canGoSpectate = Boolean(mySeat && room.phase !== "punishment" && !choices[mySeat] && !((room.settings.gameId === "tictactoe" || room.settings.gameId === "gomoku" || room.settings.gameId === "jungle") && room.phase === "choosing"));
  const opponentSeat = mySeat === "A" ? "B" : mySeat === "B" ? "A" : null;
  const opponentPlayer = opponentSeat ? seats[opponentSeat] : null;
  const forceGiveawayEligibleGame = room.settings.gameId === "rps" || room.settings.gameId === "tictactoe" || room.settings.gameId === "gomoku" || (room.settings.gameId === "othello" && room.settings.enableRanked);
  const [myPetIds, setMyPetIds] = useState<Set<string>>(new Set());
  useEffect(() => {
    if (!forceGiveawayEligibleGame) return;
    let alive = true;
    const applyPets = (payload: PetBondState) => {
      if (!alive || !payload || typeof payload !== "object") return;
      setMyPetIds(new Set((payload.pets || []).map((p) => p.playerId)));
    };
    ask<PetBondState>("petbond:getState", {}).then(applyPets).catch(() => { });
    socket.on("petbond:update", applyPets);
    return () => {
      alive = false;
      socket.off("petbond:update", applyPets);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [me.id, forceGiveawayEligibleGame]);
  const canForceGiveaway = Boolean(
    forceGiveawayEligibleGame && mySeat && opponentPlayer &&
    myPetIds.has(opponentPlayer.id) && opponentPlayer.giveawayEnabled
  );
  async function forceGiveaway() {
    if (!opponentPlayer) return;
    if (!window.confirm(`确定强制 ${opponentPlayer.name} 白给吗？`)) return;
    await act("petbond:forceGiveaway", { targetId: opponentPlayer.id });
  }
  const roomPlayers = roomPlayerList(room);
  // 聊天发言人查找源：房间名单优先（在房间内持续更新），叠加大厅名单兜底
  // （进房后大厅频道会取消订阅，可能不是最新的，但依旧好过完全查不到）。
  const chatPlayers = useMemo(() => {
    const map = new Map<string, PublicPlayer>();
    for (const player of lobby?.players || []) map.set(player.id, player);
    for (const item of roomPlayers) map.set(item.player.id, item.player);
    return Array.from(map.values());
  }, [lobby, roomPlayers]);
  const punishedNames = punishedPlayerNames(room);
  const punishedIds = room.punishedPlayerIds || [];
  const iAmPunished = punishedIds.includes(me.id);
  const visibleRoundHistory = [...room.roundHistory, ...extraHistory.filter((item) => !room.roundHistory.some((fresh) => fresh.id === item.id))];
  // visibleRoundHistory 是服务端语义的新→旧；展示上与聊天记录保持一致（旧的在上，新的在下），所以渲染时整体反转。
  const orderedRoundHistory = [...visibleRoundHistory].reverse();
  const hasMoreHistory = visibleRoundHistory.length < room.roundHistoryTotal;
  const leaveTitle = room.phase === "punishment"
    ? iAmPunished
      ? "惩罚完成前不能离开房间"
      : "离开后，服务器会自动处理你负责的审核或任务"
    : room.settings.gameId === "tictactoe" && room.phase === "choosing" && mySeat
      ? "井字棋对局进行中不能离开战斗席"
      : room.settings.gameId === "gomoku" && room.phase === "choosing" && mySeat
        ? "五子棋对局进行中不能离开战斗席"
        : room.settings.gameId === "jungle" && room.phase === "choosing" && mySeat
          ? "斗兽棋对局进行中不能离开战斗席"
          : "离开房间";

  useEffect(() => {
    if (!mySeat || room.phase === "choosing" && !choices[mySeat]) setLocalChoice(null);
  }, [mySeat, room.phase, choices]);

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  useEffect(() => {
    if (room.settings.allowProofImage === false) setProofImage("");
  }, [room.settings.allowProofImage]);

  useEffect(() => {
    setExtraHistory([]);
  }, [room.id]);

  async function act(event: string, payload: unknown = {}) {
    try {
      await ask(event, payload);
    } catch (error) {
      onError(error instanceof Error ? error.message : "操作失败");
    }
  }

  async function uploadImage(file: File) {
    let localPreview = "";
    try {
      const uploadFile = await prepareProofImageForUpload(file);
      // 立刻本地预览（blob，不显示「加载中」）
      localPreview = URL.createObjectURL(uploadFile);
      setProofImage(localPreview);
      const form = new FormData();
      form.append("token", localStorage.getItem(tokenKey) || "");
      form.append("image", uploadFile, uploadFile.name.endsWith(".webp") ? uploadFile.name : "proof.webp");
      const response = await fetch("/api/proof-image", { method: "POST", body: form });
      let data: { message?: string; imageUrl?: string } = {};
      try {
        data = await response.json();
      } catch {
        throw new Error(response.ok ? "服务器响应无效" : `上传失败（${response.status}）`);
      }
      if (!response.ok) throw new Error(data.message || "上传失败");
      if (!data.imageUrl) throw new Error("服务器未返回图片地址");
      // 提交必须用服务端 URL；展示切到远端（ProofImage 已处理缓存 onLoad）
      setProofImage(data.imageUrl);
    } catch (error) {
      setProofImage("");
      onError(error instanceof Error ? error.message : "图片上传失败");
    } finally {
      if (localPreview) {
        const url = localPreview;
        window.setTimeout(() => URL.revokeObjectURL(url), 3000);
      }
    }
  }

  async function choose(move: Move) {
    setLocalChoice(move);
    try {
      await ask("room:move", { move });
    } catch (error) {
      setLocalChoice(null);
      onError(error instanceof Error ? error.message : "出拳失败");
    }
  }

  async function playOthello(row: number, col: number) {
    try {
      await ask("othello:move", { row, col });
    } catch (error) {
      onError(error instanceof Error ? error.message : "落子失败");
    }
  }

  async function playTicTacToe(row: number, col: number) {
    try {
      await ask("tictactoe:move", { row, col });
    } catch (error) {
      onError(error instanceof Error ? error.message : "落子失败");
    }
  }

  async function chooseTicTacToeGiveaway(mode: "normal" | "giveaway") {
    try {
      await ask("tictactoe:giveawayChoice", { mode });
    } catch (error) {
      onError(error instanceof Error ? error.message : "白给选择失败");
    }
  }

  async function settleOthelloMove(mode: "normal" | "giveaway" | "tribute") {
    try {
      await ask("othello:settleMove", { mode });
    } catch (error) {
      onError(error instanceof Error ? error.message : "结算失败");
    }
  }

  async function restartOthello() {
    try {
      await ask("othello:restart", {});
    } catch (error) {
      onError(error instanceof Error ? error.message : "重新开始失败");
    }
  }

  async function readyOthello() {
    try {
      await ask("othello:ready", {});
    } catch (error) {
      onError(error instanceof Error ? error.message : "准备失败");
    }
  }

  async function readyTicTacToe() {
    try {
      await ask("tictactoe:ready", {});
    } catch (error) {
      onError(error instanceof Error ? error.message : "准备失败");
    }
  }

  async function restartTicTacToe() {
    try {
      await ask("tictactoe:restart", {});
    } catch (error) {
      onError(error instanceof Error ? error.message : "重新开始失败");
    }
  }

  async function requestOthelloSurrender() {
    try {
      await ask("othello:requestSurrender", {});
    } catch (error) {
      onError(error instanceof Error ? error.message : "申请认输失败");
    }
  }

  async function respondOthelloSurrender(accept: boolean) {
    try {
      await ask("othello:respondSurrender", { accept });
    } catch (error) {
      onError(error instanceof Error ? error.message : "处理认输失败");
    }
  }

  async function escapeOthello() {
    if (!window.confirm("确定要逃跑吗？本局会立即判负，并按剩余空格追加扣分。")) return;
    try {
      await ask("othello:escape", {});
    } catch (error) {
      onError(error instanceof Error ? error.message : "逃跑失败");
    }
  }

  async function submitProof() {
    const text = proofText.trim();
    if (!text) {
      onError("请填写文字证明");
      return;
    }
    if (proofSubmitting) return;
    setProofSubmitting(true);
    try {
      await ask("punishment:submit", { text, imageUrl: proofImage || "" });
      setProofText("");
      setProofImage("");
    } catch (error) {
      onError(error instanceof Error ? error.message : "提交失败");
    } finally {
      setProofSubmitting(false);
    }
  }

  async function reviewProof(playerId: string, action: "approve" | "forgive" | "reject") {
    try {
      await ask("punishment:review", { playerId, action, redoTaskText: redoInputs[playerId] });
      setRedoInputs((old) => ({ ...old, [playerId]: "" }));
    } catch (error) {
      onError(error instanceof Error ? error.message : "审核失败");
    }
  }

  async function assignPunishmentTask(playerId: string) {
    try {
      await ask("punishment:assignTask", { playerId, taskText: taskInputs[playerId] });
      setTaskInputs((old) => ({ ...old, [playerId]: "" }));
    } catch (error) {
      onError(error instanceof Error ? error.message : "发布任务失败");
    }
  }

  async function leaveCurrentRoom() {
    if (room.phase === "punishment" && iAmPunished) {
      onError("惩罚完成前不能离开房间");
      return;
    }
    try {
      await ask("room:leave", {});
      onBack();
    } catch (error) {
      onError(error instanceof Error ? error.message : "离开房间失败");
    }
  }

  // 瀑布流：每次向服务端补 5 条更早的对局记录（room:history 按 offset/limit 分页，offset 越大越旧）。
  async function loadMoreHistory(): Promise<number> {
    if (historyLoadingRef.current || !hasMoreHistory) return 0;
    historyLoadingRef.current = true;
    setHistoryLoading(true);
    try {
      const result = await ask<{ items: RoomSnapshot["roundHistory"]; total: number }>("room:history", {
        roomId: room.id,
        offset: visibleRoundHistory.length,
        limit: 5
      });
      const items = (result.items || []).map(normalizeRoundHistoryItem);
      setExtraHistory((old) => appendHistoryPage(old, items, room.roundHistory || []));
      return items.length;
    } catch (error) {
      onError(error instanceof Error ? error.message : "加载对局记录失败");
      return 0;
    } finally {
      historyLoadingRef.current = false;
      setHistoryLoading(false);
    }
  }

  // 新记录到达时若已在底部则自动跟到最新；用户上滑查看历史时不会被打断。
  useEffect(() => {
    const list = historyListRef.current;
    if (list && historyStickRef.current) scrollToBottomSoon(list);
  }, [visibleRoundHistory.length]);

  // 滚到顶部附近（展示上是「更旧的记录」）时瀑布流加载更早 5 条，并保持视口位置不跳动。
  async function handleHistoryScroll(event: ReactUIEvent<HTMLDivElement>) {
    const el = event.currentTarget;
    const nextStick = isNearScrollBottom(el);
    if (historyStickRef.current !== nextStick) {
      historyStickRef.current = nextStick;
      setHistoryStick(nextStick);
    }
    if (el.scrollTop < 200 && hasMoreHistory && !historyLoadingRef.current) {
      const prevHeight = el.scrollHeight;
      const added = await loadMoreHistory();
      if (added > 0) {
        window.requestAnimationFrame(() => {
          el.scrollTop = el.scrollHeight - prevHeight;
        });
      }
    }
  }

  return (
    <section className="room-layout">
      <div
        className={`panel room-header ${room.settings.roomBackgroundImage ? "has-room-header-background" : ""}`}
        style={room.settings.roomBackgroundImage ? { "--room-header-bg": `url(${room.settings.roomBackgroundImage})` } as CSSProperties : undefined}
      >
        <div>
          <h2><Swords size={20} /> {room.settings.name} <ExtremeRankedBadge enabled={room.settings.enableExtremeRanked} /> <RankMultiplierBadge multiplier={rankMultiplierForSettings(room.settings)} /></h2>
          {room.settings.enableTags && room.settings.tags?.length ? <RoomTagList tags={room.settings.tags} /> : null}
          <RoomInfoTagList tags={roomInfoTags(config, room)} />
        </div>
        <button className="soft-button" title={leaveTitle} onClick={leaveCurrentRoom}><DoorOpen size={16} /> 离开</button>
      </div>
      {room.settings.gameId !== "liarsdice" && (
        <div className="battle-panel">
          <SeatView seat="A" room={room} me={me} now={now} onSit={() => act("room:sit", { seat: "A" })} />
          <div className="versus">
            <span className="versus-label">⚔️ 对战比分</span>
            <strong className="score-number">{room.score.A} : {room.score.B}</strong>
            {room.settings.gameId === "othello" ? <OthelloScore room={room} /> : room.settings.gameId === "tictactoe" ? <TicTacToeScore room={room} /> : room.settings.gameId === "gomoku" ? <GomokuScore room={room} /> : room.settings.gameId === "jungle" ? <JungleScore room={room} /> : <Settlement room={room} />}
          </div>
          <SeatView seat="B" room={room} me={me} now={now} onSit={() => act("room:sit", { seat: "B" })} />
        </div>
      )}
      <div className="room-content-grid">
        <div className="actions-panel panel">
          {room.settings.gameId === "othello" ? (
            <OthelloPanel room={room} me={me} now={now} onMove={playOthello} onSettle={settleOthelloMove} onRestart={restartOthello} onReady={readyOthello} onRequestSurrender={requestOthelloSurrender} onRespondSurrender={respondOthelloSurrender} onEscape={escapeOthello} />
          ) : room.settings.gameId === "tictactoe" ? (
            <TicTacToePanel room={room} me={me} now={now} onMove={playTicTacToe} onReady={readyTicTacToe} onRestart={restartTicTacToe} onGiveawayChoice={chooseTicTacToeGiveaway} />
          ) : room.settings.gameId === "liarsdice" ? (
            <LiarsDicePanel room={room} me={me} onError={onError} />
          ) : room.settings.gameId === "gomoku" ? (
            <GomokuPanel room={room} me={me} now={now} onError={onError} />
          ) : room.settings.gameId === "jungle" ? (
            <JunglePanel room={room} me={me} now={now} onError={onError} />
          ) : mySeat && (
            <div className="move-panel">
              <div>
                <h3>请选择出拳</h3>
                <p className="hint">{room.phase === "punishment" ? "惩罚完成前不能出拳。" : myChoice ? `你已锁定：${choiceText(myChoice)}` : resultChoice ? `上一局：${choiceText(resultChoice)}，可直接开始下一局。` : canChoose ? "坐下不算出拳，点一个 emoji 才会锁定。" : "等待另一位玩家坐下。"}</p>
                {room.forgiveAdvantageTargetId === me.id && (
                  <p className="hint forgive-advantage-hint">上局对方放过了你，本局你将受到"命运的安排"</p>
                )}
                {room.forgiveAdvantageBeneficiaryId === me.id && (
                  <p className="hint forgive-advantage-hint">上局你放过了对方，本局你将受到"命运的安排"</p>
                )}
              </div>
              <div className="move-row emoji-row">
                <button disabled={!canChoose || Boolean(myChoice)} onClick={() => choose("rock")}>✊<span>锤子</span></button>
                <button disabled={!canChoose || Boolean(myChoice)} onClick={() => choose("scissors")}>✌️<span>剪刀</span></button>
                <button disabled={!canChoose || Boolean(myChoice)} onClick={() => choose("paper")}>🖐️<span>布</span></button>
                {canShowGiveawayButton && <button className="giveaway-move-button" disabled={!canChoose || Boolean(myChoice)} onClick={() => choose("giveaway")}>🫴<span>白给</span></button>}
              </div>
            </div>
          )}
          {(canGoSpectate || canForceGiveaway) && (
            <div className="seat-action-row">
              {canGoSpectate && <button onClick={() => act("room:spectate")}><Eye size={16} /> 去观战席</button>}
              {canForceGiveaway && <button className="force-giveaway-button" onClick={forceGiveaway}><HeartHandshake size={16} /> 强制白给</button>}
            </div>
          )}
          {room.phase === "punishment" && (
            <div className="punish-box">
              <div className="punish-head">
                <span>🎲 惩罚阶段</span>
                <strong>{punishedNames.length ? punishedNames.join("、") : "等待同步"}</strong>
              </div>
              <div className={`punishment-card-grid ${punishedIds.length === 1 ? "single" : "double"}`}>
                {punishedIds.map((playerId) => {
                  const punishedPlayer = roomPlayerById(room, playerId);
                  const proof = (room.proofs || []).find((item) => item.playerId === playerId);
                  const task = room.roundHistory[0]?.punishmentTasks?.find((item) => item.playerId === playerId);
                  const taskAssignerPlayer = task?.assignedBy ? roomPlayerById(room, task.assignedBy) : undefined;
                  const isMine = playerId === me.id;
                  const taskText = proof?.redoTaskText || task?.taskText || "";
                  const taskAssigned = Boolean(taskText.trim());
                  const canAssignTask = Boolean(room.settings.punishmentSource === "player" && task && canAssignPunishmentTask(room, me.id, playerId, task.assignedBy) && !taskAssigned);
                  // 无 status 或 rejected 可提交；pending/approved 不可重复交
                  const canSubmit = isMine && taskAssigned && (!proof || !proof.status || proof.status === "rejected");
                  const canReview = Boolean(canReviewPunishmentProof(room, me.id, playerId) && proof && proof.status !== "approved" && proof.status !== "rejected");
                  const taskCardStyle = task?.backgroundImage ? {
                    "--task-bg": `url(${task.backgroundImage})`,
                    "--task-bg-opacity": String(task.backgroundOpacity ?? 0.22)
                  } as CSSProperties : undefined;
                  return (
                    <div className="punishment-card" key={playerId}>
                      <div className="punishment-card-title">
                        <h4>{punishedPlayer ? <PlayerBadge player={punishedPlayer} compact /> : punishedPlayerName(room, playerId)} {isMine ? "（你）" : ""}</h4>
                        <em>{proof?.status === "approved" ? "已完成" : proof?.status === "pending" ? "待审核" : proof?.status === "rejected" ? "重做中" : taskAssigned ? "待提交" : "等任务"}</em>
                      </div>
                      {taskAssigned && task && (
                        <div className={`task-card designed-task-card ${task.backgroundImage ? "has-task-background" : ""}`} style={taskCardStyle}>
                          <b>{isMine ? "你的任务" : "对方任务"}</b>
                          <p>{taskTextOnly(taskText, task.factionLabel)}</p>
                          {(taskAssignerPlayer || task.assignedByName) && <small>发布者：{taskAssignerPlayer ? displayPlayerName(taskAssignerPlayer) : task.assignedByName}</small>}
                        </div>
                      )}
                      {!taskAssigned && room.settings.punishmentSource === "player" && (
                        <div className="assign-task-box">
                          {isMine && <p className="hint">等待对方发布任务，发布后你就可以提交证明。</p>}
                          {canAssignTask && (
                            <>
                              <p className="hint">请给对方发布一个本局临时惩罚任务。</p>
                              <textarea value={taskInputs[playerId] || ""} onChange={(event) => setTaskInputs((old) => ({ ...old, [playerId]: event.target.value }))} placeholder="例如：面向镜头比个耶并拍照证明" />
                              <button className="primary" onClick={() => assignPunishmentTask(playerId)}>发布任务</button>
                            </>
                          )}
                          {!isMine && !canAssignTask && <p className="hint">等待任务发布。</p>}
                        </div>
                      )}
                      {proof && <PunishmentStatus proof={proof} isMine={isMine} requireConfirm={room.settings.requireOpponentConfirm} />}
                      {taskAssigned && task?.backgroundImage && (
                        <button
                          type="button"
                          className="history-proof-image-button punishment-task-image-button"
                          title="点击查看完整任务图"
                          onClick={() => setPreviewImage(task.backgroundImage!)}
                        >
                          <img src={task.backgroundImage} alt="任务图片" loading="lazy" decoding="async" />
                        </button>
                      )}
                      {proof?.text && (
                        <div className="proof">
                          <b>已提交证明</b>
                          <p>{proof.text}</p>
                          {proof.imageUrl && (
                            <button type="button" className="history-proof-image-button" title="点击放大" onClick={() => setPreviewImage(proof.imageUrl!)}>
                              <ProofImage src={proof.imageUrl} alt="惩罚证明" />
                            </button>
                          )}
                        </div>
                      )}
                      {canSubmit && (
                        <div className="proof-submit-card">
                          <b>{proof?.status === "rejected" ? "重新提交证明" : "提交完成证明"}</b>
                          <textarea value={proofText} onChange={(event) => setProofText(event.target.value)} placeholder={proof?.status === "rejected" ? "重新提交你的惩罚完成证明" : "写下你的惩罚完成证明"} />
                          <p className="hint">文字证明必须填写；照片证明可以不交。</p>
                          {room.settings.allowProofImage !== false ? (
                            <>
                              <label className="upload">
                                <Upload size={16} /> 上传图片证明
                                {/* Chrome 的 accept="image/*" 通配符会用内置图片扩展名表过滤文件选择器，
                                    该表不含 heic/heif，导致即使同时列了 .heic/.heif 也可能被隐藏；
                                    这里去掉通配符、只列具体扩展名/MIME，纯粹影响系统选择器的可见文件，
                                    不改变选完文件之后的校验/压缩/上传流程。 */}
                                <input type="file" accept=".jpg,.jpeg,.png,.webp,.heic,.heif,.gif,.bmp,image/jpeg,image/png,image/webp,image/gif,image/bmp" onChange={(event) => event.target.files?.[0] && uploadImage(event.target.files[0]).catch((error) => onError(error.message))} />
                              </label>
                              {proofImage && (
                                <button type="button" className="history-proof-image-button" title="点击放大" onClick={() => setPreviewImage(proofImage)}>
                                  <ProofImage className="proof-preview" src={proofImage} alt="惩罚证明" />
                                </button>
                              )}
                            </>
                          ) : (
                            <p className="hint">本房间关闭了图片证明，只需要提交文字证明。</p>
                          )}
                          <button
                            type="button"
                            className="primary"
                            disabled={proofSubmitting}
                            onClick={() => { void submitProof(); }}
                          >
                            {proofSubmitting ? "提交中…" : "提交证明"}
                          </button>
                        </div>
                      )}
                      {canReview && (
                        <div className="review-box">
                          <div className="toolbar">
                            <button className="primary" onClick={() => reviewProof(playerId, "approve")}>确认完成</button>
                            <button onClick={() => reviewProof(playerId, "forgive")}>放过对方</button>
                          </div>
                          <p className="hint">放过对方后，对方下一局可能会受到一点命运安排。双方互相放过时会互相抵消。</p>
                          <textarea value={redoInputs[playerId] || ""} onChange={(event) => setRedoInputs((old) => ({ ...old, [playerId]: event.target.value }))} placeholder="不通过时，给对方发布一个新任务" />
                          <button onClick={() => reviewProof(playerId, "reject")}>不通过，发布新任务</button>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          )}
        </div>
        <div className="room-side-stack">
          <div className={`panel room-player-panel ${roomPlayersCollapsed ? "collapsed" : ""}`}>
            <h3 className="sticky-panel-title">
              房间玩家名单
              <span className="panel-title-actions">
                <span>{roomPlayers.length} 人</span>
                <CollapseToggle collapsed={roomPlayersCollapsed} onToggle={toggleRoomPlayersCollapsed} label="房间玩家名单" />
              </span>
            </h3>
            <div className={`room-player-list mobile-collapsible-body ${roomPlayersCollapsed ? "collapsed" : ""}`}>
              {roomPlayers.map((item) => <RoomPlayerRow key={`${item.role}-${item.player.id}`} player={item.player} role={item.role} now={now} />)}
              {roomPlayers.length === 0 && <p className="empty">暂无真人玩家</p>}
            </div>
          </div>
          <ChatBoardShell
            title={chatTab === "room" ? "房间聊天" : "大厅聊天"}
            collapseKey="roomChat"
            tabs={
              <div className="segmented chat-tabs">
                <button className={chatTab === "room" ? "active" : ""} onClick={() => setChatTab("room")}>本房间</button>
                <button className={chatTab === "lobby" ? "active" : ""} onClick={() => setChatTab("lobby")}>大厅</button>
              </div>
            }
          >
            {chatTab === "room" ? (
              <ChatPanel
                key={`room-${room.id}`}
                scope={room.id}
                me={me}
                players={chatPlayers}
                onError={onError}
                placeholder="发一句话...（点头像 @ 人）"
                emptyText="还没有房间聊天"
                messagesClass="room-chat-messages"
              />
            ) : (
              <ChatPanel
                key="lobby-readonly"
                scope=""
                me={me}
                players={chatPlayers}
                onError={onError}
                readOnly
                readOnlyHint="大厅聊天内容在房间内只能查看，回到大厅后可以发送。"
                emptyText="大厅还没有留言"
                subscribeLobbyChannel
                messagesClass="room-chat-messages"
              />
            )}
          </ChatBoardShell>
        </div>
        <div className={`panel round-history ${roundHistoryCollapsed ? "collapsed" : ""}`}>
          <h3 className="sticky-panel-title">
            📜 对局记录
            <span className="panel-title-actions">
              <span>{room.roundHistoryTotal || 0} 局</span>
              <CollapseToggle collapsed={roundHistoryCollapsed} onToggle={toggleRoundHistoryCollapsed} label="对局记录" />
            </span>
          </h3>
          <div className={`mobile-collapsible-body ${roundHistoryCollapsed ? "collapsed" : ""}`}>
            <div className="chat-scroll-shell">
              <div className="round-history-list" ref={historyListRef} onScroll={handleHistoryScroll}>
                {hasMoreHistory && <div className="chat-more-hint">{historyLoading ? "加载中…" : "↑ 上滑加载更早记录"}</div>}
                {orderedRoundHistory.map((item) => <RoundHistoryCard key={item.id} item={item} onOpenImage={setPreviewImage} />)}
                {visibleRoundHistory.length === 0 && <p className="empty">还没有对局记录</p>}
              </div>
              {!historyStick && visibleRoundHistory.length > 0 && (
                <button type="button" className="chat-stick-button" onClick={() => stickChatToBottom(historyListRef.current, historyStickRef, setHistoryStick)}>
                  ↓ 回到底部
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
      {previewImage && (
        <div className="modal-backdrop image-preview-backdrop" onClick={() => setPreviewImage(null)}>
          <div className="image-preview-modal" onClick={(event) => event.stopPropagation()}>
            <button className="icon-button image-preview-close" onClick={() => setPreviewImage(null)}>×</button>
            <img src={previewImage} alt="惩罚证明大图" loading="lazy" />
          </div>
        </div>
      )}
    </section>
  );
}

export function OthelloScore({ room }: { room: RoomSnapshot }) {
  const state = room.othello;
  if (!state) {
    const bothReady = room.ready.A && room.ready.B;
    return <p className="settlement-placeholder">{bothReady ? "正在随机先手" : "等待准备"}</p>;
  }
  return (
    <div className="othello-score-mini">
      <span>⚫ {state.blackCount}</span>
      <span>⚪ {state.whiteCount}</span>
      {room.settings.enableRanked && state.rankedDelta && <span className="othello-live-rank">黑 {othelloDeltaText(state, "black")} / 白 {othelloDeltaText(state, "white")}</span>}
      <strong>{state.ended ? room.resultText || "对局结束" : `轮到${state.blackSeat === state.turn ? "黑棋" : "白棋"}`}</strong>
    </div>
  );
}

export function TicTacToeScore({ room }: { room: RoomSnapshot }) {
  const state = room.tictactoe;
  if (!state) {
    const bothReady = room.ready.A && room.ready.B;
    return <p className="settlement-placeholder">{bothReady ? "正在随机先手" : "等待准备"}</p>;
  }
  return (
    <div className="tictactoe-score-mini">
      <span>❌ {state.xSeat === "A" ? "A" : "B"}</span>
      <span>⭕ {state.xSeat === "A" ? "B" : "A"}</span>
      {room.settings.enableRanked && state.rankedDelta && <span className="tictactoe-live-rank">X {tictactoeDeltaText(state, "X")} / O {tictactoeDeltaText(state, "O")}</span>}
      <strong>{state.ended ? room.resultText || "对局结束" : `轮到 ${state.xSeat === state.turn ? "X" : "O"}`}</strong>
    </div>
  );
}

export function TicTacToePanel({ room, me, now, onMove, onReady, onRestart, onGiveawayChoice }: { room: RoomSnapshot; me: PublicPlayer; now: number; onMove: (row: number, col: number) => void; onReady: () => void; onRestart: () => void; onGiveawayChoice: (mode: "normal" | "giveaway") => void }) {
  const state = room.tictactoe;
  const mySeat = room.seats.A?.id === me.id ? "A" : room.seats.B?.id === me.id ? "B" : null;
  const giveawayPrompt = state?.giveawayPrompt;
  const giveawayBlockingTurn = Boolean(giveawayPrompt && giveawayPrompt.seat === state?.turn);
  const isMyGiveawayPrompt = Boolean(state && mySeat && giveawayPrompt?.seat === mySeat && room.phase === "choosing" && !state.ended);
  const isMyTurn = Boolean(state && mySeat && state.turn === mySeat && room.phase === "choosing" && !state.ended && !giveawayBlockingTurn);
  const waitingForReady = room.phase === "ready" && Boolean(room.seats.A && room.seats.B);
  const drawingFirst = waitingForReady && room.ready.A && room.ready.B;
  const myReady = mySeat ? room.ready[mySeat] : false;
  const turnName = state?.turn === "A" ? occupantDisplay(room.seats.A) : occupantDisplay(room.seats.B);
  const giveawayPromptName = giveawayPrompt?.seat === "A" ? occupantDisplay(room.seats.A) : occupantDisplay(room.seats.B);
  const giveawaySecondsLeft = giveawayPrompt ? Math.max(0, Math.ceil((giveawayPrompt.expiresAt - now) / 1000)) : 0;
  const tictactoeGiveawayGain = formatGiveawayValue(0.3);
  const xSeat = state?.xSeat;
  const oSeat = xSeat ? xSeat === "A" ? "B" : "A" : null;
  const winningKeys = new Set((state?.winningLine || []).map((cell) => `${cell.row}-${cell.col}`));
  const board = state?.board || Array.from({ length: 3 }, () => Array.from({ length: 3 }, () => null));
  const boardTheme = tictactoeThemeStyle(room.settings.tictactoeBoardTheme);
  return (
    <div className="tictactoe-panel">
      <div className="tictactoe-head">
        <div>
          <h3>❌⭕ 井字棋</h3>
          <p className="hint">
            {!room.seats.A || !room.seats.B
              ? "等待两个战斗席坐满。"
              : drawingFirst
                ? "正在随机 X/O 先手..."
                : waitingForReady
                  ? "双方准备后随机决定谁执 X 先手。"
                  : state?.ended
                    ? room.resultText || "对局结束"
                    : giveawayPrompt?.forced
                      ? "强制白给中，系统正在随机落子..."
                      : giveawayPrompt
                        ? isMyGiveawayPrompt ? "请选择不白给或白给落子。" : `等待 ${giveawayPromptName} 选择是否白给。`
                        : isMyTurn ? "轮到你落子。" : `轮到 ${turnName} 落子。`}
          </p>
        </div>
        {state && (
          <div className="tictactoe-turn-card">
            <span>{state.xSeat === state.turn ? "❌ X" : "⭕ O"}</span>
            <strong>{state.moveCount} / 9</strong>
            {room.settings.enableRanked && state.rankedDelta && <small>本局排位：X {tictactoeDeltaText(state, "X")} / O {tictactoeDeltaText(state, "O")}</small>}
          </div>
        )}
      </div>
      {waitingForReady && (
        <div className={`tictactoe-ready-card ${drawingFirst ? "drawing" : ""}`}>
          <div className="tictactoe-draw-animation" aria-hidden="true">
            <span>❌</span>
            <span>⭕</span>
            <span>❌</span>
          </div>
          <div>
            <strong>{drawingFirst ? "抽签中..." : myReady ? "你已准备" : "准备开始"}</strong>
            <p className="hint">A：{room.ready.A ? "已准备" : "未准备"} · B：{room.ready.B ? "已准备" : "未准备"}</p>
          </div>
          {mySeat && !myReady && !drawingFirst && <button className="primary" onClick={onReady}>准备</button>}
          {mySeat && myReady && !drawingFirst && <button disabled>等待对方</button>}
        </div>
      )}
      {giveawayPrompt && room.phase === "choosing" && !state?.ended && (
        <div className={`othello-settlement-card tictactoe-giveaway-card ${giveawayPrompt.forced ? "forced" : ""}`}>
          <div>
            <strong>{giveawayPrompt.forced ? "强制白给" : isMyGiveawayPrompt ? "选择本手结算" : "等待本手结算"}</strong>
            <p className="hint">
              {giveawayPrompt.forced
                ? `${giveawayPromptName} 触发强制白给，将在 ${giveawaySecondsLeft} 秒后自动随机乱下。`
                : isMyGiveawayPrompt
                  ? `${giveawaySecondsLeft} 秒后自动选择不白给。`
                  : `等待 ${giveawayPromptName} 选择，${giveawaySecondsLeft} 秒后默认不白给。`}
            </p>
            <p className="othello-settlement-help">
              不白给：正常手动落子；白给：系统随机选择一个空格落子，白给值 +{tictactoeGiveawayGain}%。
            </p>
          </div>
          {giveawayPrompt.forced ? (
            <div className="othello-settlement-forced">
              <span>🫴 白给乱下</span>
            </div>
          ) : isMyGiveawayPrompt ? (
            <div className="othello-settlement-actions tictactoe-giveaway-actions">
              <button className="primary" onClick={() => onGiveawayChoice("normal")}>不白给</button>
              <button className="soft-button" onClick={() => onGiveawayChoice("giveaway")}>白给 +{tictactoeGiveawayGain}%</button>
            </div>
          ) : (
            <div className="othello-settlement-forced">
              <span>⏳ 等待选择</span>
            </div>
          )}
        </div>
      )}
      {state?.ended && mySeat && room.phase === "result" && <button className="primary tictactoe-restart-button" onClick={onRestart}>再来一局</button>}
      <div className="tictactoe-board" role="grid" aria-label="井字棋棋盘" style={boardTheme}>
        {board.map((row, rowIndex) => row.map((cell, colIndex) => {
          const winning = winningKeys.has(`${rowIndex}-${colIndex}`);
          return (
            <button
              type="button"
              className={`tictactoe-cell ${cell ? cell.toLowerCase() : ""} ${winning ? "winning" : ""}`}
              key={`${rowIndex}-${colIndex}`}
              disabled={!isMyTurn || Boolean(cell)}
              onClick={() => onMove(rowIndex, colIndex)}
              aria-label={`第 ${rowIndex + 1} 行第 ${colIndex + 1} 列`}
            >
              {cell ? (cell === "X" ? "×" : "○") : ""}
            </button>
          );
        }))}
      </div>
      <div className="tictactoe-legend">
        <span>❌ X：{xSeat ? occupantDisplay(room.seats[xSeat]) : "准备后随机"}</span>
        <span>⭕ O：{oSeat ? occupantDisplay(room.seats[oSeat]) : "准备后随机"}</span>
        <span>{mySeat ? `你在战斗席 ${mySeat}` : "你正在观战"}</span>
      </div>
    </div>
  );
}

export function tictactoeDeltaText(state: NonNullable<RoomSnapshot["tictactoe"]>, mark: "X" | "O") {
  const seat = mark === "X" ? state.xSeat : state.xSeat === "A" ? "B" : "A";
  const delta = state.rankedDelta?.[seat] || 0;
  return `${delta >= 0 ? "+" : ""}${delta}`;
}

function formatClockMs(ms: number): string {
  const totalSeconds = Math.max(0, Math.ceil(ms / 1000));
  const m = Math.floor(totalSeconds / 60);
  const s = totalSeconds % 60;
  return `${m}:${s.toString().padStart(2, "0")}`;
}

type TimedGameState = {
  turn: SeatKey;
  blackSeat: SeatKey;
  ended?: boolean;
  moveDeadlineAt?: number;
  clockDeadlineAt?: number;
  clockRemaining?: Record<SeatKey, number>;
};

/** 黑白棋/五子棋/斗兽棋共用：棋盘上方的每子/每局倒计时条，未启用对应计时器时不显示。 */
export function GameClockBar({ room, state, moveSeconds, gameMinutes, now, labels }: {
  room: RoomSnapshot;
  state?: TimedGameState;
  moveSeconds?: number;
  gameMinutes?: number;
  now: number;
  /** 双方总时长标签；默认 ⚫/⚪（黑白棋、五子棋） */
  labels?: { primary: string; secondary: string };
}) {
  if (!state || state.ended || room.phase !== "choosing" || (!moveSeconds && !gameMinutes)) return null;
  const whiteSeat: SeatKey = state.blackSeat === "A" ? "B" : "A";
  const primary = labels?.primary ?? "⚫";
  const secondary = labels?.secondary ?? "⚪";
  const moveSecondsLeft = moveSeconds && state.moveDeadlineAt ? Math.max(0, Math.ceil((state.moveDeadlineAt - now) / 1000)) : null;
  function remainingMsFor(seat: SeatKey): number {
    if (state!.turn === seat && state!.clockDeadlineAt) return Math.max(0, state!.clockDeadlineAt - now);
    return state!.clockRemaining?.[seat] ?? 0;
  }
  return (
    <div className="game-clock-bar">
      {moveSeconds && moveSecondsLeft !== null && (
        <span className={`game-clock-move ${moveSecondsLeft <= 10 ? "urgent" : ""}`}>本步剩余 {moveSecondsLeft} 秒</span>
      )}
      {Boolean(gameMinutes) && (
        <span className="game-clock-total">
          {primary} {formatClockMs(remainingMsFor(state.blackSeat))} · {secondary} {formatClockMs(remainingMsFor(whiteSeat))}
        </span>
      )}
    </div>
  );
}

export function OthelloPanel({ room, me, now, onMove, onSettle, onRestart, onReady, onRequestSurrender, onRespondSurrender, onEscape }: { room: RoomSnapshot; me: PublicPlayer; now: number; onMove: (row: number, col: number) => void; onSettle: (mode: "normal" | "giveaway" | "tribute") => void; onRestart: () => void; onReady: () => void; onRequestSurrender: () => void; onRespondSurrender: (accept: boolean) => void; onEscape: () => void }) {
  const state = room.othello;
  const boardTheme = othelloThemeStyle(room.settings.othelloBoardTheme);
  const mySeat = room.seats.A?.id === me.id ? "A" : room.seats.B?.id === me.id ? "B" : null;
  const pending = state?.pendingSettlement;
  const isMyTurn = Boolean(state && mySeat && state.turn === mySeat && room.phase === "choosing" && !state.ended && !pending);
  const legalKeys = new Set((state?.legalMoves || []).map((move) => `${move.row}-${move.col}`));
  const turnName = state?.turn === "A" ? occupantDisplay(room.seats.A) : occupantDisplay(room.seats.B);
  const waitingForReady = room.phase === "ready" && Boolean(room.seats.A && room.seats.B);
  const drawingFirst = waitingForReady && room.ready.A && room.ready.B;
  const myReady = mySeat ? room.ready[mySeat] : false;
  const canSurrender = Boolean(mySeat && state && room.phase === "choosing" && !state.ended && !pending);
  const surrenderRequest = state?.surrenderRequest;
  const surrenderFromMe = Boolean(mySeat && surrenderRequest?.fromSeat === mySeat);
  const surrenderToMe = Boolean(mySeat && surrenderRequest?.toSeat === mySeat);
  const surrenderFromName = surrenderRequest ? occupantDisplay(room.seats[surrenderRequest.fromSeat]) : "";
  const blackSeat = state?.blackSeat;
  const whiteSeat = blackSeat ? (blackSeat === "A" ? "B" : "A") : null;
  return (
    <div className="othello-panel">
      <div className="othello-head">
        <div>
          <h3>⚫⚪ 黑白棋</h3>
          <p className="hint">
            {!room.seats.A || !room.seats.B
              ? "等待两个战斗席坐满。"
              : drawingFirst
                ? "正在随机执黑先手..."
                : waitingForReady
                  ? "双方准备后随机决定谁执黑先手。"
                  : state?.ended
                    ? room.resultText || "对局结束"
                    : pending ? "正在结算本手白给/上贡。" : isMyTurn ? "轮到你落子。" : `轮到 ${turnName} 落子。`}
          </p>
        </div>
        {state && (
          <div className="othello-turn-card">
            <span>{state.blackSeat === state.turn ? "⚫ 黑棋" : "⚪ 白棋"}</span>
            <strong>{state.blackCount} : {state.whiteCount}</strong>
            {room.settings.enableRanked && state.rankedDelta && <small>本局排位：黑 {othelloDeltaText(state, "black")} / 白 {othelloDeltaText(state, "white")}</small>}
          </div>
        )}
      </div>
      {waitingForReady && (
        <div className={`othello-ready-card ${drawingFirst ? "drawing" : ""}`}>
          <div className="othello-draw-animation" aria-hidden="true">
            <span>⚫</span>
            <span>⚪</span>
            <span>⚫</span>
          </div>
          <div>
            <strong>{drawingFirst ? "抽签中..." : myReady ? "你已准备" : "准备开始"}</strong>
            <p className="hint">A：{room.ready.A ? "已准备" : "未准备"} · B：{room.ready.B ? "已准备" : "未准备"}</p>
          </div>
          {mySeat && !myReady && !drawingFirst && <button className="primary" onClick={onReady}>准备</button>}
          {mySeat && myReady && !drawingFirst && <button disabled>等待对方</button>}
        </div>
      )}
      {state?.ended && mySeat && room.phase === "result" && <button className="primary othello-restart-button" onClick={onRestart}>再来一局</button>}
      {canSurrender && surrenderRequest && (
        <div className={`othello-surrender-card ${surrenderToMe ? "needs-action" : ""}`}>
          <div>
            <strong>{surrenderFromMe ? "已申请认输" : `${surrenderFromName} 申请认输`}</strong>
            <p className="hint">{surrenderFromMe ? "等待对方确认；对局状态会保持不变。" : "你可以同意结束本局，或拒绝后继续下棋。"}</p>
          </div>
          {surrenderToMe && (
            <div className="othello-surrender-actions">
              <button className="primary" onClick={() => onRespondSurrender(true)}>同意认输</button>
              <button className="soft-button" onClick={() => onRespondSurrender(false)}>拒绝，继续下棋</button>
            </div>
          )}
        </div>
      )}
      {canSurrender && (
        <div className="othello-risk-actions">
          <button className="soft-button othello-surrender-button" disabled={Boolean(surrenderRequest)} onClick={onRequestSurrender}>
            申请认输
          </button>
          <button className="soft-button danger-soft othello-surrender-button" onClick={onEscape}>
            逃跑
          </button>
        </div>
      )}
      {pending && (
        <OthelloSettlementCard
          room={room}
          me={me}
          pending={pending}
          now={now}
          onSettle={onSettle}
        />
      )}
      <GameClockBar room={room} state={state} moveSeconds={room.settings.othelloMoveSeconds} gameMinutes={room.settings.othelloGameMinutes} now={now} />
      <div className="othello-board" role="grid" aria-label="黑白棋棋盘" style={boardTheme}>
        {(state?.board || Array.from({ length: 8 }, () => Array.from({ length: 8 }, () => null))).map((row, rowIndex) => row.map((cell, colIndex) => {
          const legal = legalKeys.has(`${rowIndex}-${colIndex}`);
          return (
            // 棋子/落子提示挪到 button 外层作为兄弟节点：button 的 disabled 变暗
            // (opacity:.6) 只应影响格子自身，不应连带压暗棋子，两者需要独立的合成层级。
            <div className="othello-cell-wrap" key={`${rowIndex}-${colIndex}`}>
              <button
                type="button"
                className={`othello-cell ${cell || ""} ${legal ? "legal" : ""}`}
                disabled={!isMyTurn || !legal}
                onClick={() => onMove(rowIndex, colIndex)}
                aria-label={`第 ${rowIndex + 1} 行第 ${colIndex + 1} 列`}
              />
              {cell && <span className={`othello-disc ${cell}`} />}
              {!cell && legal && <span className="othello-legal-dot" />}
            </div>
          );
        }))}
      </div>
      <div className="othello-legend">
        <span>⚫ 黑棋：{blackSeat ? occupantDisplay(room.seats[blackSeat]) : "准备后随机"}</span>
        <span>⚪ 白棋：{whiteSeat ? occupantDisplay(room.seats[whiteSeat]) : "准备后随机"}</span>
        <span>{mySeat ? `你在战斗席 ${mySeat}` : "你正在观战"}</span>
      </div>
    </div>
  );
}

export function othelloThemeStyle(themeId?: RoomSettings["othelloBoardTheme"]): CSSProperties {
  const theme = othelloBoardThemes.find((item) => item.id === themeId) || othelloBoardThemes[0];
  return {
    "--othello-board": theme.board,
    "--othello-cell": theme.cell,
    "--othello-line": theme.line,
    "--othello-hover": theme.hover,
    "--othello-border": theme.border,
    "--othello-black-disc": theme.blackDisc,
    "--othello-white-disc": theme.whiteDisc,
    "--othello-black-ring": theme.blackRing,
    "--othello-white-ring": theme.whiteRing
  } as CSSProperties;
}

export function tictactoeThemeStyle(themeId?: RoomSettings["tictactoeBoardTheme"]): CSSProperties {
  const theme = tictactoeBoardThemes.find((item) => item.id === themeId) || tictactoeBoardThemes[0];
  return {
    "--ttt-board": theme.board,
    "--ttt-cell": theme.cell,
    "--ttt-line": theme.line,
    "--ttt-hover": theme.hover,
    "--ttt-border": theme.border,
    "--ttt-x": theme.x,
    "--ttt-o": theme.o,
    "--ttt-win": theme.win
  } as CSSProperties;
}

export function othelloDeltaText(state: NonNullable<RoomSnapshot["othello"]>, color: "black" | "white") {
  const seat = color === "black" ? state.blackSeat : state.blackSeat === "A" ? "B" : "A";
  const delta = state.rankedDelta?.[seat] || 0;
  return `${delta >= 0 ? "+" : ""}${delta}`;
}

export function OthelloSettlementCard({ room, me, pending, now, onSettle }: { room: RoomSnapshot; me: PublicPlayer; pending: NonNullable<NonNullable<RoomSnapshot["othello"]>["pendingSettlement"]>; now: number; onSettle: (mode: "normal" | "giveaway" | "tribute") => void }) {
  const isMine = room.seats[pending.seat]?.id === me.id;
  const actorName = occupantDisplay(room.seats[pending.seat]);
  const opponentName = occupantDisplay(room.seats[pending.opponentSeat]);
  const secondsLeft = Math.max(0, Math.ceil((pending.expiresAt - now) / 1000));
  const forcedText = pending.forced === "tribute" ? "强制上贡" : pending.forced === "giveaway" ? "强制白给" : "";
  const giveawayGain = formatGiveawayValue(pending.flips * 0.1);
  const tributeGain = formatGiveawayValue(pending.flips * 0.2);
  return (
    <div className={`othello-settlement-card ${pending.forced ? "forced" : ""}`}>
      <div>
        <strong>{pending.forced ? forcedText : isMine ? "选择本手结算" : "等待本手结算"}</strong>
        <p className="hint">
          {actorName} 本手翻 {pending.flips} 子，基础分 {pending.stake}。
          {pending.forced ? ` ${forcedText}将在 ${secondsLeft} 秒后自动结算。` : isMine ? ` ${secondsLeft} 秒后自动选择不白给。` : ` 等待 ${actorName} 选择，${secondsLeft} 秒后默认不白给。`}
        </p>
        <p className="othello-settlement-help">
          不白给：本手按 {pending.stake} 分正常结算；白给：本手不结算排位分，白给值 +{giveawayGain}%；上贡：对方拿本手收益，你的白给值 +{tributeGain}%。
        </p>
      </div>
      {pending.forced ? (
        <div className="othello-settlement-forced">
          <span>{pending.forced === "tribute" ? "🎁 上贡给对方" : "🫴 白给本手"}</span>
        </div>
      ) : isMine ? (
        <div className="othello-settlement-actions">
          <button className="primary" onClick={() => onSettle("normal")}>不白给</button>
          <button className="soft-button" onClick={() => onSettle("giveaway")}>白给 +{giveawayGain}%</button>
          <button className="soft-button danger-soft" onClick={() => onSettle("tribute")}>上贡给 {opponentName} +{tributeGain}%</button>
        </div>
      ) : (
        <div className="othello-settlement-forced">
          <span>⏳ 等待选择</span>
        </div>
      )}
    </div>
  );
}

export function occupantDisplay(occupant: SeatOccupant) {
  if (!occupant) return "空位";
  return displayPlayerName(occupant);
}

export function RoundHistoryCard({ item, onOpenImage }: { item: RoomSnapshot["roundHistory"][number]; onOpenImage: (imageUrl: string) => void }) {
  const safe = normalizeRoundHistoryItem(item);
  const proofByPlayer = new Map(safe.proofs.map((proof) => [proof.playerId, proof]));
  const taskPlayerIds = new Set(safe.punishmentTasks.map((task) => task.playerId));
  const looseProofs = safe.proofs.filter((proof) => !taskPlayerIds.has(proof.playerId));
  return (
    <article className="history-card">
      <header className="history-card-head">
        <div>
          <b>第 {safe.round} 局</b>
          <small>{new Date(safe.at).toLocaleTimeString()}</small>
        </div>
        <div className="history-tags">
          {safe.gameId === "othello" && <em>⚫⚪ 黑白棋</em>}
          {safe.gameId === "tictactoe" && <em>❌⭕ 井字棋</em>}
          {safe.gameId === "gomoku" && <em>●○ 五子棋</em>}
          {safe.gameId === "jungle" && <em>🦁 斗兽棋</em>}
          {safe.gameId === "liarsdice" && <em>🎲 大话骰</em>}
          {safe.ranked && <em>🏆 {safe.gameId === "othello" ? `${safe.stake}分/子${safe.rankMultiplier && safe.rankMultiplier > 1 ? ` ×${safe.rankMultiplier}` : ""}` : `${safe.stake}分${safe.rankMultiplier && safe.rankMultiplier > 1 ? ` ×${safe.rankMultiplier}` : ""}`}</em>}
          {safe.extremeRanked && <em>⚡ 极限</em>}
          {safe.punishedNames.length > 0 && <em>🎲 惩罚</em>}
        </div>
      </header>
      <div className="history-duel">
        <div className="history-side">
          <span>{safe.playerA}</span>
          <strong>{historySeatLabel(safe, "A")}</strong>
        </div>
        <div className="history-result">
          <small>{safe.gameId === "othello" && safe.othelloScore ? `${safe.othelloScore.black} : ${safe.othelloScore.white}` : safe.gameId === "tictactoe" ? "3 × 3" : safe.gameId === "gomoku" ? "15 × 15" : safe.gameId === "jungle" ? "7 × 9" : "VS"}</small>
          <b>{safe.resultLabel || historyResultText(safe.result)}</b>
        </div>
        <div className="history-side">
          <span>{safe.playerB}</span>
          <strong>{historySeatLabel(safe, "B")}</strong>
        </div>
      </div>
      {safe.punishedNames.length > 0 && (
        <section className="history-section">
          <div className="history-punishment-summary">
            <b>{safe.punishmentName || "惩罚"}</b>
            <small>{safe.punishedNames.join("、")}</small>
          </div>
          {safe.punishmentTasks.map((task) => {
            const proof = proofByPlayer.get(task.playerId);
            return (
              <div
                className={`history-task ${task.backgroundImage ? "has-task-background" : ""}`}
                key={`${safe.id}-${task.playerId}-task`}
                style={task.backgroundImage ? { "--task-bg": `url(${task.backgroundImage})`, "--task-bg-opacity": String(task.backgroundOpacity ?? 0.22) } as CSSProperties : undefined}
              >
                <small>{task.playerName} 的任务{task.assignedByName ? ` · ${task.assignedByName} 发布` : ""}</small>
                <p>{task.taskText ? taskTextOnly(task.taskText, task.factionLabel) : "等待玩家发布任务"}</p>
                {proof ? (
                  <div className="history-proof inline">
                    <span>完成证明 · {historyProofStatusLabel(proof)}</span>
                    {proof.taskText && proof.taskText !== task.taskText && <small>对应任务：{proof.taskText}</small>}
                    <p>{proof.text || "（无文字）"}</p>
                    {proof.rejectReason && <small>审核备注：{proof.rejectReason}</small>}
                    {proof.redoTaskText && <small>重做任务：{proof.redoTaskText}</small>}
                    {proof.imageUrl && (
                      <button className="history-proof-image-button" onClick={() => onOpenImage(proof.imageUrl!)}>
                        <ProofImage src={proof.imageUrl} alt="惩罚证明" />
                      </button>
                    )}
                  </div>
                ) : (
                  <div className="history-proof inline muted"><span>完成证明</span><p>尚未提交</p></div>
                )}
              </div>
            );
          })}
        </section>
      )}
      {looseProofs.length > 0 && (
        <section className="history-section">
          <b>其它完成证明</b>
          {looseProofs.map((proof) => (
            <div className="history-proof" key={`${safe.id}-${proof.playerId}`}>
              <span>{proof.playerName || "玩家"} · {historyProofStatusLabel(proof)}</span>
              {proof.taskText && <small>任务：{proof.taskText}</small>}
              <p>{proof.text || "（无文字）"}</p>
              {proof.rejectReason && <small>审核备注：{proof.rejectReason}</small>}
              {proof.imageUrl && (
                <button className="history-proof-image-button" onClick={() => onOpenImage(proof.imageUrl!)}>
                  <ProofImage src={proof.imageUrl} alt="惩罚证明" />
                </button>
              )}
            </div>
          ))}
        </section>
      )}
    </article>
  );
}

export function historySeatLabel(item: RoomSnapshot["roundHistory"][number], seat: SeatKey) {
  if (item.gameId === "othello") {
    return item.othelloBlackSeat === seat ? "⚫ 黑棋" : "⚪ 白棋";
  }
  if (item.gameId === "tictactoe") {
    return item.tictactoeXSeat === seat ? "❌ X" : "⭕ O";
  }
  if (item.gameId === "gomoku") {
    return item.gomokuBlackSeat === seat ? "⚫ 黑棋" : "⚪ 白棋";
  }
  if (item.gameId === "jungle") {
    return jungleSideLabel(seat);
  }
  if (item.gameId === "liarsdice") {
    // 大话骰对局记录里 playerA 固定是本局赢家、playerB 固定是输家（见后端 game_liarsdice.go）。
    return seat === "A" ? "🏆 胜" : "💤 负";
  }
  return choiceText(seat === "A" ? item.moveA : item.moveB);
}

export function historyProofStatusLabel(proof: { status?: string; confirmedBy?: string; rejectReason?: string }) {
  if (proof.status === "approved" || proof.confirmedBy) {
    if (proof.rejectReason === "对方选择放过你" || proof.rejectReason === "双方互相放过，下一局正常开始。") return "已放过";
    return "已通过";
  }
  if (proof.status === "rejected") return "需重做";
  if (proof.status === "pending") return "待审核";
  return proof.status || "已提交";
}

// canReviewPunishmentProof 与后端 canReviewPlayer 对齐：座位制=不同座位的对手；
// 大话骰=本局赢家审核本局输家（以最近一条对局记录为准）。
export function canReviewPunishmentProof(room: RoomSnapshot, reviewerId: string, targetId: string) {
  if (!reviewerId || reviewerId === targetId) return false;
  if (room.settings.gameId === "liarsdice") {
    const latest = room.roundHistory[0];
    return Boolean(latest && latest.liarsDiceWinnerId === reviewerId && latest.liarsDiceLoserId === targetId);
  }
  const reviewerSeat = room.seats.A?.id === reviewerId ? "A" : room.seats.B?.id === reviewerId ? "B" : null;
  const targetSeat = room.seats.A?.id === targetId ? "A" : room.seats.B?.id === targetId ? "B" : null;
  return Boolean(reviewerSeat && targetSeat && reviewerSeat !== targetSeat);
}

export function canAssignPunishmentTask(room: RoomSnapshot, currentPlayerId: string, punishedPlayerId: string, assignedBy?: string) {
  if (assignedBy) return assignedBy === currentPlayerId;
  if (room.settings.gameId === "liarsdice") {
    const winnerId = room.roundHistory[0]?.liarsDiceWinnerId;
    return Boolean(winnerId && winnerId === currentPlayerId && winnerId !== punishedPlayerId);
  }
  const punishedSeat = room.seats.A?.id === punishedPlayerId ? "A" : room.seats.B?.id === punishedPlayerId ? "B" : null;
  if (!punishedSeat) return false;
  const opponent = punishedSeat === "A" ? room.seats.B : room.seats.A;
  return Boolean(opponent && opponent.id === currentPlayerId);
}

export function RoomPlayerRow({ player, role, now }: { player: PublicPlayer; role: string; now: number }) {
  const stats = safePlayerStats(player);
  return (
    <div className="room-player-row">
      <div className="room-player-main">
        <div className="room-player-identity">
          <PlayerAvatar player={player} size={28} />
          <PlayerBadge player={player} />
        </div>
        <div className="room-player-tags">
          <em>{role}</em>
          <OfflineBadge player={player} now={now} />
        </div>
      </div>
      <small className="room-player-stats">
        全局：{stats.wins}胜 {stats.losses}负 {stats.draws}平 · {stats.punishments}惩罚 · 胜率 {winRateText(player)} · {stats.rankedPoints}分
      </small>
    </div>
  );
}

export function OfflineBadge({ player, now }: { player: PublicPlayer; now: number }) {
  if (player.connected) return null;
  const seconds = player.disconnectExpiresAt ? Math.max(0, Math.ceil((player.disconnectExpiresAt - now) / 1000)) : 0;
  return <em className="offline-badge">离线 {seconds}s</em>;
}

/** 证明图：blob 立刻显示；远端图先占位，滚入视口附近（IntersectionObserver）才挂载 <img>，
 * 避免对局记录瀑布流一次性发起大量图片请求。处理缓存图 onLoad 不触发导致一直「加载中」。 */
export function ProofImage({ src, alt, className }: { src: string; alt: string; className?: string }) {
  const isLocal = src.startsWith("blob:") || src.startsWith("data:");
  const [loaded, setLoaded] = useState(isLocal);
  const [failed, setFailed] = useState(false);
  const [visible, setVisible] = useState(isLocal);
  const imgRef = useRef<HTMLImageElement | null>(null);
  const wrapRef = useRef<HTMLSpanElement | null>(null);

  useEffect(() => {
    const local = src.startsWith("blob:") || src.startsWith("data:");
    setFailed(false);
    setLoaded(local);
    setVisible(local);
  }, [src]);

  useEffect(() => {
    if (isLocal || visible) return;
    const el = wrapRef.current;
    if (!el || typeof IntersectionObserver === "undefined") {
      setVisible(true);
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          setVisible(true);
          observer.disconnect();
        }
      },
      { rootMargin: "400px" }
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [isLocal, visible, src]);

  useEffect(() => {
    if (!visible) return;
    // 下一帧检查：缓存命中时 complete 已为 true，不会再触发 onLoad
    const id = window.requestAnimationFrame(() => {
      const el = imgRef.current;
      if (el && el.complete && el.naturalWidth > 0) setLoaded(true);
    });
    return () => window.cancelAnimationFrame(id);
  }, [visible, src]);

  if (isLocal) {
    return <img className={className} src={src} alt={alt} />;
  }
  return (
    <span ref={wrapRef} className={`proof-image-wrap ${loaded ? "is-ready" : ""} ${className || ""}`}>
      {!loaded && !failed && (
        <span className="proof-image-loading">
          <span>图片加载中…</span>
          <span className="proof-image-loading-bar" aria-hidden="true"><i /></span>
        </span>
      )}
      {failed ? (
        <span className="proof-image-loading">图片加载失败</span>
      ) : visible ? (
        <img
          ref={imgRef}
          src={src}
          alt={alt}
          loading="lazy"
          decoding="async"
          className={loaded ? "is-loaded" : ""}
          onLoad={() => setLoaded(true)}
          onError={() => setFailed(true)}
        />
      ) : null}
    </span>
  );
}

export function PunishmentStatus({ proof, isMine, requireConfirm }: { proof: RoomSnapshot["proofs"][number]; isMine: boolean; requireConfirm: boolean }) {
  if (proof.status === "rejected") {
    return <p className="task-card danger"><b>{isMine ? "对方要求你重做" : "已要求对方重做"}</b>{proof.redoTaskText || "请重新完成任务。"}</p>;
  }
  if (proof.status === "approved" || proof.confirmedBy) {
    if (proof.rejectReason === "双方互相放过，下一局正常开始。") {
      return <p className="task-card success"><b>双方互相放过</b>下一局正常开始。</p>;
    }
    return <p className="task-card success"><b>{proof.rejectReason === "对方选择放过你" ? "对方选择放过你" : isMine ? "对方已确认完成" : "你已确认完成"}</b>{requireConfirm ? "等待系统进入下一局。" : "准备进入下一局。"}</p>;
  }
  return <p className="task-card"><b>证明已提交</b>{requireConfirm ? (isMine ? "等待对方验证。" : "请验证对方证明。") : "准备进入下一局。"}</p>;
}

export function punishedPlayerNames(room: RoomSnapshot) {
  const players = roomPlayerList(room).map((item) => item.player);
  return (room.punishedPlayerIds || []).map((id) => {
    const player = players.find((item) => item.id === id);
    return player ? displayPlayerName(player) : id;
  });
}

export function roomPlayerById(room: RoomSnapshot, playerId: string) {
  return roomPlayerList(room).map((item) => item.player).find((player) => player.id === playerId);
}

export function punishedPlayerName(room: RoomSnapshot, playerId: string) {
  const player = roomPlayerById(room, playerId);
  return player ? displayPlayerName(player) : playerId;
}

export function taskTextOnly(taskText: string, factionLabel?: string) {
  const labels = [factionLabel, "男性阵营", "女性阵营", "男娘阵营", "其他阵营"].filter(Boolean) as string[];
  let result = taskText;
  for (const label of labels) {
    result = result.replace(new RegExp(`^${escapeRegExp(label)}[：:]\\s*`), "");
  }
  return result.trim();
}

export function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

// ChatBoardShell：大厅聊天与房间聊天共用的外壳——标题行（含手机端折叠三角）+ 可选的 tabs + ChatPanel。
// ChatPanel 内部用 <> Fragment 直接吐出 .chat-scroll-shell 和 .send-row 两个元素，
// 让它们成为 .chat-panel 这个 3 行 grid（标题/消息/输入）的直接子项——所以折叠态不能再包一层
// wrapper div（会把 2 个 grid 子项并成 1 个，挤爆行映射），而是在 .chat-panel 本身打 collapsed class，
// 具体隐藏规则见 styles.css 里 .chat-panel.collapsed 的说明。
export function ChatBoardShell({
  title, collapseKey, tabs, children
}: {
  title: string;
  collapseKey: string;
  tabs?: ReactNode;
  children: ReactNode;
}) {
  const { collapsed, toggle } = useMobileCollapse(collapseKey);
  return (
    <div className={`panel chat-panel ${collapsed ? "collapsed" : ""}`}>
      <div className="chat-panel-head">
        <div className="chat-panel-head-main">
          <h3>{title}</h3>
          {tabs}
        </div>
        <CollapseToggle collapsed={collapsed} onToggle={toggle} label={title} />
      </div>
      {children}
    </div>
  );
}

// ChatPanel：房间聊天与大厅聊天共用。scope="" 为大厅，否则为 roomId。
// 首屏/历史走 chatStore（chat:load/loadOlder，读 SQLite），增量由 chatStore 监听 chat:new。
// 滚到顶部瀑布流加载更早 100 条并保持滚动位置；点头像 @人；@到自己的气泡高亮。
export function ChatPanel({
  scope, me, players, onError, readOnly = false, readOnlyHint, placeholder, emptyText, subscribeLobbyChannel = false, messagesClass
}: {
  scope: string;
  me: PublicPlayer;
  // 发言人当前资料：优先用调用方传入的在线/房间名单（更实时），再合并 chat:load 带回的
  // authors（服务端按 playerId 从内存玩家表取的当前资料，含离线玩家）。
  players: PublicPlayer[];
  onError: (message: string) => void;
  readOnly?: boolean;
  readOnlyHint?: string;
  placeholder?: string;
  emptyText?: string;
  subscribeLobbyChannel?: boolean;
  messagesClass?: string;
}) {
  const { messages, hasMore, loading, authors } = useChat(scope);
  const playersById = useMemo(() => {
    const map = new Map<string, PublicPlayer>();
    // 先放历史加载的 authors，再用在线名单覆盖（同 id 以最新在线快照为准）。
    for (const [id, player] of Object.entries(authors || {})) map.set(id, player);
    for (const player of players) map.set(player.id, player);
    return map;
  }, [players, authors]);
  const [text, setText] = useState("");
  const pendingMentionsRef = useRef<Array<{ playerId: string; name: string }>>([]);
  const listRef = useRef<HTMLDivElement | null>(null);
  const stickRef = useRef(true);
  const [stick, setStick] = useState(true);
  const [now, setNow] = useState(Date.now());

  useEffect(() => {
    stickRef.current = true;
    setStick(true);
    void loadChat(scope);
  }, [scope]);

  // 房间内的「大厅」tab：加入大厅聊天频道以收实时增量，卸载时退出。
  useEffect(() => {
    if (!subscribeLobbyChannel) return;
    ask("lobby:suggestions:subscribe", {}).catch(() => undefined);
    return () => {
      ask("lobby:suggestions:unsubscribe", {}).catch(() => undefined);
    };
  }, [subscribeLobbyChannel, scope]);

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  const visible = messages.filter((m) => !m.expiresAt || m.expiresAt > now);

  useEffect(() => {
    const list = listRef.current;
    if (list && stickRef.current) scrollToBottomSoon(list);
  }, [visible.length]);

  async function handleScroll(event: ReactUIEvent<HTMLDivElement>) {
    const el = event.currentTarget;
    const nextStick = isNearScrollBottom(el);
    if (stickRef.current !== nextStick) {
      stickRef.current = nextStick;
      setStick(nextStick);
    }
    // 滚到顶部附近且还有更早历史：瀑布流加载并保持视口位置。
    if (el.scrollTop < 200 && hasMore && !loading) {
      const prevHeight = el.scrollHeight;
      const added = await loadOlderChat(scope);
      if (added > 0) {
        window.requestAnimationFrame(() => {
          el.scrollTop = el.scrollHeight - prevHeight;
        });
      }
    }
  }

  function insertMention(player?: PublicPlayer) {
    if (!player || readOnly) return;
    const name = mentionLabel(player);
    if (!name) return;
    setText((t) => appendMentionText(t, name));
    pendingMentionsRef.current.push({ playerId: player.id, name });
  }

  async function send() {
    const value = text.trim();
    if (!value) return;
    // 仅保留文本中仍存在 "@昵称" 的提及（用户可能删掉了某个 @）。
    const mentions = Array.from(
      new Set(pendingMentionsRef.current.filter((m) => value.includes("@" + m.name)).map((m) => m.playerId))
    );
    setText("");
    pendingMentionsRef.current = [];
    try {
      await ask("chat:send", { roomId: scope, text: value, mentions });
    } catch (error) {
      setText(value);
      onError(error instanceof Error ? error.message : "发送失败");
    }
  }

  return (
    <>
      <div className="chat-scroll-shell">
        <div className={`messages ${messagesClass || ""}`} ref={listRef} onScroll={handleScroll}>
          {hasMore && <div className="chat-more-hint">{loading ? "加载中…" : "↑ 上滑加载更早消息"}</div>}
          {visible.map((item) => (
            <ChatBubble
              key={item.id}
              message={item}
              me={me}
              author={item.playerId === me.id ? me : playersById.get(item.playerId)}
              onMention={readOnly ? undefined : insertMention}
            />
          ))}
          {visible.length === 0 && <p className="empty">{emptyText || "还没有消息"}</p>}
        </div>
        {!stick && visible.length > 0 && (
          <button type="button" className="chat-stick-button" onClick={() => stickChatToBottom(listRef.current, stickRef, setStick)}>
            ↓ 回到底部
          </button>
        )}
      </div>
      {readOnly ? (
        <p className="hint chat-readonly-hint">{readOnlyHint || "此处只能查看。"}</p>
      ) : (
        <div className="send-row">
          <input
            value={text}
            onChange={(event) => setText(event.target.value)}
            onKeyDown={(event) => { if (event.key === "Enter") send(); }}
            placeholder={placeholder || "发一句话..."}
          />
          <button onClick={send}>发送</button>
        </div>
      )}
    </>
  );
}

// mentionLabel：@ 与头像上使用的短昵称（失格者用惩罚名）。
export function mentionLabel(player?: PublicPlayer): string {
  if (!player) return "";
  return player.nameWarPunished && player.nameWarPenaltyName ? player.nameWarPenaltyName : player.name;
}

// author：发言人当前资料，由 ChatPanel 按 message.playerId 实时查找传入（找不到则 undefined，
// 退回 message.author 纯文本）——不再使用发送时刻冻结的快照，昵称/头像变更后历史消息也会跟着更新。
export function ChatBubble({ message, me, author, onMention }: { message: ChatMessage; me: PublicPlayer; author?: PublicPlayer; onMention?: (player?: PublicPlayer) => void }) {
  if (message.system) return <p className="chat-system">{message.text}</p>;
  const mine = message.playerId === me.id;
  // 精确匹配：消息 @ 的 playerId 列表命中我时高亮气泡。
  const mentionsMe = Array.isArray(message.mentions) && message.mentions.includes(me.id);
  // 只能 @ 别人：点头像插入 @昵称（自己的气泡与只读场景不可点）。
  const canMention = Boolean(onMention) && !mine;
  return (
    <div className={`chat-bubble-row ${mine ? "mine" : ""} ${mentionsMe ? "mentioned" : ""}`}>
      {!mine && <ChatAvatar player={author} onMention={canMention && onMention ? () => onMention(author) : undefined} />}
      <div className="chat-bubble">
        <div className="chat-meta">
          {author ? <ChatName player={author} /> : <b>{message.author}</b>}
          {message.authorRole && <em>{message.authorRole}</em>}
        </div>
        <p>{message.text}</p>
      </div>
      {mine && <ChatAvatar player={author} />}
    </div>
  );
}

export function ChatAvatar({ player, onMention }: { player?: PublicPlayer; onMention?: () => void }) {
  const label = mentionLabel(player) || player?.name;
  const ch = label?.slice(0, 1) || "?";
  const content = player?.avatarUrl ? <img className="chat-avatar-image" src={player.avatarUrl} alt="" /> : ch;
  if (onMention) {
    return <button type="button" className="chat-avatar chat-avatar-button" title={`@ ${label}`} onClick={onMention}>{content}</button>;
  }
  return <span className="chat-avatar">{content}</span>;
}

export function ChatName({ player }: { player: PublicPlayer }) {
  const punished = Boolean(player.nameWarPunished && player.nameWarPenaltyName);
  const stats = safePlayerStats(player);
  return (
    <span className="chat-name">
      <span className="chat-gender" style={genderStyle(player)}>{player.genderLabel}</span>
      <span className="chat-title" style={titleStyle(stats.titleColors)}>{stats.title}</span>
      <b className={punished ? "name-war-pill" : ""}>{displayPlayerName(player)}</b>
      <ModeChip player={player} />
      <GiveawayChip player={player} />
      {!player.connected && <span className="chat-offline">离线</span>}
    </span>
  );
}

export function Settlement({ room }: { room: RoomSnapshot }) {
  const latest = room.roundHistory[0];
  if (!latest || room.phase !== "result" && room.phase !== "punishment") return <p className="settlement-placeholder">等待结算</p>;
  return (
    <div className="settlement-pop">
      <span>{choiceText(latest.moveA)} <b>vs</b> {choiceText(latest.moveB)}</span>
      <strong>{latest.resultLabel || historyResultText(latest.result)}</strong>
    </div>
  );
}

export function roomPlayerList(room: RoomSnapshot) {
  const result: Array<{ player: PublicPlayer; role: string }> = [];
  for (const seat of ["A", "B"] as SeatKey[]) {
    const occupant = room.seats[seat];
    if (occupant) result.push({ player: occupant, role: "战斗席" });
  }
  for (const player of room.spectators) result.push({ player, role: "观战" });
  return result;
}

export function SeatView({ seat, room, me, now, onSit }: { seat: SeatKey; room: RoomSnapshot; me: PublicPlayer; now: number; onSit: () => void }) {
  const occupant = room.seats[seat];
  const choice = room.revealedChoices?.[seat] || (occupant?.id === me.id ? room.choices[seat] : room.choices[seat] ? "hidden" : undefined);
  const stats = room.seatStats[seat];
  const battleSeatBlocked = Boolean(room.settings.enableRanked && Boolean(me.extremeModeEnabled) !== Boolean(room.settings.enableExtremeRanked));
  const othelloTurn = room.settings.gameId === "othello" && room.othello?.turn === seat && room.phase === "choosing" && !room.othello.ended;
  const othelloColorLabel = room.settings.gameId === "othello" && room.othello
    ? room.othello.blackSeat === seat ? "⚫ 黑棋" : "⚪ 白棋"
    : seat === "A" ? "随机后显示黑/白" : "随机后显示黑/白";
  const tictactoeTurn = room.settings.gameId === "tictactoe" && room.tictactoe?.turn === seat && room.phase === "choosing" && !room.tictactoe.ended;
  const tictactoeMarkLabel = room.settings.gameId === "tictactoe" && room.tictactoe
    ? room.tictactoe.xSeat === seat ? "❌ X" : "⭕ O"
    : "随机后显示 X/O";
  const gomokuTurn = room.settings.gameId === "gomoku" && room.gomoku?.turn === seat && room.phase === "choosing" && !room.gomoku.ended;
  const gomokuMarkLabel = room.settings.gameId === "gomoku" && room.gomoku
    ? room.gomoku.blackSeat === seat ? "⚫ 黑棋" : "⚪ 白棋"
    : "随机后显示黑/白";
  const jungleTurn = room.settings.gameId === "jungle" && room.jungle?.turn === seat && room.phase === "choosing" && !room.jungle.ended;
  const jungleMarkLabel = room.settings.gameId === "jungle" ? jungleSideLabel(seat) : "";
  return (
    <div className={`seat-card seat-${seat.toLowerCase()}`}>
      <div className="seat-identity">
        <span className="seat-label">玩家 {seat}</span>
        {occupant ? (
          <strong className="seat-occupant-row">
            <PlayerAvatar player={occupant} size={24} />
            <PlayerBadge player={occupant} compact />
          </strong>
        ) : <button disabled={battleSeatBlocked} title={battleSeatBlocked ? "当前排位类型不匹配，只能观战" : "坐到战斗席"} onClick={onSit}>{battleSeatBlocked ? "👀 只能观战" : "🪑 坐下"}</button>}
      </div>
      {occupant && <OfflineBadge player={occupant} now={now} />}
      <p className="choice-badge">
        {room.settings.gameId === "othello"
          ? othelloTurn ? `${othelloColorLabel}落子中` : othelloColorLabel
          : room.settings.gameId === "tictactoe"
            ? tictactoeTurn ? `${tictactoeMarkLabel}落子中` : tictactoeMarkLabel
            : room.settings.gameId === "gomoku"
              ? gomokuTurn ? `${gomokuMarkLabel}落子中` : gomokuMarkLabel
              : room.settings.gameId === "jungle"
                ? jungleTurn ? `${jungleMarkLabel} 走子中` : jungleMarkLabel
                : choice ? choiceText(choice) : room.seats.A && room.seats.B ? "🤔 等待出拳" : "⏳ 等人"}
      </p>
      {occupant && <SeatStatsView stats={stats} />}
    </div>
  );
}

export function SeatStatsView({ stats }: { stats: RoomSnapshot["seatStats"]["A"] }) {
  const wins = Number.isFinite(Number(stats?.wins)) ? Number(stats.wins) : 0;
  const losses = Number.isFinite(Number(stats?.losses)) ? Number(stats.losses) : 0;
  const draws = Number.isFinite(Number(stats?.draws)) ? Number(stats.draws) : 0;
  const punishments = Number.isFinite(Number(stats?.punishments)) ? Number(stats.punishments) : 0;
  const decisive = wins + losses;
  const rate = decisive === 0 ? 0 : Math.round((wins / decisive) * 100);
  return <small className="seat-stats">本席：{wins}胜 {losses}负 {draws}平 {punishments}惩罚 · 胜率 {rate}%</small>;
}

export function choiceText(choice: Move | "hidden") {
  if (choice === "hidden") return "🔒 已出拳";
  if (choice === "noMove") return "⏳ 未出拳";
  if (choice === "forfeit") return "📴 断线判负";
  if (choice === "giveaway") return "🫴 白给";
  return choice === "rock" ? "✊ 锤子" : choice === "scissors" ? "✌️ 剪刀" : "🖐️ 布";
}

export function historyResultText(result: RoundResult) {
  if (result === "doubleLoss") return "双输";
  if (result === "draw") return "平局";
  return `${result} 胜`;
}

export type RoomInfoTagView = { key: string; text: string; style: RoomInfoTagStyle };

export function roomInfoTag(config: AppConfig, key: string, extra = "", prefix = ""): RoomInfoTagView {
  const fromConfig = config.roomInfoTags?.[key];
  const fromOrder = roomInfoTagOrder.find((item) => item.key === key);
  // 配置缺失时用中文默认名+默认色，避免直接露出 gameRps 等内部 key
  const fallback: RoomInfoTagStyle = defaultRoomInfoTagStyle(fromOrder?.label || key);
  const style: RoomInfoTagStyle = {
    label: fromConfig?.label || fallback.label,
    textColor: fromConfig?.textColor || fallback.textColor,
    backgroundColor: fromConfig?.backgroundColor || fallback.backgroundColor,
    borderColor: fromConfig?.borderColor || fallback.borderColor
  };
  return { key: `${key}-${extra}`, text: `${prefix}${style.label}${extra}`, style };
}

export function punishmentInfoTag(config: AppConfig, room: RoomSnapshot) {
  if (!room.settings.enablePunishment) return roomInfoTag(config, "noPunishment");
  if (room.settings.punishmentSource === "player") return roomInfoTag(config, "punishment", "：玩家发布任务");
  return roomInfoTag(config, "punishment", punishmentSelectionText(config, room.settings));
}

export function rankedInfoExtra(stake: number, multiplier = 1, gameId: RoomSettings["gameId"] = "rps") {
  if (gameId === "othello") return multiplier > 1 ? ` ${stake} 分/子 ×${multiplier}` : ` ${stake} 分/子`;
  return multiplier > 1 ? ` ${stake} 分 ×${multiplier}` : ` ${stake} 分`;
}

const plainInfoTagStyle: RoomInfoTagStyle = { label: "", textColor: "#4d5c6f", backgroundColor: "#eef3f8", borderColor: "#c9d6e4" };

/** 大话骰参战人数区间标签，如「3-7人」；人数区间未知（如 0）时不显示。 */
function liarsDiceRosterTag(roomId: string, min?: number, max?: number): RoomInfoTagView | null {
  if (!min || !max) return null;
  return { key: `liarsdice-roster-${roomId}`, text: `${min}-${max}人`, style: plainInfoTagStyle };
}

/** 黑白棋/五子棋计时标签，如「计时：30秒/20分钟」；两个计时器都未启用时不显示。 */
function gameTimerTag(roomId: string, moveSeconds?: number, gameMinutes?: number): RoomInfoTagView | null {
  if (!moveSeconds && !gameMinutes) return null;
  const parts: string[] = [];
  if (moveSeconds) parts.push(`${moveSeconds}秒`);
  if (gameMinutes) parts.push(`${gameMinutes}分钟`);
  return { key: `game-timer-${roomId}`, text: `计时：${parts.join("/")}`, style: plainInfoTagStyle };
}

/** 五子棋悔棋次数标签，如「禁止悔棋」「允许悔棋1次」。 */
function gomokuUndoTag(roomId: string, undoLimit?: number): RoomInfoTagView {
  return {
    key: `gomoku-undo-${roomId}`,
    text: undoLimit ? `允许悔棋${undoLimit}次` : "禁止悔棋",
    style: plainInfoTagStyle
  };
}

export function roomInfoTags(config: AppConfig, room: RoomSnapshot) {
  const phaseKey = room.phase === "ready" ? "phaseReady"
    : room.phase === "choosing" ? (room.settings.gameId === "liarsdice" ? "phaseChoosingLiarsDice" : "phaseChoosing")
      : room.phase === "result" ? "phaseResult"
        : room.phase === "punishment" ? "phasePunishment"
          : "phaseReady";
  const multiplier = rankMultiplierForSettings(room.settings);
  const tags: RoomInfoTagView[] = [
    gameInfoTag(config, room.settings.gameId),
    roomInfoTag(config, phaseKey),
    room.settings.enableRanked ? roomInfoTag(config, "ranked", rankedInfoExtra(room.settings.stake, multiplier, room.settings.gameId)) : roomInfoTag(config, "normal"),
    punishmentInfoTag(config, room)
  ];
  if (room.settings.enableExtremeRanked) tags.push(roomInfoTag(config, "extremeRanked"));
  if (room.settings.enablePunishment) {
    if (room.settings.tieDoublePunish && room.settings.gameId !== "liarsdice") tags.push(roomInfoTag(config, "tieDoublePunish"));
    if (room.settings.requireOpponentConfirm) tags.push(roomInfoTag(config, "requireOpponentConfirm"));
    tags.push(roomInfoTag(config, room.settings.allowProofImage === false ? "textProofOnly" : "allowProofImage"));
  }
  if (room.settings.gameId === "liarsdice") {
    const t = liarsDiceRosterTag(room.id, room.settings.liarsDiceMinPlayers, room.settings.liarsDiceMaxPlayers);
    if (t) tags.push(t);
  } else if (room.settings.gameId === "othello") {
    const t = gameTimerTag(room.id, room.settings.othelloMoveSeconds, room.settings.othelloGameMinutes);
    if (t) tags.push(t);
  } else if (room.settings.gameId === "gomoku") {
    const t = gameTimerTag(room.id, room.settings.gomokuMoveSeconds, room.settings.gomokuGameMinutes);
    if (t) tags.push(t);
    tags.push(gomokuUndoTag(room.id, room.settings.gomokuUndoLimit));
  } else if (room.settings.gameId === "jungle") {
    const t = gameTimerTag(room.id, room.settings.jungleMoveSeconds, room.settings.jungleGameMinutes);
    if (t) tags.push(t);
  }
  return tags;
}

export function lobbyRoomInfoTags(config: AppConfig, room: LobbySnapshot["rooms"][number]) {
  const multiplier = room.rankMultiplier || 1;
  const tags: RoomInfoTagView[] = [
    gameInfoTag(config, room.gameId),
    room.enableRanked ? roomInfoTag(config, "ranked", rankedInfoExtra(room.stake, multiplier, room.gameId)) : roomInfoTag(config, "normal"),
    room.enablePunishment ? roomInfoTag(config, "punishment", punishmentSelectionText(config, room)) : roomInfoTag(config, "noPunishment")
  ];
  if (room.enableExtremeRanked) tags.push(roomInfoTag(config, "extremeRanked"));
  if (room.enablePunishment) {
    if (room.tieDoublePunish && room.gameId !== "liarsdice") tags.push(roomInfoTag(config, "tieDoublePunish"));
    if (room.requireOpponentConfirm) tags.push(roomInfoTag(config, "requireOpponentConfirm"));
  }
  if (room.gameId === "liarsdice") {
    const t = liarsDiceRosterTag(room.id, room.liarsDiceMinPlayers, room.liarsDiceMaxPlayers);
    if (t) tags.push(t);
  } else if (room.gameId === "othello") {
    const t = gameTimerTag(room.id, room.othelloMoveSeconds, room.othelloGameMinutes);
    if (t) tags.push(t);
  } else if (room.gameId === "gomoku") {
    const t = gameTimerTag(room.id, room.gomokuMoveSeconds, room.gomokuGameMinutes);
    if (t) tags.push(t);
    tags.push(gomokuUndoTag(room.id, room.gomokuUndoLimit));
  } else if (room.gameId === "jungle") {
    const t = gameTimerTag(room.id, room.jungleMoveSeconds, room.jungleGameMinutes);
    if (t) tags.push(t);
  }
  tags.push({
    key: `opponent-${room.id}`,
    text: "在线对战",
    style: plainInfoTagStyle
  });
  return tags;
}

export function gameInfoTag(config: AppConfig, gameId: RoomSettings["gameId"]) {
  return gameId === "othello"
    ? roomInfoTag(config, "gameOthello", "", "⚫⚪ ")
    : gameId === "tictactoe"
      ? roomInfoTag(config, "gameTicTacToe", "", "❌⭕ ")
      : gameId === "liarsdice"
        ? roomInfoTag(config, "gameLiarsDice", "", "🎲 ")
        : gameId === "gomoku"
          ? roomInfoTag(config, "gameGomoku", "", "●○ ")
          : gameId === "jungle"
            ? roomInfoTag(config, "gameJungle", "", "🦁 ")
            : roomInfoTag(config, "gameRps");
}

export function punishmentSelectionText(config: AppConfig, settings: Pick<RoomSettings, "punishmentId" | "punishmentIds">) {
  const punishments = selectedPunishmentIdsForConfig(config, settings as RoomSettings)
    .map((id) => config.punishments.find((punishment) => punishment.id === id))
    .filter((punishment): punishment is AppConfig["punishments"][number] => Boolean(punishment));
  if (!punishments.length) return "";
  const names = punishments.map((punishment) => punishment.name).join(" / ");
  return punishments.length > 1 ? `：${punishments.length}选1 ${names}` : `：${names}`;
}

export function RoomInfoTagList({ tags }: { tags: RoomInfoTagView[] }) {
  return (
    <div className="room-info-tags">
      {tags.map((tag) => (
        <span className="room-info-tag" style={roomInfoTagStyle(tag.style)} key={tag.key}>{tag.text}</span>
      ))}
    </div>
  );
}

export function roomInfoTagStyle(style: RoomInfoTagStyle): CSSProperties {
  return {
    "--room-info-text": style.textColor,
    "--room-info-bg": style.backgroundColor,
    "--room-info-border": style.borderColor
  } as CSSProperties;
}

export function phaseText(phase: RoomSnapshot["phase"], gameId?: RoomSettings["gameId"]) {
  if (phase === "ready") return "🪑 等待坐满";
  if (phase === "choosing") {
    if (gameId === "liarsdice") return "🎲 叫点中";
    if (gameId === "othello" || gameId === "tictactoe" || gameId === "gomoku" || gameId === "jungle") return "♟ 对局中";
    return "🤜 出拳中";
  }
  if (phase === "result") return "✨ 结果展示";
  if (phase === "punishment") return "🎲 惩罚阶段";
  return "⏳ 等待中";
}

export function roomStatusText(status: RoomSnapshot["status"]) {
  if (status === "playing") return "对战中";
  if (status === "punishment") return "惩罚中";
  return "等待中";
}

export function connectionStateText(state: "connected" | "connecting" | "disconnected") {
  if (state === "connected") return "已连接";
  if (state === "connecting") return "连接中";
  return "重连中";
}

export function Leaderboard({ title, players }: { title: string; players: PublicPlayer[] }) {
  const { collapsed, toggle } = useMobileCollapse("leaderboard");
  return (
    <div className={`panel leaderboard-panel ${collapsed ? "collapsed" : ""}`}>
      <div className="panel-header-row">
        <h2><Crown size={18} /> {title}</h2>
        <CollapseToggle collapsed={collapsed} onToggle={toggle} label={title} />
      </div>
      <div className={`leaderboard-list mobile-collapsible-body ${collapsed ? "collapsed" : ""}`}>
        {players.map((player, index) => {
          const stats = safePlayerStats(player);
          return (
            <p className="rank-row rich" key={player.id}>
              <span>{index + 1}. <PlayerAvatar player={player} size={22} /> <PlayerBadge player={player} compact /></span>
              <small>{stats.wins}胜 {stats.losses}负 {stats.draws}平 · {stats.punishments}惩罚</small>
              <b>{winRateText(player)} · {stats.rankedPoints}分</b>
            </p>
          );
        })}
        {players.length === 0 && <p className="empty">暂无在线玩家</p>}
      </div>
    </div>
  );
}

export type GlobalLeaderboardTab =
  | "positive" | "negative" | "historyPositive" | "historyNegative" | "extremePositive" | "extremeNegative" | "nameWar" | "giveaway"
  | "totalWins" | "onlineTime"
  | "rps" | "othello" | "tictactoe" | "gomoku" | "liarsdice" | "jungle";

const GAME_LEADERBOARD_TABS: Array<{ id: GlobalLeaderboardTab; label: string; title: string }> = [
  { id: "rps", label: "猜拳", title: "锤子剪刀布胜场榜" },
  { id: "othello", label: "黑白棋", title: "黑白棋胜场榜" },
  { id: "tictactoe", label: "井字棋", title: "井字棋胜场榜" },
  { id: "gomoku", label: "五子棋", title: "五子棋胜场榜" },
  { id: "jungle", label: "斗兽棋", title: "斗兽棋胜场榜" },
  { id: "liarsdice", label: "大话骰", title: "大话骰胜场榜" }
];

function gameWLDOf(player: PublicPlayer, tab: GlobalLeaderboardTab) {
  const gs = player.gameStats || { rps: {}, othello: {}, tictactoe: {}, gomoku: {}, liarsdice: {}, jungle: {} } as PublicPlayer["gameStats"];
  const raw = tab === "rps" ? gs.rps
    : tab === "othello" ? gs.othello
      : tab === "tictactoe" ? gs.tictactoe
        : tab === "gomoku" ? gs.gomoku
          : tab === "jungle" ? gs.jungle
            : tab === "liarsdice" ? gs.liarsdice
              : undefined;
  return {
    wins: Number(raw?.wins) || 0,
    losses: Number(raw?.losses) || 0,
    draws: Number(raw?.draws) || 0
  };
}

export function AboutPanel({ config, onClose, onOpenHelp }: { config: AppConfig; onClose: () => void; onOpenHelp?: () => void }) {
  const board = config.announcementBoard;
  const showBoard = board.enabled && (board.title.trim() || board.content.trim());
  return (
    <div className="modal-backdrop sponsor-backdrop" onClick={(event) => { if (event.target === event.currentTarget) onClose(); }}>
      <section className="sponsor-modal" onClick={(event) => event.stopPropagation()}>
        <div className="modal-title sponsor-title">
          <div>
            <h2><Info size={20} /> 关于</h2>
            <p className="hint">喜欢这个小站的话，可以在这里关注、进群或请作者喝杯咖啡。</p>
            {onOpenHelp && (
              <p className="hint" style={{ marginTop: "0.4em" }}>
                {" "}第一次来？<a href="#" onClick={(e) => { e.preventDefault(); onOpenHelp(); }}>看看怎么玩</a>。
              </p>
            )}
          </div>
          <button type="button" className="icon-button" onClick={onClose}>×</button>
        </div>
        {showBoard && (
          <div className="announcement-board">
            <span className="announcement-board-kicker">📢 {board.title}</span>
            <p className="announcement-board-content">{board.content}</p>
          </div>
        )}
        <div className="sponsor-hero">
          <div className="sponsor-hero-icon"><Coffee size={30} /></div>
          <div>
            <strong>谢谢你愿意支持抖喵酱</strong>
            <p>使用以下链接来联系、赞助本游戏屋原作作者。</p>
          </div>
        </div>
        <div className="sponsor-grid">
          {doumiaoLinks.map((item) => (
            <a
              className="sponsor-card"
              href={item.href}
              target="_blank"
              rel="noreferrer"
              key={item.id}
              style={{ "--sponsor-tone": item.tone } as CSSProperties}
            >
              <span className="sponsor-icon" aria-hidden="true">{item.id === "telegram" ? <Send size={22} /> : item.icon}</span>
              <span className="sponsor-copy">
                <strong>{item.title}</strong>
                <small>{item.description}</small>
              </span>
              <ExternalLink size={16} />
            </a>
          ))}
        </div>
        <div className="sponsor-hero">
          <div className="sponsor-hero-icon"><Coffee size={30} /></div>
          <div>
            <strong>谢谢你愿意支持 8023</strong>
            <p>使用以下链接来联系、赞助本游戏屋现任开发者与运维者。</p>
          </div>
        </div>
        <div className="sponsor-grid">
          {luv4uLinks.map((item) => (
            <a
              className="sponsor-card"
              href={item.href}
              target="_blank"
              rel="noreferrer"
              key={item.id}
              style={{ "--sponsor-tone": item.tone } as CSSProperties}
            >
              <span className="sponsor-icon" aria-hidden="true">{item.id === "telegram" ? <Send size={22} /> : item.icon}</span>
              <span className="sponsor-copy">
                <strong>{item.title}</strong>
                <small>{item.description}</small>
              </span>
              <ExternalLink size={16} />
            </a>
          ))}
        </div>
      </section>
    </div>
  );
}

const SECURITY_DISCLAIMER_MIN_WAIT_MS = 3000;

export function SecurityDisclaimer({ onConfirm }: { onConfirm: () => void }) {
  const [understand, setUnderstand] = useState<"yes" | "no" | null>(null);
  const [canComply, setCanComply] = useState<"yes" | "no" | null>(null);
  const [waited, setWaited] = useState(false);

  useEffect(() => {
    const timer = window.setTimeout(() => setWaited(true), SECURITY_DISCLAIMER_MIN_WAIT_MS);
    return () => window.clearTimeout(timer);
  }, []);

  const canConfirm = waited && understand === "yes" && canComply === "yes";

  return (
    <div className="security-disclaimer-backdrop" role="dialog" aria-modal="true" aria-labelledby="security-disclaimer-title">
      <section className="security-disclaimer-card">
        <h2 id="security-disclaimer-title"><Shield size={20} /> 平台用户安全与免责声明</h2>
        <p>
          欢迎来到抖喵游戏屋。本站仅供年满 18 岁的成年人休闲娱乐与文化交流，严禁任何形式的赌博、金钱交易、诈骗及泄露个人隐私的行为。游玩过程中请规范自身言行、尊重其他玩家，确保互动基于双方自愿，禁止利用游戏机制强迫或诱导他人。
        </p>
        <p>
          作为技术提供方，本站不对惩罚任务、线下接触等游戏玩法之外的任何个人行为或约定承担任何责任。因用户个人行为导致的财产损失、隐私泄露或人身安全问题，均由用户自行承担，本站概不负责。
        </p>
        <div className="security-disclaimer-question">
          <span>你是否清楚上述风险与规定？</span>
          <label><input type="radio" name="sd-understand" checked={understand === "yes"} onChange={() => setUnderstand("yes")} /> 清楚</label>
          <label><input type="radio" name="sd-understand" checked={understand === "no"} onChange={() => setUnderstand("no")} /> 不清楚</label>
        </div>
        <div className="security-disclaimer-question">
          <span>你能否做到遵守上述平台规则？</span>
          <label><input type="radio" name="sd-comply" checked={canComply === "yes"} onChange={() => setCanComply("yes")} /> 能</label>
          <label><input type="radio" name="sd-comply" checked={canComply === "no"} onChange={() => setCanComply("no")} /> 不能</label>
        </div>
        <div className="security-disclaimer-question"></div>
        <button className="primary" type="button" disabled={!canConfirm} onClick={onConfirm}>确定</button>
      </section>
    </div>
  );
}

export function GlobalLeaderboardPanel({ players, onClose }: { players: PublicPlayer[]; onClose: () => void }) {
  const [tab, setTab] = useState<GlobalLeaderboardTab>("positive");
  const [now, setNow] = useState(Date.now());

  useEffect(() => {
    const hasTimer = players.some((player) =>
      (player.nameWarRenameProtectedUntil && player.nameWarRenameProtectedUntil > Date.now()) ||
      (player.giveawayBoardExpiresAt && player.giveawayBoardExpiresAt > Date.now())
    );
    if (!hasTimer) return;
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [players]);

  const ranked = leaderboardPlayers(players, tab).slice(0, 50);
  const gameTabMeta = GAME_LEADERBOARD_TABS.find((item) => item.id === tab);
  const title = tab === "positive"
    ? "当前正分榜"
    : tab === "negative"
      ? "当前负分榜"
      : tab === "historyPositive"
        ? "历史最高分榜"
        : tab === "historyNegative"
          ? "历史最低分榜"
          : tab === "extremePositive"
            ? "极限正分榜"
            : tab === "extremeNegative"
              ? "极限负分榜"
              : tab === "nameWar"
                ? "名字争夺战榜"
                : tab === "giveaway"
                  ? "白给榜"
                  : tab === "totalWins"
                    ? "总局数榜"
                    : tab === "onlineTime"
                      ? "在线时长榜"
                      : gameTabMeta?.title || "排行榜";
  return (
    <div className="modal-backdrop leaderboard-backdrop" onClick={(event) => { if (event.target === event.currentTarget) onClose(); }}>
      <section className="leaderboard-modal">
        <div className="modal-title">
          <div>
            <h2><Crown size={20} /> 排行榜</h2>
            <p className="hint">排行榜每 10 分钟刷新一次，每类最多显示 50 名。总榜胜负平为五游戏合计。</p>
          </div>
          <button type="button" className="icon-button" onClick={onClose}>×</button>
        </div>
        <div className="segmented leaderboard-tabs">
          <button className={tab === "positive" ? "active" : ""} onClick={() => setTab("positive")}>当前正</button>
          <button className={tab === "negative" ? "active" : ""} onClick={() => setTab("negative")}>当前负</button>
          <button className={tab === "historyPositive" ? "active" : ""} onClick={() => setTab("historyPositive")}>历史正</button>
          <button className={tab === "historyNegative" ? "active" : ""} onClick={() => setTab("historyNegative")}>历史负</button>
          <button className={tab === "extremePositive" ? "active" : ""} onClick={() => setTab("extremePositive")}>极限正</button>
          <button className={tab === "extremeNegative" ? "active" : ""} onClick={() => setTab("extremeNegative")}>极限负</button>
          <button className={tab === "nameWar" ? "active" : ""} onClick={() => setTab("nameWar")}>名争</button>
          <button className={tab === "giveaway" ? "active" : ""} onClick={() => setTab("giveaway")}>白给</button>
          <button className={tab === "totalWins" ? "active" : ""} onClick={() => setTab("totalWins")}>总局数</button>
          <button className={tab === "onlineTime" ? "active" : ""} onClick={() => setTab("onlineTime")}>在线时长</button>
          {GAME_LEADERBOARD_TABS.map((item) => (
            <button key={item.id} className={tab === item.id ? "active" : ""} onClick={() => setTab(item.id)}>{item.label}</button>
          ))}
        </div>
        <div className="global-leaderboard-list">
          <h3>{title}</h3>
          {ranked.map((player, index) => {
            const stats = safePlayerStats(player);
            const game = isGameLeaderboardTab(tab) ? gameWLDOf(player, tab) : null;
            const showGameStats = Boolean(game) || tab === "totalWins";
            const wins = game ? game.wins : stats.wins;
            const losses = game ? game.losses : stats.losses;
            const draws = game ? game.draws : stats.draws;
            const decisive = wins + losses;
            const rate = decisive === 0 ? 0 : Math.round((wins / decisive) * 100);
            return (
              <article className="global-rank-card" key={`${tab}-${player.id}`}>
                <div className="global-rank-main">
                  <span className="rank-index">#{index + 1}</span>
                  <PlayerAvatar player={player} size={24} />
                  <PlayerBadge player={player} compact />
                  <span className={`online-dot ${player.connected ? "online" : "offline"}`}>{player.connected ? "在线" : "离线"}</span>
                </div>
                <div className="global-rank-stats">
                  {tab === "onlineTime" ? (
                    <>
                      <span>累计在线 {formatOnlineDuration(totalOnlineMsOf(player))}</span>
                      <span>{stats.wins}胜 {stats.losses}负 {stats.draws}平</span>
                    </>
                  ) : showGameStats ? (
                    <>
                      <span>{wins}胜 {losses}负 {draws}平</span>
                      <span>总局 {wins + losses + draws}</span>
                      <span>胜率 {rate}%</span>
                    </>
                  ) : (
                    <>
                      <span>{stats.rankedPoints} 分</span>
                      <span>历史最高 {stats.highestScore}</span>
                      <span>历史最低 {stats.lowestScore}</span>
                      <span>{stats.wins}胜 {stats.losses}负 {stats.draws}平</span>
                      <span>{stats.punishments} 惩罚</span>
                      <span>胜率 {winRateText(player)}</span>
                    </>
                  )}
                </div>
                <LeaderboardExtra player={player} tab={tab} now={now} />
              </article>
            );
          })}
          {ranked.length === 0 && <p className="empty">暂无玩家上榜</p>}
        </div>
      </section>
    </div>
  );
}

export function leaderboardPlayers(players: PublicPlayer[], tab: GlobalLeaderboardTab) {
  const copy = [...players];
  // 过滤可用展示分（已封顶）；排序一律用 sort* 真实分，避免封顶后同分乱序。
  if (tab === "positive") return copy.filter((player) => sortRankedPointsOf(player) > 0).sort((a, b) => sortRankedPointsOf(b) - sortRankedPointsOf(a) || b.stats.wins - a.stats.wins);
  if (tab === "negative") return copy.filter((player) => sortRankedPointsOf(player) < 0).sort((a, b) => sortRankedPointsOf(a) - sortRankedPointsOf(b) || b.stats.losses - a.stats.losses);
  if (tab === "historyPositive") return copy.filter((player) => sortHighestScoreOf(player) > 0).sort((a, b) => sortHighestScoreOf(b) - sortHighestScoreOf(a) || b.stats.wins - a.stats.wins);
  if (tab === "historyNegative") return copy.filter((player) => sortLowestScoreOf(player) < 0).sort((a, b) => sortLowestScoreOf(a) - sortLowestScoreOf(b) || b.stats.losses - a.stats.losses);
  if (tab === "extremePositive") return copy.filter((player) => player.extremeModeEnabled && sortRankedPointsOf(player) > 0).sort((a, b) => sortRankedPointsOf(b) - sortRankedPointsOf(a) || (b.extremeWinStreak || 0) - (a.extremeWinStreak || 0));
  if (tab === "extremeNegative") return copy.filter((player) => player.extremeModeEnabled && sortRankedPointsOf(player) < 0).sort((a, b) => sortRankedPointsOf(a) - sortRankedPointsOf(b) || (b.extremeWinStreak || 0) - (a.extremeWinStreak || 0));
  if (tab === "nameWar") {
    return copy
      .filter((player) => player.nameWarEnabled || player.nameWarPunished)
      .sort((a, b) => Number(Boolean(b.nameWarPunished)) - Number(Boolean(a.nameWarPunished)) || sortRankedPointsOf(a) - sortRankedPointsOf(b));
  }
  // 总局数：按全部游戏合计胜场排序（与 stats.wins 一致）。
  if (tab === "totalWins") {
    return copy
      .filter((player) => {
        const s = safePlayerStats(player);
        return s.wins + s.losses + s.draws > 0;
      })
      .sort((a, b) => {
        const sa = safePlayerStats(a);
        const sb = safePlayerStats(b);
        return sb.wins - sa.wins || sa.losses - sb.losses || sb.draws - sa.draws;
      });
  }
  // 在线时长：累计在线毫秒从长到短。
  if (tab === "onlineTime") {
    return copy
      .filter((player) => totalOnlineMsOf(player) > 0)
      .sort((a, b) => totalOnlineMsOf(b) - totalOnlineMsOf(a) || sortRankedPointsOf(b) - sortRankedPointsOf(a));
  }
  if (isGameLeaderboardTab(tab)) {
    return copy
      .filter((player) => {
        const g = gameWLDOf(player, tab);
        return g.wins + g.losses + g.draws > 0;
      })
      .sort((a, b) => {
        const ga = gameWLDOf(a, tab);
        const gb = gameWLDOf(b, tab);
        return gb.wins - ga.wins || ga.losses - gb.losses || gb.draws - ga.draws;
      });
  }
  return copy
    .filter((player) => player.giveawayEnabled || (player.giveawayValue || 0) > 0)
    .sort((a, b) => (b.giveawayValue || 0) - (a.giveawayValue || 0) || sortRankedPointsOf(b) - sortRankedPointsOf(a));
}

export function isGameLeaderboardTab(tab: GlobalLeaderboardTab) {
  return tab === "rps" || tab === "othello" || tab === "tictactoe" || tab === "gomoku" || tab === "jungle" || tab === "liarsdice";
}

export function LeaderboardExtra({ player, tab, now }: { player: PublicPlayer; tab: GlobalLeaderboardTab; now: number }) {
  if (tab === "totalWins") {
    const s = safePlayerStats(player);
    return (
      <p className="global-rank-extra">
        全部游戏胜场优先 · 总局 {s.wins + s.losses + s.draws}
      </p>
    );
  }
  if (tab === "onlineTime") {
    return (
      <p className="global-rank-extra">
        累计在线时长优先 · {formatOnlineDuration(totalOnlineMsOf(player))}
      </p>
    );
  }
  if (isGameLeaderboardTab(tab)) {
    const g = gameWLDOf(player, tab);
    const label = GAME_LEADERBOARD_TABS.find((item) => item.id === tab)?.label || "该游戏";
    return (
      <p className="global-rank-extra">
        {label} 胜场优先 · 总局 {g.wins + g.losses + g.draws}
      </p>
    );
  }
  if (tab === "nameWar") {
    const protectedMs = player.nameWarRenameProtectedUntil ? Math.max(0, player.nameWarRenameProtectedUntil - now) : 0;
    return (
      <p className="global-rank-extra">
        {player.nameWarPunished ? `失名中：${player.nameWarPenaltyName || "惩罚名生效"}` : "名字争夺战开启"}
        {player.nameWarAllowRename ? " · 允许他人改名" : ""}
        {protectedMs > 0 ? ` · 保护 ${formatDuration(protectedMs)}` : ""}
      </p>
    );
  }
  if (tab === "giveaway") {
    const boardMs = player.giveawayBoardExpiresAt ? Math.max(0, player.giveawayBoardExpiresAt - now) : 0;
    return (
      <p className="global-rank-extra">
        白给 {formatGiveawayValue(player.giveawayValue || 0)}%
        {player.giveawayBoardText && boardMs > 0 ? ` · 已上板 ${formatDuration(boardMs)}` : ""}
        {player.giveawayBoardText ? ` · 👍 ${player.giveawayBoardLikes || 0} / 👎 ${player.giveawayBoardDislikes || 0}` : ""}
      </p>
    );
  }
  if (tab === "extremePositive" || tab === "extremeNegative") {
    return (
      <p className="global-rank-extra">
        ⚡ 极限模式 · 连胜 {player.extremeWinStreak || 0}
      </p>
    );
  }
  return null;
}

export function winRateText(player: PublicPlayer) {
  const stats = safePlayerStats(player);
  const decisive = stats.wins + stats.losses;
  return `${decisive === 0 ? 0 : Math.round((stats.wins / decisive) * 100)}%`;
}

export function isNameWarLoser(player: PublicPlayer) {
  // 失格目标以 nameWarPunished 为准；分值线由服务端按真实分 + penaltyThreshold 判定。
  return Boolean(player.nameWarEnabled && player.nameWarAllowRename && player.nameWarPunished);
}

export function isNameWarLoserVisible(player: PublicPlayer, now = Date.now()) {
  if (!isNameWarLoser(player)) return false;
  if (player.connected) return true;
  return Boolean(player.disconnectedAt && now - player.disconnectedAt <= 1_800_000);
}

export function isExtremeRenameTarget(player: PublicPlayer) {
  return Boolean(player.extremeForceClosed);
}

export function isRenameTarget(player: PublicPlayer) {
  return isNameWarLoser(player) || isExtremeRenameTarget(player);
}

export function isRenameTargetVisible(player: PublicPlayer, now = Date.now()) {
  if (!isRenameTarget(player)) return false;
  if (player.connected) return true;
  return Boolean(player.disconnectedAt && now - player.disconnectedAt <= 1_800_000);
}

export function nameWarRenameQuotaLeft(player: PublicPlayer, now = Date.now()) {
  if (!player.nameWarRenameWindowStartedAt || now - player.nameWarRenameWindowStartedAt >= 10_800_000) return 3;
  return Math.max(0, 3 - (player.nameWarRenameCount || 0));
}

export function UniversalRenamePanel({ config, targets, me, onError }: { config: AppConfig; targets: PublicPlayer[]; me: PublicPlayer; onError: (message: string) => void }) {
  const [inputs, setInputs] = useState<Record<string, string>>({});
  const [now, setNow] = useState(Date.now());
  const nameWarQuota = nameWarRenameQuotaLeft(me, now);
  const nameWarMinPoints = Math.max(1, Math.round(config.nameWar?.renameMinPoints || DEFAULT_NAME_WAR_RENAME_MIN_POINTS));
  const canNameWarRename = me.stats.rankedPoints >= nameWarMinPoints && nameWarQuota > 0;
  const extremeMinPoints = Math.max(1, Math.round(config.extremeMode.forceRenameMinPoints || 1));
  const canExtremeRename = Boolean(me.extremeModeEnabled && me.stats.rankedPoints >= extremeMinPoints);
  const { collapsed, toggle: toggleCollapsed } = useMobileCollapse("nameWarLoser");

  useEffect(() => {
    if (!targets.some((player) => (player.nameWarRenameProtectedUntil && player.nameWarRenameProtectedUntil > now) || (player.extremeRenameProtectedUntil && player.extremeRenameProtectedUntil > now))) return;
    const timer = window.setInterval(() => setNow(Date.now()), 60_000);
    return () => window.clearInterval(timer);
  }, [targets, now]);

  async function renameTarget(targetId: string, kind: "nameWar" | "extreme") {
    const name = (inputs[targetId] || "").trim();
    if (!name) return;
    try {
      await ask("nameWar:renameTarget", { targetId, name, kind });
      setInputs((old) => ({ ...old, [targetId]: "" }));
      onError("名字修改成功");
    } catch (error) {
      onError(error instanceof Error ? error.message : "修改失败");
    }
  }

  return (
    <div className={`panel name-war-loser-panel ${collapsed ? "collapsed" : ""}`}>
      <div className="panel-header-row">
        <h2>🏷️ {config.nameWar.renamePanelTitle || config.nameWar.loserPanelTitle || "通用改名处"}</h2>
        <CollapseToggle collapsed={collapsed} onToggle={toggleCollapsed} label={config.nameWar.renamePanelTitle || config.nameWar.loserPanelTitle || "通用改名处"} />
      </div>
      <div className={`mobile-collapsible-body ${collapsed ? "collapsed" : ""}`}>
        <p className="hint">名争改名需要 {nameWarMinPoints} 分以上，你剩余 {nameWarQuota} / 3 次；极限强关改名需要开启极限模式且至少 {extremeMinPoints} 分。</p>
        <div className="name-war-loser-list">
          {targets.map((player) => {
            const nameWarTarget = isNameWarLoser(player);
            const extremeTarget = isExtremeRenameTarget(player);
            const nameWarProtectedMs = player.nameWarRenameProtectedUntil ? Math.max(0, player.nameWarRenameProtectedUntil - now) : 0;
            const extremeProtectedMs = player.extremeRenameProtectedUntil ? Math.max(0, player.extremeRenameProtectedUntil - now) : 0;
            const offlineKeepMs = !player.connected && player.disconnectedAt ? Math.max(0, 1_800_000 - (now - player.disconnectedAt)) : 0;
            const nameWarProtectedText = nameWarProtectedMs > 0 ? `名争保护 ${Math.ceil(nameWarProtectedMs / 3_600_000)} 小时` : "名争可改";
            const extremeProtectedText = extremeProtectedMs > 0 ? `极限保护 ${Math.ceil(extremeProtectedMs / 3_600_000)} 小时` : "极限可改";
            const inputValue = inputs[player.id] || "";
            const selfTarget = player.id === me.id;
            const nameWarDisabled = !nameWarTarget || !canNameWarRename || selfTarget || nameWarProtectedMs > 0;
            const extremeDisabled = !extremeTarget || !canExtremeRename || selfTarget || extremeProtectedMs > 0;
            return (
              <div className="name-war-loser-card" key={player.id}>
                <div className="admin-card-title">
                  <strong>{player.nameWarPenaltyName || player.name}</strong>
                  <small>
                    <span className={`online-dot ${player.connected ? "online" : "offline"}`}>{player.connected ? "在线" : "离线"}</span>
                    {player.stats.rankedPoints} 分
                  </small>
                </div>
                <div className="room-info-tags">
                  {nameWarTarget && <span className="room-info-tag">⚔️ {config.nameWar.nameWarLoserLabel || "名争失格"}</span>}
                  {extremeTarget && <span className="room-info-tag">⚡ {config.nameWar.extremeForceClosedLabel || "极限强关"}</span>}
                </div>
                <p className="hint">
                  {nameWarTarget ? nameWarProtectedText : ""}
                  {nameWarTarget && extremeTarget ? " · " : ""}
                  {extremeTarget ? extremeProtectedText : ""}
                </p>
                {offlineKeepMs > 0 && <p className="hint">离线保留：约 {Math.ceil(offlineKeepMs / 60_000)} 分钟后从名单隐藏。</p>}
                {player.nameWarRenamedByName && <p className="hint">名争最后改名者：{player.nameWarRenamedByName}</p>}
                {player.extremeRenamedByName && <p className="hint">极限最后改名者：{player.extremeRenamedByName}</p>}
                <div className="send-row">
                  <input value={inputValue} maxLength={12} disabled={selfTarget || (!nameWarTarget && !extremeTarget)} onChange={(event) => setInputs((old) => ({ ...old, [player.id]: event.target.value }))} placeholder={selfTarget ? "不能改自己的名字" : "输入新名字"} />
                  {nameWarTarget && <button disabled={nameWarDisabled || !inputValue.trim()} onClick={() => renameTarget(player.id, "nameWar")}>名争改名</button>}
                  {extremeTarget && <button disabled={extremeDisabled || !inputValue.trim()} onClick={() => renameTarget(player.id, "extreme")}>极限改名</button>}
                </div>
              </div>
            );
          })}
          {targets.length === 0 && <p className="empty">暂无可改名目标</p>}
        </div>
      </div>
    </div>
  );
}

export function GiveawayPanel({ config, players, me, onError }: { config: AppConfig; players: PublicPlayer[]; me: PublicPlayer; onError: (message: string) => void }) {
  const [text, setText] = useState("");
  const [now, setNow] = useState(Date.now());
  const { collapsed, toggle: toggleCollapsed } = useMobileCollapse("giveaway");
  const activeBoards = players
    .filter((player) => player.giveawayBoardText && player.giveawayBoardExpiresAt && player.giveawayBoardExpiresAt > now)
    .sort((a, b) => (b.giveawayBoardSubmittedAt || 0) - (a.giveawayBoardSubmittedAt || 0));
  const myActiveBoard = activeBoards.find((player) => player.id === me.id);
  const canSubmit = Boolean(me.giveawayEnabled && (me.giveawayValue || 0) > 0 && !myActiveBoard);
  const voteQuota = giveawayVoteQuota(me, now, config.giveaway.likeVoteLimitPerHour, config.giveaway.dislikeVoteLimitPerHour);

  useEffect(() => {
    const hasExpiry = players.some((player) => player.giveawayBoardExpiresAt && player.giveawayBoardExpiresAt > Date.now());
    if (!hasExpiry) return;
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [players]);

  async function submitBoard() {
    const cleanText = text.trim();
    if (!cleanText) return;
    try {
      const result = await ask<{ player: PublicPlayer }>("giveaway:submitBoard", { text: cleanText });
      if (result.player.id === me.id) setText("");
    } catch (error) {
      onError(error instanceof Error ? error.message : "上板失败");
    }
  }

  async function vote(targetId: string, voteType: "like" | "dislike") {
    try {
      await ask("giveaway:vote", { targetId, vote: voteType });
    } catch (error) {
      onError(error instanceof Error ? error.message : "操作失败");
    }
  }

  return (
    <div className={`panel giveaway-panel ${collapsed ? "collapsed" : ""}`}>
      <div className="panel-title compact-title">
        <h2>🫴 {config.giveaway.panelTitle}</h2>
        <span className="panel-title-actions">
          <span>{activeBoards.length} 条</span>
          <CollapseToggle collapsed={collapsed} onToggle={toggleCollapsed} label={config.giveaway.panelTitle} />
        </span>
      </div>
      <div className={`mobile-collapsible-body ${collapsed ? "collapsed" : ""}`}>
        <div className="giveaway-hints">
          <p className="hint">{config.giveaway.panelDescription}</p>
          {myActiveBoard && <p className="hint">你已经上板，过期后才能重新提交。当前剩余 {formatDuration((myActiveBoard.giveawayBoardExpiresAt || now) - now)}。</p>}
          {!canSubmit && me.giveawayEnabled && <p className="hint">你的白给值已经是 {formatGiveawayValue(me.giveawayValue || 0)}%，归零后自动关闭白给模式。</p>}
        </div>
        {canSubmit && (
          <div className="giveaway-submit">
            <textarea value={text} maxLength={300} onChange={(event) => setText(event.target.value)} placeholder={config.giveaway.submitPlaceholder} />
            <button className="primary small" onClick={submitBoard}>上板 12 小时</button>
          </div>
        )}
        {activeBoards.length > 0 && (
          <div className="giveaway-quota-line">
            <span>👍 还可 {voteQuota.likesLeft}/{config.giveaway.likeVoteLimitPerHour} · 👎 还可 {voteQuota.dislikesLeft}/{config.giveaway.dislikeVoteLimitPerHour} · {voteQuota.refreshText}</span>
          </div>
        )}
        <div className="giveaway-board-list">
          {activeBoards.map((player) => {
            const isSelf = player.id === me.id;
            const expiresText = player.giveawayBoardExpiresAt ? formatDuration(player.giveawayBoardExpiresAt - now) : "";
            return (
              <article className="giveaway-card" key={player.id}>
                <div className="giveaway-card-head">
                  <PlayerBadge player={player} />
                </div>
                <p className="giveaway-board-text">{player.giveawayBoardText}</p>
                <div className="giveaway-card-meta">
                  <span>剩余 {expiresText}</span>
                  <span>👍 {player.giveawayBoardLikes || 0} · 👎 {player.giveawayBoardDislikes || 0}</span>
                </div>
                <div className="giveaway-actions">
                  <button disabled={isSelf || voteQuota.likesLeft <= 0} onClick={() => vote(player.id, "like")}>👍 -{formatGiveawayValue(config.giveaway.likeVoteValue)}%</button>
                  <button disabled={isSelf || voteQuota.dislikesLeft <= 0} onClick={() => vote(player.id, "dislike")}>👎 +{formatGiveawayValue(config.giveaway.dislikeVoteValue)}%</button>
                </div>
              </article>
            );
          })}
          {activeBoards.length === 0 && <p className="empty">{config.giveaway.emptyText}</p>}
        </div>
      </div>
    </div>
  );
}

export function giveawayVoteQuota(player: PublicPlayer, now: number, likeLimit = 3, dislikeLimit = 10) {
  const startedAt = player.giveawayVoteWindowStartedAt || 0;
  const windowMs = 3_600_000;
  const expired = !startedAt || now - startedAt >= windowMs;
  const likesUsed = expired ? 0 : player.giveawayVoteLikesThisHour || 0;
  const dislikesUsed = expired ? 0 : player.giveawayVoteDislikesThisHour || 0;
  // 刷新文案统一用「分钟」单位（向下取整），例如 2 小时 → 120 分钟；不足 1 分钟 → 0 分钟
  const remainMinutes = expired ? 60 : Math.floor(Math.max(0, startedAt + windowMs - now) / 60_000);
  return {
    likesLeft: Math.max(0, likeLimit - likesUsed),
    dislikesLeft: Math.max(0, dislikeLimit - dislikesUsed),
    refreshText: `${remainMinutes} 分钟后刷新`
  };
}

export function formatGiveawayValue(value: number) {
  return Number.isInteger(value) ? String(value) : value.toFixed(1);
}

const emptyPetBondState = (): PetBondState => ({
  masters: [],
  pets: [],
  masterCandidates: [],
  petCandidates: [],
  incoming: [],
  outgoing: [],
  chains: [],
  config: withPetBondDefaults(null)
});

/** 从大厅玩家列表中解析完整用户信息，供徽章展示。 */
function resolveLobbyPlayer(players: PublicPlayer[], id: string, fallback?: Partial<PublicPlayer>): PublicPlayer {
  const found = players.find((p) => p.id === id);
  if (found) return found;
  return {
    id,
    name: fallback?.name || "未知玩家",
    genderId: fallback?.genderId || "",
    genderLabel: fallback?.genderLabel || "",
    factionId: fallback?.factionId || "",
    factionLabel: fallback?.factionLabel || "",
    factionColors: fallback?.factionColors || { textColor: "#243447", backgroundColor: "#eef6fc", borderColor: "#b9dcf4" },
    displayName: fallback?.displayName || fallback?.name || "未知玩家",
    avatarUrl: fallback?.avatarUrl,
    connected: Boolean(fallback?.connected),
    stats: fallback?.stats || {
      wins: 0, losses: 0, draws: 0, punishments: 0,
      rankedPoints: 0, highestScore: 0, lowestScore: 0,
      sortRankedPoints: 0, sortHighestScore: 0, sortLowestScore: 0,
      title: "暂无称号"
    },
    gameStats: fallback?.gameStats || {
      rps: { wins: 0, losses: 0, draws: 0 },
      othello: { wins: 0, losses: 0, draws: 0 },
      tictactoe: { wins: 0, losses: 0, draws: 0 },
      gomoku: { wins: 0, losses: 0, draws: 0 },
      liarsdice: { wins: 0, losses: 0, draws: 0 },
      jungle: { wins: 0, losses: 0, draws: 0 }
    }
  };
}

/** 宠物乐园用户信息：头像 → 性别 → 称号 → 用户名 → 玩法（名争/白给等）。 */
function PetBondPlayerInfo({ player, size = 28 }: { player: PublicPlayer; size?: number }) {
  return (
    <span className="pet-bond-player-info">
      <PlayerAvatar player={player} size={size} />
      <PlayerBadge player={player} compact />
    </span>
  );
}

/** 大厅「宠物乐园」：关系展示 / 认主 / 认宠 */
export function PetBondPanel({
  config, me, lobby, onError
}: {
  config: AppConfig;
  me: PublicPlayer;
  lobby: LobbySnapshot;
  onError: (message: string) => void;
}) {
  const petBondCfg = withPetBondDefaults(config.petBond);
  const [state, setState] = useState<PetBondState>(emptyPetBondState);
  const [busy, setBusy] = useState(false);
  const [titlePetId, setTitlePetId] = useState<string | null>(null);
  const [titleDraft, setTitleDraft] = useState("");
  const { collapsed, toggle: toggleCollapsed } = useMobileCollapse("petBond");

  async function reload() {
    try {
      const next = await ask<PetBondState>("petbond:getState", {});
      setState({ ...emptyPetBondState(), ...next, config: withPetBondDefaults(next?.config || petBondCfg) });
    } catch (error) {
      onError(error instanceof Error ? error.message : "加载宠物乐园失败");
    }
  }

  useEffect(() => {
    void reload();
    const onUpdate = (payload: PetBondState) => {
      if (payload && typeof payload === "object") {
        setState({ ...emptyPetBondState(), ...payload, config: withPetBondDefaults(payload.config || petBondCfg) });
      }
    };
    socket.on("petbond:update", onUpdate);
    return () => socket.off("petbond:update", onUpdate);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [me.id]);

  // 根据大厅玩家开关与公开关系边签名兜底刷新，确保他人开启功能后无需整页刷新。
  const lobbyBondSig = useMemo(
    () =>
      lobby.players
        .map((p) => `${p.id}:${p.connected ? 1 : 0}${p.bondMasterEnabled ? 1 : 0}${p.bondPetEnabled ? 1 : 0}${p.bondPublicDisplay ? 1 : 0}`)
        .sort()
        .join("|"),
    [lobby.players]
  );
  const bondsSig = useMemo(
    () =>
      (lobby.petBonds || [])
        .map((e) => `${e.masterId}>${e.petId}:${e.petTitle || ""}`)
        .sort()
        .join("|"),
    [lobby.petBonds]
  );
  useEffect(() => {
    void reload();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lobbyBondSig, bondsSig, me.bondMasterEnabled, me.bondPetEnabled, me.bondPublicDisplay]);

  async function run(event: string, payload: Record<string, unknown>, okMsg?: string) {
    setBusy(true);
    try {
      const next = await ask<PetBondState>(event, payload);
      setState({ ...emptyPetBondState(), ...next, config: withPetBondDefaults(next?.config || petBondCfg) });
      if (okMsg) onError(okMsg);
    } catch (error) {
      onError(error instanceof Error ? error.message : "操作失败");
    } finally {
      setBusy(false);
    }
  }

  const masterSlotsLeft = Math.max(0, petBondCfg.maxMastersPerPet - (state.masters?.length || 0));
  const petSlotsLeft = Math.max(0, petBondCfg.maxPetsPerMaster - (state.pets?.length || 0));

  // 候选列表：过滤掉“已是关系”，incoming 申请用户置顶并高亮。
  const masterCandidates = (state.masterCandidates || [])
    .filter((c) => c.status !== "already")
    .slice()
    .sort((a, b) => Number(Boolean(b.incoming)) - Number(Boolean(a.incoming)) || a.name.localeCompare(b.name, "zh"));
  const petCandidates = (state.petCandidates || [])
    .filter((c) => c.status !== "already")
    .slice()
    .sort((a, b) => Number(Boolean(b.incoming)) - Number(Boolean(a.incoming)) || a.name.localeCompare(b.name, "zh"));
  const chains = state.chains || [];

  return (
    <div className={`panel pet-bond-panel ${collapsed ? "collapsed" : ""}`}>
      <div className="panel-title compact-title">
        <h2>🐾 {petBondCfg.panelTitle}</h2>
        <span className="panel-title-actions">
          <CollapseToggle collapsed={collapsed} onToggle={toggleCollapsed} label={petBondCfg.panelTitle} />
        </span>
      </div>
      <div className={`mobile-collapsible-body ${collapsed ? "collapsed" : ""}`}>
        <section className="pet-bond-section">
          <h3>关系展示</h3>
          <p className="hint">仅显示在线且开启「公开展示」的 2～3 级认主链；更长链会拆成多组。</p>
          <div className="pet-bond-chain-list">
            {chains.map((chain, idx) => (
              <div className="pet-bond-chain" key={`${chain.playerIds.join(">")}-${idx}`}>
                {chain.playerIds.map((id, i) => {
                  const player = resolveLobbyPlayer(lobby.players, id, {
                    name: chain.playerNames[i],
                    displayName: chain.playerNames[i]
                  });
                  return (
                    <span className="pet-bond-chain-node" key={`${id}-${i}`}>
                      {i > 0 && <span className="pet-bond-chain-arrow">→</span>}
                      <PetBondPlayerInfo player={player} size={26} />
                    </span>
                  );
                })}
              </div>
            ))}
            {chains.length === 0 && <p className="empty">暂无公开关系链</p>}
          </div>
        </section>

        <section className="pet-bond-section">
          <div className="pet-bond-section-header">
            <h3>认主</h3>
            <span className="pet-bond-quota">剩余 {masterSlotsLeft} 名</span>
          </div>
          <div className="pet-bond-member-list">
            {state.masters.map((m) => {
              const player = resolveLobbyPlayer(lobby.players, m.playerId, {
                name: m.name,
                displayName: m.displayName,
                avatarUrl: m.avatarUrl,
                connected: m.connected
              });
              return (
                <div className="pet-bond-member-row" key={m.playerId}>
                  <PetBondPlayerInfo player={player} />
                  <span className="pet-bond-row-actions">
                    {m.releaseIncoming && m.releaseRequestId && (
                      <button type="button" className="small primary" disabled={busy} onClick={() => run("petbond:approve", { requestId: m.releaseRequestId }, "已解除关系")}>
                        同意解除关系
                      </button>
                    )}
                    {m.releasePending && m.releaseRequestId && (
                      <button type="button" className="small danger-soft" disabled={busy} onClick={() => run("petbond:cancel", { requestId: m.releaseRequestId }, "已撤销申请")}>
                        撤销申请
                      </button>
                    )}
                    {!m.releasePending && !m.releaseIncoming && m.connected && (
                      <button type="button" className="small soft-button" disabled={busy} onClick={() => run("petbond:requestRelease", { masterId: m.playerId, petId: me.id })}>
                        申请解除关系
                      </button>
                    )}
                  </span>
                </div>
              );
            })}
            {state.masters.length === 0 && <p className="hint">你还没有主人</p>}
          </div>
          {me.bondMasterEnabled ? (
            <div className="pet-bond-candidate-list">
              <p className="hint">开启认宠的在线玩家：</p>
              {masterCandidates.map((c) => {
                const player = resolveLobbyPlayer(lobby.players, c.playerId, {
                  name: c.name,
                  displayName: c.displayName,
                  avatarUrl: c.avatarUrl,
                  connected: c.connected
                });
                return (
                  <div className={`pet-bond-member-row ${c.incoming ? "pet-bond-pin" : ""}`} key={c.playerId}>
                    <PetBondPlayerInfo player={player} />
                    <span className="pet-bond-row-actions">
                      {c.incoming && c.incomingId ? (
                        <button type="button" className="small primary" disabled={busy} onClick={() => run("petbond:approve", { requestId: c.incomingId }, "已同意")}>
                          {c.incomingLabel || "同意认宠请求"}
                        </button>
                      ) : c.status === "pending" && c.requestId ? (
                        <button type="button" className="small danger-soft" disabled={busy} onClick={() => run("petbond:cancel", { requestId: c.requestId }, "已撤销申请")}>
                          撤销申请
                        </button>
                      ) : (
                        <button type="button" className="small" disabled={busy || masterSlotsLeft <= 0} onClick={() => run("petbond:seekMaster", { targetId: c.playerId })}>
                          申请认主
                        </button>
                      )}
                    </span>
                  </div>
                );
              })}
              {masterCandidates.length === 0 && <p className="empty">暂无可认主对象</p>}
            </div>
          ) : (
            <p className="hint">在个人设置开启「开启认主」后，可向在线玩家申请认主。</p>
          )}
        </section>

        <section className="pet-bond-section">
          <div className="pet-bond-section-header">
            <h3>认宠</h3>
            <span className="pet-bond-quota">剩余 {petSlotsLeft} 名</span>
          </div>
          <div className="pet-bond-member-list">
            {state.pets.map((p) => {
              const player = resolveLobbyPlayer(lobby.players, p.playerId, {
                name: p.name,
                displayName: p.displayName,
                avatarUrl: p.avatarUrl,
                connected: p.connected,
                stats: p.petTitle ? {
                  wins: 0, losses: 0, draws: 0, punishments: 0,
                  rankedPoints: 0, highestScore: 0, lowestScore: 0,
                  sortRankedPoints: 0, sortHighestScore: 0, sortLowestScore: 0,
                  title: p.petTitle
                } : undefined
              });
              return (
                <div className={`pet-bond-member-row ${p.newMasterPendingId || p.releaseIncoming ? "pet-bond-pin" : ""}`} key={p.playerId}>
                  <PetBondPlayerInfo player={player} />
                  <span className="pet-bond-row-actions">
                    {p.newMasterPendingId && (
                      <button type="button" className="small primary" disabled={busy} onClick={() => run("petbond:approve", { requestId: p.newMasterPendingId }, "已同意宠物认新主")}>
                        同意认新主{p.newMasterPendingName ? `（${p.newMasterPendingName}）` : ""}
                      </button>
                    )}
                    <button
                      type="button"
                      className="small soft-button"
                      disabled={busy}
                      onClick={() => {
                        setTitlePetId(p.playerId);
                        setTitleDraft(p.petTitle || "");
                      }}
                    >
                      设置宠物称号
                    </button>
                    {p.releaseIncoming && p.releaseRequestId && (
                      <button type="button" className="small primary" disabled={busy} onClick={() => run("petbond:approve", { requestId: p.releaseRequestId }, "已解除关系")}>
                        同意解除关系
                      </button>
                    )}
                    {p.releasePending && p.releaseRequestId && (
                      <button type="button" className="small danger-soft" disabled={busy} onClick={() => run("petbond:cancel", { requestId: p.releaseRequestId }, "已撤销申请")}>
                        撤销申请
                      </button>
                    )}
                    {!p.releasePending && !p.releaseIncoming && p.connected && (
                      <button type="button" className="small soft-button" disabled={busy} onClick={() => run("petbond:requestRelease", { masterId: me.id, petId: p.playerId })}>
                        申请解除关系
                      </button>
                    )}
                  </span>
                </div>
              );
            })}
            {state.pets.length === 0 && <p className="hint">你还没有宠物</p>}
          </div>
          {me.bondPetEnabled ? (
            <div className="pet-bond-candidate-list">
              <p className="hint">开启认主的在线玩家：</p>
              {petCandidates.map((c) => {
                const player = resolveLobbyPlayer(lobby.players, c.playerId, {
                  name: c.name,
                  displayName: c.displayName,
                  avatarUrl: c.avatarUrl,
                  connected: c.connected
                });
                return (
                  <div className={`pet-bond-member-row ${c.incoming ? "pet-bond-pin" : ""}`} key={c.playerId}>
                    <PetBondPlayerInfo player={player} />
                    <span className="pet-bond-row-actions">
                      {c.incoming && c.incomingId ? (
                        <button type="button" className="small primary" disabled={busy} onClick={() => run("petbond:approve", { requestId: c.incomingId }, "已同意")}>
                          {c.incomingLabel || "同意认主请求"}
                        </button>
                      ) : c.status === "pending" && c.requestId ? (
                        <button type="button" className="small danger-soft" disabled={busy} onClick={() => run("petbond:cancel", { requestId: c.requestId }, "已撤销申请")}>
                          撤销申请
                        </button>
                      ) : (
                        <button type="button" className="small" disabled={busy || petSlotsLeft <= 0} onClick={() => run("petbond:seekPet", { targetId: c.playerId })}>
                          申请认宠
                        </button>
                      )}
                    </span>
                  </div>
                );
              })}
              {petCandidates.length === 0 && <p className="empty">暂无可认宠对象</p>}
            </div>
          ) : (
            <p className="hint">在个人设置开启「开启认宠」后，可向在线玩家申请认宠。</p>
          )}
        </section>
      </div>

      {titlePetId && (
        <div className="modal-backdrop" onClick={() => setTitlePetId(null)}>
          <section className="logout-confirm-card" onClick={(e) => e.stopPropagation()}>
            <h3>设置宠物称号</h3>
            <p className="hint">设置对方称号标签，最多 {petBondCfg.maxTitleLength} 字。清空则恢复默认称号。</p>
            <input
              value={titleDraft}
              maxLength={petBondCfg.maxTitleLength}
              onChange={(e) => setTitleDraft(e.target.value)}
              placeholder="例如：小奶猫"
            />
            <div className="kick-confirm-actions">
              <button type="button" disabled={busy} onClick={() => setTitlePetId(null)}>取消</button>
              <button
                type="button"
                className="primary"
                disabled={busy}
                onClick={async () => {
                  await run("petbond:setTitle", { petId: titlePetId, title: titleDraft }, "称号已更新");
                  setTitlePetId(null);
                }}
              >
                保存
              </button>
            </div>
          </section>
        </div>
      )}
    </div>
  );
}

export function ProfilePanel({ config, me, theme, onThemeChange, onClose, onUpdated, onError, onLoggedOut }: { config: AppConfig; me: PublicPlayer; theme: "light" | "dark"; onThemeChange: (theme: "light" | "dark") => void; onClose: () => void; onUpdated: (player: PublicPlayer) => void; onError: (message: string) => void; onLoggedOut: () => void }) {
  const [name, setName] = useState(me.name);
  const [selfTitle, setSelfTitle] = useState(me.stats.selfTitle || "");
  const [genderId, setGenderId] = useState(me.genderId);
  const [factionId, setFactionId] = useState(me.factionId);
  const [nameWarEnabled, setNameWarEnabled] = useState(Boolean(me.nameWarEnabled));
  const [nameWarAllowRename, setNameWarAllowRename] = useState(Boolean(me.nameWarAllowRename));
  const [giveawayEnabled, setGiveawayEnabled] = useState(Boolean(me.giveawayEnabled));
  const [extremeModeEnabled, setExtremeModeEnabled] = useState(Boolean(me.extremeModeEnabled));
  const [bondMasterEnabled, setBondMasterEnabled] = useState(Boolean(me.bondMasterEnabled));
  const [bondPetEnabled, setBondPetEnabled] = useState(Boolean(me.bondPetEnabled));
  const [bondPublicDisplay, setBondPublicDisplay] = useState(Boolean(me.bondPublicDisplay));
  const [now, setNow] = useState(Date.now());
  const stats = safePlayerStats(me);
  const decisive = stats.wins + stats.losses;
  const winRate = decisive === 0 ? 0 : Math.round((stats.wins / decisive) * 100);
  const total = stats.wins + stats.losses + stats.draws;
  const nameChanged = name.trim() !== me.name;
  const cooldownMs = me.profileUpdatedAt ? Math.max(0, 60_000 - (now - me.profileUpdatedAt)) : 0;
  const nameCooldownSeconds = Math.ceil(cooldownMs / 1000);
  const nameWarChanged = nameWarEnabled !== Boolean(me.nameWarEnabled);
  const nameWarAllowRenameChanged = nameWarAllowRename !== Boolean(me.nameWarAllowRename);
  const nameWarCooldownMs = me.nameWarToggledAt ? Math.max(0, 43_200_000 - (now - me.nameWarToggledAt)) : 0;
  const nameWarCooldownHours = Math.ceil(nameWarCooldownMs / 3_600_000);
  const nameLockedByWar = Boolean(me.nameWarEnabled || nameWarEnabled);
  const giveawayValue = me.giveawayValue || 0;
  const giveawayCannotClose = Boolean(me.giveawayEnabled && !giveawayEnabled && giveawayValue > 0);
  const extremeModeChanged = extremeModeEnabled !== Boolean(me.extremeModeEnabled);
  const extremeCooldownMs = me.extremeModeCooldownUntil ? Math.max(0, me.extremeModeCooldownUntil - now) : 0;
  const extremeCooldownHours = Math.ceil(extremeCooldownMs / 3_600_000);
  const extremeCannotEnable = Boolean(!me.extremeModeEnabled && extremeModeEnabled && (stats.rankedPoints < 0 || extremeCooldownMs > 0));
  const extremeCannotClose = Boolean(me.extremeModeEnabled && !extremeModeEnabled && stats.rankedPoints <= 0);
  const [logoutConfirmOpen, setLogoutConfirmOpen] = useState(false);
  const [logoutClaimCode, setLogoutClaimCode] = useState<string | null>(null);
  const [logoutBusy, setLogoutBusy] = useState(false);
  const [pushPrefs, setPushPrefs] = useState<PushPreferences>({ mentionEnabled: false, turnEnabled: false, seatEnabled: false });
  const [notificationPermission, setNotificationPermission] = useState<NotificationPermission>(
    () => (typeof Notification === "undefined" ? "denied" : Notification.permission)
  );
  const [pushBusy, setPushBusy] = useState(false);
  const [avatarBusy, setAvatarBusy] = useState(false);
  const avatarInputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    if (!cooldownMs && !nameWarCooldownMs && !extremeCooldownMs) return;
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [cooldownMs, nameWarCooldownMs, extremeCooldownMs]);

  useEffect(() => {
    fetchPushPreferences().then(setPushPrefs).catch(() => undefined);
  }, []);

  async function enableNotifications() {
    setPushBusy(true);
    try {
      const permission = await requestNotificationPermission();
      setNotificationPermission(permission);
      if (permission === "granted") {
        await ensurePushSubscription();
      } else {
        onError("需要允许通知权限才能开启推送");
      }
    } finally {
      setPushBusy(false);
    }
  }

  async function togglePushPref(key: keyof PushPreferences, value: boolean) {
    const next = { ...pushPrefs, [key]: value };
    setPushPrefs(next);
    try {
      await updatePushPreferences({ [key]: value });
    } catch (error) {
      setPushPrefs(pushPrefs); // 回滚
      onError(error instanceof Error ? error.message : "保存推送偏好失败");
    }
  }

  useEffect(() => {
    if (!nameWarEnabled) setNameWarAllowRename(false);
  }, [nameWarEnabled]);

  async function saveProfile() {
    const genderError = genderChoiceError(config, genderId);
    if (genderError) {
      onError(genderError);
      return;
    }
    if (nameChanged && nameLockedByWar) {
      onError("名字争夺战开启后不能修改名字");
      return;
    }
    if (nameChanged && cooldownMs > 0) {
      onError(`改名冷却中，请 ${nameCooldownSeconds} 秒后再试`);
      return;
    }
    if ((nameWarChanged || nameWarAllowRenameChanged) && nameWarCooldownMs > 0) {
      onError(`名字争夺战冷却中，请 ${nameWarCooldownHours} 小时后再试`);
      return;
    }
    if (giveawayCannotClose) {
      onError("白给值归零前不能关闭白给模式");
      return;
    }
    if (extremeCannotEnable) {
      onError(extremeCooldownMs > 0 ? `极限模式冷却中，请 ${extremeCooldownHours} 小时后再开启` : "负分玩家不能开启极限模式");
      return;
    }
    if (extremeCannotClose) {
      onError("排位分必须大于 0 才能关闭极限模式，0 分不能关闭");
      return;
    }
    if (!me.extremeModeEnabled && extremeModeEnabled) {
      const ok = window.confirm("开启极限模式会把当前排位分归零，并禁止进入倍率房。胜负平和惩罚次数会保留。确认开启？");
      if (!ok) return;
    }
    try {
      const result = await ask<{ player: PublicPlayer }>("player:updateProfile", {
        name, genderId, selfTitle,
        nameWarEnabled, nameWarAllowRename, giveawayEnabled, extremeModeEnabled,
        bondMasterEnabled, bondPetEnabled, bondPublicDisplay
      });
      onUpdated(result.player);
      onError("个人资料已更新");
    } catch (error) {
      onError(error instanceof Error ? error.message : "保存失败");
    }
  }

  async function openLogoutConfirm() {
    setLogoutConfirmOpen(true);
    setLogoutBusy(true);
    try {
      const result = await refreshClaimKey();
      setLogoutClaimCode(encodeClaimCode(result.playerId, result.claimKey));
    } catch (error) {
      onError(error instanceof Error ? error.message : "生成认领密钥失败");
      setLogoutConfirmOpen(false);
    } finally {
      setLogoutBusy(false);
    }
  }

  async function confirmLogout() {
    setLogoutBusy(true);
    try {
      await logout();
      onLoggedOut();
    } catch (error) {
      onError(error instanceof Error ? error.message : "登出失败");
    } finally {
      setLogoutBusy(false);
    }
  }

  async function uploadAvatar(file: File) {
    setAvatarBusy(true);
    try {
      const uploadFile = await prepareAvatarImageForUpload(file);
      const form = new FormData();
      form.append("token", localStorage.getItem(tokenKey) || "");
      form.append("image", uploadFile, "avatar.webp");
      const response = await fetch("/api/avatar-image", { method: "POST", body: form });
      let data: { message?: string; player?: PublicPlayer } = {};
      try {
        data = await response.json();
      } catch {
        throw new Error(response.ok ? "服务器响应无效" : `上传失败（${response.status}）`);
      }
      if (!response.ok) throw new Error(data.message || "上传失败");
      if (!data.player) throw new Error("服务器未返回玩家资料");
      onUpdated(data.player);
      onError("头像已更新");
    } catch (error) {
      onError(error instanceof Error ? error.message : "头像上传失败");
    } finally {
      setAvatarBusy(false);
      if (avatarInputRef.current) avatarInputRef.current.value = "";
    }
  }

  async function clearAvatar() {
    setAvatarBusy(true);
    try {
      const form = new FormData();
      form.append("token", localStorage.getItem(tokenKey) || "");
      form.append("clear", "1");
      const response = await fetch("/api/avatar-image", { method: "POST", body: form });
      let data: { message?: string; player?: PublicPlayer } = {};
      try {
        data = await response.json();
      } catch {
        throw new Error(response.ok ? "服务器响应无效" : `清空失败（${response.status}）`);
      }
      if (!response.ok) throw new Error(data.message || "清空失败");
      if (!data.player) throw new Error("服务器未返回玩家资料");
      onUpdated(data.player);
      onError("已恢复默认头像");
    } catch (error) {
      onError(error instanceof Error ? error.message : "清空头像失败");
    } finally {
      setAvatarBusy(false);
      if (avatarInputRef.current) avatarInputRef.current.value = "";
    }
  }

  async function forceCloseExtremeMode() {
    const ok = window.confirm(config.extremeMode.forceCloseWarning || "强行关闭极限模式后，你会进入通用改名处，可被符合条件的极限玩家改名。确认继续？");
    if (!ok) return;
    try {
      const result = await ask<{ player: PublicPlayer }>("extreme:forceClose", {});
      setExtremeModeEnabled(false);
      onUpdated(result.player);
      onError("已强行关闭极限模式");
    } catch (error) {
      onError(error instanceof Error ? error.message : "强行关闭失败");
    }
  }

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <section className="profile-panel" onClick={(event) => event.stopPropagation()}>
        <div className="profile-hero">
          <div className="avatar-ring">{me.avatarUrl ? <PlayerAvatar player={me} size={56} /> : <UserRound size={34} />}</div>
          <div>
            <h2><PlayerBadge player={me} /></h2>
            <p>{`${stats.title} · ${stats.rankedPoints} 排位积分`}</p>
          </div>
          <button className="profile-close-button" type="button" aria-label="关闭个人设置" onClick={onClose}>×</button>
        </div>

        <div className="profile-edit profile-stats-card">
          <h3>统计数据</h3>
          <div className="profile-stats">
            <Stat label="总局数" value={`${total}`} />
            <Stat label="总胜率" value={`${winRate}%`} />
            <Stat label="惩罚次数" value={`${stats.punishments}`} />
            <Stat label="排位积分" value={`${stats.rankedPoints}`} />
            <Stat label="历史战绩" value={`${stats.lowestScore} ～ ${stats.highestScore}`} />
            <Stat label="当前称号" value={stats.title} />
            <Stat label="在线时长" value={formatOnlineDuration(totalOnlineMsOf(me))} />
            <Stat label="白给值" value={`${formatGiveawayValue(giveawayValue)}%`} />
            <Stat label="极限模式" value={me.extremeModeEnabled ? `连胜 ${me.extremeWinStreak || 0}` : "未开启"} />
          </div>
        </div>

        <div className="profile-edit profile-stats-card">
          <h3>对局详情（胜/负/平）</h3>
          <div className="profile-stats profile-game-stats">
            <Stat label="对局总计" value={`${stats.wins} / ${stats.losses} / ${stats.draws}`} />
            {([
              ["锤子剪刀布", me.gameStats?.rps],
              ["黑白棋", me.gameStats?.othello],
              ["井字棋", me.gameStats?.tictactoe],
              ["五子棋", me.gameStats?.gomoku],
              ["斗兽棋", me.gameStats?.jungle],
              ["大话骰", me.gameStats?.liarsdice]
            ] as const).map(([label, g]) => (
              <Stat key={label} label={label} value={`${g?.wins || 0} / ${g?.losses || 0} / ${g?.draws || 0}`} />
            ))}
          </div>
        </div>

        <div className="profile-edit">
          <h3><Pencil size={18} /> 修改资料</h3>
          <div className="profile-edit-grid">
            <label className="field-label profile-name-field">
              <span>名字</span>
              <input value={name} maxLength={12} disabled={nameLockedByWar} onChange={(event) => setName(event.target.value)} placeholder="新的名字" />
              <small>{nameLockedByWar ? "名字争夺战开启后不能修改名字" : nameChanged && cooldownMs > 0 ? `改名冷却：${nameCooldownSeconds} 秒` : "名字会显示在大厅、房间和聊天里"}</small>
            </label>
            <label className="field-label">
              <span>称号</span>
              <input value={selfTitle} maxLength={12} onChange={(event) => setSelfTitle(event.target.value)} placeholder="称号" />
            </label>
            <FactionSelect
              config={config}
              factionId={factionId}
              onFactionChange={(nextFactionId) => {
                setFactionId(nextFactionId);
                setGenderId((old) => nextGenderIdForFaction(config, nextFactionId, old));
              }}
            />
            <GenderSelect
              config={config}
              genderId={genderId}
              factionId={factionId}
              onGenderChange={setGenderId}
            />
            <div className="profile-edit-grid-full profile-avatar-upload-row">
              <input
                ref={avatarInputRef}
                type="file"
                accept="image/*"
                className="profile-avatar-upload-input"
                disabled={avatarBusy}
                onChange={(event) => {
                  const file = event.target.files?.[0];
                  if (file) void uploadAvatar(file);
                }}
              />
              <button type="button" disabled={avatarBusy} onClick={() => avatarInputRef.current?.click()}>
                {avatarBusy ? "上传中…" : "上传头像"}
              </button>
              <button type="button" disabled={avatarBusy || !me.avatarUrl} onClick={() => void clearAvatar()}>
                清空头像
              </button>
            </div>
          </div>
          <p className="hint">上次改名：{me.profileUpdatedAt ? new Date(me.profileUpdatedAt).toLocaleString() : "还没有修改过"}</p>
          <div className="name-war-card">
            <div className="admin-card-title">
              <strong>夜间模式</strong>
              <small>{theme === "dark" ? "已开启" : "未开启"}</small>
            </div>
            <Toggle label="开启夜间模式" value={theme === "dark"} onChange={(value) => onThemeChange(value ? "dark" : "light")} />
          </div>
          <div className="name-war-card">
            <div className="admin-card-title">
              <strong>名字争夺战</strong>
              <small>{me.nameWarEnabled ? "已开启" : "未开启"}</small>
            </div>
            <Toggle label="开启名字争夺战" value={nameWarEnabled} disabled={nameWarCooldownMs > 0} onChange={setNameWarEnabled} />
            <Toggle label="允许其他玩家改名" value={nameWarAllowRename} disabled={!nameWarEnabled || nameWarCooldownMs > 0} onChange={setNameWarAllowRename} />
            <p className="hint">开启后排位分展示下限变为 {withRankedScoreDefaults(config.rankedScore).nameWarMin}；积分跌到 {config.nameWar?.penaltyThreshold ?? DEFAULT_NAME_WAR_PENALTY_THRESHOLD} 及以下后，只显示系统惩罚名，不显示性别和称号。</p>
            <p className="hint">允许其他玩家改名后，真实分跌到 {config.nameWar?.penaltyThreshold ?? DEFAULT_NAME_WAR_PENALTY_THRESHOLD} 及以下会出现在大厅失格者名单，{config.nameWar?.renameMinPoints ?? DEFAULT_NAME_WAR_RENAME_MIN_POINTS} 分以上玩家可以抢先给你改名。</p>
            <p className="hint">被其他玩家改名后，保护期内即使真实分回到失格线以上也不会提前恢复；保护期结束且真实分高于失格线才恢复。</p>
            {me.nameWarRenameProtectedUntil && me.nameWarRenameProtectedUntil > now && <p className="hint">改名保护中：约 {Math.ceil((me.nameWarRenameProtectedUntil - now) / 3_600_000)} 小时。</p>}
            {me.nameWarPunished && me.nameWarPenaltyName && <p className="hint">当前惩罚名：{me.nameWarPenaltyName}。</p>}
            {nameWarCooldownMs > 0 && <p className="hint">开关冷却：{nameWarCooldownHours} 小时</p>}
          </div>
          <div className="name-war-card giveaway-profile-card">
            <div className="admin-card-title">
              <strong>白给模式</strong>
              <small>{me.giveawayEnabled ? `${formatGiveawayValue(giveawayValue)}%` : "未开启"}</small>
            </div>
            <Toggle label="开启白给模式" value={giveawayEnabled} disabled={giveawayCannotClose} onChange={setGiveawayEnabled} />
            <p className="hint">开启后白给值默认为 0.1%；锤子剪刀布会按概率改为白给，黑白棋排位支持白给/上贡，井字棋支持随机白给，五子棋支持主动选择或按概率把本手落子变成对方棋子。</p>
            <p className="hint">出拳区点击“白给”会让白给值 +{formatGiveawayValue(config.giveaway.activeBoostValue)}%，触发强制白给后也会 +{formatGiveawayValue(config.giveaway.activeBoostValue)}%，最高 100%。</p>
            <p className="hint">黑白棋白给会让本手翻子不结算排位分并按 0.1%/子增加白给值；上贡会把本手分数给对面并按 0.2%/子增加白给值。</p>
            <p className="hint">白给值归零后，该模式自动关闭。游玩任何游戏获胜会让白给值 -{formatGiveawayValue(config.giveaway.winPenaltyValue)}%；也可以在大厅的白给自救板提交宣言，等待其他玩家点赞帮你降低；</p>
            {giveawayCannotClose && <p className="hint danger-hint">当前还有 {formatGiveawayValue(giveawayValue)}% 白给值，暂时不能关闭。</p>}
          </div>
          <div className="name-war-card extreme-profile-card">
            <div className="admin-card-title">
              <strong>{config.extremeMode.emoji} {config.extremeMode.label}</strong>
              <small>{me.extremeModeEnabled ? `连胜 ${me.extremeWinStreak || 0}` : extremeCooldownMs > 0 ? `冷却 ${extremeCooldownHours} 小时` : "未开启"}</small>
            </div>
            <Toggle label="开启极限模式" value={extremeModeEnabled} disabled={(!me.extremeModeEnabled && (stats.rankedPoints < 0 || extremeCooldownMs > 0)) || extremeCannotClose} onChange={setExtremeModeEnabled} />
            <p className="hint">开启要求当前排位分不为负；开启后排位分归零，但胜负平和惩罚次数保留。</p>
            <p className="hint">极限模式不能创建倍率房；进入普通排位或倍率房时只能观战，不能上桌。非极限玩家进入极限排位房也只能观战。</p>
            <p className="hint">正分输分和负分加分会按段位折扣；整点会自动扣分，离线也会扣。</p>
            <p className="hint">极限排位连胜达到 {config.extremeMode.winStreakThreshold} 局后，每次继续获胜都有 {Math.round(config.extremeMode.winStreakCrashChance * 100)}% 几率额外扣 {config.extremeMode.crashTargetPoints} 分。</p>
            {stats.rankedPoints < 0 && !me.extremeModeEnabled && <p className="hint danger-hint">你当前是负分，不能开启极限模式。</p>}
            {extremeCannotClose && <p className="hint danger-hint">排位分必须大于 0 才能关闭极限模式，0 分不能关闭。</p>}
            {me.extremeForceClosed && <p className="hint danger-hint">你曾强行关闭极限模式，已进入通用改名处。</p>}
            {me.extremeRenameProtectedUntil && me.extremeRenameProtectedUntil > now && <p className="hint">极限改名保护中：约 {Math.ceil((me.extremeRenameProtectedUntil - now) / 3_600_000)} 小时。</p>}
            {extremeCooldownMs > 0 && <p className="hint">关闭后冷却：约 {extremeCooldownHours} 小时后可重新开启。</p>}
            {me.extremeModeEnabled && (
              <button type="button" className="danger-button" onClick={forceCloseExtremeMode}>强行关闭极限模式</button>
            )}
          </div>
          <div className="name-war-card pet-bond-profile-card">
            <div className="admin-card-title">
              <strong>🐾 认主 / 认宠</strong>
              <small>
                {[bondMasterEnabled && "认主", bondPetEnabled && "认宠", bondPublicDisplay && "公开"].filter(Boolean).join(" · ") || "未开启"}
              </small>
            </div>
            <Toggle label="开启认主" value={bondMasterEnabled} onChange={setBondMasterEnabled} />
            <Toggle label="开启认宠" value={bondPetEnabled} onChange={setBondPetEnabled} />
            <Toggle label="公开展示" value={bondPublicDisplay} onChange={setBondPublicDisplay} />
            <p className="hint">允许同时开启认主与认宠。关闭开关不会取消已有关系，但无法再新增主/宠。</p>
            <p className="hint">关闭「公开展示」后，你不会出现在大厅关系图谱中（已有关系仍保留）。</p>
            <p className="hint">主人最多 {withPetBondDefaults(config.petBond).maxPetsPerMaster} 只宠物，宠物最多 {withPetBondDefaults(config.petBond).maxMastersPerPet} 位主人；认新主人需已有主人全体同意。</p>
          </div>
          <div className="name-war-card push-settings-card">
            <div className="admin-card-title">
              <strong>🔔 推送通知</strong>
              <small>
                {notificationPermission === "granted" ? "已允许" : notificationPermission === "denied" ? "已拒绝" : "未设置"}
              </small>
            </div>
            {notificationPermission !== "granted" && (
              <>
                <p className="hint">开启后，你离线时（关闭页面/手机后台被系统冻结）也能收到系统通知；页面还开着时会优先在页面内提醒，不会重复弹。</p>
                <button type="button" disabled={pushBusy} onClick={enableNotifications}>
                  {pushBusy ? "请求中…" : "开启推送通知"}
                </button>
                {notificationPermission === "denied" && <p className="hint danger-hint">浏览器已拒绝通知权限，需要在浏览器设置里手动允许后再试。</p>}
              </>
            )}
            {notificationPermission === "granted" && (
              <>
                <Toggle label="有人 @ 我" value={pushPrefs.mentionEnabled} onChange={(v) => togglePushPref("mentionEnabled", v)} />
                <Toggle label="轮到我出招/落子" value={pushPrefs.turnEnabled} onChange={(v) => togglePushPref("turnEnabled", v)} />
                <Toggle label="我的房间参战席被坐满" value={pushPrefs.seatEnabled} onChange={(v) => togglePushPref("seatEnabled", v)} />
                <button
                  type="button"
                  className="link-button"
                  onClick={async () => { await disablePushSubscription(); onError("已停止这台设备的推送订阅"); }}
                >
                  停止这台设备的推送
                </button>
              </>
            )}
          </div>
          <div className="name-war-card account-devices-card">
            <div className="admin-card-title">
              <strong>账号与设备</strong>
              <small>最多同时记住 3 台设备</small>
            </div>
            <ClaimKeyPanel onError={onError} />
            <button type="button" className="danger-button" onClick={openLogoutConfirm}>登出（清空本设备登录状态）</button>
          </div>
          <div className="profile-action-row">
            <button className="primary" disabled={(nameChanged && (cooldownMs > 0 || nameLockedByWar)) || ((nameWarChanged || nameWarAllowRenameChanged) && nameWarCooldownMs > 0) || giveawayCannotClose || extremeCannotEnable || extremeCannotClose} onClick={saveProfile}><Save size={16} /> 保存资料</button>
            <button type="button" onClick={onClose}>关闭设置</button>
          </div>
        </div>
      </section>
      {logoutConfirmOpen && (
        <div className="modal-backdrop logout-confirm-backdrop" onClick={() => !logoutBusy && setLogoutConfirmOpen(false)}>
          <section className="logout-confirm-card" onClick={(event) => event.stopPropagation()}>
            <h3>确认登出？</h3>
            <p className="hint danger-hint">
              登出后将无法再次登录当前账号，除非使用下面这把认领密钥在需要的设备上恢复。请务必先保存到私密位置（不要发给任何人）。
            </p>
            {logoutClaimCode ? <code className="claim-key-code">{logoutClaimCode}</code> : <p className="hint">正在生成密钥…</p>}
            <div className="kick-confirm-actions">
              <button disabled={logoutBusy} onClick={() => setLogoutConfirmOpen(false)}>取消</button>
              <button className="danger-button" disabled={logoutBusy || !logoutClaimCode} onClick={confirmLogout}>
                {logoutBusy ? "处理中…" : "我已保存密钥，确认登出"}
              </button>
            </div>
          </section>
        </div>
      )}
    </div>
  );
}

export const roomInfoTagOrder = [
  { key: "gameRps", label: "锤子剪刀布" },
  { key: "gameOthello", label: "黑白棋" },
  { key: "gameTicTacToe", label: "井字棋" },
  { key: "gameGomoku", label: "五子棋" },
  { key: "gameJungle", label: "斗兽棋" },
  { key: "gameLiarsDice", label: "大话骰" },
  { key: "phaseReady", label: "等待坐满" },
  { key: "phaseChoosing", label: "出拳中" },
  { key: "phaseChoosingLiarsDice", label: "叫点中" },
  { key: "phaseResult", label: "结算中" },
  { key: "phasePunishment", label: "惩罚阶段" },
  { key: "normal", label: "普通局" },
  { key: "ranked", label: "排位" },
  { key: "extremeRanked", label: "极限排位" },
  { key: "punishment", label: "惩罚开启" },
  { key: "noPunishment", label: "无惩罚" },
  { key: "tieDoublePunish", label: "平局双罚" },
  { key: "requireOpponentConfirm", label: "需要对手确认" },
  { key: "allowProofImage", label: "允许图片证明" },
  { key: "textProofOnly", label: "仅文字证明" }
];

export function defaultRoomInfoTagStyle(label: string): RoomInfoTagStyle {
  return { label, textColor: "#4d5c6f", backgroundColor: "#eef3f8", borderColor: "#c9d6e4" };
}

// 称号标签按赋予来源着色的固定 4 档；顺序即后台"称号池"里色卡的展示顺序。
export const titleTagStyleOrder = [
  { key: "system", label: "系统默认" },
  { key: "self", label: "自定义" },
  { key: "master", label: "主人赋予" },
  { key: "admin", label: "管理员赋予" }
];

export function punishmentTasks(punishment: AppConfig["punishments"][number], draft: AppConfig): PunishmentTaskConfig[] {
  if (punishment.tasks?.length) return punishment.tasks;
  return [{
    id: "task1",
    name: "默认任务",
    backgroundImages: [],
    backgroundOpacity: 0.22,
    variants: Object.fromEntries(draft.genderFactions.map((faction) => [
      faction.id,
      punishment.variants?.[faction.id] || punishment.description || "请完成本局惩罚。"
    ]))
  }];
}

export function Toggle({ label, value, onChange, disabled = false }: { label: string; value: boolean; onChange: (value: boolean) => void; disabled?: boolean }) {
  return <label className="toggle"><input type="checkbox" checked={value} disabled={disabled} onChange={(event) => onChange(event.target.checked)} /> {label}</label>;
}

export function Select({ value, options, onChange }: { value: string; options: Array<{ value: string; label: string }>; onChange: (value: string) => void }) {
  return <select value={value} onChange={(event) => onChange(event.target.value)}>{options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select>;
}

export function Stat({ label, value }: { label: string; value: string }) {
  return <div className="stat"><span>{label}</span><b>{value}</b></div>;
}
