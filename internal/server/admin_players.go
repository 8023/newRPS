package server

import (
	"sort"

	"github.com/doumiao/newRPS/internal/types"
)

// 管理员用户列表默认/硬上限：防止一次下发数千条把浏览器卡死。
const (
	adminPlayerListDefaultLimit = 200
	adminPlayerListMaxLimit     = 500
	adminPlayerRecentLoginMs    = 7 * 24 * 60 * 60 * 1000 // 7 天
)

// adminPlayerListQuery 用户管理的筛选/排序参数。
// 布尔开关类过滤器：true=应用该条件，false=不应用。
// 排序开关：均关闭时按积分升序；开启则改为对应降序（可叠加）。
type adminPlayerListQuery struct {
	// Online：仅当前在线的玩家。
	Online bool `json:"online"`
	// NameWar：仅「开启名争」的玩家。
	NameWar bool `json:"nameWar"`
	// SortGiveawayDesc：按白给值降序（主序）。
	SortGiveawayDesc bool `json:"sortGiveawayDesc"`
	// SortRankedDesc：按积分降序（主序或白给之后的次序）。
	SortRankedDesc bool `json:"sortRankedDesc"`
	// RecentLogin7d：最近 7 天内有 lastSeen（含当前在线）。
	RecentLogin7d bool `json:"recentLogin7d"`
	// RankedNonZero：积分不为 0。
	RankedNonZero bool `json:"rankedNonZero"`
	// Limit：返回条数上限；0 用默认，超过硬上限会被夹紧。
	Limit int `json:"limit"`
}

type adminPlayerListResult struct {
	Players      []types.PublicPlayer `json:"players"`
	Total        int                  `json:"total"`
	OnlineCount  int                  `json:"onlineCount"`
	OfflineCount int                  `json:"offlineCount"`
	Limit        int                  `json:"limit"`
	Truncated    bool                 `json:"truncated"`
}

type adminPlayerSortItem struct {
	player   *PlayerState
	ranked   int
	giveaway float64
	lastSeen int64
}

func clampAdminPlayerListLimit(limit int) int {
	if limit <= 0 {
		return adminPlayerListDefaultLimit
	}
	if limit > adminPlayerListMaxLimit {
		return adminPlayerListMaxLimit
	}
	return limit
}

func playerGiveawaySortValue(p *PlayerState) float64 {
	if p == nil || !ptrBool(p.GiveawayEnabled) {
		return 0
	}
	return ptrFloat(p.GiveawayValue)
}

// matchAdminPlayerFilters 应用布尔过滤器（不含排序）。
func matchAdminPlayerFilters(p *PlayerState, q adminPlayerListQuery, nowMs int64) bool {
	if p == nil {
		return false
	}
	if q.Online && !p.Connected {
		return false
	}
	if q.NameWar && !ptrBool(p.NameWarEnabled) {
		return false
	}
	if q.RankedNonZero && p.Stats.RankedPoints == 0 {
		return false
	}
	if q.RecentLogin7d {
		// 当前在线视作最近活跃；否则看 lastSeen 是否在 7 天窗口内。
		if !p.Connected {
			if p.LastSeenAt <= 0 || nowMs-p.LastSeenAt > adminPlayerRecentLoginMs {
				return false
			}
		}
	}
	return true
}

// buildAdminPlayerList 在已持 s.mu 的前提下，从内存玩家表筛选/排序并截断。
// 返回的 PublicPlayer 走 publicPlayer（展示分封顶）；排序/过滤用真实存储分。
func (s *Server) buildAdminPlayerList(q adminPlayerListQuery) adminPlayerListResult {
	limit := clampAdminPlayerListLimit(q.Limit)
	now := nowMs()
	items := make([]adminPlayerSortItem, 0, 64)
	onlineCount, offlineCount := 0, 0

	for _, p := range s.players {
		// 在线/离线计数：在其它过滤器命中后、尚未施加「仅在线」前统计，便于前端展示规模。
		qNoOnline := q
		qNoOnline.Online = false
		if !matchAdminPlayerFilters(p, qNoOnline, now) {
			continue
		}
		if p.Connected {
			onlineCount++
		} else {
			offlineCount++
		}
		if q.Online && !p.Connected {
			continue
		}
		items = append(items, adminPlayerSortItem{
			player:   p,
			ranked:   p.Stats.RankedPoints,
			giveaway: playerGiveawaySortValue(p),
			lastSeen: p.LastSeenAt,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		// 主序：白给降序（若启用）
		if q.SortGiveawayDesc {
			if a.giveaway != b.giveaway {
				return a.giveaway > b.giveaway
			}
		}
		// 积分：默认升序；开启「积分降序」则降序
		if a.ranked != b.ranked {
			if q.SortRankedDesc {
				return a.ranked > b.ranked
			}
			return a.ranked < b.ranked
		}
		// 同分：最近登录靠前，再按 id 稳定次序
		if a.lastSeen != b.lastSeen {
			return a.lastSeen > b.lastSeen
		}
		return a.player.ID < b.player.ID
	})

	total := len(items)
	truncated := total > limit
	if truncated {
		items = items[:limit]
	}

	out := make([]types.PublicPlayer, 0, len(items))
	for _, item := range items {
		out = append(out, s.publicPlayer(item.player))
	}
	return adminPlayerListResult{
		Players:      out,
		Total:        total,
		OnlineCount:  onlineCount,
		OfflineCount: offlineCount,
		Limit:        limit,
		Truncated:    truncated,
	}
}

func (s *Server) onAdminListPlayers(client *Client, env wsEnvelope) {
	admin := s.getPlayerByClient(client)
	_, isAdminSocket := s.adminClientIDs[client.id]
	if (admin == nil || !ptrBool(admin.IsAdmin)) && !isAdminSocket {
		client.reply(env.ID, nil, "需要管理员权限")
		return
	}
	var q adminPlayerListQuery
	_ = decodeD(env, &q)
	result := s.buildAdminPlayerList(q)
	client.reply(env.ID, map[string]any{
		"players":      result.Players,
		"total":        result.Total,
		"onlineCount":  result.OnlineCount,
		"offlineCount": result.OfflineCount,
		"limit":        result.Limit,
		"truncated":    result.Truncated,
	}, "")
}
