package server

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/doumiao/newRPS/internal/geoip"
)

// currentSchemaVersion 是代码期望的数据库结构版本号。每次改动某张表的列（加/删/改名/
// 拆表）都必须：
//  1. 把对应 store 文件里的 xxxSchema 常量改成目标结构；
//  2. 在下面的 migrations 里追加一条 {version: currentSchemaVersion+1, migrate: ...}，
//     用 ALTER TABLE / 建新表倒数据 / DROP+RENAME 之类的显式语句把旧数据搬过去；
//  3. 把这个常量加一。
//
// 是 var 不是 const，只是为了让测试能临时替换掉去验证迁移机制本身；正常代码路径里
// 把它当常量对待，不要在业务逻辑里修改它。
var currentSchemaVersion = 48

// schemaVersionSchema：只有一行的版本表，openDatabase 每次启动都会先确保它存在。
const schemaVersionSchema = `
CREATE TABLE IF NOT EXISTS schema_version (
	version INTEGER NOT NULL
);
`

// schemaMigration 是一步"从更早版本升到 version"的显式迁移。migrate 拿到的是当前事务
// （*sql.Tx，满足 sqlExecer），迁移语句和版本号回写在同一个事务里提交，保证原子——
// 不会出现"迁移做了一半、版本号却没更新，下次重启重复执行报错"的情况。
//
// 迁移必须幂等：version==0 的旧库（CREATE TABLE IF NOT EXISTS 不会给已有表补列）
// 与"已经是最新结构的全新库"都会跑这些步骤；对已存在的目标列/已改名的列要跳过，
// 不能假设"旧列一定还在"。
type schemaMigration struct {
	version int
	migrate func(sqlExecer) error
}

