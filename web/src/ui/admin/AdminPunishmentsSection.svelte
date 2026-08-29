<script lang="ts">
  // 「惩罚任务」分区：标签库（房间信息卡图库 + 随机房名词库）、玩家发布任务房名词库、
  // 随机任务全局难度、系列任务步数范围。源：ui/AdminViews.tsx:788-874。
  import type { AppConfig } from "../../shared/types";
  import { nextAdminId, sampleRoomName, defaultAdminRoomNamePool, emptyRoomNamePool } from "../../lib/adminHelpers";
  import { MAX_SERIES_STEPS } from "../contributeSeries";
  import { adminStore } from "../../lib/stores/adminStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import AdminSectionHeader from "./AdminSectionHeader.svelte";
  import AdminBackgroundImageField from "./AdminBackgroundImageField.svelte";
  import RoomNamePoolEditor from "./RoomNamePoolEditor.svelte";
  import NumberField from "../shell/NumberField.svelte";

  const draft = $derived(adminStore.draft as AppConfig);
  const tags = $derived(draft.punishmentTags || []);
  const rs = $derived(draft.punishmentRandomSettings || { orderStep: 2, maxDifficultyOvershoot: 5 });
  const filteredTags = $derived.by(() => {
    const keyword = adminStore.punishmentSearch.trim().toLowerCase();
    return tags.filter((t) => !keyword || `${t.id} ${t.name}`.toLowerCase().includes(keyword));
  });
  const tagIndex = $derived(Math.max(0, tags.findIndex((t) => t.id === adminStore.activeTagId)));
  const activeTag = $derived(tags[tagIndex]);

  function addTag() {
    const nextId = nextAdminId("tag", tags.map((t) => t.id));
    adminStore.activeTagId = nextId;
    adminStore.patch({ punishmentTags: [...tags, { id: nextId, name: "新标签", roomBackgroundImages: [], roomNamePool: defaultAdminRoomNamePool() }] });
  }

  function deleteTag() {
    if (!activeTag) return;
    if (tags.length <= 1) { uiStore.notify("至少需要保留 1 个标签"); return; }
    if (!window.confirm(`确定删除标签「${activeTag.name}」吗？保存配置后，任务池里带该标签的任务会自动摘除该标签。`)) return;
    const nextTags = tags.filter((_, i) => i !== tagIndex);
    adminStore.activeTagId = nextTags[Math.max(0, tagIndex - 1)]?.id || nextTags[0]?.id || "";
    adminStore.patch({ punishmentTags: nextTags });
  }
</script>

<div class="config-section admin-section-card">
  <AdminSectionHeader title="惩罚任务" subtitle="配置标签库、随机抽取参数和玩家发布任务的房名词库。" />
  <input value={adminStore.punishmentSearch} oninput={(event) => (adminStore.punishmentSearch = event.currentTarget.value)} placeholder="搜索标签名称 / ID" style="margin-bottom: 12px" />
  <div class="punishment-manager">
    <aside class="punishment-index-panel">
      <div class="punishment-index-list">
        {#each filteredTags as item (item.id)}
          <button class={item.id === activeTag?.id ? "active" : ""} onclick={() => (adminStore.activeTagId = item.id)}>
            <span>{item.name}</span>
          </button>
        {/each}
        {#if filteredTags.length === 0}<p class="empty">没有匹配的标签</p>{/if}
      </div>
      <button onclick={addTag}>添加标签</button>
    </aside>
    {#if activeTag}
      <div class="mini-card punishment-detail-panel">
        <div class="admin-card-title">
          <strong>{activeTag.name}</strong>
          <small>{tagIndex + 1} / {tags.length} · 示例：{sampleRoomName(activeTag.roomNamePool)}</small>
        </div>
        <div class="config-row">
          <label class="field-label"><span>任务标签</span><input value={activeTag.name} oninput={(event) => adminStore.patch({ punishmentTags: tags.map((t, i) => i === tagIndex ? { ...t, name: event.currentTarget.value } : t) })} /></label>
          <div class="field-label field-label-action">
            <span>&nbsp;</span>
            <button type="button" class="danger-button" onclick={deleteTag}>删除这个标签</button>
          </div>
        </div>
        <AdminBackgroundImageField label="房间信息卡图库" values={activeTag.roomBackgroundImages || []} upload={(file) => adminStore.uploadAdminImage(file)} onError={(m) => uiStore.notify(m)} onChange={(roomBackgroundImages) => adminStore.patch({ punishmentTags: tags.map((t, i) => i === tagIndex ? { ...t, roomBackgroundImages } : t) })} />
        <RoomNamePoolEditor title="随机房名词库" pool={activeTag.roomNamePool || emptyRoomNamePool()} onChange={(roomNamePool) => adminStore.patch({ punishmentTags: tags.map((t, i) => i === tagIndex ? { ...t, roomNamePool } : t) })} />
      </div>
    {/if}
  </div>
  <div class="mini-card player-punishment-room-name-card">
    <div class="admin-card-title">
      <strong>玩家发布任务随机房名词库</strong>
      <small>示例：{sampleRoomName(draft.playerPunishmentRoomNamePool)}</small>
    </div>
    <RoomNamePoolEditor title="使用玩家自定义任务时所生成的房间名" pool={draft.playerPunishmentRoomNamePool || defaultAdminRoomNamePool()} onChange={(playerPunishmentRoomNamePool) => adminStore.patch({ playerPunishmentRoomNamePool })} />
  </div>
  <div class="mini-card">
    <div class="admin-card-title"><strong>随机任务全局难度</strong></div>
    <div class="config-row">
      <label class="field-label">
        <span>难度推进步长（默认 2）</span>
        <NumberField min={0} step={1} value={rs.orderStep ?? 2} onChange={(orderStep) => adminStore.patch({ punishmentRandomSettings: { ...rs, orderStep } })} />
      </label>
      <label class="field-label">
        <span>难度上浮硬顶（默认 5）</span>
        <NumberField min={0} step={1} value={rs.maxDifficultyOvershoot ?? 5} onChange={(maxDifficultyOvershoot) => adminStore.patch({ punishmentRandomSettings: { ...rs, maxDifficultyOvershoot } })} />
      </label>
    </div>
  </div>
  <div class="mini-card">
    <div class="admin-card-title"><strong>系列任务步数</strong></div>
    <div class="config-row">
      <label class="field-label">
        <span>最低步数（默认 5）</span>
        <NumberField min={1} max={MAX_SERIES_STEPS} value={rs.minSeriesSteps ?? 5} onChange={(minSeriesSteps) => adminStore.patch({ punishmentRandomSettings: { ...rs, minSeriesSteps } })} />
      </label>
      <label class="field-label">
        <span>最高步数（默认 20）</span>
        <NumberField min={1} max={MAX_SERIES_STEPS} value={rs.maxSeriesSteps ?? 20} onChange={(maxSeriesSteps) => adminStore.patch({ punishmentRandomSettings: { ...rs, maxSeriesSteps } })} />
      </label>
    </div>
  </div>
</div>
