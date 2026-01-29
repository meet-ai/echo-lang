/**
 * @file future_events.c
 * @brief AsyncComputation 聚合领域事件实现
 */

#include "future_events.h"
#include "../../shared/events/bus.h"  // 使用新的DomainEvent接口
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <stdio.h>
#include <stdint.h>

// 事件ID生成器
static uint64_t generate_event_id(void) {
    static uint64_t next_event_id = 1;
    return next_event_id++;
}

// ============================================================================
// FutureResolved 事件实现
// ============================================================================

static const char* future_resolved_event_get_type(const DomainEvent* event) {
    (void)event;  // 未使用
    return EVENT_TYPE_FUTURE_RESOLVED;
}

static time_t future_resolved_event_get_occurred_at(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    FutureResolvedEvent* resolved_event = (FutureResolvedEvent*)event->data;
    return resolved_event->occurred_at;
}

static uint64_t future_resolved_event_get_id(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    FutureResolvedEvent* resolved_event = (FutureResolvedEvent*)event->data;
    return resolved_event->event_id;
}

static void future_resolved_event_destroy_wrapper(DomainEvent* event) {
    if (!event || !event->data) return;
    FutureResolvedEvent* resolved_event = (FutureResolvedEvent*)event->data;
    future_resolved_event_destroy(resolved_event);
    free(event);
}

FutureResolvedEvent* future_resolved_event_new(uint64_t future_id, void* result) {
    FutureResolvedEvent* event = (FutureResolvedEvent*)calloc(1, sizeof(FutureResolvedEvent));
    if (!event) {
        fprintf(stderr, "ERROR: Failed to allocate memory for FutureResolvedEvent\n");
        return NULL;
    }

    // 设置事件数据
    event->event_id = generate_event_id();
    event->future_id = future_id;
    event->result = result;
    event->occurred_at = time(NULL);

    return event;
}

void future_resolved_event_destroy(FutureResolvedEvent* event) {
    if (!event) return;

    // 注意：result 的所有权由调用者管理，这里不释放
    free(event);
}

// ============================================================================
// FutureRejected 事件实现
// ============================================================================

static const char* future_rejected_event_get_type(const DomainEvent* event) {
    (void)event;  // 未使用
    return EVENT_TYPE_FUTURE_REJECTED;
}

static time_t future_rejected_event_get_occurred_at(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    FutureRejectedEvent* rejected_event = (FutureRejectedEvent*)event->data;
    return rejected_event->occurred_at;
}

static uint64_t future_rejected_event_get_id(const DomainEvent* event) {
    if (!event || !event->data) return 0;
    FutureRejectedEvent* rejected_event = (FutureRejectedEvent*)event->data;
    return rejected_event->event_id;
}

static void future_rejected_event_destroy_wrapper(DomainEvent* event) {
    if (!event || !event->data) return;
    FutureRejectedEvent* rejected_event = (FutureRejectedEvent*)event->data;
    future_rejected_event_destroy(rejected_event);
    free(event);
}

FutureRejectedEvent* future_rejected_event_new(uint64_t future_id, void* error) {
    FutureRejectedEvent* event = (FutureRejectedEvent*)calloc(1, sizeof(FutureRejectedEvent));
    if (!event) {
        fprintf(stderr, "ERROR: Failed to allocate memory for FutureRejectedEvent\n");
        return NULL;
    }

    // 设置事件数据
    event->event_id = generate_event_id();
    event->future_id = future_id;
    event->error = error;
    event->occurred_at = time(NULL);

    return event;
}

void future_rejected_event_destroy(FutureRejectedEvent* event) {
    if (!event) return;

    // 注意：error 的所有权由调用者管理，这里不释放
    free(event);
}

// ============================================================================
// DomainEvent接口实现
// ============================================================================

DomainEvent* future_resolved_event_to_domain_event(FutureResolvedEvent* event) {
    if (!event) return NULL;

    DomainEvent* domain_event = (DomainEvent*)calloc(1, sizeof(DomainEvent));
    if (!domain_event) {
        fprintf(stderr, "ERROR: Failed to allocate memory for DomainEvent wrapper\n");
        return NULL;
    }

    domain_event->event_type = future_resolved_event_get_type;
    domain_event->occurred_at = future_resolved_event_get_occurred_at;
    domain_event->event_id = future_resolved_event_get_id;
    domain_event->destroy = future_resolved_event_destroy_wrapper;
    domain_event->data = event;

    return domain_event;
}

DomainEvent* future_rejected_event_to_domain_event(FutureRejectedEvent* event) {
    if (!event) return NULL;

    DomainEvent* domain_event = (DomainEvent*)calloc(1, sizeof(DomainEvent));
    if (!domain_event) {
        fprintf(stderr, "ERROR: Failed to allocate memory for DomainEvent wrapper\n");
        return NULL;
    }

    domain_event->event_type = future_rejected_event_get_type;
    domain_event->occurred_at = future_rejected_event_get_occurred_at;
    domain_event->event_id = future_rejected_event_get_id;
    domain_event->destroy = future_rejected_event_destroy_wrapper;
    domain_event->data = event;

    return domain_event;
}
