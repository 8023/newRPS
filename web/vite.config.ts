import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
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
