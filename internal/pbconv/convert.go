// Package pbconv：领域类型 ↔ wire protobuf（内存转换；线路上只有 protobuf 字节）。
package pbconv

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/doumiao/newRPS/internal/types"
	"github.com/doumiao/newRPS/internal/wire"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

var (
	protoJSON = protojson.MarshalOptions{UseProtoNames: false, EmitUnpopulated: false}
	protoUN   = protojson.UnmarshalOptions{DiscardUnknown: true}
)

// HashBytes SHA-256 hex of raw protobuf payload.
func HashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// ── dynamic Struct（RPC）────────────────────────────────

// AnyToStruct 将任意 JSON 友好值转为 protobuf Struct（线路为二进制 Struct，非 JSON 文本）。
func AnyToStruct(v any) (*structpb.Struct, error) {
	if v == nil {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}, nil
	}
	// 经 JSON 中转成 map，仅用于绑定 Go 匿名结构体，不进 WebSocket
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		// 非 object：包一层
		val, err2 := structpb.NewValue(jsonRaw(b))
		if err2 != nil {
			return nil, err
		}
		return &structpb.Struct{Fields: map[string]*structpb.Value{"_": val}}, nil
	}
	return structpb.NewStruct(m)
}

func jsonRaw(b []byte) any {
	var v any
	_ = json.Unmarshal(b, &v)
	return v
}

