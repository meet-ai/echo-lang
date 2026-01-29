#ifndef COROUTINE_COMMANDS_H
#define COROUTINE_COMMANDS_H

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>  // for size_t

// 创建协程命令
typedef struct {
    char name[256];                 // 协程名称
    void* entry_point;              // 协程入口函数
    void* arg;                      // 协程参数
    size_t stack_size;              // 栈大小
    uint32_t priority;              // 协程优先级
    bool auto_start;                // 是否自动启动
} CreateCoroutineCommand;

// 挂起协程命令
typedef struct {
    uint64_t coroutine_id;          // 协程ID
    uint32_t timeout_ms;            // 挂起超时时间
} SuspendCoroutineCommand;

// 恢复协程命令
typedef struct {
    uint64_t coroutine_id;          // 协程ID
} ResumeCoroutineCommand;

// 取消协程命令
typedef struct {
    uint64_t coroutine_id;          // 协程ID
    char reason[512];               // 取消原因
} CancelCoroutineCommand;

// 切换协程上下文命令
typedef struct {
    uint64_t from_coroutine_id;     // 源协程ID
    uint64_t to_coroutine_id;       // 目标协程ID
    bool save_context;              // 是否保存上下文
} SwitchCoroutineContextCommand;

// 获取协程状态命令
typedef struct {
    uint64_t coroutine_id;          // 协程ID
} GetCoroutineStatusCommand;

// 协程启动命令
typedef struct {
    uint64_t coroutine_id;          // 协程ID
} StartCoroutineCommand;

// 协程暂停命令
typedef struct {
    uint64_t coroutine_id;          // 协程ID
} PauseCoroutineCommand;


// 协程优先级设置命令
typedef struct {
    uint64_t coroutine_id;          // 协程ID
    int priority;                   // 新优先级
} SetCoroutinePriorityCommand;

// 协程等待命令
typedef struct {
    uint64_t coroutine_id;          // 协程ID
    uint32_t timeout_ms;            // 等待超时时间
} WaitCoroutineCommand;

// 协程连接命令
typedef struct {
    uint64_t coroutine_id;          // 协程ID
    uint32_t timeout_ms;            // 连接超时时间
} JoinCoroutineCommand;

// 协程选项
typedef struct {
    bool share_stack;               // 是否共享栈
    uint32_t max_runtime_ms;        // 最大运行时间
    void* user_data;                // 用户数据
    void (*cleanup_callback)(void*); // 清理回调
} CoroutineOptions;

// 默认协程选项
extern const CoroutineOptions COROUTINE_DEFAULT_OPTIONS;

// 命令验证函数
bool validate_create_coroutine_command(const CreateCoroutineCommand* cmd);
bool validate_suspend_coroutine_command(const SuspendCoroutineCommand* cmd);
bool validate_resume_coroutine_command(const ResumeCoroutineCommand* cmd);
bool validate_cancel_coroutine_command(const CancelCoroutineCommand* cmd);
bool validate_switch_context_command(const SwitchCoroutineContextCommand* cmd);
bool validate_get_coroutine_status_command(const GetCoroutineStatusCommand* cmd);

// 命令构建函数
CreateCoroutineCommand* create_coroutine_command_create(
    const char* name,
    void* entry_point,
    void* arg,
    size_t stack_size
);

void create_coroutine_command_destroy(CreateCoroutineCommand* cmd);

#endif // COROUTINE_COMMANDS_H
