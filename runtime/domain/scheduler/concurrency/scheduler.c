#include "scheduler.h"
#include "processor.h"
#include "../../coroutine/coroutine.h"  // 包含Coroutine定义
#include "../../task_execution/aggregate/task.h"  // 包含Task定义和task_start函数
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>

// 函数声明：task_execute可能在不同的文件中
// 暂时使用task_start代替，或者直接调用任务的entry_point
static void task_execute_impl(Task* task) {
    if (!task || !task->entry_point) return;
    // 启动任务（这会调用entry_point）
    task_start(task);
}

// 全局调度器实例（单例）
static Scheduler* global_scheduler = NULL;
static pthread_mutex_t global_scheduler_lock = PTHREAD_MUTEX_INITIALIZER;

// 当前执行任务（使用新的Task聚合根类型）
Task* current_task = NULL;

// 获取全局调度器（已迁移到适配层，此函数改为static避免重复符号）
// 注意：适配层也有此函数，这里改为static避免链接冲突
static Scheduler* get_global_scheduler_internal(void) {
    pthread_mutex_lock(&global_scheduler_lock);
    Scheduler* scheduler = global_scheduler;
    pthread_mutex_unlock(&global_scheduler_lock);
    return scheduler;
}

// 导出函数：调用适配层的实现（避免重复符号）
Scheduler* get_global_scheduler(void) {
    // 如果适配层已设置，使用适配层的实现
    // 否则使用内部实现（向后兼容）
    extern Scheduler* get_global_scheduler(void);  // 声明适配层的函数
    // 注意：这里会有链接冲突，需要从scheduler_adapter.c中移除或使用不同的名称
    return get_global_scheduler_internal();
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
// TODO: 阶段5需要更新为接受TaskID而不是Task*，并使用TaskRepository查找Task
bool scheduler_add_task(Scheduler* scheduler, Task* task) {
    if (!scheduler || !task) return false;

    // 使用新的Task聚合根方法获取TaskID
    TaskID task_id = task_get_id(task);

    // 首先尝试添加到本地队列（轮询分配到处理器）
    static uint32_t next_processor = 0;

    for (uint32_t attempts = 0; attempts < scheduler->num_processors; attempts++) {
        uint32_t processor_idx = next_processor % scheduler->num_processors;
        next_processor++;

        Processor* processor = scheduler->processors[processor_idx];
        // TODO: 阶段5需要更新processor_push_local为接受TaskID
        if (processor_push_local(processor, task)) {
            printf("DEBUG: Added task %llu to processor %u local queue\n",
                   task_id, processor->id);
            return true;
        }
    }

    // 如果所有本地队列都满，添加到全局队列
    // TODO: 阶段5需要更新scheduler->global_queue为TaskID列表
    pthread_mutex_lock(&scheduler->global_lock);

    // 找到全局队列尾部
    // 注意：新的Task聚合根有next字段（用于队列管理）
    if (scheduler->global_queue == NULL) {
        scheduler->global_queue = task;  // TODO: 阶段5改为TaskID列表
    } else {
        Task* tail = scheduler->global_queue;
        // TODO: 阶段5改为TaskID列表操作
        while (tail->next) {  // 新的Task聚合根有next字段
            tail = tail->next;
        }
        tail->next = task;  // 新的Task聚合根有next字段
    }
    task->next = NULL;  // 新的Task聚合根有next字段

    scheduler->tasks_scheduled++;
    printf("DEBUG: Added task %llu to global queue\n", task_id);

    pthread_mutex_unlock(&scheduler->global_lock);
    return true;
}

// 从调度器获取工作（处理器调用）
// scheduler_get_work（已迁移到适配层，此函数改为static避免重复符号）
static Task* scheduler_get_work_internal(Scheduler* scheduler, Processor* processor) {
    if (!scheduler || !processor) return NULL;

    // 首先尝试从本地队列获取
    Task* task = processor_pop_local(processor);
    if (task) {
        TaskID task_id = task_get_id(task);  // 使用新的聚合根方法
        printf("DEBUG: Processor %u got work from local queue: task %llu\n",
               processor->id, task_id);
        return task;
    }

    // 本地队列为空，尝试工作窃取
    if (scheduler_steal_work(scheduler, processor)) {
        // 窃取成功，从本地队列重新获取
        task = processor_pop_local(processor);
        if (task) {
            TaskID task_id = task_get_id(task);  // 使用新的聚合根方法
            printf("DEBUG: Processor %u stole work: task %llu\n",
                   processor->id, task_id);
            return task;
        }
    }

    // 工作窃取失败，从全局队列获取
    pthread_mutex_lock(&scheduler->global_lock);

    if (scheduler->global_queue) {
        task = scheduler->global_queue;
        scheduler->global_queue = task->next;
        task->next = NULL;

        TaskID task_id = task_get_id(task);  // 使用新的聚合根方法
        printf("DEBUG: Processor %u got work from global queue: task %llu\n",
               processor->id, task_id);
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
                TaskID stolen_task_id = task_get_id(stolen_task);  // 使用新的聚合根方法
                printf("DEBUG: Processor %u successfully stole task %llu from processor %u\n",
                       thief->id, stolen_task_id, victim->id);
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

            // 调用适配层的实现（通过extern声明）
            extern Task* scheduler_get_work(Scheduler* scheduler, Processor* processor);
            Task* task = scheduler_get_work(scheduler, processor);

            if (task) {
                has_work = true;
                printf("DEBUG: Executing task %llu on processor %u\n",
                       task->id, processor->id);

                // 执行任务
                task_execute_impl(task);

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

// 检查调度器是否正在运行
bool scheduler_is_running(Scheduler* scheduler) {
    if (!scheduler) return false;
    return scheduler->is_running;
}

// 通知调度器协程已完成（适配层函数）
void scheduler_notify_coroutine_completed(Scheduler* scheduler, struct Coroutine* coroutine) {
    if (!scheduler || !coroutine) return;
    // 暂时不实现具体逻辑，只是占位函数
    printf("DEBUG: Coroutine %llu completed\n", coroutine->id);
}

// 根据索引获取处理器
struct Processor* scheduler_get_processor_by_index(Scheduler* scheduler, uint32_t index) {
    if (!scheduler || index >= scheduler->num_processors) return NULL;
    return scheduler->processors[index];
}

// 检查全局队列是否有工作
bool scheduler_has_work_in_global_queue(Scheduler* scheduler) {
    if (!scheduler) return false;
    pthread_mutex_lock(&scheduler->global_lock);
    bool has_work = (scheduler->global_queue != NULL);
    pthread_mutex_unlock(&scheduler->global_lock);
    return has_work;
}

// 获取处理器数量
uint32_t scheduler_get_num_processors(Scheduler* scheduler) {
    if (!scheduler) return 0;
    return scheduler->num_processors;
}

// 获取机器数量
uint32_t scheduler_get_num_machines(Scheduler* scheduler) {
    if (!scheduler) return 0;
    return scheduler->num_machines;
}

// runtime_null_ptr 和 runtime_map_get_keys 已迁移到 runtime/api/runtime_map_iter.c
// 这里不再定义，避免重复符号错误
// 如果需要使用，请链接 runtime_map_iter.o

// 以下代码已注释，实际实现请使用 runtime/api/runtime_map_iter.c
/*
void* runtime_null_ptr(void) {
    return NULL;
}

MapIterResult* runtime_map_get_keys(void* map_ptr) {
    if (!map_ptr) {
        return NULL;
    }
    
    // 分配结果结构体
    MapIterResult* result = (MapIterResult*)malloc(sizeof(MapIterResult));
    if (!result) {
        return NULL;
    }
    
    // 初始化
    result->keys = NULL;
    result->values = NULL;
    result->count = 0;
    
    // 前向声明hash_table_t和hash_entry_t结构
    // 注意：这些结构定义在runtime/shared/types/common_types.h中
    // string_t是结构体：{ char* data; size_t length; size_t capacity; }
    typedef struct {
        char* data;
        size_t length;
        size_t capacity;
    } string_t;
    
    typedef struct hash_entry_internal {
        string_t key;          // string_t key
        void* value;          // void* value（对于map[string]string，value是string_t*）
        struct hash_entry_internal* next;
    } hash_entry_internal_t;
    
    typedef struct {
        hash_entry_internal_t** buckets;  // hash_entry_t** buckets
        size_t bucket_count;              // size_t bucket_count
        size_t entry_count;               // size_t entry_count
        void* hash_function;              // uint32_t (*hash_function)(const char* key)
    } hash_table_internal_t;
    
    hash_table_internal_t* map = (hash_table_internal_t*)map_ptr;
    
    // 如果map为空，返回空结果
    if (!map || map->entry_count == 0) {
        return result;
    }
    
    // 分配键数组和值数组
    // 注意：键和值都是char*（string类型）
    result->keys = (char**)malloc(sizeof(char*) * map->entry_count);
    result->values = (char**)malloc(sizeof(char*) * map->entry_count);
    if (!result->keys || !result->values) {
        if (result->keys) free(result->keys);
        if (result->values) free(result->values);
        free(result);
        return NULL;
    }
    
    // 遍历所有bucket和entry，收集键值对
    size_t index = 0;
    for (size_t i = 0; i < map->bucket_count; i++) {
        if (!map->buckets) break;
        hash_entry_internal_t* entry = map->buckets[i];
        while (entry) {
            if (index >= map->entry_count) {
                // 防止数组越界
                break;
            }
            
            // 复制键（string_t.data是char*）
            if (entry->key.data && entry->key.length > 0) {
                size_t key_len = entry->key.length + 1;
                result->keys[index] = (char*)malloc(key_len);
                if (result->keys[index]) {
                    memcpy(result->keys[index], entry->key.data, entry->key.length);
                    result->keys[index][entry->key.length] = '\0';
                }
            } else {
                result->keys[index] = NULL;
            }
            
            // 复制值（对于map[string]string，value是string_t*）
            if (entry->value) {
                string_t* value_str = (string_t*)entry->value;
                if (value_str->data && value_str->length > 0) {
                    size_t value_len = value_str->length + 1;
                    result->values[index] = (char*)malloc(value_len);
                    if (result->values[index]) {
                        memcpy(result->values[index], value_str->data, value_str->length);
                        result->values[index][value_str->length] = '\0';
                    }
                } else {
                    result->values[index] = NULL;
                }
            } else {
                result->values[index] = NULL;
            }
            
            index++;
            entry = entry->next;
        }
    }
    
    result->count = (int32_t)index;
    
    return result;
}
*/

// 获取调度器ID
uint32_t scheduler_get_id(Scheduler* scheduler) {
    if (!scheduler) return 0;
    return scheduler->id;
}

// 获取已调度任务数
uint64_t scheduler_get_tasks_scheduled(Scheduler* scheduler) {
    if (!scheduler) return 0;
    return scheduler->tasks_scheduled;
}

// 获取已完成任务数
uint64_t scheduler_get_tasks_completed(Scheduler* scheduler) {
    if (!scheduler) return 0;
    return scheduler->tasks_completed;
}
