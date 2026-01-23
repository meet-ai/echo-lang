#ifndef TASK_APPLICATION_SERVICE_H
#define TASK_APPLICATION_SERVICE_H

#include "../commands/task_commands.h"
#include "../queries/status_queries.h"
#include "../dtos/task_dtos.h"
#include "../dtos/result_dtos.h"
#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct TaskApplicationService;
typedef struct TaskApplicationService TaskApplicationService;

// 任务应用服务接口 - 负责任务相关的用例编排
typedef struct {
    // 任务管理用例
    TaskCreationResultDTO* (*create_task)(TaskApplicationService* service, const CreateTaskCommand* cmd);
    TaskCancellationResultDTO* (*cancel_task)(TaskApplicationService* service, const CancelTaskCommand* cmd);
    OperationResultDTO* (*pause_task)(TaskApplicationService* service, const PauseTaskCommand* cmd);
    OperationResultDTO* (*resume_task)(TaskApplicationService* service, const ResumeTaskCommand* cmd);
    OperationResultDTO* (*update_task_priority)(TaskApplicationService* service, const UpdateTaskPriorityCommand* cmd);

    // 任务查询用例
    TaskDTO* (*get_task)(TaskApplicationService* service, uint64_t task_id);
    TaskListDTO* (*get_tasks)(TaskApplicationService* service, const GetTaskStatusQuery* query);
    TaskStatisticsDTO* (*get_task_statistics)(TaskApplicationService* service, const GetTaskStatusQuery* query);

    // 任务控制用例
    OperationResultDTO* (*start_scheduler)(TaskApplicationService* service);
    OperationResultDTO* (*stop_scheduler)(TaskApplicationService* service);
    OperationResultDTO* (*pause_scheduler)(TaskApplicationService* service);
    OperationResultDTO* (*resume_scheduler)(TaskApplicationService* service);

    // 调度器状态查询
    uint32_t (*get_running_task_count)(TaskApplicationService* service);
    uint32_t (*get_queued_task_count)(TaskApplicationService* service);
    bool (*is_scheduler_running)(TaskApplicationService* service);
} TaskApplicationServiceInterface;

// 任务应用服务实现
struct TaskApplicationService {
    TaskApplicationServiceInterface* vtable;  // 虚函数表

    // 依赖的领域服务
    void* task_scheduler;           // 任务调度器领域服务
    void* task_repository;          // 任务仓储接口
    void* event_publisher;          // 事件发布器

    // 应用服务状态
    bool initialized;
    char service_name[256];
    time_t started_at;
};

// 构造函数和析构函数
TaskApplicationService* task_application_service_create(void);
void task_application_service_destroy(TaskApplicationService* service);

// 初始化和清理
bool task_application_service_initialize(TaskApplicationService* service,
                                       void* task_scheduler,
                                       void* task_repository,
                                       void* event_publisher);
void task_application_service_cleanup(TaskApplicationService* service);

// 服务状态查询
bool task_application_service_is_healthy(const TaskApplicationService* service);
const char* task_application_service_get_name(const TaskApplicationService* service);
time_t task_application_service_get_started_at(const TaskApplicationService* service);

// 便捷函数（直接调用虚函数表）
static inline TaskCreationResultDTO* task_application_service_create_task(
    TaskApplicationService* service, const CreateTaskCommand* cmd) {
    return service->vtable->create_task(service, cmd);
}

static inline TaskCancellationResultDTO* task_application_service_cancel_task(
    TaskApplicationService* service, const CancelTaskCommand* cmd) {
    return service->vtable->cancel_task(service, cmd);
}

static inline TaskDTO* task_application_service_get_task(
    TaskApplicationService* service, uint64_t task_id) {
    return service->vtable->get_task(service, task_id);
}

static inline TaskListDTO* task_application_service_get_tasks(
    TaskApplicationService* service, const GetTaskStatusQuery* query) {
    return service->vtable->get_tasks(service, query);
}

#endif // TASK_APPLICATION_SERVICE_H
