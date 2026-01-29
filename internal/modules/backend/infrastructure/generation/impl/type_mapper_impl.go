package impl

import (
	"fmt"
	"strings"

	"github.com/llir/llvm/ir/types"
)

// TypeMapperImpl 类型映射器实现
type TypeMapperImpl struct {
	primitiveTypes map[string]types.Type // Echo类型 -> LLVM类型
	customTypes    map[string]interface{} // 自定义类型注册表
}

// FutureType 表示Future类型，包含元素类型信息
type FutureType struct {
	ElementType interface{} // 元素类型 (string, int, etc.)
}

// ChanType 表示Channel类型，包含元素类型信息
type ChanType struct {
	ElementType interface{} // 元素类型
}

// NewTypeMapperImpl 创建类型映射器实现
func NewTypeMapperImpl() *TypeMapperImpl {
	return &TypeMapperImpl{
		primitiveTypes: map[string]types.Type{
			"int":    types.I32,
			"i32":    types.I32, // i32 别名
			"u64":    types.I64, // u64 映射为 i64（LLVM 中 i64 可以表示无符号整数）
			"i64":    types.I64, // i64 类型
			"bool":   types.I1,
			"string": types.NewPointer(types.I8), // 字符串指针
			"float":  types.Float,
			"f32":    types.Float, // f32 别名
			"f64":    types.Double, // f64 类型
			"void":   types.Void, // 无返回值
		},
		customTypes: make(map[string]interface{}),
	}
}

// MapPrimitiveType 映射基本类型
func (tm *TypeMapperImpl) MapPrimitiveType(echoType string) (interface{}, error) {
	if llvmType, exists := tm.primitiveTypes[echoType]; exists {
		return llvmType, nil
	}

	// 检查是否是数组类型
	if strings.HasPrefix(echoType, "[") && strings.HasSuffix(echoType, "]") {
		elementType := echoType[1 : len(echoType)-1]
		return tm.MapArrayType(elementType, []int{-1}) // -1表示未指定大小
	}

	// 检查是否是自定义类型
	if customType, exists := tm.customTypes[echoType]; exists {
		return customType, nil
	}

	return nil, fmt.Errorf("unsupported primitive type: %s", echoType)
}

// MapArrayType 映射数组类型
func (tm *TypeMapperImpl) MapArrayType(elementType string, dimensions []int) (interface{}, error) {
	elemType, err := tm.MapPrimitiveType(elementType)
	if err != nil {
		return nil, fmt.Errorf("invalid element type %s: %w", elementType, err)
	}

	// 简化实现：只支持一维数组
	if len(dimensions) > 1 {
		return nil, fmt.Errorf("multi-dimensional arrays not supported yet")
	}

	size := dimensions[0]
	if size <= 0 {
		// 动态大小数组，使用指针
		return fmt.Sprintf("[%s]*", elemType), nil
	}

	// 静态大小数组
	return fmt.Sprintf("[%d x %s]", size, elemType), nil
}

// MapStructType 映射结构体类型
func (tm *TypeMapperImpl) MapStructType(structName string, fields map[string]string) (interface{}, error) {
	// 简化实现：只生成结构体类型定义
	fieldTypes := make([]string, 0, len(fields))
	for _, fieldType := range fields {
		mappedType, err := tm.MapPrimitiveType(fieldType)
		if err != nil {
			return nil, fmt.Errorf("invalid field type %s: %w", fieldType, err)
		}
		fieldTypes = append(fieldTypes, fmt.Sprintf("%v", mappedType))
	}

	structDef := fmt.Sprintf("%%%s = type { %s }", structName, strings.Join(fieldTypes, ", "))
	return structDef, nil
}

