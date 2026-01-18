package services

import (
	"testing"

	"echo/internal/modules/frontend/domain/entities"
)

func TestTypeInference_InferTypes(t *testing.T) {
	ti := NewTypeInference()

	// 创建泛型函数定义
	funcDef := &entities.FuncDef{
		Name: "identity",
		TypeParams: []entities.GenericParam{
			{Name: "T", Constraints: []string{}},
		},
		Params: []entities.Param{
			{Name: "value", Type: "T"},
		},
		ReturnType: "T",
	}

	t.Run("explicit type arguments", func(t *testing.T) {
		// identity[int](42)
		call := &entities.FuncCall{
			Name:     "identity",
			TypeArgs: []string{"int"},
			Args: []entities.Expr{
				&entities.IntLiteral{Value: 42},
			},
		}

		result, err := ti.InferTypes(funcDef, call)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if result["T"] != "int" {
			t.Errorf("Expected T=int, got T=%s", result["T"])
		}
	})

	t.Run("constraint validation", func(t *testing.T) {
		// 创建带约束的函数
		constrainedFunc := &entities.FuncDef{
			Name: "printValue",
			TypeParams: []entities.GenericParam{
				{Name: "T", Constraints: []string{"Printable"}},
			},
			Params: []entities.Param{
				{Name: "value", Type: "T"},
			},
			ReturnType: "void",
		}

		// 测试有效类型
		call := &entities.FuncCall{
			Name:     "printValue",
			TypeArgs: []string{"string"},
			Args:     []entities.Expr{},
		}

		_, err := ti.InferTypes(constrainedFunc, call)
		if err != nil {
			t.Fatalf("Expected no error for valid constraint, got %v", err)
		}
	})

	t.Run("invalid type argument count", func(t *testing.T) {
		call := &entities.FuncCall{
			Name:     "identity",
			TypeArgs: []string{"int", "string"}, // 太多类型参数
			Args:     []entities.Expr{},
		}

		_, err := ti.InferTypes(funcDef, call)
		if err == nil {
			t.Fatal("Expected error for invalid type argument count")
		}
	})
}

func TestTypeInference_InferFromReturnType(t *testing.T) {
	ti := NewTypeInference()

	funcDef := &entities.FuncDef{
		Name: "identity",
		TypeParams: []entities.GenericParam{
			{Name: "T", Constraints: []string{}},
		},
		Params:      []entities.Param{},
		ReturnType:  "T",
	}

	call := &entities.FuncCall{
		Name:     "identity",
		TypeArgs: []string{},
		Args:     []entities.Expr{},
	}

	// 从返回值推断：let x: string = identity()
	result, err := ti.InferFromReturnType("string", funcDef, call)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result["T"] != "string" {
		t.Errorf("Expected T=string, got T=%s", result["T"])
	}
}

func TestTypeInference_SatisfiesConstraint(t *testing.T) {
	ti := NewTypeInference()

	tests := []struct {
		actualType string
		constraint string
		expected   bool
	}{
		{"int", "Printable", true},
		{"string", "Printable", true},
		{"float", "Printable", false},
		{"int", "Comparable", true},
		{"string", "Comparable", true},
		{"bool", "Comparable", false},
	}

	for _, test := range tests {
		result := ti.SatisfiesConstraint(test.actualType, test.constraint)
		if result != test.expected {
			t.Errorf("SatisfiesConstraint(%s, %s) = %v, expected %v",
				test.actualType, test.constraint, result, test.expected)
		}
	}
}
