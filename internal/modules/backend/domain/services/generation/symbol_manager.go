package generation

// SymbolInfo 符号信息
type SymbolInfo struct {
	Name    string
	Type    string
	Value   interface{} // LLVM IR值
	IsAsync bool        // 是否是async函数
}

// SymbolManager 符号管理领域服务接口
// 职责：负责变量符号的注册、查找和管理
type SymbolManager interface {
	// 注册符号
	RegisterSymbol(name string, symbolType string, value interface{}) error
	// 注册函数符号
	RegisterFunctionSymbol(name string, symbolType string, value interface{}, isAsync bool) error

	// 查找符号
	LookupSymbol(name string) (*SymbolInfo, error)

	// 更新符号值
	UpdateSymbolValue(name string, value interface{}) error

	// 检查符号是否存在（在所有作用域中）
	SymbolExists(name string) bool

	// 检查符号是否在当前作用域存在
	SymbolExistsInCurrentScope(name string) bool

	// 进入新的作用域
	EnterScope() error

	// 退出当前作用域
	ExitScope() error

	// 获取当前作用域层级
	GetCurrentScopeLevel() int

	// 清理所有符号（用于重置）
	Clear() error
}
