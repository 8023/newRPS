package server

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"github.com/doumiao/newRPS/internal/types"
)

func (s *Server) punishmentPlayersForResult(room *RoomState, result types.RoundResult) []*PlayerState {
	if !room.Settings.EnablePunishment {
		return nil
	}
	var punishSeats []types.SeatKey
	if result == types.ResultDoubleLoss {
		punishSeats = []types.SeatKey{types.SeatA, types.SeatB}
	} else if result == types.ResultDraw {
		if room.Settings.TieDoublePunish {
			punishSeats = []types.SeatKey{types.SeatA, types.SeatB}
		}
	} else {
		punishSeats = []types.SeatKey{oppositeSeat(types.SeatKey(result))}
	}
	var out []*PlayerState
	for _, seat := range punishSeats {
		occ := room.Seats[seat]
		if occ == nil {
			continue
		}
		if p := s.players[occ.GetID()]; p != nil {
			out = append(out, p)
		}
	}
	return out
}

func (s *Server) addRoundHistory(room *RoomState, item types.RoundHistoryItem) {
	item = sanitizeRoundHistoryItem(item)
	room.RoundHistory = append([]types.RoundHistoryItem{item}, room.RoundHistory...)
	s.emitToRoom(room.ID, "room:historyAppend", map[string]any{
		"roomId": room.ID,
		"item":   item,
		"total":  len(room.RoundHistory),
	})
	s.requestPersist("lazy")
}

func (s *Server) currentPunishment(room *RoomState) *types.PunishmentConfig {
	selected := s.selectedPunishments(room.Settings)
	if len(selected) == 0 {
		return nil
	}
	p := selected[randIntn(len(selected))]
	return &p
}

func randIntn(n int) int {
	if n <= 0 {
		return 0
	}
	return rand.Intn(n)
}

func (s *Server) punishmentNameForRoom(room *RoomState, punishment *types.PunishmentConfig) string {
	if room.Settings.PunishmentSource == "player" {
		return "玩家发布任务"
	}
	if punishment != nil {
		return punishment.Name
	}
	return ""
}

func (s *Server) buildPunishmentTasks(room *RoomState, punishedPlayers []*PlayerState, result types.RoundResult, punishment *types.PunishmentConfig) []types.PunishmentTask {
	// 始终返回非 nil 切片，避免 history 里 punishmentTasks: null
	out := make([]types.PunishmentTask, 0, len(punishedPlayers))
	for _, player := range punishedPlayers {
		var assigner *PlayerState
		var systemTask *punishmentTaskResult
		if room.Settings.PunishmentSource == "player" {
			assigner = s.taskAssigner(room, player.ID)
		} else {
			systemTask = s.punishmentTaskForPlayer(room, player, s.winnerNameForResult(room, result), punishment)
		}
		task := types.PunishmentTask{
			PlayerID:     player.ID,
			PlayerName:   playerShortName(player),
			FactionID:    player.FactionID,
			FactionLabel: player.FactionLabel,
			TaskText:     "",
		}
		if systemTask != nil {
			task.TaskText = systemTask.TaskText
			task.BackgroundImage = systemTask.BackgroundImage
			if systemTask.BackgroundOpacity != nil {
				task.BackgroundOpacity = systemTask.BackgroundOpacity
			}
			if s.eventDB != nil {
				task.EventID = randomID()
				if err := s.eventDB.insertPunishmentTask(task.EventID, nowMs(), "system", room.ID, "", "", player.ID, task.PlayerName, task.TaskText); err != nil {
					s.errorLog("punishment_event_insert_failed", err.Error())
				}
			}
		}
		if assigner != nil {
			task.AssignedBy = assigner.ID
			task.AssignedByName = assigner.Name
		}
		out = append(out, task)
	}
	return out
}

// latestPunishmentTask 返回 room.RoundHistory[0] 中某玩家当前的惩罚任务（指针，可原地改写）。
func latestPunishmentTask(room *RoomState, playerID string) *types.PunishmentTask {
	if len(room.RoundHistory) == 0 {
		return nil
	}
	latest := &room.RoundHistory[0]
	for i := range latest.PunishmentTasks {
		if latest.PunishmentTasks[i].PlayerID == playerID {
			return &latest.PunishmentTasks[i]
		}
	}
	return nil
}

func (s *Server) taskAssigner(room *RoomState, punishedPlayerID string) *PlayerState {
	return s.punishmentReviewer(room, punishedPlayerID)
}

