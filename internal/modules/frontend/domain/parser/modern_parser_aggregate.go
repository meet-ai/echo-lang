// Package parser 实现现代化解析器聚合根
package parser

import (
	"context"
	"fmt"
	"time"

	lexicalServices "echo/internal/modules/frontend/domain/lexical/services"
	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
)

// ModernParserAggregate 现代化解析器聚合根
// 职责：协调词法分析、语法分析、错误恢复的完整解析流程
// 采用三层混合解析器架构：递归下降（顶层结构）、Pratt（表达式）、LR（歧义处理）
type ModernParserAggregate struct {
	id string

	// 词法分析服务
	lexerService *lexicalServices.AdvancedLexerService

	// 解析协调器
	coordinator *ParserCoordinator

	// 解析结果
	programAST *sharedVO.ProgramAST

	// 解析状态
	isInitialized bool
	isParsed      bool
}

// NewModernParserAggregate 创建新的现代化解析器聚合根
func NewModernParserAggregate() *ModernParserAggregate {
	return &ModernParserAggregate{
		id:            generateAggregateID(),
		lexerService:  lexicalServices.NewAdvancedLexerService(),
		coordinator:   NewParserCoordinator(),
		isInitialized: false,
		isParsed:      false,
	}
}

// ID 获取聚合根ID
func (mpa *ModernParserAggregate) ID() string {
	return mpa.id
}

// Parse 解析源代码
// 这是聚合根的主入口，协调完整的解析流程
func (mpa *ModernParserAggregate) Parse(
	ctx context.Context,
	sourceCode string,
	filename string,
) (*sharedVO.ProgramAST, error) {

	// 步骤1：词法分析
	tokenStream, err := mpa.performLexicalAnalysis(ctx, sourceCode, filename)
	if err != nil {
		return nil, fmt.Errorf("lexical analysis failed: %w", err)
	}

	// 步骤2：初始化解析协调器
	sourceFile := lexicalVO.NewSourceFile(filename, sourceCode)
	mpa.coordinator.Initialize(sourceFile, tokenStream)
	mpa.isInitialized = true

	// 步骤3：语法分析（通过解析协调器）
	programAST, err := mpa.coordinator.ParseProgram(ctx)
	if err != nil {
		return nil, fmt.Errorf("syntax analysis failed: %w", err)
	}

	// 步骤4：保存解析结果
	mpa.programAST = programAST
	mpa.isParsed = true

	return programAST, nil
}

// performLexicalAnalysis 执行词法分析
func (mpa *ModernParserAggregate) performLexicalAnalysis(
	ctx context.Context,
	sourceCode string,
	filename string,
) (*lexicalVO.EnhancedTokenStream, error) {

	// 创建源文件对象
	sourceFile := lexicalVO.NewSourceFile(filename, sourceCode)

	// 执行词法分析
	tokenStream, err := mpa.lexerService.Tokenize(ctx, sourceFile)
	if err != nil {
		return nil, fmt.Errorf("tokenization failed: %w", err)
	}

	return tokenStream, nil
}

// ParseExpression 解析表达式
// 委托给解析协调器，自动选择合适的解析器
func (mpa *ModernParserAggregate) ParseExpression(
	ctx context.Context,
) (sharedVO.ASTNode, error) {

	if !mpa.isInitialized {
		return nil, fmt.Errorf("parser not initialized, call Parse() first")
	}

	return mpa.coordinator.ParseExpression(ctx)
}

// GetProgramAST 获取解析后的程序AST
func (mpa *ModernParserAggregate) GetProgramAST() *sharedVO.ProgramAST {
	return mpa.programAST
}

// GetParsingContext 获取解析上下文
func (mpa *ModernParserAggregate) GetParsingContext() *ParsingContext {
	if mpa.coordinator == nil {
		return nil
	}
	return mpa.coordinator.GetCurrentContext()
}

// GetErrors 获取所有解析错误
func (mpa *ModernParserAggregate) GetErrors() []*sharedVO.ParseError {
	if mpa.coordinator == nil {
		return nil
	}
	return mpa.coordinator.collectAllErrors()
}

