package commands

import (
	"time"

	"echo/internal/modules/frontend/domain/exceptions"
)

// PerformLexicalAnalysisCommand represents the command to perform lexical analysis
type PerformLexicalAnalysisCommand struct {
	// Source information
	SourceFileID   string `json:"source_file_id" validate:"required"`
	SourceCode     string `json:"source_code" validate:"required"`
	SourceFilePath string `json:"source_file_path" validate:"required"`

	// Analysis options
	IncludeComments bool     `json:"include_comments"` // Whether to include comments in tokens
	Encoding        string   `json:"encoding"`         // Source code encoding
	Options         []string `json:"options"`          // Additional lexer options

	// Metadata
	RequestID string    `json:"request_id"`
	UserID    string    `json:"user_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// NewPerformLexicalAnalysisCommand creates a new lexical analysis command
func NewPerformLexicalAnalysisCommand(sourceFileID, sourceCode, sourceFilePath string) *PerformLexicalAnalysisCommand {
	return &PerformLexicalAnalysisCommand{
		SourceFileID:    sourceFileID,
		SourceCode:      sourceCode,
		SourceFilePath:  sourceFilePath,
		IncludeComments: true, // Default to include comments
		Encoding:        "utf-8",
		Options:         []string{},
		Timestamp:       time.Now(),
	}
}

// CommandType returns the command type
func (c *PerformLexicalAnalysisCommand) CommandType() string {
	return "frontend.perform_lexical_analysis"
}

// Validate validates the command
func (c *PerformLexicalAnalysisCommand) Validate() error {
	if c.SourceFileID == "" {
		return exceptions.ErrSourceFileIDRequired
	}
	if c.SourceCode == "" {
		return exceptions.ErrSourceCodeRequired
	}
	if c.SourceFilePath == "" {
		return exceptions.ErrSourceFilePathRequired
	}
	return nil
}
