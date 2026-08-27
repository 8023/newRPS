package server

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

// 白名单：未知维度一律收敛为 "other"，防止 analytics_daily 被撑爆。
var (
	analyticsEventNames = map[string]struct{}{
		"pageview": {}, "session_ping": {}, "login_success": {}, "theme_toggle": {},
		// game_start：对局进入 playing/出拳/行棋；game_round：对局结算（一局一次）。
		"game_start": {}, "game_round": {}, "other": {},
		// punish_tag_include/exclude：建房选随机惩罚任务时勾选/排除的标签（detail=tagId）。
		// punish_series_select：建房选系列惩罚任务时选中的系列（detail=seriesId）。
		"punish_tag_include": {}, "punish_tag_exclude": {}, "punish_series_select": {},
	}
	analyticsViewKeys = map[string]struct{}{
		"login": {}, "lobby": {}, "room": {}, "admin": {},
		"profile": {}, "leaderboard": {}, "about": {}, "help": {}, "other": {},
	}
	// detail 允许更宽松的集合（gameId、弹窗名、结果码等），仍做长度与字符约束。
	analyticsDetailKeys = map[string]struct{}{
		"restore": {}, "manual": {}, "light": {}, "dark": {},
		"rps": {}, "othello": {}, "tictactoe": {}, "gomoku": {}, "jungle": {}, "liarsdice": {}, "chess": {}, "coinflip": {},
		"win": {}, "draw": {}, "doubleloss": {}, "loss": {},
		"other": {},
	}
	analyticsSIDRe  = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
	analyticsHostRe = regexp.MustCompile(`^[a-z0-9.\-:]+$`)
)

// analyticsBatchPayload 是 logCh 与 writeAnalyticsBatch 之间传递的已清洗批次。
type analyticsBatchPayload struct {
	SessionID  string
	Visitor    string
	PlayerID   string
	Browser    string
	OS         string
	DeviceType string
	Referrer   string
	Landing    string
	Country    string
	Province   string
	City       string
	ISP        string
	IsNewSess  bool // 本批是否首次见到该 session（仅入队时估算；落库再校验）
	Events     []analyticsEventRow
	// 会话累加
	PageviewsDelta int
	EventsDelta    int
	StartedAt      int64 // 若为新会话，用事件最小时间
	LastAt         int64
	Day            int64
}

// 前端上报的压缩字段。
type analyticsCollectIn struct {
	Sid  string `json:"sid"`
	Ref  string `json:"ref"`
	Land string `json:"land"`
	Ev   []struct {
		N string  `json:"n"`
		V string  `json:"v"`
		D string  `json:"d"`
		X float64 `json:"x"`
		T float64 `json:"t"`
	} `json:"ev"`
}

// onAnalyticsCollect 全程持 s.mu，只做内存操作 + 非阻塞入队。
func (s *Server) onAnalyticsCollect(client *Client, env wsEnvelope) {
	if !s.analyticsEnabled {
		return
	}
	var p analyticsCollectIn
	if err := decodeD(env, &p); err != nil {
		return
	}
	sid := strings.TrimSpace(p.Sid)
	if len(sid) == 0 || len(sid) > 64 || !analyticsSIDRe.MatchString(sid) {
		return
	}
	if len(p.Ev) == 0 {
		return
	}
	if len(p.Ev) > 40 {
		p.Ev = p.Ev[:40]
	}

	now := nowMs()
	minT := now
	maxT := now
	events := make([]analyticsEventRow, 0, len(p.Ev))
	pageviews := 0
	for _, e := range p.Ev {
		name := whitelistOrOther(strings.TrimSpace(e.N), analyticsEventNames)
		if name == "" {
			name = "other"
		}
		view := whitelistOrOther(strings.TrimSpace(e.V), analyticsViewKeys)
		detail := sanitizeAnalyticsDetail(strings.TrimSpace(e.D))
		// 客户端时间戳只做批内排序并 clamp 到 [now-5min, now]
		t := int64(e.T)
		if t <= 0 {
			t = now
		}
		if t < now-5*60_000 {
			t = now - 5*60_000
		}
		if t > now {
			t = now
		}
		if t < minT {
			minT = t
		}
		if t > maxT {
			maxT = t
		}
		if name == "pageview" {
			pageviews++
		}
		events = append(events, analyticsEventRow{
			At: t, Day: analyticsDay(t, s.analyticsTZOffsetMin),
			Source: 0, SessionID: sid, Visitor: client.anaVisitor,
			PlayerID: client.playerID, Name: name, View: view, Detail: detail,
			Value: int64(e.X),
		})
	}

	landing := whitelistOrOther(strings.TrimSpace(p.Land), analyticsViewKeys)
	if landing == "other" && p.Land == "" {
		landing = ""
	}
	ref := normalizeReferrerHost(p.Ref)

	payload := analyticsBatchPayload{
		SessionID:      sid,
		Visitor:        client.anaVisitor,
		PlayerID:       client.playerID,
		Browser:        client.anaBrowser,
		OS:             client.anaOS,
		DeviceType:     client.anaDevice,
		Referrer:       ref,
		Landing:        landing,
		Country:        client.anaGeo.Country,
		Province:       client.anaGeo.Province,
		City:           client.anaGeo.City,
		ISP:            client.anaGeo.ISP,
		Events:         events,
		PageviewsDelta: pageviews,
		EventsDelta:    len(events),
		StartedAt:      minT,
		LastAt:         maxT,
		Day:            analyticsDay(minT, s.analyticsTZOffsetMin),
	}
	s.logAnalyticsBatch(payload)
	// 成功时不 reply（省一帧；前端无 ack 发送）
}

