#ifndef DEBUG_QUERIES_H
#define DEBUG_QUERIES_H

#include <stdint.h>
#include <stdbool.h>

// 调试信息级别
typedef enum {
    DEBUG_LEVEL_BASIC,              // 基本信息
    DEBUG_LEVEL_DETAILED,           // 详细信息
    DEBUG_LEVEL_VERBOSE             // 详细调试信息
} DebugLevel;

// 查询调试信息
typedef struct {
    char component[256];            // 组件名称
    DebugLevel level;               // 调试级别
    bool include_timestamps;        // 是否包含时间戳
    bool include_thread_info;       // 是否包含线程信息
    uint32_t max_entries;           // 最大条目数
    char filter_pattern[512];       // 过滤模式（正则表达式）
} GetDebugInfoQuery;

// 查询调用栈
typedef struct {
    uint64_t target_id;             // 目标ID（任务ID、协程ID等）
    char target_type[64];           // 目标类型（task, coroutine, channel等）
    bool include_variables;         // 是否包含变量信息
    uint32_t max_depth;             // 最大栈深度
} GetCallStackQuery;

// 查询内存状态
typedef struct {
    bool include_heap_layout;       // 是否包含堆布局
    bool include_object_graph;      // 是否包含对象图
    bool include_allocation_sites;  // 是否包含分配站点
    uint32_t max_objects;           // 最大对象数量
    char object_filter[256];        // 对象过滤器
} GetMemoryDebugQuery;

// 查询并发状态
typedef struct {
    bool include_lock_graph;        // 是否包含锁图
    bool include_deadlock_detection; // 是否包含死锁检测
    bool include_contention_stats;  // 是否包含争用统计
    uint32_t max_locks;             // 最大锁数量
} GetConcurrencyDebugQuery;

// 查询事件日志
typedef struct {
    char event_type[128];           // 事件类型过滤
    uint64_t entity_id;             // 实体ID过滤
    time_t start_time;              // 开始时间
    time_t end_time;                // 结束时间
    uint32_t max_events;            // 最大事件数
    bool chronological;             // 是否按时间顺序
} GetEventLogQuery;

// 查询性能剖析
typedef struct {
    bool include_cpu_profile;       // 是否包含CPU剖析
    bool include_memory_profile;    // 是否包含内存剖析
    bool include_lock_profile;      // 是否包含锁剖析
    uint32_t duration_ms;           // 剖析持续时间
    uint32_t sampling_interval_ms;  // 采样间隔
} GetPerformanceProfileQuery;

// 查询系统转储
typedef struct {
    char dump_type[64];             // 转储类型（full, mini, selective）
    char output_path[512];          // 输出路径
    bool include_runtime_state;     // 是否包含运行时状态
    bool compress_output;           // 是否压缩输出
} GetSystemDumpQuery;

// 查询验证函数
bool validate_debug_info_query(const GetDebugInfoQuery* query);
bool validate_call_stack_query(const GetCallStackQuery* query);
bool validate_memory_debug_query(const GetMemoryDebugQuery* query);
bool validate_concurrency_debug_query(const GetConcurrencyDebugQuery* query);
bool validate_event_log_query(const GetEventLogQuery* query);
bool validate_performance_profile_query(const GetPerformanceProfileQuery* query);
bool validate_system_dump_query(const GetSystemDumpQuery* query);

// 查询构建函数
GetDebugInfoQuery* get_debug_info_query_create(const char* component, DebugLevel level);
void get_debug_info_query_destroy(GetDebugInfoQuery* query);

GetCallStackQuery* get_call_stack_query_create(uint64_t target_id, const char* target_type);
void get_call_stack_query_destroy(GetCallStackQuery* query);

GetMemoryDebugQuery* get_memory_debug_query_create(bool include_heap);
void get_memory_debug_query_destroy(GetMemoryDebugQuery* query);

#endif // DEBUG_QUERIES_H
