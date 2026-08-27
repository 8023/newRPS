// 全链路 Protobuf 编解码（无 JSON 文本载荷）。
// 依赖 protobufjs 静态模块：src/gen/proto.js

import { game, wire as wireNs } from "./gen/proto.js";
import { DEFAULT_NAME_WAR_PENALTY_THRESHOLD, DEFAULT_NAME_WAR_RENAME_MIN_POINTS } from "./lib/normalize";

export type PayloadKind = 0 | 1 | 2; // RAW FULL DELTA

export type Envelope = {
  event?: string;
  id?: number;
  err?: string;
  kind?: PayloadKind;
  channel?: string;
  seq?: number;
  hash?: string;
  fullState?: any;
  delta?: any;
  rawBody?: any;
};

const EnvelopeType = wireNs.Envelope;

function jsToStruct(obj: Record<string, unknown>): any {
  const fields: Record<string, any> = {};
  for (const [k, v] of Object.entries(obj)) {
    fields[k] = jsToValue(v);
  }
  return { fields };
}

function jsToValue(v: unknown): any {
  if (v === null || v === undefined) return { nullValue: 0 };
  if (typeof v === "boolean") return { boolValue: v };
  if (typeof v === "number") return { numberValue: v };
  if (typeof v === "string") return { stringValue: v };
  if (Array.isArray(v)) return { listValue: { values: v.map(jsToValue) } };
  if (typeof v === "object") {
    const fields: Record<string, any> = {};
    for (const [k, val] of Object.entries(v as Record<string, unknown>)) {
      fields[k] = jsToValue(val);
    }
    return { structValue: { fields } };
  }
  return { stringValue: String(v) };
}

export function decodeEnvelope(buf: Uint8Array): Envelope {
  const msg = EnvelopeType.decode(buf);
  const o = EnvelopeType.toObject(msg, {
    longs: Number,
    enums: Number,
    bytes: Array,
    defaults: false,
    arrays: true,
    objects: true,
    oneofs: true
  }) as any;
  // DELTA 的 Value 必须从 Message 实例读取：toObject(defaults:false) 会丢掉 number=0
  let delta = o.delta;
  if (msg.delta?.ops?.length) {
    delta = {
      ops: msg.delta.ops.map((op: any) => ({
        path: op.path || "",
        remove: Boolean(op.remove),
        value: op.remove ? undefined : valueMsgToPlain(op.value)
      }))
    };
  }
  return {
    event: o.event,
    id: o.id != null ? Number(o.id) : undefined,
    err: o.err,
    kind: o.kind as PayloadKind,
    channel: o.channel,
    seq: o.seq != null ? Number(o.seq) : undefined,
    hash: o.hash,
    fullState: o.fullState,
    delta,
    rawBody: o.rawBody
  };
}

/** 从 protobufjs Value Message 转 plain（保留 0 / false / ""） */
function valueMsgToPlain(v: any): any {
  if (v == null) return null;
  const kind = typeof v.kind === "string" ? v.kind : null;
  if (kind === "nullValue") return null;
  if (kind === "numberValue") {
    const n = Number(v.numberValue);
    return Number.isFinite(n) ? n : 0;
  }
  if (kind === "stringValue") return v.stringValue ?? "";
  if (kind === "boolValue") return Boolean(v.boolValue);
  if (kind === "structValue") return structMsgToPlain(v.structValue);
  if (kind === "listValue") {
    const values = v.listValue?.values || [];
    return values.map((x: any) => valueMsgToPlain(x));
  }
  // 无 oneof 信息时兜底
  if (v.structValue) return structMsgToPlain(v.structValue);
  if (v.listValue) return (v.listValue.values || []).map((x: any) => valueMsgToPlain(x));
  if (typeof v.numberValue === "number") return v.numberValue;
  if (typeof v.stringValue === "string") return v.stringValue;
  if (typeof v.boolValue === "boolean") return v.boolValue;
  if (v.nullValue != null) return null;
  return null;
}

function structMsgToPlain(s: any): Record<string, any> {
  if (!s) return {};
  const fields = s.fields || {};
  const out: Record<string, any> = {};
  // protobufjs map 可能是对象或 Map
  const entries = typeof fields.forEach === "function"
    ? (() => {
        const e: Array<[string, any]> = [];
        fields.forEach((val: any, key: string) => e.push([key, val]));
        return e;
      })()
    : Object.entries(fields);
  for (const [k, val] of entries) {
    out[k] = valueMsgToPlain(val);
  }
  return out;
}

export function encodeEnvelope(env: Envelope): Uint8Array {
  const payload: any = {
    event: env.event || "",
    id: env.id || 0,
    err: env.err || "",
    kind: env.kind ?? 0,
    channel: env.channel || "",
    seq: env.seq || 0,
    hash: env.hash || ""
  };
  if (env.fullState) payload.fullState = env.fullState;
  if (env.delta) payload.delta = env.delta;
  if (env.rawBody) payload.rawBody = env.rawBody;
  const msg = EnvelopeType.create(payload);
  return EnvelopeType.encode(msg).finish();
}

/** RPC 请求：plain object → RawBody.dynamic */
export function encodeRawBodyDynamic(data: unknown): any {
  const obj = (data && typeof data === "object" && !Array.isArray(data)
    ? data
    : { value: data }) as Record<string, unknown>;
  return { dynamic: jsToStruct(obj) };
}

