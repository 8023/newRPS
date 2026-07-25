package server

import (
	"database/sql"
	"encoding/json"
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
// 去重策略：只有当玩家当前没有任何活跃 WebSocket 连接（player.Connected==false）时
// 才发送 push——只要还连着，前端自己已经实时收到了同一个事件，能自己决定要不要弹
// Notification（Level 1），不需要服务端重复推一次。

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
	if pub, priv := os.Getenv("VAPID_PUBLIC_KEY"), os.Getenv("VAPID_PRIVATE_KEY"); pub != "" && priv != "" {
		return vapidKeys{PublicKey: pub, PrivateKey: priv}, nil
	}
	path := filepath.Join(root, "work", "vapid.json")
	if data, err := os.ReadFile(path); err == nil {
		var keys vapidKeys
		if json.Unmarshal(data, &keys) == nil && keys.PublicKey != "" && keys.PrivateKey != "" {
			return keys, nil
		}
	}
	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		return vapidKeys{}, err
	}
	keys := vapidKeys{PublicKey: pub, PrivateKey: priv}
	if data, err := json.Marshal(keys); err == nil {
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		_ = os.WriteFile(path, data, 0o600)
	}
	return keys, nil
}

// pushPayload 是发给 Service Worker 的消息体，前端 push 事件里直接 JSON.parse。
type pushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Tag   string `json:"tag,omitempty"` // 同 tag 的通知会被浏览器合并/替换，避免刷屏
}

// shouldPush：只在玩家当前完全没有活跃连接、且对应偏好开启时才发送。
func shouldPush(player *PlayerState, pref *bool) bool {
	return player != nil && !player.Connected && ptrBool(pref)
}

// notifyOpponentTurn：现在轮到 waitingSeat 这个人动手了（RPS 对手已出拳 / 棋类换到 Ta 的回合）。
// 座位空/在线/没开偏好都直接跳过。
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

// notifySeatFilled：filledSeat 刚坐进一个人，如果对面座位坐着的是真人玩家且当前离线、
// 又开了这个推送源，就提醒 Ta「有人进房参战了」。座位空/在线/没开偏好都直接跳过。
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
func (s *Server) sendPush(player *PlayerState, title, body, tag string) {
	if s.pushDB == nil || s.vapid.PublicKey == "" || player == nil {
		return
	}
	subs, err := s.pushDB.subscriptionsForPlayer(player.ID)
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
				s.securityLog("push_send_rejected", map[string]any{
					"playerId": player.ID, "status": resp.StatusCode, "tag": tag,
				})
			}
		}(sub)
	}
}
