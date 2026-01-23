/**
 * @file executor.c
 * @brief Executor 聚合根实现
 *
 * Executor 负责驱动任务执行循环，实现异步运行时的核心循环。
 */

#include "executor.h"
#include "../../domain/scheduler/scheduler.h"
#include "../../domain/reactor/reactor.h"
#include "../../domain/reactor/timer_future.h"
#include "../../domain/coroutine/context.h"
#include "../../domain/coroutine/yield.h"
#include <stdlib.h>
#include <stdio.h>
#include <unistd.h>

// 前向声明（需要其他模块的接口）
struct Scheduler;
struct Reactor;
struct TaskQueue;
struct Future;

// 简单的任务队列实现（临时）
typedef struct TaskQueue {
    task_t** tasks;
    size_t size;
    size_t capacity;
} task_queue_t;

static task_queue_t* task_queue_create(size_t capacity) {
    task_queue_t* queue = (task_queue_t*)malloc(sizeof(task_queue_t));
    if (!queue) return NULL;

    queue->tasks = (task_t**)malloc(sizeof(task_t*) * capacity);
    if (!queue->tasks) {
        free(queue);
        return NULL;
    }

    queue->size = 0;
    queue->capacity = capacity;
    return queue;
}

static void task_queue_destroy(task_queue_t* queue) {
    if (queue) {
        free(queue->tasks);
        free(queue);
    }
}

static bool task_queue_push(task_queue_t* queue, task_t* task) {
    if (queue->size >= queue->capacity) {
        return false; // 队列满
    }
    queue->tasks[queue->size++] = task;
    return true;
}

static task_t* task_queue_pop(task_queue_t* queue) {
    if (queue->size == 0) {
        return NULL;
    }
    return queue->tasks[--queue->size];
}

// GMP组件现在由外部提供，不再需要临时实现

// ============================================================================
// Executor 聚合根实现
// ============================================================================

/**
 * @brief 创建执行器
 */
executor_t* executor_create(struct Scheduler* scheduler, struct Reactor* reactor) {
    executor_t* executor = (executor_t*)malloc(sizeof(executor_t));
    if (!executor) {
        fprintf(stderr, "Failed to allocate executor\n");
        return NULL;
    }

    executor->id = 1; // 简化ID分配
    executor->scheduler = (scheduler_t*)scheduler; // 使用GMP调度器
    executor->reactor = (reactor_t*)reactor;       // 使用真实反应器
    executor->ready_queue = task_queue_create(DEFAULT_QUEUE_SIZE);
    executor->running = false;

    if (!executor->ready_queue) {
        fprintf(stderr, "Failed to create ready queue\n");
        free(executor);
        return NULL;
    }

    // 为GMP调度器设置调度器上下文（用于协程yield）
    context_t* scheduler_context = context_create(4096); // 4KB栈空间
    if (scheduler_context) {
        scheduler_set_context((scheduler_t*)scheduler, scheduler_context);
        yield_set_scheduler_context(scheduler_context);
        printf("DEBUG: Set scheduler context for GMP scheduler\n");
    } else {
        fprintf(stderr, "WARNING: Failed to create scheduler context\n");
    }

    printf("DEBUG: Created Executor %llu with GMP scheduler\n", executor->id);
    return executor;
}

/**
 * @brief 销毁执行器
 */
void executor_destroy(executor_t* executor) {
    if (!executor) return;

    executor->running = false;

    if (executor->ready_queue) {
        task_queue_destroy(executor->ready_queue);
    }

    printf("DEBUG: Destroyed Executor %llu\n", executor->id);
    free(executor);
}

/**
 * @brief 执行单个任务
 * 内部辅助函数
 */
