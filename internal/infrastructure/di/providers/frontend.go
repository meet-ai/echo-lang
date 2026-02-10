package providers

import (
	"echo/internal/domain/logger"
	backendAppServices "echo/internal/modules/backend/application/services"
	backendDomainServices "echo/internal/modules/backend/domain/services"
	"echo/internal/modules/backend/infrastructure/codegen"
	"echo/internal/modules/frontend"
	errorRecoveryApp "echo/internal/modules/frontend/application/error_recovery"
	frontendAppParser "echo/internal/modules/frontend/application/parser"
	frontendAppSemantic "echo/internal/modules/frontend/application/semantic"
	frontendAppServices "echo/internal/modules/frontend/application/services"
	domainErrorRecovery "echo/internal/modules/frontend/domain/error_recovery/services"
	frontendDomainServices "echo/internal/modules/frontend/domain/services"
	lexicalServices "echo/internal/modules/frontend/domain/lexical/services"
	semanticDomainServices "echo/internal/modules/frontend/domain/semantic/services"
	syntaxServices "echo/internal/modules/frontend/domain/syntax/services"
	portServices "echo/internal/modules/frontend/ports/services"

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

	// ==================== Application 解析流水线（ParserApplicationService） ====================
	do.Provide(container, func(i *do.Injector) (portServices.LexicalAnalysisService, error) {
		return lexicalServices.NewLexerService(nil), nil
	})
	do.Provide(container, func(i *do.Injector) (portServices.SyntaxAnalysisService, error) {
		return syntaxServices.NewSyntaxAnalyzer(), nil
	})
	do.Provide(container, func(i *do.Injector) (*semanticDomainServices.SemanticAnalyzer, error) {
		return semanticDomainServices.NewSemanticAnalyzer("default"), nil
	})
	do.Provide(container, func(i *do.Injector) (portServices.ProgramSemanticAnalyzer, error) {
		analyzer := do.MustInvoke[*semanticDomainServices.SemanticAnalyzer](i)
		return frontendAppSemantic.NewSemanticAnalysisApplicationService(analyzer), nil
	})
	do.Provide(container, func(i *do.Injector) (portServices.ParserApplicationService, error) {
		lex := do.MustInvoke[portServices.LexicalAnalysisService](i)
		syn := do.MustInvoke[portServices.SyntaxAnalysisService](i)
		progSem := do.MustInvoke[portServices.ProgramSemanticAnalyzer](i)
		return frontendAppParser.NewParserApplicationService(lex, syn, progSem), nil
	})
	do.Provide(container, func(i *do.Injector) (portServices.PackageResolutionService, error) {
		return frontendAppSemantic.NewPackageResolutionApplicationService(), nil
	})
	do.Provide(container, func(i *do.Injector) (*errorRecoveryApp.ErrorRecoveryApplicationService, error) {
		basic := domainErrorRecovery.NewErrorRecoveryService(3, 1000)
		advanced := domainErrorRecovery.NewAdvancedErrorRecoveryService()
		return errorRecoveryApp.NewErrorRecoveryApplicationService(basic, advanced), nil
	})
	do.Provide(container, func(i *do.Injector) (portServices.ErrorRecoveryService, error) {
		return do.MustInvoke[*errorRecoveryApp.ErrorRecoveryApplicationService](i), nil
	})
	do.Provide(container, func(i *do.Injector) (portServices.AdvancedErrorRecoveryPort, error) {
		return errorRecoveryApp.NewAdvancedErrorRecoveryPortAdapter(do.MustInvoke[*errorRecoveryApp.ErrorRecoveryApplicationService](i)), nil
	})
	do.Provide(container, func(i *do.Injector) (frontend.ModernFrontendService, error) {
		// nil 使 run/ParseSourceCode 走 Modern 管线（parserAggregate.Parse），正确解析 spawn/await/chan
		return frontendAppServices.NewModernFrontendService(nil, nil), nil
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
