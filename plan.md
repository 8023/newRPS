# 抖喵游戏屋 · 前端 React → Svelte 重构记录

> 状态：**已完成（P0–P11，2026-08-29）**
> 目标框架：**Svelte 5（runes）+ 纯 Vite**（不引入 SvelteKit）
> 适用范围：仅 `web/` 目录。Go 后端、Protobuf 协议、SQLite、配置文件、部署形态**全部不动**。

> 本文保留迁移前的方案、风险和验收清单，并在文末记录实际落地结果。它是工程复盘文档，不是运行时配置；若与代码不一致，以代码、`README.md` 和 `CHANGELOG.md` 为准。

---

## 0. 目标与非目标

### 目标

1. 把 `web/` 的视图层从 React 19 完整替换为 Svelte 5，行为与现网**逐像素、逐交互一致**。
2. 迁移期间与迁移后，构建产物形态不变：仍然输出到 `bin/dist/`，仍由 `bin/server` 单二进制静态托管，Release 打包方式不变。
3. 迁移后彻底移除 `react` / `react-dom` / `@types/react*` / `@vitejs/plugin-react` / `lucide-react` / `recharts` 依赖。

### 非目标（本次明确不做）

- **不改协议**：`api/proto/*.proto`、`internal/wire/`、`internal/pbconv/`、`web/src/wire.ts`、`web/src/delta.ts` 在整个迁移窗口内**冻结**。这样任何行为差异都能被证明是"前端翻译错了"，而不是协议漂移。
- **不改后端**：`internal/` 一行不碰。
- **不改样式**：`web/src/styles.css`（9185 行全局 CSS）逐字保留，class 名不改。
- **不做"顺手优化"**：翻译阶段禁止改交互、改文案、改布局、修 bug（含 `MEMORY.md` 里记的 ChatPanel 1Hz 定时器缺陷）。所有已知待办另开 issue，迁移完成、回归通过之后再做。理由见 §8.1。
- **不引入 SSR / 路由库 / 状态管理库**：当前是无 SSR 需求、由 Go 托管静态产物的纯 SPA，路由是 `useState<AppView>` + `#admin` hash 的手写极简实现，没有必要在迁移中顺带上 SvelteKit 或 store 库。

---

## 1. 选型决策与理由

| 决策 | 选择 | 理由 |
|---|---|---|
| 框架版本 | **Svelte 5（runes 模式）** | `$state`/`$derived`/`$effect` 与现有 `useState`/`useMemo`/`useEffect` 近乎一一对应，翻译心智负担最低；Svelte 4 的 `$:` 响应式语句反而与 React 的心智模型差得更远。 |
| 构建工具 | **纯 Vite + `@sveltejs/vite-plugin-svelte`** | 现有 `vite.config.ts`（自定义 `markdownHtml` 插件、`libheif-js` 的 node 内建 alias、`__APP_BUILD_ID__` define、`outDir: ../bin/dist`）可以整份保留，只换一个插件。SvelteKit 会强加 adapter/路由/SSR 一整套约定，对本项目是纯负担。 |
| 类型检查 | **`svelte-check`** 替代 `tsc --noEmit` | `tsc` 无法解析 `.svelte`。根目录 `npm run test` 依赖 `npm run build --prefix web` 作为类型门禁，必须保证替换后门禁强度不下降。 |
| 图标 | **`@lucide/svelte`**（或对应版本的 `lucide-svelte`） | 全站仅 6 处 import、约 23 个图标名，同名一比一替换。⚠️ 动工时先核对包名与图标名是否都存在（尤其 `ChessKnight`），不存在的用内联 SVG 兜底。 |
| 图表 | **LayerChart（Svelte 原生）** | 见 §6.3，这是全项目唯一"无法机械翻译"的部分。 |
| 组件通信 | **函数 props**（`onError`、`onGoRoom` 等原样传） | Svelte 5 的 `$props()` 与 React props 语义几乎一致。**禁止**改用 `createEventDispatcher`（Svelte 5 已弃用），也不要中途改成 context——保持一一对应才好 review。 |
| 迁移方式 | **单分支整体重写 + 分阶段双入口对照** | 迁移期双入口，收尾后只保留 Svelte 入口，见 §3。 |

---

## 2. 现状盘点（迁移基线）

`web/src` 共 34,088 行 TS/TSX。按"要不要翻译"分三层：

### 2.1 A 层 · 逐字保留，与框架完全无关（约 4,700 行手写 + 15,643 行生成）

| 文件 | 行数 | 说明 |
|---|---|---|
| `gen/proto.d.ts` / `gen/proto.js` | 15,643 + | pbjs 生成，禁止手改 |
| `shared/types.ts` | 1,044 | `internal/types` 的前端镜像 |
| `wire.ts` | 889 | Protobuf 编解码 |
| `ws.ts` | 421 | `GameSocket` 类，自造 `on/off/emit` + FULL/DELTA 通道 + 重连心跳 |
| `delta.ts` | 97 | 与 `internal/delta` 镜像的 `applyOps`/`crc32Hex` |
| `lib/normalize.ts` | 724 | 服务端↔客户端形状协调 |
| `lib/session.ts` | 268 | 身份/token/localStorage |
| `lib/pushNotify.ts` | 214 | Notification + Service Worker |
| `lib/imagePipeline.ts` | 207 | webp/heif 转码 |
| `lib/analytics.ts` | 149 | 埋点上报 |
| `lib/constants.ts` / `avatarImage.ts` / `proofImage.ts` / `format.ts` / `rpc.ts` / `types.ts` | ~350 | 纯逻辑 |
| `fingerprint.ts` | 53 | FingerprintJS 封装 |
| `ui/contributeSeries.ts` / `ui/seriesFaction.ts` | 294 | 纯业务逻辑，已与 UI 分离 |
| `styles.css` | 9,185 | 全局 CSS，class 名全站共享 |
| `public/` | — | `icon.svg` / `manifest.webmanifest` / `push-sw-v3.js` |
| `vite-plugins/markdownHtml.ts` | — | `help.md?html` 构建期编译插件 |

**关键结论：整个通信层、状态同步层、业务逻辑层不含任何 React API**，`socket.on("room:update", fn)` 这种 EventEmitter 风格接口对 Svelte 同样自然。这是本次重构风险可控的最大依据。

### 2.2 B 层 · 需要薄适配（约 240 行）

