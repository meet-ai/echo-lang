package parser

import (
	"fmt"

	"echo/internal/modules/frontend/domain/entities"
	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
)

// TokenBasedExpressionParser 基于 Token 的表达式解析器
// 使用 Pratt Parser 算法处理运算符优先级
// 直接使用 EnhancedTokenStream 管理位置，不维护自己的 position
type TokenBasedExpressionParser struct {
	tokenStream *lexicalVO.EnhancedTokenStream
}

// NewTokenBasedExpressionParser 创建新的基于 Token 的表达式解析器
func NewTokenBasedExpressionParser() *TokenBasedExpressionParser {
	return &TokenBasedExpressionParser{}
}

// ParseExpression 解析表达式（主入口）
// 直接使用 tokenStream 的当前位置，解析完成后 tokenStream 的位置会自动更新
func (ep *TokenBasedExpressionParser) ParseExpression(tokenStream *lexicalVO.EnhancedTokenStream) (entities.Expr, error) {
	ep.tokenStream = tokenStream

	if tokenStream.IsAtEnd() {
		return nil, fmt.Errorf("unexpected end of token stream")
	}

	// 检查当前 token 是否是语句结束符
	currentToken := tokenStream.Current()
	if currentToken != nil {
		// 临时创建一个表达式解析器来检查是否是语句结束符
		tempParser := &TokenBasedExpressionParser{}
		tempParser.tokenStream = tokenStream
		if tempParser.isStatementTerminator(currentToken.Type()) {
			return nil, fmt.Errorf("expression cannot start with statement terminator: %s", currentToken.Type())
		}
	}

	// 使用 Pratt Parser 算法，从最低优先级开始
	return ep.parseExpression(0)
}

// parseExpression 使用 Pratt Parser 算法解析表达式
// minPrecedence: 最小优先级，用于控制递归深度
func (ep *TokenBasedExpressionParser) parseExpression(minPrecedence int) (entities.Expr, error) {
	// 检查当前 token 是否是语句终止符
	// 如果是，说明表达式已经结束（可能在递归调用中）
	if !ep.tokenStream.IsAtEnd() {
		currentToken := ep.tokenStream.Current()
		if currentToken != nil && ep.isStatementTerminator(currentToken.Type()) {
			return nil, fmt.Errorf("expression ended at statement terminator: %s", currentToken.Type())
		}
	}

	// 步骤1：解析前缀表达式（字面量、标识符、括号、一元运算符等）
	// 注意：语句结束符检查在 shouldContinueParsing 中进行
	left, err := ep.parsePrefixExpression()
	if err != nil {
		return nil, err
	}

	// 步骤2：循环解析中缀表达式（二元运算符）
	for !ep.tokenStream.IsAtEnd() && ep.shouldContinueParsing(minPrecedence) {
		operatorToken := ep.tokenStream.Current()

		// 获取运算符优先级
		precedence := ep.getOperatorPrecedence(operatorToken.Type())
		if precedence < minPrecedence {
			break
		}

		// 消耗运算符 token
		ep.tokenStream.Next()

		// 递归解析右侧表达式（考虑结合性）
		right, err := ep.parseExpression(precedence + 1) // 左结合：+1
		if err != nil {
			return nil, err
		}

		// 构建二元表达式
		left = &entities.BinaryExpr{
			Left:  left,
			Op:    ep.tokenTypeToOperator(operatorToken.Type()),
			Right: right,
		}
	}

	return left, nil
}

