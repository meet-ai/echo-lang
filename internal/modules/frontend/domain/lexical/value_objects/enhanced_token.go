// Package value_objects 定义词法分析上下文的值对象
package value_objects

import (
	"fmt"

	"echo/internal/modules/frontend/domain/shared/value_objects"
)

// EnhancedTokenType 增强Token类型枚举
type EnhancedTokenType string

const (
	// 关键字
	EnhancedTokenTypeKeyword EnhancedTokenType = "keyword"
	EnhancedTokenTypeFunc    EnhancedTokenType = "func"
	EnhancedTokenTypeLet     EnhancedTokenType = "let"
	EnhancedTokenTypeIf      EnhancedTokenType = "if"
	EnhancedTokenTypeElse    EnhancedTokenType = "else"
	EnhancedTokenTypeWhile   EnhancedTokenType = "while"
	EnhancedTokenTypeFor     EnhancedTokenType = "for"
	EnhancedTokenTypeReturn  EnhancedTokenType = "return"
	EnhancedTokenTypeStruct  EnhancedTokenType = "struct"
	EnhancedTokenTypeEnum    EnhancedTokenType = "enum"
	EnhancedTokenTypeTrait   EnhancedTokenType = "trait"
	EnhancedTokenTypeImpl    EnhancedTokenType = "impl"
	EnhancedTokenTypeMatch   EnhancedTokenType = "match"
	EnhancedTokenTypeAsync   EnhancedTokenType = "async"
	EnhancedTokenTypeSpawn   EnhancedTokenType = "spawn"
	EnhancedTokenTypeChan    EnhancedTokenType = "chan"
	EnhancedTokenTypePrint   EnhancedTokenType = "print"
	EnhancedTokenTypeSelect  EnhancedTokenType = "select"
	EnhancedTokenTypeCase    EnhancedTokenType = "case"
	EnhancedTokenTypePackage EnhancedTokenType = "package"
	EnhancedTokenTypePrivate EnhancedTokenType = "private"
	EnhancedTokenTypeFrom    EnhancedTokenType = "from"

	// 标识符和字面量
	EnhancedTokenTypeIdentifier EnhancedTokenType = "identifier"
	EnhancedTokenTypeString     EnhancedTokenType = "string"
	EnhancedTokenTypeNumber     EnhancedTokenType = "number" // 统一的数字类型
	EnhancedTokenTypeBool       EnhancedTokenType = "bool"

	// 运算符
	EnhancedTokenTypePlus          EnhancedTokenType = "plus"
	EnhancedTokenTypeMinus         EnhancedTokenType = "minus"
	EnhancedTokenTypeMultiply      EnhancedTokenType = "multiply"
	EnhancedTokenTypeDivide        EnhancedTokenType = "divide"
	EnhancedTokenTypeModulo        EnhancedTokenType = "modulo"
	EnhancedTokenTypeAssign        EnhancedTokenType = "assign"
	EnhancedTokenTypeDeclare       EnhancedTokenType = "declare" // :=
	EnhancedTokenTypeEqual         EnhancedTokenType = "equal"
	EnhancedTokenTypeNotEqual      EnhancedTokenType = "not_equal"
	EnhancedTokenTypeLessThan      EnhancedTokenType = "less_than"
	EnhancedTokenTypeGreaterThan   EnhancedTokenType = "greater_than"
	EnhancedTokenTypeLessEqual     EnhancedTokenType = "less_equal"
	EnhancedTokenTypeGreaterEqual  EnhancedTokenType = "greater_equal"
	EnhancedTokenTypeAnd           EnhancedTokenType = "and"
	EnhancedTokenTypeOr            EnhancedTokenType = "or"
	EnhancedTokenTypeNot           EnhancedTokenType = "not"
	EnhancedTokenTypeChannelReceive EnhancedTokenType = "channel_receive" // <- 通道接收运算符
	EnhancedTokenTypeChannelSend    EnhancedTokenType = "channel_send"    // -> 通道发送运算符（如果与 arrow 不同）

	// 分隔符
	EnhancedTokenTypeLeftParen     EnhancedTokenType = "left_paren"
	EnhancedTokenTypeRightParen    EnhancedTokenType = "right_paren"
	EnhancedTokenTypeLeftBrace     EnhancedTokenType = "left_brace"
	EnhancedTokenTypeRightBrace    EnhancedTokenType = "right_brace"
	EnhancedTokenTypeLeftBracket   EnhancedTokenType = "left_bracket"
	EnhancedTokenTypeRightBracket  EnhancedTokenType = "right_bracket"
	EnhancedTokenTypeComma         EnhancedTokenType = "comma"
	EnhancedTokenTypeDot           EnhancedTokenType = "dot"
	EnhancedTokenTypeColon         EnhancedTokenType = "colon"
	EnhancedTokenTypeSemicolon     EnhancedTokenType = "semicolon"
	EnhancedTokenTypeArrow         EnhancedTokenType = "arrow"         // ->
	EnhancedTokenTypeFatArrow      EnhancedTokenType = "fat_arrow"     // =>

	// 特殊标记
	EnhancedTokenTypeEOF           EnhancedTokenType = "eof"
	EnhancedTokenTypeError         EnhancedTokenType = "error"
)

