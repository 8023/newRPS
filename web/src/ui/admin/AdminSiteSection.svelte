<script lang="ts">
  // 「网站管理」分区：运行状态、公告/声明开关、站点信息、限流策略、提示语文案。
  // 源：ui/AdminViews.tsx:318-501（activeSection === "site"）。
  import type { AppConfig } from "../../shared/types";
  import { withAccessControlDefaults } from "../../lib/normalize";
  import { formatBytes, formatDuration } from "../../lib/format";
  import { ADMIN_MESSAGE_META } from "../../lib/adminHelpers";
  import { adminStore } from "../../lib/stores/adminStore.svelte";
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import NumberField from "../shell/NumberField.svelte";
  import Toggle from "../shell/Toggle.svelte";
  import AdminSectionHeader from "./AdminSectionHeader.svelte";

  const draft = $derived(adminStore.draft as AppConfig);
  const lobby = $derived(sessionStore.lobby!);
  const stats = $derived(lobby.serverStats);

  function patchAccessControl(next: Partial<AppConfig["accessControl"]>) {
    adminStore.patch({ accessControl: withAccessControlDefaults({ ...draft.accessControl, ...next }) });
  }
</script>

<div class="config-section admin-section-card">
  <AdminSectionHeader title="运行状态" subtitle="当前服务器与大厅的运行情况。" />
  <div class="admin-preview-card">
    <p>在线 {lobby.onlineCount} 人 · 房间 {lobby.rooms.length} 个 · 运行 {formatDuration(Date.now() - stats.startedAt)}</p>
    <p>房间广播 {stats.roomBroadcasts} 次 · 大厅广播 {stats.lobbyBroadcasts} 次</p>
    <p>最近 1 分钟：房间 {stats.recentRoomBroadcasts} 次 · 大厅 {stats.recentLobbyBroadcasts} 次</p>
    <p>断线 {stats.disconnects} 次 · 重连 {stats.reconnects} 次</p>
    <p>最近房间快照 {formatBytes(stats.lastRoomSnapshotBytes)} · 最近大厅快照 {formatBytes(stats.lastLobbySnapshotBytes)}</p>
    <p>平均快照：房间 {formatBytes(stats.averageRoomSnapshotBytes)} · 大厅 {formatBytes(stats.averageLobbySnapshotBytes)}</p>
  </div>
</div>

<div class="config-section admin-section-card">
  <AdminSectionHeader title="提示公告" subtitle="发送全服公告、开关公告板与安全声明。" />
  <div class="admin-announcement-card">
    <div class="admin-card-title">
      <strong>发送全服公告</strong>
      <small>当前在线玩家和后台页面会立即弹出</small>
    </div>
    <textarea
      value={adminStore.announcementMessage}
      maxlength="200"
      oninput={(event) => (adminStore.announcementMessage = event.currentTarget.value)}
      placeholder="输入公告内容，最多 200 字"
    ></textarea>
    <div class="admin-announcement-actions">
      <label class="field-label">
        <span>显示秒数</span>
        <input type="number" min="3" max="60" value={adminStore.announcementSeconds} oninput={(event) => (adminStore.announcementSeconds = event.currentTarget.value)} />
      </label>
      <button class="primary" onclick={() => adminStore.sendAnnouncement()}>发送公告</button>
    </div>
  </div>
  <div class="admin-announcement-card">
    <div class="admin-card-title">
      <strong>公告板</strong>
      <small>展示在顶栏「关于」面板里，不再是弹窗</small>
    </div>
    <Toggle
      label="开启公告板"
      value={draft.announcementBoard.enabled ?? false}
      onChange={(enabled) => adminStore.patch({ announcementBoard: { ...draft.announcementBoard, enabled } })}
    />
    <label class="field-label">
      <span>公告标题</span>
      <input value={draft.announcementBoard.title} maxlength="32" oninput={(event) => adminStore.patch({ announcementBoard: { ...draft.announcementBoard, title: event.currentTarget.value } })} placeholder="今日公告" />
    </label>
    <label class="field-label">
      <span>公告内容</span>
      <textarea value={draft.announcementBoard.content} maxlength="800" oninput={(event) => adminStore.patch({ announcementBoard: { ...draft.announcementBoard, content: event.currentTarget.value } })} placeholder="写下想让玩家看到的内容"></textarea>
    </label>
  </div>
  <div class={draft.accessControl?.registrationDisabled ? "admin-announcement-card admin-preview-card-warning" : "admin-announcement-card"}>
    <div class="admin-card-title">
      <strong>新用户注册开关</strong>
      <small>禁止新用户注册，防止批量注册攻击</small>
    </div>
    <Toggle label="禁止新用户注册" value={!!draft.accessControl?.registrationDisabled} onChange={(value) => patchAccessControl({ registrationDisabled: value })} />
  </div>
