package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

// setupPunishmentReviewRoom 造一个 A=winner、B=loser 的 RPS 惩罚阶段房间，loser 已提交一条
// pending 状态的证明，winner 通过 sock-w 连接可以审核它。
func setupPunishmentReviewRoom(t *testing.T) (*Server, *Client) {
	t.Helper()
	s := newTestServer(t)
	s.rooms = map[string]*RoomState{}
	s.roomBroadcastTimers = map[string]*roomBroadcastPending{}

	winner := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "w", Name: "赢家", RoomID: "room1"}, SocketID: "sock-w"}
	loser := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "l", Name: "输家", RoomID: "room1"}}
	s.players["w"] = winner
	s.players["l"] = loser

	room := &RoomState{
		ID:       "room1",
		Phase:    types.PhasePunishment,
		Settings: types.RoomSettings{GameID: types.GameRPS, EnablePunishment: true, EnableRanked: true},
		Seats: map[types.SeatKey]SeatOccupant{
			types.SeatA: &HumanSeat{Player: types.PublicPlayer{ID: "w"}},
			types.SeatB: &HumanSeat{Player: types.PublicPlayer{ID: "l"}},
		},
		PunishedPlayerIDs: []string{"l"},
		Proofs: []types.PunishmentProof{
			{PlayerID: "l", Text: "做完了", Status: "pending", SubmittedAt: nowMs()},
		},
		RoundHistory: []types.RoundHistoryItem{{
			PunishmentTasks: []types.PunishmentTask{{PlayerID: "l", PlayerName: "输家", TaskText: "任务"}},
		}},
	}
	s.rooms["room1"] = room

	client := &Client{id: "sock-w", playerID: "w", sendCh: make(chan []byte, 64)}
	return s, client
}

// TestOnPunishmentReviewRejectRejectsReplayOnAlreadyRejectedProof 复现"抓包重放"刷分：
// 对同一条已经是 rejected 状态的证明反复发送 punishment:review{action:"reject"}，此前会
// 无限递增 RejectCount 并按 100/200/300... 扣分（applyProofRejectionPenalty），几十次调用
// 就能把对方打穿排位分下限。修复后必须在 proof 状态不是 pending 时直接拒绝。
func TestOnPunishmentReviewRejectRejectsReplayOnAlreadyRejectedProof(t *testing.T) {
	s, client := setupPunishmentReviewRoom(t)
	loser := s.players["l"]

	env := wsEnvelope{E: "punishment:review", ID: 1, D: map[string]any{
		"playerId": "l", "action": "reject", "redoTaskText": "重做",
	}}
	s.onPunishmentReview(client, env)
	data := lastReplyData(t, client)
	if data["ok"] != true {
		t.Fatalf("首次驳回应成功，got %#v", data)
	}
	room := s.rooms["room1"]
	if room.Proofs[0].Status != "rejected" {
		t.Fatalf("证明状态应为 rejected，got %q", room.Proofs[0].Status)
	}

	// 重放同一条驳回请求 9 次：证明已经是 rejected，不应该再次扣分/递增 RejectCount。
	for i := 0; i < 9; i++ {
		s.onPunishmentReview(client, wsEnvelope{E: "punishment:review", ID: int64(i + 2), D: map[string]any{
			"playerId": "l", "action": "reject", "redoTaskText": "再重做一次",
		}})
		errMsg := lastReplyError(t, client)
		if errMsg != "这条证明已经审核过了" {
			t.Fatalf("重放驳回应被拒绝，got err=%q", errMsg)
		}
	}
	if loser.Stats.RankedPoints != 0 {
		t.Fatalf("重放驳回不应继续扣分，got %d", loser.Stats.RankedPoints)
	}
	if room.RoundHistory[0].PunishmentTasks[0].RejectCount != 1 {
		t.Fatalf("RejectCount 应保持为 1，got %d", room.RoundHistory[0].PunishmentTasks[0].RejectCount)
	}
}

// TestOnPunishmentReviewRejectAllowsAfterResubmit 输家重新提交证明后（onPunishmentSubmit 会
// 生成一条全新的 pending proof）应该能再次被驳回——确认修复没有把"已驳回一次"做成永久锁死。
func TestOnPunishmentReviewRejectAllowsAfterResubmit(t *testing.T) {
	s, client := setupPunishmentReviewRoom(t)
	room := s.rooms["room1"]

	s.onPunishmentReview(client, wsEnvelope{E: "punishment:review", ID: 1, D: map[string]any{
		"playerId": "l", "action": "reject", "redoTaskText": "重做",
	}})
	if data := lastReplyData(t, client); data["ok"] != true {
		t.Fatalf("首次驳回应成功，got %#v", data)
	}

	// 模拟输家重新提交证明：onPunishmentSubmit 会把这条 proof 重置回 pending。
	room.Proofs[0].Status = "pending"

	s.onPunishmentReview(client, wsEnvelope{E: "punishment:review", ID: 2, D: map[string]any{
		"playerId": "l", "action": "reject", "redoTaskText": "还要重做",
	}})
	data := lastReplyData(t, client)
	if data["ok"] != true {
		t.Fatalf("重新提交后应允许再次驳回，got %#v", data)
	}
	if room.RoundHistory[0].PunishmentTasks[0].RejectCount != 2 {
		t.Fatalf("RejectCount 应递增到 2，got %d", room.RoundHistory[0].PunishmentTasks[0].RejectCount)
	}
}

