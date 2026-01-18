# GC领域事件分析

## 1. 事件列表

基于领域动词分析和聚合设计，识别出以下关键领域事件：

### 1.1 GC执行生命周期事件
| 事件                | 发布聚合         | 触发时机     | 载荷关键字段                            |
| ------------------- | ---------------- | ------------ | --------------------------------------- |
| GCTriggered         | GarbageCollector | GC开始执行   | gcId, triggerReason, heapUsage          |
| MarkPhaseStarted    | GarbageCollector | 标记阶段开始 | gcId, phaseId, startTime                |
| MarkPhaseCompleted  | GarbageCollector | 标记阶段完成 | gcId, phaseId, markedObjects, duration  |
| SweepPhaseStarted   | GarbageCollector | 清扫阶段开始 | gcId, phaseId, startTime                |
| SweepPhaseCompleted | GarbageCollector | 清扫阶段完成 | gcId, phaseId, reclaimedBytes, duration |
| GCCycleCompleted    | GarbageCollector | GC周期完成   | gcId, totalDuration, statistics         |

### 1.2 自适应决策事件
| 事件             | 发布聚合           | 触发时机         | 载荷关键字段                                |
| ---------------- | ------------------ | ---------------- | ------------------------------------------- |
| WorkloadAnalyzed | AdaptiveController | 工作负载分析完成 | controllerId, workloadType, confidence      |
| RuleExecuted     | AdaptiveController | 自适应规则执行   | controllerId, ruleId, action, result        |
| ConfigAdjusted   | AdaptiveController | 配置参数调整     | controllerId, paramName, oldValue, newValue |
| ModuleEnabled    | AdaptiveController | 模块启用         | controllerId, moduleId, config              |
| ModuleDisabled   | AdaptiveController | 模块禁用         | controllerId, moduleId                      |

### 1.3 堆管理事件
| 事件               | 发布聚合 | 触发时机     | 载荷关键字段                              |
| ------------------ | -------- | ------------ | ----------------------------------------- |
| HeapExpanded       | Heap     | 堆大小增加   | heapId, oldSize, newSize, reason          |
| HeapCompacted      | Heap     | 堆内存压缩   | heapId, beforeSize, afterSize, savedBytes |
| AllocationPressure | Heap     | 分配压力过高 | heapId, usageRatio, allocationRate        |
| OutOfMemory        | Heap     | 内存不足     | heapId, requestedSize, availableSize      |

### 1.4 监控与诊断事件
| 事件                | 发布聚合          | 触发时机     | 载荷关键字段                        |
| ------------------- | ----------------- | ------------ | ----------------------------------- |
| MetricsCollected    | MonitoringService | 性能指标收集 | timestamp, metrics                  |
| AnomalyDetected     | MonitoringService | 异常情况检测 | anomalyType, severity, details      |
| AlertTriggered      | MonitoringService | 告警触发     | alertType, message, threshold       |
| PerformanceDegraded | MonitoringService | 性能退化     | metricName, currentValue, threshold |

### 1.5 模块管理事件
| 事件              | 发布聚合      | 触发时机       | 载荷关键字段                    |
| ----------------- | ------------- | -------------- | ------------------------------- |
| ModuleLoaded      | ModuleManager | 模块加载完成   | moduleId, version, dependencies |
| ModuleInitialized | ModuleManager | 模块初始化完成 | moduleId, config                |
| ModuleActivated   | ModuleManager | 模块激活       | moduleId, activationTime        |
| ModuleDeactivated | ModuleManager | 模块停用       | moduleId, deactivationTime      |

## 2. 事件流分析

### 2.1 GC执行主流程事件流
```
GCTriggered → MarkPhaseStarted → MarkPhaseCompleted 
    ↓
SweepPhaseStarted → SweepPhaseCompleted → GCCycleCompleted
```

