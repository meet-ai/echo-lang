package services

import (
	"context"

	"github.com/meetai/echo-lang/internal/modules/frontend"
)

// ITokenizerService defines the interface for tokenization and analysis status queries.
// This interface provides read-only operations for analysis results.
type ITokenizerService interface {
	// GetAnalysisStatus retrieves the current analysis status
	GetAnalysisStatus(ctx context.Context, query frontend.GetAnalysisStatusQuery) (*frontend.AnalysisStatusDTO, error)

	// GetASTStructure retrieves the AST structure for a given source file
	GetASTStructure(ctx context.Context, query frontend.GetASTStructureQuery) (*frontend.ASTStructureDTO, error)
}
