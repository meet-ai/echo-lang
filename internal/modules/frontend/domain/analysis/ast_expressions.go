package analysis

import "fmt"

// BinaryExpr 二元表达式
type BinaryExpr struct {
	left     ASTNode
	operator Token
	right    ASTNode
	position Position
}

func NewBinaryExpr(left ASTNode, operator Token, right ASTNode, position Position) *BinaryExpr {
	return &BinaryExpr{
		left:     left,
		operator: operator,
		right:    right,
		position: position,
	}
}

func (b *BinaryExpr) Left() ASTNode     { return b.left }
func (b *BinaryExpr) Operator() Token   { return b.operator }
func (b *BinaryExpr) Right() ASTNode    { return b.right }
func (b *BinaryExpr) Position() Position { return b.position }

func (b *BinaryExpr) Accept(visitor ASTVisitor) error {
	return visitor.VisitBinaryExpr(b)
}

func (b *BinaryExpr) String() string {
	return fmt.Sprintf("BinaryExpr{op=%s, position=%s}", b.operator.Value(), b.position.String())
}

// UnaryExpr 一元表达式
type UnaryExpr struct {
	operator Token
	operand  ASTNode
	position Position
}

func NewUnaryExpr(operator Token, operand ASTNode, position Position) *UnaryExpr {
	return &UnaryExpr{
		operator: operator,
		operand:  operand,
		position: position,
	}
}

func (u *UnaryExpr) Operator() Token    { return u.operator }
func (u *UnaryExpr) Operand() ASTNode   { return u.operand }
func (u *UnaryExpr) Position() Position { return u.position }

func (u *UnaryExpr) Accept(visitor ASTVisitor) error {
	return visitor.VisitUnaryExpr(u)
}

func (u *UnaryExpr) String() string {
	return fmt.Sprintf("UnaryExpr{op=%s, position=%s}", u.operator.Value(), u.position.String())
}

// LiteralExpr 字面量表达式
type LiteralExpr struct {
	value    interface{}
	kind     TokenKind
	position Position
}

func NewLiteralExpr(value interface{}, kind TokenKind, position Position) *LiteralExpr {
	return &LiteralExpr{
		value:    value,
		kind:     kind,
		position: position,
	}
}

func (l *LiteralExpr) Value() interface{} { return l.value }
func (l *LiteralExpr) Kind() TokenKind    { return l.kind }
func (l *LiteralExpr) Position() Position { return l.position }

func (l *LiteralExpr) Accept(visitor ASTVisitor) error {
	return visitor.VisitLiteralExpr(l)
}

func (l *LiteralExpr) String() string {
	return fmt.Sprintf("LiteralExpr{kind=%s, value=%v, position=%s}", l.kind.String(), l.value, l.position.String())
}

// IdentifierExpr 标识符表达式
type IdentifierExpr struct {
	name     string
	position Position
}

func NewIdentifierExpr(name string, position Position) *IdentifierExpr {
	return &IdentifierExpr{
		name:     name,
		position: position,
	}
}

func (i *IdentifierExpr) Name() string     { return i.name }
func (i *IdentifierExpr) Position() Position { return i.position }

func (i *IdentifierExpr) Accept(visitor ASTVisitor) error {
	return visitor.VisitIdentifierExpr(i)
}

func (i *IdentifierExpr) String() string {
	return fmt.Sprintf("IdentifierExpr{name=%s, position=%s}", i.name, i.position.String())
}

// FunctionCallExpr 函数调用表达式
type FunctionCallExpr struct {
	function  ASTNode
	arguments []ASTNode
	position  Position
}

func NewFunctionCallExpr(function ASTNode, arguments []ASTNode, position Position) *FunctionCallExpr {
	return &FunctionCallExpr{
		function:  function,
		arguments: arguments,
		position:  position,
	}
}

func (f *FunctionCallExpr) Function() ASTNode   { return f.function }
func (f *FunctionCallExpr) Arguments() []ASTNode { return f.arguments }
func (f *FunctionCallExpr) Position() Position  { return f.position }

