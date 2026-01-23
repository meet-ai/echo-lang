#include "heap.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <unistd.h>
#include <sys/mman.h>
#include <uuid/uuid.h>
#include "echo/gc.h"

// 默认配置
static const echo_heap_config_t DEFAULT_HEAP_CONFIG = {
    .initial_size = 16 * 1024 * 1024,      // 16MB
    .max_size = 1024 * 1024 * 1024,        // 1GB
    .growth_factor = 150,                  // 150%
    .enable_large_objects = true,
    .large_object_threshold = 32 * 1024    // 32KB
};

// 生成UUID
static void generate_uuid(char* buffer, size_t size) {
    uuid_t uuid;
    uuid_generate(uuid);
    uuid_unparse_lower(uuid, buffer);
}

// 获取当前时间
static void get_current_time(struct timespec* ts) {
    clock_gettime(CLOCK_MONOTONIC, ts);
}

// 默认内存分配函数
static void* default_malloc(size_t size) {
    return malloc(size);
}

static void default_free(void* ptr) {
    free(ptr);
}

// 简单的分配器实现（后面会用更复杂的）
typedef struct simple_allocator {
    echo_gc_allocator_t base;
    echo_gc_heap_t* heap;
    void* free_list[64];  // 简单空闲链表
} simple_allocator_t;

static echo_gc_error_t simple_allocate(
    echo_gc_allocator_t* allocator,
    size_t size,
    echo_obj_type_t type,
    void** result
) {
    simple_allocator_t* simple = (simple_allocator_t*)allocator;

    // 对齐到8字节
    size = (size + 7) & ~7;

    // 添加对象头
    size_t total_size = size + sizeof(echo_gc_object_header_t);

    // 从堆分配内存
    void* ptr = simple->heap->malloc_fn(total_size);
    if (!ptr) {
        return ECHO_GC_ERROR_OUT_OF_MEMORY;
    }

    // 初始化对象头
    echo_gc_object_header_t* header = (echo_gc_object_header_t*)ptr;
    header->flags = 0;
    header->size = total_size;
    header->color = ECHO_GC_COLOR_WHITE;
    header->forwarded = 0;
    header->pinned = 0;
    header->hash = 0;  // 可以基于type生成

    *result = (char*)ptr + sizeof(echo_gc_object_header_t);

    // 更新统计
    simple->heap->stats.allocated_objects++;
    simple->heap->stats.used_size += total_size;
    simple->heap->allocated_bytes_since_gc += total_size;

    return ECHO_GC_SUCCESS;
}

static void simple_deallocate(echo_gc_allocator_t* allocator, void* ptr) {
    simple_allocator_t* simple = (simple_allocator_t*)allocator;

    if (!ptr) return;

    // 获取对象头
    echo_gc_object_header_t* header = (echo_gc_object_header_t*)((char*)ptr - sizeof(echo_gc_object_header_t));

    // 更新统计
    simple->heap->stats.allocated_objects--;
    simple->heap->stats.used_size -= header->size;

    // 释放内存
    simple->heap->free_fn(header);
}

static size_t simple_get_object_size(echo_gc_allocator_t* allocator, void* ptr) {
    if (!ptr) return 0;

    echo_gc_object_header_t* header = (echo_gc_object_header_t*)((char*)ptr - sizeof(echo_gc_object_header_t));
    return header->size;
}

static echo_obj_type_t simple_get_object_type(echo_gc_allocator_t* allocator, void* ptr) {
    // 暂时不支持类型信息
    return ECHO_OBJ_TYPE_PRIMITIVE;
}

static void simple_get_stats(echo_gc_allocator_t* allocator, echo_heap_stats_t* stats) {
    // 简单实现
}

static void simple_prepare_for_gc(echo_gc_allocator_t* allocator) {
    // 准备GC
}

static void simple_complete_gc(echo_gc_allocator_t* allocator) {
    // 完成GC
}

// 创建分配器
static echo_gc_allocator_t* create_simple_allocator(echo_gc_heap_t* heap) {
    simple_allocator_t* allocator = calloc(1, sizeof(simple_allocator_t));
    if (!allocator) return NULL;

    allocator->base.allocate = simple_allocate;
    allocator->base.deallocate = simple_deallocate;
    allocator->base.get_object_size = simple_get_object_size;
    allocator->base.get_object_type = simple_get_object_type;
    allocator->base.get_stats = simple_get_stats;
    allocator->base.prepare_for_gc = simple_prepare_for_gc;
    allocator->base.complete_gc = simple_complete_gc;

    allocator->heap = heap;

    return &allocator->base;
}