// HasErrors 检查是否有错误
func (mpa *ModernParserAggregate) HasErrors() bool {
	errors := mpa.GetErrors()
	return len(errors) > 0
}

// IsParsed 检查是否已解析
func (mpa *ModernParserAggregate) IsParsed() bool {
	return mpa.isParsed
}

// Reset 重置聚合根状态
func (mpa *ModernParserAggregate) Reset() {
	mpa.programAST = nil
	mpa.isInitialized = false
	mpa.isParsed = false
	if mpa.coordinator != nil {
		mpa.coordinator.Reset()
	}
}

// OrchestrateParsingFlow 编排解析流程
// 这是解析流程编排的核心方法，协调三种解析器的工作
func (mpa *ModernParserAggregate) OrchestrateParsingFlow(
	ctx context.Context,
) error {

	if !mpa.isInitialized {
		return fmt.Errorf("parser not initialized")
	}

	parsingContext := mpa.coordinator.GetCurrentContext()
	if parsingContext == nil {
		return fmt.Errorf("parsing context not available")
	}

	// 阶段1：顶层递归下降解析
	// 解析模块、函数、类等顶级结构
	err := mpa.orchestrateTopLevelParsing(ctx, parsingContext)
	if err != nil {
		return fmt.Errorf("top-level parsing failed: %w", err)
	}

	// 阶段2：表达式解析（Pratt）
	// 在需要解析表达式时，自动切换到Pratt解析器
	// 这个阶段在递归下降解析过程中自动触发

	// 阶段3：歧义处理（LR）
	// 当遇到歧义时，自动切换到LR解析器
	// 这个阶段在Pratt解析过程中自动触发

	return nil
}

// orchestrateTopLevelParsing 编排顶层解析流程
func (mpa *ModernParserAggregate) orchestrateTopLevelParsing(
	ctx context.Context,
	parsingContext *ParsingContext,
) error {

	// 切换到递归下降解析器
	err := mpa.coordinator.SwitchToRecursiveDescent()
	if err != nil {
		return fmt.Errorf("failed to switch to recursive descent parser: %w", err)
	}

	// 推入顶层解析状态
	parsingContext.PushState(ParserTypeRecursiveDescent, "top_level_parsing")

	// 顶层解析由递归下降解析器完成
	// 在解析过程中，如果需要解析表达式，会自动切换到Pratt解析器
	// 如果遇到歧义，会自动切换到LR解析器

	return nil
}

// SwitchToExpressionParsing 切换到表达式解析模式
// 当需要解析表达式时调用
func (mpa *ModernParserAggregate) SwitchToExpressionParsing() error {
	if !mpa.isInitialized {
		return fmt.Errorf("parser not initialized")
	}

	// 切换到Pratt解析器
	return mpa.coordinator.SwitchToPratt()
}

// SwitchToAmbiguityResolution 切换到歧义解析模式
// 当遇到歧义时调用
func (mpa *ModernParserAggregate) SwitchToAmbiguityResolution() error {
	if !mpa.isInitialized {
		return fmt.Errorf("parser not initialized")
	}

	// 切换到LR解析器
	return mpa.coordinator.SwitchToLR()
}

// SaveCheckpoint 保存解析检查点
func (mpa *ModernParserAggregate) SaveCheckpoint() {
	if mpa.coordinator != nil {
		mpa.coordinator.SaveCheckpoint()
	}
}

// RestoreCheckpoint 恢复解析检查点
func (mpa *ModernParserAggregate) RestoreCheckpoint() error {
	if mpa.coordinator == nil {
		return fmt.Errorf("coordinator not initialized")
	}
	return mpa.coordinator.RestoreCheckpoint()
}

// String 返回聚合根的字符串表示
func (mpa *ModernParserAggregate) String() string {
	status := "未初始化"
	if mpa.isInitialized {
		status = "已初始化"
	}
	if mpa.isParsed {
		status = "已解析"
	}

	return fmt.Sprintf("ModernParserAggregate[ID:%s, Status:%s, HasErrors:%t]",
		mpa.id, status, mpa.HasErrors())
}

// generateAggregateID 生成聚合根ID
func generateAggregateID() string {
	return fmt.Sprintf("parser_%d", time.Now().UnixNano())
}
