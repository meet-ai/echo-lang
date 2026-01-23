package impl

import (
	"fmt"
	"strings"

	"echo/internal/modules/backend/domain/services/generation"
	"echo/internal/modules/frontend/domain/entities"

	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	llvalue "github.com/llir/llvm/ir/value"
)

// ExpressionEvaluatorImpl 表达式求值器实现
type ExpressionEvaluatorImpl struct {
	symbolManager   generation.SymbolManager
	typeMapper      generation.TypeMapper
	irModuleManager generation.IRModuleManager

	// 用于生成唯一的二进制操作变量名
	binaryOpCounter int
}

// NewExpressionEvaluatorImpl 创建表达式求值器实现
func NewExpressionEvaluatorImpl(
	symbolMgr generation.SymbolManager,
	typeMapper generation.TypeMapper,
	irManager generation.IRModuleManager,
) *ExpressionEvaluatorImpl {
	return &ExpressionEvaluatorImpl{
		symbolManager:   symbolMgr,
		typeMapper:      typeMapper,
		irModuleManager: irManager,
		binaryOpCounter: 0,
	}
}

// Evaluate 求值表达式
func (ee *ExpressionEvaluatorImpl) Evaluate(irManager generation.IRModuleManager, expr entities.Expr) (interface{}, error) {
	switch e := expr.(type) {
	case *entities.IntLiteral:
		return ee.EvaluateIntLiteral(irManager, e)
	case *entities.StringLiteral:
		return ee.EvaluateStringLiteral(irManager, e)
	case *entities.Identifier:
		return ee.EvaluateIdentifier(irManager, e)
	case *entities.BinaryExpr:
		return ee.EvaluateBinaryExpr(irManager, e)
	case *entities.FuncCall:
		return ee.EvaluateFuncCall(irManager, e)
	case *entities.ArrayLiteral:
		return ee.EvaluateArrayLiteral(irManager, e)
	case *entities.IndexExpr:
		return ee.EvaluateIndexExpr(irManager, e)
	case *entities.LenExpr:
		return ee.EvaluateLenExpr(irManager, e)
	case *entities.SliceExpr:
		return ee.EvaluateSliceExpr(irManager, e)
	case *entities.ArrayMethodCallExpr:
		return ee.EvaluateArrayMethodCallExpr(irManager, e)
	case *entities.MethodCallExpr:
		return ee.EvaluateMethodCallExpr(irManager, e)
	case *entities.BoolLiteral:
		return ee.EvaluateBoolLiteral(irManager, e)
	case *entities.ResultExpr:
		return ee.EvaluateResultExpr(irManager, e)
	case *entities.OkLiteral:
		return ee.EvaluateOkLiteral(irManager, e)
	case *entities.ErrLiteral:
		return ee.EvaluateErrLiteral(irManager, e)
	case *entities.ErrorPropagation:
		return ee.EvaluateErrorPropagation(irManager, e)
	case *entities.SpawnExpr:
		return ee.EvaluateSpawnExpr(irManager, e)
	case *entities.AwaitExpr:
		return ee.EvaluateAwaitExpr(irManager, e)
	case *entities.ChanType:
		return ee.EvaluateChanType(irManager, e)
	case *entities.SendExpr:
		return ee.EvaluateSendExpr(irManager, e)
	case *entities.ReceiveExpr:
		return ee.EvaluateReceiveExpr(irManager, e)
	default:
		return "", fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// EvaluateIntLiteral 求值整数字面量
func (ee *ExpressionEvaluatorImpl) EvaluateIntLiteral(irManager generation.IRModuleManager, expr *entities.IntLiteral) (interface{}, error) {
	// 对于常量，直接返回常量值
	return constant.NewInt(types.I32, int64(expr.Value)), nil
}

// EvaluateStringLiteral 求值字符串字面量
func (ee *ExpressionEvaluatorImpl) EvaluateStringLiteral(irManager generation.IRModuleManager, expr *entities.StringLiteral) (interface{}, error) {
	// 添加字符串常量到全局变量
	strGlobal, err := irManager.AddStringConstant(expr.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to add string constant: %w", err)
	}

	// 全局字符串变量的类型是 [N x i8]*
	// 我们需要将其转换为 i8* 类型
	// 对于字符串常量，最简单且正确的方法是使用 bitcast
	// 因为 [N x i8]* 和 i8* 在内存布局上是兼容的（都是指向连续内存的指针）
	targetType := types.NewPointer(types.I8)
	bitcastInst, err := irManager.CreateBitCast(strGlobal, targetType, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create bitcast for string: %w", err)
	}

	return bitcastInst, nil
}

// EvaluateIdentifier 求值标识符
func (ee *ExpressionEvaluatorImpl) EvaluateIdentifier(irManager generation.IRModuleManager, expr *entities.Identifier) (interface{}, error) {
	symbol, err := ee.symbolManager.LookupSymbol(expr.Name)
	if err != nil {
		return nil, fmt.Errorf("undefined symbol %s: %w", expr.Name, err)
	}

	// symbol.Value 应该是alloca指令的结果（指针）
	allocaPtr, ok := symbol.Value.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("symbol value is not a valid LLVM value: %T", symbol.Value)
	}

	// 根据Echo类型映射到LLVM类型（支持复杂类型如Future[T]）
	llvmTypeInterface, err := ee.typeMapper.MapType(symbol.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to map type %s: %w", symbol.Type, err)
	}

	// 类型断言
	llvmType, ok := llvmTypeInterface.(types.Type)
	if !ok {
		return nil, fmt.Errorf("mapped type %v is not a valid LLVM type", llvmTypeInterface)
	}

	// 对于标识符求值，我们通常需要加载它的值
	loadResult, err := irManager.CreateLoad(llvmType, allocaPtr, expr.Name+"_load")
	if err != nil {
		return nil, fmt.Errorf("failed to create load instruction: %w", err)
	}

	return loadResult, nil
}

// EvaluateBinaryExpr 求值二元表达式
func (ee *ExpressionEvaluatorImpl) EvaluateBinaryExpr(irManager generation.IRModuleManager, expr *entities.BinaryExpr) (interface{}, error) {
	// 求值左侧表达式
	leftResult, err := ee.Evaluate(irManager, expr.Left)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate left operand: %w", err)
	}

	// 求值右侧表达式
	rightResult, err := ee.Evaluate(irManager, expr.Right)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate right operand: %w", err)
	}

	// 使用irManager创建二元运算指令，为每个操作生成唯一的名称
	ee.binaryOpCounter++
	uniqueName := fmt.Sprintf("binop_%d", ee.binaryOpCounter)
	result, err := irManager.CreateBinaryOp(expr.Op, leftResult, rightResult, uniqueName)
	if err != nil {
		return nil, fmt.Errorf("failed to create binary operation: %w", err)
	}

	return result, nil
}

