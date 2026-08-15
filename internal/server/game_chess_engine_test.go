package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func emptyChessBoard() [][]*types.ChessCell {
	board := make([][]*types.ChessCell, chessSize)
	for r := 0; r < chessSize; r++ {
		board[r] = make([]*types.ChessCell, chessSize)
	}
	return board
}

func TestChessStartingMoves(t *testing.T) {
	state := freshChessState(types.SeatA)
	moves := chessLegalMoves(state)
	if len(moves) != 20 {
		t.Fatalf("starting legal moves = %d, want 20", len(moves))
	}
	if chessInCheck(state.Board, types.ChessWhite) || chessInCheck(state.Board, types.ChessBlack) {
		t.Fatal("starting position should not be in check")
	}
}

func TestChessFoolsMate(t *testing.T) {
	state := freshChessState(types.SeatA)
	// 1. f3 e5 2. g4 Qh4#
	seq := []types.ChessMove{
		{From: types.Pos{Row: 6, Col: 5}, To: types.Pos{Row: 5, Col: 5}},
		{From: types.Pos{Row: 1, Col: 4}, To: types.Pos{Row: 3, Col: 4}},
		{From: types.Pos{Row: 6, Col: 6}, To: types.Pos{Row: 4, Col: 6}},
		{From: types.Pos{Row: 0, Col: 3}, To: types.Pos{Row: 4, Col: 7}},
	}
	rep := []string{chessPositionKey(state)}
	for i, mv := range seq {
		if _, ok := chessFindLegal(state, mv); !ok {
			t.Fatalf("move %d %+v not legal", i, mv)
		}
		chessApplyMoveToState(state, mv)
		rep = append(rep, chessPositionKey(state))
	}
	if chessEvaluateOutcome(state, rep) != chessCheckmate {
		t.Fatalf("expected checkmate after fool's mate, inCheck=%v moves=%d", state.InCheck, len(state.LegalMoves))
	}
}

func TestChessCastlingKingside(t *testing.T) {
	board := emptyChessBoard()
	placeChess(board, 7, 4, types.ChessWhite, types.ChessKing)
	placeChess(board, 7, 7, types.ChessWhite, types.ChessRook)
	placeChess(board, 0, 4, types.ChessBlack, types.ChessKing)
	state := &types.ChessState{
		Board: board, Turn: types.SeatA, WhiteSeat: types.SeatA,
		CastlingWhiteK: true, CastlingWhiteQ: true, CastlingBlackK: true, CastlingBlackQ: true,
	}
	state.LegalMoves = chessLegalMoves(state)
	castle := types.ChessMove{From: types.Pos{Row: 7, Col: 4}, To: types.Pos{Row: 7, Col: 6}}
	if _, ok := chessFindLegal(state, castle); !ok {
		t.Fatal("kingside castle should be legal")
	}
	chessApplyMoveToState(state, castle)
	if _, piece, ok := parseChessCellPtr(state.Board[7][6]); !ok || piece != types.ChessKing {
		t.Fatal("king should land on g1")
	}
	if _, piece, ok := parseChessCellPtr(state.Board[7][5]); !ok || piece != types.ChessRook {
		t.Fatal("rook should land on f1")
	}
	if state.CastlingWhiteK || state.CastlingWhiteQ {
		t.Fatal("castling rights should be gone after castle")
	}
}

func TestChessCannotCastleThroughCheck(t *testing.T) {
	board := emptyChessBoard()
	placeChess(board, 7, 4, types.ChessWhite, types.ChessKing)
	placeChess(board, 7, 7, types.ChessWhite, types.ChessRook)
	placeChess(board, 0, 4, types.ChessBlack, types.ChessKing)
	placeChess(board, 5, 5, types.ChessBlack, types.ChessRook) // attacks f1
	state := &types.ChessState{
		Board: board, Turn: types.SeatA, WhiteSeat: types.SeatA,
		CastlingWhiteK: true, CastlingBlackK: true, CastlingBlackQ: true,
	}
	state.LegalMoves = chessLegalMoves(state)
	castle := types.ChessMove{From: types.Pos{Row: 7, Col: 4}, To: types.Pos{Row: 7, Col: 6}}
	if _, ok := chessFindLegal(state, castle); ok {
		t.Fatal("cannot castle through check")
	}
}

