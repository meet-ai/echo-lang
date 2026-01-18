/**
 * @file cache_utils.h
 * @brief 缓存行对齐和内存布局优化工具
 *
 * 提供缓存行对齐、伪共享避免、内存屏障等底层优化功能。
 */

#ifndef CACHE_UTILS_H
#define CACHE_UTILS_H

#include <stdint.h>
#include <stdlib.h>
#include <stdbool.h>

// ============================================================================
// 缓存行对齐宏和类型
// ============================================================================

/** 标准缓存行大小（64字节） */
#define CACHE_LINE_SIZE 64

/** 缓存行对齐宏 */
#define CACHE_LINE_ALIGNED __attribute__((aligned(CACHE_LINE_SIZE)))

/** 对齐到缓存行边界 */
#define ALIGN_TO_CACHE_LINE(size) \
    (((size) + CACHE_LINE_SIZE - 1) & ~(CACHE_LINE_SIZE - 1))

/**
 * @brief 缓存行对齐的uint64_t
 */
typedef struct {
    volatile uint64_t value;
    char _padding[CACHE_LINE_SIZE - sizeof(uint64_t)];
} cache_aligned_uint64_t CACHE_LINE_ALIGNED;

/**
 * @brief 缓存行对齐的指针
 */
typedef struct {
    void* ptr;
    char _padding[CACHE_LINE_SIZE - sizeof(void*)];
} cache_aligned_ptr_t CACHE_LINE_ALIGNED;

/**
 * @brief 缓存行对齐的原子操作包装
 */
typedef struct {
    volatile size_t value;
    char _padding[CACHE_LINE_SIZE - sizeof(size_t)];
} cache_aligned_size_t CACHE_LINE_ALIGNED;

// ============================================================================
// 内存屏障和原子操作
// ============================================================================

/**
 * @brief 读内存屏障
 * 确保在此屏障之前的所有读操作在屏障之后的所有读操作之前完成
 */
static inline void memory_barrier_read(void) {
    __asm__ __volatile__("lfence" ::: "memory");
}

/**
 * @brief 写内存屏障
 * 确保在此屏障之前的所有写操作在屏障之后的所有写操作之前完成
 */
static inline void memory_barrier_write(void) {
    __asm__ __volatile__("sfence" ::: "memory");
}

/**
 * @brief 完整内存屏障
 * 确保在此屏障之前的所有内存操作在屏障之后的所有内存操作之前完成
 */
static inline void memory_barrier_full(void) {
    __asm__ __volatile__("mfence" ::: "memory");
}

/**
 * @brief 原子加载（带获取语义）
 */
static inline uint64_t atomic_load_acquire(volatile uint64_t* ptr) {
    uint64_t value = __atomic_load_n(ptr, __ATOMIC_ACQUIRE);
    return value;
}

/**
 * @brief 原子存储（带释放语义）
 */
static inline void atomic_store_release(volatile uint64_t* ptr, uint64_t value) {
    __atomic_store_n(ptr, value, __ATOMIC_RELEASE);
}

/**
 * @brief 原子加法（返回旧值）
 */
static inline uint64_t atomic_fetch_add(volatile uint64_t* ptr, uint64_t value) {
    return __atomic_fetch_add(ptr, value, __ATOMIC_RELAXED);
}

/**
 * @brief 原子减法（返回旧值）
 */
static inline uint64_t atomic_fetch_sub(volatile uint64_t* ptr, uint64_t value) {
    return __atomic_fetch_sub(ptr, value, __ATOMIC_RELAXED);
}

// ============================================================================
// NUMA感知内存分配
// ============================================================================

/**
 * @brief NUMA节点信息
 */
typedef struct numa_node_info {
    int node_id;        // NUMA节点ID
    size_t memory_total; // 节点总内存
    size_t memory_free;  // 节点可用内存
    int cpu_count;      // 节点CPU核心数
} numa_node_info_t;

/**
 * @brief 获取当前线程的NUMA节点
 * @return NUMA节点ID，失败返回-1
 */
int get_current_numa_node(void);

/**
 * @brief 获取系统NUMA节点信息
 * @param nodes 输出数组
 * @param max_nodes 数组最大长度
 * @return 实际节点数量
 */
int get_numa_topology(numa_node_info_t* nodes, int max_nodes);

/**
 * @brief 在指定NUMA节点分配内存
 * @param size 分配大小
 * @param node_id NUMA节点ID
 * @return 分配的内存指针，失败返回NULL
 */
void* alloc_memory_on_numa_node(size_t size, int node_id);

/**
 * @brief 迁移内存到指定NUMA节点
 * @param ptr 内存指针
 * @param size 内存大小
 * @param node_id 目标NUMA节点ID
 * @return true成功，false失败
 */
bool migrate_memory_to_numa_node(void* ptr, size_t size, int node_id);

// ============================================================================
// 位图操作（用于Slab管理）
// ============================================================================

/**
 * @brief 位图结构（256KB，支持256K个对象）
 */
typedef struct bitmap_256k {
    uint64_t bits[32768]; // 256KB = 32768 * 8 bytes
} bitmap_256k_t;

