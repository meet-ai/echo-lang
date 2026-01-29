package services

import (
	"strings"
	"testing"

	"echo/internal/modules/frontend/domain/entities"
)

func TestTypeInferenceService_InferTypes(t *testing.T) {
	// 创建类型推断服务
	service := NewSimpleTypeInferenceService()

	// 创建测试程序：let a = 100
	intLiteral := &entities.IntLiteral{Value: 100}
	varDecl := &entities.VarDecl{
		Name:     "a",
		Type:     "", // 空类型，表示需要推断
		Value:    intLiteral,
		Inferred: true,
	}

	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{varDecl},
	}

	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef},
	}

	// 执行类型推断
	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	// 验证结果
	if varDecl.Type != "int" {
		t.Errorf("Expected type 'int', got '%s'", varDecl.Type)
	}

	if varDecl.Inferred {
		t.Error("Expected Inferred to be false after inference")
	}
}

func TestTypeInferenceService_InferBoolLiteral(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	boolLiteral := &entities.BoolLiteral{Value: true}
	varDecl := &entities.VarDecl{
		Name:     "b",
		Type:     "",
		Value:    boolLiteral,
		Inferred: true,
	}

	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{varDecl},
	}

	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef},
	}

	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	if varDecl.Type != "bool" {
		t.Errorf("Expected type 'bool', got '%s'", varDecl.Type)
	}
}

func TestTypeInferenceService_InferStringLiteral(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	stringLiteral := &entities.StringLiteral{Value: "hello"}
	varDecl := &entities.VarDecl{
		Name:     "s",
		Type:     "",
		Value:    stringLiteral,
		Inferred: true,
	}

	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{varDecl},
	}

	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef},
	}

	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	if varDecl.Type != "string" {
		t.Errorf("Expected type 'string', got '%s'", varDecl.Type)
	}
}

func TestTypeInferenceService_ErrorHandling(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	tests := []struct {
		name        string
		expr        entities.Expr
		expectedErr string
	}{
		{
			name:        "variable reference",
			expr:        &entities.Identifier{Name: "x"},
			expectedErr: "cannot infer type from variable reference 'x'",
		},
		{
			name:        "function call",
			expr:        &entities.FuncCall{Name: "someFunc", Args: []entities.Expr{}},
			expectedErr: "cannot infer type from function call",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			varDecl := &entities.VarDecl{
				Name:     "testVar",
				Type:     "",
				Value:    tt.expr,
				Inferred: true,
			}

			funcDef := &entities.FuncDef{
				Name: "main",
				Body: []entities.ASTNode{varDecl},
			}

			program := &entities.Program{
				Statements: []entities.ASTNode{funcDef},
			}

			err := service.InferTypes(program)
			if err == nil {
				t.Errorf("Expected error for %s, but got none", tt.name)
			} else if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedErr, err.Error())
			}
		})
	}
}

// TestTypeInferenceService_InferArrayLiteral 测试数组字面量类型推断
func TestTypeInferenceService_InferArrayLiteral(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	tests := []struct {
		name         string
		elements     []entities.Expr
		expectedType string
		expectError  bool
		errorMsg     string
	}{
		{
			name: "integer array",
			elements: []entities.Expr{
				&entities.IntLiteral{Value: 1},
				&entities.IntLiteral{Value: 2},
				&entities.IntLiteral{Value: 3},
			},
			expectedType: "[int]",
			expectError:  false,
		},
		{
			name: "float array",
			elements: []entities.Expr{
				&entities.FloatLiteral{Value: 1.0},
				&entities.FloatLiteral{Value: 2.0},
			},
			expectedType: "[float]",
			expectError:  false,
		},
		{
			name: "string array",
			elements: []entities.Expr{
				&entities.StringLiteral{Value: "a"},
				&entities.StringLiteral{Value: "b"},
			},
			expectedType: "[string]",
			expectError:  false,
		},
		{
			name:        "empty array",
			elements:    []entities.Expr{},
			expectError: true,
			errorMsg:    "cannot infer type from empty array literal",
		},
		{
			name: "mixed types",
			elements: []entities.Expr{
				&entities.IntLiteral{Value: 1},
				&entities.StringLiteral{Value: "2"},
			},
			expectError: true,
			errorMsg:    "array elements have different types",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arrayLiteral := &entities.ArrayLiteral{Elements: tt.elements}
			varDecl := &entities.VarDecl{
				Name:     "arr",
				Type:     "",
				Value:    arrayLiteral,
				Inferred: true,
			}

			funcDef := &entities.FuncDef{
				Name: "main",
				Body: []entities.ASTNode{varDecl},
			}

			program := &entities.Program{
				Statements: []entities.ASTNode{funcDef},
			}

			err := service.InferTypes(program)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Type inference failed: %v", err)
				}
				if varDecl.Type != tt.expectedType {
					t.Errorf("Expected type '%s', got '%s'", tt.expectedType, varDecl.Type)
				}
			}
		})
	}
}

