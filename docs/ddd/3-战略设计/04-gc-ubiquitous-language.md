# GC统一语言词汇表

## 1. 核心领域概念

### 1.1 GC执行相关
| 术语 | 英文 | 定义 | 所属上下文 | 备注 |
| --- | --- | --- | --- | --- |
| 垃圾回收器 | Garbage Collector | 执行内存垃圾回收的核心组件 | 核心GC | 聚合根 |
| GC周期 | GC Cycle | 一次完整的GC执行过程 | 核心GC | 聚合根 |
| 标记阶段 | Mark Phase | GC中标识可达对象的阶段 | 核心GC | 实体 |
| 清扫阶段 | Sweep Phase | GC中回收不可达内存的阶段 | 核心GC | 实体 |
| 写屏障 | Write Barrier | 维护GC不变性的内存屏障 | 核心GC | 值对象 |

### 1.2 内存管理相关
| 术语 | 英文 | 定义 | 所属上下文 | 备注 |
| --- | --- | --- | --- | --- |
| 堆 | Heap | 应用程序的内存空间 | 核心GC | 聚合根 |
| 对象 | Object | 分配在堆上的数据结构 | 核心GC | 实体 |
| 分配器 | Allocator | 负责对象分配的组件 | 核心GC | 实体 |
| 标记位图 | Mark Bitmap | 记录对象标记状态的位图 | 核心GC | 实体 |
| 大对象 | Large Object | 超过阈值的大尺寸对象 | 核心GC | 实体 |

### 1.3 自适应控制相关
| 术语 | 英文 | 定义 | 所属上下文 | 备注 |
| --- | --- | --- | --- | --- |
| 自适应决策 | Adaptive Decision | 根据运行时指标调整GC策略的过程 | 自适应GC | 聚合根 |
| 工作负载特征 | Workload Profile | 应用运行特征的描述 | 自适应GC | 值对象 |
| 性能指标 | Performance Metrics | GC运行效果的度量 | 自适应GC | 值对象 |
| 规则引擎 | Rule Engine | 执行自适应规则的组件 | 自适应GC | 实体 |
| PID控制器 | PID Controller | 用于参数调整的控制算法 | 自适应GC | 值对象 |

### 1.4 增强模块相关
| 术语 | 英文 | 定义 | 所属上下文 | 备注 |
| --- | --- | --- | --- | --- |
| GC模块 | GC Module | 可插拔的GC增强功能 | 增强GC | 聚合根 |
| 分代模块 | Generational Module | 实现分代垃圾回收的模块 | 增强GC | 实体 |
| 增量模块 | Incremental Module | 实现增量标记的模块 | 增强GC | 实体 |
| 压缩模块 | Compaction Module | 实现内存压缩的模块 | 增强GC | 实体 |
| 模块管理器 | Module Manager | 管理模块生命周期的组件 | 增强GC | 实体 |

## 2. 业务规则相关

### 2.1 性能规则
| 术语 | 英文 | 定义 | 上下文 | 示例 |
| --- | --- | --- | --- | --- |
| 暂停时间 | Pause Time | GC执行时的停顿时间 | 核心GC | P95暂停 < 10ms |
| 吞吐量损失 | Throughput Loss | GC对应用吞吐量的影响 | 核心GC | 损失 < 5% |
| 内存开销 | Memory Overhead | GC元数据的内存消耗 | 核心GC | 开销 < 20% |
| 分配率 | Allocation Rate | 内存分配的速度 | 核心GC | 10MB/s |

### 2.2 安全规则
| 术语 | 英文 | 定义 | 上下文 | 示例 |
| --- | --- | --- | --- | --- |
| 三色不变性 | Tri-color Invariance | GC标记算法的基本约束 | 核心GC | 黑对象不指向白对象 |
| 写屏障 | Write Barrier | 维护不变性的屏障 | 核心GC | Dijkstra插入屏障 |
| 安全回收 | Safe Reclamation | 只回收不可达对象的保证 | 核心GC | 零内存损坏 |

