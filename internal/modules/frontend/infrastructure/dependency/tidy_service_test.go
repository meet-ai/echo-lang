package dependency

import (
	"os"
	"path/filepath"
	"testing"

	"echo/internal/modules/frontend/infrastructure/config"
)

func TestTidyService_Tidy(t *testing.T) {
	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "echo-tidy-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建项目结构
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create src dir: %v", err)
	}

	// 创建 echo.toml（初始只有 mathlib）
	configContent := `name = "test-project"
version = "1.0.0"

[package]
entry = "src/main.eo"
module = "test"

[dependencies]
"mathlib" = "2.1.0"
`
	configPath := filepath.Join(tempDir, "echo.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write echo.toml: %v", err)
	}

	// 创建测试源文件（使用 mathlib 和 network/http，但 network/http 不在 echo.toml 中）
	mainContent := `package main

import mathlib
import "network/http" as HttpLib
from "utils" import helper
`
	mainPath := filepath.Join(srcDir, "main.eo")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main.eo: %v", err)
	}

	// 创建项目内包 utils（在 src/ 下）
	utilsDir := filepath.Join(srcDir, "utils")
	if err := os.MkdirAll(utilsDir, 0755); err != nil {
		t.Fatalf("Failed to create utils dir: %v", err)
	}
	utilsFile := filepath.Join(utilsDir, "utils.eo")
	if err := os.WriteFile(utilsFile, []byte("package utils\n"), 0644); err != nil {
		t.Fatalf("Failed to write utils.eo: %v", err)
	}

	// 创建外部依赖 network/http（在 vendor/ 下，但不在 echo.toml 中）
	vendorDir := filepath.Join(tempDir, "vendor", "network", "http")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatalf("Failed to create vendor dir: %v", err)
	}
	httpFile := filepath.Join(vendorDir, "http.eo")
	if err := os.WriteFile(httpFile, []byte("package http\n"), 0644); err != nil {
		t.Fatalf("Failed to write http.eo: %v", err)
	}

	// 创建 TidyService
	tidyService, err := NewTidyService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create tidy service: %v", err)
	}

	// 执行 tidy
	result, err := tidyService.Tidy()
	if err != nil {
		t.Fatalf("Tidy failed: %v", err)
	}

	// 验证结果
	// network/http 应该被添加（外部依赖，未配置）
	foundNetworkHttp := false
	for _, dep := range result.AddedDependencies {
		if dep == "network/http" {
			foundNetworkHttp = true
			break
		}
	}
	if !foundNetworkHttp {
		t.Errorf("Expected network/http to be added, but it wasn't. Added: %v", result.AddedDependencies)
	}

	// utils 不应该被添加（项目内包）
	foundUtils := false
	for _, pkg := range result.InternalPackages {
		if pkg == "utils" {
			foundUtils = true
			break
		}
	}
	if !foundUtils {
		t.Errorf("Expected utils to be classified as internal package, but it wasn't. Internal: %v", result.InternalPackages)
	}

	// mathlib 应该在已存在依赖中
	foundMathlib := false
	for _, dep := range result.ExistingDependencies {
		if dep == "mathlib" {
			foundMathlib = true
			break
		}
	}
	if !foundMathlib {
		t.Errorf("Expected mathlib to be in existing dependencies, but it wasn't. Existing: %v", result.ExistingDependencies)
	}

	// 验证 echo.toml 已更新
	updatedConfig, err := config.LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("Failed to load updated config: %v", err)
	}

	// 检查 network/http 是否已添加到配置中
	if !updatedConfig.HasDependency("network/http") {
		t.Error("Expected network/http to be in echo.toml after tidy, but it wasn't")
	}

	// 检查 mathlib 仍然存在
	if !updatedConfig.HasDependency("mathlib") {
		t.Error("Expected mathlib to still be in echo.toml after tidy, but it wasn't")
	}
}

func TestDependencyScanner_ScanImports(t *testing.T) {
	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "echo-scanner-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试源文件
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create src dir: %v", err)
	}

	// 创建包含多种 import 语法的文件
	content := `package main

import mathlib
import "network/http" as HttpLib
from "utils" import helper, format
`
	filePath := filepath.Join(srcDir, "main.eo")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// 创建扫描器
	scanner := NewDependencyScanner(tempDir)

	// 扫描导入
	imports, err := scanner.ScanImports()
	if err != nil {
		t.Fatalf("ScanImports failed: %v", err)
	}

	// 验证结果
	expectedImports := map[string]bool{
		"mathlib":     true,
		"network/http": true,
		"utils":       true,
	}

	if len(imports) != len(expectedImports) {
		t.Errorf("Expected %d imports, got %d: %v", len(expectedImports), len(imports), imports)
	}

	for _, imp := range imports {
		if !expectedImports[imp] {
			t.Errorf("Unexpected import: %s", imp)
		}
	}
}