// sqlExecer 是 *sql.DB / *sql.Tx 的公共子集，migrate 函数只需要这几个方法就够用。
type sqlExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// migrations 按 version 升序排列。
var migrations = []schemaMigration{
	// v2：player_activity_events.name/old_name 改名为 new_value/old_value；
	// 移除聊天消息的 author_player 快照列。
	{version: 2, migrate: func(db sqlExecer) error {
		if err := renameColumnIfExists(db, "player_activity_events", "name", "new_value"); err != nil {
			return err
		}
		if err := renameColumnIfExists(db, "player_activity_events", "old_name", "old_value"); err != nil {
			return err
		}
		if err := dropColumnIfExists(db, "lobby_messages", "author_player"); err != nil {
			return err
		}
		if err := dropColumnIfExists(db, "room_messages", "author_player"); err != nil {
			return err
		}
		return nil
	}},
	// v3：排位分历史最高/最低 + 每日衰减去重标记；以当前 ranked_points 回填极值。
	{version: 3, migrate: func(db sqlExecer) error {
		if err := addColumnIfMissing(db, "players", "highest_score", "INTEGER NOT NULL DEFAULT 0"); err != nil {
			return err
		}
		if err := addColumnIfMissing(db, "players", "lowest_score", "INTEGER NOT NULL DEFAULT 0"); err != nil {
			return err
		}
		if err := addColumnIfMissing(db, "players", "ranked_last_decay_day", "INTEGER"); err != nil {
			return err
		}
		// 回填可重复执行：只在极值仍是 0 且 ranked_points 非 0 时写，避免覆盖已有历史。
		if _, err := db.Exec(`UPDATE players SET highest_score = ranked_points WHERE highest_score = 0 AND ranked_points > 0`); err != nil {
			return fmt.Errorf("backfill highest_score: %w", err)
		}
		if _, err := db.Exec(`UPDATE players SET lowest_score = ranked_points WHERE lowest_score = 0 AND ranked_points < 0`); err != nil {
			return fmt.Errorf("backfill lowest_score: %w", err)
		}
		return nil
	}},
	// v4：把 execSchemaQuarantiningLegacyTables 隔离出的 punishment_events_legacy
	// （旧版 kind/source/player_id/at 列）尽量还原进新版单行任务结构。
	{version: 4, migrate: func(db sqlExecer) error {
		return convertLegacyPunishmentEvents(db)
	}},
	// v5：删除 rooms.code 遗留列（旧版 DM-XXXX 房间码，新代码不再写入/读取）。
	{version: 5, migrate: func(db sqlExecer) error {
		return dropColumnIfExists(db, "rooms", "code")
	}},
	// v6：删除 players.player_secret_hash 遗留列（旧版单值哈希凭据，身份认证已改为
	// 明文 PlayerSecrets 列表，见 identity.go；此列不再被任何代码读写）。
	{version: 6, migrate: func(db sqlExecer) error {
		return dropColumnIfExists(db, "players", "player_secret_hash")
	}},
	// v7：新增 players.title_custom——标记称号是否由管理员在后台手动设置（不随排位分
	// 档位变化自动改写），见 player.go 的 syncTitleForRankSegment。旧数据一律视为非自定义。
	{version: 7, migrate: func(db sqlExecer) error {
		return addColumnIfMissing(db, "players", "title_custom", "INTEGER NOT NULL DEFAULT 0")
	}},
	// v8：性别与阵营解耦——新增 players.faction_id（阵营现在独立于性别选择，不能再从
	// gender_id 反推，必须落库）和 players.custom_gender_label（自定义性别文本，1-9
	// 字符；预设性别时留空，展示文案从当前配置的 genders.json 里查）。旧数据两列都是
	// 空字符串，由 persist.go 的 ingestPersistedPlayer 按旧默认阵营 ID 一次性回填。
	{version: 8, migrate: func(db sqlExecer) error {
		if err := addColumnIfMissing(db, "players", "faction_id", "TEXT NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		return addColumnIfMissing(db, "players", "custom_gender_label", "TEXT NOT NULL DEFAULT ''")
	}},
	// v9：累计在线时长（毫秒）。断线 / 优雅关停时累加本会话时长；非优雅退出可丢
	// 未落盘会话，属有意接受的边界。
	{version: 9, migrate: func(db sqlExecer) error {
		return addColumnIfMissing(db, "players", "total_online_ms", "INTEGER NOT NULL DEFAULT 0")
	}},
	// v10：认主/认宠玩法——players 三偏好开关；pet_bonds / pet_bond_requests
	// 由 petBondSchema 的 CREATE IF NOT EXISTS 建表，此处只补 players 列。
	{version: 10, migrate: func(db sqlExecer) error {
		if err := addColumnIfMissing(db, "players", "bond_master_enabled", "INTEGER"); err != nil {
			return err
		}
		if err := addColumnIfMissing(db, "players", "bond_pet_enabled", "INTEGER"); err != nil {
			return err
		}
		return addColumnIfMissing(db, "players", "bond_public_display", "INTEGER")
	}},
	// v11：斗兽棋分游戏胜负平。
	{version: 11, migrate: func(db sqlExecer) error {
		if err := addColumnIfMissing(db, "players", "jungle_wins", "INTEGER NOT NULL DEFAULT 0"); err != nil {
			return err
		}
		if err := addColumnIfMissing(db, "players", "jungle_losses", "INTEGER NOT NULL DEFAULT 0"); err != nil {
			return err
		}
		return addColumnIfMissing(db, "players", "jungle_draws", "INTEGER NOT NULL DEFAULT 0")
	}},
	// v12：回填累计在线时长。v9 起会在断线/关停时累加，但优雅关停路径曾漏 markPersistDirty，
	// 导致大量已写入 connection_events 的 server_shutdown 会话未进 total_online_ms。
	// 用 connection_events 的时长之和抬升（不降低）已有值；表不存在时跳过。
	{version: 12, migrate: func(db sqlExecer) error {
		return backfillTotalOnlineMsFromConnectionEvents(db)
	}},
	// v13：再次回填累计在线时长。v12 之后仍有一条致命 bug——ingestPersistedPlayer 加载
	// 档案时丢掉 TotalOnlineMs，每次重启内存从 0 起算，随后 flush 把库里已回填的正确值
	// 覆盖成仅本进程会话。此处幂等抬升，配合加载修复把 connection_events 历史时长找回。
	{version: 13, migrate: func(db sqlExecer) error {
		return backfillTotalOnlineMsFromConnectionEvents(db)
	}},
	// v14：旧版 rooms（一房间一行的生命周期表）+ room_join_events（可重复发生的加入
	// 事件表）从未真正并入取代它们的 room_events 统一事件日志——历史数据被留在原地
	// 冻结，两张旧表也一直没删。这里把它们的历史行无损转换成 room_events 的
	// create/join/close 行，写完后彻底删表，不再保留中间状态。
	{version: 14, migrate: func(db sqlExecer) error {
		return migrateLegacyRoomEvents(db)
	}},
	// v15：删除自定义性别功能，回归查表法定阵营（性别决定阵营，阵营不再独立提交/存储覆盖）；
	// 新增 players.self_title（玩家自设称号，展示优先级低于管理员自定义/主人设置，高于系统
	// 按排位分自动计算的称号）。存量选了自定义性别的玩家（custom_gender_label 非空、
	// gender_id 为空）按其当前 faction_id 改成该阵营配置的默认（第一个）预设性别；生产库
	// 需要升级到这一版本才会跑这段一次性回填，回填用的 faction→默认性别映射写死在代码里，
	// 对应改动时（2026-07-27）的 config/genders.json 内容，不会随后续配置调整而更新。
	{version: 15, migrate: func(db sqlExecer) error {
		if err := migrateCustomGenderPlayers(db); err != nil {
			return err
		}
		return addColumnIfMissing(db, "players", "self_title", "TEXT NOT NULL DEFAULT ''")
	}},
	// v16：认主/认宠/解除关系的申请统一为"双方须持续在线，任一方提前下线即自动撤销"，
	// 逐条同意（Approvals）只需要在内存里追踪，重启清空重新来过完全可以接受，不必落库。
	// 删除只用于记录"谁已经同意"的 pet_bond_request_approvals 表。
	{version: 16, migrate: func(db sqlExecer) error {
		_, err := db.Exec(`DROP TABLE IF EXISTS pet_bond_request_approvals`)
		return err
	}},
	// v17：接入用户分析（analytics_* 四张新表由 allSchemas 建好，此处不重复）。
	// 本迁移只补既有审计表缺失的「时间维度」索引——这些表此前只有 (player_id, at) /
	// (room_id, at) 这类以实体为前导列的索引，任何「按时间段扫全站」的聚合都只能全表扫。
	// 分析聚合器每分钟都要按天分桶跑这些查询，没有这几个索引会在单连接 SQLite 上直接堵住写入。
	// CREATE INDEX IF NOT EXISTS 天然幂等，全新库与存量库都能重复执行。
	// 同时给 connection_events 补 province/isp 列，供归属地分析回填。
	{version: 17, migrate: func(db sqlExecer) error {
		if err := addColumnIfMissing(db, "connection_events", "province", "TEXT"); err != nil {
			return err
		}
		if err := addColumnIfMissing(db, "connection_events", "isp", "TEXT"); err != nil {
			return err
		}
		for _, stmt := range []string{
			`CREATE INDEX IF NOT EXISTS idx_connection_events_at    ON connection_events(connected_at)`,
			`CREATE INDEX IF NOT EXISTS idx_player_activity_at      ON player_activity_events(at, action)`,
			`CREATE INDEX IF NOT EXISTS idx_room_events_at          ON room_events(at, action)`,
			`CREATE INDEX IF NOT EXISTS idx_punishment_events_at    ON punishment_events(task_at)`,
			`CREATE INDEX IF NOT EXISTS idx_punishment_events_proof ON punishment_events(proof_at)`,
			`CREATE INDEX IF NOT EXISTS idx_lobby_messages_at       ON lobby_messages(at)`,
			`CREATE INDEX IF NOT EXISTS idx_room_messages_at        ON room_messages(at)`,
			`CREATE INDEX IF NOT EXISTS idx_pet_bonds_created       ON pet_bonds(created_at)`,
			`CREATE INDEX IF NOT EXISTS idx_players_created_at      ON players(created_at)`,
			`CREATE INDEX IF NOT EXISTS idx_players_last_seen_at    ON players(last_seen_at)`,
		} {
			if _, err := db.Exec(stmt); err != nil {
				return fmt.Errorf("v17 index: %w", err)
			}
		}
		return nil
	}},
	// v18：一次性回填 v17 新增的 connection_events.province/isp——用该表本就存在的 ip 列
	// 挨行跑 geoip 解析。只在这次库升级时执行一次（受 schema_version 门控，不会每次启动
	// 重跑）；geoip 未启用（ANALYTICS_GEO_ENABLED=0 或 xdb 未加载）时整段跳过，不产生空写入。
	// 要求调用方在 openDatabase（从而触发这条迁移）之前已经完成 geoip.Init/InitV6，见
	// server.go New() 里的顺序注释。
	{version: 18, migrate: func(db sqlExecer) error {
		return backfillConnectionEventsGeo(db)
	}},
	// v19：修正 v18 写入的错误数据。geoip.parseRegion 当时把字段顺序当成旧版 ip2region
	// 文档里的「国家|区域|省份|城市|ISP」，实际 ip2region_v4/v6.xdb 用的是「国家|省份|
	// 城市|ISP|iso-alpha2-code」——v18 因此把「城市」错当成「省份」写进 province（非中国
	// 地区的城市字段常年是占位符 "0"，所以这些行的 province 全是空字符串)，把两字母国家码
	// 错当成「ISP」写进 isp。geoip 包字段索引已经修好，这里无条件对所有 ip 非空的历史行
	// 重新解析并覆盖 province/isp——不能像 v18 那样只挑空值行，因为 v18 已经把它们"填满"
	// 成了错误值，不会再被空值判断选中。同样只在这次库升级时跑一次。
	{version: 19, migrate: func(db sqlExecer) error {
		return refixConnectionEventsGeo(db)
	}},
	// v20：同一个 parseRegion 字段错位 bug 也污染了 analytics_sessions.province/city/isp
	// 与 analytics_visitors.first_province（analytics_collect.go 里两边都是拿
	// client.anaGeo 直接落盘，和 connection_events 用的是同一次 geoip.Lookup 结果）。
	// 这两张表按设计不存原始 IP（隐私：分析表不存 IP/指纹/原始 UA，见 README），没法像
	// connection_events 那样重新解析出正确值，只能清空——后台"省份/ISP"图表把这些行当
	// "无归属地数据"处理，好过继续显示错误的城市名/国家码。country 字段用的是 parts[0]，
	// 新旧代码取法一致，没被这个 bug 污染，不清。
	// 同时删掉 analytics_daily 里已经按错误数据聚合出的 province/city/isp 行，避免历史
	// 错误汇总数字（比如 ISP 图表里一堆 "CN"）继续挂在面板上——这几个 metric 的历史日
	// 汇总不会被自动重算（聚合器每分钟只重算今天/昨天，backfillFromLegacyTables 只回填
	// analytics_daily 里完全没有该 day 记录的日子），删掉后这些历史日就会正确地显示"无
	// 数据"而不是错误数据；今天/昨天会在下一次聚合时用已修好的 geoip 重新算出正确值。
	// 迁移只在这次库升级时执行一次，不依赖 geoip 是否启用（纯粹是清除已知错误数据，
	// 不需要重新查表）。
	{version: 20, migrate: func(db sqlExecer) error {
		return clearWrongAnalyticsGeo(db)
	}},
	// v21：第四个推送来源——「我的主/宠上线」。新增 players.push_bond_enabled，与
	// push_mention_enabled/push_turn_enabled/push_seat_enabled 同一约定（NULL 视为未开启）。
	{version: 21, migrate: func(db sqlExecer) error {
		return addColumnIfMissing(db, "players", "push_bond_enabled", "INTEGER")
	}},
	// v22：analytics 惩罚「完成」曾按 status='done' 聚合，而 eventstore 审核通过写的是
	// approved，导致 analytics_daily 里 punish_done / funnel.punish_done 恒为 0。
	// 已 sealed 的历史日不会被每分钟 rebuild 覆盖，backfill 也只补「完全没有该 day」的日子，
	// 所以在迁移里从 punishment_events 源表按本地日重算写回（与 v20 清错误 geo 同类：
	// 一次性数据纠正，靠 schema_version 保证只跑一次）。
	{version: 22, migrate: func(db sqlExecer) error {
		return fixAnalyticsPunishmentDoneMetric(db)
	}},
	// v23：转化漏斗键重定义（visit/lobby/room/round/finish），废弃 login/punish_done。
	// sealed 历史日不会被聚合器重算，在此一次性按源表重写 funnel 行。
	{version: 23, migrate: func(db sqlExecer) error {
		if currentSchemaVersion >= 29 {
			return nil // 最终口径由同一升级链末端的 v29 一次性重算，避免重复扫全量历史。
		}
		return recomputeAnalyticsFunnelMetrics(db)
	}},
	// v24：漏斗五层统一为设备 UV（player_id→visitor 映射）；v23 曾对后三层写事件次数/玩家 UV。
	{version: 24, migrate: func(db sqlExecer) error {
		if currentSchemaVersion >= 29 {
			return nil
		}
		return recomputeAnalyticsFunnelMetrics(db)
	}},
	// v25：惩罚标签偏好 + 系列任务进度两张规范化表（见 playerstore）。
	{version: 25, migrate: func(db sqlExecer) error {
		if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS player_punishment_tag_prefs (
	player_id TEXT NOT NULL,
	tag_id    TEXT NOT NULL,
	state     TEXT NOT NULL,
	PRIMARY KEY (player_id, tag_id)
)`); err != nil {
			return err
		}
		if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_player_punishment_tag_prefs_player ON player_punishment_tag_prefs(player_id)`); err != nil {
			return err
		}
		if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS player_punishment_series_progress (
	player_id  TEXT NOT NULL,
	series_id  TEXT NOT NULL,
	next_index INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (player_id, series_id)
)`); err != nil {
			return err
		}
		if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_player_punishment_series_progress_player ON player_punishment_series_progress(player_id)`); err != nil {
			return err
		}
		return nil
	}},
	// v26：任务池 + 系列任务改为 SQLite 规范化存储（原 punishments.json 的 tasks/seriesTasks 字段废弃）。
	{version: 26, migrate: func(db sqlExecer) error {
		if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS punishment_tasks (
	id                 TEXT PRIMARY KEY,
	name               TEXT NOT NULL DEFAULT '',
	text               TEXT NOT NULL DEFAULT '',
	tag_ids            TEXT NOT NULL DEFAULT '[]',
	faction_ids        TEXT NOT NULL DEFAULT '[]',
	difficulty_order   INTEGER NOT NULL DEFAULT 50,
	background_images  TEXT NOT NULL DEFAULT '[]',
	background_opacity REAL NOT NULL DEFAULT 0.22,
	sort_index         INTEGER NOT NULL DEFAULT 0
)`); err != nil {
			return err
		}
		if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS punishment_series (
	id                     TEXT PRIMARY KEY,
	name                   TEXT NOT NULL DEFAULT '',
	room_name_pool         TEXT NOT NULL DEFAULT '{}',
	room_background_images TEXT NOT NULL DEFAULT '[]',
	steps                  TEXT NOT NULL DEFAULT '[]',
	sort_index             INTEGER NOT NULL DEFAULT 0
)`); err != nil {
			return err
		}
		return nil
	}},
	// v27：漏斗「访问」层改为 sessions∪events 的 DISTINCT visitor（此前只查
	// analytics_sessions，而会话行的 visitor/day 在首次落库后不再随 UPSERT 更新——
	// 换网/换 IP 或跨零点保持同一会话都会让同一人后续事件落到新 visitor/日期却不
	// 回写 sessions 表，导致 lobby/room/... 各层反而比 visit 还多，转化率超过
	// 100%）。已 sealed 的历史日一次性按新口径重写；同一函数也顺带重写其余四层，
	// 结果与线上聚合器口径保持一致。
	{version: 27, migrate: func(db sqlExecer) error {
		if currentSchemaVersion >= 29 {
			return nil
		}
		return recomputeAnalyticsFunnelMetrics(db)
	}},
	// v28：聊天管理改为软删除（不再有"一键清空大厅/房间聊天"，只能对检索出的具体消息
	// 单条/批量删除）。lobby_messages/room_messages 新增 deleted/deleted_at；
	// room_messages 另新增 room_name——发送时刻的房间名快照，供房间关闭/改名后仍可
	// 按房间名检索历史消息（旧行回填为空串，检索时按"房间名"过滤会漏掉存量消息，
	// 属可接受的历史数据边界，本就无法凭 room_id 反查已关闭房间当时的名字）。
	{version: 28, migrate: func(db sqlExecer) error {
		if err := addColumnIfMissing(db, "lobby_messages", "deleted", "INTEGER NOT NULL DEFAULT 0"); err != nil {
			return err
		}
		if err := addColumnIfMissing(db, "lobby_messages", "deleted_at", "INTEGER"); err != nil {
			return err
		}
		if err := addColumnIfMissing(db, "room_messages", "deleted", "INTEGER NOT NULL DEFAULT 0"); err != nil {
			return err
		}
		if err := addColumnIfMissing(db, "room_messages", "deleted_at", "INTEGER"); err != nil {
			return err
		}
		if err := addColumnIfMissing(db, "room_messages", "room_name", "TEXT NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		for _, stmt := range []string{
			`CREATE INDEX IF NOT EXISTS idx_lobby_messages_deleted ON lobby_messages(deleted)`,
			`CREATE INDEX IF NOT EXISTS idx_room_messages_deleted  ON room_messages(deleted)`,
			`CREATE INDEX IF NOT EXISTS idx_lobby_messages_id      ON lobby_messages(id)`,
			`CREATE INDEX IF NOT EXISTS idx_room_messages_id       ON room_messages(room_id, id)`,
		} {
			if _, err := db.Exec(stmt); err != nil {
				return fmt.Errorf("v28 index: %w", err)
			}
		}
		return nil
	}},
	// v29：漏斗改为「深层向浅层逐级并集回填」，五层按定义单调（见 computeFunnelUV）。
	// v27 只修好了「访问」层，但 game_start 埋点比 game_round 晚上线（v2.4.3，
	// 2026-08-12 23:16），上线当天只覆盖了一天里最后几十分钟，那天的「开局」远小于
	// 「完成对局」，区间求和后整体仍倒挂约 10%。已 sealed 的历史日按新口径重写。
	{version: 29, migrate: func(db sqlExecer) error {
		return recomputeAnalyticsFunnelMetrics(db)
	}},
	// v30：随机任务标签三态偏好改为纯浏览器本地存储（不再跨设备同步），废弃
	// player_punishment_tag_prefs 表（惩罚标签数据分析统计走独立的 analytics_events，不受影响）。
	{version: 30, migrate: func(db sqlExecer) error {
		_, err := db.Exec(`DROP TABLE IF EXISTS player_punishment_tag_prefs`)
		return err
	}},
	// v31：国际象棋分游戏胜负平。
	{version: 31, migrate: func(db sqlExecer) error {
		if err := addColumnIfMissing(db, "players", "chess_wins", "INTEGER NOT NULL DEFAULT 0"); err != nil {
			return err
		}
		if err := addColumnIfMissing(db, "players", "chess_losses", "INTEGER NOT NULL DEFAULT 0"); err != nil {
			return err
		}
		return addColumnIfMissing(db, "players", "chess_draws", "INTEGER NOT NULL DEFAULT 0")
	}},
	// v32：删除按玩家落盘的系列进度表。此后进度只存在房间内存里
	// （见 RoomState.PunishmentSeriesPlayerProgress：房间内按玩家独立计数）。
	{version: 32, migrate: func(db sqlExecer) error {
		_, err := db.Exec(`DROP TABLE IF EXISTS player_punishment_series_progress`)
		return err
	}},
	// v33：共建——性别/阵营迁 SQLite；投稿/版本/赞踩；惩罚事件补评价字段。
	{version: 33, migrate: migrateContributionV33},
	// v34：系列任务声明目标阵营，供进战斗席前端校验与发布覆盖校验。
	{version: 34, migrate: func(db sqlExecer) error {
		return addColumnIfMissing(db, "punishment_series", "target_faction_ids", "TEXT NOT NULL DEFAULT '[]'")
	}},
	// v35：投票粒度从"整份投稿/整个系列"下沉到"一个具体任务"（同一次投稿/同一个系列步骤
	// 生成的所有阵营变体行共享一个 task_group_id，系列不同步骤各不相同）——新增
	// task_group_id 列；已有行没有这个概念，先按 submission_id 回填当兜底（等价于旧行为，
	// 后续重新发布会按新规则各步骤独立分组）。
	{version: 35, migrate: func(db sqlExecer) error {
		if err := addColumnIfMissing(db, "punishment_tasks", "task_group_id", "TEXT NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		// 走过投稿流程的行用 submission_id 兜底（旧行为：一份投稿一个组）；共建上线前的
		// 存量行 submission_id 为空，不能整表折成同一个空组，改用各自的任务 id。
		_, err := db.Exec(`
			UPDATE punishment_tasks SET task_group_id = CASE
				WHEN submission_id <> '' THEN submission_id
				ELSE id
			END WHERE task_group_id = ''`)
		return err
	}},
	// v36：随机任务池 / 系列任务新增 created_at，供数据分析「随机任务」「系列任务」增长图
	// 与后台共建审核总览统计池子大小（见 punishmentstore.go / analytics_agg.go）。存量行
	// 没有这个概念，一律回填 0（视为"早于统计起点"，不计入任何一天的「新增」，但仍会被
	// 计入所有日期的「总量」——与 pet_bonds.created_at 的既有口径一致）。
	{version: 36, migrate: func(db sqlExecer) error {
		if err := addColumnIfMissing(db, "punishment_tasks", "created_at", "INTEGER NOT NULL DEFAULT 0"); err != nil {
			return err
		}
		return addColumnIfMissing(db, "punishment_series", "created_at", "INTEGER NOT NULL DEFAULT 0")
	}},
	// v37：删除示例系列任务"撸管废物改造手册"（id=series1）及其专属的 5 个子任务
	// （id=truth_task1_default/task1/task2/task5/task6，其 task_group_id 均等于自身
	// id，即从未经过投稿流程、系列内独占，删除前已核实没有其它系列的步骤引用这 5 个 id，
	// 也没有历史 punishment_events 行以 formal_series_id/formal_task_id 指向它们）。
	// 这批 id 是早期从 config/json/punishments.json 一次性迁移进 SQLite 时使用的固定
	// 字面量种子 id（见 internal/config/config.go 的迁移逻辑），共建投稿流程生成的新
	// 任务/系列 id 一律带 "ct_"/"cs_" 前缀，不会与之冲突，按字面量精确匹配删除是安全的。
	// DELETE 对不存在的行是空操作，天然幂等。
	{version: 37, migrate: func(db sqlExecer) error {
		if _, err := db.Exec(`DELETE FROM punishment_series WHERE id = 'series1'`); err != nil {
			return err
		}
		_, err := db.Exec(`DELETE FROM punishment_tasks WHERE id IN ('truth_task1_default', 'task1', 'task2', 'task5', 'task6')`)
		return err
	}},
	// v38：补两块此前只在内存里、进程重启即丢失的玩家状态：
	//  1. 白给自救板当前挂着的内容（文本+过期时间+赞踩计数），此前 ingestPersistedPlayer
	//     加载时硬编码 likes/dislikes 为 0、文本压根不读，重启后大厅"白给自救板"栏目清空；
	//  2. 极限模式"强行关闭"在"通用改名处"留下的可改名标记与改名保护期/记录——
	//     这批字段（extreme_force_closed 等）此前从未落库，与已经完整持久化的名争
	//     （name_war_* 全套字段）不是一回事，容易被误以为"通用改名处的内容都会丢"。
	{version: 38, migrate: func(db sqlExecer) error {
		for _, col := range []struct{ name, decl string }{
			{"extreme_force_closed", "INTEGER"},
			{"extreme_force_closed_at", "INTEGER"},
			{"extreme_rename_protected_until", "INTEGER"},
			{"extreme_renamed_by", "TEXT NOT NULL DEFAULT ''"},
			{"extreme_renamed_by_name", "TEXT NOT NULL DEFAULT ''"},
			{"giveaway_board_text", "TEXT NOT NULL DEFAULT ''"},
			{"giveaway_board_submitted_at", "INTEGER"},
			{"giveaway_board_expires_at", "INTEGER"},
			{"giveaway_board_likes", "INTEGER"},
			{"giveaway_board_dislikes", "INTEGER"},
		} {
			if err := addColumnIfMissing(db, "players", col.name, col.decl); err != nil {
				return err
			}
		}
		return nil
	}},
	// v39：随机任务/系列每一步都去掉"名称"输入——玩家和管理员都不会看到它：游戏内展示的
	// 分类文字来自惩罚标签名，系列本身的展示名是系列自己的 name，与每条任务/每一步各自
	// 的名称无关（见 punishment.go 的 punishmentRoundLabel/tagNamesForTask）。发布后每份
	// 变体行的 id 本就是全局唯一标识，不需要再额外维护一个人肉起的标题。
	{version: 39, migrate: func(db sqlExecer) error {
		return dropColumnIfExists(db, "punishment_tasks", "name")
	}},
	// v40：共建详情与数据分析会按 submission/version/task_group、随机池难度/创建时间、
	// 系列创建时间查询。没有这些索引时，投稿池增长后每次详情/聚合都会全表扫描；尤其详情
	// RPC 运行在全局游戏锁内，会直接放大成对实时对局的停顿。
	{version: 40, migrate: func(db sqlExecer) error {
		for _, stmt := range []string{
			`CREATE INDEX IF NOT EXISTS idx_punishment_tasks_submission_vote_group ON punishment_tasks(submission_id, vote_version, task_group_id)`,
			`CREATE INDEX IF NOT EXISTS idx_punishment_tasks_pool_created ON punishment_tasks(difficulty_order, created_at, task_group_id)`,
			`CREATE INDEX IF NOT EXISTS idx_punishment_series_created ON punishment_series(created_at)`,
			`CREATE INDEX IF NOT EXISTS idx_punishment_events_formal_task ON punishment_events(formal_task_id, formal_task_version)`,
		} {
			if _, err := db.Exec(stmt); err != nil {
				return err
			}
		}
		return nil
	}},
	// v41：性别共建支持一次投稿勾选多个阵营——一份投稿展开成每个阵营一行 gender_options，
	// 需要一列记录"这行是哪份投稿发布的"，供下架/取消撤回/编辑重发按投稿整体增删改
	// （而不是像过去单阵营那样，投稿与正式行天然一一对应，直接用 published_target_id 定位）。
	{version: 41, migrate: func(db sqlExecer) error {
		if err := addColumnIfMissing(db, "gender_options", "submission_id", "TEXT NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		// 回填存量数据：改动前每份性别投稿正式发布后天然只有一行、且
		// contribution_submissions.published_target_id 就是那一行的 id，一一对应。
		// 不回填的话，存量已通过的性别投稿下次修订重新提交时，claimGenderNameTx 会把
		// 它自己名下这行也当成"别人占用的同名行"而报错拒绝；下架/取消撤回时
		// deleteGenderRowsBySubmissionTx 按 submission_id 也会找不到这行、删不掉。
		// （这张表连同性别投稿功能本身已在 v43 一并删除/下线，这里的回填对现在的代码
		// 已经没有实际效果，但迁移步骤本身仍要保留、且不能假设 contribution_submissions
		// 一定存在——没有全部历史包袱的库/测试环境可能从没建过这张表。）
		hasSubmissions, err := tableExists(db, "contribution_submissions")
		if err != nil {
			return err
		}
		if hasSubmissions {
			if _, err := db.Exec(`
				UPDATE gender_options SET submission_id = (
					SELECT s.id FROM contribution_submissions s
					WHERE s.kind = 'gender' AND s.published_target_id = gender_options.id
					LIMIT 1
				)
				WHERE submission_id = '' AND EXISTS (
					SELECT 1 FROM contribution_submissions s
					WHERE s.kind = 'gender' AND s.published_target_id = gender_options.id
				)`); err != nil {
				return err
			}
		}
		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_gender_options_submission ON gender_options(submission_id)`)
		return err
	}},
	// v42：共建封面图访问权限从"任务是否仍在正式任务池里启用"改成"这张图当前是不是它所属
	// 投稿仍在引用的封面"（不看投稿审核状态），需要一列记录"是否仍被引用"——重新上传覆盖
	// 替换掉的旧图才是孤儿（active=0），文件本身不删。存量数据统一按 active=1（可访问）
	// 迁移：老库里从没追踪过"是否被替换过"，没法回溯哪些其实已经是孤儿，默认放行更安全，
	// 后续这些投稿只要再保存/发布一次就会被 markOtherSubmissionImagesInactiveTx 自然纠正。
	{version: 42, migrate: func(db sqlExecer) error {
		if err := addColumnIfMissing(db, "contribution_images", "active", "INTEGER NOT NULL DEFAULT 1"); err != nil {
			return err
		}
		_, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_contrib_images_submission_active ON contribution_images(submission_id, active)`)
		return err
	}},
	// v43：共建投稿存储重构——punishment_tasks/punishment_series 整表替换式存储 +
	// contribution_submissions/versions 投稿信封两层模型，改成 sub_tasks/series 两张
	// 版本化、插入不更新的表（不再有单独的投稿信封/图片/投票表，见 CLAUDE.md「共建投稿」
	// 一节）。旧内容（迁移前已在线上生效、从未走过投稿流程）原样迁移进新表并标记为
	// approved；punishment_events 的 publisher/target 改名为 approver/performer，新增
	// 评价与触发来源列；性别投稿功能下线，gender_options.submission_id 随之作废删除；
	// 所有被取代的旧表整体 DROP（迁移逻辑见 punishment_migrate_v43.go）。
	{version: 43, migrate: migratePunishmentToSubTasksV43},
	// v44：系列缩短时需要给被删除的尾部步骤写一条 inactive 新版本，才能在保留完整
	// 历史的同时明确区分“当前系列成员”和“历史上曾属于该系列的步骤”。仅靠 status
	// 无法表达这个关系：整个系列下架后的 withdrawn 步骤与被缩短移除的步骤会混在一起，
	// 取消下架时可能把已删除步骤误发布回来。全新库已由 contributionSchema 直接带此列；
	// 跑过早期 v43 草案的库在这里补列并把既有行视为 active。
	{version: 44, migrate: func(db sqlExecer) error {
		if err := addColumnIfMissing(db, "sub_tasks", "active", "INTEGER NOT NULL DEFAULT 1"); err != nil {
			return err
		}
		// CREATE INDEX IF NOT EXISTS 不会刷新早期 v43 草案已创建的旧索引定义，显式重建。
		if _, err := db.Exec(`DROP INDEX IF EXISTS idx_sub_tasks_series`); err != nil {
			return err
		}
		_, err := db.Exec(`CREATE INDEX idx_sub_tasks_series ON sub_tasks(series_id, active, step_index)`)
		return err
	}},
	// v45：早期 v43 重构草案仍把旧兼容脚手架放在 allSchemas，导致已升级数据库重启后
	// 又创建空旧表。再次清理一次，并由 ensureSchema 保证以后只在 version<43 时建脚手架。
	{version: 45, migrate: dropLegacyPunishmentTables},
	// v46：删除 gender_factions/gender_options 冗余的 normalized_label 列。它当初只是
	// 为了让 DB 层唯一约束能做大小写/全半角不敏感去重而派生出来的比较键，同一份归一化
	// 逻辑（normalizeGenderLabel）本就在应用层现算（contribution_admin.go 的后台保存
	// 校验、gender_import.go 的种子去重），不需要额外持久化一列——单纯为了撑起
	// UNIQUE(faction_id, normalized_label) 而多存一份数据。
	// gender_factions.normalized_label 从未参与任何约束/索引，直接 DROP COLUMN 即可；
	// gender_options.normalized_label 是表级 UNIQUE(faction_id, normalized_label) 的
	// 组成部分，SQLite 不允许 DROP 参与 UNIQUE 约束的列，只能整表重建为
	// UNIQUE(faction_id, label)。由"规范化后相同即冲突"（更严格）改成"原样相同才冲突"
	// （更宽松）：任何满足旧约束的存量数据集合，天然也满足新约束（相同字面量必然规范化
	// 相同，旧约束早已挡掉这类重复），重建过程不会因为约束放宽而产生新冲突。
	{version: 46, migrate: func(db sqlExecer) error {
		if err := dropColumnIfExists(db, "gender_factions", "normalized_label"); err != nil {
			return err
		}
		return rebuildGenderOptionsWithoutNormalizedLabel(db)
	}},
	// v47：players 表的名争/白给/极限模式/分游戏胜负平四组共 50 列拆到四张独立子表
	// （player_name_war/player_giveaway/player_extreme_mode/player_game_stats），
	// 详见 player_migrate_v47.go 顶部注释。
	{version: 47, migrate: migratePlayerSubTablesV47},
	// v48：punishment_events 的 task_source/formal_series_id/formal_series_version/
	// contributor_player_id/contributor_name_snapshot/contributor_anonymous 六列，以及
	// sub_tasks/series 各自的 contributor_name 列，全部可以由 formal_task_id/
	// formal_task_version（外加对 sub_tasks 的一次 JOIN）现查得到，不必在事件行/内容行上
	// 各留一份快照——尤其是昵称快照，本站的一贯约定是玩家改名立即在所有历史记录里生效
	// （聊天板块同理），不展示改名前的旧昵称，也就没有"snapshot 防止改名回溯"的需求。
	// 保留到这个版本才删，是因为这几列的列名在更早的迁移里被显式引用过（quarantine 重建、
	// v43 存量搬迁），DDL 提前少列会让那些历史迁移路径直接报错，只能等它们都跑完再删。
	{version: 48, migrate: func(db sqlExecer) error {
		for _, col := range []string{
			"task_source", "formal_series_id", "formal_series_version",
			"contributor_player_id", "contributor_name_snapshot", "contributor_anonymous",
		} {
			if err := dropColumnIfExists(db, "punishment_events", col); err != nil {
				return err
			}
		}
		if err := dropColumnIfExists(db, "sub_tasks", "contributor_name"); err != nil {
			return err
		}
		return dropColumnIfExists(db, "series", "contributor_name")
	}},
}

