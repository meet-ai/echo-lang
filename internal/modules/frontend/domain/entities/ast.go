package entities

import (
	"fmt"
	"strings"
)

// ASTNode 表示抽象语法树节点
type ASTNode interface {
	String() string
	Accept(visitor ASTVisitor) interface{}
}

// ASTVisitor 访问者模式接口
type ASTVisitor interface {
	VisitPrintStmt(stmt *PrintStmt) interface{}
	VisitExprStmt(stmt *ExprStmt) interface{}
	VisitFuncCall(call *FuncCall) interface{}
	VisitVarDecl(stmt *VarDecl) interface{}
	VisitAssignStmt(stmt *AssignStmt) interface{}
	VisitFuncDef(stmt *FuncDef) interface{}
	VisitReturnStmt(stmt *ReturnStmt) interface{}
	VisitBlockStmt(stmt *BlockStmt) interface{}
	VisitIfStmt(stmt *IfStmt) interface{}
	VisitForStmt(stmt *ForStmt) interface{}
	VisitWhileStmt(stmt *WhileStmt) interface{}
	VisitAgentCreateStmt(stmt *AgentCreateStmt) interface{}
	VisitAgentSendStmt(stmt *AgentSendStmt) interface{}
	VisitAgentReceiveStmt(stmt *AgentReceiveStmt) interface{}
	VisitBreakStmt(stmt *BreakStmt) interface{}
	VisitContinueStmt(stmt *ContinueStmt) interface{}
	VisitDeleteStmt(stmt *DeleteStmt) interface{}
	VisitStructDef(stmt *StructDef) interface{}
	VisitMethodDef(stmt *MethodDef) interface{}
	VisitEnumDef(stmt *EnumDef) interface{}
	VisitTraitDef(stmt *TraitDef) interface{}
	VisitImplDef(stmt *ImplDef) interface{}
	VisitAssociatedTypeDef(stmt *AssociatedTypeDef) interface{}
	VisitDynamicTraitRef(expr *DynamicTraitRef) interface{}
	VisitMatchExpr(expr *MatchExpr) interface{}
	VisitMatchStmt(stmt *MatchStmt) interface{}
	VisitArrayLiteral(expr *ArrayLiteral) interface{}
	VisitStructAccess(expr *StructAccess) interface{}
	VisitGenericType(genericType *GenericType) interface{}
	VisitResultExpr(expr *ResultExpr) interface{}
	VisitOptionExpr(expr *OptionExpr) interface{}
	VisitOkLiteral(literal *OkLiteral) interface{}
	VisitErrLiteral(literal *ErrLiteral) interface{}
	VisitErrorPropagation(expr *ErrorPropagation) interface{}
	VisitSomeLiteral(literal *SomeLiteral) interface{}
	VisitNoneLiteral(literal *NoneLiteral) interface{}
	VisitBoolLiteral(literal *BoolLiteral) interface{}
	VisitOkPattern(pattern *OkPattern) interface{}
	VisitErrPattern(pattern *ErrPattern) interface{}
	VisitSomePattern(pattern *SomePattern) interface{}
	VisitNonePattern(pattern *NonePattern) interface{}
	VisitWildcardPattern(pattern *WildcardPattern) interface{}
	VisitIdentifierPattern(pattern *IdentifierPattern) interface{}
	VisitLiteralPattern(pattern *LiteralPattern) interface{}
	VisitStructPattern(pattern *StructPattern) interface{}
	VisitArrayPattern(pattern *ArrayPattern) interface{}
	VisitTuplePattern(pattern *TuplePattern) interface{}
	VisitAsyncFuncDef(funcDef *AsyncFuncDef) interface{}
	VisitAwaitExpr(expr *AwaitExpr) interface{}
	VisitChanType(chanType *ChanType) interface{}
	VisitFutureType(futureType *FutureType) interface{}
	VisitChanLiteral(literal *ChanLiteral) interface{}
	VisitSendExpr(expr *SendExpr) interface{}
	VisitReceiveExpr(expr *ReceiveExpr) interface{}
	VisitSpawnExpr(expr *SpawnExpr) interface{}
	VisitSelectStmt(stmt *SelectStmt) interface{}
	VisitIndexExpr(expr *IndexExpr) interface{}
	VisitLenExpr(expr *LenExpr) interface{}
	VisitSizeOfExpr(expr *SizeOfExpr) interface{}
	VisitTypeCastExpr(expr *TypeCastExpr) interface{}
	VisitFunctionPointerExpr(expr *FunctionPointerExpr) interface{}
	VisitAddressOfExpr(expr *AddressOfExpr) interface{}
	VisitDereferenceExpr(expr *DereferenceExpr) interface{}
	VisitSliceExpr(expr *SliceExpr) interface{}
	VisitArrayMethodCallExpr(expr *ArrayMethodCallExpr) interface{}
	VisitMethodCallExpr(expr *MethodCallExpr) interface{}
	VisitStructLiteral(expr *StructLiteral) interface{}
	VisitNamespaceAccessExpr(expr *NamespaceAccessExpr) interface{}
	VisitProgram(program *Program) interface{}
}

// PrintStmt 打印语句节点
type PrintStmt struct {
	Value Expr
}

// ExprStmt 表达式语句节点（用于独立的表达式，如函数调用、赋值等）
type ExprStmt struct {
	Expression Expr
}

func (p *PrintStmt) String() string {
	return fmt.Sprintf("PrintStmt{Value: %s}", p.Value.String())
}

func (p *PrintStmt) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitPrintStmt(p)
}

func (e *ExprStmt) String() string {
	return fmt.Sprintf("ExprStmt{Expression: %s}", e.Expression)
}

func (e *ExprStmt) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitExprStmt(e)
}

// VarDecl 变量声明节点
type VarDecl struct {
	Name     string
	Type     string
	Value    Expr
	Inferred bool // 是否需要类型推断
}

func (v *VarDecl) String() string {
	inferred := ""
	if v.Inferred {
		inferred = " (inferred)"
	}
	return fmt.Sprintf("VarDecl{Name: %s, Type: %s, Value: %s%s}", v.Name, v.Type, v.Value, inferred)
}

func (v *VarDecl) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitVarDecl(v)
}

// AssignStmt 赋值语句节点
type AssignStmt struct {
	Target Expr // 赋值目标（可以是 Identifier、StructAccess、IndexExpr 等）
	Value  Expr // 赋值表达式
}

func (a *AssignStmt) String() string {
	// 为了向后兼容，如果 Target 是 Identifier，显示 Name
	if ident, ok := a.Target.(*Identifier); ok {
		return fmt.Sprintf("AssignStmt{Name: %s, Value: %s}", ident.Name, a.Value)
	}
	return fmt.Sprintf("AssignStmt{Target: %s, Value: %s}", a.Target, a.Value)
}