</div>

<div class="config-section admin-section-card">
  <AdminSectionHeader title="网站信息" subtitle="修改网站名称、说明和管理员口令。" />
  <div class="admin-preview-card">
    <span>预览</span>
    <strong>{draft.site.name}</strong>
    <p>{draft.site.description || "暂无网站说明"}</p>
  </div>
  <label class="field-label"><span>网站名称</span><input value={draft.site.name} oninput={(event) => adminStore.patch({ site: { ...draft.site, name: event.currentTarget.value } })} placeholder="网站名称" /></label>
  <label class="field-label"><span>网站说明</span><textarea value={draft.site.description} oninput={(event) => adminStore.patch({ site: { ...draft.site, description: event.currentTarget.value } })} placeholder="网站说明"></textarea></label>
  <label class="field-label"><span>管理员口令</span><input type="password" value={draft.site.adminPassword} oninput={(event) => adminStore.patch({ site: { ...draft.site, adminPassword: event.currentTarget.value } })} placeholder="管理员口令" /></label>
  <label class="field-label"><span>匿名贡献者展示名</span><input value={draft.site.anonymousContributorLabel || "匿名贡献者"} oninput={(event) => adminStore.patch({ site: { ...draft.site, anonymousContributorLabel: event.currentTarget.value } })} placeholder="匿名贡献者" /></label>
  <label class="field-label"><span>猜硬币赢家展示名</span><input value={draft.site.coinFlipWinnerLabel || "系统"} oninput={(event) => adminStore.patch({ site: { ...draft.site, coinFlipWinnerLabel: event.currentTarget.value } })} placeholder="系统" /></label>
</div>

