package analysis

import "fmt"

// DiagnosticType 表示诊断信息的类型
type DiagnosticType int

const (
	Info DiagnosticType = iota
	Warning
	Error
)

// String 返回DiagnosticType的字符串表示
func (t DiagnosticType) String() string {
	switch t {
	case Info:
		return "info"
	case Warning:
		return "warning"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}

// Diagnostic 表示分析过程中的诊断信息，是不可变的值对象
type Diagnostic struct {
	type_    DiagnosticType // 诊断类型
	message  string         // 诊断消息
	position Position       // 位置信息
	code     string         // 诊断代码（可选）
}

// NewDiagnostic 创建新的Diagnostic
func NewDiagnostic(type_ DiagnosticType, message string, position Position) Diagnostic {
	return Diagnostic{
		type_:    type_,
		message:  message,
		position: position,
	}
}

// NewDiagnosticWithCode 创建带有诊断代码的Diagnostic
func NewDiagnosticWithCode(type_ DiagnosticType, message string, position Position, code string) Diagnostic {
	return Diagnostic{
		type_:    type_,
		message:  message,
		position: position,
		code:     code,
	}
}

// Type 返回诊断类型
func (d Diagnostic) Type() DiagnosticType {
	return d.type_
}

// Message 返回诊断消息
func (d Diagnostic) Message() string {
	return d.message
}

// Position 返回位置信息
func (d Diagnostic) Position() Position {
	return d.position
}

// Code 返回诊断代码
func (d Diagnostic) Code() string {
	return d.code
}

// IsError 检查是否为错误
func (d Diagnostic) IsError() bool {
	return d.type_ == Error
}

// IsWarning 检查是否为警告
func (d Diagnostic) IsWarning() bool {
	return d.type_ == Warning
}

// IsInfo 检查是否为信息
func (d Diagnostic) IsInfo() bool {
	return d.type_ == Info
}

// String 返回Diagnostic的字符串表示
func (d Diagnostic) String() string {
	if d.code != "" {
		return fmt.Sprintf("[%s:%s] %s at %s", d.type_.String(), d.code, d.message, d.position.String())
	}
	return fmt.Sprintf("[%s] %s at %s", d.type_.String(), d.message, d.position.String())
}

// Equals 检查两个Diagnostic是否相等（值对象相等性）
func (d Diagnostic) Equals(other Diagnostic) bool {
	return d.type_ == other.type_ &&
		d.message == other.message &&
		d.position.Equals(other.position) &&
		d.code == other.code
}
