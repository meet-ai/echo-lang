// Package value_objects 定义词法分析上下文的值对象
package value_objects

import (
	"fmt"
	"math/big"
	"strconv"

	"echo/internal/modules/frontend/domain/shared/value_objects"
)

// NumberBase 数字进制枚举
type NumberBase int

const (
	BaseDecimal  NumberBase = 10 // 十进制
	BaseHex      NumberBase = 16 // 十六进制
	BaseBinary   NumberBase = 2  // 二进制
	BaseOctal    NumberBase = 8  // 八进制
)

// String 返回进制名称
func (nb NumberBase) String() string {
	switch nb {
	case BaseDecimal:
		return "decimal"
	case BaseHex:
		return "hexadecimal"
	case BaseBinary:
		return "binary"
	case BaseOctal:
		return "octal"
	default:
		return "unknown"
	}
}

// NumberLiteralType 数字字面量类型
type NumberLiteralType int

const (
	NumberTypeInt   NumberLiteralType = iota // 整数
	NumberTypeFloat                          // 浮点数
	NumberTypeBigInt                         // 大整数（超出int64范围）
)

// String 返回类型名称
func (nlt NumberLiteralType) String() string {
	switch nlt {
	case NumberTypeInt:
		return "int"
	case NumberTypeFloat:
		return "float"
	case NumberTypeBigInt:
		return "bigint"
	default:
		return "unknown"
	}
}

// NumberLiteral 数字字面量值对象
// 实现DDD值对象模式，支持多种进制和科学计数法
type NumberLiteral struct {
	// 原始字符串表示
	originalLexeme string

	// 解析后的数值
	intValue   int64         // 整数值（适用于NumberTypeInt）
	floatValue float64       // 浮点数值（适用于NumberTypeFloat）
	bigValue   *big.Int      // 大整数值（适用于NumberTypeBigInt）

	// 元数据
	base          NumberBase       // 进制
	literalType   NumberLiteralType // 类型
	hasExponent   bool             // 是否有指数部分
	isNegative    bool             // 是否为负数

	// 位置信息
	location      value_objects.SourceLocation   // 源代码位置
}

// NewNumberLiteral 创建新的数字字面量值对象
func NewNumberLiteral(lexeme string, base NumberBase, location value_objects.SourceLocation) (*NumberLiteral, error) {
	nl := &NumberLiteral{
		originalLexeme: lexeme,
		base:          base,
		location:      location,
		isNegative:    false,
		hasExponent:   false,
	}

	// 解析数字值
	if err := nl.parseValue(lexeme); err != nil {
		return nil, err
	}

	return nl, nil
}

// parseValue 解析数字字面量的值
func (nl *NumberLiteral) parseValue(lexeme string) error {
	// 处理负数
	if len(lexeme) > 0 && lexeme[0] == '-' {
		nl.isNegative = true
		lexeme = lexeme[1:]
	}

	// 检测是否为浮点数（包含小数点或指数）
	if nl.containsFloatIndicators(lexeme) {
		return nl.parseAsFloat(lexeme)
	}

	// 尝试解析为整数
	return nl.parseAsInteger(lexeme)
}

// containsFloatIndicators 检查是否包含浮点数特征
func (nl *NumberLiteral) containsFloatIndicators(lexeme string) bool {
	for _, r := range lexeme {
		if r == '.' || r == 'e' || r == 'E' {
			return true
		}
	}
	return false
}

// parseAsFloat 解析为浮点数
func (nl *NumberLiteral) parseAsFloat(lexeme string) error {
	nl.literalType = NumberTypeFloat

	// 检查指数标记
	nl.hasExponent = nl.hasExponentNotation(lexeme)

	// 解析浮点数值
	value, err := strconv.ParseFloat(lexeme, 64)
	if err != nil {
		return fmt.Errorf("invalid float literal '%s': %w", lexeme, err)
	}

	if nl.isNegative {
		value = -value
	}
	nl.floatValue = value

	return nil
}

// parseAsInteger 解析为整数
func (nl *NumberLiteral) parseAsInteger(lexeme string) error {
	// 移除进制前缀（如0x, 0b, 0o）
	cleanLexeme := nl.removeBasePrefix(lexeme)

	// 尝试解析为int64
	if value, err := strconv.ParseInt(cleanLexeme, int(nl.base), 64); err == nil {
		nl.literalType = NumberTypeInt
		if nl.isNegative {
			value = -value
		}
		nl.intValue = value
		return nil
	}

	// 如果int64不够用，使用big.Int
	bigValue := new(big.Int)
	if _, ok := bigValue.SetString(cleanLexeme, int(nl.base)); !ok {
		return fmt.Errorf("invalid integer literal '%s' for base %d", lexeme, nl.base)
	}

	if nl.isNegative {
		bigValue.Neg(bigValue)
	}

	nl.literalType = NumberTypeBigInt
	nl.bigValue = bigValue

	return nil
}

// removeBasePrefix 移除进制前缀
func (nl *NumberLiteral) removeBasePrefix(lexeme string) string {
	switch nl.base {
	case BaseHex:
		if len(lexeme) >= 2 && lexeme[0] == '0' && (lexeme[1] == 'x' || lexeme[1] == 'X') {
			return lexeme[2:]
		}
	case BaseBinary:
		if len(lexeme) >= 2 && lexeme[0] == '0' && (lexeme[1] == 'b' || lexeme[1] == 'B') {
			return lexeme[2:]
		}
	case BaseOctal:
		if len(lexeme) >= 2 && lexeme[0] == '0' && (lexeme[1] == 'o' || lexeme[1] == 'O') {
			return lexeme[2:]
		}
	}
	return lexeme
}

