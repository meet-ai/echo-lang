package services

import (
	"context"
	"fmt"

	"echo/internal/modules/frontend/domain/semantic/entities"
	value_objects "echo/internal/modules/frontend/domain/shared/value_objects"
)

// SymbolResolver 负责对 AST 中的标识符进行符号解析
type SymbolResolver struct {
	symbolTableEntity *entities.SymbolTableEntity
}

// NewSymbolResolver 创建符号解析服务
func NewSymbolResolver(symbolTableEntity *entities.SymbolTableEntity) *SymbolResolver {
	return &SymbolResolver{
		symbolTableEntity: symbolTableEntity,
	}
}

// Resolve 遍历 AST，解析所有符号引用
func (r *SymbolResolver) Resolve(
	ctx context.Context,
	programAST *value_objects.ProgramAST,
	result *value_objects.SemanticAnalysisResult,
) error {
	symbolResolver := result.ResolvedSymbols()

	for _, node := range programAST.Nodes() {
		if err := r.resolveNodeSymbols(node, symbolResolver, result); err != nil {
			result.AddError(value_objects.NewParseError(
				fmt.Sprintf("symbol resolution failed: %v", err),
				node.Location(),
				value_objects.ErrorTypeSymbol,
				value_objects.SeverityError,
			))
			return err
		}
	}

	return nil
}

// resolveNodeSymbols 解析节点中的符号引用
func (r *SymbolResolver) resolveNodeSymbols(
	node value_objects.ASTNode,
	symbolResolver *value_objects.ResolvedSymbols,
	result *value_objects.SemanticAnalysisResult,
) error {
	switch n := node.(type) {
	case *value_objects.Identifier:
		// 解析标识符引用，支持作用域查找
		symbolName := n.Name()
		symbol, found := r.symbolTableEntity.ResolveSymbol(symbolName)
		if !found {
			// 尝试在不同作用域中查找符号
			symbol, found = r.lookupSymbolInScopes(symbolName)
			if !found {
				result.AddError(value_objects.NewParseError(
					fmt.Sprintf("undefined symbol: %s", symbolName),
					n.Location(),
					value_objects.ErrorTypeSymbol,
					value_objects.SeverityError,
				))
				return nil
			}
		}

		// 记录已解析的符号
		resolvedSymbol := value_objects.NewResolvedSymbol(symbol, n)
		symbolResolver.AddResolvedSymbol(resolvedSymbol)

		return nil

	case *value_objects.BinaryExpression:
		// 递归解析子表达式
		if err := r.resolveNodeSymbols(n.Left(), symbolResolver, result); err != nil {
			return err
		}
		return r.resolveNodeSymbols(n.Right(), symbolResolver, result)

	case *value_objects.UnaryExpression:
		// 解析一元表达式
		return r.resolveNodeSymbols(n.Operand(), symbolResolver, result)

	default:
		// 其他节点类型在需要时再扩展
		return nil
	}
}

// lookupSymbolInScopes 在作用域中查找符号
func (r *SymbolResolver) lookupSymbolInScopes(name string) (*value_objects.Symbol, bool) {
	// 目前直接委托给 SymbolTableEntity 的 ResolveSymbol（其内部已处理作用域）
	symbol, found := r.symbolTableEntity.ResolveSymbol(name)
	if found {
		return symbol, true
	}

	// 为未来扩展预留：可在此增加全局作用域 / 导入符号等策略
	return nil, false
}

