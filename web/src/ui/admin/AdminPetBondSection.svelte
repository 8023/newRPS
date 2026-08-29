<script lang="ts">
  // 「宠物乐园」分区：认主/认宠数量上限与文案配置 + 主宠关系力导向图（LayerChart 版）。
  // 源：ui/AdminViews.tsx:1115-1153（activeSection === "petBond"）。
  import type { AppConfig } from "../../shared/types";
  import { adminStore } from "../../lib/stores/adminStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import AdminSectionHeader from "./AdminSectionHeader.svelte";
  import NumberField from "../shell/NumberField.svelte";
  import PetBondGraphPanel from "./PetBondGraphPanel.svelte";

  const draft = $derived(adminStore.draft as AppConfig);
  const pb = $derived(draft.petBond || { panelTitle: "宠物乐园", maxPetsPerMaster: 3, maxMastersPerPet: 3, maxTitleLength: 12 });
</script>

<div class="config-section admin-section-card">
  <AdminSectionHeader title="宠物乐园（认主/认宠）" subtitle="配置大厅面板标题、主人/宠物数量上限与称号长度。" />
  <div class="admin-preview-card">
    <span>预览</span>
    <strong>🐾 {pb.panelTitle}</strong>
    <p>主人最多 {pb.maxPetsPerMaster} 宠 · 宠物最多 {pb.maxMastersPerPet} 主 · 称号 {pb.maxTitleLength} 字</p>
  </div>
  <div class="config-row">
    <label class="field-label">
      <span>大厅面板标题</span>
      <input value={pb.panelTitle} maxlength="24" oninput={(event) => adminStore.patch({ petBond: { ...pb, panelTitle: event.currentTarget.value } })} placeholder="宠物乐园" />
    </label>
    <label class="field-label">
      <span>宠物称号最大字数</span>
      <NumberField min={1} max={24} step={1} value={pb.maxTitleLength} onChange={(maxTitleLength) => adminStore.patch({ petBond: { ...pb, maxTitleLength } })} />
    </label>
  </div>
  <div class="config-row">
    <label class="field-label">
      <span>每名主人最多宠物数</span>
      <NumberField min={1} max={20} step={1} value={pb.maxPetsPerMaster} onChange={(maxPetsPerMaster) => adminStore.patch({ petBond: { ...pb, maxPetsPerMaster } })} />
    </label>
    <label class="field-label">
      <span>每名宠物最多主人数</span>
      <NumberField min={1} max={20} step={1} value={pb.maxMastersPerPet} onChange={(maxMastersPerPet) => adminStore.patch({ petBond: { ...pb, maxMastersPerPet } })} />
    </label>
  </div>
  <p class="hint">关闭玩家侧「开启认主/认宠」不会解除已有关系，只禁止新增；关闭「公开展示」则不出现在大厅关系图。</p>
</div>
<div class="config-section admin-section-card">
  <PetBondGraphPanel onError={(m) => uiStore.notify(m)} />
</div>
