package commands

import "time"

// HandleCompilationErrorsCommand 处理编译错误命令
type HandleCompilationErrorsCommand struct {
	SourceFileID string   `json:"source_file_id" validate:"required"`
	Errors       []string `json:"errors" validate:"required"`
}

// ErrorHandlingResult 错误处理结果
type ErrorHandlingResult struct {
	SourceFileID   string        `json:"source_file_id"`
	Suggestions    []string      `json:"suggestions"`
	ProcessedCount int           `json:"processed_count"`
	Success        bool          `json:"success"`
	Duration       time.Duration `json:"duration"`
}
