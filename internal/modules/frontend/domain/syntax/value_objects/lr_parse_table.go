// Package value_objects 定义语法分析上下文的值对象
package value_objects

import "fmt"

// LRActionType LR动作类型枚举
type LRActionType int

const (
	LRActionShift  LRActionType = iota // 移进动作
	LRActionReduce                     // 归约动作
	LRActionAccept                     // 接受动作
	LRActionError                      // 错误动作
)

// String 返回动作类型的字符串表示
func (at LRActionType) String() string {
	switch at {
	case LRActionShift:
		return "shift"
	case LRActionReduce:
		return "reduce"
	case LRActionAccept:
		return "accept"
	case LRActionError:
		return "error"
	default:
		return "unknown"
	}
}

// LRAction LR动作值对象
type LRAction struct {
	ActionType LRActionType // 动作类型
	State      int          // 移进到的状态（仅对移进动作有效）
	Production int          // 归约的产生式编号（仅对归约动作有效）
}

// NewLRAction 创建新的LR动作
func NewLRAction(actionType LRActionType, state int, production int) LRAction {
	return LRAction{
		ActionType: actionType,
		State:      state,
		Production: production,
	}
}

// String 返回LR动作的字符串表示
func (action LRAction) String() string {
	switch action.ActionType {
	case LRActionShift:
		return fmt.Sprintf("shift(%d)", action.State)
	case LRActionReduce:
		return fmt.Sprintf("reduce(%d)", action.Production)
	case LRActionAccept:
		return "accept"
	case LRActionError:
		return "error"
	default:
		return "unknown"
	}
}

// Equals 比较两个LR动作是否相等（值对象语义）
func (action LRAction) Equals(other LRAction) bool {
	return action.ActionType == other.ActionType &&
		   action.State == other.State &&
		   action.Production == other.Production
}

// LRProduction LR产生式值对象
type LRProduction struct {
	LeftHandSide  string   // 左部非终结符
	RightHandSide []string // 右部符号序列
	SemanticAction func([]interface{}) interface{} // 语义动作（可选）
}

// NewLRProduction 创建新的LR产生式
func NewLRProduction(lhs string, rhs []string) LRProduction {
	return LRProduction{
		LeftHandSide:  lhs,
		RightHandSide: rhs,
		SemanticAction: nil,
	}
}

// NewLRProductionWithAction 创建带语义动作的LR产生式
func NewLRProductionWithAction(lhs string, rhs []string, action func([]interface{}) interface{}) LRProduction {
	return LRProduction{
		LeftHandSide:  lhs,
		RightHandSide: rhs,
		SemanticAction: action,
	}
}

// String 返回LR产生式的字符串表示
func (prod LRProduction) String() string {
	rhsStr := ""
	for i, symbol := range prod.RightHandSide {
		if i > 0 {
			rhsStr += " "
		}
		rhsStr += symbol
	}
	return fmt.Sprintf("%s -> %s", prod.LeftHandSide, rhsStr)
}

// Length 返回产生式右部的长度
func (prod LRProduction) Length() int {
	return len(prod.RightHandSide)
}

// LRParseTable LR解析表值对象
// 实现LR语法分析的核心数据结构
type LRParseTable struct {
	// Action表：状态 -> 终结符 -> 动作
	actionTable map[int]map[string]LRAction

	// Goto表：状态 -> 非终结符 -> 状态
	gotoTable map[int]map[string]int

	// 产生式列表
	productions []LRProduction

	// 元信息
	numStates     int // 状态数量
	numTerminals  int // 终结符数量
	numNonTerminals int // 非终结符数量
}

// NewLRParseTable 创建新的LR解析表
func NewLRParseTable() *LRParseTable {
	return &LRParseTable{
		actionTable:    make(map[int]map[string]LRAction),
		gotoTable:      make(map[int]map[string]int),
		productions:    make([]LRProduction, 0),
		numStates:      0,
		numTerminals:   0,
		numNonTerminals: 0,
	}
}

// AddAction 添加Action表条目
func (table *LRParseTable) AddAction(state int, symbol string, action LRAction) {
	if table.actionTable[state] == nil {
		table.actionTable[state] = make(map[string]LRAction)
	}
	table.actionTable[state][symbol] = action

	// 更新状态数量
	if state >= table.numStates {
		table.numStates = state + 1
	}
}

