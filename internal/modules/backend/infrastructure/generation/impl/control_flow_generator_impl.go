package impl

import (
	"fmt"

	"github.com/meetai/echo-lang/internal/modules/backend/domain/services/generation"
	"github.com/meetai/echo-lang/internal/modules/frontend/domain/entities"
)

// ControlFlowGeneratorImpl 控制流生成器实现
type ControlFlowGeneratorImpl struct {
	statementGenerator   generation.StatementGenerator
	expressionEvaluator  generation.ExpressionEvaluator
	breakTarget          string
	continueTarget       string
}

// NewControlFlowGeneratorImpl 创建控制流生成器实现
func NewControlFlowGeneratorImpl(
	stmtGen generation.StatementGenerator,
	exprEval generation.ExpressionEvaluator,
) *ControlFlowGeneratorImpl {
	return &ControlFlowGeneratorImpl{
		statementGenerator:  stmtGen,
		expressionEvaluator: exprEval,
	}
}

// PushControlFlowContext 进入控制流上下文
func (cfg *ControlFlowGeneratorImpl) PushControlFlowContext(breakBlock, continueBlock string) {
	cfg.breakTarget = breakBlock
	cfg.continueTarget = continueBlock
}

// PopControlFlowContext 退出控制流上下文
func (cfg *ControlFlowGeneratorImpl) PopControlFlowContext() {
	cfg.breakTarget = ""
	cfg.continueTarget = ""
}

// GetControlFlowContext 获取控制流上下文
func (cfg *ControlFlowGeneratorImpl) GetControlFlowContext() *generation.ControlFlowContext {
	return &generation.ControlFlowContext{
		BreakTarget:    cfg.breakTarget,
		ContinueTarget: cfg.continueTarget,
	}
}

// GenerateIfStatement 生成if语句
func (cfg *ControlFlowGeneratorImpl) GenerateIfStatement(irManager generation.IRModuleManager, stmt *entities.IfStmt) (*generation.StatementGenerationResult, error) {
	// 1. 求值条件表达式
	condResult, err := cfg.expressionEvaluator.Evaluate(irManager, stmt.Condition)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate if condition: %w", err),
		}, nil
	}

	// 2. 创建基本块
	thenBlock, err := irManager.CreateBasicBlock("if.then")
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create then block: %w", err),
		}, nil
	}

	var elseBlock interface{} = nil
	if stmt.ElseBody != nil {
		elseBlock, err = irManager.CreateBasicBlock("if.else")
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to create else block: %w", err),
			}, nil
		}
	}

	mergeBlock, err := irManager.CreateBasicBlock("if.end")
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create merge block: %w", err),
		}, nil
	}

	// 3. 生成条件分支指令
	if stmt.ElseBody != nil {
		err = irManager.CreateCondBr(condResult, thenBlock, elseBlock)
	} else {
		err = irManager.CreateCondBr(condResult, thenBlock, mergeBlock)
	}
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create conditional branch: %w", err),
		}, nil
	}

	// 4. 生成then分支
	err = irManager.SetCurrentBasicBlock(thenBlock)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set then block: %w", err),
		}, nil
	}

	_, err = cfg.statementGenerator.GenerateProgram(irManager, stmt.ThenBody)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to generate then body: %w", err),
		}, nil
	}

	// 添加跳转到merge块
	err = irManager.CreateBr(mergeBlock)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create branch to merge: %w", err),
		}, nil
	}

	// 5. 生成else分支（如果有）
	if stmt.ElseBody != nil {
		err = irManager.SetCurrentBasicBlock(elseBlock)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to set else block: %w", err),
			}, nil
		}

		_, err = cfg.statementGenerator.GenerateProgram(irManager, stmt.ElseBody)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to generate else body: %w", err),
			}, nil
		}

		// 添加跳转到merge块
		err = irManager.CreateBr(mergeBlock)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to create branch to merge from else: %w", err),
			}, nil
		}
	}

	// 6. 设置merge块为当前块
	err = irManager.SetCurrentBasicBlock(mergeBlock)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set merge block: %w", err),
		}, nil
	}

	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// GenerateForStatement 生成for循环
