package services

import (
	"context"
	"testing"

	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
)

func TestAdvancedLexerService_Tokenize_Basic(t *testing.T) {
	lexer := NewAdvancedLexerService()
	ctx := context.Background()

	tests := []struct {
		name      string
		source    string
		filename  string
		wantCount int // 期望的token数量（不包括EOF）
		wantErr   bool
	}{
		{
			name:      "空文件",
			source:    "",
			filename:  "empty.eo",
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "简单标识符",
			source:    "hello",
			filename:  "test.eo",
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "数字字面量",
			source:    "123",
			filename:  "test.eo",
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "字符串字面量",
			source:    `"hello world"`,
			filename:  "test.eo",
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "运算符",
			source:    "+ - * /",
			filename:  "test.eo",
			wantCount: 4,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceFile := lexicalVO.NewSourceFile(tt.filename, tt.source)
			tokenStream, err := lexer.Tokenize(ctx, sourceFile)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tokenStream == nil {
				t.Fatalf("tokenStream is nil")
			}

			// 检查token数量（不包括EOF）
			tokenCount := tokenStream.Count() - 1 // 减去EOF
			if tokenCount != tt.wantCount {
				t.Errorf("token count mismatch: want %d, got %d", tt.wantCount, tokenCount)
			}
		})
	}
}

func TestAdvancedLexerService_Tokenize_NumberFormats(t *testing.T) {
	lexer := NewAdvancedLexerService()
	ctx := context.Background()

	tests := []struct {
		name     string
		source   string
		wantType lexicalVO.EnhancedTokenType
	}{
		{
			name:     "十进制整数",
			source:   "123",
			wantType: lexicalVO.EnhancedTokenTypeNumber,
		},
		{
			name:     "十六进制",
			source:   "0xFF",
			wantType: lexicalVO.EnhancedTokenTypeNumber,
		},
		{
			name:     "二进制",
			source:   "0b1010",
			wantType: lexicalVO.EnhancedTokenTypeNumber,
		},
		{
			name:     "八进制",
			source:   "0o777",
			wantType: lexicalVO.EnhancedTokenTypeNumber,
		},
		{
			name:     "浮点数",
			source:   "3.14",
			wantType: lexicalVO.EnhancedTokenTypeNumber,
		},
		{
			name:     "科学计数法",
			source:   "1.5e10",
			wantType: lexicalVO.EnhancedTokenTypeNumber,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceFile := lexicalVO.NewSourceFile("test.eo", tt.source)
			tokenStream, err := lexer.Tokenize(ctx, sourceFile)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tokenStream == nil {
				t.Fatalf("tokenStream is nil")
			}
			if tokenStream.Count() == 0 {
				t.Fatalf("should have at least one token")
			}

			// 检查第一个token的类型
			firstToken := tokenStream.Current()
			if firstToken != nil {
				if firstToken.Type() != tt.wantType {
					t.Errorf("token type mismatch: want %v, got %v", tt.wantType, firstToken.Type())
				}
			}
		})
	}
}

func TestAdvancedLexerService_Tokenize_StringEscapes(t *testing.T) {
	lexer := NewAdvancedLexerService()
	ctx := context.Background()

	tests := []struct {
		name   string
		source string
		wantErr bool
	}{
		{
			name:    "普通字符串",
			source:  `"hello"`,
			wantErr: false,
		},
		{
			name:    "转义引号",
			source:  `"say \"hello\""`,
			wantErr: false,
		},
		{
			name:    "转义换行",
			source:  `"line1\nline2"`,
			wantErr: false,
		},
		{
			name:    "转义制表符",
			source:  `"tab\there"`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceFile := lexicalVO.NewSourceFile("test.eo", tt.source)
			tokenStream, err := lexer.Tokenize(ctx, sourceFile)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tokenStream == nil {
				t.Fatalf("tokenStream is nil")
			}
		})
	}
}