// TestOnPunishmentReviewApproveRejectsReplayOnAlreadyApprovedProof 同样的状态机保护也适用于
// approve/forgive 分支：已审核过的证明不能被重复处理。
// PunishedPlayerIDs 里额外放一个从未提交证明的 "l2"，让 punishmentComplete 保持 false，
// 避免 approve 触发 resetForNextRound 把 Proofs 清空，从而能观察到重放本身被拒绝。
func TestOnPunishmentReviewApproveRejectsReplayOnAlreadyApprovedProof(t *testing.T) {
	s, client := setupPunishmentReviewRoom(t)
	room := s.rooms["room1"]
	room.PunishedPlayerIDs = append(room.PunishedPlayerIDs, "l2")

	s.onPunishmentReview(client, wsEnvelope{E: "punishment:review", ID: 1, D: map[string]any{
		"playerId": "l", "action": "approve",
	}})
	if data := lastReplyData(t, client); data["ok"] != true {
		t.Fatalf("首次通过应成功，got %#v", data)
	}

	s.onPunishmentReview(client, wsEnvelope{E: "punishment:review", ID: 2, D: map[string]any{
		"playerId": "l", "action": "approve",
	}})
	errMsg := lastReplyError(t, client)
	if errMsg != "这条证明已经审核过了" {
		t.Fatalf("重放通过请求应被拒绝，got err=%q", errMsg)
	}
}

func TestOnPunishmentReviewRejectsUnknownAction(t *testing.T) {
	s, client := setupPunishmentReviewRoom(t)
	s.onPunishmentReview(client, wsEnvelope{E: "punishment:review", ID: 1, D: map[string]any{
		"playerId": "l", "action": "crafted-approve",
	}})
	if got := lastReplyError(t, client); got != "审核动作无效" {
		t.Fatalf("unknown action error=%q", got)
	}
	if got := s.rooms["room1"].Proofs[0].Status; got != "pending" {
		t.Fatalf("unknown action changed proof status to %q", got)
	}
}

func TestOnPunishmentAssignTaskPersistsEventBeforeExposingEventID(t *testing.T) {
	s := newTestServer(t)
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s.db = db
	s.eventDB = newEventStore(db)
	s.rooms = map[string]*RoomState{}
	s.roomBroadcastTimers = map[string]*roomBroadcastPending{}
	winner := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "w", Name: "赢家", RoomID: "room1"}, SocketID: "sock-w"}
	loser := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "l", Name: "输家", RoomID: "room1"}}
	s.players[winner.ID], s.players[loser.ID] = winner, loser
	room := &RoomState{
		ID: "room1", Phase: types.PhasePunishment,
		Settings:          types.RoomSettings{GameID: types.GameRPS, EnablePunishment: true, PunishmentSource: "player"},
		PunishedPlayerIDs: []string{loser.ID},
		Seats: map[types.SeatKey]SeatOccupant{
			types.SeatA: &HumanSeat{Player: winner.PublicPlayer},
			types.SeatB: &HumanSeat{Player: loser.PublicPlayer},
		},
		RoundHistory: []types.RoundHistoryItem{{PunishmentTasks: []types.PunishmentTask{{
			PlayerID: loser.ID, PlayerName: loser.Name, AssignedBy: winner.ID, AssignedByName: winner.Name,
		}}}},
	}
	s.rooms[room.ID] = room
	client := &Client{id: winner.SocketID, playerID: winner.ID, sendCh: make(chan []byte, 8)}
	s.onPunishmentAssignTask(client, wsEnvelope{ID: 1, D: map[string]any{"playerId": loser.ID, "taskText": "完成任务"}})
	if data := lastReplyData(t, client); data["ok"] != true {
		t.Fatalf("assignment failed: %#v", data)
	}
	task := latestPunishmentTask(room, loser.ID)
	if task == nil || task.EventID == "" {
		t.Fatalf("persisted task must expose an event id: %+v", task)
	}
	if _, err := s.eventDB.getPunishmentEvent(task.EventID); err != nil {
		t.Fatalf("exposed event id must already exist in SQLite: %v", err)
	}
}
