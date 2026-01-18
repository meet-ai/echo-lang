package services

import (
	"context"

	"echo/internal/modules/frontend/domain/commands"
)

// ITokenizerService defines the interface for tokenization and analysis status queries.
// This interface provides read-only operations for analysis results.
type ITokenizerService interface {
	// GetAnalysisStatus retrieves the current analysis status
	GetAnalysisStatus(ctx context.Context, query commands.GetAnalysisStatusQuery) (*commands.AnalysisStatusDTO, error)

	// GetASTStructure retrieves the AST structure for a given source file
	GetASTStructure(ctx context.Context, query commands.GetASTStructureQuery) (*commands.ASTStructureDTO, error)
}