// Name 返回赋值目标的名称（如果是 Identifier）
// 为了向后兼容，保留此方法
func (a *AssignStmt) Name() string {
	if ident, ok := a.Target.(*Identifier); ok {
		return ident.Name
	}
	return ""
}

func (a *AssignStmt) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitAssignStmt(a)
}

// FuncDef 函数定义节点
type FuncDef struct {
	Name       string
	TypeParams []GenericParam // 类型参数列表，支持泛型
	Params     []Param
	ReturnType string
	Body       []ASTNode
	IsAsync    bool // 是否为异步函数
}

func (f *FuncDef) String() string {
	params := make([]string, len(f.Params))
	for i, p := range f.Params {
		params[i] = p.String()
	}
	typeParams := make([]string, len(f.TypeParams))
	for i, tp := range f.TypeParams {
		typeParams[i] = tp.String()
	}
	return fmt.Sprintf("FuncDef{Name: %s, TypeParams: [%s], Params: [%s], ReturnType: %s, Body: %v}",
		f.Name, strings.Join(typeParams, ", "), fmt.Sprintf("%v", params), f.ReturnType, f.Body)
}

func (f *FuncDef) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitFuncDef(f)
}

// Param 函数参数
type Param struct {
	Name string
	Type string
}

func (p Param) String() string {
	return fmt.Sprintf("%s: %s", p.Name, p.Type)
}

// ReturnStmt return 语句节点
type ReturnStmt struct {
	Value Expr
}

func (r *ReturnStmt) String() string {
	return fmt.Sprintf("ReturnStmt{Value: %s}", r.Value)
}

func (r *ReturnStmt) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitReturnStmt(r)
}

// BlockStmt 块语句节点（将多条语句作为一条语句使用，如独立代码块 { stmt1; stmt2; }）
type BlockStmt struct {
	Statements []ASTNode
}

func (b *BlockStmt) String() string {
	parts := make([]string, len(b.Statements))
	for i, s := range b.Statements {
		parts[i] = s.String()
	}
	return fmt.Sprintf("BlockStmt{Statements: [%s]}", strings.Join(parts, ", "))
}

func (b *BlockStmt) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitBlockStmt(b)
}

// IfStmt if-else 语句节点
type IfStmt struct {
	Condition Expr
	ThenBody  []ASTNode
	ElseBody  []ASTNode
}

func (i *IfStmt) String() string {
	return fmt.Sprintf("IfStmt{Condition: %s}", i.Condition)
}

func (i *IfStmt) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitIfStmt(i)
}

// ForStmt for 循环节点
type ForStmt struct {
	Init      ASTNode   // 初始化语句（C风格for循环）
	Condition Expr      // 循环条件（C风格for循环）
	Increment ASTNode   // 递增语句（C风格for循环）
	Body      []ASTNode // 循环体
	
	// 范围循环支持（for i in start..end）
	LoopVar   string    // 循环变量名（如 "i"）
	RangeStart Expr     // 范围起始值（如 0）
	RangeEnd   Expr     // 范围结束值（如 n）
	IsRangeLoop bool    // 是否为范围循环
	
	// 迭代循环支持（for item in collection）
	IterVar   string    // 迭代变量名（如 "item", "ch"）
	IterKeyVar string   // 键变量名（用于map迭代，如 "key"）
	IterValueVar string // 值变量名（用于map迭代，如 "value"）
	IterCollection Expr // 被迭代的集合（如数组、字符串、map）
	IsIterLoop bool    // 是否为迭代循环
	IsMapIter  bool    // 是否为map迭代（for (key, value) in obj）
}

func (f *ForStmt) String() string {
	return fmt.Sprintf("ForStmt{Condition: %s}", f.Condition)
}

func (f *ForStmt) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitForStmt(f)
}

// WhileStmt while 循环节点
type WhileStmt struct {
	Condition Expr      // 循环条件
	Body      []ASTNode // 循环体
}

func (w *WhileStmt) String() string {
	return fmt.Sprintf("WhileStmt{Condition: %s}", w.Condition)
}

func (w *WhileStmt) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitWhileStmt(w)
}

// AgentCreateStmt 智能体创建语句
type AgentCreateStmt struct {
	Name string // 可选的智能体变量名
}

func (a *AgentCreateStmt) String() string {
	return fmt.Sprintf("AgentCreateStmt{Name: %s}", a.Name)
}

func (a *AgentCreateStmt) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitAgentCreateStmt(a)
}

// AgentSendStmt 智能体发送消息语句
type AgentSendStmt struct {
	FromAgent Expr // 发送者智能体ID表达式
	ToAgent   Expr // 接收者智能体ID表达式
	Message   Expr // 消息表达式
}

func (a *AgentSendStmt) String() string {
	return fmt.Sprintf("AgentSendStmt{From: %s, To: %s, Message: %s}", a.FromAgent, a.ToAgent, a.Message)
}

func (a *AgentSendStmt) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitAgentSendStmt(a)
}

// AgentReceiveStmt 智能体接收消息语句
type AgentReceiveStmt struct {
	AgentID Expr // 智能体ID表达式
}

func (a *AgentReceiveStmt) String() string {
	return fmt.Sprintf("AgentReceiveStmt{AgentID: %s}", a.AgentID)
}

func (a *AgentReceiveStmt) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitAgentReceiveStmt(a)
}

// BreakStmt break语句
type BreakStmt struct{}

func (b *BreakStmt) String() string {
	return "BreakStmt{}"
}

func (b *BreakStmt) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitBreakStmt(b)
}

// ContinueStmt continue语句
type ContinueStmt struct{}

func (c *ContinueStmt) String() string {
	return "ContinueStmt{}"
}

func (c *ContinueStmt) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitContinueStmt(c)
}

// DeleteStmt delete语句（用于删除Map中的键值对）
// 语法：delete(map, key)
type DeleteStmt struct {
	Map Expr // map表达式
	Key Expr // key表达式
}

func (d *DeleteStmt) String() string {
	return fmt.Sprintf("DeleteStmt{Map: %s, Key: %s}", d.Map.String(), d.Key.String())
}

func (d *DeleteStmt) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitDeleteStmt(d)
}

// StructDef 结构体定义节点
type StructDef struct {
	Name                string               // 结构体名
	TypeParams          []GenericParam       // 类型参数列表，支持泛型
	ImplTraits          []ImplAnnotation     // 实现的traits（通过注解）
	AssociatedTypeImpls []AssociatedTypeImpl // 关联类型实现
	Fields              []StructField        // 字段列表
}

