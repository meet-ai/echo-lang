// Package services 实现现代化前端服务
package services

import (
	"context"
	"fmt"
	"os"
	"time"

	"echo/internal/modules/frontend/domain/commands"
	"echo/internal/modules/frontend/domain/dtos"
	"echo/internal/modules/frontend/domain/parser"
	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
)

// IModernFrontendService 现代化前端服务接口
// 使用新的三层混合解析器架构
type IModernFrontendService interface {
	// ParseSourceCode 解析源代码（使用现代化解析器架构）
	ParseSourceCode(ctx context.Context, sourceCode string, filename string) (*dtos.ModernCompilationResult, error)

	// PerformLexicalAnalysis 执行词法分析（使用高级词法分析器）
	PerformLexicalAnalysis(ctx context.Context, cmd commands.PerformLexicalAnalysisCommand) (*commands.LexicalAnalysisResult, error)

	// PerformSyntaxAnalysis 执行语法分析（使用现代化解析器）
	PerformSyntaxAnalysis(ctx context.Context, cmd commands.PerformSyntaxAnalysisCommand) (*commands.SyntaxAnalysisResult, error)

	// CompileFile 编译文件（完整流程）
	CompileFile(ctx context.Context, filePath string) (*dtos.ModernCompilationResult, error)
}

// modernFrontendService 现代化前端服务实现
type modernFrontendService struct {
	// 现代化解析器聚合根
	parserAggregate *parser.ModernParserAggregate

	// 事件发布器（可选）
	eventPublisher EventPublisher
}

// NewModernFrontendService 创建现代化前端服务
func NewModernFrontendService(
	eventPublisher EventPublisher,
) IModernFrontendService {
	return &modernFrontendService{
		parserAggregate: parser.NewModernParserAggregate(),
		eventPublisher:  eventPublisher,
	}
}

// ParseSourceCode 解析源代码
// 使用现代化三层混合解析器架构
func (s *modernFrontendService) ParseSourceCode(
	ctx context.Context,
	sourceCode string,
	filename string,
) (*dtos.ModernCompilationResult, error) {

	startTime := time.Now()

	// 使用现代化解析器聚合根解析源代码
	programAST, err := s.parserAggregate.Parse(ctx, sourceCode, filename)
	if err != nil {
		return s.buildErrorResult(err, filename, time.Since(startTime)), nil
	}

	// 收集解析错误
	errors := s.parserAggregate.GetErrors()

	// 构建结果
	result := &dtos.ModernCompilationResult{
		Success:      len(errors) == 0,
		Filename:     filename,
		ProgramAST:   programAST,
		Errors:       errors,
		Duration:     time.Since(startTime),
		ParserType:   "modern_hybrid", // 三层混合解析器
		ParseDetails: s.buildParseDetails(s.parserAggregate),
	}

	// 发布解析完成事件（如果配置了事件发布器）
	if s.eventPublisher != nil {
		event := &AnalysisCompletedEvent{
			EventID:      generateModernEventID(),
			EventType:    "modern_parsing_completed",
			Timestamp:    time.Now(),
			SourceFileID: filename,
			AnalysisType: "modern_syntax_analysis",
			Success:      result.Success,
			Duration:     result.Duration,
		}
		if !result.Success {
			event.Error = fmt.Sprintf("%d errors found", len(errors))
		}
		_ = s.eventPublisher.Publish(ctx, event)
	}

	return result, nil
}

// PerformLexicalAnalysis 执行词法分析
// 使用高级词法分析器
func (s *modernFrontendService) PerformLexicalAnalysis(
	ctx context.Context,
	cmd commands.PerformLexicalAnalysisCommand,
) (*commands.LexicalAnalysisResult, error) {

	startTime := time.Now()

	// 创建新的解析器聚合根实例（用于词法分析）
	lexerAggregate := parser.NewModernParserAggregate()

	// 执行词法分析（只执行词法分析阶段）
	_, err := lexerAggregate.Parse(ctx, cmd.SourceCode, cmd.SourceFilePath)
	if err != nil {
		return &commands.LexicalAnalysisResult{
			Success:      false,
			SourceFileID: cmd.SourceFileID,
			Duration:     time.Since(startTime),
		}, err
	}

	// 获取Token流（通过解析上下文）
	parsingContext := lexerAggregate.GetParsingContext()
	if parsingContext == nil {
		return &commands.LexicalAnalysisResult{
			Success:      false,
			SourceFileID: cmd.SourceFileID,
			Duration:     time.Since(startTime),
		}, fmt.Errorf("parsing context not available")
	}

	tokenStream := parsingContext.TokenStream()
	tokens := tokenStream.Tokens()

	// 构建结果
	result := &commands.LexicalAnalysisResult{
		Success:      true,
		SourceFileID: cmd.SourceFileID,
		TokenCount:   len(tokens),
		Duration:     time.Since(startTime),
	}

	return result, nil
}

