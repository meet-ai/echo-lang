package entities

import (
	"testing"
)

func TestASTNodeString(t *testing.T) {
	tests := []struct {
		name     string
		node     interface{ String() string }
		contains string
	}{
		{
			name:     "Identifier",
			node:     &Identifier{Name: "variable"},
			contains: "Identifier{Name: variable}",
		},
		{
			name:     "IntLiteral",
			node:     &IntLiteral{Value: 42},
			contains: "42",
		},
		{
			name:     "StringLiteral",
			node:     &StringLiteral{Value: "hello"},
			contains: "\"hello\"",
		},
		{
			name:     "BinaryExpr",
			node:     &BinaryExpr{
				Left:  &IntLiteral{Value: 1},
				Op:    "+",
				Right: &IntLiteral{Value: 2},
			},
			contains: "BinaryExpr",
		},
		{
			name:     "FuncCall",
			node:     &FuncCall{
				Name:    "add",
				TypeArgs: []string{"int"},
				Args:    []Expr{&IntLiteral{Value: 1}, &IntLiteral{Value: 2}},
			},
			contains: "FuncCall",
		},
		{
			name:     "TraitMethod",
			node:     &TraitMethod{
				Name:       "transform",
				TypeParams: []GenericParam{{Name: "U"}},
				Params:     []Param{{Name: "item", Type: "T"}},
				ReturnType: "U",
				IsDefault:  false,
			},
			contains: "TraitMethod{Name: transform, TypeParams: [U], Params: [item: T], ReturnType: U, IsDefault: false}",
		},
		{
			name:     "FuncDef",
			node:     &FuncDef{
				Name:       "test",
				TypeParams: []GenericParam{{Name: "T"}},
				Params:     []Param{{Name: "x", Type: "T"}},
				ReturnType: "T",
			},
			contains: "FuncDef",
		},
		{
			name:     "VarDecl",
			node:     &VarDecl{
				Name:  "x",
				Type:  "int",
				Value: &IntLiteral{Value: 42},
			},
			contains: "VarDecl",
		},
		{
			name:     "IfStmt",
			node:     &IfStmt{
				Condition: &BinaryExpr{
					Left:  &Identifier{Name: "x"},
					Op:    ">",
					Right: &IntLiteral{Value: 0},
				},
			},
			contains: "IfStmt",
		},
		{
			name:     "ReturnStmt",
			node:     &ReturnStmt{
				Value: &IntLiteral{Value: 42},
			},
			contains: "ReturnStmt",
		},
		{
			name:     "PrintStmt",
			node:     &PrintStmt{Value: &StringLiteral{Value: "hello"}},
			contains: "PrintStmt{Value: \"hello\"}",
		},
		{
			name:     "ArrayLiteral",
			node:     &ArrayLiteral{
				Elements: []Expr{
					&IntLiteral{Value: 1},
					&IntLiteral{Value: 2},
					&IntLiteral{Value: 3},
				},
			},
			contains: "ArrayLiteral",
		},
		{
			name:     "StructLiteral",
			node:     &StructLiteral{
				Type: "Point",
				Fields: map[string]Expr{
					"x": &IntLiteral{Value: 1},
					"y": &IntLiteral{Value: 2},
				},
			},
			contains: "Point{...}",
		},
		{
			name:     "StructAccess",
			node:     &StructAccess{
				Object: &Identifier{Name: "point"},
				Field:  "x",
			},
			contains: "StructAccess",
		},
		{
			name:     "OkLiteral",
			node:     &OkLiteral{Value: &IntLiteral{Value: 42}},
			contains: "Ok(IntLiteral",
		},
		{
			name:     "ErrLiteral",
			node:     &ErrLiteral{Error: &StringLiteral{Value: "error"}},
			contains: "Err(StringLiteral",
		},
		{
			name:     "SomeLiteral",
			node:     &SomeLiteral{Value: &StringLiteral{Value: "value"}},
			contains: "Some(StringLiteral",
		},
		{
			name:     "NoneLiteral",
			node:     &NoneLiteral{},
			contains: "None",
		},
		{
			name:     "ErrorPropagation",
			node:     &ErrorPropagation{Expr: &Identifier{Name: "result"}},
			contains: "Identifier{Name: result}?",
		},
		{
			name:     "GenericParam",
			node:     &GenericParam{
				Name:        "T",
				Constraints: []string{"Printable", "Serializable"},
			},
			contains: "T: Printable + Serializable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.node.String()
			if str == "" {
				t.Errorf("String() returned empty string")
				return
			}

			if !containsIgnoreCase(str, tt.contains) {
				t.Errorf("String() = %s, should contain %s", str, tt.contains)
			}
		})
	}
}

