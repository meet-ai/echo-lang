// Package services 定义语法分析上下文的领域服务
package services

import (
	"context"
	"fmt"

	lexicalVO "echo/internal/modules/frontend/domain/lexical/value_objects"
	sharedVO "echo/internal/modules/frontend/domain/shared/value_objects"
	syntaxVO "echo/internal/modules/frontend/domain/syntax/value_objects"
)

// LRActionType LR动作类型
type LRActionType int

const (
	LRActionShift  LRActionType = iota // 移进
	LRActionReduce                     // 归约
	LRActionAccept                     // 接受
	LRActionError                      // 错误
)

// LRAction LR动作
type LRAction struct {
	Type      LRActionType
	State     int         // 移进到的状态（移进动作）
	Production int        // 归约的产生式编号（归约动作）
	Symbol    string      // 归约的非终结符（归约动作）
}

// LRParseTable LR解析表
type LRParseTable struct {
	// Action表：state -> terminal -> action
	ActionTable map[int]map[string]LRAction

	// Goto表：state -> non-terminal -> state
	GotoTable map[int]map[string]int

	// 产生式列表
	Productions []LRProduction
}

// LRProduction LR产生式
type LRProduction struct {
	LeftHandSide  string   // 左部非终结符
	RightHandSide []string // 右部符号序列
}

// LRParserState LR解析器状态
// 表示LR解析器在某一时刻的完整状态
type LRParserState struct {
	StateStack  []int         // 状态栈
	SymbolStack []interface{} // 符号栈（AST节点或Token）
	InputIndex  int           // 当前输入位置
	Lookahead   *lexicalVO.EnhancedToken // 预读Token
}

// NewLRParserState 创建新的LR解析器状态
func NewLRParserState() *LRParserState {
	return &LRParserState{
		StateStack:  []int{0}, // 初始状态为0
		SymbolStack: make([]interface{}, 0),
		InputIndex:  0,
		Lookahead:   nil,
	}
}

// PushState 推入新状态到状态栈
func (state *LRParserState) PushState(newState int) {
	state.StateStack = append(state.StateStack, newState)
}

// PopStates 弹出指定数量的状态
func (state *LRParserState) PopStates(count int) []int {
	if count <= 0 || len(state.StateStack) < count {
		return nil
	}

	popped := make([]int, count)
	for i := 0; i < count; i++ {
		index := len(state.StateStack) - 1 - i
		popped[count-1-i] = state.StateStack[index]
	}

	state.StateStack = state.StateStack[:len(state.StateStack)-count]
	return popped
}

// PushSymbol 推入符号到符号栈
func (state *LRParserState) PushSymbol(symbol interface{}) {
	state.SymbolStack = append(state.SymbolStack, symbol)
}

// PopSymbols 弹出指定数量的符号
func (state *LRParserState) PopSymbols(count int) []interface{} {
	if count <= 0 || len(state.SymbolStack) < count {
		return nil
	}

	popped := make([]interface{}, count)
	for i := 0; i < count; i++ {
		index := len(state.SymbolStack) - 1 - i
		popped[count-1-i] = state.SymbolStack[index]
	}

	state.SymbolStack = state.SymbolStack[:len(state.SymbolStack)-count]
	return popped
}

// CurrentState 获取当前状态
func (state *LRParserState) CurrentState() int {
	if len(state.StateStack) == 0 {
		return -1
	}
	return state.StateStack[len(state.StateStack)-1]
}

// StackSize 获取状态栈大小
func (state *LRParserState) StackSize() int {
	return len(state.StateStack)
}

// TopSymbol 获取栈顶符号
func (state *LRParserState) TopSymbol() interface{} {
	if len(state.SymbolStack) == 0 {
		return nil
	}
	return state.SymbolStack[len(state.SymbolStack)-1]
}

// Clone 克隆状态（用于回退）
func (state *LRParserState) Clone() *LRParserState {
	newState := &LRParserState{
		StateStack:  make([]int, len(state.StateStack)),
		SymbolStack: make([]interface{}, len(state.SymbolStack)),
		InputIndex:  state.InputIndex,
		Lookahead:   state.Lookahead,
	}

	copy(newState.StateStack, state.StateStack)
	copy(newState.SymbolStack, state.SymbolStack)

	return newState
}

