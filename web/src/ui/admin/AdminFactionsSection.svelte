<script lang="ts">
  // 「性别与阵营」分区：性别预设（按显示文字分组，勾选阵营）+ 阵营（名称/任务分组/配色）。
  // 源：ui/AdminViews.tsx:508-652（activeSection === "factions"）。
  import type { AppConfig } from "../../shared/types";
  import { factionStyle } from "../../lib/playerDisplay";
  import { styleString } from "../../lib/style";
  import { nextAdminId, taskGroupLabel, TASK_GROUP_OPTIONS } from "../../lib/adminHelpers";
  import { adminStore } from "../../lib/stores/adminStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import AdminSectionHeader from "./AdminSectionHeader.svelte";
  import ColorInput from "./ColorInput.svelte";

  const draft = $derived(adminStore.draft as AppConfig);

  function patchFactions(nextFactions: AppConfig["genderFactions"]) {
    adminStore.patch({ genderFactions: nextFactions });
  }
  function patchGenders(nextGenders: AppConfig["genders"]) {
    adminStore.patch({ genders: nextGenders });
  }

  const filteredFactions = $derived(
    draft.genderFactions.filter((faction) => {
      const keyword = adminStore.factionSearch.trim().toLowerCase();
      if (!keyword) return true;
      return `${faction.id} ${faction.label}`.toLowerCase().includes(keyword);
    })
  );
  const factionIndex = $derived(Math.max(0, draft.genderFactions.findIndex((faction) => faction.id === adminStore.activeFactionId)));
  const faction = $derived(draft.genderFactions[factionIndex]);

  // 性别预设按显示文字分组：同一文案可以同时勾选多个阵营（各自是独立的 GenderOption，
  // 只是 id 不同），行内的勾选框直接对应"这个文案在这个阵营下是否存在一条预设"。
  // key 用分组内第一条 GenderOption 的 id（重命名时该条目位置/id 不变），不能用 label 本身
  // 当 key——否则每敲一个字符 label 就变一次，key 跟着变，Svelte 会把整行（含 input）连同
  // 焦点一起卸载重建，表现为"输完一个字符就失焦"。
  type GenderGroup = { key: string; label: string; factionIds: Set<string> };
  const genderGroups = $derived.by<GenderGroup[]>(() => {
    const groups: GenderGroup[] = [];
    const indexByLabel = new Map<string, number>();
    draft.genders.forEach((gender) => {
      let idx = indexByLabel.get(gender.label);
      if (idx === undefined) {
        idx = groups.length;
        indexByLabel.set(gender.label, idx);
        groups.push({ key: gender.id, label: gender.label, factionIds: new Set() });
      }
      groups[idx].factionIds.add(gender.factionId);
    });
    return groups;
  });
  const groupIndexByLabel = $derived.by(() => {
    const map = new Map<string, number>();
    genderGroups.forEach((g, idx) => map.set(g.label, idx));
    return map;
  });

  // 新增一条性别选项时插入到同一 label 现有条目之后（而非数组末尾），
  // 这样某条目被取消勾选删除后，该分组在 genderGroups 里的显示顺序不会因为
  // "仅存条目排到了数组更靠后位置"而发生跳动。
  function insertGenderOption(list: AppConfig["genders"], label: string, item: AppConfig["genders"][number]) {
    let lastIndex = -1;
    list.forEach((g, idx) => { if (g.label === label) lastIndex = idx; });
    if (lastIndex === -1) return [...list, item];
    return [...list.slice(0, lastIndex + 1), item, ...list.slice(lastIndex + 1)];
  }

  function addFaction() {
    const factionId = nextAdminId("faction", draft.genderFactions.map((item) => item.id));
    adminStore.activeFactionId = factionId;
    patchFactions([...draft.genderFactions, { id: factionId, label: "新阵营", textColor: "#4d5c6f", backgroundColor: "#eef3f8", borderColor: "#c9d6e4", taskGroup: "default" }]);
  }

  function addGenderGroup() {
    let index = 1;
    while (groupIndexByLabel.has(`新性别${index}`)) index += 1;
    const label = `新性别${index}`;
    const usedIds = draft.genders.map((item) => item.id);
    const newItems = draft.genderFactions.map((f) => {
      const id = nextAdminId("gender", usedIds);
      usedIds.push(id);
      return { id, label, factionId: f.id };
    });
    patchGenders([...draft.genders, ...newItems]);
  }
</script>