// TestTypeInferenceService_InferStructLiteral 测试结构体字面量类型推断
func TestTypeInferenceService_InferStructLiteral(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	structLiteral := &entities.StructLiteral{
		Type:   "Point",
		Fields: map[string]entities.Expr{"x": &entities.IntLiteral{Value: 10}},
	}

	varDecl := &entities.VarDecl{
		Name:     "p",
		Type:     "",
		Value:    structLiteral,
		Inferred: true,
	}

	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{varDecl},
	}

	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef},
	}

	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	if varDecl.Type != "Point" {
		t.Errorf("Expected type 'Point', got '%s'", varDecl.Type)
	}
}

// TestTypeInferenceService_InferIndexExpr 测试索引表达式类型推断
func TestTypeInferenceService_InferIndexExpr(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建数组字面量
	arrayLiteral := &entities.ArrayLiteral{
		Elements: []entities.Expr{
			&entities.IntLiteral{Value: 1},
			&entities.IntLiteral{Value: 2},
			&entities.IntLiteral{Value: 3},
		},
	}

	// 创建索引表达式
	indexExpr := &entities.IndexExpr{
		Array: arrayLiteral,
		Index: &entities.IntLiteral{Value: 0},
	}

	varDecl := &entities.VarDecl{
		Name:     "item",
		Type:     "",
		Value:    indexExpr,
		Inferred: true,
	}

	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{varDecl},
	}

	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef},
	}

	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	if varDecl.Type != "int" {
		t.Errorf("Expected type 'int', got '%s'", varDecl.Type)
	}
}

// TestTypeInferenceService_InferSliceExpr 测试切片表达式类型推断
func TestTypeInferenceService_InferSliceExpr(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建数组字面量
	arrayLiteral := &entities.ArrayLiteral{
		Elements: []entities.Expr{
			&entities.IntLiteral{Value: 1},
			&entities.IntLiteral{Value: 2},
			&entities.IntLiteral{Value: 3},
		},
	}

	// 创建切片表达式
	sliceExpr := &entities.SliceExpr{
		Array: arrayLiteral,
		Start: &entities.IntLiteral{Value: 1},
		End:   &entities.IntLiteral{Value: 3},
	}

	varDecl := &entities.VarDecl{
		Name:     "slice",
		Type:     "",
		Value:    sliceExpr,
		Inferred: true,
	}

	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{varDecl},
	}

	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef},
	}

	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	if varDecl.Type != "[int]" {
		t.Errorf("Expected type '[int]', got '%s'", varDecl.Type)
	}
}

// TestTypeInferenceService_InferBinaryExprFloat 测试浮点数二元表达式类型推断
func TestTypeInferenceService_InferBinaryExprFloat(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	binaryExpr := &entities.BinaryExpr{
		Left:  &entities.FloatLiteral{Value: 1.5},
		Op:    "+",
		Right: &entities.FloatLiteral{Value: 2.5},
	}

	varDecl := &entities.VarDecl{
		Name:     "result",
		Type:     "",
		Value:    binaryExpr,
		Inferred: true,
	}

	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{varDecl},
	}

	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef},
	}

	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	if varDecl.Type != "float" {
		t.Errorf("Expected type 'float', got '%s'", varDecl.Type)
	}
}

// TestTypeInferenceService_InferBinaryExprString 测试字符串拼接类型推断
func TestTypeInferenceService_InferBinaryExprString(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	binaryExpr := &entities.BinaryExpr{
		Left:  &entities.StringLiteral{Value: "hello"},
		Op:    "+",
		Right: &entities.StringLiteral{Value: "world"},
	}

	varDecl := &entities.VarDecl{
		Name:     "result",
		Type:     "",
		Value:    binaryExpr,
		Inferred: true,
	}

	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{varDecl},
	}

	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef},
	}

	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	if varDecl.Type != "string" {
		t.Errorf("Expected type 'string', got '%s'", varDecl.Type)
	}
}

