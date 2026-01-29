package di

import (
	"echo/internal/infrastructure/di/providers"
	"echo/internal/modules/backend"
	"echo/internal/modules/frontend"

	"echo/internal/modules/middleend"
	"echo/internal/modules/runtime"

	"github.com/samber/do"
)

// Container holds all application services and dependencies
type Container struct {
	Frontend  *frontend.Module
	Backend   *backend.Module
	Middleend *middleend.Module
	Runtime   *runtime.Module
}

// Compiler represents the main compiler interface
type Compiler interface {
	Compile(filename string) error
	CompileAndRun(filename string) error
}

// BuildContainer creates and configures the dependency injection container
func BuildContainer() (*Container, error) {
	injector := do.New()

	// Register service providers
	providers.ProvideFrontendServices(injector)
	providers.ProvideBackendServices(injector)

	// 当前架构阶段仅启用 Frontend+Backend；middleend/runtime 为占位，随架构演进再注册并启用。
	// providers.ProvideMiddleendServices(injector)
	// providers.ProvideRuntimeServices(injector)

	// Build modules
	frontendModule, err := frontend.NewModule(injector)
	if err != nil {
		return nil, err
	}

	backendModule, err := backend.NewModule(injector)
	if err != nil {
		return nil, err
	}

	// 与上一致：middleend/runtime 模块暂不构建，当前聚焦 backend 代码生成。
	var middleendModule *middleend.Module
	var runtimeModule *runtime.Module

	return &Container{
		Frontend:  frontendModule,
		Backend:   backendModule,
		Middleend: middleendModule,
		Runtime:   runtimeModule,
	}, nil
}

// Compiler returns the main compiler service
// 当前为占位：入口通过 cmd 直接组装 Frontend+Backend 调用，未经此接口；后续统一入口时可在此返回 Compiler 实现。
func (c *Container) Compiler() Compiler {
	return nil
}
