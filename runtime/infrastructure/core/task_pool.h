#ifndef TASK_POOL_H
#define TASK_POOL_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>

// 前向声明
struct TaskPool;
struct TaskPoolConfig;
struct PooledTask;
struct TaskPoolStats;
struct TaskPoolWorker;

// 任务池配置
typedef struct TaskPoolConfig {
    uint32_t min_workers;        // 最小工作线程数
    uint32_t max_workers;        // 最大工作线程数
    uint32_t initial_workers;    // 初始工作线程数
    uint32_t task_queue_size;    // 任务队列大小
    uint32_t worker_idle_timeout_seconds; // 工作线程空闲超时
    uint32_t task_timeout_seconds; // 任务执行超时
    bool auto_scaling_enabled;   // 是否启用自动扩缩容
    uint32_t scale_up_threshold; // 扩容阈值（队列长度）
    uint32_t scale_down_threshold; // 缩容阈值（空闲线程数）
    char scheduling_policy[32];  // 调度策略 ("fifo", "priority", "fair")
    bool enable_task_stats;      // 是否启用任务统计
    bool enable_worker_stats;    // 是否启用工作线程统计
} TaskPoolConfig;

// 池化任务
typedef struct PooledTask {
    uint64_t task_id;            // 任务ID
    char task_name[256];         // 任务名称
    void* task_function;         // 任务函数
    void* task_arg;              // 任务参数
    uint32_t priority;           // 任务优先级
    time_t submitted_at;         // 提交时间
    time_t started_at;           // 开始时间
    time_t completed_at;         // 完成时间
    uint32_t execution_time_ms;  // 执行时间
    char status[32];             // 任务状态
    int exit_code;               // 退出代码
    char error_message[1024];    // 错误信息
    void* result_data;           // 结果数据
    size_t result_size;          // 结果大小
    void* user_data;             // 用户数据
} PooledTask;

// 工作线程信息
typedef struct TaskPoolWorker {
    uint64_t worker_id;          // 工作线程ID
    char worker_name[128];       // 工作线程名称
    char status[32];             // 工作线程状态
    time_t created_at;           // 创建时间
    time_t last_active_at;       // 最后活跃时间
    uint64_t total_tasks_executed; // 总执行任务数
    uint64_t current_task_id;    // 当前任务ID
    uint64_t execution_time_ms;  // 总执行时间
    double cpu_usage;            // CPU使用率
    uint64_t memory_usage;       // 内存使用量
} TaskPoolWorker;

// 任务池统计
typedef struct TaskPoolStats {
    // 整体统计
    uint64_t total_tasks_submitted;  // 总提交任务数
    uint64_t total_tasks_completed;  // 总完成任务数
    uint64_t total_tasks_failed;     // 总失败任务数
    uint64_t total_tasks_cancelled;  // 总取消任务数

    // 当前状态
    uint32_t active_workers;         // 活跃工作线程数
    uint32_t idle_workers;           // 空闲工作线程数
    uint32_t total_workers;          // 总工作线程数
    uint32_t queued_tasks;           // 队列中任务数
    uint32_t running_tasks;          // 运行中任务数

    // 性能指标
    double average_task_time_ms;     // 平均任务执行时间
    double average_queue_time_ms;    // 平均队列等待时间
    uint32_t max_concurrent_tasks;   // 最大并发任务数
    uint32_t worker_utilization_percent; // 工作线程利用率

    // 错误统计
    uint32_t timeout_errors;         // 超时错误数
    uint32_t execution_errors;       // 执行错误数
    uint32_t queue_full_errors;      // 队列满错误数

    // 时间统计
    time_t pool_created_at;          // 池创建时间
    time_t last_task_completed_at;   // 最后任务完成时间
    uint64_t total_uptime_seconds;   // 总运行时间
} TaskPoolStats;