/**
 * @brief 设置位图中的位
 * @param bitmap 位图指针
 * @param index 位索引（0-262143）
 */
static inline void bitmap_set(bitmap_256k_t* bitmap, uint32_t index) {
    uint32_t word_index = index / 64;
    uint32_t bit_index = index % 64;
    bitmap->bits[word_index] |= (1ULL << bit_index);
}

/**
 * @brief 清除位图中的位
 * @param bitmap 位图指针
 * @param index 位索引
 */
static inline void bitmap_clear(bitmap_256k_t* bitmap, uint32_t index) {
    uint32_t word_index = index / 64;
    uint32_t bit_index = index % 64;
    bitmap->bits[word_index] &= ~(1ULL << bit_index);
}

/**
 * @brief 测试位图中的位
 * @param bitmap 位图指针
 * @param index 位索引
 * @return true位已设置，false位未设置
 */
static inline bool bitmap_test(const bitmap_256k_t* bitmap, uint32_t index) {
    uint32_t word_index = index / 64;
    uint32_t bit_index = index % 64;
    return (bitmap->bits[word_index] & (1ULL << bit_index)) != 0;
}

/**
 * @brief 查找第一个空闲位
 * @param bitmap 位图指针
 * @param start_index 起始搜索位置
 * @return 第一个空闲位的索引，未找到返回-1
 */
int bitmap_find_first_clear(const bitmap_256k_t* bitmap, uint32_t start_index);

// ============================================================================
// SIMD加速的位图扫描
// ============================================================================

/**
 * @brief SIMD加速的位图扫描（需要AVX2支持）
 * @param bitmap 位图指针
 * @param start_index 起始位置
 * @return 第一个空闲位的索引，未找到返回-1
 */
int bitmap_find_first_clear_simd(const bitmap_256k_t* bitmap, uint32_t start_index);

// ============================================================================
// 性能监控和调优
// ============================================================================

/**
 * @brief 内存池性能统计
 */
typedef struct memory_pool_stats {
    uint64_t total_allocations;     // 总分配次数
    uint64_t total_deallocations;   // 总释放次数
    uint64_t current_allocations;   // 当前分配数量
    uint64_t peak_allocations;      // 峰值分配数量

    uint64_t cache_hits_l1;         // L1缓存命中次数
    uint64_t cache_hits_l2;         // L2缓存命中次数
    uint64_t cache_misses;          // 缓存未命中次数

    double average_allocation_time; // 平均分配时间（纳秒）
    double average_deallocation_time; // 平均释放时间（纳秒）

    size_t memory_used;             // 已用内存
    size_t memory_peak;             // 内存峰值
    double memory_fragmentation;    // 内存碎片率
} memory_pool_stats_t;

/**
 * @brief 重置性能统计
 * @param stats 统计结构指针
 */
void memory_pool_stats_reset(memory_pool_stats_t* stats);

/**
 * @brief 记录分配操作
 * @param stats 统计结构指针
 * @param size 分配大小
 * @param time_ns 耗时（纳秒）
 */
void memory_pool_stats_record_alloc(memory_pool_stats_t* stats, size_t size, uint64_t time_ns);

/**
 * @brief 记录释放操作
 * @param stats 统计结构指针
 * @param time_ns 耗时（纳秒）
 */
void memory_pool_stats_record_free(memory_pool_stats_t* stats, uint64_t time_ns);

/**
 * @brief 计算内存碎片率
 * @param stats 统计结构指针
 * @param total_memory 总内存大小
 */
void memory_pool_stats_calculate_fragmentation(memory_pool_stats_t* stats, size_t total_memory);

// ============================================================================
// 大页内存支持
// ============================================================================

/**
 * @brief 大页内存分配器
 */
typedef struct huge_page_allocator {
    size_t page_size;           // 页大小（2MB或1GB）
    void* base_address;         // 基地址
    bitmap_256k_t free_bitmap;  // 空闲位图
    size_t total_pages;         // 总页数
    size_t free_pages;          // 空闲页数
} huge_page_allocator_t;

/**
 * @brief 创建大页分配器
 * @param page_size 页大小（2MB或1GB）
 * @param num_pages 页数量
 * @return 分配器指针，失败返回NULL
 */
huge_page_allocator_t* huge_page_allocator_create(size_t page_size, size_t num_pages);

/**
 * @brief 销毁大页分配器
 * @param allocator 分配器指针
 */
void huge_page_allocator_destroy(huge_page_allocator_t* allocator);

/**
 * @brief 从大页分配器分配内存
 * @param allocator 分配器指针
 * @param size 请求大小
 * @return 分配的内存指针，失败返回NULL
 */
void* huge_page_allocator_alloc(huge_page_allocator_t* allocator, size_t size);

/**
 * @brief 释放内存到大页分配器
 * @param allocator 分配器指针
 * @param ptr 内存指针
 * @param size 内存大小
 * @return true成功，false失败
 */
bool huge_page_allocator_free(huge_page_allocator_t* allocator, void* ptr, size_t size);

#endif // CACHE_UTILS_H
