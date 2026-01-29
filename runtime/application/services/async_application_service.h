#ifndef ASYNC_APPLICATION_SERVICE_H
#define ASYNC_APPLICATION_SERVICE_H

#include "../commands/future_commands.h"
#include "../dtos/result_dtos.h"
#include "../dtos/async_dtos.h"
#include "../queries/status_queries.h"  // for GetFutureStatusQuery
#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct AsyncApplicationService;
typedef struct AsyncApplicationService AsyncApplicationService;
struct EventBus;  // 事件总线前向声明
typedef struct EventBus EventBus;

// 异步应用服务接口 - 负责异步编程相关的用例编排
typedef struct {
    // Future管理用例
    FutureCreationResultDTO* (*create_future)(AsyncApplicationService* service, const CreateFutureCommand* cmd);
    OperationResultDTO* (*resolve_future)(AsyncApplicationService* service, const ResolveFutureCommand* cmd);
    OperationResultDTO* (*reject_future)(AsyncApplicationService* service, const RejectFutureCommand* cmd);
    OperationResultDTO* (*cancel_future)(AsyncApplicationService* service, const CancelFutureCommand* cmd);

    // 异步操作用例
    AsyncOperationResultDTO* (*await_future)(AsyncApplicationService* service, const AwaitFutureCommand* cmd);
    AsyncOperationResultDTO* (*spawn_async_task)(AsyncApplicationService* service, const SpawnAsyncTaskCommand* cmd);

    // Future查询用例
    FutureDTO* (*get_future)(AsyncApplicationService* service, uint64_t future_id);
    FutureListDTO* (*get_futures)(AsyncApplicationService* service, const GetFutureStatusQuery* query);
    FutureStatisticsDTO* (*get_future_statistics)(AsyncApplicationService* service, const GetFutureStatusQuery* query);

    // 异步控制用例
    OperationResultDTO* (*set_future_timeout)(AsyncApplicationService* service, const SetFutureTimeoutCommand* cmd);
    OperationResultDTO* (*chain_futures)(AsyncApplicationService* service, const ChainFuturesCommand* cmd);
} AsyncApplicationServiceInterface;

// 异步应用服务实现
struct AsyncApplicationService {
    AsyncApplicationServiceInterface* vtable;  // 虚函数表

    // 依赖的领域服务
    void* async_runtime;       // 异步运行时领域服务
    void* future_manager;      // Future管理器领域服务
    EventBus* event_bus;       // 事件总线（用于发布领域事件）

    // 应用服务状态
    bool initialized;
    char service_name[256];
    time_t started_at;
};

// 构造函数和析构函数
AsyncApplicationService* async_application_service_create(void);
void async_application_service_destroy(AsyncApplicationService* service);

// 初始化和清理
bool async_application_service_initialize(AsyncApplicationService* service,
                                        void* async_runtime,
                                        void* future_manager,
                                        EventBus* event_bus);
void async_application_service_cleanup(AsyncApplicationService* service);

// 服务状态查询
bool async_application_service_is_healthy(const AsyncApplicationService* service);
const char* async_application_service_get_name(const AsyncApplicationService* service);
time_t async_application_service_get_started_at(const AsyncApplicationService* service);

// 便捷函数（直接调用虚函数表）
static inline FutureCreationResultDTO* async_application_service_create_future(
    AsyncApplicationService* service, const CreateFutureCommand* cmd) {
    return service->vtable->create_future(service, cmd);
}

static inline OperationResultDTO* async_application_service_resolve_future(
    AsyncApplicationService* service, const ResolveFutureCommand* cmd) {
    return service->vtable->resolve_future(service, cmd);
}

static inline AsyncOperationResultDTO* async_application_service_await_future(
    AsyncApplicationService* service, const AwaitFutureCommand* cmd) {
    return service->vtable->await_future(service, cmd);
}

static inline OperationResultDTO* async_application_service_reject_future(
    AsyncApplicationService* service, const RejectFutureCommand* cmd) {
    return service->vtable->reject_future(service, cmd);
}

static inline FutureDTO* async_application_service_get_future(
    AsyncApplicationService* service, uint64_t future_id) {
    return service->vtable->get_future(service, future_id);
}

#endif // ASYNC_APPLICATION_SERVICE_H
