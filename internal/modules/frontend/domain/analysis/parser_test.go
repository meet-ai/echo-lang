package analysis

import (
	"testing"
)

func TestNewSimpleParser(t *testing.T) {
	parser := NewSimpleParser()
	if parser == nil {
		t.Error("NewSimpleParser() returned nil")
	}

	// Test that it implements the Parser interface
	var _ Parser = parser
}

func TestSimpleParser_ParseEmptySource(t *testing.T) {
	parser := NewSimpleParser()
	source := NewSourceCode("", "test.eo")
	tokenizer := NewSimpleTokenizer()

	tokens, err := tokenizer.Tokenize(source)
	if err != nil {
		t.Fatalf("Tokenize failed: %v", err)
	}

	program, diagnostics, err := parser.Parse(tokens)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
	}

	if len(program.Statements()) != 0 {
		t.Errorf("Expected empty program, got %d statements", len(program.Statements()))
	}
}

func TestSimpleParser_ParseFunctionDef(t *testing.T) {
	parser := NewSimpleParser()
	source := NewSourceCode("func add(a: int, b: int) -> int {\n    return a + b\n}", "test.eo")
	tokenizer := NewSimpleTokenizer()

	tokens, err := tokenizer.Tokenize(source)
	if err != nil {
		t.Fatalf("Tokenize failed: %v", err)
	}

	program, diagnostics, err := parser.Parse(tokens)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
		for _, diag := range diagnostics {
			t.Logf("Diagnostic: %s", diag.String())
		}
	}

	if len(program.Statements()) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(program.Statements()))
	}

	stmt := program.Statements()[0]
	funcDef, ok := stmt.(*FunctionDef)
	if !ok {
		t.Fatalf("Expected FunctionDef, got %T", stmt)
	}

	if funcDef.Name() != "add" {
		t.Errorf("Expected function name 'add', got '%s'", funcDef.Name())
	}

	if len(funcDef.Params()) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(funcDef.Params()))
	}

	if funcDef.ReturnType() == nil || funcDef.ReturnType().Name() != "int" {
		t.Errorf("Expected return type 'int', got %v", funcDef.ReturnType())
	}
}

func TestSimpleParser_ParseVariableDecl(t *testing.T) {
	parser := NewSimpleParser()
	source := NewSourceCode("let x: int = 42;", "test.eo")
	tokenizer := NewSimpleTokenizer()

	tokens, err := tokenizer.Tokenize(source)
	if err != nil {
		t.Fatalf("Tokenize failed: %v", err)
	}

	program, diagnostics, err := parser.Parse(tokens)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
		for _, diag := range diagnostics {
			t.Logf("Diagnostic: %s", diag.String())
		}
	}

	if len(program.Statements()) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(program.Statements()))
	}

	stmt := program.Statements()[0]
	varDecl, ok := stmt.(*VariableDecl)
	if !ok {
		t.Fatalf("Expected VariableDecl, got %T", stmt)
	}

	if varDecl.Name() != "x" {
		t.Errorf("Expected variable name 'x', got '%s'", varDecl.Name())
	}

	if varDecl.Type() == nil || varDecl.Type().Name() != "int" {
		t.Errorf("Expected type 'int', got %v", varDecl.Type())
	}

	if varDecl.InitExpr() == nil {
		t.Error("Expected init expression")
	}
}

func TestSimpleParser_ParseBinaryExpr(t *testing.T) {
	parser := NewSimpleParser()
	source := NewSourceCode("let result = a + b * c;", "test.eo")
	tokenizer := NewSimpleTokenizer()

	tokens, err := tokenizer.Tokenize(source)
	if err != nil {
		t.Fatalf("Tokenize failed: %v", err)
	}

	program, diagnostics, err := parser.Parse(tokens)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
		for _, diag := range diagnostics {
			t.Logf("Diagnostic: %s", diag.String())
		}
	}

	if len(program.Statements()) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(program.Statements()))
	}

	stmt := program.Statements()[0]
	varDecl, ok := stmt.(*VariableDecl)
	if !ok {
		t.Fatalf("Expected VariableDecl, got %T", stmt)
	}

	if varDecl.InitExpr() == nil {
		t.Fatal("Expected init expression")
	}

	// 检查是否正确解析了运算符优先级：a + (b * c)
	binaryExpr, ok := varDecl.InitExpr().(*BinaryExpr)
	if !ok {
		t.Fatalf("Expected BinaryExpr, got %T", varDecl.InitExpr())
	}

	if binaryExpr.Operator().Kind() != Plus {
		t.Errorf("Expected '+' operator, got %s", binaryExpr.Operator().Kind().String())
	}
}

