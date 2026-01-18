package services

import (
	"context"
	"fmt"
	"time"

	"echo/internal/modules/backend/domain/services"
	"echo/internal/modules/frontend/domain/entities"
)

// BackendService 后端应用服务
type BackendService struct {
	codeGenerator services.CodeGenerator
}

// NewBackendService 创建后端服务
func NewBackendService(codeGenerator services.CodeGenerator) services.IBackendService {
	return &BackendService{
		codeGenerator: codeGenerator,
	}
}

// GenerateCode 生成目标代码
func (s *BackendService) GenerateCode(program *entities.Program, target string) (string, error) {
	switch target {
	case "ocaml":
		// 尝试使用扩展接口，如果不支持则使用通用接口
		if extGen, ok := s.codeGenerator.(services.OCamlCodeGenerator); ok {
			return extGen.GenerateOCaml(program), nil
		}
		return s.codeGenerator.GenerateCode(program), nil
	default:
		return s.codeGenerator.GenerateCode(program), nil
	}
}

// GenerateExecutable 生成可执行代码
func (s *BackendService) GenerateExecutable(program *entities.Program, target string) (string, error) {
	switch target {
	case "ocaml":
		// 尝试使用扩展接口，如果不支持则使用通用接口
		if extGen, ok := s.codeGenerator.(services.OCamlCodeGenerator); ok {
			return extGen.GenerateOCaml(program), nil
		}
		return s.codeGenerator.GenerateExecutable(program), nil
	default:
		return s.codeGenerator.GenerateExecutable(program), nil
	}
}

// CodeGenerationService 代码生成服务实现
type CodeGenerationService struct {
	llvmGenerator  services.LLVMCodeGenerator
	ocamlGenerator services.OCamlCodeGenerator
	assembler      services.Assembler
	linker         services.Linker
	// repositories would be injected here
}

// NewCodeGenerationService 创建代码生成服务
func NewCodeGenerationService(
	codeGenerator services.TargetCodeGenerator,
	assembler services.Assembler,
	linker services.Linker,
	executableRepo ExecutableRepository,
	objectFileRepo ObjectFileRepository,
) *CodeGenerationService {
	// 根据类型断言获取具体的生成器
	var llvmGen services.LLVMCodeGenerator
	var ocamlGen services.OCamlCodeGenerator

	if codeGenerator.Target() == "llvm" {
		if gen, ok := codeGenerator.(services.LLVMCodeGenerator); ok {
			llvmGen = gen
		}
	} else if codeGenerator.Target() == "ocaml" {
		if gen, ok := codeGenerator.(services.OCamlCodeGenerator); ok {
			ocamlGen = gen
		}
	}

	return &CodeGenerationService{
		llvmGenerator:  llvmGen,
		ocamlGenerator: ocamlGen,
		assembler:      assembler,
		linker:         linker,
	}
}

// GenerateTargetCode 生成目标代码
func (s *CodeGenerationService) GenerateTargetCode(
	ctx context.Context,
	cmd services.GenerateTargetCodeCommand,
) (*services.CodeGenerationResult, error) {
	var code string
	var err error

	switch cmd.Target {
	case "llvm":
		if s.llvmGenerator == nil {
			return nil, fmt.Errorf("LLVM code generator not available")
		}
		code = s.llvmGenerator.GenerateLLVMIR(cmd.Program)
	case "ocaml":
		if s.ocamlGenerator == nil {
			return nil, fmt.Errorf("OCaml code generator not available")
		}
		code = s.ocamlGenerator.GenerateOCaml(cmd.Program)
	default:
		return nil, fmt.Errorf("unsupported target: %s", cmd.Target)
	}

	if err != nil {
		return &services.CodeGenerationResult{
			Code:    "",
			Target:  cmd.Target,
			Success: false,
			Error:   err,
		}, nil
	}

	return &services.CodeGenerationResult{
		Code:     code,
		Target:   cmd.Target,
		FilePath: "", // TODO: implement file saving
		Success:  true,
		Error:    nil,
	}, nil
}

// GetGenerationStatus 获取生成状态
func (s *CodeGenerationService) GetGenerationStatus(
	ctx context.Context,
	query services.GetGenerationStatusQuery,
) (*services.GenerationStatusDTO, error) {
	// TODO: implement status tracking
	now := time.Now()
	return &services.GenerationStatusDTO{
		RequestID:   query.RequestID,
		Status:      "completed", // mock status
		Progress:    100,
		Error:       "",
		CompletedAt: &now,
	}, nil
}

// AssemblyService 组装服务实现
type AssemblyService struct {
	assembler services.Assembler
	// repositories would be injected here
}

// NewAssemblyService 创建组装服务
func NewAssemblyService(assembler services.Assembler, objectFileRepo ObjectFileRepository) *AssemblyService {
	return &AssemblyService{
		assembler: assembler,
	}
}

// Assemble 组装目标文件（简化实现）
func (s *AssemblyService) Assemble(ctx context.Context, cmd interface{}) (interface{}, error) {
	// TODO: implement assembly logic
	return map[string]interface{}{
		"objectFile": "temp.o",
		"success":    true,
		"error":      nil,
	}, nil
}

// GetAssemblyStatus 获取组装状态（简化实现）
func (s *AssemblyService) GetAssemblyStatus(ctx context.Context, query interface{}) (interface{}, error) {
	// TODO: implement status tracking
	now := time.Now()
	return map[string]interface{}{
		"requestID":   "temp",
		"status":      "completed",
		"progress":    100,
		"error":       "",
		"completedAt": &now,
	}, nil
}

// LinkingService 链接服务实现
type LinkingService struct {
	linker services.Linker
	// repositories would be injected here
}

// NewLinkingService 创建链接服务
func NewLinkingService(linker services.Linker, executableRepo ExecutableRepository) *LinkingService {
	return &LinkingService{
		linker: linker,
	}
}

// Link 链接可执行文件
func (s *LinkingService) Link(ctx context.Context, cmd interface{}) (interface{}, error) {
	// TODO: implement linking logic
	return map[string]interface{}{
		"executable": "program",
		"success":    true,
		"error":      nil,
	}, nil
}

// GetLinkingStatus 获取链接状态
func (s *LinkingService) GetLinkingStatus(ctx context.Context, query interface{}) (interface{}, error) {
	// TODO: implement status tracking
	now := time.Now()
	return map[string]interface{}{
		"requestID":   "temp",
		"status":      "completed",
		"progress":    100,
		"error":       "",
		"completedAt": &now,
	}, nil
}

// Repository interfaces (should be moved to domain layer)
type ExecutableRepository interface {
	Save(executable interface{}) error
	FindByID(id string) (interface{}, error)
}

type ObjectFileRepository interface {
	Save(objectFile interface{}) error
	FindByID(id string) (interface{}, error)
}
