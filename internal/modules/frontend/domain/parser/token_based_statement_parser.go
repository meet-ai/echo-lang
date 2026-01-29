package parser

import (
	"fmt"
	"log"
	"strings"

	"echo/internal/modules/frontend/domain/entities"
	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
)

// TokenBasedStatementParser 基于 Token 的语句解析器
// 使用递归下降解析算法处理语句结构
// 直接使用 EnhancedTokenStream 管理位置，不维护自己的 position
type TokenBasedStatementParser struct {
	tokenStream      *lexicalVO.EnhancedTokenStream
	expressionParser *TokenBasedExpressionParser
}

// NewTokenBasedStatementParser 创建新的基于 Token 的语句解析器
func NewTokenBasedStatementParser() *TokenBasedStatementParser {
	return &TokenBasedStatementParser{
		expressionParser: NewTokenBasedExpressionParser(),
	}
}

// ParseStatement 解析语句（主入口）
// 直接使用 tokenStream 的当前位置，解析完成后 tokenStream 的位置会自动更新
func (sp *TokenBasedStatementParser) ParseStatement(tokenStream *lexicalVO.EnhancedTokenStream) (entities.ASTNode, error) {
	sp.tokenStream = tokenStream

	currentToken := tokenStream.Current()
	if currentToken == nil || tokenStream.IsAtEnd() {
		return nil, nil // 到达末尾，返回 nil 表示没有更多语句
	}

	token := currentToken
	if token == nil {
		return nil, fmt.Errorf("unexpected end of token stream")
	}

	log.Printf("[DEBUG] ParseStatement: current token=%v (type=%s), position=%d, line=%d", token, token.Type(), tokenStream.Position(), token.Location().Line())

	// 根据 token 类型识别语句类型
	switch token.Type() {
	case lexicalVO.EnhancedTokenTypePrint:
		return sp.parsePrintStatement()
	case lexicalVO.EnhancedTokenTypeLet:
		return sp.parseVariableDeclaration()
	case lexicalVO.EnhancedTokenTypeIf:
		return sp.parseIfStatement()
	case lexicalVO.EnhancedTokenTypeWhile:
		return sp.parseWhileStatement()
	case lexicalVO.EnhancedTokenTypeLoop:
		return sp.parseLoopStatement()
	case lexicalVO.EnhancedTokenTypeFor:
		return sp.parseForStatement()
	case lexicalVO.EnhancedTokenTypePrivate:
		// ✅ 修复：private 关键字修饰符，跳过并解析下一个语句（通常是 func）
		sp.tokenStream.Next() // 消耗 'private'
		// 递归解析下一个语句（通常是 func）
		return sp.ParseStatement(sp.tokenStream)
	case lexicalVO.EnhancedTokenTypeFunc, lexicalVO.EnhancedTokenTypeAsync:
		log.Printf("[DEBUG] ParseStatement: Calling parseFunctionDefinition, current token=%v, position=%d", token, tokenStream.Position())
		return sp.parseFunctionDefinition()
	case lexicalVO.EnhancedTokenTypeStruct:
		return sp.parseStructDefinition()
	case lexicalVO.EnhancedTokenTypeEnum:
		return sp.parseEnumDefinition()
	case lexicalVO.EnhancedTokenTypeTrait:
		return sp.parseTraitDefinition()
	case lexicalVO.EnhancedTokenTypeImpl:
		return sp.parseImplAnnotation()
	case lexicalVO.EnhancedTokenTypeMatch:
		return sp.parseMatchStatement()
	case lexicalVO.EnhancedTokenTypeReturn:
		return sp.parseReturnStatement()
	case lexicalVO.EnhancedTokenTypeDelete:
		return sp.parseDeleteStatement()
	case lexicalVO.EnhancedTokenTypeSpawn:
		return sp.parseSpawnStatement()
	case lexicalVO.EnhancedTokenTypeSelect:
		return sp.parseSelectStatement()
	// break 和 continue 可能作为标识符处理，暂时跳过
	// case lexicalVO.EnhancedTokenTypeBreak:
	// case lexicalVO.EnhancedTokenTypeContinue:
	case lexicalVO.EnhancedTokenTypeLeftBrace:
		// 左大括号表示一个代码块的开始
		// 在语句解析的上下文中，LeftBrace 应该由 parseBlock 处理
		// 但如果在这里遇到，可能是嵌套的代码块，或者需要解析一个独立的代码块
		// 暂时尝试解析一个代码块
		return sp.parseBlockAsStatement()
	case lexicalVO.EnhancedTokenTypeRightBrace:
		// 块结束符，返回 nil 表示忽略
		return nil, nil
	case lexicalVO.EnhancedTokenTypeIdentifier:
		// 检查是否是类型别名：type Item = int;
		if token.Lexeme() == "type" {
			return sp.parseTypeAlias()
		}
		// ✅ 修复：检查是否是 import 语句：import <path> [as <alias>]
		if token.Lexeme() == "import" {
			log.Printf("[DEBUG] ParseStatement: Found 'import' identifier, calling parseImportStatement")
			return sp.parseImportStatement()
		}
		// break/continue 词法分析为 identifier，在此识别为控制流语句，生成 BreakStmt/ContinueStmt
		if token.Lexeme() == "break" {
			sp.tokenStream.Next() // 消耗 "break"
			return &entities.BreakStmt{}, nil
		}
		if token.Lexeme() == "continue" {
			sp.tokenStream.Next() // 消耗 "continue"
			return &entities.ContinueStmt{}, nil
		}
		// 可能是赋值语句或表达式语句
		return sp.parseAssignmentOrExpressionStatement()
	case lexicalVO.EnhancedTokenTypeAs:
		// ✅ 修复：as 关键字不应该出现在语句开始（除非是 import ... as ...）
		// 如果 as 出现在语句开始，说明前面的 import 语句解析有问题
		return nil, fmt.Errorf("unexpected 'as' at statement start (expected 'as' after import path)")
	case lexicalVO.EnhancedTokenTypeFrom:
		// ✅ 修复：from ... import 语句
		return sp.parseFromImportStatement()
	case lexicalVO.EnhancedTokenTypeLeftParen:
		// 左括号不应该出现在语句开始，可能是表达式语句
		// 或者是在解析过程中 token stream 位置不对
		// 尝试解析为表达式语句
		return sp.parseAssignmentOrExpressionStatement()
	case lexicalVO.EnhancedTokenTypeErrorPropagation:
		// ? 错误传播操作符不应该出现在语句开始
		// 这可能是表达式的一部分，尝试解析为表达式语句
		return sp.parseAssignmentOrExpressionStatement()
	case lexicalVO.EnhancedTokenTypeSemicolon:
		// 分号作为语句分隔符，返回 nil 表示忽略
		sp.tokenStream.Next() // 消耗分号
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected token in statement: %s", token.Type())
	}
}

// parsePrintStatement 解析 print 语句
func (sp *TokenBasedStatementParser) parsePrintStatement() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 'print'

	// 解析表达式
	expr, err := sp.expressionParser.ParseExpression(sp.tokenStream)
	if err != nil {
		return nil, err
	}

	return &entities.PrintStmt{
		Value: expr,
	}, nil
}

// parseSpawnStatement 解析 spawn 语句
// 格式：spawn <expression>
func (sp *TokenBasedStatementParser) parseSpawnStatement() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 'spawn'

	// 解析表达式（通常是函数调用）
	expr, err := sp.expressionParser.ParseExpression(sp.tokenStream)
	if err != nil {
		return nil, err
	}

	// 返回一个表达式语句
	return &entities.ExprStmt{
		Expression: expr,
	}, nil
}

// parseVariableDeclaration 解析变量声明
// 支持格式：let [mut] name [: type] = value
func (sp *TokenBasedStatementParser) parseVariableDeclaration() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 'let'

	// 检查是否有 'mut' 关键字（跳过，不影响解析）
	if sp.checkToken(lexicalVO.EnhancedTokenTypeIdentifier) {
		currentToken := sp.tokenStream.Current()
		if currentToken != nil && currentToken.Lexeme() == "mut" {
			sp.tokenStream.Next() // 消耗 'mut'
		}
	}

	// 解析变量名
	nameToken := sp.tokenStream.Current()
	if nameToken == nil || nameToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
		return nil, fmt.Errorf("expected identifier after 'let' or 'mut'")
	}
	name := nameToken.Lexeme()
	sp.tokenStream.Next() // 消耗变量名

	// 解析类型（可选）
	var varType string
	var inferred bool
	if sp.checkToken(lexicalVO.EnhancedTokenTypeColon) {
		sp.tokenStream.Next() // 消耗 ':'
		typeToken := sp.tokenStream.Current()
		if typeToken == nil {
			return nil, fmt.Errorf("expected type after ':'")
		}

		// 解析类型（支持 chan、标识符、数组、泛型）
		var err error
		varType, err = sp.parseType()
		if err != nil {
			return nil, err
		}
		inferred = false
	} else {
		inferred = true
	}

	// 解析初始值
	var value entities.Expr
	if sp.checkToken(lexicalVO.EnhancedTokenTypeAssign) {
		sp.tokenStream.Next() // 消耗 '='
		var err error
		value, err = sp.expressionParser.ParseExpression(sp.tokenStream)
		if err != nil {
			return nil, err
		}
	}

	// 注意：VarDecl 结构体目前没有 Mut 字段，但解析器已经支持识别 mut 关键字
	// 如果需要，可以在 VarDecl 中添加 Mut bool 字段
	return &entities.VarDecl{
		Name:     name,
		Type:     varType,
		Value:    value,
		Inferred: inferred,
	}, nil
}

