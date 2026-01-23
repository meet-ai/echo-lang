// Package services 定义词法分析上下文的领域服务
package services

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
)

// AdvancedLexerService 高级词法分析器领域服务
// 实现文档中描述的现代化词法分析，支持多种进制数字和高级特性
type AdvancedLexerService struct {
	sourceFile *lexicalVO.SourceFile
	position   int
	line       int
	column     int
	source     string
	tokens     []*lexicalVO.EnhancedToken
	errors     []*sharedVO.ParseError
}

// NewAdvancedLexerService 创建新的高级词法分析器服务
func NewAdvancedLexerService() *AdvancedLexerService {
	return &AdvancedLexerService{
		position: 0,
		line:     1,
		column:   1,
		tokens:   make([]*lexicalVO.EnhancedToken, 0),
		errors:   make([]*sharedVO.ParseError, 0),
	}
}

// Tokenize 执行词法分析，将源代码转换为增强Token流
func (als *AdvancedLexerService) Tokenize(ctx context.Context, sourceFile *lexicalVO.SourceFile) (*lexicalVO.EnhancedTokenStream, error) {
	als.sourceFile = sourceFile
	als.source = sourceFile.Content()
	als.position = 0
	als.line = 1
	als.column = 1
	als.tokens = make([]*lexicalVO.EnhancedToken, 0)
	als.errors = make([]*sharedVO.ParseError, 0)

	tokenStream := lexicalVO.NewEnhancedTokenStream(sourceFile)

	// 主词法分析循环
	for !als.isAtEnd() {
		if als.isCancelled(ctx) {
			return nil, fmt.Errorf("tokenization cancelled")
		}

		als.skipWhitespaceAndComments()

		if als.isAtEnd() {
			break
		}

		if token := als.scanToken(); token != nil {
			tokenStream.AddToken(token)
		}
	}

	// 添加EOF标记
	eofLocation := sharedVO.NewSourceLocation(
		sourceFile.Filename(),
		als.line,
		als.column,
		als.position,
	)
	eofToken := lexicalVO.NewEnhancedToken(
		lexicalVO.EnhancedTokenTypeEOF,
		"",
		eofLocation,
	)
	tokenStream.AddToken(eofToken)

	// 检查是否有错误
	if len(als.errors) > 0 {
		return nil, als.errors[0] // 返回第一个错误
	}

	return tokenStream, nil
}

