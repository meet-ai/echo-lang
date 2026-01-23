// Package services 实现错误恢复策略服务
package services

import (
	"context"
	"fmt"

	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
	errorRecoveryVO "echo/internal/modules/frontend/domain/error_recovery/value_objects"
)

// RecoveryStrategyExecutor 恢复策略执行器接口
type RecoveryStrategyExecutor interface {
	// Execute 执行恢复策略
	Execute(ctx context.Context, attempt *sharedVO.ErrorRecoveryAttempt, tokenStream *lexicalVO.EnhancedTokenStream) (*sharedVO.RecoveryResult, error)

	// CanHandle 检查是否能处理指定的策略
	CanHandle(strategy *errorRecoveryVO.RecoveryStrategy) bool
}

// PanicModeRecoveryExecutor 恐慌模式恢复执行器
type PanicModeRecoveryExecutor struct {
	syncPointRegistry *errorRecoveryVO.SyncPointRegistry
}

// NewPanicModeRecoveryExecutor 创建恐慌模式恢复执行器
func NewPanicModeRecoveryExecutor(syncPointRegistry *errorRecoveryVO.SyncPointRegistry) *PanicModeRecoveryExecutor {
	return &PanicModeRecoveryExecutor{
		syncPointRegistry: syncPointRegistry,
	}
}

// CanHandle 检查是否能处理恐慌模式策略
func (pmre *PanicModeRecoveryExecutor) CanHandle(strategy *errorRecoveryVO.RecoveryStrategy) bool {
	return strategy.StrategyType() == errorRecoveryVO.StrategyTypePanicMode
}

// Execute 执行恐慌模式恢复
func (pmre *PanicModeRecoveryExecutor) Execute(
	ctx context.Context,
	attempt *sharedVO.ErrorRecoveryAttempt,
	tokenStream *lexicalVO.EnhancedTokenStream,
) (*sharedVO.RecoveryResult, error) {

	// 获取错误位置
	errorPos := attempt.OriginalError().Location().Offset()

	// 从错误位置开始查找同步点
	syncPoint := pmre.findNearestSyncPoint(tokenStream, errorPos)
	if syncPoint == nil {
		// 没有找到同步点，恢复失败
		return sharedVO.NewRecoveryResult(
			attempt.OriginalError(),
			attempt.Strategy(),
		), nil
	}

	// 移动到同步点位置
	newPosition := syncPoint.Location().Offset()

	// 创建恢复结果
	result := sharedVO.NewRecoveryResult(
		attempt.OriginalError(),
		attempt.Strategy(),
	)

	// 设置恢复位置
	result.SetNewPosition(newPosition)
	result.SetRecovered(true)

	// 添加恢复建议
	fix := sharedVO.NewFixSuggestion(
		fmt.Sprintf("跳过到同步点: %s", syncPoint.String()),
		sharedVO.FixTypeDelete, // 使用删除类型表示跳过
		syncPoint.Location(),
		"", // oldText
		"", // newText
		0.8, // confidence
	)
	result.AddSuggestedFix(fix)

	return result, nil
}

// findNearestSyncPoint 查找最近的同步点
func (pmre *PanicModeRecoveryExecutor) findNearestSyncPoint(tokenStream *lexicalVO.EnhancedTokenStream, startPos int) *lexicalVO.EnhancedToken {
	// 从错误位置开始向前查找
	tokens := tokenStream.Tokens()

	// 限制查找范围，避免无限循环
	maxLookahead := 50

	for i := startPos; i < len(tokens) && i < startPos+maxLookahead; i++ {
		token := tokens[i]

		// 检查是否是同步点
		syncPoints := pmre.syncPointRegistry.FindSyncPoints(token)
		if len(syncPoints) > 0 {
			return token
		}
	}

	return nil
}

// InsertionRecoveryExecutor 插入模式恢复执行器
type InsertionRecoveryExecutor struct{}

// NewInsertionRecoveryExecutor 创建插入模式恢复执行器
func NewInsertionRecoveryExecutor() *InsertionRecoveryExecutor {
	return &InsertionRecoveryExecutor{}
}

// CanHandle 检查是否能处理插入模式策略
func (ire *InsertionRecoveryExecutor) CanHandle(strategy *errorRecoveryVO.RecoveryStrategy) bool {
	return strategy.StrategyType() == errorRecoveryVO.StrategyTypeInsertion
}

