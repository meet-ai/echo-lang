package impl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	llvalue "github.com/llir/llvm/ir/value"

	"echo/internal/modules/backend/domain/services/generation"
	"echo/internal/modules/frontend/domain/entities"
)

// ControlFlowGeneratorImpl 控制流生成器实现
type ControlFlowGeneratorImpl struct {
	statementGenerator   generation.StatementGenerator
	expressionEvaluator  generation.ExpressionEvaluator
	symbolManager        generation.SymbolManager  // 添加符号管理器，用于注册循环变量
	breakTarget          string
	continueTarget       string
	breakExitCounter     int // 用于生成唯一的 break 后继块名，避免 terminator 后继续生成指令
	continueExitCounter  int // 用于生成唯一的 continue 后继块名
	ifBlockCounter       int // 用于生成唯一的 if.then/if.else/if.end 块名，避免嵌套 else-if 时块名重复导致控制流错误
}

// NewControlFlowGeneratorImpl 创建控制流生成器实现
func NewControlFlowGeneratorImpl(
	stmtGen generation.StatementGenerator,
	exprEval generation.ExpressionEvaluator,
	symbolMgr generation.SymbolManager,
) *ControlFlowGeneratorImpl {
	return &ControlFlowGeneratorImpl{
		statementGenerator:  stmtGen,
		expressionEvaluator: exprEval,
		symbolManager:       symbolMgr,
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
// 使用 ifBlockCounter 生成唯一块名（if.then.N / if.else.N / if.end.N），避免嵌套 else-if 时
// 块名重复导致 LLVM 中多块同名、控制流跳转错误或执行挂起。
func (cfg *ControlFlowGeneratorImpl) GenerateIfStatement(irManager generation.IRModuleManager, stmt *entities.IfStmt) (*generation.StatementGenerationResult, error) {
	// 1. 求值条件表达式
	condResult, err := cfg.expressionEvaluator.Evaluate(irManager, stmt.Condition)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate if condition: %w", err),
		}, nil
	}

	// 2. 创建基本块（唯一命名，避免嵌套 if/else-if 时块名冲突）
	n := cfg.ifBlockCounter
	cfg.ifBlockCounter++
	thenBlock, err := irManager.CreateBasicBlock("if.then." + strconv.Itoa(n))
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create then block: %w", err),
		}, nil
	}

	var elseBlock interface{} = nil
	if stmt.ElseBody != nil {
		elseBlock, err = irManager.CreateBasicBlock("if.else." + strconv.Itoa(n))
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to create else block: %w", err),
			}, nil
		}
	}

	mergeBlock, err := irManager.CreateBasicBlock("if.end." + strconv.Itoa(n))
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
	fmt.Printf("DEBUG: GenerateForStatement - IsRangeLoop=%v, IsIterLoop=%v, IsMapIter=%v, IterKeyVar=%s, IterValueVar=%s\n", 
		stmt.IsRangeLoop, stmt.IsIterLoop, stmt.IsMapIter, stmt.IterKeyVar, stmt.IterValueVar)
	
	// 检查是否是范围循环
	if stmt.IsRangeLoop {
		return cfg.GenerateForRangeLoop(irManager, stmt)
	}
	
	// 检查是否是迭代循环
	if stmt.IsIterLoop {
		return cfg.GenerateForIterLoop(irManager, stmt)
	}

	// C风格for循环或简单条件循环（简化实现）
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

