package services

import (
	"context"
	"time"

	"echo/internal/modules/frontend/domain/entities"
	valueobjects "echo/internal/modules/frontend/domain/valueobjects"
)

// LexicalAnalyzer 词法分析器接口
type LexicalAnalyzer interface {
	Analyze(ctx context.Context, sourceFile *entities.SourceFile) (*AnalysisResult, error)
}

// SyntaxAnalyzer 语法分析器接口
type SyntaxAnalyzer interface {
	Analyze(ctx context.Context, sourceFile *entities.SourceFile) (*AnalysisResult, error)
}

// SemanticAnalyzer 语义分析器接口
type SemanticAnalyzer interface {
	Analyze(ctx context.Context, sourceFile *entities.SourceFile) (*AnalysisResult, error)
}

// ErrorHandler 错误处理器接口
type ErrorHandler interface {
	HandleAnalysisError(ctx context.Context, sourceFileID, analysisType string, err error) *ErrorInfo
	HandleCompilationErrors(ctx context.Context, sourceFileID string, errors []string) *ErrorHandlingResult
}

// Parser 解析器接口
type Parser interface {
	Parse(source string) (*entities.Program, error)
}

// AnalysisResult 分析结果
type AnalysisResult struct {
	Tokens      []valueobjects.Token // 使用具体的 Token 类型
	AST         *entities.ASTNode
	SymbolTable interface{} // 暂时使用 interface{}
	Duration    time.Duration // 使用具体的时间类型
	TokenCount  int // Token 数量
	ErrorCount  int
	Warnings    []string
}

// ErrorInfo 错误信息
type ErrorInfo struct {
	Message  string
	Type     string
	Severity string
}

// ErrorHandlingResult 错误处理结果
type ErrorHandlingResult struct {
	SourceFileID   string
	Suggestions    []string
	ProcessedCount int
	Success        bool
}
