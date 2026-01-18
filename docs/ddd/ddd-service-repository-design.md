# DDD 领域服务和仓储设计：前端模块

## 🎯 设计目标

将parser.go中的复杂逻辑拆分为职责单一的领域服务，实现：
- **职责分离**：每个服务负责一个具体的分析阶段
- **接口隔离**：通过接口实现依赖倒置
- **可测试性**：服务可独立测试和替换
- **业务表达**：服务命名反映业务意图

## 🛠️ 领域服务设计

### 1. 词法分析服务（Lexical Analysis Service）

**业务职责**：将源代码转换为Token流，实现词法分析。

```go
// Tokenizer 词法分析器接口
type Tokenizer interface {
    // Tokenize 将源代码转换为Token序列
    Tokenize(source SourceCode) ([]Token, error)
}

// SourceCode 值对象：词法分析的输入
type SourceCode struct {
    content  string
    filePath string
}

// SimpleTokenizer 简单词法分析器实现
type SimpleTokenizer struct{}

func (t *SimpleTokenizer) Tokenize(source SourceCode) ([]Token, error) {
    var tokens []Token
    lexer := NewLexer(source.content, source.filePath)

    for {
        token, err := lexer.NextToken()
        if err != nil {
            return nil, err
        }
        if token.Kind == EOF {
            break
        }
        tokens = append(tokens, token)
    }

    return tokens, nil
}

// 私有领域逻辑
func (t *SimpleTokenizer) isKeyword(word string) bool {
    keywords := []string{"func", "if", "else", "for", "while", "struct", "enum", "trait", "return"}
    for _, kw := range keywords {
        if word == kw {
            return true
        }
    }
    return false
}

func (t *SimpleTokenizer) isOperator(char byte) bool {
    operators := []byte{'+', '-', '*', '/', '=', '!', '<', '>', '&', '|', '^'}
    for _, op := range operators {
        if char == op {
            return true
        }
    }
    return false
}
```

**设计决策**：
- **单一职责**：只负责词法分析，不涉及语法分析
- **错误处理**：词法错误立即返回，不继续分析
- **可扩展性**：易于添加新Token类型

### 2. 语法分析服务（Syntax Analysis Service）

**业务职责**：将Token流转换为AST，实现语法分析。

```go
// SyntaxParser 语法分析器接口
type SyntaxParser interface {
    // Parse 将Token序列解析为AST
    Parse(tokens []Token) (*Program, error)
}

// RecursiveDescentParser 递归下降解析器
type RecursiveDescentParser struct {
    tokens []Token
    pos    int
}

func (p *RecursiveDescentParser) Parse(tokens []Token) (*Program, error) {
    p.tokens = tokens
    p.pos = 0

    program := &Program{}
    for !p.isAtEnd() {
        stmt, err := p.parseStatement()
        if err != nil {
            return nil, err
        }
        program.AddStatement(stmt)
    }

    return program, nil
}

// 核心解析方法
func (p *RecursiveDescentParser) parseStatement() (Statement, error) {
    if p.match(Keyword, "func") {
        return p.parseFunctionDeclaration()
    }
    if p.match(Keyword, "let") {
        return p.parseVariableDeclaration()
    }
    if p.match(Keyword, "if") {
        return p.parseIfStatement()
    }
    // ... 其他语句类型

    return nil, p.error("Expected statement")
}

// 辅助方法
func (p *RecursiveDescentParser) match(kind TokenKind, value string) bool {
    if p.check(kind, value) {
        p.advance()
        return true
    }
    return false
}

func (p *RecursiveDescentParser) check(kind TokenKind, value string) bool {
    if p.isAtEnd() {
        return false
    }
    token := p.peek()
    return token.Kind == kind && token.Value == value
}

func (p *RecursiveDescentParser) advance() Token {
    if !p.isAtEnd() {
        p.pos++
    }
    return p.previous()
}

func (p *RecursiveDescentParser) isAtEnd() bool {
    return p.pos >= len(p.tokens)
}

func (p *RecursiveDescentParser) peek() Token {
    return p.tokens[p.pos]
}

func (p *RecursiveDescentParser) previous() Token {
    return p.tokens[p.pos-1]
}
```

**设计决策**：
- **递归下降**：适合手写解析器，易于理解和维护
- **错误恢复**：遇到错误立即停止，不尝试继续解析
- **组合模式**：解析器组合多个子解析器

### 3. 语义分析服务（Semantic Analysis Service）

