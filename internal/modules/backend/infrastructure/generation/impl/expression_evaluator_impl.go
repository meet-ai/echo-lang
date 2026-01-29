package impl

import (
	"fmt"
	"strings"

	"echo/internal/modules/backend/domain/services/generation"
	"echo/internal/modules/frontend/domain/entities"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	llvalue "github.com/llir/llvm/ir/value"
)

// ExpressionEvaluatorImpl 表达式求值器实现
type ExpressionEvaluatorImpl struct {
	symbolManager      generation.SymbolManager
	typeMapper         generation.TypeMapper
	irModuleManager    generation.IRModuleManager
	statementGenerator generation.StatementGenerator // 用于延迟编译泛型函数

	// 用于生成唯一的二进制操作变量名
	binaryOpCounter int
	
	// structDefinitions 结构体定义映射：结构体名 -> 结构体定义
	// 用于计算结构体类型大小
	structDefinitions map[string]*entities.StructDef
	
	// generatedHashFunctions 已生成的hash函数映射：结构体名 -> 函数指针
	// 用于避免重复生成
	generatedHashFunctions map[string]*ir.Func
	
	// generatedEqualsFunctions 已生成的equals函数映射：结构体名 -> 函数指针
	// 用于避免重复生成
	generatedEqualsFunctions map[string]*ir.Func
}

// NewExpressionEvaluatorImpl 创建表达式求值器实现
func NewExpressionEvaluatorImpl(
	symbolMgr generation.SymbolManager,
	typeMapper generation.TypeMapper,
	irManager generation.IRModuleManager,
) *ExpressionEvaluatorImpl {
	return &ExpressionEvaluatorImpl{
		symbolManager:     symbolMgr,
		typeMapper:        typeMapper,
		irModuleManager:   irManager,
		statementGenerator: nil, // 稍后通过SetStatementGenerator设置
		binaryOpCounter:   0,
		structDefinitions: make(map[string]*entities.StructDef),
		generatedHashFunctions: make(map[string]*ir.Func),
		generatedEqualsFunctions: make(map[string]*ir.Func),
	}
}

// SetStatementGenerator 设置语句生成器（用于延迟编译）
func (ee *ExpressionEvaluatorImpl) SetStatementGenerator(stmtGen generation.StatementGenerator) {
	ee.statementGenerator = stmtGen
}

// SetStructDefinitions 设置结构体定义映射
// 在 GenerateProgram 开始时调用，收集所有结构体定义
func (ee *ExpressionEvaluatorImpl) SetStructDefinitions(structDefs map[string]*entities.StructDef) {
	// 如果 structDefinitions 为空，直接赋值
	// 否则，合并新的结构体定义（避免覆盖已有的定义）
	if len(ee.structDefinitions) == 0 {
		ee.structDefinitions = structDefs
	} else {
		// 合并：新定义覆盖旧定义
		for k, v := range structDefs {
			ee.structDefinitions[k] = v
		}
	}
}

// GetStructDefinition 获取结构体定义
// 用于从其他组件（如 StatementGenerator）访问结构体定义
func (ee *ExpressionEvaluatorImpl) GetStructDefinition(structName string) (*entities.StructDef, bool) {
	def, exists := ee.structDefinitions[structName]
	return def, exists
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
	case *entities.SizeOfExpr:
		return ee.EvaluateSizeOfExpr(irManager, e)
	case *entities.TypeCastExpr:
		return ee.EvaluateTypeCastExpr(irManager, e)
	case *entities.FunctionPointerExpr:
		return ee.EvaluateFunctionPointerExpr(irManager, e)
	case *entities.AddressOfExpr:
		return ee.EvaluateAddressOfExpr(irManager, e)
	case *entities.DereferenceExpr:
		return ee.EvaluateDereferenceExpr(irManager, e)
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
	case *entities.StructAccess:
		return ee.EvaluateStructAccess(irManager, e)
	case *entities.StructLiteral:
		return ee.EvaluateStructLiteral(irManager, e)
	case *entities.FloatLiteral:
		return ee.EvaluateFloatLiteral(irManager, e)
	case *entities.NoneLiteral:
		return ee.EvaluateNoneLiteral(irManager, e)
	case *entities.NamespaceAccessExpr:
		return ee.EvaluateNamespaceAccessExpr(irManager, e)
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
	// 我们需要将其转换为 i8* 类型（C字符串指针）
	// 然后调用 runtime_char_ptr_to_string 转换为 string_t*（Echo字符串类型）
	targetType := types.NewPointer(types.I8)
	charPtr, err := irManager.CreateBitCast(strGlobal, targetType, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create bitcast for string: %w", err)
	}
	
	charPtrValue, ok := charPtr.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("CreateBitCast returned invalid type: %T", charPtr)
	}
	
	// ✅ 修复：调用 runtime_char_ptr_to_string 将 C 字符串转换为 string_t*
	// 这样结构体字段中存储的就是 string_t*，而不是字符串字面量的指针
	// 获取 string_t 结构体类型定义
	// string_t 结构体：{ i8*, i64, i64 } (data, length, capacity)
	i8PtrType := types.NewPointer(types.I8)
	i64Type := types.I64
	stringStructType := types.NewStruct(i8PtrType, i64Type, i64Type) // { i8*, i64, i64 }
	
	// 在堆上分配 string_t 结构体（使用malloc，因为需要持久化）
	mallocFunc, exists := irManager.GetExternalFunction("malloc")
	var mallocFuncLLVM *ir.Func
	if !exists {
		// 声明malloc函数：i8* malloc(i64 size)
		mallocFuncLLVM = irManager.(*IRModuleManagerImpl).module.NewFunc("malloc",
			types.NewPointer(types.I8), // 返回i8*
			ir.NewParam("size", types.I64), // 参数：i64 size
		)
		irManager.(*IRModuleManagerImpl).externalFunctions["malloc"] = mallocFuncLLVM
	} else {
		var ok bool
		mallocFuncLLVM, ok = mallocFunc.(*ir.Func)
		if !ok {
			return nil, fmt.Errorf("malloc function is not *ir.Func: %T", mallocFunc)
		}
	}
	
	// 计算string_t结构体大小（16字节，考虑对齐）
	structSize := int64(24) // { i8* (8) + i64 (8) + i64 (8) } = 24字节（考虑对齐）
	structSizeConst := constant.NewInt(types.I64, structSize)
	mallocResult, err := irManager.CreateCall(mallocFuncLLVM, structSizeConst)
	if err != nil {
		return nil, fmt.Errorf("failed to call malloc: %w", err)
	}
	
	mallocResultValue, ok := mallocResult.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("CreateCall returned invalid type: %T", mallocResult)
	}
	
	// 将malloc返回的i8*转换为string_t*（结构体指针类型）
	stringStructPtrType := types.NewPointer(stringStructType)
	stringStructPtr, err := irManager.CreateBitCast(mallocResultValue, stringStructPtrType, "string_t_heap_ptr")
	if err != nil {
		return nil, fmt.Errorf("failed to bitcast malloc result to string_t*: %w", err)
	}
	
	stringStructPtrValue, ok := stringStructPtr.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("CreateBitCast returned invalid type: %T", stringStructPtr)
	}
	
	// 调用 runtime_char_ptr_to_string 将 C 字符串转换为 string_t
	charPtrToStrFunc, exists := irManager.GetExternalFunction("runtime_char_ptr_to_string")
	if !exists {
		return nil, fmt.Errorf("runtime_char_ptr_to_string function not declared")
	}
	
	charPtrToStrFuncLLVM, ok := charPtrToStrFunc.(*ir.Func)
	if !ok {
		return nil, fmt.Errorf("runtime_char_ptr_to_string is not *ir.Func: %T", charPtrToStrFunc)
	}
	
	// 调用函数获取 string_t 结构体（值类型）
	callResult, err := irManager.CreateCall(charPtrToStrFuncLLVM, charPtrValue)
	if err != nil {
		return nil, fmt.Errorf("failed to call runtime_char_ptr_to_string: %w", err)
	}
	
	callResultValue, ok := callResult.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("CreateCall returned invalid type: %T", callResult)
	}
	
	// 存储返回值到堆上分配的内存
	if err := irManager.CreateStore(callResultValue, stringStructPtrValue); err != nil {
		return nil, fmt.Errorf("failed to store string_t result to heap: %w", err)
	}
	
	// 返回指向 string_t 结构体的指针（转换为 *i8）
	resultPtr, err := irManager.CreateBitCast(stringStructPtrValue, targetType, "string_ptr")
	if err != nil {
		return nil, fmt.Errorf("failed to cast string_t pointer to string type: %w", err)
	}
	
	return resultPtr, nil
}

// EvaluateIdentifier 求值标识符
func (ee *ExpressionEvaluatorImpl) EvaluateIdentifier(irManager generation.IRModuleManager, expr *entities.Identifier) (interface{}, error) {
	currentBlock := irManager.GetCurrentBasicBlock()
	fmt.Printf("DEBUG: EvaluateIdentifier - evaluating %s, current block: %v\n", expr.Name, currentBlock)
	symbol, err := ee.symbolManager.LookupSymbol(expr.Name)
	if err != nil {
		fmt.Printf("DEBUG: EvaluateIdentifier - undefined symbol %s: %v\n", expr.Name, err)
		return nil, fmt.Errorf("undefined symbol %s: %w", expr.Name, err)
	}
	fmt.Printf("DEBUG: EvaluateIdentifier - found symbol %s, type: %s, value type: %T, value: %v\n", expr.Name, symbol.Type, symbol.Value, symbol.Value)

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
	// 验证 current block 没有被改变
	afterBlock := irManager.GetCurrentBasicBlock()
	if currentBlock != afterBlock {
		fmt.Printf("DEBUG: EvaluateIdentifier - WARNING: current block changed from %v to %v during symbol lookup\n", currentBlock, afterBlock)
		// 恢复 current block
		irManager.SetCurrentBasicBlock(currentBlock)
	}
	
	// 再次验证 current block（在 CreateLoad 之前）
	beforeLoadBlock := irManager.GetCurrentBasicBlock()
	fmt.Printf("DEBUG: EvaluateIdentifier - creating load for %s, current block: %v\n", expr.Name, beforeLoadBlock)
	loadResult, err := irManager.CreateLoad(llvmType, allocaPtr, expr.Name+"_load")
	if err != nil {
		return nil, fmt.Errorf("failed to create load instruction: %w", err)
	}
	
	// 验证 current block 没有被改变（在 CreateLoad 之后）
	afterLoadBlock := irManager.GetCurrentBasicBlock()
	if beforeLoadBlock != afterLoadBlock {
		fmt.Printf("DEBUG: EvaluateIdentifier - WARNING: current block changed from %v to %v during CreateLoad\n", beforeLoadBlock, afterLoadBlock)
		// 恢复 current block
		irManager.SetCurrentBasicBlock(beforeLoadBlock)
	}

	return loadResult, nil
}

// EvaluateBinaryExpr 求值二元表达式
func (ee *ExpressionEvaluatorImpl) EvaluateBinaryExpr(irManager generation.IRModuleManager, expr *entities.BinaryExpr) (interface{}, error) {
	currentBlock := irManager.GetCurrentBasicBlock()
	fmt.Printf("DEBUG: EvaluateBinaryExpr - evaluating %s, current block: %v\n", expr.Op, currentBlock)
	// 求值左侧表达式
	leftResult, err := ee.Evaluate(irManager, expr.Left)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate left operand: %w", err)
	}

	// 验证 current block 没有被改变
	afterLeftBlock := irManager.GetCurrentBasicBlock()
	if currentBlock != afterLeftBlock {
		fmt.Printf("DEBUG: EvaluateBinaryExpr - WARNING: current block changed from %v to %v after left evaluation\n", currentBlock, afterLeftBlock)
		// 恢复 current block
		irManager.SetCurrentBasicBlock(currentBlock)
	}

	// 求值右侧表达式
	rightResult, err := ee.Evaluate(irManager, expr.Right)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate right operand: %w", err)
	}

	// 验证 current block 没有被改变
	afterRightBlock := irManager.GetCurrentBasicBlock()
	if currentBlock != afterRightBlock {
		fmt.Printf("DEBUG: EvaluateBinaryExpr - WARNING: current block changed from %v to %v after right evaluation\n", currentBlock, afterRightBlock)
		// 恢复 current block
		irManager.SetCurrentBasicBlock(currentBlock)
	}

	// 特殊处理：指针与 nil 的比较（ptr == nil 或 ptr != nil）
	// 检查是否是指针比较操作
	if expr.Op == "==" || expr.Op == "!=" {
		// 检查左侧或右侧是否是 runtime_null_ptr() 调用
		leftIsNullPtr := ee.isNullPtrCall(expr.Left)
		rightIsNullPtr := ee.isNullPtrCall(expr.Right)
		
		if leftIsNullPtr || rightIsNullPtr {
			// 指针与 nil 的比较
			// 确定哪个是指针，哪个是 nil
			var ptrValue interface{}
			if leftIsNullPtr {
				// left 是 nil，right 是指针
				ptrValue = rightResult
			} else {
				// right 是 nil，left 是指针
				ptrValue = leftResult
			}
			
			// 创建 NULL 指针常量（i8* null）
			nullPtr := constant.NewNull(types.NewPointer(types.I8))
			
			// 确保指针值是指针类型
			ptrLLVMValue, ok := ptrValue.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("pointer value is not a valid LLVM value for null comparison")
			}
			
			// 如果指针不是 i8* 类型，需要转换为 i8*
			ptrType := ptrLLVMValue.Type()
			if !ptrType.Equal(types.NewPointer(types.I8)) {
				// 转换为 i8* 类型
				i8PtrType := types.NewPointer(types.I8)
				castResult, err := irManager.CreateBitCast(ptrLLVMValue, i8PtrType, "ptr_cast")
				if err != nil {
					return nil, fmt.Errorf("failed to cast pointer to i8* for null comparison: %w", err)
				}
				if castValue, ok := castResult.(llvalue.Value); ok {
					ptrLLVMValue = castValue
				} else {
					return nil, fmt.Errorf("CreateBitCast returned invalid type: %T", castResult)
				}
			}
			
			// 使用指针比较（icmp eq 或 icmp ne）
			// 在LLVM IR中，指针比较使用 icmp 指令
			ee.binaryOpCounter++
			uniqueName := fmt.Sprintf("null_cmp_%d", ee.binaryOpCounter)
			
			// 创建指针比较指令
			currentBlock := irManager.GetCurrentBasicBlock()
			if currentBlock == nil {
				return nil, fmt.Errorf("no current basic block set for pointer comparison")
			}
			
			// 将 currentBlock 转换为 *ir.Block
			block, ok := currentBlock.(*ir.Block)
			if !ok {
				return nil, fmt.Errorf("current block is not *ir.Block: %T", currentBlock)
			}
			
			var cmpResult llvalue.Value
			if expr.Op == "==" {
				// ptr == nil -> icmp eq ptr, null
				cmpResult = block.NewICmp(enum.IPredEQ, ptrLLVMValue, nullPtr)
			} else {
				// ptr != nil -> icmp ne ptr, null
				cmpResult = block.NewICmp(enum.IPredNE, ptrLLVMValue, nullPtr)
			}
			
			// 设置名称
			if named, ok := cmpResult.(llvalue.Named); ok {
				named.SetName(uniqueName)
			}
			
			return cmpResult, nil
		}
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

// isNullPtrCall 检查表达式是否是 runtime_null_ptr() 函数调用
func (ee *ExpressionEvaluatorImpl) isNullPtrCall(expr entities.Expr) bool {
	if funcCall, ok := expr.(*entities.FuncCall); ok {
		return funcCall.Name == "runtime_null_ptr" && len(funcCall.Args) == 0
	}
	return false
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
	// 添加调试输出
	if strings.Contains(expr.Name, "find_entry") {
		fmt.Printf("DEBUG: EvaluateFuncCall - START: funcName=%s, currentTypeParamMap=%v\n", expr.Name, irManager.GetCurrentFunctionTypeParams())
	}
	
	// 特殊处理：Ok 和 Err 是 Result 类型的构造函数，不是函数调用
	if expr.Name == "Ok" {
		if len(expr.Args) != 1 {
			return nil, fmt.Errorf("Ok constructor requires exactly one argument")
		}
		// 将 FuncCall 转换为 OkLiteral 并求值
		okLiteral := &entities.OkLiteral{Value: expr.Args[0]}
		return ee.EvaluateOkLiteral(irManager, okLiteral)
	}
	
	if expr.Name == "Err" {
		if len(expr.Args) != 1 {
			return nil, fmt.Errorf("Err constructor requires exactly one argument")
		}
		// 将 FuncCall 转换为 ErrLiteral 并求值
		errLiteral := &entities.ErrLiteral{Error: expr.Args[0]}
		return ee.EvaluateErrLiteral(irManager, errLiteral)
	}

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

	// 如果函数调用包含类型参数（泛型函数调用），设置类型参数映射
	// 例如：bucket_index[int, string](map, key) -> 设置{"K": "int", "V": "string"}
	var savedTypeParams map[string]string
	var typeArgs []string
	
	// 方法1：从expr.TypeArgs获取类型参数（如果解析器正确设置了）
	if len(expr.TypeArgs) > 0 {
		typeArgs = expr.TypeArgs
	} else {
		// 方法2：从函数名中提取类型参数（如果函数名包含[和]）
		// 例如：bucket_index[int, string] -> 提取["int", "string"]
		if strings.Contains(expr.Name, "[") && strings.Contains(expr.Name, "]") {
			openBracket := strings.Index(expr.Name, "[")
			closeBracket := strings.LastIndex(expr.Name, "]")
			if openBracket != -1 && closeBracket != -1 && closeBracket > openBracket {
				typeParamsStr := expr.Name[openBracket+1 : closeBracket]
				typeArgs = strings.Split(typeParamsStr, ",")
				for i := range typeArgs {
					typeArgs[i] = strings.TrimSpace(typeArgs[i])
				}
				// 更新函数名为基础函数名（移除类型参数）
				expr.Name = expr.Name[:openBracket]
			}
		}
	}
	
	// 方法3：如果仍然没有类型参数，尝试从参数类型推断
	if len(typeArgs) == 0 {
		// 获取函数定义的类型参数列表
		funcTypeParamNames := irManager.GetFunctionTypeParamNames(expr.Name)
		if len(funcTypeParamNames) > 0 && len(expr.Args) > 0 {
			// 从第一个参数的类型推断类型参数
			// 例如：如果第一个参数是 HashMap[int, string]，则推断出 K=int, V=string
			firstArgType := ee.getArgType(irManager, expr.Args[0])
			if firstArgType != "" {
				// 添加调试输出
				if strings.Contains(expr.Name, "find_entry") {
					fmt.Printf("DEBUG: EvaluateFuncCall - funcName=%s, firstArgType=%s (before substitution)\n", expr.Name, firstArgType)
				}
				
				// ✅ 修复：如果参数类型包含泛型类型参数（如K, V），使用当前类型参数映射替换
				// 例如：如果当前类型参数映射是{"K": "User", "V": "string"}，且参数类型是Bucket[K, V]
				// 则先替换为Bucket[User, string]，然后提取类型参数
				currentTypeParamMap := irManager.GetCurrentFunctionTypeParams()
				if strings.Contains(expr.Name, "find_entry") {
					fmt.Printf("DEBUG: EvaluateFuncCall - funcName=%s, currentTypeParamMap=%v\n", expr.Name, currentTypeParamMap)
				}
				if currentTypeParamMap != nil && len(currentTypeParamMap) > 0 {
					// 替换泛型类型参数为实际类型
					substitutedType := ee.substituteTypeParamsInString(firstArgType, currentTypeParamMap)
					if substitutedType != firstArgType {
						fmt.Printf("DEBUG: EvaluateFuncCall - substituted argType: %s -> %s\n", firstArgType, substitutedType)
						firstArgType = substitutedType
					} else {
						if strings.Contains(expr.Name, "find_entry") {
							fmt.Printf("DEBUG: EvaluateFuncCall - substitution did not change type: %s\n", firstArgType)
						}
					}
				} else {
					if strings.Contains(expr.Name, "find_entry") {
						fmt.Printf("DEBUG: EvaluateFuncCall - currentTypeParamMap is empty or nil for funcName=%s\n", expr.Name)
					}
				}
				
				// 从参数类型提取类型参数
				// 例如：HashMap[int, string] -> ["int", "string"]
				extractedTypeArgs := ee.extractTypeParamsFromArgType(firstArgType)
				if len(extractedTypeArgs) > 0 {
					typeArgs = extractedTypeArgs
					fmt.Printf("DEBUG: EvaluateFuncCall - inferred typeArgs=%v from argType=%s\n", typeArgs, firstArgType)
				} else {
					if strings.Contains(expr.Name, "find_entry") {
						fmt.Printf("DEBUG: EvaluateFuncCall - failed to extract typeArgs from argType=%s\n", firstArgType)
					}
				}
			} else {
				if strings.Contains(expr.Name, "find_entry") {
					fmt.Printf("DEBUG: EvaluateFuncCall - firstArgType is empty for funcName=%s\n", expr.Name)
				}
			}
		}
	}
	
	if len(typeArgs) > 0 {
		// 保存当前的类型参数映射（如果有）
		savedTypeParams = irManager.GetCurrentFunctionTypeParams()
		
		// 构建类型参数映射
		// 优先使用函数定义的类型参数列表（如果存在）
		typeParamMap := make(map[string]string)
		funcTypeParamNames := irManager.GetFunctionTypeParamNames(expr.Name)
		if len(funcTypeParamNames) > 0 && len(funcTypeParamNames) == len(typeArgs) {
			// 使用函数定义的类型参数列表建立映射
			// 例如：函数定义是 bucket_index[K, V]，类型参数是 ["int", "string"]
			// 则映射为 {"K": "int", "V": "string"}
			for i, typeArg := range typeArgs {
				typeParamMap[funcTypeParamNames[i]] = typeArg
			}
		} else {
			// 根据类型参数数量确定泛型参数名（向后兼容）
			// 单参数：T
			// 双参数：K, V
			// 三参数：T1, T2, T3
			for i, typeArg := range typeArgs {
				var genericParamName string
				if len(typeArgs) == 1 {
					genericParamName = "T"
				} else if len(typeArgs) == 2 {
					if i == 0 {
						genericParamName = "K"
					} else {
						genericParamName = "V"
					}
				} else {
					genericParamName = fmt.Sprintf("T%d", i+1)
				}
				typeParamMap[genericParamName] = typeArg
			}
		}
		
		// 设置类型参数映射
		irManager.SetCurrentFunctionTypeParams(typeParamMap)
		fmt.Printf("DEBUG: EvaluateFuncCall - funcName=%s, typeArgs=%v, typeParamMap=%v\n", expr.Name, typeArgs, typeParamMap)
		
		// ✅ 新增：如果是泛型函数调用且类型参数已知，延迟编译函数体
		if ee.statementGenerator != nil {
			// 检查是否是泛型函数
			funcDef, isGeneric := irManager.GetGenericFunctionDefinition(expr.Name)
			if isGeneric {
				// 获取函数定义的类型参数列表（用于生成单态化函数名）
				funcTypeParamNames := irManager.GetFunctionTypeParamNames(expr.Name)
				if len(funcTypeParamNames) > 0 {
					// 生成单态化函数名
					monoName := ee.generateMonomorphizedName(expr.Name, typeParamMap, funcTypeParamNames)
					
					// 检查是否已经编译
					if !irManager.IsMonomorphizedFunctionCompiled(monoName) {
						// 延迟编译函数体
						err := ee.statementGenerator.CompileMonomorphizedFunction(irManager, funcDef, typeParamMap, monoName)
						if err != nil {
							// 恢复类型参数映射
							irManager.SetCurrentFunctionTypeParams(savedTypeParams)
							return nil, fmt.Errorf("failed to compile monomorphized function %s: %w", monoName, err)
						}
						// 标记已编译
						irManager.MarkMonomorphizedFunctionCompiled(monoName)
						fmt.Printf("DEBUG: EvaluateFuncCall - compiled monomorphized function %s -> %s\n", expr.Name, monoName)
					}
					
					// 使用单态化函数名调用
					expr.Name = monoName
				}
			}
		}
	}

	// 求值所有参数
	args := make([]interface{}, len(expr.Args))
	for i, arg := range expr.Args {
		argResult, err := ee.Evaluate(irManager, arg)
		if err != nil {
			// 恢复类型参数映射
			if len(typeArgs) > 0 {
				irManager.SetCurrentFunctionTypeParams(savedTypeParams)
			}
			return nil, fmt.Errorf("failed to evaluate argument %d: %w", i, err)
		}
		args[i] = argResult
	}

	// 使用irManager创建函数调用
	// 这里假设函数已经声明（在运行时函数或用户定义函数中）
	callResult, err := irManager.CreateCall(expr.Name, args...)
	
	// 恢复类型参数映射
	if len(typeArgs) > 0 {
		irManager.SetCurrentFunctionTypeParams(savedTypeParams)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to create function call: %w", err)
	}

	return callResult, nil
}

