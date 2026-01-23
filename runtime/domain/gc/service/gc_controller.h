#ifndef GC_CONTROLLER_H
#define GC_CONTROLLER_H

#include <stdint.h>
#include <stdbool.h>
#include <pthread.h>

// 前向声明
struct GCController;
struct GCStats;
struct GCConfig;

// GC控制器领域服务 - 管理垃圾回收过程
typedef struct GCController {
    struct GCConfig* config;         // GC配置
    struct GCStats* stats;           // GC统计信息
    bool is_running;                 // GC是否正在运行
    pthread_t gc_thread;             // GC线程
    pthread_mutex_t mutex;           // 同步锁
    pthread_cond_t cond;             // 条件变量

    // 回调函数
    void (*on_gc_start)(void* context);
    void (*on_gc_end)(void* context);
    void* callback_context;
} GCController;

// GC算法类型
typedef enum {
    GC_ALGORITHM_MARK_SWEEP,         // 标记-清扫
    GC_ALGORITHM_MARK_COMPACT,       // 标记-整理
    GC_ALGORITHM_COPYING,            // 复制
    GC_ALGORITHM_GENERATIONAL        // 分代
} GCAlgorithm;

// GC触发策略
typedef enum {
    GC_TRIGGER_MANUAL,               // 手动触发
    GC_TRIGGER_ALLOCATION_THRESHOLD, // 分配阈值
    GC_TRIGGER_TIME_BASED,           // 定时触发
    GC_TRIGGER_ADAPTIVE              // 自适应
} GCTriggerStrategy;

// GC配置
typedef struct GCConfig {
    GCAlgorithm algorithm;           // GC算法
    GCTriggerStrategy trigger;       // 触发策略
    size_t heap_size;                // 堆大小
    size_t young_gen_size;           // 年轻代大小（分代GC）
    size_t old_gen_size;             // 老年代大小（分代GC）
    double gc_threshold;             // GC阈值（百分比）
    uint32_t gc_interval_ms;         // GC间隔（毫秒）
    bool concurrent_marking;         // 是否并发标记
    bool parallel_compaction;        // 是否并行整理
} GCConfig;

// GC统计信息
typedef struct GCStats {
    uint64_t total_collections;       // 总GC次数
    uint64_t total_pause_time_ms;     // 总暂停时间
    uint64_t max_pause_time_ms;       // 最大暂停时间
    uint64_t total_allocated_bytes;   // 总分配字节数
    uint64_t total_freed_bytes;       // 总释放字节数
    uint64_t current_heap_size;       // 当前堆大小
    uint64_t peak_heap_size;          // 峰值堆大小
    time_t last_gc_time;              // 最后GC时间
    double average_gc_time_ms;        // 平均GC时间
} GCStats;

// 默认配置
extern const GCConfig GC_DEFAULT_CONFIG;

// GC控制器方法
GCController* gc_controller_create(const GCConfig* config);
void gc_controller_destroy(GCController* controller);

// GC生命周期
bool gc_controller_start(GCController* controller);
bool gc_controller_stop(GCController* controller);
bool gc_controller_trigger_gc(GCController* controller);

// 配置管理
bool gc_controller_update_config(GCController* controller, const GCConfig* new_config);
const GCConfig* gc_controller_get_config(const GCController* controller);

// 统计信息
const GCStats* gc_controller_get_stats(const GCController* controller);
void gc_controller_reset_stats(GCController* controller);

// 回调设置
void gc_controller_set_callbacks(GCController* controller,
                                void (*on_start)(void*),
                                void (*on_end)(void*),
                                void* context);

// 状态查询
bool gc_controller_is_running(const GCController* controller);
GCAlgorithm gc_controller_get_algorithm(const GCController* controller);

#endif // GC_CONTROLLER_H
