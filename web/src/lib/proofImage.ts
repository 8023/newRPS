import { maxAspectRatio, maxOriginalImageBytes, maxProofPixels, maxProofUploadBytes } from "./constants";

/**
 * 证明图上传前处理（固定流水线，输出必须是 WebP）：
 * 选文件 → 长宽比(>21:9拒绝) → 原图大小(>10MB拒绝) → 像素数(>4MP等比缩放) → WebP 85%
 *
 * 兼容要点：
 * - Safari 的 canvas.toBlob('image/webp') 常实际产出 PNG，必须校验魔数并用 WASM 编码器兜底
 * - Chrome 不解 HEIC：heic2any（需 CSP worker-src blob:）
 */

function isHeicLike(file: File): boolean {
  const t = (file.type || "").toLowerCase();
  const n = (file.name || "").toLowerCase();
  return t === "image/heic" || t === "image/heif" || /\.heic$/i.test(n) || /\.heif$/i.test(n);
}

async function blobIsWebp(blob: Blob): Promise<boolean> {
  const head = new Uint8Array(await blob.slice(0, 12).arrayBuffer());
  if (head.length < 12) return false;
  return (
    head[0] === 0x52 && head[1] === 0x49 && head[2] === 0x46 && head[3] === 0x46 &&
    head[8] === 0x57 && head[9] === 0x45 && head[10] === 0x42 && head[11] === 0x50
  );
}

async function heicToDecodableBlob(file: File): Promise<Blob> {
  try {
    const heic2any = (await import("heic2any")).default;
    const out = await heic2any({ blob: file, toType: "image/jpeg", quality: 0.92 });
    const blob = Array.isArray(out) ? out[0] : out;
    if (!blob || !(blob instanceof Blob) || blob.size === 0) {
      throw new Error("HEIC 转换结果为空");
    }
    return blob;
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    if (/Content Security Policy|worker/i.test(msg)) {
      throw new Error("HEIC 解码被浏览器安全策略拦截，请改用 jpg/png，或联系管理员放开 worker-src");
    }
    throw new Error("无法解码 HEIC 图片，请换 jpg/png，或在相册中导出为 JPEG");
  }
}

type Decoded = {
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

async function decodeToImageSource(file: File): Promise<Decoded> {
  let decoded = await tryDecodeBlob(file);
  if (!decoded && isHeicLike(file)) {
    const jpegBlob = await heicToDecodableBlob(file);
    decoded = await tryDecodeBlob(jpegBlob);
  }
  if (!decoded) {
    throw new Error("无法读取图片，请换一张有效的图片文件（jpg/png/webp/heic）");
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
  const encode = (await import("@jsquash/webp/encode")).default;
  const buffer = await encode(imageData, { quality: q });
  return new Blob([buffer], { type: "image/webp" });
}

async function encodeCanvasToWebpFile(canvas: HTMLCanvasElement, baseName: string, quality: number): Promise<File> {
  // 1) 原生 toBlob（Chrome 快）；仅当魔数正确才采用
  const native = await canvasToBlob(canvas, "image/webp", quality);
  if (native && native.size > 0 && (await blobIsWebp(native))) {
    if (native.size <= maxProofUploadBytes) {
      return new File([native], `${baseName}.webp`, { type: "image/webp", lastModified: Date.now() });
    }
  }

  // 2) WASM 编码（Safari 等 toBlob 假 webp / 不支持时）
  let blob = await encodeWebpWithWasm(canvas, quality);
  if (!(await blobIsWebp(blob))) {
    throw new Error("图片转 WebP 失败，请换一张图片或升级浏览器后重试");
  }
  if (blob.size > maxProofUploadBytes) {
    throw new Error("SIZE");
  }
  return new File([blob], `${baseName}.webp`, { type: "image/webp", lastModified: Date.now() });
}

/**
 * 证明图上传前处理：固定输出 WebP（85%），失败给出可见错误。
 */
export async function prepareProofImageForUpload(file: File): Promise<File> {
  const type = (file.type || "").toLowerCase();
  const name = (file.name || "").toLowerCase();
  const looksImage =
    type.startsWith("image/") ||
    /\.(jpe?g|png|webp|heic|heif|gif|bmp)$/i.test(name) ||
    type === "" ||
    type === "application/octet-stream";
  if (!looksImage) {
    throw new Error("请选择图片文件（jpg/png/webp/heic 等）");
  }
  if (file.size > maxOriginalImageBytes) {
    throw new Error("原图超过 10MB，请换一张更小的图片");
  }

  const decoded = await decodeToImageSource(file);
  try {
    const { width: rawW, height: rawH } = decoded;
    const ratio = Math.max(rawW, rawH) / Math.min(rawW, rawH);
    if (ratio > maxAspectRatio + 1e-6) {
      throw new Error("图片长宽比超过 21:9，请裁剪后再上传");
    }

    let scale = 1;
    if (rawW * rawH > maxProofPixels) {
      scale = Math.sqrt(maxProofPixels / (rawW * rawH));
    }

    let width = Math.max(1, Math.round(rawW * scale));
    let height = Math.max(1, Math.round(rawH * scale));
    let quality = 0.85;
    const baseName = (file.name.replace(/\.[^.]+$/, "") || "proof").replace(/[^\w.-]+/g, "_");

    for (let attempt = 0; attempt < 6; attempt++) {
      const canvas = document.createElement("canvas");
      canvas.width = width;
      canvas.height = height;
      const ctx = canvas.getContext("2d");
      if (!ctx) throw new Error("浏览器不支持图片处理，请换一个浏览器");
      ctx.fillStyle = "#ffffff";
      ctx.fillRect(0, 0, width, height);
      ctx.drawImage(decoded.source, 0, 0, width, height);

      try {
        return await encodeCanvasToWebpFile(canvas, baseName, quality);
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        if (msg === "SIZE" || /超过 2MB|too large/i.test(msg)) {
          const next = 0.85;
          width = Math.max(320, Math.round(width * next));
          height = Math.max(320, Math.round(height * next));
          quality = Math.max(0.5, quality - 0.1);
          continue;
        }
        // WASM 失败时再缩小重试一次原生/wasm
        if (attempt < 5) {
          width = Math.max(320, Math.round(width * 0.85));
          height = Math.max(320, Math.round(height * 0.85));
          quality = Math.max(0.5, quality - 0.08);
          continue;
        }
        throw e instanceof Error ? e : new Error("图片转 WebP 失败，请换一张图片后重试");
      }
    }
    throw new Error("压缩后图片仍超过 2MB，请换一张更小的图或降低分辨率");
  } finally {
    decoded.close();
  }
}

/** 后台管理图：尽量走同一套 WebP 流水线，失败则原样上传 */
export async function compressAdminImageForUpload(file: File): Promise<File> {
  if (!file.type.startsWith("image/") && !/\.(jpe?g|png|webp|heic|heif)$/i.test(file.name || "")) {
    return file;
  }
  try {
    return await prepareProofImageForUpload(file);
  } catch {
    return file;
  }
}
