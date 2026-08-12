package server

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

// 日聚合 metric 常量（EAV key 见 plan）。
const (
	metricDAU            = "dau"
	metricSessions       = "sessions"
	metricPageviews      = "pageviews"
	metricNewVisitors    = "new_visitors"
	metricAvgSessionMs   = "avg_session_ms"
	metricBounceSessions = "bounce_sessions"
	metricSessionBucket  = "session_bucket"
	metricViewPV         = "view_pv"
	metricViewUV         = "view_uv"
	metricPeakOnline     = "peak_online"
	metricConnections    = "connections"
	metricOnlineMs       = "online_ms"
	metricRetention      = "retention"
	metricBrowser        = "browser"
	metricOS             = "os"
	metricDevice         = "device"
	metricReferrer       = "referrer"
	metricProvince       = "province"
	metricCity           = "city"
	metricISP            = "isp"
	metricRoomCreate     = "room_create"
	metricRoomJoin       = "room_join"
	metricGameRound      = "game_round"
	metricGameResult     = "game_result"
	metricPunishPublish  = "punish_publish"
	metricPunishDone     = "punish_done"
	metricPunishReject   = "punish_reject"
	metricPunishProofMs  = "punish_proof_ms_bucket"
	metricActivity       = "activity"
	metricPetbondNew     = "petbond_new"
	metricPetbondTotal   = "petbond_total"
	metricChatLobby         = "chat_lobby"
	metricChatRoom          = "chat_room"
	metricChatSpeakers      = "chat_speakers"       // 大厅发言去重人数（历史字段名）
	metricChatRoomSpeakers  = "chat_room_speakers"  // 房间发言去重人数
	metricRoomRoundsMax     = "room_rounds_max"     // 当日单房对局数最大值
	metricRoomRoundsAvg     = "room_rounds_avg"     // 当日有对局的房间局数均值（四舍五入为整数）
	metricRegister          = "register"
	metricFunnel            = "funnel"
	metricLoggedDAU         = "logged_dau"
)

// analyticsSnapshot 是聚合器产出的不可变内存快照；RPC 只读它。
type analyticsSnapshot struct {
	BuiltAt int64
	Days    []int64 // 升序，约 400 天
	// 标量按天：与 Days 等长
	DAU, Sessions, Pageviews, NewVisitors, AvgSessionMs, BounceSessions int64Slice
	PeakOnline, Connections, OnlineMs, LoggedDAU, Register              int64Slice
	PunishPublish, PunishDone, PunishReject                             int64Slice
	PetbondNew, PetbondTotal, ChatLobby, ChatRoom, ChatSpeakers         int64Slice
	ChatRoomSpeakers, RoomRoundsMax, RoomRoundsAvg                      int64Slice
	// 分维度：metric -> key -> 按天 values（与 Days 等长，缺日补 0）
	ByKey map[string]map[string]int64Slice // metric -> key -> series
	// 留存矩阵（用户维度）：cohortDay -> offset -> count
	// cohort = 当日新建 players 数（players.created_at）；offset=0 为基数；
	// Dn = 该 cohort 中在 first_day+n 仍有登录活动的 DISTINCT player_id。
	Retention map[int64]map[int]int64
	// 最近会话（脱敏，明细用）
	RecentSessions []analyticsSessionBrief
	// 实时内存值（聚合时复制）
	LiveOnline int
	LiveRooms  int
	LiveBonds  int
}

type int64Slice []int64

type analyticsSessionBrief struct {
	Visitor    string `json:"visitor"`
	PlayerName string `json:"playerName,omitempty"`
	StartedAt  int64  `json:"startedAt"`
	DurationMs int64  `json:"durationMs"`
	Device     string `json:"device"`
	Province   string `json:"province"`
	Browser    string `json:"browser"`
	Pageviews  int64  `json:"pageviews"`
}

// analyticsRangeView 是 forRange 返回给前端的结构。
type analyticsRangeView struct {
	Days       int   `json:"days"`
	BuiltAt    int64 `json:"builtAt"`
	LiveOnline int   `json:"liveOnline"`
	LiveRooms  int   `json:"liveRooms"`
	LiveBonds  int   `json:"liveBonds"`

	// KPI + 环比
	KPI analyticsKPI `json:"kpi"`

	// 趋势
	Series analyticsSeriesBlock `json:"series"`

	// 分布
	Devices        []analyticsBucket `json:"devices"`
	Browsers       []analyticsBucket `json:"browsers"`
	OS             []analyticsBucket `json:"os"`
	Referrers      []analyticsBucket `json:"referrers"`
	Provinces      []analyticsBucket `json:"provinces"`
	ISPs           []analyticsBucket `json:"isps"`
	SessionBuckets []analyticsBucket `json:"sessionBuckets"`
	ViewPV         []analyticsBucket `json:"viewPv"`

	// 游戏
	GameRounds  []analyticsNamedSeries `json:"gameRounds"`
	GameResults []analyticsBucket      `json:"gameResults"`
	RoomCreates []analyticsNamedSeries `json:"roomCreates"`

	// 玩法
	Punishment analyticsPunishmentBlock `json:"punishment"`
	// Activity 保留旧版分析接口的完整活动序列；新面板使用下面两个按语义拆分的字段。
	Activity []analyticsNamedSeries `json:"activity"`
	// ProfileChanges「用户信息变更」：性别/阵营变更、改名、更换头像、自定义称号。
	ProfileChanges []analyticsNamedSeries `json:"profileChanges"`
	// NameWarGiveaway「名争·白给」：开启名争、开启白给、白给留言板、名争改名、开启极限模式。
	NameWarGiveaway []analyticsNamedSeries `json:"nameWarGiveaway"`
	// NewOldUsers「新老用户」：按用户（playerId）维度，区别于设备维度的 NewVsReturning。
	NewOldUsers analyticsNewOldUsers  `json:"newOldUsers"`
	PetBond     analyticsPetBondBlock `json:"petBond"`
	Chat        analyticsChatBlock    `json:"chat"`
	// RoomRounds「单房对局」：当日有对局的房间里，单房局数 max 与均值 avg。
	RoomRounds analyticsRoomRoundsBlock `json:"roomRounds"`

	// 漏斗 / 留存
	Funnel    []analyticsBucket       `json:"funnel"`
	Retention analyticsRetentionBlock `json:"retention"`

	// 新 vs 回访（按设备指纹维度，见 NewOldUsers 的用户维度对照）
	NewVsReturning analyticsNewReturning `json:"newVsReturning"`
}

type analyticsKPI struct {
	DAUV           int64   `json:"dauValue"`
	SessionsV      int64   `json:"sessions"`
	PageviewsV     int64   `json:"pageviews"`
	NewVisitorsV   int64   `json:"newVisitors"`
	AvgSessionMsV  int64   `json:"avgSessionMs"`
	PeakOnlineV    int64   `json:"peakOnline"`
	DeltaDAU       float64 `json:"deltaDau"`
	DeltaSessions  float64 `json:"deltaSessions"`
	DeltaPageviews float64 `json:"deltaPageviews"`
	DeltaNew       float64 `json:"deltaNewVisitors"`
	DeltaAvg       float64 `json:"deltaAvgSessionMs"`
	DeltaPeak      float64 `json:"deltaPeakOnline"`
	SparkDAU       []int64 `json:"sparkDau"`
	SparkSessions  []int64 `json:"sparkSessions"`
	SparkPageviews []int64 `json:"sparkPageviews"`
	SparkNew       []int64 `json:"sparkNewVisitors"`
	SparkAvg       []int64 `json:"sparkAvgSessionMs"`
	SparkPeak      []int64 `json:"sparkPeakOnline"`
}

