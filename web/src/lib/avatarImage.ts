import { avatarSquareSize, maxAvatarOriginalBytes, maxAvatarSquareDiffPx, maxAvatarUploadBytes } from "./constants";
import { decodeToImageSource, encodeCanvasToWebpFile } from "./imagePipeline";

/**
 * 头像上传前处理（固定流水线，输出必须是 WebP）：
 * 选头像 → 长宽比(长宽像素差值超过 20px 拒绝，要求正方形图) → 原图大小(>3MB拒绝)
 * → 短边大于 200px 时等比缩放至 200px，居中裁切成正方形 → WebP 85% 压缩
 */
export async function prepareAvatarImageForUpload(file: File): Promise<File> {
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
  if (file.size > maxAvatarOriginalBytes) {
    throw new Error("原图超过 3MB，请换一张更小的图片");
  }

  const decoded = await decodeToImageSource(file);
  try {
    const { width: rawW, height: rawH } = decoded;
    if (Math.abs(rawW - rawH) > maxAvatarSquareDiffPx) {
      throw new Error("请上传正方形图片（长宽像素差值需在 20px 以内）");
    }

    // 居中裁切成正方形：裁切边长取原图短边，再等比缩放（短边超过 200px 时缩小到 200px，否则保持原尺寸不放大）。
    const cropSide = Math.min(rawW, rawH);
    const sx = Math.max(0, Math.round((rawW - cropSide) / 2));
    const sy = Math.max(0, Math.round((rawH - cropSide) / 2));
    const outSize = cropSide > avatarSquareSize ? avatarSquareSize : cropSide;

    let size = outSize;
    let quality = 0.85;
    const baseName = (file.name.replace(/\.[^.]+$/, "") || "avatar").replace(/[^\w.-]+/g, "_");

    for (let attempt = 0; attempt < 6; attempt++) {
      const canvas = document.createElement("canvas");
      canvas.width = size;
      canvas.height = size;
      const ctx = canvas.getContext("2d");
      if (!ctx) throw new Error("浏览器不支持图片处理，请换一个浏览器");
      ctx.fillStyle = "#ffffff";
      ctx.fillRect(0, 0, size, size);
      ctx.drawImage(decoded.source, sx, sy, cropSide, cropSide, 0, 0, size, size);

      try {
        return await encodeCanvasToWebpFile(canvas, baseName, quality, maxAvatarUploadBytes);
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        if (msg === "SIZE" || attempt < 5) {
          size = Math.max(64, Math.round(size * 0.85));
          quality = Math.max(0.4, quality - 0.1);
          continue;
        }
        throw e instanceof Error ? e : new Error("图片转 WebP 失败，请换一张图片后重试");
      }
    }
    throw new Error("压缩后头像仍超过 20KB，请换一张更简单的图片");
  } finally {
    decoded.close();
  }
}
