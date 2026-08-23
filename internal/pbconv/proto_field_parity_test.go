package pbconv

import (
	"reflect"
	"strings"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
	"github.com/doumiao/newRPS/internal/wire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// 领域 JSON 字段名 → 期望存在的 proto JSON 名（camelCase）。
// 仅覆盖「必须过 wire 同步、漏了就会静默丢」的类型；纯服务端内部字段不要加进来。
func TestDomainWireFieldParity(t *testing.T) {
	cases := []struct {
		name   string
		domain any
		msg    proto.Message
		// skip：领域有、但有意不进 wire 的 json 名
		skip map[string]bool
	}{
		{
			name:   "LobbyPlayer",
			domain: types.LobbyPlayer{},
			msg:    &wire.LobbyPlayer{},
		},
		{
			name:   "PublicPlayer",
			domain: types.PublicPlayer{},
			// rankedLastDecayDay 仅服务端日衰减用，不下发客户端
			msg:  &wire.PublicPlayer{},
			skip: map[string]bool{"rankedLastDecayDay": true},
		},
		{
			name:   "PublicStats",
			domain: types.PublicStats{},
			msg:    &wire.PublicStats{},
		},
		{
			name:   "LobbyStats",
			domain: types.LobbyStats{},
			msg:    &wire.LobbyStats{},
			// contributionApprovedCount 只随 players:roster RPC 应答下发，该 RPC 走
			// pbconv.BuildRawBody 的默认「Struct 动态」分支（generic json→Struct），
			// 不经过 lobbyStatsToProto 这条固定 wire message 路径，因此无需在
			// api/proto/game.proto 里加对应字段。
			skip: map[string]bool{"contributionApprovedCount": true},
		},
		{
			name:   "ChatMessage",
			domain: types.ChatMessage{},
			msg:    &wire.ChatMessage{},
		},
		{
			name:   "RoomSettings",
			domain: types.RoomSettings{},
			msg:    &wire.RoomSettings{},
		},
		{
			name:   "LobbyRoomInfo",
			domain: types.LobbyRoomInfo{},
			msg:    &wire.LobbyRoomInfo{},
		},
		{
			name:   "GiveawayConfig",
			domain: types.AppConfig{}.Giveaway,
			msg:    &wire.GiveawayConfig{},
		},
		{
			name:   "AccessControlConfig",
			domain: types.AppConfig{}.AccessControl,
			msg:    &wire.AccessControlConfig{},
		},
		{
			name:   "RankedScoreConfig",
			domain: types.RankedScoreConfig{},
			msg:    &wire.RankedScoreConfig{},
		},
		{
			name:   "NameWarConfig",
			domain: types.AppConfig{}.NameWar,
			msg:    &wire.NameWarConfig{},
		},
		{
			name:   "PunishmentTask",
			domain: types.PunishmentTask{},
			msg:    &wire.PunishmentTask{},
			skip:   map[string]bool{"rejectCount": true},
		},
		{
			name:   "PunishmentSeriesSummary",
			domain: types.PunishmentSeriesSummary{},
			msg:    &wire.PunishmentSeriesSummary{},
		},
		{
			name:   "PunishmentRandomSettings",
			domain: types.PunishmentRandomSettings{},
			msg:    &wire.PunishmentRandomSettings{},
		},
		{
			name:   "OthelloState",
			domain: types.OthelloState{},
			msg:    &wire.OthelloState{},
		},
		{
			name:   "GomokuState",
			domain: types.GomokuState{},
			msg:    &wire.GomokuState{},
		},
		{
			name:   "ChessState",
			domain: types.ChessState{},
			msg:    &wire.ChessState{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			domainJSON := jsonFieldNames(reflect.TypeOf(tc.domain))
			protoJSON := protoJSONNames(tc.msg.ProtoReflect().Descriptor())
			for _, name := range domainJSON {
				if tc.skip[name] {
					continue
				}
				if !protoJSON[name] {
					t.Errorf("domain field %q missing on proto %s (add to api/proto/game.proto and regenerate)", name, tc.name)
				}
			}
		})
	}
}

func TestPunishmentWireFieldNumbersRemainCompatible(t *testing.T) {
	taskFields := (&wire.PunishmentTaskConfig{}).ProtoReflect().Descriptor().Fields()
	wantTask := map[protoreflect.Name]protoreflect.FieldNumber{
		"variants":           3,
		"background_images":  4,
		"background_opacity": 5,
		"text":               6,
		"faction_ids":        7,
		"order":              8,
		"tag_ids":            9,
	}
	for name, number := range wantTask {
		field := taskFields.ByName(name)
		if field == nil || field.Number() != number {
			t.Errorf("PunishmentTaskConfig.%s field number = %v, want %d", name, fieldNumber(field), number)
		}
	}

	configFields := (&wire.PunishmentConfig{}).ProtoReflect().Descriptor().Fields()
	for name, number := range map[protoreflect.Name]protoreflect.FieldNumber{
		"description": 3,
		"variants":    4,
		"tasks":       5,
	} {
		field := configFields.ByName(name)
		if field == nil || field.Number() != number {
			t.Errorf("PunishmentConfig.%s field number = %v, want %d", name, fieldNumber(field), number)
		}
	}
}

func fieldNumber(field protoreflect.FieldDescriptor) protoreflect.FieldNumber {
	if field == nil {
		return 0
	}
	return field.Number()
}

func jsonFieldNames(t reflect.Type) []string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	var out []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous {
			// 嵌入字段展开（如 GenderColors）
			out = append(out, jsonFieldNames(f.Type)...)
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name != "" && name != "-" {
			out = append(out, name)
		}
	}
	return out
}

func protoJSONNames(d protoreflect.MessageDescriptor) map[string]bool {
	out := map[string]bool{}
	fields := d.Fields()
	for i := 0; i < fields.Len(); i++ {
		// protojson UseProtoNames:false → JSON name is camelCase of the field
		out[string(fields.Get(i).JSONName())] = true
	}
	return out
}
