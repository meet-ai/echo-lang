/**
 * @file event_publisher.h
 * @brief 事件发布接口定义 - 共享内核
 *
 * 定义统一的事件发布接口，所有上下文都可以发布领域事件
 */

#ifndef EVENT_PUBLISHER_H
#define EVENT_PUBLISHER_H

#include <stdint.h>
#include <stdbool.h>

// 事件优先级
typedef enum {
    EVENT_PRIORITY_LOW,      // 低优先级
    EVENT_PRIORITY_NORMAL,   // 普通优先级
    EVENT_PRIORITY_HIGH,     // 高优先级
    EVENT_PRIORITY_CRITICAL  // 关键优先级
} event_priority_t;

// 事件传递保证
typedef enum {
    DELIVERY_GUARANTEE_NONE,      // 无保证（尽力而为）
    DELIVERY_GUARANTEE_AT_LEAST_ONCE,  // 至少一次
    DELIVERY_GUARANTEE_EXACTLY_ONCE    // 精确一次
} delivery_guarantee_t;

// 事件头信息
typedef struct {
    const char* event_id;         // 事件唯一标识
    const char* event_type;       // 事件类型
    const char* source;           // 事件源
    uint64_t timestamp;           // 事件时间戳
    event_priority_t priority;    // 事件优先级
    delivery_guarantee_t guarantee; // 传递保证
    uint32_t version;             // 事件版本
} event_header_t;

// 通用事件接口
typedef struct event_t {
    event_header_t header;        // 事件头
    void* payload;               // 事件载荷
    size_t payload_size;         // 载荷大小
} event_t;

// 事件发布器接口
typedef struct event_publisher_t {
    // 方法
    bool (*publish)(struct event_publisher_t* self, const event_t* event);
    bool (*publish_batch)(struct event_publisher_t* self, const event_t** events, size_t count);
    bool (*flush)(struct event_publisher_t* self);
    void (*destroy)(struct event_publisher_t* self);

    // 私有数据
    void* private_data;
} event_publisher_t;

// 便利函数
event_t* event_create(const char* event_type, const char* source, void* payload, size_t payload_size);
void event_destroy(event_t* event);
bool event_set_priority(event_t* event, event_priority_t priority);
bool event_set_guarantee(event_t* event, delivery_guarantee_t guarantee);

// 发布器工厂函数
typedef event_publisher_t* (*event_publisher_factory_t)(void);

// 注册和获取发布器
bool register_event_publisher(const char* name, event_publisher_factory_t factory);
event_publisher_t* get_event_publisher(const char* name);

// 默认发布器
extern event_publisher_t* default_event_publisher;

#endif // EVENT_PUBLISHER_H