// scanToken 扫描单个Token
func (als *AdvancedLexerService) scanToken() *lexicalVO.EnhancedToken {
	char := als.advance()
	startLocation := sharedVO.NewSourceLocation(
		als.sourceFile.Filename(),
		als.line,
		als.column-1, // column已经前进过了，所以减1
		als.position-1,
	)

	switch char {
	// 单字符分隔符
	case '(':
		return als.createToken(lexicalVO.EnhancedTokenTypeLeftParen, "(", startLocation)
	case ')':
		return als.createToken(lexicalVO.EnhancedTokenTypeRightParen, ")", startLocation)
	case '{':
		return als.createToken(lexicalVO.EnhancedTokenTypeLeftBrace, "{", startLocation)
	case '}':
		return als.createToken(lexicalVO.EnhancedTokenTypeRightBrace, "}", startLocation)
	case '[':
		return als.createToken(lexicalVO.EnhancedTokenTypeLeftBracket, "[", startLocation)
	case ']':
		return als.createToken(lexicalVO.EnhancedTokenTypeRightBracket, "]", startLocation)
	case ',':
		return als.createToken(lexicalVO.EnhancedTokenTypeComma, ",", startLocation)
	case '.':
		return als.createToken(lexicalVO.EnhancedTokenTypeDot, ".", startLocation)
	case ':':
		if als.match('=') {
			return als.createToken(lexicalVO.EnhancedTokenTypeDeclare, ":=", startLocation)
		}
		return als.createToken(lexicalVO.EnhancedTokenTypeColon, ":", startLocation)
	case ';':
		return als.createToken(lexicalVO.EnhancedTokenTypeSemicolon, ";", startLocation)

	// 可能双字符的运算符
	case '+':
		return als.createToken(lexicalVO.EnhancedTokenTypePlus, "+", startLocation)
	case '*':
		return als.createToken(lexicalVO.EnhancedTokenTypeMultiply, "*", startLocation)
	case '/':
		return als.createToken(lexicalVO.EnhancedTokenTypeDivide, "/", startLocation)
	case '%':
		return als.createToken(lexicalVO.EnhancedTokenTypeModulo, "%", startLocation)
	case '=':
		if als.match('=') {
			return als.createToken(lexicalVO.EnhancedTokenTypeEqual, "==", startLocation)
		}
		if als.match('>') {
			return als.createToken(lexicalVO.EnhancedTokenTypeFatArrow, "=>", startLocation)
		}
		return als.createToken(lexicalVO.EnhancedTokenTypeAssign, "=", startLocation)
	case '!':
		if als.match('=') {
			return als.createToken(lexicalVO.EnhancedTokenTypeNotEqual, "!=", startLocation)
		}
		return als.createToken(lexicalVO.EnhancedTokenTypeNot, "!", startLocation)
	case '<':
		if als.match('=') {
			return als.createToken(lexicalVO.EnhancedTokenTypeLessEqual, "<=", startLocation)
		}
		if als.match('-') {
			return als.createToken(lexicalVO.EnhancedTokenTypeChannelReceive, "<-", startLocation)
		}
		return als.createToken(lexicalVO.EnhancedTokenTypeLessThan, "<", startLocation)
	case '>':
		if als.match('=') {
			return als.createToken(lexicalVO.EnhancedTokenTypeGreaterEqual, ">=", startLocation)
		}
		return als.createToken(lexicalVO.EnhancedTokenTypeGreaterThan, ">", startLocation)
	case '&':
		if als.match('&') {
			return als.createToken(lexicalVO.EnhancedTokenTypeAnd, "&&", startLocation)
		}
		// 单&在Echo语言中可能不支持，标记为错误
		als.addError("unexpected character '&'", startLocation)
		return nil
	case '|':
		if als.match('|') {
			return als.createToken(lexicalVO.EnhancedTokenTypeOr, "||", startLocation)
		}
		// 单|在Echo语言中可能不支持，标记为错误
		als.addError("unexpected character '|'", startLocation)
		return nil
	case '-':
		if als.match('>') {
			return als.createToken(lexicalVO.EnhancedTokenTypeArrow, "->", startLocation)
		}
		return als.createToken(lexicalVO.EnhancedTokenTypeMinus, "-", startLocation)

	// 字符串字面量
	case '"':
		return als.scanString(startLocation)

	// 数字字面量
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return als.scanNumber(char, startLocation)

	// 标识符或关键字
	default:
		if unicode.IsLetter(rune(char)) || char == '_' {
			return als.scanIdentifierOrKeyword(startLocation)
		}

		// 未知字符
		als.addError(fmt.Sprintf("unexpected character: %c", char), startLocation)
		return nil
	}
}

