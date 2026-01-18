// Package middleend provides the middle-end processing context for Echo Language compilation.
// This module handles intermediate representation generation and machine-independent optimizations.
package middleend

import (
	"context"
	"fmt"

	applicationServices "echo/internal/modules/middleend/application/services"
	"echo/internal/modules/middleend/domain/adapters"
	"echo/internal/modules/middleend/domain/commands"
	"echo/internal/modules/middleend/domain/repositories"
	domainServices "echo/internal/modules/middleend/domain/services"

	"github.com/samber/do"
)

// 类型别名，引用domain层的类型
type GenerateIRCommand = commands.GenerateIRCommand
type IRGenerationResult = commands.IRGenerationResult
type OptimizeIRCommand = commands.OptimizeIRCommand
type OptimizationResult = commands.OptimizationResult
type ApplyOptimizationsCommand = commands.ApplyOptimizationsCommand
type GetOptimizationStatusQuery = commands.GetOptimizationStatusQuery
type OptimizationStatusDTO = commands.OptimizationStatusDTO

// 类型别名，引用domain层服务和接口
type IRGenerator = domainServices.IRGenerator
type MachineIndependentOptimizer = domainServices.MachineIndependentOptimizer
type IRRepository = repositories.IRRepository
type OptimizationRepository = repositories.OptimizationRepository
type LLVMIRAdapter = adapters.LLVMIRAdapter

// Module represents the middle-end processing module
type Module struct {
	// Ports - external interfaces
	irService  IRService
	optService OptimizationService

	// Domain services
	irGenerator IRGenerator
	optimizer   MachineIndependentOptimizer

	// Infrastructure
	irRepo           IRRepository
	optimizationRepo OptimizationRepository

	// Adapters
	llvmAdapter LLVMIRAdapter

	// Dependencies from other modules
	frontendModule interface{} // Will be injected
}

// IRService defines the interface for IR generation operations
type IRService interface {
	GenerateIR(ctx context.Context, cmd GenerateIRCommand) (*IRGenerationResult, error)
	OptimizeIR(ctx context.Context, cmd OptimizeIRCommand) (*OptimizationResult, error)
}

// OptimizationService defines the interface for optimization operations
type OptimizationService interface {
	ApplyOptimizations(ctx context.Context, cmd ApplyOptimizationsCommand) (*OptimizationResult, error)
	GetOptimizationStatus(ctx context.Context, query GetOptimizationStatusQuery) (*OptimizationStatusDTO, error)
}

// NewModule creates a new middle-end module with dependency injection
func NewModule(i *do.Injector) (*Module, error) {
	// Get domain services
	irGenerator := do.MustInvoke[IRGenerator](i)
	optimizer := do.MustInvoke[MachineIndependentOptimizer](i)

	// Get repositories
	irRepo := do.MustInvoke[IRRepository](i)
	optimizationRepo := do.MustInvoke[OptimizationRepository](i)

	// Get infrastructure adapters
	llvmAdapter := do.MustInvoke[LLVMIRAdapter](i)

	// Create application services
	irSvc := applicationServices.NewIRService(irGenerator, optimizer, irRepo, optimizationRepo)
	optSvc := applicationServices.NewOptimizationService(optimizer, optimizationRepo)

	return &Module{
		irService:        irSvc,
		optService:       optSvc,
		irGenerator:      irGenerator,
		optimizer:        optimizer,
		irRepo:           irRepo,
		optimizationRepo: optimizationRepo,
		llvmAdapter:      llvmAdapter,
	}, nil
}

// IRService returns the IR service interface
func (m *Module) IRService() IRService {
	return m.irService
}

// OptimizationService returns the optimization service interface
func (m *Module) OptimizationService() OptimizationService {
	return m.optService
}

// Validate validates the module configuration
func (m *Module) Validate() error {
	if m.irService == nil {
		return fmt.Errorf("IR service is not initialized")
	}
	if m.optService == nil {
		return fmt.Errorf("optimization service is not initialized")
	}
	if m.llvmAdapter == nil {
		return fmt.Errorf("LLVM adapter is not initialized")
	}
	return nil
}
