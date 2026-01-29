// Package value_objects 定义AST相关值对象
package value_objects

import "fmt"

// ASTNode AST节点接口
type ASTNode interface {
	// Location 返回节点在源代码中的位置
	Location() SourceLocation

	// NodeType 返回节点类型
	NodeType() string

	// String 返回节点的字符串表示
	String() string
}

// ProgramAST 程序AST值对象
type ProgramAST struct {
	nodes             []ASTNode
	sourceFile        *SourceFile
	filename          string // 源文件名
	location          SourceLocation
	packageDecl       *PackageDeclaration // 包声明（新增）
	moduleDeclaration *ModuleDeclaration // 模块声明（完善实现）
	imports           []*ImportStatement // 导入语句列表（完善实现）
}

// NewProgramAST 创建新的程序AST
func NewProgramAST(nodes []ASTNode, filename string, location SourceLocation) *ProgramAST {
	return &ProgramAST{
		nodes:             nodes,
		sourceFile:        nil, // 暂时设为nil
		filename:          filename,
		location:          location,
		packageDecl:       nil,                         // 初始化为nil
		moduleDeclaration: nil,                         // 初始化为nil
		imports:           make([]*ImportStatement, 0), // 初始化为空列表
	}
}

// Nodes 获取所有AST节点
func (pa *ProgramAST) Nodes() []ASTNode {
	return pa.nodes
}

// SourceFile 获取源文件
func (pa *ProgramAST) SourceFile() *SourceFile {
	return pa.sourceFile
}

// Location 获取位置信息
func (pa *ProgramAST) Location() SourceLocation {
	return pa.location
}

// NodeType 返回节点类型
func (pa *ProgramAST) NodeType() string {
	return "Program"
}

// String 返回字符串表示
func (pa *ProgramAST) String() string {
	return "ProgramAST"
}

// ResolvedAST 解析歧义后的AST值对象
type ResolvedAST struct {
	originalAST   *ProgramAST
	resolvedNodes []ASTNode
	resolutions   []Resolution
}

// Resolution 歧义解析记录
type Resolution struct {
	ambiguousNode ASTNode
	resolvedNode  ASTNode
	strategy      string
	reason        string
}

// NewResolvedAST 创建新的解析歧义AST
func NewResolvedAST(originalAST *ProgramAST, resolvedNodes []ASTNode, resolutions []Resolution) *ResolvedAST {
	return &ResolvedAST{
		originalAST:   originalAST,
		resolvedNodes: resolvedNodes,
		resolutions:   resolutions,
	}
}

// OriginalAST 获取原始AST
func (ra *ResolvedAST) OriginalAST() *ProgramAST {
	return ra.originalAST
}

// ResolvedNodes 获取解析后的节点
func (ra *ResolvedAST) ResolvedNodes() []ASTNode {
	return ra.resolvedNodes
}

// Resolutions 获取解析记录
func (ra *ResolvedAST) Resolutions() []Resolution {
	return ra.resolutions
}

// AddNode 添加节点到程序AST
func (pa *ProgramAST) AddNode(node ASTNode) {
	pa.nodes = append(pa.nodes, node)
}

// SetPackage 设置包声明
func (pa *ProgramAST) SetPackage(pkg *PackageDeclaration) {
	pa.packageDecl = pkg
}

// Package 获取包声明
func (pa *ProgramAST) Package() *PackageDeclaration {
	return pa.packageDecl
}

// SetModuleDeclaration 设置模块声明
func (pa *ProgramAST) SetModuleDeclaration(moduleDecl *ModuleDeclaration) {
	// 完善实现：设置模块声明
	pa.moduleDeclaration = moduleDecl
}

// AddImport 添加导入语句
func (pa *ProgramAST) AddImport(importStmt *ImportStatement) {
	// 完善实现：添加导入语句到列表
	if importStmt != nil {
		pa.imports = append(pa.imports, importStmt)
	}
}

// ModuleDeclaration 获取模块声明
func (pa *ProgramAST) ModuleDeclaration() *ModuleDeclaration {
	return pa.moduleDeclaration
}

// Imports 获取导入语句列表
func (pa *ProgramAST) Imports() []*ImportStatement {
	return pa.imports
}

// ModuleDeclaration 模块声明
type ModuleDeclaration struct {
	name     string
	path     string
	location SourceLocation
}

// NewModuleDeclaration 创建新的模块声明
func NewModuleDeclaration(name, path string, location SourceLocation) *ModuleDeclaration {
	return &ModuleDeclaration{
		name:     name,
		path:     path,
		location: location,
	}
}

