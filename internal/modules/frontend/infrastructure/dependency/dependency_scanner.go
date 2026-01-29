// Package dependency 实现依赖包管理功能
package dependency

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DependencyScanner 依赖扫描器
// 职责：扫描项目代码，提取所有 import 语句中的包路径
type DependencyScanner struct {
	projectRoot string
}

// NewDependencyScanner 创建依赖扫描器
func NewDependencyScanner(projectRoot string) *DependencyScanner {
	return &DependencyScanner{
		projectRoot: projectRoot,
	}
}

// ScanImports 扫描项目目录下的所有 .eo 文件，提取所有导入的包路径
// 返回：去重后的包路径列表
func (s *DependencyScanner) ScanImports() ([]string, error) {
	importPaths := make(map[string]bool)

	// 扫描项目目录下的所有 .eo 文件
	err := filepath.Walk(s.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过隐藏目录和文件
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 跳过 vendor 目录（避免扫描依赖包）
		if info.IsDir() && info.Name() == "vendor" {
			return filepath.SkipDir
		}

		// 只处理 .eo 文件
		if !info.IsDir() && strings.HasSuffix(path, ".eo") {
			imports, err := s.extractImportsFromFile(path)
			if err != nil {
				// 记录错误但继续扫描其他文件
				fmt.Fprintf(os.Stderr, "Warning: failed to extract imports from %s: %v\n", path, err)
				return nil
			}

			for _, imp := range imports {
				// 过滤掉相对路径导入（以 . 或 .. 开头）
				if !strings.HasPrefix(imp, ".") {
					importPaths[imp] = true
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan project: %w", err)
	}

	// 转换为切片并返回
	result := make([]string, 0, len(importPaths))
	for path := range importPaths {
		result = append(result, path)
	}

	return result, nil
}

// extractImportsFromFile 从单个文件中提取所有 import 语句的包路径
// 支持以下语法：
// 1. import "package"
// 2. import package
// 3. from "package" import element1, element2
// 4. from package import element1, element2
func (s *DependencyScanner) extractImportsFromFile(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)
	imports := make([]string, 0)

	// 正则表达式匹配 import 语句
	// 匹配：import "package" 或 import package
	importPattern := regexp.MustCompile(`(?m)^\s*import\s+(?:"([^"]+)"|([a-zA-Z_][a-zA-Z0-9_/]*))`)
	matches := importPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			// 匹配组1：字符串形式的包路径
			if match[1] != "" {
				imports = append(imports, match[1])
			}
			// 匹配组2：标识符形式的包路径
			if len(match) > 2 && match[2] != "" {
				imports = append(imports, match[2])
			}
		}
	}

	// 正则表达式匹配 from ... import 语句
	// 匹配：from "package" import ... 或 from package import ...
	fromImportPattern := regexp.MustCompile(`(?m)^\s*from\s+(?:"([^"]+)"|([a-zA-Z_][a-zA-Z0-9_/]*))\s+import`)
	fromMatches := fromImportPattern.FindAllStringSubmatch(content, -1)
	for _, match := range fromMatches {
		if len(match) > 1 {
			// 匹配组1：字符串形式的包路径
			if match[1] != "" {
				imports = append(imports, match[1])
			}
			// 匹配组2：标识符形式的包路径
			if len(match) > 2 && match[2] != "" {
				imports = append(imports, match[2])
			}
		}
	}

	return imports, nil
}
