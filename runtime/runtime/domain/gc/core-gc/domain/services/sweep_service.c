#include "sweep_service.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <pthread.h>
#include <time.h>
#include "echo/gc.h"
#include "../aggregates/heap.h"

// 清扫工作线程参数
typedef struct sweep_worker_args {
    echo_gc_sweep_service_t* service;
    uint32_t worker_id;
    uint64_t objects_collected;
    uint64_t memory_reclaimed;
} sweep_worker_args_t;

// 清扫工作线程
static void* sweep_worker_thread(void* arg) {
    sweep_worker_args_t* args = (sweep_worker_args_t*)arg;

    // 简化实现：每个线程处理一部分堆
    // 实际需要更复杂的分区策略

    uint64_t heap_size = echo_gc_heap_get_size(args->service->heap);
    uint64_t chunk_size = heap_size / args->service->worker_count;
    uint64_t start_offset = args->worker_id * chunk_size;
    uint64_t end_offset = (args->worker_id + 1) * chunk_size;

    if (args->worker_id == args->service->worker_count - 1) {
        end_offset = heap_size;  // 最后一个线程处理剩余部分
    }

    // 模拟清扫过程
    for (uint64_t offset = start_offset; offset < end_offset; offset += 8) {
        // 检查对象是否标记
        void* obj_addr = echo_gc_heap_get_object_from_index(args->service->heap, offset / 8);

        if (obj_addr && !echo_gc_heap_is_marked(args->service->heap, obj_addr)) {
            // 对象未标记，回收
            args->objects_collected++;
            // 简化：假设每个对象64字节
            args->memory_reclaimed += 64;

            // 添加到空闲链表
            echo_gc_sweep_service_add_to_free_list(args->service, obj_addr, 64);
        }
    }

    return NULL;
}

// 创建清扫服务
echo_gc_sweep_service_t* echo_gc_sweep_service_create(
    struct echo_gc_heap* heap,
    struct echo_gc_collector* collector
) {
    echo_gc_sweep_service_t* service = calloc(1, sizeof(echo_gc_sweep_service_t));
    if (!service) {
        return NULL;
    }

    service->heap = heap;
    service->collector = collector;

    // 初始化空闲链表
    pthread_mutex_init(&service->free_list_mutex, NULL);

    // 默认配置
    service->concurrent = false;  // 默认非并发
    service->worker_count = 1;

    return service;
}

// 销毁清扫服务
void echo_gc_sweep_service_destroy(echo_gc_sweep_service_t* service) {
    if (!service) return;

    // 清理空闲链表
    pthread_mutex_lock(&service->free_list_mutex);
    free_block_t* current = service->free_list;
    while (current) {
        free_block_t* next = current->next;
        free(current);
        current = next;
    }
    pthread_mutex_unlock(&service->free_list_mutex);

    pthread_mutex_destroy(&service->free_list_mutex);
    free(service);
}

// 执行清扫
echo_gc_error_t echo_gc_sweep_service_perform_sweep(
    echo_gc_sweep_service_t* service,
    uint64_t* objects_collected,
    uint64_t* memory_reclaimed
) {
    if (!service) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    struct timespec start_time;
    clock_gettime(CLOCK_MONOTONIC, &start_time);

    // 重置统计
    memset(&service->stats, 0, sizeof(sweep_stats_t));

    if (service->concurrent && service->worker_count > 1) {
        return echo_gc_sweep_service_start_concurrent_sweep(service);
    }

    // 顺序清扫
    uint64_t collected = 0;
    uint64_t reclaimed = 0;

    // 简化实现：调用堆的清扫方法
    echo_gc_error_t err = echo_gc_heap_sweep(service->heap, &collected, &reclaimed);
    if (err != ECHO_GC_SUCCESS) {
        return err;
    }

    // 更新统计
    service->stats.objects_scanned = echo_gc_heap_get_stats(service->heap)->allocated_objects;
    service->stats.objects_collected = collected;
    service->stats.memory_reclaimed = reclaimed;

    // 合并空闲块
    echo_gc_sweep_service_coalesce_free_blocks(service);

    // 计算时间
    struct timespec end_time;
    clock_gettime(CLOCK_MONOTONIC, &end_time);
    service->stats.sweep_time_us =
        (end_time.tv_sec - start_time.tv_sec) * 1000000 +
        (end_time.tv_nsec - start_time.tv_nsec) / 1000;

    // 返回结果
    if (objects_collected) {
        *objects_collected = collected;
    }
    if (memory_reclaimed) {
        *memory_reclaimed = reclaimed;
    }

    return ECHO_GC_SUCCESS;
}

