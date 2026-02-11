// Package services 定义领域服务接口
package services

import (
	"context"

	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
	"echo/internal/modules/frontend/domain/shared/value_objects"
)

// LexicalAnalysisService 词法分析服务接口
// 职责：将源代码转换为Token流
type LexicalAnalysisService interface {
	// Tokenize 将源文件进行词法分析，返回Token流
	Tokenize(ctx context.Context, sourceFile *value_objects.SourceFile) (*value_objects.TokenStream, error)

	// ValidateTokens 验证Token流的语法正确性
	ValidateTokens(ctx context.Context, tokenStream *value_objects.TokenStream) (*value_objects.ValidationResult, error)
}

// SyntaxAnalysisService 语法分析服务接口
// 职责：将Token流转换为AST
type SyntaxAnalysisService interface {
	// ParseAST 将Token流解析为抽象语法树
	ParseAST(ctx context.Context, tokenStream *value_objects.TokenStream) (*value_objects.ProgramAST, error)

	// ValidateAST 验证AST的语义正确性
	ValidateAST(ctx context.Context, ast *value_objects.ProgramAST) (*value_objects.ValidationResult, error)

	// ResolveAmbiguity 解决语法歧义（如运算符优先级）
	ResolveAmbiguity(ctx context.Context, ambiguousTokens *value_objects.TokenStream) (*value_objects.ResolvedAST, error)
}

// AdvancedErrorRecoveryPort 高级错误恢复端口（EnhancedTokenStream 路径）
// 供 ParserCoordinator 等使用，由 Application 层实现并委托 AdvancedErrorRecoveryService
type AdvancedErrorRecoveryPort interface {
	RecoverFromError(ctx context.Context, parseError *value_objects.ParseError, tokenStream *lexicalVO.EnhancedTokenStream) (*value_objects.ErrorRecoveryResult, error)
}

// ErrorRecoveryService 错误恢复服务接口
// 职责：处理解析过程中的错误并尝试恢复
type ErrorRecoveryService interface {
	// RecoverFromError 从解析错误中恢复
	RecoverFromError(ctx context.Context, parseContext *value_objects.ParseContext) (*value_objects.RecoveryResult, error)

	// SuggestFixes 为解析错误提供修复建议
	SuggestFixes(ctx context.Context, parseError *value_objects.ParseError) ([]*value_objects.FixSuggestion, error)

	// ApplyRecovery 应用恢复策略
	ApplyRecovery(ctx context.Context, recoveryStrategy *value_objects.RecoveryStrategy) (*value_objects.AppliedRecovery, error)
}

// ProgramSemanticAnalyzer 语义分析应用层端口：编排“构建符号表→类型检查→符号解析”三阶段，供 ParserApplicationService 调用
type ProgramSemanticAnalyzer interface {
	AnalyzeProgram(ctx context.Context, programAST *value_objects.ProgramAST) (*value_objects.SemanticAnalysisResult, error)
}

// SemanticAnalysisService 语义分析服务接口
// 职责：进行类型检查和符号表管理
type SemanticAnalysisService interface {
	// AnalyzeSemantics 对AST进行语义分析
	AnalyzeSemantics(ctx context.Context, ast *value_objects.ProgramAST) (*value_objects.AnalyzedProgram, error)

	// BuildSymbolTable 构建符号表
	BuildSymbolTable(ctx context.Context, ast *value_objects.ProgramAST) (*value_objects.SymbolTable, error)

	// CheckTypes 执行类型检查
	CheckTypes(ctx context.Context, analyzedProgram *value_objects.AnalyzedProgram) (*value_objects.TypeCheckResult, error)

	// ResolveSymbols 解析符号引用
	ResolveSymbols(ctx context.Context, symbolTable *value_objects.SymbolTable) (*value_objects.ResolvedSymbols, error)
}

// ParserApplicationService 解析器应用服务接口
// 职责：协调各领域服务完成完整的解析流程
type ParserApplicationService interface {
	// ParseSourceFile 解析源文件（完整流程）
	ParseSourceFile(ctx context.Context, sourceFile *value_objects.SourceFile) (*value_objects.ParseResult, error)

	// ParseWithOptions 带选项的解析
	ParseWithOptions(ctx context.Context, sourceFile *value_objects.SourceFile, options *value_objects.ParseOptions) (*value_objects.ParseResult, error)

	// ValidateSourceFile 仅验证源文件（不生成AST）
	ValidateSourceFile(ctx context.Context, sourceFile *value_objects.SourceFile) (*value_objects.ValidationResult, error)
}
