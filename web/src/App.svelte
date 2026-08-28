<script lang="ts">
  // 源：App.tsx（643 行）。外壳：会话恢复、视图路由、顶部栏、全局 socket 订阅。
  import { untrack } from "svelte";
  import Crown from "@lucide/svelte/icons/crown";
  import Info from "@lucide/svelte/icons/info";
  import UserRound from "@lucide/svelte/icons/user-round";
  import { socket } from "./ws";
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
  import { connectionStateText, phaseText } from "./lib/playerDisplay";
  import SecurityDisclaimer from "./ui/shell/SecurityDisclaimer.svelte";
  import Login from "./ui/shell/Login.svelte";
  import PlayerBadge from "./ui/shell/PlayerBadge.svelte";
  import Lobby from "./ui/lobby/Lobby.svelte";
  import Room from "./ui/room/Room.svelte";
  import AdminPanel from "./ui/admin/AdminPanel.svelte";
  import ContributeView from "./ui/contribute/ContributeView.svelte";
  import AboutPanel from "./ui/about/AboutPanel.svelte";
  import HelpPanel from "./ui/about/HelpPanel.svelte";
  import ProfilePanel from "./ui/profile/ProfilePanel.svelte";
  import GlobalLeaderboardPanel from "./ui/social/GlobalLeaderboardPanel.svelte";
  import { startAnalytics, trackLoginSuccess, trackPageview, trackThemeToggle } from "./lib/analytics";

  type AppView = "login" | "lobby" | "room" | "admin" | "contribute";

  // 大型 FULL/DELTA 快照：整体替换语义，用 $state.raw 避免深度 Proxy 化整棵树
  // （见 plan.md §6.6）。
  let config = $state.raw<AppConfig | null>(null);
  let lobby = $state.raw<LobbySnapshot | null>(null);
  let room = $state.raw<RoomSnapshot | null>(null);
  let me = $state.raw<MeState | null>(null);
  // 随机任务开房标签偏好：纯本地浏览器存储，不再随 player:join 从服务端下发。
  let punishmentTagPrefs = $state<Record<string, string>>(readPunishmentTagPrefs());
  let leaderboardPlayersSnapshot = $state.raw<PublicPlayer[]>([]);
  let view = $state<AppView>(isAdminRoute() ? "admin" : "login");
  let viewBeforeAdmin: AppView = isAdminRoute() ? "login" : "lobby";
  let keepContribute = $state(false);
  // 有本地登录缓存时先进入恢复态，避免刷新时先闪一下登录页再进大厅。
  let restoringSession = $state(!isAdminRoute() && hasCachedLogin());
  let profileOpen = $state(false);
  let leaderboardOpen = $state(false);
  let aboutOpen = $state(false);
  let helpOpen = $state(false);
  let notice = $state("");
  /** toast 展示文案与离场态：notice 更新时同步文案，定时先 leave 再清空，以便播放退出动效。 */
  let toastText = $state("");
  let toastLeaving = $state(false);
  let announcement = $state.raw<AnnouncementPayload | null>(null);
  // 后端构建版本号与本次前端构建不一致时置位；只做提示，不自动刷新（避免误判/多开
  // 场景下把用户的未保存输入或对局强制打断，也从根上杜绝"版本提示触发自动刷新"
  // 被用来当作 DoS 放大手段的可能）。dismissedRemoteBuildId 记录用户已经关掉提示的
  // 那个版本号：同一版本重复到达（心跳每 25s 一次）不会让横幅重新弹出；只有服务端
  // 又发布了新的、不同的版本号，才会再次出现。
  let remoteBuildId = $state<string | null>(null);
  let dismissedRemoteBuildId = $state<string | null>(null);
  // 每天每个浏览器只需确认一次；未过期就跳过声明页，直接走原来的连接流程。
  let disclaimerConfirmed = $state(localStorage.getItem(securityDisclaimerKey) === todayKey());
  let connectionState = $state<"connected" | "connecting" | "disconnected">(socket.connected ? "connected" : "connecting");
  let restoreKickPending = $state.raw<Record<string, unknown> | null>(null);
  let restoreKickBusy = $state(false);
  let theme = $state<"light" | "dark">(localStorage.getItem("rps-online-theme") === "dark" ? "dark" : "light");

  // 纯可变盒子，不参与渲染，不需要 $state（对应原 React 版的若干 useRef）。
  let restoreInFlight = false;
  let restoreWaiters: Array<(ok: boolean) => void> = [];
  let hadConnected = socket.connected;
  let latestLobbyPlayers: PublicPlayer[] = [];
  let leaderboardSnapshotAt = 0;

  $effect(() => {
    startAnalytics();
    connectSocketWithSession().catch(() => {
      connectionState = "disconnected";
      restoringSession = false;
      notice = "连接失败，请检查网络后刷新重试。";
    });
  });

  // 主视图 pageview
  $effect(() => {
    trackPageview(view);
  });

  // 弹窗 pageview（各自只追踪自己的开关，不读其余状态）
  $effect(() => { if (profileOpen) trackPageview("profile"); });
  $effect(() => { if (leaderboardOpen) trackPageview("leaderboard"); });
  $effect(() => { if (aboutOpen) trackPageview("about"); });
  $effect(() => { if (helpOpen) trackPageview("help"); });

  async function restoreSession(options: { showRecoveredNotice?: boolean } = {}): Promise<boolean> {
    if (restoreInFlight) {
      return await new Promise<boolean>((resolve) => {
        restoreWaiters.push(resolve);
      });
    }
    const token = localStorage.getItem(tokenKey);
    const cachedName = localStorage.getItem("rps-online-name") || "";
    const { genderId: cachedGender } = readCachedJoinGender();
    if (!cachedName || !sessionTokenLooksValid(token)) {
      restoringSession = false;
      return false;
    }
    // 未连上时绝不尝试 join，更不能因此清 token（旧逻辑会把”未连接”误判成坏 token）。
    if (!socket.connected) return false;

    restoreInFlight = true;
    let ok = false;
    const payload = { name: cachedName, genderId: cachedGender, token, ...(await joinIdentityPayload()) };
    try {
      const next = await ask<MeState & { alreadyOnline?: true }>("player:join", payload);
      if (next.alreadyOnline) {
        restoreKickPending = payload;
        restoringSession = false;
        return false;
      }
      if (next.token) localStorage.setItem(tokenKey, next.token);
      if (next.reissuedSecret) localStorage.setItem(playerSecretKey, next.reissuedSecret);
      // 以服务端确认后的资料为准回写缓存，避免清洗后的资料与本地不一致。
      if (next.player) cacheJoinProfile(next.player);
      me = next;
      trackLoginSuccess("restore");
      initPushForPlayer(next.player.id);
      room = next.room ? normalizeRoomSnapshot(next.room) : null;
      if (!isAdminRoute()) {
        if (next.room) {
          view = "room";
          if (next.room.phase === "punishment") notice = "已恢复到未完成的惩罚房间。";
        } else if (view !== "contribute") {
          view = "lobby";
          if (options.showRecoveredNotice) notice = "连接已恢复，玩家状态已同步。";
        } else if (options.showRecoveredNotice) {
          notice = "连接已恢复，玩家状态已同步。";
        }
      }
      ok = true;
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
        me = null;
        room = null;
        if (!isAdminRoute()) view = "login";
      }
      if (options.showRecoveredNotice) notice = "连接已恢复，但玩家状态同步失败，请刷新或重新进入。";
    } finally {
      restoreInFlight = false;
      restoringSession = false;
      const waiters = restoreWaiters.splice(0);
      for (const wait of waiters) wait(ok);
    }
    return ok;
  }

  async function confirmRestoreKick() {
    if (!restoreKickPending) return;
    restoreKickBusy = true;
    try {
      const next = await ask<MeState>("player:join", { ...restoreKickPending, forceKick: true });
      if (next.token) localStorage.setItem(tokenKey, next.token);
      if (next.reissuedSecret) localStorage.setItem(playerSecretKey, next.reissuedSecret);
      if (next.player) cacheJoinProfile(next.player);
      me = next;
      initPushForPlayer(next.player.id);
      room = next.room ? normalizeRoomSnapshot(next.room) : null;
      if (!isAdminRoute()) view = next.room ? "room" : "lobby";
      restoreKickPending = null;
    } catch (error) {
      notice = error instanceof Error ? error.message : "登录失败";
    } finally {
      restoreKickBusy = false;
    }
  }

  function initPushForPlayer(_playerId: string) {
    fetchPushPreferences().catch(() => undefined);
    if (typeof Notification !== "undefined" && Notification.permission === "granted") {
      ensurePushSubscription().catch(() => undefined);
    }
  }

  function refreshLeaderboardSnapshot(players: PublicPlayer[], now = Date.now()) {
    latestLobbyPlayers = players;
    leaderboardSnapshotAt = now;
    leaderboardPlayersSnapshot = players;
  }

  function applyPlayerPatches(rawList: PublicPlayer[]) {
    if (!rawList?.length) return;
    const players = rawList.map(normalizePublicPlayer);
    const byId = new Map(players.map((p) => [p.id, p]));
    // O(P+k) 合并本地排行榜快照源，避免每条 patch 都 some 扫描。
    const prevSnap = latestLobbyPlayers;
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
    latestLobbyPlayers = nextSnap;

    if (lobby) lobby = replacePlayersInLobby(lobby, players);
    if (room) room = replacePlayersInRoom(room, players);
    if (me) {
      const p = byId.get(me.player.id);
      if (p) {
        // LobbyPlayer/typed PublicPlayer 会省略空串字段：avatarUrl 清空时会变成 undefined。
        // 合并时必须显式写回空值，否则旧头像会残留在 me 上。
        const merged = {
          ...mergeLobbyPlayerIntoFullPlayer(me.player, p),
          genderId: typeof p.genderId === "string" ? p.genderId : "",
          avatarUrl: p.avatarUrl || undefined
        };
        me = { ...me, player: merged, room: me.room ? replacePlayerInRoom(me.room, merged) : me.room };
      }
    }
  }

  $effect(() => {
    socket.on("lobby:update", (nextLobby: LobbySnapshot) => {
      if (nextLobby.config) config = normalizeConfig(nextLobby.config);
      const normalized = normalizeLobbySnapshot(nextLobby);
      latestLobbyPlayers = normalized.players;
      const now = Date.now();
      if (!leaderboardSnapshotAt || now - leaderboardSnapshotAt >= leaderboardRefreshMs) {
        refreshLeaderboardSnapshot(normalized.players, now);
      }
      lobby = normalizeLobbySnapshot(nextLobby, lobby ?? undefined);
    });
    socket.on("room:update", (nextRoom: RoomSnapshot) => {
      const normalized = normalizeRoomSnapshot(nextRoom);
      const old = room;
      // updatedAt 缺失/非数字时不要丢弃更新（protobuf 漏字段会导致永远用旧房态）
      const nextAt = Number(normalized.updatedAt) || 0;
      const oldAt = Number(old?.updatedAt) || 0;
      if (old?.id === normalized.id && nextAt > 0 && oldAt > 0 && nextAt < oldAt) {
        // 过期的乱序包，丢弃
      } else if (old?.id === normalized.id) {
        room = {
          ...normalized,
          // chat 走 chat:append；全量包若未带 chat 则保留本地
          chat: (normalized.chat || []).length === 0 ? (old.chat || []) : normalized.chat,
          // history 必须合并：惩罚任务/证明写在 latest history，不能因空数组丢弃，也不能整包忽略更新
          roundHistory: mergeRoundHistory(old.roundHistory, normalized.roundHistory)
        };
      } else {
        room = normalized;
      }
      if (!isAdminRoute()) view = "room";
    });
    socket.on("room:historyAppend", (payload: { roomId?: string; item?: RoomSnapshot["roundHistory"][number]; total?: number }) => {
      const roomId = payload?.roomId;
      const item = payload?.item;
      if (!roomId || !item) return;
      const safeItem = normalizeRoundHistoryItem(item);
      if (room?.id === roomId) {
        room = {
          ...room,
          roundHistory: mergeRoundHistory(room.roundHistory, [safeItem]),
          roundHistoryTotal: Number(payload.total) || room.roundHistoryTotal
        };
      }
    });
    // 聚合玩家更新（player:batch 为 LobbyPlayer[]；兼容单条 player:update）
    socket.on("player:batch", (list: PublicPlayer[]) => applyPlayerPatches(Array.isArray(list) ? list : []));
    socket.on("player:update", (player: PublicPlayer) => applyPlayerPatches(player ? [player] : []));
    socket.on("player:kicked", () => {
      localStorage.removeItem(tokenKey);
      me = null;
      room = null;
      view = "login";
      notice = "你已被管理员移出。";
    });
    // 同一账号在另一台设备完成了「顶替登录」确认：本设备的登录状态到此结束，但本地
    // playerId/playerSecret 仍然保留（这台设备依然是受信任设备之一，不是登出）。
    socket.on("session:kicked", ({ message }: { message?: string }) => {
      localStorage.removeItem(tokenKey);
      me = null;
      room = null;
      view = "login";
      notice = message || "你的账号已在其他设备登录，此会话已结束，请刷新页面重新登录。";
    });
    socket.on("room:closed", ({ message }: { message?: string }) => {
      room = null;
      if (me) me = { ...me, roomId: undefined };
      if (!isAdminRoute()) view = "lobby";
      notice = message || "房间已被管理员关闭。";
    });
    socket.on("config:update", (nextConfig: AppConfig) => {
      config = normalizeConfig(nextConfig);
    });
    // 聊天（房间 + 大厅留言板）已迁到 chatStore：chat:new / chat:deleted 由其内部监听，
    // 首屏与历史走 chat:load / chat:loadOlder，此处不再处理。
    socket.on("announcement:show", (payload: AnnouncementPayload) => {
      announcement = payload;
    });
    // server:hello：连接建立时的一次性推送 + 心跳应答兜底（见 ws.ts），带的是后端
    // 当前构建版本号。"dev"/"unknown" 是本地开发或非 git checkout 的兜底字面量，
    // 双方都可能出现，出现即视为不可比较，不触发提示。
    socket.on("server:hello", ({ buildId }: { buildId?: string }) => {
      if (!buildId || buildId === "dev" || buildId === "unknown") return;
      if (__APP_BUILD_ID__ === "dev" || __APP_BUILD_ID__ === "unknown") return;
      if (buildId === __APP_BUILD_ID__) return;
      remoteBuildId = buildId;
    });
    socket.on("connect", () => {
      connectionState = "connected";
      resetWsAuthRetryCount();
      const isReconnect = hadConnected;
      hadConnected = true;
      // 首次连接不弹「已恢复」；断线重连后才提示。
      // 重连后补拉已激活聊天频道的最近消息（chatStore 不自注册 connect，见其注释）；
      // 必须等 restoreSession 里的 player:join 落地后再发，否则新连接还没绑定玩家/房间，
      // 房间聊天的 chat:load 会被 onChatLoad 以"你不在这个房间里"拒绝，断线期间的消息就永久错过了。
      void restoreSession({ showRecoveredNotice: isReconnect }).then(() => refreshActiveChats());
    });
    socket.on("disconnect", () => {
      connectionState = "disconnected";
      // iOS 切后台也会触发 disconnect；文案轻一点，避免误以为故障
      notice = "连接已断开，正在重连…";
    });
    socket.on("connect_error", (error: Error & { data?: { code?: string } }) => {
      connectionState = "disconnected";
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
          restoringSession = false;
          notice = "连接认证失败，请刷新页面重试。";
          return;
        }
        bumpWsAuthRetryCount();
        localStorage.removeItem(tokenKey);
        connectSocketWithSession({ forceNewToken: true }).catch(() => {
          connectionState = "disconnected";
          restoringSession = false;
          notice = "连接认证失败，请刷新页面重试。";
        });
        return;
      }
      restoringSession = false;
    });
    socket.io.on("reconnect_attempt", () => { connectionState = "connecting"; });
    // 挂载时机竞态：WS 若在本 effect 运行前就已连上（例如热更新），必须在这里立刻补一次
    // restoreSession，否则 connect 事件已经错过，永远等不到自动登录。
    if (socket.connected) {
      connectionState = "connected";
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
  });

  $effect(() => {
    const timer = window.setInterval(() => {
      if (latestLobbyPlayers.length) refreshLeaderboardSnapshot(latestLobbyPlayers);
    }, leaderboardRefreshMs);
    return () => window.clearInterval(timer);
  });

  $effect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("rps-online-theme", theme);
  });

  $effect(() => {
    if (!notice) return;
    toastText = notice;
    toastLeaving = false;
    const leaveTimer = window.setTimeout(() => { toastLeaving = true; }, 3200);
    const clearTimer = window.setTimeout(() => {
      toastText = "";
      toastLeaving = false;
      notice = "";
    }, 3520);
    return () => {
      window.clearTimeout(leaveTimer);
      window.clearTimeout(clearTimer);
    };
  });

  $effect(() => {
    // 管理员把声明总开关关掉时，即使今天还没确认过也直接放行，不强行卡住。
    if (config && !config.securityDisclaimer.enabled) disclaimerConfirmed = true;
  });

  $effect(() => {
    if (!announcement) return;
    const current = announcement;
    const timer = window.setTimeout(() => {
      if (announcement === current) announcement = null;
    }, current.durationMs);
    return () => window.clearTimeout(timer);
  });

  function confirmDisclaimer() {
    localStorage.setItem(securityDisclaimerKey, todayKey());
    disclaimerConfirmed = true;
  }

  function openAdmin() {
    const current = view;
    if (current !== "admin") {
      viewBeforeAdmin = current;
      if (current === "contribute") keepContribute = true;
    }
    if (!isAdminRoute()) window.location.hash = "admin";
    view = "admin";
  }

  function leaveAdmin() {
    if (window.location.hash === "#admin") window.location.hash = "";
    const prev = viewBeforeAdmin;
    if (prev === "contribute" && me) {
      view = "contribute";
      return;
    }
    if (prev === "room" && me && room) {
      view = "room";
      return;
    }
    keepContribute = false;
    view = me ? "lobby" : "login";
  }

  $effect(() => {
    // 管理入口故意不放在普通页面按钮里：地址加 #admin，或按 Ctrl/Command + Shift + A。
    function openHiddenAdmin(event: KeyboardEvent) {
      if ((event.ctrlKey || event.metaKey) && event.shiftKey && event.key.toLowerCase() === "a") {
        openAdmin();
      }
    }
    function openFromHash() {
      if (isAdminRoute()) openAdmin();
    }
    if (isAdminRoute()) openAdmin();
    window.addEventListener("keydown", openHiddenAdmin);
    window.addEventListener("hashchange", openFromHash);
    return () => {
      window.removeEventListener("keydown", openHiddenAdmin);
      window.removeEventListener("hashchange", openFromHash);
    };
  });

  // 原 deps 是 [view, me?.player.id]：只在「视图切换」或「玩家身份变化」时才发订阅 RPC，
  // 玩家资料的其余字段变动（player:batch 高频推送）不应触发。meId 是 $derived，Svelte
  // 只在其输出值真正变化时才使下游失效，天然复现了这条窄依赖（见 plan.md §6.2）。
  let meId = $derived(me?.player.id);
  $effect(() => {
    const currentView = view;
    const id = meId;
    if (!id || isAdminRoute()) return;
    untrack(() => ask(currentView === "room" ? "lobby:unsubscribe" : "lobby:subscribe", {}).catch(() => undefined));
  });

  // 实时大厅快照会省略 GameStats；先按精简视图契约合并再比较，否则不仅会把个人设置里的
  // 分游戏战绩覆盖成 0，保留后还会因比较键不同造成重复写入。收敛依赖 playerSyncKey 的
  // 幂等比较：合并后不再变化就不再写 me，effect 自然停止重复触发（与原 React 版同构）。
  $effect(() => {
    if (!me || !lobby) return;
    const latest = lobby.players.find((player) => player.id === me!.player.id);
    if (!latest) return;
    const nextPlayer = mergeLobbyPlayerIntoFullPlayer(me.player, latest);
    if (playerSyncKey(nextPlayer) !== playerSyncKey(me.player)) {
      // LobbyPlayer 是精简视图，禁止整对象覆盖 join 时的完整 PublicPlayer（会冲掉冷却等字段）。
      const merged = mergeLobbyPlayerIntoFullPlayer(me.player, latest);
      // 头像空串/缺省：batch 路径同样显式写回，避免残留旧 URL。
      if (latest.avatarUrl === undefined || latest.avatarUrl === "") {
        merged.avatarUrl = latest.avatarUrl || "";
      }
      me = { ...me, player: merged };
    }
  });

  let leaderboardSource = $derived(leaderboardPlayersSnapshot.length ? leaderboardPlayersSnapshot : lobby?.players || []);
