package impl

import (
	"fmt"
	"strings"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
	llvalue "github.com/llir/llvm/ir/value"
)

// IRModuleManagerImpl LLVM IR模块管理器实现
type IRModuleManagerImpl struct {
	module            *ir.Module
	currentFunc       *ir.Func
	currentBlock      *ir.Block
	builder           *ir.Block           // 当前基本块的构建器
	stringCount       int                 // 字符串常量计数器
	externalFunctions map[string]*ir.Func // 外部函数缓存
	variableCounters  map[string]int      // 变量名计数器，用于生成唯一的LLVM IR变量名
}

// NewIRModuleManagerImpl 创建IR模块管理器实现
func NewIRModuleManagerImpl() *IRModuleManagerImpl {
	manager := &IRModuleManagerImpl{
		module:            ir.NewModule(),
		stringCount:       0,
		externalFunctions: make(map[string]*ir.Func),
		variableCounters:  make(map[string]int),
	}

	// 声明协程运行时函数
	manager.declareCoroutineRuntimeFunctions()

	return manager
}

// AddGlobalVariable 添加全局变量
func (m *IRModuleManagerImpl) AddGlobalVariable(name string, value interface{}) error {
	// 暂时不支持全局变量
	return fmt.Errorf("global variables not yet supported")
}

// CreateFunction 创建函数
func (m *IRModuleManagerImpl) CreateFunction(name string, returnType interface{}, paramTypes []interface{}) (interface{}, error) {
	var llvmReturnType types.Type

	// 处理不同的返回类型
	switch rt := returnType.(type) {
	case types.Type:
		llvmReturnType = rt
	case *types.IntType:
		llvmReturnType = rt
	case *types.FloatType:
		llvmReturnType = rt
	case *types.PointerType:
		llvmReturnType = rt
	case *types.ArrayType:
		llvmReturnType = rt
	case types.VoidType:
		llvmReturnType = types.Void
	case *types.VoidType:
		llvmReturnType = types.Void
	case *FutureType, *ChanType:
		// Future和Channel类型在运行时都是i8*指针
		llvmReturnType = types.NewPointer(types.I8)
	default:
		// 默认返回int类型
		llvmReturnType = types.I32
	}

	// 转换参数类型
	var llvmParamTypes []types.Type
	for _, pt := range paramTypes {
		switch paramType := pt.(type) {
		case types.Type:
			llvmParamTypes = append(llvmParamTypes, paramType)
		case *types.IntType:
			llvmParamTypes = append(llvmParamTypes, paramType)
		case *types.PointerType:
			llvmParamTypes = append(llvmParamTypes, paramType)
		case *types.FloatType:
			llvmParamTypes = append(llvmParamTypes, paramType)
		case *types.ArrayType:
			llvmParamTypes = append(llvmParamTypes, paramType)
		case *FutureType, *ChanType:
			// Future和Channel类型在运行时都是i8*指针
			llvmParamTypes = append(llvmParamTypes, types.NewPointer(types.I8))
		default:
			// 默认返回指针类型（用于其他不透明类型）
			llvmParamTypes = append(llvmParamTypes, types.NewPointer(types.I8))
		}
	}

	// 创建函数参数
	var params []*ir.Param
	for i, paramType := range llvmParamTypes {
		param := ir.NewParam(fmt.Sprintf("param%d", i), paramType)
		params = append(params, param)
	}

	// 检查函数是否已存在（避免重复创建，特别是main函数）
	existingFunc := m.findFunction(name)
	if existingFunc != nil {
		return existingFunc, nil
	}

	// 创建函数
	fn := ir.NewFunc(name, llvmReturnType, params...)

	// 添加到模块
	m.module.Funcs = append(m.module.Funcs, fn)

	return fn, nil
}

// GetCurrentFunction 获取当前函数
func (m *IRModuleManagerImpl) GetCurrentFunction() interface{} {
	return m.currentFunc
}

