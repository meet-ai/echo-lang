// Package services 实现领域服务
package services

import (
	"os"
	"path/filepath"

	"echo/internal/modules/frontend/infrastructure/config"
)

// PackageType 包类型
type PackageType int

const (
	// PackageTypeInternal 项目内包（在 src/ 目录下）
	PackageTypeInternal PackageType = iota
	// PackageTypeExternal 外部依赖包（不在 src/ 目录下）
	PackageTypeExternal
	// PackageTypeUnknown 未知类型（找不到包文件）
	PackageTypeUnknown
)

// PackageClassification 包分类结果
type PackageClassification struct {
	PackagePath string
	Type        PackageType
	IsInSrc     bool
	IsInVendor  bool
	IsConfigured bool // 是否已在 echo.toml 中配置
}

// PackageClassifier 包分类器
// 职责：判断包是项目内包还是外部依赖
type PackageClassifier struct {
	projectRoot string
	config      *config.EchoConfig
}

// NewPackageClassifier 创建包分类器
func NewPackageClassifier(projectRoot string, cfg *config.EchoConfig) *PackageClassifier {
	return &PackageClassifier{
		projectRoot: projectRoot,
		config:      cfg,
	}
}

// ClassifyPackage 分类单个包
// 判断逻辑：
// 1. 检查是否在 src/ 目录下 → 项目内包
// 2. 检查是否在 vendor/ 或配置的依赖路径下 → 外部依赖
// 3. 检查是否已在 echo.toml 中配置 → 已配置的依赖
// 4. 其他情况 → 可能是新依赖（需要添加到 echo.toml）
func (c *PackageClassifier) ClassifyPackage(packagePath string) *PackageClassification {
	classification := &PackageClassification{
		PackagePath: packagePath,
		Type:        PackageTypeUnknown,
		IsInSrc:     false,
		IsInVendor:  false,
		IsConfigured: false,
	}

	// 检查是否已在 echo.toml 中配置
	if c.config != nil && c.config.HasDependency(packagePath) {
		classification.IsConfigured = true
	}

	// 检查是否在 src/ 目录下
	srcPath := filepath.Join(c.projectRoot, "src", packagePath)
	if c.packageExists(srcPath) {
		classification.Type = PackageTypeInternal
		classification.IsInSrc = true
		return classification
	}

	// 检查是否在 vendor/ 目录下（默认路径）
	vendorPath := filepath.Join(c.projectRoot, "vendor", packagePath)
	if c.packageExists(vendorPath) {
		classification.Type = PackageTypeExternal
		classification.IsInVendor = true
		return classification
	}

	// 检查是否在配置的依赖路径下
	if c.config != nil {
		depPath := c.config.GetDependencyPath(packagePath)
		if depPath != "" {
			fullDepPath := filepath.Join(c.projectRoot, depPath)
			if c.packageExists(fullDepPath) {
				classification.Type = PackageTypeExternal
				return classification
			}
		}

		// 检查是否在版本化的 vendor 路径下（vendor/{packageName}@{version}/）
		storagePath := c.config.GetDependencyStoragePath(packagePath)
		if storagePath != "" {
			fullStoragePath := filepath.Join(c.projectRoot, storagePath)
			if c.packageExists(fullStoragePath) {
				classification.Type = PackageTypeExternal
				classification.IsInVendor = true
				return classification
			}
		}
	}

	// 如果包不存在，但已在 echo.toml 中配置，认为是外部依赖
	if classification.IsConfigured {
		classification.Type = PackageTypeExternal
		return classification
	}

	// 其他情况：可能是新依赖（需要添加到 echo.toml）
	// 注意：这里不设置为 External，因为包文件不存在，可能是用户还没下载
	// 但我们可以建议添加到 echo.toml
	classification.Type = PackageTypeExternal
	return classification
}

// ClassifyPackages 分类多个包
func (c *PackageClassifier) ClassifyPackages(packagePaths []string) []*PackageClassification {
	classifications := make([]*PackageClassification, 0, len(packagePaths))
	for _, path := range packagePaths {
		classification := c.ClassifyPackage(path)
		classifications = append(classifications, classification)
	}
	return classifications
}

// packageExists 检查包是否存在
// 支持两种形式：
// 1. 单文件包：{path}.eo
// 2. 目录包：{path}/{packageName}.eo 或 {path}/（目录存在）
func (c *PackageClassifier) packageExists(path string) bool {
	// 策略1：检查是否是 .eo 文件
	if info, err := os.Stat(path + ".eo"); err == nil && !info.IsDir() {
		return true
	}

	// 策略2：检查是否是目录
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		// 检查目录中是否有包文件
		packageName := filepath.Base(path)
		packageFile := filepath.Join(path, packageName+".eo")
		if _, err := os.Stat(packageFile); err == nil {
			return true
		}
		// 如果目录存在，也认为包存在（可能是多文件包）
		return true
	}

	return false
}

// GetExternalDependencies 获取所有外部依赖包（需要添加到 echo.toml 的包）
// 过滤条件：
// 1. 不是项目内包（不在 src/ 下）
// 2. 未在 echo.toml 中配置
func (c *PackageClassifier) GetExternalDependencies(packagePaths []string) []string {
	externalDeps := make([]string, 0)

	for _, path := range packagePaths {
		classification := c.ClassifyPackage(path)
		
		// 只添加外部依赖且未配置的包
		if classification.Type == PackageTypeExternal && !classification.IsConfigured {
			externalDeps = append(externalDeps, path)
		}
	}

	return externalDeps
}

// GetInternalPackages 获取所有项目内包（在 src/ 目录下）
func (c *PackageClassifier) GetInternalPackages(packagePaths []string) []string {
	internalPackages := make([]string, 0)

	for _, path := range packagePaths {
		classification := c.ClassifyPackage(path)
		if classification.Type == PackageTypeInternal {
			internalPackages = append(internalPackages, path)
		}
	}

	return internalPackages
}
