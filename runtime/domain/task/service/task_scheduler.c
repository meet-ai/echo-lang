#include "task_scheduler.h"
#include "../../shared/types/common_types.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <pthread.h>
#include <unistd.h>
#include <errno.h>

// 调度器状态

// 扩展的任务调度器结构体
typedef struct {
    TaskScheduler base;              // 基础调度器
    SchedulerState state;            // 调度器状态
    SchedulerPolicy policy;          // 调度策略
    pthread_mutex_t mutex;           // 保护并发访问
    pthread_cond_t condition;        // 条件变量
    hash_table_t task_map;           // 任务映射表（用于快速查找）
    uint64_t next_task_id;           // 下一个任务ID
    time_t created_at;               // 创建时间
    uint64_t total_tasks_scheduled;  // 总调度任务数
    uint64_t total_tasks_completed;  // 总完成任务数
} ExtendedTaskScheduler;

// 内部使用的任务队列实现（支持优先级的有序队列）
typedef struct PriorityTaskNode {
    Task* task;
    struct PriorityTaskNode* next;
} PriorityTaskNode;

typedef struct PriorityTaskQueue {
    TaskQueue base;                  // 继承基础接口
    PriorityTaskNode* head;          // 队列头（优先级最高的）
    size_t count;                    // 队列大小
    pthread_mutex_t mutex;           // 并发保护
    SchedulerPolicy policy;          // 调度策略
} PriorityTaskQueue;

// 优先级队列入队（保持优先级顺序）
static void priority_queue_enqueue(TaskQueue* base_queue, Task* task) {
    PriorityTaskQueue* queue = (PriorityTaskQueue*)base_queue;

    PriorityTaskNode* new_node = (PriorityTaskNode*)malloc(sizeof(PriorityTaskNode));
    if (!new_node) return;

    new_node->task = task;
    new_node->next = NULL;

    pthread_mutex_lock(&queue->mutex);

    // 空队列直接插入
    if (!queue->head) {
        queue->head = new_node;
        queue->count++;
        pthread_mutex_unlock(&queue->mutex);
        return;
    }

    // 找到插入位置（保持优先级顺序）
    PriorityTaskNode* current = queue->head;
    PriorityTaskNode* prev = NULL;

    while (current) {
        bool should_insert_here = false;

        switch (queue->policy) {
            case SCHEDULER_POLICY_PRIORITY:
                // 优先级高的排前面（数值大的优先级高）
                should_insert_here = task->priority > current->task->priority;
                break;
            case SCHEDULER_POLICY_FCFS:
                // 先来先服务：新任务排在最后
                should_insert_here = false;
                break;
            case SCHEDULER_POLICY_SJF:
                // 最短作业优先：这里简化为根据栈大小排序
                should_insert_here = task->stack_size < current->task->stack_size;
                break;
            default:
                should_insert_here = false;
                break;
        }

        if (should_insert_here) {
            break;
        }

        prev = current;
        current = current->next;
    }

    // 插入节点
    if (prev) {
        // 插入到prev和current之间
        new_node->next = current;
        prev->next = new_node;
    } else {
        // 插入到队列头部
        new_node->next = queue->head;
        queue->head = new_node;
    }

    queue->count++;
    pthread_mutex_unlock(&queue->mutex);
}

// 优先级队列出队（从头部取出）
static Task* priority_queue_dequeue(TaskQueue* base_queue) {
    PriorityTaskQueue* queue = (PriorityTaskQueue*)base_queue;

    pthread_mutex_lock(&queue->mutex);
    if (!queue->head) {
        pthread_mutex_unlock(&queue->mutex);
        return NULL;
    }

    PriorityTaskNode* node = queue->head;
    Task* task = node->task;
    queue->head = node->next;
    queue->count--;

    pthread_mutex_unlock(&queue->mutex);

    free(node);
    return task;
}

