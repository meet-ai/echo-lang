# GC上下文依赖设计与接口契约

## 1. 依赖关系回顾

基于上下文关系分析，总结依赖关系：

- **核心GC上下文**：被所有其他上下文依赖，是GC系统的核心
- **自适应GC上下文**：依赖核心GC的指标和事件，控制增强GC模块
- **增强GC上下文**：被自适应GC控制，实现具体的GC增强算法
- **监控GC上下文**：订阅所有上下文的事件，收集监控数据

## 2. 依赖解决方案选择与接口设计

### 2.1 核心GC → 自适应GC 的依赖
**关系模式**：客户-供应商 (Customer-Supplier)
**解决方案**：核心GC作为客户，自适应GC作为供应商，提供决策服务。

#### 防腐层接口设计：`AdaptiveGCProvider` (位于核心GC内)
```go
// internal/modules/core-gc/ports/adaptive_gc_provider.go
package ports

import (
    "context"
    "time"
)

// AdaptiveGCProvider 自适应GC提供者接口（防腐层）
type AdaptiveGCProvider interface {
    // AnalyzeWorkload 分析当前工作负载类型
    AnalyzeWorkload(ctx context.Context, metrics PerformanceMetrics) (WorkloadType, error)
    
    // MakeDecisions 根据指标做出决策
    MakeDecisions(ctx context.Context, metrics PerformanceMetrics) ([]GCDecision, error)
    
    // ApplyDecisions 应用决策
    ApplyDecisions(ctx context.Context, decisions []GCDecision) error
    
    // GetCurrentConfiguration 获取当前配置
    GetCurrentConfiguration(ctx context.Context) (GCConfiguration, error)
}

// 相关数据结构
type PerformanceMetrics struct {
    PauseTimeP95       time.Duration
    ThroughputLoss     float64
    MemoryOverhead     float64
    AllocationRate     int64
    HeapUsage          float64
    FragmentationRatio float64
}

type WorkloadType string

const (
    WorkloadTransactional WorkloadType = "transactional"
    WorkloadBatch         WorkloadType = "batch_processing"
    WorkloadCache         WorkloadType = "cache_server"
    WorkloadMixed         WorkloadType = "mixed"
)

type GCDecision struct {
    Type        DecisionType
    ModuleID    string
    Parameter   string
    Value       interface{}
    Reason      string
    Priority    int
}

type DecisionType string

const (
    DecisionEnableModule   DecisionType = "enable_module"
    DecisionDisableModule  DecisionType = "disable_module"
    DecisionAdjustParameter DecisionType = "adjust_parameter"
    DecisionTriggerGC      DecisionType = "trigger_gc"
)
```

#### 技术集成方案
- **集成技术**：事件驱动 + 直接调用
- **协议**：gRPC (高性能) + 事件总线 (解耦)
- **数据格式**：Protocol Buffers
- **错误处理**：重试 + 降级

### 2.2 核心GC → 增强GC 的依赖
**关系模式**：分离方式 → 开放主机服务
**解决方案**：核心GC提供模块接口，增强GC实现接口。

