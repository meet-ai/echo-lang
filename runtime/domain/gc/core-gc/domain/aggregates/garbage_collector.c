#include "garbage_collector.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <uuid/uuid.h>
#include <unistd.h>
#include "echo/gc.h"

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

// 创建垃圾回收器
echo_gc_collector_t* echo_gc_collector_create(const echo_gc_config_t* config) {
    echo_gc_collector_t* collector = calloc(1, sizeof(echo_gc_collector_t));
    if (!collector) {
        return NULL;
    }

    // 生成唯一ID
    generate_uuid(collector->id, sizeof(collector->id));

    // 初始化状态
    collector->status = ECHO_GC_STATUS_IDLE;
    collector->phase = ECHO_GC_PHASE_IDLE;

    // 复制配置
    if (config) {
        memcpy(&collector->config, config, sizeof(echo_gc_config_t));
    } else {
        // 使用默认配置
        collector->config.trigger_ratio = 0.75;
        collector->config.min_heap_size = 16 * 1024 * 1024;  // 16MB
        collector->config.max_heap_size = 1024 * 1024 * 1024; // 1GB
        collector->config.target_pause_time_us = 10000;       // 10ms
        collector->config.max_pause_time_us = 100000;         // 100ms
        collector->config.max_concurrent_workers = 4;
        collector->config.enable_tracing = false;
        collector->config.enable_profiling = false;
    }

    // 初始化统计信息
    memset(&collector->stats, 0, sizeof(echo_gc_stats_t));

    // 初始化并发控制
    pthread_mutex_init(&collector->mutex, NULL);
    pthread_cond_init(&collector->cond, NULL);

    // 设置时间戳
    get_current_time(&collector->created_at);

    return collector;
}

// 销毁垃圾回收器
void echo_gc_collector_destroy(echo_gc_collector_t* collector) {
    if (!collector) {
        return;
    }

    // 等待GC完成
    pthread_mutex_lock(&collector->mutex);
    while (collector->status == ECHO_GC_STATUS_RUNNING) {
        pthread_cond_wait(&collector->cond, &collector->mutex);
    }
    pthread_mutex_unlock(&collector->mutex);

    // 清理资源
    if (collector->current_cycle) {
        free(collector->current_cycle);
    }

    pthread_mutex_destroy(&collector->mutex);
    pthread_cond_destroy(&collector->cond);

    free(collector);
}

