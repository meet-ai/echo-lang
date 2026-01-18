# DDD 重构实施计划：前端模块

## 🎯 重构目标

将 `parser.go` (2030行，复杂度86) 重构为清晰的DDD领域模型，实现：
- **代码复杂度**：从86降低到<15
- **职责分离**：单一职责的服务和对象
- **可测试性**：独立可测的领域逻辑
- **可维护性**：易于理解和修改

## 📋 重构范围

### 核心文件
- `internal/modules/frontend/domain/services/parser.go` (2030行) → 重构目标
- `internal/modules/frontend/domain/entities/ast.go` (780行) → 扩展AST模型

### 影响范围
- 测试文件：`parser_test.go`, `try_option_test.go`
- 应用服务：`frontend_service.go`
- 基础设施：适配器和仓储实现

## 🗓️ 重构时间表

### 第1周：准备阶段（3-4天）
**目标**：建立重构基础，确保安全重构

#### Day 1：建立测试基线
```bash
# 1. 运行现有测试，确保通过
go test ./internal/modules/frontend/... -v

# 2. 记录测试覆盖率
go test -coverprofile=baseline.out ./internal/modules/frontend/...
go tool cover -html=baseline.out -o baseline.html

# 3. 识别关键逻辑（P0优先级）
# - Parse() 方法：复杂度86
# - parseExpr() 方法：复杂度38
# - parseStatement() 方法：复杂度22
```

**验收标准**：
- [ ] 所有测试通过 (0 failures)
- [ ] 测试覆盖率 > 80%
- [ ] 关键逻辑测试覆盖 100%

#### Day 2：创建领域对象框架
```bash
# 创建新的领域对象目录结构
mkdir -p internal/modules/frontend/domain/analysis

# 创建核心值对象
# - domain/analysis/token.go (Token值对象)
# - domain/analysis/source_code.go (SourceCode值对象)
# - domain/analysis/position.go (Position值对象)
# - domain/analysis/diagnostic.go (Diagnostic值对象)
```

**验收标准**：
- [ ] 值对象框架创建完成
- [ ] 基础构造函数和方法实现
- [ ] 单元测试通过

#### Day 3：创建领域服务接口
```go
# 创建领域服务接口
# - domain/analysis/tokenizer.go (Tokenizer接口)
# - domain/analysis/syntax_parser.go (SyntaxParser接口)
# - domain/analysis/semantic_analyzer.go (SemanticAnalyzer接口)
```

**验收标准**：
- [ ] 接口定义完成
- [ ] 接口文档完善
- [ ] 依赖关系清晰

#### Day 4：设置重构环境
```bash
# 1. 创建重构分支
git checkout -b feature/ddd-refactoring

# 2. 设置持续集成
# - 配置pre-commit hooks
# - 设置自动测试
# - 配置覆盖率检查

# 3. 创建重构日志
# docs/refactoring-log.md
```

### 第2周：核心重构（5天）
**目标**：实现DDD领域模型的核心功能

#### Day 5-6：实现词法分析服务
**重构内容**：
```go
// 1. 从parser.go中提取词法分析相关代码
// - 提取Tokenize相关方法
// - 创建SimpleTokenizer实现
// - 建立Token流处理逻辑

// 2. 更新依赖
// - 修改frontend_service.go使用新的Tokenizer
// - 保持现有API兼容
```

**具体步骤**：
1. 创建 `domain/analysis/tokenizer.go`
2. 实现 `SimpleTokenizer` 结构体
3. 从 `parser.go` 迁移词法分析代码
4. 更新应用服务使用新接口
5. 运行测试确保功能正常

**验收标准**：
- [ ] Tokenizer接口实现完成
- [ ] 词法分析功能正常
- [ ] 测试覆盖率维持 > 80%

#### Day 7-8：实现语法分析服务
**重构内容**：
```go
// 1. 拆分语法分析逻辑
// - 创建RecursiveDescentParser
// - 实现parseStatement方法组
// - 实现parseExpression方法组

// 2. 保持接口兼容
// - Parse方法保持相同签名
// - 错误处理逻辑保持一致
```

