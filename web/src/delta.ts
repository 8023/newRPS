// 通用 JSON 状态树增量合并（与 internal/delta 对齐）

export type PatchOp = { path: string; value?: unknown; remove?: boolean };

export async function sha256Hex(doc: unknown): Promise<string> {
  const canon = JSON.stringify(normalize(doc));
  const data = new TextEncoder().encode(canon);
  const buf = await crypto.subtle.digest("SHA-256", data);
  return [...new Uint8Array(buf)].map((b) => b.toString(16).padStart(2, "0")).join("");
}

function normalize(v: unknown): unknown {
  if (v === null || typeof v !== "object") return v;
  if (Array.isArray(v)) return v.map(normalize);
  const o = v as Record<string, unknown>;
  const keys = Object.keys(o).sort();
  const out: Record<string, unknown> = {};
  for (const k of keys) out[k] = normalize(o[k]);
  return out;
}

function unescapeKey(k: string) {
  return k.replace(/~1/g, "/").replace(/~0/g, "~");
}

function splitPointer(path: string): string[] {
  if (!path || path === "/") return [];
  const p = path.startsWith("/") ? path.slice(1) : path;
  return p.split("/").map(unescapeKey);
}

export function applyOps(base: unknown, ops: PatchOp[]): unknown {
  let doc: any = base === undefined ? null : structuredClone(base);
  for (const op of ops) {
    if (!op.path) {
      doc = op.remove ? null : op.value;
      continue;
    }
    const parts = splitPointer(op.path);
    if (!parts.length) continue;
    if (doc === null || typeof doc !== "object") {
      doc = {};
    }
    let cur: any = doc;
    for (let i = 0; i < parts.length - 1; i++) {
      const key = parts[i];
      if (Array.isArray(cur)) {
        const idx = Number(key);
        if (!cur[idx] || typeof cur[idx] !== "object") cur[idx] = {};
        cur = cur[idx];
      } else {
        if (cur[key] == null || typeof cur[key] !== "object") cur[key] = {};
        cur = cur[key];
      }
    }
    const last = parts[parts.length - 1];
    if (Array.isArray(cur)) {
      const idx = Number(last);
      if (op.remove) cur[idx] = null;
      else cur[idx] = op.value;
    } else {
      if (op.remove) delete cur[last];
      else cur[last] = op.value;
    }
  }
  return doc;
}

export function deepClone<T>(v: T): T {
  return structuredClone(v);
}