type analyticsSeriesBlock struct {
	Days        []string `json:"days"` // YYYY-MM-DD
	DAU         []int64  `json:"dau"`
	Sessions    []int64  `json:"sessions"`
	LoggedDAU   []int64  `json:"loggedDau"`
	Pageviews   []int64  `json:"pageviews"`
	NewVisitors []int64  `json:"newVisitors"`
	Returning   []int64  `json:"returning"`
}

type analyticsBucket struct {
	Key   string `json:"key"`
	Value int64  `json:"value"`
}

type analyticsNamedSeries struct {
	Key    string  `json:"key"`
	Values []int64 `json:"values"`
}

type analyticsPunishmentBlock struct {
	Publish  []int64           `json:"publish"`
	Done     []int64           `json:"done"`
	Reject   []int64           `json:"reject"`
	DoneRate float64           `json:"doneRate"`
	ProofMs  []analyticsBucket `json:"proofMs"`
}

type analyticsPetBondBlock struct {
	Total []int64 `json:"total"`
	New   []int64 `json:"new"`
}

type analyticsChatBlock struct {
	Lobby    []int64 `json:"lobby"`
	Room     []int64 `json:"room"`
	// Speakers 大厅发言去重人数（历史字段名；与 SpeakersRoom 对照）。
	Speakers []int64 `json:"speakers"`
	// SpeakersRoom 房间发言去重人数（按 player_id，跨房间合并去重）。
	SpeakersRoom []int64 `json:"speakersRoom"`
}

// analyticsRoomRoundsBlock 当日「有至少一局对局的房间」上的单房局数统计。
type analyticsRoomRoundsBlock struct {
	Max []int64 `json:"max"` // 单房最多几局
	Avg []int64 `json:"avg"` // 各房局数均值（四舍五入整数）
}

type analyticsRetentionBlock struct {
	// 用户留存：矩阵 [cohortIndex][offset] 为留存率 0-100；offsets 0..30。
	// cohort = 当日首次注册（players.created_at）的用户；跨设备合并到同一 player_id。
	Offsets []int       `json:"offsets"`
	Cohorts []string    `json:"cohorts"` // 日标签
	Matrix  [][]float64 `json:"matrix"`
}

type analyticsNewReturning struct {
	New       []int64 `json:"new"`
	Returning []int64 `json:"returning"`
}

// analyticsNewOldUsers 是按注册用户（playerId）维度的新老拆分，与 analyticsNewReturning
// 的设备指纹维度是两套不同口径（同一用户换设备会被设备维度误判为"新"，反之亦然）。
type analyticsNewOldUsers struct {
	New      []int64 `json:"new"`      // 当日新建用户数（= activity.create 当日计数）
	OldLogin []int64 `json:"oldLogin"` // 当日登录的老用户个数 = 当日登录去重用户数 - 当日新建用户数
}

func clampAnalyticsDays(d int) int {
	switch {
	case d <= 7:
		return 7
	case d <= 30:
		return 30
	case d <= 90:
		return 90
	default:
		return 30
	}
}

