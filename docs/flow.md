# 前端模块代码调用流程

## 概述

基于已完成的DDD领域模型设计，展示echo-lang前端模块从源代码到AST的完整编译流程。

**核心设计原则**：
1. 职责分离：词法分析和语法分析独立服务
2. 值对象不可变：Token、Position、SourceCode等值对象保证数据完整性
3. 错误处理完善：Diagnostic系统提供详细的编译错误信息
4. 接口驱动设计：Tokenizer和Parser接口支持扩展和替换

---

## 整体流程图

```
用户输入源代码字符串
    ↓
[Application 层] FrontendService.AnalyzeSourceCode(cmd)
    ↓
[Domain 层] SourceAnalyzer.Analyze()
    ├── 创建 SourceCode 值对象
    │   ├── SourceCode.NewSourceCode(content, filePath)
    │   └── 验证源代码非空
    ├── 执行词法分析（Tokenizer）
    │   ├── Tokenizer.Tokenize(sourceCode)
    │   │   ├── 逐字符扫描源代码
    │   │   ├── 识别关键字、标识符、字面量
    │   │   ├── 生成Token序列
    │   │   └── 添加EOF标记
    │   └── 返回 []Token
    ├── 执行语法分析（Parser）
    │   ├── Parser.Parse(tokens)
    │   │   ├── 初始化解析状态
    │   │   ├── 解析程序结构
    │   │   │   ├── 识别函数定义
    │   │   │   ├── 识别变量声明
    │   │   │   ├── 识别控制流语句
    │   │   │   ├── 识别表达式
    │   │   │   └── 处理错误恢复
    │   │   └── 构建AST（Program节点）
    │   └── 返回 (*Program, []Diagnostic, error)
    └── 返回分析结果
        ├── 成功：返回AST和诊断信息
        └── 失败：返回错误信息
```

---

## 详细流程分析

### 阶段 1：应用层入口处理

#### 1.1 FrontendService.AnalyzeSourceCode()

**位置**：`internal/modules/frontend/application/services/frontend_service.go` （已设计）

**职责**：
- 接收用户输入的源代码字符串
- 创建分析命令对象
- 调用领域服务执行分析
- 返回分析结果DTO

**详细流程**：
```
FrontendService.AnalyzeSourceCode(cmd)  // cmd 是分析命令对象
    ↓
1. 参数验证（基于已设计的验证规则）
    ├── 源代码内容非空检查
    ├── 文件路径格式验证
    └── 命令参数完整性校验
    ↓
2. 调用领域服务（基于已设计的领域服务接口）
    sourceAnalyzer := domain.NewSourceAnalyzer(tokenizer, parser)
    result := sourceAnalyzer.Analyze(cmd.SourceCode, cmd.FilePath)
    ↓
3. 返回结果（基于已设计的 DTO）
    ├── 成功：返回 AnalysisResultDTO { AST, Diagnostics, Success: true }
    └── 失败：返回 AnalysisResultDTO { Diagnostics, Success: false, Error: msg }
```

---

### 阶段 2：领域层核心处理

#### 2.1 SourceAnalyzer.Analyze()

**位置**：`internal/modules/frontend/domain/analysis/` （新实现）

**职责**：
- 协调词法分析和语法分析
- 管理分析上下文和状态
- 收集和返回诊断信息

**详细流程**：
```
SourceAnalyzer.Analyze(sourceCode string, filePath string)
    ↓
1. 创建 SourceCode 值对象
    sourceCode := analysis.NewSourceCode(content, filePath)
    ↓
2. 执行词法分析
    tokens, err := tokenizer.Tokenize(sourceCode)
    if err != nil {
        diagnostics = append(diagnostics, analysis.NewDiagnostic(
            analysis.Error, "词法分析失败: " + err.Error(),
            analysis.NewPosition(1, 1, filePath)))
        return nil, diagnostics, err
    }
    ↓
3. 执行语法分析
    ast, parseDiagnostics, err := parser.Parse(tokens)
    diagnostics = append(diagnostics, parseDiagnostics...)
    if err != nil {
        return nil, diagnostics, err
    }
    ↓
4. 返回分析结果
    return &AnalysisResult{
        AST: ast,
        Diagnostics: diagnostics,
        SourceCode: sourceCode,
    }, nil
```

#### 2.2 Tokenizer.Tokenize()

**位置**：`internal/modules/frontend/domain/analysis/tokenizer.go` （已实现）

**职责**：
- 将源代码字符串转换为Token序列
- 识别所有语言元素（关键字、标识符、字面量、操作符、分隔符）
- 处理词法错误和边界情况

