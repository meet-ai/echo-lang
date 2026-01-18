# DDD 代码领域建模：前端模块重构分析

## 📋 现状分析

### 🔍 代码异味识别

#### 1. 复杂度问题
- **parser.go**: 2030行，复杂度86（Parse方法）
- **高复杂度函数**：
  - `Parse()`: 86复杂度
  - `parseExpr()`: 38复杂度
  - `parseTraitDef()`: 22复杂度
  - `parseStatement()`: 22复杂度

#### 2. 职责混乱
**SimpleParser类承担了过多职责**：
- 词法分析（token识别）
- 语法分析（语句/表达式解析）
- AST构建（各种节点创建）
- 类型系统（泛型、trait解析）
- 模式匹配（match语句解析）

### 🎯 适用DDD重构场景
- ✅ **复杂业务流程**：语法分析流程复杂（>100行函数）
- ✅ **混合业务逻辑和技术实现**：解析逻辑与AST构建混杂
- ✅ **难以测试和维护**：单一巨大类，难以独立测试
- ✅ **需要建立领域模型**：编译器前端领域模型缺失

---

## 📚 核心建模步骤

### 1. 识别领域概念

#### 名词提取（实体/值对象候选）
**核心领域概念**：
- **SourceCode**：源代码（值对象，包含内容和位置信息）
- **Token**：词法单元（值对象，包含类型、值、位置）
- **AST**：抽象语法树（实体，树状结构）
- **Symbol**：符号（实体，变量、函数、类型等的符号信息）
- **Type**：类型（值对象，类型系统）

**语法元素**：
- **Statement**：语句（函数定义、变量声明、控制流等）
- **Expression**：表达式（运算、函数调用、字面量等）
- **Declaration**：声明（函数、结构体、枚举、trait等）
- **Pattern**：模式（匹配模式）

#### 动词提取（领域服务候选）
**解析动作**：
- **Tokenize**：词法分析（源代码 → Token流）
- **Parse**：语法分析（Token流 → AST）
- **Analyze**：语义分析（AST → Symbol Table）
- **Validate**：验证（检查语法/语义正确性）

**具体解析行为**：
- **ParseStatement**：解析语句
- **ParseExpression**：解析表达式
- **ParseDeclaration**：解析声明
- **ParseType**：解析类型
- **ParsePattern**：解析模式

### 2. 划定限界上下文

#### 建议的上下文划分

```
编译器前端领域
├── 词法分析上下文（Lexical Analysis Context）
│   ├── 职责：源代码 → Token流
│   └── 领域对象：SourceCode, Token, Tokenizer
│
├── 语法分析上下文（Syntax Analysis Context）
│   ├── 职责：Token流 → AST
│   └── 领域对象：TokenStream, AST, Parser
│
├── 语义分析上下文（Semantic Analysis Context）
│   ├── 职责：AST → Symbol Table + 类型检查
│   └── 领域对象：SymbolTable, TypeChecker, Analyzer
│
└── 符号管理上下文（Symbol Management Context）
    ├── 职责：符号定义、查找、作用域管理
    └── 领域对象：Symbol, Scope, SymbolTable
```

### 3. 识别聚合根和实体

#### 聚合根设计

**SourceFile聚合根**：
```go
// 源文件聚合根：代表一个源文件的完整分析过程
type SourceFile struct {
    id       string
    path     string
    content  string
    tokens   []Token        // 词法分析结果
    ast      *Program       // 语法分析结果
    symbols  SymbolTable    // 语义分析结果
    status   AnalysisStatus // 分析状态
}
```

**Program聚合根**：
```go
// 程序聚合根：AST的根节点
type Program struct {
    id         string
    statements []Statement  // 顶级语句列表
    symbols    SymbolTable  // 符号表引用
}
```

#### 实体设计

**Symbol实体**：
```go
// 符号实体：变量、函数、类型等的符号信息
type Symbol struct {
    id       string
    name     string
    kind     SymbolKind    // VAR, FUNC, TYPE, etc.
    scope    *Scope        // 作用域
    position Position      // 定义位置
    type_    Type         // 符号类型
}
```

**Scope实体**：
```go
// 作用域实体：符号的作用域层次
type Scope struct {
    id          string
    parent      *Scope         // 父作用域
    symbols     map[string]*Symbol
    kind        ScopeKind      // GLOBAL, FUNCTION, BLOCK
}
```

### 4. 定义值对象

**Token值对象**：
```go
// 词法单元：不可变的值对象
type Token struct {
    kind     TokenKind  // 关键字、标识符、操作符等
    value    string     // 词法值
    position Position   // 位置信息
}
```

**Type值对象**：
```go
// 类型：不可变的值对象
type Type struct {
    kind       TypeKind      // PRIMITIVE, STRUCT, GENERIC, etc.
    name       string        // 类型名称
    params     []Type       // 泛型参数
    fields     []Field      // 结构体字段
}
```

