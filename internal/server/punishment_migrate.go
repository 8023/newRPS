package server

import (
	"encoding/json"
	"fmt"
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

// migratePunishmentPoolFromJSONIfNeeded 一次性把 punishments.json 里的 tasks/seriesTasks
// 导入 SQLite。幂等条件：punishment_tasks 表行数为 0 且磁盘 tasks（或 seriesTasks）非空。
// 迁移完不清理源头 JSON（字段变成死数据，config 加载器已不再读写）。
func (s *Server) migratePunishmentPoolFromJSONIfNeeded() {
	if s.punishmentStore == nil {
		return
	}
	n, err := s.punishmentStore.countTasks()
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
			// 不能先导入 tasks：幂等哨兵正是 punishment_tasks 非空；一旦先写成功，
			// 后续重启会跳过迁移，修复 JSON 后系列也永远无法自动补回。
			return
		}
	}
	if len(tasks) == 0 && len(legacySeries) == 0 {
		return
	}

	// 系列变体打散为任务池条目，挂 series:<id> 内部标签。
	extraTasks := make([]types.PunishmentTaskConfig, 0)
	seriesOut := make([]types.PunishmentSeriesTaskConfig, 0, len(legacySeries))
	for _, ser := range legacySeries {
		sid := strings.TrimSpace(ser.ID)
		if sid == "" {
			continue
		}
		steps := make([]types.PunishmentSeriesStep, 0, len(ser.Subtasks))
		for si, sub := range ser.Subtasks {
			taskIDs := make([]string, 0, len(sub.Variants))
			for vi, v := range sub.Variants {
				tid := fmt.Sprintf("%s_s%d_v%d", sid, si, vi)
				op := v.BackgroundOpacity
				if op <= 0 {
					op = 0.22
				}
				extraTasks = append(extraTasks, types.PunishmentTaskConfig{
					ID:                tid,
					Name:              fmt.Sprintf("%s · 第%d步变体%d", strings.TrimSpace(ser.Name), si+1, vi+1),
					Text:              strings.TrimSpace(v.Text),
					TagIDs:            []string{"series:" + sid},
					FactionIDs:        append([]string(nil), v.FactionIDs...),
					Order:             50,
					BackgroundImages:  append([]string(nil), v.BackgroundImages...),
					BackgroundOpacity: op,
				})
				taskIDs = append(taskIDs, tid)
			}
			steps = append(steps, types.PunishmentSeriesStep{TaskIDs: taskIDs})
		}
		seriesOut = append(seriesOut, types.PunishmentSeriesTaskConfig{
			ID:                   sid,
			Name:                 strings.TrimSpace(ser.Name),
			RoomNamePool:         ser.RoomNamePool,
			RoomBackgroundImages: append([]string(nil), ser.RoomBackgroundImages...),
			Steps:                steps,
		})
	}

	allTasks := make([]types.PunishmentTaskConfig, 0, len(tasks)+len(extraTasks))
	allTasks = append(allTasks, tasks...)
	allTasks = append(allTasks, extraTasks...)

	// 先写系列、最后写作为幂等哨兵的 tasks。若系列写入成功而 tasks 失败，下一次
	// 启动仍会因 tasks 为空而完整重试；反过来会造成系列永久漏迁。
	if len(seriesOut) > 0 {
		if err := s.punishmentStore.replaceSeries(seriesOut); err != nil {
			s.errorLog("punishment_migrate_series_failed", err.Error())
			return
		}
	}
	if len(allTasks) > 0 {
		if err := s.punishmentStore.replaceTasks(allTasks); err != nil {
			s.errorLog("punishment_migrate_tasks_failed", err.Error())
			return
		}
	}
}
