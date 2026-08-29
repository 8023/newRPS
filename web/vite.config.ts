import { execSync } from "node:child_process";
import { resolve } from "node:path";
import type { GetModuleInfo } from "rollup";
import { defineConfig, type Plugin } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import { markdownHtml } from "./vite-plugins/markdownHtml";

const ADMIN_PANEL_ID = resolve(__dirname, "src/ui/admin/AdminPanel.svelte").replaceAll("\\", "/");
const ANALYTICS_PANEL_ID = resolve(__dirname, "src/ui/admin/AnalyticsPanel.svelte").replaceAll("\\", "/");
const LAZY_SCENE_CHUNKS = new Set(["AdminPanel", "AnalyticsPanel", "ImageWebP", "ImageHEIF"]);
const ADMIN_ONLY_SVELTE_INTERNALS = [
  "/svelte/src/internal/shared/clone.js",
  "/svelte/src/internal/client/dom/blocks/await.js",
  "/svelte/src/internal/client/timing.js",
  "/svelte/src/internal/client/loop.js",
  "/svelte/src/internal/client/dom/elements/transitions.js",
  "/svelte/src/internal/client/dom/elements/actions.js",
  "/svelte/src/internal/client/dom/elements/bindings/size.js"
];

/**
 * 判断模块的每一条反向引用路径是否都经过指定的懒加载入口。
 *
 * 必须同时检查 importers 和 dynamicImporters：只检查其中一种会把跨动态边界共享的模块
 * 误判成某个页面私有模块。走到入口模块后停止当前路径，确保入口本身及其私有依赖可以
 * 归组；在经过入口前走到应用入口或无引用模块，则说明它不是这个场景独占的。
 */
function isOnlyImportedThrough(moduleId: string, boundaryId: string, getModuleInfo: GetModuleInfo): boolean {
  const pending = [moduleId];
  const visited = new Set<string>();

  while (pending.length > 0) {
    const currentId = pending.pop()!;
    const normalizedId = currentId.replaceAll("\\", "/").split("?", 1)[0];
    if (normalizedId === boundaryId) continue;
    if (visited.has(currentId)) continue;
    visited.add(currentId);

    const info = getModuleInfo(currentId);
    if (!info) return false;
    const importers = new Set([...info.importers, ...info.dynamicImporters]);
    if (info.isEntry || importers.size === 0) return false;
    pending.push(...importers);
  }

  return true;
}

/** 构建时守住首屏边界，避免未来依赖变化让用户入口静态反向引用懒加载场景。 */
function lazySceneBoundaryGuard(): Plugin {
  return {
    name: "lazy-scene-boundary-guard",
    generateBundle(_options, bundle) {
      const chunks = new Map(
        Object.values(bundle)
          .filter((item) => item.type === "chunk")
          .map((chunk) => [chunk.fileName, chunk])
      );

      for (const entry of chunks.values()) {
        if (!entry.isEntry) continue;
        const pending = [...entry.imports];
        const visited = new Set<string>();
        while (pending.length > 0) {
          const fileName = pending.pop()!;
          if (visited.has(fileName)) continue;
          visited.add(fileName);
          const imported = chunks.get(fileName);
          if (!imported) continue;
          if (LAZY_SCENE_CHUNKS.has(imported.name)) {
            this.error(`首屏入口 ${entry.fileName} 静态引用了懒加载场景 ${imported.name}（${fileName}）`);
          }
          pending.push(...imported.imports);
        }
      }
    }
  };
}

// 与后端 package.json build:server 脚本各自独立计算 git 短哈希：只要基于同一次
// commit 构建，两边算出的值天然相同，不需要跨脚本传递环境变量。取不到（非 git
// checkout，或 npm run dev 开发环境）时退化为固定字面量 "dev"，与后端
// internal/server/version.go 的兜底值对齐，确保开发环境两端恒相等、不误报。
function resolveBuildId(): string {
  try {
    return execSync("git rev-parse --short HEAD", { cwd: __dirname }).toString().trim() || "dev";
  } catch {
    return "dev";
  }
}

