package parser

import (
	"fmt"

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
	case lexicalVO.EnhancedTokenTypeFor:
		return sp.parseForStatement()
	case lexicalVO.EnhancedTokenTypeFunc, lexicalVO.EnhancedTokenTypeAsync:
		return sp.parseFunctionDefinition()
	case lexicalVO.EnhancedTokenTypeStruct:
		return sp.parseStructDefinition()
	case lexicalVO.EnhancedTokenTypeEnum:
		return sp.parseEnumDefinition()
	case lexicalVO.EnhancedTokenTypeTrait:
		return sp.parseTraitDefinition()
	case lexicalVO.EnhancedTokenTypeMatch:
		return sp.parseMatchStatement()
	case lexicalVO.EnhancedTokenTypeReturn:
		return sp.parseReturnStatement()
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
		// 可能是赋值语句或表达式语句
		return sp.parseAssignmentOrExpressionStatement()
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
func (sp *TokenBasedStatementParser) parseVariableDeclaration() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 'let'

	// 解析变量名
	nameToken := sp.tokenStream.Current()
	if nameToken == nil || nameToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
		return nil, fmt.Errorf("expected identifier after 'let'")
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
	expr, err := sp.expressionParser.ParseExpression(sp.tokenStream)
	if err != nil {
		return nil, err
	}

	// 检查下一个 token 是否是 '=' 或 ':'
	nextToken := sp.tokenStream.Current()
	if nextToken != nil && nextToken.Type() == lexicalVO.EnhancedTokenTypeColon {
		// 遇到 ':'，说明这可能是类型注解，但当前 token 不是 'let'，这是语法错误
		// 或者这是结构体字面量的字段，但当前上下文不对
		// 或者这是 select case 的语法（case msg := <- ch1:），但 select 语句还未实现
		return nil, fmt.Errorf("unexpected ':' after identifier (expected '=' for assignment, or this should be a 'let' statement, or this is a 'select case' which is not yet implemented)")
	}
	if nextToken != nil && nextToken.Type() == lexicalVO.EnhancedTokenTypeAssign {
		// 这是赋值语句，但表达式解析器已经消耗了标识符
		// 我们需要从表达式中提取标识符名
		// 如果表达式是标识符，提取名称；否则，这是语法错误
		var name string
		if ident, ok := expr.(*entities.Identifier); ok {
			name = ident.Name
		} else {
			return nil, fmt.Errorf("assignment target must be an identifier, got %T", expr)
		}

		// 消耗 '='
		sp.tokenStream.Next()

		// 解析赋值右侧的值
		value, err := sp.expressionParser.ParseExpression(sp.tokenStream)
		if err != nil {
			return nil, err
		}

		return &entities.AssignStmt{
			Name:  name,
			Value: value,
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
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after if condition")
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

// parseForStatement 解析 for 语句
func (sp *TokenBasedStatementParser) parseForStatement() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 'for'

	// 解析初始化语句（可选）
	var init entities.ASTNode
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
	var condition entities.Expr
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
		Init:      init,
		Condition: condition,
		Increment: increment,
		Body:      body,
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

	// 解析函数名
	nameToken := sp.tokenStream.Current()
	if nameToken == nil || nameToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
		return nil, fmt.Errorf("expected function name")
	}
	name := nameToken.Lexeme()
	sp.tokenStream.Next() // 消耗函数名

	// 解析参数列表
	params, err := sp.parseParameterList()
	if err != nil {
		return nil, err
	}

	// 解析返回类型（可选）
	// Echo 语言使用 -> 箭头语法：func name() -> ReturnType
	var returnType string
	if sp.checkToken(lexicalVO.EnhancedTokenTypeArrow) {
		sp.tokenStream.Next() // 消耗 '->'
		typeToken := sp.tokenStream.Current()
		if typeToken == nil || typeToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
			return nil, fmt.Errorf("expected return type after '->'")
		}
		returnType = typeToken.Lexeme()
		sp.tokenStream.Next() // 消耗返回类型

		// 处理泛型类型，如 Result[string]
		// 如果下一个 token 是 '['，解析泛型参数
		if sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBracket) {
			sp.tokenStream.Next() // 消耗 '['
			genericParamToken := sp.tokenStream.Current()
			if genericParamToken != nil && genericParamToken.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
				returnType = returnType + "[" + genericParamToken.Lexeme() + "]"
				sp.tokenStream.Next() // 消耗泛型参数
			}
			if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
				return nil, fmt.Errorf("expected ']' after generic parameter")
			}
			sp.tokenStream.Next() // 消耗 ']'
		}
	}

	// 解析函数体
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBrace) {
		return nil, fmt.Errorf("expected '{' after function signature")
	}
	body, err := sp.parseBlock()
	if err != nil {
		return nil, err
	}

	if isAsync {
		return &entities.AsyncFuncDef{
			Name:       name,
			Params:     params,
			ReturnType: returnType,
			Body:       body,
		}, nil
	}

	return &entities.FuncDef{
		Name:       name,
		Params:     params,
		ReturnType: returnType,
		Body:       body,
	}, nil
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
	// 返回一个 BlockStmt（如果存在）或第一个语句
	if len(statements) == 0 {
		return nil, nil
	}
	if len(statements) == 1 {
		return statements[0], nil
	}
	// TODO: 创建 BlockStmt 节点
	return statements[0], nil
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

		// 解析字段类型
		typeToken := sp.tokenStream.Current()
		if typeToken == nil || typeToken.Type() != lexicalVO.EnhancedTokenTypeIdentifier {
			return nil, fmt.Errorf("expected field type after ':' for field '%s'", fieldName)
		}
		fieldType := typeToken.Lexeme()
		sp.tokenStream.Next() // 消耗类型

		// 添加字段
		fields = append(fields, entities.StructField{
			Name: fieldName,
			Type: fieldType,
		})

		// 检查是否有逗号（可选）
		if sp.checkToken(lexicalVO.EnhancedTokenTypeComma) {
			sp.tokenStream.Next() // 消耗 ','
		}
	}

	// 解析右大括号
	if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBrace) {
		return nil, fmt.Errorf("expected '}' to close struct definition")
	}
	sp.tokenStream.Next() // 消耗 '}'

	return &entities.StructDef{
		Name:   structName,
		Fields: fields,
	}, nil
}