// GenerateForRangeLoop 生成范围循环：for i in start..end { body }
func (cfg *ControlFlowGeneratorImpl) GenerateForRangeLoop(irManager generation.IRModuleManager, stmt *entities.ForStmt) (*generation.StatementGenerationResult, error) {
	// 设置控制流上下文
	cfg.PushControlFlowContext("for.end", "for.cond")

	// 1. 创建基本块
	initBlock, err := irManager.CreateBasicBlock("for.init")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create init block: %w", err),
		}, nil
	}

	condBlock, err := irManager.CreateBasicBlock("for.cond")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create condition block: %w", err),
		}, nil
	}

	bodyBlock, err := irManager.CreateBasicBlock("for.body")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create body block: %w", err),
		}, nil
	}

	incrementBlock, err := irManager.CreateBasicBlock("for.increment")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create increment block: %w", err),
		}, nil
	}

	endBlock, err := irManager.CreateBasicBlock("for.end")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create end block: %w", err),
		}, nil
	}

	// 更新控制流上下文
	cfg.breakTarget = "for.end"
	cfg.continueTarget = "for.cond"

	// 2. 从当前块跳转到init块
	err = irManager.CreateBr(initBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create branch to init: %w", err),
		}, nil
	}

	// 3. 生成init块：创建循环变量并初始化为start
	err = irManager.SetCurrentBasicBlock(initBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set init block: %w", err),
		}, nil
	}

	// 求值范围起始值
	startResult, err := cfg.expressionEvaluator.Evaluate(irManager, stmt.RangeStart)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate range start: %w", err),
		}, nil
	}

	// 创建循环变量（alloca）
	loopVarPtr, err := irManager.CreateAlloca(types.I32, stmt.LoopVar)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create loop variable: %w", err),
		}, nil
	}

	// 将起始值存储到循环变量
	if startValue, ok := startResult.(llvalue.Value); ok {
		err = irManager.CreateStore(startValue, loopVarPtr)
		if err != nil {
			cfg.PopControlFlowContext()
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to store initial value: %w", err),
			}, nil
		}
	} else {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("invalid start value type: %T", startResult),
		}, nil
	}

	// 注册循环变量到符号表（以便在循环体中使用）
	// 符号表存储的是alloca指针，在访问时会自动加载值
	err = cfg.symbolManager.RegisterSymbol(stmt.LoopVar, "int", loopVarPtr)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to register loop variable: %w", err),
		}, nil
	}

	// 跳转到条件块
	err = irManager.CreateBr(condBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create branch to condition: %w", err),
		}, nil
	}

	// 4. 生成条件块：检查 i < end
	err = irManager.SetCurrentBasicBlock(condBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set condition block: %w", err),
		}, nil
	}

	// 加载循环变量值
	loopVarValue, err := irManager.CreateLoad(types.I32, loopVarPtr, stmt.LoopVar)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to load loop variable: %w", err),
		}, nil
	}

	// 求值范围结束值
	endResult, err := cfg.expressionEvaluator.Evaluate(irManager, stmt.RangeEnd)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate range end: %w", err),
		}, nil
	}

	// 比较 i < end
	var condValue llvalue.Value
	if endValue, ok := endResult.(llvalue.Value); ok {
		// 使用 CreateBinaryOp 创建比较操作
		condResult, err := irManager.CreateBinaryOp("<", loopVarValue, endValue, "loop.cond")
		if err != nil {
			cfg.PopControlFlowContext()
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to create comparison: %w", err),
			}, nil
		}
		condValue = condResult.(llvalue.Value)
	} else {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("invalid end value type: %T", endResult),
		}, nil
	}

	// 条件分支：如果 i < end，跳转到循环体；否则跳转到结束块
	err = irManager.CreateCondBr(condValue, bodyBlock, endBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create conditional branch: %w", err),
		}, nil
	}

	// 5. 生成循环体
	err = irManager.SetCurrentBasicBlock(bodyBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set body block: %w", err),
		}, nil
	}

	// 循环变量已在init块中注册到符号表，循环体中可以直接访问

	_, err = cfg.statementGenerator.GenerateProgram(irManager, stmt.Body)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to generate for body: %w", err),
		}, nil
	}

	// 在循环体末尾跳转到increment块
	err = irManager.CreateBr(incrementBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create branch to increment: %w", err),
		}, nil
	}

	// 6. 生成increment块：i = i + 1
	err = irManager.SetCurrentBasicBlock(incrementBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set increment block: %w", err),
		}, nil
	}

	// 加载当前循环变量值
	currentValue, err := irManager.CreateLoad(types.I32, loopVarPtr, stmt.LoopVar+".current")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to load current loop variable: %w", err),
		}, nil
	}

	// 创建常量 1
	one := constant.NewInt(types.I32, 1)

	// i = i + 1
	incrementedValue, err := irManager.CreateBinaryOp("+", currentValue, one, stmt.LoopVar+".inc")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to increment loop variable: %w", err),
		}, nil
	}

	// 存储递增后的值
	err = irManager.CreateStore(incrementedValue, loopVarPtr)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to store incremented value: %w", err),
		}, nil
	}

	// 跳转回条件块
	err = irManager.CreateBr(condBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create branch back to condition: %w", err),
		}, nil
	}

	// 7. 设置结束块为当前块
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

