package analysis

import (
	"testing"
)

func TestNewToken(t *testing.T) {
	pos := NewPosition(1, 5, "test.eo")
	token := NewToken(Identifier, "variable", pos)

	if token.Kind() != Identifier {
		t.Errorf("Expected kind Identifier, got %v", token.Kind())
	}

	if token.Value() != "variable" {
		t.Errorf("Expected value 'variable', got '%s'", token.Value())
	}

	if !token.Position().Equals(pos) {
		t.Errorf("Expected position %v, got %v", pos, token.Position())
	}
}

func TestToken_IsKeyword(t *testing.T) {
	tests := []struct {
		name     string
		token    Token
		expected bool
	}{
		{"function keyword", NewToken(Function, "func", NewPosition(1, 1, "test.eo")), true},
		{"if keyword", NewToken(If, "if", NewPosition(1, 1, "test.eo")), true},
		{"identifier", NewToken(Identifier, "variable", NewPosition(1, 1, "test.eo")), false},
		{"string literal", NewToken(StringLiteral, "hello", NewPosition(1, 1, "test.eo")), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.IsKeyword(); got != tt.expected {
				t.Errorf("Token.IsKeyword() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestToken_IsIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		token    Token
		expected bool
	}{
		{"identifier", NewToken(Identifier, "variable", NewPosition(1, 1, "test.eo")), true},
		{"function keyword", NewToken(Function, "func", NewPosition(1, 1, "test.eo")), false},
		{"string literal", NewToken(StringLiteral, "hello", NewPosition(1, 1, "test.eo")), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.IsIdentifier(); got != tt.expected {
				t.Errorf("Token.IsIdentifier() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestToken_IsLiteral(t *testing.T) {
	tests := []struct {
		name     string
		token    Token
		expected bool
	}{
		{"string literal", NewToken(StringLiteral, "hello", NewPosition(1, 1, "test.eo")), true},
		{"int literal", NewToken(IntLiteral, "42", NewPosition(1, 1, "test.eo")), true},
		{"bool literal", NewToken(BoolLiteral, "true", NewPosition(1, 1, "test.eo")), true},
		{"float literal", NewToken(FloatLiteral, "3.14", NewPosition(1, 1, "test.eo")), true},
		{"identifier", NewToken(Identifier, "variable", NewPosition(1, 1, "test.eo")), false},
		{"function keyword", NewToken(Function, "func", NewPosition(1, 1, "test.eo")), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.IsLiteral(); got != tt.expected {
				t.Errorf("Token.IsLiteral() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestToken_IsEOF(t *testing.T) {
	tests := []struct {
		name     string
		token    Token
		expected bool
	}{
		{"EOF token", NewToken(EOF, "", NewPosition(1, 1, "test.eo")), true},
		{"identifier", NewToken(Identifier, "variable", NewPosition(1, 1, "test.eo")), false},
		{"function keyword", NewToken(Function, "func", NewPosition(1, 1, "test.eo")), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.IsEOF(); got != tt.expected {
				t.Errorf("Token.IsEOF() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestToken_String(t *testing.T) {
	token := NewToken(Identifier, "variable", NewPosition(1, 5, "test.eo"))
	expected := "Token{kind=identifier, value='variable', position=test.eo:1:5}"

	if got := token.String(); got != expected {
		t.Errorf("Token.String() = %q, expected %q", got, expected)
	}
}

func TestToken_Equals(t *testing.T) {
	pos1 := NewPosition(1, 5, "test.eo")
	pos2 := NewPosition(1, 5, "test.eo")
	pos3 := NewPosition(2, 5, "test.eo")

	token1 := NewToken(Identifier, "variable", pos1)
	token2 := NewToken(Identifier, "variable", pos2)
	token3 := NewToken(Function, "variable", pos1)
	token4 := NewToken(Identifier, "other", pos1)
	token5 := NewToken(Identifier, "variable", pos3)

	tests := []struct {
		name     string
		token1   Token
		token2   Token
		expected bool
	}{
		{"identical tokens", token1, token2, true},
		{"different kind", token1, token3, false},
		{"different value", token1, token4, false},
		{"different position", token1, token5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token1.Equals(tt.token2); got != tt.expected {
				t.Errorf("Token.Equals() = %v, expected %v", got, tt.expected)
			}
		})
	}
}
