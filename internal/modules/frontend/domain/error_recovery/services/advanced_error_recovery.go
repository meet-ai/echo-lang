// Package services 实现高级错误恢复服务
package services

import (
	"context"
	"fmt"
	"time"

	errorRecoveryVO "echo/internal/modules/frontend/domain/error_recovery/value_objects"
	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
)

// AdvancedErrorRecoveryService 高级错误恢复服务
// 提供智能错误恢复、策略选择、恢复跟踪等功能
type AdvancedErrorRecoveryService struct {
	strategyManager  *RecoveryStrategyManager
	strategySelector *StrategySelector
	maxAttempts      int
	timeout          time.Duration
}

// NewAdvancedErrorRecoveryService 创建新的高级错误恢复服务
func NewAdvancedErrorRecoveryService() *AdvancedErrorRecoveryService {
	strategyManager := NewRecoveryStrategyManager()
	strategySelector := NewStrategySelector(strategyManager)

	return &AdvancedErrorRecoveryService{
		strategyManager:  strategyManager,
		strategySelector: strategySelector,
		maxAttempts:      5,                // 默认最大尝试5次
		timeout:          30 * time.Second, // 默认超时30秒
	}
}

// RecoverFromError 从错误中恢复
// 使用智能策略选择和多种恢复策略
func (aers *AdvancedErrorRecoveryService) RecoverFromError(
	ctx context.Context,
	originalError *sharedVO.ParseError,
	tokenStream *lexicalVO.EnhancedTokenStream,
) (*sharedVO.ErrorRecoveryResult, error) {

	// 创建恢复结果
	result := sharedVO.NewErrorRecoveryResult()

	// 分析错误上下文
	errorContext := aers.strategySelector.AnalyzeErrorContext(ctx, originalError, tokenStream)

	// 选择恢复策略
	selectedStrategies := aers.strategySelector.SelectStrategies(ctx, originalError)

	// 如果上下文分析提供了建议策略，优先使用
	if len(errorContext.SuggestedStrategies) > 0 {
		selectedStrategies = errorContext.SuggestedStrategies
	}

	// 创建带超时的上下文
	recoveryCtx, cancel := context.WithTimeout(ctx, aers.timeout)
	defer cancel()

	// 记录开始时间，用于计算实际耗时
	startTime := time.Now()

	// 尝试每个策略，直到成功或所有策略都失败
	for attemptCount := 0; attemptCount < aers.maxAttempts && attemptCount < len(selectedStrategies); attemptCount++ {
		// 检查超时（使用context.WithTimeout）
		select {
		case <-recoveryCtx.Done():
			if recoveryCtx.Err() == context.DeadlineExceeded {
				elapsed := time.Since(startTime)
				return result, fmt.Errorf("recovery timeout after %v (elapsed: %v)", aers.timeout, elapsed)
			}
			return result, recoveryCtx.Err()
		default:
			// 继续执行
		}

		// 检查是否已经超时（额外检查，确保及时响应）
		if time.Since(startTime) >= aers.timeout {
			return result, fmt.Errorf("recovery timeout after %v", aers.timeout)
		}

		// 选择最佳策略（考虑之前的尝试）
		// 使用recoveryCtx而不是ctx，确保策略选择也受超时限制
		strategy := aers.strategySelector.SelectBestStrategy(recoveryCtx, originalError, result.Attempts())
		if strategy == nil {
			break // 没有更多策略可尝试
		}

		// 将errorRecoveryVO.RecoveryStrategy转换为sharedVO.RecoveryStrategy
		sharedStrategy := convertToSharedRecoveryStrategy(strategy)

		// 创建恢复尝试
		attempt := sharedVO.NewErrorRecoveryAttempt(
			originalError,
			sharedStrategy,
			fmt.Sprintf("尝试恢复策略: %s (第%d次)", strategy.Description(), attemptCount+1),
		)

		// 执行恢复策略（使用recoveryCtx确保策略执行也受超时限制）
		recoveryResult, err := aers.strategyManager.ExecuteStrategy(recoveryCtx, strategy, attempt, tokenStream)
		
		// 策略执行后再次检查超时
		select {
		case <-recoveryCtx.Done():
			if recoveryCtx.Err() == context.DeadlineExceeded {
				elapsed := time.Since(startTime)
				return result, fmt.Errorf("recovery timeout after %v (elapsed: %v)", aers.timeout, elapsed)
			}
			return result, recoveryCtx.Err()
		default:
			// 继续执行
		}
		
		if err != nil {
			// 策略执行失败，记录错误但继续尝试下一个策略
			sharedStrategy := convertToSharedRecoveryStrategy(strategy)
			attempt = sharedVO.NewErrorRecoveryAttempt(
				originalError,
				sharedStrategy,
				fmt.Sprintf("策略执行失败: %v", err),
			)
			result.AddAttempt(attempt)
			continue
		}

		// 记录恢复尝试
		if recoveryResult.Recovered() {
			attempt.MarkSuccess(
				recoveryResult.NewPosition()-originalError.Location().Offset(), // 恢复的token数
				1, // 恢复的节点数
			)
			result.AddAttempt(attempt)
			break // 成功恢复，停止尝试
		} else {
			// 恢复失败，继续尝试下一个策略
			result.AddAttempt(attempt)
		}
	}

	return result, nil
}