// EvaluateArrayLiteral 求值数组字面量
func (ee *ExpressionEvaluatorImpl) EvaluateArrayLiteral(irManager generation.IRModuleManager, expr *entities.ArrayLiteral) (interface{}, error) {
	// 获取数组长度
	arrayLen := len(expr.Elements)

	// 对于空数组字面量 []，返回 NULL 指针（表示空切片）
	if arrayLen == 0 {
		nullPtrFunc, exists := irManager.GetExternalFunction("runtime_null_ptr")
		if !exists {
			return nil, fmt.Errorf("runtime_null_ptr function not declared")
		}
		nullPtr, err := irManager.CreateCall(nullPtrFunc)
		if err != nil {
			return nil, fmt.Errorf("failed to call runtime_null_ptr: %w", err)
		}
		return nullPtr, nil
	}

	// 对于非空数组，创建数组常量
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
	// 检查是否是Map索引访问（map[key]）
	var mapType string
	if ident, ok := expr.Array.(*entities.Identifier); ok {
		// 从符号表获取类型信息
		symbol, err := ee.symbolManager.LookupSymbol(ident.Name)
		if err == nil {
			if strings.HasPrefix(symbol.Type, "map[") {
				// 这是Map索引访问
				mapType = symbol.Type
			}
		}
	}

	// 如果是Map类型，使用Map访问逻辑
	if mapType != "" {
		return ee.evaluateMapIndexAccess(irManager, expr, mapType)
	}

	// 否则，使用数组索引访问逻辑（原有逻辑）
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

// evaluateMapIndexAccess 求值Map索引访问表达式（map[key]）
func (ee *ExpressionEvaluatorImpl) evaluateMapIndexAccess(
	irManager generation.IRModuleManager,
	expr *entities.IndexExpr,
	mapType string,
) (interface{}, error) {
	// 1. 解析Map类型（提取键类型和值类型）
	keyType, valueType, err := ee.parseMapType(mapType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse map type %s: %w", mapType, err)
	}

	// 2. 根据键类型和值类型选择运行时函数
	runtimeFuncName, err := ee.selectMapRuntimeFunction(keyType, valueType, "get")
	if err != nil {
		return nil, fmt.Errorf("failed to select runtime function for map[%s]%s: %w", keyType, valueType, err)
	}

	// 3. 求值map表达式（获取map指针）
	mapValue, err := ee.Evaluate(irManager, expr.Array)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate map expression: %w", err)
	}

	mapValueLLVM, ok := mapValue.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("map value is not a valid LLVM value: %T", mapValue)
	}

	// 4. 求值key表达式（根据键类型处理）
	keyValue, err := ee.Evaluate(irManager, expr.Index)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate key expression: %w", err)
	}

	keyValueLLVM, ok := keyValue.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("key value is not a valid LLVM value: %T", keyValue)
	}

	// 5. 获取对应的运行时函数
	mapGetFunc, exists := irManager.GetExternalFunction(runtimeFuncName)
	if !exists {
		return nil, fmt.Errorf("runtime function %s not declared", runtimeFuncName)
	}

	// 6. ✅ 新增：检查是否是结构体键函数（runtime_map_get_struct_*）
	// 如果是，需要传递hash和equals函数指针
	if strings.HasPrefix(runtimeFuncName, "runtime_map_get_struct_") || strings.HasPrefix(runtimeFuncName, "runtime_map_set_struct_") {
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
		hashFunc, err := ee.ensureStructHashFunction(baseKeyTypeName)
		if err != nil {
			return nil, fmt.Errorf("failed to ensure hash function for %s: %w", baseKeyTypeName, err)
		}
		
		equalsFunc, err := ee.ensureStructEqualsFunction(baseKeyTypeName)
		if err != nil {
			return nil, fmt.Errorf("failed to ensure equals function for %s: %w", baseKeyTypeName, err)
		}
		
		// 获取函数指针（*ir.Func可以直接传递，CreateCall会处理）
		// 注意：在LLVM IR中，函数本身就是值类型，可以直接传递
		// CreateCall现在支持*ir.Func作为参数
		
		// 调用运行时函数，传递函数指针
		var result interface{}
		result, err = irManager.CreateCall(mapGetFunc, mapValueLLVM, keyValueLLVM, hashFunc, equalsFunc)
		if err != nil {
			return nil, fmt.Errorf("failed to call %s: %w", runtimeFuncName, err)
		}
		
		resultLLVM, ok := result.(llvalue.Value)
		if !ok {
			return nil, fmt.Errorf("runtime function %s returned invalid type: %T", runtimeFuncName, result)
		}
		// P0：key 不存在时 map_get 返回 NULL，解引用前必须检查，避免对 NULL 做 CreateLoad
		if err := ee.emitMapGetNullCheck(irManager, resultLLVM, "struct"); err != nil {
			return nil, err
		}
		if valueType == "int" || valueType == "i32" {
			derefValue, err := irManager.CreateLoad(types.I32, resultLLVM, "map_value")
			if err != nil {
				return nil, fmt.Errorf("failed to dereference map value pointer: %w", err)
			}
			return derefValue, nil
		} else if valueType == "float" || valueType == "f32" || valueType == "f64" {
			derefValue, err := irManager.CreateLoad(types.Double, resultLLVM, "map_value")
			if err != nil {
				return nil, fmt.Errorf("failed to dereference map value pointer: %w", err)
			}
			return derefValue, nil
		} else if valueType == "string" {
			return result, nil
		} else {
			return result, nil
		}
	} else {
		// 6. 调用运行时函数
		result, err := irManager.CreateCall(mapGetFunc, mapValueLLVM, keyValueLLVM)
		if err != nil {
			return nil, fmt.Errorf("failed to call %s: %w", runtimeFuncName, err)
		}
		resultLLVM, ok := result.(llvalue.Value)
		if !ok {
			return nil, fmt.Errorf("runtime function %s returned invalid type: %T", runtimeFuncName, result)
		}
		// P0：key 不存在时 map_get 返回 NULL，解引用前必须检查
		if err := ee.emitMapGetNullCheck(irManager, resultLLVM, "scalar"); err != nil {
			return nil, err
		}
		if valueType == "int" || valueType == "i32" {
			derefValue, err := irManager.CreateLoad(types.I32, resultLLVM, "map_value")
			if err != nil {
				return nil, fmt.Errorf("failed to dereference map value pointer: %w", err)
			}
			return derefValue, nil
		} else if valueType == "float" || valueType == "f32" || valueType == "f64" {
			derefValue, err := irManager.CreateLoad(types.Double, resultLLVM, "map_value")
			if err != nil {
				return nil, fmt.Errorf("failed to dereference map value pointer: %w", err)
			}
			return derefValue, nil
		} else if valueType == "string" {
			return result, nil
		} else {
			return result, nil
		}
	}
}

// emitMapGetNullCheck 对 map_get 返回的指针做 NULL 检查：若为 NULL 则分支到 unreachable，否则继续到 blockCont
func (ee *ExpressionEvaluatorImpl) emitMapGetNullCheck(irManager generation.IRModuleManager, resultLLVM interface{}, blockSuffix string) error {
	cond, err := irManager.CreatePointerIsNull(resultLLVM)
	if err != nil {
		return err
	}
	blockNull, err := irManager.CreateBasicBlock("map_get_null_" + blockSuffix)
	if err != nil {
		return err
	}
	blockCont, err := irManager.CreateBasicBlock("map_get_cont_" + blockSuffix)
	if err != nil {
		return err
	}
	if err := irManager.CreateCondBr(cond, blockNull, blockCont); err != nil {
		return err
	}
	if err := irManager.SetCurrentBasicBlock(blockNull); err != nil {
		return err
	}
	if err := irManager.CreateUnreachable(); err != nil {
		return err
	}
	return irManager.SetCurrentBasicBlock(blockCont)
}

// parseMapType 解析Map类型，返回键类型和值类型
func (ee *ExpressionEvaluatorImpl) parseMapType(mapType string) (keyType string, valueType string, error error) {
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
func (ee *ExpressionEvaluatorImpl) selectMapRuntimeFunction(keyType, valueType, operation string) (string, error) {
	// 规范化类型名称（用于函数名）
	keyTypeName := ee.normalizeTypeName(keyType)
	valueTypeName := ee.normalizeTypeName(valueType)

	// ✅ 新增：特殊处理：结构体类型作为键
	// 例如：map[User]string -> runtime_map_set_struct_string
	// 检查键类型是否是结构体类型
	if keyTypeName == "unknown" || keyTypeName == "struct" {
		// 提取基础类型名（移除类型参数，如果有）
		baseKeyTypeName := keyType
		if bracketIndex := strings.Index(keyType, "["); bracketIndex != -1 {
			baseKeyTypeName = keyType[:bracketIndex]
		}
		
		// 检查是否在structDefinitions中
		if _, exists := ee.structDefinitions[baseKeyTypeName]; exists {
			// 是结构体键类型，使用通用结构体键函数
			// 生成函数名：runtime_map_{operation}_struct_{valueType}
			valueTypeNameNormalized := ee.normalizeTypeName(valueType)
			funcName := fmt.Sprintf("runtime_map_%s_struct_%s", operation, valueTypeNameNormalized)
			return funcName, nil
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
	// 注意：虽然contains只依赖于键类型，但为了保持命名一致性，仍然包含值类型
	if operation == "contains" {
		funcName := fmt.Sprintf("runtime_map_contains_%s_%s", keyTypeName, valueTypeName)
		return funcName, nil
	}

	// 对于keys操作，函数名格式为：runtime_map_keys_{keyType}_{valueType}
	// 注意：keys只依赖于键类型，但为了保持命名一致性，仍然包含值类型
	if operation == "keys" {
		funcName := fmt.Sprintf("runtime_map_keys_%s_%s", keyTypeName, valueTypeName)
		return funcName, nil
	}

	// 对于values操作，函数名格式为：runtime_map_values_{keyType}_{valueType}
	// 注意：values只依赖于值类型，但为了保持命名一致性，仍然包含键类型
	if operation == "values" {
		funcName := fmt.Sprintf("runtime_map_values_%s_%s", keyTypeName, valueTypeName)
		return funcName, nil
	}

	// ✅ 新增：特殊处理：数组/切片类型作为值
	// 例如：map[string][]int -> runtime_map_set_string_slice_int
	// 例如：map[int][]string -> runtime_map_set_int_slice_string
	if strings.HasPrefix(valueType, "[") && strings.HasSuffix(valueType, "]") {
		// 提取切片元素类型
		elementType := strings.TrimSpace(valueType[1 : len(valueType)-1])
		elementTypeName := ee.normalizeTypeName(elementType)
		
		// 生成函数名：runtime_map_{operation}_{keyType}_slice_{elementType}
		funcName := fmt.Sprintf("runtime_map_%s_%s_slice_%s", operation, keyTypeName, elementTypeName)
		return funcName, nil
	}

	// ✅ 新增：特殊处理：嵌套Map类型作为值
	// 例如：map[string]map[int]string -> runtime_map_set_string_map_int_string
	// 例如：map[int]map[string]int -> runtime_map_set_int_map_string_int
	if strings.HasPrefix(valueType, "map[") {
		// 解析内层Map类型：map[int]string -> keyType="int", valueType="string"
		innerKeyType, innerValueType, err := ee.parseMapType(valueType)
		if err == nil {
			innerKeyTypeName := ee.normalizeTypeName(innerKeyType)
			innerValueTypeName := ee.normalizeTypeName(innerValueType)
			
			// 生成函数名：runtime_map_{operation}_{keyType}_map_{innerKeyType}_{innerValueType}
			funcName := fmt.Sprintf("runtime_map_%s_%s_map_%s_%s", operation, keyTypeName, innerKeyTypeName, innerValueTypeName)
			return funcName, nil
		}
	}

	// ✅ 新增：特殊处理：指针类型作为值
	// 例如：map[string]*User -> runtime_map_set_string_struct（复用结构体实现）
	// 例如：map[int]*User -> runtime_map_set_int_struct（复用结构体实现）
	// 指针类型在运行时是 i8* 指针，与结构体值相同，可以复用结构体值的实现
	if strings.HasPrefix(valueType, "*") {
		// 指针类型在运行时是 i8* 指针，复用结构体值的实现
		// 生成函数名：runtime_map_{operation}_{keyType}_struct
		funcName := fmt.Sprintf("runtime_map_%s_%s_struct", operation, keyTypeName)
		return funcName, nil
	}

	// ✅ 新增：特殊处理：Option类型作为值
	// 例如：map[string]Option[int] -> runtime_map_set_string_struct（复用结构体实现）
	// 例如：map[int]Option[string] -> runtime_map_set_int_struct（复用结构体实现）
	// Option[T] 在运行时是 i8* 指针，与结构体值相同，可以复用结构体值的实现
	if strings.HasPrefix(valueType, "Option[") && strings.HasSuffix(valueType, "]") {
		// Option[T] 在运行时是 i8* 指针，复用结构体值的实现
		// 生成函数名：runtime_map_{operation}_{keyType}_struct
		funcName := fmt.Sprintf("runtime_map_%s_%s_struct", operation, keyTypeName)
		return funcName, nil
	}

	// ✅ 新增：特殊处理：结构体类型作为值
	// 例如：map[string]User -> runtime_map_set_string_struct
	// 例如：map[int]User -> runtime_map_set_int_struct
	// 检查是否是结构体类型：不在基本类型列表中，且不在structDefinitions中（或存在）
	// 注意：如果normalizeTypeName返回"unknown"，且不是数组/Map类型，可能是结构体类型
	if valueTypeName == "unknown" {
		// 检查是否是结构体类型
		// 提取基础类型名（移除类型参数，如果有）
		baseTypeName := valueType
		if bracketIndex := strings.Index(valueType, "["); bracketIndex != -1 {
			baseTypeName = valueType[:bracketIndex]
		}
		
		// 检查是否在structDefinitions中
		if _, exists := ee.structDefinitions[baseTypeName]; exists {
			// 是结构体类型，使用通用结构体函数
			// 生成函数名：runtime_map_{operation}_{keyType}_struct
			funcName := fmt.Sprintf("runtime_map_%s_%s_struct", operation, keyTypeName)
			return funcName, nil
		}
	}

	// 生成函数名：runtime_map_{operation}_{keyType}_{valueType}
	funcName := fmt.Sprintf("runtime_map_%s_%s_%s", operation, keyTypeName, valueTypeName)
	return funcName, nil
}

// normalizeTypeName 规范化类型名称（用于函数名）
func (ee *ExpressionEvaluatorImpl) normalizeTypeName(echoType string) string {
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
		// 例如：*User -> "struct"（复用结构体值的实现）
		if strings.HasPrefix(echoType, "*") {
			// 指针类型在运行时是 i8* 指针，与结构体值相同
			// 复用结构体值的实现（runtime_map_set_string_struct等）
			return "struct"
		}
		
		// ✅ 新增：Option类型支持：Option[T] 在运行时是 i8* 指针，复用struct实现
		// 例如：Option[int] -> "struct"（复用结构体值的实现）
		if strings.HasPrefix(echoType, "Option[") && strings.HasSuffix(echoType, "]") {
			// Option[T] 在运行时是 i8* 指针，与结构体值相同
			// 复用结构体值的实现（runtime_map_set_string_struct等）
			return "struct"
		}
		
		// ✅ 结构体类型支持：检查是否是结构体类型
		// 提取基础类型名（移除类型参数，如 User[T] -> User）
		baseTypeName := echoType
		if bracketIndex := strings.Index(echoType, "["); bracketIndex != -1 {
			baseTypeName = echoType[:bracketIndex]
		}
		
		// 检查是否是结构体类型
		if _, exists := ee.structDefinitions[baseTypeName]; exists {
			// 是结构体类型，返回"struct"（使用通用函数）
			return "struct"
		}
		
		// ✅ 枚举类型支持：枚举在运行时是整数（i32）
		// 如果类型不是基本类型，且不是数组/Map等复合类型，假设它是枚举类型
		// 枚举类型在Map中作为键/值时，应该被当作int处理
		// 例如：map[Status]string -> map[int]string
		if !strings.Contains(echoType, "[") && !strings.Contains(echoType, "]") {
			// 不是数组/Map类型，可能是枚举类型，返回"int"
			return "int"
		}
		return "unknown"
	}
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
	// 处理不同类型的数组表达式
	switch arr := expr.Array.(type) {
	case *entities.Identifier:
		// 对于变量标识符，从符号表获取数组长度信息
		symbol, err := ee.symbolManager.LookupSymbol(arr.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup array symbol %s: %w", arr.Name, err)
		}

		// 检查是否是数组或切片类型
		if strings.HasPrefix(symbol.Type, "[") {
			// 从符号的Value字段获取存储的长度信息
			// 注意：对于切片变量，长度信息可能存储在符号表的元数据中
			if length, ok := symbol.Value.(int); ok && length > 0 {
				return constant.NewInt(types.I32, int64(length)), nil
			}
			
			// ✅ 如果符号表中没有长度信息，调用运行时函数获取长度
			// 对于切片类型（[]T），使用 runtime_slice_len 函数
			// 获取切片指针（从符号表中获取）
			slicePtr, ok := symbol.Value.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("cannot get slice pointer for %s: symbol value is not a LLVM value", arr.Name)
			}
			
			// 调用 runtime_slice_len 函数
			sliceLenFunc, exists := irManager.GetExternalFunction("runtime_slice_len")
			if !exists {
				return nil, fmt.Errorf("runtime_slice_len function not declared")
			}
			
			// 创建函数调用
			call, err := irManager.CreateCall(sliceLenFunc, slicePtr)
			if err != nil {
				return nil, fmt.Errorf("failed to call runtime_slice_len: %w", err)
			}
			
			return call, nil
		}
		
		// ✅ 新增：检查是否是Map类型（map[K]V）
		if strings.HasPrefix(symbol.Type, "map[") {
			// 对于Map类型，调用 runtime_map_len 函数
			// 获取Map指针（从符号表中获取）
			mapPtr, ok := symbol.Value.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("cannot get map pointer for %s: symbol value is not a LLVM value", arr.Name)
			}
			
			// 调用 runtime_map_len 函数（类型无关的通用函数）
			mapLenFunc, exists := irManager.GetExternalFunction("runtime_map_len")
			if !exists {
				return nil, fmt.Errorf("runtime_map_len function not declared")
			}
			
			// 创建函数调用
			call, err := irManager.CreateCall(mapLenFunc, mapPtr)
			if err != nil {
				return nil, fmt.Errorf("failed to call runtime_map_len: %w", err)
			}
			
			return call, nil
		}
		
		return nil, fmt.Errorf("cannot determine length for non-array/non-map type: %s", symbol.Type)

	case *entities.ArrayLiteral:
		// 对于数组字面量，直接返回元素数量
		arrayLen := len(arr.Elements)
		return constant.NewInt(types.I32, int64(arrayLen)), nil

	case *entities.StructAccess:
		// 对于结构体字段访问（如 m.buckets），检查字段类型是否是数组/切片
		// 如果是，则访问对应的 len 字段或调用运行时函数
		// 例如：len(m.buckets) -> 如果 buckets 是 []Bucket[K, V]，需要访问 len 字段或调用运行时函数
		
		// 1. 先获取结构体字段的类型
		fieldValue, err := ee.EvaluateStructAccess(irManager, arr)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate struct field for len(): %w", err)
		}
		
		// 2. 获取对象类型（结构体类型）
		var objectType string
		if ident, ok := arr.Object.(*entities.Identifier); ok {
			symbol, err := ee.symbolManager.LookupSymbol(ident.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to lookup object symbol for len(): %w", err)
			}
			objectType = symbol.Type
			
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
		} else {
			return nil, fmt.Errorf("len() on struct field requires identifier as object, got %T", arr.Object)
		}
		
		// 3. 提取基础类型名
		baseTypeName := objectType
		if strings.HasPrefix(baseTypeName, "*") {
			baseTypeName = baseTypeName[1:]
		}
		if bracketIndex := strings.Index(baseTypeName, "["); bracketIndex != -1 {
			baseTypeName = baseTypeName[:bracketIndex]
		}
		
		// 4. 查找结构体定义，获取字段类型
		structDef, exists := ee.structDefinitions[baseTypeName]
		if !exists {
			return nil, fmt.Errorf("struct definition not found for type %s (base type: %s)", objectType, baseTypeName)
		}
		
		// 5. 查找字段定义
		var fieldType string
		for _, field := range structDef.Fields {
			if field.Name == arr.Field {
				fieldType = field.Type
				break
			}
		}
		if fieldType == "" {
			return nil, fmt.Errorf("field %s not found in struct %s", arr.Field, baseTypeName)
		}
		
		// 6. 替换字段类型中的类型参数（如果是泛型结构体）
		if bracketIndex := strings.Index(objectType, "["); bracketIndex != -1 {
			typeArgsStr := objectType[bracketIndex+1 : len(objectType)-1]
			typeArgs := strings.Split(typeArgsStr, ",")
			for i := range typeArgs {
				typeArgs[i] = strings.TrimSpace(typeArgs[i])
			}
			// 提取类型参数名（从结构体定义中获取）
			// 简化处理：假设类型参数名是 K, V（对于 HashMap）
			if baseTypeName == "HashMap" && len(typeArgs) == 2 {
				fieldType = strings.ReplaceAll(fieldType, "K", typeArgs[0])
				fieldType = strings.ReplaceAll(fieldType, "V", typeArgs[1])
			}
		}
		
		// 7. 检查字段类型是否是数组/切片
		if strings.HasPrefix(fieldType, "[") {
			// 字段是数组/切片类型，需要调用运行时函数获取长度
			// 对于切片（[]T），运行时使用 i8* 指针，需要调用运行时函数获取长度
			// 注意：fieldValue 是 i8* 指针（运行时切片指针）
			sliceLenFunc, exists := irManager.GetExternalFunction("runtime_slice_len")
			if !exists {
				return nil, fmt.Errorf("runtime_slice_len function not declared")
			}
			
			// 调用运行时函数获取切片长度
			slicePtr, ok := fieldValue.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("field value is not a valid LLVM value")
			}
			
			lenResult, err := irManager.CreateCall(sliceLenFunc, slicePtr)
			if err != nil {
				return nil, fmt.Errorf("failed to call runtime_slice_len: %w", err)
			}
			
			return lenResult, nil
		}
		
		// 8. 检查是否是 Vec[T] 的 data 字段（特殊处理）
		if strings.HasPrefix(objectType, "Vec[") && arr.Field == "data" {
			// len(v.data) -> v.len
			lenFieldAccess := &entities.StructAccess{
				Object: arr.Object,
				Field:  "len",
			}
			return ee.EvaluateStructAccess(irManager, lenFieldAccess)
		}
		
		return nil, fmt.Errorf("len() on struct field %s.%s (type: %s) not supported", objectType, arr.Field, fieldType)

	default:
		return nil, fmt.Errorf("len() function requires array or slice operand, got %T", expr.Array)
	}
}

