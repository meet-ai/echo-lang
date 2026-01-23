// Package services 定义语义分析的领域服务
package services

import (
	"fmt"
	"path/filepath"
	"strings"

	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
)

// PackageManager 包管理器领域服务
// 职责：管理包的加载、缓存和路径解析
type PackageManager struct {
	projectRoot  string                    // 项目根目录
	packageCache map[string]*PackageInfo   // 包缓存
	internalPaths map[string]bool          // internal 路径集合（用于快速检查）
}

// PackageInfo 包信息
type PackageInfo struct {
	name         string
	path         string
	isInternal   bool
	exports      map[string]*ExportSymbol  // 导出的符号
	dependencies []string                   // 依赖的包
}

// ExportSymbol 导出符号
type ExportSymbol struct {
	name       string
	symbolType string  // function, struct, enum, trait, etc.
	visibility sharedVO.Visibility
}

// NewPackageManager 创建新的包管理器
func NewPackageManager(projectRoot string) *PackageManager {
	return &PackageManager{
		projectRoot:   projectRoot,
		packageCache:  make(map[string]*PackageInfo),
		internalPaths: make(map[string]bool),
	}
}

// LoadPackage 加载包信息
func (pm *PackageManager) LoadPackage(packagePath string) (*PackageInfo, error) {
	// 检查缓存
	if pkg, ok := pm.packageCache[packagePath]; ok {
		return pkg, nil
	}

	// 检查是否是 internal 包
	isInternal := pm.isInternalPath(packagePath)
	if isInternal {
		pm.internalPaths[packagePath] = true
	}

	// 解析包路径
	fullPath := pm.resolvePackagePath(packagePath)
	if fullPath == "" {
		return nil, fmt.Errorf("package not found: %s", packagePath)
	}

	// 创建包信息（实际解析包文件需要后续实现）
	pkg := &PackageInfo{
		name:         filepath.Base(fullPath),
		path:         fullPath,
		isInternal:  isInternal,
		exports:     make(map[string]*ExportSymbol),
		dependencies: make([]string, 0),
	}

	// TODO: 解析包文件，提取导出符号
	// 这里需要调用解析器来解析包文件，提取所有公开的符号

	pm.packageCache[packagePath] = pkg
	return pkg, nil
}

// isInternalPath 检查是否是 internal 路径
func (pm *PackageManager) isInternalPath(packagePath string) bool {
	parts := strings.Split(packagePath, "/")
	for _, part := range parts {
		if part == "internal" {
			return true
		}
	}
	return false
}

// resolvePackagePath 解析包路径
func (pm *PackageManager) resolvePackagePath(packagePath string) string {
	// 处理相对路径
	if strings.HasPrefix(packagePath, "./") || strings.HasPrefix(packagePath, "../") {
		// 相对路径解析逻辑
		// 这里需要知道当前文件的路径，暂时返回空
		// TODO: 需要传入当前文件路径来解析相对路径
		return ""
	}

	// 处理绝对路径（从项目根目录）
	fullPath := filepath.Join(pm.projectRoot, "src", packagePath)
	if pm.pathExists(fullPath) {
		return fullPath
	}

	// 检查是否是包文件（.eo 文件）
	eoPath := fullPath + ".eo"
	if pm.pathExists(eoPath) {
		return eoPath
	}

	// TODO: 检查依赖包
	// 从 echo.toml 中读取依赖，查找依赖包

	return ""
}

// pathExists 检查路径是否存在
func (pm *PackageManager) pathExists(path string) bool {
	// TODO: 实现路径检查
	// 这里需要使用文件系统接口来检查
	// 暂时返回 false，后续实现
	return false
}

// ValidateImport 验证导入是否合法
func (pm *PackageManager) ValidateImport(importPath string, fromPackage string) error {
	// 检查是否是 internal 包
	if pm.isInternalPath(importPath) {
		// 检查导入者是否在同一项目内
		if !pm.isSameProject(importPath, fromPackage) {
			return fmt.Errorf("cannot import internal package from external project: %s", importPath)
		}
	}

	// 检查包是否存在
	_, err := pm.LoadPackage(importPath)
	if err != nil {
		return err
	}

	return nil
}

// isSameProject 检查是否在同一项目内
func (pm *PackageManager) isSameProject(packagePath1, packagePath2 string) bool {
	// 简化实现：如果两个路径都在项目根目录下，则认为在同一项目内
	// TODO: 更精确的实现需要检查项目边界
	return strings.HasPrefix(packagePath1, pm.projectRoot) &&
		strings.HasPrefix(packagePath2, pm.projectRoot)
}

// GetPackageInfo 获取包信息（从缓存）
func (pm *PackageManager) GetPackageInfo(packagePath string) (*PackageInfo, bool) {
	pkg, ok := pm.packageCache[packagePath]
	return pkg, ok
}

// ClearCache 清空缓存
func (pm *PackageManager) ClearCache() {
	pm.packageCache = make(map[string]*PackageInfo)
	pm.internalPaths = make(map[string]bool)
}

