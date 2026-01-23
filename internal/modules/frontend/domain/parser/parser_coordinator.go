// Package parser 实现解析协调器
package parser

import (
	"context"
	"fmt"
	"strings"

	errorRecoveryServices "echo/internal/modules/frontend/domain/error_recovery/services"
	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
	syntaxServices "echo/internal/modules/frontend/domain/syntax/services"
	syntaxVO "echo/internal/modules/frontend/domain/syntax/value_objects"
)

// ParserCoordinator 解析协调器
// 负责协调递归下降、Pratt、LR三种解析器的工作
type ParserCoordinator struct {
	// 解析上下文
	parsingContext *ParsingContext

	// 三种解析器
	recursiveDescentParser *syntaxServices.RecursiveDescentParser
	prattParser            *syntaxServices.PrattExpressionParser
	lrResolver             *syntaxServices.LRAmbiguityResolver

	// 错误恢复服务
	errorRecoveryService *errorRecoveryServices.AdvancedErrorRecoveryService

	// 状态检查点（用于回退）
	checkpoints []*ParsingCheckpoint
}

// NewParserCoordinator 创建新的解析协调器
func NewParserCoordinator() *ParserCoordinator {
	return &ParserCoordinator{
		recursiveDescentParser: syntaxServices.NewRecursiveDescentParser(),
		prattParser:            syntaxServices.NewPrattExpressionParser(),
		lrResolver:             syntaxServices.NewLRAmbiguityResolver(),
		errorRecoveryService:   errorRecoveryServices.NewAdvancedErrorRecoveryService(),
		checkpoints:            make([]*ParsingCheckpoint, 0),
	}
}

// Initialize 初始化解析协调器
func (pc *ParserCoordinator) Initialize(
	sourceFile *lexicalVO.SourceFile,
	tokenStream *lexicalVO.EnhancedTokenStream,
) {
	pc.parsingContext = NewParsingContext(sourceFile, tokenStream)
}

// ParseProgram 解析程序
// 这是解析协调器的主入口，协调三种解析器完成解析
func (pc *ParserCoordinator) ParseProgram(ctx context.Context) (*sharedVO.ProgramAST, error) {
	if pc.parsingContext == nil {
		return nil, fmt.Errorf("parser coordinator not initialized")
	}

	// 使用递归下降解析器解析顶层结构
	programAST, err := pc.recursiveDescentParser.ParseProgram(
		ctx,
		pc.parsingContext.TokenStream(),
	)

	if err != nil {
		// 尝试错误恢复
		recoveryResult, recoveryErr := pc.handleParseError(ctx, err)
		if recoveryErr != nil {
			return nil, fmt.Errorf("parse error and recovery failed: %w, recovery: %v", err, recoveryErr)
		}

		if recoveryResult != nil && recoveryResult.HasRecovered() {
			// 恢复成功，继续解析
			// 这里可以重新尝试解析或调整解析位置
			_ = recoveryResult
		}
	}

	// 收集所有错误
	allErrors := pc.collectAllErrors()
	for _, err := range allErrors {
		// 将错误添加到程序AST（如果有错误收集机制）
		_ = err
	}

	return programAST, err
}

// SwitchToParser 切换到指定的解析器
func (pc *ParserCoordinator) SwitchToParser(parserType ParserType) error {
	if pc.parsingContext == nil {
		return fmt.Errorf("parsing context not initialized")
	}

	// 保存当前状态
	pc.SaveCheckpoint()

	// 切换到新解析器
	pc.parsingContext.SetCurrentParserType(parserType)

	// 推入状态
	pc.parsingContext.PushState(parserType, fmt.Sprintf("switched_to_%s", parserType))

	return nil
}

// SwitchToRecursiveDescent 切换到递归下降解析器
func (pc *ParserCoordinator) SwitchToRecursiveDescent() error {
	return pc.SwitchToParser(ParserTypeRecursiveDescent)
}

// SwitchToPratt 切换到Pratt表达式解析器
func (pc *ParserCoordinator) SwitchToPratt() error {
	return pc.SwitchToParser(ParserTypePratt)
}

// SwitchToLR 切换到LR歧义解析器
func (pc *ParserCoordinator) SwitchToLR() error {
	return pc.SwitchToParser(ParserTypeLR)
}