// SetCurrentFunction 设置当前函数
func (m *IRModuleManagerImpl) SetCurrentFunction(fn interface{}) error {
	if llvmFunc, ok := fn.(*ir.Func); ok {
		m.currentFunc = llvmFunc
		// 重置变量名计数器，每个函数有独立的变量命名空间
		m.variableCounters = make(map[string]int)
		return nil
	}
	return fmt.Errorf("invalid function type")
}

// CreateBasicBlock 创建基本块
func (m *IRModuleManagerImpl) CreateBasicBlock(name string) (interface{}, error) {
	if m.currentFunc == nil {
		return nil, fmt.Errorf("no current function set")
	}

	block := ir.NewBlock(name)
	m.currentFunc.Blocks = append(m.currentFunc.Blocks, block)
	return block, nil
}

// GetCurrentBasicBlock 获取当前基本块
func (m *IRModuleManagerImpl) GetCurrentBasicBlock() interface{} {
	return m.currentBlock
}

// SetCurrentBasicBlock 设置当前基本块
func (m *IRModuleManagerImpl) SetCurrentBasicBlock(block interface{}) error {
	if llvmBlock, ok := block.(*ir.Block); ok {
		m.currentBlock = llvmBlock
		return nil
	}
	return fmt.Errorf("invalid block type")
}

// CreateAlloca 创建alloca指令
func (m *IRModuleManagerImpl) CreateAlloca(typ interface{}, name string) (interface{}, error) {
	if m.currentBlock == nil {
		return nil, fmt.Errorf("no current basic block set")
	}

	llvmType, ok := typ.(types.Type)
	if !ok {
		return nil, fmt.Errorf("invalid type for alloca: %T", typ)
	}

	// 生成唯一的变量名，避免同名变量冲突
	uniqueName := m.generateUniqueVariableName(name)

	alloca := m.currentBlock.NewAlloca(llvmType)
	alloca.SetName(uniqueName)
	return alloca, nil
}

// generateUniqueVariableName 生成唯一的变量名
// 为每个变量名添加计数器后缀（如 i_0, i_1, i_2），确保所有变量名都是唯一的
func (m *IRModuleManagerImpl) generateUniqueVariableName(baseName string) string {
	// 获取当前计数器值
	count := m.variableCounters[baseName]
	// 增加计数器
	m.variableCounters[baseName] = count + 1
	// 生成唯一名称（即使是第一次使用也添加 _0 后缀）
	return fmt.Sprintf("%s_%d", baseName, count)
}

// CreateStore 创建store指令
func (m *IRModuleManagerImpl) CreateStore(value interface{}, ptr interface{}) error {
	if m.currentBlock == nil {
		return fmt.Errorf("no current basic block set")
	}

	var llvmValue llvalue.Value
	var llvmPtr llvalue.Value

	if v, ok := value.(llvalue.Value); ok {
		llvmValue = v
	} else if v, ok := value.(*ir.Global); ok {
		llvmValue = v
	} else if v, ok := value.(*constant.Int); ok {
		llvmValue = v
	} else {
		return fmt.Errorf("invalid value type for store: %T", value)
	}

	if p, ok := ptr.(llvalue.Value); ok {
		llvmPtr = p
	} else {
		return fmt.Errorf("invalid pointer type for store")
	}

	m.currentBlock.NewStore(llvmValue, llvmPtr)
	return nil
}

// CreateLoad 创建load指令
func (m *IRModuleManagerImpl) CreateLoad(typ interface{}, ptr interface{}, name string) (interface{}, error) {
	if m.currentBlock == nil {
		return nil, fmt.Errorf("no current basic block set")
	}

	var llvmType types.Type
	switch t := typ.(type) {
	case *types.IntType:
		llvmType = t
	case *types.PointerType:
		llvmType = t
	case types.Type:
		llvmType = t
	default:
		// 无法识别的类型，尝试转换为types.Type
		if t, ok := typ.(types.Type); ok {
			llvmType = t
		} else {
			return nil, fmt.Errorf("unsupported type for load: %T", typ)
		}
	}

	llvmPtr, ok := ptr.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("invalid pointer type for load")
	}

	load := m.currentBlock.NewLoad(llvmType, llvmPtr)
	// 生成唯一的变量名，避免同名变量冲突
	uniqueName := m.generateUniqueVariableName(name)
	load.SetName(uniqueName)
	return load, nil
}

