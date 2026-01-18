package commands

import (
	"time"
)

// OptimizeIRCommand 优化IR命令
type OptimizeIRCommand struct {
	IRCode        string   `json:"ir_code" validate:"required"`
	Optimizations []string `json:"optimizations"` // 优化选项，如 ["inline", "dce", "licm"]
	TargetFormat  string   `json:"target_format" validate:"required"`
}

// OptimizationResult 优化结果
type OptimizationResult struct {
	OriginalIR    string        `json:"original_ir"`
	OptimizedIR   string        `json:"optimized_ir"`
	Optimizations []string      `json:"applied_optimizations"`
	Success       bool          `json:"success"`
	Error         string        `json:"error,omitempty"`
	Duration      time.Duration `json:"duration"`
	Analysis      OptimizationAnalysis `json:"analysis"`
}

// OptimizationAnalysis 优化分析
type OptimizationAnalysis struct {
	SizeReduction     float64 `json:"size_reduction"`     // 大小减少百分比
	PerformanceGain   float64 `json:"performance_gain"`   // 性能提升估算
	CodeComplexity    int     `json:"code_complexity"`    // 代码复杂度变化
	OptimizationCount int     `json:"optimization_count"` // 应用的优化数量
}

// ApplyOptimizationsCommand 应用优化命令
type ApplyOptimizationsCommand struct {
	IRCode        string   `json:"ir_code" validate:"required"`
	Optimizations []string `json:"optimizations"`
	Priority      string   `json:"priority"` // "speed", "size", "balanced"
}

// GetOptimizationStatusQuery 获取优化状态查询
type GetOptimizationStatusQuery struct {
	RequestID string `json:"request_id" validate:"required"`
}

// OptimizationStatusDTO 优化状态DTO
type OptimizationStatusDTO struct {
	RequestID      string    `json:"request_id"`
	Status         string    `json:"status"` // "pending", "processing", "completed", "failed"
	Progress       int       `json:"progress"` // 0-100
	Error          string    `json:"error,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	AppliedOptimizations []string `json:"applied_optimizations"`
}
