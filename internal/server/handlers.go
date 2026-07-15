package server

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/doumiao/newRPS/internal/config"
	"github.com/doumiao/newRPS/internal/types"
)

// playersLogHeader 是 players.csv 的固定表头：同一张表的所有写入点必须用同一份
// header（列数/顺序一致），否则不同 action 写出的行会错位。
// oldName 只有 rename 会填；text 只有 giveaway_board_submit 会填（自救板内容）。
var playersLogHeader = []string{"time", "action", "playerId", "name", "oldName", "ip", "device", "fingerprint", "text"}

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
	case "giveaway:boost":
		return RateLimitOptions{20, 60_000, 30_000}, s.onGiveawayBoost
	case "giveaway:submitBoard":
		return RateLimitOptions{4, 60_000, 60_000}, s.onGiveawaySubmitBoard
	case "giveaway:vote":
		return RateLimitOptions{30, 60_000, 30_000}, s.onGiveawayVote
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
	case "othello:ready":
		return RateLimitOptions{12, 60_000, 30_000}, s.onOthelloReady
	case "othello:move":
		return RateLimitOptions{30, 10_000, 15_000}, s.onOthelloMove
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
	case "admin:action":
		return RateLimitOptions{30, 60_000, 60_000}, s.onAdminAction
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
	device := client.deviceKey
	var player *PlayerState
	if p.PlayerID != "" {
		if id := s.playerIdToID[p.PlayerID]; id != "" {
			player = s.players[id]
		}
	} else {
		player = s.players[sid]
	}
	if player != nil && player.Persistent {
		if p.PlayerSecret == "" || player.PlayerSecretHash != hashSecret(p.PlayerSecret) {
			s.securityLog("player_identity_invalid", map[string]any{"sid": sid, "ip": ipAddress, "device": device, "userAgent": client.userAgent})
			client.reply(env.ID, nil, "玩家身份校验失败")
			return
		}
	}
	if player == nil {
		if s.onlinePlayersFromDevice(device, "") >= s.cfg.AccessControl.MaxOnlinePerIP {
			client.reply(env.ID, nil, fmt.Sprintf("当前设备同时在线人数过多，最多允许 %d 人同时在线", s.cfg.AccessControl.MaxOnlinePerIP))
			return
		}
		if !s.canCreateFromDevice(device) {
			client.reply(env.ID, nil, fmt.Sprintf("当前设备 10 分钟内新建玩家过多，最多允许 %d 次", s.cfg.AccessControl.MaxCreatesPer10Min))
			return
		}
		player = s.createPlayer(cleanName, p.GenderID, client.token, p.PlayerID, p.PlayerSecret)
		s.activityLog("players", playersLogHeader, []string{
			time.Now().Format(time.RFC3339), "create", player.ID, cleanName, "", ipAddress, device, client.fingerprint, "",
		})
	}
	wasDisconnected := !player.Connected
	hadDisconnectHold := player.DisconnectExpiresAt != nil
	previousSocketID := player.SocketID
	previousRoomID := player.RoomID
	if previousSocketID != "" && previousSocketID != client.id {
		if prev := s.clients[previousSocketID]; prev != nil {
			prev.leaveRoom(player.ID)
			if previousRoomID != "" {
				prev.leaveRoom(previousRoomID)
			}
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
	client.joinRoom(player.ID)
	if player.RoomID != "" {
		client.leaveRoom(lobbyChannel)
		client.joinRoom(player.RoomID)
	} else {
		client.joinRoom(lobbyChannel)
		client.joinRoom(lobbySuggestionChannel)
	}
	var roomSnap any
	if player.RoomID != "" {
		if room := s.rooms[player.RoomID]; room != nil {
			roomSnap = s.roomSnapshot(room, true, true)
		}
	}
	client.reply(env.ID, map[string]any{
		"player": s.publicPlayer(player),
		"token":  sessionToken,
		"roomId": player.RoomID,
		"room":   roomSnap,
	}, "")
	if player.Persistent {
		s.requestPersist("lazy")
	}
	// 新上线/重连：玩家列表 + 在线人数都要立刻推给大厅（防抖增量容易被其它客户端漏掉 +1）
	s.broadcastPlayerUpdate(player)
	if wasDisconnected || previousSocketID == "" {
		s.forceBroadcastLobby()
	} else {
		s.broadcastLobby()
	}
	if player.RoomID != "" {
		if room := s.rooms[player.RoomID]; room != nil && room.Phase == types.PhasePunishment && hadDisconnectHold {
			s.roomNotice(room, playerShortName(player)+" 已重新连接，恢复到未完成的惩罚房间。")
		}
		s.broadcastRoom(player.RoomID, false)
	}
}

func (s *Server) onAdminLogin(client *Client, env wsEnvelope) {
	var p struct {
		Password string `json:"password"`
	}
	_ = decodeD(env, &p)
	player := s.getPlayerByClientID(client.id)
	if !s.adminPasswordMatches(p.Password) {
		client.reply(env.ID, nil, "管理员口令不正确或尚未设置")
		return
	}
	s.adminClientIDs[client.id] = struct{}{}
	if player != nil {
		player.IsAdmin = boolPtr(true)
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastLobby()
}

func (s *Server) onLobbySubscribe(client *Client, env wsEnvelope) {
	client.joinRoom(lobbyChannel)
	client.joinRoom(lobbySuggestionChannel)
	s.sendFullChannel(client, channelLobby())
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onLobbyUnsubscribe(client *Client, env wsEnvelope) {
	client.leaveRoom(lobbyChannel)
	client.leaveRoom(lobbySuggestionChannel)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

// onLobbySuggestionsSubscribe：仅加入大厅聊天实时频道（房间内的「大厅」tab 用），
// 历史消息由前端另外调 chat:load 拉取。
func (s *Server) onLobbySuggestionsSubscribe(client *Client, env wsEnvelope) {
	client.joinRoom(lobbySuggestionChannel)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onLobbySuggestionsUnsubscribe(client *Client, env wsEnvelope) {
	client.leaveRoom(lobbySuggestionChannel)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onPlayerUpdateProfile(client *Client, env wsEnvelope) {
	var p struct {
		Name              string `json:"name"`
		GenderID          string `json:"genderId"`
		NameWarEnabled    *bool  `json:"nameWarEnabled"`
		NameWarAllowRename *bool `json:"nameWarAllowRename"`
		GiveawayEnabled   *bool  `json:"giveawayEnabled"`
		ExtremeModeEnabled *bool `json:"extremeModeEnabled"`
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
		s.activityLog("players", playersLogHeader, []string{
			time.Now().Format(time.RFC3339), "rename", player.ID, cleanName, oldName,
			client.ipAddress, client.deviceKey, client.fingerprint, "",
		})
	}
	exitedHardMode := ptrBool(player.NameWarAllowRename) && !nextAllowRename
	if nameWarChanged || allowRenameChanged {
		player.NameWarEnabled = boolPtr(nextNameWarEnabled)
		player.NameWarAllowRename = boolPtr(nextAllowRename)
		player.NameWarToggledAt = int64Ptr(now)
		if !nextNameWarEnabled {
			player.Stats.RankedPoints = clamp(player.Stats.RankedPoints, -999, 999)
			s.syncTitleForRankSegment(player, false)
		}
		if nameWarChanged {
			action := "nameWar_disable"
			if nextNameWarEnabled {
				action = "nameWar_enable"
			}
			s.activityLog("players", playersLogHeader, []string{
				time.Now().Format(time.RFC3339), action, player.ID, player.Name, "",
				client.ipAddress, client.deviceKey, client.fingerprint, "",
			})
		}
	}
	s.applyGender(player, p.GenderID)
	s.refreshNameWarState(player, now)
	if exitedHardMode {
		player.Stats.Title = s.cfg.NameWar.EscapeTitle
		if player.Stats.Title == "" {
			player.Stats.Title = "逃跑的人"
		}
	}
	player.GiveawayEnabled = boolPtr(nextGiveawayEnabled)
	if giveawayChanged {
		action := "giveaway_disable"
		if nextGiveawayEnabled {
			action = "giveaway_enable"
		}
		s.activityLog("players", playersLogHeader, []string{
			time.Now().Format(time.RFC3339), action, player.ID, player.Name, "",
			client.ipAddress, client.deviceKey, client.fingerprint, "",
		})
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
		extremeAction := "extreme_disable"
		if nextExtremeModeEnabled {
			extremeAction = "extreme_enable"
		}
		s.activityLog("players", playersLogHeader, []string{
			time.Now().Format(time.RFC3339), extremeAction, player.ID, player.Name, "",
			client.ipAddress, client.deviceKey, client.fingerprint, "",
		})
	}
	if nameChanged {
		player.ProfileUpdatedAt = int64Ptr(now)
	}
	player.DisplayName = formatDisplayName(player)
	s.refreshPlayerSnapshots(player)
	s.broadcastPlayerUpdate(player)
	if player.Persistent {
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
	if !s.isHumanVsHumanRoom(room) {
		client.reply(env.ID, nil, "Bot 对战不能使用白给模式")
		return
	}
	if room.Phase == types.PhasePunishment {
		client.reply(env.ID, nil, "惩罚阶段不能增加白给值")
		return
	}
	player.GiveawayClicks = intPtr(ptrInt(player.GiveawayClicks) + 1)
	s.addGiveawayValue(player, 2)
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
	s.activityLog("players", playersLogHeader, []string{
		time.Now().Format(time.RFC3339), "giveaway_board_submit", player.ID, player.Name, "",
		client.ipAddress, client.deviceKey, client.fingerprint, cleanBoardText,
	})
	s.refreshPlayerSnapshots(player)
	s.broadcastPlayerUpdate(player)
	client.reply(env.ID, map[string]any{"player": s.publicPlayer(player)}, "")
}

func (s *Server) onGiveawayVote(client *Client, env wsEnvelope) {
	var p struct {
		TargetID string `json:"targetId"`
		Vote     string `json:"vote"`
	}
	_ = decodeD(env, &p)
	actor := s.getPlayerByClientID(client.id)
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
	if actor.GiveawayVoteWindowStartedAt == nil || now-*actor.GiveawayVoteWindowStartedAt >= 3_600_000 {
		actor.GiveawayVoteWindowStartedAt = int64Ptr(now)
		actor.GiveawayVoteCount = intPtr(0)
		actor.GiveawayVoteLikesThisHour = intPtr(0)
		actor.GiveawayVoteDislikesThisHour = intPtr(0)
	}
	if p.Vote == "like" {
		if ptrInt(actor.GiveawayVoteLikesThisHour) >= 3 {
			client.reply(env.ID, nil, "你本小时点赞降值次数已满")
			return
		}
		actor.GiveawayVoteLikesThisHour = intPtr(ptrInt(actor.GiveawayVoteLikesThisHour) + 1)
		target.GiveawayBoardLikes = intPtr(ptrInt(target.GiveawayBoardLikes) + 1)
		s.addGiveawayValue(target, -1)
		if ptrFloat(target.GiveawayValue) <= 0 {
			target.GiveawayBoardText = ""
			target.GiveawayBoardSubmittedAt = nil
			target.GiveawayBoardExpiresAt = nil
		}
	} else {
		if ptrInt(actor.GiveawayVoteDislikesThisHour) >= 10 {
			client.reply(env.ID, nil, "你本小时倒赞加值次数已满")
			return
		}
		actor.GiveawayVoteDislikesThisHour = intPtr(ptrInt(actor.GiveawayVoteDislikesThisHour) + 1)
		target.GiveawayBoardDislikes = intPtr(ptrInt(target.GiveawayBoardDislikes) + 1)
		s.addGiveawayValue(target, 0.1)
	}
	actor.GiveawayVoteCount = intPtr(ptrInt(actor.GiveawayVoteCount) + 1)
	s.broadcastPlayerUpdate(actor)
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
	player.DisplayName = formatDisplayName(player)
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
	actor := s.getPlayerByClientID(client.id)
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
		if isNameWarRenameTarget(s.publicPlayer(target)) {
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
		target.DisplayName = formatDisplayName(target)
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
	if actor.Stats.RankedPoints < 500 {
		client.reply(env.ID, nil, "需要 500 分以上才能修改失格者名字")
		return
	}
	if !isNameWarRenameTarget(s.publicPlayer(target)) {
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
	target.NameWarPenaltyName = cleanName
	target.NameWarPunished = boolPtr(true)
	target.NameWarRenameProtectedUntil = int64Ptr(now + 21_600_000)
	target.NameWarRenamedBy = actor.ID
	target.NameWarRenamedByName = playerShortName(actor)
	actor.NameWarRenameCount = intPtr(ptrInt(actor.NameWarRenameCount) + 1)
	target.DisplayName = formatDisplayName(target)
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
	if !s.adminPasswordMatches(p.Password) {
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
	if !s.adminPasswordMatches(p.Password) {
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
	valid, err := config.SaveConfig(next)
	if err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	s.cfg = valid
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
	if !s.adminPasswordMatches(p.Password) {
		client.reply(env.ID, nil, "管理员口令不正确或尚未设置")
		return
	}
	valid, err := config.ResetConfig()
	if err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	s.cfg = valid
	s.refreshAllPlayersForConfig()
	client.reply(env.ID, map[string]any{"config": s.publicConfig()}, "")
	s.broadcastLobby()
	s.emitConfigUpdate()
}
