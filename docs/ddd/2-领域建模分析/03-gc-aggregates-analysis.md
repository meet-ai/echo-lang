# GC聚合分析：划定业务一致性边界

## 1. 本阶段输入
- **业务流程**：[GC业务流程](./01-gc-process-flow.md)
- **业务规则**：[GC核心业务规则](./02-gc-business-rules.md)
- **领域名词**：[GC领域名词分析](./01-gc-domain-nouns.md)
- **领域动词**：[GC领域动词分析](./02-gc-domain-verbs.md)

## 2. 聚合边界推演过程

### 2.1 罗列修改操作涉及的所有数据

当"触发GC"命令被执行时，系统需要创建或修改以下数据：
1. **垃圾回收器** 状态（从空闲变为执行中）
2. **GC周期** 记录（开始时间、阶段状态）
3. **堆** 使用统计（使用率、分配压力）
4. **对象** 标记状态（标记位图更新）
5. **分配器** 状态（暂停分配、等待GC完成）
6. **性能指标** 记录（暂停时间、吞吐量等）

### 2.2 识别强不变条件（业务规则映射）

基于业务规则分析，识别出以下强不变条件：

- **IC1 (强一致)**：`对象标记状态`必须与`实际可达性`保持一致（GC-BR-MEM-001）
- **IC2 (强一致)**：`堆使用统计`必须实时反映`实际内存使用`（GC-BR-PERF-002）
- **IC3 (强一致)**：`GC配置参数`必须在整个GC周期内保持一致（GC-BR-CONF-001）
- **IC4 (强一致)**：`模块状态`必须与`当前GC策略`保持同步（GC-BR-MOD-001）

### 2.3 根据不变条件分组（提出候选聚合）

基于不变条件分析，提出以下候选聚合：

- **候选聚合A：GC执行聚合**
    - **包含**：`垃圾回收器`(根), `GC周期`, `标记阶段`, `清扫阶段`
    - **理由**：为了保证**IC1**，GC执行过程中的状态转换必须是原子的，包括标记和清扫阶段的协调。

- **候选聚合B：堆管理聚合**
    - **包含**：`堆`(根), `分配器`, `对象`, `标记位图`
    - **理由**：为了保证**IC2**，堆的状态变化（分配、回收）必须与统计信息保持强一致。

- **候选聚合C：自适应控制聚合**
    - **包含**：`自适应决策`(根), `工作负载特征`, `性能指标`, `规则引擎`
    - **理由**：为了保证**IC3**和**IC4**，配置调整和模块切换必须是原子的决策过程。

## 3. 聚合设计结论

### 3.1 核心聚合定义

| 聚合根                 | 内部实体/值对象                                 | 核心不变条件 | 所属业务子域 | 决策依据                   |
| :--------------------- | :---------------------------------------------- | :----------- | :----------- | :------------------------- |
| **GarbageCollector**   | GCCycle, MarkPhase, SweepPhase                  | IC1          | 核心GC       | GC执行状态必须强一致       |
| **Heap**               | Allocator, Object, MarkBitmap                   | IC2          | 核心GC       | 堆状态与统计必须实时同步   |
| **AdaptiveController** | WorkloadProfile, PerformanceMetrics, RuleEngine | IC3, IC4     | 自适应GC     | 配置和模块状态必须原子调整 |

### 3.2 聚合职责分析

#### GarbageCollector聚合
**职责**：管理GC执行生命周期
- **命令操作**：
  - `triggerGC()`: 启动新的GC周期
  - `completeGCCycle()`: 完成GC周期
- **查询操作**：
  - `getCurrentPhase()`: 获取当前GC阶段
  - `getGCStatistics()`: 获取GC统计信息
- **业务规则**：
  - 每个GC周期必须完整执行标记和清扫阶段
  - GC状态转换必须遵循正确的顺序

#### Heap聚合
**职责**：管理内存分配和回收
- **命令操作**：
  - `allocate(size)`: 分配内存
  - `markObject(addr)`: 标记对象
  - `sweepUnmarked()`: 清扫未标记对象
- **查询操作**：
  - `getUsageRatio()`: 获取使用率
  - `getFragmentation()`: 获取碎片率
- **业务规则**：
  - 分配必须检查可用空间
  - 回收必须基于正确的标记状态

#### AdaptiveController聚合
**职责**：根据运行时指标调整GC策略
- **命令操作**：
  - `analyzeWorkload()`: 分析当前工作负载
  - `executeRules()`: 执行自适应规则
  - `adjustConfiguration()`: 调整GC配置
- **查询操作**：
  - `getCurrentWorkloadType()`: 获取当前工作负载类型
  - `getActiveModules()`: 获取当前活跃模块
- **业务规则**：
  - 配置调整必须验证参数有效性
  - 模块切换必须检查依赖关系

### 3.3 聚合间关系分析

