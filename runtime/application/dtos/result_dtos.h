#ifndef RESULT_DTOS_H
#define RESULT_DTOS_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>

// 通用操作结果DTO
typedef struct {
    bool success;                   // 操作是否成功
    char message[512];              // 结果消息
    int error_code;                 // 错误代码（成功时为0）
    char error_details[1024];       // 错误详情
    time_t timestamp;               // 操作时间戳
    uint64_t operation_id;          // 操作ID（用于跟踪）
} OperationResultDTO;

// 协程创建结果DTO
typedef struct {
    bool success;                   // 创建是否成功
    uint64_t coroutine_id;          // 协程ID
    char message[512];              // 结果消息
    int error_code;                 // 错误代码
    char error_details[1024];       // 错误详情
    time_t created_at;              // 创建时间
} CoroutineCreationResultDTO;

// 异步操作结果DTO
typedef struct {
    uint64_t operation_id;          // 操作ID
    bool success;                   // 操作是否成功
    char message[512];               // 结果消息
    char status[32];                // 操作状态
    void* result_data;              // 结果数据
    size_t result_size;             // 结果数据大小
    size_t data_size;                // 数据大小（兼容字段）
    char error_message[1024];       // 错误信息
    time_t started_at;              // 开始时间
    time_t completed_at;            // 完成时间
    time_t timestamp;               // 时间戳（兼容字段）
    uint64_t duration_ms;           // 执行时长
    double progress_percentage;     // 进度百分比（0.0-100.0）
    void* details;                  // 详细信息（兼容字段）
} AsyncOperationResultDTO;

// 批量操作结果DTO
typedef struct {
    uint32_t total_operations;      // 总操作数
    uint32_t successful_operations; // 成功操作数
    uint32_t failed_operations;     // 失败操作数
    OperationResultDTO* results;    // 各操作结果数组
    size_t results_count;           // 结果数量
    time_t batch_started_at;        // 批量操作开始时间
    time_t batch_completed_at;      // 批量操作完成时间
    uint64_t total_duration_ms;     // 总执行时长
} BatchOperationResultDTO;

// 查询结果DTO
typedef struct {
    bool success;                   // 查询是否成功
    void* data;                     // 查询数据
    size_t data_size;               // 数据大小
    uint32_t total_records;         // 总记录数
    uint32_t returned_records;      // 返回记录数
    uint32_t page;                  // 当前页码
    uint32_t total_pages;           // 总页数
    char query_id[128];             // 查询ID（用于缓存和调试）
    time_t executed_at;             // 执行时间
    uint64_t execution_time_ms;     // 执行时长
} QueryResultDTO;

// 错误结果DTO
typedef struct {
    int error_code;                 // 错误代码
    char error_type[64];            // 错误类型
    char error_message[512];        // 错误消息
    char error_details[1024];       // 错误详情
    char suggested_action[512];     // 建议操作
    time_t occurred_at;             // 发生时间
    char component[128];            // 发生错误的组件
    uint64_t request_id;            // 请求ID（用于跟踪）
} ErrorResultDTO;

// 成功结果DTO
typedef struct {
    void* data;                     // 成功数据
    size_t data_size;               // 数据大小
    char message[256];              // 成功消息
    time_t completed_at;            // 完成时间
    uint64_t operation_id;          // 操作ID
    uint64_t processing_time_ms;    // 处理时间
} SuccessResultDTO;

// 验证结果DTO
typedef struct {
    bool valid;                     // 是否有效
    char* validation_errors;        // 验证错误列表（JSON格式）
    size_t error_count;             // 错误数量
    time_t validated_at;            // 验证时间
    char validator_version[32];     // 验证器版本
} ValidationResultDTO;

// DTO构建函数
OperationResultDTO* operation_result_dto_create(bool success, const char* message, int error_code);
CoroutineCreationResultDTO* coroutine_creation_result_dto_create(bool success, uint64_t coroutine_id, const char* message);
AsyncOperationResultDTO* async_operation_result_dto_create(uint64_t operation_id, const char* status);
BatchOperationResultDTO* batch_operation_result_dto_create(uint32_t total_ops);
QueryResultDTO* query_result_dto_create(bool success, void* data, size_t data_size);
ErrorResultDTO* error_result_dto_create(int error_code, const char* message);
SuccessResultDTO* success_result_dto_create(void* data, size_t data_size, const char* message);
ValidationResultDTO* validation_result_dto_create(bool valid);

// DTO销毁函数
void operation_result_dto_destroy(OperationResultDTO* dto);
void coroutine_creation_result_dto_destroy(CoroutineCreationResultDTO* dto);
void async_operation_result_dto_destroy(AsyncOperationResultDTO* dto);
void batch_operation_result_dto_destroy(BatchOperationResultDTO* dto);
void query_result_dto_destroy(QueryResultDTO* dto);
void error_result_dto_destroy(ErrorResultDTO* dto);
void success_result_dto_destroy(SuccessResultDTO* dto);
void validation_result_dto_destroy(ValidationResultDTO* dto);

// DTO序列化
char* operation_result_dto_to_json(const OperationResultDTO* dto);
char* error_result_dto_to_json(const ErrorResultDTO* dto);
char* success_result_dto_to_json(const SuccessResultDTO* dto);

// JSON反序列化
OperationResultDTO* operation_result_dto_from_json(const char* json);
ErrorResultDTO* error_result_dto_from_json(const char* json);

#endif // RESULT_DTOS_H
