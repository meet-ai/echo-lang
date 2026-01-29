package impl

import (
	"testing"

	"echo/internal/modules/frontend/domain/entities"

	"github.com/llir/llvm/ir"
)

// MockIRManager 简单的mock IR管理器用于测试
type MockIRManager struct{}

func (m *MockIRManager) AddGlobalVariable(name string, value interface{}) error { return nil }
func (m *MockIRManager) AddStringConstant(content string) (*ir.Global, error) { return nil, nil }
func (m *MockIRManager) CreateFunction(name string, returnType interface{}, paramTypes []interface{}) (interface{}, error) { return nil, nil }
func (m *MockIRManager) GetExternalFunction(name string) (interface{}, bool) { return nil, false }
func (m *MockIRManager) GetFunction(name string) (interface{}, bool) { return nil, false }
func (m *MockIRManager) GetCurrentFunction() interface{} { return nil }
func (m *MockIRManager) SetCurrentFunction(fn interface{}) error { return nil }
func (m *MockIRManager) CreateBasicBlock(name string) (interface{}, error) { return nil, nil }
func (m *MockIRManager) GetBlockByName(name string) (interface{}, error) { return nil, nil }
func (m *MockIRManager) GetCurrentBasicBlock() interface{} { return nil }
func (m *MockIRManager) SetCurrentBasicBlock(block interface{}) error { return nil }
func (m *MockIRManager) CreateAlloca(typ interface{}, name string) (interface{}, error) { return nil, nil }
func (m *MockIRManager) CreateStore(value interface{}, ptr interface{}) error { return nil }
func (m *MockIRManager) CreateLoad(typ interface{}, ptr interface{}, name string) (interface{}, error) { return nil, nil }
func (m *MockIRManager) CreateRet(value interface{}) error { return nil }
func (m *MockIRManager) CreateCall(fn interface{}, args ...interface{}) (interface{}, error) { return nil, nil }
func (m *MockIRManager) CreateBinaryOp(op string, left interface{}, right interface{}, name string) (interface{}, error) { return nil, nil }
func (m *MockIRManager) CreateBr(dest interface{}) error { return nil }
func (m *MockIRManager) CreateCondBr(cond interface{}, trueDest interface{}, falseDest interface{}) error { return nil }
func (m *MockIRManager) CreateGetElementPtr(elemType interface{}, ptr interface{}, indices ...interface{}) (interface{}, error) { return nil, nil }
func (m *MockIRManager) CreateBitCast(value interface{}, targetType interface{}, name string) (interface{}, error) { return nil, nil }
func (m *MockIRManager) GetIRString() string { return "" }
func (m *MockIRManager) Validate() error { return nil }
func (m *MockIRManager) InitializeMainFunction() error { return nil }
func (m *MockIRManager) Finalize() error { return nil }

func TestExpressionEvaluatorImpl_EvaluateIntLiteral(t *testing.T) {
	mockIR := &MockIRManager{}
	ee := NewExpressionEvaluatorImpl(NewSymbolManagerImpl(), NewTypeMapperImpl(), mockIR)

	expr := &entities.IntLiteral{Value: 42}
	result, err := ee.EvaluateIntLiteral(mockIR, expr)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	// 结果现在是LLVM常量对象，不再是字符串
	if result == nil {
		t.Errorf("Expected non-nil result")
	}
}

func TestExpressionEvaluatorImpl_EvaluateStringLiteral(t *testing.T) {
	mockIR := &MockIRManager{}
	ee := NewExpressionEvaluatorImpl(NewSymbolManagerImpl(), NewTypeMapperImpl(), mockIR)

	expr := &entities.StringLiteral{Value: "hello"}
	result, err := ee.EvaluateStringLiteral(mockIR, expr)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Errorf("Expected non-nil result")
	}
}

func TestExpressionEvaluatorImpl_EvaluateIdentifier(t *testing.T) {
	mockIR := &MockIRManager{}
	sm := NewSymbolManagerImpl()
	tm := NewTypeMapperImpl()
	ee := NewExpressionEvaluatorImpl(sm, tm, mockIR)

	// 注册符号
	err := sm.RegisterSymbol("x", "int", "i32")
	if err != nil {
		t.Errorf("Failed to register symbol: %v", err)
	}

	expr := &entities.Identifier{Name: "x"}
	result, err := ee.EvaluateIdentifier(mockIR, expr)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Errorf("Expected non-nil result")
	}
}

