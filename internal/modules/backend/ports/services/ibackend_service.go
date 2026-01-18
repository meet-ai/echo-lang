package services

import "echo/internal/modules/frontend/domain/entities"

// IBackendService 后端服务接口
type IBackendService interface {
	// GenerateCode 生成目标代码
	GenerateCode(program *entities.Program, target string) (string, error)

	// GenerateExecutable 生成可执行代码
	GenerateExecutable(program *entities.Program, target string) (string, error)
}
