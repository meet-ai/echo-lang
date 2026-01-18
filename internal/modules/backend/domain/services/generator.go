package services

import (
	"github.com/meetai/echo-lang/internal/modules/frontend/domain/entities"
)

// CodeGenerator 代码生成器接口（领域层定义纯接口）
// 注意：这里不包含任何技术实现细节
type CodeGenerator interface {
	// GenerateCode 生成代码（目标格式由实现决定）
	GenerateCode(program *entities.Program) string

	// GenerateExecutable 生成可执行程序
	GenerateExecutable(program *entities.Program) string
}
