/** 展示用格式化（无 UI 依赖） */

export function formatBytes(bytes: number) {
  const n = Number.isFinite(bytes) ? bytes : 0;
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

export function formatDuration(ms: number) {
  const minutes = Math.max(0, Math.floor((Number.isFinite(ms) ? ms : 0) / 60_000));
  if (minutes < 60) return `${minutes} 分钟`;
  const hours = Math.floor(minutes / 60);
  return `${hours} 小时 ${minutes % 60} 分钟`;
}

