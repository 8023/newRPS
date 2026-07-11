// 浏览器指纹：用于与 IP 组合做防多开（宿舍/公司共享出口 IP 时区分设备）
import FingerprintJS from "@fingerprintjs/fingerprintjs";

const storageKey = "rps-browser-fingerprint";

let cached: string | null = null;
let inflight: Promise<string> | null = null;

function sanitizeFingerprint(raw: string): string {
  const cleaned = raw.replace(/[^a-zA-Z0-9_-]/g, "").slice(0, 128);
  return cleaned || "missing";
}

/** 获取稳定 visitorId；失败时回落本地随机占位，避免完全无指纹。 */
export async function getBrowserFingerprint(): Promise<string> {
  if (cached) return cached;
  const fromStore = localStorage.getItem(storageKey);
  if (fromStore && fromStore !== "missing") {
    cached = sanitizeFingerprint(fromStore);
    return cached;
  }
  if (inflight) return inflight;

  inflight = (async () => {
    try {
      const agent = await FingerprintJS.load();
      const result = await agent.get();
      const id = sanitizeFingerprint(String(result.visitorId || ""));
      if (id && id !== "missing") {
        cached = id;
        localStorage.setItem(storageKey, id);
        return id;
      }
    } catch {
      /* ignore — 隐私模式/脚本拦截 */
    }
    // 无可用指纹时生成并缓存会话级占位（刷新仍一致，清缓存则变）
    let fallback = localStorage.getItem(storageKey);
    if (!fallback || fallback === "missing") {
      const rnd = crypto.getRandomValues(new Uint8Array(16));
      fallback = Array.from(rnd, (b) => b.toString(16).padStart(2, "0")).join("");
      localStorage.setItem(storageKey, fallback);
    }
    cached = sanitizeFingerprint(fallback);
    return cached;
  })();

  try {
    return await inflight;
  } finally {
    inflight = null;
  }
}