### 2.3 自适应规则
| 术语 | 英文 | 定义 | 上下文 | 示例 |
| --- | --- | --- | --- | --- |
| 工作负载类型 | Workload Type | 应用的负载特征分类 | 自适应GC | 事务型、批处理、缓存 |
| 规则条件 | Rule Condition | 触发规则的条件表达式 | 自适应GC | 暂停时间 > 10ms |
| 决策动作 | Decision Action | 规则执行的调整动作 | 自适应GC | 启用增量GC |
| 置信度 | Confidence | 决策的确定程度 | 自适应GC | 95% |

## 3. 技术实现相关

### 3.1 算法相关
| 术语 | 英文 | 定义 | 上下文 | 示例 |
| --- | --- | --- | --- | --- |
| 三色标记 | Tri-color Marking | 并发标记的基本算法 | 核心GC | 白、灰、黑三色 |
| 混合写屏障 | Hybrid Write Barrier | 结合插入和删除屏障 | 核心GC | Dijkstra + Yuasa |
| 卡表 | Card Table | 分代GC中的跨代引用记录 | 增强GC | 字节级别的标记 |
| 转发指针 | Forwarding Pointer | 压缩时的对象移动记录 | 增强GC | 对象重定位 |

### 3.2 数据结构相关
| 术语 | 英文 | 定义 | 上下文 | 示例 |
| --- | --- | --- | --- | --- |
| 对象头 | Object Header | 对象的元数据 | 核心GC | 8字节紧凑格式 |
| 位图 | Bitmap | 标记状态的位图表示 | 核心GC | 每字节1位 |
| TLAB | Thread Local Allocation Buffer | 线程本地分配缓冲区 | 核心GC | 减少分配竞争 |
| 工作窃取队列 | Work Stealing Queue | 并发标记的任务队列 | 核心GC | 负载均衡 |

### 3.3 配置相关
| 术语 | 英文 | 定义 | 上下文 | 示例 |
| --- | --- | --- | --- | --- |
| GC配置 | GC Configuration | GC的行为配置 | 自适应GC | 触发阈值、目标暂停 |
| 模块配置 | Module Configuration | 增强模块的配置 | 增强GC | 启用状态、参数设置 |
| 部署模式 | Deployment Mode | GC系统的部署方式 | 所有 | 嵌入式、服务器、实时 |
| 工作负载预设 | Workload Preset | 不同负载的配置预设 | 自适应GC | 事务型预设、缓存预设 |

## 4. 事件相关

### 4.1 GC生命周期事件
| 术语 | 英文 | 定义 | 上下文 | 触发条件 |
| --- | --- | --- | --- | --- |
| GC已触发 | GC Triggered | GC开始执行的事件 | 核心GC | 堆使用率达到阈值 |
| 标记阶段已开始 | Mark Phase Started | 标记阶段开始 | 核心GC | GC周期开始 |
| 标记阶段已完成 | Mark Phase Completed | 标记阶段完成 | 核心GC | 所有可达对象已标记 |
| 清扫阶段已开始 | Sweep Phase Started | 清扫阶段开始 | 核心GC | 标记阶段完成 |
| 清扫阶段已完成 | Sweep Phase Completed | 清扫阶段完成 | 核心GC | 不可达内存已回收 |
| GC周期已完成 | GC Cycle Completed | GC周期结束 | 核心GC | 清扫阶段完成 |

### 4.2 自适应决策事件
| 术语 | 英文 | 定义 | 上下文 | 触发条件 |
| --- | --- | --- | --- | --- |
| 工作负载已分析 | Workload Analyzed | 工作负载分析完成 | 自适应GC | 指标收集完成 |
| 规则已执行 | Rule Executed | 自适应规则执行 | 自适应GC | 条件满足 |
| 配置已调整 | Config Adjusted | GC配置参数调整 | 自适应GC | 决策执行 |
| 模块已启用 | Module Enabled | 增强模块启用 | 自适应GC | 决策要求 |
| 模块已禁用 | Module Disabled | 增强模块禁用 | 自适应GC | 决策要求 |

