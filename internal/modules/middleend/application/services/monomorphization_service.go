package services

import (
	"fmt"

	"echo/internal/modules/frontend/domain/entities"
	"echo/internal/modules/middleend/domain/services"
)

// MonomorphizationService 单态化应用服务
type MonomorphizationService struct {
	monomorphization *services.Monomorphization
}

// NewMonomorphizationService 创建单态化服务
func NewMonomorphizationService() *MonomorphizationService {
	return &MonomorphizationService{
		monomorphization: services.NewMonomorphization(),
	}
}

// ProcessProgram 对程序进行单态化处理
func (s *MonomorphizationService) ProcessProgram(program *entities.Program) (*entities.Program, error) {
	result, err := s.monomorphization.MonomorphizeProgram(program)
	if err != nil {
		return nil, fmt.Errorf("monomorphization failed: %v", err)
	}

	return result, nil
}

// GetMonomorphizedFunctions 获取生成的单态化函数
func (s *MonomorphizationService) GetMonomorphizedFunctions() []*services.MonomorphizedFunction {
	// 将map转换为slice格式
	functions := s.monomorphization.GetMonomorphizedFunctions()
	result := make([]*services.MonomorphizedFunction, len(functions))
	for i, fn := range functions {
		result[i] = fn
	}
	return result
}