static size_t priority_queue_size(TaskQueue* base_queue) {
    PriorityTaskQueue* queue = (PriorityTaskQueue*)base_queue;
    pthread_mutex_lock(&queue->mutex);
    size_t count = queue->count;
    pthread_mutex_unlock(&queue->mutex);
    return count;
}

static bool priority_queue_is_empty(TaskQueue* base_queue) {
    return priority_queue_size(base_queue) == 0;
}

// 创建优先级任务队列
static PriorityTaskQueue* create_priority_task_queue(SchedulerPolicy policy) {
    PriorityTaskQueue* queue = (PriorityTaskQueue*)malloc(sizeof(PriorityTaskQueue));
    if (!queue) return NULL;

    queue->head = NULL;
    queue->count = 0;
    queue->policy = policy;
    pthread_mutex_init(&queue->mutex, NULL);

    // 设置接口
    queue->base.enqueue = priority_queue_enqueue;
    queue->base.dequeue = priority_queue_dequeue;
    queue->base.size = priority_queue_size;
    queue->base.is_empty = priority_queue_is_empty;

    return queue;
}

// 销毁优先级任务队列
static void destroy_priority_task_queue(PriorityTaskQueue* queue) {
    if (!queue) return;

    // 清空队列
    PriorityTaskNode* node = queue->head;
    while (node) {
        PriorityTaskNode* next = node->next;
        free(node);
        node = next;
    }

    pthread_mutex_destroy(&queue->mutex);
    free(queue);
}

// 从队列中查找并移除特定任务
static Task* remove_task_from_queue(PriorityTaskQueue* queue, uint64_t task_id) {
    pthread_mutex_lock(&queue->mutex);

    PriorityTaskNode* current = queue->head;
    PriorityTaskNode* prev = NULL;

    while (current) {
        if (current->task->id == task_id) {
            // 找到任务，从队列中移除
            Task* task = current->task;

            if (prev) {
                prev->next = current->next;
            } else {
                queue->head = current->next;
            }

            queue->count--;
            free(current);

            pthread_mutex_unlock(&queue->mutex);
            return task;
        }

        prev = current;
        current = current->next;
    }

    pthread_mutex_unlock(&queue->mutex);
    return NULL;
}

// 创建任务调度器
TaskScheduler* task_scheduler_create(uint32_t max_concurrent) {
    ExtendedTaskScheduler* scheduler = (ExtendedTaskScheduler*)malloc(sizeof(ExtendedTaskScheduler));
    if (!scheduler) return NULL;

    // 初始化基础字段
    scheduler->base.max_concurrent_tasks = max_concurrent > 0 ? max_concurrent : 10;
    scheduler->base.running_task_count = 0;

    // 设置默认策略
    scheduler->policy = SCHEDULER_POLICY_PRIORITY;
    scheduler->state = SCHEDULER_STATE_STOPPED;

    // 初始化队列
    scheduler->base.ready_queue = (TaskQueue*)create_priority_task_queue(scheduler->policy);
    scheduler->base.waiting_queue = (TaskQueue*)create_priority_task_queue(SCHEDULER_POLICY_FCFS);

    if (!scheduler->base.ready_queue || !scheduler->base.waiting_queue) {
        if (scheduler->base.ready_queue) destroy_priority_task_queue((PriorityTaskQueue*)scheduler->base.ready_queue);
        if (scheduler->base.waiting_queue) destroy_priority_task_queue((PriorityTaskQueue*)scheduler->base.waiting_queue);
        free(scheduler);
        return NULL;
    }

    // 初始化同步原语
    pthread_mutex_init(&scheduler->mutex, NULL);
    pthread_cond_init(&scheduler->condition, NULL);

    // 初始化任务映射表
    // 这里简化实现，实际应该使用哈希表

    // 初始化统计信息
    scheduler->next_task_id = 1;
    scheduler->created_at = time(NULL);
    scheduler->total_tasks_scheduled = 0;
    scheduler->total_tasks_completed = 0;

    return (TaskScheduler*)scheduler;
}

