package entities

import (
	"fmt"
	"time"
)

// SymbolTable 符号表 - 语义分析的核心数据结构
type SymbolTable struct {
	// 符号表作用域层次结构
	globalScope  *Scope
	currentScope *Scope

	// 符号计数
	symbolCount int

	// 创建时间
	createdAt time.Time
}

// NewSymbolTable 创建新的符号表
func NewSymbolTable() *SymbolTable {
	globalScope := NewScope("global", nil)
	return &SymbolTable{
		globalScope:  globalScope,
		currentScope: globalScope,
		symbolCount:  0,
		createdAt:    time.Now(),
	}
}

// EnterScope 进入新的作用域
func (st *SymbolTable) EnterScope(name string) *Scope {
	newScope := NewScope(name, st.currentScope)
	st.currentScope = newScope
	return newScope
}

// ExitScope 退出当前作用域
func (st *SymbolTable) ExitScope() error {
	if st.currentScope.Parent() == nil {
		return fmt.Errorf("cannot exit global scope")
	}
	st.currentScope = st.currentScope.Parent()
	return nil
}

// DefineSymbol 定义符号
func (st *SymbolTable) DefineSymbol(name string, symbolType SymbolType, dataType string) (*Symbol, error) {
	// 检查当前作用域是否已定义
	if existing := st.currentScope.LookupLocal(name); existing != nil {
		return nil, fmt.Errorf("symbol '%s' already defined in current scope", name)
	}

	symbol := NewSymbol(name, symbolType, dataType, st.currentScope)
	st.currentScope.AddSymbol(symbol)
	st.symbolCount++

	return symbol, nil
}

// LookupSymbol 查找符号（从当前作用域向上查找）
func (st *SymbolTable) LookupSymbol(name string) *Symbol {
	return st.currentScope.Lookup(name)
}

// LookupGlobalSymbol 查找全局符号
func (st *SymbolTable) LookupGlobalSymbol(name string) *Symbol {
	return st.globalScope.LookupLocal(name)
}

// CurrentScope 返回当前作用域
func (st *SymbolTable) CurrentScope() *Scope {
	return st.currentScope
}

// GlobalScope 返回全局作用域
func (st *SymbolTable) GlobalScope() *Scope {
	return st.globalScope
}

// SymbolCount 返回符号总数
func (st *SymbolTable) SymbolCount() int {
	return st.symbolCount
}

// ValidateType 检查类型兼容性
func (st *SymbolTable) ValidateType(symbolName string, expectedType string) error {
	symbol := st.LookupSymbol(symbolName)
	if symbol == nil {
		return fmt.Errorf("undefined symbol: %s", symbolName)
	}

	if symbol.DataType() != expectedType {
		return fmt.Errorf("type mismatch: expected %s, got %s", expectedType, symbol.DataType())
	}

	return nil
}

// Scope 作用域
type Scope struct {
	name     string
	parent   *Scope
	symbols  map[string]*Symbol
	children []*Scope
}

// NewScope 创建新作用域
func NewScope(name string, parent *Scope) *Scope {
	return &Scope{
		name:     name,
		parent:   parent,
		symbols:  make(map[string]*Symbol),
		children: make([]*Scope, 0),
	}
}

// Name 返回作用域名
func (s *Scope) Name() string {
	return s.name
}

// Parent 返回父作用域
func (s *Scope) Parent() *Scope {
	return s.parent
}

// AddSymbol 添加符号到作用域
func (s *Scope) AddSymbol(symbol *Symbol) {
	s.symbols[symbol.Name()] = symbol
}

// LookupLocal 在当前作用域查找符号
func (s *Scope) LookupLocal(name string) *Symbol {
	return s.symbols[name]
}

// Lookup 从当前作用域向上查找符号
func (s *Scope) Lookup(name string) *Symbol {
	// 先在当前作用域查找
	if symbol := s.LookupLocal(name); symbol != nil {
		return symbol
	}

	// 在父作用域中递归查找
	if s.parent != nil {
		return s.parent.Lookup(name)
	}

	return nil
}

// Symbols 返回当前作用域的所有符号
func (s *Scope) Symbols() map[string]*Symbol {
	// Return a copy to prevent external modification
	result := make(map[string]*Symbol)
	for k, v := range s.symbols {
		result[k] = v
	}
	return result
}

// Symbol 符号
type Symbol struct {
	name       string
	symbolType SymbolType
	dataType   string
	scope      *Scope
	definedAt  time.Time
}

// NewSymbol 创建新符号
func NewSymbol(name string, symbolType SymbolType, dataType string, scope *Scope) *Symbol {
	return &Symbol{
		name:       name,
		symbolType: symbolType,
		dataType:   dataType,
		scope:      scope,
		definedAt:  time.Now(),
	}
}

// Name 返回符号名
func (s *Symbol) Name() string {
	return s.name
}

// SymbolType 返回符号类型
func (s *Symbol) SymbolType() SymbolType {
	return s.symbolType
}

// DataType 返回数据类型
func (s *Symbol) DataType() string {
	return s.dataType
}

// Scope 返回符号定义的作用域
func (s *Symbol) Scope() *Scope {
	return s.scope
}

// DefinedAt 返回符号定义时间
func (s *Symbol) DefinedAt() time.Time {
	return s.definedAt
}

// IsGlobal 检查是否为全局符号
func (s *Symbol) IsGlobal() bool {
	return s.scope.Name() == "global"
}

// SymbolType 符号类型
type SymbolType string

const (
	SymbolTypeVariable  SymbolType = "variable"
	SymbolTypeFunction  SymbolType = "function"
	SymbolTypeParameter SymbolType = "parameter"
	SymbolTypeType      SymbolType = "type"
)

// String 返回符号类型的字符串表示
func (st SymbolType) String() string {
	return string(st)
}

// IsValid 检查符号类型是否有效
func (st SymbolType) IsValid() bool {
	switch st {
	case SymbolTypeVariable, SymbolTypeFunction, SymbolTypeParameter, SymbolTypeType:
		return true
	default:
		return false
	}
}

// SymbolTableIterator 符号表迭代器
type SymbolTableIterator struct {
	currentScope *Scope
	symbolIndex  int
	symbols      []*Symbol
}

// NewSymbolTableIterator 创建符号表迭代器
func NewSymbolTableIterator(table *SymbolTable) *SymbolTableIterator {
	iterator := &SymbolTableIterator{
		currentScope: table.globalScope,
		symbolIndex:  -1,
		symbols:      make([]*Symbol, 0),
	}
	iterator.collectSymbols(table.globalScope)
	return iterator
}

// collectSymbols 递归收集作用域中的符号
func (it *SymbolTableIterator) collectSymbols(scope *Scope) {
	// 添加当前作用域的符号
	for _, symbol := range scope.symbols {
		it.symbols = append(it.symbols, symbol)
	}

	// 递归处理子作用域
	for _, child := range scope.children {
		it.collectSymbols(child)
	}
}

// Next 返回下一个符号
func (it *SymbolTableIterator) Next() *Symbol {
	it.symbolIndex++
	if it.symbolIndex >= len(it.symbols) {
		return nil
	}
	return it.symbols[it.symbolIndex]
}

// HasNext 检查是否还有下一个符号
func (it *SymbolTableIterator) HasNext() bool {
	return it.symbolIndex+1 < len(it.symbols)
}

// Reset 重置迭代器
func (it *SymbolTableIterator) Reset() {
	it.symbolIndex = -1
}
