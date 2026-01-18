#include "gc_scheduler.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <unistd.h>
#include <pthread.h>
#include <time.h>
#include "../../domain/aggregates/garbage_collector.h"
#include "../../domain/aggregates/heap.h"
#include "../commands/trigger_gc_command.h"

// 默认配置
static const gc_scheduler_config_t DEFAULT_SCHEDULER_CONFIG = {
    .check_interval_ms = 100,           // 100ms检查一次
    .heap_usage_threshold = 0.75,       // 75%触发
    .allocation_rate_threshold = 10 * 1024 * 1024, // 10MB/s
    .enable_adaptive_triggering = true,
    .max_concurrent_gcs = 1
};

// 调度器线程函数
static void* scheduler_thread_func(void* arg) {
    echo_gc_scheduler_t* scheduler = (echo_gc_scheduler_t*)arg;

    while (scheduler->running) {
        // 检查是否暂停
        pthread_mutex_lock(&scheduler->mutex);
        while (scheduler->paused && scheduler->running) {
            pthread_cond_wait(&scheduler->cond, &scheduler->mutex);
        }
        pthread_mutex_unlock(&scheduler->mutex);

        if (!scheduler->running) break;

        // 执行调度检查
        echo_gc_scheduler_perform_check(scheduler);

        // 等待下一个检查周期
        usleep(scheduler->config.check_interval_ms * 1000);
    }

    return NULL;
}

// 执行调度检查
static void echo_gc_scheduler_perform_check(echo_gc_scheduler_t* scheduler) {
    // 检查触发条件
    gc_trigger_conditions_t conditions;
    echo_gc_error_t err = echo_gc_scheduler_check_trigger_conditions(scheduler, &conditions);

    if (err != ECHO_GC_SUCCESS) {
        // 记录错误但不中断调度
        fprintf(stderr, "Failed to check GC trigger conditions: %d\n", err);
        return;
    }

    // 更新统计
    scheduler->stats.checks_performed++;

    // 判断是否需要触发GC
    bool should_trigger = false;
    echo_gc_trigger_reason_t reason = ECHO_GC_TRIGGER_MANUAL;

    if (conditions.current_heap_usage >= scheduler->config.heap_usage_threshold) {
        should_trigger = true;
        reason = ECHO_GC_TRIGGER_HEAP_USAGE;
    } else if (conditions.current_allocation_rate >= scheduler->config.allocation_rate_threshold) {
        should_trigger = true;
        reason = ECHO_GC_TRIGGER_ALLOCATION_RATE;
    } else if (conditions.memory_pressure || conditions.allocation_spike) {
        should_trigger = true;
        reason = ECHO_GC_TRIGGER_EMERGENCY;
    }

    if (should_trigger) {
        // 触发GC
        echo_gc_trigger_command_t command = {
            .reason = reason,
            .force_gc = false,
            .timeout_ms = 30000  // 30秒超时
        };

        err = echo_gc_trigger_gc_command_handle(scheduler->command_handler, &command);

        if (err == ECHO_GC_SUCCESS) {
            scheduler->stats.gcs_triggered++;

            // 触发事件回调
            if (scheduler->on_gc_triggered) {
                scheduler->on_gc_triggered(scheduler->event_context);
            }
        } else {
            // 记录失败
            fprintf(stderr, "Failed to trigger GC: %d\n", err);
        }
    }

    // 更新最后检查时间
    scheduler->stats.last_check_time = time(NULL);
}

// 创建调度器
echo_gc_scheduler_t* echo_gc_scheduler_create(
    struct echo_gc_collector* collector,
    struct echo_gc_heap* heap,
    struct echo_gc_command_handler* command_handler,
    const gc_scheduler_config_t* config
) {
    echo_gc_scheduler_t* scheduler = calloc(1, sizeof(echo_gc_scheduler_t));
    if (!scheduler) {
        return NULL;
    }

    scheduler->collector = collector;
    scheduler->heap = heap;
    scheduler->command_handler = command_handler;

    // 设置配置
    if (config) {
        memcpy(&scheduler->config, config, sizeof(gc_scheduler_config_t));
    } else {
        memcpy(&scheduler->config, &DEFAULT_SCHEDULER_CONFIG, sizeof(gc_scheduler_config_t));
    }

    // 初始化状态
    memset(&scheduler->stats, 0, sizeof(gc_scheduler_stats_t));
    scheduler->running = false;
    scheduler->paused = false;

    // 初始化并发控制
    pthread_mutex_init(&scheduler->mutex, NULL);
    pthread_cond_init(&scheduler->cond, NULL);

    return scheduler;
}

// 销毁调度器
void echo_gc_scheduler_destroy(echo_gc_scheduler_t* scheduler) {
    if (!scheduler) return;

    // 停止调度器
    echo_gc_scheduler_stop(scheduler);

    // 清理资源
    pthread_mutex_destroy(&scheduler->mutex);
    pthread_cond_destroy(&scheduler->cond);

    free(scheduler);
}

