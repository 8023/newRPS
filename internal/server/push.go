package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/doumiao/newRPS/internal/types"
)

// allowedPushGatewayHostSuffixes 是各浏览器厂商官方 Web Push 网关的域名（后缀匹配）。
// 客户端上报的 endpoint 完全不可信——deliverPush 会拿着它发起真实的服务端出网 HTTPS 请求，
// 如果不做校验，任何登录用户都能把 endpoint 设成内网/回环地址，把本服务器变成一个
// SSRF 探测器（push:test 还会把 HTTP 状态码原样回显给调用方，等于探测结果 oracle）。
// 这里选择白名单而非黑名单：黑名单式的私网地址过滤存在 DNS rebinding、IPv6 变形写法等
// 绕过手法，只信任已知网关域名更稳妥。
var allowedPushGatewayHostSuffixes = []string{
	"googleapis.com",            // Chrome / Edge 等（FCM）；部分 Samsung Internet 也走 FCM
	"push.services.mozilla.com", // Firefox
	"notify.windows.com",        // 旧版 Edge / Windows
	"push.apple.com",            // Safari（macOS/iOS 16.4+）
	"push.samsungosp.com",       // Galaxy 上 Samsung Internet 自家网关
}

// pushHTTPClient 带超时，避免某个卡住的推送网关把 deliverPush goroutine 挂到 TCP 默认超时。
var pushHTTPClient = &http.Client{Timeout: 10 * time.Second}

