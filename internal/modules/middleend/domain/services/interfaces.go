package services

import (
	"context"

	"echo/internal/modules/frontend/domain/entities"
	"echo/internal/modules/middleend/domain/commands"
)

// IRGenerator 中间表示生成器领域服务接口
// 职责：将 AST 转换为中间表示（IR）
type IRGenerator interface {
	// GenerateIR 生成中间表示
	GenerateIR(ctx context.Context, program *entities.Program) (string, error)

	// ValidateIR 验证 IR 的正确性
	ValidateIR(ctx context.Context, irCode string) error

	// OptimizeIR 进行基础优化
	OptimizeIR(ctx context.Context, irCode string) (string, error)
}

// MachineIndependentOptimizer 机器无关优化器领域服务接口
// 职责：执行不依赖特定硬件的优化
type MachineIndependentOptimizer interface {
	// ApplyOptimizations 应用优化
	ApplyOptimizations(ctx context.Context, irCode string, optimizations []string) (string, error)

	// AnalyzeOptimizationImpact 分析优化效果
	AnalyzeOptimizationImpact(ctx context.Context, originalIR, optimizedIR string) (commands.OptimizationAnalysis, error)

	// GetAvailableOptimizations 获取可用的优化选项
	GetAvailableOptimizations() []string
}

// OptimizationAnalysis 类型在 commands 包中定义
