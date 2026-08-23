package server

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/doumiao/newRPS/internal/config"
	"github.com/doumiao/newRPS/internal/types"
)

// logPlayerActivity 记录一条玩家审计事件到 player_activity_events（改名/头像/性别阵营/
// 大话骰名战/送礼/极限模式开关等）。oldValue 只有 rename/avatar_change/gender_change 这类
// "有旧值可对比"的 action 会填（清除头像也并入 avatar_change，不单独记 avatar_clear）；
// text 只有 giveaway_board_submit 会填
// （自救板内容）。调用点都在 s.mu 持锁期间，同步写库是有意为之——这类事件低频、
// user-initiated，和 eventStore 的房间/惩罚事件同一量级，可以复用同一个已验证过的权衡。
func (s *Server) logPlayerActivity(action, playerID, newValue, oldValue, ip, device, fingerprint, text string) {
	if s.activityDB == nil {
		return
	}
	if err := s.activityDB.insertPlayerActivityEvent(nowMs(), action, playerID, newValue, oldValue, ip, device, fingerprint, text); err != nil {
		s.errorLog("player_activity_insert_failed", err.Error())
	}
}

func decodeD[T any](env wsEnvelope, out *T) error {
	if env.D == nil || len(env.D) == 0 {
		return nil
	}
	// 入站已是 map（来自 protobuf Struct / RawBody），再绑定到 handler 结构体。
	// structpb 数字为 float64；json 往返可写入 int/int64 字段（含 0 坐标）。
	b, err := json.Marshal(env.D)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func (s *Server) eventHandler(event string) (RateLimitOptions, eventHandlerFunc) {
	switch event {
	case "player:join":
		return RateLimitOptions{8, 60_000, 60_000}, s.onPlayerJoin
	case "admin:login":
		return RateLimitOptions{5, 60_000, 60_000}, s.onAdminLogin
	case "sync:full":
		return RateLimitOptions{20, 60_000, 15_000}, s.onSyncFull
	case "player:get":
		return RateLimitOptions{30, 60_000, 15_000}, s.onPlayerGet
	case "players:roster":
		// 全站档案榜单按需拉取（不进大厅实时通道）。
		// 前端 pageSize=rosterMaxLimit(500)、最多约 10 页；配额需覆盖完整拉榜 + 偶发重试。
		return RateLimitOptions{30, 60_000, 15_000}, s.onPlayersRoster
	case "lobby:subscribe":
		return RateLimitOptions{20, 60_000, 10_000}, s.onLobbySubscribe
	case "lobby:unsubscribe":
		return RateLimitOptions{20, 60_000, 10_000}, s.onLobbyUnsubscribe
	case "lobby:suggestions:subscribe":
		return RateLimitOptions{20, 60_000, 10_000}, s.onLobbySuggestionsSubscribe
	case "lobby:suggestions:unsubscribe":
		return RateLimitOptions{20, 60_000, 10_000}, s.onLobbySuggestionsUnsubscribe
	case "player:updateProfile":
		return RateLimitOptions{10, 60_000, 30_000}, s.onPlayerUpdateProfile
	case "petbond:getState":
		return RateLimitOptions{20, 60_000, 10_000}, s.onPetBondGetState
	case "petbond:subscribe":
		return RateLimitOptions{20, 60_000, 10_000}, s.onPetBondSubscribe
	case "petbond:unsubscribe":
		return RateLimitOptions{20, 60_000, 10_000}, s.onPetBondUnsubscribe
	case "petbond:seekMaster":
		return RateLimitOptions{10, 60_000, 30_000}, s.onPetBondSeekMaster
	case "petbond:seekPet":
		return RateLimitOptions{10, 60_000, 30_000}, s.onPetBondSeekPet
	case "petbond:approve":
		return RateLimitOptions{20, 60_000, 15_000}, s.onPetBondApprove
	case "petbond:cancel":
		return RateLimitOptions{20, 60_000, 15_000}, s.onPetBondCancel
	case "petbond:requestRelease":
		return RateLimitOptions{10, 60_000, 30_000}, s.onPetBondRequestRelease
	case "petbond:setTitle":
		return RateLimitOptions{10, 60_000, 30_000}, s.onPetBondSetTitle
	case "petbond:forceGiveaway":
		return RateLimitOptions{6, 60_000, 15_000}, s.onPetBondForceGiveaway
	case "giveaway:boost":
		return RateLimitOptions{20, 60_000, 30_000}, s.onGiveawayBoost
	case "giveaway:submitBoard":
		return RateLimitOptions{4, 60_000, 60_000}, s.onGiveawaySubmitBoard
	case "giveaway:vote":
		return RateLimitOptions{30, 60_000, 30_000}, s.onGiveawayVote
	case "giveaway:voteQuotas":
		return RateLimitOptions{20, 60_000, 15_000}, s.onGiveawayVoteQuotas
	case "rankMultiplier:unlock":
		return RateLimitOptions{6, 60_000, 30_000}, s.onRankMultiplierUnlock
	case "extreme:forceClose":
		return RateLimitOptions{4, 60_000, 60_000}, s.onExtremeForceClose
	case "nameWar:renameTarget":
		return RateLimitOptions{8, 60_000, 60_000}, s.onNameWarRenameTarget
	case "config:get":
		return RateLimitOptions{6, 60_000, 30_000}, s.onConfigGet
	case "config:save":
		return RateLimitOptions{6, 60_000, 60_000}, s.onConfigSave
	case "config:reset":
		return RateLimitOptions{3, 60_000, 60_000}, s.onConfigReset
	case "room:create":
		return RateLimitOptions{5, 60_000, 60_000}, s.onRoomCreate
	case "room:join":
		return RateLimitOptions{12, 60_000, 45_000}, s.onRoomJoin
	case "room:leave":
		return RateLimitOptions{12, 60_000, 45_000}, s.onRoomLeave
	case "room:history":
		return RateLimitOptions{30, 60_000, 30_000}, s.onRoomHistory
	case "room:sit":
		return RateLimitOptions{12, 60_000, 30_000}, s.onRoomSit
	case "room:spectate":
		return RateLimitOptions{12, 60_000, 30_000}, s.onRoomSpectate
	case "room:move":
		return RateLimitOptions{20, 10_000, 20_000}, s.onRoomMove
	case "room:move:withdraw":
		return RateLimitOptions{20, 10_000, 20_000}, s.onRoomMoveWithdraw
	case "othello:ready":
		return RateLimitOptions{12, 60_000, 30_000}, s.onOthelloReady
	case "othello:move":
		return RateLimitOptions{30, 10_000, 15_000}, s.onOthelloMove
	case "othello:undoRequest":
		return RateLimitOptions{6, 60_000, 15_000}, s.onOthelloUndoRequest
	case "othello:undoRespond":
		return RateLimitOptions{8, 60_000, 5_000}, s.onOthelloUndoRespond
	case "othello:settleMove":
		return RateLimitOptions{12, 60_000, 3_000}, s.onOthelloSettleMove
	case "othello:requestSurrender", "othello:surrender":
		return RateLimitOptions{5, 60_000, 8_000}, s.onOthelloRequestSurrender
	case "othello:respondSurrender":
		return RateLimitOptions{8, 60_000, 5_000}, s.onOthelloRespondSurrender
	case "othello:escape":
		return RateLimitOptions{3, 60_000, 20_000}, s.onOthelloEscape
	case "othello:restart":
		return RateLimitOptions{8, 60_000, 30_000}, s.onOthelloRestart
	case "tictactoe:ready":
		return RateLimitOptions{12, 60_000, 30_000}, s.onTicTacToeReady
	case "tictactoe:move":
		return RateLimitOptions{30, 10_000, 15_000}, s.onTicTacToeMove
	case "tictactoe:giveawayChoice":
		return RateLimitOptions{20, 10_000, 15_000}, s.onTicTacToeGiveawayChoice
	case "tictactoe:restart":
		return RateLimitOptions{8, 60_000, 30_000}, s.onTicTacToeRestart
	case "gomoku:ready":
		return RateLimitOptions{12, 60_000, 30_000}, s.onGomokuReady
	case "gomoku:giveaway":
		return RateLimitOptions{20, 10_000, 15_000}, s.onGomokuGiveaway
	case "gomoku:move":
		return RateLimitOptions{30, 10_000, 15_000}, s.onGomokuMove
	case "gomoku:undoRequest":
		return RateLimitOptions{6, 60_000, 15_000}, s.onGomokuUndoRequest
	case "gomoku:undoRespond":
		return RateLimitOptions{8, 60_000, 5_000}, s.onGomokuUndoRespond
	case "gomoku:resignRequest":
		return RateLimitOptions{5, 60_000, 8_000}, s.onGomokuResignRequest
	case "gomoku:resignRespond":
		return RateLimitOptions{8, 60_000, 5_000}, s.onGomokuResignRespond
	case "gomoku:restart":
		return RateLimitOptions{8, 60_000, 30_000}, s.onGomokuRestart
	case "jungle:ready":
		return RateLimitOptions{12, 60_000, 30_000}, s.onJungleReady
	case "jungle:move":
		return RateLimitOptions{30, 10_000, 15_000}, s.onJungleMove
	case "jungle:undoRequest":
		return RateLimitOptions{6, 60_000, 15_000}, s.onJungleUndoRequest
	case "jungle:undoRespond":
		return RateLimitOptions{8, 60_000, 5_000}, s.onJungleUndoRespond
	case "jungle:resignRequest":
		return RateLimitOptions{5, 60_000, 8_000}, s.onJungleResignRequest
	case "jungle:resignRespond":
		return RateLimitOptions{8, 60_000, 5_000}, s.onJungleResignRespond
	case "jungle:restart":
		return RateLimitOptions{8, 60_000, 30_000}, s.onJungleRestart
	case "chess:ready":
		return RateLimitOptions{12, 60_000, 30_000}, s.onChessReady
	case "chess:move":
		return RateLimitOptions{30, 10_000, 15_000}, s.onChessMove
	case "chess:undoRequest":
		return RateLimitOptions{6, 60_000, 15_000}, s.onChessUndoRequest
	case "chess:undoRespond":
		return RateLimitOptions{8, 60_000, 5_000}, s.onChessUndoRespond
	case "chess:resignRequest":
		return RateLimitOptions{5, 60_000, 8_000}, s.onChessResignRequest
	case "chess:resignRespond":
		return RateLimitOptions{8, 60_000, 5_000}, s.onChessResignRespond
	case "chess:restart":
		return RateLimitOptions{8, 60_000, 30_000}, s.onChessRestart
	case "liarsdice:joinRoster":
		return RateLimitOptions{12, 60_000, 15_000}, s.onLiarsDiceJoinRoster
	case "liarsdice:leaveRoster":
		return RateLimitOptions{12, 60_000, 15_000}, s.onLiarsDiceLeaveRoster
	case "liarsdice:ready":
		return RateLimitOptions{12, 60_000, 30_000}, s.onLiarsDiceReady
	case "liarsdice:bid":
		return RateLimitOptions{30, 10_000, 15_000}, s.onLiarsDiceBid
	case "liarsdice:challenge":
		return RateLimitOptions{20, 10_000, 15_000}, s.onLiarsDiceChallenge
	case "liarsdice:nextRound":
		return RateLimitOptions{12, 60_000, 15_000}, s.onLiarsDiceNextRound
	case "punishment:submit":
		return RateLimitOptions{8, 60_000, 60_000}, s.onPunishmentSubmit
	case "punishment:assignTask":
		return RateLimitOptions{10, 60_000, 45_000}, s.onPunishmentAssignTask
	case "punishment:review":
		return RateLimitOptions{15, 60_000, 45_000}, s.onPunishmentReview
	case "punishment:confirm":
		return RateLimitOptions{15, 60_000, 45_000}, s.onPunishmentConfirm
	case "chat:send":
		return RateLimitOptions{20, 60_000, 30_000}, s.onChatSend
	case "chat:load":
		return RateLimitOptions{30, 60_000, 10_000}, s.onChatLoad
	case "chat:loadOlder":
		return RateLimitOptions{40, 60_000, 10_000}, s.onChatLoadOlder
	case "contribution:list":
		return RateLimitOptions{20, 60_000, 15_000}, s.onContributionList
	case "contribution:get":
		return RateLimitOptions{20, 60_000, 15_000}, s.onContributionGet
	case "contribution:saveDraft":
		return RateLimitOptions{12, 60_000, 30_000}, s.onContributionSaveDraft
	case "contribution:submit":
		return RateLimitOptions{8, 60_000, 60_000}, s.onContributionSubmit
	case "contribution:withdraw":
		return RateLimitOptions{8, 60_000, 30_000}, s.onContributionWithdraw
	case "contribution:vote":
		return RateLimitOptions{20, 60_000, 15_000}, s.onContributionVote
	case "contribution:votePreview":
		return RateLimitOptions{30, 60_000, 10_000}, s.onContributionVotePreview
	case "admin:action":
		return RateLimitOptions{30, 60_000, 60_000}, s.onAdminAction
	case "admin:listPlayers":
		return RateLimitOptions{30, 60_000, 15_000}, s.onAdminListPlayers
	case "admin:chatSearch":
		return RateLimitOptions{30, 60_000, 15_000}, s.onAdminChatSearch
	case "admin:petBondGraph":
		return RateLimitOptions{30, 60_000, 15_000}, s.onAdminPetBondGraph
	case "admin:petBondAdd":
		return RateLimitOptions{20, 60_000, 15_000}, s.onAdminPetBondAdd
	case "admin:petBondRemove":
		return RateLimitOptions{20, 60_000, 15_000}, s.onAdminPetBondRemove
	case "admin:analytics":
		// 前端进入分区后 60s 轮询 + 手动刷新 + 切时间范围重拉；30/min 绰绰有余。
		return RateLimitOptions{30, 60_000, 15_000}, s.onAdminAnalytics
	case "admin:analyticsDetail":
		return RateLimitOptions{30, 60_000, 15_000}, s.onAdminAnalyticsDetail
	case "analytics:collect":
		// 前端 5s 一次 flush = 12/min，留 2.5 倍余量应对可见性切换的补发；触顶冷却 30s。
		if !s.analyticsEnabled {
			return RateLimitOptions{}, nil
		}
		return RateLimitOptions{30, 60_000, 30_000}, s.onAnalyticsCollect
	case "identity:showClaimKey":
		return RateLimitOptions{10, 60_000, 15_000}, s.onIdentityShowClaimKey
	case "identity:refreshClaimKey":
		return RateLimitOptions{5, 60_000, 15_000}, s.onIdentityRefreshClaimKey
	case "identity:claim":
		return RateLimitOptions{5, 60_000, 60_000}, s.onIdentityClaim
	case "identity:logout":
		return RateLimitOptions{5, 60_000, 15_000}, s.onIdentityLogout
	case "push:subscribe":
		return RateLimitOptions{10, 60_000, 15_000}, s.onPushSubscribe
	case "push:unsubscribe":
		return RateLimitOptions{10, 60_000, 15_000}, s.onPushUnsubscribe
	case "push:getPreferences":
		return RateLimitOptions{10, 60_000, 15_000}, s.onPushGetPreferences
	case "push:updatePreferences":
		return RateLimitOptions{10, 60_000, 15_000}, s.onPushUpdatePreferences
	case "push:test":
		return RateLimitOptions{5, 60_000, 15_000}, s.onPushTest
	default:
		return RateLimitOptions{}, nil
	}
}

func (s *Server) onPlayerJoin(client *Client, env wsEnvelope) {
	var p struct {
		Name         string `json:"name"`
		GenderID     string `json:"genderId"`
		PlayerID     string `json:"playerId"`
		PlayerSecret string `json:"playerSecret"`
		Fingerprint  string `json:"fingerprint"`
		ForceKick    bool   `json:"forceKick"`
	}
	_ = decodeD(env, &p)
	cleanName := cleanText(p.Name, 12)
	if len([]rune(cleanName)) < 2 {
		msg := "请输入 2-12 个字符的名字"
		if s.cfg.Messages != nil && s.cfg.Messages["nameRequired"] != "" {
			msg = s.cfg.Messages["nameRequired"]
		}
		client.reply(env.ID, nil, msg)
		return
	}
	sid := client.sid
	ipAddress := client.ipAddress
	// join 可补全/刷新指纹（WS 握手时可能尚未带上）；若 deviceKey 变化需迁移套接字索引
	prevDevice := client.deviceKey
	if fp := normalizeFingerprint(p.Fingerprint); fp != "missing" || client.fingerprint == "" {
		client.fingerprint = normalizeFingerprint(p.Fingerprint)
		client.deviceKey = deviceKey(ipAddress, client.fingerprint)
	} else if client.deviceKey == "" {
		client.deviceKey = deviceKey(ipAddress, client.fingerprint)
	}
	if prevDevice != "" && prevDevice != client.deviceKey {
		if set := s.clientIDsByDevice[prevDevice]; set != nil {
			delete(set, client.id)
			if len(set) == 0 {
				delete(s.clientIDsByDevice, prevDevice)
			}
		}
		s.ensureDeviceSocketSet(client.deviceKey)[client.id] = struct{}{}
	}
	// 指纹补报会改 deviceKey：同步刷新分析 visitor id（不在此处做 IP 查表）。
	if prevDevice != client.deviceKey {
		client.anaVisitor = analyticsVisitorID(client.deviceKey, s.analyticsSalt)
	}
	device := client.deviceKey
	var player *PlayerState
	if p.PlayerID != "" {
		if id := s.playerIdToID[p.PlayerID]; id != "" {
			player = s.players[id]
		}
	} else {
		player = s.players[sid]
	}
	reissuedSecret := ""
	if player != nil && player.Persistent {
		if !s.verifyPlayerSecret(player, p.PlayerSecret) {
			s.securityLog("player_identity_invalid", map[string]any{"sid": sid, "ip": ipAddress, "device": device, "userAgent": client.userAgent})
			client.reply(env.ID, nil, "玩家身份校验失败")
			return
		}
		// 旧版前端曾用双 UUID 拼接生成 secret，新版统一成单个 token；验证通过后顺手
		// 静默换发一条新格式凭据，原地替换（先删后加，不占用/挤掉设备槽位）。
		if isLegacySecretFormat(p.PlayerSecret) {
			reissuedSecret = randomPlayerSecret()
			player.removePlayerSecret(p.PlayerSecret)
			player.addPlayerSecret(reissuedSecret)
		}
	}
	if player == nil {
		if s.cfg.AccessControl.RegistrationDisabled {
			client.reply(env.ID, nil, "当前暂停新用户注册，请使用已有账号登录")
			return
		}
		if s.onlinePlayersFromDevice(device, "") >= s.cfg.AccessControl.MaxOnlinePerIP {
			client.reply(env.ID, nil, fmt.Sprintf("当前设备同时在线人数过多，最多允许 %d 人同时在线", s.cfg.AccessControl.MaxOnlinePerIP))
			return
		}
		if s.onlinePlayersFromIP(ipAddress, "") >= s.cfg.AccessControl.MaxOnlinePerIPTotal {
			client.reply(env.ID, nil, "当前网络环境同时在线人数过多，请稍后再试")
			return
		}
		if !s.canCreateFromDevice(device) {
			client.reply(env.ID, nil, fmt.Sprintf("当前设备 10 分钟内新建玩家过多，最多允许 %d 次", s.cfg.AccessControl.MaxCreatesPer10Min))
			return
		}
		if !s.canCreateFromIP(ipAddress) {
			client.reply(env.ID, nil, "当前网络环境 10 分钟内新建玩家过多，请稍后再试")
			return
		}
		player = s.createPlayer(cleanName, p.GenderID, client.token, p.PlayerID, p.PlayerSecret)
		s.logPlayerActivity("create", player.ID, cleanName, "", ipAddress, device, client.fingerprint, "")
	}
	wasDisconnected := !player.Connected
	hadDisconnectHold := player.DisconnectExpiresAt != nil
	previousSocketID := player.SocketID
	previousRoomID := player.RoomID
	if previousSocketID != "" && previousSocketID != client.id {
		if prev := s.clients[previousSocketID]; prev != nil {
			sameDevice := isSameDevice(player.DeviceKey, device)
			if needsKickConfirm(sameDevice, p.ForceKick) {
				client.reply(env.ID, map[string]any{"alreadyOnline": true}, "")
				return
			}
			if !sameDevice {
				s.emitToClient(previousSocketID, "session:kicked", map[string]any{
					"message": "你的账号已在其他设备登录，此会话已结束。请刷新页面重新登录。",
				})
			}
			s.clientLeaveRoom(prev, player.ID)
			if previousRoomID != "" {
				s.clientLeaveRoom(prev, previousRoomID)
			}
		}
	}
	if player.Persistent && p.PlayerSecret != "" {
		player.ActiveSecret = p.PlayerSecret
		if reissuedSecret != "" {
			player.ActiveSecret = reissuedSecret
		}
	}
	player.SocketID = client.id
	player.IPAddress = ipAddress
	player.Fingerprint = client.fingerprint
	player.DeviceKey = device
	player.Connected = true
	player.CurrentSID = sid
	player.LastSeenAt = nowMs()
	player.DisconnectedAt = nil
	player.DisconnectExpiresAt = nil
	client.playerID = player.ID
	if sid != "" {
		s.sidToPlayerID[sid] = player.ID
	}
	sessionToken := client.token
	if sessionToken == "" {
		sessionToken = player.Token
	}
	// 与当前 WS 会话对齐，避免客户端把过期的历史 player.Token 写回 localStorage。
	if sessionToken != "" {
		if player.Token != "" && player.Token != sessionToken {
			delete(s.tokenToPlayer, player.Token)
		}
		player.Token = sessionToken
		s.tokenToPlayer[sessionToken] = player.ID
	}
	if !ptrBool(player.NameWarEnabled) {
		player.Name = cleanName
		player.NameWarOriginalName = cleanName
	}
	s.applyGender(player, p.GenderID)
	s.refreshNameWarState(player, nowMs())
	s.clearDisconnectHold(player)
	s.clearDisconnectForfeit(player)
	if wasDisconnected && hadDisconnectHold {
		s.serverStats.Reconnects++
	}
	s.refreshPlayerSnapshots(player)
	if player.RoomID != "" {
		existing := s.rooms[player.RoomID]
		if existing == nil || !s.roomHasPlayer(existing, player.ID) {
			player.RoomID = ""
		}
	}
	s.clientJoinRoom(client, player.ID)
	if player.RoomID != "" {
		s.clientLeaveRoom(client, lobbyChannel)
		s.clientJoinRoom(client, player.RoomID)
	} else {
		s.clientJoinRoom(client, lobbyChannel)
		s.clientJoinRoom(client, lobbySuggestionChannel)
	}
	var roomSnap any
	if player.RoomID != "" {
		if room := s.rooms[player.RoomID]; room != nil {
			roomSnap = s.roomSnapshot(room, true, true)
		}
	}
	joinReply := map[string]any{
		"player": s.publicPlayer(player),
		"token":  sessionToken,
		"roomId": player.RoomID,
		"room":   roomSnap,
	}
	if reissuedSecret != "" {
		joinReply["reissuedSecret"] = reissuedSecret
	}
	client.reply(env.ID, joinReply, "")
	if player.Persistent {
		s.markPlayerDirty(player)
		s.requestPersist("lazy")
	}
	// 新上线/重连：player:batch 先推本人变更；大厅走防抖 DELTA（connected/players 增删
	// 可由路径补丁表达）。forceBroadcastLobby（清空基线 → 全员 FULL）仅保留给
	// 房间删除等结构变化（见 room.go / admin close），避免重连风暴放大流量。
	s.broadcastPlayerUpdate(player)
	s.broadcastLobby()
	// 宠物乐园候选列表依赖在线状态，上线后推送给全体。
	s.notifyAllOnlinePetBondStates()
	// 离线→在线的真实转换（而非同一会话的刷新/重连保活）才提醒 Ta 的主人/宠物。
	if wasDisconnected {
		s.notifyBondOnline(player)
	}
	if player.RoomID != "" {
		if room := s.rooms[player.RoomID]; room != nil {
			if room.Phase == types.PhasePunishment && hadDisconnectHold {
				s.roomNotice(room, playerShortName(player)+" 已重新连接，恢复到未完成的惩罚房间。")
			}
			// 大话骰私有骰子只在开局时单播一次；重连玩家的房间快照里没有自己的手牌，
			// 需要在对局进行中补发一次，否则重连后看不到自己的骰子。
			if room.Settings.GameID == types.GameLiarsDice && room.Phase == types.PhaseChoosing {
				if dice := room.LiarsDiceHands[player.ID]; len(dice) > 0 {
					s.emitToClient(player.SocketID, "liarsdice:hand", map[string]any{
						"roomId": room.ID,
						"dice":   dice,
					})
				}
			}
		}
		s.broadcastRoom(player.RoomID, false)
	}
}

func (s *Server) onAdminLogin(client *Client, env wsEnvelope) {
	var p struct {
		Password string `json:"password"`
	}
	_ = decodeD(env, &p)
	player := s.getPlayerByClient(client)
	if !s.adminPasswordMatches(p.Password, client.ipAddress) {
		client.reply(env.ID, nil, "管理员口令不正确或尚未设置")
		return
	}
	s.adminClientIDs[client.id] = struct{}{}
	// IsAdmin 仅作本连接存活期间的展示标记；权限以 adminClientIDs 为准。
	// 断线时会清掉，重连后必须重新 admin:login，避免 IsAdmin 粘滞到进程结束。
	if player != nil {
		player.IsAdmin = boolPtr(true)
		s.refreshPlayerSnapshots(player)
		s.broadcastPlayerUpdate(player)
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastLobby()
}

func (s *Server) onLobbySubscribe(client *Client, env wsEnvelope) {
	s.clientJoinRoom(client, lobbyChannel)
	s.clientJoinRoom(client, lobbySuggestionChannel)
	s.sendFullChannel(client, channelLobby())
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onLobbyUnsubscribe(client *Client, env wsEnvelope) {
	s.clientLeaveRoom(client, lobbyChannel)
	s.clientLeaveRoom(client, lobbySuggestionChannel)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

// onLobbySuggestionsSubscribe：仅加入大厅聊天实时频道（房间内的「大厅」tab 用），
// 历史消息由前端另外调 chat:load 拉取。
func (s *Server) onLobbySuggestionsSubscribe(client *Client, env wsEnvelope) {
	s.clientJoinRoom(client, lobbySuggestionChannel)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onLobbySuggestionsUnsubscribe(client *Client, env wsEnvelope) {
	s.clientLeaveRoom(client, lobbySuggestionChannel)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onPetBondSubscribe(client *Client, env wsEnvelope) {
	player, ok := s.requirePlayer(client, env)
	if !ok {
		return
	}
	s.clientJoinRoom(client, petbondChannel)
	// 订阅后立刻推一包个性化状态，省去客户端再调 getState。
	client.reply(env.ID, s.buildPetBondState(player.ID), "")
}

func (s *Server) onPetBondUnsubscribe(client *Client, env wsEnvelope) {
	s.clientLeaveRoom(client, petbondChannel)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onPlayerUpdateProfile(client *Client, env wsEnvelope) {
	var p struct {
		Name               string  `json:"name"`
		GenderID           string  `json:"genderId"`
		SelfTitle          *string `json:"selfTitle"`
		NameWarEnabled     *bool   `json:"nameWarEnabled"`
		NameWarAllowRename *bool   `json:"nameWarAllowRename"`
		GiveawayEnabled    *bool   `json:"giveawayEnabled"`
		ExtremeModeEnabled *bool   `json:"extremeModeEnabled"`
		BondMasterEnabled  *bool   `json:"bondMasterEnabled"`
		BondPetEnabled     *bool   `json:"bondPetEnabled"`
		BondPublicDisplay  *bool   `json:"bondPublicDisplay"`
	}
	_ = decodeD(env, &p)
	player, ok := s.requirePlayer(client, env)
	if !ok {
		return
	}
	cleanName := cleanText(p.Name, 12)
	if len([]rune(cleanName)) < 2 {
		msg := "请输入 2-12 个字符的名字"
		if s.cfg.Messages != nil && s.cfg.Messages["nameRequired"] != "" {
			msg = s.cfg.Messages["nameRequired"]
		}
		client.reply(env.ID, nil, msg)
		return
	}
	now := nowMs()
	oldName := player.Name
	nameChanged := cleanName != player.Name
	nextNameWarEnabled := p.NameWarEnabled != nil && *p.NameWarEnabled
	nextAllowRename := nextNameWarEnabled && p.NameWarAllowRename != nil && *p.NameWarAllowRename
	nameWarChanged := nextNameWarEnabled != ptrBool(player.NameWarEnabled)
	allowRenameChanged := nextAllowRename != ptrBool(player.NameWarAllowRename)
	nextGiveawayEnabled := p.GiveawayEnabled != nil && *p.GiveawayEnabled
	giveawayChanged := nextGiveawayEnabled != ptrBool(player.GiveawayEnabled)
	nextExtremeModeEnabled := p.ExtremeModeEnabled != nil && *p.ExtremeModeEnabled
	extremeModeChanged := nextExtremeModeEnabled != ptrBool(player.ExtremeModeEnabled)

	if nameChanged && (ptrBool(player.NameWarEnabled) || nextNameWarEnabled) {
		client.reply(env.ID, nil, "名字争夺战开启后不能修改自己的名字")
		return
	}
	if nameChanged && player.ProfileUpdatedAt != nil && now-*player.ProfileUpdatedAt < 60_000 {
		seconds := int(math.Ceil(float64(60_000-(now-*player.ProfileUpdatedAt)) / 1000))
		client.reply(env.ID, nil, fmt.Sprintf("改名太频繁，请 %d 秒后再试", seconds))
		return
	}
	if (nameWarChanged || allowRenameChanged) && player.NameWarToggledAt != nil && now-*player.NameWarToggledAt < 43_200_000 {
		hours := int(math.Ceil(float64(43_200_000-(now-*player.NameWarToggledAt)) / 3_600_000))
		client.reply(env.ID, nil, fmt.Sprintf("名字争夺战冷却中，请 %d 小时后再试", hours))
		return
	}
	if ptrBool(player.GiveawayEnabled) && !nextGiveawayEnabled && ptrFloat(player.GiveawayValue) > 0 {
		client.reply(env.ID, nil, "白给值归零前不能关闭白给模式")
		return
	}
	if ok, reason := s.validGenderSubmission(p.GenderID); !ok {
		client.reply(env.ID, nil, reason)
		return
	}
	if extremeModeChanged && nextExtremeModeEnabled {
		if player.ExtremeModeCooldownUntil != nil && *player.ExtremeModeCooldownUntil > now {
			hours := int(math.Ceil(float64(*player.ExtremeModeCooldownUntil-now) / 3_600_000))
			client.reply(env.ID, nil, fmt.Sprintf("极限模式冷却中，请 %d 小时后再开启", hours))
			return
		}
		if player.Stats.RankedPoints < 0 {
			client.reply(env.ID, nil, "负分玩家不能开启极限模式")
			return
		}
	}
	if extremeModeChanged && !nextExtremeModeEnabled && player.Stats.RankedPoints <= 0 {
		client.reply(env.ID, nil, "排位分必须大于 0 才能关闭极限模式，0 分不能关闭")
		return
	}
	if nameChanged {
		player.Name = cleanName
		player.NameWarOriginalName = cleanName
		s.logPlayerActivity("rename", player.ID, cleanName, oldName, client.ipAddress, client.deviceKey, client.fingerprint, "")
	}
	exitedHardMode := ptrBool(player.NameWarAllowRename) && !nextAllowRename
	if nameWarChanged || allowRenameChanged {
		player.NameWarEnabled = boolPtr(nextNameWarEnabled)
		player.NameWarAllowRename = boolPtr(nextAllowRename)
		player.NameWarToggledAt = int64Ptr(now)
		if !nextNameWarEnabled {
			s.syncTitleForRankSegment(player, false)
		}
		// 只记「开启」，关闭不产生活动埋点（数据分析面板只关心名争的开启量）。
		if nameWarChanged && nextNameWarEnabled {
			s.logPlayerActivity("nameWar_enable", player.ID, player.Name, "", client.ipAddress, client.deviceKey, client.fingerprint, "")
		}
	}
	oldGenderSignature := player.GenderID + "|" + player.GenderLabel + "|" + player.FactionID
	oldGenderLabel := player.GenderLabel
	s.applyGender(player, p.GenderID)
	if newSignature := player.GenderID + "|" + player.GenderLabel + "|" + player.FactionID; newSignature != oldGenderSignature {
		s.logPlayerActivity("gender_change", player.ID, player.GenderLabel, oldGenderLabel, client.ipAddress, client.deviceKey, client.fingerprint, "")
	}
	s.refreshNameWarState(player, now)
	if exitedHardMode {
		player.Stats.Title = s.cfg.NameWar.EscapeTitle
		if player.Stats.Title == "" {
			player.Stats.Title = "逃跑的人"
		}
	}
	player.GiveawayEnabled = boolPtr(nextGiveawayEnabled)
	if giveawayChanged && nextGiveawayEnabled {
		// 打开白给玩法时白给值从 0 起步，固定给到 0.1%（能关闭的前提是值必须先降到 0，
		// 所以走到这里之前 GiveawayValue 必然是 0，直接赋值不用再和旧值比较）。
		player.GiveawayValue = floatPtr(0.1)
	}
	// 只记「开启」，关闭不产生活动埋点。
	if giveawayChanged && nextGiveawayEnabled {
		s.logPlayerActivity("giveaway_enable", player.ID, player.Name, "", client.ipAddress, client.deviceKey, client.fingerprint, "")
	}
	if !ptrBool(player.GiveawayEnabled) && ptrFloat(player.GiveawayValue) <= 0 {
		player.GiveawayValue = floatPtr(0)
		player.GiveawayBoardText = ""
		player.GiveawayBoardSubmittedAt = nil
		player.GiveawayBoardExpiresAt = nil
	}
	if extremeModeChanged {
		player.ExtremeModeEnabled = boolPtr(nextExtremeModeEnabled)
		player.ExtremeModeToggledAt = int64Ptr(now)
		player.ExtremeWinStreak = intPtr(0)
		if nextExtremeModeEnabled {
			player.Stats.RankedPoints = 0
			player.ExtremeLastDecayHour = int64Ptr(currentExtremeDecayHour(now))
			s.syncTitleForRankSegment(player, false)
		} else {
			player.ExtremeModeCooldownUntil = int64Ptr(now + int64(s.cfg.ExtremeMode.CooldownHours)*3_600_000)
		}
		// 只记「开启」，关闭不产生活动埋点。
		if nextExtremeModeEnabled {
			s.logPlayerActivity("extreme_enable", player.ID, player.Name, "", client.ipAddress, client.deviceKey, client.fingerprint, "")
		}
	}
	// 认主/认宠开关：关闭不解除已有关系，只禁止新增；公开展示关闭则大厅关系图隐藏自己。
	nextBondMaster := p.BondMasterEnabled != nil && *p.BondMasterEnabled
	nextBondPet := p.BondPetEnabled != nil && *p.BondPetEnabled
	nextBondPublic := p.BondPublicDisplay != nil && *p.BondPublicDisplay
	bondChanged := nextBondMaster != ptrBool(player.BondMasterEnabled) ||
		nextBondPet != ptrBool(player.BondPetEnabled) ||
		nextBondPublic != ptrBool(player.BondPublicDisplay)
	player.BondMasterEnabled = boolPtr(nextBondMaster)
	player.BondPetEnabled = boolPtr(nextBondPet)
	player.BondPublicDisplay = boolPtr(nextBondPublic)
	// SelfTitle 为 nil 表示没有改动这一栏；非空文本是玩家自设称号（展示优先级低于管理员/
	// 主人设置，高于系统按排位分自动计算的称号，见 petbond.go 的 applyDisplayTitle）；清空
	// 则视为主动清除自设称号，回退到更下一级的展示。
	if p.SelfTitle != nil {
		oldSelfTitle := player.Stats.SelfTitle
		newSelfTitle := cleanText(*p.SelfTitle, 12)
		if newSelfTitle != oldSelfTitle {
			player.Stats.SelfTitle = newSelfTitle
			s.logPlayerActivity("self_title_change", player.ID, newSelfTitle, oldSelfTitle, client.ipAddress, client.deviceKey, client.fingerprint, "")
		}
	}
	if nameChanged {
		player.ProfileUpdatedAt = int64Ptr(now)
	}
	player.DisplayName = s.formatDisplayName(player)
	s.refreshPlayerSnapshots(player)
	s.broadcastPlayerUpdate(player)
	if bondChanged {
		s.broadcastLobby()
		// 其他人的认主/认宠候选列表依赖这些开关，需实时推送。
		s.notifyAllOnlinePetBondStates()
	}
	if player.Persistent {
		s.markPlayerDirty(player)
		s.requestPersist("lazy")
	}
	client.reply(env.ID, map[string]any{"player": s.publicPlayer(player)}, "")
	if player.RoomID != "" {
		s.broadcastRoom(player.RoomID, false)
	}
}

func (s *Server) onGiveawayBoost(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if !ptrBool(player.GiveawayEnabled) {
		client.reply(env.ID, nil, "请先在个人设置开启白给模式")
		return
	}
	if _, ok := s.seatOf(room, player.ID); !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以白给")
		return
	}
	if !s.isFullHumanRoom(room) {
		client.reply(env.ID, nil, "双方都入座后才能使用白给模式")
		return
	}
	if room.Phase == types.PhasePunishment {
		client.reply(env.ID, nil, "惩罚阶段不能增加白给值")
		return
	}
	player.GiveawayClicks = intPtr(ptrInt(player.GiveawayClicks) + 1)
	s.addGiveawayValue(player, s.cfg.Giveaway.ActiveBoostValue)
	client.reply(env.ID, map[string]any{"player": s.publicPlayer(player)}, "")
	s.broadcastRoom(room.ID, false)
}

func (s *Server) onGiveawaySubmitBoard(client *Client, env wsEnvelope) {
	var p struct {
		Text string `json:"text"`
	}
	_ = decodeD(env, &p)
	player, ok := s.requirePlayerInGame(client, env)
	if !ok {
		return
	}
	if !ptrBool(player.GiveawayEnabled) || ptrFloat(player.GiveawayValue) <= 0 {
		client.reply(env.ID, nil, "白给值大于 0% 时才能上板自救")
		return
	}
	cleanBoardText := cleanText(p.Text, 300)
	if len([]rune(cleanBoardText)) < 2 {
		client.reply(env.ID, nil, "自我惩罚宣言至少需要 2 个字")
		return
	}
	now := nowMs()
	player.GiveawayBoardText = cleanBoardText
	player.GiveawayBoardSubmittedAt = int64Ptr(now)
	exp := now + int64(giveawayBoardDuration/1e6) // duration is time.Duration
	// giveawayBoardDuration is 12 hours as time.Duration
	exp = now + 12*60*60*1000
	player.GiveawayBoardExpiresAt = int64Ptr(exp)
	player.GiveawayBoardLikes = intPtr(0)
	player.GiveawayBoardDislikes = intPtr(0)
	player.GiveawayBoardLikeWindowStartedAt = int64Ptr(now)
	player.GiveawayBoardLikesThisHour = intPtr(0)
	s.logPlayerActivity("giveaway_board_submit", player.ID, player.Name, "", client.ipAddress, client.deviceKey, client.fingerprint, cleanBoardText)
	s.refreshPlayerSnapshots(player)
	s.broadcastPlayerUpdate(player)
	client.reply(env.ID, map[string]any{"player": s.publicPlayer(player)}, "")
}

// onGiveawayVote 处理白给自救板的点赞/倒赞。额度按 actor→target 这一对独立计时/计次
// （见 giveawayVoteQuotaFor），不是 actor 的全局总量——同一 actor 对不同 target 互不影响，
// 对同一 target 的点赞额度和倒赞额度也分开计。上限与升降值都按认主认宠关系分档（giveawayVoteRulesFor）。
func (s *Server) onGiveawayVote(client *Client, env wsEnvelope) {
	var p struct {
		TargetID string `json:"targetId"`
		Vote     string `json:"vote"`
	}
	_ = decodeD(env, &p)
	actor := s.getPlayerByClient(client)
	target := s.players[p.TargetID]
	if actor == nil {
		client.reply(env.ID, nil, "请先进入游戏")
		return
	}
	if target == nil {
		client.reply(env.ID, nil, "上板玩家不存在")
		return
	}
	s.refreshGiveawayBoard(target, nowMs())
	if actor.ID == target.ID {
		client.reply(env.ID, nil, "不能给自己投票")
		return
	}
	if target.GiveawayBoardText == "" || target.GiveawayBoardExpiresAt == nil || *target.GiveawayBoardExpiresAt <= nowMs() {
		client.reply(env.ID, nil, "这条自救内容已经不在板上")
		return
	}
	if p.Vote != "like" && p.Vote != "dislike" {
		client.reply(env.ID, nil, "投票类型不正确")
		return
	}
	now := nowMs()
	likeLimit, dislikeLimit, likeValue, dislikeValue := s.giveawayVoteRulesFor(actor, target)
	quota := s.giveawayVoteQuotaFor(actor, target.ID, now)
	if p.Vote == "like" {
		if quota.Likes >= likeLimit {
			client.reply(env.ID, nil, "你对这位玩家本小时的点赞降值次数已满")
			return
		}
		quota.Likes++
		target.GiveawayBoardLikes = intPtr(ptrInt(target.GiveawayBoardLikes) + 1)
		s.addGiveawayValue(target, -likeValue)
		if ptrFloat(target.GiveawayValue) <= 0 {
			target.GiveawayBoardText = ""
			target.GiveawayBoardSubmittedAt = nil
			target.GiveawayBoardExpiresAt = nil
		}
	} else {
		if quota.Dislikes >= dislikeLimit {
			client.reply(env.ID, nil, "你对这位玩家本小时的倒赞加值次数已满")
			return
		}
		quota.Dislikes++
		target.GiveawayBoardDislikes = intPtr(ptrInt(target.GiveawayBoardDislikes) + 1)
		s.addGiveawayValue(target, dislikeValue)
	}
	s.refreshPlayerSnapshots(target)
	s.updateLegacyGiveawayVoteStats(actor, p.Vote, now)
	s.broadcastPlayerUpdate(actor)
	s.broadcastPlayerUpdate(target)
	client.reply(env.ID, map[string]any{"ok": true, "quota": s.giveawayVoteQuotaView(actor, target, now)}, "")
	if target.RoomID != "" {
		s.broadcastRoom(target.RoomID, false)
	}
	if actor.RoomID != "" && actor.RoomID != target.RoomID {
		s.broadcastRoom(actor.RoomID, false)
	}
}

// onGiveawayVoteQuotas 按目标玩家 ID 批量查询 actor 自己对每个目标各自剩余的投票额度——
// 只回给发起查询的这个连接（额度是"我对每个人还剩多少"，与其他玩家无关，不走广播）。
// 用于自救板列表初次渲染 / 目标集合变化时补齐每张卡片下方的额度展示。
func (s *Server) onGiveawayVoteQuotas(client *Client, env wsEnvelope) {
	var p struct {
		TargetIDs []string `json:"targetIds"`
	}
	_ = decodeD(env, &p)
	actor := s.getPlayerByClient(client)
	if actor == nil {
		client.reply(env.ID, nil, "请先进入游戏")
		return
	}
	now := nowMs()
	quotas := map[string]any{}
	seen := make(map[string]struct{}, len(p.TargetIDs))
	for i, targetID := range p.TargetIDs {
		if i >= 500 {
			break
		}
		if targetID == "" || targetID == actor.ID {
			continue
		}
		if _, ok := seen[targetID]; ok {
			continue
		}
		target := s.players[targetID]
		if target == nil {
			continue
		}
		s.refreshGiveawayBoard(target, now)
		if target.GiveawayBoardText == "" || target.GiveawayBoardExpiresAt == nil || *target.GiveawayBoardExpiresAt <= now {
			continue
		}
		seen[targetID] = struct{}{}
		quotas[targetID] = s.giveawayVoteQuotaView(actor, target, now)
	}
	client.reply(env.ID, map[string]any{"quotas": quotas}, "")
}

func (s *Server) onRankMultiplierUnlock(client *Client, env wsEnvelope) {
	player, ok := s.requirePlayerInGame(client, env)
	if !ok {
		return
	}
	if ptrBool(player.ExtremeModeEnabled) {
		client.reply(env.ID, nil, "极限模式玩家不能解锁倍率模式")
		return
	}
	if ptrBool(player.RankMultiplierUnlocked) {
		client.reply(env.ID, map[string]any{"player": s.publicPlayer(player)}, "")
		return
	}
	if player.Stats.RankedPoints < 200 {
		client.reply(env.ID, nil, "需要至少 200 排位积分才能解锁倍率模式")
		return
	}
	s.updateRankedPoints(player, -200)
	player.RankMultiplierUnlocked = boolPtr(true)
	s.refreshPlayerSnapshots(player)
	s.broadcastPlayerUpdate(player)
	client.reply(env.ID, map[string]any{"player": s.publicPlayer(player)}, "")
	if player.RoomID != "" {
		s.broadcastRoom(player.RoomID, false)
	}
}

func (s *Server) onExtremeForceClose(client *Client, env wsEnvelope) {
	player, ok := s.requirePlayerInGame(client, env)
	if !ok {
		return
	}
	if !ptrBool(player.ExtremeModeEnabled) {
		client.reply(env.ID, nil, "你还没有开启极限模式")
		return
	}
	now := nowMs()
	player.ExtremeModeEnabled = boolPtr(false)
	player.ExtremeModeToggledAt = int64Ptr(now)
	player.ExtremeModeCooldownUntil = int64Ptr(now + int64(s.cfg.ExtremeMode.CooldownHours)*3_600_000)
	player.ExtremeWinStreak = intPtr(0)
	player.ExtremeForceClosed = boolPtr(true)
	player.ExtremeForceClosedAt = int64Ptr(now)
	player.DisplayName = s.formatDisplayName(player)
	s.refreshPlayerSnapshots(player)
	s.broadcastPlayerUpdate(player)
	client.reply(env.ID, map[string]any{"player": s.publicPlayer(player)}, "")
	if player.RoomID != "" {
		s.broadcastRoom(player.RoomID, false)
	}
}

func (s *Server) onNameWarRenameTarget(client *Client, env wsEnvelope) {
	var p struct {
		TargetID string `json:"targetId"`
		Name     string `json:"name"`
		Kind     string `json:"kind"`
	}
	_ = decodeD(env, &p)
	actor := s.getPlayerByClient(client)
	target := s.players[p.TargetID]
	if actor == nil {
		client.reply(env.ID, nil, "请先进入游戏")
		return
	}
	if target == nil {
		client.reply(env.ID, nil, "改名目标不存在")
		return
	}
	s.refreshNameWarState(target, nowMs())
	if actor.ID == target.ID {
		client.reply(env.ID, nil, "不能修改自己的名字")
		return
	}
	now := nowMs()
	cleanName := cleanText(p.Name, 12)
	if len([]rune(cleanName)) < 2 {
		client.reply(env.ID, nil, "新名字至少需要 2 个字")
		return
	}
	renameKind := p.Kind
	if renameKind == "" {
		if s.isNameWarRenameTarget(s.publicPlayer(target)) {
			renameKind = "nameWar"
		} else if ptrBool(target.ExtremeForceClosed) {
			renameKind = "extreme"
		} else {
			renameKind = "nameWar"
		}
	}
	if renameKind == "extreme" {
		if !ptrBool(target.ExtremeForceClosed) {
			client.reply(env.ID, nil, "对方不是极限强关可改名目标")
			return
		}
		if !ptrBool(actor.ExtremeModeEnabled) {
			client.reply(env.ID, nil, "只有开启极限模式的玩家可以修改极限强关目标")
			return
		}
		minPoints := s.cfg.ExtremeMode.ForceRenameMinPoints
		if minPoints < 1 {
			minPoints = 1
		}
		if actor.Stats.RankedPoints < minPoints {
			client.reply(env.ID, nil, fmt.Sprintf("需要极限模式且至少 %d 分才能修改极限强关目标", minPoints))
			return
		}
		if target.ExtremeRenameProtectedUntil != nil && *target.ExtremeRenameProtectedUntil > now {
			hours := int(math.Ceil(float64(*target.ExtremeRenameProtectedUntil-now) / 3_600_000))
			client.reply(env.ID, nil, fmt.Sprintf("对方正在极限改名保护期内，请 %d 小时后再试", hours))
			return
		}
		target.Name = cleanName
		target.NameWarOriginalName = cleanName
		hours := s.cfg.ExtremeMode.ForceRenameProtectHours
		if hours < 1 {
			hours = 4
		}
		target.ExtremeRenameProtectedUntil = int64Ptr(now + int64(hours)*3_600_000)
		target.ExtremeRenamedBy = actor.ID
		target.ExtremeRenamedByName = playerShortName(actor)
		s.refreshNameWarState(target, now)
		target.DisplayName = s.formatDisplayName(target)
		s.refreshPlayerSnapshots(target)
		s.broadcastPlayerUpdate(target)
		client.reply(env.ID, map[string]any{"ok": true}, "")
		if target.RoomID != "" {
			s.broadcastRoom(target.RoomID, false)
		}
		if actor.RoomID != "" && actor.RoomID != target.RoomID {
			s.broadcastRoom(actor.RoomID, false)
		}
		return
	}
	renameMinPoints := s.nameWarRenameMinPoints()
	if actor.Stats.RankedPoints < renameMinPoints {
		client.reply(env.ID, nil, fmt.Sprintf("需要 %d 分以上才能修改失格者名字", renameMinPoints))
		return
	}
	if !s.isNameWarRenameTarget(s.publicPlayer(target)) {
		client.reply(env.ID, nil, "对方当前不是可改名失格者")
		return
	}
	if target.NameWarRenameProtectedUntil != nil && *target.NameWarRenameProtectedUntil > now {
		hours := int(math.Ceil(float64(*target.NameWarRenameProtectedUntil-now) / 3_600_000))
		client.reply(env.ID, nil, fmt.Sprintf("对方正在保护期内，请 %d 小时后再试", hours))
		return
	}
	if s.nameWarRenameQuota(actor, now) <= 0 {
		client.reply(env.ID, nil, "你 3 小时内已经修改了 3 个名字")
		return
	}
	oldDisplayName := target.DisplayName
	target.NameWarPenaltyName = cleanName
	target.NameWarPunished = boolPtr(true)
	target.NameWarRenameProtectedUntil = int64Ptr(now + 21_600_000)
	target.NameWarRenamedBy = actor.ID
	target.NameWarRenamedByName = playerShortName(actor)
	actor.NameWarRenameCount = intPtr(ptrInt(actor.NameWarRenameCount) + 1)
	target.DisplayName = s.formatDisplayName(target)
	s.logPlayerActivity("nameWar_rename", target.ID, cleanName, oldDisplayName, client.ipAddress, client.deviceKey, client.fingerprint, "")
	s.refreshPlayerSnapshots(target)
	s.broadcastPlayerUpdate(target)
	client.reply(env.ID, map[string]any{"ok": true}, "")
	if target.RoomID != "" {
		s.broadcastRoom(target.RoomID, false)
	}
	if actor.RoomID != "" && actor.RoomID != target.RoomID {
		s.broadcastRoom(actor.RoomID, false)
	}
}

func (s *Server) onConfigGet(client *Client, env wsEnvelope) {
	var p struct {
		Password string `json:"password"`
	}
	_ = decodeD(env, &p)
	if !s.adminPasswordMatches(p.Password, client.ipAddress) {
		client.reply(env.ID, nil, "管理员口令不正确或尚未设置")
		return
	}
	client.reply(env.ID, map[string]any{"config": s.publicConfig()}, "")
}

func (s *Server) onConfigSave(client *Client, env wsEnvelope) {
	var p struct {
		Password   string          `json:"password"`
		NextConfig json.RawMessage `json:"nextConfig"`
	}
	_ = decodeD(env, &p)
	if !s.adminPasswordMatches(p.Password, client.ipAddress) {
		client.reply(env.ID, nil, "管理员口令不正确或尚未设置")
		return
	}
	var next types.AppConfig
	if err := json.Unmarshal(p.NextConfig, &next); err != nil {
		client.reply(env.ID, nil, "配置保存失败")
		return
	}
	if strings.TrimSpace(next.Site.AdminPassword) == "" {
		next.Site.AdminPassword = s.cfg.Site.AdminPassword
	}
	// 性别与阵营权威在 SQLite，不在 config/json 里；后台「性别与阵营」板块与其它板块共用
	// 同一份底部「保存配置」，这里把 draft 里的编辑落库，而不是像早期版本那样在保存前就
	// 用数据库当前值覆盖掉 next——那样做会导致这个板块的改动无论怎么点保存都不会生效。
	// 客户端没带这两个字段（旧工具/部分请求只提交子集）时保留数据库当前值，不清空整表。
	if len(next.Genders) > 0 || len(next.GenderFactions) > 0 {
		genders, factions, err := s.sanitizeGenderInput(next.Genders, next.GenderFactions)
		if err != nil {
			client.reply(env.ID, nil, err.Error())
			return
		}
		next.Genders, next.GenderFactions = genders, factions
	} else {
		next.Genders, next.GenderFactions = s.cfg.Genders, s.cfg.GenderFactions
	}
	oldTags := append([]types.PunishmentTagConfig(nil), s.cfg.PunishmentTags...)
	valid, err := config.SaveConfig(next)
	if err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	if err := s.persistGendersAndFactions(next.Genders, next.GenderFactions); err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	if err := s.loadGendersIntoConfig(&valid); err != nil {
		client.reply(env.ID, nil, "读取性别与阵营失败")
		return
	}
	s.cfg = valid
	// 删除标签时从任务池 tagIds 级联摘除（不阻断）。
	s.cascadeRemovedTagsFromTasks(oldTags, valid.PunishmentTags)
	s.refreshAllPlayersForConfig()
	client.reply(env.ID, map[string]any{"config": s.publicConfig()}, "")
	s.broadcastLobby()
	s.emitConfigUpdate()
}

func (s *Server) onConfigReset(client *Client, env wsEnvelope) {
	var p struct {
		Password string `json:"password"`
	}
	_ = decodeD(env, &p)
	if !s.adminPasswordMatches(p.Password, client.ipAddress) {
		client.reply(env.ID, nil, "管理员口令不正确或尚未设置")
		return
	}
	valid, err := config.ResetConfig()
	if err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	if err := s.loadGendersIntoConfig(&valid); err != nil {
		client.reply(env.ID, nil, "读取性别与阵营失败")
		return
	}
	s.cfg = valid
	s.refreshAllPlayersForConfig()
	client.reply(env.ID, map[string]any{"config": s.publicConfig()}, "")
	s.broadcastLobby()
	s.emitConfigUpdate()
}
