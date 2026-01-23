/**
 * @file future.c
 * @brief Future 聚合根实现
 */

#include "../../include/echo/future.h"
#include "../../include/echo/task.h"
#include "../domain/scheduler/scheduler.h"  // GMP调度器
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

// 全局Future ID计数器
static future_id_t next_future_id = 1;

// ============================================================================
// Future 聚合根实现
// ============================================================================

/**
 * @brief 创建新Future
 */
future_t* future_create(void) {
    future_t* future = (future_t*)malloc(sizeof(future_t));
    if (!future) {
        fprintf(stderr, "Failed to allocate future\n");
        return NULL;
    }

    future->id = next_future_id++;
    future->state = FUTURE_STATE_PENDING;
    future->result = NULL;
    future->consumed = false;
    future->waker = NULL;
    future->user_data = NULL;

    printf("DEBUG: Created future %llu\n", future->id);
    return future;
}

/**
 * @brief 销毁Future
 */
void future_destroy(future_t* future) {
    if (!future) return;

    if (future->waker) {
        waker_destroy(future->waker);
    }

    printf("DEBUG: Destroyed future %llu\n", future->id);
    free(future);
}

/**
 * @brief 轮询Future状态
 */
poll_result_t future_poll(future_t* future, struct Task* task) {
    poll_result_t result = { .is_ready = false, .value = NULL };

    if (!future) return result;

    if (future->state == FUTURE_STATE_RESOLVED) {
        result.is_ready = true;
        result.value = future->result;
        future->consumed = true;
    } else if (future->state == FUTURE_STATE_REJECTED) {
        result.is_ready = true;
        result.value = future->result; // 错误信息
        future->consumed = true;
    }
    // PENDING状态下不做任何操作

    return result;
}

/**
 * @brief 解决Future
 */
void future_resolve(future_t* future, void* value) {
    if (!future || future->state != FUTURE_STATE_PENDING) return;

    future->state = FUTURE_STATE_RESOLVED;
    future->result = value;

    // 唤醒等待的任务
    if (future->waker) {
        waker_wake(future->waker);
    }

    printf("DEBUG: Future %llu resolved with value %p\n", future->id, value);
}

/**
 * @brief 拒绝Future
 */
void future_reject(future_t* future, void* error) {
    if (!future || future->state != FUTURE_STATE_PENDING) return;

    future->state = FUTURE_STATE_REJECTED;
    future->result = error;

    // 唤醒等待的任务
    if (future->waker) {
        waker_wake(future->waker);
    }

    printf("DEBUG: Future %llu rejected with error %p\n", future->id, error);
}

/**
 * @brief 取消Future
 */
void future_cancel(future_t* future) {
    if (!future || future->state != FUTURE_STATE_PENDING) return;

    future->state = FUTURE_STATE_REJECTED;
    future->result = NULL; // 取消没有具体错误

    printf("DEBUG: Future %llu cancelled\n", future->id);
}

/**
 * @brief 获取Future ID
 */
future_id_t future_get_id(const future_t* future) {
    return future ? future->id : 0;
}

/**
 * @brief 获取Future状态
 */
future_state_t future_get_state(const future_t* future) {
    return future ? future->state : FUTURE_STATE_PENDING;
}

/**
 * @brief 检查Future是否就绪
 */
bool future_is_ready(const future_t* future) {
    return future && (future->state == FUTURE_STATE_RESOLVED || future->state == FUTURE_STATE_REJECTED);
}

// ============================================================================
// Waker 实现
// ============================================================================

/**
 * @brief 创建Waker
 */
waker_t* waker_create(struct Task* task, void* scheduler_ref) {
    waker_t* waker = (waker_t*)malloc(sizeof(waker_t));
    if (!waker) {
        fprintf(stderr, "Failed to allocate waker\n");
        return NULL;
    }

    waker->task_id = task ? task_get_id(task) : 0;
    waker->scheduler_ref = scheduler_ref;

    printf("DEBUG: Created waker for task %llu\n", waker->task_id);
    return waker;
}

/**
 * @brief 销毁Waker
 */
void waker_destroy(waker_t* waker) {
    if (waker) {
        printf("DEBUG: Destroyed waker for task %llu\n", waker->task_id);
        free(waker);
    }
}

/**
 * @brief 唤醒任务
 */
void waker_wake(const waker_t* waker) {
    if (!waker || !waker->scheduler_ref) return;

    // 通过GMP调度器唤醒任务
    scheduler_t* scheduler = (scheduler_t*)waker->scheduler_ref;

    // 创建一个临时的任务对象（实际上应该从任务池中获取）
    // 注意：这是一个简化的实现，实际应该维护任务引用
    task_t temp_task;
    memset(&temp_task, 0, sizeof(task_t));
    temp_task.id = waker->task_id;
    temp_task.status = TASK_STATUS_READY; // 标记为就绪

    // 将任务加入调度器的就绪队列
    if (scheduler_enqueue_ready(scheduler, &temp_task)) {
        printf("DEBUG: Waker successfully scheduled task %llu\n", waker->task_id);
    } else {
        fprintf(stderr, "ERROR: Failed to schedule task %llu via waker\n", waker->task_id);
    }
}

/**
 * @brief 释放Waker资源
 */
void waker_drop(const waker_t* waker) {
    // Waker通常在destroy时释放，这里可以做额外清理
    (void)waker;
    printf("DEBUG: Waker drop called\n");
}
