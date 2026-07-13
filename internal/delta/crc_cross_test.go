package delta

import (
	"fmt"
	"testing"
)

func TestHashSampleTrees(t *testing.T) {
	cases := []any{
		map[string]any{},
		map[string]any{"ok": true},
		map[string]any{"phase": "punishment", "proofs": []any{}, "punishedPlayerIds": []any{"abc"}},
		map[string]any{
			"proofs": []any{
				map[string]any{"playerId": "p1", "text": "done", "status": "pending", "submittedAt": float64(1)},
			},
		},
		map[string]any{"legalMoves": []any{map[string]any{"row": float64(0), "col": float64(3)}}},
		map[string]any{"name": "A&B"},
	}
	for i, c := range cases {
		h, err := Hash(c)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := ToJSON(c)
		fmt.Printf("case %d hash=%s json=%s\n", i, h, raw)
	}
}
