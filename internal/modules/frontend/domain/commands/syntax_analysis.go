package commands

import "time"

// PerformSyntaxAnalysisCommand 语法分析命令
type PerformSyntaxAnalysisCommand struct {
	SourceFileID string `json:"source_file_id" validate:"required"`
}

// SyntaxAnalysisResult 语法分析结果
type SyntaxAnalysisResult struct {
	SourceFileID string        `json:"source_file_id"`
	Success      bool          `json:"success"`
	Duration     time.Duration `json:"duration"`
}

