/**
 * @file industrial_memory_pool.h
 * @brief 工业级异步运行时内存池系统
 *
 * 基于DDD设计，实现多层级内存池架构，支持异步Runtime的高性能内存管理。
 * 核心特性：
 * - 线程本地对象池（无锁设计）
 * - 全局Slab分配器（多尺寸类）
 * - 内存回收与压缩系统
 * - 缓存行对齐优化
 * - NUMA感知分配（如果支持）
 */

#ifndef INDUSTRIAL_MEMORY_POOL_H
#define INDUSTRIAL_MEMORY_POOL_H

#include <stdint.h>
#include <stdbool.h>
#include <stdlib.h>
#include <pthread.h>
#include <stdatomic.h>

// ============================================================================
// 基本类型定义
// ============================================================================

/** 内存池错误码 */
typedef enum {
    POOL_SUCCESS = 0,
    POOL_ERROR_NULL_POINTER = -1,
    POOL_ERROR_INVALID_SIZE = -2,
    POOL_ERROR_OUT_OF_MEMORY = -3,
    POOL_ERROR_THREAD_SAFETY = -4,
    POOL_ERROR_ALIGNMENT = -5,
    POOL_ERROR_DOUBLE_FREE = -6
} pool_error_t;

/** 缓存行大小（64字节） */
#define CACHE_LINE_SIZE 64

/** 对齐到缓存行边界 */
#define CACHE_LINE_ALIGNED __attribute__((aligned(CACHE_LINE_SIZE)))

/** 内存屏障 */
#define MEMORY_BARRIER() __asm__ __volatile__("" ::: "memory")

/** 原子加载 */
#define ATOMIC_LOAD(ptr) atomic_load_explicit(ptr, memory_order_acquire)

/** 原子存储 */
#define ATOMIC_STORE(ptr, val) atomic_store_explicit(ptr, val, memory_order_release)

// ============================================================================
// 大小类定义（基于工业级方案）
// ============================================================================

/** 预定义的大小类（字节） */
#define SIZE_CLASSES_COUNT 12
static const size_t SIZE_CLASSES[SIZE_CLASSES_COUNT] = {
    8,    // Waker, 小Future
    16,   // 原子引用计数
    32,   // 小Task
    64,   // 标准Task
    96,   // Channel节点
    128,  // 大Task
    192,  //
    256,  // 小Future状态机
    384,  //
    512,  // 中Future状态机
    768,  //
    1024  // 大Future状态机
};

/** 大小类索引 */
typedef enum {
    SIZE_CLASS_8 = 0,
    SIZE_CLASS_16,
    SIZE_CLASS_32,
    SIZE_CLASS_64,
    SIZE_CLASS_96,
    SIZE_CLASS_128,
    SIZE_CLASS_192,
    SIZE_CLASS_256,
    SIZE_CLASS_384,
    SIZE_CLASS_512,
    SIZE_CLASS_768,
    SIZE_CLASS_1024
} size_class_t;

// ============================================================================
// 缓存行对齐的数据结构
// ============================================================================

/**
 * @brief 缓存行对齐的包装器
 */
#define CACHE_ALIGNED_TYPE(type) \
    typedef struct { \
        type data; \
        char padding[CACHE_LINE_SIZE - sizeof(type)]; \
    } CACHE_LINE_ALIGNED cache_aligned_##type

/** 缓存行对齐的原子尺寸 */
CACHE_ALIGNED_TYPE(atomic_size_t);

/** 缓存行对齐的原子无符号整数 */
CACHE_ALIGNED_TYPE(atomic_uint_fast32_t);

// ============================================================================
// 对象池配置
// ============================================================================

/**
 * @brief 对象池配置
 */
typedef struct {
    size_t object_size;           // 对象大小（字节）
    size_t initial_capacity;      // 初始容量
    size_t max_capacity;          // 最大容量（0表示无限制）
    bool thread_safe;             // 是否线程安全
    size_t l1_cache_size;         // L1缓存大小（热对象）
    size_t l2_cache_size;         // L2缓存大小（温对象）
    size_t batch_alloc_size;      // 批量分配大小
} object_pool_config_t;

/**
 * @brief 默认对象池配置
 */
#define OBJECT_POOL_DEFAULT_CONFIG(object_size) \
    { \
        (object_size), \
        64, \
        0, \
        true, \
        32, \
        128, \
        16 \
    }

// ============================================================================
// Slab分配器配置
// ============================================================================

/**
 * @brief Slab分配器配置
 */
typedef struct {
    size_t slabs_per_size_class;  // 每个大小类的Slab数量
    bool enable_huge_pages;       // 是否启用大页
    size_t huge_page_size;        // 大页大小（字节）
    bool numa_aware;              // 是否NUMA感知
} slab_allocator_config_t;

/**
 * @brief 默认Slab分配器配置
 */
#define SLAB_ALLOCATOR_DEFAULT_CONFIG() \
    (slab_allocator_config_t){ \
        .slabs_per_size_class = 8, \
        .enable_huge_pages = true, \
        .huge_page_size = 2 * 1024 * 1024, /* 2MB */ \
        .numa_aware = false \
    }