// EvaluateSizeOfExpr 求值 sizeof 表达式
// sizeof(T) 返回类型 T 的大小（字节数）
func (ee *ExpressionEvaluatorImpl) EvaluateSizeOfExpr(irManager generation.IRModuleManager, expr *entities.SizeOfExpr) (interface{}, error) {
	// 计算类型大小
	size, err := ee.calculateTypeSize(expr.TypeName)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate size of type %s: %w", expr.TypeName, err)
	}
	
	// 返回常量整数值
	return constant.NewInt(types.I32, int64(size)), nil
}

// calculateTypeSize 计算类型大小（字节数）
func (ee *ExpressionEvaluatorImpl) calculateTypeSize(typeName string) (int, error) {
	// 基本类型大小
	switch typeName {
	case "i8", "u8", "bool":
		return 1, nil
	case "i16", "u16":
		return 2, nil
	case "i32", "u32", "int", "float":
		return 4, nil
	case "i64", "u64", "double":
		return 8, nil
	case "string", "*i8", "*i16", "*i32", "*i64":
		// 指针类型（64位系统）
		return 8, nil
	default:
		// 对于泛型类型参数（如 "T"），暂时返回错误
		// 在泛型函数中，需要根据具体类型参数计算
		if len(typeName) == 1 && (typeName[0] >= 'A' && typeName[0] <= 'Z') {
			return 0, fmt.Errorf("cannot calculate size of generic type parameter %s - requires concrete type", typeName)
		}
		
		// 对于结构体类型，需要查找结构体定义并累加字段大小
		structDef, exists := ee.structDefinitions[typeName]
		if !exists {
			return 0, fmt.Errorf("cannot calculate size of type %s - struct definition not found", typeName)
		}
		
		// 计算结构体大小：累加所有字段的大小
		totalSize := 0
		for _, field := range structDef.Fields {
			// 递归计算字段类型大小
			fieldSize, err := ee.calculateTypeSize(field.Type)
			if err != nil {
				return 0, fmt.Errorf("cannot calculate size of field %s.%s (type %s): %w", typeName, field.Name, field.Type, err)
			}
			totalSize += fieldSize
		}
		
		return totalSize, nil
	}
}

// EvaluateTypeCastExpr 求值类型转换表达式
// 处理 expr as Type 语法，例如：new_ptr as []T
func (ee *ExpressionEvaluatorImpl) EvaluateTypeCastExpr(irManager generation.IRModuleManager, expr *entities.TypeCastExpr) (interface{}, error) {
	// 求值要转换的表达式
	exprValue, err := ee.Evaluate(irManager, expr.Expr)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression in type cast: %w", err)
	}
	
	// 映射目标类型到LLVM类型
	targetLLVMType, err := ee.typeMapper.MapType(expr.TargetType)
	if err != nil {
		return nil, fmt.Errorf("failed to map target type %s in type cast: %w", expr.TargetType, err)
	}
	
	// 获取源表达式的类型（从符号表或推断）
	// 对于 *i8 → []T 转换，两者在LLVM IR中都是 *i8，可以直接使用 BitCast
	// 但需要特殊处理：如果目标类型是切片，需要处理长度信息
	
	// 检查是否是 char* → string 转换
	if expr.TargetType == "string" {
		// 目标类型是 string，需要调用运行时函数将 char* 转换为 string_t
		// 在 LLVM IR 中，string 类型映射为 *i8（指向 string_t 结构体的指针）
		
		// 确保 exprValue 是 LLVM Value
		exprLLVMValue, ok := exprValue.(llvalue.Value)
		if !ok {
			return nil, fmt.Errorf("expression value is not a valid LLVM value: %T", exprValue)
		}
		
		// 调用运行时函数 runtime_char_ptr_to_string
		// 函数签名：string_t runtime_char_ptr_to_string(char* ptr)
		// 返回类型：string_t（在 LLVM IR 中映射为结构体指针 *i8）
		
		// 获取 string_t 结构体类型（在 LLVM IR 中，string 映射为 *i8）
		stringType, err := ee.typeMapper.MapType("string")
		if err != nil {
			return nil, fmt.Errorf("failed to map string type: %w", err)
		}
		
		stringLLVMType, ok := stringType.(types.Type)
		if !ok {
			return nil, fmt.Errorf("string type is not a valid LLVM type: %T", stringType)
		}
		
		// 调用运行时函数
		// 注意：runtime_char_ptr_to_string 返回 string_t 结构体（值类型）
		// 但我们需要返回指向 string_t 的指针（*i8）
		// 所以需要：
		// 1. 调用函数获取 string_t 结构体
		// 2. 在栈上分配 string_t 结构体
		// 3. 存储返回值到分配的内存
		// 4. 返回指向该内存的指针
		
		// 获取 string_t 结构体类型定义
		// string_t 结构体：{ i8*, i64, i64 } (data, length, capacity)
		i8PtrType := types.NewPointer(types.I8)
		i64Type := types.I64
		stringStructType := types.NewStruct(i8PtrType, i64Type, i64Type) // { i8*, i64, i64 }
		
		// 在栈上分配 string_t 结构体
		alloca, err := irManager.CreateAlloca(stringStructType, "string_t_result")
		if err != nil {
			return nil, fmt.Errorf("failed to allocate string_t struct: %w", err)
		}
		
		// 调用运行时函数
		// CreateCall 会自动从函数签名中获取返回类型
		callResult, err := irManager.CreateCall("runtime_char_ptr_to_string", exprLLVMValue)
		if err != nil {
			return nil, fmt.Errorf("failed to call runtime_char_ptr_to_string: %w", err)
		}
		
		// callResult 是 string_t 结构体（值类型）
		callResultValue, ok := callResult.(llvalue.Value)
		if !ok {
			return nil, fmt.Errorf("call result is not a valid LLVM value: %T", callResult)
		}
		
		// 存储返回值到分配的内存
		err = irManager.CreateStore(callResultValue, alloca)
		if err != nil {
			return nil, fmt.Errorf("failed to store string_t result: %w", err)
		}
		
		// 返回指向 string_t 结构体的指针（转换为 *i8）
		resultPtr, err := irManager.CreateBitCast(alloca, stringLLVMType, "string_ptr")
		if err != nil {
			return nil, fmt.Errorf("failed to cast string_t pointer to string type: %w", err)
		}
		
		return resultPtr, nil
	}
	
	// 检查是否是 []string 类型转换（从结构体指针提取字段）
	if expr.TargetType == "[]string" {
		// 目标类型是 []string，需要检查源表达式是否是结构体指针
		// 例如：从 StringSplitResult* 提取 strings 和 count 字段
		
		// 检查源表达式是否是结构体访问（如 result.strings 和 result.count）
		// 或者是否是结构体指针（需要提取字段）
		
		// 尝试检测源表达式类型
		var sourceStructType string
		var structPtrValue llvalue.Value
		
		// 如果源表达式是标识符，尝试从符号表获取类型
		if ident, ok := expr.Expr.(*entities.Identifier); ok {
			symbol, err := ee.symbolManager.LookupSymbol(ident.Name)
			if err == nil {
				sourceStructType = symbol.Type
				// 求值标识符获取结构体指针值
				identValue, err := ee.Evaluate(irManager, expr.Expr)
				if err == nil {
					if ptrValue, ok := identValue.(llvalue.Value); ok {
						structPtrValue = ptrValue
					}
				}
			}
		} else if funcCall, ok := expr.Expr.(*entities.FuncCall); ok {
			// 如果源表达式是函数调用（如 runtime_string_split），检查返回类型
			runtimeFunctionReturnTypes := map[string]string{
				"runtime_string_split":      "*StringSplitResult",
				"runtime_http_parse_request": "*HttpParseRequestResult",
			}
			if returnType, exists := runtimeFunctionReturnTypes[funcCall.Name]; exists {
				sourceStructType = returnType
				// 求值函数调用获取结构体指针
				funcResult, err := ee.Evaluate(irManager, expr.Expr)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate function call: %w", err)
				}
				if ptrValue, ok := funcResult.(llvalue.Value); ok {
					structPtrValue = ptrValue
				}
			}
		}
		
		// 如果检测到是 StringSplitResult* 类型，提取字段并调用运行时函数
		if sourceStructType == "*StringSplitResult" && structPtrValue != nil {
			// StringSplitResult 结构体字段：
			// - strings: char** (字段索引 0)
			// - count: int32_t (字段索引 1)
			
			// 1. 提取 strings 字段（char**）
			stringsFieldAccess := &entities.StructAccess{
				Object: expr.Expr,
				Field:  "strings",
			}
			stringsFieldValue, err := ee.EvaluateStructAccess(irManager, stringsFieldAccess)
			if err != nil {
				return nil, fmt.Errorf("failed to extract strings field from StringSplitResult: %w", err)
			}
			stringsFieldLLVMValue, ok := stringsFieldValue.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("strings field value is not a valid LLVM value: %T", stringsFieldValue)
			}
			
			// 2. 提取 count 字段（int32_t）
			countFieldAccess := &entities.StructAccess{
				Object: expr.Expr,
				Field:  "count",
			}
			countFieldValue, err := ee.EvaluateStructAccess(irManager, countFieldAccess)
			if err != nil {
				return nil, fmt.Errorf("failed to extract count field from StringSplitResult: %w", err)
			}
			countFieldLLVMValue, ok := countFieldValue.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("count field value is not a valid LLVM value: %T", countFieldValue)
			}
			
			// 3. 调用运行时函数 runtime_char_ptr_array_to_string_slice
			sliceResult, err := irManager.CreateCall("runtime_char_ptr_array_to_string_slice", stringsFieldLLVMValue, countFieldLLVMValue)
			if err != nil {
				return nil, fmt.Errorf("failed to call runtime_char_ptr_array_to_string_slice: %w", err)
			}
			
			sliceResultValue, ok := sliceResult.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("slice result is not a valid LLVM value: %T", sliceResult)
			}
			
			// 4. 返回切片指针（已经是 *i8 类型，符合 []string 的 LLVM IR 表示）
			return sliceResultValue, nil
		}
		
		// 如果不是从结构体提取，使用默认的 BitCast 转换
		// （例如：char** 直接转换为 []string）
		targetType, ok := targetLLVMType.(types.Type)
		if !ok {
			return nil, fmt.Errorf("target type %s is not a valid LLVM type: %T", expr.TargetType, targetLLVMType)
		}
		
		exprLLVMValue, ok := exprValue.(llvalue.Value)
		if !ok {
			return nil, fmt.Errorf("expression value is not a valid LLVM value: %T", exprValue)
		}
		
		// 执行 BitCast
		bitcast, err := irManager.CreateBitCast(exprLLVMValue, targetType, fmt.Sprintf("cast_to_%s", expr.TargetType))
		if err != nil {
			return nil, fmt.Errorf("failed to create bitcast for type cast %s: %w", expr.TargetType, err)
		}
		
		return bitcast, nil
	}
	
	// 检查是否是 *i8 → []T 转换（非 []string 的切片类型）
	if strings.HasPrefix(expr.TargetType, "[") {
		// 目标类型是切片类型（[]T）
		// 在LLVM IR中，*i8 和 []T 都是 *i8，可以直接 BitCast
		// 但长度信息需要单独处理（通过符号表或运行时函数）
		
		// 直接进行 BitCast（因为内存布局相同）
		targetType, ok := targetLLVMType.(types.Type)
		if !ok {
			return nil, fmt.Errorf("target type %s is not a valid LLVM type: %T", expr.TargetType, targetLLVMType)
		}
		
		// 确保 exprValue 是 LLVM Value
		exprLLVMValue, ok := exprValue.(llvalue.Value)
		if !ok {
			return nil, fmt.Errorf("expression value is not a valid LLVM value: %T", exprValue)
		}
		
		// 执行 BitCast
		bitcast, err := irManager.CreateBitCast(exprLLVMValue, targetType, fmt.Sprintf("cast_to_%s", expr.TargetType))
		if err != nil {
			return nil, fmt.Errorf("failed to create bitcast for type cast %s: %w", expr.TargetType, err)
		}
		
		return bitcast, nil
	}
	
	// 对于其他类型转换，使用 BitCast 或更复杂的转换逻辑
	targetType, ok := targetLLVMType.(types.Type)
	if !ok {
		return nil, fmt.Errorf("target type %s is not a valid LLVM type: %T", expr.TargetType, targetLLVMType)
	}
	
	exprLLVMValue, ok := exprValue.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("expression value is not a valid LLVM value: %T", exprValue)
	}
	
	// 执行 BitCast
	bitcast, err := irManager.CreateBitCast(exprLLVMValue, targetType, fmt.Sprintf("cast_to_%s", expr.TargetType))
	if err != nil {
		return nil, fmt.Errorf("failed to create bitcast for type cast %s: %w", expr.TargetType, err)
	}
	
	return bitcast, nil
}

// EvaluateFunctionPointerExpr 求值函数指针表达式
// &func_name 返回函数的指针（转换为 *i8）
func (ee *ExpressionEvaluatorImpl) EvaluateFunctionPointerExpr(irManager generation.IRModuleManager, expr *entities.FunctionPointerExpr) (interface{}, error) {
	// 查找函数
	funcValue, exists := irManager.GetFunction(expr.FunctionName)
	if !exists {
		return nil, fmt.Errorf("function %s not found", expr.FunctionName)
	}
	
	// 转换为 *ir.Func
	llvmFunc, ok := funcValue.(*ir.Func)
	if !ok {
		return nil, fmt.Errorf("invalid function type for %s: %T", expr.FunctionName, funcValue)
	}
	
	// 获取当前基本块（用于创建 BitCast 指令）
	// 注意：函数指针转换需要在基本块中进行
	currentBlock := irManager.GetCurrentBasicBlock()
	if currentBlock == nil {
		return nil, fmt.Errorf("no current basic block set for function pointer conversion")
	}
	
	// 将函数转换为通用指针类型（*i8）
	// 使用 BitCast 将函数指针转换为 i8*
	i8PtrType := types.NewPointer(types.I8)
	bitcast, err := irManager.CreateBitCast(llvmFunc, i8PtrType, fmt.Sprintf("func_ptr_%s", expr.FunctionName))
	if err != nil {
		return nil, fmt.Errorf("failed to convert function pointer to *i8: %w", err)
	}
	
	return bitcast, nil
}

// EvaluateAddressOfExpr 求值取地址表达式
// &expression 返回操作数的地址（转换为 *i8）
func (ee *ExpressionEvaluatorImpl) EvaluateAddressOfExpr(irManager generation.IRModuleManager, expr *entities.AddressOfExpr) (interface{}, error) {
	// 获取当前基本块
	currentBlock := irManager.GetCurrentBasicBlock()
	if currentBlock == nil {
		return nil, fmt.Errorf("no current basic block set for address-of operation")
	}
	
	// 根据操作数类型处理
	switch operand := expr.Operand.(type) {
	case *entities.Identifier:
		// &variable：获取变量的地址或值
		symbol, err := ee.symbolManager.LookupSymbol(operand.Name)
		if err != nil {
			return nil, fmt.Errorf("undefined symbol %s: %w", operand.Name, err)
		}
		
		// symbol.Value 应该是 alloca 指令的结果（指针）
		allocaPtr, ok := symbol.Value.(llvalue.Value)
		if !ok {
			return nil, fmt.Errorf("symbol value is not a valid LLVM value: %T", symbol.Value)
		}
		
		// 特殊处理：如果变量是切片类型（[]T），&variable 应该返回切片数据的指针
		// 在 LLVM IR 中，切片被映射为指针类型，所以需要加载切片的值（它是指针）
		if strings.HasPrefix(symbol.Type, "[") {
			// 切片类型：加载切片的值（它是指向数据的指针）
			// 获取切片元素类型
			elementType, err := ee.inferElementTypeFromArrayType(symbol.Type)
			if err != nil {
				return nil, fmt.Errorf("failed to infer slice element type: %w", err)
			}
			
			// 加载切片的值（切片变量存储的是指向数据的指针）
			slicePtrType := types.NewPointer(elementType)
			sliceValue, err := irManager.CreateLoad(slicePtrType, allocaPtr, fmt.Sprintf("slice_%s", operand.Name))
			if err != nil {
				return nil, fmt.Errorf("failed to load slice value: %w", err)
			}
			
			// 转换为 *i8 类型
			i8PtrType := types.NewPointer(types.I8)
			bitcast, err := irManager.CreateBitCast(sliceValue, i8PtrType, fmt.Sprintf("slice_ptr_%s", operand.Name))
			if err != nil {
				return nil, fmt.Errorf("failed to convert slice pointer to *i8: %w", err)
			}
			return bitcast, nil
		}
		
		// 非切片类型：获取变量的地址（alloca 的结果已经是地址）
		// 转换为 *i8 类型
		i8PtrType := types.NewPointer(types.I8)
		bitcast, err := irManager.CreateBitCast(allocaPtr, i8PtrType, fmt.Sprintf("addr_%s", operand.Name))
		if err != nil {
			return nil, fmt.Errorf("failed to convert address to *i8: %w", err)
		}
		return bitcast, nil
		
	case *entities.IndexExpr:
		// &array[index]：获取数组元素的地址
		// 需要直接获取元素指针，而不是加载值
		// 求值数组表达式
		arrayValue, err := ee.Evaluate(irManager, operand.Array)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate array expression: %w", err)
		}
		
		// 求值索引表达式
		indexValue, err := ee.Evaluate(irManager, operand.Index)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate index expression: %w", err)
		}
		
		// 确定数组元素类型
		var elementType types.Type
		if ident, ok := operand.Array.(*entities.Identifier); ok {
			symbol, err := ee.symbolManager.LookupSymbol(ident.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to lookup array symbol: %w", err)
			}
			elementType, err = ee.inferElementTypeFromArrayType(symbol.Type)
			if err != nil {
				return nil, fmt.Errorf("failed to infer element type: %w", err)
			}
		} else {
			elementType = types.I32 // 默认类型
		}
		
		// 生成 GetElementPtr 指令获取元素指针（不加载值）
		elementPtr, err := irManager.CreateGetElementPtr(elementType, arrayValue, indexValue)
		if err != nil {
			return nil, fmt.Errorf("failed to create GEP for index access: %w", err)
		}
		
		// 转换为 *i8 类型
		i8PtrType := types.NewPointer(types.I8)
		bitcast, err := irManager.CreateBitCast(elementPtr, i8PtrType, "addr_index")
		if err != nil {
			return nil, fmt.Errorf("failed to convert element address to *i8: %w", err)
		}
		return bitcast, nil
		
	case *entities.StructAccess:
		// &object.field：获取结构体字段的地址
		// 先求值对象表达式
		objectValue, err := ee.Evaluate(irManager, operand.Object)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate object for address-of field access: %w", err)
		}
		
		// 获取对象类型
		var objectType types.Type
		if ident, ok := operand.Object.(*entities.Identifier); ok {
			symbol, err := ee.symbolManager.LookupSymbol(ident.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to lookup object symbol: %w", err)
			}
			typeInterface, err := ee.typeMapper.MapType(symbol.Type)
			if err != nil {
				return nil, fmt.Errorf("failed to map object type: %w", err)
			}
			objectType, ok = typeInterface.(types.Type)
			if !ok {
				return nil, fmt.Errorf("mapped type is not a valid LLVM type: %T", typeInterface)
			}
		} else {
			// 默认使用 i32（简化处理）
			objectType = types.I32
		}
		
		// 获取字段指针（使用 GetElementPtr）
		// 注意：这里简化处理，假设字段索引为 0（实际需要根据字段名查找索引）
		fieldIndex := constant.NewInt(types.I32, 0) // 简化：假设第一个字段
		fieldPtr, err := irManager.CreateGetElementPtr(objectType, objectValue, fieldIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to create GEP for field access: %w", err)
		}
		
		// 转换为 *i8 类型
		i8PtrType := types.NewPointer(types.I8)
		bitcast, err := irManager.CreateBitCast(fieldPtr, i8PtrType, fmt.Sprintf("addr_field_%s", operand.Field))
		if err != nil {
			return nil, fmt.Errorf("failed to convert field address to *i8: %w", err)
		}
		return bitcast, nil
		
	default:
		return nil, fmt.Errorf("unsupported operand type for address-of: %T", operand)
	}
}

