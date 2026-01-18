# Echo并发调度器设计文档

## 1. 概述

### 1.1 项目背景
Echo语言需要支持真正的多核并发执行，实现类似Go语言的goroutine调度器。通过GMP（Goroutine-Machine-Processor）模型，实现高效的多核并发调度。

### 1.2 核心目标
- **多核并发**：充分利用多核CPU，实现真正的并行执行
- **工作窃取**：动态负载均衡，避免某些核心空闲
- **async/await语义**：提供直观的异步编程接口
- **高性能调度**：用户态调度，减少内核切换开销

### 1.3 技术选型
- **调度模型**：GMP模型（参考Go语言调度器）
- **上下文切换**：ucontext用户态协程切换
- **并发原语**：POSIX线程和同步原语
- **编译目标**：LLVM IR + C运行时库

## 2. 架构设计

### 2.1 GMP模型架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Echo 应用程序                            │
├─────────────────────────────────────────────────────────────┤
│  async/await 语法层                                          │
├─────────────────────────────────────────────────────────────┤
│  Domain层：协程管理、Future/Promise                          │
├─────────────────────────────────────────────────────────────┤
│  Infrastructure层：GMP调度器                                │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  G (Goroutine)   M (Machine)   P (Processor)        │   │
│  │  ┌─────────────┐ ┌──────────┐ ┌──────────────┐      │   │
│  │  │ 协程执行体  │ │ OS线程   │ │ 调度器       │      │   │
│  │  │ 用户代码    │ │ 上下文   │ │ 本地队列     │      │   │
│  │  │ 栈空间      │ │ 亲和性   │ │ 工作窃取     │      │   │
│  │  └─────────────┘ └──────────┘ └──────────────┘      │   │
│  └─────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│  操作系统：CPU核心、内存管理、线程调度                      │
└─────────────────────────────────────────────────────────────┘
```

#### G (Goroutine - 协程)
- **职责**：执行用户代码，维护执行上下文
- **组件**：
  - 执行函数和参数
  - 独立的栈空间
  - 上下文切换状态
  - 绑定到的处理器

#### M (Machine - 机器/OS线程)
- **职责**：提供OS线程执行环境
- **组件**：
  - POSIX线程 (pthread)
  - 线程本地存储
  - CPU亲和性绑定
  - 调度循环

#### P (Processor - 处理器)
- **职责**：管理协程队列，实现调度逻辑
- **组件**：
  - 本地协程队列 (256槽位)
  - 工作窃取算法
  - 负载均衡
  - 统计信息

### 2.2 工作窃取算法

#### 算法原理
当处理器本地队列为空时，从其他处理器"窃取"协程来执行，实现负载均衡。

#### 窃取策略
1. **LIFO策略**：从受害者队列头部窃取（后进先出）
2. **随机选择**：随机选择受害处理器，避免热点
3. **全局兜底**：所有本地队列为空时，从全局队列获取

#### 性能优势
- **减少锁竞争**：优先使用本地队列，减少跨处理器同步
- **提高缓存效率**：协程在同一处理器上连续执行
- **动态负载均衡**：自动平衡各处理器负载

### 2.3 生命周期管理

#### 协程生命周期
```
创建 (spawn) → 就绪 (ready) → 运行 (running) → 挂起 (suspended) → 完成 (completed)
     ↓            ↓            ↓              ↓                   ↓
 分配ID       加入队列     获得执行       await等待        释放资源
 分配栈       设置状态     上下文切换     保存上下文       统计信息
 绑定处理器   负载均衡     更新统计       加入等待队列     内存回收
```

#### Future生命周期
```
创建 (new) → 等待 (pending) → 完成 (resolved/rejected)
     ↓            ↓                     ↓
 分配ID       协程等待                唤醒等待协程
 设置回调     加入等待队列            传递结果/错误
 初始化状态   阻塞当前协程            继续执行