// extractValueFromTypedLiteral 从 "type value" 格式中提取值部分
func (ee *ExpressionEvaluatorImpl) extractValueFromTypedLiteral(typedLiteral string) string {
	// 简单的实现：假设格式是 "i32 value" 或 "value"
	parts := strings.Split(typedLiteral, " ")
	if len(parts) >= 2 {
		return strings.Join(parts[1:], " ")
	}
	return typedLiteral
}

// EvaluateFuncCall 求值函数调用
func (ee *ExpressionEvaluatorImpl) EvaluateFuncCall(irManager generation.IRModuleManager, expr *entities.FuncCall) (interface{}, error) {
	// 检查是否是async函数调用
	symbol, err := ee.symbolManager.LookupSymbol(expr.Name)
	if err == nil && symbol.IsAsync {
		// async函数调用：应该返回Future，不直接执行
		// 简化实现：直接调用async函数，它会返回Future
		callResult, err := irManager.CreateCall(expr.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to create async function call: %w", err)
		}
		return callResult, nil
	}

	// 求值所有参数
	args := make([]interface{}, len(expr.Args))
	for i, arg := range expr.Args {
		argResult, err := ee.Evaluate(irManager, arg)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate argument %d: %w", i, err)
		}
		args[i] = argResult
	}

	// 使用irManager创建函数调用
	// 这里假设函数已经声明（在运行时函数或用户定义函数中）
	callResult, err := irManager.CreateCall(expr.Name, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create function call: %w", err)
	}

	return callResult, nil
}

