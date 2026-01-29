package impl

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"echo/internal/modules/backend/domain/services/generation"
	"echo/internal/modules/frontend/domain/entities"
	parserPkg "echo/internal/modules/frontend/domain/parser"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	llvalue "github.com/llir/llvm/ir/value"
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
	
	// ✅ 新增：记录已加载的 stdlib 文件，避免重复加载
	loadedStdlibFiles map[string]bool
	
	// ✅ 新增：标记是否正在加载 stdlib 文件，防止递归加载
	loadingStdlibFiles map[string]bool
}

// MapIndexAssignmentTarget 标记map索引赋值目标
// 用于在GenerateAssignmentStatement中识别map索引赋值
type MapIndexAssignmentTarget struct {
	MapExpr entities.Expr // map表达式
	KeyExpr entities.Expr // key表达式
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
		loadedStdlibFiles:    make(map[string]bool), // 初始化已加载文件集合
		loadingStdlibFiles:   make(map[string]bool), // 初始化正在加载文件集合
	}
}

// getLLVMTypeFromMappedType 从映射的类型获取LLVM类型
func (sg *StatementGeneratorImpl) getLLVMTypeFromMappedType(mappedType interface{}) (types.Type, error) {
	switch t := mappedType.(type) {
	case types.Type:
		return t, nil
	case *FutureType, *ChanType:
		// Future和Channel类型在运行时都是i8*指针
		return types.NewPointer(types.I8), nil
	default:
		return nil, fmt.Errorf("unsupported mapped type: %T", t)
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

	// 创建一个包装器函数来处理await调用
	// 这个函数将在协程中执行包含await的代码

	// 外部函数应该在其他地方声明，这里只是设置必要的上下文
	// 实际的协程创建逻辑将在GenerateAsyncFuncDefinition中实现

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

	// ✅ 保存 main 的当前函数/块，加载 stdlib 后会恢复，确保顶层代码落入 main
	mainFunc := irManager.GetCurrentFunction()
	mainBlock := irManager.GetCurrentBasicBlock()

	// ✅ 新增：加载并解析导入的 stdlib 文件
	// 临时方案：加载常用的标准库文件
	// TODO: 实现完整的导入解析和加载机制（从导入语句中提取需要加载的文件）
	// 尝试多个可能的路径
	// 注意：只有在没有正在加载 stdlib 文件时才加载，防止递归调用
	
	// 需要加载的标准库文件列表
	stdlibFiles := []string{
		"stdlib/collections/hashmap.eo",
		"stdlib/net/tcp.eo",  // 添加 net/tcp.eo 以支持 net::bind
	}
	
	for _, stdlibFile := range stdlibFiles {
		possiblePaths := []string{
			stdlibFile,
			"./" + stdlibFile,
			filepath.Join(strings.Split(stdlibFile, "/")...),
		}
		
		loaded := false
		for _, filePath := range possiblePaths {
			// ✅ 修复：规范化路径，避免同一文件被加载多次
			normalizedPath, err := filepath.Abs(filePath)
			if err != nil {
				// 如果无法获取绝对路径，使用原始路径
				normalizedPath = filePath
			}
			normalizedPath = filepath.Clean(normalizedPath)
			
			// 检查是否正在加载或已经加载（使用规范化路径）
			if sg.loadingStdlibFiles[normalizedPath] {
				fmt.Printf("DEBUG: Skipping stdlib file %s (normalized: %s) - already loading (recursive call)\n", filePath, normalizedPath)
				continue
			}
			if sg.loadedStdlibFiles[normalizedPath] {
				fmt.Printf("DEBUG: Skipping stdlib file %s (normalized: %s) - already loaded\n", filePath, normalizedPath)
				loaded = true
				break
			}
			if sg.loadStdlibFileIfExists(irManager, filePath) {
				fmt.Printf("DEBUG: Loaded stdlib file: %s (normalized: %s)\n", filePath, normalizedPath)
				loaded = true
				break
			}
		}
		if !loaded {
			fmt.Printf("DEBUG: Failed to load stdlib file %s from any of the paths: %v\n", stdlibFile, possiblePaths)
		}
	}

	// ✅ 加载 stdlib 后恢复为 main，使后续顶层 VarDecl/If 等落入 main，避免 use of undefined value
	if mainFunc != nil {
		_ = irManager.SetCurrentFunction(mainFunc)
	}
	if mainBlock != nil {
		_ = irManager.SetCurrentBasicBlock(mainBlock)
	}

	// 收集所有结构体定义，用于 sizeof(T) 计算
	structDefs := make(map[string]*entities.StructDef)
	for _, stmt := range statements {
		if structDef, ok := stmt.(*entities.StructDef); ok {
			structDefs[structDef.Name] = structDef
		}
	}
	
	// 设置结构体定义到表达式求值器（如果支持）
	// 注意：ExpressionEvaluatorImpl 在同一个包内，可以直接使用类型断言
	if exprEvalImpl, ok := sg.expressionEvaluator.(*ExpressionEvaluatorImpl); ok {
		exprEvalImpl.SetStructDefinitions(structDefs)
	}

	// 检查是否有await调用
	hasAwait := sg.hasAwaitInStatements(statements)

	if hasAwait {
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
	// 顶层 VarDecl/Print/If 等必须在 main 中生成；处理 FuncDef/MethodDef/AsyncFuncDef 后会切换当前函数，
	// 因此每处理完一个需恢复当前函数/块为 main，避免后续顶层代码误用其他函数的 alloca。
	for i, stmt := range statements {
		_, isFuncDef := stmt.(*entities.FuncDef)
		_, isMethodDef := stmt.(*entities.MethodDef)
		_, isAsyncFuncDef := stmt.(*entities.AsyncFuncDef)
		if isFuncDef || isMethodDef || isAsyncFuncDef {
			savedBlock := irManager.GetCurrentBasicBlock()
			savedFunc := irManager.GetCurrentFunction()
			stmtResult, err := sg.GenerateStatement(irManager, stmt)
			// 恢复为 main，使后续顶层语句继续在 main 中生成
			if savedFunc != nil {
				_ = irManager.SetCurrentFunction(savedFunc)
			}
			if savedBlock != nil {
				_ = irManager.SetCurrentBasicBlock(savedBlock)
			}
			if err != nil {
				fmt.Printf("DEBUG: GenerateProgram - statement %d failed with error: %v\n", i, err)
				result.Success = false
				result.Error = fmt.Errorf("failed to generate statement %d: %w", i, err)
				sg.stats.FailedGenerations++
				break
			}
			if !stmtResult.Success {
				fmt.Printf("DEBUG: GenerateProgram - statement %d returned Success=false, Error: %v\n", i, stmtResult.Error)
			}
			if stmtResult.Success {
				sg.stats.SuccessfulGenerations++
			}
		} else {
			// 确保顶层可执行语句（VarDecl/If/Print 等）始终在 main 中生成：
			// 上一句的求值（如结构体 hash/equals 生成）可能改过当前函数，此处显式恢复 main
			if mainFunc != nil {
				_ = irManager.SetCurrentFunction(mainFunc)
			}
			if mainBlock != nil {
				_ = irManager.SetCurrentBasicBlock(mainBlock)
			}
			stmtResult, err := sg.GenerateStatement(irManager, stmt)
			if err != nil {
				fmt.Printf("DEBUG: GenerateProgram - statement %d failed with error: %v\n", i, err)
				result.Success = false
				result.Error = fmt.Errorf("failed to generate statement %d: %w", i, err)
				sg.stats.FailedGenerations++
				break
			}
			if !stmtResult.Success {
				fmt.Printf("DEBUG: GenerateProgram - statement %d returned Success=false, Error: %v\n", i, stmtResult.Error)
			}
			if stmtResult.Success {
				sg.stats.SuccessfulGenerations++
			}
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
	case *entities.DeleteStmt:
		return sg.GenerateDeleteStatement(irManager, s)
	case *entities.MatchStmt:
		fmt.Printf("DEBUG: GenerateStatement - MatchStmt detected, calling GenerateMatchStatement\n")
		return sg.GenerateMatchStatement(irManager, s)
	case *entities.BlockStmt:
		return sg.GenerateBlockStatement(irManager, s)
	case *entities.IfStmt:
		return sg.GenerateIfStatement(irManager, s)
	case *entities.ForStmt:
		return sg.GenerateForStatement(irManager, s)
	case *entities.WhileStmt:
		return sg.GenerateWhileStatement(irManager, s)
	case *entities.BreakStmt:
		return sg.controlFlowGenerator.GenerateBreakStatement(irManager, s)
	case *entities.ContinueStmt:
		return sg.controlFlowGenerator.GenerateContinueStatement(irManager, s)
	case *entities.TraitDef:
		return sg.GenerateTraitDefinition(irManager, s)
	case *entities.EnumDef:
		return sg.GenerateEnumDefinition(irManager, s)
	case *entities.MethodDef:
		return sg.GenerateMethodDefinition(irManager, s)
	case *entities.SelectStmt:
		return sg.GenerateSelectStatement(irManager, s)
	case *entities.StructDef:
		return sg.GenerateStructDefinition(irManager, s)
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
	printFunc, exists := irManager.GetExternalFunction(printFuncName)
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

	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// GenerateVarDeclaration 生成变量声明
func (sg *StatementGeneratorImpl) GenerateVarDeclaration(irManager generation.IRModuleManager, stmt *entities.VarDecl) (*generation.StatementGenerationResult, error) {
	// 检查变量是否在当前作用域已经存在
	// 如果存在，将其视为赋值语句而不是新的变量声明
	// 这样可以处理 "let i: int = i + 1" 这种情况（在循环中更新循环变量）
	// 注意：只检查当前作用域，不检查外层作用域（允许 shadowing）
	existsInCurrentScope := sg.symbolManager.SymbolExistsInCurrentScope(stmt.Name)
	if existsInCurrentScope {
		// 变量在当前作用域已存在，生成赋值语句
		fmt.Printf("DEBUG: VarDecl %s exists in current scope, treating as assignment\n", stmt.Name)
		return sg.GenerateAssignmentStatement(irManager, &entities.AssignStmt{
			Target: &entities.Identifier{Name: stmt.Name},
			Value:  stmt.Value,
		})
	}
	fmt.Printf("DEBUG: VarDecl %s does not exist in current scope, creating new alloca\n", stmt.Name)

	// 映射变量类型（支持复杂类型如Future[T]）
	// 如果类型为空，尝试从值推断类型
	if stmt.Type == "" {
		// 类型为空，尝试从值推断类型
		var inferredType string
		
		// 调试：打印值的类型
		fmt.Printf("DEBUG: VarDecl %s type inference: value type=%T, value=%v\n", stmt.Name, stmt.Value, stmt.Value)
		
		// 根据值表达式的类型推断变量类型
		switch val := stmt.Value.(type) {
		case *entities.ArrayLiteral:
			if len(val.Elements) == 0 {
				// 空数组字面量，推断为 []i8*（运行时切片指针）
				inferredType = "[]i8*"
			} else {
				// 非空数组，暂时返回错误（需要更复杂的推断逻辑）
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from non-empty array literal", stmt.Name),
				}, nil
			}
		case *entities.LenExpr:
			// len() 表达式总是返回 int
			inferredType = "int"
		case *entities.IntLiteral:
			// 整数字面量推断为 int
			inferredType = "int"
		case *entities.FloatLiteral:
			// 浮点数字面量推断为 float
			inferredType = "float"
		case *entities.StringLiteral:
			// 字符串字面量推断为 string
			inferredType = "string"
		case *entities.BoolLiteral:
			// 布尔字面量推断为 bool
			inferredType = "bool"
		case *entities.Identifier:
			// ✅ 特殊处理：如果标识符名称包含 [ 和 ]，可能是解析器错误地将 IndexExpr 解析为 Identifier
			// 例如：m.buckets[index] 被错误解析为 Identifier{Name: "m.buckets[index]"}
			// 尝试手动解析为 IndexExpr 并推断类型
			if strings.Contains(val.Name, "[") && strings.Contains(val.Name, "]") {
				// 尝试解析为 IndexExpr
				// 格式：array[index] 或 object.field[index]
				bracketIndex := strings.Index(val.Name, "[")
				if bracketIndex > 0 {
					arrayPart := strings.TrimSpace(val.Name[:bracketIndex])
					// indexPart 不需要，因为我们只需要推断数组元素类型
					
					// 解析数组部分（可能是标识符或结构体字段访问）
					var arrayType string
					if strings.Contains(arrayPart, ".") {
						// 结构体字段访问：m.buckets
						parts := strings.Split(arrayPart, ".")
						if len(parts) == 2 {
							// 获取对象类型
							objectSymbol, err := sg.symbolManager.LookupSymbol(parts[0])
							if err != nil {
								return &generation.StatementGenerationResult{
									Success: false,
									Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from index expression: object %s not found: %w", stmt.Name, parts[0], err),
								}, nil
							}
							objectType := objectSymbol.Type
							
							// ✅ 如果当前函数有类型参数映射，替换类型参数
							typeParamMap := irManager.GetCurrentFunctionTypeParams()
							if len(typeParamMap) > 0 && strings.Contains(objectType, "[") {
								hasPointerPrefix := strings.HasPrefix(objectType, "*")
								baseType := objectType
								if hasPointerPrefix {
									baseType = objectType[1:]
								}
								if bracketIdx := strings.Index(baseType, "["); bracketIdx != -1 {
									baseTypeName := baseType[:bracketIdx]
									typeArgsStr := baseType[bracketIdx+1 : len(baseType)-1]
									typeArgs := strings.Split(typeArgsStr, ",")
									substitutedArgs := make([]string, len(typeArgs))
									for i, arg := range typeArgs {
										arg = strings.TrimSpace(arg)
										if substituted, ok := typeParamMap[arg]; ok {
											substitutedArgs[i] = substituted
										} else {
											substitutedArgs[i] = arg
										}
									}
									newBaseType := baseTypeName + "[" + strings.Join(substitutedArgs, ", ") + "]"
									if hasPointerPrefix {
										objectType = "*" + newBaseType
									} else {
										objectType = newBaseType
									}
								}
							}
							
							// 提取基础类型名
							baseTypeName := objectType
							if strings.HasPrefix(baseTypeName, "*") {
								baseTypeName = baseTypeName[1:]
							}
							if bracketIdx := strings.Index(baseTypeName, "["); bracketIdx != -1 {
								baseTypeName = baseTypeName[:bracketIdx]
							}
							
							// 查找结构体定义，获取字段类型
							exprEvalImpl, ok := sg.expressionEvaluator.(*ExpressionEvaluatorImpl)
							if !ok {
								return &generation.StatementGenerationResult{
									Success: false,
									Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from index expression: expression evaluator is not ExpressionEvaluatorImpl", stmt.Name),
								}, nil
							}
							structDef, exists := exprEvalImpl.GetStructDefinition(baseTypeName)
							if !exists {
								return &generation.StatementGenerationResult{
									Success: false,
									Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from index expression: struct definition not found for type %s", stmt.Name, baseTypeName),
								}, nil
							}
							
							// 查找字段定义
							for _, field := range structDef.Fields {
								if field.Name == parts[1] {
									arrayType = field.Type
									break
								}
							}
							if arrayType == "" {
								return &generation.StatementGenerationResult{
									Success: false,
									Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from index expression: field %s not found in struct %s", stmt.Name, parts[1], baseTypeName),
								}, nil
							}
							
							// 替换字段类型中的类型参数（如果是泛型结构体）
							if bracketIdx := strings.Index(objectType, "["); bracketIdx != -1 {
								typeArgsStr := objectType[bracketIdx+1 : len(objectType)-1]
								typeArgs := strings.Split(typeArgsStr, ",")
								for i := range typeArgs {
									typeArgs[i] = strings.TrimSpace(typeArgs[i])
								}
								// 简化处理：假设类型参数名是 K, V（对于 HashMap）
								if baseTypeName == "HashMap" && len(typeArgs) == 2 {
									arrayType = strings.ReplaceAll(arrayType, "K", typeArgs[0])
									arrayType = strings.ReplaceAll(arrayType, "V", typeArgs[1])
								}
							}
						} else {
							// 无法解析，回退到原始处理
							symbol, err := sg.symbolManager.LookupSymbol(val.Name)
							if err != nil {
								return &generation.StatementGenerationResult{
									Success: false,
									Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from identifier %s: %w", stmt.Name, val.Name, err),
								}, nil
							}
							inferredType = symbol.Type
							break
						}
					} else {
						// 简单标识符：buckets
						symbol, err := sg.symbolManager.LookupSymbol(arrayPart)
						if err != nil {
							return &generation.StatementGenerationResult{
								Success: false,
								Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from index expression: array identifier %s not found: %w", stmt.Name, arrayPart, err),
							}, nil
						}
						arrayType = symbol.Type
					}
					
					// 从数组类型提取元素类型
					// 数组类型格式：[]T 或 [T]
					// 注意：[]Bucket[K, V] 需要正确提取 Bucket[K, V]
					fmt.Printf("DEBUG: IndexExpr type inference for %s: arrayType=%s\n", stmt.Name, arrayType)
					if strings.HasPrefix(arrayType, "[") {
						// 对于 []T 格式，需要找到数组的结束位置
						// 策略：从第二个字符开始查找，找到第一个不匹配的 ]
						// 例如：[]Bucket[K, V] -> 找到位置1的 ]（数组结束），而不是位置 len-1 的 ]（泛型结束）
						firstBracketIndex := strings.Index(arrayType, "[")
						if firstBracketIndex >= 0 {
							// 检查是否是 []T 格式（两个连续的 [）
							if len(arrayType) > firstBracketIndex+1 && arrayType[firstBracketIndex+1] == ']' {
								// []T 格式：数组结束在位置 firstBracketIndex+1
								arrayEndIndex := firstBracketIndex + 1
								// 元素类型从 arrayEndIndex+1 开始到字符串结束
								if len(arrayType) > arrayEndIndex+1 {
									elementType := arrayType[arrayEndIndex+1:]
									fmt.Printf("DEBUG: IndexExpr type inference for %s: []T format, extracted elementType=%s\n", stmt.Name, elementType)
									inferredType = elementType
								} else {
									return &generation.StatementGenerationResult{
										Success: false,
										Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from index expression: empty array type %s", stmt.Name, arrayType),
									}, nil
								}
							} else {
								// [T] 格式：找到匹配的 ]
								// 需要找到与第一个 [ 匹配的 ]
								bracketCount := 0
								arrayEndIndex := -1
								for i := firstBracketIndex; i < len(arrayType); i++ {
									if arrayType[i] == '[' {
										bracketCount++
									} else if arrayType[i] == ']' {
										bracketCount--
										if bracketCount == 0 {
											arrayEndIndex = i
											break
										}
									}
								}
								if arrayEndIndex > firstBracketIndex {
									elementType := arrayType[firstBracketIndex+1 : arrayEndIndex]
									fmt.Printf("DEBUG: IndexExpr type inference for %s: [T] format, extracted elementType=%s\n", stmt.Name, elementType)
									inferredType = elementType
								} else {
									return &generation.StatementGenerationResult{
										Success: false,
										Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from index expression: unmatched brackets in array type %s", stmt.Name, arrayType),
									}, nil
								}
							}
						} else {
							return &generation.StatementGenerationResult{
								Success: false,
								Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from index expression: array type %s does not start with [", stmt.Name, arrayType),
							}, nil
						}
					} else {
						return &generation.StatementGenerationResult{
							Success: false,
							Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from index expression: array type %s is not a valid array/slice type", stmt.Name, arrayType),
						}, nil
					}
					break // 成功推断，跳出 switch
				}
			}
			
			// 从符号表查找标识符的类型
			symbol, err := sg.symbolManager.LookupSymbol(val.Name)
			if err != nil {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from identifier %s: %w", stmt.Name, val.Name, err),
				}, nil
			}
			inferredType = symbol.Type
		case *entities.IndexExpr:
			// 索引访问表达式（如 m.buckets[i] 或 map[key]）：推断为数组元素类型或Map值类型
			// 1. 获取数组/Map表达式的类型
			var arrayType string
			if ident, ok := val.Array.(*entities.Identifier); ok {
				// 数组/Map是标识符（如 buckets 或 idToName）
				symbol, err := sg.symbolManager.LookupSymbol(ident.Name)
				if err != nil {
					return &generation.StatementGenerationResult{
						Success: false,
						Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from index expression: array/map identifier %s not found: %w", stmt.Name, ident.Name, err),
					}, nil
				}
				arrayType = symbol.Type
				
				// ✅ 特殊处理：如果是Map类型，从Map类型推断值类型
				if strings.HasPrefix(arrayType, "map[") {
					// Map类型：从map[K]V中提取值类型V
					bracketEnd := strings.Index(arrayType, "]")
					if bracketEnd != -1 && bracketEnd+1 < len(arrayType) {
						valueType := strings.TrimSpace(arrayType[bracketEnd+1:])
						if valueType != "" {
							inferredType = valueType
							break // 成功推断，跳出 switch
						}
					}
					return &generation.StatementGenerationResult{
						Success: false,
						Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from map index expression: invalid map type syntax %s", stmt.Name, arrayType),
					}, nil
				}
			} else if structAccess, ok := val.Array.(*entities.StructAccess); ok {
				// 数组是结构体字段访问（如 m.buckets）
				// 获取对象类型
				if ident, ok := structAccess.Object.(*entities.Identifier); ok {
					symbol, err := sg.symbolManager.LookupSymbol(ident.Name)
					if err != nil {
						return &generation.StatementGenerationResult{
							Success: false,
							Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from index expression: struct object %s not found: %w", stmt.Name, ident.Name, err),
						}, nil
					}
					objectType := symbol.Type
					
					// ✅ 新增：如果当前函数有类型参数映射，替换类型参数
					typeParamMap := irManager.GetCurrentFunctionTypeParams()
					if len(typeParamMap) > 0 && strings.Contains(objectType, "[") {
						hasPointerPrefix := strings.HasPrefix(objectType, "*")
						baseType := objectType
						if hasPointerPrefix {
							baseType = objectType[1:]
						}
						if bracketIndex := strings.Index(baseType, "["); bracketIndex != -1 {
							baseTypeName := baseType[:bracketIndex]
							typeArgsStr := baseType[bracketIndex+1 : len(baseType)-1]
							typeArgs := strings.Split(typeArgsStr, ",")
							substitutedArgs := make([]string, len(typeArgs))
							for i, arg := range typeArgs {
								arg = strings.TrimSpace(arg)
								if substituted, ok := typeParamMap[arg]; ok {
									substitutedArgs[i] = substituted
								} else {
									substitutedArgs[i] = arg
								}
							}
							newBaseType := baseTypeName + "[" + strings.Join(substitutedArgs, ", ") + "]"
							if hasPointerPrefix {
								objectType = "*" + newBaseType
							} else {
								objectType = newBaseType
							}
						}
					}
					
					// 提取基础类型名
					baseTypeName := objectType
					if strings.HasPrefix(baseTypeName, "*") {
						baseTypeName = baseTypeName[1:]
					}
					if bracketIndex := strings.Index(baseTypeName, "["); bracketIndex != -1 {
						baseTypeName = baseTypeName[:bracketIndex]
					}
					
					// 查找结构体定义，获取字段类型
					// ✅ 使用 GetStructDefinition 方法获取结构体定义
					// 通过类型断言访问 ExpressionEvaluatorImpl 的方法
					exprEvalImpl, ok := sg.expressionEvaluator.(*ExpressionEvaluatorImpl)
					if !ok {
						return &generation.StatementGenerationResult{
							Success: false,
							Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from index expression: expression evaluator is not ExpressionEvaluatorImpl", stmt.Name),
						}, nil
					}
					structDef, exists := exprEvalImpl.GetStructDefinition(baseTypeName)
					if !exists {
						return &generation.StatementGenerationResult{
							Success: false,
							Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from index expression: struct definition not found for type %s", stmt.Name, baseTypeName),
						}, nil
					}
					
					// 查找字段定义
					for _, field := range structDef.Fields {
						if field.Name == structAccess.Field {
							arrayType = field.Type
							break
						}
					}
					if arrayType == "" {
						return &generation.StatementGenerationResult{
							Success: false,
							Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from index expression: field %s not found in struct %s", stmt.Name, structAccess.Field, baseTypeName),
						}, nil
					}
					
					// 替换字段类型中的类型参数（如果是泛型结构体）
					if bracketIndex := strings.Index(objectType, "["); bracketIndex != -1 {
						typeArgsStr := objectType[bracketIndex+1 : len(objectType)-1]
						typeArgs := strings.Split(typeArgsStr, ",")
						for i := range typeArgs {
							typeArgs[i] = strings.TrimSpace(typeArgs[i])
						}
						// 提取类型参数名（从结构体定义中获取）
						// 简化处理：假设类型参数名是 K, V（对于 HashMap）
						if baseTypeName == "HashMap" && len(typeArgs) == 2 {
							arrayType = strings.ReplaceAll(arrayType, "K", typeArgs[0])
							arrayType = strings.ReplaceAll(arrayType, "V", typeArgs[1])
						}
					}
				} else {
					return &generation.StatementGenerationResult{
						Success: false,
						Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from index expression: struct access object must be identifier, got %T", stmt.Name, structAccess.Object),
					}, nil
				}
			} else {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from index expression: array expression type %T not yet supported", stmt.Name, val.Array),
				}, nil
			}
			
			// 2. 从数组类型提取元素类型
			// 数组类型格式：[]T 或 [T]
			if strings.HasPrefix(arrayType, "[") && strings.HasSuffix(arrayType, "]") {
				// 提取元素类型（移除 [ 和 ]）
				elementType := arrayType[1 : len(arrayType)-1]
				// 如果数组类型是 []T，元素类型是 T
				// 如果数组类型是 [T]，元素类型也是 T
				inferredType = elementType
			} else {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from index expression: array type %s is not a valid array/slice type", stmt.Name, arrayType),
				}, nil
			}
		case *entities.BinaryExpr:
			// 二元表达式：根据操作符推断类型
			// 简化处理：算术和比较操作返回 int，逻辑操作返回 bool
			switch val.Op {
			case "+", "-", "*", "/", "%":
				inferredType = "int" // 简化：假设都是整数运算
			case "==", "!=", "<", ">", "<=", ">=":
				inferredType = "bool"
			case "&&", "||":
				inferredType = "bool"
			default:
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from binary expression with operator %s", stmt.Name, val.Op),
				}, nil
			}
		case *entities.TernaryExpr:
			// 三元表达式（if ... else ...）：推断类型为 then 和 else 分支的公共类型
			// 简化处理：如果 then 和 else 都是相同类型，使用该类型
			// 否则，尝试从 then 分支推断（通常两个分支类型相同）
			// 这里简化处理：假设 then 和 else 分支类型相同，从 then 分支推断
			if val.ThenValue != nil {
				// 递归推断 then 分支的类型
				switch thenVal := val.ThenValue.(type) {
				case *entities.IntLiteral:
					inferredType = "int"
				case *entities.FloatLiteral:
					inferredType = "float"
				case *entities.StringLiteral:
					inferredType = "string"
				case *entities.BoolLiteral:
					inferredType = "bool"
				case *entities.LenExpr:
					inferredType = "int"
				case *entities.BinaryExpr:
					// 从二元表达式推断（简化：假设是算术运算返回 int）
					if thenVal.Op == "+" || thenVal.Op == "-" || thenVal.Op == "*" || thenVal.Op == "/" || thenVal.Op == "%" {
						inferredType = "int"
					} else {
						inferredType = "bool"
					}
				case *entities.Identifier:
					// 从符号表查找标识符的类型
					symbol, err := sg.symbolManager.LookupSymbol(thenVal.Name)
					if err != nil {
						return &generation.StatementGenerationResult{
							Success: false,
							Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from ternary expression: identifier %s not found in symbol table", stmt.Name, thenVal.Name),
						}, nil
					}
					inferredType = symbol.Type
				default:
					// 其他情况，暂时返回错误
					return &generation.StatementGenerationResult{
						Success: false,
						Error:   fmt.Errorf("variable type is empty for %s, cannot infer type from ternary expression then branch type %T", stmt.Name, val.ThenValue),
					}, nil
				}
			} else {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("variable type is empty for %s, ternary expression then branch is nil", stmt.Name),
				}, nil
			}
		case *entities.MethodCallExpr:
			// 方法调用：根据方法名推断返回类型
			// 特殊处理：len() 方法总是返回 int
			if val.MethodName == "len" {
				// len() 方法调用总是返回 int
				inferredType = "int"
			} else {
				// 其他方法调用：暂时返回错误（需要查询方法返回类型）
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("variable type is empty for %s, type inference from method call %s not yet implemented", stmt.Name, val.MethodName),
				}, nil
			}
		case *entities.FuncCall:
			// 函数调用：根据函数名推断返回类型
			// 特殊处理：已知的运行时函数返回类型
			switch val.Name {
			case "bucket_index":
				// bucket_index 函数返回 int（索引）
				inferredType = "int"
			case "runtime_slice_len":
				// runtime_slice_len 函数返回 int（长度）
				inferredType = "int"
			default:
				// 其他函数调用：暂时返回错误（需要查询函数返回类型）
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("variable type is empty for %s, type inference from function call %s not yet implemented", stmt.Name, val.Name),
				}, nil
			}
		default:
			// 其他情况，暂时返回错误
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("variable type is empty for %s, and type inference from value type %T not yet implemented", stmt.Name, stmt.Value),
			}, nil
		}
		
		// 使用推断的类型继续处理
		stmt.Type = inferredType
	}
	
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

	// 顶层变量（在 main 中声明）使用全局变量，以便其他函数可访问；否则使用 alloca
	var ptr interface{}
	if irManager.GetCurrentFunctionName() == "main" {
		var err error
		ptr, err = irManager.CreateGlobalVariable(llvmType, stmt.Name)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to create global variable: %w", err),
			}, nil
		}
	} else {
		var err error
		ptr, err = irManager.CreateAlloca(llvmType, stmt.Name)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to create alloca instruction: %w", err),
			}, nil
		}
	}
	alloca := ptr

	// 注册符号
	if err := sg.symbolManager.RegisterSymbol(stmt.Name, stmt.Type, alloca); err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to register symbol %s: %w", stmt.Name, err),
		}, nil
	}

	// 如果有初始值，生成赋值
	if stmt.Value != nil {
		assignResult, err := sg.GenerateAssignmentStatement(irManager, &entities.AssignStmt{
			Target: &entities.Identifier{Name: stmt.Name},
			Value:  stmt.Value,
		})
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to generate initial assignment: %w", err),
			}, nil
		}
		_ = assignResult // 赋值指令已添加到IR中

		// 如果是数组字面量，更新符号表中的长度信息
		// 注意：不要覆盖 alloca 指令，只更新长度信息（如果 SymbolInfo 支持的话）
		// 当前实现中，SymbolInfo 只有一个 Value 字段，所以对于数组长度信息，
		// 我们暂时不更新符号表，因为这会覆盖 alloca 指令
		// TODO: 未来可以考虑在 SymbolInfo 中添加 ArrayLength 字段来存储数组长度
		// if arrayLit, ok := stmt.Value.(*entities.ArrayLiteral); ok && strings.HasPrefix(stmt.Type, "[") {
		// 	// 更新符号表，存储数组长度
		// 	if err := sg.symbolManager.UpdateSymbolValue(stmt.Name, len(arrayLit.Elements)); err != nil {
		// 		return &generation.StatementGenerationResult{
		// 			Success: false,
		// 			Error:   fmt.Errorf("failed to update array length in symbol table: %w", err),
		// 		}, nil
		// 	}
		// }
	}

	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// GenerateAssignmentStatement 生成赋值语句