func TestAdvancedLexerService_Tokenize_Comments(t *testing.T) {
	lexer := NewAdvancedLexerService()
	ctx := context.Background()

	tests := []struct {
		name      string
		source    string
		wantCount int // 期望的token数量（不包括EOF和注释）
	}{
		{
			name:      "单行注释",
			source:    "hello // comment\nworld",
			wantCount: 2, // hello, world
		},
		{
			name:      "多行注释",
			source:    "hello /* comment */ world",
			wantCount: 2, // hello, world
		},
		{
			name:      "嵌套注释",
			source:    "hello /* outer /* inner */ */ world",
			wantCount: 2, // hello, world
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceFile := lexicalVO.NewSourceFile("test.eo", tt.source)
			tokenStream, err := lexer.Tokenize(ctx, sourceFile)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tokenStream == nil {
				t.Fatalf("tokenStream is nil")
			}

			// 统计非注释、非EOF的token
			count := 0
			for i := 0; i < tokenStream.Count(); i++ {
				token := tokenStream.Tokens()[i]
				if token.Type() != lexicalVO.EnhancedTokenTypeEOF {
					count++
				}
			}

			if count != tt.wantCount {
				t.Errorf("token count mismatch: want %d, got %d", tt.wantCount, count)
			}
		})
	}
}

func TestAdvancedLexerService_Tokenize_Operators(t *testing.T) {
	lexer := NewAdvancedLexerService()
	ctx := context.Background()

	tests := []struct {
		name      string
		source    string
		wantCount int
	}{
		{
			name:      "算术运算符",
			source:    "+ - * / %",
			wantCount: 5,
		},
		{
			name:      "比较运算符",
			source:    "< > <= >= == !=",
			wantCount: 6,
		},
		{
			name:      "逻辑运算符",
			source:    "&& || !",
			wantCount: 3,
		},
		{
			name:      "赋值运算符",
			source:    "= += -= *= /=",
			wantCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceFile := lexicalVO.NewSourceFile("test.eo", tt.source)
			tokenStream, err := lexer.Tokenize(ctx, sourceFile)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tokenStream == nil {
				t.Fatalf("tokenStream is nil")
			}

			// 检查token数量（不包括EOF）
			tokenCount := tokenStream.Count() - 1
			if tokenCount != tt.wantCount {
				t.Errorf("token count mismatch: want %d, got %d", tt.wantCount, tokenCount)
			}
		})
	}
}

func TestAdvancedLexerService_Tokenize_Keywords(t *testing.T) {
	lexer := NewAdvancedLexerService()
	ctx := context.Background()

	keywords := []string{
		"fn", "let", "if", "else", "while", "for", "return",
		"struct", "enum", "trait", "impl", "async", "await",
		"true", "false", "nil", "match", "select",
	}

	for _, keyword := range keywords {
		t.Run(keyword, func(t *testing.T) {
			sourceFile := lexicalVO.NewSourceFile("test.eo", keyword)
			tokenStream, err := lexer.Tokenize(ctx, sourceFile)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tokenStream == nil {
				t.Fatalf("tokenStream is nil")
			}

			firstToken := tokenStream.Current()
			if firstToken == nil {
				t.Fatalf("firstToken is nil")
			}
			if firstToken.Type() != lexicalVO.EnhancedTokenTypeKeyword {
				t.Errorf("keyword should be recognized: want %v, got %v", lexicalVO.EnhancedTokenTypeKeyword, firstToken.Type())
			}
		})
	}
}

func TestAdvancedLexerService_Tokenize_LocationTracking(t *testing.T) {
	lexer := NewAdvancedLexerService()
	ctx := context.Background()

	source := "hello\nworld\ntest"
	sourceFile := lexicalVO.NewSourceFile("test.eo", source)
	tokenStream, err := lexer.Tokenize(ctx, sourceFile)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokenStream == nil {
		t.Fatalf("tokenStream is nil")
	}

	// 检查位置信息
	tokens := tokenStream.Tokens()
	if len(tokens) == 0 {
		t.Fatalf("should have tokens")
	}

	// 第一个token应该在第一行
	if len(tokens) > 0 {
		firstToken := tokens[0]
		location := firstToken.Location()
		if location.Line() != 1 {
			t.Errorf("first token should be on line 1, got %d", location.Line())
		}
	}
}
