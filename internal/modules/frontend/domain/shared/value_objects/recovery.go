// Package value_objects 定义错误恢复相关值对象
package value_objects

import (
	"fmt"
	"time"
)

// ParseContext 解析上下文值对象
type ParseContext struct {
	sourceFile  *SourceFile
	tokenStream *TokenStream
	currentPos  int
	errorCount  int
	warningCount int
	recoveredErrors []*ParseError
	recoveryHistory []RecoveryAttempt
}

// RecoveryAttempt 恢复尝试记录
type RecoveryAttempt struct {
	errorPosition SourceLocation
	strategy      RecoveryStrategy
	success       bool
	appliedAt     time.Time
}

// NewParseContext 创建新的解析上下文
func NewParseContext(sourceFile *SourceFile, tokenStream *TokenStream) *ParseContext {
	return &ParseContext{
		sourceFile:       sourceFile,
		tokenStream:      tokenStream,
		currentPos:       0,
		errorCount:       0,
		warningCount:     0,
		recoveredErrors:  make([]*ParseError, 0),
		recoveryHistory: make([]RecoveryAttempt, 0),
	}
}

// SourceFile 获取源文件
func (pc *ParseContext) SourceFile() *SourceFile {
	return pc.sourceFile
}

// TokenStream 获取Token流
func (pc *ParseContext) TokenStream() *TokenStream {
	return pc.tokenStream
}

// CurrentPosition 获取当前位置
func (pc *ParseContext) CurrentPosition() int {
	return pc.currentPos
}

// SetCurrentPosition 设置当前位置
func (pc *ParseContext) SetCurrentPosition(pos int) {
	pc.currentPos = pos
}

// IncrementErrorCount 增加错误计数
func (pc *ParseContext) IncrementErrorCount() {
	pc.errorCount++
}

// IncrementWarningCount 增加警告计数
func (pc *ParseContext) IncrementWarningCount() {
	pc.warningCount++
}

// ErrorCount 获取错误计数
func (pc *ParseContext) ErrorCount() int {
	return pc.errorCount
}

// WarningCount 获取警告计数
func (pc *ParseContext) WarningCount() int {
	return pc.warningCount
}

// AddRecoveredError 添加已恢复的错误
func (pc *ParseContext) AddRecoveredError(err *ParseError) {
	pc.recoveredErrors = append(pc.recoveredErrors, err)
}

// RecoveredErrors 获取已恢复的错误列表
func (pc *ParseContext) RecoveredErrors() []*ParseError {
	return pc.recoveredErrors
}

// AddRecoveryAttempt 添加恢复尝试记录
func (pc *ParseContext) AddRecoveryAttempt(attempt RecoveryAttempt) {
	pc.recoveryHistory = append(pc.recoveryHistory, attempt)
}

// RecoveryHistory 获取恢复历史
func (pc *ParseContext) RecoveryHistory() []RecoveryAttempt {
	return pc.recoveryHistory
}

// NewRecoveryAttempt 创建新的恢复尝试记录
func NewRecoveryAttempt(errorPosition SourceLocation, strategy RecoveryStrategy, success bool) RecoveryAttempt {
	return RecoveryAttempt{
		errorPosition: errorPosition,
		strategy:      strategy,
		success:       success,
		appliedAt:     time.Now(),
	}
}

// ErrorPosition 获取错误位置
func (ra RecoveryAttempt) ErrorPosition() SourceLocation {
	return ra.errorPosition
}

// Strategy 获取恢复策略
func (ra RecoveryAttempt) Strategy() RecoveryStrategy {
	return ra.strategy
}

// Success 获取恢复是否成功
func (ra RecoveryAttempt) Success() bool {
	return ra.success
}

// AppliedAt 获取应用时间
func (ra RecoveryAttempt) AppliedAt() time.Time {
	return ra.appliedAt
}

// RecoveryResult 恢复结果值对象
type RecoveryResult struct {
	originalError   *ParseError
	recoveryStrategy RecoveryStrategy
	recovered       bool
	newPosition     int
	suggestedFixes  []*FixSuggestion
	recoveredAt     time.Time
}

// NewRecoveryResult 创建新的恢复结果
func NewRecoveryResult(originalError *ParseError, strategy RecoveryStrategy) *RecoveryResult {
	return &RecoveryResult{
		originalError:   originalError,
		recoveryStrategy: strategy,
		recovered:       false,
		suggestedFixes:  make([]*FixSuggestion, 0),
		recoveredAt:     time.Now(),
	}
}

