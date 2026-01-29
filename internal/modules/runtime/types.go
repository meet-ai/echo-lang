// Package runtime: 占位类型定义，使 module 可编译；随架构演进补全接口与实现。
package runtime

import "context"

// Executor 执行器接口（占位）
type Executor interface {
	Execute(ctx context.Context, code string) ([]byte, error)
}

// MemoryManager 内存管理接口（占位）
type MemoryManager interface {
	Allocate(size uintptr) (uintptr, error)
	Free(ptr uintptr) error
}

// GCController GC 控制接口（占位）
type GCController interface {
	TriggerGC(ctx context.Context) error
}

// ProcessManager 进程管理接口（占位）
type ProcessManager interface {
	StartProcess(ctx context.Context, name string, args []string) (int, error)
	StopProcess(ctx context.Context, pid int) error
}

// ResourceMonitor 资源监控接口（占位）
type ResourceMonitor interface {
	GetUsage(ctx context.Context) (*ResourceUsage, error)
}

// ResourceUsage 资源使用情况（占位）
type ResourceUsage struct {
	MemoryMB float64
	CPUPct   float64
}

// ExecuteCodeCommand 执行代码命令（占位）
type ExecuteCodeCommand struct {
	Code string
	Lang string
}

// ExecutionResult 执行结果（占位）
type ExecutionResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// ManageMemoryCommand 内存管理命令（占位）
type ManageMemoryCommand struct {
	Action string
	Size   uintptr
}

// MemoryManagementResult 内存管理结果（占位）
type MemoryManagementResult struct {
	Allocated uintptr
	Freed     uintptr
}

// MonitorResourcesCommand 资源监控命令（占位）
type MonitorResourcesCommand struct {
	Pid int
}

// ResourceMonitoringResult 资源监控结果（占位）
type ResourceMonitoringResult struct {
	Usage *ResourceUsage
}
