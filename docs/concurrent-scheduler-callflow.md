# Echo并发调度器代码调用流程

## 概述

基于Go语言GMP（Goroutine-Machine-Processor）模型设计的多核并发调度器，实现了真正的多核并发执行。

**核心设计原则**：
1. **GMP模型**：G(协程)-M(机器/OS线程)-P(处理器)三层架构
2. **工作窃取**：空闲处理器从其他处理器窃取任务
3. **本地队列优先**：每个处理器维护本地协程队列
4. **全局队列兜底**：本地队列为空时从全局队列获取任务

**技术特点**：
- 多核并发：每个CPU核心一个处理器和机器线程
- 工作窃取：负载均衡，避免某些核心空闲
- 用户态调度：协程上下文切换，无需内核介入
- 事件驱动：Future/Promise模式，支持async/await

---

## 整体流程图

```
用户调用async/await
    ↓
[Application 层] AgentService.ProcessAsyncMessage()
    ↓
[Domain 层] AsyncExecutor.Execute()
    ├── 创建协程并生成Future
    │   ├── GenerateFuture() → Future实例
    │   ├── CreateCoroutine() → 协程实例
    │   └── CoroutineSpawn() → 启动协程执行
    ├── 立即返回Future（非阻塞）
    │   └── Future{Promise: pending}
    └── 协程在后台执行
        └── 调度器分配到处理器执行

协程执行流程
    ↓
[Infrastructure 层] GMP调度器
    ├── 全局队列分发
    │   ├── Push到GlobalQueue → 唤醒等待的机器
    │   └── 信号量通知WorkAvailable
    └── 处理器本地调度
        ├── 本地队列优先
        │   ├── PopLocalWork() → 本地协程
        │   ├── 执行协程逻辑 → 业务代码
        │   └── 协程完成 → 释放资源
        ├── 本地队列为空时
        │   └── 工作窃取 → StealWorkFromOthers()
        └── 所有队列为空时
            └── 等待新任务 → CondWait(WorkAvailable)
```

---

## 详细流程分析

### 阶段 1：应用层入口处理

##### 1.1 AgentService.ProcessAsyncMessage()

**位置**：`internal/modules/agent/application/agent_service.go`（已设计）

**职责**：
- 接收用户异步请求并转换为协程任务
- 参数验证（基于已设计的验证规则）
- 调用领域层的异步执行器（基于已设计的领域服务接口）

**详细流程**：
```
AgentService.ProcessAsyncMessage(request AsyncRequest) → Future[Response]
    ↓
1. 请求验证（基于已设计的验证规则）
    ├── 必填字段检查（目标、参数、超时时间）
    ├── 业务规则校验（权限、配额限制）
    └── 参数格式验证（JSON/Protocol Buffers）
    ↓
2. 创建协程执行上下文
    ├── 生成唯一任务ID
    ├── 设置执行超时
    ├── 绑定用户上下文
    ↓
3. 调用领域服务（基于已设计的领域服务接口）
    AsyncExecutor.Execute(AsyncTask) → Future[Result]
    ↓
4. 返回Future给调用方（非阻塞）
    ├── Future状态：PENDING
    ├── Promise：异步执行承诺
    └── 回调注册：结果通知机制
```

---

### 阶段 2：领域层协程管理

##### 2.1 AsyncExecutor.Execute()

**位置**：`internal/modules/agent/domain/services/async_executor.go`（已设计）

**职责**：
- 管理协程生命周期和Future状态
- 协调多协程间的依赖关系
- 处理协程间的通信和同步

**详细流程**：
```
AsyncExecutor.Execute(task AsyncTask) → Future[Result]
    ↓
1. 创建Future对象（Promise模式）
    ├── 生成Future ID
    ├── 设置初始状态：PENDING
    ├── 绑定执行任务
    └── 注册状态监听器
    ↓
2. 创建协程实例
    ├── 分配协程ID
    ├── 设置执行函数：task.Execute()
    ├── 配置栈大小（默认64KB）
    └── 绑定处理器（负载均衡）
    ↓
3. 启动协程执行（spawn操作）
    ├── 添加到调度器队列
    │   ├── 优先本地队列
    │   ├── 本地队列满时推送到全局队列
    │   └── 唤醒等待的处理器
    ├── 设置协程状态：READY
    └── 立即返回Future（非阻塞）
```

##### 2.2 协程生命周期管理

```
协程创建 → 加入调度队列 → 处理器调度 → 执行业务逻辑 → 完成/异常处理
    ↓           ↓           ↓           ↓               ↓
GenerateID  PushQueue   GetWork     RunTask       ResolveFuture
分配唯一ID   队列管理     负载均衡    栈切换        结果通知
栈分配       优先级处理  工作窃取    上下文保存    回调触发
```

---

