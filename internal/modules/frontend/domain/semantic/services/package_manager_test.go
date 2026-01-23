// Package services 定义语义分析的领域服务测试
package services

import (
	"testing"

	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
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

