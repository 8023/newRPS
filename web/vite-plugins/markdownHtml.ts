import fs from "node:fs";
import MarkdownIt from "markdown-it";
import type { Plugin } from "vite";

const QUERY_SUFFIX = ".md?html";

// 把 "*.md?html" 的导入在构建期编译成静态 HTML 字符串，而不是把 Markdown 原文
// 塞进浏览器包、留到运行时用手写解析器现场渲染（曾经的做法容易在引用块/换行等
// 细节上跑偏，且每次打开面板都要重新解析一遍）。
export function markdownHtml(): Plugin {
  const md = new MarkdownIt({ linkify: true });

  return {
    name: "markdown-html",
    load(id) {
      if (!id.endsWith(QUERY_SUFFIX)) return null;
      const filePath = id.slice(0, -".html".length);
      const raw = fs.readFileSync(filePath, "utf-8");
      const html = md.render(raw);
      return `export default ${JSON.stringify(html)};`;
    },
    handleHotUpdate(ctx) {
      if (!ctx.file.endsWith(".md")) return;
      const mods = ctx.server.moduleGraph.getModulesByFile(ctx.file);
      if (!mods) return;
      return Array.from(mods);
    }
  };
}
