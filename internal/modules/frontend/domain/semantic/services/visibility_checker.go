// Package services 定义语义分析的领域服务
package services

import (
	"fmt"

	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
)

// VisibilityChecker 可见性检查器领域服务
// 职责：检查符号和导入的访问权限
type VisibilityChecker struct {
	packageManager *PackageManager
}

// NewVisibilityChecker 创建新的可见性检查器
func NewVisibilityChecker(packageManager *PackageManager) *VisibilityChecker {
	return &VisibilityChecker{
		packageManager: packageManager,
	}
}

// CheckAccess 检查访问权限
func (vc *VisibilityChecker) CheckAccess(
	symbolName string,
	symbolVisibility sharedVO.Visibility,
	fromPackage string,
	targetPackage string,
) error {
	// 同一包内，可以访问所有符号
	if fromPackage == targetPackage {
		return nil
	}

	// 不同包，只能访问公开符号
	if symbolVisibility == sharedVO.VisibilityPrivate {
		return fmt.Errorf("cannot access private symbol '%s' from package '%s'", symbolName, fromPackage)
	}

	return nil
}

// CheckImportAccess 检查导入访问权限
func (vc *VisibilityChecker) CheckImportAccess(
	importStmt *sharedVO.ImportStatement,
	fromPackage string,
) error {
	// 检查包级导入
	if importStmt.ImportType() == sharedVO.ImportTypePackage {
		return vc.packageManager.ValidateImport(importStmt.ImportPath(), fromPackage)
	}

	// 检查元素级导入
	if importStmt.ImportType() == sharedVO.ImportTypeElements {
		// 验证每个元素的访问权限
		pkg, err := vc.packageManager.LoadPackage(importStmt.ImportPath())
		if err != nil {
			return err
		}

		for _, element := range importStmt.Elements() {
			export, ok := pkg.exports[element.Name()]
			if !ok {
				return fmt.Errorf("symbol '%s' not found in package '%s'", element.Name(), importStmt.ImportPath())
			}

			if err := vc.CheckAccess(element.Name(), export.visibility, fromPackage, pkg.name); err != nil {
				return err
			}
		}
	}

	return nil
}

// CheckSymbolVisibility 检查符号可见性（用于语义分析）
func (vc *VisibilityChecker) CheckSymbolVisibility(
	symbolName string,
	symbolVisibility sharedVO.Visibility,
	currentPackage string,
	accessingPackage string,
) bool {
	// 同一包内，总是可以访问
	if currentPackage == accessingPackage {
		return true
	}

	// 不同包，只能访问公开符号
	return symbolVisibility == sharedVO.VisibilityPublic
}