func TestSimpleParser_ParseFunctionCall(t *testing.T) {
	parser := NewSimpleParser()
	source := NewSourceCode("print(add(1, 2));", "test.eo")
	tokenizer := NewSimpleTokenizer()

	tokens, err := tokenizer.Tokenize(source)
	if err != nil {
		t.Fatalf("Tokenize failed: %v", err)
	}

	program, diagnostics, err := parser.Parse(tokens)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
		for _, diag := range diagnostics {
			t.Logf("Diagnostic: %s", diag.String())
		}
	}

	if len(program.Statements()) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(program.Statements()))
	}

	stmt := program.Statements()[0]
	exprStmt, ok := stmt.(*ExprStmt)
	if !ok {
		t.Fatalf("Expected ExprStmt, got %T", stmt)
	}

	funcCall, ok := exprStmt.Expr().(*FunctionCallExpr)
	if !ok {
		t.Fatalf("Expected FunctionCallExpr, got %T", exprStmt.Expr())
	}

	if funcCall.Function().(*IdentifierExpr).Name() != "print" {
		t.Errorf("Expected function name 'print', got '%s'", funcCall.Function().(*IdentifierExpr).Name())
	}

	if len(funcCall.Arguments()) != 1 {
		t.Fatalf("Expected 1 argument, got %d", len(funcCall.Arguments()))
	}

	// 检查参数是否是add函数调用
	argCall, ok := funcCall.Arguments()[0].(*FunctionCallExpr)
	if !ok {
		t.Fatalf("Expected FunctionCallExpr as argument, got %T", funcCall.Arguments()[0])
	}

	if argCall.Function().(*IdentifierExpr).Name() != "add" {
		t.Errorf("Expected argument function name 'add', got '%s'", argCall.Function().(*IdentifierExpr).Name())
	}

	if len(argCall.Arguments()) != 2 {
		t.Errorf("Expected 2 arguments for add, got %d", len(argCall.Arguments()))
	}
}

func TestSimpleParser_ParseIfStmt(t *testing.T) {
	parser := NewSimpleParser()
	source := NewSourceCode("if x > 0 {\n    return x\n} else {\n    return -x\n}", "test.eo")
	tokenizer := NewSimpleTokenizer()

	tokens, err := tokenizer.Tokenize(source)
	if err != nil {
		t.Fatalf("Tokenize failed: %v", err)
	}

	program, diagnostics, err := parser.Parse(tokens)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
		for _, diag := range diagnostics {
			t.Logf("Diagnostic: %s", diag.String())
		}
	}

	if len(program.Statements()) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(program.Statements()))
	}

	stmt := program.Statements()[0]
	ifStmt, ok := stmt.(*IfStmt)
	if !ok {
		t.Fatalf("Expected IfStmt, got %T", stmt)
	}

	if ifStmt.Condition() == nil {
		t.Error("Expected condition expression")
	}

	if ifStmt.ThenBody() == nil {
		t.Error("Expected then body")
	}

	if ifStmt.ElseBody() == nil {
		t.Error("Expected else body")
	}
}

func TestSimpleParser_ParseErrorRecovery(t *testing.T) {
	parser := NewSimpleParser()
	// 包含语法错误的代码
	source := NewSourceCode("func broken(\nlet x = 1;\nfunc valid() { return 1 }", "test.eo")
	tokenizer := NewSimpleTokenizer()

	tokens, err := tokenizer.Tokenize(source)
	if err != nil {
		t.Fatalf("Tokenize failed: %v", err)
	}

	program, diagnostics, err := parser.Parse(tokens)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// 应该有诊断信息（错误）
	if len(diagnostics) == 0 {
		t.Error("Expected diagnostics for syntax errors, but got none")
	}

	// 即使有错误，也应该解析出有效的部分
	if len(program.Statements()) == 0 {
		t.Error("Expected some valid statements even with errors")
	}
}

// TestParserGenerics 测试泛型解析功能（基础测试）
func TestParserGenerics(t *testing.T) {
	parser := NewSimpleParser()

	source := NewSourceCode(`struct Container[T] {
    value: T
}

func (c Container[T]) get() -> T {
    return c.value
}`, "demo_generics.eo")

	tokenizer := NewSimpleTokenizer()
	tokens, err := tokenizer.Tokenize(source)
	if err != nil {
		t.Fatalf("Tokenize error: %v", err)
	}

	program, diagnostics, err := parser.Parse(tokens)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// 泛型功能可能还未完全实现，检查解析器不会崩溃
	t.Logf("Parsing completed with %d diagnostics", len(diagnostics))
	if len(diagnostics) > 0 {
		t.Logf("Diagnostics: %v", diagnostics)
	}

	// 即使解析不完全，也应该返回一个有效的program对象
	if program == nil {
		t.Error("Expected program to be non-nil")
	}
}

// TestParserGenericFunctionIntegration 测试泛型函数集成（基础测试）
func TestParserGenericFunctionIntegration(t *testing.T) {
	parser := NewSimpleParser()

	// 使用test_generic_integration.eo的内容
	source := NewSourceCode(`// 泛型系统基础测试

// 泛型函数定义
func identity[T](value: T) -> T {
    return value
}

// 主函数测试泛型函数
func main() -> void {
    // 显式类型参数调用
    let intResult: int = identity[int](42)
    print "Generic function test passed"
}`, "test_generic_integration.eo")

	tokenizer := NewSimpleTokenizer()
	tokens, err := tokenizer.Tokenize(source)
	if err != nil {
		t.Fatalf("Tokenize error: %v", err)
	}

	program, diagnostics, err := parser.Parse(tokens)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// 检查解析器能够处理泛型语法而不崩溃
	t.Logf("Parsing completed with %d diagnostics", len(diagnostics))
	if len(diagnostics) > 0 {
		t.Logf("Diagnostics: %v", diagnostics)
	}

	// 即使解析不完全，也应该返回一个有效的program对象
	if program == nil {
		t.Error("Expected program to be non-nil")
	}
}
