/**
 * @file async_memory_pool.c
 * @brief 异步Runtime特化内存池优化
 *
 * 针对异步编程模式优化的内存池实现：
 * - Future状态机内存布局优化
 * - 协程栈的高级管理
 * - 通道内存优化
 * - Task内存池专用优化
 */

#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <stdatomic.h>
#include <stdbool.h>

#include "echo/industrial_memory_pool.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <stdatomic.h>

// ============================================================================
// Future状态机内存布局优化
// ============================================================================

/**
 * @brief 压缩的Future表示（tagged pointer状态编码）
 * 低3位存储状态，高位存储指针或内联数据
 */
typedef struct compact_future {
    atomic_uintptr_t packed;    // tagged pointer
    void* output;               // 输出存储（可内联）
    void* waker_storage;        // Waker内联存储
} compact_future_t;

/** Future状态枚举（3位编码） */
typedef enum future_status {
    FUTURE_READY = 0,      // 000
    FUTURE_PENDING = 1,    // 001
    FUTURE_WAITING = 2,    // 010
    FUTURE_DROPPED = 3,    // 011
} future_status_t;

/** 状态掩码 */
#define FUTURE_STATUS_MASK 0x7
#define FUTURE_INLINE_FLAG (1ULL << 63)  // 内联标记（最高位）

/** 内联大小阈值 */
#define FUTURE_INLINE_SIZE 16

/**
 * @brief 获取Future状态
 */
static future_status_t compact_future_get_status(const compact_future_t* future) {
    uintptr_t packed = atomic_load(&future->packed);
    return (future_status_t)(packed & FUTURE_STATUS_MASK);
}

/**
 * @brief 设置Future状态
 */
static void compact_future_set_status(compact_future_t* future, future_status_t status) {
    uintptr_t packed = atomic_load(&future->packed);
    packed = (packed & ~FUTURE_STATUS_MASK) | (uintptr_t)status;
    atomic_store(&future->packed, packed);
}

/**
 * @brief 创建压缩Future
 */
compact_future_t* compact_future_create(void* future_data, size_t future_size) {
    compact_future_t* future = (compact_future_t*)malloc(sizeof(compact_future_t));
    if (!future) return NULL;

    memset(future, 0, sizeof(compact_future_t));

    if (future_size <= FUTURE_INLINE_SIZE) {
        // 内联存储
        memcpy(&future->output, future_data, future_size);
        uintptr_t packed = (uintptr_t)&future->output | FUTURE_INLINE_FLAG | FUTURE_PENDING;
        atomic_store(&future->packed, packed);
    } else {
        // 堆分配
        void* heap_future = malloc(future_size);
        if (!heap_future) {
            free(future);
            return NULL;
        }
        memcpy(heap_future, future_data, future_size);
        uintptr_t packed = (uintptr_t)heap_future | FUTURE_PENDING;
        atomic_store(&future->packed, packed);
    }

    return future;
}

/**
 * @brief 销毁压缩Future
 */
void compact_future_destroy(compact_future_t* future) {
    if (!future) return;

    uintptr_t packed = atomic_load(&future->packed);
    if (!(packed & FUTURE_INLINE_FLAG)) {
        // 堆分配的需要释放
        void* heap_ptr = (void*)(packed & ~FUTURE_STATUS_MASK);
        free(heap_ptr);
    }

    free(future);
}

// ============================================================================
// Task内存池专用优化
// ============================================================================

/**
 * @brief Task头部结构（分离存储优化）
 */
typedef struct task_header {
    void* future;           // Future指针
    void* waker;            // Waker指针
    atomic_int status;      // Task状态
    void* scheduler_data;   // 调度器专用数据
} task_header_t;

/**
 * @brief Task主体存储（可变大小）
 */
typedef struct task_body {
    size_t size;            // 主体大小
    char data[];            // 可变大小数据
} task_body_t;

/**
 * @brief Task内存池（头部和主体分离）
 */