// AddGoto 添加Goto表条目
func (table *LRParseTable) AddGoto(state int, nonTerminal string, nextState int) {
	if table.gotoTable[state] == nil {
		table.gotoTable[state] = make(map[string]int)
	}
	table.gotoTable[state][nonTerminal] = nextState

	// 更新状态数量
	if state >= table.numStates {
		table.numStates = state + 1
	}
	if nextState >= table.numStates {
		table.numStates = nextState + 1
	}
}

// AddProduction 添加产生式
func (table *LRParseTable) AddProduction(production LRProduction) int {
	table.productions = append(table.productions, production)
	return len(table.productions) - 1 // 返回产生式编号
}

// GetAction 获取Action表条目
func (table *LRParseTable) GetAction(state int, symbol string) (LRAction, bool) {
	if stateActions, exists := table.actionTable[state]; exists {
		if action, exists := stateActions[symbol]; exists {
			return action, true
		}
	}
	return LRAction{ActionType: LRActionError}, false
}

// GetGoto 获取Goto表条目
func (table *LRParseTable) GetGoto(state int, nonTerminal string) (int, bool) {
	if stateGotos, exists := table.gotoTable[state]; exists {
		if nextState, exists := stateGotos[nonTerminal]; exists {
			return nextState, true
		}
	}
	return -1, false
}

// GetProduction 获取产生式
func (table *LRParseTable) GetProduction(index int) (LRProduction, bool) {
	if index >= 0 && index < len(table.productions) {
		return table.productions[index], true
	}
	return LRProduction{}, false
}

// GetProductions 获取所有产生式
func (table *LRParseTable) GetProductions() []LRProduction {
	// 返回副本以保护内部状态
	productions := make([]LRProduction, len(table.productions))
	copy(productions, table.productions)
	return productions
}

// NumStates 获取状态数量
func (table *LRParseTable) NumStates() int {
	return table.numStates
}

// NumProductions 获取产生式数量
func (table *LRParseTable) NumProductions() int {
	return len(table.productions)
}

// Validate 验证解析表的完整性
func (table *LRParseTable) Validate() []string {
	errors := make([]string, 0)

	// 检查所有状态都有至少一个动作或goto
	for state := 0; state < table.numStates; state++ {
		hasAction := len(table.actionTable[state]) > 0
		hasGoto := len(table.gotoTable[state]) > 0

		if !hasAction && !hasGoto {
			errors = append(errors, fmt.Sprintf("state %d has no actions or gotos", state))
		}
	}

	// 检查产生式编号的有效性
	for state := 0; state < table.numStates; state++ {
		if stateActions, exists := table.actionTable[state]; exists {
			for symbol, action := range stateActions {
				if action.ActionType == LRActionReduce {
					if action.Production < 0 || action.Production >= len(table.productions) {
						errors = append(errors, fmt.Sprintf("state %d, symbol %s: invalid production index %d", state, symbol, action.Production))
					}
				}
			}
		}
	}

	return errors
}

// String 返回解析表的字符串表示
func (table *LRParseTable) String() string {
	return fmt.Sprintf("LRParseTable{States: %d, Productions: %d}",
		table.numStates, len(table.productions))
}

// CreateGenericTypeParseTable 创建处理泛型类型歧义的LR解析表
func CreateGenericTypeParseTable() *LRParseTable {
	table := NewLRParseTable()

	// 产生式定义
	// 0: GenericType -> IDENTIFIER '<' TypeList '>'
	// 1: TypeList -> IDENTIFIER
	// 2: TypeList -> TypeList ',' IDENTIFIER

	prod0 := NewLRProduction("GenericType", []string{"IDENTIFIER", "<", "TypeList", ">"})
	prod1 := NewLRProduction("TypeList", []string{"IDENTIFIER"})
	prod2 := NewLRProduction("TypeList", []string{"TypeList", ",", "IDENTIFIER"})

	table.AddProduction(prod0)
	table.AddProduction(prod1)
	table.AddProduction(prod2)

	// Action表和Goto表
	// 状态0: 开始状态
	table.AddAction(0, "IDENTIFIER", NewLRAction(LRActionShift, 1, -1))
	table.AddGoto(0, "GenericType", 2)

	// 状态1: 读取了标识符
	table.AddAction(1, "<", NewLRAction(LRActionShift, 3, -1))
	// 没有<时可以作为普通标识符处理（这里不添加accept，因为这不是完整的泛型类型）

	// 状态2: 归约后的GenericType
	table.AddAction(2, "", NewLRAction(LRActionAccept, -1, -1)) // EOF接受

	// 状态3: 读取了<
	table.AddAction(3, "IDENTIFIER", NewLRAction(LRActionShift, 4, -1))
	table.AddGoto(3, "TypeList", 5)

	// 状态4: 读取了类型名
	table.AddAction(4, ">", NewLRAction(LRActionShift, 6, -1))
	table.AddAction(4, ",", NewLRAction(LRActionShift, 7, -1))

	// 状态5: 读取了TypeList
	table.AddAction(5, ">", NewLRAction(LRActionReduce, -1, 1)) // 归约TypeList -> IDENTIFIER

	// 状态6: 读取了>，泛型类型完成
	table.AddAction(6, "", NewLRAction(LRActionReduce, -1, 0)) // 归约GenericType -> IDENTIFIER '<' TypeList '>'

	// 状态7: 读取了,
	table.AddAction(7, "IDENTIFIER", NewLRAction(LRActionShift, 8, -1))

	// 状态8: 读取了下一个类型名
	table.AddAction(8, ">", NewLRAction(LRActionReduce, -1, 2)) // 归约TypeList -> TypeList ',' IDENTIFIER
	table.AddAction(8, ",", NewLRAction(LRActionShift, 7, -1))
	table.AddGoto(8, "TypeList", 9)

	// 状态9: 归约后的TypeList（多参数）
	table.AddAction(9, ">", NewLRAction(LRActionReduce, -1, 2))

	return table
}

