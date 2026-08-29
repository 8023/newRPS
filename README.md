# 抖喵游戏屋

实时联机小游戏平台：**Go 后端** + **Svelte 前端**。

主玩法：锤子剪刀布 / 黑白棋 / 井字棋 / 五子棋 / 斗兽棋 / 国际象棋 / 大话骰 / 猜硬币。含大厅、房间、聊天、观战、排位、惩罚、名字争夺战、白给、极限模式、后台配置等。仅支持真人在线对局（猜硬币例外——单人惩罚小游戏，见下文）。

## 目录结构

```
.
├── cmd/
│   ├── server/          # Go 服务入口
│   └── wsprobe/         # WebSocket 探针（开发用）
├── internal/
│   ├── config/          # 配置加载/校验（config/json/*.json）
│   ├── delta/           # 通用 JSON 增量 Diff/Apply/Hash
│   ├── server/          # 游戏逻辑、HTTP、WebSocket
│   ├── types/           # 服务端领域类型
│   └── wire/            # Protobuf 生成代码
├── api/proto/           # wire.proto 协议定义
├── config/
│   ├── json/            # 按功能拆分的配置 JSON（原地读写，无 active/default 双轨）
│   └── xdb/             # IP 归属地库（gitignore；npm run fetch-geoip）
├── web/                 # 前端（Vite + Svelte 5 + TS）
│   ├── src/
│   │   ├── App.svelte   # 壳：组合根，只接线 store 生命周期与视图路由
│   │   ├── main.ts      # 入口
│   │   ├── lib/         # session、rpc、normalize、format、proofImage、stores/（uiStore/routerStore/sessionStore/adminStore）…
│   │   ├── ui/          # 按页面/关注点拆分的目录：shell/lobby/room/games/contribute/social/profile/admin/about
│   │   ├── ws.ts / wire.ts / delta.ts
│   │   └── shared/types.ts
│   ├── index.html
│   ├── package.json
│   └── vite.config.ts   # 构建输出到 bin/dist/
├── bin/                 # Go 产物目录（gitignore；单一目录，Release 升级只需覆盖它）
│   ├── server           # Go 可执行文件（Linux amd64）
│   └── dist/            # 前端构建产物（Go 静态托管）
├── work/                # db/database.db、uploads、session.secret、analytics.salt（gitignore）
├── go.mod
├── package.json         # 根脚本（并发 dev / 一键 build）
├── docker-compose.yml   # debian:trixie-slim + 挂载产物
├── .env.example
└── README.md
```

### 运行产物（不入库，走 GitHub Release）

**部署不依赖 clone 源码。** 各版本 **GitHub Release** 提供独立部署包：

| 附件 | 内容 |
|------|------|
| `newRPS-<version>-linux-amd64.tar.gz` | `bin/`（含可执行文件 `server` 与前端静态产物 `dist/`）、`docker-compose.yml`、`.env.example`、`config/json/*.json`、`config/xdb/ip2region_v4.xdb`（+ 可选 `ip2region_v6.xdb`）、空 `work/`、简要 `README.md` |

解压后即可 `docker compose up -d`。源码仓库仅用于开发；`bin/` 已 gitignore。

## 本地运行

### 方式一：生产一体（Go 托管前端）

```bash
# 根目录：装前端依赖 + 构建
npm install
npm install --prefix web
npm run build          # web → bin/dist/ 且编译 bin/server

HOST=127.0.0.1 PORT=9988 ./bin/server
```

打开：`http://127.0.0.1:9988`

### 方式二：开发热更新

```bash
# 根目录
npm install
npm install --prefix web
npm run dev            # 同时起 Go(9988) + Vite(5173)
```

或分终端：

```bash
npm run dev:server     # Go
npm run dev:web        # Vite，代理 /api /ws /uploads → 9988
```

- 前端：`http://127.0.0.1:5173`
- 后端：`http://127.0.0.1:9988`

### 方式三：Docker Compose（推荐服务器部署）

**只需 Release 部署包**（不必 clone 源码）。使用官方 **`debian:trixie-slim`**，**无需 Dockerfile**。

```bash
# 下载并解压到任意目录（示例：2.1.24）
gh release download v2.1.24 -p 'newRPS-*-linux-amd64.tar.gz' --repo 8023/newRPS
mkdir -p gamehouse && tar -xzf newRPS-2.1.24-linux-amd64.tar.gz -C gamehouse
cd gamehouse

cp .env.example .env          # 可选：ADMIN_PASSWORD、SESSION_SECRET、ALLOWED_ORIGINS
docker compose up -d

docker compose ps
docker compose logs -f gamehouse
```

无 `gh` 时在 GitHub Release 网页下载附件即可。开发机也可 `npm run build` 后用仓库内 compose 启动。

- 访问：`http://服务器IP:9988`（`HOST_PORT`，默认 9988）
- 挂载：`bin/server`、`bin/dist`（只读）+ `work` / `config`（持久化）
- 停止：`docker compose down`（数据目录保留）

#### 升级服务器且不丢玩家数据

**务必保留**以下路径（不要用新包整目录覆盖掉它们）：

| 路径 | 内容 | 说明 |
|------|------|------|
| `work/db/database.db`（及 `-wal`/`-shm`） | 玩家档案、聊天、房间/惩罚事件、Web Push | **核心存档**；`work/db/players.json` → SQLite 的一次性导入代码已删除（现网存量部署已全部迁移完成），仍停留在 pre-v2.1.28 `players.json` 且从未启动过带迁移代码版本的部署，需先在某个旧版本上启动一次完成迁移，再升级到当前版本 |
| `work/uploads/` | 证明图、后台上传图 | 丢了历史图片链会 404 |
| `work/session.secret` | 会话 HMAC（未设 `SESSION_SECRET` 时） | 丢了则旧浏览器 token 全部失效 |
| `work/vapid.json` | Web Push VAPID 密钥对（未设 `VAPID_*` 时） | **必须保留**；丢失或更换会使所有已有浏览器推送订阅失效 |
| `config/json/*.json` | 后台改过的运行时配置（按功能拆分） | **整目录备份**；升级勿用空包覆盖已改配置 |
| `config/xdb/ip2region_v4.xdb` | IPv4 归属地离线库（用户分析） | **本版本起必需**（除非 `ANALYTICS_GEO_ENABLED=0`）；见下方一次性升级步骤 |
| `config/xdb/ip2region_v6.xdb` | IPv6 归属地离线库（用户分析） | 可选；缺失只是 IPv6 来源访客解析不到归属地，不影响启动 |
| `work/analytics.salt` | 分析访客二次哈希盐 | 丢了则设备访客身份全部重置（设备 DAU/渠道断档），用户留存按 playerId 不受影响 |
| `.env` | `SESSION_SECRET`、`ADMIN_PASSWORD` 等 | **`SESSION_SECRET` 不要换**，否则等同全员掉登录 |