typedef struct task_memory_pool {
    thread_local_object_pool_t* header_pool;  // 头部池（固定大小）
    global_slab_allocator_t* body_allocator;  // 主体分配器（可变大小）
    size_t max_task_size;                      // 最大Task大小
} task_memory_pool_t;

/**
 * @brief 创建Task内存池
 */
task_memory_pool_t* task_memory_pool_create(size_t max_task_size) {
    task_memory_pool_t* pool = (task_memory_pool_t*)malloc(sizeof(task_memory_pool_t));
    if (!pool) return NULL;

    memset(pool, 0, sizeof(task_memory_pool_t));
    pool->max_task_size = max_task_size;

    // 创建头部池（固定大小）
    object_pool_config_t header_config = OBJECT_POOL_DEFAULT_CONFIG(sizeof(task_header_t));
    pool->header_pool = thread_local_object_pool_create(&header_config);
    if (!pool->header_pool) {
        free(pool);
        return NULL;
    }

    // 创建主体分配器（可变大小）
    slab_allocator_config_t slab_config = SLAB_ALLOCATOR_DEFAULT_CONFIG();
    pool->body_allocator = global_slab_allocator_create(&slab_config);
    if (!pool->body_allocator) {
        thread_local_object_pool_destroy(pool->header_pool);
        free(pool);
        return NULL;
    }

    return pool;
}

/**
 * @brief 销毁Task内存池
 */
void task_memory_pool_destroy(task_memory_pool_t* pool) {
    if (!pool) return;

    global_slab_allocator_destroy(pool->body_allocator);
    thread_local_object_pool_destroy(pool->header_pool);
    free(pool);
}

/**
 * @brief 分配Task
 */
void* task_memory_pool_allocate(task_memory_pool_t* pool, size_t body_size) {
    if (!pool || body_size > pool->max_task_size) {
        return NULL;
    }

    // 分配头部
    task_header_t* header = (task_header_t*)thread_local_object_pool_allocate(pool->header_pool);
    if (!header) {
        return NULL;
    }

    // 分配主体
    size_t total_body_size = sizeof(task_body_t) + body_size;
    task_body_t* body = (task_body_t*)global_slab_allocator_allocate(pool->body_allocator, total_body_size);
    if (!body) {
        thread_local_object_pool_deallocate(pool->header_pool, header);
        return NULL;
    }

    // 初始化
    memset(header, 0, sizeof(task_header_t));
    body->size = body_size;

    // 返回组合对象（这里简化，返回header，实际使用时需要管理关联）
    return header;
}

/**
 * @brief 释放Task
 */
bool task_memory_pool_deallocate(task_memory_pool_t* pool, void* task) {
    if (!pool || !task) return false;

    task_header_t* header = (task_header_t*)task;

    // 这里需要找到对应的body并释放
    // 简化实现：假设body存储在某个地方
    // 实际实现需要更复杂的关联管理

    return thread_local_object_pool_deallocate(pool->header_pool, header);
}

// ============================================================================
// 协程栈高级管理
// ============================================================================

/**
 * @brief 栈管理策略枚举
 */
typedef enum stack_strategy {
    STACK_SEGMENTED = 0,    // 分段栈（可增长）
    STACK_COPYING = 1,      // 拷贝栈（固定大小）
    STACK_MMAP = 2,         // mmap大栈
} stack_strategy_t;

/**
 * @brief 栈使用统计
 */
typedef struct stack_usage {
    size_t peak_usage;          // 峰值使用量
    size_t current_usage;       // 当前使用量
    int growth_pattern;         // 增长模式：-1下降，0稳定，1上升
    size_t safety_margin;       // 安全边际
} stack_usage_t;

/**
 * @brief 协程栈管理器
 */
