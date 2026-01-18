// Package backend provides the backend processing context for Echo Language compilation.
// This module handles target code generation, assembly, and linking.
package backend

import (
	"context"
	"fmt"
	"time"

	"echo/internal/modules/backend/domain/services"

	"github.com/samber/do"
)

// 类型别名，引用domain层定义的类型
type TargetCodeGenerator = services.TargetCodeGenerator
type Assembler = services.Assembler
type Linker = services.Linker
type ExecutableRepository = services.ExecutableRepository
type ObjectFileRepository = services.ObjectFileRepository
type ClangAdapter = services.ClangAdapter
type LLCAdapter = services.LLCAdapter
type LinkerAdapter = services.LinkerAdapter

// 类型别名，引用domain层的类型
type GenerateTargetCodeCommand = services.GenerateTargetCodeCommand
type CodeGenerationResult = services.CodeGenerationResult
type GetGenerationStatusQuery = services.GetGenerationStatusQuery
type GenerationStatusDTO = services.GenerationStatusDTO

// 组装相关类型（模块本地）
type AssembleCommand struct {
	IRCode    string
	Target    string
	Runtime   string
	OutputDir string
}

type AssemblyResult struct {
	ObjectFile string
	Success    bool
	Error      error
}

type GetAssemblyStatusQuery struct {
	RequestID string
}

type AssemblyStatusDTO struct {
	RequestID   string
	Status      string // "pending", "processing", "completed", "failed"
	Progress    int
	Error       string
	CompletedAt *time.Time
}

// 链接相关类型（模块本地）
type LinkCommand struct {
	ObjectFile string
	Runtime    string
	OutputDir  string
}

type LinkingResult struct {
	Executable string
	Success    bool
	Error      error
}

type GetLinkingStatusQuery struct {
	RequestID string
}

type LinkingStatusDTO struct {
	RequestID   string
	Status      string // "pending", "processing", "completed", "failed"
	Progress    int
	Error       string
	CompletedAt *time.Time
}

// Module represents the backend processing module
type Module struct {
	// Domain services (infrastructure implementations)
	CodeGenerator TargetCodeGenerator
	Assembler     Assembler
	Linker        Linker

	// Infrastructure
	ExecutableRepo ExecutableRepository
	ObjectFileRepo ObjectFileRepository

	// Adapters
	ClangAdapter ClangAdapter
	LLCAdapter   LLCAdapter
	LDAdapter    LinkerAdapter

	// Dependencies from other modules
	MiddleendModule interface{} // Will be injected
}

// CodeGenerationService defines the interface for code generation operations
type CodeGenerationService interface {
	GenerateTargetCode(ctx context.Context, cmd GenerateTargetCodeCommand) (*CodeGenerationResult, error)
	GetGenerationStatus(ctx context.Context, query GetGenerationStatusQuery) (*GenerationStatusDTO, error)
}

// AssemblyService defines the interface for assembly operations
type AssemblyService interface {
	Assemble(ctx context.Context, cmd AssembleCommand) (*AssemblyResult, error)
	GetAssemblyStatus(ctx context.Context, query GetAssemblyStatusQuery) (*AssemblyStatusDTO, error)
}

// LinkingService defines the interface for linking operations
type LinkingService interface {
	Link(ctx context.Context, cmd LinkCommand) (*LinkingResult, error)
	GetLinkingStatus(ctx context.Context, query GetLinkingStatusQuery) (*LinkingStatusDTO, error)
}

// NewModule creates a new backend module with dependency injection
func NewModule(i *do.Injector) (*Module, error) {
	// Get domain services
	codeGenerator := do.MustInvoke[TargetCodeGenerator](i)
	assembler := do.MustInvoke[Assembler](i)
	linker := do.MustInvoke[Linker](i)

	// Get repositories
	executableRepo := do.MustInvoke[ExecutableRepository](i)
	objectFileRepo := do.MustInvoke[ObjectFileRepository](i)

	// Get infrastructure adapters
	clangAdapter := do.MustInvoke[ClangAdapter](i)
	llcAdapter := do.MustInvoke[LLCAdapter](i)
	ldAdapter := do.MustInvoke[LinkerAdapter](i)

	// Note: Application services are created by the DI container
	// The module provides access to domain services and infrastructure

	// Validate that all required dependencies are available
	if codeGenerator == nil {
		return nil, fmt.Errorf("code generator is not available")
	}
	if assembler == nil {
		return nil, fmt.Errorf("assembler is not available")
	}
	if linker == nil {
		return nil, fmt.Errorf("linker is not available")
	}

	return &Module{
		CodeGenerator:   codeGenerator,
		Assembler:       assembler,
		Linker:          linker,
		ExecutableRepo:  executableRepo,
		ObjectFileRepo:  objectFileRepo,
		ClangAdapter:    clangAdapter,
		LLCAdapter:      llcAdapter,
		LDAdapter:       ldAdapter,
		MiddleendModule: nil,
	}, nil
}

// CodeGenerationService returns the code generation service interface
func (m *Module) CodeGenerationService() interface{} {
	return m.CodeGenerator
}

// AssemblyService returns the assembly service interface
func (m *Module) AssemblyService() interface{} {
	return m.Assembler
}

// LinkingService returns the linking service interface
func (m *Module) LinkingService() interface{} {
	return m.Linker
}

// Validate validates the module configuration
func (m *Module) Validate() error {
	if m.CodeGenerator == nil {
		return fmt.Errorf("code generator is not initialized")
	}
	if m.Assembler == nil {
		return fmt.Errorf("assembler is not initialized")
	}
	if m.Linker == nil {
		return fmt.Errorf("linker is not initialized")
	}
	if m.ClangAdapter == nil {
		return fmt.Errorf("Clang adapter is not initialized")
	}
	if m.LLCAdapter == nil {
		return fmt.Errorf("LLC adapter is not initialized")
	}
	if m.LDAdapter == nil {
		return fmt.Errorf("linker adapter is not initialized")
	}
	return nil
}