// MapFunctionType 映射函数类型
func (tm *TypeMapperImpl) MapFunctionType(returnType string, paramTypes []string) (interface{}, error) {
	// 映射返回类型
	retType, err := tm.MapPrimitiveType(returnType)
	if err != nil {
		return nil, fmt.Errorf("invalid return type %s: %w", returnType, err)
	}

	// 映射参数类型
	paramTypeStrings := make([]string, len(paramTypes))
	for i, paramType := range paramTypes {
		mappedType, err := tm.MapPrimitiveType(paramType)
		if err != nil {
			return nil, fmt.Errorf("invalid parameter type %s: %w", paramType, err)
		}
		paramTypeStrings[i] = fmt.Sprintf("%v", mappedType)
	}

	// 生成函数类型
	funcType := fmt.Sprintf("%v (%s)", retType, strings.Join(paramTypeStrings, ", "))
	return funcType, nil
}

// IsCompatible 检查类型兼容性
func (tm *TypeMapperImpl) IsCompatible(fromType, toType string) bool {
	// 简化实现：只有相同类型才兼容
	fromMapped, err1 := tm.MapPrimitiveType(fromType)
	toMapped, err2 := tm.MapPrimitiveType(toType)
	if err1 != nil || err2 != nil {
		return false
	}
	return fmt.Sprintf("%v", fromMapped) == fmt.Sprintf("%v", toMapped)
}

// GetDefaultValue 获取类型的默认值
func (tm *TypeMapperImpl) GetDefaultValue(echoType string) interface{} {
	switch echoType {
	case "int":
		return "0"
	case "bool":
		return "false"
	case "string":
		return `""`
	case "float":
		return "0.0"
	default:
		return "zeroinitializer"
	}
}

// RegisterCustomType 注册自定义类型
func (tm *TypeMapperImpl) RegisterCustomType(name string, mappedType interface{}) error {
	if _, exists := tm.customTypes[name]; exists {
		return fmt.Errorf("custom type %s already registered", name)
	}
	tm.customTypes[name] = mappedType
	return nil
}

// LookupType 查找已注册的类型
func (tm *TypeMapperImpl) LookupType(name string) (interface{}, bool) {
	// 先查找自定义类型
	if customType, exists := tm.customTypes[name]; exists {
		return customType, true
	}

	// 再查找基本类型
	if primitiveType, exists := tm.primitiveTypes[name]; exists {
		return primitiveType, true
	}

	return nil, false
}

