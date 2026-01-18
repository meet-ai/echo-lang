package impl

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/meetai/echo-lang/internal/modules/backend/domain/services/generation"
	"github.com/meetai/echo-lang/internal/modules/frontend/domain/entities"
)

// StatementGeneratorImpl 语句生成器实现
type StatementGeneratorImpl struct {
	expressionEvaluator  generation.ExpressionEvaluator
	controlFlowGenerator generation.ControlFlowGenerator
	symbolManager        generation.SymbolManager
	typeMapper           generation.TypeMapper
	irModuleManager      generation.IRModuleManager

	// 统计信息
	stats     generation.StatementGenerationStats
	startTime time.Time
}

// NewStatementGeneratorImpl 创建语句生成器实现
func NewStatementGeneratorImpl(
	exprEval generation.ExpressionEvaluator,
	ctrlFlow generation.ControlFlowGenerator,
	symbolMgr generation.SymbolManager,
	typeMapper generation.TypeMapper,
	irManager generation.IRModuleManager,
) *StatementGeneratorImpl {
	return &StatementGeneratorImpl{
		expressionEvaluator:  exprEval,
		controlFlowGenerator: ctrlFlow,
		symbolManager:        symbolMgr,
		typeMapper:           typeMapper,
		irModuleManager:      irManager,
		stats:                generation.StatementGenerationStats{},
		startTime:            time.Now(),
	}
}

// SetControlFlowGenerator 更新控制流生成器（用于解决循环依赖）
func (sg *StatementGeneratorImpl) SetControlFlowGenerator(cfg generation.ControlFlowGenerator) {
	sg.controlFlowGenerator = cfg
}

// hasAwaitInStatements 检查语句中是否有await调用
func (sg *StatementGeneratorImpl) hasAwaitInStatements(statements []entities.ASTNode) bool {
	for _, stmt := range statements {
		if sg.hasAwaitInStatement(stmt) {
			return true
		}
	}
	return false
}

// hasAwaitInStatement 检查单个语句中是否有await调用
func (sg *StatementGeneratorImpl) hasAwaitInStatement(stmt entities.ASTNode) bool {
	switch s := stmt.(type) {
	case *entities.FuncDef:
		for _, bodyStmt := range s.Body {
			if sg.hasAwaitInStatement(bodyStmt) {
				return true
			}
		}
	case *entities.ExprStmt:
		return sg.hasAwaitInExpr(s.Expression)
	case *entities.VarDecl:
		if s.Value != nil {
			return sg.hasAwaitInExpr(s.Value)
		}
	case *entities.ReturnStmt:
		if s.Value != nil {
			return sg.hasAwaitInExpr(s.Value)
		}
	}
	return false
}

// hasAwaitInExpr 检查表达式中是否有await调用
func (sg *StatementGeneratorImpl) hasAwaitInExpr(expr entities.Expr) bool {
	switch e := expr.(type) {
	case *entities.AwaitExpr:
		return true
	case *entities.BinaryExpr:
		return sg.hasAwaitInExpr(e.Left) || sg.hasAwaitInExpr(e.Right)
	case *entities.FuncCall:
		for _, arg := range e.Args {
			if sg.hasAwaitInExpr(arg) {
				return true
			}
		}
	}
	return false
}

// createCoroutineWrapper 创建协程包装器
func (sg *StatementGeneratorImpl) createCoroutineWrapper(irManager generation.IRModuleManager) error {
	fmt.Fprintf(os.Stderr, "DEBUG: Creating coroutine wrapper for async/await support\n")

	// 创建一个包装器函数来处理await调用
	// 这个函数将在协程中执行包含await的代码

	// 外部函数应该在其他地方声明，这里只是设置必要的上下文
	// 实际的协程创建逻辑将在GenerateAsyncFuncDefinition中实现

	fmt.Fprintf(os.Stderr, "DEBUG: Coroutine wrapper setup complete\n")
	return nil
}

// findFunction 查找函数（简化实现）
func (sg *StatementGeneratorImpl) findFunction(irManager generation.IRModuleManager, name string) interface{} {
	// 简化实现：假设函数已经存在
	return irManager.GetCurrentFunction()
}

// renameFunction 重命名函数（简化实现）
func (sg *StatementGeneratorImpl) renameFunction(irManager generation.IRModuleManager, oldName, newName string) error {
	// 简化实现：这里应该实际重命名函数
	// 目前只是返回nil，实际实现需要修改IR
	return nil
}