<div class="config-section admin-section-card">
  <AdminSectionHeader title="限流策略" subtitle="按设备、IP 与资源占用范围集中管理访问限制。" />
  <div class="admin-settings-groups">
    <section class="admin-settings-group" aria-labelledby="admin-device-limits-title">
      <div class="admin-card-title">
        <strong id="admin-device-limits-title">设备指纹限制</strong>
        <small>按「出口 IP + 浏览器指纹」识别同一设备，限制用户自己多开</small>
      </div>
      <div class="config-row">
        <label class="field-label">
          <span>同指纹同时在线人数上限</span>
          <NumberField min={1} max={100} value={draft.accessControl?.maxOnlinePerIp ?? 1} onChange={(maxOnlinePerIp) => patchAccessControl({ maxOnlinePerIp })} />
        </label>
        <label class="field-label">
          <span>同指纹 10 分钟内新建玩家上限</span>
          <NumberField min={1} max={200} value={draft.accessControl?.maxCreatesPer10Min ?? 1} onChange={(maxCreatesPer10Min) => patchAccessControl({ maxCreatesPer10Min })} />
        </label>
      </div>
      <p class="hint">指纹由 FingerprintJS 在浏览器生成，与 IP 一起哈希为设备键。</p>
    </section>

    <section class="admin-settings-group" aria-labelledby="admin-ip-limits-title">
      <div class="admin-card-title">
        <strong id="admin-ip-limits-title">IP 限制</strong>
        <small>不依赖浏览器指纹，阻断通过伪造指纹或反复换会话发起的批量请求</small>
      </div>
      <div class="config-row">
        <label class="field-label">
          <span>同 IP 同时在线人数上限</span>
          <NumberField min={1} max={500} value={draft.accessControl?.maxOnlinePerIpTotal ?? 1} onChange={(maxOnlinePerIpTotal) => patchAccessControl({ maxOnlinePerIpTotal })} />
        </label>
        <label class="field-label">
          <span>同 IP 10 分钟内新建玩家上限</span>
          <NumberField min={1} max={500} value={draft.accessControl?.maxCreatesPerIp ?? 1} onChange={(maxCreatesPerIp) => patchAccessControl({ maxCreatesPerIp })} />
        </label>
        <label class="field-label">
          <span>同 IP 10 分钟内签发会话上限</span>
          <NumberField min={1} max={500} value={draft.accessControl?.maxSessionIssuePerIp ?? 1} onChange={(maxSessionIssuePerIp) => patchAccessControl({ maxSessionIssuePerIp })} />
        </label>
      </div>
      <div class="config-row">
        <label class="field-label">
          <span>单个操作的 IP 兜底倍数</span>
          <NumberField min={1} max={100} value={draft.accessControl?.ipBackstopMultiplier ?? 1} onChange={(ipBackstopMultiplier) => patchAccessControl({ ipBackstopMultiplier })} />
        </label>
        <label class="field-label">
          <span>IP 兜底最低下限（次/窗口）</span>
          <NumberField min={1} max={1000} value={draft.accessControl?.ipBackstopMinLimit ?? 1} onChange={(ipBackstopMinLimit) => patchAccessControl({ ipBackstopMinLimit })} />
        </label>
      </div>
      <p class="hint">建房、出招、提交惩罚证明等操作会同时检查同一 IP 的总请求量；即使脚本不断换会话或指纹，也会被这层限制拦截。</p>
    </section>

    <section class="admin-settings-group" aria-labelledby="admin-resource-limits-title">
      <div class="admin-card-title">
        <strong id="admin-resource-limits-title">房间与证明图片</strong>
        <small>限制单个玩家占用的房间数量与证明图片上传频率</small>
      </div>
      <div class="config-row">
        <label class="field-label">
          <span>单玩家同时开房数量上限</span>
          <NumberField min={1} max={50} value={draft.accessControl?.maxActiveRoomsPerOwner ?? 1} onChange={(maxActiveRoomsPerOwner) => patchAccessControl({ maxActiveRoomsPerOwner })} />
        </label>
        <label class="field-label">
          <span>单玩家 10 分钟内证明图上传上限</span>
          <NumberField min={1} max={200} value={draft.accessControl?.maxProofUploadsPerPlayer ?? 1} onChange={(maxProofUploadsPerPlayer) => patchAccessControl({ maxProofUploadsPerPlayer })} />
        </label>
      </div>
    </section>
  </div>
</div>

<div class="config-section admin-section-card">
  <AdminSectionHeader title="提示语" subtitle="编辑玩家操作过程中显示的系统反馈文案。" />
  <div class="config-row admin-message-grid">
    {#each Object.entries(draft.messages) as [key, value] (key)}
      {@const meta = ADMIN_MESSAGE_META[key]}
      <label class="field-label admin-message-field">
        <span class="admin-message-label">
          <strong>{meta?.label || key}</strong>
          <small>{meta?.detail || `自定义提示语 · 配置键 ${key}`}</small>
        </span>
        <input {value} oninput={(event) => adminStore.patch({ messages: { ...draft.messages, [key]: event.currentTarget.value } })} />
      </label>
    {/each}
  </div>
</div>
