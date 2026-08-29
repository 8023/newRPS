<script lang="ts">
  // 源：ui/StepEditor.tsx
  import type { Snippet } from "svelte";
  import type { AppConfig } from "../../shared/types";
  import { tokenKey } from "../../lib/constants";
  import { prepareProofImageForUpload } from "../../lib/proofImage";
  import OptionalNumberField from "../shell/OptionalNumberField.svelte";
  import {
    DIFFICULTY_GUIDE, MAX_STEP_VARIANTS, TAG_GUIDE, isValidOrder, overlappingFactionIds, toggleId, type StepDraft
  } from "../contributeSeries";

  let {
    config, value, onChange, onError, showRandomPoolToggle, rightOfPreview, stepActions, showErrors = false
  }: {
    config: AppConfig;
    value: StepDraft;
    onChange: (next: StepDraft | ((prev: StepDraft) => StepDraft)) => void;
    onError: (message: string) => void;
    showRandomPoolToggle: boolean;
    /** 与「封面图预览」同一行、分布在左侧的附加控件（如「匿名贡献」）；仅单份编辑场景
        （随机任务）传入，系列每一步各自没有独立的匿名开关，不传。 */
    rightOfPreview?: Snippet;
    /** 系列每一步的上移/下移/删除/添加；随机任务不传。传入且这一步要进随机池时启用两栏
        布局：左栏惩罚标签，右栏提示语/难度/封面图 + 这四个按钮；未勾选「同时发布到随机
        任务」时退化为单栏，封面图与这四个按钮合并一行（见 stepActions 传入分支）。 */
    stepActions?: Snippet;
    /** 点击过一次「提交审批」但校验没通过：把还没填好的文案输入框标红，
        与阵营选重复时的 .conflict 标红是同一套视觉语言。不传时按 false 处理。 */
    showErrors?: boolean;
  } = $props();

  let fileInputEl: HTMLInputElement | null = $state(null);
  let uploadedPreview = $state<{ path: string; objectURL: string } | null>(null);
  // Svelte 模板的属性值里 {} 会被当成表达式解析，字面量花括号必须放到 JS 字符串常量里
  // 再插值，不能直接写在 placeholder="..." 里（否则 {loser}/{winner} 会被当成变量引用）。
  const variantPlaceholder = "请输入惩罚任务文案，可使用 {loser} 和 {winner} （含花括号）替代输家和赢家的昵称。";

  const factionIds = $derived(config.genderFactions.map((f) => f.id));
  const maxVariants = $derived(Math.min(MAX_STEP_VARIANTS, Math.max(1, factionIds.length || MAX_STEP_VARIANTS)));
  const showPoolFields = $derived(!showRandomPoolToggle || value.inRandomPool);
  const overlapFactions = $derived(new Set(overlappingFactionIds(value.variants)));

  $effect(() => {
    const current = uploadedPreview;
    return () => {
      if (current) URL.revokeObjectURL(current.objectURL);
    };
  });

  // 一律用函数式更新（读最新的 prev，而不是闭包捕获的 value）：uploadImage 是异步的，
  // 上传期间用户可能已经编辑/删除了别的步骤，patch 若直接拿渲染时闭包的 value 展开，
  // 上传完成后回填就会用过期快照覆盖掉这段时间内的其它修改（见 CHANGELOG 相关修复）。
  function patch(next: Partial<StepDraft>) {
    onChange((prev) => ({ ...prev, ...next }));
  }

  function patchVariant(i: number, next: Partial<StepDraft["variants"][number]>) {
    patch({ variants: value.variants.map((item, idx) => idx === i ? { ...item, ...next } : item) });
  }

  async function uploadImage(file: File) {
    const prepared = await prepareProofImageForUpload(file);
    const body = new FormData();
    body.append("image", prepared);
    body.append("token", localStorage.getItem(tokenKey) || "");
    const res = await fetch("/api/contribution-image", { method: "POST", body });
    const data = await res.json() as { imageUrl?: string; message?: string };
    if (!res.ok || !data.imageUrl) throw new Error(data.message || "上传失败");
    // 新上传的共建图在草稿真正保存、background_image 落库前按服务端规则不可公开访问，
    // 直接拿远端 URL 做 <img> 会先得到 404。用已压缩文件的本地 object URL 预览；保存
    // 后重新进入详情时再自然改用远端 URL，服务端访问控制无需为预览开口子。
    uploadedPreview = { path: data.imageUrl, objectURL: URL.createObjectURL(prepared) };
    patch({ backgroundImage: data.imageUrl });
  }
</script>

