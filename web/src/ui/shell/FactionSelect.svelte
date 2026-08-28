<script lang="ts">
  // 阵营是独立下拉 5 选 1，不套用阵营颜色（颜色只用于玩家名字前的药丸标签）。
  // 源：ui/AppViews.tsx:383-399
  import type { AppConfig } from "../../shared/types";

  let { config, factionId, onFactionChange }: { config: AppConfig; factionId: string; onFactionChange: (factionId: string) => void } = $props();

  // 和 GenderSelectControl 一样：当前 factionId 若不在配置里（比如阵营被后台改名/删除后，
  // 页面还留着旧 id），补一个占位 option，避免原生 <select> 因 value 找不到匹配 <option>
  // 而静默显示成列表第一项，让人误以为已经选中了别的阵营、保存时却提交了这个失效的旧 id。
  const hasMatch = $derived(config.genderFactions.some((faction) => faction.id === factionId));
</script>

<label class="field-label">
  <span>阵营</span>
  <select value={factionId} onchange={(event) => onFactionChange(event.currentTarget.value)}>
    {#if !hasMatch && factionId}
      <option value={factionId}>（未知阵营：{factionId}）</option>
    {/if}
    {#each config.genderFactions as faction (faction.id)}
      <option value={faction.id}>{faction.label}</option>
    {/each}
  </select>
</label>
