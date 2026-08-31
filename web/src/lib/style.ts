// React 对这些属性名的数字值不会补 px（无量纲），照搬这份名单以保持迁移前后行为一致。
const UNITLESS_PROPS = new Set([
  "opacity", "zIndex", "lineHeight", "fontWeight", "flex", "flexGrow", "flexShrink",
  "order", "zoom", "columnCount", "gridRow", "gridColumn", "gridRowStart", "gridRowEnd",
  "gridColumnStart", "gridColumnEnd", "aspectRatio", "orphans", "widows", "tabSize"
]);

// React 版用 CSSProperties 对象直接传给 style={}，其中数字值会被自动补上 px 单位（除了上面这份无量纲名单）；
// Svelte 的 style 属性只接受字符串，这里统一做一次转换：驼峰键转 kebab-case，数字值按 React 规则补单位，
// `--` 开头的 CSS 自定义属性原样保留。
export function styleString(style: Record<string, string | number | undefined> | undefined): string {
  if (!style) return "";
  const parts: string[] = [];
  for (const [key, value] of Object.entries(style)) {
    if (value === undefined || value === null || value === "") continue;
    const cssKey = key.startsWith("--") ? key : key.replace(/[A-Z]/g, (m) => "-" + m.toLowerCase());
    const cssValue = typeof value === "number" && value !== 0 && !key.startsWith("--") && !UNITLESS_PROPS.has(key) ? `${value}px` : value;
    parts.push(`${cssKey}: ${cssValue}`);
  }
  return parts.join("; ");
}
