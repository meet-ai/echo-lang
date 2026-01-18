package services

import (
	"testing"

	"echo/internal/modules/frontend/domain/entities"
)

func TestMonomorphization_Basic(t *testing.T) {
	mono := NewMonomorphization()

	// 创建一个简单的泛型函数定义
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

	// 创建函数调用
	call := &entities.FuncCall{
		Name:     "identity",
		TypeArgs: []string{"int"},
		Args: []entities.Expr{
			&entities.IntLiteral{Value: 42},
		},
	}

	// 创建程序
	program := &entities.Program{
		Statements: []entities.ASTNode{
			genericFunc,
			&entities.FuncDef{
				Name: "main",
				Params: []entities.Param{},
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

	// 执行单态化
	result, err := mono.MonomorphizeProgram(program)
	if err != nil {
		t.Fatalf("Monomorphization failed: %v", err)
	}

	// 验证结果
	if len(result.Statements) < 3 { // 原始泛型函数 + 单态化函数 + main函数
		t.Errorf("Expected at least 3 statements, got %d", len(result.Statements))
	}

	// 检查是否生成了单态化函数
	foundMonomorphized := false
	for _, stmt := range result.Statements {
		if funcDef, ok := stmt.(*entities.FuncDef); ok {
			if funcDef.Name == "identity[int]" {
				foundMonomorphized = true
				// 验证单态化函数没有类型参数
				if len(funcDef.TypeParams) != 0 {
					t.Errorf("Monomorphized function should have no type params, got %d", len(funcDef.TypeParams))
				}
				// 验证返回类型被替换
				if funcDef.ReturnType != "int" {
					t.Errorf("Expected return type 'int', got '%s'", funcDef.ReturnType)
				}
				break
			}
		}
	}

	if !foundMonomorphized {
		t.Error("Monomorphized function 'identity[int]' not found")
	}
}

func TestMonomorphization_MakeMonomorphizationKey(t *testing.T) {
	mono := NewMonomorphization()

	tests := []struct {
		funcName string
		typeArgs map[string]string
		expected string
	}{
		{"identity", map[string]string{"T": "int"}, "identity[int]"},
		{"pair", map[string]string{"T": "string", "U": "int"}, "pair[string,int]"},
		{"container", map[string]string{"T": "bool"}, "container[bool]"},
	}

	for _, test := range tests {
		result := mono.generateMonomorphizedName(test.funcName, test.typeArgs)
		if result != test.expected {
			t.Errorf("makeMonomorphizationKey(%s, %v) = %s, expected %s",
				test.funcName, test.typeArgs, result, test.expected)
		}
	}
}
