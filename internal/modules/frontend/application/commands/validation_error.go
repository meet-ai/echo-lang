package commands

import (
	"fmt"
)

// ValidationError 验证错误
type ValidationError struct {
	Field   string
	Message string
}

// NewValidationError 创建验证错误
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// Error 实现error接口
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// FieldName 返回错误字段名
func (e *ValidationError) FieldName() string {
	return e.Field
}

// ErrorMessage 返回错误消息
func (e *ValidationError) ErrorMessage() string {
	return e.Message
}
