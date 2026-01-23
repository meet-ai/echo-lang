// Package config 实现配置文件解析
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// EchoConfig Echo 项目配置
type EchoConfig struct {
	Name        string
	Version     string
	Description string
	Authors     []string
	Package     PackageConfig
	Dependencies map[string]interface{}
}

// PackageConfig 包配置
type PackageConfig struct {
	Entry  string
	Module string
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

	// TODO: 使用 toml 库解析配置文件
	// 暂时返回默认配置
	// 后续可以集成 gopkg.in/toml.v3 或类似库
	_ = data // 暂时忽略，避免未使用变量警告

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

