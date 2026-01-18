package commands

import (
	"time"

	"echo/internal/modules/frontend/domain/entities"
)

// GenerateIRCommand 生成IR命令
type GenerateIRCommand struct {
	SourceFileID string              `json:"source_file_id" validate:"required"`
	Program      *entities.Program   `json:"program" validate:"required"`
	TargetFormat string              `json:"target_format" validate:"required"` // "llvm", "wasm", etc.
	Optimization bool                `json:"optimization"`
}

// IRGenerationResult IR生成结果
type IRGenerationResult struct {
	SourceFileID string        `json:"source_file_id"`
	IRCode       string        `json:"ir_code"`
	TargetFormat string        `json:"target_format"`
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
	Duration     time.Duration `json:"duration"`
	Size         int           `json:"size"` // IR代码大小（字节）
}
