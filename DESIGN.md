# 抖喵游戏屋 Design System

从现有 `web/src/styles.css` 与大厅/房间/后台 UI 实际实现中抽出的设计规范（不是重新设计，是把已经在用的规则写下来，作为后续新增/整改页面的对照表）。凡是新页面/新模块，直接复用下表中的 token 与组件类；不要在局部另起一套颜色或间距。

本文件第一版审核时发现：共建模块（`web/src/ui/ContributeView.tsx` / `StepEditor.tsx` / `ContributeSeriesForm.tsx`）大量使用了本规范里根本不存在的类名（`stack`、`row`）、裸 `<fieldset>`/`<label>`（走浏览器默认外观）、以及套错的双栏容器（`.dashboard`），导致复选框丑、控件紧贴、右侧空一块、配色跳出主题。本文件第 5 节末尾记录了对应的正确用法，第 8 节记录了这次顺带修的问题，避免再犯。

## 1. 氛围与基调

浅蓝到浅粉的日间渐变背景，面板半透明白、圆角统一 8px、淡蓝阴影。深色模式整体降饱和度加深，token 一一对应，不单独设计深色配色。语气亲昵直接，主奴/白给基调只体现在**文案**里，不改变控件形状或交互方式——共建页、审核页、赞踩区必须沿用大厅/房间同一套按钮、表单、药丸徽标语言，不能自成一派。

## 2. 颜色 Token（`:root` / `:root[data-theme="dark"]`）

| 角色 | Token | 浅色 | 深色 | 用途 |
|------|-------|------|------|------|
| 正文 | `--page-text` | `#243447` | `#dcebf7` | 标题、正文 |
| 次级 | `--muted` | `#5d758a` | `#9fb4c5` | 提示语、说明、时间戳 |
| 面板（半透明） | `--panel` | `rgba(255,255,255,.92)` | `rgba(21,33,48,.94)` | `.panel`/`.login-card` 外框 |
| 面板（实色） | `--panel-solid` | `#ffffff` | `#182638` | 输入框、卡片、`fieldset` |
| 面板（弱底） | `--panel-soft` | `#f9fdff` | `#122031` | 分区底色、嵌套一层的子卡片 |
| 边框 | `--border` | `#d7eafa` | `#28435c` | 默认描边 |
| 边框（强） | `--border-strong` | `#b9dcf4` | `#3c6f91` | 按钮描边、虚线框 |
| 按钮底/悬停 | `--button-bg` / `--button-hover` | `#fff` / `#edf8ff` | `#1e3045` / `#263b54` | 次级按钮 |
| 主色 | `--primary` / `--primary-hover` | `#66b7ee` / `#4aa7e6` | `#4f9dda` / `#6db5ee` | 主按钮、选中态、`accent-color` |
| 粉色 | `--pink` / `--pink-border` | `#ffd7e9` / `#f2a9ca` | `#553449` / `#8b5270` | 软强调、`.soft-button` |
| 危险/驳回 | `--reject` / `--reject-border` | `#f0c4c8` / `#d48a92` | `#5a3840` / `#8b5560` | 拒绝/下架/危险操作 |
| 提示条 | `--notice-bg` / `--notice-border` | `#fff5fa` / `#f3bad3` | `#342335` / `#7b4d72` | 页级 `.notice` 通知条 |
| 阴影 | `--shadow` | `rgba(102,183,238,.12)` | `rgba(0,0,0,.28)` | 面板/弹窗投影 |

**约定色**（未提升为 CSS 变量，但已在多处复用，新增状态色时优先从这三个里选，不要现造新色相）：

- 成功/在线 `#5fbf8f`（`.online-dot.online`、审核通过）
- 强调/金色 `#f2b84b`（`.sponsor-*`、`.mode-chip`、待处理/等待中状态）
- 危险统一走 `--reject`/`--reject-border`，不单独造红色

状态类底色一律用 `color-mix(in srgb, <约定色> X%, var(--panel-solid))`，不要写死一个浅色 hex——写死的浅色 hex 在深色模式下背景切成深色后不会跟着变深，会突兀刺眼（历史上 `.status-*` 就是这么错的，见第 8 节）。文字颜色统一用 `var(--page-text)`，不要为了"更配色"单独指定文字色号——那样深色模式下容易文字看不清（对比 `.mode-chip`/`.giveaway-chip` 的写法）。