func (sg *StatementGeneratorImpl) GenerateAssignmentStatement(irManager generation.IRModuleManager, stmt *entities.AssignStmt) (*generation.StatementGenerationResult, error) {
	// DEBUG: 打印赋值目标类型
	fmt.Printf("DEBUG: GenerateAssignmentStatement - Target type: %T, Target: %v\n", stmt.Target, stmt.Target)

	// 求值右侧表达式
	valueResult, err := sg.expressionEvaluator.Evaluate(irManager, stmt.Value)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate assignment value: %w", err),
		}, nil
	}

	// 获取赋值目标地址
	targetPtr, err := sg.getAssignmentTargetAddress(irManager, stmt.Target)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to get assignment target address: %w", err),
		}, nil
	}

	// ============ 特殊处理：Map索引赋值 ============
	// 检查是否是map索引赋值（headers[key] = value）
	if mapTarget, ok := targetPtr.(*MapIndexAssignmentTarget); ok {
		return sg.generateMapIndexAssignment(irManager, mapTarget, stmt.Value)
	}

	// ============ 特殊处理：类型转换表达式 ============
	// 处理 v.data = new_ptr as []T 这种Vec内存管理场景
	if typeCast, ok := stmt.Value.(*entities.TypeCastExpr); ok {
		// 检查是否是 *i8 as []T 转换
		if strings.HasPrefix(typeCast.TargetType, "[") {
			// 目标类型是切片类型
			// 检查赋值目标是否是结构体字段访问（如 v.data）
			if structAccess, ok := stmt.Target.(*entities.StructAccess); ok {
				// 检查是否是 Vec[T] 的 data 字段
				var objectType string
				if ident, ok := structAccess.Object.(*entities.Identifier); ok {
					symbol, err := sg.symbolManager.LookupSymbol(ident.Name)
					if err == nil {
						objectType = symbol.Type
						// 检查是否是 Vec[T] 类型
						if strings.HasPrefix(objectType, "Vec[") && structAccess.Field == "data" {
							// 这是 v.data = new_ptr as []T 的场景
							// 需要从 v.len 获取长度信息，并调用 runtime_slice_from_ptr_len
							
							// 1. 获取 v.len 的值（长度信息）
							lenFieldAccess := &entities.StructAccess{
								Object: structAccess.Object,
								Field:  "len",
							}
							lenValue, err := sg.expressionEvaluator.Evaluate(irManager, lenFieldAccess)
							if err != nil {
								return &generation.StatementGenerationResult{
									Success: false,
									Error:   fmt.Errorf("failed to evaluate v.len for type cast: %w", err),
								}, nil
							}
							
							// 3. 调用 runtime_slice_from_ptr_len(new_ptr, v.len)
							funcCall := &entities.FuncCall{
								Name: "runtime_slice_from_ptr_len",
								Args: []entities.Expr{
									typeCast.Expr, // new_ptr
									lenFieldAccess, // v.len
								},
							}
							
							// 4. 求值 runtime_slice_from_ptr_len 调用
							valueResult, err = sg.expressionEvaluator.Evaluate(irManager, funcCall)
							if err != nil {
								return &generation.StatementGenerationResult{
									Success: false,
									Error:   fmt.Errorf("failed to evaluate runtime_slice_from_ptr_len call: %w", err),
								}, nil
							}
							
							// 5. 存储长度信息到符号表（如果 v.len 是常量）
							if intConst, ok := lenValue.(*constant.Int); ok {
								length := int(intConst.X.Int64())
								// 注意：这里不能直接更新 v.data 的长度，因为 v.data 是结构体字段
								// 长度信息已经在 v.len 中，不需要额外存储
								fmt.Printf("DEBUG: Type cast *i8 as []T for Vec[T].data, length=%d from v.len\n", length)
							}
							
							// 继续执行后续的存储指令（使用 runtime_slice_from_ptr_len 的结果）
						}
					}
				}
			}
		}
	}
	
	// 特殊处理：如果右侧是 runtime_slice_from_ptr_len() 调用，需要存储长度信息
	// 检查是否是 runtime_slice_from_ptr_len() 调用
	if funcCall, ok := stmt.Value.(*entities.FuncCall); ok && funcCall.Name == "runtime_slice_from_ptr_len" {
		// 如果目标是 Identifier，检查目标变量是否是切片类型
		if ident, ok := stmt.Target.(*entities.Identifier); ok {
			targetSymbol, err := sg.symbolManager.LookupSymbol(ident.Name)
			if err == nil && strings.HasPrefix(targetSymbol.Type, "[") {
				// 提取长度参数（第二个参数）
				if len(funcCall.Args) >= 2 {
					// 求值长度参数
					lenValue, err := sg.expressionEvaluator.Evaluate(irManager, funcCall.Args[1])
					if err != nil {
						return &generation.StatementGenerationResult{
							Success: false,
							Error:   fmt.Errorf("failed to evaluate length parameter: %w", err),
						}, nil
					}
					
					// 尝试从长度值中提取整数
					// 如果长度值是常量，可以直接提取
					if intConst, ok := lenValue.(*constant.Int); ok {
						length := int(intConst.X.Int64())
						// 更新符号表，存储切片长度
						if err := sg.symbolManager.UpdateSymbolValue(ident.Name, length); err != nil {
							return &generation.StatementGenerationResult{
								Success: false,
								Error:   fmt.Errorf("failed to update slice length in symbol table: %w", err),
							}, nil
						}
						fmt.Printf("DEBUG: Stored slice length %d for variable %s\n", length, ident.Name)
					} else {
						// 如果长度不是常量，需要运行时获取
						// 暂时记录到符号表的元数据中（如果支持）
						fmt.Printf("DEBUG: Warning: slice length for %s is not a compile-time constant, runtime length handling may be needed\n", ident.Name)
					}
				}
			}
		}
	}
	
	// 注意：对于 char** + int32_t → []string 的转换
	// 由于 char** 和 []string 在 LLVM IR 中都是 **i8，内存布局相同
	// 用户需要显式调用 runtime_slice_from_ptr_len() 来构造切片
	// 编译器会自动存储长度信息到符号表（见上面的 runtime_slice_from_ptr_len 处理）

	// 生成存储指令
	err = irManager.CreateStore(valueResult, targetPtr)
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

