package server

import (
	"testing"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

// 复现真实反馈的卡房场景：大话骰赢家（审核方/任务发布方）掉线离场后，
// 输家提交的证明应当被系统自动放行，而不是永远停在"待审核"
// （liarsDicePunishmentReviewer 曾经只按 s.players[winnerID] 是否存在判断，
// 而持久身份的玩家对象离场后依然存在，导致审核方永远查得到、证明永远等不到审批）。
func TestLiarsDiceWinnerLeavesThenLoserSubmitIsAutoApproved(t *testing.T) {
	s := newLiarsDiceTestServer(t)
	s.roomBroadcastTimers = map[string]*roomBroadcastPending{}
	s.roomBroadcastDelay = time.Hour
	room := newLiarsDiceTestRoom("r1", 2, 8)
	room.Settings.EnablePunishment = true
	room.Settings.PunishmentSource = "player"
	room.Settings.RequireOpponentConfirm = true
	room.Phase = types.PhasePunishment
	room.Status = "punishment"
	room.PunishedPlayerIDs = []string{"l"}
	room.LiarsDice.ParticipantIDs = []string{"w", "l"}
	room.RoundHistory = []types.RoundHistoryItem{{
		LiarsDiceWinnerID: "w", LiarsDiceLoserID: "l",
		PunishmentTasks: []types.PunishmentTask{{PlayerID: "l", PlayerName: "败方", AssignedBy: "w", AssignedByName: "胜方"}},
		Proofs:          []types.HistoryProof{},
	}}

	winner := newLiarsDiceTestPlayer("w", "胜方")
	loser := newLiarsDiceTestPlayer("l", "败方")
	winner.Connected, winner.SocketID, winner.RoomID = true, "sock-w", "r1"
	loser.Connected, loser.SocketID, loser.RoomID = true, "sock-l", "r1"
	s.players["w"], s.players["l"] = winner, loser
	s.rooms["r1"] = room
	s.clients["sock-w"] = &Client{id: "sock-w", playerID: "w", sendCh: make(chan []byte, 8), done: make(chan struct{})}
	s.clients["sock-l"] = &Client{id: "sock-l", playerID: "l", sendCh: make(chan []byte, 8), done: make(chan struct{})}

	// 胜方（审核方/任务发布方）离开房间：与真实场景一致地走一次完整的
	// handlePunishmentDeparture + 参战名单移除。
	res := s.leaveRoom(winner, LeaveManual)
	if !res.OK {
		t.Fatalf("胜方离开应当被允许，got %+v", res)
	}
	gotTask := room.RoundHistory[0].PunishmentTasks[0].TaskText
	if gotTask != "对方已离开，请提交文字说明完成本局惩罚。" {
		t.Fatalf("胜方离开后任务应补兜底文案，got %q", gotTask)
	}
	if containsString(room.LiarsDice.ParticipantIDs, "w") {
		t.Fatal("胜方离开后应从参战名单移除")
	}
	if reviewer := s.punishmentReviewer(room, "l"); reviewer != nil {
		t.Fatalf("胜方已离场，审核人应视为空，got %+v", reviewer)
	}

	env := wsEnvelope{ID: 1, D: map[string]any{"text": "已完成惩罚说明"}}
	s.onPunishmentSubmit(s.clients["sock-l"], env)

	// 证明通过后 resetForNextRound 会清空 room.Proofs，真正记录留在 RoundHistory 里。
	if room.Phase == types.PhasePunishment {
		t.Fatalf("证明应被自动放行，房间不应停留在惩罚阶段，got phase=%v", room.Phase)
	}
	histProof := room.RoundHistory[0].Proofs
	if len(histProof) != 1 || histProof[0].Status != "approved" {
		t.Fatalf("历史记录里应留下一条系统自动通过的证明，got %+v", histProof)
	}
}
