package impl

import (
	"fmt"
	"strings"

	"echo/internal/modules/frontend/domain/entities"

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
	// ✅ 修复：为每个函数维护独立的变量名计数器，避免在切换函数时丢失计数器状态
	functionVariableCounters map[string]map[string]int // 函数名 -> (变量名 -> 计数器值)
	// 当前函数的类型参数映射（用于泛型函数中的trait方法调用）
	// 例如：对于bucket_index[K, V]，如果调用时是bucket_index[int, string]，则map为{"K": "int", "V": "string"}
	currentFuncTypeParams map[string]string // 泛型类型参数名 -> 实际类型
	// 函数定义的类型参数列表（用于从参数类型推断类型参数）
	// 例如：bucket_index -> ["K", "V"]
	functionTypeParamNames map[string][]string // 函数名 -> 类型参数名列表
	// 存储泛型函数定义，供延迟编译使用
	genericFunctionDefinitions map[string]*entities.FuncDef // 函数名 -> 函数定义
	// 记录已编译的单态化函数，避免重复编译
	compiledMonomorphizedFunctions map[string]bool // 单态化函数名 -> 是否已编译
}

// NewIRModuleManagerImpl 创建IR模块管理器实现
func NewIRModuleManagerImpl() *IRModuleManagerImpl {
	manager := &IRModuleManagerImpl{
		module:                        ir.NewModule(),
		stringCount:                   0,
		externalFunctions:             make(map[string]*ir.Func),
		variableCounters:              make(map[string]int),
		functionVariableCounters:      make(map[string]map[string]int), // ✅ 修复：为每个函数维护独立的计数器
		currentFuncTypeParams:         make(map[string]string),
		functionTypeParamNames:        make(map[string][]string),
		genericFunctionDefinitions:    make(map[string]*entities.FuncDef),
		compiledMonomorphizedFunctions: make(map[string]bool),
	}

	// 声明协程运行时函数
	manager.declareCoroutineRuntimeFunctions()

	return manager
}

// AddGlobalVariable 添加全局变量
func (m *IRModuleManagerImpl) AddGlobalVariable(name string, value interface{}) error {
	// 暂时不支持通过此接口添加（使用 CreateGlobalVariable）
	return fmt.Errorf("global variables not yet supported")
}

// CreateGlobalVariable 创建模块级全局变量，返回全局指针供符号表注册；任意函数内可 Load/Store。
func (m *IRModuleManagerImpl) CreateGlobalVariable(typ interface{}, name string) (interface{}, error) {
	llvmType, ok := typ.(types.Type)
	if !ok {
		return nil, fmt.Errorf("invalid type for global variable: %T", typ)
	}
	// 模块内唯一名，避免与函数名冲突
	globalName := "g." + name
	if strings.HasPrefix(name, "g.") {
		globalName = name
	}
	global := m.module.NewGlobal(globalName, llvmType)
	global.Init = constant.NewZeroInitializer(llvmType)
	return global, nil
}

// GetCurrentFunctionName 返回当前函数名，空表示无当前函数。
func (m *IRModuleManagerImpl) GetCurrentFunctionName() string {
	if m.currentFunc == nil {
		return ""
	}
	return m.currentFunc.Name()
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
		funcName := llvmFunc.Name()
		
		// ✅ 修复：保存当前函数的计数器状态（如果存在）
		if m.currentFunc != nil {
			oldFuncName := m.currentFunc.Name()
			// 保存当前函数的计数器状态
			if m.functionVariableCounters[oldFuncName] == nil {
				m.functionVariableCounters[oldFuncName] = make(map[string]int)
			}
			// 复制当前计数器的值
			for k, v := range m.variableCounters {
				m.functionVariableCounters[oldFuncName][k] = v
			}
		}
		
		// ✅ 修复：恢复目标函数的计数器状态（如果存在），否则创建新的
		if m.functionVariableCounters[funcName] != nil {
			// 恢复之前保存的计数器状态
			m.variableCounters = make(map[string]int)
			for k, v := range m.functionVariableCounters[funcName] {
				m.variableCounters[k] = v
			}
		} else {
			// 新函数，初始化计数器
			m.variableCounters = make(map[string]int)
			m.functionVariableCounters[funcName] = make(map[string]int)
		}
		
		// 重置类型参数映射（新函数开始时清空）
		m.currentFuncTypeParams = make(map[string]string)
		m.currentFunc = llvmFunc
		// ✅ 修复 use of undefined value：切换函数时同时切换到该函数的 entry block，
		// 否则 currentBlock 可能仍是上一函数的块，导致 alloca 插入到错误函数中。
		if len(llvmFunc.Blocks) > 0 {
			m.currentBlock = llvmFunc.Blocks[0]
		}
		return nil
	}
	return fmt.Errorf("invalid function type")
}

// SetCurrentFunctionTypeParams 设置当前函数的类型参数映射
// 例如：对于bucket_index[K, V]，如果调用时是bucket_index[int, string]，则typeParams为{"K": "int", "V": "string"}
func (m *IRModuleManagerImpl) SetCurrentFunctionTypeParams(typeParams map[string]string) {
	m.currentFuncTypeParams = typeParams
}

// GetCurrentFunctionTypeParams 获取当前函数的类型参数映射
func (m *IRModuleManagerImpl) GetCurrentFunctionTypeParams() map[string]string {
	return m.currentFuncTypeParams
}

// SetFunctionTypeParamNames 设置函数定义的类型参数列表
// 例如：bucket_index -> ["K", "V"]
func (m *IRModuleManagerImpl) SetFunctionTypeParamNames(funcName string, typeParams []string) {
	m.functionTypeParamNames[funcName] = typeParams
}

// GetFunctionTypeParamNames 获取函数定义的类型参数列表
func (m *IRModuleManagerImpl) GetFunctionTypeParamNames(funcName string) []string {
	return m.functionTypeParamNames[funcName]
}

// SaveGenericFunctionDefinition 保存泛型函数定义
func (m *IRModuleManagerImpl) SaveGenericFunctionDefinition(funcName string, funcDef *entities.FuncDef) {
	m.genericFunctionDefinitions[funcName] = funcDef
}

// GetGenericFunctionDefinition 获取泛型函数定义
func (m *IRModuleManagerImpl) GetGenericFunctionDefinition(funcName string) (*entities.FuncDef, bool) {
	funcDef, exists := m.genericFunctionDefinitions[funcName]
	return funcDef, exists
}

// IsMonomorphizedFunctionCompiled 检查单态化函数是否已编译
func (m *IRModuleManagerImpl) IsMonomorphizedFunctionCompiled(monoName string) bool {
	return m.compiledMonomorphizedFunctions[monoName]
}

// MarkMonomorphizedFunctionCompiled 标记单态化函数已编译
func (m *IRModuleManagerImpl) MarkMonomorphizedFunctionCompiled(monoName string) {
	m.compiledMonomorphizedFunctions[monoName] = true
}

// CreateBasicBlock 创建基本块
func (m *IRModuleManagerImpl) CreateBasicBlock(name string) (interface{}, error) {
	if m.currentFunc == nil {
		return nil, fmt.Errorf("no current function set")
	}

	// ✅ 修复：检查 entry 块是否已存在（避免重复创建）
	if name == "entry" {
		// 查找是否已经存在 entry 块
		for _, block := range m.currentFunc.Blocks {
			if block.Name() == "entry" {
				// entry 块已存在，返回现有的块
				return block, nil
			}
		}
	}

	block := ir.NewBlock(name)
	m.currentFunc.Blocks = append(m.currentFunc.Blocks, block)
	return block, nil
}

// GetBlockByName 根据名称获取当前函数中的基本块（用于 break/continue 跳转）
func (m *IRModuleManagerImpl) GetBlockByName(name string) (interface{}, error) {
	if m.currentFunc == nil {
		return nil, fmt.Errorf("no current function set")
	}
	for _, block := range m.currentFunc.Blocks {
		if block.Name() == name {
			return block, nil
		}
	}
	return nil, fmt.Errorf("block not found: %s", name)
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
	// 增加计数器（先递增，再使用）
	m.variableCounters[baseName] = count + 1
	// ✅ 修复：同时更新函数级别的计数器
	if m.currentFunc != nil {
		funcName := m.currentFunc.Name()
		if m.functionVariableCounters[funcName] == nil {
			m.functionVariableCounters[funcName] = make(map[string]int)
		}
		m.functionVariableCounters[funcName][baseName] = m.variableCounters[baseName]
	}
	// 生成唯一名称（即使是第一次使用也添加 _0 后缀）
	uniqueName := fmt.Sprintf("%s_%d", baseName, count)
	// ✅ 添加调试输出：检查变量名生成
	if strings.Contains(baseName, "userMap") {
		fmt.Printf("DEBUG: generateUniqueVariableName - baseName=%s, count=%d, uniqueName=%s, nextCount=%d\n", baseName, count, uniqueName, m.variableCounters[baseName])
	}
	return uniqueName
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

	// 获取源值的类型和目标指针指向的类型
	srcType := llvmValue.Type()
	ptrType, ok := llvmPtr.Type().(*types.PointerType)
	if !ok {
		return fmt.Errorf("target is not a pointer type: %T", llvmPtr.Type())
	}
	dstType := ptrType.ElemType

	// 如果类型不匹配，进行类型转换
	if !types.Equal(srcType, dstType) {
		convertedValue, err := m.convertValueType(llvmValue, srcType, dstType)
		if err != nil {
			return fmt.Errorf("failed to convert value type from %s to %s: %w", srcType, dstType, err)
		}
		llvmValue = convertedValue
	}

	m.currentBlock.NewStore(llvmValue, llvmPtr)
	return nil
}

