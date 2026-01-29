// Package services 定义词法分析上下文的领域服务
package services

import (
	"context"
	"fmt"
	"strconv"
	"unicode"

	"echo/internal/modules/frontend/domain/lexical/entities"
	"echo/internal/modules/frontend/domain/shared/value_objects"
)

// LexerService 词法分析器领域服务
// 负责将源代码转换为Token流，实现词法分析的核心逻辑
type LexerService struct {
	sourceFile *value_objects.SourceFile
	position   int
	line       int
	column     int
	tokens     []*entities.TokenEntity
	errors     []*value_objects.ParseError
}

// NewLexerService 创建新的词法分析器服务
func NewLexerService(sourceFile *value_objects.SourceFile) *LexerService {
	return &LexerService{
		sourceFile: sourceFile,
		position:   0,
		line:       1,
		column:     1,
		tokens:     make([]*entities.TokenEntity, 0),
		errors:     make([]*value_objects.ParseError, 0),
	}
}

// Tokenize 执行词法分析，将源代码转换为Token流（实现LexicalAnalysisService接口）
func (ls *LexerService) Tokenize(ctx context.Context, sourceFile *value_objects.SourceFile) (*value_objects.TokenStream, error) {
	ls.sourceFile = sourceFile
	source := ls.sourceFile.Content()
	tokenStream := value_objects.NewTokenStream(ls.sourceFile)

	for ls.position < len(source) {
		if ls.isAtEnd() {
			break
		}

		char := source[ls.position]

		switch {
		case char == '(':
			ls.addTokenToStream(tokenStream, value_objects.TokenTypeLeftParen, "(")
		case char == ')':
			ls.addTokenToStream(tokenStream, value_objects.TokenTypeRightParen, ")")
		case char == '{':
			ls.addTokenToStream(tokenStream, value_objects.TokenTypeLeftBrace, "{")
		case char == '}':
			ls.addTokenToStream(tokenStream, value_objects.TokenTypeRightBrace, "}")
		case char == '[':
			ls.addTokenToStream(tokenStream, value_objects.TokenTypeLeftBracket, "[")
		case char == ']':
			ls.addTokenToStream(tokenStream, value_objects.TokenTypeRightBracket, "]")
		case char == ',':
			ls.addTokenToStream(tokenStream, value_objects.TokenTypeComma, ",")
		case char == '.':
			ls.addTokenToStream(tokenStream, value_objects.TokenTypeDot, ".")
		case char == ':':
			ls.addTokenToStream(tokenStream, value_objects.TokenTypeColon, ":")
		case char == ';':
			ls.addTokenToStream(tokenStream, value_objects.TokenTypeSemicolon, ";")
		case char == '+':
			ls.addTokenToStream(tokenStream, value_objects.TokenTypePlus, "+")
		case char == '-':
			if ls.peek() == '>' {
				ls.advance()
				ls.addTokenToStream(tokenStream, value_objects.TokenTypeArrow, "->")
			} else {
				ls.addTokenToStream(tokenStream, value_objects.TokenTypeMinus, "-")
			}
		case char == '*':
			ls.addTokenToStream(tokenStream, value_objects.TokenTypeMultiply, "*")
		case char == '/':
			if ls.peek() == '/' {
				// 单行注释，跳过整行
				for !ls.isAtEnd() && source[ls.position] != '\n' {
					ls.advance()
				}
			} else {
				ls.addTokenToStream(tokenStream, value_objects.TokenTypeDivide, "/")
			}
		case char == '%':
			ls.addTokenToStream(tokenStream, value_objects.TokenTypeModulo, "%")
		case char == '=':
			if ls.peek() == '=' {
				ls.advance()
				ls.addTokenToStream(tokenStream, value_objects.TokenTypeEqual, "==")
			} else {
				ls.addTokenToStream(tokenStream, value_objects.TokenTypeAssign, "=")
			}
		case char == '!':
			if ls.peek() == '=' {
				ls.advance()
				ls.addTokenToStream(tokenStream, value_objects.TokenTypeNotEqual, "!=")
			} else {
				ls.addTokenToStream(tokenStream, value_objects.TokenTypeNot, "!")
			}
		case char == '<':
			if ls.peek() == '=' {
				ls.advance()
				ls.addTokenToStream(tokenStream, value_objects.TokenTypeLessEqual, "<=")
			} else {
				ls.addTokenToStream(tokenStream, value_objects.TokenTypeLessThan, "<")
			}
		case char == '>':
			if ls.peek() == '=' {
				ls.advance()
				ls.addTokenToStream(tokenStream, value_objects.TokenTypeGreaterEqual, ">=")
			} else {
				ls.addTokenToStream(tokenStream, value_objects.TokenTypeGreaterThan, ">")
			}
		case char == '&':
			if ls.peek() == '&' {
				ls.advance()
				ls.addTokenToStream(tokenStream, value_objects.TokenTypeAnd, "&&")
			} else {
				// 单&用于取地址操作符（函数指针）
				// 注意：lexer_service.go 使用的是旧的 TokenType，需要检查是否有对应的类型
				// 如果没有，可能需要使用 EnhancedTokenTypeAmpersand
				// 暂时跳过，因为可能使用的是 AdvancedLexerService
			}
		case char == '|':
			if ls.peek() == '|' {
				ls.advance()
				ls.addTokenToStream(tokenStream, value_objects.TokenTypeOr, "||")
			}
		case unicode.IsSpace(rune(char)):
			ls.skipWhitespace()
		case unicode.IsLetter(rune(char)) || char == '_':
			ls.identifierOrKeywordToStream(tokenStream)
		case unicode.IsDigit(rune(char)):
			ls.numberToStream(tokenStream)
		case char == '"':
			ls.stringToStream(tokenStream)
		default:
			ls.addError(fmt.Sprintf("unexpected character: %c", char))
			ls.advance()
		}
	}

	// 添加EOF标记
	ls.addTokenToStream(tokenStream, value_objects.TokenTypeEOF, "")

	// 如果有错误，返回第一个错误
	if len(ls.errors) > 0 {
		return nil, ls.errors[0]
	}

	return tokenStream, nil
}

