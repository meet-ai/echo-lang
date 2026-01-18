// Package runtime provides runtime execution capabilities for Echo Language.
// This module handles code execution, memory management, and runtime services.
package runtime

import (
	"context"
	"fmt"

	"github.com/samber/do"
)

// Module represents the runtime execution module
type Module struct {
	// Ports - external interfaces
	runtimeService RuntimeService

	// Domain services
	executor      Executor
	memoryManager MemoryManager
	gcController  GCController

	// Infrastructure
	processManager ProcessManager
	resourceMonitor ResourceMonitor
}

// RuntimeService defines the interface for runtime execution operations
type RuntimeService interface {
	ExecuteCode(ctx context.Context, cmd ExecuteCodeCommand) (*ExecutionResult, error)
	ManageMemory(ctx context.Context, cmd ManageMemoryCommand) (*MemoryManagementResult, error)
	MonitorResources(ctx context.Context, cmd MonitorResourcesCommand) (*ResourceMonitoringResult, error)
}

// NewModule creates a new runtime module with dependency injection
func NewModule(i *do.Injector) (*Module, error) {
	// Get dependencies from DI container
	executor := do.MustInvoke[Executor](i)
	memoryManager := do.MustInvoke[MemoryManager](i)
	gcController := do.MustInvoke[GCController](i)
	processManager := do.MustInvoke[ProcessManager](i)
	resourceMonitor := do.MustInvoke[ResourceMonitor](i)

	// Create application services
	runtimeSvc := NewRuntimeService(executor, memoryManager, gcController)

	return &Module{
		runtimeService: runtimeSvc,
		executor:       executor,
		memoryManager:  memoryManager,
		gcController:   gcController,
		processManager: processManager,
		resourceMonitor: resourceMonitor,
	}, nil
}

// RuntimeService returns the runtime service interface
func (m *Module) RuntimeService() RuntimeService {
	return m.runtimeService
}

// Validate validates the module configuration
func (m *Module) Validate() error {
	if m.runtimeService == nil {
		return fmt.Errorf("runtime service is not initialized")
	}
	if m.processManager == nil {
		return fmt.Errorf("process manager is not initialized")
	}
	if m.resourceMonitor == nil {
		return fmt.Errorf("resource monitor is not initialized")
	}
	return nil
}
