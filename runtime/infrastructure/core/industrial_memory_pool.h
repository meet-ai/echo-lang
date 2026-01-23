#ifndef INDUSTRIAL_MEMORY_POOL_H
#define INDUSTRIAL_MEMORY_POOL_H

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>

// 前向声明
struct IndustrialMemoryPool;
struct MemoryBlock;
struct AllocationStats;

// 内存块头部（用于调试和统计）
typedef struct MemoryBlockHeader {
    size_t block_size;           // 块大小
    uint64_t allocation_id;      // 分配ID
    time_t allocated_at;         // 分配时间
    char allocation_source[128]; // 分配源（文件名:行号）
    uint32_t magic_number;       // 魔数，用于检测内存损坏
} MemoryBlockHeader;

// 内存池配置
typedef struct MemoryPoolConfig {
    size_t initial_pool_size;    // 初始池大小
    size_t max_pool_size;        // 最大池大小
    size_t block_size;           // 块大小
    size_t alignment;            // 对齐大小
    uint32_t preallocate_blocks; // 预分配块数量
    bool thread_safe;            // 是否线程安全
    bool enable_stats;           // 是否启用统计
    bool enable_leak_detection;  // 是否启用泄漏检测
    uint32_t max_free_blocks;    // 最大空闲块数量
} MemoryPoolConfig;

// 分配统计信息
typedef struct AllocationStats {
    uint64_t total_allocations;   // 总分配次数
    uint64_t total_deallocations; // 总释放次数
    uint64_t current_allocations; // 当前分配数量
    uint64_t peak_allocations;    // 峰值分配数量
    uint64_t total_memory_used;   // 总内存使用量
    uint64_t peak_memory_used;    // 峰值内存使用量
    uint64_t fragmentation_bytes; // 碎片字节数
    double fragmentation_ratio;   // 碎片率
    uint64_t allocation_failures; // 分配失败次数
    uint64_t corruption_detections; // 损坏检测次数
} AllocationStats;

// 内存池接口
typedef struct {
    // 内存分配
    void* (*allocate)(struct IndustrialMemoryPool* pool, size_t size, const char* source);
    void* (*allocate_aligned)(struct IndustrialMemoryPool* pool, size_t size, size_t alignment, const char* source);
    void* (*reallocate)(struct IndustrialMemoryPool* pool, void* ptr, size_t new_size, const char* source);

    // 内存释放
    void (*deallocate)(struct IndustrialMemoryPool* pool, void* ptr);

    // 内存池管理
    bool (*expand)(struct IndustrialMemoryPool* pool, size_t additional_size);
    bool (*shrink)(struct IndustrialMemoryPool* pool, size_t reduce_size);
    bool (*compact)(struct IndustrialMemoryPool* pool);

    // 统计和监控
    bool (*get_stats)(const struct IndustrialMemoryPool* pool, struct AllocationStats* stats);
    bool (*reset_stats)(struct IndustrialMemoryPool* pool);
    bool (*check_integrity)(const struct IndustrialMemoryPool* pool);

    // 泄漏检测
    bool (*enable_leak_detection)(struct IndustrialMemoryPool* pool);
    bool (*disable_leak_detection)(struct IndustrialMemoryPool* pool);
    bool (*detect_leaks)(const struct IndustrialMemoryPool* pool, char* report, size_t report_size);

    // 调试支持
    bool (*dump_state)(const struct IndustrialMemoryPool* pool, const char* filename);
    bool (*validate_pointer)(const struct IndustrialMemoryPool* pool, const void* ptr);

    // 性能优化
    bool (*warm_up)(struct IndustrialMemoryPool* pool, size_t warm_up_size);
    bool (*optimize_layout)(struct IndustrialMemoryPool* pool);
} IndustrialMemoryPoolInterface;

// 工业内存池实现
typedef struct IndustrialMemoryPool {
    IndustrialMemoryPoolInterface* vtable;  // 虚函数表

    // 配置
    struct MemoryPoolConfig config;

    // 状态
    bool initialized;
    size_t current_pool_size;
    size_t used_memory;
    uint32_t free_blocks_count;
    uint32_t total_blocks_count;

    // 内部数据结构
    void* free_list;             // 空闲链表
    void* block_list;            // 块链表
    void* allocation_map;        // 分配映射表

    // 同步原语
    void* mutex;                 // 互斥锁

    // 统计信息
    struct AllocationStats stats;

    // 调试信息
    bool leak_detection_enabled;
    void* allocation_records;    // 分配记录
} IndustrialMemoryPool;

// 内存池工厂函数
IndustrialMemoryPool* industrial_memory_pool_create(const struct MemoryPoolConfig* config);
void industrial_memory_pool_destroy(IndustrialMemoryPool* pool);

// 便捷构造函数
struct MemoryPoolConfig* memory_pool_config_create(void);
void memory_pool_config_destroy(struct MemoryPoolConfig* config);

// 便捷配置生成
struct MemoryPoolConfig* memory_pool_config_create_default(void);
struct MemoryPoolConfig* memory_pool_config_create_high_performance(void);
struct MemoryPoolConfig* memory_pool_config_create_low_latency(void);
struct MemoryPoolConfig* memory_pool_config_create_debug(void);

// 便捷函数（直接调用虚函数表）
static inline void* industrial_memory_pool_allocate(
    IndustrialMemoryPool* pool, size_t size, const char* source) {
    return pool->vtable->allocate(pool, size, source);
}

static inline void industrial_memory_pool_deallocate(IndustrialMemoryPool* pool, void* ptr) {
    pool->vtable->deallocate(pool, ptr);
}

static inline bool industrial_memory_pool_get_stats(
    const IndustrialMemoryPool* pool, struct AllocationStats* stats) {
    return pool->vtable->get_stats(pool, stats);
}

// 全局内存池实例（可选）
extern IndustrialMemoryPool* g_global_memory_pool;

// 全局便捷函数
static inline void* industrial_memory_pool_allocate_global(size_t size, const char* source) {
    if (g_global_memory_pool) {
        return industrial_memory_pool_allocate(g_global_memory_pool, size, source);
    }
    return NULL;
}

static inline void industrial_memory_pool_free_global(void* ptr) {
    if (g_global_memory_pool) {
        industrial_memory_pool_deallocate(g_global_memory_pool, ptr);
    }
}

#endif // INDUSTRIAL_MEMORY_POOL_H