// EvaluateDereferenceExpr 求值解引用表达式：*pointer
// 例如：*m, *ptr, *arr[0] 等
func (ee *ExpressionEvaluatorImpl) EvaluateDereferenceExpr(irManager generation.IRModuleManager, expr *entities.DereferenceExpr) (interface{}, error) {
	// 获取当前基本块
	currentBlock := irManager.GetCurrentBasicBlock()
	if currentBlock == nil {
		return nil, fmt.Errorf("no current basic block set for dereference operation")
	}
	
	// 求值操作数表达式（应该是指针类型）
	pointerValue, err := ee.Evaluate(irManager, expr.Operand)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate operand for dereference: %w", err)
	}
	
	// 将 pointerValue 转换为 llvalue.Value
	ptrValue, ok := pointerValue.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("operand value is not a valid LLVM value: %T", pointerValue)
	}
	
	// 获取指针指向的类型
	// 在 Echo 中，指针类型通常是 *Type，我们需要推断被指向的类型
	// 简化处理：假设指针指向 i32 类型（实际应该根据操作数类型推断）
	// TODO: 根据操作数类型推断被指向的类型
	pointedType := types.I32 // 默认类型，后续需要根据实际类型推断
	
	// 创建 load 指令，解引用指针
	loadedValue, err := irManager.CreateLoad(pointedType, ptrValue, "deref")
	if err != nil {
		return nil, fmt.Errorf("failed to create load for dereference: %w", err)
	}
	
	return loadedValue, nil
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
	// 构造方法调用名（接收者类型_方法名）
	receiverType := "unknown" // 简化实现，实际应该从接收者推断类型
	if ident, ok := expr.Receiver.(*entities.Identifier); ok {
		// 从符号表获取接收者类型
		if symbol, err := ee.symbolManager.LookupSymbol(ident.Name); err == nil {
			receiverType = symbol.Type // symbol.Type 已经是string类型
			
			// 添加调试输出
			if strings.Contains(expr.MethodName, "insert") || strings.Contains(receiverType, "HashMap") {
				fmt.Printf("DEBUG: EvaluateMethodCallExpr - START: method=%s, receiverName=%s, receiverType=%s\n", expr.MethodName, ident.Name, receiverType)
			}
			
			// ✅ 新增：如果当前函数有类型参数映射，替换类型参数
			typeParamMap := irManager.GetCurrentFunctionTypeParams()
			if strings.Contains(expr.MethodName, "insert") || strings.Contains(receiverType, "HashMap") {
				fmt.Printf("DEBUG: EvaluateMethodCallExpr - currentTypeParamMap=%v\n", typeParamMap)
			}
			if len(typeParamMap) > 0 && strings.Contains(receiverType, "[") {
				hasPointerPrefix := strings.HasPrefix(receiverType, "*")
				baseType := receiverType
				if hasPointerPrefix {
					baseType = receiverType[1:]
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
						receiverType = "*" + newBaseType
					} else {
						receiverType = newBaseType
					}
				}
			}
		}
	} else if deref, ok := expr.Receiver.(*entities.DereferenceExpr); ok {
		// 处理解引用表达式（*m）
		if ident, ok := deref.Operand.(*entities.Identifier); ok {
			if symbol, err := ee.symbolManager.LookupSymbol(ident.Name); err == nil {
				receiverType = symbol.Type // 保持原始类型（包含*前缀）
			}
		}
	} else if structAccess, ok := expr.Receiver.(*entities.StructAccess); ok {
		// ✅ 新增：处理结构体字段访问作为接收者（如 m.buckets.len()）
		// 1. 获取对象类型（结构体类型）
		var objectType string
		if ident, ok := structAccess.Object.(*entities.Identifier); ok {
			symbol, err := ee.symbolManager.LookupSymbol(ident.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to lookup object symbol for method call: %w", err)
			}
			objectType = symbol.Type
			
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
		} else {
			return nil, fmt.Errorf("method call on struct field requires identifier as object, got %T", structAccess.Object)
		}
		
		// 3. 提取基础类型名
		baseTypeName := objectType
		if strings.HasPrefix(baseTypeName, "*") {
			baseTypeName = baseTypeName[1:]
		}
		if bracketIndex := strings.Index(baseTypeName, "["); bracketIndex != -1 {
			baseTypeName = baseTypeName[:bracketIndex]
		}
		
		// 4. 查找结构体定义，获取字段类型
		structDef, exists := ee.structDefinitions[baseTypeName]
		if !exists {
			return nil, fmt.Errorf("struct definition not found for type %s (base type: %s)", objectType, baseTypeName)
		}
		
		// 5. 查找字段定义
		var fieldType string
		for _, field := range structDef.Fields {
			if field.Name == structAccess.Field {
				fieldType = field.Type
				break
			}
		}
		if fieldType == "" {
			return nil, fmt.Errorf("field %s not found in struct %s", structAccess.Field, baseTypeName)
		}
		
		// 6. 替换字段类型中的类型参数（如果是泛型结构体）
		if bracketIndex := strings.Index(objectType, "["); bracketIndex != -1 {
			typeArgsStr := objectType[bracketIndex+1 : len(objectType)-1]
			typeArgs := strings.Split(typeArgsStr, ",")
			for i := range typeArgs {
				typeArgs[i] = strings.TrimSpace(typeArgs[i])
			}
			// 提取类型参数名（从结构体定义中获取）
			// 简化处理：假设类型参数名是 K, V（对于 HashMap）
			if baseTypeName == "HashMap" && len(typeArgs) == 2 {
				fieldType = strings.ReplaceAll(fieldType, "K", typeArgs[0])
				fieldType = strings.ReplaceAll(fieldType, "V", typeArgs[1])
			}
		}
		
		// 7. 使用字段类型作为接收者类型
		receiverType = fieldType
		
		// 8. 特殊处理：如果方法是 "len" 且字段类型是数组/切片，转换为 LenExpr
		if expr.MethodName == "len" && strings.HasPrefix(fieldType, "[") {
			lenExpr := &entities.LenExpr{
				Array: structAccess,
			}
			return ee.EvaluateLenExpr(irManager, lenExpr)
		}
		
		// 使用字段类型作为接收者类型
		receiverType = fieldType
	}

	// ✅ 新增：特殊处理：如果接收者类型是数组/切片类型（以 [ 开头），且方法是 "len"，转换为 LenExpr
	if expr.MethodName == "len" && strings.HasPrefix(receiverType, "[") {
		// 将 MethodCallExpr 转换为 LenExpr
		lenExpr := &entities.LenExpr{
			Array: expr.Receiver,
		}
		return ee.EvaluateLenExpr(irManager, lenExpr)
	}
	
	// ✅ 新增：特殊处理：如果接收者类型是Map类型（以 map[ 开头），且方法是 "len"，转换为 LenExpr
	if expr.MethodName == "len" && strings.HasPrefix(receiverType, "map[") {
		// 将 MethodCallExpr 转换为 LenExpr
		lenExpr := &entities.LenExpr{
			Array: expr.Receiver,
		}
		return ee.EvaluateLenExpr(irManager, lenExpr)
	}
	
	// ✅ 新增：特殊处理：Map的clear()方法
	if expr.MethodName == "clear" && strings.HasPrefix(receiverType, "map[") {
		// 获取Map指针
		var mapPtr llvalue.Value
		if ident, ok := expr.Receiver.(*entities.Identifier); ok {
			symbol, err := ee.symbolManager.LookupSymbol(ident.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to lookup map symbol for clear(): %w", err)
			}
			mapPtrVal, ok := symbol.Value.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("map symbol value is not a LLVM value")
			}
			mapPtr = mapPtrVal
		} else {
			// 对于其他类型的接收者，先求值
			receiverValue, err := ee.Evaluate(irManager, expr.Receiver)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate map receiver for clear(): %w", err)
			}
			mapPtrVal, ok := receiverValue.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("map receiver value is not a LLVM value")
			}
			mapPtr = mapPtrVal
		}
		
		// 调用 runtime_map_clear 函数（类型无关的通用函数）
		mapClearFunc, exists := irManager.GetExternalFunction("runtime_map_clear")
		if !exists {
			return nil, fmt.Errorf("runtime_map_clear function not declared")
		}
		
		// 创建函数调用
		_, err := irManager.CreateCall(mapClearFunc, mapPtr)
		if err != nil {
			return nil, fmt.Errorf("failed to call runtime_map_clear: %w", err)
		}
		
		// clear()方法返回void，但我们需要返回一个值（可以是nil或void常量）
		return nil, nil
	}
	
	// ✅ 新增：特殊处理：Map的contains(key)方法
	if expr.MethodName == "contains" && strings.HasPrefix(receiverType, "map[") {
		// 检查参数数量
		if len(expr.Args) != 1 {
			return nil, fmt.Errorf("map.contains() requires exactly 1 argument (key), got %d", len(expr.Args))
		}
		
		// 获取Map指针
		var mapPtr llvalue.Value
		if ident, ok := expr.Receiver.(*entities.Identifier); ok {
			symbol, err := ee.symbolManager.LookupSymbol(ident.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to lookup map symbol for contains(): %w", err)
			}
			mapPtrVal, ok := symbol.Value.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("map symbol value is not a LLVM value")
			}
			mapPtr = mapPtrVal
		} else {
			// 对于其他类型的接收者，先求值
			receiverValue, err := ee.Evaluate(irManager, expr.Receiver)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate map receiver for contains(): %w", err)
			}
			mapPtrVal, ok := receiverValue.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("map receiver value is not a LLVM value")
			}
			mapPtr = mapPtrVal
		}
		
		// 解析Map类型，获取键类型
		keyType, _, err := ee.parseMapType(receiverType)
		if err != nil {
			return nil, fmt.Errorf("failed to parse map type for contains(): %w", err)
		}
		
		// 求值key参数
		keyValue, err := ee.Evaluate(irManager, expr.Args[0])
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate key argument for contains(): %w", err)
		}
		
		// 根据键类型选择对应的contains函数
		// 注意：contains函数只依赖于键类型，值类型不影响函数选择
		// 但为了保持函数命名一致性，我们仍然需要值类型参数
		// 从receiverType中解析值类型
		_, valueType, err := ee.parseMapType(receiverType)
		if err != nil {
			return nil, fmt.Errorf("failed to parse map value type for contains(): %w", err)
		}
		containsFuncName, err := ee.selectMapRuntimeFunction(keyType, valueType, "contains")
		if err != nil {
			return nil, fmt.Errorf("failed to select map contains function: %w", err)
		}
		
		// 获取运行时函数
		containsFunc, exists := irManager.GetExternalFunction(containsFuncName)
		if !exists {
			return nil, fmt.Errorf("map contains function %s not declared", containsFuncName)
		}
		
		// 创建函数调用
		// 注意：key的类型需要根据keyType进行转换
		keyLLVMValue, ok := keyValue.(llvalue.Value)
		if !ok {
			return nil, fmt.Errorf("key value is not a LLVM value")
		}
		
		// 根据键类型，可能需要转换key的类型
		// 对于基本类型（int, bool, float），直接传递值
		// 对于string，传递指针
		var keyArg llvalue.Value = keyLLVMValue
		normalizedKeyType := ee.normalizeTypeName(keyType)
		if normalizedKeyType == "string" {
			// 字符串键：key已经是i8*，直接使用
			keyArg = keyLLVMValue
		} else if normalizedKeyType == "int" {
			// 整数键：key应该是i32，直接使用
			keyArg = keyLLVMValue
		} else if normalizedKeyType == "float" {
			// 浮点数键：key应该是f64，直接使用
			keyArg = keyLLVMValue
		} else if normalizedKeyType == "bool" {
			// 布尔键：key应该是i1，直接使用
			keyArg = keyLLVMValue
		}
		
		// 创建函数调用
		result, err := irManager.CreateCall(containsFunc, mapPtr, keyArg)
		if err != nil {
			return nil, fmt.Errorf("failed to call map contains function: %w", err)
		}
		
		return result, nil
	}
	
	// ✅ 新增：特殊处理：Map的keys()方法
	if expr.MethodName == "keys" && strings.HasPrefix(receiverType, "map[") {
		// 获取Map指针
		var mapPtr llvalue.Value
		if ident, ok := expr.Receiver.(*entities.Identifier); ok {
			symbol, err := ee.symbolManager.LookupSymbol(ident.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to lookup map symbol for keys(): %w", err)
			}
			mapPtrVal, ok := symbol.Value.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("map symbol value is not a LLVM value")
			}
			mapPtr = mapPtrVal
		} else {
			receiverValue, err := ee.Evaluate(irManager, expr.Receiver)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate map receiver for keys(): %w", err)
			}
			mapPtrVal, ok := receiverValue.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("map receiver value is not a LLVM value")
			}
			mapPtr = mapPtrVal
		}
		
		// 解析Map类型，获取键类型和值类型
		keyType, valueType, err := ee.parseMapType(receiverType)
		if err != nil {
			return nil, fmt.Errorf("failed to parse map type for keys(): %w", err)
		}
		
		// 根据键类型选择对应的keys函数
		keysFuncName, err := ee.selectMapRuntimeFunction(keyType, valueType, "keys")
		if err != nil {
			return nil, fmt.Errorf("failed to select map keys function: %w", err)
		}
		
		// 获取运行时函数
		keysFunc, exists := irManager.GetExternalFunction(keysFuncName)
		if !exists {
			return nil, fmt.Errorf("map keys function %s not declared", keysFuncName)
		}
		
		// 创建函数调用
		result, err := irManager.CreateCall(keysFunc, mapPtr)
		if err != nil {
			return nil, fmt.Errorf("failed to call map keys function: %w", err)
		}
		
		return result, nil
	}
	
	// ✅ 新增：特殊处理：Map的values()方法
	if expr.MethodName == "values" && strings.HasPrefix(receiverType, "map[") {
		// 获取Map指针
		var mapPtr llvalue.Value
		if ident, ok := expr.Receiver.(*entities.Identifier); ok {
			symbol, err := ee.symbolManager.LookupSymbol(ident.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to lookup map symbol for values(): %w", err)
			}
			mapPtrVal, ok := symbol.Value.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("map symbol value is not a LLVM value")
			}
			mapPtr = mapPtrVal
		} else {
			receiverValue, err := ee.Evaluate(irManager, expr.Receiver)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate map receiver for values(): %w", err)
			}
			mapPtrVal, ok := receiverValue.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("map receiver value is not a LLVM value")
			}
			mapPtr = mapPtrVal
		}
		
		// 解析Map类型，获取键类型和值类型
		keyType, valueType, err := ee.parseMapType(receiverType)
		if err != nil {
			return nil, fmt.Errorf("failed to parse map type for values(): %w", err)
		}
		
		// 根据值类型选择对应的values函数
		valuesFuncName, err := ee.selectMapRuntimeFunction(keyType, valueType, "values")
		if err != nil {
			return nil, fmt.Errorf("failed to select map values function: %w", err)
		}
		
		// 获取运行时函数
		valuesFunc, exists := irManager.GetExternalFunction(valuesFuncName)
		if !exists {
			return nil, fmt.Errorf("map values function %s not declared", valuesFuncName)
		}
		
		// 创建函数调用
		result, err := irManager.CreateCall(valuesFunc, mapPtr)
		if err != nil {
			return nil, fmt.Errorf("failed to call map values function: %w", err)
		}
		
		return result, nil
	}

	// 将泛型实例化类型转换为泛型定义类型来查找方法
	// 例如：HashMap[int, string] -> HashMap[K, V]
	genericType := ee.eraseGenericTypeParameters(receiverType)
	
	// 在方法调用时，从接收者类型推断类型参数，并设置类型参数映射
	// 例如：如果接收者类型是 HashMap[int, string]，则推断出 K=int, V=string
	// 这对于方法内部调用其他泛型函数（如 bucket_index）很重要
	var savedTypeParams map[string]string
	var extractedTypeArgs []string // 移到外层作用域
	var typeParamMap map[string]string // 移到外层作用域
	
	// 添加调试输出
	if strings.Contains(expr.MethodName, "insert") || strings.Contains(receiverType, "HashMap") {
		fmt.Printf("DEBUG: EvaluateMethodCallExpr - method=%s, receiverType=%s\n", expr.MethodName, receiverType)
	}
	
	if strings.Contains(receiverType, "[") && strings.Contains(receiverType, "]") {
		// 从接收者类型提取类型参数
		// 例如：HashMap[int, string] -> ["int", "string"]
		extractedTypeArgs = ee.extractTypeParamsFromArgType(receiverType)
		if len(extractedTypeArgs) > 0 {
			if strings.Contains(expr.MethodName, "insert") || strings.Contains(receiverType, "HashMap") {
				fmt.Printf("DEBUG: EvaluateMethodCallExpr - extractedTypeArgs=%v from receiverType=%s\n", extractedTypeArgs, receiverType)
			}
			// 获取方法定义的类型参数列表
			// 方法名格式：*HashMap[K, V]_insert 或 HashMap[K, V]_insert
			// 我们需要从方法定义中获取类型参数列表
			methodFuncName := fmt.Sprintf("*%s_%s", genericType, expr.MethodName)
			funcTypeParamNames := irManager.GetFunctionTypeParamNames(methodFuncName)
			if len(funcTypeParamNames) == 0 {
				// 尝试值接收者格式
				methodFuncName = fmt.Sprintf("%s_%s", genericType, expr.MethodName)
				funcTypeParamNames = irManager.GetFunctionTypeParamNames(methodFuncName)
			}
			
			savedTypeParams = irManager.GetCurrentFunctionTypeParams()
			typeParamMap = make(map[string]string)
			
			if len(funcTypeParamNames) > 0 && len(funcTypeParamNames) == len(extractedTypeArgs) {
				// 使用方法定义的类型参数列表建立映射
				// 例如：方法定义是 *HashMap[K, V]_insert，类型参数是 ["int", "string"]
				// 则映射为 {"K": "int", "V": "string"}
				for i, typeArg := range extractedTypeArgs {
					typeParamMap[funcTypeParamNames[i]] = typeArg
				}
			} else {
				// 根据类型参数数量确定泛型参数名（向后兼容）
				for i, typeArg := range extractedTypeArgs {
					var genericParamName string
					if len(extractedTypeArgs) == 1 {
						genericParamName = "T"
					} else if len(extractedTypeArgs) == 2 {
						if i == 0 {
							genericParamName = "K"
						} else {
							genericParamName = "V"
						}
					} else {
						genericParamName = fmt.Sprintf("T%d", i+1)
					}
					typeParamMap[genericParamName] = typeArg
				}
			}
			
			// 合并到当前类型参数映射（如果方法调用发生在泛型函数内部）
			currentTypeParams := irManager.GetCurrentFunctionTypeParams()
			if currentTypeParams == nil {
				currentTypeParams = make(map[string]string)
			}
			for k, v := range typeParamMap {
				currentTypeParams[k] = v
			}
			irManager.SetCurrentFunctionTypeParams(currentTypeParams)
			fmt.Printf("DEBUG: EvaluateMethodCallExpr - receiverType=%s, extractedTypeArgs=%v, typeParamMap=%v, merged=%v\n", receiverType, extractedTypeArgs, typeParamMap, currentTypeParams)
		}
	}

	// 求值接收者
	receiverValue, err := ee.Evaluate(irManager, expr.Receiver)
	if err != nil {
		// 恢复类型参数映射
		if savedTypeParams != nil {
			irManager.SetCurrentFunctionTypeParams(savedTypeParams)
		}
		return nil, fmt.Errorf("failed to evaluate method receiver: %w", err)
	}

	// 求值参数
	var argValues []interface{}
	for i, arg := range expr.Args {
		argValue, err := ee.Evaluate(irManager, arg)
		if err != nil {
			// 恢复类型参数映射
			if savedTypeParams != nil {
				irManager.SetCurrentFunctionTypeParams(savedTypeParams)
			}
			return nil, fmt.Errorf("failed to evaluate method argument %d: %w", i, err)
		}
		argValues = append(argValues, argValue)
	}
	
	// 尝试多种方法名格式（因为方法定义可能使用指针接收者或值接收者）
	// 1. 先尝试指针接收者：*HashMap[K, V]_insert
	// 2. 再尝试值接收者：HashMap[K, V]_insert
	// 3. 最后尝试原始类型（向后兼容）
	methodName := ""
	
	// ✅ 新增：如果是泛型方法调用且类型参数已知，延迟编译方法体
	var monoMethodName string
	if ee.statementGenerator != nil && len(extractedTypeArgs) > 0 {
		// 检查是否是泛型方法（通过检查是否有泛型方法定义）
		// 注意：genericType 可能已经包含 * 前缀（如果 eraseGenericTypeParameters 保留了它）
		// 所以我们需要检查 genericType 是否已经包含 * 前缀
		var pointerMethodName, valueMethodName string
		if strings.HasPrefix(genericType, "*") {
			// genericType 已经包含 * 前缀，直接使用
			pointerMethodName = fmt.Sprintf("%s_%s", genericType, expr.MethodName)
			// 值接收者方法名：移除 * 前缀
			valueMethodName = fmt.Sprintf("%s_%s", genericType[1:], expr.MethodName)
		} else {
			// genericType 不包含 * 前缀，需要添加
			pointerMethodName = fmt.Sprintf("*%s_%s", genericType, expr.MethodName)
			valueMethodName = fmt.Sprintf("%s_%s", genericType, expr.MethodName)
		}
		
		fmt.Printf("DEBUG: EvaluateMethodCallExpr - Looking for method: receiverType=%s, genericType=%s, method=%s\n", 
			receiverType, genericType, expr.MethodName)
		fmt.Printf("DEBUG: EvaluateMethodCallExpr - Trying method names: pointerMethodName=%s, valueMethodName=%s\n", 
			pointerMethodName, valueMethodName)
		
		// 检查是否有泛型方法定义
		methodDef, isGenericMethod := irManager.GetGenericFunctionDefinition(pointerMethodName)
		usedMethodName := pointerMethodName
		if !isGenericMethod {
			methodDef, isGenericMethod = irManager.GetGenericFunctionDefinition(valueMethodName)
			usedMethodName = valueMethodName
		}
		
		if isGenericMethod && methodDef != nil {
			fmt.Printf("DEBUG: Found generic method definition: %s\n", usedMethodName)
		} else {
			fmt.Printf("DEBUG: ERROR - Generic method definition not found: %s or %s\n", 
				pointerMethodName, valueMethodName)
		}
		
		if isGenericMethod && methodDef != nil {
			// 获取方法定义的类型参数列表（用于生成单态化方法名）
			funcTypeParamNames := irManager.GetFunctionTypeParamNames(usedMethodName)
			if len(funcTypeParamNames) == 0 {
				// 如果 usedMethodName 是 pointerMethodName，尝试 valueMethodName
				if usedMethodName == pointerMethodName {
					funcTypeParamNames = irManager.GetFunctionTypeParamNames(valueMethodName)
				} else {
					funcTypeParamNames = irManager.GetFunctionTypeParamNames(pointerMethodName)
				}
			}
			
			if len(funcTypeParamNames) > 0 {
				fmt.Printf("DEBUG: EvaluateMethodCallExpr - Got type param names: %v for method: %s\n", funcTypeParamNames, usedMethodName)
				
				// 生成单态化方法名
				// 注意：方法名需要包含接收者类型，所以需要替换接收者类型中的类型参数
				// receiverType已经是具体类型（如*HashMap[int, string]），可以直接使用
				// 但我们需要确保格式正确（包含*前缀如果是指针接收者）
				monoReceiverType := receiverType
				
				// receiverType 已经包含 * 前缀（如果是指针类型），直接使用
				monoMethodName = fmt.Sprintf("%s_%s", monoReceiverType, expr.MethodName)
				
				fmt.Printf("DEBUG: EvaluateMethodCallExpr - Generating monomorphized method name: receiverType=%s, monoMethodName=%s\n", 
					receiverType, monoMethodName)
				
				// 检查是否已经编译
				isCompiled := irManager.IsMonomorphizedFunctionCompiled(monoMethodName)
				fmt.Printf("DEBUG: EvaluateMethodCallExpr - Is monomorphized method compiled? %s: %v\n", monoMethodName, isCompiled)
				
				if !isCompiled {
					fmt.Printf("DEBUG: EvaluateMethodCallExpr - Starting to compile monomorphized method: %s\n", monoMethodName)
					// 延迟编译方法体
					// 注意：方法定义转换为函数定义时，需要调整参数（第一个参数是接收者）
					err := ee.statementGenerator.CompileMonomorphizedFunction(irManager, methodDef, typeParamMap, monoMethodName)
					if err != nil {
						fmt.Printf("DEBUG: ERROR - Failed to compile monomorphized method %s: %v\n", monoMethodName, err)
						// 恢复类型参数映射
						if savedTypeParams != nil {
							irManager.SetCurrentFunctionTypeParams(savedTypeParams)
						}
						return nil, fmt.Errorf("failed to compile monomorphized method %s: %w", monoMethodName, err)
					}
					// 标记已编译
					irManager.MarkMonomorphizedFunctionCompiled(monoMethodName)
					fmt.Printf("DEBUG: EvaluateMethodCallExpr - Successfully compiled monomorphized method %s -> %s\n", usedMethodName, monoMethodName)
				} else {
					fmt.Printf("DEBUG: EvaluateMethodCallExpr - monomorphized method already compiled: %s\n", monoMethodName)
				}
				
				// 使用单态化方法名调用
				methodName = monoMethodName
			} else {
				fmt.Printf("DEBUG: ERROR - Cannot get type param names for method: %s (tried %s and %s)\n", usedMethodName, pointerMethodName, valueMethodName)
			}
		}
	}
	
	// 如果还没有找到方法名，尝试常规查找
	if methodName == "" {
		// 尝试指针接收者格式
		// 注意：genericType 可能已经包含 * 前缀
		var pointerMethodName, valueMethodName string
		if strings.HasPrefix(genericType, "*") {
			pointerMethodName = fmt.Sprintf("%s_%s", genericType, expr.MethodName)
			valueMethodName = fmt.Sprintf("%s_%s", genericType[1:], expr.MethodName)
		} else {
			pointerMethodName = fmt.Sprintf("*%s_%s", genericType, expr.MethodName)
			valueMethodName = fmt.Sprintf("%s_%s", genericType, expr.MethodName)
		}
		
		// 值接收者方法注册为 Type_MethodName（如 Point_distance），指针接收者为 *Type_MethodName
		// 当 receiverType 无 * 前缀时优先查 valueMethodName，否则优先 pointerMethodName
		if strings.HasPrefix(genericType, "*") {
			if ee.functionExists(irManager, pointerMethodName) {
				methodName = pointerMethodName
			} else if ee.functionExists(irManager, valueMethodName) {
				methodName = valueMethodName
			}
		} else {
			if ee.functionExists(irManager, valueMethodName) {
				methodName = valueMethodName
			} else if ee.functionExists(irManager, pointerMethodName) {
				methodName = pointerMethodName
			}
		}
		if methodName == "" {
			// 尝试原始类型（向后兼容）
			originalMethodName := fmt.Sprintf("%s_%s", receiverType, expr.MethodName)
			if ee.functionExists(irManager, originalMethodName) {
				methodName = originalMethodName
			} else {
				// 特殊处理：如果接收者类型是泛型类型参数（如K），尝试从IRModuleManager获取类型参数映射
				// 例如：如果当前函数调用是bucket_index[int, string]，那么K应该映射到int
				if ee.isGenericTypeParameter(receiverType) {
					// 从IRModuleManager获取类型参数映射（在函数调用时设置）
					typeParamMap := irManager.GetCurrentFunctionTypeParams()
					// 获取当前函数名用于调试
					currentFunc := irManager.GetCurrentFunction()
					currentFuncName := "unknown"
					if fn, ok := currentFunc.(*ir.Func); ok {
						currentFuncName = fn.Name()
					}
					fmt.Printf("DEBUG: Method call - receiverType=%s (generic param), typeParamMap=%v, currentFunc=%s\n", receiverType, typeParamMap, currentFuncName)
					
					// ✅ 修复：如果typeParamMap为空，尝试从函数名提取类型参数映射
					if typeParamMap == nil || len(typeParamMap) == 0 {
						typeParamMapFromName := ee.extractTypeParamsFromCurrentFunction(irManager)
						if len(typeParamMapFromName) > 0 {
							typeParamMap = typeParamMapFromName
							fmt.Printf("DEBUG: Method call - extracted typeParamMap from func name: %v\n", typeParamMap)
						}
					}
					
					if concreteType, ok := typeParamMap[receiverType]; ok {
						// 使用实际类型查找方法（如int_hash）
						concreteMethodName := fmt.Sprintf("%s_%s", concreteType, expr.MethodName)
						if ee.functionExists(irManager, concreteMethodName) {
							methodName = concreteMethodName
						} else {
							// 调试：打印所有尝试的方法名
							fmt.Printf("DEBUG: Method call - receiverType=%s (generic param), concreteType=%s, method=%s\n", receiverType, concreteType, expr.MethodName)
							fmt.Printf("DEBUG: Tried method names: %s, %s, %s, %s\n", pointerMethodName, valueMethodName, originalMethodName, concreteMethodName)
							methodName = concreteMethodName
						}
					} else {
						// 无法从IRModuleManager获取类型参数信息，尝试从函数名提取（向后兼容）
						typeParamMapFromName := ee.extractTypeParamsFromCurrentFunction(irManager)
						if concreteType, ok := typeParamMapFromName[receiverType]; ok {
							concreteMethodName := fmt.Sprintf("%s_%s", concreteType, expr.MethodName)
							if ee.functionExists(irManager, concreteMethodName) {
								methodName = concreteMethodName
							} else {
								fmt.Printf("DEBUG: Method call - receiverType=%s (generic param), concreteType=%s (from func name), method=%s\n", receiverType, concreteType, expr.MethodName)
								fmt.Printf("DEBUG: Tried method names: %s, %s, %s, %s\n", pointerMethodName, valueMethodName, originalMethodName, concreteMethodName)
								methodName = concreteMethodName
							}
						} else {
							// 无法获取类型参数信息，使用泛型定义类型
							fmt.Printf("DEBUG: Method call - receiverType=%s (generic param), genericType=%s, method=%s\n", receiverType, genericType, expr.MethodName)
							fmt.Printf("DEBUG: Tried method names: %s, %s, %s\n", pointerMethodName, valueMethodName, originalMethodName)
							fmt.Printf("DEBUG: Warning - cannot get type params from IRModuleManager or function name, using generic type\n")
							methodName = valueMethodName
						}
					}
				} else {
					// 如果都不存在，使用泛型定义类型（让CreateCall报错，提供更清晰的错误信息）
					// 调试：打印所有尝试的方法名
					fmt.Printf("DEBUG: Method call - receiverType=%s, genericType=%s, method=%s\n", receiverType, genericType, expr.MethodName)
					fmt.Printf("DEBUG: Tried method names: %s, %s, %s\n", pointerMethodName, valueMethodName, originalMethodName)
					methodName = valueMethodName
				}
			}
		}
	}

	// 调用方法（在LLVM IR中，方法调用被转换为普通函数调用）
	// 调试：打印参数类型
	fmt.Printf("DEBUG: Method call - methodName=%s, receiverValue type=%T, argValues count=%d\n", methodName, receiverValue, len(argValues))
	for i, arg := range argValues {
		fmt.Printf("DEBUG:   arg[%d] type=%T\n", i, arg)
	}
	
	// 构建参数列表（接收者 + 方法参数）
	allArgs := append([]interface{}{receiverValue}, argValues...)
	// 使用...展开slice为可变参数
	result, err := irManager.CreateCall(methodName, allArgs...)
	
	// 恢复类型参数映射
	if savedTypeParams != nil {
		irManager.SetCurrentFunctionTypeParams(savedTypeParams)
	}
	
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

