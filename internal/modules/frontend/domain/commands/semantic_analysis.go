package commands

import "time"

// PerformSemanticAnalysisCommand 语义分析命令
type PerformSemanticAnalysisCommand struct {
	SourceFileID string `json:"source_file_id" validate:"required"`
}

// SemanticAnalysisResult 语义分析结果
type SemanticAnalysisResult struct {
	SourceFileID string        `json:"source_file_id"`
	Success      bool          `json:"success"`
	Duration     time.Duration `json:"duration"`
}
