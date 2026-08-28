import { execSync } from "node:child_process";
import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import { markdownHtml } from "./vite-plugins/markdownHtml";

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
  plugins: [markdownHtml(), svelte()],
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
    emptyOutDir: true
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
