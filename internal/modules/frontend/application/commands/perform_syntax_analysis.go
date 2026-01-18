package commands

import (
	"time"
)

// PerformSyntaxAnalysisCommand 执行语法分析命令
type PerformSyntaxAnalysisCommand struct {
	// 源文件标识
	SourceFileID string `json:"source_file_id" validate:"required"`

	// 分析选项
	EnableErrorRecovery bool     `json:"enable_error_recovery"` // 是否启用错误恢复
	MaxErrors           int      `json:"max_errors"`            // 最大错误数量
	StrictMode          bool     `json:"strict_mode"`           // 严格模式
	AdditionalOptions   []string `json:"additional_options"`    // 额外选项

	// 元数据
	RequestID string    `json:"request_id"`
	UserID    string    `json:"user_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// NewPerformSyntaxAnalysisCommand 创建语法分析命令
func NewPerformSyntaxAnalysisCommand(sourceFileID string) *PerformSyntaxAnalysisCommand {
	return &PerformSyntaxAnalysisCommand{
		SourceFileID:        sourceFileID,
		EnableErrorRecovery: true,  // 默认启用错误恢复
		MaxErrors:           10,    // 默认最大10个错误
		StrictMode:          false, // 默认非严格模式
		AdditionalOptions:   []string{},
		Timestamp:           time.Now(),
	}
}

// CommandType 返回命令类型
func (c *PerformSyntaxAnalysisCommand) CommandType() string {
	return "frontend.perform_syntax_analysis"
}

// Validate 验证命令
func (c *PerformSyntaxAnalysisCommand) Validate() error {
	if c.SourceFileID == "" {
		return NewValidationError("source_file_id", "is required")
	}
	if c.MaxErrors < 0 {
		return NewValidationError("max_errors", "must be non-negative")
	}
	return nil
}
