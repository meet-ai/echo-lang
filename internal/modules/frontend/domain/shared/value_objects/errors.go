// Package value_objects 定义错误相关值对象
package value_objects

// ParseError 解析错误值对象
type ParseError struct {
	message  string
	location SourceLocation
	errorType ParseErrorType
	severity ParseErrorSeverity
	context  string
}

// ParseErrorType 解析错误类型
type ParseErrorType string

const (
	ErrorTypeLexical    ParseErrorType = "lexical"    // 词法错误
	ErrorTypeSyntax     ParseErrorType = "syntax"     // 语法错误
	ErrorTypeSemantic   ParseErrorType = "semantic"   // 语义错误
	ErrorTypeType       ParseErrorType = "type"       // 类型错误
	ErrorTypeSymbol     ParseErrorType = "symbol"     // 符号错误
	ErrorTypeRecovery   ParseErrorType = "recovery"   // 恢复错误
)

// ParseErrorSeverity 解析错误严重程度
type ParseErrorSeverity string

const (
	SeverityError   ParseErrorSeverity = "error"   // 错误
	SeverityWarning ParseErrorSeverity = "warning" // 警告
	SeverityInfo    ParseErrorSeverity = "info"    // 信息
)

// NewParseError 创建新的解析错误
func NewParseError(message string, location SourceLocation, errorType ParseErrorType, severity ParseErrorSeverity) *ParseError {
	return &ParseError{
		message:  message,
		location: location,
		errorType: errorType,
		severity: severity,
	}
}

// Message 获取错误消息
func (pe *ParseError) Message() string {
	return pe.message
}

// Location 获取错误位置
func (pe *ParseError) Location() SourceLocation {
	return pe.location
}

// ErrorType 获取错误类型
func (pe *ParseError) ErrorType() ParseErrorType {
	return pe.errorType
}

// Severity 获取错误严重程度
func (pe *ParseError) Severity() ParseErrorSeverity {
	return pe.severity
}

// Context 获取错误上下文
func (pe *ParseError) Context() string {
	return pe.context
}

// SetContext 设置错误上下文
func (pe *ParseError) SetContext(context string) {
	pe.context = context
}

// String 返回错误的字符串表示
func (pe *ParseError) String() string {
	return pe.location.String() + ": " + pe.message
}

// Error 实现error接口
func (pe *ParseError) Error() string {
	return pe.String()
}

// ParseWarning 解析警告值对象
type ParseWarning struct {
	message  string
	location SourceLocation
	warningType ParseWarningType
	context  string
}

// ParseWarningType 解析警告类型
type ParseWarningType string

const (
	WarningTypeUnusedVariable ParseWarningType = "unused_variable" // 未使用的变量
	WarningTypeUnusedFunction ParseWarningType = "unused_function" // 未使用的函数
	WarningTypeTypeConversion ParseWarningType = "type_conversion" // 类型转换警告
	WarningTypePerformance    ParseWarningType = "performance"     // 性能警告
)

// NewParseWarning 创建新的解析警告
func NewParseWarning(message string, location SourceLocation, warningType ParseWarningType) *ParseWarning {
	return &ParseWarning{
		message:     message,
		location:    location,
		warningType: warningType,
	}
}

// Message 获取警告消息
func (pw *ParseWarning) Message() string {
	return pw.message
}

// Location 获取警告位置
func (pw *ParseWarning) Location() SourceLocation {
	return pw.location
}

// WarningType 获取警告类型
func (pw *ParseWarning) WarningType() ParseWarningType {
	return pw.warningType
}

// Context 获取警告上下文
func (pw *ParseWarning) Context() string {
	return pw.context
}

// SetContext 设置警告上下文
func (pw *ParseWarning) SetContext(context string) {
	pw.context = context
}

// String 返回警告的字符串表示
func (pw *ParseWarning) String() string {
	return pw.location.String() + ": warning: " + pw.message
}
