#ifndef SEMAPHORE_H
#define SEMAPHORE_H

#include <stdint.h>
#include <stdbool.h>
#include <semaphore.h>

// 前向声明
struct Semaphore;

// 信号量实体 - 实现资源计数和同步
typedef struct Semaphore {
    uint64_t id;                    // 信号量唯一标识
    sem_t semaphore;                // POSIX信号量
    uint32_t initial_value;         // 初始值
    uint32_t max_value;             // 最大值（用于有界信号量）

    // 统计信息
    uint64_t post_count_total;      // 总释放次数
    uint64_t wait_count_total;      // 总等待次数
    uint64_t timeout_count;         // 超时次数
    time_t created_at;              // 创建时间
} Semaphore;

// 信号量类型
typedef enum {
    SEMAPHORE_TYPE_UNNAMED,         // 无名信号量
    SEMAPHORE_TYPE_NAMED            // 命名信号量
} SemaphoreType;

// 信号量选项
typedef struct {
    SemaphoreType type;             // 信号量类型
    uint32_t initial_value;         // 初始值
    uint32_t max_value;             // 最大值（0表示无界）
    char name[256];                 // 命名信号量的名称
} SemaphoreOptions;

// 默认选项
extern const SemaphoreOptions SEMAPHORE_DEFAULT_OPTIONS;

// 信号量方法
Semaphore* semaphore_create(const SemaphoreOptions* options);
void semaphore_destroy(Semaphore* semaphore);

// 等待操作（P操作）
bool semaphore_wait(Semaphore* semaphore);
bool semaphore_try_wait(Semaphore* semaphore);
bool semaphore_timed_wait(Semaphore* semaphore, uint32_t timeout_ms);

// 释放操作（V操作）
bool semaphore_post(Semaphore* semaphore);

// 状态查询
int semaphore_get_value(const Semaphore* semaphore);
uint64_t semaphore_get_post_count(const Semaphore* semaphore);
uint64_t semaphore_get_wait_count(const Semaphore* semaphore);
uint64_t semaphore_get_timeout_count(const Semaphore* semaphore);

// 高级操作
bool semaphore_post_multiple(Semaphore* semaphore, uint32_t count);

#endif // SEMAPHORE_H