// parseEnumDefinition 解析枚举定义
func (sp *TokenBasedStatementParser) parseEnumDefinition() (entities.ASTNode, error) {
	// TODO: 实现枚举定义解析
	return nil, fmt.Errorf("enum definition parsing not yet implemented")
}

// parseTraitDefinition 解析 trait 定义
func (sp *TokenBasedStatementParser) parseTraitDefinition() (entities.ASTNode, error) {
	// TODO: 实现 trait 定义解析
	return nil, fmt.Errorf("trait definition parsing not yet implemented")
}

// parseMatchStatement 解析 match 语句
func (sp *TokenBasedStatementParser) parseMatchStatement() (entities.ASTNode, error) {
	sp.tokenStream.Next() // 消耗 'match'

	// 解析匹配的表达式
	expr, err := sp.expressionParser.ParseExpression(sp.tokenStream)
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
		// 解析 pattern（暂时作为表达式处理）
		pattern, err := sp.expressionParser.ParseExpression(sp.tokenStream)
		if err != nil {
			return nil, fmt.Errorf("failed to parse match pattern: %w", err)
		}

		// 解析 =>
		if !sp.checkToken(lexicalVO.EnhancedTokenTypeFatArrow) {
			currentToken := sp.tokenStream.Current()
			return nil, fmt.Errorf("expected '=>' after match pattern (got %s)", currentToken.Type())
		}
		sp.tokenStream.Next() // 消耗 '=>'

		// 解析 statement
		stmt, err := sp.ParseStatement(sp.tokenStream)
		if err != nil {
			return nil, fmt.Errorf("failed to parse match case statement: %w", err)
		}

		// 如果解析 statement 后当前 token 是 `=>`，说明表达式解析器跳过了下一个 case 的 pattern
		// 跳过这个 `=>`，继续解析下一个 case
		currentTokenAfterStmt := sp.tokenStream.Current()
		if currentTokenAfterStmt != nil && currentTokenAfterStmt.Type() == lexicalVO.EnhancedTokenTypeFatArrow {
			sp.tokenStream.Next() // 跳过 `=>`
		}

		// TODO: 构建 MatchCase（需要 pattern 和 statement）
		_ = pattern
		_ = stmt
		cases = append(cases, entities.MatchCase{}) // TODO: 填充 MatchCase

		// 如果解析 statement 后当前 token 是 `=>`，说明表达式解析器跳过了下一个 case 的 pattern
		// 这是一个已知问题：表达式解析器在解析 return 语句的返回值时，可能跳过了中间的 token
		// 临时修复：跳过这个 `=>`，继续解析下一个 case
		// TODO: 修复表达式解析器，确保它在遇到 `)` 时正确停止
		if currentTokenAfterStmt != nil && currentTokenAfterStmt.Type() == lexicalVO.EnhancedTokenTypeFatArrow {
			sp.tokenStream.Next() // 跳过 `=>`
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

// parseType 解析类型（支持 chan、标识符、数组、泛型）
func (sp *TokenBasedStatementParser) parseType() (string, error) {
	typeToken := sp.tokenStream.Current()
	if typeToken == nil {
		return "", fmt.Errorf("expected type")
	}

	// 支持关键字类型（如 chan）、标识符类型、数组类型和泛型类型
	if typeToken.Type() == lexicalVO.EnhancedTokenTypeChan {
		// 解析 chan 类型，如 chan TaskResult
		sp.tokenStream.Next() // 消耗 'chan'
		varType := "chan"

		// 检查是否有类型参数（如 TaskResult）
		nextToken := sp.tokenStream.Current()
		if nextToken != nil && nextToken.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
			varType = varType + " " + nextToken.Lexeme()
			sp.tokenStream.Next() // 消耗类型参数
		}
		return varType, nil
	} else if typeToken.Type() == lexicalVO.EnhancedTokenTypeLeftBracket {
		// 解析数组类型，如 [string] 或 [Future[Result[string]]]
		sp.tokenStream.Next() // 消耗 '['
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
	} else if typeToken.Type() == lexicalVO.EnhancedTokenTypeIdentifier {
		varType := typeToken.Lexeme()
		sp.tokenStream.Next() // 消耗类型

		// 检查是否有泛型参数，如 Result[string]
		if sp.checkToken(lexicalVO.EnhancedTokenTypeLeftBracket) {
			sp.tokenStream.Next() // 消耗 '['
			genericParam, err := sp.parseType()
			if err != nil {
				return "", fmt.Errorf("failed to parse generic parameter: %w", err)
			}
			if !sp.checkToken(lexicalVO.EnhancedTokenTypeRightBracket) {
				return "", fmt.Errorf("expected ']' after generic parameter")
			}
			sp.tokenStream.Next() // 消耗 ']'
			return varType + "[" + genericParam + "]", nil
		}
		return varType, nil
	} else {
		return "", fmt.Errorf("expected type (got %s)", typeToken.Type())
	}
}

// checkToken 检查当前 token 类型（不消耗）
func (sp *TokenBasedStatementParser) checkToken(tokenType lexicalVO.EnhancedTokenType) bool {
	token := sp.tokenStream.Current()
	return token != nil && token.Type() == tokenType
}