## 3. 字体与文字

- 字体栈：`Inter, "PingFang SC", "Microsoft YaHei", system-ui, sans-serif`
- 页/面板标题 `h2`：18px（`.panel-title h2`），窄一点的用 17px（`.compact-title h2`）
- 分组小标题 `h3`：15px 左右
- 正文：继承 16px
- 说明/次级文字：`.hint`/`.muted`，12–13px，`var(--muted)`
- 药丸/徽标文字：11–12px，`font-weight: 800`
- 主按钮文字：`font-weight: 700`，白色

## 4. 间距、圆角与页面骨架

- 间距基本以 4px 为步进：常见值 4/6/8/10/12/14/16px。
- 圆角统一 8px；药丸/徽标用 999px（全圆角）。
- 卡片间距：同级卡片列表用 `gap: 8–10px`；一个面板内"标题→说明→功能区"这种大块之间用 `gap: 12–14px`。
- **绝对不要出现"元素紧贴在一起没有间距"**：任何一组兄弟元素（标题、说明、标签栏、表单、列表）之间，容器必须显式声明 `display: grid/flex` + `gap`，不能指望某个占位类名（比如以为叫 `stack`/`row` 就自动有间距——CSS 里没这两个类，写了等于没写，这正是共建模块出问题的直接原因）。

### 页面骨架类（选一个，不要混用/瞎发明）

| 类 | 布局 | 场景 |
|----|------|------|
| `.dashboard` | 两栏 `1fr / var(--page-side-width)`（416px），≤920px 收为单栏 | 大厅：主内容 + 侧栏（排行榜等）都要渲染的场景 |
| `.room-layout` | 单栏，游戏内部自己分区 | 房间页 |
| `.admin-page` | 单栏 `grid; gap:14px` | 后台顶层 |
| `.admin-tool-shell` | 两栏 `260px / 1fr`，左侧固定导航 | 后台带侧边导航的场景 |
| `.contribute-page`（新增，见第 5 节） | 单栏，居中，`max-width: 880px` | 表单类、内容不需要撑满 1480px 的独立页面 |

⚠️ **`.dashboard` 是两栏网格，只塞一个子元素时右边 416px 会天然空出来，看着像"预留了个侧栏"**——这不是 bug 是这个类本来的设计（给真的有侧栏内容的页面用的）。只有一块内容、且不需要两栏的页面，套 `.contribute-page` 这种单栏骨架，不要图省事套 `.dashboard`。

断点：`380px`（超小屏微调）/ `768px`（表格/列表切窄屏样式）/ `920px`（两栏骨架收为单栏，移动端折叠三角出现）/ `960px`（少量图表断点）。

## 5. 组件规范

### 按钮

- 主操作：`.primary`（提交、确认、创建）
- 次操作：裸 `button`（保存草稿、返回、取消）
- 软强调：`.soft-button`（粉色调，参与类入口）
- 危险：边框/底色走 `--reject`/`--reject-border`（驳回、下架、删除）
- 尺寸：默认 `min-height: 38px`；`.small` 为 `34px`
- 状态：hover `translateY(-1px)` + 底色变化；`disabled` 降透明度到 0.6 且取消位移；焦点用浏览器默认环，不得 `outline: none`

### 表单控件

