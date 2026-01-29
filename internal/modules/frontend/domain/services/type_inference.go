package services

import (
	"fmt"
	"strings"

	"echo/internal/modules/frontend/domain/entities"
)

// TypeInferenceService 类型推断服务接口
type TypeInferenceService interface {
	// InferTypes 为AST中的变量声明推断类型
	InferTypes(program *entities.Program) error
}

// SimpleTypeInferenceService 简单类型推断服务实现
// 支持符号表，用于从变量引用、函数调用、方法调用、字段访问等推断类型
type SimpleTypeInferenceService struct {
	// symbolTable 符号表：变量名 -> 类型字符串
	// 用于支持从变量引用推断类型
	symbolTable map[string]string
	// functionTable 函数表：函数名 -> 返回类型字符串
	// 用于支持从函数调用推断类型
	functionTable map[string]string
	// structTable 结构体表：结构体名 -> 字段映射（字段名 -> 字段类型）
	// 用于支持从结构体字段访问推断类型
	structTable map[string]map[string]string
	// methodTable 方法表：接收者类型 + 方法名 -> 返回类型字符串
	// 用于支持从方法调用推断类型
	// 键格式："{ReceiverType}.{MethodName}"，如 "Point.distance"
	methodTable map[string]string
	// packageExports 包导出表：导入路径 -> 符号名 -> 类型字符串
	// 用于 from <path> import x, y 时将 x,y 注册到 functionTable/symbolTable
	// 由 StdlibRegistry 或调用方在 InferTypes 前填充
	packageExports map[string]map[string]string
}

// NewSimpleTypeInferenceService 创建类型推断服务
func NewSimpleTypeInferenceService() TypeInferenceService {
	return &SimpleTypeInferenceService{
		symbolTable:     make(map[string]string),
		functionTable:   make(map[string]string),
		structTable:    make(map[string]map[string]string),
		methodTable:     make(map[string]string),
		packageExports: make(map[string]map[string]string),
	}
}

// SetPackageExports 设置某包的导出符号（供 StdlibRegistry 调用）
// path 如 "math"、"utils"；exports 为 符号名 -> 类型
func (s *SimpleTypeInferenceService) SetPackageExports(path string, exports map[string]string) {
	if s.packageExports == nil {
		s.packageExports = make(map[string]map[string]string)
	}
	s.packageExports[path] = exports
}

// InferTypes 实现类型推断逻辑
func (s *SimpleTypeInferenceService) InferTypes(program *entities.Program) error {
	return s.inferTypesInProgram(program)
}

