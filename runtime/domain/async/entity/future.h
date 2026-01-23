#ifndef FUTURE_H
#define FUTURE_H

#include <stdint.h>
#include <stdbool.h>
#include <pthread.h>

// 前向声明
struct Future;
struct Promise;

// Future状态枚举
typedef enum {
    FUTURE_STATUS_PENDING,    // 等待中
    FUTURE_STATUS_RESOLVED,   // 已解决
    FUTURE_STATUS_REJECTED,   // 已拒绝
    FUTURE_STATUS_CANCELLED   // 已取消
} FutureStatus;

// Future实体 - 代表异步操作的结果
typedef struct Future {
    uint64_t id;                    // Future唯一标识
    FutureStatus status;            // 当前状态
    void* result;                   // 结果值
    char* error_message;            // 错误信息
    time_t created_at;              // 创建时间
    time_t resolved_at;             // 解决时间

    // 内部实现
    pthread_mutex_t mutex;          // 状态同步锁
    pthread_cond_t cond;            // 条件变量
    bool has_result;                // 是否有结果

    // 回调函数（可选，用于异步通知）
    void (*on_resolved)(struct Future* future, void* result, void* context);
    void (*on_rejected)(struct Future* future, const char* error, void* context);
    void* callback_context;
} Future;

// Promise值对象 - 代表异步操作的承诺
typedef struct Promise {
    Future* future;                 // 关联的Future
} Promise;

// Future方法
Future* future_create(void);
void future_destroy(Future* future);

// 同步等待结果
void* future_get(Future* future);
bool future_get_with_timeout(Future* future, void** result, uint32_t timeout_ms);

// 检查状态
FutureStatus future_get_status(const Future* future);
bool future_is_pending(const Future* future);
bool future_is_resolved(const Future* future);
bool future_is_rejected(const Future* future);
bool future_is_cancelled(const Future* future);

// 设置回调
void future_on_resolved(Future* future, void (*callback)(Future*, void*, void*), void* context);
void future_on_rejected(Future* future, void (*callback)(Future*, const char*, void*), void* context);

// Promise方法
Promise* promise_create(void);
void promise_destroy(Promise* promise);

// 解决Promise
bool promise_resolve(Promise* promise, void* result);
bool promise_reject(Promise* promise, const char* error_message);
bool promise_cancel(Promise* promise);

// 获取关联的Future
Future* promise_get_future(const Promise* promise);

#endif // FUTURE_H
