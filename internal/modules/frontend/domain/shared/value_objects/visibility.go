// Package value_objects 定义可见性相关值对象
package value_objects

// Visibility 可见性枚举
type Visibility int

const (
	// VisibilityPublic 公开（默认）
	VisibilityPublic Visibility = iota
	// VisibilityPrivate 私有
	VisibilityPrivate
)

// String 返回可见性字符串
func (v Visibility) String() string {
	switch v {
	case VisibilityPublic:
		return "public"
	case VisibilityPrivate:
		return "private"
	default:
		return "unknown"
	}
}

// IsPublic 检查是否为公开
func (v Visibility) IsPublic() bool {
	return v == VisibilityPublic
}

// IsPrivate 检查是否为私有
func (v Visibility) IsPrivate() bool {
	return v == VisibilityPrivate
}

