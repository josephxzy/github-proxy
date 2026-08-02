// 通过 Vite 的 import.meta.glob 打包加载所有 markdown 文档，
// 构建时按需合并到 bundle 中，供文档页面渲染。

const publicDocModules = import.meta.glob("../../docs/public/**/*.md", {
  eager: true,
  import: "default",
  query: "?raw",
});

const designDocModules = import.meta.glob("../../docs/design/*.md", {
  eager: true,
  import: "default",
  query: "?raw",
});

const releaseDocModules = import.meta.glob("../../docs/releases/release-notes.md", {
  eager: true,
  import: "default",
  query: "?raw",
});

const docModules = {
  ...publicDocModules,
  ...designDocModules,
  ...releaseDocModules,
} as Record<string, string>;

// getDocContent 根据文档源路径（如 "docs/public/faq.md"）返回其 markdown 原文。
export function getDocContent(sourcePath: string): string | undefined {
  return docModules[sourcePath];
}