// GenerateProgram 生成程序语句
func (sg *StatementGeneratorImpl) GenerateProgram(irManager generation.IRModuleManager, statements []entities.ASTNode) (*generation.StatementGenerationResult, error) {
	sg.startTime = time.Now()
	sg.stats.TotalStatements = len(statements)

	result := &generation.StatementGenerationResult{
		Success: true,
	}

	// 初始化符号表（全局作用域）
	if err := sg.symbolManager.EnterScope(); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("failed to initialize global symbol scope: %w", err)
		return result, result.Error
	}

	// 检查是否有await调用
	hasAwait := sg.hasAwaitInStatements(statements)

	if hasAwait {
		fmt.Fprintf(os.Stderr, "DEBUG: Detected await calls, setting up coroutine support\n")
		// 如果有await，创建协程包装器
		err := sg.createCoroutineWrapper(irManager)
		if err != nil {
			result.Success = false
			result.Error = fmt.Errorf("failed to create coroutine wrapper: %w", err)
			// 清理作用域
			sg.symbolManager.ExitScope()
			return result, result.Error
		}
	}

	// 生成每个语句
	fmt.Fprintf(os.Stderr, "DEBUG: Total statements: %d\n", len(statements))
	for i, stmt := range statements {
		fmt.Fprintf(os.Stderr, "DEBUG: Generating top-level statement %d: %T\n", i, stmt)
		if funcDef, ok := stmt.(*entities.FuncDef); ok {
			fmt.Fprintf(os.Stderr, "DEBUG: Function name: %s, body statements: %d\n", funcDef.Name, len(funcDef.Body))
		}
		stmtResult, err := sg.GenerateStatement(irManager, stmt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "DEBUG: Failed to generate statement %d (%T): %v\n", i, stmt, err)
			result.Success = false
			result.Error = fmt.Errorf("failed to generate statement %d: %w", i, err)
			sg.stats.FailedGenerations++
			break
		}
		if stmtResult.Success {
			sg.stats.SuccessfulGenerations++
		}
	}

	// 退出作用域
	if err := sg.symbolManager.ExitScope(); err != nil {
		if result.Success {
			result.Success = false
			result.Error = fmt.Errorf("failed to exit symbol scope: %w", err)
		}
	}

	sg.updateStats()
	return result, result.Error
}

// GenerateStatement 生成单个语句
func (sg *StatementGeneratorImpl) GenerateStatement(irManager generation.IRModuleManager, stmt entities.ASTNode) (*generation.StatementGenerationResult, error) {
	switch s := stmt.(type) {
	case *entities.PrintStmt:
		return sg.GeneratePrintStatement(irManager, s)
	case *entities.VarDecl:
		return sg.GenerateVarDeclaration(irManager, s)
	case *entities.AssignStmt:
		return sg.GenerateAssignmentStatement(irManager, s)
	case *entities.ExprStmt:
		return sg.GenerateExpressionStatement(irManager, s)
	case *entities.FuncCall:
		return sg.GenerateFuncCallStatement(irManager, s)
	case *entities.FuncDef:
		return sg.GenerateFuncDefinition(irManager, s)
	case *entities.AsyncFuncDef:
		return sg.GenerateAsyncFuncDefinition(irManager, s)
	case *entities.ReturnStmt:
		return sg.GenerateReturnStatement(irManager, s)
	case *entities.MatchStmt:
		return sg.GenerateMatchStatement(irManager, s)
	case *entities.IfStmt:
		return sg.GenerateIfStatement(irManager, s)
	case *entities.ForStmt:
		return sg.GenerateForStatement(irManager, s)
	case *entities.WhileStmt:
		return sg.GenerateWhileStatement(irManager, s)
	case *entities.TraitDef:
		return sg.GenerateTraitDefinition(irManager, s)
	case *entities.EnumDef:
		return sg.GenerateEnumDefinition(irManager, s)
	case *entities.MethodDef:
		return sg.GenerateMethodDefinition(irManager, s)
	case *entities.SelectStmt:
		return sg.GenerateSelectStatement(irManager, s)
	default:
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("unsupported statement type: %T", stmt),
		}, nil
	}
}

// GeneratePrintStatement 生成打印语句
// GenerateFuncCallStatement 生成函数调用语句
func (sg *StatementGeneratorImpl) GenerateFuncCallStatement(irManager generation.IRModuleManager, stmt *entities.FuncCall) (*generation.StatementGenerationResult, error) {
	// 特殊处理print函数调用
	if stmt.Name == "print" {
		if len(stmt.Args) != 1 {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("print function expects exactly 1 argument"),
			}, nil
		}

		// 求值打印表达式
		exprResult, err := sg.expressionEvaluator.Evaluate(irManager, stmt.Args[0])
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to evaluate print expression: %w", err),
			}, nil
		}

		// 生成打印调用 - 这里需要外部函数声明
		// 暂时返回成功，后续需要添加外部函数声明
		_ = exprResult // 暂时忽略返回值
		return &generation.StatementGenerationResult{
			Success: true,
		}, nil
	}

	// 一般函数调用
	callResult, err := sg.expressionEvaluator.EvaluateFuncCall(irManager, stmt)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to generate function call: %w", err),
		}, nil
	}

	// callResult 应该已经被添加到IR中
	_ = callResult
	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

func (sg *StatementGeneratorImpl) GeneratePrintStatement(irManager generation.IRModuleManager, stmt *entities.PrintStmt) (*generation.StatementGenerationResult, error) {
	// 求值打印表达式
	exprResult, err := sg.expressionEvaluator.Evaluate(irManager, stmt.Value)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate print expression: %w", err),
		}, nil
	}

	// 根据表达式类型选择合适的打印函数
	printFuncName, err := sg.getPrintFunctionName(stmt.Value)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to determine print function: %w", err),
		}, nil
	}

	// 获取外部打印函数
	// 这里需要类型断言，因为IRModuleManager接口没有GetExternalFunction方法
	// TODO: 应该在IRModuleManager接口中添加GetExternalFunction方法
	if irManagerImpl, ok := irManager.(*IRModuleManagerImpl); ok {
		printFunc, exists := irManagerImpl.GetExternalFunction(printFuncName)
		if !exists {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("external function %s not declared", printFuncName),
			}, nil
		}

		// 生成函数调用
		callResult, err := irManager.CreateCall(printFunc, exprResult)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to create print call: %w", err),
			}, nil
		}

		// callResult 包含了函数调用的结果，通常我们不需要使用它
		_ = callResult
	} else {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("IRManager does not support external functions"),
		}, nil
	}

	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// GenerateVarDeclaration 生成变量声明