可用新版本覆盖的：`bin/`（含 `dist/`）、`docker-compose.yml`。配置仅在确认需要重置时再覆盖 `config/`。

⚠️ **本版本（用户分析）破坏性升级**：默认启动会加载 `config/xdb/ip2region_v4.xdb`（约 11MB，必需）。旧的升级命令只 `cp -a bin dist`，**不会**带上该文件，升级后进程会直接退出。请在本版本升级时做一次手动步骤：

```bash
# 从 Release 包取出 xdb，或本地拉取：
tmpdir=$(mktemp -d)
tar -xzf newRPS-*-linux-amd64.tar.gz -C "$tmpdir"
mkdir -p config/xdb
cp -a "$tmpdir/config/xdb/ip2region_v4.xdb" config/xdb/   # 若包内已含
cp -a "$tmpdir/config/xdb/ip2region_v6.xdb" config/xdb/ 2>/dev/null || true   # 可选，IPv6 归属地
# 或：npm run fetch-geoip（同时拉 v4 + v6）
# 不想要地域功能时：在 .env 写 ANALYTICS_GEO_ENABLED=0
```

玩家档案存于 `work/db/database.db`（`internal/server/playerstore.go`），启动时经 `loadPlayersFromSQLite`/`ingestPersistedPlayer` 全量加载进内存。`work/db/players.json` → SQLite 的一次性导入代码（`migratePlayersJSONIfNeeded`）与 `PlayerSecretHash` 兼容分支已随现网存量部署完成迁移后一并删除。

⚠️ **`writePlayersJSONFallback`（`internal/server/persist.go`）是 SQLite 不可用/写失败时的保底路径**：库写不进去时兜底写回 `work/db/players.json`，避免彻底丢档；不提供反向的"启动时从这份 JSON 读回"能力（SQLite 是唯一的读路径）。SQLite 持久化（`playerDB.upsertMany`）稳定运行一段时间、确认生产环境没有再触发过这个降级路径后，可以评估是否精简/删除 `writeSnapshot`/`writePlayersJSONFallback` 里的这条兜底分支，改为只记 `errorLog` 不再写 JSON。

⚠️ **改 SQLite 表结构必须同步 bump `internal/server/schema_migrations.go` 的 `currentSchemaVersion`，否则线上旧库不会迁移，读写会用错列**：`work/db/database.db` 里有张 `schema_version` 表记录当前结构版本，`openDatabase` 每次启动都会跟代码里的 `currentSchemaVersion` 比对——一致就跳过，不一致就依次执行 `migrations` 里对应版本号的显式迁移（`ALTER TABLE ADD/RENAME COLUMN`、建新表倒数据等）。改动流程：① 把 `internal/server/*.go` 里对应的 `xxxSchema` 常量改成目标结构；② 在 `migrations` 追加一条 `{version: currentSchemaVersion+1, migrate: ...}`，用真正的 SQL 把旧数据搬到新结构；③ 把 `currentSchemaVersion` 加一。只改①不做②③，等于新代码按新结构读写字段，但已经建过表的旧库还停在旧结构，轻则报错重则悄悄错位。`version==0`（全新库，或本机制引入之前就存在、结构在代码里已无法追溯的历史遗留库）时有一次性的"某条 `CREATE INDEX` 因为列不存在报错 → 把该表整体改名隔离为 `<表名>_legacy`"兜底，只用于应付"完全够不到历史"的场景（比如 `punishment_events` 曾经用过 `kind`/`source`/`player_id`/`at` 这套更早的列名），**不能**当成常规迁移手段来偷懒——它只能处理"缺列导致建索引失败"，处理不了删列/改列名（旧列会悄悄留在表里没人管）。`punishment_events` 隔离出的 `_legacy` 表不会就此撂着：v4 迁移（`convertLegacyPunishmentEvents`）按 `room_id`+被罚玩家+`task_text` 把旧版"发布"/"提交证明"两行拼回新版一行一任务的结构，尽量不丢历史，转换完即丢弃 `_legacy` 表。

```bash
# 备份数据（推荐）
tar czf backup-$(date +%F).tgz work config .env

# 解压新包到临时目录，覆盖程序（保留 work/config/.env）
tmpdir=$(mktemp -d)
tar -xzf newRPS-2.1.24-linux-amd64.tar.gz -C "$tmpdir"
cp -a "$tmpdir/bin" "$tmpdir/docker-compose.yml" .
rm -rf "$tmpdir"

# bin/server 是 bind mount；程序与前端同时升级时强制重建容器，避免继续持有旧二进制。
docker compose up -d --force-recreate
# 应返回 publicKey 和 protocolVersion；缺少 protocolVersion 说明仍在运行旧后端。
curl -fsS http://127.0.0.1:${HOST_PORT:-9988}/api/push/vapid-key
```

> 说明：Release 中的 `bin/server` 为 **Linux amd64**。ARM 服务器需在对应架构上重新 `npm run build:server`。

## WebSocket 协议

**全链路 Protobuf**（`api/proto/wire.proto` + `api/proto/game.proto`），**线路上无 JSON 文本载荷**。

| 层 | 格式 |
|----|------|
| 帧 | `Envelope`（event / id / kind / channel / seq / hash） |
| FULL | 类型化 `StateDocument`（lobby / room / config） |
| RAW | 类型化 `RawBody` 或 `google.protobuf.Struct`（动态 RPC 入参/出参，仍是 protobuf 二进制） |

> 历史说明：曾有「Protobuf 信封 + 内层 JSON」阶段；现为类型化 FULL + 路径 DELTA；状态一致性用 CRC-32 探针（非 SHA-256）。

### 压缩（UA 分流）

| 客户端 | permessage-deflate | 说明 |
|--------|-------------------|------|
| Safari / iOS（含 iOS 上的 Chrome） | **关闭** | 扩展协商后易出现 `network connection was lost` |
| Chrome / Firefox / Edge 等 | **开启**（ContextTakeover） | 省流量；与应用层 protobuf 正交 |

反代须允许 WebSocket Upgrade；压缩由浏览器与 Go 服务端协商，反代一般不用改。

### 消息类型

| 类型 | 说明 |
|------|------|
| FULL | 状态通道首包 / 补丁过大 / resync；类型化 `StateDocument` |
| DELTA | 路径补丁（`PatchOp` + `google.protobuf.Value`）+ **合并后树 CRC-32**；不一致则 `sync:full` |
| RAW | RPC 与即时推送（chat、player:batch 等，不走状态 diff） |
| `sync:full` | 客户端校验失败或本地无基线时请求全量 |
| `analytics:collect` | 前端埋点批量上报（无 ack；服务端成功时不 reply） |
| `admin:analytics` / `admin:analyticsDetail` | 后台数据分析快照 / 明细（管理员） |
| `contribution:list` / `get` / `saveDraft` / `submit` / `withdraw` | 玩家共建投稿的查询、草稿、送审、撤回 RPC；撤回是玩家自助直接生效（不再有「申请下架」两步流程），动态载荷仍封装在 Protobuf `Struct` 中 |
| `contribution:votePreview` / `vote` | 正式投稿任务的评价预览与赞踩；未投票前只返回资格和任务标识，成功投票后才返回点赞率、票数与贡献者 |
| `admin:action` 的 `contribution*` | 共建审核（批准/驳回/下架/批注/取消撤回）；仅管理员可调用 |
| `admin:action` 的 `genders*`（`gendersGet`/`gendersSave`） | 「性别与阵营」批量增删改；与共建投稿系统无关，只是复用同一个后台 action 分发入口 |

