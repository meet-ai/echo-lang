# DDD 代码生成器领域建模分析

## 📊 领域识别与分析

### 核心业务领域
**代码生成领域**：负责将Echo AST转换为LLVM IR中间表示

**业务目标**：
- 将高级语言抽象语法树转换为低级中间表示
- 生成正确的LLVM IR指令序列
- 维护符号表和类型信息
- 支持多样的控制流和数据结构

### 名词提取（实体/值对象候选）

#### 核心业务概念
- **CodeModule**：代码模块（聚合根）- LLVM模块的业务封装
- **Function**：函数实体 - 用户定义函数的代码表示
- **Variable**：变量实体 - 变量声明和使用的管理
- **Type**：类型实体 - 数据类型的定义和映射
- **Statement**：语句实体 - 代码语句的抽象
- **Expression**：表达式实体 - 计算表达式的抽象

#### 值对象
- **Symbol**：符号 - 变量名、函数名等标识符
- **TypeInfo**：类型信息 - 包含类型名称和LLVM类型映射
- **BlockLabel**：基本块标签 - LLVM基本块的命名
- **Instruction**：指令 - LLVM IR指令的封装

#### 领域事件
- **CodeGenerated**：代码生成完成
- **TypeRegistered**：类型注册完成
- **SymbolDefined**：符号定义完成

### 动词提取（领域服务/行为候选）

#### 核心业务行为
- **GenerateCode**：生成代码 - 将AST转换为IR的主流程
- **MapType**：映射类型 - Echo类型到LLVM类型的转换
- **DeclareSymbol**：声明符号 - 在符号表中注册变量/函数
- **EmitInstruction**：发射指令 - 生成具体的LLVM IR指令
- **ResolveSymbol**：解析符号 - 从符号表查找变量/函数引用

#### 领域服务候选
- **TypeMapper**：类型映射服务 - 处理所有类型转换逻辑
- **SymbolTable**：符号表管理服务 - 管理作用域和符号解析
- **CodeEmitter**：代码发射服务 - 负责最终IR代码生成
- **StatementGenerator**：语句生成服务 - 处理各种语句类型的生成
- **ExpressionGenerator**：表达式生成服务 - 处理各种表达式类型的生成

## 🔍 代码异味识别

### 主要问题分析

#### 1. 上帝类（God Class）问题
**表现**：`LLVMGenerator`类承担了太多职责
- 类型映射（`mapType`）
- 符号管理（`variables` map）
- 代码生成（各种`gen*`方法）
- 模块管理（`module`, `functions`）

**影响**：
- 难以测试和维护
- 职责耦合严重
- 违反单一职责原则

#### 2. 技术耦合问题
**表现**：业务逻辑与LLVM IR技术细节紧密耦合
```go
// 直接操作LLVM IR细节
alloca := g.builder.NewAlloca(varType)
fieldPtr := g.builder.NewGetElementPtr(structType, objPtr, ...)
```

**影响**：
- 难以替换底层技术栈
- 测试依赖具体IR实现
- 业务逻辑和技术逻辑混杂

#### 3. 过程式编程风格
**表现**：大量以`gen`开头的方法，缺乏对象封装
```go
func (g *LLVMGenerator) genBinaryExpr(expr *entities.BinaryExpr) value.Value
func (g *LLVMGenerator) genIfStmt(stmt *entities.IfStmt)
func (g *LLVMGenerator) genFuncDef(stmt *entities.FuncDef)
```

**影响**：
- 难以扩展新语句类型
- 缺乏业务语义封装
- 测试覆盖困难

#### 4. 符号表管理混乱
**表现**：使用`map[string]interface{}`管理符号
```go
variables map[string]interface{} // 变量符号表（支持参数和alloca）
```

**影响**：
- 类型安全缺失
- 符号解析逻辑复杂
- 作用域管理困难

## 🗂️ 限界上下文划分建议

### 建议的上下文结构

```
代码生成领域
├── 类型系统上下文 (Type System Context)
│   ├── 职责：类型定义、类型映射、类型检查
│   └── 聚合根：TypeRegistry
├── 符号管理上下文 (Symbol Management Context)
│   ├── 职责：符号声明、符号解析、作用域管理
│   └── 聚合根：SymbolTable
├── 代码生成上下文 (Code Generation Context)
│   ├── 职责：IR指令生成、控制流管理、数据流管理
│   └── 聚合根：CodeGeneratorContext
└── 模块管理上下文 (Module Management Context)
    ├── 职责：模块组织、依赖管理、代码整合
    └── 聚合根：CodeModule
```

### 上下文映射关系

```
类型系统上下文 ←─── 提供类型支持 ───→ 代码生成上下文
    ↓                                           ↑
    └─── 类型查询 ───→ 符号管理上下文 ←── 符号解析 ───┘
                        ↓
                        └─── 符号信息 ───→ 模块管理上下文
```

## 🏗️ 聚合根与实体设计

