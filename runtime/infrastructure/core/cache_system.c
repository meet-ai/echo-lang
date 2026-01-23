#include "cache_system.h"
#include "../../shared/types/common_types.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <pthread.h>
#include <time.h>

// 缓存项
typedef struct CacheItem {
    char key[256];              // 缓存键
    void* value;                // 缓存值
    size_t value_size;          // 值大小
    time_t created_at;          // 创建时间
    time_t accessed_at;         // 最后访问时间
    time_t expires_at;          // 过期时间
    uint32_t access_count;      // 访问次数
    uint32_t hit_count;         // 命中次数
    struct CacheItem* prev;     // LRU链表前驱
    struct CacheItem* next;     // LRU链表后继
    struct CacheItem* hash_next; // 哈希表链表
} CacheItem;

// LRU链表
typedef struct LRUList {
    CacheItem* head;            // 头节点（最近访问）
    CacheItem* tail;            // 尾节点（最久未访问）
    size_t count;               // 节点数量
} LRUList;

// 哈希表
typedef struct HashTable {
    CacheItem** buckets;        // 桶数组
    size_t bucket_count;        // 桶数量
    hash_function_t hash_func;  // 哈希函数
} HashTable;

// 缓存系统内部实现
typedef struct CacheSystemImpl {
    CacheSystemInterface base;  // 虚函数表

    // 配置
    CacheConfig config;

    // 存储结构
    HashTable hash_table;       // 哈希表
    LRUList lru_list;           // LRU链表

    // 统计信息
    CacheStats stats;

    // 同步
    pthread_mutex_t mutex;
    pthread_rwlock_t rwlock;    // 读写锁，允许多个读取者

    // 控制
    bool initialized;
    bool cleanup_running;
} CacheSystemImpl;

// 哈希函数
static size_t default_hash_function(const char* key) {
    size_t hash = 0;
    while (*key) {
        hash = (hash * 31) + (unsigned char)*key;
        key++;
    }
    return hash;
}

// 初始化LRU链表
static void lru_list_init(LRUList* list) {
    list->head = NULL;
    list->tail = NULL;
    list->count = 0;
}

// 初始化哈希表
static bool hash_table_init(HashTable* table, size_t bucket_count) {
    table->buckets = (CacheItem**)calloc(bucket_count, sizeof(CacheItem*));
    if (!table->buckets) return false;

    table->bucket_count = bucket_count;
    table->hash_func = default_hash_function;
    return true;
}

// 销毁哈希表
static void hash_table_destroy(HashTable* table) {
    if (table->buckets) {
        free(table->buckets);
        table->buckets = NULL;
    }
}

// LRU链表操作
static void lru_list_move_to_head(LRUList* list, CacheItem* item) {
    if (list->head == item) return;

    // 从当前位置移除
    if (item->prev) item->prev->next = item->next;
    if (item->next) item->next->prev = item->prev;

    // 如果是尾节点，更新尾节点
    if (list->tail == item) {
        list->tail = item->prev;
    }

    // 移到头部
    item->next = list->head;
    item->prev = NULL;

    if (list->head) {
        list->head->prev = item;
    } else {
        list->tail = item;
    }

    list->head = item;
}

static void lru_list_remove(LRUList* list, CacheItem* item) {
    if (item->prev) item->prev->next = item->next;
    if (item->next) item->next->prev = item->prev;

    if (list->head == item) list->head = item->next;
    if (list->tail == item) list->tail = item->prev;

    list->count--;
}

static CacheItem* lru_list_remove_tail(LRUList* list) {
    if (!list->tail) return NULL;

    CacheItem* item = list->tail;
    lru_list_remove(list, item);
    return item;
}

// 哈希表操作
static CacheItem* hash_table_get(const HashTable* table, const char* key) {
    size_t hash = table->hash_func(key);
    size_t bucket = hash % table->bucket_count;

    CacheItem* item = table->buckets[bucket];
    while (item) {
        if (strcmp(item->key, key) == 0) {
            return item;
        }
        item = item->hash_next;
    }
    return NULL;
}

static bool hash_table_put(HashTable* table, CacheItem* item) {
    size_t hash = table->hash_func(item->key);
    size_t bucket = hash % table->bucket_count;

    // 检查是否已存在
    CacheItem* existing = hash_table_get(table, item->key);
    if (existing) {
        // 移除旧项
        hash_table_remove(table, item->key);
    }

    // 插入新项
    item->hash_next = table->buckets[bucket];
    table->buckets[bucket] = item;

    return true;
}