// 并发清扫
echo_gc_error_t echo_gc_sweep_service_start_concurrent_sweep(
    echo_gc_sweep_service_t* service
) {
    if (!service || service->worker_count <= 1) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    pthread_t* threads = calloc(service->worker_count, sizeof(pthread_t));
    sweep_worker_args_t* args = calloc(service->worker_count, sizeof(sweep_worker_args_t));

    if (!threads || !args) {
        free(threads);
        free(args);
        return ECHO_GC_ERROR_OUT_OF_MEMORY;
    }

    // 启动工作线程
    for (uint32_t i = 0; i < service->worker_count; i++) {
        args[i].service = service;
        args[i].worker_id = i;
        args[i].objects_collected = 0;
        args[i].memory_reclaimed = 0;

        if (pthread_create(&threads[i], NULL, sweep_worker_thread, &args[i]) != 0) {
            // 清理已创建的线程
            for (uint32_t j = 0; j < i; j++) {
                pthread_cancel(threads[j]);
                pthread_join(threads[j], NULL);
            }
            free(threads);
            free(args);
            return ECHO_GC_ERROR_SYSTEM_ERROR;
        }
    }

    // 等待所有线程完成
    uint64_t total_collected = 0;
    uint64_t total_reclaimed = 0;

    for (uint32_t i = 0; i < service->worker_count; i++) {
        pthread_join(threads[i], NULL);
        total_collected += args[i].objects_collected;
        total_reclaimed += args[i].memory_reclaimed;
    }

    // 更新服务统计
    service->stats.objects_collected = total_collected;
    service->stats.memory_reclaimed = total_reclaimed;

    free(threads);
    free(args);

    return ECHO_GC_SUCCESS;
}

// 等待并发清扫完成
echo_gc_error_t echo_gc_sweep_service_wait_for_completion(
    echo_gc_sweep_service_t* service,
    uint64_t* objects_collected,
    uint64_t* memory_reclaimed
) {
    // 在简化实现中，perform_sweep已经等待完成
    if (objects_collected) {
        *objects_collected = service->stats.objects_collected;
    }
    if (memory_reclaimed) {
        *memory_reclaimed = service->stats.memory_reclaimed;
    }

    return ECHO_GC_SUCCESS;
}

// 合并空闲块
echo_gc_error_t echo_gc_sweep_service_coalesce_free_blocks(
    echo_gc_sweep_service_t* service
) {
    if (!service) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    pthread_mutex_lock(&service->free_list_mutex);

    if (!service->free_list) {
        pthread_mutex_unlock(&service->free_list_mutex);
        return ECHO_GC_SUCCESS;
    }

    // 简化实现：对空闲链表进行排序和合并
    // 实际需要更复杂的合并算法

    // 统计合并的碎片数
    service->stats.fragments_coalesced = 0;  // 简化

    pthread_mutex_unlock(&service->free_list_mutex);

    return ECHO_GC_SUCCESS;
}

// 从空闲链表分配
void* echo_gc_sweep_service_allocate_from_free_list(
    echo_gc_sweep_service_t* service,
    size_t size
) {
    if (!service) return NULL;

    pthread_mutex_lock(&service->free_list_mutex);

    free_block_t* prev = NULL;
    free_block_t* current = service->free_list;

    // 寻找合适的空闲块（首次适应）
    while (current) {
        if (current->size >= size) {
            // 找到合适的块
            void* result = current->address;

            // 如果块足够大，分割
            if (current->size > size + sizeof(free_block_t)) {
                free_block_t* new_block = malloc(sizeof(free_block_t));
                if (new_block) {
                    new_block->address = (char*)current->address + size;
                    new_block->size = current->size - size;
                    new_block->next = current->next;

                    if (prev) {
                        prev->next = new_block;
                    } else {
                        service->free_list = new_block;
                    }
                }
            } else {
                // 使用整个块
                if (prev) {
                    prev->next = current->next;
                } else {
                    service->free_list = current->next;
                }
                free(current);
            }

            pthread_mutex_unlock(&service->free_list_mutex);
            return result;
        }

        prev = current;
        current = current->next;
    }

    pthread_mutex_unlock(&service->free_list_mutex);
    return NULL;
}

// 添加到空闲链表
void echo_gc_sweep_service_add_to_free_list(
    echo_gc_sweep_service_t* service,
    void* address,
    size_t size
) {
    if (!service || !address || size == 0) return;

    free_block_t* block = malloc(sizeof(free_block_t));
    if (!block) return;

    block->address = address;
    block->size = size;

    pthread_mutex_lock(&service->free_list_mutex);

    // 插入到链表头部（LIFO）
    block->next = service->free_list;
    service->free_list = block;

    pthread_mutex_unlock(&service->free_list_mutex);
}

// 获取统计信息
const sweep_stats_t* echo_gc_sweep_service_get_stats(
    const echo_gc_sweep_service_t* service
) {
    return service ? &service->stats : NULL;
}

// 配置设置
void echo_gc_sweep_service_set_concurrent(
    echo_gc_sweep_service_t* service,
    bool concurrent
) {
    if (service) {
        service->concurrent = concurrent;
    }
}

void echo_gc_sweep_service_set_worker_count(
    echo_gc_sweep_service_t* service,
    uint32_t count
) {
    if (service && count > 0) {
        service->worker_count = count;
    }
}
