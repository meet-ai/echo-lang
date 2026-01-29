// Package dependency 实现从注册表解析 URL 并下载（T-DEV-024）
// 使用场景：source = "registry" 时，通过 GetMetadata 取下载 URL（或默认规则）再用 HTTP 下载。
package dependency

import (
	"context"
	"fmt"
	"strings"
)

// RegistryDownloader 使用 PackageRegistry 解析下载 URL 后通过 HTTP 下载
type RegistryDownloader struct {
	registry PackageRegistry
	baseURL  string
	http     *HttpDownloader
}

// NewRegistryDownloader 创建注册表下载器；baseURL 如 https://registry.echo-lang.org
func NewRegistryDownloader(registry PackageRegistry, baseURL string, http *HttpDownloader) *RegistryDownloader {
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &RegistryDownloader{registry: registry, baseURL: baseURL, http: http}
}

// Download 实现 Downloader：先 GetMetadata 取 SourceURL，否则用默认 {base}/packages/{name}/{version}.tar.gz
func (d *RegistryDownloader) Download(ctx context.Context, packageName, version, targetDir string, options map[string]interface{}) error {
	urlStr := ""
	if d.registry != nil {
		meta, err := d.registry.GetMetadata(ctx, packageName, version)
		if err != nil {
			return fmt.Errorf("registry get metadata: %w", err)
		}
		if meta != nil && meta.SourceURL != "" {
			urlStr = meta.SourceURL
		}
	}
	if urlStr == "" && d.baseURL != "" {
		urlStr = d.baseURL + "/packages/" + packageName + "/" + version + ".tar.gz"
	}
	if urlStr == "" {
		return fmt.Errorf("registry: no download URL for %s@%s", packageName, version)
	}
	opts := make(map[string]interface{})
	if options != nil {
		for k, v := range options {
			opts[k] = v
		}
	}
	opts["url"] = urlStr
	return d.http.Download(ctx, packageName, version, targetDir, opts)
}
