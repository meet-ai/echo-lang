package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"echo/internal/modules/frontend/domain/entities"
)

// ExpressionParser 表达式解析领域服务
// 职责：解析各种类型的表达式，按复杂度从高到低处理
type ExpressionParser struct {
	parser *ParserAggregate // 引用Parser聚合根，用于递归解析

	// 预编译的正则表达式，用于模式匹配
	commentRegex          *regexp.Regexp // 行内注释
	parenthesesRegex      *regexp.Regexp // 括号表达式
	stringLiteralRegex    *regexp.Regexp // 字符串字面量
	arrayLiteralRegex     *regexp.Regexp // 数组字面量
	lenFuncRegex          *regexp.Regexp // len函数调用
	structAccessRegex     *regexp.Regexp // 结构体字段访问
	errorPropagationRegex *regexp.Regexp // 错误传播操作符
	structLiteralRegex    *regexp.Regexp // 结构体字面量
	funcCallRegex         *regexp.Regexp // 函数调用
	methodCallRegex       *regexp.Regexp // 方法调用
	indexAccessRegex      *regexp.Regexp // 索引访问
	sliceOperationRegex   *regexp.Regexp // 切片操作
	binaryOpRegex         *regexp.Regexp // 二元运算符
	comparisonOpRegex     *regexp.Regexp // 比较运算符
	identifierRegex       *regexp.Regexp // 标识符

	// 特殊表达式正则表达式
	matchExprRegex   *regexp.Regexp // match表达式
	awaitExprRegex   *regexp.Regexp // await表达式
	spawnExprRegex   *regexp.Regexp // spawn表达式
	chanLiteralRegex *regexp.Regexp // 通道字面量
	sendExprRegex    *regexp.Regexp // 发送表达式
	receiveExprRegex *regexp.Regexp // 接收表达式

	// Result/Option字面量正则表达式
	okLiteralRegex   *regexp.Regexp // Ok字面量
	errLiteralRegex  *regexp.Regexp // Err字面量
	someLiteralRegex *regexp.Regexp // Some字面量
}