// TestTypeInferenceService_InferIdentifier 测试变量引用类型推断
func TestTypeInferenceService_InferIdentifier(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建第一个变量（显式类型）
	intLiteral := &entities.IntLiteral{Value: 100}
	varDecl1 := &entities.VarDecl{
		Name:     "x",
		Type:     "int", // 显式类型
		Value:    intLiteral,
		Inferred: false,
	}

	// 创建第二个变量（从第一个变量推断）
	identifier := &entities.Identifier{Name: "x"}
	varDecl2 := &entities.VarDecl{
		Name:     "y",
		Type:     "",
		Value:    identifier,
		Inferred: true,
	}

	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{varDecl1, varDecl2},
	}

	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef},
	}

	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	if varDecl2.Type != "int" {
		t.Errorf("Expected type 'int', got '%s'", varDecl2.Type)
	}
}

// TestTypeInferenceService_InferFuncCall 测试函数调用类型推断
func TestTypeInferenceService_InferFuncCall(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建函数定义
	funcDef1 := &entities.FuncDef{
		Name:       "add",
		Params:     []entities.Param{{Name: "a", Type: "int"}, {Name: "b", Type: "int"}},
		ReturnType: "int",
		Body:       []entities.ASTNode{},
	}

	// 创建函数调用
	funcCall := &entities.FuncCall{
		Name: "add",
		Args: []entities.Expr{
			&entities.IntLiteral{Value: 10},
			&entities.IntLiteral{Value: 20},
		},
	}

	varDecl := &entities.VarDecl{
		Name:     "result",
		Type:     "",
		Value:    funcCall,
		Inferred: true,
	}

	funcDef2 := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{varDecl},
	}

	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef1, funcDef2},
	}

	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	if varDecl.Type != "int" {
		t.Errorf("Expected type 'int', got '%s'", varDecl.Type)
	}
}

// TestTypeInferenceService_InferIndexExprFromIdentifier 测试从变量引用推断索引表达式类型
func TestTypeInferenceService_InferIndexExprFromIdentifier(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建数组变量（显式类型）
	arrayLiteral := &entities.ArrayLiteral{
		Elements: []entities.Expr{
			&entities.IntLiteral{Value: 1},
			&entities.IntLiteral{Value: 2},
			&entities.IntLiteral{Value: 3},
		},
	}
	varDecl1 := &entities.VarDecl{
		Name:     "arr",
		Type:     "[int]", // 显式类型
		Value:    arrayLiteral,
		Inferred: false,
	}

	// 创建索引表达式（从变量引用）
	indexExpr := &entities.IndexExpr{
		Array: &entities.Identifier{Name: "arr"},
		Index: &entities.IntLiteral{Value: 0},
	}
	varDecl2 := &entities.VarDecl{
		Name:     "item",
		Type:     "",
		Value:    indexExpr,
		Inferred: true,
	}

	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{varDecl1, varDecl2},
	}

	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef},
	}

	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	if varDecl2.Type != "int" {
		t.Errorf("Expected type 'int', got '%s'", varDecl2.Type)
	}
}

// TestTypeInferenceService_InferMethodCall 测试方法调用类型推断
func TestTypeInferenceService_InferMethodCall(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建结构体定义
	structDef := &entities.StructDef{
		Name: "Point",
		Fields: []entities.StructField{
			{Name: "x", Type: "int"},
			{Name: "y", Type: "int"},
		},
	}

	// 创建方法定义
	methodDef := &entities.MethodDef{
		Receiver:   "Point",
		Name:       "distance",
		ReturnType: "float",
		Body:       []entities.ASTNode{},
	}

	// 创建结构体字面量
	structLiteral := &entities.StructLiteral{
		Type: "Point",
		Fields: map[string]entities.Expr{
			"x": &entities.IntLiteral{Value: 1},
			"y": &entities.IntLiteral{Value: 2},
		},
	}

	// 创建变量（结构体实例）
	varDecl1 := &entities.VarDecl{
		Name:     "p",
		Type:     "Point",
		Value:    structLiteral,
		Inferred: false,
	}

	// 创建方法调用
	methodCall := &entities.MethodCallExpr{
		Receiver:   &entities.Identifier{Name: "p"},
		MethodName: "distance",
		Args:       []entities.Expr{},
	}

	varDecl2 := &entities.VarDecl{
		Name:     "d",
		Type:     "",
		Value:    methodCall,
		Inferred: true,
	}

	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{varDecl1, varDecl2},
	}

	program := &entities.Program{
		Statements: []entities.ASTNode{structDef, methodDef, funcDef},
	}

	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	if varDecl2.Type != "float" {
		t.Errorf("Expected type 'float', got '%s'", varDecl2.Type)
	}
}

