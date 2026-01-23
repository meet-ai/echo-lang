/**
 * @file advanced_memory_pool.h
 * @brief 工业级异步运行时内存池系统
 *
 * 基于DDD领域驱动设计的工业级内存池实现，包含：
 * - 线程本地无锁分配（极致性能）
 * - Slab大小类分配器（内存效率）
 * - 异步对象特化优化（Future/Task/Waker）
 * - 缓存行对齐和NUMA感知
 */

#ifndef ADVANCED_MEMORY_POOL_H
#define ADVANCED_MEMORY_POOL_H

#include <stdint.h>
#include <stdbool.h>
#include <stdlib.h>
#include <pthread.h>

// ============================================================================
// 架构常量定义
// ============================================================================

/** 缓存行大小（64字节对齐） */
#define CACHE_LINE_SIZE 64

/** Slab大小类定义（8B到1024B） */
#define NUM_SIZE_CLASSES 12
extern const size_t SIZE_CLASSES[NUM_SIZE_CLASSES];

/** 异步对象专用大小类 */
#define SIZE_CLASS_FUTURE      3    // 128B Future
#define SIZE_CLASS_TASK        4    // 192B Task
#define SIZE_CLASS_WAKER       2    // 64B Waker
#define SIZE_CLASS_CHANNEL     3    // 128B Channel Node

// ============================================================================
// 核心数据结构
// ============================================================================

/**
 * @brief 缓存行对齐的原子类型
 */
typedef struct {
    volatile uint64_t value;
    char _padding[CACHE_LINE_SIZE - sizeof(uint64_t)];
} cache_aligned_uint64_t;

/**
 * @brief 缓存行对齐的指针
 */
typedef struct {
    void* ptr;
    char _padding[CACHE_LINE_SIZE - sizeof(void*)];
} cache_aligned_ptr_t;

/**
 * @brief 内存块头部（用于空闲链表）
 */
typedef struct block_header {
    struct block_header* next;  // 下一个空闲块
    uint32_t size_class;        // 大小类索引
    uint32_t magic;             // 调试魔数
} block_header_t;

/**
 * @brief Slab分配器（固定大小类）
 */
typedef struct slab_allocator {
    size_t block_size;          // 块大小
    size_t total_blocks;        // 总块数
    size_t free_blocks;         // 空闲块数
    block_header_t* free_list;  // 空闲链表
    void* memory;               // 内存区域
    pthread_mutex_t lock;       // 保护锁
} slab_allocator_t;

/**
 * @brief 线程本地对象池（三层缓存架构）
 */
typedef struct thread_local_pool {
    // L1热缓存（FixedSizeStack LIFO）
    void** hot_cache;
    size_t hot_size;
    size_t hot_capacity;

    // L2温缓存（SegregatedList）
    void** warm_cache;
    size_t warm_size;
    size_t warm_capacity;

    // L3冷后备（Slab分配器）
    slab_allocator_t* cold_slab;

    // 统计信息
    uint64_t allocations;
    uint64_t deallocations;
    uint64_t cache_hits_l1;
    uint64_t cache_hits_l2;

    // 线程ID（用于调试）
    pthread_t thread_id;
} thread_local_pool_t;

/**
 * @brief 全局内存池系统
 */
typedef struct advanced_memory_pool {
    // Slab分配器数组（按大小类）
    slab_allocator_t* slabs[NUM_SIZE_CLASSES];

    // 线程本地池（Per-CPU）
    thread_local_pool_t** thread_pools;
    size_t num_threads;

    // 全局统计
    cache_aligned_uint64_t total_allocations;
    cache_aligned_uint64_t total_deallocations;
    cache_aligned_uint64_t total_memory_used;

    // 配置
    size_t max_memory_per_thread;
    bool enable_numa_aware;
    bool enable_huge_pages;

    // 锁
    pthread_mutex_t global_lock;
} advanced_memory_pool_t;

/**
 * @brief 异步对象专用内存池
 */
typedef struct async_memory_pool {
    advanced_memory_pool_t* base_pool;

    // Future专用池（支持内联存储）
    thread_local_pool_t* future_pool;

    // Task专用池（头部分离优化）
    thread_local_pool_t* task_header_pool;
    thread_local_pool_t* task_body_pool;

    // Waker专用池（内联存储）
    thread_local_pool_t* waker_pool;

    // Channel节点专用池
    thread_local_pool_t* channel_pool;
} async_memory_pool_t;

// ============================================================================
// Future状态机内存布局优化
// ============================================================================

/**
 * @brief 压缩的Future表示（最小化内存占用）
 */
typedef struct compact_future {
    // Tagged pointer：低3位状态，高位数据指针
    uintptr_t packed_data;

    // 内联输出存储（避免额外分配）
    union {
        void* output_ptr;
        uint8_t inline_output[16];  // 小对象内联存储
    } output;

    // 内联Waker存储
    uint8_t inline_waker[32];
} compact_future_t;