// GenerateForIterLoop 生成迭代循环：for item in collection { body }
// 支持三种类型：
// 1. for item in arr { ... } - 数组迭代
// 2. for ch in s { ... } - 字符串迭代
// 3. for (key, value) in obj { ... } - map迭代
func (cfg *ControlFlowGeneratorImpl) GenerateForIterLoop(irManager generation.IRModuleManager, stmt *entities.ForStmt) (*generation.StatementGenerationResult, error) {
	fmt.Printf("DEBUG: GenerateForIterLoop - IsMapIter=%v, IterKeyVar=%s, IterValueVar=%s\n", stmt.IsMapIter, stmt.IterKeyVar, stmt.IterValueVar)
	
	// 检查是否是map迭代
	if stmt.IsMapIter {
		fmt.Printf("DEBUG: GenerateForIterLoop - calling GenerateForMapIterLoop\n")
		return cfg.GenerateForMapIterLoop(irManager, stmt)
	}
	
	// 数组或字符串迭代
	// 策略：将迭代循环转换为范围循环
	// for item in arr { ... } → for i in 0..len(arr) { let item = arr[i]; ... }
	
	// 设置控制流上下文
	cfg.PushControlFlowContext("for.end", "for.cond")
	
	// 1. 创建基本块
	initBlock, err := irManager.CreateBasicBlock("for.init")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create init block: %w", err),
		}, nil
	}
	
	condBlock, err := irManager.CreateBasicBlock("for.cond")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create condition block: %w", err),
		}, nil
	}
	
	bodyBlock, err := irManager.CreateBasicBlock("for.body")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create body block: %w", err),
		}, nil
	}
	
	incrementBlock, err := irManager.CreateBasicBlock("for.increment")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create increment block: %w", err),
		}, nil
	}
	
	endBlock, err := irManager.CreateBasicBlock("for.end")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create end block: %w", err),
		}, nil
	}
	
	// 更新控制流上下文
	cfg.breakTarget = "for.end"
	cfg.continueTarget = "for.cond"
	
	// 2. 从当前块跳转到init块
	err = irManager.CreateBr(initBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create branch to init: %w", err),
		}, nil
	}
	
	// 3. 生成init块：求值集合表达式，获取长度，创建循环变量
	err = irManager.SetCurrentBasicBlock(initBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set init block: %w", err),
		}, nil
	}
	
	// 求值集合表达式（数组或字符串）
	collectionResult, err := cfg.expressionEvaluator.Evaluate(irManager, stmt.IterCollection)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate collection: %w", err),
		}, nil
	}
	
	// 创建循环变量（索引变量）
	loopVarPtr, err := irManager.CreateAlloca(types.I32, "iter_index")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create loop variable: %w", err),
		}, nil
	}
	
	// 初始化循环变量为0
	zero := constant.NewInt(types.I32, 0)
	err = irManager.CreateStore(zero, loopVarPtr)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to store initial value: %w", err),
		}, nil
	}
	
	// 获取集合长度
	// 策略：创建 len(collection) 表达式并求值
	// 注意：需要根据集合类型调用不同的len函数
	// 对于数组：len(arr) 返回数组长度
	// 对于字符串：len(s) 返回字符串长度（UTF-8字符数）
	
	// 创建 LenExpr 表达式（不是 FuncCall）
	lenExpr := &entities.LenExpr{
		Array: stmt.IterCollection,
	}
	
	// 求值 len() 表达式
	lenResult, err := cfg.expressionEvaluator.Evaluate(irManager, lenExpr)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate len() for collection: %w", err),
		}, nil
	}
	
	// 将长度值转换为常量（如果可能）
	var endValue llvalue.Value
	if lenConst, ok := lenResult.(*constant.Int); ok {
		endValue = lenConst
	} else if lenVal, ok := lenResult.(llvalue.Value); ok {
		endValue = lenVal
	} else {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("invalid len() result type: %T", lenResult),
		}, nil
	}
	
	// 跳转到条件块
	err = irManager.CreateBr(condBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create branch to condition: %w", err),
		}, nil
	}
	
	// 4. 生成条件块：检查 i < len(collection)
	err = irManager.SetCurrentBasicBlock(condBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set condition block: %w", err),
		}, nil
	}
	
	// 加载循环变量值
	loopVarValue, err := irManager.CreateLoad(types.I32, loopVarPtr, "iter_index.current")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to load loop variable: %w", err),
		}, nil
	}
	
	// 比较 i < len(collection)
	condResult, err := irManager.CreateBinaryOp("<", loopVarValue, endValue, "iter.cond")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create comparison: %w", err),
		}, nil
	}
	condValue := condResult.(llvalue.Value)
	
	// 条件分支：如果 i < len(collection)，跳转到循环体；否则跳转到结束块
	err = irManager.CreateCondBr(condValue, bodyBlock, endBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create conditional branch: %w", err),
		}, nil
	}
	
	// 5. 生成循环体
	err = irManager.SetCurrentBasicBlock(bodyBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set body block: %w", err),
		}, nil
	}
	
	// 在循环体中，需要：
	// 1. 从集合中获取当前元素：collection[i]
	// 2. 将元素值存储到迭代变量中
	// 3. 注册迭代变量到符号表（以便在循环体中使用）
	
	// 加载当前索引值
	currentIndex, err := irManager.CreateLoad(types.I32, loopVarPtr, stmt.IterVar+".index")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to load current index: %w", err),
		}, nil
	}
	
	// 求值索引访问（获取当前元素）
	// 注意：这里需要手动实现索引访问，因为我们需要使用currentIndex而不是常量
	// 简化处理：假设集合是数组，使用GEP指令获取元素
	// 获取集合值（指针）
	if collectionValue, ok := collectionResult.(llvalue.Value); ok {
		// 确定元素类型（简化：假设是i32或i8*）
		elementType := types.I32 // 默认类型，实际应该从集合类型推断
		
		// 如果是字符串，元素类型是i8
		if ident, ok := stmt.IterCollection.(*entities.Identifier); ok {
			symbol, err := cfg.symbolManager.LookupSymbol(ident.Name)
			if err == nil {
				if symbol.Type == "string" {
					elementType = types.I8
				} else if strings.HasPrefix(symbol.Type, "[") {
					// 数组类型，需要推断元素类型
					// 简化：假设是i32
					elementType = types.I32
				}
			}
		}
		
		// 使用GEP指令获取元素指针
		elementPtr, err := irManager.CreateGetElementPtr(elementType, collectionValue, currentIndex)
		if err != nil {
			cfg.PopControlFlowContext()
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to create GEP for element access: %w", err),
			}, nil
		}
		
		// 创建迭代变量的alloca
		iterVarPtr, err := irManager.CreateAlloca(elementType, stmt.IterVar)
		if err != nil {
			cfg.PopControlFlowContext()
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to create iteration variable: %w", err),
			}, nil
		}
		
		// 加载元素值
		elementValue, err := irManager.CreateLoad(elementType, elementPtr, stmt.IterVar+".value")
		if err != nil {
			cfg.PopControlFlowContext()
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to load element value: %w", err),
			}, nil
		}
		
		// 存储元素值到迭代变量
		err = irManager.CreateStore(elementValue, iterVarPtr)
		if err != nil {
			cfg.PopControlFlowContext()
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to store element value: %w", err),
			}, nil
		}
		
		// 注册迭代变量到符号表
		// 确定Echo类型字符串
		echoType := "int" // 默认类型
		if elementType == types.I8 {
			echoType = "char"
		}
		err = cfg.symbolManager.RegisterSymbol(stmt.IterVar, echoType, iterVarPtr)
		if err != nil {
			cfg.PopControlFlowContext()
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to register iteration variable: %w", err),
			}, nil
		}
	} else {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("invalid collection value type: %T", collectionResult),
		}, nil
	}
	
	// 生成循环体语句
	_, err = cfg.statementGenerator.GenerateProgram(irManager, stmt.Body)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to generate iteration body: %w", err),
		}, nil
	}
	
	// 在循环体末尾跳转到increment块
	err = irManager.CreateBr(incrementBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create branch to increment: %w", err),
		}, nil
	}
	
	// 6. 生成increment块：i = i + 1
	err = irManager.SetCurrentBasicBlock(incrementBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set increment block: %w", err),
		}, nil
	}
	
	// 加载当前循环变量值
	currentValue, err := irManager.CreateLoad(types.I32, loopVarPtr, "iter_index.current")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to load current loop variable: %w", err),
		}, nil
	}
	
	// 创建常量 1
	one := constant.NewInt(types.I32, 1)
	
	// i = i + 1
	incrementedValue, err := irManager.CreateBinaryOp("+", currentValue, one, "iter_index.inc")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to increment loop variable: %w", err),
		}, nil
	}
	
	// 存储递增后的值
	err = irManager.CreateStore(incrementedValue, loopVarPtr)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to store incremented value: %w", err),
		}, nil
	}
	
	// 跳转回条件块
	err = irManager.CreateBr(condBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create branch back to condition: %w", err),
		}, nil
	}
	
	// 7. 设置结束块为当前块
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

