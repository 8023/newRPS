package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

// TestDedupGenderSeeds 覆盖防御性场景：种子 JSON 里出现同阵营重名性别选项时，
// dedupGenderSeeds 应该去重而不是让导入撞 (faction_id, label) 唯一约束回滚。
func TestDedupGenderSeeds(t *testing.T) {
	in := []types.GenderOption{
		{ID: "gender2", Label: "母狗", FactionID: "female_faction"},
		{ID: "girl", Label: "女生", FactionID: "female_faction"},
		{ID: "gender6", Label: "母狗", FactionID: "female_faction"}, // 重名重复，应被丢弃并映射回 gender2
		{ID: "gender3", Label: "母狗", FactionID: "femboy_faction"}, // 同名但不同阵营，不冲突
	}
	out, remap := dedupGenderSeeds(in)
	if len(out) != 3 {
		t.Fatalf("dedup 后应剩 3 条，got %d: %+v", len(out), out)
	}
	for _, g := range out {
		if g.ID == "gender6" {
			t.Fatalf("重复项 gender6 不应保留在去重结果里")
		}
	}
	if got := remap["gender6"]; got != "gender2" {
		t.Fatalf("remap[gender6] = %q, want gender2", got)
	}
	if _, ok := remap["gender3"]; ok {
		t.Fatalf("gender3 与 gender2 不同阵营，不应出现在 remap 里")
	}
}

// TestImportGendersFromJSONDedupsAndRemapsPlayers 端到端覆盖：种子里有重名重复项时，
// 导入必须去重后成功写入（而不是撞 UNIQUE 约束导致整个事务回滚、表继续为空），并把
// 已经引用了被丢弃 ID 的存量玩家改写为保留下来的那个 ID，不产生悬空引用。
func TestImportGendersFromJSONDedupsAndRemapsPlayers(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()

	s := &Server{db: db, genderStore: newGenderStore(db)}

	// 模拟存量玩家已经引用了将被去重丢弃的 gender6。
	playerStore := newPlayerStore(db)
	if err := playerStore.upsert(persistedPlayer{
		ID: "row1", PlayerID: "p1", Name: "老玩家", GenderID: "gender6", FactionID: "female_faction",
	}); err != nil {
		t.Fatalf("seed player: %v", err)
	}

	factions := []types.GenderFaction{
		{ID: "female_faction", Label: "顺性别女", TaskGroup: "female"},
	}
	genders := []types.GenderOption{
		{ID: "gender2", Label: "母狗", FactionID: "female_faction"},
		{ID: "gender6", Label: "母狗", FactionID: "female_faction"},
	}
	deduped, remap := dedupGenderSeeds(genders)
	if err := s.genderStore.replaceAll(factions, deduped); err != nil {
		t.Fatalf("replaceAll 在去重后不应再撞 UNIQUE 约束: %v", err)
	}
	if err := s.remapPlayerGenderIDs(remap); err != nil {
		t.Fatalf("remapPlayerGenderIDs: %v", err)
	}

	gn, err := s.genderStore.countGenders()
	if err != nil || gn != 1 {
		t.Fatalf("countGenders = %d, err=%v, want 1", gn, err)
	}

	var genderID string
	if err := db.QueryRow(`SELECT gender_id FROM players WHERE id = 'row1'`).Scan(&genderID); err != nil {
		t.Fatalf("query player: %v", err)
	}
	if genderID != "gender2" {
		t.Fatalf("玩家的 gender_id 应被回填为保留下来的 gender2, got %q", genderID)
	}
}