// EnhancedToken 增强Token值对象
// 支持现代化的数字字面量处理和更丰富的元数据
type EnhancedToken struct {
	// 基础信息
	tokenType EnhancedTokenType
	lexeme    string

	// 增强的值存储
	numberLiteral *NumberLiteral      // 数字字面量（当tokenType为number时）
	stringValue   string             // 字符串值（当tokenType为string时）
	boolValue     *bool              // 布尔值（当tokenType为bool时）
	identifier    string             // 标识符名（当tokenType为identifier时）

	// 元数据
	isKeyword     bool               // 是否为关键字
	category      TokenCategory      // Token分类

	// 位置信息
	location      value_objects.SourceLocation
}

// TokenCategory Token分类枚举
type TokenCategory int

const (
	CategoryKeyword    TokenCategory = iota // 关键字
	CategoryIdentifier                      // 标识符
	CategoryLiteral                         // 字面量
	CategoryOperator                        // 运算符
	CategoryDelimiter                       // 分隔符
	CategorySpecial                         // 特殊标记
)

// NewEnhancedToken 创建新的增强Token
func NewEnhancedToken(tokenType EnhancedTokenType, lexeme string, location value_objects.SourceLocation) *EnhancedToken {
	et := &EnhancedToken{
		tokenType: tokenType,
		lexeme:    lexeme,
		location:  location,
		category:  determineCategory(tokenType),
	}

	// 根据类型设置额外属性
	et.isKeyword = isKeywordType(tokenType)

	return et
}

// NewNumberToken 创建数字Token
func NewNumberToken(numberLiteral *NumberLiteral) *EnhancedToken {
	return &EnhancedToken{
		tokenType:     EnhancedTokenTypeNumber,
		lexeme:        numberLiteral.OriginalLexeme(),
		numberLiteral: numberLiteral,
		category:      CategoryLiteral,
		location:      numberLiteral.Location(),
		isKeyword:     false,
	}
}

// NewStringToken 创建字符串Token
func NewStringToken(lexeme string, value string, location value_objects.SourceLocation) *EnhancedToken {
	return &EnhancedToken{
		tokenType:   EnhancedTokenTypeString,
		lexeme:      lexeme,
		stringValue: value,
		category:    CategoryLiteral,
		location:    location,
		isKeyword:   false,
	}
}

// NewBoolToken 创建布尔Token
func NewBoolToken(lexeme string, value bool, location value_objects.SourceLocation) *EnhancedToken {
	boolVal := value
	return &EnhancedToken{
		tokenType: EnhancedTokenTypeBool,
		lexeme:    lexeme,
		boolValue: &boolVal,
		category:  CategoryLiteral,
		location:  location,
		isKeyword: false,
	}
}

// NewIdentifierToken 创建标识符Token
func NewIdentifierToken(lexeme string, location value_objects.SourceLocation) *EnhancedToken {
	return &EnhancedToken{
		tokenType:  EnhancedTokenTypeIdentifier,
		lexeme:     lexeme,
		identifier: lexeme,
		category:   CategoryIdentifier,
		location:   location,
		isKeyword:  false,
	}
}