func (s *StructDef) String() string {
	traits := make([]string, len(s.ImplTraits))
	for i, t := range s.ImplTraits {
		traits[i] = t.String()
	}
	associatedTypes := make([]string, len(s.AssociatedTypeImpls))
	for i, at := range s.AssociatedTypeImpls {
		associatedTypes[i] = at.String()
	}
	fields := make([]string, len(s.Fields))
	for i, f := range s.Fields {
		fields[i] = f.String()
	}
	typeParams := make([]string, len(s.TypeParams))
	for i, tp := range s.TypeParams {
		typeParams[i] = tp.String()
	}
	return fmt.Sprintf("StructDef{Name: %s, TypeParams: [%s], ImplTraits: [%s], AssociatedTypes: [%s], Fields: [%s]}",
		s.Name, strings.Join(typeParams, ", "), strings.Join(traits, ", "), strings.Join(associatedTypes, ", "), strings.Join(fields, ", "))
}

func (s *StructDef) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitStructDef(s)
}

// StructField 结构体字段
type StructField struct {
	Name string // 字段名
	Type string // 字段类型
}

func (f StructField) String() string {
	return fmt.Sprintf("%s: %s", f.Name, f.Type)
}

// StructAccess 结构体字段访问 (如 point.x)
type StructAccess struct {
	Object Expr   // 对象表达式
	Field  string // 字段名
}

func (s *StructAccess) String() string {
	return fmt.Sprintf("StructAccess{Object: %s, Field: %s}", s.Object, s.Field)
}

func (s *StructAccess) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitStructAccess(s)
}

func (s *StructAccess) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitStructAccess(s)
}

// StructLiteral 结构体字面量
type StructLiteral struct {
	Type   string          // 结构体类型名
	Fields map[string]Expr // 字段名 -> 字段值
}

func (s *StructLiteral) String() string {
	return fmt.Sprintf("%s{...}", s.Type)
}

func (s *StructLiteral) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitStructLiteral(s)
}

func (s *StructLiteral) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitStructLiteral(s)
}

// NamespaceAccessExpr 命名空间访问表达式
// 例如：net::bind
type NamespaceAccessExpr struct {
	Namespace string // 命名空间名（如 "net"）
	Member    string // 成员名（如 "bind"）
}

func (n *NamespaceAccessExpr) String() string {
	return fmt.Sprintf("NamespaceAccessExpr{Namespace: %s, Member: %s}", n.Namespace, n.Member)
}

func (n *NamespaceAccessExpr) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitNamespaceAccessExpr(n)
}

func (n *NamespaceAccessExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitNamespaceAccessExpr(n)
}

// MatchExpr 模式匹配表达式
type MatchExpr struct {
	Value Expr        // 要匹配的值
	Cases []MatchCase // 匹配分支
}

func (m *MatchExpr) String() string {
	cases := make([]string, len(m.Cases))
	for i, c := range m.Cases {
		cases[i] = c.String()
	}
	return fmt.Sprintf("MatchExpr{Value: %s, Cases: [%s]}", m.Value, strings.Join(cases, ", "))
}

func (m *MatchExpr) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitMatchExpr(m)
}

func (m *MatchExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitMatchExpr(m)
}

// MatchStmt 模式匹配语句（用于语句上下文）
type MatchStmt struct {
	Value Expr        // 要匹配的值
	Cases []MatchCase // 匹配分支
}

func (m *MatchStmt) String() string {
	cases := make([]string, len(m.Cases))
	for i, c := range m.Cases {
		cases[i] = c.String()
	}
	return fmt.Sprintf("MatchStmt{Value: %s, Cases: [%s]}", m.Value, strings.Join(cases, ", "))
}

func (m *MatchStmt) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitMatchStmt(m)
}

// MatchCase 匹配分支
type MatchCase struct {
	Pattern Expr      // 匹配模式
	Guard   Expr      // 守卫条件 (可选, 如 if x > 0)
	Body    []ASTNode // 分支体
}

// AsyncFuncDef 异步函数定义
type AsyncFuncDef struct {
	Name       string         // 函数名
	TypeParams []GenericParam // 类型参数
	Params     []Param        // 参数列表
	ReturnType string         // 返回类型
	Body       []ASTNode      // 函数体
}

func (a *AsyncFuncDef) String() string {
	params := make([]string, len(a.Params))
	for i, p := range a.Params {
		params[i] = p.String()
	}
	return fmt.Sprintf("AsyncFuncDef{Name: %s, Params: [%s], ReturnType: %s}",
		a.Name, strings.Join(params, ", "), a.ReturnType)
}

func (a *AsyncFuncDef) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitAsyncFuncDef(a)
}

// AwaitExpr await表达式
type AwaitExpr struct {
	Expression Expr // 要等待的异步表达式
}

func NewAwaitExpr(expression Expr) *AwaitExpr {
	return &AwaitExpr{Expression: expression}
}

func (a *AwaitExpr) String() string {
	return fmt.Sprintf("AwaitExpr{Expression: %s}", a.Expression)
}

func (a *AwaitExpr) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitAwaitExpr(a)
}

func (a *AwaitExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitAwaitExpr(a)
}

// ChanType 通道类型
type ChanType struct {
	ElementType string // 通道元素类型
}

func (c *ChanType) AcceptExpr(visitor ExprVisitor) interface{} {
	// TODO: 需要在ExprVisitor中添加VisitChanType方法
	return visitor.VisitIdentifier((*Identifier)(nil)) // 临时实现
}

func NewChanType(elementType string) *ChanType {
	return &ChanType{ElementType: elementType}
}

func (c *ChanType) String() string {
	return fmt.Sprintf("Chan[%s]", c.ElementType)
}

func (c *ChanType) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitChanType(c)
}

// FutureType Future类型
type FutureType struct {
	ElementType string // Future结果类型
}

func NewFutureType(elementType string) *FutureType {
	return &FutureType{ElementType: elementType}
}

func (f *FutureType) String() string {
	return fmt.Sprintf("Future[%s]", f.ElementType)
}

func (f *FutureType) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitFutureType(f)
}

// ChanLiteral 通道字面量
type ChanLiteral struct {
	Type string // 通道类型，如 "int"
}

func NewChanLiteral(typeStr string) *ChanLiteral {
	return &ChanLiteral{Type: typeStr}
}

func (c *ChanLiteral) String() string {
	return fmt.Sprintf("ChanLiteral{Type: %s}", c.Type)
}

func (c *ChanLiteral) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitChanLiteral(c)
}

func (c *ChanLiteral) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitChanLiteral(c)
}

// SendExpr 发送表达式 (channel <- value)
type SendExpr struct {
	Channel Expr // 通道表达式
	Value   Expr // 要发送的值
}

func NewSendExpr(channel, value Expr) *SendExpr {
	return &SendExpr{Channel: channel, Value: value}
}

func (s *SendExpr) String() string {
	return fmt.Sprintf("SendExpr{Channel: %s, Value: %s}", s.Channel, s.Value)
}

