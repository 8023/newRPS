package server

import (
	"strings"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

// TestCoinFlipFullPunishmentFlow 端到端覆盖猜硬币的核心约定：
//   - 建房即强制开启惩罚、单座位可直接抛掷（不需要战斗席 B、不需要 ready-up）；
//   - 猜错后进入惩罚阶段，受罚者是抛掷者自己，任务占位符已替换；
//   - 没有真人对手，提交证明自动通过（reviewer==nil），随即回到可抛掷状态；
//   - 不计入排位积分、胜负场次；受罚次数与随机任务难度进度照常累加/递增。
func TestCoinFlipFullPunishmentFlow(t *testing.T) {
	s := newTestServer(t)
	s.rooms = map[string]*RoomState{}
	s.roomBroadcastTimers = map[string]*roomBroadcastPending{}
	s.cfg.AccessControl.MaxActiveRoomsPerOwner = 2
	s.punishmentTasksCache = []types.PunishmentTaskConfig{
		{ID: "task-1", Version: 1, Text: `对着镜头说"我是{loser}"`, Order: 1, FactionIDs: []string{""}},
	}

	player := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p1", Name: "玩家甲"}, SocketID: "sock-1"}
	s.players["p1"] = player
	client := &Client{id: "sock-1", playerID: "p1", sendCh: make(chan []byte, 256)}

	createEnv := wsEnvelope{E: "room:create", ID: 1, D: map[string]any{
		"settings": map[string]any{
			"gameId":           "coinflip",
			"punishmentSource": "random",
		},
	}}
	s.onRoomCreate(client, createEnv)
	if _, errMsg := lastReplyEnvelope(t, client); errMsg != "" {
		t.Fatalf("建房失败: %s", errMsg)
	}
	if len(s.rooms) != 1 {
		t.Fatalf("expected 1 room, got %d", len(s.rooms))
	}
	var room *RoomState
	for _, r := range s.rooms {
		room = r
	}
	if room.Settings.GameID != types.GameCoinFlip {
		t.Fatalf("gameId = %q", room.Settings.GameID)
	}
	if !room.Settings.EnablePunishment {
		t.Fatal("猜硬币必须结构性强制开启惩罚")
	}
	if room.Settings.EnableRanked {
		t.Fatal("猜硬币不应允许开启排位")
	}
	if room.Phase != types.PhaseChoosing {
		t.Fatalf("建房后应直接进入可抛掷状态（无需 ready-up/战斗席 B），got phase=%v", room.Phase)
	}
	if room.Seats[types.SeatB] != nil {
		t.Fatal("猜硬币不应有战斗席 B 占用者")
	}

	// 反复猜同一面直到出现一次"猜错"。
	found := false
	for i := 0; i < 200 && !found; i++ {
		guessEnv := wsEnvelope{E: "coinflip:guess", ID: int64(i + 2), D: map[string]any{"face": "char"}}
		s.onCoinFlipGuess(client, guessEnv)
		if _, errMsg := lastReplyEnvelope(t, client); errMsg != "" {
			t.Fatalf("猜硬币失败: %s", errMsg)
		}
		if room.CoinFlip == nil {
			t.Fatal("room.CoinFlip 应已写入")
		}
		if !room.CoinFlip.Correct {
			found = true
		}
	}
	if !found {
		t.Fatal("200 次尝试都没有出现猜错，随机结算可能有问题")
	}
	if room.Phase != types.PhasePunishment {
		t.Fatalf("猜错后应进入惩罚阶段，got phase=%v", room.Phase)
	}
	if len(room.PunishedPlayerIDs) != 1 || room.PunishedPlayerIDs[0] != "p1" {
		t.Fatalf("受罚者应是抛掷者自己，got %v", room.PunishedPlayerIDs)
	}
	if player.Stats.Wins != 0 || player.Stats.Losses != 0 || player.Stats.Draws != 0 {
		t.Fatalf("猜硬币不应计入胜负场次，got wins=%d losses=%d draws=%d", player.Stats.Wins, player.Stats.Losses, player.Stats.Draws)
	}
	if player.Stats.RankedPoints != 0 {
		t.Fatalf("猜硬币不应计入排位积分，got %d", player.Stats.RankedPoints)
	}
	if player.Stats.Punishments != 1 {
		t.Fatalf("受罚次数应累加为 1，got %d", player.Stats.Punishments)
	}
	if room.PunishmentTaskProgress["p1"] != 1 {
		t.Fatalf("随机任务难度进度应递增为 1，got %d", room.PunishmentTaskProgress["p1"])
	}
	if len(room.RoundHistory) == 0 || len(room.RoundHistory[0].PunishmentTasks) != 1 {
		t.Fatal("对局历史应包含一条惩罚任务")
	}
	task := room.RoundHistory[0].PunishmentTasks[0]
	if task.TaskText == "" {
		t.Fatal("任务文案不应为空")
	}
	if strings.Contains(task.TaskText, "{loser}") || strings.Contains(task.TaskText, "{winner}") {
		t.Fatalf("占位符应已替换，got %q", task.TaskText)
	}
	if room.RoundHistory[0].CoinFlipGuess != types.CoinFaceChar {
		t.Fatalf("历史记录应保留本次猜测，got %q", room.RoundHistory[0].CoinFlipGuess)
	}

	// 没有真人对手：提交证明应自动通过，立即回到可抛掷状态。
	submitEnv := wsEnvelope{E: "punishment:submit", ID: 100, D: map[string]any{"text": "已完成"}}
	s.onPunishmentSubmit(client, submitEnv)
	if _, errMsg := lastReplyEnvelope(t, client); errMsg != "" {
		t.Fatalf("提交证明失败: %s", errMsg)
	}
	if room.Phase != types.PhaseChoosing {
		t.Fatalf("证明自动通过后应回到可抛掷状态，got phase=%v", room.Phase)
	}
	if len(room.PunishedPlayerIDs) != 0 {
		t.Fatalf("惩罚完成后 PunishedPlayerIDs 应清空，got %v", room.PunishedPlayerIDs)
	}
}

// TestCoinFlipCorrectGuessNoPunishment 猜中不应产生任何惩罚，也不应污染下一次抛掷。
func TestCoinFlipCorrectGuessNoPunishment(t *testing.T) {
	s := newTestServer(t)
	s.rooms = map[string]*RoomState{}
	s.roomBroadcastTimers = map[string]*roomBroadcastPending{}
	s.cfg.AccessControl.MaxActiveRoomsPerOwner = 2

	player := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p1", Name: "玩家甲"}, SocketID: "sock-1"}
	s.players["p1"] = player
	client := &Client{id: "sock-1", playerID: "p1", sendCh: make(chan []byte, 256)}

	s.onRoomCreate(client, wsEnvelope{E: "room:create", ID: 1, D: map[string]any{
		"settings": map[string]any{"gameId": "coinflip", "punishmentSource": "random"},
	}})
	if _, errMsg := lastReplyEnvelope(t, client); errMsg != "" {
		t.Fatalf("建房失败: %s", errMsg)
	}
	var room *RoomState
	for _, r := range s.rooms {
		room = r
	}

	sawCorrect, sawWrong := false, false
	for i := 0; i < 200 && !sawCorrect; i++ {
		s.onCoinFlipGuess(client, wsEnvelope{E: "coinflip:guess", ID: int64(i + 2), D: map[string]any{"face": "char"}})
		if _, errMsg := lastReplyEnvelope(t, client); errMsg != "" {
			t.Fatalf("猜硬币失败: %s", errMsg)
		}
		if room.CoinFlip == nil {
			t.Fatal("room.CoinFlip 应已写入")
		}
		if room.CoinFlip.Correct {
			sawCorrect = true
			continue
		}
		// 这条用例只关心猜中的情形；猜错会先进惩罚阶段（另有专门用例覆盖），
		// 这里自己提交一份证明（无真人对手，自动通过）把房间清回可抛掷状态再继续。
		sawWrong = true
		s.onPunishmentSubmit(client, wsEnvelope{E: "punishment:submit", ID: int64(1000 + i), D: map[string]any{"text": "已完成"}})
		if _, errMsg := lastReplyEnvelope(t, client); errMsg != "" {
			t.Fatalf("提交证明失败: %s", errMsg)
		}
	}
	if !sawCorrect {
		t.Fatal("200 次尝试都没有出现猜中，随机结算可能有问题")
	}
	if !sawWrong {
		t.Log("200 次尝试都没有出现猜错（理论上限，不影响本用例断言）")
	}
	if room.Phase != types.PhaseChoosing {
		t.Fatalf("猜中不应进入惩罚阶段，got phase=%v", room.Phase)
	}
	if len(room.PunishedPlayerIDs) != 0 {
		t.Fatalf("猜中不应产生受罚者，got %v", room.PunishedPlayerIDs)
	}
}