// punishmentReviewer 返回有权给 punishedID 发布任务 / 审核其证明的玩家：
// 座位制游戏是对手座位上的真人；大话骰不进 Seats 体系，是本局赢家。
func (s *Server) punishmentReviewer(room *RoomState, punishedID string) *PlayerState {
	if room.Settings.GameID == types.GameLiarsDice {
		return s.liarsDicePunishmentReviewer(room, punishedID)
	}
	return s.humanOpponent(room, punishedID)
}

type punishmentTaskResult struct {
	TaskText          string
	BackgroundImage   string
	BackgroundOpacity *float64
}

func (s *Server) punishmentTaskForPlayer(room *RoomState, player *PlayerState, winnerName string, punishment *types.PunishmentConfig) *punishmentTaskResult {
	if punishment == nil {
		op := 0.22
		return &punishmentTaskResult{TaskText: "请完成本局惩罚。", BackgroundOpacity: &op}
	}
	var task *types.PunishmentTaskConfig
	if len(punishment.Tasks) > 0 {
		t := punishment.Tasks[randIntn(len(punishment.Tasks))]
		task = &t
	}
	group := s.taskGroupForFaction(player.FactionID)
	variant := ""
	if task != nil && task.Variants != nil {
		variant = strings.TrimSpace(task.Variants[group])
	}
	if variant == "" && punishment.Variants != nil {
		variant = strings.TrimSpace(punishment.Variants[group])
	}
	taskText := cleanTaskText(variant, player.FactionLabel)
	if taskText == "" {
		taskText = cleanTaskText(punishment.Description, player.FactionLabel)
	}
	taskText = applyPunishmentPlaceholders(taskText, playerShortName(player), winnerName)
	op := 0.22
	if task != nil {
		op = task.BackgroundOpacity
	}
	bg := ""
	if task != nil && len(task.BackgroundImages) > 0 {
		bg = task.BackgroundImages[randIntn(len(task.BackgroundImages))]
	}
	return &punishmentTaskResult{TaskText: taskText, BackgroundImage: bg, BackgroundOpacity: &op}
}

func cleanTaskText(taskText, factionLabel string) string {
	taskText = strings.TrimSpace(taskText)
	if factionLabel != "" {
		re := regexp.MustCompile(`^` + regexp.QuoteMeta(factionLabel) + `[：:]\s*`)
		taskText = re.ReplaceAllString(taskText, "")
	}
	// 兜底：清掉改名前的旧版硬编码阵营标签前缀。阵营标签现在完全由后台配置、可随时改名
	// （本次就把"男性阵营"等改成了"顺性别男"等），历史任务文案里可能还留着改名前的旧
	// 前缀——这些字符串已经不在当前配置里，上面按 factionLabel 动态匹配的分支测不到它们。
	re2 := regexp.MustCompile(`^(男性阵营|女性阵营|男娘阵营|其他阵营)[：:]\s*`)
	taskText = re2.ReplaceAllString(taskText, "")
	return strings.TrimSpace(taskText)
}

// applyPunishmentPlaceholders 替换系统/自定义任务文案中的占位符：
//
//	{loser}  → 败者昵称（本条任务对应的受罚玩家）
//	{winner} → 胜者昵称（本局唯一胜者座位；平局双罚/双败时为空字符串）
func applyPunishmentPlaceholders(taskText, loserName, winnerName string) string {
	if taskText == "" {
		return taskText
	}
	taskText = strings.ReplaceAll(taskText, "{loser}", loserName)
	taskText = strings.ReplaceAll(taskText, "{winner}", winnerName)
	return taskText
}

// winnerNameForResult 返回胜者展示名；无明确胜者时返回 ""。
func (s *Server) winnerNameForResult(room *RoomState, result types.RoundResult) string {
	var seat types.SeatKey
	switch result {
	case types.ResultA:
		seat = types.SeatA
	case types.ResultB:
		seat = types.SeatB
	default:
		return ""
	}
	return s.seatShortName(room, seat)
}

func (s *Server) seatShortName(room *RoomState, seat types.SeatKey) string {
	if room == nil {
		return ""
	}
	occ := room.Seats[seat]
	if occ == nil {
		return ""
	}
	if p := s.players[occ.GetID()]; p != nil {
		return playerShortName(p)
	}
	return occupantName(occ)
}

