#ifndef EXCEPTION_HANDLING_H
#define EXCEPTION_HANDLING_H

#include "../types/common_types.h"
#include <setjmp.h>
#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct Exception;
struct ExceptionHandler;
struct ExceptionContext;
struct ExceptionStack;

// 异常类型枚举
typedef enum {
    EXCEPTION_NONE = 0,

    // 系统异常 (1-999)
    EXCEPTION_SYSTEM_ERROR = 1,
    EXCEPTION_MEMORY_ERROR,
    EXCEPTION_IO_ERROR,
    EXCEPTION_NETWORK_ERROR,
    EXCEPTION_TIMEOUT_ERROR,
    EXCEPTION_PERMISSION_ERROR,
    EXCEPTION_RESOURCE_ERROR,

    // 运行时异常 (1000-1999)
    EXCEPTION_RUNTIME_ERROR = 1000,
    EXCEPTION_TYPE_ERROR,
    EXCEPTION_VALUE_ERROR,
    EXCEPTION_INDEX_ERROR,
    EXCEPTION_KEY_ERROR,
    EXCEPTION_ATTRIBUTE_ERROR,
    EXCEPTION_STATE_ERROR,

    // 并发异常 (2000-2999)
    EXCEPTION_CONCURRENCY_ERROR = 2000,
    EXCEPTION_DEADLOCK_ERROR,
    EXCEPTION_RACE_CONDITION_ERROR,
    EXCEPTION_SYNCHRONIZATION_ERROR,

    // 任务异常 (3000-3999)
    EXCEPTION_TASK_ERROR = 3000,
    EXCEPTION_COROUTINE_ERROR,
    EXCEPTION_FUTURE_ERROR,
    EXCEPTION_CHANNEL_ERROR,

    // 用户自定义异常 (4000+)
    EXCEPTION_USER_BASE = 4000
} exception_type_t;

// 异常严重程度
typedef enum {
    SEVERITY_LOW,      // 低严重程度
    SEVERITY_MEDIUM,   // 中等严重程度
    SEVERITY_HIGH,     // 高严重程度
    SEVERITY_CRITICAL  // 严重程度
} exception_severity_t;

// 异常信息
typedef struct Exception {
    exception_type_t type;        // 异常类型
    exception_severity_t severity; // 严重程度
    entity_id_t exception_id;     // 异常ID
    timestamp_t timestamp;        // 发生时间

    string_t message;             // 异常消息
    string_t source_file;         // 源文件
    uint32_t source_line;         // 源行号
    string_t source_function;     // 源函数

    string_t stack_trace;         // 栈跟踪
    hash_table_t context_data;    // 上下文数据
    void* payload;                // 异常负载
    size_t payload_size;          // 负载大小

    bool is_handled;              // 是否已被处理
    bool can_retry;               // 是否可以重试
    uint32_t retry_count;         // 重试次数
    duration_t retry_delay;       // 重试延迟

    struct Exception* cause;      // 原因异常（异常链）
} exception_t;

// 异常处理器
typedef struct ExceptionHandler {
    entity_id_t handler_id;       // 处理器ID
    string_t name;                // 处理器名称
    exception_type_t handled_type; // 处理的异常类型
    exception_severity_t min_severity; // 最小处理严重程度

    // 处理函数
    bool (*can_handle)(const exception_t* exception, void* context);
    result_status_t (*handle)(exception_t* exception, void* context);

    // 恢复函数（可选）
    bool (*can_recover)(const exception_t* exception, void* context);
    result_status_t (*recover)(exception_t* exception, void* context);

    void* user_context;           // 用户上下文
    cleanup_function_t cleanup;   // 清理函数

    bool is_active;               // 是否激活
    timestamp_t created_at;       // 创建时间
    uint64_t handled_count;       // 处理的异常数量
} exception_handler_t;

// 异常上下文
typedef struct ExceptionContext {
    entity_id_t context_id;       // 上下文ID
    string_t context_name;        // 上下文名称
    jmp_buf jump_buffer;          // 跳转缓冲区（用于setjmp/longjmp）

    exception_t* current_exception; // 当前异常
    dynamic_array_t exception_stack; // 异常栈

    hash_table_t handlers;        // 异常处理器表
    hash_table_t context_data;    // 上下文数据

    bool exception_pending;       // 是否有待处理异常
    bool unwind_in_progress;      // 是否正在展开

    timestamp_t created_at;       // 创建时间
    uint64_t exception_count;     // 异常数量
} exception_context_t;

// 异常栈
typedef struct ExceptionStack {
    exception_context_t** contexts; // 上下文数组
    size_t context_count;         // 上下文数量
    size_t max_depth;             // 最大深度
    spin_lock_t stack_lock;       // 栈锁
} exception_stack_t;

