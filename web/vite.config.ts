import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { markdownHtml } from "./vite-plugins/markdownHtml";

export default defineConfig({
  plugins: [markdownHtml(), react()],
  resolve: {
    // libheif-js 打包时会静态引用 fs/path/crypto（仅 Node 端分支会用到，浏览器端不可达）
    alias: {
      fs: "./src/lib/node-empty-shim.ts",
      path: "./src/lib/node-empty-shim.ts",
      crypto: "./src/lib/node-empty-shim.ts"
    }
  },
  // 构建产物放到仓库根 dist/，便于 Go 服务端静态托管
  build: {
    outDir: "../dist",
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
