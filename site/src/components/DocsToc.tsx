// 文档目录组件：根据 markdown 标题生成锚点目录，
// 并通过 IntersectionObserver 高亮当前阅读的标题。

import { ListTree } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import type { MouseEvent } from "react";

// DocHeading markdown 中的一个标题（h2/h3）。
export type DocHeading = {
  id: string;
  depth: 2 | 3;
  text: string;
};

// slugify 将标题文本转换为可用于锚点的 slug。
export function slugify(text: string): string {
  return (
    text
      .toLowerCase()
      .trim()
      .replace(/[`*_~()[\]{}<>]/g, "")
      .replace(/[^\p{L}\p{N}\s-]/gu, "")
      .replace(/\s+/g, "-")
      .replace(/-+/g, "-") || "section"
  );
}

// parseMarkdownHeadings 从 markdown 文本中解析出所有 h2/h3 标题。
export function parseMarkdownHeadings(markdown: string): DocHeading[] {
  return markdown
    .split(/\r?\n/)
    .map((line) => {
      const match = /^(#{2,3})\s+(.+?)\s*$/.exec(line);
      if (!match) {
        return null;
      }
      const text = match[2].replace(/#+$/, "").trim();
      return {
        id: slugify(text),
        depth: match[1].length as 2 | 3,
        text,
      };
    })
    .filter((heading): heading is DocHeading => Boolean(heading));
}

// useActiveHeading 监听滚动，返回当前处于视口内的标题 id。
function useActiveHeading(headingIds: string[]) {
  const [activeId, setActiveId] = useState<string | undefined>(headingIds[0]);

  useEffect(() => {
    if (headingIds.length === 0) {
      return undefined;
    }

    const headings = headingIds
      .map((id) => document.getElementById(id))
      .filter((element): element is HTMLElement => Boolean(element));

    if (headings.length === 0) {
      return undefined;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((entry) => entry.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)[0];
        if (visible?.target.id) {
          setActiveId(visible.target.id);
        }
      },
      {
        rootMargin: "-92px 0px -68% 0px",
        threshold: [0, 1],
      },
    );

    headings.forEach((heading) => observer.observe(heading));
    return () => observer.disconnect();
  }, [headingIds]);

  return activeId;
}

type DocsTocProps = {
  headings: DocHeading[];
};

type TocGroup = {
  heading: DocHeading;
  children: DocHeading[];
};

// groupHeadings 将标题按 h2 分组，h3 归入其前的 h2。
function groupHeadings(headings: DocHeading[]): TocGroup[] {
  const groups: TocGroup[] = [];
  for (const heading of headings) {
    if (heading.depth === 2 || groups.length === 0) {
      groups.push({ heading, children: [] });
      continue;
    }
    groups[groups.length - 1].children.push(heading);
  }
  return groups;
}

// scrollToHeading 平滑滚动到指定锚点。
function scrollToHeading(id: string) {
  const element = document.getElementById(id);
  if (element) {
    element.scrollIntoView({ behavior: "smooth", block: "start" });
  }
}

// handleTocClick 拦截目录链接默认跳转，改为平滑滚动。
function handleTocClick(event: MouseEvent<HTMLAnchorElement>, id: string) {
  event.preventDefault();
  scrollToHeading(id);
}

export function DocsToc({ headings }: DocsTocProps) {
  const headingIds = useMemo(() => headings.map((heading) => heading.id), [headings]);
  const activeId = useActiveHeading(headingIds);
  const groups = useMemo(() => groupHeadings(headings), [headings]);

  if (headings.length === 0) {
    return null;
  }

  return (
    <aside className="docs-toc" aria-label="本文目录">
      <div className="docs-toc-heading">
        <ListTree size={16} />
        <span>本文目录</span>
      </div>
      <nav>
        {groups.map((group) => {
          const childActive = group.children.some((heading) => heading.id === activeId);
          const groupActive = group.heading.id === activeId || childActive;
          return (
            <details className="docs-toc-group" key={group.heading.id} open={groupActive}>
              <summary>
                <a
                  className={group.heading.id === activeId ? "active" : ""}
                  href={`#${group.heading.id}`}
                  onClick={(event) => handleTocClick(event, group.heading.id)}
                >
                  {group.heading.text}
                </a>
              </summary>
              {group.children.map((heading) => (
                <a
                  className={`nested ${activeId === heading.id ? "active" : ""}`.trim()}
                  href={`#${heading.id}`}
                  key={heading.id}
                  onClick={(event) => handleTocClick(event, heading.id)}
                >
                  {heading.text}
                </a>
              ))}
            </details>
          );
        })}
      </nav>
    </aside>
  );
}