func (s *Server) attachProofToLatestHistory(room *RoomState, proof types.HistoryProof) {
	if len(room.RoundHistory) == 0 {
		return
	}
	latest := &room.RoundHistory[0]
	var taskText string
	for _, t := range latest.PunishmentTasks {
		if t.PlayerID == proof.PlayerID {
			taskText = t.TaskText
			break
		}
	}
	filtered := latest.Proofs[:0]
	for _, p := range latest.Proofs {
		if p.PlayerID != proof.PlayerID {
			filtered = append(filtered, p)
		}
	}
	if proof.TaskText == "" {
		proof.TaskText = taskText
	}
	latest.Proofs = append(filtered, proof)
}

func (s *Server) updateProofInLatestHistory(room *RoomState, playerID string, next types.HistoryProof) {
	if len(room.RoundHistory) == 0 {
		return
	}
	latest := &room.RoundHistory[0]
	for i := range latest.Proofs {
		if latest.Proofs[i].PlayerID == playerID {
			p := latest.Proofs[i]
			if next.Status != "" {
				p.Status = next.Status
			}
			if next.ReviewedBy != "" {
				p.ReviewedBy = next.ReviewedBy
			}
			if next.ReviewedAt != nil {
				p.ReviewedAt = next.ReviewedAt
			}
			if next.RejectReason != "" {
				p.RejectReason = next.RejectReason
			}
			if next.RedoTaskText != "" {
				p.RedoTaskText = next.RedoTaskText
			}
			if next.Text != "" {
				p.Text = next.Text
			}
			if next.ImageURL != "" {
				p.ImageURL = next.ImageURL
			}
			latest.Proofs[i] = p
			return
		}
	}
}

// proofRejectionPenaltyPoints：同一惩罚任务在同一局内被胜方连续第 N 次审核不通过时的额外扣分。
// 第 1、2 次不扣分；第 3 次起 100，第 4 次 200，第 5 次 300……此后每多一次连续不通过再加罚 100。
func proofRejectionPenaltyPoints(rejectCount int) int {
	if rejectCount < 3 {
		return 0
	}
	return (rejectCount - 2) * 100
}

// applyProofRejectionPenalty 在审核方驳回证明（要求重做）后调用：递增该任务的连续驳回计数，
// 从第 3 次起扣分并通过房间系统提示让双方都能看到扣除情况。全游戏通用（走同一套 onPunishmentReview）。
func (s *Server) applyProofRejectionPenalty(room *RoomState, punishedPlayerID string) {
	task := latestPunishmentTask(room, punishedPlayerID)
	if task == nil {
		return
	}
	task.RejectCount++
	penalty := proofRejectionPenaltyPoints(task.RejectCount)
	if penalty <= 0 {
		return
	}
	punished := s.players[punishedPlayerID]
	if punished == nil {
		return
	}
	s.updateRankedPoints(punished, -penalty)
	s.roomNotice(room, fmt.Sprintf(
		"%s 的惩罚任务已连续第 %d 次审核不通过，系统额外扣除其 %d 积分。",
		playerShortName(punished), task.RejectCount, penalty,
	))
}

func (s *Server) updatePunishmentTask(room *RoomState, playerID, taskText string, assignedBy *PlayerState) {
	if len(room.RoundHistory) == 0 {
		return
	}
	latest := &room.RoundHistory[0]
	for i := range latest.PunishmentTasks {
		if latest.PunishmentTasks[i].PlayerID == playerID {
			latest.PunishmentTasks[i].TaskText = taskText
			if assignedBy != nil {
				latest.PunishmentTasks[i].AssignedBy = assignedBy.ID
				latest.PunishmentTasks[i].AssignedByName = assignedBy.Name
			}
			return
		}
	}
}

func (s *Server) oppositeForgiveProof(room *RoomState, reviewerID, targetID string) *types.PunishmentProof {
	for i := range room.Proofs {
		proof := &room.Proofs[i]
		if proof.PlayerID == reviewerID && proof.Status == "approved" &&
			proof.ReviewedBy == targetID && proof.RejectReason == "对方选择放过你" {
			return proof
		}
	}
	return nil
}

