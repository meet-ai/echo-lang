#ifndef ECHO_GC_TRIGGER_GC_COMMAND_H
#define ECHO_GC_TRIGGER_GC_COMMAND_H

#include <stdint.h>
#include <stdbool.h>
#include "echo/gc.h"

// 前向声明
struct echo_gc_collector;
struct echo_gc_heap;
struct echo_gc_mark_service;
struct echo_gc_sweep_service;

// GC触发命令
typedef struct echo_gc_trigger_command {
    echo_gc_trigger_reason_t reason;   // 触发原因
    bool force_gc;                     // 是否强制GC（忽略安全检查）
    uint64_t timeout_ms;               // 命令超时时间
} echo_gc_trigger_command_t;

// GC完成命令
typedef struct echo_gc_complete_command {
    echo_gc_metrics_t metrics;         // GC执行指标
    bool success;                      // 是否成功完成
    const char* error_message;         // 错误信息（如果有）
} echo_gc_complete_command_t;

// 命令处理器
typedef struct echo_gc_command_handler {
    struct echo_gc_collector* collector;
    struct echo_gc_heap* heap;
    struct echo_gc_mark_service* mark_service;
    struct echo_gc_sweep_service* sweep_service;
} echo_gc_command_handler_t;

// 命令处理器生命周期
echo_gc_command_handler_t* echo_gc_command_handler_create(
    struct echo_gc_collector* collector,
    struct echo_gc_heap* heap,
    struct echo_gc_mark_service* mark_service,
    struct echo_gc_sweep_service* sweep_service
);

void echo_gc_command_handler_destroy(echo_gc_command_handler_t* handler);

// 命令处理
echo_gc_error_t echo_gc_trigger_gc_command_handle(
    echo_gc_command_handler_t* handler,
    const echo_gc_trigger_command_t* command
);

echo_gc_error_t echo_gc_complete_gc_command_handle(
    echo_gc_command_handler_t* handler,
    const echo_gc_complete_command_t* command
);

// 命令验证
bool echo_gc_trigger_command_validate(const echo_gc_trigger_command_t* command);
bool echo_gc_complete_command_validate(const echo_gc_complete_command_t* command);

// 命令结果
typedef struct command_result {
    bool success;
    echo_gc_error_t error_code;
    const char* error_message;
    void* data;                        // 附加数据
} command_result_t;

#endif // ECHO_GC_TRIGGER_GC_COMMAND_H