/** RAW 响应/推送 → plain JS */
export function rawBodyToPlain(rawBody: any): any {
  if (!rawBody) return {};
  if (rawBody.dynamic) return structToJs(rawBody.dynamic);
  if (rawBody.playerBatch) {
    return (rawBody.playerBatch.players || []).map((p: any) => materializePlayer(p));
  }
  if (rawBody.chat) return materializeChat(rawBody.chat);
  if (rawBody.suggestion) return materializeSuggestion(rawBody.suggestion);
  if (rawBody.player) return materializePlayer(rawBody.player);
  if (rawBody.me) {
    const me = { ...rawBody.me };
    if (me.room) me.room = materializeRoom(me.room);
    if (me.player) me.player = materializePlayer(me.player);
    return me;
  }
  if (rawBody.announcement) return { ...rawBody.announcement };
  if (rawBody.roomClosed) return { ...rawBody.roomClosed };
  if (rawBody.historyPage) return { ...rawBody.historyPage, item: fillRoundHistoryItemDefaults(rawBody.historyPage.item) };
  if (rawBody.ok) return { ok: !!rawBody.ok.ok };
  if (rawBody.playerResult) return { player: materializePlayer(rawBody.playerResult.player) };
  if (rawBody.suggestions) {
    const s = { ...rawBody.suggestions };
    if (Array.isArray(s?.items)) s.items = s.items.map(materializeSuggestion);
    else if (Array.isArray(s)) return s.map(materializeSuggestion);
    return s;
  }
  if (rawBody.room) return materializeRoom(rawBody.room);
  if (rawBody.config) return materializeConfig(rawBody.config);
  if (rawBody.lobby) return materializeLobby(rawBody.lobby);
  return {};
}

function structToJs(s: any): any {
  if (!s) return {};
  const fields = s.fields || {};
  const out: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(fields)) {
    out[k] = valueToJs(v);
  }
  return out;
}

/** protobufjs toObject 后的 google.protobuf.Value → 普通 JS（含 0 / false / ""） */
export function protoValueToPlain(v: any): any {
  return valueToJs(v);
}

function valueToJs(v: any): any {
  if (v == null) return null;
  // oneof kind 字符串形式
  if (v.kind === "nullValue") return null;
  if (v.kind === "numberValue") return typeof v.numberValue === "number" ? v.numberValue : 0;
  if (v.kind === "stringValue") return v.stringValue ?? "";
  if (v.kind === "boolValue") return Boolean(v.boolValue);
  if (v.kind === "structValue") return structToJs(v.structValue);
  if (v.kind === "listValue") return (v.listValue?.values || []).map(valueToJs);
  // 无 kind 时按字段探测（defaults:false 可能省略 0）
  if ("numberValue" in v && typeof v.numberValue === "number") return v.numberValue;
  if ("numberValue" in v && v.numberValue == null && !("stringValue" in v) && !("boolValue" in v) && !("structValue" in v) && !("listValue" in v) && !("nullValue" in v)) {
    return 0; // 被省略的 0 不应走到这里；保守
  }
  if ("stringValue" in v && typeof v.stringValue === "string") return v.stringValue;
  if ("boolValue" in v && typeof v.boolValue === "boolean") return v.boolValue;
  if (v.structValue) return structToJs(v.structValue);
  if (v.listValue) return (v.listValue.values || []).map(valueToJs);
  if ("nullValue" in v) return null;
  return null;
}

/** DELTA 合并后轻量归一，与服务端 NormalizeFrontTree 对齐关键坐标 0 */
export function normalizeStateTree(doc: any): any {
  if (!doc || typeof doc !== "object") return doc;
  const out = doc;
  // 大厅房间人数是合法的 0，但 protobuf defaults:false 会省略零值。DELTA 在
  // 1 -> 0 时也可能删除该 key；每次合并后补齐，保证通道缓存、CRC 和 UI 一致。
  if (Array.isArray(out.rooms)) {
    for (const room of out.rooms) {
      if (!room || typeof room !== "object") continue;
      room.players = numOr(room.players, 0);
      room.spectators = numOr(room.spectators, 0);
    }
  }
  if (out.othello && typeof out.othello === "object") {
    out.othello.board = padBoardMatrix(boardRows(out.othello.board), 8);
    out.othello.legalMoves = normalizePosList(out.othello.legalMoves);
    out.othello.blackCount = numOr(out.othello.blackCount, 0);
    out.othello.whiteCount = numOr(out.othello.whiteCount, 0);
    out.othello.passCount = numOr(out.othello.passCount, 0);
    out.othello.moveDeadlineAt = numOr(out.othello.moveDeadlineAt, 0);
    out.othello.clockDeadlineAt = numOr(out.othello.clockDeadlineAt, 0);
    if (!Array.isArray(out.othello.settlementEvents)) out.othello.settlementEvents = [];
  }
  if (out.tictactoe && typeof out.tictactoe === "object") {
    out.tictactoe.board = padBoardMatrix(boardRows(out.tictactoe.board), 3);
    out.tictactoe.winningLine = normalizePosList(out.tictactoe.winningLine);
    out.tictactoe.moveCount = numOr(out.tictactoe.moveCount, 0);
  }
  if (out.gomoku && typeof out.gomoku === "object") {
    out.gomoku.board = padBoardMatrix(boardRows(out.gomoku.board), 15);
    out.gomoku.moves = normalizePosList(out.gomoku.moves);
    out.gomoku.winningLine = normalizePosList(out.gomoku.winningLine);
    out.gomoku.moveCount = numOr(out.gomoku.moveCount, 0);
    out.gomoku.moveDeadlineAt = numOr(out.gomoku.moveDeadlineAt, 0);
    out.gomoku.clockDeadlineAt = numOr(out.gomoku.clockDeadlineAt, 0);
  }
  if (out.jungle && typeof out.jungle === "object") {
    out.jungle.board = padBoardRect(boardRows(out.jungle.board), 9, 7);
    out.jungle.moveCount = numOr(out.jungle.moveCount, 0);
    out.jungle.moveDeadlineAt = numOr(out.jungle.moveDeadlineAt, 0);
    out.jungle.clockDeadlineAt = numOr(out.jungle.clockDeadlineAt, 0);
    if (out.jungle.lastFrom) out.jungle.lastFrom = { row: numOr(out.jungle.lastFrom.row, 0), col: numOr(out.jungle.lastFrom.col, 0) };
    if (out.jungle.lastTo) out.jungle.lastTo = { row: numOr(out.jungle.lastTo.row, 0), col: numOr(out.jungle.lastTo.col, 0) };
  }
  if (out.chess && typeof out.chess === "object") {
    materializeChessTree(out.chess);
  }
  // protobufjs defaults:false 会丢掉数值零值；这些字段会在广播窗口清空时回落到 0，
  // 若不补齐，管理面板会在 DELTA 把 key 整个删掉后显示 undefined/NaN。
  if (out.serverStats && typeof out.serverStats === "object") {
    const s = out.serverStats;
    s.startedAt = numOr(s.startedAt, 0);
    s.roomBroadcasts = numOr(s.roomBroadcasts, 0);
    s.lobbyBroadcasts = numOr(s.lobbyBroadcasts, 0);
    s.disconnects = numOr(s.disconnects, 0);
    s.reconnects = numOr(s.reconnects, 0);
    s.lastRoomSnapshotBytes = numOr(s.lastRoomSnapshotBytes, 0);
    s.lastLobbySnapshotBytes = numOr(s.lastLobbySnapshotBytes, 0);
    s.recentRoomBroadcasts = numOr(s.recentRoomBroadcasts, 0);
    s.recentLobbyBroadcasts = numOr(s.recentLobbyBroadcasts, 0);
    s.averageRoomSnapshotBytes = numOr(s.averageRoomSnapshotBytes, 0);
    s.averageLobbySnapshotBytes = numOr(s.averageLobbySnapshotBytes, 0);
  }
  for (const k of ["spectators", "proofs", "roundHistory", "chat", "punishedPlayerIds", "players", "rooms", "suggestions", "lobbyChat"]) {
    if (k in out && out[k] == null) out[k] = [];
  }
  if ("roundHistoryTotal" in out) out.roundHistoryTotal = numOr(out.roundHistoryTotal, 0);
  if (Array.isArray(out.roundHistory)) {
    out.roundHistory = out.roundHistory.map((item: any) => ({
      ...item,
      tictactoeLine: normalizePosList(item?.tictactoeLine),
      gomokuLine: normalizePosList(item?.gomokuLine),
      punishmentTasks: item?.punishmentTasks || [],
      punishedNames: item?.punishedNames || [],
      proofs: item?.proofs || []
    }));
  }
  return out;
}