func (f *FunctionCallExpr) Accept(visitor ASTVisitor) error {
	return visitor.VisitFunctionCallExpr(f)
}

func (f *FunctionCallExpr) String() string {
	return fmt.Sprintf("FunctionCallExpr{args=%d, position=%s}", len(f.arguments), f.position.String())
}

// StructLiteralExpr 结构体字面量表达式
type StructLiteralExpr struct {
	typeName string
	fields   []StructField
	position Position
}

type StructField struct {
	name  string
	value ASTNode
}

func NewStructField(name string, value ASTNode) StructField {
	return StructField{name: name, value: value}
}

func NewStructLiteralExpr(typeName string, fields []StructField, position Position) *StructLiteralExpr {
	return &StructLiteralExpr{
		typeName: typeName,
		fields:   fields,
		position: position,
	}
}

func (s *StructLiteralExpr) TypeName() string        { return s.typeName }
func (s *StructLiteralExpr) Fields() []StructField   { return s.fields }
func (s *StructLiteralExpr) Position() Position      { return s.position }

func (s *StructLiteralExpr) Accept(visitor ASTVisitor) error {
	return visitor.VisitStructLiteralExpr(s)
}

func (s *StructLiteralExpr) String() string {
	return fmt.Sprintf("StructLiteralExpr{type=%s, fields=%d, position=%s}", s.typeName, len(s.fields), s.position.String())
}

// ArrayLiteralExpr 数组字面量表达式
type ArrayLiteralExpr struct {
	elements []ASTNode
	position Position
}

func NewArrayLiteralExpr(elements []ASTNode, position Position) *ArrayLiteralExpr {
	return &ArrayLiteralExpr{
		elements: elements,
		position: position,
	}
}

func (a *ArrayLiteralExpr) Elements() []ASTNode { return a.elements }
func (a *ArrayLiteralExpr) Position() Position   { return a.position }

func (a *ArrayLiteralExpr) Accept(visitor ASTVisitor) error {
	return visitor.VisitArrayLiteralExpr(a)
}

func (a *ArrayLiteralExpr) String() string {
	return fmt.Sprintf("ArrayLiteralExpr{elements=%d, position=%s}", len(a.elements), a.position.String())
}

// MatchExpr 模式匹配表达式
type MatchExpr struct {
	expr      ASTNode
	cases     []MatchCase
	position  Position
}

type MatchCase struct {
	pattern ASTNode
	body    ASTNode
}

func NewMatchCase(pattern ASTNode, body ASTNode) MatchCase {
	return MatchCase{pattern: pattern, body: body}
}

func NewMatchExpr(expr ASTNode, cases []MatchCase, position Position) *MatchExpr {
	return &MatchExpr{
		expr:     expr,
		cases:    cases,
		position: position,
	}
}

func (m *MatchExpr) Expr() ASTNode        { return m.expr }
func (m *MatchExpr) Cases() []MatchCase   { return m.cases }
func (m *MatchExpr) Position() Position   { return m.position }

func (m *MatchExpr) Accept(visitor ASTVisitor) error {
	return visitor.VisitMatchExpr(m)
}

func (m *MatchExpr) String() string {
	return fmt.Sprintf("MatchExpr{cases=%d, position=%s}", len(m.cases), m.position.String())
}

// AssignExpr 赋值表达式
type AssignExpr struct {
	target   ASTNode
	value    ASTNode
	position Position
}

func NewAssignExpr(target ASTNode, value ASTNode, position Position) *AssignExpr {
	return &AssignExpr{
		target:   target,
		value:    value,
		position: position,
	}
}

func (a *AssignExpr) Target() ASTNode   { return a.target }
func (a *AssignExpr) Value() ASTNode    { return a.value }
func (a *AssignExpr) Position() Position { return a.position }

func (a *AssignExpr) Accept(visitor ASTVisitor) error {
	return visitor.VisitAssignExpr(a)
}

func (a *AssignExpr) String() string {
	return fmt.Sprintf("AssignExpr{position=%s}", a.position.String())
}