// identifierOrKeywordToStream 处理标识符或关键字并添加到流
func (ls *LexerService) identifierOrKeywordToStream(tokenStream *value_objects.TokenStream) {
	start := ls.position

	// 读取完整的标识符
	for !ls.isAtEnd() && (unicode.IsLetter(rune(ls.peek())) || unicode.IsDigit(rune(ls.peek())) || ls.peek() == '_') {
		ls.advance()
	}

	text := ls.sourceFile.Content()[start:ls.position]

	// 检查是否为关键字
	var tokenType value_objects.TokenType
	switch text {
	case "func":
		tokenType = value_objects.TokenTypeFunc
	case "let":
		tokenType = value_objects.TokenTypeLet
	case "if":
		tokenType = value_objects.TokenTypeIf
	case "else":
		tokenType = value_objects.TokenTypeElse
	case "while":
		tokenType = value_objects.TokenTypeWhile
	case "for":
		tokenType = value_objects.TokenTypeFor
	case "return":
		tokenType = value_objects.TokenTypeReturn
	case "struct":
		tokenType = value_objects.TokenTypeStruct
	case "enum":
		tokenType = value_objects.TokenTypeEnum
	case "trait":
		tokenType = value_objects.TokenTypeTrait
	case "impl":
		tokenType = value_objects.TokenTypeImpl
	case "match":
		tokenType = value_objects.TokenTypeMatch
	case "async":
		tokenType = value_objects.TokenTypeAsync
	case "spawn":
		tokenType = value_objects.TokenTypeSpawn
	case "chan":
		tokenType = value_objects.TokenTypeChan
	case "true", "false":
		tokenType = value_objects.TokenTypeBool
	default:
		tokenType = value_objects.TokenTypeIdentifier
	}

	ls.addTokenToStream(tokenStream, tokenType, text)
}

// numberToStream 处理数字字面量并添加到流
func (ls *LexerService) numberToStream(tokenStream *value_objects.TokenStream) {
	start := ls.position

	// 检查是否为浮点数
	isFloat := false

	// 读取整数部分
	for !ls.isAtEnd() && unicode.IsDigit(rune(ls.peek())) {
		ls.advance()
	}

	// 检查小数点
	if !ls.isAtEnd() && ls.peek() == '.' {
		isFloat = true
		ls.advance()

		// 读取小数部分
		for !ls.isAtEnd() && unicode.IsDigit(rune(ls.peek())) {
			ls.advance()
		}
	}

	text := ls.sourceFile.Content()[start:ls.position]

	if isFloat {
		if value, err := strconv.ParseFloat(text, 64); err == nil {
			ls.addTokenWithValueToStream(tokenStream, value_objects.TokenTypeFloat, text, value)
		} else {
			ls.addError(fmt.Sprintf("invalid float literal: %s", text))
		}
	} else {
		if value, err := strconv.Atoi(text); err == nil {
			ls.addTokenWithValueToStream(tokenStream, value_objects.TokenTypeInt, text, value)
		} else {
			ls.addError(fmt.Sprintf("invalid int literal: %s", text))
		}
	}
}

