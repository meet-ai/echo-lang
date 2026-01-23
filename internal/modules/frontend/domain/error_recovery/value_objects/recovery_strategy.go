// Package value_objects 定义错误恢复相关值对象
package value_objects

import sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"

// RecoveryStrategyType 恢复策略类型
type RecoveryStrategyType string

const (
	// StrategyTypePanicMode 恐慌模式：跳过错误块，直到找到同步点
	StrategyTypePanicMode RecoveryStrategyType = "panic_mode"

	// StrategyTypeInsertion 插入模式：在错误位置插入缺失的标记
	StrategyTypeInsertion RecoveryStrategyType = "insertion"

	// StrategyTypeReplacement 替换模式：将错误的标记替换为正确的标记
	StrategyTypeReplacement RecoveryStrategyType = "replacement"

	// StrategyTypeDeletion 删除模式：删除多余的标记
	StrategyTypeDeletion RecoveryStrategyType = "deletion"
)

// RecoveryStrategy 错误恢复策略值对象
// 定义了如何从解析错误中恢复的策略
type RecoveryStrategy struct {
	strategyType RecoveryStrategyType
	description  string
	priority     int
	maxAttempts  int
}

// NewRecoveryStrategy 创建新的恢复策略
func NewRecoveryStrategy(strategyType RecoveryStrategyType, description string, priority int) *RecoveryStrategy {
	return &RecoveryStrategy{
		strategyType: strategyType,
		description:  description,
		priority:     priority,
		maxAttempts:  3, // 默认最大尝试次数
	}
}

// StrategyType 获取策略类型
func (rs *RecoveryStrategy) StrategyType() RecoveryStrategyType {
	return rs.strategyType
}

// Description 获取策略描述
func (rs *RecoveryStrategy) Description() string {
	return rs.description
}

// Priority 获取策略优先级（数值越小优先级越高）
func (rs *RecoveryStrategy) Priority() int {
	return rs.priority
}

// MaxAttempts 获取最大尝试次数
func (rs *RecoveryStrategy) MaxAttempts() int {
	return rs.maxAttempts
}

// SetMaxAttempts 设置最大尝试次数
func (rs *RecoveryStrategy) SetMaxAttempts(attempts int) {
	rs.maxAttempts = attempts
}

// CanApply 检查策略是否适用于给定的错误类型
func (rs *RecoveryStrategy) CanApply(errorType sharedVO.ParseErrorType, errorSeverity sharedVO.ParseErrorSeverity) bool {
	switch rs.strategyType {
	case StrategyTypePanicMode:
		// 恐慌模式适用于严重错误
		return errorSeverity == sharedVO.SeverityError
	case StrategyTypeInsertion:
		// 插入模式适用于语法错误
		return errorType == sharedVO.ErrorTypeSyntax
	case StrategyTypeReplacement:
		// 替换模式适用于词法错误
		return errorType == sharedVO.ErrorTypeLexical
	case StrategyTypeDeletion:
		// 删除模式适用于语法错误
		return errorType == sharedVO.ErrorTypeSyntax
	default:
		return false
	}
}

// String 返回策略的字符串表示
func (rs *RecoveryStrategy) String() string {
	return string(rs.strategyType) + "(" + rs.description + ")"
}