func (sg *StatementGeneratorImpl) GenerateVarDeclaration(irManager generation.IRModuleManager, stmt *entities.VarDecl) (*generation.StatementGenerationResult, error) {
	// 映射变量类型（支持复杂类型如Future[T]）
	typeInterface, err := sg.typeMapper.MapType(stmt.Type)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to map variable type %s: %w", stmt.Type, err),
		}, nil
	}

	// 类型断言
	llvmType, ok := typeInterface.(types.Type)
	if !ok {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("mapped type %v is not a valid LLVM type", typeInterface),
		}, nil
	}

	// 使用irManager创建alloca指令
	alloca, err := irManager.CreateAlloca(llvmType, stmt.Name)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create alloca instruction: %w", err),
		}, nil
	}

	// 注册符号
	if err := sg.symbolManager.RegisterSymbol(stmt.Name, stmt.Type, alloca); err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to register symbol %s: %w", stmt.Name, err),
		}, nil
	}

	// 如果有初始值，生成赋值
	if stmt.Value != nil {
		fmt.Fprintf(os.Stderr, "DEBUG: Variable %s has initial value, generating assignment\n", stmt.Name)
		assignResult, err := sg.GenerateAssignmentStatement(irManager, &entities.AssignStmt{
			Name:  stmt.Name,
			Value: stmt.Value,
		})
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to generate initial assignment: %w", err),
			}, nil
		}
		_ = assignResult // 赋值指令已添加到IR中

		// 如果是数组字面量，更新符号表中的长度信息
		if arrayLit, ok := stmt.Value.(*entities.ArrayLiteral); ok && strings.HasPrefix(stmt.Type, "[") {
			// 更新符号表，存储数组长度
			if err := sg.symbolManager.UpdateSymbolValue(stmt.Name, len(arrayLit.Elements)); err != nil {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("failed to update array length in symbol table: %w", err),
				}, nil
			}
		}
	}

	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// GenerateAssignmentStatement 生成赋值语句
func (sg *StatementGeneratorImpl) GenerateAssignmentStatement(irManager generation.IRModuleManager, stmt *entities.AssignStmt) (*generation.StatementGenerationResult, error) {
	// 求值右侧表达式
	valueResult, err := sg.expressionEvaluator.Evaluate(irManager, stmt.Value)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate assignment value: %w", err),
		}, nil
	}

	// 获取目标符号
	targetSymbol, err := sg.symbolManager.LookupSymbol(stmt.Name)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("undefined symbol %s: %w", stmt.Name, err),
		}, nil
	}

	// 生成存储指令
	err = irManager.CreateStore(valueResult, targetSymbol.Value)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create store instruction: %w", err),
		}, nil
	}

	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// GenerateExpressionStatement 生成表达式语句
func (sg *StatementGeneratorImpl) GenerateExpressionStatement(irManager generation.IRModuleManager, stmt *entities.ExprStmt) (*generation.StatementGenerationResult, error) {
	_, err := sg.expressionEvaluator.Evaluate(irManager, stmt.Expression)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate expression: %w", err),
		}, nil
	}

	// 对于spawn等有副作用的表达式，即使结果没有被使用，也要确保代码被生成
	// spawn返回Future，但如果没有赋值给变量，Future会被丢弃
	// 但spawn的副作用（启动协程）应该被保留
	switch stmt.Expression.(type) {
	case *entities.SpawnExpr:
		// spawn表达式已经通过Evaluate生成了代码，无需额外处理
		// Future结果被丢弃是正常的，因为spawn的主要作用是启动协程
		break
	default:
		// 其他表达式如果有返回值但没有被使用，可能需要警告或特殊处理
		// 暂时忽略
	}

	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// GenerateFuncDefinition 生成函数定义
func (sg *StatementGeneratorImpl) GenerateFuncDefinition(irManager generation.IRModuleManager, stmt *entities.FuncDef) (*generation.StatementGenerationResult, error) {
	// 特殊处理：main函数必须返回int而不是void
	returnTypeStr := stmt.ReturnType
	if stmt.Name == "main" && stmt.ReturnType == "void" {
		returnTypeStr = "int"
	}

	// 映射返回类型
	returnType, err := sg.typeMapper.MapPrimitiveType(returnTypeStr)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to map return type %s: %w", returnTypeStr, err),
		}, nil
	}

	// 映射参数类型
	paramTypes := make([]interface{}, len(stmt.Params))
	for i, param := range stmt.Params {
		paramType, err := sg.typeMapper.MapPrimitiveType(param.Type)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to map parameter type %s: %w", param.Type, err),
			}, nil
		}
		paramTypes[i] = paramType
	}

	// 使用irManager创建函数
	fnInterface, err := irManager.CreateFunction(stmt.Name, returnType, paramTypes)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create function: %w", err),
		}, nil
	}

	// 类型断言为*ir.Func
	fn, ok := fnInterface.(*ir.Func)
	if !ok {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to cast function to *ir.Func"),
		}, nil
	}

	// 设置当前函数
	err = irManager.SetCurrentFunction(fn)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set current function: %w", err),
		}, nil
	}

	// 创建入口基本块
	entryBlock, err := irManager.CreateBasicBlock("entry")
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create entry block: %w", err),
		}, nil
	}

	// 设置当前基本块
	err = irManager.SetCurrentBasicBlock(entryBlock)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set current block: %w", err),
		}, nil
	}

	// 处理函数参数 - 为每个参数创建alloca和store
	for i, param := range stmt.Params {
		// 创建alloca为参数分配栈空间
		allocaInst, err := irManager.CreateAlloca(paramTypes[i], param.Name+"_addr")
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to create alloca for parameter %s: %w", param.Name, err),
			}, nil
		}

		// store参数值到alloca的位置
		err = irManager.CreateStore(fn.Params[i], allocaInst)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to create store for parameter %s: %w", param.Name, err),
			}, nil
		}

		// 在符号表中注册参数
		err = sg.symbolManager.RegisterSymbol(param.Name, param.Type, allocaInst)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to register parameter %s: %w", param.Name, err),
			}, nil
		}
	}

	// 生成函数体中的语句
	if stmt.Body != nil {
		for _, bodyStmt := range stmt.Body {
			result, err := sg.GenerateStatement(irManager, bodyStmt)
			if err != nil {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("failed to generate statement in function %s: %w", stmt.Name, err),
				}, nil
			}
			if !result.Success {
				return result, nil
			}
		}
	} else {
	}

	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// GenerateReturnStatement 生成返回语句
