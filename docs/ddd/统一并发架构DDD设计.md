# 统一并发架构DDD设计文档

## 🎯 概述

### 项目背景
Echo语言需要支持真正的多核并发执行，实现类似Go语言的goroutine调度器。通过GMP（Goroutine-Machine-Processor）模型，实现高效的多核并发调度。

### 核心目标
- **多核并发**：充分利用多核CPU，实现真正的并行执行
- **工作窃取**：动态负载均衡，避免某些核心空闲
- **async/await语义**：提供直观的异步编程接口
- **高性能调度**：用户态调度，减少内核切换开销

### 技术选型
- **调度模型**：GMP模型（参考Go语言调度器）
- **上下文切换**：手写汇编实现的协程上下文切换
- **并发原语**：POSIX线程和同步原语
- **编译目标**：LLVM IR + C运行时库

---

## 📋 领域分析：7步法

### 第一步：词性分析（5分钟）

从并发架构中提取：

**名词**：
- Task, Scheduler, Future, Coroutine, Processor, Machine, ReadyQueue, EventQueue
- spawn, async, await, select, channel, goroutine, context, stack
- WorkStealing, LoadBalancing, ContextSwitching, Polling

**动词**：
- spawn, await, select, schedule, execute, switch, poll, wake, steal, balance

---

### 第二步：概念聚类（10分钟）

**第一轮聚类**：

1. **任务管理**：
   - Task, Future, spawn, await, execute

2. **调度管理**：
   - Scheduler, Processor, Machine, ReadyQueue, schedule

3. **并发原语**：
   - Coroutine, ContextSwitching, async, select

4. **资源管理**：
   - stack, LoadBalancing, WorkStealing

5. **事件驱动**：
   - EventQueue, wake, poll, channel

---

### 第三步：边界识别（15分钟）

```mermaid
graph TD
    subgraph "用户代码"
        A[spawn foo()]
        B[await bar()]
        C[select { case <-ch: ... }]
    end

    subgraph "任务领域"
        D[Task聚合根]
        E[Future值对象]
        F[TaskSpawner服务]
    end

    subgraph "调度领域"
        G[Scheduler聚合根]
        H[Processor实体]
        I[Machine实体]
        J[Scheduler服务]
    end

    subgraph "协程领域"
        K[Coroutine聚合根]
        L[Context值对象]
        M[ContextSwitcher服务]
    end

    subgraph "事件领域"
        N[EventQueue聚合根]
        O[Channel实体]
        P[EventMultiplexer服务]
    end

    A --> D
    B --> E
    C --> N
    D --> G
    G --> K
    N --> G
```

---

### 第四步：聚合根识别（20分钟）

根据名词和动词关系，识别聚合根：

1. **Task（任务聚合根）**
   - 包含：Future, Coroutine, Stack
   - 职责：管理任务生命周期，协调异步计算

2. **Scheduler（调度器聚合根）**
   - 包含：Processor[], ReadyQueue, EventQueue
   - 职责：管理处理器，调度任务执行

3. **Coroutine（协程聚合根）**
   - 包含：Context, Stack, State
   - 职责：管理协程上下文和状态

4. **Channel（通道聚合根）**
   - 包含：Buffer, SenderQueue, ReceiverQueue
   - 职责：管理通道通信和同步

---

### 第五步：值对象和领域服务（15分钟）

**值对象（不可变，无标识）**：
1. **Future**：异步计算结果（状态+值）
2. **Context**：协程上下文快照（寄存器+栈指针）
3. **PollResult**：轮询结果（Ready/Pending+值）

**领域服务（协调多个实体）**：
1. **TaskSpawner**：创建和初始化任务
2. **SchedulerService**：任务调度和负载均衡
3. **ContextSwitcher**：协程上下文切换
4. **EventMultiplexer**：事件多路复用（select）

---

### 第六步：分层架构（10分钟）

**应用层**：
```
├── ConcurrencyService (spawn/async入口)
├── FutureService (await/select入口)
└── ChannelService (通道操作入口)
```

