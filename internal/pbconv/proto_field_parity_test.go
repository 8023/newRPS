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
			name:   "OthelloState",
			domain: types.OthelloState{},
			msg:    &wire.OthelloState{},
		},
		{
			name:   "GomokuState",
			domain: types.GomokuState{},
			msg:    &wire.GomokuState{},
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