#### 开放主机服务接口设计：`GCModuleHost` (位于核心GC内)
```go
// internal/modules/core-gc/ports/gc_module_host.go
package ports

import (
    "context"
)

// GCModuleHost 开放主机服务接口
type GCModuleHost interface {
    // 注册模块
    RegisterModule(module GCModule) error
    
    // 注销模块
    UnregisterModule(moduleID string) error
    
    // 获取已注册模块
    GetRegisteredModules() ([]GCModuleInfo, error)
    
    // 调用模块生命周期
    CallModuleLifecycle(ctx context.Context, moduleID string, lifecycle ModuleLifecycle, config ModuleConfig) error
}

// 模块接口（由核心GC定义，增强GC实现）
type GCModule interface {
    // 元信息
    ID() string
    Name() string
    Version() string
    Type() ModuleType
    Dependencies() []string
    
    // 生命周期
    Init(ctx context.Context, config ModuleConfig) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Cleanup(ctx context.Context) error
    
    // GC回调
    BeforeGC(ctx context.Context, gcInfo GCInfo) error
    AfterGC(ctx context.Context, gcInfo GCInfo, stats GCStats) error
    
    // 配置管理
    UpdateConfig(ctx context.Context, config ModuleConfig) error
    GetConfig(ctx context.Context) (ModuleConfig, error)
    
    // 统计信息
    GetStats(ctx context.Context) (ModuleStats, error)
    
    // 健康检查
    HealthCheck(ctx context.Context) (ModuleHealth, error)
}

// 相关数据结构
type ModuleType string

const (
    ModuleTypeGenerational ModuleType = "generational"
    ModuleTypeIncremental  ModuleType = "incremental"
    ModuleTypeCompaction   ModuleType = "compaction"
    ModuleTypeNUMA         ModuleType = "numa"
)

type ModuleLifecycle string

const (
    LifecycleInit    ModuleLifecycle = "init"
    LifecycleStart   ModuleLifecycle = "start"
    LifecycleStop    ModuleLifecycle = "stop"
    LifecycleCleanup ModuleLifecycle = "cleanup"
)

type ModuleConfig struct {
    Enabled bool                   `json:"enabled"`
    Params  map[string]interface{} `json:"params"`
}

type GCInfo struct {
    GCID        string        `json:"gc_id"`
    Phase       GCPhase       `json:"phase"`
    StartTime   time.Time     `json:"start_time"`
    TriggerType GCTriggerType `json:"trigger_type"`
}

type ModuleStats struct {
    ModuleID         string                 `json:"module_id"`
    Uptime           time.Duration          `json:"uptime"`
    ProcessedGCs     int64                  `json:"processed_gcs"`
    AverageLatency   time.Duration          `json:"average_latency"`
    SuccessRate      float64                `json:"success_rate"`
    CustomMetrics    map[string]interface{} `json:"custom_metrics"`
}

type ModuleHealth struct {
    Status  HealthStatus `json:"status"`
    Message string       `json:"message"`
    Details map[string]interface{} `json:"details"`
}
```

#### 技术集成方案
- **集成技术**：插件架构 + 依赖注入
- **协议**：Go插件系统 + 接口调用
- **数据格式**：Go结构体
- **生命周期**：模块热插拔

### 2.3 核心GC → 监控GC 的依赖
**关系模式**：发布语言 (Published Language)
**解决方案**：通过事件总线发布GC事件，监控GC订阅处理。

#### 发布语言规范：GC事件格式
```go
// internal/shared/events/gc_events.go
package events

import (
    "time"
)

// GC事件基类
type GCEvent struct {
    EventID    string    `json:"event_id"`
    GCID       string    `json:"gc_id"`
    Timestamp  time.Time `json:"timestamp"`
    Source     string    `json:"source"` // 上下文标识
}

// 具体GC事件
type GCTriggeredEvent struct {
    GCEvent
    TriggerReason string  `json:"trigger_reason"`
    HeapUsage     float64 `json:"heap_usage"`
    AllocationRate int64  `json:"allocation_rate"`
}

type GCCycleCompletedEvent struct {
    GCEvent
    Duration        time.Duration `json:"duration"`
    PauseTime       time.Duration `json:"pause_time"`
    MemoryReclaimed int64         `json:"memory_reclaimed"`
    ObjectsCollected int64        `json:"objects_collected"`
    Statistics      GCStats       `json:"statistics"`
}

type HeapPressureEvent struct {
    GCEvent
    HeapID      string  `json:"heap_id"`
    UsageRatio  float64 `json:"usage_ratio"`
    PressureLevel string `json:"pressure_level"` // "low", "medium", "high", "critical"
}

// 事件处理器接口（由监控GC实现）
type GCEventHandler interface {
    HandleGCTriggered(event GCTriggeredEvent) error
    HandleGCCycleCompleted(event GCCycleCompletedEvent) error
    HandleHeapPressure(event HeapPressureEvent) error
    SupportedEventTypes() []string
}
```

