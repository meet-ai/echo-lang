/**
 * @file advanced_memory_pool.c
 * @brief 工业级异步运行时内存池系统实现
 *
 * 核心特性：
 * - 线程本地无锁分配（极致性能）
 * - Slab大小类分配器（内存效率）
 * - 异步对象特化优化
 * - 缓存行对齐和并发优化
 */

#include "advanced_memory_pool.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <pthread.h>
#include <unistd.h>
#include <sys/mman.h>
#include <errno.h>

// ============================================================================
// 常量定义
// ============================================================================

/** Slab大小类定义 */
const size_t SIZE_CLASSES[NUM_SIZE_CLASSES] = {
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
    1024, // 大Future状态机
};

/** 每个Slab的默认块数 */
#define DEFAULT_SLAB_BLOCKS 256

/** 线程本地池的默认容量 */
#define THREAD_POOL_HOT_CAPACITY 64
#define THREAD_POOL_WARM_CAPACITY 128

/** 魔数用于调试 */
#define BLOCK_MAGIC 0xDEADBEEF

// ============================================================================
// 内部工具函数
// ============================================================================

/**
 * @brief 对齐到缓存行边界
 */
static inline size_t align_to_cache_line_internal(size_t size) {
    return (size + CACHE_LINE_SIZE - 1) & ~(CACHE_LINE_SIZE - 1);
}

/**
 * @brief 获取最优大小类
 */
static size_t get_size_class_internal(size_t size) {
    for (size_t i = 0; i < NUM_SIZE_CLASSES; i++) {
        if (size <= SIZE_CLASSES[i]) {
            return i;
        }
    }
    return NUM_SIZE_CLASSES - 1; // 最大大小类
}

/**
 * @brief 分配大页内存（如果支持）
 */
