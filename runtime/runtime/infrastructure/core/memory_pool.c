/**
 * @file memory_pool.c
 * @brief 内存池管理实现
 *
 * 高效的内存池实现，支持固定大小块的快速分配和回收。
 */

#include "memory_pool.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <pthread.h>

// ============================================================================
// 内部数据结构
// ============================================================================

/**
 * @brief 内存块头部（用于空闲链表）
 */
typedef struct block_header {
    struct block_header* next;  // 下一个空闲块
} block_header_t;

// ============================================================================
// 内部函数声明
// ============================================================================

static bool memory_pool_expand(memory_pool_t* pool, size_t num_blocks);

// ============================================================================
// 锁管理（线程安全）
// ============================================================================

static inline void lock_init(pthread_mutex_t* lock) {
    if (lock) {
        pthread_mutex_init(lock, NULL);
    }
}

static inline void lock_destroy(pthread_mutex_t* lock) {
    if (lock) {
        pthread_mutex_destroy(lock);
    }
}

static inline void lock_acquire(pthread_mutex_t* lock) {
    if (lock) {
        pthread_mutex_lock(lock);
    }
}

static inline void lock_release(pthread_mutex_t* lock) {
    if (lock) {
        pthread_mutex_unlock(lock);
    }
}

// ============================================================================
// 内存池实现
// ============================================================================

/**
 * @brief 创建内存池
 */
memory_pool_t* memory_pool_create(const memory_pool_config_t* config) {
    if (!config || config->block_size == 0) {
        return NULL;
    }

    // 创建内存池结构体
    memory_pool_t* pool = (memory_pool_t*)malloc(sizeof(memory_pool_t));
    if (!pool) {
        return NULL;
    }

    memset(pool, 0, sizeof(memory_pool_t));
    pool->block_size = config->block_size;
    pool->thread_safe = config->thread_safe;

    // 初始化分配跟踪数组
    pool->allocated_capacity = 64; // 初始容量
    pool->allocated_blocks = (void**)malloc(sizeof(void*) * pool->allocated_capacity);
    if (!pool->allocated_blocks) {
        free(pool);
        return NULL;
    }
    pool->allocated_count = 0;

    // 初始化锁
    if (pool->thread_safe) {
        if (pthread_mutex_init(&pool->lock, NULL) != 0) {
            free(pool);
            return NULL;
        }
    }

    // 分配初始块
    size_t initial_blocks = config->initial_blocks > 0 ? config->initial_blocks : 16;
    if (!memory_pool_expand(pool, initial_blocks)) {
        free(pool);
        return NULL;
    }

    return pool;
}

/**
 * @brief 扩展内存池
 */
static bool memory_pool_expand(memory_pool_t* pool, size_t num_blocks) {
    if (!pool || num_blocks == 0) return false;

    // 计算需要的内存大小
    size_t block_size_with_header = pool->block_size + sizeof(block_header_t);
    size_t total_size = num_blocks * block_size_with_header;

    // 分配内存
    void* new_blocks = malloc(total_size);
    if (!new_blocks) {
        return false;
    }

    // 初始化空闲链表
    block_header_t* current = (block_header_t*)new_blocks;
    pool->free_list = current;

    for (size_t i = 0; i < num_blocks - 1; i++) {
        current->next = (block_header_t*)((char*)current + block_size_with_header);
        current = current->next;
    }
    current->next = NULL;

    // 更新统计信息
    pool->total_blocks += num_blocks;
    pool->free_blocks += num_blocks;

    return true;
}

/**
 * @brief 销毁内存池
 */
void memory_pool_destroy(memory_pool_t* pool) {
    if (!pool) return;

    // 销毁锁
    if (pool->thread_safe) {
        pthread_mutex_destroy(&pool->lock);
    }

    // 检测内存泄漏
    if (pool->allocated_count > 0) {
        fprintf(stderr, "WARNING: Memory pool destroyed with %zu unfreed blocks\n",
                pool->allocated_count);
        // 可以在这里打印泄漏的块地址
        for (size_t i = 0; i < pool->allocated_count; i++) {
            fprintf(stderr, "  Leaked block: %p\n", pool->allocated_blocks[i]);
        }
    }

    // 清理分配跟踪
    free(pool->allocated_blocks);

    // 释放所有已分配的块
    free(pool->blocks);
    free(pool);
}