// EvaluateArrayLiteral 求值数组字面量
func (ee *ExpressionEvaluatorImpl) EvaluateArrayLiteral(irManager generation.IRModuleManager, expr *entities.ArrayLiteral) (interface{}, error) {
	// 获取数组长度
	arrayLen := len(expr.Elements)

	// 创建数组类型（简化：假设都是i32类型）
	elemType := types.I32
	arrayType := types.NewArray(uint64(arrayLen), elemType)

	// 创建数组常量值
	var elements []constant.Constant
	for _, elem := range expr.Elements {
		// 简化：假设所有元素都是整数字面量
		if intLit, ok := elem.(*entities.IntLiteral); ok {
			elements = append(elements, constant.NewInt(elemType, int64(intLit.Value)))
		} else {
			// 对于其他类型，创建零值
			elements = append(elements, constant.NewInt(elemType, 0))
		}
	}

	arrayConstant := constant.NewArray(arrayType, elements...)

	// 在全局作用域创建数组常量
	err := irManager.AddGlobalVariable(fmt.Sprintf("array_%d", arrayLen), arrayConstant)
	if err != nil {
		return nil, fmt.Errorf("failed to create global array variable: %w", err)
	}

	// 返回数组常量的地址（全局变量）
	return arrayConstant, nil
}

// GenerateBinaryExpression 生成二元表达式
func (ee *ExpressionEvaluatorImpl) GenerateBinaryExpression(irManager generation.IRModuleManager, expr *entities.BinaryExpr) (string, error) {
	result, err := ee.EvaluateBinaryExpr(irManager, expr)
	if err != nil {
		return "", err
	}
	return result.(string), nil
}

// GenerateFuncCall 生成函数调用
func (ee *ExpressionEvaluatorImpl) GenerateFuncCall(irManager generation.IRModuleManager, expr *entities.FuncCall) (string, error) {
	result, err := ee.EvaluateFuncCall(irManager, expr)
	if err != nil {
		return "", err
	}
	return result.(string), nil
}

// GenerateArrayLiteral 生成数组字面量
func (ee *ExpressionEvaluatorImpl) GenerateArrayLiteral(irManager generation.IRModuleManager, expr *entities.ArrayLiteral) (string, error) {
	result, err := ee.EvaluateArrayLiteral(irManager, expr)
	if err != nil {
		return "", err
	}
	return result.(string), nil
}

// EvaluateMatchExpr 求值模式匹配表达式
func (ee *ExpressionEvaluatorImpl) EvaluateMatchExpr(irManager generation.IRModuleManager, expr *entities.MatchExpr) (interface{}, error) {
	// 简化实现：只返回第一个case的第一个语句的结果
	if len(expr.Cases) == 0 {
		return "", fmt.Errorf("match expression must have at least one case")
	}

	firstCase := expr.Cases[0]
	if len(firstCase.Body) == 0 {
		return "", fmt.Errorf("match case must have at least one statement")
	}

	// 如果第一个语句是表达式语句，返回其结果
	if exprStmt, ok := firstCase.Body[0].(*entities.ExprStmt); ok {
		return ee.Evaluate(irManager, exprStmt.Expression)
	}

	return "", fmt.Errorf("match case first statement must be an expression")
}

// EvaluateIndexExpr 求值索引访问表达式
func (ee *ExpressionEvaluatorImpl) EvaluateIndexExpr(irManager generation.IRModuleManager, expr *entities.IndexExpr) (interface{}, error) {
	// 求值数组表达式
	arrayValue, err := ee.Evaluate(irManager, expr.Array)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate array expression: %w", err)
	}

	// 求值索引表达式
	indexValue, err := ee.Evaluate(irManager, expr.Index)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate index expression: %w", err)
	}

	// 确定数组元素类型
	var elementType types.Type
	if ident, ok := expr.Array.(*entities.Identifier); ok {
		// 从符号表获取数组类型信息
		symbol, err := ee.symbolManager.LookupSymbol(ident.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup array symbol %s: %w", ident.Name, err)
		}

		// 从数组类型字符串中提取元素类型
		elementType, err = ee.inferElementTypeFromArrayType(symbol.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to infer element type from array type %s: %w", symbol.Type, err)
		}
	} else {
		// 对于其他类型的数组表达式，默认使用i32
		elementType = types.I32
	}

	// 生成GetElementPtr指令进行索引访问
	elementPtr, err := irManager.CreateGetElementPtr(elementType, arrayValue, indexValue)
	if err != nil {
		return nil, fmt.Errorf("failed to create GEP for index access: %w", err)
	}

	// 加载元素值
	elementValue, err := irManager.CreateLoad(elementType, elementPtr, "element")
	if err != nil {
		return nil, fmt.Errorf("failed to load element value: %w", err)
	}

	return elementValue, nil
}

