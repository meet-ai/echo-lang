package providers

import (
	"echo/internal/modules/backend/application/services"
	"echo/internal/modules/backend/domain/services"
	"echo/internal/modules/backend/infrastructure/codegen"
	"echo/internal/modules/frontend/application/services"
	"echo/internal/modules/frontend/domain/services"
	"echo/internal/modules/frontend/ports/services"

	"github.com/samber/do"
)

// ProvideFrontendServices registers frontend services with the DI container
func ProvideFrontendServices(container *do.Injector) {
	// Provide parser service
	do.Provide(container, func(i *do.Injector) services.Parser {
		return services.NewSimpleParser()
	})
}

// ProvideBackendServices registers backend services with the DI container
func ProvideBackendServices(container *do.Injector) {
	// Provide domain services
	do.Provide(container, func(i *do.Injector) services.TargetCodeGenerator {
		// Default to LLVM generator
		return codegen.NewLLVMGenerator()
	})

	// Provide infrastructure services
	// TODO: Add assembler and linker implementations
	// do.Provide(container, func(i *do.Injector) services.Assembler {
	// 	return infrastructure.NewAssembler()
	// })

	// Provide repositories
	// TODO: Add repository implementations
	// do.Provide(container, func(i *do.Injector) services.ExecutableRepository {
	// 	return infrastructure.NewExecutableRepository()
	// })

	// Provide application services
	do.Provide(container, func(i *do.Injector) services.CodeGenerationService {
		codeGenerator := do.MustInvoke[services.TargetCodeGenerator](i)
		// TODO: Add other dependencies when available
		return services.NewCodeGenerationService(
			codeGenerator,
			nil, // assembler
			nil, // linker
			nil, // executableRepo
			nil, // objectFileRepo
		)
	})
}
