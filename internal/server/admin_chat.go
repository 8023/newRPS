package server

import "github.com/doumiao/newRPS/internal/types"

// onAdminChatSearch 是后台「聊天管理」检索面板的只读入口：按用户名/聊天内容/房间名
// （或仅大厅）跨大厅+房间历史检索，支持分页；返回结果供管理员多选后走 admin:action 的
// chatSoftDelete/chatRestore 处理。删除已不再有"一键清空"入口，只能对检索出的具体消息
// 操作，见 handlers_room.go 里 onAdminAction 的对应分支。
//
// 回包额外带 authors（与 chat:load 相同，按 playerId 去重的当前档案），供前端用药丸标签
// 渲染发言人；并尽量补全空的 roomName（旧行没有快照时，用同页其它行或仍开着的房间名）。
func (s *Server) onAdminChatSearch(client *Client, env wsEnvelope) {
	_, isAdminSocket := s.adminClientIDs[client.id]
	if !isAdminSocket {
		client.reply(env.ID, nil, "需要管理员权限")
		return
	}
	var q chatSearchQuery
	_ = decodeD(env, &q)
	if s.chatDB == nil {
		client.reply(env.ID, nil, "聊天存储不可用")
		return
	}
	// 全文/子串检索无法依靠普通 B-tree 索引，历史量大时可能耗时明显。所有 WS handler
	// 默认都持有 s.mu；若在锁内扫描聊天表，会让全服游戏事件一起排队。优先用现成的
	// 独立只读连接，并在查询期间暂时释放全局锁；回到内存玩家/房间数据前再自动拿回锁。
	searchStore := s.chatDB
	if s.activityRO != nil {
		searchStore = newChatStore(s.activityRO)
	}
	messages, hasMore, err := s.searchAdminChatWithoutServerLock(searchStore, q)
	// 查询期间管理员可能已经断线或被注销；重新确认当前 socket 权限后再拼装/回包。
	if _, stillAdmin := s.adminClientIDs[client.id]; !stillAdmin {
		client.reply(env.ID, nil, "需要管理员权限")
		return
	}
	if err != nil {
		s.errorLog("chat_search_failed", err.Error())
		client.reply(env.ID, nil, "检索失败")
		return
	}
	s.fillAdminChatRoomNames(messages)
	plain := make([]types.ChatMessage, len(messages))
	for i, m := range messages {
		plain[i] = m.ChatMessage
	}
	client.reply(env.ID, map[string]any{
		"messages": messages,
		"hasMore":  hasMore,
		"authors":  s.chatAuthorsFor(plain),
	}, "")
}

// searchAdminChatWithoutServerLock 仅供 handleWSEvent 已持有 s.mu 的调用链使用。
// defer 保证即使底层查询 panic，外层 handleWSEvent 的延迟 Unlock 仍对应一把已持有的锁。
func (s *Server) searchAdminChatWithoutServerLock(store *chatStore, q chatSearchQuery) ([]adminChatMessage, bool, error) {
	s.mu.Unlock()
	defer s.mu.Lock()
	return store.search(q)
}

// setAdminChatDeletedWithoutServerLock 与检索同理：批量 UPDATE 已由 chatStore 自身事务和
// mutex 串行化，不需要占着全服状态锁等待 SQLite；返回后再在锁内广播删除事件。
func (s *Server) setAdminChatDeletedWithoutServerLock(refs []chatMessageRef, deleted bool, at int64) (int, error) {
	s.mu.Unlock()
	defer s.mu.Lock()
	return s.chatDB.setDeleted(refs, deleted, at)
}

// fillAdminChatRoomNames 给没有房间名快照的旧行补名字：先用本页其它行的同 roomId 快照，
// 再查仍开着的房间。补不上就保持空串，前端展示「已关闭房间」，不再回退成房间唯一 ID。
func (s *Server) fillAdminChatRoomNames(messages []adminChatMessage) {
	if len(messages) == 0 {
		return
	}
	byRoom := make(map[string]string)
	for _, m := range messages {
		if m.RoomID != "" && m.RoomName != "" {
			byRoom[m.RoomID] = m.RoomName
		}
	}
	for i := range messages {
		m := &messages[i]
		if m.RoomID == "" || m.RoomName != "" {
			continue
		}
		if name := byRoom[m.RoomID]; name != "" {
			m.RoomName = name
			continue
		}
		if room := s.rooms[m.RoomID]; room != nil {
			m.RoomName = room.Settings.Name
			if m.RoomName != "" {
				byRoom[m.RoomID] = m.RoomName
			}
		}
	}
}