// 启动调度器
echo_gc_error_t echo_gc_scheduler_start(echo_gc_scheduler_t* scheduler) {
    if (!scheduler) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    pthread_mutex_lock(&scheduler->mutex);

    if (scheduler->running) {
        pthread_mutex_unlock(&scheduler->mutex);
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    scheduler->running = true;
    scheduler->paused = false;

    // 创建调度线程
    int ret = pthread_create(&scheduler->scheduler_thread, NULL,
                           scheduler_thread_func, scheduler);
    if (ret != 0) {
        scheduler->running = false;
        pthread_mutex_unlock(&scheduler->mutex);
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    pthread_mutex_unlock(&scheduler->mutex);
    return ECHO_GC_SUCCESS;
}

// 停止调度器
echo_gc_error_t echo_gc_scheduler_stop(echo_gc_scheduler_t* scheduler) {
    if (!scheduler) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    pthread_mutex_lock(&scheduler->mutex);

    if (!scheduler->running) {
        pthread_mutex_unlock(&scheduler->mutex);
        return ECHO_GC_SUCCESS;
    }

    scheduler->running = false;
    scheduler->paused = false;
    pthread_cond_broadcast(&scheduler->cond);

    pthread_mutex_unlock(&scheduler->mutex);

    // 等待线程结束
    pthread_join(scheduler->scheduler_thread, NULL);

    return ECHO_GC_SUCCESS;
}

// 暂停调度器
echo_gc_error_t echo_gc_scheduler_pause(echo_gc_scheduler_t* scheduler) {
    if (!scheduler) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    pthread_mutex_lock(&scheduler->mutex);
    scheduler->paused = true;
    pthread_mutex_unlock(&scheduler->mutex);

    return ECHO_GC_SUCCESS;
}

// 恢复调度器
echo_gc_error_t echo_gc_scheduler_resume(echo_gc_scheduler_t* scheduler) {
    if (!scheduler) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    pthread_mutex_lock(&scheduler->mutex);
    scheduler->paused = false;
    pthread_cond_broadcast(&scheduler->cond);
    pthread_mutex_unlock(&scheduler->mutex);

    return ECHO_GC_SUCCESS;
}

// 手动触发GC
echo_gc_error_t echo_gc_scheduler_trigger_gc_now(
    echo_gc_scheduler_t* scheduler,
    echo_gc_trigger_reason_t reason
) {
    if (!scheduler) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    echo_gc_trigger_command_t command = {
        .reason = reason,
        .force_gc = true,  // 强制触发
        .timeout_ms = 60000  // 60秒超时
    };

    return echo_gc_trigger_gc_command_handle(scheduler->command_handler, &command);
}

// 更新配置
echo_gc_error_t echo_gc_scheduler_update_config(
    echo_gc_scheduler_t* scheduler,
    const gc_scheduler_config_t* config
) {
    if (!scheduler || !config) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    // 验证配置
    if (config->check_interval_ms == 0 ||
        config->heap_usage_threshold <= 0.0 || config->heap_usage_threshold >= 1.0) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    pthread_mutex_lock(&scheduler->mutex);
    memcpy(&scheduler->config, config, sizeof(gc_scheduler_config_t));
    pthread_mutex_unlock(&scheduler->mutex);

    return ECHO_GC_SUCCESS;
}

// 状态查询
bool echo_gc_scheduler_is_running(const echo_gc_scheduler_t* scheduler) {
    return scheduler ? scheduler->running : false;
}

bool echo_gc_scheduler_is_paused(const echo_gc_scheduler_t* scheduler) {
    return scheduler ? scheduler->paused : false;
}

const gc_scheduler_stats_t* echo_gc_scheduler_get_stats(const echo_gc_scheduler_t* scheduler) {
    return scheduler ? &scheduler->stats : NULL;
}

// 检查触发条件
echo_gc_error_t echo_gc_scheduler_check_trigger_conditions(
    const echo_gc_scheduler_t* scheduler,
    gc_trigger_conditions_t* conditions
) {
    if (!scheduler || !conditions) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    memset(conditions, 0, sizeof(gc_trigger_conditions_t));

    // 获取堆使用率
    conditions->current_heap_usage = echo_gc_heap_get_used_ratio(scheduler->heap) / 100.0;

    // 获取堆统计信息
    const echo_heap_stats_t* heap_stats = echo_gc_heap_get_stats(scheduler->heap);
    if (heap_stats) {
        // 估算分配速率（简化实现）
        static uint64_t last_allocated = 0;
        static time_t last_check = 0;

        time_t now = time(NULL);
        if (last_check > 0) {
            double time_diff = difftime(now, last_check);
            if (time_diff > 0) {
                conditions->current_allocation_rate =
                    (heap_stats->total_size - last_allocated) / time_diff;
            }
        }

        last_allocated = heap_stats->total_size;
        last_check = now;

        // 检查内存压力（简化）
        conditions->memory_pressure = (conditions->current_heap_usage > 0.9);

        // 检查分配尖峰（简化）
        conditions->allocation_spike = (conditions->current_allocation_rate >
                                      scheduler->config.allocation_rate_threshold * 2);
    }

    // 计算距离上次GC的时间
    const echo_gc_stats_t* gc_stats = echo_gc_collector_get_stats(scheduler->collector);
    if (gc_stats) {
        time_t now = time(NULL);
        conditions->time_since_last_gc = now - gc_stats->last_completed_at.tv_sec;
    }

    return ECHO_GC_SUCCESS;
}

// 设置事件回调
void echo_gc_scheduler_set_event_callback(
    echo_gc_scheduler_t* scheduler,
    void (*callback)(void* context),
    void* context
) {
    if (scheduler) {
        scheduler->on_gc_triggered = callback;
        scheduler->event_context = context;
    }
}