```

## 3. 核心组件设计

### 3.1 调度器核心算法

#### 初始化算法
```c
void global_scheduler_init() {
    // 1. 检测CPU核心数
    num_cpus = sysconf(_SC_NPROCESSORS_ONLN);
    num_cpus = min(num_cpus, MAX_PROCESSORS);

    // 2. 创建处理器和机器
    for (i = 0; i < num_cpus; i++) {
        processor_init(&processors[i], i);
        machine_init(&machines[i], i, &processors[i]);
    }

    // 3. 启动机器线程
    for (i = 0; i < num_cpus; i++) {
        pthread_create(&machines[i].thread, NULL, machine_worker, &machines[i]);
    }

    // 4. 初始化同步原语
    pthread_mutex_init(&global_lock, NULL);
    pthread_cond_init(&work_available, NULL);
}
```

#### 调度循环算法
```c
void* machine_worker(void* arg) {
    Machine* m = (Machine*)arg;
    Processor* p = m->bound_processor;

    while (global_scheduler.is_running) {
        // 1. 获取可执行协程
        Coroutine* co = processor_get_work(p);

        if (co) {
            // 2. 执行协程
            co->state = COROUTINE_RUNNING;
            co->bound_processor = p;

            // 上下文切换执行协程
            if (swapcontext(&m->context, &co->context) == 0) {
                // 协程完成处理
                processor_handle_completion(p, co);
            }
        } else {
            // 3. 没有工作，等待
            pthread_mutex_lock(&global_scheduler.global_lock);
            pthread_cond_wait(&global_scheduler.work_available,
                            &global_scheduler.global_lock);
            pthread_mutex_unlock(&global_scheduler.global_lock);
        }
    }

    return NULL;
}
```

#### 工作获取算法
```c
Coroutine* processor_get_work(Processor* p) {
    // 1. 优先从本地队列获取
    Coroutine* co = processor_pop_local(p);
    if (co) return co;

    // 2. 本地队列为空，尝试工作窃取
    co = processor_steal_work(p);
    if (co) return co;

    // 3. 窃取失败，从全局队列获取
    return global_scheduler_pop();
}
```

### 3.2 并发原语实现

#### spawn实现
```c
void* coroutine_spawn(void* entry_func, int32_t arg_count, void* args, void* future_ptr) {
    Future* future = (Future*)future_ptr;

    // 1. 创建协程
    Coroutine* co = coroutine_create((void(*)(void*))entry_func, future, 64*1024);

    // 2. 推送到调度器
    global_scheduler_push(co);

    // 3. 唤醒等待的机器
    pthread_cond_broadcast(&global_scheduler.work_available);

    return future;
}
```

#### await实现
```c
void* coroutine_await(void* future_ptr) {
    Future* future = (Future*)future_ptr;

    if (future->state == FUTURE_RESOLVED) {
        // Future已完成，直接返回结果
        return future->value;
    }

    // 1. 设置等待关系
    future->waiting_coroutine = current_coroutine;

    // 2. 挂起当前协程
    current_coroutine->state = COROUTINE_SUSPENDED;

    // 3. 让出控制权
    processor_schedule(current_coroutine->bound_processor);

    // 4. 被唤醒后，返回结果
    return future->value;
}
```

### 3.3 内存管理设计

#### 协程栈管理
- **固定栈大小**：64KB初始栈，避免频繁扩容
- **栈分配策略**：mmap分配，保证页对齐
- **栈保护**：设置保护页，防止栈溢出

#### Future内存管理
- **引用计数**：跟踪Future的引用关系
- **垃圾回收**：定期清理未使用的Future
- **内存池**：复用Future结构体，减少分配开销

## 4. 接口设计

### 4.1 应用层接口

#### AgentService接口
```go
type AgentService interface {
    // 异步消息处理
    ProcessAsyncMessage(ctx context.Context, req AsyncRequest) (*Future[Response], error)

    // 并发任务执行
    ExecuteConcurrentTasks(ctx context.Context, tasks []Task) (*Future[[]Result], error)

    // 协程池管理
    ManageCoroutinePool(ctx context.Context, config PoolConfig) error
}
```

#### 异步任务接口
```go
type AsyncTask interface {
    // 执行异步任务
    Execute(ctx context.Context) (interface{}, error)

    // 获取任务ID
    GetID() string

    // 获取超时时间
    GetTimeout() time.Duration
}
```

### 4.2 领域层接口

#### AsyncExecutor接口
```go
type AsyncExecutor interface {
    // 执行异步任务
    Execute(task AsyncTask) (*Future[interface{}], error)

    // 取消任务执行
    Cancel(taskID string) error

    // 查询任务状态
    QueryStatus(taskID string) (TaskStatus, error)
}
```

#### Future接口
```go
type Future[T any] interface {
    // 等待结果
    Await() (T, error)

    // 等待带超时
    AwaitWithTimeout(timeout time.Duration) (T, error)

    // 检查是否完成
    IsCompleted() bool

    // 获取结果（非阻塞）
    TryGet() (T, error, bool)
}
```

### 4.3 基础设施层接口

#### GMPScheduler接口
```go
type GMPScheduler interface {
    // 启动调度器
    Start() error

    // 停止调度器
    Stop() error

    // 获取统计信息
    GetStats() (*SchedulerStats, error)

    // 设置调度策略
    SetPolicy(policy SchedulingPolicy) error
}
```

#### 工作窃取器接口
```go
type WorkStealer interface {
    // 尝试窃取工作
    Steal() (Coroutine, bool)

    // 获取受害者列表
    GetVictims() []Processor

    // 记录窃取统计
    RecordSteal(success bool)
}
```

## 5. 性能优化

### 5.1 缓存优化

#### CPU缓存亲和性
- **线程绑定**：每个M绑定到特定CPU核心
- **内存分配**：在对应NUMA节点分配内存
- **数据局部性**：减少跨CPU缓存同步

#### 协程调度优化
- **批量调度**：一次获取多个协程，减少锁竞争
- **自旋等待**：短暂等待新任务，避免频繁上下文切换
- **优先级调度**：高优先级协程优先执行

### 5.2 锁优化

#### 细粒度锁
```c
// 每个处理器有独立锁
pthread_mutex_t processor_locks[MAX_PROCESSORS];