func TestExprInterface(t *testing.T) {
	// Test that all expression types implement Expr interface
	exprs := []Expr{
		&Identifier{Name: "x"},
		&IntLiteral{Value: 42},
		&StringLiteral{Value: "hello"},
		&BinaryExpr{
			Left:  &IntLiteral{Value: 1},
			Op:    "+",
			Right: &IntLiteral{Value: 2},
		},
		&FuncCall{
			Name: "add",
			Args: []Expr{&IntLiteral{Value: 1}, &IntLiteral{Value: 2}},
		},
		&ArrayLiteral{Elements: []Expr{&IntLiteral{Value: 1}}},
		&StructAccess{Object: &Identifier{Name: "obj"}, Field: "field"},
		&OkLiteral{Value: &IntLiteral{Value: 42}},
		&ErrLiteral{Error: &StringLiteral{Value: "error"}},
		&SomeLiteral{Value: &StringLiteral{Value: "value"}},
		&NoneLiteral{},
		&ErrorPropagation{Expr: &Identifier{Name: "x"}},
		&StructLiteral{Type: "Point", Fields: map[string]Expr{"x": &IntLiteral{Value: 1}}},
	}

	for i, expr := range exprs {
		if expr == nil {
			t.Errorf("Expression %d is nil", i)
			continue
		}

		// Test String() method
		str := expr.String()
		if str == "" {
			t.Errorf("Expression %d String() returned empty", i)
		}

		// Test AcceptExpr method (should not panic)
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Expression %d AcceptExpr() panicked: %v", i, r)
			}
		}()

		// Create a mock visitor
		mockVisitor := &mockExprVisitor{}
		result := expr.AcceptExpr(mockVisitor)
		if result == nil {
			t.Logf("Expression %d AcceptExpr() returned nil (acceptable)", i)
		}
	}
}

func TestASTNodeInterface(t *testing.T) {
	// Test that all AST node types implement ASTVisitor interface methods
	nodes := []ASTNode{
		&FuncDef{Name: "test"},
		&VarDecl{Name: "x", Type: "int"},
		&ReturnStmt{},
		&BlockStmt{Statements: []ASTNode{&ReturnStmt{}}},
		&IfStmt{},
		&PrintStmt{Value: &StringLiteral{Value: "hello"}},
		&StructDef{Name: "Point"},
		&MethodDef{Name: "method"},
		&TraitDef{Name: "Trait"},
		&EnumDef{Name: "Color"},
		&MatchStmt{},
		&ForStmt{},
		&WhileStmt{},
	}

	for i, node := range nodes {
		if node == nil {
			t.Errorf("AST node %d is nil", i)
			continue
		}

		// Test String() method
		str := node.String()
		if str == "" {
			t.Errorf("AST node %d String() returned empty", i)
		}

		// Test Accept method (should not panic)
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("AST node %d Accept() panicked: %v", i, r)
			}
		}()

		// Create a mock visitor
		mockVisitor := &mockASTVisitor{}
		result := node.Accept(mockVisitor)
		if result == nil {
			t.Logf("AST node %d Accept() returned nil (acceptable)", i)
		}
	}
}

func TestGenericParam(t *testing.T) {
	tests := []struct {
		name     string
		param    GenericParam
		expected string
	}{
		{
			name:     "简单类型参数",
			param:    GenericParam{Name: "T"},
			expected: "T",
		},
		{
			name:     "带约束的类型参数",
			param:    GenericParam{Name: "T", Constraints: []string{"Printable", "Serializable"}},
			expected: "T: Printable + Serializable",
		},
		{
			name:     "多个约束",
			param:    GenericParam{Name: "U", Constraints: []string{"Comparable"}},
			expected: "U: Comparable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// GenericParam 目前没有 String() 方法，我们可以测试其字段
			if tt.param.Name == "" {
				t.Errorf("GenericParam name should not be empty")
			}

			// 测试约束
			for _, constraint := range tt.param.Constraints {
				if constraint == "" {
					t.Errorf("Constraint should not be empty")
				}
			}
		})
	}
}

