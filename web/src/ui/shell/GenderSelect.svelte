<script lang="ts">
  // 性别：查表法预设下拉，候选池按已选阵营过滤。用原生 <select>——Safari 下 <input list>
  // 组合框弹出列表的配色不跟随页面暗色主题，会出现白底白字看不清的问题，<select> 原生下拉正常。
  // 源：ui/AppViews.tsx:412-448（GenderSelectControl 内联合并，仅这一处调用）
  import type { AppConfig } from "../../shared/types";
  import { gendersForFaction } from "../../lib/playerDisplay";

  let { config, genderId, factionId, onGenderChange }: {
    config: AppConfig;
    genderId: string;
    factionId: string;
    onGenderChange: (genderId: string) => void;
  } = $props();

  // 候选池按 factionId 过滤；若当前 genderId 不在过滤后的池子里（比如阵营调整前的历史数据），
  // 额外把它补回选项列表，避免原生 <select> 因 value 找不到匹配 <option> 而静默显示错乱。
  const options = $derived.by(() => {
    const pool = gendersForFaction(config, factionId);
    return genderId && !pool.some((gender) => gender.id === genderId)
      ? [...pool, ...config.genders.filter((gender) => gender.id === genderId)]
      : pool;
  });
</script>

<label class="field-label">
  <span>性别</span>
  <select value={genderId} onchange={(event) => onGenderChange(event.currentTarget.value)}>
    {#each options as gender (gender.id)}
      <option value={gender.id}>{gender.label}</option>
    {/each}
  </select>
</label>
