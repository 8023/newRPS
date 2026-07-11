package delta

import (
	"encoding/json"
	"testing"
)

func TestDiffApplyHash(t *testing.T) {
	from := map[string]any{
		"onlineCount": float64(2),
		"players": []any{
			map[string]any{"id": "a", "connected": true, "name": "A"},
			map[string]any{"id": "b", "connected": true, "name": "B"},
		},
	}
	to := map[string]any{
		"onlineCount": float64(1),
		"players": []any{
			map[string]any{"id": "a", "connected": false, "name": "A"},
			map[string]any{"id": "b", "connected": true, "name": "B"},
		},
	}
	ops, err := Diff(from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) == 0 {
		t.Fatal("expected ops")
	}
	// 不应整包替换
	for _, op := range ops {
		if op.Path == "" {
			t.Fatalf("unexpected root replace: %+v", ops)
		}
	}
	got, err := Apply(from, ops)
	if err != nil {
		t.Fatal(err)
	}
	h1, _ := Hash(to)
	h2, _ := Hash(got)
	if h1 != h2 {
		b1, _ := json.Marshal(to)
		b2, _ := json.Marshal(got)
		t.Fatalf("hash mismatch\nwant %s\ngot  %s\nops=%v\nA=%s\nB=%s", h1, h2, ops, b1, b2)
	}
}

func TestHashStable(t *testing.T) {
	a := map[string]any{"z": float64(1), "a": float64(2)}
	b := map[string]any{"a": float64(2), "z": float64(1)}
	ha, _ := Hash(a)
	hb, _ := Hash(b)
	if ha != hb {
		t.Fatalf("%s != %s", ha, hb)
	}
}

func TestHashNoHTMLEscape(t *testing.T) {
	doc := map[string]any{"name": "A&B<C>"}
	raw, err := ToJSON(doc)
	if err != nil {
		t.Fatal(err)
	}
	if bytesContain(raw, []byte(`\u0026`)) {
		t.Fatalf("unexpected HTML escape in %s", raw)
	}
	// 往返后哈希不变
	parsed, err := FromJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	h1, _ := Hash(doc)
	h2, _ := Hash(parsed)
	if h1 != h2 {
		t.Fatalf("hash roundtrip %s vs %s", h1, h2)
	}
}

func TestDiffMapRemove(t *testing.T) {
	from := map[string]any{"players": map[string]any{"a": map[string]any{"n": "1"}, "b": map[string]any{"n": "2"}}}
	to := map[string]any{"players": map[string]any{"a": map[string]any{"n": "1"}}}
	ops, err := Diff(from, to)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Apply(from, ops)
	if err != nil {
		t.Fatal(err)
	}
	h1, _ := Hash(to)
	h2, _ := Hash(got)
	if h1 != h2 {
		t.Fatalf("after remove %s != %s ops=%v", h1, h2, ops)
	}
}

func bytesContain(b, sub []byte) bool {
	return len(b) >= len(sub) && (string(b) == string(sub) || len(sub) == 0 ||
		(len(b) > 0 && containsBytes(b, sub)))
}

func containsBytes(b, sub []byte) bool {
	for i := 0; i+len(sub) <= len(b); i++ {
		ok := true
		for j := 0; j < len(sub); j++ {
			if b[i+j] != sub[j] {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}
