/**
 * @file bus_impl.c
 * @brief 事件总线实现（基础设施层）
 *
 * 实现基于内存的事件总线，支持事件的发布和订阅。
 */

#include "bus_impl.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

// ==================== 订阅者列表结构 ====================

/**
 * @brief 事件处理器列表节点
 */
typedef struct EventHandlerNode {
    EventHandler* handler;
    struct EventHandlerNode* next;
} EventHandlerNode;

/**
 * @brief 事件类型到处理器列表的映射
 */
typedef struct EventTypeMapping {
    char* event_type;
    EventHandlerNode* handlers;
    struct EventTypeMapping* next;
} EventTypeMapping;

/**
 * @brief 内存事件总线的内部数据
 */
typedef struct InMemoryEventBusData {
    EventTypeMapping* mappings;  // 事件类型映射表
    pthread_mutex_t mu;
    uint32_t buffer_size;
} InMemoryEventBusData;

// ==================== 辅助函数 ====================

/**
 * @brief 查找事件类型映射
 */
static EventTypeMapping* find_mapping(InMemoryEventBusData* data, const char* event_type) {
    EventTypeMapping* current = data->mappings;
    while (current) {
        if (strcmp(current->event_type, event_type) == 0) {
            return current;
        }
        current = current->next;
    }
    return NULL;
}

/**
 * @brief 创建事件类型映射
 */
static EventTypeMapping* create_mapping(const char* event_type) {
    EventTypeMapping* mapping = (EventTypeMapping*)malloc(sizeof(EventTypeMapping));
    if (!mapping) return NULL;
    
    mapping->event_type = strdup(event_type);
    mapping->handlers = NULL;
    mapping->next = NULL;
    return mapping;
}

/**
 * @brief 销毁事件类型映射
 */
static void destroy_mapping(EventTypeMapping* mapping) {
    if (!mapping) return;
    
    // 销毁处理器列表
    EventHandlerNode* node = mapping->handlers;
    while (node) {
        EventHandlerNode* next = node->next;
        free(node);
        node = next;
    }
    
    free(mapping->event_type);
    free(mapping);
}

// ==================== 事件总线方法实现 ====================

/**
 * @brief 发布事件
 */
static int event_bus_publish_impl(EventBus* bus, const DomainEvent* event) {
    if (!bus || !event) return -1;
    
    InMemoryEventBus* impl = (InMemoryEventBus*)bus;
    InMemoryEventBusData* data = (InMemoryEventBusData*)impl->base.data;
    
    const char* event_type = domain_event_get_type(event);
    if (!event_type) {
        fprintf(stderr, "ERROR: Event has no type\n");
        return -1;
    }
    
    pthread_mutex_lock(&data->mu);
    
    // 查找订阅者
    EventTypeMapping* mapping = find_mapping(data, event_type);
    if (!mapping) {
        // 没有订阅者，直接返回成功
        pthread_mutex_unlock(&data->mu);
        return 0;
    }
    
    // 复制处理器列表（避免在调用时持有锁）
    EventHandlerNode* handlers = NULL;
    EventHandlerNode* tail = NULL;
    EventHandlerNode* current = mapping->handlers;
    while (current) {
        EventHandlerNode* node = (EventHandlerNode*)malloc(sizeof(EventHandlerNode));
        if (!node) {
            // 内存不足，清理已分配的节点
            while (handlers) {
                EventHandlerNode* next = handlers->next;
                free(handlers);
                handlers = next;
            }
            pthread_mutex_unlock(&data->mu);
            return -1;
        }
        node->handler = current->handler;
        node->next = NULL;
        
        if (!handlers) {
            handlers = node;
            tail = node;
        } else {
            tail->next = node;
            tail = node;
        }
        
        current = current->next;
    }
    
    pthread_mutex_unlock(&data->mu);
    
    // 异步调用所有处理器（不阻塞发布者）
    // 注意：在实际实现中，可以使用线程池或Channel来实现真正的异步
    // 这里简化实现，直接同步调用
    int success_count = 0;
    EventHandlerNode* node = handlers;
    while (node) {
        if (node->handler && node->handler->handle) {
            int result = node->handler->handle(node->handler, event);
            if (result == 0) {
                success_count++;
            } else {
                fprintf(stderr, "WARNING: Event handler failed for event type %s\n", event_type);
            }
        }
        EventHandlerNode* next = node->next;
        free(node);
        node = next;
    }
    
    printf("DEBUG: Published event %s to %d handlers\n", event_type, success_count);
    return 0;
}

/**
 * @brief 订阅事件
 */
