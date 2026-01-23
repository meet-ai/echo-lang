#ifndef RUNTIME_CONFIG_H
#define RUNTIME_CONFIG_H

#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct RuntimeConfig;
struct TaskSchedulerConfig;
struct CoroutineConfig;
struct MemoryConfig;
struct GCConfig;
struct AsyncConfig;
struct ChannelConfig;
struct ConcurrencyConfig;

// 任务调度器配置
typedef struct TaskSchedulerConfig {
    char scheduler_type[64];     // 调度器类型 ("fcfs", "priority", "fair")
    uint32_t max_concurrent_tasks; // 最大并发任务数
    uint32_t task_queue_size;    // 任务队列大小
    uint32_t worker_threads;     // 工作线程数
    uint32_t priority_levels;    // 优先级级别数
    bool preemption_enabled;     // 是否启用抢占
    uint32_t time_slice_ms;      // 时间片长度（毫秒）
    uint32_t starvation_threshold_ms; // 饥饿阈值
} TaskSchedulerConfig;

// 协程配置
typedef struct CoroutineConfig {
    size_t default_stack_size;   // 默认栈大小
    size_t max_stack_size;       // 最大栈大小
    uint32_t max_coroutines;     // 最大协程数量
    bool share_stack_enabled;    // 是否启用共享栈
    uint32_t max_shared_stack_size; // 最大共享栈大小
    uint32_t yield_threshold;    // 切换阈值
    bool auto_cleanup;           // 自动清理
} CoroutineConfig;

// 内存配置
typedef struct MemoryConfig {
    size_t heap_initial_size;    // 初始堆大小
    size_t heap_max_size;        // 最大堆大小
    size_t page_size;            // 页面大小
    size_t allocation_alignment; // 分配对齐
    bool memory_pool_enabled;    // 是否启用内存池
    size_t pool_block_size;      // 内存池块大小
    uint32_t pool_initial_blocks; // 初始内存块数量
    uint32_t pool_max_blocks;    // 最大内存块数量
    bool garbage_collection_enabled; // 是否启用GC
} MemoryConfig;

// GC配置
typedef struct GCConfig {
    char gc_algorithm[64];       // GC算法 ("mark_sweep", "generational", "concurrent")
    uint32_t gc_interval_ms;     // GC间隔
    double heap_growth_factor;   // 堆增长因子
    uint32_t young_generation_ratio; // 年轻代比例
    uint32_t old_generation_ratio; // 老年代比例
    bool concurrent_marking;     // 并发标记
    bool incremental_collection; // 增量回收
    uint32_t max_pause_time_ms;  // 最大暂停时间
    bool write_barrier_enabled;  // 写屏障
} GCConfig;

// 异步配置
typedef struct AsyncConfig {
    uint32_t max_pending_futures; // 最大待处理Future数量
    uint32_t future_timeout_ms;   // Future超时时间
    uint32_t max_nested_awaits;   // 最大嵌套await层数
    bool async_stack_traces;     // 异步栈跟踪
    uint32_t promise_pool_size;  // Promise池大小
    uint32_t combinator_buffer_size; // 组合器缓冲区大小
} AsyncConfig;

// 通道配置
typedef struct ChannelConfig {
    size_t default_buffer_size;  // 默认缓冲区大小
    uint32_t max_channels;       // 最大通道数量
    uint32_t channel_timeout_ms; // 通道超时时间
    bool select_optimization;    // 选择优化
    uint32_t max_select_cases;   // 最大选择分支数
    bool buffered_channel_pool;  // 缓冲通道池
    uint32_t channel_pool_size;  // 通道池大小
} ChannelConfig;

// 并发配置
typedef struct ConcurrencyConfig {
    uint32_t max_mutexes;        // 最大互斥锁数量
    uint32_t max_semaphores;     // 最大信号量数量
    uint32_t max_condition_variables; // 最大条件变量数量
    uint32_t spin_lock_iterations; // 自旋锁迭代次数
    bool deadlock_detection;     // 死锁检测
    uint32_t deadlock_check_interval_ms; // 死锁检查间隔
    bool lock_profiling;         // 锁分析
} ConcurrencyConfig;

// 运行时配置
typedef struct RuntimeConfig {
    char config_version[32];     // 配置版本
    char runtime_name[128];      // 运行时名称
    char log_level[16];          // 日志级别
    char log_file[512];          // 日志文件路径

    // 子配置
    struct TaskSchedulerConfig task_scheduler;
    struct CoroutineConfig coroutine;
    struct MemoryConfig memory;
    struct GCConfig gc;
    struct AsyncConfig async;
    struct ChannelConfig channel;
    struct ConcurrencyConfig concurrency;

    // 扩展配置（JSON格式）
    char* extensions_config;     // 扩展配置
    size_t extensions_config_size;

    // 平台特定配置
    char platform_config[4096];  // 平台配置（JSON格式）
} RuntimeConfig;

// 配置管理函数
RuntimeConfig* runtime_config_create(void);
void runtime_config_destroy(RuntimeConfig* config);

// 配置加载
bool runtime_config_load_from_file(RuntimeConfig* config, const char* file_path);
bool runtime_config_load_from_json(RuntimeConfig* config, const char* json_config);
bool runtime_config_load_from_env(RuntimeConfig* config);

// 配置验证
bool runtime_config_validate(const RuntimeConfig* config, char* error_message, size_t error_size);

// 配置保存
bool runtime_config_save_to_file(const RuntimeConfig* config, const char* file_path);
char* runtime_config_to_json(const RuntimeConfig* config);

// 配置合并（用于配置覆盖）
bool runtime_config_merge(RuntimeConfig* base_config, const RuntimeConfig* override_config);

// 配置监听（配置热更新）
typedef void (*ConfigChangeCallback)(const char* key, const char* old_value, const char* new_value, void* user_data);

bool runtime_config_watch(RuntimeConfig* config,
                         ConfigChangeCallback callback,
                         void* user_data,
                         void** watcher_handle);
bool runtime_config_unwatch(RuntimeConfig* config, void* watcher_handle);

// 默认配置生成
RuntimeConfig* runtime_config_create_default(void);

// 配置比较（用于测试和调试）
bool runtime_config_equals(const RuntimeConfig* config1, const RuntimeConfig* config2);
char* runtime_config_diff(const RuntimeConfig* config1, const RuntimeConfig* config2);

// 配置性能统计
typedef struct ConfigStats {
    uint64_t load_count;         // 加载次数
    uint64_t save_count;         // 保存次数
    uint64_t validation_count;   // 验证次数
    uint64_t watch_count;        // 监听次数
    uint64_t change_count;       // 配置变更次数
    uint64_t total_load_time_ns; // 总加载时间
    uint64_t total_save_time_ns; // 总保存时间
} ConfigStats;

bool runtime_config_get_stats(const RuntimeConfig* config, ConfigStats* stats);

// 便捷访问函数
static inline const TaskSchedulerConfig* runtime_config_get_task_scheduler(const RuntimeConfig* config) {
    return &config->task_scheduler;
}

static inline const CoroutineConfig* runtime_config_get_coroutine(const RuntimeConfig* config) {
    return &config->coroutine;
}

static inline const MemoryConfig* runtime_config_get_memory(const RuntimeConfig* config) {
    return &config->memory;
}

#endif // RUNTIME_CONFIG_H