// inferElementTypeFromArrayType 从数组类型字符串推断元素类型
func (ee *ExpressionEvaluatorImpl) inferElementTypeFromArrayType(arrayType string) (types.Type, error) {
	// 数组类型格式如 "[int]", "[string]" 等
	if !strings.HasPrefix(arrayType, "[") || !strings.HasSuffix(arrayType, "]") {
		return nil, fmt.Errorf("invalid array type format: %s", arrayType)
	}

	elementTypeStr := arrayType[1 : len(arrayType)-1]
	switch elementTypeStr {
	case "int":
		return types.I32, nil
	case "bool":
		return types.I1, nil
	case "string":
		return types.NewPointer(types.I8), nil
	case "float":
		return types.Float, nil
	default:
		return nil, fmt.Errorf("unsupported element type: %s", elementTypeStr)
	}
}

// EvaluateLenExpr 求值长度获取表达式
func (ee *ExpressionEvaluatorImpl) EvaluateLenExpr(irManager generation.IRModuleManager, expr *entities.LenExpr) (interface{}, error) {
	// 求值数组表达式
	arrayValue, err := ee.Evaluate(irManager, expr.Array)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate array expression: %w", err)
	}
	_ = arrayValue // 标记为已使用，避免编译警告

	// 处理不同类型的数组表达式
	switch arr := expr.Array.(type) {
	case *entities.Identifier:
		// 对于变量标识符，从符号表获取数组长度信息
		symbol, err := ee.symbolManager.LookupSymbol(arr.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup array symbol %s: %w", arr.Name, err)
		}

		// 检查是否是数组类型
		if strings.HasPrefix(symbol.Type, "[") {
			// 从符号的Value字段获取存储的长度信息
			if length, ok := symbol.Value.(int); ok && length > 0 {
				return constant.NewInt(types.I32, int64(length)), nil
			}
		}
		return nil, fmt.Errorf("cannot determine length for non-array type: %s", symbol.Type)

	case *entities.ArrayLiteral:
		// 对于数组字面量，直接返回元素数量
		arrayLen := len(arr.Elements)
		return constant.NewInt(types.I32, int64(arrayLen)), nil

	default:
		return nil, fmt.Errorf("len() function requires array or slice operand, got %T", expr.Array)
	}
}

// EvaluateSliceExpr 求值切片表达式
func (ee *ExpressionEvaluatorImpl) EvaluateSliceExpr(irManager generation.IRModuleManager, expr *entities.SliceExpr) (interface{}, error) {
	// 求值数组表达式
	arrayValue, err := ee.Evaluate(irManager, expr.Array)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate array expression: %w", err)
	}
	_ = arrayValue // 标记为已使用，避免编译警告

	// 求值start表达式
	var startValue interface{}
	if expr.Start != nil {
		startValue, err = ee.Evaluate(irManager, expr.Start)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate slice start: %w", err)
		}
	} else {
		// 如果没有指定start，默认从0开始
		startValue = constant.NewInt(types.I32, 0)
	}

	// 求值end表达式
	var endValue interface{}
	if expr.End != nil {
		endValue, err = ee.Evaluate(irManager, expr.End)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate slice end: %w", err)
		}
	} else {
		// 如果没有指定end，需要计算数组长度
		// 这里简化处理，假设数组长度已知
		if ident, ok := expr.Array.(*entities.Identifier); ok {
			symbol, err := ee.symbolManager.LookupSymbol(ident.Name)
			if err == nil {
				if length, ok := symbol.Value.(int); ok && length > 0 {
					endValue = constant.NewInt(types.I32, int64(length))
				} else {
					endValue = constant.NewInt(types.I32, 10) // 默认长度
				}
			} else {
				endValue = constant.NewInt(types.I32, 10) // 默认长度
			}
		} else {
			endValue = constant.NewInt(types.I32, 10) // 默认长度
		}
	}

	// 计算切片长度：end - start
	sliceLenValue, err := irManager.CreateBinaryOp("sub", endValue, startValue, "slice_len")
	if err != nil {
		return nil, fmt.Errorf("failed to calculate slice length: %w", err)
	}
	_ = sliceLenValue // 标记为已使用，避免编译警告

	// 获取切片的起始指针
	elementType := types.I32 // 简化：假设元素类型为i32
	sliceStartPtr, err := irManager.CreateGetElementPtr(elementType, arrayValue, startValue)
	if err != nil {
		return nil, fmt.Errorf("failed to create slice start pointer: %w", err)
	}

	// 这里可以根据需要使用sliceLenValue来创建完整的切片结构
	// 目前简化实现，只返回起始指针

	// 在实际实现中，这里应该创建一个切片结构体，包含：
	// - 指向数据的指针
	// - 长度
	// - 容量（如果支持）
	// 但为了简化，这里返回起始指针

	return sliceStartPtr, nil
}

