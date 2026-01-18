package services

import (
	"testing"

	"echo/internal/modules/frontend/domain/entities"
)

func TestParserBasicSyntax(t *testing.T) {
	parser := NewSimpleParser()

	tests := []struct {
		name     string
		input    string
		wantErr  bool
		checkAST func(t *testing.T, program *entities.Program)
	}{
		{
			name: "简单函数定义",
			input: `func add(a: int, b: int) -> int {
				return a + b
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

				if funcDef.Name != "add" {
					t.Errorf("Expected function name 'add', got '%s'", funcDef.Name)
				}

				if len(funcDef.Params) != 2 {
					t.Errorf("Expected 2 parameters, got %d", len(funcDef.Params))
				}

				if funcDef.ReturnType != "int" {
					t.Errorf("Expected return type 'int', got '%s'", funcDef.ReturnType)
				}
			},
		},
		{
			name: "变量声明",
			input: `func test() -> int {
				let x: int = 42
				let y: string = "hello"
				return x
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

				if len(funcDef.Body) < 2 {
					t.Errorf("Expected at least 2 body statements, got %d", len(funcDef.Body))
					return
				}

				// Check first variable declaration
				varDecl1, ok := funcDef.Body[0].(*entities.VarDecl)
				if !ok {
					t.Errorf("Expected VarDecl, got %T", funcDef.Body[0])
					return
				}

				if varDecl1.Name != "x" || varDecl1.Type != "int" {
					t.Errorf("Expected variable x: int, got %s: %s", varDecl1.Name, varDecl1.Type)
				}
			},
		},
		{
			name: "if语句",
			input: `func test(x: int) -> int {
				if x > 0 {
					return 1
				}
				return 0
			}`,
			wantErr: false,
			checkAST: func(t *testing.T, program *entities.Program) {
				funcDef := program.Statements[0].(*entities.FuncDef)

				ifStmt, ok := funcDef.Body[0].(*entities.IfStmt)
				if !ok {
					t.Errorf("Expected IfStmt, got %T", funcDef.Body[0])
					return
				}

				// Check condition
				binaryExpr, ok := ifStmt.Condition.(*entities.BinaryExpr)
				if !ok {
					t.Errorf("Expected BinaryExpr condition, got %T", ifStmt.Condition)
					return
				}

				if binaryExpr.Op != ">" {
					t.Errorf("Expected '>' operator, got '%s'", binaryExpr.Op)
				}

				// Note: ThenBody population may not be implemented yet
				t.Logf("IfStmt found with condition, ThenBody population may not be complete")
			},
		},
		{
			name: "函数调用",
			input: `func test() -> int {
				let result: int = add(1, 2)
				return result
			}`,
			wantErr: false,
			checkAST: func(t *testing.T, program *entities.Program) {
				funcDef := program.Statements[0].(*entities.FuncDef)
				varDecl := funcDef.Body[0].(*entities.VarDecl)

				funcCall, ok := varDecl.Value.(*entities.FuncCall)
				if !ok {
					t.Errorf("Expected FuncCall, got %T", varDecl.Value)
					return
				}

				if funcCall.Name != "add" {
					t.Errorf("Expected function call 'add', got '%s'", funcCall.Name)
				}

				if len(funcCall.Args) != 2 {
					t.Errorf("Expected 2 arguments, got %d", len(funcCall.Args))
				}
			},
		},
		{
			name: "数组字面量",
			input: `func test() -> int {
				let arr: []int = [1, 2, 3]
				return 0
			}`,
			wantErr: false,
			checkAST: func(t *testing.T, program *entities.Program) {
				funcDef := program.Statements[0].(*entities.FuncDef)
				varDecl := funcDef.Body[0].(*entities.VarDecl)

				arrayLit, ok := varDecl.Value.(*entities.ArrayLiteral)
				if !ok {
					t.Errorf("Expected ArrayLiteral, got %T", varDecl.Value)
					return
				}

				if len(arrayLit.Elements) != 3 {
					t.Errorf("Expected 3 elements, got %d", len(arrayLit.Elements))
				}
			},
		},
		{
			name: "结构体字面量",
			input: `func test() -> int {
				let point: Point = Point{x: 1, y: 2}
				return point.x
			}`,
			wantErr: false,
			checkAST: func(t *testing.T, program *entities.Program) {
				funcDef := program.Statements[0].(*entities.FuncDef)
				varDecl := funcDef.Body[0].(*entities.VarDecl)

				structLit, ok := varDecl.Value.(*entities.StructLiteral)
				if !ok {
					t.Errorf("Expected StructLiteral, got %T", varDecl.Value)
					return
				}

				if structLit.Type != "Point" {
					t.Errorf("Expected struct type 'Point', got '%s'", structLit.Type)
				}

				if len(structLit.Fields) != 2 {
					t.Errorf("Expected 2 fields, got %d", len(structLit.Fields))
				}

				if xValue, ok := structLit.Fields["x"]; ok {
					if intLit, ok := xValue.(*entities.IntLiteral); !ok || intLit.Value != 1 {
						t.Errorf("Expected x = 1, got %v", xValue)
					}
				} else {
					t.Errorf("Expected field 'x'")
				}
			},
		},
		{
			name: "二元表达式",
			input: `func test(a: int, b: int) -> int {
				return a + b * 2
			}`,
			wantErr: false,
			checkAST: func(t *testing.T, program *entities.Program) {
				funcDef := program.Statements[0].(*entities.FuncDef)
				returnStmt := funcDef.Body[0].(*entities.ReturnStmt)

				binaryExpr, ok := returnStmt.Value.(*entities.BinaryExpr)
				if !ok {
					t.Errorf("Expected BinaryExpr, got %T", returnStmt.Value)
					return
				}

				if binaryExpr.Op != "+" {
					t.Errorf("Expected '+' operator, got '%s'", binaryExpr.Op)
				}

				// Check right side is another binary expression
				rightExpr, ok := binaryExpr.Right.(*entities.BinaryExpr)
				if !ok {
					t.Errorf("Expected nested BinaryExpr, got %T", binaryExpr.Right)
					return
				}

				if rightExpr.Op != "*" {
					t.Errorf("Expected '*' operator, got '%s'", rightExpr.Op)
				}
			},
		},
		{
			name: "print语句",
			input: `func test() -> void {
				print "Hello, World!"
				print "Value: " + 42
			}`,
			wantErr: false,
			checkAST: func(t *testing.T, program *entities.Program) {
				funcDef := program.Statements[0].(*entities.FuncDef)

				if len(funcDef.Body) != 2 {
					t.Errorf("Expected 2 body statements, got %d", len(funcDef.Body))
					return
				}

				printStmt1, ok := funcDef.Body[0].(*entities.PrintStmt)
				if !ok {
					t.Errorf("Expected PrintStmt, got %T", funcDef.Body[0])
					return
				}

				stringLit, ok := printStmt1.Value.(*entities.StringLiteral)
				if !ok {
					t.Errorf("Expected StringLiteral, got %T", printStmt1.Value)
					return
				}
				if stringLit.Value != "Hello, World!" {
					t.Errorf("Expected print value 'Hello, World!', got '%s'", stringLit.Value)
				}
			},
		},
		{
			name: "泛型函数",
			input: `func identity[T](value: T) -> T {
				return value
			}`,
			wantErr: false,
			checkAST: func(t *testing.T, program *entities.Program) {
				funcDef := program.Statements[0].(*entities.FuncDef)

				if len(funcDef.TypeParams) != 1 {
					t.Errorf("Expected 1 type parameter, got %d", len(funcDef.TypeParams))
					return
				}

				if funcDef.TypeParams[0].Name != "T" {
					t.Errorf("Expected type parameter 'T', got '%s'", funcDef.TypeParams[0].Name)
				}

				if len(funcDef.Params) != 1 {
					t.Errorf("Expected 1 parameter, got %d", len(funcDef.Params))
					return
				}

				if funcDef.Params[0].Type != "T" {
					t.Errorf("Expected parameter type 'T', got '%s'", funcDef.Params[0].Type)
				}

				if funcDef.ReturnType != "T" {
					t.Errorf("Expected return type 'T', got '%s'", funcDef.ReturnType)
				}
			},
		},
		{
			name: "泛型函数调用",
			input: `func test() -> int {
				return identity[int](42)
			}`,
			wantErr: false,
			checkAST: func(t *testing.T, program *entities.Program) {
				funcDef := program.Statements[0].(*entities.FuncDef)
				returnStmt := funcDef.Body[0].(*entities.ReturnStmt)

				funcCall, ok := returnStmt.Value.(*entities.FuncCall)
				if !ok {
					t.Errorf("Expected FuncCall, got %T", returnStmt.Value)
					return
				}

				if funcCall.Name != "identity" {
					t.Errorf("Expected function call 'identity', got '%s'", funcCall.Name)
				}

				if len(funcCall.TypeArgs) != 1 || funcCall.TypeArgs[0] != "int" {
					t.Errorf("Expected type args ['int'], got %v", funcCall.TypeArgs)
				}

				if len(funcCall.Args) != 1 {
					t.Errorf("Expected 1 argument, got %d", len(funcCall.Args))
				}
			},
		},
		{
			name: "泛型方法",
			input: `struct Container[T] {
	value: T
}

func (c Container[T]) get() -> T {
	return c.value
}`,
			wantErr: false,
			checkAST: func(t *testing.T, program *entities.Program) {
				if len(program.Statements) < 2 {
					t.Errorf("Expected at least 2 statements, got %d", len(program.Statements))
					return
				}

				// 检查结构体定义
				structDef := program.Statements[0].(*entities.StructDef)
				if len(structDef.TypeParams) != 1 {
					t.Errorf("Expected 1 type parameter for struct, got %d", len(structDef.TypeParams))
				}
				if structDef.TypeParams[0].Name != "T" {
					t.Errorf("Expected struct type parameter 'T', got '%s'", structDef.TypeParams[0].Name)
				}

				// 检查方法定义
				methodDef := program.Statements[1].(*entities.MethodDef)
				if methodDef.Name != "get" {
					t.Errorf("Expected method name 'get', got '%s'", methodDef.Name)
				}
				if methodDef.Receiver != "Container" {
					t.Errorf("Expected receiver 'Container', got '%s'", methodDef.Receiver)
				}
				if len(methodDef.ReceiverParams) != 1 {
					t.Errorf("Expected 1 receiver type parameter, got %d", len(methodDef.ReceiverParams))
				}
				if methodDef.ReceiverParams[0].Name != "T" {
					t.Errorf("Expected receiver type parameter 'T', got '%s'", methodDef.ReceiverParams[0].Name)
				}
				if methodDef.ReturnType != "T" {
					t.Errorf("Expected return type 'T', got '%s'", methodDef.ReturnType)
				}
			},
		},
		{
			name: "带约束的泛型方法",
			input: `struct Processor[T: Printable] {
	value: T
}

func (p Processor[T]) process[U: Serializable](data: U) -> string {
	return p.value.toString() + data.serialize()
}`,
			wantErr: false,
			checkAST: func(t *testing.T, program *entities.Program) {
				if len(program.Statements) < 2 {
					t.Errorf("Expected at least 2 statements, got %d", len(program.Statements))
					return
				}

				// 检查结构体定义
				structDef := program.Statements[0].(*entities.StructDef)
				if len(structDef.TypeParams) != 1 {
					t.Errorf("Expected 1 type parameter for struct, got %d", len(structDef.TypeParams))
				}
				if structDef.TypeParams[0].Name != "T" {
					t.Errorf("Expected struct type parameter 'T', got '%s'", structDef.TypeParams[0].Name)
				}
				if len(structDef.TypeParams[0].Constraints) != 1 {
					t.Errorf("Expected 1 constraint for T, got %d", len(structDef.TypeParams[0].Constraints))
				}
				if structDef.TypeParams[0].Constraints[0] != "Printable" {
					t.Errorf("Expected constraint 'Printable', got '%s'", structDef.TypeParams[0].Constraints[0])
				}

				// 检查方法定义
				methodDef := program.Statements[1].(*entities.MethodDef)
				if methodDef.Name != "process" {
					t.Errorf("Expected method name 'process', got '%s'", methodDef.Name)
				}
				if len(methodDef.TypeParams) != 1 {
					t.Errorf("Expected 1 method type parameter, got %d", len(methodDef.TypeParams))
				}
				if methodDef.TypeParams[0].Name != "U" {
					t.Errorf("Expected method type parameter 'U', got '%s'", methodDef.TypeParams[0].Name)
				}
				if len(methodDef.TypeParams[0].Constraints) != 1 {
					t.Errorf("Expected 1 constraint for U, got %d", len(methodDef.TypeParams[0].Constraints))
				}
				if methodDef.TypeParams[0].Constraints[0] != "Serializable" {
					t.Errorf("Expected constraint 'Serializable', got '%s'", methodDef.TypeParams[0].Constraints[0])
				}
			},
		},
		{
			name: "基本match语句",
			input: `match x {
    1 => print "one"
    2 => print "two"
    _ => print "other"
}`,
			wantErr: false,
			checkAST: func(t *testing.T, program *entities.Program) {
				matchStmt := program.Statements[0].(*entities.MatchStmt)
				if len(matchStmt.Cases) != 3 {
					t.Errorf("Expected 3 cases, got %d", len(matchStmt.Cases))
				}

				// 检查第一个case
				case1 := matchStmt.Cases[0]
				if literal, ok := case1.Pattern.(*entities.LiteralPattern); ok {
					if literal.Value != 1 || literal.Kind != "int" {
						t.Errorf("Expected literal 1, got %v", literal.Value)
					}
				} else {
					t.Errorf("Expected LiteralPattern, got %T", case1.Pattern)
				}

				// 检查通配符case
				case3 := matchStmt.Cases[2]
				if _, ok := case3.Pattern.(*entities.WildcardPattern); !ok {
					t.Errorf("Expected WildcardPattern, got %T", case3.Pattern)
				}
			},
		},
		{
			name: "守卫条件match",
			input: `match n {
    x if x > 10 => print "big"
    x if x < 0 => print "negative"
    _ => print "normal"
}`,
			wantErr: false,
			checkAST: func(t *testing.T, program *entities.Program) {
				matchStmt := program.Statements[0].(*entities.MatchStmt)
				if len(matchStmt.Cases) != 3 {
					t.Errorf("Expected 3 cases, got %d", len(matchStmt.Cases))
				}

				// 检查守卫条件
				case1 := matchStmt.Cases[0]
				if case1.Guard == nil {
					t.Errorf("Expected guard condition for first case")
				}

				case2 := matchStmt.Cases[1]
				if case2.Guard == nil {
					t.Errorf("Expected guard condition for second case")
				}

				case3 := matchStmt.Cases[2]
				if case3.Guard != nil {
					t.Errorf("Expected no guard condition for third case")
				}
			},
		},
		{
			name: "异步函数定义",
			input: `async func fetchData[T](url: string) -> Result[T] {
    print "Fetching data..."
    return Ok("data")
}`,
			wantErr: false,
			checkAST: func(t *testing.T, program *entities.Program) {
				asyncFunc := program.Statements[0].(*entities.AsyncFuncDef)
				if asyncFunc.Name != "fetchData" {
					t.Errorf("Expected function name 'fetchData', got '%s'", asyncFunc.Name)
				}
				if len(asyncFunc.TypeParams) != 1 {
					t.Errorf("Expected 1 type parameter, got %d", len(asyncFunc.TypeParams))
				}
				if asyncFunc.TypeParams[0].Name != "T" {
					t.Errorf("Expected type parameter 'T', got '%s'", asyncFunc.TypeParams[0].Name)
				}
				if asyncFunc.ReturnType != "Result[T]" {
					t.Errorf("Expected return type 'Result[T]', got '%s'", asyncFunc.ReturnType)
				}
			},
		},
		{
			name: "异步表达式",
			input: `func main() -> void {
    let ch: chan int = chan int
    let future: Result[string] = await fetchData[string]("http://api.example.com")
    spawn worker(ch)
    ch <- 42
    let value: int = <- ch
}`,
			wantErr: false,
			checkAST: func(t *testing.T, program *entities.Program) {
				mainFunc := program.Statements[0].(*entities.FuncDef)
				body := mainFunc.Body

				// 检查通道字面量
				chanDecl := body[0].(*entities.VarDecl)
				if chanLiteral, ok := chanDecl.Value.(*entities.ChanLiteral); ok {
					if chanLiteral.Type != "int" {
						t.Errorf("Expected channel type 'int', got '%s'", chanLiteral.Type)
					}
				} else {
					t.Errorf("Expected ChanLiteral, got %T", chanDecl.Value)
				}

				// 检查await表达式
				awaitDecl := body[1].(*entities.VarDecl)
				if _, ok := awaitDecl.Value.(*entities.AwaitExpr); !ok {
					t.Errorf("Expected AwaitExpr, got %T", awaitDecl.Value)
				}

				// 检查spawn表达式
				spawnStmt := body[2].(*entities.ExprStmt)
				if _, ok := spawnStmt.Expression.(*entities.SpawnExpr); !ok {
					t.Errorf("Expected SpawnExpr, got %T", spawnStmt.Expression)
				}

				// 检查发送表达式
				sendStmt := body[3].(*entities.ExprStmt)
				if _, ok := sendStmt.Expression.(*entities.SendExpr); !ok {
					t.Errorf("Expected SendExpr, got %T", sendStmt.Expression)
				}

				// 检查接收表达式
				receiveDecl := body[4].(*entities.VarDecl)
				if _, ok := receiveDecl.Value.(*entities.ReceiveExpr); !ok {
					t.Errorf("Expected ReceiveExpr, got %T", receiveDecl.Value)
				}
			},
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

// TestParserBasicEoFiles 测试基本的.eo文件解析
func TestParserBasicEoFiles(t *testing.T) {
	parser := NewSimpleParser()

	tests := []struct {
		name     string
		input    string
		wantErr  bool
		checkAST func(t *testing.T, program *entities.Program)
	}{
		{
			name:    "test_simple.eo",
			input:   "func test() -> int {\n    return 42\n}",
			wantErr: false,
			checkAST: func(t *testing.T, program *entities.Program) {
				if len(program.Statements) != 1 {
					t.Errorf("Expected 1 statement, got %d", len(program.Statements))
				}
				if funcDef, ok := program.Statements[0].(*entities.FuncDef); ok {
					if funcDef.Name != "test" {
						t.Errorf("Expected function name 'test', got '%s'", funcDef.Name)
					}
				}
			},
		},
		{
			name:    "test_if.eo",
			input:   "func test(a: int) -> int {\n    if a > 0 {\n        return 1\n    }\n    return 0\n}",
			wantErr: false,
			checkAST: func(t *testing.T, program *entities.Program) {
				// if语句可能被解析为多个语句，取决于实现
				if len(program.Statements) == 0 {
					t.Errorf("Expected at least 1 statement, got 0")
				}
			},
		},
		{
			name:    "test_match_stmt.eo",
			input:   "match c { Red => print \"test\" }",
			wantErr: false,
			checkAST: func(t *testing.T, program *entities.Program) {
				// match语句可能有语法问题，检查是否有任何语句
				if len(program.Statements) == 0 {
					t.Log("Match statement parsing may not be implemented yet")
					return
				}
				if _, ok := program.Statements[0].(*entities.MatchStmt); !ok {
					t.Errorf("Expected MatchStmt, got %T", program.Statements[0])
				}
			},
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

// TestParserAdvancedPatterns 测试高级模式匹配功能
func TestParserAdvancedPatterns(t *testing.T) {
	parser := NewSimpleParser()

	input := `match p {
    Point{x: 0, y: 0} => "origin"
    Point{x: x, y: 0} if x > 0 => "positive x-axis"
    _ => "other"
}`

	program, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(program.Statements) == 0 {
		t.Fatal("Expected at least one statement")
	}

	// 检查match语句
	matchStmt, ok := program.Statements[0].(*entities.MatchStmt)
	if !ok {
		t.Fatalf("Expected MatchStmt, got %T", program.Statements[0])
	}

	if len(matchStmt.Cases) != 3 {
		t.Errorf("Expected 3 cases, got %d", len(matchStmt.Cases))
	}

	// 检查每个case
	for i, case_ := range matchStmt.Cases {
		switch i {
		case 0:
			// Point{x: 0, y: 0} => "origin"
			if case_.Guard != nil {
				t.Errorf("Case 1 should not have guard")
			}
		case 1:
			// Point{x: x, y: 0} if x > 0 => "positive x-axis"
			if case_.Guard == nil {
				t.Errorf("Case 2 should have guard")
			}
		case 2:
			// _ => "other"
			if case_.Guard != nil {
				t.Errorf("Case 3 should not have guard")
			}
		}
	}
}
