// Package value_objects 定义语法分析上下文的值对象
package value_objects

import "fmt"

// Precedence 运算符优先级类型
// 值越大，优先级越高
type Precedence int

// 运算符优先级常量定义（按照C/Java标准）
// 参考：https://en.wikipedia.org/wiki/Operator_precedence
const (
	// 最低优先级：赋值运算符
	PrecedenceAssignment Precedence = iota

	// 条件运算符 (?:)
	PrecedenceConditional

	// 逻辑运算符
	PrecedenceLogicalOr
	PrecedenceLogicalAnd

	// 位运算符
	PrecedenceBitwiseOr
	PrecedenceBitwiseXor
	PrecedenceBitwiseAnd

	// 相等比较
	PrecedenceEquality

	// 关系比较
	PrecedenceRelational

	// 位移运算符
	PrecedenceShift

	// 加减运算
	PrecedenceAdditive

	// 乘除模
	PrecedenceMultiplicative

	// 前缀运算符
	PrecedencePrefix

	// 后缀运算符和最高优先级
	PrecedencePostfix
	PrecedencePrimary // 字面量、标识符、括号表达式等
)

// String 返回优先级名称
func (p Precedence) String() string {
	switch p {
	case PrecedenceAssignment:
		return "assignment"
	case PrecedenceConditional:
		return "conditional"
	case PrecedenceLogicalOr:
		return "logical_or"
	case PrecedenceLogicalAnd:
		return "logical_and"
	case PrecedenceBitwiseOr:
		return "bitwise_or"
	case PrecedenceBitwiseXor:
		return "bitwise_xor"
	case PrecedenceBitwiseAnd:
		return "bitwise_and"
	case PrecedenceEquality:
		return "equality"
	case PrecedenceRelational:
		return "relational"
	case PrecedenceShift:
		return "shift"
	case PrecedenceAdditive:
		return "additive"
	case PrecedenceMultiplicative:
		return "multiplicative"
	case PrecedencePrefix:
		return "prefix"
	case PrecedencePostfix:
		return "postfix"
	case PrecedencePrimary:
		return "primary"
	default:
		return "unknown"
	}
}

// Associativity 运算符结合性
type Associativity int

const (
	// 左结合：a + b + c 被解析为 (a + b) + c
	AssociativityLeft Associativity = iota
	// 右结合：a = b = c 被解析为 a = (b = c)
	AssociativityRight
	// 非结合：不能连续使用同一运算符
	AssociativityNone
)

// String 返回结合性名称
func (a Associativity) String() string {
	switch a {
	case AssociativityLeft:
		return "left"
	case AssociativityRight:
		return "right"
	case AssociativityNone:
		return "none"
	default:
		return "unknown"
	}
}

// OperatorDefinition 运算符定义
type OperatorDefinition struct {
	// 运算符词素（如 "+"、"=="、"&&"等）
	Lexeme string

	// 优先级
	Precedence Precedence

	// 结合性
	Associativity Associativity

	// 是否是前缀运算符
	IsPrefix bool

	// 是否是中缀运算符
	IsInfix bool

	// 是否是后缀运算符
	IsPostfix bool
}

// NewOperatorDefinition 创建新的运算符定义
func NewOperatorDefinition(lexeme string, precedence Precedence, associativity Associativity, isPrefix, isInfix, isPostfix bool) *OperatorDefinition {
	return &OperatorDefinition{
		Lexeme:       lexeme,
		Precedence:   precedence,
		Associativity: associativity,
		IsPrefix:     isPrefix,
		IsInfix:      isInfix,
		IsPostfix:    isPostfix,
	}
}

// String 返回运算符定义的字符串表示
func (od *OperatorDefinition) String() string {
	return fmt.Sprintf("OperatorDefinition{Lexeme: %s, Precedence: %s, Associativity: %s, Prefix: %t, Infix: %t, Postfix: %t}",
		od.Lexeme, od.Precedence.String(), od.Associativity.String(), od.IsPrefix, od.IsInfix, od.IsPostfix)
}

// OperatorRegistry 运算符注册表
// 管理所有支持的运算符定义
type OperatorRegistry struct {
	// 运算符映射表：词素 -> 定义
	operators map[string]*OperatorDefinition

	// 按优先级分组的运算符
	operatorsByPrecedence map[Precedence][]*OperatorDefinition
}