// LRAmbiguityResolver LR歧义解析器
// 专门处理语法歧义，特别是泛型与比较运算符的歧义
type LRAmbiguityResolver struct {
	// 解析表
	parseTable *syntaxVO.LRParseTable

	// Token流
	tokenStream *lexicalVO.EnhancedTokenStream

	// 错误收集
	errors []*sharedVO.ParseError

	// 状态机状态
	currentState *LRParserState
}

// NewLRAmbiguityResolver 创建新的LR歧义解析器
func NewLRAmbiguityResolver() *LRAmbiguityResolver {
	resolver := &LRAmbiguityResolver{
		parseTable: createGenericAmbiguityParseTable(),
		errors:     make([]*sharedVO.ParseError, 0),
	}

	return resolver
}

// ResolveAmbiguity 解析歧义表达式
// 主要用于处理泛型语法与比较运算符的歧义
func (resolver *LRAmbiguityResolver) ResolveAmbiguity(
	ctx context.Context,
	tokenStream *lexicalVO.EnhancedTokenStream,
	startPosition int,
) (*sharedVO.ASTNode, error) {

	resolver.tokenStream = tokenStream

	// 初始化解析器状态
	state := NewLRParserState()
	state.InputIndex = startPosition

	// 设置当前状态引用
	resolver.currentState = state

	// 主解析循环
	maxSteps := 1000 // 防止无限循环
	steps := 0
	backtrackPoints := make([]*LRParserState, 0) // 回退点

	for !resolver.isAtEnd(state) && steps < maxSteps {
		steps++

		// 检查上下文取消
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		currentToken := resolver.currentToken(state)
		if currentToken == nil {
			break
		}

		currentState := state.CurrentState()
		tokenSymbol := resolver.tokenToSymbol(currentToken)

		// 查询Action表
		action, exists := resolver.parseTable.GetAction(currentState, tokenSymbol)
		if !exists {
			// 没有找到动作，尝试回退
			if success := resolver.tryBacktrack(state, backtrackPoints); success {
				continue // 回退成功，继续解析
			}
			// 回退失败，返回nil表示这不是歧义语法
			return nil, nil
		}

		// 在执行动作前保存回退点（对于非确定性动作）
		if resolver.shouldSaveBacktrackPoint(action) {
			backtrackState := state.Clone()
			backtrackPoints = append(backtrackPoints, backtrackState)
		}

		// 执行动作
		err := resolver.executeAction(state, action, currentToken)
		if err != nil {
			// 执行失败，尝试回退
			if success := resolver.tryBacktrack(state, backtrackPoints); success {
				continue // 回退成功，继续解析
			}
			return nil, err
		}

		// 检查是否接受
		if action.ActionType == syntaxVO.LRActionAccept {
			result := resolver.getParseResult(state)
			return &result, nil
		}

		// 检查是否出现错误
		if action.ActionType == syntaxVO.LRActionError {
			// 尝试回退
			if success := resolver.tryBacktrack(state, backtrackPoints); success {
				continue // 回退成功，继续解析
			}
			return nil, nil // 不是歧义语法
		}
	}

	// 如果到达这里，说明解析未完成或失败
	return nil, nil
}

// executeAction 执行LR动作
func (resolver *LRAmbiguityResolver) executeAction(state *LRParserState, action syntaxVO.LRAction, token *lexicalVO.EnhancedToken) error {
	switch action.ActionType {
	case syntaxVO.LRActionShift:
		return resolver.performShift(state, token, action.State)

	case syntaxVO.LRActionReduce:
		return resolver.performReduce(state, action)

	case syntaxVO.LRActionAccept:
		// 接受动作在调用处处理
		return nil

	case syntaxVO.LRActionError:
		// 错误动作在调用处处理
		return nil

	default:
		return fmt.Errorf("unknown action type: %v", action.ActionType)
	}
}

// performShift 执行移进动作
func (resolver *LRAmbiguityResolver) performShift(state *LRParserState, token *lexicalVO.EnhancedToken, nextState int) error {
	// 将Token推入符号栈
	state.PushSymbol(token)
	// 将新状态推入状态栈
	state.PushState(nextState)
	// 前进输入位置
	state.InputIndex++
	return nil
}