/**
 * 玩家（PublicPlayer/LobbyPlayer）里对应 Go *T 指针字段的名字。
 * 这些字段在 Go domain 里允许真正的"从未设置"（nil），但前端从不区分
 * "从未设置" 和 "设置为 false/0"（全部只做真值判断），所以这里统一按
 * 该字段的零值补齐即可，见 fillPlayerDefaults。
 */
const PLAYER_BOOL_FIELDS = [
  "nameWarEnabled", "nameWarPunished", "nameWarAllowRename",
  "giveawayEnabled", "rankMultiplierUnlocked", "extremeModeEnabled",
  "extremeForceClosed", "isAdmin",
  "bondMasterEnabled", "bondPetEnabled", "bondPublicDisplay"
];
const PLAYER_NUM_FIELDS = [
  "disconnectedAt", "disconnectExpiresAt", "profileUpdatedAt",
  "nameWarToggledAt", "nameWarRenameProtectedUntil", "nameWarRenameWindowStartedAt", "nameWarRenameCount",
  "giveawayValue", "giveawayClicks", "giveawayBoardSubmittedAt", "giveawayBoardExpiresAt",
  "giveawayBoardLikes", "giveawayBoardDislikes", "giveawayBoardLikesThisHour", "giveawayBoardLikeWindowStartedAt",
  "giveawayVoteWindowStartedAt", "giveawayVoteCount", "giveawayVoteLikesThisHour", "giveawayVoteDislikesThisHour",
  "extremeModeToggledAt", "extremeModeCooldownUntil", "extremeWinStreak", "extremeLastDecayHour",
  "extremeForceClosedAt", "extremeRenameProtectedUntil"
];

function fillRoomSettingsDefaults(settings: any): any {
  if (!settings || typeof settings !== "object") return settings;
  // allowProofImage 对应 Go *bool，但 handlers_room.go 建房时已把 nil 归一化成具体的
  // true/false，线上房间永远不会是"未设置"——false 是它的零值，缺省按 false 补齐即可
  // 无损还原（true 是非零值，永远不会被 protojson 丢掉）。
  // 四种棋类的 UndoLimit 同理：建房时已校验为 0/1/3/10 之一，键缺失只可能是真值 0（禁止悔棋）
  // 被 defaults:false 连 key 一起丢掉，缺省按 0 补齐即可无损还原。计时设置四个字段同理，
  // 0 就是"不限时"的合法值。
  let source = settings.punishmentSource || "random";
  if (source === "system") source = "random";
  return {
    ...settings,
    punishmentSource: source,
    punishmentIds: Array.isArray(settings.punishmentIds) ? settings.punishmentIds : [],
    punishmentTagsIncluded: Array.isArray(settings.punishmentTagsIncluded) ? settings.punishmentTagsIncluded : [],
    punishmentTagsExcluded: Array.isArray(settings.punishmentTagsExcluded) ? settings.punishmentTagsExcluded : [],
    punishmentSeriesId: typeof settings.punishmentSeriesId === "string" ? settings.punishmentSeriesId : "",
    allowProofImage: settings.allowProofImage ?? false,
    enablePerPiecePunishment: settings.enablePerPiecePunishment ?? false,
    gomokuUndoLimit: numOr(settings.gomokuUndoLimit, 0),
    jungleUndoLimit: numOr(settings.jungleUndoLimit, 0),
    chessUndoLimit: numOr(settings.chessUndoLimit, 0),
    othelloUndoLimit: numOr(settings.othelloUndoLimit, 0),
    othelloMoveSeconds: numOr(settings.othelloMoveSeconds, 0),
    othelloGameMinutes: numOr(settings.othelloGameMinutes, 0),
    gomokuMoveSeconds: numOr(settings.gomokuMoveSeconds, 0),
    gomokuGameMinutes: numOr(settings.gomokuGameMinutes, 0),
    jungleMoveSeconds: numOr(settings.jungleMoveSeconds, 0),
    jungleGameMinutes: numOr(settings.jungleGameMinutes, 0),
    chessMoveSeconds: numOr(settings.chessMoveSeconds, 0),
    chessGameMinutes: numOr(settings.chessGameMinutes, 0)
  };
}