// Name 获取模块名
func (md *ModuleDeclaration) Name() string {
	return md.name
}

// Path 获取模块路径
func (md *ModuleDeclaration) Path() string {
	return md.path
}

// Location 获取位置
func (md *ModuleDeclaration) Location() SourceLocation {
	return md.location
}

// NodeType 返回节点类型
func (md *ModuleDeclaration) NodeType() string {
	return "ModuleDeclaration"
}

// String 返回字符串表示
func (md *ModuleDeclaration) String() string {
	return "ModuleDeclaration{" + md.name + "}"
}

// PackageDeclaration 包声明节点
type PackageDeclaration struct {
	packageName string
	location    SourceLocation
}

// NewPackageDeclaration 创建新的包声明
func NewPackageDeclaration(packageName string, location SourceLocation) *PackageDeclaration {
	return &PackageDeclaration{
		packageName: packageName,
		location:    location,
	}
}

// PackageName 获取包名
func (pd *PackageDeclaration) PackageName() string {
	return pd.packageName
}

// Location 获取位置
func (pd *PackageDeclaration) Location() SourceLocation {
	return pd.location
}

// NodeType 返回节点类型
func (pd *PackageDeclaration) NodeType() string {
	return "PackageDeclaration"
}

// String 返回字符串表示
func (pd *PackageDeclaration) String() string {
	return fmt.Sprintf("PackageDeclaration{package: %s}", pd.packageName)
}

// ImportType 导入类型
type ImportType int

const (
	ImportTypePackage ImportType = iota // 包级导入：import "path"
	ImportTypeElements                  // 元素级导入：from "path" import ...
)

// ImportElement 导入元素（用于 from ... import）
type ImportElement struct {
	name     string
	alias    string
	location SourceLocation
}

// NewImportElement 创建新的导入元素
func NewImportElement(name, alias string, location SourceLocation) *ImportElement {
	return &ImportElement{
		name:     name,
		alias:    alias,
		location: location,
	}
}

// Name 获取元素名
func (ie *ImportElement) Name() string {
	return ie.name
}

// Alias 获取别名
func (ie *ImportElement) Alias() string {
	return ie.alias
}

// Location 获取位置
func (ie *ImportElement) Location() SourceLocation {
	return ie.location
}

// ImportStatement 导入语句（扩展）
type ImportStatement struct {
	importPath string
	alias      string
	location   SourceLocation
	importType ImportType        // 导入类型：包级或元素级
	elements   []*ImportElement   // 元素级导入的元素列表
}

// NewImportStatement 创建新的导入语句（兼容旧接口）
func NewImportStatement(importPath, alias string, location SourceLocation) *ImportStatement {
	return NewPackageImport(importPath, alias, location)
}

// NewPackageImport 创建包级导入
func NewPackageImport(importPath, alias string, location SourceLocation) *ImportStatement {
	return &ImportStatement{
		importPath: importPath,
		alias:      alias,
		location:   location,
		importType: ImportTypePackage,
		elements:   nil,
	}
}

// NewElementImport 创建元素级导入
func NewElementImport(importPath string, elements []*ImportElement, location SourceLocation) *ImportStatement {
	return &ImportStatement{
		importPath: importPath,
		alias:      "",
		location:   location,
		importType: ImportTypeElements,
		elements:   elements,
	}
}

// ImportPath 获取导入路径
func (is *ImportStatement) ImportPath() string {
	return is.importPath
}

// Alias 获取别名
func (is *ImportStatement) Alias() string {
	return is.alias
}

// Location 获取位置
func (is *ImportStatement) Location() SourceLocation {
	return is.location
}

// NodeType 返回节点类型
func (is *ImportStatement) NodeType() string {
	return "ImportStatement"
}

// ImportType 获取导入类型
func (is *ImportStatement) ImportType() ImportType {
	return is.importType
}

// Elements 获取导入元素列表
func (is *ImportStatement) Elements() []*ImportElement {
	return is.elements
}

// String 返回字符串表示
func (is *ImportStatement) String() string {
	if is.importType == ImportTypePackage {
		if is.alias != "" {
			return fmt.Sprintf("ImportStatement{package: %s as %s}", is.importPath, is.alias)
		}
		return fmt.Sprintf("ImportStatement{package: %s}", is.importPath)
	}
	// 元素级导入
	elementsStr := ""
	for i, elem := range is.elements {
		if i > 0 {
			elementsStr += ", "
		}
		if elem.alias != "" {
			elementsStr += fmt.Sprintf("%s as %s", elem.name, elem.alias)
		} else {
			elementsStr += elem.name
		}
	}
	return fmt.Sprintf("ImportStatement{from: %s import [%s]}", is.importPath, elementsStr)
}