- `input`/`select`/`textarea`：边框 `--border`、底色 `--panel-solid`、圆角 8px、`padding: 10px 12px`；**复选框/单选框必须排除**在这条通用规则外（否则 Safari 下会被撑变形），另走专门样式。
- 单个 "标题在上、控件在下" 的字段：`<label className="field-label"><span>标题</span><input .../></label>`（`.field-label { display:grid; gap:5px }`，`span` 走 `--muted` 13px 700）。**不要写裸 `<label>文字<input/></label>`**——`label` 默认 `display: inline`，文字和控件会紧贴在同一行挤在一起。
- 单个独立复选框（如"匿名贡献"）：`<label className="checkbox-inline"><input type="checkbox"/>文案</label>`。复选框固定 18×18px、`accent-color: var(--primary)`，不用浏览器默认小方块+默认蓝。
- 一组多选复选框（阵营/标签这类"选多个"）：`<fieldset className="checkbox-fieldset"><legend>标题</legend><div className="checkbox-pill-row">...</div></fieldset>`，每一项是 `<label className="checkbox-pill"><input type="checkbox"/>文案</label>`——渲染成可点击的药丸，勾选态描边+底色变主题蓝（复用 `.room-tag-picker button.active` 那一套"选中态"语言），而不是一排孤零零的方框+文字。
- `fieldset`/`legend`：一律复位浏览器默认的灰色浮雕边框——改成和 `.step-variant` 卡片一致的 `1px solid var(--border)` + 8px 圆角 + `var(--panel-solid)` 底；嵌套在别的 `fieldset`/卡片里的下一层，底色换 `var(--panel-soft)` 做出层级区分。`legend` 去掉默认凸起效果，变成普通的小标题（`--muted`，13px，700）。

### 状态徽标（`.status-chip`）

药丸形（同 `.gender-chip` 尺寸：`min-height:24px`，`padding:3px 10px`，`border-radius:999px`），底色按状态语义走第 2 节的约定色（`color-mix` 与 `--panel-solid` 混合，深色模式自动跟着变深，不需要单独写深色覆盖）：

- 待审批 / 修改待复审 / 下架申请 → 金色 `#f2b84b` 族
- 已通过 → 绿色 `#5fbf8f` 族
- 已驳回 → `--reject`/`--reject-border`
- 草稿 / 已撤回 → 中性 `--panel-soft`

文字恒定用 `var(--page-text)`，不额外指定文字色号。

### 列表卡片

投稿列表 `.contribute-item`：内部按钮化的列表行必须 `display:grid; gap:6px; text-align:left`——不能指望 `button` 的默认 `inline-flex; justify-content:center` 把多行信息排整齐，那样类型名和状态徽标会挤成水平居中一坨。

### 嵌套通知条

`<p className="notice">` 本来是页级通知条（`max-width:1480px; margin:0 auto 14px`）。**嵌在卡片/表单内部的局部提示不能直接套这个类**——套了会带出页级的居中最大宽和 14px 独立底边距，在一个已经很窄的卡片里显得突兀。嵌套场景下需要单独归零 `max-width`/`margin`（见 `.contribute-page .notice` 覆盖规则）。

## 6. 动效

按钮过渡 `160ms ease`，只动 `background` / `border-color` / `transform` / `opacity`。不引入新的动画曲线或时长。

## 7. 无障碍

- 复选框/单选框做成药丸样式时，`<input>` 本身仍必须保留在 DOM 里可聚焦、可用键盘空格切换，不能纯用 `<div>` 模拟。
- 焦点环不得移除。
- 状态/结果不能只靠颜色，必须同时有文字（本规范里的徽标都自带文案）。

## 8. 本轮排查记录（供以后对照，避免重犯）

- `ContributeView.tsx` 用 `<section className="dashboard">` 只包一个 `<div className="panel">`：两栏网格右边 416px 天然空出，像多余的侧栏——改用单栏的 `.contribute-page`。
- `ContributeView.tsx` / `StepEditor.tsx` / `ContributeSeriesForm.tsx` / `AdminContributionReview.tsx` 大量使用 `className="stack"` / `className="row"`，但 `styles.css` 里从未定义过这两个类——控件之间零间距全靠"以为有间距"的错觉。已补上全局 `.stack`（`grid; gap:10px`）与 `.row`（`flex; wrap; gap:8px; align-items:center`）。
- 所有 `<fieldset>`/`<label>`/`<input type="checkbox">` 都是裸标签，走浏览器默认外观（灰色浮雕边框、默认方框复选框），与站内其它表单（如后台 `.field-label`、`.admin-chat-manager` 复选框）不是一套语言——已按第 5 节补齐 `.field-label`/`.checkbox-inline`/`.checkbox-fieldset`/`.checkbox-pill` 的复用。
- `.status-pending`/`.status-approved` 写死了 `#fff1c4`/`#e3f7ea` 两个和站内配色不搭的浅色 hex，深色模式下也不会变深——已改用第 2 节的约定色 + `color-mix`。