function fillReviewedAtDefault(proof: any): any {
  if (!proof || typeof proof !== "object") return proof;
  return { ...proof, reviewedAt: numOr(proof.reviewedAt, 0) };
}

function fillBackgroundOpacityDefault(task: any): any {
  if (!task || typeof task !== "object") return task;
  return { ...task, backgroundOpacity: numOr(task.backgroundOpacity, 0) };
}

export function stateDocToPlain(fullState: any): any {
  if (!fullState) return null;
  if (fullState.lobby || fullState.doc === "lobby") return materializeLobby(fullState.lobby);
  if (fullState.room || fullState.doc === "room") return materializeRoom(fullState.room);
  if (fullState.config || fullState.doc === "config") return materializeConfig(fullState.config);
  return fullState;
}

/** protobufjs defaults:false 会丢掉 int/bool 零值；玩家战绩必须补齐，否则出现 undefined/NaN 文案。
 * genderId 空串同样会被 defaults:false 丢掉——必须补成 ""，否则 {...old, ...patch} 合并
 * 会残留旧的预设 id（与 Go fillPlayerDefaults 对齐）。 */
function materializePlayer(player: any): any {
  if (!player || typeof player !== "object") return player;
  const p = { ...player };
  for (const k of PLAYER_BOOL_FIELDS) if (p[k] === undefined) p[k] = false;
  for (const k of PLAYER_NUM_FIELDS) if (p[k] === undefined) p[k] = 0;
  if (typeof p.genderId !== "string") p.genderId = "";
  p.stats = materializePublicStats(p.stats);
  p.gameStats = materializeGameStats(p.gameStats);
  return p;
}

function materializePublicStats(stats: any): any {
  const s = stats && typeof stats === "object" ? stats : {};
  const title = typeof s.title === "string" ? s.title.trim() : "";
  return {
    wins: numOr(s.wins, 0),
    losses: numOr(s.losses, 0),
    draws: numOr(s.draws, 0),
    punishments: numOr(s.punishments, 0),
    rankedPoints: numOr(s.rankedPoints, 0),
    highestScore: numOr(s.highestScore, 0),
    lowestScore: numOr(s.lowestScore, 0),
    // sort* 缺省回退到展示值，避免旧包没有这些字段时排序崩溃。
    sortRankedPoints: numOr(s.sortRankedPoints, numOr(s.rankedPoints, 0)),
    sortHighestScore: numOr(s.sortHighestScore, numOr(s.highestScore, 0)),
    sortLowestScore: numOr(s.sortLowestScore, numOr(s.lowestScore, 0)),
    title: title || "暂无称号",
    selfTitle: typeof s.selfTitle === "string" ? s.selfTitle : "",
    titleSource: (typeof s.titleSource === "string" && s.titleSource) || "system",
    titleColors: s.titleColors && typeof s.titleColors === "object" ? s.titleColors : null,
    totalOnlineMs: numOr(s.totalOnlineMs, 0),
    ...(s.titleSegmentId != null && s.titleSegmentId !== ""
      ? { titleSegmentId: s.titleSegmentId }
      : {}),
    ...(s.titleCustom ? { titleCustom: true } : {})
  };
}

function materializeGameWLD(stats: any): any {
  const s = stats && typeof stats === "object" ? stats : {};
  return { wins: numOr(s.wins, 0), losses: numOr(s.losses, 0), draws: numOr(s.draws, 0) };
}

function materializeGameStats(stats: any): any {
  const s = stats && typeof stats === "object" ? stats : {};
  return {
    rps: materializeGameWLD(s.rps),
    othello: materializeGameWLD(s.othello),
    tictactoe: materializeGameWLD(s.tictactoe),
    gomoku: materializeGameWLD(s.gomoku),
    liarsdice: materializeGameWLD(s.liarsdice),
    jungle: materializeGameWLD(s.jungle),
    chess: materializeGameWLD(s.chess)
  };
}

function materializeSeatStats(stats: any): any {
  const s = stats && typeof stats === "object" ? stats : {};
  return {
    wins: numOr(s.wins, 0),
    losses: numOr(s.losses, 0),
    draws: numOr(s.draws, 0),
    punishments: numOr(s.punishments, 0)
  };
}

function materializeChat(chat: any): any {
  if (!chat) return chat;
  return {
    ...chat,
    expiresAt: numOr(chat.expiresAt, 0),
    seq: numOr(chat.seq, 0),
    // defaults:false 下空 repeated 可能缺省；@ 高亮依赖数组存在。
    mentions: Array.isArray(chat.mentions) ? chat.mentions : []
  };
}

function materializeSuggestion(item: any): any {
  if (!item) return item;
  const s = { ...item };
  if (s.authorPlayer) s.authorPlayer = materializePlayer(s.authorPlayer);
  return s;
}