// 触发GC - 核心业务方法
echo_gc_error_t echo_gc_collector_trigger_gc(
    echo_gc_collector_t* collector,
    echo_gc_trigger_reason_t reason
) {
    if (!collector) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    pthread_mutex_lock(&collector->mutex);

    // 状态验证
    if (!echo_gc_collector_can_trigger_gc(collector)) {
        pthread_mutex_unlock(&collector->mutex);
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    // 创建新的GC周期
    echo_gc_cycle_t* cycle = calloc(1, sizeof(echo_gc_cycle_t));
    if (!cycle) {
        pthread_mutex_unlock(&collector->mutex);
        return ECHO_GC_ERROR_OUT_OF_MEMORY;
    }

    generate_uuid(cycle->id, sizeof(cycle->id));
    strcpy(cycle->collector_id, collector->id);
    get_current_time(&cycle->start_time);
    cycle->trigger_reason = reason;

    // 获取触发时的堆使用率（这里需要heap接口）
    if (collector->heap) {
        // cycle->heap_usage_at_trigger = heap_get_usage_ratio(collector->heap);
    }

    // 更新收集器状态
    collector->status = ECHO_GC_STATUS_RUNNING;
    collector->phase = ECHO_GC_PHASE_MARK;
    collector->current_cycle = cycle;
    get_current_time(&collector->last_triggered_at);

    // 更新统计
    collector->stats.total_cycles++;

    pthread_mutex_unlock(&collector->mutex);

    // 发布领域事件
    if (collector->event_publisher) {
        struct {
            const char* gc_id;
            echo_gc_trigger_reason_t reason;
            struct timespec timestamp;
        } event_data = {
            .gc_id = collector->id,
            .reason = reason,
            .timestamp = cycle->start_time
        };
        collector->event_publisher("gc.triggered", &event_data);
    }

    return ECHO_GC_SUCCESS;
}

// 完成GC周期
echo_gc_error_t echo_gc_collector_complete_gc(
    echo_gc_collector_t* collector,
    const echo_gc_metrics_t* metrics
) {
    if (!collector || !collector->current_cycle) {
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    pthread_mutex_lock(&collector->mutex);

    if (collector->status != ECHO_GC_STATUS_RUNNING) {
        pthread_mutex_unlock(&collector->mutex);
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    // 完成GC周期
    echo_gc_cycle_t* cycle = collector->current_cycle;
    get_current_time(&cycle->end_time);
    get_current_time(&collector->last_completed_at);

    // 计算周期统计
    cycle->pause_time_us = metrics->total_pause_time_us;
    cycle->objects_collected = metrics->total_objects - metrics->live_objects;
    cycle->memory_reclaimed = metrics->memory_reclaimed_bytes;

    // 更新收集器状态
    collector->status = ECHO_GC_STATUS_IDLE;
    collector->phase = ECHO_GC_PHASE_IDLE;

    // 更新统计信息
    collector->stats.successful_cycles++;
    collector->stats.total_gc_time_us += metrics->gc_duration_us;
    collector->stats.total_pause_time_us += metrics->total_pause_time_us;
    collector->stats.total_memory_allocated += metrics->allocation_rate_bps; // 估算
    collector->stats.total_memory_reclaimed += metrics->memory_reclaimed_bytes;

    if (collector->stats.successful_cycles > 0) {
        collector->stats.average_gc_duration_us =
            collector->stats.total_gc_time_us / collector->stats.successful_cycles;
    }

    collector->stats.last_completed_at = cycle->end_time;

    pthread_cond_broadcast(&collector->cond);
    pthread_mutex_unlock(&collector->mutex);

    // 发布领域事件
    if (collector->event_publisher) {
        struct {
            const char* gc_id;
            const char* cycle_id;
            const echo_gc_metrics_t* metrics;
            struct timespec timestamp;
        } event_data = {
            .gc_id = collector->id,
            .cycle_id = cycle->id,
            .metrics = metrics,
            .timestamp = cycle->end_time
        };
        collector->event_publisher("gc.cycle.completed", &event_data);
    }

    // 清理周期数据（保留给监控系统）
    // free(cycle);  // 暂时保留，可通过其他方式清理

    return ECHO_GC_SUCCESS;
}

// 查询方法
echo_gc_status_t echo_gc_collector_get_status(const echo_gc_collector_t* collector) {
    return collector ? collector->status : ECHO_GC_STATUS_ERROR;
}

echo_gc_phase_t echo_gc_collector_get_phase(const echo_gc_collector_t* collector) {
    return collector ? collector->phase : ECHO_GC_PHASE_IDLE;
}

const echo_gc_config_t* echo_gc_collector_get_config(const echo_gc_collector_t* collector) {
    return collector ? &collector->config : NULL;
}

const echo_gc_stats_t* echo_gc_collector_get_stats(const echo_gc_collector_t* collector) {
    return collector ? &collector->stats : NULL;
}

// 配置管理
echo_gc_error_t echo_gc_collector_update_config(
    echo_gc_collector_t* collector,
    const echo_gc_config_t* config
) {
    if (!collector || !config) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    // 验证配置
    if (config->trigger_ratio <= 0.0 || config->trigger_ratio >= 1.0) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    if (config->min_heap_size >= config->max_heap_size) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    pthread_mutex_lock(&collector->mutex);

    // 只允许在GC空闲时更新配置
    if (collector->status == ECHO_GC_STATUS_RUNNING) {
        pthread_mutex_unlock(&collector->mutex);
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    memcpy(&collector->config, config, sizeof(echo_gc_config_t));
    pthread_mutex_unlock(&collector->mutex);

    return ECHO_GC_SUCCESS;
}

// 事件设置
void echo_gc_collector_set_event_publisher(
    echo_gc_collector_t* collector,
    void (*publisher)(const char* event_type, const void* event_data)
) {
    if (collector) {
        collector->event_publisher = publisher;
    }
}

// 内部方法
echo_gc_error_t echo_gc_collector_start_mark_phase(echo_gc_collector_t* collector) {
    if (!collector || !collector->current_cycle) {
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    get_current_time(&collector->current_cycle->mark_phase.start_time);

    if (collector->event_publisher) {
        struct {
            const char* gc_id;
            const char* phase_id;
            struct timespec start_time;
        } event_data = {
            .gc_id = collector->id,
            .phase_id = "mark",
            .start_time = collector->current_cycle->mark_phase.start_time
        };
        collector->event_publisher("gc.mark.started", &event_data);
    }

    return ECHO_GC_SUCCESS;
}

echo_gc_error_t echo_gc_collector_complete_mark_phase(
    echo_gc_collector_t* collector,
    uint64_t marked_objects,
    uint64_t processed_bytes
) {
    if (!collector || !collector->current_cycle) {
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    echo_gc_cycle_t* cycle = collector->current_cycle;
    get_current_time(&cycle->mark_phase.end_time);
    cycle->mark_phase.marked_objects = marked_objects;
    cycle->mark_phase.processed_bytes = processed_bytes;

    if (collector->event_publisher) {
        struct {
            const char* gc_id;
            const char* phase_id;
            uint64_t marked_objects;
            uint64_t processed_bytes;
            struct timespec duration;
        } event_data = {
            .gc_id = collector->id,
            .phase_id = "mark",
            .marked_objects = marked_objects,
            .processed_bytes = processed_bytes,
            .duration = cycle->mark_phase.end_time
        };
        collector->event_publisher("gc.mark.completed", &event_data);
    }

    return ECHO_GC_SUCCESS;
}

echo_gc_error_t echo_gc_collector_start_sweep_phase(echo_gc_collector_t* collector) {
    if (!collector || !collector->current_cycle) {
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    get_current_time(&collector->current_cycle->sweep_phase.start_time);

    if (collector->event_publisher) {
        struct {
            const char* gc_id;
            const char* phase_id;
            struct timespec start_time;
        } event_data = {
            .gc_id = collector->id,
            .phase_id = "sweep",
            .start_time = collector->current_cycle->sweep_phase.start_time
        };
        collector->event_publisher("gc.sweep.started", &event_data);
    }

    return ECHO_GC_SUCCESS;
}

echo_gc_error_t echo_gc_collector_complete_sweep_phase(
    echo_gc_collector_t* collector,
    uint64_t objects_collected,
    uint64_t memory_reclaimed
) {
    if (!collector || !collector->current_cycle) {
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    echo_gc_cycle_t* cycle = collector->current_cycle;
    get_current_time(&cycle->sweep_phase.end_time);
    cycle->objects_collected = objects_collected;
    cycle->memory_reclaimed = memory_reclaimed;

    if (collector->event_publisher) {
        struct {
            const char* gc_id;
            const char* phase_id;
            uint64_t objects_collected;
            uint64_t memory_reclaimed;
            struct timespec duration;
        } event_data = {
            .gc_id = collector->id,
            .phase_id = "sweep",
            .objects_collected = objects_collected,
            .memory_reclaimed = memory_reclaimed,
            .duration = cycle->sweep_phase.end_time
        };
        collector->event_publisher("gc.sweep.completed", &event_data);
    }

    return ECHO_GC_SUCCESS;
}

// 状态验证
bool echo_gc_collector_can_trigger_gc(const echo_gc_collector_t* collector) {
    if (!collector) {
        return false;
    }

    return collector->status == ECHO_GC_STATUS_IDLE;
}

bool echo_gc_collector_is_running(const echo_gc_collector_t* collector) {
    if (!collector) {
        return false;
    }

    return collector->status == ECHO_GC_STATUS_RUNNING;
}
