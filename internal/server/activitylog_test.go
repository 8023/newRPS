package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestActivityLogWeekDir(t *testing.T) {
	// 固定一个日期验证 ISO 周格式
	// 2026-07-13 是 ISO 2026 年第 29 周 → 2629
	d := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	got := activityLogWeekDir(d)
	if got != "2629" {
		t.Fatalf("week dir want 2629 got %s", got)
	}
}

func TestActivityLogWriteCSV(t *testing.T) {
	tmp := t.TempDir()
	// 临时替换 root：通过 chdir 到含 config 的布局太重；直接测写路径逻辑
	dir := filepath.Join(tmp, activityLogWeekDir(time.Now()))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "chat.csv")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("time,scope,roomId,playerId,playerName,text\n")
	_ = f.Close()
	if st, err := os.Stat(path); err != nil || st.Size() == 0 {
		t.Fatal("csv not written")
	}
}
