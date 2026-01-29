/**
 * @file future.c
 * @brief AsyncComputation 聚合根实现
 *
 * 实现Future聚合根的所有业务方法，维护异步操作的一致性和生命周期。
 *
 * 关键逻辑：
 * - 工厂方法：创建Future时验证业务规则，初始化所有字段
 * - 业务方法：状态转换时验证业务规则，维护不变条件
 * - 不变条件验证：确保聚合内部状态一致性
 * - 领域事件管理：状态变化时发布领域事件
 * - WaitNode只存储TaskID，通过TaskRepository查找Task
 */

#include "future.h"
#include "../events/future_events.h"
#include "../../shared/events/bus.h"  // 事件总线接口
#include "../../task_execution/repository/task_repository.h"
#include "../../task_execution/aggregate/task.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <time.h>
#include <errno.h>
#include <stdatomic.h>

// ============================================================================
// 全局Future ID生成器
// ============================================================================

static atomic_uint_fast64_t g_next_future_id = ATOMIC_VAR_INIT(1);

/**
 * @brief 生成唯一Future ID
 */
FutureID future_generate_id(void) {
    return atomic_fetch_add(&g_next_future_id, 1);
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
static int future_add_domain_event(Future* future, DomainEvent* domain_event) {
    if (!future || !domain_event) {
        return -1;
    }

    // 如果事件总线存在，通过EventBus发布
    if (future->event_bus && future->event_bus->publish) {
        int result = future->event_bus->publish(future->event_bus, domain_event);
        if (result != 0) {
            fprintf(stderr, "WARNING: Failed to publish event through EventBus for future %llu\n", future->id);
            // 继续执行，至少存储到列表
        }
    }

    // 同时存储到domain_events列表（用于get_domain_events方法）
    // 扩展事件数组容量
    if (future->domain_events_count >= future->domain_events_capacity) {
        size_t new_capacity = future->domain_events_capacity == 0 ? 4 : future->domain_events_capacity * 2;
        struct DomainEvent** new_events = realloc(
            future->domain_events,
            new_capacity * sizeof(struct DomainEvent*)
        );
        if (!new_events) {
            // 如果存储失败，至少事件已经通过EventBus发布了
            return -1;
        }
        future->domain_events = new_events;
        future->domain_events_capacity = new_capacity;
    }

    // 添加事件到列表
    future->domain_events[future->domain_events_count++] = domain_event;
    return 0;
}

// ============================================================================
// 聚合根工厂方法
// ============================================================================

Future* future_factory_create(FuturePriority priority, struct EventBus* event_bus) {
    Future* future = (Future*)malloc(sizeof(Future));
    if (!future) {
        fprintf(stderr, "ERROR: Failed to allocate memory for future\n");
        return NULL;
    }

    memset(future, 0, sizeof(Future));

    // 初始化Future属性
    future->id = future_generate_id();
    future->state = FUTURE_PENDING;
    future->result = NULL;
    future->consumed = false;
    future->wait_queue_head = NULL;
    future->wait_queue_tail = NULL;
    future->wait_count = 0;
    future->priority = priority;

    // 初始化时间戳
    future->created_at = time(NULL);
    future->resolved_at = 0;
    future->rejected_at = 0;

    // 初始化统计信息
    future->poll_count = 0;
    future->wait_time_ms = 0;

    // 初始化同步机制
    if (pthread_mutex_init(&future->mutex, NULL) != 0) {
        fprintf(stderr, "ERROR: Failed to initialize future mutex\n");
        free(future);
        return NULL;
    }
    if (pthread_cond_init(&future->cond, NULL) != 0) {
        fprintf(stderr, "ERROR: Failed to initialize future condition variable\n");
        pthread_mutex_destroy(&future->mutex);
        free(future);
        return NULL;
    }

    // 设置默认方法
    future->cancel = future_cancel;
    future->cleanup = NULL;

    // 初始化领域事件列表
    future->domain_events = NULL;
    future->domain_events_count = 0;
    future->domain_events_capacity = 0;
    
    // 初始化事件总线（可选）
    future->event_bus = event_bus;

    return future;
}

void future_destroy(Future* future) {
    if (!future) return;

    // 调用清理函数
    if (future->cleanup) {
        future->cleanup(future);
    }

    // 清理等待队列
    WaitNode* current = future->wait_queue_head;
    while (current) {
        WaitNode* next = current->next;
        free(current);
        current = next;
    }

    // 清理结果数据（如果需要）
    if (future->result && future->result_needs_free) {
        free(future->result);
        future->result = NULL;
    }

    // 清理领域事件
    if (future->domain_events) {
        for (size_t i = 0; i < future->domain_events_count; i++) {
            // 注意：事件对象的销毁由事件发布者负责
            // 这里只释放数组本身
        }
        free(future->domain_events);
        future->domain_events = NULL;
    }

    // 清理同步机制
    pthread_mutex_destroy(&future->mutex);
    pthread_cond_destroy(&future->cond);

    free(future);
}

// ============================================================================
// 聚合根方法（业务操作）
// ============================================================================

bool future_resolve(Future* future, void* value) {
    if (!future) {
        return false;
    }

    pthread_mutex_lock(&future->mutex);

    // 业务规则：只有PENDING状态的Future可以解决
    if (future->state != FUTURE_PENDING) {
        fprintf(stderr, "ERROR: Cannot resolve future %llu, current state is %d\n",
                future->id, future->state);
        pthread_mutex_unlock(&future->mutex);
        return false;
    }

    // 更新状态
    time_t now = time(NULL);
    future->state = FUTURE_RESOLVED;
    future->result = value;
    future->resolved_at = now;

    // 计算等待时间
    if (future->created_at > 0) {
        future->wait_time_ms = (uint64_t)(now - future->created_at) * 1000;
    }

    // 发布领域事件
    FutureResolvedEvent* event = future_resolved_event_new(future->id, value);
    if (event) {
        DomainEvent* domain_event = future_resolved_event_to_domain_event(event);
        if (domain_event) {
            future_add_domain_event(future, domain_event);
        } else {
            // 转换失败，销毁原始事件
            future_resolved_event_destroy(event);
        }
    }

    pthread_mutex_unlock(&future->mutex);

    // 唤醒所有等待者（需要在锁外调用，避免死锁）
    // 注意：这里需要TaskRepository，但暂时先不调用，由应用层负责
    // future_wake_waiters(future, task_repository);

    // 广播条件变量
    pthread_cond_broadcast(&future->cond);

    return true;
}

bool future_reject(Future* future, void* error) {
    if (!future) {
        return false;
    }

    pthread_mutex_lock(&future->mutex);

    // 业务规则：只有PENDING状态的Future可以拒绝
    if (future->state != FUTURE_PENDING) {
        fprintf(stderr, "ERROR: Cannot reject future %llu, current state is %d\n",
                future->id, future->state);
        pthread_mutex_unlock(&future->mutex);
        return false;
    }

    // 更新状态
    time_t now = time(NULL);
    future->state = FUTURE_REJECTED;
    future->result = error;
    future->rejected_at = now;

    // 计算等待时间
    if (future->created_at > 0) {
        future->wait_time_ms = (uint64_t)(now - future->created_at) * 1000;
    }

    // 发布领域事件
    FutureRejectedEvent* event = future_rejected_event_new(future->id, error);
    if (event) {
        DomainEvent* domain_event = future_rejected_event_to_domain_event(event);
        if (domain_event) {
            future_add_domain_event(future, domain_event);
        } else {
            // 转换失败，销毁原始事件
            future_rejected_event_destroy(event);
        }
    }

    pthread_mutex_unlock(&future->mutex);

    // 唤醒所有等待者（需要在锁外调用，避免死锁）
    // 注意：这里需要TaskRepository，但暂时先不调用，由应用层负责
    // future_wake_waiters(future, task_repository);

    // 广播条件变量
    pthread_cond_broadcast(&future->cond);

    return true;
}

bool future_add_waiter(Future* future, TaskID task_id) {
    if (!future) {
        return false;
    }

    pthread_mutex_lock(&future->mutex);

    // 业务规则：只有PENDING状态的Future可以添加等待者
    if (future->state != FUTURE_PENDING) {
        fprintf(stderr, "WARN: Cannot add waiter to future %llu, current state is %d\n",
                future->id, future->state);
        pthread_mutex_unlock(&future->mutex);
        return false;
    }

    // 创建等待节点（只存储TaskID）
    WaitNode* node = (WaitNode*)malloc(sizeof(WaitNode));
    if (!node) {
        fprintf(stderr, "ERROR: Failed to allocate wait node\n");
        pthread_mutex_unlock(&future->mutex);
        return false;
    }

    node->task_id = task_id;  // 只存储TaskID，不存储Task*
    node->next = NULL;

    // 添加到等待队列
    if (future->wait_queue_tail) {
        future->wait_queue_tail->next = node;
        future->wait_queue_tail = node;
    } else {
        future->wait_queue_head = node;
        future->wait_queue_tail = node;
    }

    future->wait_count++;

    pthread_mutex_unlock(&future->mutex);

    return true;
}

bool future_wake_waiters(Future* future, void* task_repository) {
    if (!future) {
        return false;
    }

    // 注意：task_repository 是 TaskRepository* 类型，但为了避免循环依赖，使用 void*
    // 调用者需要确保传入正确的 TaskRepository* 指针
    TaskRepository* repo = (TaskRepository*)task_repository;
    if (!repo) {
        fprintf(stderr, "WARN: TaskRepository is NULL, cannot wake waiters\n");
        return false;
    }

    pthread_mutex_lock(&future->mutex);

    WaitNode* current = future->wait_queue_head;
    size_t woken_count = 0;

    while (current) {
        WaitNode* next = current->next;

        // 通过TaskRepository根据TaskID查找Task
        Task* task = task_repository_find_by_id(repo, current->task_id);
        if (task) {
            // 唤醒Task（调用Task聚合根方法）
            // 注意：task_wake 是Task聚合根的方法，需要确保已实现
            // 使用 task_execution/aggregate/task.h 中的 Task 类型
            // 但 task_wake 在 runtime/domain/task/entity/task.c 中定义
            // 暂时注释掉，因为 Task 类型不匹配
            // extern void task_wake(Task* task);  // 前向声明，实际在task.c中实现
            // task_wake(task);
            // TODO: 需要统一 Task 类型，或者使用 TaskRepository 的方法来唤醒任务
            woken_count++;
        } else {
            fprintf(stderr, "WARN: Task %llu not found in repository, cannot wake\n",
                    current->task_id);
        }

        free(current);
        current = next;
    }

    future->wait_queue_head = NULL;
    future->wait_queue_tail = NULL;
    future->wait_count = 0;

    pthread_mutex_unlock(&future->mutex);

    // 广播条件变量
    pthread_cond_broadcast(&future->cond);

    return woken_count > 0;
}

PollResult future_poll(Future* future, TaskID task_id) {
    PollResult result = { POLL_PENDING, NULL };

    if (!future) {
        return result;
    }

    pthread_mutex_lock(&future->mutex);

    // 更新轮询统计
    future->poll_count++;

    // 检查Future状态
    if (future->state == FUTURE_RESOLVED) {
        if (!future->consumed) {
            result.status = POLL_READY;
            result.value = future->result;
            future->consumed = true; // 标记为已消费
        } else {
            result.status = POLL_COMPLETED;
            result.value = future->result;
        }
    } else if (future->state == FUTURE_REJECTED) {
        if (!future->consumed) {
            result.status = POLL_READY;
            result.value = future->result; // 这里是错误信息
            future->consumed = true;
        } else {
            result.status = POLL_COMPLETED;
            result.value = future->result;
        }
    } else {
        // 仍然等待中
        result.status = POLL_PENDING;
        result.value = NULL;

        // 如果有Task ID，将其加入等待队列
        if (task_id != 0) {
            future_add_waiter(future, task_id);
        }
    }

    pthread_mutex_unlock(&future->mutex);

    return result;
}

bool future_cancel(Future* future) {
    if (!future) {
        return false;
    }

    pthread_mutex_lock(&future->mutex);

    // 业务规则：只有PENDING状态的Future可以取消
    if (future->state != FUTURE_PENDING) {
        fprintf(stderr, "WARN: Cannot cancel future %llu, current state is %d\n",
                future->id, future->state);
        pthread_mutex_unlock(&future->mutex);
        return false;
    }

    // 取消Future，状态变为REJECTED
    time_t now = time(NULL);
    future->state = FUTURE_REJECTED;
    future->result = NULL; // 取消没有具体错误信息
    future->rejected_at = now;

    pthread_mutex_unlock(&future->mutex);

    // 唤醒所有等待者（需要在锁外调用，避免死锁）
    // 注意：这里需要TaskRepository，但暂时先不调用，由应用层负责
    // future_wake_waiters(future, task_repository);

    // 广播条件变量
    pthread_cond_broadcast(&future->cond);

    return true;
}

// ============================================================================
// 查询方法（只读访问）
// ============================================================================

FutureID future_get_id(const Future* future) {
    if (!future) return 0;
    return future->id;
}

FutureState future_get_state(const Future* future) {
    if (!future) return FUTURE_PENDING;
    return future->state;
}

bool future_is_complete(const Future* future) {
    if (!future) return false;
    return future->state == FUTURE_RESOLVED || future->state == FUTURE_REJECTED;
}

bool future_is_consumed(const Future* future) {
    if (!future) return false;
    return future->consumed;
}

const void* future_get_result(const Future* future) {
    if (!future) return NULL;
    return future->result;
}

// ============================================================================
// 不变条件验证
// ============================================================================

bool future_validate_invariants(const Future* future) {
    if (!future) {
        return false;
    }

    // 验证规则1：等待队列中的TaskID必须有效（非0）
    WaitNode* current = future->wait_queue_head;
    while (current) {
        if (current->task_id == 0) {
            fprintf(stderr, "ERROR: Invalid TaskID (0) in wait queue of future %llu\n",
                    future->id);
            return false;
        }
        current = current->next;
    }

    // 验证规则2：Future状态必须有效
    if (future->state != FUTURE_PENDING &&
        future->state != FUTURE_RESOLVED &&
        future->state != FUTURE_REJECTED) {
        fprintf(stderr, "ERROR: Invalid state %d for future %llu\n",
                future->state, future->id);
        return false;
    }

    // 验证规则3：已解决的Future不能再次解决或拒绝
    // （这个规则在业务方法中已经验证，这里只是双重检查）

    // 验证规则4：等待队列计数必须与实际节点数一致
    int actual_count = 0;
    current = future->wait_queue_head;
    while (current) {
        actual_count++;
        current = current->next;
    }
    if (actual_count != future->wait_count) {
        fprintf(stderr, "ERROR: Wait count mismatch for future %llu: expected %d, actual %d\n",
                future->id, future->wait_count, actual_count);
        return false;
    }

    return true;
}

// ============================================================================
// 领域事件
// ============================================================================

bool future_get_domain_events(Future* future, struct DomainEvent*** events, size_t* count) {
    if (!future || !events || !count) {
        return false;
    }

    *events = future->domain_events;
    *count = future->domain_events_count;

    // 清空事件列表（事件已传递给调用者）
    future->domain_events = NULL;
    future->domain_events_count = 0;
    future->domain_events_capacity = 0;

    return true;
}

// ============================================================================
// 工具函数（兼容现有代码）
// ============================================================================

void* future_wait(Future* future) {
    if (!future) return NULL;

    time_t start_time = time(NULL);

    pthread_mutex_lock(&future->mutex);

    while (future->state == FUTURE_PENDING) {
        pthread_cond_wait(&future->cond, &future->mutex);
    }

    void* result = NULL;
    if (future->state == FUTURE_RESOLVED) {
        result = future->result;
        future->consumed = true;
    }

    // 记录等待时间（如果还没有记录）
    if (future->wait_time_ms == 0) {
        time_t end_time = time(NULL);
        future->wait_time_ms = (uint64_t)(end_time - start_time) * 1000;
    }

    pthread_mutex_unlock(&future->mutex);

    return result;
}