// ParseExpression 解析表达式
// 根据上下文自动选择合适的解析器
func (pc *ParserCoordinator) ParseExpression(ctx context.Context) (sharedVO.ASTNode, error) {
	if pc.parsingContext == nil {
		return nil, fmt.Errorf("parsing context not initialized")
	}

	currentParserType := pc.parsingContext.CurrentParserType()

	switch currentParserType {
	case ParserTypePratt:
		// 使用Pratt解析器解析表达式
		return pc.parseExpressionWithPratt(ctx)

	case ParserTypeLR:
		// 使用LR解析器解析歧义表达式
		return pc.parseExpressionWithLR(ctx)

	case ParserTypeRecursiveDescent:
		// 递归下降解析器通常不直接解析表达式，而是委托给Pratt
		// 临时切换到Pratt解析器
		pc.SwitchToPratt()
		return pc.parseExpressionWithPratt(ctx)

	default:
		// 默认使用Pratt解析器
		pc.SwitchToPratt()
		return pc.parseExpressionWithPratt(ctx)
	}
}

// parseExpressionWithPratt 使用Pratt解析器解析表达式
func (pc *ParserCoordinator) parseExpressionWithPratt(ctx context.Context) (sharedVO.ASTNode, error) {
	// 使用最小优先级开始解析
	minPrecedence := syntaxVO.PrecedencePrimary

	expression, err := pc.prattParser.ParseExpression(
		ctx,
		pc.parsingContext.TokenStream(),
		minPrecedence,
	)

	if err != nil {
		// 检查是否是歧义错误，如果是，尝试使用LR解析器
		if pc.isAmbiguityError(err) {
			return pc.parseExpressionWithLR(ctx)
		}
		return nil, err
	}

	return expression, nil
}

// parseExpressionWithLR 使用LR解析器解析歧义表达式
func (pc *ParserCoordinator) parseExpressionWithLR(ctx context.Context) (sharedVO.ASTNode, error) {
	currentPosition := pc.parsingContext.CurrentPosition()

	// 使用LR解析器解析歧义
	ambiguousAST, err := pc.lrResolver.ResolveAmbiguity(
		ctx,
		pc.parsingContext.TokenStream(),
		currentPosition,
	)

	if err != nil {
		return nil, err
	}

	if ambiguousAST != nil {
		// 完善实现：根据实际消耗的token数量前进位置
		// 方法1：尝试从AST节点的位置信息计算消耗的token数量（最精确）
		// 方法2：根据AST节点类型和结构估算消耗的token数量（备用方案）
		// 方法3：从tokenStream当前位置计算（如果LR解析器更新了位置）
		
		astNode := *ambiguousAST
		tokensConsumed := pc.calculateTokensConsumed(astNode, currentPosition)
		
		// 更新解析位置
		pc.parsingContext.AdvancePosition(tokensConsumed)
		return *ambiguousAST, nil
	}

	// LR解析器无法解析，返回错误
	return nil, fmt.Errorf("LR parser could not resolve ambiguity at position %d", currentPosition)
}