**领域层**：
```
├── task/
│   ├── Task (聚合根)
│   ├── Future (值对象)
│   ├── TaskSpawner (领域服务)
│   └── TaskRepository (仓储接口)
├── scheduler/
│   ├── Scheduler (聚合根)
│   ├── Processor (实体)
│   ├── Machine (实体)
│   ├── SchedulerService (领域服务)
│   └── LoadBalancer (领域服务)
├── coroutine/
│   ├── Coroutine (聚合根)
│   ├── Context (值对象)
│   ├── ContextSwitcher (领域服务)
│   └── StackAllocator (领域服务)
└── channel/
    ├── Channel (聚合根)
    ├── Message (值对象)
    ├── ChannelService (领域服务)
    └── EventMultiplexer (领域服务)
```

**基础设施层**：
```
├── threading/ (POSIX线程实现)
├── memory/ (栈分配实现)
└── timing/ (定时器实现)
```

---

### 第七步：领域对象草图（15分钟）

```c
// 1. Task 聚合根草图
typedef struct Task {
    uint64_t id;
    TaskStatus status;        // READY, WAITING, FINISHED
    Future* future;           // 当前等待的异步计算
    Coroutine* coroutine;     // 关联的协程
    void* stack;              // 执行栈
    Task* next;               // 队列指针

    // 领域方法
    void task_execute(Task* self);
    bool task_is_complete(Task* self);
    void task_set_future(Task* self, Future* future);
} Task;

// 2. Future 值对象草图
typedef enum { PENDING, READY, COMPLETED } PollStatus;

typedef struct PollResult {
    PollStatus status;
    void* value;
} PollResult;

typedef struct Future {
    uint64_t id;
    void* data;  // Future子类数据
    PollResult (*poll)(struct Future* self, Task* task);
} Future;

// 3. Scheduler 聚合根草图
typedef struct Processor {
    uint32_t id;
    Task* local_queue_head;
    Task* local_queue_tail;
    uint32_t queue_size;
    pthread_mutex_t lock;
} Processor;

typedef struct Machine {
    uint32_t id;
    pthread_t thread;
    Processor* processor;
    int is_running;
} Machine;

typedef struct Scheduler {
    Processor* processors;
    uint32_t num_processors;
    Task* global_queue;
    pthread_mutex_t global_lock;
    // ... 调度状态

    // 领域方法
    void scheduler_add_task(Scheduler* self, Task* task);
    Task* scheduler_get_work(Scheduler* self, Processor* proc);
    void scheduler_work_steal(Scheduler* self, Processor* thief);
} Scheduler;

// 4. Coroutine 聚合根草图
typedef struct Coroutine {
    uint64_t id;
    CoroutineState state;
    coro_context_t context;
    char* stack;
    size_t stack_size;
    void (*entry_point)(void*);  // 协程入口函数
    void* arg;                   // 入口函数参数
    Processor* bound_processor;  // 绑定到的处理器
    Task* task;                  // 关联的任务
    Coroutine* next;             // 用于队列

    // 领域方法
    void coroutine_resume(Coroutine* self);
    void coroutine_suspend(Coroutine* self);
    bool coroutine_is_complete(Coroutine* self);
} Coroutine;
```

---

## 📋 核心组件设计

### 1. 任务控制块 (Task)

**职责**：
- 封装并发执行单元
- 管理任务生命周期
- 关联Future和协程

**关键属性**：
- ID：唯一标识
- Status：READY/WAITING/FINISHED
- Future：异步计算结果
- Coroutine：执行上下文
- Stack：执行栈

### 2. 调度器 (Scheduler)

**职责**：
- 管理处理器和任务队列
- 实现工作窃取算法
- 协调多核执行

**关键组件**：
- Processors：处理器数组
- Global Queue：全局任务队列
- Local Queues：每个处理器的本地队列

### 3. Future抽象

**职责**：
- 统一异步计算接口
- 支持状态轮询
- 实现事件驱动模型