</script>

<!-- 声明页先于"正在连接服务器"展示：WS 连接与声明页的强制停留同时进行，不用先等连上
     服务器才弹声明，省掉两段等待叠加的时间。 -->
{#if !disclaimerConfirmed}
  <SecurityDisclaimer onConfirm={confirmDisclaimer} />
{:else if !config}
  <div class="loading">正在连接服务器...</div>
{:else}
  <main>
    <header class="topbar">
      <div>
        <h1>{config.site.name}</h1>
        {#if view === "room"}
          <span class={`connection-pill ${connectionState}`}>
            {connectionState === "connected" ? `在线 ${lobbyOnlineCount(lobby)} 人` : connectionStateText(connectionState)}
          </span>
          {#if room}
            <span class="top-summary">⚔️ {phaseText(room.phase, room.settings.gameId)}</span>
          {/if}
        {:else}
          <span class={`connection-pill ${connectionState}`}>
            {connectionState === "connected" ? `在线 ${lobbyOnlineCount(lobby)} 人` : connectionStateText(connectionState)}
          </span>
        {/if}
      </div>
      <div class="top-actions">
        {#if me}<PlayerBadge player={me.player} compact />{/if}
        <button class="soft-button top-sponsor-button" title="关于" onclick={() => (aboutOpen = true)}>
          <Info size={18} /> <span>关于</span>
        </button>
        {#if me}
          <button class="soft-button top-profile-button" title="个人设置" onclick={() => (profileOpen = true)}>
            <UserRound size={18} /> <span>个人设置</span>
          </button>
        {/if}
        {#if me && lobby}
          <button class="soft-button top-leaderboard-button" title="排行榜" onclick={() => (leaderboardOpen = true)}>
            <Crown size={18} /> <span>排行榜</span>
          </button>
        {/if}
      </div>
    </header>
    {#if toastText}
      <div class={`notice toast-notice ${toastLeaving ? "toast-leave" : "toast-enter"}`} role="status">
        {toastText}
      </div>
    {/if}
    {#if announcement}
      <div class="announcement-popup" role="alert">
        <div>
          <b>全服公告</b>
          <p>{announcement.message}</p>
        </div>
        <button class="icon-button" type="button" aria-label="关闭公告" onclick={() => (announcement = null)}>×</button>
      </div>
    {/if}
    {#if remoteBuildId && remoteBuildId !== dismissedRemoteBuildId}
      <div class="announcement-popup version-update-popup" role="alert">
        <div>
          <b>网站内容已更新</b>
          <p>检测到新版本，当前页面可能已经过期，建议刷新以获取最新功能与修复。</p>
        </div>
        <div class="version-update-actions">
          <button class="version-update-button version-update-refresh" type="button" onclick={() => window.location.reload()}>刷新</button>
          <button class="version-update-button version-update-ignore" type="button" onclick={() => (dismissedRemoteBuildId = remoteBuildId)}>忽略</button>
        </div>
      </div>
    {/if}
    {#if view === "login" && restoringSession && !restoreKickPending}
      <section class="panel">正在恢复登录状态...</section>
    {/if}
    {#if view === "login" && restoreKickPending}
      <section class="login-card kick-confirm-card">
        <h2>该账号已在其他设备登录</h2>
        <p class="hint">继续会把另一台设备顶下线（那边会收到提示）。确定要继续吗？</p>
        <div class="kick-confirm-actions">
          <button disabled={restoreKickBusy} onclick={() => { restoreKickPending = null; view = "login"; }}>取消</button>
          <button class="primary" disabled={restoreKickBusy} onclick={confirmRestoreKick}>{restoreKickBusy ? "登录中…" : "确定顶替登录"}</button>
        </div>
      </section>
    {/if}
    {#if view === "login" && !restoringSession && !restoreKickPending}
      <Login
        {config}
        onDone={(next) => {
          if (next.token) localStorage.setItem(tokenKey, next.token);
          me = next;
          trackLoginSuccess("manual");
          initPushForPlayer(next.player.id);
          restoringSession = false;
          room = next.room ? normalizeRoomSnapshot(next.room) : room;
          view = isAdminRoute() ? "admin" : next.room ? "room" : "lobby";
          if (next.room?.phase === "punishment") notice = "已恢复到未完成的惩罚房间。";
        }}
        onError={(message) => (notice = message)}
      />
    {/if}
    {#if view === "lobby" && me && lobby}
      <Lobby
        {config} {lobby} me={me.player} {punishmentTagPrefs}
        onError={(message) => (notice = message)}
        onGoRoom={(nextRoom) => { if (nextRoom) room = nextRoom; view = "room"; }}
        onContribute={() => (view = "contribute")}
        onPunishmentTagPrefsChange={(prefs) => { punishmentTagPrefs = prefs; writePunishmentTagPrefs(prefs); }}
      />
    {/if}
    {#if (view === "contribute" || (view === "admin" && keepContribute)) && me && config}
      <div hidden={view !== "contribute"}>
        <ContributeView {config} me={me.player} onBack={() => { keepContribute = false; view = "lobby"; }} onError={(message) => (notice = message)} ensureSession={() => restoreSession()} />
      </div>
    {/if}
    {#if view === "room" && me && room}
      <Room {config} {room} me={me.player} {lobby} onBack={() => (view = "lobby")} onError={(message) => (notice = message)} />
    {/if}
    {#if view === "admin" && lobby}
      <AdminPanel {config} {lobby} onBack={leaveAdmin} onError={(message) => (notice = message)} />
    {/if}
    {#if view === "room" && !room}
      <section class="panel">你暂时不在房间里。</section>
    {/if}
    {#if aboutOpen}
      <AboutPanel {config} onClose={() => (aboutOpen = false)} onOpenHelp={() => { aboutOpen = false; helpOpen = true; }} />
    {/if}
    {#if helpOpen}
      <HelpPanel onClose={() => (helpOpen = false)} />
    {/if}
    {#if profileOpen && me}
      <ProfilePanel
        {config}
        me={me.player}
        {theme}
        onThemeChange={(next) => { theme = next; trackThemeToggle(next); }}
        onClose={() => (profileOpen = false)}
        onUpdated={(player) => { me = { ...me!, player }; cacheJoinProfile(player); }}
        onError={(message) => (notice = message)}
        onLoggedOut={() => window.location.reload()}
      />
    {/if}
    {#if leaderboardOpen}
      <GlobalLeaderboardPanel players={leaderboardSource} onClose={() => (leaderboardOpen = false)} />
    {/if}
  </main>
{/if}
