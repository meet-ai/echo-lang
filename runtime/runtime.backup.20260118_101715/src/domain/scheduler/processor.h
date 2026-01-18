/**
 * @file processor.h
 * @brief Processor (P) 聚合根定义
 *
 * Processor 是GMP模型中的P，负责管理本地任务队列和执行协程。
 */

#ifndef PROCESSOR_H
#define PROCESSOR_H

#include "../../../include/echo/runtime.h"
#include "../../../include/echo/task.h"
#include <pthread.h>

// 前向声明
struct Machine;

// ============================================================================
// Processor 聚合根定义
// ============================================================================

/**
 * @brief Processor 聚合根结构体
 * GMP模型中的P：逻辑处理器，管理本地任务队列
 */
typedef struct Processor {
    uint32_t id;                    // 处理器ID
    struct Machine* bound_machine;  // 绑定的机器(M)
    struct Scheduler* scheduler;    // 所属调度器

    // 本地任务队列（双端队列，支持LIFO和FIFO）
    task_t** local_queue;           // 任务队列
    size_t queue_size;              // 当前队列大小
    size_t queue_capacity;          // 队列容量
    size_t queue_head;              // 队列头索引
    size_t queue_tail;              // 队列尾索引

    // 统计信息
    uint64_t tasks_executed;        // 已执行任务数
    uint64_t tasks_stolen;          // 被窃取的任务数
    uint64_t steal_attempts;        // 窃取尝试次数

    // 同步机制
    pthread_mutex_t queue_mutex;    // 队列锁
    pthread_cond_t queue_cond;      // 队列条件变量
} processor_t;

// ============================================================================
// Processor 聚合根行为接口
// ============================================================================

/**
 * @brief 创建处理器
 * @param id 处理器ID
 * @param queue_capacity 本地队列容量
 * @return 新创建的处理器
 */
processor_t* processor_create(uint32_t id, size_t queue_capacity);

/**
 * @brief 销毁处理器
 * @param processor 要销毁的处理器
 */
void processor_destroy(processor_t* processor);

/**
 * @brief 绑定到机器
 * @param processor 处理器
 * @param machine 要绑定的机器
 */
void processor_bind_machine(processor_t* processor, struct Machine* machine);

/**
 * @brief 设置调度器引用
 * @param processor 处理器
 * @param scheduler 调度器
 */
void processor_set_scheduler(processor_t* processor, struct Scheduler* scheduler);

/**
 * @brief 推入任务到本地队列（LIFO）
 * @param processor 处理器
 * @param task 要推入的任务
 * @return true成功，false失败（队列满）
 */
bool processor_push_task(processor_t* processor, task_t* task);

/**
 * @brief 从本地队列弹出任务（LIFO）
 * @param processor 处理器
 * @return 弹出的任务，NULL表示队列空
 */
task_t* processor_pop_task(processor_t* processor);

/**
 * @brief 从其他处理器窃取任务（FIFO）
 * @param thief_processor 窃取者处理器
 * @return 窃取到的任务，NULL表示没有窃取到
 */
task_t* processor_steal_task(processor_t* thief_processor);

/**
 * @brief 执行一个任务
 * @param processor 处理器
 * @param task 要执行的任务
 */
void processor_execute_task(processor_t* processor, task_t* task);

/**
 * @brief 获取队列大小
 * @param processor 处理器
 * @return 队列中的任务数量
 */
size_t processor_get_queue_size(const processor_t* processor);

/**
 * @brief 检查队列是否为空
 * @param processor 处理器
 * @return true如果队列为空
 */
bool processor_queue_empty(const processor_t* processor);

/**
 * @brief 获取处理器统计信息
 * @param processor 处理器
 * @param executed 执行的任务数
 * @param stolen 被窃取的任务数
 * @param attempts 窃取尝试次数
 */
void processor_get_stats(const processor_t* processor,
                        uint64_t* executed,
                        uint64_t* stolen,
                        uint64_t* attempts);

#endif // PROCESSOR_H
