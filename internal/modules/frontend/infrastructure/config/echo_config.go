// Package config 实现配置文件解析
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// EchoConfig Echo 项目配置
type EchoConfig struct {
	Name         string                 `toml:"name"`
	Version      string                 `toml:"version"`
	Description  string                 `toml:"description"`
	Authors      []string               `toml:"authors"`
	Package      PackageConfig          `toml:"package"`
	Registry     *RegistryConfig        `toml:"registry"`     // T-DEV-024：包注册表地址
	Dependencies map[string]interface{} `toml:"dependencies"`
}

// RegistryConfig 注册表配置（T-DEV-024）
type RegistryConfig struct {
	URL string `toml:"url"` // 如 https://registry.echo-lang.org
}

// PackageConfig 包配置
type PackageConfig struct {
	Entry  string `toml:"entry"`
	Module string `toml:"module"`
}

// DependencyInfo 依赖包信息
// 支持两种格式：
// 1. 简单字符串：`"mathlib" = "2.1.0"`
// 2. 详细配置：`"mathlib" = { version = "2.1.0", path = "vendor/mathlib" }`
type DependencyInfo struct {
	Version string `toml:"version"` // 版本号
	Path    string `toml:"path"`    // 本地路径（可选）
	Source  string `toml:"source"`  // 来源（git、local等，可选）
}

// GetDependencyPath 获取依赖包的路径
// 如果依赖配置是字符串，则返回默认路径 vendor/{packageName}
// 如果依赖配置是对象，则返回 path 字段或默认路径
func (c *EchoConfig) GetDependencyPath(packageName string) string {
	dep := c.Dependencies[packageName]
	if dep == nil {
		return ""
	}

	// 如果依赖是对象，尝试解析 path 字段
	if depMap, ok := dep.(map[string]interface{}); ok {
		if path, ok := depMap["path"].(string); ok && path != "" {
			return path
		}
	}

	// 默认路径：vendor/{packageName}（适用于字符串格式和没有 path 的对象格式）
	return filepath.Join("vendor", packageName)
}

// GetDependencyVersion 获取依赖包的版本号
func (c *EchoConfig) GetDependencyVersion(packageName string) string {
	dep := c.Dependencies[packageName]
	if dep == nil {
		return ""
	}

	// 如果依赖是字符串，直接返回
	if version, ok := dep.(string); ok {
		return version
	}

	// 如果依赖是对象，尝试获取 version 字段
	if depMap, ok := dep.(map[string]interface{}); ok {
		if version, ok := depMap["version"].(string); ok {
			return version
		}
	}

	return ""
}

// HasDependency 检查是否有指定的依赖
func (c *EchoConfig) HasDependency(packageName string) bool {
	_, ok := c.Dependencies[packageName]
	return ok
}

// GetDependencySource 获取依赖包的来源（git、local等）
func (c *EchoConfig) GetDependencySource(packageName string) string {
	dep := c.Dependencies[packageName]
	if dep == nil {
		return ""
	}

	// 如果依赖是对象，尝试获取 source 字段
	if depMap, ok := dep.(map[string]interface{}); ok {
		if source, ok := depMap["source"].(string); ok {
			return source
		}
	}

	// 默认返回空字符串（表示本地依赖）
	return ""
}

// GetDependencyGitURL 获取依赖包的 Git URL
func (c *EchoConfig) GetDependencyGitURL(packageName string) string {
	dep := c.Dependencies[packageName]
	if dep == nil {
		return ""
	}

	// 如果依赖是对象，尝试获取 git 字段
	if depMap, ok := dep.(map[string]interface{}); ok {
		if git, ok := depMap["git"].(string); ok {
			return git
		}
	}

	return ""
}

// GetDependencyGitRef 获取依赖包的 Git 引用（tag、branch、commit）
// 返回格式：tag=v1.0.0 或 branch=main 或 commit=abc123
func (c *EchoConfig) GetDependencyGitRef(packageName string) (refType string, refValue string) {
	dep := c.Dependencies[packageName]
	if dep == nil {
		return "", ""
	}

	// 如果依赖是对象，尝试获取 tag、branch 或 commit 字段
	if depMap, ok := dep.(map[string]interface{}); ok {
		// 优先级：tag > branch > commit
		if tag, ok := depMap["tag"].(string); ok && tag != "" {
			return "tag", tag
		}
		if branch, ok := depMap["branch"].(string); ok && branch != "" {
			return "branch", branch
		}
		if commit, ok := depMap["commit"].(string); ok && commit != "" {
			return "commit", commit
		}
	}

	return "", ""
}

// GetRegistryURL 获取注册表 URL（T-DEV-024）；未配置 [registry] 或 url 为空时返回空字符串
func (c *EchoConfig) GetRegistryURL() string {
	if c.Registry == nil {
		return ""
	}
	return c.Registry.URL
}

// GetDependencyStoragePath 获取依赖包的存储路径（用于自动下载）
// 格式：vendor/{packageName}@{version}/
// 包名中的 / 会被替换为 -
func (c *EchoConfig) GetDependencyStoragePath(packageName string) string {
	version := c.GetDependencyVersion(packageName)
	return GetDependencyStoragePathForVersion(packageName, version)
}

// GetDependencyStoragePathForVersion 按给定版本返回依赖存储路径（供 echo.lock 锁定版本使用）
func GetDependencyStoragePathForVersion(packageName, version string) string {
	if version == "" {
		version = "latest"
	}
	safePackageName := strings.ReplaceAll(packageName, "/", "-")
	return filepath.Join("vendor", fmt.Sprintf("%s@%s", safePackageName, version))
}

// LoadConfig 加载配置文件
// 注意：这里使用简单的字符串解析，后续可以集成 toml 库
func LoadConfig(projectRoot string) (*EchoConfig, error) {
	configPath := filepath.Join(projectRoot, "echo.toml")

	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 如果配置文件不存在，返回默认配置
		return &EchoConfig{
			Name:        "untitled",
			Version:     "0.1.0",
			Description: "",
			Authors:     []string{},
			Package: PackageConfig{
				Entry:  "src/main.eo",
				Module: "untitled",
			},
			Dependencies: make(map[string]interface{}),
		}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 使用 TOML 库解析配置文件
	var config EchoConfig
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 设置默认值
	if config.Name == "" {
		config.Name = "untitled"
	}
	if config.Version == "" {
		config.Version = "0.1.0"
	}
	if config.Package.Entry == "" {
		config.Package.Entry = "src/main.eo"
	}
	if config.Package.Module == "" {
		config.Package.Module = "untitled"
	}
	if config.Dependencies == nil {
		config.Dependencies = make(map[string]interface{})
	}
	if config.Authors == nil {
		config.Authors = []string{}
	}

	return &config, nil
}

// GetProjectRoot 获取项目根目录
// 从当前工作目录向上查找包含 echo.toml 的目录
func GetProjectRoot(startDir string) (string, error) {
	current := startDir
	for {
		configPath := filepath.Join(current, "echo.toml")
		if _, err := os.Stat(configPath); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			// 已经到达根目录
			break
		}
		current = parent
	}

	return "", fmt.Errorf("project root not found (no echo.toml found)")
}

