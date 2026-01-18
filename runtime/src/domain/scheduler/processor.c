/**
 * @file processor.c
 * @brief Processor (P) 聚合根实现
 *
 * Processor 管理本地任务队列，支持工作窃取算法。
 */

#include "processor.h"
#include "scheduler.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <pthread.h>

// 前向声明
struct Machine;
struct Scheduler;

// ============================================================================
// 内部辅助函数
// ============================================================================

/**
 * @brief 从指定受害者处理器窃取任务
 */
static task_t* steal_from_victim(processor_t* thief, processor_t* victim);

// ============================================================================
// Processor 聚合根实现
// ============================================================================

/**
 * @brief 创建处理器
 */
processor_t* processor_create(uint32_t id, size_t queue_capacity) {
    processor_t* processor = (processor_t*)malloc(sizeof(processor_t));
    if (!processor) {
        fprintf(stderr, "Failed to allocate processor\n");
        return NULL;
    }

    memset(processor, 0, sizeof(processor_t));

    processor->id = id;
    processor->bound_machine = NULL;
    processor->queue_capacity = queue_capacity;
    processor->queue_size = 0;
    processor->queue_head = 0;
    processor->queue_tail = 0;
    processor->tasks_executed = 0;
    processor->tasks_stolen = 0;
    processor->steal_attempts = 0;

    // 分配队列内存
    processor->local_queue = (task_t**)malloc(sizeof(task_t*) * queue_capacity);
    if (!processor->local_queue) {
        fprintf(stderr, "Failed to allocate local queue\n");
        free(processor);
        return NULL;
    }

    // 初始化同步机制
    if (pthread_mutex_init(&processor->queue_mutex, NULL) != 0) {
        fprintf(stderr, "Failed to initialize queue mutex\n");
        free(processor->local_queue);
        free(processor);
        return NULL;
    }

    if (pthread_cond_init(&processor->queue_cond, NULL) != 0) {
        fprintf(stderr, "Failed to initialize queue condition\n");
        pthread_mutex_destroy(&processor->queue_mutex);
        free(processor->local_queue);
        free(processor);
        return NULL;
    }

    printf("DEBUG: Created Processor %u with queue capacity %zu\n", id, queue_capacity);
    return processor;
}

/**
 * @brief 销毁处理器
 */
void processor_destroy(processor_t* processor) {
    if (!processor) return;

    // 清理队列中的任务
    pthread_mutex_lock(&processor->queue_mutex);
    for (size_t i = 0; i < processor->queue_size; i++) {
        size_t index = (processor->queue_head + i) % processor->queue_capacity;
        if (processor->local_queue[index]) {
            task_destroy(processor->local_queue[index]);
        }
    }
    pthread_mutex_unlock(&processor->queue_mutex);

    // 清理同步机制
    pthread_cond_destroy(&processor->queue_cond);
    pthread_mutex_destroy(&processor->queue_mutex);

    // 释放内存
    free(processor->local_queue);

    printf("DEBUG: Destroyed Processor %u\n", processor->id);
    free(processor);
}

/**
 * @brief 绑定到机器
 */
void processor_bind_machine(processor_t* processor, struct Machine* machine) {
    if (processor) {
        processor->bound_machine = machine;
        printf("DEBUG: Processor %u bound to machine\n", processor->id);
    }
}

/**
 * @brief 设置调度器引用
 */
void processor_set_scheduler(processor_t* processor, struct Scheduler* scheduler) {
    if (processor) {
        processor->scheduler = scheduler;
        printf("DEBUG: Processor %u set scheduler reference\n", processor->id);
    }
}

/**
 * @brief 推入任务到本地队列（LIFO）
 */
bool processor_push_task(processor_t* processor, task_t* task) {
    if (!processor || !task) return false;

    pthread_mutex_lock(&processor->queue_mutex);

    if (processor->queue_size >= processor->queue_capacity) {
        pthread_mutex_unlock(&processor->queue_mutex);
        return false; // 队列已满
    }

    // LIFO: 推入到头部（实际是环形队列的逻辑头部）
    size_t new_head = (processor->queue_head + processor->queue_capacity - 1) % processor->queue_capacity;
    processor->local_queue[new_head] = task;
    processor->queue_head = new_head;
    processor->queue_size++;

    pthread_cond_signal(&processor->queue_cond); // 唤醒等待的线程
    pthread_mutex_unlock(&processor->queue_mutex);

    printf("DEBUG: Processor %u pushed task %llu, queue size: %zu\n",
           processor->id, task_get_id(task), processor->queue_size);
    return true;
}

/**
 * @brief 从本地队列弹出任务（LIFO）
 */