function materializeLobby(lobby: any): any {
  if (!lobby) return null;
  const L = { ...lobby };
  const players = (L.players || []).map((e: any) => materializePlayer(e.player || e)).filter(Boolean);
  const rooms = (L.rooms || []).map((e: any) => materializeLobbyRoom(e.room || e)).filter(Boolean);
  if (L.config) L.config = materializeConfig(L.config);
  if (!Array.isArray(L.petBonds)) L.petBonds = [];
  return {
    ...L,
    players,
    rooms,
    normalLeaderboard: (L.normalLeaderboard || []).map(materializePlayer),
    rankedLeaderboard: (L.rankedLeaderboard || []).map(materializePlayer),
    suggestions: (L.suggestions || []).map(materializeSuggestion),
    lobbyChat: (L.lobbyChat || []).map(materializeChat),
    onlineCount: L.onlineCount || players.filter((p: any) => p.connected).length
  };
}

/** protobufjs defaults:false 会丢掉 players/spectators=0，导致大话骰空房显示"undefined 人参战" */
function materializeLobbyRoom(room: any): any {
  if (!room) return null;
  const r = { ...room };
  r.players = numOr(r.players, 0);
  r.spectators = numOr(r.spectators, 0);
  r.punishmentSource = r.punishmentSource === "system" ? "random" : (r.punishmentSource || "random");
  r.punishmentTagsIncluded = Array.isArray(r.punishmentTagsIncluded) ? r.punishmentTagsIncluded : [];
  r.punishmentTagsExcluded = Array.isArray(r.punishmentTagsExcluded) ? r.punishmentTagsExcluded : [];
  r.punishmentSeriesId = typeof r.punishmentSeriesId === "string" ? r.punishmentSeriesId : "";
  const versus: any = { A: null, B: null };
  for (const item of r.versus || []) {
    const key = item.key;
    const val = item.value || {};
    if (val.player) versus[key] = { player: materializePlayer(val.player) };
  }
  r.versus = versus;
  return r;
}

function materializeRoom(room: any): any {
  if (!room) return null;
  const r = { ...room };
  r.settings = fillRoomSettingsDefaults(r.settings);
  r.seats = pairsToOccupantMap(r.seats);
  r.ready = pairsToMap(r.ready, false);
  r.choices = pairsToMap(r.choices, undefined);
  r.revealedChoices = pairsToMap(r.revealedChoices, undefined);
  r.score = pairsToMap(r.score, 0);
  r.seatedScore = pairsToMap(r.seatedScore, 0);
  const rawSeatStats = pairsToMap(r.seatStats, { wins: 0, losses: 0, draws: 0, punishments: 0 });
  r.seatStats = {
    A: materializeSeatStats(rawSeatStats.A),
    B: materializeSeatStats(rawSeatStats.B)
  };
  if (r.othello) {
    // protobufjs toObject(defaults:false) 会丢掉 int32 的 0，导致 row/col=0 的合法手失效
    r.othello.board = padBoardMatrix(boardRows(r.othello.board), 8);
    r.othello.legalMoves = normalizePosList(r.othello.legalMoves);
    r.othello.rankedDelta = pairsToMap(r.othello.rankedDelta, 0);
    r.othello.undoCount = pairsToMap(r.othello.undoCount, 0);
    r.othello.clockRemaining = pairsToMap(r.othello.clockRemaining, 0);
    r.othello.blackCount = numOr(r.othello.blackCount, 0);
    r.othello.whiteCount = numOr(r.othello.whiteCount, 0);
    r.othello.passCount = numOr(r.othello.passCount, 0);
    r.othello.moveDeadlineAt = numOr(r.othello.moveDeadlineAt, 0);
    r.othello.clockDeadlineAt = numOr(r.othello.clockDeadlineAt, 0);
    if (r.othello.pendingSettlement) {
      r.othello.pendingSettlement.flips = numOr(r.othello.pendingSettlement.flips, 0);
      r.othello.pendingSettlement.stake = numOr(r.othello.pendingSettlement.stake, 0);
    }
  }
  if (r.tictactoe) {
    r.tictactoe.board = padBoardMatrix(boardRows(r.tictactoe.board), 3);
    r.tictactoe.winningLine = normalizePosList(r.tictactoe.winningLine);
    r.tictactoe.rankedDelta = pairsToMap(r.tictactoe.rankedDelta, 0);
    r.tictactoe.moveCount = numOr(r.tictactoe.moveCount, 0);
  }
  if (r.liarsDice) {
    r.liarsDice = materializeLiarsDice(r.liarsDice);
  }
  if (r.gomoku) {
    r.gomoku.board = padBoardMatrix(boardRows(r.gomoku.board), 15);
    r.gomoku.moves = normalizePosList(r.gomoku.moves);
    r.gomoku.winningLine = normalizePosList(r.gomoku.winningLine);
    r.gomoku.rankedDelta = pairsToMap(r.gomoku.rankedDelta, 0);
    r.gomoku.undoCount = pairsToMap(r.gomoku.undoCount, 0);
    r.gomoku.clockRemaining = pairsToMap(r.gomoku.clockRemaining, 0);
    r.gomoku.moveCount = numOr(r.gomoku.moveCount, 0);
    r.gomoku.moveDeadlineAt = numOr(r.gomoku.moveDeadlineAt, 0);
    r.gomoku.clockDeadlineAt = numOr(r.gomoku.clockDeadlineAt, 0);
  }
  if (r.jungle) {
    r.jungle.board = padBoardRect(boardRows(r.jungle.board), 9, 7);
    r.jungle.rankedDelta = pairsToMap(r.jungle.rankedDelta, 0);
    r.jungle.undoCount = pairsToMap(r.jungle.undoCount, 0);
    r.jungle.clockRemaining = pairsToMap(r.jungle.clockRemaining, 0);
    r.jungle.moveCount = numOr(r.jungle.moveCount, 0);
    r.jungle.moveDeadlineAt = numOr(r.jungle.moveDeadlineAt, 0);
    r.jungle.clockDeadlineAt = numOr(r.jungle.clockDeadlineAt, 0);
    if (r.jungle.lastFrom) r.jungle.lastFrom = { row: numOr(r.jungle.lastFrom.row, 0), col: numOr(r.jungle.lastFrom.col, 0) };
    if (r.jungle.lastTo) r.jungle.lastTo = { row: numOr(r.jungle.lastTo.row, 0), col: numOr(r.jungle.lastTo.col, 0) };
  }
  if (r.chess) {
    r.chess.board = padBoardMatrix(boardRows(r.chess.board), 8);
    r.chess.rankedDelta = pairsToMap(r.chess.rankedDelta, 0);
    r.chess.undoCount = pairsToMap(r.chess.undoCount, 0);
    r.chess.clockRemaining = pairsToMap(r.chess.clockRemaining, 0);
    materializeChessTree(r.chess);
  }
  if (r.coinFlip) {
    r.coinFlip.correct = Boolean(r.coinFlip.correct);
    r.coinFlip.settledAt = numOr(r.coinFlip.settledAt, 0);
  }
  // 历史里的井字连线/大话骰开牌数据同样可能丢 0 / 需要 pair 展开
  if (Array.isArray(r.roundHistory)) {
    r.roundHistory = r.roundHistory.map(fillRoundHistoryItemDefaults);
  }
  r.spectators = (r.spectators || []).map(materializePlayer);
  r.proofs = (r.proofs || []).map(fillReviewedAtDefault);
  r.roundHistory = r.roundHistory || [];
  r.chat = (r.chat || []).map(materializeChat);
  r.punishedPlayerIds = r.punishedPlayerIds || [];
  return r;
}

