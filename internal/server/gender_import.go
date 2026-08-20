package server

import (
	"fmt"

	"github.com/doumiao/newRPS/internal/config"
	"github.com/doumiao/newRPS/internal/types"
)

func (s *Server) importGendersFromJSONIfNeeded() error {
	if s.genderStore == nil {
		return nil
	}
	fn, err := s.genderStore.countFactions()
	if err != nil {
		return err
	}
	gn, err := s.genderStore.countGenders()
	if err != nil {
		return err
	}
	if fn > 0 || gn > 0 {
		return nil
	}
	factions, genders, err := config.ReadGenderSeedsFromDisk()
	if err != nil {
		return err
	}
	if len(factions) == 0 && len(genders) == 0 {
		return nil
	}
	// 旧版 genders.json 在引入 gender_options 的 (faction_id, normalized_label) 唯一约束
	// 之前，没有任何机制阻止后台在同一阵营内新增重名性别选项（生产环境曾出现"母狗"在
	// female_faction / femboy_faction 下各自重复一次）。原样导入会在 INSERT 时撞
	// UNIQUE 约束，整个事务回滚——表继续保持空，随后启动校验只会报出与真实原因无关的
	// "至少需要一个性别选项"。这里按 (faction_id, 规范化 label) 去重（保留 JSON 中先
	// 出现的那条），并把被丢弃 ID 映射到保留 ID，供导入成功后回填 players.gender_id，
	// 避免存量玩家的性别选择变成悬空引用。
	genders, remap := dedupGenderSeeds(genders)
	if err := s.genderStore.replaceAll(factions, genders); err != nil {
		return err
	}
	if len(remap) > 0 {
		if err := s.remapPlayerGenderIDs(remap); err != nil {
			return fmt.Errorf("回填重复性别选项的玩家引用失败: %w", err)
		}
	}
	return nil
}

// dedupGenderSeeds 按 (FactionID, 规范化 Label) 去重，保留每组中最先出现的一条；
// 返回值第二项是"被丢弃 ID → 保留 ID"的映射，供调用方回填历史引用。
func dedupGenderSeeds(genders []types.GenderOption) ([]types.GenderOption, map[string]string) {
	type key struct{ faction, label string }
	seen := make(map[key]string, len(genders))
	remap := make(map[string]string)
	out := make([]types.GenderOption, 0, len(genders))
	for _, g := range genders {
		k := key{g.FactionID, normalizeGenderLabel(g.Label)}
		if canonicalID, ok := seen[k]; ok {
			if canonicalID != g.ID {
				remap[g.ID] = canonicalID
			}
			continue
		}
		seen[k] = g.ID
		out = append(out, g)
	}
	return out, remap
}

// remapPlayerGenderIDs 把 players.gender_id 中命中 remap 键的存量引用改写为对应的保留
// ID，用于 dedupGenderSeeds 丢弃重复性别选项后避免产生悬空引用。
func (s *Server) remapPlayerGenderIDs(remap map[string]string) error {
	if s.db == nil || len(remap) == 0 {
		return nil
	}
	stmt, err := s.db.Prepare(`UPDATE players SET gender_id = ? WHERE gender_id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for oldID, newID := range remap {
		if _, err := stmt.Exec(newID, oldID); err != nil {
			return fmt.Errorf("remap gender_id %s -> %s: %w", oldID, newID, err)
		}
	}
	return nil
}

func (s *Server) reloadGenderCaches() {
	if s.genderStore == nil {
		return
	}
	if err := s.loadGendersIntoConfig(&s.cfg); err != nil {
		s.errorLog("gender_cache_load_failed", err.Error())
	}
}

func (s *Server) loadGendersIntoConfig(cfg *types.AppConfig) error {
	if s.genderStore == nil {
		return nil
	}
	factions, err := s.genderStore.listFactions()
	if err != nil {
		return err
	}
	genders, err := s.genderStore.listGenders()
	if err != nil {
		return err
	}
	cfg.GenderFactions = factions
	cfg.Genders = genders
	return nil
}
