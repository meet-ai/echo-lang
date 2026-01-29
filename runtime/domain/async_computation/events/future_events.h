/**
 * @file future_events.h
 * @brief AsyncComputation 聚合领域事件定义
 *
 * 领域事件用于解耦跨聚合通信，实现最终一致性。
 */

#ifndef ASYNC_COMPUTATION_EVENTS_FUTURE_EVENTS_H
#define ASYNC_COMPUTATION_EVENTS_FUTURE_EVENTS_H

#include "../../shared/events/bus.h"  // 使用新的DomainEvent接口
#include <stdint.h>
#include <time.h>

// 注意：旧的DomainEvent定义已移除，现在使用shared/events/bus.h中的DomainEvent接口

// ============================================================================
// 领域事件类型常量
// ============================================================================

#define EVENT_TYPE_FUTURE_RESOLVED "future.resolved"
#define EVENT_TYPE_FUTURE_REJECTED "future.rejected"

// ============================================================================
// FutureResolved 事件
// ============================================================================

/**
 * @brief FutureResolved 领域事件
 * 
 * 触发时机：Future被成功解决时
 * 订阅者：TaskExecution聚合（唤醒等待的Task）
 */
typedef struct FutureResolvedEvent {
    uint64_t event_id;         // 事件ID
    uint64_t future_id;         // Future ID
    void* result;               // 结果值
    time_t occurred_at;         // 事件发生时间
} FutureResolvedEvent;

/**
 * @brief 创建FutureResolved事件
 */
FutureResolvedEvent* future_resolved_event_new(uint64_t future_id, void* result);

/**
 * @brief 销毁FutureResolved事件
 */
void future_resolved_event_destroy(FutureResolvedEvent* event);

// ============================================================================
// FutureRejected 事件
// ============================================================================

/**
 * @brief FutureRejected 领域事件
 * 
 * 触发时机：Future被拒绝时
 * 订阅者：TaskExecution聚合（唤醒等待的Task，传递错误信息）
 */
typedef struct FutureRejectedEvent {
    uint64_t event_id;         // 事件ID
    uint64_t future_id;         // Future ID
    void* error;                // 错误信息
    time_t occurred_at;         // 事件发生时间
} FutureRejectedEvent;

/**
 * @brief 创建FutureRejected事件
 */
FutureRejectedEvent* future_rejected_event_new(uint64_t future_id, void* error);

/**
 * @brief 销毁FutureRejected事件
 */
void future_rejected_event_destroy(FutureRejectedEvent* event);

// ==================== DomainEvent接口实现 ====================

/**
 * @brief 将FutureResolvedEvent转换为DomainEvent
 */
DomainEvent* future_resolved_event_to_domain_event(FutureResolvedEvent* event);

/**
 * @brief 将FutureRejectedEvent转换为DomainEvent
 */
DomainEvent* future_rejected_event_to_domain_event(FutureRejectedEvent* event);

#endif // ASYNC_COMPUTATION_EVENTS_FUTURE_EVENTS_H