// 创建堆
echo_gc_heap_t* echo_gc_heap_create(const echo_heap_config_t* config) {
    const echo_heap_config_t* heap_config = config ? config : &DEFAULT_HEAP_CONFIG;

    echo_gc_heap_t* heap = calloc(1, sizeof(echo_gc_heap_t));
    if (!heap) {
        return NULL;
    }

    // 生成唯一ID
    generate_uuid(heap->id, sizeof(heap->id));

    // 复制配置
    memcpy(&heap->config, heap_config, sizeof(echo_heap_config_t));

    // 初始化统计
    memset(&heap->stats, 0, sizeof(echo_heap_stats_t));
    heap->stats.total_size = heap_config->initial_size;

    // 设置内存函数
    heap->malloc_fn = default_malloc;
    heap->free_fn = default_free;

    // 初始化并发控制
    pthread_mutex_init(&heap->mutex, NULL);
    pthread_rwlock_init(&heap->rwlock, NULL);

    // 创建分配器
    heap->allocator = create_simple_allocator(heap);
    if (!heap->allocator) {
        free(heap);
        return NULL;
    }

    // 初始化标记位图（延迟到实际需要时创建）
    heap->mark_bitmap = NULL;

    return heap;
}

// 销毁堆
void echo_gc_heap_destroy(echo_gc_heap_t* heap) {
    if (!heap) return;

    // 清理分配器
    if (heap->allocator) {
        free(heap->allocator);
    }

    // 清理标记位图
    if (heap->mark_bitmap) {
        echo_gc_mark_bitmap_destroy(heap->mark_bitmap);
        free(heap->mark_bitmap);
    }

    // 清理并发控制
    pthread_mutex_destroy(&heap->mutex);
    pthread_rwlock_destroy(&heap->rwlock);

    free(heap);
}

// 分配对象
echo_gc_error_t echo_gc_heap_allocate(
    echo_gc_heap_t* heap,
    size_t size,
    echo_obj_type_t type,
    void** result
) {
    if (!heap || !result) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    pthread_rwlock_wrlock(&heap->rwlock);

    echo_gc_error_t err = heap->allocator->allocate(heap->allocator, size, type, result);

    pthread_rwlock_unlock(&heap->rwlock);

    return err;
}

// 标记对象
echo_gc_error_t echo_gc_heap_mark_object(echo_gc_heap_t* heap, void* obj) {
    if (!heap || !obj) {
        return ECHO_GC_ERROR_INVALID_OBJECT;
    }

    if (!echo_gc_heap_contains(heap, obj)) {
        return ECHO_GC_ERROR_INVALID_OBJECT;
    }

    uint64_t index = echo_gc_heap_get_object_index(heap, obj);

    if (!heap->mark_bitmap) {
        // 延迟初始化标记位图
        heap->mark_bitmap = calloc(1, sizeof(echo_gc_mark_bitmap_t));
        if (!heap->mark_bitmap) {
            return ECHO_GC_ERROR_OUT_OF_MEMORY;
        }
        echo_gc_mark_bitmap_init(heap->mark_bitmap, (uint64_t)heap->arena_start, heap->arena_size, 8);
    }

    echo_gc_mark_bitmap_set(heap->mark_bitmap, index);
    return ECHO_GC_SUCCESS;
}

// 检查对象是否已标记
bool echo_gc_heap_is_marked(const echo_gc_heap_t* heap, void* obj) {
    if (!heap || !obj || !heap->mark_bitmap) {
        return false;
    }

    if (!echo_gc_heap_contains(heap, obj)) {
        return false;
    }

    uint64_t index = echo_gc_heap_get_object_index(heap, obj);
    return echo_gc_mark_bitmap_get(heap->mark_bitmap, index);
}

// 清扫未标记对象
echo_gc_error_t echo_gc_heap_sweep(
    echo_gc_heap_t* heap,
    uint64_t* objects_collected,
    uint64_t* memory_reclaimed
) {
    if (!heap || !objects_collected || !memory_reclaimed) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    *objects_collected = 0;
    *memory_reclaimed = 0;

    if (!heap->mark_bitmap) {
        return ECHO_GC_SUCCESS;  // 没有位图，说明没有进行标记
    }

    pthread_rwlock_wrlock(&heap->rwlock);

    // 简单实现：遍历所有对象（实际实现需要更好的数据结构）
    // 这里只是演示，清扫阶段需要具体的对象列表

    // 重置位图为下次GC准备
    echo_gc_mark_bitmap_clear_all(heap->mark_bitmap);

    // 更新统计
    heap->stats.live_objects = heap->stats.allocated_objects;  // 简化
    heap->allocated_bytes_since_gc = 0;
    get_current_time(&heap->stats.last_gc_time);

    pthread_rwlock_unlock(&heap->rwlock);

    return ECHO_GC_SUCCESS;
}

// 查询方法
uint64_t echo_gc_heap_get_used_ratio(const echo_gc_heap_t* heap) {
    if (!heap || heap->stats.total_size == 0) {
        return 0;
    }

    return (heap->stats.used_size * 100) / heap->stats.total_size;
}

uint64_t echo_gc_heap_get_size(const echo_gc_heap_t* heap) {
    return heap ? heap->stats.total_size : 0;
}

const echo_heap_stats_t* echo_gc_heap_get_stats(const echo_gc_heap_t* heap) {
    return heap ? &heap->stats : NULL;
}

