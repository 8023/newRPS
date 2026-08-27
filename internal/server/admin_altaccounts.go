package server

import (
	"database/sql"
	"errors"
	"sort"

	"github.com/doumiao/newRPS/internal/types"
)

// 后台「用户管理」的"小号查询"：以玩家关联过的 connection_events.device
// （deviceKey = sha256(ip||fingerprint)，见 activitystore.go）为线索，反查同一设备曾登录
// 过的其它玩家，并对查到的每个账号继续反查它们自己的设备关联，逐层展开（BFS）直到没有
// 新玩家出现为止——也就是把"小号的小号"也挖出来，返回传递闭包意义上的整个关联连通分量。
// connection_events 是核心审计日志，不受 ANALYTICS_ENABLED 开关影响，从这套日志上线起就
// 无条件记录每一条连接（见 activitystore.go 顶部注释），比 analytics_sessions 更完整、更权威，
// 唯一的缺口是 kill -9（非优雅关停）当时仍在线的连接不会留下记录。
//
// connection_events 无清理、只增不减，早期版本没给 device 建索引导致全表扫描，开服后随
// 连接数增长曾实测到 4-5 秒。现在三处一起兜底：1) idx_connection_events_device 索引；
// 2) 独立只读连接 s.activityRO，不占主写连接；3) 实际 SQL 遍历期间释放 s.mu，不让后台
// 查询阻塞全服游戏事件。当前在线关系在释放锁前快照，不并发读取 clients/players map。
var errAltAccountsStoreUnavailable = errors.New("活动日志不可用，无法查询关联设备")

// altAccountsReadDB 返回小号查询该用哪个连接：优先只读连接，不可用时退回主连接
// （功能不失效，只是失去"不占用主写连接"这条好处）。
func (s *Server) altAccountsReadDB() *sql.DB {
	if s.activityRO != nil {
		return s.activityRO
	}
	return s.db
}

