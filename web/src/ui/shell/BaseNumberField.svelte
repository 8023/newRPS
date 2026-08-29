<script lang="ts">
  // 内部实现：真正维护「原始输入字符串」的那一层，NumberField / OptionalNumberField 两个
  // 对外导出的组件都是它的薄包装，只是失焦时不合法要怎么处理（onInvalidBlur）、以及外部
  // value/onChange 是否允许 "" 这两点不同。业务代码一律用 NumberField / OptionalNumberField，
  // 不直接用这个组件。源：ui/NumberField.tsx:1-88。
  //
  // ⚠️ 全项目语义最微妙的输入框：故意不用 bind:value，因为这里需要维护一层"原始输入
  // 字符串"缓冲（区分"清空中""正在打负号"这些中间态与"外部已生效值"），bind:value 的
  // 双向绑定会把这层缓冲短路掉，直接退化回"删空被拗回默认值"的老问题——这正是本组件
  // 当初要修的 bug（见 plan.md §6.4）。
  import { untrack } from "svelte";

  let {
    value, onChange, min, max, step, placeholder, disabled, className, onInvalidBlur, externalInvalid
  }: {
    value: number | "";
    onChange: (next: number | "") => void;
    min?: number;
    max?: number;
    step?: number;
    placeholder?: string;
    disabled?: boolean;
    className?: string;
    onInvalidBlur: "revert" | "keep";
    externalInvalid?: boolean;
  } = $props();

  let text = $state(untrack(() => String(value)));
  let focused = $state(false);
  let touched = $state(false);

  // 外部 value 变化（切换编辑对象、被别处代码改写、父组件重置表单等）时同步显示——但正在
  // 打字（focused）时不能被这个效应抢过去覆盖，否则光标跳动、删到一半的输入被打断。
  $effect(() => {
    if (!focused) text = String(value);
  });

  function parse(raw: string): number | "" {
    const trimmed = raw.trim();
    if (trimmed === "") return "";
    const n = Number(trimmed);
    return Number.isFinite(n) ? n : "";
  }

  function isValid(n: number | ""): n is number {
    if (n === "") return false;
    if (min != null && n < min) return false;
    if (max != null && n > max) return false;
    return true;
  }

  const selfInvalid = $derived(onInvalidBlur === "keep" && touched && !isValid(parse(text)));
  const showInvalid = $derived(externalInvalid || selfInvalid);
  const inputClass = $derived([className, showInvalid ? "field-invalid" : ""].filter(Boolean).join(" ") || undefined);

  function handleInput(event: Event) {
    const raw = (event.currentTarget as HTMLInputElement).value;
    text = raw;
    // 只在能解析成有限数字时才实时回调，纯粹打字打到一半的中间态（清空、单独一个
    // "-"）不会用无意义的 0/NaN 污染外部状态——这正是原生 <input type="number"> 配合
    // Number(e.target.value) 会把清空的框强行拗回 0、删不干净的根源。
    const parsed = parse(raw);
    if (parsed !== "") onChange(parsed);
  }

  function handleBlur() {
    focused = false;
    touched = true;
    const parsed = parse(text);
    if (isValid(parsed)) {
      text = String(parsed);
      onChange(parsed);
      return;
    }
    if (onInvalidBlur === "revert") {
      // 静默恢复成上一个合法值：这类字段总有一个"当前生效值"，恢复原值就是保持现状，
      // 不需要报错打扰。
      text = String(value);
      return;
    }
    // "keep"：保留空值/非法值原样显示，外部 value 变成 ""，交给调用方在提交/发布前
    // 用业务规则（比如难度必须 1-99 的整数）校验 + 标红，这里不报错。
    onChange("");
  }
</script>

<input
  type="number"
  class={inputClass}
  {min} {max} {step} {placeholder} {disabled}
  value={text}
  onfocus={() => (focused = true)}
  oninput={handleInput}
  onblur={handleBlur}
/>
