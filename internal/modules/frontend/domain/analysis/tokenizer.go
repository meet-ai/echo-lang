package analysis

import (
	"fmt"
	"unicode"
)

// Tokenizer 词法分析器接口
type Tokenizer interface {
	// Tokenize 将源代码转换为Token序列
	Tokenize(source SourceCode) ([]Token, error)
}

// SimpleTokenizer 简单词法分析器实现
type SimpleTokenizer struct{}

// NewSimpleTokenizer 创建新的简单词法分析器
func NewSimpleTokenizer() Tokenizer {
	return &SimpleTokenizer{}
}

// Tokenize 实现Tokenizer接口
func (t *SimpleTokenizer) Tokenize(source SourceCode) ([]Token, error) {
	var tokens []Token
	lines := source.Lines()

	currentLine := 1
	for _, line := range lines {
		lineTokens, err := t.tokenizeLine(line, currentLine, source.FilePath())
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, lineTokens...)
		currentLine++
	}

	// 添加EOF标记
	// 计算EOF位置：最后一个字符的下一个位置
	content := source.Content()
	var eofPosition Position
	if content == "" {
		eofPosition = NewPosition(1, 1, source.FilePath())
	} else {
		runes := []rune(content)
		lineNum := 1
		column := 1

		for _, r := range runes {
			if r == '\n' {
				lineNum++
				column = 1
			} else {
				column++
			}
		}

		eofPosition = NewPosition(lineNum, column, source.FilePath())
	}

	tokens = append(tokens, NewToken(EOF, "", eofPosition))

	return tokens, nil
}

// tokenizeLine 对单行代码进行词法分析
func (t *SimpleTokenizer) tokenizeLine(line string, lineNum int, fileName string) ([]Token, error) {
	var tokens []Token
	runes := []rune(line)
	i := 0

	for i < len(runes) {
		char := runes[i]

		// 跳过空白字符
		if unicode.IsSpace(char) {
			i++
			continue
		}

		// 跳过注释
		if char == '/' && i+1 < len(runes) && runes[i+1] == '/' {
			break // 忽略行尾注释
		}

		// 识别不同类型的Token
		if token, consumed := t.tryTokenizeString(runes[i:], lineNum, i+1, fileName); consumed > 0 {
			tokens = append(tokens, token)
			i += consumed
		} else if token, consumed := t.tryTokenizeNumber(runes[i:], lineNum, i+1, fileName); consumed > 0 {
			tokens = append(tokens, token)
			i += consumed
		} else if token, consumed := t.tryTokenizeIdentifierOrKeyword(runes[i:], lineNum, i+1, fileName); consumed > 0 {
			tokens = append(tokens, token)
			i += consumed
		} else if token, consumed := t.tryTokenizeOperator(runes[i:], lineNum, i+1, fileName); consumed > 0 {
			tokens = append(tokens, token)
			i += consumed
		} else {
			return nil, fmt.Errorf("unexpected character '%c' at %s:%d:%d", char, fileName, lineNum, i+1)
		}
	}

	return tokens, nil
}

// tryTokenizeString 尝试识别字符串字面量
func (t *SimpleTokenizer) tryTokenizeString(runes []rune, lineNum, column int, fileName string) (Token, int) {
	if len(runes) == 0 || runes[0] != '"' {
		return Token{}, 0
	}

	i := 1
	for i < len(runes) && runes[i] != '"' {
		if runes[i] == '\\' && i+1 < len(runes) {
			i++ // 跳过转义字符
		}
		i++
	}

	if i >= len(runes) {
		// 未闭合的字符串
		return Token{}, 0
	}

	value := string(runes[1:i])
	return NewToken(StringLiteral, value, NewPosition(lineNum, column, fileName)), i + 1
}

