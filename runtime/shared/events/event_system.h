#ifndef EVENT_SYSTEM_H
#define EVENT_SYSTEM_H

#include "../types/common_types.h"
#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct EventSystem;
struct EventListener;
struct EventPublisher;

// 事件类型枚举（扩展系统事件）
typedef enum {
    // 系统事件 (0-999)
    EVENT_SYSTEM_STARTED = 1,
    EVENT_SYSTEM_STOPPING,
    EVENT_SYSTEM_STOPPED,
    EVENT_SYSTEM_ERROR,

    // 任务事件 (1000-1999)
    EVENT_TASK_CREATED = 1000,
    EVENT_TASK_STARTED,
    EVENT_TASK_COMPLETED,
    EVENT_TASK_FAILED,
    EVENT_TASK_CANCELLED,
    EVENT_TASK_SUSPENDED,
    EVENT_TASK_RESUMED,

    // 协程事件 (2000-2999)
    EVENT_COROUTINE_CREATED = 2000,
    EVENT_COROUTINE_STARTED,
    EVENT_COROUTINE_SUSPENDED,
    EVENT_COROUTINE_RESUMED,
    EVENT_COROUTINE_COMPLETED,
    EVENT_COROUTINE_FAILED,

    // 内存事件 (3000-3999)
    EVENT_MEMORY_ALLOCATED = 3000,
    EVENT_MEMORY_FREED,
    EVENT_MEMORY_LOW,
    EVENT_GC_STARTED,
    EVENT_GC_COMPLETED,

    // 网络事件 (4000-4999)
    EVENT_NETWORK_CONNECTED = 4000,
    EVENT_NETWORK_DISCONNECTED,
    EVENT_NETWORK_ERROR,
    EVENT_HTTP_REQUEST,
    EVENT_HTTP_RESPONSE,

    // 并发事件 (5000-5999)
    EVENT_MUTEX_LOCKED = 5000,
    EVENT_MUTEX_UNLOCKED,
    EVENT_SEMAPHORE_WAIT,
    EVENT_SEMAPHORE_POST,
    EVENT_DEADLOCK_DETECTED,

    // 性能事件 (6000-6999)
    EVENT_PERFORMANCE_WARNING = 6000,
    EVENT_PERFORMANCE_ERROR,
    EVENT_METRIC_UPDATED,

    // 用户自定义事件 (7000+)
    EVENT_USER_BASE = 7000
} event_type_t;

// 事件数据结构
typedef struct {
    event_type_t type;           // 事件类型
    entity_id_t source_id;       // 事件源ID
    timestamp_t timestamp;       // 事件时间戳
    priority_t priority;         // 事件优先级
    string_t category;           // 事件分类
    string_t message;            // 事件消息
    hash_table_t metadata;       // 事件元数据
    void* payload;               // 事件负载数据
    size_t payload_size;         // 负载数据大小
    cleanup_function_t payload_cleanup; // 负载清理函数
} event_t;

// 事件过滤器
typedef struct {
    event_type_t event_type;     // 事件类型过滤
    entity_id_t source_id;       // 事件源过滤
    string_t category_pattern;   // 分类模式匹配
    priority_t min_priority;     // 最小优先级
    timestamp_t start_time;      // 开始时间
    timestamp_t end_time;        // 结束时间
    hash_table_t metadata_filter; // 元数据过滤
} event_filter_t;

// 事件监听器
typedef struct EventListener {
    entity_id_t listener_id;     // 监听器ID
    string_t name;               // 监听器名称
    event_filter_t filter;       // 事件过滤器
    event_handler_t handler;     // 事件处理器
    void* user_data;             // 用户数据
    cleanup_function_t cleanup;  // 清理函数
    bool is_active;              // 是否激活
    timestamp_t created_at;      // 创建时间
    uint64_t event_count;        // 处理的事件数量
} event_listener_t;

// 事件发布器接口
typedef struct {
    // 发布事件
    bool (*publish_event)(struct EventPublisher* publisher, const event_t* event);
    bool (*publish_event_async)(struct EventPublisher* publisher, const event_t* event);

    // 监听器管理
    bool (*add_listener)(struct EventPublisher* publisher, event_listener_t* listener);
    bool (*remove_listener)(struct EventPublisher* publisher, entity_id_t listener_id);
    bool (*enable_listener)(struct EventPublisher* publisher, entity_id_t listener_id);
    bool (*disable_listener)(struct EventPublisher* publisher, entity_id_t listener_id);

    // 事件查询
    bool (*get_recent_events)(struct EventPublisher* publisher,
                             event_type_t type,
                             size_t max_count,
                             event_t** events,
                             size_t* count);
    bool (*get_events_by_source)(struct EventPublisher* publisher,
                                entity_id_t source_id,
                                size_t max_count,
                                event_t** events,
                                size_t* count);

    // 统计信息
    bool (*get_stats)(const struct EventPublisher* publisher, hash_table_t* stats);
    bool (*reset_stats)(struct EventPublisher* publisher);
} event_publisher_interface_t;

