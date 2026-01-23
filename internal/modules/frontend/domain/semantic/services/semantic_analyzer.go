// Package services 定义语义分析的领域服务
package services

import (
	"context"
	"fmt"
	"strings"

	"echo/internal/modules/frontend/domain/semantic/entities"
	"echo/internal/modules/frontend/domain/shared/value_objects"
)

// SemanticAnalyzer 语义分析服务
type SemanticAnalyzer struct {
	symbolTableEntity *entities.SymbolTableEntity
	currentFunctionReturnType value_objects.SymbolType // 当前函数的返回类型（用于return语句检查）
}

// NewSemanticAnalyzer 创建新的语义分析器
func NewSemanticAnalyzer(symbolTableID string) *SemanticAnalyzer {
	return &SemanticAnalyzer{
		symbolTableEntity: entities.NewSymbolTableEntity(symbolTableID),
	}
}

// AnalyzeSemantics 分析AST的语义正确性
func (sa *SemanticAnalyzer) AnalyzeSemantics(
	ctx context.Context,
	programAST *value_objects.ProgramAST,
) (*value_objects.SemanticAnalysisResult, error) {

	result := value_objects.NewSemanticAnalysisResult(sa.symbolTableEntity.SymbolTable())

	// 第一阶段：构建符号表
	err := sa.buildSymbolTable(ctx, programAST, result)
	if err != nil {
		return result, err
	}

	// 第二阶段：类型检查
	err = sa.performTypeChecking(ctx, programAST, result)
	if err != nil {
		return result, err
	}

	// 第三阶段：符号解析
	err = sa.performSymbolResolution(ctx, programAST, result)
	if err != nil {
		return result, err
	}

	return result, nil
}