// parsePrefixExpression 解析前缀表达式
func (ep *TokenBasedExpressionParser) parsePrefixExpression() (entities.Expr, error) {
	token := ep.tokenStream.Current()
	if token == nil {
		return nil, fmt.Errorf("unexpected end of token stream")
	}

	// 如果遇到语句结束符，说明表达式已经结束
	// 这不应该发生，因为调用者应该已经检查了，或者 shouldContinueParsing 应该已经返回 false
	// 但为了安全，我们在这里检查
	if ep.isStatementTerminator(token.Type()) {
		return nil, fmt.Errorf("expression ended at statement terminator: %s (this should have been checked by the caller or shouldContinueParsing)", token.Type())
	}

	switch token.Type() {
	// 字面量
	case lexicalVO.EnhancedTokenTypeNumber:
		ep.tokenStream.Next()
		return ep.createNumberLiteral(token)
	case lexicalVO.EnhancedTokenTypeString:
		ep.tokenStream.Next()
		return ep.createStringLiteral(token)
	case lexicalVO.EnhancedTokenTypeBool:
		ep.tokenStream.Next()
		return ep.createBoolLiteral(token)

	// 标识符
	case lexicalVO.EnhancedTokenTypeIdentifier:
		ep.tokenStream.Next()
		return ep.parseIdentifierOrCall(token)

	// 数组字面量
	case lexicalVO.EnhancedTokenTypeLeftBracket:
		return ep.parseArrayLiteral()

	// 结构体字面量
	case lexicalVO.EnhancedTokenTypeLeftBrace:
		return ep.parseStructLiteral()

	// 括号表达式
	case lexicalVO.EnhancedTokenTypeLeftParen:
		ep.tokenStream.Next() // 消耗 '('
		expr, err := ep.parseExpression(0)
		if err != nil {
			return nil, err
		}
		if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
			return nil, fmt.Errorf("expected ')' after expression")
		}
		ep.tokenStream.Next() // 消耗 ')'
		return expr, nil

	// 一元运算符
	case lexicalVO.EnhancedTokenTypeMinus:
		ep.tokenStream.Next()
		right, err := ep.parsePrefixExpression()
		if err != nil {
			return nil, err
		}
		return &entities.BinaryExpr{
			Left:  &entities.IntLiteral{Value: 0},
			Op:    "-",
			Right: right,
		}, nil
	case lexicalVO.EnhancedTokenTypeNot:
		ep.tokenStream.Next()
		right, err := ep.parsePrefixExpression()
		if err != nil {
			return nil, err
		}
		return &entities.BinaryExpr{
			Left:  &entities.BoolLiteral{Value: false},
			Op:    "!",
			Right: right,
		}, nil

	// 特殊表达式
	case lexicalVO.EnhancedTokenTypeMatch:
		return ep.parseMatchExpression()
	case lexicalVO.EnhancedTokenTypeAsync:
		// async 关键字可能用于 await 表达式，但当前实现中 await 是作为标识符处理的
		// 暂时跳过，await 表达式应该通过标识符解析
		return nil, fmt.Errorf("async keyword not yet supported in expression context")
	case lexicalVO.EnhancedTokenTypeSpawn:
		return ep.parseSpawnExpression()
	case lexicalVO.EnhancedTokenTypeChan:
		// chan 关键字用于创建通道，格式：chan TypeName
		ep.tokenStream.Next() // 消耗 'chan'
		typeToken := ep.tokenStream.Current()
		if typeToken == nil || typeToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
			return nil, fmt.Errorf("expected type name after 'chan'")
		}
		typeName := typeToken.Lexeme()
		ep.tokenStream.Next() // 消耗类型名
		// 创建通道表达式（暂时作为标识符处理，后续可以创建专门的通道表达式节点）
		return &entities.Identifier{Name: "chan " + typeName}, nil
	case lexicalVO.EnhancedTokenTypeChannelReceive:
		// <- 通道接收运算符，格式：<- channel
		ep.tokenStream.Next() // 消耗 '<-'
		// 解析通道表达式（右侧操作数）
		channelExpr, err := ep.parseExpression(0) // 通道接收是一元运算符，优先级很高
		if err != nil {
			return nil, fmt.Errorf("failed to parse channel expression after '<-': %w", err)
		}
		// 创建通道接收表达式（暂时作为函数调用处理，后续可以创建专门的通道接收表达式节点）
		return &entities.FuncCall{
			Name: "<-",
			Args: []entities.Expr{channelExpr},
		}, nil

	// 语句关键字不应该出现在表达式上下文中
	// 这些 token 应该被识别为表达式结束符，而不是错误
	// 如果在这里遇到，说明调用者没有正确检查，或者 token stream 位置不对
	case lexicalVO.EnhancedTokenTypeReturn,
		lexicalVO.EnhancedTokenTypeIf,
		lexicalVO.EnhancedTokenTypeWhile,
		lexicalVO.EnhancedTokenTypeFor,
		lexicalVO.EnhancedTokenTypeLet,
		lexicalVO.EnhancedTokenTypeFunc,
		lexicalVO.EnhancedTokenTypePrint:
		// 这些是语句关键字，不应该出现在表达式上下文中
		// 如果遇到，说明表达式已经结束，或者 token stream 位置不对
		// 返回一个特殊的错误，让调用者知道表达式已经结束
		return nil, fmt.Errorf("expression ended at statement keyword: %s (this should have been checked by the caller)", token.Type())

	default:
		// 检查是否是运算符（中缀运算符不应该出现在前缀表达式位置）
		precedence := ep.getOperatorPrecedence(token.Type())
		if precedence > 0 {
			return nil, fmt.Errorf("unexpected infix operator in prefix position: %s (this token should be handled as an infix operator)", token.Type())
		}
		return nil, fmt.Errorf("unexpected token: %s", token.Type())
	}
}