// NewOperatorRegistry 创建新的运算符注册表
func NewOperatorRegistry() *OperatorRegistry {
	registry := &OperatorRegistry{
		operators:             make(map[string]*OperatorDefinition),
		operatorsByPrecedence: make(map[Precedence][]*OperatorDefinition),
	}

	// 注册标准运算符
	registry.registerStandardOperators()

	return registry
}

// Register 注册运算符
func (or *OperatorRegistry) Register(def *OperatorDefinition) {
	// 注册到词素映射
	or.operators[def.Lexeme] = def

	// 注册到优先级分组
	if or.operatorsByPrecedence[def.Precedence] == nil {
		or.operatorsByPrecedence[def.Precedence] = make([]*OperatorDefinition, 0)
	}
	or.operatorsByPrecedence[def.Precedence] = append(or.operatorsByPrecedence[def.Precedence], def)
}

// GetOperator 获取运算符定义
func (or *OperatorRegistry) GetOperator(lexeme string) (*OperatorDefinition, bool) {
	def, exists := or.operators[lexeme]
	return def, exists
}

// GetOperatorsByPrecedence 获取指定优先级的运算符
func (or *OperatorRegistry) GetOperatorsByPrecedence(precedence Precedence) []*OperatorDefinition {
	return or.operatorsByPrecedence[precedence]
}

// GetAllOperators 获取所有运算符
func (or *OperatorRegistry) GetAllOperators() map[string]*OperatorDefinition {
	// 返回副本以保护内部状态
	result := make(map[string]*OperatorDefinition)
	for k, v := range or.operators {
		result[k] = v
	}
	return result
}

// IsOperator 检查是否为已注册的运算符
func (or *OperatorRegistry) IsOperator(lexeme string) bool {
	_, exists := or.operators[lexeme]
	return exists
}

// IsPrefixOperator 检查是否为前缀运算符
func (or *OperatorRegistry) IsPrefixOperator(lexeme string) bool {
	if def, exists := or.operators[lexeme]; exists {
		return def.IsPrefix
	}
	return false
}

// IsInfixOperator 检查是否为中缀运算符
func (or *OperatorRegistry) IsInfixOperator(lexeme string) bool {
	if def, exists := or.operators[lexeme]; exists {
		return def.IsInfix
	}
	return false
}

// IsPostfixOperator 检查是否为后缀运算符
func (or *OperatorRegistry) IsPostfixOperator(lexeme string) bool {
	if def, exists := or.operators[lexeme]; exists {
		return def.IsPostfix
	}
	return false
}

// GetPrecedence 获取运算符优先级
func (or *OperatorRegistry) GetPrecedence(lexeme string) (Precedence, bool) {
	if def, exists := or.operators[lexeme]; exists {
		return def.Precedence, true
	}
	return PrecedencePrimary, false
}

// GetAssociativity 获取运算符结合性
func (or *OperatorRegistry) GetAssociativity(lexeme string) (Associativity, bool) {
	if def, exists := or.operators[lexeme]; exists {
		return def.Associativity, true
	}
	return AssociativityLeft, false
}

