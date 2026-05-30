// Package nodereg 提供节点注册与发现服务。
// 支持多节点部署场景下的节点注册、心跳保活、Token 续约和节点列表同步功能。
package nodereg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type LocalNode struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ExternalNode struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type apiNode struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type regReq struct {
	Domain   string `json:"domain"`
	Version  string `json:"version"`
	Capacity string `json:"capacity"`
}

type regResp struct {
	Token     string    `json:"token"`
	Challenge string    `json:"challenge"`
	Nodes     []apiNode `json:"nodes"`
}

type nodesResp struct {
	Version int       `json:"version"`
	Nodes   []apiNode `json:"nodes"`
}

type NodeRegistryClient struct {
	httpClient   *http.Client
	registryURLs []string
	currentIndex int
	mu           sync.RWMutex
	stopCh       chan struct{}
	running      atomic.Bool

	token       string
	challenge   string
	tokenExpiry time.Time

	nodes     []ExternalNode
	nodeMap   map[string]*ExternalNode
	localNode LocalNode
	nodeURL   string
	version   int
}

var globalInstance *NodeRegistryClient

var localNodeForAPI LocalNode
var extNodesForAPI []ExternalNode

var challengeValue string
var challengeMu sync.RWMutex

func NewNodeRegistryClient(registryURLs ...string) *NodeRegistryClient {
	urls := make([]string, 0, len(registryURLs))
	for _, u := range registryURLs {
		if u = strings.TrimSpace(u); u != "" {
			urls = append(urls, u)
		}
	}
	if len(urls) == 0 {
		return nil
	}
	return &NodeRegistryClient{
		registryURLs: urls,
		stopCh:       make(chan struct{}),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     30 * time.Second,
			},
		},
	}
}

func InitNodeRegistry(registryURLs ...string) *NodeRegistryClient {
	client := NewNodeRegistryClient(registryURLs...)
	globalInstance = client
	return client
}

func GlobalInstance() *NodeRegistryClient {
	return globalInstance
}

func SetLocalNodeForAPI(name, url string) {
	localNodeForAPI = LocalNode{Name: name, URL: url}
	if globalInstance != nil {
		globalInstance.mu.Lock()
		globalInstance.localNode = LocalNode{Name: name, URL: url}
		globalInstance.nodeURL = url
		globalInstance.mu.Unlock()
	}
}

func SetExtNodes(nodes []ExternalNode) {
	extNodesForAPI = nodes
}

func GetNodesWithInfo() (LocalNode, []ExternalNode) {
	return localNodeForAPI, extNodesForAPI
}

func IsSharedMode() bool {
	return globalInstance != nil && globalInstance.token != ""
}

func GetChallenge() string {
	challengeMu.RLock()
	defer challengeMu.RUnlock()
	return challengeValue
}

func SetChallenge(c string) {
	challengeMu.Lock()
	defer challengeMu.Unlock()
	challengeValue = c
}

func GenerateInstanceName(host string) string {
	if host != "" && host != "0.0.0.0" && host != "127.0.0.1" && host != "localhost" {
		return host
	}
	return "localhost"
}

func (c *NodeRegistryClient) SetLocalNode(name, url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.localNode = LocalNode{Name: name, URL: url}
	c.nodeURL = url
}

func (c *NodeRegistryClient) mergeAndSetNodes(apiNodes []apiNode) {
	c.mu.Lock()
	defer c.mu.Unlock()

	localURL := strings.ToLower(c.nodeURL)

	if c.nodeMap == nil {
		c.nodeMap = make(map[string]*ExternalNode)
	}

	for _, n := range apiNodes {
		nodeURL := strings.ToLower(n.URL)
		if nodeURL == localURL || nodeURL == "" {
			continue
		}

		if _, ok := c.nodeMap[nodeURL]; !ok {
			node := &ExternalNode{
				Name: n.Name,
				URL:  n.URL,
			}
			c.nodeMap[nodeURL] = node
			c.nodes = append(c.nodes, *node)
		}
	}

	sort.Slice(c.nodes, func(i, j int) bool {
		return c.nodes[i].Name < c.nodes[j].Name
	})

	for i := range c.nodes {
		c.nodes[i].ID = i + 1
		if ptr, ok := c.nodeMap[strings.ToLower(c.nodes[i].URL)]; ok {
			ptr.ID = i + 1
		}
	}

	SetExtNodes(c.nodes)
}