**Position值对象**：
```go
// 位置信息：不可变的值对象
type Position struct {
    line   int  // 行号
    column int  // 列号
    file   string // 文件名
}
```

### 5. 定义领域服务

#### 词法分析服务
```go
// Tokenizer 词法分析器
type Tokenizer interface {
    Tokenize(source SourceCode) ([]Token, error)
}

// SimpleTokenizer 简单词法分析器实现
type SimpleTokenizer struct{}

func (t *SimpleTokenizer) Tokenize(source SourceCode) ([]Token, error) {
    // 词法分析逻辑
}
```

#### 语法分析服务
```go
// SyntaxParser 语法分析器
type SyntaxParser interface {
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
    return p.parseProgram()
}
```

#### 语义分析服务
```go
// SemanticAnalyzer 语义分析器
type SemanticAnalyzer interface {
    Analyze(program *Program) (SymbolTable, []SemanticError)
}

// TypeChecker 类型检查器
type TypeChecker struct {
    symbols SymbolTable
}

func (tc *TypeChecker) Analyze(program *Program) (SymbolTable, []SemanticError) {
    // 语义分析和类型检查逻辑
}
```

### 6. 定义仓储接口

```go
// SourceFileRepository 源文件仓储
type SourceFileRepository interface {
    Save(file *SourceFile) error
    FindByPath(path string) (*SourceFile, error)
    FindByID(id string) (*SourceFile, error)
}

// SymbolRepository 符号仓储
type SymbolRepository interface {
    Save(symbol *Symbol) error
    FindByNameAndScope(name string, scopeID string) (*Symbol, error)
    FindAllInScope(scopeID string) ([]*Symbol, error)
}
```

### 7. 定义领域事件

```go
// SourceFileAnalyzed 源文件分析完成事件
type SourceFileAnalyzed struct {
    SourceFileID string
    AnalysisType AnalysisType // LEXICAL, SYNTAX, SEMANTIC
    Status       AnalysisStatus
    Timestamp    time.Time
}

// SymbolDefined 符号定义事件
type SymbolDefined struct {
    SymbolID   string
    Name       string
    Kind       SymbolKind
    ScopeID    string
    Position   Position
    Timestamp  time.Time
}

// TypeResolved 类型解析事件
type TypeResolved struct {
    SymbolID   string
    Type       Type
    Timestamp  time.Time
}
```

---

## 🛠️ 重构实施路线图

### 第1阶段：领域识别（2天）
- [ ] 提取核心名词：SourceCode, Token, AST, Symbol, Type
- [ ] 识别业务动词：Tokenize, Parse, Analyze, Validate
- [ ] 绘制概念关系图
- [ ] 定义统一语言词典

### 第2阶段：上下文划分（2天）
- [ ] 建立词法分析上下文
- [ ] 建立语法分析上下文
- [ ] 建立语义分析上下文
- [ ] 建立符号管理上下文

### 第3阶段：模型设计（3天）
- [ ] 设计SourceFile和Program聚合根
- [ ] 设计Token、Type、Position值对象
- [ ] 设计Symbol和Scope实体
- [ ] 设计领域服务接口

### 第4阶段：重构实施（5天）

#### 重构顺序（从小到大）
1. **Tokenize模块**（相对独立）
2. **Expression解析**（复杂度38，较独立）
3. **Statement解析**（复杂度22）
4. **Declaration解析**（函数、结构体等）
5. **完整Parse流程**（最后集成）

#### 具体实施步骤
- [ ] 创建领域对象（Token, Type, Position等值对象）
- [ ] 编写单元测试
- [ ] 拆分SimpleParser为多个专用解析器
- [ ] 建立分层架构
- [ ] 渐进式替换

---

## 🎯 验收标准

### 概念清晰度
- [ ] 每个聚合根都有明确的业务含义
- [ ] 值对象不可变，业务语义明确
- [ ] 领域服务职责单一，命名反映业务能力
- [ ] 仓储接口只定义数据访问契约

### 技术隔离性
- [ ] 领域层不直接调用数据库/API
- [ ] 领域对象不依赖外部框架
- [ ] 基础设施实现可轻松替换
- [ ] 依赖通过接口注入

### 可测试性
- [ ] 单元测试覆盖核心业务逻辑（≥90%）
- [ ] 测试不依赖外部服务（Mock接口）
- [ ] 测试运行速度快（<500ms）
- [ ] 集成测试验证端到端流程

### 业务表达力
- [ ] 代码读起来像业务文档
- [ ] 方法名直接反映业务意图
- [ ] 错误信息使用业务语言
- [ ] 统一语言贯穿整个模型

---

## 📈 预期收益

**代码质量提升**：
- parser.go从2030行拆分为多个专用类
- 复杂度从86降低到<15
- 职责分离，易于维护

**测试覆盖改善**：
- 单元测试从类级别提升到方法级别
- 核心算法测试覆盖100%
- 集成测试验证完整流程

**架构演进能力**：
- 支持新语法元素的扩展
- 便于添加类型检查
- 为编译器优化奠定基础
