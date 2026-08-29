<script lang="ts">
  // 源：ui/AdminViews.tsx:2099-2107
  let { label, value, onChange }: { label: string; value: string; onChange: (value: string) => void } = $props();
</script>

<label class="color-input">
  <span>{label}</span>
  <input type="color" {value} oninput={(event) => onChange(event.currentTarget.value)} />
  <input {value} oninput={(event) => onChange(event.currentTarget.value)} placeholder="#RRGGBB" />
</label>