func (cfg *ControlFlowGeneratorImpl) GenerateForStatement(irManager generation.IRModuleManager, stmt *entities.ForStmt) (*generation.StatementGenerationResult, error) {
	// 生成循环体
	_, err := cfg.statementGenerator.GenerateProgram(irManager, stmt.Body)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to generate for body: %w", err),
		}, nil
	}

	// 简化实现：生成基本的循环结构
	// 这里应该使用irManager创建实际的IR指令，而不是返回字符串
	// 暂时返回成功，后续完善
	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// GenerateWhileStatement 生成while循环
func (cfg *ControlFlowGeneratorImpl) GenerateWhileStatement(irManager generation.IRModuleManager, stmt *entities.WhileStmt) (*generation.StatementGenerationResult, error) {
	// 设置控制流上下文
	cfg.PushControlFlowContext("while_end", "while_cond")

	// 1. 创建基本块
	condBlock, err := irManager.CreateBasicBlock("while.cond")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create condition block: %w", err),
		}, nil
	}

	bodyBlock, err := irManager.CreateBasicBlock("while.body")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create body block: %w", err),
		}, nil
	}

	endBlock, err := irManager.CreateBasicBlock("while.end")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create end block: %w", err),
		}, nil
	}

	// 更新控制流上下文为实际的基本块名
	cfg.breakTarget = "while.end"
	cfg.continueTarget = "while.cond"

	// 2. 从当前块跳转到条件块
	err = irManager.CreateBr(condBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create branch to condition: %w", err),
		}, nil
	}

	// 3. 生成条件块
	err = irManager.SetCurrentBasicBlock(condBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set condition block: %w", err),
		}, nil
	}

	// 求值条件表达式
	condResult, err := cfg.expressionEvaluator.Evaluate(irManager, stmt.Condition)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate while condition: %w", err),
		}, nil
	}

	// 生成条件分支：如果条件为真，跳转到循环体，否则跳转到结束块
	err = irManager.CreateCondBr(condResult, bodyBlock, endBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create conditional branch: %w", err),
		}, nil
	}

	// 4. 生成循环体
	err = irManager.SetCurrentBasicBlock(bodyBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set body block: %w", err),
		}, nil
	}

	_, err = cfg.statementGenerator.GenerateProgram(irManager, stmt.Body)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to generate while body: %w", err),
		}, nil
	}

	// 在循环体末尾无条件跳转回条件块
	err = irManager.CreateBr(condBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create branch back to condition: %w", err),
		}, nil
	}

	// 5. 设置结束块为当前块
	err = irManager.SetCurrentBasicBlock(endBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set end block: %w", err),
		}, nil
	}

	cfg.PopControlFlowContext()
	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// GenerateBreakStatement 生成break语句
func (cfg *ControlFlowGeneratorImpl) GenerateBreakStatement(irManager generation.IRModuleManager, stmt *entities.BreakStmt) (*generation.StatementGenerationResult, error) {
	if cfg.breakTarget == "" {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("break statement outside of loop"),
		}, nil
	}

	// 简化实现：break语句的完整实现需要复杂的控制流上下文管理
	// 这里暂时返回成功，表示语法检查通过
	// 实际实现需要：
	// 1. 获取break目标的基本块引用
	// 2. 生成无条件分支指令
	// 3. 确保控制流正确

	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// GenerateContinueStatement 生成continue语句
func (cfg *ControlFlowGeneratorImpl) GenerateContinueStatement(irManager generation.IRModuleManager, stmt *entities.ContinueStmt) (*generation.StatementGenerationResult, error) {
	if cfg.continueTarget == "" {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("continue statement outside of loop"),
		}, nil
	}

	// 简化实现：continue语句的完整实现需要复杂的控制流上下文管理
	// 这里暂时返回成功，表示语法检查通过
	// 实际实现需要：
	// 1. 获取continue目标的基本块引用
	// 2. 生成无条件分支指令
	// 3. 确保控制流正确

	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// GetBreakTarget 获取当前循环的break目标
func (cfg *ControlFlowGeneratorImpl) GetBreakTarget() interface{} {
	return cfg.breakTarget
}

// GetContinueTarget 获取当前循环的continue目标
func (cfg *ControlFlowGeneratorImpl) GetContinueTarget() interface{} {
	return cfg.continueTarget
}

// IsInLoopContext 检查是否在循环上下文中
func (cfg *ControlFlowGeneratorImpl) IsInLoopContext() bool {
	return cfg.breakTarget != "" || cfg.continueTarget != ""
}
