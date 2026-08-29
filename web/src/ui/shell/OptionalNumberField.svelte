<script lang="ts">
  // 必填但允许中途清空的数字输入框：外部 value 类型是 `number | ""`，失焦后不合法就保留
  // 为 ""，不会被静默改写成任何默认值——用于共建投稿"难度 1-99"这类字段，真正的拦截交给
  // 调用方在提交/发布前用 isValidOrder 之类的业务规则判断，这里只负责"能不能清空、失焦时
  // 标不标红"。invalid 允许调用方额外叠加"点过提交但没填"这种外部条件，与组件自己失焦时
  // 的校验取或。源：ui/NumberField.tsx:119-149。
  import BaseNumberField from "./BaseNumberField.svelte";

  let { value, onChange, min, max, step, placeholder, disabled, className, invalid }: {
    value: number | "";
    onChange: (next: number | "") => void;
    min?: number;
    max?: number;
    step?: number;
    placeholder?: string;
    disabled?: boolean;
    className?: string;
    invalid?: boolean;
  } = $props();
</script>

<BaseNumberField
  {value} {onChange}
  {min} {max} {step} {placeholder} {disabled} {className}
  onInvalidBlur="keep"
  externalInvalid={invalid}
/>
