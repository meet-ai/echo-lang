// Package project provides project management and organization capabilities for Echo Language.
// This module handles project structure, dependencies, and build configurations.
package project

import (
	"context"
	"fmt"

	"github.com/samber/do"
)

// Module represents the project management module
type Module struct {
	// Ports - external interfaces
	projectService ProjectService

	// Domain services
	projectManager     ProjectManager
	dependencyResolver DependencyResolver
	buildConfigurator  BuildConfigurator

	// Infrastructure
	fileSystem  FileSystem
	configStore ConfigStore
}

// ProjectService defines the interface for project management operations
type ProjectService interface {
	CreateProject(ctx context.Context, cmd CreateProjectCommand) (*ProjectResult, error)
	AnalyzeDependencies(ctx context.Context, cmd AnalyzeDependenciesCommand) (*DependencyAnalysisResult, error)
	ConfigureBuild(ctx context.Context, cmd ConfigureBuildCommand) (*BuildConfigurationResult, error)
}

// NewModule creates a new project module with dependency injection
func NewModule(i *do.Injector) (*Module, error) {
	// Get dependencies from DI container
	projectManager := do.MustInvoke[ProjectManager](i)
	dependencyResolver := do.MustInvoke[DependencyResolver](i)
	buildConfigurator := do.MustInvoke[BuildConfigurator](i)
	fileSystem := do.MustInvoke[FileSystem](i)
	configStore := do.MustInvoke[ConfigStore](i)

	// Create application services
	projectSvc := NewProjectService(projectManager, dependencyResolver, buildConfigurator)

	return &Module{
		projectService:     projectSvc,
		projectManager:     projectManager,
		dependencyResolver: dependencyResolver,
		buildConfigurator:  buildConfigurator,
		fileSystem:         fileSystem,
		configStore:        configStore,
	}, nil
}

// ProjectService returns the project service interface
func (m *Module) ProjectService() ProjectService {
	return m.projectService
}

// Validate validates the module configuration
func (m *Module) Validate() error {
	if m.projectService == nil {
		return fmt.Errorf("project service is not initialized")
	}
	if m.fileSystem == nil {
		return fmt.Errorf("file system is not initialized")
	}
	if m.configStore == nil {
		return fmt.Errorf("config store is not initialized")
	}
	return nil
}
