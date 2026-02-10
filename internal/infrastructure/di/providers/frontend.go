package providers

import (
	"echo/internal/domain/logger"
	backendAppServices "echo/internal/modules/backend/application/services"
	backendDomainServices "echo/internal/modules/backend/domain/services"
	"echo/internal/modules/backend/infrastructure/codegen"
	frontendAppServices "echo/internal/modules/frontend/application/services"
	frontendDomainServices "echo/internal/modules/frontend/domain/services"

	"github.com/samber/do"
)

// ProvideFrontendServices registers frontend services with the DI container
func ProvideFrontendServices(container *do.Injector) {
	// Provide domain parser service
	do.Provide(container, func(i *do.Injector) (frontendDomainServices.Parser, error) {
		l := do.MustInvoke[logger.Logger](i)
		p := frontendDomainServices.NewSimpleParser()
		if sp, ok := p.(*frontendDomainServices.SimpleParser); ok {
			sp.SetLogger(l)
		}
		return p, nil
	})

	// Domain service providers omitted until concrete implementations exist

	// Provide repositories
	// TODO: Add actual repository implementations
	// do.Provide(container, func(i *do.Injector) SourceFileRepository {
	// 	return infrastructure.NewSourceFileRepository()
	// })

	// Provide application services
	do.Provide(container, func(i *do.Injector) (frontendAppServices.IFrontendService, error) {
		lexicalAnalyzer := do.MustInvoke[frontendDomainServices.LexicalAnalyzer](i)
		syntaxAnalyzer := do.MustInvoke[frontendDomainServices.SyntaxAnalyzer](i)
		semanticAnalyzer := do.MustInvoke[frontendDomainServices.SemanticAnalyzer](i)
		errorHandler := do.MustInvoke[frontendDomainServices.ErrorHandler](i)
		parser := do.MustInvoke[frontendDomainServices.Parser](i)

		// TODO: Add repository dependencies
		return frontendAppServices.NewFrontendService(
			lexicalAnalyzer,
			syntaxAnalyzer,
			semanticAnalyzer,
			errorHandler,
			nil, // sourceFileRepo
			nil, // astRepo
			nil, // eventPublisher
			parser,
		), nil
	})

	// TokenizerService provider omitted
}

// ProvideBackendServices registers backend services with the DI container
func ProvideBackendServices(container *do.Injector) {
	// Provide domain services
	do.Provide(container, func(i *do.Injector) (backendDomainServices.TargetCodeGenerator, error) {
		log := do.MustInvoke[logger.Logger](i)
		// Default to LLVM generator
		return codegen.NewLLVMGenerator(log), nil
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
	do.Provide(container, func(i *do.Injector) (*backendAppServices.CodeGenerationService, error) {
		codeGenerator := do.MustInvoke[backendDomainServices.TargetCodeGenerator](i)
		// TODO: Add other dependencies when available
		return backendAppServices.NewCodeGenerationService(
			codeGenerator,
			nil, // assembler
			nil, // linker
			nil, // executableRepo
			nil, // objectFileRepo
		), nil
	})
}
