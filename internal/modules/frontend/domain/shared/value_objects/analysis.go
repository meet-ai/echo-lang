// Package value_objects 定义分析相关值对象
package value_objects

import "time"

// AnalyzedProgram 语义分析后的程序值对象
type AnalyzedProgram struct {
	originalAST *ProgramAST
	symbolTable *SymbolTable
	typeInfo    map[ASTNode]*TypeInfo
	analyzedAt  time.Time
}

// NewAnalyzedProgram 创建新的分析程序
func NewAnalyzedProgram(originalAST *ProgramAST, symbolTable *SymbolTable) *AnalyzedProgram {
	return &AnalyzedProgram{
		originalAST: originalAST,
		symbolTable: symbolTable,
		typeInfo:    make(map[ASTNode]*TypeInfo),
		analyzedAt:  time.Now(),
	}
}

// OriginalAST 获取原始AST
func (ap *AnalyzedProgram) OriginalAST() *ProgramAST {
	return ap.originalAST
}

// SymbolTable 获取符号表
func (ap *AnalyzedProgram) SymbolTable() *SymbolTable {
	return ap.symbolTable
}

// SetTypeInfo 设置节点类型信息
func (ap *AnalyzedProgram) SetTypeInfo(node ASTNode, typeInfo *TypeInfo) {
	ap.typeInfo[node] = typeInfo
}

// GetTypeInfo 获取节点类型信息
func (ap *AnalyzedProgram) GetTypeInfo(node ASTNode) (*TypeInfo, bool) {
	typeInfo, exists := ap.typeInfo[node]
	return typeInfo, exists
}

// AllTypeInfo 获取所有类型信息
func (ap *AnalyzedProgram) AllTypeInfo() map[ASTNode]*TypeInfo {
	return ap.typeInfo
}

// AnalyzedAt 获取分析时间
func (ap *AnalyzedProgram) AnalyzedAt() time.Time {
	return ap.analyzedAt
}

// TypeInfo 类型信息值对象
type TypeInfo struct {
	typeName      string
	category      TypeCategory
	modifiers     []TypeModifier
	elementType   *TypeInfo    // 对于数组类型
	returnType    *TypeInfo    // 对于函数类型
	parameterTypes []*TypeInfo // 对于函数类型
}

// TypeCategory 类型类别
type TypeCategory string

const (
	CategoryPrimitive TypeCategory = "primitive" // 基本类型
	CategoryStruct    TypeCategory = "struct"    // 结构体
	CategoryEnum      TypeCategory = "enum"      // 枚举
	CategoryFunction  TypeCategory = "function"  // 函数
	CategoryArray     TypeCategory = "array"     // 数组
	CategoryChannel   TypeCategory = "channel"   // 通道
)

// TypeModifier 类型修饰符
type TypeModifier string

const (
	ModifierConst    TypeModifier = "const"    // 常量
	ModifierMutable  TypeModifier = "mutable"  // 可变
	ModifierAsync    TypeModifier = "async"    // 异步
)

// NewTypeInfo 创建新的类型信息
func NewTypeInfo(typeName string, category TypeCategory) *TypeInfo {
	return &TypeInfo{
		typeName:  typeName,
		category:  category,
		modifiers: make([]TypeModifier, 0),
	}
}

// TypeName 获取类型名称
func (ti *TypeInfo) TypeName() string {
	return ti.typeName
}

// Category 获取类型类别
func (ti *TypeInfo) Category() TypeCategory {
	return ti.category
}

// Modifiers 获取类型修饰符
func (ti *TypeInfo) Modifiers() []TypeModifier {
	return ti.modifiers
}

// AddModifier 添加修饰符
func (ti *TypeInfo) AddModifier(modifier TypeModifier) {
	ti.modifiers = append(ti.modifiers, modifier)
}

// HasModifier 检查是否包含修饰符
func (ti *TypeInfo) HasModifier(modifier TypeModifier) bool {
	for _, m := range ti.modifiers {
		if m == modifier {
			return true
		}
	}
	return false
}

// ElementType 获取元素类型（数组）
func (ti *TypeInfo) ElementType() *TypeInfo {
	return ti.elementType
}

// SetElementType 设置元素类型
func (ti *TypeInfo) SetElementType(elementType *TypeInfo) {
	ti.elementType = elementType
}

// ReturnType 获取返回类型（函数）
func (ti *TypeInfo) ReturnType() *TypeInfo {
	return ti.returnType
}

// SetReturnType 设置返回类型
func (ti *TypeInfo) SetReturnType(returnType *TypeInfo) {
	ti.returnType = returnType
}

