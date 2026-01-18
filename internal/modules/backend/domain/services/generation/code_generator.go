package generation

import (
	"echo/internal/modules/frontend/domain/entities"
)

// CodeGenerationContext 代码生成上下文
type CodeGenerationContext struct {
	// 当前模块
	Module interface{}

	// 当前函数
	CurrentFunction interface{}

	// 当前基本块
	CurrentBlock interface{}

	// 循环上下文栈
	LoopContexts []LoopContext
}

// LoopContext 循环上下文
type LoopContext struct {
	BreakTarget    interface{} // break跳转目标
	ContinueTarget interface{} // continue跳转目标
}

// CodeGenerator 代码生成器领域服务接口
// 职责：协调所有代码生成子服务，生成完整程序的IR
type CodeGenerator interface {
	// 生成完整程序
	GenerateProgram(program *entities.Program) (string, error)

	// 生成可执行文件
	GenerateExecutable(program *entities.Program) (string, error)

	// 获取语句生成器
	GetStatementGenerator() StatementGenerator

	// 获取表达式求值器
	GetExpressionEvaluator() ExpressionEvaluator

	// 获取控制流生成器
	GetControlFlowGenerator() ControlFlowGenerator

	// 获取符号管理器
	GetSymbolManager() SymbolManager

	// 获取类型映射器
	GetTypeMapper() TypeMapper

	// 获取当前生成上下文
	GetContext() *CodeGenerationContext

	// 设置生成上下文
	SetContext(ctx *CodeGenerationContext)
}
