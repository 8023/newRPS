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
├── config/              # default.json / active.json（运行时）
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

**`bin/server` 与 `dist/` 不进入 git**，由各版本的 **GitHub Release 附件** 提供，避免仓库膨胀。

| 附件 | 内容 | 用途 |
|------|------|------|
| `newRPS-<version>-linux-amd64.tar.gz` | `bin/server` + `dist/` | Docker 挂载的只读运行时（与 `docker-compose.yml` 路径一致） |

发布包**仅含**上述运行时文件，不含 `docker-compose.yml` / 源码 / `config` 等（compose 与配置仍从仓库获取）。

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

使用官方 **`debian:bookworm-slim`** 挂载 `bin/server` + `dist/` 启动，**无需 Dockerfile**。运行产物从 **Release** 下载（或本机 `npm run build`）。

```bash
git clone <repo> && cd newRPS

# 下载并解压运行时（示例：2.1.22）
gh release download v2.1.22 -p 'newRPS-*-linux-amd64.tar.gz' --repo 8023/newRPS
tar -xzf newRPS-2.1.22-linux-amd64.tar.gz
# 得到 ./bin/server 与 ./dist/ ，路径与 compose 挂载一致

cp .env.example .env          # 可选：ADMIN_PASSWORD、SESSION_SECRET、ALLOWED_ORIGINS
docker compose up -d

docker compose ps
docker compose logs -f gamehouse
```

无 `gh` 时也可在 GitHub 网页下载 Release 附件后解压到仓库根目录。

- 访问：`http://服务器IP:9988`（`HOST_PORT`，默认 9988）
- 挂载：`bin/server`、`dist`（只读）+ `data` / `work` / `config`（持久化）
- 本机改源码：`npm install --prefix web && npm run build`，再 `docker compose up -d`（不必把产物提交进 git）
- 停止：`docker compose down`（数据目录保留）

#### 升级服务器且不丢玩家数据

更新代码 / 二进制时，**务必保留**以下目录与文件（compose 挂载卷不要删）：

| 路径 | 内容 | 说明 |
|------|------|------|
| `data/players.json` | 玩家档案、积分、战绩等 | **核心存档** |
| `work/uploads/` | 证明图、后台上传图 | 丢了历史图片链会 404 |
| `work/session.secret` | 会话 HMAC（未设 `SESSION_SECRET` 时） | 丢了则旧浏览器 token 全部失效 |
| `config/active.json` | 后台改过的运行时配置 | 没有则回落 `default.json` |
| `.env`（或部署环境变量） | `SESSION_SECRET`、`ADMIN_PASSWORD` 等 | **`SESSION_SECRET` 不要换**，否则等同全员掉登录 |

可替换、不必保留的：

| 路径 | 说明 |
|------|------|
| `bin/server` | 新版本可执行文件（Release 或本机构建） |
| `dist/` | 新前端静态资源 |
| `config/default.json` | 仓库默认配置（可被新版本覆盖） |

示例（在宿主机备份后替换程序）：

```bash
# 备份数据（推荐）
tar czf backup-$(date +%F).tgz data work config/active.json .env

# 拉取源码更新（compose / config 等）
git pull

# 下载新版本运行时并覆盖 bin/、dist/
gh release download v2.1.22 -p 'newRPS-*-linux-amd64.tar.gz' --repo 8023/newRPS --clobber
tar -xzf newRPS-2.1.22-linux-amd64.tar.gz

# 重启（不要 docker compose down -v）
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

## 后台

- 入口：`/admin` 或 `#admin`，或 `Ctrl/Cmd+Shift+A`
- `config/default.json` 入库；`config/active.json` 运行时（gitignore）

## 构建与测试

```bash
npm run build:web      # 仅前端
npm run build:server   # 仅 Go
go test ./...
npm run test           # go test + 前端 build
```

## 维护约定

功能变更时同步更新「最近更新记录」。同步 GitHub 需 `git commit` 与 `git push`。

## 最近更新记录

### 2026-07-13

- **房间信息标签**：`config/default.json` 补回 `roomInfoTags`（中文名+配色）；加载时用 default 补齐旧 `active.json` 缺失项；前端 key 缺失时回退中文默认名。
- **Safari 房间布局**：`.room-layout` 改为 `margin: -8px auto`，避免负边距取消水平居中。
- **加密房密码框**：`stripHasFlags` 误删业务字段 `hasPassword`，大厅不显示密码输入；改为仅剥离「有同伴字段」的 protobuf presence 标记。
- **代码审计加固**：RPC 先登记 pending 再 send；reply 编码失败仍回错误；handler panic recover；`players` 通道 resync 改 RAW batch；normalize 补齐 legalMoves 坐标 0 / 棋盘 pad；房间 `updatedAt` 缺失不丢更新；惩罚提交 UI 与数组空值防护。
- **状态增量（方案 A）**：Protobuf `StateDelta`（路径 + Value）；合并后树 CRC-32 校验，失败 `sync:full`；ops 过多/过大回退 FULL；保留 debounce / player:batch / chat:append。
- **黑白棋坐标**：protobufjs `toObject({defaults:false})` 会丢掉 `row/col=0`，合法手与棋盘需 `?? 0` 归一化并 pad 成 8×8。
- **全链路 Protobuf**：线路无 JSON 文本；FULL 为类型化 `StateDocument`，DELTA 为 Value 补丁。
- **文档（D3/D4）**：修正 WebSocket 压缩说明（UA 分流）；协议章节同步为纯 protobuf。
- **产物策略（D1）**：`bin/server` + `dist/` 不入库，由 GitHub Release 附件分发（自 v2.1.22 起）。
- **前端结构（A1）**：`App.tsx` 瘦身为壳；`lib/` + `ui/AppViews.tsx`；协议编解码 `wire.ts` + `gen/proto.js`。
- **handlers guard（B1）**：`requirePlayer` / `requireRoomPlayer` 等。
- **null 兜底（B5）** / **结算抽象（A3/A4）**：见前序提交说明。

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
- **配置加载**：去掉 `config.go` 内嵌默认文案，只认 `config/default.json`（及运行时 `active.json`）；缺文件/校验失败直接报错。

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
- **Docker 部署**：`debian:bookworm-slim` + 挂载 `bin/server` / `dist/`（从 Release 解压或本机构建）；compose / config 仍来自仓库。