func TestChessEnPassant(t *testing.T) {
	board := emptyChessBoard()
	placeChess(board, 7, 4, types.ChessWhite, types.ChessKing)
	placeChess(board, 0, 4, types.ChessBlack, types.ChessKing)
	placeChess(board, 3, 4, types.ChessWhite, types.ChessPawn)
	placeChess(board, 1, 3, types.ChessBlack, types.ChessPawn)
	state := &types.ChessState{Board: board, Turn: types.SeatB, WhiteSeat: types.SeatA}
	state.LegalMoves = chessLegalMoves(state)
	double := types.ChessMove{From: types.Pos{Row: 1, Col: 3}, To: types.Pos{Row: 3, Col: 3}}
	if _, ok := chessFindLegal(state, double); !ok {
		t.Fatal("black double pawn should be legal")
	}
	chessApplyMoveToState(state, double)
	if state.EnPassant == nil || state.EnPassant.Row != 2 || state.EnPassant.Col != 3 {
		t.Fatalf("en passant square = %+v", state.EnPassant)
	}
	ep := types.ChessMove{From: types.Pos{Row: 3, Col: 4}, To: types.Pos{Row: 2, Col: 3}}
	if _, ok := chessFindLegal(state, ep); !ok {
		t.Fatal("en passant capture should be legal")
	}
	chessApplyMoveToState(state, ep)
	if state.Board[3][3] != nil {
		t.Fatal("captured pawn should be removed")
	}
	if _, piece, ok := parseChessCellPtr(state.Board[2][3]); !ok || piece != types.ChessPawn {
		t.Fatal("capturing pawn should land on ep square")
	}
}

func TestChessPromotion(t *testing.T) {
	board := emptyChessBoard()
	placeChess(board, 7, 4, types.ChessWhite, types.ChessKing)
	placeChess(board, 0, 4, types.ChessBlack, types.ChessKing)
	placeChess(board, 1, 0, types.ChessWhite, types.ChessPawn)
	state := &types.ChessState{Board: board, Turn: types.SeatA, WhiteSeat: types.SeatA}
	state.LegalMoves = chessLegalMoves(state)
	plain := types.ChessMove{From: types.Pos{Row: 1, Col: 0}, To: types.Pos{Row: 0, Col: 0}}
	if _, ok := chessFindLegal(state, plain); ok {
		t.Fatal("promotion without piece should be illegal")
	}
	queen := types.ChessMove{From: types.Pos{Row: 1, Col: 0}, To: types.Pos{Row: 0, Col: 0}, Promote: "queen"}
	if _, ok := chessFindLegal(state, queen); !ok {
		t.Fatal("promotion to queen should be legal")
	}
	chessApplyMoveToState(state, queen)
	if _, piece, ok := parseChessCellPtr(state.Board[0][0]); !ok || piece != types.ChessQueen {
		t.Fatal("pawn should become queen")
	}
}

func TestChessStalemate(t *testing.T) {
	// 白王 c6、白兵 c7，黑王 c8，黑走：无子可走且未被将军。
	board := emptyChessBoard()
	placeChess(board, 2, 2, types.ChessWhite, types.ChessKing)
	placeChess(board, 1, 2, types.ChessWhite, types.ChessPawn)
	placeChess(board, 0, 2, types.ChessBlack, types.ChessKing)
	state := &types.ChessState{Board: board, Turn: types.SeatB, WhiteSeat: types.SeatA}
	state.LegalMoves = chessLegalMoves(state)
	state.InCheck = chessInCheck(state.Board, types.ChessBlack)
	if out := chessEvaluateOutcome(state, nil); out != chessStalemate {
		t.Fatalf("outcome=%v inCheck=%v moves=%d", out, state.InCheck, len(state.LegalMoves))
	}
}

