/**
 * @file future.h
 * @brief Future 聚合根和 Waker 接口定义
 *
 * Future 代表异步计算的结果，Waker 是唤醒任务的机制。
 */

#ifndef ECHO_FUTURE_H
#define ECHO_FUTURE_H

#include "runtime.h"

// 前向声明
struct Task;

// ============================================================================
// Waker 值对象定义
// ============================================================================

/**
 * @brief Waker 值对象结构体
 * Waker 是轻量级的任务唤醒器，包含任务引用
 */
typedef struct Waker {
    task_id_t task_id;        // 任务ID（避免直接指针引用）
    void* scheduler_ref;      // 调度器引用（用于唤醒）
} waker_t;

/**
 * @brief Waker 行为接口
 */
typedef struct waker_vtable {
    void (*wake)(const waker_t* waker);     // 唤醒任务
    void (*drop)(const waker_t* waker);     // 释放资源
} waker_vtable_t;

// ============================================================================
// Future 聚合根定义
// ============================================================================

/**
 * @brief Future 聚合根结构体
 */
typedef struct Future {
    future_id_t id;           // Future唯一标识
    future_state_t state;     // Future状态
    void* result;             // 结果或错误
    bool consumed;            // 是否已被消费
    waker_t* waker;           // 关联的唤醒器
    void* user_data;          // 用户数据
} future_t;

/**
 * @brief Future 轮询结果
 */
typedef struct poll_result {
    bool is_ready;    // 是否就绪
    void* value;      // 结果值
} poll_result_t;

// ============================================================================
// Future 聚合根行为接口
// ============================================================================

/**
 * @brief 创建新Future
 * @return 新创建的Future指针
 */
future_t* future_create(void);

/**
 * @brief 销毁Future
 * @param future 要销毁的Future
 */
void future_destroy(future_t* future);

/**
 * @brief 轮询Future状态
 * @param future 要轮询的Future
 * @param task 当前任务（用于Waker）
 * @return 轮询结果
 */
poll_result_t future_poll(future_t* future, struct Task* task);

/**
 * @brief 解决Future（设置成功结果）
 * @param future 要解决的Future
 * @param value 结果值
 */
void future_resolve(future_t* future, void* value);

/**
 * @brief 拒绝Future（设置错误结果）
 * @param future 要拒绝的Future
 * @param error 错误信息
 */
void future_reject(future_t* future, void* error);

/**
 * @brief 取消Future
 * @param future 要取消的Future
 */
void future_cancel(future_t* future);

/**
 * @brief 获取Future ID
 * @param future Future
 * @return Future ID
 */
future_id_t future_get_id(const future_t* future);

/**
 * @brief 获取Future状态
 * @param future Future
 * @return Future状态
 */
future_state_t future_get_state(const future_t* future);

/**
 * @brief 检查Future是否就绪
 * @param future Future
 * @return true如果Future已完成
 */
bool future_is_ready(const future_t* future);

// ============================================================================
// Waker 工具函数
// ============================================================================

/**
 * @brief 创建Waker
 * @param task 关联的任务
 * @param scheduler_ref 调度器引用
 * @return 新创建的Waker
 */
waker_t* waker_create(struct Task* task, void* scheduler_ref);

/**
 * @brief 销毁Waker
 * @param waker 要销毁的Waker
 */
void waker_destroy(waker_t* waker);

/**
 * @brief 唤醒任务
 * @param waker Waker
 */
void waker_wake(const waker_t* waker);

/**
 * @brief 释放Waker资源
 * @param waker Waker
 */
void waker_drop(const waker_t* waker);

#endif // ECHO_FUTURE_H