| 文件 | 需要改的部分 |
|---|---|
| `lib/chatStore.ts` | 本体是模块级 `Map` + 订阅者 `Set` 的手写 store，**逐字保留**；只有末尾 `useChat()` 的 `useSyncExternalStore` 包装需换成 Svelte store（`subscribe` 契约天然兼容，改动约 10 行） |
| `lib/uiHelpers.ts` | 纯函数保留；`useMobileCollapse()` 改写成 runes 版（约 20 行） |
| `ui/AppViews.tsx` 内的 `useNow` / `useServerNow` | 提取为独立 `.svelte.ts` 模块的 runes 版（约 25 行） |
| `ui/PetBondGraphPanel.tsx` 内的 `usePetBondSimulation` | 力导向模拟本体全部跑在 ref 里，本就是命令式代码，剥成纯 TS 工厂函数即可 |
| `main.tsx` | 换成 `main.ts` + `mount(App, { target })` |

### 2.3 C 层 · 需要逐行翻译（约 13,200 行，本次工作量主体）

| 文件 | 行数 | 内含组件数 | 备注 |
|---|---|---|---|
| `ui/AppViews.tsx` | 5,266 | **60+** | 登录/大厅/创建房间/房间/聊天/座位/惩罚证明/排行榜/个人设置/认宠/抽奖/抢名，全在一个文件里 |
| `ui/AdminViews.tsx` | 2,107 | 10 | `AdminPanel` 单个函数就占 1,330 行，内含 11 个分区 |
| `ui/AnalyticsPanel.tsx` | 1,285 | ~8 | 重度依赖 recharts |
| `App.tsx` | 643 | 1 | 外壳 + 全局 socket 事件订阅 + 手写路由 |
| `ui/AdminContributionReview.tsx` | 573 | 6 | |
| `ui/PetBondGraphPanel.tsx` | 543 | 2 | 力导向图 + 拖拽 + SVG |
| `ui/ContributeView.tsx` | 496 | 1 | |
| `ui/ChessPanel.tsx` | 419 | 3 | |
| `ui/JunglePanel.tsx` | 384 | 3 | |
| `ui/GomokuPanel.tsx` | 254 | 2 | |
| `ui/StepEditor.tsx` | 234 | 1 | |
| `ui/LiarsDicePanel.tsx` | 230 | 1 | |
| `ui/CoinFlipPanel.tsx` | 156 | 2 | |
| `ui/NumberField.tsx` | 149 | 3 | 受控数字输入，语义最微妙 |
| `ui/ContributeSeriesForm.tsx` | 145 | 1 | |
| `ui/ContributionListControls.tsx` | 120 | 2 | |
| `ui/ContributionPreview.tsx` | 91 | 1 | |
| `ui/ContributionVote.tsx` | 57 | 1 | |
| `ui/HelpPanel.tsx` | ~20 | 1 | 唯一的 `dangerouslySetInnerHTML` |

### 2.4 现有测试（约 380 行）

| 文件 | 性质 | 迁移影响 |
|---|---|---|
| `ui/contributeSeries.test.ts` | 纯函数单测 | ✅ 零改动 |
| `ui/seriesFaction.test.ts` | 纯函数单测 | ✅ 零改动 |
| `ui/ContributionVote.test.ts` | 一半纯函数、一半 `readFileSync` 读源码断言 | ⚠️ 后半段必须重写 |
| `ui/lobbyActions.test.ts` | **全部**是 `readFileSync` 读 `AppViews.tsx`/`AdminViews.tsx`/`App.tsx` 源码文本、用 `indexOf(">参与共建<")` 断言 JSX 里按钮的**前后顺序** | ⚠️ 全部必须重写（路径与标记形态都变了） |
| `ui/contributeView.test.ts` | 同上，读 `ContributeView.tsx`/`ContributeSeriesForm.tsx`/`StepEditor.tsx` 源码断言 | ⚠️ 全部必须重写 |

**全项目没有任何组件级测试**（无 `@testing-library`、无 jsdom、无 e2e）。这是本次重构最大的隐性风险：翻译错了没有自动化手段能发现。应对见 §7。

---

## 3. 迁移策略与实际落地：单分支重写 + 分阶段对照

### 3.1 为什么不做"渐进式共存"

React 与 Svelte 混跑需要两套运行时同时加载 + 状态桥接层，对这个体量（单 SPA、无微前端边界）是净负担。且 `App.tsx` 持有几乎全部全局状态（config/lobby/room/me），无法在中途沿组件边界劈开。

### 3.2 实际落地方式

在 `feat/svelte-migration` 分支上原地重写，迁移期间保留 React 入口做实时对照，收尾时统一切换为 Svelte：

```
web/index.html          → <script src="/src/main.ts">        (Svelte，新)
web/index-react.html    → <script src="/src/main.tsx">       (React，仅迁移期保留)
```

- 迁移期的 Vite 配置曾同时声明两个入口；两套代码共用同一份 `styles.css`、同一份 A 层模块、同一个 dev 后端，P11 收尾后恢复为单一入口。
- 迁移期任一时刻，`http://127.0.0.1:5173/` 是 Svelte 版，`http://127.0.0.1:5173/index-react.html` 是 React 版，可以两个标签页并排对照同一个房间、同一份数据——**这是翻译类重构最有效的验收手段**，比读代码回忆行为可靠得多。
- P11 已删除 `index-react.html`、所有 `.tsx`、React 运行时及其依赖，生产入口统一为 Svelte；原有 `styles.css`、WebSocket/Protobuf/DELTA 层、`bin/dist/` 输出路径和 Go 静态托管方式保持不变。

### 3.3 分支与提交纪律

- 一个组件（或一组强耦合的小组件）一个提交，commit message 注明来源，例如
  `refactor(web): 迁移 ChatPanel/ChatBubble/ChatAvatar 到 Svelte（源 AppViews.tsx:2963-3180）`
- 禁止"翻译 + 顺手改逻辑"混在同一提交里。要改，先单独提交一个 React 侧的行为变更（或记为 TODO），再翻译。
- **迁移窗口内冻结 `web/` 的功能开发**。`main` 目前仍在活跃提交（v2.8.3 刚发），若无法完全冻结，必须维护一份 `MIGRATION-PORT.md` 记录窗口期内 main 上每一笔前端改动，收尾前逐条补回 Svelte 侧。这是本方案第二大风险（见 §9）。