// GenericParameter 泛型参数
type GenericParameter struct {
	name        string
	constraints []string // 类型约束
	location    SourceLocation
}

// NewGenericParameter 创建新的泛型参数
func NewGenericParameter(name string, constraints []string, location SourceLocation) *GenericParameter {
	return &GenericParameter{
		name:        name,
		constraints: constraints,
		location:    location,
	}
}

// Name 获取参数名
func (gp *GenericParameter) Name() string {
	return gp.name
}

// Constraints 获取约束
func (gp *GenericParameter) Constraints() []string {
	return gp.constraints
}

// Location 获取位置
func (gp *GenericParameter) Location() SourceLocation {
	return gp.location
}

// NodeType 返回节点类型
func (gp *GenericParameter) NodeType() string {
	return "GenericParameter"
}

// String 返回字符串表示
func (gp *GenericParameter) String() string {
	return "GenericParameter{" + gp.name + "}"
}

// Parameter 函数参数
type Parameter struct {
	name           string
	typeAnnotation *TypeAnnotation
	location       SourceLocation
}

// NewParameter 创建新的参数
func NewParameter(name string, typeAnnotation *TypeAnnotation, location SourceLocation) *Parameter {
	return &Parameter{
		name:           name,
		typeAnnotation: typeAnnotation,
		location:       location,
	}
}

// Name 获取参数名
func (p *Parameter) Name() string {
	return p.name
}

// TypeAnnotation 获取类型注解
func (p *Parameter) TypeAnnotation() *TypeAnnotation {
	return p.typeAnnotation
}

// Location 获取位置
func (p *Parameter) Location() SourceLocation {
	return p.location
}

// NodeType 返回节点类型
func (p *Parameter) NodeType() string {
	return "Parameter"
}

// String 返回字符串表示
func (p *Parameter) String() string {
	return "Parameter{" + p.name + ": " + p.typeAnnotation.String() + "}"
}

// TypeAnnotation 类型注解
type TypeAnnotation struct {
	typeName    string
	genericArgs []*TypeAnnotation
	isPointer   bool
	location    SourceLocation
}

// NewTypeAnnotation 创建新的类型注解
func NewTypeAnnotation(typeName string, genericArgs []*TypeAnnotation, isPointer bool, location SourceLocation) *TypeAnnotation {
	return &TypeAnnotation{
		typeName:    typeName,
		genericArgs: genericArgs,
		isPointer:   isPointer,
		location:    location,
	}
}

// TypeName 获取类型名
func (ta *TypeAnnotation) TypeName() string {
	return ta.typeName
}

// GenericArgs 获取泛型参数
func (ta *TypeAnnotation) GenericArgs() []*TypeAnnotation {
	return ta.genericArgs
}

// IsPointer 是否为指针类型
func (ta *TypeAnnotation) IsPointer() bool {
	return ta.isPointer
}

// Location 获取位置
func (ta *TypeAnnotation) Location() SourceLocation {
	return ta.location
}

// NodeType 返回节点类型
func (ta *TypeAnnotation) NodeType() string {
	return "TypeAnnotation"
}

// String 返回字符串表示
func (ta *TypeAnnotation) String() string {
	result := ta.typeName
	if ta.isPointer {
		result = "*" + result
	}
	if len(ta.genericArgs) > 0 {
		result += "<"
		for i, arg := range ta.genericArgs {
			if i > 0 {
				result += ", "
			}
			result += arg.String()
		}
		result += ">"
	}
	return result
}

// BlockStatement 块语句
type BlockStatement struct {
	statements []ASTNode
	location   SourceLocation
}

// NewBlockStatement 创建新的块语句
func NewBlockStatement(statements []ASTNode, location SourceLocation) *BlockStatement {
	return &BlockStatement{
		statements: statements,
		location:   location,
	}
}

// Statements 获取语句列表
func (bs *BlockStatement) Statements() []ASTNode {
	return bs.statements
}

// Location 获取位置
func (bs *BlockStatement) Location() SourceLocation {
	return bs.location
}

// NodeType 返回节点类型
func (bs *BlockStatement) NodeType() string {
	return "BlockStatement"
}

