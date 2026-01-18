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

// NewTypeMapperImpl 创建类型映射器实现
func NewTypeMapperImpl() *TypeMapperImpl {
	return &TypeMapperImpl{
		primitiveTypes: map[string]types.Type{
			"int":    types.I32,
			"bool":   types.I1,
			"string": types.NewPointer(types.I8), // 字符串指针
			"float":  types.Float,
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

	// 2. 检查是否是数组类型
	if strings.HasPrefix(echoType, "[") && strings.HasSuffix(echoType, "]") {
		elementType := echoType[1 : len(echoType)-1]
		// 对于动态大小数组，我们通常返回一个指向元素类型的指针
		// 或者一个包含指针和长度的结构体。这里简化为指针。
		mappedElementType, err := tm.MapType(elementType)
		if err != nil {
			return nil, fmt.Errorf("invalid array element type %s: %w", elementType, err)
		}
		if llvmType, ok := mappedElementType.(types.Type); ok {
			return types.NewPointer(llvmType), nil // 返回指向元素类型的指针
		}
		return nil, fmt.Errorf("unsupported array element type: %v", mappedElementType)
	}

	// 3. 检查是否是Future类型 (Future[T])
	if strings.HasPrefix(echoType, "Future[") && strings.HasSuffix(echoType, "]") {
		// Future在运行时层面是一个不透明的指针 (i8*)
		// 无论T是什么类型，在运行时都统一为i8*指针
		return types.NewPointer(types.I8), nil
	}

	// 4. 检查是否是Chan类型 (chan T)
	if strings.HasPrefix(echoType, "chan ") {
		// Channel在运行时层面也是一个不透明的指针 (i8*)
		return types.NewPointer(types.I8), nil
	}

	// 5. 检查是否是Result类型 (Result[T, E])
	if strings.HasPrefix(echoType, "Result[") && strings.HasSuffix(echoType, "]") {
		// Result类型在LLVM IR层面可以映射为一个包含两个字段的结构体
		// 例如：{ i1 is_err, i8* value_ptr, i8* error_ptr }
		// 这里简化为返回一个不透明的指针 (i8*)，实际处理在运行时库中
		return types.NewPointer(types.I8), nil
	}

	// 6. 检查是否是自定义类型
	if customType, exists := tm.customTypes[echoType]; exists {
		return customType, nil
	}

	return nil, fmt.Errorf("unsupported type: %s", echoType)
}
