#ifndef TASK_COMMANDS_H
#define TASK_COMMANDS_H

#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct TaskOptions;

// 创建任务命令
typedef struct {
    char name[256];                 // 任务名称
    char description[1024];         // 任务描述
    void* entry_point;              // 任务入口函数
    void* arg;                      // 任务参数
    size_t stack_size;              // 栈大小
    struct TaskOptions* options;    // 任务选项
} CreateTaskCommand;

// 取消任务命令
typedef struct {
    uint64_t task_id;               // 任务ID
    char reason[512];               // 取消原因
} CancelTaskCommand;

// 暂停任务命令
typedef struct {
    uint64_t task_id;               // 任务ID
    bool force;                     // 是否强制暂停
} PauseTaskCommand;

// 恢复任务命令
typedef struct {
    uint64_t task_id;               // 任务ID
} ResumeTaskCommand;

// 更新任务优先级命令
typedef struct {
    uint64_t task_id;               // 任务ID
    int new_priority;               // 新优先级
} UpdateTaskPriorityCommand;

// 任务选项
struct TaskOptions {
    bool detachable;                // 是否可分离
    bool cancellable;               // 是否可取消
    uint32_t timeout_ms;            // 超时时间
    char group[256];                // 任务组
    void* user_data;                // 用户数据
};

// 默认任务选项
extern const struct TaskOptions TASK_DEFAULT_OPTIONS;

// 命令验证函数
bool validate_create_task_command(const CreateTaskCommand* cmd);
bool validate_cancel_task_command(const CancelTaskCommand* cmd);
bool validate_pause_task_command(const PauseTaskCommand* cmd);
bool validate_resume_task_command(const ResumeTaskCommand* cmd);
bool validate_update_task_priority_command(const UpdateTaskPriorityCommand* cmd);

// 命令构建函数
CreateTaskCommand* create_task_command_create(
    const char* name,
    void* entry_point,
    void* arg,
    size_t stack_size
);

void create_task_command_destroy(CreateTaskCommand* cmd);

#endif // TASK_COMMANDS_H
