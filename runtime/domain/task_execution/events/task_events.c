/**
 * @file task_events.c
 * @brief TaskExecution 聚合领域事件实现
 *
 * 实现所有Task相关领域事件的创建和销毁逻辑。
 */

#include "task_events.h"
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <stdint.h>

// ============================================================================
// 事件ID生成
// ============================================================================

static uint64_t generate_event_id(void) {
    static uint64_t next_event_id = 1;
    return next_event_id++;
}

// ============================================================================
// TaskCreated 事件实现
// ============================================================================

TaskCreatedEvent* task_created_event_create(uint64_t task_id, int priority) {
    TaskCreatedEvent* event = (TaskCreatedEvent*)calloc(1, sizeof(TaskCreatedEvent));
    if (!event) {
        return NULL;
    }

    // 初始化事件数据
    event->event_id = generate_event_id();
    event->task_id = task_id;
    event->priority = priority;
    event->created_at = time(NULL);

    return event;
}

void task_created_event_destroy(TaskCreatedEvent* event) {
    if (event) {
        free(event);
    }
}

// ============================================================================
// TaskStarted 事件实现
// ============================================================================

TaskStartedEvent* task_started_event_create(uint64_t task_id) {
    TaskStartedEvent* event = (TaskStartedEvent*)calloc(1, sizeof(TaskStartedEvent));
    if (!event) {
        return NULL;
    }

    // 初始化事件数据
    event->event_id = generate_event_id();
    event->task_id = task_id;
    event->started_at = time(NULL);

    return event;
}

void task_started_event_destroy(TaskStartedEvent* event) {
    if (event) {
        free(event);
    }
}

// ============================================================================
// TaskCompleted 事件实现
// ============================================================================

TaskCompletedEvent* task_completed_event_create(uint64_t task_id, void* result) {
    TaskCompletedEvent* event = (TaskCompletedEvent*)calloc(1, sizeof(TaskCompletedEvent));
    if (!event) {
        return NULL;
    }

    // 初始化事件数据
    event->event_id = generate_event_id();
    event->task_id = task_id;
    event->result = result;
    event->completed_at = time(NULL);

    return event;
}

void task_completed_event_destroy(TaskCompletedEvent* event) {
    if (event) {
        free(event);
    }
}

// ============================================================================
// TaskFailed 事件实现
// ============================================================================

TaskFailedEvent* task_failed_event_create(uint64_t task_id, const char* error_message) {
    TaskFailedEvent* event = (TaskFailedEvent*)calloc(1, sizeof(TaskFailedEvent));
    if (!event) {
        return NULL;
    }

    // 初始化事件数据
    event->event_id = generate_event_id();
    event->task_id = task_id;
    event->error_message = error_message ? strdup(error_message) : NULL;
    event->failed_at = time(NULL);

    return event;
}

void task_failed_event_destroy(TaskFailedEvent* event) {
    if (event) {
        if (event->error_message) {
            free((void*)event->error_message);
        }
        free(event);
    }
}

// ============================================================================
// TaskCancelled 事件实现
// ============================================================================

TaskCancelledEvent* task_cancelled_event_create(uint64_t task_id, const char* reason) {
    TaskCancelledEvent* event = (TaskCancelledEvent*)calloc(1, sizeof(TaskCancelledEvent));
    if (!event) {
        return NULL;
    }

    // 初始化事件数据
    event->event_id = generate_event_id();
    event->task_id = task_id;
    event->reason = reason ? strdup(reason) : NULL;
    event->cancelled_at = time(NULL);

    return event;
}

void task_cancelled_event_destroy(TaskCancelledEvent* event) {
    if (event) {
        if (event->reason) {
            free((void*)event->reason);
        }
        free(event);
    }
}

// ============================================================================
// TaskWaitingForFuture 事件实现
// ============================================================================

TaskWaitingForFutureEvent* task_waiting_for_future_event_create(uint64_t task_id, uint64_t future_id) {
    TaskWaitingForFutureEvent* event = (TaskWaitingForFutureEvent*)calloc(1, sizeof(TaskWaitingForFutureEvent));
    if (!event) {
        return NULL;
    }

    // 初始化事件数据
    event->event_id = generate_event_id();
    event->task_id = task_id;
    event->future_id = future_id;
    event->waiting_at = time(NULL);

    return event;
}