func (s *SendExpr) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitSendExpr(s)
}

func (s *SendExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitSendExpr(s)
}

// ReceiveExpr 接收表达式 (<-channel)
type ReceiveExpr struct {
	Channel Expr // 通道表达式
}

func NewReceiveExpr(channel Expr) *ReceiveExpr {
	return &ReceiveExpr{Channel: channel}
}

func (r *ReceiveExpr) String() string {
	return fmt.Sprintf("ReceiveExpr{Channel: %s}", r.Channel)
}

func (r *ReceiveExpr) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitReceiveExpr(r)
}

func (r *ReceiveExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitReceiveExpr(r)
}

// SpawnExpr spawn表达式
type SpawnExpr struct {
	Function Expr   // 要启动的函数
	Args     []Expr // 函数参数
}

func NewSpawnExpr(function Expr, args []Expr) *SpawnExpr {
	return &SpawnExpr{Function: function, Args: args}
}

func (s *SpawnExpr) String() string {
	args := make([]string, len(s.Args))
	for i, arg := range s.Args {
		args[i] = arg.String()
	}
	return fmt.Sprintf("SpawnExpr{Function: %s, Args: [%s]}", s.Function, strings.Join(args, ", "))
}

func (s *SpawnExpr) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitSpawnExpr(s)
}

func (s *SpawnExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitSpawnExpr(s)
}

// SelectCase select case分支
type SelectCase struct {
	Chan   Expr      // 通道表达式 (send/recv)
	Value  Expr      // 发送的值 (仅send case)
	Body   []ASTNode // case分支的语句块
	IsSend bool      // 是否为发送操作
}

// SelectStmt select语句
type SelectStmt struct {
	Cases       []SelectCase // case分支列表
	DefaultBody []ASTNode    // default分支 (可选)
}

func NewSelectStmt() *SelectStmt {
	return &SelectStmt{
		Cases: make([]SelectCase, 0),
	}
}

func (s *SelectStmt) String() string {
	return fmt.Sprintf("SelectStmt{Cases: %d}", len(s.Cases))
}

func (s *SelectStmt) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitSelectStmt(s)
}

// OkPattern Ok模式 (如 Ok(value))
type OkPattern struct {
	Variable string // 绑定的变量名，如 "value"
}

// ErrPattern Err模式 (如 Err(error))
type ErrPattern struct {
	Variable string // 绑定的变量名，如 "error"
}

// SomePattern Some模式 (如 Some(value))
type SomePattern struct {
	Variable string // 绑定的变量名，如 "value"
}

// NonePattern None模式 (如 None)
type NonePattern struct{}

// OkPattern 实现
func (o *OkPattern) String() string {
	return fmt.Sprintf("OkPattern{Variable: %s}", o.Variable)
}

func (o *OkPattern) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitOkPattern(o)
}

func (o *OkPattern) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitOkPattern(o)
}

// ErrPattern 实现
func (e *ErrPattern) String() string {
	return fmt.Sprintf("ErrPattern{Variable: %s}", e.Variable)
}

func (e *ErrPattern) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitErrPattern(e)
}

func (e *ErrPattern) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitErrPattern(e)
}

// SomePattern 实现
func (s *SomePattern) String() string {
	return fmt.Sprintf("SomePattern{Variable: %s}", s.Variable)
}

func (s *SomePattern) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitSomePattern(s)
}

func (s *SomePattern) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitSomePattern(s)
}

// NonePattern 实现
func (n *NonePattern) String() string {
	return "NonePattern{}"
}

func (n *NonePattern) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitNonePattern(n)
}

func (n *NonePattern) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitNonePattern(n)
}

// 通配符模式 (_)
type WildcardPattern struct{}

func (w *WildcardPattern) String() string {
	return "WildcardPattern{}"
}

func (w *WildcardPattern) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitWildcardPattern(w)
}

func (w *WildcardPattern) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitWildcardPattern(w)
}

// 标识符模式（变量绑定）
type IdentifierPattern struct {
	Name string // 绑定的变量名
}

func NewIdentifierPattern(name string) *IdentifierPattern {
	return &IdentifierPattern{Name: name}
}

func (i *IdentifierPattern) String() string {
	return fmt.Sprintf("IdentifierPattern{Name: %s}", i.Name)
}

func (i *IdentifierPattern) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitIdentifierPattern(i)
}

func (i *IdentifierPattern) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitIdentifierPattern(i)
}

// 字面量模式
type LiteralPattern struct {
	Value interface{} // 字面量值
	Kind  string      // 值类型: "int", "string", "bool"
}

func NewLiteralPattern(value interface{}, kind string) *LiteralPattern {
	return &LiteralPattern{Value: value, Kind: kind}
}

func (l *LiteralPattern) String() string {
	return fmt.Sprintf("LiteralPattern{Value: %v, Kind: %s}", l.Value, l.Kind)
}

func (l *LiteralPattern) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitLiteralPattern(l)
}

func (l *LiteralPattern) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitLiteralPattern(l)
}

// 结构体模式
type StructPattern struct {
	TypeName string               // 结构体类型名
	Fields   []StructPatternField // 字段模式
}

type StructPatternField struct {
	FieldName string // 字段名
	Pattern   Expr   // 字段的匹配模式
}

func NewStructPattern(typeName string, fields []StructPatternField) *StructPattern {
	return &StructPattern{TypeName: typeName, Fields: fields}
}

func NewStructPatternField(fieldName string, pattern Expr) StructPatternField {
	return StructPatternField{FieldName: fieldName, Pattern: pattern}
}

func (s *StructPattern) String() string {
	fieldStrs := make([]string, len(s.Fields))
	for i, f := range s.Fields {
		fieldStrs[i] = fmt.Sprintf("%s: %s", f.FieldName, f.Pattern)
	}
	return fmt.Sprintf("StructPattern{Type: %s, Fields: {%s}}", s.TypeName, strings.Join(fieldStrs, ", "))
}

func (s *StructPattern) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitStructPattern(s)
}

func (s *StructPattern) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitStructPattern(s)
}

// 数组模式
type ArrayPattern struct {
	Elements []Expr // 数组元素模式，nil表示通配符
	Rest     Expr   // 剩余元素模式 (如 ...rest)
}

func NewArrayPattern(elements []Expr, rest Expr) *ArrayPattern {
	return &ArrayPattern{Elements: elements, Rest: rest}
}

func (a *ArrayPattern) String() string {
	elemStrs := make([]string, len(a.Elements))
	for i, elem := range a.Elements {
		if elem == nil {
			elemStrs[i] = "_"
		} else {
			elemStrs[i] = elem.String()
		}
	}
	result := fmt.Sprintf("ArrayPattern{Elements: [%s]", strings.Join(elemStrs, ", "))
	if a.Rest != nil {
		result += fmt.Sprintf(", Rest: %s", a.Rest)
	}
	result += "}"
	return result
}

