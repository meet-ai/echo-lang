# T-PARSE-001：async/spawn/chan 解析方案

## 问题根因

- **run 当前走应用层流水线**：`ParseSourceCode` 因 DI 注入 `ParserApplicationService` 非 nil，使用 **LexerService → SyntaxAnalyzer**。
- **SyntaxAnalyzer 是简化实现**：`parseExpression` 仅支持基础表达式（标识符、字面量、简单二元运算），**未实现** `spawn`、`await`、`<- ch`、`ch <- value` 等。
- **当前兜底**：初值/表达式解析失败时“跳过”到 `;` 或 `}`，AST 中对应节点为空，故能编译运行但异步逻辑未生成。

## 两条解析管线对比

| 管线 | 词法 | 语法 | 表达式 | 结果 |
|------|------|------|--------|------|
| 应用层 | LexerService (shared TokenStream) | SyntaxAnalyzer | 简化，无 spawn/await/chan | 需“跳过”才能过 |
| Modern | AdvancedLexerService (EnhancedTokenStream) | ParserCoordinator → RecursiveDescentParser | 表达式由 parseExpression() 处理，**当前为桩** | 报错 "expression parsing should be delegated to Pratt parser" |

Modern 管线中：`ParseProgram` 使用 **RecursiveDescentParser**，其 `parseExpression()` 为桩实现，直接返回“应委托给 Pratt”错误，未调用 ParserCoordinator 的 Pratt/表达式解析。

## 方案对比

### 方案 A：run 改用 Modern 管线（推荐）

- **做法**：DI 中 `ModernFrontendService` 注入 `parserAppService = nil`，使 `ParseSourceCode` 走 `parserAggregate.Parse()`。
- **前提**：修掉 Modern 管线中“表达式未委托”的问题，否则会报约 95 处 “expression parsing should be delegated to Pratt parser”。
- **优点**：一条管线、语法完整（TokenBasedExpressionParser 已支持 spawn/await/chan），无重复实现。
- **缺点**：必须先修 RecursiveDescentParser 与 ParserCoordinator 的协作。

### 方案 B：在 SyntaxAnalyzer 中补 spawn/await/chan

- **做法**：在 `parseUnaryExpression` / `parsePrimaryExpression` 等中增加对 `spawn`、`await`、`<-`、`chan` 的规则。
- **优点**：应用层管线自洽，不依赖 Modern。
- **缺点**：与 TokenBasedExpressionParser 重复实现，工作量大且易不一致。

### 方案 C：ParseProgram 改用 TokenBasedStatementParser

- **做法**：ParserCoordinator.ParseProgram 不再用 RecursiveDescentParser，改为用 TokenBasedStatementParser 循环 ParseStatement，再把 entities 转成 shared ProgramAST。
- **优点**：直接复用已有完整语句/表达式解析。
- **缺点**：需做 entities.ASTNode ↔ sharedVO.ASTNode 的转换层，且 ProgramAST 为 shared 定义，改动面大。

---

## 推荐：方案 A + 修 Modern 表达式委托

### 根因（Modern 报错）

- `ParserCoordinator.ParseProgram()` 调用 `RecursiveDescentParser.ParseProgram()`。
- 顶层/块内遇到表达式语句时调用 `parseExpressionStatement()` → `rdp.parseExpression()`。
- `RecursiveDescentParser.parseExpression()` 为桩，直接返回 “expression parsing should be delegated to Pratt parser”，**未调用** ParserCoordinator 的 Pratt 解析。

### 任务列表

1. **RecursiveDescentParser 支持“表达式委托”**
   - 文件：`internal/modules/frontend/domain/syntax/services/recursive_descent_parser.go`
   - 新增可选依赖：表达式解析委托（接口或 *ParserCoordinator），例如：
     - `ExpressionParser interface { ParseExpression(ctx) (sharedVO.ASTNode, error) }`
   - 在 `parseExpressionStatement()` 中：若委托不为 nil，调用委托的 `ParseExpression(ctx)`，否则保留原 `parseExpression()` 行为（可继续返回错误）。

