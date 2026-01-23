#ifndef CACHE_SYSTEM_H
#define CACHE_SYSTEM_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>

// 前向声明
struct CacheSystem;
struct CacheEntry;
struct CacheConfig;
struct CacheStats;

// 缓存项
typedef struct CacheEntry {
    char* key;                   // 缓存键
    void* value;                 // 缓存值
    size_t value_size;           // 值大小
    time_t created_at;           // 创建时间
    time_t accessed_at;          // 最后访问时间
    time_t expires_at;           // 过期时间
    uint32_t access_count;       // 访问次数
    uint32_t hit_count;          // 命中次数
    char* metadata;              // 元数据（JSON格式）
    size_t metadata_size;        // 元数据大小
} CacheEntry;

// 缓存配置
typedef struct CacheConfig {
    size_t max_memory_size;      // 最大内存大小
    uint32_t max_entries;        // 最大条目数
    uint32_t default_ttl_seconds; // 默认TTL
    char eviction_policy[32];    // 驱逐策略 ("lru", "lfu", "fifo", "random")
    bool compression_enabled;    // 是否启用压缩
    bool persistence_enabled;    // 是否启用持久化
    char persistence_file[512];  // 持久化文件路径
    uint32_t persistence_interval_seconds; // 持久化间隔
    bool statistics_enabled;     // 是否启用统计
    uint32_t cleanup_interval_seconds; // 清理间隔
} CacheConfig;

// 缓存统计
typedef struct CacheStats {
    uint64_t total_requests;     // 总请求数
    uint64_t cache_hits;         // 缓存命中数
    uint64_t cache_misses;       // 缓存未命中数
    double hit_ratio;            // 命中率
    uint32_t current_entries;    // 当前条目数
    size_t current_memory_used;  // 当前内存使用量
    uint64_t total_evictions;    // 总驱逐次数
    uint64_t total_expirations;  // 总过期次数
    uint64_t total_sets;         // 总设置次数
    uint64_t total_deletes;      // 总删除次数
    time_t last_cleanup_at;      // 最后清理时间
    time_t uptime_seconds;       // 运行时间
} CacheStats;

// 缓存系统接口
typedef struct {
    // 基本操作
    bool (*set)(struct CacheSystem* cache, const char* key, const void* value,
                size_t value_size, uint32_t ttl_seconds);
    bool (*get)(struct CacheSystem* cache, const char* key, void** value, size_t* value_size);
    bool (*delete)(struct CacheSystem* cache, const char* key);
    bool (*exists)(struct CacheSystem* cache, const char* key);
    bool (*expire)(struct CacheSystem* cache, const char* key, uint32_t ttl_seconds);

    // 批量操作
    bool (*set_multiple)(struct CacheSystem* cache, const char* const* keys,
                        const void* const* values, const size_t* value_sizes,
                        const uint32_t* ttl_seconds, size_t count);
    bool (*get_multiple)(struct CacheSystem* cache, const char* const* keys,
                        void*** values, size_t** value_sizes, size_t count);
    bool (*delete_multiple)(struct CacheSystem* cache, const char* const* keys, size_t count);

    // 原子操作
    bool (*increment)(struct CacheSystem* cache, const char* key, int64_t delta, int64_t* result);
    bool (*decrement)(struct CacheSystem* cache, const char* key, int64_t delta, int64_t* result);
    bool (*append)(struct CacheSystem* cache, const char* key, const void* data, size_t data_size);

    // 模式匹配
    bool (*keys)(struct CacheSystem* cache, const char* pattern, char*** keys, size_t* count);
    bool (*scan)(struct CacheSystem* cache, const char* pattern, uint32_t cursor,
                char*** keys, uint32_t* new_cursor, size_t count);

    // 管理操作
    bool (*clear)(struct CacheSystem* cache);
    bool (*cleanup)(struct CacheSystem* cache);  // 清理过期条目
    bool (*optimize)(struct CacheSystem* cache); // 优化缓存布局

    // 持久化
    bool (*save)(struct CacheSystem* cache, const char* filename);
    bool (*load)(struct CacheSystem* cache, const char* filename);

    // 统计和监控
    bool (*get_stats)(const struct CacheSystem* cache, struct CacheStats* stats);
    bool (*reset_stats)(struct CacheSystem* cache);
    bool (*get_entry_info)(const struct CacheSystem* cache, const char* key, struct CacheEntry* entry);

    // 配置管理
    bool (*reconfigure)(struct CacheSystem* cache, const struct CacheConfig* new_config);
    bool (*get_config)(const struct CacheSystem* cache, struct CacheConfig* config);
} CacheSystemInterface;

// 缓存系统实现
typedef struct CacheSystem {
    CacheSystemInterface* vtable;  // 虚函数表

    // 配置
    struct CacheConfig config;

    // 状态
    bool initialized;
    time_t created_at;
    char cache_name[128];

    // 内部数据结构
    void* hash_table;            // 哈希表
    void* lru_list;              // LRU链表
    void* expiration_heap;       // 过期堆
    void* access_stats;          // 访问统计

    // 同步
    void* mutex;

    // 统计
    struct CacheStats stats;

    // 持久化
    bool persistence_enabled;
    time_t last_save_at;
} CacheSystem;

// 缓存系统工厂函数
CacheSystem* cache_system_create(const struct CacheConfig* config);
void cache_system_destroy(CacheSystem* cache);

// 便捷构造函数
struct CacheConfig* cache_config_create(void);
void cache_config_destroy(struct CacheConfig* config);

// 预定义配置
struct CacheConfig* cache_config_create_lru(size_t max_memory);
struct CacheConfig* cache_config_create_lfu(size_t max_memory);
struct CacheConfig* cache_config_create_high_performance(void);
struct CacheConfig* cache_config_create_persistent(const char* file_path);

// 便捷函数（直接调用虚函数表）
static inline bool cache_system_set(CacheSystem* cache, const char* key,
                                   const void* value, size_t value_size, uint32_t ttl_seconds) {
    return cache->vtable->set(cache, key, value, value_size, ttl_seconds);
}

static inline bool cache_system_get(CacheSystem* cache, const char* key,
                                   void** value, size_t* value_size) {
    return cache->vtable->get(cache, key, value, value_size);
}

static inline bool cache_system_delete(CacheSystem* cache, const char* key) {
    return cache->vtable->delete(cache, key);
}

static inline bool cache_system_get_stats(const CacheSystem* cache, struct CacheStats* stats) {
    return cache->vtable->get_stats(cache, stats);
}

// 全局缓存实例（可选）
extern CacheSystem* g_global_cache;

// 全局便捷函数
static inline bool cache_system_set_global(const char* key, const void* value,
                                          size_t value_size, uint32_t ttl_seconds) {
    if (g_global_cache) {
        return cache_system_set(g_global_cache, key, value, value_size, ttl_seconds);
    }
    return false;
}

static inline bool cache_system_get_global(const char* key, void** value, size_t* value_size) {
    if (g_global_cache) {
        return cache_system_get(g_global_cache, key, value, value_size);
    }
    return false;
}

#endif // CACHE_SYSTEM_H
