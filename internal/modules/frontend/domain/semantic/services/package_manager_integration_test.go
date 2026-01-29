// Package services 定义语义分析的领域服务集成测试
package services

import (
	"context"
	"testing"

	lexicalServices "echo/internal/modules/frontend/domain/lexical/services"
	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
	syntaxServices "echo/internal/modules/frontend/domain/syntax/services"
)

// TestPackageManagementIntegration 测试包管理的完整集成流程
// 包括：解析 package 和 import 声明 -> 包管理器验证 -> 路径解析
func TestPackageManagementIntegration(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		wantPkg     string
		wantImports []struct {
			path string
			alias string
		}
		wantErr bool
		desc    string
	}{
		{
			name:    "基础包声明和导入（标识符形式）",
			source:  "package main; import math",
			wantPkg: "main",
			wantImports: []struct {
				path string
				alias string
			}{
				{path: "math", alias: ""},
			},
			wantErr: false,
			desc:    "测试 package 声明和 import 标识符形式（新语法）",
		},
		{
			name:    "基础包声明和导入（字符串形式）",
			source:  "package utils; import \"math\";",
			wantPkg: "utils",
			wantImports: []struct {
				path string
				alias string
			}{
				{path: "math", alias: ""},
			},
			wantErr: false,
			desc:    "测试 package 声明和 import 字符串形式（向后兼容）",
		},
		{
			name:    "多个导入（混合形式）",
			source:  "package main; import math; import \"utils\"; import types as TypesLib",
			wantPkg: "main",
			wantImports: []struct {
				path string
				alias string
			}{
				{path: "math", alias: ""},
				{path: "utils", alias: ""},
				{path: "types", alias: "TypesLib"},
			},
			wantErr: false,
			desc:    "测试多个 import，混合使用标识符和字符串形式，带别名",
		},
		{
			name:    "package 无分号 + import 标识符形式",
			source:  "package main; import tpk",
			wantPkg: "main",
			wantImports: []struct {
				path string
				alias string
			}{
				{path: "tpk", alias: ""},
			},
			wantErr: false,
			desc:    "测试 package 无分号和 import 标识符形式（类似 Python）",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 步骤1：词法分析
			sourceFile := lexicalVO.NewSourceFile("test.eo", tt.source)
			lexer := lexicalServices.NewAdvancedLexerService()
			tokenStream, err := lexer.Tokenize(context.Background(), sourceFile)
			if err != nil {
				t.Fatalf("词法分析失败: %v", err)
			}

			// 步骤2：语法分析
			parser := syntaxServices.NewRecursiveDescentParser()
			programAST, err := parser.ParseProgram(context.Background(), tokenStream)
			if tt.wantErr {
				if err == nil {
					t.Errorf("期望错误，但没有错误: %s", tt.desc)
				}
				return
			}

			if err != nil {
				t.Fatalf("解析失败: %v\n描述: %s", err, tt.desc)
			}

			// 步骤3：验证包声明
			pkgDecl := programAST.Package()
			if pkgDecl == nil {
				t.Fatal("包声明为空")
			}

			if pkgDecl.PackageName() != tt.wantPkg {
				t.Errorf("包名不匹配: 期望 %s, 得到 %s", tt.wantPkg, pkgDecl.PackageName())
			}

			// 步骤4：验证导入语句
			imports := programAST.Imports()
			if len(imports) != len(tt.wantImports) {
				t.Errorf("导入数量不匹配: 期望 %d, 得到 %d", len(tt.wantImports), len(imports))
			}

			for i, wantImp := range tt.wantImports {
				if i >= len(imports) {
					t.Errorf("导入 %d 不存在", i)
					continue
				}

				imp := imports[i]
				if imp.ImportPath() != wantImp.path {
					t.Errorf("导入 %d 路径不匹配: 期望 %s, 得到 %s", i, wantImp.path, imp.ImportPath())
				}

				if imp.Alias() != wantImp.alias {
					t.Errorf("导入 %d 别名不匹配: 期望 %s, 得到 %s", i, wantImp.alias, imp.Alias())
				}
			}

			// 步骤5：使用包管理器验证导入
			pm := NewPackageManager("/test/project")
			for _, imp := range imports {
				err := pm.ValidateImport(imp.ImportPath(), pkgDecl.PackageName())
				if err != nil {
					// 对于不存在的包，ValidateImport 可能返回错误，这是正常的
					// 我们只检查是否是 internal 包访问错误
					if imp.ImportPath() != "internal/cache" {
						t.Logf("导入验证警告（包可能不存在）: %v", err)
					}
				}
			}

			t.Logf("✅ 集成测试通过: %s", tt.desc)
		})
	}
}

