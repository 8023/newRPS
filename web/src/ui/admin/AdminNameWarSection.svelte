<script lang="ts">
  // 「名争 / 极限」分区：名字争夺战文案与阈值 + 极限模式全部参数。
  // 源：ui/AdminViews.tsx:877-983（activeSection === "nameWar"）。
  import type { AppConfig } from "../../shared/types";
  import { DEFAULT_NAME_WAR_PENALTY_THRESHOLD, DEFAULT_NAME_WAR_RENAME_MIN_POINTS } from "../../lib/normalize";
  import { adminStore } from "../../lib/stores/adminStore.svelte";
  import AdminSectionHeader from "./AdminSectionHeader.svelte";
  import NumberField from "../shell/NumberField.svelte";

  const draft = $derived(adminStore.draft as AppConfig);
  const preview = $derived(`${draft.nameWar.penaltyPrefix || "失名者"}-A7K2`);
  const extreme = $derived(draft.extremeMode);

  function patchExtreme(nextExtreme: AppConfig["extremeMode"]) {
    adminStore.patch({ extremeMode: nextExtreme });
  }

  const positiveKeys = ["pos1", "pos2", "pos3", "pos4"] as const;
  const negativeKeys = ["neg1", "neg2", "neg3", "neg4"] as const;
  const hourlyKeys = ["pos4", "pos3", "pos2", "pos1", "default"] as const;
</script>

<div class="config-section admin-section-card">
  <AdminSectionHeader title="名字争夺战" subtitle="设置惩罚名前缀、通用改名处标题和退出高难度后的称号。" />
  <div class="admin-preview-card">
    <span>预览</span>
    <strong>{preview}</strong>
    <p>{draft.nameWar.renamePanelTitle || draft.nameWar.loserPanelTitle || "通用改名处"} · {draft.nameWar.nameWarLoserLabel || "名争失格"} / {draft.nameWar.extremeForceClosedLabel || "极限强关"} · 退出高难度称号：{draft.nameWar.escapeTitle || "逃跑的人"}</p>
  </div>
  <div class="config-row">
    <label class="field-label">
      <span>惩罚名前缀 XXXX</span>
      <input value={draft.nameWar.penaltyPrefix} maxlength="16" oninput={(event) => adminStore.patch({ nameWar: { ...draft.nameWar, penaltyPrefix: event.currentTarget.value } })} placeholder="例如：失名者" />
    </label>
    <label class="field-label">
      <span>旧失格者面板标题</span>
      <input value={draft.nameWar.loserPanelTitle} maxlength="24" oninput={(event) => adminStore.patch({ nameWar: { ...draft.nameWar, loserPanelTitle: event.currentTarget.value } })} placeholder="名字争夺战失格者" />
    </label>
    <label class="field-label">
      <span>通用改名处标题</span>
      <input value={draft.nameWar.renamePanelTitle || ""} maxlength="24" oninput={(event) => adminStore.patch({ nameWar: { ...draft.nameWar, renamePanelTitle: event.currentTarget.value } })} placeholder="通用改名处" />
    </label>
    <label class="field-label">
      <span>名争来源标签</span>
      <input value={draft.nameWar.nameWarLoserLabel || ""} maxlength="16" oninput={(event) => adminStore.patch({ nameWar: { ...draft.nameWar, nameWarLoserLabel: event.currentTarget.value } })} placeholder="名争失格" />
    </label>
    <label class="field-label">
      <span>极限强关标签</span>
      <input value={draft.nameWar.extremeForceClosedLabel || ""} maxlength="16" oninput={(event) => adminStore.patch({ nameWar: { ...draft.nameWar, extremeForceClosedLabel: event.currentTarget.value } })} placeholder="极限强关" />
    </label>
    <label class="field-label">
      <span>退出高难度称号</span>
      <input value={draft.nameWar.escapeTitle} maxlength="18" oninput={(event) => adminStore.patch({ nameWar: { ...draft.nameWar, escapeTitle: event.currentTarget.value } })} placeholder="逃跑的人" />
    </label>
    <label class="field-label">
      <span>失格分阈值（真实分）</span>
      <NumberField max={-1} value={draft.nameWar.penaltyThreshold ?? DEFAULT_NAME_WAR_PENALTY_THRESHOLD} onChange={(penaltyThreshold) => adminStore.patch({ nameWar: { ...draft.nameWar, penaltyThreshold } })} placeholder={String(DEFAULT_NAME_WAR_PENALTY_THRESHOLD)} />
    </label>
    <label class="field-label">
      <span>改名最低分（真实分）</span>
      <NumberField min={1} value={draft.nameWar.renameMinPoints ?? DEFAULT_NAME_WAR_RENAME_MIN_POINTS} onChange={(renameMinPoints) => adminStore.patch({ nameWar: { ...draft.nameWar, renameMinPoints } })} placeholder={String(DEFAULT_NAME_WAR_RENAME_MIN_POINTS)} />
    </label>
  </div>
  <p class="hint">随机码固定为 4 位大写字母/数字；已有惩罚名不会因为你改前缀立刻变化，新触发的玩家会使用新前缀。失格线、改名最低分均按数据库真实排位分判定，与展示封顶无关。</p>
