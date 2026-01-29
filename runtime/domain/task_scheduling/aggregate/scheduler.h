/**
 * @file scheduler.h
 * @brief TaskScheduling 聚合根定义
 *
 * Scheduler聚合负责任务调度，包括：
 * - 任务队列管理（全局队列和本地队列）
 * - 工作窃取（Work Stealing）
 * - 处理器和机器的管理
 *
 * 聚合边界：
 * - Processor和Machine是内部实体，只能通过Scheduler聚合根访问
 * - 任务队列存储TaskID列表，不直接存储Task指针
 * - 通过TaskRepository查找Task聚合
 *
 * 使用场景：
 * ```c
 * // 创建调度器
 * Scheduler* scheduler = scheduler_factory_create(num_processors, task_repo);
 *
 * // 添加任务（通过TaskID）
 * scheduler_add_task(scheduler, task_id);
 *
 * // 获取任务（返回TaskID）
 * TaskID task_id = scheduler_get_work(scheduler, processor);
 * Task* task = task_repository_find_by_id(task_repo, task_id);
 * ```
 */

#ifndef TASK_SCHEDULING_AGGREGATE_SCHEDULER_H
#define TASK_SCHEDULING_AGGREGATE_SCHEDULER_H

#include "../../task_execution/aggregate/task.h"  // TaskID定义
#include <stdint.h>
#include <stdbool.h>
#include <pthread.h>

// ============================================================================
// 前向声明
// ============================================================================

// 内部实体：Processor（处理器）
struct Processor;
typedef struct Processor Processor;

// 内部实体：Machine（OS线程）
struct Machine;
typedef struct Machine Machine;

// TaskRepository接口（用于查找Task聚合）
struct TaskRepository;
typedef struct TaskRepository TaskRepository;

// ============================================================================
// 聚合根定义
// ============================================================================

// TaskIDNode定义在task.h中（共享值对象）

/**
 * @brief Scheduler聚合根
 *
 * 职责：
 * - 管理任务队列（全局队列和本地队列）
 * - 协调处理器和机器
 * - 实现工作窃取算法
 *
 * 不变条件：
 * - 全局队列中的TaskID必须有效（通过TaskRepository验证）
 * - 每个处理器的本地队列大小不能超过限制
 * - 调度器运行状态必须与处理器状态一致
 */
typedef struct Scheduler {
    // 聚合根标识
    uint32_t id;                    // 调度器ID

    // 内部实体（只能通过聚合根访问）
    Processor** processors;         // 处理器数组（内部实体）
    uint32_t num_processors;        // 处理器数量
    Machine** machines;             // 机器(OS线程)数组（内部实体）
    uint32_t num_machines;          // 机器数量

    // 任务队列（存储TaskID列表）
    TaskIDNode* global_queue;       // 全局任务队列（TaskID列表）
    pthread_mutex_t global_lock;   // 全局队列锁

    // 聚合状态
    bool is_running;                // 调度器是否运行中

    // 统计信息
    uint64_t tasks_scheduled;       // 已调度任务数
    uint64_t tasks_completed;       // 已完成任务数

    // 领域事件（待发布）
    void* domain_events;            // 领域事件列表（TODO: 定义具体类型）

    // 依赖注入（通过构造函数传入）
    TaskRepository* task_repository; // Task仓储（用于查找Task聚合）
    struct EventBus* event_bus;      // 事件总线（可选，如果为NULL则只存储到domain_events列表）
} Scheduler;

// ============================================================================
// 工厂方法
// ============================================================================

/**
 * @brief 创建Scheduler聚合根
 *
 * @param num_processors 处理器数量
 * @param task_repository Task仓储（用于查找Task聚合）
 * @param event_bus 事件总线（可选，如果为NULL则只存储到domain_events列表）
 * @return Scheduler* 创建的调度器，失败返回NULL
 */
Scheduler* scheduler_factory_create(uint32_t num_processors, TaskRepository* task_repository, struct EventBus* event_bus);