// TestTypeInferenceService_InferFieldAccess 测试结构体字段访问类型推断
func TestTypeInferenceService_InferFieldAccess(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建结构体定义
	structDef := &entities.StructDef{
		Name: "Point",
		Fields: []entities.StructField{
			{Name: "x", Type: "int"},
			{Name: "y", Type: "int"},
		},
	}

	// 创建结构体字面量
	structLiteral := &entities.StructLiteral{
		Type: "Point",
		Fields: map[string]entities.Expr{
			"x": &entities.IntLiteral{Value: 1},
			"y": &entities.IntLiteral{Value: 2},
		},
	}

	// 创建变量（结构体实例）
	varDecl1 := &entities.VarDecl{
		Name:     "p",
		Type:     "Point",
		Value:    structLiteral,
		Inferred: false,
	}

	// 创建字段访问
	fieldAccess := &entities.StructAccess{
		Object: &entities.Identifier{Name: "p"},
		Field:  "x",
	}

	varDecl2 := &entities.VarDecl{
		Name:     "x",
		Type:     "",
		Value:    fieldAccess,
		Inferred: true,
	}

	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{varDecl1, varDecl2},
	}

	program := &entities.Program{
		Statements: []entities.ASTNode{structDef, funcDef},
	}

	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	if varDecl2.Type != "int" {
		t.Errorf("Expected type 'int', got '%s'", varDecl2.Type)
	}
}

// TestTypeInferenceService_InferLenExpr 测试长度表达式类型推断
func TestTypeInferenceService_InferLenExpr(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建数组字面量
	arrayLiteral := &entities.ArrayLiteral{
		Elements: []entities.Expr{
			&entities.IntLiteral{Value: 1},
			&entities.IntLiteral{Value: 2},
			&entities.IntLiteral{Value: 3},
		},
	}

	// 创建 len() 表达式
	lenExpr := &entities.LenExpr{
		Array: arrayLiteral,
	}

	// 创建变量声明，使用类型推断
	varDecl := &entities.VarDecl{
		Name:     "length",
		Value:    lenExpr,
		Inferred: true,
	}

	// 创建程序
	program := &entities.Program{
		Statements: []entities.ASTNode{varDecl},
	}

	// 执行类型推断
	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	// 验证推断结果：len() 应该返回 int
	if varDecl.Type != "int" {
		t.Errorf("Expected type 'int' for len() expression, got '%s'", varDecl.Type)
	}

	// 验证变量已推断（Inferred 应该为 false，表示已推断完成）
	if varDecl.Inferred {
		t.Error("Expected variable to be marked as inferred (Inferred should be false after inference)")
	}
}

// TestTypeInferenceService_InferLenExprFromIdentifier 测试从标识符推断 len() 类型
func TestTypeInferenceService_InferLenExprFromIdentifier(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建数组变量声明
	arrayVarDecl := &entities.VarDecl{
		Name:     "arr",
		Type:     "[int]",
		Value:    &entities.ArrayLiteral{Elements: []entities.Expr{&entities.IntLiteral{Value: 1}}},
		Inferred: false,
	}

	// 创建 len(arr) 表达式
	lenExpr := &entities.LenExpr{
		Array: &entities.Identifier{Name: "arr"},
	}

	// 创建使用 len() 的变量声明
	lengthVarDecl := &entities.VarDecl{
		Name:     "length",
		Value:    lenExpr,
		Inferred: true,
	}

	// 创建函数定义
	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{arrayVarDecl, lengthVarDecl},
	}

	// 创建程序
	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef},
	}

	// 执行类型推断
	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	// 验证推断结果：len() 应该返回 int
	if lengthVarDecl.Type != "int" {
		t.Errorf("Expected type 'int' for len() expression, got '%s'", lengthVarDecl.Type)
	}
}

