package dtos

import (
	"time"

	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
)

// ModernCompilationResult 现代化编译结果DTO
// 包含使用三层混合解析器架构的解析结果
type ModernCompilationResult struct {
	Success      bool                      `json:"success"`
	Filename     string                    `json:"filename"`
	ProgramAST   *sharedVO.ProgramAST      `json:"program_ast,omitempty"`
	Errors       []*sharedVO.ParseError    `json:"errors,omitempty"`
	Duration     time.Duration             `json:"duration"`
	ParserType   string                    `json:"parser_type"` // "modern_hybrid"
	ParseDetails *ParseDetails             `json:"parse_details,omitempty"`
}

// ParseDetails 解析详情
// 包含解析过程的详细信息
type ParseDetails struct {
	ParserType        string `json:"parser_type"`         // 当前使用的解析器类型
	CurrentParserType string `json:"current_parser_type"` // 当前解析器类型（递归下降/Pratt/LR）
	StateStackDepth   int    `json:"state_stack_depth"`   // 状态栈深度
	InRecoveryMode    bool   `json:"in_recovery_mode"`    // 是否在恢复模式
}

