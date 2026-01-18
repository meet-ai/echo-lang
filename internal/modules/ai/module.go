// Package ai provides AI-powered assistance and analysis capabilities for the Echo Language.
// This module handles intelligent code analysis, suggestions, and automated tasks.
package ai

import (
	"context"
	"fmt"

	"github.com/samber/do"
)

// Module represents the AI processing module
type Module struct {
	// Ports - external interfaces
	aiService AIService

	// Domain services
	codeAnalyzer   CodeAnalyzer
	suggestionEngine SuggestionEngine
	taskAutomator  TaskAutomator

	// Infrastructure
	modelClient LLMClient
	vectorStore VectorStore
}

// AIService defines the interface for AI processing operations
type AIService interface {
	AnalyzeCode(ctx context.Context, cmd AnalyzeCodeCommand) (*CodeAnalysisResult, error)
	GenerateSuggestions(ctx context.Context, cmd GenerateSuggestionsCommand) (*SuggestionResult, error)
	AutomateTask(ctx context.Context, cmd AutomateTaskCommand) (*TaskAutomationResult, error)
}

// NewModule creates a new AI module with dependency injection
func NewModule(i *do.Injector) (*Module, error) {
	// Get dependencies from DI container
	codeAnalyzer := do.MustInvoke[CodeAnalyzer](i)
	suggestionEngine := do.MustInvoke[SuggestionEngine](i)
	taskAutomator := do.MustInvoke[TaskAutomator](i)
	modelClient := do.MustInvoke[LLMClient](i)
	vectorStore := do.MustInvoke[VectorStore](i)

	// Create application services
	aiSvc := NewAIService(codeAnalyzer, suggestionEngine, taskAutomator)

	return &Module{
		aiService:       aiSvc,
		codeAnalyzer:    codeAnalyzer,
		suggestionEngine: suggestionEngine,
		taskAutomator:   taskAutomator,
		modelClient:     modelClient,
		vectorStore:     vectorStore,
	}, nil
}

// AIService returns the AI service interface
func (m *Module) AIService() AIService {
	return m.aiService
}

// Validate validates the module configuration
func (m *Module) Validate() error {
	if m.aiService == nil {
		return fmt.Errorf("AI service is not initialized")
	}
	if m.modelClient == nil {
		return fmt.Errorf("model client is not initialized")
	}
	if m.vectorStore == nil {
		return fmt.Errorf("vector store is not initialized")
	}
	return nil
}
