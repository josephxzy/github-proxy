import { useEffect } from "react";

// 页面级 SEO 元信息管理：
// 根据传入的 PageMeta 动态更新 title、description、og 标签与 canonical 链接。

const DEFAULT_TITLE = "Github Proxy · 轻量级 GitHub 资源加速反向代理";
const DEFAULT_DESCRIPTION =
  "Github Proxy 是专注于 GitHub 资源加速的轻量级反向代理工具，内置 Vue 3 前端界面，支持仓库搜索、Release 浏览、README 预览、Git 加速、断点续传和双层限速。";
const CANONICAL_BASE = "https://josephxzy.github.io/github-proxy/";

// ensureMeta 确保指定选择器的 meta 元素存在，不存在则创建。
function ensureMeta(selector: string, attribute: "name" | "property", key: string) {
  let element = document.head.querySelector<HTMLMetaElement>(selector);
  if (!element) {
    element = document.createElement("meta");
    element.setAttribute(attribute, key);
    document.head.appendChild(element);
  }
  return element;
}

// setMetaContent 设置单个 meta 标签的 content。
function setMetaContent(attribute: "name" | "property", key: string, value: string) {
  const selector = `meta[${attribute}="${key}"]`;
  const element = ensureMeta(selector, attribute, key);
  element.setAttribute("content", value);
}

// setCanonical 设置或创建 canonical 链接标签。
function setCanonical(href: string) {
  let link = document.head.querySelector<HTMLLinkElement>('link[rel="canonical"]');
  if (!link) {
    link = document.createElement("link");
    link.setAttribute("rel", "canonical");
    document.head.appendChild(link);
  }
  link.setAttribute("href", href);
}

// PageMeta 页面自定义元信息（缺省项使用站点默认值）。
export type PageMeta = {
  title?: string;
  description?: string;
  canonicalPath?: string;
};

// ResolvedPageMeta 解析后的完整元信息。
export type ResolvedPageMeta = {
  title: string;
  description: string;
  canonical: string;
};

// resolvePageMeta 将可选的 PageMeta 解析为完整的元信息（含默认值兜底）。
export function resolvePageMeta(meta: PageMeta | null | undefined): ResolvedPageMeta {
  const title = meta?.title ? `${meta.title} · Github Proxy` : DEFAULT_TITLE;
  const description = meta?.description ?? DEFAULT_DESCRIPTION;
  const canonical = meta?.canonicalPath
    ? `${CANONICAL_BASE}${meta.canonicalPath.replace(/^\//, "")}`
    : CANONICAL_BASE;

  return { title, description, canonical };
}

// usePageMeta 在页面挂载时更新 title 与 meta 标签，卸载时还原 title。
// 服务端渲染（document 不存在）时直接跳过。
export function usePageMeta(meta: PageMeta | null | undefined) {
  useEffect(() => {
    if (typeof document === "undefined") {
      return undefined;
    }
    const { title, description, canonical } = resolvePageMeta(meta);

    const previousTitle = document.title;
    document.title = title;
    setMetaContent("name", "description", description);
    setMetaContent("property", "og:title", title);
    setMetaContent("property", "og:description", description);
    setMetaContent("property", "og:url", canonical);
    setMetaContent("name", "twitter:title", title);
    setMetaContent("name", "twitter:description", description);
    setCanonical(canonical);

    return () => {
      document.title = previousTitle;
    };
  }, [meta?.title, meta?.description, meta?.canonicalPath]);
}