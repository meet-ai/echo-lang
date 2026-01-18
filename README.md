# Echo Language Compiler

一个用 Go 编写的现代编程语言编译器，将 Echo 语言 (`.eo`) 转换为 OCaml 代码，最终编译为原生可执行文件。

## 🚀 特性

- **现代语法**：Scala风格的类型系统和函数式特性
- **DDD架构**：清晰的分层架构，支持复杂应用开发
- **多目标输出**：生成OCaml代码，支持编译为原生可执行文件
- **智能体友好**：原生支持并发和智能体编程范式

## 📁 项目结构

```
├── cmd/
│   └── main.go           # 应用入口
├── internal/
│   ├── modules/          # 业务模块 (DDD)
│   │   ├── frontend/     # 前端处理 (词法/语法分析)
│   │   └── backend/      # 后端处理 (代码生成)
│   └── infrastructure/   # 基础设施层
├── examples/
│   └── hello.eo         # 示例 Echo 源文件
├── docs/                # 设计文档
├── build/               # 构建输出目录
├── Makefile            # 构建自动化
├── go.mod             # Go 模块文件
└── README.md          # 项目说明
```

## 🛠️ 快速开始

### 使用 Makefile (推荐)

```bash
# 构建编译器
make build

# 编译示例文件
make compile-hello

# 生成可执行程序并运行
make run

# 查看所有可用命令
make help
```

### 手动构建

```bash
# 1. 构建 Go 编译器
go build -o echoc ./cmd/main.go

# 2. 编译 Echo 源文件
./echoc examples/hello.eo

# 3. 如有 OCaml，可编译为原生可执行文件
make compile-ocaml
./build/hello
```

## 📝 Echo 语言语法

目前支持的语法特性：

### 变量声明
- `let name: type = value` - 变量声明和初始化

### 函数定义
- `func name(param: type) -> returnType` - 函数声明
- `func name(param1: type1, param2: type2) -> returnType` - 多参数函数
- `func name() -> returnType { statements }` - 多行函数体

### 函数调用
- `func_name(arg1, arg2)` - 函数调用
- `func_name("string", 42)` - 带字面量参数的调用
- `func_name(var1, var2)` - 带变量参数的调用

### 返回语句
- `return expr` - 从函数返回值
- `return "hello"` - 返回字符串字面量
- `return x + y` - 返回表达式结果

### 条件语句
- `if condition { statements } else { statements }` - 条件执行
- `if x > 0 { print "positive" } else { print "negative" }` - 比较运算
- `if name == "Echo" { print "Hello!" }` - 字符串比较

### 循环语句
- `while condition { statements }` - while循环
- `while x < 10 { print "loop"; x = x + 1 }` - 条件循环
- `for condition { statements }` - for循环
- `for i < 5 { print "loop"; i = i + 1 }` - 基于条件的循环

### 结构体和方法
- `struct Point { x: int, y: int }` - 结构体定义
- `func (p Point) distance(other: Point) -> int { ... }` - 方法定义
- `point.x`, `point.y` - 字段访问

### 比较运算符
- `>` `>=` `<` `<=` - 大小比较
- `==` `!=` - 相等性比较

### 基本类型
- `string` - 字符串类型
- `int` - 整数类型
- `void` - 无返回值类型

### 基本语法
- `print "message"` - 打印字符串
- `// comment` - 单行注释

### 示例代码
```eo
// hello.eo - Echo 语言示例

// 变量声明
let name: string = "Echo"
let age: int = 1

// 打印语句
print "Hello, World!"
print "This is Echo language"

// 简单表达式
let sum: int = 10

// 函数定义
func greet(name: string) -> void

func add(x: int, y: int) -> int

func getMessage() -> string

// 多行函数体
func getMessage2() -> string {
    let greeting: string = "Hello from Echo!"
    return greeting
}
```

### 🚧 开发中特性
以下特性正在开发中，将在后续版本中支持：

- 递归函数：函数自我调用支持
- 条件语句：`if condition { then } else { else }`
- 循环语句：`for condition { body }`
- 二元表达式：`a + b`, `x == y`（更多运算符）
- 更多类型：数组、结构体等

## 🔄 编译输出

编译器生成多种输出格式：

1. **OCaml 代码片段** - 可嵌入现有 OCaml 项目的代码
2. **完整 OCaml 程序** - 可直接编译运行的独立程序
3. **原生可执行文件** - 通过 OCaml 编译器生成 (需要 OCaml 环境)

## 🏗️ 架构特点

### DDD 分层架构
- **Domain 层**：核心业务逻辑 (Parser, Generator)
- **Application 层**：用例编排 (编译流程)
- **Infrastructure 层**：技术实现 (文件I/O, DI)

### 可扩展设计
- 插件化的分析器和生成器
- 支持多种目标语言输出
- 清晰的模块边界，便于功能扩展

## 🛠️ 开发工具

- **Go 1.21+** - 主编程语言
- **OCaml** - 目标代码生成 (可选)
- **Make** - 构建自动化
- **标准库** - 无外部依赖 (核心功能)

## 📚 文档

- [语言特性设计](docs/Echo-Language-Features.md) - 完整语言规范
- [实现计划](docs/roadmap.md) - 开发路线图
- [DDD 设计文档](docs/) - 架构设计文档

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！请先查看 [开发指南](docs/roadmap.md)。
