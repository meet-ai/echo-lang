/**
 * @file sync-caller.h
 * @brief 同步调用模式接口
 *
 * 定义同步调用的标准接口和协议
 */

#ifndef SYNC_CALLER_H
#define SYNC_CALLER_H

#include <stdint.h>
#include <stdbool.h>

// 同步调用配置
typedef struct {
    uint64_t timeout_ms;          // 超时时间（毫秒）
    uint32_t max_retries;         // 最大重试次数
    uint64_t retry_delay_ms;      // 重试延迟（毫秒）
    bool enable_circuit_breaker;  // 是否启用熔断器
} sync_call_config_t;

// 同步调用结果
typedef struct {
    bool success;                 // 是否成功
    void* result;                 // 结果数据
    size_t result_size;           // 结果大小
    const char* error_message;    // 错误信息
    uint64_t execution_time_ms;   // 执行时间
} sync_call_result_t;

// 同步调用器接口
typedef struct sync_caller_t {
    // 配置
    sync_call_config_t config;

    // 方法
    bool (*init)(struct sync_caller_t* self, const sync_call_config_t* config);
    sync_call_result_t* (*call)(struct sync_caller_t* self,
                               const char* service_name,
                               const char* method_name,
                               const void* request_data,
                               size_t request_size);
    bool (*health_check)(struct sync_caller_t* self, const char* service_name);
    void (*destroy)(struct sync_caller_t* self);

    // 私有数据
    void* private_data;
} sync_caller_t;

// 同步调用函数类型
typedef sync_call_result_t* (*sync_call_function_t)(
    const char* service_name,
    const char* method_name,
    const void* request_data,
    size_t request_size,
    uint64_t timeout_ms
);

// 便利函数
sync_caller_t* sync_caller_create(void);
void sync_caller_destroy(sync_caller_t* caller);

sync_call_result_t* sync_call_result_create(void);
void sync_call_result_destroy(sync_call_result_t* result);

// 默认配置
extern const sync_call_config_t default_sync_call_config;

#endif // SYNC_CALLER_H