static void executor_execute_task(executor_t* executor, task_t* task) {
    if (!task) return;

    // 状态转换：就绪 -> 运行中（已经在task_execute中处理）
    // 执行任务
    // 传入调度器上下文（如果有的话）
    context_t* scheduler_ctx = NULL;
    if (executor->scheduler) {
        scheduler_ctx = scheduler_get_context((scheduler_t*)executor->scheduler);
    }
    task_execute(task, scheduler_ctx);

    // 检查任务执行后的状态
    task_status_t status = task_get_status(task);
    switch (status) {
        case TASK_STATUS_COMPLETED:
            printf("DEBUG: Task %llu completed successfully\n", task_get_id(task));
            // 任务完成，通知调度器
            if (executor->scheduler) {
                scheduler_task_completed((scheduler_t*)executor->scheduler, task);
            }
            // 清理已完成的任务
            task_destroy(task);
            break;

        case TASK_STATUS_FAILED:
            printf("DEBUG: Task %llu failed\n", task_get_id(task));
            // 任务失败，通知调度器并清理
            if (executor->scheduler) {
                scheduler_task_completed((scheduler_t*)executor->scheduler, task);
            }
            task_destroy(task);
            break;

        case TASK_STATUS_WAITING:
            printf("DEBUG: Task %llu suspended, waiting for async operation\n",
                   task_get_id(task));
            // 任务等待异步操作，移到阻塞队列
            if (executor->scheduler) {
                scheduler_enqueue_blocked((scheduler_t*)executor->scheduler, task);
            }
            break;

        case TASK_STATUS_CANCELLED:
            printf("DEBUG: Task %llu was cancelled\n", task_get_id(task));
            // 任务被取消，直接清理
            task_destroy(task);
            break;

        default:
            fprintf(stderr, "ERROR: Unexpected task status %d for task %llu\n",
                    status, task_get_id(task));
            break;
    }
}

/**
 * @brief 运行执行循环
 * 核心业务逻辑：实现异步运行时的执行循环
 */
void executor_run_loop(executor_t* executor) {
    if (!executor || !executor->running) {
        return;
    }

    printf("DEBUG: Executor %llu delegating to GMP scheduler main loop\n", executor->id);

    // GMP关键：将控制权交给GMP调度器的主循环
    // GMP调度器负责任务选择、上下文切换和队列管理
    if (executor->scheduler) {
        scheduler_main_loop((scheduler_t*)executor->scheduler);
    } else {
        fprintf(stderr, "ERROR: Executor has no GMP scheduler\n");
    }

    printf("DEBUG: Executor %llu GMP main loop ended\n", executor->id);
}

/**
 * @brief 生成新任务
 */
void executor_spawn_task(executor_t* executor, task_t* task) {
    if (!executor || !task) return;

    // GMP关键：使用GMP调度器来调度任务，实现负载均衡
    if (executor->scheduler) {
        if (scheduler_schedule_task((scheduler_t*)executor->scheduler, task) == 0) {
            printf("DEBUG: GMP Executor spawned task %llu via scheduler\n",
                   task_get_id(task));
        } else {
            fprintf(stderr, "ERROR: GMP scheduler failed to schedule task %llu\n",
                    task_get_id(task));
        }
    } else {
        // 降级到本地队列（如果没有GMP调度器）
        if (task_queue_push(executor->ready_queue, task)) {
            printf("DEBUG: Executor spawned task %llu (fallback to local queue)\n",
                   task_get_id(task));
        } else {
            fprintf(stderr, "ERROR: Failed to spawn task, queues full\n");
        }
    }
}

/**
 * @brief 处理Future
 */
void executor_handle_future(executor_t* executor, struct Future* future) {
    // 简化实现，暂时不处理Future
    (void)executor;
    (void)future;
    printf("DEBUG: Executor handling future (not implemented yet)\n");
}

/**
 * @brief 停止执行器
 */
void executor_shutdown(executor_t* executor) {
    if (executor) {
        printf("DEBUG: Shutting down Executor %llu\n", executor->id);
        executor->running = false;
    }
}

/**
 * @brief 检查执行器是否正在运行
 */
bool executor_is_running(const executor_t* executor) {
    return executor && executor->running;
}

// ============================================================================
// 前向声明
// ============================================================================

// ============================================================================
// 启动执行器（提供给外部使用的函数）
// ============================================================================

// ============================================================================
// 公共API函数（提供给应用层使用）
// ============================================================================

/**
 * @brief 启动执行器
 */
void executor_start(executor_t* executor) {
    if (executor && !executor->running) {
        executor->running = true;
        printf("DEBUG: Started Executor %llu\n", executor->id);
    }
}