// TestTypeInferenceService_InferArrayMethodCall 测试数组方法调用类型推断
func TestTypeInferenceService_InferArrayMethodCall(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建数组变量声明
	arrayVarDecl := &entities.VarDecl{
		Name:     "arr",
		Type:     "[int]",
		Value:    &entities.ArrayLiteral{Elements: []entities.Expr{&entities.IntLiteral{Value: 1}}},
		Inferred: false,
	}

	// 创建 arr.pop() 表达式
	popMethodCall := &entities.ArrayMethodCallExpr{
		Array:  &entities.Identifier{Name: "arr"},
		Method: "pop",
		Args:   []entities.Expr{},
	}

	// 创建使用 pop() 的变量声明
	popVarDecl := &entities.VarDecl{
		Name:     "last",
		Value:    popMethodCall,
		Inferred: true,
	}

	// 创建 arr.push(6) 表达式
	pushMethodCall := &entities.ArrayMethodCallExpr{
		Array:  &entities.Identifier{Name: "arr"},
		Method: "push",
		Args:   []entities.Expr{&entities.IntLiteral{Value: 6}},
	}

	// 创建使用 push() 的变量声明（应该返回 void）
	pushVarDecl := &entities.VarDecl{
		Name:     "result",
		Value:    pushMethodCall,
		Inferred: true,
	}

	// 创建函数定义
	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{arrayVarDecl, popVarDecl, pushVarDecl},
	}

	// 创建程序
	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef},
	}

	// 执行类型推断
	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	// 验证 pop() 返回元素类型（int）
	if popVarDecl.Type != "int" {
		t.Errorf("Expected type 'int' for pop() method call, got '%s'", popVarDecl.Type)
	}

	// 验证 push() 返回 void
	if pushVarDecl.Type != "void" {
		t.Errorf("Expected type 'void' for push() method call, got '%s'", pushVarDecl.Type)
	}
}

// TestTypeInferenceService_InferArrayMethodCallFromLiteral 测试从数组字面量推断方法调用类型
func TestTypeInferenceService_InferArrayMethodCallFromLiteral(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建数组字面量
	arrayLiteral := &entities.ArrayLiteral{
		Elements: []entities.Expr{
			&entities.IntLiteral{Value: 1},
			&entities.IntLiteral{Value: 2},
			&entities.IntLiteral{Value: 3},
		},
	}

	// 创建 [1, 2, 3].pop() 表达式
	popMethodCall := &entities.ArrayMethodCallExpr{
		Array:  arrayLiteral,
		Method: "pop",
		Args:   []entities.Expr{},
	}

	// 创建变量声明
	varDecl := &entities.VarDecl{
		Name:     "last",
		Value:    popMethodCall,
		Inferred: true,
	}

	// 创建程序
	program := &entities.Program{
		Statements: []entities.ASTNode{varDecl},
	}

	// 执行类型推断
	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	// 验证 pop() 返回元素类型（int）
	if varDecl.Type != "int" {
		t.Errorf("Expected type 'int' for pop() method call, got '%s'", varDecl.Type)
	}
}

// TestTypeInferenceService_InferResultExpr 测试 Result 类型表达式推断
func TestTypeInferenceService_InferResultExpr(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建 Result[int, string] 表达式
	resultExpr := &entities.ResultExpr{
		OkType:  "int",
		ErrType: "string",
	}

	// 创建变量声明
	varDecl := &entities.VarDecl{
		Name:     "result",
		Value:    resultExpr,
		Inferred: true,
	}

	// 创建程序
	program := &entities.Program{
		Statements: []entities.ASTNode{varDecl},
	}

	// 执行类型推断
	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	// 验证推断结果：Result[int, string]
	expectedType := "Result[int, string]"
	if varDecl.Type != expectedType {
		t.Errorf("Expected type '%s', got '%s'", expectedType, varDecl.Type)
	}
}

