// Package frontend provides the frontend processing context for Echo Language compilation.
// This module handles lexical analysis, syntax analysis, and semantic analysis of .eo source files.
package frontend

import (
	"fmt"

	"echo/internal/modules/frontend/domain/commands"
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
	PerformLexicalAnalysisCommand  = commands.PerformLexicalAnalysisCommand
	LexicalAnalysisResult          = commands.LexicalAnalysisResult
	PerformSyntaxAnalysisCommand   = commands.PerformSyntaxAnalysisCommand
	SyntaxAnalysisResult           = commands.SyntaxAnalysisResult
	PerformSemanticAnalysisCommand = commands.PerformSemanticAnalysisCommand
	SemanticAnalysisResult         = commands.SemanticAnalysisResult
	HandleCompilationErrorsCommand = commands.HandleCompilationErrorsCommand
	ErrorHandlingResult            = commands.ErrorHandlingResult
	GetAnalysisStatusQuery         = commands.GetAnalysisStatusQuery
	AnalysisStatusDTO              = commands.AnalysisStatusDTO
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