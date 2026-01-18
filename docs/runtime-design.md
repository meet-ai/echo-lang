# Echo并发运行时设计

## 🎯 目标

实现真正的async/await并发执行，支持goroutine风格的并发编程。

## 🏗️ 架构设计

### 1. 并发模型
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   应用程序      │    │   并发运行时    │    │   系统线程池    │
│                 │    │                 │    │                 │
│ - async函数     │───▶│ - 协程调度器    │───▶│ - OS线程        │
│ - await表达式   │    │ - Future管理    │    │ - 异步I/O      │
│ - spawn原语     │    │ - 通道通信      │    │ - 定时器        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### 2. 核心组件

#### 2.1 协程调度器 (Coroutine Scheduler)
```go
// 协程状态
type CoroutineState int
const (
    Ready CoroutineState = iota
    Running
    Suspended
    Completed
    Failed
)

// 协程控制块
type Coroutine struct {
    ID       uint64
    State    CoroutineState
    Stack    []byte          // 协程栈
    PC       uintptr         // 程序计数器
    Context  *ExecutionContext
    Future   *Future         // 返回值
}

// 调度器
type Scheduler struct {
    readyQueue   chan *Coroutine
    suspendedMap map[uint64]*Coroutine
    workers      []*Worker
}
```

#### 2.2 Future/Promise系统
```go
// Future状态
type FutureState int
const (
    Pending FutureState = iota
    Resolved
    Rejected
)

// Future实现
type Future struct {
    ID       uint64
    State    FutureState
    Value    interface{}
    Error    error
    Callbacks []FutureCallback
    Mutex    sync.Mutex
}

// Promise实现
type Promise struct {
    Future *Future
}

// Future回调
type FutureCallback func(value interface{}, err error)
```

#### 2.3 通道系统 (CSP风格)
```go
// 通道接口
type Channel interface {
    Send(value interface{}) error
    Receive() (interface{}, error)
    Close() error
    IsClosed() bool
}

// 有缓冲通道
type BufferedChannel struct {
    buffer []interface{}
    capacity int
    senders  []*Coroutine  // 等待发送的协程
    receivers []*Coroutine // 等待接收的协程
    closed   bool
    mutex    sync.Mutex
}
```

## 🚀 实现路线图

### 阶段1: 基础并发原语 (2-3周)

#### 1.1 协程创建与切换
- [ ] 实现协程栈管理
- [ ] 实现协程上下文切换
- [ ] 实现协程生命周期管理

#### 1.2 spawn原语实现
- [ ] 修改代码生成器支持spawn
- [ ] 实现协程创建函数
- [ ] 实现协程启动机制

#### 1.3 简单调度器
- [ ] 实现轮询调度算法
- [ ] 实现协程就绪队列
- [ ] 实现基本的上下文切换

### 阶段2: Future与异步编程 (2-3周)

#### 2.1 Future类型实现
- [ ] 定义Future AST节点
- [ ] 实现Future代码生成
- [ ] 实现Future运行时库

#### 2.2 async函数编译
- [ ] 修改async函数代码生成
- [ ] 实现async函数到协程的转换
- [ ] 实现async函数返回值处理

#### 2.3 await表达式实现
- [ ] 修改await表达式代码生成
- [ ] 实现协程挂起/恢复机制
- [ ] 实现Future完成等待

### 阶段3: 通道通信 (2-3周)

#### 3.1 通道类型实现
- [ ] 定义Channel AST节点
- [ ] 实现通道代码生成
- [ ] 实现通道运行时库

#### 3.2 发送/接收操作
- [ ] 实现channel send操作
- [ ] 实现channel receive操作
- [ ] 实现阻塞/非阻塞语义

#### 3.3 通道缓冲
- [ ] 实现有缓冲通道
- [ ] 实现无缓冲通道
- [ ] 实现通道关闭机制

### 阶段4: 高级并发特性 (2-3周)

#### 4.1 select语句实现
- [ ] 扩展select语法支持
- [ ] 实现select多路复用
- [ ] 实现default分支

#### 4.2 错误处理
- [ ] 实现协程panic处理
- [ ] 实现协程错误传播
- [ ] 实现协程超时机制

