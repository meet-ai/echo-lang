#define _XOPEN_SOURCE 600
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <unistd.h>
#include <pthread.h>
#include <sched.h>
#include <setjmp.h>
#include <string.h>

// GMP模型常量
#define MAX_PROCESSORS 8
#define LOCAL_QUEUE_SIZE 256
#define GLOBAL_QUEUE_SIZE 1024

// 协程状态
typedef enum {
    COROUTINE_READY,
    COROUTINE_RUNNING,
    COROUTINE_SUSPENDED,
    COROUTINE_COMPLETED,
    COROUTINE_FAILED
} CoroutineState;

// Future状态
typedef enum {
    FUTURE_PENDING,
    FUTURE_RESOLVED,
    FUTURE_REJECTED
} FutureState;

// 上下文结构体 - 使用setjmp/longjmp实现
typedef struct Context {
    jmp_buf env;        // setjmp/longjmp环境
    int is_initial;     // 是否是初始上下文
} Context;

// Future结构体
typedef struct Future {
    uint64_t id;
    FutureState state;
    void* value;
    void* error;
    struct Coroutine* waiting_coroutine;  // 等待这个Future的协程
} Future;

// 协程结构体 (G - Goroutine)
typedef struct Coroutine {
    uint64_t id;
    CoroutineState state;
    Context context;           // 汇编实现的上下文
    char* stack;
    size_t stack_size;
    void (*entry_point)(void*);  // 协程入口函数
    void* arg;                   // 入口函数参数
    struct Processor* bound_processor;  // 绑定到的处理器
    struct Coroutine* next;      // 用于队列的下一个协程
    int is_main;                 // 是否是主协程
} Coroutine;

// 处理器结构体 (P - Processor)
typedef struct Processor {
    uint32_t id;                              // 处理器ID
    Coroutine* local_queue[LOCAL_QUEUE_SIZE]; // 本地协程队列
    uint32_t local_queue_head;                 // 队列头
    uint32_t local_queue_tail;                 // 队列尾
    uint32_t local_queue_size;                 // 队列大小
    pthread_mutex_t lock;                      // 处理器锁
    struct Machine* bound_machine;             // 绑定的机器
} Processor;

// 机器结构体 (M - Machine/OS Thread)
typedef struct Machine {
    uint32_t id;                    // 机器ID
    pthread_t thread;              // OS线程
    Processor* bound_processor;    // 绑定的处理器
    Context context;               // 机器上下文
    int is_running;                // 是否正在运行
} Machine;

// 全局调度器 (GMP模型)
typedef struct GlobalScheduler {
    // 全局协程队列
    Coroutine* global_queue[GLOBAL_QUEUE_SIZE];
    uint32_t global_queue_head;
    uint32_t global_queue_tail;
    uint32_t global_queue_size;

    // 处理器数组
    Processor processors[MAX_PROCESSORS];
    uint32_t num_processors;

    // 机器数组
    Machine machines[MAX_PROCESSORS];
    uint32_t num_machines;

    // 同步原语
    pthread_mutex_t global_lock;    // 全局锁
    pthread_cond_t work_available;  // 工作可用条件变量

    // ID生成器
    uint64_t next_coroutine_id;
    uint64_t next_future_id;

    int is_running;                 // 调度器是否正在运行
    Context main_context;           // 主线程上下文
} GlobalScheduler;

// 全局调度器实例
static GlobalScheduler global_scheduler;
static __thread Coroutine* current_coroutine = NULL;

// 函数声明
static uint64_t generate_coroutine_id(void);
static uint64_t generate_future_id(void);
static void coroutine_wrapper(Coroutine* co);
static void processor_schedule(Processor* p);
static void global_scheduler_push(Coroutine* co);
static Coroutine* global_scheduler_pop(void);
static void processor_push_local(Processor* p, Coroutine* co);
static Coroutine* processor_pop_local(Processor* p);
static void processor_init(Processor* p, uint32_t id);
static void machine_init(Machine* m, uint32_t id);
static void* machine_thread_entry(void* arg);
static void context_switch(Context* from, Context* to);
static void context_set_stack_and_entry(Context* ctx, char* stack, size_t stack_size, void (*entry)(Coroutine*), Coroutine* arg);

