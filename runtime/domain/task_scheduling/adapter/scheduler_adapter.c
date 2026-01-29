/**
 * @file scheduler_adapter.c
 * @brief TaskScheduling适配层实现
 */

#include "scheduler_adapter.h"
#include "../aggregate/scheduler.h"  // 包含新的聚合根定义
#include "../../task_execution/repository/task_repository.h"
#include "../../task_execution/aggregate/task.h"  // Task聚合根，包含task_get_id函数
#include "../../task_execution/aggregate/coroutine.h"  // Coroutine完整定义
#include "../../scheduler/concurrency/processor.h"
#include "../../scheduler/concurrency/machine.h"
// 事件总线接口
#include "../../shared/events/bus.h"
#include <stdlib.h>
#include <stdio.h>
#include <pthread.h>
#include <stdint.h>

// ==================== 全局TaskRepository管理 ====================

static TaskRepository* global_task_repository = NULL;
static pthread_mutex_t global_task_repository_lock = PTHREAD_MUTEX_INITIALIZER;

void scheduler_adapter_set_task_repository(TaskRepository* task_repo) {
    pthread_mutex_lock(&global_task_repository_lock);
    global_task_repository = task_repo;
    pthread_mutex_unlock(&global_task_repository_lock);
}

TaskRepository* scheduler_adapter_get_task_repository(void) {
    pthread_mutex_lock(&global_task_repository_lock);
    TaskRepository* repo = global_task_repository;
    pthread_mutex_unlock(&global_task_repository_lock);
    return repo;
}

// ==================== 全局EventBus管理 ====================

static EventBus* global_event_bus = NULL;
static pthread_mutex_t global_event_bus_lock = PTHREAD_MUTEX_INITIALIZER;

void scheduler_adapter_set_event_bus(EventBus* event_bus) {
    pthread_mutex_lock(&global_event_bus_lock);
    global_event_bus = event_bus;
    pthread_mutex_unlock(&global_event_bus_lock);
}

EventBus* scheduler_adapter_get_event_bus(void) {
    pthread_mutex_lock(&global_event_bus_lock);
    EventBus* bus = global_event_bus;
    pthread_mutex_unlock(&global_event_bus_lock);
    return bus;
}

// ==================== 全局Scheduler管理 ====================

static Scheduler* global_scheduler = NULL;
static pthread_mutex_t global_scheduler_lock = PTHREAD_MUTEX_INITIALIZER;

// ==================== 当前执行任务全局变量 ====================
// 注意：这个变量在scheduler.h中声明为extern，供其他模块使用

Task* current_task = NULL;  // 当前正在执行的任务（使用新的Task聚合根类型）

// ==================== 向后兼容接口实现 ====================

Scheduler* scheduler_create(uint32_t num_processors) {
    // 获取全局TaskRepository
    TaskRepository* task_repo = scheduler_adapter_get_task_repository();
    if (!task_repo) {
        // 对于简单程序（如map迭代测试），可能不需要完整的调度器
        // 静默返回NULL，让调用者决定如何处理
        return NULL;
    }

    // 获取全局EventBus（如果已设置）
    EventBus* event_bus = scheduler_adapter_get_event_bus();
    
    // 调用新的工厂方法，传入EventBus
    Scheduler* scheduler = scheduler_factory_create(num_processors, task_repo, event_bus);
    if (!scheduler) {
        return NULL;
    }

    // 设置全局调度器
    pthread_mutex_lock(&global_scheduler_lock);
    global_scheduler = scheduler;
    pthread_mutex_unlock(&global_scheduler_lock);

    return scheduler;
}

void scheduler_destroy(Scheduler* scheduler) {
    if (!scheduler) return;

    // 清除全局调度器
    pthread_mutex_lock(&global_scheduler_lock);
    if (global_scheduler == scheduler) {
        global_scheduler = NULL;
    }
    pthread_mutex_unlock(&global_scheduler_lock);

    // 调用新的聚合根销毁方法
    scheduler_aggregate_destroy(scheduler);
}

bool scheduler_add_task(Scheduler* scheduler, Task* task) {
    if (!scheduler || !task) {
        return false;
    }

    // 提取TaskID
    TaskID task_id = task_get_id(task);

    // 调用新的聚合根方法
    int result = scheduler_aggregate_add_task(scheduler, task_id);
    return (result == 0);
}

Task* scheduler_get_work(Scheduler* scheduler, Processor* processor) {
    if (!scheduler || !processor) {
        return NULL;
    }

    // 调用新的聚合根方法获取TaskID
    TaskID task_id = 0;
    int result = scheduler_aggregate_get_work(scheduler, processor, &task_id);
    if (result != 0 || task_id == 0) {
        return NULL;
    }

    // 通过TaskRepository查找Task
    TaskRepository* task_repo = scheduler_adapter_get_task_repository();
    if (!task_repo) {
        fprintf(stderr, "ERROR: TaskRepository not set\n");
        return NULL;
    }

    Task* task = task_repository_find_by_id(task_repo, task_id);
    if (!task) {
        fprintf(stderr, "ERROR: Failed to find task %llu in repository\n", task_id);
        return NULL;
    }

    return task;
}

