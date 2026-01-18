package services

import (
	"fmt"
	"strings"

	"echo/internal/modules/frontend/domain/entities"
)

// TypeInference 类型推断引擎
type TypeInference struct {
	// 类型变量映射：变量名 -> 具体类型
	typeVars map[string]string
	// 约束条件：变量名 -> 约束列表
	constraints map[string][]string
}

// NewTypeInference 创建新的类型推断引擎
func NewTypeInference() *TypeInference {
	return &TypeInference{
		typeVars:   make(map[string]string),
		constraints: make(map[string][]string),
	}
}

// InferTypes 推断泛型函数调用的类型参数
func (ti *TypeInference) InferTypes(funcDef *entities.FuncDef, call *entities.FuncCall) (map[string]string, error) {
	// 重置推断状态
	ti.typeVars = make(map[string]string)
	ti.constraints = make(map[string][]string)

	// 1. 设置显式类型参数
	if len(call.TypeArgs) > 0 {
		if len(call.TypeArgs) != len(funcDef.TypeParams) {
			return nil, fmt.Errorf("type argument count mismatch: expected %d, got %d",
				len(funcDef.TypeParams), len(call.TypeArgs))
		}
		for i, arg := range call.TypeArgs {
			paramName := funcDef.TypeParams[i].Name
			ti.typeVars[paramName] = arg
			ti.constraints[paramName] = funcDef.TypeParams[i].Constraints
		}
	}

	// 2. 从实参推断类型
	if err := ti.inferFromArgs(funcDef, call); err != nil {
		return nil, fmt.Errorf("argument inference failed: %v", err)
	}

	// 3. 检查约束
	if err := ti.checkConstraints(); err != nil {
		return nil, fmt.Errorf("constraint check failed: %v", err)
	}

	return ti.typeVars, nil
}

// inferFromArgs 从函数调用参数推断类型
func (ti *TypeInference) inferFromArgs(funcDef *entities.FuncDef, call *entities.FuncCall) error {
	for i, arg := range call.Args {
		if i >= len(funcDef.Params) {
			return fmt.Errorf("too many arguments")
		}

		paramType := funcDef.Params[i].Type
		if err := ti.inferFromArg(paramType, arg); err != nil {
			return fmt.Errorf("argument %d inference failed: %v", i, err)
		}
	}

	return nil
}

// inferFromArg 从单个参数推断类型
func (ti *TypeInference) inferFromArg(paramType string, arg entities.Expr) error {
	// 如果参数类型是类型变量，直接从实参推断
	if ti.isTypeVariable(paramType) {
		inferredType := ti.getExprType(arg)
		if inferredType != "" {
			ti.typeVars[paramType] = inferredType
		}
		return nil
	}

	// 如果参数类型是复合类型（如Container[T]），递归推断
	if strings.Contains(paramType, "[") && strings.Contains(paramType, "]") {
		return ti.inferFromGenericType(paramType, arg)
	}

	// 检查类型是否匹配（这里简化处理）
	inferredType := ti.getExprType(arg)
	if inferredType != "" && inferredType != paramType {
		return fmt.Errorf("type mismatch: expected %s, got %s", paramType, inferredType)
	}

	return nil
}

// inferFromGenericType 从泛型类型参数推断
func (ti *TypeInference) inferFromGenericType(genericType string, arg entities.Expr) error {
	// 解析泛型类型，如 "Container[T]"
	baseType, typeParams := ti.parseGenericType(genericType)

	// 检查实参是否是对应的泛型类型
	switch expr := arg.(type) {
	case *entities.FuncCall:
		// 可能是泛型函数调用，如 identity[int](42)
		if expr.Name == baseType && len(expr.TypeArgs) == len(typeParams) {
			for i, typeArg := range expr.TypeArgs {
				if i < len(typeParams) && ti.isTypeVariable(typeParams[i]) {
					ti.typeVars[typeParams[i]] = typeArg
				}
			}
		}
	}

	return nil
}

