// 通过 Vite 的 import.meta.glob 打包加载文档中的图片资源（svg/png/jpg 等），
// 构建后按 URL 形式引用，供 markdown 渲染时解析相对图片路径。

const assetModules = import.meta.glob(
  "../../docs/public/**/*.{svg,png,jpg,jpeg,webp,gif}",
  { eager: true, import: "default", query: "?url" },
);

// normalizeDocAssetKey 规范化资源路径：去除 "./"、".." 等相对段，得到稳定的 key。
function normalizeDocAssetKey(path: string): string {
  return path
    .split("/")
    .reduce<string[]>((parts, part) => {
      if (!part || part === ".") {
        return parts;
      }
      if (part === "..") {
        parts.pop();
        return parts;
      }
      parts.push(part);
      return parts;
    }, [])
    .join("/");
}

// basenameOf 返回路径的最后一段（文件名）。
function basenameOf(path: string): string {
  const last = path.split("/").pop();
  return last ?? path;
}

type AssetMaps = {
  byPath: Record<string, string>; // 规范化路径 → 资源 URL
  byBasename: Record<string, string>; // 文件名 → 资源 URL（用于仅按文件名引用的场景）
};

// buildAssetMaps 构建资源索引映射。
function buildAssetMaps(modules: Record<string, string>): AssetMaps {
  const byPath: Record<string, string> = {};
  const byBasename: Record<string, string> = {};
  for (const [rawPath, url] of Object.entries(modules)) {
    const normalized = normalizeDocAssetKey(rawPath);
    byPath[normalized] = url;
    const base = basenameOf(normalized);
    if (base) {
      byBasename[base] = url;
    }
  }
  return { byPath, byBasename };
}

const { byPath, byBasename } = buildAssetMaps(assetModules as Record<string, string>);

// safeDecode 安全地解码 URL 编码（解码失败时原样返回）。
function safeDecode(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

// resolveDocAssetUrl 将文档中的相对图片路径解析为打包后的资源 URL。
// 解析优先级：
//  1. 绝对 URL / 协议相对 URL / data: URI → 原样返回
//  2. 按"文档目录 + 相对路径"组合匹配 byPath
//  3. 仅按文件名匹配 byBasename
//  4. 均未命中 → 原样返回（保持相对引用）
export function resolveDocAssetUrl(
  docSourcePath: string,
  assetPath: string | undefined,
): string | undefined {
  if (!assetPath || /^(https?:)?\/\//.test(assetPath) || assetPath.startsWith("data:")) {
    return assetPath;
  }
  const decodedAsset = safeDecode(assetPath);
  const sourceParts = docSourcePath.split("/");
  sourceParts.pop();
  const combined = `${sourceParts.join("/")}/${decodedAsset}`;
  const normalized = normalizeDocAssetKey(combined);
  if (byPath[normalized]) {
    return byPath[normalized];
  }
  const base = basenameOf(normalizeDocAssetKey(decodedAsset));
  if (base && byBasename[base]) {
    return byBasename[base];
  }
  return assetPath;
}