func (c *NodeRegistryClient) doPost(path string, body []byte) (*http.Response, error) {
	baseURL := c.currentRegistryURL()
	req, err := http.NewRequest("POST", baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.httpClient.Do(req)
}

func (c *NodeRegistryClient) doGet(reqURL string) (*http.Response, error) {
	baseURL := c.currentRegistryURL()
	return c.httpClient.Get(baseURL + reqURL)
}

func (c *NodeRegistryClient) currentRegistryURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.currentIndex < len(c.registryURLs) {
		return c.registryURLs[c.currentIndex]
	}
	return c.registryURLs[0]
}

func (c *NodeRegistryClient) Start() {
	if c == nil || c.running.Swap(true) {
		return
	}
	globalInstance = c
	c.registerToCurrent()
	go c.runPullLoop(300)
	go c.runTokenRenewalLoop()
}

func (c *NodeRegistryClient) Stop() {
	if c == nil || !c.running.Swap(false) {
		return
	}
	close(c.stopCh)
}

func (c *NodeRegistryClient) registerToCurrent() error {
	c.mu.RLock()
	domain := c.localNode.Name
	if domain == "" {
		domain = "unknown"
	}
	c.mu.RUnlock()

	reqBody := regReq{
		Domain:   domain,
		Version:  "v1.0.0",
		Capacity: "medium",
	}
	data, _ := json.Marshal(reqBody)
	resp, err := c.doPost("/api/v1/register", data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var r regResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return err
	}
	c.mu.Lock()
	c.token = r.Token
	c.challenge = r.Challenge
	c.tokenExpiry = time.Now().Add(24 * time.Hour)
	c.mu.Unlock()
	SetChallenge(r.Challenge)
	c.mergeAndSetNodes(r.Nodes)

	return nil
}

func (c *NodeRegistryClient) pullNodes() {
	reqURL := "/api/v1/nodes?token=" + c.token
	resp, err := c.doGet(reqURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return
	}
	var r nodesResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return
	}
	c.mu.Lock()
	if r.Version > c.version {
		c.version = r.Version
		c.mu.Unlock()
		c.mergeAndSetNodes(r.Nodes)
	} else {
		c.mu.Unlock()
	}
}

func (c *NodeRegistryClient) runPullLoop(sec int) {
	if sec <= 0 {
		sec = 300
	}
	ticker := time.NewTicker(time.Duration(sec) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.pullNodes()
		case <-c.stopCh:
			return
		}
	}
}

func (c *NodeRegistryClient) runTokenRenewalLoop() {
	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.renewToken()
		case <-c.stopCh:
			return
		}
	}
}

func (c *NodeRegistryClient) renewToken() {
	c.mu.RLock()
	timeUntilExpiry := time.Until(c.tokenExpiry)
	c.mu.RUnlock()

	if timeUntilExpiry > 0 && timeUntilExpiry < 2*time.Hour {
		fmt.Printf("[NodeRegistry] Token 即将过期（剩余 %v），尝试续约...\n", timeUntilExpiry.Round(time.Minute))
	} else {
		fmt.Println("[NodeRegistry] 尝试续约 Token...")
	}

	if err := c.registerToCurrent(); err != nil {
		fmt.Printf("[NodeRegistry] Token 续约失败: %v，5秒后重试...\n", err)
		go func() {
			time.Sleep(5 * time.Second)
			select {
			case <-c.stopCh:
				return
			default:
				if retryErr := c.registerToCurrent(); retryErr != nil {
					fmt.Printf("[NodeRegistry] 重试续约仍然失败: %v，1分钟后再次尝试...\n", retryErr)
					time.Sleep(55 * time.Second)
					select {
					case <-c.stopCh:
						return
					default:
						c.renewToken()
					}
				} else {
					fmt.Println("[NodeRegistry] 重试续约成功")
				}
			}
		}()
	} else {
		fmt.Println("[NodeRegistry] Token 续约成功")
	}
}