// StructToAny 解码到 out（指针）。
func StructToAny(s *structpb.Struct, out any) error {
	if s == nil {
		return nil
	}
	b, err := protoJSON.Marshal(s)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

// ValueFromAny / AnyFromValue 用于 DELTA 补丁值。
func ValueFromAny(v any) (*structpb.Value, error) {
	if v == nil {
		return structpb.NewNullValue(), nil
	}
	return structpb.NewValue(normalizeForStructpb(v))
}

func normalizeForStructpb(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var out any
	_ = json.Unmarshal(b, &out)
	return out
}

func AnyFromValue(v *structpb.Value) any {
	if v == nil {
		return nil
	}
	return v.AsInterface()
}

// ── 状态文档 ──────────────────────────────────────────

func MarshalStateLobby(snap types.LobbySnapshot) (*wire.StateDocument, []byte, string, error) {
	pb, err := LobbyToProto(snap)
	if err != nil {
		return nil, nil, "", err
	}
	doc := &wire.StateDocument{Doc: &wire.StateDocument_Lobby{Lobby: pb}}
	raw, err := proto.Marshal(doc)
	if err != nil {
		return nil, nil, "", err
	}
	return doc, raw, HashBytes(raw), nil
}

func MarshalStateRoom(snap types.RoomSnapshot) (*wire.StateDocument, []byte, string, error) {
	pb, err := RoomToProto(snap)
	if err != nil {
		return nil, nil, "", err
	}
	doc := &wire.StateDocument{Doc: &wire.StateDocument_Room{Room: pb}}
	raw, err := proto.Marshal(doc)
	if err != nil {
		return nil, nil, "", err
	}
	return doc, raw, HashBytes(raw), nil
}

func MarshalStateConfig(cfg types.AppConfig) (*wire.StateDocument, []byte, string, error) {
	pb, err := ConfigToProto(cfg)
	if err != nil {
		return nil, nil, "", err
	}
	doc := &wire.StateDocument{Doc: &wire.StateDocument_Config{Config: pb}}
	raw, err := proto.Marshal(doc)
	if err != nil {
		return nil, nil, "", err
	}
	return doc, raw, HashBytes(raw), nil
}

// JSONCamelToProto 通用：领域 camelCase JSON → proto 消息。
func JSONCamelToProto(domain any, msg proto.Message) error {
	b, err := json.Marshal(domain)
	if err != nil {
		return err
	}
	return protoUN.Unmarshal(b, msg)
}

// ProtoToCamelMap proto → 前端习惯的 camelCase 对象树（map/slice）。
func ProtoToCamelMap(msg proto.Message) (any, error) {
	b, err := protoJSON.Marshal(msg)
	if err != nil {
		return nil, err
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// LobbyToProto 含 map→entries 重整。
func LobbyToProto(snap types.LobbySnapshot) (*wire.LobbySnapshot, error) {
	// 先展成 JSON 友好中间结构
	type midLobby struct {
		Config            *types.AppConfig      `json:"config,omitempty"`
		OnlineCount       int                   `json:"onlineCount"`
		Players           []types.LobbyPlayer   `json:"players"`
		Rooms             []types.LobbyRoomInfo `json:"rooms"`
		NormalLeaderboard []types.LobbyPlayer   `json:"normalLeaderboard"`
		RankedLeaderboard []types.LobbyPlayer   `json:"rankedLeaderboard"`
		Suggestions       []types.Suggestion    `json:"suggestions"`
		LobbyChat         []types.ChatMessage   `json:"lobbyChat"`
		ServerStats       types.ServerStats     `json:"serverStats"`
	}
	// 但 proto LobbySnapshot 用 entries；手写填充更可靠
	out := &wire.LobbySnapshot{
		OnlineCount: int32(snap.OnlineCount),
	}
	if snap.Config != nil {
		cfg, err := ConfigToProto(*snap.Config)
		if err != nil {
			return nil, err
		}
		out.Config = cfg
	}
	for id, p := range snap.Players {
		pp, err := lobbyPlayerToProto(p)
		if err != nil {
			return nil, err
		}
		if pp.Id == "" {
			pp.Id = id
		}
		out.Players = append(out.Players, &wire.LobbyPlayerEntry{Id: id, Player: pp})
	}
	// 同上：map 遍历顺序是随机的，不排序的话在线/离线玩家列表会在每次广播时乱跳；按 id 稳定排序。
	sort.Slice(out.Players, func(i, j int) bool { return out.Players[i].Id < out.Players[j].Id })
	for id, r := range snap.Rooms {
		rr, err := lobbyRoomToProto(r)
		if err != nil {
			return nil, err
		}
		if rr.Id == "" {
			rr.Id = id
		}
		out.Rooms = append(out.Rooms, &wire.LobbyRoomEntry{Id: id, Room: rr})
	}
	// map 遍历顺序是随机的，不排序的话大厅房间列表会在每次广播时乱跳；按 id 稳定排序。
	sort.Slice(out.Rooms, func(i, j int) bool { return out.Rooms[i].Id < out.Rooms[j].Id })
	for _, p := range snap.NormalLeaderboard {
		pp, err := lobbyPlayerToProto(p)
		if err != nil {
			return nil, err
		}
		out.NormalLeaderboard = append(out.NormalLeaderboard, pp)
	}
	for _, p := range snap.RankedLeaderboard {
		pp, err := lobbyPlayerToProto(p)
		if err != nil {
			return nil, err
		}
		out.RankedLeaderboard = append(out.RankedLeaderboard, pp)
	}
	for _, s := range snap.Suggestions {
		ss, err := suggestionToProto(s)
		if err != nil {
			return nil, err
		}
		out.Suggestions = append(out.Suggestions, ss)
	}
	for _, c := range snap.LobbyChat {
		cc, err := chatToProto(c)
		if err != nil {
			return nil, err
		}
		out.LobbyChat = append(out.LobbyChat, cc)
	}
	for _, e := range snap.PetBonds {
		out.PetBonds = append(out.PetBonds, &wire.PetBondEdge{
			MasterId: e.MasterID, PetId: e.PetID, PetTitle: e.PetTitle,
		})
	}
	out.ServerStats = serverStatsToProto(snap.ServerStats)
	return out, nil
}

func lobbyPlayerToProto(p types.LobbyPlayer) (*wire.LobbyPlayer, error) {
	m := &wire.LobbyPlayer{}
	if err := JSONCamelToProto(p, m); err != nil {
		return nil, err
	}
	return m, nil
}

func anyPlayerToProto(v any) (*wire.PublicPlayer, error) {
	switch p := v.(type) {
	case types.PublicPlayer:
		return publicPlayerToProto(p)
	case *types.PublicPlayer:
		if p == nil {
			return nil, nil
		}
		return publicPlayerToProto(*p)
	case types.LobbyPlayer:
		return publicPlayerToProto(p.AsPublicPlayer())
	case *types.LobbyPlayer:
		if p == nil {
			return nil, nil
		}
		return publicPlayerToProto(p.AsPublicPlayer())
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		var pub types.PublicPlayer
		if err := json.Unmarshal(b, &pub); err != nil {
			return nil, err
		}
		return publicPlayerToProto(pub)
	}
}

func publicPlayerToProto(p types.PublicPlayer) (*wire.PublicPlayer, error) {
	m := &wire.PublicPlayer{}
	if err := JSONCamelToProto(p, m); err != nil {
		return nil, err
	}
	return m, nil
}

func chatToProto(c types.ChatMessage) (*wire.ChatMessage, error) {
	m := &wire.ChatMessage{}
	if err := JSONCamelToProto(c, m); err != nil {
		return nil, err
	}
	return m, nil
}

func suggestionToProto(s types.Suggestion) (*wire.Suggestion, error) {
	m := &wire.Suggestion{}
	if err := JSONCamelToProto(s, m); err != nil {
		return nil, err
	}
	if s.AuthorPlayer != nil {
		ap, err := publicPlayerToProto(*s.AuthorPlayer)
		if err != nil {
			return nil, err
		}
		m.AuthorPlayer = ap
	}
	return m, nil
}

func serverStatsToProto(s types.ServerStats) *wire.ServerStats {
	return &wire.ServerStats{
		StartedAt:                 s.StartedAt,
		RoomBroadcasts:            int32(s.RoomBroadcasts),
		LobbyBroadcasts:           int32(s.LobbyBroadcasts),
		Disconnects:               int32(s.Disconnects),
		Reconnects:                int32(s.Reconnects),
		LastRoomSnapshotBytes:     int32(s.LastRoomSnapshotBytes),
		LastLobbySnapshotBytes:    int32(s.LastLobbySnapshotBytes),
		RecentRoomBroadcasts:      int32(s.RecentRoomBroadcasts),
		RecentLobbyBroadcasts:     int32(s.RecentLobbyBroadcasts),
		AverageRoomSnapshotBytes:  int32(s.AverageRoomSnapshotBytes),
		AverageLobbySnapshotBytes: int32(s.AverageLobbySnapshotBytes),
	}
}

func lobbyRoomToProto(r types.LobbyRoomInfo) (*wire.LobbyRoomInfo, error) {
	m := &wire.LobbyRoomInfo{
		Id: r.ID, GameId: string(r.GameID), Name: r.Name,
		HasPassword: r.HasPassword, Players: int32(r.Players), Spectators: int32(r.Spectators),
		Status: r.Status, RoomBackgroundImage: r.RoomBackgroundImage,
		EnablePunishment: r.EnablePunishment, PunishmentIds: r.PunishmentIDs, PunishmentId: r.PunishmentID,
		TieDoublePunish: r.TieDoublePunish, RequireOpponentConfirm: r.RequireOpponentConfirm,
		EnableRanked: r.EnableRanked, Stake: int32(r.Stake),
		EnableRankMultiplier: r.EnableRankMultiplier, RankMultiplier: int32(r.RankMultiplier),
		EnableExtremeRanked: r.EnableExtremeRanked, Tags: r.Tags,
		LiarsDiceMinPlayers: int32(r.LiarsDiceMinPlayers), LiarsDiceMaxPlayers: int32(r.LiarsDiceMaxPlayers),
		OthelloMoveSeconds: int32(r.OthelloMoveSeconds), OthelloGameMinutes: int32(r.OthelloGameMinutes),
		GomokuMoveSeconds: int32(r.GomokuMoveSeconds), GomokuGameMinutes: int32(r.GomokuGameMinutes),
		GomokuUndoLimit:   int32(r.GomokuUndoLimit),
		JungleMoveSeconds: int32(r.JungleMoveSeconds), JungleGameMinutes: int32(r.JungleGameMinutes),
	}
	for _, seat := range []types.SeatKey{types.SeatA, types.SeatB} {
		vs := &wire.VersusSeat{}
		occ := r.Versus[seat]
		if occ != nil {
			switch o := occ.(type) {
			case map[string]any:
				// lobby summary shape: {player: LobbyPlayer|PublicPlayer}
				if o["player"] != nil {
					pp, err := anyPlayerToProto(o["player"])
					if err != nil {
						return nil, err
					}
					vs.Player = pp
				}
			case types.PublicPlayer:
				pp, err := publicPlayerToProto(o)
				if err != nil {
					return nil, err
				}
				vs.Player = pp
			case *types.PublicPlayer:
				if o != nil {
					pp, err := publicPlayerToProto(*o)
					if err != nil {
						return nil, err
					}
					vs.Player = pp
				}
			}
		}
		m.Versus = append(m.Versus, &wire.VersusPair{Key: string(seat), Value: vs})
	}
	return m, nil
}

// RoomToProto 房间快照。
func RoomToProto(snap types.RoomSnapshot) (*wire.RoomSnapshot, error) {
	m := &wire.RoomSnapshot{
		Id: snap.ID, UpdatedAt: snap.UpdatedAt,
		Status: snap.Status, Phase: string(snap.Phase), ResultText: snap.ResultText,
		PunishedPlayerIds: snap.PunishedPlayerIDs, RoundHistoryTotal: int32(snap.RoundHistoryTotal),
		ForgiveAdvantageTargetId:      snap.ForgiveAdvantageTargetID,
		ForgiveAdvantageBeneficiaryId: snap.ForgiveAdvantageBeneficiaryID,
	}
	rs := &wire.RoomSettings{}
	if err := JSONCamelToProto(snap.Settings, rs); err != nil {
		return nil, err
	}
	rs.GameId = string(snap.Settings.GameID)
	rs.Stake = int32(snap.Settings.Stake)
	rs.RankMultiplier = int32(snap.Settings.RankMultiplier)
	m.Settings = rs

	for _, seat := range []types.SeatKey{types.SeatA, types.SeatB} {
		pair := &wire.SeatOccupantPair{Key: string(seat)}
		if occ, ok := snap.Seats[seat]; ok && occ != nil {
			so, err := seatOccupantToProto(occ)
			if err != nil {
				return nil, err
			}
			pair.Value = so
		}
		m.Seats = append(m.Seats, pair)
		if snap.Ready != nil {
			m.Ready = append(m.Ready, &wire.BoolPair{Key: string(seat), Value: snap.Ready[seat]})
		}
		if snap.Choices != nil {
			if ch, ok := snap.Choices[seat]; ok && ch != nil {
				m.Choices = append(m.Choices, &wire.StringPair{Key: string(seat), Value: fmt.Sprint(ch)})
			}
		}
		if snap.RevealedChoices != nil {
			if mv, ok := snap.RevealedChoices[seat]; ok {
				m.RevealedChoices = append(m.RevealedChoices, &wire.StringPair{Key: string(seat), Value: string(mv)})
			}
		}
		if snap.Score != nil {
			m.Score = append(m.Score, &wire.IntPair{Key: string(seat), Value: int32(snap.Score[seat])})
		}
		if snap.SeatedScore != nil {
			m.SeatedScore = append(m.SeatedScore, &wire.IntPair{Key: string(seat), Value: int32(snap.SeatedScore[seat])})
		}
		if snap.SeatStats != nil {
			st := snap.SeatStats[seat]
			m.SeatStats = append(m.SeatStats, &wire.SeatStatsPair{Key: string(seat), Value: &wire.SeatStats{
				Wins: int32(st.Wins), Losses: int32(st.Losses), Draws: int32(st.Draws), Punishments: int32(st.Punishments),
			}})
		}
	}
	for _, p := range snap.Spectators {
		pp, err := publicPlayerToProto(p)
		if err != nil {
			return nil, err
		}
		m.Spectators = append(m.Spectators, pp)
	}
	for _, pr := range snap.Proofs {
		pp := &wire.PunishmentProof{}
		_ = JSONCamelToProto(pr, pp)
		m.Proofs = append(m.Proofs, pp)
	}
	for _, h := range snap.RoundHistory {
		hh, err := historyToProto(h)
		if err != nil {
			return nil, err
		}
		m.RoundHistory = append(m.RoundHistory, hh)
	}
	for _, c := range snap.Chat {
		cc, err := chatToProto(c)
		if err != nil {
			return nil, err
		}
		m.Chat = append(m.Chat, cc)
	}
	if snap.Othello != nil {
		o, err := othelloToProto(snap.Othello)
		if err != nil {
			return nil, err
		}
		m.Othello = o
	}
	if snap.TicTacToe != nil {
		t, err := tictactoeToProto(snap.TicTacToe)
		if err != nil {
			return nil, err
		}
		m.Tictactoe = t
	}
	if snap.LiarsDice != nil {
		l, err := liarsDiceToProto(snap.LiarsDice)
		if err != nil {
			return nil, err
		}
		m.LiarsDice = l
	}
	if snap.Gomoku != nil {
		g, err := gomokuToProto(snap.Gomoku)
		if err != nil {
			return nil, err
		}
		m.Gomoku = g
	}
	if snap.Jungle != nil {
		j, err := jungleToProto(snap.Jungle)
		if err != nil {
			return nil, err
		}
		m.Jungle = j
	}
	return m, nil
}

func int32List(vals []int) *wire.Int32List {
	out := &wire.Int32List{Values: make([]int32, len(vals))}
	for i, v := range vals {
		out.Values[i] = int32(v)
	}
	return out
}

func liarsDiceToProto(s *types.LiarsDiceState) (*wire.LiarsDiceState, error) {
	m := &wire.LiarsDiceState{
		ParticipantIds: s.ParticipantIDs, ReadyPlayerIds: s.ReadyPlayerIDs,
		CurrentTurn: s.CurrentTurn, OnesWildDisabled: s.OnesWildDisabled,
		RoundNumber: int32(s.RoundNumber), Ended: s.Ended,
		WinnerId: s.WinnerID, LoserId: s.LoserID, ActualCount: int32(s.ActualCount),
		MinPlayers: int32(s.MinPlayers), MaxPlayers: int32(s.MaxPlayers),
	}
	for k, v := range s.DiceCounts {
		m.DiceCounts = append(m.DiceCounts, &wire.IntPair{Key: k, Value: int32(v)})
	}
	// map 遍历顺序是随机的，不排序的话骰子计数会在每次广播时乱跳；按 key 稳定排序。
	sort.Slice(m.DiceCounts, func(i, j int) bool { return m.DiceCounts[i].Key < m.DiceCounts[j].Key })
	if s.CurrentBid != nil {
		m.CurrentBid = &wire.LiarsDiceBid{
			PlayerId: s.CurrentBid.PlayerID, Count: int32(s.CurrentBid.Count),
			Face: int32(s.CurrentBid.Face), At: s.CurrentBid.At,
		}
	}
	for _, b := range s.BidHistory {
		m.BidHistory = append(m.BidHistory, &wire.LiarsDiceBid{
			PlayerId: b.PlayerID, Count: int32(b.Count), Face: int32(b.Face), At: b.At,
		})
	}
	for k, v := range s.RevealedHands {
		m.RevealedHands = append(m.RevealedHands, &wire.LiarsDiceHandsPair{Key: k, Value: int32List(v)})
	}
	// map 遍历顺序是随机的，不排序的话开牌列表会在每次广播时乱跳；按 key 稳定排序。
	sort.Slice(m.RevealedHands, func(i, j int) bool { return m.RevealedHands[i].Key < m.RevealedHands[j].Key })
	return m, nil
}

func seatOccupantToProto(occ any) (*wire.SeatOccupant, error) {
	switch o := occ.(type) {
	case types.PublicPlayer:
		pp, err := publicPlayerToProto(o)
		if err != nil {
			return nil, err
		}
		return &wire.SeatOccupant{Kind: &wire.SeatOccupant_Player{Player: pp}}, nil
	case *types.PublicPlayer:
		if o == nil {
			return nil, nil
		}
		pp, err := publicPlayerToProto(*o)
		if err != nil {
			return nil, err
		}
		return &wire.SeatOccupant{Kind: &wire.SeatOccupant_Player{Player: pp}}, nil
	default:
		// try JSON as PublicPlayer
		b, _ := json.Marshal(o)
		var p types.PublicPlayer
		if json.Unmarshal(b, &p) == nil && p.ID != "" {
			pp, err := publicPlayerToProto(p)
			if err != nil {
				return nil, err
			}
			return &wire.SeatOccupant{Kind: &wire.SeatOccupant_Player{Player: pp}}, nil
		}
	}
	return nil, nil
}

func historyToProto(h types.RoundHistoryItem) (*wire.RoundHistoryItem, error) {
	// LiarsDiceHands/LiarsDiceNames 是 map，通用 JSON 转换会因为消息里对应字段是
	// repeated pair（非 map）类型不匹配而报错——先置空再走通用转换，随后手动填充。
	forJSON := h
	forJSON.LiarsDiceHands = nil
	forJSON.LiarsDiceNames = nil
	m := &wire.RoundHistoryItem{}
	if err := JSONCamelToProto(forJSON, m); err != nil {
		return nil, err
	}
	m.MoveA = string(h.MoveA)
	m.MoveB = string(h.MoveB)
	m.Result = string(h.Result)
	m.GameId = string(h.GameID)
	for k, v := range h.LiarsDiceHands {
		m.LiarsDiceHands = append(m.LiarsDiceHands, &wire.LiarsDiceHandsPair{Key: k, Value: int32List(v)})
	}
	// map 遍历顺序是随机的，不排序的话历史记录里的开牌/参战者列表会每次读取都乱跳；按 key 稳定排序。
	sort.Slice(m.LiarsDiceHands, func(i, j int) bool { return m.LiarsDiceHands[i].Key < m.LiarsDiceHands[j].Key })
	for k, v := range h.LiarsDiceNames {
		m.LiarsDiceNames = append(m.LiarsDiceNames, &wire.StringPair{Key: k, Value: v})
	}
	sort.Slice(m.LiarsDiceNames, func(i, j int) bool { return m.LiarsDiceNames[i].Key < m.LiarsDiceNames[j].Key })
	return m, nil
}

func othelloToProto(s *types.OthelloState) (*wire.OthelloState, error) {
	m := &wire.OthelloState{
		Turn: string(s.Turn), BlackSeat: string(s.BlackSeat),
		PassCount: int32(s.PassCount), BlackCount: int32(s.BlackCount), WhiteCount: int32(s.WhiteCount),
		SettlementEvents: s.SettlementEvents, Ended: s.Ended, Winner: string(s.Winner),
		MoveDeadlineAt: s.MoveDeadlineAt, ClockDeadlineAt: s.ClockDeadlineAt,
	}
	for k, v := range s.ClockRemaining {
		m.ClockRemaining = append(m.ClockRemaining, &wire.IntPair{Key: string(k), Value: int32(v)})
	}
	sort.Slice(m.ClockRemaining, func(i, j int) bool { return m.ClockRemaining[i].Key < m.ClockRemaining[j].Key })
	for _, row := range s.Board {
		br := &wire.BoardRow{}
		for _, cell := range row {
			if cell == nil {
				br.Cells = append(br.Cells, "")
			} else {
				br.Cells = append(br.Cells, string(*cell))
			}
		}
		m.Board = append(m.Board, br)
	}
	for _, p := range s.LegalMoves {
		m.LegalMoves = append(m.LegalMoves, &wire.Pos{Row: int32(p.Row), Col: int32(p.Col)})
	}
	for k, v := range s.RankedDelta {
		m.RankedDelta = append(m.RankedDelta, &wire.IntPair{Key: string(k), Value: int32(v)})
	}
	sort.Slice(m.RankedDelta, func(i, j int) bool { return m.RankedDelta[i].Key < m.RankedDelta[j].Key })
	if s.PendingSettlement != nil {
		ps := s.PendingSettlement
		m.PendingSettlement = &wire.OthelloPendingSettlement{
			Id: ps.ID, Seat: string(ps.Seat), OpponentSeat: string(ps.OpponentSeat),
			Flips: int32(ps.Flips), Stake: int32(ps.Stake), NextTurn: string(ps.NextTurn),
			ExpiresAt: ps.ExpiresAt, Forced: ps.Forced, ResolvedAs: ps.ResolvedAs,
			ForcedByMasterName: ps.ForcedByMasterName,
		}
	}
	if s.SurrenderRequest != nil {
		m.SurrenderRequest = &wire.OthelloSurrenderRequest{
			FromSeat: string(s.SurrenderRequest.FromSeat), ToSeat: string(s.SurrenderRequest.ToSeat),
			CreatedAt: s.SurrenderRequest.CreatedAt,
		}
	}
	return m, nil
}

func tictactoeToProto(s *types.TicTacToeState) (*wire.TicTacToeState, error) {
	m := &wire.TicTacToeState{
		Turn: string(s.Turn), XSeat: string(s.XSeat), MoveCount: int32(s.MoveCount),
		Ended: s.Ended, Winner: string(s.Winner),
	}
	for _, row := range s.Board {
		br := &wire.BoardRow{}
		for _, cell := range row {
			if cell == nil {
				br.Cells = append(br.Cells, "")
			} else {
				br.Cells = append(br.Cells, string(*cell))
			}
		}
		m.Board = append(m.Board, br)
	}
	for _, p := range s.WinningLine {
		m.WinningLine = append(m.WinningLine, &wire.Pos{Row: int32(p.Row), Col: int32(p.Col)})
	}
	for k, v := range s.RankedDelta {
		m.RankedDelta = append(m.RankedDelta, &wire.IntPair{Key: string(k), Value: int32(v)})
	}
	sort.Slice(m.RankedDelta, func(i, j int) bool { return m.RankedDelta[i].Key < m.RankedDelta[j].Key })
	if s.GiveawayPrompt != nil {
		m.GiveawayPrompt = &wire.TicTacToeGiveawayPrompt{
			Seat: string(s.GiveawayPrompt.Seat), Forced: s.GiveawayPrompt.Forced,
			StartedAt: s.GiveawayPrompt.StartedAt, ExpiresAt: s.GiveawayPrompt.ExpiresAt,
		}
	}
	return m, nil
}

func gomokuToProto(s *types.GomokuState) (*wire.GomokuState, error) {
	m := &wire.GomokuState{
		Turn: string(s.Turn), BlackSeat: string(s.BlackSeat), MoveCount: int32(s.MoveCount),
		Ended: s.Ended, Winner: string(s.Winner),
		GiveawaySeat: string(s.GiveawaySeat), GiveawayForcedByMasterName: s.GiveawayForcedByMasterName,
		MoveDeadlineAt: s.MoveDeadlineAt, ClockDeadlineAt: s.ClockDeadlineAt,
	}
	for k, v := range s.ClockRemaining {
		m.ClockRemaining = append(m.ClockRemaining, &wire.IntPair{Key: string(k), Value: int32(v)})
	}
	sort.Slice(m.ClockRemaining, func(i, j int) bool { return m.ClockRemaining[i].Key < m.ClockRemaining[j].Key })
	for _, row := range s.Board {
		br := &wire.BoardRow{}
		for _, cell := range row {
			if cell == nil {
				br.Cells = append(br.Cells, "")
			} else {
				br.Cells = append(br.Cells, string(*cell))
			}
		}
		m.Board = append(m.Board, br)
	}
	for _, p := range s.Moves {
		m.Moves = append(m.Moves, &wire.Pos{Row: int32(p.Row), Col: int32(p.Col)})
	}
	for _, p := range s.WinningLine {
		m.WinningLine = append(m.WinningLine, &wire.Pos{Row: int32(p.Row), Col: int32(p.Col)})
	}
	for k, v := range s.RankedDelta {
		m.RankedDelta = append(m.RankedDelta, &wire.IntPair{Key: string(k), Value: int32(v)})
	}
	sort.Slice(m.RankedDelta, func(i, j int) bool { return m.RankedDelta[i].Key < m.RankedDelta[j].Key })
	for k, v := range s.UndoCount {
		m.UndoCount = append(m.UndoCount, &wire.IntPair{Key: string(k), Value: int32(v)})
	}
	sort.Slice(m.UndoCount, func(i, j int) bool { return m.UndoCount[i].Key < m.UndoCount[j].Key })
	if s.UndoRequest != nil {
		m.UndoRequest = &wire.GomokuUndoRequest{
			FromSeat: string(s.UndoRequest.FromSeat), ToSeat: string(s.UndoRequest.ToSeat),
			CreatedAt: s.UndoRequest.CreatedAt, ExpiresAt: s.UndoRequest.ExpiresAt,
		}
	}
	if s.ResignRequest != nil {
		m.ResignRequest = &wire.GomokuResignRequest{
			FromSeat: string(s.ResignRequest.FromSeat), ToSeat: string(s.ResignRequest.ToSeat),
			CreatedAt: s.ResignRequest.CreatedAt,
		}
	}
	return m, nil
}

func jungleToProto(s *types.JungleState) (*wire.JungleState, error) {
	m := &wire.JungleState{
		Turn: string(s.Turn), MoveCount: int32(s.MoveCount),
		Ended: s.Ended, Winner: string(s.Winner),
		MoveDeadlineAt: s.MoveDeadlineAt, ClockDeadlineAt: s.ClockDeadlineAt,
	}
	for k, v := range s.ClockRemaining {
		m.ClockRemaining = append(m.ClockRemaining, &wire.IntPair{Key: string(k), Value: int32(v)})
	}
	sort.Slice(m.ClockRemaining, func(i, j int) bool { return m.ClockRemaining[i].Key < m.ClockRemaining[j].Key })
	for _, row := range s.Board {
		br := &wire.BoardRow{}
		for _, cell := range row {
			if cell == nil {
				br.Cells = append(br.Cells, "")
			} else {
				br.Cells = append(br.Cells, string(*cell))
			}
		}
		m.Board = append(m.Board, br)
	}
	for k, v := range s.RankedDelta {
		m.RankedDelta = append(m.RankedDelta, &wire.IntPair{Key: string(k), Value: int32(v)})
	}
	sort.Slice(m.RankedDelta, func(i, j int) bool { return m.RankedDelta[i].Key < m.RankedDelta[j].Key })
	if s.LastFrom != nil {
		m.LastFrom = &wire.Pos{Row: int32(s.LastFrom.Row), Col: int32(s.LastFrom.Col)}
	}
	if s.LastTo != nil {
		m.LastTo = &wire.Pos{Row: int32(s.LastTo.Row), Col: int32(s.LastTo.Col)}
	}
	if s.ResignRequest != nil {
		m.ResignRequest = &wire.JungleResignRequest{
			FromSeat: string(s.ResignRequest.FromSeat), ToSeat: string(s.ResignRequest.ToSeat),
			CreatedAt: s.ResignRequest.CreatedAt,
		}
	}
	return m, nil
}

// ConfigToProto 应用配置。
func ConfigToProto(cfg types.AppConfig) (*wire.AppConfig, error) {
	// maps → pairs intermediate
	type cfgMid struct {
		Site types.AppConfig `json:"-"`
		Raw  json.RawMessage
	}
	// simpler: marshal whole config then fix maps via restructure
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var generic map[string]any
	if err := json.Unmarshal(b, &generic); err != nil {
		return nil, err
	}
	// roomInfoTags object → array
	if tags, ok := generic["roomInfoTags"].(map[string]any); ok {
		arr := make([]any, 0, len(tags))
		for k, v := range tags {
			if vm, ok := v.(map[string]any); ok {
				vm["key"] = k
				// style fields are flat in RoomInfoTagStyle with label+colors
				arr = append(arr, map[string]any{"key": k, "style": v})
			}
		}
		generic["roomInfoTags"] = arr
	}
	// messages object → pairs
	if msgs, ok := generic["messages"].(map[string]any); ok {
		arr := make([]any, 0, len(msgs))
		for k, v := range msgs {
			arr = append(arr, map[string]any{"key": k, "value": fmt.Sprint(v)})
		}
		generic["messages"] = arr
	}
	// extremeMode rate maps
	if em, ok := generic["extremeMode"].(map[string]any); ok {
		for _, field := range []string{"positiveLossRates", "negativeWinRates", "hourlyDecay"} {
			if m, ok := em[field].(map[string]any); ok {
				arr := make([]any, 0, len(m))
				for k, v := range m {
					arr = append(arr, map[string]any{"key": k, "value": v})
				}
				em[field] = arr
			}
		}
	}
	// titles factionNames
	if titles, ok := generic["titles"].([]any); ok {
		for _, t := range titles {
			tm, ok := t.(map[string]any)
			if !ok {
				continue
			}
			if fn, ok := tm["factionNames"].(map[string]any); ok {
				arr := make([]any, 0, len(fn))
				for k, v := range fn {
					arr = append(arr, map[string]any{"factionId": k, "names": v})
				}
				tm["factionNames"] = arr
			}
			// variants on punishments handled separately
		}
	}
	if puns, ok := generic["punishments"].([]any); ok {
		for _, p := range puns {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			if v, ok := pm["variants"].(map[string]any); ok {
				arr := make([]any, 0, len(v))
				for k, val := range v {
					arr = append(arr, map[string]any{"key": k, "value": fmt.Sprint(val)})
				}
				pm["variants"] = arr
			}
			if tasks, ok := pm["tasks"].([]any); ok {
				for _, t := range tasks {
					tm, ok := t.(map[string]any)
					if !ok {
						continue
					}
					if v, ok := tm["variants"].(map[string]any); ok {
						arr := make([]any, 0, len(v))
						for k, val := range v {
							arr = append(arr, map[string]any{"key": k, "value": fmt.Sprint(val)})
						}
						tm["variants"] = arr
					}
				}
			}
		}
	}
	bb, err := json.Marshal(generic)
	if err != nil {
		return nil, err
	}
	out := &wire.AppConfig{}
	if err := protoUN.Unmarshal(bb, out); err != nil {
		return nil, fmt.Errorf("config proto: %w", err)
	}
	return out, nil
}

// LobbyProtoToFront 转前端 camelCase 对象（players/rooms 为数组）。
func LobbyProtoToFront(pb *wire.LobbySnapshot) (map[string]any, error) {
	if pb == nil {
		return map[string]any{}, nil
	}
	// Use protojson then expand entries
	raw, err := ProtoToCamelMap(pb)
	if err != nil {
		return nil, err
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return map[string]any{}, nil
	}
	// players: [{id, player}] → [player...]
	if entries, ok := m["players"].([]any); ok {
		players := make([]any, 0, len(entries))
		for _, e := range entries {
			if em, ok := e.(map[string]any); ok {
				if p := em["player"]; p != nil {
					players = append(players, flattenPlayerPresence(p))
				}
			}
		}
		m["players"] = players
	}
	if entries, ok := m["rooms"].([]any); ok {
		rooms := make([]any, 0, len(entries))
		for _, e := range entries {
			if em, ok := e.(map[string]any); ok {
				if r := em["room"]; r != nil {
					rooms = append(rooms, expandLobbyRoom(r))
				}
			}
		}
		m["rooms"] = rooms
	}
	if lb, ok := m["normalLeaderboard"].([]any); ok {
		for i, p := range lb {
			lb[i] = flattenPlayerPresence(p)
		}
	}
	if lb, ok := m["rankedLeaderboard"].([]any); ok {
		for i, p := range lb {
			lb[i] = flattenPlayerPresence(p)
		}
	}
	// roomInfoTags back to object if present in nested config
	if cfg, ok := m["config"].(map[string]any); ok {
		fixConfigMaps(cfg)
	}
	return m, nil
}

func flattenPlayerPresence(p any) any {
	return fillPlayerDefaults(p)
}

// playerBoolFields/playerNumFields：对应 Go domain PublicPlayer/LobbyPlayer 里的 *T 指针字段
// （允许真正的"从未设置"，即 nil）。前端从不区分"从未设置"和"设置为 false/0"（全部只做真值
// 判断），而 protojson EmitUnpopulated:false 又会把 false/0 连 key 一起丢掉——按零值统一补齐，
// 与前端 wire.ts 的 PLAYER_BOOL_FIELDS/PLAYER_NUM_FIELDS 保持一致。
//
// genderId 同理：自定义性别时合法值就是空串，EmitUnpopulated:false 会把 key 整个丢掉；
// 若不补齐成 ""，前端用 {...old, ...patch} 合并时会残留旧的预设 genderId，表现为
// 「展示文案已是自定义、genderId 却还是 male」——重连/改资料后会把自定义盖回预设。
var playerBoolFields = []string{
	"nameWarEnabled", "nameWarPunished", "nameWarAllowRename",
	"giveawayEnabled", "rankMultiplierUnlocked", "extremeModeEnabled",
	"extremeForceClosed", "isAdmin",
	"bondMasterEnabled", "bondPetEnabled", "bondPublicDisplay",
}
var playerNumFields = []string{
	"disconnectedAt", "disconnectExpiresAt", "profileUpdatedAt",
	"nameWarToggledAt", "nameWarRenameProtectedUntil", "nameWarRenameWindowStartedAt", "nameWarRenameCount",
	"giveawayValue", "giveawayClicks", "giveawayBoardSubmittedAt", "giveawayBoardExpiresAt",
	"giveawayBoardLikes", "giveawayBoardDislikes", "giveawayBoardLikesThisHour", "giveawayBoardLikeWindowStartedAt",
	"giveawayVoteWindowStartedAt", "giveawayVoteCount", "giveawayVoteLikesThisHour", "giveawayVoteDislikesThisHour",
	"extremeModeToggledAt", "extremeModeCooldownUntil", "extremeWinStreak", "extremeLastDecayHour",
	"extremeForceClosedAt", "extremeRenameProtectedUntil",
}

func fillPlayerDefaults(p any) any {
	pm, ok := p.(map[string]any)
	if !ok {
		return p
	}
	for _, k := range playerBoolFields {
		if _, exists := pm[k]; !exists {
			pm[k] = false
		}
	}
	for _, k := range playerNumFields {
		if _, exists := pm[k]; !exists {
			pm[k] = float64(0)
		}
	}
	if _, exists := pm["genderId"]; !exists {
		pm["genderId"] = ""
	}
	// 与前端 materializePublicStats / materializeGameStats 对齐，避免 DELTA 哈希因补零键分叉。
	pm["stats"] = fillPublicStatsDefaults(pm["stats"])
	pm["gameStats"] = fillGameStatsDefaults(pm["gameStats"])
	return pm
}

func fillPublicStatsDefaults(s any) any {
	sm, _ := s.(map[string]any)
	if sm == nil {
		sm = map[string]any{}
	}
	for _, k := range []string{
		"wins", "losses", "draws", "punishments", "rankedPoints",
		"highestScore", "lowestScore", "sortRankedPoints", "sortHighestScore", "sortLowestScore",
		"totalOnlineMs",
	} {
		if _, exists := sm[k]; !exists {
			sm[k] = float64(0)
		}
	}
	if _, exists := sm["title"]; !exists {
		sm["title"] = "暂无称号"
	}
	return sm
}

func fillGameWLDDefaults(s any) any {
	sm, _ := s.(map[string]any)
	if sm == nil {
		sm = map[string]any{}
	}
	for _, k := range []string{"wins", "losses", "draws"} {
		if _, exists := sm[k]; !exists {
			sm[k] = float64(0)
		}
	}
	return sm
}

func fillGameStatsDefaults(s any) any {
	sm, _ := s.(map[string]any)
	if sm == nil {
		sm = map[string]any{}
	}
	for _, g := range []string{"rps", "othello", "tictactoe", "gomoku", "liarsdice", "jungle"} {
		sm[g] = fillGameWLDDefaults(sm[g])
	}
	return sm
}

// allowProofImage 对应 *bool 指针，但 handlers_room.go 建房时已把 nil 归一化成
// true/false 的具体值，线上真正存在的房间永远不会是"未设置"——false 就是它的零值，
// 缺省按 false 补齐即可无损还原（true 是非零值，protojson 永远不会丢，不受影响）。
// gomokuUndoLimit 同理：建房时已校验为 0/1/3/10 之一，键缺失只可能是真值 0（禁止悔棋）
// 被 protojson EmitUnpopulated:false 连 key 一起丢掉，缺省按 0 补齐即可无损还原。
func fillRoomSettingsDefaults(s any) any {
	sm, ok := s.(map[string]any)
	if !ok {
		return s
	}
	if _, exists := sm["allowProofImage"]; !exists {
		sm["allowProofImage"] = false
	}
	if _, exists := sm["gomokuUndoLimit"]; !exists {
		sm["gomokuUndoLimit"] = float64(0)
	}
	// 0 = 不限时；protojson 会丢掉数值 0，与前端 fillRoomSettingsDefaults 对齐。
	for _, k := range []string{"othelloMoveSeconds", "othelloGameMinutes", "gomokuMoveSeconds", "gomokuGameMinutes", "jungleMoveSeconds", "jungleGameMinutes"} {
		if _, exists := sm[k]; !exists {
			sm[k] = float64(0)
		}
	}
	return sm
}

// reviewedAt 对应 *int64 指针，缺省即"尚未审核"。
func fillReviewedAtDefault(p any) any {
	pm, ok := p.(map[string]any)
	if !ok {
		return p
	}
	if _, exists := pm["reviewedAt"]; !exists {
		pm["reviewedAt"] = float64(0)
	}
	return pm
}

// backgroundOpacity 对应 *float64 指针，缺省按 0 补齐。
func fillBackgroundOpacityDefault(t any) any {
	tm, ok := t.(map[string]any)
	if !ok {
		return t
	}
	if _, exists := tm["backgroundOpacity"]; !exists {
		tm["backgroundOpacity"] = float64(0)
	}
	return tm
}

// stake/rankMultiplier/effectiveStake 对应 *int 指针，缺省即"非排位/未生效"，按 0 补齐；
// 并顺带补齐嵌套的 proofs（HistoryProof.reviewedAt）与 punishmentTasks（backgroundOpacity）。
func fillRoundHistoryItemDefaults(item any) any {
	im, ok := item.(map[string]any)
	if !ok {
		return item
	}
	if _, exists := im["stake"]; !exists {
		im["stake"] = float64(0)
	}
	if _, exists := im["rankMultiplier"]; !exists {
		im["rankMultiplier"] = float64(0)
	}
	if _, exists := im["effectiveStake"]; !exists {
		im["effectiveStake"] = float64(0)
	}
	if proofs, ok := im["proofs"].([]any); ok {
		for i, p := range proofs {
			proofs[i] = fillReviewedAtDefault(p)
		}
	}
	if tasks, ok := im["punishmentTasks"].([]any); ok {
		for i, t := range tasks {
			tasks[i] = fillBackgroundOpacityDefault(t)
		}
	}
	return im
}

func expandLobbyRoom(r any) any {
	rm, ok := r.(map[string]any)
	if !ok {
		return r
	}
	if vs, ok := rm["versus"].([]any); ok {
		obj := map[string]any{"A": nil, "B": nil}
		for _, item := range vs {
			im, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key, _ := im["key"].(string)
			val := im["value"]
			if vm, ok := val.(map[string]any); ok {
				if vm["player"] != nil {
					obj[key] = map[string]any{"player": fillPlayerDefaults(vm["player"])}
				}
			}
		}
		rm["versus"] = obj
	}
	return rm
}

func fixConfigMaps(cfg map[string]any) {
	if tags, ok := cfg["roomInfoTags"].([]any); ok {
		obj := map[string]any{}
		for _, t := range tags {
			tm, ok := t.(map[string]any)
			if !ok {
				continue
			}
			k, _ := tm["key"].(string)
			if st := tm["style"]; st != nil {
				obj[k] = st
			}
		}
		cfg["roomInfoTags"] = obj
	}
	if msgs, ok := cfg["messages"].([]any); ok {
		obj := map[string]any{}
		for _, t := range msgs {
			tm, ok := t.(map[string]any)
			if !ok {
				continue
			}
			k, _ := tm["key"].(string)
			obj[k] = tm["value"]
		}
		cfg["messages"] = obj
	}
	if em, ok := cfg["extremeMode"].(map[string]any); ok {
		for _, field := range []string{"positiveLossRates", "negativeWinRates", "hourlyDecay"} {
			if arr, ok := em[field].([]any); ok {
				obj := map[string]any{}
				for _, t := range arr {
					tm, ok := t.(map[string]any)
					if !ok {
						continue
					}
					k, _ := tm["key"].(string)
					// value 合法取值含 0（如某难度掉分率为 0%），EmitUnpopulated:false 会把它连 key
					// 一起丢掉，tm["value"] 缺失时 Go 侧读到 nil——补回 0，否则后台数字框显示空白。
					v := tm["value"]
					if v == nil {
						v = float64(0)
					}
					obj[k] = v
				}
				em[field] = obj
			}
		}
	}
}

// RoomProtoToFront room → camelCase UI tree.
func RoomProtoToFront(pb *wire.RoomSnapshot) (map[string]any, error) {
	raw, err := ProtoToCamelMap(pb)
	if err != nil {
		return nil, err
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return map[string]any{}, nil
	}
	// seats pairs → {A,B}
	m["seats"] = pairsToOccupantMap(m["seats"])
	m["ready"] = pairsToBoolMap(m["ready"])
	m["choices"] = pairsToStringMap(m["choices"])
	m["revealedChoices"] = pairsToStringMap(m["revealedChoices"])
	m["score"] = pairsToIntMap(m["score"])
	m["seatedScore"] = pairsToIntMap(m["seatedScore"])
	m["seatStats"] = pairsToSeatStatsMap(m["seatStats"])
	if o, ok := m["othello"].(map[string]any); ok {
		o["board"] = boardRowsToMatrix(o["board"])
		o["rankedDelta"] = pairsToIntMap(o["rankedDelta"])
		o["clockRemaining"] = pairsToIntMap(o["clockRemaining"])
		m["othello"] = o
	}
	if t, ok := m["tictactoe"].(map[string]any); ok {
		t["board"] = boardRowsToMatrix(t["board"])
		t["rankedDelta"] = pairsToIntMap(t["rankedDelta"])
		m["tictactoe"] = t
	}
	if l, ok := m["liarsDice"].(map[string]any); ok {
		l["diceCounts"] = pairsToGenericIntMap(l["diceCounts"])
		l["revealedHands"] = pairsToIntListMap(l["revealedHands"])
		m["liarsDice"] = l
	}
	if g, ok := m["gomoku"].(map[string]any); ok {
		g["board"] = boardRowsToMatrix(g["board"])
		g["rankedDelta"] = pairsToIntMap(g["rankedDelta"])
		g["undoCount"] = pairsToIntMap(g["undoCount"])
		g["clockRemaining"] = pairsToIntMap(g["clockRemaining"])
		m["gomoku"] = g
	}
	if j, ok := m["jungle"].(map[string]any); ok {
		j["board"] = boardRowsToMatrix(j["board"])
		j["rankedDelta"] = pairsToIntMap(j["rankedDelta"])
		j["clockRemaining"] = pairsToIntMap(j["clockRemaining"])
		m["jungle"] = j
	}
	if hist, ok := m["roundHistory"].([]any); ok {
		for i, item := range hist {
			im, ok := item.(map[string]any)
			if !ok {
				continue
			}
			im["liarsDiceHands"] = pairsToIntListMap(im["liarsDiceHands"])
			im["liarsDiceNames"] = pairsToStringMap(im["liarsDiceNames"])
			hist[i] = fillRoundHistoryItemDefaults(im)
		}
	}
	// 与前端 wire.materializeRoom 对齐（allowProofImage:false / presence zero 等）
	m["settings"] = fillRoomSettingsDefaults(m["settings"])
	if seats, ok := m["seats"].(map[string]any); ok {
		for k, v := range seats {
			seats[k] = fillPlayerDefaults(v)
		}
	}
	if spectators, ok := m["spectators"].([]any); ok {
		for i, p := range spectators {
			spectators[i] = fillPlayerDefaults(p)
		}
	}
	if proofs, ok := m["proofs"].([]any); ok {
		for i, p := range proofs {
			proofs[i] = fillReviewedAtDefault(p)
		}
	}
	if chat, ok := m["chat"].([]any); ok {
		for i, c := range chat {
			cm, ok := c.(map[string]any)
			if !ok {
				continue
			}
			if _, exists := cm["expiresAt"]; !exists {
				cm["expiresAt"] = float64(0)
			}
			chat[i] = cm
		}
	}
	return m, nil
}

func pairsToOccupantMap(v any) map[string]any {
	out := map[string]any{"A": nil, "B": nil}
	arr, ok := v.([]any)
	if !ok {
		return out
	}
	for _, item := range arr {
		im, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key, _ := im["key"].(string)
		val, _ := im["value"].(map[string]any)
		if val == nil {
			out[key] = nil
			continue
		}
		if val["player"] != nil {
			out[key] = val["player"]
		}
	}
	return out
}

func pairsToBoolMap(v any) map[string]any {
	out := map[string]any{"A": false, "B": false}
	arr, _ := v.([]any)
	for _, item := range arr {
		im, _ := item.(map[string]any)
		if im == nil {
			continue
		}
		k, _ := im["key"].(string)
		// proto3 省略 false → 只有 key，无 value
		if val, ok := im["value"]; ok {
			out[k] = val
		} else {
			out[k] = false
		}
	}
	return out
}

func pairsToStringMap(v any) map[string]any {
	out := map[string]any{}
	arr, _ := v.([]any)
	for _, item := range arr {
		im, _ := item.(map[string]any)
		if im == nil {
			continue
		}
		k, _ := im["key"].(string)
		if val, ok := im["value"]; ok {
			out[k] = val
		} else {
			out[k] = ""
		}
	}
	return out
}

func pairsToIntMap(v any) map[string]any {
	out := map[string]any{"A": 0, "B": 0}
	arr, _ := v.([]any)
	for _, item := range arr {
		im, _ := item.(map[string]any)
		if im == nil {
			continue
		}
		k, _ := im["key"].(string)
		// proto3 省略 0 → 只有 key，无 value
		if val, ok := im["value"]; ok && val != nil {
			out[k] = val
		} else {
			out[k] = float64(0)
		}
	}
	return out
}

// pairsToGenericIntMap：不像 pairsToIntMap 那样预置 A/B——大话骰的 key 是任意 playerId。
func pairsToGenericIntMap(v any) map[string]any {
	out := map[string]any{}
	arr, _ := v.([]any)
	for _, item := range arr {
		im, _ := item.(map[string]any)
		if im == nil {
			continue
		}
		k, _ := im["key"].(string)
		if val, ok := im["value"]; ok && val != nil {
			out[k] = val
		} else {
			out[k] = float64(0)
		}
	}
	return out
}

func pairsToIntListMap(v any) map[string]any {
	out := map[string]any{}
	arr, _ := v.([]any)
	for _, item := range arr {
		im, _ := item.(map[string]any)
		if im == nil {
			continue
		}
		k, _ := im["key"].(string)
		val, _ := im["value"].(map[string]any)
		if vals, ok := val["values"].([]any); ok {
			out[k] = vals
		} else {
			out[k] = []any{}
		}
	}
	return out
}

func pairsToSeatStatsMap(v any) map[string]any {
	out := map[string]any{
		"A": map[string]any{"wins": 0, "losses": 0, "draws": 0, "punishments": 0},
		"B": map[string]any{"wins": 0, "losses": 0, "draws": 0, "punishments": 0},
	}
	arr, _ := v.([]any)
	for _, item := range arr {
		im, _ := item.(map[string]any)
		if im == nil {
			continue
		}
		k, _ := im["key"].(string)
		out[k] = im["value"]
	}
	return out
}

func boardRowsToMatrix(v any) []any {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]any, 0, len(arr))
	for _, row := range arr {
		rm, ok := row.(map[string]any)
		if !ok {
			continue
		}
		cells, _ := rm["cells"].([]any)
		line := make([]any, 0, len(cells))
		for _, c := range cells {
			s, _ := c.(string)
			if s == "" {
				line = append(line, nil)
			} else {
				line = append(line, s)
			}
		}
		out = append(out, line)
	}
	return out
}

// ConfigProtoToFront config maps restore.
func ConfigProtoToFront(pb *wire.AppConfig) (map[string]any, error) {
	raw, err := ProtoToCamelMap(pb)
	if err != nil {
		return nil, err
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return map[string]any{}, nil
	}
	fixConfigMaps(m)
	// titles factionNames
	if titles, ok := m["titles"].([]any); ok {
		for _, t := range titles {
			tm, ok := t.(map[string]any)
			if !ok {
				continue
			}
			if arr, ok := tm["factionNames"].([]any); ok {
				obj := map[string]any{}
				for _, e := range arr {
					em, ok := e.(map[string]any)
					if !ok {
						continue
					}
					fid, _ := em["factionId"].(string)
					obj[fid] = em["names"]
				}
				tm["factionNames"] = obj
			}
		}
	}
	if puns, ok := m["punishments"].([]any); ok {
		for _, p := range puns {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			pm["variants"] = pairsToStringMap(pm["variants"])
			if tasks, ok := pm["tasks"].([]any); ok {
				for _, t := range tasks {
					tm, ok := t.(map[string]any)
					if !ok {
						continue
					}
					tm["variants"] = pairsToStringMap(tm["variants"])
				}
			}
		}
	}
	return m, nil
}

// BuildRawBody 将 reply/push 数据封进 RawBody。
func BuildRawBody(event string, data any) (*wire.RawBody, error) {
	if data == nil {
		return &wire.RawBody{}, nil
	}
	switch event {
	case "player:batch":
		// data is []PublicPlayer or []any
		batch := &wire.PlayerBatch{}
		b, _ := json.Marshal(data)
		var list []types.PublicPlayer
		if err := json.Unmarshal(b, &list); err == nil {
			for _, p := range list {
				pp, err := publicPlayerToProto(p)
				if err != nil {
					return nil, err
				}
				batch.Players = append(batch.Players, pp)
			}
			return &wire.RawBody{Body: &wire.RawBody_PlayerBatch{PlayerBatch: batch}}, nil
		}
	case "chat:append":
		b, _ := json.Marshal(data)
		var c types.ChatMessage
		if json.Unmarshal(b, &c) == nil {
			cc, err := chatToProto(c)
			if err != nil {
				return nil, err
			}
			return &wire.RawBody{Body: &wire.RawBody_Chat{Chat: cc}}, nil
		}
	case "suggestion:append":
		b, _ := json.Marshal(data)
		var s types.Suggestion
		if json.Unmarshal(b, &s) == nil {
			ss, err := suggestionToProto(s)
			if err != nil {
				return nil, err
			}
			return &wire.RawBody{Body: &wire.RawBody_Suggestion{Suggestion: ss}}, nil
		}
	case "player:update", "player:kicked":
		// sometimes full player
		b, _ := json.Marshal(data)
		var p types.PublicPlayer
		if json.Unmarshal(b, &p) == nil && p.ID != "" {
			pp, err := publicPlayerToProto(p)
			if err != nil {
				return nil, err
			}
			return &wire.RawBody{Body: &wire.RawBody_Player{Player: pp}}, nil
		}
	case "announcement:show":
		st, err := AnyToStruct(data)
		if err != nil {
			return nil, err
		}
		// prefer typed if shape matches
		b, _ := json.Marshal(data)
		var a struct {
			ID         string `json:"id"`
			Message    string `json:"message"`
			DurationMs int32  `json:"durationMs"`
			CreatedAt  int64  `json:"createdAt"`
		}
		if json.Unmarshal(b, &a) == nil && a.ID != "" {
			return &wire.RawBody{Body: &wire.RawBody_Announcement{Announcement: &wire.AnnouncementPayload{
				Id: a.ID, Message: a.Message, DurationMs: a.DurationMs, CreatedAt: a.CreatedAt,
			}}}, nil
		}
		_ = st
	case "room:closed":
		b, _ := json.Marshal(data)
		var rc struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(b, &rc)
		return &wire.RawBody{Body: &wire.RawBody_RoomClosed{RoomClosed: &wire.RoomClosed{Message: rc.Message}}}, nil
	case "room:historyAppend":
		b, _ := json.Marshal(data)
		var hp struct {
			RoomID string                 `json:"roomId"`
			Item   types.RoundHistoryItem `json:"item"`
			Total  int                    `json:"total"`
		}
		if json.Unmarshal(b, &hp) == nil {
			item, err := historyToProto(hp.Item)
			if err != nil {
				return nil, err
			}
			return &wire.RawBody{Body: &wire.RawBody_HistoryPage{HistoryPage: &wire.HistoryPage{
				RoomId: hp.RoomID, Item: item, Total: int32(hp.Total),
			}}}, nil
		}
	}
	// 默认：Struct 动态
	st, err := AnyToStruct(data)
	if err != nil {
		return nil, err
	}
	return &wire.RawBody{Body: &wire.RawBody_Dynamic{Dynamic: st}}, nil
}

// RawBodyToFront 解码 RAW 为前端对象。
func RawBodyToFront(body *wire.RawBody) (any, error) {
	if body == nil || body.Body == nil {
		return map[string]any{}, nil
	}
	switch b := body.Body.(type) {
	case *wire.RawBody_Dynamic:
		if b.Dynamic == nil {
			return map[string]any{}, nil
		}
		return b.Dynamic.AsMap(), nil
	case *wire.RawBody_PlayerBatch:
		list := make([]any, 0, len(b.PlayerBatch.GetPlayers()))
		for _, p := range b.PlayerBatch.GetPlayers() {
			m, err := ProtoToCamelMap(p)
			if err != nil {
				return nil, err
			}
			list = append(list, m)
		}
		return list, nil
	case *wire.RawBody_Chat:
		return ProtoToCamelMap(b.Chat)
	case *wire.RawBody_Suggestion:
		return ProtoToCamelMap(b.Suggestion)
	case *wire.RawBody_Player:
		return ProtoToCamelMap(b.Player)
	case *wire.RawBody_Me:
		return ProtoToCamelMap(b.Me)
	case *wire.RawBody_Announcement:
		return ProtoToCamelMap(b.Announcement)
	case *wire.RawBody_RoomClosed:
		return ProtoToCamelMap(b.RoomClosed)
	case *wire.RawBody_HistoryPage:
		m, err := ProtoToCamelMap(b.HistoryPage)
		return m, err
	case *wire.RawBody_Ok:
		return map[string]any{"ok": b.Ok.GetOk()}, nil
	case *wire.RawBody_PlayerResult:
		m, err := ProtoToCamelMap(b.PlayerResult.GetPlayer())
		if err != nil {
			return nil, err
		}
		return map[string]any{"player": m}, nil
	case *wire.RawBody_Suggestions:
		return ProtoToCamelMap(b.Suggestions)
	case *wire.RawBody_Room:
		return RoomProtoToFront(b.Room)
	case *wire.RawBody_Config:
		return ConfigProtoToFront(b.Config)
	case *wire.RawBody_Lobby:
		return LobbyProtoToFront(b.Lobby)
	default:
		return map[string]any{}, nil
	}
}

// StateDocToFront FULL 状态 → 前端对象（与 web materialize 对齐，供 Diff/Hash）。
func StateDocToFront(doc *wire.StateDocument) (any, error) {
	if doc == nil {
		return nil, fmt.Errorf("nil state doc")
	}
	switch d := doc.Doc.(type) {
	case *wire.StateDocument_Lobby:
		m, err := LobbyProtoToFront(d.Lobby)
		if err != nil {
			return nil, err
		}
		return NormalizeFrontTree(m), nil
	case *wire.StateDocument_Room:
		m, err := RoomProtoToFront(d.Room)
		if err != nil {
			return nil, err
		}
		return NormalizeFrontTree(m), nil
	case *wire.StateDocument_Config:
		m, err := ConfigProtoToFront(d.Config)
		if err != nil {
			return nil, err
		}
		return NormalizeFrontTree(m), nil
	default:
		return nil, fmt.Errorf("unknown state doc")
	}
}

// NormalizeFrontTree 对齐前端 wire.materialize*（presence 零值补齐已在 LobbyProtoToFront/
// RoomProtoToFront 里做过，这里只负责剩下的）：
// - int 0 不得丢失（legalMoves.row/col）
// - 棋盘 pad 完整尺寸
// - nil 数组 → []
func NormalizeFrontTree(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = NormalizeFrontTree(val)
		}
		// 房间级修正
		if othello, ok := out["othello"].(map[string]any); ok {
			othello["board"] = padBoardAny(othello["board"], 8)
			othello["legalMoves"] = normalizePosListAny(othello["legalMoves"])
			othello["blackCount"] = numAny(othello["blackCount"], 0)
			othello["whiteCount"] = numAny(othello["whiteCount"], 0)
			othello["passCount"] = numAny(othello["passCount"], 0)
			othello["moveDeadlineAt"] = numAny(othello["moveDeadlineAt"], 0)
			othello["clockDeadlineAt"] = numAny(othello["clockDeadlineAt"], 0)
			if othello["settlementEvents"] == nil {
				othello["settlementEvents"] = []any{}
			}
			out["othello"] = othello
		}
		if ttt, ok := out["tictactoe"].(map[string]any); ok {
			ttt["board"] = padBoardAny(ttt["board"], 3)
			ttt["winningLine"] = normalizePosListAny(ttt["winningLine"])
			ttt["moveCount"] = numAny(ttt["moveCount"], 0)
			out["tictactoe"] = ttt
		}
		if gomoku, ok := out["gomoku"].(map[string]any); ok {
			gomoku["board"] = padBoardAny(gomoku["board"], 15)
			gomoku["moves"] = normalizePosListAny(gomoku["moves"])
			gomoku["winningLine"] = normalizePosListAny(gomoku["winningLine"])
			gomoku["moveCount"] = numAny(gomoku["moveCount"], 0)
			gomoku["moveDeadlineAt"] = numAny(gomoku["moveDeadlineAt"], 0)
			gomoku["clockDeadlineAt"] = numAny(gomoku["clockDeadlineAt"], 0)
			out["gomoku"] = gomoku
		}
		if jungle, ok := out["jungle"].(map[string]any); ok {
			jungle["board"] = padBoardRectAny(jungle["board"], 9, 7)
			jungle["moveCount"] = numAny(jungle["moveCount"], 0)
			jungle["moveDeadlineAt"] = numAny(jungle["moveDeadlineAt"], 0)
			jungle["clockDeadlineAt"] = numAny(jungle["clockDeadlineAt"], 0)
			// protojson 会丢弃 row/col 为 0 的 key；必须与 web/src/wire.ts:255-256 的
			// normalizeStateTree 逐字节对齐，否则落在第 0 行/列的落子会让两端状态树
			// 少一个 key，DELTA 的 CRC32 哈希立即分歧，触发 sync:full。
			for _, k := range []string{"lastFrom", "lastTo"} {
				if p, ok := jungle[k].(map[string]any); ok {
					p["row"] = numAny(p["row"], 0)
					p["col"] = numAny(p["col"], 0)
					jungle[k] = p
				}
			}
			out["jungle"] = jungle
		}
		if ss, ok := out["serverStats"].(map[string]any); ok {
			for _, k := range []string{
				"startedAt", "roomBroadcasts", "lobbyBroadcasts", "disconnects", "reconnects",
				"lastRoomSnapshotBytes", "lastLobbySnapshotBytes",
				"recentRoomBroadcasts", "recentLobbyBroadcasts",
				"averageRoomSnapshotBytes", "averageLobbySnapshotBytes",
			} {
				ss[k] = numAny(ss[k], 0)
			}
			out["serverStats"] = ss
		}
		for _, arrKey := range []string{"players", "rooms", "spectators", "proofs", "roundHistory", "chat", "punishedPlayerIds", "suggestions", "lobbyChat", "normalLeaderboard", "rankedLeaderboard"} {
			if out[arrKey] == nil {
				// 仅当 key 存在语义上应为数组的字段：若完全没有 key 不一定补
				continue
			}
			if _, ok := out[arrKey].([]any); !ok && out[arrKey] != nil {
				continue
			}
			if out[arrKey] == nil {
				out[arrKey] = []any{}
			}
		}
		// 保证房间关键数组非 null
		for _, arrKey := range []string{"spectators", "proofs", "roundHistory", "chat", "punishedPlayerIds"} {
			if _, has := out["id"]; has { // room-like
				if out[arrKey] == nil {
					out[arrKey] = []any{}
				}
			}
		}
		if players, ok := out["players"]; ok && players == nil {
			out["players"] = []any{}
		}
		if rooms, ok := out["rooms"]; ok && rooms == nil {
			out["rooms"] = []any{}
		}
		if hist, ok := out["roundHistory"].([]any); ok {
			for i := range hist {
				if hm, ok := hist[i].(map[string]any); ok {
					hm["tictactoeLine"] = normalizePosListAny(hm["tictactoeLine"])
					hm["gomokuLine"] = normalizePosListAny(hm["gomokuLine"])
					if hm["punishmentTasks"] == nil {
						hm["punishmentTasks"] = []any{}
					}
					if hm["punishedNames"] == nil {
						hm["punishedNames"] = []any{}
					}
					if hm["proofs"] == nil {
						hm["proofs"] = []any{}
					}
					hist[i] = hm
				}
			}
			out["roundHistory"] = hist
		}
		// 房间 settings 再走一次计时字段补零（DELTA 可能只补丁部分字段）。
		if settings, ok := out["settings"]; ok {
			out["settings"] = fillRoomSettingsDefaults(settings)
		}
		return out
	case []any:
		if t == nil {
			return []any{}
		}
		out := make([]any, len(t))
		for i := range t {
			out[i] = NormalizeFrontTree(t[i])
		}
		return out
	default:
		return v
	}
}

func padBoardAny(v any, n int) []any {
	return padBoardRectAny(v, n, n)
}

func padBoardRectAny(v any, rows, cols int) []any {
	src, _ := v.([]any)
	out := make([]any, rows)
	for r := 0; r < rows; r++ {
		var rowSrc []any
		if r < len(src) {
			rowSrc, _ = src[r].([]any)
		}
		row := make([]any, cols)
		for c := 0; c < cols; c++ {
			if c < len(rowSrc) {
				row[c] = rowSrc[c]
			} else {
				row[c] = nil
			}
		}
		out[r] = row
	}
	return out
}

func normalizePosListAny(v any) []any {
	src, ok := v.([]any)
	if !ok || src == nil {
		return []any{}
	}
	out := make([]any, 0, len(src))
	for _, p := range src {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, map[string]any{
			"row": numAny(pm["row"], 0),
			"col": numAny(pm["col"], 0),
		})
	}
	return out
}

func numAny(v any, fallback float64) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int32:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, err := t.Float64()
		if err == nil {
			return f
		}
	}
	return fallback
}
