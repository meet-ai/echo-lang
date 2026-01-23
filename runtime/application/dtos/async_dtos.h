#ifndef ASYNC_DTOS_H
#define ASYNC_DTOS_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>

// 前向声明
struct Future;

// Future DTO - 数据传输对象
typedef struct {
    uint64_t id;                    // Future ID
    char state[32];                 // Future状态 (e.g., "PENDING", "RESOLVED", "REJECTED")
    void* result_data;              // 结果数据
    size_t result_size;             // 结果数据大小
    char error_message[1024];       // 错误信息
    time_t created_at;              // 创建时间
    time_t resolved_at;             // 解决时间
    time_t rejected_at;             // 拒绝时间
    uint64_t execution_time_us;     // 执行时间（微秒）
    uint32_t poll_count;            // 轮询次数
} FutureDTO;

// Future创建结果DTO
typedef struct {
    bool success;
    uint64_t future_id;
    char message[512];
    int error_code;
} FutureCreationResultDTO;

// 注意：AsyncOperationResultDTO 已在 result_dtos.h 中定义

// Future列表DTO
typedef struct {
    FutureDTO* futures;             // Future数组
    size_t count;                   // Future数量
    size_t total_count;             // 总Future数（用于分页）
    uint32_t page;                  // 当前页码
    uint32_t page_size;             // 每页大小
} FutureListDTO;

// Future统计DTO
typedef struct {
    uint32_t total_futures;         // 总Future数
    uint32_t pending_futures;       // 待处理Future数
    uint32_t resolved_futures;      // 已解决Future数
    uint32_t rejected_futures;      // 已拒绝Future数
    uint32_t cancelled_futures;     // 已取消Future数
    double average_execution_time_us; // 平均执行时间（微秒）
    uint64_t total_execution_time_us; // 总执行时间（微秒）
    time_t last_updated;            // 最后更新时间
} FutureStatisticsDTO;

// DTO创建和销毁函数
FutureDTO* future_dto_create(uint64_t id, const char* state, void* result_data, size_t result_size, const char* error_message, time_t created_at, time_t resolved_at, time_t rejected_at, uint64_t execution_time_us, uint32_t poll_count);
void future_dto_destroy(FutureDTO* dto);

FutureCreationResultDTO* future_creation_result_dto_create(bool success, uint64_t future_id, const char* message, int error_code);
void future_creation_result_dto_destroy(FutureCreationResultDTO* dto);

// 注意：AsyncOperationResultDTO 的创建和销毁函数已在 result_dtos.h 中定义

FutureListDTO* future_list_dto_create(FutureDTO* futures, size_t count, size_t total_count, uint32_t page, uint32_t page_size);
void future_list_dto_destroy(FutureListDTO* dto);

FutureStatisticsDTO* future_statistics_dto_create(uint32_t total, uint32_t pending, uint32_t resolved, uint32_t rejected, uint32_t cancelled, double avg_exec_time, uint64_t total_exec_time, time_t last_updated);
void future_statistics_dto_destroy(FutureStatisticsDTO* dto);

// 转换函数
FutureDTO* future_to_dto(const struct Future* future);
FutureListDTO* futures_to_list_dto(struct Future** futures, size_t count, size_t total, uint32_t page, uint32_t page_size);
FutureStatisticsDTO* calculate_future_statistics(struct Future** futures, size_t count);

#endif // ASYNC_DTOS_H
