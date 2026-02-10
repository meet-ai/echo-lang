package services

import (
	"context"
	"fmt"

	"echo/internal/modules/frontend/domain/semantic/entities"
	value_objects "echo/internal/modules/frontend/domain/shared/value_objects"
)

// SymbolTableBuilder 负责从 ProgramAST 构建符号表的领域服务
type SymbolTableBuilder struct {
	symbolTableEntity *entities.SymbolTableEntity
}

// NewSymbolTableBuilder 创建符号表构建服务
func NewSymbolTableBuilder(symbolTableEntity *entities.SymbolTableEntity) *SymbolTableBuilder {
	return &SymbolTableBuilder{
		symbolTableEntity: symbolTableEntity,
	}
}

// Build 根据 AST 构建符号表并进行一致性校验
func (b *SymbolTableBuilder) Build(
	ctx context.Context,
	programAST *value_objects.ProgramAST,
	result *value_objects.SemanticAnalysisResult,
) error {
	// 遍历AST，收集所有符号定义
	for _, node := range programAST.Nodes() {
		err := b.visitNodeForSymbols(node, result)
		if err != nil {
			result.AddError(value_objects.NewParseError(
				fmt.Sprintf("failed to build symbol table: %v", err),
				node.Location(),
				value_objects.ErrorTypeSemantic,
				value_objects.SeverityError,
			))
			return err
		}
	}

	// 验证符号表一致性
	err := b.symbolTableEntity.ValidateScopes()
	if err != nil {
		result.AddError(value_objects.NewParseError(
			fmt.Sprintf("symbol table validation failed: %v", err),
			value_objects.NewSourceLocation("", 0, 0, 0),
			value_objects.ErrorTypeSemantic,
			value_objects.SeverityError,
		))
		return err
	}

	return nil
}

// visitNodeForSymbols 访问AST节点，收集符号定义
func (b *SymbolTableBuilder) visitNodeForSymbols(node value_objects.ASTNode, result *value_objects.SemanticAnalysisResult) error {
	switch n := node.(type) {
	case *value_objects.FunctionDeclaration:
		// 定义函数符号
		funcType := value_objects.NewCustomSymbolType(fmt.Sprintf("func(%s)->%s", parametersToString(n.Parameters()), n.ReturnType().String()))
		err := b.symbolTableEntity.DefineSymbol(
			n.Name(),
			value_objects.SymbolKindFunction,
			funcType,
			n.Location(), // 使用节点的位置信息
		)
		if err != nil {
			return err
		}

		// 进入函数作用域
		err = b.symbolTableEntity.EnterScope("function_"+n.Name(), n.Location())
		if err != nil {
			return err
		}

		// 定义参数符号
		for _, param := range n.Parameters() {
			paramType := value_objects.NewCustomSymbolType(param.TypeAnnotation().String())
			err = b.symbolTableEntity.DefineSymbol(
				param.Name(),
				value_objects.SymbolKindParameter,
				paramType,
				value_objects.NewSourceLocation("", 0, 0, 0),
			)
			if err != nil {
				return err
			}
		}

		// 递归处理函数体
		for _, stmt := range n.Body().Statements() {
			err = b.visitNodeForSymbols(stmt, result)
			if err != nil {
				return err
			}
		}

		// 退出函数作用域
		return b.symbolTableEntity.ExitScope()

	case *value_objects.VariableDeclaration:
		// 定义变量符号
		varType := value_objects.NewCustomSymbolType(n.VarType().String())
		err := b.symbolTableEntity.DefineSymbol(
			n.Name(),
			value_objects.SymbolKindVariable,
			varType,
			n.Location(),
		)
		if err != nil {
			return err
		}

		// 如果有初始化表达式，检查其类型（后续可以在 TypeChecker 中完善）
		if n.Initializer() != nil {
			// 这里可以添加类型推断逻辑
		}

		return nil

	default:
		// 对于其他类型的节点，默认不处理
		return nil
	}
}

