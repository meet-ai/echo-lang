#include "scheduler.h"
#include "processor.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>

// 全局调度器实例（单例）
static Scheduler* global_scheduler = NULL;
static pthread_mutex_t global_scheduler_lock = PTHREAD_MUTEX_INITIALIZER;

// 当前执行任务
struct Task* current_task = NULL;

// 获取全局调度器
Scheduler* get_global_scheduler(void) {
    pthread_mutex_lock(&global_scheduler_lock);
    Scheduler* scheduler = global_scheduler;
    pthread_mutex_unlock(&global_scheduler_lock);
    return scheduler;
}

// 设置全局调度器
void set_global_scheduler(Scheduler* scheduler) {
    pthread_mutex_lock(&global_scheduler_lock);
    global_scheduler = scheduler;
    pthread_mutex_unlock(&global_scheduler_lock);
}

// 创建调度器
Scheduler* scheduler_create(uint32_t num_processors) {
    if (num_processors == 0) {
        fprintf(stderr, "ERROR: Cannot create scheduler with 0 processors\n");
        return NULL;
    }

    Scheduler* scheduler = (Scheduler*)malloc(sizeof(Scheduler));
    if (!scheduler) {
        fprintf(stderr, "ERROR: Failed to allocate memory for scheduler\n");
        return NULL;
    }

    memset(scheduler, 0, sizeof(Scheduler));

    // 初始化调度器属性
    scheduler->id = 1; // 暂时固定ID
    scheduler->num_processors = num_processors;
    scheduler->global_queue = NULL;
    scheduler->is_running = false;

    // 创建处理器数组
    scheduler->processors = (Processor**)malloc(sizeof(Processor*) * num_processors);
    if (!scheduler->processors) {
        fprintf(stderr, "ERROR: Failed to allocate processor array\n");
        free(scheduler);
        return NULL;
    }

    // 初始化每个处理器
    for (uint32_t i = 0; i < num_processors; i++) {
        scheduler->processors[i] = processor_create(i, 256); // 每个处理器最大256个任务
        if (!scheduler->processors[i]) {
            fprintf(stderr, "ERROR: Failed to create processor %u\n", i);
            // 清理已创建的处理器
            for (uint32_t j = 0; j < i; j++) {
                processor_destroy(scheduler->processors[j]);
            }
            free(scheduler->processors);
            free(scheduler);
            return NULL;
        }
    }

    // 创建Machine数组（每个Processor绑定一个Machine）
    scheduler->num_machines = num_processors;
    scheduler->machines = (Machine**)malloc(sizeof(Machine*) * num_processors);
    if (!scheduler->machines) {
        fprintf(stderr, "ERROR: Failed to allocate machine array\n");
        // 清理已创建的处理器
        for (uint32_t j = 0; j < num_processors; j++) {
            processor_destroy(scheduler->processors[j]);
        }
        free(scheduler->processors);
        free(scheduler);
        return NULL;
    }

    // 初始化每个Machine并绑定到Processor
    for (uint32_t i = 0; i < num_processors; i++) {
        scheduler->machines[i] = machine_create(i, scheduler);
        if (!scheduler->machines[i]) {
            fprintf(stderr, "ERROR: Failed to create machine %u\n", i);
            // 清理已创建的资源
            for (uint32_t j = 0; j < i; j++) {
                machine_destroy(scheduler->machines[j]);
            }
            for (uint32_t j = 0; j < num_processors; j++) {
                processor_destroy(scheduler->processors[j]);
            }
            free(scheduler->machines);
            free(scheduler->processors);
            free(scheduler);
            return NULL;
        }

        // 绑定Machine到Processor
        scheduler->machines[i]->processor = scheduler->processors[i];
        scheduler->processors[i]->bound_machine = scheduler->machines[i];
    }

    // 初始化锁
    pthread_mutex_init(&scheduler->global_lock, NULL);

    set_global_scheduler(scheduler);

    printf("DEBUG: Created scheduler with %u processors and machines\n", num_processors);
    return scheduler;
}

// 销毁调度器
void scheduler_destroy(Scheduler* scheduler) {
    if (!scheduler) return;

    // 停止调度器
    scheduler_stop(scheduler);

    // 销毁所有Machine
    for (uint32_t i = 0; i < scheduler->num_machines; i++) {
        if (scheduler->machines[i]) {
            machine_destroy(scheduler->machines[i]);
            scheduler->machines[i] = NULL;
        }
    }

    // 清理Machine数组
    free(scheduler->machines);
    scheduler->machines = NULL;

    // 销毁所有处理器
    for (uint32_t i = 0; i < scheduler->num_processors; i++) {
        if (scheduler->processors[i]) {
            processor_destroy(scheduler->processors[i]);
            scheduler->processors[i] = NULL;
        }
    }

    // 清理处理器数组
    free(scheduler->processors);
    scheduler->processors = NULL;

    // 清理锁
    pthread_mutex_destroy(&scheduler->global_lock);

    set_global_scheduler(NULL);

    printf("DEBUG: Destroyed scheduler\n");
    free(scheduler);
}

// 向调度器添加任务
bool scheduler_add_task(Scheduler* scheduler, Task* task) {
    if (!scheduler || !task) return false;

    // 首先尝试添加到本地队列（轮询分配到处理器）
    static uint32_t next_processor = 0;

    for (uint32_t attempts = 0; attempts < scheduler->num_processors; attempts++) {
        uint32_t processor_idx = next_processor % scheduler->num_processors;
        next_processor++;

        Processor* processor = scheduler->processors[processor_idx];
        if (processor_push_local(processor, task)) {
            printf("DEBUG: Added task %llu to processor %u local queue\n",
                   task->id, processor->id);
            return true;
        }
    }

    // 如果所有本地队列都满，添加到全局队列
    pthread_mutex_lock(&scheduler->global_lock);

    // 找到全局队列尾部
    if (scheduler->global_queue == NULL) {
        scheduler->global_queue = task;
    } else {
        Task* tail = scheduler->global_queue;
        while (tail->next) {
            tail = tail->next;
        }
        tail->next = task;
    }
    task->next = NULL;

    scheduler->tasks_scheduled++;
    printf("DEBUG: Added task %llu to global queue\n", task->id);

    pthread_mutex_unlock(&scheduler->global_lock);
    return true;
}