// Execute 执行插入模式恢复
func (ire *InsertionRecoveryExecutor) Execute(
	ctx context.Context,
	attempt *sharedVO.ErrorRecoveryAttempt,
	tokenStream *lexicalVO.EnhancedTokenStream,
) (*sharedVO.RecoveryResult, error) {

	result := sharedVO.NewRecoveryResult(
		attempt.OriginalError(),
		attempt.Strategy(),
	)

	// 简单实现：假设插入分号
	errorPos := attempt.OriginalError().Location().Offset()

	// 在错误位置插入分号
	result.SetNewPosition(errorPos)
	result.SetRecovered(true)
	fix := sharedVO.NewFixSuggestion(
		"在错误位置插入分号 ';'",
		sharedVO.FixTypeInsert,
		attempt.OriginalError().Location(),
		"", // oldText
		";", // newText
		0.7, // confidence
	)
	result.AddSuggestedFix(fix)

	return result, nil
}

// ReplacementRecoveryExecutor 替换模式恢复执行器
type ReplacementRecoveryExecutor struct{}

// NewReplacementRecoveryExecutor 创建替换模式恢复执行器
func NewReplacementRecoveryExecutor() *ReplacementRecoveryExecutor {
	return &ReplacementRecoveryExecutor{}
}

// CanHandle 检查是否能处理替换模式策略
func (rre *ReplacementRecoveryExecutor) CanHandle(strategy *errorRecoveryVO.RecoveryStrategy) bool {
	return strategy.StrategyType() == errorRecoveryVO.StrategyTypeReplacement
}

// Execute 执行替换模式恢复
func (rre *ReplacementRecoveryExecutor) Execute(
	ctx context.Context,
	attempt *sharedVO.ErrorRecoveryAttempt,
	tokenStream *lexicalVO.EnhancedTokenStream,
) (*sharedVO.RecoveryResult, error) {

	result := sharedVO.NewRecoveryResult(
		attempt.OriginalError(),
		attempt.Strategy(),
	)

	// 简单实现：替换错误的标识符为标准标识符
	errorPos := attempt.OriginalError().Location().Offset()

	result.SetNewPosition(errorPos)
	result.SetRecovered(true)
	fix := sharedVO.NewFixSuggestion(
		"替换错误的token为标准token",
		sharedVO.FixTypeReplace,
		attempt.OriginalError().Location(),
		"error_token", // oldText (示例)
		"valid_token", // newText (示例)
		0.6, // confidence
	)
	result.AddSuggestedFix(fix)

	return result, nil
}

// DeletionRecoveryExecutor 删除模式恢复执行器
type DeletionRecoveryExecutor struct{}

// NewDeletionRecoveryExecutor 创建删除模式恢复执行器
func NewDeletionRecoveryExecutor() *DeletionRecoveryExecutor {
	return &DeletionRecoveryExecutor{}
}

// CanHandle 检查是否能处理删除模式策略
func (dre *DeletionRecoveryExecutor) CanHandle(strategy *errorRecoveryVO.RecoveryStrategy) bool {
	return strategy.StrategyType() == errorRecoveryVO.StrategyTypeDeletion
}

// Execute 执行删除模式恢复
func (dre *DeletionRecoveryExecutor) Execute(
	ctx context.Context,
	attempt *sharedVO.ErrorRecoveryAttempt,
	tokenStream *lexicalVO.EnhancedTokenStream,
) (*sharedVO.RecoveryResult, error) {

	result := sharedVO.NewRecoveryResult(
		attempt.OriginalError(),
		attempt.Strategy(),
	)

	// 简单实现：删除错误的token
	errorPos := attempt.OriginalError().Location().Offset()

	result.SetNewPosition(errorPos + 1) // 跳过错误token
	result.SetRecovered(true)
	fix := sharedVO.NewFixSuggestion(
		"删除错误的token",
		sharedVO.FixTypeDelete,
		attempt.OriginalError().Location(),
		"error_token", // oldText (示例)
		"", // newText
		0.5, // confidence
	)
	result.AddSuggestedFix(fix)

	return result, nil
}

// RecoveryStrategyManager 恢复策略管理器
type RecoveryStrategyManager struct {
	executors []RecoveryStrategyExecutor
}

// NewRecoveryStrategyManager 创建恢复策略管理器
func NewRecoveryStrategyManager() *RecoveryStrategyManager {
	syncRegistry := errorRecoveryVO.NewSyncPointRegistry()

	return &RecoveryStrategyManager{
		executors: []RecoveryStrategyExecutor{
			NewPanicModeRecoveryExecutor(syncRegistry),
			NewInsertionRecoveryExecutor(),
			NewReplacementRecoveryExecutor(),
			NewDeletionRecoveryExecutor(),
		},
	}
}

// ExecuteStrategy 执行指定的恢复策略
func (rsm *RecoveryStrategyManager) ExecuteStrategy(
	ctx context.Context,
	strategy *errorRecoveryVO.RecoveryStrategy,
	attempt *sharedVO.ErrorRecoveryAttempt,
	tokenStream *lexicalVO.EnhancedTokenStream,
) (*sharedVO.RecoveryResult, error) {

	// 查找能处理此策略的执行器
	for _, executor := range rsm.executors {
		if executor.CanHandle(strategy) {
			return executor.Execute(ctx, attempt, tokenStream)
		}
	}

	// 没有找到合适的执行器
	return nil, fmt.Errorf("no executor found for strategy: %s", strategy.StrategyType())
}

