/**
 * @file event_subscriber.h
 * @brief 事件订阅接口定义 - 共享内核
 *
 * 定义统一的事件订阅接口，所有上下文都可以订阅领域事件
 */

#ifndef EVENT_SUBSCRIBER_H
#define EVENT_SUBSCRIBER_H

#include "event_publisher.h"
#include <stdbool.h>

// 订阅模式
typedef enum {
    SUBSCRIPTION_MODE_SYNC,       // 同步处理
    SUBSCRIPTION_MODE_ASYNC,      // 异步处理
    SUBSCRIPTION_MODE_BATCH       // 批量处理
} subscription_mode_t;

// 订阅过滤器
typedef struct {
    const char* event_type_pattern;  // 事件类型模式（支持通配符）
    const char* source_pattern;      // 事件源模式
    event_priority_t min_priority;   // 最小优先级
    uint32_t min_version;           // 最小版本
} subscription_filter_t;

// 事件处理器函数类型
typedef bool (*event_handler_t)(const event_t* event, void* context);

// 订阅信息
typedef struct subscription_t {
    const char* subscription_id;     // 订阅唯一标识
    subscription_filter_t filter;    // 过滤器
    event_handler_t handler;         // 处理器函数
    void* context;                   // 处理器上下文
    subscription_mode_t mode;        // 处理模式
} subscription_t;

// 事件订阅器接口
typedef struct event_subscriber_t {
    // 方法
    bool (*subscribe)(struct event_subscriber_t* self, const subscription_t* subscription);
    bool (*unsubscribe)(struct event_subscriber_t* self, const char* subscription_id);
    bool (*start)(struct event_subscriber_t* self);
    bool (*stop)(struct event_subscriber_t* self);
    void (*destroy)(struct event_subscriber_t* self);

    // 私有数据
    void* private_data;
} event_subscriber_t;

// 订阅器工厂函数
typedef event_subscriber_t* (*event_subscriber_factory_t)(void);

// 注册和获取订阅器
bool register_event_subscriber(const char* name, event_subscriber_factory_t factory);
event_subscriber_t* get_event_subscriber(const char* name);

// 便利函数
subscription_t* subscription_create(const char* event_type_pattern,
                                   event_handler_t handler,
                                   void* context);
void subscription_destroy(subscription_t* subscription);

bool subscription_set_filter(subscription_t* subscription, const subscription_filter_t* filter);
bool subscription_set_mode(subscription_t* subscription, subscription_mode_t mode);

// 默认订阅器
extern event_subscriber_t* default_event_subscriber;

#endif // EVENT_SUBSCRIBER_H

