package repositories

import (
	"context"

	"echo/internal/modules/frontend/domain/entities"
)

// SourceFileRepository defines the interface for source file persistence operations.
// This repository handles the persistence of SourceFile aggregate roots.
type SourceFileRepository interface {
	// Save saves a source file to the repository
	Save(ctx context.Context, sourceFile *entities.SourceFile) error

	// FindByID finds a source file by its ID
	FindByID(ctx context.Context, id string) (*entities.SourceFile, error)

	// FindByPath finds a source file by its file path
	FindByPath(ctx context.Context, filePath string) (*entities.SourceFile, error)

	// FindByStatus finds source files by their analysis status
	FindByStatus(ctx context.Context, status entities.AnalysisStatus) ([]*entities.SourceFile, error)

	// Update updates an existing source file
	Update(ctx context.Context, sourceFile *entities.SourceFile) error

	// Delete deletes a source file by its ID
	Delete(ctx context.Context, id string) error

	// Exists checks if a source file exists by its ID
	Exists(ctx context.Context, id string) (bool, error)

	// ListAll lists all source files with pagination
	ListAll(ctx context.Context, offset, limit int) ([]*entities.SourceFile, error)

	// Count returns the total count of source files
	Count(ctx context.Context) (int64, error)
}
