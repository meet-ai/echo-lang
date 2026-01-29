package parser

import (
	"fmt"
	"log"
	"strings"

	"echo/internal/modules/frontend/domain/entities"
	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
)

// TokenBasedExpressionParser 基于 Token 的表达式解析器
// 使用 Pratt Parser 算法处理运算符优先级
// 直接使用 EnhancedTokenStream 管理位置，不维护自己的 position
type TokenBasedExpressionParser struct {
	tokenStream      *lexicalVO.EnhancedTokenStream
	statementParser  *TokenBasedStatementParser // 用于解析语句块（如 match case body）
	inMatchCaseBody  bool                        // 标记是否在 match case body 上下文中
	inMatchValue     bool                        // 标记是否在解析 match 的匹配值（match expr {），此时 Ident 后跟 { 应视为“匹配值”结束，不解析为 struct literal
}

// NewTokenBasedExpressionParser 创建新的基于 Token 的表达式解析器
func NewTokenBasedExpressionParser() *TokenBasedExpressionParser {
	return &TokenBasedExpressionParser{}
}

// SetStatementParser 设置语句解析器（用于解析语句块，如 match case body）
func (ep *TokenBasedExpressionParser) SetStatementParser(statementParser *TokenBasedStatementParser) {
	ep.statementParser = statementParser
}

// SetInMatchCaseBody 设置是否在 match case body 上下文中
// 当在 match case body 中解析表达式时，如果遇到 Ident(，可能是下一个 case 的 pattern 开始，应该停止解析
func (ep *TokenBasedExpressionParser) SetInMatchCaseBody(inMatchCaseBody bool) {
	ep.inMatchCaseBody = inMatchCaseBody
}

// SetInMatchValue 设置是否在解析 match 的匹配值上下文中（match expr {）
// 当解析 match 的值时，Ident 后跟 { 应视为匹配值结束（即只取 Ident），{ 由 parseMatchExpression 消耗，不解析为 struct literal
func (ep *TokenBasedExpressionParser) SetInMatchValue(inMatchValue bool) {
	ep.inMatchValue = inMatchValue
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
		// 注意：在表达式上下文中，if 可以是三元表达式的开始，不是语句终止符
		// 所以我们需要特殊处理：如果当前 token 是 if，允许它作为表达式开始
		if currentToken.Type() == lexicalVO.EnhancedTokenTypeIf {
			// if 在表达式上下文中是三元表达式的开始，允许
		} else if tempParser.isStatementTerminator(currentToken.Type()) {
			return nil, fmt.Errorf("expression cannot start with statement terminator: %s", currentToken.Type())
		}
	}

	// 使用 Pratt Parser 算法，从最低优先级开始
	result, err := ep.parseExpression(0)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] ParseExpression: Returning expression type=%T", result)
	return result, nil
}

