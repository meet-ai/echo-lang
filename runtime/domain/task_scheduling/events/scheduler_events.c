/**
 * @file scheduler_events.c
 * @brief TaskScheduling 聚合领域事件实现
 */

#include "scheduler_events.h"
#include <stdlib.h>
#include <time.h>
#include <stdint.h>
#include <string.h>

// 生成事件ID
static uint64_t generate_event_id(void) {
    static uint64_t next_event_id = 1;
    return next_event_id++;
}

// ==================== 事件创建函数 ====================

TaskScheduledEvent* task_scheduled_event_create(
    TaskID task_id,
    uint32_t scheduler_id,
    uint32_t processor_id
) {
    TaskScheduledEvent* event = (TaskScheduledEvent*)malloc(sizeof(TaskScheduledEvent));
    if (!event) {
        return NULL;
    }
    
    event->event_id = generate_event_id();
    event->task_id = task_id;
    event->scheduler_id = scheduler_id;
    event->processor_id = processor_id;
    event->occurred_at = time(NULL);
    
    return event;
}

TaskCompletedEvent* scheduler_task_completed_event_create(
    TaskID task_id,
    uint32_t scheduler_id,
    uint32_t processor_id
) {
    TaskCompletedEvent* event = (TaskCompletedEvent*)malloc(sizeof(TaskCompletedEvent));
    if (!event) {
        return NULL;
    }
    
    event->event_id = generate_event_id();
    event->task_id = task_id;
    event->scheduler_id = scheduler_id;
    event->processor_id = processor_id;
    event->occurred_at = time(NULL);
    
    return event;
}

// ==================== 事件销毁函数 ====================

void task_scheduled_event_destroy(TaskScheduledEvent* event) {
    if (event) {
        free(event);
    }
}

void scheduler_task_completed_event_destroy(TaskCompletedEvent* event) {
    if (event) {
        free(event);
    }
}

// ==================== DomainEvent接口实现 ====================

// TaskScheduledEvent的DomainEvent接口实现

static const char* task_scheduled_event_get_type(const DomainEvent* event) {
    (void)event;  // 未使用
    return "TaskScheduled";
}

static time_t task_scheduled_event_get_occurred_at(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    TaskScheduledEvent* scheduled_event = (TaskScheduledEvent*)event->data;
    return scheduled_event->occurred_at;
}

static uint64_t task_scheduled_event_get_id(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    TaskScheduledEvent* scheduled_event = (TaskScheduledEvent*)event->data;
    return scheduled_event->event_id;
}

static void task_scheduled_event_destroy_wrapper(DomainEvent* event) {
    if (!event || !event->data) return;
    TaskScheduledEvent* scheduled_event = (TaskScheduledEvent*)event->data;
    task_scheduled_event_destroy(scheduled_event);
    free(event);
}

DomainEvent* task_scheduled_event_to_domain_event(TaskScheduledEvent* event) {
    if (!event) return NULL;
    
    DomainEvent* domain_event = (DomainEvent*)malloc(sizeof(DomainEvent));
    if (!domain_event) return NULL;
    
    domain_event->event_type = task_scheduled_event_get_type;
    domain_event->occurred_at = task_scheduled_event_get_occurred_at;
    domain_event->event_id = task_scheduled_event_get_id;
    domain_event->destroy = task_scheduled_event_destroy_wrapper;
    domain_event->data = event;
    
    return domain_event;
}

// TaskCompletedEvent的DomainEvent接口实现

static const char* task_completed_event_get_type(const DomainEvent* event) {
    (void)event;  // 未使用
    return "TaskCompleted";
}

static time_t task_completed_event_get_occurred_at(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    TaskCompletedEvent* completed_event = (TaskCompletedEvent*)event->data;
    return completed_event->occurred_at;
}

static uint64_t task_completed_event_get_id(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    TaskCompletedEvent* completed_event = (TaskCompletedEvent*)event->data;
    return completed_event->event_id;
}

static void scheduler_task_completed_event_destroy_wrapper(DomainEvent* event) {
    if (!event || !event->data) return;
    TaskCompletedEvent* completed_event = (TaskCompletedEvent*)event->data;
    scheduler_task_completed_event_destroy(completed_event);
    free(event);
}

DomainEvent* scheduler_task_completed_event_to_domain_event(TaskCompletedEvent* event) {
    if (!event) return NULL;
    
    DomainEvent* domain_event = (DomainEvent*)malloc(sizeof(DomainEvent));
    if (!domain_event) return NULL;
    
    domain_event->event_type = task_completed_event_get_type;
    domain_event->occurred_at = task_completed_event_get_occurred_at;
    domain_event->event_id = task_completed_event_get_id;
    domain_event->destroy = scheduler_task_completed_event_destroy_wrapper;
    domain_event->data = event;
    
    return domain_event;
}