// OriginalError 获取原始错误
func (rr *RecoveryResult) OriginalError() *ParseError {
	return rr.originalError
}

// RecoveryStrategy 获取恢复策略
func (rr *RecoveryResult) RecoveryStrategy() RecoveryStrategy {
	return rr.recoveryStrategy
}

// Recovered 获取恢复状态
func (rr *RecoveryResult) Recovered() bool {
	return rr.recovered
}

// SetRecovered 设置恢复状态
func (rr *RecoveryResult) SetRecovered(recovered bool) {
	rr.recovered = recovered
}

// NewPosition 获取新的位置
func (rr *RecoveryResult) NewPosition() int {
	return rr.newPosition
}

// SetNewPosition 设置新的位置
func (rr *RecoveryResult) SetNewPosition(pos int) {
	rr.newPosition = pos
}

// SuggestedFixes 获取建议的修复
func (rr *RecoveryResult) SuggestedFixes() []*FixSuggestion {
	return rr.suggestedFixes
}

// AddSuggestedFix 添加建议的修复
func (rr *RecoveryResult) AddSuggestedFix(fix *FixSuggestion) {
	rr.suggestedFixes = append(rr.suggestedFixes, fix)
}

// RecoveredAt 获取恢复时间
func (rr *RecoveryResult) RecoveredAt() time.Time {
	return rr.recoveredAt
}

// RecoveryStrategy 恢复策略值对象
type RecoveryStrategy struct {
	strategyType RecoveryStrategyType
	description  string
	priority     int
	maxAttempts  int
}

// RecoveryStrategyType 恢复策略类型
type RecoveryStrategyType string

const (
	StrategyTypePanicMode    RecoveryStrategyType = "panic_mode"     // 恐慌模式：跳过错误块
	StrategyTypeInsertion    RecoveryStrategyType = "insertion"       // 插入：添加缺失的标记
	StrategyTypeReplacement  RecoveryStrategyType = "replacement"     // 替换：替换错误的标记
	StrategyTypeDeletion     RecoveryStrategyType = "deletion"        // 删除：删除多余的标记
)

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

// Priority 获取策略优先级
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

// FixSuggestion 修复建议值对象
type FixSuggestion struct {
	description string
	fixType     FixType
	location    SourceLocation
	oldText     string
	newText     string
	confidence  float64
}

// FixType 修复类型
type FixType string

const (
	FixTypeInsert    FixType = "insert"    // 插入
	FixTypeReplace   FixType = "replace"   // 替换
	FixTypeDelete    FixType = "delete"    // 删除
	FixTypeReorder   FixType = "reorder"   // 重新排序
)

// NewFixSuggestion 创建新的修复建议
func NewFixSuggestion(description string, fixType FixType, location SourceLocation, oldText, newText string, confidence float64) *FixSuggestion {
	return &FixSuggestion{
		description: description,
		fixType:     fixType,
		location:    location,
		oldText:     oldText,
		newText:     newText,
		confidence:  confidence,
	}
}

// Description 获取修复描述
func (fs *FixSuggestion) Description() string {
	return fs.description
}

// FixType 获取修复类型
func (fs *FixSuggestion) FixType() FixType {
	return fs.fixType
}

// Location 获取修复位置
func (fs *FixSuggestion) Location() SourceLocation {
	return fs.location
}

// OldText 获取原始文本
func (fs *FixSuggestion) OldText() string {
	return fs.oldText
}

// NewText 获取新文本
func (fs *FixSuggestion) NewText() string {
	return fs.newText
}

// Confidence 获取置信度
func (fs *FixSuggestion) Confidence() float64 {
	return fs.confidence
}

// AppliedRecovery 已应用的恢复值对象
type AppliedRecovery struct {
	recoveryResult *RecoveryResult
	appliedFixes   []*FixSuggestion
	resultAST      *ProgramAST
	appliedAt      time.Time
}

// NewAppliedRecovery 创建新的已应用恢复
func NewAppliedRecovery(recoveryResult *RecoveryResult, appliedFixes []*FixSuggestion, resultAST *ProgramAST) *AppliedRecovery {
	return &AppliedRecovery{
		recoveryResult: recoveryResult,
		appliedFixes:   appliedFixes,
		resultAST:      resultAST,
		appliedAt:      time.Now(),
	}
}

// RecoveryResult 获取恢复结果
func (ar *AppliedRecovery) RecoveryResult() *RecoveryResult {
	return ar.recoveryResult
}