static bool hash_table_remove(HashTable* table, const char* key) {
    size_t hash = table->hash_func(key);
    size_t bucket = hash % table->bucket_count;

    CacheItem* prev = NULL;
    CacheItem* current = table->buckets[bucket];

    while (current) {
        if (strcmp(current->key, key) == 0) {
            if (prev) {
                prev->hash_next = current->hash_next;
            } else {
                table->buckets[bucket] = current->hash_next;
            }
            return true;
        }
        prev = current;
        current = current->hash_next;
    }
    return false;
}

// 创建缓存项
static CacheItem* create_cache_item(const char* key, const void* value,
                                   size_t value_size, time_t ttl_seconds) {
    CacheItem* item = (CacheItem*)malloc(sizeof(CacheItem));
    if (!item) return NULL;

    memset(item, 0, sizeof(CacheItem));
    strncpy(item->key, key, sizeof(item->key) - 1);
    item->key[sizeof(item->key) - 1] = '\0';

    if (value && value_size > 0) {
        item->value = malloc(value_size);
        if (!item->value) {
            free(item);
            return NULL;
        }
        memcpy(item->value, value, value_size);
        item->value_size = value_size;
    }

    time_t now = time(NULL);
    item->created_at = now;
    item->accessed_at = now;
    item->expires_at = ttl_seconds > 0 ? now + ttl_seconds : 0;

    return item;
}

// 销毁缓存项
static void destroy_cache_item(CacheItem* item) {
    if (item) {
        if (item->value) {
            free(item->value);
        }
        free(item);
    }
}

// 检查缓存项是否过期
static bool is_cache_item_expired(const CacheItem* item) {
    if (item->expires_at == 0) return false;
    return time(NULL) > item->expires_at;
}

// 缓存系统接口实现

// 设置缓存
static bool cache_system_set(CacheSystem* cache, const char* key, const void* value,
                           size_t value_size, time_t ttl_seconds) {
    if (!cache || !key || !value || value_size == 0) return false;

    CacheSystemImpl* impl = (CacheSystemImpl*)cache;

    if (!impl->initialized) return false;

    CacheItem* item = create_cache_item(key, value, value_size, ttl_seconds);
    if (!item) return false;

    pthread_rwlock_wrlock(&impl->rwlock);

    // 检查容量限制
    if (impl->lru_list.count >= impl->config.max_entries) {
        // 移除最久未使用的项
        CacheItem* victim = lru_list_remove_tail(&impl->lru_list);
        if (victim) {
            hash_table_remove(&impl->hash_table, victim->key);
            impl->stats.evictions++;
            destroy_cache_item(victim);
        }
    }

    // 添加新项
    hash_table_put(&impl->hash_table, item);
    lru_list_move_to_head(&impl->lru_list, item);
    impl->lru_list.count++;

    // 更新统计
    impl->stats.entries++;
    impl->stats.total_sets++;

    pthread_rwlock_unlock(&impl->rwlock);

    return true;
}

// 获取缓存
static bool cache_system_get(CacheSystem* cache, const char* key,
                           void** value, size_t* value_size) {
    if (!cache || !key || !value || !value_size) return false;

    CacheSystemImpl* impl = (CacheSystemImpl*)cache;

    if (!impl->initialized) return false;

    pthread_rwlock_rdlock(&impl->rwlock);

    CacheItem* item = hash_table_get(&impl->hash_table, key);
    if (!item) {
        pthread_rwlock_unlock(&impl->rwlock);
        impl->stats.misses++;
        return false;
    }

    // 检查是否过期
    if (is_cache_item_expired(item)) {
        // 过期，标记为待清理
        item->expires_at = 0; // 标记为过期
        pthread_rwlock_unlock(&impl->rwlock);
        impl->stats.misses++;
        return false;
    }

    // 更新访问信息
    item->accessed_at = time(NULL);
    item->access_count++;
    item->hit_count++;

    // 移动到LRU头部
    lru_list_move_to_head(&impl->lru_list, item);

    // 返回值
    if (item->value && item->value_size > 0) {
        *value = malloc(item->value_size);
        if (!*value) {
            pthread_rwlock_unlock(&impl->rwlock);
            return false;
        }
        memcpy(*value, item->value, item->value_size);
        *value_size = item->value_size;
    }

    pthread_rwlock_unlock(&impl->rwlock);

    impl->stats.hits++;
    return true;
}

