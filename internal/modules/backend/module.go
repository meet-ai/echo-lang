// Package backend provides the backend processing context for Echo Language compilation.
// This module handles target code generation, assembly, and linking.
package backend

import (
	"context"
	"fmt"

	"github.com/samber/do"
)

// Module represents the backend processing module
type Module struct {
	// Ports - external interfaces
	codeGenService CodeGenerationService
	asmService     AssemblyService
	linkService    LinkingService

	// Domain services
	codeGenerator TargetCodeGenerator
	assembler     Assembler
	linker        Linker

	// Infrastructure
	executableRepo ExecutableRepository
	objectFileRepo ObjectFileRepository

	// Adapters
	clangAdapter ClangAdapter
	llcAdapter   LLCAdapter
	ldAdapter    LinkerAdapter

	// Dependencies from other modules
	middleendModule interface{} // Will be injected
}

// CodeGenerationService defines the interface for code generation operations
type CodeGenerationService interface {
	GenerateTargetCode(ctx context.Context, cmd GenerateTargetCodeCommand) (*CodeGenerationResult, error)
	AssembleObjectFile(ctx context.Context, cmd AssembleObjectFileCommand) (*AssemblyResult, error)
	LinkExecutable(ctx context.Context, cmd LinkExecutableCommand) (*LinkingResult, error)
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

	// Create application services
	codeGenSvc := NewCodeGenerationService(codeGenerator, assembler, linker, executableRepo, objectFileRepo)
	asmSvc := NewAssemblyService(assembler, objectFileRepo)
	linkSvc := NewLinkingService(linker, executableRepo)

	return &Module{
		codeGenService: codeGenSvc,
		asmService:     asmSvc,
		linkService:    linkSvc,
		codeGenerator:  codeGenerator,
		assembler:      assembler,
		linker:         linker,
		executableRepo: executableRepo,
		objectFileRepo: objectFileRepo,
		clangAdapter:   clangAdapter,
		llcAdapter:     llcAdapter,
		ldAdapter:      ldAdapter,
	}, nil
}

// CodeGenerationService returns the code generation service interface
func (m *Module) CodeGenerationService() CodeGenerationService {
	return m.codeGenService
}

// AssemblyService returns the assembly service interface
func (m *Module) AssemblyService() AssemblyService {
	return m.asmService
}

// LinkingService returns the linking service interface
func (m *Module) LinkingService() LinkingService {
	return m.linkService
}

// Validate validates the module configuration
func (m *Module) Validate() error {
	if m.codeGenService == nil {
		return fmt.Errorf("code generation service is not initialized")
	}
	if m.asmService == nil {
		return fmt.Errorf("assembly service is not initialized")
	}
	if m.linkService == nil {
		return fmt.Errorf("linking service is not initialized")
	}
	if m.clangAdapter == nil {
		return fmt.Errorf("Clang adapter is not initialized")
	}
	if m.llcAdapter == nil {
		return fmt.Errorf("LLC adapter is not initialized")
	}
	if m.ldAdapter == nil {
		return fmt.Errorf("linker adapter is not initialized")
	}
	return nil
}