// determineCategory 根据Token类型确定分类
func determineCategory(tokenType EnhancedTokenType) TokenCategory {
	switch tokenType {
	case EnhancedTokenTypeFunc, EnhancedTokenTypeLet, EnhancedTokenTypeIf,
		 EnhancedTokenTypeElse, EnhancedTokenTypeWhile, EnhancedTokenTypeFor,
		 EnhancedTokenTypeReturn, EnhancedTokenTypeStruct, EnhancedTokenTypeEnum,
		 EnhancedTokenTypeTrait, EnhancedTokenTypeImpl, EnhancedTokenTypeMatch,
		 EnhancedTokenTypeAsync, EnhancedTokenTypeSpawn, EnhancedTokenTypeChan,
		 EnhancedTokenTypePrint, EnhancedTokenTypePackage, EnhancedTokenTypePrivate,
		 EnhancedTokenTypeFrom, EnhancedTokenTypeSelect, EnhancedTokenTypeCase:
		return CategoryKeyword
	case EnhancedTokenTypeIdentifier:
		return CategoryIdentifier
	case EnhancedTokenTypeString, EnhancedTokenTypeNumber, EnhancedTokenTypeBool:
		return CategoryLiteral
	case EnhancedTokenTypePlus, EnhancedTokenTypeMinus, EnhancedTokenTypeMultiply,
		 EnhancedTokenTypeDivide, EnhancedTokenTypeModulo, EnhancedTokenTypeAssign,
		 EnhancedTokenTypeEqual, EnhancedTokenTypeNotEqual, EnhancedTokenTypeLessThan,
		 EnhancedTokenTypeGreaterThan, EnhancedTokenTypeLessEqual, EnhancedTokenTypeGreaterEqual,
		 EnhancedTokenTypeAnd, EnhancedTokenTypeOr, EnhancedTokenTypeNot:
		return CategoryOperator
	case EnhancedTokenTypeLeftParen, EnhancedTokenTypeRightParen, EnhancedTokenTypeLeftBrace,
		 EnhancedTokenTypeRightBrace, EnhancedTokenTypeLeftBracket, EnhancedTokenTypeRightBracket,
		 EnhancedTokenTypeComma, EnhancedTokenTypeDot, EnhancedTokenTypeColon,
		 EnhancedTokenTypeSemicolon, EnhancedTokenTypeArrow:
		return CategoryDelimiter
	case EnhancedTokenTypeEOF, EnhancedTokenTypeError:
		return CategorySpecial
	default:
		return CategorySpecial
	}
}

// isKeywordType 检查是否为关键字类型
func isKeywordType(tokenType EnhancedTokenType) bool {
	return determineCategory(tokenType) == CategoryKeyword
}

// Getters

// Type 获取Token类型
func (et *EnhancedToken) Type() EnhancedTokenType {
	return et.tokenType
}

// Lexeme 获取词素
func (et *EnhancedToken) Lexeme() string {
	return et.lexeme
}

// Category 获取Token分类
func (et *EnhancedToken) Category() TokenCategory {
	return et.category
}

// IsKeyword 检查是否为关键字
func (et *EnhancedToken) IsKeyword() bool {
	return et.isKeyword
}

// Location 获取位置信息
func (et *EnhancedToken) Location() value_objects.SourceLocation {
	return et.location
}

// NumberLiteral 获取数字字面量（仅当Type为Number时有效）
func (et *EnhancedToken) NumberLiteral() *NumberLiteral {
	if et.tokenType != EnhancedTokenTypeNumber {
		return nil
	}
	return et.numberLiteral
}

// StringValue 获取字符串值（仅当Type为String时有效）
func (et *EnhancedToken) StringValue() string {
	if et.tokenType != EnhancedTokenTypeString {
		return ""
	}
	return et.stringValue
}

// BoolValue 获取布尔值（仅当Type为Bool时有效）
func (et *EnhancedToken) BoolValue() *bool {
	if et.tokenType != EnhancedTokenTypeBool {
		return nil
	}
	return et.boolValue
}

// Identifier 获取标识符名（仅当Type为Identifier时有效）
func (et *EnhancedToken) Identifier() string {
	if et.tokenType != EnhancedTokenTypeIdentifier {
		return ""
	}
	return et.identifier
}

// Value 获取通用值接口（用于向后兼容）
func (et *EnhancedToken) Value() interface{} {
	switch et.tokenType {
	case EnhancedTokenTypeNumber:
		if et.numberLiteral != nil {
			return et.numberLiteral.Value()
		}
		return nil
	case EnhancedTokenTypeString:
		return et.stringValue
	case EnhancedTokenTypeBool:
		if et.boolValue != nil {
			return *et.boolValue
		}
		return nil
	case EnhancedTokenTypeIdentifier:
		return et.identifier
	default:
		return nil
	}
}