func (a *ArrayPattern) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitArrayPattern(a)
}

func (a *ArrayPattern) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitArrayPattern(a)
}

// 元组模式 (如果需要的话)
type TuplePattern struct {
	Elements []Expr // 元组元素模式
}

func NewTuplePattern(elements []Expr) *TuplePattern {
	return &TuplePattern{Elements: elements}
}

func (t *TuplePattern) String() string {
	elemStrs := make([]string, len(t.Elements))
	for i, elem := range t.Elements {
		elemStrs[i] = elem.String()
	}
	return fmt.Sprintf("TuplePattern{(%s)}", strings.Join(elemStrs, ", "))
}

func (t *TuplePattern) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitTuplePattern(t)
}

func (t *TuplePattern) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitTuplePattern(t)
}

func (c MatchCase) String() string {
	return fmt.Sprintf("%s => %v", c.Pattern, c.Body)
}

// ArrayLiteral 数组字面量
type ArrayLiteral struct {
	Elements []Expr // 数组元素
}

func (a *ArrayLiteral) String() string {
	elements := make([]string, len(a.Elements))
	for i, e := range a.Elements {
		elements[i] = e.String()
	}
	return fmt.Sprintf("ArrayLiteral{[%s]}", strings.Join(elements, ", "))
}

func (a *ArrayLiteral) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitArrayLiteral(a)
}

func (a *ArrayLiteral) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitArrayLiteral(a)
}

// IndexExpr 数组索引访问表达式
type IndexExpr struct {
	Array Expr // 被索引的数组表达式
	Index Expr // 索引表达式
}

func (i *IndexExpr) String() string {
	return fmt.Sprintf("IndexExpr{Array: %s, Index: %s}", i.Array, i.Index)
}

func (i *IndexExpr) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitIndexExpr(i)
}

func (i *IndexExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitIndexExpr(i)
}

// LenExpr 数组长度获取表达式
type LenExpr struct {
	Array Expr // 数组表达式
}

func (l *LenExpr) String() string {
	return fmt.Sprintf("LenExpr{Array: %s}", l.Array)
}

func (l *LenExpr) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitLenExpr(l)
}

func (l *LenExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitLenExpr(l)
}

// SizeOfExpr sizeof 表达式节点
// 表示 sizeof(T) 表达式，用于获取类型大小
type SizeOfExpr struct {
	TypeName string // 类型名称（如 "int", "T", "Point"）
}

func NewSizeOfExpr(typeName string) *SizeOfExpr {
	return &SizeOfExpr{TypeName: typeName}
}

func (s *SizeOfExpr) String() string {
	return fmt.Sprintf("SizeOfExpr{Type: %s}", s.TypeName)
}

func (s *SizeOfExpr) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitSizeOfExpr(s)
}

func (s *SizeOfExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitSizeOfExpr(s)
}

// TypeCastExpr 类型转换表达式节点
// 表示 expr as Type 表达式，用于显式类型转换
// 例如：new_ptr as []T
type TypeCastExpr struct {
	Expr      Expr   // 要转换的表达式
	TargetType string // 目标类型（如 "[]T", "*i8"）
}

func NewTypeCastExpr(expr Expr, targetType string) *TypeCastExpr {
	return &TypeCastExpr{
		Expr:      expr,
		TargetType: targetType,
	}
}

func (t *TypeCastExpr) String() string {
	return fmt.Sprintf("TypeCastExpr{Expr: %s, TargetType: %s}", t.Expr.String(), t.TargetType)
}

func (t *TypeCastExpr) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitTypeCastExpr(t)
}

func (t *TypeCastExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitTypeCastExpr(t)
}

// FunctionPointerExpr 函数指针表达式节点
// 表示 &func_name 表达式，用于获取函数指针
type FunctionPointerExpr struct {
	FunctionName string // 函数名称
}

func NewFunctionPointerExpr(functionName string) *FunctionPointerExpr {
	return &FunctionPointerExpr{FunctionName: functionName}
}

func (f *FunctionPointerExpr) String() string {
	return fmt.Sprintf("FunctionPointerExpr{Function: %s}", f.FunctionName)
}

func (f *FunctionPointerExpr) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitFunctionPointerExpr(f)
}

func (f *FunctionPointerExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitFunctionPointerExpr(f)
}

// AddressOfExpr 取地址表达式节点
// 表示 &expression 表达式，用于获取变量、数组元素、结构体字段等的地址
type AddressOfExpr struct {
	Operand Expr // 操作数表达式（可以是标识符、索引访问、字段访问等）
}

func NewAddressOfExpr(operand Expr) *AddressOfExpr {
	return &AddressOfExpr{Operand: operand}
}

func (a *AddressOfExpr) String() string {
	return fmt.Sprintf("AddressOfExpr{Operand: %s}", a.Operand)
}

func (a *AddressOfExpr) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitAddressOfExpr(a)
}

func (a *AddressOfExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitAddressOfExpr(a)
}

// 确保 AddressOfExpr 实现 Expr 接口
var _ Expr = (*AddressOfExpr)(nil)

// DereferenceExpr 解引用表达式节点
// 表示 *expression 表达式，用于解引用指针，获取指针指向的值
type DereferenceExpr struct {
	Operand Expr // 操作数表达式（必须是指针类型）
}

func NewDereferenceExpr(operand Expr) *DereferenceExpr {
	return &DereferenceExpr{Operand: operand}
}

func (d *DereferenceExpr) String() string {
	return fmt.Sprintf("DereferenceExpr{Operand: %s}", d.Operand)
}

func (d *DereferenceExpr) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitDereferenceExpr(d)
}

func (d *DereferenceExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitDereferenceExpr(d)
}

// 确保 DereferenceExpr 实现 Expr 接口
var _ Expr = (*DereferenceExpr)(nil)

// SliceExpr 数组切片表达式
type SliceExpr struct {
	Array Expr // 被切片的数组表达式
	Start Expr // 开始索引（可选）
	End   Expr // 结束索引（可选）
}

func (s *SliceExpr) String() string {
	return fmt.Sprintf("SliceExpr{Array: %s, Start: %s, End: %s}", s.Array, s.Start, s.End)
}

func (s *SliceExpr) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitSliceExpr(s)
}

func (s *SliceExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitSliceExpr(s)
}

// ArrayMethodCallExpr 数组方法调用表达式
type ArrayMethodCallExpr struct {
	Array  Expr   // 数组表达式
	Method string // 方法名 (push, pop, insert, remove)
	Args   []Expr // 方法参数
}

