package analysis

// Parser 语法分析器接口
type Parser interface {
	// Parse 将Token序列解析为AST
	Parse(tokens []Token) (*Program, []Diagnostic, error)
}

// SimpleParser 简单语法分析器实现
type SimpleParser struct {
	tokens  []Token
	current int
}

// NewSimpleParser 创建新的简单语法分析器
func NewSimpleParser() Parser {
	return &SimpleParser{}
}

// Parse 实现Parser接口
func (p *SimpleParser) Parse(tokens []Token) (*Program, []Diagnostic, error) {
	p.tokens = tokens
	p.current = 0

	var diagnostics []Diagnostic
	var statements []ASTNode

	for !p.isAtEnd() {
		stmt, diag := p.parseStatement()
		if diag != nil {
			diagnostics = append(diagnostics, *diag)
			// 错误恢复：跳到下一个语句
			p.advanceToNextStatement()
			continue
		}
		if stmt != nil {
			statements = append(statements, stmt)
		}
	}

	program := NewProgram(statements, p.position(0))
	return program, diagnostics, nil
}

// parseStatement 解析语句
func (p *SimpleParser) parseStatement() (ASTNode, *Diagnostic) {
	if p.isAtEnd() {
		return nil, nil
	}

	token := p.peek()
	switch token.Kind() {
	case Function:
		return p.parseFunctionDef()
	case Let:
		return p.parseVariableDecl()
	case If:
		return p.parseIfStmt()
	case For:
		return p.parseForStmt()
	case While:
		return p.parseWhileStmt()
	case Return:
		return p.parseReturnStmt()
	case Print:
		return p.parsePrintStmt()
	case Match:
		return p.parseMatchExpr()
	case Identifier:
		// 检查是否是函数调用语句
		if p.checkNext(LeftParen) {
			expr, diag := p.parseFunctionCallExpr()
			if diag != nil {
				return nil, diag
			}
			return NewExprStmt(expr, expr.Position()), nil
		}
		fallthrough
	default:
		// 解析表达式语句
		expr, diag := p.parseExpr()
		if diag != nil {
			return nil, diag
		}
		if !p.isAtEnd() && !p.check(Semicolon) {
			return nil, p.error("Expected ';' after expression", p.peek().Position())
		}
		if p.check(Semicolon) {
			p.advance()
		}
		return NewExprStmt(expr, expr.Position()), nil
	}
}

// parseFunctionDef 解析函数定义
func (p *SimpleParser) parseFunctionDef() (ASTNode, *Diagnostic) {
	pos := p.peek().Position()
	p.advance() // consume 'func'

	// 函数名
	if !p.check(Identifier) {
		return nil, p.error("Expected function name", p.peek().Position())
	}
	name := p.peek().Value()
	p.advance()

	// 参数列表
	if !p.check(LeftParen) {
		return nil, p.error("Expected '(' after function name", p.peek().Position())
	}
	p.advance()

	var params []Parameter
	if !p.check(RightParen) {
		for {
			if !p.check(Identifier) {
				return nil, p.error("Expected parameter name", p.peek().Position())
			}
			paramName := p.peek().Value()
			p.advance()

			if !p.check(Colon) {
				return nil, p.error("Expected ':' after parameter name", p.peek().Position())
			}
			p.advance()

			type_, diag := p.parseType()
			if diag != nil {
				return nil, diag
			}

			params = append(params, NewParameter(paramName, type_))

			if !p.check(Comma) {
				break
			}
			p.advance()
		}
	}

	if !p.check(RightParen) {
		return nil, p.error("Expected ')' after parameters", p.peek().Position())
	}
	p.advance()

	// 返回类型（可选）
	var returnType *TypeAnnotation
	if p.check(Minus) && p.checkNext(Greater) {
		p.advance() // consume '-'
		p.advance() // consume '>'
		type_, diag := p.parseType()
		if diag != nil {
			return nil, diag
		}
		returnType = type_
	}

	// 函数体
	body, diag := p.parseBlock()
	if diag != nil {
		return nil, diag
	}

	return NewFunctionDef(name, params, returnType, body, pos), nil
}

