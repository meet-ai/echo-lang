# UC-004 Try/Option类型设计

## 🎯 目标

实现Echo语言的Try/Option类型系统，提供现代错误处理机制，替代传统的异常处理方式。

## 📋 需求分析

### 核心功能
- ✅ **Result类型**：`Result<T>` 表示可能成功或失败的计算（结果状态）
- ✅ **Option类型**：`Option<T>` 表示可能存在或不存在的值
- ✅ **错误传播操作符**：`?` 操作符自动传播错误
- ✅ **模式匹配**：`Ok(value)`、`Err(error)`、`Some(value)`、`None`

### 实现策略
采用**代数数据类型(ADT)**实现：
- Result<T> = Ok<T> | Err<Error>
- Option<T> = Some<T> | None
- 编译时类型安全，无运行时开销

## 🏗️ 架构设计

### AST扩展

#### 新增节点类型
```go
// TryType 尝试类型节点
type TryType struct {
    Type string // 内部类型，如 "int" -> Try<int>
}

// OptionType 可选类型节点
type OptionType struct {
    Type string // 内部类型，如 "string" -> Option<string>
}

// TryExpr Try表达式节点
type TryExpr struct {
    Expr Expr // 被包装的表达式
}

// OptionExpr Option表达式节点
type OptionExpr struct {
    Expr Expr // 被包装的表达式
}

// SuccessLiteral 成功字面量
type SuccessLiteral struct {
    Value Expr // 成功的值
}

// FailureLiteral 失败字面量
type FailureLiteral struct {
    Error Expr // 错误值
}

// SomeLiteral Some字面量
type SomeLiteral struct {
    Value Expr // 存在的值
}

// NoneLiteral None字面量
type NoneLiteral struct{}

// ErrorPropagation 错误传播操作符
type ErrorPropagation struct {
    Expr Expr // 被传播的表达式
}
```

#### 扩展现有节点
```go
// 扩展FuncCall支持类型参数中的Try/Option
type FuncCall struct {
    Name     string
    TypeArgs []string // 可能包含 "Try<int>", "Option<string>"
    Args     []Expr
}

// 扩展VarDecl支持Try/Option类型注解
type VarDecl struct {
    Name  string
    Type  string // 可能为 "Try<int>", "Option<string>"
    Value Expr
}
```

### 语法设计

#### Result类型
```rust
// Result类型定义
func divide(a: int, b: int) -> Result[int] {
    if b == 0 {
        return Err("Division by zero")
    }
    return Ok(a / b)
}

// Result类型使用
let result: Result[int] = divide(10, 2)
match result {
    Ok(value) => print "Result: " + value
    Err(error) => print "Error: " + error
}
```

#### Option类型
```rust
// Option类型定义
func findUser(id: int) -> Option<User> {
    // 查找用户逻辑
    if found {
        return Some(user)
    }
    return None
}

// Option类型使用
let user: Option<User> = findUser(123)
match user {
    Some(u) => print "Found user: " + u.name
    None => print "User not found"
}
```

#### 错误传播操作符
```rust
// ?操作符自动传播错误
func processData() -> Result[string] {
    let user: Option[User] = findUser(123)?
    let result: Result[int] = divide(10, 0)?
    return Ok("Processed data")
}

// 相当于：
func processData() -> Result[string] {
    let userOpt: Option[User] = findUser(123)
    let user: User = match userOpt {
        Some(u) => u
        None => return Err("User not found")
    }

    let result: Result[int] = divide(10, 0)
    let value: int = match result {
        Ok(v) => v
        Err(e) => return Err(e)
    }

    return Ok("Processed data")
}
```

### 类型系统扩展

#### 类型检查规则
- **Try<T>赋值**：只能接收 `Success<T>` 或 `Failure<Error>`
- **Option<T>赋值**：只能接收 `Some<T>` 或 `None`
- **错误传播**：`expr?` 自动转换为对应的模式匹配

#### 类型推断增强
- 从返回值推断：`let x: Try<int> = func()` → 推断func返回Try<int>
- 从字面量推断：`Success(42)` → 推断为Try<int>

## 🔧 实现步骤

### 阶段1：AST和解析器扩展
1. 定义新的AST节点类型
2. 更新ASTVisitor接口
3. 扩展解析器支持Try/Option语法
4. 添加类型参数解析中的Try/Option

### 阶段2：类型系统实现
1. 实现Try类型的核心逻辑
2. 实现Option类型的核心逻辑
3. 实现错误传播操作符
4. 添加类型检查规则

### 阶段3：代码生成
1. 实现Try/Option到OCaml的转换
2. 生成模式匹配代码
3. 实现错误传播的语法糖

### 阶段4：高级特性
1. 实现复杂的类型推断
2. 支持嵌套的Try/Option
3. 添加便利函数和方法

## 🧪 测试策略

### 单元测试
- Try类型创建和模式匹配
- Option类型创建和模式匹配
- 错误传播操作符的行为
- 类型检查正确性

### 集成测试
- 完整的Try/Option程序编译
- 错误处理流程验证
- 类型推断在复杂场景下的表现

### 边界测试
- 嵌套Try/Option类型
- 错误传播链
- 类型推断冲突情况

## 📊 成功指标

- [ ] 能编译使用Try/Option的"Hello Error Handling"程序
- [ ] 错误传播操作符正常工作
- [ ] 类型检查准确率 > 95%
- [ ] 生成代码正确执行错误处理逻辑

## ⚠️ 风险评估

### 技术风险
- **类型系统复杂度**：Try/Option类型与现有类型系统的集成
- **模式匹配实现**：复杂的模式匹配语法解析
- **错误传播语义**：确保?操作符的语义正确性

### 缓解措施
- 从简单用例开始，逐步增加复杂度
- 借鉴现有函数式语言的实现经验
- 实现渐进式的类型检查

## 🎯 验收标准

### 功能验收
- [ ] Try类型定义和使用
- [ ] Option类型定义和使用
- [ ] 错误传播操作符(?)
- [ ] 完整的模式匹配语法

### 质量验收
- [ ] 编译错误信息清晰
- [ ] 类型检查准确
- [ ] 代码生成正确
- [ ] 运行时行为符合预期

---

## 📝 实现日志

- **2025-01-16**: 创建Try/Option类型设计文档
- **2025-01-16**: 分析代数数据类型实现策略
- **2025-01-16**: 设计错误传播操作符语法
