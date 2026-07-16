package server

// push:subscribe / push:unsubscribe：管理这台设备的 Web Push 订阅（浏览器 PushManager
// 标准返回的 endpoint + keys，原样转发给服务端登记）。
// push:updatePreferences / push:getPreferences：三个推送来源（@ 我 / 轮到我 / 参战席变化）
// 的开关，私有偏好，只回给发起请求的这条 socket，不进 PublicPlayer。

func (s *Server) onPushSubscribe(client *Client, env wsEnvelope) {
	var p struct {
		Endpoint string `json:"endpoint"`
		Keys     struct {
			P256dh string `json:"p256dh"`
			Auth   string `json:"auth"`
		} `json:"keys"`
	}
	_ = decodeD(env, &p)
	player, ok := s.requirePlayer(client, env)
	if !ok {
		return
	}
	if p.Endpoint == "" || p.Keys.P256dh == "" || p.Keys.Auth == "" {
		client.reply(env.ID, nil, "订阅信息不完整")
		return
	}
	if s.pushDB == nil {
		client.reply(env.ID, nil, "推送功能当前不可用")
		return
	}
	if err := s.pushDB.upsertSubscription(player.ID, p.Endpoint, p.Keys.P256dh, p.Keys.Auth, nowMs()); err != nil {
		client.reply(env.ID, nil, "保存订阅失败")
		return
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onPushUnsubscribe(client *Client, env wsEnvelope) {
	var p struct {
		Endpoint string `json:"endpoint"`
	}
	_ = decodeD(env, &p)
	if _, ok := s.requirePlayer(client, env); !ok {
		return
	}
	if s.pushDB != nil && p.Endpoint != "" {
		_ = s.pushDB.removeSubscription(p.Endpoint)
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onPushGetPreferences(client *Client, env wsEnvelope) {
	player, ok := s.requirePlayer(client, env)
	if !ok {
		return
	}
	client.reply(env.ID, map[string]any{
		"mentionEnabled": ptrBool(player.PushMentionEnabled),
		"turnEnabled":    ptrBool(player.PushTurnEnabled),
		"seatEnabled":    ptrBool(player.PushSeatEnabled),
	}, "")
}

func (s *Server) onPushUpdatePreferences(client *Client, env wsEnvelope) {
	var p struct {
		MentionEnabled *bool `json:"mentionEnabled"`
		TurnEnabled    *bool `json:"turnEnabled"`
		SeatEnabled    *bool `json:"seatEnabled"`
	}
	_ = decodeD(env, &p)
	player, ok := s.requirePlayer(client, env)
	if !ok {
		return
	}
	if p.MentionEnabled != nil {
		player.PushMentionEnabled = boolPtr(*p.MentionEnabled)
	}
	if p.TurnEnabled != nil {
		player.PushTurnEnabled = boolPtr(*p.TurnEnabled)
	}
	if p.SeatEnabled != nil {
		player.PushSeatEnabled = boolPtr(*p.SeatEnabled)
	}
	if player.Persistent {
		s.requestPersist("lazy")
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
}
