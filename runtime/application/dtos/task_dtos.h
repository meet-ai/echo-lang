#ifndef TASK_DTOS_H
#define TASK_DTOS_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>

// 任务DTO - 数据传输对象
typedef struct {
    uint64_t id;                    // 任务ID
    char name[256];                 // 任务名称
    char description[1024];         // 任务描述
    char status[32];                // 任务状态
    char priority[16];              // 任务优先级
    time_t created_at;              // 创建时间
    time_t scheduled_at;            // 调度时间
    time_t started_at;              // 开始时间
    time_t completed_at;            // 完成时间
    int exit_code;                  // 退出代码
    char error_message[1024];       // 错误信息
    uint64_t execution_time_ms;     // 执行时间（毫秒）
} TaskDTO;

// 任务列表DTO
typedef struct {
    TaskDTO* tasks;                 // 任务数组
    size_t count;                   // 任务数量
    size_t total_count;             // 总任务数（用于分页）
    uint32_t page;                  // 当前页码
    uint32_t page_size;             // 每页大小
} TaskListDTO;

// 任务统计DTO
typedef struct {
    uint32_t total_tasks;           // 总任务数
    uint32_t running_tasks;         // 运行中任务数
    uint32_t completed_tasks;       // 已完成任务数
    uint32_t failed_tasks;          // 失败任务数
    uint32_t cancelled_tasks;       // 已取消任务数
    uint32_t queued_tasks;          // 队列中任务数
    double average_execution_time;  // 平均执行时间
    uint64_t total_execution_time;  // 总执行时间
    time_t last_updated;            // 最后更新时间
} TaskStatisticsDTO;

// 任务创建结果DTO
typedef struct {
    bool success;                   // 是否成功
    uint64_t task_id;               // 创建的任务ID
    char message[512];              // 结果消息
    time_t created_at;              // 创建时间
} TaskCreationResultDTO;

// 任务取消结果DTO
typedef struct {
    bool success;                   // 是否成功
    uint64_t task_id;               // 取消的任务ID
    char reason[512];               // 取消原因
    time_t cancelled_at;            // 取消时间
} TaskCancellationResultDTO;

// 任务执行结果DTO
typedef struct {
    uint64_t task_id;               // 任务ID
    bool success;                   // 执行是否成功
    int exit_code;                  // 退出代码
    void* result;                   // 执行结果
    size_t result_size;             // 结果大小
    char error_message[1024];       // 错误信息
    uint64_t execution_time_ms;     // 执行时间
    time_t completed_at;            // 完成时间
} TaskExecutionResultDTO;

// DTO映射函数
TaskDTO* task_to_dto(const struct Task* task);
TaskListDTO* tasks_to_list_dto(struct Task** tasks, size_t count, size_t total, uint32_t page, uint32_t page_size);
TaskStatisticsDTO* calculate_task_statistics(struct Task** tasks, size_t count);

// DTO销毁函数
void task_dto_destroy(TaskDTO* dto);
void task_list_dto_destroy(TaskListDTO* dto);
void task_statistics_dto_destroy(TaskStatisticsDTO* dto);
void task_creation_result_dto_destroy(TaskCreationResultDTO* dto);
void task_cancellation_result_dto_destroy(TaskCancellationResultDTO* dto);
void task_execution_result_dto_destroy(TaskExecutionResultDTO* dto);

// DTO序列化（用于API响应）
char* task_dto_to_json(const TaskDTO* dto);
char* task_list_dto_to_json(const TaskListDTO* dto);
char* task_statistics_dto_to_json(const TaskStatisticsDTO* dto);

// JSON反序列化（用于API请求）
TaskDTO* task_dto_from_json(const char* json);
TaskListDTO* task_list_dto_from_json(const char* json);

#endif // TASK_DTOS_H
