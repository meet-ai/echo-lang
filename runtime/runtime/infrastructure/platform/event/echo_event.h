/**
 * @file echo_event.h
 * @brief 平台事件抽象的统一头文件
 *
 * 这个头文件定义了平台实现需要提供的接口。
 * 每个平台（如Linux epoll、macOS kqueue、Windows IOCP）都需要实现这些函数。
 */

#ifndef ECHO_EVENT_H
#define ECHO_EVENT_H

#include "../../../include/echo/reactor.h"  // for reactor_event_t

// ============================================================================
// 平台抽象类型定义
// ============================================================================

/**
 * @brief 事件回调函数类型
 */
typedef void (*event_callback_t)(void* user_data);

// ============================================================================
// 平台操作集定义
// ============================================================================

/**
 * @brief 平台事件操作集
 * 策略模式：定义平台必须实现的接口
 */
struct platform_event_operations {
    // 创建平台特定的事件循环状态
    void* (*create)(void);

    // 添加文件描述符事件
    int (*add_fd)(void* state, int fd, uint32_t events, void* user_data);

    // 移除文件描述符事件
    int (*remove_fd)(void* state, int fd);

    // 修改文件描述符事件
    int (*modify_fd)(void* state, int fd, uint32_t events);

    // 轮询事件
    int (*poll)(void* state, reactor_event_t* events, int max_events, reactor_timeout_t timeout);

    // 唤醒事件循环
    void (*wakeup)(void* state);

    // 销毁平台特定状态
    void (*destroy)(void* state);
};

/**
 * @brief 平台类型枚举
 */
typedef enum platform_type {
    PLATFORM_LINUX,
    PLATFORM_MACOS,
    PLATFORM_WINDOWS,
    PLATFORM_UNKNOWN
} platform_type_t;

/**
 * @brief 检测当前运行平台
 *
 * @return 平台类型
 */
platform_type_t detect_platform(void);

/**
 * @brief 获取当前平台的操作集
 * 工厂模式：运行时决议使用哪个平台的实现
 *
 * @return 平台操作集指针
 */
const struct platform_event_operations* get_platform_event_ops(void);

/**
 * @brief 获取epoll操作集 (Linux)
 */
const struct platform_event_operations* get_epoll_ops(void);

/**
 * @brief 获取kqueue操作集 (macOS/BSD)
 */
const struct platform_event_operations* get_kqueue_ops(void);

#endif // ECHO_EVENT_H
