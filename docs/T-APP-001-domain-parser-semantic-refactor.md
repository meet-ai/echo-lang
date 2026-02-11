## 目标

- **减少 domain 包内的用例编排**：把“先 A → 再 B → 再 C”这类固定流程，从 domain 包迁移到 application 层。
- **保留 domain 的规则能力**：domain 专注于单步规则、判断和 AST/符号表等领域模型操作。

## 涉及模块（首批）

- **frontend parser / semantic**
  - `internal/modules/frontend/domain/parser/parser_coordinator.go`
  - `internal/modules/frontend/domain/parser/parser_aggregate.go`
  - `internal/modules/frontend/domain/semantic/services/semantic_analyzer.go`
- **frontend error recovery**
  - `internal/modules/frontend/domain/error_recovery/services/advanced_error_recovery.go`
  - `internal/modules/frontend/domain/error_recovery/services/error_recovery_service.go`
- **frontend package 管理**
  - `internal/modules/frontend/domain/semantic/services/package_manager.go`
- **application 层新增**
  - `internal/modules/frontend/application/parser/...`（解析用例编排）
  - 后续视情况新增：`.../application/semantic/...`、`.../application/error_recovery/...`

## 重构原则

- **一条流水线 = 一个 Application 用例**
  - 例：源码 → Token → AST → 语义分析 → 结果，这整条流程只在 Application 层出现一次。
- **Domain 只保留单步规则**
  - 例：类型兼容性检查、符号解析、错误分类、恢复策略规则等。
- **Port 在 domain/needs 中定义，实现与调用顺序在 application/infra**

## 易错点 / 边界检查

- **把整条流水线塞进 domain**
  - `SemanticAnalyzer.AnalyzeSemantics` / `ParserAggregate` / `ParserCoordinator` 内部出现 “Tokenize → ParseAST → AnalyzeSemantics” 这类长流程时，应上移到 Application。
- **在 domain 中直接做 IO / 配置读取 / 外部依赖聚合**
  - 这些属于 Application / Infrastructure，domain 只保留纯业务规则和 AST / 符号表等领域模型操作。
- **Application 绕过 domain 直接改内部结构**
  - 建议通过 domain 暴露的规则方法 / Value Object，而不是直接修改符号表、错误集合等内部 map / slice。

## 任务拆解（文件级）

- **解析流水线**
  - 在 `internal/modules/frontend/application/parser/` 下新增 `ParserApplicationService` 实现：
    - `ParseSourceFile`：`LexicalAnalysisService.Tokenize` → `SyntaxAnalysisService.ParseAST` → `SemanticAnalysisService.AnalyzeSemantics`。
    - `ParseWithOptions`：在上一步基础上处理选项（如恢复策略、调试开关等），内部仍调用上述三个步骤。
    - `ValidateSourceFile`：只做 `Tokenize` + `ValidateTokens` / 简化 AST 校验。
  - 逐步从 `parser_coordinator.go`/`parser_aggregate.go` 中抽离“整条解析流程”，调用改为走 `ParserApplicationService`。

- **语义分析流水线**
  - 保留 `SemanticAnalyzer.buildSymbolTable` / `performTypeChecking` / `performSymbolResolution` 作为 domain 规则。
  - 新增 `SemanticAnalysisApplicationService`（位置可选：`.../application/semantic/`）：
    - 对外暴露 `AnalyzeProgram`，内部按固定顺序调用三步。
  - 外部调用从直接使用 `SemanticAnalyzer.AnalyzeSemantics` 迁移到 `SemanticAnalysisApplicationService.AnalyzeProgram`。

- **错误恢复与包管理**
  - 为 `advanced_error_recovery.go` / `error_recovery_service.go` 中的“恢复用例循环（选择策略→执行→记录）”设计 `ErrorRecoveryApplicationService`，domain 只保留策略和规则。
  - 为 `package_manager.go` 中“多路径解析 + 配置加载 + vendor 回退”设计 `PackageResolutionApplicationService`，domain 只保留路径合法性等规则函数。

## 示例：语义分析流水线拆分（简化示意）

- **当前（domain 内既有规则又有编排）**

```go
// SemanticAnalyzer.AnalyzeSemantics (domain)
result := value_objects.NewSemanticAnalysisResult(sa.symbolTableEntity.SymbolTable())
if err := sa.buildSymbolTable(ctx, programAST, result); err != nil {
    return result, err
}
if err := sa.performTypeChecking(ctx, programAST, result); err != nil {
    return result, err
}
if err := sa.performSymbolResolution(ctx, programAST, result); err != nil {
    return result, err
}
return result, nil
```

- **目标形态**

```go
// Application 层：只负责“先做什么再做什么”
type SemanticAnalysisApplicationService struct {
    analyzer *services.SemanticAnalyzer
}

func (s *SemanticAnalysisApplicationService) AnalyzeProgram(
    ctx context.Context,
    ast *value_objects.ProgramAST,
) (*value_objects.SemanticAnalysisResult, error) {
    result := value_objects.NewSemanticAnalysisResult(s.analyzer.SymbolTable())

    if err := s.analyzer.BuildSymbolTable(ctx, ast, result); err != nil {
        return result, err
    }
    if err := s.analyzer.PerformTypeChecking(ctx, ast, result); err != nil {
        return result, err
    }
    if err := s.analyzer.PerformSymbolResolution(ctx, ast, result); err != nil {
        return result, err
    }
    return result, nil
}
```