// altAccountsToPublicPlayers 把去重后的玩家 ID 列表转成后台展示用的公开档案，
// 按昵称排序；未在线内存表中查到（比如早已被清理的游客）的 ID 直接跳过。
func (s *Server) altAccountsToPublicPlayers(ids []string) []types.PublicPlayer {
	if len(ids) == 0 {
		return nil
	}
	out := make([]types.PublicPlayer, 0, len(ids))
	for _, id := range ids {
		if pl := s.players[id]; pl != nil {
			out = append(out, s.publicPlayerAdmin(pl))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// altAccountTraversalLimit 是"小号查询"单次 BFS 最多展开的玩家数（不含起点）。
// connection_events 的设备关联理论上可能因为网吧/家人共用设备等原因连成一片很大的图，
// 即使查询在独立只读连接执行，也要加个上限，避免极端共享网络形成的巨大连通分量长期占用
// CPU/数据库连接。达到上限时返回的是已发现的一部分，不是"报错"，前端会明确提示不完整。
const altAccountTraversalLimit = 500

type altAccountLiveSnapshot struct {
	devicesByPlayer map[string][]string
	playersByDevice map[string][]string
}

// snapshotAltAccountLiveLinks 把尚未落进 connection_events 的当前连接也纳入查询。
// connection_events 只在断线时写整行；若忽略这里，新设备首次登录期间点“小号查询”会
// 错误返回空，必须等相关账号至少断线一次后才能查到。
func (s *Server) snapshotAltAccountLiveLinks() altAccountLiveSnapshot {
	snap := altAccountLiveSnapshot{
		devicesByPlayer: map[string][]string{},
		playersByDevice: map[string][]string{},
	}
	playerDeviceSeen := map[string]map[string]struct{}{}
	devicePlayerSeen := map[string]map[string]struct{}{}
	for _, client := range s.clients {
		// 这里故意不使用 getPlayerByClient：同一账号被新连接顶号后，旧 socket 在真正
		// 断开并写入 connection_events 前仍是一条已经认证过的在线设备关系，也应进入
		// 审计快照；playerID 只会在认证成功后赋值，再确认档案仍存在即可。
		player := s.players[client.playerID]
		if player == nil {
			continue
		}
		dev := client.deviceKey
		if dev == "" && (client.ipAddress != "" || client.fingerprint != "") {
			dev = deviceKey(client.ipAddress, client.fingerprint)
		}
		if dev == "" {
			continue
		}
		if playerDeviceSeen[player.ID] == nil {
			playerDeviceSeen[player.ID] = map[string]struct{}{}
		}
		if _, exists := playerDeviceSeen[player.ID][dev]; !exists {
			playerDeviceSeen[player.ID][dev] = struct{}{}
			snap.devicesByPlayer[player.ID] = append(snap.devicesByPlayer[player.ID], dev)
		}
		if devicePlayerSeen[dev] == nil {
			devicePlayerSeen[dev] = map[string]struct{}{}
		}
		if _, exists := devicePlayerSeen[dev][player.ID]; !exists {
			devicePlayerSeen[dev][player.ID] = struct{}{}
			snap.playersByDevice[dev] = append(snap.playersByDevice[dev], player.ID)
		}
	}
	for playerID := range snap.devicesByPlayer {
		sort.Strings(snap.devicesByPlayer[playerID])
	}
	for device := range snap.playersByDevice {
		sort.Strings(snap.playersByDevice[device])
	}
	return snap
}

func altAccountCandidateIDs(db *sql.DB, playerID string, live altAccountLiveSnapshot) ([]string, bool, error) {
	visited := map[string]bool{playerID: true}
	queue := []string{playerID}
	var discovered []string
	for len(queue) > 0 && len(discovered) < altAccountTraversalLimit {
		cur := queue[0]
		queue = queue[1:]
		ids, devices, err := altAccountLinks(db, cur)
		if err != nil {
			return nil, false, err
		}
		devices = append(devices, live.devicesByPlayer[cur]...)
		direct := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			direct[id] = struct{}{}
		}
		for _, device := range devices {
			for _, id := range live.playersByDevice[device] {
				if id != cur {
					direct[id] = struct{}{}
				}
			}
		}
		ids = ids[:0]
		for id := range direct {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			if visited[id] {
				continue
			}
			visited[id] = true
			discovered = append(discovered, id)
			queue = append(queue, id)
			if len(discovered) >= altAccountTraversalLimit {
				break
			}
		}
	}
	return discovered, len(queue) > 0, nil
}

// altAccountCandidates 从 playerID 出发做广度优先遍历：查到的每个小号，再各自反查它们
// 自己用过的设备关联到谁，逐层展开，直到没有新玩家出现为止，返回传递闭包意义上的整个
// 关联连通分量（去掉 playerID 本人）。
//
// 用 visited 集合记录已经入队/已处理过的玩家 ID：一旦某个 ID 被标记过就再也不会重新入队，
// 这就是避免死循环的关键——A→B、B→A 这种环，B 在扩展 A 的第一层就已经 visited，
// 处理到 B 时查出 A 会被直接跳过，不会无限反复横跳。
func (s *Server) altAccountCandidates(playerID string) ([]types.PublicPlayer, error) {
	if s.activityDB == nil {
		return nil, errAltAccountsStoreUnavailable
	}
	db := s.altAccountsReadDB()
	ids, _, err := altAccountCandidateIDs(db, playerID, s.snapshotAltAccountLiveLinks())
	if err != nil {
		return nil, err
	}
	return s.altAccountsToPublicPlayers(ids), nil
}

// altAccountCandidatesForAdmin 仅供 handleWSEvent 已持有 s.mu 的后台调用链使用。
// 慢查询期间释放全局锁，避免 connection_events 增长后把所有游戏事件一起卡住；当前
// 连接关系先做不可变快照，查询结束重新持锁后才读取玩家档案。
func (s *Server) altAccountCandidatesForAdmin(playerID string) ([]types.PublicPlayer, bool, error) {
	if s.activityDB == nil {
		return nil, false, errAltAccountsStoreUnavailable
	}
	db := s.altAccountsReadDB()
	live := s.snapshotAltAccountLiveLinks()
	s.mu.Unlock()
	type result struct {
		ids       []string
		truncated bool
		err       error
	}
	query := func() result {
		defer s.mu.Lock()
		ids, truncated, err := altAccountCandidateIDs(db, playerID, live)
		return result{ids: ids, truncated: truncated, err: err}
	}()
	if query.err != nil {
		return nil, false, query.err
	}
	return s.altAccountsToPublicPlayers(query.ids), query.truncated, nil
}
