#ifndef ECHO_GC_HEAP_H
#define ECHO_GC_HEAP_H

#include <stdint.h>
#include <stdbool.h>
#include <pthread.h>
#include "echo/gc.h"

// 前向声明
struct echo_gc_allocator;
struct echo_gc_mark_bitmap;

// 堆配置
typedef struct echo_heap_config {
    uint64_t initial_size;             // 初始堆大小
    uint64_t max_size;                 // 最大堆大小
    uint64_t growth_factor;            // 增长因子（百分比）
    bool enable_large_objects;         // 启用大对象支持
    uint64_t large_object_threshold;   // 大对象阈值
} echo_heap_config_t;

// 堆统计信息
typedef struct echo_heap_stats {
    uint64_t total_size;               // 总大小
    uint64_t used_size;                // 已使用大小
    uint64_t free_size;                // 空闲大小
    uint64_t allocated_objects;        // 已分配对象数
    uint64_t live_objects;             // 存活对象数
    uint64_t fragmentation_ratio;      // 碎片率（0-100）
    uint64_t allocation_rate;          // 分配速率（bytes/sec）
    struct timespec last_gc_time;      // 最后GC时间
} echo_heap_stats_t;

// 堆聚合根
typedef struct echo_gc_heap {
    // 聚合根标识
    char id[64];

    // 配置
    echo_heap_config_t config;

    // 内存区域
    void* arena_start;                 // 堆起始地址
    void* arena_end;                   // 堆结束地址
    uint64_t arena_size;               // 堆大小

    // 分配器
    struct echo_gc_allocator* allocator;

    // 标记位图
    struct echo_gc_mark_bitmap* mark_bitmap;

    // 统计信息
    echo_heap_stats_t stats;

    // 并发控制
    pthread_mutex_t mutex;
    pthread_rwlock_t rwlock;           // 读写锁

    // GC相关
    bool in_gc;                        // 是否正在GC
    volatile uint64_t allocated_bytes_since_gc; // GC后分配的字节数

    // 事件发布
    void (*event_publisher)(const char* event_type, const void* event_data);

    // 内存管理
    void* (*malloc_fn)(size_t);        // 底层分配函数
    void (*free_fn)(void*);           // 底层释放函数
} echo_gc_heap_t;

// 分配器接口
typedef struct echo_gc_allocator {
    // 分配方法
    echo_gc_error_t (*allocate)(
        struct echo_gc_allocator* allocator,
        size_t size,
        echo_obj_type_t type,
        void** result
    );

    // 释放方法
    void (*deallocate)(
        struct echo_gc_allocator* allocator,
        void* ptr
    );

    // 查询方法
    size_t (*get_object_size)(
        struct echo_gc_allocator* allocator,
        void* ptr
    );

    echo_obj_type_t (*get_object_type)(
        struct echo_gc_allocator* allocator,
        void* ptr
    );

    // 统计方法
    void (*get_stats)(
        struct echo_gc_allocator* allocator,
        echo_heap_stats_t* stats
    );

    // GC支持
    void (*prepare_for_gc)(struct echo_gc_allocator* allocator);
    void (*complete_gc)(struct echo_gc_allocator* allocator);
} echo_gc_allocator_t;

// 标记位图
typedef struct echo_gc_mark_bitmap {
    uint8_t* bitmap;                   // 位图数据
    uint64_t bitmap_size;              // 位图大小（字节）
    uint64_t heap_start;               // 堆起始地址
    uint64_t object_alignment;         // 对象对齐
} echo_gc_mark_bitmap_t;

// 生命周期管理
echo_gc_heap_t* echo_gc_heap_create(const echo_heap_config_t* config);
void echo_gc_heap_destroy(echo_gc_heap_t* heap);

// 核心业务方法
echo_gc_error_t echo_gc_heap_allocate(
    echo_gc_heap_t* heap,
    size_t size,
    echo_obj_type_t type,
    void** result
);

echo_gc_error_t echo_gc_heap_mark_object(
    echo_gc_heap_t* heap,
    void* obj
);

bool echo_gc_heap_is_marked(
    const echo_gc_heap_t* heap,
    void* obj
);

echo_gc_error_t echo_gc_heap_sweep(
    echo_gc_heap_t* heap,
    uint64_t* objects_collected,
    uint64_t* memory_reclaimed
);

// 查询方法
uint64_t echo_gc_heap_get_used_ratio(const echo_gc_heap_t* heap);
uint64_t echo_gc_heap_get_size(const echo_gc_heap_t* heap);
const echo_heap_stats_t* echo_gc_heap_get_stats(const echo_gc_heap_t* heap);

// GC生命周期方法
echo_gc_error_t echo_gc_heap_prepare_for_gc(echo_gc_heap_t* heap);
echo_gc_error_t echo_gc_heap_complete_gc(echo_gc_heap_t* heap);

// 堆扩展
echo_gc_error_t echo_gc_heap_grow(echo_gc_heap_t* heap, uint64_t new_size);

// 事件设置
void echo_gc_heap_set_event_publisher(
    echo_gc_heap_t* heap,
    void (*publisher)(const char* event_type, const void* event_data)
);

// 内存函数设置
void echo_gc_heap_set_memory_functions(
    echo_gc_heap_t* heap,
    void* (*malloc_fn)(size_t),
    void (*free_fn)(void*)
);

// 内部辅助方法
bool echo_gc_heap_contains(const echo_gc_heap_t* heap, void* ptr);
uint64_t echo_gc_heap_get_object_index(const echo_gc_heap_t* heap, void* ptr);
void* echo_gc_heap_get_object_from_index(const echo_gc_heap_t* heap, uint64_t index);

// 位图操作
void echo_gc_mark_bitmap_init(echo_gc_mark_bitmap_t* bitmap, uint64_t heap_start, uint64_t heap_size, uint64_t alignment);
void echo_gc_mark_bitmap_destroy(echo_gc_mark_bitmap_t* bitmap);
void echo_gc_mark_bitmap_set(echo_gc_mark_bitmap_t* bitmap, uint64_t index);
void echo_gc_mark_bitmap_clear(echo_gc_mark_bitmap_t* bitmap, uint64_t index);
bool echo_gc_mark_bitmap_get(const echo_gc_mark_bitmap_t* bitmap, uint64_t index);
void echo_gc_mark_bitmap_clear_all(echo_gc_mark_bitmap_t* bitmap);

#endif // ECHO_GC_HEAP_H
