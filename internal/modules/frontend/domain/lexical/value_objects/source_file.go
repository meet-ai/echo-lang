// Package value_objects 定义词法分析上下文的值对象
package value_objects

import (
	"path/filepath"
	"strings"
)

// SourceFile 源文件值对象
// 词法分析上下文专用的源文件表示，区别于共享的SourceFile
type SourceFile struct {
	filename string
	content  string
}

// NewSourceFile 创建新的源文件值对象
func NewSourceFile(filename string, content string) *SourceFile {
	return &SourceFile{
		filename: filename,
		content:  content,
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

// Extension 获取文件扩展名
func (sf *SourceFile) Extension() string {
	return filepath.Ext(sf.filename)
}

// BaseName 获取文件名（不含路径）
func (sf *SourceFile) BaseName() string {
	return filepath.Base(sf.filename)
}

// Lines 将内容按行分割
func (sf *SourceFile) Lines() []string {
	return strings.Split(sf.content, "\n")
}

// LineCount 获取行数
func (sf *SourceFile) LineCount() int {
	return len(sf.Lines())
}

// Size 获取文件大小（字节）
func (sf *SourceFile) Size() int {
	return len(sf.content)
}

// IsEmpty 检查文件是否为空
func (sf *SourceFile) IsEmpty() bool {
	return sf.content == ""
}

// String 返回源文件的字符串表示
func (sf *SourceFile) String() string {
	return sf.filename
}
