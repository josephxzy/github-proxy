package github

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github-proxy/config"
)

var shortGitHubPathExp = regexp.MustCompile(`^[^/\.]+/[^/]+(?:/(?:releases|archive|blob|raw|info|git-)/.*)?$`)

func IsShortGitHubPath(path string) bool {
	return shortGitHubPathExp.MatchString(path)
}

var githubExps = []*regexp.Regexp{
	regexp.MustCompile(`^(?:https?://)?github\.com/([^/]+)/([^/]+)/(?:releases|archive)/.*`),
	regexp.MustCompile(`^(?:https?://)?github\.com/([^/]+)/([^/]+)/(?:blob|raw)/.*`),
	regexp.MustCompile(`^(?:https?://)?github\.com/([^/]+)/([^/]+)/(?:info|git-).*`),
	regexp.MustCompile(`^(?:https?://)?raw\.github(?:usercontent|github)\.com/([^/]+)/([^/]+)/.+?/.+`),
	regexp.MustCompile(`^(?:https?://)?gist\.(?:githubusercontent|github)\.com/([^/]+)/([^/]+).*`),
	regexp.MustCompile(`^(?:https?://)?(.+)`),
}

func MatchURL(u string) []string {
	for _, exp := range githubExps {
		if matches := exp.FindStringSubmatch(u); matches != nil {
			return matches[1:]
		}
	}
	return nil
}

func IsScriptURL(url string) bool {
	lower := strings.ToLower(url)
	return strings.HasSuffix(lower, ".sh") || strings.HasSuffix(lower, ".ps1")
}

func IsBlobURL(u string) bool {
	return githubExps[1].MatchString(u)
}

var releaseAPIExp = regexp.MustCompile(`api\.github\.com/repos/[^/]+/[^/]+/releases`)

func IsReleaseAPIURL(u string) bool {
	if !strings.Contains(u, "api.github.com") {
		return false
	}
	return releaseAPIExp.MatchString(u)
}

func IsGitHubAPIURL(u string) bool {
	return strings.Contains(u, "api.github.com")
}

func ApplyGitHubToken(req *http.Request, url string) {
	cfg := config.GetConfig()
	if cfg.Server.GitHubToken != "" && strings.Contains(url, "api.github.com/repos/") && strings.Contains(url, "/releases") {
		req.Header.Set("Authorization", "token "+cfg.Server.GitHubToken)
	}
}

func ExtractUserToken(r *http.Request) string {
	if token := r.Header.Get("X-GitHub-Token"); token != "" {
		return token
	}
	return ""
}

func ExtractUserTokenFromQuery(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return ""
	}
	return parsed.Query().Get("token")
}

func ApplyUserToken(req *http.Request, userToken string) {
	if userToken != "" {
		req.Header.Set("Authorization", "token "+userToken)
	}
}

func StripProxyQueryParams(rawURL string) string {
	if !strings.Contains(rawURL, "?") {
		return rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := parsed.Query()
	q.Del("token")
	q.Del("fast")
	parsed.RawQuery = q.Encode()
	return parsed.String()
}
