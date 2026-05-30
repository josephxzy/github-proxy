package github

import (
	"fmt"
	"net/http"
	"strings"
)

type URLNormalizer struct{}

func NewURLNormalizer() *URLNormalizer {
	return &URLNormalizer{}
}

type NormalizeResult struct {
	Valid         bool
	NormalizedURL string
	Error         error
	ErrorCode     int
	ErrorMessage  string
}

func (u *URLNormalizer) Normalize(rawPath string) *NormalizeResult {
	result := &NormalizeResult{}

	rawPath = u.ensureHTTPS(rawPath)

	if !u.validateGitHubURL(rawPath, result) {
		return result
	}

	if IsBlobURL(rawPath) {
		rawPath = strings.Replace(rawPath, "/blob/", "/raw/", 1)
	}

	result.Valid = true
	result.NormalizedURL = rawPath
	return result
}

func (u *URLNormalizer) ensureHTTPS(rawPath string) string {
	if strings.HasPrefix(rawPath, "https://") {
		return rawPath
	}
	if strings.HasPrefix(rawPath, "http://") {
		return "https://" + rawPath[7:]
	}
	return "https://" + rawPath
}

func (u *URLNormalizer) validateGitHubURL(rawPath string, result *NormalizeResult) bool {
	matches := MatchURL(rawPath)
	if matches == nil {
		result.Error = fmt.Errorf("invalid input")
		result.ErrorCode = http.StatusForbidden
		result.ErrorMessage = "无效输入"
		return false
	}
	return true
}

func (u *URLNormalizer) IsAPIURL(url string) bool {
	return IsGitHubAPIURL(url)
}

func (u *URLNormalizer) IsScriptURL(url string) bool {
	return IsScriptURL(url)
}

func (u *URLNormalizer) IsBlobURL(url string) bool {
	return IsBlobURL(url)
}

func (u *URLNormalizer) IsReleaseAPIURL(url string) bool {
	return IsReleaseAPIURL(url)
}

func (u *URLNormalizer) MatchURL(url string) []string {
	return MatchURL(url)
}

func IsAPIRequest(url string) bool {
	return IsGitHubAPIURL(url)
}

func CheckRepoAccess(matches []string) (bool, string) {
	if len(matches) < 2 {
		return false, "invalid repo format"
	}
	return true, ""
}
