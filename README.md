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
├── docker-compose.yml   # debian:bookworm-slim + 挂载产物
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

**只需 Release 部署包**（不必 clone 源码）。使用官方 **`debian:bookworm-slim`**，**无需 Dockerfile**。

```bash
# 下载并解压到任意目录（示例：2.1.23）
gh release download v2.1.23 -p 'newRPS-*-linux-amd64.tar.gz' --repo 8023/newRPS
mkdir -p gamehouse && tar -xzf newRPS-2.1.23-linux-amd64.tar.gz -C gamehouse
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
tar -xzf newRPS-2.1.23-linux-amd64.tar.gz -C "$tmpdir"
cp -a "$tmpdir/bin" "$tmpdir/dist" "$tmpdir/docker-compose.yml" .
rm -rf "$tmpdir"

docker compose up -d
```

> 说明：Release 中的 `bin/server` 为 **Linux amd64**。ARM 服务器需在对应架构上重新 `npm run build:server`。

反向代理（OpenResty/Nginx）WebSocket 必配示例：

```nginx
# 页面可用 HTTP/2；/ws 必须能完成 HTTP/1.1 Upgrade（浏览器会单独建连）
location /ws {
    proxy_pass http://127.0.0.1:9988;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_read_timeout 3600s;
    proxy_send_timeout 3600s;
    proxy_buffering off;
}
```

`ALLOWED_ORIGINS` 建议设为你的站点 Origin（如 `https://rps.rbq.io`）。

服务端：20s 协议 Ping、识别 `X-Forwarded-Host`；WebSocket 压缩见下节（**按 UA 分流**，不是全局关闭，也不是全局强制 deflate）。

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

后台「保存」会写回对应 JSON；构建脚本与服务启动会尽量把 `config/*.json` 设为可写、`bin/server` 设为可执行。

## 构建与测试

```bash
npm run build:web      # 仅前端
npm run build:server   # 仅 Go（并 chmod +x bin/server）
npm run fix-perms      # config/*.json 可写 + bin/server 可执行
npm run build          # web + server + fix-perms
go test ./...
npm run test           # go test + 前端 build
```

## 维护约定

功能变更时同步更新「最近更新记录」。同步 GitHub 需 `git commit` 与 `git push`。

## 最近更新记录

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