// rebuildGenderOptionsWithoutNormalizedLabel 把 gender_options 从
// UNIQUE(faction_id, normalized_label) 重建为 UNIQUE(faction_id, label)，去掉冗余的
// normalized_label 列。该列参与表级 UNIQUE 约束，SQLite 的 ALTER TABLE DROP COLUMN
// 不支持直接删除参与约束/索引的列，只能新建目标结构的表、搬数据、原表替换。
func rebuildGenderOptionsWithoutNormalizedLabel(db sqlExecer) error {
	ok, err := tableExists(db, "gender_options")
	if err != nil || !ok {
		return err
	}
	hasCol, err := tableHasColumn(db, "gender_options", "normalized_label")
	if err != nil || !hasCol {
		return err
	}
	if _, err := db.Exec(`
		CREATE TABLE gender_options_v46 (
			id TEXT PRIMARY KEY,
			label TEXT NOT NULL DEFAULT '',
			faction_id TEXT NOT NULL,
			sort_index INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (faction_id) REFERENCES gender_factions(id),
			UNIQUE (faction_id, label)
		)
	`); err != nil {
		return fmt.Errorf("create gender_options_v46: %w", err)
	}
	if _, err := db.Exec(`
		INSERT INTO gender_options_v46 (id, label, faction_id, sort_index, created_at, updated_at)
		SELECT id, label, faction_id, sort_index, created_at, updated_at FROM gender_options
	`); err != nil {
		return fmt.Errorf("copy gender_options rows: %w", err)
	}
	if _, err := db.Exec(`DROP TABLE gender_options`); err != nil {
		return fmt.Errorf("drop old gender_options: %w", err)
	}
	if _, err := db.Exec(`ALTER TABLE gender_options_v46 RENAME TO gender_options`); err != nil {
		return fmt.Errorf("rename gender_options_v46: %w", err)
	}
	return nil
}