// hasExponentNotation 检查是否包含指数表示法
func (nl *NumberLiteral) hasExponentNotation(lexeme string) bool {
	for _, r := range lexeme {
		if r == 'e' || r == 'E' {
			return true
		}
	}
	return false
}

// Getters

// OriginalLexeme 获取原始词素
func (nl *NumberLiteral) OriginalLexeme() string {
	return nl.originalLexeme
}

// Base 获取进制
func (nl *NumberLiteral) Base() NumberBase {
	return nl.base
}

// Type 获取数字类型
func (nl *NumberLiteral) Type() NumberLiteralType {
	return nl.literalType
}

// IsFloat 检查是否为浮点数
func (nl *NumberLiteral) IsFloat() bool {
	return nl.literalType == NumberTypeFloat
}

// IsInteger 检查是否为整数
func (nl *NumberLiteral) IsInteger() bool {
	return nl.literalType == NumberTypeInt || nl.literalType == NumberTypeBigInt
}

// IsBigInt 检查是否为大整数
func (nl *NumberLiteral) IsBigInt() bool {
	return nl.literalType == NumberTypeBigInt
}

// HasExponent 检查是否有指数部分
func (nl *NumberLiteral) HasExponent() bool {
	return nl.hasExponent
}

// IsNegative 检查是否为负数
func (nl *NumberLiteral) IsNegative() bool {
	return nl.isNegative
}

// IntValue 获取整数值（仅当Type为NumberTypeInt时有效）
func (nl *NumberLiteral) IntValue() int64 {
	if nl.literalType != NumberTypeInt {
		panic("IntValue() called on non-integer NumberLiteral")
	}
	return nl.intValue
}

// FloatValue 获取浮点数值（仅当Type为NumberTypeFloat时有效）
func (nl *NumberLiteral) FloatValue() float64 {
	if nl.literalType != NumberTypeFloat {
		panic("FloatValue() called on non-float NumberLiteral")
	}
	return nl.floatValue
}

// BigValue 获取大整数值（仅当Type为NumberTypeBigInt时有效）
func (nl *NumberLiteral) BigValue() *big.Int {
	if nl.literalType != NumberTypeBigInt {
		panic("BigValue() called on non-bigint NumberLiteral")
	}
	return nl.bigValue
}

// Location 获取源代码位置
func (nl *NumberLiteral) Location() value_objects.SourceLocation {
	return nl.location
}

// Value 获取通用值接口（用于兼容性）
func (nl *NumberLiteral) Value() interface{} {
	switch nl.literalType {
	case NumberTypeInt:
		return nl.intValue
	case NumberTypeFloat:
		return nl.floatValue
	case NumberTypeBigInt:
		return nl.bigValue
	default:
		return nil
	}
}

// Equals 比较两个NumberLiteral是否相等（值对象语义）
func (nl *NumberLiteral) Equals(other *NumberLiteral) bool {
	if other == nil {
		return false
	}

	if nl.literalType != other.literalType ||
	   nl.base != other.base ||
	   nl.hasExponent != other.hasExponent ||
	   nl.isNegative != other.isNegative {
		return false
	}

	switch nl.literalType {
	case NumberTypeInt:
		return nl.intValue == other.intValue
	case NumberTypeFloat:
		return nl.floatValue == other.floatValue
	case NumberTypeBigInt:
		return nl.bigValue.Cmp(other.bigValue) == 0
	default:
		return false
	}
}

// String 返回字符串表示
func (nl *NumberLiteral) String() string {
	typeStr := nl.literalType.String()
	baseStr := nl.base.String()

	switch nl.literalType {
	case NumberTypeInt:
		return fmt.Sprintf("NumberLiteral{%s, base=%s, value=%d}",
			typeStr, baseStr, nl.intValue)
	case NumberTypeFloat:
		return fmt.Sprintf("NumberLiteral{%s, base=%s, value=%g}",
			typeStr, baseStr, nl.floatValue)
	case NumberTypeBigInt:
		return fmt.Sprintf("NumberLiteral{%s, base=%s, value=%s}",
			typeStr, baseStr, nl.bigValue.String())
	default:
		return fmt.Sprintf("NumberLiteral{unknown, lexeme=%s}", nl.originalLexeme)
	}
}

// 工厂方法

// NewDecimalNumber 创建十进制数字字面量
func NewDecimalNumber(lexeme string, location value_objects.SourceLocation) (*NumberLiteral, error) {
	return NewNumberLiteral(lexeme, BaseDecimal, location)
}

// NewHexNumber 创建十六进制数字字面量
func NewHexNumber(lexeme string, location value_objects.SourceLocation) (*NumberLiteral, error) {
	return NewNumberLiteral(lexeme, BaseHex, location)
}

// NewBinaryNumber 创建二进制数字字面量
func NewBinaryNumber(lexeme string, location value_objects.SourceLocation) (*NumberLiteral, error) {
	return NewNumberLiteral(lexeme, BaseBinary, location)
}

// NewOctalNumber 创建八进制数字字面量
func NewOctalNumber(lexeme string, location value_objects.SourceLocation) (*NumberLiteral, error) {
	return NewNumberLiteral(lexeme, BaseOctal, location)
}