// RecoverWithPanicMode 使用恐慌模式恢复
// 这是最常用的恢复策略，跳过错误块直到找到同步点
func (aers *AdvancedErrorRecoveryService) RecoverWithPanicMode(
	ctx context.Context,
	originalError *sharedVO.ParseError,
	tokenStream *lexicalVO.EnhancedTokenStream,
) (*sharedVO.RecoveryResult, error) {

	// 创建恐慌模式策略
	panicStrategy := errorRecoveryVO.NewRecoveryStrategy(
		errorRecoveryVO.StrategyTypePanicMode,
		"恐慌模式恢复",
		1,
	)

	// 转换为shared策略
	sharedStrategy := convertToSharedRecoveryStrategy(panicStrategy)

	// 创建恢复尝试
	attempt := sharedVO.NewErrorRecoveryAttempt(
		originalError,
		sharedStrategy,
		"使用恐慌模式跳过错误块",
	)

	// 执行恢复
	return aers.strategyManager.ExecuteStrategy(ctx, panicStrategy, attempt, tokenStream)
}

// TrackRecoveryResult 跟踪恢复结果
// 记录恢复的详细信息，用于后续分析和优化
func (aers *AdvancedErrorRecoveryService) TrackRecoveryResult(
	ctx context.Context,
	result *sharedVO.ErrorRecoveryResult,
) (*RecoveryTrackingInfo, error) {

	trackingInfo := &RecoveryTrackingInfo{
		RecoveryID:      generateAdvancedRecoveryID(),
		OriginalError:   result.Attempts()[0].OriginalError(),
		TotalAttempts:   len(result.Attempts()),
		Successful:      result.HasRecovered(),
		RecoveryTime:    time.Now(),
		StrategiesUsed:  make([]string, 0),
		FinalStrategy:   "",
		RecoveryDetails: make(map[string]interface{}),
	}

	// 记录使用的策略
	for _, attempt := range result.Attempts() {
		strategy := attempt.Strategy()
		strategyType := string((&strategy).StrategyType())
		trackingInfo.StrategiesUsed = append(trackingInfo.StrategiesUsed, strategyType)

		if attempt.IsSuccessful() {
			trackingInfo.FinalStrategy = strategyType
			trackingInfo.RecoveryDetails["recovered_tokens"] = attempt.RecoveredTokens()
			trackingInfo.RecoveryDetails["recovered_nodes"] = attempt.RecoveredNodes()
		}
	}

	// 如果没有成功恢复，记录失败原因
	if !trackingInfo.Successful {
		trackingInfo.RecoveryDetails["failure_reason"] = "所有恢复策略都失败"
		trackingInfo.RecoveryDetails["last_error"] = result.Attempts()[len(result.Attempts())-1].Details()
	}

	return trackingInfo, nil
}