状态校验：对「前端形态」规范化树做 **CRC-32（IEEE）**（8 位 hex，非密码学；两端对齐）。`player:batch` / 房间广播 debounce 等合并策略不变。
| `player:get` | 拉取完整 `PublicPlayer`（大厅仅下发精简 `LobbyPlayer`） |

代码生成：`protoc` → `internal/wire/*.pb.go`；前端 `pbjs` → `web/src/gen/proto.js`。

## HTTP API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/session` | 签发会话 token |
| POST | `/api/proof-image` | 证明图：仅 webp，≤2MB |
| POST | `/api/admin-image` | 后台图：客户端统一压缩为 WebP，≤2MB |
| POST | `/api/contribution-image` | 共建投稿封面：需登录持久身份，WebP ≤2MB |
| GET | `/ws` | WebSocket |
| GET | `/uploads/*` | 上传文件（证明图/头像/后台图/共建封面）。要求 `Referer` 为本站页面（与 `ALLOWED_ORIGINS`/本机开发同一套判断），缺失或跨站一律 404，见下方说明 |
| GET | `/*` | 前端静态（`bin/dist/`） |

### 上传图片

- 前端：长宽比 >21:9 拒绝；原图 >10MB 拒绝；>4MP 缩放；WebP 85%
- 后端：非 `.webp` 拒绝；>2MB 拒绝；错误信息用户可见
- 浏览器/共享缓存：`/uploads/proofs/` 为 6 小时；`/uploads/contributions/` 共建背景图为 30 天（`max-age=2592000`）。替换图片必须换新 URL。
- 服务端留存：共建图片一旦上传成功就作为监管审计材料永久保留在磁盘；上传接口不再单独登记图片表，只有被某份投稿最新版本的 `background_image` 引用时才能访问。从未绑定投稿或后来被换下的文件仍保留，但访问会返回 404。30 天只是浏览器/共享缓存寿命，不是服务器保留期；如需清理，只能由站外专用审计清理程序另行执行。
- 访问侧：文件名本身不可猜测（随机后缀），但这不是身份鉴权；`GET /uploads/*` 额外校验
  `Referer` 必须来自本站页面（与 WS/POST 的 Origin 校验同一套白名单逻辑），拦住跨站
  热链、以及把链接粘到站外浏览器地址栏直接打开这类场景。这不是强鉴权——非浏览器客户端
  可以随意伪造 `Referer`，只作为纵深防御的一层；共建图又会永久留存，因此文件名泄露后仍
  可能被长期访问，上传和分享链接时请自行注意。
  - ⚠️ 这层校验只在请求真正打到 Go 进程时才会执行——若响应被标成可共享缓存（`public`），
    请求路径上的 CDN/反代等共享缓存会按 URL 缓存响应体、不区分 `Referer`：上传者本人在
    房间里用合法 `Referer` 查看一次，就会把图片写进共享缓存，之后任何人拿着同一个 URL、
    不带 `Referer` 也会被缓存直接命中放行，校验被绕过。本站是非盈利小站，流量成本优先于
    这点残留风险，因此 `Cache-Control` 仍是 `public`。证明图的 `max-age` 已从最初的 30 天
    收窄到 6 小时（多数房间的存活时长量级）；共建图按审计留存场景维持 30 天缓存，所以
    同一 URL 被共享缓存命中而绕过回源 Referer 校验的窗口也可能持续 30 天；
    如果更看重防泄露、能接受全部回源的流量成本，把它改成 `private` 即可让这层校验对
    每次请求都真正生效。
  - 证明图另有一层"房间已销毁则一律拒绝"的校验：上传时记录的所属房间（内存态、不落盘）
    一旦被清理（房间清空/管理员关房/进程重启），无论 `Referer` 是否合法都直接 404。这层
    只在请求真正回源时执行，6 小时内命中共享缓存的请求仍绕得开，与上面的 `Referer` 校验
    互补而非互相替代。
  - 共建图另有一层"孤儿图片一律拒绝"的校验：只要这张图当前仍是它所属投稿 id **最新版本**
    引用的封面就放行（草稿/待审/已通过/被驳回/已撤回，状态完全不影响——没有单独的图片表，
    封面图就是投稿内容表一行上的字段）；只有重新上传另一张图覆盖替换掉之后，旧图不再是
    任何 id 最新版本引用的图，才 404，文件本身永远不硬删。同样只在请求真正回源时执行，
    30 天内命中共享缓存的请求仍绕得开。

静态 hash 资源：`Cache-Control: public, max-age=31536000`

## 环境变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `HOST` | `0.0.0.0` | 监听地址 |
| `PORT` | `9988` | 监听端口 |
| `ADMIN_PASSWORD` | （空） | 后台口令 |
| `SESSION_SECRET` | `work/session.secret` | 会话 HMAC；未设置则落盘复用 |
| `VAPID_PUBLIC_KEY` / `VAPID_PRIVATE_KEY` | `work/vapid.json` | Web Push 密钥对；环境变量必须成对设置，多实例必须共享同一对 |
| `VAPID_SUBSCRIBER` | `admin@rps.rbq.io` | VAPID 联系邮箱，直接填邮箱地址、不要带 `mailto:` 前缀（服务端会自动补上，重复带前缀在 Safari/iOS 走的 Apple 推送网关上会被 403 拒绝）；也可以填 `https://` 开头的地址 |
| `SESSION_TTL_MS` | 24h | 会话有效期 |
| `ALLOWED_ORIGINS` | 本机 | 额外 Origin |
| `TRUSTED_PROXY_COUNT` | `0` | 可信反向代理层数，决定 `X-Forwarded-For`/`X-Forwarded-Host` 信任方式；默认 `0`=直连。部署在反向代理之后（Nginx/Caddy/云 LB 等）必须显式设为实际代理层数，否则会退化为按代理自身 IP 计算（限流过严但不会被伪造） |
| `MAX_SOCKETS_PER_IP` | 按防多开人数上限 ×4（至少 12） | 单设备（IP+指纹）WebSocket 套接字上限 |
| `LOBBY_BROADCAST_DELAY_MS` | 300 | 大厅广播合并 |
| `ROOM_BROADCAST_DELAY_MS` | 100 | 房间广播合并 |
| `ANALYTICS_ENABLED` | `1` | `0` 关闭用户分析采集与聚合器 |
| `ANALYTICS_GEO_ENABLED` | `1` | `0` 不载入 IP 库、不做归属地解析（也不做启动时文件检查） |
| `GEOIP_DB_PATH` | `config/xdb/ip2region_v4.xdb` | IPv4 归属地 xdb 路径（必需，除非关闭 `ANALYTICS_GEO_ENABLED`） |
| `GEOIP_DB_PATH_V6` | `config/xdb/ip2region_v6.xdb` | IPv6 归属地 xdb 路径（可选，缺失只是 IPv6 来源访客解析不到归属地） |
| `ANALYTICS_TZ_OFFSET_MIN` | `480` | 分析日切时区偏移（分钟）；默认 UTC+8，否则「今天」会在北京时间早 8 点才切日 |
| `ANALYTICS_RAW_RETENTION_DAYS` | `90` | `analytics_events` 原始事件保留天数（会话 180 天；日聚合永久） |