// parseIdentifierOrCall 解析标识符或函数调用或结构体访问
func (ep *TokenBasedExpressionParser) parseIdentifierOrCall(identifierToken *lexicalVO.EnhancedToken) (entities.Expr, error) {
	// 注意：identifierToken 已经被消耗，当前 token stream 位置指向下一个 token
	// 检查当前 token（即下一个 token）
	nextToken := ep.tokenStream.Current()
	if nextToken == nil {
		// 没有下一个 token，是普通标识符
		return &entities.Identifier{
			Name: identifierToken.Lexeme(),
		}, nil
	}

	if nextToken.Type() == lexicalVO.EnhancedTokenTypeLeftParen {
		// 函数调用
		return ep.parseFunctionCall(identifierToken)
	} else if nextToken.Type() == lexicalVO.EnhancedTokenTypeDot {
		// 结构体访问或方法调用
		return ep.parseStructAccessOrMethodCall(identifierToken)
	} else if nextToken.Type() == lexicalVO.EnhancedTokenTypeLeftBracket {
		// 数组索引访问
		return ep.parseIndexAccess(identifierToken)
	}

	// 否则是普通标识符
	return &entities.Identifier{
		Name: identifierToken.Lexeme(),
	}, nil
}

// parseStructAccessOrMethodCall 解析结构体访问或方法调用
func (ep *TokenBasedExpressionParser) parseStructAccessOrMethodCall(objectToken *lexicalVO.EnhancedToken) (entities.Expr, error) {
	object := &entities.Identifier{Name: objectToken.Lexeme()}
	ep.tokenStream.Next() // 消耗 '.'

	fieldToken := ep.tokenStream.Current()
	if fieldToken == nil || fieldToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
		return nil, fmt.Errorf("expected identifier after '.'")
	}
	ep.tokenStream.Next() // 消耗字段名

	// 检查是否是方法调用
	if ep.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
		return ep.parseMethodCall(object, fieldToken)
	}

	// 否则是结构体字段访问
	return &entities.StructAccess{
		Object: object,
		Field:  fieldToken.Lexeme(),
	}, nil
}

// parseMethodCall 解析方法调用
func (ep *TokenBasedExpressionParser) parseMethodCall(object entities.Expr, methodToken *lexicalVO.EnhancedToken) (entities.Expr, error) {
	ep.tokenStream.Next() // 消耗 '('

	var args []entities.Expr
	if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
		for {
			arg, err := ep.parseExpression(0)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)

			if ep.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
				break
			}
			if !ep.checkToken(lexicalVO.EnhancedTokenTypeComma) {
				return nil, fmt.Errorf("expected ',' or ')' in method call")
			}
			ep.tokenStream.Next() // 消耗 ','
		}
	}

	ep.tokenStream.Next() // 消耗 ')'
	return &entities.MethodCallExpr{
		Receiver:   object,
		MethodName: methodToken.Lexeme(),
		Args:       args,
	}, nil
}

// parseIndexAccess 解析数组索引访问
func (ep *TokenBasedExpressionParser) parseIndexAccess(arrayToken *lexicalVO.EnhancedToken) (entities.Expr, error) {
	array := &entities.Identifier{Name: arrayToken.Lexeme()}
	ep.tokenStream.Next() // 消耗 '['

	index, err := ep.parseExpression(0)
	if err != nil {
		return nil, err
	}

	if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
		return nil, fmt.Errorf("expected ']' after index")
	}
	ep.tokenStream.Next() // 消耗 ']'

	return &entities.IndexExpr{
		Array: array,
		Index: index,
	}, nil
}

