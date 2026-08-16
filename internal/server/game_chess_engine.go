package server

import (
	"fmt"
	"strings"

	"github.com/doumiao/newRPS/internal/types"
)

const chessSize = 8

var (
	chessKnightDeltas = [][2]int{{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2}, {1, -2}, {1, 2}, {2, -1}, {2, 1}}
	chessKingDeltas   = [][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}, {0, 1}, {1, -1}, {1, 0}, {1, 1}}
	chessBishopRays   = [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}
	chessRookRays     = [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
)

func chessInBounds(row, col int) bool {
	return row >= 0 && row < chessSize && col >= 0 && col < chessSize
}

func parseChessCell(cell types.ChessCell) (types.ChessColor, types.ChessPiece, bool) {
	parts := strings.SplitN(string(cell), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	color := types.ChessColor(parts[0])
	piece := types.ChessPiece(parts[1])
	if color != types.ChessWhite && color != types.ChessBlack {
		return "", "", false
	}
	switch piece {
	case types.ChessKing, types.ChessQueen, types.ChessRook, types.ChessBishop, types.ChessKnight, types.ChessPawn:
		return color, piece, true
	default:
		return "", "", false
	}
}

func chessCellOf(color types.ChessColor, piece types.ChessPiece) types.ChessCell {
	return types.ChessCell(string(color) + ":" + string(piece))
}

func oppositeChessColor(color types.ChessColor) types.ChessColor {
	if color == types.ChessWhite {
		return types.ChessBlack
	}
	return types.ChessWhite
}

func chessColorForSeat(state *types.ChessState, seat types.SeatKey) types.ChessColor {
	if state.WhiteSeat == seat {
		return types.ChessWhite
	}
	return types.ChessBlack
}

func chessSeatForColor(state *types.ChessState, color types.ChessColor) types.SeatKey {
	if color == types.ChessWhite {
		return state.WhiteSeat
	}
	return oppositeSeat(state.WhiteSeat)
}

func placeChess(board [][]*types.ChessCell, row, col int, color types.ChessColor, piece types.ChessPiece) {
	c := chessCellOf(color, piece)
	board[row][col] = &c
}

func initialChessBoard() [][]*types.ChessCell {
	board := make([][]*types.ChessCell, chessSize)
	for r := 0; r < chessSize; r++ {
		board[r] = make([]*types.ChessCell, chessSize)
	}
	back := []types.ChessPiece{
		types.ChessRook, types.ChessKnight, types.ChessBishop, types.ChessQueen,
		types.ChessKing, types.ChessBishop, types.ChessKnight, types.ChessRook,
	}
	for c := 0; c < chessSize; c++ {
		placeChess(board, 0, c, types.ChessBlack, back[c])
		placeChess(board, 1, c, types.ChessBlack, types.ChessPawn)
		placeChess(board, 6, c, types.ChessWhite, types.ChessPawn)
		placeChess(board, 7, c, types.ChessWhite, back[c])
	}
	return board
}

func cloneChessBoard(board [][]*types.ChessCell) [][]*types.ChessCell {
	out := make([][]*types.ChessCell, len(board))
	for i, row := range board {
		out[i] = make([]*types.ChessCell, len(row))
		for j, cell := range row {
			if cell != nil {
				c := *cell
				out[i][j] = &c
			}
		}
	}
	return out
}

func cloneChessPos(p *types.Pos) *types.Pos {
	if p == nil {
		return nil
	}
	cp := *p
	return &cp
}

func chessPosEqual(a *types.Pos, row, col int) bool {
	return a != nil && a.Row == row && a.Col == col
}

func chessKingSquare(board [][]*types.ChessCell, color types.ChessColor) (int, int, bool) {
	for r := 0; r < chessSize; r++ {
		for c := 0; c < chessSize; c++ {
			if board[r][c] == nil {
				continue
			}
			colr, piece, ok := parseChessCell(*board[r][c])
			if ok && colr == color && piece == types.ChessKing {
				return r, c, true
			}
		}
	}
	return 0, 0, false
}

func chessSquareOccupiedBy(board [][]*types.ChessCell, row, col int, color types.ChessColor) bool {
	if !chessInBounds(row, col) || board[row][col] == nil {
		return false
	}
	colr, _, ok := parseChessCell(*board[row][col])
	return ok && colr == color
}

func chessRayHits(board [][]*types.ChessCell, row, col, dr, dc int, color types.ChessColor, want ...types.ChessPiece) bool {
	r, c := row+dr, col+dc
	for chessInBounds(r, c) {
		if board[r][c] == nil {
			r += dr
			c += dc
			continue
		}
		colr, piece, ok := parseChessCell(*board[r][c])
		if !ok || colr != color {
			return false
		}
		for _, w := range want {
			if piece == w {
				return true
			}
		}
		return false
	}
	return false
}

func chessSquareAttacked(board [][]*types.ChessCell, row, col int, by types.ChessColor) bool {
	// 兵吃子方向：白兵往上（row-1），黑兵往下（row+1）。
	pawnDir := 1
	if by == types.ChessWhite {
		pawnDir = -1
	}
	for _, dc := range []int{-1, 1} {
		r, c := row-pawnDir, col+dc
		if chessInBounds(r, c) && board[r][c] != nil {
			colr, piece, ok := parseChessCell(*board[r][c])
			if ok && colr == by && piece == types.ChessPawn {
				return true
			}
		}
	}
	for _, d := range chessKnightDeltas {
		r, c := row+d[0], col+d[1]
		if chessInBounds(r, c) && board[r][c] != nil {
			colr, piece, ok := parseChessCell(*board[r][c])
			if ok && colr == by && piece == types.ChessKnight {
				return true
			}
		}
	}
	for _, d := range chessKingDeltas {
		r, c := row+d[0], col+d[1]
		if chessInBounds(r, c) && board[r][c] != nil {
			colr, piece, ok := parseChessCell(*board[r][c])
			if ok && colr == by && piece == types.ChessKing {
				return true
			}
		}
	}
	for _, d := range chessBishopRays {
		if chessRayHits(board, row, col, d[0], d[1], by, types.ChessBishop, types.ChessQueen) {
			return true
		}
	}
	for _, d := range chessRookRays {
		if chessRayHits(board, row, col, d[0], d[1], by, types.ChessRook, types.ChessQueen) {
			return true
		}
	}
	return false
}

func chessInCheck(board [][]*types.ChessCell, color types.ChessColor) bool {
	kr, kc, ok := chessKingSquare(board, color)
	if !ok {
		return true
	}
	return chessSquareAttacked(board, kr, kc, oppositeChessColor(color))
}

func appendChessMove(out *[]types.ChessMove, fromRow, fromCol, toRow, toCol int, promote string) {
	*out = append(*out, types.ChessMove{
		From:    types.Pos{Row: fromRow, Col: fromCol},
		To:      types.Pos{Row: toRow, Col: toCol},
		Promote: promote,
	})
}

func chessAddSliderMoves(board [][]*types.ChessCell, color types.ChessColor, fromRow, fromCol int, rays [][2]int, out *[]types.ChessMove) {
	for _, d := range rays {
		r, c := fromRow+d[0], fromCol+d[1]
		for chessInBounds(r, c) {
			if board[r][c] == nil {
				appendChessMove(out, fromRow, fromCol, r, c, "")
				r += d[0]
				c += d[1]
				continue
			}
			colr, _, ok := parseChessCell(*board[r][c])
			if ok && colr != color {
				appendChessMove(out, fromRow, fromCol, r, c, "")
			}
			break
		}
	}
}

func chessCastlingRights(state *types.ChessState, color types.ChessColor) (king, queen bool) {
	if color == types.ChessWhite {
		return state.CastlingWhiteK, state.CastlingWhiteQ
	}
	return state.CastlingBlackK, state.CastlingBlackQ
}

func chessPseudoLegalMoves(state *types.ChessState, color types.ChessColor) []types.ChessMove {
	var out []types.ChessMove
	board := state.Board
	for r := 0; r < chessSize; r++ {
		for c := 0; c < chessSize; c++ {
			if board[r][c] == nil {
				continue
			}
			colr, piece, ok := parseChessCell(*board[r][c])
			if !ok || colr != color {
				continue
			}
			switch piece {
			case types.ChessKnight:
				for _, d := range chessKnightDeltas {
					nr, nc := r+d[0], c+d[1]
					if !chessInBounds(nr, nc) {
						continue
					}
					if chessSquareOccupiedBy(board, nr, nc, color) {
						continue
					}
					appendChessMove(&out, r, c, nr, nc, "")
				}
			case types.ChessKing:
				for _, d := range chessKingDeltas {
					nr, nc := r+d[0], c+d[1]
					if !chessInBounds(nr, nc) {
						continue
					}
					if chessSquareOccupiedBy(board, nr, nc, color) {
						continue
					}
					appendChessMove(&out, r, c, nr, nc, "")
				}
			case types.ChessBishop:
				chessAddSliderMoves(board, color, r, c, chessBishopRays, &out)
			case types.ChessRook:
				chessAddSliderMoves(board, color, r, c, chessRookRays, &out)
			case types.ChessQueen:
				chessAddSliderMoves(board, color, r, c, chessBishopRays, &out)
				chessAddSliderMoves(board, color, r, c, chessRookRays, &out)
			case types.ChessPawn:
				dir := 1
				startRow, lastRow := 1, 7
				if color == types.ChessWhite {
					dir = -1
					startRow, lastRow = 6, 0
				}
				nr := r + dir
				if chessInBounds(nr, c) && board[nr][c] == nil {
					if nr == lastRow {
						for _, p := range []string{"queen", "rook", "bishop", "knight"} {
							appendChessMove(&out, r, c, nr, c, p)
						}
					} else {
						appendChessMove(&out, r, c, nr, c, "")
					}
					nr2 := r + 2*dir
					if r == startRow && chessInBounds(nr2, c) && board[nr2][c] == nil {
						appendChessMove(&out, r, c, nr2, c, "")
					}
				}
				for _, dc := range []int{-1, 1} {
					nc := c + dc
					if !chessInBounds(nr, nc) {
						continue
					}
					capture := false
					if board[nr][nc] != nil {
						occ, _, ok := parseChessCell(*board[nr][nc])
						capture = ok && occ != color
					} else if chessPosEqual(state.EnPassant, nr, nc) {
						capture = true
					}
					if !capture {
						continue
					}
					if nr == lastRow {
						for _, p := range []string{"queen", "rook", "bishop", "knight"} {
							appendChessMove(&out, r, c, nr, nc, p)
						}
					} else {
						appendChessMove(&out, r, c, nr, nc, "")
					}
				}
			}
		}
	}
	// 易位：王未动且仍在原位。
	kr, kc, ok := chessKingSquare(board, color)
	kingRight, queenRight := chessCastlingRights(state, color)
	homeRow := 7
	if color == types.ChessBlack {
		homeRow = 0
	}
	if ok && kr == homeRow && kc == 4 {
		if kingRight && board[homeRow][5] == nil && board[homeRow][6] == nil {
			if rookColor, rookPiece, rok := parseChessCellPtr(board[homeRow][7]); rok && rookColor == color && rookPiece == types.ChessRook {
				appendChessMove(&out, homeRow, 4, homeRow, 6, "")
			}
		}
		if queenRight && board[homeRow][3] == nil && board[homeRow][2] == nil && board[homeRow][1] == nil {
			if rookColor, rookPiece, rok := parseChessCellPtr(board[homeRow][0]); rok && rookColor == color && rookPiece == types.ChessRook {
				appendChessMove(&out, homeRow, 4, homeRow, 2, "")
			}
		}
	}
	return out
}

func parseChessCellPtr(cell *types.ChessCell) (types.ChessColor, types.ChessPiece, bool) {
	if cell == nil {
		return "", "", false
	}
	return parseChessCell(*cell)
}

func chessApplyOnBoard(board [][]*types.ChessCell, move types.ChessMove, ep *types.Pos) [][]*types.ChessCell {
	out := cloneChessBoard(board)
	src := out[move.From.Row][move.From.Col]
	if src == nil {
		return out
	}
	color, piece, ok := parseChessCell(*src)
	if !ok {
		return out
	}
	// 吃过路兵：走到空的 ep 格时删掉被吃兵。
	if piece == types.ChessPawn && out[move.To.Row][move.To.Col] == nil && chessPosEqual(ep, move.To.Row, move.To.Col) {
		dir := 1
		if color == types.ChessWhite {
			dir = -1
		}
		capRow := move.To.Row - dir
		if chessInBounds(capRow, move.To.Col) {
			out[capRow][move.To.Col] = nil
		}
	}
	if move.Promote != "" && piece == types.ChessPawn {
		promoted := types.ChessPiece(move.Promote)
		c := chessCellOf(color, promoted)
		out[move.To.Row][move.To.Col] = &c
	} else {
		out[move.To.Row][move.To.Col] = src
	}
	out[move.From.Row][move.From.Col] = nil
	// 易位：王横移两格时把车带到内侧。
	if piece == types.ChessKing && absInt(move.To.Col-move.From.Col) == 2 {
		row := move.From.Row
		if move.To.Col == 6 {
			out[row][5] = out[row][7]
			out[row][7] = nil
		} else if move.To.Col == 2 {
			out[row][3] = out[row][0]
			out[row][0] = nil
		}
	}
	return out
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func chessMoveLeavesKingSafe(state *types.ChessState, color types.ChessColor, move types.ChessMove) bool {
	next := chessApplyOnBoard(state.Board, move, state.EnPassant)
	return !chessInCheck(next, color)
}

func chessIsCastlingMove(state *types.ChessState, color types.ChessColor, move types.ChessMove) bool {
	if move.From.Col != 4 || absInt(move.To.Col-move.From.Col) != 2 {
		return false
	}
	homeRow := 7
	if color == types.ChessBlack {
		homeRow = 0
	}
	if move.From.Row != homeRow || move.To.Row != homeRow {
		return false
	}
	_, piece, ok := parseChessCellPtr(state.Board[move.From.Row][move.From.Col])
	return ok && piece == types.ChessKing
}

func chessCastlingPathSafe(state *types.ChessState, color types.ChessColor, move types.ChessMove) bool {
	if chessInCheck(state.Board, color) {
		return false
	}
	step := 1
	if move.To.Col < move.From.Col {
		step = -1
	}
	enemy := oppositeChessColor(color)
	for c := move.From.Col + step; c != move.To.Col+step; c += step {
		if chessSquareAttacked(state.Board, move.From.Row, c, enemy) {
			return false
		}
	}
	return true
}

func chessLegalMovesForColor(state *types.ChessState, color types.ChessColor) []types.ChessMove {
	var legal []types.ChessMove
	for _, move := range chessPseudoLegalMoves(state, color) {
		if chessIsCastlingMove(state, color, move) && !chessCastlingPathSafe(state, color, move) {
			continue
		}
		if chessMoveLeavesKingSafe(state, color, move) {
			legal = append(legal, move)
		}
	}
	return legal
}

func chessLegalMoves(state *types.ChessState) []types.ChessMove {
	if state == nil {
		return nil
	}
	return chessLegalMovesForColor(state, chessColorForSeat(state, state.Turn))
}

func chessMoveMatches(a, b types.ChessMove) bool {
	return a.From.Row == b.From.Row && a.From.Col == b.From.Col &&
		a.To.Row == b.To.Row && a.To.Col == b.To.Col && a.Promote == b.Promote
}

func chessFindLegal(state *types.ChessState, move types.ChessMove) (types.ChessMove, bool) {
	for _, m := range chessLegalMoves(state) {
		if chessMoveMatches(m, move) {
			return m, true
		}
	}
	return types.ChessMove{}, false
}

func chessLegalHasEnPassant(state *types.ChessState) bool {
	if state == nil || state.EnPassant == nil {
		return false
	}
	for _, mv := range state.LegalMoves {
		if !chessPosEqual(state.EnPassant, mv.To.Row, mv.To.Col) {
			continue
		}
		_, piece, ok := parseChessCellPtr(state.Board[mv.From.Row][mv.From.Col])
		if ok && piece == types.ChessPawn {
			return true
		}
	}
	return false
}

func chessBishopSquareColor(row, col int) int {
	return (row + col) % 2
}

func bishopsAllSameSquareColor(sqs []int) bool {
	if len(sqs) == 0 {
		return true
	}
	for _, sq := range sqs {
		if sq != sqs[0] {
			return false
		}
	}
	return true
}

// chessCanPossiblyMate 判断 color 一方是否存在任何合法着法序列能将死对方（含对方配合）。
// 用于 FIDE 6.9：超时方判负，除非未超时方无法将死，此时为和棋。
func chessCanPossiblyMate(board [][]*types.ChessCell, color types.ChessColor) bool {
	type tally struct {
		p, n, b, r, q int
		bsq           []int
	}
	var us, them tally
	for r := 0; r < chessSize; r++ {
		for c := 0; c < chessSize; c++ {
			if board[r][c] == nil {
				continue
			}
			colr, piece, ok := parseChessCell(*board[r][c])
			if !ok {
				continue
			}
			side := &us
			if colr != color {
				side = &them
			}
			switch piece {
			case types.ChessPawn:
				side.p++
			case types.ChessKnight:
				side.n++
			case types.ChessBishop:
				side.b++
				side.bsq = append(side.bsq, chessBishopSquareColor(r, c))
			case types.ChessRook:
				side.r++
			case types.ChessQueen:
				side.q++
			}
		}
	}
	if us.q > 0 || us.r > 0 || us.p > 0 {
		return true
	}
	minors := us.n + us.b
	if minors >= 2 {
		if us.n == 0 && bishopsAllSameSquareColor(us.bsq) {
			themSame := bishopsAllSameSquareColor(them.bsq)
			if them.p+them.n+them.r+them.q == 0 && (them.b == 0 || (themSame && len(them.bsq) > 0 && them.bsq[0] == us.bsq[0])) {
				return false
			}
		}
		return true
	}
	if minors == 0 {
		return false
	}
	if them.p+them.n+them.r+them.q == 0 && them.b == 0 {
		return false
	}
	if us.b == 1 && them.n+them.p+them.r+them.q == 0 {
		for _, sq := range them.bsq {
			if len(us.bsq) > 0 && sq != us.bsq[0] {
				return true
			}
		}
		return false
	}
	return true
}

func chessInsufficientMaterial(board [][]*types.ChessCell) bool {
	type count struct {
		n, b int
		bsq  []int
	}
	var w, b count
	for r := 0; r < chessSize; r++ {
		for c := 0; c < chessSize; c++ {
			if board[r][c] == nil {
				continue
			}
			color, piece, ok := parseChessCell(*board[r][c])
			if !ok {
				continue
			}
			side := &w
			if color == types.ChessBlack {
				side = &b
			}
			switch piece {
			case types.ChessKing:
			case types.ChessKnight:
				side.n++
			case types.ChessBishop:
				side.b++
				side.bsq = append(side.bsq, chessBishopSquareColor(r, c))
			default:
				return false
			}
		}
	}
	// K vs K
	if w.n+w.b+b.n+b.b == 0 {
		return true
	}
	// K vs KN / KB
	if (w.n+w.b == 0 && b.n+b.b == 1) || (b.n+b.b == 0 && w.n+w.b == 1) {
		return true
	}
	// KB vs KB same color
	if w.n == 0 && b.n == 0 && w.b == 1 && b.b == 1 && len(w.bsq) == 1 && len(b.bsq) == 1 && w.bsq[0] == b.bsq[0] {
		return true
	}
	return false
}

func chessPositionKey(state *types.ChessState) string {
	var b strings.Builder
	for r := 0; r < chessSize; r++ {
		for c := 0; c < chessSize; c++ {
			if state.Board[r][c] == nil {
				b.WriteByte('.')
				continue
			}
			b.WriteString(string(*state.Board[r][c]))
			b.WriteByte(',')
		}
		b.WriteByte('/')
	}
	b.WriteString(string(chessColorForSeat(state, state.Turn)))
	if state.CastlingWhiteK {
		b.WriteByte('K')
	}
	if state.CastlingWhiteQ {
		b.WriteByte('Q')
	}
	if state.CastlingBlackK {
		b.WriteByte('k')
	}
	if state.CastlingBlackQ {
		b.WriteByte('q')
	}
	if state.EnPassant != nil {
		fmt.Fprintf(&b, "e%d%d", state.EnPassant.Row, state.EnPassant.Col)
	}
	return b.String()
}

func chessUpdateCastling(state *types.ChessState, move types.ChessMove, color types.ChessColor, piece types.ChessPiece) {
	if piece == types.ChessKing {
		if color == types.ChessWhite {
			state.CastlingWhiteK, state.CastlingWhiteQ = false, false
		} else {
			state.CastlingBlackK, state.CastlingBlackQ = false, false
		}
	}
	clearRook := func(row, col int) {
		if row == 7 && col == 7 {
			state.CastlingWhiteK = false
		}
		if row == 7 && col == 0 {
			state.CastlingWhiteQ = false
		}
		if row == 0 && col == 7 {
			state.CastlingBlackK = false
		}
		if row == 0 && col == 0 {
			state.CastlingBlackQ = false
		}
	}
	if piece == types.ChessRook {
		clearRook(move.From.Row, move.From.Col)
	}
	clearRook(move.To.Row, move.To.Col)
}

func chessMoveCaptures(state *types.ChessState, move types.ChessMove) bool {
	if state == nil || state.Board[move.To.Row][move.To.Col] != nil {
		return state != nil
	}
	_, piece, ok := parseChessCellPtr(state.Board[move.From.Row][move.From.Col])
	return ok && piece == types.ChessPawn && chessPosEqual(state.EnPassant, move.To.Row, move.To.Col)
}

func chessApplyMoveToState(state *types.ChessState, move types.ChessMove) {
	src := state.Board[move.From.Row][move.From.Col]
	color, piece, ok := parseChessCellPtr(src)
	if !ok {
		return
	}
	capture := state.Board[move.To.Row][move.To.Col] != nil
	if piece == types.ChessPawn && state.Board[move.To.Row][move.To.Col] == nil && chessPosEqual(state.EnPassant, move.To.Row, move.To.Col) {
		capture = true
	}
	state.Board = chessApplyOnBoard(state.Board, move, state.EnPassant)
	chessUpdateCastling(state, move, color, piece)
	var ep *types.Pos
	if piece == types.ChessPawn && absInt(move.To.Row-move.From.Row) == 2 {
		mid := types.Pos{Row: (move.From.Row + move.To.Row) / 2, Col: move.From.Col}
		ep = &mid
	}
	state.EnPassant = ep
	if piece == types.ChessPawn || capture {
		state.HalfmoveClock = 0
	} else {
		state.HalfmoveClock++
	}
	state.MoveCount++
	from := types.Pos{Row: move.From.Row, Col: move.From.Col}
	to := types.Pos{Row: move.To.Row, Col: move.To.Col}
	state.LastFrom = &from
	state.LastTo = &to
	state.Turn = oppositeSeat(state.Turn)
	state.LegalMoves = chessLegalMoves(state)
	state.InCheck = chessInCheck(state.Board, chessColorForSeat(state, state.Turn))
	// FIDE 9.2：过路兵权利只在「确实能吃」时计入局面；否则与无过路兵的同一局面视为相同。
	if state.EnPassant != nil && !chessLegalHasEnPassant(state) {
		state.EnPassant = nil
	}
}

type chessOutcome int

const (
	chessOngoing chessOutcome = iota
	chessCheckmate
	chessStalemate
	chessFiftyMove
	chessInsufficient
	chessRepetition
)

func chessEvaluateOutcome(state *types.ChessState, repetition []string) chessOutcome {
	if len(state.LegalMoves) == 0 {
		if state.InCheck {
			return chessCheckmate
		}
		return chessStalemate
	}
	if state.HalfmoveClock >= 100 {
		return chessFiftyMove
	}
	if chessInsufficientMaterial(state.Board) {
		return chessInsufficient
	}
	key := chessPositionKey(state)
	n := 0
	for _, k := range repetition {
		if k == key {
			n++
		}
	}
	if n >= 3 {
		return chessRepetition
	}
	return chessOngoing
}

func chessOutcomeNote(out chessOutcome) string {
	switch out {
	case chessCheckmate:
		return "将死"
	case chessStalemate:
		return "逼和"
	case chessFiftyMove:
		return "五十步和棋"
	case chessInsufficient:
		return "双方子力不足"
	case chessRepetition:
		return "三次重复局面"
	default:
		return ""
	}
}

func chessColorName(color types.ChessColor) string {
	if color == types.ChessWhite {
		return "白"
	}
	return "黑"
}

func normalizeChessPromote(v string) string {
	switch v {
	case "queen", "rook", "bishop", "knight":
		return v
	default:
		return ""
	}
}
