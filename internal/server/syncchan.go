package server

import (
	"encoding/json"

	"github.com/doumiao/newRPS/internal/delta"
	"github.com/doumiao/newRPS/internal/pbconv"
	"github.com/doumiao/newRPS/internal/types"
	"github.com/doumiao/newRPS/internal/wire"
	"google.golang.org/protobuf/types/known/structpb"
)

// 同步通道：Protobuf FULL（StateDocument）+ 路径树 DELTA（PatchOp + Value）。
// 哈希对「前端形态」规范化树做 CRC-32（IEEE），与客户端 crc32Hex 对齐；不一致则 sync:full。
// 广播防抖 / player:batch / chat:append 不在此文件，调用方逻辑保持不变。

const (
	// 补丁条数过多时回退 FULL（与旧 JSON-DELTA 策略同量级）
	maxDeltaOps = 80
	// 补丁体积接近全量时回退 FULL
	deltaSizeRatioNum = 4
	deltaSizeRatioDen = 5
)

type syncChannel struct {
	name string
	seq  uint64
	hash string
	doc  any // 上一份前端形态状态树（map/slice），供 Diff
}

func (s *Server) ensureSync() {
	if s.syncChans == nil {
		s.syncChans = map[string]*syncChannel{}
	}
}

func (s *Server) getSync(name string) *syncChannel {
	s.ensureSync()
	ch := s.syncChans[name]
	if ch == nil {
		ch = &syncChannel{name: name}
		s.syncChans[name] = ch
	}
	return ch
}

func (s *Server) resetSyncChannel(name string) {
	ch := s.getSync(name)
	ch.doc = nil
	ch.hash = ""
}

// buildStateEnvelope：有历史则尝试 DELTA，否则 FULL；无变化返回 (nil,0,nil)。
func (s *Server) buildStateEnvelope(event, channel string, value any) (*wire.Envelope, int, error) {
	fullDoc, tree, hash, err := buildStateTreeAndFull(channel, value)
	if err != nil {
		return nil, 0, err
	}
	ch := s.getSync(channel)

	// 无变化
	if ch.doc != nil && ch.hash == hash {
		return nil, 0, nil
	}

	ch.seq++
	env := &wire.Envelope{
		Event:   event,
		Channel: channel,
		Seq:     ch.seq,
		Hash:    hash,
	}

	// 首包或强制无历史 → FULL
	if ch.doc == nil {
		env.Kind = wire.PayloadKind_KIND_FULL
		env.FullState = fullDoc
		ch.doc = tree
		ch.hash = hash
		nbytes := estimateTreeBytes(tree)
		return env, nbytes, nil
	}

	ops, err := delta.Diff(ch.doc, tree)
	if err != nil {
		return nil, 0, err
	}
	if len(ops) == 0 {
		// 树相等但 hash 不同（不应出现）；仍刷新本地
		ch.doc = tree
		ch.hash = hash
		return nil, 0, nil
	}

	fullSize := estimateTreeBytes(tree)
	wireOps, opsBytes, err := deltaOpsToWire(ops)
	if err != nil {
		return nil, 0, err
	}

	// 过大 / 过多 → FULL
	if len(wireOps) > maxDeltaOps || (fullSize > 0 && opsBytes*deltaSizeRatioDen > fullSize*deltaSizeRatioNum) {
		env.Kind = wire.PayloadKind_KIND_FULL
		env.FullState = fullDoc
		ch.doc = tree
		ch.hash = hash
		return env, fullSize, nil
	}

	env.Kind = wire.PayloadKind_KIND_DELTA
	env.Delta = &wire.StateDelta{Ops: wireOps}
	ch.doc = tree
	ch.hash = hash
	return env, opsBytes, nil
}

// buildFullEnvelope 强制全量（resync / 首包）。
func (s *Server) buildFullEnvelope(event, channel string, value any) (*wire.Envelope, int, error) {
	fullDoc, tree, hash, err := buildStateTreeAndFull(channel, value)
	if err != nil {
		return nil, 0, err
	}
	ch := s.getSync(channel)
	ch.seq++
	ch.doc = tree
	ch.hash = hash
	return &wire.Envelope{
		Event:     event,
		Kind:      wire.PayloadKind_KIND_FULL,
		Channel:   channel,
		Seq:       ch.seq,
		Hash:      hash,
		FullState: fullDoc,
	}, estimateTreeBytes(tree), nil
}

