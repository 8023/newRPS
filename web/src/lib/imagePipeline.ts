/**
 * 图片解码 / WebP 编码的共用底层能力，被 proofImage.ts（证明图）与 avatarImage.ts（头像）复用。
 *
 * 兼容要点：
 * - Safari 的 canvas.toBlob('image/webp') 常实际产出 PNG，必须校验魔数并用 WASM 编码器兜底
 * - Chrome 不解 HEIC：libheif-js（按需动态 import，独立 .wasm 文件而非 base64 内联，体积/流量显著更小）
 */

/**
 * 根因：`vite.config.ts` 用 `emptyOutDir: true`，每次发布都会把 dist/ 整个清空重建，
 * 分包 chunk 的文件名带内容哈希，旧哈希文件在发布后即被删除、不会保留。iOS Safari 的标签页
 * 后台常年不回收（不像 Chrome 会更激进地丢弃/重建后台标签），只要用户在发布前就打开了页面、
 * 发布后才第一次触发这类按需加载（HEIC 解码 / WebP WASM 编码器都是 `import()` 动态加载的），
 * 浏览器请求的还是旧哈希文件名，服务端已经 404——这就是 "Importing a module script failed"
 * 的真实成因，不是网络抖动，单纯重试同一个 import() 毫无意义（文件确实已经不在了）。
 * 真正的解法：整页刷新去拿这次发布最新的 index.html（里面引用的就是当前发布的新哈希文件名），
 * 刷新后旧引用问题自然消失。用 sessionStorage 打一个时间戳标记防止在真正无解的情况下
 * （比如网络本身就不通）反复刷新造成死循环。
 */
const STALE_CHUNK_RELOAD_KEY = "__doumiao_stale_chunk_reload_at";
const STALE_CHUNK_RELOAD_WINDOW_MS = 30_000;

function isModuleLoadFailure(e: unknown): boolean {
  const msg = e instanceof Error ? e.message : String(e);
  return /importing a module script failed|failed to fetch dynamically imported module|error loading dynamically imported module/i.test(msg);
}

/** 标记这是「换新版本需要整页刷新」类错误，调用方不应该再靠缩小尺寸/降质量重试——那对这类错误没有意义。 */
export class StaleChunkReloadError extends Error {}

/** 命中动态 import 失败时的兜底：整页刷新一次以换取最新构建产物；短时间内重复命中则不再刷新，避免死循环。 */
function recoverFromStaleChunk(e: unknown): never {
  let alreadyTriedRecently = false;
  try {
    const last = Number(sessionStorage.getItem(STALE_CHUNK_RELOAD_KEY) || "0");
    alreadyTriedRecently = Date.now() - last < STALE_CHUNK_RELOAD_WINDOW_MS;
    sessionStorage.setItem(STALE_CHUNK_RELOAD_KEY, String(Date.now()));
  } catch {
    // 隐私模式下 sessionStorage 可能不可用，退化为总是允许刷新一次
  }
  if (!alreadyTriedRecently) {
    window.location.reload();
    throw new StaleChunkReloadError("检测到新版本，正在刷新页面，请刷新完成后重新上传一次");
  }
  const msg = e instanceof Error ? e.message : String(e);
  throw new StaleChunkReloadError(`图片处理组件加载失败：${msg}（已尝试刷新页面仍未恢复，请检查网络连接后重试）`);
}

export function isHeicLike(file: File): boolean {
  const t = (file.type || "").toLowerCase();
  const n = (file.name || "").toLowerCase();
  return t === "image/heic" || t === "image/heif" || /\.heic$/i.test(n) || /\.heif$/i.test(n);
}

export async function blobIsWebp(blob: Blob): Promise<boolean> {
  const head = new Uint8Array(await blob.slice(0, 12).arrayBuffer());
  if (head.length < 12) return false;
  return (
    head[0] === 0x52 && head[1] === 0x49 && head[2] === 0x46 && head[3] === 0x46 &&
    head[8] === 0x57 && head[9] === 0x45 && head[10] === 0x42 && head[11] === 0x50
  );
}

