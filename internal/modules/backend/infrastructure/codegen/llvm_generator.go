package codegen

import (
	"fmt"
	"os"

	"echo/internal/modules/backend/domain/services"
	generationInfra "echo/internal/modules/backend/infrastructure/generation"
	"echo/internal/modules/frontend/domain/entities"
)

// LLVMGenerator 是基于DDD架构的LLVM IR代码生成器
type LLVMGenerator struct {
	// 领域服务容器（DDD架构）
	domainContainer *generationInfra.Container
}

// NewLLVMGenerator 创建并返回一个新的LLVMGenerator实例
func NewLLVMGenerator() services.CodeGenerator {
	// 创建领域服务容器（DDD架构）
	domainContainer := generationInfra.NewContainer()

	return &LLVMGenerator{
		// 领域服务容器（DDD架构）
		domainContainer: domainContainer,
	}
}

// GenerateCode 生成LLVM IR代码（实现CodeGenerator接口）
func (g *LLVMGenerator) GenerateCode(prog *entities.Program) string {
	// 获取IR模块管理器
	irManager := g.domainContainer.IRModuleManager()

	// 初始化IR模块（声明外部函数等）
	err := irManager.InitializeMainFunction()
	if err != nil {
		return fmt.Sprintf("; Error initializing IR module: %v\n", err)
	}

	// 使用领域服务容器中的StatementGenerator生成程序
	statementGenerator := g.domainContainer.StatementGenerator()

	// 调试：打印AST信息
	fmt.Fprintf(os.Stderr, "DEBUG: Program has %d statements\n", len(prog.Statements))
	for i, stmt := range prog.Statements {
		fmt.Fprintf(os.Stderr, "DEBUG: Statement %d: %T\n", i, stmt)
		if funcDef, ok := stmt.(*entities.FuncDef); ok {
			fmt.Fprintf(os.Stderr, "DEBUG: Function %s has %d body statements\n", funcDef.Name, len(funcDef.Body))
			for j, bodyStmt := range funcDef.Body {
				fmt.Fprintf(os.Stderr, "DEBUG:   Body statement %d: %T\n", j, bodyStmt)
				if varDecl, ok := bodyStmt.(*entities.VarDecl); ok {
					fmt.Fprintf(os.Stderr, "DEBUG:     VarDecl: %s = %v\n", varDecl.Name, varDecl.Value != nil)
				}
			}
		}
	}

	fmt.Fprintf(os.Stderr, "DEBUG: GenerateProgram called with %d statements\n", len(prog.Statements))
	result, err := statementGenerator.GenerateProgram(irManager, prog.Statements)
	if err != nil {
		// 生成过程中发生错误
		return fmt.Sprintf("; Generation error: %v\n", err)
	}
	fmt.Fprintf(os.Stderr, "DEBUG: GenerateProgram returned success=%v\n", result.Success)

	// 完成IR构建（创建main函数等）
	err = irManager.Finalize()
	if err != nil {
		return fmt.Sprintf("; Error finalizing IR: %v\n", err)
	}

	// 验证IR模块
	err = irManager.Validate()
	if err != nil {
		return fmt.Sprintf("; IR validation error: %v\n", err)
	}

	// 返回生成的IR代码
	if result.Success {
		return irManager.GetIRString()
	}

	// 生成失败，返回错误信息
	return fmt.Sprintf("; Generation failed: %v\n", result.Error)
}

// GenerateExecutable 生成可执行文件（实现CodeGenerator接口）
func (g *LLVMGenerator) GenerateExecutable(prog *entities.Program) string {
	// TODO: 实现基于DDD领域服务的可执行文件生成
	return ""
}