// buildSymbolTable 构建符号表
func (sa *SemanticAnalyzer) buildSymbolTable(
	ctx context.Context,
	programAST *value_objects.ProgramAST,
	result *value_objects.SemanticAnalysisResult,
) error {

	// 遍历AST，收集所有符号定义
	for _, node := range programAST.Nodes() {
		err := sa.visitNodeForSymbols(node, result)
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
	err := sa.symbolTableEntity.ValidateScopes()
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
func (sa *SemanticAnalyzer) visitNodeForSymbols(node value_objects.ASTNode, result *value_objects.SemanticAnalysisResult) error {
	switch n := node.(type) {
	case *value_objects.FunctionDeclaration:
		// 定义函数符号
		funcType := value_objects.NewCustomSymbolType(fmt.Sprintf("func(%s)->%s", parametersToString(n.Parameters()), n.ReturnType().String()))
		err := sa.symbolTableEntity.DefineSymbol(
			n.Name(),
			value_objects.SymbolKindFunction,
			funcType,
			n.Location(), // 使用节点的位置信息
		)
		if err != nil {
			return err
		}

		// 进入函数作用域
		err = sa.symbolTableEntity.EnterScope("function_"+n.Name(), n.Location())
		if err != nil {
			return err
		}

		// 定义参数符号
		for _, param := range n.Parameters() {
			paramType := value_objects.NewCustomSymbolType(param.TypeAnnotation().String())
			err = sa.symbolTableEntity.DefineSymbol(
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
			err = sa.visitNodeForSymbols(stmt, result)
			if err != nil {
				return err
			}
		}

		// 退出函数作用域
		return sa.symbolTableEntity.ExitScope()

	case *value_objects.VariableDeclaration:
		// 定义变量符号
		varType := value_objects.NewCustomSymbolType(n.VarType().String())
		err := sa.symbolTableEntity.DefineSymbol(
			n.Name(),
			value_objects.SymbolKindVariable,
			varType,
			n.Location(),
		)
		if err != nil {
			return err
		}

		// 如果有初始化表达式，检查其类型
		if n.Initializer != nil {
			// 这里可以添加类型推断逻辑
		}

		return nil

	default:
		// 对于其他类型的节点，默认不处理
		return nil
	}
}

// performTypeChecking 执行类型检查
func (sa *SemanticAnalyzer) performTypeChecking(
	ctx context.Context,
	programAST *value_objects.ProgramAST,
	result *value_objects.SemanticAnalysisResult,
) error {

	typeChecker := result.TypeCheckResult()

	// 遍历AST，进行类型检查
	for _, node := range programAST.Nodes() {
		err := sa.checkNodeTypes(node, typeChecker)
		if err != nil {
			typeChecker.AddError(value_objects.NewParseError(
				fmt.Sprintf("type check failed: %v", err),
				node.Location(),
				value_objects.ErrorTypeType,
				value_objects.SeverityError,
			))
			return err
		}
	}

	return nil
}

// checkNodeTypes 检查节点的类型正确性
func (sa *SemanticAnalyzer) checkNodeTypes(node value_objects.ASTNode, typeChecker *value_objects.TypeCheckResult) error {
	switch n := node.(type) {
	case *value_objects.FunctionDeclaration:
		// 保存之前的函数上下文
		previousReturnType := sa.currentFunctionReturnType
		
		// 设置当前函数的返回类型上下文
		if n.ReturnType() != nil {
			sa.currentFunctionReturnType = value_objects.NewCustomSymbolType(n.ReturnType().String())
		} else {
			// 无返回类型，视为void
			sa.currentFunctionReturnType = value_objects.NewPrimitiveSymbolType("void")
		}
		
		// 检查函数体中的类型
		if n.Body() != nil {
			for _, stmt := range n.Body().Statements() {
				err := sa.checkNodeTypes(stmt, typeChecker)
				if err != nil {
					// 恢复之前的函数上下文
					sa.currentFunctionReturnType = previousReturnType
					return err
				}
			}
		}
		
		// 恢复函数上下文（退出函数作用域）
		sa.currentFunctionReturnType = previousReturnType
		return nil
		
	case *value_objects.VariableDeclaration:
		// 检查变量初始化表达式的类型是否与声明类型匹配
		if n.Initializer() != nil {
			initializerType := sa.inferExpressionType(n.Initializer())
			declaredType := value_objects.NewCustomSymbolType(n.VarType().String())

			if !sa.typesCompatible(initializerType, declaredType) {
				typeChecker.AddError(value_objects.NewParseError(
					fmt.Sprintf("type mismatch: cannot assign %s to %s", initializerType.String(), declaredType.String()),
					n.Location(),
					value_objects.ErrorTypeType,
					value_objects.SeverityError,
				))
			}
		}
		return nil

	case *value_objects.BinaryExpression:
		// 检查二元运算的类型兼容性
		leftType := sa.inferExpressionType(n.Left())
		rightType := sa.inferExpressionType(n.Right())

		if !sa.canApplyOperator(n.Operator(), leftType, rightType) {
			typeChecker.AddError(value_objects.NewParseError(
				fmt.Sprintf("invalid operation: %s %s %s", leftType.String(), n.Operator(), rightType.String()),
				n.Location(),
				value_objects.ErrorTypeType,
				value_objects.SeverityError,
			))
		}
		return nil

	case *value_objects.ReturnStatement:
		// 完善实现：检查返回值的类型是否与函数返回类型匹配
		if n.Value != nil {
			// 推断返回值的类型
			returnType := sa.inferExpressionType(n.Value)
			
			// 如果当前函数有返回类型，检查是否匹配
			if sa.currentFunctionReturnType != nil {
				// 检查类型兼容性（支持隐式类型转换）
				if !sa.typesCompatible(returnType, sa.currentFunctionReturnType) {
					// 检查是否可以进行隐式类型转换（如 int -> float）
					if !sa.canImplicitlyConvert(returnType, sa.currentFunctionReturnType) {
						typeChecker.AddError(value_objects.NewParseError(
							fmt.Sprintf("return type mismatch: expected %s, got %s", 
								sa.currentFunctionReturnType.String(), returnType.String()),
							n.Location(),
							value_objects.ErrorTypeType,
							value_objects.SeverityError,
						))
					}
				}
			} else {
				// 如果没有当前函数上下文，可能是全局return（错误情况）
				typeChecker.AddError(value_objects.NewParseError(
					"return statement outside of function",
					n.Location(),
					value_objects.ErrorTypeSemantic,
					value_objects.SeverityError,
				))
			}
		} else {
			// 无返回值，检查函数是否期望void返回类型
			if sa.currentFunctionReturnType != nil {
				// 检查返回类型是否为void
				if sa.currentFunctionReturnType.TypeName() != "void" {
					typeChecker.AddError(value_objects.NewParseError(
						fmt.Sprintf("function expects return value of type %s, but return statement has no value", 
							sa.currentFunctionReturnType.String()),
						n.Location(),
						value_objects.ErrorTypeType,
						value_objects.SeverityError,
					))
				}
			} else {
				// 如果没有当前函数上下文，可能是全局return（错误情况）
				typeChecker.AddError(value_objects.NewParseError(
					"return statement outside of function",
					n.Location(),
					value_objects.ErrorTypeSemantic,
					value_objects.SeverityError,
				))
			}
		}
		return nil

	default:
		return nil
	}
}

// performSymbolResolution 执行符号解析
func (sa *SemanticAnalyzer) performSymbolResolution(
	ctx context.Context,
	programAST *value_objects.ProgramAST,
	result *value_objects.SemanticAnalysisResult,
) error {

	symbolResolver := result.ResolvedSymbols()

	// 遍历AST，解析所有符号引用
	for _, node := range programAST.Nodes() {
		err := sa.resolveNodeSymbols(node, symbolResolver)
		if err != nil {
			symbolResolver.AddError(value_objects.NewParseError(
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
func (sa *SemanticAnalyzer) resolveNodeSymbols(node value_objects.ASTNode, symbolResolver *value_objects.ResolvedSymbols) error {
	switch n := node.(type) {
	case *value_objects.Identifier:
		// 完善实现：解析标识符引用，支持作用域查找
		symbol, found := sa.symbolTableEntity.ResolveSymbol(n.Name)
		if !found {
			// 尝试在不同作用域中查找符号
			// 首先在当前作用域查找，然后向上查找父作用域
			symbol, found = sa.lookupSymbolInScopes(n.Name)
			if !found {
				symbolResolver.AddError(value_objects.NewParseError(
					fmt.Sprintf("undefined symbol: %s", n.Name),
					n.Location(),
					value_objects.ErrorTypeSymbol,
					value_objects.SeverityError,
				))
				return nil
			}
		}
		
		// 记录已解析的符号
		// 创建ResolvedSymbol并添加到结果中
		resolvedSymbol := value_objects.NewResolvedSymbol(symbol, n)
		symbolResolver.AddResolvedSymbol(resolvedSymbol)
		
		// 验证符号是否在正确的作用域中使用
		// 例如：检查是否在定义之前使用（如果语言要求）
		// 这里可以根据需要添加更多验证逻辑
		
		return nil

	case *value_objects.BinaryExpression:
		// 递归解析子表达式
		err := sa.resolveNodeSymbols(n.Left, symbolResolver)
		if err != nil {
			return err
		}
		return sa.resolveNodeSymbols(n.Right, symbolResolver)

	case *value_objects.UnaryExpression:
		// 解析一元表达式
		return sa.resolveNodeSymbols(n.Operand, symbolResolver)

	case *value_objects.IfStatement:
		// 解析条件表达式
		err := sa.resolveNodeSymbols(n.Condition, symbolResolver)
		if err != nil {
			return err
		}

		err = sa.resolveNodeSymbols(n.ThenBranch, symbolResolver)
		if err != nil {
			return err
		}

		if n.ElseBranch != nil {
			return sa.resolveNodeSymbols(n.ElseBranch, symbolResolver)
		}
		return nil

	// 为其他节点类型添加符号解析逻辑...

	default:
		return nil
	}
}

// inferExpressionType 推断表达式的类型
func (sa *SemanticAnalyzer) inferExpressionType(expr value_objects.ASTNode) value_objects.SymbolType {
	switch e := expr.(type) {
	case *value_objects.IntegerLiteral:
		return value_objects.NewPrimitiveSymbolType("int")
	case *value_objects.FloatLiteral:
		return value_objects.NewPrimitiveSymbolType("float")
	case *value_objects.StringLiteral:
		return value_objects.NewPrimitiveSymbolType("string")
	case *value_objects.BooleanLiteral:
		return value_objects.NewPrimitiveSymbolType("bool")
	case *value_objects.Identifier:
		if symbol, found := sa.symbolTableEntity.ResolveSymbol(e.Name); found {
			return symbol.Type()
		}
		return value_objects.NewCustomSymbolType("unknown")
	case *value_objects.BinaryExpression:
		// 简化的类型推断
		leftType := sa.inferExpressionType(e.Left)
		// 对于算术运算，通常返回左操作数的类型
		if e.Operator == "+" || e.Operator == "-" || e.Operator == "*" || e.Operator == "/" {
			return leftType
		}
		// 对于比较运算，返回布尔类型
		return value_objects.NewPrimitiveSymbolType("bool")
	default:
		return value_objects.NewCustomSymbolType("unknown")
	}
}

// typesCompatible 检查两种类型是否兼容
func (sa *SemanticAnalyzer) typesCompatible(actual, expected value_objects.SymbolType) bool {
	// 完善实现：类型兼容性检查
	// 1. 完全匹配
	if actual.TypeName() == expected.TypeName() {
		return true
	}
	
	// 2. 检查是否可以隐式转换
	return sa.canImplicitlyConvert(actual, expected)
}

// canImplicitlyConvert 检查是否可以进行隐式类型转换
func (sa *SemanticAnalyzer) canImplicitlyConvert(from, to value_objects.SymbolType) bool {
	fromName := from.TypeName()
	toName := to.TypeName()
	
	// 允许的隐式转换规则
	// int -> float（数值提升）
	if fromName == "int" && toName == "float" {
		return true
	}
	
	// 相同的基础类型（忽略大小写）
	if strings.ToLower(fromName) == strings.ToLower(toName) {
		return true
	}
	
	return false
}

// lookupSymbolInScopes 在作用域中查找符号
func (sa *SemanticAnalyzer) lookupSymbolInScopes(name string) (*value_objects.Symbol, bool) {
	// 完善实现：在多个作用域中查找符号
	// 首先尝试使用符号表的ResolveSymbol方法（它应该已经支持作用域查找）
	symbol, found := sa.symbolTableEntity.ResolveSymbol(name)
	if found {
		return symbol, true
	}
	
	// 如果ResolveSymbol没有找到，可以尝试其他查找策略
	// 例如：查找全局作用域、查找导入的符号等
	// 这里可以根据需要添加更多查找逻辑
	
	return nil, false
}

// canApplyOperator 检查是否可以对给定的类型应用运算符
func (sa *SemanticAnalyzer) canApplyOperator(operator string, leftType, rightType value_objects.SymbolType) bool {
	// 简化的运算符应用检查
	switch operator {
	case "+", "-", "*", "/":
		// 数值类型可以进行算术运算
		return sa.isNumericType(leftType) && sa.isNumericType(rightType)
	case "==", "!=", "<", ">", "<=", ">=":
		// 可以比较相同类型的值
		return leftType.TypeName() == rightType.TypeName()
	case "&&", "||":
		// 逻辑运算符需要布尔类型
		return sa.isBooleanType(leftType) && sa.isBooleanType(rightType)
	default:
		return false
	}
}

// isNumericType 检查是否为数值类型
func (sa *SemanticAnalyzer) isNumericType(symbolType value_objects.SymbolType) bool {
	typeName := symbolType.TypeName()
	return typeName == "int" || typeName == "float"
}

// isBooleanType 检查是否为布尔类型
func (sa *SemanticAnalyzer) isBooleanType(symbolType value_objects.SymbolType) bool {
	return symbolType.TypeName() == "bool"
}

// parametersToString 将参数列表转换为字符串表示
func parametersToString(parameters []*value_objects.Parameter) string {
	if len(parameters) == 0 {
		return ""
	}

	result := ""
	for i, param := range parameters {
		if i > 0 {
			result += ", "
		}
		result += param.Type
	}
	return result
}
