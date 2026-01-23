// Package value_objects 定义Token相关值对象
package value_objects

import "fmt"

// TokenType 词法单元类型
type TokenType string

const (
	// 关键字
	TokenTypeKeyword TokenType = "keyword"
	TokenTypeFunc    TokenType = "func"
	TokenTypeLet     TokenType = "let"
	TokenTypeIf      TokenType = "if"
	TokenTypeElse    TokenType = "else"
	TokenTypeWhile   TokenType = "while"
	TokenTypeFor     TokenType = "for"
	TokenTypeReturn  TokenType = "return"
	TokenTypeStruct  TokenType = "struct"
	TokenTypeEnum    TokenType = "enum"
	TokenTypeTrait   TokenType = "trait"
	TokenTypeImpl    TokenType = "impl"
	TokenTypeMatch   TokenType = "match"
	TokenTypeAsync   TokenType = "async"
	TokenTypeSpawn   TokenType = "spawn"
	TokenTypeChan    TokenType = "chan"
	TokenTypePrint   TokenType = "print"

	// 标识符和字面量
	TokenTypeIdentifier TokenType = "identifier"
	TokenTypeString     TokenType = "string"
	TokenTypeInt        TokenType = "int"
	TokenTypeFloat      TokenType = "float"
	TokenTypeBool       TokenType = "bool"

	// 运算符
	TokenTypePlus          TokenType = "plus"
	TokenTypeMinus         TokenType = "minus"
	TokenTypeMultiply      TokenType = "multiply"
	TokenTypeDivide        TokenType = "divide"
	TokenTypeModulo        TokenType = "modulo"
	TokenTypeAssign        TokenType = "assign"
	TokenTypeEqual         TokenType = "equal"
	TokenTypeNotEqual      TokenType = "not_equal"
	TokenTypeLessThan      TokenType = "less_than"
	TokenTypeGreaterThan   TokenType = "greater_than"
	TokenTypeLessEqual     TokenType = "less_equal"
	TokenTypeGreaterEqual  TokenType = "greater_equal"
	TokenTypeAnd           TokenType = "and"
	TokenTypeOr            TokenType = "or"
	TokenTypeNot           TokenType = "not"

	// 分隔符
	TokenTypeLeftParen     TokenType = "left_paren"
	TokenTypeRightParen    TokenType = "right_paren"
	TokenTypeLeftBrace     TokenType = "left_brace"
	TokenTypeRightBrace    TokenType = "right_brace"
	TokenTypeLeftBracket   TokenType = "left_bracket"
	TokenTypeRightBracket  TokenType = "right_bracket"
	TokenTypeComma         TokenType = "comma"
	TokenTypeDot           TokenType = "dot"
	TokenTypeColon         TokenType = "colon"
	TokenTypeSemicolon     TokenType = "semicolon"
	TokenTypeArrow         TokenType = "arrow"

	// 特殊标记
	TokenTypeEOF           TokenType = "eof"
	TokenTypeError         TokenType = "error"
)

// Token 词法单元值对象
type Token struct {
	tokenType TokenType
	lexeme    string
	value     interface{}
	location  SourceLocation
}

// NewToken 创建新的Token
func NewToken(tokenType TokenType, lexeme string, value interface{}, location SourceLocation) *Token {
	return &Token{
		tokenType: tokenType,
		lexeme:    lexeme,
		value:     value,
		location:  location,
	}
}

// Type 获取Token类型
func (t *Token) Type() TokenType {
	return t.tokenType
}

// Lexeme 获取词素
func (t *Token) Lexeme() string {
	return t.lexeme
}

// Value 获取值
func (t *Token) Value() interface{} {
	return t.value
}

// Location 获取位置信息
func (t *Token) Location() SourceLocation {
	return t.location
}

// String 返回Token的字符串表示
func (t *Token) String() string {
	if t.value != nil {
		return fmt.Sprintf("%s(%s)", t.tokenType, t.lexeme)
	}
	return fmt.Sprintf("%s", t.tokenType)
}

// TokenStream Token流值对象
type TokenStream struct {
	tokens    []*Token
	sourceFile *SourceFile
	position  int
}

// NewTokenStream 创建新的Token流
func NewTokenStream(sourceFile *SourceFile) *TokenStream {
	return &TokenStream{
		tokens:     make([]*Token, 0),
		sourceFile: sourceFile,
		position:   0,
	}
}

// AddToken 添加Token到流中
func (ts *TokenStream) AddToken(token *Token) {
	ts.tokens = append(ts.tokens, token)
}

// Tokens 获取所有Token
func (ts *TokenStream) Tokens() []*Token {
	return ts.tokens
}

// Count 获取Token数量
func (ts *TokenStream) Count() int {
	return len(ts.tokens)
}

// Current 获取当前位置的Token
func (ts *TokenStream) Current() *Token {
	if ts.position >= len(ts.tokens) {
		return nil
	}
	return ts.tokens[ts.position]
}

// Next 移动到下一个Token
func (ts *TokenStream) Next() *Token {
	if ts.position >= len(ts.tokens) {
		return nil
	}
	token := ts.tokens[ts.position]
	ts.position++
	return token
}

// Peek 查看下一个Token但不移动位置
func (ts *TokenStream) Peek() *Token {
	if ts.position >= len(ts.tokens) {
		return nil
	}
	return ts.tokens[ts.position]
}

// Reset 重置位置到开头
func (ts *TokenStream) Reset() {
	ts.position = 0
}

// IsAtEnd 检查是否到达流末尾
func (ts *TokenStream) IsAtEnd() bool {
	return ts.position >= len(ts.tokens)
}
