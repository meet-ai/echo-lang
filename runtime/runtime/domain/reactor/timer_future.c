/**
 * @file timer_future.c
 * @brief TimerFuture 示例实现
 *
 * TimerFuture 演示了如何创建一个简单的异步Future。
 */

#include "timer_future.h"
#include <stdlib.h>
#include <stdio.h>
#include <time.h>

// 全局Reactor引用（简化实现，实际应该通过依赖注入）
static struct Reactor* global_reactor = NULL;

/**
 * @brief 设置全局Reactor引用
 */
void timer_future_set_reactor(struct Reactor* reactor) {
    global_reactor = reactor;
}

// ============================================================================
// TimerFuture 实现
// ============================================================================

/**
 * @brief 创建TimerFuture
 */
timer_future_t* timer_future_create(int milliseconds) {
    timer_future_t* timer_future = (timer_future_t*)malloc(sizeof(timer_future_t));
    if (!timer_future) {
        fprintf(stderr, "Failed to allocate TimerFuture\n");
        return NULL;
    }

    // 初始化基础Future
    timer_future->base_future.id = 1; // 简化ID分配
    timer_future->base_future.state = FUTURE_STATE_PENDING;
    timer_future->base_future.result = NULL;
    timer_future->base_future.consumed = false;
    timer_future->base_future.waker = NULL;
    timer_future->base_future.user_data = timer_future; // 指向自身

    // 设置截止时间
    clock_gettime(CLOCK_MONOTONIC, &timer_future->deadline);
    timer_future->deadline.tv_sec += milliseconds / 1000;
    timer_future->deadline.tv_nsec += (milliseconds % 1000) * 1000000;

    // 标准化时间
    if (timer_future->deadline.tv_nsec >= 1000000000) {
        timer_future->deadline.tv_sec += 1;
        timer_future->deadline.tv_nsec -= 1000000000;
    }

    timer_future->registered = false;

    printf("DEBUG: Created TimerFuture for %d ms\n", milliseconds);
    return timer_future;
}

/**
 * @brief 销毁TimerFuture
 */
void timer_future_destroy(timer_future_t* timer_future) {
    if (timer_future) {
        // 清理Waker
        if (timer_future->base_future.waker) {
            waker_destroy(timer_future->base_future.waker);
        }
        free(timer_future);
        printf("DEBUG: Destroyed TimerFuture\n");
    }
}

/**
 * @brief TimerFuture的poll实现
 * 核心业务逻辑：检查定时器是否到期
 */
static poll_result_t timer_future_poll(timer_future_t* timer_future, struct Task* task) {
    poll_result_t result = { .is_ready = false, .value = NULL };

    // 检查当前时间是否已到达截止时间
    struct timespec now;
    clock_gettime(CLOCK_MONOTONIC, &now);

    if (now.tv_sec > timer_future->deadline.tv_sec ||
        (now.tv_sec == timer_future->deadline.tv_sec && now.tv_nsec >= timer_future->deadline.tv_nsec)) {

        // 定时器到期 - 解决Future
        future_resolve((future_t*)timer_future, (void*)"timeout");
        result.is_ready = true;
        result.value = (void*)"timeout"; // 简单返回值
        printf("DEBUG: TimerFuture expired\n");

    } else {
        // 定时器未到期，需要注册到Reactor
        if (!timer_future->registered && global_reactor) {
            // 创建Waker
            if (!timer_future->base_future.waker) {
                timer_future->base_future.waker = waker_create(task, NULL); // 简化实现
            }

            // 注册到Reactor
            // TODO: 实现reactor_register_future
            // reactor_register_future(global_reactor, (struct Future*)timer_future, timer_future->base_future.waker);
            timer_future->registered = true;

            printf("DEBUG: TimerFuture registered to Reactor, waiting...\n");
        }
    }

    return result;
}

/**
 * @brief 等待TimerFuture完成
 */
void* timer_future_await(timer_future_t* timer_future) {
    if (!timer_future) return NULL;

    // 简化await实现
    // 实际应该由编译器生成的代码处理异步等待
    while (!future_is_ready((future_t*)timer_future)) {
        // 轮询Future
        poll_result_t result = timer_future_poll(timer_future, NULL);
        if (result.is_ready) {
            return result.value;
        }

        // 短暂等待
        struct timespec sleep_time = { .tv_sec = 0, .tv_nsec = 1000000 }; // 1ms
        nanosleep(&sleep_time, NULL);
    }

    return timer_future->base_future.result;
}

// ============================================================================
// 集成到Future系统
// ============================================================================

/**
 * @brief TimerFuture的Future接口适配
 */
static poll_result_t timer_future_poll_adapter(future_t* future, struct Task* task) {
    return timer_future_poll((timer_future_t*)future->user_data, task);
}

// Future vtable for TimerFuture
static const struct {
    poll_result_t (*poll)(future_t*, struct Task*);
} timer_future_vtable = {
    .poll = timer_future_poll_adapter
};