// CreateRet 创建return指令
func (m *IRModuleManagerImpl) CreateRet(value interface{}) error {
	if m.currentBlock == nil {
		return fmt.Errorf("no current basic block set")
	}

	if value == nil {
		m.currentBlock.NewRet(nil)
		return nil
	}

	if llvmValue, ok := value.(llvalue.Value); ok {
		m.currentBlock.NewRet(llvmValue)
		return nil
	}

	return fmt.Errorf("invalid return value type")
}

// CreateCall 创建函数调用指令
func (m *IRModuleManagerImpl) CreateCall(fn interface{}, args ...interface{}) (interface{}, error) {
	if m.currentBlock == nil {
		return nil, fmt.Errorf("no current basic block set")
	}

	var llvmFunc *ir.Func

	// 支持字符串函数名
	if funcName, ok := fn.(string); ok {
		// 在模块中查找函数
		for _, f := range m.module.Funcs {
			if f.Name() == funcName {
				llvmFunc = f
				break
			}
		}
		if llvmFunc == nil {
			return nil, fmt.Errorf("function %s not found in module", funcName)
		}
	} else if f, ok := fn.(*ir.Func); ok {
		llvmFunc = f
	} else {
		return nil, fmt.Errorf("invalid function type for call: %T", fn)
	}

	var llvmArgs []llvalue.Value
	for _, arg := range args {
		if llvmArg, ok := arg.(llvalue.Value); ok {
			llvmArgs = append(llvmArgs, llvmArg)
		} else {
			return nil, fmt.Errorf("invalid argument type for call")
		}
	}

	call := m.currentBlock.NewCall(llvmFunc, llvmArgs...)
	return call, nil
}

