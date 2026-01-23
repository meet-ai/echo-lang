/**
 * @file runtime.h
 * @brief Echo 运行时核心类型定义和常量
 *
 * 这个头文件定义了运行时的基础类型、枚举和常量。
 * 它是稳定的ABI接口，不会频繁变化。
 */

#ifndef ECHO_RUNTIME_H
#define ECHO_RUNTIME_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>

// ============================================================================
// 基础类型定义
// ============================================================================

/** 任务ID类型 */
typedef uint64_t task_id_t;

/** Future ID类型 */
typedef uint64_t future_id_t;

/** 协程ID类型 */
typedef uint64_t coroutine_id_t;

/** 通道ID类型 */
typedef uint64_t channel_id_t;

/** 文件描述符类型 */
typedef int fd_t;

// ============================================================================
// 枚举类型定义
// ============================================================================

/**
 * @brief 任务状态枚举
 */
typedef enum task_status {
    TASK_STATUS_READY,      // 就绪状态，可以执行
    TASK_STATUS_RUNNING,    // 正在执行
    TASK_STATUS_WAITING,    // 等待异步操作完成
    TASK_STATUS_COMPLETED,  // 执行完成
    TASK_STATUS_FAILED,     // 执行失败
    TASK_STATUS_CANCELLED   // 被取消
} task_status_t;

/**
 * @brief Future状态枚举
 */
typedef enum future_state {
    FUTURE_STATE_PENDING,   // 异步操作未完成
    FUTURE_STATE_RESOLVED,  // 异步操作成功完成
    FUTURE_STATE_REJECTED   // 异步操作失败
} future_state_t;

/**
 * @brief 协程状态枚举
 */
typedef enum coroutine_state {
    COROUTINE_STATE_READY,      // 就绪
    COROUTINE_STATE_RUNNING,    // 运行中
    COROUTINE_STATE_SUSPENDED,  // 已挂起
    COROUTINE_STATE_COMPLETED,  // 已完成
    COROUTINE_STATE_FAILED      // 失败
} coroutine_state_t;

/**
 * @brief 事件类型枚举
 */
typedef enum event_type {
    EVENT_TYPE_READ    = 0x01,  // 可读事件
    EVENT_TYPE_WRITE   = 0x02,  // 可写事件
    EVENT_TYPE_ERROR   = 0x04,  // 错误事件
    EVENT_TYPE_TIMER   = 0x08,  // 定时器事件
    EVENT_TYPE_SIGNAL  = 0x10   // 信号事件
} event_type_t;

/**
 * @brief 通道类型枚举
 */
typedef enum channel_type {
    CHANNEL_TYPE_BUFFERED,    // 有缓冲区
    CHANNEL_TYPE_UNBUFFERED   // 无缓冲区
} channel_type_t;

// ============================================================================
// 值对象结构体定义
// ============================================================================

/**
 * @brief 事件掩码值对象
 */
typedef struct event_mask {
    uint32_t mask;
} event_mask_t;

/**
 * @brief 超时时间值对象
 */
typedef struct timeout {
    struct timespec duration;
} timeout_t;

/**
 * @brief 事件通知结构体
 */
typedef struct event_notification {
    fd_t fd;                        // 文件描述符
    event_mask_t events;           // 发生的事件
    struct timespec timestamp;     // 事件发生时间
    void* user_data;               // 用户数据
} event_notification_t;

// ============================================================================
// 构造函数和工具函数
// ============================================================================

/**
 * @brief 创建事件掩码
 */
static inline event_mask_t event_mask_create(uint32_t mask) {
    return (event_mask_t){ .mask = mask };
}

/**
 * @brief 检查事件掩码是否包含指定事件
 */
static inline bool event_mask_has(event_mask_t mask, event_type_t type) {
    return (mask.mask & (uint32_t)type) != 0;
}

/**
 * @brief 创建超时时间（毫秒）
 */
static inline timeout_t timeout_create(int milliseconds) {
    timeout_t t;
    if (milliseconds < 0) {
        // 负数表示无限等待
        t.duration.tv_sec = -1;
        t.duration.tv_nsec = -1;
    } else {
        t.duration.tv_sec = milliseconds / 1000;
        t.duration.tv_nsec = (milliseconds % 1000) * 1000000;
    }
    return t;
}

/**
 * @brief 创建事件通知
 */
static inline event_notification_t event_notification_create(fd_t fd, event_mask_t events, void* user_data) {
    event_notification_t notification = {
        .fd = fd,
        .events = events,
        .user_data = user_data
    };
    clock_gettime(CLOCK_MONOTONIC, &notification.timestamp);
    return notification;
}

// ============================================================================
// 常量定义
// ============================================================================

/** 默认栈大小 (64KB) */
#define DEFAULT_STACK_SIZE (64 * 1024)

/** 默认任务池大小 */
#define DEFAULT_TASK_POOL_SIZE 1024

/** 默认队列大小 */
#define DEFAULT_QUEUE_SIZE 256

/** 默认缓冲区大小 */
#define DEFAULT_BUFFER_SIZE 16

/** 最大文件描述符数量 */
#define MAX_FD_COUNT 1024

/** 定时器精度 (毫秒) */
#define TIMER_PRECISION_MS 1

#endif // ECHO_RUNTIME_H

