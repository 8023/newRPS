<script lang="ts">
  // 源：ui/AppViews.tsx:5256-5258
  let { label, value, onChange, disabled = false }: { label: string; value: boolean; onChange: (value: boolean) => void; disabled?: boolean } = $props();
</script>

<label class="toggle">
  <input type="checkbox" checked={value} {disabled} onchange={(event) => onChange(event.currentTarget.checked)} /> {label}
</label>