// getAssignmentTargetAddress 获取赋值目标的地址
// 支持 Identifier、StructAccess、IndexExpr 作为赋值目标
func (sg *StatementGeneratorImpl) getAssignmentTargetAddress(irManager generation.IRModuleManager, target entities.Expr) (interface{}, error) {
	switch t := target.(type) {
	case *entities.Identifier:
		// 简单标识符赋值：x = value
		targetSymbol, err := sg.symbolManager.LookupSymbol(t.Name)
		if err != nil {
			return nil, fmt.Errorf("undefined symbol %s: %w", t.Name, err)
		}
		return targetSymbol.Value, nil
		
	case *entities.StructAccess:
		// 结构体字段赋值：v.len = value
		// 重用 expressionEvaluator 的逻辑来获取字段地址
		// 注意：我们需要获取地址，而不是值
		return sg.getStructFieldAddress(irManager, t)
		
	case *entities.IndexExpr:
		// 索引赋值：arr[i] = value 或 headers[key] = value
		// 判断是数组索引还是map索引
		if ident, ok := t.Array.(*entities.Identifier); ok {
			symbol, err := sg.symbolManager.LookupSymbol(ident.Name)
			if err != nil {
				// DEBUG: 打印错误信息，帮助调试
				fmt.Printf("DEBUG: getAssignmentTargetAddress - failed to lookup symbol %s: %v\n", ident.Name, err)
				return nil, fmt.Errorf("undefined symbol %s: %w", ident.Name, err)
			}
			fmt.Printf("DEBUG: getAssignmentTargetAddress - found symbol %s, type: %s\n", ident.Name, symbol.Type)
			
			// 判断类型
			if strings.HasPrefix(symbol.Type, "map[") {
				// Map索引赋值：headers[key] = value
				// 返回特殊标记，在GenerateAssignmentStatement中处理
				return &MapIndexAssignmentTarget{
					MapExpr: t.Array,
					KeyExpr: t.Index,
				}, nil
			} else if strings.HasPrefix(symbol.Type, "[") {
				// 数组索引赋值：arr[i] = value
				return sg.getArrayIndexAddress(irManager, t, symbol.Type)
			} else {
				return nil, fmt.Errorf("unsupported index expression type: %s", symbol.Type)
			}
		} else {
			// 对于非标识符的数组表达式，默认当作数组处理
			return sg.getArrayIndexAddress(irManager, t, "")
		}
		
	default:
		return nil, fmt.Errorf("unsupported assignment target type: %T", target)
	}
}

// getArrayIndexAddress 获取数组索引的地址（用于赋值）
// 返回元素地址，供CreateStore使用
func (sg *StatementGeneratorImpl) getArrayIndexAddress(
	irManager generation.IRModuleManager,
	expr *entities.IndexExpr,
	arrayType string,
) (interface{}, error) {
	// 1. 求值数组表达式
	arrayValue, err := sg.expressionEvaluator.Evaluate(irManager, expr.Array)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate array expression: %w", err)
	}

	// 2. 求值索引表达式
	indexValue, err := sg.expressionEvaluator.Evaluate(irManager, expr.Index)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate index expression: %w", err)
	}

	// 3. 确定元素类型
	var elementType types.Type
	if arrayType != "" {
		// 从数组类型推断元素类型
		// 重用 expressionEvaluator 的逻辑
		if exprEvalImpl, ok := sg.expressionEvaluator.(*ExpressionEvaluatorImpl); ok {
			elementType, err = exprEvalImpl.inferElementTypeFromArrayType(arrayType)
			if err != nil {
				return nil, fmt.Errorf("failed to infer element type: %w", err)
			}
		} else {
			// 如果无法访问内部方法，使用默认类型
			elementType = types.I32
		}
	} else {
		elementType = types.I32 // 默认类型
	}

	// 4. 生成GetElementPtr指令获取元素地址（不加载值）
	elementPtr, err := irManager.CreateGetElementPtr(elementType, arrayValue, indexValue)
	if err != nil {
		return nil, fmt.Errorf("failed to create GEP for array index: %w", err)
	}

	return elementPtr, nil
}