// parseArrayLiteral 解析数组字面量
func (ep *TokenBasedExpressionParser) parseArrayLiteral() (entities.Expr, error) {
	ep.tokenStream.Next() // 消耗 '['

	var elements []entities.Expr
	if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
		for {
			element, err := ep.parseExpression(0)
			if err != nil {
				return nil, err
			}
			elements = append(elements, element)

			if ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
				break
			}
			if !ep.checkToken(lexicalVO.EnhancedTokenTypeComma) {
				return nil, fmt.Errorf("expected ',' or ']' in array literal")
			}
			ep.tokenStream.Next() // 消耗 ','
		}
	}

	ep.tokenStream.Next() // 消耗 ']'
	return &entities.ArrayLiteral{
		Elements: elements,
	}, nil
}

// parseStructLiteral 解析结构体字面量
func (ep *TokenBasedExpressionParser) parseStructLiteral() (entities.Expr, error) {
	ep.tokenStream.Next() // 消耗 '{'

	fields := make(map[string]entities.Expr)
	if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
		for {
			// 解析字段名
			fieldToken := ep.tokenStream.Current()
			if fieldToken == nil || fieldToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
				return nil, fmt.Errorf("expected field name in struct literal")
			}
			fieldName := fieldToken.Lexeme()
			ep.tokenStream.Next() // 消耗字段名

			// 期望 ':'
			if !ep.checkToken(lexicalVO.EnhancedTokenTypeColon) {
				return nil, fmt.Errorf("expected ':' after field name")
			}
			ep.tokenStream.Next() // 消耗 ':'

			// 解析字段值
			fieldValue, err := ep.parseExpression(0)
			if err != nil {
				return nil, err
			}
			fields[fieldName] = fieldValue

			if ep.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
				break
			}
			if !ep.checkToken(lexicalVO.EnhancedTokenTypeComma) {
				return nil, fmt.Errorf("expected ',' or '}' in struct literal")
			}
			ep.tokenStream.Next() // 消耗 ','
		}
	}

	ep.tokenStream.Next() // 消耗 '}'
	// 注意：这里需要知道结构体类型名，但当前无法从字面量推断
	// 暂时返回一个通用的结构体字面量
	return &entities.StructLiteral{
		Type:   "", // 需要从上下文推断
		Fields: fields,
	}, nil
}

// parseFunctionCall 解析函数调用
func (ep *TokenBasedExpressionParser) parseFunctionCall(nameToken *lexicalVO.EnhancedToken) (entities.Expr, error) {
	ep.tokenStream.Next() // 消耗 '('

	var args []entities.Expr
	if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
		// 解析参数列表
		for {
			arg, err := ep.parseExpression(0)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)

			if ep.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
				break
			}
			if !ep.checkToken(lexicalVO.EnhancedTokenTypeComma) {
				currentToken := ep.tokenStream.Current()
				return nil, fmt.Errorf("expected ',' or ')' in function call (got %s)", currentToken.Type())
			}
			ep.tokenStream.Next() // 消耗 ','
		}
	}

	ep.tokenStream.Next() // 消耗 ')'

	return &entities.FuncCall{
		Name: nameToken.Lexeme(),
		Args: args,
	}, nil
}

// createNumberLiteral 创建数字字面量
func (ep *TokenBasedExpressionParser) createNumberLiteral(token *lexicalVO.EnhancedToken) (entities.Expr, error) {
	numLit := token.NumberLiteral()
	if numLit == nil {
		return nil, fmt.Errorf("token is not a number literal")
	}

	// 根据数字类型返回相应的字面量
	if numLit.IsInteger() {
		return &entities.IntLiteral{
			Value: int(numLit.IntValue()), // 转换为 int
		}, nil
	} else {
		return &entities.FloatLiteral{
			Value: numLit.FloatValue(),
		}, nil
	}
}

// createStringLiteral 创建字符串字面量
func (ep *TokenBasedExpressionParser) createStringLiteral(token *lexicalVO.EnhancedToken) (entities.Expr, error) {
	return &entities.StringLiteral{
		Value: token.StringValue(),
	}, nil
}

