# T-005-DDD前端模块重构-任务清单

## 概述

基于DDD代码领域建模规则，对echo-lang前端模块进行重构，将复杂的parser.go(2030行，复杂度86)拆分为清晰的领域模型。

**重构目标**：
- 代码复杂度从86降低到<15
- 实现职责分离和单一职责
- 建立完整的DDD分层架构
- 提升代码的可测试性和可维护性

**时间计划**：4周实施
- Week 1：准备阶段，建立测试基线
- Week 2：核心重构，实现领域模型
- Week 3：架构完善，建立分层架构
- Week 4：测试优化，完成验收

## 文件列表与任务分解

### `internal/modules/frontend/domain/analysis/`

#### `token.go` - Token值对象
- [ ] 定义Token结构体（Kind, Value, Position）
- [ ] 实现Token构造函数和工厂方法
- [ ] 实现Token业务方法（IsKeyword, IsIdentifier等）
- [ ] 添加Token的字符串表示方法
- [ ] 编写Token单元测试
- [ ] 实现Token不可变性保证

#### `source_code.go` - SourceCode值对象
- [ ] 定义SourceCode结构体（content, filePath）
- [ ] 实现SourceCode构造函数
- [ ] 实现SourceCode业务方法（LineAt, PositionAt等）
- [ ] 添加SourceCode验证逻辑
- [ ] 编写SourceCode单元测试

#### `position.go` - Position值对象
- [ ] 定义Position结构体（line, column, file）
- [ ] 实现Position构造函数和工厂方法
- [ ] 实现Position比较方法（IsBefore, IsAfter等）
- [ ] 添加Position字符串表示
- [ ] 编写Position单元测试

#### `diagnostic.go` - Diagnostic值对象
- [ ] 定义Diagnostic结构体（type, message, position）
- [ ] 定义DiagnosticType枚举
- [ ] 实现Diagnostic构造函数
- [ ] 编写Diagnostic单元测试

#### `tokenizer.go` - Tokenizer接口和服务
- [ ] 定义Tokenizer接口（Tokenize方法）
- [ ] 实现SimpleTokenizer结构体
- [ ] 迁移词法分析逻辑从parser.go
- [ ] 实现Tokenize方法的核心逻辑
- [ ] 处理词法错误和边界情况
- [ ] 编写Tokenizer单元测试
- [ ] 编写Tokenizer集成测试

#### `syntax_parser.go` - SyntaxParser接口和服务
- [ ] 定义SyntaxParser接口（Parse方法）
- [ ] 实现RecursiveDescentParser结构体
- [ ] 迁移parseStatement相关方法
- [ ] 迁移parseExpression相关方法
- [ ] 实现错误恢复机制
- [ ] 编写SyntaxParser单元测试
- [ ] 编写SyntaxParser集成测试

#### `semantic_analyzer.go` - SemanticAnalyzer接口和服务
- [ ] 定义SemanticAnalyzer接口（Analyze方法）
- [ ] 实现ComprehensiveSemanticAnalyzer结构体
- [ ] 实现符号收集逻辑
- [ ] 实现符号解析逻辑
- [ ] 实现类型检查框架
- [ ] 编写SemanticAnalyzer单元测试

#### `symbol_resolver.go` - SymbolResolver服务
- [ ] 定义SymbolResolver接口
- [ ] 实现DefaultSymbolResolver结构体
- [ ] 实现作用域管理逻辑
- [ ] 实现符号查找算法
- [ ] 编写SymbolResolver单元测试

#### `type_checker.go` - TypeChecker服务
- [ ] 定义TypeChecker接口
- [ ] 实现ComprehensiveTypeChecker结构体
- [ ] 实现表达式类型检查
- [ ] 实现语句类型检查
- [ ] 实现类型兼容性检查
- [ ] 编写TypeChecker单元测试

### `internal/modules/frontend/domain/entities/`

