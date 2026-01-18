package ocaml

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// OCamlCompiler provides OCaml compilation services for the Echo Language.
// This handles the compilation of OCaml code generated from Echo source.
type OCamlCompiler struct {
	// Configuration
	ocamlcPath   string
	ocamloptPath string
	workingDir   string
	timeout      time.Duration
	optimization bool

	// Runtime state
	isInitialized bool
}

// NewOCamlCompiler creates a new OCaml compiler instance
func NewOCamlCompiler(ocamlcPath, ocamloptPath, workingDir string, timeout time.Duration) *OCamlCompiler {
	return &OCamlCompiler{
		ocamlcPath:    ocamlcPath,
		ocamloptPath:  ocamloptPath,
		workingDir:    workingDir,
		timeout:       timeout,
		optimization:  true, // Default to optimized compilation
		isInitialized: false,
	}
}

// Initialize initializes the OCaml compiler environment
func (c *OCamlCompiler) Initialize(ctx context.Context) error {
	if c.isInitialized {
		return nil
	}

	// Check if OCaml compiler is available
	if err := c.checkOCamlAvailability(ctx); err != nil {
		return fmt.Errorf("OCaml compiler not available: %w", err)
	}

	c.isInitialized = true
	return nil
}

// Compile compiles OCaml source code to bytecode or native code
func (c *OCamlCompiler) Compile(ctx context.Context, sourceFile string, outputFile string, options CompileOptions) (*CompileResult, error) {
	if !c.isInitialized {
		if err := c.Initialize(ctx); err != nil {
			return nil, err
		}
	}

	startTime := time.Now()

	// Choose compiler based on optimization setting
	var compilerPath string
	if c.optimization && c.ocamloptPath != "" {
		compilerPath = c.ocamloptPath // Native code compiler
	} else {
		compilerPath = c.ocamlcPath // Bytecode compiler
	}

	// Build compilation command
	args := []string{sourceFile, "-o", outputFile}

	// Add compilation options
	if options.IncludeDirs != nil {
		for _, dir := range options.IncludeDirs {
			args = append(args, "-I", dir)
		}
	}

	if options.Warnings {
		args = append(args, "-w", "+A")
	}

	if options.Debug {
		args = append(args, "-g")
	}

	if options.AdditionalArgs != nil {
		args = append(args, options.AdditionalArgs...)
	}

	// Execute compilation
	output, err := c.executeCommand(ctx, compilerPath, args...)
	duration := time.Since(startTime)

	if err != nil {
		return &CompileResult{
			Success:      false,
			OutputFile:   outputFile,
			Errors:       []string{err.Error()},
			Warnings:     []string{},
			Duration:     duration,
			CompilerUsed: compilerPath,
		}, nil
	}

	// Parse output for warnings and errors
	warnings, errors := c.parseOutput(output)

	return &CompileResult{
		Success:      len(errors) == 0,
		OutputFile:   outputFile,
		Errors:       errors,
		Warnings:     warnings,
		Duration:     duration,
		CompilerUsed: compilerPath,
	}, nil
}

// checkOCamlAvailability checks if OCaml compiler is available
func (c *OCamlCompiler) checkOCamlAvailability(ctx context.Context) error {
	// Check ocamlc
	if err := c.checkCompiler(ctx, c.ocamlcPath); err != nil {
		return fmt.Errorf("ocamlc check failed: %w", err)
	}

	// Check ocamlopt if optimization is enabled
	if c.optimization && c.ocamloptPath != "" {
		if err := c.checkCompiler(ctx, c.ocamloptPath); err != nil {
			return fmt.Errorf("ocamlopt check failed: %w", err)
		}
	}

	return nil
}

// checkCompiler checks if a specific OCaml compiler is available
func (c *OCamlCompiler) checkCompiler(ctx context.Context, compilerPath string) error {
	cmd := exec.CommandContext(ctx, compilerPath, "-version")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	outputStr := strings.TrimSpace(string(output))
	if !strings.Contains(outputStr, "The OCaml compiler") {
		return fmt.Errorf("unexpected OCaml compiler output: %s", outputStr)
	}

	return nil
}

// executeCommand executes a command with timeout
func (c *OCamlCompiler) executeCommand(ctx context.Context, command string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = c.workingDir

	output, err := cmd.CombinedOutput()
	return string(output), err
}

// parseOutput parses compilation output for warnings and errors
func (c *OCamlCompiler) parseOutput(output string) ([]string, []string) {
	var warnings, errors []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "Warning") || strings.Contains(line, "warning") {
			warnings = append(warnings, line)
		} else if strings.Contains(line, "Error") || strings.Contains(line, "error") {
			errors = append(errors, line)
		}
	}

	return warnings, errors
}

// SetOptimization enables or disables optimization
func (c *OCamlCompiler) SetOptimization(enabled bool) {
	c.optimization = enabled
}

// IsInitialized returns true if the compiler has been initialized
func (c *OCamlCompiler) IsInitialized() bool {
	return c.isInitialized
}

// CompileOptions represents compilation options
type CompileOptions struct {
	IncludeDirs    []string // Include directories
	Warnings       bool     // Enable warnings
	Debug          bool     // Include debug information
	AdditionalArgs []string // Additional compiler arguments
}

// CompileResult represents the result of a compilation
type CompileResult struct {
	Success      bool
	OutputFile   string
	Errors       []string
	Warnings     []string
	Duration     time.Duration
	CompilerUsed string
}

// HasErrors returns true if there are compilation errors
func (r *CompileResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings returns true if there are compilation warnings
func (r *CompileResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// String returns a string representation of the compile result
func (r *CompileResult) String() string {
	status := "SUCCESS"
	if !r.Success {
		status = "FAILED"
	}

	return fmt.Sprintf("Compilation %s: %s (%.2fs, %d warnings, %d errors)",
		status, r.OutputFile, r.Duration.Seconds(), len(r.Warnings), len(r.Errors))
}