#### 技术集成方案
- **集成技术**：事件驱动架构
- **协议**：消息队列 (NATS/Redis) + 事件总线
- **数据格式**：JSON/Protocol Buffers
- **一致性**：最终一致性

### 2.4 自适应GC → 增强GC 的依赖
**关系模式**：客户-供应商
**解决方案**：自适应GC作为客户，通过命令模式控制增强GC。

#### 接口设计：`ModuleController` (位于自适应GC内)
```go
// internal/modules/adaptive-gc/ports/module_controller.go
package ports

import (
    "context"
)

// ModuleController 模块控制器接口（防腐层）
type ModuleController interface {
    // 模块管理
    EnableModule(ctx context.Context, moduleID string, config ModuleConfig) error
    DisableModule(ctx context.Context, moduleID string) error
    ListAvailableModules(ctx context.Context) ([]ModuleInfo, error)
    
    // 模块配置
    UpdateModuleConfig(ctx context.Context, moduleID string, config ModuleConfig) error
    GetModuleConfig(ctx context.Context, moduleID string) (ModuleConfig, error)
    
    // 模块状态查询
    GetModuleStatus(ctx context.Context, moduleID string) (ModuleStatus, error)
    GetModuleStats(ctx context.Context, moduleID string) (ModuleStats, error)
    
    // 批量操作
    BatchUpdateModules(ctx context.Context, updates []ModuleUpdate) error
}

// 相关数据结构
type ModuleInfo struct {
    ID          string      `json:"id"`
    Name        string      `json:"name"`
    Type        ModuleType  `json:"type"`
    Version     string      `json:"version"`
    Status      ModuleStatus `json:"status"`
    Description string      `json:"description"`
}

type ModuleStatus struct {
    ModuleID    string         `json:"module_id"`
    State       ModuleState    `json:"state"`
    Uptime      time.Duration  `json:"uptime"`
    LastError   string         `json:"last_error,omitempty"`
    Health      ModuleHealth   `json:"health"`
}

type ModuleState string

const (
    ModuleStateDisabled ModuleState = "disabled"
    ModuleStateEnabling ModuleState = "enabling"
    ModuleStateEnabled  ModuleState = "enabled"
    ModuleStateDisabling ModuleState = "disabling"
    ModuleStateError    ModuleState = "error"
)

type ModuleUpdate struct {
    ModuleID string       `json:"module_id"`
    Action   UpdateAction `json:"action"`
    Config   ModuleConfig `json:"config,omitempty"`
}

type UpdateAction string

const (
    UpdateActionEnable  UpdateAction = "enable"
    UpdateActionDisable UpdateAction = "disable"
    UpdateActionUpdate  UpdateAction = "update"
)
```

#### 技术集成方案
- **集成技术**：命令模式 + 同步调用
- **协议**：gRPC
- **数据格式**：Protocol Buffers
- **错误处理**：重试 + 补偿

## 3. 技术集成方案汇总

| 依赖方向 | 解决方案模式 | 技术实现 | 接口/契约位置 | 通信模式 | 一致性要求 |
| --- | --- | --- | --- | --- | --- |
| 核心GC → 自适应GC | 客户-供应商 + 防腐层 | 事件总线 + gRPC | 核心GC定义接口 | 异步+同步 | 最终+强 |
| 核心GC → 增强GC | 开放主机服务 | 插件系统 + 接口 | 核心GC定义接口 | 同步调用 | 强 |
| 核心GC → 监控GC | 发布语言 | 事件总线 | 共享事件格式 | 异步 | 最终 |
| 自适应GC → 增强GC | 客户-供应商 + 防腐层 | gRPC | 自适应GC定义接口 | 同步 | 强 |
| 自适应GC → 监控GC | 发布语言 | 事件总线 | 共享事件格式 | 异步 | 最终 |
| 增强GC → 监控GC | 发布语言 | 事件总线 | 共享事件格式 | 异步 | 最终 |

## 4. 依赖倒置与防腐层实现

