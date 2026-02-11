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
	currentFunctionReturnType *value_objects.SymbolType // 当前函数的返回类型（用于return语句检查）
}

// NewSemanticAnalyzer 创建新的语义分析器
func NewSemanticAnalyzer(symbolTableID string) *SemanticAnalyzer {
	return &SemanticAnalyzer{
		symbolTableEntity: entities.NewSymbolTableEntity(symbolTableID),
	}
}

// SymbolTable 返回当前符号表（供 Application 层创建 SemanticAnalysisResult 等使用）
func (sa *SemanticAnalyzer) SymbolTable() *value_objects.SymbolTable {
	return sa.symbolTableEntity.SymbolTable()
}

// AnalyzeSemantics 分析AST的语义正确性
func (sa *SemanticAnalyzer) AnalyzeSemantics(
	ctx context.Context,
	programAST *value_objects.ProgramAST,
) (*value_objects.SemanticAnalysisResult, error) {
	result := value_objects.NewSemanticAnalysisResult(sa.symbolTableEntity.SymbolTable())
	if err := sa.BuildSymbolTable(ctx, programAST, result); err != nil {
		return result, err
	}
	if err := sa.PerformTypeChecking(ctx, programAST, result); err != nil {
		return result, err
	}
	if err := sa.PerformSymbolResolution(ctx, programAST, result); err != nil {
		return result, err
	}
	return result, nil
}

// BuildSymbolTable 构建符号表（对外暴露的单步规则方法）
func (sa *SemanticAnalyzer) BuildSymbolTable(
	ctx context.Context,
	programAST *value_objects.ProgramAST,
	result *value_objects.SemanticAnalysisResult,
) error {
	builder := NewSymbolTableBuilder(sa.symbolTableEntity)
	return builder.Build(ctx, programAST, result)
}

// PerformTypeChecking 执行类型检查（对外暴露的单步规则方法）
func (sa *SemanticAnalyzer) PerformTypeChecking(
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
			returnType := value_objects.NewCustomSymbolType(n.ReturnType().String())
			sa.currentFunctionReturnType = &returnType
		} else {
			// 无返回类型，视为void
			voidType := value_objects.NewPrimitiveSymbolType("void")
			sa.currentFunctionReturnType = &voidType
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
		if n.Expression() != nil {
			// 推断返回值的类型
			returnType := sa.inferExpressionType(n.Expression())
			
			// 如果当前函数有返回类型，检查是否匹配
			if sa.currentFunctionReturnType != nil {
				// 检查类型兼容性（支持隐式类型转换）
				if !sa.typesCompatible(returnType, *sa.currentFunctionReturnType) {
					// 检查是否可以进行隐式类型转换（如 int -> float）
					if !sa.canImplicitlyConvert(returnType, *sa.currentFunctionReturnType) {
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
				if sa.currentFunctionReturnType.String() != "void" {
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

// PerformSymbolResolution 执行符号解析（对外暴露的单步规则方法）
func (sa *SemanticAnalyzer) PerformSymbolResolution(
	ctx context.Context,
	programAST *value_objects.ProgramAST,
	result *value_objects.SemanticAnalysisResult,
) error {
	resolver := NewSymbolResolver(sa.symbolTableEntity)
	return resolver.Resolve(ctx, programAST, result)
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
		if symbol, found := sa.symbolTableEntity.ResolveSymbol(e.Name()); found {
			return symbol.Type()
		}
		return value_objects.NewCustomSymbolType("unknown")
	case *value_objects.BinaryExpression:
		// 简化的类型推断
		leftType := sa.inferExpressionType(e.Left())
		// 对于算术运算，通常返回左操作数的类型
		op := e.Operator()
		if op == "+" || op == "-" || op == "*" || op == "/" {
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
	if actual.String() == expected.String() {
		return true
	}
	
	// 2. 检查是否可以隐式转换
	return sa.canImplicitlyConvert(actual, expected)
}

// canImplicitlyConvert 检查是否可以进行隐式类型转换
func (sa *SemanticAnalyzer) canImplicitlyConvert(from, to value_objects.SymbolType) bool {
	fromName := from.String()
	toName := to.String()
	
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

// canApplyOperator 检查是否可以对给定的类型应用运算符
func (sa *SemanticAnalyzer) canApplyOperator(operator string, leftType, rightType value_objects.SymbolType) bool {
	// 简化的运算符应用检查
	switch operator {
	case "+", "-", "*", "/":
		// 数值类型可以进行算术运算
		return sa.isNumericType(leftType) && sa.isNumericType(rightType)
	case "==", "!=", "<", ">", "<=", ">=":
		// 可以比较相同类型的值
		return leftType.String() == rightType.String()
	case "&&", "||":
		// 逻辑运算符需要布尔类型
		return sa.isBooleanType(leftType) && sa.isBooleanType(rightType)
	default:
		return false
	}
}

// isNumericType 检查是否为数值类型
func (sa *SemanticAnalyzer) isNumericType(symbolType value_objects.SymbolType) bool {
		typeName := symbolType.String()
	return typeName == "int" || typeName == "float"
}

// isBooleanType 检查是否为布尔类型
func (sa *SemanticAnalyzer) isBooleanType(symbolType value_objects.SymbolType) bool {
	return symbolType.String() == "bool"
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
		if param.TypeAnnotation() != nil {
			result += param.TypeAnnotation().String()
		} else {
			result += "unknown"
		}
	}
	return result
}