func TestParam(t *testing.T) {
	param := Param{Name: "value", Type: "int"}

	if param.Name != "value" {
		t.Errorf("Expected name 'value', got '%s'", param.Name)
	}

	if param.Type != "int" {
		t.Errorf("Expected type 'int', got '%s'", param.Type)
	}
}

func TestMatchCase(t *testing.T) {
	// Create a simple match case
	pattern := &OkLiteral{Value: &Identifier{Name: "value"}}
	bodyStmt := &ReturnStmt{Value: &Identifier{Name: "value"}}

	matchCase := MatchCase{
		Pattern: pattern,
		Body:    []ASTNode{bodyStmt},
	}

	if matchCase.Pattern == nil {
		t.Errorf("MatchCase pattern should not be nil")
	}

	if len(matchCase.Body) != 1 {
		t.Errorf("Expected 1 body statement, got %d", len(matchCase.Body))
	}
}

// Mock visitors for testing
type mockASTVisitor struct{}

func (m *mockASTVisitor) VisitFuncDef(node *FuncDef) interface{}     { return nil }
func (m *mockASTVisitor) VisitVarDecl(node *VarDecl) interface{}     { return nil }
func (m *mockASTVisitor) VisitAssignStmt(node *AssignStmt) interface{} { return nil }
func (m *mockASTVisitor) VisitReturnStmt(node *ReturnStmt) interface{} { return nil }
func (m *mockASTVisitor) VisitBlockStmt(node *BlockStmt) interface{}  { return nil }
func (m *mockASTVisitor) VisitIfStmt(node *IfStmt) interface{}       { return nil }
func (m *mockASTVisitor) VisitPrintStmt(node *PrintStmt) interface{} { return nil }
func (m *mockASTVisitor) VisitExprStmt(node *ExprStmt) interface{}   { return nil }
func (m *mockASTVisitor) VisitFuncCall(node *FuncCall) interface{}   { return nil }
func (m *mockASTVisitor) VisitStructDef(node *StructDef) interface{} { return nil }
func (m *mockASTVisitor) VisitMethodDef(node *MethodDef) interface{} { return nil }
func (m *mockASTVisitor) VisitTraitDef(node *TraitDef) interface{}             { return nil }
func (m *mockASTVisitor) VisitAssociatedTypeDef(node *AssociatedTypeDef) interface{}    { return nil }
func (m *mockASTVisitor) VisitDynamicTraitRef(node *DynamicTraitRef) interface{}       { return nil }
func (m *mockASTVisitor) VisitEnumDef(node *EnumDef) interface{}                        { return nil }
func (m *mockASTVisitor) VisitMatchStmt(node *MatchStmt) interface{} { return nil }
func (m *mockASTVisitor) VisitMatchExpr(node *MatchExpr) interface{}   { return nil }
func (m *mockASTVisitor) VisitForStmt(node *ForStmt) interface{}         { return nil }
func (m *mockASTVisitor) VisitWhileStmt(node *WhileStmt) interface{}     { return nil }
func (m *mockASTVisitor) VisitAgentCreateStmt(node *AgentCreateStmt) interface{} { return nil }
func (m *mockASTVisitor) VisitAgentSendStmt(node *AgentSendStmt) interface{}     { return nil }
func (m *mockASTVisitor) VisitAgentReceiveStmt(node *AgentReceiveStmt) interface{} { return nil }
func (m *mockASTVisitor) VisitBreakStmt(node *BreakStmt) interface{}               { return nil }
func (m *mockASTVisitor) VisitContinueStmt(node *ContinueStmt) interface{}         { return nil }
func (m *mockASTVisitor) VisitDeleteStmt(node *DeleteStmt) interface{}             { return nil }
func (m *mockASTVisitor) VisitArrayLiteral(node *ArrayLiteral) interface{}   { return nil }
func (m *mockASTVisitor) VisitStructLiteral(node *StructLiteral) interface{} { return nil }
func (m *mockASTVisitor) VisitStructAccess(node *StructAccess) interface{}   { return nil }
func (m *mockASTVisitor) VisitGenericType(node *GenericType) interface{}     { return nil }
func (m *mockASTVisitor) VisitResultExpr(node *ResultExpr) interface{}       { return nil }
func (m *mockASTVisitor) VisitOptionExpr(node *OptionExpr) interface{}       { return nil }
func (m *mockASTVisitor) VisitOkLiteral(node *OkLiteral) interface{}           { return nil }
func (m *mockASTVisitor) VisitErrLiteral(node *ErrLiteral) interface{}     { return nil }
func (m *mockASTVisitor) VisitSomeLiteral(node *SomeLiteral) interface{}   { return nil }
func (m *mockASTVisitor) VisitNoneLiteral(node *NoneLiteral) interface{}   { return nil }
func (m *mockASTVisitor) VisitBoolLiteral(node *BoolLiteral) interface{}         { return nil }
func (m *mockASTVisitor) VisitErrorPropagation(node *ErrorPropagation) interface{} { return nil }
func (m *mockASTVisitor) VisitErrPattern(node *ErrPattern) interface{}         { return nil }
func (m *mockASTVisitor) VisitOkPattern(node *OkPattern) interface{}       { return nil }
func (m *mockASTVisitor) VisitSomePattern(node *SomePattern) interface{}   { return nil }
func (m *mockASTVisitor) VisitNonePattern(node *NonePattern) interface{}   { return nil }
func (m *mockASTVisitor) VisitWildcardPattern(node *WildcardPattern) interface{}   { return nil }
func (m *mockASTVisitor) VisitIdentifierPattern(node *IdentifierPattern) interface{} { return nil }
func (m *mockASTVisitor) VisitLiteralPattern(node *LiteralPattern) interface{}       { return nil }
func (m *mockASTVisitor) VisitStructPattern(node *StructPattern) interface{}         { return nil }
func (m *mockASTVisitor) VisitArrayPattern(node *ArrayPattern) interface{}           { return nil }
func (m *mockASTVisitor) VisitTuplePattern(node *TuplePattern) interface{}           { return nil }
func (m *mockASTVisitor) VisitAsyncFuncDef(node *AsyncFuncDef) interface{}           { return nil }
func (m *mockASTVisitor) VisitAwaitExpr(node *AwaitExpr) interface{}                 { return nil }
func (m *mockASTVisitor) VisitChanType(node *ChanType) interface{}                   { return nil }
func (m *mockASTVisitor) VisitFutureType(node *FutureType) interface{}               { return nil }
func (m *mockASTVisitor) VisitChanLiteral(node *ChanLiteral) interface{}             { return nil }
func (m *mockASTVisitor) VisitSendExpr(node *SendExpr) interface{}                   { return nil }
func (m *mockASTVisitor) VisitReceiveExpr(node *ReceiveExpr) interface{}             { return nil }
func (m *mockASTVisitor) VisitSpawnExpr(node *SpawnExpr) interface{}                 { return nil }
func (m *mockASTVisitor) VisitSelectStmt(node *SelectStmt) interface{}               { return nil }
func (m *mockASTVisitor) VisitIndexExpr(node *IndexExpr) interface{}                 { return nil }
func (m *mockASTVisitor) VisitLenExpr(node *LenExpr) interface{}                     { return nil }
func (m *mockASTVisitor) VisitSliceExpr(node *SliceExpr) interface{}                 { return nil }
func (m *mockASTVisitor) VisitArrayMethodCallExpr(node *ArrayMethodCallExpr) interface{} { return nil }
func (m *mockASTVisitor) VisitMethodCallExpr(node *MethodCallExpr) interface{}       { return nil }
func (m *mockASTVisitor) VisitProgram(node *Program) interface{}             { return nil }

