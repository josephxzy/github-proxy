package nodereg

import (
	"context"
	"fmt"
	"strings"

	"github-proxy/config"
)

type NodeInfo struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	URL     string `json:"url"`
	IsLocal bool   `json:"isLocal"`
}

type NodeManager struct {
	config       *config.AppConfig
	instanceName string
	publicHost   string
}

func NewNodeManager(cfg *config.AppConfig) *NodeManager {
	mgr := &NodeManager{config: cfg}
	mgr.resolve()
	return mgr
}

func (mgr *NodeManager) resolve() {
	mgr.publicHost = resolvePublicHost(mgr.config)
	mgr.instanceName = resolveInstanceName(mgr.publicHost, mgr.config)

	SetLocalNodeForAPI(mgr.instanceName, mgr.publicHost)
}

func (mgr *NodeManager) InstanceName() string {
	return mgr.instanceName
}

func (mgr *NodeManager) PublicHost() string {
	return mgr.publicHost
}

func ResolvePublicHost(cfg *config.AppConfig) string {
	return resolvePublicHost(cfg)
}

func ResolveInstanceName(publicHost string, cfg *config.AppConfig) string {
	return resolveInstanceName(publicHost, cfg)
}

func resolvePublicHost(cfg *config.AppConfig) string {
	publicURL := cfg.NodeRegistry.PublicURL
	if publicURL != "" {
		publicHost := strings.TrimPrefix(publicURL, "https://")
		publicHost = strings.TrimPrefix(publicHost, "http://")
		return publicHost
	}

	host := cfg.Server.Host
	if host == "" {
		host = "127.0.0.1"
	}
	if cfg.Server.Port != 80 && cfg.Server.Port != 443 {
		return fmt.Sprintf("%s:%d", host, cfg.Server.Port)
	}
	return host
}

func resolveInstanceName(publicHost string, cfg *config.AppConfig) string {
	var instanceName string
	if strings.Contains(publicHost, ":") {
		instanceName = strings.Split(publicHost, ":")[0]
	} else {
		instanceName = publicHost
	}

	if instanceName == "0.0.0.0" || instanceName == "127.0.0.1" || instanceName == "localhost" {
		instanceName = GenerateInstanceName(cfg.Server.Host)
	}

	return instanceName
}

type NodeRegistryService struct {
	config      *config.AppConfig
	client      *NodeRegistryClient
	nodeManager *NodeManager
}

func NewNodeRegistryService(cfg *config.AppConfig) *NodeRegistryService {
	var client *NodeRegistryClient
	if urls := cfg.GetRegistryURLs(); len(urls) > 0 {
		client = InitNodeRegistry(urls...)
	}

	nodeManager := NewNodeManager(cfg)

	return &NodeRegistryService{
		config:      cfg,
		client:      client,
		nodeManager: nodeManager,
	}
}

func (s *NodeRegistryService) Start(ctx context.Context) error {
	if s.client == nil {
		return nil
	}

	s.client.SetLocalNode(s.nodeManager.InstanceName(), s.nodeManager.PublicHost())
	s.client.Start()
	return nil
}

func (s *NodeRegistryService) Stop() {
	if s.client != nil {
		s.client.Stop()
	}
}

func (s *NodeRegistryService) GetNodes() []NodeInfo {
	if s.client == nil {
		return []NodeInfo{}
	}

	local, ext := GetNodesWithInfo()
	nodes := make([]NodeInfo, 0, 1+len(ext))
	nodes = append(nodes, NodeInfo{
		ID:      1,
		Name:    local.Name,
		URL:     local.URL,
		IsLocal: true,
	})

	for i, n := range ext {
		nodes = append(nodes, NodeInfo{
			ID:      i + 2,
			Name:    n.Name,
			URL:     n.URL,
			IsLocal: false,
		})
	}

	return nodes
}

func (s *NodeRegistryService) IsSharedMode() bool {
	if s.client == nil {
		return false
	}
	return IsSharedMode()
}

func (s *NodeRegistryService) GetChallenge() string {
	if s.client == nil {
		return ""
	}
	return GetChallenge()
}

func (s *NodeRegistryService) InstanceName() string {
	local, _ := GetNodesWithInfo()
	return local.Name
}

func (s *NodeRegistryService) PublicHost() string {
	local, _ := GetNodesWithInfo()
	return local.URL
}

func (s *NodeRegistryService) HealthCheck(ctx context.Context) error {
	if s.client == nil {
		return nil
	}
	return fmt.Errorf("not implemented")
}

func (s *NodeRegistryService) IsRunning() bool {
	if s.client == nil {
		return false
	}
	return s.client.running.Load()
}