// CreateBinaryOp 创建二元运算指令
func (m *IRModuleManagerImpl) CreateBinaryOp(op string, left interface{}, right interface{}, name string) (interface{}, error) {
	if m.currentBlock == nil {
		return nil, fmt.Errorf("no current basic block set")
	}

	llvmLeft, ok := left.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("invalid left operand type")
	}

	llvmRight, ok := right.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("invalid right operand type")
	}

	var result value.Value
	switch op {
	case "+":
		// 检查是否为字符串拼接
		leftTypeStr := llvmLeft.Type().String()
		rightTypeStr := llvmRight.Type().String()
		fmt.Printf("DEBUG: Binary op + with types: left=%s, right=%s\n", leftTypeStr, rightTypeStr)
		
		leftIsString := strings.HasSuffix(leftTypeStr, "*")
		rightIsString := strings.HasSuffix(rightTypeStr, "*")
		leftIsInt := strings.HasPrefix(leftTypeStr, "i32") || strings.HasPrefix(leftTypeStr, "i64")
		rightIsInt := strings.HasPrefix(rightTypeStr, "i32") || strings.HasPrefix(rightTypeStr, "i64")
		
		if leftIsString || rightIsString {
			// 字符串拼接：需要将整数转换为字符串
			var leftStr, rightStr llvalue.Value
			
			if leftIsString {
				leftStr = llvmLeft
			} else if leftIsInt {
				// 将整数转换为字符串
				intToStrFunc, exists := m.externalFunctions["int_to_string"]
				if !exists {
					// 声明 int_to_string 函数
					intToStrFunc = m.module.NewFunc("int_to_string",
						types.NewPointer(types.I8), // 返回字符串指针
						ir.NewParam("value", types.I32), // 整数参数
					)
					m.externalFunctions["int_to_string"] = intToStrFunc
				}
				callResult, err := m.CreateCall(intToStrFunc, llvmLeft)
				if err != nil {
					return nil, fmt.Errorf("failed to convert int to string: %w", err)
				}
				leftStr = callResult.(llvalue.Value)
			} else {
				return nil, fmt.Errorf("unsupported type for string concatenation: %s", leftTypeStr)
			}
			
			if rightIsString {
				rightStr = llvmRight
			} else if rightIsInt {
				// 将整数转换为字符串
				intToStrFunc, exists := m.externalFunctions["int_to_string"]
				if !exists {
					// 声明 int_to_string 函数
					intToStrFunc = m.module.NewFunc("int_to_string",
						types.NewPointer(types.I8), // 返回字符串指针
						ir.NewParam("value", types.I32), // 整数参数
					)
					m.externalFunctions["int_to_string"] = intToStrFunc
				}
				callResult, err := m.CreateCall(intToStrFunc, llvmRight)
				if err != nil {
					return nil, fmt.Errorf("failed to convert int to string: %w", err)
				}
				rightStr = callResult.(llvalue.Value)
			} else {
				return nil, fmt.Errorf("unsupported type for string concatenation: %s", rightTypeStr)
			}
			
			// 调用 string_concat 函数
			fmt.Printf("DEBUG: Detected string concatenation, calling string_concat\n")
			callResult, err := m.CreateCall("string_concat", leftStr, rightStr)
			if err != nil {
				return nil, fmt.Errorf("failed to create string concatenation call: %w", err)
			}
			result = callResult.(llvalue.Value)
		} else {
			fmt.Printf("DEBUG: Performing numeric addition\n")
			// 数值加法
			result = m.currentBlock.NewAdd(llvmLeft, llvmRight)
		}
	case "-":
		result = m.currentBlock.NewSub(llvmLeft, llvmRight)
	case "*":
		result = m.currentBlock.NewMul(llvmLeft, llvmRight)
	case "/":
		result = m.currentBlock.NewSDiv(llvmLeft, llvmRight)
	case "%":
		// 模运算符：使用 SRem（有符号取余）
		result = m.currentBlock.NewSRem(llvmLeft, llvmRight)
	case "==":
		result = m.currentBlock.NewICmp(enum.IPredEQ, llvmLeft, llvmRight)
	case "!=":
		result = m.currentBlock.NewICmp(enum.IPredNE, llvmLeft, llvmRight)
	case "<":
		result = m.currentBlock.NewICmp(enum.IPredSLT, llvmLeft, llvmRight)
	case ">":
		result = m.currentBlock.NewICmp(enum.IPredSGT, llvmLeft, llvmRight)
	case "<=":
		result = m.currentBlock.NewICmp(enum.IPredSLE, llvmLeft, llvmRight)
	case ">=":
		result = m.currentBlock.NewICmp(enum.IPredSGE, llvmLeft, llvmRight)
	default:
		return nil, fmt.Errorf("unsupported binary operator: %s", op)
	}

	if name != "" {
		// 生成唯一的变量名，避免同名变量冲突
		uniqueName := m.generateUniqueVariableName(name)
		result.(value.Named).SetName(uniqueName)
	}

	return result, nil
}

// CreateBr 创建无条件分支指令
func (m *IRModuleManagerImpl) CreateBr(dest interface{}) error {
	if m.currentBlock == nil {
		return fmt.Errorf("no current basic block set")
	}

	llvmDest, ok := dest.(*ir.Block)
	if !ok {
		return fmt.Errorf("invalid destination block type")
	}

	m.currentBlock.NewBr(llvmDest)
	return nil
}

// CreateCondBr 创建条件分支指令
func (m *IRModuleManagerImpl) CreateCondBr(cond interface{}, trueDest interface{}, falseDest interface{}) error {
	if m.currentBlock == nil {
		return fmt.Errorf("no current basic block set")
	}

	llvmCond, ok := cond.(llvalue.Value)
	if !ok {
		return fmt.Errorf("invalid condition type")
	}

	llvmTrueDest, ok := trueDest.(*ir.Block)
	if !ok {
		return fmt.Errorf("invalid true destination type")
	}

	llvmFalseDest, ok := falseDest.(*ir.Block)
	if !ok {
		return fmt.Errorf("invalid false destination type")
	}

	m.currentBlock.NewCondBr(llvmCond, llvmTrueDest, llvmFalseDest)
	return nil
}