// String 返回字符串表示
func (bs *BlockStatement) String() string {
	return "BlockStatement{...}"
}

// FunctionDeclaration 函数声明
type FunctionDeclaration struct {
	name          string
	visibility    Visibility // 可见性（新增）
	genericParams []*GenericParameter
	parameters    []*Parameter
	returnType    *TypeAnnotation
	body          *BlockStatement
	location      SourceLocation
}

// NewFunctionDeclaration 创建新的函数声明
func NewFunctionDeclaration(name string, genericParams []*GenericParameter, parameters []*Parameter, returnType *TypeAnnotation, body *BlockStatement, location SourceLocation) *FunctionDeclaration {
	return NewFunctionDeclarationWithVisibility(name, VisibilityPublic, genericParams, parameters, returnType, body, location)
}

// NewFunctionDeclarationWithVisibility 创建新的函数声明（带可见性）
func NewFunctionDeclarationWithVisibility(name string, visibility Visibility, genericParams []*GenericParameter, parameters []*Parameter, returnType *TypeAnnotation, body *BlockStatement, location SourceLocation) *FunctionDeclaration {
	return &FunctionDeclaration{
		name:          name,
		visibility:    visibility,
		genericParams: genericParams,
		parameters:    parameters,
		returnType:    returnType,
		body:          body,
		location:      location,
	}
}

// Name 获取函数名
func (fd *FunctionDeclaration) Name() string {
	return fd.name
}

// Visibility 获取可见性
func (fd *FunctionDeclaration) Visibility() Visibility {
	return fd.visibility
}

// GenericParams 获取泛型参数
func (fd *FunctionDeclaration) GenericParams() []*GenericParameter {
	return fd.genericParams
}

// Parameters 获取参数列表
func (fd *FunctionDeclaration) Parameters() []*Parameter {
	return fd.parameters
}

// ReturnType 获取返回类型
func (fd *FunctionDeclaration) ReturnType() *TypeAnnotation {
	return fd.returnType
}

// Body 获取函数体
func (fd *FunctionDeclaration) Body() *BlockStatement {
	return fd.body
}

// Location 获取位置
func (fd *FunctionDeclaration) Location() SourceLocation {
	return fd.location
}

// NodeType 返回节点类型
func (fd *FunctionDeclaration) NodeType() string {
	return "FunctionDeclaration"
}

// String 返回字符串表示
func (fd *FunctionDeclaration) String() string {
	return "FunctionDeclaration{" + fd.name + "}"
}

// AsyncFunctionDeclaration 异步函数声明
type AsyncFunctionDeclaration struct {
	functionDeclaration *FunctionDeclaration
	location            SourceLocation
}

// NewAsyncFunctionDeclaration 创建新的异步函数声明
func NewAsyncFunctionDeclaration(functionDeclaration *FunctionDeclaration, location SourceLocation) *AsyncFunctionDeclaration {
	return &AsyncFunctionDeclaration{
		functionDeclaration: functionDeclaration,
		location:            location,
	}
}

// FunctionDeclaration 获取基础函数声明
func (afd *AsyncFunctionDeclaration) FunctionDeclaration() *FunctionDeclaration {
	return afd.functionDeclaration
}

// Location 获取位置
func (afd *AsyncFunctionDeclaration) Location() SourceLocation {
	return afd.location
}

// NodeType 返回节点类型
func (afd *AsyncFunctionDeclaration) NodeType() string {
	return "AsyncFunctionDeclaration"
}

// String 返回字符串表示
func (afd *AsyncFunctionDeclaration) String() string {
	return "AsyncFunctionDeclaration{" + afd.functionDeclaration.Name() + "}"
}

// StructField 结构体字段
type StructField struct {
	name      string
	fieldType *TypeAnnotation
	location  SourceLocation
}

// NewStructField 创建新的结构体字段
func NewStructField(name string, fieldType *TypeAnnotation, location SourceLocation) *StructField {
	return &StructField{
		name:      name,
		fieldType: fieldType,
		location:  location,
	}
}

// Name 获取字段名
func (sf *StructField) Name() string {
	return sf.name
}

// FieldType 获取字段类型
func (sf *StructField) FieldType() *TypeAnnotation {
	return sf.fieldType
}

// Location 获取位置
func (sf *StructField) Location() SourceLocation {
	return sf.location
}

// NodeType 返回节点类型
func (sf *StructField) NodeType() string {
	return "StructField"
}

// String 返回字符串表示
func (sf *StructField) String() string {
	return "StructField{" + sf.name + ": " + sf.fieldType.String() + "}"
}