// scanNumber 扫描数字字面量（支持多种进制）
func (als *AdvancedLexerService) scanNumber(firstChar byte, startLocation sharedVO.SourceLocation) *lexicalVO.EnhancedToken {
	start := als.position - 1 // 回退到数字开始的位置

	// 检查进制前缀
	base := lexicalVO.BaseDecimal
	if firstChar == '0' && !als.isAtEnd() {
		next := als.peek()
		switch next {
		case 'x', 'X':
			base = lexicalVO.BaseHex
			als.advance() // 消耗'x'或'X'
		case 'b', 'B':
			base = lexicalVO.BaseBinary
			als.advance() // 消耗'b'或'B'
		case 'o', 'O':
			base = lexicalVO.BaseOctal
			als.advance() // 消耗'o'或'O'
		}
	}

	// 扫描数字主体
	for !als.isAtEnd() {
		char := als.peek()
		if als.isDigitForBase(char, base) {
			als.advance()
		} else if char == '.' && base == lexicalVO.BaseDecimal {
			// 小数点只在十进制中有效
			als.advance()
			// 扫描小数部分
			for !als.isAtEnd() && unicode.IsDigit(rune(als.peek())) {
				als.advance()
			}
		} else if (char == 'e' || char == 'E') && base == lexicalVO.BaseDecimal {
			// 指数只在十进制中有效
			als.advance()
			// 处理指数符号
			if !als.isAtEnd() && (als.peek() == '+' || als.peek() == '-') {
				als.advance()
			}
			// 扫描指数数字
			for !als.isAtEnd() && unicode.IsDigit(rune(als.peek())) {
				als.advance()
			}
		} else {
			break
		}
	}

	// 提取完整的数字词素
	lexeme := als.source[start:als.position]

	// 创建NumberLiteral
	numberLiteral, err := als.createNumberLiteral(lexeme, base, startLocation)
	if err != nil {
		als.addError(fmt.Sprintf("invalid number literal: %v", err), startLocation)
		return nil
	}

	return lexicalVO.NewNumberToken(numberLiteral)
}

// isDigitForBase 检查字符是否为指定进制的有效数字
func (als *AdvancedLexerService) isDigitForBase(char byte, base lexicalVO.NumberBase) bool {
	switch base {
	case lexicalVO.BaseBinary:
		return char == '0' || char == '1'
	case lexicalVO.BaseOctal:
		return char >= '0' && char <= '7'
	case lexicalVO.BaseDecimal:
		return unicode.IsDigit(rune(char))
	case lexicalVO.BaseHex:
		return unicode.IsDigit(rune(char)) ||
			   (char >= 'a' && char <= 'f') ||
			   (char >= 'A' && char <= 'F')
	default:
		return false
	}
}

// createNumberLiteral 创建数字字面量值对象
func (als *AdvancedLexerService) createNumberLiteral(lexeme string, base lexicalVO.NumberBase, location sharedVO.SourceLocation) (*lexicalVO.NumberLiteral, error) {
	switch base {
	case lexicalVO.BaseHex:
		return lexicalVO.NewHexNumber(lexeme, location)
	case lexicalVO.BaseBinary:
		return lexicalVO.NewBinaryNumber(lexeme, location)
	case lexicalVO.BaseOctal:
		return lexicalVO.NewOctalNumber(lexeme, location)
	default:
		return lexicalVO.NewDecimalNumber(lexeme, location)
	}
}

// scanString 扫描字符串字面量
func (als *AdvancedLexerService) scanString(startLocation sharedVO.SourceLocation) *lexicalVO.EnhancedToken {
	start := als.position - 1 // 包含开头的引号

	for !als.isAtEnd() {
		char := als.peek()
		als.advance()

		if char == '"' {
			// 找到结束引号
			break
		}

		if char == '\n' {
			// 字符串不能跨行
			als.addError("unterminated string literal", startLocation)
			return nil
		}

		// 处理转义序列
		if char == '\\' && !als.isAtEnd() {
			als.advance() // 消耗转义字符
			// 这里可以扩展支持更多的转义序列
		}
	}

	if als.isAtEnd() {
		als.addError("unterminated string literal", startLocation)
		return nil
	}

	// 提取完整词素（包含引号）
	lexeme := als.source[start:als.position]

	// 解析字符串值（去除引号并处理转义）
	stringValue := als.parseStringValue(lexeme)

	return lexicalVO.NewStringToken(lexeme, stringValue, startLocation)
}