// parseAssignmentOrExpressionStatement 解析赋值语句或表达式语句
func (sp *TokenBasedStatementParser) parseAssignmentOrExpressionStatement() (entities.ASTNode, error) {
	// 策略：先让表达式解析器解析整个表达式（包括标识符）
	// 然后检查下一个 token 是否是 '='，如果是，就是赋值语句
	// 否则，就是表达式语句

	// 保存当前位置，以便在需要时回退
	// 但由于 tokenStream 不支持回退，我们需要使用不同的策略
	// 方案：先尝试解析表达式，如果成功，检查下一个 token

	// 先尝试解析表达式（从标识符开始）
	log.Printf("[DEBUG] parseAssignmentOrExpressionStatement: Before ParseExpression, current token=%v, position=%d", sp.tokenStream.Current(), sp.tokenStream.Position())
	expr, err := sp.expressionParser.ParseExpression(sp.tokenStream)
	if err != nil {
		log.Printf("[ERROR] parseAssignmentOrExpressionStatement: ParseExpression failed: %v", err)
		return nil, err
	}
	log.Printf("[DEBUG] parseAssignmentOrExpressionStatement: After ParseExpression, expr type=%T, current token=%v, position=%d", expr, sp.tokenStream.Current(), sp.tokenStream.Position())

	// 检查下一个 token 是否是 '=' 或 ':'
	nextToken := sp.tokenStream.Current()
	log.Printf("[DEBUG] parseAssignmentOrExpressionStatement: nextToken=%v (type=%s)", nextToken, nextToken.Type())
	if nextToken != nil && nextToken.Type() == lexicalVO.EnhancedTokenTypeColon {
		// 遇到 ':'，说明这可能是类型注解，但当前 token 不是 'let'，这是语法错误
		// 或者这是结构体字面量的字段，但当前上下文不对
		// 或者这是 select case 的语法（case msg := <- ch1:），但 select 语句还未实现
		return nil, fmt.Errorf("unexpected ':' after identifier (expected '=' for assignment, or this should be a 'let' statement, or this is a 'select case' which is not yet implemented)")
	}
	if nextToken != nil && nextToken.Type() == lexicalVO.EnhancedTokenTypeAssign {
		// 这是赋值语句
		// 赋值目标可以是 Identifier、StructAccess 或 IndexExpr
		// 检查表达式是否是有效的赋值目标
		switch expr.(type) {
		case *entities.Identifier, *entities.StructAccess, *entities.IndexExpr:
			// 有效的赋值目标
		default:
			// 如果表达式不是有效的赋值目标，这可能是解析错误
			// 例如：v.data.push(...) = ... 是语法错误
			return nil, fmt.Errorf("assignment target must be an identifier, struct field access, or array index, got %T (this may indicate a parsing error where the expression parser stopped too early)", expr)
		}

		// 消耗 '='
		sp.tokenStream.Next()

		// 解析赋值右侧的值
		value, err := sp.expressionParser.ParseExpression(sp.tokenStream)
		if err != nil {
			return nil, err
		}

		return &entities.AssignStmt{
			Target: expr,
			Value:  value,
		}, nil
	} else {
		// 这是表达式语句
		return &entities.ExprStmt{
			Expression: expr,
		}, nil
	}
}

// parseIfStatement 解析 if 语句
func (sp *TokenBasedStatementParser) parseIfStatement() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 'if'

	// 解析条件表达式
	condition, err := sp.expressionParser.ParseExpression(sp.tokenStream)
	if err != nil {
		return nil, err
	}

	// 解析 then 分支
	currentToken := sp.tokenStream.Current()
	log.Printf("[DEBUG] parseIfStatement: After ParseExpression, current token=%v (type=%s), position=%d", currentToken, currentToken.Type(), sp.tokenStream.Position())
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after if condition, got %s", currentToken.Type())
	}
	thenBody, err := sp.parseBlock()
	if err != nil {
		return nil, err
	}

	// 检查是否有 else 分支
	var elseBody []entities.ASTNode
	if sp.checkToken(lexicalVO.EnhancedTokenTypeElse) {
		sp.tokenStream.Next() // 消耗 'else'
		if sp.checkToken(lexicalVO.EnhancedTokenTypeIf) {
			// else if
			elseIfStmt, err := sp.parseIfStatement()
			if err != nil {
				return nil, err
			}
			elseBody = []entities.ASTNode{elseIfStmt}
		} else if sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
			// else { ... }
			elseBody, err = sp.parseBlock()
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("expected '{' or 'if' after 'else'")
		}
	}

	return &entities.IfStmt{
		Condition: condition,
		ThenBody:  thenBody,
		ElseBody:  elseBody,
	}, nil
}

// parseWhileStatement 解析 while 语句
func (sp *TokenBasedStatementParser) parseWhileStatement() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 'while'

	// 解析条件表达式
	condition, err := sp.expressionParser.ParseExpression(sp.tokenStream)
	if err != nil {
		return nil, err
	}

	// 解析循环体
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after while condition")
	}
	body, err := sp.parseBlock()
	if err != nil {
		return nil, err
	}

	return &entities.WhileStmt{
		Condition: condition,
		Body:      body,
	}, nil
}

// parseLoopStatement 解析 loop 语句（无限循环）
// 语法：loop { body }
// 等价于 while(true) { body }
func (sp *TokenBasedStatementParser) parseLoopStatement() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 'loop'
	
	// 解析循环体
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after 'loop'")
	}
	body, err := sp.parseBlock()
	if err != nil {
		return nil, fmt.Errorf("failed to parse loop body: %w", err)
	}
	
	// loop 等价于 while(true)
	return &entities.WhileStmt{
		Condition: &entities.BoolLiteral{Value: true},
		Body:      body,
	}, nil
}

// parseForStatement 解析 for 语句
// 支持三种语法：
// 1. for i in start..end { body } - 范围循环（Echo 风格）
// 2. for condition { body } - 简单条件循环（Echo 风格）
// 3. for init; condition; increment { body } - C 风格循环
func (sp *TokenBasedStatementParser) parseForStatement() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 'for'

	// 检查是否是map迭代：for (key, value) in obj { body }
	currentToken := sp.tokenStream.Current()
	if currentToken != nil && currentToken.Type() == lexicalVO.EnhancedTokenTypeLeftParen {
		// 这是map迭代循环
		return sp.parseForInStatement()
	}

	// 检查是否是范围循环或迭代循环：for i in start..end { body } 或 for item in collection { body }
	if currentToken != nil && currentToken.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
		// 可能是范围循环或迭代循环，检查下一个token是否是 'in'
		peekToken := sp.tokenStream.Peek()
		if peekToken != nil && peekToken.Type() == lexicalVO.EnhancedTokenTypeIdentifier && peekToken.Lexeme() == "in" {
			// 检查是否是范围循环（start..end）还是迭代循环（collection）
			// 先解析 'in' 后面的表达式，然后检查是否是 '..' 操作符
			// 由于token stream不支持回退，我们需要先解析表达式，然后判断
			// 这里简化处理：先尝试解析范围循环，如果失败再尝试迭代循环
			// 实际上，我们可以先解析 'in' 后面的表达式，然后检查下一个token是否是 '..'
			return sp.parseForInStatement()
		}
	}

	// 检查是否是 C 风格的 for 循环（包含分号）
	// 先尝试解析一个表达式或语句，然后检查下一个 token
	// 如果下一个 token 是分号，则是 C 风格；否则是简单条件

	// 保存当前位置（用于可能的回退）
	// 由于 token stream 不支持回退，我们采用不同的策略：
	// 先解析一个表达式，如果成功且下一个是分号，则按 C 风格处理
	// 否则按简单条件处理

	// 尝试解析条件表达式
	condition, err := sp.expressionParser.ParseExpression(sp.tokenStream)
	if err != nil {
		// 如果解析失败，可能是 C 风格的 for 循环（init 语句）
		// 或者语法错误
		// 尝试按 C 风格解析
		return sp.parseForStatementCStyle()
	}

	// 检查下一个 token
	if sp.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) {
		// 是 C 风格的 for 循环，但我们已经解析了条件
		// 需要重新解析（但 token stream 不支持回退）
		// 所以这里我们假设条件已经被正确解析，继续 C 风格解析
		sp.tokenStream.Next() // 消耗 ';'
		
		// 解析递增语句（可选）
		var increment entities.ASTNode
		if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
			var err error
			increment, err = sp.ParseStatement(sp.tokenStream)
			if err != nil {
				return nil, err
			}
		}

		// 解析循环体
		if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
			return nil, fmt.Errorf("expected '{' after for loop")
		}
		body, err := sp.parseBlock()
		if err != nil {
			return nil, err
		}

		return &entities.ForStmt{
			Condition: condition,
			Increment: increment,
			Body:      body,
		}, nil
	}

	// 简单条件循环：for condition { body }
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after for condition")
	}
	body, err := sp.parseBlock()
	if err != nil {
		return nil, err
	}

	return &entities.ForStmt{
		Condition: condition,
		Body:      body,
	}, nil
}

// parseForStatementCStyle 解析 C 风格的 for 循环：for init; condition; increment { body }
// 注意：这个方法假设 'for' 已经被消耗，且当前位置是 init 语句的开始
func (sp *TokenBasedStatementParser) parseForStatementCStyle() (entities.ASTNode, error) {
	var init entities.ASTNode
	var condition entities.Expr
	var increment entities.ASTNode

	// 解析初始化语句（可选）
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) {
		var err error
		init, err = sp.ParseStatement(sp.tokenStream)
		if err != nil {
			return nil, err
		}
	}
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) {
		return nil, fmt.Errorf("expected ';' after for init")
	}
	sp.tokenStream.Next() // 消耗 ';'

	// 解析条件表达式（可选）
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) {
		var err error
		condition, err = sp.expressionParser.ParseExpression(sp.tokenStream)
		if err != nil {
			return nil, err
		}
	}
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) {
		return nil, fmt.Errorf("expected ';' after for condition")
	}
	sp.tokenStream.Next() // 消耗 ';'

	// 解析递增语句（可选）
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		var err error
		increment, err = sp.ParseStatement(sp.tokenStream)
		if err != nil {
			return nil, err
		}
	}

	// 解析循环体
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after for loop")
	}
	body, err := sp.parseBlock()
	if err != nil {
		return nil, err
	}

	return &entities.ForStmt{
		Init:      init,
		Condition: condition,
		Increment: increment,
		Body:      body,
		IsRangeLoop: false,
	}, nil
}