// ============================================================================
// 内存回收配置
// ============================================================================

/**
 * @brief 内存回收配置
 */
typedef struct {
    double gc_threshold;          // GC阈值（内存使用率）
    size_t max_pause_us;          // 最大暂停时间（微秒）
    bool enable_incremental_gc;   // 是否启用增量GC
    bool enable_compaction;       // 是否启用压缩
    size_t compaction_threshold;  // 压缩阈值（碎片率%）
} memory_reclamation_config_t;

/**
 * @brief 默认内存回收配置
 */
#define MEMORY_RECLAMATION_DEFAULT_CONFIG() \
    (memory_reclamation_config_t){ \
        .gc_threshold = 0.8, \
        .max_pause_us = 100, \
        .enable_incremental_gc = true, \
        .enable_compaction = true, \
        .compaction_threshold = 30 \
    }

// ============================================================================
// 线程本地对象池接口
// ============================================================================

/**
 * @brief 线程本地对象池结构体
 */
typedef struct thread_local_object_pool thread_local_object_pool_t;

/**
 * @brief 创建线程本地对象池
 *
 * @param config 池配置
 * @return 池实例，失败返回NULL
 */
thread_local_object_pool_t* thread_local_object_pool_create(const object_pool_config_t* config);

/**
 * @brief 销毁线程本地对象池
 *
 * @param pool 池实例
 */
void thread_local_object_pool_destroy(thread_local_object_pool_t* pool);

/**
 * @brief 从池中分配对象
 *
 * @param pool 池实例
 * @return 分配的对象指针，失败返回NULL
 */
void* thread_local_object_pool_allocate(thread_local_object_pool_t* pool);

/**
 * @brief 释放对象到池中
 *
 * @param pool 池实例
 * @param ptr 要释放的对象指针
 * @return true成功，false失败
 */
bool thread_local_object_pool_deallocate(thread_local_object_pool_t* pool, void* ptr);

/**
 * @brief 获取池统计信息
 *
 * @param pool 池实例
 * @param allocated_count 已分配数量
 * @param free_count 空闲数量
 * @param hit_rate 缓存命中率（0.0-1.0）
 */
void thread_local_object_pool_get_stats(
    const thread_local_object_pool_t* pool,
    size_t* allocated_count,
    size_t* free_count,
    double* hit_rate
);

// ============================================================================
// 全局Slab分配器接口
// ============================================================================

/**
 * @brief 全局Slab分配器结构体
 */
typedef struct global_slab_allocator global_slab_allocator_t;

/**
 * @brief 创建全局Slab分配器
 *
 * @param config 分配器配置
 * @return 分配器实例，失败返回NULL
 */
global_slab_allocator_t* global_slab_allocator_create(const slab_allocator_config_t* config);

/**
 * @brief 销毁全局Slab分配器
 *
 * @param allocator 分配器实例
 */
void global_slab_allocator_destroy(global_slab_allocator_t* allocator);

/**
 * @brief 根据大小分配内存
 *
 * @param allocator 分配器实例
 * @param size 请求的大小
 * @return 分配的内存指针，失败返回NULL
 */
void* global_slab_allocator_allocate(global_slab_allocator_t* allocator, size_t size);

/**
 * @brief 释放内存
 *
 * @param allocator 分配器实例
 * @param ptr 要释放的内存指针
 * @param size 内存大小
 * @return true成功，false失败
 */
bool global_slab_allocator_deallocate(global_slab_allocator_t* allocator, void* ptr, size_t size);

/**
 * @brief 获取分配器统计信息
 *
 * @param allocator 分配器实例
 * @param total_allocated 总分配量
 * @param total_free 总空闲量
 * @param fragmentation_rate 碎片率（0.0-1.0）
 */
void global_slab_allocator_get_stats(
    const global_slab_allocator_t* allocator,
    size_t* total_allocated,
    size_t* total_free,
    double* fragmentation_rate
);

// ============================================================================
// 内存回收器接口
// ============================================================================

/**
 * @brief 内存回收器结构体
 */
typedef struct memory_reclaimer memory_reclaimer_t;

/**
 * @brief 创建内存回收器
 *
 * @param config 回收配置
 * @return 回收器实例，失败返回NULL
 */
memory_reclaimer_t* memory_reclaimer_create(const memory_reclamation_config_t* config);

/**
 * @brief 销毁内存回收器
 *
 * @param reclaimer 回收器实例
 */
void memory_reclaimer_destroy(memory_reclaimer_t* reclaimer);

/**
 * @brief 执行垃圾回收
 *
 * @param reclaimer 回收器实例
 * @param force 是否强制执行（忽略阈值）
 * @return 回收的内存量（字节）
 */
size_t memory_reclaimer_gc(memory_reclaimer_t* reclaimer, bool force);

/**
 * @brief 执行内存压缩
 *
 * @param reclaimer 回收器实例
 * @return 压缩的内存量（字节）
 */
size_t memory_reclaimer_compact(memory_reclaimer_t* reclaimer);

