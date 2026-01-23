#ifndef TASK_H
#define TASK_H

#include <stdint.h>
#include <time.h>
#include <stdbool.h>

// 任务状态枚举
typedef enum {
    TASK_STATUS_CREATED,      // 已创建
    TASK_STATUS_READY,        // 就绪
    TASK_STATUS_RUNNING,      // 运行中
    TASK_STATUS_SUSPENDED,    // 暂停
    TASK_STATUS_COMPLETED,    // 已完成
    TASK_STATUS_FAILED,       // 失败
    TASK_STATUS_CANCELLED     // 已取消
} TaskStatus;

// 任务优先级枚举
typedef enum {
    TASK_PRIORITY_LOW,
    TASK_PRIORITY_NORMAL,
    TASK_PRIORITY_HIGH,
    TASK_PRIORITY_CRITICAL
} TaskPriority;

// 任务统计信息
typedef struct {
    uint64_t execution_time_ms;     // 执行时间
    uint64_t wait_time_ms;          // 等待时间
    uint64_t total_time_ms;         // 总时间
    uint32_t context_switches;      // 上下文切换次数
    uint32_t priority_changes;      // 优先级变更次数
} TaskStats;

// 任务实体 - 代表一个可执行的任务
typedef struct Task {
    uint64_t id;                    // 任务唯一标识
    char name[256];                 // 任务名称
    char description[1024];         // 任务描述
    TaskStatus status;       // 任务状态
    TaskPriority priority;   // 任务优先级
    void* entry_point;              // 任务入口函数指针
    void* arg;                      // 任务参数
    size_t stack_size;              // 任务栈大小
    time_t created_at;              // 创建时间
    time_t scheduled_at;            // 调度时间
    time_t started_at;              // 开始执行时间
    time_t completed_at;            // 完成时间
    int exit_code;                  // 退出代码
    char error_message[1024];       // 错误信息
    TaskStats stats;                // 统计信息
    void* user_data;                // 用户数据
    bool is_cancelled;              // 是否已取消
} Task;

// 任务方法
Task* task_create(const char* name, void* entry_point, void* arg, size_t stack_size);
void task_destroy(Task* task);
bool task_can_schedule(const Task* task);
bool task_can_cancel(const Task* task);
bool task_is_completed(const Task* task);
bool task_is_running(const Task* task);
bool task_can_restart(const Task* task);
void task_update_status(Task* task, TaskStatus new_status);
bool task_set_description(Task* task, const char* description);
bool task_set_priority(Task* task, TaskPriority priority);
bool task_set_exit_code(Task* task, int exit_code);
bool task_set_error_message(Task* task, const char* error_message);
bool task_reset(Task* task);
Task* task_clone(const Task* original);
uint64_t task_get_execution_time_us(const Task* task);
uint64_t task_get_total_time_us(const Task* task);
bool task_get_statistics(const Task* task, TaskStats* stats);
bool task_validate_state(const Task* task);

// 字符串转换函数
const char* task_status_string(TaskStatus status);
const char* task_priority_string(TaskPriority priority);

// 比较函数（用于排序）
int task_compare_by_priority(const Task* a, const Task* b);
int task_compare_by_creation_time(const Task* a, const Task* b);

#endif // TASK_H
