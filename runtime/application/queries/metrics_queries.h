#ifndef METRICS_QUERIES_H
#define METRICS_QUERIES_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>

// 前向声明
struct MetricFilter;
struct AggregationConfig;

// 查询性能指标
typedef struct {
    char metric_name[256];          // 指标名称
    struct MetricFilter* filter;    // 过滤条件
    struct AggregationConfig* aggregation; // 聚合配置
    time_t start_time;              // 开始时间
    time_t end_time;                // 结束时间
    uint32_t resolution_ms;         // 时间分辨率（毫秒）
} GetMetricsQuery;

// 查询任务指标
typedef struct {
    struct MetricFilter* filter;    // 过滤条件
    struct AggregationConfig* aggregation; // 聚合配置
    time_t time_window;             // 时间窗口
    bool include_histogram;         // 是否包含直方图
} GetTaskMetricsQuery;

// 查询协程指标
typedef struct {
    struct MetricFilter* filter;    // 过滤条件
    struct AggregationConfig* aggregation; // 聚合配置
    time_t time_window;             // 时间窗口
    bool include_stack_info;        // 是否包含栈信息
} GetCoroutineMetricsQuery;

// 查询内存指标
typedef struct {
    bool include_heap_stats;        // 是否包含堆统计
    bool include_gc_stats;          // 是否包含GC统计
    bool include_pool_stats;        // 是否包含内存池统计
    time_t time_window;             // 时间窗口
} GetMemoryMetricsQuery;

// 查询通道指标
typedef struct {
    struct MetricFilter* filter;    // 过滤条件
    struct AggregationConfig* aggregation; // 聚合配置
    time_t time_window;             // 时间窗口
    bool include_buffer_stats;      // 是否包含缓冲区统计
} GetChannelMetricsQuery;

// 查询异步操作指标
typedef struct {
    struct MetricFilter* filter;    // 过滤条件
    struct AggregationConfig* aggregation; // 聚合配置
    time_t time_window;             // 时间窗口
    bool include_future_stats;      // 是否包含Future统计
} GetAsyncMetricsQuery;

// 查询系统资源指标
typedef struct {
    bool include_cpu;               // 是否包含CPU使用率
    bool include_memory;            // 是否包含内存使用率
    bool include_disk;              // 是否包含磁盘使用率
    bool include_network;           // 是否包含网络使用率
    time_t time_window;             // 时间窗口
    uint32_t sampling_interval_ms;  // 采样间隔
} GetSystemMetricsQuery;

// 指标过滤器
struct MetricFilter {
    char* tags;                     // 标签过滤（key:value,key:value格式）
    char* metric_types;             // 指标类型过滤（逗号分隔）
    double min_value;               // 最小值过滤
    double max_value;               // 最大值过滤
    bool exclude_zeros;             // 是否排除零值
};

// 聚合配置
struct AggregationConfig {
    char aggregation_type[64];      // 聚合类型：sum, avg, min, max, count, percentile
    double percentile;              // 百分位数（当aggregation_type为percentile时使用）
    uint32_t bucket_count;          // 直方图桶数量
    bool include_raw_data;          // 是否包含原始数据
};

// 查询验证函数
bool validate_metrics_query(const GetMetricsQuery* query);
bool validate_task_metrics_query(const GetTaskMetricsQuery* query);
bool validate_coroutine_metrics_query(const GetCoroutineMetricsQuery* query);
bool validate_memory_metrics_query(const GetMemoryMetricsQuery* query);
bool validate_channel_metrics_query(const GetChannelMetricsQuery* query);
bool validate_async_metrics_query(const GetAsyncMetricsQuery* query);
bool validate_system_metrics_query(const GetSystemMetricsQuery* query);

// 查询构建函数
GetMetricsQuery* get_metrics_query_create(const char* metric_name, time_t start_time, time_t end_time);
void get_metrics_query_destroy(GetMetricsQuery* query);

GetTaskMetricsQuery* get_task_metrics_query_create(time_t time_window);
void get_task_metrics_query_destroy(GetTaskMetricsQuery* query);

struct MetricFilter* metric_filter_create(const char* tags, const char* metric_types);
void metric_filter_destroy(struct MetricFilter* filter);

struct AggregationConfig* aggregation_config_create(const char* type, double percentile, uint32_t bucket_count);
void aggregation_config_destroy(struct AggregationConfig* config);

#endif // METRICS_QUERIES_H
