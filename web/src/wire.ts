// 手写 protobuf wire 编解码（仅 Envelope / PatchOp），无额外 npm 依赖。
// 与 api/proto/wire.proto 字段编号保持一致。

export type PayloadKind = 0 | 1 | 2; // RAW FULL DELTA

export type PatchOp = {
  path: string;
  value?: Uint8Array;
  remove?: boolean;
};

export type Envelope = {
  event?: string;
  id?: number;
  err?: string;
  kind?: PayloadKind;
  channel?: string;
  seq?: number;
  hash?: string;
  full?: Uint8Array;
  ops?: PatchOp[];
  raw?: Uint8Array;
};

/** 读 varint；用乘法避免 JS 32 位移位在 seq/id 变大时截断 */
function readVarint(buf: Uint8Array, i: { o: number }): number {
  let n = 0;
  let mul = 1;
  for (let guard = 0; guard < 10; guard++) {
    if (i.o >= buf.length) throw new Error("varint overflow");
    const b = buf[i.o++];
    n += (b & 0x7f) * mul;
    if ((b & 0x80) === 0) return n;
    mul *= 128;
  }
  throw new Error("varint too long");
}

function writeVarint(n: number, out: number[]) {
  n = Math.max(0, Math.floor(Number(n)) || 0);
  while (n > 0x7f) {
    out.push((n & 0x7f) | 0x80);
    n = Math.floor(n / 128);
  }
  out.push(n);
}

function readBytes(buf: Uint8Array, i: { o: number }): Uint8Array {
  const len = readVarint(buf, i);
  const slice = buf.subarray(i.o, i.o + len);
  i.o += len;
  return slice;
}

function writeBytes(bytes: Uint8Array, out: number[]) {
  writeVarint(bytes.length, out);
  for (let k = 0; k < bytes.length; k++) out.push(bytes[k]);
}

function writeString(s: string, out: number[]) {
  writeBytes(new TextEncoder().encode(s), out);
}

function readString(buf: Uint8Array, i: { o: number }): string {
  return new TextDecoder().decode(readBytes(buf, i));
}

function readPatchOp(buf: Uint8Array): PatchOp {
  const i = { o: 0 };
  const op: PatchOp = { path: "" };
  while (i.o < buf.length) {
    const tag = readVarint(buf, i);
    const field = tag >>> 3;
    const wt = tag & 7;
    if (field === 1 && wt === 2) op.path = readString(buf, i);
    else if (field === 2 && wt === 2) op.value = readBytes(buf, i);
    else if (field === 3 && wt === 0) op.remove = readVarint(buf, i) !== 0;
    else skip(buf, i, wt);
  }
  return op;
}

function writePatchOp(op: PatchOp, out: number[]) {
  const body: number[] = [];
  if (op.path) {
    body.push((1 << 3) | 2);
    writeString(op.path, body);
  }
  if (op.value && op.value.length) {
    body.push((2 << 3) | 2);
    writeBytes(op.value, body);
  }
  if (op.remove) {
    body.push((3 << 3) | 0);
    writeVarint(1, body);
  }
  writeBytes(new Uint8Array(body), out);
}

function skip(buf: Uint8Array, i: { o: number }, wt: number) {
  if (wt === 0) readVarint(buf, i);
  else if (wt === 2) {
    const len = readVarint(buf, i);
    i.o += len;
  } else if (wt === 5) i.o += 4;
  else if (wt === 1) i.o += 8;
  else throw new Error("unsupported wire type " + wt);
}

export function decodeEnvelope(buf: Uint8Array): Envelope {
  const i = { o: 0 };
  const env: Envelope = {};
  while (i.o < buf.length) {
    const tag = readVarint(buf, i);
    const field = tag >>> 3;
    const wt = tag & 7;
    if (field === 1 && wt === 2) env.event = readString(buf, i);
    else if (field === 2 && wt === 0) env.id = readVarint(buf, i);
    else if (field === 3 && wt === 2) env.err = readString(buf, i);
    else if (field === 4 && wt === 0) env.kind = readVarint(buf, i) as PayloadKind;
    else if (field === 5 && wt === 2) env.channel = readString(buf, i);
    else if (field === 6 && wt === 0) env.seq = readVarint(buf, i);
    else if (field === 7 && wt === 2) env.hash = readString(buf, i);
    else if (field === 8 && wt === 2) env.full = readBytes(buf, i);
    else if (field === 9 && wt === 2) {
      if (!env.ops) env.ops = [];
      env.ops.push(readPatchOp(readBytes(buf, i)));
    } else if (field === 10 && wt === 2) env.raw = readBytes(buf, i);
    else skip(buf, i, wt);
  }
  return env;
}

export function encodeEnvelope(env: Envelope): Uint8Array {
  const out: number[] = [];
  if (env.event) {
    out.push((1 << 3) | 2);
    writeString(env.event, out);
  }
  if (env.id) {
    out.push((2 << 3) | 0);
    writeVarint(env.id, out);
  }
  if (env.err) {
    out.push((3 << 3) | 2);
    writeString(env.err, out);
  }
  if (env.kind != null) {
    out.push((4 << 3) | 0);
    writeVarint(env.kind, out);
  }
  if (env.channel) {
    out.push((5 << 3) | 2);
    writeString(env.channel, out);
  }
  if (env.seq) {
    out.push((6 << 3) | 0);
    writeVarint(env.seq, out);
  }
  if (env.hash) {
    out.push((7 << 3) | 2);
    writeString(env.hash, out);
  }
  if (env.full) {
    out.push((8 << 3) | 2);
    writeBytes(env.full, out);
  }
  if (env.ops) {
    for (const op of env.ops) {
      out.push((9 << 3) | 2);
      writePatchOp(op, out);
    }
  }
  if (env.raw) {
    out.push((10 << 3) | 2);
    writeBytes(env.raw, out);
  }
  return new Uint8Array(out);
}

export function utf8(bytes?: Uint8Array): string {
  if (!bytes) return "";
  return new TextDecoder().decode(bytes);
}

export function jsonParse<T = any>(bytes?: Uint8Array): T | undefined {
  if (!bytes || !bytes.length) return undefined;
  return JSON.parse(utf8(bytes)) as T;
}
