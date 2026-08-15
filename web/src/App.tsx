import { Suspense, lazy, useEffect, useRef, useState } from "react";
import { Crown, Info, UserRound } from "lucide-react";
import { socket } from "./main";
import type { AppConfig, LobbySnapshot, PublicPlayer, RoomSnapshot } from "./shared/types";
import { leaderboardRefreshMs, playerSecretKey, securityDisclaimerKey, tokenKey } from "./lib/constants";
import {
  bumpWsAuthRetryCount, cacheJoinProfile, clearPlayerIdentity, connectSocketWithSession, getWsAuthRetryCount, hasCachedLogin,
  joinIdentityPayload, readCachedJoinGender, readPunishmentTagPrefs, resetWsAuthRetryCount, sessionTokenLooksValid, writePunishmentTagPrefs
} from "./lib/session";
import { ask, isAdminRoute, todayKey } from "./lib/rpc";
import {
  lobbyOnlineCount, mergeRoundHistory, normalizeConfig, normalizeLobbySnapshot,
  mergeLobbyPlayerIntoFullPlayer, normalizePublicPlayer, normalizeRoomSnapshot, normalizeRoundHistoryItem, playerSyncKey,
  replacePlayersInLobby, replacePlayersInRoom, replacePlayerInRoom
} from "./lib/normalize";
import { refreshActiveChats } from "./lib/chatStore";
import { ensurePushSubscription, fetchPushPreferences } from "./lib/pushNotify";
import type { AnnouncementPayload, MeState } from "./lib/types";
import {
  AboutPanel, GlobalLeaderboardPanel, Lobby, Login, PlayerBadge, ProfilePanel, Room, SecurityDisclaimer,
  connectionStateText, phaseText
} from "./ui/AppViews";
import { HelpPanel } from "./ui/HelpPanel";
import { startAnalytics, trackLoginSuccess, trackPageview, trackThemeToggle } from "./lib/analytics";

// 后台管理面板（含可能新增的图表等重型组件）单独打包，普通玩家不会触发这次 import。
const AdminPanel = lazy(() => import("./ui/AdminViews").then((module) => ({ default: module.AdminPanel })));