// parseExpression 使用 Pratt Parser 算法解析表达式
// minPrecedence: 最小优先级，用于控制递归深度
func (ep *TokenBasedExpressionParser) parseExpression(minPrecedence int) (entities.Expr, error) {
	// ✅ 修复：在开始解析之前，检查是否是 fat_arrow
	// 如果遇到 fat_arrow，说明表达式已经结束（在 pattern 解析上下文中）
	if !ep.tokenStream.IsAtEnd() {
		currentToken := ep.tokenStream.Current()
		if currentToken != nil && currentToken.Type() == lexicalVO.EnhancedTokenTypeFatArrow {
			return nil, fmt.Errorf("expression cannot start with fat_arrow")
		}
	}

	// 检查当前 token 是否是语句终止符
	// 如果是，说明表达式已经结束（可能在递归调用中）
	if !ep.tokenStream.IsAtEnd() {
		currentToken := ep.tokenStream.Current()
		// 注意：在表达式上下文中，if 可以是三元表达式的开始，不是语句终止符
		if currentToken != nil && currentToken.Type() != lexicalVO.EnhancedTokenTypeIf && ep.isStatementTerminator(currentToken.Type()) {
			return nil, fmt.Errorf("expression ended at statement terminator: %s", currentToken.Type())
		}
	}

	// 步骤1：解析前缀表达式（字面量、标识符、括号、一元运算符等）
	// 注意：语句结束符检查在 shouldContinueParsing 中进行
	left, err := ep.parsePrefixExpression()
	if err != nil {
		return nil, err
	}

	// 步骤2：循环解析中缀表达式（二元运算符）和后缀运算符
	for !ep.tokenStream.IsAtEnd() && ep.shouldContinueParsing(minPrecedence) {
		operatorToken := ep.tokenStream.Current()
		log.Printf("[DEBUG] Main loop: operatorToken=%v (type=%s), left type=%T, position=%d", operatorToken, operatorToken.Type(), left, ep.tokenStream.Position())

		// ✅ 修复：在每次循环迭代时，额外检查是否是 fat_arrow
		// 如果遇到 fat_arrow，立即停止解析（用于 match case 的边界）
		if operatorToken.Type() == lexicalVO.EnhancedTokenTypeFatArrow {
			log.Printf("[DEBUG] Main loop: Found fat_arrow, stopping expression parsing (position: %d)", ep.tokenStream.Position())
			break
		}

		// 检查是否是后缀运算符（错误传播操作符 ?）
		if operatorToken.Type() == lexicalVO.EnhancedTokenTypeErrorPropagation {
			ep.tokenStream.Next() // 消耗 '?'
			left = &entities.ErrorPropagation{
				Expr: left,
			}
			continue
		}

		// ============ 关键修复：处理 Identifier 后跟 '(' 的情况（方法调用） ============
		// 例如：v.data.push(item) 中，在创建 StructAccess 后，Current() 是 push（Identifier）
		// shouldContinueParsing 已经检查了 push 后面是 '('，所以循环继续
		// 这里需要处理方法调用
		if operatorToken.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
			log.Printf("[DEBUG] Main loop: Found Identifier %s as operatorToken", operatorToken.Lexeme())
			// 检查下一个token是否是 '('
			savedPosition := ep.tokenStream.Position()
			ep.tokenStream.Next()
			nextToken := ep.tokenStream.Current()
			ep.tokenStream.SetPosition(savedPosition) // 回退
			
			log.Printf("[DEBUG] Main loop: Lookahead result: nextToken=%v (type=%s)", nextToken, nextToken.Type())
			
			// ✅ 修复：print_string 和 print_int 是函数调用，不是方法调用
			// 如果左边已经有表达式（left != nil），且 Identifier 是 print_string 或 print_int，应该停止解析
			// 让它们作为独立的函数调用处理（由语句解析器处理）
			if left != nil && nextToken != nil && nextToken.Type() == lexicalVO.EnhancedTokenTypeLeftParen {
				identifierName := operatorToken.Lexeme()
				if identifierName == "print_string" || identifierName == "print_int" {
					log.Printf("[DEBUG] Main loop: Found %s(...) after expression, treating as function call (not method call), stopping", identifierName)
					return left, nil
				}
			}
			
			// ✅ 修复 T-DEV-018：在 match case body 上下文中，如果遇到 Ident(，可能是下一个 case 的 pattern 开始
			// 例如：Ok(TcpStream { ... }) 后面的 Err(e) 是下一个 case 的 pattern，不应该解析为方法调用
			if ep.inMatchCaseBody && nextToken != nil && nextToken.Type() == lexicalVO.EnhancedTokenTypeLeftParen {
				log.Printf("[DEBUG] Main loop: In match case body context, Ident( detected, stopping to avoid consuming next case pattern")
				break
			}
			
			if nextToken != nil && nextToken.Type() == lexicalVO.EnhancedTokenTypeLeftParen {
				log.Printf("[DEBUG] Main loop: Processing method call: %s(...)", operatorToken.Lexeme())
				// 这是方法调用
				methodToken := operatorToken
				ep.tokenStream.Next() // 消耗方法名
				// 此时 Current() 应该是 '('
				log.Printf("[DEBUG] Main loop: Before parseMethodCall, current token=%v", ep.tokenStream.Current())
				methodCall, err := ep.parseMethodCall(left, methodToken)
				if err != nil {
					log.Printf("[ERROR] Main loop: parseMethodCall failed: %v", err)
					return nil, err
				}
				left = methodCall
				log.Printf("[DEBUG] Main loop: Method call parsed, left type=%T, current token=%v", left, ep.tokenStream.Current())
				
				// ✅ 修复：在 continue 之前检查 shouldContinueParsing
				// 如果遇到 fat_arrow 或其他停止条件，应该停止解析
				if !ep.shouldContinueParsing(minPrecedence) {
					log.Printf("[DEBUG] Main loop: shouldContinueParsing returned false, stopping after method call")
					return left, nil
				}
				// parseMethodCall 已经消耗了 '(' 和参数，继续循环检查链式调用
				continue
			}
			// 如果不是方法调用，不应该进入这个分支（shouldContinueParsing 应该返回 false）
			// 但为了安全，这里也处理
			log.Printf("[DEBUG] Main loop: Identifier %s is NOT followed by '(', returning left", operatorToken.Lexeme())
			return left, nil
		}
		// ============ 修复结束 ============

		// 特殊处理：结构体访问或方法调用操作符 '.'
		if operatorToken.Type() == lexicalVO.EnhancedTokenTypeDot {
			// shouldContinueParsing 已经检查了优先级，这里不需要再次检查
			// 消耗 '.'
			ep.tokenStream.Next()
			
			// 解析字段名或方法名
			fieldToken := ep.tokenStream.Current()
			if fieldToken == nil {
				return nil, fmt.Errorf("expected identifier after '.'")
			}
			
			var fieldName string
			if fieldToken.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
				fieldName = fieldToken.Lexeme()
			} else if fieldToken.Type() == lexicalVO.EnhancedTokenTypePrint {
				fieldName = "print"
			} else {
				return nil, fmt.Errorf("expected identifier after '.', got %s", fieldToken.Type())
			}
			ep.tokenStream.Next() // 消耗字段名
			log.Printf("[DEBUG] After consuming field name '%s', current token=%v, position=%d", fieldName, ep.tokenStream.Current(), ep.tokenStream.Position())
			
			// 检查是否是方法调用
			if ep.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
				log.Printf("[DEBUG] Field '%s' is followed by '(', treating as method call", fieldName)
				// 方法调用
				methodCall, err := ep.parseMethodCall(left, fieldToken)
				if err != nil {
					return nil, err
				}
				left = methodCall
				log.Printf("[DEBUG] Field '%s' method call completed, left type=%T, current token=%v", fieldName, left, ep.tokenStream.Current())
				// 方法调用后继续循环，检查是否还有链式调用（如 obj.method1().method2()）
				// shouldContinueParsing 会检查下一个 token 是否是运算符
				continue
			}
			
			log.Printf("[DEBUG] Field '%s' is NOT followed by '(', creating StructAccess", fieldName)
			// 结构体字段访问
			left = &entities.StructAccess{
				Object: left,
				Field:  fieldName,
			}
			
			log.Printf("[DEBUG] Created StructAccess, field=%s, current token=%v, position=%d", fieldName, ep.tokenStream.Current(), ep.tokenStream.Position())
			
			// ============ 关键修复：立即处理链式调用 ============
			// 在循环条件检查之前，主动检查并处理链式调用
			// 这样可以避免 shouldContinueParsing 在遇到 Identifier 时返回 false
			// 例如：v.data.push(item) 中，消耗 data 后，Current() 是 push（Identifier）
			// 如果 push 后面是 '('，则是方法调用，需要立即处理
			if !ep.tokenStream.IsAtEnd() {
				nextToken := ep.tokenStream.Current()
				log.Printf("[DEBUG] After StructAccess, nextToken=%v (type=%s), position=%d", nextToken, nextToken.Type(), ep.tokenStream.Position())
				if nextToken != nil {
					// 情况1：检查是否是方法调用（如 v.data.push(item)）
					if nextToken.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
						log.Printf("[DEBUG] Found Identifier after StructAccess: %s", nextToken.Lexeme())
						// 使用 SetPosition + Position 实现 lookahead
						savedPosition := ep.tokenStream.Position()
						ep.tokenStream.Next() // 移动到下一个token
						peekToken := ep.tokenStream.Current()
						ep.tokenStream.SetPosition(savedPosition) // 回退
						
						log.Printf("[DEBUG] Lookahead: peekToken=%v (type=%s)", peekToken, peekToken.Type())
						
						if peekToken != nil && peekToken.Type() == lexicalVO.EnhancedTokenTypeLeftParen {
							log.Printf("[DEBUG] Found method call: %s(...)", nextToken.Lexeme())
							// 这是方法调用：Identifier + '('
							methodToken := nextToken
							ep.tokenStream.Next() // 消耗方法名
							// 此时 Current() 应该是 '('
							log.Printf("[DEBUG] Before parseMethodCall, current token=%v", ep.tokenStream.Current())
							methodCall, err := ep.parseMethodCall(left, methodToken)
							if err != nil {
								log.Printf("[ERROR] parseMethodCall failed: %v", err)
								return nil, err
							}
							left = methodCall
							log.Printf("[DEBUG] Method call parsed successfully, left type=%T, current token=%v", left, ep.tokenStream.Current())
							// parseMethodCall 已经消耗了 '(' 和参数，继续循环检查链式调用
							continue
						} else {
							log.Printf("[DEBUG] Identifier %s is not followed by '(', skipping method call handling", nextToken.Lexeme())
						}
					}
					
					// 情况2：检查是否是链式字段访问（如 v.data.field）
					if nextToken.Type() == lexicalVO.EnhancedTokenTypeDot {
						log.Printf("[DEBUG] Found chain field access after StructAccess")
						// 直接 continue，让循环处理下一个 '.'
						continue
					}
				}
			}
			// ============ 修复结束 ============
			
			// 如果没有匹配的链式调用，继续执行后续逻辑（数组索引等）
			
			// 检查是否有数组索引访问（如 obj.field[index]）
			if ep.checkToken(lexicalVO.EnhancedTokenTypeLeftBracket) {
				ep.tokenStream.Next() // 消耗 '['
				
				// 检查是否是空索引或切片
				if ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
					ep.tokenStream.Next() // 消耗 ']'
					// 空切片
					left = &entities.SliceExpr{
						Array: left,
						Start: nil,
						End:   nil,
					}
					continue
				}
				
				// 解析索引表达式
				index, err := ep.parseExpression(0)
				if err != nil {
					return nil, err
				}
				
				// 检查是否是切片操作
				if ep.checkToken(lexicalVO.EnhancedTokenTypeColon) {
					ep.tokenStream.Next() // 消耗 ':'
					var end entities.Expr
					if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
						end, err = ep.parseExpression(0)
						if err != nil {
							return nil, err
						}
					}
					if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
						return nil, fmt.Errorf("expected ']' after slice end")
					}
					ep.tokenStream.Next() // 消耗 ']'
					left = &entities.SliceExpr{
						Array: left,
						Start: index,
						End:   end,
					}
					continue
				}
				
				// 普通索引访问
				if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
					return nil, fmt.Errorf("expected ']' after index")
				}
				ep.tokenStream.Next() // 消耗 ']'
				left = &entities.IndexExpr{
					Array: left,
					Index: index,
				}
				continue
			}
			
			// 如果没有更多的链式调用或数组索引，继续循环让 shouldContinueParsing 检查其他运算符
			// 注意：这里 continue 后，循环会再次检查 shouldContinueParsing
			// 如果下一个token不是运算符，shouldContinueParsing 会返回false，循环退出
			continue
		}

		// ============ 类型转换表达式解析 ============
		// 处理 expr as Type 语法
		if operatorToken.Type() == lexicalVO.EnhancedTokenTypeAs {
			// 消耗 'as' 关键字
			ep.tokenStream.Next()
			
			// 解析目标类型
			targetType, err := ep.parseTypeName()
			if err != nil {
				return nil, fmt.Errorf("failed to parse target type in type cast: %w", err)
			}
			
			log.Printf("[DEBUG] Parsed type cast: %s as %s", left.String(), targetType)
			
			// 创建类型转换表达式
			left = &entities.TypeCastExpr{
				Expr:      left,
				TargetType: targetType,
			}
			
			// 继续循环，检查是否还有链式调用
			continue
		}
		// ============ 类型转换解析结束 ============

		// 获取运算符优先级
		precedence := ep.getOperatorPrecedence(operatorToken.Type())
		if precedence < minPrecedence {
			log.Printf("[DEBUG] Main loop: precedence %d < minPrecedence %d, breaking", precedence, minPrecedence)
			break
		}

		// 消耗运算符 token
		ep.tokenStream.Next()
		log.Printf("[DEBUG] Main loop: After consuming operator, current token=%v (type=%s), position=%d", ep.tokenStream.Current(), ep.tokenStream.Current().Type(), ep.tokenStream.Position())

		// 递归解析右侧表达式（考虑结合性）
		right, err := ep.parseExpression(precedence + 1) // 左结合：+1
		if err != nil {
			return nil, err
		}
		log.Printf("[DEBUG] Main loop: After parsing right expression, current token=%v (type=%s), position=%d", ep.tokenStream.Current(), ep.tokenStream.Current().Type(), ep.tokenStream.Position())

		// 构建二元表达式
		left = &entities.BinaryExpr{
			Left:  left,
			Op:    ep.tokenTypeToOperator(operatorToken.Type()),
			Right: right,
		}
	}

	// 检查是否有后缀运算符（在循环结束后）
	if !ep.tokenStream.IsAtEnd() {
		operatorToken := ep.tokenStream.Current()
		if operatorToken != nil && operatorToken.Type() == lexicalVO.EnhancedTokenTypeErrorPropagation {
			ep.tokenStream.Next() // 消耗 '?'
			left = &entities.ErrorPropagation{
				Expr: left,
			}
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
	// 注意：在表达式上下文中，if 可以是三元表达式的开始，不是语句终止符
	if token.Type() == lexicalVO.EnhancedTokenTypeIf {
		// if 在表达式上下文中是三元表达式的开始，允许
	} else if ep.isStatementTerminator(token.Type()) {
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
		// 检查是否是 await 关键字（await 可能被识别为标识符）
		if token.Lexeme() == "await" {
			return ep.parseAwaitExpression()
		}
		// 检查是否是 nil 关键字（用于指针比较）
		if token.Lexeme() == "nil" {
			ep.tokenStream.Next()
			// 将 nil 转换为 runtime_null_ptr() 函数调用
			return &entities.FuncCall{
				Name: "runtime_null_ptr",
				Args: []entities.Expr{},
			}, nil
		}
		// 检查是否是 Some/None/Ok/Err 关键字（Option/Result 字面量）
		if token.Lexeme() == "Some" || token.Lexeme() == "None" || 
		   token.Lexeme() == "Ok" || token.Lexeme() == "Err" {
			return ep.parseOptionResultLiteral(token)
		}
		ep.tokenStream.Next()
		// 注意：parseIdentifierOrCall 会检查下一个 token 是否是 '(', '.', '[' 等
		// 如果是 '.'，会调用 parseStructAccessOrMethodCall
		// 如果是 '['，会处理数组索引或泛型函数调用
		// 如果是 '('，会调用 parseFunctionCall
		return ep.parseIdentifierOrCall(token)

	// 数组字面量或切片类型构造函数
	case lexicalVO.EnhancedTokenTypeLeftBracket:
		// 检查是否是切片类型构造函数 []T(expr)
		// 通过 lookahead 检查： [ ] T ( expr )
		savedPosition := ep.tokenStream.Position()
		ep.tokenStream.Next() // 消耗 '['
		
		// 检查是否是 ']'（切片类型标记）
		if ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
			ep.tokenStream.Next() // 消耗 ']'
			
			// 检查后面是否是类型名（标识符）
			if ep.checkToken(lexicalVO.EnhancedTokenTypeIdentifier) {
				// 解析元素类型
				elementTypeToken := ep.tokenStream.Current()
				elementType := elementTypeToken.Lexeme()
				ep.tokenStream.Next() // 消耗元素类型
				
				// 检查是否是 '('（类型构造函数）
				if ep.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
					// 这是切片类型构造函数：[]T(expr)
					sliceType := "[]" + elementType
					
					// 解析参数表达式
					ep.tokenStream.Next() // 消耗 '('
					arg, err := ep.parseExpression(0)
					if err != nil {
						ep.tokenStream.SetPosition(savedPosition)
						return nil, fmt.Errorf("failed to parse slice type constructor argument: %w", err)
					}
					
					// 检查是否有多个参数
					if ep.checkToken(lexicalVO.EnhancedTokenTypeComma) {
						ep.tokenStream.SetPosition(savedPosition)
						return nil, fmt.Errorf("slice type constructor []%s expects exactly one argument", elementType)
					}
					
					// 检查右括号
					if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
						currentToken := ep.tokenStream.Current()
						ep.tokenStream.SetPosition(savedPosition)
						return nil, fmt.Errorf("expected ')' after slice type constructor argument, got %s", currentToken.Type())
					}
					ep.tokenStream.Next() // 消耗 ')'
					
					// 生成类型转换表达式
					log.Printf("[DEBUG] Parsed slice type constructor: []%s(%s)", elementType, arg.String())
					return &entities.TypeCastExpr{
						Expr:      arg,
						TargetType: sliceType,
					}, nil
				}
			}
		}
		
		// 如果不是切片类型构造函数，回退并解析数组字面量
		ep.tokenStream.SetPosition(savedPosition)
		return ep.parseArrayLiteral()

	// 结构体字面量
	case lexicalVO.EnhancedTokenTypeLeftBrace:
		return ep.parseStructLiteral()

	// 括号表达式
	case lexicalVO.EnhancedTokenTypeLeftParen:
		ep.tokenStream.Next() // 消耗 '('
		// 检查是否是空括号 ()
		if ep.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
			// 空括号 () - 返回 nil 表示空值
			ep.tokenStream.Next() // 消耗 ')'
			return nil, nil // nil 表示空元组/空值
		}
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
	case lexicalVO.EnhancedTokenTypeAmpersand:
		// & 操作符：可能是函数指针或取地址
		return ep.parseAmpersandExpression()
	case lexicalVO.EnhancedTokenTypeMultiply:
		// * 操作符：解引用运算符
		return ep.parseDereferenceExpression()

	// 特殊表达式
	case lexicalVO.EnhancedTokenTypeIf:
		// 三元表达式：if condition { value1 } else { value2 }
		return ep.parseTernaryExpression()
	case lexicalVO.EnhancedTokenTypeMatch:
		return ep.parseMatchExpression()
	case lexicalVO.EnhancedTokenTypeSizeOf:
		return ep.parseSizeOfExpression()
	case lexicalVO.EnhancedTokenTypeAsync:
		// async 关键字可能用于 await 表达式，但当前实现中 await 是作为标识符处理的
		// 暂时跳过，await 表达式应该通过标识符解析
		return nil, fmt.Errorf("async keyword not yet supported in expression context")
	case lexicalVO.EnhancedTokenTypeSpawn:
		return ep.parseSpawnExpression()
	case lexicalVO.EnhancedTokenTypeChan:
		// chan 关键字用于创建通道，格式：chan TypeName 或 chan Result[string] 或 chan[int]
		ep.tokenStream.Next() // 消耗 'chan'
		
		// 解析类型（支持泛型类型）
		// 注意：这里我们需要解析类型，但表达式解析器没有 parseType 方法
		// 我们需要手动解析类型表达式
		typeToken := ep.tokenStream.Current()
		if typeToken == nil {
			return nil, fmt.Errorf("expected type after 'chan'")
		}
		
		// 检查是否是直接泛型格式：chan[int]（chan 后直接跟 [）
		if typeToken.Type() == lexicalVO.EnhancedTokenTypeLeftBracket {
			ep.tokenStream.Next() // 消耗 '['
			var genericParams []string
			for {
				// 解析泛型参数类型
				paramToken := ep.tokenStream.Current()
				if paramToken == nil || paramToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
					return nil, fmt.Errorf("expected generic parameter type")
				}
				genericParams = append(genericParams, paramToken.Lexeme())
				ep.tokenStream.Next() // 消耗参数类型
				
				// 检查是否有更多参数
				if ep.checkToken(lexicalVO.EnhancedTokenTypeComma) {
					ep.tokenStream.Next() // 消耗 ','
					continue
				}
				
				// 检查是否结束
				if ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
					break
				}
				
				return nil, fmt.Errorf("expected ',' or ']' in generic parameters")
			}
			if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
				return nil, fmt.Errorf("expected ']' after generic parameters")
			}
			ep.tokenStream.Next() // 消耗 ']'
			typeName := "[" + strings.Join(genericParams, ", ") + "]"
			
			// ✅ 修复：检查后面是否是 '('（函数调用，如 chan[int]()）
			if ep.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
				// 这是函数调用：chan[int](args)
				// 创建一个临时的 token 来调用 parseFunctionCall
				// 注意：parseFunctionCall 需要原始的 token，但我们这里已经消耗了 chan 和 [int]
				// 所以我们需要创建一个包含完整类型名的临时 token
				chanToken := lexicalVO.NewEnhancedToken(
					lexicalVO.EnhancedTokenTypeIdentifier,
					"chan "+typeName,
					sharedVO.SourceLocation{}, // 临时 token，位置不重要
				)
				return ep.parseFunctionCall(chanToken)
			}
			
			// 创建通道表达式（暂时作为标识符处理，后续可以创建专门的通道表达式节点）
			return &entities.Identifier{Name: "chan " + typeName}, nil
		}
		
		// 检查是否是标识符类型
		if typeToken.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
			typeName := typeToken.Lexeme()
			ep.tokenStream.Next() // 消耗类型名
			
			// 检查是否有泛型参数，如 Result[string]
			if ep.checkToken(lexicalVO.EnhancedTokenTypeLeftBracket) {
				ep.tokenStream.Next() // 消耗 '['
				var genericParams []string
				for {
					// 解析泛型参数类型
					paramToken := ep.tokenStream.Current()
					if paramToken == nil || paramToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
						return nil, fmt.Errorf("expected generic parameter type")
					}
					genericParams = append(genericParams, paramToken.Lexeme())
					ep.tokenStream.Next() // 消耗参数类型
					
					// 检查是否有更多参数
					if ep.checkToken(lexicalVO.EnhancedTokenTypeComma) {
						ep.tokenStream.Next() // 消耗 ','
						continue
					}
					
					// 检查是否结束
					if ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
						break
					}
					
					return nil, fmt.Errorf("expected ',' or ']' in generic parameters")
				}
				if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
					return nil, fmt.Errorf("expected ']' after generic parameters")
				}
				ep.tokenStream.Next() // 消耗 ']'
				typeName = typeName + "[" + strings.Join(genericParams, ", ") + "]"
			}
			
			// 创建通道表达式（暂时作为标识符处理，后续可以创建专门的通道表达式节点）
			return &entities.Identifier{Name: "chan " + typeName}, nil
		} else {
			return nil, fmt.Errorf("expected type name after 'chan'")
		}
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
	// 注意：EnhancedTokenTypeIf 已经在上面作为三元表达式处理了，所以这里不包含它
	case lexicalVO.EnhancedTokenTypeReturn,
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
	} else if nextToken.Type() == lexicalVO.EnhancedTokenTypeLeftBrace {
		// ✅ 修复 T-DEV-019：在解析 match 的匹配值时，Ident 后跟 { 是 "match expr {"，只取 expr，不解析为 struct literal
		if ep.inMatchValue {
			return &entities.Identifier{
				Name: identifierToken.Lexeme(),
			}, nil
		}
		// ✅ 修复：在表达式上下文中，Ident 后跟 { 可能是 struct literal（TypeName { ... }）或循环体开始
		// 判断方法：如果标识符是类型名（内置类型或用户定义类型），解析为 struct literal
		// 否则，可能是循环体开始，返回标识符，让 shouldContinueParsing 处理
		
		// 检查是否是类型名（类型构造函数）
		// 注意：isTypeName 只检查内置类型，不检查用户定义类型
		// 但在表达式上下文中，如果标识符后面是 {，通常应该是 struct literal
		// 所以，如果不是内置类型，也尝试解析为 struct literal（可能是用户定义类型）
		// 但如果解析失败（例如在 if/for 语句中，{ 是循环体开始），返回标识符
		if ep.isTypeName(identifierToken.Lexeme()) {
			// 内置类型，解析为 struct literal
			structLiteral, err := ep.parseStructLiteral()
			if err != nil {
				return nil, fmt.Errorf("failed to parse struct literal: %w", err)
			}
			// 设置结构体类型名
			if sl, ok := structLiteral.(*entities.StructLiteral); ok {
				sl.Type = identifierToken.Lexeme()
			}
			return structLiteral, nil
		} else {
			// ✅ 修复：不是内置类型，可能是用户定义类型（如 Config）或循环体开始
			// 尝试解析为 struct literal，如果失败，可能是循环体开始
			// 但这里无法区分，所以总是尝试解析为 struct literal
			// 如果解析失败（例如在 for ... in obj { 或 if expr { 中），返回标识符
			// 注意：parseStructLiteral 会消耗 '{'，如果解析失败，需要回退
			savedPosition := ep.tokenStream.Position()
			structLiteral, err := ep.parseStructLiteral()
			if err != nil {
				// 解析失败，可能是循环体开始，回退并返回标识符
				// 让 shouldContinueParsing 处理，它会检查 { 并停止解析
				ep.tokenStream.SetPosition(savedPosition)
				log.Printf("[DEBUG] parseIdentifierOrCall: Failed to parse struct literal for %s, returning Identifier (position: %d)", identifierToken.Lexeme(), savedPosition)
				return &entities.Identifier{
					Name: identifierToken.Lexeme(),
				}, nil
			}
			// 设置结构体类型名
			if sl, ok := structLiteral.(*entities.StructLiteral); ok {
				sl.Type = identifierToken.Lexeme()
			}
			return structLiteral, nil
		}
	} else if nextToken.Type() == lexicalVO.EnhancedTokenTypeNamespace {
		// 命名空间访问：net::bind
		return ep.parseNamespaceAccess(identifierToken)
	} else if nextToken.Type() == lexicalVO.EnhancedTokenTypeDot {
		// 结构体访问或方法调用
		return ep.parseStructAccessOrMethodCall(identifierToken)
	} else if nextToken.Type() == lexicalVO.EnhancedTokenTypeLeftBracket {
		// 可能是数组索引访问、切片操作，也可能是泛型函数调用的类型参数
		// 先检查是否是泛型函数调用：identifier[Type](args)
		// 通过检查 [ 后面是否是类型（标识符），然后是 ]，然后是 (
		ep.tokenStream.Next() // 消耗 '['
		
		// 先检查是否是空切片或切片操作（以 : 开始）
		if ep.checkToken(lexicalVO.EnhancedTokenTypeColon) {
			// 切片操作，使用 parseIndexAccess 处理
			return ep.parseIndexAccess(identifierToken)
		}
		
		// 检查是否是类型参数（标识符）
		// ✅ 修复：只有在类型上下文中才应该将 identifier[Type] 解析为泛型类型
		// 在表达式上下文中，identifier[expr] 应该是索引访问
		// 判断方法：检查 identifier[Type] 后面是否是 '(' 或 '{'（泛型函数/结构体）
		// 否则，应该解析为索引访问
		if ep.checkToken(lexicalVO.EnhancedTokenTypeIdentifier) {
			// 可能是泛型参数，继续检查
			typeParamToken := ep.tokenStream.Current()
			typeParamName := typeParamToken.Lexeme()
			
			// 使用 lookahead 检查是否是泛型类型（identifier[Type] 后面是 '(' 或 '{'）
			savedPosition := ep.tokenStream.Position()
			ep.tokenStream.Next() // 消耗类型参数
			
			// 检查是否是 ']'
			if ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
				ep.tokenStream.Next() // 消耗 ']'
				
				// 检查后面是否是 '('（泛型函数调用）或 '{'（结构体字面量）
				peekToken := ep.tokenStream.Current()
				ep.tokenStream.SetPosition(savedPosition) // 回退到类型参数前
				
				if peekToken != nil && (peekToken.Type() == lexicalVO.EnhancedTokenTypeLeftParen || peekToken.Type() == lexicalVO.EnhancedTokenTypeLeftBrace) {
					// 这是泛型类型：identifier[Type](...) 或 identifier[Type] { ... }
					ep.tokenStream.Next() // 消耗类型参数
					ep.tokenStream.Next() // 消耗 ']'
					
					// 构建完整的类型名称（包含泛型参数）
					typeName := identifierToken.Lexeme() + "[" + typeParamName + "]"
					
					// 检查后面是否是 '('（泛型函数调用）
					if ep.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
						// 这是泛型函数调用：identifier[Type](args)
						// 解析函数调用（泛型参数信息可以存储在函数名中或单独存储）
						return ep.parseFunctionCall(identifierToken)
					} else if ep.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
						// 这是结构体字面量：identifier[Type] { ... }
						// 解析结构体字面量（从 '{' 开始）
						structLiteral, err := ep.parseStructLiteral()
						if err != nil {
							return nil, fmt.Errorf("failed to parse struct literal after generic type: %w", err)
						}
						// 设置结构体类型名
						if sl, ok := structLiteral.(*entities.StructLiteral); ok {
							sl.Type = typeName
						}
						return structLiteral, nil
					} else {
						// 可能是泛型类型引用（如 Vec[int]），返回标识符
						return &entities.Identifier{
							Name: typeName,
						}, nil
					}
				} else {
					// 不是泛型类型，是索引访问：identifier[expr]
					// 回退，让下面的代码处理索引访问
					ep.tokenStream.SetPosition(savedPosition)
					// 继续执行下面的索引访问逻辑
				}
			} else if ep.checkToken(lexicalVO.EnhancedTokenTypeComma) {
				// 多个泛型参数，继续解析
				// 保存第一个类型参数（已经在上面保存了）
				ep.tokenStream.Next() // 消耗 ','
				
				// 收集所有类型参数
				typeParams := []string{typeParamName}
				
				// 继续解析更多类型参数
				for {
					if !ep.checkToken(lexicalVO.EnhancedTokenTypeIdentifier) {
						return nil, fmt.Errorf("expected generic type parameter")
					}
					typeParamToken := ep.tokenStream.Current()
					typeParams = append(typeParams, typeParamToken.Lexeme())
					ep.tokenStream.Next() // 消耗类型参数
					
					if ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
						break
					}
					if !ep.checkToken(lexicalVO.EnhancedTokenTypeComma) {
						return nil, fmt.Errorf("expected ',' or ']' in generic type parameters")
					}
					ep.tokenStream.Next() // 消耗 ','
				}
				if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
					return nil, fmt.Errorf("expected ']' after generic type parameters")
				}
				ep.tokenStream.Next() // 消耗 ']'
				
				// 构建完整的类型名称（包含所有泛型参数）
				typeName := identifierToken.Lexeme() + "[" + strings.Join(typeParams, ", ") + "]"
				
				// 检查后面是否是 '('（泛型函数调用）
				if ep.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
					// 这是泛型函数调用：identifier[Type1, Type2](args)
					return ep.parseFunctionCall(identifierToken)
				} else if ep.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
					// 这是结构体字面量：identifier[Type1, Type2] { ... }
					// 解析结构体字面量（从 '{' 开始）
					structLiteral, err := ep.parseStructLiteral()
					if err != nil {
						return nil, fmt.Errorf("failed to parse struct literal after generic type: %w", err)
					}
					// 设置结构体类型名
					if sl, ok := structLiteral.(*entities.StructLiteral); ok {
						sl.Type = typeName
					}
					return structLiteral, nil
				} else {
					// 可能是泛型类型引用（如 Vec[int, string]），返回标识符
					return &entities.Identifier{
						Name: typeName,
					}, nil
				}
			} else {
				// 不是 ']' 也不是 ','，可能是索引访问（如 array[expr]）
				// 回退，让下面的代码处理索引访问
				ep.tokenStream.SetPosition(savedPosition)
				// 继续执行下面的索引访问逻辑
			}
		}
		
		// ✅ 修复：如果不是泛型类型，解析为索引访问
		// 注意：我们已经消耗了 '['，现在需要解析索引表达式
		// 检查是否是空索引或切片
		if ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
			ep.tokenStream.Next() // 消耗 ']'
			// 空切片
			return &entities.SliceExpr{
				Array: &entities.Identifier{Name: identifierToken.Lexeme()},
				Start: nil,
				End:   nil,
			}, nil
		}

		// 解析索引表达式
		index, err := ep.parseExpression(0)
		if err != nil {
			return nil, err
		}

		// 检查是否是切片操作
		if ep.checkToken(lexicalVO.EnhancedTokenTypeColon) {
			ep.tokenStream.Next() // 消耗 ':'
			var end entities.Expr
			if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
				end, err = ep.parseExpression(0)
				if err != nil {
					return nil, err
				}
			}
			if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
				return nil, fmt.Errorf("expected ']' after slice end")
			}
			ep.tokenStream.Next() // 消耗 ']'
			return &entities.SliceExpr{
				Array: &entities.Identifier{Name: identifierToken.Lexeme()},
				Start: index,
				End:   end,
			}, nil
		}

		// 普通索引访问
		if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
			return nil, fmt.Errorf("expected ']' after index")
		}
		ep.tokenStream.Next() // 消耗 ']'
		return &entities.IndexExpr{
			Array: &entities.Identifier{Name: identifierToken.Lexeme()},
			Index: index,
		}, nil
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
	if fieldToken == nil {
		return nil, fmt.Errorf("expected identifier after '.'")
	}
	
	var fieldName string
	if fieldToken.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
		fieldName = fieldToken.Lexeme()
	} else if fieldToken.Type() == lexicalVO.EnhancedTokenTypePrint {
		// print 是关键字，但可以作为字段名或方法名
		fieldName = "print"
	} else {
		return nil, fmt.Errorf("expected identifier after '.', got %s", fieldToken.Type())
	}
	ep.tokenStream.Next() // 消耗字段名

	// 检查是否是方法调用
	if ep.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
		return ep.parseMethodCall(object, fieldToken)
	}

	// 否则是结构体字段访问
	structAccess := &entities.StructAccess{
		Object: object,
		Field:  fieldName,
	}

	// 检查是否有数组索引访问（如 ia.data[index]）或泛型函数调用（如 collections.new[int]()）
	if ep.checkToken(lexicalVO.EnhancedTokenTypeLeftBracket) {
		ep.tokenStream.Next() // 消耗 '['
		
		// 检查是否是空索引或切片
		if ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
			ep.tokenStream.Next() // 消耗 ']'
			// 空切片
			return &entities.SliceExpr{
				Array: structAccess,
				Start: nil,
				End:   nil,
			}, nil
		}

		// ============ 关键修复：检查是否是泛型函数调用 ============
		// 检查是否是类型参数（标识符），然后是 ']'，然后是 '('
		// 例如：collections.new[int]() 中的 [int]
		// 注意：在数组索引访问中，索引表达式也可能以标识符开始（如 v.data[v.len]）
		// 所以我们需要更仔细地判断：只有在标识符后面是 ']' 且再后面是 '(' 时，才是泛型函数调用
		// 否则，应该作为索引表达式处理
		if ep.checkToken(lexicalVO.EnhancedTokenTypeIdentifier) {
			// 可能是泛型参数，先保存位置以便回退
			savedPosition := ep.tokenStream.Position()
			typeParamToken := ep.tokenStream.Current()
			ep.tokenStream.Next() // 消耗类型参数
			
			// 检查是否是 ']'
			if ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
				ep.tokenStream.Next() // 消耗 ']'
				
				// 检查后面是否是 '('（泛型函数调用）
				if ep.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
					// 这是泛型函数调用：collections.new[int]()
					// 函数名应该是 "collections.new[int]"
					funcName := objectToken.Lexeme() + "." + fieldName + "[" + typeParamToken.Lexeme() + "]"
					// 创建一个临时token用于函数调用
					// 使用当前token的位置信息
					funcToken := lexicalVO.NewIdentifierToken(funcName, typeParamToken.Location())
					return ep.parseFunctionCall(funcToken)
				}
				// 如果不是 '('，回退并继续作为索引访问处理
				// 注意：我们已经消耗了 '[', 'Type', ']'，无法完全回退
				// 这种情况下，应该返回一个泛型类型引用
				// 但为了简化，我们暂时返回一个标识符
				typeName := objectToken.Lexeme() + "." + fieldName + "[" + typeParamToken.Lexeme() + "]"
				return &entities.Identifier{
					Name: typeName,
				}, nil
			}
			// 如果不是 ']'，说明这不是泛型函数调用，而是数组索引访问
			// 例如：v.data[v.len] 中的 [v.len]
			// 此时我们已经消耗了标识符，需要回退到标识符位置，然后作为索引表达式处理
			// 回退到标识符位置
			ep.tokenStream.SetPosition(savedPosition)
		}

		// 解析索引表达式（普通数组索引访问）
		index, err := ep.parseExpression(0)
		if err != nil {
			return nil, err
		}

		// 检查是否是切片操作
		if ep.checkToken(lexicalVO.EnhancedTokenTypeColon) {
			ep.tokenStream.Next() // 消耗 ':'
			var end entities.Expr
			if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
				end, err = ep.parseExpression(0)
				if err != nil {
					return nil, err
				}
			}
			if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
				return nil, fmt.Errorf("expected ']' after slice end")
			}
			ep.tokenStream.Next() // 消耗 ']'
			return &entities.SliceExpr{
				Array: structAccess,
				Start: index,
				End:   end,
			}, nil
		}

		// 普通索引访问
		if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
			return nil, fmt.Errorf("expected ']' after index")
		}
		ep.tokenStream.Next() // 消耗 ']'
		return &entities.IndexExpr{
			Array: structAccess,
			Index: index,
		}, nil
	}

	return structAccess, nil
}