// EvaluateMethodCallExpr 求值方法调用表达式
func (ee *ExpressionEvaluatorImpl) EvaluateMethodCallExpr(irManager generation.IRModuleManager, expr *entities.MethodCallExpr) (interface{}, error) {
	// 求值接收者
	receiverValue, err := ee.Evaluate(irManager, expr.Receiver)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate method receiver: %w", err)
	}

	// 求值参数
	var argValues []interface{}
	for i, arg := range expr.Args {
		argValue, err := ee.Evaluate(irManager, arg)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate method argument %d: %w", i, err)
		}
		argValues = append(argValues, argValue)
	}

	// 构造方法调用名（接收者类型_方法名）
	receiverType := "unknown" // 简化实现，实际应该从接收者推断类型
	if ident, ok := expr.Receiver.(*entities.Identifier); ok {
		// 从符号表获取接收者类型
		if symbol, err := ee.symbolManager.LookupSymbol(ident.Name); err == nil {
			receiverType = symbol.Type // symbol.Type 已经是string类型
		}
	}

	methodName := fmt.Sprintf("%s_%s", receiverType, expr.MethodName)

	// 调用方法（在LLVM IR中，方法调用被转换为普通函数调用）
	result, err := irManager.CreateCall(methodName, append([]interface{}{receiverValue}, argValues...))
	if err != nil {
		return nil, fmt.Errorf("failed to create method call to %s: %w", methodName, err)
	}

	return result, nil
}

// EvaluateBoolLiteral 求值布尔字面量
func (ee *ExpressionEvaluatorImpl) EvaluateBoolLiteral(irManager generation.IRModuleManager, expr *entities.BoolLiteral) (interface{}, error) {
	// 在LLVM IR中，布尔值使用i1类型
	var value int64
	if expr.Value {
		value = 1
	} else {
		value = 0
	}
	return constant.NewInt(types.I1, value), nil
}

// EvaluateResultExpr 求值Result表达式
func (ee *ExpressionEvaluatorImpl) EvaluateResultExpr(irManager generation.IRModuleManager, expr *entities.ResultExpr) (interface{}, error) {
	// Result[T, E] 是一种特殊的泛型类型，目前返回一个占位符
	// 实际实现需要运行时库支持
	fmt.Printf("DEBUG: Result type evaluation - OkType: %s, ErrType: %s\n", expr.OkType, expr.ErrType)
	return constant.NewInt(types.I32, 0), nil
}

// EvaluateOkLiteral 求值Ok字面量
func (ee *ExpressionEvaluatorImpl) EvaluateOkLiteral(irManager generation.IRModuleManager, expr *entities.OkLiteral) (interface{}, error) {
	// 求值内部值，目前返回占位符
	// 实际实现需要Result类型的运行时支持
	fmt.Printf("DEBUG: Ok literal evaluation\n")
	return constant.NewInt(types.I32, 0), nil
}