**业务职责**：对AST进行语义分析，建立符号表，进行类型检查。

```go
// SemanticAnalyzer 语义分析器接口
type SemanticAnalyzer interface {
    // Analyze 对AST进行语义分析
    Analyze(program *Program) (SymbolTable, []SemanticError)
}

// ComprehensiveSemanticAnalyzer 综合语义分析器
type ComprehensiveSemanticAnalyzer struct {
    symbolResolver SymbolResolver
    typeChecker    TypeChecker
}

func (sa *ComprehensiveSemanticAnalyzer) Analyze(program *Program) (SymbolTable, []SemanticError) {
    var errors []SemanticError

    // 第一遍：收集符号定义
    symbolCollector := NewSymbolCollector()
    symbolTable, collectErrors := symbolCollector.Collect(program)
    errors = append(errors, collectErrors...)

    // 第二遍：解析符号引用
    resolver := NewSymbolResolver(symbolTable)
    resolveErrors := resolver.Resolve(program)
    errors = append(errors, resolveErrors...)

    // 第三遍：类型检查
    typeChecker := NewTypeChecker(symbolTable)
    typeErrors := typeChecker.Check(program)
    errors = append(errors, typeErrors...)

    return symbolTable, errors
}

// SymbolCollector 符号收集器
type SymbolCollector struct {
    symbolTable SymbolTable
    scopeStack  []*Scope
}

func (sc *SymbolCollector) Collect(program *Program) (SymbolTable, []SemanticError) {
    sc.symbolTable = NewSymbolTable()
    sc.scopeStack = []*Scope{sc.symbolTable.GlobalScope()}

    var errors []SemanticError
    for _, stmt := range program.Statements {
        if err := stmt.Accept(sc); err != nil {
            errors = append(errors, SemanticError{Message: err.Error()})
        }
    }

    return sc.symbolTable, errors
}

// 访问者模式：收集不同类型的符号
func (sc *SymbolCollector) VisitFunctionDeclaration(fd *FunctionDeclaration) error {
    symbol := &Symbol{
        Name:     fd.Name,
        Kind:     FunctionSymbol,
        Type:     fd.Signature,
        Position: fd.Position,
    }
    return sc.currentScope().DefineSymbol(symbol)
}

func (sc *SymbolCollector) VisitVariableDeclaration(vd *VariableDeclaration) error {
    symbol := &Symbol{
        Name:     vd.Name,
        Kind:     VariableSymbol,
        Type:     vd.Type,
        Position: vd.Position,
    }
    return sc.currentScope().DefineSymbol(symbol)
}
```

**设计决策**：
- **多遍分析**：分阶段进行符号收集、引用解析、类型检查
- **访问者模式**：解耦分析逻辑和AST节点类型
- **错误收集**：不因单个错误停止分析，收集所有错误

### 4. 符号解析服务（Symbol Resolution Service）

**业务职责**：提供符号查找和解析功能，支持作用域管理。

```go
// SymbolResolver 符号解析器接口
type SymbolResolver interface {
    // ResolveSymbol 在作用域链中查找符号
    ResolveSymbol(name string, scope *Scope) (*Symbol, error)

    // CreateScope 创建新的作用域
    CreateScope(parent *Scope, kind ScopeKind) *Scope

    // EnterScope 进入作用域
    EnterScope(scope *Scope)

    // ExitScope 退出作用域
    ExitScope()
}

// DefaultSymbolResolver 默认符号解析器
type DefaultSymbolResolver struct {
    currentScope *Scope
    scopeStack   []*Scope
}

func (sr *DefaultSymbolResolver) ResolveSymbol(name string, scope *Scope) (*Symbol, error) {
    current := scope
    for current != nil {
        if symbol, exists := current.Symbols[name]; exists {
            return symbol, nil
        }
        current = current.Parent
    }
    return nil, fmt.Errorf("undefined symbol: %s", name)
}

func (sr *DefaultSymbolResolver) CreateScope(parent *Scope, kind ScopeKind) *Scope {
    return &Scope{
        ID:      generateID(),
        Parent:  parent,
        Symbols: make(map[string]*Symbol),
        Kind:    kind,
    }
}

func (sr *DefaultSymbolResolver) EnterScope(scope *Scope) {
    sr.scopeStack = append(sr.scopeStack, sr.currentScope)
    sr.currentScope = scope
}

func (sr *DefaultSymbolResolver) ExitScope() {
    if len(sr.scopeStack) > 0 {
        sr.currentScope = sr.scopeStack[len(sr.scopeStack)-1]
        sr.scopeStack = sr.scopeStack[:len(sr.scopeStack)-1]
    }
}
```