// parseForRangeStatement 解析范围循环：for i in start..end { body }
func (sp *TokenBasedStatementParser) parseForRangeStatement() (entities.ASTNode, error) {
	// 当前token应该是循环变量名（如 "i"）
	currentToken := sp.tokenStream.Current()
	if currentToken == nil || currentToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
		return nil, fmt.Errorf("expected identifier for loop variable in range for loop")
	}
	loopVar := currentToken.Lexeme()
	sp.tokenStream.Next() // 消耗循环变量名

	// 检查 'in' 关键字
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeIdentifier) || sp.tokenStream.Current().Lexeme() != "in" {
		return nil, fmt.Errorf("expected 'in' keyword in range for loop")
	}
	sp.tokenStream.Next() // 消耗 'in'

	// 解析范围起始值
	rangeStart, err := sp.expressionParser.ParseExpression(sp.tokenStream)
	if err != nil {
		return nil, fmt.Errorf("failed to parse range start: %w", err)
	}

	// 检查 '..' 操作符（两个点）
	// 注意：'..' 可能是两个连续的 '.' token，或者是一个特殊的 ".." token
	currentToken = sp.tokenStream.Current()
	if currentToken == nil {
		return nil, fmt.Errorf("expected '..' operator in range for loop")
	}
	
	// 检查是否是 ".." 操作符
	// 方法1：检查当前token是否是 '.' 且下一个也是 '.'
	if currentToken.Type() == lexicalVO.EnhancedTokenTypeDot {
		peekToken := sp.tokenStream.Peek()
		if peekToken != nil && peekToken.Type() == lexicalVO.EnhancedTokenTypeDot {
			sp.tokenStream.Next() // 消耗第一个 '.'
			sp.tokenStream.Next() // 消耗第二个 '.'
		} else {
			return nil, fmt.Errorf("expected '..' operator in range for loop")
		}
	} else if currentToken.Lexeme() == ".." {
		// 方法2：如果词法分析器已经识别 ".." 为一个token
		sp.tokenStream.Next() // 消耗 ".."
	} else {
		return nil, fmt.Errorf("expected '..' operator in range for loop, got: %s (type: %s)", currentToken.Lexeme(), currentToken.Type())
	}

	// 解析范围结束值
	rangeEnd, err := sp.expressionParser.ParseExpression(sp.tokenStream)
	if err != nil {
		return nil, fmt.Errorf("failed to parse range end: %w", err)
	}

	// 解析循环体
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after range for loop")
	}
	body, err := sp.parseBlock()
	if err != nil {
		return nil, fmt.Errorf("failed to parse for range body: %w", err)
	}

	return &entities.ForStmt{
		LoopVar:     loopVar,
		RangeStart:  rangeStart,
		RangeEnd:    rangeEnd,
		IsRangeLoop: true,
		IsIterLoop:  false,
		Body:        body,
	}, nil
}

// parseForInStatement 解析 for...in 语句（可能是范围循环或迭代循环）
func (sp *TokenBasedStatementParser) parseForInStatement() (entities.ASTNode, error) {
	// 检查是否是map迭代：for (key, value) in obj { body }
	// 注意：如果当前token是 '('，说明是map迭代
	currentToken := sp.tokenStream.Current()
	if currentToken != nil && currentToken.Type() == lexicalVO.EnhancedTokenTypeLeftParen {
		return sp.parseForMapIteration()
	}
	
	// 当前token应该是循环变量名（如 "i", "item", "ch"）
	if currentToken == nil || currentToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
		return nil, fmt.Errorf("expected identifier for loop variable in for...in loop")
	}
	
	loopVar := currentToken.Lexeme()
	sp.tokenStream.Next() // 消耗循环变量名

	// 检查 'in' 关键字
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeIdentifier) || sp.tokenStream.Current().Lexeme() != "in" {
		return nil, fmt.Errorf("expected 'in' keyword in for...in loop")
	}
	sp.tokenStream.Next() // 消耗 'in'

	// 检查是否是范围循环：for i in start..end
	// 先解析起始值
	startExpr, err := sp.expressionParser.ParseExpression(sp.tokenStream)
	if err != nil {
		return nil, fmt.Errorf("failed to parse start expression: %w", err)
	}
	
	// 检查下一个token是否是 '..' 操作符
	currentToken = sp.tokenStream.Current()
	if currentToken != nil {
		if currentToken.Type() == lexicalVO.EnhancedTokenTypeDot {
			peekToken := sp.tokenStream.Peek()
			if peekToken != nil && peekToken.Type() == lexicalVO.EnhancedTokenTypeDot {
				// 这是范围循环：for i in start..end
				sp.tokenStream.Next() // 消耗第一个 '.'
				sp.tokenStream.Next() // 消耗第二个 '.'
				
				// 解析结束值
				endExpr, err := sp.expressionParser.ParseExpression(sp.tokenStream)
				if err != nil {
					return nil, fmt.Errorf("failed to parse end expression: %w", err)
				}
				
				// 解析循环体
				if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
					return nil, fmt.Errorf("expected '{' after range for loop")
				}
				body, err := sp.parseBlock()
				if err != nil {
					return nil, fmt.Errorf("failed to parse range for body: %w", err)
				}
				
				return &entities.ForStmt{
					LoopVar:     loopVar,
					RangeStart:  startExpr,
					RangeEnd:    endExpr,
					IsRangeLoop: true,
					IsIterLoop:  false,
					Body:        body,
				}, nil
			}
		}
	}
	
	// 如果不是范围循环，则认为是迭代循环：for item in collection { body }
	// 注意：startExpr 实际上就是 collectionExpr
	// 解析循环体
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after for...in loop")
	}
	body, err := sp.parseBlock()
	if err != nil {
		return nil, fmt.Errorf("failed to parse for...in body: %w", err)
	}

	return &entities.ForStmt{
		IterVar:        loopVar,
		IterCollection: startExpr, // startExpr 实际上就是 collectionExpr
		IsIterLoop:     true,
		IsRangeLoop:    false,
		IsMapIter:      false,
		Body:           body,
	}, nil
}

// parseForMapIteration 解析map迭代：for (key, value) in obj { body }
func (sp *TokenBasedStatementParser) parseForMapIteration() (entities.ASTNode, error) {
	// 当前token应该是 '('
	sp.tokenStream.Next() // 消耗 '('
	
	// 解析键变量名
	currentToken := sp.tokenStream.Current()
	if currentToken == nil || currentToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
		return nil, fmt.Errorf("expected identifier for key variable in map iteration")
	}
	keyVar := currentToken.Lexeme()
	sp.tokenStream.Next() // 消耗键变量名
	
	// 检查 ','
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeComma) {
		return nil, fmt.Errorf("expected ',' after key variable in map iteration")
	}
	sp.tokenStream.Next() // 消耗 ','
	
	// 解析值变量名
	currentToken = sp.tokenStream.Current()
	if currentToken == nil || currentToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
		return nil, fmt.Errorf("expected identifier for value variable in map iteration")
	}
	valueVar := currentToken.Lexeme()
	sp.tokenStream.Next() // 消耗值变量名
	
	// 检查 ')'
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
		return nil, fmt.Errorf("expected ')' after value variable in map iteration")
	}
	sp.tokenStream.Next() // 消耗 ')'
	
	// 检查 'in' 关键字
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeIdentifier) || sp.tokenStream.Current().Lexeme() != "in" {
		return nil, fmt.Errorf("expected 'in' keyword in map iteration")
	}
	sp.tokenStream.Next() // 消耗 'in'
	
	// 解析map表达式
	mapExpr, err := sp.expressionParser.ParseExpression(sp.tokenStream)
	if err != nil {
		return nil, fmt.Errorf("failed to parse map expression: %w", err)
	}
	
	// 解析循环体
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after map iteration")
	}
	body, err := sp.parseBlock()
	if err != nil {
		return nil, fmt.Errorf("failed to parse map iteration body: %w", err)
	}

	return &entities.ForStmt{
		IterKeyVar:     keyVar,
		IterValueVar:   valueVar,
		IterCollection: mapExpr,
		IsIterLoop:     true,
		IsMapIter:      true,
		IsRangeLoop:    false,
		Body:           body,
	}, nil
}

// parseFunctionDefinition 解析函数定义
func (sp *TokenBasedStatementParser) parseFunctionDefinition() (entities.ASTNode, error) {
	// 检查是否是异步函数
	isAsync := sp.checkToken(lexicalVO.EnhancedTokenTypeAsync)
	if isAsync {
		sp.tokenStream.Next() // 消耗 'async'
	}

	if !sp.checkToken(lexicalVO.EnhancedTokenTypeFunc) {
		return nil, fmt.Errorf("expected 'func' after 'async'")
	}
	sp.tokenStream.Next() // 消耗 'func'

	// 检查是否是方法定义（有接收者）：func (receiver Type) methodName(...)
	var receiver *MethodReceiver
	if sp.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
		log.Printf("[DEBUG] parseFunctionDefinition: Found method receiver, current token=%v, position=%d", sp.tokenStream.Current(), sp.tokenStream.Position())
		var err error
		receiver, err = sp.parseMethodReceiver()
		if err != nil {
			return nil, err
		}
		log.Printf("[DEBUG] parseFunctionDefinition: Method receiver parsed successfully, receiverType=%s, current token=%v, position=%d", receiver.Type, sp.tokenStream.Current(), sp.tokenStream.Position())
	}

	// 解析函数名（必须在泛型参数之前）
	// 可能是标识符或关键字如 print
	nameToken := sp.tokenStream.Current()
	if nameToken == nil {
		return nil, fmt.Errorf("expected function name")
	}
	
	var name string
	if nameToken.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
		name = nameToken.Lexeme()
	} else if nameToken.Type() == lexicalVO.EnhancedTokenTypePrint {
		// print 是关键字，但可以作为函数名
		name = "print"
	} else {
		return nil, fmt.Errorf("expected function name, got %s", nameToken.Type())
	}
	sp.tokenStream.Next() // 消耗函数名

	// 解析泛型参数（可选）：func name[T](...) 或 func name[T: Trait](...) 或 func name[int](...)（方法实现时的具体类型）
	var genericParams []string
	if sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBracket) {
		sp.tokenStream.Next() // 消耗 '['
		var typeParams []string
		for {
			// 解析泛型参数类型（可能是类型参数名如 T，也可能是具体类型如 int）
			typeParam, err := sp.parseType()
			if err != nil {
				return nil, fmt.Errorf("failed to parse generic type parameter: %w", err)
			}
			typeParams = append(typeParams, typeParam)
			
			// 检查是否有约束：T: Trait
			if sp.checkToken(lexicalVO.EnhancedTokenTypeColon) {
				sp.tokenStream.Next() // 消耗 ':'
				constraintType, err := sp.parseType()
				if err != nil {
					return nil, fmt.Errorf("failed to parse generic parameter constraint: %w", err)
				}
				// 将约束信息附加到类型参数中
				typeParams[len(typeParams)-1] = typeParam + ":" + constraintType
			}
			
			if sp.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
				break
			}
			if !sp.checkToken(lexicalVO.EnhancedTokenTypeComma) {
				return nil, fmt.Errorf("expected ',' or ']' in generic parameters")
			}
			sp.tokenStream.Next() // 消耗 ','
		}
		if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
			return nil, fmt.Errorf("expected ']' after generic parameters")
		}
		sp.tokenStream.Next() // 消耗 ']'
		genericParams = typeParams
	}

	// 解析参数列表
	// 检查当前 token 是否是 '('
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
		currentToken := sp.tokenStream.Current()
		if currentToken != nil {
			return nil, fmt.Errorf("expected '(' for parameter list, got %s", currentToken.Type())
		}
		return nil, fmt.Errorf("expected '(' for parameter list")
	}
	params, err := sp.parseParameterList()
	if err != nil {
		return nil, err
	}

	// 解析返回类型（可选）
	// 支持三种语法：
	// 1. func name() -> ReturnType { (旧语法，保持向后兼容)
	// 2. func name() ReturnType { (新语法，不需要箭头)
	// 3. func name() { (无返回类型，默认为 void)
	var returnType string = "void" // 默认返回类型为 void
	if sp.checkToken(lexicalVO.EnhancedTokenTypeArrow) {
		// 旧语法：使用 -> 箭头
		sp.tokenStream.Next() // 消耗 '->'
		// 使用 parseType() 解析返回类型（支持数组、泛型、函数类型等）
		var err error
		returnType, err = sp.parseType()
		if err != nil {
			return nil, fmt.Errorf("failed to parse return type: %w", err)
		}
	} else {
		// 新语法：尝试直接解析类型（如果后面不是 '{'）
		// 检查下一个 token 是否是 '{'，如果不是，可能是返回类型
		if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
			// 尝试解析类型（可能是返回类型）
			// 保存当前位置以便回退
			savedPosition := sp.tokenStream.Position()
			parsedType, err := sp.parseType()
			if err == nil {
				// 解析成功，检查后面是否是 '{'
				if sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
					// 确实是返回类型
					returnType = parsedType
				} else {
					// 不是返回类型，回退（保持默认 void）
					sp.tokenStream.SetPosition(savedPosition)
				}
			}
			// 如果解析失败，继续（保持默认 void）
		}
		// 如果直接是 '{'，returnType 保持为默认的 "void"
	}

	// 解析函数体
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after function signature")
	}
	body, err := sp.parseBlock()
	if err != nil {
		return nil, err
	}

	// 如果有接收者，返回方法定义
	if receiver != nil {
		// 从接收者类型中提取类型参数（如 *HashMap[K, V] -> [K, V]）
		receiverParams := sp.extractReceiverTypeParams(receiver.Type)
		log.Printf("[DEBUG] parseFunctionDefinition: Method receiverType=%s, extracted ReceiverParams count=%d", receiver.Type, len(receiverParams))
		if len(receiverParams) > 0 {
			for i, rp := range receiverParams {
				log.Printf("[DEBUG] parseFunctionDefinition: ReceiverParams[%d]=%s", i, rp.Name)
			}
		}
		
		return &entities.MethodDef{
			Receiver:       receiver.Type,
			ReceiverVar:    receiver.VarName,
			ReceiverParams: receiverParams, // 填充接收者类型参数
			Name:           name,
			Params:         params,
			ReturnType:     returnType,
			Body:           body,
		}, nil
	}

	if isAsync {
		return &entities.AsyncFuncDef{
			Name:       name,
			Params:     params,
			ReturnType: returnType,
			Body:       body,
		}, nil
	}

	// 将 genericParams ([]string) 转换为 []GenericParam
	var typeParams []entities.GenericParam
	for _, gp := range genericParams {
		// 解析类型参数名和约束（格式：T 或 T: Trait）
		parts := strings.Split(gp, ":")
		paramName := strings.TrimSpace(parts[0])
		var constraints []string
		if len(parts) > 1 {
			// 有约束条件
			constraintStr := strings.TrimSpace(parts[1])
			// 支持多个约束，用 + 分隔（如 T: Trait1 + Trait2）
			constraintParts := strings.Split(constraintStr, "+")
			for _, cp := range constraintParts {
				constraints = append(constraints, strings.TrimSpace(cp))
			}
		}
		typeParams = append(typeParams, entities.GenericParam{
			Name:        paramName,
			Constraints: constraints,
		})
	}

	return &entities.FuncDef{
		Name:       name,
		TypeParams: typeParams, // ✅ 修复：将泛型参数存储到函数定义中
		Params:     params,
		ReturnType: returnType,
		Body:       body,
	}, nil
}

