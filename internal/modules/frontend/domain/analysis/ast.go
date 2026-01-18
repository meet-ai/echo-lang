package analysis

import "fmt"

// ASTNode 表示抽象语法树节点的接口
type ASTNode interface {
	// Accept 接受访问者模式
	Accept(visitor ASTVisitor) error
	// Position 返回节点在源代码中的位置
	Position() Position
	// String 返回节点的字符串表示
	String() string
}

// ASTVisitor 访问者接口，用于遍历AST
type ASTVisitor interface {
	VisitProgram(*Program) error
	VisitFunctionDef(*FunctionDef) error
	VisitVariableDecl(*VariableDecl) error
	VisitIfStmt(*IfStmt) error
	VisitForStmt(*ForStmt) error
	VisitWhileStmt(*WhileStmt) error
	VisitReturnStmt(*ReturnStmt) error
	VisitExprStmt(*ExprStmt) error
	VisitBlock(*Block) error
	VisitBinaryExpr(*BinaryExpr) error
	VisitUnaryExpr(*UnaryExpr) error
	VisitLiteralExpr(*LiteralExpr) error
	VisitIdentifierExpr(*IdentifierExpr) error
	VisitFunctionCallExpr(*FunctionCallExpr) error
	VisitStructLiteralExpr(*StructLiteralExpr) error
	VisitArrayLiteralExpr(*ArrayLiteralExpr) error
	VisitMatchExpr(*MatchExpr) error
	VisitAssignExpr(*AssignExpr) error
}

// Program 根节点，表示整个程序
type Program struct {
	statements []ASTNode
	position   Position
}

func NewProgram(statements []ASTNode, position Position) *Program {
	return &Program{
		statements: statements,
		position:   position,
	}
}

func (p *Program) Statements() []ASTNode { return p.statements }
func (p *Program) Position() Position    { return p.position }

func (p *Program) Accept(visitor ASTVisitor) error {
	return visitor.VisitProgram(p)
}

func (p *Program) String() string {
	return fmt.Sprintf("Program{statements=%d, position=%s}", len(p.statements), p.position.String())
}

// FunctionDef 函数定义
type FunctionDef struct {
	name       string
	params     []Parameter
	returnType *TypeAnnotation
	body       *Block
	position   Position
}

func NewFunctionDef(name string, params []Parameter, returnType *TypeAnnotation, body *Block, position Position) *FunctionDef {
	return &FunctionDef{
		name:       name,
		params:     params,
		returnType: returnType,
		body:       body,
		position:   position,
	}
}

func (f *FunctionDef) Name() string                { return f.name }
func (f *FunctionDef) Params() []Parameter         { return f.params }
func (f *FunctionDef) ReturnType() *TypeAnnotation { return f.returnType }
func (f *FunctionDef) Body() *Block                { return f.body }
func (f *FunctionDef) Position() Position          { return f.position }

func (f *FunctionDef) Accept(visitor ASTVisitor) error {
	return visitor.VisitFunctionDef(f)
}

func (f *FunctionDef) String() string {
	return fmt.Sprintf("FunctionDef{name=%s, params=%d, position=%s}", f.name, len(f.params), f.position.String())
}

// Parameter 函数参数
type Parameter struct {
	name  string
	type_ *TypeAnnotation
}

func NewParameter(name string, type_ *TypeAnnotation) Parameter {
	return Parameter{name: name, type_: type_}
}

func (p Parameter) Name() string          { return p.name }
func (p Parameter) Type() *TypeAnnotation { return p.type_ }

// VariableDecl 变量声明
type VariableDecl struct {
	name     string
	type_    *TypeAnnotation
	initExpr ASTNode
	position Position
}

func NewVariableDecl(name string, type_ *TypeAnnotation, initExpr ASTNode, position Position) *VariableDecl {
	return &VariableDecl{
		name:     name,
		type_:    type_,
		initExpr: initExpr,
		position: position,
	}
}

func (v *VariableDecl) Name() string          { return v.name }
func (v *VariableDecl) Type() *TypeAnnotation { return v.type_ }
func (v *VariableDecl) InitExpr() ASTNode     { return v.initExpr }
func (v *VariableDecl) Position() Position    { return v.position }

func (v *VariableDecl) Accept(visitor ASTVisitor) error {
	return visitor.VisitVariableDecl(v)
}

func (v *VariableDecl) String() string {
	return fmt.Sprintf("VariableDecl{name=%s, position=%s}", v.name, v.position.String())
}

// TypeAnnotation 类型注解
type TypeAnnotation struct {
	name     string
	position Position
}

func NewTypeAnnotation(name string, position Position) *TypeAnnotation {
	return &TypeAnnotation{
		name:     name,
		position: position,
	}
}

