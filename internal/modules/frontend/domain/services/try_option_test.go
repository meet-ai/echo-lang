package services

import (
	"strings"
	"testing"

	"github.com/meetai/echo-lang/internal/modules/backend/domain/services"
	"github.com/meetai/echo-lang/internal/modules/frontend/domain/entities"
)

func TestTryOptionParsing(t *testing.T) {
	parser := NewSimpleParser()

	tests := []struct {
		name     string
		input    string
		wantErr  bool
		checkAST func(t *testing.T, program *entities.Program)
	}{
		{
			name: "Result类型函数定义",
			input: `func divide(a: int, b: int) -> Result[int] {
				return Ok(a / b)
			}`,
			wantErr: false,
			checkAST: func(t *testing.T, program *entities.Program) {
				if len(program.Statements) != 1 {
					t.Errorf("Expected 1 statement, got %d", len(program.Statements))
					return
				}

				funcDef, ok := program.Statements[0].(*entities.FuncDef)
				if !ok {
					t.Errorf("Expected FuncDef, got %T", program.Statements[0])
					return
				}

				if funcDef.ReturnType != "Result[int]" {
					t.Errorf("Expected return type 'Result[int]', got '%s'", funcDef.ReturnType)
				}
			},
		},
		{
			name: "Option类型函数定义",
			input: `func findUser(id: int) -> Option[string] {
				return Some("Alice")
			}`,
			wantErr: false,
			checkAST: func(t *testing.T, program *entities.Program) {
				if len(program.Statements) != 1 {
					t.Errorf("Expected 1 statement, got %d", len(program.Statements))
					return
				}

				funcDef, ok := program.Statements[0].(*entities.FuncDef)
				if !ok {
					t.Errorf("Expected FuncDef, got %T", program.Statements[0])
					return
				}

				if funcDef.ReturnType != "Option[string]" {
					t.Errorf("Expected return type 'Option[string]', got '%s'", funcDef.ReturnType)
				}
			},
		},
		{
			name: "Ok字面量",
			input: `func test() -> int {
				let result: Result[int] = Ok(42)
				return 42
			}`,
			wantErr: false,
		},
		{
			name: "Err字面量",
			input: `func test() -> int {
				let result: Result[string] = Err("error")
				return 42
			}`,
			wantErr: false,
		},
		{
			name: "Some字面量",
			input: `func test() -> int {
				let result: Option[string] = Some("value")
				return 42
			}`,
			wantErr: false,
		},
		{
			name: "None字面量",
			input: `func test() -> int {
				let result: Option[int] = None
				return 42
			}`,
			wantErr: false,
		},
		{
			name: "错误传播操作符",
			input: `func test() -> Result[int] {
				return divide(10, 2)?
			}`,
			wantErr: false,
		},
		{
			name: "Result模式匹配",
			input: `func test() -> int {
				let result: Result[int] = divide(10, 2)
				match result {
					Ok(value) => return value
					Err(error) => return -1
				}
			}`,
			wantErr: false,
		},
		{
			name: "Option模式匹配",
			input: `func test() -> int {
				let option: Option[string] = findUser(1)
				match option {
					Some(name) => return 1
					None => return 0
				}
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, err := parser.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkAST != nil {
				tt.checkAST(t, program)
			}
		})
	}
}

func TestTryOptionCodeGeneration(t *testing.T) {
	parser := NewSimpleParser()
	generator := services.NewOCamlGenerator()

	tests := []struct {
		name         string
		input        string
		checkOutput  func(t *testing.T, output string)
	}{
		{
			name: "Result类型映射",
			input: `func test() -> Result[int] {
				return Ok(42)
			}`,
			checkOutput: func(t *testing.T, output string) {
				// 检查是否生成了Ok字面量
				if !strings.Contains(output, "Ok (42)") {
					t.Errorf("Expected Ok literal, but got: %s", output)
				}
			},
		},
		{
			name: "Option类型映射",
			input: `func test() -> Option[string] {
				return Some("hello")
			}`,
			checkOutput: func(t *testing.T, output string) {
				// 检查是否生成了Some字面量
				if !strings.Contains(output, "Some (\"hello\")") {
					t.Errorf("Expected Some literal, but got: %s", output)
				}
			},
		},
		{
			name: "Result模式匹配",
			input: `func test() -> int {
				let result: Result[int] = divide(10, 2)
				match result {
					Ok(value) => return value
					Err(error) => return -1
				}
			}`,
			checkOutput: func(t *testing.T, output string) {
				// 检查是否生成正确的match表达式
				if !strings.Contains(output, "match result with Ok (value) ->") {
					t.Errorf("Expected match expression for Result, but got: %s", output)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			output := generator.GenerateOCaml(program)
			tt.checkOutput(t, output)
		})
	}
}