// backfillConnectionEventsGeo 用 connection_events.ip 批量解析归属地，回填同一行的
// province/isp（v17 新增，默认空）。只处理 ip 非空、province/isp 仍为空的行，天然幂等。
// 内网/回环 IP 解析结果没有省份/ISP（geoip.Lookup 只给 Country="本地"），不写入。
func backfillConnectionEventsGeo(db sqlExecer) error {
	if !geoip.Enabled() {
		return nil
	}
	ok, err := tableExists(db, "connection_events")
	if err != nil || !ok {
		return err
	}
	rows, err := db.Query(`
		SELECT seq, ip FROM connection_events
		WHERE ip IS NOT NULL AND ip <> ''
		  AND (province IS NULL OR province = '')
		  AND (isp IS NULL OR isp = '')
	`)
	if err != nil {
		return fmt.Errorf("query connection_events for geo backfill: %w", err)
	}
	type pendingRow struct {
		seq int64
		ip  string
	}
	var pending []pendingRow
	for rows.Next() {
		var r pendingRow
		if err := rows.Scan(&r.seq, &r.ip); err != nil {
			rows.Close()
			return fmt.Errorf("scan connection_events for geo backfill: %w", err)
		}
		pending = append(pending, r)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close() // 必须在下面继续用同一个 *sql.Tx 执行 UPDATE 前关闭，否则连接仍被结果集占用。

	for _, r := range pending {
		region := geoip.Lookup(r.ip)
		if region.Province == "" && region.ISP == "" {
			continue
		}
		if _, err := db.Exec(`UPDATE connection_events SET province = ?, isp = ? WHERE seq = ?`,
			region.Province, region.ISP, r.seq); err != nil {
			return fmt.Errorf("backfill connection_events geo seq=%d: %w", r.seq, err)
		}
	}
	return nil
}

// refixConnectionEventsGeo 无条件对所有 ip 非空的历史行重新解析并覆盖 province/isp，
// 用于修正 v18 因字段索引错位写入的错误数据（见 v19 迁移注释）。与
// backfillConnectionEventsGeo 的区别：不按"province/isp 是否已为空"过滤——旧错误值
// 本身就非空，用空值判断会漏掉。geoip 未启用时跳过，不产生空写入（此时也不清空已有
// 的错误数据，等下次 geoip 就绪、库版本仍是 19 但内容有旧数据时不会重跑——这是可接受的
// 权衡，geoip 长期禁用的部署本就不关心 province/isp）。
func refixConnectionEventsGeo(db sqlExecer) error {
	if !geoip.Enabled() {
		return nil
	}
	ok, err := tableExists(db, "connection_events")
	if err != nil || !ok {
		return err
	}
	rows, err := db.Query(`SELECT seq, ip FROM connection_events WHERE ip IS NOT NULL AND ip <> ''`)
	if err != nil {
		return fmt.Errorf("query connection_events for geo refix: %w", err)
	}
	type pendingRow struct {
		seq int64
		ip  string
	}
	var pending []pendingRow
	for rows.Next() {
		var r pendingRow
		if err := rows.Scan(&r.seq, &r.ip); err != nil {
			rows.Close()
			return fmt.Errorf("scan connection_events for geo refix: %w", err)
		}
		pending = append(pending, r)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close() // 必须在下面继续用同一个 *sql.Tx 执行 UPDATE 前关闭，否则连接仍被结果集占用。

	for _, r := range pending {
		region := geoip.Lookup(r.ip)
		if _, err := db.Exec(`UPDATE connection_events SET province = ?, isp = ? WHERE seq = ?`,
			region.Province, region.ISP, r.seq); err != nil {
			return fmt.Errorf("refix connection_events geo seq=%d: %w", r.seq, err)
		}
	}
	return nil
}

// clearWrongAnalyticsGeo 清空 analytics_sessions.province/city/isp、
// analytics_visitors.first_province 里被 parseRegion 字段错位污染的数据，并删掉
// analytics_daily 里按这些错误数据聚合出的 province/city/isp 历史行。这两张表不存
// 原始 IP，没法像 connection_events 那样重新解析，只能清空（见 v20 迁移注释）。
func clearWrongAnalyticsGeo(db sqlExecer) error {
	if ok, err := tableExists(db, "analytics_sessions"); err != nil {
		return err
	} else if ok {
		if _, err := db.Exec(`UPDATE analytics_sessions SET province = '', city = '', isp = ''`); err != nil {
			return fmt.Errorf("clear analytics_sessions geo: %w", err)
		}
	}
	if ok, err := tableExists(db, "analytics_visitors"); err != nil {
		return err
	} else if ok {
		if _, err := db.Exec(`UPDATE analytics_visitors SET first_province = ''`); err != nil {
			return fmt.Errorf("clear analytics_visitors geo: %w", err)
		}
	}
	if ok, err := tableExists(db, "analytics_daily"); err != nil {
		return err
	} else if ok {
		if _, err := db.Exec(`DELETE FROM analytics_daily WHERE metric IN ('province', 'city', 'isp')`); err != nil {
			return fmt.Errorf("clear analytics_daily geo aggregates: %w", err)
		}
	}
	return nil
}

// recomputeAnalyticsFunnelMetrics 按「五层设备 UV」重写 analytics_daily 的 funnel 键：
// 删除旧 login/punish_done 等废弃键，对已有 daily 覆盖的每一天重算 visit/lobby/
// room/round/finish。已 sealed 的历史日不会再被聚合器重算，只能靠这里一次性刷。
//
// 口径与线上聚合器（analytics_agg.go rebuildDay）共用同一个 computeFunnelUV——两边
// 各写一份是 v23~v26 期间反复出现口径漂移的根因，不要再复制粘贴出第二份实现。
func recomputeAnalyticsFunnelMetrics(db sqlExecer) error {
	dailyOK, err := tableExists(db, "analytics_daily")
	if err != nil || !dailyOK {
		return err
	}
	// 必须先枚举日期再删除旧漏斗行：极简历史日可能只剩 funnel 指标，如果先删，
	// 该日会从 DISTINCT day 结果里彻底消失，迁移反而把它永久漏掉。
	dayRows, err := db.Query(`SELECT DISTINCT day FROM analytics_daily ORDER BY day`)
	if err != nil {
		return fmt.Errorf("list analytics days: %w", err)
	}
	var days []int64
	for dayRows.Next() {
		var d int64
		if err := dayRows.Scan(&d); err != nil {
			dayRows.Close()
			return err
		}
		days = append(days, d)
	}
	if err := dayRows.Close(); err != nil {
		return err
	}
	if _, err := db.Exec(`DELETE FROM analytics_daily WHERE metric = ?`, metricFunnel); err != nil {
		return fmt.Errorf("delete old funnel rows: %w", err)
	}
	if len(days) == 0 {
		return nil
	}

	offsetMs := int64(envIntDefault("ANALYTICS_TZ_OFFSET_MIN", 480)) * 60_000

	insert := func(day int64, key string, value int64) error {
		_, err := db.Exec(
			`INSERT INTO analytics_daily (day, metric, key, value, sealed)
			 VALUES (?, ?, ?, ?, 1)
			 ON CONFLICT(day, metric, key) DO UPDATE SET value = excluded.value`,
			day, metricFunnel, key, value,
		)
		return err
	}

	for _, day := range days {
		dayStart := day*86_400_000 - offsetMs
		dayEnd := dayStart + 86_400_000

		funnel, err := computeFunnelUV(db, day, dayStart, dayEnd)
		if err != nil {
			return fmt.Errorf("compute funnel day=%d: %w", day, err)
		}
		for _, key := range funnelStageKeys {
			if err := insert(day, key, funnel[key]); err != nil {
				return fmt.Errorf("insert funnel %s day=%d: %w", key, day, err)
			}
		}
	}
	return nil
}

// fixAnalyticsPunishmentDoneMetric 把 analytics_daily 里错误恒 0 的 punish_done /
// funnel.punish_done 按 punishment_events.status='approved' 重算写回。
// 只 UPDATE 已有 daily 行，不 INSERT 新 day——避免 backfill 因「该日已有任意 metric」
// 而跳过全量 rebuild，留下只有惩罚指标的空壳日。
// 日界与聚合器一致：day = (task_at + ANALYTICS_TZ_OFFSET_MIN*60_000) / 86400000，
// 默认 UTC+8（480）。
func fixAnalyticsPunishmentDoneMetric(db sqlExecer) error {
	dailyOK, err := tableExists(db, "analytics_daily")
	if err != nil || !dailyOK {
		return err
	}
	peOK, err := tableExists(db, "punishment_events")
	if err != nil || !peOK {
		return err
	}
	// 旧隔离表/极简测试夹具可能没有 status 列，跳过以免迁移失败。
	hasStatus, err := tableHasColumn(db, "punishment_events", "status")
	if err != nil || !hasStatus {
		return err
	}
	hasTaskAt, err := tableHasColumn(db, "punishment_events", "task_at")
	if err != nil || !hasTaskAt {
		return err
	}

	offsetMs := int64(envIntDefault("ANALYTICS_TZ_OFFSET_MIN", 480)) * 60_000
	// 相关子查询：无匹配行时 SUM 为 NULL → COALESCE 成 0（该日无 approved 即完成 0）。
	const recompute = `
		UPDATE analytics_daily
		SET value = COALESCE((
			SELECT SUM(CASE WHEN pe.status = 'approved' THEN 1 ELSE 0 END)
			FROM punishment_events pe
			WHERE ((pe.task_at + ?) / 86400000) = analytics_daily.day
		), 0)
		WHERE metric = ? AND key = ?`
	if _, err := db.Exec(recompute, offsetMs, metricPunishDone, ""); err != nil {
		return fmt.Errorf("recompute punish_done: %w", err)
	}
	if _, err := db.Exec(recompute, offsetMs, metricFunnel, "punish_done"); err != nil {
		return fmt.Errorf("recompute funnel.punish_done: %w", err)
	}
	return nil
}

// migrateCustomGenderPlayers 把存量"自定义性别"玩家（gender_id 为空、custom_gender_label
// 非空）按其当前阵营改写成该阵营配置的默认预设性别，然后删除 custom_gender_label 列。
// faction_id 无法识别（阵营已被后台改名/删除等异常情况）时统一兜底成 male_faction/boy。
func migrateCustomGenderPlayers(db sqlExecer) error {
	hasTable, err := tableExists(db, "players")
	if err != nil || !hasTable {
		return err
	}
	hasCol, err := tableHasColumn(db, "players", "custom_gender_label")
	if err != nil || !hasCol {
		return err
	}
	// gender_id/faction_id 本应从表最初创建时就存在，理论上不会缺失；这里仍防御性地跳过
	// 缺列情况（比如测试里手搭的、只覆盖某一版本新增列的极简旧表），保持迁移幂等不 panic。
	hasGenderID, err := tableHasColumn(db, "players", "gender_id")
	if err != nil || !hasGenderID {
		return err
	}
	hasFactionID, err := tableHasColumn(db, "players", "faction_id")
	if err != nil || !hasFactionID {
		return err
	}
	defaultGenderByFaction := map[string]string{
		"male_faction":       "boy",
		"female_faction":     "girl",
		"femboy_faction":     "femboy",
		"trans_male_faction": "ftm",
		"other_faction":      "attack_helicopter",
	}
	for factionID, genderID := range defaultGenderByFaction {
		if _, err := db.Exec(
			`UPDATE players SET gender_id = ? WHERE custom_gender_label <> '' AND gender_id = '' AND faction_id = ?`,
			genderID, factionID,
		); err != nil {
			return fmt.Errorf("migrate custom gender (%s): %w", factionID, err)
		}
	}
	// 兜底：faction_id 不在上面的已知映射里的残留行。
	if _, err := db.Exec(
		`UPDATE players SET gender_id = 'boy', faction_id = 'male_faction' WHERE custom_gender_label <> '' AND gender_id = ''`,
	); err != nil {
		return fmt.Errorf("migrate custom gender (fallback): %w", err)
	}
	return dropColumnIfExists(db, "players", "custom_gender_label")
}

// migrateLegacyRoomEvents 把旧版 rooms/room_join_events 的历史数据转换进 room_events：
//   - rooms 每一行拆成一条 create 事件（user_id/user_name 用创建者信息），closed_at
//     非空时再补一条 close 事件——旧表不记录具体是谁触发的关闭（管理员关闭时也只留了
//     创建者信息），沿用创建者信息占位，与 insertRoomEvent 文档里"empty_cleanup/
//     server_shutdown 没有具体操作者，沿用创建者信息占位"的既有约定一致。
//   - room_join_events 每一行转成一条 join 事件；room_name/game_id 从 rooms 按
//     room_id 关联补齐，关联不到时留空，不阻塞迁移。
//   - 房间密码哈希、创建者 IP 等 room_events 独有的新字段，旧表从未记录，一律留空
//     字符串占位。
//
// 两张旧表均不存在（全新库，或已迁移过）时直接跳过，保证幂等；只有 rooms 存在、
// room_join_events 不存在时，仍会把 rooms 转成 create/close 事件。
func migrateLegacyRoomEvents(db sqlExecer) error {
	roomsExist, err := tableExists(db, "rooms")
	if err != nil || !roomsExist {
		return err
	}
	joinExists, err := tableExists(db, "room_join_events")
	if err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO room_events (at, room_id, room_name, game_id, user_id, user_name, action, role, reason, password_hash, ip)
		SELECT created_at, room_id, room_name, game_id, creator_id, creator_name, 'create', '', '', '', ''
		FROM rooms
	`); err != nil {
		return fmt.Errorf("migrate rooms create events: %w", err)
	}
	if _, err := db.Exec(`
		INSERT INTO room_events (at, room_id, room_name, game_id, user_id, user_name, action, role, reason, password_hash, ip)
		SELECT closed_at, room_id, room_name, game_id, creator_id, creator_name, 'close', '', COALESCE(close_reason, ''), '', ''
		FROM rooms
		WHERE closed_at IS NOT NULL
	`); err != nil {
		return fmt.Errorf("migrate rooms close events: %w", err)
	}

	if joinExists {
		if _, err := db.Exec(`
			INSERT INTO room_events (at, room_id, room_name, game_id, user_id, user_name, action, role, reason, password_hash, ip)
			SELECT j.at, j.room_id, COALESCE(r.room_name, ''), COALESCE(r.game_id, ''), j.player_id, j.player_name, 'join', COALESCE(j.role, ''), '', '', COALESCE(j.ip, '')
			FROM room_join_events j
			LEFT JOIN rooms r ON r.room_id = j.room_id
		`); err != nil {
			return fmt.Errorf("migrate room_join_events: %w", err)
		}
		if _, err := db.Exec(`DROP TABLE room_join_events`); err != nil {
			return fmt.Errorf("drop room_join_events: %w", err)
		}
	}

	if _, err := db.Exec(`DROP TABLE rooms`); err != nil {
		return fmt.Errorf("drop rooms: %w", err)
	}
	return nil
}

// legacyPunishmentRow 是隔离表 punishment_events_legacy 的一行（旧 schema：
// seq/at/kind/source/room_id/player_id/player_name/target_id/task_text/status/
// proof_text/image_file）。
type legacyPunishmentRow struct {
	seq        int64
	at         int64
	kind       string
	source     string
	roomID     string
	playerID   string
	playerName string
	targetID   string
	taskText   string
	status     string
	proofText  string
	imageFile  string
}

// convertLegacyPunishmentEvents 把旧版 punishment_events（一个任务发布/证明提交各占一行，
// 靠 kind 区分、没有任何外键关联）尽量拼回新版 punishment_events（一个任务一行，proof
// 字段原地回填）。旧库里 kind=task 时 player_id 是发布者、target_id 是被罚玩家；
// kind=proof 时 player_id 是提交证明的被罚玩家本人、target_id 恒为空——按
// room_id+被罚玩家+task_text 把 task 行与其后最早一条未匹配的 proof 行配对（先进先出，
// 同一份任务只能靠这三者共同确定，旧库没有任何显式外键）。配对不到 proof 的 task 行
// 视为仍处于 assigned（未完成）状态；配对不到 task 的 proof 行（比如早于任务发布事件
// 开始入库）单独起一行，发布者信息留空。
// legacy 表不存在（没触发过隔离，或已转换过）时直接跳过，保证幂等。
func convertLegacyPunishmentEvents(db sqlExecer) error {
	legacyTable, err := findLegacyPunishmentTable(db)
	if err != nil || legacyTable == "" {
		return err
	}
	rows, err := db.Query(fmt.Sprintf(
		`SELECT seq, at, COALESCE(kind,''), COALESCE(source,''), room_id, COALESCE(player_id,''),
		        COALESCE(player_name,''), COALESCE(target_id,''), COALESCE(task_text,''),
		        COALESCE(status,''), COALESCE(proof_text,''), COALESCE(image_file,'')
		 FROM %q ORDER BY seq`,
		legacyTable,
	))
	if err != nil {
		return fmt.Errorf("read %s: %w", legacyTable, err)
	}
	var legacy []legacyPunishmentRow
	for rows.Next() {
		var r legacyPunishmentRow
		if err := rows.Scan(&r.seq, &r.at, &r.kind, &r.source, &r.roomID, &r.playerID,
			&r.playerName, &r.targetID, &r.taskText, &r.status, &r.proofText, &r.imageFile); err != nil {
			rows.Close()
			return err
		}
		legacy = append(legacy, r)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	type newRow struct {
		id, roomID, taskSource, publisherID, publisherName string
		targetID, targetName, taskText                     string
		taskAt                                             int64
		proofText, imageFile                               string
		proofAt                                            *int64
		hasProof                                           bool
		status                                             string
	}
	byID := map[string]*newRow{}
	var order []string
	// open：每个 (room_id, 被罚玩家, task_text) 组合里，当前还没配到 proof 的 task 行 id。
	open := map[string]string{}
	key := func(roomID, playerID, taskText string) string {
		return roomID + "\x00" + playerID + "\x00" + taskText
	}

	for _, r := range legacy {
		switch r.kind {
		case "task":
			id := fmt.Sprintf("legacy-%d", r.seq)
			nr := &newRow{
				id: id, roomID: r.roomID, taskSource: r.source,
				publisherID: r.playerID, publisherName: r.playerName,
				targetID: r.targetID, taskText: r.taskText, taskAt: r.at,
				status: "assigned",
			}
			byID[id] = nr
			order = append(order, id)
			open[key(r.roomID, r.targetID, r.taskText)] = id
		case "proof":
			k := key(r.roomID, r.playerID, r.taskText)
			if id, ok := open[k]; ok {
				nr := byID[id]
				nr.targetName = r.playerName
				nr.proofText, nr.imageFile = r.proofText, r.imageFile
				at := r.at
				nr.proofAt = &at
				nr.hasProof = true
				nr.status = r.status
				delete(open, k)
				continue
			}
			// 找不到对应 task 行：单独起一行，发布者信息缺失。
			id := fmt.Sprintf("legacy-p%d", r.seq)
			at := r.at
			nr := &newRow{
				id: id, roomID: r.roomID,
				targetID: r.playerID, targetName: r.playerName,
				taskText: r.taskText, taskAt: r.at,
				proofText: r.proofText, imageFile: r.imageFile,
				proofAt: &at, hasProof: true, status: r.status,
			}
			byID[id] = nr
			order = append(order, id)
		}
	}

	for _, id := range order {
		nr := byID[id]
		var proofAt, proofText, imageFile any
		if nr.hasProof {
			proofText, imageFile = nr.proofText, nr.imageFile
			if nr.proofAt != nil {
				proofAt = *nr.proofAt
			}
		}
		if nr.status == "" {
			nr.status = "assigned"
		}
		if _, err := db.Exec(
			`INSERT INTO punishment_events (id, room_id, task_source, approver_id, approver_name, performer_id, performer_name, task_text, task_at, proof_text, image_file, proof_at, status)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			nr.id, nr.roomID, nr.taskSource, nr.publisherID, nr.publisherName,
			nr.targetID, nr.targetName, nr.taskText, nr.taskAt,
			proofText, imageFile, proofAt, nr.status,
		); err != nil {
			return fmt.Errorf("insert converted %s: %w", nr.id, err)
		}
	}

	_, err = db.Exec(fmt.Sprintf(`DROP TABLE %q`, legacyTable))
	return err
}

