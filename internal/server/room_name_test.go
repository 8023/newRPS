package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

// TestGeneratedRoomNameDefaultsPerGame 回归覆盖：generatedRoomName 在没有主题房名词库
// （未选惩罚标签/标签没配房名词库）时必须按 settings.GameID 兜底成对应游戏的占位名，
// 不能不分青红皂白地统一兜底成 RPS 的 defaultRoomName——这正是新建猜硬币房间在大厅列表里
// 被打上「锤子剪刀布」标签的根因（生成的房名恰好等于 defaultRoomName，又被
// normalizeRoomName 当作占位符触发二次生成，最终把游戏无关的 RPS 占位名落地成了房名）。
func TestGeneratedRoomNameDefaultsPerGame(t *testing.T) {
	s := newTestServer(t)
	cases := []struct {
		name             string
		gameID           types.GameID
		enablePunishment bool
		want             string
	}{
		{"rps 无惩罚", types.GameRPS, false, defaultRoomName},
		{"猜硬币 强制开启惩罚但无主题词库", types.GameCoinFlip, true, defaultCoinFlipRoomName},
		{"大话骰 无惩罚", types.GameLiarsDice, false, defaultLiarsDiceRoomName},
		{"大话骰 开启惩罚但无主题词库", types.GameLiarsDice, true, defaultLiarsDiceRoomName},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			settings := types.RoomSettings{GameID: c.gameID, EnablePunishment: c.enablePunishment, PunishmentSource: "random"}
			got := s.generatedRoomName(settings)
			if got != c.want {
				t.Fatalf("generatedRoomName(%s) = %q, want %q", c.gameID, got, c.want)
			}
		})
	}
}

// TestNormalizeRoomNameRecognizesAllGamePlaceholders 回归覆盖：normalizeRoomName 必须把
// 全部 8 款游戏各自的占位默认名都识别为"待重新生成"，而不仅仅是最早的 5 款——否则前端一旦
// 发生同类 bug、把猜硬币/大话骰房间的名字算错成别的占位符发给服务端，服务端会原样接受，
// 不会按游戏纠正。
func TestNormalizeRoomNameRecognizesAllGamePlaceholders(t *testing.T) {
	s := newTestServer(t)
	settings := types.RoomSettings{GameID: types.GameCoinFlip, EnablePunishment: true, PunishmentSource: "random", Name: defaultRoomName}
	got := s.normalizeRoomName(settings)
	if got != defaultCoinFlipRoomName {
		t.Fatalf("normalizeRoomName 对误传 RPS 占位名的猜硬币房间应纠正为 %q，实际得到 %q", defaultCoinFlipRoomName, got)
	}
}
