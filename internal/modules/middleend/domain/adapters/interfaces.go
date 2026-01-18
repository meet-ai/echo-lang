package adapters

import (
	"context"
)

// LLVMIRAdapter LLVM IR适配器接口
// 职责：适配不同的LLVM IR操作
type LLVMIRAdapter interface {
	// GenerateFromIR 将通用IR转换为LLVM IR
	GenerateFromIR(ctx context.Context, irCode string) (string, error)

	// ValidateLLVMIR 验证LLVM IR的正确性
	ValidateLLVMIR(ctx context.Context, llvmIR string) error

	// OptimizeLLVMIR 优化LLVM IR
	OptimizeLLVMIR(ctx context.Context, llvmIR string, optimizations []string) (string, error)

	// ConvertToTarget 转换为目标格式（如object文件）
	ConvertToTarget(ctx context.Context, llvmIR string, targetFormat string) ([]byte, error)
}

// WebAssemblyAdapter WebAssembly适配器接口
// 职责：处理WebAssembly相关的操作
type WebAssemblyAdapter interface {
	// GenerateFromIR 将通用IR转换为WebAssembly
	GenerateFromIR(ctx context.Context, irCode string) ([]byte, error)

	// ValidateWASM 验证WebAssembly模块
	ValidateWASM(ctx context.Context, wasmBytes []byte) error

	// OptimizeWASM 优化WebAssembly模块
	OptimizeWASM(ctx context.Context, wasmBytes []byte) ([]byte, error)
}

// TargetCodeAdapter 目标代码适配器通用接口
// 职责：统一不同目标代码格式的适配
type TargetCodeAdapter interface {
	// TargetFormat 返回支持的目标格式
	TargetFormat() string

	// Generate 生成目标代码
	Generate(ctx context.Context, irCode string) (interface{}, error)

	// Validate 验证目标代码
	Validate(ctx context.Context, code interface{}) error

	// Optimize 优化目标代码
	Optimize(ctx context.Context, code interface{}, optimizations []string) (interface{}, error)
}