// parseOptionResultLiteral 解析 Option/Result 字面量（Some/None/Ok/Err）
func (ep *TokenBasedExpressionParser) parseOptionResultLiteral(token *lexicalVO.EnhancedToken) (entities.Expr, error) {
	ep.tokenStream.Next() // 消耗 Some/None/Ok/Err
	
	switch token.Lexeme() {
	case "Some":
		// Some(value) - 需要解析括号内的值
		if !ep.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
			return nil, fmt.Errorf("expected '(' after Some")
		}
		ep.tokenStream.Next() // 消耗 '('
		
		// 解析值表达式（可能含结构体字面量）
		value, err := ep.parseExpression(0)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Some value: %w", err)
		}
		if ep.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
			ep.tokenStream.Next() // 消耗 '}'
		}
		if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
			return nil, fmt.Errorf("expected ')' after Some value")
		}
		ep.tokenStream.Next() // 消耗 ')'
		
		return &entities.SomeLiteral{Value: value}, nil
		
	case "None":
		// None - 不需要参数
		return &entities.NoneLiteral{}, nil
		
	case "Ok":
		// Ok(value) - 需要解析括号内的值
		if !ep.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
			return nil, fmt.Errorf("expected '(' after Ok")
		}
		ep.tokenStream.Next() // 消耗 '('
		
		// 检查是否是空值 Ok()
		if ep.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
			// Ok() - 空值
			ep.tokenStream.Next() // 消耗 ')'
			return &entities.OkLiteral{Value: nil}, nil
		}
		
		// 解析值表达式（可能是普通值、空元组 () 或结构体字面量 Type { ... }）
		// 注意：parseExpression 现在支持空括号 ()，会返回 nil
		value, err := ep.parseExpression(0)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Ok value: %w", err)
		}
		// 若值为结构体字面量，parseStructLiteral 会消耗 '}'，但部分路径可能停在 '}'
		if ep.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
			ep.tokenStream.Next() // 消耗 '}'
		}
		if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
			return nil, fmt.Errorf("expected ')' after Ok value")
		}
		ep.tokenStream.Next() // 消耗 ')'
		
		return &entities.OkLiteral{Value: value}, nil
		
	case "Err":
		// Err(error) - 需要解析括号内的错误值
		if !ep.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
			return nil, fmt.Errorf("expected '(' after Err")
		}
		ep.tokenStream.Next() // 消耗 '('
		
		// 解析错误表达式（可能含结构体字面量）
		errValue, err := ep.parseExpression(0)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Err value: %w", err)
		}
		if ep.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
			ep.tokenStream.Next() // 消耗 '}'
		}
		if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
			return nil, fmt.Errorf("expected ')' after Err value")
		}
		ep.tokenStream.Next() // 消耗 ')'
		
		return &entities.ErrLiteral{Error: errValue}, nil
		
	default:
		return nil, fmt.Errorf("unexpected Option/Result literal: %s", token.Lexeme())
	}
}