// performReduce 执行归约动作
func (resolver *LRAmbiguityResolver) performReduce(state *LRParserState, action syntaxVO.LRAction) error {
	production, exists := resolver.parseTable.GetProduction(action.Production)
	if !exists {
		return fmt.Errorf("invalid production index: %d", action.Production)
	}

	// 检查栈深度
	if state.StackSize() < production.Length() {
		return fmt.Errorf("stack underflow during reduction")
	}

	// 弹出右部符号
	rightHandSymbols := state.PopSymbols(production.Length())
	if len(rightHandSymbols) != production.Length() {
		return fmt.Errorf("failed to pop %d symbols from stack", production.Length())
	}

	// 弹出相应数量的状态
	poppedStates := state.PopStates(production.Length())
	if len(poppedStates) != production.Length() {
		return fmt.Errorf("failed to pop %d states from stack", production.Length())
	}

	// 创建归约结果（AST节点）
	reducedNode := resolver.createReducedNode(production.LeftHandSide, rightHandSymbols)

	// 将归约结果推入符号栈
	state.PushSymbol(reducedNode)

	// 查询Goto表获取新状态
	currentState := state.CurrentState()
	nextState, exists := resolver.parseTable.GetGoto(currentState, production.LeftHandSide)
	if !exists {
		return fmt.Errorf("no goto entry for state %d and non-terminal %s", currentState, production.LeftHandSide)
	}

	// 将新状态推入状态栈
	state.PushState(nextState)

	return nil
}

// createReducedNode 创建归约节点
func (resolver *LRAmbiguityResolver) createReducedNode(lhs string, rhs []interface{}) interface{} {
	// 根据产生式左部和右部创建相应的AST节点
	switch lhs {
	case "GenericType":
		// 创建泛型类型节点
		if len(rhs) >= 3 {
			// GenericType -> Identifier '<' TypeList '>'
			baseType := rhs[0].(*lexicalVO.EnhancedToken).Identifier()
			// 暂时返回字符串表示，实际应该创建GenericType节点
			return fmt.Sprintf("GenericType{%s<...>}", baseType)
		}
		// 如果参数不够，返回默认值
		return "GenericType{invalid}"
	case "TypeList":
		// 类型列表
		return "TypeList{...}"
	default:
		// 默认处理
		return fmt.Sprintf("%s{...}", lhs)
	}
}

// getParseResult 获取解析结果
func (resolver *LRAmbiguityResolver) getParseResult(state *LRParserState) sharedVO.ASTNode {
	if len(state.SymbolStack) == 0 {
		return nil
	}

	// 栈顶应该是一个完整的AST节点
	result := state.SymbolStack[len(state.SymbolStack)-1]

	// 转换为sharedVO.ASTNode接口
	// 这里需要根据实际的节点类型进行转换
	// 暂时返回一个简单的字符串节点
	// 使用默认位置信息
	location := sharedVO.NewSourceLocation("", 1, 1, 0)
	return sharedVO.NewStringLiteral(fmt.Sprintf("LR_PARSED: %v", result), location)
}

// currentToken 获取当前位置的Token
func (resolver *LRAmbiguityResolver) currentToken(state *LRParserState) *lexicalVO.EnhancedToken {
	if resolver.isAtEnd(state) {
		return nil
	}
	return resolver.tokenStream.Tokens()[state.InputIndex]
}

// isAtEnd 检查是否到达输入末尾
func (resolver *LRAmbiguityResolver) isAtEnd(state *LRParserState) bool {
	return state.InputIndex >= len(resolver.tokenStream.Tokens()) ||
		   resolver.currentToken(state).Type() == lexicalVO.EnhancedTokenTypeEOF
}

// tokenToSymbol 将Token转换为文法符号
func (resolver *LRAmbiguityResolver) tokenToSymbol(token *lexicalVO.EnhancedToken) string {
	switch token.Type() {
	case lexicalVO.EnhancedTokenTypeLessThan:
		return "<"
	case lexicalVO.EnhancedTokenTypeGreaterThan:
		return ">"
	case lexicalVO.EnhancedTokenTypeIdentifier:
		return "IDENTIFIER"
	case lexicalVO.EnhancedTokenTypeLeftParen:
		return "("
	case lexicalVO.EnhancedTokenTypeRightParen:
		return ")"
	case lexicalVO.EnhancedTokenTypeComma:
		return ","
	case lexicalVO.EnhancedTokenTypeLeftBrace:
		return "{"
	case lexicalVO.EnhancedTokenTypeRightBrace:
		return "}"
	case lexicalVO.EnhancedTokenTypeLeftBracket:
		return "["
	case lexicalVO.EnhancedTokenTypeRightBracket:
		return "]"
	case lexicalVO.EnhancedTokenTypeSemicolon:
		return ";"
	case lexicalVO.EnhancedTokenTypeColon:
		return ":"
	case lexicalVO.EnhancedTokenTypeDot:
		return "."
	case lexicalVO.EnhancedTokenTypeAssign:
		return "="
	case lexicalVO.EnhancedTokenTypePlus:
		return "+"
	case lexicalVO.EnhancedTokenTypeMinus:
		return "-"
	case lexicalVO.EnhancedTokenTypeMultiply:
		return "*"
	case lexicalVO.EnhancedTokenTypeDivide:
		return "/"
	case lexicalVO.EnhancedTokenTypeEqual:
		return "=="
	case lexicalVO.EnhancedTokenTypeNotEqual:
		return "!="
	case lexicalVO.EnhancedTokenTypeEOF:
		return "" // EOF用空字符串表示
	default:
		// 对于其他Token，直接返回词素
		return token.Lexeme()
	}
}

