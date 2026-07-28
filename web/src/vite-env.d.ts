declare module "*.md?raw" {
  const content: string;
  export default content;
}

declare module "*.md?html" {
  const html: string;
  export default html;
}