// ParameterTypes 获取参数类型列表
func (ti *TypeInfo) ParameterTypes() []*TypeInfo {
	return ti.parameterTypes
}

// SetParameterTypes 设置参数类型列表
func (ti *TypeInfo) SetParameterTypes(parameterTypes []*TypeInfo) {
	ti.parameterTypes = parameterTypes
}

// String 返回类型信息的字符串表示
func (ti *TypeInfo) String() string {
	result := ti.typeName
	for _, modifier := range ti.modifiers {
		result = string(modifier) + " " + result
	}
	return result
}

// TypeCheckResult 类型检查结果值对象
type TypeCheckResult struct {
	isValid     bool
	errors      []*ParseError
	warnings    []*ParseWarning
	typeErrors  []*TypeError
	checkedAt   time.Time
}

// TypeError 类型错误
type TypeError struct {
	node        ASTNode
	expectedType *TypeInfo
	actualType   *TypeInfo
	message     string
	location    SourceLocation
}

// NewTypeCheckResult 创建新的类型检查结果
func NewTypeCheckResult() *TypeCheckResult {
	return &TypeCheckResult{
		isValid:    true,
		errors:     make([]*ParseError, 0),
		warnings:   make([]*ParseWarning, 0),
		typeErrors: make([]*TypeError, 0),
		checkedAt:  time.Now(),
	}
}

// IsValid 获取检查结果
func (tcr *TypeCheckResult) IsValid() bool {
	return tcr.isValid
}

// AddTypeError 添加类型错误
func (tcr *TypeCheckResult) AddTypeError(typeError *TypeError) {
	tcr.typeErrors = append(tcr.typeErrors, typeError)
	tcr.isValid = false
}

// AddError 添加一般错误
func (tcr *TypeCheckResult) AddError(err *ParseError) {
	tcr.errors = append(tcr.errors, err)
	tcr.isValid = false
}

// AddWarning 添加警告
func (tcr *TypeCheckResult) AddWarning(warning *ParseWarning) {
	tcr.warnings = append(tcr.warnings, warning)
}

// TypeErrors 获取类型错误列表
func (tcr *TypeCheckResult) TypeErrors() []*TypeError {
	return tcr.typeErrors
}

// Errors 获取一般错误列表
func (tcr *TypeCheckResult) Errors() []*ParseError {
	return tcr.errors
}

// Warnings 获取警告列表
func (tcr *TypeCheckResult) Warnings() []*ParseWarning {
	return tcr.warnings
}

// CheckedAt 获取检查时间
func (tcr *TypeCheckResult) CheckedAt() time.Time {
	return tcr.checkedAt
}

// NewTypeError 创建新的类型错误
func NewTypeError(node ASTNode, expectedType, actualType *TypeInfo, message string, location SourceLocation) *TypeError {
	return &TypeError{
		node:         node,
		expectedType: expectedType,
		actualType:   actualType,
		message:      message,
		location:     location,
	}
}

// Node 获取错误节点
func (te *TypeError) Node() ASTNode {
	return te.node
}

// ExpectedType 获取期望类型
func (te *TypeError) ExpectedType() *TypeInfo {
	return te.expectedType
}

// ActualType 获取实际类型
func (te *TypeError) ActualType() *TypeInfo {
	return te.actualType
}

// Message 获取错误消息
func (te *TypeError) Message() string {
	return te.message
}

// Location 获取错误位置
func (te *TypeError) Location() SourceLocation {
	return te.location
}

// ResolvedSymbols 解析后的符号值对象
type ResolvedSymbols struct {
	symbolTable     *SymbolTable
	resolvedSymbols map[string]*ResolvedSymbol
	unresolvedSymbols []string
	resolvedAt      time.Time
}

// ResolvedSymbol 解析后的符号
type ResolvedSymbol struct {
	originalSymbol *Symbol
	definition     ASTNode
	references     []ASTNode
	isUsed         bool
}

// NewResolvedSymbols 创建新的解析符号结果
func NewResolvedSymbols(symbolTable *SymbolTable) *ResolvedSymbols {
	return &ResolvedSymbols{
		symbolTable:      symbolTable,
		resolvedSymbols:  make(map[string]*ResolvedSymbol),
		unresolvedSymbols: make([]string, 0),
		resolvedAt:       time.Now(),
	}
}

// AddResolvedSymbol 添加解析后的符号
func (rs *ResolvedSymbols) AddResolvedSymbol(resolvedSymbol *ResolvedSymbol) {
	key := resolvedSymbol.originalSymbol.Name()
	rs.resolvedSymbols[key] = resolvedSymbol
}

