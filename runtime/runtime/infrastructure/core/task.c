/**
 * @file task.c
 * @brief Task 聚合根实现
 *
 * Task 是异步运行时的核心执行单元。
 */

#include "../../include/echo/task.h"
#include "../../include/echo/future.h"
#include "../domain/coroutine/coroutine.h"
#include <stdlib.h>
#include <stdio.h>

// 全局任务ID计数器
static task_id_t next_task_id = 1;

// ============================================================================
// Task 聚合根实现
// ============================================================================

/**
 * @brief 创建新任务
 */
task_t* task_create(void (*entry_point)(void*), void* arg, size_t stack_size) {
    task_t* task = (task_t*)malloc(sizeof(task_t));
    if (!task) {
        fprintf(stderr, "Failed to allocate task\n");
        return NULL;
    }

    task->id = next_task_id++;
    task->status = TASK_STATUS_READY;
    task->future = NULL;
    task->result = NULL;
    task->user_data = arg;

    // 创建协程
    task->coroutine = coroutine_create(entry_point, arg, stack_size, NULL);  // 暂时传入NULL，在execute时设置
    if (!task->coroutine) {
        fprintf(stderr, "Failed to create coroutine for task\n");
        free(task);
        return NULL;
    }

    printf("DEBUG: Created task %llu with coroutine\n", task->id);
    return task;
}

/**
 * @brief 销毁任务
 */
void task_destroy(task_t* task) {
    if (!task) return;

    // 清理协程
    if (task->coroutine) {
        coroutine_destroy(task->coroutine);
        task->coroutine = NULL;
    }

    // 清理Future
    if (task->future) {
        future_destroy(task->future);
        task->future = NULL;
    }

    // 清理用户数据（如果需要）
    if (task->user_data) {
        // 注意：用户数据由创建者负责清理，这里只是置空
        task->user_data = NULL;
    }

    // 清理执行结果
    if (task->result) {
        free(task->result);
        task->result = NULL;
    }

    printf("DEBUG: Destroyed task %llu\n", task->id);
    free(task);
}

/**
 * @brief 执行任务
 */
void task_execute(task_t* task, context_t* scheduler_context) {
    if (!task) return;

    task->status = TASK_STATUS_RUNNING;
    printf("DEBUG: Executing task %llu\n", task->id);

    // 通过协程执行任务
    if (task->coroutine) {
        // 设置调度器上下文（在context.c中会用到）
        // 这里我们通过全局变量传递，因为context_switch需要知道如何返回
        extern context_t* g_scheduler_context;
        if (scheduler_context) {
            g_scheduler_context = scheduler_context;
        } else {
            // 测试环境没有调度器上下文，协程完成时会警告但不会崩溃
            g_scheduler_context = NULL;
        }

        coroutine_resume(task->coroutine);
    }

    // 检查协程是否完成
    if (task->coroutine) {
        bool is_completed = coroutine_is_completed(task->coroutine);
        printf("DEBUG: Task %llu coroutine completed: %s\n", task->id, is_completed ? "true" : "false");
        if (is_completed) {
            task->status = TASK_STATUS_COMPLETED;
            printf("DEBUG: Task %llu completed\n", task->id);
        }
    }
}

/**
 * @brief 挂起任务
 */
void task_suspend(task_t* task) {
    if (task && task->status == TASK_STATUS_RUNNING) {
        task->status = TASK_STATUS_WAITING;
        if (task->coroutine) {
            coroutine_suspend(task->coroutine);
        }
        printf("DEBUG: Task %llu suspended\n", task->id);
    }
}

/**
 * @brief 恢复任务执行
 */
void task_resume(task_t* task) {
    if (task && task->status == TASK_STATUS_WAITING) {
        task->status = TASK_STATUS_READY;
        if (task->coroutine) {
            coroutine_resume(task->coroutine);
        }
        printf("DEBUG: Task %llu resumed\n", task->id);
    }
}

/**
 * @brief 设置任务等待的Future
 */
void task_set_future(task_t* task, struct Future* future) {
    if (task) {
        task->future = future;
        printf("DEBUG: Task %llu set to wait for future\n", task->id);
    }
}

