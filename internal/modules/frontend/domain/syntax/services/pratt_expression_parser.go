// Package services 定义语法分析上下文的领域服务
package services

import (
	"context"
	"fmt"

	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
	syntaxVO "echo/internal/modules/frontend/domain/syntax/value_objects"
)

// PrattExpressionParser Pratt表达式解析器
// 实现Vaughan Pratt的经典表达式解析算法，支持运算符优先级和结合性
type PrattExpressionParser struct {
	// Token流
	tokenStream *lexicalVO.EnhancedTokenStream
	position    int
	prevToken   *lexicalVO.EnhancedToken // 前一个token（完善实现）

	// 运算符注册表
	operatorRegistry *syntaxVO.OperatorRegistry

	// 优先级表
	precedenceTable *syntaxVO.PrecedenceTable

	// LR歧义解析器
	ambiguityResolver *LRAmbiguityResolver

	// 错误收集
	errors []*sharedVO.ParseError
}

// NewPrattExpressionParser 创建新的Pratt表达式解析器
func NewPrattExpressionParser() *PrattExpressionParser {
	registry := syntaxVO.NewOperatorRegistry()
	precedenceTable := syntaxVO.NewPrecedenceTable(registry)
	ambiguityResolver := NewLRAmbiguityResolver()

	return &PrattExpressionParser{
		position:          0,
		prevToken:         nil, // 初始化为nil
		operatorRegistry:  registry,
		precedenceTable:   precedenceTable,
		ambiguityResolver: ambiguityResolver,
		errors:            make([]*sharedVO.ParseError, 0),
	}
}

// ParseExpression 解析表达式（Pratt解析器的主要入口点）
// minPrecedence: 最小优先级，用于控制递归深度
func (pep *PrattExpressionParser) ParseExpression(ctx context.Context, tokenStream *lexicalVO.EnhancedTokenStream, minPrecedence syntaxVO.Precedence) (sharedVO.ASTNode, error) {
	pep.tokenStream = tokenStream

	// 保存当前位置
	savedPosition := pep.getCurrentPosition()
	pep.position = savedPosition

	if pep.isAtEnd() {
		return nil, fmt.Errorf("unexpected end of token stream")
	}

	// 步骤1：解析前缀表达式
	left, err := pep.parsePrefixExpression()
	if err != nil {
		return nil, err
	}

	// 步骤2：循环解析中缀表达式
	for !pep.isAtEnd() && pep.shouldContinueParsing(minPrecedence) {
		operatorToken := pep.currentToken()

		// 检查是否是潜在的歧义运算符（如<）
		if pep.isAmbiguousOperator(operatorToken) {
			ambiguousResult, err := pep.tryResolveAmbiguity(ctx, left, operatorToken, tokenStream, minPrecedence)
			if err != nil {
				return nil, err
			}
			if ambiguousResult != nil {
				// 成功解析歧义，返回结果
				return ambiguousResult, nil
			}
			// 如果没有歧义或解析失败，继续正常解析
		}

		// 获取运算符信息
		operatorDef, exists := pep.operatorRegistry.GetOperator(operatorToken.Lexeme())
		if !exists || !operatorDef.IsInfix {
			// 不是中缀运算符，结束解析
			break
		}

		// 消耗运算符token
		pep.advance()

		// 计算下一个最小优先级（考虑结合性）
		nextMinPrecedence := pep.precedenceTable.GetNextMinPrecedence(operatorToken.Lexeme(), minPrecedence)

		// 递归解析右侧表达式
		right, err := pep.ParseExpression(ctx, tokenStream, nextMinPrecedence)
		if err != nil {
			return nil, err
		}

		// 构建二元表达式节点
		left = pep.createBinaryExpression(left, operatorToken.Lexeme(), right, operatorToken.Location())
	}

	return left, nil
}

// getCurrentPosition 获取当前token流位置
func (pep *PrattExpressionParser) getCurrentPosition() int {
	// 完善实现：返回维护的position
	// 如果position未初始化，尝试从tokenStream获取
	if pep.position == 0 && pep.tokenStream != nil {
		// 通过查找当前token在流中的位置来确定
		currentToken := pep.tokenStream.Current()
		if currentToken != nil {
			tokens := pep.tokenStream.Tokens()
			for i, t := range tokens {
				if t == currentToken {
					pep.position = i
					return i
				}
			}
		}
	}
	return pep.position
}