// createBoolLiteral 创建布尔字面量
func (ep *TokenBasedExpressionParser) createBoolLiteral(token *lexicalVO.EnhancedToken) (entities.Expr, error) {
	boolVal := token.BoolValue()
	if boolVal == nil {
		return nil, fmt.Errorf("token is not a bool literal")
	}
	return &entities.BoolLiteral{
		Value: *boolVal,
	}, nil
}

// parseMatchExpression 解析 match 表达式
func (ep *TokenBasedExpressionParser) parseMatchExpression() (entities.Expr, error) {
	ep.tokenStream.Next() // 消耗 'match'

	// 解析匹配的值
	value, err := ep.parseExpression(0)
	if err != nil {
		return nil, err
	}

	if !ep.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after match value")
	}
	ep.tokenStream.Next() // 消耗 '{'

	var cases []entities.MatchCase
	for !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
		caseExpr, err := ep.parseMatchCase()
		if err != nil {
			return nil, err
		}
		cases = append(cases, caseExpr)
	}

	ep.tokenStream.Next() // 消耗 '}'
	return &entities.MatchExpr{
		Value: value,
		Cases: cases,
	}, nil
}

// parseMatchCase 解析 match case
func (ep *TokenBasedExpressionParser) parseMatchCase() (entities.MatchCase, error) {
	// TODO: 实现 match case 解析
	return entities.MatchCase{}, fmt.Errorf("match case parsing not yet implemented")
}

// parseAwaitExpression 解析 await 表达式
func (ep *TokenBasedExpressionParser) parseAwaitExpression() (entities.Expr, error) {
	ep.tokenStream.Next() // 消耗 'await'
	expr, err := ep.parseExpression(0)
	if err != nil {
		return nil, err
	}
	return &entities.AwaitExpr{
		Expression: expr,
	}, nil
}

// parseSpawnExpression 解析 spawn 表达式
func (ep *TokenBasedExpressionParser) parseSpawnExpression() (entities.Expr, error) {
	ep.tokenStream.Next() // 消耗 'spawn'
	// spawn 后面应该是函数调用表达式
	expr, err := ep.parseExpression(0)
	if err != nil {
		return nil, err
	}
	// 如果 expr 是 FuncCall，提取函数和参数
	if funcCall, ok := expr.(*entities.FuncCall); ok {
		return &entities.SpawnExpr{
			Function: &entities.Identifier{Name: funcCall.Name},
			Args:     funcCall.Args,
		}, nil
	}
	return &entities.SpawnExpr{
		Function: expr,
		Args:     []entities.Expr{},
	}, nil
}

// getOperatorPrecedence 获取运算符优先级
func (ep *TokenBasedExpressionParser) getOperatorPrecedence(tokenType lexicalVO.EnhancedTokenType) int {
	switch tokenType {
	case lexicalVO.EnhancedTokenTypeOr:
		return 1
	case lexicalVO.EnhancedTokenTypeAnd:
		return 2
	case lexicalVO.EnhancedTokenTypeEqual, lexicalVO.EnhancedTokenTypeNotEqual:
		return 3
	case lexicalVO.EnhancedTokenTypeLessThan, lexicalVO.EnhancedTokenTypeGreaterThan,
		lexicalVO.EnhancedTokenTypeLessEqual, lexicalVO.EnhancedTokenTypeGreaterEqual:
		return 4
	case lexicalVO.EnhancedTokenTypePlus, lexicalVO.EnhancedTokenTypeMinus:
		return 5
	case lexicalVO.EnhancedTokenTypeMultiply, lexicalVO.EnhancedTokenTypeDivide, lexicalVO.EnhancedTokenTypeModulo:
		return 6
	case lexicalVO.EnhancedTokenTypeChannelReceive:
		// <- 通道发送运算符（中缀），优先级与赋值类似，但比大多数运算符低
		return 2
	default:
		return 0 // 不是运算符
	}
}

