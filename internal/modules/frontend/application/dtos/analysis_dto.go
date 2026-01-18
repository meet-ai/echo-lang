package dtos

import (
	"time"
)

// AnalysisStatusDTO 分析状态数据传输对象
type AnalysisStatusDTO struct {
	SourceFileID   string     `json:"source_file_id"`
	FilePath       string     `json:"file_path"`
	AnalysisStatus string     `json:"analysis_status"`
	TokenCount     int        `json:"token_count"`
	HasAST         bool       `json:"has_ast"`
	HasSymbolTable bool       `json:"has_symbol_table"`
	ErrorCount     int        `json:"error_count"`
	Errors         []string   `json:"errors,omitempty"`
	LastAnalyzedAt *time.Time `json:"last_analyzed_at,omitempty"`
}

// ASTStructureDTO AST结构数据传输对象
type ASTStructureDTO struct {
	NodeType string                 `json:"node_type"`
	Details  map[string]interface{} `json:"details,omitempty"`
	Children []ASTStructureDTO      `json:"children,omitempty"`
}

// LexicalAnalysisResultDTO 词法分析结果DTO
type LexicalAnalysisResultDTO struct {
	SourceFileID string        `json:"source_file_id"`
	TokenCount   int           `json:"token_count"`
	Success      bool          `json:"success"`
	Duration     time.Duration `json:"duration"`
	Tokens       []TokenDTO    `json:"tokens,omitempty"`
	Errors       []ErrorDTO    `json:"errors,omitempty"`
}

// SyntaxAnalysisResultDTO 语法分析结果DTO
type SyntaxAnalysisResultDTO struct {
	SourceFileID string           `json:"source_file_id"`
	Success      bool             `json:"success"`
	Duration     time.Duration    `json:"duration"`
	AST          *ASTStructureDTO `json:"ast,omitempty"`
	Errors       []ErrorDTO       `json:"errors,omitempty"`
}

// SemanticAnalysisResultDTO 语义分析结果DTO
type SemanticAnalysisResultDTO struct {
	SourceFileID string        `json:"source_file_id"`
	Success      bool          `json:"success"`
	Duration     time.Duration `json:"duration"`
	SymbolCount  int           `json:"symbol_count"`
	Errors       []ErrorDTO    `json:"errors,omitempty"`
}

// ErrorHandlingResultDTO 错误处理结果DTO
type ErrorHandlingResultDTO struct {
	SourceFileID string   `json:"source_file_id"`
	ErrorCount   int      `json:"error_count"`
	Suggestions  []string `json:"suggestions"`
	FixedCount   int      `json:"fixed_count"`
}

// TokenDTO Token数据传输对象
type TokenDTO struct {
	Type    string      `json:"type"`
	Lexeme  string      `json:"lexeme"`
	Line    int         `json:"line"`
	Column  int         `json:"column"`
	Offset  int         `json:"offset"`
	Literal interface{} `json:"literal,omitempty"`
}

// ErrorDTO 错误数据传输对象
type ErrorDTO struct {
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Offset   int    `json:"offset"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // "error", "warning", "info"
}