// 上下文切换函数 - 使用内联汇编
// 上下文切换函数 - 使用setjmp/longjmp
static void context_switch(Context* from, Context* to) {
    if (setjmp(from->env) == 0) {
        // 保存当前上下文成功，现在跳转到新上下文
        longjmp(to->env, 1);
    }
    // 从这里返回时，说明从新上下文切换回来了
}

// 设置上下文的栈和入口点 (setjmp版本不需要手动设置)
static void context_set_stack_and_entry(Context* ctx, char* stack, size_t stack_size, void (*entry)(Coroutine*), Coroutine* arg) {
    // 对于setjmp/longjmp，上下文会在第一次调度时自动设置
    ctx->is_initial = 1;
    memset(&ctx->env, 0, sizeof(jmp_buf));
}

// ID生成器
static uint64_t generate_coroutine_id(void) {
    static uint64_t id = 0;
    return __atomic_fetch_add(&id, 1, __ATOMIC_SEQ_CST);
}

static uint64_t generate_future_id(void) {
    static uint64_t id = 0;
    return __atomic_fetch_add(&id, 1, __ATOMIC_SEQ_CST);
}

// 全局调度器初始化
void global_scheduler_init(void) {
    printf("DEBUG: Initializing global scheduler with assembly-based context switching\n");

    memset(&global_scheduler, 0, sizeof(GlobalScheduler));

    // 获取CPU核心数 (macOS兼容性处理)
    int num_cores = 4; // 简化：假设4个核心，在生产环境中应该检测实际核心数

    global_scheduler.num_processors = num_cores;
    global_scheduler.num_machines = num_cores;

    printf("DEBUG: Detected %d CPU cores, creating %d processors and machines\n", num_cores, num_cores);

    // 初始化同步原语
    pthread_mutex_init(&global_scheduler.global_lock, NULL);
    pthread_cond_init(&global_scheduler.work_available, NULL);

    // 初始化处理器
    for (uint32_t i = 0; i < global_scheduler.num_processors; i++) {
        processor_init(&global_scheduler.processors[i], i);
    }

    // 初始化机器
    for (uint32_t i = 0; i < global_scheduler.num_machines; i++) {
        machine_init(&global_scheduler.machines[i], i);
        // 绑定机器到处理器
        global_scheduler.machines[i].bound_processor = &global_scheduler.processors[i];
        global_scheduler.processors[i].bound_machine = &global_scheduler.machines[i];
    }

    global_scheduler.is_running = 0;
    printf("DEBUG: Global scheduler initialized\n");
}

// 处理器初始化
static void processor_init(Processor* p, uint32_t id) {
    p->id = id;
    p->local_queue_head = 0;
    p->local_queue_tail = 0;
    p->local_queue_size = 0;
    pthread_mutex_init(&p->lock, NULL);
    printf("DEBUG: Processor %u initialized\n", id);
}

// 机器初始化
static void machine_init(Machine* m, uint32_t id) {
    m->id = id;
    m->is_running = 0;
    printf("DEBUG: Machine %u initialized\n", id);
}

// 协程创建
Coroutine* coroutine_create(void (*entry_func)(void*), void* arg, size_t stack_size) {
    Coroutine* co = (Coroutine*)malloc(sizeof(Coroutine));
    if (!co) {
        perror("Failed to allocate Coroutine");
        return NULL;
    }

    co->id = generate_coroutine_id();
    co->state = COROUTINE_READY;
    co->entry_point = entry_func;
    co->arg = arg;
    co->stack = NULL;
    co->stack_size = 0;
    co->next = NULL;
    co->bound_processor = NULL;
    co->is_main = 0;

    if (stack_size > 0) {
        co->stack = malloc(stack_size);
        if (!co->stack) {
            perror("Failed to allocate stack for coroutine");
            free(co);
            return NULL;
        }
        co->stack_size = stack_size;

        // 设置上下文
        context_set_stack_and_entry(&co->context, co->stack, stack_size, coroutine_wrapper, co);
    }

    printf("DEBUG: Created coroutine %llu with entry_func=%p, arg=%p, stack_size=%zu\n", co->id, entry_func, arg, stack_size);
    return co;
}