#### 3.3.1 依赖关系
```
GarbageCollector → Heap [强依赖]
  理由：GC需要操作堆进行标记和清扫

AdaptiveController → GarbageCollector [弱依赖]  
  理由：自适应控制器需要了解GC状态来做决策

AdaptiveController → Heap [弱依赖]
  理由：需要堆的使用统计来分析工作负载
```

#### 3.3.2 通信模式
- **GarbageCollector → Heap**：直接方法调用（强一致性）
- **AdaptiveController → GarbageCollector**：事件订阅（最终一致性）
- **AdaptiveController → Heap**：指标查询（只读访问）

### 3.4 聚合设计验证

#### 3.4.1 事务一致性验证
- **GarbageCollector聚合**：单聚合事务，确保GC状态一致性
- **Heap聚合**：单聚合事务，确保内存状态一致性
- **AdaptiveController聚合**：单聚合事务，确保配置一致性

#### 3.4.2 并发访问验证
- **GarbageCollector**：使用锁保护状态转换
- **Heap**：使用无锁分配器，支持并发分配
- **AdaptiveController**：使用乐观锁处理配置更新

#### 3.4.3 性能影响验证
- **聚合边界**：确保不会因为聚合划分影响GC性能
- **数据局部性**：相关数据在同一聚合中提高缓存效率
- **通信开销**：最小化聚合间通信延迟

## 4. 聚合演化规划

### 4.1 当前版本聚合
- **V1.0**：基础三聚合结构
  - GarbageCollector：核心GC执行
  - Heap：内存管理
  - AdaptiveController：自适应控制

### 4.2 未来扩展聚合
- **Monitoring聚合**：独立的监控和诊断功能
- **Module聚合**：模块管理和生命周期
- **Configuration聚合**：配置管理和验证

### 4.3 聚合拆分策略
- **拆分时机**：当聚合变得过于复杂（>10个方法）
- **拆分原则**：保持领域概念的内聚性
- **迁移策略**：逐步迁移，保持向后兼容

## 5. 聚合与限界上下文映射

### 5.1 上下文识别
基于聚合分析，自然形成以下限界上下文：

- **核心GC上下文**：包含GarbageCollector和Heap聚合
- **自适应GC上下文**：包含AdaptiveController聚合
- **增强GC上下文**：模块化扩展功能
- **监控GC上下文**：监控和诊断功能

### 5.2 上下文边界
- **核心GC上下文**：必须功能，基础GC算法
- **自适应GC上下文**：可选功能，智能调优
- **增强GC上下文**：可选功能，高级特性
- **监控GC上下文**：可选功能，可观测性

## 6. 聚合设计质量评估

### 6.1 设计原则符合性
- ✅ **单一职责**：每个聚合只负责一个业务能力
- ✅ **封闭操作**：聚合内操作保持不变性
- ✅ **独立性**：聚合间松耦合，通过ID引用

### 6.2 技术实现可行性
- ✅ **事务边界**：聚合对应事务边界
- ✅ **并发控制**：合理的并发访问策略
- ✅ **性能要求**：满足GC的性能需求

### 6.3 业务价值验证
- ✅ **业务概念清晰**：聚合根对应业务实体
- ✅ **规则完整性**：不变条件覆盖核心业务规则
- ✅ **演化空间**：为未来扩展预留空间

## 7. 聚合实现的指导原则

### 7.1 聚合根实现模式
```go
// 聚合根接口定义
type GarbageCollector interface {
    // 命令方法
    TriggerGC() error
    GetCurrentPhase() GCPhase
    
    // 查询方法（可选）
    GetStatistics() GCStatistics
}

// 聚合根实现
type garbageCollector struct {
    id       string
    phase    GCPhase
    cycle    *GCCycle
    heap     Heap  // 引用其他聚合的ID，而非对象
}
```

### 7.2 实体实现模式
```go
// 实体定义
type GCCycle struct {
    id          string
    startTime   time.Time
    endTime     *time.Time
    phases      []GCPhaseRecord
    statistics  GCStatistics
}
```

### 7.3 值对象实现模式
```go
// 值对象（不可变）
type GCStatistics struct {
    duration        time.Duration
    objectsCollected int64
    memoryReclaimed  int64
    pauseTime        time.Duration
}

// 工厂方法确保不变性
func NewGCStatistics(duration time.Duration, objectsCollected, memoryReclaimed int64, pauseTime time.Duration) GCStatistics {
    return GCStatistics{
        duration:        duration,
        objectsCollected: objectsCollected,
        memoryReclaimed:  memoryReclaimed,
        pauseTime:        pauseTime,
    }
}
```

### 7.4 仓储接口设计
```go
// 仓储接口（按聚合定义）
type GarbageCollectorRepository interface {
    FindByID(id string) (*GarbageCollector, error)
    Save(gc *GarbageCollector) error
    FindActive() (*GarbageCollector, error)
}
```
