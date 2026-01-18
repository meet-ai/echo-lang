#ifndef ECHO_GC_SWEEP_SERVICE_H
#define ECHO_GC_SWEEP_SERVICE_H

#include <stdint.h>
#include <stdbool.h>
#include "echo/gc.h"

// 前向声明
struct echo_gc_heap;
struct echo_gc_collector;

// 清扫统计信息
typedef struct sweep_stats {
    uint64_t objects_scanned;          // 扫描的对象数
    uint64_t objects_collected;        // 回收的对象数
    uint64_t memory_reclaimed;         // 回收的内存量
    uint64_t fragments_coalesced;      // 合并的碎片数
    uint64_t sweep_time_us;            // 清扫时间
} sweep_stats_t;

// 空闲块信息
typedef struct free_block {
    void* address;                     // 块地址
    size_t size;                       // 块大小
    struct free_block* next;           // 链表下一个
} free_block_t;

// 清扫服务
typedef struct echo_gc_sweep_service {
    struct echo_gc_heap* heap;         // 关联的堆
    struct echo_gc_collector* collector; // 关联的收集器

    // 空闲链表
    free_block_t* free_list;           // 空闲块链表
    pthread_mutex_t free_list_mutex;   // 空闲链表锁

    // 统计信息
    sweep_stats_t stats;

    // 控制标志
    bool concurrent;                   // 是否并发清扫
    uint32_t worker_count;             // 工作线程数
} echo_gc_sweep_service_t;

// 清扫服务生命周期
echo_gc_sweep_service_t* echo_gc_sweep_service_create(
    struct echo_gc_heap* heap,
    struct echo_gc_collector* collector
);

void echo_gc_sweep_service_destroy(echo_gc_sweep_service_t* service);

// 清扫执行
echo_gc_error_t echo_gc_sweep_service_perform_sweep(
    echo_gc_sweep_service_t* service,
    uint64_t* objects_collected,
    uint64_t* memory_reclaimed
);

// 并发清扫
echo_gc_error_t echo_gc_sweep_service_start_concurrent_sweep(
    echo_gc_sweep_service_t* service
);

echo_gc_error_t echo_gc_sweep_service_wait_for_completion(
    echo_gc_sweep_service_t* service,
    uint64_t* objects_collected,
    uint64_t* memory_reclaimed
);

// 内存整理
echo_gc_error_t echo_gc_sweep_service_coalesce_free_blocks(
    echo_gc_sweep_service_t* service
);

// 空闲内存管理
void* echo_gc_sweep_service_allocate_from_free_list(
    echo_gc_sweep_service_t* service,
    size_t size
);

void echo_gc_sweep_service_add_to_free_list(
    echo_gc_sweep_service_t* service,
    void* address,
    size_t size
);

// 统计查询
const sweep_stats_t* echo_gc_sweep_service_get_stats(
    const echo_gc_sweep_service_t* service
);

// 配置设置
void echo_gc_sweep_service_set_concurrent(
    echo_gc_sweep_service_t* service,
    bool concurrent
);

void echo_gc_sweep_service_set_worker_count(
    echo_gc_sweep_service_t* service,
    uint32_t count
);

#endif // ECHO_GC_SWEEP_SERVICE_H
