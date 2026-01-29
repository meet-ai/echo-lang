/**
 * @file task.c
 * @brief TaskExecution 聚合根实现
 *
 * 实现Task聚合根的所有业务方法，维护任务执行的一致性和生命周期。
 *
 * 关键逻辑：
 * - 工厂方法：创建任务时验证业务规则，初始化所有字段
 * - 业务方法：状态转换时验证业务规则，维护不变条件
 * - 不变条件验证：确保聚合内部状态一致性
 * - 领域事件管理：状态变化时发布领域事件
 */

#include "task.h"
#include "../events/task_events.h"
#include "../../shared/events/bus.h"  // 事件总线接口
#include "coroutine.h"  // 内部实体，在同一聚合目录内
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <time.h>
#include <errno.h>

// ============================================================================
// 全局任务ID生成器
// ============================================================================

static uint64_t g_next_task_id = 1;
static pthread_mutex_t g_task_id_mutex = PTHREAD_MUTEX_INITIALIZER;

/**
 * @brief 生成唯一任务ID
 */
static TaskID generate_task_id(void) {
    pthread_mutex_lock(&g_task_id_mutex);
    TaskID id = g_next_task_id++;
    pthread_mutex_unlock(&g_task_id_mutex);
    return id;
}

// ============================================================================
// 内部辅助函数
// ============================================================================

/**
 * @brief 发布领域事件（通过事件总线或存储到列表）
 * 
 * 如果event_bus存在，通过EventBus发布事件。
 * 同时存储到domain_events列表（用于get_domain_events方法）。
 */
static int task_add_domain_event(Task* task, DomainEvent* domain_event) {
    if (!task || !domain_event) {
        return -1;
    }

    // 如果事件总线存在，通过EventBus发布
    if (task->event_bus && task->event_bus->publish) {
        int result = task->event_bus->publish(task->event_bus, domain_event);
        if (result != 0) {
            fprintf(stderr, "WARNING: Failed to publish event through EventBus for task %llu\n", task->id);
            // 继续执行，至少存储到列表
        }
    }

    // 同时存储到domain_events列表（用于get_domain_events方法）
    // 扩展事件数组容量
    if (task->domain_events_count >= task->domain_events_capacity) {
        size_t new_capacity = task->domain_events_capacity == 0 ? 4 : task->domain_events_capacity * 2;
        struct DomainEvent** new_events = realloc(task->domain_events, new_capacity * sizeof(struct DomainEvent*));
        if (!new_events) {
            // 如果存储失败，至少事件已经通过EventBus发布了
            return -1;
        }
        task->domain_events = new_events;
        task->domain_events_capacity = new_capacity;
    }

    // 添加事件到列表
    task->domain_events[task->domain_events_count++] = domain_event;
    return 0;
}

/**
 * @brief 更新任务状态（内部方法，需要持有锁）
 */
static int task_set_status_locked(Task* task, TaskStatus new_status) {
    if (!task) {
        return -1;
    }

    // 验证状态转换合法性
    if (!task_is_valid_state_transition(task, new_status)) {
        fprintf(stderr, "ERROR: Invalid state transition from %d to %d for task %llu\n",
                task->status, new_status, task->id);
        return -1;
    }

    task->status = new_status;
    return 0;
}

// ============================================================================
// 聚合根工厂方法
// ============================================================================