// MapType 映射Echo类型到LLVM类型，支持基本类型、数组、Future和Channel
func (tm *TypeMapperImpl) MapType(echoType string) (interface{}, error) {
	// 1. 尝试映射基本类型
	if llvmType, exists := tm.primitiveTypes[echoType]; exists {
		return llvmType, nil
	}

	// 2. 检查是否是数组/切片类型
	// 支持两种格式：[T]（单括号，如 [int]）和 []T（双括号，如 []int）
	// 单括号格式 [T]：Echo 语法 let arr: [int] = ...
	if len(echoType) >= 3 && echoType[0] == '[' && echoType[len(echoType)-1] == ']' && (len(echoType) < 2 || echoType[1] != ']') {
		// 格式 [T]，提取元素类型 T 并校验，切片在运行时为 *i8
		elementTypeStr := echoType[1 : len(echoType)-1]
		_, err := tm.MapType(elementTypeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid array element type %s: %w", elementTypeStr, err)
		}
		return types.NewPointer(types.I8), nil
	}
	// 对于 []T 格式，直接返回 *i8（运行时切片指针）
	if echoType == "[]T" {
		// 直接返回 *i8，因为 []T 在运行时都是 *i8 指针
		return types.NewPointer(types.I8), nil
	}
	// 特殊处理：[]i8* 类型（空数组字面量推断出的类型）
	if echoType == "[]i8*" {
		// 空数组字面量推断出的类型，在运行时是 *i8 指针
		return types.NewPointer(types.I8), nil
	}
	// 检查是否是切片类型 []T（必须以 [] 开头）
	// 注意：切片类型格式是 []T，其中 T 是元素类型
	// 例如：[]string, []int, []T
	// 注意：这个检查必须在其他包含 [ 的类型检查之前
	if len(echoType) >= 2 && strings.HasPrefix(echoType, "[]") {
		// 对于动态大小数组（切片），在运行时都是 *i8 指针（运行时切片指针）
		// 这是合理的，因为运行时切片不依赖于元素类型，所有切片在 LLVM IR 中都是 *i8
		// 例如：[]string, []int, []T 都映射为 *i8
		// 注意：这里不尝试映射元素类型，因为运行时切片类型统一为 *i8
		return types.NewPointer(types.I8), nil
	}

	// 3. 检查是否是Future类型 (Future[T])
	if strings.HasPrefix(echoType, "Future[") && strings.HasSuffix(echoType, "]") {
		// 解析元素类型
		elementTypeStr := echoType[7 : len(echoType)-1] // 移除 "Future[" 和 "]"
		elementType, err := tm.MapType(elementTypeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid Future element type %s: %w", elementTypeStr, err)
		}

		// 返回结构化的Future类型（运行时仍然是i8*指针，但在编译时保留类型信息）
		return &FutureType{ElementType: elementType}, nil
	}

	// 4. 检查是否是Chan类型 (chan T)
	if strings.HasPrefix(echoType, "chan ") {
		// 解析元素类型
		elementTypeStr := strings.TrimSpace(echoType[5:]) // 移除 "chan "
		elementType, err := tm.MapType(elementTypeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid channel element type %s: %w", elementTypeStr, err)
		}

		// 返回结构化的Channel类型（运行时仍然是i8*指针，但在编译时保留类型信息）
		return &ChanType{ElementType: elementType}, nil
	}

	// 5. 检查是否是Map类型 (map[K]V)
	if strings.HasPrefix(echoType, "map[") {
		// 解析键类型和值类型：map[K]V
		// 例如：map[string]string -> keyType="string", valueType="string"
		// 注意：需要找到第一个"]"的位置，然后获取后面的值类型
		bracketEnd := strings.Index(echoType, "]")
		if bracketEnd == -1 {
			return nil, fmt.Errorf("invalid map type syntax: missing ']' in %s", echoType)
		}
		keyTypeStr := echoType[4:bracketEnd] // 移除 "map["，获取键类型
		valueTypeStr := strings.TrimSpace(echoType[bracketEnd+1:]) // 获取"]"之后的值类型
		
		if valueTypeStr == "" {
			return nil, fmt.Errorf("invalid map type syntax: missing value type in %s", echoType)
		}
		
		// 映射键类型和值类型（用于类型检查，但运行时map都是i8*指针）
		_, err := tm.MapType(keyTypeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid map key type %s: %w", keyTypeStr, err)
		}
		_, err = tm.MapType(valueTypeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid map value type %s: %w", valueTypeStr, err)
		}
		
		// map类型在LLVM IR中映射为i8*（不透明指针）
		// 实际的map结构由运行时管理（hash_table_t）
		return types.NewPointer(types.I8), nil
	}

	// 6. 检查是否是Result类型 (Result[T, E])
	// 统一表示：{ i8 tag, i8* data }，tag 0=Ok 1=Err，data 指向 payload；对外使用指针 *struct。
	if strings.HasPrefix(echoType, "Result[") && strings.HasSuffix(echoType, "]") {
		// 解析 T,E 仅用于类型检查，不改变运行时结构
		inner := echoType[7 : len(echoType)-1]
		if strings.Index(inner, ",") >= 0 {
			_ = inner // 可在此解析 T,E 用于文档或后续扩展
		}
		resultStructType := types.NewStruct(types.I8, types.NewPointer(types.I8))
		return types.NewPointer(resultStructType), nil
	}

	// 7. 检查是否是Vec类型 (Vec[T])
	if strings.HasPrefix(echoType, "Vec[") && strings.HasSuffix(echoType, "]") {
		// 解析元素类型
		elementTypeStr := echoType[4 : len(echoType)-1] // 移除 "Vec[" 和 "]"
		
		// 映射元素类型（用于类型检查，但运行时Vec都是相同的结构）
		// 注意：如果元素类型是类型参数（如 T），无法直接映射，但这是允许的
		// 因为 Vec[T] 的结构不依赖于 T 的具体类型
		_, err := tm.MapType(elementTypeStr)
		if err != nil {
			// 如果元素类型无法映射，可能是类型参数（如 T）
			// 这是允许的，因为 Vec[T] 的结构不依赖于 T 的具体类型
			// 我们仍然可以返回 Vec 的结构体类型
		}
		
		// Vec[T] 在 LLVM IR 中映射为结构体：{ i8*, i32, i32 }
		// i8*: data (指向运行时切片的指针)
		// i32: len
		// i32: cap
		return types.NewStruct(
			types.NewPointer(types.I8), // data: *i8 (运行时切片指针)
			types.I32,                  // len: i32
			types.I32,                  // cap: i32
		), nil
	}

	// 8. 检查是否是Option类型 (Option[T])
	if strings.HasPrefix(echoType, "Option[") && strings.HasSuffix(echoType, "]") {
		// 解析元素类型
		elementTypeStr := echoType[7 : len(echoType)-1] // 移除 "Option[" 和 "]"
		
		// 映射元素类型（用于类型检查，但运行时Option都是i8*指针）
		// 注意：如果元素类型是类型参数（如 T），无法直接映射，但这是允许的
		// 因为 Option[T] 在运行时都是 i8* 指针
		_, err := tm.MapType(elementTypeStr)
		if err != nil {
			// 如果元素类型无法映射，可能是类型参数（如 T）
			// 这是允许的，因为 Option[T] 在运行时都是 i8* 指针
		}
		
		// Option[T] 在 LLVM IR 中映射为 i8*（不透明指针）
		// 实际的 Option 结构由运行时管理
		return types.NewPointer(types.I8), nil
	}

	// 9. 检查是否是HashMap类型 (HashMap[K, V])
	if strings.HasPrefix(echoType, "HashMap[") && strings.HasSuffix(echoType, "]") {
		// 解析键类型和值类型：HashMap[K, V]
		// 例如：HashMap[int, string] -> keyType="int", valueType="string"
		// 注意：需要找到第一个","的位置，然后获取键类型和值类型
		innerType := echoType[8 : len(echoType)-1] // 移除 "HashMap[" 和 "]"
		
		// 查找逗号分隔符（注意：类型参数可能包含嵌套的泛型，需要正确处理）
		// 简化实现：假设类型参数不包含嵌套泛型，直接按逗号分割
		commaIndex := strings.Index(innerType, ",")
		if commaIndex == -1 {
			return nil, fmt.Errorf("invalid HashMap type syntax: missing ',' in %s", echoType)
		}
		
		keyTypeStr := strings.TrimSpace(innerType[:commaIndex])
		valueTypeStr := strings.TrimSpace(innerType[commaIndex+1:])
		
		if keyTypeStr == "" || valueTypeStr == "" {
			return nil, fmt.Errorf("invalid HashMap type syntax: missing key or value type in %s", echoType)
		}
		
		// 映射键类型和值类型（用于类型检查，但运行时HashMap都是i8*指针）
		// 注意：如果键类型或值类型是类型参数（如 K, V），无法直接映射，但这是允许的
		// 因为 HashMap[K, V] 在运行时都是 i8* 指针（类似map[K]V）
		_, err := tm.MapType(keyTypeStr)
		if err != nil {
			// 如果键类型无法映射，可能是类型参数（如 K）
			// 这是允许的，因为 HashMap[K, V] 在运行时都是 i8* 指针
		}
		_, err = tm.MapType(valueTypeStr)
		if err != nil {
			// 如果值类型无法映射，可能是类型参数（如 V）
			// 这是允许的，因为 HashMap[K, V] 在运行时都是 i8* 指针
		}
		
		// HashMap[K, V] 在 LLVM IR 中映射为 i8*（不透明指针）
		// 实际的HashMap结构由运行时管理（hash_table_t）
		// 注意：HashMap[K, V] 和 map[K]V 在运行时使用相同的底层结构
		return types.NewPointer(types.I8), nil
	}

	// 10. 检查是否是泛型结构体类型 (如 Bucket[K, V], Entry[K, V])
	// 格式：StructName[TypeParam1, TypeParam2, ...]
	// 注意：这个检查必须在切片类型检查之后，且需要确保不是切片类型（切片类型以 [ 开头）
	if strings.Contains(echoType, "[") && strings.HasSuffix(echoType, "]") && !strings.HasPrefix(echoType, "[") {
		// 查找第一个 "[" 的位置
		bracketStart := strings.Index(echoType, "[")
		if bracketStart > 0 {
			structName := echoType[:bracketStart]
			innerType := echoType[bracketStart+1 : len(echoType)-1] // 移除 "[" 和 "]"
			
			// 检查是否是已知的泛型结构体类型
			// 对于泛型结构体，在 LLVM IR 中映射为 i8*（不透明指针）
			// 因为泛型结构体的具体结构在运行时确定
			switch structName {
			case "Bucket", "Entry":
				// Bucket[K, V] 和 Entry[K, V] 在运行时都是 i8* 指针
				// 解析类型参数（用于类型检查，但不影响运行时类型）
				// 注意：类型参数可能包含逗号，需要正确处理
				// 简化实现：假设类型参数不包含嵌套泛型，直接按逗号分割
				typeParams := strings.Split(innerType, ",")
				for _, param := range typeParams {
					param = strings.TrimSpace(param)
					// 尝试映射类型参数（如果失败，可能是类型参数，这是允许的）
					_, _ = tm.MapType(param)
				}
				return types.NewPointer(types.I8), nil
			default:
				// 其他泛型结构体类型，也映射为 i8* 指针
				// 解析类型参数（用于类型检查）
				typeParams := strings.Split(innerType, ",")
				for _, param := range typeParams {
					param = strings.TrimSpace(param)
					// 尝试映射类型参数（如果失败，可能是类型参数，这是允许的）
					_, _ = tm.MapType(param)
				}
				return types.NewPointer(types.I8), nil
			}
		}
	}

	// 11. 检查是否是类型参数（单个标识符，如 K, V, T）
	// 类型参数通常是单个大写字母或标识符
	// 在 LLVM IR 中，类型参数映射为 i8*（不透明指针）
	// 因为类型参数的具体类型在运行时确定
	if len(echoType) > 0 && (len(echoType) == 1 || (len(echoType) > 1 && echoType[0] >= 'A' && echoType[0] <= 'Z')) {
		// 检查是否是单个字母或标识符（类型参数通常是大写字母开头）
		// 注意：这是一个启发式判断，可能不够准确，但对于常见的类型参数（K, V, T）是有效的
		isTypeParam := true
		for _, char := range echoType {
			if !((char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_') {
				isTypeParam = false
				break
			}
		}
		// 如果看起来像类型参数，且不是已注册的类型，则作为类型参数处理
		if isTypeParam && echoType[0] >= 'A' && echoType[0] <= 'Z' {
			// 检查是否不是已注册的类型
			if _, exists := tm.primitiveTypes[echoType]; !exists {
				if _, exists := tm.customTypes[echoType]; !exists {
					// 类型参数在 LLVM IR 中映射为 i8*（不透明指针）
					return types.NewPointer(types.I8), nil
				}
			}
		}
	}

	// 12. 检查是否是自定义类型
	if customType, exists := tm.customTypes[echoType]; exists {
		return customType, nil
	}

	return nil, fmt.Errorf("unsupported type: %s", echoType)
}