// 全局锁只保护全局队列
pthread_mutex_t global_lock;
```

#### 锁竞争减少
- **读写锁**：统计信息使用读写锁
- **CAS操作**：无锁队列操作
- **分层锁**：不同层级使用不同锁

### 5.3 内存优化

#### 对象池
```c
// 协程对象池
Coroutine* coroutine_pool_alloc();
void coroutine_pool_free(Coroutine* co);

// Future对象池
Future* future_pool_alloc();
void future_pool_free(Future* f);
```

#### 栈复用
- **栈缓存**：缓存已完成的协程栈
- **栈压缩**：协程挂起时压缩栈空间
- **栈预分配**：预分配常用大小的栈

## 6. 测试策略

### 6.1 单元测试

#### 调度器测试
```go
func TestWorkStealing(t *testing.T) {
    // 创建多个处理器
    p1 := NewProcessor(1)
    p2 := NewProcessor(2)

    // 向p1添加大量协程
    for i := 0; i < 100; i++ {
        co := NewCoroutine(func() { /* 简单任务 */ })
        p1.PushLocal(co)
    }

    // p2应该能够窃取协程
    stolen := p2.Steal()
    assert.True(t, stolen != nil)
}
```

#### Future测试
```go
func TestFutureAwait(t *testing.T) {
    future := NewFuture()

    // 在另一个协程中resolve
    go func() {
        time.Sleep(10 * time.Millisecond)
        future.Resolve("result")
    }()

    // await应该阻塞并返回结果
    result := future.Await()
    assert.Equal(t, "result", result)
}
```

### 6.2 集成测试

#### 并发性能测试
```go
func BenchmarkConcurrentTasks(b *testing.B) {
    // 创建N个并发任务
    tasks := make([]AsyncTask, b.N)
    for i := 0; i < b.N; i++ {
        tasks[i] = NewComputeTask(i)
    }

    b.ResetTimer()
    results := ExecuteConcurrent(tasks)

    // 验证所有任务都正确执行
    assert.Len(b, results, b.N)
}
```

#### 工作窃取测试
```go
func TestLoadBalancing(t *testing.T) {
    scheduler := NewGMPScheduler()

    // 创建不均衡负载
    for i := 0; i < 1000; i++ {
        if i < 100 {
            // 处理器1获得很少任务
            scheduler.ScheduleToProcessor(0, NewTask())
        } else {
            // 处理器2获得大量任务
            scheduler.ScheduleToProcessor(1, NewTask())
        }
    }

    // 运行一段时间
    time.Sleep(100 * time.Millisecond)

    // 检查负载均衡
    stats := scheduler.GetStats()
    p1Load := float64(stats.ProcessorStats[0].ExecutedTasks)
    p2Load := float64(stats.ProcessorStats[1].ExecutedTasks)

    // 负载应该相对均衡（允许10%的偏差）
    ratio := p1Load / p2Load
    assert.True(t, ratio > 0.9 && ratio < 1.1)
}
```

### 6.3 压力测试

#### 高并发测试
- **协程数量**：10,000+并发协程
- **CPU利用率**：接近100%多核利用
- **内存使用**：监控内存泄漏
- **响应时间**：P99延迟控制

#### 长时间运行测试
- **稳定性**：运行24小时不崩溃
- **资源使用**：CPU/内存稳定
- **性能衰减**：无性能退化

## 7. 监控和调试

### 7.1 性能指标

#### 调度器指标
- **协程创建率**：每秒创建协程数量
- **协程完成率**：每秒完成协程数量
- **队列长度**：各个处理器的本地队列长度
- **窃取成功率**：工作窃取的成功率

#### 运行时指标
- **上下文切换次数**：每秒上下文切换次数
- **CPU利用率**：每个处理器的CPU使用率
- **内存使用量**：协程栈和Future的内存使用
- **锁竞争情况**：锁等待时间和冲突次数

### 7.2 调试支持

#### 协程追踪
```c
// 协程执行日志
#define COROUTINE_TRACE(co, action) \
    fprintf(stderr, "[COROUTINE] %llu %s at %s:%d\n", \
            co->id, action, __FILE__, __LINE__)
