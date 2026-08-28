// 会话 / 大厅 / 房间 / 配置的唯一权威来源：所有由服务端主动推送的全局状态（config/
// lobby/room/me/公告/构建版本号）都汇总在这里，连带对应的 socket 事件订阅——与既有
// chatStore.ts「一个模块管一类服务端推送」的架构是同一套思路，只是这里管的是玩家/
// 房间/大厅而不是聊天。业务组件一律直接 import sessionStore 读取，不再像原 React 版
// App.tsx 那样把 config/lobby/room/me 一层层用 props 往下传。
//
// 依赖方向：sessionStore → routerStore/uiStore（单向），避免循环 import；routerStore/
// uiStore 反过来不知道 sessionStore 的存在。
import { socket } from "../../ws";
import type { AppConfig, LobbySnapshot, PublicPlayer, RoomSnapshot } from "../../shared/types";
import type { AnnouncementPayload, MeState } from "../types";
import { leaderboardRefreshMs, playerSecretKey, tokenKey } from "../constants";
import {
  bumpWsAuthRetryCount, cacheJoinProfile, clearPlayerIdentity, connectSocketWithSession, getWsAuthRetryCount, hasCachedLogin,
  joinIdentityPayload, readCachedJoinGender, resetWsAuthRetryCount, sessionTokenLooksValid
} from "../session";
import { ask, isAdminRoute } from "../rpc";
import {
  lobbyOnlineCount, mergeRoundHistory, normalizeConfig, normalizeLobbySnapshot,
  mergeLobbyPlayerIntoFullPlayer, normalizePublicPlayer, normalizeRoomSnapshot, normalizeRoundHistoryItem, playerSyncKey,
  replacePlayersInLobby, replacePlayersInRoom, replacePlayerInRoom
} from "../normalize";
import { refreshActiveChats } from "../chatStore";
import { ensurePushSubscription, fetchPushPreferences } from "../pushNotify";
import { trackLoginSuccess } from "../analytics";
import { routerStore } from "./routerStore.svelte";
import { uiStore } from "./uiStore.svelte";

class SessionStore {
  // 大型 FULL/DELTA 快照：整体替换语义，用 $state.raw 避免深度 Proxy 化整棵树。
  config = $state.raw<AppConfig | null>(null);
  lobby = $state.raw<LobbySnapshot | null>(null);
  room = $state.raw<RoomSnapshot | null>(null);
  me = $state.raw<MeState | null>(null);

  // 有本地登录缓存时先进入恢复态，避免刷新时先闪一下登录页再进大厅。
  restoringSession = $state(!isAdminRoute() && hasCachedLogin());
  connectionState = $state<"connected" | "connecting" | "disconnected">(socket.connected ? "connected" : "connecting");
  restoreKickPending = $state.raw<Record<string, unknown> | null>(null);
  restoreKickBusy = $state(false);

  // 后端构建版本号与本次前端构建不一致时置位；只做提示，不自动刷新（避免误判/多开场景下
  // 把用户的未保存输入或对局强制打断）。dismissedRemoteBuildId 记录用户已关掉提示的版本号，
  // 同一版本重复到达（心跳每 25s 一次）不会重新弹出，只有服务端发布了新版本号才会再次出现。
  remoteBuildId = $state<string | null>(null);
  dismissedRemoteBuildId = $state<string | null>(null);
  announcement = $state.raw<AnnouncementPayload | null>(null);

  // 全站榜单快照：独立 TTL 缓存，不随每次 lobby:update 抖动（见 refreshLeaderboardSnapshot）。
  leaderboardPlayersSnapshot = $state.raw<PublicPlayer[]>([]);

  // 纯可变盒子，不参与渲染，不需要 $state（对应原 React 版的若干 useRef）。
  #restoreInFlight = false;
  #restoreWaiters: Array<(ok: boolean) => void> = [];
  #hadConnected = socket.connected;
  #latestLobbyPlayers: PublicPlayer[] = [];
  #leaderboardSnapshotAt = 0;

