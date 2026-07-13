import { useEffect, useRef, useState } from "react";
import { Crown, HeartHandshake, Moon, Sun, UserRound } from "lucide-react";
import { socket } from "./main";
import type { AppConfig, ChatMessage, LobbySnapshot, PublicPlayer, RoomSnapshot } from "./shared/types";
import { dailyAnnouncementKey, leaderboardRefreshMs, tokenKey } from "./lib/constants";
import {
  bumpWsAuthRetryCount, connectSocketWithSession, getWsAuthRetryCount, hasCachedLogin,
  joinIdentityPayload, resetWsAuthRetryCount, sessionTokenLooksValid
} from "./lib/session";
import { ask, dailyAnnouncementSeenKey, isAdminRoute } from "./lib/rpc";
import {
  lobbyOnlineCount, mergeRoundHistory, normalizeConfig, normalizeLobbySnapshot,
  normalizeRoomSnapshot, normalizeRoundHistoryItem, playerSyncKey, replacePlayerInLobby, replacePlayerInRoom
} from "./lib/normalize";
import { appendCappedUnique, prependCappedUnique } from "./lib/uiHelpers";
import type { AnnouncementPayload, MeState } from "./lib/types";
import {
  AdminPanel, GlobalLeaderboardPanel, Lobby, Login, PlayerBadge, ProfilePanel, Room, SponsorPanel,
  connectionStateText, phaseText
} from "./ui/AppViews";

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
  const [sponsorOpen, setSponsorOpen] = useState(false);
  const [notice, setNotice] = useState("");
  const [announcement, setAnnouncement] = useState<AnnouncementPayload | null>(null);
  const [dailyAnnouncementOpen, setDailyAnnouncementOpen] = useState(false);
  const [connectionState, setConnectionState] = useState<"connected" | "connecting" | "disconnected">(() => socket.connected ? "connected" : "connecting");
  const [theme, setTheme] = useState<"light" | "dark">(() => (localStorage.getItem("rps-online-theme") === "dark" ? "dark" : "light"));
  const restoreInFlightRef = useRef(false);
  const hadConnectedRef = useRef(socket.connected);
  const latestLobbyPlayersRef = useRef<PublicPlayer[]>([]);
  const leaderboardSnapshotAtRef = useRef(0);

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
    if (!cachedName || !sessionTokenLooksValid(token)) {
      setRestoringSession(false);
      return;
    }
    // 未连上时绝不尝试 join，更不能因此清 token（旧逻辑会把“未连接”误判成坏 token）。
    if (!socket.connected) return;

    restoreInFlightRef.current = true;
    try {
      const next = await ask<MeState>("player:join", { name: cachedName, genderId: cachedGender, token, ...(await joinIdentityPayload()) });
      if (next.token) localStorage.setItem(tokenKey, next.token);
      setMe(next);
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
        return { ...old, player: { ...old.player, ...p }, room: old.room ? replacePlayerInRoom(old.room, { ...old.player, ...p }) : old.room };
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
    socket.on("room:closed", ({ message }: { message?: string }) => {
      setRoom(null);
      setMe((old) => old ? { ...old, roomId: undefined } : old);
      if (!isAdminRoute()) setView("lobby");
      setNotice(message || "房间已被管理员关闭。");
    });
    socket.on("config:update", (config: AppConfig) => {
      setConfig(normalizeConfig(config));
    });
    socket.on("chat:append", (message: ChatMessage) => {
      if (!message.roomId) {
        setLobby((old) => old ? { ...old, lobbyChat: appendCappedUnique(old.lobbyChat || [], message, 100) } : old);
        return;
      }
      setRoom((old) => {
        if (!old || message.roomId !== old.id) return old;
        return { ...old, chat: appendCappedUnique(old.chat || [], message, 200) };
      });
    });
    socket.on("suggestion:append", (suggestion: LobbySnapshot["suggestions"][number]) => {
      setLobby((old) => old ? { ...old, suggestions: prependCappedUnique(old.suggestions || [], suggestion, 100) } : old);
    });
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
      socket.off("room:closed");
      socket.off("config:update");
      socket.off("chat:append");
      socket.off("suggestion:append");
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
    if (!config) return;
    if (!config.dailyAnnouncement.enabled) {
      setDailyAnnouncementOpen(false);
      return;
    }
    const seenKey = dailyAnnouncementSeenKey(config);
    setDailyAnnouncementOpen(localStorage.getItem(dailyAnnouncementKey) !== seenKey);
  }, [config]);

  useEffect(() => {
    if (!announcement) return;
    const timer = window.setTimeout(() => setAnnouncement(null), announcement.durationMs);
    return () => window.clearTimeout(timer);
  }, [announcement]);

  function closeDailyAnnouncement() {
    if (!config) return;
    localStorage.setItem(dailyAnnouncementKey, dailyAnnouncementSeenKey(config));
    setDailyAnnouncementOpen(false);
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

  if (!config) return <div className="loading">正在连接服务器...</div>;
  const leaderboardSource = leaderboardPlayersSnapshot.length ? leaderboardPlayersSnapshot : lobby?.players || [];

  return (
    <main>
      <header className="topbar">
        <div>
          <h1>{config.site.name}</h1>
          <span className="top-summary">{view === "room" && room ? `⚔️ ${phaseText(room.phase)}` : lobby ? `当前连接 ${lobbyOnlineCount(lobby)} 人` : "正在连接"}</span>
          <span className={`connection-pill ${connectionState}`}>{connectionStateText(connectionState)}</span>
        </div>
        <div className="top-actions">
          {me && <PlayerBadge player={me.player} compact />}
          <button className="soft-button top-sponsor-button" title="赞助支持" onClick={() => setSponsorOpen(true)}>
            <HeartHandshake size={18} /> <span>赞助</span>
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
      {dailyAnnouncementOpen && (
        <div className="daily-announcement-backdrop" role="dialog" aria-modal="true" aria-labelledby="daily-announcement-title">
          <section className="daily-announcement-card">
            <div>
              <span className="daily-announcement-kicker">📢 每日公告</span>
              <h2 id="daily-announcement-title">{config.dailyAnnouncement.title}</h2>
              <p className="daily-announcement-content">{config.dailyAnnouncement.content}</p>
            </div>
            <button className="primary" type="button" onClick={closeDailyAnnouncement}>{config.dailyAnnouncement.buttonText}</button>
          </section>
        </div>
      )}
      {view === "login" && restoringSession && <section className="panel">正在恢复登录状态...</section>}
      {view === "login" && !restoringSession && <Login config={config} onDone={(next) => {
        if (next.token) localStorage.setItem(tokenKey, next.token);
        setMe(next);
        setRestoringSession(false);
        if (next.room) setRoom(normalizeRoomSnapshot(next.room));
        setView(isAdminRoute() ? "admin" : next.room ? "room" : "lobby");
        if (next.room?.phase === "punishment") setNotice("已恢复到未完成的惩罚房间。");
      }} onError={setNotice} />}
      {view === "lobby" && me && lobby && <Lobby config={config} lobby={lobby} me={me.player} onError={setNotice} onGoRoom={(nextRoom) => { if (nextRoom) setRoom(nextRoom); setView("room"); }} />}
      {view === "room" && me && room && <Room config={config} room={room} lobbySuggestions={lobby?.suggestions || []} me={me.player} onBack={() => setView("lobby")} onError={setNotice} />}
      {view === "admin" && lobby && <AdminPanel config={config} lobby={lobby} onBack={() => { if (window.location.hash === "#admin") window.location.hash = ""; setView(me ? "lobby" : "login"); }} onError={setNotice} />}
      {view === "room" && !room && <section className="panel">你暂时不在房间里。</section>}
      {sponsorOpen && <SponsorPanel onClose={() => setSponsorOpen(false)} />}
      {profileOpen && me && <ProfilePanel config={config} me={me.player} onClose={() => setProfileOpen(false)} onUpdated={(player) => { setMe({ ...me, player }); localStorage.setItem("rps-online-name", player.name); localStorage.setItem("rps-online-gender", player.genderId); }} onError={setNotice} />}
      {leaderboardOpen && <GlobalLeaderboardPanel players={leaderboardSource} onClose={() => setLeaderboardOpen(false)} />}
    </main>
  );
}