// 从调度器获取工作（处理器调用）
Task* scheduler_get_work(Scheduler* scheduler, Processor* processor) {
    if (!scheduler || !processor) return NULL;

    // 首先尝试从本地队列获取
    Task* task = processor_pop_local(processor);
    if (task) {
        printf("DEBUG: Processor %u got work from local queue: task %llu\n",
               processor->id, task->id);
        return task;
    }

    // 本地队列为空，尝试工作窃取
    if (scheduler_steal_work(scheduler, processor)) {
        // 窃取成功，从本地队列重新获取
        task = processor_pop_local(processor);
        if (task) {
            printf("DEBUG: Processor %u stole work: task %llu\n",
                   processor->id, task->id);
            return task;
        }
    }

    // 工作窃取失败，从全局队列获取
    pthread_mutex_lock(&scheduler->global_lock);

    if (scheduler->global_queue) {
        task = scheduler->global_queue;
        scheduler->global_queue = task->next;
        task->next = NULL;

        printf("DEBUG: Processor %u got work from global queue: task %llu\n",
               processor->id, task->id);
    }

    pthread_mutex_unlock(&scheduler->global_lock);
    return task;
}

// 工作窃取（一个处理器尝试从其他处理器窃取任务）
bool scheduler_steal_work(Scheduler* scheduler, Processor* thief) {
    if (!scheduler || !thief) return false;

    // 尝试从每个其他处理器窃取
    for (uint32_t i = 0; i < scheduler->num_processors; i++) {
        Processor* victim = scheduler->processors[i];
        if (victim == thief) continue; // 不从自己窃取

        Task* stolen_task = processor_steal_local(victim);
        if (stolen_task) {
            // 将窃取的任务添加到自己的本地队列
            if (processor_push_local(thief, stolen_task)) {
                printf("DEBUG: Processor %u successfully stole task %llu from processor %u\n",
                       thief->id, stolen_task->id, victim->id);
                return true;
            } else {
                // 如果添加失败，暂时放弃这个任务
                // 在实际实现中可能需要重新放回受害者队列
                fprintf(stderr, "WARNING: Failed to add stolen task to thief queue\n");
            }
        }
    }

    return false;
}

// 运行调度器（暂时恢复单线程实现，用于调试）
void scheduler_run(Scheduler* scheduler) {
    if (!scheduler || scheduler->is_running) return;

    scheduler->is_running = true;
    printf("DEBUG: Scheduler started with %u processors (single-threaded)\n", scheduler->num_processors);

    // 单线程轮询实现（临时恢复，用于确保spawn功能正常）
    while (scheduler->is_running) {
        bool has_work = false;

        for (uint32_t i = 0; i < scheduler->num_processors; i++) {
            Processor* processor = scheduler->processors[i];

            // 调试：检查队列大小
            uint32_t queue_size = processor_get_queue_size(processor);
            printf("DEBUG: Processor %u queue size: %u\n", processor->id, queue_size);

            Task* task = scheduler_get_work(scheduler, processor);

            if (task) {
                has_work = true;
                printf("DEBUG: Executing task %llu on processor %u\n",
                       task->id, processor->id);

                // 执行任务
                task_execute(task);

                // 检查任务是否完成
                if (task_is_complete(task)) {
                    scheduler->tasks_completed++;
                    printf("DEBUG: Task %llu completed\n", task->id);

                    // 清理任务
                    task_destroy(task);
                } else {
                    // 任务未完成，重新放回队列
                    processor_push_local(processor, task);
                }
            } else {
                printf("DEBUG: No work found for processor %u\n", processor->id);
            }
        }

        // 如果没有工作且队列为空，退出循环
        if (!has_work && scheduler->global_queue == NULL) {
            bool all_queues_empty = true;
            for (uint32_t i = 0; i < scheduler->num_processors; i++) {
                if (processor_get_queue_size(scheduler->processors[i]) > 0) {
                    all_queues_empty = false;
                    break;
                }
            }

            if (all_queues_empty) {
                printf("DEBUG: All queues empty, stopping scheduler\n");
                break;
            }
        }

        // 避免忙等待
        usleep(1000); // 1ms
    }

    scheduler->is_running = false;
    printf("DEBUG: Scheduler stopped\n");
}

// 停止调度器
void scheduler_stop(Scheduler* scheduler) {
    if (!scheduler) return;

    scheduler->is_running = false;
    printf("DEBUG: Scheduler stop requested\n");
}

// 打印调度器统计信息
void scheduler_print_stats(Scheduler* scheduler) {
    if (!scheduler) return;

    printf("Scheduler stats:\n");
    printf("  Processors: %u\n", scheduler->num_processors);
    printf("  Tasks scheduled: %llu\n", scheduler->tasks_scheduled);
    printf("  Tasks completed: %llu\n", scheduler->tasks_completed);
    printf("  Completion rate: %.2f%%\n",
           scheduler->tasks_scheduled > 0 ?
           (double)scheduler->tasks_completed / scheduler->tasks_scheduled * 100 : 0);

    printf("  Processor stats:\n");
    for (uint32_t i = 0; i < scheduler->num_processors; i++) {
        processor_print_stats(scheduler->processors[i]);
    }
}