// MethodReceiver 方法接收者
type MethodReceiver struct {
	VarName string // 接收者变量名，如 "p"
	Type    string // 接收者类型，如 "Point"
}

// parseMethodReceiver 解析方法接收者：func (receiver Type) ...
func (sp *TokenBasedStatementParser) parseMethodReceiver() (*MethodReceiver, error) {
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
		return nil, fmt.Errorf("expected '(' for method receiver")
	}
	sp.tokenStream.Next() // 消耗 '('

	// 解析接收者变量名
	receiverVarToken := sp.tokenStream.Current()
	if receiverVarToken == nil || receiverVarToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
		return nil, fmt.Errorf("expected receiver variable name")
	}
	receiverVar := receiverVarToken.Lexeme()
	sp.tokenStream.Next() // 消耗接收者变量名

	// 解析接收者类型（使用 parseType 支持泛型类型，如 Container[T]）
	currentToken := sp.tokenStream.Current()
	if currentToken != nil {
		log.Printf("[DEBUG] parseMethodReceiver: Before parseType, current token=%v (type=%s), position=%d", currentToken, currentToken.Type(), sp.tokenStream.Position())
	}
	receiverType, err := sp.parseType()
	if err != nil {
		return nil, fmt.Errorf("failed to parse receiver type: %w", err)
	}
	log.Printf("[DEBUG] parseMethodReceiver: After parseType, receiverType=%s, current token=%v", receiverType, sp.tokenStream.Current())

	// 解析右括号
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
		return nil, fmt.Errorf("expected ')' after receiver type")
	}
	sp.tokenStream.Next() // 消耗 ')'

	return &MethodReceiver{
		VarName: receiverVar,
		Type:    receiverType,
	}, nil
}

// parseGenericParameters 解析泛型参数：func name[T, U](...) 或 func name[T: Trait, U: Comparable](...)
func (sp *TokenBasedStatementParser) parseGenericParameters() ([]string, error) {
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBracket) {
		return nil, nil // 没有泛型参数
	}
	sp.tokenStream.Next() // 消耗 '['

	var params []string
	for {
		// 解析泛型参数名
		paramToken := sp.tokenStream.Current()
		if paramToken == nil || paramToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
			return nil, fmt.Errorf("expected generic parameter name")
		}
		paramName := paramToken.Lexeme()
		sp.tokenStream.Next() // 消耗参数名

		// 检查是否有约束：T: Trait
		if sp.checkToken(lexicalVO.EnhancedTokenTypeColon) {
			sp.tokenStream.Next() // 消耗 ':'
			
			// 解析约束 trait 名称（可能是泛型类型，如 Container[int]）
			constraintType, err := sp.parseType()
			if err != nil {
				return nil, fmt.Errorf("failed to parse constraint type: %w", err)
			}
			
			// 暂时只存储参数名，约束信息可以后续扩展
			// 格式：paramName:constraintName（简化处理）
			params = append(params, paramName+":"+constraintType)
		} else {
			// 无约束的泛型参数
			params = append(params, paramName)
		}

		// 检查是否有更多参数
		if sp.checkToken(lexicalVO.EnhancedTokenTypeComma) {
			sp.tokenStream.Next() // 消耗 ','
			continue
		}

		// 检查是否结束
		if sp.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
			break
		}

		return nil, fmt.Errorf("expected ',' or ']' in generic parameters")
	}

	if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
		return nil, fmt.Errorf("expected ']' after generic parameters")
	}
	sp.tokenStream.Next() // 消耗 ']'

	return params, nil
}

// parseParameterList 解析参数列表
func (sp *TokenBasedStatementParser) parseParameterList() ([]entities.Param, error) {
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
		return nil, fmt.Errorf("expected '(' for parameter list")
	}
	sp.tokenStream.Next() // 消耗 '('

	var params []entities.Param
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
		for {
			// 解析参数名
			nameToken := sp.tokenStream.Current()
			if nameToken == nil || nameToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
				return nil, fmt.Errorf("expected parameter name")
			}
			name := nameToken.Lexeme()
			sp.tokenStream.Next() // 消耗参数名

			// 解析参数类型
			if !sp.checkToken(lexicalVO.EnhancedTokenTypeColon) {
				return nil, fmt.Errorf("expected ':' after parameter name")
			}
			sp.tokenStream.Next() // 消耗 ':'
			// 使用 parseType() 解析参数类型（支持 chan、标识符、数组、泛型）
			paramType, err := sp.parseType()
			if err != nil {
				return nil, fmt.Errorf("failed to parse parameter type: %w", err)
			}

			params = append(params, entities.Param{
				Name: name,
				Type: paramType,
			})

			if sp.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
				break
			}
			if !sp.checkToken(lexicalVO.EnhancedTokenTypeComma) {
				return nil, fmt.Errorf("expected ',' or ')' in parameter list")
			}
			sp.tokenStream.Next() // 消耗 ','
		}
	}

	sp.tokenStream.Next() // 消耗 ')'
	return params, nil
}

// parseBlock 解析代码块
func (sp *TokenBasedStatementParser) parseBlock() ([]entities.ASTNode, error) {
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{'")
	}
	sp.tokenStream.Next() // 消耗 '{'

	var statements []entities.ASTNode
	for !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) && !sp.tokenStream.IsAtEnd() {
		stmt, err := sp.ParseStatement(sp.tokenStream)
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			statements = append(statements, stmt)
		}
	}

	if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
		return nil, fmt.Errorf("expected '}'")
	}
	sp.tokenStream.Next() // 消耗 '}'

	return statements, nil
}

// parseBlockAsStatement 将代码块解析为语句（用于处理独立的代码块）
func (sp *TokenBasedStatementParser) parseBlockAsStatement() (entities.ASTNode, error) {
	statements, err := sp.parseBlock()
	if err != nil {
		return nil, err
	}
	// 返回 BlockStmt 或单条语句
	if len(statements) == 0 {
		return nil, nil
	}
	if len(statements) == 1 {
		return statements[0], nil
	}
	return &entities.BlockStmt{Statements: statements}, nil
}