### 防多开（IP + 浏览器指纹）

中国公网出口常被整栋楼/公司共用，**不能只按 IP 限流**。

- 前端用 [FingerprintJS](https://github.com/fingerprintjs/fingerprintjs) 生成 `visitorId`
- 服务端 `deviceKey = sha256(ip + "\\0" + fingerprint)`
- 配置项（字段名兼容旧版）：
  - `accessControl.maxOnlinePerIp` → **同指纹同时在线人数上限**
  - `accessControl.maxCreatesPer10Min` → **同指纹 10 分钟内新建玩家上限**
- 上报路径：`POST /api/session`（Header/Body）、WebSocket 握手的 `Sec-WebSocket-Protocol` 头（`fp.<base64url 指纹>`，与 `auth.<token>` 一起传递，不再拼进 `/ws` 查询串——反向代理访问日志会记录完整请求 URL，会话 token 出现在其中就等于把凭据写进了日志）、`player:join.fingerprint`

**纯 IP 兜底（防批量脚本攻击）**：`fingerprint` 是客户端上报的字符串，服务端不校验真实性——攻击脚本只要每次请求都随机换一个指纹，就能让上面按 `deviceKey` 计算的限制失效；`sid` 同理，每调一次 `/api/session` 就能免费换发一个新的，导致所有按 `event:ip:sid` 维度的 WS 事件限流也能被"换 sid"重置。因此在按指纹/按会话限流之外，又加了一层完全不看指纹、只看出口 IP 的兜底限制（同样可在 `/admin` → 网站管理 → 限流策略调整）：

- `accessControl.ipBackstopMultiplier` / `ipBackstopMinLimit` —— 每个 WS 事件（`room:create`、`room:move`、`punishment:submit` 等）除了原有的 `event:ip:sid` 限流桶，还并行检查一个 `event:ip` 粗粒度桶，阈值为 `max(该事件 per-sid 阈值 × 倍数, 最低下限)`；不管客户端怎么换 sid，同一 IP 在同一窗口内的总请求量都会被这层桶钉住
- `accessControl.maxSessionIssuePerIp` —— `POST /api/session` 纯 IP 维度的 10 分钟签发上限（在原有按 `devKey` 的限流之外并行生效）
- `accessControl.maxOnlinePerIpTotal` / `maxCreatesPerIp` —— `player:join` 纯 IP 维度的同时在线人数 / 10 分钟新建人数上限（并行于原有按 `deviceKey` 的检查）
- `accessControl.maxActiveRoomsPerOwner` —— 单个玩家同时开着（未关闭）的房间数量上限，堵住"创建频率虽被限流、但攒着不关"的房间数量爆炸
- `accessControl.maxProofUploadsPerPlayer` —— 惩罚任务证明图片按玩家维度的 10 分钟上传上限（并行于路由层已有的纯 IP 限流）

以上这层"纯 IP"防护的前提是 `TRUSTED_PROXY_COUNT` 与实际部署拓扑一致——直连公网必须设为 `0`，否则 `X-Forwarded-For` 可被伪造，所有基于 IP 的限制都形同虚设。

**一键止血开关**：`accessControl.registrationDisabled`（`/admin` → 网站管理 → 提示公告 → 「新用户注册开关」）勾选后，`player:join` 里 `player == nil`（即将新建身份）的分支会直接拒绝，提示"当前暂停新用户注册，请使用已有账号登录"；已持有 `playerId`+`playerSecret` 的老用户走的是 `player != nil` 分支，不受影响，仍可正常登录游玩。用于遭遇批量注册攻击时先整体止血，再人工排查、用后台单独封禁具体的恶意老账号。

## 后台与配置文件

- 入口：`/admin` 或 `#admin`，或 `Ctrl/Cmd+Shift+A`
- 配置在 `config/json/` 下**按功能拆分、原地读写**（无 active/default 双轨）。旧版单体 `default.json`/`active.json` 启动时会自动迁移并改名为 `*.bak`。

| 文件（相对 `config/json/`） | 内容 |
|------|------|
| `site.json` | 站点名、简介、管理员口令 |
| `announcement-board.json` | 公告板（展示于顶栏「关于」面板，不再是弹窗） |
| `security-disclaimer.json` | 安全与免责声明开关（内容固定写在前端，仅一个 `enabled`） |
| `gender-factions.json` | 阵营种子（配色、`taskGroup`）。库空时首次导入 SQLite，之后不再作为运行时权威，后台保存不写回 |
| `genders.json` | 性别种子。库空时首次导入 SQLite，之后不再作为运行时权威，后台保存不写回 |
| `titles.json` | 称号段 |
| `punishments.json` | 惩罚配置：`tags`（标签，含房名词库/背景图库）+ 全局 `orderStep`/`maxDifficultyOvershoot`/`minSeriesSteps`（系列投稿最低步数，默认 5）/`maxSeriesSteps`（管理员可配置的系列投稿最高步数，默认 20）。最低/最高值均不得超过 1000，最高值不得小于最低值，后台保存越界配置会明确报错；1000 同时是后端拦截畸形 payload 的硬上限。任务池与系列任务详情落在 SQLite 的 `sub_tasks` / `series`（全量版本化、插入不更新，见 CLAUDE.md「共建投稿」一节），且新内容只能通过玩家投稿 + 管理员审批（`contribution:*` / `admin:action` 的 `contribution*` 系列）产生；管理员可在共建审核详情中修订原稿并立即发布，但后台不提供绕过投稿记录、直接操作正式任务池/系列表的入口。旧文件里的 `tasks`/`seriesTasks` 启动时一次性导入后不再读写 |
| `player-punishment-room-name-pool.json` | 玩家发布任务房名词库 |
| `room-tags.json` / `room-info-tags.json` | 房间 Tag 与信息标签样式 |
| `title-tag-styles.json` | 称号标签按赋予来源（系统默认/自定义/主人赋予/管理员赋予）的配色 |
| `access-control.json` | 多开与限流策略（后台：网站管理 → 限流策略） |
| `name-war.json` / `giveaway.json` / `extreme-mode.json` | 名争 / 白给 / 极限模式文案与参数 |
| `ranked-score.json` | 排位分「展示」上下限（含名字争夺战下限）与每日衰减比例；存储分数本身不设上下限，仅展示时封顶，详见「排位积分」章节 |
| `pet-bond.json` | 宠物乐园（认主/认宠）面板标题、主人/宠物数量上限与称号长度 |
| `games.json` / `messages.json` | 游戏列表（含各游戏排位赌分档位 `stakes`，黑白棋按每子、其余按整局结算，用于平衡各游戏耗时）、提示文案 |
| `../xdb/ip2region_v4.xdb` | IPv4 归属地离线数据，不在本目录（必需；不入 git；`npm run fetch-geoip` 或 Release 包内附带） |
| `../xdb/ip2region_v6.xdb` | IPv6 归属地离线数据，不在本目录（可选，缺失不影响启动；不入 git） |

后台「保存」会写回对应 JSON；服务启动时会把 `config/json/*.json` 权限收紧为 `0600`（仅运行用户可读写，防同机其他用户读取其中的管理员口令），`bin/server` 设为可执行。

## 用户分析

后台「数据分析」分区展示访问、**用户留存**（按 `playerId` 首次注册日 cohort，跨设备合并）、设备渠道、游戏与站点玩法趋势；设备→注册→进房等转化看「转化漏斗」。架构三层解耦，**RPC 路径上一句 SQL 都没有**：

1. **独立只读连接**（`mode=ro`，WAL 下读不阻塞写）跑聚合查询；
2. **后台协程 + `analytics_daily` 日聚合表**（已封板的日子永不重算，每分钟只重算今天/昨天）；
3. **`atomic.Pointer` 内存快照**：`admin:analytics` 只做指针 Load + 切片。

采集双路：前端埋点（页面/会话/来源，`web/src/lib/analytics.ts`）+ 既有审计表（`connection_events` / `room_events` / `punishment_events` / `player_activity_events`）与服务端 `game_round` 事件。历史审计表首次启动会全量回填进日聚合，面板上线即有趋势曲线。

**IP 归属地**：使用 [ip2region](https://github.com/lionsoul2014/ip2region) 离线库，IPv4 库必需，放在 **`config/xdb/ip2region_v4.xdb`**；IPv6 库可选，放在 **`config/xdb/ip2region_v6.xdb`**（缺失时 IPv6 来源的访客只是解析不到归属地，不影响启动）；解析时按访客 IP 的地址族自动选库，两者都不嵌二进制、不入 git。本地开发：

```bash
npm run fetch-geoip   # 同时下载 config/xdb/ip2region_v4.xdb 与 ip2region_v6.xdb
```

更新 xdb：重新执行 `fetch-geoip` 或从上游 release 覆盖同路径后重启进程。

**隐私**：分析表不存 IP/指纹/原始 UA；访客 id 为 `sha256(deviceKey ‖ salt)` 前 16 hex；地域/来源展示做 k-匿名（访客数 &lt; 3 折进「其他」）；面板只显示省份 + ISP。

**总开关**：`ANALYTICS_ENABLED=0` 关闭采集与聚合；`ANALYTICS_GEO_ENABLED=0` 关闭归属地（不载入 11MB xdb）。

## 多端身份认领

没有传统用户名密码：每台浏览器首次访问时在本地生成一对 `playerId` + `playerSecret`（长随机串），存 `localStorage`，此后 `player:join` 每次都带上这对凭据重连。战绩/积分只跟 `playerId` 走。

- **认领新设备**：个人资料页展示一个一次性「认领密钥」（`ClaimKey`，与长期 `playerSecret` 是两个不同的值），复制到另一台设备的输入框提交（`identity:claim`），服务端校验通过后给新设备签发一份全新的 `playerSecret` 并**立即轮换掉这把密钥**（用过即作废）；旧设备完全不受影响。
- **多端同时"记住"，但同一时刻只有一端在线**：一个身份最多同时记住 3 台设备的凭据（`PlayerSecrets`，超出后挤掉最早一条，绝不挤当前活跃会话），但服务端仍是单 socket 模型——已有设备在线时，新设备登录会先收到 `alreadyOnline` 提示确认是否顶替，确认后走 `forceKick`，被顶替端会收到 `session:kicked` 事件（同设备刷新重连不受影响，不会误触发确认）。
- **登出**：`identity:logout` 撤销当前设备的那条 `playerSecret`，前端随后清空本地 `localStorage`。

身份凭据统一存明文 `PlayerSecrets` 列表（`verifyPlayerSecret` 只查这一份）。早期版本用单值哈希（`PlayerState.PlayerSecretHash`，`hashSecret`）存凭据的兼容分支已删除；当时仍停留在旧哈希格式、从未在删除前重连过的账号，其明文 secret 从未完成自动迁移，现已无法再用老设备身份登录（档案本身仍保留，见 `internal/server/schema_migrations.go` v6 迁移）。

## 大话骰（Liar's Dice）

第四种游戏类型，与 RPS/黑白棋/井字棋同级，但刻意不复用它们共用的 `Seats`/`SeatKey`（固定两人）模型——大话骰是 2-8 人，独立走 `RoomState.LiarsDice`（公开状态，进房间快照）+ `RoomState.LiarsDiceHands`（私有骰子，只通过 `emitToClient` 单播给玩家自己，从不广播，也不加密——保密性来自"这份字节压根不会出现在其他玩家的连接上"）。核心逻辑集中在 `internal/server/game_liarsdice.go`。

- **入席**：进房间默认观战，对局未开始前可自由 `liarsdice:joinRoster` / `liarsdice:leaveRoster`；房间设置里 `liarsDiceMinPlayers`（2~上限，默认 3）/`liarsDiceMaxPlayers`（默认 3，上限 8）。
- **开局**：参战名单全员 `liarsdice:ready` 且名单 5 秒无变动 → 每人现摇 5 颗骰子，随机选首个叫点者（按入席顺序循环叫点）。
- **叫点规则**：叫点（`liarsdice:bid`，颗数更多，或颗数不变但点数更大）按入席顺序循环，只有 `CurrentTurn` 指向的玩家能叫，没有"过"。开牌（`liarsdice:challenge`，质疑当前叫点）不受回合顺序限制——只要 `room.Phase == PhaseChoosing` 且存在 `CurrentBid`，任意在场参战玩家（`ParticipantIDs` 里任意一个，不要求等于 `CurrentTurn`）随时都能发起，不推进 `CurrentTurn`、直接结算并进入 `PhaseResult`。第一个叫点数至少是"在场人数 + 1"。1 是万能点，但只要本局有人喊过 1，之后 1 不再算万能，仅算实际点数。
- **结算**：开牌揭晓全体参战玩家骰子（不只叫点者和质疑者），按叫点面值（含万能 1，若未禁用）计数；成立则叫点者胜、质疑者负，反之相反；其余参战玩家本局"平"，不计分不受罚。下一局重新摇骰，不延续骰子数量、不淘汰。
- **断线判负**：断线判负的规则是"上家"（入席顺序里的前一位，固定关系，与谁最后叫过点无关）胜——`createLiarsDiceDisconnectForfeit`/`applyLiarsDiceDisconnectForfeit`，与其它三个游戏的 `DisconnectForfeit` 走独立的 `LiarsDiceDisconnectForfeit`（字段是 playerID 而非 SeatKey）。
- **惩罚**：`punishment.go` 的 `setupPunishmentForPlayers` 从原来 Seat/RoundResult 耦合的 `setupPunishmentOrNext` 里抽出通用尾段（按 playerID 列表工作），大话骰和其它三个游戏共用这一段；`buildLiarsDicePunishmentTasks` 单独实现（赢家直接作为"玩家发布任务"模式下的任务发布人，不走 Seat 反查）。

## 猜硬币（Coin Flip）

唯一不需要联机对手的惩罚小游戏，核心逻辑集中在 `internal/server/game_coinflip.go`。只用战斗席 A（`Seats[B]` 永远锁死），建房自动坐上座位 A、无需 ready-up；观战者可以在座位 A 空出来后接棒坐下继续玩，房间照常出现在大厅列表和后台房间管理里。

- **抛掷**：`coinflip:guess` 一次 RPC 完成"选面 + 服务端随机开出结果 + 结算"，没有像 RPS 那样的两阶段出拳/揭示流程；`RoomState.CoinFlip` 存当前一次抛掷的展示态（猜测/结果/是否猜中/落定时间），落定时间只用于前端本地重放 1 秒翻面动画，服务端不等这段时间。
- **猜错即罚**：由于 `Seats[B]` 恒为空，`humanOpponent`/`punishmentReviewer` 天然返回 `nil`——`onPunishmentSubmit` 里 `approvedBySystem := reviewer == nil || ...` 因此自动短路成 `true`，猜硬币不需要任何额外代码就做到"提交证明立即通过"；同理 `createDisconnectForfeit` 也因为对手座位为空而天然跳过，没有断线判负。任务结算直接调用 `buildPunishmentTasksWithWinnerName`（与大话骰同构，不走 Seat/RoundResult 耦合的 `buildMatchHistoryShell`），`{winner}` 占位符固定替换成 `AppConfig.Site.CoinFlipWinnerLabel`（管理员可在后台「站点」板块自定义，默认"系统"）。
- **结构性限制**：建房时强制 `enablePunishment=true`、结构性关闭排位/倍率/极限模式/平局双罚/需对手确认/每子惩罚；`punishmentSource` 只允许 `random`/`series`（没有真人对手，"玩家发布任务"没有意义，选了会被后端拒绝）。
- **不计分**：不调用 `recordGameOutcome`/`applySeatOutcome`/任何排位分结算——胜负场次、`GameStats`、排位积分、白给值都不受影响；受罚次数（`PublicStats.Punishments`）与随机任务难度进度（`RoomState.PunishmentTaskProgress`）复用 `setupPunishmentForPlayers`/`pickSystemTaskForPlayerAdvancing` 的默认行为照常累加/递增，与其它游戏一致。

## 排位积分

`PlayerState.Stats.RankedPoints`（`internal/server/player.go`）在数据库/内存中**永远不设上下限**——胜负结算（`updateRankedPoints`）、管理员手动改分（`setRankedPointsByAdmin`）、以及下面的每日衰减，全部直接对存储值做加减，从不 clamp。`config/json/ranked-score.json`（`types.RankedScoreConfig`：`max`/`min`/`nameWarMin`/`dailyDecayRatio`，后台「排位分设置」可调）只在**下发展示**时生效：`internal/server/player.go` 的 `publicPlayer()` 是所有出站玩家快照（大厅、房间座位、`player:get`、观战列表等）唯一的组装入口，会把真实分数的一份副本按 `max`/`min`（开启「名字争夺战」的玩家用 `nameWarMin` 代替 `min`）夹紧后再下发，真实存储值不受影响。

- **后台调低上/下限**：已经"超范围"的老用户分数在数据库里原样保留，只在前端展示时被新的上/下限封顶；之后正常输赢分或每日衰减，仍然直接对真实（可能超范围的）存储值结算，不会被这次展示层的调整拖拽。
- **称号分段**：`titleSegmentFor`（`internal/server/player.go`）在真实分数落在所有称号分段范围之外时，会夹到最近的边界分段（而不是固定回退到最低档），因此称号池（`config/json/titles.json`）不需要跟着 `ranked-score.json` 的范围同步扩大。
- **每日衰减**：`scheduleRankedDailyDecay`/`applyRankedDailyDecay`（`internal/server/player.go`）每 24 小时（对齐到 UTC 天边界，`time.AfterFunc` 定位到下一个边界后切换为 `time.Ticker`，与「极限模式」整点衰减是完全独立的两套机制）把每个玩家的真实 `RankedPoints` 乘以 `dailyDecayRatio`（默认 `0.98`）并向 0 截断小数——正负分都会朝 0 方向收缩。每个玩家用 `RankedLastDecayDay` 记录已衰减到的"天桶"，防止服务重启后重复衰减。
- **历史最高/最低分**：`recordRankedExtremes` 持续记录真实极值（存储永不回退）；下发展示时与当前分一样按 `max`/`min`/`nameWarMin` 封顶。管理员后台保留按真实分排序的能力；普通玩家收到的公开快照会把 `sortRankedPoints`/`sortHighestScore`/`sortLowestScore` 一并按展示值封顶，避免通过未渲染字段反推出真实分。
- **称号分段用百分比**：`config/json/titles.json` 的 `minPercent`/`maxPercent`（-100～100）相对 `ranked-score.json` 的展示上下限换算真实分所属段；改展示上下限无需改称号绝对分。极限模式的 pos/neg 系数表与同一百分比刻度对齐。
- **管理员自定义称号**：后台「玩家管理」可直接给某个玩家填一个不在 `titles.json` 池里的称号（`editPlayer` action，`internal/server/handlers_room.go`），此时会置位 `PublicStats.TitleCustom`；`syncTitleForRankSegment`（`internal/server/player.go`）一旦发现该标记就直接跳过重算，不再随排位分升降、跨档、改性别/阵营、后台调整 `ranked-score.json` 的 `max`/`min` 而被自动改写。把后台称号输入框清空并保存会清掉该标记、立即按当前排位分重算回自动称号。前端输入框此时会用黄色边框区分，`web/src/ui/admin/AdminPlayerEditor.svelte`。
- 「名字争夺战」失格线：`config/json/name-war.json` 的 `penaltyThreshold`（默认 `-4999`，后台可调），按**真实存储分**判定，与展示封顶无关。改名所需最低分同样在该文件里，`renameMinPoints`（默认 `500`，后台可调），只有真实分达到此值的玩家才能给失格者改名。

## 构建与测试

```bash
npm run build:web      # 仅前端
npm run build:server   # 仅 Go（并 chmod +x bin/server）
npm run fix-perms      # config/json/*.json 收紧为仅属主可读写（0600）+ bin/server 可执行
npm run build          # 先 server、后 web，再 fix-perms；后端失败时不会留下“新前端 + 旧后端”
go test ./...
npm run test           # go test + 前端 build
```

## 最近更新记录

- **新增猜硬币**：唯一不需要联机对手的惩罚小游戏，见上文「猜硬币（Coin Flip）」一节。只用战斗席 A，观战者可接棒坐下，房间照常出现在大厅与后台房间管理；猜错立即进惩罚阶段，没有真人对手，提交证明自动通过；不计入排位积分、胜负场次或白给值，受罚次数与随机任务难度进度照常累加/递增。协议新增 `CoinFlipState`（`RoomSnapshot.coin_flip`）、`RoundHistoryItem.coin_flip_guess`/`coin_flip_result`、`SiteConfig.coin_flip_winner_label`（前后端需同步发布）。
- **公开积分字段收紧**：普通玩家的大厅、房间、资料和全站排行榜快照不再携带未封顶真实积分；管理员玩家管理、关联账号和关系图谱等专属回执仍保留真实分排序字段。名字争夺战的失格判断也直接读取服务端真实积分，展示封顶不会改变业务判定。
- **共建列表与排行榜体验优化**：玩家端和后台共建审核列表加入标题/投稿者搜索、时间和点赞率排序，系列另支持完成率排序，新增投稿入口固定在列表顶部；全站排行榜保留短时缓存并并发拉取分页，服务端对共建统计做短 TTL 缓存，减少重复查询和打开等待。
- **移除后台「导出配置」**：不再提供 `GET /api/config/export` 与对应的「导出配置」按钮。
- **共建封面图访问控制按「最新版本是否仍引用」判定**：不看投稿审核状态；`sub_tasks` 中任一逻辑任务最新且 active 的版本仍引用该 URL 时即可访问。被换下、从未写入草稿或随系列缩短而失活的图片会变成不可访问的孤儿，但文件仍永久保留在磁盘。校验通过独立只读连接查询，不占用主写连接（`SetMaxOpenConns(1)`）。
- **共建投稿审核体验优化**：后台「共建审核」新增「取消撤回」——已下架/已撤回的投稿现在可以浏览到，一键恢复上线（曾正式发布过的）或退回初审队列（从未发布过的）；系列任务编辑表单的「取消编辑」移到表单下方，发布按钮改叫「保存并批准」；系列每一步的编辑区左右两栏互换（左栏惩罚标签、右栏难度/封面图/上移下移删除添加），未勾选「同时发布到随机任务」时封面图与四个操作按钮合并成一行；投稿状态标签统一「待审批」「已驳回」文案（不再区分初审与复审）；列表日期不再显示年份，详情页保持 `YY/MM/DD`。
- **惩罚任务评价交互修复**：对方发布的任务完成审核通过后，评价按钮此前不会出现（组件只在任务刚发布、证明尚未通过审核时拉取过一次评价资格，之后不再刷新）；现在证明状态变化会重新拉取，评价交互正常出现。
- **对局记录 UI 扁平化**：单场惩罚的任务与完成证明从三层嵌套（系列/任务池名 → 任务发布者与任务内容 → 完成证明）改为两个同级卡片，中间用虚线分隔，不再展示任务来自哪个系列/随机池/玩家自定义。
- **随机任务难度进度改为房间内按玩家独立计数**：与系列任务进度同构，`RoomState.PunishmentTaskProgress` 由房间级单一 `int` 改为按玩家 persistent ID 分槽的 `map[string]int`——谁挨罚就推进谁自己的难度基数，房间里其他人挨罚不影响自己；退座位/进观众席/断线重连/退房再进同一房间都保留，换房间从 0 开始，房间销毁即释放，不落盘。
- **共建投稿存储完成重构**：任务和系列分别落在全量版本化的 `sub_tasks` / `series`；每次编辑插入新版本，状态流转只更新最新行。玩家编辑已通过内容时，旧版会立即退出正式池，待新版重新审批；同一投稿最多保留 1000 个版本，防止长期刷写撑大数据库。系列缩短会给移除步骤写 inactive 墓碑，避免取消下架时复活旧步骤。投票按惩罚事件防重，并在同一事务内累加到玩家实际经历的精确任务版本。旧任务池、投稿信封、图片、投票等表在 schema v43 迁移成功后自动删除；v44 增加步骤 active 标记，v45 清理早期重构版本可能重建的空旧表。
- **每子惩罚（国际象棋 / 斗兽棋）**：开房勾选后，每被吃一枚棋子（或兽）立即触发一次惩罚，棋钟暂停到审批结束、审批后恢复对局（不计排位分、不改白给值）。终局那步若已因吃子受过罚不再重复罚；走进兽穴、将死、超时、认输这类没吃子的结束仍按终局惩罚。协议 `RoomSettings` 新增 `enable_per_piece_punishment` 字段（前后端需同步发布）。
- **斗兽棋白给**：开启白给后，轮到你走子时按白给值一半的概率直接跳过本手、对方继续走（除以 2 避免双方白给值都很高时无限互相跳过）；主人可强制宠物跳过本手（强制时宠物白给值按激活加成累加）。开局与每次走子后都会检查，可连环跳过（最多 4 手）。
- **各游戏排位档位可配置**：`games.json`（后台「称号与排位 → 各游戏排位档位」）可为每个游戏单独设置建房可选的赌分档位（最多 4 档、升序、第一个为默认档），用于平衡各游戏耗时——例如把国际象棋档位设为 15/30/60，赢一局相当于三局默认赌注的猜拳。黑白棋保持按每子结算的小额档位（默认 1/2/5/10），其余游戏默认 5/10/20 不变。协议 `GameConfig` 新增 `stakes` 字段（前后端需同步发布）；结算逻辑零改动，房间标签/历史记录/断线判负自动使用新档位。
- **系列任务进度改为房间内按玩家独立计数**：进度挂在当前 `*RoomState` 上、按玩家 persistent ID 分槽——谁输就推进谁自己的步数，房间里其他人受罚不影响自己；退座位/进观众席/断线重连/退房再进同一房间都保留，换房间（即便系列相同）或房内换成另一个系列都从 0 开始，房间销毁即释放。替代此前按玩家个人落盘、跨房间跨对手持续推进的模式（schema v32 删除废弃表 `player_punishment_series_progress`）。未覆盖阵营时从随机池抽替补，替补步同样推进该玩家自己的进度。
- **系列目标阵营与外部阵营替补**：系列提交和审批时要求每一步覆盖它声明的全部目标阵营。目标集合之外的玩家进入房间前会收到知情确认；若仍进入，某一步没有该玩家阵营的文案时，从其阵营随机任务池抽一条顶替（难度按当前步序/总步数比例，标签由该系列自身词频推导并逐级放宽），个人进度照常推进。协议使用 `PunishmentSeriesSummary.target_faction_ids` 下发目标集合。
- **后台惩罚内容改为审核制**：旧的「系列任务 / 任务池」直接编辑入口及保存 action 已删除；后台保留全局惩罚配置、性别与阵营、共建审核、批注和下架。管理员可以在共建审核详情中编辑原稿并立即发布，但所有正式任务/系列仍统一经投稿版本事务发布并保留审计记录。审核详情可查看真实评价统计，不提供管理员人工覆盖展示点赞率的功能；系列整体点赞率按全部步骤的真实赞踩票合并计算（票数加权）。正式任务分配后双方即可评价，证明被驳回或要求重做不会撤销已有票数。
- **RPS 撤回出拳**：锤子剪刀布中一方出拳、另一方尚未出拳时，出拳方可以撤回并重新选择；主人强制的白给不可撤回。白给点击次数和数值加成在回合结算时统一发放，避免撤回操作产生重复加成。
- **共建排行榜**：全局排行榜新增「共建」页签，按已投稿且通过审批的独立随机任务和系列步骤数量排序；该统计只在 `players:roster` 查询时按需聚合，不进入大厅实时快照。
- **无标签惩罚任务改为常驻随机池**：任务池里未勾选任何标签的任务不再被随机模式排除——它们无视建房时的标签选中/拒绝筛选，任何随机房间都可能抽到（此前无标签任务永远抽不到）。想专门留给系列任务用的任务，请把难度设为 -1，或干脆不勾选阵营——两种都会让它不进随机池（本次变化只针对标签：留空标签不再排除随机候选）。阵营勾选逻辑不变：未勾选阵营的任务仍然抽不到。
- **棋类悔棋扩展**：斗兽棋、国际象棋、黑白棋现与五子棋同样支持建房设置每人 0/1/3/10 次悔棋；仅当前行棋方可申请，对方 30 秒内确认后回退最近两手，等待期间暂停棋钟。黑白棋会同步逆转被撤销两手已经实时结算的排位分与白给值；国际象棋完整恢复易位权、吃过路兵、升变、半子钟与重复局面历史。新增事件：`othello:undoRequest` / `othello:undoRespond`、`jungle:undoRequest` / `jungle:undoRespond`、`chess:undoRequest` / `chess:undoRespond`。
- **新增国际象棋**：与五子棋/斗兽棋同级的座位制玩法，8×8 棋盘白方先走，双方准备后随机执白。支持王车易位、吃过路兵、兵升变；将死判胜，逼和/子力不足/五十步/三次重复判和。支持认输请求、棋钟、排位、惩罚与断线判负，不支持白给。事件：`chess:ready` / `chess:move` / `chess:resignRequest` / `chess:resignRespond` / `chess:restart`。
- **后台配置重组**：原「防多开」一级菜单并入「网站管理」——「限流策略」按设备指纹、IP、房间与证明图片分组展示，「提示语」独立为单独区块并使用易读的中文名称；「禁止新用户注册」开关移至「提示公告」下；新增服务器「运行状态」卡片；用户管理并入「用户与房间」。「防多开」一级菜单删除。
- **用户分析「对局时长」改均值口径**：删除「对局结果」统计；原「每局时长」「房间时长」两张总时长图合并为一张「对局时长」均值图——按游戏拆分的单局均值分钟数（总耗时 ÷ 总局数，保留 1 位小数，避免 RPS 这类单局 <1 分钟的游戏被取整拍成 0）左轴堆叠柱 + 全站单房均值分钟数（房间存活总时长 ÷ 开房总数）右轴折线；图表横轴日期刻度省略年份只显示 mm/dd。
- **斗兽棋初始布局对称修正**：双方棋子改为按各自视角 180° 旋转对称摆放（修正此前后排象/狮左右不一致）；规则说明补充「敌兽进入己方陷阱，可被任意己方兽吃掉」。
- **后台聊天管理**：新增 `admin:chatSearch` 历史检索与单条/批量软删除、恢复（schema v28）；房间聊天落盘时保存发送时刻的房间名快照，房间改名/关闭后仍可按当时的房间名检索。
- **用户管理**：玩家列表新增昵称筛选；新增同设备关联账号（小号）BFS 查询（独立只读连接 + `connection_events(device)` 索引）。
- **上传文件访问控制**：`GET /uploads/*` 新增 Referer 校验（缺失/跨站一律 404，与 Origin 校验同一套白名单逻辑）；`Cache-Control` `max-age` 由 30 天收窄到 6 小时；证明图随所属房间销毁而失效（房间清空/管理员关房/进程重启后一律 404）。仍非强鉴权，详见「证明图」小节。
- **用户分析**：修复转化漏斗倒挂——「访问」层改 sessions∪events 并集口径、五层「深层向浅层逐级并集回填」（schema v27/v29 重算已封板历史日）；新增随机惩罚标签勾选/排除与系列任务选择统计。
- **对局结算**：五个双座位游戏胜方文案统一为「玩家A/B「昵称」胜利」；`DisplayName` 与资料页称号共用同一套解析优先级（修复宠物称号/自设称号在结算文案里不同步）；RPS 结算结果说明作为房间通知实时推送。
- **随机惩罚标签偏好**：改为浏览器本地保存（`localStorage`），不再落玩家档案、不再跨设备同步（schema v30 废弃旧偏好表）；建房惩罚入口合并为无惩罚/随机/系列/玩家发布四选一；房间内可直接发大厅聊天。
- **惩罚任务系统重做**：`punishmentSource` 三态改为 `random`（原 `system`）/ `series`/`player`；随机任务改标签化三态筛选 + 按房间走的倒伽马难度加权，新增按玩家个人进度推进的系列任务模式。任务池/系列详情迁出 `AppConfig` 落 SQLite（`punishment_tasks` / `punishment_series`，schema v22–v26），`punishments.json` 只保留 `tags` + 全局难度；建房选系列用公开摘要 `punishmentSeriesSummaries`；后台重排为「惩罚配置 / 系列任务 / 任务池」三个 tab。
- **前后端版本一致性提示**：后端 `-ldflags` 注入 git 短哈希、前端 `vite.config.ts` 同步注入，连接建立与心跳 ping 携带构建版本，前端检测到与后端不一致时展示横幅提示手动刷新（不自动强刷）。
- **用户分析 + 后台图表面板**：前端埋点与审计表聚合、ip2region 归属地（`config/xdb/ip2region_v4.xdb` 必需 + 可选 `ip2region_v6.xdb`）、日聚合三层架构、`/admin`「数据分析」分区（LayerChart）；转化漏斗重定义、新增用户留存矩阵与单房对局统计。升级须一次性放置 xdb 或设 `ANALYTICS_GEO_ENABLED=0`。

详见 [CHANGELOG.md](./CHANGELOG.md)。
