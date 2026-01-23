package parser

import (
	"fmt"
	"regexp"
	"strings"

	"echo/internal/modules/frontend/domain/entities"
)

// StatementParser 语句解析领域服务
// 职责：识别语句类型和初步解析，调用BlockExtractor处理块结构
type StatementParser struct {
	parser         *ParserAggregate // 引用Parser聚合根
	blockExtractor *BlockExtractor

	// 预编译的正则表达式，用于模式匹配
	printStmtRegex     *regexp.Regexp // print语句
	letStmtRegex       *regexp.Regexp // let语句
	assignStmtRegex    *regexp.Regexp // 赋值语句
	ifStmtRegex        *regexp.Regexp // if语句
	whileStmtRegex     *regexp.Regexp // while语句
	forStmtRegex       *regexp.Regexp // for语句
	funcStmtRegex      *regexp.Regexp // func语句
	structStmtRegex    *regexp.Regexp // struct语句
	enumStmtRegex      *regexp.Regexp // enum语句
	traitStmtRegex     *regexp.Regexp // trait语句
	implStmtRegex      *regexp.Regexp // impl语句
	asyncFuncStmtRegex *regexp.Regexp // async函数语句
	matchStmtRegex     *regexp.Regexp // match语句
	selectStmtRegex    *regexp.Regexp // select语句
	returnStmtRegex    *regexp.Regexp // return语句
	breakStmtRegex     *regexp.Regexp // break语句
	continueStmtRegex  *regexp.Regexp // continue语句
}

