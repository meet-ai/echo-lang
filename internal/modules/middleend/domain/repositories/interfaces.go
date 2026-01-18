package repositories

import (
	"context"

	"echo/internal/modules/middleend/domain/commands"
)

// IRRepository IR仓储接口
// 职责：管理中间表示的存储和检索
type IRRepository interface {
	// Save 保存IR代码
	Save(ctx context.Context, sourceFileID string, irCode string) error

	// FindBySourceFileID 根据源文件ID查找IR代码
	FindBySourceFileID(ctx context.Context, sourceFileID string) (string, error)

	// Update 更新IR代码
	Update(ctx context.Context, sourceFileID string, irCode string) error

	// Delete 删除IR代码
	Delete(ctx context.Context, sourceFileID string) error

	// Exists 检查IR是否存在
	Exists(ctx context.Context, sourceFileID string) (bool, error)

	// ListByStatus 按状态列出IR
	ListByStatus(ctx context.Context, status string) ([]string, error)
}

// OptimizationRepository 优化仓储接口
// 职责：管理优化任务的状态和结果
type OptimizationRepository interface {
	// SaveOptimizationRequest 保存优化请求
	SaveOptimizationRequest(ctx context.Context, requestID string, cmd commands.ApplyOptimizationsCommand) error

	// UpdateOptimizationStatus 更新优化状态
	UpdateOptimizationStatus(ctx context.Context, requestID string, status string, progress int, errorMsg string) error

	// FindOptimizationResult 查找优化结果
	FindOptimizationResult(ctx context.Context, requestID string) (*commands.OptimizationResult, error)

	// GetOptimizationStatus 获取优化状态
	GetOptimizationStatus(ctx context.Context, requestID string) (*commands.OptimizationStatusDTO, error)

	// ListPendingOptimizations 列出待处理的优化任务
	ListPendingOptimizations(ctx context.Context) ([]string, error)
}
