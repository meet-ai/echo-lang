package parser

import (
	"context"
	"fmt"

	"echo/internal/modules/frontend/domain/shared/value_objects"
	portServices "echo/internal/modules/frontend/ports/services"
)

// ParserApplicationServiceImpl 解析应用服务实现
// 职责：编排词法分析 -> 语法分析 -> 语义分析（通过 ProgramSemanticAnalyzer）的完整流程
type ParserApplicationServiceImpl struct {
	lexicalService           portServices.LexicalAnalysisService
	syntaxService            portServices.SyntaxAnalysisService
	programSemanticAnalyzer  portServices.ProgramSemanticAnalyzer
}

// NewParserApplicationService 创建解析应用服务实现
func NewParserApplicationService(
	lexicalService portServices.LexicalAnalysisService,
	syntaxService portServices.SyntaxAnalysisService,
	programSemanticAnalyzer portServices.ProgramSemanticAnalyzer,
) *ParserApplicationServiceImpl {
	return &ParserApplicationServiceImpl{
		lexicalService:          lexicalService,
		syntaxService:           syntaxService,
		programSemanticAnalyzer: programSemanticAnalyzer,
	}
}

// ParseSourceFile 解析源文件（完整流程）
// 流程：Tokenize -> ParseAST -> AnalyzeSemantics
func (p *ParserApplicationServiceImpl) ParseSourceFile(
	ctx context.Context,
	sourceFile *value_objects.SourceFile,
) (*value_objects.ParseResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if sourceFile == nil {
		return nil, fmt.Errorf("sourceFile is nil")
	}

	// 1. 词法分析
	tokenStream, err := p.lexicalService.Tokenize(ctx, sourceFile)
	if err != nil {
		return nil, fmt.Errorf("lexical analysis failed: %w", err)
	}

	// 2. 语法分析
	programAST, err := p.syntaxService.ParseAST(ctx, tokenStream)
	if err != nil {
		return nil, fmt.Errorf("syntax analysis failed: %w", err)
	}

	// 3. 语义分析（通过 Application 层编排的三阶段流水线）
	semanticResult, err := p.programSemanticAnalyzer.AnalyzeProgram(ctx, programAST)
	if err != nil {
		return nil, fmt.Errorf("semantic analysis failed: %w", err)
	}

	parseResult := value_objects.NewParseResult(sourceFile)
	parseResult.SetAST(programAST)
	if semanticResult != nil {
		if st := semanticResult.SymbolTable(); st != nil {
			parseResult.SetSymbolTable(st)
		}
		for _, e := range semanticResult.Errors() {
			parseResult.AddError(e)
		}
	}
	parseResult.Complete()

	return parseResult, nil
}

// ParseWithOptions 带选项的解析
// 当前实现直接复用 ParseSourceFile，后续根据 options 扩展（如：切换错误恢复策略、调试开关等）
func (p *ParserApplicationServiceImpl) ParseWithOptions(
	ctx context.Context,
	sourceFile *value_objects.SourceFile,
	options *value_objects.ParseOptions,
) (*value_objects.ParseResult, error) {
	// TODO: 根据 options 控制具体步骤（例如：是否启用高级错误恢复、是否只做语法检查等）
	return p.ParseSourceFile(ctx, sourceFile)
}

// ValidateSourceFile 仅验证源文件（不生成完整语义结果）
// 默认流程：Tokenize -> ValidateTokens -> 可选 ParseAST 进行结构校验
func (p *ParserApplicationServiceImpl) ValidateSourceFile(
	ctx context.Context,
	sourceFile *value_objects.SourceFile,
) (*value_objects.ValidationResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if sourceFile == nil {
		return nil, fmt.Errorf("sourceFile is nil")
	}

	// 1. 词法分析
	tokenStream, err := p.lexicalService.Tokenize(ctx, sourceFile)
	if err != nil {
		return nil, fmt.Errorf("lexical analysis failed: %w", err)
	}

	// 2. Token 级验证
	validationResult, err := p.lexicalService.ValidateTokens(ctx, tokenStream)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// TODO: 需要时，可以在这里追加语法级/语义级的轻量验证

	return validationResult, nil
}

