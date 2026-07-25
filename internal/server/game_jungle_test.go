package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestJungleInitialSetup(t *testing.T) {
	board := initialJungleBoard()
	// A rat bottom-left area, B elephant top-left facing it
	if board[6][0] == nil || *board[6][0] != jungleCellOf(types.SeatA, types.JungleRat) {
		t.Fatalf("A rat expected at 6,0 got %v", board[6][0])
	}
	if board[2][0] == nil || *board[2][0] != jungleCellOf(types.SeatB, types.JungleElephant) {
		t.Fatalf("B elephant expected at 2,0")
	}
	if board[8][6] == nil || *board[8][6] != jungleCellOf(types.SeatA, types.JungleLion) {
		t.Fatalf("A lion expected at 8,6 (right hand)")
	}
	if !jungleIsWater(3, 1) || jungleIsWater(3, 3) {
		t.Fatalf("water layout wrong")
	}
	if !jungleIsDen(8, 3, types.SeatA) || !jungleIsDen(0, 3, types.SeatB) {
		t.Fatalf("den layout wrong")
	}
}

func TestJungleRatEatsElephantButNotFromWater(t *testing.T) {
	board := make([][]*types.JungleCell, jungleRows)
	for r := 0; r < jungleRows; r++ {
		board[r] = make([]*types.JungleCell, jungleCols)
	}
	placeJungle(board, 3, 0, types.SeatA, types.JungleRat) // land
	placeJungle(board, 3, 3, types.SeatB, types.JungleElephant)
	// Adjacent land: rat at 4,3 water next to elephant at 4,0 land? put clearer
	board = make([][]*types.JungleCell, jungleRows)
	for r := 0; r < jungleRows; r++ {
		board[r] = make([]*types.JungleCell, jungleCols)
	}
	placeJungle(board, 4, 0, types.SeatA, types.JungleRat)
	placeJungle(board, 4, 1, types.SeatB, types.JungleElephant) // water? col1 row4 is water — put elephant on land
	board[4][1] = nil
	placeJungle(board, 5, 0, types.SeatB, types.JungleElephant)
	// rat land can eat elephant
	if !jungleCanCapture(types.SeatA, types.JungleRat, 4, 0, 5, 0, types.JungleElephant) {
		t.Fatal("rat should eat elephant from land")
	}
	// elephant cannot eat rat
	if jungleCanCapture(types.SeatB, types.JungleElephant, 5, 0, 4, 0, types.JungleRat) {
		t.Fatal("elephant should not eat rat")
	}
	// rat in water cannot land-eat elephant
	if jungleCanCapture(types.SeatA, types.JungleRat, 4, 1, 4, 0, types.JungleElephant) {
		t.Fatal("rat from water should not eat elephant")
	}
}

func TestJungleTrapWeakens(t *testing.T) {
	// B piece on A's trap can be eaten by A rat
	if !jungleCanCapture(types.SeatA, types.JungleRat, 7, 2, 7, 3, types.JungleElephant) {
		// 7,3 is A trap
		t.Fatal("any piece should capture defender in own trap")
	}
	if jungleIsTrap(7, 3, types.SeatA) != true {
		t.Fatal("7,3 should be A trap")
	}
}

func TestJungleLionJumpBlockedByRat(t *testing.T) {
	board := make([][]*types.JungleCell, jungleRows)
	for r := 0; r < jungleRows; r++ {
		board[r] = make([]*types.JungleCell, jungleCols)
	}
	placeJungle(board, 2, 1, types.SeatA, types.JungleLion)
	// jump down over water cols 1 rows 3-5 to land 6,1
	dest, ok := jungleJumpDest(board, 2, 1, 1, 0)
	if !ok || dest.Row != 6 || dest.Col != 1 {
		t.Fatalf("expected jump to 6,1 got %v ok=%v", dest, ok)
	}
	placeJungle(board, 4, 1, types.SeatB, types.JungleRat)
	if _, ok := jungleJumpDest(board, 2, 1, 1, 0); ok {
		t.Fatal("jump should be blocked by rat in river")
	}
}

func TestJungleCannotEnterOwnDen(t *testing.T) {
	board := make([][]*types.JungleCell, jungleRows)
	for r := 0; r < jungleRows; r++ {
		board[r] = make([]*types.JungleCell, jungleCols)
	}
	placeJungle(board, 8, 2, types.SeatA, types.JungleDog)
	if jungleCanMoveTo(board, types.SeatA, types.JungleDog, 8, 2, 8, 3, false) {
		t.Fatal("cannot enter own den")
	}
}

func TestJungleNonRatCannotEnterWater(t *testing.T) {
	board := make([][]*types.JungleCell, jungleRows)
	for r := 0; r < jungleRows; r++ {
		board[r] = make([]*types.JungleCell, jungleCols)
	}
	placeJungle(board, 3, 0, types.SeatA, types.JungleDog)
	if jungleCanMoveTo(board, types.SeatA, types.JungleDog, 3, 0, 3, 1, false) {
		t.Fatal("dog cannot enter water")
	}
	placeJungle(board, 3, 0, types.SeatA, types.JungleRat)
	board[3][0] = nil
	placeJungle(board, 3, 0, types.SeatA, types.JungleRat)
	if !jungleCanMoveTo(board, types.SeatA, types.JungleRat, 3, 0, 3, 1, false) {
		t.Fatal("rat should enter water")
	}
}

func TestJungleLegalMoveFromStart(t *testing.T) {
	board := initialJungleBoard()
	// A rat at 6,0 can move to 5,0
	dests := jungleLegalDests(board, types.SeatA, 6, 0)
	found := false
	for _, d := range dests {
		if d.Row == 5 && d.Col == 0 {
			found = true
		}
	}
	if !found {
		t.Fatalf("A rat should move to 5,0, dests=%v", dests)
	}
}