void task_waiting_for_future_event_destroy(TaskWaitingForFutureEvent* event) {
    if (event) {
        free(event);
    }
}

// ==================== DomainEvent接口实现 ====================

// TaskCreatedEvent的DomainEvent接口实现

static const char* task_created_event_get_type(const DomainEvent* event) {
    (void)event;  // 未使用
    return "TaskCreated";
}

static time_t task_created_event_get_occurred_at(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    TaskCreatedEvent* created_event = (TaskCreatedEvent*)event->data;
    return created_event->created_at;
}

static uint64_t task_created_event_get_id(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    TaskCreatedEvent* created_event = (TaskCreatedEvent*)event->data;
    return created_event->event_id;
}

static void task_created_event_destroy_wrapper(DomainEvent* event) {
    if (!event || !event->data) return;
    TaskCreatedEvent* created_event = (TaskCreatedEvent*)event->data;
    task_created_event_destroy(created_event);
    free(event);
}

DomainEvent* task_created_event_to_domain_event(TaskCreatedEvent* event) {
    if (!event) return NULL;
    
    DomainEvent* domain_event = (DomainEvent*)malloc(sizeof(DomainEvent));
    if (!domain_event) return NULL;
    
    domain_event->event_type = task_created_event_get_type;
    domain_event->occurred_at = task_created_event_get_occurred_at;
    domain_event->event_id = task_created_event_get_id;
    domain_event->destroy = task_created_event_destroy_wrapper;
    domain_event->data = event;
    
    return domain_event;
}

// TaskStartedEvent的DomainEvent接口实现

static const char* task_started_event_get_type(const DomainEvent* event) {
    (void)event;
    return "TaskStarted";
}

static time_t task_started_event_get_occurred_at(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    TaskStartedEvent* started_event = (TaskStartedEvent*)event->data;
    return started_event->started_at;
}

static uint64_t task_started_event_get_id(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    TaskStartedEvent* started_event = (TaskStartedEvent*)event->data;
    return started_event->event_id;
}

static void task_started_event_destroy_wrapper(DomainEvent* event) {
    if (!event || !event->data) return;
    TaskStartedEvent* started_event = (TaskStartedEvent*)event->data;
    task_started_event_destroy(started_event);
    free(event);
}

DomainEvent* task_started_event_to_domain_event(TaskStartedEvent* event) {
    if (!event) return NULL;
    
    DomainEvent* domain_event = (DomainEvent*)malloc(sizeof(DomainEvent));
    if (!domain_event) return NULL;
    
    domain_event->event_type = task_started_event_get_type;
    domain_event->occurred_at = task_started_event_get_occurred_at;
    domain_event->event_id = task_started_event_get_id;
    domain_event->destroy = task_started_event_destroy_wrapper;
    domain_event->data = event;
    
    return domain_event;
}

// TaskCompletedEvent的DomainEvent接口实现

static const char* task_completed_event_get_type(const DomainEvent* event) {
    (void)event;
    return "TaskCompleted";
}

static time_t task_completed_event_get_occurred_at(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    TaskCompletedEvent* completed_event = (TaskCompletedEvent*)event->data;
    return completed_event->completed_at;
}

static uint64_t task_completed_event_get_id(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    TaskCompletedEvent* completed_event = (TaskCompletedEvent*)event->data;
    return completed_event->event_id;
}

static void task_completed_event_destroy_wrapper(DomainEvent* event) {
    if (!event || !event->data) return;
    TaskCompletedEvent* completed_event = (TaskCompletedEvent*)event->data;
    task_completed_event_destroy(completed_event);
    free(event);
}

DomainEvent* task_completed_event_to_domain_event(TaskCompletedEvent* event) {
    if (!event) return NULL;
    
    DomainEvent* domain_event = (DomainEvent*)malloc(sizeof(DomainEvent));
    if (!domain_event) return NULL;
    
    domain_event->event_type = task_completed_event_get_type;
    domain_event->occurred_at = task_completed_event_get_occurred_at;
    domain_event->event_id = task_completed_event_get_id;
    domain_event->destroy = task_completed_event_destroy_wrapper;
    domain_event->data = event;
    
    return domain_event;
}

// TaskFailedEvent的DomainEvent接口实现