// 协程包装器
static void coroutine_wrapper(Coroutine* co) {
    printf("DEBUG: Coroutine %llu wrapper entered\n", co->id);

    if (co->entry_point) {
        co->entry_point(co->arg);
    }

    printf("DEBUG: Coroutine %llu entry_point returned\n", co->id);
    co->state = COROUTINE_COMPLETED;

    // 如果协程完成，切换到调度器
    if (co->bound_processor) {
        processor_schedule(co->bound_processor);
    }
}

// 处理器调度
static void processor_schedule(Processor* p) {
    if (!p) return;

    Coroutine* prev_co = current_coroutine;
    current_coroutine = NULL; // 清空当前协程，等待调度器选择下一个

    // 如果前一个协程是挂起状态，重新加入队列
    if (prev_co && prev_co->state == COROUTINE_SUSPENDED) {
        printf("DEBUG: Coroutine %llu suspended, pushing back to local queue\n", prev_co->id);
        processor_push_local(p, prev_co);
    }

    // 尝试获取下一个可执行协程
    Coroutine* next_co = processor_pop_local(p);
    if (!next_co) {
        // 尝试从全局队列窃取
        next_co = global_scheduler_pop();
        if (!next_co) {
            // 没有可执行协程，切换回机器的上下文（等待新任务）
            printf("DEBUG: Processor %u has no work, yielding to machine context\n", p->id);
            if (prev_co) {
                context_switch(&prev_co->context, &p->bound_machine->context);
            } else {
                // 如果没有前一个协程，直接设置机器上下文
                // 这里简化处理，实际应该有更复杂的逻辑
                return;
            }
        }
    }

    if (next_co) {
        printf("DEBUG: Processor %u scheduling coroutine %llu (state: %d)\n", p->id, next_co->id, next_co->state);
        current_coroutine = next_co;
        next_co->state = COROUTINE_RUNNING;
        next_co->bound_processor = p;

        // 切换上下文
        if (prev_co) {
            printf("DEBUG: Switching context from %llu to %llu\n", prev_co ? prev_co->id : 0, next_co->id);
            context_switch(&prev_co->context, &next_co->context);
        } else {
            printf("DEBUG: First time activating coroutine %llu\n", next_co->id);
            // 设置rip到协程入口点，然后切换
            context_switch(&p->bound_machine->context, &next_co->context);
        }
    }
}

// 队列操作
static void processor_push_local(Processor* p, Coroutine* co) {
    pthread_mutex_lock(&p->lock);
    if (p->local_queue_size < LOCAL_QUEUE_SIZE) {
        p->local_queue[p->local_queue_tail] = co;
        p->local_queue_tail = (p->local_queue_tail + 1) % LOCAL_QUEUE_SIZE;
        p->local_queue_size++;
        printf("DEBUG: Pushed coroutine %llu to processor %u local queue (size: %u)\n", co->id, p->id, p->local_queue_size);
    }
    pthread_mutex_unlock(&p->lock);
}

static Coroutine* processor_pop_local(Processor* p) {
    pthread_mutex_lock(&p->lock);
    Coroutine* co = NULL;
    if (p->local_queue_size > 0) {
        co = p->local_queue[p->local_queue_head];
        p->local_queue_head = (p->local_queue_head + 1) % LOCAL_QUEUE_SIZE;
        p->local_queue_size--;
        printf("DEBUG: Popped coroutine %llu from processor %u local queue (size: %u)\n", co->id, p->id, p->local_queue_size);
    }
    pthread_mutex_unlock(&p->lock);
    return co;
}

static void global_scheduler_push(Coroutine* co) {
    pthread_mutex_lock(&global_scheduler.global_lock);
    if (global_scheduler.global_queue_size < GLOBAL_QUEUE_SIZE) {
        global_scheduler.global_queue[global_scheduler.global_queue_tail] = co;
        global_scheduler.global_queue_tail = (global_scheduler.global_queue_tail + 1) % GLOBAL_QUEUE_SIZE;
        global_scheduler.global_queue_size++;
        printf("DEBUG: Pushed coroutine %llu to global queue (size: %u)\n", co->id, global_scheduler.global_queue_size);
        pthread_cond_signal(&global_scheduler.work_available);
    }
    pthread_mutex_unlock(&global_scheduler.global_lock);
}