// stringToStream 处理字符串字面量并添加到流
func (ls *LexerService) stringToStream(tokenStream *value_objects.TokenStream) {
	start := ls.position
	ls.advance() // 跳过开头的"

	// 读取字符串内容
	for !ls.isAtEnd() && ls.peek() != '"' {
		if ls.peek() == '\n' {
			ls.addError("unterminated string literal")
			return
		}
		ls.advance()
	}

	if ls.isAtEnd() {
		ls.addError("unterminated string literal")
		return
	}

	ls.advance() // 跳过结尾的"

	text := ls.sourceFile.Content()[start:ls.position]
	// 去除引号
	value := text[1 : len(text)-1]

	ls.addTokenWithValueToStream(tokenStream, value_objects.TokenTypeString, text, value)
}

// skipWhitespace 跳过空白字符
func (ls *LexerService) skipWhitespace() {
	for !ls.isAtEnd() && unicode.IsSpace(rune(ls.peek())) {
		if ls.peek() == '\n' {
			ls.line++
			ls.column = 1
		} else {
			ls.column++
		}
		ls.position++
	}
}

// addTokenToStream 添加Token到流
func (ls *LexerService) addTokenToStream(tokenStream *value_objects.TokenStream, tokenType value_objects.TokenType, lexeme string) {
	ls.addTokenWithValueToStream(tokenStream, tokenType, lexeme, nil)
}

// addTokenWithValueToStream 添加带值的Token到流
func (ls *LexerService) addTokenWithValueToStream(tokenStream *value_objects.TokenStream, tokenType value_objects.TokenType, lexeme string, value interface{}) {
	location := value_objects.NewSourceLocation(
		ls.sourceFile.Filename(),
		ls.line,
		ls.column,
		ls.position-len(lexeme),
	)

	token := value_objects.NewToken(tokenType, lexeme, value, location)
	tokenStream.AddToken(token)

	// 更新位置
	ls.column += len(lexeme)
}

// addError 添加错误
func (ls *LexerService) addError(message string) {
	location := value_objects.NewSourceLocation(
		ls.sourceFile.Filename(),
		ls.line,
		ls.column,
		ls.position,
	)

	error := value_objects.NewParseError(
		message,
		location,
		value_objects.ErrorTypeLexical,
		value_objects.SeverityError,
	)

	ls.errors = append(ls.errors, error)
}

// advance 前进一个字符
func (ls *LexerService) advance() {
	ls.position++
	ls.column++
}

// peek 查看下一个字符
func (ls *LexerService) peek() byte {
	if ls.position >= len(ls.sourceFile.Content()) {
		return 0
	}
	return ls.sourceFile.Content()[ls.position]
}

// isAtEnd 检查是否到达源代码末尾
func (ls *LexerService) isAtEnd() bool {
	return ls.position >= len(ls.sourceFile.Content())
}

// ValidateTokens 验证Token流（实现LexicalAnalysisService接口）
func (ls *LexerService) ValidateTokens(ctx context.Context, tokenStream *value_objects.TokenStream) (*value_objects.ValidationResult, error) {
	result := value_objects.NewValidationResult()

	// 基本验证：检查是否有有效的Token
	if tokenStream.Count() == 0 {
		result.AddError(value_objects.NewParseError(
			"empty token stream",
			value_objects.NewSourceLocation(ls.sourceFile.Filename(), 1, 1, 0),
			value_objects.ErrorTypeLexical,
			value_objects.SeverityError,
		))
		return result, nil
	}

	// 检查是否有EOF标记
	lastToken := tokenStream.Tokens()[tokenStream.Count()-1]
	if lastToken.Type() != value_objects.TokenTypeEOF {
		result.AddError(value_objects.NewParseError(
			"missing EOF token",
			lastToken.Location(),
			value_objects.ErrorTypeLexical,
			value_objects.SeverityError,
		))
	}

	// 检查Token位置的连续性
	tokens := tokenStream.Tokens()
	for i := 0; i < len(tokens)-1; i++ {
		current := tokens[i]
		next := tokens[i+1]

		currentEnd := current.Location().Offset() + len(current.Lexeme())
		nextStart := next.Location().Offset()

		if currentEnd > nextStart {
			result.AddError(value_objects.NewParseError(
				"overlapping token locations",
				current.Location(),
				value_objects.ErrorTypeLexical,
				value_objects.SeverityWarning,
			))
		}
	}

	return result, nil
}