typedef struct coroutine_stack_manager {
    stack_strategy_t strategy;              // 栈策略
    size_t initial_size;                    // 初始大小
    size_t max_size;                        // 最大大小
    size_t segment_size;                    // 分段大小
    thread_local_object_pool_t* stack_pool; // 栈池
    global_slab_allocator_t* allocator;     // 通用分配器

    // 统计和调优
    struct {
        size_t total_stacks;
        size_t active_stacks;
        double avg_usage_rate;
    } stats;
} coroutine_stack_manager_t;

/**
 * @brief 创建协程栈管理器
 */
coroutine_stack_manager_t* coroutine_stack_manager_create(
    stack_strategy_t strategy,
    size_t initial_size,
    size_t max_size
) {
    coroutine_stack_manager_t* manager = (coroutine_stack_manager_t*)malloc(sizeof(coroutine_stack_manager_t));
    if (!manager) return NULL;

    memset(manager, 0, sizeof(coroutine_stack_manager_t));
    manager->strategy = strategy;
    manager->initial_size = initial_size;
    manager->max_size = max_size;
    manager->segment_size = 4096; // 4KB段

    // 创建栈池
    object_pool_config_t pool_config = OBJECT_POOL_DEFAULT_CONFIG(initial_size);
    manager->stack_pool = thread_local_object_pool_create(&pool_config);
    if (!manager->stack_pool) {
        free(manager);
        return NULL;
    }

    // 创建通用分配器
    slab_allocator_config_t slab_config = SLAB_ALLOCATOR_DEFAULT_CONFIG();
    manager->allocator = global_slab_allocator_create(&slab_config);
    if (!manager->allocator) {
        thread_local_object_pool_destroy(manager->stack_pool);
        free(manager);
        return NULL;
    }

    return manager;
}

/**
 * @brief 销毁协程栈管理器
 */
void coroutine_stack_manager_destroy(coroutine_stack_manager_t* manager) {
    if (!manager) return;

    global_slab_allocator_destroy(manager->allocator);
    thread_local_object_pool_destroy(manager->stack_pool);
    free(manager);
}

/**
 * @brief 分配协程栈
 */
void* coroutine_stack_manager_allocate(coroutine_stack_manager_t* manager, size_t estimated_size) {
    if (!manager) return NULL;

    size_t actual_size;

    // 基于策略确定实际大小
    switch (manager->strategy) {
        case STACK_SEGMENTED:
            // 分段栈：从小开始，可增长
            actual_size = manager->segment_size;
            break;

        case STACK_COPYING:
            // 拷贝栈：固定大小，选择合适的尺寸类
            if (estimated_size <= 4096) {
                actual_size = 4096;
            } else if (estimated_size <= 65536) {
                actual_size = 65536;
            } else {
                actual_size = manager->max_size;
            }
            break;

        case STACK_MMAP:
            // 大栈：使用mmap
            actual_size = estimated_size > manager->max_size ? manager->max_size : estimated_size;
            // 确保页对齐
            actual_size = (actual_size + 4095) & ~4095;
            break;

        default:
            return NULL;
    }

    // 从池中分配或创建新栈
    void* stack = thread_local_object_pool_allocate(manager->stack_pool);
    if (!stack) {
        // 池中没有，分配新栈
        stack = global_slab_allocator_allocate(manager->allocator, actual_size);
        if (!stack) return NULL;

        // 初始化栈内容（这里简化，实际需要设置栈保护等）
        memset(stack, 0, actual_size);
    }

    manager->stats.total_stacks++;
    manager->stats.active_stacks++;

    return stack;
}

/**
 * @brief 释放协程栈
 */
bool coroutine_stack_manager_deallocate(coroutine_stack_manager_t* manager, void* stack) {
    if (!manager || !stack) return false;

    // 放回池中
    bool success = thread_local_object_pool_deallocate(manager->stack_pool, stack);
    if (success) {
        manager->stats.active_stacks--;
    }

    return success;
}

/**
 * @brief 智能栈大小调整
 */
