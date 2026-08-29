<script lang="ts">
  // 配置类数字输入框：外部 value 恒为合法数字，清空/输入非法值后失焦会静默恢复成上一个
  // 合法值——用于后台各类"总有一个当前生效值"的设置项（限流阈值、随机任务难度步长、
  // 极限模式参数……）。源：ui/NumberField.tsx:90-117。
  import BaseNumberField from "./BaseNumberField.svelte";

  let { value, onChange, min, max, step, placeholder, disabled, className }: {
    value: number;
    onChange: (next: number) => void;
    min?: number;
    max?: number;
    step?: number;
    placeholder?: string;
    disabled?: boolean;
    className?: string;
  } = $props();
</script>

<BaseNumberField
  {value}
  onChange={(next) => { if (typeof next === "number") onChange(next); }}
  {min} {max} {step} {placeholder} {disabled} {className}
  onInvalidBlur="revert"
/>
