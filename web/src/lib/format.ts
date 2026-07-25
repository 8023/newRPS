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

/** 累计在线时长展示：天/时/分，不足 1 分钟显示秒。 */
export function formatOnlineDuration(ms: number) {
  const totalSec = Math.max(0, Math.floor((Number.isFinite(ms) ? ms : 0) / 1000));
  if (totalSec < 60) return `${totalSec} 秒`;
  const totalMin = Math.floor(totalSec / 60);
  if (totalMin < 60) return `${totalMin} 分钟`;
  const totalHour = Math.floor(totalMin / 60);
  if (totalHour < 24) {
    const remMin = totalMin % 60;
    return remMin ? `${totalHour} 小时 ${remMin} 分钟` : `${totalHour} 小时`;
  }
  const days = Math.floor(totalHour / 24);
  const remHour = totalHour % 24;
  return remHour ? `${days} 天 ${remHour} 小时` : `${days} 天`;
}