func (snap *analyticsSnapshot) forRange(days int) *analyticsRangeView {
	if snap == nil || len(snap.Days) == 0 {
		return &analyticsRangeView{Days: days, BuiltAt: 0}
	}
	days = clampAnalyticsDays(days)
	n := len(snap.Days)
	start := n - days
	if start < 0 {
		start = 0
	}
	slice := func(s int64Slice) []int64 {
		if len(s) == 0 {
			out := make([]int64, n-start)
			return out
		}
		if len(s) < n {
			// pad
			padded := make([]int64, n)
			copy(padded[n-len(s):], s)
			s = padded
		}
		return append([]int64(nil), s[start:]...)
	}
	dayLabels := make([]string, 0, n-start)
	for _, d := range snap.Days[start:] {
		dayLabels = append(dayLabels, analyticsDayLabel(d, 0)) // label 用 day 序号反推；offset 在 build 时已本地化
	}
	// day label：day 序号 * 86400000 近似 UTC 日，再按无 offset 显示——聚合写入时 day 已是本地日序号，
	// 反推：ms = day*86400000，格式化为 UTC 日期即可对应本地日。
	for i, d := range snap.Days[start:] {
		dayLabels[i] = analyticsDayLabel(d, 0)
	}

	dau := slice(snap.DAU)
	sessions := slice(snap.Sessions)
	pageviews := slice(snap.Pageviews)
	newVis := slice(snap.NewVisitors)
	avgMs := slice(snap.AvgSessionMs)
	peak := slice(snap.PeakOnline)
	logged := slice(snap.LoggedDAU)

	sum := func(xs []int64) int64 {
		var t int64
		for _, v := range xs {
			t += v
		}
		return t
	}
	avg := func(xs []int64) int64 {
		if len(xs) == 0 {
			return 0
		}
		return sum(xs) / int64(len(xs))
	}
	maxv := func(xs []int64) int64 {
		var m int64
		for _, v := range xs {
			if v > m {
				m = v
			}
		}
		return m
	}
	// 环比：本窗 vs 前一等长窗
	prevStart := start - (n - start)
	if prevStart < 0 {
		prevStart = 0
	}
	prevSlice := func(s int64Slice) []int64 {
		if len(s) < n {
			padded := make([]int64, n)
			copy(padded[n-len(s):], s)
			s = padded
		}
		if prevStart >= start {
			return nil
		}
		return s[prevStart:start]
	}
	delta := func(cur, prev []int64, useAvg, useMax bool) float64 {
		var c, p float64
		if useMax {
			c, p = float64(maxv(cur)), float64(maxv(prev))
		} else if useAvg {
			if len(cur) > 0 {
				c = float64(sum(cur)) / float64(len(cur))
			}
			if len(prev) > 0 {
				p = float64(sum(prev)) / float64(len(prev))
			}
		} else {
			c, p = float64(sum(cur)), float64(sum(prev))
		}
		if p == 0 {
			if c == 0 {
				return 0
			}
			return 1
		}
		return (c - p) / p
	}
	spark := func(xs []int64) []int64 {
		if len(xs) > 12 {
			return xs[len(xs)-12:]
		}
		return xs
	}

	returning := make([]int64, len(dau))
	for i := range dau {
		r := dau[i] - newVis[i]
		if r < 0 {
			r = 0
		}
		returning[i] = r
	}

	// 按用户（playerId）维度的新老拆分：create 当日计数即当日新建用户数（每个 playerId
	// 一生只触发一次 create），loggedDAU 减去它就是当日登录的老用户个数。
	newAccounts := keySeries(snap, metricActivity, "create", start, n)
	oldLogin := make([]int64, len(logged))
	for i := range logged {
		v := logged[i] - newAccounts[i]
		if v < 0 {
			v = 0
		}
		oldLogin[i] = v
	}

	view := &analyticsRangeView{
		Days: days, BuiltAt: snap.BuiltAt,
		LiveOnline: snap.LiveOnline, LiveRooms: snap.LiveRooms, LiveBonds: snap.LiveBonds,
		KPI: analyticsKPI{
			DAUV: avg(dau), SessionsV: sum(sessions), PageviewsV: sum(pageviews),
			NewVisitorsV: sum(newVis), AvgSessionMsV: avg(avgMs), PeakOnlineV: maxv(peak),
			DeltaDAU:       delta(dau, prevSlice(snap.DAU), true, false),
			DeltaSessions:  delta(sessions, prevSlice(snap.Sessions), false, false),
			DeltaPageviews: delta(pageviews, prevSlice(snap.Pageviews), false, false),
			DeltaNew:       delta(newVis, prevSlice(snap.NewVisitors), false, false),
			DeltaAvg:       delta(avgMs, prevSlice(snap.AvgSessionMs), true, false),
			DeltaPeak:      delta(peak, prevSlice(snap.PeakOnline), false, true),
			SparkDAU:       spark(dau), SparkSessions: spark(sessions), SparkPageviews: spark(pageviews),
			SparkNew: spark(newVis), SparkAvg: spark(avgMs), SparkPeak: spark(peak),
		},
		Series: analyticsSeriesBlock{
			Days: dayLabels, DAU: dau, Sessions: sessions, LoggedDAU: logged,
			Pageviews: pageviews, NewVisitors: newVis, Returning: returning,
		},
		NewVsReturning: analyticsNewReturning{New: newVis, Returning: returning},
		Devices:        sumByKey(snap, metricDevice, start, n),
		Browsers:       sumByKey(snap, metricBrowser, start, n),
		OS:             sumByKey(snap, metricOS, start, n),
		Referrers:      sumByKey(snap, metricReferrer, start, n),
		Provinces:      sumByKey(snap, metricProvince, start, n),
		ISPs:           sumByKey(snap, metricISP, start, n),
		SessionBuckets: orderedBuckets(sumByKey(snap, metricSessionBucket, start, n), []string{"10s", "1min", "2min", "5min", "10min", "30min", "60min", "60min+"}),
		ViewPV:         sumByKey(snap, metricViewPV, start, n),
		GameRounds:     namedSeries(snap, metricGameRound, start, n, dayLabels),
		GameResults:    sumByKey(snap, metricGameResult, start, n),
		RoomCreates:    namedSeries(snap, metricRoomCreate, start, n, dayLabels),
		Punishment: analyticsPunishmentBlock{
			Publish: slice(snap.PunishPublish), Done: slice(snap.PunishDone), Reject: slice(snap.PunishReject),
			DoneRate: rate(sum(slice(snap.PunishDone)), sum(slice(snap.PunishPublish))),
			// 证明耗时统计上几乎不出现 >5min；去掉 20min 桶，≥10min 统一归 10min+。
			// 历史 sealed 日若仍有 20min/20min+ 键，经 mergeBucketAlias 并入 10min+ 以免丢数。
			ProofMs: orderedBuckets(
				mergeBucketAlias(sumByKey(snap, metricPunishProofMs, start, n), map[string]string{
					"20min": "10min+", "20min+": "10min+",
				}),
				[]string{"10s", "30s", "1min", "5min", "10min", "10min+"},
			),
		},
		Activity:        namedSeries(snap, metricActivity, start, n, dayLabels),
		ProfileChanges:  namedSeriesForKeys(snap, metricActivity, []string{"gender_change", "rename", "avatar_change", "self_title_change"}, start, n),
		NameWarGiveaway: namedSeriesForKeys(snap, metricActivity, []string{"nameWar_enable", "giveaway_enable", "giveaway_board_submit", "nameWar_rename", "extreme_enable"}, start, n),
		NewOldUsers:     analyticsNewOldUsers{New: newAccounts, OldLogin: oldLogin},
		PetBond: analyticsPetBondBlock{Total: slice(snap.PetbondTotal), New: slice(snap.PetbondNew)},
		Chat: analyticsChatBlock{
			Lobby: slice(snap.ChatLobby), Room: slice(snap.ChatRoom),
			Speakers: slice(snap.ChatSpeakers), SpeakersRoom: slice(snap.ChatRoomSpeakers),
		},
		RoomRounds: analyticsRoomRoundsBlock{
			Max: slice(snap.RoomRoundsMax), Avg: slice(snap.RoomRoundsAvg),
		},
		Funnel:    sumByKey(snap, metricFunnel, start, n),
		Retention: buildRetentionView(snap, start, n),
	}
	// k-匿名：访客数 < 3 的地域/来源桶折进「其他」
	view.Provinces = kAnonymizeBuckets(view.Provinces, 3)
	view.Referrers = kAnonymizeBuckets(view.Referrers, 3)
	view.ISPs = kAnonymizeBuckets(view.ISPs, 3)
	return view
}

func rate(num, den int64) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}

func analyticsDayLabel(day int64, _ int) string {
	// day 是本地日序号 = (ms+offset)/86400000；反推用 day*86400000 的 UTC 日期
	t := time.UnixMilli(day * 86_400_000).UTC()
	return t.Format("2006-01-02")
}

func sumByKey(snap *analyticsSnapshot, metric string, start, n int) []analyticsBucket {
	m := snap.ByKey[metric]
	if m == nil {
		return nil
	}
	type kv struct {
		k string
		v int64
	}
	var list []kv
	for k, series := range m {
		if len(series) < n {
			padded := make([]int64, n)
			copy(padded[n-len(series):], series)
			series = padded
		}
		var total int64
		for _, v := range series[start:] {
			total += v
		}
		if total > 0 {
			list = append(list, kv{k, total})
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].v > list[j].v })
	out := make([]analyticsBucket, 0, len(list))
	for _, x := range list {
		out = append(out, analyticsBucket{Key: x.k, Value: x.v})
	}
	return out
}

// orderedBuckets 只按 order 给定的固定桶重排/补零；不在 order 里的历史遗留桶键
// （比如改颗粒度前 analytics_daily 里存量的旧桶名）直接丢弃，不与当前桶并列显示。
// 若需把旧键并入新键（而非丢弃），先经 mergeBucketAlias 再传入。
func orderedBuckets(buckets []analyticsBucket, order []string) []analyticsBucket {
	idx := map[string]int64{}
	for _, b := range buckets {
		idx[b.Key] = b.Value
	}
	out := make([]analyticsBucket, 0, len(order))
	for _, k := range order {
		out = append(out, analyticsBucket{Key: k, Value: idx[k]})
	}
	return out
}

// mergeBucketAlias 将 alias 中的旧桶键累加到目标键上（同名键合并），用于改桶后读 sealed 历史。
func mergeBucketAlias(buckets []analyticsBucket, alias map[string]string) []analyticsBucket {
	if len(buckets) == 0 || len(alias) == 0 {
		return buckets
	}
	idx := map[string]int64{}
	for _, b := range buckets {
		k := b.Key
		if to, ok := alias[k]; ok {
			k = to
		}
		idx[k] += b.Value
	}
	out := make([]analyticsBucket, 0, len(idx))
	for k, v := range idx {
		out = append(out, analyticsBucket{Key: k, Value: v})
	}
	return out
}

func namedSeries(snap *analyticsSnapshot, metric string, start, n int, _ []string) []analyticsNamedSeries {
	m := snap.ByKey[metric]
	if m == nil {
		return nil
	}
	// 按总量排序取 top
	type kv struct {
		k string
		s int64Slice
		t int64
	}
	var list []kv
	for k, series := range m {
		if len(series) < n {
			padded := make([]int64, n)
			copy(padded[n-len(series):], series)
			series = padded
		}
		var t int64
		for _, v := range series[start:] {
			t += v
		}
		list = append(list, kv{k, series, t})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].t > list[j].t })
	if len(list) > 8 {
		list = list[:8]
	}
	out := make([]analyticsNamedSeries, 0, len(list))
	for _, x := range list {
		out = append(out, analyticsNamedSeries{Key: x.k, Values: append([]int64(nil), x.s[start:]...)})
	}
	return out
}