---

## 4. 前置准备：Phase 0（建议先在 React 侧完成，独立有价值）

这一阶段**不写一行 Svelte**，纯粹是行为不变的 React 侧整理，目的是把 5,266 行的 `AppViews.tsx` 拆小、把非组件代码剥离，让后续翻译的每一片都足够小到能被完整审阅。即使重构中途放弃，这部分收益也保留。

1. **提取纯函数到 `lib/`**：`connectionStateText`、`phaseText`、`mentionLabel`、以及散落在 `AppViews.tsx` 里的各类格式化/判定辅助函数，全部移入 `lib/*.ts`。理由：Svelte 组件文件不适合承载大量导出函数（虽然 `<script module>` 可以，但会让 diff 更难读）。
2. **提取自定义 hook 的无框架内核**：`usePetBondSimulation` 的模拟算法、`useNow` 的时钟逻辑，抽成不依赖 React 的模块。
3. **按功能域拆分 `AppViews.tsx`**，建议目标结构（同时也是最终 Svelte 的目录结构）：
   ```
   ui/shell/      Login, SecurityDisclaimer, PlayerBadge, PlayerAvatar, ModeChip, GiveawayChip, CollapseToggle, Toggle, Select, Stat
   ui/lobby/      Lobby, CreateRoom, GameTimerSettings, TagPicker, RoomTagList, RoomVersusLine, RoomVersusSeat, RankMultiplierBadge, ExtremeRankedBadge
   ui/room/       Room, SeatView, SeatStatsView, Settlement, RoundHistoryCard, RoomPlayerRow, OfflineBadge, ProofImage, PunishmentStatus, GameClockBar, RoomInfoTagList
   ui/games/      TicTacToePanel, OthelloPanel, OthelloScore, OthelloSettlementCard, TicTacToeScore, GomokuPanel, JunglePanel, ChessPanel, LiarsDicePanel, CoinFlipPanel
   ui/chat/       ChatBoardShell, ChatPanel, ChatBubble, ChatAvatar, ChatName
   ui/profile/    ProfilePanel, ClaimKeyPanel, GenderSelect, FactionSelect, GenderSelectControl
   ui/social/     Leaderboard, GlobalLeaderboardPanel, LeaderboardExtra, PetBondPanel, PetBondGraphPanel, PetBondPlayerInfo, GiveawayPanel, UniversalRenamePanel
   ui/about/      AboutPanel, HelpPanel
   ui/contribute/ (已经基本拆好，保持)
   ui/admin/      (按 AdminSection 的 11 个分区拆)
   ```
4. **统一 `socket` 的 import 来源**：目前 `App.tsx` 写的是 `import { socket } from "./main"`，而 `main.tsx` 只是 `export { socket } from "./ws"` 的中转，其余文件都直接从 `../ws` 导入。迁移前统一成一律 `from "./ws"`，消除这条绕路依赖（否则新入口 `main.ts` 还得为它保留一个莫名其妙的 re-export）。
5. **收敛 `socket.off(event)` 的无参调用**：`App.tsx` 的 cleanup 用的是 `socket.off("connect")`（不传 handler = 清掉该事件的**所有**监听）。`lib/chatStore.ts` 顶部有一条注释明确说"因为 App 会清空所有 connect 监听，所以本模块不自注册 connect"——这是一处隐式的跨模块耦合。迁移前改成一律 `socket.off(event, handler)` 精确摘除，并把 chatStore 的注释同步更新。**不改的话，Svelte 侧组件挂载/卸载时序一变，很容易踩到"某个模块的 connect 监听被别人清掉了"这种极难定位的问题。**

> Phase 0 完成后应当满足：`go test ./... && npm run build --prefix web` 全绿，且现网行为零变化。这是一个可以独立合入 `main` 的提交序列。

---

## 5. 分阶段迁移路线

依赖自底向上，每一阶段结束时 Svelte 入口都能真实连上本地 Go 后端跑到该阶段覆盖的功能。

| Phase | 内容 | 产出验收 |
|---|---|---|
| **P0** | §4 的 React 侧整理 | `main` 上行为零变化，构建全绿 |
| **P1** | 构建链路：装 `svelte` + `@sveltejs/vite-plugin-svelte` + `svelte-check`；改 `vite.config.ts`（双入口、保留 markdownHtml/alias/define）；`main.ts` + `App.svelte` 空壳；`svelte.config.js`；`tsconfig` 调整 | `npm run dev:web` 打开 Svelte 空页面，能 `import "./styles.css"`；`index-react.html` 仍完整可用 |
| **P2** | B 层适配：chatStore 的 Svelte store 出口、`useNow`/`useServerNow`/`useMobileCollapse` 的 runes 版 | 单测（纯函数部分）全绿 |
| **P3** | 外壳：`App.svelte`（全局 socket 订阅、手写路由、toast、公告、版本横幅）+ `SecurityDisclaimer` + `Login` + `ClaimKeyPanel` + `PlayerBadge`/`PlayerAvatar` | **能登录**、能恢复会话、能被顶号踢下线、免责声明每日一次生效 |
| **P4** | 大厅：`Lobby` + `CreateRoom`（760 行，含惩罚标签三态、系列选择、棋钟设置）+ 各类房间卡片小组件 | 能看到房间列表、能建房、标签偏好 localStorage 读写正常 |
| **P5** | 房间外壳 + 聊天：`Room`（670 行）+ `SeatView` + `Settlement` + `RoundHistoryCard` + `ProofImage` + `PunishmentStatus` + `ChatPanel` 全家桶 | 能进房、坐下、准备、聊天翻页、上传惩罚证明并审核 |
| **P6** | 8 个游戏面板：RPS（在 `Room` 内）、TicTacToe、Othello、Gomoku、Jungle、Chess、LiarsDice、CoinFlip + `GameClockBar` | 每款游戏能完整打完一局，含棋钟/悔棋/认输/逃跑/白给分支 |
| **P7** | 个人与社交：`ProfilePanel`（550 行）+ `GlobalLeaderboardPanel` + `LeaderboardExtra` + `PetBondPanel` + `PetBondGraphPanel` + `GiveawayPanel` + `UniversalRenamePanel` + `AboutPanel` + `HelpPanel` | 改资料/头像、主题切换、推送开关、认宠关系图拖拽、抽奖、抢名 |
| **P8** | 共建投稿：`ContributeView` + `ContributeSeriesForm` + `StepEditor` + `ContributionPreview` + `ContributionListControls` + `ContributionVote` | 投稿草稿/提交/编辑/撤回全流程、封面图上传 |
| **P9** | 后台：`AdminPanel` 11 个分区 + `AdminContributionReview` + `AdminPlayerEditor` + 各编辑器小组件 | 11 个分区逐个走查 |
| **P10** | 数据分析图表（见 §6.3，独立最大不确定项，故排最后） | 与 React 版并排比对每张图的数值与形态 |
| **P11** | 收尾：删 `.tsx` 与 `index-react.html`、卸载 React 系依赖、重写源码断言型测试、更新 `README.md` / `CLAUDE.md` / 根 `package.json` description、Release 验证 | `npm run test` 全绿；打包解压后目录结构符合 `CLAUDE.md` 的 Release 规范 |