// isAmbiguityError 检查是否是歧义错误
func (pc *ParserCoordinator) isAmbiguityError(err error) bool {
	// 完善实现：智能分析错误类型和消息
	if err == nil {
		return false
	}
	
	errMsg := err.Error()
	errMsgLower := strings.ToLower(errMsg)
	
	// 检查错误消息中的歧义关键词
	ambiguityKeywords := []string{
		"ambiguity", "ambiguous", "歧义",
		"cannot determine", "unclear", "ambiguous operator",
		"operator precedence", "operator associativity",
		"shift/reduce", "reduce/reduce", // LR解析器冲突
	}
	
	for _, keyword := range ambiguityKeywords {
		if strings.Contains(errMsgLower, strings.ToLower(keyword)) {
			return true
		}
	}
	
	// 检查特定语法模式，这些模式通常表示歧义
	ambiguityPatterns := []string{
		"<", ">", // 泛型/比较运算符歧义
		"*",      // 指针/乘法运算符歧义
		"&",      // 引用/按位与运算符歧义
	}
	
	// 检查错误消息中是否包含这些模式，且上下文暗示歧义
	for _, pattern := range ambiguityPatterns {
		if strings.Contains(errMsg, pattern) {
			// 检查是否在歧义相关的上下文中
			if strings.Contains(errMsgLower, "operator") ||
				strings.Contains(errMsgLower, "type") ||
				strings.Contains(errMsgLower, "expression") {
				return true
			}
		}
	}
	
	// 检查是否是ParseError类型，并分析错误类型
	// 如果错误是语法错误且包含特定模式，可能是歧义
	if parseErr, ok := err.(*sharedVO.ParseError); ok {
		// 语法错误中，某些特定模式表示歧义
		if parseErr.ErrorType() == sharedVO.ErrorTypeSyntax {
			msg := strings.ToLower(parseErr.Message())
			// 检查错误消息是否包含运算符或类型相关的歧义
			if strings.Contains(msg, "operator") ||
				strings.Contains(msg, "type") ||
				strings.Contains(msg, "generic") ||
				strings.Contains(msg, "precedence") ||
				strings.Contains(msg, "associativity") {
				return true
			}
		}
	}
	
	// 检查错误位置附近的token，判断是否是常见的歧义场景
	if pc.parsingContext != nil && pc.parsingContext.TokenStream() != nil {
		currentPos := pc.parsingContext.CurrentPosition()
		tokenStream := pc.parsingContext.TokenStream()
		
		// 检查当前位置的token是否是容易产生歧义的token
		if currentPos < tokenStream.Count() {
			// 获取当前token（通过Tokens()方法）
			tokens := tokenStream.Tokens()
			if currentPos < len(tokens) {
				currentToken := tokens[currentPos]
				if currentToken != nil {
					tokenType := currentToken.Type()
					// 检查是否是容易产生歧义的token类型
					ambiguousTokenTypes := []string{
						"<", ">", "*", "&", // 运算符/类型歧义
					}
					for _, ambiguousType := range ambiguousTokenTypes {
						if string(tokenType) == ambiguousType {
							// 结合错误消息判断
							if strings.Contains(errMsgLower, "expected") ||
								strings.Contains(errMsgLower, "unexpected") {
								return true
							}
						}
					}
				}
			}
		}
	}
	
	return false
}

// calculateTokensConsumed 计算AST节点消耗的token数量
func (pc *ParserCoordinator) calculateTokensConsumed(astNode sharedVO.ASTNode, startPosition int) int {
	if astNode == nil {
		return 1 // 保守估计至少1个token
	}
	
	// 方法1：尝试从AST节点的位置信息计算（最精确）
	// 如果AST节点包含位置信息，可以通过位置计算消耗的token数量
	nodeLocation := astNode.Location()
	tokenStream := pc.parsingContext.TokenStream()
	
	if tokenStream != nil {
		// 尝试通过遍历token流，找到在节点位置范围内的token数量
		tokenCount := 0
		startOffset := nodeLocation.Offset()
		tokens := tokenStream.Tokens()
		
		// 遍历从起始位置开始的token，直到超出节点范围
		for i := startPosition; i < len(tokens) && i < tokenStream.Count(); i++ {
			token := tokens[i]
			if token == nil {
				break
			}
			tokenLocation := token.Location()
			tokenOffset := tokenLocation.Offset()
			
			// 如果token的偏移量在节点范围内，计入计数
			// 注意：这里使用简单的偏移量比较，实际可能需要更复杂的逻辑
			if tokenOffset >= startOffset {
				tokenCount++
				// 如果已经找到足够的token（防止无限循环），停止
				if tokenCount > 100 {
					break
				}
			} else if tokenOffset < startOffset {
				// token在节点之前，继续查找
				continue
			} else {
				// token超出节点范围，停止
				break
			}
		}
		
		if tokenCount > 0 {
			return tokenCount
		}
	}
	
	// 方法2：根据AST节点类型和结构估算消耗的token数量（备用方案）
	nodeType := astNode.NodeType()
	switch nodeType {
	case "BinaryExpression":
		// 二元表达式：left + operator + right，至少3个token
		// 实际可能更多，因为left和right可能是复杂表达式
		if be, ok := astNode.(*sharedVO.BinaryExpression); ok {
			leftTokens := pc.calculateTokensConsumed(be.Left(), startPosition)
			rightTokens := pc.calculateTokensConsumed(be.Right(), startPosition+leftTokens+1)
			return leftTokens + 1 + rightTokens // operator占1个token
		}
		return 3
	case "UnaryExpression":
		// 一元表达式：operator + operand，至少2个token
		if ue, ok := astNode.(*sharedVO.UnaryExpression); ok {
			operandTokens := pc.calculateTokensConsumed(ue.Operand(), startPosition+1)
			return 1 + operandTokens // operator占1个token
		}
		return 2
	case "FunctionCall", "CallExpression":
		// 函数调用：name + ( + args + )，至少3个token
		// 实际可能更多，取决于参数数量
		return 3
	case "GenericType", "TypeAnnotation":
		// 泛型类型：name + < + args + >，至少3个token
		// 实际可能更多，取决于类型参数数量
		return 3
	case "Identifier":
		// 标识符：1个token
		return 1
	case "IntegerLiteral", "FloatLiteral", "StringLiteral", "BooleanLiteral":
		// 字面量：1个token
		return 1
	default:
		// 其他类型，保守估计至少1个token
		return 1
	}
}