// 异常处理接口
typedef struct {
    // 异常创建和管理
    exception_t* (*create_exception)(exception_type_t type,
                                   exception_severity_t severity,
                                   const char* message,
                                   const char* file,
                                   uint32_t line,
                                   const char* function);
    void (*destroy_exception)(exception_t* exception);

    // 异常抛出和捕获
    void (*throw_exception)(exception_context_t* context, exception_t* exception);
    exception_t* (*catch_exception)(exception_context_t* context);

    // 异常处理器管理
    bool (*register_handler)(exception_context_t* context, exception_handler_t* handler);
    bool (*unregister_handler)(exception_context_t* context, entity_id_t handler_id);
    exception_handler_t* (*find_handler)(exception_context_t* context, const exception_t* exception);

    // 异常处理
    result_status_t (*handle_exception)(exception_context_t* context, exception_t* exception);
    result_status_t (*try_recover)(exception_context_t* context, exception_t* exception);

    // 上下文管理
    exception_context_t* (*create_context)(const char* name);
    void (*destroy_context)(exception_context_t* context);
    exception_context_t* (*get_current_context)(void);
    bool (*set_current_context)(exception_context_t* context);

    // 异常链管理
    bool (*chain_exceptions)(exception_t* exception, exception_t* cause);
    exception_t* (*get_root_cause)(const exception_t* exception);

    // 诊断和调试
    string_t (*format_exception)(const exception_t* exception);
    bool (*dump_exception_stack)(const exception_context_t* context, string_t* output);
    bool (*get_exception_stats)(const exception_context_t* context, hash_table_t* stats);
} exception_handling_interface_t;

// 异常处理系统
typedef struct ExceptionHandlingSystem {
    exception_handling_interface_t* vtable;  // 虚函数表
    exception_stack_t exception_stack;       // 异常栈
    atomic_int32_t active_contexts;          // 活跃上下文计数
    hash_table_t global_handlers;            // 全局处理器
    bool is_initialized;                     // 是否已初始化
} exception_handling_system_t;

// 异常处理系统工厂函数
exception_handling_system_t* exception_handling_system_create(void);
void exception_handling_system_destroy(exception_handling_system_t* system);

// 便捷宏定义异常处理
#define EXCEPTION_TRY(context) \
    do { \
        exception_context_t* _ctx = (context); \
        if (setjmp(_ctx->jump_buffer) == 0) {

#define EXCEPTION_CATCH(exception_var) \
        } else { \
            exception_t* exception_var = exception_handling_system_get_current()->vtable->catch_exception(_ctx);

#define EXCEPTION_END \
        } \
    } while(0)

#define EXCEPTION_THROW(context, type, severity, message) \
    do { \
        exception_handling_system_t* _sys = exception_handling_system_get_current(); \
        exception_t* _ex = _sys->vtable->create_exception(type, severity, message, \
                                                        __FILE__, __LINE__, __func__); \
        _sys->vtable->throw_exception(context, _ex); \
    } while(0)

#define EXCEPTION_THROW_SIMPLE(context, type, message) \
    EXCEPTION_THROW(context, type, SEVERITY_MEDIUM, message)

// 便捷函数
static inline exception_handling_system_t* exception_handling_system_get_current(void) {
    static exception_handling_system_t* current_system = NULL;
    if (!current_system) {
        current_system = exception_handling_system_create();
    }
    return current_system;
}

static inline exception_context_t* exception_context_get_current(void) {
    exception_handling_system_t* system = exception_handling_system_get_current();
    return system->vtable->get_current_context();
}

static inline void exception_throw(exception_type_t type, exception_severity_t severity, const char* message) {
    exception_context_t* context = exception_context_get_current();
    EXCEPTION_THROW(context, type, severity, message);
}

// 标准异常创建宏
#define THROW_SYSTEM_ERROR(msg) EXCEPTION_THROW_SIMPLE(exception_context_get_current(), EXCEPTION_SYSTEM_ERROR, msg)
#define THROW_MEMORY_ERROR(msg) EXCEPTION_THROW_SIMPLE(exception_context_get_current(), EXCEPTION_MEMORY_ERROR, msg)
#define THROW_RUNTIME_ERROR(msg) EXCEPTION_THROW_SIMPLE(exception_context_get_current(), EXCEPTION_RUNTIME_ERROR, msg)
#define THROW_CONCURRENCY_ERROR(msg) EXCEPTION_THROW_SIMPLE(exception_context_get_current(), EXCEPTION_CONCURRENCY_ERROR, msg)

// 断言宏（抛出异常）
#define EXCEPTION_ASSERT(condition, type, message) \
    do { \
        if (!(condition)) { \
            EXCEPTION_THROW(exception_context_get_current(), type, SEVERITY_HIGH, message); \
        } \
    } while(0)

#endif // EXCEPTION_HANDLING_H
