/**
 * @file bus_impl.h
 * @brief 事件总线实现（基础设施层）
 *
 * 实现基于内存的事件总线，支持事件的发布和订阅。
 */

#ifndef INFRASTRUCTURE_EVENT_BUS_IMPL_H
#define INFRASTRUCTURE_EVENT_BUS_IMPL_H

#include "../../domain/shared/events/bus.h"
#include <pthread.h>

// ==================== 内存事件总线实现 ====================

/**
 * @brief 内存事件总线实现
 * 
 * 使用Channel作为事件传递媒介，支持异步非阻塞处理。
 */
typedef struct InMemoryEventBus {
    EventBus base;  // 基类（接口）
    
    // 订阅者映射：事件类型 -> 处理器列表
    struct EventHandlerList* subscribers;  // 简化为链表
    
    // 保护subscribers的读写锁
    pthread_mutex_t mu;
    
    // Channel缓冲大小（避免阻塞发布者）
    uint32_t buffer_size;
} InMemoryEventBus;

/**
 * @brief 创建内存事件总线
 * 
 * @param buffer_size Channel缓冲大小（默认100）
 * @return 事件总线实例，失败返回NULL
 */
EventBus* in_memory_event_bus_create(uint32_t buffer_size);

/**
 * @brief 销毁内存事件总线
 * 
 * @param bus 事件总线实例
 */
void in_memory_event_bus_destroy(EventBus* bus);

#endif // INFRASTRUCTURE_EVENT_BUS_IMPL_H
