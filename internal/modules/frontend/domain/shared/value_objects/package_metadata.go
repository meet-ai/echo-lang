// Package value_objects 定义包元数据相关值对象
package value_objects

// PackageMetadata 包元数据
type PackageMetadata struct {
	name         string
	version      string
	path         string
	exports      []ExportInfo
	dependencies []DependencyInfo
}

// NewPackageMetadata 创建新的包元数据
func NewPackageMetadata(name, version, path string) *PackageMetadata {
	return &PackageMetadata{
		name:         name,
		version:      version,
		path:         path,
		exports:     make([]ExportInfo, 0),
		dependencies: make([]DependencyInfo, 0),
	}
}

// Name 获取包名
func (pm *PackageMetadata) Name() string {
	return pm.name
}

// Version 获取版本
func (pm *PackageMetadata) Version() string {
	return pm.version
}

// Path 获取路径
func (pm *PackageMetadata) Path() string {
	return pm.path
}

// Exports 获取导出信息列表
func (pm *PackageMetadata) Exports() []ExportInfo {
	return pm.exports
}

// Dependencies 获取依赖信息列表
func (pm *PackageMetadata) Dependencies() []DependencyInfo {
	return pm.dependencies
}

// AddExport 添加导出信息
func (pm *PackageMetadata) AddExport(export ExportInfo) {
	pm.exports = append(pm.exports, export)
}

// AddDependency 添加依赖信息
func (pm *PackageMetadata) AddDependency(dep DependencyInfo) {
	pm.dependencies = append(pm.dependencies, dep)
}

// ExportInfo 导出信息
type ExportInfo struct {
	name       string
	symbolType string  // function, struct, enum, trait, etc.
	visibility Visibility
}

// NewExportInfo 创建新的导出信息
func NewExportInfo(name, symbolType string, visibility Visibility) ExportInfo {
	return ExportInfo{
		name:       name,
		symbolType: symbolType,
		visibility: visibility,
	}
}

// Name 获取符号名
func (ei ExportInfo) Name() string {
	return ei.name
}

// SymbolType 获取符号类型
func (ei ExportInfo) SymbolType() string {
	return ei.symbolType
}

// Visibility 获取可见性
func (ei ExportInfo) Visibility() Visibility {
	return ei.visibility
}

// DependencyInfo 依赖信息
type DependencyInfo struct {
	name    string
	version string
	path    string
}

// NewDependencyInfo 创建新的依赖信息
func NewDependencyInfo(name, version, path string) DependencyInfo {
	return DependencyInfo{
		name:    name,
		version: version,
		path:    path,
	}
}

// Name 获取依赖名
func (di DependencyInfo) Name() string {
	return di.name
}

// Version 获取版本
func (di DependencyInfo) Version() string {
	return di.version
}

// Path 获取路径
func (di DependencyInfo) Path() string {
	return di.path
}