// findLegacyPunishmentTable 找 execSchemaQuarantiningLegacyTables 隔离出的
// punishment_events_legacy（冲突时会带时间戳后缀）；不存在则返回空字符串。
func findLegacyPunishmentTable(db sqlExecer) (string, error) {
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'punishment_events_legacy%'`)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var name string
	if rows.Next() {
		if err := rows.Scan(&name); err != nil {
			return "", err
		}
	}
	return name, rows.Err()
}

// ensureSchema 是 openDatabase 建表/升级结构的唯一入口：
//  1. 确保 schema_version 表存在，读出当前记录的版本号（表刚建出来、还没写过版本号
//     视为 0）。
//  2. version==0 时，先跑 quarantine：历史遗留库结构无法追溯时，把不兼容的旧表改名隔离。
//  3. 建出/补全所有表（CREATE TABLE/INDEX IF NOT EXISTS）。
//  4. 依次跑 version 之后的 migrations（幂等）。注意：version==0 的旧库也会跑迁移——
//     仅靠 CREATE IF NOT EXISTS 不会给已有 players 表补 highest_score 等新列；
//     跳过迁移会把 schema_version 误标成最新，导致永久缺列、玩家加载失败。
func ensureSchema(db *sql.DB) error {
	if _, err := db.Exec(schemaVersionSchema); err != nil {
		return err
	}
	version, err := readSchemaVersion(db)
	if err != nil {
		return err
	}
	freshBootstrap := version == 0

	for _, schema := range allSchemas {
		if freshBootstrap {
			if err := execSchemaQuarantiningLegacyTables(db, schema); err != nil {
				return err
			}
			continue
		}
		if _, err := db.Exec(schema); err != nil {
			return fmt.Errorf("%s: %w", schema, err)
		}
	}

	// v43 之前的迁移依赖旧任务池/投稿信封表存在；只为尚未走到 v43 的数据库临时建出。
	// v43 会在数据完整搬运后删除它们。已经升级的库绝不能再创建这些死表。
	if version < 43 {
		for _, schema := range legacyMigrationSchemas {
			if freshBootstrap {
				if err := execSchemaQuarantiningLegacyTables(db, schema); err != nil {
					return err
				}
				continue
			}
			if _, err := db.Exec(schema); err != nil {
				return fmt.Errorf("%s: %w", schema, err)
			}
		}
	}

	for _, m := range migrations {
		if m.version <= version {
			continue
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if err := m.migrate(tx); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration v%d: %w", m.version, err)
		}
		if err := writeSchemaVersion(tx, m.version); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		version = m.version
	}

	if version < currentSchemaVersion {
		return writeSchemaVersion(db, currentSchemaVersion)
	}
	return nil
}

func readSchemaVersion(db sqlExecer) (int, error) {
	var version int
	err := db.QueryRow(`SELECT version FROM schema_version LIMIT 1`).Scan(&version)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return version, err
}

func writeSchemaVersion(db sqlExecer, version int) error {
	res, err := db.Exec(`UPDATE schema_version SET version = ?`, version)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	_, err = db.Exec(`INSERT INTO schema_version (version) VALUES (?)`, version)
	return err
}

// tableHasColumn 查 sqlite_master / PRAGMA table_info，判断列是否存在。
func tableHasColumn(db sqlExecer, table, column string) (bool, error) {
	// PRAGMA 不支持占位符绑表名；table/column 仅来自代码常量，不接用户输入。
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(name, column) {
			return true, nil
		}
	}
	return false, rows.Err()
}

func tableExists(db sqlExecer, table string) (bool, error) {
	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func renameColumnIfExists(db sqlExecer, table, from, to string) error {
	ok, err := tableExists(db, table)
	if err != nil || !ok {
		return err
	}
	hasFrom, err := tableHasColumn(db, table, from)
	if err != nil {
		return err
	}
	if !hasFrom {
		return nil
	}
	hasTo, err := tableHasColumn(db, table, to)
	if err != nil {
		return err
	}
	if hasTo {
		// 目标列已在：旧列残留时尽量丢掉旧列，避免双列并存。
		return dropColumnIfExists(db, table, from)
	}
	_, err = db.Exec(fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN %s TO %s`, table, from, to))
	if err != nil {
		return fmt.Errorf("rename %s.%s -> %s: %w", table, from, to, err)
	}
	return nil
}