// parseStringValue 解析字符串值
func (als *AdvancedLexerService) parseStringValue(lexeme string) string {
	if len(lexeme) < 2 {
		return ""
	}

	// 去除首尾引号
	content := lexeme[1 : len(lexeme)-1]

	// 简单转义处理（可以扩展）
	content = strings.ReplaceAll(content, "\\n", "\n")
	content = strings.ReplaceAll(content, "\\t", "\t")
	content = strings.ReplaceAll(content, "\\\"", "\"")
	content = strings.ReplaceAll(content, "\\\\", "\\")

	return content
}

// scanIdentifierOrKeyword 扫描标识符或关键字
func (als *AdvancedLexerService) scanIdentifierOrKeyword(startLocation sharedVO.SourceLocation) *lexicalVO.EnhancedToken {
	start := als.position - 1

	for !als.isAtEnd() && (unicode.IsLetter(rune(als.peek())) ||
						   unicode.IsDigit(rune(als.peek())) ||
						   als.peek() == '_') {
		als.advance()
	}

	lexeme := als.source[start:als.position]

	// 检查是否为关键字
	tokenType := als.getKeywordType(lexeme)

	location := sharedVO.NewSourceLocation(
		als.sourceFile.Filename(),
		startLocation.Line(),
		startLocation.Column(),
		start,
	)

	if tokenType != lexicalVO.EnhancedTokenTypeIdentifier {
		// 是关键字或字面量
		if tokenType == lexicalVO.EnhancedTokenTypeBool {
			// bool 字面量需要特殊处理
			boolValue := lexeme == "true"
			return lexicalVO.NewBoolToken(lexeme, boolValue, location)
		}
		return lexicalVO.NewEnhancedToken(tokenType, lexeme, location)
	} else {
		// 是标识符
		return lexicalVO.NewIdentifierToken(lexeme, location)
	}
}

// getKeywordType 获取关键字类型
func (als *AdvancedLexerService) getKeywordType(word string) lexicalVO.EnhancedTokenType {
	switch word {
	case "func":
		return lexicalVO.EnhancedTokenTypeFunc
	case "let":
		return lexicalVO.EnhancedTokenTypeLet
	case "if":
		return lexicalVO.EnhancedTokenTypeIf
	case "else":
		return lexicalVO.EnhancedTokenTypeElse
	case "while":
		return lexicalVO.EnhancedTokenTypeWhile
	case "for":
		return lexicalVO.EnhancedTokenTypeFor
	case "return":
		return lexicalVO.EnhancedTokenTypeReturn
	case "struct":
		return lexicalVO.EnhancedTokenTypeStruct
	case "enum":
		return lexicalVO.EnhancedTokenTypeEnum
	case "trait":
		return lexicalVO.EnhancedTokenTypeTrait
	case "impl":
		return lexicalVO.EnhancedTokenTypeImpl
	case "match":
		return lexicalVO.EnhancedTokenTypeMatch
	case "async":
		return lexicalVO.EnhancedTokenTypeAsync
	case "spawn":
		return lexicalVO.EnhancedTokenTypeSpawn
	case "chan":
		return lexicalVO.EnhancedTokenTypeChan
	case "print":
		return lexicalVO.EnhancedTokenTypePrint
	case "package":
		return lexicalVO.EnhancedTokenTypePackage
	case "private":
		return lexicalVO.EnhancedTokenTypePrivate
	case "from":
		return lexicalVO.EnhancedTokenTypeFrom
	case "select":
		return lexicalVO.EnhancedTokenTypeSelect
	case "case":
		return lexicalVO.EnhancedTokenTypeCase
	case "true", "false":
		return lexicalVO.EnhancedTokenTypeBool
	default:
		return lexicalVO.EnhancedTokenTypeIdentifier
	}
}

// skipWhitespaceAndComments 跳过空白字符和注释
func (als *AdvancedLexerService) skipWhitespaceAndComments() {
	for !als.isAtEnd() {
		char := als.peek()

		switch {
		case unicode.IsSpace(rune(char)):
			als.skipWhitespace()
		case char == '/' && als.peekNext() == '/':
			als.skipLineComment()
		case char == '/' && als.peekNext() == '*':
			als.skipBlockComment()
		default:
			return
		}
	}
}