// TestTypeInferenceService_InferOptionExpr 测试 Option 类型表达式推断
func TestTypeInferenceService_InferOptionExpr(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建 Option(42) 表达式（包装一个整数）
	optionExpr := &entities.OptionExpr{
		Expr: &entities.IntLiteral{Value: 42},
	}

	// 创建变量声明
	varDecl := &entities.VarDecl{
		Name:     "opt",
		Value:    optionExpr,
		Inferred: true,
	}

	// 创建程序
	program := &entities.Program{
		Statements: []entities.ASTNode{varDecl},
	}

	// 执行类型推断
	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	// 验证推断结果：Option[int]
	expectedType := "Option[int]"
	if varDecl.Type != expectedType {
		t.Errorf("Expected type '%s', got '%s'", expectedType, varDecl.Type)
	}
}

// TestTypeInferenceService_InferOptionExprFromIdentifier 测试从标识符推断 Option 类型
func TestTypeInferenceService_InferOptionExprFromIdentifier(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建整数变量
	intVarDecl := &entities.VarDecl{
		Name:     "value",
		Type:     "int",
		Value:    &entities.IntLiteral{Value: 42},
		Inferred: false,
	}

	// 创建 Option(value) 表达式
	optionExpr := &entities.OptionExpr{
		Expr: &entities.Identifier{Name: "value"},
	}

	// 创建变量声明
	optVarDecl := &entities.VarDecl{
		Name:     "opt",
		Value:    optionExpr,
		Inferred: true,
	}

	// 创建函数定义
	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{intVarDecl, optVarDecl},
	}

	// 创建程序
	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef},
	}

	// 执行类型推断
	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	// 验证推断结果：Option[int]
	expectedType := "Option[int]"
	if optVarDecl.Type != expectedType {
		t.Errorf("Expected type '%s', got '%s'", expectedType, optVarDecl.Type)
	}
}

// TestTypeInferenceService_InferOkLiteral 测试 Ok 字面量类型推断
func TestTypeInferenceService_InferOkLiteral(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建 Ok(42) 表达式
	okLiteral := &entities.OkLiteral{
		Value: &entities.IntLiteral{Value: 42},
	}

	// 创建变量声明
	varDecl := &entities.VarDecl{
		Name:     "result",
		Value:    okLiteral,
		Inferred: true,
	}

	// 创建程序
	program := &entities.Program{
		Statements: []entities.ASTNode{varDecl},
	}

	// 执行类型推断
	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	// 验证推断结果：Result[int, string]
	expectedType := "Result[int, string]"
	if varDecl.Type != expectedType {
		t.Errorf("Expected type '%s', got '%s'", expectedType, varDecl.Type)
	}
}

// TestTypeInferenceService_InferOkLiteralFromIdentifier 测试从变量推断 Ok 字面量类型
func TestTypeInferenceService_InferOkLiteralFromIdentifier(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建字符串变量
	strVarDecl := &entities.VarDecl{
		Name:     "value",
		Type:     "string",
		Value:    &entities.StringLiteral{Value: "success"},
		Inferred: false,
	}

	// 创建 Ok(value) 表达式
	okLiteral := &entities.OkLiteral{
		Value: &entities.Identifier{Name: "value"},
	}

	// 创建变量声明
	okVarDecl := &entities.VarDecl{
		Name:     "result",
		Value:    okLiteral,
		Inferred: true,
	}

	// 创建函数定义
	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{strVarDecl, okVarDecl},
	}

	// 创建程序
	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef},
	}

	// 执行类型推断
	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	// 验证推断结果：Result[string, string]
	expectedType := "Result[string, string]"
	if okVarDecl.Type != expectedType {
		t.Errorf("Expected type '%s', got '%s'", expectedType, okVarDecl.Type)
	}
}

// TestTypeInferenceService_InferErrLiteral 测试 Err 字面量类型推断（需要上下文）
func TestTypeInferenceService_InferErrLiteral(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建 Err("error") 表达式
	errLiteral := &entities.ErrLiteral{
		Error: &entities.StringLiteral{Value: "error"},
	}

	// 创建变量声明
	varDecl := &entities.VarDecl{
		Name:     "result",
		Value:    errLiteral,
		Inferred: true,
	}

	// 创建程序
	program := &entities.Program{
		Statements: []entities.ASTNode{varDecl},
	}

	// 执行类型推断（应该失败，因为 ErrLiteral 需要上下文推断 OkType）
	err := service.InferTypes(program)
	if err == nil {
		t.Error("Expected error for ErrLiteral without context, but got nil")
	}

	// 验证错误信息包含提示
	if !strings.Contains(err.Error(), "context") {
		t.Errorf("Expected error message to mention 'context', got: %v", err)
	}
}

