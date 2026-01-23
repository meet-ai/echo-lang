// Package services 定义语法分析上下文的领域服务
package services

import (
	"context"
	"fmt"

	"echo/internal/modules/frontend/domain/shared/value_objects"
)

// SyntaxAnalyzer 语法分析器领域服务
// 负责将Token流转换为AST，实现语法分析的核心逻辑
type SyntaxAnalyzer struct {
	tokenStream *value_objects.TokenStream
	position    int
	errors      []*value_objects.ParseError
}

// NewSyntaxAnalyzer 创建新的语法分析器
func NewSyntaxAnalyzer() *SyntaxAnalyzer {
	return &SyntaxAnalyzer{
		position: 0,
		errors:   make([]*value_objects.ParseError, 0),
	}
}

// ParseAST 解析Token流为抽象语法树（实现SyntaxAnalysisService接口）
func (sa *SyntaxAnalyzer) ParseAST(ctx context.Context, tokenStream *value_objects.TokenStream) (*value_objects.ProgramAST, error) {
	sa.tokenStream = tokenStream
	sa.position = 0
	sa.errors = make([]*value_objects.ParseError, 0)

	// 创建程序AST
	nodes := make([]value_objects.ASTNode, 0)
	filename := "unknown" // TODO: 从tokenStream获取文件名
	programAST := value_objects.NewProgramAST(
		nodes,
		filename,
		value_objects.NewSourceLocation(filename, 1, 1, 0),
	)

	// 解析程序体
	for !sa.isAtEnd() {
		// 跳过空行和注释（已在词法分析中处理）
		if sa.currentToken().Type() == value_objects.TokenTypeEOF {
			break
		}

		// 解析顶级声明或语句
		node, err := sa.parseTopLevel()
		if err != nil {
			sa.addError(err.Error(), sa.currentToken().Location())
			sa.advance() // 跳过错误Token，继续解析
			continue
		}

		if node != nil {
			nodes = append(nodes, node)
		}
	}

	// 如果有错误，返回第一个错误
	if len(sa.errors) > 0 {
		return nil, sa.errors[0]
	}

	return programAST, nil
}

// parseTopLevel 解析顶级声明或语句
func (sa *SyntaxAnalyzer) parseTopLevel() (value_objects.ASTNode, error) {
	token := sa.currentToken()

	switch token.Type() {
	case value_objects.TokenTypeFunc:
		return sa.parseFunctionDeclaration()
	case value_objects.TokenTypeAsync:
		return sa.parseAsyncFunctionDeclaration()
	case value_objects.TokenTypeStruct:
		return sa.parseStructDeclaration()
	case value_objects.TokenTypeEnum:
		return sa.parseEnumDeclaration()
	case value_objects.TokenTypeTrait:
		return sa.parseTraitDeclaration()
	case value_objects.TokenTypeImpl:
		return sa.parseImplDeclaration()
	default:
		// 其他情况当作语句处理
		return sa.parseStatement()
	}
}

// parseFunctionDeclaration 解析函数声明
func (sa *SyntaxAnalyzer) parseFunctionDeclaration() (value_objects.ASTNode, error) {
	// 消耗 'func' 关键字
	sa.advance()

	// 解析函数名
	if !sa.match(value_objects.TokenTypeIdentifier) {
		return nil, fmt.Errorf("expected function name after 'func'")
	}
	funcName := sa.previous().Lexeme()

	// 解析参数列表 (简化实现)
	if !sa.match(value_objects.TokenTypeLeftParen) {
		return nil, fmt.Errorf("expected '(' after function name")
	}

	parameters := make([]*value_objects.Parameter, 0)
	// 简化：跳过参数解析，直接找到右括号
	for !sa.isAtEnd() && !sa.check(value_objects.TokenTypeRightParen) {
		sa.advance()
	}

	if !sa.match(value_objects.TokenTypeRightParen) {
		return nil, fmt.Errorf("expected ')' after parameters")
	}

	// 解析返回类型 (简化实现)
	var returnTypeAnnotation *value_objects.TypeAnnotation
	if sa.match(value_objects.TokenTypeArrow) {
		if !sa.match(value_objects.TokenTypeIdentifier) {
			return nil, fmt.Errorf("expected return type after '->'")
		}
		returnTypeStr := sa.previous().Lexeme()
		returnTypeLocation := sa.previous().Location()
		returnTypeAnnotation = value_objects.NewTypeAnnotation(returnTypeStr, nil, false, returnTypeLocation)
	}

	// 解析函数体 (简化实现)
	if !sa.match(value_objects.TokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' before function body")
	}
	bodyStartLocation := sa.previous().Location()

	bodyNodes := sa.parseBlock()
	bodyBlock := value_objects.NewBlockStatement(bodyNodes, bodyStartLocation)

	if !sa.match(value_objects.TokenTypeRightBrace) {
		return nil, fmt.Errorf("expected '}' after function body")
	}

	// 使用函数名的位置作为函数声明的位置
	funcLocation := sa.previous().Location()
	return value_objects.NewFunctionDeclaration(
		funcName,
		nil, // genericParams - 暂时设为nil
		parameters,
		returnTypeAnnotation,
		bodyBlock,
		funcLocation,
	), nil
}