### 5. 类型检查服务（Type Checking Service）

**业务职责**：验证表达式的类型正确性，确保类型安全。

```go
// TypeChecker 类型检查器接口
type TypeChecker interface {
    // Check 对程序进行类型检查
    Check(program *Program) []TypeError

    // CheckExpression 检查表达式的类型
    CheckExpression(expr Expression, symbolTable SymbolTable) (Type, error)

    // CheckStatement 检查语句的类型
    CheckStatement(stmt Statement, symbolTable SymbolTable) error
}

// ComprehensiveTypeChecker 综合类型检查器
type ComprehensiveTypeChecker struct {
    symbolTable SymbolTable
}

func (tc *ComprehensiveTypeChecker) Check(program *Program) []TypeError {
    var errors []TypeError

    for _, stmt := range program.Statements {
        if err := tc.CheckStatement(stmt, tc.symbolTable); err != nil {
            errors = append(errors, TypeError{Message: err.Error()})
        }
    }

    return errors
}

func (tc *ComprehensiveTypeChecker) CheckExpression(expr Expression, symbolTable SymbolTable) (Type, error) {
    switch e := expr.(type) {
    case *BinaryExpression:
        return tc.checkBinaryExpression(e, symbolTable)
    case *FunctionCall:
        return tc.checkFunctionCall(e, symbolTable)
    case *Identifier:
        return tc.checkIdentifier(e, symbolTable)
    case *Literal:
        return tc.checkLiteral(e, symbolTable)
    default:
        return Type{}, fmt.Errorf("unknown expression type: %T", expr)
    }
}

func (tc *ComprehensiveTypeChecker) checkBinaryExpression(expr *BinaryExpression, symbolTable SymbolTable) (Type, error) {
    leftType, err := tc.CheckExpression(expr.Left, symbolTable)
    if err != nil {
        return Type{}, err
    }

    rightType, err := tc.CheckExpression(expr.Right, symbolTable)
    if err != nil {
        return Type{}, err
    }

    // 类型兼容性检查
    if !tc.areTypesCompatible(leftType, rightType, expr.Operator) {
        return Type{}, fmt.Errorf("incompatible types for operator %s: %s and %s",
            expr.Operator, leftType.Name, rightType.Name)
    }

    // 返回结果类型
    return tc.getResultType(leftType, rightType, expr.Operator), nil
}

func (tc *ComprehensiveTypeChecker) areTypesCompatible(left, right Type, operator string) bool {
    // 算术运算符
    if operator == "+" || operator == "-" || operator == "*" || operator == "/" {
        return (left.IsNumeric() && right.IsNumeric()) ||
               (left.Name == "string" && right.Name == "string" && operator == "+")
    }

    // 比较运算符
    if operator == "==" || operator == "!=" || operator == "<" || operator == ">" {
        return left.Equals(right) || (left.IsNumeric() && right.IsNumeric())
    }

    return false
}
```

## 🗄️ 仓储接口设计

### 1. 源文件仓储（SourceFile Repository）

**业务职责**：管理源文件的持久化，提供按路径和ID查找功能。

```go
// SourceFileRepository 源文件仓储接口
type SourceFileRepository interface {
    // Save 保存源文件
    Save(file *SourceFile) error

    // FindByID 按ID查找源文件
    FindByID(id string) (*SourceFile, error)

    // FindByPath 按路径查找源文件
    FindByPath(path string) (*SourceFile, error)

    // FindAll 获取所有源文件
    FindAll() ([]*SourceFile, error)

    // Delete 删除源文件
    Delete(id string) error

    // Exists 检查源文件是否存在
    Exists(id string) bool
}
```

**设计决策**：
- **按聚合根设计**：围绕SourceFile聚合根提供操作
- **查询方法丰富**：支持按ID、路径等多种查询方式
- **业务语义**：方法名反映业务意图

### 2. 符号仓储（Symbol Repository）

**业务职责**：管理符号的持久化，支持符号的定义和查找。

