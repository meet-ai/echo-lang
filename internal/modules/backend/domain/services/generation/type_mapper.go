package generation

// TypeMapper 类型映射领域服务接口
// 职责：负责Echo语言类型到目标平台类型的映射
type TypeMapper interface {
	// 映射基本类型
	MapPrimitiveType(echoType string) (interface{}, error)

	// 映射类型（支持复杂类型）
	MapType(echoType string) (interface{}, error)

	// 映射数组类型
	MapArrayType(elementType string, dimensions []int) (interface{}, error)

	// 映射结构体类型
	MapStructType(structName string, fields map[string]string) (interface{}, error)

	// 映射函数类型
	MapFunctionType(returnType string, paramTypes []string) (interface{}, error)

	// 检查类型兼容性
	IsCompatible(fromType, toType string) bool

	// 获取类型的默认值
	GetDefaultValue(echoType string) interface{}

	// 注册自定义类型
	RegisterCustomType(name string, mappedType interface{}) error

	// 查找已注册的类型
	LookupType(name string) (interface{}, bool)
}
