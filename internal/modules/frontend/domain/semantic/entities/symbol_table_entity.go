// Package entities 定义语义分析的实体
package entities

import (
	"fmt"
	"time"

	"echo/internal/modules/frontend/domain/shared/value_objects"
)

// SymbolTableEntity 聚合根，表示符号表及其管理状态
type SymbolTableEntity struct {
	id             string
	symbolTable    *value_objects.SymbolTable
	scopeStack     []*value_objects.Scope
	isValid        bool
	validatedAt    *time.Time
	lastModifiedAt time.Time
}

// NewSymbolTableEntity 创建新的符号表实体
func NewSymbolTableEntity(id string) *SymbolTableEntity {
	symbolTable := value_objects.NewSymbolTable()
	return &SymbolTableEntity{
		id:             id,
		symbolTable:    symbolTable,
		scopeStack:     []*value_objects.Scope{symbolTable.GlobalScope()},
		isValid:        true, // 初始状态为有效
		lastModifiedAt: time.Now(),
	}
}

// ID 获取符号表实体的唯一标识
func (ste *SymbolTableEntity) ID() string {
	return ste.id
}

// SymbolTable 获取底层的符号表值对象
func (ste *SymbolTableEntity) SymbolTable() *value_objects.SymbolTable {
	return ste.symbolTable
}

// IsValid 检查符号表是否有效
func (ste *SymbolTableEntity) IsValid() bool {
	return ste.isValid
}

// ValidatedAt 获取验证时间
func (ste *SymbolTableEntity) ValidatedAt() *time.Time {
	return ste.validatedAt
}

// LastModifiedAt 获取最后修改时间
func (ste *SymbolTableEntity) LastModifiedAt() time.Time {
	return ste.lastModifiedAt
}

// CurrentScope 获取当前作用域
func (ste *SymbolTableEntity) CurrentScope() *value_objects.Scope {
	if len(ste.scopeStack) == 0 {
		return nil
	}
	return ste.scopeStack[len(ste.scopeStack)-1]
}

// EnterScope 进入新的作用域
func (ste *SymbolTableEntity) EnterScope(name string, location value_objects.SourceLocation) error {
	// 使用SymbolTable的EnterScope方法，它会自动管理作用域栈
	enteredScope := ste.symbolTable.EnterScope(name, location)
	ste.scopeStack = append(ste.scopeStack, enteredScope)
	ste.lastModifiedAt = time.Now()
	return nil
}

// ExitScope 退出当前作用域
func (ste *SymbolTableEntity) ExitScope() error {
	if len(ste.scopeStack) <= 1 {
		return fmt.Errorf("cannot exit global scope")
	}

	ste.symbolTable.ExitScope()
	ste.scopeStack = ste.scopeStack[:len(ste.scopeStack)-1]
	ste.lastModifiedAt = time.Now()
	return nil
}

// DefineSymbol 定义符号
func (ste *SymbolTableEntity) DefineSymbol(name string, kind value_objects.SymbolKind, symbolType value_objects.SymbolType, location value_objects.SourceLocation) error {
	symbol := value_objects.NewSymbol(name, kind, symbolType, ste.CurrentScope(), location)
	err := ste.symbolTable.AddSymbol(symbol)
	if err != nil {
		ste.MarkInvalid()
		return err
	}
	ste.lastModifiedAt = time.Now()
	return nil
}

// ResolveSymbol 解析符号
func (ste *SymbolTableEntity) ResolveSymbol(name string) (*value_objects.Symbol, bool) {
	return ste.symbolTable.LookupSymbol(name, ste.CurrentScope())
}

// MarkValid 标记符号表为有效
func (ste *SymbolTableEntity) MarkValid() {
	now := time.Now()
	ste.isValid = true
	ste.validatedAt = &now
}

// MarkInvalid 标记符号表为无效
func (ste *SymbolTableEntity) MarkInvalid() {
	now := time.Now()
	ste.isValid = false
	ste.validatedAt = &now
}

// ValidateScopes 验证作用域的一致性
func (ste *SymbolTableEntity) ValidateScopes() error {
	// 检查作用域栈的基本一致性
	if len(ste.scopeStack) == 0 {
		ste.MarkInvalid()
		return fmt.Errorf("scope stack is empty")
	}

	// 检查全局作用域
	if ste.scopeStack[0] != ste.symbolTable.GlobalScope() {
		ste.MarkInvalid()
		return fmt.Errorf("global scope mismatch")
	}

	// 检查当前作用域是否在栈中
	currentFound := false
	for _, scope := range ste.scopeStack {
		if scope == ste.symbolTable.CurrentScope() {
			currentFound = true
			break
		}
	}

	if !currentFound {
		ste.MarkInvalid()
		return fmt.Errorf("current scope not found in scope stack")
	}

	ste.MarkValid()
	return nil
}

// GetAllSymbols 获取所有符号的快照
func (ste *SymbolTableEntity) GetAllSymbols() []*value_objects.Symbol {
	// 简化的实现：返回当前作用域中的符号
	// 实际实现应该遍历所有作用域
	allSymbols := ste.CurrentScope().AllSymbols()
	symbols := make([]*value_objects.Symbol, 0, len(allSymbols))
	for _, symbol := range allSymbols {
		symbols = append(symbols, symbol)
	}
	return symbols
}

// String 返回SymbolTableEntity的字符串表示
func (ste *SymbolTableEntity) String() string {
	status := "Invalid"
	if ste.isValid {
		status = "Valid"
	}
	return fmt.Sprintf("SymbolTableEntity{ID: %s, Status: %s, Scopes: %d, LastModified: %s}",
		ste.id, status, len(ste.scopeStack), ste.lastModifiedAt.Format(time.RFC3339))
}
