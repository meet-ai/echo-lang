// Package dependency 实现依赖包下载功能
package dependency

import (
	"context"
	"fmt"
)

// Downloader 依赖下载器接口
// 支持从不同来源下载依赖包
type Downloader interface {
	// Download 下载依赖包到指定目录
	// packageName: 包名
	// version: 版本号
	// targetDir: 目标目录（完整路径）
	// options: 下载选项（如 Git URL、tag、branch 等）
	Download(ctx context.Context, packageName, version, targetDir string, options map[string]interface{}) error
}

// DownloadOptions 下载选项
type DownloadOptions struct {
	Source string            // 来源类型（git、http等）
	GitURL string            // Git 仓库 URL
	Tag    string            // Git 标签
	Branch string            // Git 分支
	Commit string            // Git 提交哈希
	Extra  map[string]string // 其他选项
}

// NewDownloadOptions 从配置创建下载选项
func NewDownloadOptions(source string, options map[string]interface{}) *DownloadOptions {
	opts := &DownloadOptions{
		Source: source,
		Extra:  make(map[string]string),
	}

	if options != nil {
		if git, ok := options["git"].(string); ok {
			opts.GitURL = git
		}
		if tag, ok := options["tag"].(string); ok {
			opts.Tag = tag
		}
		if branch, ok := options["branch"].(string); ok {
			opts.Branch = branch
		}
		if commit, ok := options["commit"].(string); ok {
			opts.Commit = commit
		}
	}

	return opts
}

// GetDownloader 根据来源类型获取对应的下载器
// source 支持：git、http、https；http/https 时 options 需包含 "url"（压缩包地址，.tar.gz/.tgz/.zip）
func GetDownloader(source string) (Downloader, error) {
	switch source {
	case "git":
		return NewGitDownloader(), nil
	case "http", "https":
		return NewHttpDownloader(), nil
	default:
		return nil, fmt.Errorf("unsupported source type: %s", source)
	}
}