// NewExpressionParser 创建新的表达式解析器
func NewExpressionParser(parser *ParserAggregate) *ExpressionParser {
	ep := &ExpressionParser{parser: parser}

	// 初始化预编译的正则表达式
	ep.commentRegex = regexp.MustCompile(`\s*//.*$`)                                                      // 行内注释（考虑前导空格）
	ep.parenthesesRegex = regexp.MustCompile(`^\s*\(.*\)\s*$`)                                            // 括号表达式（考虑前后空格）
	ep.stringLiteralRegex = regexp.MustCompile(`^"[^"]*"$`)                                               // 字符串字面量
	ep.arrayLiteralRegex = regexp.MustCompile(`^\s*\[.*\]\s*$`)                                           // 数组字面量
	ep.lenFuncRegex = regexp.MustCompile(`^\s*len\s*\(\s*(.+)\s*\)\s*$`)                                  // len函数调用
	ep.structAccessRegex = regexp.MustCompile(`^([^.]+)\s*\.\s*([^.]+)$`)                                 // 结构体字段访问
	ep.errorPropagationRegex = regexp.MustCompile(`^(.+)\s*\?\s*$`)                                       // 错误传播操作符
	ep.structLiteralRegex = regexp.MustCompile(`^([^}]+)\s*\{\s*(.*)\s*}\s*$`)                            // 结构体字面量
	ep.funcCallRegex = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)\s*\(\s*(.*)\s*\)$`)                  // 函数调用
	ep.methodCallRegex = regexp.MustCompile(`^([^.]+)\s*\.\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\(\s*(.*)\s*\)$`) // 方法调用
	ep.indexAccessRegex = regexp.MustCompile(`^(.+)\s*\[\s*(.+)\s*\]$`)                                   // 索引访问
	ep.sliceOperationRegex = regexp.MustCompile(`^(.+)\s*\[\s*(.*:.*)\s*\]$`)                             // 切片操作
	ep.binaryOpRegex = regexp.MustCompile(`^(.+)\s*([*\/%+\-])\s*(.+)$`)                                  // 二元运算符
	ep.comparisonOpRegex = regexp.MustCompile(`^(.+)\s*(<=|>=|<|>|==|!=)\s*(.+)$`)                        // 比较运算符
	ep.identifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)                                   // 标识符

	// 特殊表达式正则表达式
	ep.matchExprRegex = regexp.MustCompile(`^\s*match\s+`)         // match表达式
	ep.awaitExprRegex = regexp.MustCompile(`^\s*await\s+`)         // await表达式
	ep.spawnExprRegex = regexp.MustCompile(`^\s*spawn\s+`)         // spawn表达式
	ep.chanLiteralRegex = regexp.MustCompile(`^\s*chan\s+|^chan$`) // 通道字面量
	ep.sendExprRegex = regexp.MustCompile(`^(.+)\s*<-\s*(.+)$`)    // 发送表达式
	ep.receiveExprRegex = regexp.MustCompile(`^\s*<-\s*(.+)$`)     // 接收表达式

	// Result/Option字面量正则表达式
	ep.okLiteralRegex = regexp.MustCompile(`^\s*Ok\s*\(\s*(.+)\s*\)\s*$`)        // Ok字面量
	ep.errLiteralRegex = regexp.MustCompile(`^\s*Err\s*\(\s*(.+)\s*\)\s*$`)      // Err字面量
	ep.someLiteralRegex = regexp.MustCompile(`^\s*Some(\s*\(\s*(.+)\s*\))?\s*$`) // Some字面量

	return ep
}

// ParseExpr 解析表达式 - 领域服务核心方法
func (ep *ExpressionParser) ParseExpr(expr string) (entities.Expr, error) {
	expr = strings.TrimSpace(expr)

	// 移除行内注释
	expr = ep.commentRegex.ReplaceAllString(expr, "")

	// 处理括号表达式
	if ep.parenthesesRegex.MatchString(expr) {
		innerExpr := strings.TrimSpace(expr[1 : len(expr)-1])
		return ep.ParseExpr(innerExpr)
	}

	// 字符串字面量
	if ep.stringLiteralRegex.MatchString(expr) {
		return &entities.StringLiteral{
			Value: expr[1 : len(expr)-1],
		}, nil
	}

	// 整数字面量
	if intVal, err := strconv.Atoi(expr); err == nil {
		return &entities.IntLiteral{
			Value: intVal,
		}, nil
	}

	// 浮点数字面量
	if floatVal, err := strconv.ParseFloat(expr, 64); err == nil {
		return &entities.FloatLiteral{
			Value: floatVal,
		}, nil
	}

	// 布尔字面量
	if expr == "true" {
		return &entities.BoolLiteral{Value: true}, nil
	}
	if expr == "false" {
		return &entities.BoolLiteral{Value: false}, nil
	}

	// 特殊表达式处理
	if expr := ep.tryParseSpecialExpressions(expr); expr != nil {
		return expr, nil
	}

	// 数组字面量
	if ep.arrayLiteralRegex.MatchString(expr) {
		return ep.parseArrayLiteral(expr)
	}

	// len函数调用
	if ep.lenFuncRegex.MatchString(expr) {
		return ep.parseLenExpr(expr)
	}

	// 结构体字段访问
	if strings.Contains(expr, ".") {
		return ep.parseStructAccess(expr)
	}

	// Result/Option字面量
	if expr := ep.tryParseResultOption(expr); expr != nil {
		return expr, nil
	}

	// 错误传播操作符
	if ep.errorPropagationRegex.MatchString(expr) {
		return ep.parseErrorPropagation(expr)
	}

	// 结构体字面量
	if strings.Contains(expr, "{") && strings.Contains(expr, "}") && strings.Contains(expr, ":") {
		return ep.parseStructLiteral(expr)
	}

	// 函数调用表达式 (不匹配包含运算符的复杂表达式)
	// 注意：如果表达式包含二元运算符（+、-、*、/、%），则不是函数调用
	if strings.Contains(expr, "(") && strings.Contains(expr, ")") &&
		!strings.Contains(expr, "&&") && !strings.Contains(expr, "||") &&
		!strings.Contains(expr, "==") && !strings.Contains(expr, "!=") &&
		!strings.Contains(expr, "<=") && !strings.Contains(expr, ">=") &&
		!strings.Contains(expr, "<") && !strings.Contains(expr, ">") &&
		!strings.Contains(expr, "+") && !strings.Contains(expr, "-") &&
		!strings.Contains(expr, "*") && !strings.Contains(expr, "/") &&
		!strings.Contains(expr, "%") {
		return ep.parseFuncCallExpr(expr)
	}

	// 方法调用表达式
	if ep.isMethodCall(expr) {
		return ep.parseMethodCallExpr(expr)
	}

	// 索引访问表达式
	if ep.isIndexAccess(expr) {
		return ep.parseIndexExpr(expr)
	}

	// 切片操作表达式
	if ep.isSliceOperation(expr) {
		return ep.parseSliceExpr(expr)
	}

	// 二元运算表达式
	if expr := ep.tryParseBinaryExpr(expr); expr != nil {
		return expr, nil
	}

	// 比较运算表达式
	if expr := ep.tryParseComparisonExpr(expr); expr != nil {
		return expr, nil
	}

	// 逻辑运算表达式
	if expr := ep.tryParseLogicalExpr(expr); expr != nil {
		return expr, nil
	}

	// 标识符（变量名）
	if ep.identifierRegex.MatchString(expr) {
		return &entities.Identifier{Name: expr}, nil
	}

	return nil, fmt.Errorf("unsupported expression: %s", expr)
}

// tryParseSpecialExpressions 尝试解析特殊表达式
func (ep *ExpressionParser) tryParseSpecialExpressions(expr string) entities.Expr {
	// Match表达式
	if ep.matchExprRegex.MatchString(expr) {
		return ep.parseMatchExpr(expr)
	}

	// Await表达式
	if ep.awaitExprRegex.MatchString(expr) {
		return ep.parseAwaitExpr(expr)
	}

	// Spawn表达式
	if ep.spawnExprRegex.MatchString(expr) {
		return ep.parseSpawnExpr(expr)
	}

	// 通道字面量
	if ep.chanLiteralRegex.MatchString(expr) {
		return ep.parseChanLiteral(expr)
	}

	// 发送表达式
	if ep.sendExprRegex.MatchString(expr) {
		return ep.parseSendExpr(expr)
	}

	// 接收表达式
	if ep.receiveExprRegex.MatchString(expr) {
		return ep.parseReceiveExpr(expr)
	}

	return nil
}

// parseArrayLiteral 解析数组字面量
func (ep *ExpressionParser) parseArrayLiteral(expr string) (entities.Expr, error) {
	content := strings.TrimSpace(expr[1 : len(expr)-1])
	var elements []entities.Expr

	if content != "" {
		elementStrs := strings.Split(content, ",")
		for _, elementStr := range elementStrs {
			elementStr = strings.TrimSpace(elementStr)
			if elementStr == "" {
				continue
			}
			element, err := ep.ParseExpr(elementStr)
			if err != nil {
				return nil, fmt.Errorf("invalid array element: %v", err)
			}
			elements = append(elements, element)
		}
	}

	return &entities.ArrayLiteral{Elements: elements}, nil
}

// parseLenExpr 解析len函数调用
func (ep *ExpressionParser) parseLenExpr(expr string) (entities.Expr, error) {
	matches := ep.lenFuncRegex.FindStringSubmatch(expr)
	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid len function call: %s", expr)
	}

	content := strings.TrimSpace(matches[1])
	arrayExpr, err := ep.ParseExpr(content)
	if err != nil {
		return nil, fmt.Errorf("invalid len argument: %v", err)
	}

	return &entities.LenExpr{Array: arrayExpr}, nil
}

// parseStructAccess 解析结构体字段访问
func (ep *ExpressionParser) parseStructAccess(expr string) (entities.Expr, error) {
	matches := ep.structAccessRegex.FindStringSubmatch(expr)
	if len(matches) < 3 {
		return nil, fmt.Errorf("invalid struct access: %s", expr)
	}

	objExpr, err := ep.ParseExpr(strings.TrimSpace(matches[1]))
	if err != nil {
		return nil, fmt.Errorf("invalid struct object: %v", err)
	}

	field := strings.TrimSpace(matches[2])
	return &entities.StructAccess{Object: objExpr, Field: field}, nil
}

// tryParseResultOption 尝试解析Result/Option字面量
func (ep *ExpressionParser) tryParseResultOption(expr string) entities.Expr {
	if ep.okLiteralRegex.MatchString(expr) {
		matches := ep.okLiteralRegex.FindStringSubmatch(expr)
		if len(matches) >= 2 {
			content := strings.TrimSpace(matches[1])
			value, _ := ep.ParseExpr(content)
			return &entities.OkLiteral{Value: value}
		}
	}

	if ep.errLiteralRegex.MatchString(expr) {
		matches := ep.errLiteralRegex.FindStringSubmatch(expr)
		if len(matches) >= 2 {
			content := strings.TrimSpace(matches[1])
			err, _ := ep.ParseExpr(content)
			return &entities.ErrLiteral{Error: err}
		}
	}

	if ep.someLiteralRegex.MatchString(expr) {
		matches := ep.someLiteralRegex.FindStringSubmatch(expr)
		if len(matches) >= 3 {
			content := strings.TrimSpace(matches[2])
			value, _ := ep.ParseExpr(content)
			return &entities.SomeLiteral{Value: value}
		} else {
			return &entities.SomeLiteral{Value: nil}
		}
	}

	if expr == "None" {
		return &entities.NoneLiteral{}
	}

	return nil
}

// parseErrorPropagation 解析错误传播操作符
func (ep *ExpressionParser) parseErrorPropagation(expr string) (entities.Expr, error) {
	matches := ep.errorPropagationRegex.FindStringSubmatch(expr)
	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid error propagation: %s", expr)
	}

	baseExpr := strings.TrimSpace(matches[1])
	base, err := ep.ParseExpr(baseExpr)
	if err != nil {
		return nil, fmt.Errorf("invalid expression in error propagation: %v", err)
	}

	return &entities.ErrorPropagation{Expr: base}, nil
}

// parseStructLiteral 解析结构体字面量
func (ep *ExpressionParser) parseStructLiteral(expr string) (entities.Expr, error) {
	braceIndex := strings.Index(expr, "{")
	if braceIndex == -1 {
		return nil, fmt.Errorf("invalid struct literal: %s", expr)
	}

	typeName := strings.TrimSpace(expr[:braceIndex])
	fieldsStr := strings.TrimSpace(expr[braceIndex+1 : len(expr)-1])

	fields := make(map[string]entities.Expr)
	if fieldsStr != "" {
		fieldPairs := strings.Split(fieldsStr, ",")
		for _, pair := range fieldPairs {
			pair = strings.TrimSpace(pair)
			if strings.Contains(pair, ":") {
				parts := strings.SplitN(pair, ":", 2)
				fieldName := strings.TrimSpace(parts[0])
				fieldValueStr := strings.TrimSpace(parts[1])

				fieldValue, err := ep.ParseExpr(fieldValueStr)
				if err != nil {
					return nil, fmt.Errorf("invalid field value in struct literal: %v", err)
				}
				fields[fieldName] = fieldValue
			}
		}
	}

	return &entities.StructLiteral{Type: typeName, Fields: fields}, nil
}

// parseFuncCallExpr 解析函数调用表达式
func (ep *ExpressionParser) parseFuncCallExpr(expr string) (entities.Expr, error) {
	matches := ep.funcCallRegex.FindStringSubmatch(expr)
	if len(matches) < 3 {
		return nil, fmt.Errorf("invalid function call: %s", expr)
	}

	funcName := strings.TrimSpace(matches[1])
	argsStr := strings.TrimSpace(matches[2])

	var args []entities.Expr
	if argsStr != "" {
		argStrs := strings.Split(argsStr, ",")
		for _, argStr := range argStrs {
			argStr = strings.TrimSpace(argStr)
			if argStr == "" {
				continue
			}
			arg, err := ep.ParseExpr(argStr)
			if err != nil {
				return nil, fmt.Errorf("invalid function argument: %v", err)
			}
			args = append(args, arg)
		}
	}

	return &entities.FuncCall{Name: funcName, Args: args}, nil
}

// isMethodCall 检查是否为方法调用
func (ep *ExpressionParser) isMethodCall(expr string) bool {
	dotIndex := strings.Index(expr, ".")
	if dotIndex == -1 || dotIndex == 0 {
		return false
	}

	// 检查点号后是否有括号
	afterDot := expr[dotIndex+1:]
	return strings.Contains(afterDot, "(") && strings.Contains(afterDot, ")")
}

// parseMethodCallExpr 解析方法调用表达式
func (ep *ExpressionParser) parseMethodCallExpr(expr string) (entities.Expr, error) {
	matches := ep.methodCallRegex.FindStringSubmatch(expr)
	if len(matches) < 4 {
		return nil, fmt.Errorf("invalid method call: %s", expr)
	}

	receiverStr := strings.TrimSpace(matches[1])
	methodName := strings.TrimSpace(matches[2])
	argsStr := strings.TrimSpace(matches[3])

	receiver, err := ep.ParseExpr(receiverStr)
	if err != nil {
		return nil, fmt.Errorf("invalid receiver in method call: %v", err)
	}

	var args []entities.Expr
	if argsStr != "" {
		argStrs := strings.Split(argsStr, ",")
		for _, argStr := range argStrs {
			argStr = strings.TrimSpace(argStr)
			if argStr == "" {
				continue
			}
			arg, err := ep.ParseExpr(argStr)
			if err != nil {
				return nil, fmt.Errorf("invalid method argument: %v", err)
			}
			args = append(args, arg)
		}
	}

	return &entities.MethodCallExpr{
		Receiver:   receiver,
		MethodName: methodName,
		Args:       args,
	}, nil
}

// isIndexAccess 检查是否为索引访问
func (ep *ExpressionParser) isIndexAccess(expr string) bool {
	return strings.Contains(expr, "[") && strings.Contains(expr, "]") &&
		strings.Index(expr, "[") < strings.LastIndex(expr, "]") &&
		!strings.Contains(expr, ":") // 不匹配包含冒号的表达式（那是切片操作）
}

// parseIndexExpr 解析索引访问表达式
func (ep *ExpressionParser) parseIndexExpr(expr string) (entities.Expr, error) {
	matches := ep.indexAccessRegex.FindStringSubmatch(expr)
	if len(matches) < 3 {
		return nil, fmt.Errorf("invalid index access: %s", expr)
	}

	arrayStr := strings.TrimSpace(matches[1])
	indexStr := strings.TrimSpace(matches[2])

	arrayExpr, err := ep.ParseExpr(arrayStr)
	if err != nil {
		return nil, fmt.Errorf("invalid array expression in index: %v", err)
	}

	indexExpr, err := ep.ParseExpr(indexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid index expression: %v", err)
	}

	return &entities.IndexExpr{Array: arrayExpr, Index: indexExpr}, nil
}

// isSliceOperation 检查是否为切片操作
func (ep *ExpressionParser) isSliceOperation(expr string) bool {
	bracketCount := strings.Count(expr, "[")
	return bracketCount == 1 && strings.Contains(expr, ":")
}

// parseSliceExpr 解析切片操作表达式
func (ep *ExpressionParser) parseSliceExpr(expr string) (entities.Expr, error) {
	matches := ep.sliceOperationRegex.FindStringSubmatch(expr)
	if len(matches) < 3 {
		return nil, fmt.Errorf("invalid slice operation: %s", expr)
	}

	arrayStr := strings.TrimSpace(matches[1])
	sliceStr := strings.TrimSpace(matches[2])

	arrayExpr, err := ep.ParseExpr(arrayStr)
	if err != nil {
		return nil, fmt.Errorf("invalid array expression in slice: %v", err)
	}

	var startExpr, endExpr entities.Expr

	if strings.Contains(sliceStr, ":") {
		parts := strings.Split(sliceStr, ":")
		if len(parts) >= 1 && parts[0] != "" {
			startExpr, err = ep.ParseExpr(parts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid start expression in slice: %v", err)
			}
		}
		if len(parts) >= 2 && parts[1] != "" {
			endExpr, err = ep.ParseExpr(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid end expression in slice: %v", err)
			}
		}
	}

	return &entities.SliceExpr{
		Array: arrayExpr,
		Start: startExpr,
		End:   endExpr,
	}, nil
}

// tryParseBinaryExpr 尝试解析二元运算表达式
func (ep *ExpressionParser) tryParseBinaryExpr(expr string) entities.Expr {
	matches := ep.binaryOpRegex.FindStringSubmatch(expr)
	if len(matches) >= 4 {
		left, err1 := ep.ParseExpr(strings.TrimSpace(matches[1]))
		op := matches[2]
		right, err2 := ep.ParseExpr(strings.TrimSpace(matches[3]))
		if err1 == nil && err2 == nil {
			return &entities.BinaryExpr{Left: left, Op: op, Right: right}
		}
	}

	return nil
}

// tryParseComparisonExpr 尝试解析比较运算表达式
func (ep *ExpressionParser) tryParseComparisonExpr(expr string) entities.Expr {
	matches := ep.comparisonOpRegex.FindStringSubmatch(expr)
	if len(matches) >= 4 {
		left, err1 := ep.ParseExpr(strings.TrimSpace(matches[1]))
		op := matches[2]
		right, err2 := ep.ParseExpr(strings.TrimSpace(matches[3]))
		if err1 == nil && err2 == nil {
			return &entities.BinaryExpr{Left: left, Op: op, Right: right}
		}
	}

	return nil
}

// tryParseLogicalExpr 尝试解析逻辑运算表达式
func (ep *ExpressionParser) tryParseLogicalExpr(expr string) entities.Expr {
	// 处理逻辑与
	if strings.Contains(expr, "&&") {
		parts := strings.Split(expr, "&&")
		if len(parts) == 2 {
			left, err1 := ep.ParseExpr(strings.TrimSpace(parts[0]))
			right, err2 := ep.ParseExpr(strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil {
				return &entities.BinaryExpr{Left: left, Op: "&&", Right: right}
			}
		}
	}

	// 处理逻辑或
	if strings.Contains(expr, "||") {
		parts := strings.Split(expr, "||")
		if len(parts) == 2 {
			left, err1 := ep.ParseExpr(strings.TrimSpace(parts[0]))
			right, err2 := ep.ParseExpr(strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil {
				return &entities.BinaryExpr{Left: left, Op: "||", Right: right}
			}
		}
	}

	return nil
}

// isIdentifier 检查是否为有效标识符
func (ep *ExpressionParser) isIdentifier(s string) bool {
	return ep.identifierRegex.MatchString(s)
}

// parseMatchExpr 解析match表达式
func (ep *ExpressionParser) parseMatchExpr(expr string) entities.Expr {
	// TODO: 实现match表达式解析
	return &entities.MatchExpr{Value: nil, Cases: []entities.MatchCase{}}
}

// parseAwaitExpr 解析await表达式
func (ep *ExpressionParser) parseAwaitExpr(expr string) entities.Expr {
	inner := strings.TrimSpace(expr[6:]) // Remove "await "
	innerExpr, _ := ep.ParseExpr(inner)
	return &entities.AwaitExpr{Expression: innerExpr}
}

// parseSpawnExpr 解析spawn表达式
func (ep *ExpressionParser) parseSpawnExpr(expr string) entities.Expr {
	inner := strings.TrimSpace(expr[6:]) // Remove "spawn "
	innerExpr, _ := ep.ParseExpr(inner)
	return &entities.SpawnExpr{Function: innerExpr}
}

// parseChanLiteral 解析通道字面量
func (ep *ExpressionParser) parseChanLiteral(expr string) entities.Expr {
	parts := strings.Fields(expr)
	if len(parts) >= 2 {
		elementType := parts[1]
		return &entities.ChanLiteral{Type: elementType}
	}
	return &entities.ChanLiteral{Type: "interface{}"}
}

// parseSendExpr 解析发送表达式
func (ep *ExpressionParser) parseSendExpr(expr string) entities.Expr {
	matches := ep.sendExprRegex.FindStringSubmatch(expr)
	if len(matches) >= 3 {
		chanExpr, _ := ep.ParseExpr(strings.TrimSpace(matches[1]))
		valueExpr, _ := ep.ParseExpr(strings.TrimSpace(matches[2]))
		return &entities.SendExpr{Channel: chanExpr, Value: valueExpr}
	}
	return nil
}

// parseReceiveExpr 解析接收表达式
func (ep *ExpressionParser) parseReceiveExpr(expr string) entities.Expr {
	matches := ep.receiveExprRegex.FindStringSubmatch(expr)
	if len(matches) >= 2 {
		chanStr := strings.TrimSpace(matches[1])
		chanExpr, _ := ep.ParseExpr(chanStr)
		return &entities.ReceiveExpr{Channel: chanExpr}
	}
	return nil
}
