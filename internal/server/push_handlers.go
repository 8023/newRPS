package server

import "time"

// push:subscribe / push:unsubscribe：管理这台设备的 Web Push 订阅（浏览器 PushManager
// 标准返回的 endpoint + keys，原样转发给服务端登记）。
// push:updatePreferences / push:getPreferences：四个推送来源（@ 我 / 轮到我 / 参战席变化 /
// 我的主人或宠物上线）的开关，私有偏好，只回给发起请求的这条 socket，不进 PublicPlayer。

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
	if !isAllowedPushEndpoint(p.Endpoint) {
		s.securityLog("push_endpoint_rejected", map[string]any{"sid": client.sid, "ip": client.ipAddress, "userAgent": client.userAgent})
		client.reply(env.ID, nil, "不支持的推送服务地址")
		return
	}
	if s.pushDB == nil {
		client.reply(env.ID, nil, "推送功能当前不可用")
		return
	}
	if err := s.pushDB.upsertSubscription(player.ID, p.Endpoint, p.Keys.P256dh, p.Keys.Auth, nowMs()); err != nil {
		s.errorLog("push_subscription_save_failed", err.Error())
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
	player, ok := s.requirePlayer(client, env)
	if !ok {
		return
	}
	if s.pushDB != nil && p.Endpoint != "" {
		if err := s.pushDB.removeSubscriptionForPlayer(player.ID, p.Endpoint); err != nil {
			s.errorLog("push_subscription_remove_failed", err.Error())
			client.reply(env.ID, nil, "删除订阅失败")
			return
		}
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
		"bondEnabled":    ptrBool(player.PushBondEnabled),
	}, "")
}

func (s *Server) onPushUpdatePreferences(client *Client, env wsEnvelope) {
	var p struct {
		MentionEnabled *bool `json:"mentionEnabled"`
		TurnEnabled    *bool `json:"turnEnabled"`
		SeatEnabled    *bool `json:"seatEnabled"`
		BondEnabled    *bool `json:"bondEnabled"`
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
	if p.BondEnabled != nil {
		player.PushBondEnabled = boolPtr(*p.BondEnabled)
	}
	if player.Persistent {
		s.markPlayerDirty(player)
		s.requestPersist("lazy")
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

// onPushTest 发送一条真实 Web Push，供用户验证浏览器订阅、服务器出网和 Service Worker
// 展示的完整链路。调用后需将本站切到后台或锁屏；前台可见页面会由 Service Worker 抑制。
func (s *Server) onPushTest(client *Client, env wsEnvelope) {
	player, ok := s.requirePlayer(client, env)
	if !ok {
		return
	}
	if s.pushDB == nil {
		client.reply(env.ID, nil, "推送数据库当前不可用")
		return
	}
	playerID := player.ID
	go func() {
		subs, err := s.pushDB.subscriptionsForPlayer(playerID)
		if err != nil {
			s.errorLog("push_test_subscription_query_failed", err.Error())
			client.reply(env.ID, nil, "检查推送订阅失败")
			return
		}
		if len(subs) == 0 {
			client.reply(env.ID, nil, "服务端没有本账号的推送订阅，请先重新订阅这台设备")
			return
		}
		timeAfterFunc(3*time.Second, func() {
			result := s.deliverPush(playerID, subs, "推送测试成功", "这台设备已经可以接收抖喵游戏屋通知。", "push-test")
			if errMessage := pushDeliveryClientError(result); errMessage != "" {
				client.reply(env.ID, nil, errMessage)
				return
			}
			client.reply(env.ID, map[string]any{
				"ok":                true,
				"subscriptionCount": len(subs),
				"acceptedCount":     result.Accepted,
				"failedCount":       result.Failed,
				"expiredCount":      result.Expired,
			}, "")
		})
	}()
}
