#ifndef COROUTINE_DTOS_H
#define COROUTINE_DTOS_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>

// 协程DTO - 数据传输对象
typedef struct {
    uint64_t coroutine_id;          // 协程ID
    char name[256];                 // 协程名称
    char description[512];          // 协程描述
    char status[32];                // 协程状态
    char priority[16];              // 协程优先级
    time_t created_at;              // 创建时间
    time_t started_at;              // 开始时间
    time_t completed_at;            // 完成时间
    time_t last_yield_at;           // 最后yield时间
    time_t last_resume_at;          // 最后resume时间
    int exit_code;                  // 退出代码
    char error_message[1024];       // 错误信息
    uint64_t execution_time_ms;     // 执行时间（毫秒）
    size_t stack_usage;             // 栈使用量
    uint32_t yield_count;           // yield次数
    uint32_t resume_count;          // resume次数
} CoroutineDTO;

// 协程列表DTO
typedef struct {
    CoroutineDTO* coroutines;       // 协程数组
    size_t count;                   // 协程数量
    size_t total_count;             // 总协程数（用于分页）
    uint32_t page;                  // 当前页码
    uint32_t page_size;             // 每页大小
} CoroutineListDTO;

// 协程统计DTO
typedef struct {
    uint32_t total_coroutines;      // 总协程数
    uint32_t running_coroutines;    // 运行中协程数
    uint32_t suspended_coroutines;  // 挂起协程数
    uint32_t completed_coroutines;  // 已完成协程数
    uint32_t failed_coroutines;     // 失败协程数
    uint32_t cancelled_coroutines;  // 已取消协程数
    double average_execution_time;  // 平均执行时间
    uint64_t total_execution_time;  // 总执行时间
    size_t total_stack_usage;       // 总栈使用量
    uint32_t total_yield_count;     // 总yield次数
    time_t last_updated;            // 最后更新时间
} CoroutineStatisticsDTO;

// DTO构造函数
CoroutineDTO* coroutine_dto_create(uint64_t coroutine_id, const char* name, const char* status);
CoroutineListDTO* coroutine_list_dto_create(size_t count);
CoroutineStatisticsDTO* coroutine_statistics_dto_create(void);

// DTO销毁函数
void coroutine_dto_destroy(CoroutineDTO* dto);
void coroutine_list_dto_destroy(CoroutineListDTO* dto);
void coroutine_statistics_dto_destroy(CoroutineStatisticsDTO* dto);

#endif // COROUTINE_DTOS_H