// CreateGetElementPtr 创建GetElementPtr指令
func (m *IRModuleManagerImpl) CreateGetElementPtr(elemType interface{}, ptr interface{}, indices ...interface{}) (interface{}, error) {
	if m.currentBlock == nil {
		return nil, fmt.Errorf("no current basic block set")
	}

	llvmElemType, ok := elemType.(types.Type)
	if !ok {
		return nil, fmt.Errorf("invalid element type for GEP")
	}

	llvmPtr, ok := ptr.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("invalid pointer type for GEP")
	}

	var llvmIndices []llvalue.Value
	for _, idx := range indices {
		if llvmIdx, ok := idx.(llvalue.Value); ok {
			llvmIndices = append(llvmIndices, llvmIdx)
		} else {
			return nil, fmt.Errorf("invalid index type for GEP")
		}
	}

	return m.currentBlock.NewGetElementPtr(llvmElemType, llvmPtr, llvmIndices...), nil
}

// CreateBitCast 创建bitcast指令
func (m *IRModuleManagerImpl) CreateBitCast(value interface{}, targetType interface{}, name string) (interface{}, error) {
	if m.currentBlock == nil {
		return nil, fmt.Errorf("no current basic block set")
	}

	llvmValue, ok := value.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("invalid value type for bitcast")
	}

	llvmTargetType, ok := targetType.(types.Type)
	if !ok {
		return nil, fmt.Errorf("invalid target type for bitcast")
	}

	bitcast := m.currentBlock.NewBitCast(llvmValue, llvmTargetType)
	if name != "" {
		// 生成唯一的变量名，避免同名变量冲突
		uniqueName := m.generateUniqueVariableName(name)
		bitcast.SetName(uniqueName)
	}
	return bitcast, nil
}

// GetIRString 获取最终的IR字符串
func (m *IRModuleManagerImpl) GetIRString() string {
	return m.module.String()
}

// Validate 验证IR模块完整性
func (m *IRModuleManagerImpl) Validate() error {
	// 基本验证：确保有至少一个函数
	if len(m.module.Funcs) == 0 {
		return fmt.Errorf("no functions defined in module")
	}

	// TODO: 更详细的验证
	return nil
}

// InitializeMainFunction 初始化主函数
func (m *IRModuleManagerImpl) InitializeMainFunction() error {
	// 声明外部函数（打印函数和协程运行时函数）
	m.declareExternalFunctions()

	return nil
}

// AddStringConstant 添加字符串常量，返回全局变量
func (m *IRModuleManagerImpl) AddStringConstant(s string) (*ir.Global, error) {
	// 创建唯一名称
	m.stringCount++
	counter := int32(m.stringCount)
	name := fmt.Sprintf(".str.%d", counter)

	// 创建字符串常量数组
	strConst := constant.NewCharArrayFromString(s)

	// 创建全局变量
	global := m.module.NewGlobal(name, strConst.Type())
	global.Init = strConst
	// global.Linkage = ir.LinkagePrivate // 暂时注释，可能不可用
	// global.IsConstant = true // 暂时注释，可能不可用

	return global, nil
}

