// Package services 定义语义分析的领域服务独立测试
// 此文件独立测试包管理器功能，不依赖其他有问题的代码
package services

import (
	"testing"
)

// TestPackageManager_Standalone 独立测试包管理器核心功能
// 不依赖 semantic_analyzer 等其他可能有问题的代码
func TestPackageManager_Standalone(t *testing.T) {
	pm := NewPackageManager("/test/project")

	t.Run("isInternalPath 测试", func(t *testing.T) {
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
	})

	t.Run("isSameProject 测试", func(t *testing.T) {
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
	})

	t.Run("ValidateImport 测试", func(t *testing.T) {
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
				wantErr:     true, // 因为包不存在，LoadPackage 会返回错误
				errContains: "package not found",
			},
			{
				name:        "internal 包导入（同项目）",
				importPath:  "/test/project/internal/cache", // 使用绝对路径，确保在同一项目内
				fromPackage: "/test/project/internal/utils", // 使用绝对路径，确保在同一项目内
				wantErr:     true, // 因为包不存在
				errContains: "package not found",
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
					} else if tt.errContains != "" && !containsStandalone(err.Error(), tt.errContains) {
						t.Errorf("错误消息不包含期望的文本: 期望包含 %q, 得到 %q", tt.errContains, err.Error())
					} else {
						t.Logf("✅ 验证通过（期望错误）: %v", err)
					}
				} else {
					if err != nil {
						t.Errorf("不期望错误，但得到: %v", err)
					}
				}
			})
		}
	})

	t.Run("包缓存测试", func(t *testing.T) {
		pm := NewPackageManager("/test/project")

		// 测试缓存功能
		_, ok := pm.GetPackageInfo("math")
		if ok {
			t.Error("缓存应该为空")
		}

		// 清空缓存
		pm.ClearCache()

		_, ok = pm.GetPackageInfo("math")
		if ok {
			t.Error("清空后缓存应该为空")
		}
	})
}

// containsStandalone 检查字符串是否包含子串（独立版本，避免与 package_manager_test.go 中的重复）
func containsStandalone(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
		s[len(s)-len(substr):] == substr || 
		containsMiddleStandalone(s, substr))))
}

func containsMiddleStandalone(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
