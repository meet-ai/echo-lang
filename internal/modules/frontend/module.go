// Package frontend provides the frontend processing context for Echo Language compilation.
// This module handles lexical analysis, syntax analysis, and semantic analysis of .eo source files.
package frontend

import (
	"fmt"

	appcommands "echo/internal/modules/frontend/application/commands"
	domaincommands "echo/internal/modules/frontend/domain/commands"
	"echo/internal/modules/frontend/domain/dtos"

	"github.com/samber/do"
)

// Module represents the frontend processing module
type Module struct {
	// Ports - external interfaces
	frontendService  FrontendService
	tokenizerService TokenizerService
}

// FrontendService defines the interface for frontend operations
type FrontendService interface {
	AnalyzeSource(filename string) (*dtos.AnalysisResultDTO, error)
}

// TokenizerService defines the interface for tokenization operations
type TokenizerService interface {
	Tokenize(source string) ([]Token, error)
}

// NewModule creates a new frontend module with dependency injection
func NewModule(i *do.Injector) (*Module, error) {
	// TODO: Implement actual services
	return &Module{
		frontendService:  nil,
		tokenizerService: nil,
	}, nil
}

// FrontendService returns the frontend service interface
func (m *Module) FrontendService() FrontendService {
	return m.frontendService
}

// Validate validates the module configuration
func (m *Module) Validate() error {
	if m.frontendService == nil {
		return fmt.Errorf("frontend service is not initialized")
	}
	if m.tokenizerService == nil {
		return fmt.Errorf("tokenizer service is not initialized")
	}
	return nil
}

// Type aliases for backward compatibility
type (
	PerformLexicalAnalysisCommand  = appcommands.PerformLexicalAnalysisCommand
	LexicalAnalysisResult          = domaincommands.LexicalAnalysisResult
	PerformSyntaxAnalysisCommand   = domaincommands.PerformSyntaxAnalysisCommand
	SyntaxAnalysisResult           = domaincommands.SyntaxAnalysisResult
	PerformSemanticAnalysisCommand = domaincommands.PerformSemanticAnalysisCommand
	SemanticAnalysisResult         = domaincommands.SemanticAnalysisResult
	HandleCompilationErrorsCommand = domaincommands.HandleCompilationErrorsCommand
	ErrorHandlingResult            = domaincommands.ErrorHandlingResult
	GetAnalysisStatusQuery         = domaincommands.GetAnalysisStatusQuery
	AnalysisStatusDTO              = domaincommands.AnalysisStatusDTO
	CompilationResult              = dtos.CompilationResult
)

// Legacy type definitions (keeping for backward compatibility)
// These will be removed once all references are updated to use commands/dtos packages

type Token struct {
	Type     TokenType
	Lexeme   string
	Position Position
}

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdentifier
	TokenNumber
	TokenString
	TokenOperator
	TokenKeyword
)

type Position struct {
	Line   int
	Column int
	File   string
}