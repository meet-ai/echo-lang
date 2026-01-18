#ifndef ECHO_GC_GARBAGE_COLLECTOR_H
#define ECHO_GC_GARBAGE_COLLECTOR_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>
#include <pthread.h>
#include "echo/gc.h"

// 前向声明
struct echo_gc_cycle;
struct echo_gc_heap;

// 垃圾回收器聚合根
typedef struct echo_gc_collector {
    // 聚合根标识
    char id[64];                      // 唯一标识符

    // GC状态
    echo_gc_status_t status;          // 当前状态
    echo_gc_phase_t phase;            // 当前阶段

    // 配置
    echo_gc_config_t config;          // GC配置

    // 关联对象
    struct echo_gc_heap* heap;        // 关联的堆
    struct echo_gc_cycle* current_cycle; // 当前GC周期

    // 统计信息
    echo_gc_stats_t stats;            // 执行统计

    // 并发控制
    pthread_mutex_t mutex;            // 状态保护锁
    pthread_cond_t cond;              // 条件变量

    // 时间戳
    struct timespec created_at;       // 创建时间
    struct timespec last_triggered_at; // 最后触发时间
    struct timespec last_completed_at; // 最后完成时间

    // 事件发布
    void (*event_publisher)(const char* event_type, const void* event_data);
} echo_gc_collector_t;

// GC周期实体
typedef struct echo_gc_cycle {
    char id[64];                      // 周期唯一标识
    char collector_id[64];            // 所属收集器ID

    // 时间信息
    struct timespec start_time;
    struct timespec end_time;

    // 阶段信息
    struct {
        struct timespec start_time;
        struct timespec end_time;
        uint64_t marked_objects;
        uint64_t processed_bytes;
    } mark_phase, sweep_phase;

    // 统计信息
    uint64_t objects_collected;
    uint64_t memory_reclaimed;
    uint64_t pause_time_us;

    // 触发信息
    echo_gc_trigger_reason_t trigger_reason;
    double heap_usage_at_trigger;
} echo_gc_cycle_t;

// 生命周期管理
echo_gc_collector_t* echo_gc_collector_create(const echo_gc_config_t* config);
void echo_gc_collector_destroy(echo_gc_collector_t* collector);

// 业务方法
echo_gc_error_t echo_gc_collector_trigger_gc(
    echo_gc_collector_t* collector,
    echo_gc_trigger_reason_t reason
);

echo_gc_error_t echo_gc_collector_complete_gc(
    echo_gc_collector_t* collector,
    const echo_gc_metrics_t* metrics
);

// 查询方法
echo_gc_status_t echo_gc_collector_get_status(const echo_gc_collector_t* collector);
echo_gc_phase_t echo_gc_collector_get_phase(const echo_gc_collector_t* collector);
const echo_gc_config_t* echo_gc_collector_get_config(const echo_gc_collector_t* collector);
const echo_gc_stats_t* echo_gc_collector_get_stats(const echo_gc_collector_t* collector);

// 配置管理
echo_gc_error_t echo_gc_collector_update_config(
    echo_gc_collector_t* collector,
    const echo_gc_config_t* config
);

// 事件设置
void echo_gc_collector_set_event_publisher(
    echo_gc_collector_t* collector,
    void (*publisher)(const char* event_type, const void* event_data)
);

// 内部方法（供领域服务调用）
echo_gc_error_t echo_gc_collector_start_mark_phase(echo_gc_collector_t* collector);
echo_gc_error_t echo_gc_collector_complete_mark_phase(
    echo_gc_collector_t* collector,
    uint64_t marked_objects,
    uint64_t processed_bytes
);

echo_gc_error_t echo_gc_collector_start_sweep_phase(echo_gc_collector_t* collector);
echo_gc_error_t echo_gc_collector_complete_sweep_phase(
    echo_gc_collector_t* collector,
    uint64_t objects_collected,
    uint64_t memory_reclaimed
);

// 状态验证
bool echo_gc_collector_can_trigger_gc(const echo_gc_collector_t* collector);
bool echo_gc_collector_is_running(const echo_gc_collector_t* collector);

#endif // ECHO_GC_GARBAGE_COLLECTOR_H
