#ifndef FUTURE_H
#define FUTURE_H

#include <stdint.h>
#include <stdbool.h>
#include <pthread.h>

// 前向声明Task结构体，避免循环依赖
struct Task;

// Future状态枚举
typedef enum {
    FUTURE_PENDING,   // 异步操作未完成
    FUTURE_RESOLVED,  // 异步操作已完成（成功）
    FUTURE_REJECTED   // 异步操作已完成（失败）
} FutureState;

// 轮询状态枚举
typedef enum {
    POLL_PENDING,   // 异步操作未完成
    POLL_READY,     // 异步操作已完成，结果可用
    POLL_COMPLETED  // Future已完成并被消费
} PollStatus;

// 轮询结果结构体
typedef struct PollResult {
    PollStatus status;
    void* value;    // 结果值
} PollResult;

// 等待队列节点
typedef struct WaitNode {
    struct Task* task;          // 等待的Task
    struct WaitNode* next;      // 下一个节点
} WaitNode;

// Future优先级枚举
typedef enum {
    FUTURE_PRIORITY_LOW,
    FUTURE_PRIORITY_NORMAL,
    FUTURE_PRIORITY_HIGH,
    FUTURE_PRIORITY_CRITICAL
} FuturePriority;

// Future结构体
typedef struct Future {
    uint64_t id;               // Future唯一ID
    FutureState state;         // Future状态
    void* result;              // 结果值（成功）或错误（失败）
    bool consumed;             // 是否已被消费

    // 等待队列
    WaitNode* wait_queue_head; // 等待队列头
    WaitNode* wait_queue_tail; // 等待队列尾
    int wait_count;            // 等待的任务数量

    // 同步机制
    pthread_mutex_t mutex;     // 保护Future状态
    pthread_cond_t cond;       // 条件变量，用于等待

    // 时间戳
    time_t created_at;         // 创建时间
    time_t resolved_at;        // 完成时间（成功）
    time_t rejected_at;        // 完成时间（失败）

    // 统计信息
    uint64_t poll_count;       // 轮询次数
    uint64_t wait_time_ms;     // 等待时间（毫秒）
    FuturePriority priority;   // Future优先级

    // 资源管理
    bool result_needs_free;    // 结果是否需要释放

    // 可选方法
    void (*cancel)(struct Future* self);     // 取消异步操作
    void (*cleanup)(struct Future* self);    // 清理资源
} Future;

// Future管理函数声明
Future* future_new(void);
void future_destroy(Future* future);
struct PollResult future_poll(Future* future, struct Task* task);
void future_cancel(Future* future);
uint64_t future_generate_id(void);

// 工具函数
void future_resolve(Future* future, void* value);
void future_reject(Future* future, void* error);
void* future_wait(Future* future);

#endif // FUTURE_H