{#snippet tagsFieldset()}
  <fieldset class="checkbox-fieldset">
    <legend>惩罚标签</legend>
    <p class="hint">{TAG_GUIDE}</p>
    <div class="checkbox-pill-row">
      {#each config.punishmentTags || [] as tag (tag.id)}
        <button type="button" class={`checkbox-pill${value.tagIds.includes(tag.id) ? " active" : ""}`} onclick={() => patch({ tagIds: toggleId(value.tagIds, tag.id) })}>
          {tag.name}
        </button>
      {/each}
    </div>
  </fieldset>
{/snippet}

{#snippet difficultyField()}
  <label class="field-label">
    <span>难度 1-99</span>
    <OptionalNumberField min={1} max={99} value={value.order} onChange={(order) => patch({ order })} invalid={showErrors && !isValidOrder(value.order)} />
  </label>
{/snippet}

{#snippet coverPicker()}
  <label class="field-label">
    <span>封面图</span>
    <div class="cover-picker">
      <button type="button" onclick={() => fileInputEl?.click()}>选取文件</button>
      {#if value.backgroundImage}
        <button type="button" onclick={() => { uploadedPreview = null; patch({ backgroundImage: "" }); }}>移除封面</button>
        <img src={uploadedPreview?.path === value.backgroundImage ? uploadedPreview.objectURL : value.backgroundImage} alt="" class="cover-thumb" />
      {/if}
      <input
        bind:this={fileInputEl}
        class="cover-picker-input"
        type="file"
        accept="image/*,.heic,.heif"
        onchange={(e) => {
          const file = e.currentTarget.files?.[0];
          e.currentTarget.value = "";
          if (file) uploadImage(file).catch((err: unknown) => onError(err instanceof Error ? err.message : "上传失败"));
        }}
      />
    </div>
  </label>
{/snippet}

<div class="stack step-editor">
  {#each value.variants as variant, i (i)}
    <fieldset class="step-variant">
      <legend>文案{value.variants.length > 1 ? ` ${i + 1}` : ""}</legend>
      <label>
        <textarea
          class={showErrors && !variant.text.trim() ? "field-invalid" : undefined}
          value={variant.text}
          oninput={(e) => patchVariant(i, { text: e.currentTarget.value })}
          required
          placeholder={variantPlaceholder}
        ></textarea>
      </label>
      <fieldset class="checkbox-fieldset">
        <legend>适用阵营</legend>
        <div class="checkbox-pill-row">
          {#each config.genderFactions as f (f.id)}
            {@const selected = variant.factionIds.includes(f.id)}
            {@const conflict = selected && overlapFactions.has(f.id)}
            <button type="button" class={`checkbox-pill${selected ? " active" : ""}${conflict ? " conflict" : ""}`} onclick={() => patchVariant(i, { factionIds: toggleId(variant.factionIds, f.id) })}>
              {f.label}
            </button>
          {/each}
        </div>
      </fieldset>
      {#if value.variants.length > 1}
        <button type="button" class="small" onclick={() => patch({ variants: value.variants.filter((_, idx) => idx !== i) })}>删除这份</button>
      {/if}
    </fieldset>
  {/each}
  {#if overlapFactions.size > 0}<p class="notice">多份文案不能勾同一个阵营，点红色的再取消。</p>{/if}
  <button type="button" class="small" disabled={value.variants.length >= maxVariants} onclick={() => patch({ variants: [...value.variants, { text: "", factionIds: [] }] })}>
    + 添加特定阵营文案
  </button>
  {#if showRandomPoolToggle}
    <label class="checkbox-inline">
      <input type="checkbox" checked={value.inRandomPool} onchange={(e) => patch({ inRandomPool: e.currentTarget.checked })} />
      同时发布到随机任务
    </label>
  {/if}
  {#if stepActions}
    {#if showPoolFields}
      <!-- 系列每一步、且这一步要进随机池：两栏布局——左栏惩罚标签，右栏是提示语 +
           难度/封面图（合并一行，各占右栏一半宽度）+ 紧跟在下面的上移/下移/删除/添加。 -->
      <div class="step-columns">
        <div class="step-column">{@render tagsFieldset()}</div>
        <div class="step-column">
          <p class="hint">{DIFFICULTY_GUIDE}</p>
          <div class="contribute-split-row">
            {@render difficultyField()}
            {@render coverPicker()}
          </div>
          <div class="series-step-actions">{@render stepActions()}</div>
        </div>
      </div>
    {:else}
      <!-- 未勾选「同时发布到随机任务」：没有标签/难度可放，退化成单栏——封面图选择器占
           左半，上移/下移/删除/添加四个按钮占右半（各占整行八分之一），且都对齐到「选取
           文件」按钮所在的那一行，而不是上面「封面图」三个字那一行（.contribute-split-row
           的 align-items:end 保证这一点，与「难度/封面图」合并一行是同一套写法）。 -->
      <div class="contribute-split-row">
        {@render coverPicker()}
        <div class="series-step-actions">{@render stepActions()}</div>
      </div>
    {/if}
  {:else}
    {#if showPoolFields}{@render tagsFieldset()}{/if}
    {#if showPoolFields}<p class="hint">{DIFFICULTY_GUIDE}</p>{/if}
    <div class="contribute-split-row">
      {#if showPoolFields}{@render difficultyField()}{/if}
      {@render coverPicker()}
    </div>
  {/if}
  {#if rightOfPreview}{@render rightOfPreview()}{/if}
</div>