### 4.3 监控诊断事件
| 术语 | 英文 | 定义 | 上下文 | 触发条件 |
| --- | --- | --- | --- | --- |
| 指标已收集 | Metrics Collected | 性能指标收集 | 监控GC | GC周期结束 |
| 异常已检测 | Anomaly Detected | 异常情况检测 | 监控GC | 指标异常 |
| 告警已触发 | Alert Triggered | 性能告警触发 | 监控GC | 阈值超出 |
| 性能已退化 | Performance Degraded | 性能退化 | 监控GC | 持续恶化 |

## 5. 统一语言使用指南

### 5.1 命名一致性
- **类名**：使用 PascalCase，体现领域概念
  ```go
  // ✅ 好的命名
  type GarbageCollector struct {}    // 聚合根
  type AdaptiveDecision struct {}    // 值对象
  
  // ❌ 不好的命名
  type GC struct {}                  // 缩写
  type AdaptiveDecider struct {}     // 技术化
  ```

- **方法名**：使用 camelCase，体现业务行为
  ```go
  // ✅ 好的命名
  func (gc *GarbageCollector) TriggerGC() error        // 触发GC
  func (ad *AdaptiveDecision) AnalyzeWorkload() error  // 分析工作负载
  
  // ❌ 不好的命名
  func (gc *GarbageCollector) Start() error             // 太通用
  func (ad *AdaptiveDecision) Process() error           // 技术化
  ```

### 5.2 注释规范化
- **包注释**：说明包的领域职责
  ```go
  // Package coregc 提供核心垃圾回收功能
  package coregc
  ```

- **类型注释**：说明业务含义
  ```go
  // GarbageCollector 垃圾回收器，负责执行内存回收
  type GarbageCollector struct {
      // id 回收器唯一标识
      id string
      
      // phase 当前GC阶段
      phase GCPhase
  }
  ```

- **方法注释**：说明业务意图和参数含义
  ```go
  // TriggerGC 触发垃圾回收过程
  // 当堆使用率达到配置阈值时调用
  func (gc *GarbageCollector) TriggerGC() error {
      // 实现...
  }
  ```

### 5.3 错误信息统一
- **错误类型**：使用业务语言
  ```go
  // ✅ 好的错误信息
  return fmt.Errorf("堆使用率 %.2f 超过阈值 %.2f", usage, threshold)
  
  // ❌ 不好的错误信息
  return fmt.Errorf("usage > threshold")
  ```

### 5.4 测试命名统一
- **测试文件名**：`{源文件名}_test.go`
- **测试函数名**：`Test{方法名}_{场景}`
  ```go
  func TestTriggerGC_WhenHeapUsageExceedsThreshold(t *testing.T) {
      // 测试：当堆使用率超过阈值时触发GC
  }
  
  func TestTriggerGC_WhenAlreadyRunning(t *testing.T) {
      // 测试：当GC已在运行时拒绝新触发
  }
  ```

## 6. 词汇表维护

### 6.1 新术语添加流程
1. **识别新概念**：在领域建模过程中发现新概念
2. **定义术语**：与领域专家讨论，确定统一名称
3. **添加定义**：更新词汇表，包含英文、中文、定义、上下文
4. **代码应用**：在代码中使用新术语
5. **文档同步**：更新相关文档

### 6.2 术语冲突解决
1. **识别冲突**：发现同一概念有不同名称
2. **利益相关方讨论**：与团队和领域专家协商
3. **选择标准名称**：选择最准确、最一致的名称
4. **重构代码**：逐步替换旧名称
5. **更新文档**：同步所有相关文档

### 6.3 定期审查
- **频率**：每季度进行一次词汇表审查
- **参与者**：开发团队、领域专家、产品经理
- **内容**：检查一致性、准确性、完整性
- **输出**：更新词汇表、代码重构计划
