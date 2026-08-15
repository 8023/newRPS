package server

import (
	"reflect"
	"testing"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

func newTurnBasedUndoTestRoom(gameID types.GameID) (*Server, *RoomState) {
	playerA := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "a", Name: "甲"}}
	playerB := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "b", Name: "乙"}}
	room := &RoomState{
		ID: "undo-room", Settings: types.RoomSettings{GameID: gameID, Stake: 1},
		Phase: types.PhaseChoosing, Status: "playing",
		Seats: map[types.SeatKey]SeatOccupant{
			types.SeatA: &HumanSeat{Player: playerA.PublicPlayer},
			types.SeatB: &HumanSeat{Player: playerB.PublicPlayer},
		},
		Ready: map[types.SeatKey]bool{types.SeatA: true, types.SeatB: true},
	}
	server := &Server{
		players:              map[string]*PlayerState{"a": playerA, "b": playerB},
		rooms:                map[string]*RoomState{room.ID: room},
		clients:              map[string]*Client{},
		roomClients:          map[string]map[string]struct{}{},
		turnBasedUndoTimers:  map[string]*time.Timer{},
		turnBasedClockTimers: map[string]*turnBasedClockTimer{},
		roomBroadcastTimers:  map[string]*roomBroadcastPending{},
	}
	return server, room
}

func TestJungleUndoRestoresCapturedPieces(t *testing.T) {
	s, room := newTurnBasedUndoTestRoom(types.GameJungle)
	s.mu.Lock()
	defer s.mu.Unlock()
	room.Settings.JungleUndoLimit = 1
	room.Jungle = freshJungleState(types.SeatA)
	board := make([][]*types.JungleCell, jungleRows)
	for row := range board {
		board[row] = make([]*types.JungleCell, jungleCols)
	}
	placeJungle(board, 4, 0, types.SeatA, types.JungleRat)
	placeJungle(board, 3, 0, types.SeatB, types.JungleElephant)
	placeJungle(board, 0, 0, types.SeatB, types.JungleCat)
	room.Jungle.Board = board
	wantBoard := cloneJungleBoard(board)

	// Given: A 可吃掉 B 的象，B 随后有一手合法应手。
	// When: 双方各走一手，A 申请悔棋且 B 同意。
	if ok, msg := s.applyJungleMove(room, types.SeatA, 4, 0, 3, 0); !ok {
		t.Fatalf("first move failed: %s", msg)
	}
	if ok, msg := s.applyJungleMove(room, types.SeatB, 0, 0, 1, 0); !ok {
		t.Fatalf("second move failed: %s", msg)
	}
	if ok, msg := s.requestJungleUndo(room, types.SeatA); !ok {
		t.Fatalf("undo request failed: %s", msg)
	}
	if ok, msg := s.respondJungleUndo(room, types.SeatB, true); !ok {
		t.Fatalf("undo response failed: %s", msg)
	}

	// Then: 两手全部回退，被吃的象和移动的猫都回到原位。
	if !reflect.DeepEqual(room.Jungle.Board, wantBoard) {
		t.Fatal("jungle board was not restored")
	}
	if room.Jungle.MoveCount != 0 || room.Jungle.Turn != types.SeatA {
		t.Fatalf("moveCount=%d turn=%s, want 0/A", room.Jungle.MoveCount, room.Jungle.Turn)
	}
	if room.Jungle.UndoCount[types.SeatA] != 1 || len(room.jungleMoves) != 0 {
		t.Fatalf("undoCount=%d history=%d, want 1/0", room.Jungle.UndoCount[types.SeatA], len(room.jungleMoves))
	}
}