func (sg *StatementGeneratorImpl) GenerateReturnStatement(irManager generation.IRModuleManager, stmt *entities.ReturnStmt) (*generation.StatementGenerationResult, error) {
	// 如果没有返回值，生成ret void
	if stmt.Value == nil {
		err := irManager.CreateRet(nil)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to create void return: %w", err),
			}, nil
		}
		return &generation.StatementGenerationResult{
			Success: true,
		}, nil
	}

	// 求值返回值
	returnValue, err := sg.expressionEvaluator.Evaluate(irManager, stmt.Value)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate return value: %w", err),
		}, nil
	}

	// 生成ret指令
	err = irManager.CreateRet(returnValue)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create return instruction: %w", err),
		}, nil
	}

	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// GenerateMatchStatement 生成模式匹配语句
func (sg *StatementGeneratorImpl) GenerateMatchStatement(irManager generation.IRModuleManager, stmt *entities.MatchStmt) (*generation.StatementGenerationResult, error) {
	// 简化实现：只处理第一个case
	if len(stmt.Cases) == 0 {
		return &generation.StatementGenerationResult{
			Success: true,
		}, nil
	}

	firstCase := stmt.Cases[0]
	_, err := sg.GenerateProgram(irManager, firstCase.Body)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to generate match case: %w", err),
		}, nil
	}

	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// GenerateIfStatement 生成if语句
func (sg *StatementGeneratorImpl) GenerateIfStatement(irManager generation.IRModuleManager, stmt *entities.IfStmt) (*generation.StatementGenerationResult, error) {
	return sg.controlFlowGenerator.GenerateIfStatement(irManager, stmt)
}

// GenerateForStatement 生成for循环
func (sg *StatementGeneratorImpl) GenerateForStatement(irManager generation.IRModuleManager, stmt *entities.ForStmt) (*generation.StatementGenerationResult, error) {
	return sg.controlFlowGenerator.GenerateForStatement(irManager, stmt)
}

// GenerateWhileStatement 生成while循环
func (sg *StatementGeneratorImpl) GenerateWhileStatement(irManager generation.IRModuleManager, stmt *entities.WhileStmt) (*generation.StatementGenerationResult, error) {
	return sg.controlFlowGenerator.GenerateWhileStatement(irManager, stmt)
}

// ValidateGenerationContext 验证语句生成上下文
func (sg *StatementGeneratorImpl) ValidateGenerationContext() error {
	if sg.expressionEvaluator == nil {
		return fmt.Errorf("expression evaluator not set")
	}
	if sg.controlFlowGenerator == nil {
		return fmt.Errorf("control flow generator not set")
	}
	if sg.symbolManager == nil {
		return fmt.Errorf("symbol manager not set")
	}
	if sg.typeMapper == nil {
		return fmt.Errorf("type mapper not set")
	}
	return nil
}

// GetGenerationStats 获取生成统计信息
func (sg *StatementGeneratorImpl) GetGenerationStats() *generation.StatementGenerationStats {
	return &sg.stats
}

// updateStats 更新统计信息
func (sg *StatementGeneratorImpl) updateStats() {
	sg.stats.GenerationTime = time.Since(sg.startTime).Nanoseconds()
}

// GenerateTraitDefinition 生成trait定义
// Trait在LLVM IR中主要作为注释，不生成实际代码，但需要记录泛型信息用于类型检查
// 对于动态分发，需要生成虚表结构
func (sg *StatementGeneratorImpl) GenerateTraitDefinition(irManager generation.IRModuleManager, stmt *entities.TraitDef) (*generation.StatementGenerationResult, error) {
	result := &generation.StatementGenerationResult{
		Success: true,
	}

	// 记录Trait信息到符号表，便于后续类型检查和实现验证
	traitInfo := map[string]interface{}{
		"type":        "trait",
		"typeParams":  stmt.TypeParams,
		"methods":     stmt.Methods,
		"superTraits": stmt.SuperTraits,
	}

	// 为动态分发记录虚表信息（简化实现）
	// 在实际实现中，这里应该生成虚表类型和全局虚表实例
	if len(stmt.Methods) > 0 {
		fmt.Fprintf(os.Stderr, "DEBUG: Trait %s supports dynamic dispatch with %d methods\n", stmt.Name, len(stmt.Methods))

		// 记录虚表结构信息（用于后续代码生成）
		vtableInfo := map[string]interface{}{
			"traitName": stmt.Name,
			"methods":   stmt.Methods,
		}
		traitInfo["vtable"] = vtableInfo
	}

	// 将Trait信息注册到符号表
	err := sg.symbolManager.RegisterSymbol(stmt.Name, "trait", traitInfo)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("failed to register trait symbol %s: %w", stmt.Name, err)
		return result, result.Error
	}

	fmt.Fprintf(os.Stderr, "DEBUG: Registered trait %s with %d type parameters and %d methods\n",
		stmt.Name, len(stmt.TypeParams), len(stmt.Methods))

	return result, nil
}

