package analysis

import (
	"testing"
)

func TestNewSourceCode(t *testing.T) {
	content := "func add(a: int, b: int) -> int {\n    return a + b\n}"
	filePath := "test.eo"

	source := NewSourceCode(content, filePath)

	if source.Content() != content {
		t.Errorf("Expected content %q, got %q", content, source.Content())
	}

	if source.FilePath() != filePath {
		t.Errorf("Expected file path %q, got %q", filePath, source.FilePath())
	}
}

func TestSourceCode_LineCount(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{"single line", "func add(a: int) -> int { return a }", 1},
		{"multiple lines", "func add(a: int) -> int {\n    return a\n}", 3},
		{"empty content", "", 1}, // strings.Split("", "\n") returns [""] which has length 1
		{"only newline", "\n", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := NewSourceCode(tt.content, "test.eo")
			if got := source.LineCount(); got != tt.expected {
				t.Errorf("SourceCode.LineCount() = %d, expected %d", got, tt.expected)
			}
		})
	}
}

func TestSourceCode_LineAt(t *testing.T) {
	content := "line 1\nline 2\nline 3"
	source := NewSourceCode(content, "test.eo")

	tests := []struct {
		name        string
		lineNum     int
		expected    string
		expectError bool
	}{
		{"first line", 1, "line 1", false},
		{"second line", 2, "line 2", false},
		{"third line", 3, "line 3", false},
		{"line 0", 0, "", true},
		{"line 4", 4, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, err := source.LineAt(tt.lineNum)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for line %d, but got none", tt.lineNum)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for line %d: %v", tt.lineNum, err)
				}
				if line != tt.expected {
					t.Errorf("SourceCode.LineAt(%d) = %q, expected %q", tt.lineNum, line, tt.expected)
				}
			}
		})
	}
}

func TestSourceCode_PositionAt(t *testing.T) {
	content := "hello\nworld"
	source := NewSourceCode(content, "test.eo")

	tests := []struct {
		name        string
		line        int
		column      int
		expected    rune
		expectError bool
	}{
		{"first char of first line", 1, 1, 'h', false},
		{"second char of first line", 1, 2, 'e', false},
		{"first char of second line", 2, 1, 'w', false},
		{"line 0", 0, 1, 0, true},
		{"line 3", 3, 1, 0, true},
		{"column 0", 1, 0, 0, true},
		{"column beyond line", 1, 10, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			char, err := source.PositionAt(tt.line, tt.column)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for position (%d,%d), but got none", tt.line, tt.column)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for position (%d,%d): %v", tt.line, tt.column, err)
				}
				if char != tt.expected {
					t.Errorf("SourceCode.PositionAt(%d, %d) = %c, expected %c", tt.line, tt.column, char, tt.expected)
				}
			}
		})
	}
}

func TestSourceCode_Substring(t *testing.T) {
	content := "hello\nworld\ntest"
	source := NewSourceCode(content, "test.eo")

	tests := []struct {
		name        string
		startLine    int
		startColumn int
		endLine      int
		endColumn   int
		expected    string
		expectError bool
	}{
		{"single line substring", 1, 1, 1, 3, "hel", false},
		{"full first line", 1, 1, 1, 5, "hello", false},
		{"multi-line substring", 1, 3, 2, 3, "llo\nwor", false},
		{"single character", 2, 2, 2, 2, "o", false},
		{"invalid range", 2, 1, 1, 1, "", true},
		{"line out of bounds", 0, 1, 1, 1, "", true},
		{"column out of bounds", 1, 10, 1, 11, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			substring, err := source.Substring(tt.startLine, tt.startColumn, tt.endLine, tt.endColumn)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for range (%d,%d)-(%d,%d), but got none",
						tt.startLine, tt.startColumn, tt.endLine, tt.endColumn)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for range (%d,%d)-(%d,%d): %v",
						tt.startLine, tt.startColumn, tt.endLine, tt.endColumn, err)
				}
				if substring != tt.expected {
					t.Errorf("SourceCode.Substring(%d,%d,%d,%d) = %q, expected %q",
						tt.startLine, tt.startColumn, tt.endLine, tt.endColumn, substring, tt.expected)
				}
			}
		})
	}
}

func TestSourceCode_Size(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{"empty", "", 0},
		{"single char", "a", 1},
		{"with newline", "hello\nworld", 11},
		{"unicode", "你好", 6}, // 3 bytes per Chinese character
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := NewSourceCode(tt.content, "test.eo")
			if got := source.Size(); got != tt.expected {
				t.Errorf("SourceCode.Size() = %d, expected %d", got, tt.expected)
			}
		})
	}
}

func TestSourceCode_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{"empty", "", true},
		{"whitespace only", "   \n\t  ", true},
		{"with content", "hello", false},
		{"with whitespace and content", "  hello  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := NewSourceCode(tt.content, "test.eo")
			if got := source.IsEmpty(); got != tt.expected {
				t.Errorf("SourceCode.IsEmpty() = %v, expected %v for content %q", got, tt.expected, tt.content)
			}
		})
	}
}

func TestSourceCode_String(t *testing.T) {
	source := NewSourceCode("hello\nworld", "test.eo")
	expected := "SourceCode{file=test.eo, size=11, lines=2}"

	if got := source.String(); got != expected {
		t.Errorf("SourceCode.String() = %q, expected %q", got, expected)
	}
}

func TestSourceCode_Equals(t *testing.T) {
	source1 := NewSourceCode("hello", "test.eo")
	source2 := NewSourceCode("hello", "test.eo")
	source3 := NewSourceCode("world", "test.eo")
	source4 := NewSourceCode("hello", "other.eo")

	tests := []struct {
		name     string
		source1  SourceCode
		source2  SourceCode
		expected bool
	}{
		{"identical sources", source1, source2, true},
		{"different content", source1, source3, false},
		{"different file path", source1, source4, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.source1.Equals(tt.source2); got != tt.expected {
				t.Errorf("SourceCode.Equals() = %v, expected %v", got, tt.expected)
			}
		})
	}
}