// keySeries 取某个 metric 下固定一个 key 的按天序列（补零对齐到 [start,n) 窗口）。
// 与 sumByKey/namedSeries「按总量排序取 top」不同，这里保证返回调用方指定的 key，
// 哪怕它总量很小、排不进 top N 也不会被裁掉。
func keySeries(snap *analyticsSnapshot, metric, key string, start, n int) []int64 {
	var series int64Slice
	if m := snap.ByKey[metric]; m != nil {
		series = m[key]
	}
	if len(series) < n {
		padded := make([]int64, n)
		copy(padded[n-len(series):], series)
		series = padded
	}
	return append([]int64(nil), series[start:]...)
}

// namedSeriesForKeys 是 namedSeries 的定制版：返回调用方指定的固定 keys（保持传入顺序），
// 用于把笼统的 metricActivity 拆成几个有语义的分组图表，而不是笼统按总量取 top N。
func namedSeriesForKeys(snap *analyticsSnapshot, metric string, keys []string, start, n int) []analyticsNamedSeries {
	out := make([]analyticsNamedSeries, 0, len(keys))
	for _, k := range keys {
		out = append(out, analyticsNamedSeries{Key: k, Values: keySeries(snap, metric, k, start, n)})
	}
	return out
}

func kAnonymizeBuckets(buckets []analyticsBucket, k int64) []analyticsBucket {
	var other int64
	out := make([]analyticsBucket, 0, len(buckets))
	for _, b := range buckets {
		if b.Value < k {
			other += b.Value
			continue
		}
		out = append(out, b)
	}
	if other > 0 {
		out = append(out, analyticsBucket{Key: "其他", Value: other})
	}
	return out
}

func buildRetentionView(snap *analyticsSnapshot, start, n int) analyticsRetentionBlock {
	// 取窗口内作为 cohort 的日子
	cohorts := snap.Days[start:]
	offsets := make([]int, 31)
	for i := range offsets {
		offsets[i] = i
	}
	labels := make([]string, len(cohorts))
	matrix := make([][]float64, len(cohorts))
	for i, c := range cohorts {
		labels[i] = analyticsDayLabel(c, 0)
		row := make([]float64, 31)
		base := int64(0)
		if m := snap.Retention[c]; m != nil {
			base = m[0]
			for off := 0; off <= 30; off++ {
				if base > 0 {
					row[off] = float64(m[off]) / float64(base) * 100
				}
			}
		}
		matrix[i] = row
	}
	return analyticsRetentionBlock{Offsets: offsets, Cohorts: labels, Matrix: matrix}
}

// runAnalyticsAggregator 后台聚合协程。
func (s *Server) runAnalyticsAggregator() {
	// 启动延迟 10s，让服务先稳住
	time.Sleep(10 * time.Second)
	s.backfillFromLegacyTables()
	s.rebuildUnsealedAndPublish()

	tick := time.NewTicker(60 * time.Second)
	hour := time.NewTicker(time.Hour)
	defer tick.Stop()
	defer hour.Stop()

	for {
		select {
		case <-tick.C:
			s.rebuildUnsealedAndPublish()
		case <-s.analyticsKick:
			s.rebuildUnsealedAndPublish()
		case <-hour.C:
			s.sealAndPrune()
		}
	}
}

func (s *Server) rebuildUnsealedAndPublish() {
	if s.analyticsRO == nil || s.analyticsDB == nil {
		return
	}
	// 复制内存实时值
	s.mu.Lock()
	liveOnline := len(s.clients)
	liveRooms := len(s.rooms)
	liveBonds := len(s.petBonds)
	tz := s.analyticsTZOffsetMin
	s.mu.Unlock()

	today := analyticsDay(nowMs(), tz)
	// 重算今天 + 昨天
	for d := today - 1; d <= today; d++ {
		if d < 0 {
			continue
		}
		rows, err := s.rebuildDay(d, tz)
		if err != nil {
			s.errorLog("analytics_rebuild_day_failed", fmt.Sprintf("day=%d err=%v", d, err))
			continue
		}
		if err := s.analyticsDB.upsertDaily(rows); err != nil {
			s.errorLog("analytics_upsert_daily_failed", err.Error())
		}
	}
	// 最近 30 个 cohort 的用户留存仍会随登录到来而变化；历史 cohort 在
	// 启动全量回填后已成熟，不需要每分钟重复重算。只更新这段活动窗口，
	// 避免把完整 400 天矩阵写回唯一的 SQLite 写连接。
	if err := s.backfillRetention(today-30, today, tz); err != nil {
		s.errorLog("analytics_retention_refresh_failed", err.Error())
	}
	snap, err := s.buildSnapshot(tz, liveOnline, liveRooms, liveBonds)
	if err != nil {
		s.errorLog("analytics_build_snapshot_failed", err.Error())
		return
	}
	s.analyticsSnap.Store(snap)
}