// inferTypesInProgram 遍历程序中的所有节点，进行类型推断
func (s *SimpleTypeInferenceService) inferTypesInProgram(program *entities.Program) error {
	// 保存标准库模块符号（在重置前）
	stdlibSymbols := make(map[string]string)
	stdlibMethods := make(map[string]string)
	stdlibFunctions := make(map[string]string)
	
	// 保存所有标准库模块符号和方法
	// 模块符号：time, math, atomic, sync, io, core, collections 等
	for k, v := range s.symbolTable {
		if strings.HasPrefix(v, "module:") || 
			k == "time" || k == "math" || k == "atomic" || k == "sync" || 
			k == "io" || k == "core" || k == "collections" {
			stdlibSymbols[k] = v
		}
	}
	// 模块函数和方法：所有以 "module:" 开头的方法，以及标准库类型、内置类型的方法
	// 标准库类型：string, Time, AtomicInt, AtomicBool, Option[T], Result[T, E], Vec[T], Mutex[T], RwLock[T] 等
	// 内置类型：i64, i32, int, f64, float, bool 的 print_string/print_int 等
	stdlibTypePrefixes := []string{
		"module:",
		"string.",
		"Time.",
		"AtomicInt.",
		"AtomicBool.",
		"Option[T].",
		"Result[T, E].",
		"Vec[T].",
		"Mutex[T].",
		"RwLock[T].",
		"Duration.",
		"File.",
		"StringBuilder.",
		"i64.", "i32.", "int.",
		"f64.", "float.",
		"bool.",
		"void.",
		"string.",
		"map[T, E].",
		"[T].",
	}
	for k, v := range s.methodTable {
		for _, prefix := range stdlibTypePrefixes {
			if strings.HasPrefix(k, prefix) {
				stdlibMethods[k] = v
				break
			}
		}
	}
	// 保存内置函数（len、Option 等），重置后需恢复
	for _, name := range []string{"len", "Option"} {
		if v, ok := s.functionTable[name]; ok {
			stdlibFunctions[name] = v
		}
	}
	
	// 重置符号表
	s.symbolTable = make(map[string]string)
	s.functionTable = make(map[string]string)
	s.structTable = make(map[string]map[string]string)
	s.methodTable = make(map[string]string)
	
	// 恢复标准库模块符号
	for k, v := range stdlibSymbols {
		s.symbolTable[k] = v
	}
	for k, v := range stdlibMethods {
		s.methodTable[k] = v
	}
	for k, v := range stdlibFunctions {
		s.functionTable[k] = v
	}
	
	// 第一遍：收集定义（函数、结构体、方法、from-import 符号）
	// ✅ 修复：递归收集函数内部的结构体定义
	var collectDefinitions func(stmts []entities.ASTNode)
	collectDefinitions = func(stmts []entities.ASTNode) {
		for _, stmt := range stmts {
			// from ... import：将导入的符号注册到 functionTable/symbolTable
			if fromImport, ok := stmt.(*entities.FromImportStatement); ok {
				exports := s.packageExports[fromImport.ImportPath]
				if exports != nil {
					for _, elem := range fromImport.Elements {
						typ, has := exports[elem.Name]
						if !has {
							continue
						}
						exportName := elem.Alias
						if exportName == "" {
							exportName = elem.Name
						}
						s.functionTable[exportName] = typ
						// 同时作为类型名注册到 symbolTable，便于 let x: Point 等
						s.symbolTable[exportName] = typ
					}
				}
				continue
			}
			// 收集函数定义（函数名 -> 返回类型）
			if funcDef, ok := stmt.(*entities.FuncDef); ok {
				if funcDef.ReturnType != "" {
					s.functionTable[funcDef.Name] = funcDef.ReturnType
				}
				// ✅ 修复：递归收集函数体中的结构体定义
				collectDefinitions(funcDef.Body)
			}
			
			// ✅ 修复：收集异步函数定义（函数名 -> Future[返回类型]）
			if asyncFuncDef, ok := stmt.(*entities.AsyncFuncDef); ok {
				// 异步函数返回 Future[ReturnType]
				if asyncFuncDef.ReturnType != "" {
					returnType := fmt.Sprintf("Future[%s]", asyncFuncDef.ReturnType)
					s.functionTable[asyncFuncDef.Name] = returnType
				} else {
					// 如果没有返回类型，默认为 Future[void]
					s.functionTable[asyncFuncDef.Name] = "Future[void]"
				}
				// ✅ 修复：递归收集异步函数体中的结构体定义
				collectDefinitions(asyncFuncDef.Body)
			}
			
			// 收集结构体定义（结构体名 -> 字段映射）
			if structDef, ok := stmt.(*entities.StructDef); ok {
				fieldMap := make(map[string]string)
				for _, field := range structDef.Fields {
					fieldMap[field.Name] = field.Type
				}
				s.structTable[structDef.Name] = fieldMap
			}
			
			// 收集方法定义（接收者类型 + 方法名 -> 返回类型）
			if methodDef, ok := stmt.(*entities.MethodDef); ok {
				if methodDef.ReturnType != "" {
					// 键格式："{ReceiverType}.{MethodName}"
					key := fmt.Sprintf("%s.%s", methodDef.Receiver, methodDef.Name)
					s.methodTable[key] = methodDef.ReturnType
				}
				// ✅ 修复：递归收集方法体中的结构体定义
				collectDefinitions(methodDef.Body)
			}
		}
	}
	collectDefinitions(program.Statements)
	
	// 第二遍：进行类型推断（此时可以查询所有表）
	for _, stmt := range program.Statements {
		if err := s.inferTypesInStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

// inferTypesInFunction 在函数中进行类型推断
func (s *SimpleTypeInferenceService) inferTypesInFunction(funcDef *entities.FuncDef) error {
	// 将函数参数添加到符号表
	for _, param := range funcDef.Params {
		s.symbolTable[param.Name] = param.Type
	}
	
	// ✅ 修复：先收集函数内部定义的函数和结构体（第一遍）
	for _, stmt := range funcDef.Body {
		// 收集函数内部定义的函数
		if innerFuncDef, ok := stmt.(*entities.FuncDef); ok {
			if innerFuncDef.ReturnType != "" {
				s.functionTable[innerFuncDef.Name] = innerFuncDef.ReturnType
			}
		}
		// ✅ 修复：收集函数内部定义的异步函数
		if innerAsyncFuncDef, ok := stmt.(*entities.AsyncFuncDef); ok {
			// 异步函数返回 Future[ReturnType]
			if innerAsyncFuncDef.ReturnType != "" {
				returnType := fmt.Sprintf("Future[%s]", innerAsyncFuncDef.ReturnType)
				s.functionTable[innerAsyncFuncDef.Name] = returnType
			} else {
				// 如果没有返回类型，默认为 Future[void]
				s.functionTable[innerAsyncFuncDef.Name] = "Future[void]"
			}
		}
		// 收集函数内部定义的结构体
		if structDef, ok := stmt.(*entities.StructDef); ok {
			fieldMap := make(map[string]string)
			for _, field := range structDef.Fields {
				fieldMap[field.Name] = field.Type
			}
			s.structTable[structDef.Name] = fieldMap
		}
	}
	
	// 推断函数体中的变量类型（第二遍）
	for _, stmt := range funcDef.Body {
		if err := s.inferTypesInStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

// inferTypesInStatement 在语句中进行类型推断
func (s *SimpleTypeInferenceService) inferTypesInStatement(stmt entities.ASTNode) error {
	switch stmt := stmt.(type) {
	case *entities.VarDecl:
		return s.inferType(stmt)
	case *entities.FuncDef:
		return s.inferTypesInFunction(stmt)
	}
	return nil
}

// inferType 为变量声明推断类型
func (s *SimpleTypeInferenceService) inferType(varDecl *entities.VarDecl) error {
	var varType string
	
	if !varDecl.Inferred {
		// 已经有显式类型，不需要推断
		varType = varDecl.Type
	} else {
		// 需要推断类型
		// 检查 Value 是否为 nil（可能发生在 mut 变量声明中）
		if varDecl.Value == nil {
			return fmt.Errorf("cannot infer type for variable %s: value expression is nil", varDecl.Name)
		}
		inferredType, err := s.inferTypeFromExpression(varDecl.Value)
		if err != nil {
			return fmt.Errorf("cannot infer type for variable %s: %v", varDecl.Name, err)
		}
		varType = inferredType
		varDecl.Type = inferredType
		varDecl.Inferred = false // 标记为已推断
	}
	
	// 将变量添加到符号表（无论是否推断，都需要记录）
	s.symbolTable[varDecl.Name] = varType
	
	return nil
}

// inferTypeFromExpression 从表达式推断类型
func (s *SimpleTypeInferenceService) inferTypeFromExpression(expr entities.Expr) (string, error) {
	switch e := expr.(type) {
	// 已有支持：基础字面量
	case *entities.IntLiteral:
		return "int", nil
	case *entities.FloatLiteral:
		return "float", nil
	case *entities.StringLiteral:
		return "string", nil
	case *entities.BoolLiteral:
		return "bool", nil
	case *entities.BinaryExpr:
		// 对于二元表达式，需要根据操作符和操作数推断
		return s.inferTypeFromBinaryExpr(e)
	
	// 新增支持（第一阶段）：无需符号表的表达式类型
	case *entities.ArrayLiteral:
		return s.inferTypeFromArrayLiteral(e)
	case *entities.StructLiteral:
		return s.inferTypeFromStructLiteral(e)
	case *entities.IndexExpr:
		return s.inferTypeFromIndexExpr(e)
	case *entities.SliceExpr:
		return s.inferTypeFromSliceExpr(e)
	
	// 新增支持（第二阶段）：需要符号表的表达式类型
	case *entities.Identifier:
		return s.inferTypeFromIdentifier(e)
	case *entities.FuncCall:
		return s.inferTypeFromFuncCall(e)
	case *entities.MethodCallExpr:
		return s.inferTypeFromMethodCall(e)
	case *entities.StructAccess:
		return s.inferTypeFromFieldAccess(e)
	case *entities.ArrayMethodCallExpr:
		// 数组方法调用：先推断数组类型，然后根据方法名推断返回类型
		return s.inferTypeFromArrayMethodCall(e)
	case *entities.LenExpr:
		// len() 函数总是返回 int 类型
		return s.inferTypeFromLenExpr(e)
	case *entities.SizeOfExpr:
		// sizeof(T) 表达式总是返回 int 类型
		return s.inferTypeFromSizeOfExpr(e)
	case *entities.FunctionPointerExpr:
		// &func_name 表达式总是返回 *i8 类型（通用指针）
		return s.inferTypeFromFunctionPointerExpr(e)
	case *entities.AddressOfExpr:
		// &expression 表达式总是返回 *i8 类型（通用指针）
		return s.inferTypeFromAddressOfExpr(e)
	case *entities.ResultExpr:
		// Result 类型表达式：Result[OkType, ErrType]
		return s.inferTypeFromResultExpr(e)
	case *entities.OptionExpr:
		// Option 类型表达式：Option[InternalType]
		return s.inferTypeFromOptionExpr(e)
	case *entities.OkLiteral:
		// Ok 字面量：Ok(value) -> Result[ValueType, string]
		return s.inferTypeFromOkLiteral(e)
	case *entities.ErrLiteral:
		// Err 字面量：Err(error) -> Result[T, ErrorType]（T 需要上下文，暂时返回错误）
		return s.inferTypeFromErrLiteral(e)
	case *entities.SomeLiteral:
		// Some 字面量：Some(value) -> Option[ValueType]
		return s.inferTypeFromSomeLiteral(e)
	case *entities.NoneLiteral:
		// None 字面量：None -> Option[T]（T 需要上下文，暂时返回错误）
		return s.inferTypeFromNoneLiteral(e)
	case *entities.ChanLiteral:
		// Channel 字面量：chan[Type]() -> chan[Type]
		return s.inferTypeFromChanLiteral(e)
	case *entities.SendExpr:
		// 发送表达式：channel <- value -> void
		return s.inferTypeFromSendExpr(e)
	case *entities.ReceiveExpr:
		// 接收表达式：<-channel -> ElementType（从 Channel 类型提取）
		return s.inferTypeFromReceiveExpr(e)
	case *entities.AwaitExpr:
		// 异步等待表达式：await future -> T（从 Future[T] 提取）
		return s.inferTypeFromAwaitExpr(e)
	case *entities.SpawnExpr:
		// 生成协程表达式：spawn function() -> Future[T]（T 是函数返回类型）
		return s.inferTypeFromSpawnExpr(e)
	case *entities.TernaryExpr:
		// 三元表达式：if condition { thenValue } else { elseValue }
		// 类型应该是 then 和 else 分支的公共类型
		return s.inferTypeFromTernaryExpr(e)
	
	default:
		return "", fmt.Errorf("unsupported expression type for type inference: %T (value: %v)", expr, expr)
	}
}

// inferTypeFromBinaryExpr 从二元表达式推断类型
func (s *SimpleTypeInferenceService) inferTypeFromBinaryExpr(expr *entities.BinaryExpr) (string, error) {
	// ✅ 修复：通道发送运算符 channel <- value -> void
	if expr.Op == "<-" {
		// 检查左边是否是通道类型
		leftType, err := s.inferTypeFromExpression(expr.Left)
		if err != nil {
			return "", err
		}
		// 验证左边是通道类型（格式：chan[T] 或 chan [T]）
		normalizedType := strings.ReplaceAll(leftType, " ", "")
		if !strings.HasPrefix(normalizedType, "chan[") {
			return "", fmt.Errorf("channel send operator '<-' requires channel type on left, got: %s", leftType)
		}
		// 通道发送表达式返回 void
		return "void", nil
	}
	
	leftType, err := s.inferTypeFromExpression(expr.Left)
	if err != nil {
		return "", err
	}

	rightType, err := s.inferTypeFromExpression(expr.Right)
	if err != nil {
		return "", err
	}

	// 类型必须相同
	if leftType != rightType {
		return "", fmt.Errorf("type mismatch in binary expression: %s vs %s", leftType, rightType)
	}

	// 根据操作符确定结果类型
	switch expr.Op {
	case "+", "-", "*", "/":
		// 算术运算：int 和 float 都支持
		if leftType == "int" || leftType == "float" {
			return leftType, nil
		}
		// 字符串拼接
		if leftType == "string" && expr.Op == "+" {
			return "string", nil
		}
	case "==", "!=", "<", "<=", ">", ">=":
		// 比较运算：返回 bool
		return "bool", nil
	case "&&", "||":
		// 逻辑运算：需要 bool 类型
		if leftType == "bool" {
			return "bool", nil
		}
	}

	return "", fmt.Errorf("unsupported binary operation: %s %s %s", leftType, expr.Op, rightType)
}

// inferTypeFromArrayLiteral 从数组字面量推断类型
// 从数组元素类型推断数组类型，所有元素类型必须相同
func (s *SimpleTypeInferenceService) inferTypeFromArrayLiteral(expr *entities.ArrayLiteral) (string, error) {
	if len(expr.Elements) == 0 {
		// 空数组字面量，推断为 []i8*（运行时切片指针）
		// 这是简化处理，实际应该从上下文推断具体类型
		// 但在类型推断阶段，我们无法从上下文推断，所以使用默认类型
		return "[]i8*", nil
	}

	// 推断第一个元素的类型
	elementType, err := s.inferTypeFromExpression(expr.Elements[0])
	if err != nil {
		return "", fmt.Errorf("cannot infer element type: %v", err)
	}

	// 验证所有元素类型相同
	for i := 1; i < len(expr.Elements); i++ {
		elemType, err := s.inferTypeFromExpression(expr.Elements[i])
		if err != nil {
			return "", fmt.Errorf("cannot infer element type at index %d: %v", i, err)
		}
		if elemType != elementType {
			return "", fmt.Errorf("array elements have different types: element[0] is %s, element[%d] is %s",
				elementType, i, elemType)
		}
	}

	// 返回数组类型：[ElementType]
	return fmt.Sprintf("[%s]", elementType), nil
}

// inferTypeFromStructLiteral 从结构体字面量推断类型
// 结构体字面量在解析时已确定类型，直接返回类型名
func (s *SimpleTypeInferenceService) inferTypeFromStructLiteral(expr *entities.StructLiteral) (string, error) {
	if expr.Type == "" {
		return "", fmt.Errorf("struct literal missing type information")
	}
	return expr.Type, nil
}

// inferTypeFromIndexExpr 从索引表达式推断类型
// 支持数组类型（[ElementType]）和 map 类型（map[K]V）
func (s *SimpleTypeInferenceService) inferTypeFromIndexExpr(expr *entities.IndexExpr) (string, error) {
	// 推断数组或 map 表达式的类型
	arrayOrMapType, err := s.inferTypeFromExpression(expr.Array)
	if err != nil {
		return "", fmt.Errorf("cannot infer array or map type: %v", err)
	}

	// 检查是否是数组类型（格式：[ElementType]）
	if strings.HasPrefix(arrayOrMapType, "[") && strings.HasSuffix(arrayOrMapType, "]") {
		// 提取元素类型
		elementType := strings.TrimPrefix(strings.TrimSuffix(arrayOrMapType, "]"), "[")
		if elementType == "" {
			return "", fmt.Errorf("invalid array type format: %s", arrayOrMapType)
		}
		return elementType, nil
	}

	// 检查是否是 map 类型（格式：map[K]V）
	if strings.HasPrefix(arrayOrMapType, "map[") {
		// 使用 parseGenericType 解析 map 类型
		baseType, typeArgs, err := s.parseGenericType(arrayOrMapType)
		if err != nil {
			return "", fmt.Errorf("cannot parse map type: %v", err)
		}
		if baseType == "map" && len(typeArgs) == 2 {
			// map[K]V：返回 value 类型（第二个类型参数）
			return typeArgs[1], nil
		}
		return "", fmt.Errorf("invalid map type format: %s (expected map[K]V, got baseType=%s, typeArgs=%v)", arrayOrMapType, baseType, typeArgs)
	}

	// 既不是数组也不是 map
	return "", fmt.Errorf("index expression requires array or map type, got: %s", arrayOrMapType)
}

// inferTypeFromSliceExpr 从切片表达式推断类型
// 切片类型与数组类型相同
func (s *SimpleTypeInferenceService) inferTypeFromSliceExpr(expr *entities.SliceExpr) (string, error) {
	// 推断数组表达式的类型
	arrayType, err := s.inferTypeFromExpression(expr.Array)
	if err != nil {
		return "", fmt.Errorf("cannot infer array type: %v", err)
	}

	// 验证是数组类型
	if !strings.HasPrefix(arrayType, "[") || !strings.HasSuffix(arrayType, "]") {
		return "", fmt.Errorf("slice expression requires array type, got: %s", arrayType)
	}

	// 切片类型与数组类型相同
	return arrayType, nil
}

// inferTypeFromIdentifier 从变量引用推断类型
// 从符号表查找变量类型
// 注意：如果是模块标识符（如 "string", "time"），返回 "module:xxx" 格式
func (s *SimpleTypeInferenceService) inferTypeFromIdentifier(expr *entities.Identifier) (string, error) {
	// ✅ 修复：检查是否是类型表达式（如 chan [int]）
	// 如果标识符名是类型名格式（如 chan [int]），直接返回类型名
	if strings.HasPrefix(expr.Name, "chan ") {
		// chan [T] 格式：返回 chan[T] 类型（去掉空格）
		typeName := strings.ReplaceAll(expr.Name, " ", "")
		return typeName, nil
	}
	
	varType, exists := s.symbolTable[expr.Name]
	if !exists {
		// 检查是否是标准库模块（string 是内置类型，但也可能是模块）
		// 如果符号表中没有，但方法表中有 "module:xxx.method" 格式的方法，则认为是模块
		for k := range s.methodTable {
			if strings.HasPrefix(k, "module:"+expr.Name+".") {
				return "module:" + expr.Name, nil
			}
		}
		return "", fmt.Errorf("cannot infer type from variable reference '%s' - variable not found in symbol table", expr.Name)
	}
	return varType, nil
}

// inferTypeFromFuncCall 从函数调用推断类型
// 从函数表查找函数返回类型；对 Option(x) 根据参数类型返回 Option[argType]
func (s *SimpleTypeInferenceService) inferTypeFromFuncCall(expr *entities.FuncCall) (string, error) {
	// ✅ 修复：通道接收运算符 <-channel -> ElementType
	if expr.Name == "<-" && len(expr.Args) >= 1 {
		// 推断 Channel 类型
		channelType, err := s.inferTypeFromExpression(expr.Args[0])
		if err != nil {
			return "", fmt.Errorf("cannot infer type from channel receive '<-' - cannot infer channel type: %w", err)
		}
		// ✅ 修复：支持 chan[ElementType] 和 chan ElementType 格式
		// 标准化通道类型格式：将 "chan int" 转换为 "chan[int]"
		normalizedType := strings.ReplaceAll(channelType, " ", "")
		
		// 如果格式是 "chanint"（chan 和类型之间没有空格也没有方括号），需要手动添加方括号
		// 例如："chanint" -> "chan[int]"
		if strings.HasPrefix(normalizedType, "chan") && !strings.HasPrefix(normalizedType, "chan[") {
			// 提取类型部分（去掉 "chan" 前缀）
			elementType := normalizedType[4:] // "chan" 长度为 4
			if elementType != "" {
				normalizedType = "chan[" + elementType + "]"
			}
		}
		
		// 解析 Channel 类型格式：chan[ElementType]
		if !strings.HasPrefix(normalizedType, "chan[") || !strings.HasSuffix(normalizedType, "]") {
			return "", fmt.Errorf("cannot infer type from channel receive '<-' - invalid channel type format: %s (expected chan[ElementType])", channelType)
		}
		// 提取元素类型
		elementType := normalizedType[5 : len(normalizedType)-1] // 提取 "chan[" 和 "]" 之间的内容
		return elementType, nil
	}
	
	// Option(value) 根据第一个参数类型返回 Option[argType]
	if expr.Name == "Option" && len(expr.Args) >= 1 {
		argType, err := s.inferTypeFromExpression(expr.Args[0])
		if err != nil {
			return "", fmt.Errorf("cannot infer type from Option() - argument type: %w", err)
		}
		return fmt.Sprintf("Option[%s]", argType), nil
	}
	
	// ✅ 修复：处理模块泛型函数调用（如 collections.new[int]()）
	// 格式：module.function[TypeArgs]()
	if strings.Contains(expr.Name, ".") && strings.Contains(expr.Name, "[") {
		// 解析模块名、函数名和泛型参数
		// 例如："collections.new[int]" -> module="collections", func="new", typeArgs=["int"]
		parts := strings.Split(expr.Name, ".")
		if len(parts) == 2 {
			moduleName := parts[0]
			funcWithGeneric := parts[1]
			
			// 检查是否有泛型参数
			if strings.Contains(funcWithGeneric, "[") && strings.Contains(funcWithGeneric, "]") {
				// 提取函数名和泛型参数
				genericStart := strings.Index(funcWithGeneric, "[")
				funcName := funcWithGeneric[:genericStart]
				genericPart := funcWithGeneric[genericStart+1 : len(funcWithGeneric)-1] // 去掉 [ 和 ]
				
				// ✅ 修复：查找模块函数（格式：module:moduleName.funcName）
				// 注意：模块函数可能注册在 methodTable 中，而不是 functionTable 中
				moduleFuncKey := fmt.Sprintf("module:%s.%s", moduleName, funcName)
				
				// 先查找 methodTable
				returnType, exists := s.methodTable[moduleFuncKey]
				if !exists {
					// 如果 methodTable 中没有，再查找 functionTable
					returnType, exists = s.functionTable[moduleFuncKey]
				}
				
				if exists {
					// 替换返回类型中的泛型参数
					// 例如：Vec[T] + [int] -> Vec[int]
					instantiatedType := strings.ReplaceAll(returnType, "T", genericPart)
					// 如果返回类型是 Vec[T]，需要替换为 Vec[int]
					// 但也要处理多个泛型参数的情况（如 HashMap[K, V]）
					// 暂时只处理单个泛型参数的情况
					return instantiatedType, nil
				}
			}
		}
	}
	
	returnType, exists := s.functionTable[expr.Name]
	if !exists {
		return "", fmt.Errorf("cannot infer type from function call '%s' - function not found in function table", expr.Name)
	}
	return returnType, nil
}

// inferTypeFromMethodCall 从方法调用推断类型
// 1. 推断接收者类型
// 2. 查找方法定义（接收者类型 + 方法名 -> 返回类型）
// 3. 支持模块函数调用（如 time.now()、math.sqrt()）
// 4. 支持泛型类型匹配（如 Vec[T] 可以匹配 Vec[int]）
func (s *SimpleTypeInferenceService) inferTypeFromMethodCall(expr *entities.MethodCallExpr) (string, error) {
	// 1. 推断接收者类型
	receiverType, err := s.inferTypeFromExpression(expr.Receiver)
	if err != nil {
		return "", fmt.Errorf("cannot infer receiver type for method call '%s': %v", expr.MethodName, err)
	}
	
	// 2. 检查是否是模块函数调用（接收者类型为 "module:xxx"）
	if strings.HasPrefix(receiverType, "module:") {
		// 模块函数调用：查找 "module:xxx.methodName"
		key := fmt.Sprintf("%s.%s", receiverType, expr.MethodName)
		returnType, exists := s.methodTable[key]
		if !exists {
			return "", fmt.Errorf("cannot infer type from module function call '%s.%s' - function not found in method table", receiverType, expr.MethodName)
		}
		// 如果返回类型包含泛型参数（如 "Vec[T]"），需要根据调用上下文替换
		// 但模块函数调用通常不涉及泛型参数替换，直接返回
		return returnType, nil
	}
	
	// 3. 普通方法调用：先尝试精确匹配
	key := fmt.Sprintf("%s.%s", receiverType, expr.MethodName)
	returnType, exists := s.methodTable[key]
	if exists {
		return returnType, nil
	}
	
	// 4. 如果精确匹配失败，尝试泛型类型匹配
	// 例如：receiverType = "Vec[int]", methodName = "push"
	// 尝试匹配 "Vec[T].push" 并替换类型参数
	genericMatch, genericErr := s.findGenericMethodMatch(receiverType, expr.MethodName)
	if genericErr == nil && genericMatch != "" {
		return genericMatch, nil
	}
	// 如果泛型匹配失败，记录错误以便调试
	err = genericErr
	
	// 5. 如果泛型匹配也失败，返回详细错误信息（包含尝试的键）
	// 用于调试
	attemptedKeys := []string{key}
	if err == nil {
		// 如果 findGenericMethodMatch 返回了错误，说明不是泛型类型或方法不存在
		baseType, typeArgs, parseErr := s.parseGenericType(receiverType)
		if parseErr == nil && len(typeArgs) > 0 {
			genericKey := fmt.Sprintf("%s[T].%s", baseType, expr.MethodName)
			attemptedKeys = append(attemptedKeys, genericKey)
		}
	}
	
	return "", fmt.Errorf("cannot infer type from method call '%s' on type '%s' - method not found in method table (attempted keys: %v)", expr.MethodName, receiverType, attemptedKeys)
}

// findGenericMethodMatch 查找泛型类型的方法匹配
// 例如：receiverType = "Vec[int]", methodName = "push"
// 尝试匹配 "Vec[T].push" 并替换类型参数
// 注意：对于方法本身也是泛型的情况（如 ok_or[E]），需要从参数推断方法的类型参数
func (s *SimpleTypeInferenceService) findGenericMethodMatch(receiverType, methodName string) (string, error) {
	// 1. 解析接收者类型，提取基础类型和类型参数
	// 例如："Vec[int]" -> baseType = "Vec", typeArgs = ["int"]
	baseType, typeArgs, err := s.parseGenericType(receiverType)
	if err != nil {
		return "", err
	}
	
	// 如果没有类型参数，说明不是泛型类型，无法匹配
	if len(typeArgs) == 0 {
		return "", fmt.Errorf("not a generic type: %s", receiverType)
	}
	
	// 2. 构造泛型方法键，尝试多种格式
	// 对于单类型参数：Vec[T].push
	// 对于切片 [int]：baseType 为 "["，键应为 "[T].pop" 而非 "[[T].pop"
	// 对于多类型参数：Result[T, E].ok
	var genericKey string
	var genericReturnType string
	var exists bool
	
	if baseType == "[" {
		// 切片类型 [T]：methodTable 键为 "[T].pop" 等
		genericKey = "[T]." + methodName
	} else {
		// 普通泛型：BaseType[T].method
		genericKey = fmt.Sprintf("%s[T].%s", baseType, methodName)
	}
	genericReturnType, exists = s.methodTable[genericKey]
	
	// 如果单类型参数格式失败，且类型参数数量 > 1，尝试多类型参数格式
	if !exists && len(typeArgs) > 1 {
		// 尝试双类型参数格式：BaseType[T, E].method
		genericKey = fmt.Sprintf("%s[T, E].%s", baseType, methodName)
		genericReturnType, exists = s.methodTable[genericKey]
	}
	
	// 如果仍然失败，返回错误
	if !exists {
		// 调试：检查方法表中是否有相关的方法
		var relatedKeys []string
		for k := range s.methodTable {
			if strings.Contains(k, baseType) && strings.Contains(k, methodName) {
				relatedKeys = append(relatedKeys, k)
			}
		}
		return "", fmt.Errorf("generic method not found: %s (method table has %d entries, related keys: %v)", genericKey, len(s.methodTable), relatedKeys)
	}
	
	// 4. 替换返回类型中的类型参数
	// 例如：genericReturnType = "Option[T]", typeArgs = ["int"]
	// 结果：Option[int]
	// 注意：如果返回类型包含方法的类型参数（如 Result[T, E]），需要特殊处理
	returnType := s.substituteTypeParameters(genericReturnType, typeArgs)
	
	// 5. 特殊处理：如果返回类型包含方法的类型参数（如 Result[T, E] 中的 E）
	// 对于常见情况，使用默认类型：
	// - 如果返回类型是 Result[T, E]，且 E 未被替换，默认 E = string
	// 注意：这里需要更精确的匹配，避免错误替换
	if strings.Contains(returnType, "Result[") {
		// 检查是否包含未替换的 E
		if strings.Contains(returnType, ", E]") {
			returnType = strings.ReplaceAll(returnType, ", E]", ", string]")
		}
		if returnType == "Result[T, E]" {
			returnType = "Result[T, string]"
		}
		// 如果 T 已被替换但 E 未替换，例如 "Result[int, E]"
		if strings.Contains(returnType, ", E]") {
			returnType = strings.ReplaceAll(returnType, ", E]", ", string]")
		}
	}
	
	return returnType, nil
}

// parseGenericType 解析泛型类型
// 例如："Vec[int]" -> ("Vec", ["int"], nil)
// 例如："[int]" -> ("[", ["int"], nil)（切片类型，用于匹配 [T].pop 等）
// 例如："map[string]string" -> ("map", ["string", "string"], nil)
// 例如："HashMap[string, int]" -> ("HashMap", ["string", "int"], nil)
func (s *SimpleTypeInferenceService) parseGenericType(typeStr string) (string, []string, error) {
	// 检查是否有泛型参数（包含 '['）
	if !strings.Contains(typeStr, "[") {
		return typeStr, nil, nil
	}
	
	// 切片类型 "[int]" / "[string]"：以 '[' 开头，用于匹配 [T].pop 等
	if len(typeStr) >= 2 && typeStr[0] == '[' && typeStr[len(typeStr)-1] == ']' {
		inner := strings.TrimSpace(typeStr[1 : len(typeStr)-1])
		if inner != "" {
			parts := strings.Split(inner, ",")
			typeArgs := make([]string, 0, len(parts))
			for _, p := range parts {
				typeArgs = append(typeArgs, strings.TrimSpace(p))
			}
			return "[", typeArgs, nil
		}
	}
	
	// 找到 '[' 的位置
	leftBracket := strings.Index(typeStr, "[")
	baseType := typeStr[:leftBracket]
	
	// 提取类型参数部分："int]" 或 "string, int]" 或 "string]string"（map[K]V）
	inner := typeStr[leftBracket+1:]
	rightBracket := strings.Index(inner, "]")
	if rightBracket < 0 {
		return "", nil, fmt.Errorf("no matching ']' in generic type: %s", typeStr)
	}
	typeArgsStr := inner[:rightBracket]
	afterFirst := strings.TrimSpace(inner[rightBracket+1:])
	
	var typeArgs []string
	if typeArgsStr != "" {
		if strings.Contains(typeArgsStr, ",") {
			// 逗号分隔：HashMap[string, int] 或 map[T, E]
			parts := strings.Split(typeArgsStr, ",")
			for _, part := range parts {
				typeArgs = append(typeArgs, strings.TrimSpace(part))
			}
		} else if baseType == "map" && afterFirst != "" {
			// map[K]V 格式：两个类型参数，无逗号
			typeArgs = append(typeArgs, strings.TrimSpace(typeArgsStr))
			typeArgs = append(typeArgs, strings.TrimSpace(afterFirst))
		} else {
			typeArgs = append(typeArgs, strings.TrimSpace(typeArgsStr))
		}
	}
	
	return baseType, typeArgs, nil
}

// substituteTypeParameters 替换类型参数
// 例如：typeStr = "Option[T]", typeArgs = ["int"] -> "Option[int]"
// 例如：typeStr = "Result[T, E]", typeArgs = ["int", "string"] -> "Result[int, string]"
// 例如：typeStr = "Vec[T]", typeArgs = ["int"] -> "Vec[int]"
func (s *SimpleTypeInferenceService) substituteTypeParameters(typeStr string, typeArgs []string) string {
	// 如果类型参数为空，直接返回
	if len(typeArgs) == 0 {
		return typeStr
	}
	
	// 简单替换策略：
	// 1. 替换 "[T]" -> "[typeArgs[0]]"
	// 2. 替换独立的 "T" -> typeArgs[0]（但要注意不要替换 "Option[T]" 中的 T）
	// 3. 如果有第二个类型参数，替换 "E" -> typeArgs[1]
	
	result := typeStr
	
	// 替换策略：
	// 1. 先替换 "[T]" -> "[typeArgs[0]]"（精确匹配）
	// 2. 再替换独立的 "T"（但要注意边界）
	// 3. 如果有第二个类型参数，替换 "E" -> typeArgs[1]
	// 4. 如果没有第二个类型参数但返回类型包含 "E"，使用默认类型（如 string）
	
	if len(typeArgs) >= 1 {
		// 先替换 "[T]" -> "[typeArgs[0]]"
		result = strings.ReplaceAll(result, "[T]", fmt.Sprintf("[%s]", typeArgs[0]))
		// 再替换独立的 "T"（但要注意边界，避免替换 "Option[T]" 中的 T）
		// 使用更精确的替换：只替换独立的 T（前后不是字母数字）
		result = s.replaceTypeParameter(result, "T", typeArgs[0])
	}
	
	// 替换 "E" -> typeArgs[1]（如果存在）
	if len(typeArgs) >= 2 {
		result = s.replaceTypeParameter(result, "E", typeArgs[1])
	} else {
		// 如果没有第二个类型参数但返回类型包含 "E"，使用默认类型 string
		// 这是为了处理 ok_or 等方法，其中 E 从参数推断（通常是 string）
		if strings.Contains(result, "E") {
			result = s.replaceTypeParameter(result, "E", "string")
		}
	}
	
	return result
}

// replaceTypeParameter 替换类型参数（更精确的替换）
// 只替换独立的类型参数名（前后不是字母数字或下划线）
func (s *SimpleTypeInferenceService) replaceTypeParameter(typeStr, paramName, replacement string) string {
	// 使用正则表达式替换独立的类型参数名
	// 匹配模式：前后不是字母数字下划线的 paramName
	// 例如：在 "Option[T]" 中，T 前后是 [ 和 ]，应该替换
	// 但在 "TypeName" 中，T 前后是字母，不应该替换
	
	// 简单实现：替换所有独立的 paramName
	// 更精确的实现需要使用正则表达式，但为了简单，先使用字符串替换
	// 注意：这可能会错误替换一些情况，但对于标准库类型应该足够
	
	// 替换独立的 paramName（前后是 [、]、,、空格等）
	patterns := []string{
		fmt.Sprintf("[%s]", paramName),     // [T] -> [replacement]
		fmt.Sprintf(",%s,", paramName),      // ,T, -> ,replacement,
		fmt.Sprintf(",%s]", paramName),      // ,T] -> ,replacement]
		fmt.Sprintf("[%s,", paramName),      // [T, -> [replacement,
		fmt.Sprintf(" %s ", paramName),      //  T  ->  replacement
		fmt.Sprintf(" %s,", paramName),      //  T, ->  replacement,
		fmt.Sprintf(" %s]", paramName),      //  T] ->  replacement]
		fmt.Sprintf(",%s ", paramName),     // ,T  -> ,replacement
		fmt.Sprintf("[%s ", paramName),      // [T  -> [replacement
	}
	
	replacements := []string{
		fmt.Sprintf("[%s]", replacement),
		fmt.Sprintf(",%s,", replacement),
		fmt.Sprintf(",%s]", replacement),
		fmt.Sprintf("[%s,", replacement),
		fmt.Sprintf(" %s ", replacement),
		fmt.Sprintf(" %s,", replacement),
		fmt.Sprintf(" %s]", replacement),
		fmt.Sprintf(",%s ", replacement),
		fmt.Sprintf("[%s ", replacement),
	}
	
	result := typeStr
	for i, pattern := range patterns {
		result = strings.ReplaceAll(result, pattern, replacements[i])
	}
	
	return result
}

// inferTypeFromFieldAccess 从结构体字段访问推断类型
// 1. 推断对象类型
// 2. 查找结构体定义，获取字段类型
func (s *SimpleTypeInferenceService) inferTypeFromFieldAccess(expr *entities.StructAccess) (string, error) {
	// 1. 推断对象类型
	objectType, err := s.inferTypeFromExpression(expr.Object)
	if err != nil {
		return "", fmt.Errorf("cannot infer object type for field access '%s': %v", expr.Field, err)
	}
	
	// 2. 查找结构体定义
	fieldMap, exists := s.structTable[objectType]
	if !exists {
		return "", fmt.Errorf("cannot infer type from field access '%s' on type '%s' - struct definition not found", expr.Field, objectType)
	}
	
	// 3. 查找字段类型
	fieldType, exists := fieldMap[expr.Field]
	if !exists {
		return "", fmt.Errorf("cannot infer type from field access '%s' on type '%s' - field not found in struct definition", expr.Field, objectType)
	}
	
	return fieldType, nil
}

// inferTypeFromLenExpr 从长度表达式推断类型
// len() 函数总是返回 int 类型
func (s *SimpleTypeInferenceService) inferTypeFromLenExpr(expr *entities.LenExpr) (string, error) {
	// 验证数组表达式是否有效（可选，但有助于错误提示）
	_, err := s.inferTypeFromExpression(expr.Array)
	if err != nil {
		return "", fmt.Errorf("cannot infer type from len() - invalid array expression: %v", err)
	}
	
	// len() 函数总是返回 int 类型
	return "int", nil
}

// inferTypeFromSizeOfExpr 从 sizeof 表达式推断类型
// sizeof(T) 表达式总是返回 int 类型
func (s *SimpleTypeInferenceService) inferTypeFromSizeOfExpr(expr *entities.SizeOfExpr) (string, error) {
	// 验证类型名称是否有效（可选，但有助于错误提示）
	// 注意：在编译时，类型大小会在 IR 生成阶段计算
	// 这里只验证类型名称格式是否正确
	
	// sizeof(T) 表达式总是返回 int 类型
	return "int", nil
}

// inferTypeFromFunctionPointerExpr 从函数指针表达式推断类型
// &func_name 表达式总是返回 *i8 类型（通用指针）
func (s *SimpleTypeInferenceService) inferTypeFromFunctionPointerExpr(expr *entities.FunctionPointerExpr) (string, error) {
	// 验证函数是否存在（可选，但有助于错误提示）
	// 注意：在编译时，函数指针会在 IR 生成阶段转换为 *i8
	// 这里只验证函数名称格式是否正确
	
	// &func_name 表达式总是返回 *i8 类型（通用指针）
	return "*i8", nil
}

// inferTypeFromAddressOfExpr 从取地址表达式推断类型
// &expression 表达式总是返回 *i8 类型（通用指针）
func (s *SimpleTypeInferenceService) inferTypeFromAddressOfExpr(expr *entities.AddressOfExpr) (string, error) {
	// 验证操作数表达式是否有效（可选，但有助于错误提示）
	_, err := s.inferTypeFromExpression(expr.Operand)
	if err != nil {
		return "", fmt.Errorf("cannot infer type from address-of operand: %w", err)
	}
	// &expression 表达式总是返回 *i8 类型（通用指针）
	return "*i8", nil
}

// inferTypeFromArrayMethodCall 从数组方法调用推断类型
// 支持的数组方法：
// - pop() -> 返回数组元素类型（如 [int] -> int）
// - push(), insert(), remove() -> 返回 void
func (s *SimpleTypeInferenceService) inferTypeFromArrayMethodCall(expr *entities.ArrayMethodCallExpr) (string, error) {
	// 1. 推断数组类型
	arrayType, err := s.inferTypeFromExpression(expr.Array)
	if err != nil {
		return "", fmt.Errorf("cannot infer array type for method call '%s': %v", expr.Method, err)
	}

	// 2. 验证数组类型格式（应该是 [ElementType]）
	if !strings.HasPrefix(arrayType, "[") || !strings.HasSuffix(arrayType, "]") {
		return "", fmt.Errorf("cannot infer type from array method call '%s' - expected array type, got '%s'", expr.Method, arrayType)
	}

	// 3. 提取元素类型（去掉 [ 和 ]）
	elementType := strings.TrimPrefix(strings.TrimSuffix(arrayType, "]"), "[")

	// 4. 根据方法名推断返回类型
	switch expr.Method {
	case "pop":
		// pop() 返回数组元素类型
		return elementType, nil
	case "push", "insert", "remove":
		// push(), insert(), remove() 返回 void（修改操作，不返回值）
		return "void", nil
	default:
		return "", fmt.Errorf("cannot infer type from array method call '%s' - unsupported method", expr.Method)
	}
}

// inferTypeFromResultExpr 从 Result 类型表达式推断类型
// ResultExpr 包含 OkType 和 ErrType，直接返回 Result[OkType, ErrType] 格式
func (s *SimpleTypeInferenceService) inferTypeFromResultExpr(expr *entities.ResultExpr) (string, error) {
	// ResultExpr 已经包含了类型信息，直接格式化返回
	if expr.OkType == "" {
		return "", fmt.Errorf("cannot infer type from ResultExpr - OkType is empty")
	}
	if expr.ErrType == "" {
		return "", fmt.Errorf("cannot infer type from ResultExpr - ErrType is empty")
	}
	return fmt.Sprintf("Result[%s, %s]", expr.OkType, expr.ErrType), nil
}

// inferTypeFromOptionExpr 从 Option 类型表达式推断类型
// OptionExpr 包装一个表达式，需要推断内部表达式的类型，然后返回 Option[InternalType]
func (s *SimpleTypeInferenceService) inferTypeFromOptionExpr(expr *entities.OptionExpr) (string, error) {
	// 推断内部表达式的类型
	internalType, err := s.inferTypeFromExpression(expr.Expr)
	if err != nil {
		return "", fmt.Errorf("cannot infer type from OptionExpr - cannot infer internal expression type: %v", err)
	}
	
	// 返回 Option[InternalType] 格式
	return fmt.Sprintf("Option[%s]", internalType), nil
}

// inferTypeFromOkLiteral 从 Ok 字面量推断类型
// Ok(value) -> Result[ValueType, string]
// 默认错误类型为 string，如果未来需要支持其他错误类型，可以通过上下文推断
func (s *SimpleTypeInferenceService) inferTypeFromOkLiteral(expr *entities.OkLiteral) (string, error) {
	// 推断内部值的类型
	valueType, err := s.inferTypeFromExpression(expr.Value)
	if err != nil {
		return "", fmt.Errorf("cannot infer type from OkLiteral - cannot infer value type: %v", err)
	}
	
	// 返回 Result[ValueType, string] 格式（默认错误类型为 string）
	return fmt.Sprintf("Result[%s, string]", valueType), nil
}

// inferTypeFromErrLiteral 从 Err 字面量推断类型
// Err(error) -> Result[T, ErrorType]
// 注意：OkType (T) 需要从上下文推断，这里暂时返回错误，要求显式类型注解
func (s *SimpleTypeInferenceService) inferTypeFromErrLiteral(expr *entities.ErrLiteral) (string, error) {
	// 推断错误值的类型
	errorType, err := s.inferTypeFromExpression(expr.Error)
	if err != nil {
		return "", fmt.Errorf("cannot infer type from ErrLiteral - cannot infer error type: %v", err)
	}
	
	// 注意：ErrLiteral 无法推断 OkType，需要从上下文（如函数返回类型）推断
	// 这里暂时返回错误，要求显式类型注解
	// 未来可以扩展：从函数返回类型或变量声明类型推断
	return "", fmt.Errorf("cannot infer type from ErrLiteral - OkType requires context (function return type or explicit type annotation). Error type is '%s'", errorType)
}

// inferTypeFromSomeLiteral 从 Some 字面量推断类型
// Some(value) -> Option[ValueType]
func (s *SimpleTypeInferenceService) inferTypeFromSomeLiteral(expr *entities.SomeLiteral) (string, error) {
	// 推断内部值的类型
	valueType, err := s.inferTypeFromExpression(expr.Value)
	if err != nil {
		return "", fmt.Errorf("cannot infer type from SomeLiteral - cannot infer value type: %v", err)
	}
	
	// 返回 Option[ValueType] 格式
	return fmt.Sprintf("Option[%s]", valueType), nil
}

// inferTypeFromNoneLiteral 从 None 字面量推断类型
// None -> Option[T]
// 注意：T 需要从上下文推断，这里暂时返回错误，要求显式类型注解
func (s *SimpleTypeInferenceService) inferTypeFromNoneLiteral(expr *entities.NoneLiteral) (string, error) {
	// None 字面量没有值，无法推断内部类型
	// 需要从上下文（如函数返回类型或变量声明类型）推断
	// 这里暂时返回错误，要求显式类型注解
	return "", fmt.Errorf("cannot infer type from NoneLiteral - internal type requires context (function return type or explicit type annotation)")
}

// inferTypeFromChanLiteral 从 Channel 字面量推断类型
// chan[Type]() -> chan[Type]
func (s *SimpleTypeInferenceService) inferTypeFromChanLiteral(expr *entities.ChanLiteral) (string, error) {
	// Channel 字面量包含元素类型信息
	if expr.Type == "" {
		return "", fmt.Errorf("cannot infer type from ChanLiteral - element type is empty")
	}
	// 返回 chan[ElementType] 格式
	return fmt.Sprintf("chan[%s]", expr.Type), nil
}

// inferTypeFromSendExpr 从发送表达式推断类型
// channel <- value -> void
func (s *SimpleTypeInferenceService) inferTypeFromSendExpr(expr *entities.SendExpr) (string, error) {
	// 验证 Channel 表达式
	channelType, err := s.inferTypeFromExpression(expr.Channel)
	if err != nil {
		return "", fmt.Errorf("cannot infer type from SendExpr - cannot infer channel type: %v", err)
	}
	
	// ✅ 修复：支持 chan[ElementType] 和 chan ElementType 格式
	// 标准化通道类型格式：将 "chan int" 转换为 "chan[int]"
	normalizedType := strings.ReplaceAll(channelType, " ", "")
	
	// 如果格式是 "chanint"（chan 和类型之间没有空格也没有方括号），需要手动添加方括号
	// 例如："chanint" -> "chan[int]"
	if strings.HasPrefix(normalizedType, "chan") && !strings.HasPrefix(normalizedType, "chan[") {
		// 提取类型部分（去掉 "chan" 前缀）
		elementType := normalizedType[4:] // "chan" 长度为 4
		if elementType != "" {
			normalizedType = "chan[" + elementType + "]"
		}
	}
	
	// 验证 Channel 类型格式（应该是 chan[T]）
	if !strings.HasPrefix(normalizedType, "chan[") || !strings.HasSuffix(normalizedType, "]") {
		return "", fmt.Errorf("cannot infer type from SendExpr - invalid channel type: %s", channelType)
	}
	
	// 验证发送值的类型（可选，用于类型检查）
	valueType, err := s.inferTypeFromExpression(expr.Value)
	if err != nil {
		return "", fmt.Errorf("cannot infer type from SendExpr - cannot infer value type: %v", err)
	}
	
	// 提取 Channel 元素类型
	elementType := normalizedType[5 : len(normalizedType)-1] // 提取 "chan[" 和 "]" 之间的内容
	
	// 验证值类型与 Channel 元素类型匹配（可选，用于类型检查）
	if valueType != elementType {
		// 注意：这里可以返回警告，但不阻止类型推断
		// 实际类型检查应该在类型检查阶段进行
	}
	
	// 发送操作返回 void
	return "void", nil
}

// inferTypeFromReceiveExpr 从接收表达式推断类型
// <-channel -> ElementType（从 Channel 类型提取）
func (s *SimpleTypeInferenceService) inferTypeFromReceiveExpr(expr *entities.ReceiveExpr) (string, error) {
	// 推断 Channel 类型
	channelType, err := s.inferTypeFromExpression(expr.Channel)
	if err != nil {
		return "", fmt.Errorf("cannot infer type from ReceiveExpr - cannot infer channel type: %v", err)
	}
	
	// ✅ 修复：支持 chan[ElementType] 和 chan ElementType 格式
	// 标准化通道类型格式：将 "chan int" 转换为 "chan[int]"
	normalizedType := strings.ReplaceAll(channelType, " ", "")
	
	// 如果格式是 "chanint"（chan 和类型之间没有空格也没有方括号），需要手动添加方括号
	// 例如："chanint" -> "chan[int]"
	if strings.HasPrefix(normalizedType, "chan") && !strings.HasPrefix(normalizedType, "chan[") {
		// 提取类型部分（去掉 "chan" 前缀）
		elementType := normalizedType[4:] // "chan" 长度为 4
		if elementType != "" {
			normalizedType = "chan[" + elementType + "]"
		}
	}
	
	// 解析 Channel 类型格式：chan[ElementType]
	if !strings.HasPrefix(normalizedType, "chan[") || !strings.HasSuffix(normalizedType, "]") {
		return "", fmt.Errorf("cannot infer type from ReceiveExpr - invalid channel type format: %s (expected chan[ElementType])", channelType)
	}
	
	// 提取元素类型
	elementType := normalizedType[5 : len(normalizedType)-1] // 提取 "chan[" 和 "]" 之间的内容
	
	if elementType == "" {
		return "", fmt.Errorf("cannot infer type from ReceiveExpr - empty element type in channel: %s", channelType)
	}
	
	// 返回元素类型
	return elementType, nil
}

// inferTypeFromAwaitExpr 从异步等待表达式推断类型
// await future -> T（从 Future[T] 提取）
func (s *SimpleTypeInferenceService) inferTypeFromAwaitExpr(expr *entities.AwaitExpr) (string, error) {
	// 推断异步表达式的类型（应该是 Future[T]）
	futureType, err := s.inferTypeFromExpression(expr.Expression)
	if err != nil {
		return "", fmt.Errorf("cannot infer type from AwaitExpr - cannot infer future type: %v", err)
	}
	
	// 解析 Future 类型格式：Future[ResultType]
	if !strings.HasPrefix(futureType, "Future[") || !strings.HasSuffix(futureType, "]") {
		return "", fmt.Errorf("cannot infer type from AwaitExpr - invalid future type format: %s (expected Future[ResultType])", futureType)
	}
	
	// 提取结果类型
	resultType := futureType[7 : len(futureType)-1] // 提取 "Future[" 和 "]" 之间的内容
	
	if resultType == "" {
		return "", fmt.Errorf("cannot infer type from AwaitExpr - empty result type in future: %s", futureType)
	}
	
	// 返回结果类型
	return resultType, nil
}

// inferTypeFromSpawnExpr 从生成协程表达式推断类型
// spawn function() -> Future[T]（T 是函数返回类型）
func (s *SimpleTypeInferenceService) inferTypeFromSpawnExpr(expr *entities.SpawnExpr) (string, error) {
	// 推断函数的返回类型
	// 如果 Function 是 FuncCall，从符号表查找函数返回类型
	// 如果 Function 是 Identifier，从符号表查找函数类型
	
	var returnType string
	var err error
	
	switch f := expr.Function.(type) {
	case *entities.FuncCall:
		// 从函数调用推断返回类型
		returnType, err = s.inferTypeFromExpression(f)
		if err != nil {
			return "", fmt.Errorf("cannot infer type from SpawnExpr - cannot infer function return type: %v", err)
		}
	case *entities.Identifier:
		// 从标识符查找函数返回类型
		var exists bool
		returnType, exists = s.functionTable[f.Name]
		if !exists {
			return "", fmt.Errorf("cannot infer type from SpawnExpr - function '%s' not found in function table", f.Name)
		}
	default:
		// 尝试推断表达式类型
		returnType, err = s.inferTypeFromExpression(expr.Function)
		if err != nil {
			return "", fmt.Errorf("cannot infer type from SpawnExpr - cannot infer function type: %v", err)
		}
		// 如果推断出的类型不是函数类型，可能需要进一步处理
		// 这里假设推断出的类型就是返回类型（简化处理）
	}
	
	if returnType == "" {
		return "", fmt.Errorf("cannot infer type from SpawnExpr - empty return type")
	}
	
	// 包装为 Future[T] 类型
	return fmt.Sprintf("Future[%s]", returnType), nil
}

// inferTypeFromTernaryExpr 从三元表达式推断类型
// 三元表达式：if condition { thenValue } else { elseValue }
// 类型应该是 then 和 else 分支的公共类型（如果类型相同）
func (s *SimpleTypeInferenceService) inferTypeFromTernaryExpr(expr *entities.TernaryExpr) (string, error) {
	// 推断 then 分支的类型
	thenType, err := s.inferTypeFromExpression(expr.ThenValue)
	if err != nil {
		return "", fmt.Errorf("cannot infer type from ternary expression then branch: %v", err)
	}
	
	// 推断 else 分支的类型
	elseType, err := s.inferTypeFromExpression(expr.ElseValue)
	if err != nil {
		return "", fmt.Errorf("cannot infer type from ternary expression else branch: %v", err)
	}
	
	// 如果类型相同，返回该类型
	if thenType == elseType {
		return thenType, nil
	}
	
	// 如果类型不同，尝试找到公共类型
	// 例如：int 和 int 应该返回 int
	// 例如：int 和 float 可能需要返回更通用的类型（这里简化处理，返回错误）
	// 注意：在实际使用中，可能需要类型转换或更复杂的类型系统
	return "", fmt.Errorf("type mismatch in ternary expression: then branch is %s, else branch is %s", thenType, elseType)
}