func (a *ArrayMethodCallExpr) String() string {
	argsStr := ""
	for i, arg := range a.Args {
		if i > 0 {
			argsStr += ", "
		}
		argsStr += arg.String()
	}
	return fmt.Sprintf("%s.%s(%s)", a.Array, a.Method, argsStr)
}

func (a *ArrayMethodCallExpr) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitArrayMethodCallExpr(a)
}

func (a *ArrayMethodCallExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitArrayMethodCallExpr(a)
}

// ResultExpr Result表达式 (如 Result[int, string])
type ResultExpr struct {
	OkType  string // 成功值的类型
	ErrType string // 错误值的类型
}

func (r *ResultExpr) String() string {
	return fmt.Sprintf("Result[%s, %s]", r.OkType, r.ErrType)
}

func (r *ResultExpr) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitResultExpr(r)
}

func (r *ResultExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitResultExpr(r)
}

// OkLiteral Ok字面量 (如 Ok(value))
type OkLiteral struct {
	Value Expr // 成功值
}

func (o *OkLiteral) String() string {
	return fmt.Sprintf("Ok(%s)", o.Value)
}

func (o *OkLiteral) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitOkLiteral(o)
}

func (o *OkLiteral) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitOkLiteral(o)
}

// ErrLiteral Err字面量 (如 Err(error))
type ErrLiteral struct {
	Error Expr // 错误值
}

func (e *ErrLiteral) String() string {
	return fmt.Sprintf("Err(%s)", e.Error)
}

func (e *ErrLiteral) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitErrLiteral(e)
}

func (e *ErrLiteral) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitErrLiteral(e)
}

// ErrorPropagation 错误传播操作符 (如 expr?)
type ErrorPropagation struct {
	Expr Expr // 要传播错误的表达式
}

func (e *ErrorPropagation) String() string {
	return fmt.Sprintf("%s?", e.Expr)
}

func (e *ErrorPropagation) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitErrorPropagation(e)
}

func (e *ErrorPropagation) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitErrorPropagation(e)
}

// MethodCallExpr 方法调用表达式
type MethodCallExpr struct {
	Receiver   Expr     // 接收者表达式
	MethodName string   // 方法名
	Args       []Expr   // 参数列表
	TypeArgs   []string // 类型参数（用于泛型方法）
}

func (m *MethodCallExpr) String() string {
	args := make([]string, len(m.Args))
	for i, arg := range m.Args {
		args[i] = arg.String()
	}

	result := fmt.Sprintf("%s.%s(%s)", m.Receiver, m.MethodName, strings.Join(args, ", "))
	if len(m.TypeArgs) > 0 {
		result = fmt.Sprintf("%s.%s[%s](%s)", m.Receiver, m.MethodName, strings.Join(m.TypeArgs, ", "), strings.Join(args, ", "))
	}
	return result
}

func (m *MethodCallExpr) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitMethodCallExpr(m)
}

func (m *MethodCallExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitMethodCallExpr(m)
}

// TraitDef Trait定义
type TraitDef struct {
	Name            string           // Trait名称
	TypeParams      []GenericParam   // 类型参数 (泛型)，支持约束
	SuperTraits     []string         // 继承的Trait列表
	AssociatedTypes []AssociatedType // 关联类型声明
	Methods         []TraitMethod    // 方法列表
}

// ImplDef Trait实现定义
type ImplDef struct {
	TraitName string      // 实现的Trait名称
	Methods   []MethodDef // 实现的方法列表
}

func (i *ImplDef) String() string {
	return fmt.Sprintf("ImplDef{TraitName: %s, Methods: %d}", i.TraitName, len(i.Methods))
}

func (i *ImplDef) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitImplDef(i)
}

func (t *TraitDef) String() string {
	methods := make([]string, len(t.Methods))
	for i, m := range t.Methods {
		methods[i] = m.String()
	}
	typeParams := make([]string, len(t.TypeParams))
	for i, tp := range t.TypeParams {
		typeParams[i] = tp.String()
	}
	associatedTypes := make([]string, len(t.AssociatedTypes))
	for i, at := range t.AssociatedTypes {
		associatedTypes[i] = at.String()
	}
	superTraits := strings.Join(t.SuperTraits, ", ")
	return fmt.Sprintf("TraitDef{Name: %s, TypeParams: [%s], SuperTraits: [%s], AssociatedTypes: [%s], Methods: [%s]}",
		t.Name, strings.Join(typeParams, ", "), superTraits, strings.Join(associatedTypes, ", "), strings.Join(methods, ", "))
}

func (t *TraitDef) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitTraitDef(t)
}

// TraitMethod Trait方法
type TraitMethod struct {
	Name       string         // 方法名
	TypeParams []GenericParam // 方法类型参数 (新增)
	Params     []Param        // 参数列表
	ReturnType string         // 返回类型
	Body       []ASTNode      // 默认实现 (可选)
	IsDefault  bool           // 是否有默认实现
}

func (m TraitMethod) String() string {
	params := make([]string, len(m.Params))
	for i, p := range m.Params {
		params[i] = p.String()
	}
	typeParams := make([]string, len(m.TypeParams))
	for i, tp := range m.TypeParams {
		typeParams[i] = tp.String()
	}
	return fmt.Sprintf("TraitMethod{Name: %s, TypeParams: [%s], Params: [%s], ReturnType: %s, IsDefault: %v}",
		m.Name, strings.Join(typeParams, ", "), strings.Join(params, ", "), m.ReturnType, m.IsDefault)
}

// ImplAnnotation 实现注解（@impl TraitName）
type ImplAnnotation struct {
	TraitName string   // Trait名称
	TypeArgs  []string // 类型参数，如 ["int", "string"]
}

func (i *ImplAnnotation) String() string {
	if len(i.TypeArgs) > 0 {
		return fmt.Sprintf("@impl %s[%s]", i.TraitName, strings.Join(i.TypeArgs, ", "))
	}
	return fmt.Sprintf("@impl %s", i.TraitName)
}

// AssociatedType 关联类型声明 (在Trait定义中)
type AssociatedType struct {
	Name string // 关联类型名称，如 "Item"
}

func (a *AssociatedType) String() string {
	return fmt.Sprintf("type %s;", a.Name)
}

// AssociatedTypeImpl 关联类型实现 (在Trait实现中)
// AssociatedTypeDef 关联类型定义节点（在Trait中声明）
type AssociatedTypeDef struct {
	Name string // 关联类型名，如 "Item"
}

func (a *AssociatedTypeDef) String() string {
	return fmt.Sprintf("type %s;", a.Name)
}

func (a *AssociatedTypeDef) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitAssociatedTypeDef(a)
}