// tokenTypeToOperator 将 Token 类型转换为运算符字符串
func (ep *TokenBasedExpressionParser) tokenTypeToOperator(tokenType lexicalVO.EnhancedTokenType) string {
	switch tokenType {
	case lexicalVO.EnhancedTokenTypePlus:
		return "+"
	case lexicalVO.EnhancedTokenTypeMinus:
		return "-"
	case lexicalVO.EnhancedTokenTypeMultiply:
		return "*"
	case lexicalVO.EnhancedTokenTypeDivide:
		return "/"
	case lexicalVO.EnhancedTokenTypeModulo:
		return "%"
	case lexicalVO.EnhancedTokenTypeEqual:
		return "=="
	case lexicalVO.EnhancedTokenTypeNotEqual:
		return "!="
	case lexicalVO.EnhancedTokenTypeLessThan:
		return "<"
	case lexicalVO.EnhancedTokenTypeGreaterThan:
		return ">"
	case lexicalVO.EnhancedTokenTypeLessEqual:
		return "<="
	case lexicalVO.EnhancedTokenTypeGreaterEqual:
		return ">="
	case lexicalVO.EnhancedTokenTypeAnd:
		return "&&"
	case lexicalVO.EnhancedTokenTypeOr:
		return "||"
	case lexicalVO.EnhancedTokenTypeChannelReceive:
		return "<-"
	default:
		return ""
	}
}

// shouldContinueParsing 判断是否应该继续解析
func (ep *TokenBasedExpressionParser) shouldContinueParsing(minPrecedence int) bool {
	if ep.tokenStream.IsAtEnd() {
		return false
	}
	token := ep.tokenStream.Current()
	if token == nil {
		return false
	}

	// 如果遇到语句结束符，停止解析
	if ep.isStatementTerminator(token.Type()) {
		return false
	}

	// 如果遇到左大括号，停止解析（在条件表达式等上下文中，{ 表示表达式结束）
	// 注意：这不是语句终止符，因为结构体字面量需要 {，但在条件表达式等上下文中，{ 应该结束表达式
	if token.Type() == lexicalVO.EnhancedTokenTypeLeftBrace {
		return false
	}

	// 如果遇到 =>（fat arrow），停止解析（用于 match case 的 pattern）
	if token.Type() == lexicalVO.EnhancedTokenTypeFatArrow {
		return false
	}

	// 如果遇到右括号，停止解析（用于函数调用、括号表达式等的结束）
	// 注意：这不是语句终止符，因为它是表达式的一部分，但在解析中缀表达式时，应该停止
	if token.Type() == lexicalVO.EnhancedTokenTypeRightParen {
		return false
	}

	// 如果遇到 match 关键字，停止解析（match 是语句关键字，不是表达式的一部分）
	if token.Type() == lexicalVO.EnhancedTokenTypeMatch {
		return false
	}

	// 获取运算符优先级
	precedence := ep.getOperatorPrecedence(token.Type())
	// 如果 precedence 为 0，说明不是运算符，应该停止解析
	if precedence == 0 {
		return false
	}
	// 只有当 precedence >= minPrecedence 时才继续解析
	return precedence >= minPrecedence
}

// isStatementTerminator 检查是否是语句结束符
func (ep *TokenBasedExpressionParser) isStatementTerminator(tokenType lexicalVO.EnhancedTokenType) bool {
	switch tokenType {
	case lexicalVO.EnhancedTokenTypeSemicolon,
		lexicalVO.EnhancedTokenTypeRightBrace,
		lexicalVO.EnhancedTokenTypeRightBracket,
		lexicalVO.EnhancedTokenTypeRightParen,
		// 语句关键字也应该是表达式结束符
		lexicalVO.EnhancedTokenTypeReturn,
		lexicalVO.EnhancedTokenTypeIf,
		lexicalVO.EnhancedTokenTypeWhile,
		lexicalVO.EnhancedTokenTypeFor,
		lexicalVO.EnhancedTokenTypeLet,
		lexicalVO.EnhancedTokenTypeFunc,
		lexicalVO.EnhancedTokenTypePrint,
		lexicalVO.EnhancedTokenTypeEOF:
		return true
	default:
		return false
	}
}

// checkToken 检查当前 token 类型（不消耗）
func (ep *TokenBasedExpressionParser) checkToken(tokenType lexicalVO.EnhancedTokenType) bool {
	token := ep.tokenStream.Current()
	return token != nil && token.Type() == tokenType
}