### 阶段 3：基础设施层GMP调度

##### 3.1 GMP调度器架构

**位置**：`runtime/coroutine_runtime.c`（已实现）

**核心组件**：
- **G (Goroutine)**：协程，包含执行上下文和栈
- **M (Machine)**：OS线程，执行协程调度循环
- **P (Processor)**：调度器，管理协程队列

**详细流程**：
```
GMP调度器初始化
    ↓
1. 检测CPU核心数
    ├── sysconf(_SC_NPROCESSORS_ONLN)
    ├── 限制最大处理器数：MAX_PROCESSORS=8
    └── 创建相应数量的P和M
    ↓
2. 初始化处理器(P)
    ├── 为每个CPU核心创建Processor
    ├── 初始化本地队列（LOCAL_QUEUE_SIZE=256）
    ├── 设置处理器ID和锁
    └── 绑定到对应的Machine
    ↓
3. 初始化机器(M)
    ├── 创建OS线程（pthread_create）
    ├── 设置线程亲和性（绑定到特定CPU核心）
    ├── 初始化上下文
    └── 启动调度循环
    ↓
4. 启动全局调度器
    ├── 初始化全局队列（GLOBAL_QUEUE_SIZE=1024）
    ├── 设置同步原语（锁、条件变量）
    └── 标记调度器为运行状态
```

##### 3.2 协程调度循环

```
机器(M)调度循环
    ↓
while (scheduler.is_running) {
    ↓
    1. 获取可执行协程
        ├── 优先从本地队列获取
        │   ├── processor_pop_local() → 本地协程
        │   └── 更新队列统计信息
        ├── 本地队列为空时
        │   └── 尝试工作窃取：processor_steal_work()
        └── 仍然没有工作时
            └── 等待新任务：pthread_cond_wait()
    ↓
    2. 执行协程
        ├── 设置协程状态：RUNNING
        ├── 上下文切换：swapcontext()
        ├── 执行协程函数体
        └── 处理协程完成/异常
    ↓
    3. 协程完成处理
        ├── 解析Future结果
        ├── 通知等待协程
        ├── 释放协程资源
        └── 继续下一个协程
}
```

##### 3.3 工作窃取算法

**位置**：`processor_steal_work()`函数

**详细流程**：
```
processor_steal_work(thief_processor) → 窃取的协程
    ↓
1. 遍历所有其他处理器
    ├── for (i = 0; i < num_processors; i++)
    ├── 跳过自己：if (victim == thief) continue
    └── 尝试从受害者窃取
    ↓
2. 从受害者本地队列窃取
    ├── 获取受害者锁：pthread_mutex_lock(victim->lock)
    ├── 检查是否有可窃取协程：victim->local_queue_size > 0
    ├── 从队列头部窃取（LIFO策略）
    │   ├── 调整队列头指针
    │   ├── 减少队列大小
    │   └── 获取协程指针
    └── 释放受害者锁
    ↓
3. 窃取成功处理
    ├── 重新绑定协程到窃取者：stolen->bound_processor = thief
    ├── 记录窃取统计
    └── 返回窃取的协程
    ↓
4. 窃取失败时
    └── 从全局队列获取：global_scheduler_pop()
```

---

### 阶段 4：并发执行优化

##### 4.1 多核负载均衡

```
负载均衡策略
    ↓
1. 协程创建时选择处理器
    ├── 轮询调度：round_robin_counter++
    ├── 负载最轻：min(local_queue_size)
    └── 缓存亲和：last_processor_for_task_type
    ↓
2. 运行时动态平衡
    ├── 定期检查队列长度
    ├── 触发工作窃取阈值
    └── 全局队列再平衡
    ↓
3. NUMA感知调度
    ├── CPU亲和性绑定
    ├── 内存局部性优化
    └── 跨NUMA节点通信最小化
```

##### 4.2 性能监控

```
性能指标收集
    ↓
1. 协程统计
    ├── 创建协程数
    ├── 完成协程数
    ├── 平均执行时间
    └── 协程队列长度
    ↓
2. 处理器统计
    ├── 本地队列大小
    ├── 窃取成功次数
    ├── 上下文切换次数
    └── CPU利用率
    ↓
3. 全局统计
    ├── 全局队列大小
    ├── 工作窃取频率
    └── 系统负载因子
```

---

## 关键数据流

### 1. 协程创建到执行

**描述**：从用户发起async调用到协程实际执行的完整路径

```
用户代码：await asyncTask()
    ↓
[Application] AgentService.ProcessAsyncMessage()
    ↓
[Domain] AsyncExecutor.Execute()
    ├── Future future = Future.new()
    ├── Coroutine co = Coroutine.create(task_func)
    └── GMP.spawn(co, future)
    ↓
[Infrastructure] GMP调度器
    ├── 推送到处理器本地队列
    │   ├── processor_push_local(p, co)
    │   └── 唤醒等待的机器：pthread_cond_signal()
    └── 机器调度循环获取执行
        ├── processor_pop_local(p) → co
        ├── 上下文切换执行协程
        └── 协程完成时resolve Future
```