// EvaluateFloatLiteral 求值浮点数字面量
func (ee *ExpressionEvaluatorImpl) EvaluateFloatLiteral(irManager generation.IRModuleManager, expr *entities.FloatLiteral) (interface{}, error) {
	// 在LLVM IR中，浮点数使用double类型（f64）
	// FloatLiteral.Value 是 float64 类型
	return constant.NewFloat(types.Double, expr.Value), nil
}

// EvaluateResultExpr 求值Result表达式
func (ee *ExpressionEvaluatorImpl) EvaluateResultExpr(irManager generation.IRModuleManager, expr *entities.ResultExpr) (interface{}, error) {
	// Result[T, E] 是一种特殊的泛型类型，目前返回一个占位符
	// 实际实现需要运行时库支持
	fmt.Printf("DEBUG: Result type evaluation - OkType: %s, ErrType: %s\n", expr.OkType, expr.ErrType)
	return constant.NewInt(types.I32, 0), nil
}

// EvaluateOkLiteral 求值Ok字面量，构造 Result 结构体 { tag=0, data=payload_ptr }，返回 *{ i8, i8* }。
func (ee *ExpressionEvaluatorImpl) EvaluateOkLiteral(irManager generation.IRModuleManager, expr *entities.OkLiteral) (interface{}, error) {
	resultStructType := types.NewStruct(types.I8, types.NewPointer(types.I8))
	resultPtr, err := irManager.CreateAlloca(resultStructType, "result_ok")
	if err != nil {
		return nil, fmt.Errorf("alloca result struct: %w", err)
	}
	resultPtrValue, _ := resultPtr.(llvalue.Value)

	// tag = 0 (Ok)
	tagPtr, err := irManager.CreateGetElementPtr(resultStructType, resultPtr, constant.NewInt(types.I32, 0), constant.NewInt(types.I32, 0))
	if err != nil {
		return nil, fmt.Errorf("gep tag: %w", err)
	}
	if err = irManager.CreateStore(constant.NewInt(types.I8, 0), tagPtr); err != nil {
		return nil, fmt.Errorf("store tag: %w", err)
	}

	var dataPtrVal llvalue.Value
	if expr.Value != nil {
		valueResult, err := ee.Evaluate(irManager, expr.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate Ok value: %w", err)
		}
		dataPtrVal, err = ee.payloadToI8Ptr(irManager, valueResult)
		if err != nil {
			return nil, fmt.Errorf("Ok payload to i8*: %w", err)
		}
	} else {
		// Ok() 无参：data 存 null
		dataPtrVal = constant.NewNull(types.NewPointer(types.I8))
	}

	dataPtr, err := irManager.CreateGetElementPtr(resultStructType, resultPtr, constant.NewInt(types.I32, 0), constant.NewInt(types.I32, 1))
	if err != nil {
		return nil, fmt.Errorf("gep data: %w", err)
	}
	if err = irManager.CreateStore(dataPtrVal, dataPtr); err != nil {
		return nil, fmt.Errorf("store data: %w", err)
	}
	return resultPtrValue, nil
}

// EvaluateErrLiteral 求值Err字面量，构造 Result 结构体 { tag=1, data=payload_ptr }，返回 *{ i8, i8* }。
func (ee *ExpressionEvaluatorImpl) EvaluateErrLiteral(irManager generation.IRModuleManager, expr *entities.ErrLiteral) (interface{}, error) {
	resultStructType := types.NewStruct(types.I8, types.NewPointer(types.I8))
	resultPtr, err := irManager.CreateAlloca(resultStructType, "result_err")
	if err != nil {
		return nil, fmt.Errorf("alloca result struct: %w", err)
	}
	resultPtrValue, _ := resultPtr.(llvalue.Value)

	// tag = 1 (Err)
	tagPtr, err := irManager.CreateGetElementPtr(resultStructType, resultPtr, constant.NewInt(types.I32, 0), constant.NewInt(types.I32, 0))
	if err != nil {
		return nil, fmt.Errorf("gep tag: %w", err)
	}
	if err = irManager.CreateStore(constant.NewInt(types.I8, 1), tagPtr); err != nil {
		return nil, fmt.Errorf("store tag: %w", err)
	}

	var dataPtrVal llvalue.Value
	if expr.Error != nil {
		valueResult, err := ee.Evaluate(irManager, expr.Error)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate Err value: %w", err)
		}
		dataPtrVal, err = ee.payloadToI8Ptr(irManager, valueResult)
		if err != nil {
			return nil, fmt.Errorf("Err payload to i8*: %w", err)
		}
	} else {
		dataPtrVal = constant.NewNull(types.NewPointer(types.I8))
	}

	dataPtr, err := irManager.CreateGetElementPtr(resultStructType, resultPtr, constant.NewInt(types.I32, 0), constant.NewInt(types.I32, 1))
	if err != nil {
		return nil, fmt.Errorf("gep data: %w", err)
	}
	if err = irManager.CreateStore(dataPtrVal, dataPtr); err != nil {
		return nil, fmt.Errorf("store data: %w", err)
	}
	return resultPtrValue, nil
}

// payloadToI8Ptr 将求值结果转为 i8*：若已是指针则 BitCast，否则 alloca+Store 再 BitCast。
func (ee *ExpressionEvaluatorImpl) payloadToI8Ptr(irManager generation.IRModuleManager, valueResult interface{}) (llvalue.Value, error) {
	val, ok := valueResult.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("value is not llvm value: %T", valueResult)
	}
	t := val.Type()
	if ptrType, ok := t.(*types.PointerType); ok && ptrType.ElemType == types.I8 {
		return val, nil
	}
	if _, ok := t.(*types.PointerType); ok {
		cast, err := irManager.CreateBitCast(val, types.NewPointer(types.I8), "payload_i8")
		if err != nil {
			return nil, err
		}
		c, _ := cast.(llvalue.Value)
		return c, nil
	}
	// 标量或聚合：alloca + store + 返回 alloca 转 i8*
	alloca, err := irManager.CreateAlloca(t, "payload_slot")
	if err != nil {
		return nil, err
	}
	if err = irManager.CreateStore(valueResult, alloca); err != nil {
		return nil, err
	}
	cast, err := irManager.CreateBitCast(alloca, types.NewPointer(types.I8), "payload_i8")
	if err != nil {
		return nil, err
	}
	c, _ := cast.(llvalue.Value)
	return c, nil
}

