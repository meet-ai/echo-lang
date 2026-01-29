// Package value_objects 定义符号相关值对象
package value_objects

// Symbol 符号值对象
type Symbol struct {
	name      string
	symbolKind SymbolKind
	symbolType SymbolType
	scope     *Scope
	location  SourceLocation
}

// SymbolKind 符号种类
type SymbolKind string

const (
	SymbolKindVariable  SymbolKind = "variable"  // 变量
	SymbolKindFunction  SymbolKind = "function"  // 函数
	SymbolKindParameter SymbolKind = "parameter" // 参数
	SymbolKindStruct    SymbolKind = "struct"    // 结构体
	SymbolKindEnum      SymbolKind = "enum"      // 枚举
	SymbolKindTrait     SymbolKind = "trait"     // trait
	SymbolKindMethod    SymbolKind = "method"    // 方法
	SymbolKindType      SymbolKind = "type"      // 类型别名
)

// SymbolType 符号类型信息
type SymbolType struct {
	typeName    string
	isPrimitive bool
	isArray     bool
	elementType *SymbolType
}

// String 返回SymbolType的字符串表示
func (st SymbolType) String() string {
	if st.isArray && st.elementType != nil {
		return st.elementType.String() + "[]"
	}
	return st.typeName
}

// NewSymbol 创建新的符号
func NewSymbol(name string, kind SymbolKind, symbolType SymbolType, scope *Scope, location SourceLocation) *Symbol {
	return &Symbol{
		name:       name,
		symbolKind: kind,
		symbolType: symbolType,
		scope:      scope,
		location:   location,
	}
}

// Name 获取符号名称
func (s *Symbol) Name() string {
	return s.name
}

// Kind 获取符号种类
func (s *Symbol) Kind() SymbolKind {
	return s.symbolKind
}

// Type 获取符号类型
func (s *Symbol) Type() SymbolType {
	return s.symbolType
}

// Scope 获取符号作用域
func (s *Symbol) Scope() *Scope {
	return s.scope
}

// Location 获取符号位置
func (s *Symbol) Location() SourceLocation {
	return s.location
}

// IsGlobal 检查是否为全局符号
func (s *Symbol) IsGlobal() bool {
	return s.scope != nil && s.scope.IsGlobal()
}

// String 返回符号的字符串表示
func (s *Symbol) String() string {
	return string(s.symbolKind) + " " + s.name + ": " + s.symbolType.String()
}

// SymbolTable 符号表值对象
type SymbolTable struct {
	symbols map[string]*Symbol
	scopes  []*Scope
	globalScope *Scope
	currentScope *Scope
}

// NewSymbolTable 创建新的符号表
func NewSymbolTable() *SymbolTable {
	globalScope := NewScope("global", nil, SourceLocation{})
	return &SymbolTable{
		symbols:      make(map[string]*Symbol),
		scopes:       []*Scope{globalScope},
		globalScope:  globalScope,
		currentScope: globalScope,
	}
}

// AddSymbol 添加符号到当前作用域
func (st *SymbolTable) AddSymbol(symbol *Symbol) error {
	key := st.makeKey(symbol.name, symbol.scope)
	if _, exists := st.symbols[key]; exists {
		return NewParseError(
			"symbol already defined: " + symbol.name,
			symbol.location,
			ErrorTypeSymbol,
			SeverityError,
		)
	}
	st.symbols[key] = symbol
	return nil
}

// LookupSymbol 查找符号
func (st *SymbolTable) LookupSymbol(name string, scope *Scope) (*Symbol, bool) {
	// 从当前作用域开始向上查找
	current := scope
	for current != nil {
		key := st.makeKey(name, current)
		if symbol, exists := st.symbols[key]; exists {
			return symbol, true
		}
		current = current.Parent()
	}
	return nil, false
}

// LookupSymbolInCurrentScope 在当前作用域查找符号
func (st *SymbolTable) LookupSymbolInCurrentScope(name string) (*Symbol, bool) {
	key := st.makeKey(name, st.currentScope)
	symbol, exists := st.symbols[key]
	return symbol, exists
}

// EnterScope 进入新作用域
func (st *SymbolTable) EnterScope(name string, location SourceLocation) *Scope {
	newScope := NewScope(name, st.currentScope, location)
	st.scopes = append(st.scopes, newScope)
	st.currentScope = newScope
	return newScope
}

// ExitScope 退出当前作用域
func (st *SymbolTable) ExitScope() {
	if st.currentScope != st.globalScope {
		st.currentScope = st.currentScope.Parent()
	}
}

// CurrentScope 获取当前作用域
func (st *SymbolTable) CurrentScope() *Scope {
	return st.currentScope
}

// GlobalScope 获取全局作用域
func (st *SymbolTable) GlobalScope() *Scope {
	return st.globalScope
}

// AllSymbols 获取所有符号
func (st *SymbolTable) AllSymbols() map[string]*Symbol {
	return st.symbols
}

// makeKey 生成符号键
func (st *SymbolTable) makeKey(name string, scope *Scope) string {
	if scope == nil {
		return name
	}
	return scope.Name() + "::" + name
}

// Scope 作用域值对象
type Scope struct {
	name     string
	parent   *Scope
	location SourceLocation
	symbols  map[string]*Symbol
}

// NewScope 创建新的作用域
func NewScope(name string, parent *Scope, location SourceLocation) *Scope {
	return &Scope{
		name:     name,
		parent:   parent,
		location: location,
		symbols:  make(map[string]*Symbol),
	}
}

// Name 获取作用域名称
func (s *Scope) Name() string {
	return s.name
}

// Parent 获取父作用域
func (s *Scope) Parent() *Scope {
	return s.parent
}

// Location 获取作用域位置
func (s *Scope) Location() SourceLocation {
	return s.location
}

// IsGlobal 检查是否为全局作用域
func (s *Scope) IsGlobal() bool {
	return s.parent == nil
}

// AddSymbol 添加符号到作用域
func (s *Scope) AddSymbol(symbol *Symbol) {
	s.symbols[symbol.Name()] = symbol
}

// LookupSymbol 在当前作用域查找符号
func (s *Scope) LookupSymbol(name string) (*Symbol, bool) {
	symbol, exists := s.symbols[name]
	return symbol, exists
}

// AllSymbols 获取作用域内所有符号
func (s *Scope) AllSymbols() map[string]*Symbol {
	return s.symbols
}

// NewCustomSymbolType 创建自定义类型
func NewCustomSymbolType(typeName string) SymbolType {
	return SymbolType{typeName: typeName, isPrimitive: false}
}

// NewPrimitiveSymbolType 创建基本类型
func NewPrimitiveSymbolType(typeName string) SymbolType {
	return SymbolType{typeName: typeName, isPrimitive: true}
}
