#include "core_gc.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <pthread.h>
#include <unistd.h>
#include <inttypes.h>
#include "domain/aggregates/garbage_collector.h"
#include "domain/aggregates/heap.h"
#include "domain/services/mark_service.h"
#include "domain/services/sweep_service.h"
#include "application/commands/trigger_gc_command.h"
#include "application/services/gc_scheduler.h"
#include "ports/gc_ports.h"

// 全局GC实例
static struct {
    echo_gc_collector_t* collector;
    echo_gc_heap_t* heap;
    echo_gc_mark_service_t* mark_service;
    echo_gc_sweep_service_t* sweep_service;
    echo_gc_command_handler_t* command_handler;
    echo_gc_scheduler_t* scheduler;
    echo_gc_write_barrier_t* write_barrier;

    // 配置
    echo_gc_config_t config;

    // 状态
    bool initialized;
    pthread_mutex_t mutex;

    // 事件总线
    echo_gc_event_publisher_t event_publisher;
} g_gc_instance = {0};

// 默认配置
static const echo_gc_config_t DEFAULT_GC_CONFIG = {
    .trigger_ratio = 0.75,
    .min_heap_size = 16 * 1024 * 1024,      // 16MB
    .max_heap_size = 1024 * 1024 * 1024,    // 1GB
    .target_pause_time_us = 10000,          // 10ms
    .max_pause_time_us = 100000,            // 100ms
    .max_concurrent_workers = 4,
    .enable_tracing = false,
    .enable_profiling = false
};

// 事件发布器
static void default_event_publisher(const char* event_type, const void* event_data) {
    // 默认实现：打印事件
    printf("[GC Event] %s\n", event_type);
}

