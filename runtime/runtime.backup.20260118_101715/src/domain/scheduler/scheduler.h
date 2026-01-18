/**
 * @file scheduler.h
 * @brief GMP Scheduler 聚合根定义
 *
 * GMP Scheduler 是整个并发系统的核心，协调Goroutine、Machine和Processor。
 */

#ifndef SCHEDULER_H
#define SCHEDULER_H

#include "../../../include/echo/runtime.h"
#include "../../../include/echo/task.h"
#include "../coroutine/context.h"  // 上下文切换
#include <pthread.h>

// ============================================================================
// 全局ID分配器（原子计数器）
// ============================================================================

/**
 * @brief 全局调度器ID分配器
 */
static uint64_t g_scheduler_id_counter = 0;
static pthread_mutex_t g_scheduler_id_mutex = PTHREAD_MUTEX_INITIALIZER;

/**
 * @brief 原子分配调度器ID
 * @return 新的唯一调度器ID
 */
static uint64_t scheduler_allocate_id(void) {
    uint64_t id;
    pthread_mutex_lock(&g_scheduler_id_mutex);
    g_scheduler_id_counter++;
    id = g_scheduler_id_counter;
    pthread_mutex_unlock(&g_scheduler_id_mutex);
    return id;
}

// 前向声明
struct Processor;
struct Machine;

// ============================================================================
// GMP Scheduler 聚合根定义
// ============================================================================

/**
 * @brief GMP Scheduler 聚合根结构体
 * 协调整个GMP并发模型
 */
typedef struct Scheduler {
    uint64_t id;                    // 调度器ID

    // GMP组件
    struct Processor** processors;  // 处理器数组 (P)
    struct Machine** machines;      // 机器数组 (M)
    uint32_t num_processors;        // 处理器数量
    uint32_t num_machines;          // 机器数量

    // GMP队列管理
    task_t** ready_queue;           // 就绪队列（全局）
    size_t ready_queue_size;        // 就绪队列大小
    size_t ready_queue_capacity;    // 就绪队列容量

    task_t** blocked_queue;         // 阻塞队列（等待I/O等）
    size_t blocked_queue_size;      // 阻塞队列大小
    size_t blocked_queue_capacity;  // 阻塞队列容量

    // 状态管理
    bool is_running;                // 是否正在运行

    // 统计信息
    uint64_t tasks_scheduled;       // 已调度的任务总数
    uint64_t tasks_completed;       // 已完成的任务总数

    // 同步机制
    pthread_mutex_t global_mutex;   // 全局锁
    pthread_cond_t global_cond;     // 全局条件变量

    // 上下文切换支持
    context_t* scheduler_context;   // 调度器主上下文
} scheduler_t;

// ============================================================================
// 上下文切换支持函数
// ============================================================================

/**
 * @brief 获取调度器上下文
 */
context_t* scheduler_get_context(scheduler_t* scheduler);

/**
 * @brief 设置调度器上下文
 */
void scheduler_set_context(scheduler_t* scheduler, context_t* context);

/**
 * @brief 调度器主循环
 * 选择下一个要执行的任务并进行上下文切换
 */
void scheduler_main_loop(scheduler_t* scheduler);

/**
 * @brief 选择下一个要执行的任务
 */
task_t* scheduler_select_next_task(scheduler_t* scheduler);

/**
 * @brief 将任务加入就绪队列
 */
bool scheduler_enqueue_ready(scheduler_t* scheduler, task_t* task);

/**
 * @brief 从就绪队列取出任务
 */
task_t* scheduler_dequeue_ready(scheduler_t* scheduler);

/**
 * @brief 将任务加入阻塞队列
 */
bool scheduler_enqueue_blocked(scheduler_t* scheduler, task_t* task);

/**
 * @brief 将任务从阻塞队列移动到就绪队列
 */
bool scheduler_unblock_task(scheduler_t* scheduler, task_t* task);

// ============================================================================
// GMP Scheduler 聚合根行为接口
// ============================================================================

/**
 * @brief 创建GMP调度器
 * @param num_processors 处理器数量
 * @param num_machines 机器数量
 * @return 新创建的调度器
 */
scheduler_t* scheduler_create(uint32_t num_processors, uint32_t num_machines);

/**
 * @brief 销毁GMP调度器
 * @param scheduler 要销毁的调度器
 */
void scheduler_destroy(scheduler_t* scheduler);

/**
 * @brief 启动调度器
 * @param scheduler 调度器
 * @return 成功返回0，失败返回-1
 */
int scheduler_start(scheduler_t* scheduler);

/**
 * @brief 停止调度器
 * @param scheduler 调度器
 */
void scheduler_stop(scheduler_t* scheduler);

/**
 * @brief 调度任务
 * @param scheduler 调度器
 * @param task 要调度的任务
 * @return 成功返回0，失败返回-1
 */
int scheduler_schedule_task(scheduler_t* scheduler, task_t* task);

/**
 * @brief 获取就绪任务
 * @param scheduler 调度器
 * @param processor 请求任务的处理器
 * @return 任务指针，NULL表示无可用任务
 */
task_t* scheduler_get_ready_task(scheduler_t* scheduler, struct Processor* processor);

/**
 * @brief 任务完成通知
 * @param scheduler 调度器
 * @param task 已完成的任务
 */
void scheduler_task_completed(scheduler_t* scheduler, task_t* task);

/**
 * @brief 执行负载均衡
 * @param scheduler 调度器
 */
void scheduler_balance_load(scheduler_t* scheduler);

/**
 * @brief 获取调度器统计信息
 * @param scheduler 调度器
 * @param scheduled 已调度的任务数
 * @param completed 已完成的任务数
 */
void scheduler_get_stats(const scheduler_t* scheduler,
                        uint64_t* scheduled,
                        uint64_t* completed);

#endif // SCHEDULER_H