// parseAsyncFunctionDeclaration 解析异步函数声明
func (sa *SyntaxAnalyzer) parseAsyncFunctionDeclaration() (value_objects.ASTNode, error) {
	// 消耗 'async' 关键字
	sa.advance()

	// 其余逻辑与普通函数相同
	funcDecl, err := sa.parseFunctionDeclaration()
	if err != nil {
		return nil, err
	}

	// 标记为异步函数
	if asyncFunc, ok := funcDecl.(*value_objects.FunctionDeclaration); ok {
		// 使用'async'关键字的位置作为异步函数声明的位置
		asyncLocation := sa.previous().Location() // 这是函数名的位置，我们需要'async'的位置
		// 由于已经消耗了'async'，我们使用函数的位置
		asyncLocation = asyncFunc.Location()
		return value_objects.NewAsyncFunctionDeclaration(asyncFunc, asyncLocation), nil
	}

	return funcDecl, nil
}

// parseStructDeclaration 解析结构体声明
func (sa *SyntaxAnalyzer) parseStructDeclaration() (value_objects.ASTNode, error) {
	// 简化实现：暂时跳过结构体声明
	sa.advance() // 跳过 'struct'
	for !sa.isAtEnd() && !sa.check(value_objects.TokenTypeLeftBrace) {
		sa.advance()
	}
	sa.skipBlock()
	return nil, nil // 暂时返回nil
}

// parseEnumDeclaration 解析枚举声明
func (sa *SyntaxAnalyzer) parseEnumDeclaration() (value_objects.ASTNode, error) {
	// 简化实现：暂时跳过枚举声明
	sa.advance() // 跳过 'enum'
	for !sa.isAtEnd() && !sa.check(value_objects.TokenTypeLeftBrace) {
		sa.advance()
	}
	sa.skipBlock()
	return nil, nil
}

// parseTraitDeclaration 解析trait声明
func (sa *SyntaxAnalyzer) parseTraitDeclaration() (value_objects.ASTNode, error) {
	// 简化实现：暂时跳过trait声明
	sa.advance() // 跳过 'trait'
	for !sa.isAtEnd() && !sa.check(value_objects.TokenTypeLeftBrace) {
		sa.advance()
	}
	sa.skipBlock()
	return nil, nil
}

// parseImplDeclaration 解析impl声明
func (sa *SyntaxAnalyzer) parseImplDeclaration() (value_objects.ASTNode, error) {
	// 简化实现：暂时跳过impl声明
	sa.advance() // 跳过 'impl'
	for !sa.isAtEnd() && !sa.check(value_objects.TokenTypeLeftBrace) {
		sa.advance()
	}
	sa.skipBlock()
	return nil, nil
}

// skipBlock 跳过代码块
func (sa *SyntaxAnalyzer) skipBlock() {
	braceCount := 0
	if sa.match(value_objects.TokenTypeLeftBrace) {
		braceCount++
	}

	for !sa.isAtEnd() && braceCount > 0 {
		if sa.match(value_objects.TokenTypeLeftBrace) {
			braceCount++
		} else if sa.match(value_objects.TokenTypeRightBrace) {
			braceCount--
		} else {
			sa.advance()
		}
	}
}

// parseStatement 解析语句
func (sa *SyntaxAnalyzer) parseStatement() (value_objects.ASTNode, error) {
	token := sa.currentToken()

	switch token.Type() {
	case value_objects.TokenTypeLet:
		return sa.parseVariableDeclaration()
	case value_objects.TokenTypeIf:
		return sa.parseIfStatement()
	case value_objects.TokenTypeWhile:
		return sa.parseWhileStatement()
	case value_objects.TokenTypeFor:
		return sa.parseForStatement()
	case value_objects.TokenTypeReturn:
		return sa.parseReturnStatement()
	case value_objects.TokenTypePrint:
		return sa.parsePrintStatement()
	default:
		// 表达式语句
		return sa.parseExpressionStatement()
	}
}