// 删除缓存
static bool cache_system_delete(CacheSystem* cache, const char* key) {
    if (!cache || !key) return false;

    CacheSystemImpl* impl = (CacheSystemImpl*)cache;

    if (!impl->initialized) return false;

    pthread_rwlock_wrlock(&impl->rwlock);

    CacheItem* item = hash_table_get(&impl->hash_table, key);
    if (item) {
        hash_table_remove(&impl->hash_table, item->key);
        lru_list_remove(&impl->lru_list, item);

        impl->stats.entries--;
        impl->stats.total_deletes++;

        destroy_cache_item(item);

        pthread_rwlock_unlock(&impl->rwlock);
        return true;
    }

    pthread_rwlock_unlock(&impl->rwlock);
    return false;
}

// 检查缓存存在
static bool cache_system_exists(CacheSystem* cache, const char* key) {
    if (!cache || !key) return false;

    CacheSystemImpl* impl = (CacheSystemImpl*)cache;

    if (!impl->initialized) return false;

    pthread_rwlock_rdlock(&impl->rwlock);
    CacheItem* item = hash_table_get(&impl->hash_table, key);
    bool exists = item && !is_cache_item_expired(item);
    pthread_rwlock_unlock(&impl->rwlock);

    return exists;
}

// 清空缓存
static bool cache_system_clear(CacheSystem* cache) {
    if (!cache) return false;

    CacheSystemImpl* impl = (CacheSystemImpl*)cache;

    if (!impl->initialized) return false;

    pthread_rwlock_wrlock(&impl->rwlock);

    // 清空LRU链表
    CacheItem* current = impl->lru_list.head;
    while (current) {
        CacheItem* next = current->next;
        destroy_cache_item(current);
        current = next;
    }

    // 清空哈希表
    for (size_t i = 0; i < impl->hash_table.bucket_count; i++) {
        impl->hash_table.buckets[i] = NULL;
    }

    // 重置LRU链表
    lru_list_init(&impl->lru_list);

    // 更新统计
    impl->stats.entries = 0;
    impl->stats.total_clears++;

    pthread_rwlock_unlock(&impl->rwlock);

    return true;
}

// 获取统计信息
static bool cache_system_get_stats(const CacheSystem* cache, CacheStats* stats) {
    if (!cache || !stats) return false;

    const CacheSystemImpl* impl = (CacheSystemImpl*)cache;
    *stats = impl->stats;
    return true;
}

// 创建缓存系统
CacheSystem* cache_system_create(const CacheConfig* config) {
    if (!config) return NULL;

    CacheSystemImpl* impl = (CacheSystemImpl*)malloc(sizeof(CacheSystemImpl));
    if (!impl) return NULL;

    memset(impl, 0, sizeof(CacheSystemImpl));
    impl->config = *config;

    // 初始化LRU链表
    lru_list_init(&impl->lru_list);

    // 初始化哈希表
    size_t bucket_count = config->max_entries / 4; // 负载因子0.25
    if (bucket_count < 16) bucket_count = 16;
    if (!hash_table_init(&impl->hash_table, bucket_count)) {
        free(impl);
        return NULL;
    }

    // 初始化同步原语
    pthread_rwlock_init(&impl->rwlock, NULL);

    // 初始化统计
    memset(&impl->stats, 0, sizeof(CacheStats));
    impl->stats.created_at = time(NULL);

    // 设置虚函数表
    impl->base.set = cache_system_set;
    impl->base.get = cache_system_get;
    impl->base.delete = cache_system_delete;
    impl->base.exists = cache_system_exists;
    impl->base.clear = cache_system_clear;
    impl->base.get_stats = cache_system_get_stats;

    impl->initialized = true;

    return (CacheSystem*)impl;
}

// 销毁缓存系统
void cache_system_destroy(CacheSystem* cache) {
    if (!cache) return;

    CacheSystemImpl* impl = (CacheSystemImpl*)cache;

    if (impl->initialized) {
        // 清空所有缓存项
        cache_system_clear(cache);

        // 销毁同步原语
        pthread_rwlock_destroy(&impl->rwlock);

        // 销毁哈希表
        hash_table_destroy(&impl->hash_table);
    }

    free(impl);
}

// 便捷构造函数
CacheConfig* cache_config_create(void) {
    CacheConfig* config = (CacheConfig*)malloc(sizeof(CacheConfig));
    if (config) {
        memset(config, 0, sizeof(CacheConfig));
    }
    return config;
}

void cache_config_destroy(CacheConfig* config) {
    free(config);
}

CacheConfig* cache_config_create_default(void) {
    CacheConfig* config = cache_config_create();
    if (config) {
        config->max_entries = 10000;
        config->ttl_seconds = 3600; // 1小时
        config->max_memory_mb = 256;
        config->eviction_policy = CACHE_EVICTION_LRU;
        config->compression_enabled = false;
        config->stats_enabled = true;
    }
    return config;
}