static Coroutine* global_scheduler_pop(void) {
    pthread_mutex_lock(&global_scheduler.global_lock);
    Coroutine* co = NULL;
    if (global_scheduler.global_queue_size > 0) {
        co = global_scheduler.global_queue[global_scheduler.global_queue_head];
        global_scheduler.global_queue_head = (global_scheduler.global_queue_head + 1) % GLOBAL_QUEUE_SIZE;
        global_scheduler.global_queue_size--;
        printf("DEBUG: Popped coroutine %llu from global queue (size: %u)\n", co->id, global_scheduler.global_queue_size);
    }
    pthread_mutex_unlock(&global_scheduler.global_lock);
    return co;
}

// Future相关函数
void* future_new(void) {
    Future* future = (Future*)malloc(sizeof(Future));
    if (!future) {
        perror("Failed to allocate Future");
        return NULL;
    }
    future->id = generate_future_id();
    future->state = FUTURE_PENDING;
    future->value = NULL;
    future->error = NULL;
    future->waiting_coroutine = NULL;
    printf("DEBUG: Created future %llu\n", future->id);
    return future;
}

void future_resolve(void* future_ptr, void* value) {
    Future* future = (Future*)future_ptr;
    if (!future) return;

    future->state = FUTURE_RESOLVED;
    future->value = value;

    printf("DEBUG: Resolved future %llu with value %p\n", future->id, value);

    // 唤醒等待的协程
    if (future->waiting_coroutine) {
        Coroutine* waiting_co = future->waiting_coroutine;
        future->waiting_coroutine = NULL;
        waiting_co->state = COROUTINE_READY;

        // 将协程重新加入调度队列
        if (waiting_co->bound_processor) {
            processor_push_local(waiting_co->bound_processor, waiting_co);
        } else {
            global_scheduler_push(waiting_co);
        }

        printf("DEBUG: Woke up waiting coroutine %llu for future %llu\n", waiting_co->id, future->id);
    }
}

void future_reject(void* future_ptr, void* error) {
    Future* future = (Future*)future_ptr;
    if (!future) return;

    future->state = FUTURE_REJECTED;
    future->error = error;

    printf("DEBUG: Rejected future %llu with error %p\n", future->id, error);

    // 唤醒等待的协程
    if (future->waiting_coroutine) {
        Coroutine* waiting_co = future->waiting_coroutine;
        future->waiting_coroutine = NULL;
        waiting_co->state = COROUTINE_READY;

        if (waiting_co->bound_processor) {
            processor_push_local(waiting_co->bound_processor, waiting_co);
        } else {
            global_scheduler_push(waiting_co);
        }

        printf("DEBUG: Woke up waiting coroutine %llu for rejected future %llu\n", waiting_co->id, future->id);
    }
}

// 协程让出
void scheduler_yield(void) {
    if (current_coroutine && current_coroutine->bound_processor) {
        printf("DEBUG: Coroutine %llu yielding\n", current_coroutine->id);
        processor_schedule(current_coroutine->bound_processor);
    }
}

// 运行调度器
void run_scheduler(void) {
    printf("DEBUG: Starting scheduler with assembly-based context switching\n");

    if (global_scheduler.is_running) {
        printf("DEBUG: Scheduler already running\n");
        return;
    }

    global_scheduler.is_running = 1;

    // 保存主线程上下文
    // 注意：这里简化处理，实际应该在第一次调用时保存

    // 启动机器线程
    for (uint32_t i = 0; i < global_scheduler.num_machines; i++) {
        Machine* m = &global_scheduler.machines[i];
        m->is_running = 1;

        if (pthread_create(&m->thread, NULL, machine_thread_entry, m) != 0) {
            perror("Failed to create machine thread");
            continue;
        }
        printf("DEBUG: Started machine thread %u\n", m->id);
    }

    // 等待所有机器线程完成
    for (uint32_t i = 0; i < global_scheduler.num_machines; i++) {
        Machine* m = &global_scheduler.machines[i];
        if (m->is_running) {
            pthread_join(m->thread, NULL);
            printf("DEBUG: Machine thread %u joined\n", m->id);
        }
    }

    global_scheduler.is_running = 0;
    printf("DEBUG: Scheduler stopped\n");
}

