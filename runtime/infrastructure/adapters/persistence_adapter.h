#ifndef PERSISTENCE_ADAPTER_H
#define PERSISTENCE_ADAPTER_H

#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct PersistenceAdapter;
struct Entity;
struct QueryCriteria;
struct QueryResult;
struct TransactionContext;

// 实体基类
typedef struct Entity {
    uint64_t id;                 // 实体ID
    uint64_t version;            // 版本号（乐观锁）
    time_t created_at;           // 创建时间
    time_t updated_at;           // 更新时间
    char entity_type[128];       // 实体类型
    bool deleted;                // 软删除标志
} Entity;

// 查询条件
typedef struct QueryCriteria {
    char entity_type[128];       // 实体类型
    char* filters;               // 过滤条件（JSON格式）
    char* sorts;                 // 排序条件（JSON格式）
    uint32_t limit;              // 限制数量
    uint32_t offset;             // 偏移量
    bool include_deleted;        // 是否包含已删除的记录
} QueryCriteria;

// 查询结果
typedef struct QueryResult {
    struct Entity** entities;    // 实体数组
    size_t count;                // 实体数量
    size_t total_count;          // 总数量（用于分页）
    bool has_more;               // 是否还有更多数据
    char* metadata;              // 元数据（JSON格式）
} QueryResult;

// 事务上下文
typedef struct TransactionContext {
    void* transaction_handle;    // 事务句柄
    bool is_active;              // 事务是否激活
    time_t started_at;           // 事务开始时间
    uint32_t timeout_ms;         // 事务超时时间
} TransactionContext;

// 持久化适配器接口 - 统一数据访问
typedef struct {
    // 连接管理
    bool (*connect)(struct PersistenceAdapter* adapter, const char* connection_string);
    bool (*disconnect)(struct PersistenceAdapter* adapter);
    bool (*is_connected)(const struct PersistenceAdapter* adapter);
    bool (*ping)(struct PersistenceAdapter* adapter);

    // 实体操作
    bool (*save_entity)(struct PersistenceAdapter* adapter,
                       const struct Entity* entity,
                       struct TransactionContext* tx);
    bool (*find_entity_by_id)(struct PersistenceAdapter* adapter,
                            const char* entity_type,
                            uint64_t id,
                            struct Entity** entity);
    bool (*find_entities)(struct PersistenceAdapter* adapter,
                         const struct QueryCriteria* criteria,
                         struct QueryResult* result);
    bool (*update_entity)(struct PersistenceAdapter* adapter,
                         const struct Entity* entity,
                         struct TransactionContext* tx);
    bool (*delete_entity)(struct PersistenceAdapter* adapter,
                         const char* entity_type,
                         uint64_t id,
                         struct TransactionContext* tx);
    bool (*delete_entities)(struct PersistenceAdapter* adapter,
                           const struct QueryCriteria* criteria,
                           struct TransactionContext* tx,
                           uint64_t* deleted_count);

    // 事务管理
    bool (*begin_transaction)(struct PersistenceAdapter* adapter,
                             struct TransactionContext* tx);
    bool (*commit_transaction)(struct PersistenceAdapter* adapter,
                              struct TransactionContext* tx);
    bool (*rollback_transaction)(struct PersistenceAdapter* adapter,
                                struct TransactionContext* tx);

    // 批量操作
    bool (*batch_save_entities)(struct PersistenceAdapter* adapter,
                               const struct Entity** entities,
                               size_t count,
                               struct TransactionContext* tx);
    bool (*batch_delete_entities)(struct PersistenceAdapter* adapter,
                                 const uint64_t* ids,
                                 size_t count,
                                 const char* entity_type,
                                 struct TransactionContext* tx);

    // 查询优化
    bool (*create_index)(struct PersistenceAdapter* adapter,
                        const char* entity_type,
                        const char* field_name,
                        bool unique);
    bool (*drop_index)(struct PersistenceAdapter* adapter,
                      const char* entity_type,
                      const char* field_name);
    bool (*get_query_plan)(struct PersistenceAdapter* adapter,
                          const struct QueryCriteria* criteria,
                          char* plan_description,
                          size_t description_size);

    // 元数据管理
    bool (*get_schema_info)(struct PersistenceAdapter* adapter,
                           const char* entity_type,
                           char* schema_json,
                           size_t json_size);
    bool (*create_schema)(struct PersistenceAdapter* adapter,
                         const char* entity_type,
                         const char* schema_json);
    bool (*update_schema)(struct PersistenceAdapter* adapter,
                         const char* entity_type,
                         const char* schema_json);
    bool (*migrate_schema)(struct PersistenceAdapter* adapter,
                          const char* entity_type,
                          const char* old_schema_json,
                          const char* new_schema_json);

    // 性能监控
    bool (*get_statistics)(struct PersistenceAdapter* adapter,
                          char* stats_json,
                          size_t json_size);
    bool (*explain_query)(struct PersistenceAdapter* adapter,
                         const struct QueryCriteria* criteria,
                         char* explanation,
                         size_t explanation_size);

    // 缓存管理（如果支持）
    bool (*enable_caching)(struct PersistenceAdapter* adapter);
    bool (*disable_caching)(struct PersistenceAdapter* adapter);
    bool (*clear_cache)(struct PersistenceAdapter* adapter);
    bool (*warm_cache)(struct PersistenceAdapter* adapter, const char* entity_type);
} PersistenceAdapterInterface;

// 持久化适配器基类
typedef struct PersistenceAdapter {
    PersistenceAdapterInterface* vtable;  // 虚函数表

    // 适配器状态
    bool connected;
    char connection_string[2048];
    char adapter_name[128];
    char adapter_version[32];

    // 性能统计
    uint64_t total_operations;
    uint64_t total_latency_ns;
    uint64_t error_count;

    // 连接池
    void* connection_pool;
    uint32_t active_connections;
    uint32_t max_connections;

    // 缓存（如果支持）
    void* cache;
    bool caching_enabled;
} PersistenceAdapter;

// 持久化适配器工厂函数
PersistenceAdapter* create_persistence_adapter(void);
void destroy_persistence_adapter(PersistenceAdapter* adapter);

// 便捷构造函数
struct Entity* entity_create(const char* entity_type, uint64_t id);
void entity_destroy(struct Entity* entity);

struct QueryCriteria* query_criteria_create(const char* entity_type);
void query_criteria_destroy(struct QueryCriteria* criteria);

struct QueryResult* query_result_create(void);
void query_result_destroy(struct QueryResult* result);

struct TransactionContext* transaction_context_create(void);
void transaction_context_destroy(struct TransactionContext* context);

// 便捷函数（直接调用虚函数表）
static inline bool persistence_adapter_save_entity(
    PersistenceAdapter* adapter, const struct Entity* entity, struct TransactionContext* tx) {
    return adapter->vtable->save_entity(adapter, entity, tx);
}

static inline bool persistence_adapter_find_entity_by_id(
    PersistenceAdapter* adapter, const char* entity_type, uint64_t id, struct Entity** entity) {
    return adapter->vtable->find_entity_by_id(adapter, entity_type, id, entity);
}

static inline bool persistence_adapter_find_entities(
    PersistenceAdapter* adapter, const struct QueryCriteria* criteria, struct QueryResult* result) {
    return adapter->vtable->find_entities(adapter, criteria, result);
}

static inline bool persistence_adapter_begin_transaction(
    PersistenceAdapter* adapter, struct TransactionContext* tx) {
    return adapter->vtable->begin_transaction(adapter, tx);
}

#endif // PERSISTENCE_ADAPTER_H