// getStructFieldAddress 获取结构体字段的地址（用于赋值）
// 类似于 EvaluateStructAccess，但返回地址而不是值
func (sg *StatementGeneratorImpl) getStructFieldAddress(irManager generation.IRModuleManager, expr *entities.StructAccess) (interface{}, error) {
	// 重用 expressionEvaluator 的逻辑
	// 通过类型断言访问 ExpressionEvaluatorImpl 的内部字段
	exprEvalImpl, ok := sg.expressionEvaluator.(*ExpressionEvaluatorImpl)
	if !ok {
		return nil, fmt.Errorf("expressionEvaluator is not ExpressionEvaluatorImpl, cannot access internal fields")
	}
	
	// 1. 求值对象表达式（可能是标识符、函数调用等）
	objectValue, err := exprEvalImpl.Evaluate(irManager, expr.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate struct object: %w", err)
	}

	// 2. 获取对象类型（从符号表、函数调用或类型推断）
	var objectType string
	var isPointer bool

	// 尝试从标识符获取类型
	if ident, ok := expr.Object.(*entities.Identifier); ok {
		symbol, err := exprEvalImpl.symbolManager.LookupSymbol(ident.Name)
		if err == nil {
			objectType = symbol.Type
			// 检查是否是指针类型
			if strings.HasPrefix(objectType, "*") {
				isPointer = true
				objectType = strings.TrimPrefix(objectType, "*")
			} else {
				// 检查LLVM类型是否是指针
				typeInterface, err := exprEvalImpl.typeMapper.MapType(symbol.Type)
				if err == nil {
					if llvmType, ok := typeInterface.(types.Type); ok {
						if _, ok := llvmType.(*types.PointerType); ok {
							isPointer = true
						}
					}
				}
			}
		}
	}

	// 3. 提取基础类型名（移除类型参数）
	// 例如：HashMap[int, string] -> HashMap
	baseTypeName := objectType
	if bracketIndex := strings.Index(objectType, "["); bracketIndex != -1 {
		baseTypeName = objectType[:bracketIndex]
	}
	
	// ✅ 如果当前函数有类型参数映射，需要替换类型参数
	typeParamMap := irManager.GetCurrentFunctionTypeParams()
	if len(typeParamMap) > 0 && strings.Contains(objectType, "[") {
		hasPointerPrefix := strings.HasPrefix(objectType, "*")
		baseType := objectType
		if hasPointerPrefix {
			baseType = objectType[1:]
		}
		if bracketIdx := strings.Index(baseType, "["); bracketIdx != -1 {
			// typeArgsStr := baseType[bracketIdx+1 : len(baseType)-1]
			// typeArgs := strings.Split(typeArgsStr, ",")
			// substitutedArgs := make([]string, len(typeArgs))
			// for i, arg := range typeArgs {
			// 	arg = strings.TrimSpace(arg)
			// 	if substituted, ok := typeParamMap[arg]; ok {
			// 		substitutedArgs[i] = substituted
			// 	} else {
			// 		substitutedArgs[i] = arg
			// 	}
			// }
			// 注意：这里只需要提取基础类型名用于查找结构体定义
			// 类型参数替换已经在 objectType 中完成（通过 typeParamMap）
		}
	}
	
	// 4. 查找结构体定义（使用基础类型名）
	structDef, exists := exprEvalImpl.GetStructDefinition(baseTypeName)
	if !exists {
		return nil, fmt.Errorf("struct definition not found for type %s (base type: %s)", objectType, baseTypeName)
	}

	// 5. 查找字段索引
	fieldIndex := -1
	for i, field := range structDef.Fields {
		if field.Name == expr.Field {
			fieldIndex = i
			break
		}
	}
	if fieldIndex == -1 {
		return nil, fmt.Errorf("field %s not found in struct %s", expr.Field, objectType)
	}

	// 5. 处理指针类型：如果对象是指针，需要先解引用
	var structPtr llvalue.Value
	if isPointer {
		// 对象已经是指针，直接使用
		structPtr, ok = objectValue.(llvalue.Value)
		if !ok {
			return nil, fmt.Errorf("object value is not a valid LLVM value for pointer access")
		}
	} else {
		// 对象是值，需要获取其地址
		// 如果对象是标识符，symbol.Value 已经是 alloca 的结果（指针）
		if ident, ok := expr.Object.(*entities.Identifier); ok {
			symbol, err := exprEvalImpl.symbolManager.LookupSymbol(ident.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to lookup symbol for struct access: %w", err)
			}
			structPtr, ok = symbol.Value.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("symbol value is not a valid LLVM value")
			}
		} else {
			// 对于其他表达式，需要创建一个临时变量存储值，然后获取地址
			// 这里简化处理：假设对象表达式的结果已经是指针
			structPtr, ok = objectValue.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("object value is not a valid LLVM value")
			}
		}
	}

	// 6. 构建结构体LLVM类型（用于GEP）
	structLLVMType, err := exprEvalImpl.buildStructLLVMType(structDef)
	if err != nil {
		return nil, fmt.Errorf("failed to build struct LLVM type: %w", err)
	}

	// 7. 使用 GetElementPtr 获取字段指针（返回地址，不加载值）
	fieldIndexConst := constant.NewInt(types.I32, int64(fieldIndex))
	fieldPtr, err := irManager.CreateGetElementPtr(structLLVMType, structPtr, fieldIndexConst)
	if err != nil {
		return nil, fmt.Errorf("failed to create GEP for field access: %w", err)
	}

	return fieldPtr, nil
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
	// ✅ 添加调试输出：检查函数名和类型参数
	if strings.Contains(stmt.Name, "find_entry") {
		fmt.Printf("DEBUG: GenerateFuncDefinition - START: funcName=%s, TypeParams count=%d\n", stmt.Name, len(stmt.TypeParams))
		if len(stmt.TypeParams) > 0 {
			typeParamNames := make([]string, len(stmt.TypeParams))
			for i, tp := range stmt.TypeParams {
				typeParamNames[i] = tp.Name
			}
			fmt.Printf("DEBUG: GenerateFuncDefinition - funcName=%s, typeParams=%v\n", stmt.Name, typeParamNames)
		}
	}
	
	// ✅ 修复：如果是泛型函数，先检查是否已经保存了泛型函数定义
	// 如果已经保存，说明这是延迟编译的调用，应该继续生成函数体
	// 如果还没有保存，说明这是第一次调用，应该只保存定义并返回
	if len(stmt.TypeParams) > 0 {
		// 检查是否已经保存了泛型函数定义
		_, exists := irManager.GetGenericFunctionDefinition(stmt.Name)
		if !exists {
			// 第一次调用：只保存泛型函数定义，不生成函数体
			typeParamNames := make([]string, len(stmt.TypeParams))
			for i, tp := range stmt.TypeParams {
				typeParamNames[i] = tp.Name
			}
			irManager.SetFunctionTypeParamNames(stmt.Name, typeParamNames)
			fmt.Printf("DEBUG: GenerateFuncDefinition - funcName=%s, typeParams=%v, saving generic function definition (first call)\n", stmt.Name, typeParamNames)
			irManager.SaveGenericFunctionDefinition(stmt.Name, stmt)
			// 创建函数声明（不生成函数体）
			returnTypeStr := stmt.ReturnType
			if stmt.Name == "main" && stmt.ReturnType == "void" {
				returnTypeStr = "int"
			}
			returnTypeMapped, err := sg.typeMapper.MapType(returnTypeStr)
			if err != nil {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("failed to map return type %s: %w", returnTypeStr, err),
				}, nil
			}
			returnType, err := sg.getLLVMTypeFromMappedType(returnTypeMapped)
			if err != nil {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("failed to get LLVM return type: %w", err),
				}, nil
			}
			paramTypes := make([]interface{}, len(stmt.Params))
			for i, param := range stmt.Params {
				paramTypeMapped, err := sg.typeMapper.MapType(param.Type)
				if err != nil {
					return &generation.StatementGenerationResult{
						Success: false,
						Error:   fmt.Errorf("failed to map parameter type %s: %w", param.Type, err),
					}, nil
				}
				paramType, err := sg.getLLVMTypeFromMappedType(paramTypeMapped)
				if err != nil {
					return &generation.StatementGenerationResult{
						Success: false,
						Error:   fmt.Errorf("failed to get LLVM parameter type: %w", err),
					}, nil
				}
				paramTypes[i] = paramType
			}
			// 创建函数声明（不生成函数体）
			_, err = irManager.CreateFunction(stmt.Name, returnType, paramTypes)
			if err != nil {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("failed to create function declaration: %w", err),
				}, nil
			}
			return &generation.StatementGenerationResult{
				Success: true,
			}, nil
		}
		// 已经保存了泛型函数定义，这是延迟编译的调用，继续生成函数体
		fmt.Printf("DEBUG: GenerateFuncDefinition - funcName=%s, generic function definition already exists, compiling monomorphized function body\n", stmt.Name)
	}
	
	// 特殊处理：main函数必须返回int而不是void
	returnTypeStr := stmt.ReturnType
	if stmt.Name == "main" && stmt.ReturnType == "void" {
		returnTypeStr = "int"
	}

	// 映射返回类型
	returnTypeMapped, err := sg.typeMapper.MapType(returnTypeStr)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to map return type %s: %w", returnTypeStr, err),
		}, nil
	}
	returnType, err := sg.getLLVMTypeFromMappedType(returnTypeMapped)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to get LLVM return type: %w", err),
		}, nil
	}

	// 映射参数类型
	paramTypes := make([]interface{}, len(stmt.Params))
	for i, param := range stmt.Params {
		paramTypeMapped, err := sg.typeMapper.MapType(param.Type)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to map parameter type %s: %w", param.Type, err),
			}, nil
		}
		paramType, err := sg.getLLVMTypeFromMappedType(paramTypeMapped)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to get LLVM parameter type: %w", err),
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
	// ✅ 修复：在SetCurrentFunction之前保存类型参数映射（如果已设置）
	// 因为SetCurrentFunction会重置类型参数映射，我们需要在之后恢复
	savedTypeParamsBeforeSetFunc := irManager.GetCurrentFunctionTypeParams()
	
	err = irManager.SetCurrentFunction(fn)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set current function: %w", err),
		}, nil
	}
	
	// ✅ 修复：如果类型参数映射在SetCurrentFunction之前已设置，恢复它
	// 这发生在CompileMonomorphizedFunction调用GenerateFuncDefinition时
	if savedTypeParamsBeforeSetFunc != nil && len(savedTypeParamsBeforeSetFunc) > 0 {
		irManager.SetCurrentFunctionTypeParams(savedTypeParamsBeforeSetFunc)
		fmt.Printf("DEBUG: GenerateFuncDefinition - funcName=%s, restored typeParamMap=%v after SetCurrentFunction\n", stmt.Name, savedTypeParamsBeforeSetFunc)
	}
	
	// 保存函数定义的类型参数列表（用于从参数类型推断类型参数）
	// 添加调试输出
	if strings.Contains(stmt.Name, "find_entry") {
		fmt.Printf("DEBUG: GenerateFuncDefinition - funcName=%s, TypeParams count=%d\n", stmt.Name, len(stmt.TypeParams))
		if len(stmt.TypeParams) > 0 {
			typeParamNames := make([]string, len(stmt.TypeParams))
			for i, tp := range stmt.TypeParams {
				typeParamNames[i] = tp.Name
			}
			fmt.Printf("DEBUG: GenerateFuncDefinition - funcName=%s, typeParams=%v\n", stmt.Name, typeParamNames)
		} else {
			fmt.Printf("DEBUG: GenerateFuncDefinition - ERROR: funcName=%s has NO typeParams, but should be generic!\n", stmt.Name)
		}
	}
	if len(stmt.TypeParams) > 0 {
		typeParamNames := make([]string, len(stmt.TypeParams))
		for i, tp := range stmt.TypeParams {
			typeParamNames[i] = tp.Name
		}
		irManager.SetFunctionTypeParamNames(stmt.Name, typeParamNames)
		fmt.Printf("DEBUG: GenerateFuncDefinition - funcName=%s, typeParams=%v\n", stmt.Name, typeParamNames)
		
		// ✅ 新增：保存泛型函数定义，供延迟编译使用
		irManager.SaveGenericFunctionDefinition(stmt.Name, stmt)
		
		// ✅ 修改：如果是泛型函数，只生成函数声明，不生成函数体
		// 函数体将在函数调用时延迟编译（当类型参数已知时）
		// 注意：函数声明已经通过CreateFunction创建，这里只需要返回成功即可
		return &generation.StatementGenerationResult{
			Success: true,
		}, nil
	}
	
	// 非泛型函数：正常生成函数体
	// ✅ 修复：检查函数是否已经有 body（已经编译过），避免重复处理
	if len(fn.Blocks) > 0 {
		// 函数已经有 body，跳过函数体生成
		fmt.Printf("DEBUG: GenerateFuncDefinition - function %s already has body (%d blocks), skipping\n", stmt.Name, len(fn.Blocks))
		return &generation.StatementGenerationResult{
			Success: true,
		}, nil
	}

	// 为函数参数创建独立作用域，避免参数名（如 name）泄漏到全局，
	// 导致后续生成 main/顶层代码时误用其他函数的 alloca（use of undefined value）。
	// 流程：EnterScope(函数作用域) -> 注册参数 -> EnterScope(函数体) -> 生成体 -> ExitScope x2
	if err := sg.symbolManager.EnterScope(); err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to enter function scope for parameters: %w", err),
		}, nil
	}

	// 创建入口基本块
	entryBlock, err := irManager.CreateBasicBlock("entry")
	if err != nil {
		_ = sg.symbolManager.ExitScope() // 弹出刚进入的函数作用域
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create entry block: %w", err),
		}, nil
	}

	// 设置当前基本块
	err = irManager.SetCurrentBasicBlock(entryBlock)
	if err != nil {
		_ = sg.symbolManager.ExitScope()
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
			_ = sg.symbolManager.ExitScope()
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to create alloca for parameter %s: %w", param.Name, err),
			}, nil
		}

		// store参数值到alloca的位置
		err = irManager.CreateStore(fn.Params[i], allocaInst)
		if err != nil {
			_ = sg.symbolManager.ExitScope()
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to create store for parameter %s: %w", param.Name, err),
			}, nil
		}

		// 在符号表中注册参数
		// 注意：param.Type 应该是已经被替换的类型（对于单态化函数）
		fmt.Printf("DEBUG: GenerateFuncDefinition - registering parameter %s with type %s\n", param.Name, param.Type)
		
		// ✅ 如果符号已存在（延迟编译时可能已注册），先尝试更新
		if sg.symbolManager.SymbolExistsInCurrentScope(param.Name) {
			// 符号已存在，更新其值（用于延迟编译场景）
			if err := sg.symbolManager.UpdateSymbolValue(param.Name, allocaInst); err != nil {
				// 如果更新失败，尝试重新注册（可能会失败，但至少尝试了）
				fmt.Printf("DEBUG: GenerateFuncDefinition - parameter %s exists, updating value\n", param.Name)
			}
		} else {
			// 符号不存在，正常注册
			err = sg.symbolManager.RegisterSymbol(param.Name, param.Type, allocaInst)
			if err != nil {
				_ = sg.symbolManager.ExitScope()
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("failed to register parameter %s: %w", param.Name, err),
				}, nil
			}
		}
	}

	// 为函数体创建新的作用域
	if err := sg.symbolManager.EnterScope(); err != nil {
		_ = sg.symbolManager.ExitScope()
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to enter function body scope: %w", err),
		}, nil
	}

	// 生成函数体中的语句
	if stmt.Body != nil {
		for _, bodyStmt := range stmt.Body {
			result, err := sg.GenerateStatement(irManager, bodyStmt)
			if err != nil {
				_, _ = sg.symbolManager.ExitScope(), sg.symbolManager.ExitScope() // body + 函数参数作用域
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("failed to generate statement in function %s: %w", stmt.Name, err),
				}, nil
			}
			if !result.Success {
				_, _ = sg.symbolManager.ExitScope(), sg.symbolManager.ExitScope()
				return result, nil
			}
		}
	}

	// 退出函数体作用域
	if err := sg.symbolManager.ExitScope(); err != nil {
		_ = sg.symbolManager.ExitScope() // 仍须退出函数参数作用域
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to exit function body scope: %w", err),
		}, nil
	}

	// 退出函数参数作用域，避免参数名泄漏到全局（导致 use of undefined value）
	if err := sg.symbolManager.ExitScope(); err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to exit function scope: %w", err),
		}, nil
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
	// 确保在正确的 block 中求值
	currentBlock := irManager.GetCurrentBasicBlock()
	returnValue, err := sg.expressionEvaluator.Evaluate(irManager, stmt.Value)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate return value: %w", err),
		}, nil
	}
	// 验证 current block 没有被改变
	afterBlock := irManager.GetCurrentBasicBlock()
	if currentBlock != afterBlock {
		// 恢复 current block
		irManager.SetCurrentBasicBlock(currentBlock)
	}

	// 检查是否需要加载结构体值
	// 如果返回值是指针，但函数返回类型是结构体值类型，则需要加载值
	returnValueLLVM, ok := returnValue.(llvalue.Value)
	if !ok {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("return value is not a valid LLVM value: %T", returnValue),
		}, nil
	}
	
	// 获取当前函数的返回类型
	currentFuncInterface := irManager.GetCurrentFunction()
	if currentFuncInterface != nil {
		if currentFunc, ok := currentFuncInterface.(*ir.Func); ok {
			funcReturnType := currentFunc.Sig.RetType
			
			// 检查返回值类型和函数返回类型
			returnValueType := returnValueLLVM.Type()
			
			// ✅ 修复：如果返回值类型和函数返回类型不匹配，进行类型转换
			if !returnValueType.Equal(funcReturnType) {
				// 使用 IRModuleManager 的 convertValueType 方法进行类型转换
				// 注意：convertValueType 是内部方法，需要通过 CreateStore 间接使用
				// 或者我们可以直接在这里实现转换逻辑
				convertedValue, err := sg.convertReturnValueType(irManager, returnValueLLVM, returnValueType, funcReturnType)
				if err != nil {
					return &generation.StatementGenerationResult{
						Success: false,
						Error:   fmt.Errorf("failed to convert return value type from %s to %s: %w", returnValueType, funcReturnType, err),
					}, nil
				}
				returnValueLLVM = convertedValue
			}
			
			// 如果返回值是指针类型，但函数返回类型是结构体值类型，则需要加载值
			if ptrType, ok := returnValueType.(*types.PointerType); ok {
				// 返回值是指针
				if structType, ok := funcReturnType.(*types.StructType); ok {
					// 函数返回类型是结构体值类型
					// 检查指针指向的类型是否与函数返回类型匹配（使用 Equal 方法比较）
					if ptrType.ElemType.Equal(structType) {
						// 需要加载结构体值
						structValue, err := irManager.CreateLoad(structType, returnValueLLVM, "struct_value")
						if err != nil {
							return &generation.StatementGenerationResult{
								Success: false,
								Error:   fmt.Errorf("failed to load struct value for return: %w", err),
							}, nil
						}
						returnValueLLVM = structValue.(llvalue.Value)
					}
				}
			}
		}
	}

	// 生成ret指令
	err = irManager.CreateRet(returnValueLLVM)
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