// SaveCheckpoint 保存检查点
func (pc *ParserCoordinator) SaveCheckpoint() {
	if pc.parsingContext == nil {
		return
	}

	checkpoint := pc.parsingContext.SaveCheckpoint()
	pc.checkpoints = append(pc.checkpoints, checkpoint)
}

// RestoreCheckpoint 恢复检查点
func (pc *ParserCoordinator) RestoreCheckpoint() error {
	if len(pc.checkpoints) == 0 {
		return fmt.Errorf("no checkpoint to restore")
	}

	// 弹出最后一个检查点
	checkpoint := pc.checkpoints[len(pc.checkpoints)-1]
	pc.checkpoints = pc.checkpoints[:len(pc.checkpoints)-1]

	// 恢复上下文
	return pc.parsingContext.RestoreCheckpoint(checkpoint)
}

// GetCurrentContext 获取当前解析上下文
func (pc *ParserCoordinator) GetCurrentContext() *ParsingContext {
	return pc.parsingContext
}

// handleParseError 处理解析错误
func (pc *ParserCoordinator) handleParseError(
	ctx context.Context,
	parseErr error,
) (*sharedVO.ErrorRecoveryResult, error) {

	if pc.parsingContext == nil {
		return nil, fmt.Errorf("parsing context not initialized")
	}

	// 创建ParseError对象
	currentToken := pc.parsingContext.TokenStream().Current()
	if currentToken == nil {
		// 如果没有当前token，使用默认位置
		currentToken = &lexicalVO.EnhancedToken{}
	}

	parseError := sharedVO.NewParseError(
		parseErr.Error(),
		currentToken.Location(),
		sharedVO.ErrorTypeSyntax,
		sharedVO.SeverityError,
	)

	// 进入恢复模式
	pc.parsingContext.EnterRecoveryMode(parseError)

	// 使用错误恢复服务进行恢复
	recoveryResult, err := pc.errorRecoveryService.RecoverFromError(
		ctx,
		parseError,
		pc.parsingContext.TokenStream(),
	)

	if err != nil {
		return nil, err
	}

	// 如果恢复成功，退出恢复模式
	if recoveryResult.HasRecovered() {
		pc.parsingContext.ExitRecoveryMode()
	}

	return recoveryResult, nil
}

// collectAllErrors 收集所有解析器的错误
func (pc *ParserCoordinator) collectAllErrors() []*sharedVO.ParseError {
	allErrors := make([]*sharedVO.ParseError, 0)

	// 收集上下文中的错误
	if pc.parsingContext != nil {
		allErrors = append(allErrors, pc.parsingContext.Errors()...)
	}

	// 收集各个解析器的错误
	if pc.recursiveDescentParser != nil {
		// 假设RecursiveDescentParser有GetErrors方法
		// allErrors = append(allErrors, pc.recursiveDescentParser.GetErrors()...)
	}

	if pc.prattParser != nil {
		allErrors = append(allErrors, pc.prattParser.GetErrors()...)
	}

	if pc.lrResolver != nil {
		allErrors = append(allErrors, pc.lrResolver.GetErrors()...)
	}

	return allErrors
}

// Reset 重置解析协调器
func (pc *ParserCoordinator) Reset() {
	pc.parsingContext = nil
	pc.checkpoints = make([]*ParsingCheckpoint, 0)
}