func (s *Server) applyForgiveReview(room *RoomState, reviewerID, targetID string) string {
	opposite := s.oppositeForgiveProof(room, reviewerID, targetID)
	if opposite == nil {
		room.ForgiveAdvantage = &forgiveAdvantage{BeneficiaryID: reviewerID, TargetID: targetID}
		return "对方选择放过你"
	}
	room.ForgiveAdvantage = nil
	opposite.RejectReason = "双方互相放过，下一局正常开始。"
	s.updateProofInLatestHistory(room, reviewerID, types.HistoryProof{RejectReason: "双方互相放过，下一局正常开始。"})
	return "双方互相放过，下一局正常开始。"
}

func (s *Server) setupPunishmentOrNext(room *RoomState, result types.RoundResult) {
	if !room.Settings.EnablePunishment {
		return
	}
	humanIDs := make([]string, 0)
	for _, p := range s.punishmentPlayersForResult(room, result) {
		humanIDs = append(humanIDs, p.ID)
	}
	s.setupPunishmentForPlayers(room, humanIDs)
}

// setupPunishmentForPlayers：进入惩罚阶段的公共尾段（不依赖 Seat/RoundResult，
// 直接吃 playerID 列表）——大话骰（不进 Seats 体系）和其它三个游戏共用这一段。
func (s *Server) setupPunishmentForPlayers(room *RoomState, humanIDs []string) {
	if !room.Settings.EnablePunishment || len(humanIDs) == 0 {
		return
	}
	room.Phase = types.PhasePunishment
	room.Status = "punishment"
	room.PunishedPlayerIDs = humanIDs
	room.LockedSeatIDs = map[string]struct{}{}
	for _, playerID := range humanIDs {
		room.LockedSeatIDs[playerID] = struct{}{}
		if player := s.players[playerID]; player != nil {
			player.Stats.Punishments++
		}
		if seat, ok := s.seatOf(room, playerID); ok {
			ss := room.SeatStats[seat]
			ss.Punishments++
			room.SeatStats[seat] = ss
		}
	}
}

func (s *Server) punishmentComplete(room *RoomState) bool {
	for _, playerID := range room.PunishedPlayerIDs {
		var task *types.PunishmentTask
		if len(room.RoundHistory) > 0 {
			for i := range room.RoundHistory[0].PunishmentTasks {
				if room.RoundHistory[0].PunishmentTasks[i].PlayerID == playerID {
					task = &room.RoundHistory[0].PunishmentTasks[i]
					break
				}
			}
		}
		if room.Settings.PunishmentSource == "player" && (task == nil || strings.TrimSpace(task.TaskText) == "") {
			return false
		}
		var proof *types.PunishmentProof
		for i := range room.Proofs {
			if room.Proofs[i].PlayerID == playerID {
				proof = &room.Proofs[i]
				break
			}
		}
		if proof == nil || proof.Status == "rejected" {
			return false
		}
		// 仅 approved（或带 ConfirmedBy）算完成。pending 必须等胜方审批；
		// 关闭「需对手确认」时提交路径会立刻写成 approved。
		if proof.Status != "approved" && proof.ConfirmedBy == "" {
			return false
		}
	}
	return true
}

func (s *Server) humanOpponent(room *RoomState, playerID string) *PlayerState {
	seat, ok := s.seatOf(room, playerID)
	if !ok {
		return nil
	}
	return s.humanPlayerFromSeat(room, oppositeSeat(seat))
}

func proofNeedsReview(proof types.PunishmentProof) bool {
	return proof.Status == "pending" || proof.Status == "rejected"
}

func (s *Server) canReviewPlayer(room *RoomState, reviewerID, targetID string) bool {
	if reviewerID == "" || reviewerID == targetID {
		return false
	}
	if room.Settings.GameID == types.GameLiarsDice {
		reviewer := s.liarsDicePunishmentReviewer(room, targetID)
		return reviewer != nil && reviewer.ID == reviewerID
	}
	rs, ok1 := s.seatOf(room, reviewerID)
	ts, ok2 := s.seatOf(room, targetID)
	return ok1 && ok2 && rs != ts
}

func (s *Server) approveProofBySystem(room *RoomState, playerID, message string) bool {
	var proof *types.PunishmentProof
	for i := range room.Proofs {
		if room.Proofs[i].PlayerID == playerID {
			proof = &room.Proofs[i]
			break
		}
	}
	if proof == nil || proof.Status == "approved" {
		return false
	}
	reviewedAt := nowMs()
	proof.Status = "approved"
	proof.ConfirmedBy = "system-auto-forgive"
	proof.ReviewedBy = "system-auto-forgive"
	proof.ReviewedAt = &reviewedAt
	proof.RejectReason = message
	s.updateProofInLatestHistory(room, playerID, types.HistoryProof{
		Status: "approved", ReviewedBy: "system-auto-forgive", ReviewedAt: &reviewedAt, RejectReason: message,
	})
	return true
}

