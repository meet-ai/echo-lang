package services

import (
	"context"
	"testing"

	lexicalServices "echo/internal/modules/frontend/domain/lexical/services"
	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
	syntaxVO "echo/internal/modules/frontend/domain/syntax/value_objects"
)

func TestPrattExpressionParser_ParseExpression_Basic(t *testing.T) {
	parser := NewPrattExpressionParser()
	lexer := lexicalServices.NewAdvancedLexerService()
	ctx := context.Background()

	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name:    "简单数字",
			source:  "123",
			wantErr: false,
		},
		{
			name:    "简单标识符",
			source:  "x",
			wantErr: false,
		},
		{
			name:    "字符串字面量",
			source:  `"hello"`,
			wantErr: false,
		},
		{
			name:    "布尔值",
			source:  "true",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 先进行词法分析
			sourceFile := lexicalVO.NewSourceFile("test.eo", tt.source)
			tokenStream, err := lexer.Tokenize(ctx, sourceFile)
			if err != nil {
				t.Fatalf("lexical analysis failed: %v", err)
			}

			// 解析表达式
			expr, err := parser.ParseExpression(ctx, tokenStream, syntaxVO.PrecedencePrimary)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expr == nil {
				t.Fatalf("expression is nil")
			}
		})
	}
}

func TestPrattExpressionParser_ParseExpression_BinaryOperators(t *testing.T) {
	parser := NewPrattExpressionParser()
	lexer := lexicalServices.NewAdvancedLexerService()
	ctx := context.Background()

	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name:    "加法",
			source:  "1 + 2",
			wantErr: false,
		},
		{
			name:    "减法",
			source:  "5 - 3",
			wantErr: false,
		},
		{
			name:    "乘法",
			source:  "2 * 3",
			wantErr: false,
		},
		{
			name:    "除法",
			source:  "10 / 2",
			wantErr: false,
		},
		{
			name:    "复杂表达式",
			source:  "1 + 2 * 3",
			wantErr: false,
		},
		{
			name:    "括号表达式",
			source:  "(1 + 2) * 3",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 先进行词法分析
			sourceFile := lexicalVO.NewSourceFile("test.eo", tt.source)
			tokenStream, err := lexer.Tokenize(ctx, sourceFile)
			if err != nil {
				t.Fatalf("lexical analysis failed: %v", err)
			}

			// 解析表达式
			expr, err := parser.ParseExpression(ctx, tokenStream, syntaxVO.PrecedencePrimary)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expr == nil {
				t.Fatalf("expression is nil")
			}
		})
	}
}

func TestPrattExpressionParser_ParseExpression_UnaryOperators(t *testing.T) {
	parser := NewPrattExpressionParser()
	lexer := lexicalServices.NewAdvancedLexerService()
	ctx := context.Background()

	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name:    "一元负号",
			source:  "-5",
			wantErr: false,
		},
		{
			name:    "一元正号",
			source:  "+10",
			wantErr: false,
		},
		{
			name:    "逻辑非",
			source:  "!true",
			wantErr: false,
		},
		{
			name:    "双重否定",
			source:  "--5",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 先进行词法分析
			sourceFile := lexicalVO.NewSourceFile("test.eo", tt.source)
			tokenStream, err := lexer.Tokenize(ctx, sourceFile)
			if err != nil {
				t.Fatalf("lexical analysis failed: %v", err)
			}

			// 解析表达式
			expr, err := parser.ParseExpression(ctx, tokenStream, syntaxVO.PrecedencePrimary)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expr == nil {
				t.Fatalf("expression is nil")
			}
		})
	}
}