// GenerateDeleteStatement 生成delete语句
// 语法：delete(map, key)
// 生成代码：根据Map类型调用相应的运行时删除函数
func (sg *StatementGeneratorImpl) GenerateDeleteStatement(irManager generation.IRModuleManager, stmt *entities.DeleteStmt) (*generation.StatementGenerationResult, error) {
	// 1. 获取Map类型（从map表达式中推断）
	mapType := sg.getMapTypeFromExpr(stmt.Map)
	if mapType == "" {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("cannot determine map type from expression in delete statement"),
		}, nil
	}

	// 2. 解析Map类型（提取键类型和值类型）
	keyType, valueType, err := sg.parseMapType(mapType)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to parse map type %s: %w", mapType, err),
		}, nil
	}

	// 3. 根据键类型和值类型选择运行时函数
	runtimeFuncName, err := sg.selectMapRuntimeFunction(keyType, valueType, "delete")
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to select runtime function for map[%s]%s: %w", keyType, valueType, err),
		}, nil
	}

	// 4. 求值map表达式（获取map指针）
	mapValue, err := sg.expressionEvaluator.Evaluate(irManager, stmt.Map)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate map expression: %w", err),
		}, nil
	}

	mapValueLLVM, ok := mapValue.(llvalue.Value)
	if !ok {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("map value is not a valid LLVM value: %T", mapValue),
		}, nil
	}

	// 5. 求值key表达式（根据键类型处理）
	keyValue, err := sg.expressionEvaluator.Evaluate(irManager, stmt.Key)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate key expression: %w", err),
		}, nil
	}

	keyValueLLVM, ok := keyValue.(llvalue.Value)
	if !ok {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("key value is not a valid LLVM value: %T", keyValue),
		}, nil
	}

	// 6. 获取对应的运行时函数
	mapDeleteFunc, exists := irManager.GetExternalFunction(runtimeFuncName)
	if !exists {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("runtime function %s not declared", runtimeFuncName),
		}, nil
	}

	// 7. 调用运行时函数（根据类型选择不同的参数类型）
	_, err = irManager.CreateCall(mapDeleteFunc, mapValueLLVM, keyValueLLVM)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to call %s: %w", runtimeFuncName, err),
		}, nil
	}

	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// convertReturnValueType 转换返回值的类型以匹配函数返回类型
func (sg *StatementGeneratorImpl) convertReturnValueType(
	irManager generation.IRModuleManager,
	value llvalue.Value,
	fromType types.Type,
	toType types.Type,
) (llvalue.Value, error) {
	// 如果类型已经匹配，直接返回
	if types.Equal(fromType, toType) {
		return value, nil
	}

	// 处理整数类型转换
	fromInt, fromIsInt := fromType.(*types.IntType)
	toInt, toIsInt := toType.(*types.IntType)

	if fromIsInt && toIsInt {
		fromBits := fromInt.BitSize
		toBits := toInt.BitSize

		if fromBits < toBits {
			// 扩展：小整数类型 → 大整数类型（如 i32 → i64）
			// 使用 ZExt（零扩展），适用于无符号整数
			// 注意：如果需要区分有符号和无符号，需要从类型信息中获取
			currentBlock := irManager.GetCurrentBasicBlock()
			if currentBlock == nil {
				return nil, fmt.Errorf("no current basic block set")
			}
			block, ok := currentBlock.(*ir.Block)
			if !ok {
				return nil, fmt.Errorf("invalid block type")
			}
			ext := block.NewZExt(value, toType)
			return ext, nil
		} else if fromBits > toBits {
			// 截断：大整数类型 → 小整数类型（如 i64 → i32）
			currentBlock := irManager.GetCurrentBasicBlock()
			if currentBlock == nil {
				return nil, fmt.Errorf("no current basic block set")
			}
			block, ok := currentBlock.(*ir.Block)
			if !ok {
				return nil, fmt.Errorf("invalid block type")
			}
			trunc := block.NewTrunc(value, toType)
			return trunc, nil
		}
	}

	// 处理浮点类型转换（float <-> double）
	// LLVM 不允许 BitCast 在 double 与 float 之间，必须用 FPExt/FPTrunc
	// 先按 types.Equal 与预定义类型判断，避免依赖 IsFloat/FloatType 在不同 llir 版本下的行为
	currentBlock := irManager.GetCurrentBasicBlock()
	if currentBlock == nil {
		return nil, fmt.Errorf("no current basic block set")
	}
	block, ok := currentBlock.(*ir.Block)
	if !ok {
		return nil, fmt.Errorf("invalid block type")
	}
	if types.Equal(fromType, types.Double) && types.Equal(toType, types.Float) {
		return block.NewFPTrunc(value, toType), nil
	}
	if types.Equal(fromType, types.Float) && types.Equal(toType, types.Double) {
		return block.NewFPExt(value, toType), nil
	}
	if types.IsFloat(fromType) && types.IsFloat(toType) {
		fromFloat := fromType.(*types.FloatType)
		toFloat := toType.(*types.FloatType)
		fromKind := fromFloat.Kind
		toKind := toFloat.Kind
		if fromKind == types.FloatKindFloat && toKind == types.FloatKindDouble {
			return block.NewFPExt(value, toType), nil
		}
		if fromKind == types.FloatKindDouble && toKind == types.FloatKindFloat {
			return block.NewFPTrunc(value, toType), nil
		}
	}

	// 处理结构体指针到i8*的转换（用于MapIterResult等）
	// 例如：{ i8*, i32, float }* -> i8*
	if srcPtrType, srcIsPtr := fromType.(*types.PointerType); srcIsPtr {
		if dstPtrType, dstIsPtr := toType.(*types.PointerType); dstIsPtr {
			// 如果源类型和目标类型都是指针，使用BitCast
			if _, srcIsStruct := srcPtrType.ElemType.(*types.StructType); srcIsStruct {
				// 源类型是结构体指针，目标类型是i8*指针，使用BitCast
				if dstPtrType.ElemType == types.I8 {
					currentBlock := irManager.GetCurrentBasicBlock()
					if currentBlock == nil {
						return nil, fmt.Errorf("no current basic block set")
					}
					block, ok := currentBlock.(*ir.Block)
					if !ok {
						return nil, fmt.Errorf("invalid block type")
					}
					bitcast := block.NewBitCast(value, toType)
					return bitcast, nil
				}
			}
		}
	}

	// 其他类型转换暂不支持
	return nil, fmt.Errorf("unsupported type conversion from %s to %s", fromType, toType)
}

// GenerateMatchStatement 生成模式匹配语句
func (sg *StatementGeneratorImpl) GenerateMatchStatement(irManager generation.IRModuleManager, stmt *entities.MatchStmt) (*generation.StatementGenerationResult, error) {
	fmt.Printf("DEBUG: GenerateMatchStatement called with %d cases\n", len(stmt.Cases))
	if len(stmt.Cases) == 0 {
		return &generation.StatementGenerationResult{
			Success: true,
		}, nil
	}

	// 求值 match 表达式的值
	// 注意：这个求值会在当前的 block（entry block）中生成指令
	// 但这是正常的，因为我们需要在 entry block 中求值 match 表达式
	currentBlockBeforeEval := irManager.GetCurrentBasicBlock()
	fmt.Printf("DEBUG: GenerateMatchStatement - evaluating match expression, current block: %v\n", currentBlockBeforeEval)
	matchValue, err := sg.expressionEvaluator.Evaluate(irManager, stmt.Value)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate match expression: %w", err),
		}, nil
	}
	currentBlockAfterEval := irManager.GetCurrentBasicBlock()
	if currentBlockBeforeEval != currentBlockAfterEval {
		fmt.Printf("DEBUG: GenerateMatchStatement - WARNING: current block changed during match expression evaluation\n")
		// 恢复 current block
		irManager.SetCurrentBasicBlock(currentBlockBeforeEval)
	}
	fmt.Printf("DEBUG: GenerateMatchStatement - matchValue type: %T\n", matchValue)

	// 保存当前基本块
	currentBlock := irManager.GetCurrentBasicBlock()
	if currentBlock == nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("no current basic block set"),
		}, nil
	}

	// 创建 merge 块（所有 case 执行完后跳转到这里）
	mergeBlock, err := irManager.CreateBasicBlock("match.end")
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create merge block: %w", err),
		}, nil
	}

	// 第一步：收集所有 pattern 变量，并在 entry block 中创建它们的 alloca
	// LLVM 要求 alloca 必须在函数的 entry block 中
	patternVarAllocas := make(map[string]interface{}) // pattern variable name -> alloca
	for _, caseStmt := range stmt.Cases {
		var patternVarName string
		if funcCall, ok := caseStmt.Pattern.(*entities.FuncCall); ok {
			if len(funcCall.Args) > 0 {
				if varIdent, ok := funcCall.Args[0].(*entities.Identifier); ok {
					patternVarName = varIdent.Name
				}
			}
		} else if ident, ok := caseStmt.Pattern.(*entities.Identifier); ok {
			patternVarName = ident.Name
		}
		
		// 如果提取到了变量名且尚未创建 alloca，则在 entry block 中创建
		// Result 的 Ok/Err 分支变量存 data 指针，类型为 i8*
		if patternVarName != "" && patternVarAllocas[patternVarName] == nil {
			err = irManager.SetCurrentBasicBlock(currentBlock)
			if err != nil {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("failed to set entry block for alloca creation: %w", err),
				}, nil
			}
			llvmType := types.NewPointer(types.I8)
			alloca, err := irManager.CreateAlloca(llvmType, patternVarName)
			if err != nil {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("failed to create alloca for pattern variable %s: %w", patternVarName, err),
				}, nil
			}
			patternVarAllocas[patternVarName] = alloca
			err = sg.symbolManager.RegisterSymbol(patternVarName, "i8*", alloca)
			if err != nil {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("failed to register pattern variable %s: %w", patternVarName, err),
				}, nil
			}
		}
	}

	// 创建所有 case 块
	caseBlocks := make([]interface{}, len(stmt.Cases))
	for i := range stmt.Cases {
		caseBlockName := fmt.Sprintf("match.case.%d", i)
		caseBlock, err := irManager.CreateBasicBlock(caseBlockName)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to create case block: %w", err),
			}, nil
		}
		caseBlocks[i] = caseBlock
		fmt.Printf("DEBUG: GenerateMatchStatement - created case block %d: %v\n", i, caseBlock)
	}

	// 处理每个 case：生成链式的 if-else 结构
	nextCheckBlock := currentBlock
	for i, caseStmt := range stmt.Cases {
		caseBlock := caseBlocks[i]
		
		// 处理 pattern：提取变量名
		var patternVarName string
		
		if funcCall, ok := caseStmt.Pattern.(*entities.FuncCall); ok {
			fmt.Printf("DEBUG: GenerateMatchStatement - case %d pattern is FuncCall: %s\n", i, funcCall.Name)
			if len(funcCall.Args) > 0 {
				if varIdent, ok := funcCall.Args[0].(*entities.Identifier); ok {
					patternVarName = varIdent.Name
				}
			}
		} else if ident, ok := caseStmt.Pattern.(*entities.Identifier); ok {
			patternVarName = ident.Name
		}

		// 设置当前块为检查块
		err = irManager.SetCurrentBasicBlock(nextCheckBlock)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to set check block: %w", err),
			}, nil
		}

		// 确定下一个检查块（下一个 case 或 merge block）
		var nextBlock interface{}
		if i < len(stmt.Cases)-1 {
			// 还有下一个 case，创建检查块
			checkBlockName := fmt.Sprintf("match.check.%d", i+1)
			nextCheckBlock, err = irManager.CreateBasicBlock(checkBlockName)
			if err != nil {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("failed to create check block: %w", err),
				}, nil
			}
			nextBlock = nextCheckBlock
		} else {
			// 最后一个 case，不匹配时跳转到 merge block
			nextBlock = mergeBlock
		}

		// 生成条件分支：Result 为 *{ i8 tag, i8* data }，tag 0=Ok 1=Err；根据 case 的 pattern Ok/Err 比较 tag 后 CondBr。
		resultStructType := types.NewStruct(types.I8, types.NewPointer(types.I8))
		matchValueVal, ok := matchValue.(llvalue.Value)
		if !ok {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("match expression value is not llvm value: %T", matchValue),
			}, nil
		}
		tagPtr, err := irManager.CreateGetElementPtr(resultStructType, matchValueVal, constant.NewInt(types.I32, 0), constant.NewInt(types.I32, 0))
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("gep tag: %w", err),
			}, nil
		}
		tagValue, err := irManager.CreateLoad(types.I8, tagPtr, "match_tag")
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("load tag: %w", err),
			}, nil
		}
		var cond interface{}
		if funcCall, ok := caseStmt.Pattern.(*entities.FuncCall); ok {
			switch funcCall.Name {
			case "Ok":
				cond, err = irManager.CreateBinaryOp("==", tagValue, constant.NewInt(types.I8, 0), "is_ok")
			case "Err":
				cond, err = irManager.CreateBinaryOp("==", tagValue, constant.NewInt(types.I8, 1), "is_err")
			default:
				cond = constant.NewInt(types.I1, 1)
				err = nil
			}
		} else {
			cond = constant.NewInt(types.I1, 1)
			err = nil
		}
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("tag comparison: %w", err),
			}, nil
		}
		err = irManager.CreateCondBr(cond, caseBlock, nextBlock)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to create conditional branch: %w", err),
			}, nil
		}

		// 生成 case body
		// 确保在 case block 中生成 case body
		fmt.Printf("DEBUG: GenerateMatchStatement - setting current block to case %d block before generating case body, current block: %v\n", i, irManager.GetCurrentBasicBlock())
		err = irManager.SetCurrentBasicBlock(caseBlock)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to set case block: %w", err),
			}, nil
		}
		
		// 验证当前 block 确实是 case block
		actualBlock := irManager.GetCurrentBasicBlock()
		fmt.Printf("DEBUG: GenerateMatchStatement - case %d: set current block to case block, actual block: %v\n", i, actualBlock)
		if actualBlock != caseBlock {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("current block mismatch: expected %v, got %v", caseBlock, actualBlock),
			}, nil
		}

		// 从 Result 的 data 字段提取 payload 指针并写入 pattern 变量的 alloca
		if patternVarName != "" {
			alloca := patternVarAllocas[patternVarName]
			if alloca == nil {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("alloca for pattern variable %s not found", patternVarName),
				}, nil
			}
			dataPtr, err := irManager.CreateGetElementPtr(resultStructType, matchValueVal, constant.NewInt(types.I32, 0), constant.NewInt(types.I32, 1))
			if err != nil {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("gep data for pattern %s: %w", patternVarName, err),
				}, nil
			}
			dataValue, err := irManager.CreateLoad(types.NewPointer(types.I8), dataPtr, patternVarName+"_data")
			if err != nil {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("load data for pattern %s: %w", patternVarName, err),
				}, nil
			}
			if err = irManager.CreateStore(dataValue, alloca); err != nil {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("store pattern variable %s: %w", patternVarName, err),
				}, nil
			}
		}

		// 为 case body 创建新的作用域
		err = sg.symbolManager.EnterScope()
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to enter case body scope: %w", err),
			}, nil
		}

		// 生成 case body 语句
		fmt.Printf("DEBUG: GenerateMatchStatement - case %d body has %d statements, current block before generation: %v\n", i, len(caseStmt.Body), irManager.GetCurrentBasicBlock())
		for _, bodyStmt := range caseStmt.Body {
			_, err = sg.GenerateStatement(irManager, bodyStmt)
			if err != nil {
				// 退出 case body 作用域
				sg.symbolManager.ExitScope()
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("failed to generate match case body: %w", err),
				}, nil
			}
		}
		// 退出 case body 作用域
		err = sg.symbolManager.ExitScope()
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to exit case body scope: %w", err),
			}, nil
		}
		fmt.Printf("DEBUG: GenerateMatchStatement - case %d body generated, current block after generation: %v\n", i, irManager.GetCurrentBasicBlock())

		// 检查 case body 是否以 return 结束
		// 如果没有 terminator，添加跳转到 merge 块
		currentCaseBlock := irManager.GetCurrentBasicBlock()
		if currentCaseBlock != nil {
			if llvmBlock, ok := currentCaseBlock.(*ir.Block); ok {
				if llvmBlock.Term == nil {
					fmt.Printf("DEBUG: GenerateMatchStatement - case %d block has no terminator, adding branch to merge\n", i)
					err = irManager.CreateBr(mergeBlock)
					if err != nil {
						return &generation.StatementGenerationResult{
							Success: false,
							Error:   fmt.Errorf("failed to create branch to merge: %w", err),
						}, nil
					}
				} else {
					fmt.Printf("DEBUG: GenerateMatchStatement - case %d block has terminator: %T\n", i, llvmBlock.Term)
				}
			}
		}
	}

	// 设置 merge 块为当前块
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