// parseStructDefinition 解析结构体定义
// 格式：struct StructName { field1: type1, field2: type2, }
func (sp *TokenBasedStatementParser) parseStructDefinition() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 'struct'

	// 解析结构体名称
	nameToken := sp.tokenStream.Current()
	if nameToken == nil || nameToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
		return nil, fmt.Errorf("expected struct name after 'struct'")
	}
	structName := nameToken.Lexeme()
	sp.tokenStream.Next() // 消耗结构体名

	// 解析泛型参数（可选）：struct Container[T]
	var genericParams []string
	if sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBracket) {
		var err error
		genericParams, err = sp.parseGenericParameters()
		if err != nil {
			return nil, err
		}
		// genericParams 在下方转换为 GenericParam 并存入 StructDef.TypeParams
	}

	// 解析左大括号
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after struct name")
	}
	sp.tokenStream.Next() // 消耗 '{'

	// 解析字段列表
	var fields []entities.StructField
	for !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) && !sp.tokenStream.IsAtEnd() {
		// 跳过逗号（如果存在）
		if sp.checkToken(lexicalVO.EnhancedTokenTypeComma) {
			sp.tokenStream.Next()
			continue
		}

		// 跳过分号（如果存在，作为字段分隔符）
		if sp.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) {
			sp.tokenStream.Next()
			// 如果下一个 token 是右大括号，结束字段解析
			if sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
				break
			}
			continue
		}

		// 解析字段名
		fieldNameToken := sp.tokenStream.Current()
		if fieldNameToken == nil || fieldNameToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
			// 可能是右大括号，结束字段解析
			if sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
				break
			}
			return nil, fmt.Errorf("expected field name in struct definition")
		}
		fieldName := fieldNameToken.Lexeme()
		sp.tokenStream.Next() // 消耗字段名

		// 解析冒号
		if !sp.checkToken(lexicalVO.EnhancedTokenTypeColon) {
			return nil, fmt.Errorf("expected ':' after field name '%s'", fieldName)
		}
		sp.tokenStream.Next() // 消耗 ':'

		// 解析字段类型（使用 parseType 方法支持数组、泛型等）
		fieldType, err := sp.parseType()
		if err != nil {
			return nil, fmt.Errorf("failed to parse field type for field '%s': %w", fieldName, err)
		}

		// 添加字段
		fields = append(fields, entities.StructField{
			Name: fieldName,
			Type: fieldType,
		})

		// 检查是否有逗号或分号（可选，作为字段分隔符）
		if sp.checkToken(lexicalVO.EnhancedTokenTypeComma) {
			sp.tokenStream.Next() // 消耗 ','
		} else if sp.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) {
			sp.tokenStream.Next() // 消耗 ';'
		}
	}

	// 解析右大括号
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
		return nil, fmt.Errorf("expected '}' to close struct definition")
	}
	sp.tokenStream.Next() // 消耗 '}'

	// 将 genericParams 转换为 GenericParam 列表
	typeParams := make([]entities.GenericParam, len(genericParams))
	for i, paramName := range genericParams {
		typeParams[i] = entities.GenericParam{
			Name: paramName,
		}
	}

	return &entities.StructDef{
		Name:       structName,
		TypeParams: typeParams,
		Fields:     fields,
	}, nil
}

// parseEnumDefinition 解析枚举定义
func (sp *TokenBasedStatementParser) parseEnumDefinition() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 'enum'

	// 解析枚举名称
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeIdentifier) {
		return nil, fmt.Errorf("expected enum name after 'enum'")
	}
	enumNameToken := sp.tokenStream.Current()
	enumName := enumNameToken.Lexeme()
	sp.tokenStream.Next() // 消耗枚举名称

	// 解析左大括号
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after enum name")
	}
	sp.tokenStream.Next() // 消耗 '{'

	// 解析枚举变体
	variants := make([]entities.EnumVariant, 0)
	for !sp.tokenStream.IsAtEnd() && !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
		// 解析变体名称
		if !sp.checkToken(lexicalVO.EnhancedTokenTypeIdentifier) {
			return nil, fmt.Errorf("expected enum variant name")
		}
		variantNameToken := sp.tokenStream.Current()
		variantName := variantNameToken.Lexeme()
		sp.tokenStream.Next() // 消耗变体名称

		variants = append(variants, entities.EnumVariant{
			Name: variantName,
		})

		// ✅ 修复：支持换行符分隔的枚举变体（不需要逗号）
		// 检查是否有逗号、右大括号，或者下一个 token 是标识符（下一个枚举变体）
		if sp.checkToken(lexicalVO.EnhancedTokenTypeComma) {
			sp.tokenStream.Next() // 消耗 ','
		} else if sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
			// 遇到右大括号，退出循环
			break
		} else if sp.checkToken(lexicalVO.EnhancedTokenTypeIdentifier) {
			// 下一个 token 是标识符，可能是下一个枚举变体，继续循环
			continue
		} else {
			// 其他情况（如换行符、分号等），也继续循环
			// 换行符和分号会被自动跳过，下一个 token 应该是标识符或右大括号
			// 如果遇到右大括号，循环条件会捕获
			continue
		}
	}

	// 解析右大括号
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
		return nil, fmt.Errorf("expected '}' after enum variants")
	}
	sp.tokenStream.Next() // 消耗 '}'

	return &entities.EnumDef{
		Name:     enumName,
		Variants: variants,
	}, nil
}

// parseTraitDefinition 解析 trait 定义
func (sp *TokenBasedStatementParser) parseTraitDefinition() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 'trait'

	// 解析 trait 名称
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeIdentifier) {
		return nil, fmt.Errorf("expected trait name after 'trait'")
	}
	traitNameToken := sp.tokenStream.Current()
	traitName := traitNameToken.Lexeme()
	sp.tokenStream.Next() // 消耗 trait 名称

	// 解析泛型参数（可选）：trait Container[T]
	var genericParams []string
	if sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBracket) {
		var err error
		genericParams, err = sp.parseGenericParameters()
		if err != nil {
			return nil, err
		}
		// genericParams 在下方转换为 GenericParam 并存入 TraitDef.TypeParams
	}

	// 解析 Trait 继承（可选）：trait Printable: Display 或 trait AdvancedPrintable: Printable, Formattable
	var superTraits []string
	if sp.checkToken(lexicalVO.EnhancedTokenTypeColon) {
		sp.tokenStream.Next() // 消耗 ':'
		
		// 解析父 trait 列表（逗号分隔）
		for {
			if !sp.checkToken(lexicalVO.EnhancedTokenTypeIdentifier) {
				return nil, fmt.Errorf("expected super trait name after ':'")
			}
			superTraitToken := sp.tokenStream.Current()
			superTraitName := superTraitToken.Lexeme()
			sp.tokenStream.Next() // 消耗父 trait 名称
			
			superTraits = append(superTraits, superTraitName)
			
			// 检查是否有逗号（多个父 trait）
			if sp.checkToken(lexicalVO.EnhancedTokenTypeComma) {
				sp.tokenStream.Next() // 消耗 ','
				continue
			}
			break
		}
	}

	// 解析左大括号
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after trait name")
	}
	sp.tokenStream.Next() // 消耗 '{'

	// 解析 trait 方法和关联类型
	methods := make([]entities.TraitMethod, 0)
	associatedTypes := make([]entities.AssociatedType, 0)
	for !sp.tokenStream.IsAtEnd() && !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
		// 跳过分号（如果存在）
		if sp.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) {
			sp.tokenStream.Next()
			continue
		}

		// 检查是否是关联类型声明：type Item;
		if sp.checkToken(lexicalVO.EnhancedTokenTypeIdentifier) {
			typeToken := sp.tokenStream.Current()
			if typeToken.Lexeme() == "type" {
				sp.tokenStream.Next() // 消耗 'type'
				
				// 解析关联类型名
				if !sp.checkToken(lexicalVO.EnhancedTokenTypeIdentifier) {
					return nil, fmt.Errorf("expected associated type name after 'type'")
				}
				associatedTypeNameToken := sp.tokenStream.Current()
				associatedTypeName := associatedTypeNameToken.Lexeme()
				sp.tokenStream.Next() // 消耗关联类型名
				
				associatedTypes = append(associatedTypes, entities.AssociatedType{
					Name: associatedTypeName,
				})
				
				// 跳过分号（如果存在）
				if sp.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) {
					sp.tokenStream.Next()
				}
				continue
			}
		}

		// 解析方法定义
		// trait 方法可以是：func methodName() -> ReturnType 或 func methodName() -> ReturnType { body }
		if !sp.checkToken(lexicalVO.EnhancedTokenTypeFunc) {
			// 如果不是 func，可能是 trait 定义的结束或其他内容
			// 检查是否是右大括号
			if sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
				break
			}
			currentToken := sp.tokenStream.Current()
			if currentToken != nil {
				return nil, fmt.Errorf("expected 'func' or 'type' in trait definition, got %s", currentToken.Type())
			}
			return nil, fmt.Errorf("expected 'func' or 'type' in trait definition")
		}
		sp.tokenStream.Next() // 消耗 'func'

		// 解析方法名（可能是标识符或关键字如 print）
		methodNameToken := sp.tokenStream.Current()
		if methodNameToken == nil {
			return nil, fmt.Errorf("expected method name after 'func'")
		}
		
		var methodName string
		if methodNameToken.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
			methodName = methodNameToken.Lexeme()
		} else if methodNameToken.Type() == lexicalVO.EnhancedTokenTypePrint {
			// print 是关键字，但可以作为方法名
			methodName = "print"
		} else {
			return nil, fmt.Errorf("expected method name after 'func', got %s", methodNameToken.Type())
		}
		sp.tokenStream.Next() // 消耗方法名

		// 检查是否有方法级泛型参数：func transform[U](...) 或 func transform[int](...)
		// 对于 trait 方法定义，泛型参数可能是类型参数（如 [U]）或带约束的类型参数（如 [U: Comparable]）
		// 对于方法实现，泛型参数可能是具体类型（如 [int]）
		var methodGenericParams []string
		if sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBracket) {
			// 直接解析为类型列表（支持类型参数、约束和具体类型）
			sp.tokenStream.Next() // 消耗 '['
			var typeParams []string
			for {
				// 解析泛型参数类型（可能是类型参数名如 U，也可能是具体类型如 int）
				typeParam, err := sp.parseType()
				if err != nil {
					return nil, fmt.Errorf("failed to parse method generic type parameter: %w", err)
				}
				typeParams = append(typeParams, typeParam)
				
				// 检查是否有约束：U: Comparable
				if sp.checkToken(lexicalVO.EnhancedTokenTypeColon) {
					sp.tokenStream.Next() // 消耗 ':'
					constraintType, err := sp.parseType()
					if err != nil {
						return nil, fmt.Errorf("failed to parse generic parameter constraint: %w", err)
					}
					// 将约束信息附加到类型参数中
					typeParams[len(typeParams)-1] = typeParam + ":" + constraintType
				}
				
				if sp.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
					break
				}
				if !sp.checkToken(lexicalVO.EnhancedTokenTypeComma) {
					return nil, fmt.Errorf("expected ',' or ']' in method generic type parameters")
				}
				sp.tokenStream.Next() // 消耗 ','
			}
			if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
				return nil, fmt.Errorf("expected ']' after method generic type parameters")
			}
			sp.tokenStream.Next() // 消耗 ']'
			methodGenericParams = typeParams
			// methodGenericParams 在下方转换为 GenericParam 并存入 TraitMethod.TypeParams
		}

		// 解析参数列表
		params, err := sp.parseParameterList()
		if err != nil {
			return nil, err
		}

		// 解析返回类型（可选）
		// 支持三种语法：
		// 1. func name() -> ReturnType (旧语法，保持向后兼容)
		// 2. func name() ReturnType (新语法，不需要箭头)
		// 3. func name() { (无返回类型，默认为 void)
		var returnType string = "void" // 默认返回类型为 void
		if sp.checkToken(lexicalVO.EnhancedTokenTypeArrow) {
			// 旧语法：使用 -> 箭头
			sp.tokenStream.Next() // 消耗 '->'
			// 使用 parseType() 解析返回类型（支持泛型类型）
			var err error
			returnType, err = sp.parseType()
			if err != nil {
				return nil, fmt.Errorf("failed to parse return type: %w", err)
			}
		} else {
			// 新语法：尝试直接解析类型（如果后面不是 '{'、';' 或 '}'）
			// 检查下一个 token 是否是 '{'、';' 或 '}'，如果不是，可能是返回类型
			if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) &&
			   !sp.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) &&
			   !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
				// 尝试解析类型（可能是返回类型）
				// 保存当前位置以便回退
				savedPosition := sp.tokenStream.Position()
				parsedType, err := sp.parseType()
				if err == nil {
					// 解析成功，检查后面是否是 '{'、';' 或 '}'
					if sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) ||
					   sp.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) ||
					   sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
						// 确实是返回类型
						returnType = parsedType
					} else {
						// 不是返回类型，回退（保持默认 void）
						sp.tokenStream.SetPosition(savedPosition)
					}
				}
				// 如果解析失败，继续（保持默认 void）
			}
			// 如果直接是 '{'、';' 或 '}'，returnType 保持为默认的 "void"
		}

		// 检查是否有方法体（默认实现）
		var body []entities.ASTNode
		if sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
			// 有方法体，解析它
			var err error
			body, err = sp.parseBlock()
			if err != nil {
				return nil, err
			}
		}

		// 将 methodGenericParams 转换为 GenericParam 列表
		methodTypeParams := make([]entities.GenericParam, len(methodGenericParams))
		for i, paramName := range methodGenericParams {
			methodTypeParams[i] = entities.GenericParam{
				Name: paramName,
			}
		}

		methods = append(methods, entities.TraitMethod{
			Name:       methodName,
			TypeParams: methodTypeParams,
			Params:     params,
			ReturnType: returnType,
			Body:       body,
		})
	}

	// 解析右大括号
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
		return nil, fmt.Errorf("expected '}' after trait methods")
	}
	sp.tokenStream.Next() // 消耗 '}'

	// 将 genericParams 转换为 GenericParam 列表并存入 TraitDef
	typeParams := make([]entities.GenericParam, len(genericParams))
	for i, paramName := range genericParams {
		typeParams[i] = entities.GenericParam{Name: paramName}
	}

	return &entities.TraitDef{
		Name:            traitName,
		TypeParams:      typeParams,
		SuperTraits:     superTraits,
		Methods:         methods,
		AssociatedTypes: associatedTypes,
	}, nil
}

