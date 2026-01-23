#ifndef TASK_SCHEDULER_H
#define TASK_SCHEDULER_H

#include "../entity/task.h"
#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct TaskQueue;
struct TaskScheduler;

// 任务调度器领域服务 - 负责任务的调度和管理
typedef struct TaskScheduler {
    struct TaskQueue* ready_queue;    // 就绪队列
    struct TaskQueue* waiting_queue;  // 等待队列
    uint32_t max_concurrent_tasks;    // 最大并发任务数
    uint32_t running_task_count;      // 当前运行任务数
} TaskScheduler;

// 任务队列接口
typedef struct TaskQueue {
    void (*enqueue)(struct TaskQueue* queue, Task* task);
    Task* (*dequeue)(struct TaskQueue* queue);
    size_t (*size)(struct TaskQueue* queue);
    bool (*is_empty)(struct TaskQueue* queue);
} TaskQueue;

// 调度策略
typedef enum {
    SCHEDULER_POLICY_FCFS,      // 先来先服务
    SCHEDULER_POLICY_PRIORITY,  // 优先级调度
    SCHEDULER_POLICY_SJF,       // 最短作业优先
    SCHEDULER_POLICY_ROUND_ROBIN // 时间片轮转
} SchedulerPolicy;

// 任务调度器方法
TaskScheduler* task_scheduler_create(uint32_t max_concurrent);
void task_scheduler_destroy(TaskScheduler* scheduler);

// 任务管理
bool task_scheduler_submit(TaskScheduler* scheduler, Task* task);
bool task_scheduler_cancel(TaskScheduler* scheduler, uint64_t task_id);
Task* task_scheduler_get_next_task(TaskScheduler* scheduler);
Task* task_scheduler_wait_for_task(TaskScheduler* scheduler, uint32_t timeout_ms);

// 调度控制
void task_scheduler_start(TaskScheduler* scheduler);
void task_scheduler_stop(TaskScheduler* scheduler);
void task_scheduler_pause(TaskScheduler* scheduler);
void task_scheduler_resume(TaskScheduler* scheduler);

// 状态查询
uint32_t task_scheduler_get_running_count(const TaskScheduler* scheduler);
uint32_t task_scheduler_get_queued_count(const TaskScheduler* scheduler);
bool task_scheduler_is_running(const TaskScheduler* scheduler);

// 扩展状态查询
typedef enum {
    SCHEDULER_STATE_STOPPED,
    SCHEDULER_STATE_RUNNING,
    SCHEDULER_STATE_PAUSED
} SchedulerState;

SchedulerState task_scheduler_get_state(const TaskScheduler* scheduler);
bool task_scheduler_get_statistics(const TaskScheduler* scheduler,
                                  uint64_t* total_scheduled,
                                  uint64_t* total_completed,
                                  uint32_t* current_running,
                                  uint32_t* queued_count);

// 任务完成通知
void task_scheduler_on_task_completed(TaskScheduler* scheduler, uint64_t task_id, int exit_code);
void task_scheduler_on_task_failed(TaskScheduler* scheduler, uint64_t task_id, const char* error);

#endif // TASK_SCHEDULER_H
