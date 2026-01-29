// Package services 定义语义分析的领域服务测试
package services

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPackageManager_isInternalPath 测试 internal 路径检测
func TestPackageManager_isInternalPath(t *testing.T) {
	pm := NewPackageManager("/test/project")

	tests := []struct {
		name     string
		path     string
		want     bool
	}{
		{
			name: "internal 路径",
			path: "internal/cache",
			want: true,
		},
		{
			name: "嵌套 internal 路径",
			path: "src/internal/utils",
			want: true,
		},
		{
			name: "非 internal 路径",
			path: "math/geometry",
			want: false,
		},
		{
			name: "包含 internal 但不是路径部分",
			path: "internalized",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pm.isInternalPath(tt.path)
			if got != tt.want {
				t.Errorf("isInternalPath(%q) = %v, 期望 %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestPackageManager_ValidateImport 测试导入验证
func TestPackageManager_ValidateImport(t *testing.T) {
	pm := NewPackageManager("/test/project")

	tests := []struct {
		name        string
		importPath  string
		fromPackage string
		wantErr     bool
		errContains string
	}{
		{
			name:        "正常导入",
			importPath:  "math",
			fromPackage: "utils",
			wantErr:     false,
		},
		{
			name:        "internal 包导入（同项目）",
			importPath:  "internal/cache",
			fromPackage: "internal/utils",
			wantErr:     false, // 同项目内允许
		},
		{
			name:        "internal 包导入（不同项目）",
			importPath:  "internal/cache",
			fromPackage: "/other/project/utils",
			wantErr:     true,
			errContains: "cannot import internal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pm.ValidateImport(tt.importPath, tt.fromPackage)
			if tt.wantErr {
				if err == nil {
					t.Errorf("期望错误，但没有错误")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("错误消息不包含期望的文本: 期望包含 %q, 得到 %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
				}
			}
		})
	}
}

// TestPackageManager_isSameProject 测试同项目检测
func TestPackageManager_isSameProject(t *testing.T) {
	pm := NewPackageManager("/test/project")

	tests := []struct {
		name     string
		path1    string
		path2    string
		want     bool
	}{
		{
			name:  "同项目路径",
			path1: "/test/project/src/math",
			path2: "/test/project/src/utils",
			want:  true,
		},
		{
			name:  "不同项目路径",
			path1: "/test/project/src/math",
			path2: "/other/project/src/utils",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pm.isSameProject(tt.path1, tt.path2)
			if got != tt.want {
				t.Errorf("isSameProject(%q, %q) = %v, 期望 %v", tt.path1, tt.path2, got, tt.want)
			}
		})
	}
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
		s[len(s)-len(substr):] == substr || 
		containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestPackageManager_ResolveDependencyPackage 测试依赖包路径解析
func TestPackageManager_ResolveDependencyPackage(t *testing.T) {
	// 创建临时项目目录
	tmpDir := t.TempDir()
	projectRoot := filepath.Join(tmpDir, "test-project")
	os.MkdirAll(projectRoot, 0755)

	// 创建 echo.toml 配置文件
	configFile := filepath.Join(projectRoot, "echo.toml")
	tomlContent := `name = "test-project"
version = "1.0.0"

[package]
entry = "src/main.eo"
module = "test-module"

[dependencies]
"mathlib" = "2.1.0"
"network/http" = { version = "1.0.0", path = "vendor/http" }
`

	err := os.WriteFile(configFile, []byte(tomlContent), 0644)
	if err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	// 创建依赖包目录结构
	// 1. mathlib 使用默认路径 vendor/mathlib
	mathlibDir := filepath.Join(projectRoot, "vendor", "mathlib")
	os.MkdirAll(mathlibDir, 0755)
	mathlibFile := filepath.Join(mathlibDir, "mathlib.eo")
	os.WriteFile(mathlibFile, []byte("package mathlib"), 0644)

	// 2. network/http 使用配置的路径 vendor/http
	httpDir := filepath.Join(projectRoot, "vendor", "http")
	os.MkdirAll(httpDir, 0755)
	httpFile := filepath.Join(httpDir, "http.eo")
	os.WriteFile(httpFile, []byte("package http"), 0644)

	// 创建包管理器
	pm := NewPackageManager(projectRoot)

	// 测试：解析 mathlib 依赖包（使用默认路径）
	mathlibPath := pm.resolvePackagePath("mathlib")
	if mathlibPath == "" {
		t.Error("应该能解析 mathlib 依赖包路径")
	}
	if mathlibPath != mathlibFile {
		t.Errorf("mathlib 路径不匹配: 期望 %q, 得到 %q", mathlibFile, mathlibPath)
	}

	// 测试：解析 network/http 依赖包（使用配置路径）
	httpPath := pm.resolvePackagePath("network/http")
	if httpPath == "" {
		t.Error("应该能解析 network/http 依赖包路径")
	}
	if httpPath != httpFile {
		t.Errorf("network/http 路径不匹配: 期望 %q, 得到 %q", httpFile, httpPath)
	}

	// 测试：不存在的依赖包
	nonexistentPath := pm.resolvePackagePath("nonexistent")
	if nonexistentPath != "" {
		t.Errorf("不存在的依赖包应该返回空路径，得到 %q", nonexistentPath)
	}
}

// TestPackageManager_LoadDependencyPackage 测试加载依赖包
func TestPackageManager_LoadDependencyPackage(t *testing.T) {
	// 创建临时项目目录
	tmpDir := t.TempDir()
	projectRoot := filepath.Join(tmpDir, "test-project")
	os.MkdirAll(projectRoot, 0755)

	// 创建 echo.toml 配置文件
	configFile := filepath.Join(projectRoot, "echo.toml")
	tomlContent := `name = "test-project"
version = "1.0.0"

[package]
entry = "src/main.eo"
module = "test-module"

[dependencies]
"utils" = "1.0.0"
`

	err := os.WriteFile(configFile, []byte(tomlContent), 0644)
	if err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	// 创建依赖包
	utilsDir := filepath.Join(projectRoot, "vendor", "utils")
	os.MkdirAll(utilsDir, 0755)
	utilsFile := filepath.Join(utilsDir, "utils.eo")
	os.WriteFile(utilsFile, []byte("package utils"), 0644)

	// 创建包管理器
	pm := NewPackageManager(projectRoot)

	// 测试：加载依赖包
	pkg, err := pm.LoadPackage("utils")
	if err != nil {
		t.Fatalf("LoadPackage() 错误 = %v", err)
	}

	if pkg == nil {
		t.Fatal("LoadPackage() 返回 nil")
	}

	if pkg.path != utilsFile {
		t.Errorf("包路径不匹配: 期望 %q, 得到 %q", utilsFile, pkg.path)
	}

	if pkg.isInternal {
		t.Error("依赖包不应该是 internal 包")
	}

	// 测试：从缓存加载
	pkg2, err := pm.LoadPackage("utils")
	if err != nil {
		t.Fatalf("从缓存加载包错误 = %v", err)
	}

	if pkg2 != pkg {
		t.Error("应该从缓存返回同一个包对象")
	}
}

