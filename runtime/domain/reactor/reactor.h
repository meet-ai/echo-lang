/**
 * @file reactor.h
 * @brief Reactor 聚合根定义
 *
 * Reactor 是异步运行时的反应器，负责监听I/O事件并通过Waker唤醒任务。
 */

#ifndef REACTOR_H
#define REACTOR_H

#include "../../../include/echo/runtime.h"
#include "../../../include/echo/future.h"

// 前向声明
struct EventLoop;

// ============================================================================
// Reactor 聚合根定义
// ============================================================================

/**
 * @brief Reactor 聚合根结构体
 */
typedef struct Reactor {
    uint64_t id;                      // 反应器ID
    struct EventLoop* event_loop;      // 事件循环
    void* waker_registry;             // Waker注册表 (fd -> waker 映射)
    void* event_dispatcher;           // 事件分发器
} reactor_t;

// ============================================================================
// Reactor 聚合根行为接口
// ============================================================================

/**
 * @brief 创建反应器
 * @param event_loop 事件循环
 * @return 新创建的反应器
 */
reactor_t* reactor_create(struct EventLoop* event_loop);

/**
 * @brief 销毁反应器
 * @param reactor 要销毁的反应器
 */
void reactor_destroy(reactor_t* reactor);

/**
 * @brief 注册Future到反应器
 * @param reactor 反应器
 * @param future 要注册的Future
 * @param waker 关联的Waker
 */
void reactor_register_future(reactor_t* reactor, struct Future* future, waker_t* waker);

/**
 * @brief 取消注册Future
 * @param reactor 反应器
 * @param future 要取消注册的Future
 */
void reactor_unregister_future(reactor_t* reactor, struct Future* future);

/**
 * @brief 轮询事件
 * @param reactor 反应器
 * @param timeout 超时时间
 * @return 事件数量，-1表示错误
 */
int reactor_poll_events(reactor_t* reactor, timeout_t timeout);

/**
 * @brief 唤醒反应器
 * @param reactor 反应器
 */
void reactor_wakeup(reactor_t* reactor);

#endif // REACTOR_H

