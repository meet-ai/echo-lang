#ifndef ECHO_GC_SCHEDULER_H
#define ECHO_GC_SCHEDULER_H

#include <stdint.h>
#include <stdbool.h>
#include <pthread.h>
#include "echo/gc.h"

// 前向声明
struct echo_gc_collector;
struct echo_gc_heap;
struct echo_gc_command_handler;

// 调度器配置
typedef struct gc_scheduler_config {
    uint64_t check_interval_ms;         // 检查间隔
    double heap_usage_threshold;        // 堆使用率阈值
    uint64_t allocation_rate_threshold; // 分配速率阈值
    bool enable_adaptive_triggering;    // 启用自适应触发
    uint32_t max_concurrent_gcs;        // 最大并发GC数
} gc_scheduler_config_t;

// 调度器状态
typedef struct gc_scheduler_stats {
    uint64_t checks_performed;          // 执行的检查数
    uint64_t gcs_triggered;             // 触发的GC数
    uint64_t gcs_completed;             // 完成的GC数
    uint64_t gcs_failed;                // 失败的GC数
    uint64_t last_check_time;           // 最后检查时间
    uint64_t average_check_interval;    // 平均检查间隔
} gc_scheduler_stats_t;

// GC调度器应用服务
typedef struct echo_gc_scheduler {
    // 依赖
    struct echo_gc_collector* collector;
    struct echo_gc_heap* heap;
    struct echo_gc_command_handler* command_handler;

    // 配置
    gc_scheduler_config_t config;

    // 状态
    gc_scheduler_stats_t stats;

    // 并发控制
    pthread_t scheduler_thread;
    pthread_mutex_t mutex;
    pthread_cond_t cond;
    bool running;
    bool paused;

    // 事件回调
    void (*on_gc_triggered)(void* context);
    void* event_context;
} echo_gc_scheduler_t;

// 调度器生命周期
echo_gc_scheduler_t* echo_gc_scheduler_create(
    struct echo_gc_collector* collector,
    struct echo_gc_heap* heap,
    struct echo_gc_command_handler* command_handler,
    const gc_scheduler_config_t* config
);

void echo_gc_scheduler_destroy(echo_gc_scheduler_t* scheduler);

// 调度器控制
echo_gc_error_t echo_gc_scheduler_start(echo_gc_scheduler_t* scheduler);
echo_gc_error_t echo_gc_scheduler_stop(echo_gc_scheduler_t* scheduler);
echo_gc_error_t echo_gc_scheduler_pause(echo_gc_scheduler_t* scheduler);
echo_gc_error_t echo_gc_scheduler_resume(echo_gc_scheduler_t* scheduler);

// 手动触发
echo_gc_error_t echo_gc_scheduler_trigger_gc_now(
    echo_gc_scheduler_t* scheduler,
    echo_gc_trigger_reason_t reason
);

// 配置管理
echo_gc_error_t echo_gc_scheduler_update_config(
    echo_gc_scheduler_t* scheduler,
    const gc_scheduler_config_t* config
);

// 状态查询
bool echo_gc_scheduler_is_running(const echo_gc_scheduler_t* scheduler);
bool echo_gc_scheduler_is_paused(const echo_gc_scheduler_t* scheduler);
const gc_scheduler_stats_t* echo_gc_scheduler_get_stats(const echo_gc_scheduler_t* scheduler);

// 触发条件检查
typedef struct gc_trigger_conditions {
    double current_heap_usage;          // 当前堆使用率
    uint64_t current_allocation_rate;   // 当前分配速率
    bool memory_pressure;               // 内存压力
    bool allocation_spike;              // 分配尖峰
    uint64_t time_since_last_gc;        // 距离上次GC的时间
} gc_trigger_conditions_t;

echo_gc_error_t echo_gc_scheduler_check_trigger_conditions(
    const echo_gc_scheduler_t* scheduler,
    gc_trigger_conditions_t* conditions
);

// 事件设置
void echo_gc_scheduler_set_event_callback(
    echo_gc_scheduler_t* scheduler,
    void (*callback)(void* context),
    void* context
);

#endif // ECHO_GC_SCHEDULER_H
