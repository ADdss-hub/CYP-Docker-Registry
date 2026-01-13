// Package accelerator provides image acceleration and caching functionality.
package accelerator

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// UpstreamSource represents an upstream registry source.
type UpstreamSource struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Priority int    `json:"priority"`
	Enabled  bool   `json:"enabled"`
}

// ProxyConfig represents proxy configuration.
type ProxyConfig struct {
	Upstreams []UpstreamSource `json:"upstreams"`
}

// ProxyService handles proxying requests to upstream registries.
type ProxyService struct {
	cache      *LRUCache
	upstreams  []UpstreamSource
	httpClient *http.Client
	configPath string
	mu         sync.RWMutex
}

// NewProxyService creates a new proxy service.
func NewProxyService(cache *LRUCache, configPath string) (*ProxyService, error) {
	service := &ProxyService{
		cache:      cache,
		configPath: configPath,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Load upstream configuration
	if err := service.loadConfig(); err != nil {
		// Use default upstreams if config load fails
		service.upstreams = getDefaultUpstreams()
	}

	return service, nil
}

// getDefaultUpstreams returns default upstream sources.
func getDefaultUpstreams() []UpstreamSource {
	return []UpstreamSource{
		{Name: "Docker Hub", URL: "https://registry-1.docker.io", Priority: 1, Enabled: true},
		{Name: "阿里云", URL: "https://registry.cn-hangzhou.aliyuncs.com", Priority: 2, Enabled: true},
	}
}


// ProxyPull pulls an image layer through the proxy, using cache if available.
func (p *ProxyService) ProxyPull(name, digest string) (io.ReadCloser, int64, error) {
	// Check cache first
	if reader, size, err := p.cache.Get(digest); err == nil {
		return reader, size, nil
	}

	// Try upstreams in priority order
	upstreams := p.GetUpstreams()
	var lastErr error

	for _, upstream := range upstreams {
		if !upstream.Enabled {
			continue
		}

		reader, size, err := p.pullFromUpstream(upstream, name, digest)
		if err != nil {
			lastErr = err
			continue
		}

		// Cache the blob while returning it
		cachedReader, cachedSize, err := p.cacheAndReturn(digest, reader, size)
		if err != nil {
			reader.Close()
			lastErr = err
			continue
		}

		return cachedReader, cachedSize, nil
	}

	if lastErr != nil {
		return nil, 0, fmt.Errorf("all upstreams failed: %w", lastErr)
	}
	return nil, 0, fmt.Errorf("no enabled upstreams available")
}

// ProxyPullManifest pulls a manifest through the proxy.
func (p *ProxyService) ProxyPullManifest(name, reference string) ([]byte, string, error) {
	upstreams := p.GetUpstreams()
	var lastErr error

	for _, upstream := range upstreams {
		if !upstream.Enabled {
			continue
		}

		data, contentType, err := p.pullManifestFromUpstream(upstream, name, reference)
		if err != nil {
			lastErr = err
			continue
		}

		return data, contentType, nil
	}

	if lastErr != nil {
		return nil, "", fmt.Errorf("all upstreams failed: %w", lastErr)
	}
	return nil, "", fmt.Errorf("no enabled upstreams available")
}

// pullFromUpstream pulls a blob from a specific upstream.
func (p *ProxyService) pullFromUpstream(upstream UpstreamSource, name, digest string) (io.ReadCloser, int64, error) {
	url := fmt.Sprintf("%s/v2/%s/blobs/%s", upstream.URL, name, digest)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Add Docker registry headers
	req.Header.Set("Accept", "application/vnd.docker.image.rootfs.diff.tar.gzip")
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("upstream request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	return resp.Body, resp.ContentLength, nil
}

// pullManifestFromUpstream pulls a manifest from a specific upstream.
func (p *ProxyService) pullManifestFromUpstream(upstream UpstreamSource, name, reference string) ([]byte, string, error) {
	url := fmt.Sprintf("%s/v2/%s/manifests/%s", upstream.URL, name, reference)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add Docker registry headers
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("upstream request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	return data, contentType, nil
}

// cacheAndReturn caches the blob and returns a reader.
func (p *ProxyService) cacheAndReturn(digest string, reader io.ReadCloser, size int64) (io.ReadCloser, int64, error) {
	defer reader.Close()

	// Store in cache
	_, err := p.cache.Put(digest, reader)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to cache blob: %w", err)
	}

	// Return from cache
	return p.cache.Get(digest)
}


// GetUpstreams returns upstreams sorted by priority.
func (p *ProxyService) GetUpstreams() []UpstreamSource {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Return a copy sorted by priority
	upstreams := make([]UpstreamSource, len(p.upstreams))
	copy(upstreams, p.upstreams)

	sort.Slice(upstreams, func(i, j int) bool {
		return upstreams[i].Priority < upstreams[j].Priority
	})

	return upstreams
}

// SetUpstreams updates the upstream sources.
func (p *ProxyService) SetUpstreams(upstreams []UpstreamSource) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.upstreams = upstreams
	return p.saveConfig()
}

// AddUpstream adds a new upstream source.
func (p *ProxyService) AddUpstream(upstream UpstreamSource) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check for duplicate
	for _, u := range p.upstreams {
		if u.Name == upstream.Name {
			return fmt.Errorf("upstream %s already exists", upstream.Name)
		}
	}

	p.upstreams = append(p.upstreams, upstream)
	return p.saveConfig()
}