func TestPrattExpressionParser_ParseExpression_Precedence(t *testing.T) {
	parser := NewPrattExpressionParser()
	lexer := lexicalServices.NewAdvancedLexerService()
	ctx := context.Background()

	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name:    "乘法优先于加法",
			source:  "1 + 2 * 3",
			wantErr: false,
		},
		{
			name:    "除法优先于减法",
			source:  "10 - 8 / 2",
			wantErr: false,
		},
		{
			name:    "括号改变优先级",
			source:  "(1 + 2) * 3",
			wantErr: false,
		},
		{
			name:    "复杂优先级",
			source:  "1 + 2 * 3 - 4 / 2",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 先进行词法分析
			sourceFile := lexicalVO.NewSourceFile("test.eo", tt.source)
			tokenStream, err := lexer.Tokenize(ctx, sourceFile)
			if err != nil {
				t.Fatalf("lexical analysis failed: %v", err)
			}

			// 解析表达式
			expr, err := parser.ParseExpression(ctx, tokenStream, syntaxVO.PrecedencePrimary)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expr == nil {
				t.Fatalf("expression is nil")
			}
		})
	}
}

func TestPrattExpressionParser_ParseExpression_Associativity(t *testing.T) {
	parser := NewPrattExpressionParser()
	lexer := lexicalServices.NewAdvancedLexerService()
	ctx := context.Background()

	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name:    "左结合加法",
			source:  "1 + 2 + 3",
			wantErr: false,
		},
		{
			name:    "左结合减法",
			source:  "10 - 5 - 2",
			wantErr: false,
		},
		{
			name:    "左结合乘法",
			source:  "2 * 3 * 4",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 先进行词法分析
			sourceFile := lexicalVO.NewSourceFile("test.eo", tt.source)
			tokenStream, err := lexer.Tokenize(ctx, sourceFile)
			if err != nil {
				t.Fatalf("lexical analysis failed: %v", err)
			}

			// 解析表达式
			expr, err := parser.ParseExpression(ctx, tokenStream, syntaxVO.PrecedencePrimary)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expr == nil {
				t.Fatalf("expression is nil")
			}
		})
	}
}

func TestPrattExpressionParser_ParseExpression_ArrayLiteral(t *testing.T) {
	parser := NewPrattExpressionParser()
	lexer := lexicalServices.NewAdvancedLexerService()
	ctx := context.Background()

	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name:    "空数组",
			source:  "[]",
			wantErr: false,
		},
		{
			name:    "单元素数组",
			source:  "[1]",
			wantErr: false,
		},
		{
			name:    "多元素数组",
			source:  "[1, 2, 3]",
			wantErr: false,
		},
		{
			name:    "嵌套数组",
			source:  "[[1, 2], [3, 4]]",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 先进行词法分析
			sourceFile := lexicalVO.NewSourceFile("test.eo", tt.source)
			tokenStream, err := lexer.Tokenize(ctx, sourceFile)
			if err != nil {
				t.Fatalf("lexical analysis failed: %v", err)
			}

			// 解析表达式
			expr, err := parser.ParseExpression(ctx, tokenStream, syntaxVO.PrecedencePrimary)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expr == nil {
				t.Fatalf("expression is nil")
			}
		})
	}
}

func TestPrattExpressionParser_ParseExpression_FunctionCall(t *testing.T) {
	parser := NewPrattExpressionParser()
	lexer := lexicalServices.NewAdvancedLexerService()
	ctx := context.Background()

	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name:    "无参数函数调用",
			source:  "foo()",
			wantErr: false,
		},
		{
			name:    "单参数函数调用",
			source:  "foo(1)",
			wantErr: false,
		},
		{
			name:    "多参数函数调用",
			source:  "foo(1, 2, 3)",
			wantErr: false,
		},
		{
			name:    "嵌套函数调用",
			source:  "foo(bar(1))",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 先进行词法分析
			sourceFile := lexicalVO.NewSourceFile("test.eo", tt.source)
			tokenStream, err := lexer.Tokenize(ctx, sourceFile)
			if err != nil {
				t.Fatalf("lexical analysis failed: %v", err)
			}

			// 解析表达式
			expr, err := parser.ParseExpression(ctx, tokenStream, syntaxVO.PrecedencePrimary)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expr == nil {
				t.Fatalf("expression is nil")
			}
		})
	}
}