// 销毁任务调度器
void task_scheduler_destroy(TaskScheduler* scheduler) {
    if (!scheduler) return;

    ExtendedTaskScheduler* ext_scheduler = (ExtendedTaskScheduler*)scheduler;

    // 停止调度器
    ext_scheduler->state = SCHEDULER_STATE_STOPPED;

    // 销毁队列
    if (scheduler->ready_queue) {
        destroy_priority_task_queue((PriorityTaskQueue*)scheduler->ready_queue);
    }
    if (scheduler->waiting_queue) {
        destroy_priority_task_queue((PriorityTaskQueue*)scheduler->waiting_queue);
    }

    // 销毁同步原语
    pthread_mutex_destroy(&ext_scheduler->mutex);
    pthread_cond_destroy(&ext_scheduler->condition);

    free(scheduler);
}

// 提交任务
bool task_scheduler_submit(TaskScheduler* scheduler, Task* task) {
    if (!scheduler || !task || !task_can_schedule(task)) {
        return false;
    }

    ExtendedTaskScheduler* ext_scheduler = (ExtendedTaskScheduler*)scheduler;

    pthread_mutex_lock(&ext_scheduler->mutex);

    // 检查调度器状态
    if (ext_scheduler->state != SCHEDULER_STATE_RUNNING) {
        pthread_mutex_unlock(&ext_scheduler->mutex);
        return false;
    }

    // 生成唯一任务ID
    task->id = ext_scheduler->next_task_id++;

    // 设置为就绪状态
    task_update_status(task, TASK_STATUS_READY);

    // 添加到就绪队列
    scheduler->ready_queue->enqueue(scheduler->ready_queue, task);

    // 增加调度统计
    ext_scheduler->total_tasks_scheduled++;

    // 通知等待的线程
    pthread_cond_signal(&ext_scheduler->condition);

    pthread_mutex_unlock(&ext_scheduler->mutex);

    return true;
}

// 取消任务
bool task_scheduler_cancel(TaskScheduler* scheduler, uint64_t task_id) {
    if (!scheduler || task_id == 0) return false;

    ExtendedTaskScheduler* ext_scheduler = (ExtendedTaskScheduler*)scheduler;

    pthread_mutex_lock(&ext_scheduler->mutex);

    // 尝试从就绪队列中移除
    Task* task = remove_task_from_queue((PriorityTaskQueue*)scheduler->ready_queue, task_id);
    if (task) {
        task_update_status(task, TASK_STATUS_CANCELLED);
        task_set_error_message(task, "Task cancelled by scheduler");
        // 任务需要被调用者释放
        pthread_mutex_unlock(&ext_scheduler->mutex);
        return true;
    }

    // TODO: 处理正在运行的任务取消
    // 这需要与执行器协作

    pthread_mutex_unlock(&ext_scheduler->mutex);
    return false;
}

// 获取下一个要执行的任务
Task* task_scheduler_get_next_task(TaskScheduler* scheduler) {
    if (!scheduler) return NULL;

    ExtendedTaskScheduler* ext_scheduler = (ExtendedTaskScheduler*)scheduler;

    pthread_mutex_lock(&ext_scheduler->mutex);

    // 检查调度器状态
    if (ext_scheduler->state != SCHEDULER_STATE_RUNNING) {
        pthread_mutex_unlock(&ext_scheduler->mutex);
        return NULL;
    }

    // 检查是否超过最大并发数
    if (scheduler->running_task_count >= scheduler->max_concurrent_tasks) {
        pthread_mutex_unlock(&ext_scheduler->mutex);
        return NULL;
    }

    // 从就绪队列获取任务
    Task* task = scheduler->ready_queue->dequeue(scheduler->ready_queue);
    if (task) {
        task_update_status(task, TASK_STATUS_RUNNING);
        scheduler->running_task_count++;
    }

    pthread_mutex_unlock(&ext_scheduler->mutex);

    return task;
}

