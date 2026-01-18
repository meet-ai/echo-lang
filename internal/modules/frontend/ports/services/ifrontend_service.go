package services

import (
	"context"

	"github.com/meetai/echo-lang/internal/modules/frontend"
)

// IFrontendService defines the interface for frontend processing operations.
// This interface is exposed to other modules for source code analysis.
type IFrontendService interface {
	// PerformLexicalAnalysis performs lexical analysis on source code
	PerformLexicalAnalysis(ctx context.Context, cmd frontend.PerformLexicalAnalysisCommand) (*frontend.LexicalAnalysisResult, error)

	// PerformSyntaxAnalysis performs syntax analysis on tokens
	PerformSyntaxAnalysis(ctx context.Context, cmd frontend.PerformSyntaxAnalysisCommand) (*frontend.SyntaxAnalysisResult, error)

	// PerformSemanticAnalysis performs semantic analysis on AST
	PerformSemanticAnalysis(ctx context.Context, cmd frontend.PerformSemanticAnalysisCommand) (*frontend.SemanticAnalysisResult, error)

	// HandleCompilationErrors handles and reports compilation errors
	HandleCompilationErrors(ctx context.Context, cmd frontend.HandleCompilationErrorsCommand) (*frontend.ErrorHandlingResult, error)

	// CompileFile compiles a source file end-to-end
	CompileFile(filePath string) (*CompilationResult, error)
}

// CompilationResult 编译结果
type CompilationResult struct {
	SourceFile    string
	AST           string
	GeneratedCode string
	Success       bool
	Error         error
}
