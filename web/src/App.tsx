import { Suspense, lazy, useEffect, useRef, useState } from "react";
import { Crown, Info, Moon, Sun, UserRound } from "lucide-react";
import { socket } from "./main";
import type { AppConfig, ChatMessage, LobbySnapshot, PublicPlayer, RoomSnapshot } from "./shared/types";
import { leaderboardRefreshMs, playerSecretKey, securityDisclaimerKey, tokenKey } from "./lib/constants";
import {
  bumpWsAuthRetryCount, clearPlayerIdentity, connectSocketWithSession, getWsAuthRetryCount, hasCachedLogin,
  joinIdentityPayload, resetWsAuthRetryCount, sessionTokenLooksValid
} from "./lib/session";
import { ask, isAdminRoute, todayKey } from "./lib/rpc";
import {
  lobbyOnlineCount, mergeRoundHistory, normalizeConfig, normalizeLobbySnapshot,
  normalizeRoomSnapshot, normalizeRoundHistoryItem, playerSyncKey, replacePlayerInLobby, replacePlayerInRoom
} from "./lib/normalize";
import { refreshActiveChats } from "./lib/chatStore";
import { fetchPushPreferences, notifyMentionIfHidden, notifySeatIfHidden, notifyTurnIfHidden, setPushMeId } from "./lib/pushNotify";
import type { AnnouncementPayload, MeState } from "./lib/types";
import {
  AboutPanel, GlobalLeaderboardPanel, Lobby, Login, PlayerBadge, ProfilePanel, Room, SecurityDisclaimer,
  connectionStateText, phaseText
} from "./ui/AppViews";

// 后台管理面板（含可能新增的图表等重型组件）单独打包，普通玩家不会触发这次 import。
const AdminPanel = lazy(() => import("./ui/AdminViews").then((module) => ({ default: module.AdminPanel })));

// Level 1（页面在前台开着但被切到后台/最小化）：对比新旧房间快照，检测"对手座位刚被
// 坐满"和"轮到我了"这两类事件，命中就交给 notify* 决定要不要弹 Notification（它们内部
// 会检查 document.hidden 和用户的推送偏好开关，这里只管"事件本身有没有发生"）。
// 只在 old 和 new 是同一个房间时才比较，避免切房间那一刻误判成"状态变化"。
function checkRoomLevel1Notifications(old: RoomSnapshot | null, next: RoomSnapshot, me: MeState | null) {
  const meId = me?.player.id;
  if (!meId || !old || old.id !== next.id) return;
  const mySeat = next.seats.A && "id" in next.seats.A && next.seats.A.id === meId ? "A"
    : next.seats.B && "id" in next.seats.B && next.seats.B.id === meId ? "B" : null;

  // 大话骰不用 A/B 座位（参战名单在 liarsDice.participantIds 里），轮到我叫点/开牌靠
  // currentTurn（playerId，不是座位）切到我；没有"对面座位坐满"这个概念，不发那类通知。
  if (!mySeat) {
    if (next.settings.gameId === "liarsdice" && next.liarsDice) {
      if (old.liarsDice?.currentTurn !== meId && next.liarsDice.currentTurn === meId) notifyTurnIfHidden();
    }
    return;
  }
  const otherSeat = mySeat === "A" ? "B" : "A";

  // 对面座位刚从空/无人变成有真人坐下。
  const otherOccOld = old.seats[otherSeat];
  const otherOccNext = next.seats[otherSeat];
  const wasEmpty = !otherOccOld;
  const nowHuman = Boolean(otherOccNext) && !("isBot" in otherOccNext!);
  if (wasEmpty && nowHuman) notifySeatIfHidden();

  // 轮到我了：RPS 靠 choices（对方刚锁定、我还没锁定）；黑白棋/井字棋靠 turn 字段切到我。
  if (next.settings.gameId === "rps") {
    const otherJustChose = !old.choices?.[otherSeat] && Boolean(next.choices?.[otherSeat]);
    const iHaventChosen = !next.choices?.[mySeat];
    if (otherJustChose && iHaventChosen) notifyTurnIfHidden();
  } else if (next.settings.gameId === "othello" && next.othello) {
    if (old.othello?.turn !== mySeat && next.othello.turn === mySeat) notifyTurnIfHidden();
  } else if (next.settings.gameId === "tictactoe" && next.tictactoe) {
    if (old.tictactoe?.turn !== mySeat && next.tictactoe.turn === mySeat) notifyTurnIfHidden();
  } else if (next.settings.gameId === "gomoku" && next.gomoku) {
    if (old.gomoku?.turn !== mySeat && next.gomoku.turn === mySeat) notifyTurnIfHidden();
  }
}

