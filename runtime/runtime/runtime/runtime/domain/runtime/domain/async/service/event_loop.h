/**
 * @file event_loop.h
 * @brief EventLoop（事件循环）接口定义
 *
 * EventLoop 是异步I/O的核心组件，负责监听和处理各种异步事件。
 * 支持定时器事件、文件描述符事件等。
 */

#ifndef EVENT_LOOP_H
#define EVENT_LOOP_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>
#include "../../../include/echo/runtime.h"  // for event_type_t

// ============================================================================
// 事件类型定义
// ============================================================================

// 事件类型使用 runtime.h 中已定义的 event_type_t

/**
 * @brief 事件回调函数类型
 */
typedef void (*event_callback_t)(void* user_data);

/**
 * @brief 事件句柄
 */
typedef struct event_handle {
    uint64_t id;                    // 事件唯一ID
    event_type_t type;              // 事件类型
    void* user_data;                // 用户数据
    event_callback_t callback;      // 回调函数
    struct event_handle* next;      // 链表下一个
} event_handle_t;

// ============================================================================
// 定时器事件定义
// ============================================================================

/**
 * @brief 定时器事件
 */
typedef struct timer_event {
    event_handle_t handle;          // 基础事件句柄
    struct timespec deadline;       // 触发时间
    bool periodic;                  // 是否周期性
    struct timespec interval;       // 周期间隔（如果是周期性）
} timer_event_t;

// ============================================================================
// 文件描述符事件定义
// ============================================================================

/**
 * @brief 文件描述符事件
 */
typedef struct fd_event {
    event_handle_t handle;          // 基础事件句柄
    int fd;                         // 文件描述符
    uint32_t events;                // 监听的事件（EPOLLIN, EPOLLOUT等）
    uint32_t triggered_events;      // 实际触发的事件
} fd_event_t;

// ============================================================================
// 信号事件定义
// ============================================================================

/**
 * @brief 信号事件
 */
typedef struct signal_event {
    event_handle_t handle;          // 基础事件句柄
    int signum;                     // 信号编号
} signal_event_t;

// ============================================================================
// EventLoop 接口
// ============================================================================

/**
 * @brief EventLoop 结构体
 */
typedef struct event_loop {
    // 事件队列
    event_handle_t* events;         // 注册的事件链表
    uint64_t next_event_id;         // 下一个事件ID

    // 状态
    bool running;                   // 是否正在运行
    int wakeup_fd[2];              // 用于唤醒的管道

    // 统计信息
    uint64_t total_events;          // 总事件数
    uint64_t processed_events;      // 已处理事件数
} event_loop_t;

/**
 * @brief 创建EventLoop
 */
event_loop_t* event_loop_create(void);

/**
 * @brief 销毁EventLoop
 */
void event_loop_destroy(event_loop_t* loop);

/**
 * @brief 启动EventLoop
 */
bool event_loop_start(event_loop_t* loop);

/**
 * @brief 停止EventLoop
 */
void event_loop_stop(event_loop_t* loop);

/**
 * @brief 轮询事件（非阻塞）
 *
 * @param loop EventLoop实例
 * @param timeout_ms 超时时间（毫秒），0表示立即返回，-1表示无限等待
 * @return 处理的事件数量
 */
int event_loop_poll(event_loop_t* loop, int timeout_ms);

/**
 * @brief 唤醒EventLoop
 */
void event_loop_wakeup(event_loop_t* loop);

/**
 * @brief 注册定时器事件
 */
event_handle_t* event_loop_add_timer(
    event_loop_t* loop,
    struct timespec deadline,
    event_callback_t callback,
    void* user_data
);

/**
 * @brief 注册周期性定时器事件
 */
event_handle_t* event_loop_add_periodic_timer(
    event_loop_t* loop,
    struct timespec interval,
    event_callback_t callback,
    void* user_data
);

/**
 * @brief 注册文件描述符事件
 */
event_handle_t* event_loop_add_fd_event(
    event_loop_t* loop,
    int fd,
    uint32_t events,
    event_callback_t callback,
    void* user_data
);

/**
 * @brief 注册信号事件
 */
event_handle_t* event_loop_add_signal_event(
    event_loop_t* loop,
    int signum,
    event_callback_t callback,
    void* user_data
);

/**
 * @brief 移除事件
 */
bool event_loop_remove_event(event_loop_t* loop, event_handle_t* handle);

/**
 * @brief 修改文件描述符事件
 */
bool event_loop_modify_fd_event(event_loop_t* loop, event_handle_t* handle, uint32_t new_events);

/**
 * @brief 获取统计信息
 */
void event_loop_get_stats(event_loop_t* loop, uint64_t* total, uint64_t* processed);

#endif // EVENT_LOOP_H
