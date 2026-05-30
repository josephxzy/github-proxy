package service

import (
	"fmt"
	"net/http"

	"github-proxy/config"
)

// AccessControlService 仓库访问控制服务。
// 实现基于黑白名单的仓库访问控制策略
//
// 控制策略：
//   - 白名单模式：仅允许访问白名单中的仓库（非空时启用）
//   - 黑名单模式：禁止访问黑名单中的仓库（非空时启用）
//   - 两个列表都为空时不做任何限制
type AccessControlService struct {
	whiteList []string // 仓库白名单（格式："owner/repo" 或 "owner"）
	blackList []string // 仓库黑名单（格式："owner/repo" 或 "owner"）
}

// NewAccessControlService 创建访问控制服务实例。
//
// 参数:
//   - whiteList: 白名单列表
//   - blackList: 黑名单列表
func NewAccessControlService(whiteList, blackList []string) *AccessControlService {
	return &AccessControlService{
		whiteList: whiteList,
		blackList: blackList,
	}
}

// AccessResult 访问控制检查结果。
type AccessResult struct {
	Allowed      bool   // 是否允许访问
	Error        error  // 错误对象（不允许时设置）
	ErrorCode    int    // HTTP 错误状态码
	ErrorMessage string // 错误消息（面向用户的提示）
}

// CheckRepoAccess 检查对指定仓库的访问权限。
//
// 检查流程：
//  1. 验证仓库格式有效性（至少需要 owner 和 repo 两部分）
//  2. 白名单检查（如果配置了白名单，仅白名单中的仓库可通过）
//  3. 黑名单检查（如果配置了黑名单，黑名单中的仓库被拒绝）
//
// 参数:
//   - matches: 从 URL 中匹配到的仓库信息（通常包含 owner 和 repo）
//
// 返回:
//   - *AccessResult: 访问检查结果
func (s *AccessControlService) CheckRepoAccess(matches []string) *AccessResult {
	result := &AccessResult{}

	// 步骤1：验证仓库格式（至少需要 owner 和 repo 两部分）
	if len(matches) < 2 {
		result.Error = fmt.Errorf("invalid repo format")
		result.ErrorCode = http.StatusForbidden
		result.ErrorMessage = "无效的GitHub仓库格式"
		return result
	}

	// 获取最新配置（支持热更新）
	cfg := config.GetConfig()

	// 步骤2：白名单检查（仅当白名单非空时生效）
	if len(cfg.Access.WhiteList) > 0 && !s.checkList(matches, cfg.Access.WhiteList) {
		result.Error = fmt.Errorf("not in whitelist")
		result.ErrorCode = http.StatusForbidden
		result.ErrorMessage = "不在GitHub仓库白名单内"
		return result
	}

	// 步骤3：黑名单检查（仅当黑名单非空时生效）
	if len(cfg.Access.BlackList) > 0 && s.checkList(matches, cfg.Access.BlackList) {
		result.Error = fmt.Errorf("in blacklist")
		result.ErrorCode = http.StatusForbidden
		result.ErrorMessage = "GitHub仓库在黑名单内"
		return result
	}

	// 所有检查通过，允许访问
	result.Allowed = true
	return result
}

// checkList 检查仓库是否在指定的列表中。
//
// 匹配规则：
//   - 完全匹配："owner/repo" 必须完全相同
//   - 所有者匹配：仅匹配 "owner"，表示该用户的所有仓库都匹配
//
// 参数:
//   - matches: 仓库信息数组（[owner, repo, ...]）
//   - list: 要检查的白名单或黑名单
//
// 返回:
//   - bool: 是否在列表中找到匹配项
func (s *AccessControlService) checkList(matches []string, list []string) bool {
	if len(matches) < 2 {
		return false
	}

	// 构建完整的仓库标识符 "owner/repo"
	repo := matches[0] + "/" + matches[1]

	for _, item := range list {
		// 支持两种匹配方式
		if item == repo || item == matches[0] {
			return true
		}
	}
	return false
}
