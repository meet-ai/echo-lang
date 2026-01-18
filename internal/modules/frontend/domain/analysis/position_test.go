package analysis

import (
	"testing"
)

func TestNewPosition(t *testing.T) {
	pos := NewPosition(10, 5, "test.eo")

	if pos.Line() != 10 {
		t.Errorf("Expected line 10, got %d", pos.Line())
	}

	if pos.Column() != 5 {
		t.Errorf("Expected column 5, got %d", pos.Column())
	}

	if pos.File() != "test.eo" {
		t.Errorf("Expected file 'test.eo', got '%s'", pos.File())
	}
}

func TestPosition_String(t *testing.T) {
	pos := NewPosition(10, 5, "test.eo")
	expected := "test.eo:10:5"

	if got := pos.String(); got != expected {
		t.Errorf("Position.String() = %q, expected %q", got, expected)
	}
}

func TestPosition_IsBefore(t *testing.T) {
	pos1 := NewPosition(1, 5, "test.eo")
	pos2 := NewPosition(1, 10, "test.eo")
	pos3 := NewPosition(2, 5, "test.eo")
	pos4 := NewPosition(1, 5, "other.eo")
	pos5 := NewPosition(1, 5, "test.eo")

	tests := []struct {
		name     string
		pos1     Position
		pos2     Position
		expected bool
	}{
		{"same line, earlier column", pos1, pos2, true},
		{"same line, later column", pos2, pos1, false},
		{"earlier line", pos1, pos3, true},
		{"later line", pos3, pos1, false},
		{"different file, lexicographically smaller", pos4, pos1, true},
		{"different file, lexicographically larger", pos1, pos4, false},
		{"identical positions", pos1, pos5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pos1.IsBefore(tt.pos2); got != tt.expected {
				t.Errorf("Position.IsBefore() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestPosition_IsAfter(t *testing.T) {
	pos1 := NewPosition(1, 5, "test.eo")
	pos2 := NewPosition(1, 10, "test.eo")
	pos3 := NewPosition(2, 5, "test.eo")

	tests := []struct {
		name     string
		pos1     Position
		pos2     Position
		expected bool
	}{
		{"same line, later column", pos2, pos1, true},
		{"same line, earlier column", pos1, pos2, false},
		{"later line", pos3, pos1, true},
		{"earlier line", pos1, pos3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pos1.IsAfter(tt.pos2); got != tt.expected {
				t.Errorf("Position.IsAfter() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestPosition_Equals(t *testing.T) {
	pos1 := NewPosition(1, 5, "test.eo")
	pos2 := NewPosition(1, 5, "test.eo")
	pos3 := NewPosition(2, 5, "test.eo")
	pos4 := NewPosition(1, 10, "test.eo")
	pos5 := NewPosition(1, 5, "other.eo")

	tests := []struct {
		name     string
		pos1     Position
		pos2     Position
		expected bool
	}{
		{"identical positions", pos1, pos2, true},
		{"different line", pos1, pos3, false},
		{"different column", pos1, pos4, false},
		{"different file", pos1, pos5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pos1.Equals(tt.pos2); got != tt.expected {
				t.Errorf("Position.Equals() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestPosition_NextColumn(t *testing.T) {
	pos := NewPosition(1, 5, "test.eo")
	next := pos.NextColumn()

	expected := NewPosition(1, 6, "test.eo")
	if !next.Equals(expected) {
		t.Errorf("Position.NextColumn() = %v, expected %v", next, expected)
	}
}

func TestPosition_NextLine(t *testing.T) {
	pos := NewPosition(1, 5, "test.eo")
	next := pos.NextLine()

	expected := NewPosition(2, 1, "test.eo")
	if !next.Equals(expected) {
		t.Errorf("Position.NextLine() = %v, expected %v", next, expected)
	}
}