Task* task_factory_create(void (*entry_point)(void*), void* arg, size_t stack_size, struct EventBus* event_bus) {
    // 业务规则验证：entry_point不能为NULL
    if (!entry_point) {
        fprintf(stderr, "ERROR: entry_point cannot be NULL\n");
        return NULL;
    }

    // 业务规则验证：stack_size必须大于等于4096字节
    if (stack_size < 4096) {
        fprintf(stderr, "ERROR: stack_size must be at least 4096 bytes, got %zu\n", stack_size);
        return NULL;
    }

    // 分配内存
    Task* task = (Task*)calloc(1, sizeof(Task));
    if (!task) {
        fprintf(stderr, "ERROR: Failed to allocate memory for task\n");
        return NULL;
    }

    // 初始化聚合根标识
    task->id = generate_task_id();

    // 初始化状态值对象
    task->status = TASK_STATUS_CREATED;
    task->priority = TASK_PRIORITY_NORMAL;

    // 初始化执行参数
    task->entry_point = entry_point;
    task->arg = arg;
    task->stack_size = stack_size;

    // 初始化外部聚合引用（通过ID）
    task->waiting_future_id = 0;

    // 初始化内部实体
    task->coroutine = NULL;

    // 初始化执行结果
    task->result = NULL;
    task->exit_code = 0;
    memset(task->error_message, 0, sizeof(task->error_message));

    // 初始化时间戳
    time_t now = time(NULL);
    task->created_at = now;
    task->started_at = 0;
    task->completed_at = 0;

    // 初始化同步机制
    if (pthread_mutex_init(&task->mutex, NULL) != 0) {
        fprintf(stderr, "ERROR: Failed to initialize mutex for task %llu\n", task->id);
        free(task);
        return NULL;
    }
    if (pthread_cond_init(&task->cond, NULL) != 0) {
        fprintf(stderr, "ERROR: Failed to initialize cond for task %llu\n", task->id);
        pthread_mutex_destroy(&task->mutex);
        free(task);
        return NULL;
    }

    // 初始化领域事件列表
    task->domain_events = NULL;
    task->domain_events_count = 0;
    task->domain_events_capacity = 0;
    
    // 初始化事件总线（可选）
    task->event_bus = event_bus;

    // 初始化队列管理
    task->next = NULL;

    // 初始化回调函数
    task->on_complete = NULL;
    task->user_data = NULL;

    // 发布TaskCreated事件
    TaskCreatedEvent* event = task_created_event_create(task->id, task->priority);
    if (event) {
        DomainEvent* domain_event = task_created_event_to_domain_event(event);
        if (domain_event) {
            task_add_domain_event(task, domain_event);
        } else {
            // 转换失败，销毁原始事件
            task_created_event_destroy(event);
        }
    }

    // 验证不变条件
    if (!task_validate_invariants(task)) {
        fprintf(stderr, "ERROR: Invariants violated after creating task %llu\n", task->id);
        task_destroy(task);
        return NULL;
    }

    return task;
}

// ============================================================================
// 聚合根业务方法
// ============================================================================

int task_start(Task* task) {
    if (!task) {
        return -1;
    }

    pthread_mutex_lock(&task->mutex);

    // 业务规则：只有TASK_STATUS_CREATED或TASK_STATUS_READY状态的任务可以启动
    if (task->status != TASK_STATUS_CREATED && task->status != TASK_STATUS_READY) {
        fprintf(stderr, "ERROR: Task %llu cannot be started from status %d\n", task->id, task->status);
        return -1;
    }

    // 创建Coroutine内部实体
    // TODO: 实现Coroutine创建逻辑
    // task->coroutine = coroutine_create(task->entry_point, task->arg, task->stack_size);
    // if (!task->coroutine) {
    //     fprintf(stderr, "ERROR: Failed to create coroutine for task %llu\n", task->id);
    //     return -1;
    // }

    // 更新状态
    if (task_set_status_locked(task, TASK_STATUS_RUNNING) != 0) {
        return -1;
    }

    // 更新时间戳
    task->started_at = time(NULL);

    // 发布TaskStarted事件
    TaskStartedEvent* event = task_started_event_create(task->id);
    if (event) {
        DomainEvent* domain_event = task_started_event_to_domain_event(event);
        if (domain_event) {
            task_add_domain_event(task, domain_event);
        } else {
            // 转换失败，销毁原始事件
            task_started_event_destroy(event);
        }
    }

    // 验证不变条件
    if (!task_validate_invariants(task)) {
        fprintf(stderr, "ERROR: Invariants violated after starting task %llu\n", task->id);
        pthread_mutex_unlock(&task->mutex);
        return -1;
    }

    pthread_mutex_unlock(&task->mutex);
    return 0;
}

int task_suspend(Task* task) {
    if (!task) {
        return -1;
    }

    pthread_mutex_lock(&task->mutex);

    // 业务规则：只有TASK_STATUS_RUNNING状态的任务可以挂起
    if (task->status != TASK_STATUS_RUNNING) {
        fprintf(stderr, "ERROR: Task %llu cannot be suspended from status %d\n", task->id, task->status);
        return -1;
    }

    // 更新状态
    if (task_set_status_locked(task, TASK_STATUS_WAITING) != 0) {
        return -1;
    }

    // 验证不变条件
    if (!task_validate_invariants(task)) {
        fprintf(stderr, "ERROR: Invariants violated after suspending task %llu\n", task->id);
        pthread_mutex_unlock(&task->mutex);
        return -1;
    }

    pthread_mutex_unlock(&task->mutex);
    return 0;
}