// Finalize 完成IR构建
func (m *IRModuleManagerImpl) Finalize() error {
	// 为所有函数的所有基本块添加terminator（如果缺失）
	for _, fn := range m.module.Funcs {
		// 检查所有基本块，确保每个基本块都有terminator
		for i, block := range fn.Blocks {
			if block.Term == nil {
				// 如果基本块没有terminator，需要添加
				// 检查是否是最后一个基本块
				isLastBlock := (i == len(fn.Blocks)-1)
				
				// 检查基本块名称，判断是否是控制流结束块（如 while.end, if.end）
				// 注意：ir.Block 没有 Name() 方法，我们需要通过其他方式判断
				// 暂时简化处理：对于非最后一个基本块，如果有后续基本块，跳转到下一个
				
				if isLastBlock {
					// 最后一个基本块：根据函数返回类型添加ret指令
					if fn.Sig.RetType == types.Void {
						block.NewRet(nil)
					} else if fn.Sig.RetType == types.I32 {
						block.NewRet(constant.NewInt(types.I32, 0))
					} else {
						// 对于其他类型，暂时添加unreachable
						block.NewUnreachable()
					}
				} else {
					// 非最后一个基本块缺少terminator
					// 检查是否有后续基本块，如果有，跳转到下一个
					// 这通常发生在控制流结束块（如 while.end, if.end）之后还有语句的情况
					if i+1 < len(fn.Blocks) {
						nextBlock := fn.Blocks[i+1]
						block.NewBr(nextBlock)
					} else {
						// 没有后续基本块，添加ret指令
						if fn.Sig.RetType == types.Void {
							block.NewRet(nil)
						} else if fn.Sig.RetType == types.I32 {
							block.NewRet(constant.NewInt(types.I32, 0))
						} else {
							block.NewUnreachable()
						}
					}
					// 其他情况：非最后一个基本块缺少terminator，这通常不应该发生
					// 但为了安全，添加unreachable
					block.NewUnreachable()
				}
			}
		}
	}

	// 确保main函数存在并有正确的terminator
	mainFunc := m.findFunction("main")
	if mainFunc == nil {
		// 如果没有main函数，创建一个默认的
		err := m.createMainFunction()
		if err != nil {
			return fmt.Errorf("failed to create main function: %v", err)
		}
		mainFunc = m.findFunction("main")
	}

	// 确保main函数有正确的terminator
	if mainFunc != nil && len(mainFunc.Blocks) > 0 {
		lastBlock := mainFunc.Blocks[len(mainFunc.Blocks)-1]
		if lastBlock.Term == nil {
			// 在return之前调用run_scheduler启动协程调度器
			if runSchedulerFunc, exists := m.externalFunctions["run_scheduler"]; exists {
				lastBlock.NewCall(runSchedulerFunc)
			}

			if mainFunc.Sig.RetType == types.I32 {
				lastBlock.NewRet(constant.NewInt(types.I32, 0))
			} else if mainFunc.Sig.RetType == types.Void {
				lastBlock.NewRet(nil)
			}
		} else {
			// 如果已经有terminator，修改最后的block
			blocks := mainFunc.Blocks
			if len(blocks) > 0 {
				lastBlock := blocks[len(blocks)-1]

				// 检查block是否有terminator，如果有则修改
				instructions := lastBlock.Insts
				if len(instructions) > 0 {
					lastInst := instructions[len(instructions)-1]
					// 检查是否是return指令（通过类型名）
					if strings.Contains(fmt.Sprintf("%T", lastInst), "Ret") {
						// 移除return指令
						lastBlock.Insts = instructions[:len(instructions)-1]

						// 在return之前调用run_scheduler
						if runSchedulerFunc, exists := m.externalFunctions["run_scheduler"]; exists {
							lastBlock.NewCall(runSchedulerFunc)
						}

						// 重新添加return指令
						if mainFunc.Sig.RetType == types.I32 {
							lastBlock.NewRet(constant.NewInt(types.I32, 0))
						} else if mainFunc.Sig.RetType == types.Void {
							lastBlock.NewRet(nil)
						}
					} else {
						// 如果没有return指令，直接添加run_scheduler调用
						if runSchedulerFunc, exists := m.externalFunctions["run_scheduler"]; exists {
							lastBlock.NewCall(runSchedulerFunc)
						}
					}
				} else {
					// 如果block为空，添加run_scheduler调用和return
					if runSchedulerFunc, exists := m.externalFunctions["run_scheduler"]; exists {
						lastBlock.NewCall(runSchedulerFunc)
					}
					if mainFunc.Sig.RetType == types.I32 {
						lastBlock.NewRet(constant.NewInt(types.I32, 0))
					} else if mainFunc.Sig.RetType == types.Void {
						lastBlock.NewRet(nil)
					}
				}
			}
		}
	}

	return nil
}

