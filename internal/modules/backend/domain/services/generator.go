package services

import (
	"context"
	"time"

	"echo/internal/modules/frontend/domain/entities"
)

// CodeGenerator 代码生成器接口（领域层定义纯接口）
// 注意：这里不包含任何技术实现细节
type CodeGenerator interface {
	// GenerateCode 生成代码（目标格式由实现决定）
	GenerateCode(program *entities.Program) string

	// GenerateExecutable 生成可执行程序
	GenerateExecutable(program *entities.Program) string
}

// TargetCodeGenerator 目标代码生成器接口（按目标语言区分）
// 这是一个更清晰的接口设计，每个实现对应一种目标语言
type TargetCodeGenerator interface {
	CodeGenerator

	// Target 返回目标语言标识符（如"llvm", "ocaml"）
	Target() string
}

// LLVMCodeGenerator LLVM IR代码生成器接口
type LLVMCodeGenerator interface {
	TargetCodeGenerator

	// GenerateLLVMIR 生成LLVM IR代码
	GenerateLLVMIR(program *entities.Program) string
}

// OCamlCodeGenerator OCaml代码生成器接口
type OCamlCodeGenerator interface {
	TargetCodeGenerator

	// GenerateOCaml 生成OCaml代码
	GenerateOCaml(program *entities.Program) string
}

// Assembler 组装器接口
type Assembler interface {
	// Assemble 将IR代码组装为目标文件
	Assemble(irCode string, target string) (string, error)
}

// Linker 链接器接口
type Linker interface {
	// Link 将目标文件链接为可执行文件
	Link(objectFile string, runtime string) (string, error)
}

// CodeGenerationService 代码生成服务接口（应用层）
type CodeGenerationService interface {
	// GenerateTargetCode 生成目标代码（统一入口）
	GenerateTargetCode(ctx context.Context, cmd GenerateTargetCodeCommand) (*CodeGenerationResult, error)

	// GetGenerationStatus 获取生成状态
	GetGenerationStatus(ctx context.Context, query GetGenerationStatusQuery) (*GenerationStatusDTO, error)
}

// GenerateTargetCodeCommand 生成目标代码命令
type GenerateTargetCodeCommand struct {
	Program   *entities.Program `validate:"required"`
	Target    string            `validate:"required,oneof=llvm ocaml"`
	Runtime   string            `validate:"omitempty,oneof=ucontext asm"`
	OutputDir string            `validate:"omitempty"`
}

// CodeGenerationResult 代码生成结果
type CodeGenerationResult struct {
	Code     string
	Target   string
	FilePath string
	Success  bool
	Error    error
}

// GetGenerationStatusQuery 获取生成状态查询
type GetGenerationStatusQuery struct {
	RequestID string
}

// GenerationStatusDTO 生成状态DTO
type GenerationStatusDTO struct {
	RequestID   string
	Status      string // "pending", "processing", "completed", "failed"
	Progress    int
	Error       string
	CompletedAt *time.Time
}

// Repository interfaces
type ExecutableRepository interface {
	Save(executable interface{}) error
	FindByID(id string) (interface{}, error)
}

type ObjectFileRepository interface {
	Save(objectFile interface{}) error
	FindByID(id string) (interface{}, error)
}

// Adapter interfaces
type ClangAdapter interface {
	CompileIR(irFile string, runtimeLib string, output string) error
}

type LLCAdapter interface {
	AssembleIR(irFile string, output string) error
}

type LinkerAdapter interface {
	LinkObjects(objects []string, runtimeLib string, output string) error
}

// IBackendService 后端服务接口（向后兼容）
type IBackendService interface {
	GenerateCode(program *entities.Program, target string) (string, error)
	GenerateExecutable(program *entities.Program, target string) (string, error)
}
