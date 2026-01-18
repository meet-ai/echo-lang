package analysis

import "fmt"

// Position 表示源码中的位置信息，是不可变的值对象
type Position struct {
	line   int    // 行号（1-based）
	column int    // 列号（1-based）
	file   string // 文件名
}

// NewPosition 创建新的Position
func NewPosition(line, column int, file string) Position {
	return Position{
		line:   line,
		column: column,
		file:   file,
	}
}

// Line 返回行号
func (p Position) Line() int {
	return p.line
}

// Column 返回列号
func (p Position) Column() int {
	return p.column
}

// File 返回文件名
func (p Position) File() string {
	return p.file
}

// String 返回Position的字符串表示
func (p Position) String() string {
	return fmt.Sprintf("%s:%d:%d", p.file, p.line, p.column)
}

// IsBefore 检查当前位置是否在另一个位置之前
func (p Position) IsBefore(other Position) bool {
	if p.file != other.file {
		return p.file < other.file
	}
	if p.line != other.line {
		return p.line < other.line
	}
	return p.column < other.column
}

// IsAfter 检查当前位置是否在另一个位置之后
func (p Position) IsAfter(other Position) bool {
	return other.IsBefore(p)
}

// Equals 检查两个Position是否相等（值对象相等性）
func (p Position) Equals(other Position) bool {
	return p.line == other.line &&
		p.column == other.column &&
		p.file == other.file
}

// NextColumn 返回下一列的位置
func (p Position) NextColumn() Position {
	return Position{
		line:   p.line,
		column: p.column + 1,
		file:   p.file,
	}
}

// NextLine 返回下一行的位置
func (p Position) NextLine() Position {
	return Position{
		line:   p.line + 1,
		column: 1,
		file:   p.file,
	}
}