---

## 6. 逐项技术要点

### 6.1 React → Svelte 5 语义映射表

| React | Svelte 5 | 陷阱 |
|---|---|---|
| `useState(x)` | `let x = $state(init)` | 大型不可变快照（`config`/`lobby`/`room`）请用 **`$state.raw`**，见 §6.2 |
| `setX(v => ...)` 更新式 | 直接 `x = f(x)` | Svelte 无批处理队列，读到的永远是最新值，比 React 更直观 |
| `useMemo(fn, deps)` | `let y = $derived.by(fn)` | **依赖是自动追踪的，不是手写数组**。React 里"故意写窄的 deps"会变成完全不同的行为，见 §6.2 |
| `useEffect(fn, deps)` | `$effect(fn)` | 同上；返回值仍是清理函数 |
| `useLayoutEffect` | `$effect.pre` | 本项目暂无使用 |
| `useEffect(fn, [])` 挂载一次 | `$effect(() => { ... })` + `untrack`，或直接写在 `<script>` 顶层 + `onMount` | 纯挂载副作用（注册 socket 监听）建议用 `onMount`，语义最接近且不会被意外追踪 |
| `useRef(v).current`（可变盒子） | 普通 `let v`（**不加 `$state`**） | Svelte 组件的 `let` 本身就是每实例私有的可变槽 |
| `useRef<HTMLElement>(null)` | `bind:this={el}` | |
| `viewRef.current = view` 读最新值套路 | **直接删掉**，闭包里直接读变量即可 | Svelte 没有 stale closure 问题，这类 ref 全部可以消失（`App.tsx` 里有 `viewRef` / `stickRef` / `graphRef` / `widthRef` 等多处） |
| `useSyncExternalStore(sub, get)` | Svelte store 契约（`{ subscribe }`）+ `$store` | `chatStore` 已是标准 subscribe 模型，改造极小 |
| `children` / `ReactNode` prop | `{#snippet}` + `{@render}` | 项目里有 6 处具名 ReactNode prop：`tabs`、`table`、`rightOfPreview`、`stepActions`、`extraSubmitActions`、`children` |
| **返回 JSX 的普通函数**（`gameIcon()`） | 改成 snippet 或小组件 | Svelte 里函数不能返回标记，必须结构性改写。`AppViews.tsx:1445 gameIcon()` 是唯一一处 |
| `lazy()` + `<Suspense>` | `{#await import("./X.svelte") then M}<M.default … />{/await}` | 仅 `AdminPanel` 一处 |
| `dangerouslySetInnerHTML` | `{@html helpHtml}` | 仅 `HelpPanel.tsx:15` 一处 |
| `<>…</>` Fragment | 不需要（Svelte 模板天然多根） | `ChatPanel` / `AnalyticsPanel` 各一处 |
| `key={id}` | `{#each list as item (item.id)}` | **必须补 key**，尤其 `ChatBubble` 列表与棋盘格子，否则 DOM 复用错乱 |
| `className` | `class` | |
| `htmlFor` / `tabIndex` / `readOnly` / `autoFocus` | `for` / `tabindex` / `readonly` / `autofocus` | |
| `onClick` / `onKeyDown` / `onScroll` | `onclick` / `onkeydown` / `onscroll` | Svelte 5 事件属性，全小写 |
| **`onChange` on `<input type=text/number>`** | **`oninput`** | ⚠️ **最容易踩的坑**：React 的 `onChange` 实际绑定的是 DOM `input` 事件（每次按键都触发）；Svelte 的 `onchange` 是原生 change（失焦才触发）。文本/数字/textarea 一律映射成 `oninput`；`<select>`/checkbox/radio 用 `onchange` |
| 受控输入 `value={x}` | 显式 `value={x}` + `oninput` 回写，**或** `bind:value` | 见 §6.4，`NumberField` 必须用前者 |
| SVG 属性 `strokeWidth` | `stroke-width` | `PetBondGraphPanel` 大量 SVG |
| `React.StrictMode` 双调用 | 无对应 | 见 §6.5 |

### 6.2 头号语义陷阱：`$effect`/`$derived` 是自动依赖追踪

React 的 deps 数组是**手写的**，项目里存在多处"deps 故意比实际读取的变量窄"的写法。直译成 `$effect` 会因为自动追踪而**多触发**，轻则性能退化，重则死循环。必须逐个识别并用 `untrack()` 显式排除。

已识别的高危点：

1. **`App.tsx:487-490`**
   ```js
   useEffect(() => {
     if (!me || isAdminRoute()) return;
     ask(view === "room" ? "lobby:unsubscribe" : "lobby:subscribe", {});
   }, [view, me?.player.id]);
   ```
   deps 是 `me?.player.id`，但函数体读的是整个 `me`。`me` 每秒可能因 `player:batch` 被整对象替换多次——直译会导致**每次玩家资料变动都往服务端发一次 lobby 订阅/退订 RPC**。必须写成只追踪 `view` 与 `me?.player.id`，其余 `untrack`。

2. **`App.tsx:492-511`**（effect 依赖 `[lobby, me]`，函数体内又 `setMe`）
   这是"读 me → 写 me"的自反 effect。React 靠 `playerSyncKey` 比较收敛。直译成 `$effect` 同样会收敛，但只要有人以后动了比较逻辑就会变成无限循环。**建议改写成显式的、由 `lobby` 单向驱动的 `$effect` + `untrack(me)`**，并在注释里写死"禁止在此 effect 内追踪 me"。

