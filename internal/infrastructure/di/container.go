package di

import (
	"fmt"

	"github.com/meetai/echo-lang/internal/infrastructure/di/providers"
	"github.com/samber/do"
)

// Container wraps the do.Injector with additional functionality
type Container struct {
	*do.Injector
}

// NewContainer creates a new dependency injection container
func NewContainer() *Container {
	container := &Container{
		Injector: do.New(),
	}

	// Register all service providers
	providers.ProvideFrontendServices(container.Injector)
	// TODO: Add other module providers when implemented

	return container
}

// ProvideNamed provides a service with a specific name
func (c *Container) ProvideNamed(name string, constructor interface{}) {
	// In a real implementation, this would use do.ProvideNamed
	// For now, we'll use regular Provide
	do.Provide(c.Injector, constructor)
}

// InvokeNamed invokes a service by name
func (c *Container) InvokeNamed(name string, ptr interface{}) error {
	// In a real implementation, this would use do.InvokeNamed
	// For now, we'll use regular Invoke
	return do.Invoke(c.Injector, ptr)
}

// HasService checks if a service is registered
func (c *Container) HasService(serviceType interface{}) bool {
	// Try to invoke the service to check if it exists
	err := do.Invoke(c.Injector, serviceType)
	return err == nil
}

// ListServices returns a list of registered service types
func (c *Container) ListServices() []string {
	// This would require extending do.Injector to track registered services
	// For now, return empty slice
	return []string{}
}

// Validate validates the container configuration
func (c *Container) Validate() error {
	// Basic validation - check if container is initialized
	if c.Injector == nil {
		return fmt.Errorf("container is not initialized")
	}
	return nil
}

// Shutdown gracefully shuts down the container and all services
func (c *Container) Shutdown() error {
	// In a real implementation, this would call shutdown methods on services
	// that implement a Shutdown() interface
	return nil
}

// GetFrontendService gets the frontend service from the container
func (c *Container) GetFrontendService() (interface{}, error) {
	var service interface{}
	err := do.Invoke(c.Injector, &service)
	if err != nil {
		return nil, fmt.Errorf("failed to get frontend service: %w", err)
	}
	return service, nil
}
