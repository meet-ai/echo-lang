#ifndef STATUS_DTOS_H
#define STATUS_DTOS_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>

// 运行时状态DTO
typedef struct {
    char version[32];               // 运行时版本
    char status[32];                // 运行时状态
    time_t started_at;              // 启动时间
    uint64_t uptime_seconds;        // 运行时长（秒）
    uint32_t active_tasks;          // 活跃任务数
    uint32_t active_coroutines;     // 活跃协程数
    uint32_t active_channels;       // 活跃通道数
    uint32_t active_futures;        // 活跃Future数
    uint64_t total_memory_used;     // 总内存使用
    uint64_t heap_size;             // 堆大小
    double cpu_usage_percentage;    // CPU使用率
    time_t last_updated;            // 最后更新时间
} RuntimeStatusDTO;

// 任务状态DTO
typedef struct {
    uint64_t task_id;               // 任务ID
    char name[256];                 // 任务名称
    char status[32];                // 任务状态
    char priority[16];              // 任务优先级
    time_t created_at;              // 创建时间
    time_t started_at;              // 开始时间
    uint64_t execution_time_ms;     // 执行时间
    double progress_percentage;     // 进度百分比
    char current_operation[256];    // 当前操作
} TaskStatusDTO;

// 协程状态DTO
typedef struct {
    uint64_t coroutine_id;          // 协程ID
    char name[256];                 // 协程名称
    char status[32];                // 协程状态
    uint64_t stack_size;            // 栈大小
    uint64_t stack_used;            // 栈使用量
    time_t created_at;              // 创建时间
    time_t last_yield_at;           // 最后yield时间
    uint32_t yield_count;           // yield次数
    char current_function[256];     // 当前函数
} CoroutineStatusDTO;

// 通道状态DTO
typedef struct {
    uint64_t channel_id;            // 通道ID
    char name[256];                 // 通道名称（如果有）
    char type[32];                  // 通道类型（buffered/unbuffered）
    size_t element_size;            // 元素大小
    size_t buffer_size;             // 缓冲区大小
    size_t current_length;          // 当前长度
    char status[32];                // 通道状态
    uint64_t send_count;            // 发送计数
    uint64_t receive_count;         // 接收计数
    time_t created_at;              // 创建时间
} ChannelStatusDTO;

// Future状态DTO
typedef struct {
    uint64_t future_id;             // Future ID
    char description[512];          // Future描述
    char status[32];                // Future状态
    time_t created_at;              // 创建时间
    time_t resolved_at;             // 解决时间
    bool has_result;                // 是否有结果
    size_t result_size;             // 结果大小
    char error_message[1024];       // 错误信息（如果有）
} FutureStatusDTO;

// GC状态DTO
typedef struct {
    char status[32];                // GC状态
    char algorithm[64];             // GC算法
    uint64_t heap_size;             // 堆大小
    uint64_t used_heap;             // 已用堆
    uint64_t free_heap;             // 空闲堆
    uint32_t collection_count;      // 回收次数
    uint64_t total_pause_time_ms;   // 总暂停时间
    uint32_t current_phase;         // 当前阶段
    time_t last_collection_at;      // 最后回收时间
    uint64_t collection_duration_ms; // 最近回收时长
} GCStatusDTO;

// 并发状态DTO
typedef struct {
    uint32_t active_threads;        // 活跃线程数
    uint32_t total_mutexes;         // 互斥锁总数
    uint32_t locked_mutexes;        // 已锁定互斥锁数
    uint32_t total_semaphores;      // 信号量总数
    uint32_t waiting_threads;       // 等待线程数
    uint64_t total_contention_events; // 总争用事件数
    double average_lock_wait_time;  // 平均锁等待时间
    time_t last_updated;            // 最后更新时间
} ConcurrencyStatusDTO;

// 内存状态DTO
typedef struct {
    uint64_t total_allocated;       // 总分配内存
    uint64_t currently_used;        // 当前使用内存
    uint64_t peak_usage;            // 峰值使用
    uint32_t allocation_count;      // 分配次数
    uint32_t deallocation_count;    // 释放次数
    uint32_t live_objects;          // 存活对象数
    double fragmentation_ratio;     // 碎片化比率
    time_t last_gc_at;              // 最后GC时间
} MemoryStatusDTO;

// 系统状态DTO
typedef struct {
    char hostname[256];             // 主机名
    char os_version[128];           // 操作系统版本
    uint32_t cpu_cores;             // CPU核心数
    uint64_t total_memory;          // 总内存
    uint64_t available_memory;      // 可用内存
    double cpu_usage;               // CPU使用率
    double memory_usage;            // 内存使用率
    uint32_t process_count;         // 进程数
    time_t boot_time;               // 启动时间
    time_t last_updated;            // 最后更新时间
} SystemStatusDTO;

// DTO构建函数
RuntimeStatusDTO* runtime_status_dto_create(void);
TaskStatusDTO* task_status_dto_create(uint64_t task_id, const char* name);
CoroutineStatusDTO* coroutine_status_dto_create(uint64_t coroutine_id, const char* name);
ChannelStatusDTO* channel_status_dto_create(uint64_t channel_id);
FutureStatusDTO* future_status_dto_create(uint64_t future_id);
GCStatusDTO* gc_status_dto_create(void);
ConcurrencyStatusDTO* concurrency_status_dto_create(void);
MemoryStatusDTO* memory_status_dto_create(void);
SystemStatusDTO* system_status_dto_create(void);

// DTO销毁函数
void runtime_status_dto_destroy(RuntimeStatusDTO* dto);
void task_status_dto_destroy(TaskStatusDTO* dto);
void coroutine_status_dto_destroy(CoroutineStatusDTO* dto);
void channel_status_dto_destroy(ChannelStatusDTO* dto);
void future_status_dto_destroy(FutureStatusDTO* dto);
void gc_status_dto_destroy(GCStatusDTO* dto);
void concurrency_status_dto_destroy(ConcurrencyStatusDTO* dto);
void memory_status_dto_destroy(MemoryStatusDTO* dto);
void system_status_dto_destroy(SystemStatusDTO* dto);

// DTO序列化
char* runtime_status_dto_to_json(const RuntimeStatusDTO* dto);
char* task_status_dto_to_json(const TaskStatusDTO* dto);
char* gc_status_dto_to_json(const GCStatusDTO* dto);

// JSON反序列化
RuntimeStatusDTO* runtime_status_dto_from_json(const char* json);
TaskStatusDTO* task_status_dto_from_json(const char* json);

#endif // STATUS_DTOS_H
