"""
GitHub 文件大小探测测试脚本

测试不同 GitHub 资源类型的文件大小获取方式，找出可靠的 Content-Length 来源。
用法: python size_probe_test.py
"""
import urllib.request
import json
import re
import sys

# Windows 下避免 Unicode 编码错误
if sys.platform == "win32":
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")

TOKEN = None  # 可选: 填入 GitHub PAT 提高 API 限额

TEST_URLS = [
    # Release Asset — 一个真实的 GitHub Release 二进制文件
    ("Release Asset", "https://github.com/cli/cli/releases/download/v2.67.0/gh_2.67.0_windows_amd64.zip"),
    # Archive via codeload (zip)
    ("Archive codeload zip", "https://codeload.github.com/cli/cli/zip/refs/tags/v2.67.0"),
    # Archive via codeload (tar.gz)
    ("Archive codeload tar.gz", "https://codeload.github.com/cli/cli/tar.gz/refs/tags/v2.67.0"),
    # Archive via github.com (302 → codeload)
    ("Archive github.com", "https://github.com/cli/cli/archive/refs/tags/v2.67.0.zip"),
    # Raw file
    ("Raw file", "https://raw.githubusercontent.com/cli/cli/refs/tags/v2.67.0/README.md"),
    # Blob page
    ("Blob page", "https://github.com/cli/cli/blob/trunk/README.md"),
]


def req(url, method="GET", headers=None, follow_redirects=True):
    """发送请求，返回 (status, headers, final_url)"""
    hdrs = {}
    if TOKEN:
        hdrs["Authorization"] = f"token {TOKEN}"
    if headers:
        hdrs.update(headers)

    # 先发 HEAD 看看是否会重定向
    req = urllib.request.Request(url, headers=hdrs, method=method)
    try:
        resp = urllib.request.urlopen(req, timeout=10)
        return resp.status, dict(resp.headers), resp.url
    except urllib.request.HTTPError as e:
        return e.code, dict(e.headers), url
    except Exception as e:
        return 0, {}, url


def parse_content_range(val):
    """从 Content-Range: bytes 0-0/12345 提取总大小"""
    if not val:
        return None
    m = re.search(r'/(\d+)', val)
    return int(m.group(1)) if m else None


def probe_with_range(url):
    """发送 Range: bytes=0-0 探测文件大小"""
    status, headers, final_url = req(url, method="GET", headers={"Range": "bytes=0-0"})
    cl = headers.get("Content-Length")
    cr = headers.get("Content-Range")
    size_from_cr = parse_content_range(cr)
    return status, cl, cr, size_from_cr, final_url


def probe_with_head(url):
    """发送 HEAD 请求查看响应头"""
    status, headers, final_url = req(url, method="HEAD")
    cl = headers.get("Content-Length")
    ce = headers.get("Content-Encoding")
    te = headers.get("Transfer-Encoding")
    return status, cl, ce, te, final_url


def probe_api_release(owner, repo, tag):
    """通过 GitHub API 查询 Release 文件大小"""
    api_url = f"https://api.github.com/repos/{owner}/{repo}/releases/tags/{tag}"
    status, headers, _ = req(api_url)
    if status != 200:
        return None
    try:
        req2 = urllib.request.Request(api_url, headers={"Accept": "application/vnd.github.v3+json"})
        if TOKEN:
            req2.add_header("Authorization", f"token {TOKEN}")
        resp = urllib.request.urlopen(req2, timeout=10)
        data = json.loads(resp.read())
        assets = data.get("assets", [])
        return [(a["name"], a["size"], a.get("browser_download_url", "")) for a in assets]
    except Exception as e:
        return f"API error: {e}"


def extract_repo_info(url):
    """从 GitHub URL 提取 owner/repo"""
    m = re.search(r'github\.com/([^/]+)/([^/]+)', url)
    return (m.group(1), m.group(2)) if m else (None, None)


def extract_tag(url):
    """从 Archive URL 提取 tag/branch"""
    m = re.search(r'/(?:tags|heads)/([^/.]+)', url)
    if m:
        return m.group(1)
    m = re.search(r'/tar\.zip/([^/.]+)', url)
    return m.group(1) if m else None


def format_size(s):
    """人性化大小"""
    if s is None:
        return "N/A"
    if isinstance(s, str):
        return s
    if s < 1024:
        return f"{s}B"
    if s < 1024 * 1024:
        return f"{s / 1024:.1f}KB"
    return f"{s / (1024 * 1024):.1f}MB"


print("=" * 80)
print("GitHub 文件大小探测测试")
print("=" * 80)

for desc, url in TEST_URLS:
    print(f"\n--- {desc} ---")
    print(f"URL: {url}")

    # 1. HEAD 请求
    status, cl, ce, te, final = probe_with_head(url)
    print(f"\n  [HEAD] {status} → {final}")
    print(f"    Content-Length:   {format_size(cl)}")
    print(f"    Content-Encoding: {ce}")
    print(f"    Transfer-Encoding:{te}")

    # 2. Range 探测
    status, cl, cr, size, final = probe_with_range(url)
    print(f"\n  [Range: bytes=0-0] {status} → {final}")
    print(f"    Content-Length:   {format_size(cl)}")
    print(f"    Content-Range:    {cr}")
    print(f"    解析总大小:        {format_size(size)}")

    # 3. API 方式（仅 Release archive）
    if "/archive/" in url or "codeload" in url:
        owner, repo = extract_repo_info(url)
        tag = extract_tag(url)
        if owner and tag:
            print(f"\n  [API] repos/{owner}/{repo}/releases/tags/{tag}")
            assets = probe_api_release(owner, repo, tag)
            if isinstance(assets, list):
                for name, size, dl_url in assets:
                    print(f"    {name}: {format_size(size)}")
                total = sum(s for _, s, _ in assets)
                print(f"    *** API asset sum: {format_size(total)} ***")
                print(f"    NOTE: asset.size != archive zip size!")
            else:
                print(f"    {assets}")

    # 4. 判定结论
    print(f"\n  [可靠来源]: ", end="")
    sources = []
    r_status, r_cl, r_cr, r_size, _ = probe_with_range(url)
    if r_status == 206 and r_size:
        sources.append(f"Range={format_size(r_size)}")
    h_status, h_cl, _, _, _ = probe_with_head(url)
    if h_cl and h_cl != "0":
        sources.append(f"HEAD={format_size(h_cl)}")
    if not sources:
        sources.append("无 (chunked)")
    print(", ".join(sources))

print()
print("=" * 80)
print("结论:")
print("  Release Asset : Range 返回 206 + Content-Range -> 准确总大小")
print("  Archive        : 302 -> codeload, chunked 无 CL -> 无法预知")
print("  Raw 文件       : HEAD 有 Content-Length -> 直接用")
print("  API asset.size : 与源码压缩包大小无关, 不可用于 Archive")
print("=" * 80)
