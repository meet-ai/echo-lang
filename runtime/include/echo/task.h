/**
 * @file task.h
 * @brief Task 聚合根定义
 *
 * Task 是异步运行时的核心执行单元，封装了协程和异步状态。
 */

#ifndef ECHO_TASK_H
#define ECHO_TASK_H

#include "runtime.h"

// 前向声明context_t（定义在内部模块中）
typedef struct Context context_t;

// 前向声明
struct Future;
struct Coroutine;

// ============================================================================
// Task 聚合根定义
// ============================================================================

/**
 * @brief Task 聚合根结构体
 */
typedef struct Task {
    task_id_t id;                  // 任务唯一标识
    task_status_t status;         // 任务状态
    struct Future* future;         // 当前等待的Future
    void* result;                  // 执行结果
    void* user_data;               // 用户数据

    // 协程支持
    struct Coroutine* coroutine;   // 协程实现
} task_t;

// ============================================================================
// Task 聚合根行为接口
// ============================================================================

/**
 * @brief 创建新任务
 * @param entry_point 任务入口函数
 * @param arg 函数参数
 * @param stack_size 栈大小
 * @return 新创建的任务指针
 */
task_t* task_create(void (*entry_point)(void*), void* arg, size_t stack_size);

/**
 * @brief 销毁任务
 * @param task 要销毁的任务
 */
void task_destroy(task_t* task);

/**
 * @brief 执行任务
 * @param task 要执行的任务
 * @param scheduler_context 调度器上下文（用于协程完成时切换回）
 */
void task_execute(task_t* task, context_t* scheduler_context);

/**
 * @brief 挂起任务
 * @param task 要挂起的任务
 */
void task_suspend(task_t* task);

/**
 * @brief 恢复任务执行
 * @param task 要恢复的任务
 */
void task_resume(task_t* task);

/**
 * @brief 设置任务等待的Future
 * @param task 任务
 * @param future 要等待的Future
 */
void task_set_future(task_t* task, struct Future* future);

/**
 * @brief 获取任务ID
 * @param task 任务
 * @return 任务ID
 */
task_id_t task_get_id(const task_t* task);

/**
 * @brief 获取任务状态
 * @param task 任务
 * @return 任务状态
 */
task_status_t task_get_status(const task_t* task);

/**
 * @brief 检查任务是否完成
 * @param task 任务
 * @return true如果任务已完成
 */
bool task_is_completed(const task_t* task);

/**
 * @brief 获取任务的协程
 * @param task 任务
 * @return 任务的协程
 */
struct Coroutine* task_get_coroutine(const task_t* task);

// ============================================================================
// 任务状态管理函数
// ============================================================================

/**
 * @brief 任务状态转换：准备就绪
 * @param task 任务
 * @return true如果转换成功
 */
bool task_mark_ready(task_t* task);

/**
 * @brief 任务状态转换：开始执行
 * @param task 任务
 * @return true如果转换成功
 */
bool task_mark_running(task_t* task);

/**
 * @brief 任务状态转换：等待异步操作
 * @param task 任务
 * @return true如果转换成功
 */
bool task_mark_waiting(task_t* task);

/**
 * @brief 任务状态转换：执行完成
 * @param task 任务
 * @return true如果转换成功
 */
bool task_mark_completed(task_t* task);

/**
 * @brief 任务状态转换：执行失败
 * @param task 任务
 * @return true如果转换成功
 */
bool task_mark_failed(task_t* task);

/**
 * @brief 任务状态转换：被取消
 * @param task 任务
 * @return true如果转换成功
 */
bool task_mark_cancelled(task_t* task);

/**
 * @brief 检查任务是否可以被调度
 * @param task 任务
 * @return true如果可以调度
 */
bool task_can_schedule(const task_t* task);

/**
 * @brief 检查任务是否处于等待状态
 * @param task 任务
 * @return true如果正在等待
 */
bool task_is_waiting(const task_t* task);

/**
 * @brief 检查任务是否已终结
 * @param task 任务
 * @return true如果已终结
 */
bool task_is_terminated(const task_t* task);

#endif // ECHO_TASK_H