/**
 * @brief 获取回收器统计信息
 *
 * @param reclaimer 回收器实例
 * @param gc_count GC执行次数
 * @param total_reclaimed 总回收量
 * @param avg_pause_us 平均暂停时间（微秒）
 */
void memory_reclaimer_get_stats(
    const memory_reclaimer_t* reclaimer,
    uint64_t* gc_count,
    size_t* total_reclaimed,
    double* avg_pause_us
);

// ============================================================================
// 工业级内存池系统接口
// ============================================================================

/**
 * @brief 工业级内存池系统结构体
 */
typedef struct industrial_memory_pool industrial_memory_pool_t;

/**
 * @brief 内存池系统配置
 */
typedef struct {
    object_pool_config_t task_pool_config;           // Task对象池配置
    object_pool_config_t waker_pool_config;          // Waker对象池配置
    object_pool_config_t channel_node_pool_config;   // Channel节点池配置
    slab_allocator_config_t slab_config;             // Slab分配器配置
    memory_reclamation_config_t reclamation_config;  // 回收配置
    size_t max_threads;                              // 最大线程数
} industrial_memory_pool_config_t;

/**
 * @brief 默认工业级内存池配置
 */
#define INDUSTRIAL_MEMORY_POOL_DEFAULT_CONFIG() \
    { \
        OBJECT_POOL_DEFAULT_CONFIG(128), \
        OBJECT_POOL_DEFAULT_CONFIG(64), \
        OBJECT_POOL_DEFAULT_CONFIG(96), \
        SLAB_ALLOCATOR_DEFAULT_CONFIG(), \
        MEMORY_RECLAMATION_DEFAULT_CONFIG(), \
        64 \
    }

/**
 * @brief 创建工业级内存池系统
 *
 * @param config 系统配置
 * @return 内存池系统实例，失败返回NULL
 */
industrial_memory_pool_t* industrial_memory_pool_create(const industrial_memory_pool_config_t* config);

/**
 * @brief 销毁工业级内存池系统
 *
 * @param pool 系统实例
 */
void industrial_memory_pool_destroy(industrial_memory_pool_t* pool);

/**
 * @brief 分配Task对象
 *
 * @param pool 系统实例
 * @return 分配的Task对象指针，失败返回NULL
 */
void* industrial_memory_pool_allocate_task(industrial_memory_pool_t* pool);

/**
 * @brief 释放Task对象
 *
 * @param pool 系统实例
 * @param task Task对象指针
 * @return true成功，false失败
 */
bool industrial_memory_pool_deallocate_task(industrial_memory_pool_t* pool, void* task);

/**
 * @brief 分配Waker对象
 *
 * @param pool 系统实例
 * @return 分配的Waker对象指针，失败返回NULL
 */
void* industrial_memory_pool_allocate_waker(industrial_memory_pool_t* pool);

/**
 * @brief 释放Waker对象
 *
 * @param pool 系统实例
 * @param waker Waker对象指针
 * @return true成功，false失败
 */
bool industrial_memory_pool_deallocate_waker(industrial_memory_pool_t* pool, void* waker);

/**
 * @brief 分配Channel节点
 *
 * @param pool 系统实例
 * @return 分配的Channel节点指针，失败返回NULL
 */
void* industrial_memory_pool_allocate_channel_node(industrial_memory_pool_t* pool);

/**
 * @brief 释放Channel节点
 *
 * @param pool 系统实例
 * @param node Channel节点指针
 * @return true成功，false失败
 */
bool industrial_memory_pool_deallocate_channel_node(industrial_memory_pool_t* pool, void* node);

/**
 * @brief 通用内存分配（使用Slab分配器）
 *
 * @param pool 系统实例
 * @param size 请求的大小
 * @return 分配的内存指针，失败返回NULL
 */
void* industrial_memory_pool_allocate(industrial_memory_pool_t* pool, size_t size);

/**
 * @brief 通用内存释放
 *
 * @param pool 系统实例
 * @param ptr 要释放的内存指针
 * @param size 内存大小
 * @return true成功，false失败
 */
bool industrial_memory_pool_deallocate(industrial_memory_pool_t* pool, void* ptr, size_t size);

/**
 * @brief 触发垃圾回收
 *
 * @param pool 系统实例
 * @return 回收的内存量（字节）
 */
size_t industrial_memory_pool_gc(industrial_memory_pool_t* pool);

/**
 * @brief 获取系统统计信息
 *
 * @param pool 系统实例
 * @param total_allocated 总分配量
 * @param total_free 总空闲量
 * @param fragmentation_rate 碎片率
 * @param gc_count GC执行次数
 */
void industrial_memory_pool_get_stats(
    const industrial_memory_pool_t* pool,
    size_t* total_allocated,
    size_t* total_free,
    double* fragmentation_rate,
    uint64_t* gc_count
);

/**
 * @brief 获取最后错误码
 *
 * @return 错误码
 */
pool_error_t industrial_memory_pool_get_last_error(void);

#endif // INDUSTRIAL_MEMORY_POOL_H