// AssociatedTypeImpl 关联类型实现节点（在Trait实现中指定）
type AssociatedTypeImpl struct {
	Name       string // 关联类型名称，如 "Item"
	TargetType string // 具体类型，如 "int"
}

func (a *AssociatedTypeImpl) String() string {
	return fmt.Sprintf("type %s = %s;", a.Name, a.TargetType)
}

// DynamicTraitRef 动态Trait引用节点（如 &dyn Trait）
type DynamicTraitRef struct {
	TraitName string // Trait名称
}

func (d *DynamicTraitRef) String() string {
	return fmt.Sprintf("&dyn %s", d.TraitName)
}

func (d *DynamicTraitRef) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitDynamicTraitRef(d)
}

func (d *DynamicTraitRef) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitDynamicTraitRef(d)
}

// MethodDef 方法定义节点
type MethodDef struct {
	Receiver       string         // 接收者类型 (如 "Point")
	ReceiverVar    string         // 接收者变量名 (如 "p")
	ReceiverParams []GenericParam // 接收者类型参数 (如 Point[T, U])
	TypeParams     []GenericParam // 方法类型参数 (如 methodName[V])
	Name           string         // 方法名
	Params         []Param        // 参数列表
	ReturnType     string         // 返回类型
	Body           []ASTNode      // 方法体
}

func (m *MethodDef) String() string {
	params := make([]string, len(m.Params))
	for i, p := range m.Params {
		params[i] = p.String()
	}
	typeParams := make([]string, len(m.TypeParams))
	for i, tp := range m.TypeParams {
		typeParams[i] = tp.String()
	}
	receiverParams := make([]string, len(m.ReceiverParams))
	for i, rp := range m.ReceiverParams {
		receiverParams[i] = rp.String()
	}

	receiverStr := m.Receiver
	if len(m.ReceiverParams) > 0 {
		receiverStr += "[" + strings.Join(receiverParams, ", ") + "]"
	}

	return fmt.Sprintf("MethodDef{Receiver: (%s %s), TypeParams: [%s], Name: %s, Params: [%s], ReturnType: %s, Body: %v}",
		m.ReceiverVar, receiverStr, strings.Join(typeParams, ", "), m.Name, strings.Join(params, ", "), m.ReturnType, len(m.Body))
}

func (m *MethodDef) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitMethodDef(m)
}

// EnumDef 枚举定义节点
type EnumDef struct {
	Name       string           // 枚举名
	ImplTraits []ImplAnnotation // 实现的traits（通过注解）
	Variants   []EnumVariant    // 枚举值列表
}

func (e *EnumDef) String() string {
	traits := make([]string, len(e.ImplTraits))
	for i, t := range e.ImplTraits {
		traits[i] = t.String()
	}
	variants := make([]string, len(e.Variants))
	for i, v := range e.Variants {
		variants[i] = v.String()
	}
	return fmt.Sprintf("EnumDef{Name: %s, ImplTraits: [%s], Variants: [%s]}",
		e.Name, strings.Join(traits, ", "), strings.Join(variants, ", "))
}

func (e *EnumDef) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitEnumDef(e)
}

// EnumVariant 枚举值
type EnumVariant struct {
	Name string // 枚举值名
}

func (v EnumVariant) String() string {
	return v.Name
}

// GenericParam 类型参数节点
type GenericParam struct {
	Name        string   // 参数名，如 "T"
	Constraints []string // 约束条件，如 ["Printable", "Serializable"]
}

func (g *GenericParam) String() string {
	constraints := ""
	if len(g.Constraints) > 0 {
		constraints = ": " + strings.Join(g.Constraints, " + ")
	}
	return fmt.Sprintf("%s%s", g.Name, constraints)
}

// GenericType 泛型类型引用节点
type GenericType struct {
	BaseType string   // 基础类型，如 "Container"
	TypeArgs []string // 类型参数，如 ["int", "string"]
}

func (g *GenericType) String() string {
	args := strings.Join(g.TypeArgs, ", ")
	return fmt.Sprintf("%s[%s]", g.BaseType, args)
}

func (g *GenericType) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitGenericType(g)
}

// ResultType Result类型节点
type ResultType struct {
	Type string // 内部类型，如 "int" 表示 Result[int]
}

func (r *ResultType) String() string {
	return fmt.Sprintf("Result[%s]", r.Type)
}

// OptionType Option类型节点
type OptionType struct {
	Type string // 内部类型，如 "string" 表示 Option[string]
}

func (o *OptionType) String() string {
	return fmt.Sprintf("Option[%s]", o.Type)
}

// OptionExpr Option表达式节点
type OptionExpr struct {
	Expr Expr // 被包装的表达式
}

func (o *OptionExpr) String() string {
	return fmt.Sprintf("Option(%s)", o.Expr)
}

func (o *OptionExpr) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitOptionExpr(o)
}

func (o *OptionExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitOptionExpr(o)
}

// SomeLiteral Some字面量
type SomeLiteral struct {
	Value Expr // 存在的值
}

func (s *SomeLiteral) String() string {
	return fmt.Sprintf("Some(%s)", s.Value)
}

func (s *SomeLiteral) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitSomeLiteral(s)
}

func (s *SomeLiteral) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitSomeLiteral(s)
}

// NoneLiteral None字面量
type NoneLiteral struct{}

func (n *NoneLiteral) String() string {
	return "None"
}

func (n *NoneLiteral) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitNoneLiteral(n)
}

func (n *NoneLiteral) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitNoneLiteral(n)
}

// BoolLiteral 布尔字面量
type BoolLiteral struct {
	Value bool
}

func (b *BoolLiteral) String() string {
	if b.Value {
		return "true"
	}
	return "false"
}

func (b *BoolLiteral) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitBoolLiteral(b)
}

func (b *BoolLiteral) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitBoolLiteral(b)
}

// Expr 表达式接口
type Expr interface {
	String() string
	AcceptExpr(visitor ExprVisitor) interface{}
}