export function App() {
  const [config, setConfig] = useState<AppConfig | null>(null);
  const [lobby, setLobby] = useState<LobbySnapshot | null>(null);
  const [room, setRoom] = useState<RoomSnapshot | null>(null);
  const [me, setMe] = useState<MeState | null>(null);
  // 随机任务开房标签偏好：纯本地浏览器存储，不再随 player:join 从服务端下发。
  const [punishmentTagPrefs, setPunishmentTagPrefs] = useState<Record<string, string>>(() => readPunishmentTagPrefs());
  const [leaderboardPlayersSnapshot, setLeaderboardPlayersSnapshot] = useState<PublicPlayer[]>([]);
  const [view, setView] = useState<"login" | "lobby" | "room" | "admin">(() => isAdminRoute() ? "admin" : "login");
  // 有本地登录缓存时先进入恢复态，避免刷新时先闪一下登录页再进大厅。
  const [restoringSession, setRestoringSession] = useState(() => !isAdminRoute() && hasCachedLogin());
  const [profileOpen, setProfileOpen] = useState(false);
  const [leaderboardOpen, setLeaderboardOpen] = useState(false);
  const [aboutOpen, setAboutOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  const [notice, setNotice] = useState("");
  /** toast 展示文案与离场态：notice 更新时同步文案，定时先 leave 再清空，以便播放退出动效。 */
  const [toastText, setToastText] = useState("");
  const [toastLeaving, setToastLeaving] = useState(false);
  const [announcement, setAnnouncement] = useState<AnnouncementPayload | null>(null);
  // 后端构建版本号与本次前端构建不一致时置位；只做提示，不自动刷新（避免误判/多开
  // 场景下把用户的未保存输入或对局强制打断，也从根上杜绝"版本提示触发自动刷新"
  // 被用来当作 DoS 放大手段的可能）。dismissedRemoteBuildId 记录用户已经关掉提示的
  // 那个版本号：同一版本重复到达（心跳每 25s 一次）不会让横幅重新弹出；只有服务端
  // 又发布了新的、不同的版本号，才会再次出现。
  const [remoteBuildId, setRemoteBuildId] = useState<string | null>(null);
  const [dismissedRemoteBuildId, setDismissedRemoteBuildId] = useState<string | null>(null);
  // 每天每个浏览器只需确认一次；未过期就跳过声明页，直接走原来的连接流程。
  const [disclaimerConfirmed, setDisclaimerConfirmed] = useState(() => localStorage.getItem(securityDisclaimerKey) === todayKey());
  const [connectionState, setConnectionState] = useState<"connected" | "connecting" | "disconnected">(() => socket.connected ? "connected" : "connecting");
  const [restoreKickPending, setRestoreKickPending] = useState<Record<string, unknown> | null>(null);
  const [restoreKickBusy, setRestoreKickBusy] = useState(false);
  const [theme, setTheme] = useState<"light" | "dark">(() => (localStorage.getItem("rps-online-theme") === "dark" ? "dark" : "light"));
  const restoreInFlightRef = useRef(false);
  const hadConnectedRef = useRef(socket.connected);
  const latestLobbyPlayersRef = useRef<PublicPlayer[]>([]);
  const leaderboardSnapshotAtRef = useRef(0);

  useEffect(() => {
    startAnalytics();
    connectSocketWithSession().catch(() => {
      setConnectionState("disconnected");
      setRestoringSession(false);
      setNotice("连接失败，请检查网络后刷新重试。");
    });
  }, []);

  // 主视图 pageview
  useEffect(() => {
    trackPageview(view);
  }, [view]);

  // 弹窗 pageview
  useEffect(() => {
    if (profileOpen) trackPageview("profile");
  }, [profileOpen]);
  useEffect(() => {
    if (leaderboardOpen) trackPageview("leaderboard");
  }, [leaderboardOpen]);
  useEffect(() => {
    if (aboutOpen) trackPageview("about");
  }, [aboutOpen]);
  useEffect(() => {
    if (helpOpen) trackPageview("help");
  }, [helpOpen]);

  async function restoreSession(options: { showRecoveredNotice?: boolean } = {}) {
    if (restoreInFlightRef.current) return;
    const token = localStorage.getItem(tokenKey);
    const cachedName = localStorage.getItem("rps-online-name") || "";
    const { genderId: cachedGender } = readCachedJoinGender();
    if (!cachedName || !sessionTokenLooksValid(token)) {
      setRestoringSession(false);
      return;
    }
    // 未连上时绝不尝试 join，更不能因此清 token（旧逻辑会把”未连接”误判成坏 token）。
    if (!socket.connected) return;

    restoreInFlightRef.current = true;
    const payload = { name: cachedName, genderId: cachedGender, token, ...(await joinIdentityPayload()) };
    try {
      const next = await ask<MeState & { alreadyOnline?: true }>("player:join", payload);
      if (next.alreadyOnline) {
        setRestoreKickPending(payload);
        setRestoringSession(false);
        return;
      }
      if (next.token) localStorage.setItem(tokenKey, next.token);
      if (next.reissuedSecret) localStorage.setItem(playerSecretKey, next.reissuedSecret);
      // 以服务端确认后的资料为准回写缓存，避免清洗后的资料与本地不一致。
      if (next.player) cacheJoinProfile(next.player);
      setMe(next);
      trackLoginSuccess("restore");
      initPushForPlayer(next.player.id);
      if (next.room) setRoom(normalizeRoomSnapshot(next.room));
      else setRoom(null);
      if (!isAdminRoute()) {
        setView(next.room ? "room" : "lobby");
        if (next.room?.phase === "punishment") setNotice("已恢复到未完成的惩罚房间。");
        else if (options.showRecoveredNotice) setNotice("连接已恢复，玩家状态已同步。");
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : "";
      // 仅身份/会话类错误放弃自动登录；瞬时断线等保持缓存，下次 connect 再试。
      if (/身份校验|Session|session|token|令牌|会话/i.test(message)) {
        localStorage.removeItem(tokenKey);
        if (message === "玩家身份校验失败") {
          // 本地缓存的 playerId/playerSecret 服务端已经不认（常见于老账号未完成迁移）：
          // 只清 token 治标不治本——下次登录页仍会用同一对失效凭据再挂一次，必须连
          // 身份一起清掉，让登录页那次注册用全新身份重试。
          clearPlayerIdentity();
        }
        setMe(null);
        setRoom(null);
        if (!isAdminRoute()) setView("login");
      }
      if (options.showRecoveredNotice) setNotice("连接已恢复，但玩家状态同步失败，请刷新或重新进入。");
    } finally {
      restoreInFlightRef.current = false;
      setRestoringSession(false);
    }
  }

  async function confirmRestoreKick() {
    if (!restoreKickPending) return;
    setRestoreKickBusy(true);
    try {
      const next = await ask<MeState>("player:join", { ...restoreKickPending, forceKick: true });
      if (next.token) localStorage.setItem(tokenKey, next.token);
      if (next.reissuedSecret) localStorage.setItem(playerSecretKey, next.reissuedSecret);
      if (next.player) cacheJoinProfile(next.player);
      setMe(next);
      initPushForPlayer(next.player.id);
      if (next.room) setRoom(normalizeRoomSnapshot(next.room));
      else setRoom(null);
      if (!isAdminRoute()) setView(next.room ? "room" : "lobby");
      setRestoreKickPending(null);
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "登录失败");
    } finally {
      setRestoreKickBusy(false);
    }
  }

  function initPushForPlayer(_playerId: string) {
    fetchPushPreferences().catch(() => undefined);
    if (typeof Notification !== "undefined" && Notification.permission === "granted") {
      ensurePushSubscription().catch(() => undefined);
    }
  }

  function refreshLeaderboardSnapshot(players: PublicPlayer[], now = Date.now()) {
    latestLobbyPlayersRef.current = players;
    leaderboardSnapshotAtRef.current = now;
    setLeaderboardPlayersSnapshot(players);
  }

  useEffect(() => {
    socket.on("lobby:update", (nextLobby: LobbySnapshot) => {
      if (nextLobby.config) setConfig(normalizeConfig(nextLobby.config));
      const normalized = normalizeLobbySnapshot(nextLobby);
      latestLobbyPlayersRef.current = normalized.players;
      const now = Date.now();
      if (!leaderboardSnapshotAtRef.current || now - leaderboardSnapshotAtRef.current >= leaderboardRefreshMs) {
        refreshLeaderboardSnapshot(normalized.players, now);
      }
      setLobby((old) => normalizeLobbySnapshot(nextLobby, old));
    });
    socket.on("room:update", (nextRoom: RoomSnapshot) => {
      setRoom((old) => {
        const normalized = normalizeRoomSnapshot(nextRoom);
        // updatedAt 缺失/非数字时不要丢弃更新（protobuf 漏字段会导致永远用旧房态）
        const nextAt = Number(normalized.updatedAt) || 0;
        const oldAt = Number(old?.updatedAt) || 0;
        if (old?.id === normalized.id && nextAt > 0 && oldAt > 0 && nextAt < oldAt) return old;
        if (old?.id === normalized.id) {
          return {
            ...normalized,
            // chat 走 chat:append；全量包若未带 chat 则保留本地
            chat: (normalized.chat || []).length === 0 ? (old.chat || []) : normalized.chat,
            // history 必须合并：惩罚任务/证明写在 latest history，不能因空数组丢弃，也不能整包忽略更新
            roundHistory: mergeRoundHistory(old.roundHistory, normalized.roundHistory)
          };
        }
        return normalized;
      });
      if (!isAdminRoute()) setView("room");
    });
    socket.on("room:historyAppend", (payload: { roomId?: string; item?: RoomSnapshot["roundHistory"][number]; total?: number }) => {
      const roomId = payload?.roomId;
      const item = payload?.item;
      if (!roomId || !item) return;
      const safeItem = normalizeRoundHistoryItem(item);
      setRoom((old) => old?.id === roomId ? {
        ...old,
        roundHistory: mergeRoundHistory(old.roundHistory, [safeItem]),
        roundHistoryTotal: Number(payload.total) || old.roundHistoryTotal
      } : old);
    });
    // 聚合玩家更新（player:batch 为 LobbyPlayer[]；兼容单条 player:update）
    function applyPlayerPatches(rawList: PublicPlayer[]) {
      if (!rawList?.length) return;
      const players = rawList.map(normalizePublicPlayer);
      const byId = new Map(players.map((p) => [p.id, p]));
      // O(P+k) 合并本地排行榜快照源，避免每条 patch 都 some 扫描。
      const prevSnap = latestLobbyPlayersRef.current;
      const seen = new Set<string>();
      const nextSnap: PublicPlayer[] = [];
      for (const item of prevSnap) {
        const patched = byId.get(item.id);
        if (patched) {
          nextSnap.push(patched);
          seen.add(item.id);
        } else {
          nextSnap.push(item);
        }
      }
      for (const p of players) {
        if (!seen.has(p.id)) nextSnap.push(p);
      }
      latestLobbyPlayersRef.current = nextSnap;

      setLobby((old) => (old ? replacePlayersInLobby(old, players) : old));
      setRoom((old) => (old ? replacePlayersInRoom(old, players) : old));
      setMe((old) => {
        if (!old) return old;
        const p = byId.get(old.player.id);
        if (!p) return old;
        // LobbyPlayer/typed PublicPlayer 会省略空串字段：avatarUrl 清空时会变成 undefined。
        // 合并时必须显式写回空值，否则旧头像会残留在 me 上。
        const merged = {
          ...mergeLobbyPlayerIntoFullPlayer(old.player, p),
          genderId: typeof p.genderId === "string" ? p.genderId : "",
          avatarUrl: p.avatarUrl || undefined
        };
        return { ...old, player: merged, room: old.room ? replacePlayerInRoom(old.room, merged) : old.room };
      });
    }
    socket.on("player:batch", (list: PublicPlayer[]) => applyPlayerPatches(Array.isArray(list) ? list : []));
    socket.on("player:update", (player: PublicPlayer) => applyPlayerPatches(player ? [player] : []));
    socket.on("player:kicked", () => {
      localStorage.removeItem(tokenKey);
      setMe(null);
      setRoom(null);
      setView("login");
      setNotice("你已被管理员移出。");
    });
    // 同一账号在另一台设备完成了「顶替登录」确认：本设备的登录状态到此结束，但本地
    // playerId/playerSecret 仍然保留（这台设备依然是受信任设备之一，不是登出）。
    socket.on("session:kicked", ({ message }: { message?: string }) => {
      localStorage.removeItem(tokenKey);
      setMe(null);
      setRoom(null);
      setView("login");
      setNotice(message || "你的账号已在其他设备登录，此会话已结束，请刷新页面重新登录。");
    });
    socket.on("room:closed", ({ message }: { message?: string }) => {
      setRoom(null);
      setMe((old) => old ? { ...old, roomId: undefined } : old);
      if (!isAdminRoute()) setView("lobby");
      setNotice(message || "房间已被管理员关闭。");
    });
    socket.on("config:update", (config: AppConfig) => {
      setConfig(normalizeConfig(config));
    });
    // 聊天（房间 + 大厅留言板）已迁到 chatStore：chat:new / chat:deleted 由其内部监听，
    // 首屏与历史走 chat:load / chat:loadOlder，此处不再处理。
    socket.on("announcement:show", (payload: AnnouncementPayload) => {
      setAnnouncement(payload);
    });
    // server:hello：连接建立时的一次性推送 + 心跳应答兜底（见 ws.ts），带的是后端
    // 当前构建版本号。"dev"/"unknown" 是本地开发或非 git checkout 的兜底字面量，
    // 双方都可能出现，出现即视为不可比较，不触发提示。
    socket.on("server:hello", ({ buildId }: { buildId?: string }) => {
      if (!buildId || buildId === "dev" || buildId === "unknown") return;
      if (__APP_BUILD_ID__ === "dev" || __APP_BUILD_ID__ === "unknown") return;
      if (buildId === __APP_BUILD_ID__) return;
      setRemoteBuildId(buildId);
    });
    socket.on("connect", () => {
      setConnectionState("connected");
      resetWsAuthRetryCount();
      const isReconnect = hadConnectedRef.current;
      hadConnectedRef.current = true;
      // 首次连接不弹「已恢复」；断线重连后才提示。
      // 重连后补拉已激活聊天频道的最近消息（chatStore 不自注册 connect，见其注释）；
      // 必须等 restoreSession 里的 player:join 落地后再发，否则新连接还没绑定玩家/房间，
      // 房间聊天的 chat:load 会被 onChatLoad 以"你不在这个房间里"拒绝，断线期间的消息就永久错过了。
      void restoreSession({ showRecoveredNotice: isReconnect }).then(() => refreshActiveChats());
    });
    socket.on("disconnect", () => {
      setConnectionState("disconnected");
      // iOS 切后台也会触发 disconnect；文案轻一点，避免误以为故障
      setNotice("连接已断开，正在重连…");
    });
    socket.on("connect_error", (error: Error & { data?: { code?: string } }) => {
      setConnectionState("disconnected");
      const code = error?.data?.code;
      // 握手失败（401 等在浏览器里常变成 1006）或明确会话错误 → 丢弃本地 token 换发。
      // 本地 sessionTokenLooksValid 验不了签名，密钥轮换后旧 token 会永远卡死。
      const needNewToken =
        code === "SESSION_INVALID" ||
        code === "SESSION_EXPIRED" ||
        code === "SESSION_MISSING" ||
        code === "SESSION_HANDSHAKE_FAILED";
      if (needNewToken) {
        if (getWsAuthRetryCount() >= 2) {
          setRestoringSession(false);
          setNotice("连接认证失败，请刷新页面重试。");
          return;
        }
        bumpWsAuthRetryCount();
        localStorage.removeItem(tokenKey);
        connectSocketWithSession({ forceNewToken: true }).catch(() => {
          setConnectionState("disconnected");
          setRestoringSession(false);
          setNotice("连接认证失败，请刷新页面重试。");
        });
        return;
      }
      setRestoringSession(false);
    });
    socket.io.on("reconnect_attempt", () => setConnectionState("connecting"));
    // StrictMode 重挂载或热更新时 connect 事件可能已错过，已连接则立刻恢复会话。
    if (socket.connected) {
      setConnectionState("connected");
      void restoreSession({ showRecoveredNotice: false });
    }
    return () => {
      socket.off("lobby:update");
      socket.off("room:update");
      socket.off("room:historyAppend");
      socket.off("player:update");
      socket.off("player:batch");
      socket.off("player:kicked");
      socket.off("session:kicked");
      socket.off("room:closed");
      socket.off("config:update");
      socket.off("announcement:show");
      socket.off("server:hello");
      socket.off("connect");
      socket.off("disconnect");
      socket.off("connect_error");
      socket.io.off("reconnect_attempt");
    };
  }, []);

  useEffect(() => {
    const timer = window.setInterval(() => {
      if (latestLobbyPlayersRef.current.length) refreshLeaderboardSnapshot(latestLobbyPlayersRef.current);
    }, leaderboardRefreshMs);
    return () => window.clearInterval(timer);
  }, []);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("rps-online-theme", theme);
  }, [theme]);

  useEffect(() => {
    if (!notice) return;
    setToastText(notice);
    setToastLeaving(false);
    const leaveTimer = window.setTimeout(() => setToastLeaving(true), 3200);
    const clearTimer = window.setTimeout(() => {
      setToastText("");
      setToastLeaving(false);
      setNotice("");
    }, 3520);
    return () => {
      window.clearTimeout(leaveTimer);
      window.clearTimeout(clearTimer);
    };
  }, [notice]);

  useEffect(() => {
    // 管理员把声明总开关关掉时，即使今天还没确认过也直接放行，不强行卡住。
    if (config && !config.securityDisclaimer.enabled) setDisclaimerConfirmed(true);
  }, [config]);

  useEffect(() => {
    if (!announcement) return;
    const timer = window.setTimeout(() => setAnnouncement(null), announcement.durationMs);
    return () => window.clearTimeout(timer);
  }, [announcement]);

  function confirmDisclaimer() {
    localStorage.setItem(securityDisclaimerKey, todayKey());
    setDisclaimerConfirmed(true);
  }

  useEffect(() => {
    // 管理入口故意不放在普通页面按钮里：地址加 #admin，或按 Ctrl/Command + Shift + A。
    function openHiddenAdmin(event: KeyboardEvent) {
      if ((event.ctrlKey || event.metaKey) && event.shiftKey && event.key.toLowerCase() === "a") {
        window.location.hash = "admin";
        setView("admin");
      }
    }
    function openFromHash() {
      if (isAdminRoute()) setView("admin");
    }
    if (isAdminRoute()) setView("admin");
    window.addEventListener("keydown", openHiddenAdmin);
    window.addEventListener("hashchange", openFromHash);
    return () => {
      window.removeEventListener("keydown", openHiddenAdmin);
      window.removeEventListener("hashchange", openFromHash);
    };
  }, []);

  useEffect(() => {
    if (!me || isAdminRoute()) return;
    ask(view === "room" ? "lobby:unsubscribe" : "lobby:subscribe", {}).catch(() => undefined);
  }, [view, me?.player.id]);

  useEffect(() => {
    if (!me || !lobby) return;
    const latest = lobby.players.find((player) => player.id === me.player.id);
    if (!latest) return;
    // 实时大厅快照会省略 GameStats；先按精简视图契约合并再比较，否则不仅会把
    // 个人设置里的分游戏战绩覆盖成 0，保留后还会因比较键不同造成重复 setState。
    const nextPlayer = mergeLobbyPlayerIntoFullPlayer(me.player, latest);
    if (playerSyncKey(nextPlayer) !== playerSyncKey(me.player)) {
      // LobbyPlayer 是精简视图，禁止整对象覆盖 join 时的完整 PublicPlayer（会冲掉冷却等字段）。
      setMe((old) => {
        if (!old) return old;
        const merged = mergeLobbyPlayerIntoFullPlayer(old.player, latest);
        // 头像空串/缺省：batch 路径同样显式写回，避免残留旧 URL。
        if (latest.avatarUrl === undefined || latest.avatarUrl === "") {
          merged.avatarUrl = latest.avatarUrl || "";
        }
        return { ...old, player: merged };
      });
    }
  }, [lobby, me]);

  // 声明页先于"正在连接服务器"展示：WS 连接（上面的 useEffect）与声明页的强制停留同时进行，
  // 不用先等连上服务器才弹声明，省掉两段等待叠加的时间。
  if (!disclaimerConfirmed) return <SecurityDisclaimer onConfirm={confirmDisclaimer} />;
  if (!config) return <div className="loading">正在连接服务器...</div>;
  const leaderboardSource = leaderboardPlayersSnapshot.length ? leaderboardPlayersSnapshot : lobby?.players || [];

  return (
    <main>
      <header className="topbar">
        <div>
          <h1>{config.site.name}</h1>
          {view === "room" ? (
            <>
              <span className={`connection-pill ${connectionState}`}>
                {connectionState === "connected"
                  ? `在线 ${lobbyOnlineCount(lobby)} 人`
                  : connectionStateText(connectionState)}
              </span>
              {room && (
                <span className="top-summary">⚔️ {phaseText(room.phase, room.settings.gameId)}</span>
              )}
            </>
          ) : (
            <span className={`connection-pill ${connectionState}`}>
              {connectionState === "connected"
                ? `在线 ${lobbyOnlineCount(lobby)} 人`
                : connectionStateText(connectionState)}
            </span>
          )}
        </div>
        <div className="top-actions">
          {me && <PlayerBadge player={me.player} compact />}
          <button className="soft-button top-sponsor-button" title="关于" onClick={() => setAboutOpen(true)}>
            <Info size={18} /> <span>关于</span>
          </button>
          {me && (
            <button className="soft-button top-profile-button" title="个人设置" onClick={() => setProfileOpen(true)}>
              <UserRound size={18} /> <span>个人设置</span>
            </button>
          )}
          {me && lobby && (
            <button className="soft-button top-leaderboard-button" title="排行榜" onClick={() => setLeaderboardOpen(true)}>
              <Crown size={18} /> <span>排行榜</span>
            </button>
          )}
        </div>
      </header>
      {toastText && (
        <div className={`notice toast-notice ${toastLeaving ? "toast-leave" : "toast-enter"}`} role="status">
          {toastText}
        </div>
      )}
      {announcement && (
        <div className="announcement-popup" role="alert">
          <div>
            <b>全服公告</b>
            <p>{announcement.message}</p>
          </div>
          <button className="icon-button" type="button" aria-label="关闭公告" onClick={() => setAnnouncement(null)}>×</button>
        </div>
      )}
      {remoteBuildId && remoteBuildId !== dismissedRemoteBuildId && (
        <div className="announcement-popup version-update-popup" role="alert">
          <div>
            <b>网站内容已更新</b>
            <p>检测到新版本，当前页面可能已经过期，建议刷新以获取最新功能与修复。</p>
          </div>
          <div className="version-update-actions">
            <button className="version-update-button version-update-refresh" type="button" onClick={() => window.location.reload()}>刷新</button>
            <button className="version-update-button version-update-ignore" type="button" onClick={() => setDismissedRemoteBuildId(remoteBuildId)}>忽略</button>
          </div>
        </div>
      )}
      {view === "login" && restoringSession && !restoreKickPending && <section className="panel">正在恢复登录状态...</section>}
      {view === "login" && restoreKickPending && (
        <section className="login-card kick-confirm-card">
          <h2>该账号已在其他设备登录</h2>
          <p className="hint">继续会把另一台设备顶下线（那边会收到提示）。确定要继续吗？</p>
          <div className="kick-confirm-actions">
            <button disabled={restoreKickBusy} onClick={() => { setRestoreKickPending(null); setView("login"); }}>取消</button>
            <button className="primary" disabled={restoreKickBusy} onClick={confirmRestoreKick}>{restoreKickBusy ? "登录中…" : "确定顶替登录"}</button>
          </div>
        </section>
      )}
      {view === "login" && !restoringSession && !restoreKickPending && <Login config={config} onDone={(next) => {
        if (next.token) localStorage.setItem(tokenKey, next.token);
        setMe(next);
        trackLoginSuccess("manual");
        initPushForPlayer(next.player.id);
        setRestoringSession(false);
        if (next.room) setRoom(normalizeRoomSnapshot(next.room));
        setView(isAdminRoute() ? "admin" : next.room ? "room" : "lobby");
        if (next.room?.phase === "punishment") setNotice("已恢复到未完成的惩罚房间。");
      }} onError={setNotice} />}
      {view === "lobby" && me && lobby && <Lobby config={config} lobby={lobby} me={me.player} punishmentTagPrefs={punishmentTagPrefs} onError={setNotice} onGoRoom={(nextRoom) => { if (nextRoom) setRoom(nextRoom); setView("room"); }} onPunishmentTagPrefsChange={(prefs) => { setPunishmentTagPrefs(prefs); writePunishmentTagPrefs(prefs); }} />}
      {view === "room" && me && room && <Room config={config} room={room} me={me.player} lobby={lobby} onBack={() => setView("lobby")} onError={setNotice} />}
      {view === "admin" && lobby && (
        <Suspense fallback={<div className="loading">正在加载后台管理…</div>}>
          <AdminPanel config={config} lobby={lobby} onBack={() => { if (window.location.hash === "#admin") window.location.hash = ""; setView(me ? "lobby" : "login"); }} onError={setNotice} />
        </Suspense>
      )}
      {view === "room" && !room && <section className="panel">你暂时不在房间里。</section>}
      {aboutOpen && <AboutPanel config={config} onClose={() => setAboutOpen(false)} onOpenHelp={() => { setAboutOpen(false); setHelpOpen(true); }} />}
      {helpOpen && <HelpPanel onClose={() => setHelpOpen(false)} />}
      {profileOpen && me && (
        <ProfilePanel
          config={config}
          me={me.player}
          theme={theme}
          onThemeChange={(next) => {
            setTheme(next);
            trackThemeToggle(next);
          }}
          onClose={() => setProfileOpen(false)}
          onUpdated={(player) => {
            setMe({ ...me, player });
            cacheJoinProfile(player);
          }}
          onError={setNotice}
          onLoggedOut={() => { window.location.reload(); }}
        />
      )}
      {leaderboardOpen && <GlobalLeaderboardPanel players={leaderboardSource} onClose={() => setLeaderboardOpen(false)} />}
    </main>
  );
}