int task_wait_for_future(Task* task, FutureID future_id) {
    if (!task) {
        return -1;
    }

    // 业务规则验证：future_id必须有效（非0）
    if (future_id == 0) {
        fprintf(stderr, "ERROR: Invalid future_id (0) for task %llu\n", task->id);
        return -1;
    }

    pthread_mutex_lock(&task->mutex);

    // 业务规则：只有TASK_STATUS_RUNNING状态的任务可以等待Future
    if (task->status != TASK_STATUS_RUNNING) {
        fprintf(stderr, "ERROR: Task %llu cannot wait for future from status %d\n", task->id, task->status);
        return -1;
    }

    // 设置waiting_future_id
    task->waiting_future_id = future_id;

    // 更新状态
    if (task_set_status_locked(task, TASK_STATUS_WAITING) != 0) {
        return -1;
    }

    // 发布TaskWaitingForFuture事件
    TaskWaitingForFutureEvent* event = task_waiting_for_future_event_create(task->id, future_id);
    if (event) {
        DomainEvent* domain_event = task_waiting_for_future_event_to_domain_event(event);
        if (domain_event) {
            task_add_domain_event(task, domain_event);
        } else {
            // 转换失败，销毁原始事件
            task_waiting_for_future_event_destroy(event);
        }
    }

    // 验证不变条件
    if (!task_validate_invariants(task)) {
        fprintf(stderr, "ERROR: Invariants violated after waiting for future %llu in task %llu\n",
                future_id, task->id);
        pthread_mutex_unlock(&task->mutex);
        return -1;
    }

    pthread_mutex_unlock(&task->mutex);
    return 0;
}

int task_handle_future_resolved(Task* task, FutureID future_id, void* result) {
    if (!task) {
        return -1;
    }

    pthread_mutex_lock(&task->mutex);

    // 业务规则：只有TASK_STATUS_WAITING状态的任务可以处理Future完成
    if (task->status != TASK_STATUS_WAITING) {
        fprintf(stderr, "ERROR: Task %llu cannot handle future resolved from status %d\n",
                task->id, task->status);
        return -1;
    }

    // 业务规则：waiting_future_id必须匹配
    if (task->waiting_future_id != future_id) {
        fprintf(stderr, "ERROR: Task %llu is waiting for future %llu, not %llu\n",
                task->id, task->waiting_future_id, future_id);
        return -1;
    }

    // 清除waiting_future_id
    task->waiting_future_id = 0;

    // 更新状态
    if (task_set_status_locked(task, TASK_STATUS_RUNNING) != 0) {
        return -1;
    }

    // 验证不变条件
    if (!task_validate_invariants(task)) {
        fprintf(stderr, "ERROR: Invariants violated after handling future resolved in task %llu\n", task->id);
        pthread_mutex_unlock(&task->mutex);
        return -1;
    }

    pthread_mutex_unlock(&task->mutex);
    return 0;
}

int task_complete(Task* task, void* result) {
    if (!task) {
        return -1;
    }

    pthread_mutex_lock(&task->mutex);

    // 业务规则：只有TASK_STATUS_RUNNING状态的任务可以完成
    if (task->status != TASK_STATUS_RUNNING) {
        fprintf(stderr, "ERROR: Task %llu cannot be completed from status %d\n", task->id, task->status);
        return -1;
    }

    // 保存执行结果
    task->result = result;
    task->exit_code = 0;

    // 更新状态
    if (task_set_status_locked(task, TASK_STATUS_COMPLETED) != 0) {
        return -1;
    }

    // 更新时间戳
    task->completed_at = time(NULL);

    // 发布TaskCompleted事件
    TaskCompletedEvent* event = task_completed_event_create(task->id, result);
    if (event) {
        DomainEvent* domain_event = task_completed_event_to_domain_event(event);
        if (domain_event) {
            task_add_domain_event(task, domain_event);
        } else {
            // 转换失败，销毁原始事件
            task_completed_event_destroy(event);
        }
    }

    // 验证不变条件
    if (!task_validate_invariants(task)) {
        fprintf(stderr, "ERROR: Invariants violated after completing task %llu\n", task->id);
        return -1;
    }

    // 触发完成回调
    if (task->on_complete) {
        task->on_complete(task);
    }

    // 通知等待的线程
    pthread_cond_broadcast(&task->cond);

    pthread_mutex_unlock(&task->mutex);
    return 0;
}

