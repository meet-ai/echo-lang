package impl

import (
	"testing"
)

func TestSymbolManagerImpl_RegisterAndLookup(t *testing.T) {
	sm := NewSymbolManagerImpl()

	// 测试注册符号
	err := sm.RegisterSymbol("x", "int", "i32")
	if err != nil {
		t.Errorf("Failed to register symbol: %v", err)
	}

	// 测试查找符号
	symbol, err := sm.LookupSymbol("x")
	if err != nil {
		t.Errorf("Failed to lookup symbol: %v", err)
	}
	if symbol == nil {
		t.Error("Symbol should not be nil")
	}
	if symbol.Name != "x" {
		t.Errorf("Expected symbol name 'x', got '%s'", symbol.Name)
	}
	if symbol.Type != "int" {
		t.Errorf("Expected symbol type 'int', got '%s'", symbol.Type)
	}
	if symbol.Value != "i32" {
		t.Errorf("Expected symbol value 'i32', got '%v'", symbol.Value)
	}
}

func TestSymbolManagerImpl_ScopeManagement(t *testing.T) {
	sm := NewSymbolManagerImpl()

	// 进入作用域
	err := sm.EnterScope()
	if err != nil {
		t.Errorf("Failed to enter scope: %v", err)
	}

	// 在新作用域中注册符号
	err = sm.RegisterSymbol("localVar", "string", "i8*")
	if err != nil {
		t.Errorf("Failed to register symbol in new scope: %v", err)
	}

	// 在全局作用域注册符号
	err = sm.ExitScope()
	if err != nil {
		t.Errorf("Failed to exit scope: %v", err)
	}

	err = sm.RegisterSymbol("globalVar", "bool", "i1")
	if err != nil {
		t.Errorf("Failed to register symbol in global scope: %v", err)
	}

	// 测试符号查找（退出作用域后，localVar应该不可见）
	localSymbol, err := sm.LookupSymbol("localVar")
	if err == nil {
		t.Error("Expected error when looking up localVar after exiting scope")
	}
	if localSymbol != nil {
		t.Error("Local symbol should not be accessible after exiting scope")
	}

	globalSymbol, err := sm.LookupSymbol("globalVar")
	if err != nil {
		t.Errorf("Failed to lookup global symbol: %v", err)
	}
	if globalSymbol == nil {
		t.Error("Global symbol should not be nil")
	}
}

func TestSymbolManagerImpl_SymbolNotFound(t *testing.T) {
	sm := NewSymbolManagerImpl()

	_, err := sm.LookupSymbol("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent symbol")
	}
}

func TestSymbolManagerImpl_SymbolAlreadyDeclared(t *testing.T) {
	sm := NewSymbolManagerImpl()

	// 第一次注册应该成功
	err := sm.RegisterSymbol("x", "int", "i32")
	if err != nil {
		t.Errorf("First registration failed: %v", err)
	}

	// 第二次注册应该失败
	err = sm.RegisterSymbol("x", "int", "i32")
	if err == nil {
		t.Error("Expected error for duplicate symbol registration")
	}
}

func TestSymbolManagerImpl_ExitGlobalScope(t *testing.T) {
	sm := NewSymbolManagerImpl()

	// 尝试退出全局作用域应该失败
	err := sm.ExitScope()
	if err == nil {
		t.Error("Expected error when exiting global scope")
	}
}

func TestSymbolManagerImpl_SymbolExists(t *testing.T) {
	sm := NewSymbolManagerImpl()

	// 初始状态不存在
	if sm.SymbolExists("x") {
		t.Error("Symbol should not exist initially")
	}

	// 注册后应该存在
	err := sm.RegisterSymbol("x", "int", "i32")
	if err != nil {
		t.Errorf("Failed to register symbol: %v", err)
	}

	if !sm.SymbolExists("x") {
		t.Error("Symbol should exist after registration")
	}
}

func TestSymbolManagerImpl_GetCurrentScopeLevel(t *testing.T) {
	sm := NewSymbolManagerImpl()

	// 初始作用域级别应该是1（全局作用域）
	level := sm.GetCurrentScopeLevel()
	if level != 0 {
		t.Errorf("Expected initial scope level 0, got %d", level)
	}

	// 进入作用域后级别应该增加
	err := sm.EnterScope()
	if err != nil {
		t.Errorf("Failed to enter scope: %v", err)
	}

	level = sm.GetCurrentScopeLevel()
	if level != 1 {
		t.Errorf("Expected scope level 1 after entering, got %d", level)
	}

	// 退出作用域后级别应该减少
	err = sm.ExitScope()
	if err != nil {
		t.Errorf("Failed to exit scope: %v", err)
	}

	level = sm.GetCurrentScopeLevel()
	if level != 0 {
		t.Errorf("Expected scope level 0 after exiting, got %d", level)
	}
}

func TestSymbolManagerImpl_Clear(t *testing.T) {
	sm := NewSymbolManagerImpl()

	// 注册符号
	err := sm.RegisterSymbol("x", "int", "i32")
	if err != nil {
		t.Errorf("Failed to register symbol: %v", err)
	}

	// 清除后应该不存在
	err = sm.Clear()
	if err != nil {
		t.Errorf("Failed to clear symbols: %v", err)
	}

	if sm.SymbolExists("x") {
		t.Error("Symbol should not exist after clear")
	}

	// 清除后作用域级别应该是0
	level := sm.GetCurrentScopeLevel()
	if level != 0 {
		t.Errorf("Expected scope level 0 after clear, got %d", level)
	}
}