func (s *Server) submitSystemPunishmentProof(room *RoomState, player *PlayerState, message string) {
	var taskText string
	if len(room.RoundHistory) > 0 {
		for _, t := range room.RoundHistory[0].PunishmentTasks {
			if t.PlayerID == player.ID {
				taskText = t.TaskText
				break
			}
		}
	}
	for _, p := range room.Proofs {
		if p.PlayerID == player.ID && p.RedoTaskText != "" {
			taskText = p.RedoTaskText
		}
	}
	submittedAt := nowMs()
	filtered := room.Proofs[:0]
	for _, p := range room.Proofs {
		if p.PlayerID != player.ID {
			filtered = append(filtered, p)
		}
	}
	room.Proofs = append(filtered, types.PunishmentProof{
		PlayerID: player.ID, Text: message, TaskText: taskText, Status: "approved",
		ConfirmedBy: "system-timeout", ReviewedBy: "system-timeout", ReviewedAt: &submittedAt,
		RejectReason: message, SubmittedAt: submittedAt,
	})
	s.attachProofToLatestHistory(room, types.HistoryProof{
		PlayerID: player.ID, PlayerName: playerShortName(player), Text: message, TaskText: taskText,
		Status: "approved", ReviewedBy: "system-timeout", ReviewedAt: &submittedAt,
		RejectReason: message, SubmittedAt: submittedAt,
	})
}

func (s *Server) finishPunishmentIfComplete(room *RoomState) bool {
	if room.Phase == types.PhasePunishment && s.punishmentComplete(room) {
		s.resetForNextRound(room)
		return true
	}
	return false
}

func (s *Server) resetForNextRound(room *RoomState) {
	s.prepareNextChoice(room)
	room.PunishedPlayerIDs = []string{}
	room.Proofs = []types.PunishmentProof{}
	room.LockedSeatIDs = map[string]struct{}{}
	s.broadcastRoom(room.ID, true)
}

func (s *Server) handlePunishmentDeparture(room *RoomState, player *PlayerState, reason LeaveReason) {
	if room.Phase != types.PhasePunishment {
		return
	}
	isPunished := containsString(room.PunishedPlayerIDs, player.ID)
	var latest *types.RoundHistoryItem
	if len(room.RoundHistory) > 0 {
		latest = &room.RoundHistory[0]
	}
	if isPunished && (reason == LeaveDisconnectTimeout || reason == LeaveAdminKick) {
		playerName := playerShortName(player)
		message := fmt.Sprintf("%s 超时未返回，系统已处理本局惩罚。", playerName)
		if reason == LeaveAdminKick {
			message = fmt.Sprintf("%s 被管理员移出，系统已处理本局惩罚。", playerName)
		}
		s.submitSystemPunishmentProof(room, player, message)
		delete(room.LockedSeatIDs, player.ID)
		s.roomNotice(room, message)
		if s.finishPunishmentIfComplete(room) {
			return
		}
	}
	if latest != nil {
		for _, task := range latest.PunishmentTasks {
			if task.AssignedBy == player.ID && strings.TrimSpace(task.TaskText) == "" {
				s.updatePunishmentTask(room, task.PlayerID, "对方已离开，请提交文字说明完成本局惩罚。", nil)
				s.roomNotice(room, fmt.Sprintf("%s 离开，系统已为 %s 发布兜底任务。", playerShortName(player), task.PlayerName))
			}
		}
	}
	for _, proof := range append([]types.PunishmentProof{}, room.Proofs...) {
		if proofNeedsReview(proof) && s.canReviewPlayer(room, player.ID, proof.PlayerID) {
			target := s.players[proof.PlayerID]
			s.approveProofBySystem(room, proof.PlayerID, "审核方离开，系统已自动放过对方。")
			name := "对方"
			if target != nil {
				name = playerShortName(target)
			}
			s.roomNotice(room, fmt.Sprintf("%s 离开，系统已自动放过 %s。", playerShortName(player), name))
		}
	}
	s.finishPunishmentIfComplete(room)
}