/** libheif-js 的独立 .wasm 变体：解到 ImageData 后转 ImageBitmap，按需动态 import 触发 WASM 加载 */
async function heicToImageBitmap(file: File): Promise<ImageBitmap> {
  try {
    const [{ default: createHeifModule }, { default: wasmUrl }] = await Promise.all([
      import("libheif-js/libheif-wasm/libheif.js"),
      import("libheif-js/libheif-wasm/libheif.wasm?url")
    ]);
    // Emscripten 的异步 fetch 路径在这套打包环境下会走同步 XHR 兜底并失败
    // （"sync fetching of the wasm failed"），必须自己 fetch 后以 wasmBinary 传入。
    const wasmBinary = new Uint8Array(await (await fetch(wasmUrl)).arrayBuffer());
    const heif = await createHeifModule({ wasmBinary, locateFile: (path) => (path.endsWith(".wasm") ? wasmUrl : path) });
    const decoder = new heif.HeifDecoder();
    const buffer = new Uint8Array(await file.arrayBuffer());
    const images = decoder.decode(buffer);
    if (!images.length) {
      throw new Error("HEIC 转换结果为空");
    }
    const image = images[0];
    const width = image.get_width();
    const height = image.get_height();
    if (!width || !height) {
      throw new Error("HEIC 转换结果为空");
    }
    const imageData = new ImageData(width, height);
    await new Promise<void>((resolve, reject) => {
      image.display(imageData, (result) => (result ? resolve() : reject(new Error("HEIC 解码失败"))));
    });
    image.free();
    return await createImageBitmap(imageData);
  } catch (e) {
    if (isModuleLoadFailure(e)) recoverFromStaleChunk(e);
    const msg = e instanceof Error ? e.message : String(e);
    if (/Content Security Policy/i.test(msg)) {
      throw new Error("HEIC 解码被浏览器安全策略拦截，请改用 jpg/png，或联系管理员放开 script-src wasm-unsafe-eval");
    }
    throw new Error("无法解码 HEIC 图片，请换 jpg/png，或在相册中导出为 JPEG");
  }
}

export type Decoded = {
  source: CanvasImageSource;
  width: number;
  height: number;
  close: () => void;
};

async function tryDecodeBlob(blob: Blob): Promise<Decoded | null> {
  try {
    const bitmap = await createImageBitmap(blob);
    return {
      source: bitmap,
      width: bitmap.width,
      height: bitmap.height,
      close: () => bitmap.close?.()
    };
  } catch {
    /* fall through */
  }
  const url = URL.createObjectURL(blob);
  try {
    const img = await new Promise<HTMLImageElement>((resolve, reject) => {
      const el = new Image();
      el.onload = () => resolve(el);
      el.onerror = () => reject(new Error("decode failed"));
      el.src = url;
    });
    return {
      source: img,
      width: img.naturalWidth || img.width,
      height: img.naturalHeight || img.height,
      close: () => URL.revokeObjectURL(url)
    };
  } catch {
    URL.revokeObjectURL(url);
    return null;
  }
}

export async function decodeToImageSource(file: File): Promise<Decoded> {
  let decoded = await tryDecodeBlob(file);
  if (!decoded && isHeicLike(file)) {
    const bitmap = await heicToImageBitmap(file);
    decoded = { source: bitmap, width: bitmap.width, height: bitmap.height, close: () => bitmap.close?.() };
  }
  if (!decoded) {
    throw new Error("无法读取图片，请换一张有效的图片文件（jpg/png/webp/heic 等）");
  }
  if (!decoded.width || !decoded.height) {
    decoded.close();
    throw new Error("图片尺寸无效");
  }
  return decoded;
}

/** canvas.toBlob 的 webp：Safari 可能回调成功但内容是 PNG，必须验魔数 */
function canvasToBlob(canvas: HTMLCanvasElement, type: string, quality?: number): Promise<Blob | null> {
  return new Promise((resolve) => {
    try {
      if (quality == null) canvas.toBlob((b) => resolve(b), type);
      else canvas.toBlob((b) => resolve(b), type, quality);
    } catch {
      resolve(null);
    }
  });
}

/** WASM 编码器：跨浏览器可靠产出真 WebP（Safari 兜底） */
async function encodeWebpWithWasm(canvas: HTMLCanvasElement, quality01: number): Promise<Blob> {
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("浏览器不支持图片处理");
  const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
  // quality: 0–100
  const q = Math.round(Math.min(100, Math.max(1, quality01 * 100)));
  let encode: (typeof import("@jsquash/webp/encode"))["default"];
  try {
    encode = (await import("@jsquash/webp/encode")).default;
  } catch (e) {
    if (isModuleLoadFailure(e)) recoverFromStaleChunk(e);
    throw e instanceof Error ? e : new Error("图片压缩组件加载失败，请刷新页面后重试");
  }
  const buffer = await encode(imageData, { quality: q });
  return new Blob([buffer], { type: "image/webp" });
}

/** 固定输出 WebP：先试原生 toBlob（快），魔数不对或超限则退回 WASM 编码器。 */
export async function encodeCanvasToWebpFile(canvas: HTMLCanvasElement, baseName: string, quality: number, maxBytes: number): Promise<File> {
  // 1) 原生 toBlob（Chrome 快）；仅当魔数正确才采用
  const native = await canvasToBlob(canvas, "image/webp", quality);
  if (native && native.size > 0 && (await blobIsWebp(native))) {
    if (native.size <= maxBytes) {
      return new File([native], `${baseName}.webp`, { type: "image/webp", lastModified: Date.now() });
    }
  }

  // 2) WASM 编码（Safari 等 toBlob 假 webp / 不支持时）
  const blob = await encodeWebpWithWasm(canvas, quality);
  if (!(await blobIsWebp(blob))) {
    throw new Error("图片转 WebP 失败，请换一张图片或升级浏览器后重试");
  }
  if (blob.size > maxBytes) {
    throw new Error("SIZE");
  }
  return new File([blob], `${baseName}.webp`, { type: "image/webp", lastModified: Date.now() });
}
