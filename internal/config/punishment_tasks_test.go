package config

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

// TestNormalizePunishmentTasksNegativeOrderBecomesNegativeOne：-1 是"不参与随机抽取，
// 仅供系列任务按 ID 引用"的唯一合法负数标记；任何其他负数都会被归一化为 -1（不保留具体
// 负值，避免被误解为有额外含义）。0 及正数才是真实难度，仍按原逻辑夹紧到 1-99。
func TestNormalizePunishmentTasksNegativeOrderBecomesNegativeOne(t *testing.T) {
	tasks := []types.PunishmentTaskConfig{
		{ID: "a", Text: "文案 a", Order: -1},
		{ID: "b", Text: "文案 b", Order: -99},
		{ID: "c", Text: "文案 c", Order: 0},
		{ID: "d", Text: "文案 d", Order: 150},
		{ID: "e", Text: "文案 e", Order: 50},
	}
	got := NormalizePunishmentTasks(tasks)
	want := map[string]int{"a": -1, "b": -1, "c": 1, "d": 99, "e": 50}
	if len(got) != len(want) {
		t.Fatalf("got %d tasks want %d", len(got), len(want))
	}
	for _, task := range got {
		if wantOrder, ok := want[task.ID]; !ok || task.Order != wantOrder {
			t.Fatalf("task %s order=%d want %d", task.ID, task.Order, want[task.ID])
		}
	}
}
