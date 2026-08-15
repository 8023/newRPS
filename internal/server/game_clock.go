package server

import (
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

// turnBasedClockState 把不同棋类状态中的公共计时字段映射到同一套实现。
// 各游戏只负责提供状态访问、暂停条件和超时结算规则。
type turnBasedClockState struct {
	turn            types.SeatKey
	ended           bool
	blocked         bool
	moveDeadlineAt  *int64
	clockDeadlineAt *int64
	clockRemaining  *map[types.SeatKey]int64
}

type turnBasedClockHooks struct {
	state     func(*RoomState) (turnBasedClockState, bool)
	settings  func(*RoomState) (moveSeconds, gameMinutes int)
	onTimeout func(*RoomState, types.SeatKey)
}

type turnBasedClockTimer struct {
	timer *time.Timer
}

func (s *Server) clearTurnBasedClockTimer(roomID string) {
	if entry := s.turnBasedClockTimers[roomID]; entry != nil {
		if entry.timer != nil {
			entry.timer.Stop()
		}
		delete(s.turnBasedClockTimers, roomID)
	}
}

// freezeTurnBasedClock 把当前行动方正在走动的总时长冻结成静态剩余毫秒数。
func (s *Server) freezeTurnBasedClock(room *RoomState, seat types.SeatKey, hooks turnBasedClockHooks) {
	state, ok := hooks.state(room)
	if !ok || state.clockRemaining == nil || *state.clockRemaining == nil {
		return
	}
	if *state.clockDeadlineAt > 0 {
		remaining := *state.clockDeadlineAt - nowMs()
		if remaining < 0 {
			remaining = 0
		}
		(*state.clockRemaining)[seat] = remaining
	}
	*state.clockDeadlineAt = 0
}

// pauseTurnBasedClock 用于悔棋、认输和游戏自己的结算等待窗口。
func (s *Server) pauseTurnBasedClock(room *RoomState, seat types.SeatKey, hooks turnBasedClockHooks) {
	state, ok := hooks.state(room)
	if !ok {
		return
	}
	s.freezeTurnBasedClock(room, seat, hooks)
	*state.moveDeadlineAt = 0
	s.clearTurnBasedClockTimer(room.ID)
}

func (s *Server) resumeTurnBasedClock(room *RoomState, hooks turnBasedClockHooks) {
	state, ok := hooks.state(room)
	if !ok || state.ended {
		return
	}
	s.armTurnBasedClock(room, state.turn, hooks)
}

// armTurnBasedClock 在真正轮到 seat 落子时统一重置每子倒计时、恢复总时长并安排超时检测。
func (s *Server) armTurnBasedClock(room *RoomState, seat types.SeatKey, hooks turnBasedClockHooks) {
	state, ok := hooks.state(room)
	if !ok {
		return
	}
	moveSeconds, gameMinutes := hooks.settings(room)
	now := nowMs()
	if moveSeconds > 0 {
		*state.moveDeadlineAt = now + int64(moveSeconds)*1000
	} else {
		*state.moveDeadlineAt = 0
	}
	if gameMinutes > 0 {
		if *state.clockRemaining == nil {
			total := int64(gameMinutes) * 60_000
			*state.clockRemaining = map[types.SeatKey]int64{types.SeatA: total, types.SeatB: total}
		}
		*state.clockDeadlineAt = now + (*state.clockRemaining)[seat]
	} else {
		*state.clockDeadlineAt = 0
	}
	s.scheduleTurnBasedClockTimer(room, hooks)
}

func (s *Server) scheduleTurnBasedClockTimer(room *RoomState, hooks turnBasedClockHooks) {
	s.clearTurnBasedClockTimer(room.ID)
	state, ok := hooks.state(room)
	if !ok || state.ended {
		return
	}
	deadline := earliestPositiveDeadline(*state.moveDeadlineAt, *state.clockDeadlineAt)
	if deadline == 0 {
		return
	}
	delay := deadline - nowMs()
	if delay < 0 {
		delay = 0
	}
	seat := state.turn
	roomID := room.ID
	if s.turnBasedClockTimers == nil {
		s.turnBasedClockTimers = map[string]*turnBasedClockTimer{}
	}
	// 先登记唯一槽位，再启动可能为 0 延迟的回调；这样无需依赖调用方必须持锁。
	entry := &turnBasedClockTimer{}
	s.turnBasedClockTimers[roomID] = entry
	entry.timer = timeAfterFunc(time.Duration(delay)*time.Millisecond, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		// Stop 与回调开始可能恰好撞在一起；旧回调不能删掉同房间刚换手建立的新计时器。
		if s.turnBasedClockTimers[roomID] != entry {
			return
		}
		delete(s.turnBasedClockTimers, roomID)
		current := s.rooms[roomID]
		if current == nil {
			return
		}
		currentState, ok := hooks.state(current)
		if !ok || currentState.ended || currentState.turn != seat || currentState.blocked {
			return
		}
		now := nowMs()
		moveExpired := *currentState.moveDeadlineAt > 0 && now >= *currentState.moveDeadlineAt
		clockExpired := *currentState.clockDeadlineAt > 0 && now >= *currentState.clockDeadlineAt
		if !moveExpired && !clockExpired {
			s.scheduleTurnBasedClockTimer(current, hooks)
			return
		}
		hooks.onTimeout(current, seat)
		s.broadcastRoom(current.ID, true)
	})
}
