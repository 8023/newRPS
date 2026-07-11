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
│   ├── src/             # 页面、WS 客户端、样式
│   ├── index.html
│   ├── package.json
│   └── vite.config.ts   # 构建输出到仓库根 dist/
├── dist/                # 前端构建产物（Go 静态托管，gitignore）
├── bin/                 # go build 产物（gitignore）
├── data/                # players.json 等（gitignore）
├── work/                # uploads、session.secret（gitignore）
├── go.mod
├── package.json         # 根脚本（并发 dev / 一键 build）
└── README.md
```

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

## WebSocket 协议

二进制 **Protobuf** 信封（`api/proto/wire.proto`），启用 permessage-deflate。

| 类型 | 说明 |
|------|------|
| FULL / DELTA | 状态通道 `lobby` / `room:*` / `config`；DELTA 带路径补丁 + SHA-256 |
| RAW | RPC 请求/响应与即时推送（chat、player:batch 等） |
| `sync:full` | 客户端哈希不一致时请求全量 |
| `player:get` | 拉取完整 `PublicPlayer`（大厅仅下发精简 `LobbyPlayer`） |

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
