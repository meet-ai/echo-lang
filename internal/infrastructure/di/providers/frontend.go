package providers

import (
	"github.com/meetai/echo-lang/internal/modules/frontend/application/services"
	"github.com/meetai/echo-lang/internal/modules/frontend/domain/services"
	"github.com/meetai/echo-lang/internal/modules/frontend/ports/services"
	"github.com/samber/do"
)

// ProvideFrontendServices registers frontend services with the DI container
func ProvideFrontendServices(container *do.Injector) {
	// Provide parser service
	do.Provide(container, func(i *do.Injector) services.Parser {
		return services.NewSimpleParser()
	})
}
