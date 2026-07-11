package server

import (
	"encoding/json"

	"github.com/doumiao/newRPS/internal/delta"
	"github.com/doumiao/newRPS/internal/wire"
)

// 同步通道：服务端保留上一份全量树，广播时发 DELTA + 合并后哈希。

type syncChannel struct {
	name string
	seq  uint64
	doc  any // last full document tree
	hash string
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

// buildStateEnvelope 生成 FULL 或 DELTA 的 protobuf 信封。
func (s *Server) buildStateEnvelope(event, channel string, value any) (*wire.Envelope, int, error) {
	doc, err := delta.Snapshot(value)
	if err != nil {
		return nil, 0, err
	}
	hash, err := delta.Hash(doc)
	if err != nil {
		return nil, 0, err
	}
	fullJSON, err := delta.ToJSON(doc)
	if err != nil {
		return nil, 0, err
	}

	ch := s.getSync(channel)
	ch.seq++
	env := &wire.Envelope{
		Event:   event,
		Channel: channel,
		Seq:     ch.seq,
		Hash:    hash,
	}

	// 无历史或强制全量
	if ch.doc == nil {
		env.Kind = wire.PayloadKind_KIND_FULL
		env.Full = fullJSON
		ch.doc = doc
		ch.hash = hash
		return env, len(fullJSON), nil
	}

	ops, err := delta.Diff(ch.doc, doc)
	if err != nil {
		return nil, 0, err
	}
	// 无变化：不广播（避免空包刷屏）；仍刷新本地 doc/hash
	if len(ops) == 0 {
		ch.doc = doc
		ch.hash = hash
		return nil, 0, nil
	}
	if len(ops) > 80 {
		env.Kind = wire.PayloadKind_KIND_FULL
		env.Full = fullJSON
		ch.doc = doc
		ch.hash = hash
		return env, len(fullJSON), nil
	}

	wireOps := make([]*wire.PatchOp, 0, len(ops))
	opsBytes := 0
	for _, op := range ops {
		wo := &wire.PatchOp{Path: op.Path, Remove: op.Remove, Value: op.Value}
		wireOps = append(wireOps, wo)
		opsBytes += len(op.Path) + len(op.Value)
	}
	if opsBytes > len(fullJSON)*4/5 {
		env.Kind = wire.PayloadKind_KIND_FULL
		env.Full = fullJSON
		ch.doc = doc
		ch.hash = hash
		return env, len(fullJSON), nil
	}

	env.Kind = wire.PayloadKind_KIND_DELTA
	env.Ops = wireOps
	ch.doc = doc
	ch.hash = hash
	return env, opsBytes, nil
}

// buildFullEnvelope 强制全量（客户端 resync 或首包）。
func (s *Server) buildFullEnvelope(event, channel string, value any) (*wire.Envelope, int, error) {
	doc, err := delta.Snapshot(value)
	if err != nil {
		return nil, 0, err
	}
	hash, err := delta.Hash(doc)
	if err != nil {
		return nil, 0, err
	}
	fullJSON, err := delta.ToJSON(doc)
	if err != nil {
		return nil, 0, err
	}
	ch := s.getSync(channel)
	ch.seq++
	ch.doc = doc
	ch.hash = hash
	return &wire.Envelope{
		Event:   event,
		Kind:    wire.PayloadKind_KIND_FULL,
		Channel: channel,
		Seq:     ch.seq,
		Hash:    hash,
		Full:    fullJSON,
	}, len(fullJSON), nil
}

func (s *Server) buildRawEnvelope(event string, id int64, data any, errMsg string) (*wire.Envelope, error) {
	env := &wire.Envelope{
		Event: event,
		Id:    id,
		Kind:  wire.PayloadKind_KIND_RAW,
		Err:   errMsg,
	}
	if data != nil && errMsg == "" {
		b, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		env.Raw = b
	}
	return env, nil
}

func channelLobby() string { return "lobby" }
func channelRoom(id string) string { return "room:" + id }
func channelPlayers() string { return "players" }
func channelConfig() string { return "config" }