3. **`ChatPanel`（AppViews.tsx:3040）**
   ```js
   useEffect(() => { if (list && stickRef.current) scrollToBottomSoon(list); }, [visible.length]);
   ```
   只在**长度**变化时滚底。直译会因读了 `visible` 数组而在任何消息内容变化（如 `expiresAt` 倒计时导致重算）时都滚底。必须只追踪 `visible.length`。

4. **`NumberField.tsx:26-28`**
   ```js
   useEffect(() => { if (!focused) setText(String(value)); }, [value, focused]);
   ```
   同时依赖 `value` 与 `focused`，且写 `text`。直译需确保不追踪 `text` 本身，否则自触发。

5. **`useNow` / `useServerNow`**：deps 是 `[intervalMs, enabled]`，函数体写 `now`。runes 版必须保证 `now` 的写入不被自身追踪。

**执行要求：翻译每一个 `useEffect` 时，逐条对照 deps 数组与函数体实际读取的变量，把差集列出来，明确决定"追踪"还是 `untrack`。这一步不能凭直觉略过。**

### 6.3 数据分析图表（LayerChart）——唯一的非机械翻译项

迁移前的 `ui/AnalyticsPanel.tsx` 使用 `LineChart` / `BarChart` / `PieChart` / `ComposedChart` / `Cell` / `Rectangle`（自定义柱形）/ `ResponsiveContainer` / 自定义 `Tooltip` / `Legend`，共约 10+ 张图。Recharts 是纯 React 组件库，**无任何 Svelte 移植版**，因此整体改用 LayerChart 原语重画。

**实际方案：LayerChart（Svelte 原生、基于 d3 的公开原语）**

已发布包公开 `Chart`/`Svg`/`Axis`/`Bars`/`Spline`/`Arc`/`Pie`/`Group`/`Legend`/`ForceSimulation` 等原语，项目在 `lib/charts/*.svelte` 上封装折线、双轴、堆叠柱、组合图、横向柱、环图和 Sparkline。双 Y 轴用两个重叠坐标系模拟，堆叠柱顶端圆角使用 LayerChart 的 stack-top 判断；数据分析面板仍由后台动态导入，图表代码不会进入普通玩家首屏。

Vite 的 `manualChunks` 与 `lazySceneBoundaryGuard` 固定后台/分析懒加载边界；构建产物中如果首屏静态依赖这些场景会直接失败。

**执行要点：**
- 这部分**不是翻译而是重画**，必须逐图与 React 版并排比对：数据点数量、坐标轴刻度与格式、tooltip 内容、图例、配色（含 light/dark 主题变量）、空数据态。
- 该面板已经是 lazy chunk 且仅管理员可见，**排在最后一个 Phase**，不阻塞任何玩家侧功能上线。
- 若时间紧张，可接受的降级路径：P10 之前先用一个"图表加载中/暂不可用"的占位，把玩家侧先发出去。**但这是产品降级，需你明确拍板，不要默认执行。**

### 6.4 受控输入与 `bind:` 的取舍

React 的 `value={x}` 是**强受控**：每次渲染都把 DOM 值拗回 `x`。Svelte 的 `value={x}` 只在 `x` 变化时写入，用户输入不会被自动回滚。

- **`NumberField.tsx` 必须用显式 `value` + `oninput`/`onblur`，禁止 `bind:value`。** 它的核心价值就是"维护一层原始输入字符串 `text`，失焦时按 `revert`/`keep` 两种策略处理非法值"，`bind:value` 的双向绑定会把这层缓冲短路掉，直接退化回它当初就是为了修掉的"删空被拗回 0"的老问题。
- `<select>` 建议用 `bind:value`（Svelte 对 select 的 value 绑定处理最稳妥）。
- 普通文本框（聊天输入、昵称等）可以用 `bind:value` 简化，但**同一次提交里要么全用要么全不用**，别混。
- ⚠️ 已知平台行为：`MEMORY.md` 记录了"弹窗内原生 `<select>` 第一次点击自动收回"是 Windows Chrome 滚轮手势锁存所致、非本项目缺陷。迁移后如果复现，**不要误判成 Svelte 引入的回归**。

### 6.5 StrictMode 消失带来的影响

`main.tsx` 用了 `<React.StrictMode>`，开发环境下 effect 会双调用。`App.tsx:375` 有针对性注释：

```js
// StrictMode 重挂载或热更新时 connect 事件可能已错过，已连接则立刻恢复会话。
if (socket.connected) { setConnectionState("connected"); void restoreSession(...); }
```

Svelte 无 StrictMode，effect 只跑一次。**但这段代码不能删**：它同时也覆盖了"WS 在组件挂载前就已连上"这个真实竞态（`connectSocketWithSession()` 是在另一个 effect 里发起的，且重连不经过组件挂载）。翻译时原样保留，只把注释里的 "StrictMode" 改成 "挂载时机竞态"。

反过来，迁移后失去了 StrictMode 这层"副作用幂等性"体检，**注册/注销类副作用（socket 监听、定时器、订阅 RPC）的清理必须人工复查**，尤其 `ChatPanel` 的 `lobby:suggestions:subscribe/unsubscribe` 与 `PetBondPanel` 的 `petbond:update` 订阅。

### 6.6 `$state.raw` 与大型快照

`config` / `lobby` / `room` / `me` 是从 WS 的 FULL/DELTA 通道整棵替换下来的大对象（房间快照含 `roundHistory`、`chat`、`seatStats` 等深层结构）。Svelte 5 的 `$state` 会**深度 Proxy 化**整棵树，对这种"每秒可能整体替换若干次、且从不做细粒度就地修改"的数据是纯开销。

**要求：`config` / `lobby` / `room` / `me` / `leaderboardPlayersSnapshot` 一律用 `$state.raw`**，保持"整体替换触发更新"的不可变语义——这恰好也是现有 React 代码的写法（`setRoom(old => ({...old, ...}))`），语义完全一致，且性能更好。

只有真正需要就地改字段的局部表单状态才用普通 `$state`。

### 6.7 保持挂载但隐藏的视图（`keepContribute`）

`App.tsx:608-612` 有一处刻意设计：从"参与共建"页进后台时，用 `keepContribute` + `<div hidden>` 把 `ContributeView` **保持挂载**，这样从后台返回时草稿输入不丢。