// GenerateForMapIterLoop 生成map迭代循环：for (key, value) in obj { body }
func (cfg *ControlFlowGeneratorImpl) GenerateForMapIterLoop(irManager generation.IRModuleManager, stmt *entities.ForStmt) (*generation.StatementGenerationResult, error) {
	fmt.Printf("DEBUG: GenerateForMapIterLoop - called with IsMapIter=%v, IterKeyVar=%s, IterValueVar=%s\n", stmt.IsMapIter, stmt.IterKeyVar, stmt.IterValueVar)
	
	// 实现map迭代循环的策略：
	// 1. 调用运行时函数 runtime_map_get_keys(map_ptr) 获取键值对数组
	// 2. 返回结构体 MapIterResult { keys: char**, values: char**, count: i32 }
	// 3. 使用范围循环遍历键数组（for i in 0..count）
	// 4. 在循环体中，从键数组和值数组中获取对应的键值对
	// 5. 注册key和value变量到符号表
	
	// 1. 求值map表达式
	fmt.Printf("DEBUG: GenerateForMapIterLoop - evaluating map expression: %T\n", stmt.IterCollection)
	mapResult, err := cfg.expressionEvaluator.Evaluate(irManager, stmt.IterCollection)
	if err != nil {
		fmt.Printf("DEBUG: GenerateForMapIterLoop - failed to evaluate map expression: %v\n", err)
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate map expression: %w", err),
		}, nil
	}
	fmt.Printf("DEBUG: GenerateForMapIterLoop - map expression evaluated successfully: %T\n", mapResult)
	
	// 确保map值是指针类型
	mapValue, ok := mapResult.(llvalue.Value)
	if !ok {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("map value is not a valid LLVM value: %T", mapResult),
		}, nil
	}
	
	// 2. 调用运行时函数获取键值对数组
	// runtime_map_get_keys(map_ptr: *i8) -> *MapIterResult
	// MapIterResult 结构体：{ keys: *i8* (char**), values: *i8* (char**), count: i32 }
	fmt.Printf("DEBUG: GenerateForMapIterLoop - getting runtime_map_get_keys function\n")
	mapGetKeysFunc, exists := irManager.GetExternalFunction("runtime_map_get_keys")
	if !exists {
		fmt.Printf("DEBUG: GenerateForMapIterLoop - ERROR: runtime_map_get_keys function not declared\n")
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("runtime_map_get_keys function not declared"),
		}, nil
	}
	fmt.Printf("DEBUG: GenerateForMapIterLoop - runtime_map_get_keys function found: %T\n", mapGetKeysFunc)
	
	// 调用运行时函数
	fmt.Printf("DEBUG: GenerateForMapIterLoop - calling runtime_map_get_keys with mapValue: %T\n", mapValue)
	iterResultPtr, err := irManager.CreateCall(mapGetKeysFunc, mapValue)
	if err != nil {
		fmt.Printf("DEBUG: GenerateForMapIterLoop - ERROR: failed to call runtime_map_get_keys: %v\n", err)
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to call runtime_map_get_keys: %w", err),
		}, nil
	}
	fmt.Printf("DEBUG: GenerateForMapIterLoop - runtime_map_get_keys called successfully, result: %T\n", iterResultPtr)
	
	iterResultValue, ok := iterResultPtr.(llvalue.Value)
	if !ok {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("runtime_map_get_keys returned invalid type: %T", iterResultPtr),
		}, nil
	}
	
	// 2.5. 获取当前函数和基本块（用于创建循环块和NULL检查）
	fmt.Printf("DEBUG: GenerateForMapIterLoop - getting current function for loop blocks\n")
	currentFuncInterface := irManager.GetCurrentFunction()
	if currentFuncInterface == nil {
		fmt.Printf("DEBUG: GenerateForMapIterLoop - ERROR: no current function set\n")
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("no current function set"),
		}, nil
	}
	currentFunc, ok := currentFuncInterface.(*ir.Func)
	if !ok {
		fmt.Printf("DEBUG: GenerateForMapIterLoop - ERROR: current function is not *ir.Func: %T\n", currentFuncInterface)
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("current function is not *ir.Func: %T", currentFuncInterface),
		}, nil
	}
	fmt.Printf("DEBUG: GenerateForMapIterLoop - current function: %s\n", currentFunc.Name())
	
	// 获取当前基本块（用于添加NULL检查）
	currentBlockForNullCheck := irManager.GetCurrentBasicBlock()
	entryBlockForNullCheck, ok := currentBlockForNullCheck.(*ir.Block)
	if !ok {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("current block is not *ir.Block for null check: %T", currentBlockForNullCheck),
		}, nil
	}
	
	// 先创建所有循环块（包括endBlock，用于NULL检查的分支目标）
	initBlock := currentFunc.NewBlock("map_iter.init")
	condBlock := currentFunc.NewBlock("map_iter.cond")
	bodyBlock := currentFunc.NewBlock("map_iter.body")
	incrementBlock := currentFunc.NewBlock("map_iter.increment")
	endBlock := currentFunc.NewBlock("map_iter.end")
	fmt.Printf("DEBUG: GenerateForMapIterLoop - loop blocks created\n")
	
	// 在entry块中添加NULL检查：如果iterResultPtr为NULL，跳转到endBlock
	fmt.Printf("DEBUG: GenerateForMapIterLoop - adding NULL check in entry block\n")
	nullPtr := constant.NewNull(types.NewPointer(types.I8))
	nullCheckResult := entryBlockForNullCheck.NewICmp(enum.IPredEQ, iterResultValue, nullPtr)
	nullCheckResult.SetName("map_iter_null_check")
	
	// 条件分支：如果为NULL，跳转到endBlock；否则跳转到initBlock
	err = irManager.CreateCondBr(nullCheckResult, endBlock, initBlock)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create null check branch: %w", err),
		}, nil
	}
	fmt.Printf("DEBUG: GenerateForMapIterLoop - NULL check branch created\n")
	
	// 3. 设置控制流上下文（使用基本块名称字符串）
	fmt.Printf("DEBUG: GenerateForMapIterLoop - pushing control flow context\n")
	cfg.PushControlFlowContext("map_iter.end", "map_iter.cond")
	fmt.Printf("DEBUG: GenerateForMapIterLoop - control flow context pushed\n")
	
	// 4. 生成init块：提取MapIterResult字段并初始化循环变量 i = 0
	err = irManager.SetCurrentBasicBlock(initBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set init block: %w", err),
		}, nil
	}
	
	// 4.1. 从 MapIterResult 结构体中提取字段（在initBlock中执行，因为只有非NULL时才执行）
	// MapIterResult 结构体布局：
	// - keys: *i8* (char**) - 偏移 0
	// - values: *i8* (char**) - 偏移 8 (64位系统)
	// - count: i32 - 偏移 16
	
	// 定义 MapIterResult 结构体类型
	// { i8**, i8**, i32 } - keys, values, count
	fmt.Printf("DEBUG: GenerateForMapIterLoop - creating MapIterResult struct type\n")
	mapIterResultType := types.NewStruct(
		types.NewPointer(types.NewPointer(types.I8)), // keys: char**
		types.NewPointer(types.NewPointer(types.I8)), // values: char**
		types.I32, // count: i32
	)
	
	// 提取 count 字段（偏移 16，索引 2）
	fmt.Printf("DEBUG: GenerateForMapIterLoop - extracting count field from MapIterResult\n")
	countIndex := constant.NewInt(types.I32, 2)
	countPtr, err := irManager.CreateGetElementPtr(mapIterResultType, iterResultValue, constant.NewInt(types.I32, 0), countIndex)
	if err != nil {
		fmt.Printf("DEBUG: GenerateForMapIterLoop - ERROR: failed to create GEP for count field: %v\n", err)
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create GEP for count field: %w", err),
		}, nil
	}
	fmt.Printf("DEBUG: GenerateForMapIterLoop - GEP for count field created successfully: %T\n", countPtr)
	
	countPtrValue, ok := countPtr.(llvalue.Value)
	if !ok {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("CreateGetElementPtr returned invalid type: %T", countPtr),
		}, nil
	}
	
	// 加载 count 值
	fmt.Printf("DEBUG: GenerateForMapIterLoop - loading count value\n")
	countValue, err := irManager.CreateLoad(types.I32, countPtrValue, "map_count")
	if err != nil {
		fmt.Printf("DEBUG: GenerateForMapIterLoop - ERROR: failed to load count: %v\n", err)
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to load count: %w", err),
		}, nil
	}
	fmt.Printf("DEBUG: GenerateForMapIterLoop - count value loaded successfully: %T\n", countValue)
	
	countLLVMValue, ok := countValue.(llvalue.Value)
	if !ok {
		fmt.Printf("DEBUG: GenerateForMapIterLoop - ERROR: CreateLoad returned invalid type: %T\n", countValue)
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("CreateLoad returned invalid type: %T", countValue),
		}, nil
	}
	fmt.Printf("DEBUG: GenerateForMapIterLoop - count value converted to llvalue.Value\n")
	
	// 提取 keys 和 values 字段
	keysIndex := constant.NewInt(types.I32, 0)
	keysPtr, err := irManager.CreateGetElementPtr(mapIterResultType, iterResultValue, constant.NewInt(types.I32, 0), keysIndex)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create GEP for keys field: %w", err),
		}, nil
	}
	
	keysPtrValue, ok := keysPtr.(llvalue.Value)
	if !ok {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("CreateGetElementPtr returned invalid type: %T", keysPtr),
		}, nil
	}
	
	// 加载 keys 指针（char**）
	keysArrayPtr, err := irManager.CreateLoad(types.NewPointer(types.NewPointer(types.I8)), keysPtrValue, "map_keys_ptr")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to load keys pointer: %w", err),
		}, nil
	}
	
	keysArrayPtrValue, ok := keysArrayPtr.(llvalue.Value)
	if !ok {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("CreateLoad returned invalid type: %T", keysArrayPtr),
		}, nil
	}
	
	// 提取 values 字段
	valuesIndex := constant.NewInt(types.I32, 1)
	valuesPtr, err := irManager.CreateGetElementPtr(mapIterResultType, iterResultValue, constant.NewInt(types.I32, 0), valuesIndex)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create GEP for values field: %w", err),
		}, nil
	}
	
	valuesPtrValue, ok := valuesPtr.(llvalue.Value)
	if !ok {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("CreateGetElementPtr returned invalid type: %T", valuesPtr),
		}, nil
	}
	
	// 加载 values 指针（char**）
	valuesArrayPtr, err := irManager.CreateLoad(types.NewPointer(types.NewPointer(types.I8)), valuesPtrValue, "map_values_ptr")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to load values pointer: %w", err),
		}, nil
	}
	
	valuesArrayPtrValue, ok := valuesArrayPtr.(llvalue.Value)
	if !ok {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("CreateLoad returned invalid type: %T", valuesArrayPtr),
		}, nil
	}
	
	// 4.2. 创建循环变量 i（用于索引）
	loopVarPtr, err := irManager.CreateAlloca(types.I32, "map_iter_index")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to allocate loop variable: %w", err),
		}, nil
	}
	
	loopVarPtrValue, ok := loopVarPtr.(llvalue.Value)
	if !ok {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("CreateAlloca returned invalid type: %T", loopVarPtr),
		}, nil
	}
	
	// 初始化 i = 0
	zero := constant.NewInt(types.I32, 0)
	err = irManager.CreateStore(zero, loopVarPtrValue)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to initialize loop variable: %w", err),
		}, nil
	}
	
	// 跳转到条件块
	err = irManager.CreateBr(condBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create branch to condition: %w", err),
		}, nil
	}
	
	// 7. 生成cond块：i < count
	err = irManager.SetCurrentBasicBlock(condBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set cond block: %w", err),
		}, nil
	}
	
	// 加载当前索引值
	currentIndex, err := irManager.CreateLoad(types.I32, loopVarPtrValue, "map_iter_index.current")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to load current index: %w", err),
		}, nil
	}
	
	currentIndexValue, ok := currentIndex.(llvalue.Value)
	if !ok {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("CreateLoad returned invalid type: %T", currentIndex),
		}, nil
	}
	
	// 比较 i < count（在condBlock中创建，而不是在entry块中）
	condResult := condBlock.NewICmp(enum.IPredSLT, currentIndexValue, countLLVMValue)
	condResult.SetName("map_iter_cond")
	
	// 条件分支
	err = irManager.CreateCondBr(condResult, bodyBlock, endBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create conditional branch: %w", err),
		}, nil
	}
	
	// 8. 生成body块：从键数组和值数组中获取键值对
	err = irManager.SetCurrentBasicBlock(bodyBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set body block: %w", err),
		}, nil
	}
	
	// 加载当前索引（在body块中也需要）
	currentIndexInBody, err := irManager.CreateLoad(types.I32, loopVarPtrValue, "map_iter_index.body")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to load current index in body: %w", err),
		}, nil
	}
	
	currentIndexInBodyValue, ok := currentIndexInBody.(llvalue.Value)
	if !ok {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("CreateLoad returned invalid type: %T", currentIndexInBody),
		}, nil
	}
	
	// 获取 keys[i]（char*）
	// keysArrayPtr 是 char**，需要索引访问
	keyPtr, err := irManager.CreateGetElementPtr(types.NewPointer(types.I8), keysArrayPtrValue, currentIndexInBodyValue)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create GEP for key: %w", err),
		}, nil
	}
	
	keyPtrValue, ok := keyPtr.(llvalue.Value)
	if !ok {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("CreateGetElementPtr returned invalid type: %T", keyPtr),
		}, nil
	}
	
	// 加载 key 值（char*）
	keyValue, err := irManager.CreateLoad(types.NewPointer(types.I8), keyPtrValue, "map_key")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to load key value: %w", err),
		}, nil
	}
	
	keyValueLLVM, ok := keyValue.(llvalue.Value)
	if !ok {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("CreateLoad returned invalid type: %T", keyValue),
		}, nil
	}
	
	// 获取 values[i]（char*）
	valuePtr, err := irManager.CreateGetElementPtr(types.NewPointer(types.I8), valuesArrayPtrValue, currentIndexInBodyValue)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create GEP for value: %w", err),
		}, nil
	}
	
	valuePtrValue, ok := valuePtr.(llvalue.Value)
	if !ok {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("CreateGetElementPtr returned invalid type: %T", valuePtr),
		}, nil
	}
	
	// 加载 value 值（char*）
	valueValue, err := irManager.CreateLoad(types.NewPointer(types.I8), valuePtrValue, "map_value")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to load value value: %w", err),
		}, nil
	}
	
	valueValueLLVM, ok := valueValue.(llvalue.Value)
	if !ok {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("CreateLoad returned invalid type: %T", valueValue),
		}, nil
	}
	
	// 9. 为key和value变量分配内存并存储值
	// 分配key变量内存
	keyVarPtr, err := irManager.CreateAlloca(types.NewPointer(types.I8), stmt.IterKeyVar)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to allocate key variable: %w", err),
		}, nil
	}
	
	keyVarPtrValue, ok := keyVarPtr.(llvalue.Value)
	if !ok {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("CreateAlloca returned invalid type: %T", keyVarPtr),
		}, nil
	}
	
	// 存储key值
	err = irManager.CreateStore(keyValueLLVM, keyVarPtrValue)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to store key value: %w", err),
		}, nil
	}
	
	// 注册key变量到符号表
	err = cfg.symbolManager.RegisterSymbol(stmt.IterKeyVar, "string", keyVarPtrValue)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to register key variable: %w", err),
		}, nil
	}
	
	// 分配value变量内存
	valueVarPtr, err := irManager.CreateAlloca(types.NewPointer(types.I8), stmt.IterValueVar)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to allocate value variable: %w", err),
		}, nil
	}
	
	valueVarPtrValue, ok := valueVarPtr.(llvalue.Value)
	if !ok {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("CreateAlloca returned invalid type: %T", valueVarPtr),
		}, nil
	}
	
	// 存储value值
	err = irManager.CreateStore(valueValueLLVM, valueVarPtrValue)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to store value value: %w", err),
		}, nil
	}
	
	// 注册value变量到符号表
	err = cfg.symbolManager.RegisterSymbol(stmt.IterValueVar, "string", valueVarPtrValue)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to register value variable: %w", err),
		}, nil
	}
	
	// 10. 生成循环体语句
	_, err = cfg.statementGenerator.GenerateProgram(irManager, stmt.Body)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to generate map iteration body: %w", err),
		}, nil
	}
	
	// 在循环体末尾跳转到increment块
	err = irManager.CreateBr(incrementBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create branch to increment: %w", err),
		}, nil
	}
	
	// 11. 生成increment块：i = i + 1
	err = irManager.SetCurrentBasicBlock(incrementBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set increment block: %w", err),
		}, nil
	}
	
	// 加载当前值
	currentValue, err := irManager.CreateLoad(types.I32, loopVarPtrValue, "map_iter_index.current")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to load current loop variable: %w", err),
		}, nil
	}
	
	currentValueLLVM, ok := currentValue.(llvalue.Value)
	if !ok {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("CreateLoad returned invalid type: %T", currentValue),
		}, nil
	}
	
	// 创建常量 1
	one := constant.NewInt(types.I32, 1)
	
	// i = i + 1
	incrementedValue, err := irManager.CreateBinaryOp("+", currentValueLLVM, one, "map_iter_index.inc")
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to increment loop variable: %w", err),
		}, nil
	}
	
	incrementedValueLLVM, ok := incrementedValue.(llvalue.Value)
	if !ok {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("CreateBinaryOp returned invalid type: %T", incrementedValue),
		}, nil
	}
	
	// 存储递增后的值
	err = irManager.CreateStore(incrementedValueLLVM, loopVarPtrValue)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to store incremented value: %w", err),
		}, nil
	}
	
	// 跳转回条件块
	err = irManager.CreateBr(condBlock)
	if err != nil {
		cfg.PopControlFlowContext()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create branch back to condition: %w", err),
		}, nil
	}
	
	// 12. 设置结束块为当前块
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
// 发出 br 后必须切换当前块，否则循环体内 break 之后的语句会落在同一块内（terminator 之后不能有指令），导致死循环。
func (cfg *ControlFlowGeneratorImpl) GenerateBreakStatement(irManager generation.IRModuleManager, stmt *entities.BreakStmt) (*generation.StatementGenerationResult, error) {
	if cfg.breakTarget == "" {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("break statement outside of loop"),
		}, nil
	}

	destBlock, err := irManager.GetBlockByName(cfg.breakTarget)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("break target block not found: %w", err),
		}, nil
	}
	if err := irManager.CreateBr(destBlock); err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create break branch: %w", err),
		}, nil
	}
	// 创建并切换到“后继”块，使 break 之后的语句生成到死代码块，当前块仅以 br 结束
	name := "after.break." + strconv.Itoa(cfg.breakExitCounter)
	cfg.breakExitCounter++
	nextBlock, err := irManager.CreateBasicBlock(name)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create block after break: %w", err),
		}, nil
	}
	if err := irManager.SetCurrentBasicBlock(nextBlock); err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set block after break: %w", err),
		}, nil
	}
	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// GenerateContinueStatement 生成continue语句
// 发出 br 后必须切换当前块，否则循环体内 continue 之后的语句会落在同一块内（terminator 之后不能有指令），导致错误控制流。
func (cfg *ControlFlowGeneratorImpl) GenerateContinueStatement(irManager generation.IRModuleManager, stmt *entities.ContinueStmt) (*generation.StatementGenerationResult, error) {
	if cfg.continueTarget == "" {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("continue statement outside of loop"),
		}, nil
	}

	destBlock, err := irManager.GetBlockByName(cfg.continueTarget)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("continue target block not found: %w", err),
		}, nil
	}
	if err := irManager.CreateBr(destBlock); err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create continue branch: %w", err),
		}, nil
	}
	// 创建并切换到“后继”块，使 continue 之后的语句生成到死代码块
	name := "after.continue." + strconv.Itoa(cfg.continueExitCounter)
	cfg.continueExitCounter++
	nextBlock, err := irManager.CreateBasicBlock(name)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create block after continue: %w", err),
		}, nil
	}
	if err := irManager.SetCurrentBasicBlock(nextBlock); err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set block after continue: %w", err),
		}, nil
	}
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
