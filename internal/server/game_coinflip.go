package server

import (
	"fmt"
	"strings"

	"github.com/doumiao/newRPS/internal/types"
)

// 猜硬币（GameCoinFlip）是唯一不需要联机对手的惩罚小游戏：只用战斗席 A（座位 B 永远锁死，
// 房间照常在大厅/后台展示、支持观战席，观战者可以在座位 A 空出来后接棒坐下），
// 由玩家自己选"字"（1999-2018 版 1 元硬币的数字面）或"花"（菊花面），服务端立即随机开出
// 结果并结算。猜中不产生任何效果（不计入排位分、胜负场次、白给值——这些机制本就没有接入
// 这个游戏），猜错立即进入 PhasePunishment：因为 punishmentReviewer/humanOpponent 在座位 B
// 恒为空时天然返回 nil，证明提交（onPunishmentSubmit）里的
// `approvedBySystem := reviewer == nil || ...` 会自动短路成 true——不需要任何额外代码就能做到
// "提交即通过"，也不需要单独的断线判负逻辑（createDisconnectForfeit 同样因为 winnerSeat 对应
// 座位为空而天然跳过）。受罚次数（PublicStats.Punishments）与随机任务难度进度
// （RoomState.PunishmentTaskProgress）都是复用 setupPunishmentForPlayers /
// pickSystemTaskForPlayerAdvancing 得到的默认行为，照常累加/递增。
//
// PunishmentSource 只允许 random/series（建房时校验，见 handlers_room.go）——没有真人对手，
// "玩家发布任务"没有意义。任务文案里的 {winner} 占位符固定替换成
// AppConfig.Site.CoinFlipWinnerLabel（管理员可在后台自定义，默认"系统"）。

// coinFlipWinnerLabel 返回猜硬币任务文案里 {winner} 占位符的展示名。
func (s *Server) coinFlipWinnerLabel() string {
	label := strings.TrimSpace(s.cfg.Site.CoinFlipWinnerLabel)
	if label == "" {
		return types.DefaultCoinFlipWinnerLabel
	}
	return label
}

func coinFaceLabel(face types.CoinFace) string {
	switch face {
	case types.CoinFaceChar:
		return "字"
	case types.CoinFaceFlower:
		return "花"
	default:
		return "？"
	}
}

// resetCoinFlipRoom：座位 A 有人入座（建房自动入座、观战者接棒坐下、上一次惩罚结束）时的
// 公共入口——清空上一位玩家留下的抛掷展示态，进入可抛掷状态。调用方必须已确认 Seats[A] != nil。
func (s *Server) resetCoinFlipRoom(room *RoomState) {
	room.CoinFlip = &types.CoinFlipState{}
	s.startTurnBasedPlaying(room)
}

func (s *Server) onCoinFlipGuess(client *Client, env wsEnvelope) {
	var p struct {
		Face types.CoinFace `json:"face"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameCoinFlip {
		client.reply(env.ID, nil, "当前玩法不能猜硬币")
		return
	}
	if p.Face != types.CoinFaceChar && p.Face != types.CoinFaceFlower {
		client.reply(env.ID, nil, "请选择字或花")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok || seat != types.SeatA {
		client.reply(env.ID, nil, "只有参战席玩家可以猜硬币")
		return
	}
	if room.Phase != types.PhaseChoosing {
		client.reply(env.ID, nil, "现在还不能猜硬币")
		return
	}
	result := types.CoinFaceChar
	if randIntn(2) == 1 {
		result = types.CoinFaceFlower
	}
	correct := result == p.Face
	room.CoinFlip = &types.CoinFlipState{Guess: p.Face, Result: result, Correct: correct, SettledAt: nowMs()}
	oldStatus := room.Status
	s.settleCoinFlip(room, player, p.Face, result, correct)
	s.broadcastRoom(room.ID, oldStatus != room.Status)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

// settleCoinFlip 写入本次抛掷的对局历史，猜错时进入惩罚阶段。
func (s *Server) settleCoinFlip(room *RoomState, player *PlayerState, guess, result types.CoinFace, correct bool) {
	flipperName := playerShortName(player)
	winnerLabel := s.coinFlipWinnerLabel()
	guessLabel, resultLabel := coinFaceLabel(guess), coinFaceLabel(result)

	var punishedPlayers []*PlayerState
	var resultCode types.RoundResult
	var headline string
	if correct {
		resultCode = types.ResultA
		headline = "🪙 猜中"
		room.ResultText = fmt.Sprintf("%s 猜「%s」，硬币开出「%s」，猜中了！", flipperName, guessLabel, resultLabel)
	} else {
		resultCode = types.ResultB
		headline = "🪙 猜错"
		room.ResultText = fmt.Sprintf("%s 猜「%s」，硬币开出「%s」，猜错了，将接受%s的惩罚。", flipperName, guessLabel, resultLabel, winnerLabel)
		punishedPlayers = []*PlayerState{player}
	}
	punishmentTasks := s.buildPunishmentTasksWithWinnerName(room, punishedPlayers, winnerLabel, "coin_flip")
	punishedNames := make([]string, len(punishedPlayers))
	for i, p := range punishedPlayers {
		punishedNames[i] = playerShortName(p)
	}
	item := types.RoundHistoryItem{
		ID: randomID(), Round: len(room.RoundHistory) + 1, At: nowMs(),
		PlayerA: flipperName, PlayerB: winnerLabel,
		MoveA: types.MoveNoMove, MoveB: types.MoveNoMove,
		Result: resultCode, ResultLabel: headline, ResultText: room.ResultText,
		GameID:          types.GameCoinFlip,
		CoinFlipGuess:   guess,
		CoinFlipResult:  result,
		Ranked:          false,
		PunishmentTasks: punishmentTasks,
		PunishedNames:   punishedNames,
		Proofs:          []types.HistoryProof{},
	}
	if len(punishedNames) > 0 {
		item.PunishmentName = s.punishmentRoundLabel(room, punishmentTasks)
	}
	s.roomNotice(room, room.ResultText)
	s.addRoundHistory(room, item)
	s.setupPunishmentForPlayers(room, punishedIDs(punishedPlayers))
}
