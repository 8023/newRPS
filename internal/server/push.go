package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/doumiao/newRPS/internal/types"
)

// Web Push（Level 2 推送）：不依赖任何外部付费服务，VAPID 密钥对本地生成/落盘，
// 推送网关是浏览器厂商自己的公共端点（Chrome→Google、Firefox→Mozilla…），
// 服务端只是往这个端点发一条签名过的消息。
//
// 只要偏好开启就发送 Web Push，不再用 WebSocket Connected 状态做互斥。手机切后台后
// JavaScript 可能被冻结但连接仍被服务端视为在线；旧互斥会让前后端两条路径同时不发送。
// 是否存在前台可见页面由 Service Worker 最终判断并抑制通知，避免重复弹出。

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

func (p *pushStore) removeSubscription(endpoint string) error {
	if p == nil || p.db == nil || endpoint == "" {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	_, err := p.db.Exec(`DELETE FROM push_subscriptions WHERE endpoint = ?`, endpoint)
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
		if err != nil || len(subs) == 0 {
			return
		}
		payload, err := json.Marshal(pushPayload{Title: title, Body: body, Tag: tag})
		if err != nil {
			return
		}
		opts := &webpush.Options{
			Subscriber:      "mailto:support@example.com",
			VAPIDPublicKey:  s.vapid.PublicKey,
			VAPIDPrivateKey: s.vapid.PrivateKey,
			TTL:             300,
		}
		for _, sub := range subs {
			go func(sub storedSubscription) {
				resp, err := webpush.SendNotification(payload, &webpush.Subscription{
					Endpoint: sub.Endpoint,
					Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
				}, opts)
				if err != nil {
					s.errorLog("push_send_failed", err.Error())
					return
				}
				defer resp.Body.Close()
				if resp.StatusCode == 404 || resp.StatusCode == 410 {
					_ = s.pushDB.removeSubscription(sub.Endpoint)
				} else if resp.StatusCode >= 300 {
					s.errorLog("push_send_rejected", fmt.Sprintf(
						"player=%s status=%d tag=%s", playerID, resp.StatusCode, tag,
					))
				}
			}(sub)
		}
	}()
}