#### 4.3 性能优化
- [ ] 实现工作窃取调度
- [ ] 实现协程池复用
- [ ] 实现NUMA感知调度

## 🔧 技术实现细节

### 1. 协程实现策略

#### 方案A: 栈拷贝 (推荐)
```go
// 使用汇编实现栈切换
func switchCoroutine(from, to *Coroutine) {
    // 保存当前协程状态
    saveCoroutineState(from)

    // 恢复目标协程状态
    restoreCoroutineState(to)
}
```

#### 方案B: 分段栈
```go
// 动态栈增长
func growStack(coroutine *Coroutine, neededSize int) {
    newStack := make([]byte, len(coroutine.Stack)*2)
    copy(newStack, coroutine.Stack)
    coroutine.Stack = newStack
}
```

### 2. 内存管理

#### 协程栈管理
```go
const (
    InitialStackSize = 4096  // 4KB初始栈
    MaxStackSize     = 1024 * 1024  // 1MB最大栈
)

// 协程栈分配
func allocateStack(size int) []byte {
    return make([]byte, size)
}
```

#### 垃圾回收
```go
// 协程退出时清理资源
func cleanupCoroutine(coroutine *Coroutine) {
    // 清理协程栈
    coroutine.Stack = nil

    // 清理协程上下文
    coroutine.Context = nil

    // 通知等待的协程
    notifyWaiters(coroutine)
}
```

### 3. 系统集成

#### 与LLVM IR集成
```go
// 生成协程创建代码
func generateSpawnCall(irManager IRModuleManager, spawnExpr *SpawnExpr) {
    // 创建新协程
    coroutinePtr := irManager.CreateCall(createCoroutineFunc, ...)

    // 设置协程入口函数
    irManager.CreateCall(setCoroutineEntryFunc, coroutinePtr, spawnExpr.Function)

    // 启动协程
    irManager.CreateCall(startCoroutineFunc, coroutinePtr)
}
```

#### 外部函数声明
```go
// 运行时库函数声明
extern void* create_coroutine(void* entry);
extern void start_coroutine(void* coroutine);
extern void yield_coroutine();
extern void* await_future(void* future);
```

## 📊 性能目标

### 并发性能指标
- **协程创建时间**: < 10μs
- **协程切换时间**: < 1μs
- **内存开销**: < 4KB/协程
- **最大并发数**: > 10万个协程

### 通道性能指标
- **发送/接收延迟**: < 100ns
- **缓冲区开销**: < 64字节/通道
- **吞吐量**: > 1M ops/sec

### Future性能指标
- **Future创建时间**: < 5μs
- **回调注册延迟**: < 50ns
- **内存开销**: < 128字节/Future

## 🧪 测试策略

### 单元测试
- [ ] 协程生命周期测试
- [ ] 调度器算法测试
- [ ] Future状态转换测试
- [ ] 通道通信测试

### 集成测试
- [ ] async/await完整流程测试
- [ ] spawn并发执行测试
- [ ] 通道通信集成测试
- [ ] 错误处理测试

### 性能测试
- [ ] 协程创建性能测试
- [ ] 并发执行性能测试
- [ ] 内存使用测试
- [ ] 通道吞吐量测试

### 压力测试
- [ ] 高并发场景测试
- [ ] 长时间运行稳定性测试
- [ ] 资源泄漏测试
- [ ] 异常恢复测试

## 📚 参考实现

### 1. Go语言协程 (goroutine)
- M:N调度模型
- 栈拷贝协程实现
- 通道通信原语

### 2. Lua协程
- 栈式协程
- yield/resume语义
- 轻量级实现

### 3. async/await运行时
- Promise/Future模式
- 事件循环
- 异步I/O集成

## 🎯 验收标准

### 功能验收
- [ ] async函数可以并发执行
- [ ] await可以正确等待异步操作
- [ ] spawn可以启动新协程
- [ ] 通道通信正常工作
- [ ] select语句支持多路复用

### 性能验收
- [ ] 协程创建时间 < 10μs
- [ ] 协程切换时间 < 1μs
- [ ] 支持10万个并发协程
- [ ] 通道吞吐量 > 1M ops/sec

### 稳定性验收
- [ ] 无死锁问题
- [ ] 无内存泄漏
- [ ] 错误处理完善
- [ ] 异常恢复正常
