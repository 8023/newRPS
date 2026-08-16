package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

// 对照 plan 第 3 节：a/b/c 三任务标签过滤边界。
func TestCandidateTasksForTags(t *testing.T) {
	tasks := []types.PunishmentTaskConfig{
		{ID: "a", TagIDs: []string{"truth"}},
		{ID: "b", TagIDs: []string{"truth", "challenge"}},
		{ID: "c", TagIDs: []string{"challenge", "sing"}},
		{ID: "d", TagIDs: nil}, // 未勾选任何标签：无视标签筛选，任何过滤条件下都应出现
	}
	ids := func(list []types.PunishmentTaskConfig) []string {
		out := make([]string, len(list))
		for i, t := range list {
			out[i] = t.ID
		}
		return out
	}
	// S={truth} R={} → a,b,d
	got := candidateTasksForTags(tasks, []string{"truth"}, nil)
	if want := []string{"a", "b", "d"}; !sameStringSlice(ids(got), want) {
		t.Fatalf("include truth: got %v want %v", ids(got), want)
	}
	// S={truth,challenge} R={} → b,d
	got = candidateTasksForTags(tasks, []string{"truth", "challenge"}, nil)
	if want := []string{"b", "d"}; !sameStringSlice(ids(got), want) {
		t.Fatalf("include truth+challenge: got %v want %v", ids(got), want)
	}
	// S={} R={sing} → a,b,d
	got = candidateTasksForTags(tasks, nil, []string{"sing"})
	if want := []string{"a", "b", "d"}; !sameStringSlice(ids(got), want) {
		t.Fatalf("exclude sing: got %v want %v", ids(got), want)
	}
	// S={challenge} R={sing} → b,d（c 有 sing，b 无 sing 有 challenge，d 无标签直接入选）
	got = candidateTasksForTags(tasks, []string{"challenge"}, []string{"sing"})
	if want := []string{"b", "d"}; !sameStringSlice(ids(got), want) {
		t.Fatalf("include challenge exclude sing: got %v want %v", ids(got), want)
	}
	// S={} R={} → 全池
	got = candidateTasksForTags(tasks, nil, nil)
	if len(got) != 4 {
		t.Fatalf("empty filters should keep all, got %v", ids(got))
	}
}

// 对照 plan：难度为负数的任务应被随机候选池排除，正数/零不受影响
// （零在保存前已被 config.NormalizePunishmentTasks 夹紧到 1，这里直接构造已落库的值验证过滤本身）。
func TestCandidateTasksForRandomDifficulty(t *testing.T) {
	tasks := []types.PunishmentTaskConfig{
		{ID: "positive", Order: 50},
		{ID: "negative", Order: -1},
		{ID: "very_negative", Order: -99},
		{ID: "zero", Order: 0},
	}
	got := candidateTasksForRandomDifficulty(tasks)
	ids := make([]string, len(got))
	for i, t := range got {
		ids[i] = t.ID
	}
	if want := []string{"positive", "zero"}; !sameStringSlice(ids, want) {
		t.Fatalf("got %v want %v", ids, want)
	}
}

func sameStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
