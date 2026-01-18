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

	// TODO: Add middleend and runtime providers when available
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

	// TODO: Build middleend and runtime modules when available
	// Temporarily disable middleend and runtime modules to focus on backend
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
func (c *Container) Compiler() Compiler {
	// TODO: Implement compiler service
	return nil
}