/**
 * @brief 获取任务ID
 */
task_id_t task_get_id(const task_t* task) {
    return task ? task->id : 0;
}

/**
 * @brief 获取任务状态
 */
task_status_t task_get_status(const task_t* task) {
    return task ? task->status : TASK_STATUS_READY;
}

/**
 * @brief 检查任务是否完成
 */
bool task_is_completed(const task_t* task) {
    return task && task->status == TASK_STATUS_COMPLETED;
}

/**
 * @brief 获取任务的协程
 */
struct Coroutine* task_get_coroutine(const task_t* task) {
    return task ? task->coroutine : NULL;
}

// ============================================================================
// 任务状态管理
// ============================================================================

/**
 * @brief 设置任务状态（带状态转换验证）
 */
static bool task_set_status(task_t* task, task_status_t new_status) {
    if (!task) return false;

    // 状态转换验证
    task_status_t current = task->status;
    bool valid_transition = false;

    switch (current) {
        case TASK_STATUS_READY:
            valid_transition = (new_status == TASK_STATUS_RUNNING ||
                              new_status == TASK_STATUS_CANCELLED);
            break;

        case TASK_STATUS_RUNNING:
            valid_transition = (new_status == TASK_STATUS_COMPLETED ||
                              new_status == TASK_STATUS_FAILED ||
                              new_status == TASK_STATUS_WAITING ||
                              new_status == TASK_STATUS_CANCELLED);
            break;

        case TASK_STATUS_WAITING:
            valid_transition = (new_status == TASK_STATUS_READY ||
                              new_status == TASK_STATUS_COMPLETED ||
                              new_status == TASK_STATUS_FAILED ||
                              new_status == TASK_STATUS_CANCELLED);
            break;

        case TASK_STATUS_COMPLETED:
        case TASK_STATUS_FAILED:
        case TASK_STATUS_CANCELLED:
            // 终结状态，不能转换
            valid_transition = false;
            break;
    }

    if (!valid_transition) {
        fprintf(stderr, "ERROR: Invalid status transition from %d to %d for task %llu\n",
                current, new_status, task->id);
        return false;
    }

    task->status = new_status;
    printf("DEBUG: Task %llu status changed from %d to %d\n",
           task->id, current, new_status);

    return true;
}

/**
 * @brief 任务状态转换：准备就绪
 */
bool task_mark_ready(task_t* task) {
    return task_set_status(task, TASK_STATUS_READY);
}

/**
 * @brief 任务状态转换：开始执行
 */
bool task_mark_running(task_t* task) {
    return task_set_status(task, TASK_STATUS_RUNNING);
}

/**
 * @brief 任务状态转换：等待异步操作
 */
bool task_mark_waiting(task_t* task) {
    return task_set_status(task, TASK_STATUS_WAITING);
}

/**
 * @brief 任务状态转换：执行完成
 */
bool task_mark_completed(task_t* task) {
    return task_set_status(task, TASK_STATUS_COMPLETED);
}

/**
 * @brief 任务状态转换：执行失败
 */
bool task_mark_failed(task_t* task) {
    return task_set_status(task, TASK_STATUS_FAILED);
}

/**
 * @brief 任务状态转换：被取消
 */
bool task_mark_cancelled(task_t* task) {
    return task_set_status(task, TASK_STATUS_CANCELLED);
}

/**
 * @brief 检查任务是否可以被调度
 */
bool task_can_schedule(const task_t* task) {
    if (!task) return false;
    return task->status == TASK_STATUS_READY;
}

/**
 * @brief 检查任务是否处于等待状态
 */
bool task_is_waiting(const task_t* task) {
    if (!task) return false;
    return task->status == TASK_STATUS_WAITING;
}

/**
 * @brief 检查任务是否已终结
 */
bool task_is_terminated(const task_t* task) {
    if (!task) return true; // NULL任务视为已终结

    return task->status == TASK_STATUS_COMPLETED ||
           task->status == TASK_STATUS_FAILED ||
           task->status == TASK_STATUS_CANCELLED;
}