// createGenericAmbiguityParseTable 创建处理泛型歧义的LR解析表
func createGenericAmbiguityParseTable() *syntaxVO.LRParseTable {
	table := syntaxVO.NewLRParseTable()

	// 产生式定义
	// 0: GenericType -> IDENTIFIER '<' TypeList '>'
	// 1: TypeList -> IDENTIFIER
	// 2: TypeList -> TypeList ',' IDENTIFIER

	prod0 := syntaxVO.NewLRProduction("GenericType", []string{"IDENTIFIER", "<", "TypeList", ">"})
	prod1 := syntaxVO.NewLRProduction("TypeList", []string{"IDENTIFIER"})
	prod2 := syntaxVO.NewLRProduction("TypeList", []string{"TypeList", ",", "IDENTIFIER"})

	table.AddProduction(prod0)
	table.AddProduction(prod1)
	table.AddProduction(prod2)

	// Action表和Goto表
	// 状态0: 开始状态
	table.AddAction(0, "IDENTIFIER", syntaxVO.NewLRAction(syntaxVO.LRActionShift, 1, -1))
	table.AddGoto(0, "GenericType", 2)

	// 状态1: 读取了标识符
	table.AddAction(1, "<", syntaxVO.NewLRAction(syntaxVO.LRActionShift, 3, -1))
	// 没有<时可以作为普通标识符处理（这里不添加accept，因为这不是完整的泛型类型）

	// 状态2: 读取了<
	table.AddAction(2, "IDENTIFIER", syntaxVO.NewLRAction(syntaxVO.LRActionShift, 4, -1))
	table.AddGoto(2, "TypeList", 5)

	// 状态3: 读取了类型名
	table.AddAction(3, ">", syntaxVO.NewLRAction(syntaxVO.LRActionShift, 6, -1))
	table.AddAction(3, ",", syntaxVO.NewLRAction(syntaxVO.LRActionShift, 7, -1))

	// 状态4: 读取了>，泛型类型完成
	table.AddAction(4, "", syntaxVO.NewLRAction(syntaxVO.LRActionAccept, -1, -1)) // EOF接受

	// 状态5: 读取了TypeList
	table.AddAction(5, ">", syntaxVO.NewLRAction(syntaxVO.LRActionReduce, -1, 1)) // 归约TypeList -> IDENTIFIER

	// 状态6: 读取了下一个类型名
	table.AddAction(6, ">", syntaxVO.NewLRAction(syntaxVO.LRActionReduce, -1, 2)) // 归约TypeList -> TypeList ',' IDENTIFIER
	table.AddAction(6, ",", syntaxVO.NewLRAction(syntaxVO.LRActionShift, 7, -1))
	table.AddGoto(6, "TypeList", 8)

	// 状态7: 读取了,
	table.AddAction(7, "IDENTIFIER", syntaxVO.NewLRAction(syntaxVO.LRActionShift, 4, -1))

	// 状态8: 归约后的TypeList（多参数）
	table.AddAction(8, ">", syntaxVO.NewLRAction(syntaxVO.LRActionReduce, -1, 2))

	return table
}

// GetErrors 获取解析错误
func (resolver *LRAmbiguityResolver) GetErrors() []*sharedVO.ParseError {
	return resolver.errors
}

// shouldSaveBacktrackPoint 检查是否应该保存回退点
func (resolver *LRAmbiguityResolver) shouldSaveBacktrackPoint(action syntaxVO.LRAction) bool {
	// 对于归约动作，保存回退点，因为可能有其他解析路径
	return action.ActionType == syntaxVO.LRActionReduce
}