### CodeModule（代码模块聚合根）
```go
type CodeModule struct {
    id          string
    name        string
    functions   []*Function
    globalVars  []*Variable
    types       []*TypeDefinition

    // 业务行为
    func AddFunction(fn *Function)
    func AddGlobalVar(v *Variable)
    func AddType(t *TypeDefinition)
    func GenerateIR() string
}
```

### Function（函数实体）
```go
type Function struct {
    name       string
    returnType Type
    params     []*Parameter
    body       []*Statement

    // 业务行为
    func AddParameter(p *Parameter)
    func AddStatement(s *Statement)
    func Validate() error
}
```

### Variable（变量实体）
```go
type Variable struct {
    name  string
    typ   Type
    scope Scope
    value Expression

    // 业务行为
    func AllocateIn(scope Scope)
    func GetValue() Expression
    func SetValue(expr Expression)
}
```

## 🎯 值对象设计

### Symbol（符号值对象）
```go
type Symbol struct {
    name  string  // 不可变
    kind  SymbolKind  // VARIABLE, FUNCTION, TYPE
    scope Scope   // 作用域信息
}

// 业务行为
func (s Symbol) IsDefinedIn(scope Scope) bool
func (s Symbol) GetQualifiedName() string
```

### TypeInfo（类型信息值对象）
```go
type TypeInfo struct {
    name      string     // Echo类型名
    llvmType  types.Type // LLVM类型
    category  TypeCategory // PRIMITIVE, STRUCT, FUNCTION
}

// 业务行为
func (t TypeInfo) IsCompatibleWith(other TypeInfo) bool
func (t TypeInfo) GetSize() int
func (t TypeInfo) IsPointerType() bool
```

## 🔧 领域服务设计

### TypeMapper（类型映射服务）
```go
type TypeMapper interface {
    MapToLLVM(echoType string) (types.Type, error)
    RegisterStruct(name string, fields []FieldDef) error
    IsPrimitiveType(echoType string) bool
}
```

### SymbolTable（符号表服务）
```go
type SymbolTable interface {
    DeclareSymbol(name string, symbol *Symbol) error
    LookupSymbol(name string) (*Symbol, error)
    EnterScope(scopeName string)
    ExitScope()
    GetCurrentScope() Scope
}
```

### CodeEmitter（代码发射服务）
```go
type CodeEmitter interface {
    EmitModule(module *CodeModule) (string, error)
    EmitFunction(fn *Function) (*ir.Func, error)
    EmitInstruction(inst Instruction) error
    EmitGlobalVar(v *Variable) (*ir.Global, error)
}
```

## 📋 重构计划

### 第一阶段：建立领域模型（1周）
1. 定义聚合根和实体
2. 定义值对象和领域服务接口
3. 创建仓储接口

### 第二阶段：实现领域服务（2周）
1. 实现TypeMapper服务
2. 实现SymbolTable服务
3. 实现CodeEmitter服务
4. 实现StatementGenerator和ExpressionGenerator

### 第三阶段：重构现有代码（2周）
1. 将现有方法拆分到相应领域服务
2. 更新LLVMGenerator使用新的领域模型
3. 逐步替换直接的LLVM IR操作

### 第四阶段：测试与优化（1周）
1. 编写领域模型的单元测试
2. 集成测试确保功能正确
3. 性能优化和代码清理

## ✅ 验收标准

### 领域模型质量
- [x] 每个聚合根都有明确的业务含义（CodeModule, FunctionEntity, VariableEntity, TypeEntity）
- [x] 值对象不可变，业务语义明确（BinaryExpressionEntity, FuncCallExpressionEntity, IdentifierEntity）
- [x] 领域服务职责单一，接口清晰（TypeMapper, SymbolTable, CodeEmitter, StatementGenerator, ExpressionGenerator）
- [ ] 仓储接口只定义数据访问契约（暂无独立仓储层，计划后续实现）

### 技术隔离性
- [x] 领域层不直接调用LLVM IR API（通过CodeEmitter接口隔离）
- [x] 领域对象不依赖外部框架（只依赖标准库和内部entities）
- [x] 基础设施实现可轻松替换（通过接口依赖注入）
- [x] 依赖通过接口注入（构造函数注入领域服务）

### 可测试性
- [ ] 单元测试覆盖核心业务逻辑（≥90%）- **待实现**
- [ ] 测试不依赖外部服务（Mock接口）- **待实现**
- [ ] 测试运行速度快（<500ms）- **待实现**
- [ ] 集成测试验证端到端流程- **待实现**

### 业务表达力
- [x] 代码读起来像业务文档（方法名直接反映业务意图）
- [x] 方法名直接反映业务意图（GenerateCode, MapToLLVM, EmitModule等）
- [x] 错误信息使用业务语言（"symbol not found", "unknown type"等）
- [x] 统一语言贯穿整个模型（TypeMapper, CodeEmitter, SymbolTable等）