翻译时**绝不能**简化成 `{#if view === "contribute"}`——那会销毁组件、清空用户正在编辑的投稿。必须保留同样的"外层 `{#if}` 判定是否需要存在 + 内层 `hidden` 属性控制可见性"两层结构。

### 6.8 路由与页面埋点

手写路由（`AppView` 联合类型 + `#admin` hash + `Ctrl/Cmd+Shift+A` 快捷键）逐条翻译即可。注意 `App.tsx` 有 5 个独立的 `trackPageview` effect（主视图 + profile/leaderboard/about/help 四个弹窗），翻译后必须保证**触发次数与时机完全一致**——埋点多打少打会直接污染后台数据分析面板的历史曲线。

### 6.9 CSS 与 Svelte 的作用域

- `styles.css` 是全站共享的全局样式表，**继续在 `main.ts` 里 `import "./styles.css"`**。
- **禁止**在 `.svelte` 组件里写 `<style>` 块。Svelte 会给组件内样式加 scope hash，而现有 class 名是跨组件复用的（`.panel`、`.soft-button`、`.chat-panel` 等），scope 化必然破坏级联。
- 因为样式不在组件内，Svelte 的"未使用 CSS 选择器"编译警告不会误报，这点是好事。
- 主题切换靠 `document.documentElement.dataset.theme`，与框架无关，原样保留。

### 6.10 构建配置改动清单

`web/vite.config.ts`：
- `plugins: [markdownHtml(), react()]` → `[markdownHtml(), svelte()]`（**保持 `markdownHtml` 在前**，它处理 `?html` 后缀导入）
- `define.__APP_BUILD_ID__` — 原样保留（与后端 `-ldflags` 注入的 git 短哈希对齐，是版本不一致横幅的依据）
- `resolve.alias` 的 `fs`/`path`/`crypto` → `node-empty-shim.ts` — 原样保留（`libheif-js` 需要）
- `build.outDir: "../bin/dist"` + `emptyOutDir: true` — **原样保留**，这是 Release 打包规范的硬约束
- `server.proxy` 的 `/api` `/uploads` `/ws` — 原样保留
- 新增：后台/数据分析/图片处理场景的懒加载分包与 `lazySceneBoundaryGuard` 构建期边界检查

`web/package.json`：
- `"build": "tsc --noEmit && vite build"` → `"build": "svelte-check --tsconfig ./tsconfig.json && vite build"`
- 已删 `react` / `react-dom` / `@types/react` / `@types/react-dom` / `@vitejs/plugin-react` / `lucide-react` / `recharts`
- 已增 `svelte` / `@sveltejs/vite-plugin-svelte` / `svelte-check` / `@lucide/svelte` / `layerchart` 及其 d3 依赖
- 保留 `protobufjs` / `protobufjs-cli` / `@fingerprintjs/fingerprintjs` / `@jsquash/webp` / `libheif-js` / `markdown-it` / `vitest` / `typescript` / `vite`
- `overrides` 段（postcss/minimatch/brace-expansion 的安全版本约束）保留

`web/tsconfig.json`：
- 删 `"jsx": "react-jsx"`
- 加 `"types": ["svelte"]`（或 extends `@tsconfig/svelte`）
- `"moduleResolution": "Node"` 建议升到 `"Bundler"`（Svelte 生态包多用 exports map）
- `include` 加 `"src/**/*.svelte"`

`web/index.html`：`<script type="module" src="/src/main.tsx">` → `/src/main.ts`

新增 `web/svelte.config.js`（`vitePreprocess()` 以支持 `<script lang="ts">`）。

**根 `package.json` 不需要改任何脚本**——`dev:web` / `build:web` / `test` 全部是对 `web` 的委托调用。只有 `description` 字段的 "Go 后端 + React 前端" 要在 P11 改掉。

---

## 7. 测试与验收策略

现状是**零组件测试**，翻译类重构又恰恰最容易在"某个 effect 的边界条件漏搬"上出事。三层应对：

### 7.1 保留与重写现有单测

- `contributeSeries.test.ts`、`seriesFaction.test.ts`：零改动，继续跑在 vitest 下（它们测的是纯函数）。
- `lobbyActions.test.ts`、`contributeView.test.ts`、`ContributionVote.test.ts` 的源码断言部分：**已重写**。这些测试现读取对应 `.svelte` 源码文本，并继续覆盖元素顺序、关键文案和迁移后的状态同步；Vitest 当前共 39 项通过。
  - 重写时改读对应 `.svelte` 文件即可，断言逻辑（元素顺序、某文案存在）基本可以平移。
  - **建议借这次机会把它们升级成真正的渲染断言**（见 7.2），源码文本断言本身是很脆的替代品。

### 7.2 新增组件冒烟测试（建议）

引入 `@testing-library/svelte` + `jsdom`（或 `happy-dom`），为下列高风险组件各写 3-5 条冒烟用例。这不是追求覆盖率，是为翻译提供一张最低限度的安全网：

- `NumberField` / `OptionalNumberField`：清空、输入非法值、失焦 revert vs keep、外部 value 变更时不打断正在输入 —— **这是全项目语义最微妙的组件，必须有测试**
- `ChatPanel`：滚动到顶触发 loadOlder 并保持视口位置、stick 到底逻辑、@提及插入与提交前过滤
- `useNow` 的 runes 版：`enabled=false` 时不起定时器（顺带把 `MEMORY.md` 里记的 ChatPanel 无条件 1Hz 缺陷钉住，避免翻译时把已有的 `hasExpiry` 守卫弄丢）

### 7.3 人工回归清单（真正的主力验收手段）

迁移期间利用 §3.2 的临时 React 入口与 Svelte 入口并排跑同一个本地后端逐项对照；P11 收尾后 React 入口已删除。清单至少覆盖：

**身份与会话**
- [ ] 首次访问注册身份 → 登录 → 刷新页面自动恢复
- [ ] 第二设备登录 → `alreadyOnline` 确认 → 顶号 → 原设备收到 `session:kicked`
- [ ] ClaimKey 跨设备认领
- [ ] 登出（`identity:logout`）后本地清理正确
- [ ] 断线重连后 `restoreSession` → `refreshActiveChats` 顺序正确（房间聊天不被 "你不在这个房间里" 拒绝）
- [ ] 连接认证失败重试上限（2 次）与 token 换发