export default defineConfig({
  plugins: [markdownHtml(), svelte(), lazySceneBoundaryGuard()],
  define: {
    __APP_BUILD_ID__: JSON.stringify(resolveBuildId())
  },
  resolve: {
    // libheif-js 打包时会静态引用 fs/path/crypto（仅 Node 端分支会用到，浏览器端不可达）
    alias: {
      fs: "./src/lib/node-empty-shim.ts",
      path: "./src/lib/node-empty-shim.ts",
      crypto: "./src/lib/node-empty-shim.ts"
    }
  },
  // 构建产物放到 ../bin/dist/（与 bin/server 同目录），便于 Go 服务端静态托管，
  // 也让 Release 部署包只有一个 bin/ 目录需要覆盖（不再有独立的顶层 dist/）
  build: {
    outDir: "../bin/dist",
    emptyOutDir: true,
    rollupOptions: {
      output: {
        // 后台是用户主包之外的一级懒加载场景，数据分析又是后台里的二级懒加载场景。
        // 先判 AnalyticsPanel 独占，再判 AdminPanel 独占，可以把 LayerChart 按渲染变体拆出的
        // 零散模块分别收敛进两个真实使用入口；rpc/ws/wire/proto 和 Svelte runtime 等只要有
        // 一条路径能绕过对应入口，就不会被手动归组，仍由 Rollup 放在共享方/主包。
        manualChunks(id, { getModuleInfo }) {
          const normalizedId = id.replaceAll("\\", "/").split("?", 1)[0];

          // WebP 的调度/能力检测与 HEIF 解码器各自形成图片场景入口；WebP 的 SIMD/非 SIMD
          // Emscripten 胶水仍保留为两个自动 chunk，由 encode.js 在运行时只选择其中一个。
          if (
            normalizedId.includes("/node_modules/@jsquash/webp/encode.js") ||
            normalizedId.includes("/node_modules/@jsquash/webp/meta.js") ||
            normalizedId.includes("/node_modules/@jsquash/webp/utils.js") ||
            normalizedId.includes("/node_modules/wasm-feature-detect/")
          ) {
            return "ImageWebP";
          }
          if (
            normalizedId.includes("/node_modules/libheif-js/libheif-wasm/") ||
            normalizedId.endsWith("/src/lib/node-empty-shim.ts")
          ) {
            return "ImageHEIF";
          }

          // Svelte 的 client/index.js 是全站共享 barrel，模块级 importer 图看不出它的单个
          // re-export 最终只被 LayerChart 使用。当前产物中这些内部模块只服务后台图表；显式
          // 放入 AdminPanel 后，下面的构建期 guard 会在它们未来被首屏真正使用时立即报错。
          if (ADMIN_ONLY_SVELTE_INTERNALS.some((suffix) => normalizedId.endsWith(suffix))) return "AdminPanel";
          if (isOnlyImportedThrough(id, ANALYTICS_PANEL_ID, getModuleInfo)) return "AnalyticsPanel";
          if (isOnlyImportedThrough(id, ADMIN_PANEL_ID, getModuleInfo)) return "AdminPanel";
        },
        // Rollup 4 默认会把手动 chunk 入口的隐式依赖也卷入具名 chunk；这正是共享底层模块
        // 被错误塞进后台 chunk、再让主包静态反向引用它的成因。开启后只归入上面明确命中的
        // 模块，其余依赖继续完全交给 Rollup 的依赖图分析（Rollup 5 会把它改成默认行为）。
        onlyExplicitManualChunks: true
      }
    }
  },
  server: {
    port: 5173,
    proxy: {
      "/api": "http://127.0.0.1:9988",
      "/uploads": "http://127.0.0.1:9988",
      "/ws": {
        target: "http://127.0.0.1:9988",
        ws: true
      }
    }
  }
});