// parseMethodCall 解析方法调用
func (ep *TokenBasedExpressionParser) parseMethodCall(object entities.Expr, methodToken *lexicalVO.EnhancedToken) (entities.Expr, error) {
	log.Printf("[DEBUG] parseMethodCall: method=%s, current token before Next()=%v", methodToken.Lexeme(), ep.tokenStream.Current())
	ep.tokenStream.Next() // 消耗 '('
	log.Printf("[DEBUG] parseMethodCall: After consuming '(', current token=%v", ep.tokenStream.Current())

	var args []entities.Expr
	if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
		for {
			log.Printf("[DEBUG] parseMethodCall: Parsing argument %d", len(args))
			arg, err := ep.parseExpression(0)
			if err != nil {
				log.Printf("[ERROR] parseMethodCall: Failed to parse argument: %v", err)
				return nil, err
			}
			args = append(args, arg)
			log.Printf("[DEBUG] parseMethodCall: Argument %d parsed, current token=%v", len(args)-1, ep.tokenStream.Current())

			if ep.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
				break
			}
			if !ep.checkToken(lexicalVO.EnhancedTokenTypeComma) {
				return nil, fmt.Errorf("expected ',' or ')' in method call")
			}
			ep.tokenStream.Next() // 消耗 ','
		}
	}

	log.Printf("[DEBUG] parseMethodCall: Consuming ')', current token=%v, position=%d", ep.tokenStream.Current(), ep.tokenStream.Position())
	ep.tokenStream.Next() // 消耗 ')'
	log.Printf("[DEBUG] parseMethodCall: Method call completed, current token=%v, position=%d", ep.tokenStream.Current(), ep.tokenStream.Position())
	result := &entities.MethodCallExpr{
		Receiver:   object,
		MethodName: methodToken.Lexeme(),
		Args:       args,
	}
	log.Printf("[DEBUG] parseMethodCall: Returning MethodCallExpr, receiver type=%T, method=%s, args count=%d", object, methodToken.Lexeme(), len(args))
	return result, nil
}