// RemoveUpstream removes an upstream source by name.
func (p *ProxyService) RemoveUpstream(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, u := range p.upstreams {
		if u.Name == name {
			p.upstreams = append(p.upstreams[:i], p.upstreams[i+1:]...)
			return p.saveConfig()
		}
	}

	return fmt.Errorf("upstream %s not found", name)
}

// UpdateUpstream updates an existing upstream source.
func (p *ProxyService) UpdateUpstream(name string, upstream UpstreamSource) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, u := range p.upstreams {
		if u.Name == name {
			p.upstreams[i] = upstream
			return p.saveConfig()
		}
	}

	return fmt.Errorf("upstream %s not found", name)
}

// EnableUpstream enables or disables an upstream source.
func (p *ProxyService) EnableUpstream(name string, enabled bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, u := range p.upstreams {
		if u.Name == name {
			p.upstreams[i].Enabled = enabled
			return p.saveConfig()
		}
	}

	return fmt.Errorf("upstream %s not found", name)
}

// GetCache returns the underlying cache.
func (p *ProxyService) GetCache() *LRUCache {
	return p.cache
}

// getConfigPath returns the path to the proxy config file.
func (p *ProxyService) getConfigPath() string {
	if p.configPath != "" {
		return filepath.Join(p.configPath, "proxy_config.json")
	}
	return "proxy_config.json"
}

// loadConfig loads proxy configuration from disk.
func (p *ProxyService) loadConfig() error {
	configPath := p.getConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			p.upstreams = getDefaultUpstreams()
			return nil
		}
		return err
	}

	var config ProxyConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	p.upstreams = config.Upstreams
	return nil
}

// saveConfig saves proxy configuration to disk.
func (p *ProxyService) saveConfig() error {
	config := ProxyConfig{
		Upstreams: p.upstreams,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	configPath := p.getConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// CheckUpstreamHealth checks if an upstream is reachable.
func (p *ProxyService) CheckUpstreamHealth(name string) (bool, error) {
	p.mu.RLock()
	var upstream *UpstreamSource
	for _, u := range p.upstreams {
		if u.Name == name {
			upstream = &u
			break
		}
	}
	p.mu.RUnlock()

	if upstream == nil {
		return false, fmt.Errorf("upstream %s not found", name)
	}

	url := fmt.Sprintf("%s/v2/", upstream.URL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, nil // Unreachable but not an error
	}
	defer resp.Body.Close()

	// Docker Registry V2 returns 200 or 401 for valid registries
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized, nil
}
