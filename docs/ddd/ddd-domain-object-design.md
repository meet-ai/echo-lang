# DDD 领域对象设计：前端模块

## 🎯 设计目标

基于DDD领域建模规则，将复杂的parser.go拆分为清晰的领域对象，实现：
- **职责分离**：每个对象职责单一
- **业务表达**：代码像业务文档一样可读
- **可测试性**：对象独立可测
- **可扩展性**：易于添加新语法元素

## 📋 聚合根设计

### 1. SourceFile 聚合根
**业务含义**：代表一个源文件的完整编译过程，从源码到可执行代码的完整生命周期。

```go
// SourceFile 源文件聚合根
type SourceFile struct {
    id          string           // 聚合根ID
    path        string           // 文件路径
    content     string           // 源代码内容
    tokens      []Token         // 词法分析结果
    ast         *Program        // 语法分析结果
    symbols     SymbolTable     // 语义分析结果
    diagnostics []Diagnostic    // 分析过程中的诊断信息
    status      AnalysisStatus  // 分析状态
    version     int            // 版本号，用于并发控制
}

// 业务行为
func (sf *SourceFile) Tokenize(tokenizer Tokenizer) error {
    tokens, err := tokenizer.Tokenize(SourceCode(sf.content))
    if err != nil {
        sf.addDiagnostic(Diagnostic{Type: Error, Message: err.Error()})
        return err
    }
    sf.tokens = tokens
    sf.status = LexicallyAnalyzed
    sf.version++
    return nil
}

func (sf *SourceFile) Parse(parser SyntaxParser) error {
    if sf.status < LexicallyAnalyzed {
        return errors.New("must tokenize before parsing")
    }

    program, err := parser.Parse(sf.tokens)
    if err != nil {
        sf.addDiagnostic(Diagnostic{Type: Error, Message: err.Error()})
        return err
    }
    sf.ast = program
    sf.status = SyntacticallyAnalyzed
    sf.version++
    return nil
}

func (sf *SourceFile) Analyze(analyzer SemanticAnalyzer) error {
    if sf.status < SyntacticallyAnalyzed {
        return errors.New("must parse before analyzing")
    }

    symbolTable, errors := analyzer.Analyze(sf.ast)
    sf.symbols = symbolTable
    for _, err := range errors {
        sf.addDiagnostic(Diagnostic{Type: Error, Message: err.Error()})
    }

    if len(errors) == 0 {
        sf.status = SemanticallyAnalyzed
    }
    sf.version++
    return nil
}

// 私有方法
func (sf *SourceFile) addDiagnostic(diag Diagnostic) {
    sf.diagnostics = append(sf.diagnostics, diag)
}
```

**设计决策**：
- **状态机管理**：通过status字段确保分析步骤的顺序性
- **诊断收集**：收集分析过程中的所有问题
- **版本控制**：支持并发修改检测

### 2. Program 聚合根
**业务含义**：AST的根节点，代表一个完整的程序结构。

```go
// Program 程序聚合根
type Program struct {
    id         string           // 聚合根ID
    sourceFile *SourceFile      // 所属源文件
    statements []Statement     // 顶级语句列表
    imports    []Import        // 导入声明
    symbols    SymbolTable     // 符号表引用
}

// 业务行为
func (p *Program) AddStatement(stmt Statement) error {
    if err := p.validateStatement(stmt); err != nil {
        return err
    }
    p.statements = append(p.statements, stmt)
    return nil
}

func (p *Program) ResolveSymbols(resolver SymbolResolver) error {
    for _, stmt := range p.statements {
        if err := stmt.ResolveSymbols(p.symbols, resolver); err != nil {
            return err
        }
    }
    return nil
}

// 私有方法
func (p *Program) validateStatement(stmt Statement) error {
    // 验证语句的语义正确性
    return nil
}
```

## 💎 值对象设计

### 1. Token 值对象
**业务含义**：词法单元，不可变的原子语法元素。

```go
// Token 词法单元值对象
type Token struct {
    kind     TokenKind  // 词法类型（关键字、标识符、字面量等）
    value    string     // 词法值
    position Position   // 位置信息
}

// 构造函数
func NewToken(kind TokenKind, value string, pos Position) Token {
    return Token{
        kind:     kind,
        value:    value,
        position: pos,
    }
}

// 值对象方法（不改变状态，返回新对象）
func (t Token) WithValue(newValue string) Token {
    return Token{
        kind:     t.kind,
        value:    newValue,
        position: t.position,
    }
}

// 业务方法
func (t Token) IsKeyword(keyword string) bool {
    return t.kind == Keyword && t.value == keyword
}

func (t Token) IsIdentifier() bool {
    return t.kind == Identifier
}

func (t Token) IsLiteral() bool {
    return t.kind == StringLiteral || t.kind == IntLiteral || t.kind == BoolLiteral
}
```

### 2. Type 值对象
**业务含义**：类型系统中的类型定义。