// parseVariableDecl 解析变量声明
func (p *SimpleParser) parseVariableDecl() (ASTNode, *Diagnostic) {
	pos := p.peek().Position()
	p.advance() // consume 'let'

	if !p.check(Identifier) {
		return nil, p.error("Expected variable name", p.peek().Position())
	}
	name := p.peek().Value()
	p.advance()

	var type_ *TypeAnnotation
	var initExpr ASTNode

	// 可选类型注解
	if p.check(Colon) {
		p.advance()
		var diag *Diagnostic
		type_, diag = p.parseType()
		if diag != nil {
			return nil, diag
		}
	}

	// 可选初始化表达式
	if p.check(Assign) {
		p.advance()
		var diag *Diagnostic
		initExpr, diag = p.parseExpr()
		if diag != nil {
			return nil, diag
		}
	}

	if !p.check(Semicolon) {
		return nil, p.error("Expected ';' after variable declaration", p.peek().Position())
	}
	p.advance()

	return NewVariableDecl(name, type_, initExpr, pos), nil
}

// parseIfStmt 解析if语句
func (p *SimpleParser) parseIfStmt() (ASTNode, *Diagnostic) {
	pos := p.peek().Position()
	p.advance() // consume 'if'

	condition, diag := p.parseExpr()
	if diag != nil {
		return nil, diag
	}

	thenBody, diag := p.parseBlock()
	if diag != nil {
		return nil, diag
	}

	var elseBody *Block
	if p.check(Else) {
		p.advance()
		elseBody, diag = p.parseBlock()
		if diag != nil {
			return nil, diag
		}
	}

	return NewIfStmt(condition, thenBody, elseBody, pos), nil
}

// parseForStmt 解析for循环
func (p *SimpleParser) parseForStmt() (ASTNode, *Diagnostic) {
	pos := p.peek().Position()
	p.advance() // consume 'for'

	// for循环语法：for init; condition; update { body }
	var init ASTNode
	var condition ASTNode
	var update ASTNode

	// 初始化语句（可选）
	if !p.check(Semicolon) {
		var diag *Diagnostic
		init, diag = p.parseStatement()
		if diag != nil {
			return nil, diag
		}
	}

	if !p.check(Semicolon) {
		return nil, p.error("Expected ';' after for init", p.peek().Position())
	}
	p.advance()

	// 条件表达式（可选）
	if !p.check(Semicolon) {
		var diag *Diagnostic
		condition, diag = p.parseExpr()
		if diag != nil {
			return nil, diag
		}
	}

	if !p.check(Semicolon) {
		return nil, p.error("Expected ';' after for condition", p.peek().Position())
	}
	p.advance()

	// 更新表达式（可选）
	if !p.check(LeftBrace) {
		var diag *Diagnostic
		update, diag = p.parseExpr()
		if diag != nil {
			return nil, diag
		}
	}

	body, diag := p.parseBlock()
	if diag != nil {
		return nil, diag
	}

	return NewForStmt(init, condition, update, body, pos), nil
}

// parseWhileStmt 解析while循环
func (p *SimpleParser) parseWhileStmt() (ASTNode, *Diagnostic) {
	pos := p.peek().Position()
	p.advance() // consume 'while'

	condition, diag := p.parseExpr()
	if diag != nil {
		return nil, diag
	}

	body, diag := p.parseBlock()
	if diag != nil {
		return nil, diag
	}

	return NewWhileStmt(condition, body, pos), nil
}

// parseReturnStmt 解析return语句
func (p *SimpleParser) parseReturnStmt() (ASTNode, *Diagnostic) {
	pos := p.peek().Position()
	p.advance() // consume 'return'

	var expr ASTNode
	if !p.isAtEnd() && !p.check(Semicolon) && !p.check(RightBrace) {
		var diag *Diagnostic
		expr, diag = p.parseExpr()
		if diag != nil {
			return nil, diag
		}
	}

	// 在语句上下文中，分号是可选的
	if p.check(Semicolon) {
		p.advance()
	}

	return NewReturnStmt(expr, pos), nil
}

// parsePrintStmt 解析print语句
func (p *SimpleParser) parsePrintStmt() (ASTNode, *Diagnostic) {
	pos := p.peek().Position()
	p.advance() // consume 'print'

	if !p.check(LeftParen) {
		return nil, p.error("Expected '(' after print", p.peek().Position())
	}
	p.advance()

	expr, diag := p.parseExpr()
	if diag != nil {
		return nil, diag
	}

	if !p.check(RightParen) {
		return nil, p.error("Expected ')' after print expression", p.peek().Position())
	}
	p.advance()

	if !p.check(Semicolon) {
		return nil, p.error("Expected ';' after print statement", p.peek().Position())
	}
	p.advance()

	// 转换为函数调用表达式
	printIdent := NewIdentifierExpr("print", pos)
	args := []ASTNode{expr}
	callExpr := NewFunctionCallExpr(printIdent, args, pos)
	return NewExprStmt(callExpr, pos), nil
}