// parseGenericType 解析泛型类型，返回基础类型和类型参数
func (ti *TypeInference) parseGenericType(genericType string) (string, []string) {
	start := strings.Index(genericType, "[")
	if start == -1 {
		return genericType, nil
	}

	baseType := genericType[:start]
	paramStr := genericType[start+1 : len(genericType)-1]
	params := strings.Split(paramStr, ",")
	for i, param := range params {
		params[i] = strings.TrimSpace(param)
	}

	return baseType, params
}

// getExprType 获取表达式的类型
func (ti *TypeInference) getExprType(expr entities.Expr) string {
	switch expr.(type) {
	case *entities.IntLiteral:
		return "int"
	case *entities.StringLiteral:
		return "string"
	case *entities.Identifier:
		// 这里简化处理，实际应该从符号表获取
		return ""
	case *entities.FuncCall:
		// 函数调用返回类型需要更复杂的推断
		return ""
	default:
		return ""
	}
}

// isTypeVariable 检查是否是类型变量（如 T, U）
func (ti *TypeInference) isTypeVariable(typeName string) bool {
	// 简化检查：单个大写字母认为是类型变量
	if len(typeName) == 1 && typeName >= "A" && typeName <= "Z" {
		return true
	}
	return false
}

// checkConstraints 检查类型变量是否满足约束条件
func (ti *TypeInference) checkConstraints() error {
	for varName, constraints := range ti.constraints {
		if len(constraints) == 0 {
			continue
		}

		actualType, exists := ti.typeVars[varName]
		if !exists {
			return fmt.Errorf("type variable %s not inferred", varName)
		}

		for _, constraint := range constraints {
			if !ti.SatisfiesConstraint(actualType, constraint) {
				return fmt.Errorf("type %s does not satisfy constraint %s", actualType, constraint)
			}
		}
	}

	return nil
}

// SatisfiesConstraint 检查类型是否满足约束（公开方法）
func (ti *TypeInference) SatisfiesConstraint(actualType, constraint string) bool {
	// 这里简化实现，实际需要根据Trait定义检查
	// 例如，如果constraint是"Printable"，需要检查actualType是否实现了Printable trait

	// 临时实现：假设所有基本类型都满足常见约束
	switch constraint {
	case "Printable":
		return actualType == "int" || actualType == "string" || actualType == "bool"
	case "Comparable":
		return actualType == "int" || actualType == "string"
	default:
		return true // 未知约束暂时通过
	}
}

// InferFromReturnType 从返回值类型推断（用于变量赋值）
func (ti *TypeInference) InferFromReturnType(expectedType string, funcDef *entities.FuncDef, call *entities.FuncCall) (map[string]string, error) {
	// 重置推断状态
	ti.typeVars = make(map[string]string)

	// 如果返回值类型是类型变量，从期望类型推断
	if ti.isTypeVariable(funcDef.ReturnType) {
		ti.typeVars[funcDef.ReturnType] = expectedType
		return ti.typeVars, nil
	}

	// 如果返回值类型是泛型，从期望类型反推
	if strings.Contains(funcDef.ReturnType, "[") && strings.Contains(funcDef.ReturnType, "]") {
		return ti.inferFromReturnGenericType(expectedType, funcDef.ReturnType)
	}

	return ti.typeVars, nil
}

// inferFromReturnGenericType 从返回泛型类型反推参数类型
func (ti *TypeInference) inferFromReturnGenericType(expectedType, returnType string) (map[string]string, error) {
	expectedBase, expectedParams := ti.parseGenericType(expectedType)
	returnBase, returnParams := ti.parseGenericType(returnType)

	if expectedBase != returnBase || len(expectedParams) != len(returnParams) {
		return nil, fmt.Errorf("return type mismatch")
	}

	for i, returnParam := range returnParams {
		if ti.isTypeVariable(returnParam) && i < len(expectedParams) {
			ti.typeVars[returnParam] = expectedParams[i]
		}
	}

	return ti.typeVars, nil
}

// GetTypeVars 获取推断出的类型变量映射
func (ti *TypeInference) GetTypeVars() map[string]string {
	result := make(map[string]string)
	for k, v := range ti.typeVars {
		result[k] = v
	}
	return result
}
