// vite.config.ts 的 define 注入，构建期算好的 git 短哈希（开发环境固定为 "dev"）
declare const __APP_BUILD_ID__: string;

declare module "*.md?raw" {
  const content: string;
  export default content;
}

declare module "*.md?html" {
  const html: string;
  export default html;
}
