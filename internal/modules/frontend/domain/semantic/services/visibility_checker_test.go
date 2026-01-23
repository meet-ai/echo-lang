// Package services 定义语义分析的领域服务测试
package services

import (
	"testing"

	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
)

// TestVisibilityChecker_CheckAccess 测试访问权限检查
func TestVisibilityChecker_CheckAccess(t *testing.T) {
	pm := NewPackageManager("/test/project")
	vc := NewVisibilityChecker(pm)

	tests := []struct {
		name            string
		symbolName      string
		symbolVisibility sharedVO.Visibility
		fromPackage     string
		targetPackage   string
		wantErr         bool
	}{
		{
			name:            "同包内访问私有符号",
			symbolName:      "helper",
			symbolVisibility: sharedVO.VisibilityPrivate,
			fromPackage:     "math",
			targetPackage:   "math",
			wantErr:         false, // 同包内可以访问
		},
		{
			name:            "跨包访问公开符号",
			symbolName:      "sqrt",
			symbolVisibility: sharedVO.VisibilityPublic,
			fromPackage:     "utils",
			targetPackage:   "math",
			wantErr:         false, // 可以访问
		},
		{
			name:            "跨包访问私有符号",
			symbolName:      "helper",
			symbolVisibility: sharedVO.VisibilityPrivate,
			fromPackage:     "utils",
			targetPackage:   "math",
			wantErr:         true, // 不能访问
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := vc.CheckAccess(tt.symbolName, tt.symbolVisibility, tt.fromPackage, tt.targetPackage)
			if tt.wantErr {
				if err == nil {
					t.Errorf("期望错误，但没有错误")
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
				}
			}
		})
	}
}

// TestVisibilityChecker_CheckSymbolVisibility 测试符号可见性检查
func TestVisibilityChecker_CheckSymbolVisibility(t *testing.T) {
	pm := NewPackageManager("/test/project")
	vc := NewVisibilityChecker(pm)

	tests := []struct {
		name            string
		symbolName      string
		symbolVisibility sharedVO.Visibility
		currentPackage  string
		accessingPackage string
		want            bool
	}{
		{
			name:            "同包内访问私有符号",
			symbolName:      "helper",
			symbolVisibility: sharedVO.VisibilityPrivate,
			currentPackage:  "math",
			accessingPackage: "math",
			want:            true,
		},
		{
			name:            "跨包访问公开符号",
			symbolName:      "sqrt",
			symbolVisibility: sharedVO.VisibilityPublic,
			currentPackage:  "math",
			accessingPackage: "utils",
			want:            true,
		},
		{
			name:            "跨包访问私有符号",
			symbolName:      "helper",
			symbolVisibility: sharedVO.VisibilityPrivate,
			currentPackage:  "math",
			accessingPackage: "utils",
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := vc.CheckSymbolVisibility(tt.symbolName, tt.symbolVisibility, tt.currentPackage, tt.accessingPackage)
			if got != tt.want {
				t.Errorf("CheckSymbolVisibility() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

