package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

// TestMetaForFormalTaskAttributesTaskIDDirectly 确认投票目标恒定落在「这一条具体任务」
// 自己的 (ID, Version) 上——独立随机任务、系列每一步各自的 ID 都是 sub_tasks 的行 ID，
// 不再需要 task_group_id 那层间接。贡献者是谁、系列归属、要不要匿名都不再存进这份
// meta——需要时由 contribution_vote_rpc.go 按 FormalTaskID/FormalTaskVersion 现查
// sub_tasks（见 metaForFormalTask 的注释）。
func TestMetaForFormalTaskAttributesTaskIDDirectly(t *testing.T) {
	s := newTestServer(t)

	task := types.PunishmentTaskConfig{ID: "st_1", Version: 2}
	meta := s.metaForFormalTask(task)
	if meta.FormalTaskID != "st_1" || meta.FormalTaskVersion != 2 {
		t.Fatalf("should attribute to the task's own id/version, got %+v", meta)
	}

	step := types.PunishmentTaskConfig{ID: "st_2a", Version: 1, SeriesID: "sr_2", StepIndex: 0}
	stepMeta := s.metaForFormalTask(step)
	if stepMeta.FormalTaskID != "st_2a" || stepMeta.FormalTaskVersion != 1 {
		t.Fatalf("series steps must vote on their own id, got %+v", stepMeta)
	}
}