**大厅与建房**
- [ ] 房间列表实时增删、在线人数、对战双方展示
- [ ] 建房：8 款游戏各建一次，房名/背景/标签/棋钟/排位倍率/极限模式
- [ ] 惩罚来源三态（random / series / player）各建一次
- [ ] 惩罚标签三态偏好写入 localStorage 且跨刷新保持
- [ ] 选系列时目标阵营不匹配的警告弹出

**房间与对局（8 款逐一）**
- [ ] RPS：出拳、即时结算、白给、放过对方
- [ ] 井字棋 / 黑白棋 / 五子棋 / 斗兽棋 / 国际象棋：落子、轮次、悔棋请求与应答、认输、逃跑判负
- [ ] 棋钟：每步倒计时、每局总时长、暂停/恢复、超时判负（黑白棋/五子棋/斗兽棋/象棋）
- [ ] 黑白棋结算卡（normal / giveaway / tribute 三选一）
- [ ] 大话骰：2-8 人、私有骰子只单播给自己（**开两个浏览器验证对方看不到**）、断线判负
- [ ] 猜硬币：单人惩罚流程、证明自动通过
- [ ] 每子惩罚触发（象棋/斗兽棋 `piece_capture`）

**惩罚与证明**
- [ ] 任务分配、占位符 `{loser}`/`{winner}` 替换
- [ ] 证明图上传（含大图缩放、HEIC 转码、>10MB 与 21:9 拒绝）
- [ ] 审核通过 / 驳回 / 重做
- [ ] 赞踩投票资格（受罚者与审核者各一次，重复投票被拒）
- [ ] 系列任务房内按玩家独立进度推进

**聊天**
- [ ] 大厅留言板与房间聊天双 scope
- [ ] 上翻加载更早 100 条 + 滚动位置保持
- [ ] @提及高亮、点头像插入 @
- [ ] 管理员软删除后本地即时摘除
- [ ] 断线重连补拉

**社交与个人**
- [ ] 个人设置：改名、改性别/阵营、头像上传、主题切换、推送开关与测试推送
- [ ] 全服排行榜各 tab + `LeaderboardExtra` 的倒计时
- [ ] 认宠：关系图渲染、节点拖拽、力导向稳定
- [ ] 抽奖、抢名大战
- [ ] 关于页 + 游戏说明弹窗（`help.md` 编译产物正确注入）

**后台（11 个分区逐一）**
- [ ] site / factions（性别与阵营整表覆盖保存）/ titles / punishments / contributions（审批全流程）/ roomTags / nameWar / giveaway / petBond / rooms（强制关房）/ analytics
- [ ] 玩家管理：编辑、踢人、自定义称号（黄色边框提示）、ClaimKey 揭示
- [ ] 数据分析：每张图的数值与 React 版一致

**全局**
- [ ] toast 进出场动画时序（3200ms leave / 3520ms clear）
- [ ] 全服公告自动消失
- [ ] 版本不一致横幅：同版本重复到达不重弹，新版本才重弹，"忽略"后不再弹
- [ ] 手机端各模块折叠状态记忆（localStorage `rps-collapse:` 前缀）
- [ ] light / dark 双主题全页面走查
- [ ] **iOS Safari 实机验证**（`permessage-deflate` 在 Safari 被服务端禁用，压缩路径不同，是历史事故高发点）

### 7.4 可选但强烈建议：Playwright 冒烟

如果愿意多投入 1-2 天，为"登录 → 建房 → 双人对局一轮 → 惩罚证明 → 审核"这条主干链路写一个 Playwright 脚本（双 browser context 模拟两个玩家）。这条链路串起了 WS、FULL/DELTA、身份、房间、惩罚五个子系统，**一个脚本能挡住绝大多数灾难性回归**，且迁移完成后长期有效。

---

## 8. 注意事项汇总

### 8.1 翻译期纪律（最重要）

- **一次只做一件事**：翻译就是翻译。看到烂代码想改、看到 bug 想修、看到重复想抽象——全部记进 `TODO-after-migration.md`，不要在翻译提交里动手。没有测试网的重写里，"顺手改"是回归的头号来源。
- **保留所有中文注释**。现有代码里大量注释记录的是踩过的坑（为什么 `updatedAt` 缺失时不能丢弃更新、为什么不自动刷新版本、为什么 chatStore 不自注册 connect、为什么棋钟用服务端时间轴）。这些注释的价值远高于代码本身，**逐条搬过去**，只在描述失效时（如 StrictMode）改措辞。
- **对照原文件行号**。每个 `.svelte` 文件头部写一行 `// 源：ui/AppViews.tsx:2963-3180`，直到 P11 删除 `.tsx` 为止，方便 review 与查错。

### 8.2 容易漏搬的具体点

- `App.tsx:220-239` 的 `room:update` 合并逻辑：`updatedAt` 倒序保护、`chat` 空数组时保留本地、`roundHistory` 必须 merge 不能覆盖。三个条件缺一不可。
- `App.tsx:252-289` 的 `applyPlayerPatches`：`avatarUrl` / `genderId` 在 LobbyPlayer 精简视图里会变成 `undefined`，必须**显式写回空值**否则旧头像残留。这个坑在 `me` 合并与 `lobby` 合并两处各出现一次。
- `App.tsx:492-511`：`LobbyPlayer` 是精简视图，**禁止整对象覆盖** `me.player`（会冲掉冷却等字段），必须走 `mergeLobbyPlayerIntoFullPlayer`。
- `ChatPanel` 的 `pendingMentionsRef`：发送前要过滤掉用户已从文本里删掉的 @。
- `ProfilePanel` / `AdminPlayerEditor` 里的图片上传前置处理（`prepareProofImageForUpload` / `prepareAvatarImageForUpload`）调用时机。
- 各游戏面板的 `useMemo` 派生（`ChessPanel` 有 6 个）——多是棋盘坐标/合法着法的计算，直译成 `$derived.by` 但要确认依赖范围。
- 惩罚证明图的懒加载组件 `ContributionVoteLazy`（AppViews.tsx:5241）。

### 8.3 部署与发版约束（迁移不得破坏）

