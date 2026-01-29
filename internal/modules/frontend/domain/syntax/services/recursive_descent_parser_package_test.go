// Package services 定义语法分析上下文的领域服务测试
package services

import (
	"context"
	"testing"

	lexicalServices "echo/internal/modules/frontend/domain/lexical/services"
	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
)

// TestParsePackageDeclaration 测试包声明解析
func TestParsePackageDeclaration(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantPkg string
		wantErr bool
	}{
		{
			name:    "简单包声明",
			source:  "package math;",
			wantPkg: "math",
			wantErr: false,
		},
		{
			name:    "包声明无分号",
			source:  "package utils",
			wantPkg: "utils",
			wantErr: false,
		},
		{
			name:    "多级包名",
			source:  "package math.geometry;",
			wantPkg: "math.geometry",
			wantErr: false,
		},
		{
			name:    "缺少包名",
			source:  "package;",
			wantPkg: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建词法分析器
			sourceFile := lexicalVO.NewSourceFile("test.eo", tt.source)
			lexer := lexicalServices.NewAdvancedLexerService()
			tokenStream, err := lexer.Tokenize(context.Background(), sourceFile)
			if err != nil {
				t.Fatalf("词法分析失败: %v", err)
			}

			// 创建解析器
			parser := NewRecursiveDescentParser()
			programAST, err := parser.ParseProgram(context.Background(), tokenStream)
			if tt.wantErr {
				if err == nil {
					t.Errorf("期望错误，但没有错误")
				}
				return
			}

			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}

			// 检查包声明
			pkgDecl := programAST.Package()
			if pkgDecl == nil {
				t.Fatal("包声明为空")
			}

			if pkgDecl.PackageName() != tt.wantPkg {
				t.Errorf("包名不匹配: 期望 %s, 得到 %s", tt.wantPkg, pkgDecl.PackageName())
			}
		})
	}
}

// TestParseFromImportStatement 测试 from ... import 语法解析
func TestParseFromImportStatement(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		wantPath  string
		wantElems []string
		wantErr   bool
	}{
		{
			name:      "单个元素导入",
			source:    "package test; from \"math\" import sqrt;",
			wantPath:  "math",
			wantElems: []string{"sqrt"},
			wantErr:   false,
		},
		{
			name:      "多个元素导入",
			source:    "package test; from \"math\" import sin, cos, tan;",
			wantPath:  "math",
			wantElems: []string{"sin", "cos", "tan"},
			wantErr:   false,
		},
		{
			name:      "带别名的元素导入",
			source:    "package test; from \"math\" import sqrt as squareRoot;",
			wantPath:  "math",
			wantElems: []string{"sqrt"},
			wantErr:   false,
		},
		{
			name:      "相对路径导入",
			source:    "package test; from \"./utils\" import helper;",
			wantPath:  "./utils",
			wantElems: []string{"helper"},
			wantErr:   false,
		},
		{
			name:    "缺少 import 关键字",
			source:  "package test; from \"math\" sqrt;",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建词法分析器
			sourceFile := lexicalVO.NewSourceFile("test.eo", tt.source)
			lexer := lexicalServices.NewAdvancedLexerService()
			tokenStream, err := lexer.Tokenize(context.Background(), sourceFile)
			if err != nil {
				t.Fatalf("词法分析失败: %v", err)
			}

			// 创建解析器
			parser := NewRecursiveDescentParser()
			programAST, err := parser.ParseProgram(context.Background(), tokenStream)
			if tt.wantErr {
				if err == nil {
					t.Errorf("期望错误，但没有错误")
				}
				return
			}

			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}

			// 检查导入语句
			imports := programAST.Imports()
			if len(imports) == 0 {
				t.Fatal("没有导入语句")
			}

			importStmt := imports[0]
			if importStmt.ImportPath() != tt.wantPath {
				t.Errorf("导入路径不匹配: 期望 %s, 得到 %s", tt.wantPath, importStmt.ImportPath())
			}

			if importStmt.ImportType() != sharedVO.ImportTypeElements {
				t.Errorf("导入类型不匹配: 期望 ImportTypeElements, 得到 %v", importStmt.ImportType())
			}

			elements := importStmt.Elements()
			if len(elements) != len(tt.wantElems) {
				t.Errorf("元素数量不匹配: 期望 %d, 得到 %d", len(tt.wantElems), len(elements))
			}

			for i, elem := range elements {
				if elem.Name() != tt.wantElems[i] {
					t.Errorf("元素 %d 不匹配: 期望 %s, 得到 %s", i, tt.wantElems[i], elem.Name())
				}
			}
		})
	}
}