// 机器线程入口
static void* machine_thread_entry(void* arg) {
    Machine* m = (Machine*)arg;
    Processor* p = m->bound_processor;

    printf("DEBUG: Machine thread %u started, bound to processor %u\n", m->id, p->id);

    while (global_scheduler.is_running) {
        // 尝试从本地队列获取工作
        Coroutine* co = processor_pop_local(p);
        if (!co) {
            // 尝试从全局队列获取工作
            co = global_scheduler_pop();
            if (!co) {
                // 没有工作，等待
                pthread_mutex_lock(&global_scheduler.global_lock);
                if (global_scheduler.global_queue_size == 0) {
                    printf("DEBUG: Machine %u waiting for work\n", m->id);
                    pthread_cond_wait(&global_scheduler.work_available, &global_scheduler.global_lock);
                }
                pthread_mutex_unlock(&global_scheduler.global_lock);
                continue;
            }
        }

        if (co) {
            printf("DEBUG: Machine %u executing coroutine %llu\n", m->id, co->id);
            current_coroutine = co;
            co->state = COROUTINE_RUNNING;
            co->bound_processor = p;

            // 切换到协程上下文
            context_switch(&m->context, &co->context);

            // 从协程返回后，继续循环
        }
    }

    printf("DEBUG: Machine thread %u exiting\n", m->id);
    return NULL;
}

// 协程生成
void* coroutine_spawn(void (*entry_func)(void*), int32_t arg_count, void* args, void* future_ptr) {
    printf("DEBUG: coroutine_spawn called with entry_func=%p, future_ptr=%p\n", entry_func, future_ptr);

    // 创建协程
    Coroutine* co = coroutine_create(entry_func, future_ptr, 64 * 1024); // 64KB栈
    if (!co) {
        printf("DEBUG: Failed to create coroutine\n");
        return NULL;
    }

    // 绑定到处理器（简化：绑定到第一个处理器）
    Processor* p = &global_scheduler.processors[0];
    co->bound_processor = p;

    // 推送到本地队列
    processor_push_local(p, co);

    // 唤醒等待的机器
    pthread_cond_signal(&global_scheduler.work_available);

    printf("DEBUG: Spawned coroutine %llu, returning future %p\n", co->id, future_ptr);
    return future_ptr;
}

// 协程等待
void* coroutine_await(void* future_ptr) {
    Future* future = (Future*)future_ptr;
    if (!future) {
        printf("ERROR: coroutine_await called with NULL future\n");
        return NULL;
    }

    printf("DEBUG: Awaiting future %llu, current state: %d\n", future->id, future->state);

    // 如果Future已经解决，直接返回结果
    if (future->state == FUTURE_RESOLVED) {
        printf("DEBUG: Future %llu already resolved, returning value %p\n", future->id, future->value);
        return future->value;
    }

    // 如果当前不在协程中（例如在主线程中调用await），则无法等待
    if (!current_coroutine) {
        printf("DEBUG: Await called outside coroutine, returning NULL (simplified)\n");
        return NULL;
    }

    // 设置当前协程为等待状态
    future->waiting_coroutine = current_coroutine;
    current_coroutine->state = COROUTINE_SUSPENDED;
    printf("DEBUG: Coroutine %llu awaiting future %llu, suspending\n", current_coroutine->id, future->id);

    // 让出控制权给调度器
    if (current_coroutine->bound_processor) {
        processor_schedule(current_coroutine->bound_processor);
    }

    // 协程被唤醒后，从这里继续执行
    printf("DEBUG: Coroutine %llu resumed from await, future %llu resolved with value %p\n", current_coroutine->id, future->id, future->value);
    return future->value;
}

// 运行时初始化
void echo_main(void) {
    printf("DEBUG: echo_main called\n");
    global_scheduler_init();
    run_scheduler();
    printf("DEBUG: echo_main completed\n");
}

// 打印函数
void print_int(int64_t value) {
    printf("%lld", value);
}

void print_string(char* value) {
    printf("%s", value);
}