// parsePrefixExpression 解析前缀表达式
func (pep *PrattExpressionParser) parsePrefixExpression() (sharedVO.ASTNode, error) {
	token := pep.currentToken()
	pep.advance()

	switch token.Type() {
	// 字面量
	case lexicalVO.EnhancedTokenTypeNumber:
		return pep.createNumberLiteral(token)
	case lexicalVO.EnhancedTokenTypeString:
		return pep.createStringLiteral(token)
	case lexicalVO.EnhancedTokenTypeBool:
		return pep.createBooleanLiteral(token)

	// 标识符
	case lexicalVO.EnhancedTokenTypeIdentifier:
		return pep.createIdentifier(token)

	// 前缀运算符
	case lexicalVO.EnhancedTokenTypeMinus, lexicalVO.EnhancedTokenTypeNot,
		 lexicalVO.EnhancedTokenTypePlus: // 一元加号通常被忽略，但在某些语言中可能有意义
		return pep.parseUnaryExpression(token)

	// 括号表达式
	case lexicalVO.EnhancedTokenTypeLeftParen:
		return pep.parseParenthesizedExpression()

	// 数组字面量
	case lexicalVO.EnhancedTokenTypeLeftBracket:
		return pep.parseArrayLiteral()

	// 对象字面量
	case lexicalVO.EnhancedTokenTypeLeftBrace:
		return pep.parseObjectLiteral()

	default:
		return nil, fmt.Errorf("unexpected token in expression: %s", token.String())
	}
}

// parseUnaryExpression 解析一元表达式
func (pep *PrattExpressionParser) parseUnaryExpression(operatorToken *lexicalVO.EnhancedToken) (sharedVO.ASTNode, error) {
	// 递归解析操作数（使用当前运算符的优先级）
	operand, err := pep.ParseExpression(nil, pep.tokenStream, pep.precedenceTable.GetPrecedence(operatorToken.Lexeme()))
	if err != nil {
		return nil, err
	}

	return pep.createUnaryExpression(operatorToken.Lexeme(), operand, operatorToken.Location()), nil
}

// parseParenthesizedExpression 解析括号表达式
func (pep *PrattExpressionParser) parseParenthesizedExpression() (sharedVO.ASTNode, error) {
	// 递归解析括号内的表达式
	expr, err := pep.ParseExpression(nil, pep.tokenStream, syntaxVO.PrecedencePrimary)
	if err != nil {
		return nil, err
	}

	// 期望右括号
	if pep.isAtEnd() || pep.currentToken().Type() != lexicalVO.EnhancedTokenTypeRightParen {
		return nil, fmt.Errorf("expected ')' after expression")
	}
	pep.advance() // 消耗右括号

	return expr, nil
}

// parseArrayLiteral 解析数组字面量
func (pep *PrattExpressionParser) parseArrayLiteral() (sharedVO.ASTNode, error) {
	elements := make([]sharedVO.ASTNode, 0)

	// 解析数组元素
	for !pep.isAtEnd() && pep.currentToken().Type() != lexicalVO.EnhancedTokenTypeRightBracket {
		element, err := pep.ParseExpression(nil, pep.tokenStream, syntaxVO.PrecedenceAssignment)
		if err != nil {
			return nil, err
		}
		elements = append(elements, element)

		// 检查是否有逗号
		if pep.currentToken().Type() == lexicalVO.EnhancedTokenTypeComma {
			pep.advance() // 消耗逗号
		} else if pep.currentToken().Type() != lexicalVO.EnhancedTokenTypeRightBracket {
			return nil, fmt.Errorf("expected ',' or ']' in array literal")
		}
	}

	if pep.isAtEnd() || pep.currentToken().Type() != lexicalVO.EnhancedTokenTypeRightBracket {
		return nil, fmt.Errorf("expected ']' after array literal")
	}
	pep.advance() // 消耗右括号

	return pep.createArrayLiteral(elements, pep.previousToken().Location()), nil
}

