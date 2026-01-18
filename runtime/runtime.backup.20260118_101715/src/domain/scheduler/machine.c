/**
 * @file machine.c
 * @brief Machine (M) 实体实现
 *
 * Machine 管理操作系统线程，执行处理器调度循环。
 */

#include "machine.h"
#include "processor.h"
#include "scheduler.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>

// 线程函数参数
typedef struct ThreadArgs {
    machine_t* machine;
} thread_args_t;

// ============================================================================
// 线程函数
// ============================================================================

/**
 * @brief 机器线程的主函数
 * 执行处理器调度循环
 */
static void* machine_thread_func(void* arg) {
    thread_args_t* thread_args = (thread_args_t*)arg;
    machine_t* machine = thread_args->machine;
    free(thread_args);

    printf("DEBUG: Machine %u thread started\n", machine->id);

    // 线程主循环
    while (true) {
        pthread_mutex_lock(&machine->state_mutex);

        // 检查是否应该停止
        if (machine->should_stop) {
            pthread_mutex_unlock(&machine->state_mutex);
            break;
        }

        // 设置运行状态
        machine->is_running = true;

        // 获取绑定的处理器
        struct Processor* processor = machine->bound_processor;

        pthread_mutex_unlock(&machine->state_mutex);

        if (processor) {
            // 执行处理器调度循环
            while (!machine->should_stop) {
                // 检查是否应该停止（再次确认）
                if (machine->should_stop) {
                    printf("DEBUG: Machine %u detected stop signal, exiting loop\n", machine->id);
                    break;
                }

                // 首先尝试从本地队列获取任务
                task_t* task = processor_pop_task(processor);
                if (task) {
                    // 执行任务
                    processor_execute_task(processor, task);
                    machine->tasks_processed++;

                    // 任务执行完毕，可以清理
                    task_destroy(task);
                } else {
                    // 本地队列为空，尝试从全局队列获取任务
                    task = scheduler_get_ready_task(machine->bound_processor->scheduler, machine->bound_processor);
                    if (task) {
                        // 将任务推入本地队列，然后在下次循环中执行
                        if (processor_push_task(machine->bound_processor, task)) {
                            printf("DEBUG: Machine %u got task %llu from global queue\n",
                                   machine->id, task_get_id(task));
                        }
                    } else {
                        // 没有任务，尝试工作窃取
                        task = processor_steal_task(machine->bound_processor);
                        if (task) {
                            printf("DEBUG: Machine %u stole task %llu\n",
                                   machine->id, task_get_id(task));
                            // 将窃取的任务推入本地队列
                            processor_push_task(machine->bound_processor, task);
                        } else {
                            // 仍然没有任务，短暂休眠避免忙等待
                            // 在休眠前再次检查停止标志
                            if (!machine->should_stop) {
                                usleep(10000); // 10ms
                            }
                        }
                    }
                }
            }
        } else {
            // 没有绑定的处理器，休眠等待
            usleep(10000); // 10ms
        }
    }

    pthread_mutex_lock(&machine->state_mutex);
    machine->is_running = false;
    pthread_cond_signal(&machine->state_cond); // 唤醒等待的线程
    pthread_mutex_unlock(&machine->state_mutex);

    printf("DEBUG: Machine %u thread stopped\n", machine->id);
    return NULL;
}

// ============================================================================
// Machine 实体实现
// ============================================================================

/**
 * @brief 创建机器
 */
machine_t* machine_create(uint32_t id) {
    machine_t* machine = (machine_t*)malloc(sizeof(machine_t));
    if (!machine) {
        fprintf(stderr, "Failed to allocate machine\n");
        return NULL;
    }

    memset(machine, 0, sizeof(machine_t));

    machine->id = id;
    machine->bound_processor = NULL;
    machine->is_running = false;
    machine->should_stop = false;
    machine->tasks_processed = 0;

    // 初始化同步机制
    if (pthread_mutex_init(&machine->state_mutex, NULL) != 0) {
        fprintf(stderr, "Failed to initialize state mutex\n");
        free(machine);
        return NULL;
    }

    if (pthread_cond_init(&machine->state_cond, NULL) != 0) {
        fprintf(stderr, "Failed to initialize state condition\n");
        pthread_mutex_destroy(&machine->state_mutex);
        free(machine);
        return NULL;
    }

    printf("DEBUG: Created Machine %u\n", id);
    return machine;
}

/**
 * @brief 销毁机器
 */
void machine_destroy(machine_t* machine) {
    if (!machine) return;

    // 确保线程已停止
    machine_stop(machine);
    machine_join(machine);

    // 清理同步机制
    pthread_cond_destroy(&machine->state_cond);
    pthread_mutex_destroy(&machine->state_mutex);

    printf("DEBUG: Destroyed Machine %u\n", machine->id);
    free(machine);
}

/**
 * @brief 绑定处理器
 */
void machine_bind_processor(machine_t* machine, struct Processor* processor) {
    if (machine) {
        machine->bound_processor = processor;
        printf("DEBUG: Machine %u bound to processor\n", machine->id);
    }
}

/**
 * @brief 启动机器线程
 */
int machine_start(machine_t* machine) {
    if (!machine) return -1;

    pthread_mutex_lock(&machine->state_mutex);

    if (machine->is_running) {
        pthread_mutex_unlock(&machine->state_mutex);
        return -1; // 已经在运行
    }

    // 准备线程参数
    thread_args_t* args = (thread_args_t*)malloc(sizeof(thread_args_t));
    if (!args) {
        pthread_mutex_unlock(&machine->state_mutex);
        return -1;
    }
    args->machine = machine;

    // 创建线程
    int result = pthread_create(&machine->thread, NULL, machine_thread_func, args);
    if (result != 0) {
        fprintf(stderr, "Failed to create machine thread: %d\n", result);
        free(args);
        pthread_mutex_unlock(&machine->state_mutex);
        return -1;
    }

    pthread_mutex_unlock(&machine->state_mutex);
    printf("DEBUG: Started Machine %u thread\n", machine->id);
    return 0;
}

/**
 * @brief 停止机器线程
 */
void machine_stop(machine_t* machine) {
    if (!machine) return;

    pthread_mutex_lock(&machine->state_mutex);
    machine->should_stop = true;
    pthread_cond_signal(&machine->state_cond); // 唤醒线程
    pthread_mutex_unlock(&machine->state_mutex);

    printf("DEBUG: Requested Machine %u to stop\n", machine->id);
}

/**
 * @brief 等待机器线程结束
 */
int machine_join(machine_t* machine) {
    if (!machine) return -1;

    int result = pthread_join(machine->thread, NULL);
    if (result != 0) {
        fprintf(stderr, "Failed to join machine thread: %d\n", result);
        return -1;
    }

    printf("DEBUG: Joined Machine %u thread\n", machine->id);
    return 0;
}

/**
 * @brief 检查机器是否正在运行
 */
bool machine_is_running(const machine_t* machine) {
    if (!machine) return false;

    pthread_mutex_lock((pthread_mutex_t*)&machine->state_mutex);
    bool running = machine->is_running;
    pthread_mutex_unlock((pthread_mutex_t*)&machine->state_mutex);

    return running;
}

/**
 * @brief 获取机器统计信息
 */
void machine_get_stats(const machine_t* machine, uint64_t* processed) {
    if (!machine || !processed) return;

    *processed = machine->tasks_processed;
}