// 任务池接口
typedef struct {
    // 任务提交
    bool (*submit_task)(struct TaskPool* pool, struct PooledTask* task);
    bool (*submit_task_function)(struct TaskPool* pool, const char* task_name,
                                void* task_function, void* task_arg, uint32_t priority);
    bool (*submit_batch_tasks)(struct TaskPool* pool, struct PooledTask* tasks, size_t count);

    // 任务管理
    bool (*cancel_task)(struct TaskPool* pool, uint64_t task_id);
    bool (*get_task_status)(struct TaskPool* pool, uint64_t task_id, struct PooledTask* task);
    bool (*wait_for_task)(struct TaskPool* pool, uint64_t task_id, uint32_t timeout_ms);
    bool (*wait_for_all_tasks)(struct TaskPool* pool, uint32_t timeout_ms);

    // 工作线程管理
    bool (*get_worker_info)(struct TaskPool* pool, uint64_t worker_id, struct TaskPoolWorker* worker);
    bool (*get_all_workers)(struct TaskPool* pool, struct TaskPoolWorker** workers, size_t* count);
    bool (*set_worker_count)(struct TaskPool* pool, uint32_t count);

    // 池管理
    bool (*pause_pool)(struct TaskPool* pool);
    bool (*resume_pool)(struct TaskPool* pool);
    bool (*shutdown_pool)(struct TaskPool* pool, uint32_t timeout_ms);

    // 统计和监控
    bool (*get_stats)(const struct TaskPool* pool, struct TaskPoolStats* stats);
    bool (*reset_stats)(struct TaskPool* pool);

    // 配置管理
    bool (*reconfigure)(struct TaskPool* pool, const struct TaskPoolConfig* new_config);
    bool (*get_config)(const struct TaskPool* pool, struct TaskPoolConfig* config);

    // 健康检查
    bool (*health_check)(const struct TaskPool* pool);
    bool (*diagnostics)(const struct TaskPool* pool, char* report, size_t report_size);
} TaskPoolInterface;

// 任务池实现
typedef struct TaskPool {
    TaskPoolInterface* vtable;    // 虚函数表

    // 配置
    struct TaskPoolConfig config;

    // 状态
    bool initialized;
    bool paused;
    bool shutting_down;
    time_t created_at;
    char pool_name[128];

    // 内部数据结构
    void* task_queue;             // 任务队列
    void* worker_threads;         // 工作线程数组
    void* worker_pool;            // 工作线程池
    void* task_map;               // 任务映射表

    // 同步原语
    void* queue_mutex;
    void* workers_mutex;
    void* stats_mutex;
    void* condition;              // 条件变量

    // 统计
    struct TaskPoolStats stats;

    // 监控
    bool monitoring_enabled;
    void* metrics_collector;
} TaskPool;

// 任务池工厂函数
TaskPool* task_pool_create(const struct TaskPoolConfig* config);
void task_pool_destroy(TaskPool* pool);

// 便捷构造函数
struct TaskPoolConfig* task_pool_config_create(void);
void task_pool_config_destroy(struct TaskPoolConfig* config);

// 预定义配置
struct TaskPoolConfig* task_pool_config_create_default(void);
struct TaskPoolConfig* task_pool_config_create_high_throughput(void);
struct TaskPoolConfig* task_pool_config_create_low_latency(void);
struct TaskPoolConfig* task_pool_config_create_cpu_bound(void);

// 任务构造函数
struct PooledTask* pooled_task_create(const char* task_name, void* task_function, void* task_arg, uint32_t priority);
void pooled_task_destroy(struct PooledTask* task);

// 便捷函数（直接调用虚函数表）
static inline bool task_pool_submit_task_function(TaskPool* pool, const char* task_name,
                                                 void* task_function, void* task_arg, uint32_t priority) {
    return pool->vtable->submit_task_function(pool, task_name, task_function, task_arg, priority);
}

static inline bool task_pool_get_stats(const TaskPool* pool, struct TaskPoolStats* stats) {
    return pool->vtable->get_stats(pool, stats);
}

static inline bool task_pool_shutdown(TaskPool* pool, uint32_t timeout_ms) {
    return pool->vtable->shutdown_pool(pool, timeout_ms);
}

// 全局任务池实例（可选）
extern TaskPool* g_global_task_pool;

// 全局便捷函数
static inline bool task_pool_submit_global(const char* task_name, void* task_function, void* task_arg, uint32_t priority) {
    if (g_global_task_pool) {
        return task_pool_submit_task_function(g_global_task_pool, task_name, task_function, task_arg, priority);
    }
    return false;
}

#endif // TASK_POOL_H