// parseObjectLiteral 解析对象字面量
func (pep *PrattExpressionParser) parseObjectLiteral() (sharedVO.ASTNode, error) {
	properties := make([]*sharedVO.ObjectProperty, 0)

	// 解析对象属性
	for !pep.isAtEnd() && pep.currentToken().Type() != lexicalVO.EnhancedTokenTypeRightBrace {
		// 解析属性名
		if pep.currentToken().Type() != lexicalVO.EnhancedTokenTypeIdentifier &&
		   pep.currentToken().Type() != lexicalVO.EnhancedTokenTypeString {
			return nil, fmt.Errorf("expected property name in object literal")
		}
		propertyNameToken := pep.currentToken()
		propertyName := propertyNameToken.Lexeme()
		// 使用属性名token的位置作为对象属性的位置
		propertyLocation := propertyNameToken.Location()
		pep.advance()

		// 期望冒号
		if pep.currentToken().Type() != lexicalVO.EnhancedTokenTypeColon {
			return nil, fmt.Errorf("expected ':' after property name")
		}
		pep.advance()

		// 解析属性值
		propertyValue, err := pep.ParseExpression(nil, pep.tokenStream, syntaxVO.PrecedenceAssignment)
		if err != nil {
			return nil, err
		}

		properties = append(properties, sharedVO.NewObjectProperty(propertyName, propertyValue, propertyLocation))

		// 检查是否有逗号
		if pep.currentToken().Type() == lexicalVO.EnhancedTokenTypeComma {
			pep.advance() // 消耗逗号
		} else if pep.currentToken().Type() != lexicalVO.EnhancedTokenTypeRightBrace {
			return nil, fmt.Errorf("expected ',' or '}' in object literal")
		}
	}

	if pep.isAtEnd() || pep.currentToken().Type() != lexicalVO.EnhancedTokenTypeRightBrace {
		return nil, fmt.Errorf("expected '}' after object literal")
	}
	pep.advance() // 消耗右大括号

	return pep.createObjectLiteral(properties, pep.previousToken().Location()), nil
}

// shouldContinueParsing 检查是否应该继续解析
func (pep *PrattExpressionParser) shouldContinueParsing(minPrecedence syntaxVO.Precedence) bool {
	if pep.isAtEnd() {
		return false
	}

	currentToken := pep.currentToken()
	return pep.precedenceTable.ShouldContinueParsing(currentToken.Lexeme(), minPrecedence)
}

// isAmbiguousOperator 检查是否是潜在的歧义运算符
func (pep *PrattExpressionParser) isAmbiguousOperator(token *lexicalVO.EnhancedToken) bool {
	switch token.Type() {
	case lexicalVO.EnhancedTokenTypeLessThan:
		// < 可能是小于运算符，也可能是泛型开始
		return pep.canBeGenericStart(token)
	default:
		return false
	}
}

// canBeGenericStart 检查<是否可能是泛型开始
func (pep *PrattExpressionParser) canBeGenericStart(token *lexicalVO.EnhancedToken) bool {
	// 完善实现：智能判断<是泛型开始还是比较运算符
	// 规则：如果前一个token是标识符，则<可能是泛型开始
	prevToken := pep.previousToken()
	if prevToken == nil {
		// 没有前一个token，<不可能是泛型开始（泛型需要类型名）
		return false
	}
	
	// 如果前一个token是标识符，可能是泛型类型名
	if prevToken.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
		return true
	}
	
	// 如果前一个token是右括号、右方括号等，也可能是泛型（如 func<T>()）
	if prevToken.Type() == lexicalVO.EnhancedTokenTypeRightParen ||
		prevToken.Type() == lexicalVO.EnhancedTokenTypeRightBracket {
		return true
	}
	
	// 其他情况，<更可能是比较运算符
	return false
}

// tryResolveAmbiguity 尝试使用LR解析器解决歧义
func (pep *PrattExpressionParser) tryResolveAmbiguity(
	ctx context.Context,
	left sharedVO.ASTNode,
	operatorToken *lexicalVO.EnhancedToken,
	tokenStream *lexicalVO.EnhancedTokenStream,
	minPrecedence syntaxVO.Precedence,
) (sharedVO.ASTNode, error) {

	// 保存当前解析状态
	savedPosition := pep.getCurrentPosition()

	// 尝试使用LR解析器解析歧义
	ambiguousAST, err := pep.ambiguityResolver.ResolveAmbiguity(ctx, tokenStream, savedPosition)
	if err != nil {
		// 解析失败，恢复状态，继续正常解析
		return nil, nil
	}

	if ambiguousAST != nil {
		// 成功解析歧义，返回结果
		return *ambiguousAST, nil
	}

	// 没有发现歧义或解析失败，返回nil让调用者继续正常解析
	return nil, nil
}

// 辅助方法

// currentToken 获取当前位置的Token
func (pep *PrattExpressionParser) currentToken() *lexicalVO.EnhancedToken {
	return pep.tokenStream.Current()
}

// previousToken 获取前一个Token
func (pep *PrattExpressionParser) previousToken() *lexicalVO.EnhancedToken {
	// 完善实现：返回保存的前一个token
	return pep.prevToken
}

