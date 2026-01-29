/**
 * @file task_events.h
 * @brief TaskExecution 聚合领域事件定义
 *
 * 领域事件用于解耦聚合之间的通信，支持异步处理。
 * 
 * 使用场景：
 * - Task创建后发布TaskCreated事件，通知调度器
 * - Task完成后发布TaskCompleted事件，通知等待者
 * - Task失败后发布TaskFailed事件，触发错误处理
 * 
 * 示例：
 * ```c
 * // 在聚合根方法中发布事件
 * TaskCreatedEvent* event = task_created_event_create(task->id, task->priority);
 * task_add_domain_event(task, (DomainEvent*)event);
 * 
 * // 应用服务获取并发布事件
 * DomainEvent** events;
 * size_t count;
 * task_get_domain_events(task, &events, &count);
 * for (size_t i = 0; i < count; i++) {
 *     event_bus_publish(event_bus, events[i]);
 * }
 * ```
 */

#ifndef TASK_EXECUTION_EVENTS_TASK_EVENTS_H
#define TASK_EXECUTION_EVENTS_TASK_EVENTS_H

#include "../../shared/events/bus.h"  // 使用新的DomainEvent接口
#include <stdint.h>
#include <time.h>

// 注意：旧的DomainEvent定义已移除，现在使用shared/events/bus.h中的DomainEvent接口

// ============================================================================
// TaskCreated 事件
// ============================================================================

/**
 * @brief TaskCreated 事件
 * 
 * 触发时机：任务创建成功
 * 订阅者：调度器（将任务加入队列）
 */
typedef struct TaskCreatedEvent {
    uint64_t event_id;         // 事件ID
    uint64_t task_id;          // 任务ID
    int priority;              // 任务优先级
    time_t created_at;         // 创建时间
} TaskCreatedEvent;

/**
 * @brief 创建TaskCreated事件
 */
TaskCreatedEvent* task_created_event_create(uint64_t task_id, int priority);

/**
 * @brief 销毁TaskCreated事件
 */
void task_created_event_destroy(TaskCreatedEvent* event);

// ============================================================================
// TaskStarted 事件
// ============================================================================

/**
 * @brief TaskStarted 事件
 * 
 * 触发时机：任务开始执行
 * 订阅者：监控服务（记录任务执行开始）
 */
typedef struct TaskStartedEvent {
    uint64_t event_id;         // 事件ID
    uint64_t task_id;          // 任务ID
    time_t started_at;         // 开始时间
} TaskStartedEvent;

TaskStartedEvent* task_started_event_create(uint64_t task_id);
void task_started_event_destroy(TaskStartedEvent* event);

// ============================================================================
// TaskCompleted 事件
// ============================================================================

/**
 * @brief TaskCompleted 事件
 * 
 * 触发时机：任务执行完成
 * 订阅者：调度器（清理任务）、监控服务（记录完成）
 */
typedef struct TaskCompletedEvent {
    uint64_t event_id;         // 事件ID
    uint64_t task_id;          // 任务ID
    void* result;              // 执行结果
    time_t completed_at;       // 完成时间
} TaskCompletedEvent;

TaskCompletedEvent* task_completed_event_create(uint64_t task_id, void* result);
void task_completed_event_destroy(TaskCompletedEvent* event);

// ============================================================================
// TaskFailed 事件
// ============================================================================

/**
 * @brief TaskFailed 事件
 * 
 * 触发时机：任务执行失败
 * 订阅者：错误处理服务、监控服务
 */
typedef struct TaskFailedEvent {
    uint64_t event_id;         // 事件ID
    uint64_t task_id;          // 任务ID
    const char* error_message; // 错误信息
    time_t failed_at;          // 失败时间
} TaskFailedEvent;

TaskFailedEvent* task_failed_event_create(uint64_t task_id, const char* error_message);
void task_failed_event_destroy(TaskFailedEvent* event);

// ============================================================================
// TaskCancelled 事件
// ============================================================================

/**
 * @brief TaskCancelled 事件
 * 
 * 触发时机：任务被取消
 * 订阅者：调度器（清理任务）、资源管理器（释放资源）
 */
typedef struct TaskCancelledEvent {
    uint64_t event_id;         // 事件ID
    uint64_t task_id;          // 任务ID
    const char* reason;        // 取消原因
    time_t cancelled_at;       // 取消时间
} TaskCancelledEvent;

TaskCancelledEvent* task_cancelled_event_create(uint64_t task_id, const char* reason);
void task_cancelled_event_destroy(TaskCancelledEvent* event);

// ============================================================================
// TaskWaitingForFuture 事件
// ============================================================================

/**
 * @brief TaskWaitingForFuture 事件
 * 
 * 触发时机：任务开始等待Future完成
 * 订阅者：Future聚合（注册等待者）
 */
typedef struct TaskWaitingForFutureEvent {
    uint64_t event_id;         // 事件ID
    uint64_t task_id;          // 任务ID
    uint64_t future_id;        // 等待的Future ID
    time_t waiting_at;         // 开始等待时间
} TaskWaitingForFutureEvent;

TaskWaitingForFutureEvent* task_waiting_for_future_event_create(uint64_t task_id, uint64_t future_id);
void task_waiting_for_future_event_destroy(TaskWaitingForFutureEvent* event);

// ==================== DomainEvent接口实现 ====================

/**
 * @brief 将TaskCreatedEvent转换为DomainEvent
 */
DomainEvent* task_created_event_to_domain_event(TaskCreatedEvent* event);

/**
 * @brief 将TaskStartedEvent转换为DomainEvent
 */
DomainEvent* task_started_event_to_domain_event(TaskStartedEvent* event);

/**
 * @brief 将TaskCompletedEvent转换为DomainEvent
 */
DomainEvent* task_completed_event_to_domain_event(TaskCompletedEvent* event);

/**
 * @brief 将TaskFailedEvent转换为DomainEvent
 */
DomainEvent* task_failed_event_to_domain_event(TaskFailedEvent* event);

/**
 * @brief 将TaskCancelledEvent转换为DomainEvent
 */
DomainEvent* task_cancelled_event_to_domain_event(TaskCancelledEvent* event);

/**
 * @brief 将TaskWaitingForFutureEvent转换为DomainEvent
 */
DomainEvent* task_waiting_for_future_event_to_domain_event(TaskWaitingForFutureEvent* event);

#endif // TASK_EXECUTION_EVENTS_TASK_EVENTS_H
