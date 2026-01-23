/**
 * @file timer_future.h
 * @brief TimerFuture 示例实现
 *
 * TimerFuture 是最简单的异步操作示例，用于演示异步闭环。
 */

#ifndef TIMER_FUTURE_H
#define TIMER_FUTURE_H

#include "../../../include/echo/future.h"

// 前向声明
struct Reactor;

// ============================================================================
// TimerFuture 定义
// ============================================================================

/**
 * @brief TimerFuture 结构体
 * 继承自Future，添加定时器特定状态
 */
typedef struct TimerFuture {
    future_t base_future;     // 继承Future
    struct timespec deadline; // 截止时间
    bool registered;          // 是否已注册到Reactor
} timer_future_t;

// ============================================================================
// TimerFuture 接口
// ============================================================================

/**
 * @brief 创建TimerFuture
 * @param milliseconds 等待毫秒数
 * @return 新创建的TimerFuture
 */
timer_future_t* timer_future_create(int milliseconds);

/**
 * @brief 销毁TimerFuture
 * @param timer_future 要销毁的TimerFuture
 */
void timer_future_destroy(timer_future_t* timer_future);

/**
 * @brief 等待TimerFuture完成
 * @param timer_future TimerFuture
 * @return 结果值
 */
void* timer_future_await(timer_future_t* timer_future);

/**
 * @brief 设置TimerFuture使用的Reactor
 * @param reactor Reactor实例
 */
void timer_future_set_reactor(struct Reactor* reactor);

#endif // TIMER_FUTURE_H