func (t *TypeAnnotation) Name() string       { return t.name }
func (t *TypeAnnotation) Position() Position { return t.position }

// Block 代码块
type Block struct {
	statements []ASTNode
	position   Position
}

func NewBlock(statements []ASTNode, position Position) *Block {
	return &Block{
		statements: statements,
		position:   position,
	}
}

func (b *Block) Statements() []ASTNode { return b.statements }
func (b *Block) Position() Position    { return b.position }

func (b *Block) Accept(visitor ASTVisitor) error {
	return visitor.VisitBlock(b)
}

func (b *Block) String() string {
	return fmt.Sprintf("Block{statements=%d, position=%s}", len(b.statements), b.position.String())
}

// IfStmt if语句
type IfStmt struct {
	condition ASTNode
	thenBody  *Block
	elseBody  *Block
	position  Position
}

func NewIfStmt(condition ASTNode, thenBody *Block, elseBody *Block, position Position) *IfStmt {
	return &IfStmt{
		condition: condition,
		thenBody:  thenBody,
		elseBody:  elseBody,
		position:  position,
	}
}

func (i *IfStmt) Condition() ASTNode { return i.condition }
func (i *IfStmt) ThenBody() *Block   { return i.thenBody }
func (i *IfStmt) ElseBody() *Block   { return i.elseBody }
func (i *IfStmt) Position() Position { return i.position }

func (i *IfStmt) Accept(visitor ASTVisitor) error {
	return visitor.VisitIfStmt(i)
}

func (i *IfStmt) String() string {
	return fmt.Sprintf("IfStmt{position=%s}", i.position.String())
}

// ForStmt for循环
type ForStmt struct {
	init      ASTNode
	condition ASTNode
	update    ASTNode
	body      *Block
	position  Position
}

func NewForStmt(init ASTNode, condition ASTNode, update ASTNode, body *Block, position Position) *ForStmt {
	return &ForStmt{
		init:      init,
		condition: condition,
		update:    update,
		body:      body,
		position:  position,
	}
}

func (f *ForStmt) Init() ASTNode      { return f.init }
func (f *ForStmt) Condition() ASTNode { return f.condition }
func (f *ForStmt) Update() ASTNode    { return f.update }
func (f *ForStmt) Body() *Block       { return f.body }
func (f *ForStmt) Position() Position { return f.position }

func (f *ForStmt) Accept(visitor ASTVisitor) error {
	return visitor.VisitForStmt(f)
}

func (f *ForStmt) String() string {
	return fmt.Sprintf("ForStmt{position=%s}", f.position.String())
}

// WhileStmt while循环
type WhileStmt struct {
	condition ASTNode
	body      *Block
	position  Position
}

func NewWhileStmt(condition ASTNode, body *Block, position Position) *WhileStmt {
	return &WhileStmt{
		condition: condition,
		body:      body,
		position:  position,
	}
}

func (w *WhileStmt) Condition() ASTNode { return w.condition }
func (w *WhileStmt) Body() *Block       { return w.body }
func (w *WhileStmt) Position() Position { return w.position }

func (w *WhileStmt) Accept(visitor ASTVisitor) error {
	return visitor.VisitWhileStmt(w)
}

func (w *WhileStmt) String() string {
	return fmt.Sprintf("WhileStmt{position=%s}", w.position.String())
}

// ReturnStmt 返回语句
type ReturnStmt struct {
	expr     ASTNode
	position Position
}

func NewReturnStmt(expr ASTNode, position Position) *ReturnStmt {
	return &ReturnStmt{
		expr:     expr,
		position: position,
	}
}

func (r *ReturnStmt) Expr() ASTNode      { return r.expr }
func (r *ReturnStmt) Position() Position { return r.position }

func (r *ReturnStmt) Accept(visitor ASTVisitor) error {
	return visitor.VisitReturnStmt(r)
}

func (r *ReturnStmt) String() string {
	return fmt.Sprintf("ReturnStmt{position=%s}", r.position.String())
}

// ExprStmt 表达式语句
type ExprStmt struct {
	expr     ASTNode
	position Position
}

func NewExprStmt(expr ASTNode, position Position) *ExprStmt {
	return &ExprStmt{
		expr:     expr,
		position: position,
	}
}

func (e *ExprStmt) Expr() ASTNode      { return e.expr }
func (e *ExprStmt) Position() Position { return e.position }

func (e *ExprStmt) Accept(visitor ASTVisitor) error {
	return visitor.VisitExprStmt(e)
}

func (e *ExprStmt) String() string {
	return fmt.Sprintf("ExprStmt{position=%s}", e.position.String())
}