// GetAvailableStrategies 获取所有可用的恢复策略
func (rsm *RecoveryStrategyManager) GetAvailableStrategies() []*errorRecoveryVO.RecoveryStrategy {
	return []*errorRecoveryVO.RecoveryStrategy{
		errorRecoveryVO.NewRecoveryStrategy(errorRecoveryVO.StrategyTypePanicMode, "恐慌模式恢复", 1),
		errorRecoveryVO.NewRecoveryStrategy(errorRecoveryVO.StrategyTypeInsertion, "插入模式恢复", 2),
		errorRecoveryVO.NewRecoveryStrategy(errorRecoveryVO.StrategyTypeReplacement, "替换模式恢复", 3),
		errorRecoveryVO.NewRecoveryStrategy(errorRecoveryVO.StrategyTypeDeletion, "删除模式恢复", 4),
	}
}

// StrategySelector 策略选择器
type StrategySelector struct {
	strategyManager *RecoveryStrategyManager
}

// NewStrategySelector 创建策略选择器
func NewStrategySelector(strategyManager *RecoveryStrategyManager) *StrategySelector {
	return &StrategySelector{
		strategyManager: strategyManager,
	}
}

// SelectStrategies 为给定的错误选择合适的恢复策略
func (ss *StrategySelector) SelectStrategies(
	ctx context.Context,
	originalError *sharedVO.ParseError,
) []*errorRecoveryVO.RecoveryStrategy {

	availableStrategies := ss.strategyManager.GetAvailableStrategies()
	selectedStrategies := make([]*errorRecoveryVO.RecoveryStrategy, 0)

	// 根据错误类型和严重程度选择策略
	errorType := originalError.ErrorType()
	errorSeverity := originalError.Severity()

	for _, strategy := range availableStrategies {
		if strategy.CanApply(errorType, errorSeverity) {
			selectedStrategies = append(selectedStrategies, strategy)
		}
	}

	// 如果没有找到合适的策略，使用恐慌模式作为后备
	if len(selectedStrategies) == 0 {
		panicMode := errorRecoveryVO.NewRecoveryStrategy(
			errorRecoveryVO.StrategyTypePanicMode,
			"默认恐慌模式恢复",
			999, // 最低优先级
		)
		selectedStrategies = append(selectedStrategies, panicMode)
	}

	// 按优先级排序（优先级数值越小越优先）
	ss.sortStrategiesByPriority(selectedStrategies)

	return selectedStrategies
}

// SelectBestStrategy 选择最佳的恢复策略
func (ss *StrategySelector) SelectBestStrategy(
	ctx context.Context,
	originalError *sharedVO.ParseError,
	previousAttempts []*sharedVO.ErrorRecoveryAttempt,
) *errorRecoveryVO.RecoveryStrategy {

	candidates := ss.SelectStrategies(ctx, originalError)

	// 如果没有之前的尝试，直接选择优先级最高的策略
	if len(previousAttempts) == 0 {
		if len(candidates) > 0 {
			return candidates[0]
		}
		return nil
	}

	// 分析之前的尝试，避免重复失败的策略
	failedStrategies := make(map[string]bool)
	for _, attempt := range previousAttempts {
		if !attempt.IsSuccessful() {
			strategy := attempt.Strategy()
			failedStrategies[string(strategy.StrategyType())] = true
		}
	}

	// 选择第一个未失败的策略
	for _, strategy := range candidates {
		if !failedStrategies[string(strategy.StrategyType())] {
			return strategy
		}
	}

	// 如果所有策略都失败过，选择优先级最高的（可能有不同的执行结果）
	if len(candidates) > 0 {
		return candidates[0]
	}

	return nil
}

// sortStrategiesByPriority 按优先级排序策略
func (ss *StrategySelector) sortStrategiesByPriority(strategies []*errorRecoveryVO.RecoveryStrategy) {
	// 简单冒泡排序（策略数量少，性能不是问题）
	for i := 0; i < len(strategies)-1; i++ {
		for j := 0; j < len(strategies)-i-1; j++ {
			if strategies[j].Priority() > strategies[j+1].Priority() {
				strategies[j], strategies[j+1] = strategies[j+1], strategies[j]
			}
		}
	}
}

