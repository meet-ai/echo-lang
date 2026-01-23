#ifndef ASYNC_RUNTIME_H
#define ASYNC_RUNTIME_H

#include "../entity/future.h"
#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct AsyncRuntime;
struct AsyncTask;

// 异步任务函数类型
typedef void* (*AsyncTaskFunc)(void* arg);

// 异步运行时领域服务 - 管理异步操作的执行
typedef struct AsyncRuntime {
    uint32_t max_concurrent_tasks;    // 最大并发任务数
    uint32_t running_task_count;      // 当前运行任务数

    // 内部实现（具体实现由基础设施层提供）
    void* internal_runtime;
} AsyncRuntime;

// 异步任务
typedef struct AsyncTask {
    uint64_t id;                    // 任务ID
    AsyncTaskFunc function;         // 任务函数
    void* arg;                      // 任务参数
    Promise* promise;               // 关联的Promise
    time_t submitted_at;            // 提交时间
} AsyncTask;

// AsyncRuntime方法
AsyncRuntime* async_runtime_create(uint32_t max_concurrent);
void async_runtime_destroy(AsyncRuntime* runtime);

// 任务管理
Future* async_runtime_spawn(AsyncRuntime* runtime, AsyncTaskFunc func, void* arg);
bool async_runtime_cancel_task(AsyncRuntime* runtime, uint64_t task_id);

// 运行时控制
void async_runtime_start(AsyncRuntime* runtime);
void async_runtime_stop(AsyncRuntime* runtime);
bool async_runtime_is_running(const AsyncRuntime* runtime);

// 状态查询
uint32_t async_runtime_get_running_count(const AsyncRuntime* runtime);
uint32_t async_runtime_get_pending_count(const AsyncRuntime* runtime);

// 工具函数
Future* async_runtime_all(AsyncRuntime* runtime, Future** futures, size_t count);
Future* async_runtime_race(AsyncRuntime* runtime, Future** futures, size_t count);
Future* async_runtime_any(AsyncRuntime* runtime, Future** futures, size_t count);

// 高级异步操作
Future* async_runtime_timeout(AsyncRuntime* runtime, Future* future, uint32_t timeout_ms);
Future* async_runtime_retry(AsyncRuntime* runtime, AsyncTaskFunc func, void* arg, uint32_t max_retries);

#endif // ASYNC_RUNTIME_H
