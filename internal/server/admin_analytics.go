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
	client.reply(env.ID, snap.forRange(clampAnalyticsDays(q.Days)), "")
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