static const char* task_failed_event_get_type(const DomainEvent* event) {
    (void)event;
    return "TaskFailed";
}

static time_t task_failed_event_get_occurred_at(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    TaskFailedEvent* failed_event = (TaskFailedEvent*)event->data;
    return failed_event->failed_at;
}

static uint64_t task_failed_event_get_id(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    TaskFailedEvent* failed_event = (TaskFailedEvent*)event->data;
    return failed_event->event_id;
}

static void task_failed_event_destroy_wrapper(DomainEvent* event) {
    if (!event || !event->data) return;
    TaskFailedEvent* failed_event = (TaskFailedEvent*)event->data;
    task_failed_event_destroy(failed_event);
    free(event);
}

DomainEvent* task_failed_event_to_domain_event(TaskFailedEvent* event) {
    if (!event) return NULL;
    
    DomainEvent* domain_event = (DomainEvent*)malloc(sizeof(DomainEvent));
    if (!domain_event) return NULL;
    
    domain_event->event_type = task_failed_event_get_type;
    domain_event->occurred_at = task_failed_event_get_occurred_at;
    domain_event->event_id = task_failed_event_get_id;
    domain_event->destroy = task_failed_event_destroy_wrapper;
    domain_event->data = event;
    
    return domain_event;
}

// TaskCancelledEvent的DomainEvent接口实现

static const char* task_cancelled_event_get_type(const DomainEvent* event) {
    (void)event;
    return "TaskCancelled";
}

static time_t task_cancelled_event_get_occurred_at(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    TaskCancelledEvent* cancelled_event = (TaskCancelledEvent*)event->data;
    return cancelled_event->cancelled_at;
}

static uint64_t task_cancelled_event_get_id(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    TaskCancelledEvent* cancelled_event = (TaskCancelledEvent*)event->data;
    return cancelled_event->event_id;
}

static void task_cancelled_event_destroy_wrapper(DomainEvent* event) {
    if (!event || !event->data) return;
    TaskCancelledEvent* cancelled_event = (TaskCancelledEvent*)event->data;
    task_cancelled_event_destroy(cancelled_event);
    free(event);
}

DomainEvent* task_cancelled_event_to_domain_event(TaskCancelledEvent* event) {
    if (!event) return NULL;
    
    DomainEvent* domain_event = (DomainEvent*)malloc(sizeof(DomainEvent));
    if (!domain_event) return NULL;
    
    domain_event->event_type = task_cancelled_event_get_type;
    domain_event->occurred_at = task_cancelled_event_get_occurred_at;
    domain_event->event_id = task_cancelled_event_get_id;
    domain_event->destroy = task_cancelled_event_destroy_wrapper;
    domain_event->data = event;
    
    return domain_event;
}

// TaskWaitingForFutureEvent的DomainEvent接口实现

static const char* task_waiting_for_future_event_get_type(const DomainEvent* event) {
    (void)event;
    return "TaskWaitingForFuture";
}

static time_t task_waiting_for_future_event_get_occurred_at(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    TaskWaitingForFutureEvent* waiting_event = (TaskWaitingForFutureEvent*)event->data;
    return waiting_event->waiting_at;
}

static uint64_t task_waiting_for_future_event_get_id(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    TaskWaitingForFutureEvent* waiting_event = (TaskWaitingForFutureEvent*)event->data;
    return waiting_event->event_id;
}

static void task_waiting_for_future_event_destroy_wrapper(DomainEvent* event) {
    if (!event || !event->data) return;
    TaskWaitingForFutureEvent* waiting_event = (TaskWaitingForFutureEvent*)event->data;
    task_waiting_for_future_event_destroy(waiting_event);
    free(event);
}

DomainEvent* task_waiting_for_future_event_to_domain_event(TaskWaitingForFutureEvent* event) {
    if (!event) return NULL;
    
    DomainEvent* domain_event = (DomainEvent*)malloc(sizeof(DomainEvent));
    if (!domain_event) return NULL;
    
    domain_event->event_type = task_waiting_for_future_event_get_type;
    domain_event->occurred_at = task_waiting_for_future_event_get_occurred_at;
    domain_event->event_id = task_waiting_for_future_event_get_id;
    domain_event->destroy = task_waiting_for_future_event_destroy_wrapper;
    domain_event->data = event;
    
    return domain_event;
}
