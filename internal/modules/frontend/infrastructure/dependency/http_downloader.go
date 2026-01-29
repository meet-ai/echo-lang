// Package dependency 实现 HTTP/HTTPS 源依赖下载
// 使用场景：echo.toml 中配置 source = "http" 或 "https"，并指定 url 为 tar.gz/zip 的地址，
// 执行 echoc fetch 时从该 URL 下载并解压到 vendor 目录。
package dependency

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// HttpDownloader HTTP/HTTPS 下载器实现
type HttpDownloader struct {
	client *http.Client
}

// NewHttpDownloader 创建 HTTP 下载器
func NewHttpDownloader() *HttpDownloader {
	return &HttpDownloader{
		client: &http.Client{},
	}
}

// Download 从 URL 下载压缩包并解压到 targetDir
// options 需包含 "url"（string）：压缩包地址，支持 .tar.gz、.tgz、.zip
func (d *HttpDownloader) Download(ctx context.Context, packageName, version, targetDir string, options map[string]interface{}) error {
	urlStr := ""
	if options != nil {
		if u, ok := options["url"].(string); ok && u != "" {
			urlStr = u
		}
	}
	if urlStr == "" {
		return fmt.Errorf("url is required for http/https source")
	}

	parentDir := filepath.Dir(targetDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}
	if _, err := os.Stat(targetDir); err == nil {
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("failed to remove existing directory: %w", err)
		}
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", urlStr, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// 先写入临时文件，以便 tar.gz 可读两遍（先探测顶层目录再解压）
	tmpFile, err := os.CreateTemp("", "echo-http-download-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	lower := strings.ToLower(urlStr)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return d.extractZipFromFile(tmpPath, targetDir)
	case strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
		return d.extractTarGzFromFile(tmpPath, targetDir)
	default:
		return fmt.Errorf("unsupported archive format (use .tar.gz, .tgz or .zip): %s", urlStr)
	}
}

// extractTarGzFromFile 从本地 tar.gz 文件解压；若仅有一个顶层目录则去掉该层再解压到 targetDir
func (d *HttpDownloader) extractTarGzFromFile(tarGzPath, targetDir string) error {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	topDir, err := d.detectTarTopDir(tr)
	if err != nil {
		return err
	}
	gz.Close()
	f.Close()

	f2, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer f2.Close()
	gz2, err := gzip.NewReader(f2)
	if err != nil {
		return err
	}
	defer gz2.Close()
	tr2 := tar.NewReader(gz2)
	return d.untarToDir(tr2, targetDir, topDir)
}

func (d *HttpDownloader) detectTarTopDir(tr *tar.Reader) (string, error) {
	var topDir string
	var entries []string
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		name := filepath.Clean(h.Name)
		if name == "." || name == ".." {
			continue
		}
		entries = append(entries, name)
		if topDir == "" {
			if idx := strings.Index(name, string(filepath.Separator)); idx >= 0 {
				topDir = name[:idx]
			} else {
				topDir = name
			}
		}
	}
	if topDir == "" {
		return "", nil
	}
	allUnderTop := true
	for _, e := range entries {
		if e != topDir && !strings.HasPrefix(e, topDir+string(filepath.Separator)) {
			allUnderTop = false
			break
		}
	}
	if !allUnderTop {
		return "", nil
	}
	return topDir, nil
}

func (d *HttpDownloader) untarToDir(tr *tar.Reader, targetDir, stripTopDir string) error {
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(h.Name)
		if name == "." || name == ".." {
			continue
		}
		if stripTopDir != "" && (name == stripTopDir || strings.HasPrefix(name, stripTopDir+string(filepath.Separator))) {
			if name == stripTopDir {
				continue
			}
			name = name[len(stripTopDir)+1:]
		}
		dest := filepath.Join(targetDir, name)
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dest, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(h.Mode)&0755)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}

// extractZipFromFile 从本地 zip 文件解压；若仅有一个顶层目录则去掉该层再解压到 targetDir
func (d *HttpDownloader) extractZipFromFile(zipPath, targetDir string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer zr.Close()

	var topDir string
	for _, f := range zr.File {
		name := filepath.Clean(f.Name)
		if name == "." || name == ".." {
			continue
		}
		if idx := strings.Index(name, "/"); idx >= 0 {
			if topDir == "" {
				topDir = name[:idx]
			}
			break
		}
	}
	stripTop := false
	if topDir != "" {
		count := 0
		for _, f := range zr.File {
			name := filepath.Clean(f.Name)
			if name == topDir || strings.HasPrefix(name, topDir+"/") {
				count++
			}
		}
		stripTop = count == len(zr.File)
	}

	for _, f := range zr.File {
		name := filepath.Clean(f.Name)
		if name == "." || name == ".." {
			continue
		}
		if stripTop && (name == topDir || strings.HasPrefix(name, topDir+"/")) {
			if name == topDir {
				continue
			}
			name = name[len(topDir)+1:]
		}
		dest := filepath.Join(targetDir, name)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(dest, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
