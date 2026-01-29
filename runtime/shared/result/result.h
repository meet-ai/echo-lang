#ifndef RESULT_H
#define RESULT_H

#include <stdint.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>

// ============================================================================
// Result 类型运行时支持
// ============================================================================
// Result[T, E] 在 LLVM IR 中被映射为 i8*（不透明指针）
// 运行时系统需要提供创建和提取 Result 值的函数

// Result 结构体（内部表示）
// 注意：这个结构体只在运行时系统内部使用，编译器生成的代码只看到 i8* 指针
typedef struct {
    bool is_ok;           // true 表示 Ok，false 表示 Err
    void* value;          // Ok 值或 Err 值的指针
    size_t value_size;    // 值的大小（字节）
    char error_type[64];  // 错误类型（如果是 Err）
} Result;

// ============================================================================
// Result 创建函数
// ============================================================================

/**
 * @brief 创建 Ok 值
 * @param value 值的指针
 * @param value_size 值的大小（字节）
 * @return Result 指针（i8*），失败返回 NULL
 * 
 * 使用场景：
 * - runtime_open_file 成功时返回 Ok(FileHandle)
 * - runtime_tcp_connect 成功时返回 Ok(SocketHandle)
 */
void* result_ok(void* value, size_t value_size);

/**
 * @brief 创建 Err 值
 * @param error_message 错误消息字符串
 * @return Result 指针（i8*），失败返回 NULL
 * 
 * 使用场景：
 * - runtime_open_file 失败时返回 Err(string)
 * - runtime_tcp_connect 失败时返回 Err(string)
 */
void* result_err(const char* error_message);

/**
 * @brief 创建 Err 值（带错误类型）
 * @param error_type 错误类型字符串
 * @param error_message 错误消息字符串
 * @return Result 指针（i8*），失败返回 NULL
 */
void* result_err_with_type(const char* error_type, const char* error_message);

// ============================================================================
// Result 提取函数
// ============================================================================

/**
 * @brief 检查 Result 是否为 Ok
 * @param result Result 指针（i8*）
 * @return true 表示 Ok，false 表示 Err
 */
bool result_is_ok(void* result);

/**
 * @brief 检查 Result 是否为 Err
 * @param result Result 指针（i8*）
 * @return true 表示 Err，false 表示 Ok
 */
bool result_is_err(void* result);

/**
 * @brief 提取 Ok 值
 * @param result Result 指针（i8*）
 * @param value_buffer 输出缓冲区
 * @param buffer_size 缓冲区大小
 * @param actual_size 输出实际值的大小
 * @return true 表示成功提取，false 表示 Result 是 Err 或参数无效
 */
bool result_unwrap_ok(void* result, void* value_buffer, size_t buffer_size, size_t* actual_size);

/**
 * @brief 提取 Err 错误消息
 * @param result Result 指针（i8*）
 * @param error_buffer 输出缓冲区
 * @param buffer_size 缓冲区大小
 * @return true 表示成功提取，false 表示 Result 是 Ok 或参数无效
 */
bool result_unwrap_err(void* result, char* error_buffer, size_t buffer_size);

/**
 * @brief 释放 Result 内存
 * @param result Result 指针（i8*）
 */
void result_free(void* result);

// ============================================================================
// Result 辅助函数
// ============================================================================

/**
 * @brief 从 Result 中提取值（如果 Ok）或返回默认值
 * @param result Result 指针（i8*）
 * @param default_value 默认值指针
 * @param value_size 值的大小
 * @param output_buffer 输出缓冲区
 * @return true 表示使用 Ok 值，false 表示使用默认值
 */
bool result_unwrap_or(void* result, void* default_value, size_t value_size, void* output_buffer);

/**
 * @brief 从 Result 中提取值（如果 Ok）或使用错误回调
 * @param result Result 指针（i8*）
 * @param value_buffer 输出缓冲区
 * @param buffer_size 缓冲区大小
 * @param error_callback 错误回调函数（可选，NULL 表示忽略）
 * @return true 表示成功提取 Ok 值，false 表示是 Err
 */
bool result_unwrap_or_else(void* result, void* value_buffer, size_t buffer_size, 
                           void (*error_callback)(const char*));

#endif // RESULT_H