// EvaluateNoneLiteral 求值None字面量
// None在Echo语言中表示空值（类似Rust的()或Go的nil）
func (ee *ExpressionEvaluatorImpl) EvaluateNoneLiteral(irManager generation.IRModuleManager, expr *entities.NoneLiteral) (interface{}, error) {
	// None字面量在LLVM IR中通常表示为void类型或null指针
	// 根据上下文，可能需要返回不同的值
	// 对于return语句，如果函数返回void，返回nil；否则返回对应类型的零值
	currentFuncInterface := irManager.GetCurrentFunction()
	if currentFuncInterface != nil {
		if currentFunc, ok := currentFuncInterface.(*ir.Func); ok {
			if currentFunc.Sig.RetType == types.Void {
				// 函数返回void，None表示无返回值
				return nil, nil
			} else {
				// 函数有返回值，None表示返回零值
				retType := currentFunc.Sig.RetType
				if retType == types.I32 {
					return constant.NewInt(types.I32, 0), nil
				} else if ptrType, ok := retType.(*types.PointerType); ok {
					// 返回类型是指针，None表示null指针
					return constant.NewNull(ptrType), nil
				} else {
					// 其他类型，返回零值（需要根据类型创建）
					return constant.NewInt(types.I32, 0), nil
				}
			}
		}
	}
	// 默认返回i32的0值
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

	case "len":
		// array.len() -> int - 返回数组长度
		if len(expr.Args) != 0 {
			return nil, fmt.Errorf("len method expects 0 arguments, got %d", len(expr.Args))
		}
		// 将 ArrayMethodCallExpr 转换为 LenExpr 来处理
		lenExpr := &entities.LenExpr{
			Array: expr.Array,
		}
		return ee.EvaluateLenExpr(irManager, lenExpr)

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

// EvaluateStructAccess 求值结构体字段访问表达式
// 支持从结构体值或结构体指针访问字段
// 例如：result.success, result.scheme, (*result).host
func (ee *ExpressionEvaluatorImpl) EvaluateStructAccess(irManager generation.IRModuleManager, expr *entities.StructAccess) (interface{}, error) {
	// 1. 求值对象表达式（可能是标识符、函数调用等）
	objectValue, err := ee.Evaluate(irManager, expr.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate struct object: %w", err)
	}

	// 2. 获取对象类型（从符号表、函数调用或类型推断）
	var objectType string
	var isPointer bool

	// 尝试从标识符获取类型
	if ident, ok := expr.Object.(*entities.Identifier); ok {
		symbol, err := ee.symbolManager.LookupSymbol(ident.Name)
		if err == nil {
			objectType = symbol.Type
			
			// ✅ 新增：如果当前函数有类型参数映射，替换类型参数
			// 例如：在单态化方法 HashMap[int, string]_insert 中，m 的类型应该是 *HashMap[int, string]
			// 而不是 *HashMap[K, V]
			typeParamMap := irManager.GetCurrentFunctionTypeParams()
			if len(typeParamMap) > 0 && strings.Contains(objectType, "[") {
				// 替换类型参数：HashMap[K, V] -> HashMap[int, string]
				// 注意：这里需要替换类型参数，但保留指针前缀
				hasPointerPrefix := strings.HasPrefix(objectType, "*")
				baseType := objectType
				if hasPointerPrefix {
					baseType = objectType[1:] // 移除 * 前缀
				}
				
				// 提取类型参数并替换
				if bracketIndex := strings.Index(baseType, "["); bracketIndex != -1 {
					baseTypeName := baseType[:bracketIndex]
					typeArgsStr := baseType[bracketIndex+1 : len(baseType)-1]
					typeArgs := strings.Split(typeArgsStr, ",")
					
					// 替换每个类型参数
					substitutedArgs := make([]string, len(typeArgs))
					for i, arg := range typeArgs {
						arg = strings.TrimSpace(arg)
						if substituted, ok := typeParamMap[arg]; ok {
							substitutedArgs[i] = substituted
						} else {
							substitutedArgs[i] = arg // 保持原样（可能是具体类型）
						}
					}
					
					// 重新构建类型
					newBaseType := baseTypeName + "[" + strings.Join(substitutedArgs, ", ") + "]"
					if hasPointerPrefix {
						objectType = "*" + newBaseType
					} else {
						objectType = newBaseType
					}
					
					fmt.Printf("DEBUG: EvaluateStructAccess - substituted type: %s -> %s (typeParamMap=%v)\n", symbol.Type, objectType, typeParamMap)
				}
			}
			
			// 检查是否是指针类型
			if strings.HasPrefix(objectType, "*") {
				isPointer = true
				// 移除指针符号，获取基础类型
				objectType = strings.TrimPrefix(objectType, "*")
			} else {
				// 检查LLVM类型是否是指针
				typeInterface, err := ee.typeMapper.MapType(symbol.Type)
				if err == nil {
					if llvmType, ok := typeInterface.(types.Type); ok {
						if _, ok := llvmType.(*types.PointerType); ok {
							isPointer = true
						}
					}
				}
			}
		}
	} else if funcCall, ok := expr.Object.(*entities.FuncCall); ok {
		// 对象是函数调用，尝试从函数名推断返回类型
		// 运行时函数返回结构体指针的映射
		runtimeFunctionReturnTypes := map[string]string{
			"runtime_url_parse":           "*UrlParseResult",
			"runtime_time_parse":         "*TimeParseResult",
			"runtime_string_to_bytes":     "*StringToBytesResult",
			"runtime_string_split":        "*StringSplitResult",
			"runtime_http_parse_request":   "*HttpParseRequestResult",
			"runtime_http_build_response": "*HttpBuildResponseResult",
			"runtime_map_get_keys":        "*MapIterResult",
		}
		
		if returnType, exists := runtimeFunctionReturnTypes[funcCall.Name]; exists {
			objectType = returnType
			isPointer = true
			// 移除指针符号
			if strings.HasPrefix(objectType, "*") {
				objectType = strings.TrimPrefix(objectType, "*")
			}
		}
	}

	// 3. 提取基础类型名（移除类型参数，如 HashMap[int, string] -> HashMap）
	baseTypeName := objectType
	if bracketIndex := strings.Index(objectType, "["); bracketIndex != -1 {
		baseTypeName = objectType[:bracketIndex]
	}
	
	fmt.Printf("DEBUG: EvaluateStructAccess - objectType=%s, baseTypeName=%s, field=%s\n", objectType, baseTypeName, expr.Field)
	fmt.Printf("DEBUG: EvaluateStructAccess - structDefinitions keys: %v\n", func() []string {
		keys := make([]string, 0, len(ee.structDefinitions))
		for k := range ee.structDefinitions {
			keys = append(keys, k)
		}
		return keys
	}())
	
	// 4. 查找结构体定义（使用基础类型名）
	structDef, exists := ee.structDefinitions[baseTypeName]
	if !exists {
		fmt.Printf("DEBUG: EvaluateStructAccess - struct definition not found for baseTypeName=%s, trying runtime struct access\n", baseTypeName)
		// 如果找不到结构体定义，可能是运行时函数返回的不透明指针
		// 这种情况下，我们需要特殊处理（使用字段偏移）
		// 但是，对于用户定义的结构体（如 HashMap），不应该使用运行时结构体访问
		// 应该返回错误，提示结构体定义未找到
		return nil, fmt.Errorf("struct definition not found for type %s (base type: %s). Available structs: %v", objectType, baseTypeName, func() []string {
			keys := make([]string, 0, len(ee.structDefinitions))
			for k := range ee.structDefinitions {
				keys = append(keys, k)
			}
			return keys
		}())
	}
	
	fmt.Printf("DEBUG: EvaluateStructAccess - found struct definition for baseTypeName=%s\n", baseTypeName)

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

	// 6. 获取字段类型（需要替换类型参数）
	fieldTypeStr := structDef.Fields[fieldIndex].Type
	// 如果 objectType 是泛型实例化类型（如 HashMap[int, string]），需要替换字段类型中的类型参数
	// 例如：字段类型是 []Bucket[K, V]，需要替换为 []Bucket[int, string]
	if bracketIndex := strings.Index(objectType, "["); bracketIndex != -1 {
		// 提取类型参数：HashMap[int, string] -> ["int", "string"]
		typeArgsStr := objectType[bracketIndex+1 : len(objectType)-1]
		typeArgs := strings.Split(typeArgsStr, ",")
		for i := range typeArgs {
			typeArgs[i] = strings.TrimSpace(typeArgs[i])
		}
		// 提取基础类型名中的类型参数名：HashMap[K, V] -> ["K", "V"]
		// 注意：这里需要从结构体定义中获取类型参数名，但结构体定义可能没有类型参数信息
		// 简化处理：假设类型参数名是 K, V（对于 HashMap）
		// TODO: 从结构体定义中获取类型参数名
		if baseTypeName == "HashMap" && len(typeArgs) == 2 {
			// 替换字段类型中的类型参数
			fieldTypeStr = strings.ReplaceAll(fieldTypeStr, "K", typeArgs[0])
			fieldTypeStr = strings.ReplaceAll(fieldTypeStr, "V", typeArgs[1])
		}
	}
	fieldLLVMTypeInterface, err := ee.typeMapper.MapType(fieldTypeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to map field type %s: %w", fieldTypeStr, err)
	}
	fieldLLVMType, ok := fieldLLVMTypeInterface.(types.Type)
	if !ok {
		return nil, fmt.Errorf("field type %s is not a valid LLVM type", fieldTypeStr)
	}

	// 7. 处理指针类型：如果对象是指针，需要先解引用
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
			symbol, err := ee.symbolManager.LookupSymbol(ident.Name)
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

	// 8. 构建结构体LLVM类型（用于GEP）
	// 需要根据结构体定义构建LLVM结构体类型
	// 注意：对于泛型结构体，字段类型可能包含类型参数，需要替换
	structLLVMType, err := ee.buildStructLLVMType(structDef)
	if err != nil {
		return nil, fmt.Errorf("failed to build struct LLVM type: %w", err)
	}

	// 9. 使用 GetElementPtr 获取字段指针
	fieldIndexConst := constant.NewInt(types.I32, int64(fieldIndex))
	fieldPtr, err := irManager.CreateGetElementPtr(structLLVMType, structPtr, fieldIndexConst)
	if err != nil {
		return nil, fmt.Errorf("failed to create GEP for field access: %w", err)
	}

	// 10. 加载字段值
	fieldValue, err := irManager.CreateLoad(fieldLLVMType, fieldPtr, fmt.Sprintf("%s_%s", objectType, expr.Field))
	if err != nil {
		return nil, fmt.Errorf("failed to load field value: %w", err)
	}

	return fieldValue, nil
}

// buildStructLLVMType 根据结构体定义构建LLVM结构体类型
func (ee *ExpressionEvaluatorImpl) buildStructLLVMType(structDef *entities.StructDef) (types.Type, error) {
	fieldTypes := make([]types.Type, len(structDef.Fields))
	for i, field := range structDef.Fields {
		typeInterface, err := ee.typeMapper.MapType(field.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to map field type %s: %w", field.Type, err)
		}
		llvmType, ok := typeInterface.(types.Type)
		if !ok {
			return nil, fmt.Errorf("field type %s is not a valid LLVM type", field.Type)
		}
		fieldTypes[i] = llvmType
	}
	return types.NewStruct(fieldTypes...), nil
}

// evaluateRuntimeStructFieldAccess 处理运行时函数返回的结构体指针字段访问
// 运行时函数返回的是 *i8（不透明指针），我们需要使用字段偏移来访问
func (ee *ExpressionEvaluatorImpl) evaluateRuntimeStructFieldAccess(irManager generation.IRModuleManager, objectValue interface{}, fieldName string, structTypeName string) (interface{}, error) {
	// 运行时结构体字段偏移映射
	// 这些偏移值需要与C运行时结构体定义保持一致
	// 注意：偏移值考虑了结构体对齐规则（64位系统）
	// - bool: 1字节，对齐到1字节
	// - int32_t: 4字节，对齐到4字节
	// - int64_t: 8字节，对齐到8字节
	// - char*: 8字节（64位系统），对齐到8字节
	runtimeStructFieldOffsets := map[string]map[string]int{
		"UrlParseResult": {
			// C结构体定义顺序：scheme, host, port, path, query, fragment, success
			"scheme":   0,  // char* (8 bytes)
			"host":     8,  // char* (8 bytes)
			"port":     16, // int32_t (4 bytes，对齐到4字节)
			"path":     24, // char* (8 bytes，需要8字节对齐，所以是24而不是20)
			"query":    32, // char* (8 bytes)
			"fragment": 40, // char* (8 bytes)
			"success":  48, // bool (1 byte，对齐到1字节)
		},
		"TimeParseResult": {
			// C结构体定义顺序：success, sec, nsec
			"success": 0,  // bool (1 byte)
			"sec":     8,  // int64_t (8 bytes，需要8字节对齐)
			"nsec":    16, // int32_t (4 bytes，对齐到4字节)
		},
		"StringToBytesResult": {
			// C结构体定义顺序：bytes, len
			"bytes": 0, // uint8_t* (8 bytes)
			"len":   8, // int32_t (4 bytes，对齐到4字节)
		},
		"StringSplitResult": {
			// C结构体定义顺序：strings, count
			"strings": 0, // char** (8 bytes)
			"count":   8, // int32_t (4 bytes，对齐到4字节)
		},
		"HttpParseRequestResult": {
			// C结构体定义顺序：method, path, version, header_keys, header_values, header_count, body, body_len, success
			"method":       0,  // char* (8 bytes)
			"path":         8,  // char* (8 bytes)
			"version":      16, // char* (8 bytes)
			"header_keys":  24, // char** (8 bytes)
			"header_values": 32, // char** (8 bytes)
			"header_count": 40, // int32_t (4 bytes，对齐到4字节)
			"body":         48, // uint8_t* (8 bytes，需要8字节对齐，所以是48而不是44)
			"body_len":     56, // int32_t (4 bytes，对齐到4字节)
			"success":      60, // bool (1 byte，对齐到1字节)
		},
		"HttpBuildResponseResult": {
			// C结构体定义顺序：success, data, data_len
			"success":  0,  // bool (1 byte)
			"data":     8,  // uint8_t* (8 bytes，需要8字节对齐)
			"data_len": 16, // int32_t (4 bytes，对齐到4字节)
		},
		"MapIterResult": {
			// C结构体定义顺序：keys, values, count
			"keys":   0,  // char** (8 bytes)
			"values": 8,  // char** (8 bytes)
			"count":  16, // int32_t (4 bytes，对齐到4字节)
		},
	}

	// 查找字段偏移
	fieldOffsets, exists := runtimeStructFieldOffsets[structTypeName]
	if !exists {
		return nil, fmt.Errorf("unknown runtime struct type: %s", structTypeName)
	}

	offset, exists := fieldOffsets[fieldName]
	if !exists {
		return nil, fmt.Errorf("field %s not found in runtime struct %s", fieldName, structTypeName)
	}

	// 获取对象指针（运行时函数返回的是 *i8）
	objectPtr, ok := objectValue.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("object value is not a valid LLVM value")
	}

	// 根据字段名推断字段类型
	fieldLLVMType, err := ee.inferRuntimeStructFieldType(structTypeName, fieldName)
	if err != nil {
		return nil, fmt.Errorf("failed to infer field type: %w", err)
	}

	// 使用指针算术获取字段地址
	// 1. 将 *i8 转换为 i8*（objectPtr 已经是 *i8）
	// 2. 使用 GEP 获取字段地址（通过字节偏移）
	// 注意：GEP 的索引单位是元素大小，所以对于 i8，偏移就是字节数
	offsetConst := constant.NewInt(types.I64, int64(offset))
	fieldPtr, err := irManager.CreateGetElementPtr(types.I8, objectPtr, offsetConst)
	if err != nil {
		return nil, fmt.Errorf("failed to create GEP for field offset: %w", err)
	}

	// 3. 将 i8* 转换为字段类型指针
	fieldPtrType := types.NewPointer(fieldLLVMType)
	fieldPtrCast, err := irManager.CreateBitCast(fieldPtr, fieldPtrType, fmt.Sprintf("field_%s_ptr", fieldName))
	if err != nil {
		return nil, fmt.Errorf("failed to cast field pointer: %w", err)
	}

	// 4. 加载字段值
	fieldValue, err := irManager.CreateLoad(fieldLLVMType, fieldPtrCast, fmt.Sprintf("%s_%s", structTypeName, fieldName))
	if err != nil {
		return nil, fmt.Errorf("failed to load field value: %w", err)
	}

	return fieldValue, nil
}

// inferRuntimeStructFieldType 推断运行时结构体字段的LLVM类型
func (ee *ExpressionEvaluatorImpl) inferRuntimeStructFieldType(structTypeName string, fieldName string) (types.Type, error) {
	// 运行时结构体字段类型映射
	runtimeStructFieldTypes := map[string]map[string]types.Type{
		"UrlParseResult": {
			"success":  types.I1,  // bool
			"scheme":   types.NewPointer(types.I8), // char*
			"host":     types.NewPointer(types.I8), // char*
			"port":     types.I32,  // int32_t
			"path":     types.NewPointer(types.I8), // char*
			"query":    types.NewPointer(types.I8), // char*
			"fragment": types.NewPointer(types.I8), // char*
		},
		"TimeParseResult": {
			"success": types.I1,  // bool
			"sec":     types.I64,  // int64_t
			"nsec":    types.I32,  // int32_t
		},
		"StringToBytesResult": {
			"bytes": types.NewPointer(types.I8), // uint8_t*
			"len":   types.I32,                  // int32_t
		},
		"StringSplitResult": {
			"strings": types.NewPointer(types.NewPointer(types.I8)), // char**
			"count":   types.I32,                                    // int32_t
		},
		"HttpParseRequestResult": {
			"success":       types.I1,                                    // bool
			"method":        types.NewPointer(types.I8),                   // char*
			"path":          types.NewPointer(types.I8),                   // char*
			"version":       types.NewPointer(types.I8),                   // char*
			"header_keys":   types.NewPointer(types.NewPointer(types.I8)), // char**
			"header_values": types.NewPointer(types.NewPointer(types.I8)), // char**
			"header_count":  types.I32,                                    // int32_t
			"body":          types.NewPointer(types.I8),                   // uint8_t*
			"body_len":      types.I32,                                    // int32_t
		},
		"HttpBuildResponseResult": {
			"success":  types.I1,                  // bool
			"data":     types.NewPointer(types.I8), // uint8_t*
			"data_len": types.I32,                 // int32_t
		},
		"MapIterResult": {
			"keys":   types.NewPointer(types.NewPointer(types.I8)), // char**
			"values": types.NewPointer(types.NewPointer(types.I8)), // char**
			"count":   types.I32,                                    // int32_t
		},
	}

	fieldTypes, exists := runtimeStructFieldTypes[structTypeName]
	if !exists {
		return nil, fmt.Errorf("unknown runtime struct type: %s", structTypeName)
	}

	fieldType, exists := fieldTypes[fieldName]
	if !exists {
		return nil, fmt.Errorf("field %s not found in runtime struct %s", fieldName, structTypeName)
	}

	return fieldType, nil
}

// EvaluateStructLiteral 求值结构体字面量
// 注意：map字面量 `{}` 也可能被解析为 StructLiteral（Type为空，Fields为空）
// 对于map类型，我们需要返回一个指向map结构的指针（i8*）
func (ee *ExpressionEvaluatorImpl) EvaluateStructLiteral(irManager generation.IRModuleManager, expr *entities.StructLiteral) (interface{}, error) {
	// 检查是否是空map字面量：Type为空或Type是map类型，Fields为空
	if expr.Type == "" && len(expr.Fields) == 0 {
		// 这可能是空map字面量 `{}`。依赖说明：真实空 map 需运行时提供创建空 map 的函数（如 runtime_map_new 或按 key/value 类型区分的创建函数）；当前用 runtime_null_ptr 作为占位，仅保证可编译。
		nullPtrFunc, exists := irManager.GetExternalFunction("runtime_null_ptr")
		if !exists {
			return nil, fmt.Errorf("runtime_null_ptr function not declared")
		}
		nullPtr, err := irManager.CreateCall(nullPtrFunc)
		if err != nil {
			return nil, fmt.Errorf("failed to call runtime_null_ptr: %w", err)
		}
		return nullPtr, nil
	}
	
	// 对于非空的结构体字面量，需要根据结构体类型创建结构体实例
	// 1. 获取结构体类型名（如 Vec[T]）
	structTypeName := expr.Type
	
	// 2. 提取基础类型名（移除类型参数）
	// 例如：Vec[T] -> Vec
	baseTypeName := structTypeName
	if bracketIndex := strings.Index(structTypeName, "["); bracketIndex != -1 {
		baseTypeName = structTypeName[:bracketIndex]
	}
	
	// 3. 从 structDefinitions 获取结构体定义（使用基础类型名）
	structDef, exists := ee.structDefinitions[baseTypeName]
	if !exists {
		// 调试：打印所有可用的结构体定义键
		availableKeys := make([]string, 0, len(ee.structDefinitions))
		for k := range ee.structDefinitions {
			availableKeys = append(availableKeys, k)
		}
		return nil, fmt.Errorf("struct definition not found for type %s (base type: %s, available keys: %v)", structTypeName, baseTypeName, availableKeys)
	}
	
	// 3. 构建LLVM结构体类型
	structLLVMType, err := ee.buildStructLLVMType(structDef)
	if err != nil {
		return nil, fmt.Errorf("failed to build struct LLVM type for %s: %w", structTypeName, err)
	}
	
	// 4. 使用 alloca 分配内存
	structPtr, err := irManager.CreateAlloca(structLLVMType, fmt.Sprintf("%s_literal", structTypeName))
	if err != nil {
		return nil, fmt.Errorf("failed to allocate memory for struct %s: %w", structTypeName, err)
	}
	
	structPtrValue, ok := structPtr.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("CreateAlloca returned invalid type: %T", structPtr)
	}
	
	// 5. 为每个字段求值并存储
	// 注意：expr.Fields 是 map[string]Expr，需要按照结构体定义的字段顺序处理
	for i, field := range structDef.Fields {
		fieldName := field.Name
		fieldValueExpr, exists := expr.Fields[fieldName]
		if !exists {
			// 字段未提供，使用默认值
			// TODO: 根据字段类型获取默认值
			continue
		}
		
		// 求值字段值
		fieldValue, err := ee.Evaluate(irManager, fieldValueExpr)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate field %s.%s: %w", structTypeName, fieldName, err)
		}
		
		// 获取字段LLVM类型（用于类型检查，暂时不强制检查）
		_, err = ee.typeMapper.MapType(field.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to map field type %s.%s (%s): %w", structTypeName, fieldName, field.Type, err)
		}
		
		// 使用 GEP 获取字段指针
		// GEP 需要两个索引：0（结构体本身），i（字段索引）
		fieldIndexConst := constant.NewInt(types.I32, int64(i))
		fieldPtr, err := irManager.CreateGetElementPtr(structLLVMType, structPtrValue, constant.NewInt(types.I32, 0), fieldIndexConst)
		if err != nil {
			return nil, fmt.Errorf("failed to create GEP for field %s.%s: %w", structTypeName, fieldName, err)
		}
		
		fieldPtrValue, ok := fieldPtr.(llvalue.Value)
		if !ok {
			return nil, fmt.Errorf("CreateGetElementPtr returned invalid type: %T", fieldPtr)
		}
		
		// 存储字段值
		fieldValueLLVM, ok := fieldValue.(llvalue.Value)
		if !ok {
			// 如果字段值是常量，需要转换为LLVM值
			if constValue, ok := fieldValue.(*constant.Int); ok {
				fieldValueLLVM = constValue
			} else {
				return nil, fmt.Errorf("field value for %s.%s is not a valid LLVM value: %T", structTypeName, fieldName, fieldValue)
			}
		}
		
		err = irManager.CreateStore(fieldValueLLVM, fieldPtrValue)
		if err != nil {
			return nil, fmt.Errorf("failed to store field %s.%s: %w", structTypeName, fieldName, err)
		}
	}
	
	// 6. 返回结构体指针
	// 注意：对于返回语句，如果返回类型是值类型，需要在返回语句中加载值
	return structPtrValue, nil
}

// eraseGenericTypeParameters 将泛型实例化类型转换为泛型定义类型
// 例如：HashMap[int, string] -> HashMap[K, V]
//      Vec[int] -> Vec[T]
//      Option[string] -> Option[T]
func (ee *ExpressionEvaluatorImpl) eraseGenericTypeParameters(typeName string) string {
	// 检查是否包含泛型参数（包含 [ 和 ]）
	if !strings.Contains(typeName, "[") || !strings.Contains(typeName, "]") {
		return typeName // 不是泛型类型，直接返回
	}

	// 处理指针类型（如 *HashMap[int, string]）
	isPointer := false
	if strings.HasPrefix(typeName, "*") {
		isPointer = true
		typeName = strings.TrimPrefix(typeName, "*")
	}

	// 找到泛型参数开始位置
	openBracket := strings.Index(typeName, "[")
	if openBracket == -1 {
		return typeName
	}

	// 提取基础类型名（如 HashMap）
	baseTypeName := typeName[:openBracket]

	// 提取泛型参数部分（如 int, string）
	genericParams := typeName[openBracket+1 : len(typeName)-1]

	// 计算泛型参数数量（通过逗号分隔）
	paramCount := strings.Count(genericParams, ",") + 1

	// 根据参数数量生成泛型定义类型参数
	// 单参数：T
	// 双参数：K, V
	// 三参数：T1, T2, T3
	var genericDefParams []string
	if paramCount == 1 {
		genericDefParams = []string{"T"}
	} else if paramCount == 2 {
		genericDefParams = []string{"K", "V"}
	} else {
		// 多个参数：T1, T2, T3, ...
		for i := 0; i < paramCount; i++ {
			genericDefParams = append(genericDefParams, fmt.Sprintf("T%d", i+1))
		}
	}

	// 构建泛型定义类型
	genericDefType := baseTypeName + "[" + strings.Join(genericDefParams, ", ") + "]"

	// 如果原来是指针类型，添加 * 前缀
	if isPointer {
		genericDefType = "*" + genericDefType
	}

	return genericDefType
}

// functionExists 检查函数是否存在于IR模块中
func (ee *ExpressionEvaluatorImpl) functionExists(irManager generation.IRModuleManager, funcName string) bool {
	_, exists := irManager.GetFunction(funcName)
	if !exists {
		// 调试：打印尝试查找的函数名
		fmt.Printf("DEBUG: functionExists - function %s not found\n", funcName)
	}
	return exists
}

// isGenericTypeParameter 检查类型是否是泛型类型参数（如K、V、T等）
// 泛型类型参数通常是单个大写字母或T1、T2等格式
func (ee *ExpressionEvaluatorImpl) isGenericTypeParameter(typeName string) bool {
	// 移除指针前缀
	typeName = strings.TrimPrefix(typeName, "*")
	
	// 检查是否是单个大写字母（如K、V、T）
	if len(typeName) == 1 && typeName[0] >= 'A' && typeName[0] <= 'Z' {
		return true
	}
	
	// 检查是否是T1、T2等格式
	if len(typeName) >= 2 && typeName[0] == 'T' {
		// 检查后面是否都是数字
		allDigits := true
		for i := 1; i < len(typeName); i++ {
			if typeName[i] < '0' || typeName[i] > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return true
		}
	}
	
	return false
}

