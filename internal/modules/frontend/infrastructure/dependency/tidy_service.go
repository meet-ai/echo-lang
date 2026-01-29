// Package dependency 实现依赖包管理功能
package dependency

import (
	"fmt"
	"os"

	"echo/internal/modules/frontend/domain/semantic/services"
	"echo/internal/modules/frontend/infrastructure/config"
)

// TidyResult tidy 操作的结果
type TidyResult struct {
	AddedDependencies   []string // 新添加的依赖
	ExistingDependencies []string // 已存在的依赖
	InternalPackages   []string // 项目内包（未添加）
	SkippedPackages    []string // 跳过的包（已配置）
}

// TidyService tidy 服务
// 职责：整合扫描、分类、更新功能，执行完整的 tidy 操作
type TidyService struct {
	projectRoot string
	scanner     *DependencyScanner
	classifier  *services.PackageClassifier
	updater     *config.ConfigUpdater
}

// NewTidyService 创建 tidy 服务
func NewTidyService(projectRoot string) (*TidyService, error) {
	// 加载配置
	cfg, err := config.LoadConfig(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 创建配置更新器
	updater, err := config.NewConfigUpdater(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create config updater: %w", err)
	}

	// 创建扫描器
	scanner := NewDependencyScanner(projectRoot)

	// 创建分类器
	classifier := services.NewPackageClassifier(projectRoot, cfg)

	return &TidyService{
		projectRoot: projectRoot,
		scanner:     scanner,
		classifier:  classifier,
		updater:     updater,
	}, nil
}

// Tidy 执行 tidy 操作
// 流程：
// 1. 扫描项目代码，提取所有 import 语句
// 2. 分类包（项目内包 vs 外部依赖）
// 3. 过滤出需要添加到 echo.toml 的依赖
// 4. 更新 echo.toml
// 5. 返回操作结果
func (s *TidyService) Tidy() (*TidyResult, error) {
	result := &TidyResult{
		AddedDependencies:   make([]string, 0),
		ExistingDependencies: make([]string, 0),
		InternalPackages:    make([]string, 0),
		SkippedPackages:     make([]string, 0),
	}

	// 步骤1：扫描项目代码，提取所有 import 语句
	fmt.Fprintf(os.Stdout, "Scanning project files for imports...\n")
	importPaths, err := s.scanner.ScanImports()
	if err != nil {
		return nil, fmt.Errorf("failed to scan imports: %w", err)
	}

	if len(importPaths) == 0 {
		fmt.Fprintf(os.Stdout, "No imports found in project.\n")
		return result, nil
	}

	fmt.Fprintf(os.Stdout, "Found %d unique import(s).\n", len(importPaths))

	// 步骤2：分类包
	fmt.Fprintf(os.Stdout, "Classifying packages...\n")
	_ = s.classifier.ClassifyPackages(importPaths) // 分类包（用于后续判断）

	// 步骤3：过滤出需要添加的依赖
	externalDeps := s.classifier.GetExternalDependencies(importPaths)
	internalPackages := s.classifier.GetInternalPackages(importPaths)

	result.InternalPackages = internalPackages

	// 区分新依赖和已存在的依赖
	for _, path := range importPaths {
		classification := s.classifier.ClassifyPackage(path)
		if classification.IsConfigured {
			result.ExistingDependencies = append(result.ExistingDependencies, path)
		} else if classification.Type == services.PackageTypeExternal {
			// 检查是否在 externalDeps 中
			for _, dep := range externalDeps {
				if dep == path {
					result.AddedDependencies = append(result.AddedDependencies, path)
					break
				}
			}
		}
	}

	// 步骤4：更新 echo.toml
	if len(result.AddedDependencies) > 0 {
		fmt.Fprintf(os.Stdout, "Adding %d new dependency(ies) to echo.toml...\n", len(result.AddedDependencies))
		if err := s.updater.AddDependencies(result.AddedDependencies); err != nil {
			return nil, fmt.Errorf("failed to add dependencies: %w", err)
		}

		if err := s.updater.Save(); err != nil {
			return nil, fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Fprintf(os.Stdout, "Successfully updated echo.toml\n")
	} else {
		fmt.Fprintf(os.Stdout, "No new dependencies to add.\n")
	}

	return result, nil
}

// PrintResult 打印 tidy 结果
func (s *TidyService) PrintResult(result *TidyResult) {
	fmt.Fprintf(os.Stdout, "\n=== Tidy Result ===\n")

	if len(result.AddedDependencies) > 0 {
		fmt.Fprintf(os.Stdout, "\nAdded dependencies:\n")
		for _, dep := range result.AddedDependencies {
			fmt.Fprintf(os.Stdout, "  + %s\n", dep)
		}
	}

	if len(result.ExistingDependencies) > 0 {
		fmt.Fprintf(os.Stdout, "\nExisting dependencies (already in echo.toml):\n")
		for _, dep := range result.ExistingDependencies {
			fmt.Fprintf(os.Stdout, "  = %s\n", dep)
		}
	}

	if len(result.InternalPackages) > 0 {
		fmt.Fprintf(os.Stdout, "\nInternal packages (in src/, not added):\n")
		for _, pkg := range result.InternalPackages {
			fmt.Fprintf(os.Stdout, "  - %s\n", pkg)
		}
	}

	fmt.Fprintf(os.Stdout, "\n")
}
