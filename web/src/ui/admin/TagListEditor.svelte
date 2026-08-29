<script lang="ts">
  // 源：ui/AdminViews.tsx:2062-2089
  let { label, placeholder, values, onChange }: { label: string; placeholder: string; values: string[]; onChange: (values: string[]) => void } = $props();

  let draftTag = $state("");

  function addTag() {
    const next = draftTag.trim();
    if (!next) return;
    if (!values.includes(next)) onChange([...values, next]);
    draftTag = "";
  }
</script>

<div class="tag-list-editor">
  <span>{label}</span>
  <div class="tag-list">
    {#each values as value (value)}
      <button type="button" class="tag-chip" onclick={() => onChange(values.filter((item) => item !== value))}>
        {value}<small>×</small>
      </button>
    {/each}
    {#if values.length === 0}<em>暂无词条</em>{/if}
  </div>
  <div class="tag-input-row">
    <input value={draftTag} oninput={(event) => (draftTag = event.currentTarget.value)} onkeydown={(event) => { if (event.key === "Enter") { event.preventDefault(); addTag(); } }} {placeholder} />
    <button type="button" onclick={addTag}>添加</button>
  </div>
</div>
