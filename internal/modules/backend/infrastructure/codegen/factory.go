package codegen

import (
	"github.com/meetai/echo-lang/internal/modules/backend/domain/services"
)

// NewCodeGenerator 根据类型创建代码生成器
// 这是一个基础设施层的工厂函数
func NewCodeGenerator(generatorType string) services.CodeGenerator {
	switch generatorType {
	case "llvm", "complex-llvm":
		return NewLLVMGenerator() // 使用DDD架构的LLVM生成器
	case "ocaml":
		return NewOCamlGenerator()
	default:
		// 默认使用DDD架构的LLVM生成器
		return NewLLVMGenerator()
	}
}