bool coroutine_stack_manager_adjust_size(
    coroutine_stack_manager_t* manager,
    void* stack,
    const stack_usage_t* usage
) {
    if (!manager || !stack || !usage) return false;

    size_t current_size = manager->initial_size; // 简化，实际需要记录每栈大小
    size_t new_size = current_size;

    // 基于使用模式调整大小
    if (usage->growth_pattern > 0) {
        // 增长趋势：增加大小
        new_size = current_size * 2;
        if (new_size > manager->max_size) {
            new_size = manager->max_size;
        }
    } else if (usage->growth_pattern < 0 && usage->peak_usage < current_size / 2) {
        // 下降趋势且使用率低：减少大小
        new_size = current_size / 2;
        if (new_size < manager->segment_size) {
            new_size = manager->segment_size;
        }
    }

    if (new_size != current_size) {
        // 重新分配栈（热迁移）
        void* new_stack = coroutine_stack_manager_allocate(manager, new_size);
        if (new_stack) {
            // 复制栈内容（简化实现）
            size_t copy_size = usage->current_usage;
            memcpy(new_stack, stack, copy_size);

            // 释放旧栈
            coroutine_stack_manager_deallocate(manager, stack);

            // 这里需要更新协程的栈指针，实际实现会更复杂
            return true;
        }
    }

    return false;
}

// ============================================================================
// 缓存感知通道内存优化
// ============================================================================

/**
 * @brief 缓存行对齐的通道节点
 */
typedef CACHE_LINE_ALIGNED struct channel_node {
    void* data;                    // 数据指针
    atomic_uintptr_t next;         // 下一个节点（tagged pointer）
    atomic_int state;              // 节点状态
} channel_node_t;

/**
 * @brief 缓存感知通道
 */
typedef struct cache_aware_channel {
    // 生产者缓存行（避免伪共享）
    CACHE_LINE_ALIGNED struct {
        atomic_size_t head;
        atomic_size_t reserved_head;
        atomic_size_t batch_counter;
    } producer;

    // 消费者缓存行（避免伪共享）
    CACHE_LINE_ALIGNED struct {
        atomic_size_t tail;
        atomic_size_t ack_tail;
        atomic_size_t wake_counter;
    } consumer;

    // 数据缓存行
    CACHE_LINE_ALIGNED struct {
        channel_node_t* buffer;
        size_t capacity;
        size_t mask; // 环形缓冲区掩码
    } data;

    // 内存池（专用节点池）
    thread_local_object_pool_t* node_pool;
} cache_aware_channel_t;

/**
 * @brief 创建缓存感知通道
 */
cache_aware_channel_t* cache_aware_channel_create(size_t capacity) {
    cache_aware_channel_t* channel = (cache_aware_channel_t*)malloc(sizeof(cache_aware_channel_t));
    if (!channel) return NULL;

    memset(channel, 0, sizeof(cache_aware_channel_t));

    // 初始化环形缓冲区
    channel->data.capacity = capacity;
    channel->data.mask = capacity - 1; // 假设capacity是2的幂

    // 分配缓冲区
    channel->data.buffer = (channel_node_t*)malloc(sizeof(channel_node_t) * capacity);
    if (!channel->data.buffer) {
        free(channel);
        return NULL;
    }

    memset(channel->data.buffer, 0, sizeof(channel_node_t) * capacity);

    // 创建节点池
    object_pool_config_t pool_config = OBJECT_POOL_DEFAULT_CONFIG(sizeof(channel_node_t));
    channel->node_pool = thread_local_object_pool_create(&pool_config);
    if (!channel->node_pool) {
        free(channel->data.buffer);
        free(channel);
        return NULL;
    }

    return channel;
}

/**
 * @brief 销毁缓存感知通道
 */
void cache_aware_channel_destroy(cache_aware_channel_t* channel) {
    if (!channel) return;

    thread_local_object_pool_destroy(channel->node_pool);
    free(channel->data.buffer);
    free(channel);
}

/**
 * @brief 批量发送（零拷贝优化）
 */
