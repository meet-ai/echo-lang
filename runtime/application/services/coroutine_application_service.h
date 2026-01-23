#ifndef COROUTINE_APPLICATION_SERVICE_H
#define COROUTINE_APPLICATION_SERVICE_H

#include "../commands/coroutine_commands.h"
#include "../commands/create_coroutine.h"
#include "../dtos/result_dtos.h"
#include "../dtos/coroutine_dtos.h"
#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct CoroutineApplicationService;
typedef struct CoroutineApplicationService CoroutineApplicationService;

// 协程应用服务接口 - 负责协程相关的用例编排
typedef struct {
    // 协程管理用例
    CoroutineCreationResultDTO* (*create_coroutine)(CoroutineApplicationService* service, const CreateCoroutineCommand* cmd);
    OperationResultDTO* (*start_coroutine)(CoroutineApplicationService* service, const StartCoroutineCommand* cmd);
    OperationResultDTO* (*pause_coroutine)(CoroutineApplicationService* service, const PauseCoroutineCommand* cmd);
    OperationResultDTO* (*resume_coroutine)(CoroutineApplicationService* service, const ResumeCoroutineCommand* cmd);
    OperationResultDTO* (*cancel_coroutine)(CoroutineApplicationService* service, const CancelCoroutineCommand* cmd);

    // 协程查询用例
    CoroutineDTO* (*get_coroutine)(CoroutineApplicationService* service, uint64_t coroutine_id);
    CoroutineListDTO* (*get_coroutines)(CoroutineApplicationService* service, const GetCoroutineStatusQuery* query);
    CoroutineStatisticsDTO* (*get_coroutine_statistics)(CoroutineApplicationService* service, const GetCoroutineStatusQuery* query);

    // 协程控制用例
    OperationResultDTO* (*yield_current)(CoroutineApplicationService* service);
    OperationResultDTO* (*set_coroutine_priority)(CoroutineApplicationService* service, const SetCoroutinePriorityCommand* cmd);

    // 协程同步用例
    OperationResultDTO* (*wait_coroutine)(CoroutineApplicationService* service, const WaitCoroutineCommand* cmd);
    OperationResultDTO* (*join_coroutine)(CoroutineApplicationService* service, const JoinCoroutineCommand* cmd);
} CoroutineApplicationServiceInterface;

// 协程应用服务实现
struct CoroutineApplicationService {
    CoroutineApplicationServiceInterface* vtable;  // 虚函数表

    // 依赖的领域服务
    void* coroutine_runtime;       // 协程运行时领域服务
    void* coroutine_scheduler;     // 协程调度器领域服务
    void* event_publisher;         // 事件发布器

    // 应用服务状态
    bool initialized;
    char service_name[256];
    time_t started_at;
};

// 构造函数和析构函数
CoroutineApplicationService* coroutine_application_service_create(void);
void coroutine_application_service_destroy(CoroutineApplicationService* service);

// 初始化和清理
bool coroutine_application_service_initialize(CoroutineApplicationService* service,
                                           void* coroutine_runtime,
                                           void* coroutine_scheduler,
                                           void* event_publisher);
void coroutine_application_service_cleanup(CoroutineApplicationService* service);

// 服务状态查询
bool coroutine_application_service_is_healthy(const CoroutineApplicationService* service);
const char* coroutine_application_service_get_name(const CoroutineApplicationService* service);
time_t coroutine_application_service_get_started_at(const CoroutineApplicationService* service);

// 便捷函数（直接调用虚函数表）
static inline CoroutineCreationResultDTO* coroutine_application_service_create_coroutine(
    CoroutineApplicationService* service, const CreateCoroutineCommand* cmd) {
    return service->vtable->create_coroutine(service, cmd);
}

static inline OperationResultDTO* coroutine_application_service_start_coroutine(
    CoroutineApplicationService* service, const StartCoroutineCommand* cmd) {
    return service->vtable->start_coroutine(service, cmd);
}

static inline CoroutineDTO* coroutine_application_service_get_coroutine(
    CoroutineApplicationService* service, uint64_t coroutine_id) {
    return service->vtable->get_coroutine(service, coroutine_id);
}

#endif // COROUTINE_APPLICATION_SERVICE_H