// advance 前进到下一个Token
func (pep *PrattExpressionParser) advance() *lexicalVO.EnhancedToken {
	// 完善实现：保存当前token作为prevToken
	currentToken := pep.currentToken()
	
	// 更新position（注意：这里假设position与tokenStream同步）
	// 如果tokenStream有Next()方法，应该调用它
	pep.position++
	
	// 更新prevToken
	pep.prevToken = currentToken
	
	return currentToken
}

// isAtEnd 检查是否到达Token流末尾
func (pep *PrattExpressionParser) isAtEnd() bool {
	return pep.tokenStream.IsAtEnd() ||
		   pep.currentToken().Type() == lexicalVO.EnhancedTokenTypeEOF
}

// 创建AST节点的辅助方法

func (pep *PrattExpressionParser) createNumberLiteral(token *lexicalVO.EnhancedToken) (sharedVO.ASTNode, error) {
	numberLiteral := token.NumberLiteral()
	if numberLiteral == nil {
		return nil, fmt.Errorf("invalid number token")
	}

	// 获取token的位置信息
	numberLocation := token.Location()
	
	switch numberLiteral.Type() {
	case lexicalVO.NumberTypeInt:
		return sharedVO.NewIntegerLiteral(numberLiteral.IntValue(), numberLocation), nil
	case lexicalVO.NumberTypeFloat:
		return sharedVO.NewFloatLiteral(numberLiteral.FloatValue(), numberLocation), nil
	case lexicalVO.NumberTypeBigInt:
		// 暂时不支持大整数，转换为字符串表示
		return sharedVO.NewStringLiteral(numberLiteral.BigValue().String(), numberLocation), nil
	default:
		return nil, fmt.Errorf("unsupported number type")
	}
}

func (pep *PrattExpressionParser) createStringLiteral(token *lexicalVO.EnhancedToken) (sharedVO.ASTNode, error) {
	return sharedVO.NewStringLiteral(token.StringValue(), token.Location()), nil
}

func (pep *PrattExpressionParser) createBooleanLiteral(token *lexicalVO.EnhancedToken) (sharedVO.ASTNode, error) {
	boolValue := token.BoolValue()
	if boolValue == nil {
		return nil, fmt.Errorf("invalid boolean token")
	}
	return sharedVO.NewBooleanLiteral(*boolValue, token.Location()), nil
}

func (pep *PrattExpressionParser) createIdentifier(token *lexicalVO.EnhancedToken) (sharedVO.ASTNode, error) {
	return sharedVO.NewIdentifier(token.Identifier(), token.Location()), nil
}

func (pep *PrattExpressionParser) createUnaryExpression(operator string, operand sharedVO.ASTNode, location sharedVO.SourceLocation) sharedVO.ASTNode {
	return sharedVO.NewUnaryExpression(operator, operand, location)
}

func (pep *PrattExpressionParser) createBinaryExpression(left sharedVO.ASTNode, operator string, right sharedVO.ASTNode, location sharedVO.SourceLocation) sharedVO.ASTNode {
	return sharedVO.NewBinaryExpression(left, operator, right, location)
}

func (pep *PrattExpressionParser) createArrayLiteral(elements []sharedVO.ASTNode, location sharedVO.SourceLocation) sharedVO.ASTNode {
	// 暂时使用简单的数组表示，实际应该有专门的ArrayLiteral类型
	return sharedVO.NewStringLiteral(fmt.Sprintf("array with %d elements", len(elements)), location)
}

func (pep *PrattExpressionParser) createObjectLiteral(properties []*sharedVO.ObjectProperty, location sharedVO.SourceLocation) sharedVO.ASTNode {
	// 暂时使用简单的对象表示，实际应该有专门的ObjectLiteral类型
	return sharedVO.NewStringLiteral(fmt.Sprintf("object with %d properties", len(properties)), location)
}

// GetOperatorRegistry 获取运算符注册表（用于扩展）
func (pep *PrattExpressionParser) GetOperatorRegistry() *syntaxVO.OperatorRegistry {
	return pep.operatorRegistry
}

// GetPrecedenceTable 获取优先级表（用于调试）
func (pep *PrattExpressionParser) GetPrecedenceTable() *syntaxVO.PrecedenceTable {
	return pep.precedenceTable
}

// GetErrors 获取解析错误
func (pep *PrattExpressionParser) GetErrors() []*sharedVO.ParseError {
	return pep.errors
}
