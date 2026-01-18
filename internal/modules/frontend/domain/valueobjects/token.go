package valueobjects

import (
	"fmt"
)

// Token represents a lexical token in the source code.
// This is a value object - immutable and identified by its properties.
type Token struct {
	// Token type
	tokenType TokenType

	// Token content
	lexeme string

	// Position information
	line   int
	column int
	offset int

	// Additional metadata
	literal interface{} // For literals like numbers, strings
}

// NewToken creates a new token
func NewToken(tokenType TokenType, lexeme string, line, column, offset int) Token {
	return Token{
		tokenType: tokenType,
		lexeme:    lexeme,
		line:      line,
		column:    column,
		offset:    offset,
	}
}

// NewLiteralToken creates a token with a literal value
func NewLiteralToken(tokenType TokenType, lexeme string, literal interface{}, line, column, offset int) Token {
	token := NewToken(tokenType, lexeme, line, column, offset)
	token.literal = literal
	return token
}

// TokenType returns the token type
func (t Token) TokenType() TokenType {
	return t.tokenType
}

// Lexeme returns the token lexeme
func (t Token) Lexeme() string {
	return t.lexeme
}

// Line returns the line number (1-based)
func (t Token) Line() int {
	return t.line
}

// Column returns the column number (1-based)
func (t Token) Column() int {
	return t.column
}

// Offset returns the byte offset in the source
func (t Token) Offset() int {
	return t.offset
}

// Literal returns the literal value if any
func (t Token) Literal() interface{} {
	return t.literal
}

// IsLiteral returns true if this token has a literal value
func (t Token) IsLiteral() bool {
	return t.literal != nil
}

// String returns a string representation of the token
func (t Token) String() string {
	if t.IsLiteral() {
		return fmt.Sprintf("%s(%s, %v)", t.tokenType, t.lexeme, t.literal)
	}
	return fmt.Sprintf("%s(%s)", t.tokenType, t.lexeme)
}

// Equals checks if two tokens are equal
func (t Token) Equals(other Token) bool {
	return t.tokenType == other.tokenType &&
		t.lexeme == other.lexeme &&
		t.line == other.line &&
		t.column == other.column &&
		t.offset == other.offset
}

// TokenType represents the type of a token
type TokenType string

// Predefined token types
const (
	// Keywords
	TokenTypeFunc    TokenType = "func"
	TokenTypeVar     TokenType = "var"
	TokenTypeIf      TokenType = "if"
	TokenTypeElse    TokenType = "else"
	TokenTypeFor     TokenType = "for"
	TokenTypeReturn  TokenType = "return"
	TokenTypeImport  TokenType = "import"
	TokenTypePackage TokenType = "package"

	// Literals
	TokenTypeIdentifier TokenType = "identifier"
	TokenTypeString     TokenType = "string"
	TokenTypeInt        TokenType = "int"
	TokenTypeFloat      TokenType = "float"
	TokenTypeBool       TokenType = "bool"

	// Operators
	TokenTypePlus     TokenType = "plus"
	TokenTypeMinus    TokenType = "minus"
	TokenTypeMultiply TokenType = "multiply"
	TokenTypeDivide   TokenType = "divide"
	TokenTypeAssign   TokenType = "assign"
	TokenTypeEqual    TokenType = "equal"
	TokenTypeNotEqual TokenType = "not_equal"
	TokenTypeLess     TokenType = "less"
	TokenTypeGreater  TokenType = "greater"
	TokenTypeAnd      TokenType = "and"
	TokenTypeOr       TokenType = "or"
	TokenTypeNot      TokenType = "not"

	// Punctuation
	TokenTypeLeftParen    TokenType = "left_paren"
	TokenTypeRightParen   TokenType = "right_paren"
	TokenTypeLeftBrace    TokenType = "left_brace"
	TokenTypeRightBrace   TokenType = "right_brace"
	TokenTypeLeftBracket  TokenType = "left_bracket"
	TokenTypeRightBracket TokenType = "right_bracket"
	TokenTypeComma        TokenType = "comma"
	TokenTypeDot          TokenType = "dot"
	TokenTypeSemicolon    TokenType = "semicolon"
	TokenTypeColon        TokenType = "colon"

	// Special tokens
	TokenTypeEOF     TokenType = "eof"
	TokenTypeIllegal TokenType = "illegal"
)

// IsKeyword checks if the token type is a keyword
func (tt TokenType) IsKeyword() bool {
	switch tt {
	case TokenTypeFunc, TokenTypeVar, TokenTypeIf, TokenTypeElse,
		TokenTypeFor, TokenTypeReturn, TokenTypeImport, TokenTypePackage:
		return true
	default:
		return false
	}
}

// IsLiteral checks if the token type represents a literal value
func (tt TokenType) IsLiteral() bool {
	switch tt {
	case TokenTypeString, TokenTypeInt, TokenTypeFloat, TokenTypeBool:
		return true
	default:
		return false
	}
}

// IsOperator checks if the token type is an operator
func (tt TokenType) IsOperator() bool {
	switch tt {
	case TokenTypePlus, TokenTypeMinus, TokenTypeMultiply, TokenTypeDivide,
		TokenTypeAssign, TokenTypeEqual, TokenTypeNotEqual, TokenTypeLess,
		TokenTypeGreater, TokenTypeAnd, TokenTypeOr, TokenTypeNot:
		return true
	default:
		return false
	}
}

// IsPunctuation checks if the token type is punctuation
func (tt TokenType) IsPunctuation() bool {
	switch tt {
	case TokenTypeLeftParen, TokenTypeRightParen, TokenTypeLeftBrace,
		TokenTypeRightBrace, TokenTypeLeftBracket, TokenTypeRightBracket,
		TokenTypeComma, TokenTypeDot, TokenTypeSemicolon, TokenTypeColon:
		return true
	default:
		return false
	}
}