### 2. 工作窃取负载均衡

**描述**：空闲处理器从忙碌处理器窃取工作的机制

```
处理器发现本地队列为空
    ↓
processor_pop_local(current_p) → NULL（队列为空）
    ↓
触发工作窃取：processor_steal_work(current_p)
    ↓
遍历其他处理器寻找工作
    ├── 检查处理器1：victim1->local_queue_size > 0
    │   ├── 窃取协程：steal_coroutine_from(victim1)
    │   └── 返回窃取的协程
    ├── 检查处理器2：victim2->local_queue_size == 0
    │   └── 跳过，继续下一个
    └── 所有本地队列都空
        └── 从全局队列获取：global_scheduler_pop()
    ↓
窃取成功后立即执行
    ├── 设置协程绑定：stolen->bound_processor = current_p
    ├── 上下文切换执行
    └── 更新窃取统计
```

### 3. Future异步通信

**描述**：协程间的异步通信和结果传递

```
协程A：Future f = spawn(taskB)
    ↓
协程B异步执行
    ├── 协程B执行业务逻辑
    ├── 计算结果：result
    └── Future.resolve(f, result)
    ↓
Future状态变更通知
    ├── 查找等待协程：waiting_co = f->waiting_coroutine
    ├── 设置等待协程为就绪状态
    └── 推送到处理器队列：processor_push_local(waiting_co->processor, waiting_co)
    ↓
协程A被唤醒执行
    ├── 调度器切换到协程A
    ├── 协程A从Future获取结果：result = await f
    └── 继续执行后续逻辑
```

---

## 各层职责总结

### Application 层：业务编排
- **AgentService**：接收异步请求，创建协程任务
- **请求验证**：参数校验、权限检查、配额控制
- **Future管理**：创建Promise，处理异步结果
- **错误处理**：超时处理、重试逻辑、降级策略

### Domain 层：并发语义
- **AsyncExecutor**：协程生命周期管理，Future状态维护
- **协程协调**：多协程依赖关系，通信协议
- **并发控制**：同步原语，竞态条件处理
- **业务规则**：并发业务逻辑约束

### Infrastructure 层：GMP调度
- **GMP调度器**：多核并发调度，工作窃取算法
- **协程管理**：创建、销毁、上下文切换
- **队列管理**：本地队列、全局队列、负载均衡
- **同步原语**：锁、条件变量、信号量

---

## 设计一致性验证

### Domain 层验证
- [x] 协程实体正确封装状态和上下文
- [x] Future值对象正确实现Promise模式
- [x] 领域服务正确协调协程间通信
- [x] 聚合边界遵守GMP模型约束

### Application 层验证
- [x] 应用服务正确调用领域异步执行器
- [x] Future返回结构符合异步编程模式
- [x] 事务边界设置合理（无事务跨越协程）
- [x] 异常处理符合异步错误传播模式

### Infrastructure 层验证
- [x] GMP调度器正确实现多核并发
- [x] 工作窃取算法有效平衡负载
- [x] 上下文切换正确保存协程状态
- [x] 同步原语正确保护共享数据

---

## 关键设计决策回顾

### 1. GMP模型选择
**决策**：采用Go语言的GMP并发模型
**原因**：Go的GMP模型经过多年优化，性能卓越，支持真正的多核并发
**影响**：需要复杂的调度器实现，但能充分利用多核CPU

### 2. 工作窃取算法
**决策**：实现LIFO工作窃取策略
**原因**：LIFO有利于缓存局部性，减少上下文切换开销
**验证**：窃取成功时立即执行，提高CPU利用率

### 3. 用户态调度
**决策**：使用ucontext进行用户态协程切换
**原因**：避免频繁的内核态切换，提升并发性能
**影响**：需要手动管理协程栈和上下文

### 4. Future/Promise模式
**决策**：实现Promise模式支持async/await
**原因**：提供直观的异步编程接口
**验证**：await语法正确转换为Future等待

---

## 检查清单

- [x] 流程图与GMP模型设计一致
- [x] 所有已设计的接口和方法都被正确调用
- [x] 技术细节准确反映多核并发实现
- [x] 文档结构清晰易读，便于开发实现

---

## 下一步

**测试验证阶段**：
1. 单核模式验证基本功能
2. 多核模式验证并发性能
3. 负载均衡测试工作窃取效果
4. 压力测试极限并发能力

**优化阶段**：
1. NUMA感知调度优化
2. 协程池复用减少创建开销
3. 内存管理优化减少GC压力
4. 监控指标完善便于调优