#### `source_file.go` - SourceFile聚合根
- [ ] 定义SourceFile结构体（ID, Path, Content, Tokens, AST, Symbols, Status, Version）
- [ ] 定义AnalysisStatus枚举
- [ ] 实现SourceFile构造函数
- [ ] 实现Tokenize业务方法
- [ ] 实现Parse业务方法
- [ ] 实现Analyze业务方法
- [ ] 实现状态转换逻辑
- [ ] 实现并发控制（版本检查）
- [ ] 编写SourceFile单元测试

#### `program.go` - Program聚合根
- [ ] 定义Program结构体（ID, SourceFile, Statements, Imports, Symbols）
- [ ] 实现Program构造函数
- [ ] 实现AddStatement业务方法
- [ ] 实现ResolveSymbols业务方法
- [ ] 实现语句验证逻辑
- [ ] 编写Program单元测试

#### `symbol.go` - Symbol实体
- [ ] 定义Symbol结构体（ID, Name, Kind, Scope, Position, Type, Mutable）
- [ ] 定义SymbolKind枚举
- [ ] 实现Symbol构造函数
- [ ] 实现Rename业务方法
- [ ] 实现ChangeType业务方法
- [ ] 实现类型兼容性检查
- [ ] 编写Symbol单元测试

#### `scope.go` - Scope实体
- [ ] 定义Scope结构体（ID, Parent, Symbols, Kind, Position）
- [ ] 定义ScopeKind枚举
- [ ] 实现Scope构造函数
- [ ] 实现DefineSymbol业务方法
- [ ] 实现LookupSymbol业务方法
- [ ] 实现CreateChild业务方法
- [ ] 实现作用域链查找逻辑
- [ ] 编写Scope单元测试

### `internal/modules/frontend/domain/repositories/`

#### `source_file_repository.go` - SourceFile仓储接口
- [ ] 定义SourceFileRepository接口
- [ ] 定义仓储方法（Save, FindByID, FindByPath, Delete等）
- [ ] 添加业务查询方法
- [ ] 编写仓储接口测试

#### `symbol_repository.go` - Symbol仓储接口
- [ ] 定义SymbolRepository接口
- [ ] 定义符号查询方法（FindByNameAndScope, FindAllInScope等）
- [ ] 定义符号更新方法
- [ ] 编写仓储接口测试

#### `program_repository.go` - Program仓储接口
- [ ] 定义ProgramRepository接口
- [ ] 定义程序查询和更新方法
- [ ] 编写仓储接口测试

### `internal/modules/frontend/infrastructure/analysis/`

#### `tokenizer_impl.go` - Tokenizer实现
- [ ] 实现SimpleTokenizer的具体逻辑
- [ ] 迁移词法分析算法
- [ ] 优化性能和内存使用
- [ ] 添加详细错误信息
- [ ] 编写实现层测试

#### `parser_impl.go` - Parser实现
- [ ] 实现RecursiveDescentParser的具体逻辑
- [ ] 迁移语法分析算法
- [ ] 处理复杂的语法结构
- [ ] 优化递归深度
- [ ] 编写实现层测试

#### `analyzer_impl.go` - Analyzer实现
- [ ] 实现ComprehensiveSemanticAnalyzer的具体逻辑
- [ ] 实现符号表构建
- [ ] 实现类型推断
- [ ] 优化分析性能
- [ ] 编写实现层测试

### `internal/modules/frontend/infrastructure/persistence/`

#### `repositories/source_file_repository_impl.go` - SourceFile仓储实现
- [ ] 实现SourceFileRepository接口
- [ ] 实现数据映射和转换
- [ ] 处理并发访问
- [ ] 实现缓存策略
- [ ] 编写仓储实现测试

#### `repositories/symbol_repository_impl.go` - Symbol仓储实现
- [ ] 实现SymbolRepository接口
- [ ] 实现高效的符号查询
- [ ] 处理作用域关系
- [ ] 实现索引优化
- [ ] 编写仓储实现测试

#### `repositories/program_repository_impl.go` - Program仓储实现
- [ ] 实现ProgramRepository接口
- [ ] 处理AST序列化
- [ ] 实现增量更新
- [ ] 编写仓储实现测试

### `internal/modules/frontend/infrastructure/di/`

#### `container.go` - 依赖注入容器
- [ ] 定义服务容器接口
- [ ] 实现依赖注入配置
- [ ] 注册所有领域服务
- [ ] 配置仓储实现
- [ ] 编写DI配置测试

