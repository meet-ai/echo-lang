// Package value_objects 定义解析结果相关值对象
package value_objects

import "time"

// ParseResult 解析结果值对象
type ParseResult struct {
	sourceFile    *SourceFile
	ast          *ProgramAST
	symbolTable  *SymbolTable
	errors       []*ParseError
	warnings     []*ParseWarning
	metadata     *ParseMetadata
}

// ParseMetadata 解析元数据
type ParseMetadata struct {
	startTime      time.Time
	endTime        time.Time
	tokenCount     int
	nodeCount      int
	parsingTimeMs  int64
	analysisTimeMs int64
}

// NewParseResult 创建新的解析结果
func NewParseResult(sourceFile *SourceFile) *ParseResult {
	return &ParseResult{
		sourceFile: sourceFile,
		errors:     make([]*ParseError, 0),
		warnings:   make([]*ParseWarning, 0),
		metadata: &ParseMetadata{
			startTime: time.Now(),
		},
	}
}

// SetAST 设置AST
func (pr *ParseResult) SetAST(ast *ProgramAST) {
	pr.ast = ast
}

// SetSymbolTable 设置符号表
func (pr *ParseResult) SetSymbolTable(symbolTable *SymbolTable) {
	pr.symbolTable = symbolTable
}

// AddError 添加错误
func (pr *ParseResult) AddError(err *ParseError) {
	pr.errors = append(pr.errors, err)
}

// AddWarning 添加警告
func (pr *ParseResult) AddWarning(warning *ParseWarning) {
	pr.warnings = append(pr.warnings, warning)
}

// Complete 完成解析
func (pr *ParseResult) Complete() {
	pr.metadata.endTime = time.Now()
	duration := pr.metadata.endTime.Sub(pr.metadata.startTime)
	pr.metadata.parsingTimeMs = duration.Milliseconds()
}

// SourceFile 获取源文件
func (pr *ParseResult) SourceFile() *SourceFile {
	return pr.sourceFile
}

// AST 获取AST
func (pr *ParseResult) AST() *ProgramAST {
	return pr.ast
}

// SymbolTable 获取符号表
func (pr *ParseResult) SymbolTable() *SymbolTable {
	return pr.symbolTable
}

// Errors 获取错误列表
func (pr *ParseResult) Errors() []*ParseError {
	return pr.errors
}

// Warnings 获取警告列表
func (pr *ParseResult) Warnings() []*ParseWarning {
	return pr.warnings
}

// HasErrors 检查是否有错误
func (pr *ParseResult) HasErrors() bool {
	return len(pr.errors) > 0
}

// Metadata 获取元数据
func (pr *ParseResult) Metadata() *ParseMetadata {
	return pr.metadata
}

// ValidationResult 验证结果值对象
type ValidationResult struct {
	isValid   bool
	errors    []*ParseError
	warnings  []*ParseWarning
	validatedAt time.Time
}

// NewValidationResult 创建新的验证结果
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		isValid:     true,
		errors:      make([]*ParseError, 0),
		warnings:    make([]*ParseWarning, 0),
		validatedAt: time.Now(),
	}
}

// SetValid 设置验证状态
func (vr *ValidationResult) SetValid(valid bool) {
	vr.isValid = valid
}

// AddError 添加错误
func (vr *ValidationResult) AddError(err *ParseError) {
	vr.errors = append(vr.errors, err)
	vr.isValid = false
}

// AddWarning 添加警告
func (vr *ValidationResult) AddWarning(warning *ParseWarning) {
	vr.warnings = append(vr.warnings, warning)
}

// IsValid 获取验证状态
func (vr *ValidationResult) IsValid() bool {
	return vr.isValid
}

// Errors 获取错误列表
func (vr *ValidationResult) Errors() []*ParseError {
	return vr.errors
}

// Warnings 获取警告列表
func (vr *ValidationResult) Warnings() []*ParseWarning {
	return vr.warnings
}

// ValidatedAt 获取验证时间
func (vr *ValidationResult) ValidatedAt() time.Time {
	return vr.validatedAt
}

// ParseOptions 解析选项值对象
type ParseOptions struct {
	enableOptimization bool
	enableWarnings     bool
	maxErrors          int
	timeoutMs          int
}

// NewParseOptions 创建默认解析选项
func NewParseOptions() *ParseOptions {
	return &ParseOptions{
		enableOptimization: true,
		enableWarnings:     true,
		maxErrors:          100,
		timeoutMs:          30000, // 30秒
	}
}

// EnableOptimization 获取优化选项
func (po *ParseOptions) EnableOptimization() bool {
	return po.enableOptimization
}

// SetEnableOptimization 设置优化选项
func (po *ParseOptions) SetEnableOptimization(enable bool) {
	po.enableOptimization = enable
}

// EnableWarnings 获取警告选项
func (po *ParseOptions) EnableWarnings() bool {
	return po.enableWarnings
}

// SetEnableWarnings 设置警告选项
func (po *ParseOptions) SetEnableWarnings(enable bool) {
	po.enableWarnings = enable
}

// MaxErrors 获取最大错误数
func (po *ParseOptions) MaxErrors() int {
	return po.maxErrors
}

// SetMaxErrors 设置最大错误数
func (po *ParseOptions) SetMaxErrors(max int) {
	po.maxErrors = max
}

// TimeoutMs 获取超时时间
func (po *ParseOptions) TimeoutMs() int {
	return po.timeoutMs
}

// SetTimeoutMs 设置超时时间
func (po *ParseOptions) SetTimeoutMs(timeout int) {
	po.timeoutMs = timeout
}
