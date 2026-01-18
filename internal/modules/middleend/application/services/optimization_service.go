package services

import (
	"context"
	"fmt"
	"time"

	"echo/internal/modules/middleend/domain/commands"
	"echo/internal/modules/middleend/domain/repositories"
	"echo/internal/modules/middleend/domain/services"
)

// OptimizationService 优化服务应用层实现
type OptimizationService struct {
	optimizer        services.MachineIndependentOptimizer
	optimizationRepo repositories.OptimizationRepository
}

// NewOptimizationService 创建优化服务
func NewOptimizationService(
	optimizer services.MachineIndependentOptimizer,
	optimizationRepo repositories.OptimizationRepository,
) *OptimizationService {
	return &OptimizationService{
		optimizer:        optimizer,
		optimizationRepo: optimizationRepo,
	}
}

// ApplyOptimizations 应用优化用例
func (s *OptimizationService) ApplyOptimizations(ctx context.Context, cmd commands.ApplyOptimizationsCommand) (*commands.OptimizationResult, error) {
	startTime := time.Now()

	// 保存优化请求
	requestID := fmt.Sprintf("opt_%d", startTime.UnixNano())
	err := s.optimizationRepo.SaveOptimizationRequest(ctx, requestID, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to save optimization request: %w", err)
	}

	// 更新状态为处理中
	s.optimizationRepo.UpdateOptimizationStatus(ctx, requestID, "processing", 10, "")

	// 执行优化
	optimizedIR, err := s.optimizer.ApplyOptimizations(ctx, cmd.IRCode, cmd.Optimizations)
	if err != nil {
		// 更新状态为失败
		s.optimizationRepo.UpdateOptimizationStatus(ctx, requestID, "failed", 0, err.Error())
		return &commands.OptimizationResult{
			OriginalIR:    cmd.IRCode,
			OptimizedIR:   cmd.IRCode,
			Optimizations: cmd.Optimizations,
			Success:       false,
			Error:         err.Error(),
			Duration:      time.Since(startTime),
		}, nil
	}

	// 分析优化效果
	analysis, err := s.optimizer.AnalyzeOptimizationImpact(ctx, cmd.IRCode, optimizedIR)
	if err != nil {
		analysis = commands.OptimizationAnalysis{
			OptimizationCount: len(cmd.Optimizations),
		}
	}

	// 更新状态为成功
	s.optimizationRepo.UpdateOptimizationStatus(ctx, requestID, "completed", 100, "")

	return &commands.OptimizationResult{
		OriginalIR:    cmd.IRCode,
		OptimizedIR:   optimizedIR,
		Optimizations: cmd.Optimizations,
		Success:       true,
		Duration:      time.Since(startTime),
		Analysis:      analysis,
	}, nil
}

// GetOptimizationStatus 获取优化状态用例
func (s *OptimizationService) GetOptimizationStatus(ctx context.Context, query commands.GetOptimizationStatusQuery) (*commands.OptimizationStatusDTO, error) {
	return s.optimizationRepo.GetOptimizationStatus(ctx, query.RequestID)
}