<div class="config-section admin-section-card">
  <AdminSectionHeader title="性别预设" subtitle="系统预设性别池，玩家只能从中选择；同一显示文字可勾选多个阵营，阵营由所选性别查表决定" />
  <div class="faction-gender-list">
    {#each genderGroups as group (group.key)}
      <div class="mini-card faction-gender-card">
        <div class="config-row faction-gender-row">
          <input
            class="faction-gender-label-input"
            value={group.label}
            oninput={(event) => {
              const nextLabel = event.currentTarget.value;
              patchGenders(draft.genders.map((item) => item.label === group.label ? { ...item, label: nextLabel } : item));
            }}
            placeholder="显示文字"
          />
          <div class="faction-gender-checkboxes">
            {#each draft.genderFactions as f (f.id)}
              <label class="faction-gender-checkbox">
                <input
                  type="checkbox"
                  checked={group.factionIds.has(f.id)}
                  onchange={(event) => {
                    let nextGenders: AppConfig["genders"];
                    if (event.currentTarget.checked) {
                      const genderId = nextAdminId("gender", draft.genders.map((item) => item.id));
                      nextGenders = insertGenderOption(draft.genders, group.label, { id: genderId, label: group.label, factionId: f.id });
                    } else {
                      nextGenders = draft.genders.filter((item) => !(item.label === group.label && item.factionId === f.id));
                    }
                    if (nextGenders.length === 0) {
                      uiStore.notify("至少需要保留 1 个性别预设");
                      return;
                    }
                    patchGenders(nextGenders);
                  }}
                />
                <span>{f.label}</span>
              </label>
            {/each}
          </div>
        </div>
      </div>
    {/each}
  </div>
  <button onclick={addGenderGroup}>添加性别预设</button>
</div>

<div class="config-section admin-section-card">
  <AdminSectionHeader title="阵营" subtitle="玩家选择的阵营分组，标签颜色和任务分组在这里配置。" />
  <div class="punishment-manager faction-manager">
    <aside class="punishment-index-panel">
      <input value={adminStore.factionSearch} oninput={(event) => (adminStore.factionSearch = event.currentTarget.value)} placeholder="搜索阵营" />
      <div class="punishment-index-list">
        {#each filteredFactions as item (item.id)}
          <button class={item.id === faction?.id ? "active" : ""} onclick={() => (adminStore.activeFactionId = item.id)}>
            <span>{item.label}</span>
            <small>{taskGroupLabel(item.taskGroup)}</small>
          </button>
        {/each}
        {#if filteredFactions.length === 0}<p class="empty">没有匹配的阵营</p>{/if}
      </div>
      <button onclick={addFaction}>添加阵营</button>
    </aside>
    {#if faction}
      <div class="mini-card punishment-detail-panel faction-editor">
        <div class="admin-card-title">
          <strong>{faction.label}</strong>
          <small>{factionIndex + 1} / {draft.genderFactions.length} · {taskGroupLabel(faction.taskGroup)}</small>
        </div>
        <div class="admin-preview-strip compact-preview-strip">
          <span class="faction-preview" style={styleString(factionStyle(faction))}>预览：{faction.label}</span>
        </div>
        <div class="config-row">
          <label class="field-label"><span>阵营名称</span><input value={faction.label} oninput={(event) => patchFactions(draft.genderFactions.map((item, i) => i === factionIndex ? { ...item, label: event.currentTarget.value } : item))} placeholder="阵营名称" /></label>
          <label class="field-label">
            <span>任务分组（决定系统任务/称号取哪份文案）</span>
            <select value={faction.taskGroup} onchange={(event) => patchFactions(draft.genderFactions.map((item, i) => i === factionIndex ? { ...item, taskGroup: event.currentTarget.value } : item))}>
              {#each TASK_GROUP_OPTIONS as group (group.id)}<option value={group.id}>{group.label}</option>{/each}
            </select>
          </label>
        </div>
        <div class="color-grid">
          <ColorInput label="文字颜色" value={faction.textColor} onChange={(value) => patchFactions(draft.genderFactions.map((item, i) => i === factionIndex ? { ...item, textColor: value } : item))} />
          <ColorInput label="背景颜色" value={faction.backgroundColor} onChange={(value) => patchFactions(draft.genderFactions.map((item, i) => i === factionIndex ? { ...item, backgroundColor: value } : item))} />
          <ColorInput label="边框颜色" value={faction.borderColor} onChange={(value) => patchFactions(draft.genderFactions.map((item, i) => i === factionIndex ? { ...item, borderColor: value } : item))} />
        </div>
      </div>
    {/if}
  </div>
</div>