// logAnalyticsServerEvent 记录服务端侧产生的分析事件（source=1）。
// 调用点都在 s.mu 持锁期间，只做非阻塞入队。
// roomID 写入 analytics_events.view（服务端事件不用 view 做页面路由）：
// 供「单房对局 max/avg」按房间聚合；空串表示未知房间，不参与该聚合。
func (s *Server) logAnalyticsServerEvent(name, detail string, value int64, playerID, roomID string) {
	if !s.analyticsEnabled {
		return
	}
	name = whitelistOrOther(name, analyticsEventNames)
	detail = sanitizeAnalyticsDetail(detail)
	roomID = sanitizeAnalyticsRoomID(roomID)
	now := nowMs()
	payload := analyticsBatchPayload{
		// 无 session/visitor：服务端事件只进 analytics_events
		Events: []analyticsEventRow{{
			At: now, Day: analyticsDay(now, s.analyticsTZOffsetMin),
			Source: 1, PlayerID: playerID, Name: name, Detail: detail, Value: value,
			View: roomID,
		}},
		EventsDelta: 1,
		LastAt:      now,
		Day:         analyticsDay(now, s.analyticsTZOffsetMin),
	}
	s.logAnalyticsBatch(payload)
}

// sanitizeAnalyticsRoomID 约束 room_id 字符与长度，避免脏数据撑爆按 view GROUP BY 的聚合。
// randomID 是 base64.RawURLEncoding，可能含 A-Za-z0-9-_。
func sanitizeAnalyticsRoomID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" || len(id) > 64 {
		return ""
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		ok := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_'
		if !ok {
			return ""
		}
	}
	return id
}

func (s *Server) logAnalyticsBatch(p analyticsBatchPayload) {
	entry := activityLogEntry{table: "analytics", analytics: &p}
	if s.logCh != nil {
		select {
		case s.logCh <- entry:
		default:
			// 队列满：丢弃，绝不阻塞业务锁
		}
		return
	}
	s.writeAnalyticsBatch(&p)
}

