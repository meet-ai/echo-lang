#ifndef FUTURE_COMMANDS_H
#define FUTURE_COMMANDS_H

#include <stdint.h>
#include <stdbool.h>

// 创建Future命令
typedef struct {
    char description[512];          // Future描述
    void* initial_value;            // 初始值（可选）
} CreateFutureCommand;

// 解决Future命令
typedef struct {
    uint64_t future_id;             // Future ID
    void* result;                   // 结果值
} ResolveFutureCommand;

// 拒绝Future命令
typedef struct {
    uint64_t future_id;             // Future ID
    char error_message[1024];       // 错误信息
} RejectFutureCommand;

// 取消Future命令
typedef struct {
    uint64_t future_id;             // Future ID
    char reason[512];               // 取消原因
} CancelFutureCommand;

// 创建Promise命令
typedef struct {
    char description[512];          // Promise描述
} CreatePromiseCommand;

// 等待Future命令
typedef struct {
    uint64_t future_id;             // Future ID
    uint32_t timeout_ms;            // 等待超时时间
    bool blocking;                  // 是否阻塞等待
} AwaitFutureCommand;

// Future组合命令
typedef struct {
    uint64_t* future_ids;           // Future ID列表
    size_t count;                   // Future数量
    char operation[64];             // 操作类型："all", "race", "any", "allSettled"
} CombineFuturesCommand;

// 超时Future命令
typedef struct {
    uint64_t future_id;             // Future ID
    uint32_t timeout_ms;            // 超时时间
} TimeoutFutureCommand;

// 重试Future命令
typedef struct {
    uint64_t future_id;             // Future ID
    uint32_t max_retries;           // 最大重试次数
    uint32_t delay_ms;              // 重试延迟
} RetryFutureCommand;

// 延迟Future命令
typedef struct {
    void* value;                    // 延迟值
    uint32_t delay_ms;              // 延迟时间
} DelayFutureCommand;

// 条件等待命令
typedef struct {
    bool (*condition)(void* context); // 条件函数
    void* context;                   // 条件上下文
    uint32_t check_interval_ms;      // 检查间隔
    uint32_t timeout_ms;             // 总超时时间
} WhenFutureCommand;

// 生成异步任务命令
typedef struct {
    char task_name[256];             // 任务名称
    void (*task_func)(void*);        // 任务函数
    void* task_arg;                  // 任务参数
    uint32_t priority;               // 任务优先级
} SpawnAsyncTaskCommand;

// 设置Future超时命令
typedef struct {
    uint64_t future_id;              // Future ID
    uint32_t timeout_ms;             // 超时时间
} SetFutureTimeoutCommand;

// 链式Future命令
typedef struct {
    uint64_t source_future_id;       // 源Future ID
    uint64_t target_future_id;       // 目标Future ID
    void (*transform_func)(void*, void**); // 转换函数
} ChainFuturesCommand;

// 命令验证函数
bool validate_create_future_command(const CreateFutureCommand* cmd);
bool validate_resolve_future_command(const ResolveFutureCommand* cmd);
bool validate_reject_future_command(const RejectFutureCommand* cmd);
bool validate_cancel_future_command(const CancelFutureCommand* cmd);
bool validate_create_promise_command(const CreatePromiseCommand* cmd);
bool validate_await_future_command(const AwaitFutureCommand* cmd);
bool validate_combine_futures_command(const CombineFuturesCommand* cmd);
bool validate_timeout_future_command(const TimeoutFutureCommand* cmd);
bool validate_retry_future_command(const RetryFutureCommand* cmd);
bool validate_delay_future_command(const DelayFutureCommand* cmd);
bool validate_when_future_command(const WhenFutureCommand* cmd);

// 命令构建函数
CreateFutureCommand* create_future_command_create(const char* description);
void create_future_command_destroy(CreateFutureCommand* cmd);

CombineFuturesCommand* combine_futures_command_create(uint64_t* future_ids, size_t count, const char* operation);
void combine_futures_command_destroy(CombineFuturesCommand* cmd);

#endif // FUTURE_COMMANDS_H