func dropColumnIfExists(db sqlExecer, table, column string) error {
	ok, err := tableExists(db, table)
	if err != nil || !ok {
		return err
	}
	has, err := tableHasColumn(db, table, column)
	if err != nil || !has {
		return err
	}
	_, err = db.Exec(fmt.Sprintf(`ALTER TABLE %s DROP COLUMN %s`, table, column))
	if err != nil {
		return fmt.Errorf("drop %s.%s: %w", table, column, err)
	}
	return nil
}

func addColumnIfMissing(db sqlExecer, table, column, decl string) error {
	ok, err := tableExists(db, table)
	if err != nil {
		return err
	}
	if !ok {
		// 表还不存在时由 CREATE TABLE IF NOT EXISTS 建出完整结构，这里无需 ADD。
		return nil
	}
	has, err := tableHasColumn(db, table, column)
	if err != nil || has {
		return err
	}
	_, err = db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, column, decl))
	if err != nil {
		return fmt.Errorf("add %s.%s: %w", table, column, err)
	}
	return nil
}

// backfillTotalOnlineMsFromConnectionEvents 用已结束连接的时长之和抬升 players.total_online_ms。
// 幂等：只在汇总大于当前值时更新；缺表/缺列时跳过（全新库或尚无 activity 表）。
func backfillTotalOnlineMsFromConnectionEvents(db sqlExecer) error {
	okPlayers, err := tableExists(db, "players")
	if err != nil || !okPlayers {
		return err
	}
	okEvents, err := tableExists(db, "connection_events")
	if err != nil || !okEvents {
		return err
	}
	hasCol, err := tableHasColumn(db, "players", "total_online_ms")
	if err != nil || !hasCol {
		return err
	}
	_, err = db.Exec(`
		UPDATE players
		SET total_online_ms = (
			SELECT COALESCE(SUM(ce.disconnected_at - ce.connected_at), 0)
			FROM connection_events ce
			WHERE ce.player_id = players.id
			  AND ce.disconnected_at > ce.connected_at
		)
		WHERE total_online_ms < (
			SELECT COALESCE(SUM(ce.disconnected_at - ce.connected_at), 0)
			FROM connection_events ce
			WHERE ce.player_id = players.id
			  AND ce.disconnected_at > ce.connected_at
		)
	`)
	if err != nil {
		return fmt.Errorf("backfill total_online_ms: %w", err)
	}
	return nil
}