// parseBlock 解析代码块
func (p *SimpleParser) parseBlock() (*Block, *Diagnostic) {
	pos := p.peek().Position()

	if !p.check(LeftBrace) {
		return nil, p.error("Expected '{' to start block", p.peek().Position())
	}
	p.advance()

	var statements []ASTNode
	for !p.isAtEnd() && !p.check(RightBrace) {
		stmt, diag := p.parseStatement()
		if diag != nil {
			return nil, diag
		}
		if stmt != nil {
			statements = append(statements, stmt)
		}
	}

	if !p.check(RightBrace) {
		return nil, p.error("Expected '}' to end block", p.peek().Position())
	}
	p.advance()

	return NewBlock(statements, pos), nil
}

// parseExpr 解析表达式
func (p *SimpleParser) parseExpr() (ASTNode, *Diagnostic) {
	return p.parseAssignExpr()
}

// parseAssignExpr 解析赋值表达式
func (p *SimpleParser) parseAssignExpr() (ASTNode, *Diagnostic) {
	expr, diag := p.parseOrExpr()
	if diag != nil {
		return nil, diag
	}

	if p.check(Assign) {
		pos := p.peek().Position()
		p.advance()

		value, diag := p.parseAssignExpr()
		if diag != nil {
			return nil, diag
		}

		return NewAssignExpr(expr, value, pos), nil
	}

	return expr, nil
}

// parseOrExpr 解析或表达式
func (p *SimpleParser) parseOrExpr() (ASTNode, *Diagnostic) {
	expr, diag := p.parseAndExpr()
	if diag != nil {
		return nil, diag
	}

	for p.check(Or) {
		pos := p.peek().Position()
		operator := p.peek()
		p.advance()

		right, diag := p.parseAndExpr()
		if diag != nil {
			return nil, diag
		}

		expr = NewBinaryExpr(expr, operator, right, pos)
	}

	return expr, nil
}

// parseAndExpr 解析与表达式
func (p *SimpleParser) parseAndExpr() (ASTNode, *Diagnostic) {
	expr, diag := p.parseComparisonExpr()
	if diag != nil {
		return nil, diag
	}

	for p.check(And) {
		pos := p.peek().Position()
		operator := p.peek()
		p.advance()

		right, diag := p.parseComparisonExpr()
		if diag != nil {
			return nil, diag
		}

		expr = NewBinaryExpr(expr, operator, right, pos)
	}

	return expr, nil
}

// parseComparisonExpr 解析比较表达式
func (p *SimpleParser) parseComparisonExpr() (ASTNode, *Diagnostic) {
	expr, diag := p.parseTermExpr()
	if diag != nil {
		return nil, diag
	}

	for p.isComparisonOperator(p.peek()) {
		pos := p.peek().Position()
		operator := p.peek()
		p.advance()

		right, diag := p.parseTermExpr()
		if diag != nil {
			return nil, diag
		}

		expr = NewBinaryExpr(expr, operator, right, pos)
	}

	return expr, nil
}

// parseTermExpr 解析项表达式（加减）
func (p *SimpleParser) parseTermExpr() (ASTNode, *Diagnostic) {
	expr, diag := p.parseFactorExpr()
	if diag != nil {
		return nil, diag
	}

	for p.isTermOperator(p.peek()) {
		pos := p.peek().Position()
		operator := p.peek()
		p.advance()

		right, diag := p.parseFactorExpr()
		if diag != nil {
			return nil, diag
		}

		expr = NewBinaryExpr(expr, operator, right, pos)
	}

	return expr, nil
}

// parseFactorExpr 解析因子表达式（乘除）
func (p *SimpleParser) parseFactorExpr() (ASTNode, *Diagnostic) {
	expr, diag := p.parseUnaryExpr()
	if diag != nil {
		return nil, diag
	}

	for p.isFactorOperator(p.peek()) {
		pos := p.peek().Position()
		operator := p.peek()
		p.advance()

		right, diag := p.parseUnaryExpr()
		if diag != nil {
			return nil, diag
		}

		expr = NewBinaryExpr(expr, operator, right, pos)
	}

	return expr, nil
}