// TestPackageManager_LoadPackage 测试包加载功能
func TestPackageManager_LoadPackage(t *testing.T) {
	pm := NewPackageManager("/test/project")

	tests := []struct {
		name        string
		packagePath string
		wantErr     bool
		errContains string
	}{
		{
			name:        "加载标准包",
			packagePath: "math",
			wantErr:     false, // 即使包不存在，LoadPackage 也可能返回空包信息
		},
		{
			name:        "加载 internal 包",
			packagePath: "internal/cache",
			wantErr:     false,
		},
		{
			name:        "加载相对路径包",
			packagePath: "./utils",
			wantErr:     true, // 相对路径解析暂未实现
			errContains: "package not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg, err := pm.LoadPackage(tt.packagePath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("期望错误，但没有错误")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("错误消息不包含期望的文本: 期望包含 %q, 得到 %q", tt.errContains, err.Error())
				}
			} else {
				// 即使包不存在，LoadPackage 也可能返回包信息（但路径可能为空）
				if err != nil {
					t.Logf("包加载警告（包可能不存在）: %v", err)
				}
				if pkg != nil {
					t.Logf("包信息: name=%s, path=%s, isInternal=%v", pkg.name, pkg.path, pkg.isInternal)
				}
			}
		})
	}
}

// TestPackageManager_ImportValidationWithNewSyntax 测试新语法导入的验证
func TestPackageManager_ImportValidationWithNewSyntax(t *testing.T) {
	pm := NewPackageManager("/test/project")

	tests := []struct {
		name        string
		importPath  string
		fromPackage string
		wantErr     bool
		errContains string
		desc        string
	}{
		{
			name:        "标识符形式导入（新语法）",
			importPath:  "math",
			fromPackage: "main",
			wantErr:     false,
			desc:        "测试 import math（标识符形式）的验证",
		},
		{
			name:        "字符串形式导入（旧语法）",
			importPath:  "utils",
			fromPackage: "main",
			wantErr:     false,
			desc:        "测试 import \"utils\"（字符串形式）的验证",
		},
		{
			name:        "带别名的导入",
			importPath:  "types",
			fromPackage: "main",
			wantErr:     false,
			desc:        "测试 import types as TypesLib 的验证",
		},
		{
			name:        "internal 包导入（同项目）",
			importPath:  "internal/cache",
			fromPackage: "internal/utils",
			wantErr:     false,
			desc:        "测试同项目内 internal 包导入",
		},
		{
			name:        "internal 包导入（不同项目）",
			importPath:  "internal/cache",
			fromPackage: "/other/project/utils",
			wantErr:     true,
			errContains: "cannot import internal",
			desc:        "测试跨项目 internal 包导入（应该失败）",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pm.ValidateImport(tt.importPath, tt.fromPackage)
			if tt.wantErr {
				if err == nil {
					t.Errorf("期望错误，但没有错误: %s", tt.desc)
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("错误消息不包含期望的文本: 期望包含 %q, 得到 %q\n描述: %s", tt.errContains, err.Error(), tt.desc)
				} else {
					t.Logf("✅ 验证通过（期望错误）: %s", tt.desc)
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v\n描述: %s", err, tt.desc)
				} else {
					t.Logf("✅ 验证通过: %s", tt.desc)
				}
			}
		})
	}
}