func (s *Server) sealAndPrune() {
	if s.analyticsDB == nil {
		return
	}
	tz := s.analyticsTZOffsetMin
	today := analyticsDay(nowMs(), tz)
	// 把 day < 今天-1 定稿
	if err := s.analyticsDB.sealDays(today - 1); err != nil {
		s.errorLog("analytics_seal_failed", err.Error())
	}
	// 分批 prune
	rawDays := s.analyticsRawRetentionDays
	if rawDays <= 0 {
		rawDays = 90
	}
	eventsBefore := today - int64(rawDays)
	sessionsBefore := today - 180
	for i := 0; i < 20; i++ {
		n, err := s.analyticsDB.pruneRaw(eventsBefore, sessionsBefore, 5000)
		if err != nil {
			s.errorLog("analytics_prune_failed", err.Error())
			break
		}
		if n == 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// rebuildDay 用只读连接扫原料表，产出该日全部 metric 行。
func (s *Server) rebuildDay(day int64, tz int) ([]analyticsDailyRow, error) {
	db := s.analyticsRO
	offsetMs := int64(tz) * 60_000
	dayStart := day*86_400_000 - offsetMs
	dayEnd := dayStart + 86_400_000

	var rows []analyticsDailyRow
	add := func(metric, key string, value int64) {
		rows = append(rows, analyticsDailyRow{Day: day, Metric: metric, Key: key, Value: value})
	}

	// 会话聚合
	// 注意：AVG(duration_ms) 在 SQLite 里永远是 REAL（即便外面包了
	// COALESCE(...,0)），必须用 NullFloat64 接，否则平均值一旦带小数
	// （即字符串表示带 "."），database/sql 的 float64->int64 转换
	// （走的是格式化成字符串再 ParseInt 这条路）就会直接报错，
	// 导致本次 Scan 整体失败、当天全部维度（含来源/省份/ISP）都聚合不出来。
	var dau, loggedDAU, sessions, pageviews, newVis, bounces sql.NullInt64
	var avgMs sql.NullFloat64
	err := db.QueryRow(`
		SELECT COUNT(DISTINCT visitor),
		       COUNT(DISTINCT NULLIF(player_id,'')),
		       COUNT(*),
		       COALESCE(SUM(pageviews),0),
		       COALESCE(SUM(is_new),0),
		       COALESCE(AVG(duration_ms),0),
		       COALESCE(SUM(CASE WHEN pageviews<=1 AND duration_ms<10000 THEN 1 ELSE 0 END),0)
		FROM analytics_sessions WHERE day = ?`, day).Scan(
		&dau, &loggedDAU, &sessions, &pageviews, &newVis, &avgMs, &bounces,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	add(metricDAU, "", dau.Int64)
	add(metricLoggedDAU, "", loggedDAU.Int64)
	add(metricSessions, "", sessions.Int64)
	add(metricPageviews, "", pageviews.Int64)
	add(metricNewVisitors, "", newVis.Int64)
	add(metricAvgSessionMs, "", int64(avgMs.Float64))
	add(metricBounceSessions, "", bounces.Int64)

	// 时长桶
	if err := s.queryKeyCounts(db, `
		SELECT CASE WHEN duration_ms <    10000 THEN '10s'
		            WHEN duration_ms <    60000 THEN '1min'
		            WHEN duration_ms <   120000 THEN '2min'
		            WHEN duration_ms <   300000 THEN '5min'
		            WHEN duration_ms <   600000 THEN '10min'
		            WHEN duration_ms <  1800000 THEN '30min'
		            WHEN duration_ms <  3600000 THEN '60min'
		            ELSE '60min+' END, COUNT(*)
		FROM analytics_sessions WHERE day = ? GROUP BY 1`, day, metricSessionBucket, &rows); err != nil {
		return nil, err
	}
	// 设备/浏览器/OS/来源/地域
	for _, q := range []struct {
		sql    string
		metric string
	}{
		{`SELECT device_type, COUNT(*) FROM analytics_sessions WHERE day=? AND device_type<>'' GROUP BY 1`, metricDevice},
		{`SELECT browser, COUNT(*) FROM analytics_sessions WHERE day=? AND browser<>'' GROUP BY 1`, metricBrowser},
		{`SELECT os, COUNT(*) FROM analytics_sessions WHERE day=? AND os<>'' GROUP BY 1`, metricOS},
		// 来源/地域面板会做 k-匿名，必须按访客 UV 计数；按会话数会让
		// 一个访客开三次会话后错误地脱离匿名阈值。
		{`SELECT referrer, COUNT(DISTINCT visitor) FROM analytics_sessions WHERE day=? AND referrer<>'' GROUP BY 1`, metricReferrer},
		{`SELECT province, COUNT(DISTINCT visitor) FROM analytics_sessions WHERE day=? AND province<>'' GROUP BY 1`, metricProvince},
		{`SELECT city, COUNT(*) FROM analytics_sessions WHERE day=? AND city<>'' GROUP BY 1`, metricCity},
		{`SELECT isp, COUNT(DISTINCT visitor) FROM analytics_sessions WHERE day=? AND isp<>'' GROUP BY 1`, metricISP},
	} {
		if err := s.queryKeyCounts(db, q.sql, day, q.metric, &rows); err != nil {
			return nil, err
		}
	}

	// pageview by view
	if err := s.queryKeyCounts(db, `
		SELECT view, COUNT(*) FROM analytics_events
		WHERE day=? AND name='pageview' AND view<>'' GROUP BY 1`, day, metricViewPV, &rows); err != nil {
		return nil, err
	}
	if err := s.queryKeyCounts(db, `
		SELECT view, COUNT(DISTINCT visitor) FROM analytics_events
		WHERE day=? AND name='pageview' AND view<>'' GROUP BY 1`, day, metricViewUV, &rows); err != nil {
		return nil, err
	}

	// 服务端 game_round：value>0 为一局主记录（参战副记录 value=0 只供漏斗 UV，不计入对局数）
	if err := s.queryKeyCounts(db, `
		SELECT CASE WHEN instr(detail, ':') > 0
		            THEN substr(detail, 1, instr(detail, ':') - 1)
		            ELSE detail END, COUNT(*) FROM analytics_events
		WHERE day=? AND source=1 AND name='game_round' AND value>0 AND detail<>'' GROUP BY 1`, day, metricGameRound, &rows); err != nil {
		return nil, err
	}
	// game_result：detail 已是 gameId:result
	if err := s.queryKeyCounts(db, `
		SELECT detail, COUNT(*) FROM analytics_events
		WHERE day=? AND source=1 AND name='game_round' AND value>0 AND detail LIKE '%:%' GROUP BY 1`, day, metricGameResult, &rows); err != nil {
		return nil, err
	}

	// connection_events 峰值 / 连接数 / 在线时长
	connRows, err := db.Query(`
		SELECT connected_at, disconnected_at FROM connection_events
		WHERE connected_at < ? AND disconnected_at > ?`, dayEnd, dayStart)
	if err != nil {
		return nil, err
	}
	type se struct{ Start, End int64 }
	var spans []se
	var connCount, onlineMs int64
	for connRows.Next() {
		var st, en int64
		if err := connRows.Scan(&st, &en); err != nil {
			connRows.Close()
			return nil, err
		}
		spans = append(spans, se{st, en})
		connCount++
		// 裁剪到日内
		a, b := st, en
		if a < dayStart {
			a = dayStart
		}
		if b > dayEnd {
			b = dayEnd
		}
		if b > a {
			onlineMs += b - a
		}
	}
	connRows.Close()
	add(metricConnections, "", connCount)
	add(metricOnlineMs, "", onlineMs)
	peakSpans := make([]struct{ Start, End int64 }, len(spans))
	for i, sp := range spans {
		peakSpans[i] = struct{ Start, End int64 }{sp.Start, sp.End}
	}
	add(metricPeakOnline, "", int64(sweepPeakOnline(peakSpans)))

	// 若 analytics_sessions 无 DAU，用 connection_events 回填 dau
	if dau.Int64 == 0 && connCount > 0 {
		var cDau sql.NullInt64
		_ = db.QueryRow(`
			SELECT COUNT(DISTINCT COALESCE(NULLIF(player_id,''), socket_id))
			FROM connection_events WHERE connected_at >= ? AND connected_at < ?`, dayStart, dayEnd).Scan(&cDau)
		if cDau.Int64 > 0 {
			// 覆盖前面的 0
			for i := range rows {
				if rows[i].Metric == metricDAU && rows[i].Key == "" {
					rows[i].Value = cDau.Int64
				}
			}
		}
	}

	// room_events
	if err := queryKeyCountsRange(db, `
		SELECT COALESCE(NULLIF(game_id,''),'unknown'), COUNT(*) FROM room_events
		WHERE action='create' AND at >= ? AND at < ? GROUP BY 1`, dayStart, dayEnd, metricRoomCreate, day, &rows); err != nil {
		return nil, err
	}
	if err := queryKeyCountsRange(db, `
		SELECT COALESCE(NULLIF(game_id,''),'unknown'), COUNT(*) FROM room_events
		WHERE action='join' AND at >= ? AND at < ? GROUP BY 1`, dayStart, dayEnd, metricRoomJoin, day, &rows); err != nil {
		return nil, err
	}

	// punishment：status 与 eventstore 一致——assigned / pending / approved / rejected。
	// 「完成」对应审核通过的 approved（不是 done；历史上曾误写成 done 导致完成数恒为 0）。
	var pub, done, rej sql.NullInt64
	_ = db.QueryRow(`
		SELECT COUNT(*),
		       SUM(CASE WHEN status='approved' THEN 1 ELSE 0 END),
		       SUM(CASE WHEN status='rejected' THEN 1 ELSE 0 END)
		FROM punishment_events WHERE task_at >= ? AND task_at < ?`, dayStart, dayEnd).Scan(&pub, &done, &rej)
	add(metricPunishPublish, "", pub.Int64)
	add(metricPunishDone, "", done.Int64)
	add(metricPunishReject, "", rej.Int64)
	if err := queryKeyCountsRange(db, `
		SELECT CASE WHEN (proof_at - task_at) <    10000 THEN '10s'
		            WHEN (proof_at - task_at) <    30000 THEN '30s'
		            WHEN (proof_at - task_at) <    60000 THEN '1min'
		            WHEN (proof_at - task_at) <   300000 THEN '5min'
		            WHEN (proof_at - task_at) <   600000 THEN '10min'
		            ELSE '10min+' END, COUNT(*)
		FROM punishment_events
		WHERE task_at >= ? AND task_at < ? AND proof_at > task_at
		GROUP BY 1`, dayStart, dayEnd, metricPunishProofMs, day, &rows); err != nil {
		return nil, err
	}

	// activity
	if err := queryKeyCountsRange(db, `
		SELECT action, COUNT(*) FROM player_activity_events
		WHERE at >= ? AND at < ? GROUP BY 1`, dayStart, dayEnd, metricActivity, day, &rows); err != nil {
		return nil, err
	}

	// petbond
	var pbNew sql.NullInt64
	_ = db.QueryRow(`SELECT COUNT(*) FROM pet_bonds WHERE created_at >= ? AND created_at < ?`, dayStart, dayEnd).Scan(&pbNew)
	add(metricPetbondNew, "", pbNew.Int64)
	var pbTotal sql.NullInt64
	_ = db.QueryRow(`SELECT COUNT(*) FROM pet_bonds WHERE created_at < ?`, dayEnd).Scan(&pbTotal)
	add(metricPetbondTotal, "", pbTotal.Int64)

	// chat：消息条数 + 发言去重人数（大厅 / 房间分开）
	var lobbyN, lobbySp, roomN, roomSp sql.NullInt64
	_ = db.QueryRow(`SELECT COUNT(*), COUNT(DISTINCT player_id) FROM lobby_messages WHERE at >= ? AND at < ?`, dayStart, dayEnd).Scan(&lobbyN, &lobbySp)
	_ = db.QueryRow(`SELECT COUNT(*), COUNT(DISTINCT NULLIF(player_id,'')) FROM room_messages WHERE at >= ? AND at < ?`, dayStart, dayEnd).Scan(&roomN, &roomSp)
	add(metricChatLobby, "", lobbyN.Int64)
	add(metricChatRoom, "", roomN.Int64)
	add(metricChatSpeakers, "", lobbySp.Int64)
	add(metricChatRoomSpeakers, "", roomSp.Int64)

	// 单房对局 max / avg：依赖 game_round 主记录（value>0）的 view=room_id。
	// 无 room_id 的历史事件不计入分母；当日无带 room_id 的对局时 max/avg 均为 0。
	var roomMax sql.NullInt64
	var roomAvg sql.NullFloat64
	_ = db.QueryRow(`
		SELECT COALESCE(MAX(c), 0), COALESCE(AVG(c), 0) FROM (
			SELECT COUNT(*) AS c FROM analytics_events
			WHERE day=? AND source=1 AND name='game_round' AND value>0 AND view<>''
			GROUP BY view
		)`, day).Scan(&roomMax, &roomAvg)
	add(metricRoomRoundsMax, "", roomMax.Int64)
	add(metricRoomRoundsAvg, "", int64(roomAvg.Float64+0.5))

	// register
	var reg sql.NullInt64
	_ = db.QueryRow(`SELECT COUNT(*) FROM players WHERE created_at >= ? AND created_at < ?`, dayStart, dayEnd).Scan(&reg)
	add(metricRegister, "", reg.Int64)

	// funnel：五层均为设备 UV（DISTINCT visitor）。
	// visit  访问     = 当日会话 DISTINCT visitor
	// lobby  进大厅   = pageview view=lobby 的 DISTINCT visitor
	// room   进房     = create|join 玩家经当日 player_id→visitor 映射后的 DISTINCT visitor
	// round  开局     = game_start 参战玩家映射后的 DISTINCT visitor
	// finish 完成对局 = game_round 参战玩家映射后的 DISTINCT visitor（含 value=0 副记录）
	// player→visitor 来自当日 analytics_sessions / analytics_events 中非空的 (player_id, visitor)。
	add(metricFunnel, "visit", dau.Int64)
	var lobbyUV sql.NullInt64
	_ = db.QueryRow(`
		SELECT COUNT(DISTINCT visitor) FROM analytics_events
		WHERE day=? AND name='pageview' AND view='lobby' AND visitor<>''`, day).Scan(&lobbyUV)
	add(metricFunnel, "lobby", lobbyUV.Int64)
	roomUV, err := funnelDeviceUVFromPlayers(db, day, dayStart, dayEnd, funnelPlayersRoom)
	if err != nil {
		return nil, err
	}
	add(metricFunnel, "room", roomUV)
	startUV, err := funnelDeviceUVFromPlayers(db, day, dayStart, dayEnd, funnelPlayersGameStart)
	if err != nil {
		return nil, err
	}
	add(metricFunnel, "round", startUV)
	finishUV, err := funnelDeviceUVFromPlayers(db, day, dayStart, dayEnd, funnelPlayersGameRound)
	if err != nil {
		return nil, err
	}
	add(metricFunnel, "finish", finishUV)

	// retention：用户 cohort 的 D0 基数 = 当日新建 players。
	// 完整 D0–D30 矩阵由 backfillRetention 按 player_id 活动重算并 UPSERT。
	var cohortBase sql.NullInt64
	_ = db.QueryRow(`SELECT COUNT(*) FROM players WHERE created_at >= ? AND created_at < ?`, dayStart, dayEnd).Scan(&cohortBase)
	add(metricRetention, fmt.Sprintf("%d:0", day), cohortBase.Int64)

	return rows, nil
}

// funnelPlayerSource 漏斗后三层「业务玩家集合」来源。
type funnelPlayerSource int

const (
	funnelPlayersRoom funnelPlayerSource = iota
	funnelPlayersGameStart
	funnelPlayersGameRound
)

// dayPlayerVisitorSQL 当日 player_id → visitor 映射（会话 + 带 visitor 的事件并集）。
const dayPlayerVisitorSQL = `
	SELECT player_id, visitor FROM analytics_sessions
	WHERE day = ? AND player_id <> '' AND visitor <> ''
	UNION
	SELECT player_id, visitor FROM analytics_events
	WHERE day = ? AND player_id <> '' AND visitor <> ''`

// funnelDeviceUVFromPlayers 将业务侧玩家集合经当日 player→visitor 映射折成设备 UV。
func funnelDeviceUVFromPlayers(db *sql.DB, day, dayStart, dayEnd int64, src funnelPlayerSource) (int64, error) {
	var q string
	var args []any
	switch src {
	case funnelPlayersRoom:
		q = `
			SELECT COUNT(DISTINCT pv.visitor) FROM (
				` + dayPlayerVisitorSQL + `
			) pv
			INNER JOIN room_events r ON r.user_id = pv.player_id
			WHERE r.action IN ('create','join') AND r.user_id <> '' AND r.at >= ? AND r.at < ?`
		args = []any{day, day, dayStart, dayEnd}
	case funnelPlayersGameStart:
		q = `
			SELECT COUNT(DISTINCT pv.visitor) FROM (
				` + dayPlayerVisitorSQL + `
			) pv
			INNER JOIN analytics_events e ON e.player_id = pv.player_id
			WHERE e.day = ? AND e.source = 1 AND e.name = 'game_start' AND e.player_id <> ''`
		args = []any{day, day, day}
	case funnelPlayersGameRound:
		// 含 value=0 的参战副记录，才能覆盖双方/全员
		q = `
			SELECT COUNT(DISTINCT pv.visitor) FROM (
				` + dayPlayerVisitorSQL + `
			) pv
			INNER JOIN analytics_events e ON e.player_id = pv.player_id
			WHERE e.day = ? AND e.source = 1 AND e.name = 'game_round' AND e.player_id <> ''`
		args = []any{day, day, day}
	default:
		return 0, nil
	}
	var n sql.NullInt64
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n.Int64, nil
}

func (s *Server) queryKeyCounts(db *sql.DB, query string, day int64, metric string, rows *[]analyticsDailyRow) error {
	rs, err := db.Query(query, day)
	if err != nil {
		return err
	}
	defer rs.Close()
	for rs.Next() {
		var k string
		var v int64
		if err := rs.Scan(&k, &v); err != nil {
			return err
		}
		if k == "" {
			k = "(empty)"
		}
		*rows = append(*rows, analyticsDailyRow{Day: day, Metric: metric, Key: k, Value: v})
	}
	return rs.Err()
}

func queryKeyCountsRange(db *sql.DB, query string, start, end int64, metric string, day int64, rows *[]analyticsDailyRow) error {
	rs, err := db.Query(query, start, end)
	if err != nil {
		return err
	}
	defer rs.Close()
	for rs.Next() {
		var k string
		var v int64
		if err := rs.Scan(&k, &v); err != nil {
			return err
		}
		if k == "" {
			k = "(empty)"
		}
		*rows = append(*rows, analyticsDailyRow{Day: day, Metric: metric, Key: k, Value: v})
	}
	return rs.Err()
}

// sweepPeakOnline 扫描线求最大同时在线。
func sweepPeakOnline(rows []struct{ Start, End int64 }) int {
	type ev struct {
		t int64
		d int
	}
	events := make([]ev, 0, len(rows)*2)
	for _, r := range rows {
		events = append(events, ev{r.Start, 1}, ev{r.End, -1})
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].t != events[j].t {
			return events[i].t < events[j].t
		}
		return events[i].d < events[j].d // 先 -1 再 +1，避免边界虚高
	})
	cur, peak := 0, 0
	for _, e := range events {
		cur += e.d
		if cur > peak {
			peak = cur
		}
	}
	return peak
}

// backfillFromLegacyTables 首次启动：对尚无 analytics_daily 覆盖的历史日做回填。
func (s *Server) backfillFromLegacyTables() {
	if s.analyticsRO == nil || s.analyticsDB == nil {
		return
	}
	tz := s.analyticsTZOffsetMin
	today := analyticsDay(nowMs(), tz)
	// 找最早 connection_events
	var minAt sql.NullInt64
	_ = s.analyticsRO.QueryRow(`SELECT MIN(connected_at) FROM connection_events`).Scan(&minAt)
	startDay := today - 400
	if minAt.Valid {
		d := analyticsDay(minAt.Int64, tz)
		if d < startDay {
			startDay = d
		}
	}
	// 已有 sealed 日跳过：查 analytics_daily 里已有的 day
	existing := map[int64]bool{}
	rs, err := s.analyticsRO.Query(`SELECT DISTINCT day FROM analytics_daily`)
	if err == nil {
		for rs.Next() {
			var d int64
			if rs.Scan(&d) == nil {
				existing[d] = true
			}
		}
		rs.Close()
	}
	for d := startDay; d <= today; d++ {
		if existing[d] && d < today-1 {
			continue // 已有历史日不重算
		}
		rows, err := s.rebuildDay(d, tz)
		if err != nil {
			s.errorLog("analytics_backfill_day_failed", fmt.Sprintf("day=%d err=%v", d, err))
			continue
		}
		// 历史日直接 sealed
		sealed := 0
		if d < today-1 {
			sealed = 1
			for i := range rows {
				rows[i].Sealed = 1
			}
		}
		_ = sealed
		if err := s.analyticsDB.upsertDaily(rows); err != nil {
			s.errorLog("analytics_backfill_upsert_failed", err.Error())
		}
	}
	// 补全用户留存矩阵到 daily。后续每分钟只刷新最近 30 个 cohort。
	if err := s.backfillRetention(startDay, today, tz); err != nil {
		s.errorLog("analytics_retention_backfill_failed", err.Error())
	}
}

// backfillRetention 按 player_id 重算 [startDay, today] 各 cohort 的 D0–D30 留存人数。
//
// 口径：
//   - cohort 日 = players.created_at 折算的本地日（成功注册的用户，不含未同意条款就离开的访客）
//   - Dn 回访 = 该用户在 first_day+n 仍有活动（analytics_sessions.player_id 或
//     connection_events.player_id；注册日本身并入活动，保证 D0=基数）
//   - 多设备认领到同一 player_id 会合并，不会因换端被算成流失
//
// 先删除这些 cohort 日上的旧 retention 行再写入，避免设备口径残留格子或缺失 offset 脏数据。
func (s *Server) backfillRetention(startDay, today int64, tz int) error {
	db := s.analyticsRO
	if db == nil || s.analyticsDB == nil {
		return nil
	}
	if startDay > today {
		return nil
	}
	offsetMs := int64(tz) * 60_000
	// 活动窗口覆盖到 cohort+30（today 的 D30 要读到 today+30）
	actStart := startDay
	actEnd := today + 30

	if err := s.analyticsDB.deleteDailyMetricDays(metricRetention, startDay, today); err != nil {
		return err
	}

	rs, err := db.Query(`
		WITH cohort AS (
			SELECT id AS player_id,
			       (created_at + ?) / 86400000 AS first_day
			FROM players
			WHERE created_at > 0
			  AND (created_at + ?) / 86400000 BETWEEN ? AND ?
		),
		activity AS (
			SELECT player_id, day AS act_day
			FROM analytics_sessions
			WHERE player_id <> ''
			  AND day BETWEEN ? AND ?
			UNION
			SELECT player_id, (connected_at + ?) / 86400000 AS act_day
			FROM connection_events
			WHERE player_id IS NOT NULL AND player_id <> ''
			  AND (connected_at + ?) / 86400000 BETWEEN ? AND ?
			UNION
			SELECT player_id, first_day AS act_day
			FROM cohort
		)
		SELECT c.first_day AS cohort, a.act_day - c.first_day AS offset, COUNT(DISTINCT c.player_id)
		FROM cohort c
		JOIN activity a ON a.player_id = c.player_id
		WHERE a.act_day - c.first_day BETWEEN 0 AND 30
		GROUP BY cohort, offset`,
		offsetMs, offsetMs, startDay, today,
		actStart, actEnd,
		offsetMs, offsetMs, actStart, actEnd,
	)
	if err != nil {
		return err
	}
	defer rs.Close()
	var rows []analyticsDailyRow
	for rs.Next() {
		var cohort, offset, cnt int64
		if err := rs.Scan(&cohort, &offset, &cnt); err != nil {
			return err
		}
		rows = append(rows, analyticsDailyRow{
			Day: cohort, Metric: metricRetention,
			Key: fmt.Sprintf("%d:%d", cohort, offset), Value: cnt,
			Sealed: boolToInt(cohort < today-1),
		})
	}
	if err := rs.Err(); err != nil {
		return err
	}
	if len(rows) > 0 {
		if err := s.analyticsDB.upsertDaily(rows); err != nil {
			return err
		}
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *Server) buildSnapshot(tz, liveOnline, liveRooms, liveBonds int) (*analyticsSnapshot, error) {
	db := s.analyticsRO
	today := analyticsDay(nowMs(), tz)
	fromDay := today - 400
	rs, err := db.Query(`
		SELECT day, metric, key, value FROM analytics_daily
		WHERE day >= ? ORDER BY day`, fromDay)
	if err != nil {
		return nil, err
	}
	defer rs.Close()

	daySet := map[int64]struct{}{}
	type cell struct {
		day    int64
		metric string
		key    string
		value  int64
	}
	var cells []cell
	for rs.Next() {
		var c cell
		if err := rs.Scan(&c.day, &c.metric, &c.key, &c.value); err != nil {
			return nil, err
		}
		daySet[c.day] = struct{}{}
		cells = append(cells, c)
	}
	days := make([]int64, 0, len(daySet))
	for d := range daySet {
		days = append(days, d)
	}
	sort.Slice(days, func(i, j int) bool { return days[i] < days[j] })
	// 保证包含今天
	if len(days) == 0 || days[len(days)-1] < today {
		days = append(days, today)
	}
	idx := map[int64]int{}
	for i, d := range days {
		idx[d] = i
	}
	n := len(days)
	scalar := func() int64Slice { return make(int64Slice, n) }
	snap := &analyticsSnapshot{
		BuiltAt: nowMs(), Days: days,
		DAU: scalar(), Sessions: scalar(), Pageviews: scalar(), NewVisitors: scalar(),
		AvgSessionMs: scalar(), BounceSessions: scalar(), PeakOnline: scalar(),
		Connections: scalar(), OnlineMs: scalar(), LoggedDAU: scalar(), Register: scalar(),
		PunishPublish: scalar(), PunishDone: scalar(), PunishReject: scalar(),
		PetbondNew: scalar(), PetbondTotal: scalar(),
		ChatLobby: scalar(), ChatRoom: scalar(), ChatSpeakers: scalar(),
		ChatRoomSpeakers: scalar(), RoomRoundsMax: scalar(), RoomRoundsAvg: scalar(),
		ByKey:      map[string]map[string]int64Slice{},
		Retention:  map[int64]map[int]int64{},
		LiveOnline: liveOnline, LiveRooms: liveRooms, LiveBonds: liveBonds,
	}
	setScalar := func(series *int64Slice, day, v int64) {
		if i, ok := idx[day]; ok {
			(*series)[i] = v
		}
	}
	setKey := func(metric, key string, day, v int64) {
		if key == "" {
			return
		}
		m := snap.ByKey[metric]
		if m == nil {
			m = map[string]int64Slice{}
			snap.ByKey[metric] = m
		}
		ser := m[key]
		if ser == nil {
			ser = make(int64Slice, n)
			m[key] = ser
		}
		if i, ok := idx[day]; ok {
			ser[i] = v
		}
	}
	for _, c := range cells {
		switch c.metric {
		case metricDAU:
			setScalar(&snap.DAU, c.day, c.value)
		case metricSessions:
			setScalar(&snap.Sessions, c.day, c.value)
		case metricPageviews:
			setScalar(&snap.Pageviews, c.day, c.value)
		case metricNewVisitors:
			setScalar(&snap.NewVisitors, c.day, c.value)
		case metricAvgSessionMs:
			setScalar(&snap.AvgSessionMs, c.day, c.value)
		case metricBounceSessions:
			setScalar(&snap.BounceSessions, c.day, c.value)
		case metricPeakOnline:
			setScalar(&snap.PeakOnline, c.day, c.value)
		case metricConnections:
			setScalar(&snap.Connections, c.day, c.value)
		case metricOnlineMs:
			setScalar(&snap.OnlineMs, c.day, c.value)
		case metricLoggedDAU:
			setScalar(&snap.LoggedDAU, c.day, c.value)
		case metricRegister:
			setScalar(&snap.Register, c.day, c.value)
		case metricPunishPublish:
			setScalar(&snap.PunishPublish, c.day, c.value)
		case metricPunishDone:
			setScalar(&snap.PunishDone, c.day, c.value)
		case metricPunishReject:
			setScalar(&snap.PunishReject, c.day, c.value)
		case metricPetbondNew:
			setScalar(&snap.PetbondNew, c.day, c.value)
		case metricPetbondTotal:
			setScalar(&snap.PetbondTotal, c.day, c.value)
		case metricChatLobby:
			setScalar(&snap.ChatLobby, c.day, c.value)
		case metricChatRoom:
			setScalar(&snap.ChatRoom, c.day, c.value)
		case metricChatSpeakers:
			setScalar(&snap.ChatSpeakers, c.day, c.value)
		case metricChatRoomSpeakers:
			setScalar(&snap.ChatRoomSpeakers, c.day, c.value)
		case metricRoomRoundsMax:
			setScalar(&snap.RoomRoundsMax, c.day, c.value)
		case metricRoomRoundsAvg:
			setScalar(&snap.RoomRoundsAvg, c.day, c.value)
		case metricRetention:
			// key = cohort:offset 或 我们写入时 day=cohort
			parts := strings.Split(c.key, ":")
			if len(parts) == 2 {
				var off int
				fmt.Sscanf(parts[1], "%d", &off)
				if snap.Retention[c.day] == nil {
					snap.Retention[c.day] = map[int]int64{}
				}
				snap.Retention[c.day][off] = c.value
			}
		default:
			// metricGameRound 也走这里：rebuildDay 按 gameId GROUP BY 后每天每个
			// gameId 只产出一行，且 upsertDaily 是覆盖写（PRIMARY KEY (day,metric,key)），
			// 不会有同一 (day,gameId) 重复累加的可能，不需要特殊处理。
			setKey(c.metric, c.key, c.day, c.value)
		}
	}

	// 最近会话
	srs, err := db.Query(`
		SELECT visitor, player_id, started_at, duration_ms, device_type, province, browser, pageviews
		FROM analytics_sessions ORDER BY last_at DESC LIMIT 20`)
	if err == nil {
		defer srs.Close()
		for srs.Next() {
			var b analyticsSessionBrief
			var pid string
			if err := srs.Scan(&b.Visitor, &pid, &b.StartedAt, &b.DurationMs, &b.Device, &b.Province, &b.Browser, &b.Pageviews); err != nil {
				break
			}
			if len(b.Visitor) > 8 {
				b.Visitor = b.Visitor[:8]
			}
			if pid != "" {
				s.mu.Lock()
				if p := s.players[pid]; p != nil {
					b.PlayerName = p.Name
				}
				s.mu.Unlock()
			}
			snap.RecentSessions = append(snap.RecentSessions, b)
		}
	}
	return snap, nil
}