// parseIndexAccess 解析数组索引访问或切片操作
func (ep *TokenBasedExpressionParser) parseIndexAccess(arrayToken *lexicalVO.EnhancedToken) (entities.Expr, error) {
	array := &entities.Identifier{Name: arrayToken.Lexeme()}
	ep.tokenStream.Next() // 消耗 '['

	// 检查是否是空索引或切片
	if ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
		ep.tokenStream.Next() // 消耗 ']'
		// 空切片 numbers[:]
		return &entities.SliceExpr{
			Array: array,
			Start: nil,
			End:   nil,
		}, nil
	}

	// 检查是否是切片操作（以 : 开始，如 [:3]）
	if ep.checkToken(lexicalVO.EnhancedTokenTypeColon) {
		ep.tokenStream.Next() // 消耗 ':'
		
		// 检查是否有结束索引
		var end entities.Expr
		if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
			var err error
			end, err = ep.parseExpression(0)
			if err != nil {
				return nil, err
			}
		}
		
		if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
			return nil, fmt.Errorf("expected ']' after slice end")
		}
		ep.tokenStream.Next() // 消耗 ']'
		
		return &entities.SliceExpr{
			Array: array,
			Start: nil,
			End:   end,
		}, nil
	}

	// 解析索引表达式或切片的开始
	start, err := ep.parseExpression(0)
	if err != nil {
		return nil, err
	}

	// 检查是否是切片操作（包含 :，如 [1:3] 或 [1:]）
	if ep.checkToken(lexicalVO.EnhancedTokenTypeColon) {
		ep.tokenStream.Next() // 消耗 ':'
		
		// 检查是否有结束索引
		var end entities.Expr
		if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
			var err error
			end, err = ep.parseExpression(0)
			if err != nil {
				return nil, err
			}
		}
		
		if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
			return nil, fmt.Errorf("expected ']' after slice end")
		}
		ep.tokenStream.Next() // 消耗 ']'
		
		return &entities.SliceExpr{
			Array: array,
			Start: start,
			End:   end,
		}, nil
	}

	// 普通索引访问
	if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
		currentToken := ep.tokenStream.Current()
		if currentToken != nil {
			return nil, fmt.Errorf("expected ']' after index, got %s", currentToken.Type())
		}
		return nil, fmt.Errorf("expected ']' after index")
	}
	ep.tokenStream.Next() // 消耗 ']'

	return &entities.IndexExpr{
		Array: array,
		Index: start,
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

	// 在 match case body 中，若 { 后紧跟下一 case 的 pattern（Ident 后接 =>），说明误把下一 case 当结构体
	if ep.inMatchCaseBody && !ep.tokenStream.IsAtEnd() {
		savedPos := ep.tokenStream.Position()
		// 跳过换行、逗号
		for ep.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) || ep.checkToken(lexicalVO.EnhancedTokenTypeComma) {
			ep.tokenStream.Next()
		}
		tok := ep.tokenStream.Current()
		if tok != nil && tok.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
			ep.tokenStream.Next()
			for ep.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) || ep.checkToken(lexicalVO.EnhancedTokenTypeComma) {
				ep.tokenStream.Next()
			}
			next := ep.tokenStream.Current()
			ep.tokenStream.SetPosition(savedPos)
			if next != nil && next.Type() == lexicalVO.EnhancedTokenTypeFatArrow {
				return nil, fmt.Errorf("match case body ended at next case pattern (missing ',' or field list before next case?)")
			}
		} else {
			ep.tokenStream.SetPosition(savedPos)
		}
	}

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

			// 跳过可选的逗号/分号（多行书写时字段名后可能有换行或分隔符）
			for ep.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) || ep.checkToken(lexicalVO.EnhancedTokenTypeComma) {
				ep.tokenStream.Next()
			}

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

			// ✅ 修复：检查是否遇到了 fat_arrow（match case 的边界）
			// 如果遇到 fat_arrow，说明字段值解析消耗了太多 token，应该停止
			if ep.checkToken(lexicalVO.EnhancedTokenTypeFatArrow) {
				return nil, fmt.Errorf("unexpected fat_arrow in struct literal field value (position: %d)", ep.tokenStream.Position())
			}

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

