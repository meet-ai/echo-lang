#ifndef TASK_EVENTS_H
#define TASK_EVENTS_H

#include <stdint.h>
#include <time.h>

// 任务已提交事件
typedef struct {
    uint64_t event_id;
    uint64_t task_id;
    char task_name[256];
    time_t occurred_at;
} TaskSubmittedEvent;

// 任务已调度事件
typedef struct {
    uint64_t event_id;
    uint64_t task_id;
    time_t scheduled_at;
    time_t occurred_at;
} TaskScheduledEvent;

// 任务已开始事件
typedef struct {
    uint64_t event_id;
    uint64_t task_id;
    time_t started_at;
    time_t occurred_at;
} TaskStartedEvent;

// 任务已完成事件
typedef struct {
    uint64_t event_id;
    uint64_t task_id;
    int exit_code;
    time_t completed_at;
    time_t occurred_at;
} TaskCompletedEvent;

// 任务已失败事件
typedef struct {
    uint64_t event_id;
    uint64_t task_id;
    char error_message[1024];
    time_t failed_at;
    time_t occurred_at;
} TaskFailedEvent;

// 任务已取消事件
typedef struct {
    uint64_t event_id;
    uint64_t task_id;
    time_t cancelled_at;
    time_t occurred_at;
} TaskCancelledEvent;

// 事件类型枚举
typedef enum {
    EVENT_TYPE_TASK_SUBMITTED,
    EVENT_TYPE_TASK_SCHEDULED,
    EVENT_TYPE_TASK_STARTED,
    EVENT_TYPE_TASK_COMPLETED,
    EVENT_TYPE_TASK_FAILED,
    EVENT_TYPE_TASK_CANCELLED
} TaskEventType;

// 通用事件联合体
typedef union {
    TaskSubmittedEvent submitted;
    TaskScheduledEvent scheduled;
    TaskStartedEvent started;
    TaskCompletedEvent completed;
    TaskFailedEvent failed;
    TaskCancelledEvent cancelled;
} TaskEventData;

// 通用任务事件
typedef struct {
    TaskEventType type;
    TaskEventData data;
} TaskEvent;

// 事件处理函数类型
typedef void (*TaskEventHandler)(const TaskEvent* event, void* context);

// 事件发布器接口
typedef struct TaskEventPublisher {
    void (*publish)(struct TaskEventPublisher* publisher, const TaskEvent* event);
} TaskEventPublisher;

// 创建事件函数
TaskEvent* task_event_create_submitted(uint64_t task_id, const char* task_name);
TaskEvent* task_event_create_scheduled(uint64_t task_id);
TaskEvent* task_event_create_started(uint64_t task_id);
TaskEvent* task_event_create_completed(uint64_t task_id, int exit_code);
TaskEvent* task_event_create_failed(uint64_t task_id, const char* error_message);
TaskEvent* task_event_create_cancelled(uint64_t task_id);

// 销毁事件
void task_event_destroy(TaskEvent* event);

// 获取事件名称
const char* task_event_type_name(TaskEventType type);

#endif // TASK_EVENTS_H