void scheduler_run(Scheduler* scheduler) {
    if (!scheduler) return;

    // 调用新的聚合根方法
    scheduler_aggregate_start(scheduler);

    // TODO: 实现scheduler_run的完整逻辑（启动所有Machine线程等）
    // 这需要查看旧的scheduler_run实现
}

void scheduler_stop(Scheduler* scheduler) {
    if (!scheduler) return;

    // 调用新的聚合根方法
    scheduler_aggregate_stop(scheduler);
}

bool scheduler_steal_work(Scheduler* scheduler, Processor* thief) {
    if (!scheduler || !thief) {
        return false;
    }

    // 调用新的聚合根方法
    int result = scheduler_aggregate_steal_work(scheduler, thief);
    return (result == 0);
}

// scheduler_print_stats 已在聚合根中定义，这里移除重复定义
// 如果需要向后兼容，可以在这里调用聚合根的方法
// 但为了避免重复符号错误，暂时注释掉
// void scheduler_print_stats(Scheduler* scheduler) {
//     if (!scheduler) return;
// 
//     printf("=== Scheduler Stats ===\n");
//     printf("  Scheduler ID: %u\n", scheduler_get_id(scheduler));
//     printf("  Is running: %s\n", scheduler_is_running(scheduler) ? "yes" : "no");
//     printf("  Processors: %u\n", scheduler->num_processors);
//     printf("  Machines: %u\n", scheduler->num_machines);
//     printf("  Tasks scheduled: %llu\n", scheduler_get_tasks_scheduled(scheduler));
//     printf("  Tasks completed: %llu\n", scheduler_get_tasks_completed(scheduler));
//     
//     // TODO: 实现Processor统计信息的打印
// }
// 注意：scheduler_print_stats 应该在聚合根中定义，适配层不应该重新实现

// scheduler_notify_coroutine_completed 适配层包装函数
// 注意：这个函数在适配层中定义是为了向后兼容，它调用聚合根的方法
void scheduler_notify_coroutine_completed(Scheduler* scheduler, struct Coroutine* coroutine) {
    if (!scheduler || !coroutine) return;

    // 通过coroutine->task获取TaskID
    if (!coroutine->task) {
        fprintf(stderr, "WARNING: Coroutine %llu has no associated task\n", coroutine->id);
        return;
    }

    // 使用Task聚合根方法获取TaskID
    extern TaskID task_get_id(const struct Task* task);
    TaskID task_id = task_get_id(coroutine->task);

    // 调用聚合根方法处理任务完成（更新计数并发布事件）
    extern int scheduler_aggregate_notify_task_completed(Scheduler* scheduler, TaskID task_id);
    int result = scheduler_aggregate_notify_task_completed(scheduler, task_id);
    if (result != 0) {
        fprintf(stderr, "ERROR: Failed to notify task completion for task %llu\n", task_id);
    }
}

Scheduler* get_global_scheduler(void) {
    pthread_mutex_lock(&global_scheduler_lock);
    Scheduler* scheduler = global_scheduler;
    pthread_mutex_unlock(&global_scheduler_lock);
    return scheduler;
}

// ==================== 辅助查询函数实现 ====================

// scheduler_has_work_in_global_queue 已在聚合根中定义，这里移除重复定义
// bool scheduler_has_work_in_global_queue(Scheduler* scheduler) {
//     if (!scheduler) return false;
//     
//     // 检查新的聚合根中的TaskIDNode队列
//     pthread_mutex_lock(&scheduler->global_lock);
//     bool has_work = (scheduler->global_queue != NULL);
//     pthread_mutex_unlock(&scheduler->global_lock);
//     
//     return has_work;
// }

uint32_t scheduler_get_num_processors(Scheduler* scheduler) {
    if (!scheduler) return 0;
    return scheduler->num_processors;
}

uint32_t scheduler_get_num_machines(Scheduler* scheduler) {
    if (!scheduler) return 0;
    return scheduler->num_machines;
}

Processor* scheduler_get_processor_by_index(Scheduler* scheduler, uint32_t index) {
    if (!scheduler) return NULL;
    if (index >= scheduler->num_processors) return NULL;
    if (!scheduler->processors) return NULL;
    return scheduler->processors[index];
}

int scheduler_add_task_to_global_queue(Scheduler* scheduler, Task* task) {
    if (!scheduler || !task) return -1;
    
    // 提取TaskID
    TaskID task_id = task_get_id(task);
    
    // 调用聚合根方法添加到全局队列
    return scheduler_aggregate_add_task(scheduler, task_id);
}
