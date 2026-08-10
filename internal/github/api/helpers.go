package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github-proxy/internal/config"
	"github-proxy/internal/network"
)

// GetDefaultBranch 获取指定仓库的默认分支名称。
// 通过调用 GitHub API 获取仓库信息。
//
// 参数:
//   - owner: 仓库所有者（用户名或组织名）
//   - repo: 仓库名称
//
// 返回值:
//   - string: 默认分支名称（如 "main"、"master" 等）
//   - error: 错误信息
func GetDefaultBranch(owner, repo string) (string, error) {
	// 构造 API URL
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	// 应用认证（如果有配置 Token）
	cfg := config.GetConfig()
	if cfg.Server.GitHubToken != "" {
		req.Header.Set("Authorization", "token "+cfg.Server.GitHubToken)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// 发送请求
	client := network.GetGlobalHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// 解析 JSON 响应
	var repoInfo struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return "", err
	}

	// 验证默认分支不为空
	if repoInfo.DefaultBranch == "" {
		return "", fmt.Errorf("no default branch found")
	}

	return repoInfo.DefaultBranch, nil
}
