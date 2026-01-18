package adapters

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/meetai/echo-lang/internal/modules/frontend/domain/valueobjects"
)

// OCamlLexAdapter adapts the ocamllex lexer for use in the frontend infrastructure layer.
// This adapter handles the interaction with the OCaml lexical analyzer.
type OCamlLexAdapter struct {
	// Configuration
	lexerPath       string
	workingDir      string
	timeout         time.Duration
	includeComments bool

	// Runtime state
	isInitialized bool
}

// NewOCamlLexAdapter creates a new OCaml lexer adapter
func NewOCamlLexAdapter(lexerPath, workingDir string, timeout time.Duration) *OCamlLexAdapter {
	return &OCamlLexAdapter{
		lexerPath:       lexerPath,
		workingDir:      workingDir,
		timeout:         timeout,
		includeComments: true, // Default to include comments
		isInitialized:   false,
	}
}

// Initialize initializes the OCaml lexer
func (a *OCamlLexAdapter) Initialize(ctx context.Context) error {
	if a.isInitialized {
		return nil
	}

	// Check if ocamllex is available
	if err := a.checkOCamlLexAvailability(ctx); err != nil {
		return fmt.Errorf("ocamllex not available: %w", err)
	}

	a.isInitialized = true
	return nil
}

// Tokenize performs lexical analysis on the given source code
func (a *OCamlLexAdapter) Tokenize(ctx context.Context, sourceCode string, options TokenizeOptions) ([]valueobjects.Token, error) {
	if !a.isInitialized {
		if err := a.Initialize(ctx); err != nil {
			return nil, err
		}
	}

	// Create a temporary file for the source code
	tempFile, err := a.createTempSourceFile(sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer a.cleanupTempFile(tempFile)

	// Execute the OCaml lexer
	output, err := a.executeLexer(ctx, tempFile, options)
	if err != nil {
		return nil, fmt.Errorf("lexer execution failed: %w", err)
	}

	// Parse the lexer output into tokens
	tokens, err := a.parseLexerOutput(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lexer output: %w", err)
	}

	return tokens, nil
}

// checkOCamlLexAvailability checks if ocamllex is available on the system
func (a *OCamlLexAdapter) checkOCamlLexAvailability(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "ocamllex", "--version")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	// Check if the output indicates ocamllex is available
	outputStr := strings.TrimSpace(string(output))
	if !strings.Contains(outputStr, "ocamllex") {
		return fmt.Errorf("unexpected ocamllex output: %s", outputStr)
	}

	return nil
}

// createTempSourceFile creates a temporary file with the source code
func (a *OCamlLexAdapter) createTempSourceFile(sourceCode string) (string, error) {
	// In a real implementation, this would create a temporary file
	// For demonstration, we'll return a mock path
	return "/tmp/echo_source_temp.eo", nil
}

// cleanupTempFile cleans up the temporary file
func (a *OCamlLexAdapter) cleanupTempFile(filePath string) {
	// In a real implementation, this would delete the temporary file
	// For demonstration, we'll do nothing
}

// executeLexer executes the OCaml lexer on the source file
func (a *OCamlLexAdapter) executeLexer(ctx context.Context, sourceFile string, options TokenizeOptions) (string, error) {
	args := []string{a.lexerPath, sourceFile}

	// Add options based on configuration
	if !a.includeComments {
		args = append(args, "--no-comments")
	}

	if options.Encoding != "" {
		args = append(args, "--encoding", options.Encoding)
	}

	cmd := exec.CommandContext(ctx, "ocamlrun", args...)
	cmd.Dir = a.workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("lexer command failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// parseLexerOutput parses the lexer output into tokens
func (a *OCamlLexAdapter) parseLexerOutput(output string) ([]valueobjects.Token, error) {
	// In a real implementation, this would parse the structured output from ocamllex
	// For demonstration, we'll return mock tokens
	tokens := []valueobjects.Token{
		valueobjects.NewToken(valueobjects.TokenTypeFunc, "func", 1, 1, 0),
		valueobjects.NewToken(valueobjects.TokenTypeIdentifier, "main", 1, 6, 5),
		valueobjects.NewToken(valueobjects.TokenTypeLeftParen, "(", 1, 10, 9),
		valueobjects.NewToken(valueobjects.TokenTypeRightParen, ")", 1, 11, 10),
		valueobjects.NewToken(valueobjects.TokenTypeEOF, "", 1, 12, 11),
	}

	return tokens, nil
}

// TokenizeOptions represents options for tokenization
type TokenizeOptions struct {
	Encoding        string
	IncludeComments bool
	AdditionalArgs  []string
}

// IsInitialized returns true if the adapter has been initialized
func (a *OCamlLexAdapter) IsInitialized() bool {
	return a.isInitialized
}

// SetIncludeComments sets whether to include comments in tokenization
func (a *OCamlLexAdapter) SetIncludeComments(include bool) {
	a.includeComments = include
}