// GenerateBlockStatement 生成块语句：按顺序生成块内每条语句
func (sg *StatementGeneratorImpl) GenerateBlockStatement(irManager generation.IRModuleManager, stmt *entities.BlockStmt) (*generation.StatementGenerationResult, error) {
	for _, subStmt := range stmt.Statements {
		result, err := sg.GenerateStatement(irManager, subStmt)
		if err != nil {
			return &generation.StatementGenerationResult{Success: false, Error: err}, err
		}
		if !result.Success {
			return result, result.Error
		}
	}
	return &generation.StatementGenerationResult{Success: true}, nil
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
		constName := fmt.Sprintf("%s_%s", stmt.Name, variant.Name)

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
	result := &generation.StatementGenerationResult{
		Success: true,
	}

	// 解析返回类型（使用 MapType 支持复杂类型如 Option[T]）
	returnType, err := sg.typeMapper.MapType(stmt.ReturnType)
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
	
	// 在方法定义时，从接收者类型推断类型参数映射
	// 例如：如果接收者类型是 *HashMap[K, V]，则设置类型参数映射
	// 注意：这里我们只能设置类型参数名，实际类型值需要在方法调用时推断
	// 但是，我们可以保存接收者类型参数列表，供方法体内部使用
	fmt.Printf("DEBUG: GenerateMethodDefinition - funcName=%s, receiverType=%s, ReceiverParams count=%d\n", funcName, stmt.Receiver, len(stmt.ReceiverParams))
	if len(stmt.ReceiverParams) > 0 {
		// 保存接收者类型参数列表（用于方法体内部的类型推断）
		receiverTypeParamNames := make([]string, len(stmt.ReceiverParams))
		for i, tp := range stmt.ReceiverParams {
			receiverTypeParamNames[i] = tp.Name
		}
		// 将接收者类型参数列表保存到函数定义中
		// 注意：这里我们使用函数名作为key，但实际上方法名是 Type_MethodName
		irManager.SetFunctionTypeParamNames(funcName, receiverTypeParamNames)
		fmt.Printf("DEBUG: GenerateMethodDefinition - funcName=%s, receiverTypeParams=%v\n", funcName, receiverTypeParamNames)
		
		// ✅ 新增：保存泛型方法定义，供延迟编译使用
		// 注意：方法定义也需要延迟编译，因为方法体中的trait方法调用需要知道类型参数的实际值
		// 将方法定义转换为函数定义格式，以便复用延迟编译逻辑
		// 重要：需要将接收者参数添加到 Params 的开头，这样 CompileMonomorphizedFunction 才能正确处理
		methodParams := make([]entities.Param, 0, len(stmt.Params)+1)
		// 第一个参数是接收者参数
		methodParams = append(methodParams, entities.Param{
			Name: stmt.ReceiverVar, // 接收者变量名，如 "m"
			Type: stmt.Receiver,    // 接收者类型，如 "*HashMap[K, V]"
		})
		// 然后是方法参数
		methodParams = append(methodParams, stmt.Params...)
		
		methodAsFuncDef := &entities.FuncDef{
			Name:       funcName,
			TypeParams: stmt.ReceiverParams, // 使用接收者类型参数作为函数类型参数
			Params:     methodParams,         // 包含接收者参数和方法参数
			ReturnType: stmt.ReturnType,
			Body:       stmt.Body,
		}
		irManager.SaveGenericFunctionDefinition(funcName, methodAsFuncDef)
		fmt.Printf("DEBUG: SaveGenericFunctionDefinition - funcName=%s, saved generic method definition\n", funcName)
		
		// ✅ 验证保存是否成功
		savedDef, exists := irManager.GetGenericFunctionDefinition(funcName)
		if !exists || savedDef == nil {
			fmt.Printf("DEBUG: ERROR - Failed to save method definition: %s\n", funcName)
			result.Success = false
			result.Error = fmt.Errorf("failed to save generic method definition: %s", funcName)
			return result, result.Error
		} else {
			fmt.Printf("DEBUG: Verified - Method definition saved successfully: %s\n", funcName)
		}
		
		// ✅ 修改：如果是泛型方法，只生成方法声明，不生成方法体
		// 方法体将在方法调用时延迟编译（当类型参数已知时）
		// 注意：方法声明已经通过CreateFunction创建，这里只需要返回成功即可
		return &generation.StatementGenerationResult{
			Success: true,
		}, nil
	}

	// 非泛型方法：正常生成方法体
	// ✅ 修复：检查方法是否已经有 body（已经编译过），避免重复处理
	if len(function.(*ir.Func).Blocks) > 0 {
		// 方法已经有 body，跳过方法体生成
		fmt.Printf("DEBUG: GenerateMethodDefinition - method %s already has body (%d blocks), skipping\n", funcName, len(function.(*ir.Func).Blocks))
		return &generation.StatementGenerationResult{
			Success: true,
		}, nil
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

	// 处理接收者参数（第一个参数）
	if len(function.(*ir.Func).Params) > 0 {
		receiverParam := function.(*ir.Func).Params[0]
		receiverAlloca, err := irManager.CreateAlloca(receiverParam.Type(), "receiver_addr")
		if err != nil {
			result.Success = false
			result.Error = fmt.Errorf("failed to create alloca for receiver: %w", err)
			return result, result.Error
		}
		if err := irManager.CreateStore(receiverParam, receiverAlloca); err != nil {
			result.Success = false
			result.Error = fmt.Errorf("failed to store receiver: %w", err)
			return result, result.Error
		}
		// 注册接收者变量到符号表
		if err := sg.symbolManager.RegisterSymbol(stmt.ReceiverVar, stmt.Receiver, receiverAlloca); err != nil {
			result.Success = false
			result.Error = fmt.Errorf("failed to register receiver symbol: %w", err)
			return result, result.Error
		}
	}

	// 处理方法参数
	for i, param := range stmt.Params {
		paramIndex := i + 1 // 第一个参数是接收者
		if paramIndex < len(function.(*ir.Func).Params) {
			funcParam := function.(*ir.Func).Params[paramIndex]
			paramAlloca, err := irManager.CreateAlloca(funcParam.Type(), param.Name+"_addr")
			if err != nil {
				result.Success = false
				result.Error = fmt.Errorf("failed to create alloca for parameter %s: %w", param.Name, err)
				return result, result.Error
			}
			if err := irManager.CreateStore(funcParam, paramAlloca); err != nil {
				result.Success = false
				result.Error = fmt.Errorf("failed to store parameter %s: %w", param.Name, err)
				return result, result.Error
			}
			// 注册参数到符号表
			if err := sg.symbolManager.RegisterSymbol(param.Name, param.Type, paramAlloca); err != nil {
				result.Success = false
				result.Error = fmt.Errorf("failed to register parameter symbol %s: %w", param.Name, err)
				return result, result.Error
			}
		}
	}

	// 进入方法体作用域
	if err := sg.symbolManager.EnterScope(); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("failed to enter method body scope: %w", err)
		return result, result.Error
	}

	// 生成方法体中的语句
	if stmt.Body != nil {
		for _, bodyStmt := range stmt.Body {
			stmtResult, err := sg.GenerateStatement(irManager, bodyStmt)
			if err != nil {
				sg.symbolManager.ExitScope()
				result.Success = false
				result.Error = fmt.Errorf("failed to generate statement in method %s: %w", stmt.Name, err)
				return result, result.Error
			}
			if !stmtResult.Success {
				sg.symbolManager.ExitScope()
				return stmtResult, nil
			}
		}
	}

	// 退出方法体作用域
	if err := sg.symbolManager.ExitScope(); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("failed to exit method body scope: %w", err)
		return result, result.Error
	}

	// 如果方法没有显式返回语句，根据返回类型添加返回
	if stmt.ReturnType == "void" {
		if err := irManager.CreateRet(nil); err != nil {
			result.Success = false
			result.Error = fmt.Errorf("failed to create return instruction: %w", err)
			return result, result.Error
		}
	} else {
		// 为非void方法添加默认返回值：根据当前函数的 LLVM 返回类型生成对应零值
		defaultRetVal, err := sg.createDefaultReturnValueForType(irManager, function.(*ir.Func).Sig.RetType)
		if err != nil {
			result.Success = false
			result.Error = fmt.Errorf("failed to create default return value: %w", err)
			return result, result.Error
		}
		if err := irManager.CreateRet(defaultRetVal); err != nil {
			result.Success = false
			result.Error = fmt.Errorf("failed to create return instruction: %w", err)
			return result, result.Error
		}
	}

	return result, nil
}