// isAllowedPushEndpoint 校验浏览器上报的 PushSubscription.endpoint：必须是合法的 https
// URL，且主机名落在已知推送网关域名白名单内。
func isAllowedPushEndpoint(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Hostname() == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, suffix := range allowedPushGatewayHostSuffixes {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}

// Web Push（Level 2 推送）：不依赖任何外部付费服务，VAPID 密钥对本地生成/落盘，
// 推送网关是浏览器厂商自己的公共端点（Chrome→Google、Firefox→Mozilla…），
// 服务端只是往这个端点发一条签名过的消息。
//
// 只要偏好开启就发送 Web Push，不再用 WebSocket Connected 状态做互斥。手机切后台后
// JavaScript 可能被冻结但连接仍被服务端视为在线；旧互斥会让前后端两条路径同时不发送。
// 前端不再自行创建系统通知，由 Service Worker 统一展示，前台与后台行为保持一致。

const pushProtocolVersion = 3

const pushSubscriptionSchema = `
CREATE TABLE IF NOT EXISTS push_subscriptions (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	player_id  TEXT NOT NULL,
	endpoint   TEXT NOT NULL UNIQUE,
	p256dh     TEXT NOT NULL,
	auth       TEXT NOT NULL,
	created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_push_subscriptions_player ON push_subscriptions(player_id);
`

type pushStore struct {
	db *sql.DB
	mu sync.Mutex
}

func newPushStore(db *sql.DB) *pushStore {
	return &pushStore{db: db}
}

// upsertSubscription 保存/更新一条设备订阅（endpoint 是浏览器分配的唯一地址，天然去重）。
func (p *pushStore) upsertSubscription(playerID, endpoint, p256dh, auth string, createdAt int64) error {
	if p == nil || p.db == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	_, err := p.db.Exec(
		`INSERT INTO push_subscriptions (player_id, endpoint, p256dh, auth, created_at) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(endpoint) DO UPDATE SET player_id = excluded.player_id, p256dh = excluded.p256dh, auth = excluded.auth`,
		playerID, endpoint, p256dh, auth, createdAt,
	)
	return err
}

// removeSubscription 按 endpoint 删除（用于推送网关返回 404/410 时的失效清理，无玩家上下文）。
func (p *pushStore) removeSubscription(endpoint string) error {
	if p == nil || p.db == nil || endpoint == "" {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	_, err := p.db.Exec(`DELETE FROM push_subscriptions WHERE endpoint = ?`, endpoint)
	return err
}

// removeSubscriptionForPlayer 仅当 endpoint 属于该玩家时才删除，防止已登录用户删除他人订阅。
// 删除 0 行视为成功（幂等：本就不存在或不属于自己时不报错）。
func (p *pushStore) removeSubscriptionForPlayer(playerID, endpoint string) error {
	if p == nil || p.db == nil || playerID == "" || endpoint == "" {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	_, err := p.db.Exec(`DELETE FROM push_subscriptions WHERE endpoint = ? AND player_id = ?`, endpoint, playerID)
	return err
}

type storedSubscription struct {
	Endpoint string
	P256dh   string
	Auth     string
}

func (p *pushStore) subscriptionsForPlayer(playerID string) ([]storedSubscription, error) {
	if p == nil || p.db == nil || playerID == "" {
		return nil, nil
	}
	rows, err := p.db.Query(`SELECT endpoint, p256dh, auth FROM push_subscriptions WHERE player_id = ?`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []storedSubscription
	for rows.Next() {
		var s storedSubscription
		if err := rows.Scan(&s.Endpoint, &s.P256dh, &s.Auth); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// vapidKeys 是本地生成/落盘的 VAPID 密钥对（work/vapid.json），也可用环境变量覆盖
// （多实例部署时需要共享同一对密钥，否则旧订阅会在切到另一台实例后失效）。
type vapidKeys struct {
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

func loadOrGenerateVAPIDKeys(root string) (vapidKeys, error) {
	pub, priv := os.Getenv("VAPID_PUBLIC_KEY"), os.Getenv("VAPID_PRIVATE_KEY")
	if (pub == "") != (priv == "") {
		return vapidKeys{}, fmt.Errorf("VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY must be configured together")
	}
	if pub != "" {
		return vapidKeys{PublicKey: pub, PrivateKey: priv}, nil
	}
	path := filepath.Join(root, "work", "vapid.json")
	data, err := os.ReadFile(path)
	if err == nil {
		var keys vapidKeys
		if err := json.Unmarshal(data, &keys); err != nil {
			return vapidKeys{}, fmt.Errorf("decode persisted VAPID keys: %w", err)
		}
		if keys.PublicKey == "" || keys.PrivateKey == "" {
			return vapidKeys{}, fmt.Errorf("persisted VAPID key pair is incomplete")
		}
		return keys, nil
	}
	if !os.IsNotExist(err) {
		return vapidKeys{}, fmt.Errorf("read persisted VAPID keys: %w", err)
	}
	priv, pub, err = webpush.GenerateVAPIDKeys()
	if err != nil {
		return vapidKeys{}, err
	}
	keys := vapidKeys{PublicKey: pub, PrivateKey: priv}
	data, err = json.Marshal(keys)
	if err != nil {
		return vapidKeys{}, fmt.Errorf("encode VAPID keys: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return vapidKeys{}, fmt.Errorf("create VAPID directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return vapidKeys{}, fmt.Errorf("persist VAPID keys: %w", err)
	}
	return keys, nil
}

// pushPayload 是发给 Service Worker 的消息体，前端 push 事件里直接 JSON.parse。
type pushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Tag   string `json:"tag,omitempty"` // 同 tag 的通知会被浏览器合并/替换，避免刷屏
}

// shouldPush：偏好开启就发送；前台可见页面由 Service Worker 抑制重复通知。
func shouldPush(player *PlayerState, pref *bool) bool {
	return player != nil && ptrBool(pref)
}

// notifyOpponentTurn：现在轮到 waitingSeat 这个人动手了（RPS 对手已出拳 / 棋类换到 Ta 的回合）。
// 座位空/没开偏好都直接跳过。
func (s *Server) notifyOpponentTurn(room *RoomState, waitingSeat types.SeatKey) {
	occ := room.Seats[waitingSeat]
	if occ == nil {
		return
	}
	waiting := s.players[occ.GetID()]
	if waiting != nil && shouldPush(waiting, waiting.PushTurnEnabled) {
		s.sendPush(waiting, "轮到你了", "对方已经行动，轮到你出招/落子了。", "turn-"+room.ID)
	}
}

// notifySeatFilled：filledSeat 刚坐进一个人，如果对面座位坐着的是真人玩家且开了这个
// 推送源，就提醒 Ta「有人进房参战了」。座位空/没开偏好都直接跳过。
func (s *Server) notifySeatFilled(room *RoomState, filledSeat types.SeatKey) {
	occ := room.Seats[oppositeSeat(filledSeat)]
	if occ == nil {
		return
	}
	opponent := s.players[occ.GetID()]
	if opponent != nil && shouldPush(opponent, opponent.PushSeatEnabled) {
		s.sendPush(opponent, "你的房间来人了", "有玩家坐上了对面的战斗席，快回来看看吧。", "seat-"+room.ID)
	}
}

// sendPush 给这个玩家名下所有登记过的设备发一条 push；410/404（订阅已失效，用户卸载/清缓存）
// 会顺手把这条订阅删掉，避免越攒越多打没人收的空气。
// sendPush 的调用方通常持有 s.mu；subscriptionsForPlayer 是同步 SQLite 查询，
// 绝不能在锁内执行（会和其它所有 WS 事件/广播/计时器争抢唯一的 DB 连接），
// 因此整个查询+发送都挪到异步 goroutine 里，仅在锁内做最基本的 nil 判断和值捕获。
func (s *Server) sendPush(player *PlayerState, title, body, tag string) {
	if s.pushDB == nil || s.vapid.PublicKey == "" || player == nil {
		return
	}
	playerID := player.ID
	go func() {
		subs, err := s.pushDB.subscriptionsForPlayer(playerID)
		if err != nil {
			s.errorLog("push_subscription_query_failed", err.Error())
			return
		}
		if len(subs) == 0 {
			return
		}
		s.deliverPush(playerID, subs, title, body, tag)
	}()
}

type pushDeliveryResult struct {
	Accepted  int
	Failed    int
	Expired   int
	LastError string
}

// deliverPush 同步等待各浏览器厂商网关的 HTTP 回应。普通业务通知从 sendPush 的 goroutine
// 调用；测试通知直接使用返回值，把“网关是否接受”反馈给设置页，不再只显示盲目的已发送。
func (s *Server) deliverPush(playerID string, subs []storedSubscription, title, body, tag string) pushDeliveryResult {
	result := pushDeliveryResult{}
	payload, err := json.Marshal(pushPayload{Title: title, Body: body, Tag: tag})
	if err != nil {
		result.Failed = len(subs)
		result.LastError = err.Error()
		s.errorLog("push_payload_encode_failed", err.Error())
		return result
	}
	subscriber := os.Getenv("VAPID_SUBSCRIBER")
	if subscriber == "" {
		subscriber = "mailto:admin@rps.rbq.io"
	}
	opts := &webpush.Options{
		Subscriber:      subscriber,
		VAPIDPublicKey:  s.vapid.PublicKey,
		VAPIDPrivateKey: s.vapid.PrivateKey,
		TTL:             300,
		HTTPClient:      pushHTTPClient,
	}
	for _, sub := range subs {
		resp, err := webpush.SendNotification(payload, &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
		}, opts)
		if err != nil {
			result.Failed++
			result.LastError = err.Error()
			s.errorLog("push_send_failed", err.Error())
			continue
		}
		status := resp.StatusCode
		_ = resp.Body.Close()
		switch {
		case status >= 200 && status < 300:
			result.Accepted++
		case status == 404 || status == 410:
			result.Failed++
			result.Expired++
			result.LastError = fmt.Sprintf("HTTP %d", status)
			_ = s.pushDB.removeSubscription(sub.Endpoint)
		default:
			result.Failed++
			result.LastError = fmt.Sprintf("HTTP %d", status)
			s.errorLog("push_send_rejected", fmt.Sprintf(
				"player=%s status=%d tag=%s", playerID, status, tag,
			))
		}
	}
	return result
}

func pushDeliveryClientError(result pushDeliveryResult) string {
	if result.Accepted > 0 {
		return ""
	}
	if result.Expired > 0 {
		return "浏览器推送订阅已经失效，请重新订阅这台设备"
	}
	lower := strings.ToLower(result.LastError)
	if strings.Contains(lower, "x509") || strings.Contains(lower, "certificate") {
		return "服务器无法验证推送网关的 TLS 证书（缺少 CA 根证书）"
	}
	if strings.HasPrefix(result.LastError, "HTTP ") {
		return "浏览器推送网关拒绝了通知（" + result.LastError + "）"
	}
	return "服务器连接浏览器推送网关失败，请检查出站 HTTPS 和服务端错误日志"
}
