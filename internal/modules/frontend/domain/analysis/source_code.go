package analysis

import (
	"fmt"
	"strings"
)

// SourceCode 表示源代码内容，是不可变的值对象
type SourceCode struct {
	content  string
	filePath string
}

// NewSourceCode 创建新的SourceCode
func NewSourceCode(content, filePath string) SourceCode {
	return SourceCode{
		content:  content,
		filePath: filePath,
	}
}

// Content 返回源代码内容
func (sc SourceCode) Content() string {
	return sc.content
}

// FilePath 返回文件路径
func (sc SourceCode) FilePath() string {
	return sc.filePath
}

// Lines 返回按行分割的源代码
func (sc SourceCode) Lines() []string {
	return strings.Split(sc.content, "\n")
}

// LineCount 返回总行数
func (sc SourceCode) LineCount() int {
	return len(sc.Lines())
}

// LineAt 返回指定行的内容（1-based）
func (sc SourceCode) LineAt(lineNum int) (string, error) {
	lines := sc.Lines()
	if lineNum < 1 || lineNum > len(lines) {
		return "", fmt.Errorf("line number %d out of range [1, %d]", lineNum, len(lines))
	}
	return lines[lineNum-1], nil
}

// PositionAt 返回指定位置的字符（1-based）
func (sc SourceCode) PositionAt(line, column int) (rune, error) {
	lineContent, err := sc.LineAt(line)
	if err != nil {
		return 0, err
	}

	runes := []rune(lineContent)
	if column < 1 || column > len(runes) {
		return 0, fmt.Errorf("column %d out of range [1, %d] in line %d", column, len(runes), line)
	}

	return runes[column-1], nil
}

// Substring 返回指定范围的子串
func (sc SourceCode) Substring(startLine, startColumn, endLine, endColumn int) (string, error) {
	if startLine > endLine || (startLine == endLine && startColumn > endColumn) {
		return "", fmt.Errorf("invalid range: start (%d,%d) > end (%d,%d)",
			startLine, startColumn, endLine, endColumn)
	}

	lines := sc.Lines()
	if startLine < 1 || endLine > len(lines) {
		return "", fmt.Errorf("line range out of bounds")
	}

	if startLine == endLine {
		line := lines[startLine-1]
		runes := []rune(line)
		if startColumn < 1 || endColumn > len(runes) {
			return "", fmt.Errorf("column range out of bounds")
		}
		return string(runes[startColumn-1:endColumn]), nil
	}

	// 多行情况
	var result strings.Builder
	result.WriteString(lines[startLine-1][startColumn-1:]) // 第一行
	result.WriteString("\n")

	for i := startLine + 1; i < endLine; i++ {
		result.WriteString(lines[i-1])
		result.WriteString("\n")
	}

	if endLine <= len(lines) {
		lastLine := lines[endLine-1]
		runes := []rune(lastLine)
		if endColumn <= len(runes) {
			result.WriteString(string(runes[:endColumn]))
		}
	}

	return result.String(), nil
}

// Size 返回源代码的大小（字符数）
func (sc SourceCode) Size() int {
	return len(sc.content)
}

// IsEmpty 检查是否为空
func (sc SourceCode) IsEmpty() bool {
	return len(strings.TrimSpace(sc.content)) == 0
}

// String 返回SourceCode的字符串表示
func (sc SourceCode) String() string {
	return fmt.Sprintf("SourceCode{file=%s, size=%d, lines=%d}",
		sc.filePath, sc.Size(), sc.LineCount())
}

// Equals 检查两个SourceCode是否相等（值对象相等性）
func (sc SourceCode) Equals(other SourceCode) bool {
	return sc.content == other.content && sc.filePath == other.filePath
}