// registerStandardOperators 注册标准运算符
func (or *OperatorRegistry) registerStandardOperators() {
	// 赋值运算符 (右结合)
	assignmentOps := []string{"=", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>="}
	for _, op := range assignmentOps {
		or.Register(NewOperatorDefinition(op, PrecedenceAssignment, AssociativityRight, false, true, false))
	}

	// 条件运算符 ?:
	// 注意：在Pratt解析器中，条件运算符通常需要特殊处理
	// or.Register(NewOperatorDefinition("?", PrecedenceConditional, AssociativityRight, false, true, false))
	// or.Register(NewOperatorDefinition(":", PrecedenceConditional, AssociativityRight, false, true, false))

	// 逻辑运算符 (左结合)
	or.Register(NewOperatorDefinition("||", PrecedenceLogicalOr, AssociativityLeft, false, true, false))
	or.Register(NewOperatorDefinition("&&", PrecedenceLogicalAnd, AssociativityLeft, false, true, false))

	// 位运算符 (左结合)
	or.Register(NewOperatorDefinition("|", PrecedenceBitwiseOr, AssociativityLeft, false, true, false))
	or.Register(NewOperatorDefinition("^", PrecedenceBitwiseXor, AssociativityLeft, false, true, false))
	or.Register(NewOperatorDefinition("&", PrecedenceBitwiseAnd, AssociativityLeft, false, true, false))

	// 相等比较 (左结合)
	or.Register(NewOperatorDefinition("==", PrecedenceEquality, AssociativityLeft, false, true, false))
	or.Register(NewOperatorDefinition("!=", PrecedenceEquality, AssociativityLeft, false, true, false))

	// 关系比较 (左结合)
	relationalOps := []string{"<", ">", "<=", ">="}
	for _, op := range relationalOps {
		or.Register(NewOperatorDefinition(op, PrecedenceRelational, AssociativityLeft, false, true, false))
	}

	// 位移运算符 (左结合)
	shiftOps := []string{"<<", ">>"}
	for _, op := range shiftOps {
		or.Register(NewOperatorDefinition(op, PrecedenceShift, AssociativityLeft, false, true, false))
	}

	// 加减运算 (左结合)
	additiveOps := []string{"+", "-"}
	for _, op := range additiveOps {
		or.Register(NewOperatorDefinition(op, PrecedenceAdditive, AssociativityLeft, true, true, false))
	}

	// 乘除模运算 (左结合)
	multiplicativeOps := []string{"*", "/", "%"}
	for _, op := range multiplicativeOps {
		or.Register(NewOperatorDefinition(op, PrecedenceMultiplicative, AssociativityLeft, false, true, false))
	}

	// 前缀运算符 (右结合，因为前缀运算符是从右到左结合的)
	prefixOps := []string{"!", "~", "-", "+"}
	for _, op := range prefixOps {
		or.Register(NewOperatorDefinition(op, PrecedencePrefix, AssociativityRight, true, false, false))
	}

	// 后缀运算符
	// 注意：数组索引[]、函数调用()、成员访问.通常作为特殊语法处理，不作为普通运算符
	// 这里可以根据需要添加其他后缀运算符
}

// PrecedenceTable 优先级表
// 用于快速查询运算符优先级
type PrecedenceTable struct {
	registry *OperatorRegistry
}

// NewPrecedenceTable 创建新的优先级表
func NewPrecedenceTable(registry *OperatorRegistry) *PrecedenceTable {
	return &PrecedenceTable{
		registry: registry,
	}
}

// GetPrecedence 获取运算符优先级
func (pt *PrecedenceTable) GetPrecedence(lexeme string) Precedence {
	if prec, exists := pt.registry.GetPrecedence(lexeme); exists {
		return prec
	}
	return PrecedencePrimary
}

// ComparePrecedence 比较两个运算符的优先级
// 返回值：
//   -1: left优先级低于right
//    0: left优先级等于right
//    1: left优先级高于right
func (pt *PrecedenceTable) ComparePrecedence(leftLexeme, rightLexeme string) int {
	leftPrec := pt.GetPrecedence(leftLexeme)
	rightPrec := pt.GetPrecedence(rightLexeme)

	if leftPrec < rightPrec {
		return -1
	} else if leftPrec > rightPrec {
		return 1
	}
	return 0
}

// ShouldContinueParsing 检查是否应该继续解析（用于Pratt算法）
// 如果当前运算符的优先级高于minPrecedence，则应该继续解析
func (pt *PrecedenceTable) ShouldContinueParsing(operatorLexeme string, minPrecedence Precedence) bool {
	opPrec := pt.GetPrecedence(operatorLexeme)
	return opPrec > minPrecedence
}

// GetNextMinPrecedence 计算下一个最小优先级（用于Pratt算法）
// 根据结合性决定如何调整优先级
func (pt *PrecedenceTable) GetNextMinPrecedence(operatorLexeme string, currentMinPrecedence Precedence) Precedence {
	opPrec := pt.GetPrecedence(operatorLexeme)

	if associativity, exists := pt.registry.GetAssociativity(operatorLexeme); exists {
		if associativity == AssociativityLeft {
			// 左结合：下一个优先级为当前优先级+1
			return opPrec + 1
		} else if associativity == AssociativityRight {
			// 右结合：下一个优先级为当前优先级
			return opPrec
		}
	}

	// 默认左结合
	return opPrec + 1
}
