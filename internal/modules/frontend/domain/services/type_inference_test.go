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