// parseUnaryExpr 解析一元表达式
func (p *SimpleParser) parseUnaryExpr() (ASTNode, *Diagnostic) {
	if p.check(Not) || p.check(Minus) {
		pos := p.peek().Position()
		operator := p.peek()
		p.advance()

		operand, diag := p.parseUnaryExpr()
		if diag != nil {
			return nil, diag
		}

		return NewUnaryExpr(operator, operand, pos), nil
	}

	return p.parsePrimaryExpr()
}

// parsePrimaryExpr 解析基本表达式
func (p *SimpleParser) parsePrimaryExpr() (ASTNode, *Diagnostic) {
	token := p.peek()

	switch token.Kind() {
	case IntLiteral:
		p.advance()
		return NewLiteralExpr(token.Value(), IntLiteral, token.Position()), nil

	case FloatLiteral:
		p.advance()
		return NewLiteralExpr(token.Value(), FloatLiteral, token.Position()), nil

	case StringLiteral:
		p.advance()
		return NewLiteralExpr(token.Value(), StringLiteral, token.Position()), nil

	case BoolLiteral:
		p.advance()
		return NewLiteralExpr(token.Value(), BoolLiteral, token.Position()), nil

	case Identifier:
		if p.checkNext(LeftParen) {
			return p.parseFunctionCallExpr()
		}
		p.advance()
		return NewIdentifierExpr(token.Value(), token.Position()), nil

	case LeftParen:
		p.advance()
		expr, diag := p.parseExpr()
		if diag != nil {
			return nil, diag
		}
		if !p.check(RightParen) {
			return nil, p.error("Expected ')' after expression", p.peek().Position())
		}
		p.advance()
		return expr, nil

	case LeftBracket:
		return p.parseArrayLiteralExpr()

	case LeftBrace:
		return p.parseStructLiteralExpr()

	default:
		return nil, p.error("Unexpected token: " + token.String(), token.Position())
	}
}

// parseFunctionCallExpr 解析函数调用表达式
func (p *SimpleParser) parseFunctionCallExpr() (ASTNode, *Diagnostic) {
	pos := p.peek().Position()
	function := NewIdentifierExpr(p.peek().Value(), p.peek().Position())
	p.advance() // consume function name

	if !p.check(LeftParen) {
		return nil, p.error("Expected '(' after function name", p.peek().Position())
	}
	p.advance()

	var arguments []ASTNode
	if !p.check(RightParen) {
		for {
			arg, diag := p.parseExpr()
			if diag != nil {
				return nil, diag
			}
			arguments = append(arguments, arg)

			if !p.check(Comma) {
				break
			}
			p.advance()
		}
	}

	if !p.check(RightParen) {
		return nil, p.error("Expected ')' after arguments", p.peek().Position())
	}
	p.advance()

	return NewFunctionCallExpr(function, arguments, pos), nil
}

// parseArrayLiteralExpr 解析数组字面量
func (p *SimpleParser) parseArrayLiteralExpr() (ASTNode, *Diagnostic) {
	pos := p.peek().Position()
	p.advance() // consume '['

	var elements []ASTNode
	if !p.check(RightBracket) {
		for {
			elem, diag := p.parseExpr()
			if diag != nil {
				return nil, diag
			}
			elements = append(elements, elem)

			if !p.check(Comma) {
				break
			}
			p.advance()
		}
	}

	if !p.check(RightBracket) {
		return nil, p.error("Expected ']' after array elements", p.peek().Position())
	}
	p.advance()

	return NewArrayLiteralExpr(elements, pos), nil
}

// parseStructLiteralExpr 解析结构体字面量
func (p *SimpleParser) parseStructLiteralExpr() (ASTNode, *Diagnostic) {
	pos := p.peek().Position()
	p.advance() // consume '{'

	var fields []StructField
	if !p.check(RightBrace) {
		for {
			if !p.check(Identifier) {
				return nil, p.error("Expected field name", p.peek().Position())
			}
			fieldName := p.peek().Value()
			p.advance()

			if !p.check(Colon) {
				return nil, p.error("Expected ':' after field name", p.peek().Position())
			}
			p.advance()

			fieldValue, diag := p.parseExpr()
			if diag != nil {
				return nil, diag
			}

			fields = append(fields, NewStructField(fieldName, fieldValue))

			if !p.check(Comma) {
				break
			}
			p.advance()
		}
	}

	if !p.check(RightBrace) {
		return nil, p.error("Expected '}' after struct fields", p.peek().Position())
	}
	p.advance()

	// 简单起见，使用空类型名，后续可以扩展
	return NewStructLiteralExpr("", fields, pos), nil
}