// EvaluateErrLiteral 求值Err字面量
func (ee *ExpressionEvaluatorImpl) EvaluateErrLiteral(irManager generation.IRModuleManager, expr *entities.ErrLiteral) (interface{}, error) {
	// 求值内部错误值，目前返回占位符
	// 实际实现需要Result类型的运行时支持
	fmt.Printf("DEBUG: Err literal evaluation\n")
	return constant.NewInt(types.I32, 0), nil
}

// EvaluateErrorPropagation 求值错误传播操作符
func (ee *ExpressionEvaluatorImpl) EvaluateErrorPropagation(irManager generation.IRModuleManager, expr *entities.ErrorPropagation) (interface{}, error) {
	// 求值内部表达式，然后传播错误
	// 实际实现需要Result类型的运行时支持和控制流
	fmt.Printf("DEBUG: Error propagation evaluation\n")
	return constant.NewInt(types.I32, 0), nil
}

// EvaluateArrayMethodCallExpr 求值数组方法调用表达式
func (ee *ExpressionEvaluatorImpl) EvaluateArrayMethodCallExpr(irManager generation.IRModuleManager, expr *entities.ArrayMethodCallExpr) (interface{}, error) {
	// 目前数组方法需要运行时库支持，这里先实现基础框架
	// 未来需要实现完整的运行时库

	switch expr.Method {
	case "push":
		// array.push(element) - 添加元素到数组末尾
		if len(expr.Args) != 1 {
			return nil, fmt.Errorf("push method expects 1 argument, got %d", len(expr.Args))
		}
		// TODO: 实现push逻辑，需要运行时库
		fmt.Printf("DEBUG: Array push operation on array with 1 argument\n")

	case "pop":
		// array.pop() -> element - 移除并返回最后一个元素
		if len(expr.Args) != 0 {
			return nil, fmt.Errorf("pop method expects 0 arguments, got %d", len(expr.Args))
		}
		// TODO: 实现pop逻辑，需要运行时库
		fmt.Printf("DEBUG: Array pop operation on array\n")

	case "insert":
		// array.insert(index, element) - 在指定位置插入元素
		if len(expr.Args) != 2 {
			return nil, fmt.Errorf("insert method expects 2 arguments, got %d", len(expr.Args))
		}
		// TODO: 实现insert逻辑，需要运行时库
		fmt.Printf("DEBUG: Array insert operation on array with 2 arguments\n")

	case "remove":
		// array.remove(index) -> element - 移除指定位置的元素
		if len(expr.Args) != 1 {
			return nil, fmt.Errorf("remove method expects 1 argument, got %d", len(expr.Args))
		}
		// TODO: 实现remove逻辑，需要运行时库
		fmt.Printf("DEBUG: Array remove operation on array with 1 argument\n")

	default:
		return nil, fmt.Errorf("unsupported array method: %s", expr.Method)
	}

	// 临时返回一个常量值，实际实现需要运行时库
	return constant.NewInt(types.I32, 0), nil
}

// EvaluateSpawnExpr 求值spawn表达式 - 创建新协程
func (ee *ExpressionEvaluatorImpl) EvaluateSpawnExpr(irManager generation.IRModuleManager, expr *entities.SpawnExpr) (interface{}, error) {

	// 检查是否是函数标识符
	ident, ok := expr.Function.(*entities.Identifier)
	if !ok {
		return nil, fmt.Errorf("spawn currently only supports identifier functions")
	}

	funcName := ident.Name

	// 获取目标函数
	targetFunc, exists := irManager.GetFunction(funcName)
	if !exists {
		return nil, fmt.Errorf("target function %s not declared", funcName)
	}


	// 创建Future
	futureNewFunc, exists := irManager.GetExternalFunction("future_new")
	if !exists {
		return nil, fmt.Errorf("future_new function not declared")
	}
	futureValue, err := irManager.CreateCall(futureNewFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to create future: %w", err)
	}


	// spawn协程 - 传递目标函数、参数和future
	spawnFunc, spawnExists := irManager.GetExternalFunction("coroutine_spawn")
	if spawnExists {
		// 求值所有参数
		var args []interface{}
		for _, argExpr := range expr.Args {
			argValue, err := ee.Evaluate(irManager, argExpr)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate spawn argument: %w", err)
			}
			args = append(args, argValue)
		}

		// 构建spawn参数
		spawnArgs := []interface{}{
			targetFunc,                             // target function
			constant.NewInt(types.I32, int64(len(args))), // arg_count
			constant.NewNull(types.NewPointer(types.I8)), // args (暂时使用null，后续需要传递实际参数)
			futureValue,                            // future
		}

		_, err = irManager.CreateCall(spawnFunc, spawnArgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to create spawn call: %w", err)
		}

	} else {
	}


	// spawn表达式返回Future
	return futureValue, nil
}