  get leaderboardSource(): PublicPlayer[] {
    return this.leaderboardPlayersSnapshot.length ? this.leaderboardPlayersSnapshot : this.lobby?.players || [];
  }

  connect() {
    connectSocketWithSession().catch(() => {
      this.connectionState = "disconnected";
      this.restoringSession = false;
      uiStore.notify("连接失败，请检查网络后刷新重试。");
    });
  }

  #initPushForPlayer(_playerId: string) {
    fetchPushPreferences().catch(() => undefined);
    if (typeof Notification !== "undefined" && Notification.permission === "granted") {
      ensurePushSubscription().catch(() => undefined);
    }
  }

  /** 一次成功的 player:join 结果落地为本地状态的公共部分（token/身份缓存/推送/埋点）。
      导航方式因调用场景而异，由各自调用方在此之后自行决定，不在这里统一处理。 */
  #persistLoginSuccess(next: MeState, source: "manual" | "restore") {
    if (next.token) localStorage.setItem(tokenKey, next.token);
    if (next.reissuedSecret) localStorage.setItem(playerSecretKey, next.reissuedSecret);
    // 以服务端确认后的资料为准回写缓存，避免清洗后的资料与本地不一致。
    if (next.player) cacheJoinProfile(next.player);
    this.me = next;
    trackLoginSuccess(source);
    this.#initPushForPlayer(next.player.id);
  }

  /** 登录页手动登录成功：总是导航（登录页此前不可能停留在房间/共建视图里）。 */
  completeManualLogin(next: MeState) {
    this.#persistLoginSuccess(next, "manual");
    this.restoringSession = false;
    this.room = next.room ? normalizeRoomSnapshot(next.room) : this.room;
    if (isAdminRoute()) return;
    routerStore.goto(next.room ? "room" : "lobby");
    if (next.room?.phase === "punishment") uiStore.notify("已恢复到未完成的惩罚房间。");
  }

  /** 另一台设备顶替登录确认后的结果：语义等价于手动登录成功。 */
  completeRestoreKick(next: MeState) {
    this.completeManualLogin(next);
    this.restoreKickPending = null;
  }

  async confirmRestoreKick() {
    if (!this.restoreKickPending) return;
    this.restoreKickBusy = true;
    try {
      const next = await ask<MeState>("player:join", { ...this.restoreKickPending, forceKick: true });
      this.completeRestoreKick(next);
    } catch (error) {
      uiStore.notify(error instanceof Error ? error.message : "登录失败");
    } finally {
      this.restoreKickBusy = false;
    }
  }

  /** 断线重连 / 刷新页面后的自动登录。与 completeManualLogin 的关键差异：不能打断用户
      正停留的「共建」视图，且要区分「首次连接」与「断线重连后恢复」两种提示文案。 */
  async restoreSession(options: { showRecoveredNotice?: boolean } = {}): Promise<boolean> {
    if (this.#restoreInFlight) {
      return await new Promise<boolean>((resolve) => {
        this.#restoreWaiters.push(resolve);
      });
    }
    const token = localStorage.getItem(tokenKey);
    const cachedName = localStorage.getItem("rps-online-name") || "";
    const { genderId: cachedGender } = readCachedJoinGender();
    if (!cachedName || !sessionTokenLooksValid(token)) {
      this.restoringSession = false;
      return false;
    }
    // 未连上时绝不尝试 join，更不能因此清 token（旧逻辑会把"未连接"误判成坏 token）。
    if (!socket.connected) return false;

    this.#restoreInFlight = true;
    let ok = false;
    const payload = { name: cachedName, genderId: cachedGender, token, ...(await joinIdentityPayload()) };
    try {
      const next = await ask<MeState & { alreadyOnline?: true }>("player:join", payload);
      if (next.alreadyOnline) {
        this.restoreKickPending = payload;
        this.restoringSession = false;
        return false;
      }
      this.#persistLoginSuccess(next, "restore");
      this.room = next.room ? normalizeRoomSnapshot(next.room) : null;
      if (!isAdminRoute()) {
        if (next.room) {
          routerStore.goto("room");
          if (next.room.phase === "punishment") uiStore.notify("已恢复到未完成的惩罚房间。");
        } else if (routerStore.view !== "contribute") {
          routerStore.goto("lobby");
          if (options.showRecoveredNotice) uiStore.notify("连接已恢复，玩家状态已同步。");
        } else if (options.showRecoveredNotice) {
          uiStore.notify("连接已恢复，玩家状态已同步。");
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
        this.me = null;
        this.room = null;
        if (!isAdminRoute()) routerStore.goto("login");
      }
      if (options.showRecoveredNotice) uiStore.notify("连接已恢复，但玩家状态同步失败，请刷新或重新进入。");
    } finally {
      this.#restoreInFlight = false;
      this.restoringSession = false;
      const waiters = this.#restoreWaiters.splice(0);
      for (const wait of waiters) wait(ok);
    }
    return ok;
  }

  #refreshLeaderboardSnapshot(players: PublicPlayer[], now = Date.now()) {
    this.#latestLobbyPlayers = players;
    this.#leaderboardSnapshotAt = now;
    this.leaderboardPlayersSnapshot = players;
  }

  #applyPlayerPatches(rawList: PublicPlayer[]) {
    if (!rawList?.length) return;
    const players = rawList.map(normalizePublicPlayer);
    const byId = new Map(players.map((p) => [p.id, p]));
    // O(P+k) 合并本地排行榜快照源，避免每条 patch 都 some 扫描。
    const prevSnap = this.#latestLobbyPlayers;
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
    this.#latestLobbyPlayers = nextSnap;

    if (this.lobby) this.lobby = replacePlayersInLobby(this.lobby, players);
    if (this.room) this.room = replacePlayersInRoom(this.room, players);
    if (this.me) {
      const p = byId.get(this.me.player.id);
      if (p) {
        // LobbyPlayer/typed PublicPlayer 会省略空串字段：avatarUrl 清空时会变成 undefined。
        // 合并时必须显式写回空值，否则旧头像会残留在 me 上。
        const merged = {
          ...mergeLobbyPlayerIntoFullPlayer(this.me.player, p),
          genderId: typeof p.genderId === "string" ? p.genderId : "",
          avatarUrl: p.avatarUrl || undefined
        };
        this.me = { ...this.me, player: merged, room: this.me.room ? replacePlayerInRoom(this.me.room, merged) : this.me.room };
      }
    }
  }

  /** 实时大厅快照会省略 GameStats；先按精简视图契约合并再比较，否则不仅会把个人设置里的
      分游戏战绩覆盖成 0，保留后还会因比较键不同造成重复写入。收敛依赖 playerSyncKey 的
      幂等比较：合并后不再变化就不再写 me，effect 自然停止重复触发。供 App.svelte 用
      $effect 包一层调用（读 lobby/me 两个字段，天然只在它们变化时重新执行）。 */
  syncMeFromLobby() {
    if (!this.me || !this.lobby) return;
    const latest = this.lobby.players.find((player) => player.id === this.me!.player.id);
    if (!latest) return;
    const nextPlayer = mergeLobbyPlayerIntoFullPlayer(this.me.player, latest);
    if (playerSyncKey(nextPlayer) === playerSyncKey(this.me.player)) return;
    // LobbyPlayer 是精简视图，禁止整对象覆盖 join 时的完整 PublicPlayer（会冲掉冷却等字段）。
    const merged = mergeLobbyPlayerIntoFullPlayer(this.me.player, latest);
    // 头像空串/缺省：batch 路径同样显式写回，避免残留旧 URL。
    if (latest.avatarUrl === undefined || latest.avatarUrl === "") {
      merged.avatarUrl = latest.avatarUrl || "";
    }
    this.me = { ...this.me, player: merged };
  }

  logout() {
    localStorage.removeItem(tokenKey);
    this.me = null;
    this.room = null;
    routerStore.goto("login");
  }

  onlineCount(): number {
    return lobbyOnlineCount(this.lobby);
  }

  /** 订阅全部服务端主动推送事件；返回清理函数，供 App.svelte 用 $effect 包一层挂载一次。 */
  wireSocketHandlers(): () => void {
    socket.on("lobby:update", (nextLobby: LobbySnapshot) => {
      if (nextLobby.config) this.config = normalizeConfig(nextLobby.config);
      const normalized = normalizeLobbySnapshot(nextLobby);
      this.#latestLobbyPlayers = normalized.players;
      const now = Date.now();
      if (!this.#leaderboardSnapshotAt || now - this.#leaderboardSnapshotAt >= leaderboardRefreshMs) {
        this.#refreshLeaderboardSnapshot(normalized.players, now);
      }
      this.lobby = normalizeLobbySnapshot(nextLobby, this.lobby ?? undefined);
    });
    socket.on("room:update", (nextRoom: RoomSnapshot) => {
      const normalized = normalizeRoomSnapshot(nextRoom);
      const old = this.room;
      // updatedAt 缺失/非数字时不要丢弃更新（protobuf 漏字段会导致永远用旧房态）
      const nextAt = Number(normalized.updatedAt) || 0;
      const oldAt = Number(old?.updatedAt) || 0;
      if (old?.id === normalized.id && nextAt > 0 && oldAt > 0 && nextAt < oldAt) {
        // 过期的乱序包，丢弃
      } else if (old?.id === normalized.id) {
        this.room = {
          ...normalized,
          // chat 走 chat:append；全量包若未带 chat 则保留本地
          chat: (normalized.chat || []).length === 0 ? (old.chat || []) : normalized.chat,
          // history 必须合并：惩罚任务/证明写在 latest history，不能因空数组丢弃，也不能整包忽略更新
          roundHistory: mergeRoundHistory(old.roundHistory, normalized.roundHistory)
        };
      } else {
        this.room = normalized;
      }
      if (!isAdminRoute()) routerStore.goto("room");
    });
    socket.on("room:historyAppend", (payload: { roomId?: string; item?: RoomSnapshot["roundHistory"][number]; total?: number }) => {
      const roomId = payload?.roomId;
      const item = payload?.item;
      if (!roomId || !item) return;
      const safeItem = normalizeRoundHistoryItem(item);
      if (this.room?.id === roomId) {
        this.room = {
          ...this.room,
          roundHistory: mergeRoundHistory(this.room.roundHistory, [safeItem]),
          roundHistoryTotal: Number(payload.total) || this.room.roundHistoryTotal
        };
      }
    });
    // 聚合玩家更新（player:batch 为 LobbyPlayer[]；兼容单条 player:update）
    socket.on("player:batch", (list: PublicPlayer[]) => this.#applyPlayerPatches(Array.isArray(list) ? list : []));
    socket.on("player:update", (player: PublicPlayer) => this.#applyPlayerPatches(player ? [player] : []));
    socket.on("player:kicked", () => {
      localStorage.removeItem(tokenKey);
      this.me = null;
      this.room = null;
      routerStore.goto("login");
      uiStore.notify("你已被管理员移出。");
    });
    // 同一账号在另一台设备完成了「顶替登录」确认：本设备的登录状态到此结束，但本地
    // playerId/playerSecret 仍然保留（这台设备依然是受信任设备之一，不是登出）。
    socket.on("session:kicked", ({ message }: { message?: string }) => {
      localStorage.removeItem(tokenKey);
      this.me = null;
      this.room = null;
      routerStore.goto("login");
      uiStore.notify(message || "你的账号已在其他设备登录，此会话已结束，请刷新页面重新登录。");
    });
    socket.on("room:closed", ({ message }: { message?: string }) => {
      this.room = null;
      if (this.me) this.me = { ...this.me, roomId: undefined };
      if (!isAdminRoute()) routerStore.goto("lobby");
      uiStore.notify(message || "房间已被管理员关闭。");
    });
    socket.on("config:update", (nextConfig: AppConfig) => {
      this.config = normalizeConfig(nextConfig);
    });
    // 聊天（房间 + 大厅留言板）已迁到 chatStore：chat:new / chat:deleted 由其内部监听，
    // 首屏与历史走 chat:load / chat:loadOlder，此处不再处理。
    socket.on("announcement:show", (payload: AnnouncementPayload) => {
      this.announcement = payload;
    });
    // server:hello：连接建立时的一次性推送 + 心跳应答兜底（见 ws.ts），带的是后端
    // 当前构建版本号。"dev"/"unknown" 是本地开发或非 git checkout 的兜底字面量，
    // 双方都可能出现，出现即视为不可比较，不触发提示。
    socket.on("server:hello", ({ buildId }: { buildId?: string }) => {
      if (!buildId || buildId === "dev" || buildId === "unknown") return;
      if (__APP_BUILD_ID__ === "dev" || __APP_BUILD_ID__ === "unknown") return;
      if (buildId === __APP_BUILD_ID__) return;
      this.remoteBuildId = buildId;
    });
    socket.on("connect", () => {
      this.connectionState = "connected";
      resetWsAuthRetryCount();
      const isReconnect = this.#hadConnected;
      this.#hadConnected = true;
      // 首次连接不弹「已恢复」；断线重连后才提示。
      // 重连后补拉已激活聊天频道的最近消息（chatStore 不自注册 connect，见其注释）；
      // 必须等 restoreSession 里的 player:join 落地后再发，否则新连接还没绑定玩家/房间，
      // 房间聊天的 chat:load 会被 onChatLoad 以"你不在这个房间里"拒绝，断线期间的消息就永久错过了。
      void this.restoreSession({ showRecoveredNotice: isReconnect }).then(() => refreshActiveChats());
    });
    socket.on("disconnect", () => {
      this.connectionState = "disconnected";
      // iOS 切后台也会触发 disconnect；文案轻一点，避免误以为故障
      uiStore.notify("连接已断开，正在重连…");
    });
    socket.on("connect_error", (error: Error & { data?: { code?: string } }) => {
      this.connectionState = "disconnected";
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
          this.restoringSession = false;
          uiStore.notify("连接认证失败，请刷新页面重试。");
          return;
        }
        bumpWsAuthRetryCount();
        localStorage.removeItem(tokenKey);
        connectSocketWithSession({ forceNewToken: true }).catch(() => {
          this.connectionState = "disconnected";
          this.restoringSession = false;
          uiStore.notify("连接认证失败，请刷新页面重试。");
        });
        return;
      }
      this.restoringSession = false;
    });
    socket.io.on("reconnect_attempt", () => { this.connectionState = "connecting"; });
    // 挂载时机竞态：WS 若在本方法运行前就已连上（例如热更新），必须在这里立刻补一次
    // restoreSession，否则 connect 事件已经错过，永远等不到自动登录。
    if (socket.connected) {
      this.connectionState = "connected";
      void this.restoreSession({ showRecoveredNotice: false });
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
  }

  /** 返回清理函数，供 App.svelte 用 $effect 包一层挂载一次。 */
  startLeaderboardRefreshTimer(): () => void {
    const timer = window.setInterval(() => {
      if (this.#latestLobbyPlayers.length) this.#refreshLeaderboardSnapshot(this.#latestLobbyPlayers);
    }, leaderboardRefreshMs);
    return () => window.clearInterval(timer);
  }
}

export const sessionStore = new SessionStore();
