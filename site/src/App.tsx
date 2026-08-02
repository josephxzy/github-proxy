// 站点根组件：基于 Hash-free 的 History 路由切换首页与文档页，
// 并实现站内链接的 SPA 拦截跳转（避免整页刷新）。

import {
  ArrowRight,
  CheckCircle2,
  Cog,
  Download,
  FileText,
  GitBranch,
  Github,
  Gauge,
  Shield,
  Star,
  Zap,
} from "lucide-react";
import { useEffect, useSyncExternalStore } from "react";
import { formatStarCount, useGithubStars } from "./hooks/useGithubStars";
import { usePageMeta } from "./hooks/usePageMeta";
import DocsPage from "./DocsPage";
import { docsPath, isSitePath, parseRoute, sitePath } from "./routing";

const repoUrl = "https://github.com/josephxzy/github-proxy";
const releaseUrl = `${repoUrl}/releases/latest`;

const features = [
  { icon: Zap, title: "全类型加速", text: "Raw / Blob / Archive / Release / Gist 全覆盖" },
  { icon: GitBranch, title: "Git 加速", text: "Clone / Fetch / Push，短路径格式支持" },
  { icon: Shield, title: "私有仓库", text: "前端设置 Token，代理透传认证" },
  { icon: Gauge, title: "双层限速", text: "单流 + global，白名单免限速" },
  { icon: Cog, title: "智能反压", text: "水位线缓冲区，TCP 窗口永不归零" },
  { icon: FileText, title: "仓库搜索", text: "支持排序、筛选，一键查看 ReadMe" },
];

// 站内导航触发的自定义事件名（pushState 不触发 popstate）。
const routeChangeEvent = "github-proxy-site:navigation";

// subscribePath 订阅路径变化（浏览器前进/后退 + 站内 SPA 跳转）。
function subscribePath(callback: () => void) {
  window.addEventListener("popstate", callback);
  window.addEventListener(routeChangeEvent, callback);
  return () => {
    window.removeEventListener("popstate", callback);
    window.removeEventListener(routeChangeEvent, callback);
  };
}

// getPathSnapshot 返回当前 pathname（useSyncExternalStore 的快照）。
function getPathSnapshot() {
  return window.location.pathname;
}

// usePathRoute 通过 useSyncExternalStore 订阅当前路径（服务端预渲染时用 initialPath 兜底）。
function usePathRoute(initialPath = "/") {
  return useSyncExternalStore(subscribePath, getPathSnapshot, () => initialPath);
}

// useHistoryNavigation 拦截站内链接点击，改用 history.pushState 实现无刷新跳转。
// 跳过：被阻止的事件、非左键、带修饰键、外链、新窗口/下载链接、纯 hash 变化。
function useHistoryNavigation() {
  useEffect(() => {
    function handleClick(event: MouseEvent) {
      if (event.defaultPrevented || event.button !== 0 || event.metaKey || event.altKey || event.ctrlKey || event.shiftKey) {
        return;
      }
      const target = event.target as Element | null;
      const link = target?.closest<HTMLAnchorElement>("a[href]");
      if (!link || link.target || link.hasAttribute("download")) {
        return;
      }
      const url = new URL(link.href);
      if (url.origin !== window.location.origin || !isSitePath(url.pathname)) {
        return;
      }
      const currentPath = `${window.location.pathname}${window.location.search}${window.location.hash}`;
      const nextPath = `${url.pathname}${url.search}${url.hash}`;
      if (nextPath === currentPath) {
        return;
      }
      if (url.hash && url.pathname === window.location.pathname && url.search === window.location.search) {
        return;
      }
      event.preventDefault();
      window.history.pushState(null, "", nextPath);
      window.dispatchEvent(new Event(routeChangeEvent));
      window.scrollTo({ top: 0, behavior: "instant" });
    }

    document.addEventListener("click", handleClick);
    return () => document.removeEventListener("click", handleClick);
  }, []);
}

type AppProps = {
  initialPath?: string;
};

// App 根据当前路由渲染首页或文档页。
function App({ initialPath }: AppProps) {
  useHistoryNavigation();
  const pathname = usePathRoute(initialPath);
  const route = parseRoute(pathname);

  return (
    <main>
      <SiteNav page={route.page} />
      {route.page === "docs" ? (
        <DocsPage docId={route.docId} />
      ) : (
        <HomePage />
      )}
    </main>
  );
}