// parseIfStatement 解析if语句
func (sa *SyntaxAnalyzer) parseIfStatement() (value_objects.ASTNode, error) {
	// 简化实现：暂时跳过if语句
	sa.advance() // 跳过 'if'
	// 跳过条件和代码块
	for !sa.isAtEnd() && !sa.check(value_objects.TokenTypeLeftBrace) {
		sa.advance()
	}
	sa.skipBlock()
	return nil, nil
}

// parseWhileStatement 解析while语句
func (sa *SyntaxAnalyzer) parseWhileStatement() (value_objects.ASTNode, error) {
	// 简化实现：暂时跳过while语句
	sa.advance() // 跳过 'while'
	for !sa.isAtEnd() && !sa.check(value_objects.TokenTypeLeftBrace) {
		sa.advance()
	}
	sa.skipBlock()
	return nil, nil
}

// parseForStatement 解析for语句
func (sa *SyntaxAnalyzer) parseForStatement() (value_objects.ASTNode, error) {
	// 简化实现：暂时跳过for语句
	sa.advance() // 跳过 'for'
	for !sa.isAtEnd() && !sa.check(value_objects.TokenTypeLeftBrace) {
		sa.advance()
	}
	sa.skipBlock()
	return nil, nil
}

// parseReturnStatement 解析return语句
func (sa *SyntaxAnalyzer) parseReturnStatement() (value_objects.ASTNode, error) {
	// 简化实现：暂时跳过return语句
	sa.advance() // 跳过 'return'
	for !sa.isAtEnd() && !sa.check(value_objects.TokenTypeSemicolon) {
		sa.advance()
	}
	return nil, nil
}

// parsePrintStatement 解析print语句
func (sa *SyntaxAnalyzer) parsePrintStatement() (value_objects.ASTNode, error) {
	// 简化实现：暂时跳过print语句
	sa.advance() // 跳过 'print'
	for !sa.isAtEnd() && !sa.check(value_objects.TokenTypeSemicolon) {
		sa.advance()
	}
	return nil, nil
}

// parseExpressionStatement 解析表达式语句
func (sa *SyntaxAnalyzer) parseExpressionStatement() (value_objects.ASTNode, error) {
	expr, err := sa.parseExpression()
	if err != nil {
		return nil, err
	}
	return expr, nil
}

// parseVariableDeclaration 解析变量声明
func (sa *SyntaxAnalyzer) parseVariableDeclaration() (value_objects.ASTNode, error) {
	// 消耗 'let' 关键字
	sa.advance()

	if !sa.match(value_objects.TokenTypeIdentifier) {
		return nil, fmt.Errorf("expected variable name after 'let'")
	}
	varName := sa.previous().Lexeme()

	if !sa.match(value_objects.TokenTypeColon) {
		return nil, fmt.Errorf("expected ':' after variable name")
	}

	if !sa.match(value_objects.TokenTypeIdentifier) {
		return nil, fmt.Errorf("expected variable type")
	}
	varTypeStr := sa.previous().Lexeme()
	varTypeLocation := sa.previous().Location()
	varTypeAnnotation := value_objects.NewTypeAnnotation(varTypeStr, nil, false, varTypeLocation)

	var initializer value_objects.ASTNode
	if sa.match(value_objects.TokenTypeAssign) {
		expr, err := sa.parseExpression()
		if err != nil {
			return nil, err
		}
		initializer = expr
	}

	// 使用变量名的位置作为变量声明的位置
	varLocation := sa.previous().Location() // 这是变量名的位置
	return value_objects.NewVariableDeclaration(
		varName,
		varTypeAnnotation,
		initializer,
		varLocation,
	), nil
}

// parseExpression 解析表达式
func (sa *SyntaxAnalyzer) parseExpression() (value_objects.ASTNode, error) {
	return sa.parseBinaryExpression()
}

// parseBinaryExpression 解析二元表达式
func (sa *SyntaxAnalyzer) parseBinaryExpression() (value_objects.ASTNode, error) {
	left, err := sa.parseUnaryExpression()
	if err != nil {
		return nil, err
	}

	for sa.matchBinaryOperator() {
		operator := sa.previous().Lexeme()
		right, err := sa.parseUnaryExpression()
		if err != nil {
			return nil, err
		}

		// 使用左操作数的位置作为二元表达式的起始位置
		expressionLocation := left.Location()
		left = value_objects.NewBinaryExpression(
			left,
			operator,
			right,
			expressionLocation,
		)
	}

	return left, nil
}

// parseUnaryExpression 解析一元表达式
func (sa *SyntaxAnalyzer) parseUnaryExpression() (value_objects.ASTNode, error) {
	if sa.match(value_objects.TokenTypeMinus) || sa.match(value_objects.TokenTypeNot) {
		operator := sa.previous().Lexeme()
		operatorLocation := sa.previous().Location()
		operand, err := sa.parseUnaryExpression()
		if err != nil {
			return nil, err
		}

		// 使用运算符的位置作为一元表达式的起始位置
		return value_objects.NewUnaryExpression(
			operator,
			operand,
			operatorLocation,
		), nil
	}

	return sa.parsePrimaryExpression()
}