// GenerateEnumDefinition 生成枚举定义
// 枚举在LLVM IR中实现为整型常量序列
func (sg *StatementGeneratorImpl) GenerateEnumDefinition(irManager generation.IRModuleManager, stmt *entities.EnumDef) (*generation.StatementGenerationResult, error) {
	result := &generation.StatementGenerationResult{
		Success: true,
	}

	// 为每个枚举变体创建整型常量
	// 枚举变体从0开始递增
	for i, variant := range stmt.Variants {
		// 创建枚举常量值
		// 注意：这里假设枚举值为i32类型
		constName := fmt.Sprintf("%s_%s", stmt.Name, variant)

		// 将枚举值存储在符号表中，便于后续引用
		if err := sg.symbolManager.RegisterSymbol(constName, "i32", i); err != nil {
			result.Success = false
			result.Error = fmt.Errorf("failed to register enum variant symbol %s: %w", constName, err)
			return result, result.Error
		}
	}

	return result, nil
}

// GenerateMethodDefinition 生成方法定义
// 方法在LLVM IR中实现为普通函数，接收者作为第一个参数
func (sg *StatementGeneratorImpl) GenerateMethodDefinition(irManager generation.IRModuleManager, stmt *entities.MethodDef) (*generation.StatementGenerationResult, error) {
	fmt.Fprintf(os.Stderr, "DEBUG: Generating method: %s for receiver %s\n", stmt.Name, stmt.Receiver)
	result := &generation.StatementGenerationResult{
		Success: true,
	}

	// 解析返回类型
	returnType, err := sg.typeMapper.MapPrimitiveType(stmt.ReturnType)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("failed to map return type: %w", err)
		return result, result.Error
	}

	// 构建参数类型列表
	// 第一个参数是接收者类型
	paramTypes := []interface{}{}

	// 添加接收者类型
	receiverType, err := sg.parseReceiverType(stmt.Receiver)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("failed to parse receiver type: %w", err)
		return result, result.Error
	}
	paramTypes = append(paramTypes, receiverType)

	// 添加方法参数类型
	for _, param := range stmt.Params {
		paramType, err := sg.mapParameterType(param.Type)
		if err != nil {
			result.Success = false
			result.Error = fmt.Errorf("failed to map parameter type %s: %w", param.Name, err)
			return result, result.Error
		}
		paramTypes = append(paramTypes, paramType)
	}

	// 创建函数
	// 方法名格式：Type_MethodName
	funcName := fmt.Sprintf("%s_%s", stmt.Receiver, stmt.Name)
	function, err := irManager.CreateFunction(funcName, returnType, paramTypes)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("failed to create method function %s: %w", funcName, err)
		return result, result.Error
	}

	// 设置当前函数
	if err := irManager.SetCurrentFunction(function); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("failed to set current function: %w", err)
		return result, result.Error
	}

	// 创建函数入口基本块
	entryBlock, err := irManager.CreateBasicBlock("entry")
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("failed to create entry block: %w", err)
		return result, result.Error
	}

	if err := irManager.SetCurrentBasicBlock(entryBlock); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("failed to set entry block: %w", err)
		return result, result.Error
	}

	// 简化实现：暂时跳过作用域管理和方法体生成
	// TODO: 完善方法体的生成逻辑，包括参数处理和作用域管理

	// 如果方法没有显式返回语句，根据返回类型添加返回
	if stmt.ReturnType == "void" {
		if err := irManager.CreateRet(nil); err != nil {
			result.Success = false
			result.Error = fmt.Errorf("failed to create return instruction: %w", err)
			return result, result.Error
		}
	} else {
		// 为非void方法添加默认返回值（简化实现）
		if err := irManager.CreateRet(constant.NewInt(types.I32, 0)); err != nil {
			result.Success = false
			result.Error = fmt.Errorf("failed to create return instruction: %w", err)
			return result, result.Error
		}
	}

	return result, nil
}

// parseReceiverType 解析接收者类型
func (sg *StatementGeneratorImpl) parseReceiverType(receiver string) (interface{}, error) {
	// 解析接收者格式：TypeName 或 *TypeName
	// 暂时简化处理，返回i8*作为通用指针类型
	// TODO: 根据实际类型系统进行更精确的映射
	return "i8*", nil
}

