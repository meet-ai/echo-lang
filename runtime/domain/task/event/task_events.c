#include "task_events.h"
#include <stdlib.h>
#include <string.h>
#include <time.h>

// 静态事件ID生成器
static uint64_t next_event_id = 1;

// 创建任务已提交事件
TaskEvent* task_event_create_submitted(uint64_t task_id, const char* task_name) {
    TaskEvent* event = (TaskEvent*)malloc(sizeof(TaskEvent));
    if (!event) return NULL;

    event->type = EVENT_TYPE_TASK_SUBMITTED;
    event->data.submitted.event_id = next_event_id++;
    event->data.submitted.task_id = task_id;
    strncpy(event->data.submitted.task_name, task_name, sizeof(event->data.submitted.task_name) - 1);
    event->data.submitted.task_name[sizeof(event->data.submitted.task_name) - 1] = '\0';
    event->data.submitted.occurred_at = time(NULL);

    return event;
}

// 创建任务已调度事件
TaskEvent* task_event_create_scheduled(uint64_t task_id) {
    TaskEvent* event = (TaskEvent*)malloc(sizeof(TaskEvent));
    if (!event) return NULL;

    event->type = EVENT_TYPE_TASK_SCHEDULED;
    event->data.scheduled.event_id = next_event_id++;
    event->data.scheduled.task_id = task_id;
    event->data.scheduled.scheduled_at = time(NULL);
    event->data.scheduled.occurred_at = time(NULL);

    return event;
}

// 创建任务已开始事件
TaskEvent* task_event_create_started(uint64_t task_id) {
    TaskEvent* event = (TaskEvent*)malloc(sizeof(TaskEvent));
    if (!event) return NULL;

    event->type = EVENT_TYPE_TASK_STARTED;
    event->data.started.event_id = next_event_id++;
    event->data.started.task_id = task_id;
    event->data.started.started_at = time(NULL);
    event->data.started.occurred_at = time(NULL);

    return event;
}

// 创建任务已完成事件
TaskEvent* task_event_create_completed(uint64_t task_id, int exit_code) {
    TaskEvent* event = (TaskEvent*)malloc(sizeof(TaskEvent));
    if (!event) return NULL;

    event->type = EVENT_TYPE_TASK_COMPLETED;
    event->data.completed.event_id = next_event_id++;
    event->data.completed.task_id = task_id;
    event->data.completed.exit_code = exit_code;
    event->data.completed.completed_at = time(NULL);
    event->data.completed.occurred_at = time(NULL);

    return event;
}

// 创建任务已失败事件
TaskEvent* task_event_create_failed(uint64_t task_id, const char* error_message) {
    TaskEvent* event = (TaskEvent*)malloc(sizeof(TaskEvent));
    if (!event) return NULL;

    event->type = EVENT_TYPE_TASK_FAILED;
    event->data.failed.event_id = next_event_id++;
    event->data.failed.task_id = task_id;
    if (error_message) {
        strncpy(event->data.failed.error_message, error_message, sizeof(event->data.failed.error_message) - 1);
        event->data.failed.error_message[sizeof(event->data.failed.error_message) - 1] = '\0';
    } else {
        event->data.failed.error_message[0] = '\0';
    }
    event->data.failed.failed_at = time(NULL);
    event->data.failed.occurred_at = time(NULL);

    return event;
}

// 创建任务已取消事件
TaskEvent* task_event_create_cancelled(uint64_t task_id) {
    TaskEvent* event = (TaskEvent*)malloc(sizeof(TaskEvent));
    if (!event) return NULL;

    event->type = EVENT_TYPE_TASK_CANCELLED;
    event->data.cancelled.event_id = next_event_id++;
    event->data.cancelled.task_id = task_id;
    event->data.cancelled.cancelled_at = time(NULL);
    event->data.cancelled.occurred_at = time(NULL);

    return event;
}

// 销毁事件
void task_event_destroy(TaskEvent* event) {
    if (event) {
        free(event);
    }
}

// 获取事件类型名称
const char* task_event_type_name(TaskEventType type) {
    switch (type) {
        case EVENT_TYPE_TASK_SUBMITTED: return "TaskSubmitted";
        case EVENT_TYPE_TASK_SCHEDULED: return "TaskScheduled";
        case EVENT_TYPE_TASK_STARTED: return "TaskStarted";
        case EVENT_TYPE_TASK_COMPLETED: return "TaskCompleted";
        case EVENT_TYPE_TASK_FAILED: return "TaskFailed";
        case EVENT_TYPE_TASK_CANCELLED: return "TaskCancelled";
        default: return "Unknown";
    }
}