// TestTypeInferenceService_InferSomeLiteral 测试 Some 字面量类型推断
func TestTypeInferenceService_InferSomeLiteral(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建 Some(42) 表达式
	someLiteral := &entities.SomeLiteral{
		Value: &entities.IntLiteral{Value: 42},
	}

	// 创建变量声明
	varDecl := &entities.VarDecl{
		Name:     "opt",
		Value:    someLiteral,
		Inferred: true,
	}

	// 创建程序
	program := &entities.Program{
		Statements: []entities.ASTNode{varDecl},
	}

	// 执行类型推断
	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	// 验证推断结果：Option[int]
	expectedType := "Option[int]"
	if varDecl.Type != expectedType {
		t.Errorf("Expected type '%s', got '%s'", expectedType, varDecl.Type)
	}
}

// TestTypeInferenceService_InferSomeLiteralFromIdentifier 测试从变量推断 Some 字面量类型
func TestTypeInferenceService_InferSomeLiteralFromIdentifier(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建字符串变量
	strVarDecl := &entities.VarDecl{
		Name:     "value",
		Type:     "string",
		Value:    &entities.StringLiteral{Value: "hello"},
		Inferred: false,
	}

	// 创建 Some(value) 表达式
	someLiteral := &entities.SomeLiteral{
		Value: &entities.Identifier{Name: "value"},
	}

	// 创建变量声明
	someVarDecl := &entities.VarDecl{
		Name:     "opt",
		Value:    someLiteral,
		Inferred: true,
	}

	// 创建函数定义
	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{strVarDecl, someVarDecl},
	}

	// 创建程序
	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef},
	}

	// 执行类型推断
	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	// 验证推断结果：Option[string]
	expectedType := "Option[string]"
	if someVarDecl.Type != expectedType {
		t.Errorf("Expected type '%s', got '%s'", expectedType, someVarDecl.Type)
	}
}

// TestTypeInferenceService_InferNoneLiteral 测试 None 字面量类型推断（需要上下文）
func TestTypeInferenceService_InferNoneLiteral(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建 None 表达式
	noneLiteral := &entities.NoneLiteral{}

	// 创建变量声明
	varDecl := &entities.VarDecl{
		Name:     "opt",
		Value:    noneLiteral,
		Inferred: true,
	}

	// 创建程序
	program := &entities.Program{
		Statements: []entities.ASTNode{varDecl},
	}

	// 执行类型推断（应该失败，因为 NoneLiteral 需要上下文推断内部类型）
	err := service.InferTypes(program)
	if err == nil {
		t.Error("Expected error for NoneLiteral without context, but got nil")
	}

	// 验证错误信息包含提示
	if !strings.Contains(err.Error(), "context") {
		t.Errorf("Expected error message to mention 'context', got: %v", err)
	}
}

// TestTypeInferenceService_InferChanLiteral 测试 Channel 字面量类型推断
func TestTypeInferenceService_InferChanLiteral(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建 chan[int]() 表达式
	chanLiteral := &entities.ChanLiteral{
		Type: "int",
	}

	// 创建变量声明
	varDecl := &entities.VarDecl{
		Name:     "ch",
		Value:    chanLiteral,
		Inferred: true,
	}

	// 创建程序
	program := &entities.Program{
		Statements: []entities.ASTNode{varDecl},
	}

	// 执行类型推断
	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	// 验证推断结果：chan[int]
	expectedType := "chan[int]"
	if varDecl.Type != expectedType {
		t.Errorf("Expected type '%s', got '%s'", expectedType, varDecl.Type)
	}
}