/**
 * @brief 销毁Scheduler聚合根
 *
 * @param scheduler 调度器实例
 * 
 * 注意：为了避免与适配层函数名冲突，使用scheduler_aggregate_destroy
 */
void scheduler_aggregate_destroy(Scheduler* scheduler);

// ============================================================================
// 聚合根方法（业务操作）
// ============================================================================

/**
 * @brief 添加任务到调度器（聚合根方法）
 *
 * @param scheduler 调度器实例
 * @param task_id 任务ID
 * @return int 成功返回0，失败返回-1
 */
int scheduler_aggregate_add_task(Scheduler* scheduler, TaskID task_id);

/**
 * @brief 从调度器获取任务（供处理器使用）（聚合根方法）
 *
 * @param scheduler 调度器实例
 * @param processor 处理器实例
 * @param task_id 输出参数，返回的任务ID
 * @return int 成功返回0，失败返回-1
 */
int scheduler_aggregate_get_work(Scheduler* scheduler, Processor* processor, TaskID* task_id);

/**
 * @brief 工作窃取（一个处理器尝试从其他处理器窃取任务）（聚合根方法）
 *
 * @param scheduler 调度器实例
 * @param thief 窃取者处理器
 * @return int 成功返回0，失败返回-1
 */
int scheduler_aggregate_steal_work(Scheduler* scheduler, Processor* thief);

/**
 * @brief 启动调度器（聚合根方法）
 *
 * @param scheduler 调度器实例
 * @return int 成功返回0，失败返回-1
 */
int scheduler_aggregate_start(Scheduler* scheduler);

/**
 * @brief 停止调度器（聚合根方法）
 *
 * @param scheduler 调度器实例
 * @return int 成功返回0，失败返回-1
 */
int scheduler_aggregate_stop(Scheduler* scheduler);

/**
 * @brief 通知任务完成（聚合根方法）
 *
 * 当任务完成时调用此方法，更新统计信息并发布TaskCompleted事件。
 *
 * @param scheduler 调度器实例
 * @param task_id 已完成的任务ID
 * @return int 成功返回0，失败返回-1
 */
int scheduler_aggregate_notify_task_completed(Scheduler* scheduler, TaskID task_id);

// ============================================================================
// 查询方法
// ============================================================================

/**
 * @brief 获取调度器ID
 */
uint32_t scheduler_get_id(const Scheduler* scheduler);

/**
 * @brief 检查调度器是否运行中
 */
bool scheduler_is_running(const Scheduler* scheduler);

/**
 * @brief 获取已调度任务数
 */
uint64_t scheduler_get_tasks_scheduled(const Scheduler* scheduler);

/**
 * @brief 检查全局队列是否有工作
 */
bool scheduler_has_work_in_global_queue(Scheduler* scheduler);

/**
 * @brief 打印调度器统计信息
 */
void scheduler_print_stats(Scheduler* scheduler);

/**
 * @brief 获取已完成任务数
 */
uint64_t scheduler_get_tasks_completed(const Scheduler* scheduler);

// ============================================================================
// 不变条件验证
// ============================================================================

/**
 * @brief 验证聚合不变条件
 *
 * @param scheduler 调度器实例
 * @return int 验证通过返回0，失败返回-1
 */
int scheduler_validate_invariants(const Scheduler* scheduler);

// ============================================================================
// 领域事件管理
// ============================================================================

/**
 * @brief 获取并清空领域事件列表
 *
 * @param scheduler 调度器实例
 * @return void* 领域事件列表（TODO: 定义具体类型）
 */
void* scheduler_get_domain_events(Scheduler* scheduler);

// ============================================================================
// 内部实体访问（仅用于实现，不对外暴露）
// ============================================================================

// 注意：Processor和Machine的内部方法不在此处声明
// 它们只能通过Scheduler聚合根的方法间接访问

#endif // TASK_SCHEDULING_AGGREGATE_SCHEDULER_H