/** RoundHistoryItem 里 stake/rankMultiplier/effectiveStake 对应 Go *int 指针，缺省即"未评级"，统一按 0 补齐 */
function fillRoundHistoryItemDefaults(item: any): any {
  if (!item || typeof item !== "object") return item;
  return {
    ...item,
    stake: numOr(item.stake, 0),
    rankMultiplier: numOr(item.rankMultiplier, 0),
    effectiveStake: numOr(item.effectiveStake, 0),
    tictactoeLine: normalizePosList(item.tictactoeLine),
    gomokuLine: normalizePosList(item.gomokuLine),
    liarsDiceBidCount: numOr(item.liarsDiceBidCount, 0),
    liarsDiceBidFace: numOr(item.liarsDiceBidFace, 0),
    liarsDiceActualCount: numOr(item.liarsDiceActualCount, 0),
    liarsDiceHands: pairsToGenericMap(item.liarsDiceHands, (v: any) =>
      Array.isArray(v?.values) ? v.values.map((x: any) => numOr(x, 0)) : []
    ),
    liarsDiceNames: pairsToGenericMap(item.liarsDiceNames),
    proofs: Array.isArray(item.proofs) ? item.proofs.map(fillReviewedAtDefault) : item.proofs,
    punishmentTasks: Array.isArray(item.punishmentTasks) ? item.punishmentTasks.map(fillBackgroundOpacityDefault) : item.punishmentTasks
  };
}