- 构建产物必须仍在 `bin/dist/`，与 `bin/server` 同级。
- Release tar 包**绝不能多包一层父目录**，`bin/`、`docker-compose.yml`、`.env.example`、`config/`、空 `work/`、`README.md` 必须直接在 tar 根目录（见 `CLAUDE.md`）。
- GitHub Release 的 title 只写版本号。
- `npm run fix-perms` 的 `chmod 600 config/json/*.json` 不受影响。
- 前后端版本一致性提示依赖 `__APP_BUILD_ID__` 与后端 `-ldflags` 注入值相等——**迁移不能改变这个 define 的计算方式**，否则会全站误报"网站内容已更新"。

### 8.4 文档同步（P11 已完成）

- `README.md` 已切换为 Go + Svelte，并同步目录结构、LayerChart、`AdminPlayerEditor.svelte` 与 v3.1.1 更新记录。
- `CLAUDE.md` 的前端架构说明已按新目录结构更新；之后维护时继续以 `web/src/` 的实际分层为准。
- 根 `package.json` 的 `description` 已切换为 Go 后端 + Svelte 前端。
- `CHANGELOG.md` 已加入 v3.1.1 迁移、构建与验证记录。
- `help.md` 的玩家规则无需因技术栈迁移改写，但关于页提示已更正为覆盖八款游戏；构建仍需验证 `markdownHtml` 插件正常注入。

---

## 9. 风险登记

| # | 风险 | 影响 | 缓解 |
|---|---|---|---|
| R1 | **无自动化测试网**，翻译错漏只能靠人工发现 | 高 | §7.2 冒烟测试 + §7.3 逐 Phase 双入口并排对照 + §7.4 Playwright 主干链路 |
| R2 | **`$effect` 自动追踪导致多触发/死循环** | 高 | §6.2 逐条审计 deps 差集；已识别 5 处高危点 |
| R3 | **迁移窗口内 `main` 继续开发导致分叉** | 高 | 冻结 `web/` 功能开发；无法冻结则维护 `MIGRATION-PORT.md` 逐笔补齐 |
| R4 | **recharts 无替代品，图表需重画** | 中 | 排到最后一个 Phase；管理员专属、lazy chunk，不阻塞玩家侧；必要时可先占位降级（需拍板） |
| R5 | `onChange` → `oninput` 映射错误导致输入框失灵 | 中 | 全局 grep `onChange` 逐个确认元素类型；`NumberField` 有专门测试 |
| R6 | 大快照用 `$state` 深 Proxy 化导致性能退化（正是本次重构想避免的） | 中 | §6.6 强制 `$state.raw` |
| R7 | `keepContribute` 之类的"保持挂载"设计被简化掉，用户草稿丢失 | 中 | §6.7 已单列；review 时重点看 |
| R8 | 未 keyed 的 `{#each}` 导致列表 DOM 复用错乱（聊天气泡、棋盘） | 中 | 强制所有 `{#each}` 带 key |
| R9 | socket 监听清理时序变化（`off` 无参调用的隐式耦合） | 中 | P0 阶段先在 React 侧改成精确 `off(event, handler)` |
| R10 | iOS Safari 特殊路径（禁用 `permessage-deflate`）回归 | 中 | 实机验证列入必测项；`ws.go` 的 UA 嗅探逻辑不动 |
| R11 | 埋点触发时机变化污染数据分析历史曲线 | 低 | §6.8；对照 5 个 `trackPageview` 调用点 |
| R12 | `@lucide/svelte` 图标名与 `lucide-react` 不完全一致 | 低 | P1 阶段先把 23 个图标名过一遍，缺的用内联 SVG |

---

## 10. 工作量估算（粗略，需按实际速度校准）

以"约 13,200 行 JSX 组件代码 + 240 行适配 + 图表重画 + 全量人工回归"计：

| Phase | 相对体量 | 说明 |
|---|---|---|
| P0（React 侧整理） | 中 | 拆文件 + 提函数，机械但量大；独立可合入 |
| P1-P2（构建 + 适配层） | 小 | 约 250 行 |
| P3（外壳 + 登录） | 中 | `App.svelte` 是全局状态中枢，最需要小心 |
| P4（大厅 + 建房） | 大 | `CreateRoom` 单个 760 行 |
| P5（房间 + 聊天） | 大 | `Room` 670 行 + 聊天全家桶 |
| P6（8 个游戏面板） | 大 | 约 1,800 行，但结构高度重复，越做越快 |
| P7（个人 + 社交） | 大 | `ProfilePanel` 550 行 + 力导向图 543 行 |
| P8（共建投稿） | 中 | 约 1,150 行，且已拆分较好 |
| P9（后台 11 分区） | 大 | 约 2,700 行 |
| P10（图表重画） | 中-大 | 不确定性最高，非翻译而是重实现 |
| P11（收尾 + 回归 + 文档） | 中 | 全量人工回归清单是主要耗时 |

**关键判断：这不是一个"高风险架构重写"，而是一个"范围明确、边界清晰、体量偏大的表现层替换"。** 决定总耗时的是 13,200 行的翻译体量与回归测试面，而不是技术可行性。建议按 Phase 逐块推进、每块独立验收，不要一口气翻译完再统一联调。

---

## 11. 开工检查单（历史记录）

动工前请确认：

- [x] 确认 `web/` 的功能冻结窗口并在迁移分支完成收尾
- [x] 选择 LayerChart 作为图表实现
- [x] 完成纯逻辑测试与迁移生命周期测试；组件级浏览器冒烟仍属于后续增强项
- [x] 在 `feat/svelte-migration` 分支完成 P0–P11
- [x] 同步 `README.md`、`CLAUDE.md`、`CHANGELOG.md` 与代码注释

## 12. 实际落地结果（v3.1.1）

- `web/src` 已无 React/TSX 入口，`web/index.html` 使用 `main.ts` 挂载 Svelte `App.svelte`。
- `web/package.json` 已移除 React、Recharts 和 React 图标依赖，使用 Svelte 5、LayerChart、`@lucide/svelte` 与 `svelte-check`。
- `web/src/lib/stores/` 成为全站状态入口；后端 Go、协议生成代码、SQLite schema、静态托管和 Release 包目录约束均未改变。
- `web/vite.config.ts` 保留 markdown HTML 插件、版本号注入、Node shim 与代理配置，并增加懒加载场景分包及构建期边界检查。
- 已通过 `CGO_ENABLED=1 /usr/local/go/bin/go test ./...`、`go vet ./...`、前端 Vitest（39 项）、`svelte-check`（0 errors/0 warnings）和 Vite 生产构建。
