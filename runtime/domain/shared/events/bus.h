/**
 * @file bus.h
 * @brief 事件总线接口定义（领域层）
 *
 * 定义事件总线的接口，用于发布和订阅领域事件。
 * 实现应该在基础设施层。
 */

#ifndef DOMAIN_SHARED_EVENTS_BUS_H
#define DOMAIN_SHARED_EVENTS_BUS_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>

// ==================== 领域事件接口 ====================

/**
 * @brief 领域事件接口
 * 
 * 所有领域事件都必须实现此接口。
 * 在C语言中，我们使用函数指针来模拟接口。
 */
typedef struct DomainEvent {
    /**
     * @brief 获取事件类型
     * @return 事件类型字符串（如 "TaskScheduled", "TaskCompleted"）
     */
    const char* (*event_type)(const struct DomainEvent* event);
    
    /**
     * @brief 获取事件发生时间
     * @return 事件发生时间
     */
    time_t (*occurred_at)(const struct DomainEvent* event);
    
    /**
     * @brief 获取事件ID
     * @return 事件唯一ID
     */
    uint64_t (*event_id)(const struct DomainEvent* event);
    
    /**
     * @brief 销毁事件
     * @param event 事件指针
     */
    void (*destroy)(struct DomainEvent* event);
    
    /**
     * @brief 事件数据（由具体事件类型实现）
     */
    void* data;
} DomainEvent;

// ==================== 事件处理器接口 ====================

/**
 * @brief 事件处理器接口
 * 
 * 所有事件处理器都必须实现此接口。
 */
typedef struct EventHandler {
    /**
     * @brief 处理事件
     * @param handler 处理器实例
     * @param event 领域事件
     * @return 成功返回0，失败返回-1
     */
    int (*handle)(struct EventHandler* handler, const DomainEvent* event);
    
    /**
     * @brief 处理器数据（由具体处理器实现）
     */
    void* data;
} EventHandler;

// ==================== 事件总线接口 ====================

/**
 * @brief 事件总线接口（领域层）
 * 
 * 负责事件的发布和订阅，实现应该在基础设施层。
 */
typedef struct EventBus {
    /**
     * @brief 发布事件
     * 
     * 事件会被异步分发给所有订阅了该事件类型的处理器。
     * 
     * @param bus 事件总线实例
     * @param event 领域事件
     * @return 成功返回0，失败返回-1
     */
    int (*publish)(struct EventBus* bus, const DomainEvent* event);
    
    /**
     * @brief 订阅事件
     * 
     * @param bus 事件总线实例
     * @param event_type 事件类型（如 "TaskScheduled"）
     * @param handler 事件处理器
     * @return 成功返回0，失败返回-1
     */
    int (*subscribe)(struct EventBus* bus, const char* event_type, EventHandler* handler);
    
    /**
     * @brief 取消订阅
     * 
     * @param bus 事件总线实例
     * @param event_type 事件类型
     * @param handler 要取消订阅的处理器
     * @return 成功返回0，失败返回-1
     */
    int (*unsubscribe)(struct EventBus* bus, const char* event_type, EventHandler* handler);
    
    /**
     * @brief 事件总线数据（由具体实现使用）
     */
    void* data;
} EventBus;

// ==================== 事件总线工厂函数 ====================

/**
 * @brief 创建事件总线实例
 * 
 * 这是一个工厂函数，由基础设施层实现。
 * 
 * @param buffer_size Channel缓冲大小（用于异步处理）
 * @return 事件总线实例，失败返回NULL
 */
EventBus* event_bus_create(uint32_t buffer_size);

/**
 * @brief 销毁事件总线实例
 * 
 * @param bus 事件总线实例
 */
void event_bus_destroy(EventBus* bus);

// ==================== 辅助函数 ====================

/**
 * @brief 获取事件类型字符串
 * 
 * 辅助函数，用于从DomainEvent获取事件类型。
 * 
 * @param event 领域事件
 * @return 事件类型字符串
 */
static inline const char* domain_event_get_type(const DomainEvent* event) {
    if (!event || !event->event_type) return NULL;
    return event->event_type(event);
}

/**
 * @brief 获取事件发生时间
 * 
 * @param event 领域事件
 * @return 事件发生时间
 */
static inline time_t domain_event_get_occurred_at(const DomainEvent* event) {
    if (!event || !event->occurred_at) return 0;
    return event->occurred_at(event);
}

/**
 * @brief 获取事件ID
 * 
 * @param event 领域事件
 * @return 事件ID
 */
static inline uint64_t domain_event_get_id(const DomainEvent* event) {
    if (!event || !event->event_id) return 0;
    return event->event_id(event);
}

/**
 * @brief 销毁领域事件
 * 
 * @param event 领域事件
 */
static inline void domain_event_destroy(DomainEvent* event) {
    if (event && event->destroy) {
        event->destroy(event);
    }
}

#endif // DOMAIN_SHARED_EVENTS_BUS_H
