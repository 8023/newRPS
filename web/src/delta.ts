// 通用状态树增量合并 + CRC32 校验（与 internal/delta 对齐）

export type PatchOp = { path: string; value?: unknown; remove?: boolean };

/** IEEE CRC-32 表（与 Go hash/crc32.ChecksumIEEE 一致） */
const CRC32_TABLE = (() => {
  const table = new Uint32Array(256);
  for (let i = 0; i < 256; i++) {
    let c = i;
    for (let k = 0; k < 8; k++) {
      c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    }
    table[i] = c >>> 0;
  }
  return table;
})();

function crc32IEEE(bytes: Uint8Array): number {
  let crc = 0xffffffff;
  for (let i = 0; i < bytes.length; i++) {
    crc = CRC32_TABLE[(crc ^ bytes[i]) & 0xff] ^ (crc >>> 8);
  }
  return (crc ^ 0xffffffff) >>> 0;
}

/**
 * 对规范化树做 CRC-32（IEEE），返回 8 位小写 hex。
 * 与 Go delta.Hash 对齐；仅作合并后一致性探针，非安全哈希。
 */
export function crc32Hex(doc: unknown): string {
  const canon = JSON.stringify(normalize(doc));
  const data = new TextEncoder().encode(canon);
  const sum = crc32IEEE(data);
  return sum.toString(16).padStart(8, "0");
}

/** @deprecated 使用 crc32Hex；保留别名避免旧引用 */
export async function sha256Hex(doc: unknown): Promise<string> {
  return crc32Hex(doc);
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
