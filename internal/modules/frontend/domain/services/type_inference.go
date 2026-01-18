package services

import (
	"fmt"

	"echo/internal/modules/frontend/domain/entities"
)

// TypeInferenceService 类型推断服务接口
type TypeInferenceService interface {
	// InferTypes 为AST中的变量声明推断类型
	InferTypes(program *entities.Program) error
}

// SimpleTypeInferenceService 简单类型推断服务实现
type SimpleTypeInferenceService struct{}

// NewSimpleTypeInferenceService 创建类型推断服务
func NewSimpleTypeInferenceService() TypeInferenceService {
	return &SimpleTypeInferenceService{}
}

// InferTypes 实现类型推断逻辑
func (s *SimpleTypeInferenceService) InferTypes(program *entities.Program) error {
	return s.inferTypesInProgram(program)
}

// inferTypesInProgram 遍历程序中的所有节点，进行类型推断
func (s *SimpleTypeInferenceService) inferTypesInProgram(program *entities.Program) error {
	for _, stmt := range program.Statements {
		if err := s.inferTypesInStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

// inferTypesInFunction 在函数中进行类型推断
func (s *SimpleTypeInferenceService) inferTypesInFunction(funcDef *entities.FuncDef) error {
	for _, stmt := range funcDef.Body {
		if err := s.inferTypesInStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

// inferTypesInStatement 在语句中进行类型推断
func (s *SimpleTypeInferenceService) inferTypesInStatement(stmt entities.ASTNode) error {
	switch stmt := stmt.(type) {
	case *entities.VarDecl:
		return s.inferType(stmt)
	case *entities.FuncDef:
		return s.inferTypesInFunction(stmt)
	}
	return nil
}

// inferType 为变量声明推断类型
func (s *SimpleTypeInferenceService) inferType(varDecl *entities.VarDecl) error {
	if !varDecl.Inferred {
		// 已经有显式类型，不需要推断
		return nil
	}

	inferredType, err := s.inferTypeFromExpression(varDecl.Value)
	if err != nil {
		return fmt.Errorf("cannot infer type for variable %s: %v", varDecl.Name, err)
	}

	varDecl.Type = inferredType
	varDecl.Inferred = false // 标记为已推断
	return nil
}

// inferTypeFromExpression 从表达式推断类型
func (s *SimpleTypeInferenceService) inferTypeFromExpression(expr entities.Expr) (string, error) {
	switch e := expr.(type) {
	case *entities.IntLiteral:
		return "int", nil
	case *entities.FloatLiteral:
		return "float", nil
	case *entities.StringLiteral:
		return "string", nil
	case *entities.BoolLiteral:
		return "bool", nil
	case *entities.BinaryExpr:
		// 对于二元表达式，需要根据操作符和操作数推断
		return s.inferTypeFromBinaryExpr(e)
	case *entities.Identifier:
		// 变量引用无法在编译时推断，需要运行时信息
		return "", fmt.Errorf("cannot infer type from variable reference '%s' - variable references require explicit type annotation", e.Name)
	case *entities.FuncCall:
		// 函数调用结果类型无法在编译时确定
		return "", fmt.Errorf("cannot infer type from function call - function return types require explicit type annotation")
	default:
		return "", fmt.Errorf("unsupported expression type for type inference: %T (value: %v)", expr, expr)
	}
}

// inferTypeFromBinaryExpr 从二元表达式推断类型
func (s *SimpleTypeInferenceService) inferTypeFromBinaryExpr(expr *entities.BinaryExpr) (string, error) {
	leftType, err := s.inferTypeFromExpression(expr.Left)
	if err != nil {
		return "", err
	}

	rightType, err := s.inferTypeFromExpression(expr.Right)
	if err != nil {
		return "", err
	}

	// 类型必须相同
	if leftType != rightType {
		return "", fmt.Errorf("type mismatch in binary expression: %s vs %s", leftType, rightType)
	}

	// 根据操作符确定结果类型
	switch expr.Op {
	case "+", "-", "*", "/":
		if leftType == "int" {
			return leftType, nil
		}
	case "==", "!=", "<", "<=", ">", ">=":
		return "bool", nil
	case "&&", "||":
		if leftType == "bool" {
			return "bool", nil
		}
	}

	return "", fmt.Errorf("unsupported binary operation: %s %s %s", leftType, expr.Op, rightType)
}