**具体步骤**：
1. 创建 `domain/analysis/syntax_parser.go`
2. 实现 `RecursiveDescentParser` 结构体
3. 迁移语句解析逻辑（parseStatement, parseVarDecl, parseFuncDef等）
4. 迁移表达式解析逻辑（parseExpr, parseBinaryExpr等）
5. 更新应用服务集成
6. 运行完整测试套件

**验收标准**：
- [ ] 语法分析服务实现完成
- [ ] 所有现有测试通过
- [ ] parser.go代码量减少30%

#### Day 9：实现语义分析服务
**重构内容**：
```go
// 1. 提取语义分析逻辑
// - 创建符号收集逻辑
// - 实现作用域管理
// - 建立类型检查框架

// 2. 设计SymbolTable
// - 实现符号定义和查找
// - 支持作用域嵌套
```

**具体步骤**：
1. 创建 `domain/analysis/semantic_analyzer.go`
2. 实现符号表和作用域管理
3. 迁移语义分析相关代码
4. 创建类型检查基础框架
5. 集成到应用服务

**验收标准**：
- [ ] 语义分析服务实现完成
- [ ] 符号解析功能正常
- [ ] 类型检查框架建立

### 第3周：架构完善（4天）
**目标**：完善DDD架构，确保分层清晰

#### Day 10：实现仓储接口
**重构内容**：
```go
// 1. 创建仓储接口
// - SourceFileRepository
// - SymbolRepository
// - ProgramRepository

// 2. 实现内存版本（用于测试）
// - InMemorySourceFileRepository
// - InMemorySymbolRepository
```

**验收标准**：
- [ ] 仓储接口定义完成
- [ ] 内存实现可用
- [ ] 单元测试覆盖仓储操作

#### Day 11：完善聚合根
**重构内容**：
```go
// 1. 增强SourceFile聚合根
// - 添加分析状态管理
// - 实现业务方法（Tokenize, Parse, Analyze）

// 2. 增强Program聚合根
// - 添加符号表集成
// - 实现语句管理方法
```

**验收标准**：
- [ ] SourceFile聚合根功能完整
- [ ] Program聚合根业务方法实现
- [ ] 聚合边界数据一致性保证

#### Day 12：依赖注入配置
**重构内容**：
```go
// 1. 创建DI容器配置
// - 注册所有领域服务
// - 配置仓储实现
// - 设置服务依赖关系

// 2. 更新应用服务
// - 使用依赖注入
// - 移除直接实例化
```

**验收标准**：
- [ ] DI容器配置完成
- [ ] 服务依赖关系正确
- [ ] 应用服务使用DI注入

### 第4周：测试和优化（4天）
**目标**：确保重构质量，优化性能

#### Day 13：完善测试套件
**重构内容**：
```go
// 1. 为新领域对象编写测试
// - 值对象单元测试
// - 领域服务单元测试
// - 仓储接口测试

// 2. 重构现有测试
// - 更新parser_test.go使用新架构
// - 确保测试覆盖率不下降
```

**验收标准**：
- [ ] 新领域对象测试覆盖100%
- [ ] 整体测试覆盖率 > 85%
- [ ] 所有集成测试通过

#### Day 14：性能优化
**重构内容**：
```go
// 1. 分析性能瓶颈
// - 词法分析性能
// - 语法分析性能
// - 符号解析性能

// 2. 优化实现
// - 使用更高效的数据结构
// - 减少不必要的内存分配
// - 优化算法复杂度
```

**验收标准**：
- [ ] 分析性能提升20%
- [ ] 内存使用优化
- [ ] 无性能回归

#### Day 15：文档和清理
**重构内容**：
```go
// 1. 更新文档
// - 更新README.md
// - 更新架构文档
// - 编写领域模型说明

// 2. 代码清理
// - 移除废弃代码
// - 统一代码风格
// - 添加注释
```

