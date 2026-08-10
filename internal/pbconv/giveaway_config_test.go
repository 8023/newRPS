package pbconv

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/doumiao/newRPS/internal/config"
)

// TestGiveawayConfigNumericFieldsRoundTrip 防止 GiveawayConfig 数值字段只写在
// Go/TS 类型、未进 game.proto 时再次被 ConfigToProto 静默丢掉（后台保存看似成功、
// config:update / 大厅同步却永远回落到默认值）。
func TestGiveawayConfigNumericFieldsRoundTrip(t *testing.T) {
	root, _ := os.Getwd()
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(root, "config", "json", "site.json")); err == nil {
			break
		}
		root = filepath.Dir(root)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Skip(err)
	}
	cfg.Giveaway.ActiveBoostValue = 4.5
	cfg.Giveaway.WinPenaltyValue = 2.5
	cfg.Giveaway.LikeVoteLimitPerHour = 17
	cfg.Giveaway.LikeVoteValue = 3.5
	cfg.Giveaway.DislikeVoteLimitPerHour = 23
	cfg.Giveaway.DislikeVoteValue = 0.7
	cfg.Giveaway.PetLikeVoteLimitPerHour = 31
	cfg.Giveaway.PetLikeVoteValue = 1.1
	cfg.Giveaway.PetDislikeVoteLimitPerHour = 37
	cfg.Giveaway.PetDislikeVoteValue = 0.11
	cfg.Giveaway.MasterLikeVoteLimitPerHour = 41
	cfg.Giveaway.MasterLikeVoteValue = 1.2
	cfg.Giveaway.MasterDislikeVoteLimitPerHour = 43
	cfg.Giveaway.MasterDislikeVoteValue = 0.12

	pb, err := ConfigToProto(cfg)
	if err != nil {
		t.Fatal(err)
	}
	g := pb.GetGiveaway()
	if g == nil {
		t.Fatal("nil Giveaway")
	}
	if g.ActiveBoostValue != 4.5 || g.WinPenaltyValue != 2.5 {
		t.Fatalf("boost/penalty: got %v / %v", g.ActiveBoostValue, g.WinPenaltyValue)
	}
	if g.LikeVoteLimitPerHour != 17 || g.LikeVoteValue != 3.5 {
		t.Fatalf("like: got limit=%d value=%v", g.LikeVoteLimitPerHour, g.LikeVoteValue)
	}
	if g.DislikeVoteLimitPerHour != 23 || g.DislikeVoteValue != 0.7 {
		t.Fatalf("dislike: got limit=%d value=%v", g.DislikeVoteLimitPerHour, g.DislikeVoteValue)
	}
	if g.PetLikeVoteLimitPerHour != 31 || g.PetLikeVoteValue != 1.1 || g.PetDislikeVoteLimitPerHour != 37 || g.PetDislikeVoteValue != 0.11 {
		t.Fatalf("pet: got likeLimit=%d likeValue=%v dislikeLimit=%d dislikeValue=%v", g.PetLikeVoteLimitPerHour, g.PetLikeVoteValue, g.PetDislikeVoteLimitPerHour, g.PetDislikeVoteValue)
	}
	if g.MasterLikeVoteLimitPerHour != 41 || g.MasterLikeVoteValue != 1.2 || g.MasterDislikeVoteLimitPerHour != 43 || g.MasterDislikeVoteValue != 0.12 {
		t.Fatalf("master: got likeLimit=%d likeValue=%v dislikeLimit=%d dislikeValue=%v", g.MasterLikeVoteLimitPerHour, g.MasterLikeVoteValue, g.MasterDislikeVoteLimitPerHour, g.MasterDislikeVoteValue)
	}

	front, err := ConfigProtoToFront(pb)
	if err != nil {
		t.Fatal(err)
	}
	fg, _ := front["giveaway"].(map[string]any)
	if fg == nil {
		t.Fatalf("no giveaway in front: %#v", front)
	}
	if int(asFloat(fg["likeVoteLimitPerHour"])) != 17 {
		t.Fatalf("front likeVoteLimitPerHour=%#v", fg["likeVoteLimitPerHour"])
	}
	if int(asFloat(fg["petLikeVoteLimitPerHour"])) != 31 || int(asFloat(fg["masterDislikeVoteLimitPerHour"])) != 43 {
		t.Fatalf("front pet/master limits=%#v / %#v", fg["petLikeVoteLimitPerHour"], fg["masterDislikeVoteLimitPerHour"])
	}
	if asFloat(fg["petLikeVoteValue"]) != 1.1 || asFloat(fg["masterDislikeVoteValue"]) != 0.12 {
		t.Fatalf("front pet/master values=%#v / %#v", fg["petLikeVoteValue"], fg["masterDislikeVoteValue"])
	}
	if asFloat(fg["dislikeVoteValue"]) != 0.7 {
		t.Fatalf("front dislikeVoteValue=%#v", fg["dislikeVoteValue"])
	}
	if asFloat(fg["activeBoostValue"]) != 4.5 {
		t.Fatalf("front activeBoostValue=%#v", fg["activeBoostValue"])
	}
}

func asFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	case string:
		// protojson 默认把 int64 编成字符串
		var f float64
		_, _ = fmt.Sscanf(n, "%f", &f)
		return f
	default:
		return 0
	}
}