// createMainFunction 创建main函数
func (m *IRModuleManagerImpl) createMainFunction() error {
	// 检查是否已经有main函数
	existingMain := m.findFunction("main")
	if existingMain != nil {
		// 如果用户定义了main函数，确保它有正确的返回类型和terminator
		if existingMain.Sig.RetType != types.I32 {
			return fmt.Errorf("user-defined main function must return i32")
		}
		// 确保有terminator（已在Finalize中处理）
		return nil
	}

	// 用户没有定义main函数，创建一个空的
	// 注意：m.module.NewFunc 可能已经自动添加到模块，所以先检查
	var mainFunc *ir.Func
	mainFunc = m.findFunction("main")
	if mainFunc == nil {
		mainFunc = m.module.NewFunc("main", types.I32)
		// 只有在函数不存在时才添加到模块（NewFunc可能已经自动添加）
		if m.findFunction("main") == nil {
			m.module.Funcs = append(m.module.Funcs, mainFunc)
		} else {
			// 如果NewFunc已经自动添加，重新获取
			mainFunc = m.findFunction("main")
		}
	}

	entryBlock := mainFunc.NewBlock("entry")
	entryBlock.NewRet(constant.NewInt(types.I32, 0))
	return nil
}

// findFunction 在模块中查找函数
func (m *IRModuleManagerImpl) findFunction(name string) *ir.Func {
	for _, fn := range m.module.Funcs {
		if fn.Name() == name {
			return fn
		}
	}
	return nil
}

// declareExternalFunctions 声明外部函数
func (m *IRModuleManagerImpl) declareExternalFunctions() {
	// 声明打印函数
	printIntFunc := m.module.NewFunc("print_int", types.Void, ir.NewParam("value", types.I32))
	printStringFunc := m.module.NewFunc("print_string", types.Void, ir.NewParam("str", types.NewPointer(types.I8)))

	// 将函数存储在缓存中，便于后续查找
	m.externalFunctions["print_int"] = printIntFunc
	m.externalFunctions["print_string"] = printStringFunc
}

// GetExternalFunction 获取外部函数
func (m *IRModuleManagerImpl) GetExternalFunction(name string) (interface{}, bool) {
	fn, exists := m.externalFunctions[name]
	return fn, exists
}

// GetFunction 获取模块中的函数（包括外部函数和模块内函数）
func (m *IRModuleManagerImpl) GetFunction(name string) (interface{}, bool) {
	// 先检查外部函数
	if fn, exists := m.externalFunctions[name]; exists {
		return fn, true
	}

	// 再检查模块内的函数
	for _, fn := range m.module.Funcs {
		if fn.Name() == name {
			return fn, true
		}
	}

	return nil, false
}

// RegisterFunction 注册函数到外部函数缓存中
func (m *IRModuleManagerImpl) RegisterFunction(name string, fn interface{}) error {
	if funcPtr, ok := fn.(*ir.Func); ok {
		m.externalFunctions[name] = funcPtr
		return nil
	}
	return fmt.Errorf("invalid function type for registration: %T", fn)
}

// RegisterExternalFunction 注册外部函数（与RegisterFunction相同，用于接口兼容性）
func (m *IRModuleManagerImpl) RegisterExternalFunction(name string, fn interface{}) error {
	return m.RegisterFunction(name, fn)
}