// parseType 解析类型注解
func (p *SimpleParser) parseType() (*TypeAnnotation, *Diagnostic) {
	if !p.check(Identifier) {
		return nil, p.error("Expected type name", p.peek().Position())
	}

	typeName := p.peek().Value()
	pos := p.peek().Position()
	p.advance()

	return NewTypeAnnotation(typeName, pos), nil
}

// 辅助方法

func (p *SimpleParser) isAtEnd() bool {
	return p.current >= len(p.tokens) || p.peek().IsEOF()
}

func (p *SimpleParser) peek() Token {
	return p.tokens[p.current]
}

func (p *SimpleParser) check(kind TokenKind) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek().Kind() == kind
}

func (p *SimpleParser) checkNext(kind TokenKind) bool {
	if p.current+1 >= len(p.tokens) {
		return false
	}
	return p.tokens[p.current+1].Kind() == kind
}

func (p *SimpleParser) advance() Token {
	if !p.isAtEnd() {
		p.current++
	}
	return p.tokens[p.current-1]
}

func (p *SimpleParser) position(offset int) Position {
	idx := p.current + offset
	if idx >= 0 && idx < len(p.tokens) {
		return p.tokens[idx].Position()
	}
	if len(p.tokens) > 0 {
		return p.tokens[len(p.tokens)-1].Position()
	}
	return NewPosition(1, 1, "unknown")
}

func (p *SimpleParser) isComparisonOperator(token Token) bool {
	switch token.Kind() {
	case Equal, NotEqual, Less, Greater, LessEqual, GreaterEqual:
		return true
	default:
		return false
	}
}

func (p *SimpleParser) isTermOperator(token Token) bool {
	switch token.Kind() {
	case Plus, Minus:
		return true
	default:
		return false
	}
}

func (p *SimpleParser) isFactorOperator(token Token) bool {
	switch token.Kind() {
	case Multiply, Divide:
		return true
	default:
		return false
	}
}

func (p *SimpleParser) error(message string, position Position) *Diagnostic {
	return &Diagnostic{
		type_:    Error,
		message:  message,
		position: position,
	}
}

func (p *SimpleParser) advanceToNextStatement() {
	for !p.isAtEnd() {
		token := p.peek()
		p.advance()

		if token.Kind() == Semicolon || token.Kind() == RightBrace {
			break
		}
	}
}

// parseMatchExpr 解析模式匹配表达式
func (p *SimpleParser) parseMatchExpr() (ASTNode, *Diagnostic) {
	// 消耗 'match' 关键字
	p.advance() // skip 'match'

	// 解析匹配表达式
	expr, diag := p.parseExpr()
	if diag != nil {
		return nil, diag
	}

	// 期望左大括号
	if !p.check(LeftBrace) {
		return nil, p.error("Expected '{' after match expression", p.peek().Position())
	}
	p.advance() // skip '{'

	var cases []MatchCase

	// 解析匹配分支
	for !p.isAtEnd() && !p.check(RightBrace) {
		case_, diag := p.parseMatchCase()
		if diag != nil {
			return nil, diag
		}
		cases = append(cases, case_)

		// 如果后面还有逗号，消耗它（可选）
		if p.check(Comma) {
			p.advance()
		} else if !p.check(RightBrace) {
			// 如果没有逗号，也没有右大括号，说明语法错误
			return nil, p.error("Expected ',' or '}' after match case", p.peek().Position())
		}
	}

	// 期望右大括号
	if !p.check(RightBrace) {
		return nil, p.error("Expected '}' after match cases", p.peek().Position())
	}
	p.advance() // skip '}'

	return NewMatchExpr(expr, cases, expr.Position()), nil
}

// parseMatchCase 解析匹配分支
func (p *SimpleParser) parseMatchCase() (MatchCase, *Diagnostic) {
	// 解析模式
	pattern, diag := p.parseExpr()
	if diag != nil {
		return MatchCase{}, diag
	}

	// 期望 '=>'
	if !p.check(FatArrow) {
		return MatchCase{}, p.error("Expected '=>' in match case", p.peek().Position())
	}
	p.advance() // skip '=>'

	// 解析分支体
	body, diag := p.parseExpr()
	if diag != nil {
		return MatchCase{}, diag
	}

	return NewMatchCase(pattern, body), nil
}