// StructDeclaration 结构体声明
type StructDeclaration struct {
	name     string
	fields   []*StructField
	location SourceLocation
}

// NewStructDeclaration 创建新的结构体声明
func NewStructDeclaration(name string, fields []*StructField, location SourceLocation) *StructDeclaration {
	return &StructDeclaration{
		name:     name,
		fields:   fields,
		location: location,
	}
}

// Name 获取结构体名
func (sd *StructDeclaration) Name() string {
	return sd.name
}

// Fields 获取字段列表
func (sd *StructDeclaration) Fields() []*StructField {
	return sd.fields
}

// Location 获取位置
func (sd *StructDeclaration) Location() SourceLocation {
	return sd.location
}

// NodeType 返回节点类型
func (sd *StructDeclaration) NodeType() string {
	return "StructDeclaration"
}

// String 返回字符串表示
func (sd *StructDeclaration) String() string {
	return "StructDeclaration{" + sd.name + "}"
}

// EnumVariant 枚举变体
type EnumVariant struct {
	name     string
	location SourceLocation
}

// NewEnumVariant 创建新的枚举变体
func NewEnumVariant(name string, location SourceLocation) *EnumVariant {
	return &EnumVariant{
		name:     name,
		location: location,
	}
}

// Name 获取变体名
func (ev *EnumVariant) Name() string {
	return ev.name
}

// Location 获取位置
func (ev *EnumVariant) Location() SourceLocation {
	return ev.location
}

// NodeType 返回节点类型
func (ev *EnumVariant) NodeType() string {
	return "EnumVariant"
}

// String 返回字符串表示
func (ev *EnumVariant) String() string {
	return "EnumVariant{" + ev.name + "}"
}

// EnumDeclaration 枚举声明
type EnumDeclaration struct {
	name     string
	variants []*EnumVariant
	location SourceLocation
}

// NewEnumDeclaration 创建新的枚举声明
func NewEnumDeclaration(name string, variants []*EnumVariant, location SourceLocation) *EnumDeclaration {
	return &EnumDeclaration{
		name:     name,
		variants: variants,
		location: location,
	}
}

// Name 获取枚举名
func (ed *EnumDeclaration) Name() string {
	return ed.name
}

// Variants 获取变体列表
func (ed *EnumDeclaration) Variants() []*EnumVariant {
	return ed.variants
}

// Location 获取位置
func (ed *EnumDeclaration) Location() SourceLocation {
	return ed.location
}

// NodeType 返回节点类型
func (ed *EnumDeclaration) NodeType() string {
	return "EnumDeclaration"
}

// String 返回字符串表示
func (ed *EnumDeclaration) String() string {
	return "EnumDeclaration{" + ed.name + "}"
}

// TraitMethod Trait方法
type TraitMethod struct {
	name       string
	typeParams []*GenericParameter // 方法级泛型参数，如 func foo[T, U]()
	parameters []*Parameter
	returnType *TypeAnnotation
	isAbstract bool
	location   SourceLocation
}

// NewTraitMethod 创建新的Trait方法
// typeParams 可为 nil，表示无泛型参数
func NewTraitMethod(name string, typeParams []*GenericParameter, parameters []*Parameter, returnType *TypeAnnotation, isAbstract bool, location SourceLocation) *TraitMethod {
	return &TraitMethod{
		name:       name,
		typeParams: typeParams,
		parameters: parameters,
		returnType: returnType,
		isAbstract: isAbstract,
		location:   location,
	}
}

// Name 获取方法名
func (tm *TraitMethod) Name() string {
	return tm.name
}

// TypeParams 获取方法级泛型参数
func (tm *TraitMethod) TypeParams() []*GenericParameter {
	return tm.typeParams
}

// Parameters 获取参数列表
func (tm *TraitMethod) Parameters() []*Parameter {
	return tm.parameters
}

// ReturnType 获取返回类型
func (tm *TraitMethod) ReturnType() *TypeAnnotation {
	return tm.returnType
}

// IsAbstract 是否为抽象方法
func (tm *TraitMethod) IsAbstract() bool {
	return tm.isAbstract
}

// Location 获取位置
func (tm *TraitMethod) Location() SourceLocation {
	return tm.location
}

// NodeType 返回节点类型
func (tm *TraitMethod) NodeType() string {
	return "TraitMethod"
}

// String 返回字符串表示
func (tm *TraitMethod) String() string {
	return "TraitMethod{" + tm.name + "}"
}