// AppliedFixes 获取已应用的修复
func (ar *AppliedRecovery) AppliedFixes() []*FixSuggestion {
	return ar.appliedFixes
}

// ResultAST 获取恢复后的AST
func (ar *AppliedRecovery) ResultAST() *ProgramAST {
	return ar.resultAST
}

// AppliedAt 获取应用时间
func (ar *AppliedRecovery) AppliedAt() time.Time {
	return ar.appliedAt
}

// ErrorRecoveryAttempt 记录一次错误恢复尝试（用于ErrorRecoveryEntity）
type ErrorRecoveryAttempt struct {
	originalError *ParseError
	strategy      RecoveryStrategy
	success       bool
	recoveredTokens int
	recoveredNodes  int
	attemptedAt   time.Time
	details       string
}

// NewErrorRecoveryAttempt 创建新的ErrorRecoveryAttempt
func NewErrorRecoveryAttempt(originalError *ParseError, strategy RecoveryStrategy, details string) *ErrorRecoveryAttempt {
	return &ErrorRecoveryAttempt{
		originalError: originalError,
		strategy:      strategy,
		success:       false,
		attemptedAt:   time.Now(),
		details:       details,
	}
}

// MarkSuccess 标记恢复成功
func (era *ErrorRecoveryAttempt) MarkSuccess(recoveredTokens, recoveredNodes int) {
	era.success = true
	era.recoveredTokens = recoveredTokens
	era.recoveredNodes = recoveredNodes
}

// OriginalError 获取原始错误
func (era *ErrorRecoveryAttempt) OriginalError() *ParseError {
	return era.originalError
}

// Strategy 获取使用的恢复策略
func (era *ErrorRecoveryAttempt) Strategy() RecoveryStrategy {
	return era.strategy
}

// IsSuccessful 检查恢复是否成功
func (era *ErrorRecoveryAttempt) IsSuccessful() bool {
	return era.success
}

// RecoveredTokens 获取恢复的Token数量
func (era *ErrorRecoveryAttempt) RecoveredTokens() int {
	return era.recoveredTokens
}

// RecoveredNodes 获取恢复的AST节点数量
func (era *ErrorRecoveryAttempt) RecoveredNodes() int {
	return era.recoveredNodes
}

// Details 获取恢复尝试的详细信息
func (era *ErrorRecoveryAttempt) Details() string {
	return era.details
}

// String 返回ErrorRecoveryAttempt的字符串表示
func (era *ErrorRecoveryAttempt) String() string {
	status := "Failed"
	if era.success {
		status = fmt.Sprintf("Successful (Tokens: %d, Nodes: %d)", era.recoveredTokens, era.recoveredNodes)
	}
	return fmt.Sprintf("Recovery Attempt: Strategy=%s, Status=%s, OriginalError=[%s], Details='%s'",
		era.strategy, status, era.originalError.Error(), era.details)
}

// ErrorRecoveryResult 值对象，表示错误恢复的整体结果
type ErrorRecoveryResult struct {
	attempts    []*ErrorRecoveryAttempt
	hasRecovered bool
	recoveredAt time.Time
}

// NewErrorRecoveryResult 创建新的ErrorRecoveryResult
func NewErrorRecoveryResult() *ErrorRecoveryResult {
	return &ErrorRecoveryResult{
		attempts:    make([]*ErrorRecoveryAttempt, 0),
		hasRecovered: false,
		recoveredAt: time.Now(),
	}
}

// HasRecovered 检查是否有任何恢复尝试成功
func (err *ErrorRecoveryResult) HasRecovered() bool {
	return err.hasRecovered
}

// Attempts 获取所有恢复尝试
func (err *ErrorRecoveryResult) Attempts() []*ErrorRecoveryAttempt {
	return err.attempts
}

// AddAttempt 添加恢复尝试
func (err *ErrorRecoveryResult) AddAttempt(attempt *ErrorRecoveryAttempt) {
	err.attempts = append(err.attempts, attempt)
	if attempt.IsSuccessful() {
		err.hasRecovered = true
	}
}

// String 返回ErrorRecoveryResult的字符串表示
func (err *ErrorRecoveryResult) String() string {
	s := fmt.Sprintf("Error Recovery Result (Has Recovered: %t) at %s\n",
		err.hasRecovered, err.recoveredAt.Format(time.RFC3339))
	if len(err.attempts) == 0 {
		s += "  No recovery attempts made.\n"
	} else {
		for _, attempt := range err.attempts {
			s += "  " + attempt.String() + "\n"
		}
	}
	return s
}
