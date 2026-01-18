/**
 * @file event_loop.h
 * @brief 事件循环领域层接口定义
 *
 * 领域驱动设计 (DDD) 架构：
 * - 领域层：定义核心业务抽象，屏蔽平台差异
 * - 基础设施层：实现平台特定的技术细节
 * - 应用层：使用领域接口实现业务逻辑
 */

#ifndef EVENT_LOOP_H
#define EVENT_LOOP_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>

// 前向声明（避免循环依赖）
struct Task;

// ============================================================================
// 领域层：核心业务抽象定义
// ============================================================================

/**
 * @brief 事件类型枚举
 * 领域概念：定义业务层关心的事件类型
 */
typedef enum {
    EVENT_READ   = 0x01,  // 可读事件
    EVENT_WRITE  = 0x02,  // 可写事件
    EVENT_ERROR  = 0x04,  // 错误事件
    EVENT_TIMER  = 0x08,  // 定时器事件
    EVENT_SIGNAL = 0x10   // 信号事件
} event_type_t;

/**
 * @brief 事件掩码结构体
 * 值对象模式：不可变的事件类型组合
 */
typedef struct {
    uint32_t mask;  // 位掩码存储事件类型
} event_mask_t;

/**
 * @brief 事件通知结构体
 * DTO模式：领域间传递事件信息的载体
 */
typedef struct {
    int fd;                    // 文件描述符
    event_mask_t events;       // 发生的事件
    struct timespec timestamp; // 事件发生时间
    void* user_data;           // 用户数据（指向Task）
} event_notification_t;

/**
 * @brief 超时时间值对象
 * 值对象模式：表示时间间隔的不可变对象
 */
typedef struct {
    struct timespec duration;
} timeout_t;

/**
 * @brief 事件回调函数类型
 * 函数式接口：定义事件处理的契约
 */
typedef void (*event_callback_t)(event_notification_t* notification);

// ============================================================================
// 领域服务：业务逻辑接口定义
// ============================================================================

/**
 * @brief 事件循环接口
 * 领域服务模式：定义事件循环的核心业务能力
 * 适配器模式：通过此接口屏蔽不同平台的实现差异
 */
typedef struct event_loop_vtable {
    // 平台信息
    const char* (*platform_name)(void);

    // 事件注册管理（聚合根行为）
    int (*add_event)(void* self, int fd, event_mask_t mask, event_callback_t callback, void* user_data);
    int (*remove_event)(void* self, int fd);
    int (*modify_event)(void* self, int fd, event_mask_t mask);

    // 事件轮询（核心业务逻辑）
    int (*poll_events)(void* self, timeout_t timeout, event_notification_t* notifications, int max_events);

    // 跨线程唤醒（解耦机制）
    int (*wakeup)(void* self);

    // 生命周期管理
    void (*destroy)(void* self);
} event_loop_vtable_t;

/**
 * @brief 事件循环聚合根
 * 聚合根模式：封装事件循环的完整生命周期和状态
 */
typedef struct event_loop {
    void* impl;                          // 平台特定实现（基础设施层）
    const event_loop_vtable_t* vtable;   // 虚函数表（领域接口）
} event_loop_t;

// ============================================================================
// 领域层：业务逻辑函数声明
// ============================================================================

/**
 * @brief 创建事件循环实例
 * 工厂方法模式：根据运行平台创建相应的事件循环实现
 *
 * @return 成功返回事件循环指针，失败返回NULL
 */
event_loop_t* event_loop_create(void);

/**
 * @brief 销毁事件循环实例
 * @param loop 要销毁的事件循环
 */
void event_loop_destroy(event_loop_t* loop);

/**
 * @brief 添加事件监听
 * 聚合根行为：管理单个事件注册的完整生命周期
 *
 * @param loop 事件循环
 * @param fd 文件描述符
 * @param mask 事件掩码
 * @param callback 事件回调函数
 * @param user_data 用户数据（通常指向Task）
 * @return 成功返回0，失败返回-1
 */
int event_loop_add_event(event_loop_t* loop, int fd, event_mask_t mask,
                        event_callback_t callback, void* user_data);

/**
 * @brief 移除事件监听
 * @param loop 事件循环
 * @param fd 文件描述符
 * @return 成功返回0，失败返回-1
 */
int event_loop_remove_event(event_loop_t* loop, int fd);

/**
 * @brief 修改事件监听
 * @param loop 事件循环
 * @param fd 文件描述符
 * @param mask 新的事件掩码
 * @return 成功返回0，失败返回-1
 */
int event_loop_modify_event(event_loop_t* loop, int fd, event_mask_t mask);

/**
 * @brief 执行事件轮询
 * 核心业务逻辑：阻塞等待事件发生并通知
 *
 * @param loop 事件循环
 * @param timeout 超时时间（NULL表示无限等待）
 * @param notifications 输出事件通知数组
 * @param max_events 数组最大长度
 * @return 成功返回事件数量，失败返回-1
 */
int event_loop_poll(event_loop_t* loop, timeout_t* timeout,
                   event_notification_t* notifications, int max_events);

/**
 * @brief 唤醒事件循环
 * 跨线程唤醒机制：允许其他线程中断事件循环的等待
 *
 * @param loop 事件循环
 * @return 成功返回0，失败返回-1
 */
int event_loop_wakeup(event_loop_t* loop);

// ============================================================================
// 值对象构造函数和工具函数
// ============================================================================

/**
 * @brief 创建事件掩码
 * 工厂方法模式：创建不可变的值对象
 */
static inline event_mask_t event_mask_create(event_type_t types) {
    return (event_mask_t){ .mask = (uint32_t)types };
}

/**
 * @brief 添加事件类型到掩码
 */
static inline event_mask_t event_mask_add(event_mask_t mask, event_type_t type) {
    return (event_mask_t){ .mask = mask.mask | (uint32_t)type };
}

/**
 * @brief 检查掩码是否包含事件类型
 */
static inline bool event_mask_has(event_mask_t mask, event_type_t type) {
    return (mask.mask & (uint32_t)type) != 0;
}

/**
 * @brief 创建超时时间
 * @param milliseconds 毫秒数（-1表示无限等待）
 */
static inline timeout_t timeout_create(int milliseconds) {
    timeout_t t;
    if (milliseconds < 0) {
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
static inline event_notification_t event_notification_create(int fd, event_mask_t events, void* user_data) {
    event_notification_t notification;
    notification.fd = fd;
    notification.events = events;
    notification.user_data = user_data;
    clock_gettime(CLOCK_MONOTONIC, &notification.timestamp);
    return notification;
}

#endif // EVENT_LOOP_H