// TraitDeclaration Trait声明
type TraitDeclaration struct {
	name          string
	genericParams []*GenericParameter
	methods       []*TraitMethod
	location      SourceLocation
}

// NewTraitDeclaration 创建新的Trait声明
func NewTraitDeclaration(name string, genericParams []*GenericParameter, methods []*TraitMethod, location SourceLocation) *TraitDeclaration {
	return &TraitDeclaration{
		name:          name,
		genericParams: genericParams,
		methods:       methods,
		location:      location,
	}
}

// Name 获取Trait名
func (td *TraitDeclaration) Name() string {
	return td.name
}

// GenericParams 获取泛型参数
func (td *TraitDeclaration) GenericParams() []*GenericParameter {
	return td.genericParams
}

// Methods 获取方法列表
func (td *TraitDeclaration) Methods() []*TraitMethod {
	return td.methods
}

// Location 获取位置
func (td *TraitDeclaration) Location() SourceLocation {
	return td.location
}

// NodeType 返回节点类型
func (td *TraitDeclaration) NodeType() string {
	return "TraitDeclaration"
}

// String 返回字符串表示
func (td *TraitDeclaration) String() string {
	return "TraitDeclaration{" + td.name + "}"
}

// ImplDeclaration 实现声明
type ImplDeclaration struct {
	traitType  *TypeAnnotation
	targetType *TypeAnnotation
	methods    []*FunctionDeclaration
	location   SourceLocation
}

// NewImplDeclaration 创建新的实现声明
func NewImplDeclaration(traitType, targetType *TypeAnnotation, methods []*FunctionDeclaration, location SourceLocation) *ImplDeclaration {
	return &ImplDeclaration{
		traitType:  traitType,
		targetType: targetType,
		methods:    methods,
		location:   location,
	}
}

// TraitType 获取Trait类型
func (id *ImplDeclaration) TraitType() *TypeAnnotation {
	return id.traitType
}

// TargetType 获取目标类型
func (id *ImplDeclaration) TargetType() *TypeAnnotation {
	return id.targetType
}

// Methods 获取方法列表
func (id *ImplDeclaration) Methods() []*FunctionDeclaration {
	return id.methods
}

// Location 获取位置
func (id *ImplDeclaration) Location() SourceLocation {
	return id.location
}

// NodeType 返回节点类型
func (id *ImplDeclaration) NodeType() string {
	return "ImplDeclaration"
}

// String 返回字符串表示
func (id *ImplDeclaration) String() string {
	return "ImplDeclaration{" + id.traitType.String() + " for " + id.targetType.String() + "}"
}

// ReturnStatement 返回语句
type ReturnStatement struct {
	expression ASTNode
	location   SourceLocation
}

// NewReturnStatement 创建新的返回语句
func NewReturnStatement(expression ASTNode, location SourceLocation) *ReturnStatement {
	return &ReturnStatement{
		expression: expression,
		location:   location,
	}
}

// Expression 获取返回值表达式
func (rs *ReturnStatement) Expression() ASTNode {
	return rs.expression
}

// Location 获取位置
func (rs *ReturnStatement) Location() SourceLocation {
	return rs.location
}

// NodeType 返回节点类型
func (rs *ReturnStatement) NodeType() string {
	return "ReturnStatement"
}

// String 返回字符串表示
func (rs *ReturnStatement) String() string {
	return "ReturnStatement{...}"
}

// ExpressionStatement 表达式语句
type ExpressionStatement struct {
	expression ASTNode
	location   SourceLocation
}

// NewExpressionStatement 创建新的表达式语句
func NewExpressionStatement(expression ASTNode, location SourceLocation) *ExpressionStatement {
	return &ExpressionStatement{
		expression: expression,
		location:   location,
	}
}

// Expression 获取表达式
func (es *ExpressionStatement) Expression() ASTNode {
	return es.expression
}

// Location 获取位置
func (es *ExpressionStatement) Location() SourceLocation {
	return es.location
}

// NodeType 返回节点类型
func (es *ExpressionStatement) NodeType() string {
	return "ExpressionStatement"
}

// String 返回字符串表示
func (es *ExpressionStatement) String() string {
	return fmt.Sprintf("ExpressionStatement{%s}", es.expression.String())
}

// TypeAliasDeclaration 类型别名声明
type TypeAliasDeclaration struct {
	aliasName    string
	originalType *TypeAnnotation
	location     SourceLocation
}