size_t cache_aware_channel_send_batch(cache_aware_channel_t* channel, void** items, size_t count) {
    if (!channel || !items || count == 0) return 0;

    size_t head = atomic_load(&channel->producer.head);
    size_t reserved_head = head + count;

    // 检查是否有足够空间
    size_t tail = atomic_load(&channel->consumer.tail);
    if (reserved_head - tail > channel->data.capacity) {
        return 0; // 没有足够空间
    }

    // 预保留空间
    atomic_store(&channel->producer.reserved_head, reserved_head);

    // 批量写入
    for (size_t i = 0; i < count; i++) {
        size_t idx = (head + i) & channel->data.mask;
        channel->data.buffer[idx].data = items[i];
        atomic_store(&channel->data.buffer[idx].state, 1); // 标记为有效
    }

    // 更新头部
    atomic_store(&channel->producer.head, reserved_head);

    // 内存屏障确保数据可见性
    atomic_thread_fence(memory_order_release);

    return count;
}

/**
 * @brief 零拷贝接收
 */
void* cache_aware_channel_try_recv_zero_copy(cache_aware_channel_t* channel) {
    if (!channel) return NULL;

    size_t tail = atomic_load(&channel->consumer.tail);
    size_t head = atomic_load(&channel->producer.head);

    if (tail >= head) {
        return NULL; // 空
    }

    size_t idx = tail & channel->data.mask;
    channel_node_t* node = &channel->data.buffer[idx];

    // 检查节点状态
    if (atomic_load(&node->state) != 1) {
        return NULL; // 无效节点
    }

    return node->data; // 零拷贝返回引用
}

/**
 * @brief 确认接收（延迟释放）
 */
void cache_aware_channel_ack_recv(cache_aware_channel_t* channel) {
    if (!channel) return;

    size_t tail = atomic_load(&channel->consumer.tail);
    size_t idx = tail & channel->data.mask;

    // 清除节点状态
    atomic_store(&channel->data.buffer[idx].state, 0);

    // 更新尾部
    atomic_store(&channel->consumer.tail, tail + 1);
}

// ============================================================================
// 异步Runtime内存池集成
// ============================================================================

/**
 * @brief 异步Runtime内存池（集成所有优化）
 */
typedef struct async_runtime_memory_pool {
    industrial_memory_pool_t* base_pool;           // 基础工业级池
    task_memory_pool_t* task_pool;                 // Task专用池
    coroutine_stack_manager_t* stack_manager;      // 协程栈管理器
    cache_aware_channel_t* channel;                // 缓存感知通道

    // Future优化
    thread_local_object_pool_t* future_pool;       // Future池
    thread_local_object_pool_t* compact_future_pool; // 压缩Future池
} async_runtime_memory_pool_t;

/**
 * @brief 创建异步Runtime内存池
 */
async_runtime_memory_pool_t* async_runtime_memory_pool_create(void) {
    async_runtime_memory_pool_t* pool = (async_runtime_memory_pool_t*)malloc(sizeof(async_runtime_memory_pool_t));
    if (!pool) return NULL;

    memset(pool, 0, sizeof(async_runtime_memory_pool_t));

    // 创建基础工业级池
    industrial_memory_pool_config_t config = INDUSTRIAL_MEMORY_POOL_DEFAULT_CONFIG();
    pool->base_pool = industrial_memory_pool_create(&config);
    if (!pool->base_pool) {
        free(pool);
        return NULL;
    }

    // 创建Task专用池
    pool->task_pool = task_memory_pool_create(65536); // 最大64KB
    if (!pool->task_pool) {
        industrial_memory_pool_destroy(pool->base_pool);
        free(pool);
        return NULL;
    }

    // 创建协程栈管理器（分段栈策略）
    pool->stack_manager = coroutine_stack_manager_create(STACK_SEGMENTED, 4096, 1048576); // 4KB-1MB
    if (!pool->stack_manager) {
        task_memory_pool_destroy(pool->task_pool);
        industrial_memory_pool_destroy(pool->base_pool);
        free(pool);
        return NULL;
    }

    // 创建缓存感知通道
    pool->channel = cache_aware_channel_create(1024);
    if (!pool->channel) {
        coroutine_stack_manager_destroy(pool->stack_manager);
        task_memory_pool_destroy(pool->task_pool);
        industrial_memory_pool_destroy(pool->base_pool);
        free(pool);
        return NULL;
    }

    // 创建Future池
    object_pool_config_t future_config = OBJECT_POOL_DEFAULT_CONFIG(64); // 64B Future
    pool->future_pool = thread_local_object_pool_create(&future_config);
    if (!pool->future_pool) {
        cache_aware_channel_destroy(pool->channel);
        coroutine_stack_manager_destroy(pool->stack_manager);
        task_memory_pool_destroy(pool->task_pool);
        industrial_memory_pool_destroy(pool->base_pool);
        free(pool);
        return NULL;
    }

    // 创建压缩Future池
    object_pool_config_t compact_config = OBJECT_POOL_DEFAULT_CONFIG(sizeof(compact_future_t));
    pool->compact_future_pool = thread_local_object_pool_create(&compact_config);

    return pool;
}