// EvaluateAwaitExpr 求值await表达式 - 等待异步操作并启动执行器
func (ee *ExpressionEvaluatorImpl) EvaluateAwaitExpr(irManager generation.IRModuleManager, expr *entities.AwaitExpr) (interface{}, error) {
	// 1. 求值异步表达式 - 获取Future
	futureValue, err := ee.Evaluate(irManager, expr.Expression)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate await expression: %v", err)
	}

	// spawn逻辑移到了EvaluateSpawnExpr中，这里不再需要

	// 3. 调用await运行时函数等待结果
	// 生成: coroutine_await(future)

	awaitFunc, exists := irManager.GetExternalFunction("coroutine_await")
	if !exists {
		return nil, fmt.Errorf("coroutine_await function not declared")
	}

	// 生成函数调用
	awaitCall, err := irManager.CreateCall(awaitFunc, futureValue)
	if err != nil {
		return nil, fmt.Errorf("failed to create await call: %v", err)
	}

	// 暂时移除run_scheduler调用，用于调试
	// TODO: 重新启用run_scheduler

	return awaitCall, nil
}

// EvaluateChanType 求值通道类型 (chan T)
func (ee *ExpressionEvaluatorImpl) EvaluateChanType(irManager generation.IRModuleManager, expr *entities.ChanType) (interface{}, error) {
	// 通道类型求值 - 创建通道实例
	// 调用运行时库函数创建通道

	createChanFunc, exists := irManager.GetExternalFunction("channel_create")
	if !exists {
		return nil, fmt.Errorf("channel_create function not declared")
	}

	// 调用channel_create() - 目前简化实现，不传递类型信息
	chanCall, err := irManager.CreateCall(createChanFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel call: %v", err)
	}

	return chanCall, nil
}

// EvaluateSendExpr 求值发送表达式 (channel <- value)
func (ee *ExpressionEvaluatorImpl) EvaluateSendExpr(irManager generation.IRModuleManager, expr *entities.SendExpr) (interface{}, error) {
	// 1. 求值通道表达式
	chanValue, err := ee.Evaluate(irManager, expr.Channel)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate channel in send expression: %v", err)
	}

	// 2. 求值要发送的值
	value, err := ee.Evaluate(irManager, expr.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate value in send expression: %v", err)
	}

	// 3. 调用通道发送运行时函数
	sendFunc, exists := irManager.GetExternalFunction("channel_send")
	if !exists {
		return nil, fmt.Errorf("channel_send function not declared")
	}

	// 生成channel_send(channel, value)调用
	sendCall, err := irManager.CreateCall(sendFunc, chanValue, value)
	if err != nil {
		return nil, fmt.Errorf("failed to create send call: %v", err)
	}

	return sendCall, nil
}

// EvaluateReceiveExpr 求值接收表达式 (<- channel)
func (ee *ExpressionEvaluatorImpl) EvaluateReceiveExpr(irManager generation.IRModuleManager, expr *entities.ReceiveExpr) (interface{}, error) {
	// 1. 求值通道表达式
	chanValue, err := ee.Evaluate(irManager, expr.Channel)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate channel in receive expression: %v", err)
	}

	// 2. 调用通道接收运行时函数
	receiveFunc, exists := irManager.GetExternalFunction("channel_receive")
	if !exists {
		return nil, fmt.Errorf("channel_receive function not declared")
	}

	// 生成channel_receive(channel)调用
	receiveCall, err := irManager.CreateCall(receiveFunc, chanValue)
	if err != nil {
		return nil, fmt.Errorf("failed to create receive call: %v", err)
	}

	return receiveCall, nil
}