```

#### 死锁检测
```c
// 定期检查死锁
void deadlock_detector() {
    for (int i = 0; i < num_processors; i++) {
        Processor* p = &processors[i];

        if (p->local_queue_size == 0 &&
            p->stealing_attempts > DEADLOCK_THRESHOLD) {
            fprintf(stderr, "Potential deadlock detected on processor %d\n", i);
        }
    }
}
```

#### 性能分析
```c
// 性能分析工具
typedef struct {
    uint64_t total_switches;
    uint64_t total_steals;
    uint64_t total_spawns;
    double avg_switch_time;
    double avg_steal_time;
} PerformanceStats;
```

## 8. 部署和运维

### 8.1 配置管理

#### 调度器配置
```yaml
scheduler:
  max_processors: 8      # 最大处理器数量
  local_queue_size: 256  # 本地队列大小
  global_queue_size: 1024 # 全局队列大小
  stack_size: 65536      # 默认栈大小 (64KB)
  steal_threshold: 10    # 窃取尝试阈值
```

#### 性能调优
```yaml
performance:
  numa_aware: true       # NUMA感知调度
  spin_before_wait: true # 自旋等待
  batch_scheduling: true # 批量调度
  stack_pool: true       # 栈池复用
```

### 8.2 故障处理

#### 常见故障
- **死锁**：协程互相等待，解决方案：死锁检测和超时
- **栈溢出**：协程栈空间不足，解决方案：动态栈扩容
- **内存泄漏**：协程或Future未正确释放，解决方案：引用计数和GC

#### 恢复策略
- **优雅降级**：故障时退化为单线程执行
- **自动重启**：检测到严重故障时重启调度器
- **负载削减**：高负载时拒绝新任务

### 8.3 扩展性考虑

#### 动态扩缩容
- **CPU热插拔**：运行时添加/移除CPU核心
- **负载感知**：根据系统负载动态调整处理器数量
- **资源限制**：控制内存和CPU使用上限

#### 多进程支持
- **进程间通信**：协程可以跨进程调度
- **负载均衡**：多进程间的任务分发
- **故障隔离**：进程级别的故障隔离

## 9. 总结

### 9.1 设计优势

1. **高性能**：用户态调度，减少内核切换开销
2. **可扩展**：多核并发，充分利用CPU资源
3. **负载均衡**：工作窃取算法，动态平衡负载
4. **易编程**：async/await语义，直观易用

### 9.2 技术挑战

1. **复杂性**：GMP模型实现复杂，需要精细的同步控制
2. **调试困难**：并发程序调试困难，需要完善的监控工具
3. **内存管理**：协程栈管理复杂，需要高效的内存分配策略

### 9.3 未来优化方向

1. **NUMA优化**：感知内存亲和性，优化跨NUMA访问
2. **垃圾回收**：实现协程友好的GC算法
3. **实时调度**：支持实时任务的优先级调度
4. **分布式调度**：支持跨机器的协程调度
