// Package dependency 实现 HTTP 包注册表客户端（T-DEV-024）
// 使用场景：source = "registry" 时请求注册表 ListVersions/GetMetadata，带超时与可选缓存。
package dependency

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	defaultRegistryTimeout = 10 * time.Second
	defaultCacheTTL        = 5 * time.Minute
)

// versionsResponse 兼容两种格式：["1.0.0"] 或 { "versions": ["1.0.0"] }
type versionsResponse struct {
	Versions []string `json:"versions"`
}

// metadataResponse 元数据 API 返回格式
type metadataResponse struct {
	Version      string            `json:"version"`
	SourceURL    string            `json:"source_url"`
	Dependencies map[string]string `json:"dependencies"`
}

// HTTPRegistryClient HTTP 注册表客户端，实现 PackageRegistry
type HTTPRegistryClient struct {
	baseURL    string
	client     *http.Client
	cache      map[string]cacheEntry
	cacheTTL   time.Duration
	cacheMu    sync.RWMutex
}

type cacheEntry struct {
	versions []string
	until    time.Time
}

// HTTPRegistryClientOption 可选配置
type HTTPRegistryClientOption func(*HTTPRegistryClient)

// WithRegistryTimeout 设置请求超时
func WithRegistryTimeout(d time.Duration) HTTPRegistryClientOption {
	return func(c *HTTPRegistryClient) {
		c.client.Timeout = d
	}
}

// WithRegistryCacheTTL 设置 ListVersions 缓存 TTL，0 表示不缓存
func WithRegistryCacheTTL(d time.Duration) HTTPRegistryClientOption {
	return func(c *HTTPRegistryClient) {
		c.cacheTTL = d
	}
}

// NewHTTPRegistryClient 创建 HTTP 注册表客户端；baseURL 如 https://registry.echo-lang.org，末尾勿带 /
func NewHTTPRegistryClient(baseURL string, opts ...HTTPRegistryClientOption) *HTTPRegistryClient {
	baseURL = strings.TrimSuffix(baseURL, "/")
	client := &HTTPRegistryClient{
		baseURL:  baseURL,
		client:   &http.Client{Timeout: defaultRegistryTimeout},
		cache:    make(map[string]cacheEntry),
		cacheTTL: defaultCacheTTL,
	}
	for _, o := range opts {
		o(client)
	}
	return client
}

// ListVersions 实现 PackageRegistry
func (c *HTTPRegistryClient) ListVersions(ctx context.Context, packageName string) ([]string, error) {
	if c.cacheTTL > 0 {
		c.cacheMu.RLock()
		if e, ok := c.cache[packageName]; ok && time.Now().Before(e.until) {
			c.cacheMu.RUnlock()
			return e.versions, nil
		}
		c.cacheMu.RUnlock()
	}

	url := c.baseURL + "/packages/" + packageName + "/versions"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("registry list versions: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("registry request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d for %s", resp.StatusCode, url)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("registry read body: %w", err)
	}
	return c.listVersionsFromResponse(body, packageName)
}

// 从 body 解析：先读入内存再尝试两种格式
func (c *HTTPRegistryClient) listVersionsFromResponse(body []byte, packageName string) ([]string, error) {
	var versions []string
	if err := json.Unmarshal(body, &versions); err == nil {
		c.putCache(packageName, versions)
		return versions, nil
	}
	var wrap versionsResponse
	if err := json.Unmarshal(body, &wrap); err == nil {
		c.putCache(packageName, wrap.Versions)
		return wrap.Versions, nil
	}
	return nil, fmt.Errorf("registry list versions: unsupported response format for %s", packageName)
}

func (c *HTTPRegistryClient) putCache(packageName string, versions []string) {
	if c.cacheTTL <= 0 {
		return
	}
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	c.cache[packageName] = cacheEntry{versions: versions, until: time.Now().Add(c.cacheTTL)}
}

// GetMetadata 实现 PackageRegistry
func (c *HTTPRegistryClient) GetMetadata(ctx context.Context, packageName, version string) (*PackageMetadata, error) {
	url := c.baseURL + "/packages/" + packageName + "/versions/" + version
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("registry get metadata: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("registry request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // 注册表不提供元数据时返回 nil,nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d for %s", resp.StatusCode, url)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("registry read body: %w", err)
	}
	var m metadataResponse
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, nil // 解析失败视为无元数据
	}
	return &PackageMetadata{
		Version:      m.Version,
		SourceURL:    m.SourceURL,
		Dependencies: m.Dependencies,
	}, nil
}