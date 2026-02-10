package services

import (
	"context"

	"echo/internal/modules/frontend"
	"echo/internal/modules/frontend/domain/commands"
	"echo/internal/modules/frontend/domain/dtos"
)

// IFrontendService defines the interface for frontend processing operations.
// This interface is exposed to other modules for source code analysis.
type IFrontendService interface {
	// PerformLexicalAnalysis performs lexical analysis on source code
	PerformLexicalAnalysis(ctx context.Context, cmd frontend.PerformLexicalAnalysisCommand) (*commands.LexicalAnalysisResult, error)

	// PerformSyntaxAnalysis performs syntax analysis on tokens
	PerformSyntaxAnalysis(ctx context.Context, cmd commands.PerformSyntaxAnalysisCommand) (*commands.SyntaxAnalysisResult, error)

	// PerformSemanticAnalysis performs semantic analysis on AST
	PerformSemanticAnalysis(ctx context.Context, cmd commands.PerformSemanticAnalysisCommand) (*commands.SemanticAnalysisResult, error)

	// HandleCompilationErrors handles and reports compilation errors
	HandleCompilationErrors(ctx context.Context, cmd commands.HandleCompilationErrorsCommand) (*commands.ErrorHandlingResult, error)

	// CompileFile compiles a source file end-to-end
	CompileFile(filePath string) (*dtos.CompilationResult, error)
}