// NewStatementParser 创建新的语句解析器
func NewStatementParser(parser *ParserAggregate, blockExtractor *BlockExtractor) *StatementParser {
	sp := &StatementParser{
		parser:         parser,
		blockExtractor: blockExtractor,
	}

	// 初始化预编译的正则表达式
	sp.printStmtRegex = regexp.MustCompile(`^\s*print\s+`)
	sp.letStmtRegex = regexp.MustCompile(`^\s*let\s+`)
	sp.assignStmtRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*\s*=\s*`)
	sp.ifStmtRegex = regexp.MustCompile(`^\s*if\s+`)
	sp.whileStmtRegex = regexp.MustCompile(`^\s*while\s+`)
	sp.forStmtRegex = regexp.MustCompile(`^\s*for\s+`)
	sp.funcStmtRegex = regexp.MustCompile(`^\s*(async\s+)?func\s+`)
	sp.structStmtRegex = regexp.MustCompile(`^\s*struct\s+.*\{\s*$`)
	sp.enumStmtRegex = regexp.MustCompile(`^\s*enum\s+`)
	sp.traitStmtRegex = regexp.MustCompile(`^\s*trait\s+`)
	sp.implStmtRegex = regexp.MustCompile(`^\s*@impl\s+`)
	sp.matchStmtRegex = regexp.MustCompile(`^\s*match\s+`)
	sp.selectStmtRegex = regexp.MustCompile(`^\s*select\s*`)
	sp.returnStmtRegex = regexp.MustCompile(`^\s*return\s*`)
	sp.breakStmtRegex = regexp.MustCompile(`^\s*break\s*$`)
	sp.continueStmtRegex = regexp.MustCompile(`^\s*continue\s*$`)

	return sp
}

// ParseStatement 解析单个语句 - 领域服务核心方法
func (sp *StatementParser) ParseStatement(line string, lineNum int) (entities.ASTNode, error) {
	line = strings.TrimSpace(line)

	// 跳过块结束符
	if line == "}" {
		return nil, nil // 忽略块结束符
	}

	// 各种语句类型处理（按优先级顺序）
	if sp.printStmtRegex.MatchString(line) {
		return sp.parsePrintStatement(line, lineNum)
	}

	if sp.letStmtRegex.MatchString(line) {
		return sp.parseVariableDeclaration(line, lineNum)
	}

	if sp.assignStmtRegex.MatchString(line) && !strings.Contains(line, "==") {
		return sp.parseAssignment(line, lineNum)
	}

	if sp.ifStmtRegex.MatchString(line) {
		return sp.parseIfStatement(line, lineNum)
	}

	if sp.whileStmtRegex.MatchString(line) {
		return sp.parseWhileStatement(line, lineNum)
	}

	if sp.forStmtRegex.MatchString(line) {
		return sp.parseForStatement(line, lineNum)
	}

	if sp.funcStmtRegex.MatchString(line) {
		return sp.parseFunctionDefinition(line, lineNum)
	}

	if sp.structStmtRegex.MatchString(line) {
		return sp.parseStructDefinition(line, lineNum)
	}

	if sp.enumStmtRegex.MatchString(line) {
		return sp.parseEnumDefinition(line, lineNum)
	}

	if sp.traitStmtRegex.MatchString(line) {
		return sp.parseTraitDefinition(line, lineNum)
	}

	if sp.implStmtRegex.MatchString(line) {
		return sp.parseImplDefinition(line, lineNum)
	}

	if sp.matchStmtRegex.MatchString(line) {
		return sp.parseMatchStatement(line, lineNum)
	}

	if sp.selectStmtRegex.MatchString(line) {
		return sp.parseSelectStatement(line, lineNum)
	}

	if sp.returnStmtRegex.MatchString(line) {
		return sp.parseReturnStatement(line, lineNum)
	}

	if sp.breakStmtRegex.MatchString(line) {
		return &entities.BreakStmt{}, nil
	}

	if sp.continueStmtRegex.MatchString(line) {
		return &entities.ContinueStmt{}, nil
	}

	// 异步表达式语句
	if sp.isAsyncExpressionStatement(line) {
		return sp.parseAsyncExpressionStatement(line, lineNum)
	}

	// 方法调用语句
	if sp.isMethodCallStatement(line) {
		return sp.parseMethodCallStatement(line, lineNum)
	}

	// 函数调用语句
	if sp.isFunctionCallStatement(line) {
		return sp.parseFunctionCallStatement(line, lineNum)
	}

	return nil, fmt.Errorf("line %d: unknown statement: %s", lineNum, line)
}

// parsePrintStatement 解析print语句
func (sp *StatementParser) parsePrintStatement(line string, lineNum int) (entities.ASTNode, error) {
	exprStr := strings.TrimSpace(sp.printStmtRegex.ReplaceAllString(line, ""))
	if exprStr == "" {
		return nil, fmt.Errorf("line %d: print statement requires expression", lineNum)
	}

	expr, err := sp.parser.expressionParser.ParseExpr(exprStr)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid print expression: %v", lineNum, err)
	}

	return &entities.PrintStmt{Value: expr}, nil
}

// parseVariableDeclaration 解析变量声明
func (sp *StatementParser) parseVariableDeclaration(line string, lineNum int) (entities.ASTNode, error) {
	content := strings.TrimSpace(sp.letStmtRegex.ReplaceAllString(line, ""))

	var name, typeStr, valueStr string
	var inferred bool

	// 查找"="的位置
	equalIndex := strings.Index(content, " = ")
	if equalIndex == -1 {
		equalIndex = strings.Index(content, "=")
		if equalIndex == -1 {
			return nil, fmt.Errorf("line %d: variable declaration requires value", lineNum)
		}
	} else {
		equalIndex += 1 // 指向"="
	}

	beforeEqual := strings.TrimSpace(content[:equalIndex])
	valueStr = strings.TrimSpace(content[equalIndex+1:]) // +1 for "="

	// 检查是否有类型声明
	if colonIndex := strings.Index(beforeEqual, ":"); colonIndex >= 0 {
		name = strings.TrimSpace(beforeEqual[:colonIndex])
		typeStr = strings.TrimSpace(beforeEqual[colonIndex+1:])
	} else {
		name = beforeEqual
		inferred = true
	}

	// 解析值表达式
	value, err := sp.parser.expressionParser.ParseExpr(valueStr)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid variable initializer: %v", lineNum, err)
	}

	return &entities.VarDecl{
		Name:     name,
		Type:     typeStr,
		Value:    value,
		Inferred: inferred,
	}, nil
}

// parseAssignment 解析赋值语句
func (sp *StatementParser) parseAssignment(line string, lineNum int) (entities.ASTNode, error) {
	// 手动查找"="的位置，因为assignStmtRegex匹配整个模式
	equalIndex := strings.Index(line, " = ")
	if equalIndex == -1 {
		// 尝试没有空格的"="
		equalIndex = strings.Index(line, "=")
		if equalIndex == -1 {
			return nil, fmt.Errorf("line %d: invalid assignment statement", lineNum)
		}
	} else {
		// 包含空格的情况，equalIndex指向" ="的开始
		equalIndex += 1 // 指向"="
	}

	varName := strings.TrimSpace(line[:equalIndex])
	valueStr := strings.TrimSpace(line[equalIndex+1:]) // +1 for "="

	if varName == "" {
		return nil, fmt.Errorf("line %d: assignment requires variable name", lineNum)
	}

	value, err := sp.parser.expressionParser.ParseExpr(valueStr)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid assignment value: %v", lineNum, err)
	}

	return &entities.AssignStmt{Name: varName, Value: value}, nil
}

// parseIfStatement 解析if语句
func (sp *StatementParser) parseIfStatement(line string, lineNum int) (entities.ASTNode, error) {
	conditionStr := strings.TrimSpace(sp.ifStmtRegex.ReplaceAllString(line, ""))

	// 查找条件结束的位置（通常是" {"）
	braceIndex := strings.Index(conditionStr, " {")
	if braceIndex == -1 {
		return nil, fmt.Errorf("line %d: invalid if statement syntax", lineNum)
	}

	conditionStr = strings.TrimSpace(conditionStr[:braceIndex])

	condition, err := sp.parser.expressionParser.ParseExpr(conditionStr)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid if condition: %v", lineNum, err)
	}

	return &entities.IfStmt{
		Condition: condition,
		ThenBody:  []entities.ASTNode{}, // 后续填充
		ElseBody:  []entities.ASTNode{}, // 后续填充
	}, nil
}

// parseWhileStatement 解析while循环语句
func (sp *StatementParser) parseWhileStatement(line string, lineNum int) (entities.ASTNode, error) {
	conditionStr := strings.TrimSpace(sp.whileStmtRegex.ReplaceAllString(line, ""))

	braceIndex := strings.Index(conditionStr, " {")
	if braceIndex == -1 {
		return nil, fmt.Errorf("line %d: invalid while statement syntax", lineNum)
	}

	conditionStr = strings.TrimSpace(conditionStr[:braceIndex])

	condition, err := sp.parser.expressionParser.ParseExpr(conditionStr)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid while condition: %v", lineNum, err)
	}

	return &entities.WhileStmt{
		Condition: condition,
		Body:      []entities.ASTNode{}, // 后续填充
	}, nil
}

// parseForStatement 解析for循环语句
func (sp *StatementParser) parseForStatement(line string, lineNum int) (entities.ASTNode, error) {
	conditionStr := strings.TrimSpace(sp.forStmtRegex.ReplaceAllString(line, ""))

	braceIndex := strings.Index(conditionStr, " {")
	if braceIndex == -1 {
		return nil, fmt.Errorf("line %d: invalid for statement syntax", lineNum)
	}

	conditionStr = strings.TrimSpace(conditionStr[:braceIndex])

	condition, err := sp.parser.expressionParser.ParseExpr(conditionStr)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid for condition: %v", lineNum, err)
	}

	return &entities.ForStmt{
		Condition: condition,
		Body:      []entities.ASTNode{}, // 后续填充
	}, nil
}

// parseFunctionDefinition 解析函数定义
func (sp *StatementParser) parseFunctionDefinition(line string, lineNum int) (entities.ASTNode, error) {
	isAsync := strings.Contains(line, "async")
	content := strings.TrimSpace(sp.funcStmtRegex.ReplaceAllString(line, ""))

	// 解析函数签名
	parenIndex := strings.Index(content, "(")
	if parenIndex == -1 {
		return nil, fmt.Errorf("line %d: invalid function definition", lineNum)
	}

	funcName := strings.TrimSpace(content[:parenIndex])

	// 解析参数列表
	paramsEndIndex := strings.Index(content, ")")
	if paramsEndIndex == -1 {
		return nil, fmt.Errorf("line %d: invalid function parameters", lineNum)
	}

	paramsStr := content[parenIndex+1 : paramsEndIndex]
	var params []entities.Param

	if paramsStr != "" {
		paramPairs := strings.Split(paramsStr, ",")
		for _, paramPair := range paramPairs {
			paramPair = strings.TrimSpace(paramPair)
			if paramPair == "" {
				continue
			}

			if colonIndex := strings.Index(paramPair, ":"); colonIndex >= 0 {
				paramName := strings.TrimSpace(paramPair[:colonIndex])
				paramType := strings.TrimSpace(paramPair[colonIndex+1:])
				params = append(params, entities.Param{Name: paramName, Type: paramType})
			}
		}
	}

	// 解析返回类型
	var returnType string
	remaining := strings.TrimSpace(content[paramsEndIndex+1:])
	if strings.HasPrefix(remaining, "-> ") {
		returnType = strings.TrimSpace(remaining[3:])
	}

	return &entities.FuncDef{
		Name:       funcName,
		Params:     params,
		ReturnType: returnType,
		Body:       []entities.ASTNode{}, // 后续填充
		IsAsync:    isAsync,
	}, nil
}

// parseStructDefinition 解析结构体定义
func (sp *StatementParser) parseStructDefinition(line string, lineNum int) (entities.ASTNode, error) {
	content := strings.TrimSpace(sp.structStmtRegex.ReplaceAllString(line, ""))

	// content现在是空的，因为regex匹配了整个行
	// 需要重新解析行
	content = strings.TrimSpace(strings.TrimPrefix(line, "struct"))
	content = strings.TrimSpace(content)

	braceIndex := strings.Index(content, "{")
	if braceIndex == -1 {
		return nil, fmt.Errorf("line %d: invalid struct definition", lineNum)
	}

	structName := strings.TrimSpace(content[:braceIndex])

	return &entities.StructDef{
		Name:   structName,
		Fields: []entities.StructField{}, // 后续填充
	}, nil
}

// parseTraitDefinition 解析trait定义
func (sp *StatementParser) parseTraitDefinition(line string, lineNum int) (entities.ASTNode, error) {
	content := strings.TrimSpace(sp.traitStmtRegex.ReplaceAllString(line, ""))

	spaceIndex := strings.Index(content, " ")
	if spaceIndex == -1 {
		return nil, fmt.Errorf("line %d: invalid trait definition", lineNum)
	}

	traitName := strings.TrimSpace(content[:spaceIndex])

	return &entities.TraitDef{
		Name:    traitName,
		Methods: []entities.TraitMethod{}, // 后续填充
	}, nil
}

// parseImplDefinition 解析impl定义
func (sp *StatementParser) parseImplDefinition(line string, lineNum int) (entities.ASTNode, error) {
	content := strings.TrimSpace(sp.implStmtRegex.ReplaceAllString(line, ""))

	return &entities.ImplDef{
		TraitName: content,
		Methods:   []entities.MethodDef{}, // 后续填充
	}, nil
}

// parseEnumDefinition 解析枚举定义（只解析开始部分，变体由状态机处理）
func (sp *StatementParser) parseEnumDefinition(line string, lineNum int) (entities.ASTNode, error) {
	fmt.Printf("DEBUG: parseEnumDefinition called with line: '%s'\n", line)
	content := strings.TrimSpace(sp.enumStmtRegex.ReplaceAllString(line, ""))

	// 找到枚举名称和左大括号
	braceIndex := strings.Index(content, "{")
	if braceIndex == -1 {
		return nil, fmt.Errorf("line %d: enum definition missing opening brace", lineNum)
	}

	enumName := strings.TrimSpace(content[:braceIndex])

	// 验证枚举名称
	if enumName == "" {
		return nil, fmt.Errorf("line %d: enum name cannot be empty", lineNum)
	}

	// 只返回枚举定义的开始部分，变体将在枚举状态下逐行解析
	enumDef := &entities.EnumDef{
		Name:     enumName,
		Variants: []entities.EnumVariant{}, // 变体将在状态机中填充
	}
	fmt.Printf("DEBUG: parseEnumDefinition returning EnumDef: %s\n", enumName)
	return enumDef, nil
}

// parseMatchStatement 解析match语句
func (sp *StatementParser) parseMatchStatement(line string, lineNum int) (entities.ASTNode, error) {
	content := strings.TrimSpace(sp.matchStmtRegex.ReplaceAllString(line, ""))

	braceIndex := strings.Index(content, " {")
	if braceIndex == -1 {
		return nil, fmt.Errorf("line %d: invalid match statement", lineNum)
	}

	valueStr := strings.TrimSpace(content[:braceIndex])

	value, err := sp.parser.expressionParser.ParseExpr(valueStr)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid match value: %v", lineNum, err)
	}

	return &entities.MatchStmt{
		Value: value,
		Cases: []entities.MatchCase{}, // 后续填充
	}, nil
}

// parseReturnStatement 解析return语句
func (sp *StatementParser) parseReturnStatement(line string, lineNum int) (entities.ASTNode, error) {
	returnStr := strings.TrimSpace(sp.returnStmtRegex.ReplaceAllString(line, ""))

	var value entities.Expr
	if returnStr != "" {
		var err error
		value, err = sp.parser.expressionParser.ParseExpr(returnStr)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid return value: %v", lineNum, err)
		}
	}

	return &entities.ReturnStmt{Value: value}, nil
}

// isAsyncExpressionStatement 检查是否为异步表达式语句
func (sp *StatementParser) isAsyncExpressionStatement(line string) bool {
	return strings.HasPrefix(line, "await ") ||
		strings.HasPrefix(line, "spawn ") ||
		strings.HasPrefix(line, "chan") ||
		strings.Contains(line, " <- ") ||
		strings.HasPrefix(line, "<- ")
}

// parseAsyncExpressionStatement 解析异步表达式语句
func (sp *StatementParser) parseAsyncExpressionStatement(line string, lineNum int) (entities.ASTNode, error) {
	expr, err := sp.parser.expressionParser.ParseExpr(line)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid async expression: %v", lineNum, err)
	}

	return &entities.ExprStmt{Expression: expr}, nil
}

// isMethodCallStatement 检查是否为方法调用语句
func (sp *StatementParser) isMethodCallStatement(line string) bool {
	dotIndex := strings.Index(line, ".")
	if dotIndex == -1 || dotIndex == 0 {
		return false
	}

	afterDot := line[dotIndex+1:]
	return strings.Contains(afterDot, "(") && strings.Contains(afterDot, ")") &&
		!strings.Contains(line, " = ") // 排除赋值中的方法调用
}

// parseMethodCallStatement 解析方法调用语句
func (sp *StatementParser) parseMethodCallStatement(line string, lineNum int) (entities.ASTNode, error) {
	expr, err := sp.parser.expressionParser.ParseExpr(line)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid method call: %v", lineNum, err)
	}

	return &entities.ExprStmt{Expression: expr}, nil
}

// isFunctionCallStatement 检查是否为函数调用语句
func (sp *StatementParser) isFunctionCallStatement(line string) bool {
	return strings.Contains(line, "(") && strings.Contains(line, ")") &&
		!strings.Contains(line, ".") && // 不是方法调用
		!sp.isAsyncExpressionStatement(line) && // 不是异步表达式
		!strings.Contains(line, " = ") // 不是赋值中的函数调用
}

// parseFunctionCallStatement 解析函数调用语句
func (sp *StatementParser) parseFunctionCallStatement(line string, lineNum int) (entities.ASTNode, error) {
	expr, err := sp.parser.expressionParser.ParseExpr(line)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid function call: %v", lineNum, err)
	}

	return &entities.ExprStmt{Expression: expr}, nil
}

// calculateStatementComplexity 计算语句复杂度
func (sp *StatementParser) calculateStatementComplexity(stmt entities.ASTNode) int {
	complexity := 1

	switch s := stmt.(type) {
	case *entities.IfStmt:
		complexity += 2 // 条件判断 + 分支
		if len(s.ElseBody) > 0 {
			complexity += 1 // else分支
		}
	case *entities.WhileStmt, *entities.ForStmt:
		complexity += 3 // 循环控制 + 条件判断
	case *entities.FuncDef:
		complexity += len(s.Params) + 2 // 参数处理 + 函数定义
	case *entities.MatchStmt:
		complexity += len(s.Cases) + 1 // 匹配分支
	}

	return complexity
}

// parseSelectStatement 解析select语句
func (sp *StatementParser) parseSelectStatement(line string, lineNum int) (entities.ASTNode, error) {
	// select语句格式：select { case ... }
	braceIndex := strings.Index(line, "{")
	if braceIndex == -1 {
		return nil, fmt.Errorf("line %d: invalid select statement", lineNum)
	}

	// 创建空的select语句，case将在后续处理中填充
	return &entities.SelectStmt{
		Cases: []entities.SelectCase{}, // 后续填充
	}, nil
}
