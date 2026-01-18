package analysis

import "fmt"

// TokenKind 表示词法单元的类型
type TokenKind int

const (
	// 关键字
	Keyword TokenKind = iota
	Function
	If
	Else
	For
	While
	Return
	Struct
	Enum
	Trait
	Impl
	Print
	Let
	Match
	Async
	Await
	Chan
	Spawn
	Future

	// 标识符和字面量
	Identifier
	StringLiteral
	IntLiteral
	BoolLiteral
	FloatLiteral

	// 操作符
	Plus         // +
	Minus        // -
	Multiply     // *
	Divide       // /
	Assign       // =
	Equal        // ==
	NotEqual     // !=
	Less         // <
	Greater      // >
	LessEqual    // <=
	GreaterEqual // >=
	Not          // !
	And          // &&
	Or           // ||
	Dot          // .
	Comma        // ,
	Semicolon    // ;
	Colon        // :
	LeftParen    // (
	RightParen   // )
	LeftBrace    // {
	RightBrace   // }
	LeftBracket  // [
	RightBracket // ]
	FatArrow      // =>

	// 特殊标记
	EOF
)

// String 返回TokenKind的字符串表示
func (k TokenKind) String() string {
	switch k {
	case Keyword:
		return "keyword"
	case Function:
		return "function"
	case If:
		return "if"
	case Else:
		return "else"
	case For:
		return "for"
	case While:
		return "while"
	case Return:
		return "return"
	case Struct:
		return "struct"
	case Enum:
		return "enum"
	case Trait:
		return "trait"
	case Impl:
		return "impl"
	case Print:
		return "print"
	case Let:
		return "let"
	case Match:
		return "match"
	case Async:
		return "async"
	case Await:
		return "await"
	case Chan:
		return "chan"
	case Spawn:
		return "spawn"
	case Future:
		return "future"
	case Identifier:
		return "identifier"
	case StringLiteral:
		return "string_literal"
	case IntLiteral:
		return "int_literal"
	case BoolLiteral:
		return "bool_literal"
	case FloatLiteral:
		return "float_literal"
	case Plus:
		return "plus"
	case Minus:
		return "minus"
	case Multiply:
		return "multiply"
	case Divide:
		return "divide"
	case Assign:
		return "assign"
	case Equal:
		return "equal"
	case NotEqual:
		return "not_equal"
	case Less:
		return "less"
	case Greater:
		return "greater"
	case LessEqual:
		return "less_equal"
	case GreaterEqual:
		return "greater_equal"
	case Not:
		return "not"
	case And:
		return "and"
	case Or:
		return "or"
	case Dot:
		return "dot"
	case Comma:
		return "comma"
	case Semicolon:
		return "semicolon"
	case Colon:
		return "colon"
	case LeftParen:
		return "left_paren"
	case RightParen:
		return "right_paren"
	case LeftBrace:
		return "left_brace"
	case RightBrace:
		return "right_brace"
	case LeftBracket:
		return "left_bracket"
	case RightBracket:
		return "right_bracket"
	case FatArrow:
		return "fat_arrow"
	case EOF:
		return "eof"
	default:
		return "unknown"
	}
}

// Token 表示词法单元，是不可变的值对象
type Token struct {
	kind     TokenKind // 词法类型
	value    string    // 词法值
	position Position  // 位置信息
}

// NewToken 创建新的Token
func NewToken(kind TokenKind, value string, position Position) Token {
	return Token{
		kind:     kind,
		value:    value,
		position: position,
	}
}

// Kind 返回Token的类型
func (t Token) Kind() TokenKind {
	return t.kind
}

// Value 返回Token的值
func (t Token) Value() string {
	return t.value
}

// Position 返回Token的位置
func (t Token) Position() Position {
	return t.position
}

// IsKeyword 检查是否为关键字
func (t Token) IsKeyword() bool {
	return t.kind >= Keyword && t.kind <= Match
}

// IsIdentifier 检查是否为标识符
func (t Token) IsIdentifier() bool {
	return t.kind == Identifier
}

// IsLiteral 检查是否为字面量
func (t Token) IsLiteral() bool {
	return t.kind >= StringLiteral && t.kind <= FloatLiteral
}

// IsOperator 检查是否为操作符
func (t Token) IsOperator() bool {
	return t.kind >= Plus && t.kind <= RightBracket
}

// IsEOF 检查是否为文件结束标记
func (t Token) IsEOF() bool {
	return t.kind == EOF
}

// String 返回Token的字符串表示
func (t Token) String() string {
	return fmt.Sprintf("Token{kind=%s, value='%s', position=%s}",
		t.kind.String(), t.value, t.position.String())
}

// Equals 检查两个Token是否相等（值对象相等性）
func (t Token) Equals(other Token) bool {
	return t.kind == other.kind &&
		t.value == other.value &&
		t.position.Equals(other.position)
}