### 4.1 依赖倒置原则应用
```go
// 高层模块（核心GC）不依赖低层模块（自适应GC实现）
// 而是通过接口依赖抽象

// ❌ 错误：高层直接依赖低层
type CoreGC struct {
    adaptiveGC *adaptive.Service // 直接依赖具体实现
}

// ✅ 正确：高层依赖抽象
type CoreGC struct {
    adaptiveProvider ports.AdaptiveGCProvider // 依赖接口
}
```

### 4.2 防腐层实现模式
```go
// internal/infrastructure/adapters/adaptive_gc_adapter.go
package adapters

import (
    "context"
    
    "your-project/internal/modules/adaptive-gc/service"
    "your-project/internal/modules/core-gc/ports"
)

type AdaptiveGCAdapter struct {
    adaptiveService *service.AdaptiveGCService
    translator      *AdaptiveGCTranslator
}

func (a *AdaptiveGCAdapter) AnalyzeWorkload(ctx context.Context, metrics ports.PerformanceMetrics) (ports.WorkloadType, error) {
    // 1. 转换数据格式（防腐层作用）
    internalMetrics := a.translator.TranslateMetrics(metrics)
    
    // 2. 调用内部服务
    workload, err := a.adaptiveService.AnalyzeWorkload(ctx, internalMetrics)
    if err != nil {
        return "", err
    }
    
    // 3. 转换返回结果
    return a.translator.TranslateWorkloadType(workload), nil
}

// 数据转换器
type AdaptiveGCTranslator struct{}

func (t *AdaptiveGCTranslator) TranslateMetrics(external ports.PerformanceMetrics) internal.PerformanceMetrics {
    return internal.PerformanceMetrics{
        PauseTimeP95:   external.PauseTimeP95,
        ThroughputLoss: external.ThroughputLoss,
        // ... 其他字段映射
    }
}
```

### 4.3 依赖注入配置
```go
// internal/infrastructure/di/container.go
package di

import (
    "your-project/internal/infrastructure/adapters"
    "your-project/internal/modules/adaptive-gc/service"
    "your-project/internal/modules/core-gc"
)

func BuildContainer() *Container {
    // 创建具体实现
    adaptiveService := service.NewAdaptiveGCService()
    
    // 创建适配器（防腐层）
    adaptiveAdapter := adapters.NewAdaptiveGCAdapter(adaptiveService)
    
    // 创建核心GC，注入接口
    coreGC := coregc.NewCoreGC(adaptiveAdapter /* 其他依赖 */)
    
    return &Container{
        CoreGC: coreGC,
        // ...
    }
}
```

## 5. 接口契约管理

### 5.1 接口版本控制
```go
// internal/shared/contracts/interface_versions.go
package contracts

const (
    AdaptiveGCProviderVersion = "v1.2.0"
    GCModuleHostVersion       = "v1.1.0"
    ModuleControllerVersion   = "v1.0.0"
)

// 接口兼容性检查
type InterfaceCompatibility struct {
    RequiredVersion string
    CurrentVersion  string
}

func CheckCompatibility(required, current string) error {
    // 语义化版本比较
    // v1.2.0 与 v1.1.0 兼容
    // v2.0.0 与 v1.x.x 不兼容
}
```

### 5.2 契约测试
```go
// internal/shared/contracts/contract_tests.go
package contracts

import (
    "testing"
    "your-project/internal/modules/core-gc/ports"
)

// 消费者契约测试（测试接口的期望行为）
func TestAdaptiveGCProvider_AnalyzeWorkload(t *testing.T) {
    // 1. 创建mock实现
    mockProvider := &MockAdaptiveGCProvider{}
    
    // 2. 设置期望
    mockProvider.On("AnalyzeWorkload", mock.Anything, mock.AnythingOfType("ports.PerformanceMetrics")).
        Return(ports.WorkloadTransactional, nil)
    
    // 3. 测试消费者代码
    coreGC := coregc.NewCoreGC(mockProvider)
    
    workload, err := coreGC.GetWorkloadType()
    
    // 4. 验证
    assert.NoError(t, err)
    assert.Equal(t, ports.WorkloadTransactional, workload)
    mockProvider.AssertExpectations(t)
}

// 提供者契约测试（测试接口的实现）
func TestAdaptiveGCService_ImplementsProvider(t *testing.T) {
    service := service.NewAdaptiveGCService()
    
    // 编译时检查：确保实现接口
    var _ ports.AdaptiveGCProvider = service
}
```

