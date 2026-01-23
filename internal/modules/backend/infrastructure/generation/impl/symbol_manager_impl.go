package impl

import (
	"fmt"

	"echo/internal/modules/backend/domain/services/generation"
)

// SymbolManagerImpl 符号管理器实现
type SymbolManagerImpl struct {
	scopes []map[string]*generation.SymbolInfo
}

// NewSymbolManagerImpl 创建符号管理器实现
func NewSymbolManagerImpl() *SymbolManagerImpl {
	return &SymbolManagerImpl{
		scopes: []map[string]*generation.SymbolInfo{
			make(map[string]*generation.SymbolInfo), // 全局作用域
		},
	}
}

// RegisterSymbol 注册符号
func (sm *SymbolManagerImpl) RegisterSymbol(name string, symbolType string, value interface{}) error {
	return sm.RegisterFunctionSymbol(name, symbolType, value, false)
}

// RegisterFunctionSymbol 注册函数符号
func (sm *SymbolManagerImpl) RegisterFunctionSymbol(name string, symbolType string, value interface{}, isAsync bool) error {
	currentScope := sm.scopes[len(sm.scopes)-1]
	if _, exists := currentScope[name]; exists {
		return fmt.Errorf("symbol '%s' already declared in current scope", name)
	}

	currentScope[name] = &generation.SymbolInfo{
		Name:    name,
		Type:    symbolType,
		Value:   value,
		IsAsync: isAsync,
	}
	return nil
}

// LookupSymbol 查找符号
func (sm *SymbolManagerImpl) LookupSymbol(name string) (*generation.SymbolInfo, error) {
	// 从当前作用域向外层作用域查找
	for i := len(sm.scopes) - 1; i >= 0; i-- {
		if symbol, exists := sm.scopes[i][name]; exists {
			return symbol, nil
		}
	}
	return nil, fmt.Errorf("symbol '%s' not found", name)
}

// SymbolExists 检查符号是否存在（在所有作用域中）
func (sm *SymbolManagerImpl) SymbolExists(name string) bool {
	for i := len(sm.scopes) - 1; i >= 0; i-- {
		if _, exists := sm.scopes[i][name]; exists {
			return true
		}
	}
	return false
}

// SymbolExistsInCurrentScope 检查符号是否在当前作用域存在
func (sm *SymbolManagerImpl) SymbolExistsInCurrentScope(name string) bool {
	if len(sm.scopes) == 0 {
		return false
	}
	currentScope := sm.scopes[len(sm.scopes)-1]
	_, exists := currentScope[name]
	return exists
}

// EnterScope 进入新的作用域
func (sm *SymbolManagerImpl) EnterScope() error {
	sm.scopes = append(sm.scopes, make(map[string]*generation.SymbolInfo))
	return nil
}

// ExitScope 退出当前作用域
func (sm *SymbolManagerImpl) ExitScope() error {
	if len(sm.scopes) <= 1 {
		return fmt.Errorf("cannot exit global scope")
	}
	sm.scopes = sm.scopes[:len(sm.scopes)-1]
	return nil
}

// GetCurrentScopeLevel 获取当前作用域层级
func (sm *SymbolManagerImpl) GetCurrentScopeLevel() int {
	return len(sm.scopes) - 1
}

// UpdateSymbolValue 更新符号的值
func (sm *SymbolManagerImpl) UpdateSymbolValue(name string, value interface{}) error {
	// 从当前作用域向外查找符号
	for i := len(sm.scopes) - 1; i >= 0; i-- {
		if symbol, exists := sm.scopes[i][name]; exists {
			symbol.Value = value
			return nil
		}
	}
	return fmt.Errorf("symbol %s not found", name)
}

// Clear 清理所有符号
func (sm *SymbolManagerImpl) Clear() error {
	sm.scopes = []map[string]*generation.SymbolInfo{
		make(map[string]*generation.SymbolInfo),
	}
	return nil
}