task_t* processor_pop_task(processor_t* processor) {
    if (!processor) return NULL;

    pthread_mutex_lock(&processor->queue_mutex);

    if (processor->queue_size == 0) {
        pthread_mutex_unlock(&processor->queue_mutex);
        return NULL;
    }

    // LIFO: 从头部弹出
    task_t* task = processor->local_queue[processor->queue_head];
    processor->queue_head = (processor->queue_head + 1) % processor->queue_capacity;
    processor->queue_size--;

    pthread_mutex_unlock(&processor->queue_mutex);

    printf("DEBUG: Processor %u popped task %llu, queue size: %zu\n",
           processor->id, task_get_id(task), processor->queue_size);
    return task;
}

/**
 * @brief 从其他处理器窃取任务（FIFO）
 */
task_t* processor_steal_task(processor_t* thief_processor) {
    if (!thief_processor || !thief_processor->scheduler) return NULL;

    thief_processor->steal_attempts++;

    struct Scheduler* scheduler = thief_processor->scheduler;

    // 工作窃取算法：从其他处理器的队列尾部窃取任务
    for (uint32_t i = 0; i < scheduler->num_processors; i++) {
        processor_t* victim_processor = scheduler->processors[i];

        // 不从自己这里窃取
        if (victim_processor == thief_processor) continue;

        // 尝试从受害者处理器窃取任务
        task_t* stolen_task = steal_from_victim(thief_processor, victim_processor);
        if (stolen_task) {
            printf("DEBUG: Processor %u successfully stole task %llu from Processor %u\n",
                   thief_processor->id, task_get_id(stolen_task), victim_processor->id);
            return stolen_task;
        }
    }

    return NULL;
}

/**
 * @brief 执行一个任务
 */
void processor_execute_task(processor_t* processor, task_t* task) {
    if (!processor || !task) return;

    printf("DEBUG: Processor %u executing task %llu\n", processor->id, task_get_id(task));

    // 执行任务
    task_execute(task, NULL);  // 暂时传入NULL，后面会传入正确的scheduler_context
    processor->tasks_executed++;

    // 检查任务是否完成
    if (task_is_completed(task)) {
        printf("DEBUG: Processor %u completed task %llu\n", processor->id, task_get_id(task));

        // 通知调度器任务完成
        if (processor->scheduler) {
            scheduler_task_completed(processor->scheduler, task);
        }

        // 任务完成，可以清理或放回池中
        // 这里暂时不清理，让调用者处理
    }
}

/**
 * @brief 获取队列大小
 */
size_t processor_get_queue_size(const processor_t* processor) {
    if (!processor) return 0;

    pthread_mutex_lock((pthread_mutex_t*)&processor->queue_mutex);
    size_t size = processor->queue_size;
    pthread_mutex_unlock((pthread_mutex_t*)&processor->queue_mutex);

    return size;
}

// ============================================================================
// 内部辅助函数实现
// ============================================================================

/**
 * @brief 从指定受害者处理器窃取任务
 */
static task_t* steal_from_victim(processor_t* thief, processor_t* victim) {
    if (!thief || !victim) return NULL;

    // 尝试获取受害者队列的锁（非阻塞）
    if (pthread_mutex_trylock(&victim->queue_mutex) != 0) {
        // 获取锁失败，说明受害者正在使用队列，跳过
        return NULL;
    }

    task_t* stolen_task = NULL;

    if (victim->queue_size > 0) {
        // 从受害者队列的尾部窃取任务（FIFO for victim）
        // 由于我们的队列实现是LIFO的，我们需要从逻辑尾部窃取
        // 在环形队列中，尾部是head前面的位置

        // 计算要窃取的位置（队列的逻辑尾部）
        size_t steal_index = victim->queue_head;
        if (victim->queue_size > 1) {
            // 如果有多个任务，我们从"尾部"窃取（实际是数组中的前面位置）
            steal_index = (victim->queue_head + victim->queue_capacity - 1) % victim->queue_capacity;
        }

        stolen_task = victim->local_queue[steal_index];
        victim->local_queue[steal_index] = NULL;
        victim->queue_size--;

        if (stolen_task) {
            victim->tasks_stolen++;
            printf("DEBUG: Processor %u stole task from Processor %u\n",
                   thief->id, victim->id);
        }
    }

    pthread_mutex_unlock(&victim->queue_mutex);
    return stolen_task;
}

/**
 * @brief 检查队列是否为空
 */
bool processor_queue_empty(const processor_t* processor) {
    return processor_get_queue_size(processor) == 0;
}

/**
 * @brief 获取处理器统计信息
 */
void processor_get_stats(const processor_t* processor,
                        uint64_t* executed,
                        uint64_t* stolen,
                        uint64_t* attempts) {
    if (!processor) return;

    if (executed) *executed = processor->tasks_executed;
    if (stolen) *stolen = processor->tasks_stolen;
    if (attempts) *attempts = processor->steal_attempts;
}
