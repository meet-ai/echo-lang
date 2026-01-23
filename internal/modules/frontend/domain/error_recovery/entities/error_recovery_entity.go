// Package entities 定义错误恢复的实体
package entities

import (
	"fmt"
	"time"

	"echo/internal/modules/frontend/domain/shared/value_objects"
)

// ErrorRecoveryEntity 聚合根，表示错误恢复会话及其状态
type ErrorRecoveryEntity struct {
	id              string
	originalError   *value_objects.ParseError
	strategy        value_objects.RecoveryStrategy
	status          RecoveryStatus
	attempts        []*value_objects.ErrorRecoveryAttempt
	recoveredTokens int
	recoveredNodes  int
	startedAt       time.Time
	lastAttemptedAt *time.Time
}


// RecoveryStatus 错误恢复状态
type RecoveryStatus string

const (
	StatusInProgress RecoveryStatus = "in_progress"
	StatusSucceeded  RecoveryStatus = "succeeded"
	StatusFailed     RecoveryStatus = "failed"
	StatusAbandoned  RecoveryStatus = "abandoned"
)

// NewErrorRecoveryEntity 创建新的错误恢复实体
func NewErrorRecoveryEntity(id string, originalError *value_objects.ParseError, strategy value_objects.RecoveryStrategy) *ErrorRecoveryEntity {
	return &ErrorRecoveryEntity{
		id:            id,
		originalError: originalError,
		strategy:      strategy,
		status:        StatusInProgress,
		attempts:      make([]*value_objects.ErrorRecoveryAttempt, 0),
		startedAt:     time.Now(),
	}
}

// ID 获取错误恢复实体的唯一标识
func (ere *ErrorRecoveryEntity) ID() string {
	return ere.id
}

// OriginalError 获取原始错误
func (ere *ErrorRecoveryEntity) OriginalError() *value_objects.ParseError {
	return ere.originalError
}

// Strategy 获取使用的恢复策略
func (ere *ErrorRecoveryEntity) Strategy() value_objects.RecoveryStrategy {
	return ere.strategy
}

// Status 获取恢复状态
func (ere *ErrorRecoveryEntity) Status() RecoveryStatus {
	return ere.status
}

// Attempts 获取所有恢复尝试
func (ere *ErrorRecoveryEntity) Attempts() []*value_objects.ErrorRecoveryAttempt {
	return ere.attempts
}

// RecoveredTokens 获取已恢复的Token数量
func (ere *ErrorRecoveryEntity) RecoveredTokens() int {
	return ere.recoveredTokens
}

// RecoveredNodes 获取已恢复的AST节点数量
func (ere *ErrorRecoveryEntity) RecoveredNodes() int {
	return ere.recoveredNodes
}

// StartedAt 获取开始时间
func (ere *ErrorRecoveryEntity) StartedAt() time.Time {
	return ere.startedAt
}

// LastAttemptedAt 获取最后尝试时间
func (ere *ErrorRecoveryEntity) LastAttemptedAt() *time.Time {
	return ere.lastAttemptedAt
}

// AddAttempt 添加一次恢复尝试
func (ere *ErrorRecoveryEntity) AddAttempt(attempt *value_objects.ErrorRecoveryAttempt) {
	ere.attempts = append(ere.attempts, attempt)
	now := time.Now()
	ere.lastAttemptedAt = &now

	if attempt.IsSuccessful() {
		ere.recoveredTokens += attempt.RecoveredTokens()
		ere.recoveredNodes += attempt.RecoveredNodes()
	}
}

// MarkSucceeded 标记恢复成功
func (ere *ErrorRecoveryEntity) MarkSucceeded() {
	ere.status = StatusSucceeded
}

// MarkFailed 标记恢复失败
func (ere *ErrorRecoveryEntity) MarkFailed() {
	ere.status = StatusFailed
}

// MarkAbandoned 标记恢复被放弃
func (ere *ErrorRecoveryEntity) MarkAbandoned() {
	ere.status = StatusAbandoned
}

// IsActive 检查恢复是否仍在进行
func (ere *ErrorRecoveryEntity) IsActive() bool {
	return ere.status == StatusInProgress
}

// IsSuccessful 检查恢复是否成功
func (ere *ErrorRecoveryEntity) IsSuccessful() bool {
	return ere.status == StatusSucceeded
}

// TotalAttempts 获取总尝试次数
func (ere *ErrorRecoveryEntity) TotalAttempts() int {
	return len(ere.attempts)
}

// SuccessfulAttempts 获取成功尝试次数
func (ere *ErrorRecoveryEntity) SuccessfulAttempts() int {
	count := 0
	for _, attempt := range ere.attempts {
		if attempt.IsSuccessful() {
			count++
		}
	}
	return count
}

// String 返回ErrorRecoveryEntity的字符串表示
func (ere *ErrorRecoveryEntity) String() string {
	return fmt.Sprintf("ErrorRecoveryEntity{ID: %s, Status: %s, Strategy: %s, Attempts: %d, RecoveredTokens: %d, RecoveredNodes: %d}",
		ere.id, ere.status, ere.strategy, len(ere.attempts), ere.recoveredTokens, ere.recoveredNodes)
}
