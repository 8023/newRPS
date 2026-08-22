package server

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/doumiao/newRPS/internal/config"
	"github.com/doumiao/newRPS/internal/types"
)

// legacySeriesDisk 承接旧 punishments.json 里 seriesTasks 的 subtasks/variants 结构。
type legacySeriesDisk struct {
	ID                   string              `json:"id"`
	Name                 string              `json:"name"`
	RoomNamePool         *types.RoomNamePool `json:"roomNamePool"`
	RoomBackgroundImages []string            `json:"roomBackgroundImages"`
	TargetFactionIDs     []string            `json:"targetFactionIds"`
	Subtasks             []legacySubtaskDisk `json:"subtasks"`
}

type legacySubtaskDisk struct {
	Variants []legacyVariantDisk `json:"variants"`
}

type legacyVariantDisk struct {
	FactionIDs        []string `json:"factionIds"`
	Text              string   `json:"text"`
	BackgroundImages  []string `json:"backgroundImages"`
	BackgroundOpacity float64  `json:"backgroundOpacity"`
}

// migratePunishmentPoolFromJSONIfNeeded 一次性把 punishments.json 里已废弃的 tasks/
// seriesTasks 字段导入 SQLite（sub_tasks/series，均标记 status=approved，贡献者归属为
// legacyPunishmentContributorID，与 schema_migrations.go v43 迁移存量数据时的约定一致）。
// 幂等条件：sub_tasks 表行数为 0 且磁盘 tasks（或 seriesTasks）非空——只有全新安装、且
// punishments.json 里仍残留这两个死字段时才会走到这里；线上库早就迁移过，不会重复导入。
func (s *Server) migratePunishmentPoolFromJSONIfNeeded() {
	if s.contributionStore == nil {
		return
	}
	n, err := s.contributionStore.tasks.countAll()
	if err != nil {
		s.errorLog("punishment_migrate_count_failed", err.Error())
		return
	}
	if n > 0 {
		return
	}
	tasks, seriesRaw, err := config.ReadLegacyPunishmentPoolFromDisk()
	if err != nil {
		s.errorLog("punishment_migrate_read_failed", err.Error())
		return
	}
	var legacySeries []legacySeriesDisk
	if len(seriesRaw) > 0 && string(seriesRaw) != "null" {
		if err := json.Unmarshal(seriesRaw, &legacySeries); err != nil {
			s.errorLog("punishment_migrate_series_parse_failed", err.Error())
			// 不能先导入 tasks：幂等哨兵正是 sub_tasks 非空；一旦先写成功，
			// 后续重启会跳过迁移，修复 JSON 后系列也永远无法自动补回。
			return
		}
	}
	if len(tasks) == 0 && len(legacySeries) == 0 {
		return
	}

	tx, err := s.db.Begin()
	if err != nil {
		s.errorLog("punishment_migrate_begin_failed", err.Error())
		return
	}
	if err := seedLegacyPunishmentTx(tx, s.contributionStore, tasks, legacySeries); err != nil {
		_ = tx.Rollback()
		s.errorLog("punishment_migrate_seed_failed", err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		s.errorLog("punishment_migrate_commit_failed", err.Error())
	}
}

// seedLegacyPunishmentTx 把独立随机任务（每条本身就是一个只含单一变体的 StepDraft）与
// 系列（每个 subtask 是一步，其 variants 直接搬进该步的 Variants）写入 sub_tasks/series。
func seedLegacyPunishmentTx(tx *sql.Tx, cs *contributionStore, tasks []types.PunishmentTaskConfig, series []legacySeriesDisk) error {
	for _, t := range tasks {
		draft := types.StepDraft{
			Variants:          []types.TaskVariantDraft{{Text: strings.TrimSpace(t.Text), FactionIDs: t.FactionIDs}},
			InRandomPool:      t.Order != -1,
			Order:             t.Order,
			TagIDs:            t.TagIDs,
			BackgroundImage:   t.BackgroundImage,
			BackgroundOpacity: fallbackOpacity(t.BackgroundOpacity),
		}
		if _, err := cs.tasks.insertVersion(tx, strings.TrimSpace(t.ID), "", 0, draft, types.ContributionApproved,
			legacyPunishmentContributorID, false, legacyPunishmentContributorID, ""); err != nil {
			return err
		}
	}

	for _, ser := range series {
		sid := strings.TrimSpace(ser.ID)
		if sid == "" {
			continue
		}
		seriesRow, err := cs.series.insertVersion(tx, sid, strings.TrimSpace(ser.Name), ser.TargetFactionIDs, types.ContributionApproved,
			legacyPunishmentContributorID, false, legacyPunishmentContributorID, "")
		if err != nil {
			return err
		}
		pool := any(seriesRow.RoomNamePool)
		if ser.RoomNamePool != nil {
			pool = ser.RoomNamePool
		}
		poolJSON, err := json.Marshal(pool)
		if err != nil {
			return err
		}
		bgJSON, err := json.Marshal(nonNilStrings(ser.RoomBackgroundImages))
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE series SET room_name_pool=?, room_background_images=? WHERE id=? AND version=?`,
			string(poolJSON), string(bgJSON), seriesRow.ID, seriesRow.Version); err != nil {
			return err
		}
		for si, sub := range ser.Subtasks {
			variants := make([]types.TaskVariantDraft, 0, len(sub.Variants))
			bgImage := ""
			op := 0.22
			for vi, v := range sub.Variants {
				variants = append(variants, types.TaskVariantDraft{
					Text: strings.TrimSpace(v.Text), FactionIDs: v.FactionIDs,
				})
				if vi == 0 {
					if len(v.BackgroundImages) > 0 {
						bgImage = v.BackgroundImages[0]
					}
					op = fallbackOpacity(v.BackgroundOpacity)
				}
			}
			draft := types.StepDraft{
				Variants: variants, InRandomPool: true, Order: 50,
				BackgroundImage: bgImage, BackgroundOpacity: op,
			}
			if _, err := cs.tasks.insertVersion(tx, "", seriesRow.ID, si, draft, types.ContributionApproved,
				legacyPunishmentContributorID, false, legacyPunishmentContributorID, ""); err != nil {
				return err
			}
		}
	}
	return nil
}

func fallbackOpacity(op float64) float64 {
	if op <= 0 {
		return 0.22
	}
	return op
}
