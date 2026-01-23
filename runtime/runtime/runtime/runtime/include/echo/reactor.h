/**
 * @file reactor.h
 * @brief Echo Runtime 反应器接口
 *
 * 这是EventLoop的稳定对外接口，定义了异步I/O的核心抽象。
 * 上层代码（如调度器、执行器）通过此接口使用事件循环功能。
 */

#ifndef ECHO_REACTOR_H
#define ECHO_REACTOR_H

#include <stdint.h>
#include <stdbool.h>

// ============================================================================
// 领域层：核心业务抽象定义
// ============================================================================

/**
 * @brief 事件类型枚举
 * 领域概念：定义业务层关心的事件类型
 */
typedef enum {
    REACTOR_EVENT_READ   = 0x01,  // 可读事件
    REACTOR_EVENT_WRITE  = 0x02,  // 可写事件
    REACTOR_EVENT_ERROR  = 0x04,  // 错误事件
    REACTOR_EVENT_TIMER  = 0x08,  // 定时器事件
    REACTOR_EVENT_SIGNAL = 0x10   // 信号事件
} reactor_event_type_t;

/**
 * @brief 事件结构体
 * DTO模式：领域间传递事件信息的载体
 */
typedef struct reactor_event {
    int fd;                    // 文件描述符
    reactor_event_type_t type; // 事件类型
    void* user_data;           // 用户数据（通常是Waker）
} reactor_event_t;

/**
 * @brief 事件循环句柄
 * 领域实体：代表一个事件循环实例
 */
typedef struct reactor_handle reactor_handle_t;

/**
 * @brief 超时结构体
 */
typedef struct reactor_timeout {
    int milliseconds;  // 毫秒
} reactor_timeout_t;

// ============================================================================
// 应用层：业务逻辑接口
// ============================================================================

/**
 * @brief 创建事件循环
 * @return 事件循环句柄，失败返回NULL
 */
reactor_handle_t* reactor_create(void);

/**
 * @brief 销毁事件循环
 * @param handle 事件循环句柄
 */
void reactor_destroy(reactor_handle_t* handle);

/**
 * @brief 启动事件循环
 * @param handle 事件循环句柄
 * @return true成功，false失败
 */
bool reactor_start(reactor_handle_t* handle);

/**
 * @brief 停止事件循环
 * @param handle 事件循环句柄
 */
void reactor_stop(reactor_handle_t* handle);

/**
 * @brief 轮询事件
 * @param handle 事件循环句柄
 * @param timeout 超时时间
 * @return 处理的事件数量，-1表示错误
 */
int reactor_poll(reactor_handle_t* handle, reactor_timeout_t timeout);

/**
 * @brief 注册文件描述符事件
 * @param handle 事件循环句柄
 * @param fd 文件描述符
 * @param events 监听的事件类型（位掩码）
 * @param user_data 用户数据
 * @return 0成功，-1失败
 */
int reactor_add_fd(reactor_handle_t* handle, int fd, uint32_t events, void* user_data);

/**
 * @brief 移除文件描述符事件
 * @param handle 事件循环句柄
 * @param fd 文件描述符
 * @return 0成功，-1失败
 */
int reactor_remove_fd(reactor_handle_t* handle, int fd);

/**
 * @brief 修改文件描述符事件
 * @param handle 事件循环句柄
 * @param fd 文件描述符
 * @param events 新的监听事件类型
 * @return 0成功，-1失败
 */
int reactor_modify_fd(reactor_handle_t* handle, int fd, uint32_t events);

/**
 * @brief 唤醒事件循环
 * @param handle 事件循环句柄
 */
void reactor_wakeup(reactor_handle_t* handle);

/**
 * @brief 获取统计信息
 * @param handle 事件循环句柄
 * @param total_events 总事件数
 * @param processed_events 已处理事件数
 */
void reactor_get_stats(reactor_handle_t* handle, uint64_t* total_events, uint64_t* processed_events);

#endif // ECHO_REACTOR_H

