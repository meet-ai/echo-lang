// Package runtime: RuntimeService 占位实现，使 module 可编译。
package runtime

import "context"

// runtimeService 实现 RuntimeService 接口（占位）
type runtimeService struct {
	executor      Executor
	memoryManager MemoryManager
	gcController  GCController
}

// NewRuntimeService 创建运行时应用服务（占位）
func NewRuntimeService(executor Executor, memoryManager MemoryManager, gcController GCController) RuntimeService {
	return &runtimeService{
		executor:      executor,
		memoryManager: memoryManager,
		gcController:  gcController,
	}
}

// ExecuteCode 执行代码（占位）
func (s *runtimeService) ExecuteCode(ctx context.Context, cmd ExecuteCodeCommand) (*ExecutionResult, error) {
	if s.executor == nil {
		return &ExecutionResult{ExitCode: -1}, nil
	}
	_, err := s.executor.Execute(ctx, cmd.Code)
	if err != nil {
		return &ExecutionResult{Stderr: err.Error(), ExitCode: 1}, nil
	}
	return &ExecutionResult{ExitCode: 0}, nil
}

// ManageMemory 内存管理（占位）
func (s *runtimeService) ManageMemory(ctx context.Context, cmd ManageMemoryCommand) (*MemoryManagementResult, error) {
	return &MemoryManagementResult{}, nil
}

// MonitorResources 资源监控（占位）
func (s *runtimeService) MonitorResources(ctx context.Context, cmd MonitorResourcesCommand) (*ResourceMonitoringResult, error) {
	return &ResourceMonitoringResult{Usage: &ResourceUsage{}}, nil
}