// SiteNav 顶部导航栏（含 GitHub star 展示）。
function SiteNav({ page }: { page: "home" | "docs" }) {
  const stars = useGithubStars("josephxzy", "github-proxy");
  return (
    <nav className="site-nav" aria-label="主导航">
      <a className="brand" href={sitePath("/")} aria-label="Github Proxy 首页">
        <span className="brand-mark">
          <Github size={22} />
        </span>
        <span>Github Proxy</span>
      </a>
      <div className="nav-links">
        <a href={docsPath()}>文档</a>
        {page === "home" ? (
          <>
            <a href="#features">功能</a>
            <a href="#quickstart">快速开始</a>
          </>
        ) : (
          <a href={releaseUrl}>下载</a>
        )}
        <a className="nav-github" href={repoUrl} aria-label={stars !== null ? `GitHub · ${stars} stars` : "GitHub"}>
          <Github size={15} />
          <span>GitHub</span>
          {stars !== null ? (
            <span className="nav-stars">
              <Star size={11} strokeWidth={2.4} />
              {formatStarCount(stars)}
            </span>
          ) : null}
        </a>
      </div>
    </nav>
  );
}

// HomePage 首页：Hero、功能特性、快速开始、文档与 CTA 区块。
function HomePage() {
  const stars = useGithubStars("josephxzy", "github-proxy");
  usePageMeta(null);
  return (
    <>
      <section className="hero">
        <div className="hero-content">
          <p className="eyebrow">Lightweight GitHub Reverse Proxy</p>
          <h1>GitHub 资源加速，<br />一行命令搞定</h1>
          <p className="hero-copy">
            专注于 GitHub 资源加速的轻量级反向代理工具，内置 Vue 3 前端界面，
            支持仓库搜索、Release 浏览、README 预览、Git 加速、断点续传和双层限速。
          </p>
          <div className="hero-actions">
            <a className="button primary" href={releaseUrl}>
              <Download size={18} />
              下载最新版
            </a>
            <a className="button ghost" href={repoUrl}>
              <Github size={18} />
              查看源码
            </a>
            <a className="button ghost" href={docsPath()}>
              <FileText size={18} />
              阅读文档
            </a>
            {stars !== null ? (
              <a
                className="button star"
                href={`${repoUrl}/stargazers`}
                aria-label={`GitHub ${stars} 颗 star`}
              >
                <Star size={18} strokeWidth={2.2} />
                <span>给个 Star</span>
                <span className="star-count">{formatStarCount(stars)}</span>
              </a>
            ) : null}
          </div>
        </div>
      </section>

      <section id="features" className="section">
        <div className="section-kicker">
          <p className="eyebrow">Features</p>
          <h2>轻量但完整，专注一件事</h2>
          <p>
            不做通用代理，只做 GitHub 加速。从文件下载到 Git 操作，从匿名访问到私有仓库，
            从单流限速到全局限流，一个二进制覆盖所有场景。
          </p>
        </div>
        <div className="feature-grid">
          {features.map((feature) => {
            const Icon = feature.icon;
            return (
              <article className="feature-card" key={feature.title}>
                <Icon size={24} />
                <h3>{feature.title}</h3>
                <p>{feature.text}</p>
              </article>
            );
          })}
        </div>
      </section>

      <section id="quickstart" className="section quickstart-section">
        <div className="section-kicker">
          <p className="eyebrow">Quick Start</p>
          <h2>一行命令，立即体验</h2>
          <p>下载二进制，设置 Token，启动服务，三步完成部署。</p>
        </div>
        <div className="quickstart-code">
          <pre><code># 设置 GitHub Token（可选，提升 API 限流）
export GITHUB_TOKEN=ghp_xxx

# 启动服务，默认监听 0.0.0.0:5000
./github-proxy

# 或自定义端口
SERVER_PORT=8080 ./github-proxy</code></pre>
        </div>
      </section>

      <section className="docs-teaser section">
        <div>
          <p className="eyebrow">Documentation</p>
          <h2>查看完整文档</h2>
          <p>文档站覆盖安装部署、URL 格式、Git 加速、私有仓库、配置参考和系统设计文档。</p>
        </div>
        <a className="button primary" href={docsPath()}>
          <FileText size={18} />
          打开文档
        </a>
      </section>

      <section className="cta-section">
        <p className="eyebrow">Open source</p>
        <h2>轻量、透明、可自建。专注 GitHub 资源加速。</h2>
        <div className="cta-actions">
          <a className="button primary" href={releaseUrl}>
            <Download size={18} />
            下载最新版
          </a>
          <a className="button ghost" href={repoUrl}>
            <Github size={18} />
            查看源码
          </a>
        </div>
      </section>
    </>
  );
}

export default App;