// parseFunctionCall 解析函数调用或类型构造函数
func (ep *TokenBasedExpressionParser) parseFunctionCall(nameToken *lexicalVO.EnhancedToken) (entities.Expr, error) {
	funcName := nameToken.Lexeme()

	// 检查是否是类型名（类型构造函数语法 Type(expr) 或 Type()）
	if ep.isTypeName(funcName) {
		// 这是类型构造函数
		if !ep.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
			return nil, fmt.Errorf("expected '(' after type name %s", funcName)
		}
		ep.tokenStream.Next() // 消耗 '('

		// ✅ 修复：支持无参数的类型构造函数（如 chan[int]()）
		if ep.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
			// 无参数的类型构造函数，返回类型本身（用于创建空值，如空通道、空向量等）
			ep.tokenStream.Next() // 消耗 ')'
			log.Printf("[DEBUG] Parsed type constructor (no args): %s()", funcName)
			// 返回一个标识符，表示类型本身（类型推断系统会处理）
			return &entities.Identifier{Name: funcName}, nil
		}

		// 解析参数表达式（类型构造函数只接受一个参数）
		arg, err := ep.parseExpression(0)
		if err != nil {
			return nil, fmt.Errorf("failed to parse type constructor argument: %w", err)
		}

		// 检查是否有多个参数（类型构造函数只接受一个参数）
		if ep.checkToken(lexicalVO.EnhancedTokenTypeComma) {
			return nil, fmt.Errorf("type constructor %s expects exactly one argument, got multiple arguments", funcName)
		}

		// 检查右括号
		if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
			currentToken := ep.tokenStream.Current()
			return nil, fmt.Errorf("expected ')' after type constructor argument, got %s", currentToken.Type())
		}
		ep.tokenStream.Next() // 消耗 ')'

		// 生成类型转换表达式（与 as 关键字语法生成的 AST 相同）
		log.Printf("[DEBUG] Parsed type constructor: %s(%s)", funcName, arg.String())
		return &entities.TypeCastExpr{
			Expr:      arg,
			TargetType: funcName,
		}, nil
	}

	// 否则，正常解析函数调用
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

			// ✅ 修复：检查是否遇到了 fat_arrow（match case 的边界）
			// 如果遇到 fat_arrow，说明参数解析消耗了太多 token，应该停止
			if ep.checkToken(lexicalVO.EnhancedTokenTypeFatArrow) {
				return nil, fmt.Errorf("unexpected fat_arrow in function call arguments (position: %d)", ep.tokenStream.Position())
			}

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
		Name: funcName,
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
	log.Printf("[DEBUG] parseMatchExpression: Starting")
	ep.tokenStream.Next() // 消耗 'match'

	// 解析匹配的值（仅到 '{' 前；避免把 "c {" 解析成 struct literal）
	ep.SetInMatchValue(true)
	defer ep.SetInMatchValue(false)
	value, err := ep.parseExpression(0)
	if err != nil {
		return nil, err
	}

	if !ep.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after match value")
	}
	ep.tokenStream.Next() // 消耗 '{'
	
	log.Printf("[DEBUG] parseMatchExpression: After '{', current token=%s", 
		func() string { 
			token := ep.tokenStream.Current()
			if token != nil { return token.Lexeme() }
			return "nil"
		}())

	var cases []entities.MatchCase
	for !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) && !ep.tokenStream.IsAtEnd() {
		// 跳过分号和逗号（match case 之间可能用逗号分隔）
		if ep.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) || ep.checkToken(lexicalVO.EnhancedTokenTypeComma) {
			ep.tokenStream.Next()
			// 如果下一个 token 是右大括号，说明逗号是最后一个 case 后的，可以忽略
			if ep.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
				break
			}
			continue
		}

		log.Printf("[DEBUG] parseMatchExpression: About to call parseMatchCase, current token=%s", 
			func() string { 
				token := ep.tokenStream.Current()
				if token != nil { return token.Lexeme() }
				return "nil"
			}())
		caseExpr, err := ep.parseMatchCase()
		if err != nil {
			return nil, fmt.Errorf("failed to parse match case: %w", err)
		}
		cases = append(cases, caseExpr)

		// 检查是否有逗号或分号（多个 case 之间的分隔符）
		if ep.checkToken(lexicalVO.EnhancedTokenTypeComma) || ep.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) {
			ep.tokenStream.Next() // 消耗逗号或分号
			// 如果下一个 token 是右大括号，说明这是最后一个 case 后的分隔符
			if ep.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
				break
			}
		}
	}

	ep.tokenStream.Next() // 消耗 '}'
	return &entities.MatchExpr{
		Value: value,
		Cases: cases,
	}, nil
}

// parseSizeOfExpression 解析 sizeof(T) 表达式
// 语法：sizeof(T)，其中 T 可以是基本类型、结构体类型或泛型类型参数
func (ep *TokenBasedExpressionParser) parseSizeOfExpression() (entities.Expr, error) {
	ep.tokenStream.Next() // 消耗 'sizeof'
	
	// 解析 '('
	if !ep.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
		return nil, fmt.Errorf("expected '(' after 'sizeof'")
	}
	ep.tokenStream.Next() // 消耗 '('
	
	// 解析类型名称（支持泛型类型）
	typeName, err := ep.parseTypeName()
	if err != nil {
		return nil, fmt.Errorf("failed to parse type name in sizeof: %w", err)
	}
	
	// 解析 ')'
	if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
		return nil, fmt.Errorf("expected ')' after type name in sizeof")
	}
	ep.tokenStream.Next() // 消耗 ')'
	
	return entities.NewSizeOfExpr(typeName), nil
}

// parseTypeName 解析类型名称（支持泛型类型，如 Vec[int]，以及切片类型 []T）
func (ep *TokenBasedExpressionParser) parseTypeName() (string, error) {
	typeToken := ep.tokenStream.Current()
	if typeToken == nil {
		return "", fmt.Errorf("expected type name")
	}
	
	// 检查是否是切片类型：[]T（以 '[' 开头）
	if typeToken.Type() == lexicalVO.EnhancedTokenTypeLeftBracket {
		ep.tokenStream.Next() // 消耗 '['
		
		// 检查是否是切片类型：[]T（两个连续的 ']'）
		if ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
			// 这是切片类型：[]T
			ep.tokenStream.Next() // 消耗 ']'
			
			// 递归解析切片元素类型
			elementType, err := ep.parseTypeName()
			if err != nil {
				return "", fmt.Errorf("failed to parse slice element type: %w", err)
			}
			return "[]" + elementType, nil
		}
		
		// 这是数组类型：[T]（固定大小数组）
		// 递归解析数组元素类型
		elementType, err := ep.parseTypeName()
		if err != nil {
			return "", fmt.Errorf("failed to parse array element type: %w", err)
		}
		if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
			return "", fmt.Errorf("expected ']' after array element type")
		}
		ep.tokenStream.Next() // 消耗 ']'
		return "[" + elementType + "]", nil
	}
	
	// 检查是否是标识符类型
	if typeToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
		return "", fmt.Errorf("expected type name (identifier)")
	}
	
	typeName := typeToken.Lexeme()
	ep.tokenStream.Next() // 消耗类型名
	
	// 检查是否有泛型参数，如 Vec[int] 或 Result[int, string]
	if ep.checkToken(lexicalVO.EnhancedTokenTypeLeftBracket) {
		ep.tokenStream.Next() // 消耗 '['
		var genericParams []string
		for {
			// 解析泛型参数类型（递归调用 parseTypeName 以支持嵌套类型）
			paramType, err := ep.parseTypeName()
			if err != nil {
				return "", fmt.Errorf("failed to parse generic parameter type: %w", err)
			}
			genericParams = append(genericParams, paramType)
			
			// 检查是否有更多参数
			if ep.checkToken(lexicalVO.EnhancedTokenTypeComma) {
				ep.tokenStream.Next() // 消耗 ','
				continue
			}
			
			// 检查是否结束
			if ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
				break
			}
			
			return "", fmt.Errorf("expected ',' or ']' in generic parameters")
		}
		if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
			return "", fmt.Errorf("expected ']' after generic parameters")
		}
		ep.tokenStream.Next() // 消耗 ']'
		typeName = typeName + "[" + strings.Join(genericParams, ", ") + "]"
	}
	
	return typeName, nil
}

// isTypeName 检查标识符是否是类型名（用于类型构造函数语法 Type(expr)）
func (ep *TokenBasedExpressionParser) isTypeName(name string) bool {
	// 内置类型
	builtinTypes := map[string]bool{
		"int":   true,
		"i32":   true,
		"i64":   true,
		"u64":   true,
		"float": true,
		"f32":   true,
		"f64":   true,
		"bool":  true,
		"string": true,
		"void":  true,
	}
	if builtinTypes[name] {
		return true
	}

	// 检查是否是切片类型 []T
	if strings.HasPrefix(name, "[") && strings.HasSuffix(name, "]") {
		return true
	}

	// ✅ 修复：检查是否是通道类型 chan [T] 或 chan TypeName
	if strings.HasPrefix(name, "chan ") {
		return true
	}

	// TODO: 从符号表检查是否是用户定义类型（结构体、枚举等）
	// 目前先支持内置类型和切片类型

	return false
}