// parsePrimaryExpression 解析基本表达式
func (sa *SyntaxAnalyzer) parsePrimaryExpression() (value_objects.ASTNode, error) {
	token := sa.currentToken()

	switch token.Type() {
	case value_objects.TokenTypeIdentifier:
		sa.advance()
		return value_objects.NewIdentifier(token.Lexeme(), token.Location()), nil

	case value_objects.TokenTypeInt:
		sa.advance()
		return value_objects.NewIntegerLiteral(token.Value().(int64), token.Location()), nil

	case value_objects.TokenTypeFloat:
		sa.advance()
		return value_objects.NewFloatLiteral(token.Value().(float64), token.Location()), nil

	case value_objects.TokenTypeString:
		sa.advance()
		return value_objects.NewStringLiteral(token.Value().(string), token.Location()), nil

	case value_objects.TokenTypeBool:
		sa.advance()
		value := token.Lexeme() == "true"
		return value_objects.NewBooleanLiteral(value, token.Location()), nil

	case value_objects.TokenTypeLeftParen:
		sa.advance()
		expr, err := sa.parseExpression()
		if err != nil {
			return nil, err
		}
		if !sa.match(value_objects.TokenTypeRightParen) {
			return nil, fmt.Errorf("expected ')' after expression")
		}
		return expr, nil

	default:
		return nil, fmt.Errorf("unexpected token: %s", token.Lexeme())
	}
}

// parseBlock 解析代码块
func (sa *SyntaxAnalyzer) parseBlock() []value_objects.ASTNode {
	statements := make([]value_objects.ASTNode, 0)

	for !sa.check(value_objects.TokenTypeRightBrace) && !sa.isAtEnd() {
		stmt, err := sa.parseStatement()
		if err != nil {
			sa.addError(err.Error(), sa.currentToken().Location())
			sa.advance() // 跳过错误Token
			continue
		}
		statements = append(statements, stmt)
	}

	return statements
}

// 辅助方法

// currentToken 获取当前Token
func (sa *SyntaxAnalyzer) currentToken() *value_objects.Token {
	return sa.tokenStream.Current()
}

// previous 获取前一个Token
func (sa *SyntaxAnalyzer) previous() *value_objects.Token {
	return sa.tokenStream.Tokens()[sa.position-1]
}

// advance 前进到下一个Token
func (sa *SyntaxAnalyzer) advance() {
	if !sa.isAtEnd() {
		sa.position++
		sa.tokenStream.Next()
	}
}

// match 检查并消耗指定类型的Token
func (sa *SyntaxAnalyzer) match(tokenType value_objects.TokenType) bool {
	if sa.check(tokenType) {
		sa.advance()
		return true
	}
	return false
}

// check 检查当前Token是否为指定类型
func (sa *SyntaxAnalyzer) check(tokenType value_objects.TokenType) bool {
	if sa.isAtEnd() {
		return false
	}
	return sa.currentToken().Type() == tokenType
}

// isAtEnd 检查是否到达Token流末尾
func (sa *SyntaxAnalyzer) isAtEnd() bool {
	return sa.position >= sa.tokenStream.Count() ||
		   sa.currentToken().Type() == value_objects.TokenTypeEOF
}

// matchBinaryOperator 检查并消耗二元运算符
func (sa *SyntaxAnalyzer) matchBinaryOperator() bool {
	if sa.isAtEnd() {
		return false
	}

	tokenType := sa.currentToken().Type()
	switch tokenType {
	case value_objects.TokenTypePlus, value_objects.TokenTypeMinus,
		 value_objects.TokenTypeMultiply, value_objects.TokenTypeDivide,
		 value_objects.TokenTypeModulo, value_objects.TokenTypeEqual,
		 value_objects.TokenTypeNotEqual, value_objects.TokenTypeLessThan,
		 value_objects.TokenTypeGreaterThan, value_objects.TokenTypeLessEqual,
		 value_objects.TokenTypeGreaterEqual, value_objects.TokenTypeAnd,
		 value_objects.TokenTypeOr:
		sa.advance()
		return true
	default:
		return false
	}
}

// addError 添加错误
func (sa *SyntaxAnalyzer) addError(message string, location value_objects.SourceLocation) {
	error := value_objects.NewParseError(message, location, value_objects.ErrorTypeSyntax, value_objects.SeverityError)
	sa.errors = append(sa.errors, error)
}
