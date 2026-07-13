package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestApplyPunishmentPlaceholders(t *testing.T) {
	got := applyPunishmentPlaceholders("{loser} 需要拥抱 {winner}", "小败", "大胜")
	want := "小败 需要拥抱 大胜"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	// 多次出现
	got = applyPunishmentPlaceholders("{loser} 对 {winner} 说：我是 {loser}", "A", "B")
	want = "A 对 B 说：我是 A"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	// 无占位符
	got = applyPunishmentPlaceholders("喝一杯水", "A", "B")
	if got != "喝一杯水" {
		t.Fatalf("got %q", got)
	}
	// 无胜者（平局双罚）时 winner 置空
	got = applyPunishmentPlaceholders("{loser} 请向 {winner} 道歉", "A", "")
	want = "A 请向  道歉"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestWinnerNameForResult(t *testing.T) {
	s := &Server{players: map[string]*PlayerState{}}
	winner := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "w1", Name: "胜者甲"}}
	loser := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "l1", Name: "败者乙"}}
	s.players[winner.ID] = winner
	s.players[loser.ID] = loser
	room := &RoomState{
		Seats: map[types.SeatKey]SeatOccupant{
			types.SeatA: &HumanSeat{Player: winner.PublicPlayer},
			types.SeatB: &HumanSeat{Player: loser.PublicPlayer},
		},
	}
	if got := s.winnerNameForResult(room, types.ResultA); got != "胜者甲" {
		t.Fatalf("ResultA winner=%q", got)
	}
	if got := s.winnerNameForResult(room, types.ResultB); got != "败者乙" {
		t.Fatalf("ResultB winner=%q", got)
	}
	if got := s.winnerNameForResult(room, types.ResultDraw); got != "" {
		t.Fatalf("draw winner should be empty, got %q", got)
	}
	if got := s.winnerNameForResult(room, types.ResultDoubleLoss); got != "" {
		t.Fatalf("doubleLoss winner should be empty, got %q", got)
	}
}

func TestPunishmentTaskForPlayerPlaceholders(t *testing.T) {
	s := &Server{players: map[string]*PlayerState{}}
	winner := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "w1", Name: "Alice", FactionID: "male_faction", FactionLabel: "男性阵营"}}
	loser := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "l1", Name: "Bob", FactionID: "female_faction", FactionLabel: "女性阵营"}}
	s.players[winner.ID] = winner
	s.players[loser.ID] = loser
	room := &RoomState{
		Seats: map[types.SeatKey]SeatOccupant{
			types.SeatA: &HumanSeat{Player: winner.PublicPlayer},
			types.SeatB: &HumanSeat{Player: loser.PublicPlayer},
		},
	}
	punishment := &types.PunishmentConfig{
		Name: "拥抱",
		Tasks: []types.PunishmentTaskConfig{
			{
				ID:   "t1",
				Name: "默认",
				Variants: map[string]string{
					"female_faction": "{loser} 需要拥抱 {winner}",
					"male_faction":   "{loser} 需要拥抱 {winner}",
				},
			},
		},
	}
	// B 败（ResultA 表示 A 胜）
	res := s.punishmentTaskForPlayer(room, loser, types.ResultA, punishment)
	if res == nil || res.TaskText != "Bob 需要拥抱 Alice" {
		t.Fatalf("task=%#v", res)
	}
}
