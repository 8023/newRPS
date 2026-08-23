import { useEffect, useState } from "react";

/** 内部实现：真正维护「原始输入字符串」的那一层，两个对外导出的组件都是它的薄包装，
    只是失焦时不合法要怎么处理（onInvalidBlur）、以及外部 value/onChange 是否允许 ""
    这两点不同。不直接导出，业务代码一律用下面的 NumberField / OptionalNumberField。 */
function BaseNumberField({
  value, onChange, min, max, step, placeholder, disabled, className, onInvalidBlur, externalInvalid,
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
}) {
  const [text, setText] = useState(String(value));
  const [focused, setFocused] = useState(false);
  const [touched, setTouched] = useState(false);

  // 外部 value 变化（切换编辑对象、被别处代码改写、父组件重置表单等）时同步显示——但正在
  // 打字（focused）时不能被这个效应抢过去覆盖，否则光标跳动、删到一半的输入被打断。
  useEffect(() => {
    if (!focused) setText(String(value));
  }, [value, focused]);

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

  const selfInvalid = onInvalidBlur === "keep" && touched && !isValid(parse(text));
  const showInvalid = externalInvalid || selfInvalid;

  return (
    <input
      type="number"
      className={[className, showInvalid ? "field-invalid" : ""].filter(Boolean).join(" ") || undefined}
      min={min}
      max={max}
      step={step}
      placeholder={placeholder}
      disabled={disabled}
      value={text}
      onFocus={() => setFocused(true)}
      onChange={(e) => {
        const raw = e.target.value;
        setText(raw);
        // 只在能解析成有限数字时才实时回调，纯粹打字打到一半的中间态（清空、单独一个
        // "-"）不会用无意义的 0/NaN 污染外部状态——这正是原生 <input type="number"> 配合
        // Number(e.target.value) 会把清空的框强行拗回 0、删不干净的根源。
        const parsed = parse(raw);
        if (parsed !== "") onChange(parsed);
      }}
      onBlur={() => {
        setFocused(false);
        setTouched(true);
        const parsed = parse(text);
        if (isValid(parsed)) {
          setText(String(parsed));
          onChange(parsed);
          return;
        }
        if (onInvalidBlur === "revert") {
          // 静默恢复成上一个合法值：这类字段总有一个"当前生效值"，恢复原值就是保持现状，
          // 不需要报错打扰。
          setText(String(value));
          return;
        }
        // "keep"：保留空值/非法值原样显示，外部 value 变成 ""，交给调用方在提交/发布前
        // 用业务规则（比如难度必须 1-99 的整数）校验 + 标红，这里不报错。
        onChange("");
      }}
    />
  );
}

/** 配置类数字输入框：外部 value 恒为合法数字，清空/输入非法值后失焦会静默恢复成上一个
    合法值——用于后台各类"总有一个当前生效值"的设置项（限流阈值、随机任务难度步长、
    极限模式参数……），替代原来 `Number(e.target.value)` 每次按键都强制转换、一删空就被
    拗回 0 的写法。 */
export function NumberField({ value, onChange, min, max, step, placeholder, disabled, className }: {
  value: number;
  onChange: (next: number) => void;
  min?: number;
  max?: number;
  step?: number;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
}) {
  return (
    <BaseNumberField
      value={value}
      onChange={(next) => { if (typeof next === "number") onChange(next); }}
      min={min}
      max={max}
      step={step}
      placeholder={placeholder}
      disabled={disabled}
      className={className}
      onInvalidBlur="revert"
    />
  );
}

/** 必填但允许中途清空的数字输入框：外部 value 类型是 `number | ""`，失焦后不合法就保留
    为 ""，不会被静默改写成任何默认值——用于共建投稿"难度 1-99"这类字段，真正的拦截交给
    调用方在提交/发布前用 isValidOrder 之类的业务规则判断，这里只负责"能不能清空、失焦时
    标不标红"。invalid 允许调用方额外叠加"点过提交但没填"这种外部条件，与组件自己失焦时
    的校验取或。 */
export function OptionalNumberField({ value, onChange, min, max, step, placeholder, disabled, className, invalid }: {
  value: number | "";
  onChange: (next: number | "") => void;
  min?: number;
  max?: number;
  step?: number;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  invalid?: boolean;
}) {
  return (
    <BaseNumberField
      value={value}
      onChange={onChange}
      min={min}
      max={max}
      step={step}
      placeholder={placeholder}
      disabled={disabled}
      className={className}
      onInvalidBlur="keep"
      externalInvalid={invalid}
    />
  );
}
