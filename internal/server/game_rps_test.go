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
