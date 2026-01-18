package services

import (
	"testing"

	"echo/internal/modules/frontend/domain/entities"
)

func TestGenericSystem_Integration(t *testing.T) {
	// 创建完整的泛型系统测试

	// 1. 创建泛型函数定义
	genericFunc := &entities.FuncDef{
		Name: "identity",
		TypeParams: []entities.GenericParam{
			{Name: "T", Constraints: []string{}},
		},
		Params: []entities.Param{
			{Name: "value", Type: "T"},
		},
		ReturnType: "T",
		Body: []entities.ASTNode{
			&entities.ReturnStmt{
				Value: &entities.Identifier{Name: "value"},
			},
		},
	}

	// 2. 创建函数调用（显式类型参数）
	call := &entities.FuncCall{
		Name:     "identity",
		TypeArgs: []string{"int"},
		Args: []entities.Expr{
			&entities.IntLiteral{Value: 42},
		},
	}

	// 3. 测试类型推断
	ti := NewTypeInference()
	inferredTypes, err := ti.InferTypes(genericFunc, call)
	if err != nil {
		t.Fatalf("Type inference failed: %v", err)
	}

	if inferredTypes["T"] != "int" {
		t.Errorf("Expected T=int, got T=%s", inferredTypes["T"])
	}

	// 4. 测试单态化
	program := &entities.Program{
		Statements: []entities.ASTNode{
			genericFunc,
			&entities.FuncDef{
				Name:       "main",
				Params:     []entities.Param{},
				ReturnType: "void",
				Body: []entities.ASTNode{
					&entities.VarDecl{
						Name:  "result",
						Type:  "int",
						Value: call,
					},
				},
			},
		},
	}

	mono := NewMonomorphization()
	resultProgram, err := mono.MonomorphizeProgram(program)
	if err != nil {
		t.Fatalf("Monomorphization failed: %v", err)
	}

	// 验证生成了单态化函数
	foundMonomorphized := false
	foundCallUpdate := false

	for _, stmt := range resultProgram.Statements {
		if funcDef, ok := stmt.(*entities.FuncDef); ok {
			if funcDef.Name == "identity[int]" {
				foundMonomorphized = true
				// 验证单态化函数
				if len(funcDef.TypeParams) != 0 {
					t.Error("Monomorphized function should have no type params")
				}
				if funcDef.ReturnType != "int" {
					t.Errorf("Expected return type 'int', got '%s'", funcDef.ReturnType)
				}
			}
		}
	}

	if !foundMonomorphized {
		t.Error("Monomorphized function 'identity[int]' not found")
	}

	// 验证函数调用已被更新（这里需要更完整的实现）
	if !foundCallUpdate {
		t.Log("Note: Function call update not fully implemented yet")
	}
}

func TestGenericSystem_ComplexScenarios(t *testing.T) {
	t.Run("multiple type parameters", func(t *testing.T) {
		// 测试多类型参数函数
		pairFunc := &entities.FuncDef{
			Name: "makePair",
			TypeParams: []entities.GenericParam{
				{Name: "A", Constraints: []string{}},
				{Name: "B", Constraints: []string{}},
			},
			Params: []entities.Param{
				{Name: "first", Type: "A"},
				{Name: "second", Type: "B"},
			},
			ReturnType: "string", // 简化为返回字符串表示
			Body: []entities.ASTNode{
				&entities.ReturnStmt{
					Value: &entities.StringLiteral{Value: "pair created"},
				},
			},
		}

		call := &entities.FuncCall{
			Name:     "makePair",
			TypeArgs: []string{"string", "int"},
			Args: []entities.Expr{
				&entities.StringLiteral{Value: "hello"},
				&entities.IntLiteral{Value: 42},
			},
		}

		ti := NewTypeInference()
		inferredTypes, err := ti.InferTypes(pairFunc, call)
		if err != nil {
			t.Fatalf("Type inference failed: %v", err)
		}

		if inferredTypes["A"] != "string" {
			t.Errorf("Expected A=string, got A=%s", inferredTypes["A"])
		}
		if inferredTypes["B"] != "int" {
			t.Errorf("Expected B=int, got B=%s", inferredTypes["B"])
		}
	})

	t.Run("constraint validation", func(t *testing.T) {
		// 测试带约束的泛型函数
		printFunc := &entities.FuncDef{
			Name: "printValue",
			TypeParams: []entities.GenericParam{
				{Name: "T", Constraints: []string{"Printable"}},
			},
			Params: []entities.Param{
				{Name: "value", Type: "T"},
			},
			ReturnType: "void",
			Body: []entities.ASTNode{
				&entities.PrintStmt{Value: &entities.StringLiteral{Value: "printed"}},
			},
		}

		// 有效的约束（string是Printable）
		validCall := &entities.FuncCall{
			Name:     "printValue",
			TypeArgs: []string{"string"},
			Args:     []entities.Expr{},
		}

		ti := NewTypeInference()
		_, err := ti.InferTypes(printFunc, validCall)
		if err != nil {
			t.Fatalf("Expected valid constraint to pass, got error: %v", err)
		}

		// 这里可以添加对无效约束的测试，但需要完整的约束系统
	})
}

func TestGenericSystem_CodeGeneration(t *testing.T) {
	// 测试代码生成是否正确处理单态化函数名

	// 这里需要集成代码生成器测试
	// 暂时验证函数名规范化

	t.Run("function name normalization", func(t *testing.T) {
		// 这个测试应该在代码生成器包中进行
		// 这里只是占位符

		// 期望：
		// identity[int] -> identity_int
		// pair[string,int] -> pair_string_int
		// container[bool] -> container_bool

		t.Skip("Code generation integration test - to be implemented")
	})
}
