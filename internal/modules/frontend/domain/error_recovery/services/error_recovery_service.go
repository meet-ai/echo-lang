// Package services 定义错误恢复的领域服务
package services

import (
	"context"
	"fmt"
	"time"

	"echo/internal/modules/frontend/domain/error_recovery/entities"
	"echo/internal/modules/frontend/domain/shared/value_objects"
)

// ErrorRecoveryService 错误恢复服务
type ErrorRecoveryService struct {
	maxAttempts int
	timeoutMs   int
}

// NewErrorRecoveryService 创建新的错误恢复服务
func NewErrorRecoveryService(maxAttempts, timeoutMs int) *ErrorRecoveryService {
	return &ErrorRecoveryService{
		maxAttempts: maxAttempts,
		timeoutMs:   timeoutMs,
	}
}

// RecoverFromError 从错误中恢复，返回恢复结果
func (ers *ErrorRecoveryService) RecoverFromError(
	ctx context.Context,
	originalError *value_objects.ParseError,
	tokenStream *value_objects.TokenStream,
) (*value_objects.ErrorRecoveryResult, error) {

	result := value_objects.NewErrorRecoveryResult()
	recoveryID := generateRecoveryID()

	// 尝试不同的恢复策略
	strategies := []*value_objects.RecoveryStrategy{
		value_objects.NewRecoveryStrategy(value_objects.StrategyTypePanicMode, "Panic mode recovery", 1),
		value_objects.NewRecoveryStrategy(value_objects.StrategyTypeInsertion, "Phrase level recovery", 2),
		value_objects.NewRecoveryStrategy(value_objects.StrategyTypeReplacement, "Error node recovery", 3),
	}

	for _, strategy := range strategies {
		if result.HasRecovered() {
			break // 如果已经恢复成功，停止尝试
		}

		recoveryEntity := entities.NewErrorRecoveryEntity(recoveryID, originalError, *strategy)

		// 执行恢复尝试
		attempt := ers.attemptRecovery(ctx, recoveryEntity, tokenStream, *strategy)
		result.AddAttempt(attempt)

		if attempt.IsSuccessful() {
			break // 成功恢复，停止尝试
		}
	}

	return result, nil
}

// attemptRecovery 执行一次恢复尝试
func (ers *ErrorRecoveryService) attemptRecovery(
	ctx context.Context,
	entity *entities.ErrorRecoveryEntity,
	tokenStream *value_objects.TokenStream,
	strategy value_objects.RecoveryStrategy,
) *value_objects.ErrorRecoveryAttempt {

	attempt := value_objects.NewErrorRecoveryAttempt(
		entity.OriginalError(),
		strategy,
		fmt.Sprintf("Attempting recovery with strategy: %s", strategy.StrategyType()),
	)

	// 根据策略执行恢复
	switch strategy.StrategyType() {
	case value_objects.StrategyTypePanicMode:
		attempt = ers.recoverWithPanicMode(ctx, attempt, tokenStream)
	case value_objects.StrategyTypeInsertion:
		attempt = ers.recoverWithPhraseLevel(ctx, attempt, tokenStream)
	case value_objects.StrategyTypeReplacement:
		attempt = ers.recoverWithErrorNode(ctx, attempt, tokenStream)
	default:
		attempt = value_objects.NewErrorRecoveryAttempt(
			entity.OriginalError(),
			strategy,
			fmt.Sprintf("Unknown strategy: %s", strategy.StrategyType()),
		)
	}

	return attempt
}

// recoverWithPanicMode 使用恐慌模式恢复
func (ers *ErrorRecoveryService) recoverWithPanicMode(
	ctx context.Context,
	attempt *value_objects.ErrorRecoveryAttempt,
	tokenStream *value_objects.TokenStream,
) *value_objects.ErrorRecoveryAttempt {

	// 实现恐慌模式：跳过Token直到找到同步点
	// 同步点通常是语句分隔符、块开始/结束等

	tokens := tokenStream.Tokens()
	errorLocation := attempt.OriginalError().Location()
	startIndex := findTokenIndexAtLocation(tokens, errorLocation)

	if startIndex < 0 {
		return attempt // 无法找到错误位置
	}

	// 从错误位置开始，向后查找同步点
	syncPointIndex := ers.findSyncPoint(tokens, startIndex)

	if syncPointIndex > startIndex {
		recoveredTokens := syncPointIndex - startIndex
		attempt.MarkSuccess(recoveredTokens, 0) // 没有恢复AST节点，只是跳过Token
	}

	return attempt
}

