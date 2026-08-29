<script lang="ts">
  // 单张背景图字段（房间信息卡图库 / 任务背景图库），交互对齐个人资料页的头像上传：
  // 点击上传即设置/覆盖，点击清空则删除。数据仍存成 string[]（历史上是随机图库），
  // 这里只取/写第一个元素，约束成最多 1 张。源：ui/AdminViews.tsx:2019-2060。
  import Upload from "@lucide/svelte/icons/upload";

  let { label, values, upload, onChange, onError }: {
    label: string;
    values: string[];
    upload: (file: File) => Promise<string>;
    onChange: (values: string[]) => void;
    onError: (message: string) => void;
  } = $props();

  let busy = $state(false);
  let inputEl: HTMLInputElement | null = $state(null);
  const current = $derived(values[0] || "");

  async function handleUpload(file: File) {
    busy = true;
    try {
      const imageUrl = await upload(file);
      onChange([imageUrl]);
    } catch (error) {
      onError(error instanceof Error ? error.message : "上传失败");
    } finally {
      busy = false;
      if (inputEl) inputEl.value = "";
    }
  }
</script>

<div class="admin-bg-image-field">
  <span>{label}</span>
  <div class="admin-bg-image-row">
    {#if current}<img class="admin-bg-image-preview" src={current} alt="" />{:else}<em>暂无背景图</em>{/if}
    <input
      bind:this={inputEl}
      type="file"
      accept="image/png,image/jpeg,image/webp"
      class="admin-bg-image-input"
      disabled={busy}
      onchange={(event) => {
        const file = event.currentTarget.files?.[0];
        if (file) void handleUpload(file);
      }}
    />
    <button type="button" disabled={busy} onclick={() => inputEl?.click()}>
      <Upload size={15} /> {busy ? "上传中…" : current ? "更换背景图" : "上传背景图"}
    </button>
    <button type="button" disabled={busy || !current} onclick={() => onChange([])}>清空</button>
  </div>
</div>
