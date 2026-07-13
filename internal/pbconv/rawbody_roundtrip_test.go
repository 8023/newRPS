package pbconv

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/doumiao/newRPS/internal/wire"
	"google.golang.org/protobuf/proto"
)

func TestRawBodyFromJSEncodedFile(t *testing.T) {
	b, err := os.ReadFile("/tmp/punish.bin")
	if err != nil {
		t.Skip(err)
	}
	var wenv wire.Envelope
	if err := proto.Unmarshal(b, &wenv); err != nil {
		t.Fatal(err)
	}
	if wenv.Event != "punishment:submit" {
		t.Fatalf("event %q", wenv.Event)
	}
	payload, err := RawBodyToFront(wenv.RawBody)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(payload)
	var p struct {
		Text     string `json:"text"`
		ImageURL string `json:"imageUrl"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	if p.Text != "完成了" {
		t.Fatalf("text=%q payload=%v", p.Text, payload)
	}
}
