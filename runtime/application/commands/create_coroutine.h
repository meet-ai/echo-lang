/**
 * @file create_coroutine.h
 * @brief 创建协程命令
 *
 * 定义创建协程的命令对象，包含所有必要的参数
 */

#ifndef CREATE_COROUTINE_COMMAND_H
#define CREATE_COROUTINE_COMMAND_H

#include <stdint.h>
#include <stdbool.h>

// 协程配置
typedef struct {
    uint32_t stack_size_kb;       // 栈大小（KB），0表示使用默认值
    int priority;                 // 优先级，-1表示默认优先级
    const char* name;             // 协程名称，可为空
    bool enable_debug;            // 是否启用调试
} coroutine_config_t;

// 创建协程命令
typedef struct create_coroutine_command_t {
    // 命令标识
    const char* command_id;       // 命令唯一标识

    // 协程信息
    void (*entry_point)(void*);   // 入口函数
    void* argument;               // 入口函数参数
    coroutine_config_t config;    // 协程配置

    // 元数据
    uint64_t timestamp;           // 命令创建时间戳
    const char* source;           // 命令来源
} create_coroutine_command_t;

// 命令验证结果
typedef struct {
    bool valid;                   // 是否有效
    const char* error_message;    // 错误信息，无效时填写
} command_validation_result_t;

// 便利函数
create_coroutine_command_t* create_coroutine_command_create(
    void (*entry_point)(void*),
    void* argument,
    const coroutine_config_t* config
);

void create_coroutine_command_destroy(create_coroutine_command_t* command);

// 验证命令
command_validation_result_t create_coroutine_command_validate(
    const create_coroutine_command_t* command
);

// 获取默认配置
coroutine_config_t create_coroutine_command_default_config(void);

#endif // CREATE_COROUTINE_COMMAND_H