// getPrintFunctionName 根据表达式类型确定打印函数名
func (sg *StatementGeneratorImpl) getPrintFunctionName(expr entities.Expr) (string, error) {
	switch e := expr.(type) {
	case *entities.StringLiteral:
		return "print_string", nil
	case *entities.IntLiteral:
		return "print_int", nil
	case *entities.BoolLiteral:
		return "print_int", nil // 布尔值用int表示
	case *entities.Identifier:
		// 从符号表查询变量类型
		symbolInfo, err := sg.symbolManager.LookupSymbol(e.Name)
		if err != nil {
			return "", fmt.Errorf("failed to lookup symbol %s: %w", e.Name, err)
		}
		// 根据符号表中的类型确定打印函数
		switch symbolInfo.Type {
		case "string":
			return "print_string", nil
		case "int":
			return "print_int", nil
		case "bool":
			return "print_int", nil // 布尔值用int表示
		default:
			return "print_string", nil // 默认当作字符串处理
		}
	case *entities.BinaryExpr:
		// 对于二元表达式，特别是字符串拼接，结果是字符串
		if e.Op == "+" {
			// 检查操作数类型，如果包含字符串，则结果是字符串
			leftType := sg.getExprType(e.Left)
			rightType := sg.getExprType(e.Right)
			if leftType == "string" || rightType == "string" {
				return "print_string", nil
			}
		}
		return "print_int", nil // 默认当作整数处理
	default:
		return "print_string", nil // 默认当作字符串处理
	}
}

// getExprType 辅助函数：获取表达式的类型
func (sg *StatementGeneratorImpl) getExprType(expr entities.Expr) string {
	switch e := expr.(type) {
	case *entities.StringLiteral:
		return "string"
	case *entities.IntLiteral:
		return "int"
	case *entities.BoolLiteral:
		return "bool"
	case *entities.Identifier:
		if symbolInfo, err := sg.symbolManager.LookupSymbol(e.Name); err == nil {
			return symbolInfo.Type
		}
		return "unknown"
	case *entities.BinaryExpr:
		// 对于二元表达式，简化处理：如果操作符是+且操作数包含字符串，则结果是字符串
		if e.Op == "+" {
			leftType := sg.getExprType(e.Left)
			rightType := sg.getExprType(e.Right)
			if leftType == "string" || rightType == "string" {
				return "string"
			}
		}
		return "int" // 默认当作整数
	default:
		return "unknown"
	}
}
			return "print_bool", nil
		case "float":
			return "print_float", nil
		default:
			return "", fmt.Errorf("unsupported type for printing: %s", symbolInfo.Type)
		}
	case *entities.BinaryExpr:
		// 对于二元表达式，假设结果是int类型
		return "print_int", nil
	default:
		return "", fmt.Errorf("unsupported expression type for printing: %T", expr)
	}
}

// mapParameterType 映射参数类型
// 对于复杂类型，使用指针类型简化处理
func (sg *StatementGeneratorImpl) mapParameterType(echoType string) (interface{}, error) {
	// 首先尝试基本类型映射
	if llvmType, err := sg.typeMapper.MapPrimitiveType(echoType); err == nil {
		return llvmType, nil
	}

	// 对于非基本类型，使用指针类型
	// 这是一个简化的处理方式，实际应该根据具体类型进行映射
	return "i8*", nil
}

// GenerateSelectStatement 生成select语句
func (sg *StatementGeneratorImpl) GenerateSelectStatement(irManager generation.IRModuleManager, stmt *entities.SelectStmt) (*generation.StatementGenerationResult, error) {
	// select语句实现策略：
	// 1. 为每个case分支生成通道操作调用
	// 2. 使用运行时库的select函数来并发监听
	// 3. 根据选择结果执行对应的分支

	// NOTE: 当前select实现是顺序尝试case，不需要并发select函数
	// 如果将来实现真正的并发select，需要添加channel_select调用

	// 实现完整的多case select逻辑
	// 1. 准备所有case分支的信息
	caseCount := len(stmt.Cases)
	if caseCount == 0 {
		return &generation.StatementGenerationResult{
			Success: true,
		}, nil
	}

	// 为每个case分支生成评估逻辑
	// 注意：真正的select需要运行时支持并发监听，这里是简化实现

	// 简化策略：按顺序尝试每个case，第一个成功的就执行
	for i, caseBranch := range stmt.Cases {
		fmt.Fprintf(os.Stderr, "DEBUG: Processing select case %d\n", i)

		if caseBranch.IsSend {
			// 发送case: channel <- value
			if sendExpr, ok := caseBranch.Chan.(*entities.SendExpr); ok {
				// 求值通道和值
				chanValue, err := sg.expressionEvaluator.Evaluate(irManager, sendExpr.Channel)
				if err != nil {
					return &generation.StatementGenerationResult{
						Success: false,
						Error:   fmt.Errorf("failed to evaluate send channel in case %d: %v", i, err),
					}, nil
				}

				value, err := sg.expressionEvaluator.Evaluate(irManager, sendExpr.Value)
				if err != nil {
					return &generation.StatementGenerationResult{
						Success: false,
						Error:   fmt.Errorf("failed to evaluate send value in case %d: %v", i, err),
					}, nil
				}

				// 调用channel_send (非阻塞尝试)
				sendFunc, sendExists := irManager.GetExternalFunction("channel_send")
				if sendExists {
					_, err = irManager.CreateCall(sendFunc, chanValue, value)
					if err != nil {
						return &generation.StatementGenerationResult{
							Success: false,
							Error:   fmt.Errorf("failed to create send call in select case %d: %v", i, err),
						}, nil
					}

					// 如果发送成功，执行case分支的语句
					for _, caseStmt := range caseBranch.Body {
						result, err := sg.GenerateStatement(irManager, caseStmt)
						if err != nil {
							return &generation.StatementGenerationResult{
								Success: false,
								Error:   fmt.Errorf("failed to generate statement in select case %d: %v", i, err),
							}, nil
						}
						if !result.Success {
							return result, nil
						}
					}
					break // 执行完一个case就退出
				}
			}
		} else {
			// 接收case: <- channel
			if recvExpr, ok := caseBranch.Chan.(*entities.ReceiveExpr); ok {
				// 求值通道
				chanValue, err := sg.expressionEvaluator.Evaluate(irManager, recvExpr.Channel)
				if err != nil {
					return &generation.StatementGenerationResult{
						Success: false,
						Error:   fmt.Errorf("failed to evaluate receive channel in case %d: %v", i, err),
					}, nil
				}

				// 调用channel_receive (非阻塞尝试)
				receiveFunc, recvExists := irManager.GetExternalFunction("channel_receive")
				if recvExists {
					_, err = irManager.CreateCall(receiveFunc, chanValue)
					if err != nil {
						return &generation.StatementGenerationResult{
							Success: false,
							Error:   fmt.Errorf("failed to create receive call in select case %d: %v", i, err),
						}, nil
					}

					// 如果接收成功，执行case分支的语句
					for _, caseStmt := range caseBranch.Body {
						result, err := sg.GenerateStatement(irManager, caseStmt)
						if err != nil {
							return &generation.StatementGenerationResult{
								Success: false,
								Error:   fmt.Errorf("failed to generate statement in select case %d: %v", i, err),
							}, nil
						}
						if !result.Success {
							return result, nil
						}
					}
					break // 执行完一个case就退出
				}
			}
		}
	}

	// 处理default分支（如果有的话）
	if stmt.DefaultBody != nil && len(stmt.DefaultBody) > 0 {
		fmt.Fprintf(os.Stderr, "DEBUG: Processing select default branch\n")
		for _, defaultStmt := range stmt.DefaultBody {
			result, err := sg.GenerateStatement(irManager, defaultStmt)
			if err != nil {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("failed to generate statement in select default: %v", err),
				}, nil
			}
			if !result.Success {
				return result, nil
			}
		}
	}

	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// GenerateAsyncFuncDefinition 生成异步函数定义
