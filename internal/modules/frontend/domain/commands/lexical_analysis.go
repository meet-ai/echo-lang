package commands

import "time"

// PerformLexicalAnalysisCommand 词法分析命令
type PerformLexicalAnalysisCommand struct {
	SourceFileID   string `json:"source_file_id" validate:"required"`
	SourceCode     string `json:"source_code" validate:"required"`
	SourceFilePath string `json:"source_file_path" validate:"required"`
}

// LexicalAnalysisResult 词法分析结果
type LexicalAnalysisResult struct {
	SourceFileID string        `json:"source_file_id"`
	TokenCount   int           `json:"token_count"`
	Success      bool          `json:"success"`
	Duration     time.Duration `json:"duration"`
}
