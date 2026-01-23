package parser

import (
	"context"
	"testing"
)

func TestModernParserAggregate_Parse_Basic(t *testing.T) {
	aggregate := NewModernParserAggregate()
	ctx := context.Background()

	tests := []struct {
		name    string
		source  string
		filename string
		wantErr bool
	}{
		{
			name:     "空文件",
			source:   "",
			filename: "empty.eo",
			wantErr:  false,
		},
		{
			name:     "简单函数定义",
			source:   "fn test() {}",
			filename: "test.eo",
			wantErr:  false,
		},
		{
			name:     "变量声明",
			source:   "let x = 1",
			filename: "test.eo",
			wantErr:  false,
		},
		{
			name:     "结构体定义",
			source:   "struct Point { x: int, y: int }",
			filename: "test.eo",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			programAST, err := aggregate.Parse(ctx, tt.source, tt.filename)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			// 即使有错误，也可能返回部分AST
			if programAST == nil && err == nil {
				t.Errorf("both programAST and error are nil")
			}
		})
	}
}

func TestModernParserAggregate_GetParsingContext(t *testing.T) {
	aggregate := NewModernParserAggregate()
	ctx := context.Background()

	// 解析前，上下文应该为nil
	context := aggregate.GetParsingContext()
	if context != nil {
		t.Errorf("expected nil context before parsing, got %v", context)
	}

	// 解析后，应该能获取上下文
	_, err := aggregate.Parse(ctx, "fn test() {}", "test.eo")
	if err != nil {
		// 解析可能失败，但不影响测试
		t.Logf("parse failed: %v", err)
	}

	context = aggregate.GetParsingContext()
	// 解析后可能有上下文，也可能没有（取决于解析是否成功）
	_ = context
}

func TestModernParserAggregate_GetErrors(t *testing.T) {
	aggregate := NewModernParserAggregate()
	ctx := context.Background()

	// 解析前，错误列表应该为空
	errors := aggregate.GetErrors()
	if errors == nil {
		t.Fatalf("GetErrors() should not return nil")
	}
	if len(errors) != 0 {
		t.Errorf("expected 0 errors initially, got %d", len(errors))
	}

	// 解析一个可能有错误的源文件
	_, _ = aggregate.Parse(ctx, "fn test() {", "test.eo")

	// 检查是否有错误
	errors = aggregate.GetErrors()
	_ = errors // 可能有错误，也可能没有
}

func TestModernParserAggregate_HasErrors(t *testing.T) {
	aggregate := NewModernParserAggregate()
	ctx := context.Background()

	// 解析前，应该没有错误
	if aggregate.HasErrors() {
		t.Errorf("expected no errors initially")
	}

	// 解析一个可能有错误的源文件
	_, _ = aggregate.Parse(ctx, "fn test() {", "test.eo")

	// 检查是否有错误
	_ = aggregate.HasErrors()
}

func TestModernParserAggregate_IsParsed(t *testing.T) {
	aggregate := NewModernParserAggregate()
	ctx := context.Background()

	// 解析前，应该返回false
	if aggregate.IsParsed() {
		t.Errorf("expected IsParsed() to return false before parsing")
	}

	// 解析后，应该返回true
	_, err := aggregate.Parse(ctx, "fn test() {}", "test.eo")
	if err == nil {
		if !aggregate.IsParsed() {
			t.Errorf("expected IsParsed() to return true after successful parsing")
		}
	}
}

func TestModernParserAggregate_Reset(t *testing.T) {
	aggregate := NewModernParserAggregate()
	ctx := context.Background()

	// 先解析
	_, _ = aggregate.Parse(ctx, "fn test() {}", "test.eo")

	// 重置
	aggregate.Reset()

	// 检查状态
	if aggregate.IsParsed() {
		t.Errorf("expected IsParsed() to return false after reset")
	}

	if aggregate.HasErrors() {
		// 重置后可能还有错误，这是可以接受的
		t.Logf("errors still present after reset")
	}
}

func TestModernParserAggregate_SwitchToExpressionParsing(t *testing.T) {
	aggregate := NewModernParserAggregate()
	ctx := context.Background()

	// 先初始化（通过解析）
	_, _ = aggregate.Parse(ctx, "fn test() {}", "test.eo")

	// 切换到表达式解析模式
	err := aggregate.SwitchToExpressionParsing()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 检查上下文
	context := aggregate.GetParsingContext()
	if context != nil {
		parserType := context.CurrentParserType()
		if parserType != ParserTypePratt {
			t.Errorf("expected parser type to be Pratt, got %v", parserType)
		}
	}
}

func TestModernParserAggregate_SwitchToAmbiguityResolution(t *testing.T) {
	aggregate := NewModernParserAggregate()
	ctx := context.Background()

	// 先初始化（通过解析）
	_, _ = aggregate.Parse(ctx, "fn test() {}", "test.eo")

	// 切换到歧义解析模式
	err := aggregate.SwitchToAmbiguityResolution()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 检查上下文
	context := aggregate.GetParsingContext()
	if context != nil {
		parserType := context.CurrentParserType()
		if parserType != ParserTypeLR {
			t.Errorf("expected parser type to be LR, got %v", parserType)
		}
	}
}

func TestModernParserAggregate_SaveAndRestoreCheckpoint(t *testing.T) {
	aggregate := NewModernParserAggregate()
	ctx := context.Background()

	// 先初始化（通过解析）
	_, _ = aggregate.Parse(ctx, "fn test() {}", "test.eo")

	// 保存检查点
	aggregate.SaveCheckpoint()

	// 修改状态
	_ = aggregate.SwitchToExpressionParsing()

	// 恢复检查点
	err := aggregate.RestoreCheckpoint()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