// PerformSyntaxAnalysis 执行语法分析
// 使用现代化解析器架构
// 注意：此方法需要先执行词法分析，或者从SourceFileID读取文件
func (s *modernFrontendService) PerformSyntaxAnalysis(
	ctx context.Context,
	cmd commands.PerformSyntaxAnalysisCommand,
) (*commands.SyntaxAnalysisResult, error) {

	startTime := time.Now()

	// 从文件路径读取源代码（假设SourceFileID是文件路径）
	sourceCode, err := readFileContent(cmd.SourceFileID)
	if err != nil {
		return &commands.SyntaxAnalysisResult{
			Success:      false,
			SourceFileID: cmd.SourceFileID,
			Duration:     time.Since(startTime),
		}, err
	}

	// 使用现代化解析器聚合根解析
	programAST, err := s.parserAggregate.Parse(ctx, sourceCode, cmd.SourceFileID)
	if err != nil {
		return &commands.SyntaxAnalysisResult{
			Success:      false,
			SourceFileID: cmd.SourceFileID,
			Duration:     time.Since(startTime),
		}, err
	}

	// 收集解析错误
	errors := s.parserAggregate.GetErrors()

	// 构建结果
	result := &commands.SyntaxAnalysisResult{
		Success:      len(errors) == 0,
		SourceFileID: cmd.SourceFileID,
		Duration:     time.Since(startTime),
	}

	// 验证解析结果
	if programAST == nil {
		result.Success = false
	}

	return result, nil
}

// CompileFile 编译文件
// 完整编译流程（词法分析 + 语法分析）
func (s *modernFrontendService) CompileFile(
	ctx context.Context,
	filePath string,
) (*dtos.ModernCompilationResult, error) {

	// 读取文件内容
	sourceCode, err := readFileContent(filePath)
	if err != nil {
		return &dtos.ModernCompilationResult{
			Success:  false,
			Filename: filePath,
			Errors: []*sharedVO.ParseError{
			sharedVO.NewParseError(
				fmt.Sprintf("failed to read file: %v", err),
				sharedVO.NewSourceLocation(filePath, 0, 0, 0),
				sharedVO.ErrorTypeSyntax,
				sharedVO.SeverityError,
			),
			},
		}, nil
	}

	// 解析源代码
	return s.ParseSourceCode(ctx, sourceCode, filePath)
}

// buildErrorResult 构建错误结果
func (s *modernFrontendService) buildErrorResult(
	err error,
	filename string,
	duration time.Duration,
) *dtos.ModernCompilationResult {
	return &dtos.ModernCompilationResult{
		Success:  false,
		Filename: filename,
		Errors: []*sharedVO.ParseError{
			sharedVO.NewParseError(
				err.Error(),
				sharedVO.NewSourceLocation(filename, 0, 0, 0),
				sharedVO.ErrorTypeSyntax,
				sharedVO.SeverityError,
			),
		},
		Duration:   duration,
		ParserType: "modern_hybrid",
	}
}

// buildParseDetails 构建解析详情
func (s *modernFrontendService) buildParseDetails(
	aggregate *parser.ModernParserAggregate,
) *dtos.ParseDetails {
	parsingContext := aggregate.GetParsingContext()
	if parsingContext == nil {
		return &dtos.ParseDetails{
			ParserType: "modern_hybrid",
		}
	}

	return &dtos.ParseDetails{
		ParserType:        "modern_hybrid",
		CurrentParserType: string(parsingContext.CurrentParserType()),
		StateStackDepth:   parsingContext.StateStackDepth(),
		InRecoveryMode:    parsingContext.IsInRecoveryMode(),
	}
}

// countASTNodes 统计AST节点数量
func (s *modernFrontendService) countASTNodes(programAST *sharedVO.ProgramAST) int {
	if programAST == nil {
		return 0
	}

	count := 1 // 程序根节点
	
	// 递归统计所有顶层节点及其子节点
	for _, node := range programAST.Nodes() {
		count += s.countNodeRecursive(node)
	}
	
	// 统计模块声明
	if programAST.ModuleDeclaration() != nil {
		count += s.countNodeRecursive(programAST.ModuleDeclaration())
	}
	
	// 统计导入语句
	for _, importStmt := range programAST.Imports() {
		count += s.countNodeRecursive(importStmt)
	}
	
	return count
}

