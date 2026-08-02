export type SiteDocCategory = {
  id: string;
  title: string;
  description: string;
  docs: SiteDocEntry[];
};

export type SiteDocEntry = {
  id: string;
  title: string;
  description: string;
  sourcePath: string;
  githubPath: string;
};

export type FlattenedSiteDocEntry = SiteDocEntry & {
  categoryId: string;
  categoryTitle: string;
};

function doc(
  id: string,
  title: string,
  description: string,
  githubPath: string,
): SiteDocEntry {
  return {
    id,
    title,
    description,
    sourcePath: `../../${githubPath}`,
    githubPath,
  };
}

export const docsManifest: SiteDocCategory[] = [
  {
    id: "getting-started",
    title: "快速开始",
    description: "从安装部署到首次使用，快速上手 Github Proxy。",
    docs: [
      doc(
        "introduction",
        "项目介绍",
        "了解 Github Proxy 是什么、支持哪些功能以及适用场景。",
        "docs/public/introduction.md",
      ),
      doc(
        "installation",
        "安装部署",
        "二进制部署、源码构建、Docker 和 Nginx 反向代理配置。",
        "docs/public/getting-started/installation.md",
      ),
      doc(
        "faq",
        "常见问题",
        "快速处理 Token 失效、下载无进度条、Git 认证等高频问题。",
        "docs/public/faq.md",
      ),
      doc(
        "troubleshooting",
        "故障排查",
        "按日志、限速、水位线和 Token 认证定位问题。",
        "docs/public/troubleshooting.md",
      ),
    ],
  },
  {
    id: "configuration",
    title: "配置参考",
    description: "完整的环境变量、配置文件和白名单机制说明。",
    docs: [
      doc(
        "config-reference",
        "配置参考",
        "所有环境变量和 config.toml 配置项完整说明。",
        "docs/public/configuration/reference.md",
      ),
      doc(
        "repo-list",
        "仓库黑白名单",
        "控制可访问仓库的访问控制模式与匹配规则。",
        "docs/public/configuration/repo-list.md",
      ),
      doc(
        "token-whitelist",
        "Token 白名单",
        "配置白名单 Token 以跳过限速与 IP 限流。",
        "docs/public/configuration/token-whitelist.md",
      ),
    ],
  },
  {
    id: "guides",
    title: "使用指南",
    description: "URL 格式、Git 加速、上传、私有仓库和仓库搜索的使用方法。",
    docs: [
      doc(
        "url-format",
        "URL 格式",
        "完整 URL 和短路径两种格式，覆盖所有资源类型。",
        "docs/public/guides/url-format.md",
      ),
      doc(
        "git-acceleration",
        "Git 加速",
        "Clone、Fetch、Push 的代理配置和私有仓库认证。",
        "docs/public/guides/git-acceleration.md",
      ),
      doc(
        "push",
        "中转上传",
        "通过代理 Push 到 GitHub，享受双向加速体验。",
        "docs/public/guides/push.md",
      ),
      doc(
        "private-repo",
        "私有仓库",
        "前端设置 Token 访问私有仓库的完整流程。",
        "docs/public/guides/private-repo.md",
      ),
      doc(
        "search",
        "仓库搜索",
        "搜索语法、排序筛选和结果操作。",
        "docs/public/guides/search.md",
      ),
    ],
  },
  {
    id: "design",
    title: "设计文档",
    description: "Token 认证、下载管道、限速和 IP 限流的系统设计。",
    docs: [
      doc(
        "token",
        "Token 透传与认证",
        "三源 Token 提取、统一应用和服务器 Token 兜底机制。",
        "docs/design/token.md",
      ),
      doc(
        "download",
        "下载与断点续传",
        "下载管道、Range 预检、脚本处理和流式输出。",
        "docs/design/download.md",
      ),
      doc(
        "rate-limit",
        "限速与稳定性",
        "双层限速、水位线反压和真环形缓冲区设计。",
        "docs/design/rate-limit.md",
      ),
      doc(
        "ip-limit",
        "IP 限流设计",
        "三层限流体系、固定窗口计数器和 IPv6 归一化。",
        "docs/design/ip-limit.md",
      ),
    ],
  },
  {
    id: "project-updates",
    title: "项目动态",
    description: "查看版本更新历史和发布说明。",
    docs: [
      doc(
        "release-notes",
        "版本更新说明",
        "项目完整用户可见更新历史。",
        "docs/releases/release-notes.md",
      ),
    ],
  },
];

export const flattenedDocs: FlattenedSiteDocEntry[] = docsManifest.flatMap((category) =>
  category.docs.map((doc) => ({ ...doc, categoryId: category.id, categoryTitle: category.title })),
);