int task_fail(Task* task, const char* error_message) {
    if (!task) {
        return -1;
    }

    pthread_mutex_lock(&task->mutex);

    // 业务规则：只有TASK_STATUS_RUNNING状态的任务可以失败
    if (task->status != TASK_STATUS_RUNNING) {
        fprintf(stderr, "ERROR: Task %llu cannot be failed from status %d\n", task->id, task->status);
        return -1;
    }

    // 保存错误信息
    if (error_message) {
        strncpy(task->error_message, error_message, sizeof(task->error_message) - 1);
        task->error_message[sizeof(task->error_message) - 1] = '\0';
    } else {
        task->error_message[0] = '\0';
    }
    task->exit_code = -1;

    // 更新状态
    if (task_set_status_locked(task, TASK_STATUS_FAILED) != 0) {
        return -1;
    }

    // 更新时间戳
    task->completed_at = time(NULL);

    // 发布TaskFailed事件
    TaskFailedEvent* event = task_failed_event_create(task->id, error_message ? error_message : "Unknown error");
    if (event) {
        DomainEvent* domain_event = task_failed_event_to_domain_event(event);
        if (domain_event) {
            task_add_domain_event(task, domain_event);
        } else {
            // 转换失败，销毁原始事件
            task_failed_event_destroy(event);
        }
    }

    // 验证不变条件
    if (!task_validate_invariants(task)) {
        fprintf(stderr, "ERROR: Invariants violated after failing task %llu\n", task->id);
        return -1;
    }

    // 通知等待的线程
    pthread_cond_broadcast(&task->cond);

    pthread_mutex_unlock(&task->mutex);
    return 0;
}

int task_cancel(Task* task) {
    if (!task) {
        return -1;
    }

    pthread_mutex_lock(&task->mutex);

    // 业务规则：只有TASK_STATUS_CREATED、TASK_STATUS_READY或TASK_STATUS_WAITING状态的任务可以取消
    if (task->status != TASK_STATUS_CREATED &&
        task->status != TASK_STATUS_READY &&
        task->status != TASK_STATUS_WAITING) {
        fprintf(stderr, "ERROR: Task %llu cannot be cancelled from status %d\n", task->id, task->status);
        return -1;
    }

    // 更新状态
    if (task_set_status_locked(task, TASK_STATUS_CANCELLED) != 0) {
        return -1;
    }

    // 更新时间戳
    task->completed_at = time(NULL);

    // 发布TaskCancelled事件
    TaskCancelledEvent* event = task_cancelled_event_create(task->id, "User cancelled");
    if (event) {
        DomainEvent* domain_event = task_cancelled_event_to_domain_event(event);
        if (domain_event) {
            task_add_domain_event(task, domain_event);
        } else {
            // 转换失败，销毁原始事件
            task_cancelled_event_destroy(event);
        }
    }

    // 验证不变条件
    if (!task_validate_invariants(task)) {
        fprintf(stderr, "ERROR: Invariants violated after cancelling task %llu\n", task->id);
        return -1;
    }

    // 通知等待的线程
    pthread_cond_broadcast(&task->cond);

    pthread_mutex_unlock(&task->mutex);
    return 0;
}

// ============================================================================
// 不变条件验证
// ============================================================================

bool task_validate_invariants(const Task* task) {
    if (!task) {
        return false;
    }

    // IC1：如果status为TASK_STATUS_WAITING，waiting_future_id必须有效（非0）
    if (task->status == TASK_STATUS_WAITING) {
        if (task->waiting_future_id == 0) {
            fprintf(stderr, "ERROR: Invariant IC1 violated: TASK_STATUS_WAITING but waiting_future_id is 0\n");
            return false;
        }
    }

    // IC2：如果status为TASK_STATUS_RUNNING，coroutine必须存在
    // 注意：由于Coroutine创建逻辑还未实现，暂时注释
    // if (task->status == TASK_STATUS_RUNNING) {
    //     if (!task->coroutine) {
    //         fprintf(stderr, "ERROR: Invariant IC2 violated: TASK_STATUS_RUNNING but coroutine is NULL\n");
    //         return false;
    //     }
    // }

    // IC3：如果status为TASK_STATUS_COMPLETED或TASK_STATUS_FAILED，completed_at必须有效
    if (task->status == TASK_STATUS_COMPLETED || task->status == TASK_STATUS_FAILED) {
        if (task->completed_at == 0) {
            fprintf(stderr, "ERROR: Invariant IC3 violated: status is %d but completed_at is 0\n", task->status);
            return false;
        }
    }

    return true;
}