// ExprVisitor 表达式访问者接口
type ExprVisitor interface {
	VisitIdentifier(expr *Identifier) interface{}
	VisitStringLiteral(expr *StringLiteral) interface{}
	VisitIntLiteral(expr *IntLiteral) interface{}
	VisitFloatLiteral(expr *FloatLiteral) interface{}
	VisitBinaryExpr(expr *BinaryExpr) interface{}
	VisitFuncCall(expr *FuncCall) interface{}
	VisitResultExpr(expr *ResultExpr) interface{}
	VisitOptionExpr(expr *OptionExpr) interface{}
	VisitOkLiteral(expr *OkLiteral) interface{}
	VisitErrLiteral(expr *ErrLiteral) interface{}
	VisitErrorPropagation(expr *ErrorPropagation) interface{}
	VisitSomeLiteral(expr *SomeLiteral) interface{}
	VisitNoneLiteral(expr *NoneLiteral) interface{}
	VisitBoolLiteral(expr *BoolLiteral) interface{}
	VisitOkPattern(pattern *OkPattern) interface{}
	VisitErrPattern(pattern *ErrPattern) interface{}
	VisitSomePattern(pattern *SomePattern) interface{}
	VisitNonePattern(pattern *NonePattern) interface{}
	VisitWildcardPattern(pattern *WildcardPattern) interface{}
	VisitIdentifierPattern(pattern *IdentifierPattern) interface{}
	VisitLiteralPattern(pattern *LiteralPattern) interface{}
	VisitStructPattern(pattern *StructPattern) interface{}
	VisitArrayPattern(pattern *ArrayPattern) interface{}
	VisitTuplePattern(pattern *TuplePattern) interface{}
	VisitAwaitExpr(expr *AwaitExpr) interface{}
	VisitChanLiteral(literal *ChanLiteral) interface{}
	VisitSendExpr(expr *SendExpr) interface{}
	VisitReceiveExpr(expr *ReceiveExpr) interface{}
	VisitSpawnExpr(expr *SpawnExpr) interface{}
	VisitIndexExpr(expr *IndexExpr) interface{}
	VisitLenExpr(expr *LenExpr) interface{}
	VisitSizeOfExpr(expr *SizeOfExpr) interface{}
	VisitTypeCastExpr(expr *TypeCastExpr) interface{}
	VisitFunctionPointerExpr(expr *FunctionPointerExpr) interface{}
	VisitAddressOfExpr(expr *AddressOfExpr) interface{}
	VisitDereferenceExpr(expr *DereferenceExpr) interface{}
	VisitSliceExpr(expr *SliceExpr) interface{}
	VisitArrayMethodCallExpr(expr *ArrayMethodCallExpr) interface{}
	VisitMethodCallExpr(expr *MethodCallExpr) interface{}
	VisitDynamicTraitRef(expr *DynamicTraitRef) interface{}
	VisitMatchExpr(expr *MatchExpr) interface{}
	VisitArrayLiteral(expr *ArrayLiteral) interface{}
	VisitStructAccess(expr *StructAccess) interface{}
	VisitStructLiteral(expr *StructLiteral) interface{}
	VisitNamespaceAccessExpr(expr *NamespaceAccessExpr) interface{}
}

// Identifier 标识符表达式
type Identifier struct {
	Name string
}

func (i *Identifier) String() string {
	return fmt.Sprintf("Identifier{Name: %s}", i.Name)
}

func (i *Identifier) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitIdentifier(i)
}

// StringLiteral 字符串字面量
type StringLiteral struct {
	Value string
}

func (s *StringLiteral) String() string {
	return fmt.Sprintf("StringLiteral{Value: %q}", s.Value)
}

func (s *StringLiteral) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitStringLiteral(s)
}

// IntLiteral 整数字面量
type IntLiteral struct {
	Value int
}

func (i *IntLiteral) String() string {
	return fmt.Sprintf("IntLiteral{Value: %d}", i.Value)
}

func (i *IntLiteral) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitIntLiteral(i)
}

// FloatLiteral 浮点数字面量
type FloatLiteral struct {
	Value float64
}

func (f *FloatLiteral) String() string {
	return fmt.Sprintf("FloatLiteral{Value: %f}", f.Value)
}

func (f *FloatLiteral) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitFloatLiteral(f)
}

// BinaryExpr 二元表达式
type BinaryExpr struct {
	Left  Expr
	Op    string
	Right Expr
}

func (b *BinaryExpr) String() string {
	return fmt.Sprintf("BinaryExpr{Left: %s, Op: %s, Right: %s}", b.Left, b.Op, b.Right)
}

func (b *BinaryExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitBinaryExpr(b)
}

// TernaryExpr 三元表达式（条件表达式）
// 格式：if condition { value1 } else { value2 }
type TernaryExpr struct {
	Condition Expr // 条件表达式
	ThenValue Expr // then 分支的值
	ElseValue Expr // else 分支的值
}

func (t *TernaryExpr) String() string {
	return fmt.Sprintf("TernaryExpr{Condition: %s, Then: %s, Else: %s}", t.Condition, t.ThenValue, t.ElseValue)
}

func (t *TernaryExpr) AcceptExpr(visitor ExprVisitor) interface{} {
	// 注意：ExprVisitor 接口可能没有 VisitTernaryExpr 方法
	// 暂时返回 nil，后续需要添加 VisitTernaryExpr 到 ExprVisitor 接口
	return nil
}

// FuncCall 函数调用节点
type FuncCall struct {
	Name     string   // 函数名
	TypeArgs []string // 类型参数，如 ["int", "string"] 用于泛型函数调用
	Args     []Expr   // 值参数，支持表达式参数
}

func (f *FuncCall) String() string {
	typeArgs := ""
	if len(f.TypeArgs) > 0 {
		typeArgs = fmt.Sprintf("[%s]", strings.Join(f.TypeArgs, ", "))
	}
	return fmt.Sprintf("FuncCall{Name: %s%s, Args: %v}", f.Name, typeArgs, f.Args)
}

func (f *FuncCall) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitFuncCall(f)
}

func (f *FuncCall) AcceptExpr(visitor ExprVisitor) interface{} {
	return visitor.VisitFuncCall(f)
}

// FromImportElement 表示 from ... import 的单个元素（名称及可选别名）
type FromImportElement struct {
	Name  string // 导入的符号名
	Alias string // 别名，空表示无别名
}

// FromImportStatement 表示 from <path> import elem1, elem2 ... 语句
// 用于类型推断阶段将导入的符号注册到 functionTable/symbolTable
type FromImportStatement struct {
	ImportPath string               // 导入路径，如 "math"、"utils"
	Elements   []FromImportElement  // 导入的元素列表
}

func (f *FromImportStatement) String() string {
	parts := make([]string, len(f.Elements))
	for i, e := range f.Elements {
		if e.Alias != "" {
			parts[i] = e.Name + " as " + e.Alias
		} else {
			parts[i] = e.Name
		}
	}
	return fmt.Sprintf("FromImport{from %s import [%s]}", f.ImportPath, strings.Join(parts, ", "))
}

func (f *FromImportStatement) Accept(visitor ASTVisitor) interface{} {
	return nil
}

// Program 程序AST根节点
type Program struct {
	Statements []ASTNode
}

func (p *Program) String() string {
	stmts := make([]string, len(p.Statements))
	for i, stmt := range p.Statements {
		stmts[i] = stmt.String()
	}
	return fmt.Sprintf("Program{Statements: [%s]}", fmt.Sprintf("%v", stmts))
}

func (p *Program) Accept(visitor ASTVisitor) interface{} {
	return visitor.VisitProgram(p)
}