### 5.3 接口文档生成
```go
// internal/shared/contracts/interface_docs.go
package contracts

import (
    "fmt"
)

// 接口文档生成器
type InterfaceDocumenter struct{}

func (d *InterfaceDocumenter) GenerateInterfaceDocs() string {
    docs := `
# GC上下文接口契约文档

## AdaptiveGCProvider 接口

### 方法列表
1. AnalyzeWorkload(ctx, metrics) -> (WorkloadType, error)
   - 分析当前工作负载类型
   - 输入：性能指标
   - 输出：工作负载类型
   - 异常：分析失败

2. MakeDecisions(ctx, metrics) -> ([]GCDecision, error)
   - 根据指标做出GC决策
   - 输入：性能指标
   - 输出：决策列表
   - 异常：决策失败

### 版本兼容性
- v1.0.0：初始版本
- v1.1.0：添加批量决策支持
- v1.2.0：支持自定义指标

### 使用示例
` + "```go" + `
provider := container.GetAdaptiveGCProvider()
workload, err := provider.AnalyzeWorkload(ctx, metrics)
` + "```" + `
`
    return docs
}
```

## 6. 集成测试策略

### 6.1 上下文集成测试
```go
// internal/integration_test/gc_contexts_integration_test.go
package integration_test

import (
    "testing"
    "your-project/internal/infrastructure/di"
)

func TestGCContextsIntegration(t *testing.T) {
    // 1. 启动所有上下文
    container := di.BuildIntegrationContainer()
    
    // 2. 模拟应用运行
    app := container.GetApplication()
    
    // 3. 执行一些操作触发GC
    for i := 0; i < 1000; i++ {
        app.AllocateObject()
    }
    
    // 4. 等待GC完成
    time.Sleep(100 * time.Millisecond)
    
    // 5. 验证各上下文协作结果
    coreGC := container.GetCoreGC()
    adaptiveGC := container.GetAdaptiveGC()
    monitoringGC := container.GetMonitoringGC()
    
    // 断言GC正常执行
    stats := coreGC.GetGCStats()
    assert.True(t, stats.TotalGCCycles > 0)
    
    // 断言自适应决策生效
    config := adaptiveGC.GetCurrentConfig()
    assert.NotNil(t, config)
    
    // 断言监控数据收集
    metrics := monitoringGC.GetCollectedMetrics()
    assert.True(t, len(metrics) > 0)
}
```

### 6.2 端到端测试
```go
func TestGCEndToEnd(t *testing.T) {
    // 1. 启动完整GC系统
    system := StartGCSystem(t)
    defer system.Shutdown()
    
    // 2. 模拟真实应用负载
    loadGenerator := NewLoadGenerator(system)
    loadGenerator.GenerateTransactionalLoad(10 * time.Second)
    
    // 3. 验证系统行为
    system.AssertGCBehavior(t, ExpectedBehavior{
        MaxPauseTime: 10 * time.Millisecond,
        MinThroughput: 0.95,
        AdaptationTime: 2 * time.Second,
    })
}
```

## 7. 部署与运维考虑

### 7.1 上下文部署策略
- **核心GC**：始终嵌入应用进程
- **自适应GC**：可选独立部署，支持远程调用
- **增强GC**：模块化，按需部署
- **监控GC**：独立服务，支持多租户

### 7.2 故障隔离
- **隔离级别**：进程级隔离防止故障传播
- **降级策略**：核心GC始终可用，其他上下文可降级
- **恢复机制**：自动重启和状态恢复

### 7.3 监控与告警
- **健康检查**：各上下文提供健康状态
- **性能监控**：上下文间调用链路跟踪
- **告警规则**：接口调用失败、服务不可用等
