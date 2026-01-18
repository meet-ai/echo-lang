/**
 * @file executor.h
 * @brief Executor 聚合根定义
 *
 * Executor 是异步运行时的执行器，负责驱动任务执行循环。
 */

#ifndef EXECUTOR_H
#define EXECUTOR_H

#include "../../../include/echo/runtime.h"
#include "../../../include/echo/task.h"

// 前向声明
struct Scheduler;
struct Reactor;
struct TaskQueue;

// ============================================================================
// Executor 聚合根定义
// ============================================================================

/**
 * @brief Executor 聚合根结构体
 */
typedef struct Executor {
    uint64_t id;                  // 执行器ID
    struct Scheduler* scheduler;   // 简单调度器
    struct Reactor* reactor;       // 反应器
    struct TaskQueue* ready_queue; // 就绪队列
    bool running;                  // 运行状态
} executor_t;

// ============================================================================
// Executor 聚合根行为接口
// ============================================================================

/**
 * @brief 创建执行器
 * @param scheduler 调度器
 * @param reactor 反应器
 * @return 新创建的执行器
 */
executor_t* executor_create(struct Scheduler* scheduler, struct Reactor* reactor);

/**
 * @brief 销毁执行器
 * @param executor 要销毁的执行器
 */
void executor_destroy(executor_t* executor);

/**
 * @brief 运行执行循环
 * @param executor 执行器
 */
void executor_run_loop(executor_t* executor);

/**
 * @brief 生成新任务
 * @param executor 执行器
 * @param task 要生成的任务
 */
void executor_spawn_task(executor_t* executor, task_t* task);

/**
 * @brief 处理Future
 * @param executor 执行器
 * @param future 要处理的Future
 */
void executor_handle_future(executor_t* executor, struct Future* future);

/**
 * @brief 停止执行器
 * @param executor 执行器
 */
void executor_shutdown(executor_t* executor);

/**
 * @brief 检查执行器是否正在运行
 * @param executor 执行器
 * @return true如果正在运行
 */
bool executor_is_running(const executor_t* executor);

#endif // EXECUTOR_H