// declareCoroutineRuntimeFunctions 声明协程运行时函数
func (m *IRModuleManagerImpl) declareCoroutineRuntimeFunctions() {
	// 协程管理函数
	spawnFunc := m.module.NewFunc("coroutine_spawn",
		types.NewPointer(types.I8), // 返回协程句柄
		ir.NewParam("entry", types.NewPointer(&types.FuncType{RetType: types.Void, Params: []types.Type{types.NewPointer(types.I8)}})), // 入口函数指针类型
		ir.NewParam("arg_count", types.I32),               // 参数数量
		ir.NewParam("args", types.NewPointer(types.I8)),   // 参数数组
		ir.NewParam("future", types.NewPointer(types.I8)), // Future指针
	)
	m.externalFunctions["coroutine_spawn"] = spawnFunc

	awaitFunc := m.module.NewFunc("coroutine_await",
		types.NewPointer(types.I8),                        // 返回结果
		ir.NewParam("future", types.NewPointer(types.I8)), // Future指针
	)
	m.externalFunctions["coroutine_await"] = awaitFunc

	// Future管理函数
	futureNewFunc := m.module.NewFunc("future_new",
		types.NewPointer(types.I8), // 返回Future指针
	)
	m.externalFunctions["future_new"] = futureNewFunc

	futureResolveFunc := m.module.NewFunc("future_resolve",
		types.Void,
		ir.NewParam("future", types.NewPointer(types.I8)), // Future指针
		ir.NewParam("value", types.NewPointer(types.I8)),  // 结果值
	)
	m.externalFunctions["future_resolve"] = futureResolveFunc

	// 调度器yield函数
	schedulerYieldFunc := m.module.NewFunc("scheduler_yield",
		types.Void, // 无返回值
	)
	m.externalFunctions["scheduler_yield"] = schedulerYieldFunc

	// 通道管理函数
	channelCreateFunc := m.module.NewFunc("channel_create",
		types.NewPointer(types.I8), // 返回通道指针
	)
	m.externalFunctions["channel_create"] = channelCreateFunc

	channelSendFunc := m.module.NewFunc("channel_send",
		types.Void,
		ir.NewParam("channel", types.NewPointer(types.I8)), // 通道指针
		ir.NewParam("value", types.NewPointer(types.I8)),   // 发送的值
	)
	m.externalFunctions["channel_send"] = channelSendFunc

	channelReceiveFunc := m.module.NewFunc("channel_receive",
		types.NewPointer(types.I8),                         // 返回接收的值
		ir.NewParam("channel", types.NewPointer(types.I8)), // 通道指针
	)
	m.externalFunctions["channel_receive"] = channelReceiveFunc

	channelSelectFunc := m.module.NewFunc("channel_select",
		types.I32, // 返回选择的case索引
		ir.NewParam("channels", types.NewPointer(types.I8)),   // 通道数组
		ir.NewParam("operations", types.NewPointer(types.I8)), // 操作数组
		ir.NewParam("count", types.I32),                       // case数量
	)
	m.externalFunctions["channel_select"] = channelSelectFunc

	channelCloseFunc := m.module.NewFunc("channel_close",
		types.Void,
		ir.NewParam("channel", types.NewPointer(types.I8)), // 通道指针
	)
	m.externalFunctions["channel_close"] = channelCloseFunc

	channelDestroyFunc := m.module.NewFunc("channel_destroy",
		types.Void,
		ir.NewParam("channel", types.NewPointer(types.I8)), // 通道指针
	)
	m.externalFunctions["channel_destroy"] = channelDestroyFunc

	// 字符串拼接函数
	stringConcatFunc := m.module.NewFunc("string_concat",
		types.NewPointer(types.I8),                       // 返回拼接后的字符串指针
		ir.NewParam("left", types.NewPointer(types.I8)),  // 左侧字符串
		ir.NewParam("right", types.NewPointer(types.I8)), // 右侧字符串
	)
	m.externalFunctions["string_concat"] = stringConcatFunc

	// 整数转字符串函数
	intToStringFunc := m.module.NewFunc("int_to_string",
		types.NewPointer(types.I8),  // 返回字符串指针
		ir.NewParam("value", types.I32), // 整数参数
	)
	m.externalFunctions["int_to_string"] = intToStringFunc

	// 运行调度器函数
	runSchedulerFunc := m.module.NewFunc("run_scheduler",
		types.Void, // 无返回值
	)
	m.externalFunctions["run_scheduler"] = runSchedulerFunc

	futureRejectFunc := m.module.NewFunc("future_reject",
		types.Void,
		ir.NewParam("future", types.NewPointer(types.I8)), // Future指针
		ir.NewParam("error", types.NewPointer(types.I8)),  // 错误信息
	)
	m.externalFunctions["future_reject"] = futureRejectFunc
}