**接口设计**：
```c
typedef struct Future Future;
struct Future {
    void* data;
    PollResult (*poll)(Future* self, Task* task);
};
```

### 4. 协程上下文

**职责**：
- 保存/恢复执行状态
- 实现用户态切换
- 管理栈空间

**关键技术**：
- 手写汇编保存寄存器
- 栈指针管理和切换
- 跨平台兼容性

---

## 🔄 并发原语映射

### Spawn映射
```
spawn foo() → TaskSpawner.create_task(foo)
           ↓
TaskSpawner → Scheduler.add_task()
           ↓
Scheduler → Processor.local_queue
```

### Async/Await映射
```
async func bar() → Future.new(bar_executor)
await future → future.poll() + task.suspend()
```

### Select映射
```
select { case <-ch: ... } → EventMultiplexer.poll_channels()
                         ↓
EventMultiplexer → multiple Future.poll()
```

---

## 🏗️ 目录结构设计

```
runtime/
├── task/                    # 任务管理
│   ├── task.h
│   ├── task.c
│   ├── future.h
│   ├── future.c
│   └── task_spawner.h
├── scheduler/               # 调度器管理
│   ├── scheduler.h
│   ├── scheduler.c
│   ├── processor.h
│   ├── processor.c
│   ├── machine.h
│   ├── machine.c
│   └── load_balancer.h
├── coroutine/               # 协程管理
│   ├── coroutine.h
│   ├── coroutine.c
│   ├── context.h
│   ├── context.c           # 平台相关实现
│   └── stack_allocator.h
├── channel/                 # 通道通信
│   ├── channel.h
│   ├── channel.c
│   ├── message.h
│   ├── event_queue.h
│   └── event_multiplexer.h
└── runtime.h               # 统一接口
```

---

## 📋 实施计划

### 第一阶段：核心基础设施（2天）

**任务1：实现任务控制块**
- [ ] 定义Task结构体
- [ ] 实现任务生命周期管理
- [ ] 关联Future和协程

**任务2：实现基础调度器**
- [ ] 单核调度器框架
- [ ] 任务队列管理
- [ ] 基本的调度循环

### 第二阶段：Future抽象（2天）

**任务3：实现Future接口**
- [ ] 定义Future trait
- [ ] 实现PollResult
- [ ] 支持状态轮询

**任务4：修复spawn实现**
- [ ] 创建Task而非协程
- [ ] 任务放入调度队列
- [ ] 正确返回Future

### 第三阶段：Async/Await状态机（3天）

**任务5：编译器支持**
- [ ] async函数编译为Future
- [ ] await点生成状态机
- [ ] 正确的executor生成

**任务6：运行时集成**
- [ ] Future.resolve()实现
- [ ] 任务唤醒机制
- [ ] 状态机执行逻辑

### 第四阶段：Select多路复用（2天）

**任务7：通道机制完善**
- [ ] 完整的send/receive
- [ ] 阻塞和非阻塞操作
- [ ] 缓冲区管理

**任务8：Select实现**
- [ ] 多路事件等待
- [ ] 复合Future管理
- [ ] 事件取消和清理

### 第五阶段：多核GMP（3天）

**任务9：GMP架构**
- [ ] 处理器和机器管理
- [ ] 多线程调度器
- [ ] 工作窃取算法

**任务10：性能优化**
- [ ] 负载均衡
- [ ] 缓存友好性
- [ ] 内存管理优化

---

## ✅ 验收标准

### 架构层面
- [ ] 所有并发原语基于统一Task模型
- [ ] 调度器支持多核工作窃取
- [ ] Future抽象统一异步接口

### 功能层面
- [ ] spawn创建真正的并发任务
- [ ] async/await实现状态机等待
- [ ] select支持多路事件复用

### 性能层面
- [ ] 多核CPU得到充分利用
- [ ] 上下文切换开销最小化
- [ ] 内存使用高效

---

## 📚 参考资料

- Go语言GMP调度器设计
- Rust异步运行时架构
- C++协程TS标准
- POSIX线程编程
