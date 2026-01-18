package generation

import (
	"echo/internal/modules/frontend/domain/entities"
)

// ControlFlowContext 控制流上下文
type ControlFlowContext struct {
	BreakTarget    string // break语句跳转的目标
	ContinueTarget string // continue语句跳转的目标
}

// ControlFlowGenerator 控制流生成领域服务接口
// 职责：负责条件分支、循环等控制流结构的代码生成
type ControlFlowGenerator interface {
	// 生成if语句
	GenerateIfStatement(irManager IRModuleManager, stmt *entities.IfStmt) (*StatementGenerationResult, error)

	// 生成for循环
	GenerateForStatement(irManager IRModuleManager, stmt *entities.ForStmt) (*StatementGenerationResult, error)

	// 生成while循环
	GenerateWhileStatement(irManager IRModuleManager, stmt *entities.WhileStmt) (*StatementGenerationResult, error)

	// 生成break语句
	GenerateBreakStatement(irManager IRModuleManager, stmt *entities.BreakStmt) (*StatementGenerationResult, error)

	// 生成continue语句
	GenerateContinueStatement(irManager IRModuleManager, stmt *entities.ContinueStmt) (*StatementGenerationResult, error)

	// 检查是否在循环上下文中
	IsInLoopContext() bool

	// 获取当前循环的break目标
	GetBreakTarget() interface{}

	// 获取当前循环的continue目标
	GetContinueTarget() interface{}
}