function materializeConfig(cfg: any): any {
  if (!cfg) return null;
  const c = { ...cfg };
  // protobufjs 把 max_creates_per_10_min 编成 maxCreatesPer_10Min，
  // 与业务/Go JSON 的 maxCreatesPer10Min 不一致 → 后台显示 undefined、保存后被推送覆盖回空。
  if (c.accessControl && typeof c.accessControl === "object") {
    const ac = c.accessControl;
    if (ac.maxCreatesPer10Min == null && ac.maxCreatesPer_10Min != null) {
      ac.maxCreatesPer10Min = Number(ac.maxCreatesPer_10Min);
    }
    if ("maxCreatesPer_10Min" in ac) delete ac.maxCreatesPer_10Min;
    if (ac.maxOnlinePerIp != null) ac.maxOnlinePerIp = Number(ac.maxOnlinePerIp);
    if (ac.maxCreatesPer10Min != null) ac.maxCreatesPer10Min = Number(ac.maxCreatesPer10Min);
    for (const key of ["ipBackstopMultiplier", "ipBackstopMinLimit", "maxSessionIssuePerIp", "maxOnlinePerIpTotal", "maxCreatesPerIp", "maxActiveRoomsPerOwner", "maxProofUploadsPerPlayer"]) {
      if (ac[key] != null) ac[key] = Number(ac[key]);
    }
  }
  if (Array.isArray(c.roomInfoTags)) {
    const obj: any = {};
    for (const item of c.roomInfoTags) {
      if (item?.key) obj[item.key] = item.style || item;
    }
    c.roomInfoTags = obj;
  }
  if (Array.isArray(c.titleTagStyles)) {
    const obj: any = {};
    for (const item of c.titleTagStyles) {
      if (item?.key) obj[item.key] = item.style || item;
    }
    c.titleTagStyles = obj;
  }
  if (Array.isArray(c.messages)) {
    const obj: any = {};
    for (const item of c.messages) {
      if (item?.key != null) obj[item.key] = item.value;
    }
    c.messages = obj;
  }
  if (c.extremeMode) {
    for (const field of ["positiveLossRates", "negativeWinRates", "hourlyDecay"]) {
      if (Array.isArray(c.extremeMode[field])) {
        const obj: any = {};
        for (const item of c.extremeMode[field]) {
          // value 合法取值含 0（如某难度掉分率为 0%），defaults:false 会把它丢成 undefined，
          // 补回 0 否则后台数字框显示空白（跟 Go 侧 fixConfigMaps 保持一致）。
          if (item?.key != null) obj[item.key] = item.value ?? 0;
        }
        c.extremeMode[field] = obj;
      }
    }
  }
  if (Array.isArray(c.titles)) {
    for (const t of c.titles) {
      if (t.minPercent != null) t.minPercent = Number(t.minPercent);
      if (t.maxPercent != null) t.maxPercent = Number(t.maxPercent);
      // 兼容旧配置字段名（绝对分）——已废弃，仅防后台空白。
      if (t.minPercent == null && t.min != null) t.minPercent = Number(t.min);
      if (t.maxPercent == null && t.max != null) t.maxPercent = Number(t.max);
      if (Array.isArray(t.factionNames)) {
        const obj: any = {};
        for (const item of t.factionNames) {
          if (item?.factionId) obj[item.factionId] = item.names;
        }
        t.factionNames = obj;
      }
    }
  }
  // rankedScore / nameWar.penaltyThreshold / giveaway 数值：proto 漏嵌套消息或 defaults:false 丢字段时，
  // 在 wire 层先补齐；业务层 normalizeConfig 还会再兜一层（与 extremeMode 同模式）。
  // 负分必须用 Number.isFinite，不能 ||（0 合法时也不该误替换，这里 min 虽为负但保持一致）。
  {
    const rs = c.rankedScore && typeof c.rankedScore === "object" ? c.rankedScore : {};
    c.rankedScore = {
      max: numOr(rs.max, 4999),
      min: numOr(rs.min, -4999),
      nameWarMin: numOr(rs.nameWarMin, -9999),
      dailyDecayRatio: numOr(rs.dailyDecayRatio, 0.98)
    };
  }
  // giveaway 数值字段：历史上曾只写在 Go/TS 类型里、未进 GiveawayConfig proto，
  // 旧 wire 会整段丢失；0 在业务上非法（Validate 拒绝），按正数回退默认。
  {
    const g = c.giveaway && typeof c.giveaway === "object" ? c.giveaway : {};
    const pos = (v: any, fallback: number) => {
      const n = Number(v);
      return Number.isFinite(n) && n > 0 ? n : fallback;
    };
    c.giveaway = {
      ...g,
      activeBoostValue: pos(g.activeBoostValue, 2),
      winPenaltyValue: pos(g.winPenaltyValue, 1),
      likeVoteLimitPerHour: pos(g.likeVoteLimitPerHour, 3),
      likeVoteValue: pos(g.likeVoteValue, 1),
      dislikeVoteLimitPerHour: pos(g.dislikeVoteLimitPerHour, 10),
      dislikeVoteValue: pos(g.dislikeVoteValue, 0.1),
      petLikeVoteLimitPerHour: pos(g.petLikeVoteLimitPerHour, 5),
      petLikeVoteValue: pos(g.petLikeVoteValue, 1),
      petDislikeVoteLimitPerHour: pos(g.petDislikeVoteLimitPerHour, 15),
      petDislikeVoteValue: pos(g.petDislikeVoteValue, 0.1),
      masterLikeVoteLimitPerHour: pos(g.masterLikeVoteLimitPerHour, 5),
      masterLikeVoteValue: pos(g.masterLikeVoteValue, 1),
      masterDislikeVoteLimitPerHour: pos(g.masterDislikeVoteLimitPerHour, 15),
      masterDislikeVoteValue: pos(g.masterDislikeVoteValue, 0.1)
    };
  }
  if (!c.nameWar || typeof c.nameWar !== "object") c.nameWar = {};
  c.nameWar.penaltyThreshold = numOr(c.nameWar.penaltyThreshold, DEFAULT_NAME_WAR_PENALTY_THRESHOLD);
  c.nameWar.renameMinPoints = numOr(c.nameWar.renameMinPoints, DEFAULT_NAME_WAR_RENAME_MIN_POINTS);
  {
    const pb = c.petBond && typeof c.petBond === "object" ? c.petBond : {};
    c.petBond = {
      panelTitle: typeof pb.panelTitle === "string" && pb.panelTitle.trim() ? pb.panelTitle : "宠物乐园",
      maxPetsPerMaster: numOr(pb.maxPetsPerMaster, 3) || 3,
      maxMastersPerPet: numOr(pb.maxMastersPerPet, 3) || 3,
      maxTitleLength: numOr(pb.maxTitleLength, 12) || 12
    };
  }
  if (!c.accessControl || typeof c.accessControl !== "object") c.accessControl = {};
  c.punishmentTags = Array.isArray(c.punishmentTags) ? c.punishmentTags : [];
  c.punishmentSeriesSummaries = Array.isArray(c.punishmentSeriesSummaries) ? c.punishmentSeriesSummaries : [];
  if (!c.punishmentRandomSettings || typeof c.punishmentRandomSettings !== "object") {
    c.punishmentRandomSettings = { orderStep: 2, maxDifficultyOvershoot: 5, minSeriesSteps: 5, maxSeriesSteps: 20 };
  } else {
    const minSteps = numOr(c.punishmentRandomSettings.minSeriesSteps, 5);
    const maxSteps = numOr(c.punishmentRandomSettings.maxSeriesSteps, 20);
    c.punishmentRandomSettings = {
      orderStep: numOr(c.punishmentRandomSettings.orderStep, 2),
      maxDifficultyOvershoot: numOr(c.punishmentRandomSettings.maxDifficultyOvershoot, 5),
      minSeriesSteps: minSteps > 0 ? minSteps : 5,
      maxSeriesSteps: maxSteps > 0 ? maxSteps : 20
    };
  }
  // 旧版 punishments 字段只保留安全解码默认值；任务池/系列详情已迁 SQLite，
  // 不再重建旧前端所需的完整目录（前后端需同步发布并刷新缓存）。
  if (!Array.isArray(c.punishments)) c.punishments = [];
  return c;
}

