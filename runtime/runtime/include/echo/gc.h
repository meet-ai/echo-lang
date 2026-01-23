#ifndef ECHO_GC_H
#define ECHO_GC_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>

// 前向声明
struct echo_gc_heap;
struct echo_gc_collector;
struct echo_gc_config;

// GC状态枚举
typedef enum {
    ECHO_GC_STATUS_IDLE = 0,
    ECHO_GC_STATUS_RUNNING = 1,
    ECHO_GC_STATUS_PAUSED = 2,
    ECHO_GC_STATUS_ERROR = 3
} echo_gc_status_t;

// GC阶段枚举
typedef enum {
    ECHO_GC_PHASE_IDLE = 0,
    ECHO_GC_PHASE_MARK = 1,
    ECHO_GC_PHASE_SWEEP = 2,
    ECHO_GC_PHASE_COMPLETE = 3
} echo_gc_phase_t;

// GC触发原因
typedef enum {
    ECHO_GC_TRIGGER_HEAP_USAGE = 0,
    ECHO_GC_TRIGGER_ALLOCATION_RATE = 1,
    ECHO_GC_TRIGGER_MANUAL = 2,
    ECHO_GC_TRIGGER_EMERGENCY = 3
} echo_gc_trigger_reason_t;

// 三色标记状态
typedef enum {
    ECHO_GC_COLOR_WHITE = 0,  // 未标记
    ECHO_GC_COLOR_GRAY = 1,   // 已标记，但子对象未标记
    ECHO_GC_COLOR_BLACK = 2   // 已标记，且子对象已标记
} echo_gc_color_t;

// 对象类型
typedef enum {
    ECHO_OBJ_TYPE_PRIMITIVE = 0,
    ECHO_OBJ_TYPE_ARRAY = 1,
    ECHO_OBJ_TYPE_STRUCT = 2,
    ECHO_OBJ_TYPE_INTERFACE = 3,
    ECHO_OBJ_TYPE_CHANNEL = 4,
    ECHO_OBJ_TYPE_SLICE = 5,
    ECHO_OBJ_TYPE_MAP = 6,
    ECHO_OBJ_TYPE_STRING = 7,
    ECHO_OBJ_TYPE_COROUTINE = 8,
    ECHO_OBJ_TYPE_FUTURE = 9
} echo_obj_type_t;

// GC配置
typedef struct echo_gc_config {
    // 触发配置
    double trigger_ratio;              // GC触发阈值 (0.0-1.0)
    uint64_t min_heap_size;            // 最小堆大小
    uint64_t max_heap_size;            // 最大堆大小

    // 性能目标
    uint64_t target_pause_time_us;     // 目标暂停时间 (微秒)
    uint64_t max_pause_time_us;        // 最大暂停时间 (微秒)

    // 并发配置
    int max_concurrent_workers;        // 最大并发工作线程数

    // 调试配置
    bool enable_tracing;               // 启用追踪
    bool enable_profiling;             // 启用性能剖析
} echo_gc_config_t;

// 性能指标
typedef struct echo_gc_metrics {
    // 时间指标
    uint64_t pause_time_p50_us;        // P50暂停时间 (微秒)
    uint64_t pause_time_p95_us;        // P95暂停时间 (微秒)
    uint64_t pause_time_p99_us;        // P99暂停时间 (微秒)
    uint64_t total_pause_time_us;      // 总暂停时间 (微秒)
    uint64_t gc_duration_us;           // GC总持续时间 (微秒)

    // 内存指标
    double heap_usage_ratio;           // 堆使用率
    uint64_t memory_reclaimed_bytes;   // 回收内存量
    uint64_t allocation_rate_bps;      // 分配速率 (bytes/sec)
    double fragmentation_ratio;        // 碎片率

    // 对象统计
    uint64_t total_objects;            // 总对象数
    uint64_t live_objects;             // 存活对象数
    uint64_t marked_objects;           // 已标记对象数

    // CPU指标
    double gc_cpu_percentage;          // GC CPU使用率

    // 时间戳
    struct timespec collected_at;      // 收集时间
} echo_gc_metrics_t;

