// Package config 实现配置文件解析和更新
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// ConfigUpdater 配置更新器
// 职责：读取和更新 echo.toml 配置文件
type ConfigUpdater struct {
	projectRoot string
	config      *EchoConfig
}

// NewConfigUpdater 创建配置更新器
func NewConfigUpdater(projectRoot string) (*ConfigUpdater, error) {
	cfg, err := LoadConfig(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &ConfigUpdater{
		projectRoot: projectRoot,
		config:      cfg,
	}, nil
}

// AddDependency 添加依赖到配置
// 如果依赖已存在，不会覆盖现有配置
// 新依赖使用默认版本号 "1.0.0"
func (u *ConfigUpdater) AddDependency(packageName string) error {
	// 检查依赖是否已存在
	if u.config.HasDependency(packageName) {
		return nil // 已存在，不添加
	}

	// 添加新依赖（使用简单字符串格式，默认版本号）
	u.config.Dependencies[packageName] = "1.0.0"

	return nil
}

// AddDependencies 批量添加依赖
func (u *ConfigUpdater) AddDependencies(packageNames []string) error {
	for _, name := range packageNames {
		if err := u.AddDependency(name); err != nil {
			return fmt.Errorf("failed to add dependency %s: %w", name, err)
		}
	}
	return nil
}

// Save 保存配置到文件
func (u *ConfigUpdater) Save() error {
	configPath := filepath.Join(u.projectRoot, "echo.toml")

	// 如果配置文件不存在，创建它
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 确保目录存在
		if err := os.MkdirAll(u.projectRoot, 0755); err != nil {
			return fmt.Errorf("failed to create project directory: %w", err)
		}
	}

	// 将配置转换为 TOML 格式
	tomlData, err := u.marshalConfig()
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(configPath, tomlData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// marshalConfig 将配置转换为 TOML 格式
// 注意：需要保持现有格式和注释（如果可能）
func (u *ConfigUpdater) marshalConfig() ([]byte, error) {
	// 使用 toml.Marshal 序列化配置
	// 注意：这会丢失注释，但保持结构
	data, err := toml.Marshal(u.config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config to TOML: %w", err)
	}

	return data, nil
}

// GetConfig 获取当前配置（只读）
func (u *ConfigUpdater) GetConfig() *EchoConfig {
	return u.config
}

// FormatDependencyKey 格式化依赖键名（用于 TOML 输出）
// 如果包名包含特殊字符（如 /），需要用引号包裹
func FormatDependencyKey(packageName string) string {
	// 如果包名包含特殊字符，需要用引号包裹
	if strings.Contains(packageName, "/") || strings.Contains(packageName, "-") {
		return fmt.Sprintf(`"%s"`, packageName)
	}
	return packageName
}