**事件流说明**：
1. `GCTriggered` 启动GC周期
2. 进入标记阶段：`MarkPhaseStarted` → `MarkPhaseCompleted`
3. 进入清扫阶段：`SweepPhaseStarted` → `SweepPhaseCompleted`
4. `GCCycleCompleted` 结束GC周期

### 2.2 自适应控制事件流
```
GCCycleCompleted → MetricsCollected → WorkloadAnalyzed 
    ↓
RuleExecuted → ConfigAdjusted → (ModuleEnabled|ModuleDisabled)
```

**事件流说明**：
1. GC完成后收集指标：`GCCycleCompleted` → `MetricsCollected`
2. 分析工作负载：`MetricsCollected` → `WorkloadAnalyzed`
3. 执行规则调整：`WorkloadAnalyzed` → `RuleExecuted` → `ConfigAdjusted`
4. 根据需要启用/禁用模块：`ConfigAdjusted` → `ModuleEnabled`/`ModuleDisabled`

### 2.3 异常处理事件流
```
AnomalyDetected → AlertTriggered
    ↓
PerformanceDegraded → ConfigAdjusted
```

**事件流说明**：
1. 检测到异常：`AnomalyDetected` → `AlertTriggered`
2. 性能退化时自动调整：`PerformanceDegraded` → `ConfigAdjusted`

## 3. 事件发布策略

### 3.1 强一致性事件
**必须在同一事务中发布的事件**：
- `GCTriggered`, `MarkPhaseStarted`, `SweepPhaseStarted`：GC状态转换
- `ConfigAdjusted`, `ModuleEnabled`, `ModuleDisabled`：配置变更
- `HeapExpanded`, `HeapCompacted`：堆结构变更

**发布策略**：同步发布，确保事件与状态变更的原子性

### 3.2 最终一致性事件
**允许异步处理的事件**：
- `GCCycleCompleted`, `MetricsCollected`：统计信息
- `WorkloadAnalyzed`, `RuleExecuted`：分析结果
- `AnomalyDetected`, `AlertTriggered`：监控告警

**发布策略**：异步发布，通过事件总线分发

### 3.3 事件版本控制
```go
// 事件版本定义
type EventVersion struct {
    Major int // 不兼容变更
    Minor int // 向后兼容新增
    Patch int // 向后兼容修复
}

// 事件元数据
type EventMetadata struct {
    EventID     string
    EventType   string
    AggregateID string
    Version     EventVersion
    Timestamp   time.Time
    CorrelationID string // 关联ID，用于追踪
    CausationID   string // 因果ID，事件链追踪
}
```

## 4. 事件订阅者分析

### 4.1 核心GC上下文订阅者
| 事件               | 订阅者                 | 处理逻辑                 |
| ------------------ | ---------------------- | ------------------------ |
| GCTriggered        | 监控服务               | 开始性能监控             |
| MarkPhaseCompleted | 自适应控制器           | 评估标记效率             |
| GCCycleCompleted   | 自适应控制器、监控服务 | 触发自适应调整、生成报告 |

### 4.2 自适应GC上下文订阅者
| 事件             | 订阅者     | 处理逻辑              |
| ---------------- | ---------- | --------------------- |
| WorkloadAnalyzed | 规则引擎   | 基于负载类型选择规则  |
| RuleExecuted     | 配置管理器 | 应用规则执行结果      |
| ConfigAdjusted   | 模块管理器 | 根据配置启用/禁用模块 |

### 4.3 监控上下文订阅者
| 事件                | 订阅者       | 处理逻辑           |
| ------------------- | ------------ | ------------------ |
| MetricsCollected    | 分析引擎     | 计算趋势和异常检测 |
| AnomalyDetected     | 告警系统     | 发送告警通知       |
| PerformanceDegraded | 自适应控制器 | 触发紧急调整策略   |

## 5. 事件数据结构设计

