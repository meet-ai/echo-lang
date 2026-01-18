package services

import (
	"fmt"

	"github.com/meetai/echo-lang/internal/modules/frontend/domain/entities"
)

// MonomorphizationService 单态化应用服务
type MonomorphizationService struct {
	monomorphization *Monomorphization
}

// NewMonomorphizationService 创建单态化服务
func NewMonomorphizationService() *MonomorphizationService {
	return &MonomorphizationService{
		monomorphization: NewMonomorphization(),
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
func (s *MonomorphizationService) GetMonomorphizedFunctions() map[string]*MonomorphizedFunction {
	return s.monomorphization.GetMonomorphizedFunctions()
}
