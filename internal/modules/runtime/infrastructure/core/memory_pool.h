/**
 * @file memory_pool.h
 * @brief 内存池管理接口
 *
 * 提供高效的内存分配和回收，避免频繁的malloc/free调用。
 */

#ifndef MEMORY_POOL_H
#define MEMORY_POOL_H

#include <stdint.h>
#include <stdbool.h>
#include <stdlib.h>
#include <pthread.h>

// ============================================================================
// 内存池配置
// ============================================================================

/**
 * @brief 内存池配置
 */
typedef struct memory_pool_config {
    size_t block_size;        // 每个块的大小
    size_t initial_blocks;    // 初始块数量
    size_t max_blocks;        // 最大块数量（0表示无限制）
    bool thread_safe;         // 是否线程安全
} memory_pool_config_t;

/**
 * @brief 默认配置
 */
#define MEMORY_POOL_DEFAULT_CONFIG(block_size) \
    (memory_pool_config_t){ \
        .block_size = (block_size), \
        .initial_blocks = 16, \
        .max_blocks = 0, \
        .thread_safe = true \
    }

// ============================================================================
// 内存池接口
// ============================================================================

/**
 * @brief 内存池结构体
 */
typedef struct memory_pool {
    size_t block_size;        // 块大小
    size_t total_blocks;      // 总块数
    size_t free_blocks;       // 空闲块数
    void* free_list;          // 空闲块链表
    void* blocks;             // 已分配的块数组
    bool thread_safe;         // 是否线程安全
    pthread_mutex_t lock;     // 线程锁（当thread_safe为true时使用）

    // 统计信息
    uint64_t alloc_count;     // 分配次数
    uint64_t free_count;      // 释放次数
    uint64_t alloc_failures;  // 分配失败次数

    // 分配跟踪（用于内存泄漏检测）
    void** allocated_blocks;   // 已分配块指针数组
    size_t allocated_count;    // 已分配块数量
    size_t allocated_capacity; // 数组容量
} memory_pool_t;

/**
 * @brief 创建内存池
 *
 * @param config 内存池配置
 * @return 内存池实例，失败返回NULL
 */
memory_pool_t* memory_pool_create(const memory_pool_config_t* config);

/**
 * @brief 销毁内存池
 *
 * @param pool 内存池实例
 */
void memory_pool_destroy(memory_pool_t* pool);

/**
 * @brief 从内存池分配内存
 *
 * @param pool 内存池实例
 * @return 分配的内存块，失败返回NULL
 */
void* memory_pool_alloc(memory_pool_t* pool);

/**
 * @brief 释放内存到内存池
 *
 * @param pool 内存池实例
 * @param ptr 要释放的内存块
 * @return true成功，false失败
 */
bool memory_pool_free(memory_pool_t* pool, void* ptr);

/**
 * @brief 获取内存池统计信息
 *
 * @param pool 内存池实例
 * @param total_blocks 总块数
 * @param free_blocks 空闲块数
 * @param alloc_count 分配次数
 * @param free_count 释放次数
 */
void memory_pool_get_stats(
    const memory_pool_t* pool,
    size_t* total_blocks,
    size_t* free_blocks,
    uint64_t* alloc_count,
    uint64_t* free_count
);

/**
 * @brief 获取内存池利用率（百分比）
 *
 * @param pool 内存池实例
 * @return 利用率（0-100）
 */
double memory_pool_get_utilization(const memory_pool_t* pool);

/**
 * @brief 检查内存泄漏
 *
 * @param pool 内存池实例
 * @return 未释放的块数量
 */
size_t memory_pool_check_leaks(const memory_pool_t* pool);

/**
 * @brief 获取已分配的块列表（用于调试）
 *
 * @param pool 内存池实例
 * @param blocks 输出数组
 * @param max_count 数组最大容量
 * @return 实际返回的块数量
 */
size_t memory_pool_get_allocated_blocks(const memory_pool_t* pool,
                                       void** blocks, size_t max_count);

#endif // MEMORY_POOL_H