func TestChessUndoRestoresPositionAndRepetition(t *testing.T) {
	s, room := newTurnBasedUndoTestRoom(types.GameChess)
	s.mu.Lock()
	defer s.mu.Unlock()
	room.Settings.ChessUndoLimit = 1
	room.Chess = freshChessState(types.SeatA)
	room.chessRepetition = []string{chessPositionKey(room.Chess)}
	wantBoard := cloneChessBoard(room.Chess.Board)

	// Given: 白方 e2-e4、黑方 e7-e5 已形成两手历史。
	// When: 轮到白方时申请悔棋，黑方同意。
	if ok, msg := s.applyChessMove(room, types.SeatA, 6, 4, 4, 4, ""); !ok {
		t.Fatalf("first move failed: %s", msg)
	}
	if ok, msg := s.applyChessMove(room, types.SeatB, 1, 4, 3, 4, ""); !ok {
		t.Fatalf("second move failed: %s", msg)
	}
	if ok, msg := s.requestChessUndo(room, types.SeatA); !ok {
		t.Fatalf("undo request failed: %s", msg)
	}
	if ok, msg := s.respondChessUndo(room, types.SeatB, true); !ok {
		t.Fatalf("undo response failed: %s", msg)
	}

	// Then: 初始局面、易位权与三次重复历史都完整恢复。
	if !reflect.DeepEqual(room.Chess.Board, wantBoard) {
		t.Fatal("chess board was not restored")
	}
	if room.Chess.MoveCount != 0 || room.Chess.Turn != types.SeatA {
		t.Fatalf("moveCount=%d turn=%s, want 0/A", room.Chess.MoveCount, room.Chess.Turn)
	}
	if !room.Chess.CastlingWhiteK || !room.Chess.CastlingWhiteQ || !room.Chess.CastlingBlackK || !room.Chess.CastlingBlackQ {
		t.Fatal("castling rights were not restored")
	}
	if len(room.chessRepetition) != 1 || len(room.chessUndoStack) != 0 {
		t.Fatalf("repetition=%d history=%d, want 1/0", len(room.chessRepetition), len(room.chessUndoStack))
	}
}

func TestOthelloUndoRestoresBoardAndRankedPoints(t *testing.T) {
	s, room := newTurnBasedUndoTestRoom(types.GameOthello)
	s.mu.Lock()
	defer func() {
		if s.playerUpdateTimer != nil {
			s.playerUpdateTimer.Stop()
		}
		s.mu.Unlock()
	}()
	room.Settings.OthelloUndoLimit = 1
	room.Settings.EnableRanked = true
	room.Othello = s.freshOthelloState(types.SeatA)
	board := make([][]*types.OthelloCell, 8)
	for row := range board {
		board[row] = make([]*types.OthelloCell, 8)
	}
	black, white := types.CellBlack, types.CellWhite
	board[3][1], board[3][2], board[3][3] = &white, &white, &black
	board[5][1], board[5][2] = &black, &white
	board[7][1], board[7][2] = &white, &black
	room.Othello.Board = board
	room.Othello.LegalMoves = othelloLegalMoves(board, black)
	room.Othello.BlackCount, room.Othello.WhiteCount = othelloCounts(board)
	wantBoard := cloneOthelloBoard(board)

	// Given: A 一手翻 2 子，B 一手翻 1 子，两手都已实时结算排位分。
	// When: 轮到 A 时申请悔棋，B 同意。
	if ok, msg := s.applyOthelloMove(room, types.SeatA, 3, 0); !ok {
		t.Fatalf("first move failed: %s", msg)
	}
	if ok, msg := s.applyOthelloMove(room, types.SeatB, 5, 0); !ok {
		t.Fatalf("second move failed: %s", msg)
	}
	if room.Othello.RankedDelta[types.SeatA] != 1 || room.Othello.RankedDelta[types.SeatB] != -1 {
		t.Fatalf("ranked delta before undo=%v, want A:+1 B:-1", room.Othello.RankedDelta)
	}
	if ok, msg := s.requestOthelloUndo(room, types.SeatA); !ok {
		t.Fatalf("undo request failed: %s", msg)
	}
	if ok, msg := s.respondOthelloUndo(room, types.SeatB, true); !ok {
		t.Fatalf("undo response failed: %s", msg)
	}

	// Then: 棋盘、实时排位分与玩家真实分全部回到两手之前。
	if !reflect.DeepEqual(room.Othello.Board, wantBoard) {
		t.Fatal("othello board was not restored")
	}
	if room.Othello.RankedDelta[types.SeatA] != 0 || room.Othello.RankedDelta[types.SeatB] != 0 {
		t.Fatalf("ranked delta after undo=%v, want zero", room.Othello.RankedDelta)
	}
	if s.players["a"].Stats.RankedPoints != 0 || s.players["b"].Stats.RankedPoints != 0 {
		t.Fatalf("player points=%d/%d, want 0/0", s.players["a"].Stats.RankedPoints, s.players["b"].Stats.RankedPoints)
	}
	if room.Othello.UndoCount[types.SeatA] != 1 || len(room.othelloUndoStack) != 0 {
		t.Fatalf("undoCount=%d history=%d, want 1/0", room.Othello.UndoCount[types.SeatA], len(room.othelloUndoStack))
	}
}
