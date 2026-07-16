# 抖喵游戏屋

实时联机小游戏平台：**Go 后端** + **React 前端**。

主玩法：锤子剪刀布 / 黑白棋 / 井字棋。含大厅、房间、聊天、观战、Bot、排位、惩罚、名字争夺战、白给、极限模式、后台配置等。

## 目录结构

```
.
├── cmd/
│   ├── server/          # Go 服务入口
│   └── wsprobe/         # WebSocket 探针（开发用）
├── internal/
│   ├── config/          # 配置加载/校验（config/*.json）
│   ├── delta/           # 通用 JSON 增量 Diff/Apply/Hash
│   ├── server/          # 游戏逻辑、HTTP、WebSocket
│   ├── types/           # 服务端领域类型
│   └── wire/            # Protobuf 生成代码
├── api/proto/           # wire.proto 协议定义
├── config/              # 按功能拆分的 JSON（原地读写，无 active/default 双轨）
├── web/                 # 前端（Vite + React + TS）
│   ├── src/
│   │   ├── App.tsx      # 壳：会话恢复 / 视图路由 / 顶栏
│   │   ├── lib/         # session、rpc、normalize、format、proofImage…
│   │   ├── ui/AppViews  # 登录/大厅/房间/后台等页面与组件
│   │   ├── ws.ts / wire.ts / delta.ts
│   │   └── shared/types.ts
│   ├── index.html
│   ├── package.json
│   └── vite.config.ts   # 构建输出到仓库根 dist/
├── dist/                # 前端构建产物（gitignore；Go 静态托管）
├── bin/server           # Go 可执行文件（gitignore；Linux amd64）
├── data/                # players.json 等（gitignore）
├── work/                # uploads、session.secret（gitignore）
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
| `newRPS-<version>-linux-amd64.tar.gz` | `bin/server`、`dist/`、`docker-compose.yml`、`.env.example`、`config/*.json`、空 `data/`/`work/`、简要 `README.md` |

解压后即可 `docker compose up -d`。源码仓库仅用于开发；`bin/`、`dist/` 已 gitignore。

## 本地运行

### 方式一：生产一体（Go 托管前端）

```bash
# 根目录：装前端依赖 + 构建
npm install
npm install --prefix web
npm run build          # web → dist/ 且编译 bin/server

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
- 挂载：`bin/server`、`dist`（只读）+ `data` / `work` / `config`（持久化）
- 停止：`docker compose down`（数据目录保留）

#### 升级服务器且不丢玩家数据

**务必保留**以下路径（不要用新包整目录覆盖掉它们）：

| 路径 | 内容 | 说明 |
|------|------|------|
| `data/players.json` | 玩家档案、积分、战绩等 | **核心存档** |
| `work/uploads/` | 证明图、后台上传图 | 丢了历史图片链会 404 |
| `work/session.secret` | 会话 HMAC（未设 `SESSION_SECRET` 时） | 丢了则旧浏览器 token 全部失效 |
| `config/*.json` | 后台改过的运行时配置（按功能拆分） | **整目录备份**；升级勿用空包覆盖已改配置 |
| `.env` | `SESSION_SECRET`、`ADMIN_PASSWORD` 等 | **`SESSION_SECRET` 不要换**，否则等同全员掉登录 |

可用新版本覆盖的：`bin/`、`dist/`、`docker-compose.yml`。配置仅在确认需要重置时再覆盖 `config/`。

```bash
# 备份数据（推荐）
tar czf backup-$(date +%F).tgz data work config .env

# 解压新包到临时目录，覆盖程序（保留 data/work/config/.env）
tmpdir=$(mktemp -d)
tar -xzf newRPS-2.1.24-linux-amd64.tar.gz -C "$tmpdir"
cp -a "$tmpdir/bin" "$tmpdir/dist" "$tmpdir/docker-compose.yml" .
rm -rf "$tmpdir"

docker compose up -d
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

状态校验：对「前端形态」规范化树做 **CRC-32（IEEE）**（8 位 hex，非密码学；两端对齐）。`player:batch` / 房间广播 debounce 等合并策略不变。
| `player:get` | 拉取完整 `PublicPlayer`（大厅仅下发精简 `LobbyPlayer`） |

代码生成：`protoc` → `internal/wire/*.pb.go`；前端 `pbjs` → `web/src/gen/proto.js`。

## HTTP API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/session` | 签发会话 token |
| POST | `/api/proof-image` | 证明图：仅 webp，≤2MB |
| POST | `/api/admin-image` | 后台图：jpg/png/webp |
| GET | `/api/config/export` | 导出配置（需管理员口令） |
| GET | `/ws` | WebSocket |
| GET | `/uploads/*` | 上传文件 |
| GET | `/*` | 前端静态（`dist/`） |

### 证明图

- 前端：长宽比 >21:9 拒绝；原图 >10MB 拒绝；>4MP 缩放；WebP 85%
- 后端：非 `.webp` 拒绝；>2MB 拒绝；错误信息用户可见

静态 hash 资源：`Cache-Control: public, max-age=31536000`

## 环境变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `HOST` | `0.0.0.0` | 监听地址 |
| `PORT` | `9988` | 监听端口 |
| `ADMIN_PASSWORD` | （空） | 后台口令 |
| `SESSION_SECRET` | `work/session.secret` | 会话 HMAC；未设置则落盘复用 |
| `SESSION_TTL_MS` | 24h | 会话有效期 |
| `ALLOWED_ORIGINS` | 本机 | 额外 Origin |
| `TRUSTED_PROXY_COUNT` | `1` | 可信反向代理层数，决定 `X-Forwarded-For`/`X-Forwarded-Host` 信任方式；直连部署（无反代）需设为 `0`，否则客户端可伪造来源 IP 绕过限流/防多开 |
| `MAX_SOCKETS_PER_IP` | 按防多开人数上限 ×4（至少 12） | 单设备（IP+指纹）WebSocket 套接字上限 |
| `LOBBY_BROADCAST_DELAY_MS` | 300 | 大厅广播合并 |
| `ROOM_BROADCAST_DELAY_MS` | 100 | 房间广播合并 |

### 防多开（IP + 浏览器指纹）

中国公网出口常被整栋楼/公司共用，**不能只按 IP 限流**。

- 前端用 [FingerprintJS](https://github.com/fingerprintjs/fingerprintjs) 生成 `visitorId`
- 服务端 `deviceKey = sha256(ip + "\\0" + fingerprint)`
- 配置项（字段名兼容旧版）：
  - `accessControl.maxOnlinePerIp` → **同指纹同时在线人数上限**
  - `accessControl.maxCreatesPer10Min` → **同指纹 10 分钟内新建玩家上限**
- 上报路径：`POST /api/session`（Header/Body）、`/ws?fp=`、`player:join.fingerprint`

## 后台与配置文件

- 入口：`/admin` 或 `#admin`，或 `Ctrl/Cmd+Shift+A`
- 配置在 `config/` 下**按功能拆分、原地读写**（无 active/default 双轨）。旧版单体 `default.json`/`active.json` 启动时会自动迁移并改名为 `*.bak`。

| 文件 | 内容 |
|------|------|
| `site.json` | 站点名、简介、管理员口令 |
| `daily-announcement.json` | 每日公告 |
| `gender-factions.json` | 性别阵营（genders 由阵营展开） |
| `titles.json` | 称号段 |
| `punishments.json` | 系统惩罚池 |
| `player-punishment-room-name-pool.json` | 玩家发布任务房名词库 |
| `room-tags.json` / `room-info-tags.json` | 房间 Tag 与信息标签样式 |
| `access-control.json` | 防多开 |
| `name-war.json` / `giveaway.json` / `extreme-mode.json` | 名争 / 白给 / 极限模式文案与参数 |
| `bots.json` / `games.json` / `messages.json` | Bot、游戏列表、提示文案 |

后台「保存」会写回对应 JSON；服务启动时会把 `config/*.json` 权限收紧为 `0600`（仅运行用户可读写，防同机其他用户读取其中的管理员口令），`bin/server` 设为可执行。

## 多端身份认领

没有传统用户名密码：每台浏览器首次访问时在本地生成一对 `playerId` + `playerSecret`（长随机串），存 `localStorage`，此后 `player:join` 每次都带上这对凭据重连。战绩/积分只跟 `playerId` 走。

- **认领新设备**：个人资料页展示一个一次性「认领密钥」（`ClaimKey`，与长期 `playerSecret` 是两个不同的值），复制到另一台设备的输入框提交（`identity:claim`），服务端校验通过后给新设备签发一份全新的 `playerSecret` 并**立即轮换掉这把密钥**（用过即作废）；旧设备完全不受影响。
- **多端同时"记住"，但同一时刻只有一端在线**：一个身份最多同时记住 3 台设备的凭据（`PlayerSecrets`，超出后挤掉最早一条，绝不挤当前活跃会话），但服务端仍是单 socket 模型——已有设备在线时，新设备登录会先收到 `alreadyOnline` 提示确认是否顶替，确认后走 `forceKick`，被顶替端会收到 `session:kicked` 事件（同设备刷新重连不受影响，不会误触发确认）。
- **登出**：`identity:logout` 撤销当前设备的那条 `playerSecret`，前端随后清空本地 `localStorage`。

⚠️ **一次性迁移代码，请在合并后一个月内删除**：早期版本身份凭据只存 `hashSecret(secret)`（`internal/server/util.go` 的 `hashSecret`），字段是 `PlayerState.PlayerSecretHash` / `persistedPlayer.PlayerSecretHash`。现在改为明文存储的 `PlayerSecrets` 列表（认领密钥本身就要求服务端能把密钥"读出来"展示给用户，哈希做不到这点；而"服务器数据泄露"这个威胁模型下，明文认领密钥本来就足以接管账号，给另一个字段单独加密没有实际收益，详见迁移决策讨论）。`internal/server/identity.go` 的 `verifyPlayerSecret` 里有一段兼容分支：老账号第一次带着老 secret 重连时，校验通过后会自动把明文迁移进 `PlayerSecrets`，不需要用户做任何操作。

一个月后（正常活跃账号届时都已完成自动迁移）应整体删除：
- `internal/server/identity.go` 的 `verifyPlayerSecret` 里 `PlayerSecretHash` 兜底分支
- `PlayerState.PlayerSecretHash` / `persistedPlayer.PlayerSecretHash` 字段
- `internal/server/util.go` 的 `hashSecret` 函数

**代价**：这一个月内始终没有上线过的账号，届时会无法再用老设备的身份登录（因为兜底校验代码被删掉、且它们的明文 secret 从未被自动迁移过）——这是时间窗口本身带来的必然结果，不是 bug。

## 大话骰（Liar's Dice）

第四种游戏类型，与 RPS/黑白棋/井字棋同级，但刻意不复用它们共用的 `Seats`/`SeatKey`（固定两人）模型——大话骰是 2-8 人，独立走 `RoomState.LiarsDice`（公开状态，进房间快照）+ `RoomState.LiarsDiceHands`（私有骰子，只通过 `emitToClient` 单播给玩家自己，从不广播，也不加密——保密性来自"这份字节压根不会出现在其他玩家的连接上"）。核心逻辑集中在 `internal/server/game_liarsdice.go`。

- **入席**：进房间默认观战，对局未开始前可自由 `liarsdice:joinRoster` / `liarsdice:leaveRoster`；房间设置里 `liarsDiceMinPlayers`（2~上限，默认 3）/`liarsDiceMaxPlayers`（默认 3，上限 8）。
- **开局**：参战名单全员 `liarsdice:ready` 且名单 5 秒无变动 → 每人现摇 5 颗骰子，随机选首个叫点者（按入席顺序循环叫点）。
- **叫点规则**：每回合只能"叫"（`liarsdice:bid`，颗数更多，或颗数不变但点数更大）或"开"（`liarsdice:challenge`，质疑上家），没有"过"。第一个叫点数至少是"在场人数 + 1"。1 是万能点，但只要本局有人喊过 1，之后 1 不再算万能，仅算实际点数。
- **结算**：开牌揭晓全体参战玩家骰子（不只叫点者和质疑者），按叫点面值（含万能 1，若未禁用）计数；成立则叫点者胜、质疑者负，反之相反；其余参战玩家本局"平"，不计分不受罚。下一局重新摇骰，不延续骰子数量、不淘汰。
- **断线判负**：断线判负的规则是"上家"（入席顺序里的前一位，固定关系，与谁最后叫过点无关）胜——`createLiarsDiceDisconnectForfeit`/`applyLiarsDiceDisconnectForfeit`，与其它三个游戏的 `DisconnectForfeit` 走独立的 `LiarsDiceDisconnectForfeit`（字段是 playerID 而非 SeatKey）。
- **惩罚**：`punishment.go` 的 `setupPunishmentForPlayers` 从原来 Seat/RoundResult 耦合的 `setupPunishmentOrNext` 里抽出通用尾段（按 playerID 列表工作），大话骰和其它三个游戏共用这一段；`buildLiarsDicePunishmentTasks` 单独实现（赢家直接作为"玩家发布任务"模式下的任务发布人，不走 Seat 反查）。

## 构建与测试

```bash
npm run build:web      # 仅前端
npm run build:server   # 仅 Go（并 chmod +x bin/server）
npm run fix-perms      # config/*.json 收紧为仅属主可读写（0600）+ bin/server 可执行
npm run build          # web + server + fix-perms
go test ./...
npm run test           # go test + 前端 build
```

## 最近更新记录

### v2.1.27（2026-07-16）

- **大话骰体验修复**：
  - 叫点面板颗数改为下拉选择（原为数字输入），并按"上家点数 &lt;6 则颗数不变点数+1，=6 则颗数+1点数回到1"自动预选最小合法叫点；除首次摇骰展示自己手牌外，其余叫点/结算文案不再用骰子字体，直接显示数字。
  - 开牌结果面板"下一局"按钮与骰子列表贴在一起的问题修掉，胜负配色改用页面既有的蓝/粉色系（不再是刺眼红绿）。
  - 房间标签/默认房间名此前会显示成"锤子剪刀布""出拳中"/"新的锤子剪刀布房间"，补上大话骰专属文案与配色。
  - 对局记录里大话骰此前显示无意义的"未出拳"，改为按胜负显示；切到后台时收不到"轮到你了"提醒的问题（判断逻辑只认 A/B 座位，大话骰没有座位）一并修复。
  - **开惩罚的大话骰房间结算时看不到每人骰子点数、直接跳到惩罚阶段**：惩罚流程会在同一次结算里把房间阶段从"结算展示"直接推进到"惩罚阶段"（这是通用设计，其它玩法的结算面板本就同时兼容这两种阶段），大话骰的开牌揭晓面板之前漏了这个兼容判断，现已修复。

### v2.1.26（2026-07-16）

- **多端身份认领**：个人资料页新增一次性「认领密钥」，可在另一台设备粘贴认领当前身份；一个身份最多同时"记住" 3 台设备（挤掉最早一条，不挤当前活跃会话）；服务端仍是单 socket 模型，新设备登录会先确认是否顶替旧设备在线会话，被顶替端收到提示。新增登出按钮（清空本地凭据）。详见「多端身份认领」章节。⚠️ 旧版哈希凭据自动迁移为明文列表，一个月后将删除迁移兼容代码。
- **推送通知**：个人资料页可分别开启「聊天 @ 我」「轮到我出招/落子」「我的房间来人了」三类提醒。页面在前台/后台切换时优先走浏览器原生 `Notification`（Level 1）；玩家完全断线（关闭标签页/浏览器）时改由 Service Worker + Web Push 推送（Level 2），自建 VAPID 密钥，不依赖第三方推送服务。**需要 HTTPS（或 `localhost`）才能授权通知权限**，纯 HTTP 内网访问会一直提示无权限，属浏览器安全限制。
- **新增大话骰（Liar's Dice）**：与锤子剪刀布/黑白棋/井字棋同级的第四种玩法，支持 2-8 人（房间设置可调最少/最多参战人数）。进房默认观战，对局开始前可自由加入/离开参战席；全员准备且名单 5 秒无变动后自动开局，每人摇 5 颗私有骰子（只推送给玩家本人）；按入席顺序轮流叫点或开牌，1 为万能点（一旦被喊出则本局失效）；断线判负规则为"上家"（入席顺序前一位）胜。支持排位积分与惩罚（含"玩家发布任务"模式）。详见「大话骰」章节。

### v2.1.25（2026-07-15）

- **聊天持久化**：房间聊天与大厅留言板改为写入 SQLite（`data/chat.db`，`mattn/go-sqlite3`），重启不再丢失历史；历史通过 `chat:load` / `chat:loadOlder` 分页拉取（瀑布流，滚到顶部自动加载更早 100 条），实时增量走新的 `chat:new` 频道推送。
- **留言板并入聊天**：大厅原有的独立「留言板」（`suggestion:add`）下线，与大厅聊天合并为同一套 UI/存储，房间内的「大厅」tab 与大厅页共用同一份数据。
- **@提及**：聊天可以点头像 @ 某位玩家，被 @ 的消息气泡高亮显示；每条消息最多记录 20 个提及对象。
- **构建方式变更**：`bin/server` 恢复为 CGO 动态链接（`CGO_ENABLED=1`，因为 sqlite3 驱动需要 CGO），不再是 v2.1.24 的纯静态二进制；`docker-compose.yml` 基础镜像由 `debian:bookworm-slim` 换为 `debian:trixie-slim`（更高版本 glibc），避免构建机 glibc 版本高于运行环境导致二进制无法启动。**升级注意**：沿用旧部署包（`docker-compose.yml` 仍是 `bookworm-slim`）的用户需要拉取本次 Release 包中的新 `docker-compose.yml`，否则新二进制可能因 glibc 版本不够而无法启动。

### v2.1.24（2026-07-15）

- **黑白棋棋子清晰度**：`disabled` 格子整体变暗（`opacity:.6`）会连带压暗棋子；棋子改为格子外层的独立兄弟节点，格子变暗与棋子透明度（固定 0.8）互不影响，格子仍是正方形、棋子占格子边长 50%。
- **Chrome 无法选择 HEIC 证明图**：`<input accept="image/*">` 通配符会用 Chrome 内置的图片扩展名表过滤文件选择器，该表不含 heic/heif，导致即使额外列了 `.heic,.heif` 也可能被隐藏；改为只列具体扩展名/MIME，不再用通配符，选完文件后的校验/压缩/上传流程不变。
- **安全与稳定性加固**：
  - WebSocket 广播/应答不再在持有全局锁时做同步网络写，改为每连接独立发送队列 + 写协程，避免单个慢客户端拖慢全服；连接队列写满会主动断开该连接。
  - 修复握手阶段在锁外操作共享同步状态可能触发的并发 panic。
  - `X-Forwarded-For` / `X-Forwarded-Host` 改为按 `TRUSTED_PROXY_COUNT`（默认 1）解析可信代理层数，不再无条件信任客户端可伪造的最左侧值。
  - 管理员口令改为恒定时间比较；`config/*.json` 权限收紧为 `0600`；`/api/config/export` 增加专属限流，口令改用请求头传递（不再出现在 URL / 反代访问日志里）。
  - 管理员踢人（kick）不再删除持久玩家的存档（积分/战绩/称号），只强制其下线，账号可正常重新登录。
  - 惩罚/房名等随机选择改为真随机（原实现按时间戳取模，可预测且同毫秒内取值相同）。
  - 限流、防多开、玩家历史出拳等内存状态定期清理，避免长期运行内存增长。
  - 进程收到关停信号时改为 `Shutdown`（原为 `Close`），给在途 HTTP 请求收尾时间，而不是硬切。
- **日志体系重做**：
  - 命令行只保留启动横幅和「进程无法继续运行」级别的致命错误；连接、建号、改名、白给/名字争夺战/极限模式开关、留言、聊天、惩罚等事件不再打印到终端，全部落盘到 `work/logs/{ISO年+周}/*.csv`。
  - 新增 `error.csv` 记录鉴权失败、限流、越权等拒绝/异常事件，字段拆成标准 CSV 列（而非一整块 JSON 字符串）。
  - `players.csv` 补充 `rename`（改名）、`giveaway_enable`/`giveaway_disable`、`giveaway_board_submit`（白给自救板内容）、`nameWar_enable`/`nameWar_disable`、`extreme_enable`/`extreme_disable` 等操作记录。
  - `chat.csv` 的 `channel` 只保留 `room` / `lobby` 两种取值；留言板（原 `lobby_suggestion`）并入 `lobby`，对应真正在用的「大厅聊天」体验。
- **文档**：README 删除反向代理配置示例与维护约定说明；补充 `TRUSTED_PROXY_COUNT`、`MAX_SOCKETS_PER_IP` 环境变量说明。

### v2.1.23（2026-07-13）

- **配置拆分**：取消 `default.json` / `active.json` 双轨；按功能拆成 `config/*.json` 原地读写；旧单体启动时自动迁移为 `*.bak`。构建与启动修正 `config/*.json` 写权限、`bin/server` 执行权限。
- **独立部署包**：Release 含 `bin/server`、`dist/`、`config/*.json`、`docker-compose.yml`、`.env.example` 与简要 README，解压即可 `docker compose up -d`。
- **战绩展示**：protobuf 省略 `0` 导致个人资料/排行榜出现 `undefined`/`NaN`（如总局数、胜负平、排位积分）；`materializePlayer` + `safePlayerStats` 统一补 0，空称号显示「暂无称号」。
- **防多开配置字段**：`maxCreatesPer10Min` 与 protobufjs `maxCreatesPer_10Min` 错位导致后台显示 undefined、保存后看似不生效；前端归一化并修正 gen 字段名。
- **惩罚任务占位符**：系统/玩家任务文案支持 `{loser}`（败者）、`{winner}`（胜者）昵称替换。
- **加密房密码框**：`stripHasFlags` 不再误删业务字段 `hasPassword`；presence 标记仅在有 companion 时剥离，并还原 `false`/`0` 省略值。
- **房间坐标 / 增量状态**：黑白棋·井字 `row/col=0` 归一化；DELTA 路径补丁 + CRC-32 合并校验；`ready`/`score` pair 缺省零值修复。
- **房间信息标签**：`room-info-tags.json` 中文名与配色；前端 key 缺失回退默认文案。
- **其它**：惩罚提交/审批链路加固；RPC pending 与 reply 错误回传；Safari 房间布局；产物不入库改走 GitHub Release。

### 2026-07-13（前序）

- **全链路 Protobuf**、状态增量方案 A、前端结构拆分、handlers guard、null 兜底等（见 git 历史）。

### 2026-07-12

- **在线人数**：`player:batch` 支持插入新玩家并按 `connected` 重算人数；新上线 `forceBroadcastLobby`，修复「下线 -1 正常、上线需刷新才 +1」。
- **WebSocket 稳定性（生产 / Safari）**：
  - 根因确认：Safari 对 `permessage-deflate` 不兼容；`NoContextTakeover` 仍是同一扩展，不能单独救 Safari。
  - **UA 分流压缩**：Safari / iOS 关闭压缩，Chrome 等使用 `CompressionContextTakeover` 省流量。
  - 服务端 20s 协议 Ping；客户端 25s 应用层心跳、半开检测、回前台探活。
  - 反代：`X-Forwarded-Host` Origin 校验；同 SID 重连先释放连接名额；WS 升级少塞响应头。
- **防多开**：引入 FingerprintJS；`deviceKey = sha256(ip||fingerprint)`；同时在线 / 10 分钟新建 / 套接字上限均按设备键（配置字段名兼容旧版）。
- **体验**：惩罚阶段任务图、证明图可点击放大（与对局记录一致）。
- **会话 token**：WS 握手失败（含 401→浏览器 1006）自动丢弃本地 token 并 `POST /api/session` 换发（限次），避免密钥轮换后旧 token 永久卡在「正在连接…」。
- **同 SID 双连**：旧 socket 的 onclose 不再误清新连接；`replaced` 不盲目重连，降低双标签互踢。
- **稳定性**：修复 `socket_duplicate` 后 `assignment to entry in nil map` panic（顶替连接与 device map 竞态）；相关 map 写入统一 ensure。
- **欢迎公告**：默认文案增加交流 QQ 群 **432398160**（Bug 反馈 / 新功能）。
- **配置加载**：去掉 `config.go` 内嵌默认文案，只认 `config/*.json` 拆分文件；缺文件/校验失败直接报错。

### 2026-07-11

- **目录整理**：前端迁入 `web/`；删除旧 Node.js 后端（`src/server`、socket.io/express 等）；根目录只保留 Go 模块与编排脚本（`npm run dev` / `npm run build` / `npm start`）。
- 后端由 Node.js/Express/Socket.IO **完整重写为 Go**；Protobuf 线协议 + 通用增量同步 + 大厅 `LobbyPlayer` 精简 + 玩家更新 100ms 聚合 + 房间广播默认 100ms。
- 证明图前后端校验与长期静态缓存；默认端口 **9988**。
- 修复 `nil` 切片 JSON `null` 白屏、player:batch 误删状态、哈希 HTML 转义不一致等。
- **登录态刷新**：修复已登录用户反复刷新时在登录页/大厅间跳转；仅在 WS 连通后恢复会话，避免「未连接」误清 token。
- **实时推送**：
  - 房间广播带上 recent `roundHistory`（自定义任务、证明不再需要手动刷新）。
  - 空房删除时大厅立即全量刷新；离房后补发大厅 FULL，避免幽灵房间。
  - `roomNotice` 同步 `chat:append`，系统提示即时可见。
- **惩罚流程**：创建房间默认开启「惩罚需对手确认」；系统任务与自定义任务一致，败方提交后进入 `pending` 由胜方审批，不再直接开新局。
- **对局记录**：展示任务完成证明（状态/文字/图片/审核备注）；前端按 id 合并 history，避免覆盖丢失。
- **黑白棋终局文案**：白给/上贡改为统计摘要，不再把每一手明细拼进结果句。
- **Docker 部署**：Release 独立部署包（含 compose / config / env 模板 + bin/dist）；也可源码本机构建后挂载。