func TestExpressionEvaluatorImpl_EvaluateBinaryExpr(t *testing.T) {
	mockIR := &MockIRManager{}
	ee := NewExpressionEvaluatorImpl(NewSymbolManagerImpl(), NewTypeMapperImpl(), mockIR)

	tests := []struct {
		left     int
		operator string
		right    int
	}{
		{5, "+", 3},
		{10, "-", 4},
		{6, "*", 7},
		{8, "/", 2},
		{5, "==", 5},
		{3, "!=", 4},
		{7, "<", 10},
		{9, ">", 6},
		{5, "<=", 5},
		{8, ">=", 7},
	}

	for _, test := range tests {
		expr := &entities.BinaryExpr{
			Left:     &entities.IntLiteral{Value: test.left},
			Op:       test.operator,
			Right:    &entities.IntLiteral{Value: test.right},
		}
		result, err := ee.EvaluateBinaryExpr(mockIR, expr)
		if err != nil {
			t.Errorf("Unexpected error for %d %s %d: %v",
				test.left, test.operator, test.right, err)
		}
		if result == nil {
			t.Errorf("For %d %s %d, expected non-nil result",
				test.left, test.operator, test.right)
		}
	}
}

func TestExpressionEvaluatorImpl_EvaluateFuncCall(t *testing.T) {
	mockIR := &MockIRManager{}
	ee := NewExpressionEvaluatorImpl(NewSymbolManagerImpl(), NewTypeMapperImpl(), mockIR)

	expr := &entities.FuncCall{
		Name: "add",
		Args: []entities.Expr{
			&entities.IntLiteral{Value: 3},
			&entities.IntLiteral{Value: 4},
		},
	}
	result, err := ee.EvaluateFuncCall(mockIR, expr)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Errorf("Expected non-nil result")
	}
}

func TestExpressionEvaluatorImpl_EvaluateArrayLiteral(t *testing.T) {
	mockIR := &MockIRManager{}
	ee := NewExpressionEvaluatorImpl(NewSymbolManagerImpl(), NewTypeMapperImpl(), mockIR)

	expr := &entities.ArrayLiteral{
		Elements: []entities.Expr{
			&entities.IntLiteral{Value: 1},
			&entities.IntLiteral{Value: 2},
			&entities.IntLiteral{Value: 3},
		},
	}
	result, err := ee.EvaluateArrayLiteral(mockIR, expr)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Errorf("Expected non-nil result")
	}
}

func TestExpressionEvaluatorImpl_Evaluate_Integration(t *testing.T) {
	mockIR := &MockIRManager{}
	sm := NewSymbolManagerImpl()
	tm := NewTypeMapperImpl()
	ee := NewExpressionEvaluatorImpl(sm, tm, mockIR)

	// 注册变量
	sm.RegisterSymbol("x", "int", "i32")

	tests := []struct {
		name string
		expr entities.Expr
	}{
		{"int literal", &entities.IntLiteral{Value: 42}},
		{"identifier", &entities.Identifier{Name: "x"}},
		{"binary expr", &entities.BinaryExpr{
			Left:  &entities.IntLiteral{Value: 5},
			Op:    "+",
			Right: &entities.IntLiteral{Value: 3},
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ee.Evaluate(mockIR, test.expr)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result == nil {
				t.Errorf("Expected non-nil result")
			}
		})
	}
}

func TestExpressionEvaluatorImpl_Evaluate_UnsupportedExpression(t *testing.T) {
	mockIR := &MockIRManager{}
	ee := NewExpressionEvaluatorImpl(NewSymbolManagerImpl(), NewTypeMapperImpl(), mockIR)

	// 创建一个不支持的表达式类型（假设有一个UnsupportedExpr）
	// 这里我们使用nil来模拟不支持的类型
	var unsupported entities.Expr = nil

	_, err := ee.Evaluate(mockIR, unsupported)
	if err == nil {
		t.Error("Expected error for unsupported expression type")
	}
}

func TestExpressionEvaluatorImpl_EvaluateIdentifier_NotFound(t *testing.T) {
	mockIR := &MockIRManager{}
	ee := NewExpressionEvaluatorImpl(NewSymbolManagerImpl(), NewTypeMapperImpl(), mockIR)

	expr := &entities.Identifier{Name: "nonexistent"}
	_, err := ee.EvaluateIdentifier(mockIR, expr)
	if err == nil {
		t.Error("Expected error for non-existent identifier")
	}
}