// writeAnalyticsBatch 后台协程落库（锁外）。
func (s *Server) writeAnalyticsBatch(p *analyticsBatchPayload) {
	if p == nil || s.analyticsDB == nil {
		return
	}
	// 纯服务端事件：无 session，只插 events
	if p.SessionID == "" {
		if err := s.analyticsDB.insertEvents(p.Events); err != nil {
			s.errorLog("analytics_events_insert_failed", err.Error())
		}
		return
	}
	if p.Visitor == "" {
		// 无 visitor（分析关闭或 salt 未就绪）时仍尽量记 events
		if err := s.analyticsDB.insertEvents(p.Events); err != nil {
			s.errorLog("analytics_events_insert_failed", err.Error())
		}
		return
	}

	exists, err := s.analyticsDB.sessionExists(p.SessionID)
	if err != nil {
		s.errorLog("analytics_session_exists_failed", err.Error())
	}
	visExists, err := s.analyticsDB.visitorExists(p.Visitor)
	if err != nil {
		s.errorLog("analytics_visitor_exists_failed", err.Error())
	}

	isNewVisitor := !visExists
	sessionsDelta := 0
	if !exists {
		sessionsDelta = 1
	}
	isNewSessFlag := 0
	if isNewVisitor && !exists {
		isNewSessFlag = 1
	}

	visitor := analyticsVisitorRow{
		Visitor: p.Visitor, FirstAt: p.StartedAt, FirstDay: p.Day, LastAt: p.LastAt,
		SessionsDelta: sessionsDelta, FirstReferrer: p.Referrer, FirstProvince: p.Province,
	}
	session := analyticsSessionRow{
		ID: p.SessionID, Visitor: p.Visitor, StartedAt: p.StartedAt, LastAt: p.LastAt,
		Day: p.Day, PlayerID: p.PlayerID, IsNew: isNewSessFlag,
		Browser: p.Browser, OS: p.OS, DeviceType: p.DeviceType,
		Referrer: p.Referrer, Landing: p.Landing,
		Country: p.Country, Province: p.Province, City: p.City, ISP: p.ISP,
		PageviewsDelta: p.PageviewsDelta, EventsDelta: p.EventsDelta,
	}
	if err := s.analyticsDB.writeBatch(visitor, !exists, session, p.Events); err != nil {
		s.errorLog("analytics_batch_write_failed", err.Error())
	}
}

// analyticsVisitorID 二次加盐哈希 deviceKey → 16 hex，不可逆。
func analyticsVisitorID(deviceKey string, salt []byte) string {
	if deviceKey == "" || len(salt) == 0 {
		return ""
	}
	h := sha256.New()
	h.Write([]byte(deviceKey))
	h.Write(salt)
	sum := h.Sum(nil)
	return hex.EncodeToString(sum[:8]) // 16 hex chars
}

// analyticsDay 把毫秒时间戳折算为本地日序号。
func analyticsDay(ms int64, tzOffsetMin int) int64 {
	offsetMs := int64(tzOffsetMin) * 60_000
	return (ms + offsetMs) / 86_400_000
}

func whitelistOrOther(v string, set map[string]struct{}) string {
	if v == "" {
		return ""
	}
	if _, ok := set[v]; ok {
		return v
	}
	// 允许 gameId 等已知枚举的大小写变体归一
	lower := strings.ToLower(v)
	if _, ok := set[lower]; ok {
		return lower
	}
	return "other"
}

func sanitizeAnalyticsDetail(d string) string {
	if d == "" {
		return ""
	}
	if _, ok := analyticsDetailKeys[d]; ok {
		return d
	}
	lower := strings.ToLower(d)
	if _, ok := analyticsDetailKeys[lower]; ok {
		return lower
	}
	// 复合 key 如 "gomoku:win" / 弹窗名：限制长度与字符
	if utf8.RuneCountInString(d) > 64 {
		d = string([]rune(d)[:64])
	}
	for _, r := range d {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '_' || r == '-' || r == ':' || r == '.' {
			continue
		}
		return "other"
	}
	return d
}

// normalizeReferrerHost 只保留 host，小写，≤64，非法置 (other)。
func normalizeReferrerHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// 可能是完整 URL 或纯 host
	host := raw
	if strings.Contains(raw, "://") {
		if u, err := url.Parse(raw); err == nil && u.Host != "" {
			host = u.Host
		}
	} else if i := strings.IndexAny(raw, "/?"); i >= 0 {
		host = raw[:i]
	}
	host = strings.ToLower(strings.TrimSpace(host))
	// 去掉端口
	if h, _, err := splitHostPortSafe(host); err == nil {
		host = h
	}
	if len(host) > 64 {
		host = host[:64]
	}
	if host == "" || !analyticsHostRe.MatchString(host) {
		return "(other)"
	}
	return host
}

func splitHostPortSafe(hostport string) (host, port string, err error) {
	// net.SplitHostPort 要求有端口；这里宽松处理
	if i := strings.LastIndex(hostport, ":"); i > 0 && !strings.Contains(hostport[i:], "]") {
		// 简单判断：最后一段全是数字才当端口
		p := hostport[i+1:]
		allDigit := len(p) > 0
		for _, c := range p {
			if c < '0' || c > '9' {
				allDigit = false
				break
			}
		}
		if allDigit {
			return hostport[:i], p, nil
		}
	}
	return hostport, "", nil
}
