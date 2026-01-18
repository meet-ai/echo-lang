package services

import (
	"context"
	"fmt"
	"time"

	"echo/internal/modules/middleend/domain/commands"
	"echo/internal/modules/middleend/domain/repositories"
	"echo/internal/modules/middleend/domain/services"
)

// IRService IR服务应用层实现
type IRService struct {
	irGenerator      services.IRGenerator
	optimizer        services.MachineIndependentOptimizer
	irRepo           repositories.IRRepository
	optimizationRepo repositories.OptimizationRepository
}

// NewIRService 创建IR服务
func NewIRService(
	irGenerator services.IRGenerator,
	optimizer services.MachineIndependentOptimizer,
	irRepo repositories.IRRepository,
	optimizationRepo repositories.OptimizationRepository,
) *IRService {
	return &IRService{
		irGenerator:      irGenerator,
		optimizer:        optimizer,
		irRepo:           irRepo,
		optimizationRepo: optimizationRepo,
	}
}

// GenerateIR 生成IR用例
func (s *IRService) GenerateIR(ctx context.Context, cmd commands.GenerateIRCommand) (*commands.IRGenerationResult, error) {
	startTime := time.Now()

	// 生成IR代码
	irCode, err := s.irGenerator.GenerateIR(ctx, cmd.Program)
	if err != nil {
		return &commands.IRGenerationResult{
			SourceFileID: cmd.SourceFileID,
			Success:      false,
			Error:        fmt.Sprintf("failed to generate IR: %v", err),
			Duration:     time.Since(startTime),
		}, nil
	}

	// 如果需要优化
	if cmd.Optimization {
		optimizedIR, err := s.irGenerator.OptimizeIR(ctx, irCode)
		if err != nil {
			// 优化失败不影响整体结果，使用原始IR
			fmt.Printf("Warning: IR optimization failed: %v\n", err)
		} else {
			irCode = optimizedIR
		}
	}

	// 保存IR到仓储
	err = s.irRepo.Save(ctx, cmd.SourceFileID, irCode)
	if err != nil {
		return &commands.IRGenerationResult{
			SourceFileID: cmd.SourceFileID,
			Success:      false,
			Error:        fmt.Sprintf("failed to save IR: %v", err),
			Duration:     time.Since(startTime),
		}, nil
	}

	return &commands.IRGenerationResult{
		SourceFileID: cmd.SourceFileID,
		IRCode:       irCode,
		TargetFormat: cmd.TargetFormat,
		Success:      true,
		Duration:     time.Since(startTime),
		Size:         len(irCode),
	}, nil
}

// OptimizeIR 优化IR用例
func (s *IRService) OptimizeIR(ctx context.Context, cmd commands.OptimizeIRCommand) (*commands.OptimizationResult, error) {
	startTime := time.Now()

	// 保存优化请求
	requestID := fmt.Sprintf("opt_%d", startTime.UnixNano())
	err := s.optimizationRepo.SaveOptimizationRequest(ctx, requestID, commands.ApplyOptimizationsCommand{
		IRCode:        cmd.IRCode,
		Optimizations: cmd.Optimizations,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save optimization request: %w", err)
	}

	// 执行优化
	optimizedIR, err := s.optimizer.ApplyOptimizations(ctx, cmd.IRCode, cmd.Optimizations)
	if err != nil {
		// 更新状态为失败
		s.optimizationRepo.UpdateOptimizationStatus(ctx, requestID, "failed", 0, err.Error())
		return &commands.OptimizationResult{
			OriginalIR:    cmd.IRCode,
			OptimizedIR:   cmd.IRCode, // 返回原始IR
			Optimizations: cmd.Optimizations,
			Success:       false,
			Error:         err.Error(),
			Duration:      time.Since(startTime),
		}, nil
	}

	// 分析优化效果
	analysis, err := s.optimizer.AnalyzeOptimizationImpact(ctx, cmd.IRCode, optimizedIR)
	if err != nil {
		// 分析失败但优化成功
		analysis = commands.OptimizationAnalysis{
			OptimizationCount: len(cmd.Optimizations),
		}
	}

	// 更新状态为成功
	s.optimizationRepo.UpdateOptimizationStatus(ctx, requestID, "completed", 100, "")

	result := &commands.OptimizationResult{
		OriginalIR:    cmd.IRCode,
		OptimizedIR:   optimizedIR,
		Optimizations: cmd.Optimizations,
		Success:       true,
		Duration:      time.Since(startTime),
		Analysis:      analysis,
	}

	return result, nil
}