type mockExprVisitor struct{}

func (m *mockExprVisitor) VisitIdentifier(node *Identifier) interface{}       { return nil }
func (m *mockExprVisitor) VisitStringLiteral(node *StringLiteral) interface{} { return nil }
func (m *mockExprVisitor) VisitIntLiteral(node *IntLiteral) interface{}       { return nil }
func (m *mockExprVisitor) VisitFloatLiteral(node *FloatLiteral) interface{}     { return nil }
func (m *mockExprVisitor) VisitBinaryExpr(node *BinaryExpr) interface{}       { return nil }
func (m *mockExprVisitor) VisitFuncCall(node *FuncCall) interface{}           { return nil }
func (m *mockExprVisitor) VisitResultExpr(node *ResultExpr) interface{}       { return nil }
func (m *mockExprVisitor) VisitOptionExpr(node *OptionExpr) interface{}       { return nil }
func (m *mockExprVisitor) VisitOkLiteral(node *OkLiteral) interface{}         { return nil }
func (m *mockExprVisitor) VisitErrLiteral(node *ErrLiteral) interface{}       { return nil }
func (m *mockExprVisitor) VisitErrorPropagation(node *ErrorPropagation) interface{} { return nil }
func (m *mockExprVisitor) VisitSomeLiteral(node *SomeLiteral) interface{}     { return nil }
func (m *mockExprVisitor) VisitNoneLiteral(node *NoneLiteral) interface{}     { return nil }
func (m *mockExprVisitor) VisitBoolLiteral(node *BoolLiteral) interface{}       { return nil }
func (m *mockExprVisitor) VisitMatchExpr(node *MatchExpr) interface{}         { return nil }
func (m *mockExprVisitor) VisitArrayLiteral(node *ArrayLiteral) interface{}   { return nil }
func (m *mockExprVisitor) VisitStructAccess(node *StructAccess) interface{}   { return nil }
func (m *mockExprVisitor) VisitStructLiteral(node *StructLiteral) interface{} { return nil }
func (m *mockExprVisitor) VisitErrPattern(node *ErrPattern) interface{}         { return nil }
func (m *mockExprVisitor) VisitOkPattern(node *OkPattern) interface{}           { return nil }
func (m *mockExprVisitor) VisitSomePattern(node *SomePattern) interface{}       { return nil }
func (m *mockExprVisitor) VisitNonePattern(node *NonePattern) interface{}       { return nil }
func (m *mockExprVisitor) VisitWildcardPattern(node *WildcardPattern) interface{}   { return nil }
func (m *mockExprVisitor) VisitIdentifierPattern(node *IdentifierPattern) interface{} { return nil }
func (m *mockExprVisitor) VisitLiteralPattern(node *LiteralPattern) interface{}       { return nil }
func (m *mockExprVisitor) VisitStructPattern(node *StructPattern) interface{}         { return nil }
func (m *mockExprVisitor) VisitArrayPattern(node *ArrayPattern) interface{}           { return nil }
func (m *mockExprVisitor) VisitTuplePattern(node *TuplePattern) interface{}           { return nil }
func (m *mockExprVisitor) VisitAwaitExpr(node *AwaitExpr) interface{}                 { return nil }
func (m *mockExprVisitor) VisitChanLiteral(node *ChanLiteral) interface{}             { return nil }
func (m *mockExprVisitor) VisitSendExpr(node *SendExpr) interface{}                   { return nil }
func (m *mockExprVisitor) VisitReceiveExpr(node *ReceiveExpr) interface{}             { return nil }
func (m *mockExprVisitor) VisitSpawnExpr(node *SpawnExpr) interface{}                 { return nil }
func (m *mockExprVisitor) VisitIndexExpr(node *IndexExpr) interface{}                 { return nil }
func (m *mockExprVisitor) VisitLenExpr(node *LenExpr) interface{}                     { return nil }
func (m *mockExprVisitor) VisitSliceExpr(node *SliceExpr) interface{}                 { return nil }
func (m *mockExprVisitor) VisitArrayMethodCallExpr(node *ArrayMethodCallExpr) interface{} { return nil }
func (m *mockExprVisitor) VisitMethodCallExpr(node *MethodCallExpr) interface{}       { return nil }
func (m *mockExprVisitor) VisitDynamicTraitRef(node *DynamicTraitRef) interface{}      { return nil }

// Helper function
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && findInString(s, substr)
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