// AnalyzeErrorContext 分析错误上下文以改进策略选择
func (ss *StrategySelector) AnalyzeErrorContext(
	ctx context.Context,
	originalError *sharedVO.ParseError,
	tokenStream *lexicalVO.EnhancedTokenStream,
) *ErrorContextAnalysis {

	analysis := &ErrorContextAnalysis{
		ErrorPosition: originalError.Location(),
		ErrorType:     originalError.ErrorType(),
		Severity:      originalError.Severity(),
	}

	// 分析错误位置附近的Token
	errorPos := originalError.Location().Offset()
	analysis.TokensBefore = ss.getTokensAround(errorPos, -3, tokenStream)
	analysis.TokensAfter = ss.getTokensAround(errorPos, 3, tokenStream)

	// 分析可能的恢复策略适用性
	analysis.SuggestedStrategies = ss.analyzeStrategyApplicability(analysis)

	return analysis
}

// getTokensAround 获取指定位置周围的Token
func (ss *StrategySelector) getTokensAround(centerPos int, offset int, tokenStream *lexicalVO.EnhancedTokenStream) []*lexicalVO.EnhancedToken {
	tokens := tokenStream.Tokens()
	result := make([]*lexicalVO.EnhancedToken, 0)

	start := centerPos + offset
	end := centerPos + offset + 1

	if offset < 0 {
		start = centerPos + offset
		end = centerPos
	}

	for i := start; i < end && i >= 0 && i < len(tokens); i++ {
		result = append(result, tokens[i])
	}

	return result
}

// analyzeStrategyApplicability 分析策略适用性
func (ss *StrategySelector) analyzeStrategyApplicability(analysis *ErrorContextAnalysis) []*errorRecoveryVO.RecoveryStrategy {
	applicableStrategies := make([]*errorRecoveryVO.RecoveryStrategy, 0)

	availableStrategies := ss.strategyManager.GetAvailableStrategies()

	for _, strategy := range availableStrategies {
		if ss.isStrategyApplicable(strategy, analysis) {
			applicableStrategies = append(applicableStrategies, strategy)
		}
	}

	return applicableStrategies
}

// isStrategyApplicable 检查策略是否适用于给定的错误上下文
func (ss *StrategySelector) isStrategyApplicable(strategy *errorRecoveryVO.RecoveryStrategy, analysis *ErrorContextAnalysis) bool {
	// 基础检查：策略是否适用于错误类型
	if !strategy.CanApply(analysis.ErrorType, analysis.Severity) {
		return false
	}

	// 上下文特定检查
	switch strategy.StrategyType() {
	case errorRecoveryVO.StrategyTypeInsertion:
		// 检查是否可能是缺失的分号或其他分隔符
		return ss.looksLikeMissingDelimiter(analysis)
	case errorRecoveryVO.StrategyTypeDeletion:
		// 检查是否可能是多余的token
		return ss.looksLikeExtraToken(analysis)
	case errorRecoveryVO.StrategyTypeReplacement:
		// 检查是否可能是错误的标识符或关键字
		return ss.looksLikeWrongToken(analysis)
	case errorRecoveryVO.StrategyTypePanicMode:
		// 恐慌模式总是适用（作为最后手段）
		return true
	}

	return false
}

// looksLikeMissingDelimiter 检查是否看起来像是缺失分隔符
func (ss *StrategySelector) looksLikeMissingDelimiter(analysis *ErrorContextAnalysis) bool {
	// 简单启发式：检查前后token是否期望分隔符
	beforeTokens := analysis.TokensBefore
	afterTokens := analysis.TokensAfter

	if len(beforeTokens) > 0 && len(afterTokens) > 0 {
		before := beforeTokens[len(beforeTokens)-1]
		after := afterTokens[0]

		// 如果前一个是标识符，后一个也是，可能需要分号
		if before.Type() == lexicalVO.EnhancedTokenTypeIdentifier &&
		   after.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
			return true
		}
	}

	return false
}

// looksLikeExtraToken 检查是否看起来像是多余的token
func (ss *StrategySelector) looksLikeExtraToken(analysis *ErrorContextAnalysis) bool {
	// 简单启发式：检查是否有重复的关键字
	// 这里可以实现更复杂的逻辑
	return false
}

// looksLikeWrongToken 检查是否看起来像是错误的token
func (ss *StrategySelector) looksLikeWrongToken(analysis *ErrorContextAnalysis) bool {
	// 简单启发式：检查token拼写错误
	// 这里可以实现更复杂的逻辑
	return false
}

// ErrorContextAnalysis 错误上下文分析结果
type ErrorContextAnalysis struct {
	ErrorPosition       sharedVO.SourceLocation
	ErrorType           sharedVO.ParseErrorType
	Severity            sharedVO.ParseErrorSeverity
	TokensBefore        []*lexicalVO.EnhancedToken
	TokensAfter         []*lexicalVO.EnhancedToken
	SuggestedStrategies []*errorRecoveryVO.RecoveryStrategy
}