/**
 * @brief 从内存池分配内存
 */
void* memory_pool_alloc(memory_pool_t* pool) {
    if (!pool) return NULL;

    lock_acquire(pool->thread_safe ? &pool->lock : NULL);

    void* result = NULL;

    if (pool->free_list) {
        // 从空闲链表取出一个块
        block_header_t* block = (block_header_t*)pool->free_list;
        pool->free_list = block->next;
        pool->free_blocks--;

        // 返回块的数据部分（跳过头部）
        result = (void*)((char*)block + sizeof(block_header_t));

        // 添加到分配跟踪列表
        if (pool->allocated_count >= pool->allocated_capacity) {
            // 扩展数组容量
            size_t new_capacity = pool->allocated_capacity * 2;
            void** new_array = (void**)realloc(pool->allocated_blocks,
                                               sizeof(void*) * new_capacity);
            if (!new_array) {
                // 扩展失败，但分配仍然成功
                fprintf(stderr, "WARNING: Failed to expand allocation tracking array\n");
            } else {
                pool->allocated_blocks = new_array;
                pool->allocated_capacity = new_capacity;
            }
        }

        if (pool->allocated_count < pool->allocated_capacity) {
            pool->allocated_blocks[pool->allocated_count++] = result;
        }

        pool->alloc_count++;
    } else {
        // 空闲链表为空，尝试扩展
        if (memory_pool_expand(pool, pool->total_blocks / 2 + 1)) {
            // 递归调用自己
            lock_release(pool->thread_safe ? &pool->lock : NULL);
            return memory_pool_alloc(pool);
        } else {
            pool->alloc_failures++;
        }
    }

    lock_release(pool->thread_safe ? &pool->lock : NULL);
    return result;
}

/**
 * @brief 释放内存到内存池
 */
bool memory_pool_free(memory_pool_t* pool, void* ptr) {
    if (!pool || !ptr) return false;

    lock_acquire(pool->thread_safe ? &pool->lock : NULL);

    // 从分配跟踪列表中移除
    for (size_t i = 0; i < pool->allocated_count; i++) {
        if (pool->allocated_blocks[i] == ptr) {
            // 找到并移除
            pool->allocated_blocks[i] = pool->allocated_blocks[--pool->allocated_count];
            break;
        }
    }

    // 将指针回退到块头部
    block_header_t* block = (block_header_t*)((char*)ptr - sizeof(block_header_t));

    // 添加到空闲链表头部
    block->next = (block_header_t*)pool->free_list;
    pool->free_list = block;
    pool->free_blocks++;

    pool->free_count++;

    lock_release(pool->thread_safe ? &pool->lock : NULL);
    return true;
}

/**
 * @brief 获取内存池统计信息
 */
void memory_pool_get_stats(
    const memory_pool_t* pool,
    size_t* total_blocks,
    size_t* free_blocks,
    uint64_t* alloc_count,
    uint64_t* free_count
) {
    if (!pool) return;

    if (total_blocks) *total_blocks = pool->total_blocks;
    if (free_blocks) *free_blocks = pool->free_blocks;
    if (alloc_count) *alloc_count = pool->alloc_count;
    if (free_count) *free_count = pool->free_count;
}

/**
 * @brief 获取内存池利用率
 */
double memory_pool_get_utilization(const memory_pool_t* pool) {
    if (!pool || pool->total_blocks == 0) return 0.0;

    size_t used_blocks = pool->total_blocks - pool->free_blocks;
    return (double)used_blocks / (double)pool->total_blocks * 100.0;
}

/**
 * @brief 检查内存泄漏
 */
size_t memory_pool_check_leaks(const memory_pool_t* pool) {
    return pool ? pool->allocated_count : 0;
}

/**
 * @brief 获取已分配的块列表（用于调试）
 */
size_t memory_pool_get_allocated_blocks(const memory_pool_t* pool,
                                       void** blocks, size_t max_count) {
    if (!pool || !blocks || max_count == 0) {
        return 0;
    }

    size_t count = pool->allocated_count < max_count ? pool->allocated_count : max_count;
    memcpy(blocks, pool->allocated_blocks, count * sizeof(void*));
    return count;
}
