package exceptions

import (
	"fmt"
)

// AnalysisException 分析异常
type AnalysisException struct {
	// 异常基本信息
	message      string
	cause        error
	sourceFileID string

	// 分析上下文
	analysisType string
	line         int
	column       int
	offset       int

	// 异常分类
	exceptionType ExceptionType
	severity      Severity

	// 恢复信息
	recoverable bool
	suggestions []string
}

// ExceptionType 异常类型
type ExceptionType string

const (
	ExceptionTypeLexical  ExceptionType = "lexical"
	ExceptionTypeSyntax   ExceptionType = "syntax"
	ExceptionTypeSemantic ExceptionType = "semantic"
	ExceptionTypeInternal ExceptionType = "internal"
)

// Severity 严重程度
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// NewAnalysisException 创建分析异常
func NewAnalysisException(message string, exceptionType ExceptionType, severity Severity) *AnalysisException {
	return &AnalysisException{
		message:       message,
		exceptionType: exceptionType,
		severity:      severity,
		recoverable:   severity != SeverityCritical,
		suggestions:   make([]string, 0),
	}
}

// NewLexicalException 创建词法分析异常
func NewLexicalException(message string, line, column, offset int) *AnalysisException {
	exc := NewAnalysisException(message, ExceptionTypeLexical, SeverityHigh)
	exc.line = line
	exc.column = column
	exc.offset = offset
	return exc
}

// NewSyntaxException 创建语法分析异常
func NewSyntaxException(message string, line, column, offset int) *AnalysisException {
	exc := NewAnalysisException(message, ExceptionTypeSyntax, SeverityHigh)
	exc.line = line
	exc.column = column
	exc.offset = offset
	return exc
}

// NewSemanticException 创建语义分析异常
func NewSemanticException(message string, severity Severity) *AnalysisException {
	return NewAnalysisException(message, ExceptionTypeSemantic, severity)
}

// NewInternalException 创建内部异常
func NewInternalException(message string, cause error) *AnalysisException {
	exc := NewAnalysisException(message, ExceptionTypeInternal, SeverityCritical)
	exc.cause = cause
	return exc
}

// Error 实现error接口
func (e *AnalysisException) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.message, e.cause)
	}

	location := ""
	if e.line > 0 {
		location = fmt.Sprintf(" at line %d, column %d", e.line, e.column)
		if e.offset > 0 {
			location += fmt.Sprintf(" (offset %d)", e.offset)
		}
	}

	return fmt.Sprintf("[%s] %s%s", e.exceptionType, e.message, location)
}

// Message 返回异常消息
func (e *AnalysisException) Message() string {
	return e.message
}

// Cause 返回根本原因
func (e *AnalysisException) Cause() error {
	return e.cause
}

// SourceFileID 返回源文件ID
func (e *AnalysisException) SourceFileID() string {
	return e.sourceFileID
}

// SetSourceFileID 设置源文件ID
func (e *AnalysisException) SetSourceFileID(sourceFileID string) {
	e.sourceFileID = sourceFileID
}

// AnalysisType 返回分析类型
func (e *AnalysisException) AnalysisType() string {
	return string(e.exceptionType)
}

// Line 返回行号
func (e *AnalysisException) Line() int {
	return e.line
}

// Column 返回列号
func (e *AnalysisException) Column() int {
	return e.column
}

// Offset 返回偏移量
func (e *AnalysisException) Offset() int {
	return e.offset
}

// ExceptionType 返回异常类型
func (e *AnalysisException) ExceptionType() ExceptionType {
	return e.exceptionType
}

// Severity 返回严重程度
func (e *AnalysisException) Severity() Severity {
	return e.severity
}

// IsRecoverable 返回是否可恢复
func (e *AnalysisException) IsRecoverable() bool {
	return e.recoverable
}

// Suggestions 返回修复建议
func (e *AnalysisException) Suggestions() []string {
	return append([]string(nil), e.suggestions...)
}

// AddSuggestion 添加修复建议
func (e *AnalysisException) AddSuggestion(suggestion string) {
	e.suggestions = append(e.suggestions, suggestion)
}

// SetSuggestions 设置修复建议
func (e *AnalysisException) SetSuggestions(suggestions []string) {
	e.suggestions = make([]string, len(suggestions))
	copy(e.suggestions, suggestions)
}

// Location 返回位置信息
func (e *AnalysisException) Location() string {
	if e.line > 0 {
		return fmt.Sprintf("line %d, column %d", e.line, e.column)
	}
	return "unknown location"
}

// IsLexical 是否为词法异常
func (e *AnalysisException) IsLexical() bool {
	return e.exceptionType == ExceptionTypeLexical
}

// IsSyntax 是否为语法异常
func (e *AnalysisException) IsSyntax() bool {
	return e.exceptionType == ExceptionTypeSyntax
}

// IsSemantic 是否为语义异常
func (e *AnalysisException) IsSemantic() bool {
	return e.exceptionType == ExceptionTypeSemantic
}

// IsInternal 是否为内部异常
func (e *AnalysisException) IsInternal() bool {
	return e.exceptionType == ExceptionTypeInternal
}

// IsCritical 是否为严重异常
func (e *AnalysisException) IsCritical() bool {
	return e.severity == SeverityCritical
}

// Wrap 包装另一个错误
func (e *AnalysisException) Wrap(err error) *AnalysisException {
	e.cause = err
	return e
}

// Unwrap 解包错误（实现errors包的Unwrap接口）
func (e *AnalysisException) Unwrap() error {
	return e.cause
}