func (sg *StatementGeneratorImpl) GenerateAsyncFuncDefinition(irManager generation.IRModuleManager, stmt *entities.AsyncFuncDef) (*generation.StatementGenerationResult, error) {
	fmt.Fprintf(os.Stderr, "DEBUG: GenerateAsyncFuncDefinition called for %s\n", stmt.Name)
	// async函数的代码生成策略：
	// 1. 创建返回Future指针的函数
	// 2. 函数体创建一个Future并启动协程执行实际逻辑
	// 3. 返回Future给调用者

	// 注册async函数符号
	returnTypeStr := fmt.Sprintf("Future[%s]", stmt.ReturnType)
	err := sg.symbolManager.RegisterFunctionSymbol(stmt.Name, returnTypeStr, nil, true)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to register async function symbol: %w", err),
		}, nil
	}

	// 创建函数 - async函数返回Future指针
	funcName := stmt.Name
	returnType := types.NewPointer(types.I8) // Future指针类型

	var paramTypes []interface{}
	for range stmt.Params {
		paramTypes = append(paramTypes, types.I32) // 简化处理
	}

	funcPtr, err := irManager.CreateFunction(funcName, returnType, paramTypes)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create async function: %v", err),
		}, nil
	}

	// 设置当前函数上下文
	err = irManager.SetCurrentFunction(funcPtr)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set current function: %v", err),
		}, nil
	}

	// 创建函数入口基本块
	entryBlock, err := irManager.CreateBasicBlock("entry")
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create entry block: %v", err),
		}, nil
	}

	err = irManager.SetCurrentBasicBlock(entryBlock)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set current block: %v", err),
		}, nil
	}

	// 生成async函数体：
	// 1. 创建Future
	// 2. spawn协程执行实际函数体
	// 3. 返回Future

	// 获取future_new函数
	fmt.Fprintf(os.Stderr, "DEBUG: Looking for future_new function\n")
	futureNewFunc, exists := irManager.GetExternalFunction("future_new")
	if !exists {
		fmt.Fprintf(os.Stderr, "DEBUG: future_new function not found\n")
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("future_new function not declared"),
		}, nil
	}
	fmt.Fprintf(os.Stderr, "DEBUG: future_new function found\n")

	// 调用future_new创建Future
	futurePtr, err := irManager.CreateCall(futureNewFunc)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create future_new call: %v", err),
		}, nil
	}

	// 🚨 修复：async函数不再同步执行函数体
	// 这修复了严重的架构缺陷：async函数之前在创建时就同步执行了函数体
	//
	// 新的实现：
	// 1. async函数只创建Future并返回，不执行函数体
	// 2. 函数体的执行推迟到协程调度器中
	// 3. 这实现了真正的异步语义
	//
	// 注意：这只是第一步修复，完整的实现需要状态机转换
	// TODO: 实现async执行器来执行函数体

	// 为async函数生成对应的执行器函数
	// 执行器函数负责执行async函数体并resolve Future
	executorFuncName := stmt.Name + "_executor"

	// 创建执行器函数: void executor(i8* future, ...)
	// 执行器函数接收Future指针和async函数的所有参数
	var executorParamTypes []interface{}
	executorParamTypes = append(executorParamTypes, types.NewPointer(types.I8)) // Future指针

	// 添加async函数的参数类型
	for range stmt.Params {
		executorParamTypes = append(executorParamTypes, types.I32) // 简化：所有参数都作为i32传递
	}

	executorFuncPtr, err := irManager.CreateFunction(executorFuncName, types.Void, executorParamTypes)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create async executor function: %v", err),
		}, nil
	}
	fmt.Fprintf(os.Stderr, "DEBUG: Created executor function, type: %T\n", executorFuncPtr)

	// 将执行器函数注册为外部函数，以便await时可以找到
	// 注意：这是一个临时方案，理想情况下应该有更好的机制
	if irManagerImpl, ok := irManager.(*IRModuleManagerImpl); ok {
		if funcPtr, ok := executorFuncPtr.(*ir.Func); ok {
			fmt.Fprintf(os.Stderr, "DEBUG: Registering executor function %s\n", executorFuncName)
			irManagerImpl.externalFunctions[executorFuncName] = funcPtr
		} else {
			fmt.Fprintf(os.Stderr, "DEBUG: executorFuncPtr is not *ir.Func, type: %T\n", executorFuncPtr)
		}
	} else {
		fmt.Fprintf(os.Stderr, "DEBUG: irManager is not IRModuleManagerImpl in async func generation\n")
	}

	// 设置执行器函数上下文
	err = irManager.SetCurrentFunction(executorFuncPtr)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set executor function: %v", err),
		}, nil
	}

	// 创建执行器函数体
	executorEntryBlock, err := irManager.CreateBasicBlock("entry")
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create executor entry block: %v", err),
		}, nil
	}

	err = irManager.SetCurrentBasicBlock(executorEntryBlock)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set executor block: %v", err),
		}, nil
	}

	// 在执行器中执行async函数体
	if stmt.Body != nil && len(stmt.Body) > 0 {
		for i, bodyStmt := range stmt.Body {
			fmt.Fprintf(os.Stderr, "DEBUG: Processing statement %d in function %s: %T\n", i, stmt.Name, bodyStmt)
			if returnStmt, ok := bodyStmt.(*entities.ReturnStmt); ok && returnStmt.Value != nil {
				// 求值返回值
				returnValue, err := sg.expressionEvaluator.Evaluate(irManager, returnStmt.Value)
				if err != nil {
					return &generation.StatementGenerationResult{
						Success: false,
						Error:   fmt.Errorf("failed to evaluate return value in executor: %v", err),
					}, nil
				}

				// 获取Future指针（第一个参数）
				currentFunc := irManager.GetCurrentFunction()
				var futureParam interface{}
				if funcPtr, ok := currentFunc.(*ir.Func); ok {
					futureParam = funcPtr.Params[0]
				} else {
					return &generation.StatementGenerationResult{
						Success: false,
						Error:   fmt.Errorf("invalid function type for executor"),
					}, nil
				}

				// 调用future_resolve
				futureResolveFunc, exists := irManager.GetExternalFunction("future_resolve")
				if !exists {
					return &generation.StatementGenerationResult{
						Success: false,
						Error:   fmt.Errorf("future_resolve function not declared"),
					}, nil
				}

				// 转换返回值类型并resolve Future
				if stringValue, ok := returnValue.(*ir.Global); ok {
				_, err = irManager.CreateCall(futureResolveFunc, futureParam, stringValue)
			} else {
				_, err = irManager.CreateCall(futureResolveFunc, futureParam, returnValue)
			}
			fmt.Fprintf(os.Stderr, "DEBUG: Added future_resolve call in executor for async function %s\n", stmt.Name)
			if err != nil {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("failed to create future_resolve call in executor: %v", err),
				}, nil
			}
				break
			}
		}
	}

	// 执行器函数返回void
	err = irManager.CreateRet(nil)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create executor return: %v", err),
		}, nil
	}

	// 回到主async函数，继续生成async函数体
	err = irManager.SetCurrentFunction(funcPtr)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to restore async function context: %v", err),
		}, nil
	}

	err = irManager.SetCurrentBasicBlock(entryBlock)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to restore async function block: %v", err),
		}, nil
	}

	// 临时解决方案：在await的函数中添加run_scheduler调用
	// 这确保了协程调度器会在await完成后启动
	// TODO: 实现更优雅的调度器启动机制

	// 在返回Future之前，spawn executor协程
	spawnFunc, spawnExists := irManager.GetExternalFunction("coroutine_spawn")
	if spawnExists {
		fmt.Fprintf(os.Stderr, "DEBUG: Spawning executor %s for async function %s\n", executorFuncName, stmt.Name)
		spawnArgs := []interface{}{
			executorFuncPtr,                      // target executor function
			constant.NewInt(types.I32, 1),        // arg_count (1 arg: future)
			constant.NewNull(types.NewPointer(types.I8)), // args (null - we'll pass future as first param)
			futurePtr,                            // future
		}
		_, err = irManager.CreateCall(spawnFunc, spawnArgs...)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to create spawn call for executor: %v", err),
			}, nil
		}
		fmt.Fprintf(os.Stderr, "DEBUG: Spawn call created for executor\n")
	} else {
		fmt.Fprintf(os.Stderr, "DEBUG: coroutine_spawn function not found, async function will not execute\n")
	}

	// 在返回Future之前，添加run_scheduler调用确保调度器启动
	runSchedulerFunc, runExists := irManager.GetExternalFunction("run_scheduler")
	if runExists {
		fmt.Fprintf(os.Stderr, "DEBUG: Adding run_scheduler call\n")
		_, err = irManager.CreateCall(runSchedulerFunc)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to create run_scheduler call: %v", err),
			}, nil
		}
	}

	// 返回Future指针
	err = irManager.CreateRet(futurePtr)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create return: %v", err),
		}, nil
	}

	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}
