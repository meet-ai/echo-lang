/**
 * @file exception_handler.h
 * @brief 异常处理器接口定义
 *
 * 定义异常处理器的标准接口，所有上下文都可以使用统一的异常处理机制
 */

#ifndef EXCEPTION_HANDLER_H
#define EXCEPTION_HANDLER_H

#include "runtime_exception.h"
#include <stdbool.h>

// 异常处理策略
typedef enum {
    EXCEPTION_STRATEGY_IGNORE,      // 忽略异常
    EXCEPTION_STRATEGY_LOG,         // 只记录日志
    EXCEPTION_STRATEGY_RETRY,       // 重试操作
    EXCEPTION_STRATEGY_RECOVER,     // 尝试恢复
    EXCEPTION_STRATEGY_ABORT        // 终止程序
} exception_strategy_t;

// 异常处理器配置
typedef struct {
    exception_strategy_t strategy;  // 处理策略
    int max_retries;               // 最大重试次数
    uint64_t retry_delay_ms;       // 重试延迟（毫秒）
    bool log_stack_trace;          // 是否记录栈追踪
    const char* log_level;         // 日志级别
} exception_handler_config_t;

// 异常处理器接口
typedef struct exception_handler_t {
    // 配置
    exception_handler_config_t config;

    // 方法
    bool (*can_handle)(struct exception_handler_t* self, exception_type_t type);
    exception_strategy_t (*get_strategy)(struct exception_handler_t* self, runtime_exception_t* exception);
    bool (*handle)(struct exception_handler_t* self, runtime_exception_t* exception);
    void (*log_exception)(struct exception_handler_t* self, runtime_exception_t* exception);
    bool (*should_retry)(struct exception_handler_t* self, runtime_exception_t* exception, int attempt_count);
    void (*cleanup)(struct exception_handler_t* self);

    // 私有数据
    void* private_data;
} exception_handler_t;

// 便利函数
exception_handler_t* exception_handler_create(const exception_handler_config_t* config);
void exception_handler_destroy(exception_handler_t* handler);

// 默认异常处理器实现
extern exception_handler_t* default_exception_handler;

#endif // EXCEPTION_HANDLER_H
