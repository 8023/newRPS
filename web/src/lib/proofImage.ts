import { maxAspectRatio, maxOriginalImageBytes, maxProofPixels, maxProofUploadBytes } from "./constants";

/**
 * 证明图上传前处理：
 * 选文件 → 长宽比(>21:9拒绝) → 原图大小(>10MB拒绝) → 像素数(>4MP等比缩放) → Canvas → WebP 85%
 */
export async function prepareProofImageForUpload(file: File): Promise<File> {
  if (!file.type.startsWith("image/")) {
    throw new Error("请选择图片文件（jpg/png/webp 等）");
  }
  if (file.size > maxOriginalImageBytes) {
    throw new Error("原图超过 10MB，请换一张更小的图片");
  }
  let bitmap: ImageBitmap;
  try {
    bitmap = await createImageBitmap(file);
  } catch {
    throw new Error("无法读取图片，请换一张有效的图片文件");
  }
  const { width: rawW, height: rawH } = bitmap;
  if (!rawW || !rawH) {
    bitmap.close?.();
    throw new Error("图片尺寸无效");
  }
  const ratio = Math.max(rawW, rawH) / Math.min(rawW, rawH);
  if (ratio > maxAspectRatio + 1e-6) {
    bitmap.close?.();
    throw new Error("图片长宽比超过 21:9，请裁剪后再上传");
  }
  const pixels = rawW * rawH;
  let scale = 1;
  if (pixels > maxProofPixels) {
    scale = Math.sqrt(maxProofPixels / pixels);
  }
  const width = Math.max(1, Math.round(rawW * scale));
  const height = Math.max(1, Math.round(rawH * scale));
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const context = canvas.getContext("2d");
  if (!context) {
    bitmap.close?.();
    throw new Error("浏览器不支持图片处理，请换一个浏览器");
  }
  context.drawImage(bitmap, 0, 0, width, height);
  bitmap.close?.();
  const blob = await new Promise<Blob | null>((resolve) => {
    canvas.toBlob(resolve, "image/webp", 0.85);
  });
  if (!blob) throw new Error("图片转 WebP 失败，请换一张图片");
  if (blob.size > maxProofUploadBytes) {
    throw new Error("压缩后图片仍超过 2MB，请换一张更小的图或降低分辨率");
  }
  const baseName = file.name.replace(/\.[^.]+$/, "") || "proof";
  return new File([blob], `${baseName}.webp`, { type: "image/webp", lastModified: Date.now() });
}

/** 后台管理图：允许较大体积，尽量转 WebP，失败则原样上传 */
export async function compressAdminImageForUpload(file: File): Promise<File> {
  if (!file.type.startsWith("image/")) return file;
  try {
    const bitmap = await createImageBitmap(file);
    const maxSide = 2400;
    const scale = Math.min(1, maxSide / Math.max(bitmap.width, bitmap.height));
    const width = Math.max(1, Math.round(bitmap.width * scale));
    const height = Math.max(1, Math.round(bitmap.height * scale));
    const canvas = document.createElement("canvas");
    canvas.width = width;
    canvas.height = height;
    const context = canvas.getContext("2d");
    if (!context) {
      bitmap.close?.();
      return file;
    }
    context.drawImage(bitmap, 0, 0, width, height);
    bitmap.close?.();
    const blob = await new Promise<Blob | null>((resolve) => {
      canvas.toBlob(resolve, "image/webp", 0.85);
    });
    if (!blob || blob.size >= file.size) return file;
    const baseName = file.name.replace(/\.[^.]+$/, "") || "image";
    return new File([blob], `${baseName}.webp`, { type: "image/webp", lastModified: Date.now() });
  } catch {
    return file;
  }
}