// TestParsePrivateFunction 测试 private 关键字解析
func TestParsePrivateFunction(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		wantPrivate  bool
		wantFuncName string
		wantErr      bool
	}{
		{
			name:         "私有函数",
			source:       "package test; private func helper() {}",
			wantPrivate:  true,
			wantFuncName: "helper",
			wantErr:      false,
		},
		{
			name:         "公开函数（默认）",
			source:       "package test; func publicFunc() {}",
			wantPrivate:  false,
			wantFuncName: "publicFunc",
			wantErr:      false,
		},
		{
			name:         "私有异步函数",
			source:       "package test; private async func asyncHelper() {}",
			wantPrivate:  true,
			wantFuncName: "asyncHelper",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建词法分析器
			sourceFile := lexicalVO.NewSourceFile("test.eo", tt.source)
			lexer := lexicalServices.NewAdvancedLexerService()
			tokenStream, err := lexer.Tokenize(context.Background(), sourceFile)
			if err != nil {
				t.Fatalf("词法分析失败: %v", err)
			}

			// 创建解析器
			parser := NewRecursiveDescentParser()
			programAST, err := parser.ParseProgram(context.Background(), tokenStream)
			if tt.wantErr {
				if err == nil {
					t.Errorf("期望错误，但没有错误")
				}
				return
			}

			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}

			// 检查函数声明
			nodes := programAST.Nodes()
			if len(nodes) == 0 {
				t.Fatal("没有解析到节点")
			}

			// 查找函数声明节点（支持普通函数和异步函数）
			var funcDecl *sharedVO.FunctionDeclaration
			for _, node := range nodes {
				if fd, ok := node.(*sharedVO.FunctionDeclaration); ok {
					funcDecl = fd
					break
				}
				// 检查是否是异步函数声明
				if afd, ok := node.(*sharedVO.AsyncFunctionDeclaration); ok {
					funcDecl = afd.FunctionDeclaration()
					break
				}
			}

			if funcDecl == nil {
				t.Fatal("没有找到函数声明")
			}

			if funcDecl.Name() != tt.wantFuncName {
				t.Errorf("函数名不匹配: 期望 %s, 得到 %s", tt.wantFuncName, funcDecl.Name())
			}

			if funcDecl.Visibility().IsPrivate() != tt.wantPrivate {
				t.Errorf("可见性不匹配: 期望 private=%v, 得到 %v", tt.wantPrivate, funcDecl.Visibility().IsPrivate())
			}
		})
	}
}

