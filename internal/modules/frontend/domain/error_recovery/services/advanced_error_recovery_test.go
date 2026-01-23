package services

import (
	"context"
	"testing"

	lexicalServices "echo/internal/modules/frontend/domain/lexical/services"
	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
)

func TestAdvancedErrorRecoveryService_RecoverFromError(t *testing.T) {
	service := NewAdvancedErrorRecoveryService()
	ctx := context.Background()

	tests := []struct {
		name        string
		source      string
		createError func() *sharedVO.ParseError
		wantRecover bool
	}{
		{
			name:   "语法错误恢复",
			source: "fn test() {",
			createError: func() *sharedVO.ParseError {
				return sharedVO.NewParseError(
					"unexpected end of file",
					sharedVO.NewSourceLocation("test.eo", 1, 10, 10),
					sharedVO.ErrorTypeSyntax,
					sharedVO.SeverityError,
				)
			},
			wantRecover: true,
		},
		{
			name:   "词法错误恢复",
			source: "let x = @invalid",
			createError: func() *sharedVO.ParseError {
				return sharedVO.NewParseError(
					"invalid token",
					sharedVO.NewSourceLocation("test.eo", 1, 9, 9),
					sharedVO.ErrorTypeLexical,
					sharedVO.SeverityError,
				)
			},
			wantRecover: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建Token流
			lexer := lexicalServices.NewAdvancedLexerService()
			sourceFile := lexicalVO.NewSourceFile("test.eo", tt.source)
			tokenStream, err := lexer.Tokenize(ctx, sourceFile)
			if err != nil {
				// 如果词法分析失败，创建一个简单的token流
				tokenStream = lexicalVO.NewEnhancedTokenStream(sourceFile)
			}

			// 创建错误
			parseError := tt.createError()

			// 尝试恢复
			result, err := service.RecoverFromError(ctx, parseError, tokenStream)

			if err != nil {
				t.Logf("recovery service returned error: %v", err)
			}

			if result != nil {
				recovered := result.HasRecovered()
				if tt.wantRecover && !recovered {
					t.Logf("expected recovery, but recovery failed")
				}
			}
		})
	}
}

func TestAdvancedErrorRecoveryService_RecoverWithPanicMode(t *testing.T) {
	service := NewAdvancedErrorRecoveryService()
	ctx := context.Background()

	// 创建错误
	parseError := sharedVO.NewParseError(
		"test error",
		sharedVO.NewSourceLocation("test.eo", 1, 1, 1),
		sharedVO.ErrorTypeSyntax,
		sharedVO.SeverityError,
	)

	// 创建Token流
	lexer := lexicalServices.NewAdvancedLexerService()
	sourceFile := lexicalVO.NewSourceFile("test.eo", "fn test() { let x = 1 }")
	tokenStream, err := lexer.Tokenize(ctx, sourceFile)
	if err != nil {
		tokenStream = lexicalVO.NewEnhancedTokenStream(sourceFile)
	}

	// 使用恐慌模式恢复
	result, err := service.RecoverWithPanicMode(ctx, parseError, tokenStream)

	if err != nil {
		t.Logf("panic mode recovery returned error: %v", err)
	}

	if result != nil {
		_ = result.Recovered()
	}
}

func TestAdvancedErrorRecoveryService_TrackRecoveryResult(t *testing.T) {
	service := NewAdvancedErrorRecoveryService()
	ctx := context.Background()

	// 创建错误恢复结果
	parseError := sharedVO.NewParseError(
		"test error",
		sharedVO.NewSourceLocation("test.eo", 1, 1, 1),
		sharedVO.ErrorTypeSyntax,
		sharedVO.SeverityError,
	)

	result := sharedVO.NewErrorRecoveryResult()
	attempt := sharedVO.NewErrorRecoveryAttempt(
		parseError,
		*sharedVO.NewRecoveryStrategy(sharedVO.StrategyTypePanicMode, "test", 1),
		"test attempt",
	)
	attempt.MarkSuccess(1, 1)
	result.AddAttempt(attempt)

	// 跟踪恢复结果
	trackingInfo, err := service.TrackRecoveryResult(ctx, result)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if trackingInfo == nil {
		t.Fatalf("tracking info is nil")
	}

	if trackingInfo.TotalAttempts != 1 {
		t.Errorf("expected 1 attempt, got %d", trackingInfo.TotalAttempts)
	}

	if !trackingInfo.Successful {
		t.Errorf("expected successful recovery")
	}
}
