// Package frontend provides the frontend processing context for Echo Language compilation.
// This module handles lexical analysis, syntax analysis, and semantic analysis of .eo source files.
package frontend

import (
	"context"
	"fmt"

	"github.com/samber/do"
)

// Module represents the frontend processing module
type Module struct {
	// Ports - external interfaces
	frontendService  FrontendService
	tokenizerService TokenizerService

	// Domain services
	lexicalAnalyzer  LexicalAnalyzer
	syntaxAnalyzer   SyntaxAnalyzer
	semanticAnalyzer SemanticAnalyzer
	errorHandler     ErrorHandler

	// Infrastructure
	sourceFileRepo SourceFileRepository
	astRepo        ASTRepository

	// Adapters
	ocamllexAdapter OCamlLexAdapter
	menhirAdapter   MenhirAdapter
	ocamlASTAdapter OCamlASTAdapter
}

// FrontendService defines the interface for frontend processing operations
type FrontendService interface {
	PerformLexicalAnalysis(ctx context.Context, cmd PerformLexicalAnalysisCommand) (*LexicalAnalysisResult, error)
	PerformSyntaxAnalysis(ctx context.Context, cmd PerformSyntaxAnalysisCommand) (*SyntaxAnalysisResult, error)
	PerformSemanticAnalysis(ctx context.Context, cmd PerformSemanticAnalysisCommand) (*SemanticAnalysisResult, error)
	HandleCompilationErrors(ctx context.Context, cmd HandleCompilationErrorsCommand) (*ErrorHandlingResult, error)
}

// TokenizerService defines the interface for tokenization operations
type TokenizerService interface {
	GetAnalysisStatus(ctx context.Context, query GetAnalysisStatusQuery) (*AnalysisStatusDTO, error)
	GetASTStructure(ctx context.Context, query GetASTStructureQuery) (*ASTStructureDTO, error)
}

// NewModule creates a new frontend module with dependency injection
func NewModule(i *do.Injector) (*Module, error) {
	// Get dependencies from DI container
	lexicalAnalyzer := do.MustInvoke[LexicalAnalyzer](i)
	syntaxAnalyzer := do.MustInvoke[SyntaxAnalyzer](i)
	semanticAnalyzer := do.MustInvoke[SemanticAnalyzer](i)
	errorHandler := do.MustInvoke[ErrorHandler](i)
	sourceFileRepo := do.MustInvoke[SourceFileRepository](i)
	astRepo := do.MustInvoke[ASTRepository](i)

	// Get infrastructure adapters
	ocamllexAdapter := do.MustInvoke[OCamlLexAdapter](i)
	menhirAdapter := do.MustInvoke[MenhirAdapter](i)
	ocamlASTAdapter := do.MustInvoke[OCamlASTAdapter](i)

	// Create application services
	frontendSvc := NewFrontendService(lexicalAnalyzer, syntaxAnalyzer, semanticAnalyzer, errorHandler)
	tokenizerSvc := NewTokenizerService(sourceFileRepo, astRepo)

	return &Module{
		frontendService:  frontendSvc,
		tokenizerService: tokenizerSvc,
		lexicalAnalyzer:  lexicalAnalyzer,
		syntaxAnalyzer:   syntaxAnalyzer,
		semanticAnalyzer: semanticAnalyzer,
		errorHandler:     errorHandler,
		sourceFileRepo:   sourceFileRepo,
		astRepo:          astRepo,
		ocamllexAdapter:  ocamllexAdapter,
		menhirAdapter:    menhirAdapter,
		ocamlASTAdapter:  ocamlASTAdapter,
	}, nil
}

// FrontendService returns the frontend service interface
func (m *Module) FrontendService() FrontendService {
	return m.frontendService
}

// TokenizerService returns the tokenizer service interface
func (m *Module) TokenizerService() TokenizerService {
	return m.tokenizerService
}

// Validate validates the module configuration
func (m *Module) Validate() error {
	if m.frontendService == nil {
		return fmt.Errorf("frontend service is not initialized")
	}
	if m.tokenizerService == nil {
		return fmt.Errorf("tokenizer service is not initialized")
	}
	if m.ocamllexAdapter == nil {
		return fmt.Errorf("ocamllex adapter is not initialized")
	}
	if m.menhirAdapter == nil {
		return fmt.Errorf("menhir adapter is not initialized")
	}
	if m.ocamlASTAdapter == nil {
		return fmt.Errorf("ocaml AST adapter is not initialized")
	}
	return nil
}
