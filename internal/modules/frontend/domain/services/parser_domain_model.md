# Echo语言语法解析器领域建模设计

## 🎯 核心目标

将复杂的语法解析流程重构为清晰的领域模型，提升代码的可维护性、可测试性和业务表达力。

## 📋 适用场景

- ✅ 复杂语法解析流程（3000+行代码）
- ✅ 混合语法分析和技术实现的代码
- ✅ 难以测试和维护的遗留解析器
- ✅ 需要建立领域模型的语法解析系统

## 🏗️ 领域模型设计

### 核心聚合根

#### Parser（解析器聚合根）
```go
type Parser struct {
    statementParser  *StatementParser
    expressionParser *ExpressionParser
    blockExtractor   *BlockExtractor
}
```
**职责**：
- 协调整个解析流程
- 管理解析状态
- 确保解析结果的一致性

**不变量**：
- 解析结果必须是有效的AST结构
- 错误信息必须包含准确的位置信息

### 领域服务

#### StatementParser（语句解析服务）
```go
type StatementParser struct {
    expressionParser *ExpressionParser
}
```
**职责**：
- 解析各种类型的语句（if, while, assign, print等）
- 验证语句语法正确性
- 返回相应的AST节点

#### ExpressionParser（表达式解析服务）
```go
type ExpressionParser struct {
    // 无依赖，只处理表达式解析
}
```
**职责**：
- 解析各种类型的表达式（二元表达式、函数调用、字面量等）
- 处理运算符优先级
- 验证表达式语法

#### BlockExtractor（代码块提取服务）
```go
type BlockExtractor struct {
    // 无依赖，只处理代码块提取
}
```
**职责**：
- 从源码行中提取代码块
- 处理嵌套块的边界识别
- 返回块内容和消耗的行数

### 领域实体（复用现有AST）

**Program实体**（复用现有设计）
```go
type Program struct {
    Statements []entities.ASTNode
}
```

**现有AST节点**（保持不变）
- IfStmt, WhileStmt, ForStmt等控制流语句
- FuncCall, MethodCall等调用表达式
- VarDecl, AssignStmt等声明语句
- 各种字面量和表达式类型

### 值对象

#### SourceLocation（源码位置）
```go
type SourceLocation struct {
    Line   int
    Column int
    File   string
}
```

#### ParseResult（解析结果）
```go
type ParseResult struct {
    Success  bool
    Node     entities.ASTNode
    Error    error
    Location SourceLocation
}
```

## 🎯 重构策略

### 核心设计原则
1. **聚合根负责协调**：Parser作为唯一入口，协调各个领域服务
2. **领域服务职责单一**：每个服务只负责一种类型的解析逻辑
3. **值对象不可变**：SourceLocation和ParseResult作为不可变的值对象
4. **逐步迁移**：保持现有功能完整性，逐步替换原有代码

### 重构步骤

#### 步骤1：创建领域服务类
```go
// 新建领域服务文件
type StatementParser struct { ... }
type ExpressionParser struct { ... }
type BlockExtractor struct { ... }
```

#### 步骤2：重构Parser聚合根
```go
// 重构现有SimpleParser
type Parser struct {
    statementParser  *StatementParser
    expressionParser *ExpressionParser
    blockExtractor   *BlockExtractor
}

func (p *Parser) Parse(content string) (*entities.Program, error) {
    // 使用领域服务协作完成解析
}
```

#### 步骤3：迁移现有方法
- 将现有解析方法逐步迁移到对应的领域服务
- 保持方法签名不变，确保现有调用不受影响
- 编写单元测试验证功能正确性

#### 步骤4：优化和清理
- 移除重复代码
- 统一错误处理
- 添加必要的文档和注释

### 预期效果

| 维度         | 重构前           | 重构后              |
| ------------ | ---------------- | ------------------- |
| **代码组织** | 单个3000+行文件  | 4个职责清晰的领域类 |
| **可测试性** | 难以进行单元测试 | 每个服务可独立测试  |
| **可维护性** | 修改影响整个文件 | 领域服务独立演进    |
| **业务表达** | 技术实现细节混杂 | 清晰的领域概念分离  |