// tryBacktrack 尝试回退到上一个保存点
func (resolver *LRAmbiguityResolver) tryBacktrack(currentState *LRParserState, backtrackPoints []*LRParserState) bool {
	if len(backtrackPoints) == 0 {
		return false // 没有回退点
	}

	// 弹出最后一个回退点
	lastBacktrackPoint := backtrackPoints[len(backtrackPoints)-1]
	backtrackPoints = backtrackPoints[:len(backtrackPoints)-1]

	// 恢复状态
	*currentState = *lastBacktrackPoint

	// 记录回退操作
	resolver.AddError("backtracked to previous state", sharedVO.NewSourceLocation("", 1, 1, currentState.InputIndex))

	return true
}

// TryAlternativeParse 尝试替代解析路径
// 当标准LR解析失败时，可以尝试其他解析策略
func (resolver *LRAmbiguityResolver) TryAlternativeParse(
	ctx context.Context,
	tokenStream *lexicalVO.EnhancedTokenStream,
	startPosition int,
) (sharedVO.ASTNode, error) {

	// 策略1: 尝试作为比较表达式解析
	comparisonResult := resolver.tryParseAsComparison(tokenStream, startPosition)
	if comparisonResult != nil {
		return comparisonResult, nil
	}

	// 策略2: 尝试作为泛型表达式解析
	genericResult := resolver.tryParseAsGeneric(tokenStream, startPosition)
	if genericResult != nil {
		return genericResult, nil
	}

	// 都没有成功，返回nil
	return nil, nil
}

// tryParseAsComparison 尝试作为比较表达式解析
func (resolver *LRAmbiguityResolver) tryParseAsComparison(tokenStream *lexicalVO.EnhancedTokenStream, startPosition int) sharedVO.ASTNode {
	// 简化的比较表达式解析逻辑
	// 这里可以实现更复杂的比较表达式识别
	if startPosition+2 < len(tokenStream.Tokens()) {
		token1 := tokenStream.Tokens()[startPosition]
		token2 := tokenStream.Tokens()[startPosition+1]
		token3 := tokenStream.Tokens()[startPosition+2]

		// 检查模式: identifier < identifier
		if token1.Type() == lexicalVO.EnhancedTokenTypeIdentifier &&
		   token2.Type() == lexicalVO.EnhancedTokenTypeLessThan &&
		   token3.Type() == lexicalVO.EnhancedTokenTypeIdentifier {

			// 创建比较表达式
			leftLocation := token1.Location()
			rightLocation := token3.Location()
			// 使用左操作数的位置作为表达式的起始位置
			expressionLocation := leftLocation
			
			left := sharedVO.NewIdentifier(token1.Identifier(), leftLocation)
			right := sharedVO.NewIdentifier(token3.Identifier(), rightLocation)
			comparison := sharedVO.NewBinaryExpression(left, "<", right, expressionLocation)

			return comparison
		}
	}

	return nil
}

// tryParseAsGeneric 尝试作为泛型表达式解析
func (resolver *LRAmbiguityResolver) tryParseAsGeneric(tokenStream *lexicalVO.EnhancedTokenStream, startPosition int) sharedVO.ASTNode {
	// 简化的泛型表达式解析逻辑
	// 这里可以实现更复杂的泛型识别
	if startPosition+3 < len(tokenStream.Tokens()) {
		token1 := tokenStream.Tokens()[startPosition]
		token2 := tokenStream.Tokens()[startPosition+1]
		token3 := tokenStream.Tokens()[startPosition+2]
		token4 := tokenStream.Tokens()[startPosition+3]

		// 检查模式: identifier < identifier >
		if token1.Type() == lexicalVO.EnhancedTokenTypeIdentifier &&
		   token2.Type() == lexicalVO.EnhancedTokenTypeLessThan &&
		   token3.Type() == lexicalVO.EnhancedTokenTypeIdentifier &&
		   token4.Type() == lexicalVO.EnhancedTokenTypeGreaterThan {

			// 创建泛型类型表示（简化）
			genericStr := fmt.Sprintf("GenericType{%s<%s>}",
				token1.Identifier(), token3.Identifier())
			// 使用第一个token的位置作为泛型类型的位置
			genericLocation := token1.Location()
			generic := sharedVO.NewStringLiteral(genericStr, genericLocation)

			return generic
		}
	}

	return nil
}

// AddError 添加错误
func (resolver *LRAmbiguityResolver) AddError(message string, location sharedVO.SourceLocation) {
	error := sharedVO.NewParseError(message, location,
		sharedVO.ErrorTypeSyntax, sharedVO.SeverityError)
	resolver.errors = append(resolver.errors, error)
}
