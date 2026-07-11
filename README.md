# 抖喵游戏屋（Go 后端）

实时联机小游戏平台。前端 React + TypeScript，后端已 **1:1 重写为 Go**（标准库 `net/http` + `github.com/coder/websocket`）。

主玩法：锤子剪刀布 / 黑白棋 / 井字棋。含大厅、房间、聊天、观战、Bot、排位、惩罚、名字争夺战、白给、极限模式、后台配置等。

## 本地运行

### 方式一：一体启动（推荐）

```bash
# 编译后端
go build -o bin/server ./cmd/server

# 构建前端并启动（默认端口 9988）
npm install
npm run build
HOST=127.0.0.1 PORT=9988 ./bin/server
```

打开：`http://127.0.0.1:9988`

### 方式二：开发热更新

```bash
# 终端 1：Go 后端
HOST=127.0.0.1 PORT=9988 go run ./cmd/server

# 终端 2：Vite 前端（代理到 9988）
npm run dev:client
```

- 前端开发地址：`http://127.0.0.1:5173`
- 后端地址：`http://127.0.0.1:9988`

## 项目结构

```
cmd/server/          # 入口
internal/config/     # 配置加载/校验/持久化（config/default.json → active.json）
internal/server/     # 游戏状态、房间、WebSocket、HTTP
internal/types/      # 前后端共享的 JSON 契约类型
src/client/          # React 前端（原生 WebSocket 客户端）
src/shared/types.ts  # 前端类型定义
config/              # 默认/运行时配置
work/uploads/        # 证明图与后台图片
data/players.json    # 持久玩家档案
dist/                # 前端构建产物（由 Go 静态托管）
```

## WebSocket 协议

替代原 Socket.IO，JSON 信封：

| 方向 | 格式 |
|------|------|
| 客户端请求 | `{"e":"player:join","id":1,"d":{...}}` |
| 服务端应答 | `{"id":1,"d":{...}}` 或 `{"id":1,"err":"错误"}` |
| 服务端推送 | `{"e":"lobby:update","d":{...}}` |

连接：`GET /ws?token=<sessionToken>`，已启用 **permessage-deflate 压缩**。

事件名与原 Node 项目保持一致（`player:join`、`room:create`、`room:move`、`othello:*`、`tictactoe:*`、`punishment:*` 等共 42 个请求事件）。

## HTTP API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/session` | 签发会话 token |
| POST | `/api/proof-image` | 证明图：仅 `.webp`，≤2MB |
| POST | `/api/admin-image` | 后台图：jpg/png/webp |
| GET | `/api/config/export` | 导出配置（需管理员口令） |
| GET | `/ws` | WebSocket |
| GET | `/uploads/*` | 上传文件 |
| GET | `/*` | 前端静态资源 |

### 证明图上传规则

**前端**

1. 选文件  
2. 长宽比 > 21:9 → 拒绝  
3. 原图 > 10MB → 拒绝  
4. 像素 > 4MP → 等比缩放至约 4MP  
5. Canvas 绘制 → WebP 质量 85%  
6. 上传 `/api/proof-image`

**后端**

1. 后缀非 `.webp` → 拒绝  
2. 大小 > 2MB → 拒绝  
3. 内容非有效 WebP → 拒绝  

错误信息对用户可见（JSON `message` / 前端 `notice`）。

### 静态缓存

带 hash 的构建资源：`Cache-Control: public, max-age=31536000, immutable`  
`index.html`：`no-cache`

## 环境变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `HOST` | `127.0.0.1` | 监听地址 |
| `PORT` | `9988` | 监听端口 |
| `ADMIN_PASSWORD` | （空） | 后台口令 |
| `SESSION_SECRET` | 见说明 | 会话 HMAC 密钥；未设置时读写 `work/session.secret`，避免每次重启令牌全失效 |
| `SESSION_TTL_MS` | 24h | 会话有效期 |
| `ALLOWED_ORIGINS` | 本机 | 逗号分隔额外 Origin |
| `LOBBY_BROADCAST_DELAY_MS` | 300 | 大厅广播合并 |
| `ROOM_BROADCAST_DELAY_MS` | 60 | 房间广播合并 |

## 后台

- 入口：`/admin` 或 `#admin`，或 `Ctrl/Cmd+Shift+A`
- 配置文件：`config/default.json`（入库）、`config/active.json`（运行时，gitignore）

## 构建与测试

```bash
npm run build          # 前端
go build -o bin/server ./cmd/server
go test ./...
```

## 维护约定

功能变更时同步更新本 README「最近更新记录」。需要同步 GitHub 时执行 `git commit` 与 `git push`。

## 最近更新记录

### 2026-07-11

- 后端由 Node.js/Express/Socket.IO **完整重写为 Go**（`net/http` + `coder/websocket`），业务 1:1 复刻：锤子剪刀布 / 黑白棋 / 井字棋、大厅房间、惩罚、排位倍率与极限、白给、名争、Bot、管理后台与配置热更新等。
- 实时通道改为原生 WebSocket JSON 信封（替代 Socket.IO），启用 **permessage-deflate 压缩**；事件名与原 42 个请求事件及推送事件对齐。
- 证明图上传链路调整：前端校验长宽比 21:9、原图 10MB、像素约 4MP 缩放后 WebP 85%；后端仅接受 `.webp` 且 ≤2MB，错误信息对用户可见。
- 静态资源 `Cache-Control: public, max-age=31536000`；`index.html` 不缓存；默认监听端口 **9988**。
- 修复创建房间白屏：Go `nil` 切片序列化为 JSON `null` 导致前端 `.includes`/`.map` 崩溃；新增出站 `jsonsafe` 清洗，集合字段统一输出 `[]` 而非 `null`。
- 前端入口增加房间/配置/对局记录 normalize 兜底；WebSocket 握手失败时上报 `SESSION_INVALID` 以便清 token 重签。
- 会话密钥默认落盘 `work/session.secret`，减少进程重启后全站重连鉴权失败。
- 前端脚本与代理改为对接 Go 服务；依赖中移除 express/socket.io/multer 等 Node 服务端包。
