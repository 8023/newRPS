package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

// TestApplyForgiveAdvantageNeverOverridesDraw 锁定回归：即使有待生效的"放过对方"
// 命运安排，真实平局也绝不能被强改成某一方胜利（历史上这里漏排除了 ResultDraw，
// 导致双方出拳相同却被判某方胜利、双罚规则被跳过）。
func TestApplyForgiveAdvantageNeverOverridesDraw(t *testing.T) {
	s := newTestServer(t)
	room := &RoomState{
		Seats: map[types.SeatKey]SeatOccupant{
			types.SeatA: &HumanSeat{Player: types.PublicPlayer{ID: "playerA"}},
			types.SeatB: &HumanSeat{Player: types.PublicPlayer{ID: "playerB"}},
		},
	}

	for i := 0; i < 200; i++ {
		room.ForgiveAdvantage = &forgiveAdvantage{BeneficiaryID: "playerA", TargetID: "playerB"}
		result, outcome := s.applyForgiveAdvantage(room, types.ResultDraw)
		if result != types.ResultDraw {
			t.Fatalf("iteration %d: applyForgiveAdvantage overrode a draw, got result %v", i, result)
		}
		if outcome != nil {
			t.Fatalf("iteration %d: applyForgiveAdvantage returned non-nil outcome for a draw", i)
		}
		if room.ForgiveAdvantage != nil {
			t.Fatalf("iteration %d: room.ForgiveAdvantage should be consumed after resolving a round", i)
		}
	}
}

// TestApplyForgiveAdvantageCanOverrideNonDraw 确认非平局情况下机制仍按设计工作：
// 多次尝试后应能观察到恩惠方被判获胜（说明该分支未被误伤）。
func TestApplyForgiveAdvantageCanOverrideNonDraw(t *testing.T) {
	s := newTestServer(t)
	overridden := false
	for i := 0; i < 500 && !overridden; i++ {
		room := &RoomState{
			Seats: map[types.SeatKey]SeatOccupant{
				types.SeatA: &HumanSeat{Player: types.PublicPlayer{ID: "playerA"}},
				types.SeatB: &HumanSeat{Player: types.PublicPlayer{ID: "playerB"}},
			},
			ForgiveAdvantage: &forgiveAdvantage{BeneficiaryID: "playerA", TargetID: "playerB"},
		}
		result, outcome := s.applyForgiveAdvantage(room, types.ResultB)
		if outcome != nil {
			if result != types.ResultA {
				t.Fatalf("outcome reported but result %v does not favor beneficiary seat A", result)
			}
			overridden = true
		}
	}
	if !overridden {
		t.Fatal("applyForgiveAdvantage never overrode a non-draw result across 500 attempts (66% chance each) — mechanism may be broken")
	}
}

// TestShouldTriggerGiveawayUsesOnlyPlayerProbability 锁定主人强制不再污染玩家全局状态；
// 此函数只根据玩家是否开启白给及其白给值判断自然触发。
func TestShouldTriggerGiveawayUsesOnlyPlayerProbability(t *testing.T) {
	s := newTestServer(t)
	player := &PlayerState{PublicPlayer: types.PublicPlayer{
		GiveawayValue: floatPtr(0), GiveawayEnabled: boolPtr(true),
	}}
	if s.shouldTriggerGiveaway(player) {
		t.Fatal("GiveawayValue=0 时不应自然触发白给")
	}
	player.GiveawayValue = floatPtr(100)
	if !s.shouldTriggerGiveaway(player) {
		t.Fatal("GiveawayValue=100 时应自然触发白给")
	}
	player.GiveawayEnabled = boolPtr(false)
	if s.shouldTriggerGiveaway(player) {
		t.Fatal("白给模式关闭后不应自然触发")
	}
}

func TestResultWithGiveawayBothPlayersDraw(t *testing.T) {
	s := newTestServer(t)
	room := &RoomState{ForgiveAdvantage: &forgiveAdvantage{BeneficiaryID: "A", TargetID: "B"}}
	result, outcome := s.resultWithGiveaway(room, types.ResultDraw, map[types.SeatKey]types.Move{
		types.SeatA: types.MoveGiveaway,
		types.SeatB: types.MoveGiveaway,
	})
	if result != types.ResultDraw {
		t.Fatalf("双方白给应为平局，got %v", result)
	}
	if outcome != nil || room.ForgiveAdvantage != nil {
		t.Fatal("双方白给结算时不应保留命运安排")
	}
}
