package server

// onAdminAnalytics 返回近 N 天聚合快照。全程零 SQL——只读 atomic.Pointer。
func (s *Server) onAdminAnalytics(client *Client, env wsEnvelope) {
	if !s.requireAdminSocket(client, env) {
		return
	}
	var q struct {
		Days    int  `json:"days"`
		Refresh bool `json:"refresh"`
	}
	_ = decodeD(env, &q)
	snap := s.analyticsSnap.Load()
	if snap == nil {
		client.reply(env.ID, nil, "统计数据尚未就绪，请稍后重试")
		return
	}
	if q.Refresh {
		select {
		case s.analyticsKick <- struct{}{}:
		default:
		}
	}
	days := clampAnalyticsDays(q.Days)
	// 快照发布时已经把三档时间范围各编好一份（见 analyticsSnapshot.encoded）：这里只查表，
	// 不在全局锁里重新做那 ~2.3ms 的 Struct 编码。取不到（编码曾失败）才退回即时编码。
	if body := snap.encoded[days]; body != nil {
		client.replyRaw(env.ID, body)
		return
	}
	client.reply(env.ID, snap.forRange(days), "")
}

// onAdminAnalyticsDetail 返回明细表 / 最近会话等。
func (s *Server) onAdminAnalyticsDetail(client *Client, env wsEnvelope) {
	if !s.requireAdminSocket(client, env) {
		return
	}
	snap := s.analyticsSnap.Load()
	if snap == nil {
		client.reply(env.ID, nil, "统计数据尚未就绪，请稍后重试")
		return
	}
	var q struct {
		Days int `json:"days"`
	}
	_ = decodeD(env, &q)
	days := clampAnalyticsDays(q.Days)
	view := snap.forRange(days)
	client.reply(env.ID, map[string]any{
		"days":           days,
		"builtAt":        snap.BuiltAt,
		"viewPv":         view.ViewPV,
		"referrers":      view.Referrers,
		"provinces":      view.Provinces,
		"recentSessions": snap.RecentSessions,
		"retention":      view.Retention,
	}, "")
}
