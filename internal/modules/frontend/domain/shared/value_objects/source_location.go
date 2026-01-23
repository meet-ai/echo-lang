// Package value_objects 定义位置相关值对象
package value_objects

import "fmt"

// SourceLocation 源代码位置值对象
// 表示代码中的具体位置信息
type SourceLocation struct {
	filename string
	line     int
	column   int
	offset   int
}

// NewSourceLocation 创建新的位置信息
func NewSourceLocation(filename string, line, column, offset int) SourceLocation {
	return SourceLocation{
		filename: filename,
		line:     line,
		column:   column,
		offset:   offset,
	}
}

// Filename 获取文件名
func (sl SourceLocation) Filename() string {
	return sl.filename
}

// Line 获取行号
func (sl SourceLocation) Line() int {
	return sl.line
}

// Column 获取列号
func (sl SourceLocation) Column() int {
	return sl.column
}

// Offset 获取偏移量
func (sl SourceLocation) Offset() int {
	return sl.offset
}

// String 返回位置的字符串表示
func (sl SourceLocation) String() string {
	return fmt.Sprintf("%s:%d:%d", sl.filename, sl.line, sl.column)
}

// IsValid 检查位置是否有效
func (sl SourceLocation) IsValid() bool {
	return sl.line > 0 && sl.column > 0 && sl.offset >= 0
}

// AdvanceColumn 前进一列
func (sl SourceLocation) AdvanceColumn() SourceLocation {
	return SourceLocation{
		filename: sl.filename,
		line:     sl.line,
		column:   sl.column + 1,
		offset:   sl.offset + 1,
	}
}

// AdvanceLine 前进一行
func (sl SourceLocation) AdvanceLine() SourceLocation {
	return SourceLocation{
		filename: sl.filename,
		line:     sl.line + 1,
		column:   1,
		offset:   sl.offset + 1,
	}
}