/**
 * @brief Future状态枚举
 */
typedef enum future_status {
    FUTURE_READY = 0,
    FUTURE_PENDING = 1,
    FUTURE_WAITING = 2,
    FUTURE_DROPPED = 3
} future_status_t;

/**
 * @brief Task内存布局优化（头部分离）
 */
typedef struct task_header {
    void* vtable;           // 虚表指针
    uint32_t ref_count;     // 引用计数
    uint32_t flags;         // 状态标志
    void* scheduler;        // 调度器指针
} task_header_t;

// ============================================================================
// 核心API接口
// ============================================================================

/**
 * @brief 创建高级内存池
 * @param max_memory_per_thread 每个线程最大内存使用
 * @param enable_numa 是否启用NUMA感知
 * @param enable_huge_pages 是否启用大页
 * @return 内存池实例
 */
advanced_memory_pool_t* advanced_memory_pool_create(
    size_t max_memory_per_thread,
    bool enable_numa,
    bool enable_huge_pages
);

/**
 * @brief 销毁高级内存池
 * @param pool 内存池实例
 */
void advanced_memory_pool_destroy(advanced_memory_pool_t* pool);

/**
 * @brief 从内存池分配内存
 * @param pool 内存池实例
 * @param size 请求大小
 * @return 分配的内存块
 */
void* advanced_memory_pool_alloc(advanced_memory_pool_t* pool, size_t size);

/**
 * @brief 释放内存到内存池
 * @param pool 内存池实例
 * @param ptr 要释放的内存块
 * @param size 内存块大小
 * @return true成功，false失败
 */
bool advanced_memory_pool_free(advanced_memory_pool_t* pool, void* ptr, size_t size);

// ============================================================================
// 异步对象专用API
// ============================================================================

/**
 * @brief 创建异步内存池
 * @param base_pool 基础内存池
 * @return 异步内存池实例
 */
async_memory_pool_t* async_memory_pool_create(advanced_memory_pool_t* base_pool);

/**
 * @brief 销毁异步内存池
 * @param pool 异步内存池实例
 */
void async_memory_pool_destroy(async_memory_pool_t* pool);

/**
 * @brief 分配Future对象
 * @param pool 异步内存池实例
 * @param future_size Future大小
 * @return Future对象指针
 */
compact_future_t* async_memory_pool_alloc_future(async_memory_pool_t* pool, size_t future_size);

/**
 * @brief 释放Future对象
 * @param pool 异步内存池实例
 * @param future Future对象指针
 */
void async_memory_pool_free_future(async_memory_pool_t* pool, compact_future_t* future);

/**
 * @brief 分配Task对象
 * @param pool 异步内存池实例
 * @param task_size Task大小
 * @return Task对象指针
 */
void* async_memory_pool_alloc_task(async_memory_pool_t* pool, size_t task_size);

/**
 * @brief 释放Task对象
 * @param pool 异步内存池实例
 * @param task Task对象指针
 */
void async_memory_pool_free_task(async_memory_pool_t* pool, void* task);

/**
 * @brief 分配Waker对象
 * @param pool 异步内存池实例
 * @return Waker对象指针
 */
void* async_memory_pool_alloc_waker(async_memory_pool_t* pool);

/**
 * @brief 释放Waker对象
 * @param pool 异步内存池实例
 * @param waker Waker对象指针
 */
void async_memory_pool_free_waker(async_memory_pool_t* pool, void* waker);

/**
 * @brief 分配Channel节点
 * @param pool 异步内存池实例
 * @return Channel节点指针
 */
void* async_memory_pool_alloc_channel_node(async_memory_pool_t* pool);

/**
 * @brief 释放Channel节点
 * @param pool 异步内存池实例
 * @param node Channel节点指针
 */
void async_memory_pool_free_channel_node(async_memory_pool_t* pool, void* node);

// ============================================================================
// 工具函数
// ============================================================================

/**
 * @brief 获取Future状态
 * @param future Future对象
 * @return Future状态
 */
future_status_t compact_future_status(const compact_future_t* future);

/**
 * @brief 对齐到缓存行边界
 * @param size 原始大小
 * @return 对齐后的大小
 */
size_t align_to_cache_line(size_t size);

/**
 * @brief 获取最优大小类
 * @param size 请求大小
 * @return 大小类索引
 */
size_t get_size_class(size_t size);

/**
 * @brief 获取内存池统计信息
 * @param pool 内存池实例
 * @param total_alloc 总分配次数
 * @param total_free 总释放次数
 * @param memory_used 已用内存
 */
void advanced_memory_pool_stats(
    const advanced_memory_pool_t* pool,
    uint64_t* total_alloc,
    uint64_t* total_free,
    uint64_t* memory_used
);

#endif // ADVANCED_MEMORY_POOL_H
