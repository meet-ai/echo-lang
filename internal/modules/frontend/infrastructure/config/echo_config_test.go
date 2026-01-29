// Package config 实现配置文件解析测试
package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadConfig_Default 测试默认配置加载
func TestLoadConfig_Default(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 测试：配置文件不存在时返回默认配置
	config, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() 错误 = %v", err)
	}

	if config == nil {
		t.Fatal("LoadConfig() 返回 nil")
	}

	// 检查默认值
	if config.Name != "untitled" {
		t.Errorf("默认名称不匹配: 期望 'untitled', 得到 %q", config.Name)
	}

	if config.Version != "0.1.0" {
		t.Errorf("默认版本不匹配: 期望 '0.1.0', 得到 %q", config.Version)
	}

	if config.Package.Entry != "src/main.eo" {
		t.Errorf("默认入口不匹配: 期望 'src/main.eo', 得到 %q", config.Package.Entry)
	}
}

// TestGetProjectRoot 测试项目根目录查找
func TestGetProjectRoot(t *testing.T) {
	// 创建临时目录结构
	tmpDir := t.TempDir()
	projectRoot := filepath.Join(tmpDir, "myproject")
	os.MkdirAll(projectRoot, 0755)

	// 创建 echo.toml 文件
	configFile := filepath.Join(projectRoot, "echo.toml")
	os.WriteFile(configFile, []byte("# test config"), 0644)

	// 测试：从项目根目录查找
	root, err := GetProjectRoot(projectRoot)
	if err != nil {
		t.Fatalf("GetProjectRoot() 错误 = %v", err)
	}

	if root != projectRoot {
		t.Errorf("GetProjectRoot() = %q, 期望 %q", root, projectRoot)
	}

	// 测试：从子目录查找
	subDir := filepath.Join(projectRoot, "src", "main")
	os.MkdirAll(subDir, 0755)

	root, err = GetProjectRoot(subDir)
	if err != nil {
		t.Fatalf("GetProjectRoot() 从子目录查找错误 = %v", err)
	}

	if root != projectRoot {
		t.Errorf("GetProjectRoot() 从子目录 = %q, 期望 %q", root, projectRoot)
	}

	// 测试：找不到配置文件
	otherDir := t.TempDir()
	_, err = GetProjectRoot(otherDir)
	if err == nil {
		t.Error("期望错误，但没有错误（找不到配置文件）")
	}
}

// TestLoadConfig_WithDependencies 测试带依赖的配置加载
func TestLoadConfig_WithDependencies(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "echo.toml")

	// 创建测试配置文件
	tomlContent := `name = "test-project"
version = "1.0.0"
description = "Test project"

[package]
entry = "src/main.eo"
module = "test-module"

[dependencies]
"mathlib" = "2.1.0"
"network/http" = { version = "1.0.0", path = "vendor/http" }
"utils" = { version = "0.5.0" }
`

	err := os.WriteFile(configFile, []byte(tomlContent), 0644)
	if err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	// 加载配置
	config, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() 错误 = %v", err)
	}

	// 检查基本配置
	if config.Name != "test-project" {
		t.Errorf("名称不匹配: 期望 'test-project', 得到 %q", config.Name)
	}
	if config.Version != "1.0.0" {
		t.Errorf("版本不匹配: 期望 '1.0.0', 得到 %q", config.Version)
	}

	// 检查依赖
	if !config.HasDependency("mathlib") {
		t.Error("应该包含 mathlib 依赖")
	}
	if !config.HasDependency("network/http") {
		t.Error("应该包含 network/http 依赖")
	}
	if !config.HasDependency("utils") {
		t.Error("应该包含 utils 依赖")
	}

	// 检查依赖版本
	if version := config.GetDependencyVersion("mathlib"); version != "2.1.0" {
		t.Errorf("mathlib 版本不匹配: 期望 '2.1.0', 得到 %q", version)
	}
	if version := config.GetDependencyVersion("network/http"); version != "1.0.0" {
		t.Errorf("network/http 版本不匹配: 期望 '1.0.0', 得到 %q", version)
	}

	// 检查依赖路径
	if path := config.GetDependencyPath("network/http"); path != "vendor/http" {
		t.Errorf("network/http 路径不匹配: 期望 'vendor/http', 得到 %q", path)
	}
	// mathlib 使用默认路径
	if path := config.GetDependencyPath("mathlib"); path != filepath.Join("vendor", "mathlib") {
		t.Errorf("mathlib 路径不匹配: 期望 'vendor/mathlib', 得到 %q", path)
	}
}

// TestEchoConfig_DependencyMethods 测试依赖相关方法
func TestEchoConfig_DependencyMethods(t *testing.T) {
	config := &EchoConfig{
		Dependencies: map[string]interface{}{
			"simple": "1.0.0",
			"with-path": map[string]interface{}{
				"version": "2.0.0",
				"path":    "custom/path",
			},
			"with-version-only": map[string]interface{}{
				"version": "3.0.0",
			},
		},
	}

	// 测试 HasDependency
	if !config.HasDependency("simple") {
		t.Error("应该包含 simple 依赖")
	}
	if config.HasDependency("nonexistent") {
		t.Error("不应该包含 nonexistent 依赖")
	}

	// 测试 GetDependencyVersion
	if version := config.GetDependencyVersion("simple"); version != "1.0.0" {
		t.Errorf("simple 版本不匹配: 期望 '1.0.0', 得到 %q", version)
	}
	if version := config.GetDependencyVersion("with-path"); version != "2.0.0" {
		t.Errorf("with-path 版本不匹配: 期望 '2.0.0', 得到 %q", version)
	}

	// 测试 GetDependencyPath
	if path := config.GetDependencyPath("with-path"); path != "custom/path" {
		t.Errorf("with-path 路径不匹配: 期望 'custom/path', 得到 %q", path)
	}
	if path := config.GetDependencyPath("simple"); path != filepath.Join("vendor", "simple") {
		t.Errorf("simple 路径不匹配: 期望 'vendor/simple', 得到 %q", path)
	}
}