// 启动调度器
void task_scheduler_start(TaskScheduler* scheduler) {
    if (!scheduler) return;

    ExtendedTaskScheduler* ext_scheduler = (ExtendedTaskScheduler*)scheduler;

    pthread_mutex_lock(&ext_scheduler->mutex);

    if (ext_scheduler->state == SCHEDULER_STATE_STOPPED) {
        ext_scheduler->state = SCHEDULER_STATE_RUNNING;
        // TODO: 启动调度线程
        // 这里可以启动一个后台线程来处理调度逻辑
    }

    pthread_mutex_unlock(&ext_scheduler->mutex);
}

// 停止调度器
void task_scheduler_stop(TaskScheduler* scheduler) {
    if (!scheduler) return;

    ExtendedTaskScheduler* ext_scheduler = (ExtendedTaskScheduler*)scheduler;

    pthread_mutex_lock(&ext_scheduler->mutex);

    ext_scheduler->state = SCHEDULER_STATE_STOPPED;

    // 取消所有等待的任务
    // TODO: 实现取消逻辑

    // 通知所有等待的线程
    pthread_cond_broadcast(&ext_scheduler->condition);

    pthread_mutex_unlock(&ext_scheduler->mutex);
}

// 暂停调度器
void task_scheduler_pause(TaskScheduler* scheduler) {
    if (!scheduler) return;

    ExtendedTaskScheduler* ext_scheduler = (ExtendedTaskScheduler*)scheduler;

    pthread_mutex_lock(&ext_scheduler->mutex);

    if (ext_scheduler->state == SCHEDULER_STATE_RUNNING) {
        ext_scheduler->state = SCHEDULER_STATE_PAUSED;
    }

    pthread_mutex_unlock(&ext_scheduler->mutex);
}

// 恢复调度器
void task_scheduler_resume(TaskScheduler* scheduler) {
    if (!scheduler) return;

    ExtendedTaskScheduler* ext_scheduler = (ExtendedTaskScheduler*)scheduler;

    pthread_mutex_lock(&ext_scheduler->mutex);

    if (ext_scheduler->state == SCHEDULER_STATE_PAUSED) {
        ext_scheduler->state = SCHEDULER_STATE_RUNNING;
        // 通知等待的线程
        pthread_cond_broadcast(&ext_scheduler->condition);
    }

    pthread_mutex_unlock(&ext_scheduler->mutex);
}

// 获取运行任务数
uint32_t task_scheduler_get_running_count(const TaskScheduler* scheduler) {
    if (!scheduler) return 0;

    // 对于扩展调度器，需要线程安全访问
    const ExtendedTaskScheduler* ext_scheduler = (const ExtendedTaskScheduler*)scheduler;
    return scheduler->running_task_count; // 已经通过其他方法保护
}

// 获取队列任务数
uint32_t task_scheduler_get_queued_count(const TaskScheduler* scheduler) {
    return scheduler ? scheduler->ready_queue->size(scheduler->ready_queue) : 0;
}

// 检查调度器是否运行
bool task_scheduler_is_running(const TaskScheduler* scheduler) {
    if (!scheduler) return false;

    const ExtendedTaskScheduler* ext_scheduler = (const ExtendedTaskScheduler*)scheduler;
    return ext_scheduler->state == SCHEDULER_STATE_RUNNING;
}

// 获取调度器状态
SchedulerState task_scheduler_get_state(const TaskScheduler* scheduler) {
    if (!scheduler) return SCHEDULER_STATE_STOPPED;

    const ExtendedTaskScheduler* ext_scheduler = (const ExtendedTaskScheduler*)scheduler;
    return ext_scheduler->state;
}