// buildStateTreeAndFull：领域对象 → 类型化 FULL + 与客户端一致的前端树 + 树哈希。
func buildStateTreeAndFull(channel string, value any) (*wire.StateDocument, any, string, error) {
	fullDoc, _, err := marshalStateDocument(channel, value)
	if err != nil {
		return nil, nil, "", err
	}
	tree, err := pbconv.StateDocToFront(fullDoc)
	if err != nil {
		return nil, nil, "", err
	}
	// 空 slice 保证为 [] 而非 null，避免前端 .map 与哈希分叉
	tree = ensureTreeEmptyCollections(tree)
	hash, err := delta.Hash(tree)
	if err != nil {
		return nil, nil, "", err
	}
	return fullDoc, tree, hash, nil
}

func marshalStateDocument(channel string, value any) (*wire.StateDocument, []byte, error) {
	switch {
	case channel == channelConfig():
		cfg, ok := value.(types.AppConfig)
		if !ok {
			if p, ok := value.(*types.AppConfig); ok && p != nil {
				cfg = *p
			} else {
				return nil, nil, errBadStateType("config")
			}
		}
		doc, raw, _, err := pbconv.MarshalStateConfig(cfg)
		return doc, raw, err
	case channel == channelLobby():
		snap, ok := value.(types.LobbySnapshot)
		if !ok {
			if p, ok := value.(*types.LobbySnapshot); ok && p != nil {
				snap = *p
			} else {
				return nil, nil, errBadStateType("lobby")
			}
		}
		doc, raw, _, err := pbconv.MarshalStateLobby(snap)
		return doc, raw, err
	default:
		snap, ok := value.(types.RoomSnapshot)
		if !ok {
			if p, ok := value.(*types.RoomSnapshot); ok && p != nil {
				snap = *p
			} else {
				return nil, nil, errBadStateType("room")
			}
		}
		doc, raw, _, err := pbconv.MarshalStateRoom(snap)
		return doc, raw, err
	}
}

type stateTypeError string

func (e stateTypeError) Error() string { return string(e) }

func errBadStateType(kind string) error {
	return stateTypeError("bad state type for " + kind)
}

func deltaOpsToWire(ops []delta.Op) ([]*wire.PatchOp, int, error) {
	out := make([]*wire.PatchOp, 0, len(ops))
	bytes := 0
	for _, op := range ops {
		wo := &wire.PatchOp{Path: op.Path, Remove: op.Remove}
		bytes += len(op.Path)
		if !op.Remove {
			var v any
			if len(op.Value) == 0 || string(op.Value) == "null" {
				v = nil
			} else if err := json.Unmarshal(op.Value, &v); err != nil {
				return nil, 0, err
			}
			// json null → nil；[] → []any{}（不会变成 null）
			sv, err := anyToStructpbValue(v)
			if err != nil {
				return nil, 0, err
			}
			wo.Value = sv
			if b, err := sv.MarshalJSON(); err == nil {
				bytes += len(b)
			} else {
				bytes += len(op.Value)
			}
		}
		out = append(out, wo)
	}
	return out, bytes, nil
}

// anyToStructpbValue：保证空数组/空对象可编码，nil 为 NullValue。
func anyToStructpbValue(v any) (*structpb.Value, error) {
	if v == nil {
		return structpb.NewNullValue(), nil
	}
	// 规范化 json.Number 等
	switch t := v.(type) {
	case []any:
		if t == nil {
			return structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{}}), nil
		}
	case map[string]any:
		if t == nil {
			return structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{}}), nil
		}
	}
	return structpb.NewValue(v)
}

func estimateTreeBytes(tree any) int {
	b, err := delta.ToJSON(tree)
	if err != nil {
		return 0
	}
	return len(b)
}

// ensureTreeEmptyCollections 递归把 nil slice 收成空数组（map 里的 null 保留语义上的删除由 Diff remove 处理）。
func ensureTreeEmptyCollections(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if val == nil {
				out[k] = nil
				continue
			}
			out[k] = ensureTreeEmptyCollections(val)
		}
		return out
	case []any:
		if t == nil {
			return []any{}
		}
		out := make([]any, len(t))
		for i := range t {
			out[i] = ensureTreeEmptyCollections(t[i])
		}
		return out
	default:
		return v
	}
}

func (s *Server) buildRawEnvelope(event string, id int64, data any, errMsg string) (*wire.Envelope, error) {
	env := &wire.Envelope{
		Event: event,
		Id:    id,
		Kind:  wire.PayloadKind_KIND_RAW,
		Err:   errMsg,
	}
	if data != nil && errMsg == "" {
		body, err := pbconv.BuildRawBody(event, data)
		if err != nil {
			return nil, err
		}
		env.RawBody = body
	}
	return env, nil
}

func channelLobby() string         { return "lobby" }
func channelRoom(id string) string { return "room:" + id }
func channelConfig() string        { return "config" }
