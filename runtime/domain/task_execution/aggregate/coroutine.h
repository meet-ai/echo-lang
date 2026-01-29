/**
 * @file coroutine.h
 * @brief Coroutine 内部实体定义（TaskExecution 聚合内部实体）
 *
 * 注意：Coroutine 是 TaskExecution 聚合的内部实体，不对外暴露。
 * 外部代码应该通过 Task 聚合根访问 Coroutine，而不是直接访问。
 *
 * 使用场景：
 * - Task 聚合根内部使用（task.c）
 * - 调度器通过 Task 聚合根访问（不直接访问）
 *
 * 示例：
 * ```c
 * // ✅ 正确：通过Task聚合根访问
 * const Coroutine* coro = task_get_coroutine(task);
 *
 * // ❌ 错误：直接访问Coroutine
 * Coroutine* coro = task->coroutine; // 不应该这样做
 * ```
 */

#ifndef TASK_EXECUTION_AGGREGATE_COROUTINE_H
#define TASK_EXECUTION_AGGREGATE_COROUTINE_H

// 注意：此文件是TaskExecution聚合的内部实体定义，不对外暴露
// 外部代码应该通过Task聚合根访问Coroutine，而不是直接include此文件

#include <stdint.h>
#include <stdbool.h>
#include <time.h>
#include <pthread.h>
#include "../../coroutine/context.h"  // 上下文切换相关

// 前向声明
struct Processor;
struct Task;

// 协程状态枚举
typedef enum {
    COROUTINE_NEW,         // 新建
    COROUTINE_READY,       // 准备执行
    COROUTINE_RUNNING,     // 正在执行
    COROUTINE_SUSPENDED,   // 已挂起
    COROUTINE_COMPLETED,   // 已完成
    COROUTINE_FAILED,      // 执行失败
    COROUTINE_CANCELLED    // 已取消
} CoroutineState;

// 协程优先级
typedef enum {
    COROUTINE_PRIORITY_LOW,
    COROUTINE_PRIORITY_NORMAL,
    COROUTINE_PRIORITY_HIGH,
    COROUTINE_PRIORITY_CRITICAL
} CoroutinePriority;

// 协程执行统计
typedef struct {
    uint64_t total_execution_time;    // 总执行时间（纳秒）
    uint64_t total_suspend_time;      // 总挂起时间（纳秒）
    uint32_t yield_count;             // yield次数
    uint32_t resume_count;            // resume次数
    size_t peak_stack_usage;          // 峰值栈使用量
    uint64_t context_switch_count;    // 上下文切换次数
} CoroutineStats;

// 协程结构体
typedef struct Coroutine {
    uint64_t id;                    // 协程唯一ID
    char name[256];                 // 协程名称
    char description[512];          // 协程描述
    CoroutineState state;           // 协程状态
    CoroutinePriority priority;     // 协程优先级

    // 上下文切换相关
    context_t* context;             // 协程上下文
    char* stack;                    // 栈内存
    size_t stack_size;              // 栈大小
    size_t stack_used;              // 当前栈使用量

    // 执行相关
    void (*entry_point)(void*);     // 协程入口函数
    void* arg;                      // 入口函数参数
    void* result_data;              // 执行结果数据
    size_t result_size;             // 结果数据大小
    int exit_code;                  // 退出代码
    char error_message[1024];       // 错误信息

    // 时间戳
    time_t created_at;              // 创建时间
    time_t started_at;              // 开始时间
    time_t completed_at;            // 完成时间
    time_t last_yield_at;           // 最后yield时间
    time_t last_resume_at;          // 最后resume时间

    // 调度相关
    struct Processor* bound_processor;  // 绑定到的处理器
    struct Task* task;               // 关联的任务
    struct Coroutine* next;          // 用于队列的下一个协程
    uint64_t schedule_count;         // 被调度次数

    // 统计信息
    CoroutineStats stats;

    // 同步机制
    pthread_mutex_t mutex;           // 协程状态保护
    pthread_cond_t condition;        // 条件变量（用于等待）

    // 回调函数
    void (*on_complete)(struct Coroutine*);  // 完成回调
    void (*on_error)(struct Coroutine*, const char*);  // 错误回调
    void* user_data;                 // 用户数据
} Coroutine;

// 协程管理函数声明
Coroutine* coroutine_create(const char* name, void (*entry_point)(void*), void* arg, size_t stack_size);
void coroutine_destroy(Coroutine* coroutine);

// 协程生命周期管理
bool coroutine_start(Coroutine* coroutine);
void coroutine_resume(Coroutine* coroutine);
void coroutine_suspend(Coroutine* coroutine);
bool coroutine_join(Coroutine* coroutine, uint32_t timeout_ms);
bool coroutine_cancel(Coroutine* coroutine, const char* reason);

// 协程状态查询
CoroutineState coroutine_get_state(const Coroutine* coroutine);
bool coroutine_is_running(const Coroutine* coroutine);
bool coroutine_is_complete(Coroutine* coroutine);
bool coroutine_is_cancelled(const Coroutine* coroutine);

// 协程属性管理
bool coroutine_set_name(Coroutine* coroutine, const char* name);
bool coroutine_set_description(Coroutine* coroutine, const char* description);
bool coroutine_set_priority(Coroutine* coroutine, CoroutinePriority priority);
bool coroutine_set_callbacks(Coroutine* coroutine,
                           void (*on_complete)(Coroutine*),
                           void (*on_error)(Coroutine*, const char*));

// 协程结果管理
bool coroutine_set_result(Coroutine* coroutine, const void* data, size_t size);
bool coroutine_get_result(Coroutine* coroutine, void** data, size_t* size);
bool coroutine_set_error(Coroutine* coroutine, const char* error_message);
const char* coroutine_get_error(const Coroutine* coroutine);

// 协程统计和监控
bool coroutine_get_stats(const Coroutine* coroutine, CoroutineStats* stats);
uint64_t coroutine_get_execution_time_us(const Coroutine* coroutine);
uint64_t coroutine_get_total_time_us(const Coroutine* coroutine);
size_t coroutine_get_stack_usage(const Coroutine* coroutine);
uint32_t coroutine_get_yield_count(const Coroutine* coroutine);

// 协程ID生成
uint64_t coroutine_generate_id(void);

// 协程yield函数（全局函数）
bool coroutine_yield(void);

// 协程包装函数（处理协程完成和清理）
void coroutine_wrapper(Coroutine* coroutine);

#endif // TASK_EXECUTION_AGGREGATE_COROUTINE_H