// createDefaultReturnValueForType 根据函数返回类型创建默认零值（用于无显式 return 时的默认返回）
func (sg *StatementGeneratorImpl) createDefaultReturnValueForType(irManager generation.IRModuleManager, retType types.Type) (llvalue.Value, error) {
	switch t := retType.(type) {
	case *types.IntType:
		return constant.NewInt(t, 0), nil
	case *types.FloatType:
		// float/f32 -> 0.0；double/f64 -> 0.0
		return constant.NewFloat(t, 0), nil
	case *types.PointerType:
		return constant.NewNull(t), nil
	case *types.StructType:
		return constant.NewZeroInitializer(t), nil
	case *types.ArrayType:
		return constant.NewZeroInitializer(t), nil
	default:
		// 兜底：常见为 i32
		if types.Equal(retType, types.I32) {
			return constant.NewInt(types.I32, 0), nil
		}
		return nil, fmt.Errorf("unsupported return type for default value: %s", retType)
	}
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
			return "print_bool", nil
		case "float":
			return "print_float", nil
		default:
			return "", fmt.Errorf("unsupported type for printing: %s", symbolInfo.Type)
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

// mapParameterType 映射参数类型
// 对于复杂类型，使用指针类型简化处理
func (sg *StatementGeneratorImpl) mapParameterType(echoType string) (interface{}, error) {
	// 使用 MapType 支持复杂类型（如 Vec[T], Option[T] 等）
	if llvmType, err := sg.typeMapper.MapType(echoType); err == nil {
		return llvmType, nil
	}

	// 对于无法映射的类型，使用指针类型
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
	// 简化策略：async函数的行为和普通函数完全一样
	// async关键字只是一个标记，表示这个函数可以被异步调用
	// 实际的异步行为由spawn和await表达式实现

	// 注册async函数符号（返回Future类型）
	returnTypeStr := fmt.Sprintf("Future[%s]", stmt.ReturnType)
	err := sg.symbolManager.RegisterFunctionSymbol(stmt.Name, returnTypeStr, nil, true)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to register async function symbol: %w", err),
		}, nil
	}

	// 创建普通的同步函数（不返回Future）
	funcName := stmt.Name + "_sync" // 内部同步函数名
	returnType, err := sg.typeMapper.MapType(stmt.ReturnType)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to map return type %s: %w", stmt.ReturnType, err),
		}, nil
	}
	returnType, err = sg.getLLVMTypeFromMappedType(returnType)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to get LLVM return type: %w", err),
		}, nil
	}

	// 映射参数类型
	var paramTypes []interface{}
	for _, param := range stmt.Params {
		paramType, err := sg.typeMapper.MapType(param.Type)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to map parameter type %s: %w", param.Type, err),
			}, nil
		}
		paramTypes = append(paramTypes, paramType)
	}

	// 创建同步函数
	syncFuncPtr, err := irManager.CreateFunction(funcName, returnType, paramTypes)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to create sync function: %v", err),
		}, nil
	}

	// 设置同步函数上下文并生成函数体
	err = irManager.SetCurrentFunction(syncFuncPtr)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to set sync function: %v", err),
		}, nil
	}

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
			Error:   fmt.Errorf("failed to set entry block: %v", err),
		}, nil
	}

	// 生成函数体
	if stmt.Body != nil && len(stmt.Body) > 0 {
		for _, bodyStmt := range stmt.Body {
			result, err := sg.GenerateStatement(irManager, bodyStmt)
			if err != nil {
				return &generation.StatementGenerationResult{
					Success: false,
					Error:   fmt.Errorf("failed to generate statement: %v", err),
				}, nil
			}
			if !result.Success {
				return result, nil
			}
		}
	}

	// 注册同步函数
	err = irManager.RegisterFunction(funcName, syncFuncPtr)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to register sync function: %w", err),
		}, nil
	}

	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// GenerateStructDefinition 生成结构体定义
// 职责：为结构体定义生成 LLVM 结构体类型，并注册到类型映射器
func (sg *StatementGeneratorImpl) GenerateStructDefinition(irManager generation.IRModuleManager, stmt *entities.StructDef) (*generation.StatementGenerationResult, error) {
	// 1. 构建字段类型列表
	fieldTypes := make([]types.Type, len(stmt.Fields))
	for i, field := range stmt.Fields {
		// 映射字段类型
		// 注意：对于泛型结构体，字段类型可能包含类型参数（如 []T）
		// 这种情况下，我们需要特殊处理
		typeInterface, err := sg.typeMapper.MapType(field.Type)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to map field type %s: %w", field.Type, err),
			}, nil
		}

		// 转换为 LLVM 类型
		llvmType, err := sg.getLLVMTypeFromMappedType(typeInterface)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to get LLVM type for field %s (type %s): %w", field.Name, field.Type, err),
			}, nil
		}
		fieldTypes[i] = llvmType
	}

	// 2. 创建 LLVM 结构体类型
	structType := types.NewStruct(fieldTypes...)

	// 3. 注册结构体类型到类型映射器（供后续使用）
	// 注意：对于泛型结构体（如 Vec[T]），类型名包含类型参数
	// 但这里我们只注册基础名称，泛型实例化会在使用时处理
	structTypeName := stmt.Name

	// 4. 注册到类型映射器
	err := sg.typeMapper.RegisterCustomType(structTypeName, structType)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to register struct type %s: %w", structTypeName, err),
		}, nil
	}

	// 5. 在 IR 模块中声明结构体类型（如果需要）
	// 注意：LLVM IR 中的结构体类型是隐式声明的，不需要显式创建
	// 当使用结构体类型时，LLVM 会自动处理类型定义

	// 6. ✅ 新增：将结构体定义添加到表达式求值器的 structDefinitions 中
	// 这样在访问结构体字段时（如 m.buckets）可以找到结构体定义
	if exprEvalImpl, ok := sg.expressionEvaluator.(*ExpressionEvaluatorImpl); ok {
		// 提取基础类型名（移除类型参数，如 HashMap[K, V] -> HashMap）
		baseTypeName := structTypeName
		if bracketIndex := strings.Index(structTypeName, "["); bracketIndex != -1 {
			baseTypeName = structTypeName[:bracketIndex]
		}
		// 如果还没有注册，则注册（避免覆盖已存在的定义）
		if _, exists := exprEvalImpl.structDefinitions[baseTypeName]; !exists {
			exprEvalImpl.structDefinitions[baseTypeName] = stmt
			fmt.Printf("DEBUG: GenerateStructDefinition - registered struct definition: %s (base type: %s)\n", structTypeName, baseTypeName)
		}
	}

	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// generateMapIndexAssignment 生成map索引赋值的LLVM IR
// 调用运行时函数 runtime_map_set(map_ptr, key, value)
func (sg *StatementGeneratorImpl) generateMapIndexAssignment(
	irManager generation.IRModuleManager,
	mapTarget *MapIndexAssignmentTarget,
	valueExpr entities.Expr,
) (*generation.StatementGenerationResult, error) {
	// 1. 获取Map的类型信息
	mapType := sg.getMapTypeFromExpr(mapTarget.MapExpr)
	if mapType == "" {
		// 如果无法确定类型，默认使用map[string]string（向后兼容）
		mapType = "map[string]string"
	}

	// 2. 解析Map类型（提取键类型和值类型）
	keyType, valueType, err := sg.parseMapType(mapType)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to parse map type %s: %w", mapType, err),
		}, nil
	}

	// 3. 根据键类型和值类型选择运行时函数
	runtimeFuncName, err := sg.selectMapRuntimeFunction(keyType, valueType, "set")
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to select runtime function for map[%s]%s: %w", keyType, valueType, err),
		}, nil
	}
	
	// DEBUG: 打印选择的运行时函数名
	fmt.Printf("DEBUG: generateMapIndexAssignment - keyType=%s, valueType=%s, runtimeFuncName=%s\n", keyType, valueType, runtimeFuncName)

	// 4. 求值map表达式（获取map指针）
	mapValue, err := sg.expressionEvaluator.Evaluate(irManager, mapTarget.MapExpr)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate map expression: %w", err),
		}, nil
	}

	mapValueLLVM, ok := mapValue.(llvalue.Value)
	if !ok {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("map value is not a valid LLVM value: %T", mapValue),
		}, nil
	}

	// 5. 求值key表达式（根据键类型处理）
	keyValue, err := sg.expressionEvaluator.Evaluate(irManager, mapTarget.KeyExpr)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate key expression: %w", err),
		}, nil
	}

	keyValueLLVM, ok := keyValue.(llvalue.Value)
	if !ok {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("key value is not a valid LLVM value: %T", keyValue),
		}, nil
	}

	// 6. 求值value表达式（根据值类型处理）
	valueResult, err := sg.expressionEvaluator.Evaluate(irManager, valueExpr)
	if err != nil {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("failed to evaluate value expression: %w", err),
		}, nil
	}

	valueResultLLVM, ok := valueResult.(llvalue.Value)
	if !ok {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("value is not a valid LLVM value: %T", valueResult),
		}, nil
	}

	// 7. 获取对应的运行时函数
	mapSetFunc, exists := irManager.GetExternalFunction(runtimeFuncName)
	if !exists {
		return &generation.StatementGenerationResult{
			Success: false,
			Error:   fmt.Errorf("runtime function %s not declared", runtimeFuncName),
		}, nil
	}

	// 8. ✅ 新增：检查是否是结构体键函数（runtime_map_set_struct_*）
	// 如果是，需要传递hash和equals函数指针
	if strings.HasPrefix(runtimeFuncName, "runtime_map_set_struct_") || strings.HasPrefix(runtimeFuncName, "runtime_map_get_struct_") {
		// 提取结构体类型名（从keyType中提取）
		baseKeyTypeName := keyType
		if bracketIndex := strings.Index(keyType, "["); bracketIndex != -1 {
			baseKeyTypeName = keyType[:bracketIndex]
		}
		// 移除指针前缀（如果有）
		if strings.HasPrefix(baseKeyTypeName, "*") {
			baseKeyTypeName = baseKeyTypeName[1:]
		}
		
		// 确保hash和equals函数已生成
		exprEvalImpl, ok := sg.expressionEvaluator.(*ExpressionEvaluatorImpl)
		if !ok {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("expressionEvaluator is not ExpressionEvaluatorImpl"),
			}, nil
		}
		
		hashFunc, err := exprEvalImpl.ensureStructHashFunction(baseKeyTypeName)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to ensure hash function for %s: %w", baseKeyTypeName, err),
			}, nil
		}
		
		equalsFunc, err := exprEvalImpl.ensureStructEqualsFunction(baseKeyTypeName)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to ensure equals function for %s: %w", baseKeyTypeName, err),
			}, nil
		}
		
		// 获取函数指针（*ir.Func可以直接传递，CreateCall会处理）
		// 注意：在LLVM IR中，函数本身就是值类型，可以直接传递
		// CreateCall现在支持*ir.Func作为参数
		
		// 调用运行时函数，传递函数指针
		_, err = irManager.CreateCall(mapSetFunc, mapValueLLVM, keyValueLLVM, valueResultLLVM, hashFunc, equalsFunc)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to call %s: %w", runtimeFuncName, err),
			}, nil
		}
	} else {
		// 8. 调用运行时函数（根据类型选择不同的参数类型）
		// 例如：map[int]string -> runtime_map_set_int_string(map, key_i32, value_string)
		//      map[string]string -> runtime_map_set_string_string(map, key_string, value_string)
		_, err = irManager.CreateCall(mapSetFunc, mapValueLLVM, keyValueLLVM, valueResultLLVM)
		if err != nil {
			return &generation.StatementGenerationResult{
				Success: false,
				Error:   fmt.Errorf("failed to call %s: %w", runtimeFuncName, err),
			}, nil
		}
	}

	return &generation.StatementGenerationResult{
		Success: true,
	}, nil
}

// getMapTypeFromExpr 从表达式获取Map类型
func (sg *StatementGeneratorImpl) getMapTypeFromExpr(expr entities.Expr) string {
	// 如果表达式是标识符，从符号表获取类型
	if ident, ok := expr.(*entities.Identifier); ok {
		if symbolInfo, err := sg.symbolManager.LookupSymbol(ident.Name); err == nil {
			return symbolInfo.Type
		}
	}
	// 如果表达式是结构体访问（如 map.buckets），获取结构体类型
	if structAccess, ok := expr.(*entities.StructAccess); ok {
		if ident, ok := structAccess.Object.(*entities.Identifier); ok {
			if symbolInfo, err := sg.symbolManager.LookupSymbol(ident.Name); err == nil {
				return symbolInfo.Type
			}
		}
	}
	return ""
}

// parseMapType 解析Map类型，返回键类型和值类型
// 例如：map[int]string -> keyType="int", valueType="string"
func (sg *StatementGeneratorImpl) parseMapType(mapType string) (keyType string, valueType string, error error) {
	if !strings.HasPrefix(mapType, "map[") {
		return "", "", fmt.Errorf("not a map type: %s", mapType)
	}

	bracketEnd := strings.Index(mapType, "]")
	if bracketEnd == -1 {
		return "", "", fmt.Errorf("invalid map type syntax: missing ']' in %s", mapType)
	}

	keyType = mapType[4:bracketEnd] // 移除 "map["，获取键类型
	valueType = strings.TrimSpace(mapType[bracketEnd+1:]) // 获取"]"之后的值类型

	if valueType == "" {
		return "", "", fmt.Errorf("invalid map type syntax: missing value type in %s", mapType)
	}

	return keyType, valueType, nil
}

// selectMapRuntimeFunction 根据键类型和值类型选择运行时函数名
func (sg *StatementGeneratorImpl) selectMapRuntimeFunction(keyType, valueType, operation string) (string, error) {
	// 规范化类型名称（用于函数名）
	keyTypeName := sg.normalizeTypeName(keyType)
	valueTypeName := sg.normalizeTypeName(valueType)

	// ✅ 新增：特殊处理：结构体类型作为键
	// 例如：map[User]string -> runtime_map_set_struct_string
	// 检查键类型是否是结构体类型
	if keyTypeName == "unknown" || keyTypeName == "struct" {
		// 提取基础类型名（移除类型参数，如果有）
		baseKeyTypeName := keyType
		if bracketIndex := strings.Index(keyType, "["); bracketIndex != -1 {
			baseKeyTypeName = keyType[:bracketIndex]
		}
		
		// 检查是否在structDefinitions中（通过expressionEvaluator访问）
		exprEvalImpl, ok := sg.expressionEvaluator.(*ExpressionEvaluatorImpl)
		if ok {
			fmt.Printf("DEBUG: selectMapRuntimeFunction - checking struct key: baseKeyTypeName=%s, structDefinitions keys: %v\n", baseKeyTypeName, func() []string {
				keys := make([]string, 0, len(exprEvalImpl.structDefinitions))
				for k := range exprEvalImpl.structDefinitions {
					keys = append(keys, k)
				}
				return keys
			}())
			if _, exists := exprEvalImpl.structDefinitions[baseKeyTypeName]; exists {
				// 是结构体键类型，使用通用结构体键函数
				// 生成函数名：runtime_map_{operation}_struct_{valueType}
				valueTypeNameNormalized := sg.normalizeTypeName(valueType)
				funcName := fmt.Sprintf("runtime_map_%s_struct_%s", operation, valueTypeNameNormalized)
				fmt.Printf("DEBUG: selectMapRuntimeFunction - struct key detected, funcName=%s\n", funcName)
				return funcName, nil
			}
		}
	}

	// 特殊处理：对于len和clear操作，使用类型无关的通用函数
	if operation == "len" {
		return "runtime_map_len", nil
	}
	if operation == "clear" {
		return "runtime_map_clear", nil
	}

	// 对于contains操作，函数名格式为：runtime_map_contains_{keyType}_{valueType}
	if operation == "contains" {
		funcName := fmt.Sprintf("runtime_map_contains_%s_%s", keyTypeName, valueTypeName)
		return funcName, nil
	}

	// 对于keys操作，函数名格式为：runtime_map_keys_{keyType}_{valueType}
	if operation == "keys" {
		funcName := fmt.Sprintf("runtime_map_keys_%s_%s", keyTypeName, valueTypeName)
		return funcName, nil
	}

	// 对于values操作，函数名格式为：runtime_map_values_{keyType}_{valueType}
	if operation == "values" {
		funcName := fmt.Sprintf("runtime_map_values_%s_%s", keyTypeName, valueTypeName)
		return funcName, nil
	}

	// ✅ 新增：特殊处理：结构体类型作为值
	// 例如：map[string]User -> runtime_map_set_string_struct
	if valueTypeName == "unknown" {
		// 提取基础类型名（移除类型参数，如果有）
		baseTypeName := valueType
		if bracketIndex := strings.Index(valueType, "["); bracketIndex != -1 {
			baseTypeName = valueType[:bracketIndex]
		}
		
		// 检查是否在structDefinitions中（通过expressionEvaluator访问）
		exprEvalImpl, ok := sg.expressionEvaluator.(*ExpressionEvaluatorImpl)
		if ok {
			if _, exists := exprEvalImpl.structDefinitions[baseTypeName]; exists {
				// 是结构体类型，使用通用结构体函数
				// 生成函数名：runtime_map_{operation}_{keyType}_struct
				funcName := fmt.Sprintf("runtime_map_%s_%s_struct", operation, keyTypeName)
				return funcName, nil
			}
		}
	}

	// ✅ 新增：特殊处理：指针类型作为值
	// 例如：map[string]*User -> runtime_map_set_string_struct（复用结构体实现）
	if strings.HasPrefix(valueType, "*") {
		// 指针类型在运行时是 i8* 指针，复用结构体值的实现
		// 生成函数名：runtime_map_{operation}_{keyType}_struct
		funcName := fmt.Sprintf("runtime_map_%s_%s_struct", operation, keyTypeName)
		return funcName, nil
	}

	// ✅ 新增：特殊处理：Option类型作为值
	// 例如：map[string]Option[int] -> runtime_map_set_string_struct（复用结构体实现）
	if strings.HasPrefix(valueType, "Option[") && strings.HasSuffix(valueType, "]") {
		// Option[T] 在运行时是 i8* 指针，复用结构体值的实现
		// 生成函数名：runtime_map_{operation}_{keyType}_struct
		funcName := fmt.Sprintf("runtime_map_%s_%s_struct", operation, keyTypeName)
		return funcName, nil
	}

	// 生成函数名：runtime_map_{operation}_{keyType}_{valueType}
	funcName := fmt.Sprintf("runtime_map_%s_%s_%s", operation, keyTypeName, valueTypeName)
	return funcName, nil
}

