package analysis

import (
	"reflect"
	"testing"
)

func TestNewSimpleTokenizer(t *testing.T) {
	tokenizer := NewSimpleTokenizer()
	if tokenizer == nil {
		t.Error("NewSimpleTokenizer() returned nil")
	}

	// Test that it implements the Tokenizer interface
	var _ Tokenizer = tokenizer
}

func TestSimpleTokenizer_Tokenize(t *testing.T) {
	tokenizer := NewSimpleTokenizer()

	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "empty source",
			input: "",
			expected: []Token{
				{kind: EOF, value: "", position: Position{line: 1, column: 1, file: "test.eo"}},
			},
		},
		{
			name:  "simple identifier",
			input: "variable",
			expected: []Token{
				{kind: Identifier, value: "variable", position: Position{line: 1, column: 1, file: "test.eo"}},
				{kind: EOF, value: "", position: Position{line: 1, column: 9, file: "test.eo"}},
			},
		},
		{
			name:  "function keyword",
			input: "func",
			expected: []Token{
				{kind: Function, value: "func", position: Position{line: 1, column: 1, file: "test.eo"}},
				{kind: EOF, value: "", position: Position{line: 1, column: 5, file: "test.eo"}},
			},
		},
		{
			name:  "string literal",
			input: `"hello world"`,
			expected: []Token{
				{kind: StringLiteral, value: "hello world", position: Position{line: 1, column: 1, file: "test.eo"}},
				{kind: EOF, value: "", position: Position{line: 1, column: 14, file: "test.eo"}},
			},
		},
		{
			name:  "integer literal",
			input: "42",
			expected: []Token{
				{kind: IntLiteral, value: "42", position: Position{line: 1, column: 1, file: "test.eo"}},
				{kind: EOF, value: "", position: Position{line: 1, column: 3, file: "test.eo"}},
			},
		},
		{
			name:  "float literal",
			input: "3.14",
			expected: []Token{
				{kind: FloatLiteral, value: "3.14", position: Position{line: 1, column: 1, file: "test.eo"}},
				{kind: EOF, value: "", position: Position{line: 1, column: 5, file: "test.eo"}},
			},
		},
		{
			name:  "boolean literal",
			input: "true",
			expected: []Token{
				{kind: BoolLiteral, value: "true", position: Position{line: 1, column: 1, file: "test.eo"}},
				{kind: EOF, value: "", position: Position{line: 1, column: 5, file: "test.eo"}},
			},
		},
		{
			name:  "operators",
			input: "+ - * / = == != < > <= >=",
			expected: []Token{
				{kind: Plus, value: "+", position: Position{line: 1, column: 1, file: "test.eo"}},
				{kind: Minus, value: "-", position: Position{line: 1, column: 3, file: "test.eo"}},
				{kind: Multiply, value: "*", position: Position{line: 1, column: 5, file: "test.eo"}},
				{kind: Divide, value: "/", position: Position{line: 1, column: 7, file: "test.eo"}},
				{kind: Assign, value: "=", position: Position{line: 1, column: 9, file: "test.eo"}},
				{kind: Equal, value: "==", position: Position{line: 1, column: 11, file: "test.eo"}},
				{kind: NotEqual, value: "!=", position: Position{line: 1, column: 14, file: "test.eo"}},
				{kind: Less, value: "<", position: Position{line: 1, column: 17, file: "test.eo"}},
				{kind: Greater, value: ">", position: Position{line: 1, column: 19, file: "test.eo"}},
				{kind: LessEqual, value: "<=", position: Position{line: 1, column: 21, file: "test.eo"}},
				{kind: GreaterEqual, value: ">=", position: Position{line: 1, column: 24, file: "test.eo"}},
				{kind: EOF, value: "", position: Position{line: 1, column: 26, file: "test.eo"}},
			},
		},
		{
			name:  "punctuation",
			input: "( ) { } [ ] , ; : .",
			expected: []Token{
				{kind: LeftParen, value: "(", position: Position{line: 1, column: 1, file: "test.eo"}},
				{kind: RightParen, value: ")", position: Position{line: 1, column: 3, file: "test.eo"}},
				{kind: LeftBrace, value: "{", position: Position{line: 1, column: 5, file: "test.eo"}},
				{kind: RightBrace, value: "}", position: Position{line: 1, column: 7, file: "test.eo"}},
				{kind: LeftBracket, value: "[", position: Position{line: 1, column: 9, file: "test.eo"}},
				{kind: RightBracket, value: "]", position: Position{line: 1, column: 11, file: "test.eo"}},
				{kind: Comma, value: ",", position: Position{line: 1, column: 13, file: "test.eo"}},
				{kind: Semicolon, value: ";", position: Position{line: 1, column: 15, file: "test.eo"}},
				{kind: Colon, value: ":", position: Position{line: 1, column: 17, file: "test.eo"}},
				{kind: Dot, value: ".", position: Position{line: 1, column: 19, file: "test.eo"}},
				{kind: EOF, value: "", position: Position{line: 1, column: 20, file: "test.eo"}},
			},
		},
		{
			name:  "multiple lines",
			input: "func add\nreturn a",
			expected: []Token{
				{kind: Function, value: "func", position: Position{line: 1, column: 1, file: "test.eo"}},
				{kind: Identifier, value: "add", position: Position{line: 1, column: 6, file: "test.eo"}},
				{kind: Return, value: "return", position: Position{line: 2, column: 1, file: "test.eo"}},
				{kind: Identifier, value: "a", position: Position{line: 2, column: 8, file: "test.eo"}},
				{kind: EOF, value: "", position: Position{line: 2, column: 9, file: "test.eo"}},
			},
		},
		{
			name:  "comments are ignored",
			input: "func add // this is a comment\nreturn a",
			expected: []Token{
				{kind: Function, value: "func", position: Position{line: 1, column: 1, file: "test.eo"}},
				{kind: Identifier, value: "add", position: Position{line: 1, column: 6, file: "test.eo"}},
				{kind: Return, value: "return", position: Position{line: 2, column: 1, file: "test.eo"}},
				{kind: Identifier, value: "a", position: Position{line: 2, column: 8, file: "test.eo"}},
				{kind: EOF, value: "", position: Position{line: 2, column: 9, file: "test.eo"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := NewSourceCode(tt.input, "test.eo")
			tokens, err := tokenizer.Tokenize(source)

			if err != nil {
				t.Fatalf("Tokenize failed: %v", err)
			}

			if len(tokens) != len(tt.expected) {
				t.Fatalf("Expected %d tokens, got %d", len(tt.expected), len(tokens))
			}

			for i, expected := range tt.expected {
				actual := tokens[i]
				if !reflect.DeepEqual(actual, expected) {
					t.Errorf("Token %d: expected %+v, got %+v", i, expected, actual)
				}
			}
		})
	}
}

func TestSimpleTokenizer_TokenizeErrors(t *testing.T) {
	tokenizer := NewSimpleTokenizer()

	tests := []struct {
		name     string
		input    string
		expected string // error message substring
	}{
		{
			name:     "unclosed string",
			input:    `"hello world`,
			expected: "unexpected character",
		},
		{
			name:     "invalid character",
			input:    "@invalid",
			expected: "unexpected character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := NewSourceCode(tt.input, "test.eo")
			_, err := tokenizer.Tokenize(source)

			if err == nil {
				t.Errorf("Expected error containing %q, but got no error", tt.expected)
			} else if !containsString(err.Error(), tt.expected) {
				t.Errorf("Expected error containing %q, got %q", tt.expected, err.Error())
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsString(s[1:], substr) || (len(s) > 0 && s[:len(substr)] == substr))
}