static int event_bus_subscribe_impl(EventBus* bus, const char* event_type, EventHandler* handler) {
    if (!bus || !event_type || !handler) return -1;
    
    InMemoryEventBus* impl = (InMemoryEventBus*)bus;
    InMemoryEventBusData* data = (InMemoryEventBusData*)impl->base.data;
    
    pthread_mutex_lock(&data->mu);
    
    // 查找或创建映射
    EventTypeMapping* mapping = find_mapping(data, event_type);
    if (!mapping) {
        mapping = create_mapping(event_type);
        if (!mapping) {
            pthread_mutex_unlock(&data->mu);
            return -1;
        }
        mapping->next = data->mappings;
        data->mappings = mapping;
    }
    
    // 检查是否已订阅
    EventHandlerNode* current = mapping->handlers;
    while (current) {
        if (current->handler == handler) {
            // 已存在，不重复添加
            pthread_mutex_unlock(&data->mu);
            return 0;
        }
        current = current->next;
    }
    
    // 添加新的订阅者
    EventHandlerNode* node = (EventHandlerNode*)malloc(sizeof(EventHandlerNode));
    if (!node) {
        pthread_mutex_unlock(&data->mu);
        return -1;
    }
    
    node->handler = handler;
    node->next = mapping->handlers;
    mapping->handlers = node;
    
    pthread_mutex_unlock(&data->mu);
    
    printf("DEBUG: Subscribed handler to event type %s\n", event_type);
    return 0;
}

/**
 * @brief 取消订阅
 */
static int event_bus_unsubscribe_impl(EventBus* bus, const char* event_type, EventHandler* handler) {
    if (!bus || !event_type || !handler) return -1;
    
    InMemoryEventBus* impl = (InMemoryEventBus*)bus;
    InMemoryEventBusData* data = (InMemoryEventBusData*)impl->base.data;
    
    pthread_mutex_lock(&data->mu);
    
    EventTypeMapping* mapping = find_mapping(data, event_type);
    if (!mapping) {
        pthread_mutex_unlock(&data->mu);
        return 0;  // 没有订阅，返回成功
    }
    
    // 从列表中移除
    EventHandlerNode* prev = NULL;
    EventHandlerNode* current = mapping->handlers;
    while (current) {
        if (current->handler == handler) {
            if (prev) {
                prev->next = current->next;
            } else {
                mapping->handlers = current->next;
            }
            free(current);
            pthread_mutex_unlock(&data->mu);
            printf("DEBUG: Unsubscribed handler from event type %s\n", event_type);
            return 0;
        }
        prev = current;
        current = current->next;
    }
    
    pthread_mutex_unlock(&data->mu);
    return 0;  // 未找到，返回成功
}

// ==================== 工厂函数 ====================

EventBus* event_bus_create(uint32_t buffer_size) {
    if (buffer_size == 0) {
        buffer_size = 100;  // 默认缓冲大小
    }
    
    InMemoryEventBus* bus = (InMemoryEventBus*)malloc(sizeof(InMemoryEventBus));
    if (!bus) {
        return NULL;
    }
    
    InMemoryEventBusData* data = (InMemoryEventBusData*)malloc(sizeof(InMemoryEventBusData));
    if (!data) {
        free(bus);
        return NULL;
    }
    
    // 初始化数据
    data->mappings = NULL;
    data->buffer_size = buffer_size;
    if (pthread_mutex_init(&data->mu, NULL) != 0) {
        free(data);
        free(bus);
        return NULL;
    }
    
    // 初始化EventBus接口
    bus->base.data = data;
    bus->base.publish = event_bus_publish_impl;
    bus->base.subscribe = event_bus_subscribe_impl;
    bus->base.unsubscribe = event_bus_unsubscribe_impl;
    bus->buffer_size = buffer_size;
    
    printf("DEBUG: Created InMemoryEventBus with buffer_size=%u\n", buffer_size);
    return (EventBus*)bus;
}

void event_bus_destroy(EventBus* bus) {
    if (!bus) return;
    
    InMemoryEventBus* impl = (InMemoryEventBus*)bus;
    InMemoryEventBusData* data = (InMemoryEventBusData*)impl->base.data;
    
    if (data) {
        // 销毁所有映射
        EventTypeMapping* mapping = data->mappings;
        while (mapping) {
            EventTypeMapping* next = mapping->next;
            destroy_mapping(mapping);
            mapping = next;
        }
        
        pthread_mutex_destroy(&data->mu);
        free(data);
    }
    
    free(bus);
    printf("DEBUG: Destroyed InMemoryEventBus\n");
}