2. **ParserCoordinator 注入 RecursiveDescentParser**
   - 文件：`internal/modules/frontend/domain/parser/parser_coordinator.go`
   - 创建 RecursiveDescentParser 时传入“表达式解析委托”：即 ParserCoordinator 自身（或实现上述接口的适配器），使 RecursiveDescent 在解析表达式语句时调用 `pc.ParseExpression(ctx)`。
   - 注意：RecursiveDescent 与 Coordinator 共用同一 `ParsingContext`/TokenStream，当前 token 位置一致，ParseExpression 从当前位读即可。

3. **RecursiveDescentParser 与 sharedVO 节点类型**
   - `ParseExpression` 返回 `sharedVO.ASTNode`，RecursiveDescent 当前构造的也是 shared 的 ExpressionStatement，类型应对齐；若 Pratt 返回的是 entities 侧节点，需在 Coordinator 或适配层做一次 entities → sharedVO 的转换（若现有代码已统一为 sharedVO 则无需此步）。

4. **DI 切换 run 到 Modern**
   - 文件：`internal/infrastructure/di/providers/frontend.go`
   - 将 `ModernFrontendService` 的构造改为传入 `nil` 的 `parserAppService`，使 run 使用 `parserAggregate.Parse()`。

5. **回归**
   - 跑 `./build/echoc run examples/async_test.eo`，确认无 “delegated to Pratt” 类错误且异步逻辑执行正常。
   - 跑现有 demo/测试，确认无回归。

## 已做修复（2026-02-11）

- **parseTypeAnnotation 支持 `[T]` 数组类型**：RecursiveDescentParser 在类型注解中遇到 `[` 时解析为数组类型，返回 typeName `"[]"`、genericArgs `[elem]`；TypeAnnotation.String() 与 typeAnnotationToString 对 `"[]"` 单参输出为 `[T]`。修复后 `examples/arrays_basic_test.eo` 解析通过。

### 实现要点（易错点）

- **共享 TokenStream 位置**：RecursiveDescent 与 Coordinator 必须共用同一 stream；ParseProgram 前 Coordinator 已 Initialize(sourceFile, tokenStream)，RecursiveDescent 被传入的即该 stream，故在 parseExpressionStatement 里调用 `pc.ParseExpression(ctx)` 时，Pratt 从当前 position 读即可，**不要在委托前后随意 reset/seek stream**。委托返回后 RecursiveDescent 的 position 需与 stream 当前位一致（若 parser 自维护 position，须在调用后从 stream 同步回来）。
- **递归与入口**：ParseProgram 的入口仍是 RecursiveDescent，只有“表达式语句”内的表达式走 Coordinator.ParseExpression；其它（如函数体、块）若也调 parseExpression，需统一走同一委托，避免再次落进桩实现。
- **错误与恢复**：ParseExpression 失败时，错误应通过现有 ParseError 机制收集，与当前 “parsing completed with N errors” 行为一致；若启用错误恢复，需保证恢复后 stream 位置一致。

### 可选：保留应用层管线

- 若希望 build/其他入口仍走应用层，可保留 DI 中 `parserAppService` 非 nil，仅对 **run** 使用单独入口或配置（例如通过 run 时指定使用 Modern 前端），这样 run 用 Modern、其他用应用层，两套管线并存直至后续统一。

---

## 小结

| 项目 | 内容 |
|------|------|
| 目标 | run 时正确解析 async/spawn/chan，不再“跳过” |
| 推荐 | 方案 A：run 走 Modern 管线，并修 RecursiveDescent 的表达式委托 |
| 关键修改 | RecursiveDescentParser 注入表达式委托；ParseExpressionStatement 中调用委托；DI 中 run 用 nil parserApp 走 Modern |
| 易错 | 共享 TokenStream 位置、表达式节点类型一致、错误收集与恢复 |
