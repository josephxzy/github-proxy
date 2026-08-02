// 站点基础路径（来自 Vite 的 BASE_URL），站点可能部署在子目录下。
export const SITE_BASE_PATH = normalizeBasePath(import.meta.env.BASE_URL);

// 站点路由类型：首页或文档页（可携带具体文档 id）。
export type SiteRoute = { page: "home" } | { page: "docs"; docId?: string };

// normalizeBasePath 将 Vite 的 BASE_URL 归一化为 "/path/" 形式。
function normalizeBasePath(base: string): string {
  if (!base || base === "./") {
    return "/";
  }
  const withSlashes = `/${base.replace(/^\/+|\/+$/g, "")}/`;
  return withSlashes.replace(/\/+/g, "/");
}

// stripBasePath 移除路径中的基础路径前缀，得到相对站点的路由路径。
function stripBasePath(pathname: string): string {
  const cleanPathname = pathname.startsWith("/") ? pathname : `/${pathname}`;
  if (SITE_BASE_PATH !== "/" && cleanPathname.startsWith(SITE_BASE_PATH)) {
    return `/${cleanPathname.slice(SITE_BASE_PATH.length)}`.replace(/\/+/g, "/");
  }
  const baseWithoutTrailingSlash = SITE_BASE_PATH.replace(/\/$/, "");
  if (baseWithoutTrailingSlash && cleanPathname === baseWithoutTrailingSlash) {
    return "/";
  }
  return cleanPathname;
}

// sitePath 将站内路径拼上基础路径前缀，用于生成链接 href。
export function sitePath(path = "/"): string {
  const cleanPath = path.startsWith("/") ? path : `/${path}`;
  const baseWithoutTrailingSlash = SITE_BASE_PATH.replace(/\/$/, "");
  if (cleanPath === "/") {
    return SITE_BASE_PATH;
  }
  return `${baseWithoutTrailingSlash}${cleanPath}`;
}

// docsPath 生成文档页链接（docId 缺省时指向文档首页）。
export function docsPath(docId?: string): string {
  return sitePath(docId ? `/docs/${encodeURIComponent(docId)}` : "/docs");
}

// isSitePath 判断给定 pathname 是否属于本站点（而非外链）。
export function isSitePath(pathname: string): boolean {
  if (SITE_BASE_PATH === "/") {
    return pathname.startsWith("/");
  }
  return pathname === SITE_BASE_PATH.replace(/\/$/, "") || pathname.startsWith(SITE_BASE_PATH);
}

// normalizeRoutePath 将 pathname 归一化为站内路由路径。
export function normalizeRoutePath(pathname: string): string {
  const routePath = stripBasePath(pathname);
  return routePath === "" ? "/" : routePath;
}

// parseRoute 解析 pathname 为站点路由：
//   - /docs 或 /docs/ → 文档首页
//   - /docs/:id       → 具体文档页
//   - 其他            → 首页
export function parseRoute(pathname: string): SiteRoute {
  const routePath = normalizeRoutePath(pathname);
  if (routePath === "/docs" || routePath === "/docs/") {
    return { page: "docs" };
  }
  if (routePath.startsWith("/docs/")) {
    return { page: "docs", docId: decodeURIComponent(routePath.replace(/^\/docs\//, "")) };
  }
  return { page: "home" };
}