// countNodeRecursive 递归统计单个节点及其所有子节点
func (s *modernFrontendService) countNodeRecursive(node sharedVO.ASTNode) int {
	if node == nil {
		return 0
	}
	
	count := 1 // 当前节点
	
	// 根据节点类型递归统计子节点
	switch n := node.(type) {
	case *sharedVO.BlockStatement:
		// 统计块语句中的所有语句
		for _, stmt := range n.Statements() {
			count += s.countNodeRecursive(stmt)
		}
		
	case *sharedVO.FunctionDeclaration:
		// 统计函数体的子节点
		if n.Body() != nil {
			count += s.countNodeRecursive(n.Body())
		}
		// 统计参数
		for _, param := range n.Parameters() {
			count += s.countNodeRecursive(param)
		}
		// 统计泛型参数
		for _, gp := range n.GenericParams() {
			count += s.countNodeRecursive(gp)
		}
		// 统计返回类型
		if n.ReturnType() != nil {
			count += s.countNodeRecursive(n.ReturnType())
		}
		
	case *sharedVO.AsyncFunctionDeclaration:
		// 统计异步函数的基础函数声明
		if n.FunctionDeclaration() != nil {
			count += s.countNodeRecursive(n.FunctionDeclaration())
		}
		
	case *sharedVO.StructDeclaration:
		// 统计结构体字段
		for _, field := range n.Fields() {
			count += s.countNodeRecursive(field)
		}
		
	case *sharedVO.EnumDeclaration:
		// 统计枚举变体
		for _, variant := range n.Variants() {
			count += s.countNodeRecursive(variant)
		}
		
	case *sharedVO.TraitDeclaration:
		// 统计Trait方法
		for _, method := range n.Methods() {
			count += s.countNodeRecursive(method)
		}
		// 统计泛型参数
		for _, gp := range n.GenericParams() {
			count += s.countNodeRecursive(gp)
		}
		
	case *sharedVO.ImplDeclaration:
		// 统计实现的方法
		for _, method := range n.Methods() {
			count += s.countNodeRecursive(method)
		}
		// 统计Trait类型和目标类型
		if n.TraitType() != nil {
			count += s.countNodeRecursive(n.TraitType())
		}
		if n.TargetType() != nil {
			count += s.countNodeRecursive(n.TargetType())
		}
		
	case *sharedVO.ReturnStatement:
		// 统计返回表达式
		if n.Expression() != nil {
			count += s.countNodeRecursive(n.Expression())
		}
		
	case *sharedVO.VariableDeclaration:
		// 统计初始化表达式
		if n.Initializer() != nil {
			count += s.countNodeRecursive(n.Initializer())
		}
		// 统计类型注解
		if n.VarType() != nil {
			count += s.countNodeRecursive(n.VarType())
		}
		
	case *sharedVO.BinaryExpression:
		// 统计左右操作数
		if n.Left() != nil {
			count += s.countNodeRecursive(n.Left())
		}
		if n.Right() != nil {
			count += s.countNodeRecursive(n.Right())
		}
		
	case *sharedVO.UnaryExpression:
		// 统计操作数
		if n.Operand() != nil {
			count += s.countNodeRecursive(n.Operand())
		}
		
	case *sharedVO.TypeAnnotation:
		// 统计泛型参数
		for _, arg := range n.GenericArgs() {
			count += s.countNodeRecursive(arg)
		}
		
	case *sharedVO.Parameter:
		// 统计参数类型注解
		if n.TypeAnnotation() != nil {
			count += s.countNodeRecursive(n.TypeAnnotation())
		}
		
	case *sharedVO.StructField:
		// 统计字段类型注解
		if n.FieldType() != nil {
			count += s.countNodeRecursive(n.FieldType())
		}
		
	case *sharedVO.TraitMethod:
		// 统计参数
		for _, param := range n.Parameters() {
			count += s.countNodeRecursive(param)
		}
		// 统计返回类型
		if n.ReturnType() != nil {
			count += s.countNodeRecursive(n.ReturnType())
		}
		
	case *sharedVO.ObjectProperty:
		// 统计属性值
		if n.Value() != nil {
			count += s.countNodeRecursive(n.Value())
		}
		
	case *sharedVO.ExpressionStatement:
		// 统计表达式
		if n.Expression() != nil {
			count += s.countNodeRecursive(n.Expression())
		}
		
	case *sharedVO.TypeAliasDeclaration:
		// 统计原始类型
		if n.OriginalType() != nil {
			count += s.countNodeRecursive(n.OriginalType())
		}
	}
	
	return count
}

// readFileContent 读取文件内容
func readFileContent(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(data), nil
}

// generateModernEventID 生成现代化事件ID
func generateModernEventID() string {
	return fmt.Sprintf("modern_event_%d", time.Now().UnixNano())
}