**验收标准**：
- [ ] 文档更新完成
- [ ] 代码风格统一
- [ ] 无废弃代码

#### Day 16：验收和发布
**重构内容**：
```go
// 1. 最终验证
// - 运行完整测试套件
// - 性能基准测试
// - 代码质量检查

// 2. 准备发布
// - 更新CHANGELOG
// - 编写迁移指南
// - 准备发布说明
```

**验收标准**：
- [ ] 所有验收标准通过
- [ ] 性能无回归
- [ ] 代码质量达标

## 🔧 具体实施步骤详解

### 步骤11：从小处开始重构

#### 选择第一个重构目标
**标准**：相对独立、业务价值明确、代码不太复杂

**推荐选择**：Tokenize模块
- **理由**：词法分析相对独立，与语法分析耦合较少
- **复杂度**：parser.go中的词法分析代码约200-300行
- **测试基础**：已有相关测试用例

#### 重构步骤
1. **创建新模块**：
   ```bash
   mkdir -p internal/modules/frontend/domain/analysis
   ```

2. **定义接口**：
   ```go
   // domain/analysis/tokenizer.go
   type Tokenizer interface {
       Tokenize(source SourceCode) ([]Token, error)
   }
   ```

3. **创建值对象**：
   ```go
   // domain/analysis/token.go
   type Token struct {
       Kind     TokenKind
       Value    string
       Position Position
   }

   // domain/analysis/source_code.go
   type SourceCode struct {
       content  string
       filePath string
   }
   ```

4. **实现服务**：
   ```go
   // 提取parser.go中的词法分析代码
   type SimpleTokenizer struct{}

   func (t *SimpleTokenizer) Tokenize(source SourceCode) ([]Token, error) {
       // 迁移词法分析逻辑
   }
   ```

5. **更新应用层**：
   ```go
   // 修改frontend_service.go
   type FrontendService struct {
       tokenizer Tokenizer  // 使用接口
   }

   func (s *FrontendService) AnalyzeLexical(content string) ([]Token, error) {
       source := analysis.SourceCode{Content: content}
       return s.tokenizer.Tokenize(source)
   }
   ```

6. **运行测试**：
   ```bash
   go test ./internal/modules/frontend/... -v
   ```

### 步骤12：建立分层架构

#### 当前架构分析
```
frontend/
├── application/          # 应用层
│   └── services/         # 应用服务
├── domain/              # 领域层
│   ├── services/        # 领域服务（当前混合了太多逻辑）
│   ├── entities/        # 实体（AST模型）
│   └── valueobjects/    # 值对象（Token等）
└── infrastructure/      # 基础设施层
```

#### 目标架构
```
frontend/
├── application/          # 应用层
│   ├── services/         # 应用服务（编排）
│   ├── dtos/            # 数据传输对象
│   ├── commands/        # 命令对象
│   └── queries/         # 查询对象
├── domain/              # 领域层
│   ├── analysis/        # 分析子域
│   │   ├── tokenizer.go          # 词法分析服务
│   │   ├── syntax_parser.go      # 语法分析服务
│   │   ├── semantic_analyzer.go  # 语义分析服务
│   │   ├── token.go              # Token值对象
│   │   ├── source_code.go        # SourceCode值对象
│   │   └── position.go           # Position值对象
│   ├── entities/        # 实体（增强的AST模型）
│   │   ├── source_file.go        # SourceFile聚合根
│   │   ├── program.go            # Program聚合根
│   │   ├── symbol.go             # Symbol实体
│   │   └── scope.go              # Scope实体
│   ├── valueobjects/    # 值对象
│   ├── services/        # 领域服务接口
│   └── repositories/    # 仓储接口
└── infrastructure/      # 基础设施层
    ├── analysis/        # 分析基础设施
    │   ├── tokenizer_impl.go     # Tokenizer实现
    │   ├── parser_impl.go        # Parser实现
    │   └── analyzer_impl.go      # Analyzer实现
    ├── persistence/     # 持久化
    │   ├── repositories/         # 仓储实现
    │   └── models/               # 数据模型
    ├── external/        # 外部服务适配
    └── di/              # 依赖注入
```

