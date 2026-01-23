// Package value_objects 定义共享的值对象
package value_objects

import (
	"strings"
	"time"
)

// SourceFile 源文件值对象
// 表示待解析的源代码文件
type SourceFile struct {
	filename  string
	content   string
	createdAt time.Time
}

// NewSourceFile 创建新的源文件
func NewSourceFile(filename, content string) *SourceFile {
	return &SourceFile{
		filename:  filename,
		content:   content,
		createdAt: time.Now(),
	}
}

// Filename 获取文件名
func (sf *SourceFile) Filename() string {
	return sf.filename
}

// Content 获取文件内容
func (sf *SourceFile) Content() string {
	return sf.content
}

// CreatedAt 获取创建时间
func (sf *SourceFile) CreatedAt() time.Time {
	return sf.createdAt
}

// Size 获取文件大小
func (sf *SourceFile) Size() int {
	return len(sf.content)
}

// Lines 获取文件行数
func (sf *SourceFile) Lines() []string {
	// 简单的行分割实现
	// TODO: 处理不同的换行符
	return strings.Split(sf.content, "\n")
}