// convertValueType 转换值的类型
func (m *IRModuleManagerImpl) convertValueType(value llvalue.Value, srcType types.Type, dstType types.Type) (llvalue.Value, error) {
	// 如果类型已经匹配，直接返回
	if types.Equal(srcType, dstType) {
		return value, nil
	}

	// 处理整数类型转换
	srcInt, srcIsInt := srcType.(*types.IntType)
	dstInt, dstIsInt := dstType.(*types.IntType)

	if srcIsInt && dstIsInt {
		srcBits := srcInt.BitSize
		dstBits := dstInt.BitSize

		if srcBits < dstBits {
			// 扩展：小整数类型 → 大整数类型
			// 对于 u64（无符号），使用 ZExt；对于 i64（有符号），使用 SExt
			// 由于 u64 映射为 i64，我们使用 ZExt（零扩展）
			// 注意：这里简化处理，假设所有扩展都使用 ZExt（适用于无符号整数）
			// 如果需要区分有符号和无符号，需要从类型信息中获取
			ext := m.currentBlock.NewZExt(value, dstType)
			return ext, nil
		} else 		if srcBits > dstBits {
			// 截断：大整数类型 → 小整数类型
			trunc := m.currentBlock.NewTrunc(value, dstType)
			return trunc, nil
		}
	}

	// 处理浮点类型转换（float <-> double）
	// LLVM 不允许 BitCast 在 double 与 float 之间，必须用 FPExt/FPTrunc
	srcFloat, srcIsFloat := srcType.(*types.FloatType)
	dstFloat, dstIsFloat := dstType.(*types.FloatType)
	if srcIsFloat && dstIsFloat {
		srcKind := srcFloat.Kind
		dstKind := dstFloat.Kind
		if srcKind == types.FloatKindFloat && dstKind == types.FloatKindDouble {
			// float -> double: FPExt
			return m.currentBlock.NewFPExt(value, dstType), nil
		}
		if srcKind == types.FloatKindDouble && dstKind == types.FloatKindFloat {
			// double -> float: FPTrunc
			return m.currentBlock.NewFPTrunc(value, dstType), nil
		}
	}

	// 处理整数到指针的转换（int -> i8*）
	// 对于泛型类型参数（K, V），它们被映射为i8*不透明指针
	// 当传入int类型的值时，需要将其转换为i8*指针
	if srcIsInt {
		if _, dstIsPtr := dstType.(*types.PointerType); dstIsPtr {
			// int到指针的转换：需要将int值存储到内存中，然后获取指针
			// 1. 分配内存
			alloca, err := m.CreateAlloca(srcType, fmt.Sprintf("int_to_ptr_%d", m.variableCounters["temp"]))
			if err != nil {
				return nil, fmt.Errorf("failed to allocate memory for int to ptr conversion: %w", err)
			}
			m.variableCounters["temp"]++
			
			// 2. 存储int值
			allocaValue, ok := alloca.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("CreateAlloca returned invalid type: %T", alloca)
			}
			if err := m.CreateStore(value, allocaValue); err != nil {
				return nil, fmt.Errorf("failed to store int value: %w", err)
			}
			
			// 3. 转换为目标指针类型（如果需要）
			if !types.Equal(allocaValue.Type(), dstType) {
				bitcast, err := m.CreateBitCast(allocaValue, dstType, "")
				if err != nil {
					return nil, fmt.Errorf("failed to bitcast to target pointer type: %w", err)
				}
				bitcastValue, ok := bitcast.(llvalue.Value)
				if !ok {
					return nil, fmt.Errorf("CreateBitCast returned invalid type: %T", bitcast)
				}
				return bitcastValue, nil
			}
			
			return allocaValue, nil
		}
	}

	// ✅ 新增：处理结构体值到指针的转换（struct -> i8*）
	// 例如：{ i32, i8* } -> i8*（用于runtime_map_set_struct_string等）
	// ⚠️ 重要：必须在堆上分配内存，因为运行时函数会存储指针，栈上分配的内存会在函数返回后失效
	if srcStructType, srcIsStruct := srcType.(*types.StructType); srcIsStruct {
		if _, dstIsPtr := dstType.(*types.PointerType); dstIsPtr {
			// 结构体值到指针的转换：需要在堆上分配内存并拷贝结构体值
			// 1. 计算结构体大小（使用sizeof）
			structSize, err := m.getStructSize(srcStructType)
			if err != nil {
				return nil, fmt.Errorf("failed to get struct size: %w", err)
			}
			
			// 2. 在堆上分配内存（使用malloc）
			mallocFunc, exists := m.GetExternalFunction("malloc")
			var mallocFuncLLVM *ir.Func
			if !exists {
				// 声明malloc函数：i8* malloc(i64 size)
				mallocFuncLLVM = m.module.NewFunc("malloc",
					types.NewPointer(types.I8), // 返回i8*
					ir.NewParam("size", types.I64), // 参数：i64 size
				)
				m.externalFunctions["malloc"] = mallocFuncLLVM
			} else {
				// 类型断言
				var ok bool
				mallocFuncLLVM, ok = mallocFunc.(*ir.Func)
				if !ok {
					return nil, fmt.Errorf("malloc function is not *ir.Func: %T", mallocFunc)
				}
			}
			
			// 调用malloc分配内存
			structSizeConst := constant.NewInt(types.I64, structSize)
			mallocResult, err := m.CreateCall(mallocFuncLLVM, structSizeConst)
			if err != nil {
				return nil, fmt.Errorf("failed to call malloc: %w", err)
			}
			
			mallocResultValue, ok := mallocResult.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("CreateCall returned invalid type: %T", mallocResult)
			}
			
			// 3. 将malloc返回的i8*转换为结构体指针类型
			structPtrType := types.NewPointer(srcStructType)
			structPtr, err := m.CreateBitCast(mallocResultValue, structPtrType, fmt.Sprintf("struct_heap_ptr_%d", m.variableCounters["temp"]))
			if err != nil {
				return nil, fmt.Errorf("failed to bitcast malloc result to struct pointer: %w", err)
			}
			m.variableCounters["temp"]++
			
			structPtrValue, ok := structPtr.(llvalue.Value)
			if !ok {
				return nil, fmt.Errorf("CreateBitCast returned invalid type: %T", structPtr)
			}
			
			// 4. 存储结构体值到堆上分配的内存
			if err := m.CreateStore(value, structPtrValue); err != nil {
				return nil, fmt.Errorf("failed to store struct value to heap: %w", err)
			}
			
			// 5. 转换为目标指针类型（i8*）
			if !types.Equal(structPtrValue.Type(), dstType) {
				bitcast, err := m.CreateBitCast(structPtrValue, dstType, "")
				if err != nil {
					return nil, fmt.Errorf("failed to bitcast struct pointer to target type: %w", err)
				}
				bitcastValue, ok := bitcast.(llvalue.Value)
				if !ok {
					return nil, fmt.Errorf("CreateBitCast returned invalid type: %T", bitcast)
				}
				return bitcastValue, nil
			}

			return structPtrValue, nil
		}
	}

	// 处理指针类型转换（使用 BitCast）
	if srcPtrType, srcIsPtr := srcType.(*types.PointerType); srcIsPtr {
		if dstPtrType, dstIsPtr := dstType.(*types.PointerType); dstIsPtr {
			// 特殊处理：结构体指针到i8*的转换（用于MapIterResult等）
			// 例如：{ i8*, i32, float }* -> i8*
			if _, srcIsStruct := srcPtrType.ElemType.(*types.StructType); srcIsStruct {
				if dstPtrType.ElemType == types.I8 {
					// 结构体指针到i8*指针，使用BitCast
					bitcast := m.currentBlock.NewBitCast(value, dstType)
					return bitcast, nil
				}
			}
			// 其他指针类型转换
			bitcast := m.currentBlock.NewBitCast(value, dstType)
			return bitcast, nil
		}
	}

	// 其他类型转换（使用 BitCast）
	bitcast := m.currentBlock.NewBitCast(value, dstType)
	return bitcast, nil
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

	// 确保在正确的 block 中创建 load 指令
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

	// 获取函数签名，用于参数类型转换
	funcSig := llvmFunc.Sig
	
	var llvmArgs []llvalue.Value
	for i, arg := range args {
		var llvmArg llvalue.Value
		var ok bool
		
		// ✅ 支持 *ir.Func 作为函数指针参数
		if funcArg, isFunc := arg.(*ir.Func); isFunc {
			// 函数指针：直接使用函数本身（*ir.Func实现了value.Value接口）
			llvmArg = funcArg
			ok = true
		} else {
			llvmArg, ok = arg.(llvalue.Value)
		}
		
		if !ok {
			return nil, fmt.Errorf("invalid argument type for call: %T", arg)
		}
		
		// 获取期望的参数类型
		if i < len(funcSig.Params) {
			expectedType := funcSig.Params[i] // funcSig.Params是[]types.Type
			argType := llvmArg.Type()
			
			// 如果类型不匹配，尝试转换
			if !types.Equal(argType, expectedType) {
				// 使用convertValueType进行类型转换
				convertedArg, err := m.convertValueType(llvmArg, argType, expectedType)
				if err != nil {
					return nil, fmt.Errorf("failed to convert argument %d from %s to %s: %w", i, argType, expectedType, err)
				}
				llvmArgs = append(llvmArgs, convertedArg)
			} else {
				llvmArgs = append(llvmArgs, llvmArg)
			}
		} else {
			// 参数数量不匹配，直接使用原值（让LLVM报错）
			llvmArgs = append(llvmArgs, llvmArg)
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
	case "==", "!=", "<", ">", "<=", ">=":
		// ✅ 修复：浮点比较使用 FCmp，整数/指针比较使用 ICmp
		// 仅当两个操作数均为浮点且均非整数时使用 FCmp，否则使用 ICmp（避免 IntType 误入 FCmp 导致 panic）
		leftType := llvmLeft.Type()
		rightType := llvmRight.Type()
		leftIsFloat := types.IsFloat(leftType)
		rightIsFloat := types.IsFloat(rightType)
		leftIsInt := types.IsInt(leftType)
		rightIsInt := types.IsInt(rightType)
		
		if leftIsFloat && rightIsFloat && !leftIsInt && !rightIsInt {
			var pred enum.FPred
			switch op {
			case "==":
				pred = enum.FPredOEQ
			case "!=":
				pred = enum.FPredONE
			case "<":
				pred = enum.FPredOLT
			case ">":
				pred = enum.FPredOGT
			case "<=":
				pred = enum.FPredOLE
			case ">=":
				pred = enum.FPredOGE
			default:
				return nil, fmt.Errorf("unsupported float comparison: %s", op)
			}
			result = m.currentBlock.NewFCmp(pred, llvmLeft, llvmRight)
		} else {
			// 如果有一个操作数不是浮点类型，使用 ICmp（整数比较）
			// 注意：如果类型不匹配，可能需要类型转换，但这里先使用 ICmp
			var pred enum.IPred
			switch op {
			case "==":
				pred = enum.IPredEQ
			case "!=":
				pred = enum.IPredNE
			case "<":
				pred = enum.IPredSLT
			case ">":
				pred = enum.IPredSGT
			case "<=":
				pred = enum.IPredSLE
			case ">=":
				pred = enum.IPredSGE
			default:
				return nil, fmt.Errorf("unsupported integer comparison: %s", op)
			}
			result = m.currentBlock.NewICmp(pred, llvmLeft, llvmRight)
		}
	case "shl":
		// 左移操作：shl i64 %left, i32 %right
		// 注意：LLVM的shl操作要求两个操作数类型相同，但shift amount通常是i32
		// 如果right是i32，需要转换为与left相同的类型
		leftType := llvmLeft.Type()
		rightType := llvmRight.Type()
		if leftType != rightType {
			// 将right转换为left的类型（通常是i64）
			rightConverted := m.currentBlock.NewZExt(llvmRight, leftType)
			result = m.currentBlock.NewShl(llvmLeft, rightConverted)
		} else {
			result = m.currentBlock.NewShl(llvmLeft, llvmRight)
		}
	case "xor":
		// 异或操作：xor i64 %left, i64 %right
		result = m.currentBlock.NewXor(llvmLeft, llvmRight)
	case "and":
		// 逻辑与操作：and i1 %left, i1 %right
		// 注意：LLVM的And指令用于位运算，逻辑与使用ICmp + And组合
		// 但这里我们直接使用And，因为操作数已经是i1类型
		result = m.currentBlock.NewAnd(llvmLeft, llvmRight)
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

// CreatePointerIsNull 生成 (ptr == null) 的 i1 条件，用于 map_get 等返回指针的 NULL 检查
func (m *IRModuleManagerImpl) CreatePointerIsNull(ptr interface{}) (interface{}, error) {
	if m.currentBlock == nil {
		return nil, fmt.Errorf("no current basic block set")
	}
	llvmPtr, ok := ptr.(llvalue.Value)
	if !ok {
		return nil, fmt.Errorf("invalid pointer type for CreatePointerIsNull: %T", ptr)
	}
	ptrType, ok := llvmPtr.Type().(*types.PointerType)
	if !ok {
		return nil, fmt.Errorf("CreatePointerIsNull requires pointer type, got %T", llvmPtr.Type())
	}
	nullConst := constant.NewNull(ptrType)
	cond := m.currentBlock.NewICmp(enum.IPredEQ, llvmPtr, nullConst)
	return cond, nil
}

// CreateUnreachable 在当前块插入 unreachable 终止符（用于 NULL 等错误路径）
func (m *IRModuleManagerImpl) CreateUnreachable() error {
	if m.currentBlock == nil {
		return fmt.Errorf("no current basic block set")
	}
	m.currentBlock.NewUnreachable()
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
// 注意：float <-> double 不能使用 BitCast，会走 convertValueType 用 FPExt/FPTrunc。
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

	srcType := llvmValue.Type()
	// float/double 之间必须用 FPExt/FPTrunc，不能用 BitCast
	if _, srcIsFloat := srcType.(*types.FloatType); srcIsFloat {
		if _, dstIsFloat := llvmTargetType.(*types.FloatType); dstIsFloat && !types.Equal(srcType, llvmTargetType) {
			converted, err := m.convertValueType(llvmValue, srcType, llvmTargetType)
			if err != nil {
				return nil, err
			}
			if name != "" {
				uniqueName := m.generateUniqueVariableName(name)
				if inst, ok := converted.(*ir.InstFPExt); ok {
					inst.SetName(uniqueName)
				} else if inst, ok := converted.(*ir.InstFPTrunc); ok {
					inst.SetName(uniqueName)
				}
			}
			return converted, nil
		}
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
// 若尚未存在 main，则创建空的 main 与 entry 块，并设为当前函数/块，
// 使顶层 VarDecl/Print/If 等生成的 alloca 与指令落入 main，避免 "use of undefined value"。
func (m *IRModuleManagerImpl) InitializeMainFunction() error {
	// 声明外部函数（打印函数和协程运行时函数）
	m.declareExternalFunctions()

	// 若没有 main，先创建 main 与空 entry，并设为当前函数/块，供顶层代码生成使用
	if m.findFunction("main") == nil {
		mainFunc := m.module.NewFunc("main", types.I32)
		if m.findFunction("main") == nil {
			m.module.Funcs = append(m.module.Funcs, mainFunc)
		} else {
			mainFunc = m.findFunction("main")
		}
		entryBlock := mainFunc.NewBlock("entry")
		// 不在此处添加 ret，由 Finalize 统一补全
		m.currentFunc = mainFunc
		m.currentBlock = entryBlock
		// 为 main 初始化变量名计数器
		m.functionVariableCounters["main"] = make(map[string]int)
		m.variableCounters = make(map[string]int)
	}

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
					// 注意：这里不应该添加第二个 terminator
					// 上面的 if-else 已经处理了所有情况，不应该再添加 unreachable
					// 移除这行，避免重复添加 terminator
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

	// 确保main函数所有以 Ret 结尾的出口块在 ret 前调用 run_scheduler。
	// 仅处理 Term 为 Ret 的块；以 Br/CondBr 结尾的块（如 else-if 的内层 merge）不插入，
	// 避免在非出口块后追加指令导致非法 IR 或段错误。
	if mainFunc != nil && len(mainFunc.Blocks) > 0 {
		runSchedulerFunc, hasRunScheduler := m.externalFunctions["run_scheduler"]
		for _, block := range mainFunc.Blocks {
			if block.Term == nil {
				// 无 terminator：补 run_scheduler + ret
				if hasRunScheduler {
					block.NewCall(runSchedulerFunc)
				}
				if mainFunc.Sig.RetType == types.I32 {
					block.NewRet(constant.NewInt(types.I32, 0))
				} else if mainFunc.Sig.RetType == types.Void {
					block.NewRet(nil)
				}
				continue
			}
			// 仅对以 Ret 结尾的块在 ret 前插入 run_scheduler；Br/CondBr 结尾的块跳过
			termType := fmt.Sprintf("%T", block.Term)
			if !strings.Contains(termType, "Ret") {
				continue
			}
			// 在 ret 前插入 run_scheduler：移除原 ret，插入 call，再添加 ret
			block.Term = nil
			if hasRunScheduler {
				block.NewCall(runSchedulerFunc)
			}
			if mainFunc.Sig.RetType == types.I32 {
				block.NewRet(constant.NewInt(types.I32, 0))
			} else if mainFunc.Sig.RetType == types.Void {
				block.NewRet(nil)
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
	printFloatFunc := m.module.NewFunc("print_float", types.Void, ir.NewParam("value", types.Double)) // f64，与 EvaluateFloatLiteral 一致
	printBoolFunc := m.module.NewFunc("print_bool", types.Void, ir.NewParam("value", types.I1))

	// 将函数存储在缓存中，便于后续查找
	m.externalFunctions["print_int"] = printIntFunc
	m.externalFunctions["print_string"] = printStringFunc
	m.externalFunctions["print_float"] = printFloatFunc
	m.externalFunctions["print_bool"] = printBoolFunc

	// 声明 P0 优先级的运行时函数（核心功能）
	m.declareP0RuntimeFunctions()

	// 声明 P1 优先级的运行时函数（重要功能）
	m.declareP1RuntimeFunctions()

	// 声明 P2 优先级的运行时函数（增强功能）
	m.declareP2RuntimeFunctions()
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

// declareP0RuntimeFunctions 声明 P0 优先级的运行时函数（核心功能）
// P0 函数是标准库核心功能必需的，必须优先实现
func (m *IRModuleManagerImpl) declareP0RuntimeFunctions() {
	// ==================== 时间操作 ====================
	// runtime_time_now_ms() -> i64
	// 获取当前时间（Unix 时间戳，毫秒）
	timeNowMsFunc := m.module.NewFunc("runtime_time_now_ms", types.I64)
	m.externalFunctions["runtime_time_now_ms"] = timeNowMsFunc

	// runtime_time_format(sec: i64, nsec: i32, layout: string) -> string
	// 格式化时间
	// 编译器签名：i8* runtime_time_format(i64 sec, i32 nsec, i8* layout)
	timeFormatFunc := m.module.NewFunc("runtime_time_format",
		types.NewPointer(types.I8), // 返回字符串指针
		ir.NewParam("sec", types.I64),   // Unix时间戳（秒）
		ir.NewParam("nsec", types.I32),  // 纳秒偏移
		ir.NewParam("layout", types.NewPointer(types.I8)), // 格式化模板字符串
	)
	m.externalFunctions["runtime_time_format"] = timeFormatFunc

	// runtime_time_parse(layout: string, value: string) -> *TimeParseResult
	// 解析时间字符串
	// 编译器签名：i8* runtime_time_parse(i8* layout, i8* value)
	// 注意：返回结构体指针，包含解析结果（success, sec, nsec）
	timeParseFunc := m.module.NewFunc("runtime_time_parse",
		types.NewPointer(types.I8), // 返回结构体指针（TimeParseResult*）
		ir.NewParam("layout", types.NewPointer(types.I8)), // 格式化模板字符串
		ir.NewParam("value", types.NewPointer(types.I8)),  // 时间字符串
	)
	m.externalFunctions["runtime_time_parse"] = timeParseFunc

	// runtime_get_random_seed() -> u64
	// 获取随机种子（使用系统熵源）
	// 注意：u64 在 LLVM IR 中映射为 i64
	randomSeedFunc := m.module.NewFunc("runtime_get_random_seed", types.I64)
	m.externalFunctions["runtime_get_random_seed"] = randomSeedFunc

	// ==================== 字符串操作 ====================
	// runtime_char_ptr_to_string(ptr: char*) -> string_t
	// 将 C 字符串指针转换为 Echo string_t 结构体
	// 编译器签名：i8* runtime_char_ptr_to_string(i8* ptr)
	// 注意：返回 string_t 结构体（值类型），在 LLVM IR 中需要特殊处理
	// string_t 结构体：{ i8*, i64, i64 } (data, length, capacity)
	i8PtrType := types.NewPointer(types.I8)
	i64Type := types.I64
	stringStructType := types.NewStruct(i8PtrType, i64Type, i64Type) // { i8*, i64, i64 }
	charPtrToStringFunc := m.module.NewFunc("runtime_char_ptr_to_string",
		stringStructType, // 返回 string_t 结构体（值类型）
		ir.NewParam("ptr", types.NewPointer(types.I8)), // char* 类型（i8*）
	)
	m.externalFunctions["runtime_char_ptr_to_string"] = charPtrToStringFunc

	// runtime_string_len_utf8(s: string) -> int
	// 计算字符串长度（UTF-8 编码，返回字符数）
	stringLenUtf8Func := m.module.NewFunc("runtime_string_len_utf8",
		types.I32, // 返回 int（字符数）
		ir.NewParam("s", types.NewPointer(types.I8)), // string 类型在 LLVM IR 中是 i8*
	)
	m.externalFunctions["runtime_string_len_utf8"] = stringLenUtf8Func

	// runtime_string_hash(s: string) -> u64
	// 计算字符串哈希值（FNV-1a 算法）
	stringHashFunc := m.module.NewFunc("runtime_string_hash",
		types.I64, // 返回 u64（哈希值）
		ir.NewParam("s", types.NewPointer(types.I8)), // string 类型在 LLVM IR 中是 i8*
	)
	m.externalFunctions["runtime_string_hash"] = stringHashFunc

	// runtime_float_hash(f: f64) -> u64
	// 计算浮点数哈希值（位表示转换）
	// 注意：u64 在 LLVM IR 中映射为 i64
	floatHashFunc := m.module.NewFunc("runtime_float_hash",
		types.I64, // 返回 u64（哈希值）
		ir.NewParam("f", types.Double), // f64 类型在 LLVM IR 中是 double
	)
	m.externalFunctions["runtime_float_hash"] = floatHashFunc

	// runtime_string_contains(s: string, sub: string) -> bool
	// 检查字符串是否包含子字符串
	stringContainsFunc := m.module.NewFunc("runtime_string_contains",
		types.I1, // 返回 bool
		ir.NewParam("s", types.NewPointer(types.I8)),   // 主字符串
		ir.NewParam("sub", types.NewPointer(types.I8)),  // 子字符串
	)
	m.externalFunctions["runtime_string_contains"] = stringContainsFunc

	// runtime_string_starts_with(s: string, prefix: string) -> bool
	// 检查字符串是否以指定字符串开头
	stringStartsWithFunc := m.module.NewFunc("runtime_string_starts_with",
		types.I1, // 返回 bool
		ir.NewParam("s", types.NewPointer(types.I8)),     // 主字符串
		ir.NewParam("prefix", types.NewPointer(types.I8)), // 前缀字符串
	)
	m.externalFunctions["runtime_string_starts_with"] = stringStartsWithFunc

	// runtime_string_ends_with(s: string, suffix: string) -> bool
	// 检查字符串是否以指定字符串结尾
	stringEndsWithFunc := m.module.NewFunc("runtime_string_ends_with",
		types.I1, // 返回 bool
		ir.NewParam("s", types.NewPointer(types.I8)),     // 主字符串
		ir.NewParam("suffix", types.NewPointer(types.I8)), // 后缀字符串
	)
	m.externalFunctions["runtime_string_ends_with"] = stringEndsWithFunc

	// runtime_string_equals(s1: string, s2: string) -> bool
	// 比较两个字符串是否相等
	stringEqualsFunc := m.module.NewFunc("runtime_string_equals",
		types.I1, // 返回 bool
		ir.NewParam("s1", types.NewPointer(types.I8)), // 第一个字符串
		ir.NewParam("s2", types.NewPointer(types.I8)), // 第二个字符串
	)
	m.externalFunctions["runtime_string_equals"] = stringEqualsFunc

	// runtime_string_trim(s: string) -> string
	// 去除字符串首尾空白字符
	stringTrimFunc := m.module.NewFunc("runtime_string_trim",
		types.NewPointer(types.I8), // 返回新字符串指针
		ir.NewParam("s", types.NewPointer(types.I8)), // 输入字符串
	)
	m.externalFunctions["runtime_string_trim"] = stringTrimFunc

	// runtime_string_to_upper(s: string) -> string
	// 将字符串转换为大写
	stringToUpperFunc := m.module.NewFunc("runtime_string_to_upper",
		types.NewPointer(types.I8), // 返回新字符串指针
		ir.NewParam("s", types.NewPointer(types.I8)), // 输入字符串
	)
	m.externalFunctions["runtime_string_to_upper"] = stringToUpperFunc

	// runtime_string_to_lower(s: string) -> string
	// 将字符串转换为小写
	stringToLowerFunc := m.module.NewFunc("runtime_string_to_lower",
		types.NewPointer(types.I8), // 返回新字符串指针
		ir.NewParam("s", types.NewPointer(types.I8)), // 输入字符串
	)
	m.externalFunctions["runtime_string_to_lower"] = stringToLowerFunc

	// runtime_string_to_bytes(s: string) -> *StringToBytesResult
	// 将字符串转换为UTF-8字节数组
	// 注意：返回结构体指针，包含字节数组指针和长度
	// 编译器签名：i8* runtime_string_to_bytes(i8* s)
	stringToBytesFunc := m.module.NewFunc("runtime_string_to_bytes",
		types.NewPointer(types.I8), // 返回结构体指针（StringToBytesResult*）
		ir.NewParam("s", types.NewPointer(types.I8)), // 输入字符串
	)
	m.externalFunctions["runtime_string_to_bytes"] = stringToBytesFunc

	// runtime_base64_encode_string(s: string) -> string
	// 直接编码字符串为Base64字符串（临时实现，绕过切片类型转换）
	// 编译器签名：i8* runtime_base64_encode_string(i8* s)
	base64EncodeStringFunc := m.module.NewFunc("runtime_base64_encode_string",
		types.NewPointer(types.I8), // 返回Base64编码后的字符串指针
		ir.NewParam("s", types.NewPointer(types.I8)), // 输入字符串
	)
	m.externalFunctions["runtime_base64_encode_string"] = base64EncodeStringFunc

	// runtime_bytes_to_string(bytes: []u8) -> string
	// 将UTF-8字节数组转换为字符串
	// 编译器签名：i8* runtime_bytes_to_string(i8* bytes, i32 len)
	bytesToStringFunc := m.module.NewFunc("runtime_bytes_to_string",
		types.NewPointer(types.I8), // 返回字符串指针
		ir.NewParam("bytes", types.NewPointer(types.I8)), // 字节数组指针
		ir.NewParam("len", types.I32),                    // 字节数组长度
	)
	m.externalFunctions["runtime_bytes_to_string"] = bytesToStringFunc

	// runtime_f64_to_string(f: f64) -> string
	// 将浮点数转换为字符串
	// 编译器签名：i8* runtime_f64_to_string(f64 f)
	f64ToStringFunc := m.module.NewFunc("runtime_f64_to_string",
		types.NewPointer(types.I8), // 返回字符串指针
		ir.NewParam("f", types.Double), // f64 类型在 LLVM IR 中是 double
	)
	m.externalFunctions["runtime_f64_to_string"] = f64ToStringFunc

	// runtime_string_to_f64(s: string) -> f64
	// 将字符串解析为浮点数
	// 编译器签名：f64 runtime_string_to_f64(i8* s)
	stringToF64Func := m.module.NewFunc("runtime_string_to_f64",
		types.Double, // 返回 f64
		ir.NewParam("s", types.NewPointer(types.I8)), // 输入字符串
	)
	m.externalFunctions["runtime_string_to_f64"] = stringToF64Func

	// runtime_string_split(s: string, delimiter: string) -> *StringSplitResult
	// 分割字符串，返回字符串数组结构体指针
	// 注意：返回类型是结构体指针，包含字符串指针数组和数量
	// 编译器需要将结构体转换为 []string 切片
	stringSplitFunc := m.module.NewFunc("runtime_string_split",
		types.NewPointer(types.I8), // 返回结构体指针（StringSplitResult*）
		ir.NewParam("s", types.NewPointer(types.I8)),         // 输入字符串
		ir.NewParam("delimiter", types.NewPointer(types.I8)), // 分隔符
	)
	m.externalFunctions["runtime_string_split"] = stringSplitFunc

	// runtime_char_ptr_array_to_string_slice(ptrs: char**, count: int32_t) -> []string
	// 将 C 字符串指针数组转换为 Echo []string 切片
	// 编译器签名：i8* runtime_char_ptr_array_to_string_slice(i8** ptrs, i32 count)
	// 使用位置：类型转换 char** + int32_t → []string（如从 StringSplitResult 提取字段）
	charPtrArrayToStringSliceFunc := m.module.NewFunc("runtime_char_ptr_array_to_string_slice",
		types.NewPointer(types.I8), // 返回切片指针（[]string 在 LLVM IR 中是 *i8）
		ir.NewParam("ptrs", types.NewPointer(types.NewPointer(types.I8))), // char** (i8**)
		ir.NewParam("count", types.I32), // int32_t
	)
	m.externalFunctions["runtime_char_ptr_array_to_string_slice"] = charPtrArrayToStringSliceFunc

	// ==================== 内存管理 ====================
	// 注意：泛型函数 runtime_realloc_slice[T] 需要特殊处理
	// 暂时先声明一个通用版本，使用 i8* 作为元素类型指针
	// runtime_realloc_slice(ptr: *i8, old_cap: int, new_cap: int, elem_size: int) -> *i8
	// 参数说明：
	//   - ptr: 原始切片数据指针
	//   - old_cap: 当前容量
	//   - new_cap: 新容量
	//   - elem_size: 元素大小（字节）
	// 返回：新的切片数据指针（如果失败返回 nil）
	reallocSliceFunc := m.module.NewFunc("runtime_realloc_slice",
		types.NewPointer(types.I8), // 返回新的切片数据指针
		ir.NewParam("ptr", types.NewPointer(types.I8)),     // 原始切片数据指针
		ir.NewParam("old_cap", types.I32),                  // 当前容量
		ir.NewParam("new_cap", types.I32),                  // 新容量
		ir.NewParam("elem_size", types.I32),                // 元素大小（字节）
	)
	m.externalFunctions["runtime_realloc_slice"] = reallocSliceFunc

	// runtime_alloc_slice(cap: int, elem_size: int) -> *i8
	// 分配新切片内存
	allocSliceFunc := m.module.NewFunc("runtime_alloc_slice",
		types.NewPointer(types.I8), // 返回新分配的切片数据指针
		ir.NewParam("cap", types.I32),       // 初始容量
		ir.NewParam("elem_size", types.I32), // 元素大小（字节）
	)
	m.externalFunctions["runtime_alloc_slice"] = allocSliceFunc

	// runtime_free_slice(ptr: *i8) -> void
	// 释放切片内存
	freeSliceFunc := m.module.NewFunc("runtime_free_slice",
		types.Void,
		ir.NewParam("ptr", types.NewPointer(types.I8)), // 切片数据指针
	)
	m.externalFunctions["runtime_free_slice"] = freeSliceFunc

	// runtime_slice_from_ptr_len(ptr: *i8, len: int) -> *i8
	// 从指针和长度构造切片（辅助函数，主要用于类型转换）
	// 注意：这个函数不分配新内存，只是返回输入指针
	// 长度信息需要在符号表中存储，或通过其他方式传递
	sliceFromPtrLenFunc := m.module.NewFunc("runtime_slice_from_ptr_len",
		types.NewPointer(types.I8), // 返回数据指针
		ir.NewParam("ptr", types.NewPointer(types.I8)), // 数据指针
		ir.NewParam("len", types.I32),                   // 长度
	)
	m.externalFunctions["runtime_slice_from_ptr_len"] = sliceFromPtrLenFunc

	// runtime_slice_len(ptr: *i8) -> int
	// 获取切片长度（从运行时切片结构中提取）
	// 注意：运行时切片结构包含长度信息，此函数从结构中提取长度
	sliceLenFunc := m.module.NewFunc("runtime_slice_len",
		types.I32, // 返回长度（int）
		ir.NewParam("ptr", types.NewPointer(types.I8)), // 切片数据指针（运行时切片指针）
	)
	m.externalFunctions["runtime_slice_len"] = sliceLenFunc
}

// declareP1RuntimeFunctions 声明 P1 优先级的运行时函数（重要功能）
// P1 函数是常用功能必需的，包括文件 I/O 和同步原语
func (m *IRModuleManagerImpl) declareP1RuntimeFunctions() {
	// ==================== 文件 I/O ====================
	// 注意：Result 类型在 LLVM IR 中被映射为 *i8（不透明指针）
	// Result[FileHandle, string] -> *i8
	// Result[int, string] -> *i8
	// Result[void, string] -> *i8

	// runtime_open_file(path: string) -> Result[FileHandle, string]
	// 打开文件，返回文件句柄或错误
	openFileFunc := m.module.NewFunc("runtime_open_file",
		types.NewPointer(types.I8), // Result[FileHandle, string] -> *i8
		ir.NewParam("path", types.NewPointer(types.I8)), // string -> *i8
	)
	m.externalFunctions["runtime_open_file"] = openFileFunc

	// runtime_create_file(path: string) -> Result[FileHandle, string]
	// 创建文件，返回文件句柄或错误
	createFileFunc := m.module.NewFunc("runtime_create_file",
		types.NewPointer(types.I8), // Result[FileHandle, string] -> *i8
		ir.NewParam("path", types.NewPointer(types.I8)), // string -> *i8
	)
	m.externalFunctions["runtime_create_file"] = createFileFunc

	// runtime_close_file(handle: FileHandle) -> Result[void, string]
	// 关闭文件
	closeFileFunc := m.module.NewFunc("runtime_close_file",
		types.NewPointer(types.I8), // Result[void, string] -> *i8
		ir.NewParam("handle", types.I32), // FileHandle -> i32
	)
	m.externalFunctions["runtime_close_file"] = closeFileFunc

	// runtime_read_file(handle: FileHandle, buf: []u8) -> Result[int, string]
	// 读取文件，返回读取的字节数或错误
	// 注意：[]u8 在 LLVM IR 中需要传递指针和长度
	readFileFunc := m.module.NewFunc("runtime_read_file",
		types.NewPointer(types.I8), // Result[int, string] -> *i8
		ir.NewParam("handle", types.I32),              // FileHandle -> i32
		ir.NewParam("buf_ptr", types.NewPointer(types.I8)), // []u8 数据指针
		ir.NewParam("buf_len", types.I32),              // []u8 长度
	)
	m.externalFunctions["runtime_read_file"] = readFileFunc

	// runtime_write_file(handle: FileHandle, buf: []u8) -> Result[int, string]
	// 写入文件，返回写入的字节数或错误
	writeFileFunc := m.module.NewFunc("runtime_write_file",
		types.NewPointer(types.I8), // Result[int, string] -> *i8
		ir.NewParam("handle", types.I32),              // FileHandle -> i32
		ir.NewParam("buf_ptr", types.NewPointer(types.I8)), // []u8 数据指针
		ir.NewParam("buf_len", types.I32),              // []u8 长度
	)
	m.externalFunctions["runtime_write_file"] = writeFileFunc

	// runtime_flush_file(handle: FileHandle) -> Result[void, string]
	// 刷新文件缓冲区
	flushFileFunc := m.module.NewFunc("runtime_flush_file",
		types.NewPointer(types.I8), // Result[void, string] -> *i8
		ir.NewParam("handle", types.I32), // FileHandle -> i32
	)
	m.externalFunctions["runtime_flush_file"] = flushFileFunc

	// ==================== 同步原语 ====================

	// runtime_create_mutex() -> MutexHandle
	// 创建互斥锁，返回锁句柄
	createMutexFunc := m.module.NewFunc("runtime_create_mutex",
		types.I32, // MutexHandle -> i32
	)
	m.externalFunctions["runtime_create_mutex"] = createMutexFunc

	// runtime_mutex_lock(handle: MutexHandle) -> void
	// 获取锁（阻塞）
	mutexLockFunc := m.module.NewFunc("runtime_mutex_lock",
		types.Void,
		ir.NewParam("handle", types.I32), // MutexHandle -> i32
	)
	m.externalFunctions["runtime_mutex_lock"] = mutexLockFunc

	// runtime_mutex_try_lock(handle: MutexHandle) -> bool
	// 尝试获取锁（非阻塞）
	mutexTryLockFunc := m.module.NewFunc("runtime_mutex_try_lock",
		types.I1, // bool -> i1
		ir.NewParam("handle", types.I32), // MutexHandle -> i32
	)
	m.externalFunctions["runtime_mutex_try_lock"] = mutexTryLockFunc

	// runtime_mutex_unlock(handle: MutexHandle) -> void
	// 释放锁
	mutexUnlockFunc := m.module.NewFunc("runtime_mutex_unlock",
		types.Void,
		ir.NewParam("handle", types.I32), // MutexHandle -> i32
	)
	m.externalFunctions["runtime_mutex_unlock"] = mutexUnlockFunc

	// runtime_create_rwlock() -> RwLockHandle
	// 创建读写锁，返回锁句柄
	createRwLockFunc := m.module.NewFunc("runtime_create_rwlock",
		types.I32, // RwLockHandle -> i32
	)
	m.externalFunctions["runtime_create_rwlock"] = createRwLockFunc

	// runtime_rwlock_read_lock(handle: RwLockHandle) -> void
	// 获取读锁（阻塞）
	rwlockReadLockFunc := m.module.NewFunc("runtime_rwlock_read_lock",
		types.Void,
		ir.NewParam("handle", types.I32), // RwLockHandle -> i32
	)
	m.externalFunctions["runtime_rwlock_read_lock"] = rwlockReadLockFunc

	// runtime_rwlock_try_read_lock(handle: RwLockHandle) -> bool
	// 尝试获取读锁（非阻塞）
	rwlockTryReadLockFunc := m.module.NewFunc("runtime_rwlock_try_read_lock",
		types.I1, // bool -> i1
		ir.NewParam("handle", types.I32), // RwLockHandle -> i32
	)
	m.externalFunctions["runtime_rwlock_try_read_lock"] = rwlockTryReadLockFunc

	// runtime_rwlock_write_lock(handle: RwLockHandle) -> void
	// 获取写锁（阻塞）
	rwlockWriteLockFunc := m.module.NewFunc("runtime_rwlock_write_lock",
		types.Void,
		ir.NewParam("handle", types.I32), // RwLockHandle -> i32
	)
	m.externalFunctions["runtime_rwlock_write_lock"] = rwlockWriteLockFunc

	// runtime_rwlock_try_write_lock(handle: RwLockHandle) -> bool
	// 尝试获取写锁（非阻塞）
	rwlockTryWriteLockFunc := m.module.NewFunc("runtime_rwlock_try_write_lock",
		types.I1, // bool -> i1
		ir.NewParam("handle", types.I32), // RwLockHandle -> i32
	)
	m.externalFunctions["runtime_rwlock_try_write_lock"] = rwlockTryWriteLockFunc

	// runtime_rwlock_read_unlock(handle: RwLockHandle) -> void
	// 释放读锁
	rwlockReadUnlockFunc := m.module.NewFunc("runtime_rwlock_read_unlock",
		types.Void,
		ir.NewParam("handle", types.I32), // RwLockHandle -> i32
	)
	m.externalFunctions["runtime_rwlock_read_unlock"] = rwlockReadUnlockFunc

	// runtime_rwlock_write_unlock(handle: RwLockHandle) -> void
	// 释放写锁
	rwlockWriteUnlockFunc := m.module.NewFunc("runtime_rwlock_write_unlock",
		types.Void,
		ir.NewParam("handle", types.I32), // RwLockHandle -> i32
	)
	m.externalFunctions["runtime_rwlock_write_unlock"] = rwlockWriteUnlockFunc

	// ==================== 原子操作 ====================

	// i32 原子操作
	atomicLoadI32Func := m.module.NewFunc("runtime_atomic_load_i32",
		types.I32, // 返回 i32
		ir.NewParam("ptr", types.NewPointer(types.I32)), // *i32
	)
	m.externalFunctions["runtime_atomic_load_i32"] = atomicLoadI32Func

	atomicStoreI32Func := m.module.NewFunc("runtime_atomic_store_i32",
		types.Void,
		ir.NewParam("ptr", types.NewPointer(types.I32)), // *i32
		ir.NewParam("value", types.I32),                 // i32
	)
	m.externalFunctions["runtime_atomic_store_i32"] = atomicStoreI32Func

	atomicSwapI32Func := m.module.NewFunc("runtime_atomic_swap_i32",
		types.I32, // 返回旧值
		ir.NewParam("ptr", types.NewPointer(types.I32)), // *i32
		ir.NewParam("value", types.I32),                 // i32
	)
	m.externalFunctions["runtime_atomic_swap_i32"] = atomicSwapI32Func

	atomicCasI32Func := m.module.NewFunc("runtime_atomic_cas_i32",
		types.I1, // 返回 bool（是否成功）
		ir.NewParam("ptr", types.NewPointer(types.I32)), // *i32
		ir.NewParam("expected", types.I32),              // i32
		ir.NewParam("desired", types.I32),               // i32
	)
	m.externalFunctions["runtime_atomic_cas_i32"] = atomicCasI32Func

	atomicAddI32Func := m.module.NewFunc("runtime_atomic_add_i32",
		types.I32, // 返回旧值
		ir.NewParam("ptr", types.NewPointer(types.I32)), // *i32
		ir.NewParam("delta", types.I32),                 // i32
	)
	m.externalFunctions["runtime_atomic_add_i32"] = atomicAddI32Func

	// i64 原子操作
	atomicLoadI64Func := m.module.NewFunc("runtime_atomic_load_i64",
		types.I64, // 返回 i64
		ir.NewParam("ptr", types.NewPointer(types.I64)), // *i64
	)
	m.externalFunctions["runtime_atomic_load_i64"] = atomicLoadI64Func

	atomicStoreI64Func := m.module.NewFunc("runtime_atomic_store_i64",
		types.Void,
		ir.NewParam("ptr", types.NewPointer(types.I64)), // *i64
		ir.NewParam("value", types.I64),                 // i64
	)
	m.externalFunctions["runtime_atomic_store_i64"] = atomicStoreI64Func

	atomicSwapI64Func := m.module.NewFunc("runtime_atomic_swap_i64",
		types.I64, // 返回旧值
		ir.NewParam("ptr", types.NewPointer(types.I64)), // *i64
		ir.NewParam("value", types.I64),                 // i64
	)
	m.externalFunctions["runtime_atomic_swap_i64"] = atomicSwapI64Func

	atomicCasI64Func := m.module.NewFunc("runtime_atomic_cas_i64",
		types.I1, // 返回 bool（是否成功）
		ir.NewParam("ptr", types.NewPointer(types.I64)), // *i64
		ir.NewParam("expected", types.I64),              // i64
		ir.NewParam("desired", types.I64),               // i64
	)
	m.externalFunctions["runtime_atomic_cas_i64"] = atomicCasI64Func

	atomicAddI64Func := m.module.NewFunc("runtime_atomic_add_i64",
		types.I64, // 返回旧值
		ir.NewParam("ptr", types.NewPointer(types.I64)), // *i64
		ir.NewParam("delta", types.I64),                 // i64
	)
	m.externalFunctions["runtime_atomic_add_i64"] = atomicAddI64Func

	// ==================== 原子操作（bool）====================
	// 注意：bool 在 LLVM IR 中是 i1，但在 C 运行时中使用 i8（uint8_t）实现
	// 所以函数签名使用 i8* 和 i8，返回值转换为 i1

	// runtime_atomic_load_bool(ptr: *bool) -> bool
	// 原子加载 bool 值
	atomicLoadBoolFunc := m.module.NewFunc("runtime_atomic_load_bool",
		types.I1, // 返回 bool（i1）
		ir.NewParam("ptr", types.NewPointer(types.I8)), // *bool 在 C 中是 *i8
	)
	m.externalFunctions["runtime_atomic_load_bool"] = atomicLoadBoolFunc

	// runtime_atomic_store_bool(ptr: *bool, value: bool) -> void
	// 原子存储 bool 值
	atomicStoreBoolFunc := m.module.NewFunc("runtime_atomic_store_bool",
		types.Void,
		ir.NewParam("ptr", types.NewPointer(types.I8)), // *bool 在 C 中是 *i8
		ir.NewParam("value", types.I8),                 // bool 在 C 中是 i8
	)
	m.externalFunctions["runtime_atomic_store_bool"] = atomicStoreBoolFunc

	// runtime_atomic_swap_bool(ptr: *bool, value: bool) -> bool
	// 原子交换 bool 值
	atomicSwapBoolFunc := m.module.NewFunc("runtime_atomic_swap_bool",
		types.I1, // 返回 bool（i1）
		ir.NewParam("ptr", types.NewPointer(types.I8)), // *bool 在 C 中是 *i8
		ir.NewParam("value", types.I8),                 // bool 在 C 中是 i8
	)
	m.externalFunctions["runtime_atomic_swap_bool"] = atomicSwapBoolFunc

	// runtime_atomic_cas_bool(ptr: *bool, expected: bool, desired: bool) -> bool
	// 原子比较并交换 bool 值
	atomicCasBoolFunc := m.module.NewFunc("runtime_atomic_cas_bool",
		types.I1, // 返回 bool（i1，true 表示成功）
		ir.NewParam("ptr", types.NewPointer(types.I8)),     // *bool 在 C 中是 *i8
		ir.NewParam("expected", types.I8),                 // bool 在 C 中是 i8
		ir.NewParam("desired", types.I8),                   // bool 在 C 中是 i8
	)
	m.externalFunctions["runtime_atomic_cas_bool"] = atomicCasBoolFunc

	// ==================== 异步运行时 ====================
	// 注意：Task 和 Channel 相关的函数已经在 declareCoroutineRuntimeFunctions 中声明
	// 这里只添加标准库需要的额外函数

	// runtime_spawn_task(f: func() -> T) -> TaskId
	// 创建任务，返回任务 ID
	// 注意：函数指针在 LLVM IR 中需要特殊处理，这里简化实现
	spawnTaskFunc := m.module.NewFunc("runtime_spawn_task",
		types.I32, // TaskId -> i32
		ir.NewParam("func_ptr", types.NewPointer(types.I8)), // 函数指针（泛型处理）
		ir.NewParam("arg_ptr", types.NewPointer(types.I8)),  // 参数指针
	)
	m.externalFunctions["runtime_spawn_task"] = spawnTaskFunc

	// runtime_task_status(task_id: TaskId) -> string
	// 检查任务状态，返回状态字符串
	taskStatusFunc := m.module.NewFunc("runtime_task_status",
		types.NewPointer(types.I8), // string -> *i8
		ir.NewParam("task_id", types.I32), // TaskId -> i32
	)
	m.externalFunctions["runtime_task_status"] = taskStatusFunc

	// runtime_cancel_task(task_id: TaskId) -> Result[void, string]
	// 取消任务
	cancelTaskFunc := m.module.NewFunc("runtime_cancel_task",
		types.NewPointer(types.I8), // Result[void, string] -> *i8
		ir.NewParam("task_id", types.I32), // TaskId -> i32
	)
	m.externalFunctions["runtime_cancel_task"] = cancelTaskFunc

	// runtime_create_channel_unbuffered() -> ChannelHandle
	// 创建无缓冲通道
	createChannelUnbufferedFunc := m.module.NewFunc("runtime_create_channel_unbuffered",
		types.I32, // ChannelHandle -> i32
	)
	m.externalFunctions["runtime_create_channel_unbuffered"] = createChannelUnbufferedFunc

	// runtime_create_channel_buffered(cap: int) -> ChannelHandle
	// 创建有缓冲通道
	createChannelBufferedFunc := m.module.NewFunc("runtime_create_channel_buffered",
		types.I32, // ChannelHandle -> i32
		ir.NewParam("cap", types.I32), // int -> i32
	)
	m.externalFunctions["runtime_create_channel_buffered"] = createChannelBufferedFunc

	// runtime_channel_send(handle: ChannelHandle, item_ptr: *i8, item_size: int) -> Result[void, string]
	// 发送数据（阻塞）
	channelSendFunc := m.module.NewFunc("runtime_channel_send",
		types.NewPointer(types.I8), // Result[void, string] -> *i8
		ir.NewParam("handle", types.I32),              // ChannelHandle -> i32
		ir.NewParam("item_ptr", types.NewPointer(types.I8)), // 数据指针
		ir.NewParam("item_size", types.I32),            // 数据大小（字节）
	)
	m.externalFunctions["runtime_channel_send"] = channelSendFunc

	// runtime_channel_recv(handle: ChannelHandle, item_ptr: *i8, item_size: int) -> Result[int, string]
	// 接收数据（阻塞），返回接收到的数据大小或错误
	channelRecvFunc := m.module.NewFunc("runtime_channel_recv",
		types.NewPointer(types.I8), // Result[int, string] -> *i8
		ir.NewParam("handle", types.I32),              // ChannelHandle -> i32
		ir.NewParam("item_ptr", types.NewPointer(types.I8)), // 数据指针（输出）
		ir.NewParam("item_size", types.I32),            // 数据大小（字节）
	)
	m.externalFunctions["runtime_channel_recv"] = channelRecvFunc

	// runtime_channel_try_send(handle: ChannelHandle, item_ptr: *i8, item_size: int) -> Result[void, string]
	// 尝试发送数据（非阻塞）
	channelTrySendFunc := m.module.NewFunc("runtime_channel_try_send",
		types.NewPointer(types.I8), // Result[void, string] -> *i8
		ir.NewParam("handle", types.I32),              // ChannelHandle -> i32
		ir.NewParam("item_ptr", types.NewPointer(types.I8)), // 数据指针
		ir.NewParam("item_size", types.I32),            // 数据大小（字节）
	)
	m.externalFunctions["runtime_channel_try_send"] = channelTrySendFunc

	// runtime_channel_try_recv(handle: ChannelHandle, item_ptr: *i8, item_size: int) -> Result[int, string]
	// 尝试接收数据（非阻塞）
	channelTryRecvFunc := m.module.NewFunc("runtime_channel_try_recv",
		types.NewPointer(types.I8), // Result[int, string] -> *i8
		ir.NewParam("handle", types.I32),              // ChannelHandle -> i32
		ir.NewParam("item_ptr", types.NewPointer(types.I8)), // 数据指针（输出）
		ir.NewParam("item_size", types.I32),            // 数据大小（字节）
	)
	m.externalFunctions["runtime_channel_try_recv"] = channelTryRecvFunc

	// runtime_channel_close(handle: ChannelHandle) -> void
	// 关闭通道
	channelCloseFunc := m.module.NewFunc("runtime_channel_close",
		types.Void,
		ir.NewParam("handle", types.I32), // ChannelHandle -> i32
	)
	m.externalFunctions["runtime_channel_close"] = channelCloseFunc

	// runtime_channel_status(handle: ChannelHandle) -> string
	// 检查通道状态，返回状态字符串
	channelStatusFunc := m.module.NewFunc("runtime_channel_status",
		types.NewPointer(types.I8), // string -> *i8
		ir.NewParam("handle", types.I32), // ChannelHandle -> i32
	)
	m.externalFunctions["runtime_channel_status"] = channelStatusFunc
}

// declareP2RuntimeFunctions 声明 P2 优先级的运行时函数（增强功能）
// P2 函数是增强功能，包括网络 I/O 和数学函数
func (m *IRModuleManagerImpl) declareP2RuntimeFunctions() {
	// ==================== Map迭代支持 ====================
	// runtime_map_get_keys(map_ptr: *i8) -> *MapIterResult
	// 获取map的键数组和值数组
	// 返回结构体指针，包含 keys: *i8* (char**), values: *i8* (char**), count: i32
	// 注意：map在LLVM IR中被映射为 i8* 指针（不透明指针）
	mapGetKeysFunc := m.module.NewFunc("runtime_map_get_keys",
		types.NewPointer(types.I8), // 返回 MapIterResult* 结构体指针
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
	)
	m.externalFunctions["runtime_map_get_keys"] = mapGetKeysFunc

	// runtime_map_set(map_ptr: *i8, key: *i8, value: *i8) -> void
	// 设置map中的键值对
	// 注意：map在LLVM IR中被映射为 i8* 指针（不透明指针）
	// key和value都是char*（string类型）
	mapSetFunc := m.module.NewFunc("runtime_map_set",
		types.Void, // 返回void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.NewPointer(types.I8)),     // key（char*）
		ir.NewParam("value", types.NewPointer(types.I8)),   // value（char*）
	)
	m.externalFunctions["runtime_map_set"] = mapSetFunc

	// runtime_map_set_string_string(map_ptr: *i8, key: *i8, value: *i8) -> void
	// map[string]string 赋值，与 runtime_map_set 同义
	mapSetStringStringFunc := m.module.NewFunc("runtime_map_set_string_string",
		types.Void,
		ir.NewParam("map_ptr", types.NewPointer(types.I8)),
		ir.NewParam("key", types.NewPointer(types.I8)),
		ir.NewParam("value", types.NewPointer(types.I8)),
	)
	m.externalFunctions["runtime_map_set_string_string"] = mapSetStringStringFunc

	// runtime_map_get_string_string(map_ptr: *i8, key: *i8) -> *i8
	// map[string]string 索引访问
	mapGetStringStringFunc := m.module.NewFunc("runtime_map_get_string_string",
		types.NewPointer(types.I8),
		ir.NewParam("map_ptr", types.NewPointer(types.I8)),
		ir.NewParam("key", types.NewPointer(types.I8)),
	)
	m.externalFunctions["runtime_map_get_string_string"] = mapGetStringStringFunc

	// runtime_map_delete(map_ptr: *i8, key: *i8) -> void
	// 删除map中的键值对
	// 注意：map在LLVM IR中被映射为 i8* 指针（不透明指针）
	// key是char*（string类型）
	mapDeleteFunc := m.module.NewFunc("runtime_map_delete",
		types.Void, // 返回void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.NewPointer(types.I8)),     // key（char*）
	)
	m.externalFunctions["runtime_map_delete"] = mapDeleteFunc

	// ==================== Map多类型支持：map[int]string ====================
	// runtime_int32_hash(key: i32) -> i64
	// 计算32位整数哈希值
	int32HashFunc := m.module.NewFunc("runtime_int32_hash",
		types.I64, // 返回i64
		ir.NewParam("key", types.I32), // key（i32）
	)
	m.externalFunctions["runtime_int32_hash"] = int32HashFunc

	// runtime_map_set_int_string(map_ptr: *i8, key: i32, value: *i8) -> void
	// 设置map[int]string中的键值对
	mapSetIntStringFunc := m.module.NewFunc("runtime_map_set_int_string",
		types.Void, // 返回void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.I32),                       // key（i32）
		ir.NewParam("value", types.NewPointer(types.I8)),   // value（char*）
	)
	m.externalFunctions["runtime_map_set_int_string"] = mapSetIntStringFunc

	// runtime_map_get_int_string(map_ptr: *i8, key: i32) -> *i8
	// 获取map[int]string中的值
	mapGetIntStringFunc := m.module.NewFunc("runtime_map_get_int_string",
		types.NewPointer(types.I8), // 返回char*（字符串指针）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.I32),                       // key（i32）
	)
	m.externalFunctions["runtime_map_get_int_string"] = mapGetIntStringFunc

	// runtime_map_delete_int_string(map_ptr: *i8, key: i32) -> void
	// 删除map[int]string中的键值对
	mapDeleteIntStringFunc := m.module.NewFunc("runtime_map_delete_int_string",
		types.Void, // 返回void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.I32),                       // key（i32）
	)
	m.externalFunctions["runtime_map_delete_int_string"] = mapDeleteIntStringFunc

	// ==================== Map多类型支持：map[string]int ====================

	// runtime_map_set_string_int(map_ptr: *i8, key: *i8, value: i32) -> void
	// 设置map[string]int中的键值对
	mapSetStringIntFunc := m.module.NewFunc("runtime_map_set_string_int",
		types.Void, // 返回void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.NewPointer(types.I8)),     // key（char*）
		ir.NewParam("value", types.I32),                     // value（i32）
	)
	m.externalFunctions["runtime_map_set_string_int"] = mapSetStringIntFunc

	// runtime_map_get_string_int(map_ptr: *i8, key: *i8) -> *i32
	// 获取map[string]int中的值
	mapGetStringIntFunc := m.module.NewFunc("runtime_map_get_string_int",
		types.NewPointer(types.I32), // 返回i32*（整数指针）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.NewPointer(types.I8)),     // key（char*）
	)
	m.externalFunctions["runtime_map_get_string_int"] = mapGetStringIntFunc

	// runtime_map_delete_string_int(map_ptr: *i8, key: *i8) -> void
	// 删除map[string]int中的键值对
	mapDeleteStringIntFunc := m.module.NewFunc("runtime_map_delete_string_int",
		types.Void, // 返回void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.NewPointer(types.I8)),     // key（char*）
	)
	m.externalFunctions["runtime_map_delete_string_int"] = mapDeleteStringIntFunc

	// ==================== Map多类型支持：map[int]int ====================

	// runtime_map_set_int_int(map_ptr: *i8, key: i32, value: i32) -> void
	// 设置map[int]int中的键值对
	mapSetIntIntFunc := m.module.NewFunc("runtime_map_set_int_int",
		types.Void, // 返回void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.I32),                       // key（i32）
		ir.NewParam("value", types.I32),                     // value（i32）
	)
	m.externalFunctions["runtime_map_set_int_int"] = mapSetIntIntFunc

	// runtime_map_get_int_int(map_ptr: *i8, key: i32) -> *i32
	// 获取map[int]int中的值
	mapGetIntIntFunc := m.module.NewFunc("runtime_map_get_int_int",
		types.NewPointer(types.I32), // 返回i32*（整数指针）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.I32),                       // key（i32）
	)
	m.externalFunctions["runtime_map_get_int_int"] = mapGetIntIntFunc

	// runtime_map_delete_int_int(map_ptr: *i8, key: i32) -> void
	// 删除map[int]int中的键值对
	mapDeleteIntIntFunc := m.module.NewFunc("runtime_map_delete_int_int",
		types.Void, // 返回void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.I32),                       // key（i32）
	)
	m.externalFunctions["runtime_map_delete_int_int"] = mapDeleteIntIntFunc

	// ==================== Map多类型支持：map[float]string ====================

	// runtime_map_set_float_string(map_ptr: *i8, key: f64, value: *i8) -> void
	// 设置map[float]string中的键值对
	mapSetFloatStringFunc := m.module.NewFunc("runtime_map_set_float_string",
		types.Void, // 返回void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.Double),                    // key（f64）
		ir.NewParam("value", types.NewPointer(types.I8)),   // value（char*）
	)
	m.externalFunctions["runtime_map_set_float_string"] = mapSetFloatStringFunc

	// runtime_map_get_float_string(map_ptr: *i8, key: f64) -> *i8
	// 获取map[float]string中的值
	mapGetFloatStringFunc := m.module.NewFunc("runtime_map_get_float_string",
		types.NewPointer(types.I8), // 返回i8*（字符串指针）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.Double),                    // key（f64）
	)
	m.externalFunctions["runtime_map_get_float_string"] = mapGetFloatStringFunc

	// runtime_map_delete_float_string(map_ptr: *i8, key: f64) -> void
	// 删除map[float]string中的键值对
	mapDeleteFloatStringFunc := m.module.NewFunc("runtime_map_delete_float_string",
		types.Void, // 返回void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.Double),                    // key（f64）
	)
	m.externalFunctions["runtime_map_delete_float_string"] = mapDeleteFloatStringFunc

	// ==================== Map多类型支持：map[string]float ====================

	// runtime_map_set_string_float(map_ptr: *i8, key: *i8, value: f64) -> void
	// 设置map[string]float中的键值对
	mapSetStringFloatFunc := m.module.NewFunc("runtime_map_set_string_float",
		types.Void, // 返回void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.NewPointer(types.I8)),     // key（char*）
		ir.NewParam("value", types.Double),                  // value（f64）
	)
	m.externalFunctions["runtime_map_set_string_float"] = mapSetStringFloatFunc

	// runtime_map_get_string_float(map_ptr: *i8, key: *i8) -> *f64
	// 获取map[string]float中的值
	mapGetStringFloatFunc := m.module.NewFunc("runtime_map_get_string_float",
		types.NewPointer(types.Double), // 返回f64*（浮点数指针）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.NewPointer(types.I8)),     // key（char*）
	)
	m.externalFunctions["runtime_map_get_string_float"] = mapGetStringFloatFunc

	// runtime_map_delete_string_float(map_ptr: *i8, key: *i8) -> void
	// 删除map[string]float中的键值对
	mapDeleteStringFloatFunc := m.module.NewFunc("runtime_map_delete_string_float",
		types.Void, // 返回void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.NewPointer(types.I8)),     // key（char*）
	)
	m.externalFunctions["runtime_map_delete_string_float"] = mapDeleteStringFloatFunc

	// ==================== Map多类型支持：map[bool]string ====================

	// runtime_map_set_bool_string(map_ptr: *i8, key: i1, value: *i8) -> void
	// 设置map[bool]string中的键值对
	mapSetBoolStringFunc := m.module.NewFunc("runtime_map_set_bool_string",
		types.Void, // 返回void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.I1),                       // key（i1，布尔值）
		ir.NewParam("value", types.NewPointer(types.I8)),   // value（char*）
	)
	m.externalFunctions["runtime_map_set_bool_string"] = mapSetBoolStringFunc

	// runtime_map_get_bool_string(map_ptr: *i8, key: i1) -> *i8
	// 获取map[bool]string中的值
	mapGetBoolStringFunc := m.module.NewFunc("runtime_map_get_bool_string",
		types.NewPointer(types.I8), // 返回i8*（字符串指针）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.I1),                       // key（i1，布尔值）
	)
	m.externalFunctions["runtime_map_get_bool_string"] = mapGetBoolStringFunc

	// runtime_map_delete_bool_string(map_ptr: *i8, key: i1) -> void
	// 删除map[bool]string中的键值对
	mapDeleteBoolStringFunc := m.module.NewFunc("runtime_map_delete_bool_string",
		types.Void, // 返回void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.I1),                       // key（i1，布尔值）
	)
	m.externalFunctions["runtime_map_delete_bool_string"] = mapDeleteBoolStringFunc

	// ==================== Map操作功能：通用函数 ====================

	// runtime_map_len(map_ptr: *i8) -> i32
	// 获取Map的长度（条目数量）
	mapLenFunc := m.module.NewFunc("runtime_map_len",
		types.I32, // 返回i32（条目数量）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
	)
	m.externalFunctions["runtime_map_len"] = mapLenFunc

	// runtime_map_clear(map_ptr: *i8) -> void
	// 清空Map（删除所有条目）
	mapClearFunc := m.module.NewFunc("runtime_map_clear",
		types.Void, // 返回void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
	)
	m.externalFunctions["runtime_map_clear"] = mapClearFunc

	// ==================== Map操作功能：contains函数（类型特定）====================

	// runtime_map_contains_string_string(map_ptr: *i8, key: *i8) -> i32
	// 检查map[string]string中是否包含指定的键
	mapContainsStringStringFunc := m.module.NewFunc("runtime_map_contains_string_string",
		types.I32, // 返回i32（1=存在，0=不存在）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.NewPointer(types.I8)),     // key（char*）
	)
	m.externalFunctions["runtime_map_contains_string_string"] = mapContainsStringStringFunc

	// runtime_map_contains_int_string(map_ptr: *i8, key: i32) -> i32
	// 检查map[int]string中是否包含指定的键
	mapContainsIntStringFunc := m.module.NewFunc("runtime_map_contains_int_string",
		types.I32, // 返回i32（1=存在，0=不存在）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.I32),                     // key（i32）
	)
	m.externalFunctions["runtime_map_contains_int_string"] = mapContainsIntStringFunc

	// runtime_map_contains_string_int(map_ptr: *i8, key: *i8) -> i32
	// 检查map[string]int中是否包含指定的键
	mapContainsStringIntFunc := m.module.NewFunc("runtime_map_contains_string_int",
		types.I32, // 返回i32（1=存在，0=不存在）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.NewPointer(types.I8)),     // key（char*）
	)
	m.externalFunctions["runtime_map_contains_string_int"] = mapContainsStringIntFunc

	// runtime_map_contains_int_int(map_ptr: *i8, key: i32) -> i32
	// 检查map[int]int中是否包含指定的键
	mapContainsIntIntFunc := m.module.NewFunc("runtime_map_contains_int_int",
		types.I32, // 返回i32（1=存在，0=不存在）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.I32),                     // key（i32）
	)
	m.externalFunctions["runtime_map_contains_int_int"] = mapContainsIntIntFunc

	// runtime_map_contains_float_string(map_ptr: *i8, key: f64) -> i32
	// 检查map[float]string中是否包含指定的键
	mapContainsFloatStringFunc := m.module.NewFunc("runtime_map_contains_float_string",
		types.I32, // 返回i32（1=存在，0=不存在）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.Double),                   // key（f64）
	)
	m.externalFunctions["runtime_map_contains_float_string"] = mapContainsFloatStringFunc

	// runtime_map_contains_string_float(map_ptr: *i8, key: *i8) -> i32
	// 检查map[string]float中是否包含指定的键
	mapContainsStringFloatFunc := m.module.NewFunc("runtime_map_contains_string_float",
		types.I32, // 返回i32（1=存在，0=不存在）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.NewPointer(types.I8)),     // key（char*）
	)
	m.externalFunctions["runtime_map_contains_string_float"] = mapContainsStringFloatFunc

	// runtime_map_contains_bool_string(map_ptr: *i8, key: i1) -> i32
	// 检查map[bool]string中是否包含指定的键
	mapContainsBoolStringFunc := m.module.NewFunc("runtime_map_contains_bool_string",
		types.I32, // 返回i32（1=存在，0=不存在）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.I1),                       // key（i1）
	)
	m.externalFunctions["runtime_map_contains_bool_string"] = mapContainsBoolStringFunc

	// ==================== Map操作功能：keys()函数（类型特定）====================

	// runtime_map_keys_string_string(map_ptr: *i8) -> *i8
	// 获取map[string]string的键数组（返回i8**，字符串数组）
	mapKeysStringStringFunc := m.module.NewFunc("runtime_map_keys_string_string",
		types.NewPointer(types.NewPointer(types.I8)), // 返回i8**（字符串数组）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
	)
	m.externalFunctions["runtime_map_keys_string_string"] = mapKeysStringStringFunc

	// runtime_map_keys_int_string(map_ptr: *i8) -> *i32
	// 获取map[int]string的键数组（返回i32*，整数数组）
	mapKeysIntStringFunc := m.module.NewFunc("runtime_map_keys_int_string",
		types.NewPointer(types.I32), // 返回i32*（整数数组）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
	)
	m.externalFunctions["runtime_map_keys_int_string"] = mapKeysIntStringFunc

	// runtime_map_keys_string_int(map_ptr: *i8) -> *i8
	// 获取map[string]int的键数组（返回i8**，字符串数组）
	mapKeysStringIntFunc := m.module.NewFunc("runtime_map_keys_string_int",
		types.NewPointer(types.NewPointer(types.I8)), // 返回i8**（字符串数组）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
	)
	m.externalFunctions["runtime_map_keys_string_int"] = mapKeysStringIntFunc

	// runtime_map_keys_int_int(map_ptr: *i8) -> *i32
	// 获取map[int]int的键数组（返回i32*，整数数组）
	mapKeysIntIntFunc := m.module.NewFunc("runtime_map_keys_int_int",
		types.NewPointer(types.I32), // 返回i32*（整数数组）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
	)
	m.externalFunctions["runtime_map_keys_int_int"] = mapKeysIntIntFunc

	// runtime_map_keys_float_string(map_ptr: *i8) -> *f64
	// 获取map[float]string的键数组（返回f64*，浮点数数组）
	mapKeysFloatStringFunc := m.module.NewFunc("runtime_map_keys_float_string",
		types.NewPointer(types.Double), // 返回f64*（浮点数数组）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
	)
	m.externalFunctions["runtime_map_keys_float_string"] = mapKeysFloatStringFunc

	// runtime_map_keys_string_float(map_ptr: *i8) -> *i8
	// 获取map[string]float的键数组（返回i8**，字符串数组）
	mapKeysStringFloatFunc := m.module.NewFunc("runtime_map_keys_string_float",
		types.NewPointer(types.NewPointer(types.I8)), // 返回i8**（字符串数组）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
	)
	m.externalFunctions["runtime_map_keys_string_float"] = mapKeysStringFloatFunc

	// runtime_map_keys_bool_string(map_ptr: *i8) -> *i1
	// 获取map[bool]string的键数组（返回i1*，布尔数组）
	mapKeysBoolStringFunc := m.module.NewFunc("runtime_map_keys_bool_string",
		types.NewPointer(types.I1), // 返回i1*（布尔数组）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
	)
	m.externalFunctions["runtime_map_keys_bool_string"] = mapKeysBoolStringFunc

	// ==================== Map操作功能：values()函数（类型特定）====================

	// runtime_map_values_string_string(map_ptr: *i8) -> *i8
	// 获取map[string]string的值数组（返回i8**，字符串数组）
	mapValuesStringStringFunc := m.module.NewFunc("runtime_map_values_string_string",
		types.NewPointer(types.NewPointer(types.I8)), // 返回i8**（字符串数组）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
	)
	m.externalFunctions["runtime_map_values_string_string"] = mapValuesStringStringFunc

	// runtime_map_values_int_string(map_ptr: *i8) -> *i8
	// 获取map[int]string的值数组（返回i8**，字符串数组）
	mapValuesIntStringFunc := m.module.NewFunc("runtime_map_values_int_string",
		types.NewPointer(types.NewPointer(types.I8)), // 返回i8**（字符串数组）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
	)
	m.externalFunctions["runtime_map_values_int_string"] = mapValuesIntStringFunc

	// runtime_map_values_string_int(map_ptr: *i8) -> *i32
	// 获取map[string]int的值数组（返回i32*，整数数组）
	mapValuesStringIntFunc := m.module.NewFunc("runtime_map_values_string_int",
		types.NewPointer(types.I32), // 返回i32*（整数数组）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
	)
	m.externalFunctions["runtime_map_values_string_int"] = mapValuesStringIntFunc

	// runtime_map_values_int_int(map_ptr: *i8) -> *i32
	// 获取map[int]int的值数组（返回i32*，整数数组）
	mapValuesIntIntFunc := m.module.NewFunc("runtime_map_values_int_int",
		types.NewPointer(types.I32), // 返回i32*（整数数组）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
	)
	m.externalFunctions["runtime_map_values_int_int"] = mapValuesIntIntFunc

	// runtime_map_values_float_string(map_ptr: *i8) -> *i8
	// 获取map[float]string的值数组（返回i8**，字符串数组）
	mapValuesFloatStringFunc := m.module.NewFunc("runtime_map_values_float_string",
		types.NewPointer(types.NewPointer(types.I8)), // 返回i8**（字符串数组）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
	)
	m.externalFunctions["runtime_map_values_float_string"] = mapValuesFloatStringFunc

	// runtime_map_values_string_float(map_ptr: *i8) -> *f64
	// 获取map[string]float的值数组（返回f64*，浮点数数组）
	mapValuesStringFloatFunc := m.module.NewFunc("runtime_map_values_string_float",
		types.NewPointer(types.Double), // 返回f64*（浮点数数组）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
	)
	m.externalFunctions["runtime_map_values_string_float"] = mapValuesStringFloatFunc

	// runtime_map_values_bool_string(map_ptr: *i8) -> *i8
	// 获取map[bool]string的值数组（返回i8**，字符串数组）
	mapValuesBoolStringFunc := m.module.NewFunc("runtime_map_values_bool_string",
		types.NewPointer(types.NewPointer(types.I8)), // 返回i8**（字符串数组）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
	)
	m.externalFunctions["runtime_map_values_bool_string"] = mapValuesBoolStringFunc

	// ==================== Map操作功能：数组/切片类型支持 ====================

	// runtime_map_set_string_slice_int(map_ptr: *i8, key: *i8, slice_ptr: *i8) -> void
	// 设置map[string][]int中的键值对
	mapSetStringSliceIntFunc := m.module.NewFunc("runtime_map_set_string_slice_int",
		types.Void, // void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.NewPointer(types.I8)),     // key（i8*，字符串）
		ir.NewParam("slice_ptr", types.NewPointer(types.I8)), // value（i8*，切片指针）
	)
	m.externalFunctions["runtime_map_set_string_slice_int"] = mapSetStringSliceIntFunc

	// runtime_map_get_string_slice_int(map_ptr: *i8, key: *i8) -> *i8
	// 获取map[string][]int中的值
	mapGetStringSliceIntFunc := m.module.NewFunc("runtime_map_get_string_slice_int",
		types.NewPointer(types.I8), // 返回i8*（切片指针）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.NewPointer(types.I8)),     // key（i8*，字符串）
	)
	m.externalFunctions["runtime_map_get_string_slice_int"] = mapGetStringSliceIntFunc

	// runtime_map_set_int_slice_string(map_ptr: *i8, key: i32, slice_ptr: *i8) -> void
	// 设置map[int][]string中的键值对
	mapSetIntSliceStringFunc := m.module.NewFunc("runtime_map_set_int_slice_string",
		types.Void, // void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.I32),                      // key（i32，整数）
		ir.NewParam("slice_ptr", types.NewPointer(types.I8)), // value（i8*，切片指针）
	)
	m.externalFunctions["runtime_map_set_int_slice_string"] = mapSetIntSliceStringFunc

	// runtime_map_get_int_slice_string(map_ptr: *i8, key: i32) -> *i8
	// 获取map[int][]string中的值
	mapGetIntSliceStringFunc := m.module.NewFunc("runtime_map_get_int_slice_string",
		types.NewPointer(types.I8), // 返回i8*（切片指针）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.I32),                      // key（i32，整数）
	)
	m.externalFunctions["runtime_map_get_int_slice_string"] = mapGetIntSliceStringFunc

	// ==================== Map操作功能：嵌套Map类型支持 ====================

	// runtime_map_set_string_map_int_string(map_ptr: *i8, key: *i8, inner_map_ptr: *i8) -> void
	// 设置map[string]map[int]string中的键值对
	mapSetStringMapIntStringFunc := m.module.NewFunc("runtime_map_set_string_map_int_string",
		types.Void, // void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.NewPointer(types.I8)),     // key（i8*，字符串）
		ir.NewParam("inner_map_ptr", types.NewPointer(types.I8)), // value（i8*，内层Map指针）
	)
	m.externalFunctions["runtime_map_set_string_map_int_string"] = mapSetStringMapIntStringFunc

	// runtime_map_get_string_map_int_string(map_ptr: *i8, key: *i8) -> *i8
	// 获取map[string]map[int]string中的值
	mapGetStringMapIntStringFunc := m.module.NewFunc("runtime_map_get_string_map_int_string",
		types.NewPointer(types.I8), // 返回i8*（内层Map指针）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.NewPointer(types.I8)),     // key（i8*，字符串）
	)
	m.externalFunctions["runtime_map_get_string_map_int_string"] = mapGetStringMapIntStringFunc

	// runtime_map_set_int_map_string_int(map_ptr: *i8, key: i32, inner_map_ptr: *i8) -> void
	// 设置map[int]map[string]int中的键值对
	mapSetIntMapStringIntFunc := m.module.NewFunc("runtime_map_set_int_map_string_int",
		types.Void, // void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.I32),                      // key（i32，整数）
		ir.NewParam("inner_map_ptr", types.NewPointer(types.I8)), // value（i8*，内层Map指针）
	)
	m.externalFunctions["runtime_map_set_int_map_string_int"] = mapSetIntMapStringIntFunc

	// runtime_map_get_int_map_string_int(map_ptr: *i8, key: i32) -> *i8
	// 获取map[int]map[string]int中的值
	mapGetIntMapStringIntFunc := m.module.NewFunc("runtime_map_get_int_map_string_int",
		types.NewPointer(types.I8), // 返回i8*（内层Map指针）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.I32),                      // key（i32，整数）
	)
	m.externalFunctions["runtime_map_get_int_map_string_int"] = mapGetIntMapStringIntFunc

	// ==================== Map操作功能：结构体作为值支持 ====================

	// runtime_map_set_string_struct(map_ptr: *i8, key: *i8, struct_ptr: *i8) -> void
	// 设置map[string]struct中的键值对（通用结构体值）
	mapSetStringStructFunc := m.module.NewFunc("runtime_map_set_string_struct",
		types.Void, // void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.NewPointer(types.I8)),     // key（i8*，字符串）
		ir.NewParam("struct_ptr", types.NewPointer(types.I8)), // value（i8*，结构体指针）
	)
	m.externalFunctions["runtime_map_set_string_struct"] = mapSetStringStructFunc

	// runtime_map_get_string_struct(map_ptr: *i8, key: *i8) -> *i8
	// 获取map[string]struct中的值（通用结构体值）
	mapGetStringStructFunc := m.module.NewFunc("runtime_map_get_string_struct",
		types.NewPointer(types.I8), // 返回i8*（结构体指针）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.NewPointer(types.I8)),     // key（i8*，字符串）
	)
	m.externalFunctions["runtime_map_get_string_struct"] = mapGetStringStructFunc

	// runtime_map_set_int_struct(map_ptr: *i8, key: i32, struct_ptr: *i8) -> void
	// 设置map[int]struct中的键值对（通用结构体值）
	mapSetIntStructFunc := m.module.NewFunc("runtime_map_set_int_struct",
		types.Void, // void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.I32),                      // key（i32，整数）
		ir.NewParam("struct_ptr", types.NewPointer(types.I8)), // value（i8*，结构体指针）
	)
	m.externalFunctions["runtime_map_set_int_struct"] = mapSetIntStructFunc

	// runtime_map_get_int_struct(map_ptr: *i8, key: i32) -> *i8
	// 获取map[int]struct中的值（通用结构体值）
	mapGetIntStructFunc := m.module.NewFunc("runtime_map_get_int_struct",
		types.NewPointer(types.I8), // 返回i8*（结构体指针）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key", types.I32),                      // key（i32，整数）
	)
	m.externalFunctions["runtime_map_get_int_struct"] = mapGetIntStructFunc

	// ==================== Map操作功能：结构体作为键支持 ====================

	// 函数指针类型定义（用于结构体键的hash和equals函数）
	// i64 (*hash_func)(i8*) - 哈希函数指针
	hashFuncType := types.NewPointer(types.NewFunc(types.I64, types.NewPointer(types.I8)))
	// i1 (*equals_func)(i8*, i8*) - 比较函数指针
	equalsFuncType := types.NewPointer(types.NewFunc(types.I1, types.NewPointer(types.I8), types.NewPointer(types.I8)))

	// runtime_map_set_struct_string(map_ptr: *i8, key_ptr: *i8, value: *i8, hash_func: *func, equals_func: *func) -> void
	// 设置map[struct]string中的键值对（通用结构体键版本）
	mapSetStructStringFunc := m.module.NewFunc("runtime_map_set_struct_string",
		types.Void, // void
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key_ptr", types.NewPointer(types.I8)), // key（i8*，结构体指针）
		ir.NewParam("value", types.NewPointer(types.I8)),   // value（i8*，字符串）
		ir.NewParam("hash_func", hashFuncType),             // 哈希函数指针
		ir.NewParam("equals_func", equalsFuncType),         // 比较函数指针
	)
	m.externalFunctions["runtime_map_set_struct_string"] = mapSetStructStringFunc

	// runtime_map_get_struct_string(map_ptr: *i8, key_ptr: *i8, hash_func: *func, equals_func: *func) -> *i8
	// 获取map[struct]string中的值（通用结构体键版本）
	mapGetStructStringFunc := m.module.NewFunc("runtime_map_get_struct_string",
		types.NewPointer(types.I8), // 返回i8*（字符串指针）
		ir.NewParam("map_ptr", types.NewPointer(types.I8)), // map指针（i8*）
		ir.NewParam("key_ptr", types.NewPointer(types.I8)), // key（i8*，结构体指针）
		ir.NewParam("hash_func", hashFuncType),             // 哈希函数指针
		ir.NewParam("equals_func", equalsFuncType),         // 比较函数指针
	)
	m.externalFunctions["runtime_map_get_struct_string"] = mapGetStructStringFunc
	
	// ==================== 网络 I/O ====================
	// 注意：Result 类型在 LLVM IR 中被映射为 *i8（不透明指针）
	// SocketHandle 映射为 i32

	// runtime_tcp_connect(addr: string) -> Result[SocketHandle, string]
	// 建立 TCP 连接
	tcpConnectFunc := m.module.NewFunc("runtime_tcp_connect",
		types.NewPointer(types.I8), // Result[SocketHandle, string] -> *i8
		ir.NewParam("addr", types.NewPointer(types.I8)), // string -> *i8
	)
	m.externalFunctions["runtime_tcp_connect"] = tcpConnectFunc

	// runtime_tcp_bind(addr: string) -> Result[SocketHandle, string]
	// 绑定地址并监听
	tcpBindFunc := m.module.NewFunc("runtime_tcp_bind",
		types.NewPointer(types.I8), // Result[SocketHandle, string] -> *i8
		ir.NewParam("addr", types.NewPointer(types.I8)), // string -> *i8
	)
	m.externalFunctions["runtime_tcp_bind"] = tcpBindFunc

	// runtime_tcp_accept(listener: SocketHandle) -> Result[SocketHandle, string]
	// 接受连接
	tcpAcceptFunc := m.module.NewFunc("runtime_tcp_accept",
		types.NewPointer(types.I8), // Result[SocketHandle, string] -> *i8
		ir.NewParam("listener", types.I32), // SocketHandle -> i32
	)
	m.externalFunctions["runtime_tcp_accept"] = tcpAcceptFunc

	// runtime_tcp_read(socket: SocketHandle, buf: []u8) -> Result[int, string]
	// 读取 TCP 数据
	tcpReadFunc := m.module.NewFunc("runtime_tcp_read",
		types.NewPointer(types.I8), // Result[int, string] -> *i8
		ir.NewParam("socket", types.I32),              // SocketHandle -> i32
		ir.NewParam("buf_ptr", types.NewPointer(types.I8)), // []u8 数据指针
		ir.NewParam("buf_len", types.I32),              // []u8 长度
	)
	m.externalFunctions["runtime_tcp_read"] = tcpReadFunc

	// runtime_tcp_write(socket: SocketHandle, buf: []u8) -> Result[int, string]
	// 写入 TCP 数据
	tcpWriteFunc := m.module.NewFunc("runtime_tcp_write",
		types.NewPointer(types.I8), // Result[int, string] -> *i8
		ir.NewParam("socket", types.I32),              // SocketHandle -> i32
		ir.NewParam("buf_ptr", types.NewPointer(types.I8)), // []u8 数据指针
		ir.NewParam("buf_len", types.I32),              // []u8 长度
	)
	m.externalFunctions["runtime_tcp_write"] = tcpWriteFunc

	// runtime_tcp_close(socket: SocketHandle) -> Result[void, string]
	// 关闭 TCP 连接
	tcpCloseFunc := m.module.NewFunc("runtime_tcp_close",
		types.NewPointer(types.I8), // Result[void, string] -> *i8
		ir.NewParam("socket", types.I32), // SocketHandle -> i32
	)
	m.externalFunctions["runtime_tcp_close"] = tcpCloseFunc

	// runtime_tcp_flush(socket: SocketHandle) -> Result[void, string]
	// 刷新 TCP 缓冲区（TCP 是流式协议，实际上无需 flush，此函数为 API 一致性提供）
	tcpFlushFunc := m.module.NewFunc("runtime_tcp_flush",
		types.NewPointer(types.I8), // Result[void, string] -> *i8
		ir.NewParam("socket", types.I32), // SocketHandle -> i32
	)
	m.externalFunctions["runtime_tcp_flush"] = tcpFlushFunc

	// runtime_tcp_local_addr(socket: SocketHandle) -> Result[string, string]
	// 获取 TCP 连接的本地地址
	tcpLocalAddrFunc := m.module.NewFunc("runtime_tcp_local_addr",
		types.NewPointer(types.I8), // Result[string, string] -> *i8
		ir.NewParam("socket", types.I32), // SocketHandle -> i32
	)
	m.externalFunctions["runtime_tcp_local_addr"] = tcpLocalAddrFunc

	// runtime_tcp_peer_addr(socket: SocketHandle) -> Result[string, string]
	// 获取 TCP 连接的远程地址（对端地址）
	tcpPeerAddrFunc := m.module.NewFunc("runtime_tcp_peer_addr",
		types.NewPointer(types.I8), // Result[string, string] -> *i8
		ir.NewParam("socket", types.I32), // SocketHandle -> i32
	)
	m.externalFunctions["runtime_tcp_peer_addr"] = tcpPeerAddrFunc

	// ==================== 数学函数 ====================
	// 注意：所有数学函数使用 f64（双精度浮点数），在 LLVM IR 中映射为 double

	// runtime_math_sqrt(x: f64) -> f64
	// 平方根
	mathSqrtFunc := m.module.NewFunc("runtime_math_sqrt",
		types.Double, // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_sqrt"] = mathSqrtFunc

	// runtime_math_cbrt(x: f64) -> f64
	// 立方根
	mathCbrtFunc := m.module.NewFunc("runtime_math_cbrt",
		types.Double, // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_cbrt"] = mathCbrtFunc

	// runtime_math_pow(base: f64, exp: f64) -> f64
	// 幂运算
	mathPowFunc := m.module.NewFunc("runtime_math_pow",
		types.Double, // f64 -> double
		ir.NewParam("base", types.Double), // f64 -> double
		ir.NewParam("exp", types.Double),  // f64 -> double
	)
	m.externalFunctions["runtime_math_pow"] = mathPowFunc

	// runtime_math_exp(x: f64) -> f64
	// 自然指数（e^x）
	mathExpFunc := m.module.NewFunc("runtime_math_exp",
		types.Double, // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_exp"] = mathExpFunc

	// runtime_math_ln(x: f64) -> f64
	// 自然对数（ln(x)）
	mathLnFunc := m.module.NewFunc("runtime_math_ln",
		types.Double, // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_ln"] = mathLnFunc

	// runtime_math_log10(x: f64) -> f64
	// 常用对数（log10(x)）
	mathLog10Func := m.module.NewFunc("runtime_math_log10",
		types.Double, // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_log10"] = mathLog10Func

	// runtime_math_log2(x: f64) -> f64
	// 二进制对数（log2(x)）
	mathLog2Func := m.module.NewFunc("runtime_math_log2",
		types.Double, // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_log2"] = mathLog2Func

	// runtime_math_sin(x: f64) -> f64
	// 正弦函数
	mathSinFunc := m.module.NewFunc("runtime_math_sin",
		types.Double, // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_sin"] = mathSinFunc

	// runtime_math_cos(x: f64) -> f64
	// 余弦函数
	mathCosFunc := m.module.NewFunc("runtime_math_cos",
		types.Double, // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_cos"] = mathCosFunc

	// runtime_math_tan(x: f64) -> f64
	// 正切函数
	mathTanFunc := m.module.NewFunc("runtime_math_tan",
		types.Double, // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_tan"] = mathTanFunc

	// runtime_math_asin(x: f64) -> f64
	// 反正弦函数
	mathAsinFunc := m.module.NewFunc("runtime_math_asin",
		types.Double, // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_asin"] = mathAsinFunc

	// runtime_math_acos(x: f64) -> f64
	// 反余弦函数
	mathAcosFunc := m.module.NewFunc("runtime_math_acos",
		types.Double, // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_acos"] = mathAcosFunc

	// runtime_math_atan(x: f64) -> f64
	// 反正切函数
	mathAtanFunc := m.module.NewFunc("runtime_math_atan",
		types.Double, // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_atan"] = mathAtanFunc

	// runtime_math_ceil(x: f64) -> f64
	// 向上取整
	mathCeilFunc := m.module.NewFunc("runtime_math_ceil",
		types.Double, // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_ceil"] = mathCeilFunc

	// runtime_math_floor(x: f64) -> f64
	// 向下取整
	mathFloorFunc := m.module.NewFunc("runtime_math_floor",
		types.Double, // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_floor"] = mathFloorFunc

	// runtime_math_round(x: f64) -> f64
	// 四舍五入
	mathRoundFunc := m.module.NewFunc("runtime_math_round",
		types.Double, // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_round"] = mathRoundFunc

	// runtime_math_trunc(x: f64) -> f64
	// 截断（向零取整）
	mathTruncFunc := m.module.NewFunc("runtime_math_trunc",
		types.Double, // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_trunc"] = mathTruncFunc

	// runtime_math_atan2(y: f64, x: f64) -> f64
	// 反正切函数（两个参数版本）
	mathAtan2Func := m.module.NewFunc("runtime_math_atan2",
		types.Double, // f64 -> double
		ir.NewParam("y", types.Double), // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_atan2"] = mathAtan2Func

	// runtime_math_sinh(x: f64) -> f64
	// 双曲正弦函数
	mathSinhFunc := m.module.NewFunc("runtime_math_sinh",
		types.Double, // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_sinh"] = mathSinhFunc

	// runtime_math_cosh(x: f64) -> f64
	// 双曲余弦函数
	mathCoshFunc := m.module.NewFunc("runtime_math_cosh",
		types.Double, // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_cosh"] = mathCoshFunc

	// runtime_math_tanh(x: f64) -> f64
	// 双曲正切函数
	mathTanhFunc := m.module.NewFunc("runtime_math_tanh",
		types.Double, // f64 -> double
		ir.NewParam("x", types.Double), // f64 -> double
	)
	m.externalFunctions["runtime_math_tanh"] = mathTanhFunc

	// ==================== HTTP 和 URL 处理 ====================

	// runtime_url_parse(url: string) -> *UrlParseResult
	// 解析URL字符串，提取scheme、host、port、path等组件
	// 编译器签名：i8* runtime_url_parse(i8* url)
	urlParseFunc := m.module.NewFunc("runtime_url_parse",
		types.NewPointer(types.I8), // UrlParseResult* -> *i8
		ir.NewParam("url", types.NewPointer(types.I8)), // string -> *i8
	)
	m.externalFunctions["runtime_url_parse"] = urlParseFunc

	// runtime_http_parse_request(data: []u8) -> *HttpParseRequestResult
	// 解析HTTP请求字节数组，提取请求行、头部、正文
	// 编译器签名：i8* runtime_http_parse_request(i8* data, i32 data_len)
	httpParseRequestFunc := m.module.NewFunc("runtime_http_parse_request",
		types.NewPointer(types.I8), // HttpParseRequestResult* -> *i8
		ir.NewParam("data", types.NewPointer(types.I8)), // []u8 数据指针
		ir.NewParam("data_len", types.I32),              // []u8 长度
	)
	m.externalFunctions["runtime_http_parse_request"] = httpParseRequestFunc

	// runtime_http_build_response(...) -> *HttpBuildResponseResult
	// 构建HTTP响应字节数组，包含状态行、头部、正文
	// 编译器签名：i8* runtime_http_build_response(i32 status_code, i8* status_text, i8** header_keys, i8** header_values, i32 header_count, i8* body, i32 body_len)
	httpBuildResponseFunc := m.module.NewFunc("runtime_http_build_response",
		types.NewPointer(types.I8), // HttpBuildResponseResult* -> *i8
		ir.NewParam("status_code", types.I32),                    // 状态码
		ir.NewParam("status_text", types.NewPointer(types.I8)),   // 状态文本
		ir.NewParam("header_keys", types.NewPointer(types.NewPointer(types.I8))),   // 头部键数组（char**）
		ir.NewParam("header_values", types.NewPointer(types.NewPointer(types.I8))), // 头部值数组（char**）
		ir.NewParam("header_count", types.I32),                  // 头部数量
		ir.NewParam("body", types.NewPointer(types.I8)),          // 响应正文（[]u8 数据指针）
		ir.NewParam("body_len", types.I32),                       // 正文长度
	)
	m.externalFunctions["runtime_http_build_response"] = httpBuildResponseFunc

	// runtime_null_ptr() -> *i8
	// 返回 NULL 指针，用于在Echo语言中表示空指针
	// 编译器签名：i8* runtime_null_ptr()
	nullPtrFunc := m.module.NewFunc("runtime_null_ptr",
		types.NewPointer(types.I8), // 返回 NULL 指针
	)
	m.externalFunctions["runtime_null_ptr"] = nullPtrFunc
}

// getStructSize 计算结构体类型的大小（字节数）
// 在LLVM IR中，结构体的大小需要考虑对齐（alignment）
// 在64位系统上，结构体字段通常需要对齐到其大小的边界
func (m *IRModuleManagerImpl) getStructSize(structType *types.StructType) (int64, error) {
	// 使用LLVM的类型大小计算
	// 注意：需要考虑对齐（alignment）因素
	// 在64位系统上，结构体字段通常需要对齐到其大小的边界
	
	var totalSize int64 = 0
	var maxAlignment int64 = 1
	
	for _, fieldType := range structType.Fields {
		fieldSize, err := m.getTypeSize(fieldType)
		if err != nil {
			return 0, fmt.Errorf("failed to get field size: %w", err)
		}
		
		// 计算字段的对齐要求（通常是字段大小，但不超过8字节）
		fieldAlignment := fieldSize
		if fieldAlignment > 8 {
			fieldAlignment = 8
		}
		if fieldAlignment > maxAlignment {
			maxAlignment = fieldAlignment
		}
		
		// 对齐当前总大小到字段对齐边界
		if totalSize%fieldAlignment != 0 {
			totalSize = (totalSize/fieldAlignment + 1) * fieldAlignment
		}
		
		// 添加字段大小
		totalSize += fieldSize
	}
	
	// 结构体总大小需要对齐到最大对齐要求
	if totalSize%maxAlignment != 0 {
		totalSize = (totalSize/maxAlignment + 1) * maxAlignment
	}
	
	return totalSize, nil
}

// getTypeSize 计算类型的大小（字节数）
func (m *IRModuleManagerImpl) getTypeSize(typ types.Type) (int64, error) {
	switch t := typ.(type) {
	case *types.IntType:
		// int类型的大小（i32 = 4字节，i64 = 8字节）
		switch t.BitSize {
		case 32:
			return 4, nil
		case 64:
			return 8, nil
		default:
			return int64(t.BitSize / 8), nil
		}
	case *types.PointerType:
		// 指针类型的大小（通常是8字节，64位系统）
		return 8, nil
	case *types.FloatType:
		// float类型的大小（f32 = 4字节，f64 = 8字节）
		switch t.Kind {
		case types.FloatKindFloat:
			return 4, nil
		case types.FloatKindDouble:
			return 8, nil
		default:
			return 4, nil
		}
	case *types.StructType:
		// 结构体类型：递归计算所有字段的大小
		return m.getStructSize(t)
	default:
		return 0, fmt.Errorf("unsupported type for size calculation: %T", typ)
	}
}
