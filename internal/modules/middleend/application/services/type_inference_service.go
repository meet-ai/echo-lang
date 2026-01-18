package services

import (
	"fmt"

	"github.com/meetai/echo-lang/internal/modules/frontend/domain/entities"
	"github.com/meetai/echo-lang/internal/modules/middleend/domain/services"
)

// TypeInferenceService 类型推断应用服务
type TypeInferenceService struct {
	inferenceEngine *services.TypeInference
}

// NewTypeInferenceService 创建类型推断服务
func NewTypeInferenceService() *TypeInferenceService {
	return &TypeInferenceService{
		inferenceEngine: services.NewTypeInference(),
	}
}

// InferFuncCallTypes 推断函数调用的类型参数
func (s *TypeInferenceService) InferFuncCallTypes(funcDef *entities.FuncDef, call *entities.FuncCall) (map[string]string, error) {
	if len(funcDef.TypeParams) == 0 {
		// 非泛型函数，无需推断
		return make(map[string]string), nil
	}

	inferredTypes, err := s.inferenceEngine.InferTypes(funcDef, call)
	if err != nil {
		return nil, fmt.Errorf("type inference failed: %v", err)
	}

	return inferredTypes, nil
}

// InferFromAssignment 从赋值推断类型参数
func (s *TypeInferenceService) InferFromAssignment(expectedType string, funcDef *entities.FuncDef, call *entities.FuncCall) (map[string]string, error) {
	if len(funcDef.TypeParams) == 0 {
		return make(map[string]string), nil
	}

	inferredTypes, err := s.inferenceEngine.InferFromReturnType(expectedType, funcDef, call)
	if err != nil {
		return nil, fmt.Errorf("assignment type inference failed: %v", err)
	}

	return inferredTypes, nil
}

// ValidateTypeArgs 验证显式类型参数
func (s *TypeInferenceService) ValidateTypeArgs(funcDef *entities.FuncDef, typeArgs []string) error {
	if len(typeArgs) != len(funcDef.TypeParams) {
		return fmt.Errorf("type argument count mismatch: expected %d, got %d",
			len(funcDef.TypeParams), len(typeArgs))
	}

	// 检查约束
	for i, typeArg := range typeArgs {
		constraints := funcDef.TypeParams[i].Constraints
		for _, constraint := range constraints {
			if !s.inferenceEngine.SatisfiesConstraint(typeArg, constraint) {
				return fmt.Errorf("type %s does not satisfy constraint %s", typeArg, constraint)
			}
		}
	}

	return nil
}

// GetInferredTypes 获取推断出的类型映射
func (s *TypeInferenceService) GetInferredTypes() map[string]string {
	return s.inferenceEngine.GetTypeVars()
}
