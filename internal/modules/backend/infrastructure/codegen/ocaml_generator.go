package codegen

import (
	"echo/internal/modules/backend/domain/services"
	"echo/internal/modules/frontend/domain/entities"
	"strings"
)

// OCamlGenerator OCaml代码生成器实现（基础设施层）
type OCamlGenerator struct{}

// CGenerator C代码生成器实现（基础设施层，从历史版本恢复）
type CGenerator struct {
	stringLiterals []string
	stringIndexMap map[string]int
}

// NewOCamlGenerator 创建新的OCaml代码生成器
func NewOCamlGenerator() services.CodeGenerator {
	return &OCamlGenerator{}
}

// GenerateCode 生成OCaml代码（实现CodeGenerator接口）
func (g *OCamlGenerator) GenerateCode(program *entities.Program) string {
	var ocamlCode strings.Builder

	// OCaml 头部
	ocamlCode.WriteString("(* Generated OCaml code from .eo file *)\n\n")

	// 基本结构
	ocamlCode.WriteString("let main () =\n")
	ocamlCode.WriteString("  print_endline \"Hello from Echo!\"\n\n")

	// 简化实现
	return "(* Trait definition would go here *)\n"
}

// GenerateExecutable 生成可执行的OCaml程序（实现CodeGenerator接口）
func (g *OCamlGenerator) GenerateExecutable(program *entities.Program) string {
	return g.GenerateCode(program)
}
