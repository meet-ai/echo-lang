#ifndef STATUS_QUERIES_H
#define STATUS_QUERIES_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>

// 前向声明
struct StatusFilter;
struct TimeRange;
struct Pagination;

// 查询运行时状态
typedef struct {
    bool include_tasks;             // 是否包含任务状态
    bool include_coroutines;        // 是否包含协程状态
    bool include_channels;          // 是否包含通道状态
    bool include_memory;            // 是否包含内存状态
    bool detailed;                  // 是否返回详细信息
} GetRuntimeStatusQuery;

// 查询任务状态
typedef struct {
    struct StatusFilter* filter;    // 状态过滤器
    struct TimeRange* time_range;   // 时间范围
    struct Pagination* pagination;  // 分页信息
    bool include_history;           // 是否包含历史记录
} GetTaskStatusQuery;

// 查询协程状态
typedef struct {
    struct StatusFilter* filter;    // 状态过滤器
    struct TimeRange* time_range;   // 时间范围
    struct Pagination* pagination;  // 分页信息
    bool include_stack_trace;       // 是否包含栈跟踪
} GetCoroutineStatusQuery;

// 查询通道状态
typedef struct {
    struct StatusFilter* filter;    // 状态过滤器
    struct TimeRange* time_range;   // 时间范围
    struct Pagination* pagination;  // 分页信息
    bool include_buffer_info;       // 是否包含缓冲区信息
} GetChannelStatusQuery;

// 查询Future状态
typedef struct {
    struct StatusFilter* filter;    // 状态过滤器
    struct TimeRange* time_range;   // 时间范围
    struct Pagination* pagination;  // 分页信息
    bool include_dependencies;      // 是否包含依赖关系
} GetFutureStatusQuery;

// 查询GC状态
typedef struct {
    bool include_heap_stats;        // 是否包含堆统计
    bool include_collection_stats;  // 是否包含回收统计
    bool include_generation_stats;  // 是否包含分代统计
    time_t since_time;              // 从什么时间开始统计
} GetGCStatusQuery;

// 查询并发状态
typedef struct {
    bool include_mutexes;           // 是否包含互斥锁状态
    bool include_semaphores;        // 是否包含信号量状态
    bool include_atomics;           // 是否包含原子操作统计
    bool include_contention_stats;  // 是否包含争用统计
} GetConcurrencyStatusQuery;

// 状态过滤器
struct StatusFilter {
    char* status_values;            // 状态值列表（逗号分隔）
    size_t status_count;            // 状态数量
    bool exclude_completed;         // 是否排除已完成的项目
    bool only_active;               // 是否只包含活跃项目
};

// 时间范围
struct TimeRange {
    time_t start_time;              // 开始时间
    time_t end_time;                // 结束时间
    bool relative;                  // 是否为相对时间（相对于当前时间）
};

// 分页信息
struct Pagination {
    uint32_t page;                  // 页码（从1开始）
    uint32_t page_size;             // 每页大小
    char* sort_by;                  // 排序字段
    bool ascending;                 // 是否升序
};

// 查询验证函数
bool validate_runtime_status_query(const GetRuntimeStatusQuery* query);
bool validate_task_status_query(const GetTaskStatusQuery* query);
bool validate_coroutine_status_query(const GetCoroutineStatusQuery* query);
bool validate_channel_status_query(const GetChannelStatusQuery* query);
bool validate_future_status_query(const GetFutureStatusQuery* query);
bool validate_gc_status_query(const GetGCStatusQuery* query);
bool validate_concurrency_status_query(const GetConcurrencyStatusQuery* query);

// 查询构建函数
GetRuntimeStatusQuery* get_runtime_status_query_create(bool detailed);
void get_runtime_status_query_destroy(GetRuntimeStatusQuery* query);

GetTaskStatusQuery* get_task_status_query_create(struct StatusFilter* filter, struct Pagination* pagination);
void get_task_status_query_destroy(GetTaskStatusQuery* query);

struct StatusFilter* status_filter_create(const char* status_values);
void status_filter_destroy(struct StatusFilter* filter);

struct TimeRange* time_range_create(time_t start, time_t end, bool relative);
void time_range_destroy(struct TimeRange* range);

struct Pagination* pagination_create(uint32_t page, uint32_t page_size, const char* sort_by, bool ascending);
void pagination_destroy(struct Pagination* pagination);

#endif // STATUS_QUERIES_H
