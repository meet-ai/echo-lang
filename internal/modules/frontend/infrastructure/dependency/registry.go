// Package dependency 定义包注册表领域接口（T-DEV-024）
// 使用场景：source = "registry" 时，从注册表解析版本列表与元数据，供 ResolveVersions 与下载使用。
package dependency

import "context"

// PackageRegistry 包注册表：按包名查询可用版本与元数据
type PackageRegistry interface {
	// ListVersions 返回某包的可用版本列表（如 ["1.0.0","1.1.0","2.0.0"]），按版本号升序或降序均可
	ListVersions(ctx context.Context, packageName string) ([]string, error)
	// GetMetadata 返回某包某版本的元数据（下载 URL、依赖等）；若注册表不提供可返回 nil,nil，下载仍走默认规则
	GetMetadata(ctx context.Context, packageName, version string) (*PackageMetadata, error)
}

// PackageMetadata 单版本元数据（可选）
type PackageMetadata struct {
	Version      string            // 版本号
	SourceURL    string            // 下载地址（tar.gz/zip），空则用默认规则
	Dependencies map[string]string // 依赖 包名 -> 版本约束，空则无传递依赖
}