// skipWhitespace 跳过空白字符
func (als *AdvancedLexerService) skipWhitespace() {
	for !als.isAtEnd() && unicode.IsSpace(rune(als.peek())) {
		if als.peek() == '\n' {
			als.line++
			als.column = 1
		} else {
			als.column++
		}
		als.position++
	}
}

// skipLineComment 跳过单行注释
func (als *AdvancedLexerService) skipLineComment() {
	for !als.isAtEnd() && als.peek() != '\n' {
		als.position++
		als.column++
	}
}

// skipBlockComment 跳过多行注释
func (als *AdvancedLexerService) skipBlockComment() {
	als.position += 2 // 跳过 /*
	als.column += 2

	for !als.isAtEnd() {
		if als.peek() == '*' && als.peekNext() == '/' {
			als.position += 2 // 跳过 */
			als.column += 2
			return
		}

		if als.peek() == '\n' {
			als.line++
			als.column = 1
		} else {
			als.column++
		}
		als.position++
	}

	als.addError("unterminated block comment", sharedVO.NewSourceLocation(
		als.sourceFile.Filename(), als.line, als.column, als.position))
}

// 工具方法

// createToken 创建普通Token
func (als *AdvancedLexerService) createToken(tokenType lexicalVO.EnhancedTokenType, lexeme string, location sharedVO.SourceLocation) *lexicalVO.EnhancedToken {
	return lexicalVO.NewEnhancedToken(tokenType, lexeme, location)
}

// advance 前进一个字符并更新位置
func (als *AdvancedLexerService) advance() byte {
	char := als.source[als.position]
	als.position++
	als.column++
	return char
}

// peek 查看下一个字符
func (als *AdvancedLexerService) peek() byte {
	if als.isAtEnd() {
		return 0
	}
	return als.source[als.position]
}

// peekNext 查看下下个字符
func (als *AdvancedLexerService) peekNext() byte {
	if als.position+1 >= len(als.source) {
		return 0
	}
	return als.source[als.position+1]
}

// match 如果下一个字符匹配则消耗它
func (als *AdvancedLexerService) match(expected byte) bool {
	if als.isAtEnd() || als.peek() != expected {
		return false
	}
	als.advance()
	return true
}

// isAtEnd 检查是否到达源代码末尾
func (als *AdvancedLexerService) isAtEnd() bool {
	return als.position >= len(als.source)
}

// addError 添加错误
func (als *AdvancedLexerService) addError(message string, location sharedVO.SourceLocation) {
	error := sharedVO.NewParseError(message, location,
		sharedVO.ErrorTypeLexical, sharedVO.SeverityError)
	als.errors = append(als.errors, error)
}

// isCancelled 检查是否被取消
func (als *AdvancedLexerService) isCancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// ValidateTokens 验证Token流（实现接口）
func (als *AdvancedLexerService) ValidateTokens(ctx context.Context, tokenStream *lexicalVO.EnhancedTokenStream) (*sharedVO.ValidationResult, error) {
	result := sharedVO.NewValidationResult()

	// 基本验证：检查是否有有效的Token
	if tokenStream.Count() == 0 {
		result.AddError(sharedVO.NewParseError(
			"empty token stream",
			sharedVO.NewSourceLocation(als.sourceFile.Filename(), 1, 1, 0),
			sharedVO.ErrorTypeLexical,
			sharedVO.SeverityError,
		))
		return result, nil
	}

	// 检查是否有EOF标记
	tokens := tokenStream.Tokens()
	lastToken := tokens[len(tokens)-1]
	if lastToken.Type() != lexicalVO.EnhancedTokenTypeEOF {
		result.AddError(sharedVO.NewParseError(
			"missing EOF token",
			lastToken.Location(),
			sharedVO.ErrorTypeLexical,
			sharedVO.SeverityError,
		))
	}

	return result, nil
}