// parseAmpersandExpression 解析 & 操作符表达式
// 语法：&identifier 或 &expression[index] 或 &expression.field
// 如果 identifier 是函数名且没有后续操作，返回 FunctionPointerExpr
// 否则，返回 AddressOfExpr（取地址表达式）
func (ep *TokenBasedExpressionParser) parseAmpersandExpression() (entities.Expr, error) {
	ep.tokenStream.Next() // 消耗 '&'
	
	// 检查下一个 token 是否是标识符
	nextToken := ep.tokenStream.Current()
	if nextToken == nil {
		return nil, fmt.Errorf("expected expression after '&'")
	}
	
	if nextToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
		return nil, fmt.Errorf("expected identifier after '&', got %s", nextToken.Type())
	}
	
	// 保存标识符 token，然后调用 parseIdentifierOrCall 来解析完整表达式
	// parseIdentifierOrCall 会自动处理字段访问和索引访问
	identifierToken := nextToken
	ep.tokenStream.Next() // 消耗标识符
	
	// 检查是否是函数调用（如 &func()），暂时不支持
	if ep.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
		return nil, fmt.Errorf("function pointer from function call not yet supported: &%s()", identifierToken.Lexeme())
	}
	
	// 检查是否有后续操作（字段访问、索引访问）
	hasFieldAccess := ep.checkToken(lexicalVO.EnhancedTokenTypeDot)
	hasIndexAccess := ep.checkToken(lexicalVO.EnhancedTokenTypeLeftBracket)
	
	if !hasFieldAccess && !hasIndexAccess {
		// 单独的 &identifier，可能是函数指针
		return entities.NewFunctionPointerExpr(identifierToken.Lexeme()), nil
	}
	
	// 有后续操作，需要解析完整表达式
	// 重新解析：从标识符开始，解析完整表达式（包括字段访问、索引访问）
	// 注意：我们已经消耗了标识符 token，所以需要重新构造标识符表达式
	// 然后继续解析后续操作
	identExpr := &entities.Identifier{Name: identifierToken.Lexeme()}
	
	// 继续解析字段访问或索引访问
	var fullExpr entities.Expr = identExpr
	
	if hasFieldAccess {
		// 解析字段访问
		fieldAccess, err := ep.parseStructAccessOrMethodCall(identifierToken)
		if err != nil {
			return nil, fmt.Errorf("failed to parse field access: %w", err)
		}
		fullExpr = fieldAccess
	} else if hasIndexAccess {
		// 解析索引访问
		indexAccess, err := ep.parseIndexAccess(identifierToken)
		if err != nil {
			return nil, fmt.Errorf("failed to parse index access: %w", err)
		}
		fullExpr = indexAccess
	}
	
	// 创建取地址表达式
	return entities.NewAddressOfExpr(fullExpr), nil
}

// parseDereferenceExpression 解析解引用表达式：*expression
// 例如：*m, *ptr, *arr[0] 等
func (ep *TokenBasedExpressionParser) parseDereferenceExpression() (entities.Expr, error) {
	ep.tokenStream.Next() // 消耗 '*'
	
	// 解析被解引用的表达式（可以是标识符、字段访问、索引访问等）
	operand, err := ep.parsePrefixExpression()
	if err != nil {
		return nil, fmt.Errorf("failed to parse operand for dereference: %w", err)
	}
	
	// 创建解引用表达式
	return entities.NewDereferenceExpr(operand), nil
}

// parsePattern 解析 match pattern
// 支持的模式类型：
// - Some(identifier) → SomePattern{Variable: "identifier"}
// - None → NonePattern{}
// - Ok(identifier) → OkPattern{Variable: "identifier"}
// - Err(identifier) → ErrPattern{Variable: "identifier"}
// - identifier → IdentifierPattern{Name: "identifier"}
// - _ → WildcardPattern{}
func (ep *TokenBasedExpressionParser) parsePattern() (entities.Expr, error) {
	currentToken := ep.tokenStream.Current()
	if currentToken == nil {
		return nil, fmt.Errorf("unexpected end of token stream in pattern")
	}

	log.Printf("[DEBUG] parsePattern: Starting, current token=%s (type=%s), position=%d", 
		currentToken.Lexeme(), currentToken.Type(), ep.tokenStream.Position())

	switch currentToken.Type() {
	case lexicalVO.EnhancedTokenTypeIdentifier:
		identifierName := currentToken.Lexeme()
		ep.tokenStream.Next() // 消耗标识符

		// 检查是否是 Some/None/Ok/Err
		switch identifierName {
		case "Some":
			// Some(identifier) → SomePattern{Variable: "identifier"}
			if !ep.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
				return nil, fmt.Errorf("expected '(' after Some in pattern")
			}
			ep.tokenStream.Next() // 消耗 '('

			// 解析括号内的标识符（变量名）
			varToken := ep.tokenStream.Current()
			if varToken == nil || varToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
				return nil, fmt.Errorf("expected identifier in Some pattern")
			}
			variableName := varToken.Lexeme()
			ep.tokenStream.Next() // 消耗变量名

			if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
				return nil, fmt.Errorf("expected ')' after identifier in Some pattern")
			}
			ep.tokenStream.Next() // 消耗 ')'

			return &entities.SomePattern{Variable: variableName}, nil

		case "None":
			// None → NonePattern{}
			return &entities.NonePattern{}, nil

		case "Ok":
			// Ok(identifier) → OkPattern{Variable: "identifier"}
			if !ep.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
				return nil, fmt.Errorf("expected '(' after Ok in pattern")
			}
			ep.tokenStream.Next() // 消耗 '('

			varToken := ep.tokenStream.Current()
			if varToken == nil || varToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
				return nil, fmt.Errorf("expected identifier in Ok pattern")
			}
			variableName := varToken.Lexeme()
			ep.tokenStream.Next() // 消耗变量名

			if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
				return nil, fmt.Errorf("expected ')' after identifier in Ok pattern")
			}
			ep.tokenStream.Next() // 消耗 ')'

			log.Printf("[DEBUG] parsePattern: Ok pattern parsed, variable=%s, next token=%s (type=%s)", 
				variableName, 
				func() string { 
					token := ep.tokenStream.Current()
					if token != nil { return token.Lexeme() }
					return "nil"
				}(),
				func() string { 
					token := ep.tokenStream.Current()
					if token != nil { return string(token.Type()) }
					return "nil"
				}())
			return &entities.OkPattern{Variable: variableName}, nil

		case "Err":
			// Err(identifier) → ErrPattern{Variable: "identifier"}
			if !ep.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
				return nil, fmt.Errorf("expected '(' after Err in pattern")
			}
			ep.tokenStream.Next() // 消耗 '('

			varToken := ep.tokenStream.Current()
			if varToken == nil || varToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
				return nil, fmt.Errorf("expected identifier in Err pattern")
			}
			variableName := varToken.Lexeme()
			ep.tokenStream.Next() // 消耗变量名

			if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
				return nil, fmt.Errorf("expected ')' after identifier in Err pattern")
			}
			ep.tokenStream.Next() // 消耗 ')'

			log.Printf("[DEBUG] parsePattern: Err pattern parsed, variable=%s, next token=%s (type=%s)", 
				variableName, 
				func() string { 
					token := ep.tokenStream.Current()
					if token != nil { return token.Lexeme() }
					return "nil"
				}(),
				func() string { 
					token := ep.tokenStream.Current()
					if token != nil { return string(token.Type()) }
					return "nil"
				}())
			return &entities.ErrPattern{Variable: variableName}, nil

		default:
			// 检查是否是通配符模式 _
			if identifierName == "_" {
				// _ → WildcardPattern{}
				return &entities.WildcardPattern{}, nil
			}
			// 普通标识符 → IdentifierPattern{Name: "identifier"}
			return entities.NewIdentifierPattern(identifierName), nil
		}

	default:
		// 其他情况，尝试解析为字面量模式
		// 例如：数字字面量、字符串字面量等
		// ✅ 关键修复：如果遇到 fat_arrow，说明 pattern 解析已经完成
		// 这不应该进入 default 分支，但如果进入了，说明 token stream 位置不对
		// 直接返回错误，不要尝试解析
		if ep.checkToken(lexicalVO.EnhancedTokenTypeFatArrow) {
			// 如果 token stream 已经指向 fat_arrow，说明 pattern 解析应该已经完成
			// 这可能是调用者的问题，不应该在这里调用 parsePattern
			// 返回一个更友好的错误信息
			return nil, fmt.Errorf("parsePattern called at fat_arrow position (pattern parsing should have completed, current position: %d)", 
				ep.tokenStream.Position())
		}
		
		// ✅ 修复：对于非标识符类型的 token，尝试解析为字面量
		// 但需要限制解析范围，在遇到 fat_arrow 时停止
		// 注意：这里不应该调用 parseExpression，因为它会消耗太多 token
		// 应该只解析当前 token 作为字面量 pattern
		switch currentToken.Type() {
		case lexicalVO.EnhancedTokenTypeNumber:
			// 数字字面量 pattern
			numberLit := currentToken.NumberLiteral()
			if numberLit == nil {
				return nil, fmt.Errorf("invalid number literal in pattern: %s", currentToken.Lexeme())
			}
			// 尝试转换为整数
			if intVal, ok := numberLit.Value().(int64); ok {
				ep.tokenStream.Next() // 消耗数字
				return &entities.IntLiteral{Value: int(intVal)}, nil
			}
			// 如果是浮点数，也尝试处理
			if floatVal, ok := numberLit.Value().(float64); ok {
				ep.tokenStream.Next() // 消耗数字
				return &entities.FloatLiteral{Value: floatVal}, nil
			}
			return nil, fmt.Errorf("unsupported number literal type in pattern: %s", currentToken.Lexeme())
			
		case lexicalVO.EnhancedTokenTypeString:
			// 字符串字面量 pattern
			value := currentToken.StringValue()
			ep.tokenStream.Next() // 消耗字符串
			return &entities.StringLiteral{Value: value}, nil
			
		case lexicalVO.EnhancedTokenTypeBool:
			// 布尔字面量 pattern
			boolVal := currentToken.BoolValue()
			if boolVal == nil {
				return nil, fmt.Errorf("invalid boolean literal in pattern: %s", currentToken.Lexeme())
			}
			ep.tokenStream.Next() // 消耗布尔值
			return &entities.BoolLiteral{Value: *boolVal}, nil
			
		default:
			// ✅ 关键修复：对于其他类型，不应该调用 parseExpression
			// 因为 parseExpression 会消耗太多 token，导致 token stream 位置不对
			// 如果 parsePattern 进入了 default 分支，说明当前 token 类型不支持作为 pattern
			// 应该返回错误，而不是尝试解析为表达式
			return nil, fmt.Errorf("unsupported token type in pattern: %s (expected identifier, number, string, or bool)", currentToken.Type())
		}
	}
}