// normalizeTypeName 规范化类型名称（用于函数名）
func (sg *StatementGeneratorImpl) normalizeTypeName(echoType string) string {
	switch echoType {
	case "int", "i32":
		return "int"
	case "i64":
		return "int64"
	case "string":
		return "string"
	case "float", "f32", "f64":
		return "float"
	case "bool":
		return "bool"
	default:
		// ✅ 新增：指针类型支持：*Type 在运行时是 i8* 指针，复用struct实现
		if strings.HasPrefix(echoType, "*") {
			return "struct"
		}
		
		// ✅ 新增：Option类型支持：Option[T] 在运行时是 i8* 指针，复用struct实现
		if strings.HasPrefix(echoType, "Option[") && strings.HasSuffix(echoType, "]") {
			return "struct"
		}
		
		// ✅ 结构体类型支持：检查是否是结构体类型
		// 提取基础类型名（移除类型参数，如 User[T] -> User）
		baseTypeName := echoType
		if bracketIndex := strings.Index(echoType, "["); bracketIndex != -1 {
			baseTypeName = echoType[:bracketIndex]
		}
		
		// 检查是否是结构体类型（通过expressionEvaluator访问）
		exprEvalImpl, ok := sg.expressionEvaluator.(*ExpressionEvaluatorImpl)
		if ok {
			if _, exists := exprEvalImpl.structDefinitions[baseTypeName]; exists {
				// 是结构体类型，返回"struct"（使用通用函数）
				return "struct"
			}
		}
		
		return "unknown"
	}
}

// CompileMonomorphizedFunction 编译单态化函数（延迟编译）
// 当泛型函数被调用且类型参数已知时，调用此方法生成单态化版本的函数
func (sg *StatementGeneratorImpl) CompileMonomorphizedFunction(
	irManager generation.IRModuleManager,
	funcDef *entities.FuncDef,
	typeParamMap map[string]string,
	monoName string,
) error {
	// 1. 创建单态化函数定义（复制原始函数定义）
	monoFuncDef := &entities.FuncDef{
		Name:       monoName,
		TypeParams: []entities.GenericParam{}, // 单态化函数没有类型参数
		Params:     make([]entities.Param, len(funcDef.Params)),
		ReturnType: sg.substituteTypeInString(funcDef.ReturnType, typeParamMap),
		Body:       funcDef.Body, // 复用函数体（类型参数替换在编译时通过类型参数映射完成）
	}

	// 2. 替换参数类型中的类型参数为具体类型
	for i, param := range funcDef.Params {
		substitutedType := sg.substituteTypeInString(param.Type, typeParamMap)
		monoFuncDef.Params[i] = entities.Param{
			Name: param.Name,
			Type: substitutedType,
		}
		fmt.Printf("DEBUG: CompileMonomorphizedFunction - param[%d]: %s: %s -> %s\n", i, param.Name, param.Type, substitutedType)
	}

	// 3. 设置类型参数映射（供函数体内部使用）
	savedTypeParams := irManager.GetCurrentFunctionTypeParams()
	irManager.SetCurrentFunctionTypeParams(typeParamMap)
	defer irManager.SetCurrentFunctionTypeParams(savedTypeParams)
	fmt.Printf("DEBUG: CompileMonomorphizedFunction - funcName=%s, monoName=%s, typeParamMap=%v\n", funcDef.Name, monoName, typeParamMap)

	// ✅ 3.5 清理可能存在的参数符号（避免重复注册）
	// 在延迟编译时，如果之前已经尝试编译过这个函数，参数可能已经在符号表中
	// 我们需要先清理这些参数，然后让 GenerateFuncDefinition 重新注册
	// 注意：这里只清理当前作用域的符号，不影响外层作用域
	for _, param := range monoFuncDef.Params {
		// 检查符号是否在当前作用域存在
		if sg.symbolManager.SymbolExistsInCurrentScope(param.Name) {
			// 如果存在，需要先删除（但 SymbolManager 没有删除方法）
			// 暂时跳过，让 GenerateFuncDefinition 处理（可能会报错，但我们可以捕获并处理）
			fmt.Printf("DEBUG: CompileMonomorphizedFunction - parameter %s already exists in current scope, will be overwritten\n", param.Name)
		}
	}
	
	// ✅ 3.6 进入新的函数作用域（确保参数注册在独立的作用域中）
	// 注意：GenerateFuncDefinition 内部会再次 EnterScope 用于函数体，但参数注册在函数作用域中
	if err := sg.symbolManager.EnterScope(); err != nil {
		return fmt.Errorf("failed to enter function scope for monomorphized function: %w", err)
	}
	defer func() {
		// 延迟退出作用域
		if err := sg.symbolManager.ExitScope(); err != nil {
			fmt.Printf("WARNING: failed to exit function scope: %v\n", err)
		}
	}()

	// 4. 生成函数定义（调用GenerateFuncDefinition，但这次会生成函数体，因为TypeParams为空）
	result, err := sg.GenerateFuncDefinition(irManager, monoFuncDef)
	if err != nil {
		return fmt.Errorf("failed to generate monomorphized function: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("failed to generate monomorphized function: %v", result.Error)
	}

	fmt.Printf("DEBUG: CompileMonomorphizedFunction - compiled %s -> %s\n", funcDef.Name, monoName)
	return nil
}

// substituteTypeInString 在类型字符串中替换类型参数为具体类型
// 例如：HashMap[K, V] + {"K": "int", "V": "string"} -> HashMap[int, string]
func (sg *StatementGeneratorImpl) substituteTypeInString(typeStr string, typeParamMap map[string]string) string {
	if typeStr == "" {
		return typeStr
	}

	result := typeStr
	// 按类型参数名长度从长到短排序，避免部分匹配问题（如Key被替换为intey）
	typeParams := make([]string, 0, len(typeParamMap))
	for typeParam := range typeParamMap {
		typeParams = append(typeParams, typeParam)
	}
	// 简单排序：按长度降序，然后按字母顺序
	for i := 0; i < len(typeParams)-1; i++ {
		for j := i + 1; j < len(typeParams); j++ {
			if len(typeParams[i]) < len(typeParams[j]) || 
			   (len(typeParams[i]) == len(typeParams[j]) && typeParams[i] > typeParams[j]) {
				typeParams[i], typeParams[j] = typeParams[j], typeParams[i]
			}
		}
	}

	// 替换类型参数
	for _, typeParam := range typeParams {
		concreteType := typeParamMap[typeParam]
		// 使用字符串替换，但需要确保是完整的类型参数名（不是部分匹配）
		// 简单实现：直接替换，实际应该使用更精确的匹配
		result = strings.ReplaceAll(result, typeParam, concreteType)
	}

	return result
}

// generateMonomorphizedName 生成单态化函数名
// 例如：bucket_index + {"K": "int", "V": "string"} -> bucket_index_int_string
func (sg *StatementGeneratorImpl) generateMonomorphizedName(originalName string, typeParamMap map[string]string, typeParamNames []string) string {
	name := originalName + "_"
	
	// 按照函数定义的类型参数顺序生成名称
	for _, typeParamName := range typeParamNames {
		if concreteType, ok := typeParamMap[typeParamName]; ok {
			// 清理类型名称（移除空格、特殊字符等）
			cleanType := strings.ReplaceAll(concreteType, " ", "")
			cleanType = strings.ReplaceAll(cleanType, "[", "_")
			cleanType = strings.ReplaceAll(cleanType, "]", "_")
			cleanType = strings.ReplaceAll(cleanType, ",", "_")
			name += cleanType + "_"
		}
	}
	
	// 移除末尾的下划线
	if len(name) > 0 && name[len(name)-1] == '_' {
		name = name[:len(name)-1]
	}
	
	return name
}

// loadStdlibFileIfExists 加载并解析 stdlib 文件（如果存在）
// 返回是否成功加载
func (sg *StatementGeneratorImpl) loadStdlibFileIfExists(irManager generation.IRModuleManager, filePath string) bool {
	// ✅ 修复：规范化路径，避免同一文件被加载多次
	normalizedPath, err := filepath.Abs(filePath)
	if err != nil {
		// 如果无法获取绝对路径，使用原始路径
		normalizedPath = filePath
	}
	normalizedPath = filepath.Clean(normalizedPath)
	
	// ✅ 检查是否已经加载过，避免重复加载（使用规范化路径）
	if sg.loadedStdlibFiles[normalizedPath] {
		fmt.Printf("DEBUG: loadStdlibFileIfExists - file already loaded: %s (normalized: %s)\n", filePath, normalizedPath)
		return true
	}
	
	// ✅ 检查是否正在加载，避免递归加载（使用规范化路径）
	if sg.loadingStdlibFiles[normalizedPath] {
		fmt.Printf("DEBUG: loadStdlibFileIfExists - file already loading (recursive call): %s (normalized: %s)\n", filePath, normalizedPath)
		return false // 返回 false，让调用者知道文件还没有加载完成
	}
	
	// 标记为正在加载（使用规范化路径）
	sg.loadingStdlibFiles[normalizedPath] = true
	defer func() {
		// 加载完成后，清除正在加载标记
		delete(sg.loadingStdlibFiles, normalizedPath)
	}()
	
	fmt.Printf("DEBUG: loadStdlibFileIfExists - attempting to load: %s\n", filePath)
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("DEBUG: loadStdlibFileIfExists - file does not exist: %s\n", filePath)
		return false
	}
	fmt.Printf("DEBUG: loadStdlibFileIfExists - file exists: %s\n", filePath)
	
	// 读取文件内容
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("DEBUG: Failed to read stdlib file %s: %v\n", filePath, err)
		return false
	}
	
	// 解析文件
	// 使用 ParserAggregate 以支持完整的语法特性（包括 match pattern）
	// 这样可以正确解析 match 语句中的 pattern（如 Ok(socket)）
	parserAggregate := parserPkg.NewParserAggregate()
	stdlibProgram, err := parserAggregate.ParseProgramWithTokens(string(content), filePath)
	if err != nil {
		fmt.Printf("DEBUG: Failed to parse stdlib file %s: %v\n", filePath, err)
		return false
	}
	fmt.Printf("DEBUG: Parsed stdlib file %s, got %d statements\n", filePath, len(stdlibProgram.Statements))
	
	// 生成 stdlib 文件的语句（只处理函数和方法定义，不处理 main 函数）
	processedCount := 0
	fmt.Printf("DEBUG: loadStdlibFileIfExists - Starting to process %d statements from %s\n", len(stdlibProgram.Statements), filePath)
	
	for i, stmt := range stdlibProgram.Statements {
		fmt.Printf("DEBUG: Processing stdlib statement %d/%d: %T\n", i+1, len(stdlibProgram.Statements), stmt)
		
		// 跳过 main 函数
		if funcDef, ok := stmt.(*entities.FuncDef); ok && funcDef.Name == "main" {
			fmt.Printf("DEBUG: Skipping main function in stdlib file %s\n", filePath)
			continue
		}
		
		// 检查是否是方法定义
		if methodDef, ok := stmt.(*entities.MethodDef); ok {
			fmt.Printf("DEBUG: Found MethodDef: %s.%s, ReceiverParams count=%d\n", 
				methodDef.Receiver, methodDef.Name, len(methodDef.ReceiverParams))
		} else {
			// 详细打印其他类型的语句
			switch s := stmt.(type) {
			case *entities.FuncDef:
				fmt.Printf("DEBUG: Found FuncDef: %s\n", s.Name)
			case *entities.StructDef:
				fmt.Printf("DEBUG: Found StructDef: %s\n", s.Name)
			default:
				fmt.Printf("DEBUG: Found statement type: %T\n", stmt)
			}
		}
		
		// 生成语句（这会保存泛型函数/方法定义）
		result, err := sg.GenerateStatement(irManager, stmt)
		if err != nil {
			fmt.Printf("DEBUG: ERROR - Failed to generate statement %d: %v\n", i, err)
			// 继续处理其他语句，不中断
		} else if result != nil && !result.Success {
			fmt.Printf("DEBUG: ERROR - Statement %d generation failed: %v\n", i, result.Error)
		} else {
			fmt.Printf("DEBUG: Successfully generated statement %d\n", i)
			processedCount++
		}
	}
	
	fmt.Printf("DEBUG: Successfully loaded stdlib file: %s (processed %d statements)\n", filePath, processedCount)
	// 标记为已加载（使用规范化路径）
	sg.loadedStdlibFiles[normalizedPath] = true
	return true
}

