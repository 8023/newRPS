import { maxAspectRatio, maxOriginalImageBytes, maxProofPixels, maxProofUploadBytes } from "./constants";
import { decodeToImageSource, encodeCanvasToWebpFile, StaleChunkReloadError } from "./imagePipeline";

/**
 * 证明图上传前处理（固定流水线，输出必须是 WebP）：
 * 选文件 → 长宽比(>21:9拒绝) → 原图大小(>10MB拒绝) → 像素数(>4MP等比缩放) → WebP 85%
 */

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
        return await encodeCanvasToWebpFile(canvas, baseName, quality, maxProofUploadBytes);
      } catch (e) {
        // 换新版本需要整页刷新的错误：缩小尺寸/降质量重试没有意义，直接抛出。
        if (e instanceof StaleChunkReloadError) throw e;
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