// parseTypeAlias 解析类型别名：type Item = int;
func (sp *TokenBasedStatementParser) parseTypeAlias() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 'type'

	// 解析类型别名名称
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeIdentifier) {
		return nil, fmt.Errorf("expected type alias name after 'type'")
	}
	aliasNameToken := sp.tokenStream.Current()
	aliasName := aliasNameToken.Lexeme()
	sp.tokenStream.Next() // 消耗别名名称

	// 解析 '='
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeAssign) {
		return nil, fmt.Errorf("expected '=' after type alias name")
	}
	sp.tokenStream.Next() // 消耗 '='

	// 解析目标类型
	_, err := sp.parseType()
	if err != nil {
		return nil, fmt.Errorf("failed to parse target type: %w", err)
	}

	// 跳过分号（如果存在）
	if sp.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) {
		sp.tokenStream.Next()
	}

	// 暂时使用 AssociatedTypeDef，后续可以扩展以支持目标类型
	// TODO: 创建专门的类型别名节点或扩展 AssociatedTypeImpl 以支持 ASTNode
	return &entities.AssociatedTypeDef{
		Name: aliasName,
	}, nil
}

// parseImplAnnotation 解析 @impl 注解
// @impl 注解通常单独成行，后面跟着结构体或枚举定义
// 这里我们暂时忽略注解（仅用于信息），让后续的解析器处理
func (sp *TokenBasedStatementParser) parseImplAnnotation() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 '@impl' 或 'impl'

	// 解析 trait 名称（可能包含泛型参数，如 Container[int]）
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeIdentifier) {
		return nil, fmt.Errorf("expected trait name after '@impl'")
	}
	sp.tokenStream.Next() // 消耗 trait 名称

	// 解析泛型参数（可选）：@impl Container[int]
	if sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBracket) {
		var err error
		_, err = sp.parseGenericParameters()
		if err != nil {
			return nil, err
		}
		// TODO: 将泛型参数存储到注解中
	}

	// @impl 注解通常单独成行，后面可能跟着结构体或枚举定义
	// 暂时返回 nil，表示忽略这个注解（后续可以改进以支持注解关联）
	// TODO: 实现注解与结构体/枚举的关联
	return nil, nil
}

// parseMatchStatement 解析 match 语句
func (sp *TokenBasedStatementParser) parseMatchStatement() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 'match'

	// 解析匹配的表达式（仅到 '{' 前；避免把 "expr {" 解析成 struct literal）
	sp.expressionParser.SetInMatchValue(true)
	expr, err := sp.expressionParser.ParseExpression(sp.tokenStream)
	sp.expressionParser.SetInMatchValue(false) // 立即清除，case body 内需解析 struct literal（如 Ok(Type { ... })）
	if err != nil {
		return nil, fmt.Errorf("failed to parse match expression: %w", err)
	}

	// 解析 match 块
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after match expression")
	}
	sp.tokenStream.Next() // 消耗 '{'

	// 解析 match cases
	var cases []entities.MatchCase
	for !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) && !sp.tokenStream.IsAtEnd() {
		// 跳过分号、逗号和换行（match case 之间可能用逗号分隔）
		if sp.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) || sp.checkToken(lexicalVO.EnhancedTokenTypeComma) {
			sp.tokenStream.Next()
			// 如果下一个 token 是右大括号，说明分隔符是最后一个 case 后的，可以忽略
			if sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
				break
			}
			continue
		}

		// ✅ 修复：在解析 pattern 之前，检查是否是 fat_arrow
		// 如果是，说明上一个 case 的 body 解析有问题，或者 token stream 位置不对
		if sp.checkToken(lexicalVO.EnhancedTokenTypeFatArrow) {
			// 不应该在解析 pattern 之前就遇到 fat_arrow
			// 这可能是上一个 case 的 body 解析有问题，消耗了太多 token
			return nil, fmt.Errorf("unexpected fat_arrow before pattern (previous case body may have consumed too many tokens, position: %d)", sp.tokenStream.Position())
		}

		// 解析 pattern（使用专门的 pattern 解析方法）
		// 注意：需要直接调用 parsePattern 方法，而不是通过 ParseExpression
		// 因为 pattern 和 expression 的解析逻辑不同
		// ✅ 修复：确保 expressionParser 的 tokenStream 被正确设置
		sp.expressionParser.tokenStream = sp.tokenStream
		pattern, err := sp.expressionParser.parsePattern()
		if err != nil {
			return nil, fmt.Errorf("failed to parse match pattern: %w", err)
		}

		// 解析 =>
		if !sp.checkToken(lexicalVO.EnhancedTokenTypeFatArrow) {
			currentToken := sp.tokenStream.Current()
			if currentToken != nil {
				return nil, fmt.Errorf("expected '=>' after match pattern, got %s", currentToken.Type())
			}
			return nil, fmt.Errorf("expected '=>' after match pattern")
		}
		sp.tokenStream.Next() // 消耗 '=>'

		// 解析 statement 或表达式
		// match case 的 body 可以是语句或表达式
		var bodyStmts []entities.ASTNode
		// ✅ 修复 T-DEV-018：在解析整个 case body 前设置上下文，使 return 内的 ParseExpression 遇 Ident( 即停
		sp.expressionParser.SetInMatchCaseBody(true)
		defer func() { sp.expressionParser.SetInMatchCaseBody(false) }()

		// 检查是否是语句（如 print, return 等）
		currentToken := sp.tokenStream.Current()
		if currentToken != nil {
			if currentToken.Type() == lexicalVO.EnhancedTokenTypePrint ||
				currentToken.Type() == lexicalVO.EnhancedTokenTypeReturn {
				// 是语句，解析语句
				stmt, err := sp.ParseStatement(sp.tokenStream)
				if err != nil {
					return nil, fmt.Errorf("failed to parse match case statement: %w", err)
				}
				if stmt != nil {
					bodyStmts = append(bodyStmts, stmt)
				}
			} else if currentToken.Type() == lexicalVO.EnhancedTokenTypeLeftBrace {
				// 是语句块（用 { } 包围），解析语句块
				blockStmts, err := sp.parseBlock()
				if err != nil {
					return nil, fmt.Errorf("failed to parse match case body block: %w", err)
				}
				bodyStmts = blockStmts
			} else {
				// 是表达式，解析表达式并包装为表达式语句
				expr, err := sp.expressionParser.ParseExpression(sp.tokenStream)
				if err != nil {
					return nil, fmt.Errorf("failed to parse match case expression: %w", err)
				}
				bodyStmts = append(bodyStmts, &entities.ExprStmt{
					Expression: expr,
				})
			}
		} else {
			return nil, fmt.Errorf("unexpected end of token stream in match case")
		}

		// ✅ 修复：在解析 body 后，检查 token stream 位置是否正确
		// 下一个 token 应该是右大括号、逗号、分号或下一个 case 的 pattern
		// 如果遇到 fat_arrow，说明 body 解析消耗了太多 token
		nextToken := sp.tokenStream.Current()
		if nextToken != nil {
			// 如果下一个 token 是 fat_arrow，说明 body 解析消耗了下一个 case 的 token
			if nextToken.Type() == lexicalVO.EnhancedTokenTypeFatArrow {
				return nil, fmt.Errorf("match case body consumed too many tokens, unexpected fat_arrow (position: %d). Previous case body may have consumed the next case's pattern", sp.tokenStream.Position())
			}
		}

		// 构建 MatchCase
		cases = append(cases, entities.MatchCase{
			Pattern: pattern,
			Body:    bodyStmts,
		})

		// 检查下一个 token 是否是 match 块的结束符
		if sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
			break
		}

		// 检查是否有逗号或分号（多个 case 之间的分隔符）
		if sp.checkToken(lexicalVO.EnhancedTokenTypeComma) || sp.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) {
			sp.tokenStream.Next() // 消耗逗号或分号
			// 如果下一个 token 是右大括号，说明这是最后一个 case 后的分隔符
			if sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
				break
			}
		}
	}

	if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
		return nil, fmt.Errorf("expected '}' after match cases")
	}
	sp.tokenStream.Next() // 消耗 '}'

	return &entities.MatchStmt{
		Value: expr,
		Cases: cases,
	}, nil
}