// tryTokenizeNumber 尝试识别数字字面量
func (t *SimpleTokenizer) tryTokenizeNumber(runes []rune, lineNum, column int, fileName string) (Token, int) {
	if len(runes) == 0 || !unicode.IsDigit(runes[0]) {
		return Token{}, 0
	}

	i := 0
	hasDot := false

	for i < len(runes) {
		char := runes[i]
		if unicode.IsDigit(char) {
			i++
		} else if char == '.' && !hasDot {
			hasDot = true
			i++
		} else {
			break
		}
	}

	value := string(runes[:i])
	if hasDot {
		return NewToken(FloatLiteral, value, NewPosition(lineNum, column, fileName)), i
	}
	return NewToken(IntLiteral, value, NewPosition(lineNum, column, fileName)), i
}

// tryTokenizeIdentifierOrKeyword 尝试识别标识符或关键字
func (t *SimpleTokenizer) tryTokenizeIdentifierOrKeyword(runes []rune, lineNum, column int, fileName string) (Token, int) {
	if len(runes) == 0 || (!unicode.IsLetter(runes[0]) && runes[0] != '_') {
		return Token{}, 0
	}

	i := 0
	for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_') {
		i++
	}

	value := string(runes[:i])

	// 检查是否为关键字
	if kind := t.getKeywordKind(value); kind != Identifier {
		return NewToken(kind, value, NewPosition(lineNum, column, fileName)), i
	}

	return NewToken(Identifier, value, NewPosition(lineNum, column, fileName)), i
}

// tryTokenizeOperator 尝试识别操作符
func (t *SimpleTokenizer) tryTokenizeOperator(runes []rune, lineNum, column int, fileName string) (Token, int) {
	if len(runes) == 0 {
		return Token{}, 0
	}

	// 双字符操作符
	if len(runes) >= 2 {
		twoChars := string(runes[:2])
		if kind := t.getOperatorKind(twoChars); kind != 0 {
			return NewToken(kind, twoChars, NewPosition(lineNum, column, fileName)), 2
		}
	}

	// 单字符操作符
	oneChar := string(runes[0])
	if kind := t.getOperatorKind(oneChar); kind != 0 {
		return NewToken(kind, oneChar, NewPosition(lineNum, column, fileName)), 1
	}

	return Token{}, 0
}

// getKeywordKind 返回关键字对应的TokenKind
func (t *SimpleTokenizer) getKeywordKind(word string) TokenKind {
	switch word {
	case "func":
		return Function
	case "if":
		return If
	case "else":
		return Else
	case "for":
		return For
	case "while":
		return While
	case "return":
		return Return
	case "struct":
		return Struct
	case "enum":
		return Enum
	case "trait":
		return Trait
	case "impl":
		return Impl
	case "print":
		return Print
	case "let":
		return Let
	case "match":
		return Match
	case "async":
		return Async
	case "await":
		return Await
	case "chan":
		return Chan
	case "spawn":
		return Spawn
	case "future":
		return Future
	case "true", "false":
		return BoolLiteral
	default:
		return Identifier
	}
}

// getOperatorKind 返回操作符对应的TokenKind
func (t *SimpleTokenizer) getOperatorKind(op string) TokenKind {
	switch op {
	case "+":
		return Plus
	case "-":
		return Minus
	case "*":
		return Multiply
	case "/":
		return Divide
	case "=":
		return Assign
	case "==":
		return Equal
	case "!=":
		return NotEqual
	case "<":
		return Less
	case ">":
		return Greater
	case "<=":
		return LessEqual
	case ">=":
		return GreaterEqual
	case "!":
		return Not
	case "&&":
		return And
	case "||":
		return Or
	case ".":
		return Dot
	case ",":
		return Comma
	case ";":
		return Semicolon
	case ":":
		return Colon
	case "(":
		return LeftParen
	case ")":
		return RightParen
	case "{":
		return LeftBrace
	case "}":
		return RightBrace
	case "[":
		return LeftBracket
	case "]":
		return RightBracket
	case "=>":
		return FatArrow
	default:
		return 0
	}
}
