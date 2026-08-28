<script lang="ts">
  // 源：ui/AppViews.tsx:5260-5262
  let { value, options, onChange }: { value: string; options: Array<{ value: string; label: string }>; onChange: (value: string) => void } = $props();
</script>

<select {value} onchange={(event) => onChange(event.currentTarget.value)}>
  {#each options as option (option.value)}
    <option value={option.value}>{option.label}</option>
  {/each}
</select>