export function App() {
  const [config, setConfig] = useState<AppConfig | null>(null);
  const [lobby, setLobby] = useState<LobbySnapshot | null>(null);
  const [room, setRoom] = useState<RoomSnapshot | null>(null);
  const [me, setMe] = useState<MeState | null>(null);
  const [leaderboardPlayersSnapshot, setLeaderboardPlayersSnapshot] = useState<PublicPlayer[]>([]);
  const [view, setView] = useState<"login" | "lobby" | "room" | "admin">(() => isAdminRoute() ? "admin" : "login");
  // 有本地登录缓存时先进入恢复态，避免刷新时先闪一下登录页再进大厅。
  const [restoringSession, setRestoringSession] = useState(() => !isAdminRoute() && hasCachedLogin());
  const [profileOpen, setProfileOpen] = useState(false);
  const [leaderboardOpen, setLeaderboardOpen] = useState(false);
  const [aboutOpen, setAboutOpen] = useState(false);
  const [notice, setNotice] = useState("");
  const [announcement, setAnnouncement] = useState<AnnouncementPayload | null>(null);
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
  // meRef/roomRef：socket 事件回调注册一次（[] deps）就不会再拿到最新的闭包，读 ref 才是当前值。
  const meRef = useRef<MeState | null>(null);
  const roomRef = useRef<RoomSnapshot | null>(null);

  useEffect(() => {
    meRef.current = me;
  }, [me]);
  useEffect(() => {
    roomRef.current = room;
  }, [room]);

  useEffect(() => {
    connectSocketWithSession().catch(() => {
      setConnectionState("disconnected");
      setRestoringSession(false);
      setNotice("连接失败，请检查网络后刷新重试。");
    });
  }, []);

  async function restoreSession(options: { showRecoveredNotice?: boolean } = {}) {
    if (restoreInFlightRef.current) return;
    const token = localStorage.getItem(tokenKey);
    const cachedName = localStorage.getItem("rps-online-name") || "";
    const cachedGender = localStorage.getItem("rps-online-gender") || "male";
    const cachedCustomGender = localStorage.getItem("rps-online-custom-gender") || "";
    const cachedFaction = localStorage.getItem("rps-online-faction") || "";
    if (!cachedName || !sessionTokenLooksValid(token)) {
      setRestoringSession(false);
      return;
    }
    // 未连上时绝不尝试 join，更不能因此清 token（旧逻辑会把“未连接”误判成坏 token）。
    if (!socket.connected) return;

    restoreInFlightRef.current = true;
    // 阵营现在独立于性别选择，重连/刷新页面时必须把缓存的 factionId/customGenderLabel
    // 一起带上——不传就会被服务端 applyGender 兜底成第一个已配置阵营，等于每次刷新都重置阵营。
    const payload = { name: cachedName, genderId: cachedGender, customGenderLabel: cachedCustomGender, factionId: cachedFaction, token, ...(await joinIdentityPayload()) };
    try {
      const next = await ask<MeState & { alreadyOnline?: true }>("player:join", payload);
      if (next.alreadyOnline) {
        setRestoreKickPending(payload);
        setRestoringSession(false);
        return;
      }
      if (next.token) localStorage.setItem(tokenKey, next.token);
      if (next.reissuedSecret) localStorage.setItem(playerSecretKey, next.reissuedSecret);
      setMe(next);
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

  function initPushForPlayer(playerId: string) {
    setPushMeId(playerId);
    fetchPushPreferences().catch(() => undefined);
  }

  function refreshLeaderboardSnapshot(players: PublicPlayer[], now = Date.now()) {
    latestLobbyPlayersRef.current = players;
    leaderboardSnapshotAtRef.current = now;
    setLeaderboardPlayersSnapshot(players);
  }

  useEffect(() => {
    function handleChatNewForMention(msg: ChatMessage) {
      const meId = meRef.current?.player.id;
      if (!meId || !msg || !Array.isArray(msg.mentions) || !msg.mentions.includes(meId)) return;
      notifyMentionIfHidden(msg.author, msg.text);
    }
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
      checkRoomLevel1Notifications(roomRef.current, normalizeRoomSnapshot(nextRoom), meRef.current);
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
    // Level 1 @提及提醒：只做检测/弹通知，不参与消息列表渲染（那是 chatStore 自己的事）。
    // 注意：chatStore.ts 在模块加载时也注册了一个独立的 "chat:new" 监听（渲染消息列表用），
    // socket.off(event) 不带 handler 参数会清空该事件的整个监听集合——必须传具体函数引用，
    // 否则下面 cleanup 里的 socket.off("chat:new") 会把 chatStore 的监听也一起干掉。
    socket.on("chat:new", handleChatNewForMention);
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
    function applyPlayerPatches(players: PublicPlayer[]) {
      if (!players?.length) return;
      const byId = new Map(players.map((p) => [p.id, p]));
      latestLobbyPlayersRef.current = latestLobbyPlayersRef.current.map((item) => byId.get(item.id) || item);
      for (const p of players) {
        if (!latestLobbyPlayersRef.current.some((x) => x.id === p.id)) latestLobbyPlayersRef.current.push(p);
      }
      setLobby((old) => {
        if (!old) return old;
        let next = old;
        for (const p of players) next = replacePlayerInLobby(next, p);
        return next;
      });
      setRoom((old) => {
        if (!old) return old;
        let next = old;
        for (const p of players) next = replacePlayerInRoom(next, p);
        return next;
      });
      setMe((old) => {
        if (!old) return old;
        const p = byId.get(old.player.id);
        if (!p) return old;
        // LobbyPlayer 省略空 avatarUrl；合并时必须显式清空，否则旧头像会残留。
        const merged = { ...old.player, ...p, avatarUrl: p.avatarUrl || undefined };
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
    // 聊天（房间 + 大厅留言板）已迁到 chatStore：chat:new / chat:cleared 由其内部监听，
    // 首屏与历史走 chat:load / chat:loadOlder，此处不再处理。
    socket.on("announcement:show", (payload: AnnouncementPayload) => {
      setAnnouncement(payload);
    });
    socket.on("connect", () => {
      setConnectionState("connected");
      resetWsAuthRetryCount();
      const isReconnect = hadConnectedRef.current;
      hadConnectedRef.current = true;
      // 首次连接不弹「已恢复」；断线重连后才提示。
      void restoreSession({ showRecoveredNotice: isReconnect });
      // 重连后补拉已激活聊天频道的最近消息（chatStore 不自注册 connect，见其注释）。
      refreshActiveChats();
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
      socket.off("chat:new", handleChatNewForMention);
      socket.off("player:update");
      socket.off("player:batch");
      socket.off("player:kicked");
      socket.off("session:kicked");
      socket.off("room:closed");
      socket.off("config:update");
      socket.off("announcement:show");
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
    const timer = window.setTimeout(() => setNotice(""), 3500);
    return () => window.clearTimeout(timer);
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
    if (latest && playerSyncKey(latest) !== playerSyncKey(me.player)) {
      setMe((old) => old ? { ...old, player: latest } : old);
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
          <span className="top-summary">{view === "room" && room ? `⚔️ ${phaseText(room.phase, room.settings.gameId)}` : lobby ? `当前连接 ${lobbyOnlineCount(lobby)} 人` : "正在连接"}</span>
          <span className={`connection-pill ${connectionState}`}>{connectionStateText(connectionState)}</span>
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
          <button className="icon-button" title={theme === "dark" ? "切换到日间模式" : "切换到夜间模式"} onClick={() => setTheme(theme === "dark" ? "light" : "dark")}>
            {theme === "dark" ? <Sun size={18} /> : <Moon size={18} />}
          </button>
        </div>
      </header>
      {notice && <div className="notice">{notice}</div>}
      {announcement && (
        <div className="announcement-popup" role="alert">
          <div>
            <b>全服公告</b>
            <p>{announcement.message}</p>
          </div>
          <button className="icon-button" type="button" aria-label="关闭公告" onClick={() => setAnnouncement(null)}>×</button>
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
        initPushForPlayer(next.player.id);
        setRestoringSession(false);
        if (next.room) setRoom(normalizeRoomSnapshot(next.room));
        setView(isAdminRoute() ? "admin" : next.room ? "room" : "lobby");
        if (next.room?.phase === "punishment") setNotice("已恢复到未完成的惩罚房间。");
      }} onError={setNotice} />}
      {view === "lobby" && me && lobby && <Lobby config={config} lobby={lobby} me={me.player} onError={setNotice} onGoRoom={(nextRoom) => { if (nextRoom) setRoom(nextRoom); setView("room"); }} />}
      {view === "room" && me && room && <Room config={config} room={room} me={me.player} lobby={lobby} onBack={() => setView("lobby")} onError={setNotice} />}
      {view === "admin" && lobby && (
        <Suspense fallback={<div className="loading">正在加载后台管理…</div>}>
          <AdminPanel config={config} lobby={lobby} onBack={() => { if (window.location.hash === "#admin") window.location.hash = ""; setView(me ? "lobby" : "login"); }} onError={setNotice} />
        </Suspense>
      )}
      {view === "room" && !room && <section className="panel">你暂时不在房间里。</section>}
      {aboutOpen && <AboutPanel config={config} onClose={() => setAboutOpen(false)} />}
      {profileOpen && me && (
        <ProfilePanel
          config={config}
          me={me.player}
          onClose={() => setProfileOpen(false)}
          onUpdated={(player) => {
            setMe({ ...me, player });
            localStorage.setItem("rps-online-name", player.name);
            localStorage.setItem("rps-online-gender", player.genderId);
            localStorage.setItem("rps-online-custom-gender", player.genderId ? "" : player.genderLabel);
            localStorage.setItem("rps-online-faction", player.factionId);
          }}
          onError={setNotice}
          onLoggedOut={() => { window.location.reload(); }}
        />
      )}
      {leaderboardOpen && <GlobalLeaderboardPanel players={leaderboardSource} onClose={() => setLeaderboardOpen(false)} />}
    </main>
  );
}