// CreateComparisonExpressionParseTable 创建处理比较表达式歧义的LR解析表
func CreateComparisonExpressionParseTable() *LRParseTable {
	table := NewLRParseTable()

	// 产生式定义
	// 0: Comparison -> Expression '<' Expression
	// 1: Expression -> IDENTIFIER

	prod0 := NewLRProduction("Comparison", []string{"Expression", "<", "Expression"})
	prod1 := NewLRProduction("Expression", []string{"IDENTIFIER"})

	table.AddProduction(prod0)
	table.AddProduction(prod1)

	// Action表
	table.AddAction(0, "IDENTIFIER", NewLRAction(LRActionShift, 1, -1))
	table.AddGoto(0, "Comparison", 2)
	table.AddGoto(0, "Expression", 3)

	// 状态1: 读取了标识符
	table.AddAction(1, "", NewLRAction(LRActionReduce, -1, 1)) // 归约Expression -> IDENTIFIER

	// 状态2: 归约后的Comparison
	table.AddAction(2, "", NewLRAction(LRActionAccept, -1, -1))

	// 状态3: 读取了Expression
	table.AddAction(3, "<", NewLRAction(LRActionShift, 4, -1))

	// 状态4: 读取了<
	table.AddAction(4, "IDENTIFIER", NewLRAction(LRActionShift, 5, -1))
	table.AddGoto(4, "Expression", 6)

	// 状态5: 读取了右侧标识符
	table.AddAction(5, "", NewLRAction(LRActionReduce, -1, 1)) // 归约右侧Expression

	// 状态6: 读取了右侧Expression
	table.AddAction(6, "", NewLRAction(LRActionReduce, -1, 0)) // 归约Comparison

	return table
}

// LRTableBuilder LR解析表构建器
// 用于动态构建LR解析表
type LRTableBuilder struct {
	table *LRParseTable
}

// NewLRTableBuilder 创建新的LR表构建器
func NewLRTableBuilder() *LRTableBuilder {
	return &LRTableBuilder{
		table: NewLRParseTable(),
	}
}

// AddProduction 添加产生式
func (builder *LRTableBuilder) AddProduction(lhs string, rhs []string) int {
	production := NewLRProduction(lhs, rhs)
	return builder.table.AddProduction(production)
}

// AddAction 添加动作
func (builder *LRTableBuilder) AddAction(state int, symbol string, actionType LRActionType, param int) {
	var action LRAction
	switch actionType {
	case LRActionShift:
		action = NewLRAction(LRActionShift, param, -1)
	case LRActionReduce:
		action = NewLRAction(LRActionReduce, -1, param)
	case LRActionAccept:
		action = NewLRAction(LRActionAccept, -1, -1)
	case LRActionError:
		action = NewLRAction(LRActionError, -1, -1)
	}
	builder.table.AddAction(state, symbol, action)
}

// AddGoto 添加Goto
func (builder *LRTableBuilder) AddGoto(state int, nonTerminal string, nextState int) {
	builder.table.AddGoto(state, nonTerminal, nextState)
}

// Build 构建解析表
func (builder *LRTableBuilder) Build() *LRParseTable {
	return builder.table
}
