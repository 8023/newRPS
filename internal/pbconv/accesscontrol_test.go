package pbconv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/doumiao/newRPS/internal/config"
)

func TestAccessControlFrontFieldName(t *testing.T) {
	// LoadConfig 需要从仓库根找到 config/json/site.json
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
	cfg.AccessControl.MaxOnlinePerIP = 3
	cfg.AccessControl.MaxCreatesPer10Min = 7
	pb, err := ConfigToProto(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if pb.GetAccessControl() == nil {
		t.Fatal("nil AccessControl")
	}
	if pb.AccessControl.MaxCreatesPer_10Min != 7 {
		t.Fatalf("proto creates=%d", pb.AccessControl.MaxCreatesPer_10Min)
	}
	front, err := ConfigProtoToFront(pb)
	if err != nil {
		t.Fatal(err)
	}
	ac, _ := front["accessControl"].(map[string]any)
	if ac == nil {
		t.Fatalf("no accessControl: %#v", front)
	}
	// server 侧 front tree 用 protojson → maxCreatesPer10Min
	if ac["maxCreatesPer10Min"] == nil {
		t.Fatalf("front accessControl keys=%#v", ac)
	}
}