// NewTypeAliasDeclaration 创建新的类型别名声明
func NewTypeAliasDeclaration(aliasName string, originalType *TypeAnnotation, location SourceLocation) *TypeAliasDeclaration {
	return &TypeAliasDeclaration{
		aliasName:    aliasName,
		originalType: originalType,
		location:     location,
	}
}

// AliasName 获取别名名称
func (tad *TypeAliasDeclaration) AliasName() string {
	return tad.aliasName
}

// OriginalType 获取原始类型
func (tad *TypeAliasDeclaration) OriginalType() *TypeAnnotation {
	return tad.originalType
}

// Location 获取位置
func (tad *TypeAliasDeclaration) Location() SourceLocation {
	return tad.location
}

// NodeType 返回节点类型
func (tad *TypeAliasDeclaration) NodeType() string {
	return "TypeAliasDeclaration"
}

// String 返回字符串表示
func (tad *TypeAliasDeclaration) String() string {
	return fmt.Sprintf("TypeAliasDeclaration{alias=%s, type=%s}", tad.aliasName, tad.originalType.String())
}

// ObjectProperty 对象属性
type ObjectProperty struct {
	name     string
	value    ASTNode
	location SourceLocation
}

// NewObjectProperty 创建新的对象属性
func NewObjectProperty(name string, value ASTNode, location SourceLocation) *ObjectProperty {
	return &ObjectProperty{
		name:     name,
		value:    value,
		location: location,
	}
}

// Name 获取属性名
func (op *ObjectProperty) Name() string {
	return op.name
}

// Value 获取属性值
func (op *ObjectProperty) Value() ASTNode {
	return op.value
}

// Location 获取位置
func (op *ObjectProperty) Location() SourceLocation {
	return op.location
}

// NodeType 返回节点类型
func (op *ObjectProperty) NodeType() string {
	return "ObjectProperty"
}

// String 返回字符串表示
func (op *ObjectProperty) String() string {
	return "ObjectProperty{" + op.name + ": ...}"
}

// VariableDeclaration 变量声明
type VariableDeclaration struct {
	name        string
	varType     *TypeAnnotation
	initializer ASTNode
	location    SourceLocation
}

// NewVariableDeclaration 创建新的变量声明
func NewVariableDeclaration(name string, varType *TypeAnnotation, initializer ASTNode, location SourceLocation) *VariableDeclaration {
	return &VariableDeclaration{
		name:        name,
		varType:     varType,
		initializer: initializer,
		location:    location,
	}
}

// Name 获取变量名
func (vd *VariableDeclaration) Name() string {
	return vd.name
}

// VarType 获取变量类型
func (vd *VariableDeclaration) VarType() *TypeAnnotation {
	return vd.varType
}

// Initializer 获取初始化表达式
func (vd *VariableDeclaration) Initializer() ASTNode {
	return vd.initializer
}

// Location 获取位置
func (vd *VariableDeclaration) Location() SourceLocation {
	return vd.location
}

// NodeType 返回节点类型
func (vd *VariableDeclaration) NodeType() string {
	return "VariableDeclaration"
}

// String 返回字符串表示
func (vd *VariableDeclaration) String() string {
	return "VariableDeclaration{" + vd.name + "}"
}

// BinaryExpression 二元表达式
type BinaryExpression struct {
	left     ASTNode
	operator string
	right    ASTNode
	location SourceLocation
}

// NewBinaryExpression 创建新的二元表达式
func NewBinaryExpression(left ASTNode, operator string, right ASTNode, location SourceLocation) *BinaryExpression {
	return &BinaryExpression{
		left:     left,
		operator: operator,
		right:    right,
		location: location,
	}
}

// Left 获取左操作数
func (be *BinaryExpression) Left() ASTNode {
	return be.left
}

// Operator 获取运算符
func (be *BinaryExpression) Operator() string {
	return be.operator
}

// Right 获取右操作数
func (be *BinaryExpression) Right() ASTNode {
	return be.right
}

// Location 获取位置
func (be *BinaryExpression) Location() SourceLocation {
	return be.location
}

// NodeType 返回节点类型
func (be *BinaryExpression) NodeType() string {
	return "BinaryExpression"
}

// String 返回字符串表示
func (be *BinaryExpression) String() string {
	return "BinaryExpression{...}"
}

// UnaryExpression 一元表达式
type UnaryExpression struct {
	operator string
	operand  ASTNode
	location SourceLocation
}