// parseSelectStatement 解析 select 语句
// 格式：select { case <pattern> := <- <channel>: <statement> ... }
func (sp *TokenBasedStatementParser) parseSelectStatement() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 'select'

	// 解析 select 块
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after 'select'")
	}
	sp.tokenStream.Next() // 消耗 '{'

	// 解析 select cases
	var cases []entities.SelectCase
	for !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) && !sp.tokenStream.IsAtEnd() {
		// 检查是否是 case 关键字
		if !sp.checkToken(lexicalVO.EnhancedTokenTypeCase) {
			// 可能是 default 分支或其他，暂时只支持 case
			currentToken := sp.tokenStream.Current()
			if currentToken != nil && currentToken.Type() == lexicalVO.EnhancedTokenTypeRightBrace {
				break // 遇到 '}'，结束
			}
			return nil, fmt.Errorf("expected 'case' in select statement (got %s)", currentToken.Type())
		}
		sp.tokenStream.Next() // 消耗 'case'

		// 解析 pattern（变量名）
		patternToken := sp.tokenStream.Current()
		if patternToken == nil || patternToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
			return nil, fmt.Errorf("expected identifier after 'case'")
		}
		_ = patternToken.Lexeme() // patternName，暂时未使用
		sp.tokenStream.Next()     // 消耗标识符

		// 解析 :=
		if !sp.checkToken(lexicalVO.EnhancedTokenTypeDeclare) {
			return nil, fmt.Errorf("expected ':=' after case pattern")
		}
		sp.tokenStream.Next() // 消耗 ':='

		// 解析通道接收表达式（<- channel）
		// 这里需要解析 <- channel 表达式
		// 由于 <- 是前缀运算符，我们需要特殊处理
		if !sp.checkToken(lexicalVO.EnhancedTokenTypeChannelReceive) {
			return nil, fmt.Errorf("expected '<-' after ':=' in select case")
		}
		sp.tokenStream.Next() // 消耗 '<-'

		// 解析通道表达式
		channelExpr, err := sp.expressionParser.ParseExpression(sp.tokenStream)
		if err != nil {
			return nil, fmt.Errorf("failed to parse channel expression in select case: %w", err)
		}

		// 解析 ':'
		if !sp.checkToken(lexicalVO.EnhancedTokenTypeColon) {
			return nil, fmt.Errorf("expected ':' after channel expression in select case")
		}
		sp.tokenStream.Next() // 消耗 ':'

		// 解析 statement（可能是一个或多个语句）
		// 注意：select case 的 body 可能包含多个语句，直到遇到下一个 case 或 }
		var bodyStmts []entities.ASTNode
		for {
			// 检查是否到达 select 块的结束或下一个 case
			currentToken := sp.tokenStream.Current()
			if currentToken == nil {
				break
			}
			if currentToken.Type() == lexicalVO.EnhancedTokenTypeRightBrace ||
				currentToken.Type() == lexicalVO.EnhancedTokenTypeCase {
				break
			}

			// 解析一个语句
			stmt, err := sp.ParseStatement(sp.tokenStream)
			if err != nil {
				return nil, fmt.Errorf("failed to parse select case statement: %w", err)
			}
			if stmt != nil {
				bodyStmts = append(bodyStmts, stmt)
			}
		}

		// 检查是否到达 select 块的结束
		if sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
			break
		}

		// 构建 SelectCase
		selectCase := entities.SelectCase{
			Chan:   channelExpr,
			IsSend: false, // 接收操作
			Body:   bodyStmts,
		}
		cases = append(cases, selectCase)
	}

	if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
		return nil, fmt.Errorf("expected '}' after select cases")
	}
	sp.tokenStream.Next() // 消耗 '}'

	return &entities.SelectStmt{
		Cases: cases,
	}, nil
}

// parseReturnStatement 解析 return 语句
func (sp *TokenBasedStatementParser) parseReturnStatement() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 'return'

	// 解析返回值（可选）
	var value entities.Expr
	currentToken := sp.tokenStream.Current()
	if currentToken == nil || sp.tokenStream.IsAtEnd() {
		// 没有更多 token，return 语句没有返回值
		value = nil
	} else if sp.checkToken(lexicalVO.EnhancedTokenTypeSemicolon) ||
		sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
		// 遇到分号或右大括号，return 语句没有返回值
		value = nil
	} else if currentToken.Type() == lexicalVO.EnhancedTokenTypeReturn ||
		currentToken.Type() == lexicalVO.EnhancedTokenTypeIf ||
		currentToken.Type() == lexicalVO.EnhancedTokenTypeWhile ||
		currentToken.Type() == lexicalVO.EnhancedTokenTypeFor ||
		currentToken.Type() == lexicalVO.EnhancedTokenTypeLet ||
		currentToken.Type() == lexicalVO.EnhancedTokenTypeFunc ||
		currentToken.Type() == lexicalVO.EnhancedTokenTypePrint {
		// 遇到语句关键字，说明没有返回值
		value = nil
	} else {
		// 有返回值，解析表达式
		var err error
		value, err = sp.expressionParser.ParseExpression(sp.tokenStream)
		if err != nil {
			return nil, fmt.Errorf("failed to parse return value: %w", err)
		}
	}

	return &entities.ReturnStmt{
		Value: value,
	}, nil
}

// parseDeleteStatement 解析 delete 语句
// 语法：delete(map, key)
func (sp *TokenBasedStatementParser) parseDeleteStatement() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 'delete'

	// 期望 '('
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
		return nil, fmt.Errorf("expected '(' after 'delete'")
	}
	sp.tokenStream.Next() // 消耗 '('

	// 解析map表达式
	mapExpr, err := sp.expressionParser.ParseExpression(sp.tokenStream)
	if err != nil {
		return nil, fmt.Errorf("failed to parse map expression in delete statement: %w", err)
	}

	// 期望 ','
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeComma) {
		return nil, fmt.Errorf("expected ',' after map expression in delete statement")
	}
	sp.tokenStream.Next() // 消耗 ','

	// 解析key表达式
	keyExpr, err := sp.expressionParser.ParseExpression(sp.tokenStream)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key expression in delete statement: %w", err)
	}

	// 期望 ')'
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
		return nil, fmt.Errorf("expected ')' after key expression in delete statement")
	}
	sp.tokenStream.Next() // 消耗 ')'

	return &entities.DeleteStmt{
		Map: mapExpr,
		Key: keyExpr,
	}, nil
}

// parseType 解析类型（支持 chan、标识符、数组、泛型）
func (sp *TokenBasedStatementParser) parseType() (string, error) {
	typeToken := sp.tokenStream.Current()
	if typeToken == nil {
		return "", fmt.Errorf("expected type")
	}

	log.Printf("[DEBUG] parseType: current token=%v (type=%s), position=%d", typeToken, typeToken.Type(), sp.tokenStream.Position())

	// 支持指针类型：*Type 或 *GenericType[T]
	if typeToken.Type() == lexicalVO.EnhancedTokenTypeMultiply {
		log.Printf("[DEBUG] parseType: Found pointer type, consuming '*'")
		sp.tokenStream.Next() // 消耗 '*'
		// 递归解析被指向的类型
		pointedType, err := sp.parseType()
		if err != nil {
			return "", fmt.Errorf("failed to parse pointed type: %w", err)
		}
		result := "*" + pointedType
		log.Printf("[DEBUG] parseType: Parsed pointer type: %s", result)
		return result, nil
	}

	// 支持关键字类型（如 chan）、标识符类型、数组类型和泛型类型
	if typeToken.Type() == lexicalVO.EnhancedTokenTypeChan {
		// 解析 chan 类型，如 chan TaskResult 或 chan Result[string]
		sp.tokenStream.Next() // 消耗 'chan'
		varType := "chan"

		// 检查是否有类型参数（可能是标识符或泛型类型）
		// 使用 parseType 递归解析，支持泛型类型
		nextToken := sp.tokenStream.Current()
		if nextToken != nil {
			// 递归解析类型（支持泛型类型如 Result[string]）
			elementType, err := sp.parseType()
			if err == nil {
				varType = varType + " " + elementType
			}
			// 如果解析失败，可能是其他情况，暂时忽略
		}
		return varType, nil
	} else if typeToken.Type() == lexicalVO.EnhancedTokenTypeLeftBracket {
		// 解析数组类型或切片类型
		// 数组类型：[T]（固定大小数组）
		// 切片类型：[]T（动态切片）
		sp.tokenStream.Next() // 消耗 '['
		
		// 检查是否是切片类型：[]T（两个连续的 ']'）
		if sp.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
			// 这是切片类型：[]T
			sp.tokenStream.Next() // 消耗 ']'
			// 递归解析切片元素类型
			elementType, err := sp.parseType()
			if err != nil {
				return "", fmt.Errorf("failed to parse slice element type: %w", err)
			}
			return "[]" + elementType, nil
		}
		
		// 这是数组类型：[T]
		// 递归解析数组元素类型
		elementType, err := sp.parseType()
		if err != nil {
			return "", fmt.Errorf("failed to parse array element type: %w", err)
		}
		if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
			return "", fmt.Errorf("expected ']' after array element type")
		}
		sp.tokenStream.Next() // 消耗 ']'
		return "[" + elementType + "]", nil
	} else if typeToken.Type() == lexicalVO.EnhancedTokenTypeFunc {
		// 函数类型：func(ParamType1, ParamType2) -> ReturnType
		sp.tokenStream.Next() // 消耗 'func'
		
		// 解析参数列表
		if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftParen) {
			return "", fmt.Errorf("expected '(' after 'func' in function type")
		}
		sp.tokenStream.Next() // 消耗 '('
		
		var paramTypes []string
		if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
			// 有参数
			for {
				paramType, err := sp.parseType()
				if err != nil {
					return "", fmt.Errorf("failed to parse function parameter type: %w", err)
				}
				paramTypes = append(paramTypes, paramType)
				
				if sp.checkToken(lexicalVO.EnhancedTokenTypeRightParen) {
					break
				}
				if !sp.checkToken(lexicalVO.EnhancedTokenTypeComma) {
					return "", fmt.Errorf("expected ',' or ')' in function type parameter list")
				}
				sp.tokenStream.Next() // 消耗 ','
			}
		}
		sp.tokenStream.Next() // 消耗 ')'
		
		// 解析返回类型
		var returnType string
		if sp.checkToken(lexicalVO.EnhancedTokenTypeArrow) {
			sp.tokenStream.Next() // 消耗 '->'
			var err error
			returnType, err = sp.parseType()
			if err != nil {
				return "", fmt.Errorf("failed to parse function return type: %w", err)
			}
		}
		
		// 构建函数类型字符串
		funcType := "func(" + strings.Join(paramTypes, ", ") + ")"
		if returnType != "" {
			funcType = funcType + " -> " + returnType
		}
		return funcType, nil
	} else if typeToken.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
		varType := typeToken.Lexeme()
		sp.tokenStream.Next() // 消耗类型

		// 检查是否是map类型：map[K]V
		if varType == "map" && sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBracket) {
			sp.tokenStream.Next() // 消耗 '['
			
			// 解析键类型
			keyType, err := sp.parseType()
			if err != nil {
				return "", fmt.Errorf("failed to parse map key type: %w", err)
			}
			
			// 检查是否有 ']'
			if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
				return "", fmt.Errorf("expected ']' after map key type")
			}
			sp.tokenStream.Next() // 消耗 ']'
			
			// 解析值类型（在']'之后）
			valueType, err := sp.parseType()
			if err != nil {
				return "", fmt.Errorf("failed to parse map value type: %w", err)
			}
			
			return "map[" + keyType + "]" + valueType, nil
		}

		// 检查是否有泛型参数，如 Result[string] 或 Result[int, string]
		if sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBracket) {
			sp.tokenStream.Next() // 消耗 '['
			var genericParams []string
			for {
				genericParam, err := sp.parseType()
				if err != nil {
					return "", fmt.Errorf("failed to parse generic parameter: %w", err)
				}
				genericParams = append(genericParams, genericParam)
				
				// 检查是否有更多参数
				if sp.checkToken(lexicalVO.EnhancedTokenTypeComma) {
					sp.tokenStream.Next() // 消耗 ','
					continue
				}
				
				// 检查是否结束
				if sp.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
					break
				}
				
				return "", fmt.Errorf("expected ',' or ']' in generic parameters")
			}
			if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
				return "", fmt.Errorf("expected ']' after generic parameters")
			}
			sp.tokenStream.Next() // 消耗 ']'
			return varType + "[" + strings.Join(genericParams, ", ") + "]", nil
		}
		return varType, nil
	} else {
		return "", fmt.Errorf("expected type (got %s)", typeToken.Type())
	}
}

