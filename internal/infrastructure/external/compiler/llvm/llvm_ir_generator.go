package llvm

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/meetai/echo-lang/internal/modules/middleend"
)

// LLVMIRGenerator LLVM IR生成器适配器
type LLVMIRGenerator struct {
	// Configuration
	llvmPath          string
	workingDir        string
	timeout           time.Duration
	optimizationLevel int

	// Runtime state
	isInitialized bool
}

// NewLLVMIRGenerator 创建LLVM IR生成器
func NewLLVMIRGenerator(llvmPath, workingDir string, timeout time.Duration) *LLVMIRGenerator {
	return &LLVMIRGenerator{
		llvmPath:          llvmPath,
		workingDir:        workingDir,
		timeout:           timeout,
		optimizationLevel: 0, // Default to -O0
		isInitialized:     false,
	}
}

// Initialize 初始化LLVM IR生成器
func (g *LLVMIRGenerator) Initialize(ctx context.Context) error {
	if g.isInitialized {
		return nil
	}

	// Check if LLVM tools are available
	if err := g.checkLLVMAvailability(ctx); err != nil {
		return fmt.Errorf("LLVM tools not available: %w", err)
	}

	g.isInitialized = true
	return nil
}

// GenerateIR 生成LLVM IR
func (g *LLVMIRGenerator) GenerateIR(ctx context.Context, sourceCode string, options IROptions) (*IRGenerationResult, error) {
	if !g.isInitialized {
		if err := g.Initialize(ctx); err != nil {
			return nil, err
		}
	}

	startTime := time.Now()

	// Create temporary source file
	sourceFile, err := g.createTempSourceFile(sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp source file: %w", err)
	}
	defer g.cleanupTempFile(sourceFile)

	// Generate IR using clang or llc
	irCode, err := g.compileToIR(ctx, sourceFile, options)
	if err != nil {
		return &IRGenerationResult{
			Success:  false,
			Errors:   []string{err.Error()},
			Duration: time.Since(startTime),
		}, nil
	}

	// Validate generated IR
	if err := g.validateIR(irCode); err != nil {
		return &IRGenerationResult{
			Success:  false,
			Errors:   []string{fmt.Sprintf("IR validation failed: %v", err)},
			Duration: time.Since(startTime),
		}, nil
	}

	return &IRGenerationResult{
		Success:           true,
		IRCode:            irCode,
		SourceFile:        sourceFile,
		Duration:          time.Since(startTime),
		OptimizationLevel: g.optimizationLevel,
	}, nil
}

// OptimizeIR 优化IR
func (g *LLVMIRGenerator) OptimizeIR(ctx context.Context, irCode string, level int) (*OptimizationResult, error) {
	if !g.isInitialized {
		if err := g.Initialize(ctx); err != nil {
			return nil, err
		}
	}

	startTime := time.Now()

	// Create temporary IR file
	irFile, err := g.createTempIRFile(irCode)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp IR file: %w", err)
	}
	defer g.cleanupTempFile(irFile)

	// Run opt for optimization
	optimizedIR, err := g.runOptimization(ctx, irFile, level)
	if err != nil {
		return &OptimizationResult{
			Success:  false,
			Errors:   []string{err.Error()},
			Duration: time.Since(startTime),
		}, nil
	}

	return &OptimizationResult{
		Success:           true,
		OriginalIR:        irCode,
		OptimizedIR:       optimizedIR,
		OptimizationLevel: level,
		Duration:          time.Since(startTime),
	}, nil
}

// checkLLVMAvailability 检查LLVM工具可用性
func (g *LLVMIRGenerator) checkLLVMAvailability(ctx context.Context) error {
	// Check clang
	if err := g.checkTool(ctx, "clang", "--version"); err != nil {
		return fmt.Errorf("clang check failed: %w", err)
	}

	// Check opt
	if err := g.checkTool(ctx, "opt", "--version"); err != nil {
		return fmt.Errorf("opt check failed: %w", err)
	}

	return nil
}

// checkTool 检查工具可用性
func (g *LLVMIRGenerator) checkTool(ctx context.Context, tool, args string) error {
	cmd := exec.CommandContext(ctx, tool, args)
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	outputStr := strings.TrimSpace(string(output))
	if !strings.Contains(outputStr, tool) {
		return fmt.Errorf("unexpected %s output: %s", tool, outputStr)
	}

	return nil
}

// createTempSourceFile 创建临时源文件
func (g *LLVMIRGenerator) createTempSourceFile(sourceCode string) (string, error) {
	// In real implementation, create a temporary file
	return "/tmp/echo_source_temp.c", nil
}

// createTempIRFile 创建临时IR文件
func (g *LLVMIRGenerator) createTempIRFile(irCode string) (string, error) {
	// In real implementation, create a temporary file
	return "/tmp/echo_ir_temp.ll", nil
}

// cleanupTempFile 清理临时文件
func (g *LLVMIRGenerator) cleanupTempFile(filePath string) {
	// In real implementation, delete the file
}

// compileToIR 编译为IR
func (g *LLVMIRGenerator) compileToIR(ctx context.Context, sourceFile string, options IROptions) (string, error) {
	args := []string{
		"-S",         // Emit assembly
		"-emit-llvm", // Emit LLVM IR
		"-o", "-",
		sourceFile,
	}

	// Add optimization level
	optLevel := fmt.Sprintf("-O%d", g.optimizationLevel)
	args = append(args, optLevel)

	cmd := exec.CommandContext(ctx, "clang", args...)
	cmd.Dir = g.workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("clang compilation failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// runOptimization 运行优化
func (g *LLVMIRGenerator) runOptimization(ctx context.Context, irFile string, level int) (string, error) {
	args := []string{
		fmt.Sprintf("-O%d", level),
		"-S",
		irFile,
		"-o", "-",
	}

	cmd := exec.CommandContext(ctx, "opt", args...)
	cmd.Dir = g.workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("opt optimization failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// validateIR 验证IR
func (g *LLVMIRGenerator) validateIR(irCode string) error {
	// Basic validation - check if it contains LLVM IR markers
	if !strings.Contains(irCode, "target triple") && !strings.Contains(irCode, "define") {
		return fmt.Errorf("generated code does not appear to be valid LLVM IR")
	}
	return nil
}

// SetOptimizationLevel 设置优化级别
func (g *LLVMIRGenerator) SetOptimizationLevel(level int) {
	if level >= 0 && level <= 3 {
		g.optimizationLevel = level
	}
}

// IsInitialized 返回是否已初始化
func (g *LLVMIRGenerator) IsInitialized() bool {
	return g.isInitialized
}

// IROptions IR生成选项
type IROptions struct {
	TargetTriple   string
	DataLayout     string
	AdditionalArgs []string
}

// IRGenerationResult IR生成结果
type IRGenerationResult struct {
	Success           bool
	IRCode            string
	SourceFile        string
	Errors            []string
	Warnings          []string
	Duration          time.Duration
	OptimizationLevel int
}

// OptimizationResult 优化结果
type OptimizationResult struct {
	Success           bool
	OriginalIR        string
	OptimizedIR       string
	Errors            []string
	Duration          time.Duration
	OptimizationLevel int
}