// AddUnresolvedSymbol 添加未解析的符号
func (rs *ResolvedSymbols) AddUnresolvedSymbol(symbolName string) {
	rs.unresolvedSymbols = append(rs.unresolvedSymbols, symbolName)
}

// GetResolvedSymbol 获取解析后的符号
func (rs *ResolvedSymbols) GetResolvedSymbol(name string) (*ResolvedSymbol, bool) {
	symbol, exists := rs.resolvedSymbols[name]
	return symbol, exists
}

// AllResolvedSymbols 获取所有解析后的符号
func (rs *ResolvedSymbols) AllResolvedSymbols() map[string]*ResolvedSymbol {
	return rs.resolvedSymbols
}

// UnresolvedSymbols 获取未解析的符号列表
func (rs *ResolvedSymbols) UnresolvedSymbols() []string {
	return rs.unresolvedSymbols
}

// HasUnresolvedSymbols 检查是否有未解析的符号
func (rs *ResolvedSymbols) HasUnresolvedSymbols() bool {
	return len(rs.unresolvedSymbols) > 0
}

// ResolvedAt 获取解析时间
func (rs *ResolvedSymbols) ResolvedAt() time.Time {
	return rs.resolvedAt
}

// NewResolvedSymbol 创建新的解析符号
func NewResolvedSymbol(originalSymbol *Symbol, definition ASTNode) *ResolvedSymbol {
	return &ResolvedSymbol{
		originalSymbol: originalSymbol,
		definition:     definition,
		references:     make([]ASTNode, 0),
		isUsed:         false,
	}
}

// AddReference 添加符号引用
func (rs *ResolvedSymbol) AddReference(reference ASTNode) {
	rs.references = append(rs.references, reference)
	rs.isUsed = true
}

// OriginalSymbol 获取原始符号
func (rs *ResolvedSymbol) OriginalSymbol() *Symbol {
	return rs.originalSymbol
}

// Definition 获取符号定义
func (rs *ResolvedSymbol) Definition() ASTNode {
	return rs.definition
}

// References 获取符号引用列表
func (rs *ResolvedSymbol) References() []ASTNode {
	return rs.references
}

// IsUsed 检查符号是否被使用
func (rs *ResolvedSymbol) IsUsed() bool {
	return rs.isUsed
}

// SemanticAnalysisResult 值对象，表示语义分析的整体结果
type SemanticAnalysisResult struct {
	isValid          bool
	errors           []*ParseError
	warnings         []*ParseWarning
	symbolTable      *SymbolTable
	typeCheckResult  *TypeCheckResult
	resolvedSymbols  *ResolvedSymbols
	analyzedAt       time.Time
}

// NewSemanticAnalysisResult 创建新的SemanticAnalysisResult
func NewSemanticAnalysisResult(symbolTable *SymbolTable) *SemanticAnalysisResult {
	return &SemanticAnalysisResult{
		isValid:         true,
		errors:          make([]*ParseError, 0),
		warnings:        make([]*ParseWarning, 0),
		symbolTable:     symbolTable,
		typeCheckResult: NewTypeCheckResult(),
		resolvedSymbols: NewResolvedSymbols(symbolTable),
		analyzedAt:      time.Now(),
	}
}

// IsValid 检查语义分析结果是否有效
func (sar *SemanticAnalysisResult) IsValid() bool {
	return sar.isValid && len(sar.errors) == 0 && sar.typeCheckResult.IsValid()
}

// AddError 添加一个错误
func (sar *SemanticAnalysisResult) AddError(err *ParseError) {
	sar.errors = append(sar.errors, err)
	sar.isValid = false
}

// AddWarning 添加一个警告
func (sar *SemanticAnalysisResult) AddWarning(warn *ParseWarning) {
	sar.warnings = append(sar.warnings, warn)
}

// Errors 获取所有错误
func (sar *SemanticAnalysisResult) Errors() []*ParseError {
	return sar.errors
}

// Warnings 获取所有警告
func (sar *SemanticAnalysisResult) Warnings() []*ParseWarning {
	return sar.warnings
}

// SymbolTable 获取符号表
func (sar *SemanticAnalysisResult) SymbolTable() *SymbolTable {
	return sar.symbolTable
}

// TypeCheckResult 获取类型检查结果
func (sar *SemanticAnalysisResult) TypeCheckResult() *TypeCheckResult {
	return sar.typeCheckResult
}

// ResolvedSymbols 获取符号解析结果
func (sar *SemanticAnalysisResult) ResolvedSymbols() *ResolvedSymbols {
	return sar.resolvedSymbols
}
