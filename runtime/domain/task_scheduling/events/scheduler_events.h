/**
 * @file scheduler_events.h
 * @brief TaskScheduling 聚合领域事件定义
 *
 * 定义Scheduler聚合发布的所有领域事件：
 * - TaskScheduled: 任务被调度
 * - TaskCompleted: 任务完成
 */

#ifndef TASK_SCHEDULING_EVENTS_SCHEDULER_EVENTS_H
#define TASK_SCHEDULING_EVENTS_SCHEDULER_EVENTS_H

#include "../../task_execution/aggregate/task.h"  // TaskID定义
#include "../../shared/events/bus.h"  // DomainEvent接口
#include <stdint.h>
#include <time.h>

// ==================== 领域事件定义 ====================

/**
 * @brief TaskScheduled 事件
 * 
 * 触发时机：任务被成功添加到调度器
 * 携带数据：任务ID、调度器ID、目标处理器ID（如果分配到本地队列）
 */
typedef struct TaskScheduledEvent {
    uint64_t event_id;
    TaskID task_id;
    uint32_t scheduler_id;
    uint32_t processor_id;  // 如果为UINT32_MAX，表示添加到全局队列
    time_t occurred_at;
} TaskScheduledEvent;

/**
 * @brief TaskCompleted 事件
 * 
 * 触发时机：任务执行完成
 * 携带数据：任务ID、调度器ID、执行任务的处理器ID
 */
typedef struct TaskCompletedEvent {
    uint64_t event_id;
    TaskID task_id;
    uint32_t scheduler_id;
    uint32_t processor_id;
    time_t occurred_at;
} TaskCompletedEvent;

// ==================== 事件创建函数 ====================

/**
 * @brief 创建TaskScheduled事件
 */
TaskScheduledEvent* task_scheduled_event_create(
    TaskID task_id,
    uint32_t scheduler_id,
    uint32_t processor_id
);

/**
 * @brief 创建TaskCompleted事件
 */
TaskCompletedEvent* task_completed_event_create(
    TaskID task_id,
    uint32_t scheduler_id,
    uint32_t processor_id
);

// ==================== 事件销毁函数 ====================

void task_scheduled_event_destroy(TaskScheduledEvent* event);
void scheduler_task_completed_event_destroy(TaskCompletedEvent* event);

// ==================== DomainEvent接口实现 ====================

/**
 * @brief 将TaskScheduledEvent转换为DomainEvent
 * 
 * @param event TaskScheduledEvent实例
 * @return DomainEvent接口指针
 */
DomainEvent* task_scheduled_event_to_domain_event(TaskScheduledEvent* event);

/**
 * @brief 将TaskCompletedEvent转换为DomainEvent
 * 
 * @param event TaskCompletedEvent实例
 * @return DomainEvent接口指针
 */
DomainEvent* scheduler_task_completed_event_to_domain_event(TaskCompletedEvent* event);

#endif // TASK_SCHEDULING_EVENTS_SCHEDULER_EVENTS_H
