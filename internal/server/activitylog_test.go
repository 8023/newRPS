package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/doumiao/newRPS/internal/config"
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

func TestWriteActivityLogWritesLogFile(t *testing.T) {
	tmp := t.TempDir()
	// activityLogBaseDir 读 config.GetRootDir()/work/logs；临时劫持 root。
	prev := config.GetRootDir()
	config.SetRootDirForTest(tmp)
	t.Cleanup(func() { config.SetRootDirForTest(prev) })

	writeActivityLog("chat", []string{"scope", "text"}, []string{"lobby", "hello"})

	dir := filepath.Join(tmp, "work", "logs", activityLogWeekDir(time.Now()))
	path := filepath.Join(dir, "chat.log")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected chat.log written: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "scope=lobby") || !strings.Contains(body, "text=hello") {
		t.Fatalf("log body missing fields: %q", body)
	}
	if strings.HasSuffix(path, ".csv") {
		t.Fatal("should write .log not .csv")
	}
}
