package pbconv

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

// 自定义性别 genderId 为空串：typed wire 路径会因 EmitUnpopulated:false 丢掉 key，
// 动态 Struct 路径则保留。两端都要能正确表达「自定义」。
func TestEmptyGenderIdSurvivesWirePaths(t *testing.T) {
	p := types.PublicPlayer{
		ID: "x", Name: "n", GenderID: "", GenderLabel: "狼娘",
		FactionID: "male_faction", FactionLabel: "顺男",
	}

	// dynamic（player:join / updateProfile 应答）
	body, err := BuildRawBody("", map[string]any{"player": p, "token": "t"})
	if err != nil {
		t.Fatal(err)
	}
	front, err := RawBodyToFront(body)
	if err != nil {
		t.Fatal(err)
	}
	player := front.(map[string]any)["player"].(map[string]any)
	gid, has := player["genderId"]
	if !has {
		t.Fatal("dynamic path dropped empty genderId key")
	}
	if gid != "" {
		t.Fatalf("dynamic genderId want empty string, got %#v", gid)
	}
	if player["genderLabel"] != "狼娘" {
		t.Fatalf("genderLabel=%#v", player["genderLabel"])
	}

	// typed player:update：key 会被省略，fillPlayerDefaults 必须补回 ""
	body2, err := BuildRawBody("player:update", p)
	if err != nil {
		t.Fatal(err)
	}
	front2, err := RawBodyToFront(body2)
	if err != nil {
		t.Fatal(err)
	}
	p2 := front2.(map[string]any)
	filled := fillPlayerDefaults(p2).(map[string]any)
	if filled["genderId"] != "" {
		t.Fatalf("after fillPlayerDefaults genderId want \"\", got %#v", filled["genderId"])
	}
	if filled["genderLabel"] != "狼娘" {
		t.Fatalf("genderLabel=%#v", filled["genderLabel"])
	}
}

func TestFillPlayerDefaults_EmptyGenderId(t *testing.T) {
	// 模拟 typed proto 省略空 genderId 后的前端/哈希树形态
	in := map[string]any{
		"id":          "p1",
		"genderLabel": "狼娘",
		"factionId":   "male_faction",
	}
	out, ok := fillPlayerDefaults(in).(map[string]any)
	if !ok {
		t.Fatal("not map")
	}
	if out["genderId"] != "" {
		t.Fatalf("genderId want empty string when missing, got %#v", out["genderId"])
	}
	// 已有非空 genderId 不得被覆盖
	in2 := map[string]any{"genderId": "male", "genderLabel": "男"}
	out2 := fillPlayerDefaults(in2).(map[string]any)
	if out2["genderId"] != "male" {
		t.Fatalf("preset genderId must be kept, got %#v", out2["genderId"])
	}
}