// 获取调度器统计信息
bool task_scheduler_get_statistics(const TaskScheduler* scheduler,
                                  uint64_t* total_scheduled,
                                  uint64_t* total_completed,
                                  uint32_t* current_running,
                                  uint32_t* queued_count) {
    if (!scheduler || !total_scheduled || !total_completed || !current_running || !queued_count) {
        return false;
    }

    const ExtendedTaskScheduler* ext_scheduler = (const ExtendedTaskScheduler*)scheduler;

    *total_scheduled = ext_scheduler->total_tasks_scheduled;
    *total_completed = ext_scheduler->total_tasks_completed;
    *current_running = scheduler->running_task_count;
    *queued_count = task_scheduler_get_queued_count(scheduler);

    return true;
}

// 等待任务变为可用（用于调度线程）
Task* task_scheduler_wait_for_task(TaskScheduler* scheduler, uint32_t timeout_ms) {
    if (!scheduler) return NULL;

    ExtendedTaskScheduler* ext_scheduler = (ExtendedTaskScheduler*)scheduler;

    struct timespec timeout_time;
    if (timeout_ms > 0) {
        clock_gettime(CLOCK_REALTIME, &timeout_time);
        timeout_time.tv_sec += timeout_ms / 1000;
        timeout_time.tv_nsec += (timeout_ms % 1000) * 1000000;
        if (timeout_time.tv_nsec >= 1000000000) {
            timeout_time.tv_sec++;
            timeout_time.tv_nsec -= 1000000000;
        }
    }

    pthread_mutex_lock(&ext_scheduler->mutex);

    while (ext_scheduler->state == SCHEDULER_STATE_RUNNING &&
           scheduler->ready_queue->is_empty(scheduler->ready_queue)) {

        if (timeout_ms == 0) {
            // 无限等待
            pthread_cond_wait(&ext_scheduler->condition, &ext_scheduler->mutex);
        } else {
            // 带超时等待
            int result = pthread_cond_timedwait(&ext_scheduler->condition,
                                              &ext_scheduler->mutex,
                                              &timeout_time);
            if (result == ETIMEDOUT) {
                pthread_mutex_unlock(&ext_scheduler->mutex);
                return NULL; // 超时
            }
        }
    }

    // 获取任务
    Task* task = NULL;
    if (ext_scheduler->state == SCHEDULER_STATE_RUNNING) {
        task = scheduler->ready_queue->dequeue(scheduler->ready_queue);
        if (task) {
            task_update_status(task, TASK_STATUS_RUNNING);
            scheduler->running_task_count++;
        }
    }

    pthread_mutex_unlock(&ext_scheduler->mutex);

    return task;
}

// 任务完成通知
void task_scheduler_on_task_completed(TaskScheduler* scheduler, uint64_t task_id, int exit_code) {
    if (!scheduler) return;

    ExtendedTaskScheduler* ext_scheduler = (ExtendedTaskScheduler*)scheduler;

    pthread_mutex_lock(&ext_scheduler->mutex);

    // 减少运行任务计数
    if (scheduler->running_task_count > 0) {
        scheduler->running_task_count--;
        ext_scheduler->total_tasks_completed++;
    }

    // TODO: 可以在任务映射表中查找并更新任务状态
    // 这里需要任务对象的引用，但目前没有

    pthread_mutex_unlock(&ext_scheduler->mutex);
}

// 任务失败通知
void task_scheduler_on_task_failed(TaskScheduler* scheduler, uint64_t task_id, const char* error) {
    if (!scheduler) return;

    ExtendedTaskScheduler* ext_scheduler = (ExtendedTaskScheduler*)scheduler;

    pthread_mutex_lock(&ext_scheduler->mutex);

    // 减少运行任务计数
    if (scheduler->running_task_count > 0) {
        scheduler->running_task_count--;
        // 失败的任务也算已完成（虽然失败了）
        ext_scheduler->total_tasks_completed++;
    }

    // TODO: 记录错误信息和更新任务状态

    pthread_mutex_unlock(&ext_scheduler->mutex);
}
