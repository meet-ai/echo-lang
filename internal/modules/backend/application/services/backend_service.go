package services

import (
	"fmt"

	"github.com/meetai/echo-lang/internal/modules/backend/domain/services"
	"github.com/meetai/echo-lang/internal/modules/backend/ports/services"
	"github.com/meetai/echo-lang/internal/modules/frontend/domain/entities"
)

// BackendService 后端应用服务
type BackendService struct {
	codeGenerator services.CodeGenerator
}

// NewBackendService 创建后端服务
func NewBackendService(codeGenerator services.CodeGenerator) services.IBackendService {
	return &BackendService{
		codeGenerator: codeGenerator,
	}
}

// GenerateCode 生成目标代码
func (s *BackendService) GenerateCode(program *entities.Program, target string) (string, error) {
	switch target {
	case "ocaml":
		return s.codeGenerator.GenerateOCaml(program), nil
	default:
		return "", fmt.Errorf("unsupported target: %s", target)
	}
}

// GenerateExecutable 生成可执行代码
func (s *BackendService) GenerateExecutable(program *entities.Program, target string) (string, error) {
	switch target {
	case "ocaml":
		return s.codeGenerator.GenerateExecutableOCaml(program), nil
	default:
		return "", fmt.Errorf("unsupported target: %s", target)
	}
}