// createIndexTableRE 从 "CREATE INDEX IF NOT EXISTS xxx ON <table>(...)" 里取出表名，
// 供下面 execSchemaQuarantiningLegacyTables 判断建索引失败时该隔离哪张表。
var createIndexTableRE = regexp.MustCompile(`(?is)CREATE\s+INDEX\s+IF\s+NOT\s+EXISTS\s+\S+\s+ON\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)

// execSchemaQuarantiningLegacyTables 只在 ensureSchema 判定 version==0（未纳入版本管理
// 的历史库）时使用：逐条执行一段 schema 里的语句；某条 CREATE INDEX 因为列不存在报错，
// 说明该索引所在的表是旧结构（CREATE TABLE IF NOT EXISTS 对已存在的同名旧表是空操作，
// 不会补列），就把整张旧表改名隔离，再回到这段 schema 的开头重新执行。
func execSchemaQuarantiningLegacyTables(db *sql.DB, schema string) error {
	stmts := splitSQLStatements(schema)
	quarantined := map[string]bool{}
	for i := 0; i < len(stmts); i++ {
		stmt := stmts[i]
		if _, err := db.Exec(stmt); err != nil {
			table := createIndexTableRE.FindStringSubmatch(stmt)
			if table == nil || !isMissingColumnErr(err) || quarantined[table[1]] {
				return fmt.Errorf("%s: %w", stmt, err)
			}
			if err := renameLegacyTable(db, table[1]); err != nil {
				return fmt.Errorf("quarantine legacy %s: %w", table[1], err)
			}
			quarantined[table[1]] = true
			i = -1 // 回到这段 schema 的开头，让对应的 CREATE TABLE 重新跑一遍
			continue
		}
	}
	return nil
}

func splitSQLStatements(schema string) []string {
	parts := strings.Split(schema, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p := strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func isMissingColumnErr(err error) bool {
	return strings.Contains(err.Error(), "no such column")
}

// renameLegacyTable 把 table 改名让路给即将重新创建的新结构；名字冲突时追加时间戳。
func renameLegacyTable(db *sql.DB, table string) error {
	legacyName := table + "_legacy"
	var exists string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, legacyName).Scan(&exists)
	switch {
	case err == nil:
		legacyName = fmt.Sprintf("%s_legacy_%d", table, time.Now().Unix())
	case err != sql.ErrNoRows:
		return err
	}
	_, err = db.Exec(fmt.Sprintf(`ALTER TABLE %q RENAME TO %q`, table, legacyName))
	return err
}
