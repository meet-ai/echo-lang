/**
 * @file scheduler.c
 * @brief GMP Scheduler 聚合根实现
 *
 * GMP调度器实现工作窃取算法和负载均衡。
 */

#include "scheduler.h"
#include "processor.h"
#include "machine.h"
#include "../coroutine/context.h"  // 上下文切换
#include "../coroutine/coroutine.h"  // 协程相关
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

// ============================================================================
// GMP Scheduler 聚合根实现
// ============================================================================

/**
 * @brief 创建GMP调度器
 */
scheduler_t* scheduler_create(uint32_t num_processors, uint32_t num_machines) {
    scheduler_t* scheduler = (scheduler_t*)malloc(sizeof(scheduler_t));
    if (!scheduler) {
        fprintf(stderr, "Failed to allocate scheduler\n");
        return NULL;
    }

    memset(scheduler, 0, sizeof(scheduler_t));

    // 使用原子计数器分配唯一ID
    scheduler->id = scheduler_allocate_id();
    scheduler->num_processors = num_processors;
    scheduler->num_machines = num_machines;
    scheduler->is_running = false;
    scheduler->tasks_scheduled = 0;
    scheduler->tasks_completed = 0;

    // 分配处理器数组
    if (num_processors > 0) {
        scheduler->processors = (struct Processor**)malloc(sizeof(struct Processor*) * num_processors);
        if (!scheduler->processors) {
            fprintf(stderr, "Failed to allocate processors array\n");
            free(scheduler);
            return NULL;
        }

        // 创建处理器
        for (uint32_t i = 0; i < num_processors; i++) {
            scheduler->processors[i] = processor_create(i, DEFAULT_QUEUE_SIZE);
            if (!scheduler->processors[i]) {
                fprintf(stderr, "Failed to create processor %u\n", i);
                // 清理已创建的处理器
                for (uint32_t j = 0; j < i; j++) {
                    processor_destroy(scheduler->processors[j]);
                }
                free(scheduler->processors);
                free(scheduler);
                return NULL;
            }
            // 设置调度器引用以支持工作窃取
            processor_set_scheduler(scheduler->processors[i], scheduler);
        }
    }

    // 分配机器数组
    if (num_machines > 0) {
        scheduler->machines = (struct Machine**)malloc(sizeof(struct Machine*) * num_machines);
        if (!scheduler->machines) {
            fprintf(stderr, "Failed to allocate machines array\n");
            // 清理处理器
            for (uint32_t i = 0; i < num_processors; i++) {
                processor_destroy(scheduler->processors[i]);
            }
            free(scheduler->processors);
            free(scheduler);
            return NULL;
        }

        // 创建机器
        for (uint32_t i = 0; i < num_machines; i++) {
            scheduler->machines[i] = machine_create(i);
            if (!scheduler->machines[i]) {
                fprintf(stderr, "Failed to create machine %u\n", i);
                // 清理已创建的机器和处理器
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
        }

        // 绑定处理器到机器（简化：每个机器绑定一个处理器）
        for (uint32_t i = 0; i < num_machines && i < num_processors; i++) {
            machine_bind_processor(scheduler->machines[i], scheduler->processors[i]);
            processor_bind_machine(scheduler->processors[i], scheduler->machines[i]);
        }
    }

    // 初始化GMP队列
    size_t queue_capacity = DEFAULT_QUEUE_SIZE * 4; // 更大的队列

    // 分配就绪队列
    scheduler->ready_queue_capacity = queue_capacity;
    scheduler->ready_queue = (task_t**)malloc(sizeof(task_t*) * scheduler->ready_queue_capacity);
    if (!scheduler->ready_queue) {
        fprintf(stderr, "Failed to allocate ready queue\n");
        goto cleanup;
    }

    // 分配阻塞队列
    scheduler->blocked_queue_capacity = queue_capacity / 2; // 阻塞队列可以小一些
    scheduler->blocked_queue = (task_t**)malloc(sizeof(task_t*) * scheduler->blocked_queue_capacity);
    if (!scheduler->blocked_queue) {
        fprintf(stderr, "Failed to allocate blocked queue\n");
        goto cleanup;
    }

    scheduler->ready_queue_size = 0;
    scheduler->blocked_queue_size = 0;

    // 初始化同步机制
    if (pthread_mutex_init(&scheduler->global_mutex, NULL) != 0) {
        fprintf(stderr, "Failed to initialize global mutex\n");
        goto cleanup;
    }

    if (pthread_cond_init(&scheduler->global_cond, NULL) != 0) {
        fprintf(stderr, "Failed to initialize global condition\n");
        goto cleanup;
    }

    printf("DEBUG: Created GMP Scheduler with %u processors and %u machines\n",
           num_processors, num_machines);
    return scheduler;

cleanup:
    // 清理资源
    if (scheduler->blocked_queue) free(scheduler->blocked_queue);
    if (scheduler->ready_queue) free(scheduler->ready_queue);
    for (uint32_t i = 0; i < num_machines; i++) {
        if (scheduler->machines && scheduler->machines[i]) {
            machine_destroy(scheduler->machines[i]);
        }
    }
    for (uint32_t i = 0; i < num_processors; i++) {
        if (scheduler->processors && scheduler->processors[i]) {
            processor_destroy(scheduler->processors[i]);
        }
    }
    if (scheduler->machines) free(scheduler->machines);
    if (scheduler->processors) free(scheduler->processors);
    free(scheduler);
    return NULL;
}

/**
 * @brief 销毁GMP调度器
 */
void scheduler_destroy(scheduler_t* scheduler) {
    if (!scheduler) return;

    // 停止调度器
    scheduler_stop(scheduler);

    // 清理同步机制
    pthread_cond_destroy(&scheduler->global_cond);
    pthread_mutex_destroy(&scheduler->global_mutex);

    // 清理GMP队列
    if (scheduler->ready_queue) {
        free(scheduler->ready_queue);
    }
    if (scheduler->blocked_queue) {
        free(scheduler->blocked_queue);
    }

    // 清理机器
    if (scheduler->machines) {
        for (uint32_t i = 0; i < scheduler->num_machines; i++) {
            if (scheduler->machines[i]) {
                machine_destroy(scheduler->machines[i]);
            }
        }
        free(scheduler->machines);
    }

    // 清理处理器
    if (scheduler->processors) {
        for (uint32_t i = 0; i < scheduler->num_processors; i++) {
            if (scheduler->processors[i]) {
                processor_destroy(scheduler->processors[i]);
            }
        }
        free(scheduler->processors);
    }

    printf("DEBUG: Destroyed GMP Scheduler %llu\n", scheduler->id);
    free(scheduler);
}

/**
 * @brief 启动调度器
 */
int scheduler_start(scheduler_t* scheduler) {
    if (!scheduler) return -1;

    if (scheduler->is_running) {
        return -1; // 已经在运行
    }

    printf("DEBUG: Starting GMP Scheduler %llu\n", scheduler->id);

    // 启动所有机器
    for (uint32_t i = 0; i < scheduler->num_machines; i++) {
        if (machine_start(scheduler->machines[i]) != 0) {
            fprintf(stderr, "Failed to start machine %u\n", i);
            // 停止已启动的机器
            for (uint32_t j = 0; j < i; j++) {
                machine_stop(scheduler->machines[j]);
            }
            return -1;
        }
    }

    scheduler->is_running = true;
    printf("DEBUG: GMP Scheduler %llu started\n", scheduler->id);
    return 0;
}

/**
 * @brief 停止调度器
 */
void scheduler_stop(scheduler_t* scheduler) {
    if (!scheduler || !scheduler->is_running) return;

    printf("DEBUG: Stopping GMP Scheduler %llu\n", scheduler->id);
    scheduler->is_running = false;

    // 停止所有机器
    for (uint32_t i = 0; i < scheduler->num_machines; i++) {
        machine_stop(scheduler->machines[i]);
    }

    // 等待所有机器停止
    for (uint32_t i = 0; i < scheduler->num_machines; i++) {
        machine_join(scheduler->machines[i]);
    }

    printf("DEBUG: GMP Scheduler %llu stopped\n", scheduler->id);
}

/**
 * @brief 调度任务
 * 使用简单的负载均衡策略
 */
int scheduler_schedule_task(scheduler_t* scheduler, task_t* task) {
    if (!scheduler || !task) return -1;

    // GMP负载均衡策略：优先本地处理器队列，全局就绪队列兜底
    bool scheduled = false;

    if (scheduler->num_processors > 0) {
        // 简化负载均衡：轮询到不同的处理器
        static uint32_t next_processor = 0;
        uint32_t processor_id = next_processor % scheduler->num_processors;
        next_processor++;

        struct Processor* processor = scheduler->processors[processor_id];
        if (processor_push_task(processor, task)) {
            scheduled = true;
            printf("DEBUG: Scheduled task %llu to processor %u local queue\n",
                   task_get_id(task), processor_id);
        }
    }

    // 如果本地队列满，放入全局就绪队列
    if (!scheduled) {
        if (scheduler_enqueue_ready(scheduler, task)) {
            printf("DEBUG: Scheduled task %llu to global ready queue\n",
                   task_get_id(task));
        } else {
            fprintf(stderr, "ERROR: Failed to schedule task %llu - queues full\n",
                    task_get_id(task));
            return -1;
        }
    }
    return 0;
}

/**
 * @brief 获取就绪任务
 */
task_t* scheduler_get_ready_task(scheduler_t* scheduler, struct Processor* processor) {
    if (!scheduler) return NULL;

    // 简化实现：暂时不考虑processor参数
    (void)processor;

    pthread_mutex_lock(&scheduler->global_mutex);

            if (scheduler->ready_queue_size == 0) {
        pthread_mutex_unlock(&scheduler->global_mutex);
        return NULL;
    }

    if (scheduler->ready_queue_size == 0) {
        pthread_mutex_unlock(&scheduler->global_mutex);
        return NULL;
    }

    task_t* task = scheduler->ready_queue[--scheduler->ready_queue_size];

    pthread_mutex_unlock(&scheduler->global_mutex);

    printf("DEBUG: Got ready task %llu, ready queue size: %zu\n",
           task_get_id(task), scheduler->ready_queue_size);
    return task;
}

/**
 * @brief 任务完成通知
 */
void scheduler_task_completed(scheduler_t* scheduler, task_t* task) {
    if (!scheduler || !task) return;

    scheduler->tasks_completed++;

    printf("DEBUG: Task %llu completed, total completed: %llu\n",
           task_get_id(task), scheduler->tasks_completed);
}

/**
 * @brief 执行负载均衡
 * 基于队列长度的负载均衡策略
 */
void scheduler_balance_load(scheduler_t* scheduler) {
    if (!scheduler || scheduler->num_processors < 2) return;

    // 计算平均队列长度
    size_t total_tasks = 0;
    size_t* queue_sizes = calloc(scheduler->num_processors, sizeof(size_t));

    if (!queue_sizes) return; // 内存分配失败

    // 收集所有处理器的队列大小
    for (uint32_t i = 0; i < scheduler->num_processors; i++) {
        queue_sizes[i] = processor_get_queue_size(scheduler->processors[i]);
        total_tasks += queue_sizes[i];
    }

    if (total_tasks == 0) {
        free(queue_sizes);
        return; // 没有任务需要平衡
    }

    size_t avg_tasks = total_tasks / scheduler->num_processors;
    size_t threshold = avg_tasks / 4; // 允许的偏差阈值

    // 找出负载过重的处理器（队列长度 > 平均值 + 阈值）
    for (uint32_t i = 0; i < scheduler->num_processors; i++) {
        if (queue_sizes[i] > avg_tasks + threshold) {
            // 从该处理器窃取任务
            size_t tasks_to_move = (queue_sizes[i] - avg_tasks) / 2;

            // 尝试移动到负载较轻的处理器
            for (uint32_t j = 0; j < scheduler->num_processors && tasks_to_move > 0; j++) {
                if (i == j) continue; // 不移动给自己

                size_t target_size = queue_sizes[j];
                if (target_size >= avg_tasks) continue; // 目标处理器负载已够

                size_t can_move = avg_tasks - target_size;
                if (can_move > tasks_to_move) {
                    can_move = tasks_to_move;
                }

                // 执行任务迁移
                for (size_t k = 0; k < can_move; k++) {
                    task_t* task = processor_pop_task(scheduler->processors[i]);
                    if (!task) break; // 没有更多任务

                    if (processor_push_task(scheduler->processors[j], task)) {
                        tasks_to_move--;
                        queue_sizes[i]--;
                        queue_sizes[j]++;
                    } else {
                        // 推送失败，重新放回原队列
                        processor_push_task(scheduler->processors[i], task);
                        break;
                    }
                }
            }
        }
    }

    free(queue_sizes);
}

/**
 * @brief 获取调度器统计信息
 */
void scheduler_get_stats(const scheduler_t* scheduler,
                        uint64_t* scheduled,
                        uint64_t* completed) {
    if (!scheduler) return;

    if (scheduled) *scheduled = scheduler->tasks_scheduled;
    if (completed) *completed = scheduler->tasks_completed;
}

// ============================================================================
// 上下文切换支持函数实现
// ============================================================================

/**
 * @brief 获取调度器上下文
 */
context_t* scheduler_get_context(scheduler_t* scheduler) {
    if (!scheduler) return NULL;
    return scheduler->scheduler_context;
}

/**
 * @brief 设置调度器上下文
 */
void scheduler_set_context(scheduler_t* scheduler, context_t* context) {
    if (!scheduler) return;
    scheduler->scheduler_context = context;
}

/**
 * @brief 选择下一个要执行的任务
 * GMP调度策略：本地队列 → 全局就绪队列 → 工作窃取
 */
task_t* scheduler_select_next_task(scheduler_t* scheduler) {
    if (!scheduler) return NULL;

    // 1. 检查本地处理器队列（GMP策略：优先本地队列）
    // 简化实现：假设当前在processor 0上运行
    if (scheduler->num_processors > 0) {
        struct Processor* processor = scheduler->processors[0];
        task_t* task = processor_pop_task(processor);
        if (task) {
            printf("DEBUG: Selected task from local processor queue\n");
            return task;
        }
    }

    // 2. 检查全局就绪队列
    task_t* task = scheduler_dequeue_ready(scheduler);
    if (task) {
        printf("DEBUG: Selected task from global ready queue\n");
        return task;
    }

    // 3. 工作窃取：从其他处理器的队列窃取任务
    if (scheduler->num_processors > 1) {
        for (uint32_t i = 0; i < scheduler->num_processors; i++) {
            struct Processor* thief_processor = scheduler->processors[i];
            task_t* stolen_task = processor_steal_task(thief_processor);
            if (stolen_task) {
                printf("DEBUG: Stolen task from processor %u\n", i);
                return stolen_task;
            }
        }
    }

    return NULL; // 没有可执行的任务
}

/**
 * @brief 调度器主循环
 * 核心调度逻辑：不断选择任务并执行上下文切换
 */
void scheduler_main_loop(scheduler_t* scheduler) {
    if (!scheduler) return;

    printf("DEBUG: Scheduler main loop started\n");

    while (scheduler->is_running) {
        // GMP调度循环：选择任务并执行上下文切换
        task_t* next_task = scheduler_select_next_task(scheduler);

        if (next_task) {
            printf("DEBUG: GMP: Switching to task %llu\n", task_get_id(next_task));

            // 执行任务：通过协程上下文切换
            coroutine_t* coroutine = task_get_coroutine(next_task);
            if (coroutine) {
                context_t* task_context = coroutine_get_context(coroutine);

                if (task_context && scheduler->scheduler_context) {
                    // GMP核心：上下文切换到任务协程
                    printf("DEBUG: GMP: Context switch scheduler -> task %llu\n",
                           task_get_id(next_task));
                    context_switch(scheduler->scheduler_context, task_context);

                    // 任务执行完成后回到这里
                    printf("DEBUG: GMP: Task %llu completed, back to scheduler\n",
                           task_get_id(next_task));

                    // 任务完成统计
                    scheduler->tasks_completed++;
                } else {
                    fprintf(stderr, "ERROR: Invalid contexts for task %llu\n",
                            task_get_id(next_task));
                }
            } else {
                fprintf(stderr, "ERROR: Task %llu has no coroutine\n",
                        task_get_id(next_task));
            }
        } else {
            // GMP空闲策略：等待新任务或执行工作窃取
            pthread_mutex_lock(&scheduler->global_mutex);
            if (scheduler->ready_queue_size == 0) {
                printf("DEBUG: GMP: No ready tasks, waiting...\n");
                pthread_cond_wait(&scheduler->global_cond, &scheduler->global_mutex);
            }
            pthread_mutex_unlock(&scheduler->global_mutex);

            // 可以在这里添加工作窃取或GC等空闲任务
        }
    }

    printf("DEBUG: Scheduler main loop ended\n");
}

// ============================================================================
// GMP队列管理函数实现
// ============================================================================

/**
 * @brief 将任务加入就绪队列
 */
bool scheduler_enqueue_ready(scheduler_t* scheduler, task_t* task) {
    if (!scheduler || !task) return false;

    pthread_mutex_lock(&scheduler->global_mutex);

    if (scheduler->ready_queue_size >= scheduler->ready_queue_capacity) {
        pthread_mutex_unlock(&scheduler->global_mutex);
        return false; // 队列满
    }

    scheduler->ready_queue[scheduler->ready_queue_size++] = task;
    scheduler->tasks_scheduled++;

    // 唤醒等待的线程
    pthread_cond_broadcast(&scheduler->global_cond);

    pthread_mutex_unlock(&scheduler->global_mutex);

    printf("DEBUG: Enqueued task %llu to ready queue, size: %zu\n",
           task_get_id(task), scheduler->ready_queue_size);
    return true;
}

/**
 * @brief 从就绪队列取出任务
 */
task_t* scheduler_dequeue_ready(scheduler_t* scheduler) {
    if (!scheduler) return NULL;

    pthread_mutex_lock(&scheduler->global_mutex);

    if (scheduler->ready_queue_size == 0) {
        pthread_mutex_unlock(&scheduler->global_mutex);
        return NULL;
    }

    task_t* task = scheduler->ready_queue[--scheduler->ready_queue_size];

    pthread_mutex_unlock(&scheduler->global_mutex);

    printf("DEBUG: Dequeued task %llu from ready queue, size: %zu\n",
           task_get_id(task), scheduler->ready_queue_size);
    return task;
}

/**
 * @brief 将任务加入阻塞队列
 */
bool scheduler_enqueue_blocked(scheduler_t* scheduler, task_t* task) {
    if (!scheduler || !task) return false;

    pthread_mutex_lock(&scheduler->global_mutex);

    if (scheduler->blocked_queue_size >= scheduler->blocked_queue_capacity) {
        pthread_mutex_unlock(&scheduler->global_mutex);
        return false; // 队列满
    }

    scheduler->blocked_queue[scheduler->blocked_queue_size++] = task;

    pthread_mutex_unlock(&scheduler->global_mutex);

    printf("DEBUG: Enqueued task %llu to blocked queue, size: %zu\n",
           task_get_id(task), scheduler->blocked_queue_size);
    return true;
}

/**
 * @brief 将任务从阻塞队列移动到就绪队列
 */
bool scheduler_unblock_task(scheduler_t* scheduler, task_t* task) {
    if (!scheduler || !task) return false;

    pthread_mutex_lock(&scheduler->global_mutex);

    // 从阻塞队列中移除任务
    bool found = false;
    for (size_t i = 0; i < scheduler->blocked_queue_size; i++) {
        if (scheduler->blocked_queue[i] == task) {
            // 移动到就绪队列
            if (scheduler->ready_queue_size < scheduler->ready_queue_capacity) {
                scheduler->ready_queue[scheduler->ready_queue_size++] = task;
                // 从阻塞队列移除
                for (size_t j = i; j < scheduler->blocked_queue_size - 1; j++) {
                    scheduler->blocked_queue[j] = scheduler->blocked_queue[j + 1];
                }
                scheduler->blocked_queue_size--;
                found = true;

                // 唤醒等待的线程
                pthread_cond_broadcast(&scheduler->global_cond);
            }
            break;
        }
    }

    pthread_mutex_unlock(&scheduler->global_mutex);

    if (found) {
        printf("DEBUG: Unblocked task %llu, moved to ready queue\n", task_get_id(task));
    }
    return found;
}