```go
// SymbolRepository 符号仓储接口
type SymbolRepository interface {
    // Save 保存符号
    Save(symbol *Symbol) error

    // FindByID 按ID查找符号
    FindByID(id string) (*Symbol, error)

    // FindByNameAndScope 在指定作用域中按名称查找符号
    FindByNameAndScope(name string, scopeID string) (*Symbol, error)

    // FindAllInScope 查找作用域中的所有符号
    FindAllInScope(scopeID string) ([]*Symbol, error)

    // FindAllInFile 查找文件中所有符号
    FindAllInFile(fileID string) ([]*Symbol, error)

    // UpdateType 更新符号类型
    UpdateType(symbolID string, newType Type) error

    // Delete 删除符号
    Delete(id string) error
}
```

**设计决策**：
- **作用域感知**：支持按作用域查询符号
- **类型更新**：支持符号类型的动态更新
- **批量查询**：支持作用域和文件的批量查询

### 3. 程序仓储（Program Repository）

**业务职责**：管理AST的持久化，支持程序结构的存储和查询。

```go
// ProgramRepository 程序仓储接口
type ProgramRepository interface {
    // Save 保存程序AST
    Save(program *Program) error

    // FindByID 按ID查找程序
    FindByID(id string) (*Program, error)

    // FindBySourceFile 查找源文件的程序
    FindBySourceFile(sourceFileID string) (*Program, error)

    // UpdateStatements 更新程序语句
    UpdateStatements(programID string, statements []Statement) error

    // Delete 删除程序
    Delete(id string) error
}
```

## 🎯 服务协作设计

### 应用服务编排

```go
// FrontendAnalysisService 应用服务：编排分析流程
type FrontendAnalysisService struct {
    tokenizer        Tokenizer
    syntaxParser     SyntaxParser
    semanticAnalyzer SemanticAnalyzer
    sourceFileRepo   SourceFileRepository
}

func (s *FrontendAnalysisService) AnalyzeSourceFile(filePath string) (*AnalysisResult, error) {
    // 1. 加载或创建源文件
    sourceFile, err := s.sourceFileRepo.FindByPath(filePath)
    if err != nil {
        // 创建新源文件
        content, err := ioutil.ReadFile(filePath)
        if err != nil {
            return nil, err
        }
        sourceFile = &SourceFile{
            ID:      generateID(),
            Path:    filePath,
            Content: string(content),
            Status:  Created,
        }
    }

    // 2. 词法分析
    if err := sourceFile.Tokenize(s.tokenizer); err != nil {
        return nil, err
    }

    // 3. 语法分析
    if err := sourceFile.Parse(s.syntaxParser); err != nil {
        return nil, err
    }

    // 4. 语义分析
    if err := sourceFile.Analyze(s.semanticAnalyzer); err != nil {
        return nil, err
    }

    // 5. 保存结果
    if err := s.sourceFileRepo.Save(sourceFile); err != nil {
        return nil, err
    }

    return &AnalysisResult{
        SourceFile: sourceFile,
        Success:    len(sourceFile.Diagnostics) == 0,
    }, nil
}
```

### 依赖注入配置

```go
// DI配置：连接所有服务
func BuildFrontendServices() *FrontendServices {
    return &FrontendServices{
        Tokenizer:        &SimpleTokenizer{},
        SyntaxParser:     &RecursiveDescentParser{},
        SemanticAnalyzer: &ComprehensiveSemanticAnalyzer{
            SymbolResolver: &DefaultSymbolResolver{},
            TypeChecker:    &ComprehensiveTypeChecker{},
        },
        SourceFileRepo:   &InMemorySourceFileRepository{},
        SymbolRepo:       &InMemorySymbolRepository{},
        ProgramRepo:      &InMemoryProgramRepository{},
    }
}
```

## 📈 设计质量评估

### 职责分离
- ✅ **Tokenizer**：只负责词法分析
- ✅ **SyntaxParser**：只负责语法分析
- ✅ **SemanticAnalyzer**：只负责语义分析
- ✅ **TypeChecker**：只负责类型检查
- ✅ **SymbolResolver**：只负责符号解析

### 接口隔离
- ✅ **依赖倒置**：应用层依赖领域服务接口
- ✅ **单一职责**：每个接口职责明确
- ✅ **易于测试**：接口可轻松Mock

### 业务表达力
- ✅ **方法命名**：Tokenize、Parse、Analyze等直接反映业务意图
- ✅ **错误处理**：错误信息使用业务语言
- ✅ **类型安全**：强类型确保编译时检查

### 可扩展性
- ✅ **新分析器**：可轻松添加新的分析阶段
- ✅ **新语法**：通过扩展Parser接口支持
- ✅ **新类型**：通过扩展Type系统支持