#### 迁移步骤
1. **创建新目录结构**
2. **迁移领域对象**：将值对象和实体移到合适位置
3. **拆分领域服务**：将parser.go拆分为多个专用服务
4. **创建仓储接口**：定义数据访问接口
5. **更新应用服务**：使用新的领域服务接口
6. **配置DI容器**：设置依赖注入

### 步骤13：渐进式替换

#### 替换策略
**原则**：先外围，后核心；保持API兼容，逐步迁移

1. **Phase 1**: 词法分析（1-2天）
   - 创建Tokenizer接口和实现
   - 更新应用服务使用新接口
   - 保持原有Parse方法不变

2. **Phase 2**: 语法分析（3-5天）
   - 创建SyntaxParser接口和实现
   - 迁移语句和表达式解析逻辑
   - 更新Parse方法使用新解析器

3. **Phase 3**: 语义分析（2-3天）
   - 创建SemanticAnalyzer接口和实现
   - 迁移符号表和类型检查逻辑
   - 扩展应用服务支持语义分析

4. **Phase 4**: 聚合根和仓储（2-3天）
   - 实现SourceFile和Program聚合根
   - 创建仓储接口和实现
   - 重构应用服务使用聚合根模式

5. **Phase 5**: 清理和优化（1-2天）
   - 移除废弃代码
   - 优化性能
   - 更新文档

#### 兼容性保证
- **API兼容**：保持原有public方法的签名
- **测试兼容**：现有测试继续通过
- **渐进迁移**：可以随时停止，回滚到上一个稳定状态

## 🎯 验收标准

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

## 📊 进度跟踪

### 每日进度报告模板
```markdown
## Day N 进度报告

### 完成任务
- [x] 任务1
- [x] 任务2
- [ ] 任务3

### 测试状态
- 单元测试：通过 X/总 Y (Z%)
- 集成测试：通过 A/总 B (C%)

### 代码质量
- 复杂度最高方法：方法名 (复杂度值)
- 测试覆盖率：D%
- 代码行数变化：parser.go: E行 → F行

### 遇到问题
- 问题1：描述 + 解决方案
- 问题2：描述 + 解决方案

### 明日计划
- 计划任务1
- 计划任务2
- 风险评估
```

### 里程碑检查点
- **Week 1 结束**：测试基线建立，领域对象框架完成
- **Week 2 结束**：核心领域服务实现，parser.go拆分50%
- **Week 3 结束**：完整DDD架构建立，聚合根和仓储实现
- **Week 4 结束**：测试完善，性能优化，文档更新

## 🛠️ 工具和资源

### 开发工具
- **IDE**: GoLand/VS Code with Go插件
- **测试**: `go test` + `testify`断言库
- **覆盖率**: `go tool cover`
- **复杂度**: `gocyclo`
- **依赖分析**: `go mod graph`

### 文档资源
- **DDD参考**: Domain-Driven Design by Eric Evans
- **Go最佳实践**: Effective Go
- **测试驱动**: Test-Driven Development by Kent Beck

### 团队协作
- **代码审查**: 每次提交前进行
- **结对编程**: 复杂重构时采用
- **每日站会**: 同步进度和问题
- **重构日志**: 记录决策和变更

## ⚠️ 风险控制

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

## 🎉 成功标志

重构成功的核心标志：
1. **代码可读性**：新开发者能快速理解业务逻辑
2. **维护效率**：修改功能时只需改动相关领域对象
3. **测试稳定性**：重构后测试更加稳定可靠
4. **扩展便利性**：添加新语法元素变得简单
5. **性能保持**：分析速度不低于原有实现
6. **团队满意度**：开发者对新架构感到满意

通过这个详细的重构计划，我们将把一个复杂的单体parser.go重构为清晰、可维护的DDD领域模型，为echo-lang编译器的持续发展奠定坚实的基础。