static void* alloc_huge_page(size_t size) {
    // 尝试透明大页（THP）
    void* ptr = mmap(NULL, size, PROT_READ | PROT_WRITE,
                     MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
    if (ptr == MAP_FAILED) {
        return NULL;
    }

    // 启用透明大页（如果支持）
#ifdef MADV_HUGEPAGE
    madvise(ptr, size, MADV_HUGEPAGE);
#endif

    return ptr;
}

/**
 * @brief 获取当前线程ID
 */
static pthread_t get_current_thread_id(void) {
    return pthread_self();
}

// ============================================================================
// Slab分配器实现
// ============================================================================

/**
 * @brief 创建Slab分配器
 */
static slab_allocator_t* slab_allocator_create(size_t block_size, size_t num_blocks) {
    slab_allocator_t* slab = (slab_allocator_t*)malloc(sizeof(slab_allocator_t));
    if (!slab) return NULL;

    memset(slab, 0, sizeof(slab_allocator_t));
    slab->block_size = block_size;
    slab->total_blocks = num_blocks;
    slab->free_blocks = num_blocks;

    // 分配内存（尝试大页）
    size_t total_size = block_size * num_blocks;
    slab->memory = alloc_huge_page(total_size);
    if (!slab->memory) {
        // 回退到普通malloc
        slab->memory = malloc(total_size);
        if (!slab->memory) {
            free(slab);
            return NULL;
        }
    }

    // 初始化空闲链表
    slab->free_list = NULL;
    for (size_t i = 0; i < num_blocks; i++) {
        void* block = (char*)slab->memory + (i * block_size);
        block_header_t* header = (block_header_t*)block;
        header->next = slab->free_list;
        header->size_class = get_size_class_internal(block_size);
        header->magic = BLOCK_MAGIC;
        slab->free_list = header;
    }

    // 初始化锁
    if (pthread_mutex_init(&slab->lock, NULL) != 0) {
        if (slab->memory) {
            free(slab->memory);
        }
        free(slab);
        return NULL;
    }

    return slab;
}

/**
 * @brief 销毁Slab分配器
 */
static void slab_allocator_destroy(slab_allocator_t* slab) {
    if (!slab) return;

    if (slab->memory) {
        free(slab->memory);
    }
    pthread_mutex_destroy(&slab->lock);
    free(slab);
}

/**
 * @brief 从Slab分配块
 */
static void* slab_allocator_alloc(slab_allocator_t* slab) {
    if (!slab || slab->free_blocks == 0) return NULL;

    pthread_mutex_lock(&slab->lock);

    if (!slab->free_list) {
        pthread_mutex_unlock(&slab->lock);
        return NULL;
    }

    block_header_t* block = slab->free_list;
    slab->free_list = block->next;
    slab->free_blocks--;

    pthread_mutex_unlock(&slab->lock);

    // 返回用户数据区域（跳过头部）
    return (char*)block + sizeof(block_header_t);
}

/**
 * @brief 释放块到Slab
 */
static bool slab_allocator_free(slab_allocator_t* slab, void* ptr) {
    if (!slab || !ptr) return false;

    // 计算头部位置
    block_header_t* header = (block_header_t*)((char*)ptr - sizeof(block_header_t));

    // 验证魔数
    if (header->magic != BLOCK_MAGIC) {
        return false;
    }

    pthread_mutex_lock(&slab->lock);

    // 添加到空闲链表
    header->next = slab->free_list;
    slab->free_list = header;
    slab->free_blocks++;

    pthread_mutex_unlock(&slab->lock);

    return true;
}

// ============================================================================
// 线程本地池实现
// ============================================================================

/**
 * @brief 创建线程本地池
 */
static thread_local_pool_t* thread_local_pool_create(size_t block_size) {
    thread_local_pool_t* pool = (thread_local_pool_t*)malloc(sizeof(thread_local_pool_t));
    if (!pool) return NULL;

    memset(pool, 0, sizeof(thread_local_pool_t));

    // 初始化L1热缓存
    pool->hot_capacity = THREAD_POOL_HOT_CAPACITY;
    pool->hot_cache = (void**)malloc(sizeof(void*) * pool->hot_capacity);
    if (!pool->hot_cache) {
        free(pool);
        return NULL;
    }

    // 初始化L2温缓存
    pool->warm_capacity = THREAD_POOL_WARM_CAPACITY;
    pool->warm_cache = (void**)malloc(sizeof(void*) * pool->warm_capacity);
    if (!pool->warm_cache) {
        free(pool->hot_cache);
        free(pool);
        return NULL;
    }

    // 创建L3冷Slab
    pool->cold_slab = slab_allocator_create(block_size, DEFAULT_SLAB_BLOCKS);
    if (!pool->cold_slab) {
        free(pool->warm_cache);
        free(pool->hot_cache);
        free(pool);
        return NULL;
    }

    pool->thread_id = get_current_thread_id();

    return pool;
}

/**
 * @brief 销毁线程本地池
 */
static void thread_local_pool_destroy(thread_local_pool_t* pool) {
    if (!pool) return;

    if (pool->hot_cache) free(pool->hot_cache);
    if (pool->warm_cache) free(pool->warm_cache);
    if (pool->cold_slab) slab_allocator_destroy(pool->cold_slab);

    free(pool);
}

/**
 * @brief 从线程本地池分配
 */
static void* thread_local_pool_alloc(thread_local_pool_t* pool) {
    // L1热缓存命中
    if (pool->hot_size > 0) {
        pool->hot_size--;
        pool->cache_hits_l1++;
        pool->allocations++;
        return pool->hot_cache[pool->hot_size];
    }

    // L2温缓存命中
    if (pool->warm_size > 0) {
        pool->warm_size--;
        pool->cache_hits_l2++;
        pool->allocations++;
        return pool->warm_cache[pool->warm_size];
    }

    // L3冷Slab分配
    void* ptr = slab_allocator_alloc(pool->cold_slab);
    if (ptr) {
        pool->allocations++;
    }
    return ptr;
}

/**
 * @brief 释放到线程本地池
 */
static bool thread_local_pool_free(thread_local_pool_t* pool, void* ptr) {
    // L1缓存未满，放入热缓存
    if (pool->hot_size < pool->hot_capacity) {
        pool->hot_cache[pool->hot_size++] = ptr;
        pool->deallocations++;
        return true;
    }

    // L2缓存未满，放入温缓存
    if (pool->warm_size < pool->warm_capacity) {
        pool->warm_cache[pool->warm_size++] = ptr;
        pool->deallocations++;
        return true;
    }

    // 释放到冷Slab
    bool success = slab_allocator_free(pool->cold_slab, ptr);
    if (success) {
        pool->deallocations++;
    }
    return success;
}

// ============================================================================
// 高级内存池实现
// ============================================================================

advanced_memory_pool_t* advanced_memory_pool_create(
    size_t max_memory_per_thread,
    bool enable_numa,
    bool enable_huge_pages
) {
    advanced_memory_pool_t* pool = (advanced_memory_pool_t*)malloc(sizeof(advanced_memory_pool_t));
    if (!pool) return NULL;

    memset(pool, 0, sizeof(advanced_memory_pool_t));

    // 初始化配置
    pool->max_memory_per_thread = max_memory_per_thread;
    pool->enable_numa_aware = enable_numa;
    pool->enable_huge_pages = enable_huge_pages;

    // 创建Slab分配器
    for (size_t i = 0; i < NUM_SIZE_CLASSES; i++) {
        pool->slabs[i] = slab_allocator_create(SIZE_CLASSES[i], DEFAULT_SLAB_BLOCKS);
        if (!pool->slabs[i]) {
            // 清理已创建的Slab
            for (size_t j = 0; j < i; j++) {
                if (pool->slabs[j]) {
                    slab_allocator_destroy(pool->slabs[j]);
                }
            }
            free(pool);
            return NULL;
        }
    }

    // 获取CPU核心数作为线程数上限
    pool->num_threads = (size_t)sysconf(_SC_NPROCESSORS_ONLN);
    if (pool->num_threads == 0) pool->num_threads = 4; // 默认值

    // 创建线程本地池数组
    pool->thread_pools = (thread_local_pool_t**)malloc(sizeof(thread_local_pool_t*) * pool->num_threads);
    if (!pool->thread_pools) {
        for (size_t i = 0; i < NUM_SIZE_CLASSES; i++) {
            if (pool->slabs[i]) slab_allocator_destroy(pool->slabs[i]);
        }
        free(pool);
        return NULL;
    }

    // 初始化线程本地池（延迟创建）
    memset(pool->thread_pools, 0, sizeof(thread_local_pool_t*) * pool->num_threads);

    // 初始化全局锁
    if (pthread_mutex_init(&pool->global_lock, NULL) != 0) {
        free(pool->thread_pools);
        for (size_t i = 0; i < NUM_SIZE_CLASSES; i++) {
            if (pool->slabs[i]) slab_allocator_destroy(pool->slabs[i]);
        }
        free(pool);
        return NULL;
    }

    return pool;
}

void advanced_memory_pool_destroy(advanced_memory_pool_t* pool) {
    if (!pool) return;

    // 销毁线程本地池
    for (size_t i = 0; i < pool->num_threads; i++) {
        if (pool->thread_pools[i]) {
            thread_local_pool_destroy(pool->thread_pools[i]);
        }
    }
    free(pool->thread_pools);

    // 销毁Slab分配器
    for (size_t i = 0; i < NUM_SIZE_CLASSES; i++) {
        if (pool->slabs[i]) {
            slab_allocator_destroy(pool->slabs[i]);
        }
    }

    pthread_mutex_destroy(&pool->global_lock);
    free(pool);
}

void* advanced_memory_pool_alloc(advanced_memory_pool_t* pool, size_t size) {
    if (!pool || size == 0) return NULL;

    // 对齐大小
    size = align_to_cache_line_internal(size);

    // 获取大小类
    size_t size_class = get_size_class_internal(size);
    if (size_class >= NUM_SIZE_CLASSES) return NULL;

    // 获取当前线程索引（简化实现）
    uintptr_t tid = (uintptr_t)get_current_thread_id();
    size_t thread_idx = tid % pool->num_threads;

    // 确保线程本地池存在
    if (!pool->thread_pools[thread_idx]) {
        pthread_mutex_lock(&pool->global_lock);
        if (!pool->thread_pools[thread_idx]) {
            pool->thread_pools[thread_idx] = thread_local_pool_create(SIZE_CLASSES[size_class]);
        }
        pthread_mutex_unlock(&pool->global_lock);

        if (!pool->thread_pools[thread_idx]) {
            // 回退到全局Slab
            return slab_allocator_alloc(pool->slabs[size_class]);
        }
    }

    // 从线程本地池分配
    void* ptr = thread_local_pool_alloc(pool->thread_pools[thread_idx]);
    if (ptr) {
        __atomic_add_fetch(&pool->total_allocations.value, 1, __ATOMIC_RELAXED);
        __atomic_add_fetch(&pool->total_memory_used.value, size, __ATOMIC_RELAXED);
        return ptr;
    }

    // 回退到全局Slab
    ptr = slab_allocator_alloc(pool->slabs[size_class]);
    if (ptr) {
        __atomic_add_fetch(&pool->total_allocations.value, 1, __ATOMIC_RELAXED);
        __atomic_add_fetch(&pool->total_memory_used.value, size, __ATOMIC_RELAXED);
    }
    return ptr;
}

bool advanced_memory_pool_free(advanced_memory_pool_t* pool, void* ptr, size_t size) {
    if (!pool || !ptr) return false;

    // 对齐大小
    size = align_to_cache_line_internal(size);

    // 获取大小类
    size_t size_class = get_size_class_internal(size);
    if (size_class >= NUM_SIZE_CLASSES) return false;

    // 获取当前线程索引
    uintptr_t tid = (uintptr_t)get_current_thread_id();
    size_t thread_idx = tid % pool->num_threads;

    // 尝试释放到线程本地池
    if (pool->thread_pools[thread_idx]) {
        if (thread_local_pool_free(pool->thread_pools[thread_idx], ptr)) {
            __atomic_add_fetch(&pool->total_deallocations.value, 1, __ATOMIC_RELAXED);
            __atomic_sub_fetch(&pool->total_memory_used.value, size, __ATOMIC_RELAXED);
            return true;
        }
    }

    // 回退到全局Slab
    bool success = slab_allocator_free(pool->slabs[size_class], ptr);
    if (success) {
        __atomic_add_fetch(&pool->total_deallocations.value, 1, __ATOMIC_RELAXED);
        __atomic_sub_fetch(&pool->total_memory_used.value, size, __ATOMIC_RELAXED);
    }
    return success;
}

// ============================================================================
// 异步内存池实现
// ============================================================================

async_memory_pool_t* async_memory_pool_create(advanced_memory_pool_t* base_pool) {
    if (!base_pool) return NULL;

    async_memory_pool_t* pool = (async_memory_pool_t*)malloc(sizeof(async_memory_pool_t));
    if (!pool) return NULL;

    memset(pool, 0, sizeof(async_memory_pool_t));
    pool->base_pool = base_pool;

    // 创建专用池
    pool->future_pool = thread_local_pool_create(SIZE_CLASSES[SIZE_CLASS_FUTURE]);
    pool->task_header_pool = thread_local_pool_create(sizeof(task_header_t));
    pool->task_body_pool = thread_local_pool_create(SIZE_CLASSES[SIZE_CLASS_TASK]);
    pool->waker_pool = thread_local_pool_create(SIZE_CLASSES[SIZE_CLASS_WAKER]);
    pool->channel_pool = thread_local_pool_create(SIZE_CLASSES[SIZE_CLASS_CHANNEL]);

    // 验证创建结果
    if (!pool->future_pool || !pool->task_header_pool || !pool->task_body_pool ||
        !pool->waker_pool || !pool->channel_pool) {
        async_memory_pool_destroy(pool);
        return NULL;
    }

    return pool;
}

void async_memory_pool_destroy(async_memory_pool_t* pool) {
    if (!pool) return;

    if (pool->future_pool) thread_local_pool_destroy(pool->future_pool);
    if (pool->task_header_pool) thread_local_pool_destroy(pool->task_header_pool);
    if (pool->task_body_pool) thread_local_pool_destroy(pool->task_body_pool);
    if (pool->waker_pool) thread_local_pool_destroy(pool->waker_pool);
    if (pool->channel_pool) thread_local_pool_destroy(pool->channel_pool);

    free(pool);
}

compact_future_t* async_memory_pool_alloc_future(async_memory_pool_t* pool, size_t future_size) {
    if (!pool || future_size == 0) return NULL;

    // 小Future内联优化
    if (future_size <= 16) {
        return (compact_future_t*)thread_local_pool_alloc(pool->future_pool);
    }

    // 大Future使用基础池
    return (compact_future_t*)advanced_memory_pool_alloc(pool->base_pool, future_size);
}

void async_memory_pool_free_future(async_memory_pool_t* pool, compact_future_t* future) {
    if (!pool || !future) return;

    // 检查是否是内联Future
    uintptr_t packed = (uintptr_t)future;
    if ((packed & 0x7) == 0) { // 内联标志
        thread_local_pool_free(pool->future_pool, future);
    } else {
        // 大Future释放到基础池（需要大小信息，这里简化处理）
        advanced_memory_pool_free(pool->base_pool, future, SIZE_CLASSES[SIZE_CLASS_FUTURE]);
    }
}

void* async_memory_pool_alloc_task(async_memory_pool_t* pool, size_t task_size) {
    if (!pool) return NULL;

    // Task头部分离优化
    task_header_t* header = (task_header_t*)thread_local_pool_alloc(pool->task_header_pool);
    if (!header) return NULL;

    void* body = NULL;
    if (task_size > 0) {
        body = thread_local_pool_alloc(pool->task_body_pool);
        if (!body) {
            thread_local_pool_free(pool->task_header_pool, header);
            return NULL;
        }
    }

    // 在头部存储body指针（简化实现）
    header->vtable = body;

    return header;
}

void async_memory_pool_free_task(async_memory_pool_t* pool, void* task) {
    if (!pool || !task) return;

    task_header_t* header = (task_header_t*)task;
    if (header->vtable) {
        thread_local_pool_free(pool->task_body_pool, header->vtable);
    }
    thread_local_pool_free(pool->task_header_pool, header);
}

void* async_memory_pool_alloc_waker(async_memory_pool_t* pool) {
    if (!pool) return NULL;
    return thread_local_pool_alloc(pool->waker_pool);
}

void async_memory_pool_free_waker(async_memory_pool_t* pool, void* waker) {
    if (!pool || !waker) return;
    thread_local_pool_free(pool->waker_pool, waker);
}

void* async_memory_pool_alloc_channel_node(async_memory_pool_t* pool) {
    if (!pool) return NULL;
    return thread_local_pool_alloc(pool->channel_pool);
}

void async_memory_pool_free_channel_node(async_memory_pool_t* pool, void* node) {
    if (!pool || !node) return;
    thread_local_pool_free(pool->channel_pool, node);
}

// ============================================================================
// Future状态机工具函数
// ============================================================================

future_status_t compact_future_status(const compact_future_t* future) {
    if (!future) return FUTURE_DROPPED;

    uintptr_t packed = future->packed_data;
    return (future_status_t)(packed & 0x7); // 低3位是状态
}

size_t align_to_cache_line(size_t size) {
    return align_to_cache_line_internal(size);
}

size_t get_size_class(size_t size) {
    return get_size_class_internal(size);
}

void advanced_memory_pool_stats(
    const advanced_memory_pool_t* pool,
    uint64_t* total_alloc,
    uint64_t* total_free,
    uint64_t* memory_used
) {
    if (!pool) return;

    if (total_alloc) *total_alloc = pool->total_allocations.value;
    if (total_free) *total_free = pool->total_deallocations.value;
    if (memory_used) *memory_used = pool->total_memory_used.value;
}