// NewUnaryExpression 创建新的一元表达式
func NewUnaryExpression(operator string, operand ASTNode, location SourceLocation) *UnaryExpression {
	return &UnaryExpression{
		operator: operator,
		operand:  operand,
		location: location,
	}
}

// Operator 获取运算符
func (ue *UnaryExpression) Operator() string {
	return ue.operator
}

// Operand 获取操作数
func (ue *UnaryExpression) Operand() ASTNode {
	return ue.operand
}

// Location 获取位置
func (ue *UnaryExpression) Location() SourceLocation {
	return ue.location
}

// NodeType 返回节点类型
func (ue *UnaryExpression) NodeType() string {
	return "UnaryExpression"
}

// String 返回字符串表示
func (ue *UnaryExpression) String() string {
	return "UnaryExpression{...}"
}

// Identifier 标识符
type Identifier struct {
	name     string
	location SourceLocation
}

// NewIdentifier 创建新的标识符
func NewIdentifier(name string, location SourceLocation) *Identifier {
	return &Identifier{
		name:     name,
		location: location,
	}
}

// Name 获取标识符名
func (i *Identifier) Name() string {
	return i.name
}

// Location 获取位置
func (i *Identifier) Location() SourceLocation {
	return i.location
}

// NodeType 返回节点类型
func (i *Identifier) NodeType() string {
	return "Identifier"
}

// String 返回字符串表示
func (i *Identifier) String() string {
	return i.name
}

// IntegerLiteral 整数字面量
type IntegerLiteral struct {
	value    int64
	location SourceLocation
}

// NewIntegerLiteral 创建新的整数字面量
func NewIntegerLiteral(value int64, location SourceLocation) *IntegerLiteral {
	return &IntegerLiteral{
		value:    value,
		location: location,
	}
}

// Value 获取整数值
func (il *IntegerLiteral) Value() int64 {
	return il.value
}

// Location 获取位置
func (il *IntegerLiteral) Location() SourceLocation {
	return il.location
}

// NodeType 返回节点类型
func (il *IntegerLiteral) NodeType() string {
	return "IntegerLiteral"
}

// String 返回字符串表示
func (il *IntegerLiteral) String() string {
	return fmt.Sprintf("%d", il.value)
}

// FloatLiteral 浮点数字面量
type FloatLiteral struct {
	value    float64
	location SourceLocation
}

// NewFloatLiteral 创建新的浮点数字面量
func NewFloatLiteral(value float64, location SourceLocation) *FloatLiteral {
	return &FloatLiteral{
		value:    value,
		location: location,
	}
}

// Value 获取浮点数值
func (fl *FloatLiteral) Value() float64 {
	return fl.value
}

// Location 获取位置
func (fl *FloatLiteral) Location() SourceLocation {
	return fl.location
}

// NodeType 返回节点类型
func (fl *FloatLiteral) NodeType() string {
	return "FloatLiteral"
}

// String 返回字符串表示
func (fl *FloatLiteral) String() string {
	return fmt.Sprintf("%g", fl.value)
}

// StringLiteral 字符串字面量
type StringLiteral struct {
	value    string
	location SourceLocation
}

// NewStringLiteral 创建新的字符串字面量
func NewStringLiteral(value string, location SourceLocation) *StringLiteral {
	return &StringLiteral{
		value:    value,
		location: location,
	}
}

// Value 获取字符串值
func (sl *StringLiteral) Value() string {
	return sl.value
}

// Location 获取位置
func (sl *StringLiteral) Location() SourceLocation {
	return sl.location
}

// NodeType 返回节点类型
func (sl *StringLiteral) NodeType() string {
	return "StringLiteral"
}

// String 返回字符串表示
func (sl *StringLiteral) String() string {
	return fmt.Sprintf("\"%s\"", sl.value)
}

// BooleanLiteral 布尔字面量
type BooleanLiteral struct {
	value    bool
	location SourceLocation
}

// NewBooleanLiteral 创建新的布尔字面量
func NewBooleanLiteral(value bool, location SourceLocation) *BooleanLiteral {
	return &BooleanLiteral{
		value:    value,
		location: location,
	}
}

// Value 获取布尔值
func (bl *BooleanLiteral) Value() bool {
	return bl.value
}

// Location 获取位置
func (bl *BooleanLiteral) Location() SourceLocation {
	return bl.location
}

// NodeType 返回节点类型
func (bl *BooleanLiteral) NodeType() string {
	return "BooleanLiteral"
}

// String 返回字符串表示
func (bl *BooleanLiteral) String() string {
	return fmt.Sprintf("%t", bl.value)
}