// 事件发布器基类
typedef struct EventPublisher {
    event_publisher_interface_t* vtable;  // 虚函数表
    entity_id_t publisher_id;             // 发布器ID
    string_t name;                        // 发布器名称
    bool is_active;                       // 是否激活
    timestamp_t created_at;               // 创建时间

    // 内部状态
    hash_table_t listeners;               // 监听器表
    queue_t event_queue;                  // 事件队列
    spin_lock_t queue_lock;               // 队列锁
    atomic_int32_t active_listeners;      // 活跃监听器计数
} event_publisher_t;

// 事件系统接口
typedef struct {
    // 发布器管理
    bool (*create_publisher)(struct EventSystem* system,
                            const char* name,
                            event_publisher_t** publisher);
    bool (*destroy_publisher)(struct EventSystem* system,
                             event_publisher_t* publisher);
    bool (*get_publisher)(struct EventSystem* system,
                         const char* name,
                         event_publisher_t** publisher);

    // 全局事件发布
    bool (*publish_global_event)(struct EventSystem* system, const event_t* event);
    bool (*publish_global_event_async)(struct EventSystem* system, const event_t* event);

    // 系统管理
    bool (*start_system)(struct EventSystem* system);
    bool (*stop_system)(struct EventSystem* system);
    bool (*is_system_running)(const struct EventSystem* system);

    // 统计和监控
    bool (*get_system_stats)(const struct EventSystem* system, hash_table_t* stats);
    bool (*get_publisher_stats)(const struct EventSystem* system,
                               const char* publisher_name,
                               hash_table_t* stats);
} event_system_interface_t;

// 事件系统实现
typedef struct EventSystem {
    event_system_interface_t* vtable;     // 虚函数表
    entity_id_t system_id;                // 系统ID
    bool is_running;                      // 是否运行中
    timestamp_t started_at;               // 启动时间

    // 内部状态
    hash_table_t publishers;              // 发布器表
    event_publisher_t* global_publisher;  // 全局发布器
    spin_lock_t system_lock;              // 系统锁
    atomic_int32_t total_events;          // 总事件数
} event_system_t;

// 事件系统工厂函数
event_system_t* event_system_create(void);
void event_system_destroy(event_system_t* system);

// 事件构造函数和析构函数
event_t* event_create(event_type_t type, entity_id_t source_id, const char* message);
void event_destroy(event_t* event);

event_listener_t* event_listener_create(const char* name, event_handler_t handler, void* user_data);
void event_listener_destroy(event_listener_t* listener);

// 过滤器构造函数
event_filter_t* event_filter_create(void);
void event_filter_destroy(event_filter_t* filter);

// 便捷函数
static inline event_system_t* event_system_get_global(void) {
    static event_system_t* global_system = NULL;
    if (!global_system) {
        global_system = event_system_create();
    }
    return global_system;
}

static inline bool event_publish(event_type_t type, entity_id_t source_id, const char* message) {
    event_system_t* system = event_system_get_global();
    if (!system) return false;

    event_t* event = event_create(type, source_id, message);
    if (!event) return false;

    bool result = system->vtable->publish_global_event(system, event);
    event_destroy(event);
    return result;
}

static inline bool event_publish_with_payload(event_type_t type, entity_id_t source_id,
                                             const char* message, void* payload, size_t payload_size) {
    event_system_t* system = event_system_get_global();
    if (!system) return false;

    event_t* event = event_create(type, source_id, message);
    if (!event) return false;

    event->payload = payload;
    event->payload_size = payload_size;

    bool result = system->vtable->publish_global_event(system, event);
    event_destroy(event); // 注意：payload不会被清理，需要调用者管理
    return result;
}

// 宏定义便捷事件发布
#define EVENT_PUBLISH(type, source, msg) event_publish(type, source, msg)
#define EVENT_PUBLISH_PAYLOAD(type, source, msg, payload, size) \
    event_publish_with_payload(type, source, msg, payload, size)

#endif // EVENT_SYSTEM_H