// GC生命周期方法
echo_gc_error_t echo_gc_heap_prepare_for_gc(echo_gc_heap_t* heap) {
    if (!heap) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    pthread_mutex_lock(&heap->mutex);
    heap->in_gc = true;

    // 通知分配器准备GC
    if (heap->allocator && heap->allocator->prepare_for_gc) {
        heap->allocator->prepare_for_gc(heap->allocator);
    }

    pthread_mutex_unlock(&heap->mutex);

    return ECHO_GC_SUCCESS;
}

echo_gc_error_t echo_gc_heap_complete_gc(echo_gc_heap_t* heap) {
    if (!heap) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    pthread_mutex_lock(&heap->mutex);
    heap->in_gc = false;

    // 通知分配器完成GC
    if (heap->allocator && heap->allocator->complete_gc) {
        heap->allocator->complete_gc(heap->allocator);
    }

    pthread_mutex_unlock(&heap->mutex);

    return ECHO_GC_SUCCESS;
}

// 堆扩展
echo_gc_error_t echo_gc_heap_grow(echo_gc_heap_t* heap, uint64_t new_size) {
    if (!heap || new_size <= heap->stats.total_size) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    if (new_size > heap->config.max_size) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    // 这里简化实现，实际需要重新分配堆空间
    heap->stats.total_size = new_size;

    return ECHO_GC_SUCCESS;
}

// 事件设置
void echo_gc_heap_set_event_publisher(
    echo_gc_heap_t* heap,
    void (*publisher)(const char* event_type, const void* event_data)
) {
    if (heap) {
        heap->event_publisher = publisher;
    }
}

// 内存函数设置
void echo_gc_heap_set_memory_functions(
    echo_gc_heap_t* heap,
    void* (*malloc_fn)(size_t),
    void (*free_fn)(void*)
) {
    if (heap) {
        heap->malloc_fn = malloc_fn ? malloc_fn : default_malloc;
        heap->free_fn = free_fn ? free_fn : default_free;
    }
}

// 辅助方法
bool echo_gc_heap_contains(const echo_gc_heap_t* heap, void* ptr) {
    if (!heap || !ptr) {
        return false;
    }

    // 简化检查，实际需要更精确的实现
    return true;  // 暂时假设所有指针都在堆中
}

uint64_t echo_gc_heap_get_object_index(const echo_gc_heap_t* heap, void* ptr) {
    if (!heap || !ptr) {
        return 0;
    }

    // 简化实现，实际需要基于对象地址计算索引
    return (uint64_t)ptr / 8;  // 假设8字节对齐
}

void* echo_gc_heap_get_object_from_index(const echo_gc_heap_t* heap, uint64_t index) {
    // 简化实现
    return (void*)(index * 8);
}

// 标记位图实现
void echo_gc_mark_bitmap_init(
    echo_gc_mark_bitmap_t* bitmap,
    uint64_t heap_start,
    uint64_t heap_size,
    uint64_t alignment
) {
    if (!bitmap) return;

    bitmap->heap_start = heap_start;
    bitmap->object_alignment = alignment;

    // 计算需要的位图大小（每8字节对象1位）
    uint64_t objects_count = heap_size / alignment;
    bitmap->bitmap_size = (objects_count + 7) / 8;  // 字节数

    bitmap->bitmap = calloc(1, bitmap->bitmap_size);
}

void echo_gc_mark_bitmap_destroy(echo_gc_mark_bitmap_t* bitmap) {
    if (bitmap && bitmap->bitmap) {
        free(bitmap->bitmap);
        bitmap->bitmap = NULL;
    }
}

void echo_gc_mark_bitmap_set(echo_gc_mark_bitmap_t* bitmap, uint64_t index) {
    if (!bitmap || !bitmap->bitmap) return;

    uint64_t byte_index = index / 8;
    uint8_t bit_index = index % 8;

    if (byte_index < bitmap->bitmap_size) {
        bitmap->bitmap[byte_index] |= (1 << bit_index);
    }
}

void echo_gc_mark_bitmap_clear(echo_gc_mark_bitmap_t* bitmap, uint64_t index) {
    if (!bitmap || !bitmap->bitmap) return;

    uint64_t byte_index = index / 8;
    uint8_t bit_index = index % 8;

    if (byte_index < bitmap->bitmap_size) {
        bitmap->bitmap[byte_index] &= ~(1 << bit_index);
    }
}

bool echo_gc_mark_bitmap_get(const echo_gc_mark_bitmap_t* bitmap, uint64_t index) {
    if (!bitmap || !bitmap->bitmap) return false;

    uint64_t byte_index = index / 8;
    uint8_t bit_index = index % 8;

    if (byte_index >= bitmap->bitmap_size) {
        return false;
    }

    return (bitmap->bitmap[byte_index] & (1 << bit_index)) != 0;
}

void echo_gc_mark_bitmap_clear_all(echo_gc_mark_bitmap_t* bitmap) {
    if (bitmap && bitmap->bitmap) {
        memset(bitmap->bitmap, 0, bitmap->bitmap_size);
    }
}