// RecoveryTrackingInfo 恢复跟踪信息
type RecoveryTrackingInfo struct {
	RecoveryID      string
	OriginalError   *sharedVO.ParseError
	TotalAttempts   int
	Successful      bool
	RecoveryTime    time.Time
	StrategiesUsed  []string
	FinalStrategy   string
	RecoveryDetails map[string]interface{}
}

// String 返回跟踪信息的字符串表示
func (rti *RecoveryTrackingInfo) String() string {
	status := "失败"
	if rti.Successful {
		status = "成功"
	}

	return fmt.Sprintf("恢复跟踪[%s]: %s, 尝试次数: %d, 最终策略: %s",
		rti.RecoveryID, status, rti.TotalAttempts, rti.FinalStrategy)
}

// generateAdvancedRecoveryID 生成高级恢复ID（避免与error_recovery_service.go中的重复）
func generateAdvancedRecoveryID() string {
	return fmt.Sprintf("advanced_recovery_%d", time.Now().UnixNano())
}

// convertToSharedRecoveryStrategy 将errorRecoveryVO.RecoveryStrategy转换为sharedVO.RecoveryStrategy
func convertToSharedRecoveryStrategy(strategy *errorRecoveryVO.RecoveryStrategy) sharedVO.RecoveryStrategy {
	// 将errorRecoveryVO的StrategyType转换为sharedVO的StrategyType
	var sharedStrategyType sharedVO.RecoveryStrategyType
	switch strategy.StrategyType() {
	case errorRecoveryVO.StrategyTypePanicMode:
		sharedStrategyType = sharedVO.StrategyTypePanicMode
	case errorRecoveryVO.StrategyTypeInsertion:
		sharedStrategyType = sharedVO.StrategyTypeInsertion
	case errorRecoveryVO.StrategyTypeReplacement:
		sharedStrategyType = sharedVO.StrategyTypeReplacement
	case errorRecoveryVO.StrategyTypeDeletion:
		sharedStrategyType = sharedVO.StrategyTypeDeletion
	default:
		sharedStrategyType = sharedVO.StrategyTypePanicMode // 默认
	}

	return *sharedVO.NewRecoveryStrategy(
		sharedStrategyType,
		strategy.Description(),
		strategy.Priority(),
	)
}

// RecoverWithContext 带上下文的恢复
// 允许传入额外的上下文信息来改进恢复策略选择
func (aers *AdvancedErrorRecoveryService) RecoverWithContext(
	ctx context.Context,
	originalError *sharedVO.ParseError,
	tokenStream *lexicalVO.EnhancedTokenStream,
	recoveryContext *RecoveryContext,
) (*sharedVO.ErrorRecoveryResult, error) {

	// 如果提供了上下文，可以调整策略选择
	if recoveryContext != nil {
		// 根据上下文调整最大尝试次数
		if recoveryContext.MaxAttempts > 0 {
			aers.maxAttempts = recoveryContext.MaxAttempts
		}

		// 根据上下文调整超时时间
		if recoveryContext.Timeout > 0 {
			aers.timeout = recoveryContext.Timeout
		}
	}

	// 执行恢复
	result, err := aers.RecoverFromError(ctx, originalError, tokenStream)
	if err != nil {
		return nil, err
	}

	// 跟踪恢复结果
	if recoveryContext != nil && recoveryContext.TrackResult {
		_, trackErr := aers.TrackRecoveryResult(ctx, result)
		if trackErr != nil {
			// 跟踪失败不影响恢复结果
			_ = trackErr
		}
	}

	return result, nil
}

// RecoveryContext 恢复上下文
// 提供额外的恢复配置信息
type RecoveryContext struct {
	MaxAttempts int           // 最大尝试次数
	Timeout     time.Duration // 超时时间
	TrackResult bool          // 是否跟踪恢复结果
	Priority    int           // 恢复优先级
}

// NewRecoveryContext 创建新的恢复上下文
func NewRecoveryContext() *RecoveryContext {
	return &RecoveryContext{
		MaxAttempts: 5,
		Timeout:     30 * time.Second,
		TrackResult: true,
		Priority:    1,
	}
}