// Equals 比较两个EnhancedToken是否相等（值对象语义）
func (et *EnhancedToken) Equals(other *EnhancedToken) bool {
	if other == nil {
		return false
	}

	if et.tokenType != other.tokenType ||
	   et.lexeme != other.lexeme ||
	   et.category != other.category ||
	   et.isKeyword != other.isKeyword {
		return false
	}

	// 比较具体值
	switch et.tokenType {
	case EnhancedTokenTypeNumber:
		if et.numberLiteral == nil || other.numberLiteral == nil {
			return et.numberLiteral == other.numberLiteral
		}
		return et.numberLiteral.Equals(other.numberLiteral)
	case EnhancedTokenTypeString:
		return et.stringValue == other.stringValue
	case EnhancedTokenTypeBool:
		if et.boolValue == nil || other.boolValue == nil {
			return et.boolValue == other.boolValue
		}
		return *et.boolValue == *other.boolValue
	case EnhancedTokenTypeIdentifier:
		return et.identifier == other.identifier
	default:
		return true
	}
}

// String 返回Token的字符串表示
func (et *EnhancedToken) String() string {
	switch et.tokenType {
	case EnhancedTokenTypeNumber:
		if et.numberLiteral != nil {
			return fmt.Sprintf("%s(%s)", et.tokenType, et.numberLiteral.String())
		}
		return fmt.Sprintf("%s(%s)", et.tokenType, et.lexeme)
	case EnhancedTokenTypeString:
		return fmt.Sprintf("%s(%s)", et.tokenType, et.lexeme)
	case EnhancedTokenTypeBool:
		return fmt.Sprintf("%s(%s)", et.tokenType, et.lexeme)
	case EnhancedTokenTypeIdentifier:
		return fmt.Sprintf("%s(%s)", et.tokenType, et.lexeme)
	default:
		if et.Value() != nil {
			return fmt.Sprintf("%s(%s)", et.tokenType, et.lexeme)
		}
		return fmt.Sprintf("%s", et.tokenType)
	}
}

// EnhancedTokenStream 增强Token流值对象
type EnhancedTokenStream struct {
	tokens     []*EnhancedToken
	sourceFile *SourceFile
	position   int
}

// NewEnhancedTokenStream 创建新的增强Token流
func NewEnhancedTokenStream(sourceFile *SourceFile) *EnhancedTokenStream {
	return &EnhancedTokenStream{
		tokens:     make([]*EnhancedToken, 0),
		sourceFile: sourceFile,
		position:   0,
	}
}

// AddToken 添加Token到流中
func (ets *EnhancedTokenStream) AddToken(token *EnhancedToken) {
	ets.tokens = append(ets.tokens, token)
}

// Tokens 获取所有Token
func (ets *EnhancedTokenStream) Tokens() []*EnhancedToken {
	return ets.tokens
}

// Count 获取Token数量
func (ets *EnhancedTokenStream) Count() int {
	return len(ets.tokens)
}

// Current 获取当前位置的Token
func (ets *EnhancedTokenStream) Current() *EnhancedToken {
	if ets.position >= len(ets.tokens) {
		return nil
	}
	return ets.tokens[ets.position]
}

// Next 移动到下一个Token
func (ets *EnhancedTokenStream) Next() *EnhancedToken {
	if ets.position >= len(ets.tokens) {
		return nil
	}
	token := ets.tokens[ets.position]
	ets.position++
	return token
}

// Peek 查看下一个Token但不移动位置
func (ets *EnhancedTokenStream) Peek() *EnhancedToken {
	if ets.position >= len(ets.tokens) {
		return nil
	}
	return ets.tokens[ets.position]
}

// Reset 重置位置到开头
func (ets *EnhancedTokenStream) Reset() {
	ets.position = 0
}

// IsAtEnd 检查是否到达流末尾
func (ets *EnhancedTokenStream) IsAtEnd() bool {
	return ets.position >= len(ets.tokens)
}

// SourceFile 获取源文件
func (ets *EnhancedTokenStream) SourceFile() *SourceFile {
	return ets.sourceFile
}