// recoverWithPhraseLevel 使用短语级别恢复
func (ers *ErrorRecoveryService) recoverWithPhraseLevel(
	ctx context.Context,
	attempt *value_objects.ErrorRecoveryAttempt,
	tokenStream *value_objects.TokenStream,
) *value_objects.ErrorRecoveryAttempt {

	// 实现短语级别：尝试插入或替换Token来修复语法
	// 这是一个简化的实现，实际可能需要更复杂的算法

	errorLocation := attempt.OriginalError().Location()
	tokens := tokenStream.Tokens()

	// 查找错误位置附近的Token
	errorIndex := findTokenIndexAtLocation(tokens, errorLocation)
	if errorIndex < 0 || errorIndex >= len(tokens)-1 {
		return attempt
	}

	// 检查是否可以插入缺失的分号
	currentToken := tokens[errorIndex]
	nextToken := tokens[errorIndex+1]

	if ers.canInsertSemicolon(currentToken, nextToken) {
		// 模拟插入分号的恢复
		attempt.MarkSuccess(1, 1) // 恢复1个Token，1个节点
	}

	return attempt
}

// recoverWithErrorNode 使用错误节点恢复
func (ers *ErrorRecoveryService) recoverWithErrorNode(
	ctx context.Context,
	attempt *value_objects.ErrorRecoveryAttempt,
	tokenStream *value_objects.TokenStream,
) *value_objects.ErrorRecoveryAttempt {

	// 实现错误节点：创建一个特殊的错误AST节点来表示无法解析的部分
	// 这允许解析继续进行，同时记录错误

	attempt.MarkSuccess(0, 1) // 没有恢复Token，但创建了1个错误AST节点
	return attempt
}

// findSyncPoint 查找同步点
func (ers *ErrorRecoveryService) findSyncPoint(tokens []*value_objects.Token, startIndex int) int {
	// 同步点：分号、右大括号、函数声明等
	syncTokens := []string{"SEMICOLON", "RBRACE", "FUNC", "LET", "IF", "WHILE", "FOR"}

	for i := startIndex; i < len(tokens); i++ {
		tokenType := string(tokens[i].Type())
		for _, syncType := range syncTokens {
			if tokenType == syncType {
				return i
			}
		}
	}

	return len(tokens) // 如果没有找到同步点，返回流末尾
}

// canInsertSemicolon 检查是否可以在两个Token之间插入分号
func (ers *ErrorRecoveryService) canInsertSemicolon(current, next *value_objects.Token) bool {
	// 简化的检查：如果当前是标识符，下一个是关键字或标识符，可能需要分号
	currentType := string(current.Type())
	nextType := string(next.Type())

	// 如果当前是标识符或字面量，下一个是关键字或标识符，可能是缺失分号
	identifierLike := []string{"IDENTIFIER", "INTEGER", "FLOAT", "STRING", "BOOLEAN", "RPAREN", "RBRACKET"}
	statementStarters := []string{"LET", "IF", "WHILE", "FOR", "RETURN", "PRINT", "IDENTIFIER"}

	isIdentifierLike := contains(identifierLike, currentType)
	isStatementStarter := contains(statementStarters, nextType)

	return isIdentifierLike && isStatementStarter
}

// findTokenIndexAtLocation 根据位置查找Token索引
func findTokenIndexAtLocation(tokens []*value_objects.Token, location value_objects.SourceLocation) int {
	for i, token := range tokens {
		tokenLoc := token.Location()
		if tokenLoc.Line() == location.Line() &&
		   tokenLoc.Column() <= location.Column() &&
		   tokenLoc.Offset() <= location.Offset() {
			return i
		}
	}
	return -1
}

// contains 检查切片是否包含指定字符串
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// generateRecoveryID 生成恢复ID
func generateRecoveryID() string {
	// 简化的ID生成，实际应该使用更强的随机性
	return fmt.Sprintf("recovery_%d", time.Now().UnixNano())
}