func TestChessInsufficientMaterial(t *testing.T) {
	board := emptyChessBoard()
	placeChess(board, 7, 4, types.ChessWhite, types.ChessKing)
	placeChess(board, 0, 4, types.ChessBlack, types.ChessKing)
	placeChess(board, 3, 3, types.ChessWhite, types.ChessBishop)
	if !chessInsufficientMaterial(board) {
		t.Fatal("K+B vs K should be insufficient")
	}
	placeChess(board, 4, 4, types.ChessBlack, types.ChessQueen)
	if chessInsufficientMaterial(board) {
		t.Fatal("queen on board is sufficient")
	}
}

func TestChessCanPossiblyMate(t *testing.T) {
	board := emptyChessBoard()
	placeChess(board, 7, 4, types.ChessWhite, types.ChessKing)
	placeChess(board, 0, 4, types.ChessBlack, types.ChessKing)
	if chessCanPossiblyMate(board, types.ChessWhite) || chessCanPossiblyMate(board, types.ChessBlack) {
		t.Fatal("K vs K cannot mate")
	}
	placeChess(board, 3, 3, types.ChessWhite, types.ChessBishop)
	if chessCanPossiblyMate(board, types.ChessWhite) || chessCanPossiblyMate(board, types.ChessBlack) {
		t.Fatal("K+B vs K cannot mate either side")
	}
	placeChess(board, 4, 4, types.ChessWhite, types.ChessQueen)
	if !chessCanPossiblyMate(board, types.ChessWhite) {
		t.Fatal("side with queen can mate")
	}
	if chessCanPossiblyMate(board, types.ChessBlack) {
		t.Fatal("lone king still cannot mate against queen")
	}
}

func TestChessEnPassantClearedWhenUncapturable(t *testing.T) {
	board := emptyChessBoard()
	placeChess(board, 7, 4, types.ChessWhite, types.ChessKing)
	placeChess(board, 0, 4, types.ChessBlack, types.ChessKing)
	placeChess(board, 1, 0, types.ChessBlack, types.ChessPawn)
	state := &types.ChessState{Board: board, Turn: types.SeatB, WhiteSeat: types.SeatA}
	state.LegalMoves = chessLegalMoves(state)
	double := types.ChessMove{From: types.Pos{Row: 1, Col: 0}, To: types.Pos{Row: 3, Col: 0}}
	if _, ok := chessFindLegal(state, double); !ok {
		t.Fatal("black double pawn should be legal")
	}
	chessApplyMoveToState(state, double)
	if state.EnPassant != nil {
		t.Fatalf("uncapturable en passant should be cleared, got %+v", state.EnPassant)
	}
}

func TestChessCannotCastleWithNonRookOnCorner(t *testing.T) {
	board := emptyChessBoard()
	placeChess(board, 7, 4, types.ChessWhite, types.ChessKing)
	placeChess(board, 7, 7, types.ChessWhite, types.ChessQueen)
	placeChess(board, 0, 4, types.ChessBlack, types.ChessKing)
	state := &types.ChessState{
		Board: board, Turn: types.SeatA, WhiteSeat: types.SeatA,
		CastlingWhiteK: true,
	}
	state.LegalMoves = chessLegalMoves(state)
	castle := types.ChessMove{From: types.Pos{Row: 7, Col: 4}, To: types.Pos{Row: 7, Col: 6}}
	if _, ok := chessFindLegal(state, castle); ok {
		t.Fatal("cannot castle when h1 is not a rook")
	}
}

func TestChessPinCannotExposeKing(t *testing.T) {
	board := emptyChessBoard()
	placeChess(board, 7, 4, types.ChessWhite, types.ChessKing)
	placeChess(board, 6, 4, types.ChessWhite, types.ChessKnight)
	placeChess(board, 0, 4, types.ChessBlack, types.ChessRook)
	placeChess(board, 0, 0, types.ChessBlack, types.ChessKing)
	state := &types.ChessState{Board: board, Turn: types.SeatA, WhiteSeat: types.SeatA}
	state.LegalMoves = chessLegalMoves(state)
	// 马被钉住，不能离开 e 线。
	for _, mv := range state.LegalMoves {
		if mv.From.Row == 6 && mv.From.Col == 4 && mv.To.Col != 4 {
			t.Fatalf("pinned knight moved off file: %+v", mv)
		}
	}
}