```go
// Type 类型值对象
type Type struct {
    kind       TypeKind      // 类型种类
    name       string        // 类型名称
    params     []Type       // 泛型参数
    fields     []Field      // 结构体字段
    methods    []Method     // 方法列表
    returnType *Type        // 返回类型（函数类型）
}

// 构造函数
func PrimitiveType(name string) Type {
    return Type{kind: Primitive, name: name}
}

func GenericType(name string, params []Type) Type {
    return Type{kind: Generic, name: name, params: params}
}

func StructType(name string, fields []Field) Type {
    return Type{kind: Struct, name: name, fields: fields}
}

// 值对象方法
func (t Type) IsPrimitive() bool {
    return t.kind == Primitive
}

func (t Type) IsGeneric() bool {
    return t.kind == Generic
}

func (t Type) Instantiate(params []Type) (Type, error) {
    if !t.IsGeneric() {
        return Type{}, errors.New("not a generic type")
    }
    // 泛型实例化逻辑
    return Type{
        kind:   Instantiated,
        name:   t.name,
        params: params,
    }, nil
}
```

### 3. Position 值对象
**业务含义**：源码中的位置信息，用于错误报告和调试。

```go
// Position 位置信息值对象
type Position struct {
    line   int    // 行号（1-based）
    column int    // 列号（1-based）
    file   string // 文件名
}

// 构造函数
func NewPosition(line, column int, file string) Position {
    return Position{line: line, column: column, file: file}
}

// 值对象方法
func (p Position) String() string {
    return fmt.Sprintf("%s:%d:%d", p.file, p.line, p.column)
}

func (p Position) IsBefore(other Position) bool {
    if p.file != other.file {
        return p.file < other.file
    }
    if p.line != other.line {
        return p.line < other.line
    }
    return p.column < other.column
}
```

## 🏗️ 实体设计

### 1. Symbol 实体
**业务含义**：程序中的符号（变量、函数、类型等）。

```go
// Symbol 符号实体
type Symbol struct {
    id       string        // 实体ID
    name     string        // 符号名称
    kind     SymbolKind    // 符号类型
    scope    *Scope        // 所属作用域
    position Position      // 定义位置
    type_    Type         // 符号类型
    mutable  bool         // 是否可变
}

// 业务行为
func (s *Symbol) Rename(newName string) error {
    if !isValidIdentifier(newName) {
        return errors.New("invalid identifier")
    }
    s.name = newName
    return nil
}

func (s *Symbol) ChangeType(newType Type) error {
    if !s.isTypeCompatible(newType) {
        return errors.New("type incompatible")
    }
    s.type_ = newType
    return nil
}

// 私有方法
func (s *Symbol) isTypeCompatible(newType Type) bool {
    // 类型兼容性检查逻辑
    return true
}
```

### 2. Scope 实体
**业务含义**：符号的作用域层次结构。

```go
// Scope 作用域实体
type Scope struct {
    id       string           // 实体ID
    parent   *Scope           // 父作用域
    symbols  map[string]*Symbol // 符号映射
    kind     ScopeKind        // 作用域类型
    position Position         // 作用域位置
}

// 业务行为
func (s *Scope) DefineSymbol(symbol *Symbol) error {
    if s.symbols[symbol.name] != nil {
        return errors.New("symbol already defined")
    }
    s.symbols[symbol.name] = symbol
    symbol.scope = s
    return nil
}

func (s *Scope) LookupSymbol(name string) (*Symbol, error) {
    if symbol, exists := s.symbols[name]; exists {
        return symbol, nil
    }

    if s.parent != nil {
        return s.parent.LookupSymbol(name)
    }

    return nil, errors.New("symbol not found")
}

func (s *Scope) CreateChild(kind ScopeKind, pos Position) *Scope {
    child := &Scope{
        id:       generateID(),
        parent:   s,
        symbols:  make(map[string]*Symbol),
        kind:     kind,
        position: pos,
    }
    return child
}
```

## 🎯 设计原则验证

### 单一职责原则
- **SourceFile**：管理源文件的完整分析生命周期
- **Program**：管理AST的结构和语义
- **Token**：表示词法单元的不可变值
- **Type**：表示类型系统的值对象
- **Symbol**：管理符号的定义和属性
- **Scope**：管理作用域层次和符号查找

### 值对象不可变性
- **Token**：通过WithValue()返回新对象
- **Type**：通过Instantiate()返回新对象
- **Position**：纯数据结构，无修改方法

### 实体标识和行为
- **Symbol**：有ID，支持Rename()、ChangeType()行为
- **Scope**：有ID，支持DefineSymbol()、LookupSymbol()行为

### 聚合边界
- **SourceFile聚合**：包含tokens、ast、symbols，维护分析状态
- **Program聚合**：包含statements、symbols，维护程序结构

## 📈 设计质量评估

### 概念完整性
- ✅ 核心概念都有对应的领域对象
- ✅ 对象职责清晰，无重叠
- ✅ 业务语言贯穿设计

### 技术可行性
- ✅ 值对象不可变，易于测试
- ✅ 实体有明确标识和行为
- ✅ 聚合边界清晰，数据一致性有保障

### 扩展性
- ✅ 新语法元素可通过扩展Type和Token支持
- ✅ 新分析阶段可通过扩展SourceFile状态支持
- ✅ 新符号类型可通过扩展Symbol支持

### 可测试性
- ✅ 值对象可独立测试
- ✅ 实体行为可单元测试
- ✅ 聚合边界便于集成测试