// parseMatchCase 解析 match case
func (ep *TokenBasedExpressionParser) parseMatchCase() (entities.MatchCase, error) {
	currentToken := ep.tokenStream.Current()
	log.Printf("[DEBUG] parseMatchCase: Starting, current token=%s (type=%s)", 
		func() string { if currentToken != nil { return currentToken.Lexeme() } else { return "nil" } }(),
		func() string { if currentToken != nil { return string(currentToken.Type()) } else { return "nil" } }())
	
	// 跳过分号和换行
	if ep.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) {
		ep.tokenStream.Next()
	}

	// 解析 pattern（使用专门的 pattern 解析方法）
	pattern, err := ep.parsePattern()
	if err != nil {
		return entities.MatchCase{}, fmt.Errorf("failed to parse match pattern: %w", err)
	}
	log.Printf("[DEBUG] parseMatchCase: Pattern parsed, type=%T", pattern)

	// 解析 =>
	if !ep.checkToken(lexicalVO.EnhancedTokenTypeFatArrow) {
		currentToken := ep.tokenStream.Current()
		if currentToken != nil {
			return entities.MatchCase{}, fmt.Errorf("expected '=>' after match pattern, got %s", currentToken.Type())
		}
		return entities.MatchCase{}, fmt.Errorf("expected '=>' after match pattern")
	}
	ep.tokenStream.Next() // 消耗 '=>'

	// 解析 body（表达式或语句块）
	var bodyStmts []entities.ASTNode
	
	bodyToken := ep.tokenStream.Current()
	if bodyToken == nil {
		return entities.MatchCase{}, fmt.Errorf("unexpected end of token stream in match case")
	}
	
	log.Printf("[DEBUG] parseMatchCase: After '=>', current token=%s (type=%s)", bodyToken.Lexeme(), bodyToken.Type())
	
	// 解析 body 时标记为 match case body 上下文，避免把下一个 case 的 pattern（如 Err(e)）误解析为表达式
	ep.SetInMatchCaseBody(true)
	defer ep.SetInMatchCaseBody(false)

	// 检查是否是语句块（用 { } 包围）
	if bodyToken.Type() == lexicalVO.EnhancedTokenTypeLeftBrace {
		log.Printf("[DEBUG] parseMatchCase: Parsing body as block")
		// 解析语句块
		if ep.statementParser == nil {
			return entities.MatchCase{}, fmt.Errorf("statement parser not set, cannot parse match case body block")
		}
		// 使用语句解析器解析代码块
		// 注意：需要临时设置 statementParser 的 tokenStream
		if ep.statementParser == nil {
			return entities.MatchCase{}, fmt.Errorf("statement parser not set, cannot parse match case body block")
		}
		originalTokenStream := ep.statementParser.tokenStream
		ep.statementParser.tokenStream = ep.tokenStream
		blockStmts, err := ep.statementParser.parseBlock()
		ep.statementParser.tokenStream = originalTokenStream
		if err != nil {
			return entities.MatchCase{}, fmt.Errorf("failed to parse match case body block: %w", err)
		}
		bodyStmts = blockStmts
		log.Printf("[DEBUG] parseMatchCase: Block parsed, %d statements", len(bodyStmts))
	} else {
		log.Printf("[DEBUG] parseMatchCase: Parsing body as expression")
		// 解析为表达式
		expr, err := ep.parseExpression(0)
		if err != nil {
			return entities.MatchCase{}, fmt.Errorf("failed to parse match case expression: %w", err)
		}
		// 将表达式包装为表达式语句
		bodyStmts = append(bodyStmts, &entities.ExprStmt{
			Expression: expr,
		})
	}

	return entities.MatchCase{
		Pattern: pattern,
		Body:    bodyStmts,
	}, nil
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
	case lexicalVO.EnhancedTokenTypeDot:
		// 结构体访问和方法调用，优先级最高（绑定最紧密）
		return 10
	case lexicalVO.EnhancedTokenTypeAs:
		// 类型转换，优先级9（高于赋值，低于方法调用）
		return 9
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

	// 如果遇到右大括号，停止解析（用于代码块结束，表达式应该在此之前结束）
	// 注意：这是语句终止符，但为了安全，在这里也检查
	if token.Type() == lexicalVO.EnhancedTokenTypeRightBrace {
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

	// 检查是否是后缀运算符（错误传播操作符 ?）
	if token.Type() == lexicalVO.EnhancedTokenTypeErrorPropagation {
		return true // 后缀运算符，继续解析
	}

	// ============ 关键修复：特殊处理 Identifier（可能是方法调用） ============
	// 如果当前token是 Identifier，检查是否是方法调用的一部分
	// 例如：v.data.push(item) 中，在创建 StructAccess 后，Current() 是 push（Identifier）
	// 如果 push 后面是 '('，则是方法调用，应该继续解析
	if token.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
		log.Printf("[DEBUG] shouldContinueParsing: Found Identifier %s, checking if method call", token.Lexeme())
		// 使用 SetPosition + Position 实现 lookahead
		savedPosition := ep.tokenStream.Position()
		ep.tokenStream.Next() // 移动到下一个token
		nextToken := ep.tokenStream.Current()
		ep.tokenStream.SetPosition(savedPosition) // 回退
		
		log.Printf("[DEBUG] shouldContinueParsing: Lookahead result: nextToken=%v (type=%s)", nextToken, nextToken.Type())
		
		if nextToken != nil && nextToken.Type() == lexicalVO.EnhancedTokenTypeLeftParen {
			// Identifier + '(' 表示方法调用，应该继续解析
			log.Printf("[DEBUG] shouldContinueParsing: Identifier %s is followed by '(', returning true", token.Lexeme())
			return true
		}
		// 否则，Identifier 不是运算符，停止解析
		log.Printf("[DEBUG] shouldContinueParsing: Identifier %s is NOT followed by '(', returning false", token.Lexeme())
		return false
	}
	// ============ 修复结束 ============

	// 检查是否是类型转换关键字 'as'
	if token.Type() == lexicalVO.EnhancedTokenTypeAs {
		// 类型转换的优先级是9，只有当 minPrecedence <= 9 时才继续解析
		return 9 >= minPrecedence
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

// parseTernaryExpression 解析三元表达式（条件表达式）
// 格式：if condition { value1 } else { value2 }
func (ep *TokenBasedExpressionParser) parseTernaryExpression() (entities.Expr, error) {
	ep.tokenStream.Next() // 消耗 'if'

	// 解析条件表达式
	condition, err := ep.parseExpression(0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ternary condition: %w", err)
	}

	// 解析 then 分支：{ value1 }
	if !ep.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after ternary condition")
	}
	ep.tokenStream.Next() // 消耗 '{'

	// 解析 then 值（单个表达式）
	thenValue, err := ep.parseExpression(0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ternary then value: %w", err)
	}

	if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
		return nil, fmt.Errorf("expected '}' after ternary then value")
	}
	ep.tokenStream.Next() // 消耗 '}'

	// 解析 else 分支：else { value2 }
	if !ep.checkToken(lexicalVO.EnhancedTokenTypeElse) {
		return nil, fmt.Errorf("expected 'else' after ternary then branch")
	}
	ep.tokenStream.Next() // 消耗 'else'

	if !ep.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after 'else'")
	}
	ep.tokenStream.Next() // 消耗 '{'

	// 解析 else 值（单个表达式）
	elseValue, err := ep.parseExpression(0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ternary else value: %w", err)
	}

	if !ep.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
		return nil, fmt.Errorf("expected '}' after ternary else value")
	}
	ep.tokenStream.Next() // 消耗 '}'

	return &entities.TernaryExpr{
		Condition: condition,
		ThenValue: thenValue,
		ElseValue: elseValue,
	}, nil
}

// parseNamespaceAccess 解析命名空间访问：net::bind
func (ep *TokenBasedExpressionParser) parseNamespaceAccess(namespaceToken *lexicalVO.EnhancedToken) (entities.Expr, error) {
	// 消耗 ::
	ep.tokenStream.Next()
	
	// 解析命名空间内的标识符
	if !ep.checkToken(lexicalVO.EnhancedTokenTypeIdentifier) {
		return nil, fmt.Errorf("expected identifier after '::'")
	}
	memberToken := ep.tokenStream.Current()
	memberName := memberToken.Lexeme()
	ep.tokenStream.Next() // 消耗成员名
	
	// 检查是否是函数调用
	nextToken := ep.tokenStream.Current()
	if nextToken != nil && nextToken.Type() == lexicalVO.EnhancedTokenTypeLeftParen {
		// 命名空间函数调用：net::bind(...)
		// 将命名空间访问转换为函数调用，函数名为 "namespace_member"
		// 例如：net::bind(...) -> net_bind(...)
		funcName := namespaceToken.Lexeme() + "_" + memberName
		// 创建一个新的 token 用于函数调用
		funcToken := lexicalVO.NewEnhancedToken(
			lexicalVO.EnhancedTokenTypeIdentifier,
			funcName,
			namespaceToken.Location(),
		)
		return ep.parseFunctionCall(funcToken)
	}
	
	// 返回命名空间访问表达式（非函数调用情况）
	return &entities.NamespaceAccessExpr{
		Namespace: namespaceToken.Lexeme(),
		Member:    memberName,
	}, nil
}