function pairsToMap(arr: any, fill: any): any {
  const cloneFill = () =>
    typeof fill === "object" && fill !== null ? { ...fill } : fill;
  const out: any = { A: cloneFill(), B: cloneFill() };
  if (!Array.isArray(arr)) return out;
  for (const item of arr) {
    if (item?.key == null) continue;
    // proto3 + defaults:false 会省略 false/0，pair 只剩 {key:"A"} —— 用 fill 还原
    if (item && Object.prototype.hasOwnProperty.call(item, "value")) {
      out[item.key] = item.value;
    } else {
      out[item.key] = cloneFill();
    }
  }
  return out;
}

/** key 为任意 id（大话骰参战玩家）而非固定 A/B 时用这个，不预置任何 key。 */
function pairsToGenericMap(arr: any, valueMapper?: (v: any) => any): any {
  const out: any = {};
  if (!Array.isArray(arr)) return out;
  for (const item of arr) {
    if (item?.key == null) continue;
    out[item.key] = valueMapper ? valueMapper(item.value) : item.value ?? "";
  }
  return out;
}

function materializeLiarsDice(ld: any): any {
  if (!ld || typeof ld !== "object") return ld;
  const out: any = { ...ld };
  out.participantIds = out.participantIds || [];
  out.readyPlayerIds = out.readyPlayerIds || [];
  out.bidHistory = (out.bidHistory || []).map((b: any) => ({
    playerId: b?.playerId || "",
    count: numOr(b?.count, 0),
    face: numOr(b?.face, 0),
    at: numOr(b?.at, 0)
  }));
  out.diceCounts = pairsToGenericMap(out.diceCounts, (v: any) => numOr(v, 0));
  out.revealedHands = pairsToGenericMap(out.revealedHands, (v: any) =>
    Array.isArray(v?.values) ? v.values.map((x: any) => numOr(x, 0)) : []
  );
  out.roundNumber = numOr(out.roundNumber, 0);
  out.actualCount = numOr(out.actualCount, 0);
  out.minPlayers = numOr(out.minPlayers, 0);
  out.maxPlayers = numOr(out.maxPlayers, 0);
  if (out.currentBid) {
    out.currentBid = {
      playerId: out.currentBid.playerId || "",
      count: numOr(out.currentBid.count, 0),
      face: numOr(out.currentBid.face, 0),
      at: numOr(out.currentBid.at, 0)
    };
  } else {
    out.currentBid = null;
  }
  return out;
}

function pairsToOccupantMap(arr: any): any {
  const out: any = { A: null, B: null };
  if (!Array.isArray(arr)) return out;
  for (const item of arr) {
    const key = item?.key;
    const val = item?.value;
    if (!key) continue;
    if (!val) {
      out[key] = null;
      continue;
    }
    if (val.player) out[key] = materializePlayer(val.player);
    else out[key] = null;
  }
  return out;
}

function boardRows(rows: any): any[] {
  if (!Array.isArray(rows)) return [];
  return rows.map((row) => {
    const cells = row?.cells || row || [];
    return (cells as any[]).map((c) => (c === "" || c == null ? null : c));
  });
}

function materializeChessTree(chess: any) {
  if (!chess || typeof chess !== "object") return;
  chess.board = padBoardMatrix(boardRows(chess.board), 8);
  chess.moveCount = numOr(chess.moveCount, 0);
  chess.moveDeadlineAt = numOr(chess.moveDeadlineAt, 0);
  chess.clockDeadlineAt = numOr(chess.clockDeadlineAt, 0);
  chess.halfmoveClock = numOr(chess.halfmoveClock, 0);
  if (chess.lastFrom) chess.lastFrom = { row: numOr(chess.lastFrom.row, 0), col: numOr(chess.lastFrom.col, 0) };
  if (chess.lastTo) chess.lastTo = { row: numOr(chess.lastTo.row, 0), col: numOr(chess.lastTo.col, 0) };
  if (chess.enPassant) chess.enPassant = { row: numOr(chess.enPassant.row, 0), col: numOr(chess.enPassant.col, 0) };
  if (Array.isArray(chess.legalMoves)) {
    chess.legalMoves = chess.legalMoves.map((mv: any) => ({
      from: { row: numOr(mv?.from?.row, 0), col: numOr(mv?.from?.col, 0) },
      to: { row: numOr(mv?.to?.row, 0), col: numOr(mv?.to?.col, 0) },
      promote: typeof mv?.promote === "string" ? mv.promote : ""
    }));
  } else {
    chess.legalMoves = [];
  }
}

/** 保证 n×n 棋盘；缺行/列补 null（避免 proto 省略空串后尺寸变短） */
function padBoardMatrix(matrix: any[], n: number): any[] {
  return padBoardRect(matrix, n, n);
}

/** 保证 rows×cols 棋盘（斗兽棋 9×7） */
function padBoardRect(matrix: any[], rows: number, cols: number): any[] {
  const out: any[] = [];
  for (let r = 0; r < rows; r++) {
    const src = Array.isArray(matrix[r]) ? matrix[r] : [];
    const row: any[] = [];
    for (let c = 0; c < cols; c++) {
      row.push(c < src.length ? src[c] ?? null : null);
    }
    out.push(row);
  }
  return out;
}

/** protobufjs defaults:false 会丢掉 0，坐标必须用 ?? 0 还原 */
function normalizePosList(list: any): Array<{ row: number; col: number }> {
  if (!Array.isArray(list)) return [];
  return list.map((p) => ({
    row: numOr(p?.row, 0),
    col: numOr(p?.col, 0)
  }));
}

function numOr(v: any, fallback: number): number {
  const n = Number(v);
  return Number.isFinite(n) ? n : fallback;
}

// silence unused import if tree-shaken
void game;
