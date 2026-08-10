package download

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// githubRegex 匹配脚本内容中的 GitHub 相关 URL。
// 前缀断言（\s'"(=,\[{;|&<>）用于排除误匹配（如注释、拼接字符串等），
// 匹配后经 transformURL 改写为代理地址。
var githubRegex = regexp.MustCompile(`(?:^|[\s'"(=,\[{;|&<>])https?://(?:github\.com|raw\.githubusercontent\.com|raw\.github\.com|gist\.githubusercontent\.com|gist\.github\.com|api\.github\.com)[^\s'")]*`)

// MaxShellSize 脚本内容处理的最大字节数（10MB）。
// 超过该限制的脚本拒绝处理，防止超大文件被整体读入内存。
const MaxShellSize = 10 * 1024 * 1024

// ProcessSmart 智能处理脚本文件内容。
// 处理流程：
//  1. 按是否 gzip 压缩读取完整内容（readShellContent）
//  2. 内容不包含任何 GitHub 域名时直接原样返回（避免无谓的字符串处理开销）
//  3. 否则将内容中的所有 GitHub URL 改写为经过代理的地址（processGitHubURLs）
//
// 返回值：
//   - io.Reader: 处理后的内容
//   - int64: 处理后的内容大小（用于重新设置 Content-Length）
//   - error: 读取或处理过程中的错误
func ProcessSmart(input io.Reader, isCompressed bool, host string) (io.Reader, int64, error) {
	content, err := readShellContent(input, isCompressed)
	if err != nil {
		return nil, 0, err
	}

	if len(content) == 0 {
		return strings.NewReader(""), 0, nil
	}

	if !bytes.Contains(content, []byte("github.com")) && !bytes.Contains(content, []byte("githubusercontent.com")) {
		return bytes.NewReader(content), int64(len(content)), nil
	}

	processed := processGitHubURLs(string(content), host)

	return strings.NewReader(processed), int64(len(processed)), nil
}

// readShellContent 读取脚本的全部内容，并处理可选的 gzip 解压。
// 通过魔数（0x1f 0x8b）探测是否为 gzip 流，避免依赖 Content-Encoding 头
// （该头可能缺失或被 Go 的 DisableCompression 配置影响）。
// 内容超过 MaxShellSize 时报错，防止内存耗尽。
func readShellContent(input io.Reader, isCompressed bool) ([]byte, error) {
	var reader io.Reader = input

	if isCompressed {
		peek := make([]byte, 2)
		n, err := input.Read(peek)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("读取数据失败: %v", err)
		}

		if n >= 2 && peek[0] == 0x1f && peek[1] == 0x8b {
			combinedReader := io.MultiReader(bytes.NewReader(peek[:n]), input)
			gzReader, err := gzip.NewReader(combinedReader)
			if err != nil {
				return nil, fmt.Errorf("gzip解压失败: %v", err)
			}
			defer gzReader.Close()
			reader = gzReader
		} else {
			reader = io.MultiReader(bytes.NewReader(peek[:n]), input)
		}
	}

	limit := int64(MaxShellSize + 1)
	limitedReader := io.LimitReader(reader, limit)

	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("读取内容失败: %v", err)
	}

	if int64(len(data)) > MaxShellSize {
		return nil, fmt.Errorf("脚本文件过大，超过 %d MB 限制", MaxShellSize/1024/1024)
	}

	return data, nil
}

// processGitHubURLs 将内容中的所有 GitHub URL 改写为经过代理的地址。
// githubRegex 的匹配可能包含前缀字符（如引号、空格），改写时保留该前缀。
func processGitHubURLs(content, host string) string {
	return githubRegex.ReplaceAllStringFunc(content, func(match string) string {
		if len(match) > 0 && match[0] != 'h' {
			prefix := match[0:1]
			url := match[1:]
			return prefix + transformURL(url, host)
		}
		return transformURL(match, host)
	})
}

// transformURL 将单个 GitHub URL 改写为代理地址（host + 原始 URL）。
// 规则：
//   - 已包含代理 host 的 URL 原样返回（避免重复代理）
//   - 补全 https:// 前缀
//   - 拼接为 "代理host/原始URL" 形式
func transformURL(url, host string) string {
	if strings.Contains(url, host) {
		return url
	}

	if strings.HasPrefix(url, "http://") {
		url = "https" + url[4:]
	} else if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "//") {
		url = "https://" + url
	}

	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}
	host = strings.TrimSuffix(host, "/")

	return host + "/" + url
}