// TestTypeInferenceService_InferSendExpr 测试发送表达式类型推断
func TestTypeInferenceService_InferSendExpr(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建 Channel 变量
	chanVarDecl := &entities.VarDecl{
		Name:     "ch",
		Type:     "chan[int]",
		Value:    &entities.ChanLiteral{Type: "int"},
		Inferred: false,
	}

	// 创建发送表达式：ch <- 42
	sendExpr := &entities.SendExpr{
		Channel: &entities.Identifier{Name: "ch"},
		Value:   &entities.IntLiteral{Value: 42},
	}

	// 创建变量声明
	sendVarDecl := &entities.VarDecl{
		Name:     "result",
		Value:    sendExpr,
		Inferred: true,
	}

	// 创建函数定义
	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{chanVarDecl, sendVarDecl},
	}

	// 创建程序
	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef},
	}

	// 执行类型推断
	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	// 验证推断结果：void
	expectedType := "void"
	if sendVarDecl.Type != expectedType {
		t.Errorf("Expected type '%s', got '%s'", expectedType, sendVarDecl.Type)
	}
}

// TestTypeInferenceService_InferReceiveExpr 测试接收表达式类型推断
func TestTypeInferenceService_InferReceiveExpr(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建 Channel 变量
	chanVarDecl := &entities.VarDecl{
		Name:     "ch",
		Type:     "chan[string]",
		Value:    &entities.ChanLiteral{Type: "string"},
		Inferred: false,
	}

	// 创建接收表达式：<-ch
	receiveExpr := &entities.ReceiveExpr{
		Channel: &entities.Identifier{Name: "ch"},
	}

	// 创建变量声明
	receiveVarDecl := &entities.VarDecl{
		Name:     "value",
		Value:    receiveExpr,
		Inferred: true,
	}

	// 创建函数定义
	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{chanVarDecl, receiveVarDecl},
	}

	// 创建程序
	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef},
	}

	// 执行类型推断
	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	// 验证推断结果：string（从 chan[string] 提取）
	expectedType := "string"
	if receiveVarDecl.Type != expectedType {
		t.Errorf("Expected type '%s', got '%s'", expectedType, receiveVarDecl.Type)
	}
}

// TestTypeInferenceService_InferAwaitExpr 测试异步等待表达式类型推断
func TestTypeInferenceService_InferAwaitExpr(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建 Future 变量（模拟异步函数返回）
	futureVarDecl := &entities.VarDecl{
		Name:     "future",
		Type:     "Future[int]",
		Value:    &entities.IntLiteral{Value: 0}, // 占位符
		Inferred: false,
	}

	// 创建异步等待表达式：await future
	awaitExpr := &entities.AwaitExpr{
		Expression: &entities.Identifier{Name: "future"},
	}

	// 创建变量声明
	awaitVarDecl := &entities.VarDecl{
		Name:     "result",
		Value:    awaitExpr,
		Inferred: true,
	}

	// 创建函数定义
	funcDef := &entities.FuncDef{
		Name: "main",
		Body: []entities.ASTNode{futureVarDecl, awaitVarDecl},
	}

	// 创建程序
	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef},
	}

	// 执行类型推断
	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	// 验证推断结果：int（从 Future[int] 提取）
	expectedType := "int"
	if awaitVarDecl.Type != expectedType {
		t.Errorf("Expected type '%s', got '%s'", expectedType, awaitVarDecl.Type)
	}
}

// TestTypeInferenceService_InferSpawnExpr 测试生成协程表达式类型推断
func TestTypeInferenceService_InferSpawnExpr(t *testing.T) {
	service := NewSimpleTypeInferenceService()

	// 创建函数定义（用于注册到函数表）
	funcDef := &entities.FuncDef{
		Name:       "fetchData",
		ReturnType: "string",
		Params:     []entities.Param{},
		Body:       []entities.ASTNode{},
	}

	// 创建生成协程表达式：spawn fetchData()
	spawnExpr := &entities.SpawnExpr{
		Function: &entities.Identifier{Name: "fetchData"},
		Args:     []entities.Expr{},
	}

	// 创建变量声明
	spawnVarDecl := &entities.VarDecl{
		Name:     "future",
		Value:    spawnExpr,
		Inferred: true,
	}

	// 创建程序（先注册函数，再推断类型）
	program := &entities.Program{
		Statements: []entities.ASTNode{funcDef, spawnVarDecl},
	}

	// 执行类型推断
	err := service.InferTypes(program)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	// 验证推断结果：Future[string]
	expectedType := "Future[string]"
	if spawnVarDecl.Type != expectedType {
		t.Errorf("Expected type '%s', got '%s'", expectedType, spawnVarDecl.Type)
	}
}
