// React 版用 CSSProperties 对象直接传给 style={}；Svelte 的 style 属性只接受字符串，
// 这里统一做一次转换：驼峰键转 kebab-case，`--` 开头的 CSS 自定义属性原样保留。
export function styleString(style: Record<string, string | number | undefined> | undefined): string {
  if (!style) return "";
  const parts: string[] = [];
  for (const [key, value] of Object.entries(style)) {
    if (value === undefined || value === null || value === "") continue;
    const cssKey = key.startsWith("--") ? key : key.replace(/[A-Z]/g, (m) => "-" + m.toLowerCase());
    parts.push(`${cssKey}: ${value}`);
  }
  return parts.join("; ");
}