// GC统计信息
typedef struct echo_gc_stats {
    // 基本统计
    uint64_t total_cycles;             // 总GC周期数
    uint64_t successful_cycles;        // 成功GC周期数
    uint64_t failed_cycles;            // 失败GC周期数

    // 时间统计
    uint64_t total_gc_time_us;         // 总GC时间
    uint64_t total_pause_time_us;      // 总暂停时间
    uint64_t average_gc_duration_us;   // 平均GC持续时间

    // 内存统计
    uint64_t total_memory_allocated;   // 总分配内存
    uint64_t total_memory_reclaimed;   // 总回收内存

    // 时间戳
    struct timespec last_completed_at; // 最后完成时间
} echo_gc_stats_t;

// 对象头（紧凑8字节）
typedef struct echo_gc_object_header {
    union {
        struct {
            uint32_t size : 28;        // 对象大小（包括头部）
            uint32_t color : 2;        // 三色标记状态
            uint32_t forwarded : 1;    // 转发指针标志
            uint32_t pinned : 1;       // 固定对象（不移动）
        };
        uint32_t flags;
    };
    union {
        void* forward;                 // 转发指针
        uint32_t hash;                 // 类型哈希
    };
} echo_gc_object_header_t;

// GC对象
typedef struct echo_gc_object {
    echo_gc_object_header_t header;
    char data[];                      // 可变长度数据
} echo_gc_object_t;

// 堆接口
typedef struct echo_gc_heap echo_gc_heap_t;

// 垃圾回收器接口
typedef struct echo_gc_collector echo_gc_collector_t;

// 错误码
typedef enum {
    ECHO_GC_SUCCESS = 0,
    ECHO_GC_ERROR_INVALID_CONFIG = 1,
    ECHO_GC_ERROR_OUT_OF_MEMORY = 2,
    ECHO_GC_ERROR_INVALID_OBJECT = 3,
    ECHO_GC_ERROR_CONCURRENT_MODIFICATION = 4,
    ECHO_GC_ERROR_MARK_PHASE_FAILED = 5,
    ECHO_GC_ERROR_SWEEP_PHASE_FAILED = 6,
    ECHO_GC_ERROR_SYSTEM_ERROR = 7
} echo_gc_error_t;

// 核心API
#ifdef __cplusplus
extern "C" {
#endif

// 初始化和清理
echo_gc_error_t echo_gc_init(const echo_gc_config_t* config);
void echo_gc_shutdown(void);

// 配置管理
echo_gc_error_t echo_gc_get_config(echo_gc_config_t* config);
echo_gc_error_t echo_gc_update_config(const echo_gc_config_t* config);

// 堆管理
echo_gc_heap_t echo_gc_get_heap(void);
echo_gc_error_t echo_gc_allocate(size_t size, echo_obj_type_t type, void** ptr);
echo_gc_error_t echo_gc_collect(void);

// 垃圾回收器控制
echo_gc_collector_t echo_gc_get_collector(void);
echo_gc_status_t echo_gc_get_status(void);
echo_gc_phase_t echo_gc_get_phase(void);

// 统计和监控
echo_gc_error_t echo_gc_get_stats(echo_gc_stats_t* stats);
echo_gc_error_t echo_gc_get_metrics(echo_gc_metrics_t* metrics);

// 写屏障（编译器插入）
void echo_gc_write_barrier(void** slot, void* new_value);

// 调试接口
void echo_gc_dump_heap(void);
void echo_gc_dump_stats(void);
bool echo_gc_validate_heap(void);

// 事件回调注册
typedef void (*echo_gc_event_callback_t)(const char* event_type, const void* event_data);
echo_gc_error_t echo_gc_register_event_callback(const char* event_type, echo_gc_event_callback_t callback);

#ifdef __cplusplus
}
#endif

#endif // ECHO_GC_H
