package server

// 房间/玩家前置校验：统一 reply 文案，减少 handlers 里重复的 nil / RoomID 判断。

// requirePlayer 要求 client 已绑定玩家（已进大厅）。失败时已 reply。
func (s *Server) requirePlayer(client *Client, env wsEnvelope) (*PlayerState, bool) {
	player := s.getPlayerByClient(client)
	if player == nil {
		client.reply(env.ID, nil, "请先进入大厅")
		return nil, false
	}
	return player, true
}

// requirePlayerInGame 与 requirePlayer 相同语义，文案为「请先进入游戏」（部分大厅外操作）。
func (s *Server) requirePlayerInGame(client *Client, env wsEnvelope) (*PlayerState, bool) {
	player := s.getPlayerByClient(client)
	if player == nil {
		client.reply(env.ID, nil, "请先进入游戏")
		return nil, false
	}
	return player, true
}

// requireRoomPlayer 要求玩家当前在某个房间内，并返回该房间。失败时已 reply「你不在房间中」。
func (s *Server) requireRoomPlayer(client *Client, env wsEnvelope) (*PlayerState, *RoomState, bool) {
	player := s.getPlayerByClient(client)
	if player == nil || player.RoomID == "" {
		client.reply(env.ID, nil, "你不在房间中")
		return nil, nil, false
	}
	room := s.rooms[player.RoomID]
	if room == nil {
		client.reply(env.ID, nil, "你不在房间中")
		return nil, nil, false
	}
	return player, room, true
}

// requirePlayerQuiet 取玩家；不在房间/未登录时不 reply（用于 leave 等可幂等路径）。
func (s *Server) requirePlayerQuiet(client *Client) *PlayerState {
	return s.getPlayerByClient(client)
}