// TestParsePackageImportStatement 测试包级导入解析
func TestParsePackageImportStatement(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantPath string
		wantAlias string
		wantErr  bool
	}{
		{
			name:     "简单导入（字符串形式）",
			source:   "package test; import \"math\";",
			wantPath: "math",
			wantAlias: "",
			wantErr:  false,
		},
		{
			name:     "简单导入（标识符形式，类似 Python）",
			source:   "package test; import math;",
			wantPath: "math",
			wantAlias: "",
			wantErr:  false,
		},
		{
			name:     "简单导入（标识符形式，无分号）",
			source:   "package test; import math",
			wantPath: "math",
			wantAlias: "",
			wantErr:  false,
		},
		{
			name:     "带别名的导入（字符串形式）",
			source:   "package test; import \"math\" as Math;",
			wantPath: "math",
			wantAlias: "Math",
			wantErr:  false,
		},
		{
			name:     "带别名的导入（标识符形式）",
			source:   "package test; import math as Math;",
			wantPath: "math",
			wantAlias: "Math",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建词法分析器
			sourceFile := lexicalVO.NewSourceFile("test.eo", tt.source)
			lexer := lexicalServices.NewAdvancedLexerService()
			tokenStream, err := lexer.Tokenize(context.Background(), sourceFile)
			if err != nil {
				t.Fatalf("词法分析失败: %v", err)
			}

			// 创建解析器
			parser := NewRecursiveDescentParser()
			programAST, err := parser.ParseProgram(context.Background(), tokenStream)
			if tt.wantErr {
				if err == nil {
					t.Errorf("期望错误，但没有错误")
				}
				return
			}

			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}

			// 检查导入语句
			imports := programAST.Imports()
			if len(imports) == 0 {
				t.Fatal("没有导入语句")
			}

			importStmt := imports[0]
			if importStmt.ImportPath() != tt.wantPath {
				t.Errorf("导入路径不匹配: 期望 %s, 得到 %s", tt.wantPath, importStmt.ImportPath())
			}

			if importStmt.Alias() != tt.wantAlias {
				t.Errorf("别名不匹配: 期望 %s, 得到 %s", tt.wantAlias, importStmt.Alias())
			}

			if importStmt.ImportType() != sharedVO.ImportTypePackage {
				t.Errorf("导入类型不匹配: 期望 ImportTypePackage, 得到 %v", importStmt.ImportType())
			}
		})
	}
}

// TestNewSyntaxFeatures 测试新语法特性
func TestNewSyntaxFeatures(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		wantErr     bool
		desc        string
		wantImports int
	}{
		{
			name:        "package 无分号 + import 标识符形式",
			source:      "package main; import math",
			wantErr:     false,
			desc:        "测试 package 无分号和 import 标识符形式（类似 Python）",
			wantImports: 1,
		},
		{
			name:        "package 无分号 + import 字符串形式",
			source:      "package main; import \"math\";",
			wantErr:     false,
			desc:        "测试 package 无分号和 import 字符串形式（向后兼容）",
			wantImports: 1,
		},
		{
			name:        "package 有分号 + import 标识符形式",
			source:      "package main; import math",
			wantErr:     false,
			desc:        "测试 package 有分号和 import 标识符形式",
			wantImports: 1,
		},
		{
			name:        "多个 import（混合形式）",
			source:      "package main; import math; import \"utils\"; import types as TypesLib",
			wantErr:     false,
			desc:        "测试多个 import，混合使用标识符和字符串形式",
			wantImports: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建词法分析器
			sourceFile := lexicalVO.NewSourceFile("test.eo", tt.source)
			lexer := lexicalServices.NewAdvancedLexerService()
			tokenStream, err := lexer.Tokenize(context.Background(), sourceFile)
			if err != nil {
				t.Fatalf("词法分析失败: %v", err)
			}

			// 创建解析器
			parser := NewRecursiveDescentParser()
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

			// 验证包声明存在
			pkgDecl := programAST.Package()
			if pkgDecl == nil {
				t.Fatal("包声明为空")
			}

			// 验证导入语句
			imports := programAST.Imports()
			if len(imports) != tt.wantImports {
				t.Errorf("导入数量不匹配: 期望 %d, 得到 %d", tt.wantImports, len(imports))
			}

			// 验证每个导入语句
			for i, imp := range imports {
				if imp.ImportPath() == "" {
					t.Errorf("导入 %d 的路径为空", i)
				}
			}

			t.Logf("✅ 测试通过: %s", tt.desc)
		})
	}
}