// extractReceiverTypeParams 从接收者类型字符串中提取类型参数
// 例如：*HashMap[K, V] -> [GenericParam{Name: "K"}, GenericParam{Name: "V"}]
// 例如：HashMap[K, V] -> [GenericParam{Name: "K"}, GenericParam{Name: "V"}]
// 例如：Point -> []
func (sp *TokenBasedStatementParser) extractReceiverTypeParams(receiverType string) []entities.GenericParam {
	// 移除指针前缀（如果有）
	typeStr := receiverType
	if strings.HasPrefix(typeStr, "*") {
		typeStr = typeStr[1:]
	}
	
	// 检查是否有泛型参数（包含 '['）
	if !strings.Contains(typeStr, "[") {
		return []entities.GenericParam{} // 非泛型类型，返回空列表
	}
	
	// 找到 '[' 的位置
	leftBracket := strings.Index(typeStr, "[")
	// 提取类型参数部分："K, V]" 或 "K]"
	typeArgsStr := typeStr[leftBracket+1 : len(typeStr)-1]
	
	// 分割类型参数（支持逗号分隔）
	var typeParams []entities.GenericParam
	if typeArgsStr != "" {
		parts := strings.Split(typeArgsStr, ",")
		for _, part := range parts {
			paramName := strings.TrimSpace(part)
			// 检查是否有约束（如 K: Hash）
			if strings.Contains(paramName, ":") {
				constraintParts := strings.SplitN(paramName, ":", 2)
				paramName = strings.TrimSpace(constraintParts[0])
				constraint := strings.TrimSpace(constraintParts[1])
				typeParams = append(typeParams, entities.GenericParam{
					Name:        paramName,
					Constraints: []string{constraint},
				})
			} else {
				typeParams = append(typeParams, entities.GenericParam{
					Name:        paramName,
					Constraints: []string{},
				})
			}
		}
	}
	
	return typeParams
}

// parseImportStatement 解析 import 语句
// 格式：import <path> [as <alias>] [;]
// path 可以是字符串（"types"）或标识符（math）
func (sp *TokenBasedStatementParser) parseImportStatement() (entities.ASTNode, error) {
	log.Printf("[DEBUG] parseImportStatement: current token=%v", sp.tokenStream.Current())
	sp.tokenStream.Next() // 消耗 'import'
	log.Printf("[DEBUG] parseImportStatement: after consuming 'import', current token=%v", sp.tokenStream.Current())
	
	// 解析导入路径（可能是字符串或标识符）
	if sp.tokenStream.IsAtEnd() {
		return nil, fmt.Errorf("expected import path after 'import'")
	}
	pathToken := sp.tokenStream.Current()
	if pathToken == nil {
		return nil, fmt.Errorf("expected import path after 'import'")
	}
	
	// 解析导入路径（可能是字符串或标识符）
	if pathToken.Type() == lexicalVO.EnhancedTokenTypeString {
		sp.tokenStream.Next() // 消耗字符串
	} else if pathToken.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
		sp.tokenStream.Next() // 消耗标识符
	} else {
		return nil, fmt.Errorf("expected string or identifier as import path, got: %s", pathToken.Type())
	}
	
	// ✅ 修复：解析可选的 'as <alias>'
	// 注意：需要检查下一个 token 是否是 'as'（可能是关键字或标识符）
	if !sp.tokenStream.IsAtEnd() {
		nextToken := sp.tokenStream.Current()
		if nextToken != nil {
			// 检查是否是 'as' 关键字
			isAs := nextToken.Type() == lexicalVO.EnhancedTokenTypeAs
			// 或者是否是标识符且 lexeme 是 "as"
			if !isAs && nextToken.Type() == lexicalVO.EnhancedTokenTypeIdentifier && nextToken.Lexeme() == "as" {
				isAs = true
			}
			
			if isAs {
				sp.tokenStream.Next() // 消耗 'as'
				// 解析别名
				if sp.tokenStream.IsAtEnd() {
					return nil, fmt.Errorf("expected alias after 'as'")
				}
				aliasToken := sp.tokenStream.Current()
				if aliasToken == nil || aliasToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
					return nil, fmt.Errorf("expected identifier as alias, got: %s", aliasToken.Type())
				}
				sp.tokenStream.Next() // 消耗别名
			}
		}
	}
	
	// 跳过可选的分号
	if !sp.tokenStream.IsAtEnd() {
		nextToken := sp.tokenStream.Current()
		if nextToken != nil && nextToken.Type() == lexicalVO.EnhancedTokenTypeSemicolon {
			sp.tokenStream.Next() // 消耗分号
		}
	}
	
	// 注意：import 语句在 parser_aggregate.go 中处理，这里只负责解析语法
	// 返回 nil 表示忽略（import 信息会在更高层处理）
	// TODO: 如果需要创建 Import 节点，需要先定义 entities.Import 类型
	return nil, nil
}

// parseFromImportStatement 解析 from ... import 语句
// 格式：from <path> import <element1> [, <element2>] [as <alias>] [;]
// 返回 FromImportStatement 节点，供类型推断将导入符号注册到 functionTable
func (sp *TokenBasedStatementParser) parseFromImportStatement() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 'from'
	
	// 解析导入路径（字符串或标识符）
	if sp.tokenStream.IsAtEnd() {
		return nil, fmt.Errorf("expected import path after 'from'")
	}
	pathToken := sp.tokenStream.Current()
	if pathToken == nil {
		return nil, fmt.Errorf("expected import path after 'from'")
	}
	var importPath string
	if pathToken.Type() == lexicalVO.EnhancedTokenTypeString {
		importPath = pathToken.StringValue()
		sp.tokenStream.Next() // 消耗字符串
	} else if pathToken.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
		importPath = pathToken.Lexeme()
		sp.tokenStream.Next() // 消耗标识符
	} else {
		return nil, fmt.Errorf("expected string or identifier as import path, got: %s", pathToken.Type())
	}
	
	// 解析 'import' 关键字
	if sp.tokenStream.IsAtEnd() {
		return nil, fmt.Errorf("expected 'import' after 'from <path>'")
	}
	importToken := sp.tokenStream.Current()
	if importToken == nil || (importToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier || importToken.Lexeme() != "import") {
		return nil, fmt.Errorf("expected 'import' after 'from <path>', got: %s", importToken.Type())
	}
	sp.tokenStream.Next() // 消耗 'import'
	
	var elements []entities.FromImportElement
	// 解析导入元素列表（element1 [, element2] [as alias]）
	for !sp.tokenStream.IsAtEnd() {
		// 解析元素名
		if !sp.checkToken(lexicalVO.EnhancedTokenTypeIdentifier) {
			return nil, fmt.Errorf("expected identifier as import element")
		}
		elemName := sp.tokenStream.Current().Lexeme()
		sp.tokenStream.Next() // 消耗元素名
		
		elemAlias := ""
		// 检查是否有 'as <alias>'
		if !sp.tokenStream.IsAtEnd() {
			nextToken := sp.tokenStream.Current()
			if nextToken != nil && nextToken.Type() == lexicalVO.EnhancedTokenTypeAs {
				sp.tokenStream.Next() // 消耗 'as'
				// 解析别名
				if sp.tokenStream.IsAtEnd() {
					return nil, fmt.Errorf("expected alias after 'as'")
				}
				aliasToken := sp.tokenStream.Current()
				if aliasToken == nil || aliasToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
					return nil, fmt.Errorf("expected identifier as alias, got: %s", aliasToken.Type())
				}
				elemAlias = aliasToken.Lexeme()
				sp.tokenStream.Next() // 消耗别名
			}
		}
		elements = append(elements, entities.FromImportElement{Name: elemName, Alias: elemAlias})
		
		// 检查是否有更多元素（逗号分隔）
		if !sp.tokenStream.IsAtEnd() {
			nextToken := sp.tokenStream.Current()
			if nextToken != nil && nextToken.Type() == lexicalVO.EnhancedTokenTypeComma {
				sp.tokenStream.Next() // 消耗 ','
				continue // 继续解析下一个元素
			} else if nextToken != nil && nextToken.Type() == lexicalVO.EnhancedTokenTypeSemicolon {
				sp.tokenStream.Next() // 消耗 ';'
				break // 语句结束
			} else {
				// 没有逗号或分号，可能是语句结束（换行）
				break
			}
		} else {
			break
		}
	}
	
	return &entities.FromImportStatement{ImportPath: importPath, Elements: elements}, nil
}

// checkToken 检查当前 token 类型（不消耗）
func (sp *TokenBasedStatementParser) checkToken(tokenType lexicalVO.EnhancedTokenType) bool {
	token := sp.tokenStream.Current()
	return token != nil && token.Type() == tokenType
}