- **Domain 层：只暴露规则方法**
  - `SemanticAnalyzer.BuildSymbolTable(...)`
  - `SemanticAnalyzer.PerformTypeChecking(...)`
  - `SemanticAnalyzer.PerformSymbolResolution(...)`

## 当前状态与下一步

- **已落地**
  - `SemanticAnalyzer.SymbolTable()`：供 Application 层创建 `SemanticAnalysisResult`。
  - `SemanticAnalysisApplicationService.AnalyzeProgram`：三阶段流水线入口，内部使用 `s.analyzer.SymbolTable()` 创建 result。
  - 端口 `ProgramSemanticAnalyzer`（`ports/services`）：`AnalyzeProgram(ctx, ast) (*SemanticAnalysisResult, error)`，由 `SemanticAnalysisApplicationService` 实现。
  - `ParserApplicationServiceImpl`：依赖 `ProgramSemanticAnalyzer`，`ParseSourceFile` 改为调用 `AnalyzeProgram`，用 `SemanticAnalysisResult` 填充 `ParseResult`（AST、符号表、错误）。
  - `ModernFrontendService`：新增可选依赖 `ParserApplicationService`；若注入非 nil，`ParseSourceCode`/`CompileFile` 走 Application 流水线并产出 `ParseResult`，否则仍用 `ModernParserAggregate`。

- **已落地（本轮）**
  - 在 DI 中装配 `ParserApplicationService`：`providers.ProvideFrontendServices` 注册 LexicalAnalysisService（LexerService）、SyntaxAnalysisService（SyntaxAnalyzer）、ProgramSemanticAnalyzer（SemanticAnalysisApplicationService）、ParserApplicationService，并注册 `frontend.ModernFrontendService`（由 `NewModernFrontendService(nil, parserApp)` 实现）。入口通过 `frontend.Module.ModernFrontendService()` 获取即走 Application 流水线。
  - `frontend.NewModule` 从 injector 解析 `ModernFrontendService` 并暴露 `ModernFrontendService()`；`Validate()` 在 `modernFrontendSvc != nil` 时通过。
  - 新增 `ErrorRecoveryApplicationService`（`internal/modules/frontend/application/error_recovery/`），基于 `ParseContext` 编排一次完整的错误恢复用例：构造 `ParseError` → 调用领域 `ErrorRecoveryService` → 更新 `ParseContext` → 聚合为 `RecoveryResult` 返回；并在 DI 中将其实现注册为 `ports.ErrorRecoveryService`。

- **已落地（错误恢复收口）**
  - 新增端口 `AdvancedErrorRecoveryPort`（`RecoverFromError(ctx, parseError, enhancedTokenStream)`），由 `ErrorRecoveryApplicationService` 通过 `RecoverFromErrorWithEnhancedStream` + 适配器实现。
  - `ParserCoordinator` 构造函数改为 `NewParserCoordinator(advancedRecoveryPort)`；`handleParseError` 优先调用端口，为 nil 时回退到 domain 的 `AdvancedErrorRecoveryService`。
  - `NewModernParserAggregate(advancedRecoveryPort)` 将端口传入协调器；DI 中注册 `AdvancedErrorRecoveryPort`，需走 Application 恢复时从容器取端口并传入聚合根。
- **已落地（包管理 Application 用例）**
  - 新增端口 `PackageResolutionService`（`ResolveImport(ctx, projectRoot, importPath, fromPackage)`），由 `PackageResolutionApplicationService` 实现。
  - Application 层编排 `ValidateImport` → `LoadPackage` 的完整导入用例，调用方可通过该端口加载包信息，而无需直接操作 `PackageManager`。
- **已落地（方案 B：CLI + 转换器，禁止回退）**
  - `ast_to_entities.ConvertProgram(programAST)`：ProgramAST → entities.Program，供 backend 使用；已覆盖常用语句/表达式，UnaryExpression 映射为 BinaryExpr(0 - x) 或 operand。
  - `di.BuildFrontendOnlyContainer()`：仅注册 Frontend，避免依赖未注册的 Backend.Assembler。
  - `parseSourceToProgram(content, filename, useModernPipeline)`：为 true 时**仅**走 ModernFrontendService.ParseSourceCode → ConvertProgram，失败即返回错误（禁止回退 SimpleParser）；为 false 时用 SimpleParser。build/run 已统一经此函数，当前 `useModernPipeline=true`。
  - **LexerService**：单字符分支补全 `advance()` 防死循环；单 `&`/`|`、`@` 已处理；`skipWhitespace` 在换行后若仅空格且下一字符为 `{` 时跳过到 `{`，避免漏产 LeftBrace。
  - **SyntaxAnalyzer**：支持方法语法 `func (receiverName receiverType) methodName (...) -> retType { ... }`，在 `parseFunctionDeclaration` 中识别首 token 为 `(` 时按方法解析并插入接收者参数。