// extractTypeParamsFromCurrentFunction 从当前函数名提取类型参数映射
// 例如：如果当前函数名是"bucket_index[int, string]"，返回map[string]string{"K": "int", "V": "string"}
// 如果函数名不包含类型参数，返回空map
func (ee *ExpressionEvaluatorImpl) extractTypeParamsFromCurrentFunction(irManager generation.IRModuleManager) map[string]string {
	typeParamMap := make(map[string]string)
	
	// 获取当前函数
	currentFunc := irManager.GetCurrentFunction()
	if currentFunc == nil {
		return typeParamMap
	}
	
	// 类型断言为*ir.Func
	llvmFunc, ok := currentFunc.(*ir.Func)
	if !ok {
		return typeParamMap
	}
	
	// 获取函数名
	funcName := llvmFunc.Name()
	
	// 检查函数名是否包含类型参数（包含[和]）
	if !strings.Contains(funcName, "[") || !strings.Contains(funcName, "]") {
		return typeParamMap
	}
	
	// 提取类型参数部分
	openBracket := strings.Index(funcName, "[")
	closeBracket := strings.LastIndex(funcName, "]")
	if openBracket == -1 || closeBracket == -1 || closeBracket <= openBracket {
		return typeParamMap
	}
	
	// 提取类型参数部分（如int, string）
	typeParamsStr := funcName[openBracket+1 : closeBracket]
	
	// 分割类型参数
	typeParams := strings.Split(typeParamsStr, ",")
	for i, param := range typeParams {
		param = strings.TrimSpace(param)
		if param == "" {
			continue
		}
		
		// 根据位置确定泛型类型参数名
		// 单参数：T
		// 双参数：K, V
		// 三参数：T1, T2, T3
		var genericParamName string
		if len(typeParams) == 1 {
			genericParamName = "T"
		} else if len(typeParams) == 2 {
			if i == 0 {
				genericParamName = "K"
			} else {
				genericParamName = "V"
			}
		} else {
			genericParamName = fmt.Sprintf("T%d", i+1)
		}
		
		// 建立映射
		typeParamMap[genericParamName] = param
	}
	
	fmt.Printf("DEBUG: extractTypeParamsFromCurrentFunction - funcName=%s, typeParamMap=%v\n", funcName, typeParamMap)
	return typeParamMap
}

// getArgType 获取参数的类型
// 从符号表或表达式推断参数类型
// ✅ 修复：如果返回的类型包含泛型类型参数，使用当前类型参数映射替换
func (ee *ExpressionEvaluatorImpl) getArgType(irManager generation.IRModuleManager, arg entities.Expr) string {
	var argType string
	
	// 如果参数是标识符，从符号表获取类型
	if ident, ok := arg.(*entities.Identifier); ok {
		symbol, err := ee.symbolManager.LookupSymbol(ident.Name)
		if err == nil {
			argType = symbol.Type
		}
	} else if deref, ok := arg.(*entities.DereferenceExpr); ok {
		// 如果参数是解引用表达式（*m），获取被解引用的变量类型
		if ident, ok := deref.Operand.(*entities.Identifier); ok {
			symbol, err := ee.symbolManager.LookupSymbol(ident.Name)
			if err == nil {
				// 移除指针前缀（如果有）
				argType = strings.TrimPrefix(symbol.Type, "*")
			}
		}
	} else if structAccess, ok := arg.(*entities.StructAccess); ok {
		// 如果参数是结构体访问（m.buckets），获取结构体类型
		if ident, ok := structAccess.Object.(*entities.Identifier); ok {
			symbol, err := ee.symbolManager.LookupSymbol(ident.Name)
			if err == nil {
				argType = symbol.Type
			}
		}
	}
	
	// ✅ 修复：如果类型包含泛型类型参数，使用当前类型参数映射替换
	if argType != "" {
		currentTypeParamMap := irManager.GetCurrentFunctionTypeParams()
		// 添加调试输出
		if strings.Contains(argType, "Bucket") || strings.Contains(argType, "K") || strings.Contains(argType, "V") {
			fmt.Printf("DEBUG: getArgType - argType=%s, currentTypeParamMap=%v\n", argType, currentTypeParamMap)
		}
		if currentTypeParamMap != nil && len(currentTypeParamMap) > 0 {
			// 检查是否包含泛型类型参数（简单检查：是否包含单个大写字母）
			hasGenericParams := false
			for typeParam := range currentTypeParamMap {
				// 检查类型字符串中是否包含这个类型参数
				if strings.Contains(argType, typeParam) {
					hasGenericParams = true
					break
				}
			}
			
			if hasGenericParams {
				fmt.Printf("DEBUG: getArgType - hasGenericParams=true, argType=%s, typeParamMap=%v\n", argType, currentTypeParamMap)
				substitutedType := ee.substituteTypeParamsInString(argType, currentTypeParamMap)
				if substitutedType != argType {
					fmt.Printf("DEBUG: getArgType - substituted type: %s -> %s\n", argType, substitutedType)
					argType = substitutedType
				} else {
					fmt.Printf("DEBUG: getArgType - substitution did not change type: %s\n", argType)
				}
			} else {
				if strings.Contains(argType, "Bucket") || strings.Contains(argType, "K") || strings.Contains(argType, "V") {
					fmt.Printf("DEBUG: getArgType - hasGenericParams=false, argType=%s, typeParamMap=%v\n", argType, currentTypeParamMap)
				}
			}
		} else {
			if strings.Contains(argType, "Bucket") || strings.Contains(argType, "K") || strings.Contains(argType, "V") {
				fmt.Printf("DEBUG: getArgType - currentTypeParamMap is empty or nil, argType=%s\n", argType)
			}
		}
	}
	
	return argType
}

// extractTypeParamsFromArgType 从参数类型提取类型参数
// 例如：HashMap[int, string] -> ["int", "string"]
// Vec[int] -> ["int"]
// int -> [] (非泛型类型)
func (ee *ExpressionEvaluatorImpl) extractTypeParamsFromArgType(argType string) []string {
	// 移除指针前缀（如果有）
	argType = strings.TrimPrefix(argType, "*")
	
	// 检查是否包含泛型类型参数（包含[和]）
	if !strings.Contains(argType, "[") || !strings.Contains(argType, "]") {
		return nil
	}
	
	// 提取类型参数部分
	openBracket := strings.Index(argType, "[")
	closeBracket := strings.LastIndex(argType, "]")
	if openBracket == -1 || closeBracket == -1 || closeBracket <= openBracket {
		return nil
	}
	
	// 提取类型参数部分（如int, string）
	typeParamsStr := argType[openBracket+1 : closeBracket]
	
	// 分割类型参数（处理嵌套泛型，如 Vec[HashMap[int, string]]）
	// 简单实现：按逗号分割，但需要处理嵌套的方括号
	typeParams := []string{}
	currentParam := ""
	bracketDepth := 0
	
	for _, char := range typeParamsStr {
		if char == '[' {
			bracketDepth++
			currentParam += string(char)
		} else if char == ']' {
			bracketDepth--
			currentParam += string(char)
		} else if char == ',' && bracketDepth == 0 {
			// 顶层逗号，分割类型参数
			param := strings.TrimSpace(currentParam)
			if param != "" {
				typeParams = append(typeParams, param)
			}
			currentParam = ""
		} else {
			currentParam += string(char)
		}
	}
	
	// 添加最后一个类型参数
	param := strings.TrimSpace(currentParam)
	if param != "" {
		typeParams = append(typeParams, param)
	}
	
	return typeParams
}

// substituteTypeParamsInString 在类型字符串中替换类型参数为具体类型
// 例如：Bucket[K, V] + {"K": "User", "V": "string"} -> Bucket[User, string]
func (ee *ExpressionEvaluatorImpl) substituteTypeParamsInString(typeStr string, typeParamMap map[string]string) string {
	if typeStr == "" || typeParamMap == nil || len(typeParamMap) == 0 {
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
	
	// 替换类型参数（只替换完整的单词，避免部分匹配）
	// ✅ 改进：使用更精确的替换逻辑，处理泛型类型参数在方括号内的情况
	for _, typeParam := range typeParams {
		concreteType := typeParamMap[typeParam]
		
		// 使用正则表达式或精确的字符串匹配，确保只替换完整的类型参数名
		// 匹配模式：
		// 1. [K] - 单个类型参数
		// 2. [K, - 第一个类型参数
		// 3. ,K] - 最后一个类型参数
		// 4. ,K, - 中间的类型参数
		// 5. [K V] - 空格分隔的类型参数（较少见）
		// 6. 空格+K+空格 - 独立出现的类型参数
		
		// 先处理方括号内的类型参数（最常见的情况）
		patterns := []string{
			"[" + typeParam + "]",      // [K] -> [User]
			"[" + typeParam + ",",      // [K, -> [User,
			"," + typeParam + "]",      // ,K] -> ,User]
			"," + typeParam + ",",      // ,K, -> ,User,
			"[" + typeParam + " ",      // [K  -> [User 
			" " + typeParam + "]",      //  K] ->  User]
			" " + typeParam + ",",      //  K, ->  User,
			"," + typeParam + " ",      // ,K  -> ,User 
			" " + typeParam + " ",      //  K  ->  User 
		}
		replacements := []string{
			"[" + concreteType + "]",
			"[" + concreteType + ",",
			"," + concreteType + "]",
			"," + concreteType + ",",
			"[" + concreteType + " ",
			" " + concreteType + "]",
			" " + concreteType + ",",
			"," + concreteType + " ",
			" " + concreteType + " ",
		}
		
		// 按顺序替换（从最具体的模式开始）
		for i, pattern := range patterns {
			if strings.Contains(result, pattern) {
				result = strings.ReplaceAll(result, pattern, replacements[i])
				fmt.Printf("DEBUG: substituteTypeParamsInString - replaced %s with %s in %s\n", pattern, replacements[i], typeStr)
			}
		}
	}
	
	return result
}

// generateMonomorphizedName 生成单态化函数名
// 例如：bucket_index + {"K": "int", "V": "string"} + ["K", "V"] -> bucket_index_int_string
func (ee *ExpressionEvaluatorImpl) generateMonomorphizedName(originalName string, typeParamMap map[string]string, typeParamNames []string) string {
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

// ensureStructHashFunction 确保结构体的hash函数已生成，如果未生成则生成
// 返回hash函数的函数指针（*ir.Func）
func (ee *ExpressionEvaluatorImpl) ensureStructHashFunction(structName string) (*ir.Func, error) {
	// 检查是否已生成
	if hashFunc, exists := ee.generatedHashFunctions[structName]; exists {
		return hashFunc, nil
	}
	
	// 获取结构体定义
	structDef, exists := ee.structDefinitions[structName]
	if !exists {
		return nil, fmt.Errorf("struct definition not found: %s", structName)
	}
	
	// 生成hash函数
	hashFunc, err := ee.generateStructHashFunction(structName, structDef)
	if err != nil {
		return nil, fmt.Errorf("failed to generate hash function for %s: %w", structName, err)
	}
	
	// 缓存生成的函数
	ee.generatedHashFunctions[structName] = hashFunc
	
	return hashFunc, nil
}

// ensureStructEqualsFunction 确保结构体的equals函数已生成，如果未生成则生成
// 返回equals函数的函数指针（*ir.Func）
func (ee *ExpressionEvaluatorImpl) ensureStructEqualsFunction(structName string) (*ir.Func, error) {
	// 检查是否已生成
	if equalsFunc, exists := ee.generatedEqualsFunctions[structName]; exists {
		return equalsFunc, nil
	}
	
	// 获取结构体定义
	structDef, exists := ee.structDefinitions[structName]
	if !exists {
		return nil, fmt.Errorf("struct definition not found: %s", structName)
	}
	
	// 生成equals函数
	equalsFunc, err := ee.generateStructEqualsFunction(structName, structDef)
	if err != nil {
		return nil, fmt.Errorf("failed to generate equals function for %s: %w", structName, err)
	}
	
	// 缓存生成的函数
	ee.generatedEqualsFunctions[structName] = equalsFunc
	
	return equalsFunc, nil
}

// generateStructHashFunction 为结构体生成hash函数
// 函数签名：i64 runtime_{TypeName}_hash(i8* key_ptr)
func (ee *ExpressionEvaluatorImpl) generateStructHashFunction(structName string, structDef *entities.StructDef) (*ir.Func, error) {
	irManager := ee.irModuleManager
	
	// 函数名：runtime_{TypeName}_hash
	funcName := fmt.Sprintf("runtime_%s_hash", structName)
	
	// 检查函数是否已存在
	if existingFunc, exists := ee.generatedHashFunctions[structName]; exists {
		return existingFunc, nil
	}
	
	// 检查模块中是否已存在
	if irManagerImpl, ok := irManager.(*IRModuleManagerImpl); ok {
		if existingFunc := irManagerImpl.findFunction(funcName); existingFunc != nil {
			ee.generatedHashFunctions[structName] = existingFunc
			return existingFunc, nil
		}
	}
	
	// 1. 创建函数：i64 runtime_{TypeName}_hash(i8* key_ptr)
	hashFunc, err := irManager.CreateFunction(
		funcName,
		types.I64, // 返回类型：i64
		[]interface{}{types.NewPointer(types.I8)}, // 参数：i8* key_ptr
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create hash function: %w", err)
	}
	
	hashFuncLLVM, ok := hashFunc.(*ir.Func)
	if !ok {
		return nil, fmt.Errorf("CreateFunction returned invalid type: %T", hashFunc)
	}
	
	// 2. 设置当前函数和基本块
	oldFunc := irManager.GetCurrentFunction()
	oldBlock := irManager.GetCurrentBasicBlock()
	
	err = irManager.SetCurrentFunction(hashFuncLLVM)
	if err != nil {
		return nil, fmt.Errorf("failed to set current function: %w", err)
	}
	
	entryBlock, err := irManager.CreateBasicBlock("entry")
	if err != nil {
		return nil, fmt.Errorf("failed to create entry block: %w", err)
	}
	
	err = irManager.SetCurrentBasicBlock(entryBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to set entry block: %w", err)
	}
	
	// 3. 获取参数：i8* key_ptr
	keyParam := hashFuncLLVM.Params[0]
	
	// 4. 构建结构体LLVM类型
	structLLVMType, err := ee.buildStructLLVMType(structDef)
	if err != nil {
		return nil, fmt.Errorf("failed to build struct LLVM type: %w", err)
	}
	
	// 5. 将 void* (i8*) 转换为结构体指针
	structPtrType := types.NewPointer(structLLVMType)
	structPtr, err := irManager.CreateBitCast(keyParam, structPtrType, fmt.Sprintf("%s_ptr", structName))
	if err != nil {
		return nil, fmt.Errorf("failed to cast key_ptr to struct pointer: %w", err)
	}
	
	structPtrValue, ok := structPtr.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("CreateBitCast returned invalid type: %T", structPtr)
	}
	
	// 6. 初始化组合哈希值（从0开始）
	combinedHashInit := constant.NewInt(types.I64, 0)
	var combinedHashValue llvalue.Value = combinedHashInit
	
	// 7. 遍历所有字段，为每个字段生成hash调用并组合
	for i, field := range structDef.Fields {
		// 7.1 获取字段hash函数名
		fieldHashFuncName, err := ee.getFieldHashFunctionName(field.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to get hash function for field %s.%s (type %s): %w", structName, field.Name, field.Type, err)
		}
		
		// 7.2 如果字段hash函数名为空，跳过（不支持的类型）
		if fieldHashFuncName == "" {
			continue
		}
		
		// 7.3 获取字段hash函数
		fieldHashFunc, exists := irManager.GetExternalFunction(fieldHashFuncName)
		if !exists {
			// 如果是结构体字段，递归生成hash函数
			if _, structExists := ee.structDefinitions[extractBaseTypeName(field.Type)]; structExists {
				_, err := ee.ensureStructHashFunction(extractBaseTypeName(field.Type))
				if err != nil {
					return nil, fmt.Errorf("failed to generate hash function for nested struct field %s: %w", field.Type, err)
				}
				// 重新获取函数
				fieldHashFunc, exists = irManager.GetExternalFunction(fieldHashFuncName)
				if !exists {
					return nil, fmt.Errorf("hash function %s not found after generation", fieldHashFuncName)
				}
			} else {
				return nil, fmt.Errorf("hash function %s not found for field %s.%s", fieldHashFuncName, structName, field.Name)
			}
		}
		
		// 7.4 使用GEP获取字段指针
		fieldIndexConst := constant.NewInt(types.I32, int64(i))
		fieldPtr, err := irManager.CreateGetElementPtr(structLLVMType, structPtrValue, constant.NewInt(types.I32, 0), fieldIndexConst)
		if err != nil {
			return nil, fmt.Errorf("failed to create GEP for field %s.%s: %w", structName, field.Name, err)
		}
		
		fieldPtrValue, ok := fieldPtr.(llvalue.Value)
		if !ok {
			return nil, fmt.Errorf("CreateGetElementPtr returned invalid type: %T", fieldPtr)
		}
		
		// 7.5 加载字段值
		fieldLLVMTypeInterface, err := ee.typeMapper.MapType(field.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to map field type %s: %w", field.Type, err)
		}
		fieldLLVMType, ok := fieldLLVMTypeInterface.(types.Type)
		if !ok {
			return nil, fmt.Errorf("field type %s is not a valid LLVM type", field.Type)
		}
		
		fieldValue, err := irManager.CreateLoad(fieldLLVMType, fieldPtrValue, fmt.Sprintf("%s_%s", structName, field.Name))
		if err != nil {
			return nil, fmt.Errorf("failed to load field %s.%s: %w", structName, field.Name, err)
		}
		
		fieldValueLLVM, ok := fieldValue.(llvalue.Value)
		if !ok {
			return nil, fmt.Errorf("CreateLoad returned invalid type: %T", fieldValue)
		}
		
		// 7.6 调用字段hash函数
		// 注意：对于string类型，需要从string_t结构体中提取data字段（char*），然后传递给runtime_string_hash
		// 对于bool类型，需要转换为i32
		var hashArg llvalue.Value
		if field.Type == "string" {
			// ✅ 修复：string类型：需要从string_t结构体中提取data字段
			// fieldValueLLVM是i8*（指向string_t结构体的指针）
			// runtime_string_hash期望const char*（C字符串），即string_t.data字段
			// string_t结构体：{ i8* data, i64 length, i64 capacity }
			
			// 1. 将i8*转换为string_t*（结构体指针）
			stringStructType := types.NewStruct(
				types.NewPointer(types.I8), // data: char*
				types.I64,                  // length: size_t
				types.I64,                  // capacity: size_t
			)
			stringStructPtrType := types.NewPointer(stringStructType)
			stringStructPtr, err := irManager.CreateBitCast(fieldValueLLVM, stringStructPtrType, fmt.Sprintf("%s_string_t_ptr", structName))
			if err != nil {
				return nil, fmt.Errorf("failed to bitcast string pointer to string_t*: %w", err)
			}
			
			stringStructPtrValue, ok := stringStructPtr.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("CreateBitCast returned invalid type: %T", stringStructPtr)
			}
			
			// 2. 使用GEP获取data字段（索引0）
			dataFieldPtr, err := irManager.CreateGetElementPtr(stringStructType, stringStructPtrValue, constant.NewInt(types.I32, 0), constant.NewInt(types.I32, 0))
			if err != nil {
				return nil, fmt.Errorf("failed to create GEP for string_t.data field: %w", err)
			}
			
			dataFieldPtrValue, ok := dataFieldPtr.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("CreateGetElementPtr returned invalid type: %T", dataFieldPtr)
			}
			
			// 3. 加载data字段的值（char*）
			dataValue, err := irManager.CreateLoad(types.NewPointer(types.I8), dataFieldPtrValue, fmt.Sprintf("%s_string_data", structName))
			if err != nil {
				return nil, fmt.Errorf("failed to load string_t.data field: %w", err)
			}
			
			dataValueLLVM, ok := dataValue.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("CreateLoad returned invalid type: %T", dataValue)
			}
			
			// 4. 使用data字段（char*）作为hash函数的参数
			hashArg = dataValueLLVM
		} else if field.Type == "bool" {
			// bool类型：转换为i32（因为runtime_int32_hash接受i32）
			// 使用zext将i1转换为i32
			currentBlock := irManager.GetCurrentBasicBlock()
			if currentBlock == nil {
				return nil, fmt.Errorf("no current basic block set for bool conversion")
			}
			block, ok := currentBlock.(*ir.Block)
			if !ok {
				return nil, fmt.Errorf("invalid block type for bool conversion: %T", currentBlock)
			}
			boolToInt32 := block.NewZExt(fieldValueLLVM, types.I32)
			hashArg = boolToInt32
		} else {
			// 其他类型：传递值
			hashArg = fieldValueLLVM
		}
		
		fieldHashResult, err := irManager.CreateCall(fieldHashFunc, hashArg)
		if err != nil {
			return nil, fmt.Errorf("failed to call hash function %s for field %s.%s: %w", fieldHashFuncName, structName, field.Name, err)
		}
		
		fieldHashResultValue, ok := fieldHashResult.(llvalue.Value)
		if !ok {
			return nil, fmt.Errorf("CreateCall returned invalid type: %T", fieldHashResult)
		}
		
		// 7.7 组合哈希值：combinedHash = combinedHash ^ (fieldHash << i)
		// 使用左移和XOR组合
		shiftAmount := constant.NewInt(types.I32, int64(i))
		shiftedHash, err := irManager.CreateBinaryOp("shl", fieldHashResultValue, shiftAmount, fmt.Sprintf("field_%s_hash_shifted", field.Name))
		if err != nil {
			return nil, fmt.Errorf("failed to shift field hash: %w", err)
		}
		
		shiftedHashValue, ok := shiftedHash.(llvalue.Value)
		if !ok {
			return nil, fmt.Errorf("CreateBinaryOp returned invalid type: %T", shiftedHash)
		}
		
		combinedHashResult, err := irManager.CreateBinaryOp("xor", combinedHashValue, shiftedHashValue, fmt.Sprintf("combined_hash_%d", i))
		if err != nil {
			return nil, fmt.Errorf("failed to combine hash: %w", err)
		}
		
		combinedHashValue, ok = combinedHashResult.(llvalue.Value)
		if !ok {
			return nil, fmt.Errorf("CreateBinaryOp returned invalid type: %T", combinedHashResult)
		}
	}
	
	// 8. 返回组合哈希值
	entryBlockValue, ok := entryBlock.(*ir.Block)
	if !ok {
		return nil, fmt.Errorf("CreateBasicBlock returned invalid type: %T", entryBlock)
	}
	entryBlockValue.NewRet(combinedHashValue)
	
	// 9. 恢复之前的函数和基本块
	if oldFunc != nil {
		irManager.SetCurrentFunction(oldFunc)
	}
	if oldBlock != nil {
		irManager.SetCurrentBasicBlock(oldBlock)
	}
	
	return hashFuncLLVM, nil
}