### 5.1 事件基类设计
```go
// 事件接口
type DomainEvent interface {
    EventType() string
    EventID() string
    AggregateID() string
    OccurredAt() time.Time
    EventVersion() EventVersion
}

// 事件基类
type BaseEvent struct {
    eventID     string
    aggregateID string
    occurredAt  time.Time
    version     EventVersion
}

func (e BaseEvent) EventID() string { return e.eventID }
func (e BaseEvent) AggregateID() string { return e.aggregateID }
func (e BaseEvent) OccurredAt() time.Time { return e.occurredAt }
func (e BaseEvent) EventVersion() EventVersion { return e.version }
```

### 5.2 具体事件实现示例
```go
// GCTriggered 事件
type GCTriggered struct {
    BaseEvent
    TriggerReason string
    HeapUsage     float64
    AllocationRate int64
}

func (e GCTriggered) EventType() string { return "gc.triggered" }

// GCCycleCompleted 事件
type GCCycleCompleted struct {
    BaseEvent
    TotalDuration   time.Duration
    ObjectsCollected int64
    MemoryReclaimed  int64
    PauseTime        time.Duration
    Statistics       GCStatistics
}

func (e GCCycleCompleted) EventType() string { return "gc.cycle.completed" }
```

### 5.3 事件序列化设计
```go
// 事件序列化接口
type EventSerializer interface {
    Serialize(event DomainEvent) ([]byte, error)
    Deserialize(data []byte, eventType string) (DomainEvent, error)
}

// JSON序列化实现
type JSONEventSerializer struct {
    eventTypes map[string]reflect.Type
}

func (s *JSONEventSerializer) RegisterEvent(event DomainEvent) {
    eventType := event.EventType()
    s.eventTypes[eventType] = reflect.TypeOf(event)
}
```

## 6. 事件驱动架构设计

### 6.1 事件总线设计
```go
// 事件总线接口
type EventBus interface {
    Publish(event DomainEvent) error
    Subscribe(eventType string, handler EventHandler) error
    Unsubscribe(eventType string, handler EventHandler) error
}

// 事件处理器接口
type EventHandler interface {
    Handle(event DomainEvent) error
    EventTypes() []string
}

// 异步事件总线实现
type AsyncEventBus struct {
    handlers map[string][]EventHandler
    queue    chan DomainEvent
    wg       sync.WaitGroup
}
```

### 6.2 事件溯源支持
```go
// 聚合根支持事件溯源
type EventSourcedAggregate interface {
    AggregateRoot
    UncommittedEvents() []DomainEvent
    MarkEventsCommitted()
    LoadFromEvents(events []DomainEvent) error
}

// 事件存储接口
type EventStore interface {
    SaveEvents(aggregateID string, events []DomainEvent, expectedVersion int) error
    LoadEvents(aggregateID string) ([]DomainEvent, error)
    LoadEventsFromVersion(aggregateID string, version int) ([]DomainEvent, error)
}
```

## 7. 事件处理保证

### 7.1 至少一次传递保证
- **事件存储**：事件发布前持久化到数据库
- **去重机制**：基于事件ID去重处理
- **幂等处理**：事件处理器实现幂等性

### 7.2 有序处理保证
- **分区策略**：相同聚合的事件按序处理
- **版本控制**：事件版本号确保顺序
- **补偿机制**：乱序事件暂存等待前序事件

### 7.3 错误处理策略
- **重试机制**：失败事件自动重试
- **死信队列**：多次失败的事件进入死信队列
- **监控告警**：事件处理异常及时告警

## 8. 事件演化策略

### 8.1 向后兼容性
- **新增字段**：可选字段，旧版本忽略
- **字段重命名**：保持旧名称，提供别名
- **枚举扩展**：新增枚举值，保持兼容

### 8.2 版本迁移
- **事件升级器**：自动升级旧版本事件
- **并存策略**：新旧版本处理器并存
- **逐步迁移**：灰度发布新版本处理器

### 8.3 事件模式治理
- **事件目录**：维护所有事件的标准定义
- **版本控制**：事件schema版本管理
- **文档同步**：代码与文档保持同步