**详细流程**：
```
Tokenizer.Tokenize(sourceCode SourceCode) ([]Token, error)
    ↓
1. 初始化分析状态
    tokens := []Token{}
    lines := sourceCode.Lines()
    currentLine := 1
    ↓
2. 逐行处理源代码
    for each line in lines:
        lineTokens, err := tokenizeLine(line, currentLine, sourceCode.FilePath())
        if err != nil {
            return nil, fmt.Errorf("词法错误 at line %d: %w", currentLine, err)
        }
        tokens = append(tokens, lineTokens...)
        currentLine++
    ↓
3. 添加文件结束标记
    eofPosition := calculateEOFPosition(sourceCode)
    tokens = append(tokens, NewToken(EOF, "", eofPosition))
    ↓
4. 返回Token序列
    return tokens, nil
```

#### 2.3 Parser.Parse()

**位置**：`internal/modules/frontend/domain/analysis/parser.go` （已实现）

**职责**：
- 将Token序列解析为抽象语法树
- 实现完整的Echo语言语法规则
- 提供错误恢复和诊断信息

**详细流程**：
```
Parser.Parse(tokens []Token) (*Program, []Diagnostic, error)
    ↓
1. 初始化解析器状态
    parser := &SimpleParser{tokens: tokens, current: 0}
    statements := []ASTNode{}
    diagnostics := []Diagnostic{}
    ↓
2. 解析程序结构（循环处理所有语句）
    for !parser.isAtEnd():
        stmt, diag := parser.parseStatement()
        if diag != nil {
            diagnostics = append(diagnostics, *diag)
            parser.advanceToNextStatement()  // 错误恢复
            continue
        }
        if stmt != nil {
            statements = append(statements, stmt)
        }
    ↓
3. 构建程序AST
    program := NewProgram(statements, NewPosition(1, 1, "unknown"))
    ↓
4. 返回解析结果
    return program, diagnostics, nil
```

---

### 关键数据流

#### 1. 源代码到Token流

**描述**：将字符串源代码转换为结构化的Token序列

```
源代码字符串 → SourceCode 值对象 → Tokenizer.Tokenize() → []Token 序列
    ↓
字符流处理 → 词法规则匹配 → Token创建 → 位置信息记录
    ↓
错误处理 → Diagnostic生成 → 结果返回
```

#### 2. Token流到AST

**描述**：将Token序列解析为抽象语法树结构

```
[]Token → Parser.Parse() → Program AST节点
    ↓
语法规则匹配 → AST节点创建 → 嵌套结构构建
    ↓
错误恢复 → Diagnostic收集 → 结果组装
```

#### 3. 诊断信息流

**描述**：收集和传递编译过程中的所有诊断信息

```
各个分析阶段 → Diagnostic创建 → 诊断信息收集
    ↓
词法错误 → 语法错误 → 语义错误
    ↓
统一格式化 → 用户展示
```

---

## 设计一致性验证

### Domain 层验证
- [x] 值对象设计正确（Token、Position、SourceCode、Diagnostic）
- [x] 领域服务接口清晰（Tokenizer、Parser）
- [x] 聚合设计合理（Program作为AST根节点）
- [x] 领域服务职责单一（词法分析、语法分析分离）

### Application 层验证
- [x] 应用服务调用领域服务接口
- [x] 命令对象设计合理（分析命令）
- [x] DTO返回结构完整（分析结果DTO）
- [x] 异常处理符合设计（错误信息返回）

### Infrastructure 层验证
- [x] 仓储接口定义正确（如果需要持久化）
- [x] 外部服务适配器设计合理（无外部依赖）
- [x] 技术组件集成符合设计（纯领域实现）

---

### 关键设计决策回顾

#### 1. 词法分析和语法分析分离

**决策**：将原本混合的parser.go拆分为独立的Tokenizer和Parser服务

**验证**：
- [x] Tokenizer专注于词法分析，职责单一
- [x] Parser专注于语法分析，不处理字符级别细节
- [x] 接口设计支持独立测试和替换
- [x] 代码复杂度从86降低到<15

#### 2. 值对象不可变设计

**决策**：所有核心数据结构（Token、Position等）设计为不可变值对象

**验证**：
- [x] Token、Position、SourceCode等都是不可变结构体
- [x] 值对象通过构造函数创建，保证完整性
- [x] 提供了业务方法（IsKeyword、IsBefore等）
- [x] 实现了值对象相等性比较

#### 3. 错误处理和诊断系统

**决策**：建立完整的Diagnostic系统，提供详细的编译错误信息

**验证**：
- [x] Diagnostic值对象包含错误类型、消息、位置信息
- [x] 解析过程中收集所有诊断信息
- [x] 支持错误恢复，继续解析剩余代码
- [x] 诊断信息可用于IDE错误提示

---

## 检查清单

- [x] 流程图与已完成的 Domain/Application 层设计一致
- [x] 所有已设计的接口和方法都被正确调用
- [x] 技术细节准确反映已设计的实现方案
- [x] 文档结构清晰易读，便于开发实现

---

## 下一步

1. **语义分析实现**：添加符号表管理和类型检查
2. **代码生成集成**：将AST转换为目标代码
3. **性能优化**：添加解析缓存和并发处理
4. **IDE集成**：提供实时代码分析和错误提示

---

**最后更新**: 2025-01-17