// generateStructEqualsFunction 为结构体生成equals函数
// 函数签名：i1 runtime_{TypeName}_equals(i8* key1_ptr, i8* key2_ptr)
func (ee *ExpressionEvaluatorImpl) generateStructEqualsFunction(structName string, structDef *entities.StructDef) (*ir.Func, error) {
	irManager := ee.irModuleManager
	
	// 函数名：runtime_{TypeName}_equals
	funcName := fmt.Sprintf("runtime_%s_equals", structName)
	
	// 检查函数是否已存在
	if equalsFunc, exists := ee.generatedEqualsFunctions[structName]; exists {
		return equalsFunc, nil
	}
	
	// 检查模块中是否已存在
	if irManagerImpl, ok := irManager.(*IRModuleManagerImpl); ok {
		if existingFunc := irManagerImpl.findFunction(funcName); existingFunc != nil {
			ee.generatedEqualsFunctions[structName] = existingFunc
			return existingFunc, nil
		}
	}
	
	// 1. 创建函数：i1 runtime_{TypeName}_equals(i8* key1_ptr, i8* key2_ptr)
	equalsFunc, err := irManager.CreateFunction(
		funcName,
		types.I1, // 返回类型：i1 (bool)
		[]interface{}{
			types.NewPointer(types.I8), // 参数1：i8* key1_ptr
			types.NewPointer(types.I8), // 参数2：i8* key2_ptr
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create equals function: %w", err)
	}
	
	equalsFuncLLVM, ok := equalsFunc.(*ir.Func)
	if !ok {
		return nil, fmt.Errorf("CreateFunction returned invalid type: %T", equalsFunc)
	}
	
	// 2. 设置当前函数和基本块
	oldFunc := irManager.GetCurrentFunction()
	oldBlock := irManager.GetCurrentBasicBlock()
	
	err = irManager.SetCurrentFunction(equalsFuncLLVM)
	if err != nil {
		return nil, fmt.Errorf("failed to set current function: %w", err)
	}
	
	entryBlock, err := irManager.CreateBasicBlock("entry")
	if err != nil {
		return nil, fmt.Errorf("failed to create entry block: %w", err)
	}
	
	err = irManager.SetCurrentBasicBlock(entryBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to set entry block: %w", err)
	}
	
	// 3. 获取参数：i8* key1_ptr, i8* key2_ptr
	key1Param := equalsFuncLLVM.Params[0]
	key2Param := equalsFuncLLVM.Params[1]
	
	// 4. 构建结构体LLVM类型
	structLLVMType, err := ee.buildStructLLVMType(structDef)
	if err != nil {
		return nil, fmt.Errorf("failed to build struct LLVM type: %w", err)
	}
	
	// 5. 将 void* (i8*) 转换为结构体指针
	structPtrType := types.NewPointer(structLLVMType)
	struct1Ptr, err := irManager.CreateBitCast(key1Param, structPtrType, fmt.Sprintf("%s1_ptr", structName))
	if err != nil {
		return nil, fmt.Errorf("failed to cast key1_ptr to struct pointer: %w", err)
	}
	
	struct1PtrValue, ok := struct1Ptr.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("CreateBitCast returned invalid type: %T", struct1Ptr)
	}
	
	struct2Ptr, err := irManager.CreateBitCast(key2Param, structPtrType, fmt.Sprintf("%s2_ptr", structName))
	if err != nil {
		return nil, fmt.Errorf("failed to cast key2_ptr to struct pointer: %w", err)
	}
	
	struct2PtrValue, ok := struct2Ptr.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("CreateBitCast returned invalid type: %T", struct2Ptr)
	}
	
	// 6. 初始化组合结果（从true开始，使用AND组合）
	combinedResultInit := constant.NewInt(types.I1, 1) // true
	var combinedResultValue llvalue.Value = combinedResultInit
	
	// 7. 遍历所有字段，为每个字段生成equals调用并组合
	for i, field := range structDef.Fields {
		// 7.1 使用GEP获取字段指针
		fieldIndexConst := constant.NewInt(types.I32, int64(i))
		field1Ptr, err := irManager.CreateGetElementPtr(structLLVMType, struct1PtrValue, constant.NewInt(types.I32, 0), fieldIndexConst)
		if err != nil {
			return nil, fmt.Errorf("failed to create GEP for field1 %s.%s: %w", structName, field.Name, err)
		}
		
		field1PtrValue, ok := field1Ptr.(llvalue.Value)
		if !ok {
			return nil, fmt.Errorf("CreateGetElementPtr returned invalid type: %T", field1Ptr)
		}
		
		field2Ptr, err := irManager.CreateGetElementPtr(structLLVMType, struct2PtrValue, constant.NewInt(types.I32, 0), fieldIndexConst)
		if err != nil {
			return nil, fmt.Errorf("failed to create GEP for field2 %s.%s: %w", structName, field.Name, err)
		}
		
		field2PtrValue, ok := field2Ptr.(llvalue.Value)
		if !ok {
			return nil, fmt.Errorf("CreateGetElementPtr returned invalid type: %T", field2Ptr)
		}
		
		// 7.2 获取字段equals函数名
		fieldEqualsFuncName, err := ee.getFieldEqualsFunctionName(field.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to get equals function for field %s.%s (type %s): %w", structName, field.Name, field.Type, err)
		}
		
		// 7.3 比较字段值
		var fieldEqualsResult llvalue.Value
		
		if fieldEqualsFuncName == "" {
			// 基本类型：直接比较（使用 == 操作符）
			fieldLLVMTypeInterface, err := ee.typeMapper.MapType(field.Type)
			if err != nil {
				return nil, fmt.Errorf("failed to map field type %s: %w", field.Type, err)
			}
			fieldLLVMType, ok := fieldLLVMTypeInterface.(types.Type)
			if !ok {
				return nil, fmt.Errorf("field type %s is not a valid LLVM type", field.Type)
			}
			
			// 加载字段值
			field1Value, err := irManager.CreateLoad(fieldLLVMType, field1PtrValue, fmt.Sprintf("%s1_%s", structName, field.Name))
			if err != nil {
				return nil, fmt.Errorf("failed to load field1 %s.%s: %w", structName, field.Name, err)
			}
			
			field1ValueLLVM, ok := field1Value.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("CreateLoad returned invalid type: %T", field1Value)
			}
			
			field2Value, err := irManager.CreateLoad(fieldLLVMType, field2PtrValue, fmt.Sprintf("%s2_%s", structName, field.Name))
			if err != nil {
				return nil, fmt.Errorf("failed to load field2 %s.%s: %w", structName, field.Name, err)
			}
			
			field2ValueLLVM, ok := field2Value.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("CreateLoad returned invalid type: %T", field2Value)
			}
			
			// 使用 == 操作符比较
			fieldEqualsResultRaw, err := irManager.CreateBinaryOp("==", field1ValueLLVM, field2ValueLLVM, fmt.Sprintf("field_%s_eq", field.Name))
			if err != nil {
				return nil, fmt.Errorf("failed to compare field %s.%s: %w", structName, field.Name, err)
			}
			
			fieldEqualsResult, ok = fieldEqualsResultRaw.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("CreateBinaryOp returned invalid type: %T", fieldEqualsResultRaw)
			}
		} else {
			// 复杂类型：调用equals函数
			fieldEqualsFunc, exists := irManager.GetExternalFunction(fieldEqualsFuncName)
			if !exists {
			// 如果是结构体字段，递归生成equals函数
			if _, structExists := ee.structDefinitions[extractBaseTypeName(field.Type)]; structExists {
				_, err2 := ee.ensureStructEqualsFunction(extractBaseTypeName(field.Type))
				if err2 != nil {
					return nil, fmt.Errorf("failed to generate equals function for nested struct field %s: %w", field.Type, err2)
				}
				// 重新获取函数
				fieldEqualsFunc, exists = irManager.GetExternalFunction(fieldEqualsFuncName)
				if !exists {
					return nil, fmt.Errorf("equals function %s not found after generation", fieldEqualsFuncName)
				}
			} else {
				return nil, fmt.Errorf("equals function %s not found for field %s.%s", fieldEqualsFuncName, structName, field.Name)
			}
			}
			
			// 调用字段equals函数
			// 注意：对于string类型，需要从string_t结构体中提取data字段（char*），然后传递给runtime_string_equals
			var equalsArg1, equalsArg2 llvalue.Value
			if field.Type == "string" {
				// ✅ 修复：string类型：需要从string_t结构体中提取data字段
				// field1PtrValue和field2PtrValue是i8**（指向string_t*的指针）
				// runtime_string_equals期望const char*（C字符串），即string_t.data字段
				// string_t结构体：{ i8* data, i64 length, i64 capacity }
				
				// 1. 加载string_t*指针
				fieldLLVMTypeInterface, err := ee.typeMapper.MapType(field.Type)
				if err != nil {
					return nil, fmt.Errorf("failed to map field type %s: %w", field.Type, err)
				}
				fieldLLVMType, ok := fieldLLVMTypeInterface.(types.Type)
				if !ok {
					return nil, fmt.Errorf("field type %s is not a valid LLVM type", field.Type)
				}
				
				field1Value, err := irManager.CreateLoad(fieldLLVMType, field1PtrValue, fmt.Sprintf("%s1_%s", structName, field.Name))
				if err != nil {
					return nil, fmt.Errorf("failed to load field1 %s.%s: %w", structName, field.Name, err)
				}
				field1ValueLLVM, ok := field1Value.(llvalue.Value)
				if !ok {
					return nil, fmt.Errorf("CreateLoad returned invalid type: %T", field1Value)
				}
				
				field2Value, err := irManager.CreateLoad(fieldLLVMType, field2PtrValue, fmt.Sprintf("%s2_%s", structName, field.Name))
				if err != nil {
					return nil, fmt.Errorf("failed to load field2 %s.%s: %w", structName, field.Name, err)
				}
				field2ValueLLVM, ok := field2Value.(llvalue.Value)
				if !ok {
					return nil, fmt.Errorf("CreateLoad returned invalid type: %T", field2Value)
				}
				
				// 2. 将i8*转换为string_t*（结构体指针）
				stringStructType := types.NewStruct(
					types.NewPointer(types.I8), // data: char*
					types.I64,                  // length: size_t
					types.I64,                  // capacity: size_t
				)
				stringStructPtrType := types.NewPointer(stringStructType)
				
				string1StructPtr, err := irManager.CreateBitCast(field1ValueLLVM, stringStructPtrType, fmt.Sprintf("%s1_string_t_ptr", structName))
				if err != nil {
					return nil, fmt.Errorf("failed to bitcast string1 pointer to string_t*: %w", err)
				}
				string1StructPtrValue, ok := string1StructPtr.(llvalue.Value)
				if !ok {
					return nil, fmt.Errorf("CreateBitCast returned invalid type: %T", string1StructPtr)
				}
				
				string2StructPtr, err := irManager.CreateBitCast(field2ValueLLVM, stringStructPtrType, fmt.Sprintf("%s2_string_t_ptr", structName))
				if err != nil {
					return nil, fmt.Errorf("failed to bitcast string2 pointer to string_t*: %w", err)
				}
				string2StructPtrValue, ok := string2StructPtr.(llvalue.Value)
				if !ok {
					return nil, fmt.Errorf("CreateBitCast returned invalid type: %T", string2StructPtr)
				}
				
				// 3. 使用GEP获取data字段（索引0）
				data1FieldPtr, err := irManager.CreateGetElementPtr(stringStructType, string1StructPtrValue, constant.NewInt(types.I32, 0), constant.NewInt(types.I32, 0))
				if err != nil {
					return nil, fmt.Errorf("failed to create GEP for string1_t.data field: %w", err)
				}
				data1FieldPtrValue, ok := data1FieldPtr.(llvalue.Value)
				if !ok {
					return nil, fmt.Errorf("CreateGetElementPtr returned invalid type: %T", data1FieldPtr)
				}
				
				data2FieldPtr, err := irManager.CreateGetElementPtr(stringStructType, string2StructPtrValue, constant.NewInt(types.I32, 0), constant.NewInt(types.I32, 0))
				if err != nil {
					return nil, fmt.Errorf("failed to create GEP for string2_t.data field: %w", err)
				}
				data2FieldPtrValue, ok := data2FieldPtr.(llvalue.Value)
				if !ok {
					return nil, fmt.Errorf("CreateGetElementPtr returned invalid type: %T", data2FieldPtr)
				}
				
				// 4. 加载data字段的值（char*）
				data1Value, err := irManager.CreateLoad(types.NewPointer(types.I8), data1FieldPtrValue, fmt.Sprintf("%s1_string_data", structName))
				if err != nil {
					return nil, fmt.Errorf("failed to load string1_t.data field: %w", err)
				}
				data1ValueLLVM, ok := data1Value.(llvalue.Value)
				if !ok {
					return nil, fmt.Errorf("CreateLoad returned invalid type: %T", data1Value)
				}
				
				data2Value, err := irManager.CreateLoad(types.NewPointer(types.I8), data2FieldPtrValue, fmt.Sprintf("%s2_string_data", structName))
				if err != nil {
					return nil, fmt.Errorf("failed to load string2_t.data field: %w", err)
				}
				data2ValueLLVM, ok := data2Value.(llvalue.Value)
				if !ok {
					return nil, fmt.Errorf("CreateLoad returned invalid type: %T", data2Value)
				}
				
				// 5. 使用data字段（char*）作为equals函数的参数
				equalsArg1 = data1ValueLLVM
				equalsArg2 = data2ValueLLVM
			} else {
				// 其他类型：需要加载值
				fieldLLVMTypeInterface, err := ee.typeMapper.MapType(field.Type)
				if err != nil {
					return nil, fmt.Errorf("failed to map field type %s: %w", field.Type, err)
				}
				fieldLLVMType, ok := fieldLLVMTypeInterface.(types.Type)
				if !ok {
					return nil, fmt.Errorf("field type %s is not a valid LLVM type", field.Type)
				}
				
				field1Value, err := irManager.CreateLoad(fieldLLVMType, field1PtrValue, fmt.Sprintf("%s1_%s", structName, field.Name))
				if err != nil {
					return nil, fmt.Errorf("failed to load field1 %s.%s: %w", structName, field.Name, err)
				}
				
				field1ValueLLVM, ok := field1Value.(llvalue.Value)
				if !ok {
					return nil, fmt.Errorf("CreateLoad returned invalid type: %T", field1Value)
				}
				
				field2Value, err := irManager.CreateLoad(fieldLLVMType, field2PtrValue, fmt.Sprintf("%s2_%s", structName, field.Name))
				if err != nil {
					return nil, fmt.Errorf("failed to load field2 %s.%s: %w", structName, field.Name, err)
				}
				
				field2ValueLLVM, ok := field2Value.(llvalue.Value)
				if !ok {
					return nil, fmt.Errorf("CreateLoad returned invalid type: %T", field2Value)
				}
				
				equalsArg1 = field1ValueLLVM
				equalsArg2 = field2ValueLLVM
			}
			
			fieldEqualsResultRaw, err := irManager.CreateCall(fieldEqualsFunc, equalsArg1, equalsArg2)
			if err != nil {
				return nil, fmt.Errorf("failed to call equals function %s for field %s.%s: %w", fieldEqualsFuncName, structName, field.Name, err)
			}
			
			fieldEqualsResultValue, ok := fieldEqualsResultRaw.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("CreateCall returned invalid type: %T", fieldEqualsResultRaw)
			}
			fieldEqualsResult = fieldEqualsResultValue
		}
		
		// 7.4 组合结果：combinedResult = combinedResult && fieldEqualsResult
		// 确保 fieldEqualsResult 是 i1 类型（布尔值）
		fieldEqualsResultI1, ok := fieldEqualsResult.(llvalue.Value)
		if !ok {
			return nil, fmt.Errorf("fieldEqualsResult is not a valid LLVM value: %T", fieldEqualsResult)
		}
		// 如果类型不匹配，进行类型转换
		if fieldEqualsResultI1.Type() != types.I1 {
			// 将结果转换为 i1（如果是 i32，使用 trunc；如果是其他类型，使用 icmp）
			currentBlock := irManager.GetCurrentBasicBlock()
			if currentBlock == nil {
				return nil, fmt.Errorf("no current basic block set for type conversion")
			}
			block, ok := currentBlock.(*ir.Block)
			if !ok {
				return nil, fmt.Errorf("invalid block type for type conversion: %T", currentBlock)
			}
			// 使用 icmp ne 0 将非零值转换为 i1
			// 注意：fieldEqualsResult 应该是 i1 类型（equals 函数返回 i1）
			// 但如果类型不匹配，可能是调用返回了其他类型
			// 这里假设 equals 函数应该返回 i1，如果类型不匹配，可能是实现问题
			// 暂时直接使用，如果出错再处理
			fieldType := fieldEqualsResultI1.Type()
			if intType, ok := fieldType.(*types.IntType); ok {
				zero := constant.NewInt(intType, 0)
				fieldEqualsResultI1 = block.NewICmp(enum.IPredNE, fieldEqualsResultI1, zero)
			} else {
				// 如果不是整数类型，可能是 equals 函数返回了错误的类型
				// 这种情况下，我们假设结果已经是布尔值，直接使用
				// 如果类型不匹配，会在 CreateBinaryOp 中报错
			}
		}
		combinedResultRaw, err := irManager.CreateBinaryOp("and", combinedResultValue, fieldEqualsResultI1, fmt.Sprintf("combined_equals_%d", i))
		if err != nil {
			return nil, fmt.Errorf("failed to combine equals result: %w", err)
		}
		
		combinedResultValue, ok = combinedResultRaw.(llvalue.Value)
		if !ok {
			return nil, fmt.Errorf("CreateBinaryOp returned invalid type: %T", combinedResultRaw)
		}
	}
	
	// 8. 返回组合结果
	entryBlockValue, ok := entryBlock.(*ir.Block)
	if !ok {
		return nil, fmt.Errorf("CreateBasicBlock returned invalid type: %T", entryBlock)
	}
	entryBlockValue.NewRet(combinedResultValue)
	
	// 9. 恢复之前的函数和基本块
	if oldFunc != nil {
		irManager.SetCurrentFunction(oldFunc)
	}
	if oldBlock != nil {
		irManager.SetCurrentBasicBlock(oldBlock)
	}
	
	return equalsFuncLLVM, nil
}

// getFieldHashFunctionName 获取字段类型的hash函数名
// 返回函数名，如果字段类型不支持hash则返回错误
func (ee *ExpressionEvaluatorImpl) getFieldHashFunctionName(fieldType string) (string, error) {
	switch fieldType {
	case "int", "i32":
		return "runtime_int32_hash", nil
	case "i64":
		return "runtime_int64_hash", nil
	case "string":
		return "runtime_string_hash", nil
	case "float", "f32", "f64":
		return "runtime_float_hash", nil
	case "bool":
		// 注意：runtime_bool_hash 是静态内联函数，不在头文件中声明
		// 我们需要在运行时API中添加声明，或者使用int32_hash代替
		// 暂时使用int32_hash（bool在C中是整数）
		return "runtime_int32_hash", nil
	default:
		// 检查是否是结构体类型
		baseTypeName := extractBaseTypeName(fieldType)
		if _, exists := ee.structDefinitions[baseTypeName]; exists {
			// 结构体字段：递归调用 runtime_{FieldType}_hash
			return fmt.Sprintf("runtime_%s_hash", baseTypeName), nil
		}
		// 不支持的类型
		return "", fmt.Errorf("unsupported field type for hash: %s", fieldType)
	}
}

// getFieldEqualsFunctionName 获取字段类型的equals函数名
// 返回函数名，如果字段类型使用直接比较（==）则返回空字符串
func (ee *ExpressionEvaluatorImpl) getFieldEqualsFunctionName(fieldType string) (string, error) {
	switch fieldType {
	case "int", "i32", "i64", "bool":
		// 基本类型：直接比较（使用 == 操作符），不调用函数
		return "", nil
	case "string":
		return "runtime_string_equals", nil
	case "float", "f32", "f64":
		// 注意：runtime_float_equals 可能不存在，暂时使用直接比较（==）
		// 未来可以添加runtime_float_equals函数（考虑epsilon和NaN）
		// 暂时使用直接比较，不调用函数
		return "", nil
	default:
		// 检查是否是结构体类型
		baseTypeName := extractBaseTypeName(fieldType)
		if _, exists := ee.structDefinitions[baseTypeName]; exists {
			// 结构体字段：递归调用 runtime_{FieldType}_equals
			return fmt.Sprintf("runtime_%s_equals", baseTypeName), nil
		}
		// 不支持的类型
		return "", fmt.Errorf("unsupported field type for equals: %s", fieldType)
	}
}

// extractBaseTypeName 提取基础类型名（移除类型参数和指针前缀）
// 例如：*User -> User, Option[int] -> Option, User -> User
func extractBaseTypeName(typeName string) string {
	// 移除指针前缀
	if strings.HasPrefix(typeName, "*") {
		typeName = typeName[1:]
	}
	
	// 移除类型参数
	if bracketIndex := strings.Index(typeName, "["); bracketIndex != -1 {
		typeName = typeName[:bracketIndex]
	}
	
	return strings.TrimSpace(typeName)
}

// EvaluateNamespaceAccessExpr 求值命名空间访问表达式
// 例如：net::bind -> net_bind (函数调用)
func (ee *ExpressionEvaluatorImpl) EvaluateNamespaceAccessExpr(irManager generation.IRModuleManager, expr *entities.NamespaceAccessExpr) (interface{}, error) {
	// 将命名空间访问转换为函数调用：namespace::member -> namespace_member
	funcName := expr.Namespace + "_" + expr.Member
	
	// 创建函数调用表达式
	funcCall := &entities.FuncCall{
		Name: funcName,
		Args: []entities.Expr{}, // 命名空间访问本身不包含参数，参数在后续的函数调用中
	}
	
	// 求值函数调用
	return ee.EvaluateFuncCall(irManager, funcCall)
}