bool task_is_valid_state_transition(const Task* task, TaskStatus new_status) {
    if (!task) {
        return false;
    }

    TaskStatus current_status = task->status;

    // 终态不能转换
    if (current_status == TASK_STATUS_COMPLETED ||
        current_status == TASK_STATUS_FAILED ||
        current_status == TASK_STATUS_CANCELLED) {
        return false;
    }

    // 状态转换规则
    switch (current_status) {
        case TASK_STATUS_CREATED:
            return new_status == TASK_STATUS_READY || new_status == TASK_STATUS_CANCELLED;

        case TASK_STATUS_READY:
            return new_status == TASK_STATUS_RUNNING || new_status == TASK_STATUS_CANCELLED;

        case TASK_STATUS_RUNNING:
            return new_status == TASK_STATUS_WAITING ||
                   new_status == TASK_STATUS_COMPLETED ||
                   new_status == TASK_STATUS_FAILED;

        case TASK_STATUS_WAITING:
            return new_status == TASK_STATUS_RUNNING || new_status == TASK_STATUS_CANCELLED;

        default:
            return false;
    }
}

// ============================================================================
// 领域事件管理
// ============================================================================

int task_get_domain_events(Task* task, struct DomainEvent*** events, size_t* count) {
    if (!task || !events || !count) {
        return -1;
    }

    pthread_mutex_lock(&task->mutex);

    *events = task->domain_events;
    *count = task->domain_events_count;

    // 清空事件列表（但不释放内存，由调用者负责释放事件对象）
    task->domain_events = NULL;
    task->domain_events_count = 0;
    task->domain_events_capacity = 0;

    pthread_mutex_unlock(&task->mutex);
    return 0;
}

// ============================================================================
// 查询方法（只读访问）
// ============================================================================

TaskID task_get_id(const Task* task) {
    return task ? task->id : 0;
}

TaskStatus task_get_status(const Task* task) {
    if (!task) {
        return TASK_STATUS_CREATED;
    }
    pthread_mutex_lock((pthread_mutex_t*)&task->mutex);
    TaskStatus status = task->status;
    pthread_mutex_unlock((pthread_mutex_t*)&task->mutex);
    return status;
}

FutureID task_get_waiting_future_id(const Task* task) {
    if (!task) {
        return 0;
    }
    pthread_mutex_lock((pthread_mutex_t*)&task->mutex);
    FutureID future_id = task->waiting_future_id;
    pthread_mutex_unlock((pthread_mutex_t*)&task->mutex);
    return future_id;
}

const struct Coroutine* task_get_coroutine(const Task* task) {
    if (!task) {
        return NULL;
    }
    pthread_mutex_lock((pthread_mutex_t*)&task->mutex);
    const struct Coroutine* coroutine = task->coroutine;
    pthread_mutex_unlock((pthread_mutex_t*)&task->mutex);
    return coroutine;
}

bool task_is_complete(const Task* task) {
    return task && task_get_status(task) == TASK_STATUS_COMPLETED;
}

bool task_is_failed(const Task* task) {
    return task && task_get_status(task) == TASK_STATUS_FAILED;
}

bool task_is_cancelled(const Task* task) {
    return task && task_get_status(task) == TASK_STATUS_CANCELLED;
}

// ============================================================================
// 资源管理
// ============================================================================

void task_destroy(Task* task) {
    if (!task) {
        return;
    }

    // 清理Coroutine内部实体
    if (task->coroutine) {
        // TODO: 实现Coroutine销毁逻辑
        // coroutine_destroy(task->coroutine);
        task->coroutine = NULL;
    }

    // 清理领域事件（由调用者负责释放事件对象，这里只清理数组）
    if (task->domain_events) {
        free(task->domain_events);
        task->domain_events = NULL;
    }
    task->domain_events_count = 0;
    task->domain_events_capacity = 0;

    // 清理同步机制
    pthread_mutex_destroy(&task->mutex);
    pthread_cond_destroy(&task->cond);

    // 释放内存
    free(task);
}