/**
 * @brief 销毁异步Runtime内存池
 */
void async_runtime_memory_pool_destroy(async_runtime_memory_pool_t* pool) {
    if (!pool) return;

    thread_local_object_pool_destroy(pool->compact_future_pool);
    thread_local_object_pool_destroy(pool->future_pool);
    cache_aware_channel_destroy(pool->channel);
    coroutine_stack_manager_destroy(pool->stack_manager);
    task_memory_pool_destroy(pool->task_pool);
    industrial_memory_pool_destroy(pool->base_pool);

    free(pool);
}

/**
 * @brief 分配异步Task（优化版本）
 */
void* async_runtime_memory_pool_allocate_task(async_runtime_memory_pool_t* pool, size_t body_size) {
    return pool && pool->task_pool ?
        task_memory_pool_allocate(pool->task_pool, body_size) : NULL;
}

/**
 * @brief 分配协程栈（智能策略）
 */
void* async_runtime_memory_pool_allocate_stack(async_runtime_memory_pool_t* pool, size_t estimated_size) {
    return pool && pool->stack_manager ?
        coroutine_stack_manager_allocate(pool->stack_manager, estimated_size) : NULL;
}

/**
 * @brief 通道批量发送
 */
size_t async_runtime_memory_pool_channel_send_batch(async_runtime_memory_pool_t* pool, void** items, size_t count) {
    return pool && pool->channel ?
        cache_aware_channel_send_batch(pool->channel, items, count) : 0;
}

/**
 * @brief 通道零拷贝接收
 */
void* async_runtime_memory_pool_channel_try_recv_zero_copy(async_runtime_memory_pool_t* pool) {
    return pool && pool->channel ?
        cache_aware_channel_try_recv_zero_copy(pool->channel) : NULL;
}

/**
 * @brief 创建压缩Future
 */
compact_future_t* async_runtime_memory_pool_create_compact_future(async_runtime_memory_pool_t* pool, void* future_data, size_t future_size) {
    if (!pool || !pool->compact_future_pool) return NULL;

    // 从池中分配Future结构体
    compact_future_t* future = (compact_future_t*)thread_local_object_pool_allocate(pool->compact_future_pool);
    if (!future) return NULL;

    // 初始化压缩Future
    compact_future_t* result = compact_future_create(future_data, future_size);
    if (!result) {
        thread_local_object_pool_deallocate(pool->compact_future_pool, future);
        return NULL;
    }

    return result;
}

/**
 * @brief 销毁压缩Future
 */
void async_runtime_memory_pool_destroy_compact_future(async_runtime_memory_pool_t* pool, compact_future_t* future) {
    if (!pool || !future) return;

    compact_future_destroy(future);

    if (pool->compact_future_pool) {
        thread_local_object_pool_deallocate(pool->compact_future_pool, future);
    }
}