</div>

<div class="config-section admin-section-card">
  <AdminSectionHeader title="极限模式" subtitle="修改极限模式名称、标志、折扣、整点扣分和连胜风险。" />
  <div class="admin-preview-card">
    <span>预览</span>
    <strong>{extreme.emoji} {extreme.label}</strong>
    <p>关闭后冷却 {extreme.cooldownHours} 小时；{extreme.winStreakThreshold} 连胜后 {Math.round((extreme.winStreakCrashChance ?? 0) * 100)}% 额外扣 {extreme.crashTargetPoints} 分。</p>
  </div>
  <div class="config-row">
    <label class="field-label"><span>显示名称</span><input value={extreme.label} maxlength="16" oninput={(event) => patchExtreme({ ...extreme, label: event.currentTarget.value })} /></label>
    <label class="field-label"><span>标志 Emoji</span><input value={extreme.emoji} maxlength="4" oninput={(event) => patchExtreme({ ...extreme, emoji: event.currentTarget.value })} /></label>
    <label class="field-label"><span>关闭后冷却小时</span><NumberField min={1} max={168} value={extreme.cooldownHours} onChange={(cooldownHours) => patchExtreme({ ...extreme, cooldownHours })} /></label>
    <label class="field-label"><span>连胜阈值</span><NumberField min={1} max={100} value={extreme.winStreakThreshold} onChange={(winStreakThreshold) => patchExtreme({ ...extreme, winStreakThreshold })} /></label>
    <label class="field-label"><span>连胜风险概率 0-1</span><NumberField min={0} max={1} step={0.01} value={extreme.winStreakCrashChance ?? 0} onChange={(winStreakCrashChance) => patchExtreme({ ...extreme, winStreakCrashChance })} /></label>
    <label class="field-label"><span>连胜风险扣分</span><NumberField min={1} max={1999} value={extreme.crashTargetPoints} onChange={(crashTargetPoints) => patchExtreme({ ...extreme, crashTargetPoints })} /></label>
    <label class="field-label"><span>强关改名最低分</span><NumberField min={1} max={999} value={extreme.forceRenameMinPoints || 1} onChange={(forceRenameMinPoints) => patchExtreme({ ...extreme, forceRenameMinPoints })} /></label>
    <label class="field-label"><span>强关保护小时</span><NumberField min={1} max={168} value={extreme.forceRenameProtectHours || 4} onChange={(forceRenameProtectHours) => patchExtreme({ ...extreme, forceRenameProtectHours })} /></label>
  </div>
  <label class="field-label">
    <span>强行关闭提示</span>
    <textarea value={extreme.forceCloseWarning || ""} maxlength="180" oninput={(event) => patchExtreme({ ...extreme, forceCloseWarning: event.currentTarget.value })} placeholder="强行关闭极限模式后..."></textarea>
  </label>
  <div class="admin-card">
    <div class="admin-card-title">
      <strong>正分输分比例</strong>
      <small>0.9 表示只扣 90%</small>
    </div>
    <div class="config-row">
      {#each positiveKeys as key (key)}
        <label class="field-label"><span>{key}</span><NumberField min={0} max={1} step={0.01} value={extreme.positiveLossRates[key]} onChange={(v) => patchExtreme({ ...extreme, positiveLossRates: { ...extreme.positiveLossRates, [key]: v } })} /></label>
      {/each}
    </div>
  </div>
  <div class="admin-card">
    <div class="admin-card-title">
      <strong>负分赢分比例</strong>
      <small>最负分段按 neg4</small>
    </div>
    <div class="config-row">
      {#each negativeKeys as key (key)}
        <label class="field-label"><span>{key}</span><NumberField min={0} max={1} step={0.01} value={extreme.negativeWinRates[key]} onChange={(v) => patchExtreme({ ...extreme, negativeWinRates: { ...extreme.negativeWinRates, [key]: v } })} /></label>
      {/each}
    </div>
  </div>
  <div class="admin-card">
    <div class="admin-card-title">
      <strong>整点扣分</strong>
      <small>default 用于 0 分及负分</small>
    </div>
    <div class="config-row">
      {#each hourlyKeys as key (key)}
        <label class="field-label"><span>{key}</span><NumberField min={0} max={999} value={extreme.hourlyDecay[key]} onChange={(v) => patchExtreme({ ...extreme, hourlyDecay: { ...extreme.hourlyDecay, [key]: v } })} /></label>
      {/each}
    </div>
  </div>
</div>