// 初始化GC
echo_gc_error_t echo_gc_init(const echo_gc_config_t* config) {
    pthread_mutex_lock(&g_gc_instance.mutex);

    if (g_gc_instance.initialized) {
        pthread_mutex_unlock(&g_gc_instance.mutex);
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    // 设置配置
    if (config) {
        memcpy(&g_gc_instance.config, config, sizeof(echo_gc_config_t));
    } else {
        memcpy(&g_gc_instance.config, &DEFAULT_GC_CONFIG, sizeof(echo_gc_config_t));
    }

    // 创建堆
    echo_heap_config_t heap_config = {
        .initial_size = g_gc_instance.config.min_heap_size,
        .max_size = g_gc_instance.config.max_heap_size,
        .growth_factor = 150,
        .enable_large_objects = true,
        .large_object_threshold = 32 * 1024
    };

    g_gc_instance.heap = echo_gc_heap_create(&heap_config);
    if (!g_gc_instance.heap) {
        pthread_mutex_unlock(&g_gc_instance.mutex);
        return ECHO_GC_ERROR_OUT_OF_MEMORY;
    }

    // 创建垃圾回收器
    g_gc_instance.collector = echo_gc_collector_create(&g_gc_instance.config);
    if (!g_gc_instance.collector) {
        echo_gc_heap_destroy(g_gc_instance.heap);
        g_gc_instance.heap = NULL;
        pthread_mutex_unlock(&g_gc_instance.mutex);
        return ECHO_GC_ERROR_OUT_OF_MEMORY;
    }

    // 关联堆和回收器
    // 注意：这里需要扩展聚合根接口来设置关联

    // 创建标记服务
    g_gc_instance.mark_service = echo_gc_mark_service_create(
        g_gc_instance.heap, g_gc_instance.collector);
    if (!g_gc_instance.mark_service) {
        echo_gc_collector_destroy(g_gc_instance.collector);
        echo_gc_heap_destroy(g_gc_instance.heap);
        g_gc_instance.collector = NULL;
        g_gc_instance.heap = NULL;
        pthread_mutex_unlock(&g_gc_instance.mutex);
        return ECHO_GC_ERROR_OUT_OF_MEMORY;
    }

    // 设置根对象扫描器
    echo_gc_mark_service_set_root_scanner(
        g_gc_instance.mark_service,
        echo_gc_scan_stack_roots, NULL);

    // 创建清扫服务
    g_gc_instance.sweep_service = echo_gc_sweep_service_create(
        g_gc_instance.heap, g_gc_instance.collector);
    if (!g_gc_instance.sweep_service) {
        echo_gc_mark_service_destroy(g_gc_instance.mark_service);
        echo_gc_collector_destroy(g_gc_instance.collector);
        echo_gc_heap_destroy(g_gc_instance.heap);
        g_gc_instance.mark_service = NULL;
        g_gc_instance.collector = NULL;
        g_gc_instance.heap = NULL;
        pthread_mutex_unlock(&g_gc_instance.mutex);
        return ECHO_GC_ERROR_OUT_OF_MEMORY;
    }

    // 创建命令处理器
    g_gc_instance.command_handler = echo_gc_command_handler_create(
        g_gc_instance.collector,
        g_gc_instance.heap,
        g_gc_instance.mark_service,
        g_gc_instance.sweep_service);
    if (!g_gc_instance.command_handler) {
        echo_gc_sweep_service_destroy(g_gc_instance.sweep_service);
        echo_gc_mark_service_destroy(g_gc_instance.mark_service);
        echo_gc_collector_destroy(g_gc_instance.collector);
        echo_gc_heap_destroy(g_gc_instance.heap);
        g_gc_instance.sweep_service = NULL;
        g_gc_instance.mark_service = NULL;
        g_gc_instance.collector = NULL;
        g_gc_instance.heap = NULL;
        pthread_mutex_unlock(&g_gc_instance.mutex);
        return ECHO_GC_ERROR_OUT_OF_MEMORY;
    }

    // 创建写屏障
    g_gc_instance.write_barrier = echo_gc_write_barrier_create(g_gc_instance.heap);
    if (!g_gc_instance.write_barrier) {
        echo_gc_command_handler_destroy(g_gc_instance.command_handler);
        echo_gc_sweep_service_destroy(g_gc_instance.sweep_service);
        echo_gc_mark_service_destroy(g_gc_instance.mark_service);
        echo_gc_collector_destroy(g_gc_instance.collector);
        echo_gc_heap_destroy(g_gc_instance.heap);
        g_gc_instance.command_handler = NULL;
        g_gc_instance.sweep_service = NULL;
        g_gc_instance.mark_service = NULL;
        g_gc_instance.collector = NULL;
        g_gc_instance.heap = NULL;
        pthread_mutex_unlock(&g_gc_instance.mutex);
        return ECHO_GC_ERROR_OUT_OF_MEMORY;
    }

    // 启用写屏障
    echo_gc_write_barrier_enable(g_gc_instance.write_barrier);

    // 设置事件发布器
    echo_gc_collector_set_event_publisher(g_gc_instance.collector, default_event_publisher);
    echo_gc_heap_set_event_publisher(g_gc_instance.heap, default_event_publisher);

    // 创建调度器
    gc_scheduler_config_t scheduler_config = {
        .check_interval_ms = 100,
        .heap_usage_threshold = g_gc_instance.config.trigger_ratio,
        .allocation_rate_threshold = 10 * 1024 * 1024, // 10MB/s
        .enable_adaptive_triggering = true,
        .max_concurrent_gcs = 1
    };

    g_gc_instance.scheduler = echo_gc_scheduler_create(
        g_gc_instance.collector,
        g_gc_instance.heap,
        g_gc_instance.command_handler,
        &scheduler_config);

    if (!g_gc_instance.scheduler) {
        echo_gc_write_barrier_destroy(g_gc_instance.write_barrier);
        echo_gc_command_handler_destroy(g_gc_instance.command_handler);
        echo_gc_sweep_service_destroy(g_gc_instance.sweep_service);
        echo_gc_mark_service_destroy(g_gc_instance.mark_service);
        echo_gc_collector_destroy(g_gc_instance.collector);
        echo_gc_heap_destroy(g_gc_instance.heap);
        g_gc_instance.write_barrier = NULL;
        g_gc_instance.command_handler = NULL;
        g_gc_instance.sweep_service = NULL;
        g_gc_instance.mark_service = NULL;
        g_gc_instance.collector = NULL;
        g_gc_instance.heap = NULL;
        pthread_mutex_unlock(&g_gc_instance.mutex);
        return ECHO_GC_ERROR_OUT_OF_MEMORY;
    }

    // 启动调度器
    echo_gc_error_t err = echo_gc_scheduler_start(g_gc_instance.scheduler);
    if (err != ECHO_GC_SUCCESS) {
        echo_gc_scheduler_destroy(g_gc_instance.scheduler);
        echo_gc_write_barrier_destroy(g_gc_instance.write_barrier);
        echo_gc_command_handler_destroy(g_gc_instance.command_handler);
        echo_gc_sweep_service_destroy(g_gc_instance.sweep_service);
        echo_gc_mark_service_destroy(g_gc_instance.mark_service);
        echo_gc_collector_destroy(g_gc_instance.collector);
        echo_gc_heap_destroy(g_gc_instance.heap);
        g_gc_instance.scheduler = NULL;
        g_gc_instance.write_barrier = NULL;
        g_gc_instance.command_handler = NULL;
        g_gc_instance.sweep_service = NULL;
        g_gc_instance.mark_service = NULL;
        g_gc_instance.collector = NULL;
        g_gc_instance.heap = NULL;
        pthread_mutex_unlock(&g_gc_instance.mutex);
        return err;
    }

    g_gc_instance.initialized = true;
    pthread_mutex_unlock(&g_gc_instance.mutex);

    return ECHO_GC_SUCCESS;
}

// 关闭GC
void echo_gc_shutdown(void) {
    pthread_mutex_lock(&g_gc_instance.mutex);

    if (!g_gc_instance.initialized) {
        pthread_mutex_unlock(&g_gc_instance.mutex);
        return;
    }

    // 停止调度器
    if (g_gc_instance.scheduler) {
        echo_gc_scheduler_stop(g_gc_instance.scheduler);
        echo_gc_scheduler_destroy(g_gc_instance.scheduler);
        g_gc_instance.scheduler = NULL;
    }

    // 禁用写屏障
    if (g_gc_instance.write_barrier) {
        echo_gc_write_barrier_disable(g_gc_instance.write_barrier);
        echo_gc_write_barrier_destroy(g_gc_instance.write_barrier);
        g_gc_instance.write_barrier = NULL;
    }

    // 销毁其他组件
    if (g_gc_instance.command_handler) {
        echo_gc_command_handler_destroy(g_gc_instance.command_handler);
        g_gc_instance.command_handler = NULL;
    }

    if (g_gc_instance.sweep_service) {
        echo_gc_sweep_service_destroy(g_gc_instance.sweep_service);
        g_gc_instance.sweep_service = NULL;
    }

    if (g_gc_instance.mark_service) {
        echo_gc_mark_service_destroy(g_gc_instance.mark_service);
        g_gc_instance.mark_service = NULL;
    }

    if (g_gc_instance.collector) {
        echo_gc_collector_destroy(g_gc_instance.collector);
        g_gc_instance.collector = NULL;
    }

    if (g_gc_instance.heap) {
        echo_gc_heap_destroy(g_gc_instance.heap);
        g_gc_instance.heap = NULL;
    }

    g_gc_instance.initialized = false;
    pthread_mutex_unlock(&g_gc_instance.mutex);
}

// 配置管理
echo_gc_error_t echo_gc_get_config(echo_gc_config_t* config) {
    if (!config) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    pthread_mutex_lock(&g_gc_instance.mutex);
    if (!g_gc_instance.initialized) {
        pthread_mutex_unlock(&g_gc_instance.mutex);
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    memcpy(config, &g_gc_instance.config, sizeof(echo_gc_config_t));
    pthread_mutex_unlock(&g_gc_instance.mutex);

    return ECHO_GC_SUCCESS;
}

echo_gc_error_t echo_gc_update_config(const echo_gc_config_t* config) {
    if (!config) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    pthread_mutex_lock(&g_gc_instance.mutex);
    if (!g_gc_instance.initialized) {
        pthread_mutex_unlock(&g_gc_instance.mutex);
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    // 验证配置
    if (config->trigger_ratio <= 0.0 || config->trigger_ratio >= 1.0) {
        pthread_mutex_unlock(&g_gc_instance.mutex);
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    // 更新配置
    memcpy(&g_gc_instance.config, config, sizeof(echo_gc_config_t));

    // 更新相关组件
    echo_gc_collector_update_config(g_gc_instance.collector, config);

    // 更新调度器配置
    gc_scheduler_config_t scheduler_config = {
        .check_interval_ms = 100,
        .heap_usage_threshold = config->trigger_ratio,
        .allocation_rate_threshold = 10 * 1024 * 1024,
        .enable_adaptive_triggering = true,
        .max_concurrent_gcs = 1
    };
    echo_gc_scheduler_update_config(g_gc_instance.scheduler, &scheduler_config);

    pthread_mutex_unlock(&g_gc_instance.mutex);

    return ECHO_GC_SUCCESS;
}

// 堆管理
echo_gc_heap_t* echo_gc_get_heap(void) {
    return g_gc_instance.initialized ? g_gc_instance.heap : NULL;
}

echo_gc_error_t echo_gc_allocate(size_t size, echo_obj_type_t type, void** ptr) {
    if (!ptr) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    if (!g_gc_instance.initialized || !g_gc_instance.heap) {
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    return echo_gc_heap_allocate(g_gc_instance.heap, size, type, ptr);
}

echo_gc_error_t echo_gc_collect(void) {
    if (!g_gc_instance.initialized) {
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    return echo_gc_scheduler_trigger_gc_now(g_gc_instance.scheduler, ECHO_GC_TRIGGER_MANUAL);
}

// 垃圾回收器控制
echo_gc_collector_t* echo_gc_get_collector(void) {
    return g_gc_instance.initialized ? g_gc_instance.collector : NULL;
}

echo_gc_status_t echo_gc_get_status(void) {
    if (!g_gc_instance.initialized || !g_gc_instance.collector) {
        return ECHO_GC_STATUS_ERROR;
    }

    return echo_gc_collector_get_status(g_gc_instance.collector);
}

echo_gc_phase_t echo_gc_get_phase(void) {
    if (!g_gc_instance.initialized || !g_gc_instance.collector) {
        return ECHO_GC_PHASE_IDLE;
    }

    return echo_gc_collector_get_phase(g_gc_instance.collector);
}

// 统计和监控
echo_gc_error_t echo_gc_get_stats(echo_gc_stats_t* stats) {
    if (!stats) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    if (!g_gc_instance.initialized || !g_gc_instance.collector) {
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    const echo_gc_stats_t* gc_stats = echo_gc_collector_get_stats(g_gc_instance.collector);
    if (!gc_stats) {
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    memcpy(stats, gc_stats, sizeof(echo_gc_stats_t));
    return ECHO_GC_SUCCESS;
}

echo_gc_error_t echo_gc_get_metrics(echo_gc_metrics_t* metrics) {
    if (!metrics) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    if (!g_gc_instance.initialized) {
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    // 计算当前指标（简化实现）
    memset(metrics, 0, sizeof(echo_gc_metrics_t));

    const echo_heap_stats_t* heap_stats = echo_gc_heap_get_stats(g_gc_instance.heap);
    if (heap_stats) {
        metrics->heap_usage_ratio = (double)heap_stats->used_size / heap_stats->total_size;
        metrics->allocation_rate_bps = heap_stats->allocation_rate;
        metrics->total_objects = heap_stats->allocated_objects;
        metrics->live_objects = heap_stats->live_objects;
        metrics->memory_reclaimed_bytes = heap_stats->total_memory_reclaimed;
    }

    clock_gettime(CLOCK_MONOTONIC, &metrics->collected_at);

    return ECHO_GC_SUCCESS;
}

// 写屏障（编译器插入）
void echo_gc_write_barrier(void** slot, void* new_value) {
    if (g_gc_instance.initialized && g_gc_instance.write_barrier) {
        echo_gc_write_barrier_write(g_gc_instance.write_barrier, slot, new_value);
    } else {
        *slot = new_value;
    }
}

// 调试接口
void echo_gc_dump_heap(void) {
    if (!g_gc_instance.initialized || !g_gc_instance.heap) {
        printf("GC not initialized\n");
        return;
    }

    const echo_heap_stats_t* stats = echo_gc_heap_get_stats(g_gc_instance.heap);
    if (stats) {
        printf("Heap Statistics:\n");
        printf("  Total Size: %" PRIu64 " bytes\n", stats->total_size);
        printf("  Used Size: %" PRIu64 " bytes\n", stats->used_size);
        printf("  Free Size: %" PRIu64 " bytes\n", stats->free_size);
        printf("  Live Objects: %" PRIu64 "\n", stats->live_objects);
        printf("  Fragmentation: %" PRIu64 "%%\n", stats->fragmentation_ratio);
    }
}

void echo_gc_dump_stats(void) {
    if (!g_gc_instance.initialized || !g_gc_instance.collector) {
        printf("GC not initialized\n");
        return;
    }

    const echo_gc_stats_t* stats = echo_gc_collector_get_stats(g_gc_instance.collector);
    if (stats) {
        printf("GC Statistics:\n");
        printf("  Total Cycles: %" PRIu64 "\n", stats->total_cycles);
        printf("  Successful Cycles: %" PRIu64 "\n", stats->successful_cycles);
        printf("  Failed Cycles: %" PRIu64 "\n", stats->failed_cycles);
        printf("  Total GC Time: %" PRIu64 " us\n", stats->total_gc_time_us);
        printf("  Total Pause Time: %" PRIu64 " us\n", stats->total_pause_time_us);
        printf("  Total Memory Allocated: %" PRIu64 " bytes\n", stats->total_memory_allocated);
        printf("  Total Memory Reclaimed: %" PRIu64 " bytes\n", stats->total_memory_reclaimed);
    }
}

bool echo_gc_validate_heap(void) {
    // 简化实现
    return g_gc_instance.initialized && g_gc_instance.heap != NULL;
}

// 事件回调注册
echo_gc_error_t echo_gc_register_event_callback(const char* event_type, echo_gc_event_callback_t callback) {
    // 简化实现 - 暂时忽略参数
    (void)event_type;
    (void)callback;
    return ECHO_GC_SUCCESS;
}
