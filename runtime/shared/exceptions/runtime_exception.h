/**
 * @file runtime_exception.h
 * @brief Runtime异常类型定义 - 共享内核
 *
 * 定义所有上下文共享的异常类型和异常处理器接口
 */

#ifndef RUNTIME_EXCEPTION_H
#define RUNTIME_EXCEPTION_H

#include <stdint.h>
#include <stdbool.h>

// 异常类型枚举
typedef enum {
    EXCEPTION_TYPE_SYSTEM,      // 系统级异常
    EXCEPTION_TYPE_RUNTIME,     // 运行时异常
    EXCEPTION_TYPE_APPLICATION, // 应用级异常
    EXCEPTION_TYPE_USER         // 用户自定义异常
} exception_type_t;

// 异常上下文信息
typedef struct {
    const char* file;           // 异常发生文件
    int line;                   // 异常发生行号
    const char* function;       // 异常发生函数
    uint64_t timestamp;         // 异常发生时间戳
    void* context_data;         // 上下文相关数据
} exception_context_t;

// 栈帧信息
typedef struct {
    const char* function_name;  // 函数名
    const char* file_name;      // 文件名
    int line_number;           // 行号
    void* instruction_ptr;     // 指令指针
} stack_frame_t;

// Runtime异常接口
typedef struct runtime_exception_t {
    // 基本信息
    exception_type_t type;              // 异常类型
    const char* message;                // 异常消息
    uint32_t error_code;                // 错误码

    // 上下文信息
    exception_context_t context;        // 异常上下文

    // 栈追踪
    stack_frame_t* stack_trace;         // 栈帧数组
    size_t stack_trace_size;            // 栈帧数量

    // 方法
    const char* (*error)(struct runtime_exception_t* self);
    exception_type_t (*get_type)(struct runtime_exception_t* self);
    exception_context_t* (*get_context)(struct runtime_exception_t* self);
    stack_frame_t* (*get_stack_trace)(struct runtime_exception_t* self, size_t* size);
} runtime_exception_t;

// 异常处理器接口
typedef struct exception_handler_t {
    // 方法
    bool (*can_handle)(struct exception_handler_t* self, exception_type_t type);
    bool (*handle)(struct exception_handler_t* self, runtime_exception_t* exception);
    void (*cleanup)(struct exception_handler_t* self);
} exception_handler_t;

// 便利函数
runtime_exception_t* runtime_exception_create(exception_type_t type, const char* message, uint32_t error_code);
void runtime_exception_destroy(runtime_exception_t* exception);
void runtime_exception_add_stack_frame(runtime_exception_t* exception, const char* function, const char* file, int line);

#endif // RUNTIME_EXCEPTION_H
