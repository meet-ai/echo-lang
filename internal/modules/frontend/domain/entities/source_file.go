package entities

import (
	"fmt"
	"time"

	"github.com/meetai/echo-lang/internal/modules/frontend/domain/valueobjects"
)

// SourceFile represents a source file entity in the frontend domain.
// This is the aggregate root for frontend processing.
type SourceFile struct {
	// Identity
	id          string
	filePath    string
	content     string
	contentHash string

	// Metadata
	fileSize   int64
	modifiedAt time.Time
	createdAt  time.Time

	// Processing state
	analysisStatus AnalysisStatus
	lastAnalyzedAt *time.Time
	errorMessages  []string

	// Analysis results
	tokens      []valueobjects.Token
	ast         *ASTNode
	symbolTable *SymbolTable

	// Domain events
	domainEvents []interface{}
}

// NewSourceFile creates a new source file entity
func NewSourceFile(id, filePath, content string) (*SourceFile, error) {
	if id == "" {
		return nil, ErrInvalidSourceFileID
	}
	if filePath == "" {
		return nil, ErrInvalidFilePath
	}

	now := time.Now()
	contentHash := calculateContentHash(content)

	return &SourceFile{
		id:             id,
		filePath:       filePath,
		content:        content,
		contentHash:    contentHash,
		fileSize:       int64(len(content)),
		createdAt:      now,
		modifiedAt:     now,
		analysisStatus: AnalysisStatusPending,
		tokens:         make([]valueobjects.Token, 0),
		errorMessages:  make([]string, 0),
		domainEvents:   make([]interface{}, 0),
	}, nil
}

// ID returns the source file ID
func (sf *SourceFile) ID() string {
	return sf.id
}

// FilePath returns the file path
func (sf *SourceFile) FilePath() string {
	return sf.filePath
}

// Content returns the file content
func (sf *SourceFile) Content() string {
	return sf.content
}

// ContentHash returns the content hash
func (sf *SourceFile) ContentHash() string {
	return sf.contentHash
}

// AnalysisStatus returns the current analysis status
func (sf *SourceFile) AnalysisStatus() AnalysisStatus {
	return sf.analysisStatus
}

// SetAnalysisStatus sets the analysis status
func (sf *SourceFile) SetAnalysisStatus(status AnalysisStatus) {
	if sf.analysisStatus != status {
		sf.analysisStatus = status
		sf.lastAnalyzedAt = &time.Time{}
		*sf.lastAnalyzedAt = time.Now()
	}
}

// AddErrorMessage adds an error message
func (sf *SourceFile) AddErrorMessage(message string) {
	sf.errorMessages = append(sf.errorMessages, message)
}

// ClearErrorMessages clears all error messages
func (sf *SourceFile) ClearErrorMessages() {
	sf.errorMessages = make([]string, 0)
}

// ErrorMessages returns all error messages
func (sf *SourceFile) ErrorMessages() []string {
	return append([]string(nil), sf.errorMessages...) // Return a copy
}

// SetTokens sets the tokens from lexical analysis
func (sf *SourceFile) SetTokens(tokens []valueobjects.Token) {
	sf.tokens = make([]valueobjects.Token, len(tokens))
	copy(sf.tokens, tokens)
}

// Tokens returns the tokens
func (sf *SourceFile) Tokens() []valueobjects.Token {
	return append([]valueobjects.Token(nil), sf.tokens...) // Return a copy
}

// SetAST sets the AST from syntax analysis
func (sf *SourceFile) SetAST(ast *ASTNode) {
	sf.ast = ast
}

// AST returns the AST
func (sf *SourceFile) AST() *ASTNode {
	return sf.ast
}

// SetSymbolTable sets the symbol table from semantic analysis
func (sf *SourceFile) SetSymbolTable(symbolTable *SymbolTable) {
	sf.symbolTable = symbolTable
}

// SymbolTable returns the symbol table
func (sf *SourceFile) SymbolTable() *SymbolTable {
	return sf.symbolTable
}

// DomainEvents returns and clears the domain events
func (sf *SourceFile) DomainEvents() []interface{} {
	events := make([]interface{}, len(sf.domainEvents))
	copy(events, sf.domainEvents)
	sf.domainEvents = make([]interface{}, 0)
	return events
}

// UpdateContent updates the source file content
func (sf *SourceFile) UpdateContent(newContent string) error {
	newHash := calculateContentHash(newContent)

	// Only update if content actually changed
	if newHash != sf.contentHash {
		sf.content = newContent
		sf.contentHash = newHash
		sf.fileSize = int64(len(newContent))
		sf.modifiedAt = time.Now()

		// Reset analysis state
		sf.analysisStatus = AnalysisStatusPending
		sf.tokens = make([]valueobjects.Token, 0)
		sf.ast = nil
		sf.symbolTable = nil
		sf.ClearErrorMessages()
	}

	return nil
}

// calculateContentHash calculates a simple hash of the content
func calculateContentHash(content string) string {
	// Simple hash implementation - in real implementation use crypto/sha256
	hash := 0
	for _, char := range content {
		hash = hash*31 + int(char)
	}
	return string(rune(hash)) // Simplified for demonstration
}

// AnalysisStatus represents the analysis status of a source file
type AnalysisStatus string

const (
	AnalysisStatusPending  AnalysisStatus = "pending"
	AnalysisStatusLexical  AnalysisStatus = "lexical_completed"
	AnalysisStatusSyntax   AnalysisStatus = "syntax_completed"
	AnalysisStatusSemantic AnalysisStatus = "semantic_completed"
	AnalysisStatusFailed   AnalysisStatus = "failed"
)

// Errors
var (
	ErrInvalidSourceFileID = fmt.Errorf("invalid source file ID")
	ErrInvalidFilePath     = fmt.Errorf("invalid file path")
)