#### `providers.go` - 服务提供者
- [ ] 实现领域服务提供者
- [ ] 实现基础设施服务提供者
- [ ] 处理服务生命周期
- [ ] 配置服务依赖关系

### `internal/modules/frontend/application/`

#### `services/frontend_service.go` - 前端应用服务重构
- [ ] 重构FrontendService使用新的领域服务
- [ ] 实现分析流程编排
- [ ] 更新API保持兼容性
- [ ] 添加新的分析功能
- [ ] 编写应用服务测试

#### `dtos/analysis_result.go` - 分析结果DTO
- [ ] 定义AnalysisResult DTO
- [ ] 定义Diagnostic DTO
- [ ] 实现DTO转换逻辑
- [ ] 编写DTO测试

#### `commands/analyze_command.go` - 分析命令
- [ ] 定义AnalyzeCommand结构体
- [ ] 实现命令验证逻辑
- [ ] 编写命令测试

### `internal/modules/frontend/domain/services/`

#### `parser.go` - 原Parser重构
- [ ] 重构SimpleParser使用新的架构
- [ ] 保持向后兼容性
- [ ] 逐步迁移到新架构
- [ ] 移除重复代码
- [ ] 更新现有测试

## 任务清单（总表）

- [ ] 创建领域对象框架（token.go, source_code.go, position.go, diagnostic.go）
- [ ] 实现词法分析服务（tokenizer.go + tokenizer_impl.go）
- [ ] 实现语法分析服务（syntax_parser.go + parser_impl.go）
- [ ] 实现语义分析服务（semantic_analyzer.go + analyzer_impl.go）
- [ ] 实现符号解析服务（symbol_resolver.go）
- [ ] 实现类型检查服务（type_checker.go）
- [ ] 实现聚合根（source_file.go, program.go, symbol.go, scope.go）
- [ ] 实现仓储接口（source_file_repository.go, symbol_repository.go, program_repository.go）
- [ ] 实现仓储具体类（source_file_repository_impl.go, symbol_repository_impl.go, program_repository_impl.go）
- [ ] 配置依赖注入（container.go, providers.go）
- [ ] 重构应用服务（frontend_service.go）
- [ ] 创建DTO和命令对象（analysis_result.go, analyze_command.go）
- [ ] 重构原有Parser（parser.go）
- [ ] 编写完整测试套件
- [ ] 性能优化和代码清理
- [ ] 更新文档和README

## 进度统计

- 完成任务数：0 / 41（0%）
- 已完成文件数：0 / 25

## 更新日志

<!-- 每完成5个任务更新一次 -->

---

## 验收标准

### 代码质量标准
- [ ] **复杂度控制**：所有方法复杂度 < 15
- [ ] **职责单一**：每个类/服务职责明确
- [ ] **测试覆盖**：单元测试覆盖率 > 85%
- [ ] **代码行数**：parser.go从2030行减少到<500行

### 架构质量标准
- [ ] **分层清晰**：应用/领域/基础设施层边界明确
- [ ] **依赖方向**：内层不依赖外层
- [ ] **接口隔离**：依赖抽象而非实现
- [ ] **领域模型完整**：聚合根、实体、值对象、服务、仓储齐全

### 业务功能标准
- [ ] **功能完整**：所有原有功能正常工作
- [ ] **性能不降**：分析速度不低于原有实现
- [ ] **错误处理**：错误信息准确，易于理解
- [ ] **扩展性好**：易于添加新语法元素

## 风险控制

### 技术风险
- **功能回归**：通过完整测试套件控制
- **性能下降**：性能基准测试监控
- **接口不兼容**：保持向后兼容

### 进度风险
- **时间评估偏差**：按周设置里程碑
- **技术难度过高**：预留buffer时间
- **团队学习曲线**：准备DDD培训资料

### 应对策略
- **每日构建**：确保代码始终可编译
- **分支策略**：主分支稳定，重构使用feature分支
- **回滚计划**：每个里程碑都是稳定点
- **质量门禁**：测试覆盖率、复杂度检查
